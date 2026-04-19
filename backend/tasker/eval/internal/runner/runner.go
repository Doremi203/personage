package runner

import (
	"context"
	"errors"
	"fmt"
	"os"
	"time"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/fixture"
	"github.com/Doremi203/personage/backend/tasker/eval/internal/match"
	"github.com/Doremi203/personage/backend/tasker/eval/internal/report"
	"github.com/Doremi203/personage/backend/tasker/eval/internal/score"
	"github.com/Doremi203/personage/backend/tasker/eval/internal/tokenf1"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

// TraitexClient sends a processing snapshot to a specific SQS queue.
type TraitexClient interface {
	SendProcessingSnapshot(ctx context.Context, snapshotID, targetQueueURL string) (sentCount int, err error)
}

// TaskerClient lists generated tasks for a user.
type TaskerClient interface {
	ListTasks(ctx context.Context, userID string) ([]domain.Task, error)
}

// DBResetter wipes and re-applies migrations on the eval database.
type DBResetter interface {
	Reset(ctx context.Context) error
}

// Config controls the runner's timing behaviour.
type Config struct {
	EvalQueueURL string

	// PollInterval is how often to call ListTasks while waiting for generated tasks.
	PollInterval time.Duration // default 10s

	// OverallTimeout is how long to wait before collecting the latest task list.
	OverallTimeout time.Duration // default 15m
}

var minTaskWaitTimeout = 15 * time.Minute

const matchCostThreshold = 0.5

// Runner orchestrates one full eval run: reset → replay → poll → match → score.
type Runner struct {
	Traitex TraitexClient
	Tasker  TaskerClient
	DB      DBResetter
	Cfg     Config
}

// Run executes one full eval run against fix and returns a populated report.
func (r *Runner) Run(ctx context.Context, fix fixture.Fixture, fixtureName string) (report.Report, error) {
	if err := r.DB.Reset(ctx); err != nil {
		return report.Report{}, fmt.Errorf("reset eval DB: %w", err)
	}

	sentCount, err := r.Traitex.SendProcessingSnapshot(ctx, fix.SnapshotID, r.Cfg.EvalQueueURL)
	if err != nil {
		return report.Report{}, fmt.Errorf("send processing snapshot: %w", err)
	}

	fmt.Fprintf(os.Stderr, "replayed snapshot %s: sent %d events to eval queue\n", fix.SnapshotID, sentCount)

	generated, err := r.poll(ctx, fix.UserID)
	if err != nil {
		return report.Report{}, err
	}

	return r.buildReport(fix, fixtureName, generated), nil
}

func (r *Runner) poll(ctx context.Context, userID string) ([]domain.Task, error) {
	waitTimeout := r.Cfg.OverallTimeout
	if waitTimeout < minTaskWaitTimeout {
		waitTimeout = minTaskWaitTimeout
	}

	overallCtx, cancel := context.WithTimeout(ctx, waitTimeout)
	defer cancel()

	ticker := time.NewTicker(r.Cfg.PollInterval)
	defer ticker.Stop()

	start := time.Now()
	deadline := start.Add(waitTimeout)
	var lastTasks []domain.Task

	fmt.Fprintf(os.Stderr, "waiting for tasks: user_id=%s timeout=%s poll_interval=%s\n", userID, waitTimeout, r.Cfg.PollInterval)

	pollOnce := func() error {
		tasks, err := r.Tasker.ListTasks(overallCtx, userID)
		if err != nil {
			return fmt.Errorf("list tasks: %w", err)
		}

		lastTasks = tasks
		remaining := time.Until(deadline)
		if remaining < 0 {
			remaining = 0
		}

		fmt.Fprintf(
			os.Stderr,
			"task wait progress: elapsed=%s remaining=%s generated=%d\n",
			time.Since(start).Round(time.Second),
			remaining.Round(time.Second),
			len(tasks),
		)

		return nil
	}

	if err := pollOnce(); err != nil {
		return nil, err
	}

	for {
		select {
		case <-overallCtx.Done():
			if err := overallCtx.Err(); err != nil && !errors.Is(err, context.DeadlineExceeded) {
				return nil, fmt.Errorf("wait for tasks: %w", err)
			}

			fmt.Fprintf(os.Stderr, "task wait complete: waited=%s generated=%d\n", waitTimeout, len(lastTasks))
			return lastTasks, nil
		case <-ticker.C:
			if err := pollOnce(); err != nil {
				return nil, err
			}
		}
	}
}

func (r *Runner) buildReport(fix fixture.Fixture, fixtureName string, generated []domain.Task) report.Report {
	expected := make([]score.Task, len(fix.ExpectedTasks))
	for i, et := range fix.ExpectedTasks {
		expected[i] = et.ToScoreTask()
	}

	gen := make([]score.Task, len(generated))
	for i, t := range generated {
		gen[i] = score.Task{
			ID:              t.ID.String(),
			Title:           t.Title,
			Description:     t.Description,
			DurationMinutes: int(t.Duration.Minutes()),
			Priority:        t.Priority,
			Deadline:        t.Deadline,
			StartTime:       t.StartTime,
			Category:        t.Category,
		}
	}

	// Build cost matrix.
	cost := make([][]float64, len(gen))
	for i := range cost {
		cost[i] = make([]float64, len(expected))
		for j := range cost[i] {
			cost[i][j] = 1 - tokenf1.Score(gen[i].Title, expected[j].Title)
		}
	}

	matchResult := match.Greedy(cost, matchCostThreshold)

	// Build matched pairs for secondary scoring.
	pairs := make([]score.MatchedPair, len(matchResult.Pairs))
	for k, p := range matchResult.Pairs {
		pairs[k] = score.MatchedPair{Pred: gen[p.GeneratedIdx], Gold: expected[p.ExpectedIdx]}
	}

	fieldAccuracy := score.MatchedFieldAccuracy(pairs)

	// Build match details for report.
	matchDetails := make([]report.MatchDetail, len(matchResult.Pairs))
	for k, p := range matchResult.Pairs {
		detail := report.MatchDetail{
			ExpectedIdx: p.ExpectedIdx,
			GeneratedID: gen[p.GeneratedIdx].ID,
			TitleF1:     p.TitleF1,
		}
		// Classify per-field pass/fail.
		pred := gen[p.GeneratedIdx]
		gold := expected[p.ExpectedIdx]
		if score.DurationBucket(pred.DurationMinutes) == score.DurationBucket(gold.DurationMinutes) {
			detail.FieldsPassed = append(detail.FieldsPassed, "duration_minutes")
		} else {
			detail.FieldsFailed = append(detail.FieldsFailed, "duration_minutes")
		}
		if score.PriorityMatches(pred.Priority, gold.Priority) {
			detail.FieldsPassed = append(detail.FieldsPassed, "priority")
		} else {
			detail.FieldsFailed = append(detail.FieldsFailed, "priority")
		}
		if pred.Category == gold.Category {
			detail.FieldsPassed = append(detail.FieldsPassed, "category")
		} else {
			detail.FieldsFailed = append(detail.FieldsFailed, "category")
		}
		matchDetails[k] = detail
	}

	// Unmatched generated with closest expected.
	unmatchedGen := make([]report.UnmatchedGenerated, len(matchResult.UnmatchedGenerated))
	for k, gi := range matchResult.UnmatchedGenerated {
		ug := report.UnmatchedGenerated{ID: gen[gi].ID, Title: gen[gi].Title, ClosestExpected: -1}
		for j := range expected {
			f1 := tokenf1.Score(gen[gi].Title, expected[j].Title)
			if f1 > ug.ClosestTitleF1 {
				ug.ClosestTitleF1 = f1
				ug.ClosestExpected = j
			}
		}
		unmatchedGen[k] = ug
	}

	unmatchedExp := make([]report.UnmatchedExpected, len(matchResult.UnmatchedExpected))
	for k, ei := range matchResult.UnmatchedExpected {
		unmatchedExp[k] = report.UnmatchedExpected{Idx: ei, Title: expected[ei].Title}
	}

	rep := report.Report{
		Fixture:    fixtureName,
		SnapshotID: fix.SnapshotID,
		Timestamp:  time.Now().UTC(),
		Counts: report.Counts{
			Expected:  len(expected),
			Generated: len(gen),
			TP:        len(matchResult.Pairs),
			FP:        len(matchResult.UnmatchedGenerated),
			FN:        len(matchResult.UnmatchedExpected),
		},
		MatchedFieldAccuracy: fieldAccuracy,
		Matches:              matchDetails,
		UnmatchedGenerated:   unmatchedGen,
		UnmatchedExpected:    unmatchedExp,
	}
	rep.Compute()
	return rep
}
