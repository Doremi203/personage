package match_test

import (
	"testing"

	"github.com/Doremi203/personage/backend/tasker/eval/internal/match"
)

func buildIdentityCost(n int) [][]float64 {
	cost := make([][]float64, n)
	for i := range cost {
		cost[i] = make([]float64, n)
		for j := range cost[i] {
			if i == j {
				cost[i][j] = 0.0
			} else {
				cost[i][j] = 1.0
			}
		}
	}
	return cost
}

func TestGreedy_PerfectPairing(t *testing.T) {
	cost := buildIdentityCost(3)
	r := match.Greedy(cost, 0.5)

	if len(r.Pairs) != 3 {
		t.Fatalf("expected 3 pairs, got %d", len(r.Pairs))
	}
	if len(r.UnmatchedGenerated) != 0 || len(r.UnmatchedExpected) != 0 {
		t.Errorf("unexpected unmatched: gen=%v exp=%v", r.UnmatchedGenerated, r.UnmatchedExpected)
	}
	for _, p := range r.Pairs {
		if p.GeneratedIdx != p.ExpectedIdx {
			t.Errorf("expected diagonal match, got gen=%d exp=%d", p.GeneratedIdx, p.ExpectedIdx)
		}
		if p.TitleF1 != 1.0 {
			t.Errorf("expected TitleF1=1.0, got %f", p.TitleF1)
		}
	}
}

func TestGreedy_OversupplyGenerated(t *testing.T) {
	// 4 generated × 2 expected; gen 0↔exp0, gen1↔exp1 are perfect matches
	cost := [][]float64{
		{0.0, 1.0},
		{1.0, 0.0},
		{0.4, 0.4},
		{0.4, 0.4},
	}
	r := match.Greedy(cost, 0.5)

	if len(r.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(r.Pairs))
	}
	if len(r.UnmatchedGenerated) != 2 {
		t.Errorf("expected 2 unmatched generated, got %v", r.UnmatchedGenerated)
	}
	if len(r.UnmatchedExpected) != 0 {
		t.Errorf("expected 0 unmatched expected, got %v", r.UnmatchedExpected)
	}
}

func TestGreedy_UndersupplyGenerated(t *testing.T) {
	// 2 generated × 4 expected; gen0↔exp0, gen1↔exp1 are perfect
	cost := [][]float64{
		{0.0, 1.0, 1.0, 1.0},
		{1.0, 0.0, 1.0, 1.0},
	}
	r := match.Greedy(cost, 0.5)

	if len(r.Pairs) != 2 {
		t.Fatalf("expected 2 pairs, got %d", len(r.Pairs))
	}
	if len(r.UnmatchedGenerated) != 0 {
		t.Errorf("expected 0 unmatched gen, got %v", r.UnmatchedGenerated)
	}
	if len(r.UnmatchedExpected) != 2 {
		t.Errorf("expected 2 unmatched exp, got %v", r.UnmatchedExpected)
	}
}

func TestGreedy_AllAboveThreshold(t *testing.T) {
	cost := [][]float64{
		{0.6, 0.7},
		{0.8, 0.9},
	}
	r := match.Greedy(cost, 0.5)

	if len(r.Pairs) != 0 {
		t.Fatalf("expected 0 pairs, got %d", len(r.Pairs))
	}
	if len(r.UnmatchedGenerated) != 2 || len(r.UnmatchedExpected) != 2 {
		t.Errorf("expected all unmatched: gen=%v exp=%v", r.UnmatchedGenerated, r.UnmatchedExpected)
	}
}

func TestGreedy_Determinism(t *testing.T) {
	// Equal costs — should always produce the same pairing
	cost := [][]float64{
		{0.3, 0.3},
		{0.3, 0.3},
	}
	r1 := match.Greedy(cost, 0.5)
	r2 := match.Greedy(cost, 0.5)

	if len(r1.Pairs) != len(r2.Pairs) {
		t.Fatalf("non-deterministic: run1=%d pairs, run2=%d pairs", len(r1.Pairs), len(r2.Pairs))
	}
	for i := range r1.Pairs {
		if r1.Pairs[i] != r2.Pairs[i] {
			t.Errorf("pair[%d] differs between runs: %+v vs %+v", i, r1.Pairs[i], r2.Pairs[i])
		}
	}
}

func TestGreedy_EmptyInputs(t *testing.T) {
	r := match.Greedy(nil, 0.5)
	if len(r.Pairs) != 0 || len(r.UnmatchedGenerated) != 0 || len(r.UnmatchedExpected) != 0 {
		t.Error("expected empty result for nil cost")
	}

	r2 := match.Greedy([][]float64{}, 0.5)
	if len(r2.Pairs) != 0 {
		t.Error("expected empty result for empty cost")
	}
}
