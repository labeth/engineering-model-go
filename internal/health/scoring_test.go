// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"fmt"
	"math"
	"reflect"
	"testing"
)

// near compares scores that were computed from decimal weights.
func near(a, b float64) bool { return math.Abs(a-b) < 1e-9 }

// finding builds a finding of a code and severity.
func finding(code string, sev Severity) Finding {
	return Finding{Code: code, Severity: sev, StartLine: 1, EndLine: 2, Metric: "m", Value: 1, Message: code}
}

// fnWith builds a function with findings.
func fnWith(name string, start int, fs ...Finding) Function {
	return Function{Name: name, Kind: "function", StartLine: start, EndLine: start + 9, LOC: 10, Findings: fs}
}

// The published weights table of scoring version 2 and the severity factors.
// The weights are those of version 1; version 2 changed the combination (the
// tail cap), not the weights. Mutation: changing any weight, or the cap,
// without bumping the version fails this test.
//
// TRLC-LINKS: REQ-EMG-151
func TestWeightsTableIsScoringVersion2(t *testing.T) {
	want := map[string]float64{
		"health.complex_method": 1.0, "health.large_method": 0.8, "health.deep_nesting": 1.0,
		"health.bumpy_road": 1.0, "health.complex_conditional": 0.6, "health.excess_arguments": 0.5,
		"health.primitive_obsession": 0.3, "health.brain_method": 1.5,
		"health.large_assertion_block": 0.3, "health.duplicated_assertion_block": 0.3,
	}
	if ScoringVersion != 2 || TailCap != 2.0 || !reflect.DeepEqual(Weights(), want) {
		t.Fatalf("weights = %v (version %d, tail cap %v), want %v at version 2 with cap 2.0", Weights(), ScoringVersion, TailCap, want)
	}
	for _, code := range Codes() {
		if _, ok := want[code]; !ok {
			t.Errorf("code %s has no weight", code)
		}
	}
	w := Weights()
	w[CodeComplexMethod] = 99
	if Weights()[CodeComplexMethod] != 1.0 {
		t.Error("Weights returned the shared table")
	}
	if SeverityFactor(SeverityWarn) != 1 || SeverityFactor(SeverityHigh) != 2 || SeverityFactor("other") != 0 {
		t.Error("severity factors are not warn 1, high 2")
	}
}

// A file without findings is 10.0 and green with no deductions.
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileWithoutFindingsIsTenGreen(t *testing.T) {
	got := ScoreFile(FileResult{Path: "a.go", Functions: []Function{fnWith("f", 1)}})
	if got.Score != 10.0 || got.Band != BandGreen || len(got.Deductions) != 0 {
		t.Fatalf("got %+v, want 10.0 green without deductions", got)
	}
	if got := ScoreFile(FileResult{Path: "empty.go"}); got.Score != 10.0 || got.Band != BandGreen {
		t.Fatalf("a file without functions: %+v", got)
	}
}

// The worked example of ADR-EMG-048, computed by hand.
//
//	function A: complex_method high  1.0 x 2 = 2.0
//	            deep_nesting   warn  1.0 x 1 = 1.0   sum 3.0   rank 0 -> x1
//	function B: large_method   warn  0.8 x 1 = 0.8   sum 0.8   rank 1 -> x0.5
//	function C: excess_arguments warn 0.5 x 1 = 0.5  sum 0.5   rank 2 -> x0.25
//	counted: 2.0 + 1.0 + 0.4 + 0.125 = 3.525
//	score:   10.0 - 3.525 = 6.475 -> 6.5 (half up), yellow
//
// The functions are given in the order C, A, B to show that the rank follows
// the sums and not the source order.
//
// Mutation: ranking by source order, or applying 0.5 to all but the first,
// changes the number and fails this test.
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileWorkedExample(t *testing.T) {
	r := FileResult{Path: "w.go", Functions: []Function{
		fnWith("C", 1, finding(CodeExcessArguments, SeverityWarn)),
		fnWith("A", 20, finding(CodeComplexMethod, SeverityHigh), finding(CodeDeepNesting, SeverityWarn)),
		fnWith("B", 40, finding(CodeLargeMethod, SeverityWarn)),
	}}
	got := ScoreFile(r)
	if got.Score != 6.5 || got.Band != BandYellow {
		t.Fatalf("score = %v %s, want 6.5 yellow", got.Score, got.Band)
	}
	want := []Deduction{
		{Code: CodeComplexMethod, Function: "A", Severity: SeverityHigh, Points: 2.0, Counted: 2.0},
		{Code: CodeDeepNesting, Function: "A", Severity: SeverityWarn, Points: 1.0, Counted: 1.0},
		{Code: CodeLargeMethod, Function: "B", Severity: SeverityWarn, Points: 0.8, Counted: 0.4},
		{Code: CodeExcessArguments, Function: "C", Severity: SeverityWarn, Points: 0.5, Counted: 0.125},
	}
	if !reflect.DeepEqual(got.Deductions, want) {
		t.Fatalf("deductions = %+v, want %+v", got.Deductions, want)
	}
	total := 0.0
	for _, d := range got.Deductions {
		total += d.Counted
	}
	if !near(total, 3.525) {
		t.Errorf("counted deductions add up to %v, want 3.525", total)
	}
}

// Three smells in one function and one in another (REQ-EMG-151): the deductions
// list each smell, and the counted amounts add up to 10.0 minus the score.
//
//	F: complex_method warn 1.0 + deep_nesting warn 1.0 + bumpy_road warn 1.0 = 3.0, x1
//	G: large_method warn 0.8, x0.5 = 0.4
//	score = 10.0 - 3.4 = 6.6
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileDeductionsSumToTheScore(t *testing.T) {
	got := ScoreFile(FileResult{Functions: []Function{
		fnWith("F", 1, finding(CodeComplexMethod, SeverityWarn), finding(CodeDeepNesting, SeverityWarn), finding(CodeBumpyRoad, SeverityWarn)),
		fnWith("G", 20, finding(CodeLargeMethod, SeverityWarn)),
	}})
	if got.Score != 6.6 || len(got.Deductions) != 4 {
		t.Fatalf("got %+v, want 6.6 with four deductions", got)
	}
	total := 0.0
	for _, d := range got.Deductions {
		total += d.Counted
	}
	if !near(10.0-total, got.Score) {
		t.Errorf("10.0 - counted (%v) = %v, want the score %v", total, 10.0-total, got.Score)
	}
}

// Diminishing returns over four functions with one warn complex_method each
// (equal sums keep the source order): 1.0 + 0.5 + 0.25 + 0.25 = 2.0, so the
// score is 8.0, exactly on the green limit.
//
// Mutation: counting every later function at 0.5, or the rest at 0.5, fails.
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileDiminishingReturnsAndGreenBoundary(t *testing.T) {
	var fns []Function
	for i, name := range []string{"f1", "f2", "f3", "f4"} {
		fns = append(fns, fnWith(name, 1+i*20, finding(CodeComplexMethod, SeverityWarn)))
	}
	got := ScoreFile(FileResult{Functions: fns})
	if got.Score != 8.0 || got.Band != BandGreen {
		t.Fatalf("got %v %s, want 8.0 green", got.Score, got.Band)
	}
	counted := []float64{1.0, 0.5, 0.25, 0.25}
	for i, d := range got.Deductions {
		if d.Function != fns[i].Name || d.Counted != counted[i] {
			t.Errorf("deduction %d = %+v, want %s counted %v", i, d, fns[i].Name, counted[i])
		}
	}
}

// Clamping: every code high in one function deducts (1.0+0.8+1.0+1.0+0.6+0.5+
// 0.3+1.5+0.3+0.3 = 7.3) x 2 = 14.6, so the score is clamped to 1.0 and red;
// the counted amounts still add up to 14.6.
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileClampsAtOne(t *testing.T) {
	var fs []Finding
	for _, code := range Codes() {
		fs = append(fs, finding(code, SeverityHigh))
	}
	got := ScoreFile(FileResult{Functions: []Function{fnWith("everything", 1, fs...)}})
	if got.Score != 1.0 || got.Band != BandRed || len(got.Deductions) != 10 {
		t.Fatalf("got %v %s with %d deductions, want 1.0 red with 10", got.Score, got.Band, len(got.Deductions))
	}
	total := 0.0
	for _, d := range got.Deductions {
		total += d.Counted
	}
	if !near(total, 14.6) {
		t.Errorf("counted = %v, want 14.6", total)
	}
}

// Rounding to one decimal is half up. Function A has a warn brain_method
// (1.5) and function B a warn excess_arguments (0.5 x 0.5 = 0.25), so the
// counted amounts are 1.75 and the score 10.0 - 1.75 = 8.25, which rounds to 8.3
// (round-half-even would give 8.2), green. A finding of an unknown code deducts
// nothing and does not make the function count.
//
// Mutation: rounding half down or to even fails.
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileRoundsHalfUp(t *testing.T) {
	a := fnWith("A", 1, finding(CodeBrainMethod, SeverityWarn), finding("health.unknown", SeverityHigh))
	b := fnWith("B", 20, finding(CodeExcessArguments, SeverityWarn))
	c := fnWith("C", 40, finding("health.unknown", SeverityHigh))
	got := ScoreFile(FileResult{Functions: []Function{a, b, c}})
	if got.Score != 8.3 || got.Band != BandGreen || len(got.Deductions) != 2 {
		t.Fatalf("got %+v, want 8.3 green with two deductions", got)
	}
}

// tailFile is a file of n functions that each have one warn complex_method
// (1.0 point). All sums are equal, so the rank follows the source order.
//
// TRLC-LINKS: REQ-EMG-151
func tailFile(n int) FileResult {
	var fns []Function
	for i := 0; i < n; i++ {
		fns = append(fns, fnWith(fmt.Sprintf("f%02d", i), 1+i*20, finding(CodeComplexMethod, SeverityWarn)))
	}
	return FileResult{Path: "tail.go", Functions: fns}
}

// countedSum adds up the counted amounts of a score.
//
// TRLC-LINKS: REQ-EMG-151
func countedSum(s FileScore) float64 {
	total := 0.0
	for _, d := range s.Deductions {
		total += d.Counted
	}
	return total
}

// The cap is exactly at the boundary: ten functions with one warn complex_method
// each count 1.0 + 0.5 + 8 x 0.25 = 3.5, the tail is exactly 2.0 and not scaled,
// so every tail deduction counts 0.25 and the score is 6.5. With nine tail
// functions (11 in all) the tail would be 2.25 and is scaled by 2.0/2.25, so the
// score is the same 6.5 and each tail deduction counts 0.2222.
//
// Mutation: a cap that applies at "at least 2.0", a cap of another size, or a
// cap that also touches the second function fails.
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileTailCapBoundary(t *testing.T) {
	at := ScoreFile(tailFile(10))
	if at.Score != 6.5 || at.Band != BandYellow {
		t.Fatalf("ten functions: %v %s, want 6.5 yellow", at.Score, at.Band)
	}
	for i, d := range at.Deductions {
		want := 0.25
		switch i {
		case 0:
			want = 1.0
		case 1:
			want = 0.5
		}
		if d.Counted != want {
			t.Errorf("ten functions: deduction %d (%s) counted %v, want %v", i, d.Function, d.Counted, want)
		}
	}
	over := ScoreFile(tailFile(11))
	if over.Score != 6.5 {
		t.Fatalf("eleven functions: %v, want the same 6.5", over.Score)
	}
	for i, d := range over.Deductions {
		want := 2.0 / 9
		switch i {
		case 0:
			want = 1.0
		case 1:
			want = 0.5
		}
		if math.Abs(d.Counted-want) > 5e-5 {
			t.Errorf("eleven functions: deduction %d counted %v, want %.4f", i, d.Counted, want)
		}
	}
}

// Monotone and bounded: adding mediocre functions never raises the score, five
// functions score 10.0 - (1.0 + 0.5 + 3 x 0.25) = 7.75 -> 7.8, thirty functions
// score no higher than that, and since the tail is capped they score exactly
// what ten do, 6.5 (10.0 - 1.0 - 0.5 - 2.0), not 10.0 - 1.5 - 7.0 = 1.5 as in
// scoring version 1.
//
// Mutation: no cap (the score keeps falling), or a cap that is lifted for large
// files, fails.
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileTailCapIsMonotoneAndBounded(t *testing.T) {
	prev := MaxScore
	for n := 1; n <= 60; n++ {
		got := ScoreFile(tailFile(n)).Score
		if got > prev {
			t.Fatalf("%d functions score %v, more than %d functions (%v)", n, got, n-1, prev)
		}
		if floor := MaxScore - 1.0 - 0.5 - TailCap; got < floor {
			t.Fatalf("%d functions score %v, below the bound %v", n, got, floor)
		}
		prev = got
	}
	five, ten, thirty := ScoreFile(tailFile(5)), ScoreFile(tailFile(10)), ScoreFile(tailFile(30))
	if five.Score != 7.8 || ten.Score != 6.5 || thirty.Score != 6.5 || thirty.Score > five.Score {
		t.Errorf("scores of 5, 10 and 30 functions = %v, %v, %v; want 7.8, 6.5, 6.5", five.Score, ten.Score, thirty.Score)
	}
}

// The counted amounts stay honest when the cap bites: the worst function and
// the second keep their factors, and the tail entries are scaled together, so
// they keep their proportions and everything adds up to what was deducted.
//
//	A: brain_method high 3.0, rank 0, counts 3.0
//	B: brain_method high 3.0, rank 1, counts 1.5
//	T1..T3: brain_method high 3.0, counted 0.75 before the cap
//	T4: brain_method warn 1.5, counted 0.375 before the cap
//	tail before the cap 3 x 0.75 + 0.375 = 2.625, scale 2.0/2.625 = 0.7619
//	T1..T3 count 0.5714, T4 counts 0.2857 (still half of T1)
//	deducted 3.0 + 1.5 + 2.0 = 6.5, score 3.5, red
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileCappedTailKeepsCountedHonest(t *testing.T) {
	r := FileResult{Functions: []Function{
		fnWith("A", 1, finding(CodeBrainMethod, SeverityHigh)),
		fnWith("B", 20, finding(CodeBrainMethod, SeverityHigh)),
		fnWith("T1", 40, finding(CodeBrainMethod, SeverityHigh)),
		fnWith("T2", 60, finding(CodeBrainMethod, SeverityHigh)),
		fnWith("T3", 80, finding(CodeBrainMethod, SeverityHigh)),
		fnWith("T4", 100, finding(CodeBrainMethod, SeverityWarn)),
	}}
	got := ScoreFile(r)
	if got.Score != 3.5 || got.Band != BandRed {
		t.Fatalf("score = %v %s, want 3.5 red", got.Score, got.Band)
	}
	counted := map[string]float64{}
	for _, d := range got.Deductions {
		counted[d.Function] = d.Counted
	}
	if counted["A"] != 3.0 || counted["B"] != 1.5 {
		t.Errorf("A and B count %v and %v, want 3.0 and 1.5", counted["A"], counted["B"])
	}
	for _, name := range []string{"T1", "T2", "T3"} {
		if math.Abs(counted[name]-0.5714) > 1e-4 {
			t.Errorf("%s counts %v, want 0.5714", name, counted[name])
		}
	}
	if math.Abs(counted["T4"]-0.2857) > 1e-4 || math.Abs(counted["T1"]/counted["T4"]-2) > 1e-3 {
		t.Errorf("T4 counts %v against %v for T1, want half", counted["T4"], counted["T1"])
	}
	if sum := countedSum(got); math.Abs(sum-6.5) > 5e-4 || !near(RoundTo(10.0-sum, 1), got.Score) {
		t.Errorf("counted amounts add up to %v, want 6.5", sum)
	}
	for _, d := range got.Deductions {
		if d.Counted > d.Points {
			t.Errorf("%s counts %v, more than its %v points", d.Function, d.Counted, d.Points)
		}
	}
}

// The yellow boundary through the cap: A is a high brain_method (3.0, counts
// 3.0), B a high complex_method (2.0, rank 1, counts 1.0) and six more functions
// with a high complex_method (2.0 each, equal sums rank in source order) would
// count 6 x 0.5 = 3.0 in the tail but are capped at 2.0, so 3.0 + 1.0 + 2.0 = 6.0
// is deducted and the score is exactly 4.0, yellow. Without the cap it would be
// 10.0 - 7.0 = 3.0, red.
//
// Mutation: dropping the cap turns the band red.
//
// TRLC-LINKS: REQ-EMG-151
func TestScoreFileCapMovesTheBandBoundary(t *testing.T) {
	fns := []Function{
		fnWith("A", 1, finding(CodeBrainMethod, SeverityHigh)),
		fnWith("B", 20, finding(CodeComplexMethod, SeverityHigh)),
	}
	for i := 0; i < 6; i++ {
		fns = append(fns, fnWith(fmt.Sprintf("T%d", i), 40+i*20, finding(CodeComplexMethod, SeverityHigh)))
	}
	got := ScoreFile(FileResult{Functions: fns})
	if got.Score != 4.0 || got.Band != BandYellow {
		t.Fatalf("score = %v %s, want 4.0 yellow", got.Score, got.Band)
	}
	if sum := countedSum(got); math.Abs(sum-6.0) > 5e-4 {
		t.Errorf("counted amounts add up to %v, want 6.0", sum)
	}
}

// The band limits: 8.0 green, 7.9 yellow, 4.0 yellow, 3.9 red.
//
// TRLC-LINKS: REQ-EMG-151
func TestBandOfLimits(t *testing.T) {
	for score, want := range map[float64]string{10.0: BandGreen, 8.0: BandGreen, 7.9: BandYellow, 4.0: BandYellow, 3.9: BandRed, 1.0: BandRed} {
		if got := BandOf(score); got != want {
			t.Errorf("BandOf(%v) = %s, want %s", score, got, want)
		}
	}
}

// ScoreFile is a pure function of the result: the same input gives the same
// output, and the input is not changed.
//
// TRLC-LINKS: REQ-EMG-151, REQ-EMG-159
func TestScoreFileIsDeterministic(t *testing.T) {
	r := FileResult{Functions: []Function{
		fnWith("x", 1, finding(CodeBumpyRoad, SeverityHigh), finding(CodeComplexConditional, SeverityWarn)),
		fnWith("y", 20, finding(CodeBumpyRoad, SeverityHigh), finding(CodeComplexConditional, SeverityWarn)),
	}}
	first := ScoreFile(r)
	for i := 0; i < 20; i++ {
		if got := ScoreFile(r); !reflect.DeepEqual(got, first) {
			t.Fatalf("run %d differs: %+v vs %+v", i, got, first)
		}
	}
	if r.Functions[0].Findings[0].Code != CodeBumpyRoad {
		t.Error("ScoreFile changed its input")
	}
}
