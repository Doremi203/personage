package score

import (
	"time"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/tokenf1"
	"github.com/Doremi203/personage/backend/tasker/internal/domain"
)

const (
	DescriptionTokenF1Threshold = 0.4
	DeadlineTolerance           = time.Hour
)

// Task is the uniform task shape used for scoring. Both generated and expected
// tasks are converted to this type before comparison.
type Task struct {
	ID              string
	Title           string
	Description     string
	DurationMinutes int
	Priority        int
	Deadline        *time.Time
	StartTime       *time.Time
	Category        domain.TaskCategory
}

func DurationBucket(minutes int) int {
	switch {
	case minutes < 30:
		return 0
	case minutes < 60:
		return 1
	case minutes < 120:
		return 2
	default:
		return 3
	}
}

func PriorityMatches(a, b int) bool {
	return domain.PriorityFromInt(a) == domain.PriorityFromInt(b)
}

// TimeMatches reports TP/FP/FN semantics for an optional time field.
// nil+nil → no contribution; one-sided → FP or FN; both present and within
// tolerance → TP; both present but outside tolerance → FP+FN.
func TimeMatches(pred, gold *time.Time, tol time.Duration) (tp, fp, fn bool) {
	switch {
	case pred == nil && gold == nil:
		return false, false, false
	case pred != nil && gold == nil:
		return false, true, false
	case pred == nil && gold != nil:
		return false, false, true
	default:
		d := pred.Sub(*gold)
		if d < 0 {
			d = -d
		}
		if d <= tol {
			return true, false, false
		}
		return false, true, true
	}
}

// MatchedPair holds one matched generated↔expected pair for secondary scoring.
type MatchedPair struct {
	Pred Task
	Gold Task
}

// FieldScore aggregates pass/fail or TP/FP/FN for a single field.
type FieldScore struct {
	Passed int `json:"passed"`
	Total  int `json:"total"`
	TP     int `json:"tp,omitempty"`
	FP     int `json:"fp,omitempty"`
	FN     int `json:"fn,omitempty"`
}

func (fs FieldScore) Accuracy() float64 {
	if fs.Total == 0 {
		return 0
	}
	return float64(fs.Passed) / float64(fs.Total)
}

func (fs FieldScore) F1() float64 {
	if fs.TP == 0 {
		return 0
	}
	p := float64(fs.TP) / float64(fs.TP+fs.FP)
	r := float64(fs.TP) / float64(fs.TP+fs.FN)
	if p+r == 0 {
		return 0
	}
	return 2 * p * r / (p + r)
}

// MatchedFieldAccuracy computes per-field scores across matched pairs.
func MatchedFieldAccuracy(pairs []MatchedPair) map[string]FieldScore {
	fields := map[string]*FieldScore{
		"description":      {},
		"duration_minutes": {},
		"priority":         {},
		"category":         {},
		"deadline":         {},
		"start_time":       {},
	}

	for _, mp := range pairs {
		// description: token-F1 >= threshold
		descF1 := tokenf1.Score(mp.Pred.Description, mp.Gold.Description)
		fields["description"].Total++
		if descF1 >= DescriptionTokenF1Threshold {
			fields["description"].Passed++
		}

		// duration_minutes: same bucket
		fields["duration_minutes"].Total++
		if DurationBucket(mp.Pred.DurationMinutes) == DurationBucket(mp.Gold.DurationMinutes) {
			fields["duration_minutes"].Passed++
		}

		// priority: same band
		fields["priority"].Total++
		if PriorityMatches(mp.Pred.Priority, mp.Gold.Priority) {
			fields["priority"].Passed++
		}

		// category: exact match
		fields["category"].Total++
		if mp.Pred.Category == mp.Gold.Category {
			fields["category"].Passed++
		}

		// deadline: TP/FP/FN
		tp, fp, fn := TimeMatches(mp.Pred.Deadline, mp.Gold.Deadline, DeadlineTolerance)
		if tp {
			fields["deadline"].TP++
			fields["deadline"].Passed++
		}
		if fp {
			fields["deadline"].FP++
		}
		if fn {
			fields["deadline"].FN++
		}
		if tp || fp || fn {
			fields["deadline"].Total++
		}

		// start_time: TP/FP/FN
		tp, fp, fn = TimeMatches(mp.Pred.StartTime, mp.Gold.StartTime, DeadlineTolerance)
		if tp {
			fields["start_time"].TP++
			fields["start_time"].Passed++
		}
		if fp {
			fields["start_time"].FP++
		}
		if fn {
			fields["start_time"].FN++
		}
		if tp || fp || fn {
			fields["start_time"].Total++
		}
	}

	result := make(map[string]FieldScore, len(fields))
	for k, v := range fields {
		result[k] = *v
	}
	return result
}
