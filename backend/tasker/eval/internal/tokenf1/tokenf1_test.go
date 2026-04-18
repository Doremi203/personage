package tokenf1_test

import (
	"math"
	"testing"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/tokenf1"
)

func approxEqual(a, b, eps float64) bool {
	return math.Abs(a-b) < eps
}

func TestScore(t *testing.T) {
	eps := 1e-9
	tests := []struct {
		name      string
		predicted string
		gold      string
		want      float64
	}{
		{
			name:      "identical strings",
			predicted: "Review Q4 budget proposal",
			gold:      "Review Q4 budget proposal",
			want:      1.0,
		},
		{
			name:      "disjoint token sets",
			predicted: "foo bar baz",
			gold:      "qux quux corge",
			want:      0.0,
		},
		{
			name:      "empty predicted",
			predicted: "",
			gold:      "review budget",
			want:      0.0,
		},
		{
			name:      "empty gold",
			predicted: "review budget",
			gold:      "",
			want:      0.0,
		},
		{
			name:      "both empty",
			predicted: "",
			gold:      "",
			want:      0.0,
		},
		{
			name:      "case and punctuation insensitive",
			predicted: "Review, budget.",
			gold:      "review budget",
			want:      1.0,
		},
		{
			name:      "cyrillic utf-8",
			predicted: "Проверить бюджет",
			gold:      "проверить бюджет",
			want:      1.0,
		},
		{
			name:      "repeated tokens multiset",
			predicted: "foo foo bar",
			gold:      "foo bar bar",
			// common=2 (1 foo + 1 bar), |pred|=3, |gold|=3 → P=2/3, R=2/3, F1=2/3
			want: 2.0 / 3.0,
		},
		{
			name:      "single-letter tokens dropped",
			predicted: "a b c review",
			gold:      "review",
			// only "review" survives in predicted; gold=["review"]; P=1, R=1, F1=1
			want: 1.0,
		},
		{
			name:      "partial overlap",
			predicted: "review budget proposal",
			gold:      "review budget quarterly",
			// common=2, |pred|=3, |gold|=3 → P=2/3, R=2/3, F1=2/3
			want: 2.0 / 3.0,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			got := tokenf1.Score(tc.predicted, tc.gold)
			if !approxEqual(got, tc.want, eps) {
				t.Errorf("Score(%q, %q) = %.6f, want %.6f", tc.predicted, tc.gold, got, tc.want)
			}
		})
	}
}

func TestTokenize(t *testing.T) {
	got := tokenf1.Tokenize("Hello, World! a")
	want := []string{"hello", "world"}
	if len(got) != len(want) {
		t.Fatalf("Tokenize: got %v, want %v", got, want)
	}
	for i := range got {
		if got[i] != want[i] {
			t.Errorf("token[%d]: got %q, want %q", i, got[i], want[i])
		}
	}
}
