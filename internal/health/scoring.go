// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"math"
	"sort"
)

// The file score of ADR-EMG-048 as of scoring version 2 (ADR-EMG-059). It
// starts at 10.0; every finding deducts the weight of its code times a severity
// factor (warn 1, high 2). The deductions of one function add up, and the
// functions of a file combine with diminishing returns: the function with the
// largest sum counts fully, the second times 0.5 and every other function times
// 0.25, but the sum of what those others count is capped at TailCap points, so
// that the size of a file alone does not decide its band. The result is clamped
// to 1.0..10.0 and rounded to one decimal (a tie rounds up). The weights, the
// factors and the cap are one definition; a change of any of them bumps
// ScoringVersion in types.go. Version 1 had no cap; scores of the two versions
// are not comparable.
//
// The weights, the factors and the bands are this project's own. They are not
// derived from CodeScene or from any other product.

// weights is the weights table of scoring versions 1 and 2 (ADR-EMG-048); version 2
// changed the combination (TailCap), not the weights.
var weights = map[string]float64{
	CodeComplexMethod:       1.0,
	CodeLargeMethod:         0.8,
	CodeDeepNesting:         1.0,
	CodeBumpyRoad:           1.0,
	CodeComplexConditional:  0.6,
	CodeExcessArguments:     0.5,
	CodePrimitiveObsession:  0.3,
	CodeBrainMethod:         1.5,
	CodeLargeAssertionBlock: 0.3,
	CodeDuplicatedAssertion: 0.3,
}

// Band limits: a score at or above GreenFrom is green, at or above YellowFrom
// yellow, anything lower red (ADR-EMG-048, OWNER items).
const (
	GreenFrom  = 8.0
	YellowFrom = 4.0
	// MinScore and MaxScore bound every file score.
	MinScore = 1.0
	MaxScore = 10.0
	// TailCap is the most that the functions beyond the second may count in
	// total (scoring version 2, ADR-EMG-059).
	TailCap = 2.0
)

// Weights returns a copy of the weights table of ScoringVersion.
//
// TRLC-LINKS: REQ-EMG-151
func Weights() map[string]float64 {
	out := make(map[string]float64, len(weights))
	for code, w := range weights {
		out[code] = w
	}
	return out
}

// SeverityFactor is the multiplier of a severity: warn 1, high 2. An unknown
// severity deducts nothing.
//
// TRLC-LINKS: REQ-EMG-151
func SeverityFactor(s Severity) float64 {
	switch s {
	case SeverityWarn:
		return 1
	case SeverityHigh:
		return 2
	}
	return 0
}

// BandOf returns the band of a score.
//
// TRLC-LINKS: REQ-EMG-151
func BandOf(score float64) string {
	switch {
	case score >= GreenFrom:
		return BandGreen
	case score >= YellowFrom:
		return BandYellow
	}
	return BandRed
}

// RoundTo rounds x half up to the given number of decimals after removing
// floating point noise at the sixth decimal.
//
// TRLC-LINKS: REQ-EMG-151
func RoundTo(x float64, decimals int) float64 {
	x = math.Round(x*1e6) / 1e6
	scale := math.Pow(10, float64(decimals))
	return math.Round(x*scale) / scale
}

// codeOrder maps a code to its position in Codes() for stable sorting.
//
// TRLC-LINKS: REQ-EMG-151
func codeOrder(code string) int {
	for i, c := range Codes() {
		if c == code {
			return i
		}
	}
	return len(Codes())
}

// combineFactor is the diminishing-returns factor of the function at a rank
// (0 is the worst function of the file).
//
// TRLC-LINKS: REQ-EMG-151
func combineFactor(rank int) float64 {
	switch rank {
	case 0:
		return 1
	case 1:
		return 0.5
	}
	return 0.25
}

// scoredDeduction is a deduction with the index of its function.
type scoredDeduction struct {
	Deduction
	fn int // index of the function in FileResult.Functions
}

// functionDeductions lists the deductions of one function and the sum of
// their points; a finding of a code without a weight deducts nothing.
//
// TRLC-LINKS: REQ-EMG-151
func functionDeductions(f Function, index int) ([]scoredDeduction, float64) {
	var deds []scoredDeduction
	sum := 0.0
	for _, finding := range f.Findings {
		points := weights[finding.Code] * SeverityFactor(finding.Severity)
		if points == 0 {
			continue
		}
		deds = append(deds, scoredDeduction{Deduction{Code: finding.Code, Function: f.Name, Severity: finding.Severity, Points: points}, index})
		sum += points
	}
	return deds, sum
}

// collectDeductions lists the deductions of a result and the sum of the points
// of each function (by index) that has any.
//
// TRLC-LINKS: REQ-EMG-151
func collectDeductions(r FileResult) ([]scoredDeduction, map[int]float64) {
	var deds []scoredDeduction
	sums := map[int]float64{}
	for i, f := range r.Functions {
		fd, sum := functionDeductions(f, i)
		if len(fd) == 0 {
			continue
		}
		deds = append(deds, fd...)
		sums[i] = sum
	}
	return deds, sums
}

// rankFunctions orders the function indices by their sum, largest first; ties
// keep source order.
//
// TRLC-LINKS: REQ-EMG-151
func rankFunctions(sums map[int]float64) []int {
	ranked := make([]int, 0, len(sums))
	for i := range sums {
		ranked = append(ranked, i)
	}
	sort.Slice(ranked, func(a, b int) bool {
		if sums[ranked[a]] != sums[ranked[b]] {
			return sums[ranked[a]] > sums[ranked[b]]
		}
		return ranked[a] < ranked[b]
	})
	return ranked
}

// tailStart is the rank from which a function belongs to the tail.
const tailStart = 2

// scaleTail multiplies the Counted of the tail deductions by factor and
// returns the sum of what they counted before.
//
// TRLC-LINKS: REQ-EMG-151
func scaleTail(deds []scoredDeduction, rank map[int]int, factor float64) float64 {
	before := 0.0
	for i := range deds {
		if rank[deds[i].fn] >= tailStart {
			before += deds[i].Counted
			deds[i].Counted *= factor
		}
	}
	return before
}

// countDeductions sets Counted of every deduction: the rank factor of its
// function, with the tail (rank 2 and below) scaled down together when what
// it counts would exceed TailCap, so that the counted amounts add up to what
// is deducted from the file.
//
// TRLC-LINKS: REQ-EMG-151
func countDeductions(deds []scoredDeduction, ranked []int) {
	rank := map[int]int{}
	for r, i := range ranked {
		rank[i] = r
	}
	for i := range deds {
		deds[i].Counted = deds[i].Points * combineFactor(rank[deds[i].fn])
	}
	tail := scaleTail(deds, rank, 1)
	if RoundTo(tail, 6) > TailCap {
		scaleTail(deds, rank, TailCap/tail)
	}
}

// sortDeductions orders deductions by counted amount, then function and code.
//
// TRLC-LINKS: REQ-EMG-151
func sortDeductions(deds []scoredDeduction) {
	sort.SliceStable(deds, func(a, b int) bool {
		x, y := deds[a], deds[b]
		if x.Counted != y.Counted {
			return x.Counted > y.Counted
		}
		if x.Function != y.Function {
			return x.Function < y.Function
		}
		return codeOrder(x.Code) < codeOrder(y.Code)
	})
}

// ScoreFile scores one file result (ADR-EMG-048, ADR-EMG-059). A file without
// findings is 10.0 and green with no deductions. Every deduction carries its
// Points (weight x severity factor) and the Counted amount that the
// combination across functions left of it; before clamping and rounding the
// Counted amounts add up to 10.0 minus the score.
//
// TRLC-LINKS: REQ-EMG-151
// ENGMODEL-LINKS: FU-CODE-HEALTH
func ScoreFile(r FileResult) FileScore {
	deds, sums := collectDeductions(r)
	if len(deds) == 0 {
		return FileScore{Score: MaxScore, Band: BandGreen}
	}
	countDeductions(deds, rankFunctions(sums))
	sortDeductions(deds)

	total := 0.0
	out := make([]Deduction, len(deds))
	for i, d := range deds {
		total += d.Counted
		d.Points, d.Counted = RoundTo(d.Points, 4), RoundTo(d.Counted, 4)
		out[i] = d.Deduction
	}
	score := RoundTo(math.Min(MaxScore, math.Max(MinScore, MaxScore-total)), 1)
	return FileScore{Score: score, Band: BandOf(score), Deductions: out}
}

var _ Scorer = ScoreFile
