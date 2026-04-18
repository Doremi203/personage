package tokenf1

import (
	"strings"
	"unicode"
)

// Tokenize lowercases s, splits on whitespace and Unicode punctuation,
// and drops tokens of length <= 1.
func Tokenize(s string) []string {
	s = strings.ToLower(s)
	fields := strings.FieldsFunc(s, func(r rune) bool {
		return unicode.IsSpace(r) || unicode.IsPunct(r)
	})
	out := make([]string, 0, len(fields))
	for _, f := range fields {
		if len([]rune(f)) > 1 {
			out = append(out, f)
		}
	}
	return out
}

// Score returns the token-level F1 between predicted and gold strings.
// Empty predicted or gold → 0.
func Score(predicted, gold string) float64 {
	p := Tokenize(predicted)
	g := Tokenize(gold)
	if len(p) == 0 || len(g) == 0 {
		return 0
	}

	predCount := map[string]int{}
	for _, t := range p {
		predCount[t]++
	}
	goldCount := map[string]int{}
	for _, t := range g {
		goldCount[t]++
	}

	common := 0
	for t, pc := range predCount {
		gc, ok := goldCount[t]
		if !ok {
			continue
		}
		if pc < gc {
			common += pc
		} else {
			common += gc
		}
	}
	if common == 0 {
		return 0
	}

	precision := float64(common) / float64(len(p))
	recall := float64(common) / float64(len(g))
	return 2 * precision * recall / (precision + recall)
}
