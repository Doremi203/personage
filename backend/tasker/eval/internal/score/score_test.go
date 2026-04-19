package score_test

import (
	"testing"
	"time"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/score"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

func TestDurationBucket(t *testing.T) {
	tests := []struct {
		minutes int
		want    int
	}{
		{0, 0}, {29, 0},
		{30, 1}, {59, 1},
		{60, 2}, {119, 2},
		{120, 3}, {999, 3},
	}
	for _, tc := range tests {
		if got := score.DurationBucket(tc.minutes); got != tc.want {
			t.Errorf("DurationBucket(%d) = %d, want %d", tc.minutes, got, tc.want)
		}
	}
}

func TestPriorityMatches(t *testing.T) {
	tests := []struct {
		a, b int
		want bool
	}{
		{1, 3, true},  // both low
		{4, 7, true},  // both mid
		{8, 10, true}, // both high
		{3, 4, false}, // low vs mid
		{7, 8, false}, // mid vs high
		{1, 10, false},
	}
	for _, tc := range tests {
		if got := score.PriorityMatches(tc.a, tc.b); got != tc.want {
			t.Errorf("PriorityMatches(%d,%d) = %v, want %v", tc.a, tc.b, got, tc.want)
		}
	}
}

func TestTimeMatches(t *testing.T) {
	now := time.Now()
	within := now.Add(30 * time.Minute)
	outside := now.Add(2 * time.Hour)

	tests := []struct {
		name   string
		pred   *time.Time
		gold   *time.Time
		wantTP bool
		wantFP bool
		wantFN bool
	}{
		{"both nil", nil, nil, false, false, false},
		{"pred set gold nil", &now, nil, false, true, false},
		{"pred nil gold set", nil, &now, false, false, true},
		{"within tolerance", &within, &now, true, false, false},
		{"outside tolerance", &outside, &now, false, true, true},
	}
	for _, tc := range tests {
		tp, fp, fn := score.TimeMatches(tc.pred, tc.gold, score.DeadlineTolerance)
		if tp != tc.wantTP || fp != tc.wantFP || fn != tc.wantFN {
			t.Errorf("%s: got tp=%v fp=%v fn=%v, want tp=%v fp=%v fn=%v",
				tc.name, tp, fp, fn, tc.wantTP, tc.wantFP, tc.wantFN)
		}
	}
}

func TestFieldQualityFromPairs_empty(t *testing.T) {
	fq := score.FieldQualityFromPairs(nil)
	if fq.Title.TokenF1Mean != 0 || fq.Category.Accuracy != 0 {
		t.Error("empty pairs should return zero FieldQuality")
	}
}

func TestFieldQualityFromPairs(t *testing.T) {
	now := time.Date(2026, 4, 18, 12, 0, 0, 0, time.UTC)

	pairs := []score.MatchedPair{
		{
			Pred: score.Task{
				Title: "review budget", Description: "read spreadsheet and comment",
				DurationMinutes: 30, Priority: 7, Category: domain.TaskCategoryWork,
				Deadline: &now,
			},
			Gold: score.Task{
				Title: "review budget", Description: "read the spreadsheet and comment by friday",
				DurationMinutes: 30, Priority: 7, Category: domain.TaskCategoryWork,
				Deadline: &now,
			},
		},
		{
			Pred: score.Task{
				Title: "send report", Description: "quarterly report",
				DurationMinutes: 45, Priority: 5, Category: domain.TaskCategoryWork,
			},
			Gold: score.Task{
				Title: "send report", Description: "send quarterly report to manager",
				DurationMinutes: 60, Priority: 5, Category: domain.TaskCategoryPersonal,
			},
		},
		{
			Pred: score.Task{
				Title: "buy groceries", Description: "",
				DurationMinutes: 25, Priority: 2, Category: domain.TaskCategoryPersonal,
			},
			Gold: score.Task{
				Title: "buy groceries", Description: "milk bread eggs",
				DurationMinutes: 20, Priority: 2, Category: domain.TaskCategoryPersonal,
			},
		},
	}

	fq := score.FieldQualityFromPairs(pairs)

	// Title: all identical titles → F1 = 1.0 for each pair → mean = 1.0
	if fq.Title.TokenF1Mean < 0.99 {
		t.Errorf("title token_f1_mean = %.3f, want ~1.0", fq.Title.TokenF1Mean)
	}

	// Priority: all pairs have matching bands → band_acc = 1.0, exact_acc = 1.0
	if fq.Priority.BandAcc != 1.0 {
		t.Errorf("priority band_acc = %.3f, want 1.0", fq.Priority.BandAcc)
	}
	if fq.Priority.ExactAcc != 1.0 {
		t.Errorf("priority exact_acc = %.3f, want 1.0", fq.Priority.ExactAcc)
	}

	// Category: pair0=work/work=pass, pair1=work/personal=fail, pair2=personal/personal=pass → 2/3
	wantCatAcc := 2.0 / 3.0
	if fq.Category.Accuracy < wantCatAcc-0.001 || fq.Category.Accuracy > wantCatAcc+0.001 {
		t.Errorf("category accuracy = %.3f, want %.3f", fq.Category.Accuracy, wantCatAcc)
	}

	// Duration MAE: |30-30|=0, |45-60|=15, |25-20|=5 → mean = 20/3 ≈ 6.67
	wantMAE := 20.0 / 3.0
	if fq.DurationMinutes.MAE < wantMAE-0.1 || fq.DurationMinutes.MAE > wantMAE+0.1 {
		t.Errorf("duration MAE = %.3f, want %.3f", fq.DurationMinutes.MAE, wantMAE)
	}

	// Duration tol_15: all diffs ≤ 15 → acc = 1.0
	if fq.DurationMinutes.Tol15Acc != 1.0 {
		t.Errorf("duration tol_15_acc = %.3f, want 1.0", fq.DurationMinutes.Tol15Acc)
	}

	// Deadline: pair0 both set (same time → within tol), pair1,pair2 both nil → presence_acc = 1.0
	if fq.Deadline.PresenceAcc != 1.0 {
		t.Errorf("deadline presence_acc = %.3f, want 1.0", fq.Deadline.PresenceAcc)
	}
	// value MAE: only pair0 contributes, diff = 0
	if fq.Deadline.ValueMAEMinutes != 0 {
		t.Errorf("deadline value_mae = %.3f, want 0", fq.Deadline.ValueMAEMinutes)
	}
}

func TestFieldQualityFromPairs_embeddingSim(t *testing.T) {
	pairs := []score.MatchedPair{
		{
			Pred:              score.Task{Title: "foo"},
			Gold:              score.Task{Title: "foo"},
			TitleEmbeddingSim: 0.9,
		},
		{
			Pred:              score.Task{Title: "bar"},
			Gold:              score.Task{Title: "bar"},
			TitleEmbeddingSim: 0.8,
		},
	}
	fq := score.FieldQualityFromPairs(pairs)
	want := 0.85
	if fq.Title.EmbeddingSimMean < want-0.001 || fq.Title.EmbeddingSimMean > want+0.001 {
		t.Errorf("embedding_sim_mean = %.3f, want %.3f", fq.Title.EmbeddingSimMean, want)
	}
}

func TestFieldQualityFromPairs_toleranceAccuracies(t *testing.T) {
	t1 := time.Date(2026, 1, 1, 10, 0, 0, 0, time.UTC)
	t2 := t1.Add(20 * time.Minute) // within 1h
	t3 := t1.Add(90 * time.Minute) // outside 1h, within 24h
	t4 := t1.Add(25 * time.Hour)   // outside 24h

	pairs := []score.MatchedPair{
		{Pred: score.Task{StartTime: &t2}, Gold: score.Task{StartTime: &t1}}, // diff=20m
		{Pred: score.Task{StartTime: &t3}, Gold: score.Task{StartTime: &t1}}, // diff=90m
		{Pred: score.Task{StartTime: &t4}, Gold: score.Task{StartTime: &t1}}, // diff=25h
	}
	fq := score.FieldQualityFromPairs(pairs)

	// tol_1h: only pair0 (20m ≤ 1h) → 1/3
	want1h := 1.0 / 3.0
	if fq.StartTime.Tol1hAcc < want1h-0.001 || fq.StartTime.Tol1hAcc > want1h+0.001 {
		t.Errorf("start_time tol_1h_acc = %.3f, want %.3f", fq.StartTime.Tol1hAcc, want1h)
	}

	// tol_24h: pair0 and pair1 (both ≤ 24h) → 2/3
	want24h := 2.0 / 3.0
	if fq.StartTime.Tol24hAcc < want24h-0.001 || fq.StartTime.Tol24hAcc > want24h+0.001 {
		t.Errorf("start_time tol_24h_acc = %.3f, want %.3f", fq.StartTime.Tol24hAcc, want24h)
	}
}
