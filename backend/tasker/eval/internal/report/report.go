package report

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"time"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/score"
)

// MatchDetail describes one matched pair in the report.
type MatchDetail struct {
	ExpectedIdx  int      `json:"expected_idx"`
	GeneratedID  string   `json:"generated_id"`
	TitleF1      float64  `json:"title_f1"`
	FieldsPassed []string `json:"fields_passed"`
	FieldsFailed []string `json:"fields_failed"`
}

// UnmatchedGenerated describes a generated task with no match.
type UnmatchedGenerated struct {
	ID              string  `json:"id"`
	Title           string  `json:"title"`
	ClosestExpected int     `json:"closest_expected"`
	ClosestTitleF1  float64 `json:"closest_title_f1"`
}

// UnmatchedExpected describes an expected task with no match.
type UnmatchedExpected struct {
	Idx   int    `json:"idx"`
	Title string `json:"title"`
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
	Fixture              string                      `json:"fixture"`
	SnapshotID           string                      `json:"snapshot_id"`
	Timestamp            time.Time                   `json:"timestamp"`
	LLMModel             string                      `json:"llm_model"`
	Counts               Counts                      `json:"counts"`
	Precision            float64                     `json:"precision"`
	Recall               float64                     `json:"recall"`
	F1                   float64                     `json:"f1"`
	MatchedFieldAccuracy map[string]score.FieldScore `json:"matched_field_accuracy"`
	Matches              []MatchDetail               `json:"matches"`
	UnmatchedGenerated   []UnmatchedGenerated        `json:"unmatched_generated"`
	UnmatchedExpected    []UnmatchedExpected         `json:"unmatched_expected"`
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
		return fmt.Errorf("marshal report: %w", err)
	}
	if err := os.WriteFile(path, data, 0600); err != nil { //#nosec G306
		return fmt.Errorf("write report %s: %w", path, err)
	}
	return nil
}

// Summarize prints a short human-readable table of r to w.
func Summarize(w io.Writer, r Report) {
	p := func(format string, args ...any) { _, _ = fmt.Fprintf(w, format, args...) }
	p("=== F1 Eval Report: %s ===\n", r.Fixture)
	p("Snapshot : %s\n", r.SnapshotID)
	p("Model    : %s\n", r.LLMModel)
	p("Tasks    : expected=%d generated=%d TP=%d FP=%d FN=%d\n",
		r.Counts.Expected, r.Counts.Generated, r.Counts.TP, r.Counts.FP, r.Counts.FN)
	p("Precision: %.3f  Recall: %.3f  F1: %.3f\n", r.Precision, r.Recall, r.F1)
	if len(r.MatchedFieldAccuracy) > 0 {
		p("Field accuracy (matched pairs):\n")
		for field, fs := range r.MatchedFieldAccuracy {
			if fs.Total > 0 {
				p("  %-20s %d/%d (%.0f%%)\n", field, fs.Passed, fs.Total, fs.Accuracy()*100)
			}
		}
	}
}
