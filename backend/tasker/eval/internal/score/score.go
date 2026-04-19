package score

import (
	"math"
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
	Pred              Task
	Gold              Task
	TitleEmbeddingSim float64 // 0 if embeddings not used
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

// TitleQuality holds title similarity metrics.
type TitleQuality struct {
	TokenF1Mean      float64 `json:"token_f1_mean"`
	EmbeddingSimMean float64 `json:"embedding_sim_mean,omitempty"`
}

// TextQuality holds text field similarity metrics.
type TextQuality struct {
	TokenF1Mean float64 `json:"token_f1_mean"`
}

// DurationQuality holds duration field metrics.
type DurationQuality struct {
	MAE          float64 `json:"mae"`
	RelErrorMean float64 `json:"rel_error_mean"`
	Tol15Acc     float64 `json:"tol_15_acc"`
	Tol30Acc     float64 `json:"tol_30_acc"`
	Tol60Acc     float64 `json:"tol_60_acc"`
}

// PriorityQuality holds priority field metrics.
type PriorityQuality struct {
	ExactAcc float64 `json:"exact_acc"`
	BandAcc  float64 `json:"band_acc"`
	MAE      float64 `json:"mae"`
}

// ClassScore holds per-class precision/recall/F1.
type ClassScore struct {
	P  float64 `json:"p"`
	R  float64 `json:"r"`
	F1 float64 `json:"f1"`
}

// CategoryQuality holds category field metrics.
type CategoryQuality struct {
	Accuracy float64               `json:"accuracy"`
	PerClass map[string]ClassScore `json:"per_class"`
}

// TimeFieldQuality holds optional-time-field metrics.
type TimeFieldQuality struct {
	PresenceAcc     float64 `json:"presence_acc"`
	ValueMAEMinutes float64 `json:"value_mae_minutes,omitempty"`
	Tol1hAcc        float64 `json:"tol_1h_acc,omitempty"`
	Tol24hAcc       float64 `json:"tol_24h_acc,omitempty"`
}

// FieldQuality is the full per-field quality report over matched pairs.
type FieldQuality struct {
	Title           TitleQuality     `json:"title"`
	Description     TextQuality      `json:"description"`
	DurationMinutes DurationQuality  `json:"duration_minutes"`
	Priority        PriorityQuality  `json:"priority"`
	Category        CategoryQuality  `json:"category"`
	Deadline        TimeFieldQuality `json:"deadline"`
	StartTime       TimeFieldQuality `json:"start_time"`
}

// FieldQualityFromPairs computes rich per-field quality metrics over matched pairs.
func FieldQualityFromPairs(pairs []MatchedPair) FieldQuality {
	if len(pairs) == 0 {
		return FieldQuality{}
	}
	n := float64(len(pairs))

	// Title
	var titleF1Sum, titleEmbSum float64
	embCount := 0
	for _, p := range pairs {
		titleF1Sum += tokenf1.Score(p.Pred.Title, p.Gold.Title)
		if p.TitleEmbeddingSim > 0 {
			titleEmbSum += p.TitleEmbeddingSim
			embCount++
		}
	}
	tq := TitleQuality{TokenF1Mean: titleF1Sum / n}
	if embCount > 0 {
		tq.EmbeddingSimMean = titleEmbSum / float64(embCount)
	}

	// Description
	var descF1Sum float64
	for _, p := range pairs {
		descF1Sum += tokenf1.Score(p.Pred.Description, p.Gold.Description)
	}
	dq := TextQuality{TokenF1Mean: descF1Sum / n}

	// Duration
	var durationMAE, durationRelErr float64
	var dur15, dur30, dur60 int
	for _, p := range pairs {
		diff := math.Abs(float64(p.Pred.DurationMinutes - p.Gold.DurationMinutes))
		durationMAE += diff
		durationRelErr += diff / math.Max(float64(p.Gold.DurationMinutes), 1)
		if diff <= 15 {
			dur15++
		}
		if diff <= 30 {
			dur30++
		}
		if diff <= 60 {
			dur60++
		}
	}
	durQ := DurationQuality{
		MAE:          durationMAE / n,
		RelErrorMean: durationRelErr / n,
		Tol15Acc:     float64(dur15) / n,
		Tol30Acc:     float64(dur30) / n,
		Tol60Acc:     float64(dur60) / n,
	}

	// Priority
	var priExact, priBand int
	var priMAE float64
	for _, p := range pairs {
		if p.Pred.Priority == p.Gold.Priority {
			priExact++
		}
		if PriorityMatches(p.Pred.Priority, p.Gold.Priority) {
			priBand++
		}
		priMAE += math.Abs(float64(p.Pred.Priority - p.Gold.Priority))
	}
	priQ := PriorityQuality{
		ExactAcc: float64(priExact) / n,
		BandAcc:  float64(priBand) / n,
		MAE:      priMAE / n,
	}

	// Category: overall accuracy + per-class precision/recall/F1.
	type counts struct{ tp, fp, fn int }
	catCounts := make(map[string]*counts)
	catCorrect := 0
	for _, p := range pairs {
		pred := string(p.Pred.Category)
		gold := string(p.Gold.Category)
		if _, ok := catCounts[pred]; !ok {
			catCounts[pred] = &counts{}
		}
		if _, ok := catCounts[gold]; !ok {
			catCounts[gold] = &counts{}
		}
		if pred == gold {
			catCorrect++
			catCounts[pred].tp++
		} else {
			catCounts[pred].fp++
			catCounts[gold].fn++
		}
	}
	perClass := make(map[string]ClassScore, len(catCounts))
	for cls, cc := range catCounts {
		var cp, cr, cf1 float64
		if cc.tp+cc.fp > 0 {
			cp = float64(cc.tp) / float64(cc.tp+cc.fp)
		}
		if cc.tp+cc.fn > 0 {
			cr = float64(cc.tp) / float64(cc.tp+cc.fn)
		}
		if cp+cr > 0 {
			cf1 = 2 * cp * cr / (cp + cr)
		}
		perClass[cls] = ClassScore{P: cp, R: cr, F1: cf1}
	}
	catQ := CategoryQuality{
		Accuracy: float64(catCorrect) / n,
		PerClass: perClass,
	}

	return FieldQuality{
		Title:           tq,
		Description:     dq,
		DurationMinutes: durQ,
		Priority:        priQ,
		Category:        catQ,
		Deadline:        timeFieldQuality(pairs, func(t Task) *time.Time { return t.Deadline }),
		StartTime:       timeFieldQuality(pairs, func(t Task) *time.Time { return t.StartTime }),
	}
}

func timeFieldQuality(pairs []MatchedPair, extract func(Task) *time.Time) TimeFieldQuality {
	n := float64(len(pairs))
	presenceAgree := 0
	var valueDiffs []float64
	tol1hHit := 0
	tol24hHit := 0

	for _, p := range pairs {
		pred := extract(p.Pred)
		gold := extract(p.Gold)
		if (pred == nil) == (gold == nil) {
			presenceAgree++
		}
		if pred != nil && gold != nil {
			diff := pred.Sub(*gold)
			if diff < 0 {
				diff = -diff
			}
			valueDiffs = append(valueDiffs, diff.Minutes())
			if diff <= time.Hour {
				tol1hHit++
			}
			if diff <= 24*time.Hour {
				tol24hHit++
			}
		}
	}

	q := TimeFieldQuality{PresenceAcc: float64(presenceAgree) / n}
	if len(valueDiffs) > 0 {
		var sum float64
		for _, d := range valueDiffs {
			sum += d
		}
		vn := float64(len(valueDiffs))
		q.ValueMAEMinutes = sum / vn
		q.Tol1hAcc = float64(tol1hHit) / vn
		q.Tol24hAcc = float64(tol24hHit) / vn
	}
	return q
}
