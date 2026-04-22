package match

import "sort"

// Pair is one matched (generatedIdx, expectedIdx) pair with the title F1 used to form it.
type Pair struct {
	GeneratedIdx int
	ExpectedIdx  int
	TitleF1      float64
}

// Result holds matched pairs and unmatched indices on each side.
type Result struct {
	Pairs              []Pair
	UnmatchedGenerated []int
	UnmatchedExpected  []int
}

// Greedy matches generated tasks to expected tasks using the supplied cost
// matrix (cost[i][j] = 1 - title_f1). A pair with cost > maxCost is not a
// match. Ties broken by lower (generatedIdx, expectedIdx) for determinism.
func Greedy(cost [][]float64, maxCost float64) Result {
	n := len(cost)
	m := 0
	if n > 0 {
		m = len(cost[0])
	}

	type cand struct {
		g, e int
		c    float64
	}
	cands := make([]cand, 0, n*m)
	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			if cost[i][j] <= maxCost {
				cands = append(cands, cand{i, j, cost[i][j]})
			}
		}
	}
	sort.SliceStable(cands, func(a, b int) bool {
		if cands[a].c != cands[b].c {
			return cands[a].c < cands[b].c
		}
		if cands[a].g != cands[b].g {
			return cands[a].g < cands[b].g
		}
		return cands[a].e < cands[b].e
	})

	usedG := make([]bool, n)
	usedE := make([]bool, m)
	var pairs []Pair
	for _, c := range cands {
		if usedG[c.g] || usedE[c.e] {
			continue
		}
		usedG[c.g] = true
		usedE[c.e] = true
		pairs = append(pairs, Pair{c.g, c.e, 1 - c.c})
	}

	var ug []int
	for i, u := range usedG {
		if !u {
			ug = append(ug, i)
		}
	}
	var ue []int
	for j, u := range usedE {
		if !u {
			ue = append(ue, j)
		}
	}
	return Result{Pairs: pairs, UnmatchedGenerated: ug, UnmatchedExpected: ue}
}
