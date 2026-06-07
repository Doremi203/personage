package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Doremi203/personage/backend/libs/go/errors"
	"github.com/Doremi203/personage/backend/tasker/eval/internal/score"
)

// TaskSnapshot holds a complete task record for human-readable report output.
type TaskSnapshot struct {
	ID              string     `json:"id,omitzero"`
	UserID          string     `json:"user_id,omitzero"`
	ClusterID       *string    `json:"cluster_id,omitzero"`
	Title           string     `json:"title"`
	Description     string     `json:"description"`
	DurationMinutes int        `json:"duration_minutes"`
	Priority        int        `json:"priority"`
	Deadline        *time.Time `json:"deadline,omitzero"`
	StartTime       *time.Time `json:"start_time,omitzero"`
	EndTime         *time.Time `json:"end_time,omitzero"`
	Category        string     `json:"category"`
}

type ClusterSnapshot struct {
	ID                 string    `json:"id"`
	UserID             string    `json:"user_id"`
	Status             string    `json:"status"`
	EventCount         int       `json:"event_count"`
	GenerationOutcome  *string   `json:"generation_outcome,omitzero"`
	GenerationReason   *string   `json:"generation_reason,omitzero"`
	GeneratedTaskCount int       `json:"generated_task_count"`
	CreatedAt          time.Time `json:"created_at"`
	UpdatedAt          time.Time `json:"updated_at"`
}

type ClusterDiagnostics struct {
	Total                int               `json:"total"`
	Closed               int               `json:"closed"`
	SkippedNonActionable int               `json:"skipped_non_actionable"`
	TasklessClusterRate  float64           `json:"taskless_cluster_rate"`
	Clusters             []ClusterSnapshot `json:"clusters"`
}

// MatchDetail describes one matched pair in the report.
type MatchDetail struct {
	ExpectedID        string       `json:"expected_id"`
	GeneratedID       string       `json:"generated_id"`
	TitleF1           float64      `json:"title_f1"`
	TitleEmbeddingSim float64      `json:"title_embedding_sim,omitzero"`
	FieldsPassed      []string     `json:"fields_passed"`
	FieldsFailed      []string     `json:"fields_failed"`
	Generated         TaskSnapshot `json:"generated"`
	Expected          TaskSnapshot `json:"expected"`
}

// UnmatchedGenerated describes a generated task with no match.
type UnmatchedGenerated struct {
	ID                       string       `json:"id"`
	Title                    string       `json:"title"`
	ClosestExpected          string       `json:"closest_expected"`
	ClosestTitleF1           float64      `json:"closest_title_f1"`
	ClosestTitleEmbeddingSim float64      `json:"closest_title_embedding_sim,omitzero"`
	Task                     TaskSnapshot `json:"task"`
}

// UnmatchedExpected describes an expected task with no match.
type UnmatchedExpected struct {
	ID    string       `json:"id"`
	Title string       `json:"title"`
	Task  TaskSnapshot `json:"task"`
}

// Counts holds task-level matching counts.
type Counts struct {
	Expected  int `json:"expected"`
	Generated int `json:"generated"`
	TP        int `json:"tp"`
	FP        int `json:"fp"`
	FN        int `json:"fn"`
}

// Report is the full JSON output written by the eval script.
type Report struct {
	Fixture    string    `json:"fixture"`
	SnapshotID string    `json:"snapshot_id"`
	Timestamp  time.Time `json:"timestamp"`
	LLMModel   string    `json:"llm_model"`
	// MatcherType is "embedding" when embedding-based title matching was used,
	// "tokenf1" otherwise.
	MatcherType        string               `json:"matcher_type"`
	Counts             Counts               `json:"counts"`
	Precision          float64              `json:"precision"`
	Recall             float64              `json:"recall"`
	F1                 float64              `json:"f1"`
	FieldQuality       score.FieldQuality   `json:"field_quality"`
	ClusterDiagnostics ClusterDiagnostics   `json:"cluster_diagnostics"`
	Matches            []MatchDetail        `json:"matches"`
	UnmatchedGenerated []UnmatchedGenerated `json:"unmatched_generated"`
	UnmatchedExpected  []UnmatchedExpected  `json:"unmatched_expected"`
}

// Compute calculates Precision, Recall, F1 from counts and sets them.
func (r *Report) Compute() {
	tp := float64(r.Counts.TP)
	fp := float64(r.Counts.FP)
	fn := float64(r.Counts.FN)

	if tp+fp > 0 {
		r.Precision = tp / (tp + fp)
	}
	if tp+fn > 0 {
		r.Recall = tp / (tp + fn)
	}
	if r.Precision+r.Recall > 0 {
		r.F1 = 2 * r.Precision * r.Recall / (r.Precision + r.Recall)
	}
}

// Write serialises r as indented JSON to path (creates or overwrites).
func Write(path string, r Report) error {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return errors.WrapFail(err, "marshal report")
	}
	if err := os.WriteFile(path, data, 0600); err != nil { //#nosec G306
		return errors.WrapFailf(err, "write report %v", errors.Token("path", path))
	}
	return nil
}

// Summarize prints a short human-readable table of r to w.
func Summarize(w io.Writer, r Report) {
	p := func(format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }
	p("=== F1 Eval Report: %s ===\n", r.Fixture)
	p("Snapshot : %s\n", r.SnapshotID)
	p("Model    : %s\n", r.LLMModel)
	matcher := r.MatcherType
	if matcher == "" {
		matcher = "tokenf1"
	}
	p("Matcher  : %s\n", matcher)
	p("Tasks    : expected=%d generated=%d TP=%d FP=%d FN=%d\n",
		r.Counts.Expected, r.Counts.Generated, r.Counts.TP, r.Counts.FP, r.Counts.FN)
	p("Precision: %.3f  Recall: %.3f  F1: %.3f\n", r.Precision, r.Recall, r.F1)
	p(
		"Clusters : total=%d closed=%d skipped_non_actionable=%d taskless_rate=%.3f\n",
		r.ClusterDiagnostics.Total,
		r.ClusterDiagnostics.Closed,
		r.ClusterDiagnostics.SkippedNonActionable,
		r.ClusterDiagnostics.TasklessClusterRate,
	)

	fq := r.FieldQuality
	if r.Counts.TP > 0 {
		p("Field quality (matched pairs=%d):\n", r.Counts.TP)
		if fq.Title.EmbeddingSimMean > 0 {
			p("  %-20s token_f1=%.3f  emb_sim=%.3f\n", "title", fq.Title.TokenF1Mean, fq.Title.EmbeddingSimMean)
		} else {
			p("  %-20s token_f1=%.3f\n", "title", fq.Title.TokenF1Mean)
		}
		p("  %-20s token_f1=%.3f\n", "description", fq.Description.TokenF1Mean)
		p("  %-20s mae=%.1fmin  rel_err=%.0f%%  tol_30=%.0f%%  tol_60=%.0f%%\n",
			"duration_minutes",
			fq.DurationMinutes.MAE,
			fq.DurationMinutes.RelErrorMean*100,
			fq.DurationMinutes.Tol30Acc*100,
			fq.DurationMinutes.Tol60Acc*100,
		)
		p("  %-20s exact=%.0f%%  band=%.0f%%  mae=%.1f\n",
			"priority",
			fq.Priority.ExactAcc*100,
			fq.Priority.BandAcc*100,
			fq.Priority.MAE,
		)
		p("  %-20s accuracy=%.0f%%\n", "category", fq.Category.Accuracy*100)
		for cls, cs := range fq.Category.PerClass {
			p("    %-18s p=%.0f%%  r=%.0f%%  f1=%.0f%%\n", cls, cs.P*100, cs.R*100, cs.F1*100)
		}
		printTimeField(w, "deadline", fq.Deadline)
		printTimeField(w, "start_time", fq.StartTime)
	}
}

func printTimeField(w io.Writer, name string, q score.TimeFieldQuality) {
	p := func(format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }
	if q.ValueMAEMinutes > 0 {
		p("  %-20s presence=%.0f%%  value_mae=%.1fmin  tol_1h=%.0f%%  tol_24h=%.0f%%\n",
			name,
			q.PresenceAcc*100,
			q.ValueMAEMinutes,
			q.Tol1hAcc*100,
			q.Tol24hAcc*100,
		)
	} else {
		p("  %-20s presence=%.0f%%\n", name, q.PresenceAcc*100)
	}
}
