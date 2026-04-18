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

func TestMatchedFieldAccuracy(t *testing.T) {
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

	result := score.MatchedFieldAccuracy(pairs)

	// priority: all match (7→mid, 5→mid, 2→low same bands)
	if result["priority"].Passed != 3 || result["priority"].Total != 3 {
		t.Errorf("priority: got passed=%d total=%d, want 3/3", result["priority"].Passed, result["priority"].Total)
	}

	// category: pair0=work/work=pass, pair1=work/personal=fail, pair2=personal/personal=pass → 2/3
	if result["category"].Passed != 2 || result["category"].Total != 3 {
		t.Errorf("category: got passed=%d total=%d, want 2/3", result["category"].Passed, result["category"].Total)
	}

	// duration: pair0=30/30 (bucket1/1)=pass, pair1=45(bucket1)/60(bucket2)=fail, pair2=25(0)/20(0)=pass → 2/3
	if result["duration_minutes"].Passed != 2 || result["duration_minutes"].Total != 3 {
		t.Errorf("duration_minutes: got passed=%d total=%d, want 2/3", result["duration_minutes"].Passed, result["duration_minutes"].Total)
	}

	// deadline: pair0 both set, within tol → TP; pair1,pair2 both nil → no contribution → total=1
	if result["deadline"].TP != 1 || result["deadline"].Total != 1 {
		t.Errorf("deadline: got tp=%d total=%d, want 1/1", result["deadline"].TP, result["deadline"].Total)
	}
}
