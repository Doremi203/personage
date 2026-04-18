package runner

import (
	"context"
	"fmt"
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

	// PollInterval is how often to call ListTasks while waiting for tasks to stabilise.
	PollInterval time.Duration // default 10s

	// MinStableInterval is how long the task count must be stable before stopping.
	MinStableInterval time.Duration // default 30s

	// OverallTimeout is the maximum wall-clock time allowed for one run.
	OverallTimeout time.Duration // default 15m
}

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

	_, err := r.Traitex.SendProcessingSnapshot(ctx, fix.SnapshotID, r.Cfg.EvalQueueURL)
	if err != nil {
		return report.Report{}, fmt.Errorf("send processing snapshot: %w", err)
	}

	generated, err := r.poll(ctx, fix.UserID)
	if err != nil {
		return report.Report{}, err
	}

	return r.buildReport(fix, fixtureName, generated), nil
}

func (r *Runner) poll(ctx context.Context, userID string) ([]domain.Task, error) {
	overallCtx, cancel := context.WithTimeout(ctx, r.Cfg.OverallTimeout)
	defer cancel()

	ticker := time.NewTicker(r.Cfg.PollInterval)
	defer ticker.Stop()

	lastCount := -1
	stableSince := time.Time{}

	for {
		select {
		case <-overallCtx.Done():
			return nil, fmt.Errorf("eval timed out after %s waiting for tasks to stabilise", r.Cfg.OverallTimeout)
		case <-ticker.C:
			tasks, err := r.Tasker.ListTasks(overallCtx, userID)
			if err != nil {
				return nil, fmt.Errorf("list tasks: %w", err)
			}

			n := len(tasks)
			if n != lastCount {
				lastCount = n
				stableSince = time.Now()
				continue
			}

			if !stableSince.IsZero() && time.Since(stableSince) >= r.Cfg.MinStableInterval {
				return tasks, nil
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
