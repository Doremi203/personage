package runner

import (
	"context"
	"fmt"
	"os"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/eval/internal/embscore"
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
	ListClusterDiagnostics(ctx context.Context, userID string) ([]domain.ClusterGenerationDiagnostic, error)
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

	// ReportOnly skips DB reset and snapshot replay; does a single ListTasks poll.
	ReportOnly bool

	// EmbeddingAPIKey enables embedding-based title matching when non-empty.
	// Falls back to token F1 when empty.
	EmbeddingAPIKey  string
	EmbeddingModel   string // default: embscore.DefaultModel
	EmbeddingBaseURL string // default: embscore.DefaultBaseURL

	// MatchTokenThreshold is the maximum cost (1 − tokenF1) to accept a match.
	// Defaults to matchCostThreshold when 0.
	MatchTokenThreshold float64

	// MatchEmbedThreshold is the maximum cost (1 − cosine) to accept a match
	// when embedding-based matching is used. Defaults to matchEmbCostThreshold when 0.
	MatchEmbedThreshold float64
}

var minTaskWaitTimeout = 10 * time.Minute

const (
	matchCostThreshold    = 0.7  // accept tokenF1 ≥ 0.3
	matchEmbCostThreshold = 0.45 // accept cosine ≥ 0.55
)

// Runner orchestrates one full eval run: reset → replay → poll → match → score.
type Runner struct {
	Traitex TraitexClient
	Tasker  TaskerClient
	DB      DBResetter
	Cfg     Config
}

// Run executes one full eval run against fix and returns a populated report.
func (r *Runner) Run(ctx context.Context, fix fixture.Fixture, fixtureName string) (report.Report, error) {
	if r.Cfg.ReportOnly {
		fmt.Fprintf(os.Stderr, "report-only mode: skipping DB reset and snapshot replay\n")
	} else {
		if err := r.DB.Reset(ctx); err != nil {
			return report.Report{}, errors.WrapFail(err, "reset eval DB")
		}

		sentCount, err := r.Traitex.SendProcessingSnapshot(ctx, fix.SnapshotID, r.Cfg.EvalQueueURL)
		if err != nil {
			return report.Report{}, errors.WrapFail(err, "send processing snapshot")
		}

		fmt.Fprintf(os.Stderr, "replayed snapshot %s: sent %d events to eval queue\n", fix.SnapshotID, sentCount)
	}

	generated, err := r.poll(ctx, fix.UserID)
	if err != nil {
		return report.Report{}, err
	}

	clusterDiagnostics, err := r.Tasker.ListClusterDiagnostics(ctx, fix.UserID)
	if err != nil {
		return report.Report{}, errors.WrapFail(err, "list cluster diagnostics")
	}

	return r.buildReport(ctx, fix, fixtureName, generated, clusterDiagnostics), nil
}

func (r *Runner) poll(ctx context.Context, userID string) ([]domain.Task, error) {
	if r.Cfg.ReportOnly {
		tasks, err := r.Tasker.ListTasks(ctx, userID)
		if err != nil {
			return nil, errors.WrapFail(err, "list tasks")
		}
		fmt.Fprintf(os.Stderr, "report-only poll: generated=%d\n", len(tasks))
		return tasks, nil
	}

	waitTimeout := max(r.Cfg.OverallTimeout, minTaskWaitTimeout)

	ticker := time.NewTicker(r.Cfg.PollInterval)
	defer ticker.Stop()

	timer := time.NewTimer(waitTimeout)
	defer timer.Stop()

	start := time.Now()
	deadline := start.Add(waitTimeout)
	var lastTasks []domain.Task

	fmt.Fprintf(os.Stderr, "waiting for tasks: user_id=%s timeout=%s poll_interval=%s\n", userID, waitTimeout, r.Cfg.PollInterval)

	pollOnce := func() error {
		tasks, err := r.Tasker.ListTasks(ctx, userID)
		if err != nil {
			return errors.WrapFail(err, "list tasks")
		}

		lastTasks = tasks
		remaining := max(time.Until(deadline), 0)

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

loop:
	for {
		select {
		case <-ctx.Done():
			return nil, errors.WrapFail(ctx.Err(), "wait for tasks")
		case <-timer.C:
			break loop
		case <-ticker.C:
			if err := pollOnce(); err != nil {
				return nil, err
			}
		}
	}

	fmt.Fprintf(os.Stderr, "task wait complete: waited=%s generated=%d\n", waitTimeout, len(lastTasks))
	return lastTasks, nil
}

func taskSnapshot(t domain.Task) report.TaskSnapshot {
	snap := report.TaskSnapshot{
		ID:               t.ID.String(),
		UserID:           t.UserID.String(),
		Title:            t.Title,
		Description:      t.Description,
		DurationMinutes:  int(t.Duration.Minutes()),
		Priority:         t.Priority,
		Deadline:         t.Deadline,
		StartTime:        t.StartTime,
		EndTime:          t.EndTime,
		Category:         string(t.Category),
		EvidenceEventIDs: eventIDsToStrings(t.EvidenceEventIDs),
	}
	if t.ClusterID != nil {
		snap.ClusterID = new(t.ClusterID.String())
	}
	return snap
}

func expectedSnapshot(et fixture.ExpectedTask) report.TaskSnapshot {
	return report.TaskSnapshot{
		Title:           et.Title,
		Description:     et.Description,
		DurationMinutes: et.DurationMinutes,
		Priority:        et.Priority,
		Deadline:        et.Deadline,
		StartTime:       et.StartTime,
		Category:        et.Category,
	}
}

func (r *Runner) buildReport(
	ctx context.Context,
	fix fixture.Fixture,
	fixtureName string,
	generated []domain.Task,
	clusterDiagnostics []domain.ClusterGenerationDiagnostic,
) report.Report {
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

	// Optionally pre-embed all titles for semantic matching.
	var (
		genEmbs [][]float64
		expEmbs [][]float64
		useEmb  bool
	)
	if r.Cfg.EmbeddingAPIKey != "" {
		scorer := embscore.New(r.Cfg.EmbeddingAPIKey, r.Cfg.EmbeddingModel, r.Cfg.EmbeddingBaseURL)
		allTitles := make([]string, 0, len(gen)+len(expected))
		for _, g := range gen {
			allTitles = append(allTitles, g.Title)
		}
		for _, e := range expected {
			allTitles = append(allTitles, e.Title)
		}
		allEmbs, err := scorer.EmbedBatch(ctx, allTitles)
		if err != nil {
			fmt.Fprintf(os.Stderr, "warn: embedding failed, falling back to tokenF1: %v\n", err)
		} else {
			genEmbs = allEmbs[:len(gen)]
			expEmbs = allEmbs[len(gen):]
			useEmb = true
		}
	}

	// Determine cost thresholds, honouring per-run config overrides.
	tokenThreshold := r.Cfg.MatchTokenThreshold
	if tokenThreshold == 0 {
		tokenThreshold = matchCostThreshold
	}
	embedThreshold := r.Cfg.MatchEmbedThreshold
	if embedThreshold == 0 {
		embedThreshold = matchEmbCostThreshold
	}
	matcherType := "tokenf1"
	costThreshold := tokenThreshold
	if useEmb {
		costThreshold = embedThreshold
		matcherType = "embedding"
	}

	cost := make([][]float64, len(gen))
	for i := range cost {
		cost[i] = make([]float64, len(expected))
		for j := range cost[i] {
			if useEmb {
				cost[i][j] = 1 - embscore.CosineSim(genEmbs[i], expEmbs[j])
			} else {
				cost[i][j] = 1 - tokenf1.Score(gen[i].Title, expected[j].Title)
			}
		}
	}

	matchResult := match.Greedy(cost, costThreshold)

	// Build matched pairs for field quality scoring.
	pairs := make([]score.MatchedPair, len(matchResult.Pairs))
	for k, p := range matchResult.Pairs {
		pair := score.MatchedPair{Pred: gen[p.GeneratedIdx], Gold: expected[p.ExpectedIdx]}
		if useEmb {
			pair.TitleEmbeddingSim = embscore.CosineSim(genEmbs[p.GeneratedIdx], expEmbs[p.ExpectedIdx])
		}
		pairs[k] = pair
	}

	fieldQuality := score.FieldQualityFromPairs(pairs)

	// Build match details for report.
	matchDetails := make([]report.MatchDetail, len(matchResult.Pairs))
	for k, p := range matchResult.Pairs {
		detail := report.MatchDetail{
			ExpectedID:        expected[p.ExpectedIdx].ID,
			GeneratedID:       gen[p.GeneratedIdx].ID,
			TitleF1:           tokenf1.Score(gen[p.GeneratedIdx].Title, expected[p.ExpectedIdx].Title),
			TitleEmbeddingSim: pairs[k].TitleEmbeddingSim,
			Generated:         taskSnapshot(generated[p.GeneratedIdx]),
			Expected:          expectedSnapshot(fix.ExpectedTasks[p.ExpectedIdx]),
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

	// Unmatched generated with closest expected (always tokenF1 for diagnostics).
	unmatchedGen := make([]report.UnmatchedGenerated, len(matchResult.UnmatchedGenerated))
	for k, gi := range matchResult.UnmatchedGenerated {
		ug := report.UnmatchedGenerated{
			ID:    gen[gi].ID,
			Title: gen[gi].Title,
			Task:  taskSnapshot(generated[gi]),
		}
		bestJ := -1
		for j := range expected {
			f1 := tokenf1.Score(gen[gi].Title, expected[j].Title)
			if f1 > ug.ClosestTitleF1 {
				ug.ClosestTitleF1 = f1
				ug.ClosestExpected = expected[j].ID
				bestJ = j
			}
		}
		if useEmb && bestJ >= 0 {
			ug.ClosestTitleEmbeddingSim = embscore.CosineSim(genEmbs[gi], expEmbs[bestJ])
		}
		unmatchedGen[k] = ug
	}

	unmatchedExp := make([]report.UnmatchedExpected, len(matchResult.UnmatchedExpected))
	for k, ei := range matchResult.UnmatchedExpected {
		unmatchedExp[k] = report.UnmatchedExpected{
			ID:    expected[ei].ID,
			Title: expected[ei].Title,
			Task:  expectedSnapshot(fix.ExpectedTasks[ei]),
		}
	}

	rep := report.Report{
		Fixture:     fixtureName,
		SnapshotID:  fix.SnapshotID,
		Timestamp:   time.Now().UTC(),
		MatcherType: matcherType,
		Counts: report.Counts{
			Expected:  len(expected),
			Generated: len(gen),
			TP:        len(matchResult.Pairs),
			FP:        len(matchResult.UnmatchedGenerated),
			FN:        len(matchResult.UnmatchedExpected),
		},
		FieldQuality:       fieldQuality,
		ClusterDiagnostics: buildClusterDiagnostics(clusterDiagnostics),
		Matches:            matchDetails,
		UnmatchedGenerated: unmatchedGen,
		UnmatchedExpected:  unmatchedExp,
	}
	rep.Compute()
	return rep
}

func buildClusterDiagnostics(diagnostics []domain.ClusterGenerationDiagnostic) report.ClusterDiagnostics {
	clusterSnapshots := make([]report.ClusterSnapshot, len(diagnostics))
	closedClusters := 0
	tasklessClosedClusters := 0
	skippedNonActionable := 0

	for i, diagnostic := range diagnostics {
		clusterSnapshots[i] = report.ClusterSnapshot{
			ID:                 diagnostic.ClusterID.String(),
			UserID:             diagnostic.UserID.String(),
			Status:             string(diagnostic.Status),
			EventCount:         diagnostic.EventCount,
			GenerationOutcome:  clusterOutcomeToString(diagnostic.GenerationOutcome),
			GenerationReason:   diagnostic.GenerationReason,
			GeneratedTaskCount: diagnostic.GeneratedTaskCount,
			CreatedAt:          diagnostic.CreatedAt,
			UpdatedAt:          diagnostic.UpdatedAt,
		}

		if diagnostic.GenerationOutcome != nil && *diagnostic.GenerationOutcome == domain.ClusterGenerationOutcomeNonActionable {
			skippedNonActionable++
		}

		if diagnostic.Status != domain.ClusterStatusClosed {
			continue
		}

		closedClusters++
		if diagnostic.GeneratedTaskCount == 0 {
			tasklessClosedClusters++
		}
	}

	tasklessClusterRate := 0.0
	if closedClusters > 0 {
		tasklessClusterRate = float64(tasklessClosedClusters) / float64(closedClusters)
	}

	return report.ClusterDiagnostics{
		Total:                len(diagnostics),
		Closed:               closedClusters,
		SkippedNonActionable: skippedNonActionable,
		TasklessClusterRate:  tasklessClusterRate,
		Clusters:             clusterSnapshots,
	}
}

func clusterOutcomeToString(outcome *domain.ClusterGenerationOutcome) *string {
	if outcome == nil {
		return nil
	}

	value := string(*outcome)
	return &value
}

func eventIDsToStrings(ids []domain.EventID) []string {
	values := make([]string, len(ids))
	for i, id := range ids {
		values[i] = id.String()
	}

	return values
}
