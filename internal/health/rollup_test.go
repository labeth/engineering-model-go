// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"testing"
)

// rollupFiles is a small synthetic project.
//
//	a1.go  FU-A  LOC 100 score 9.0 green   functions Foo 10-30, Inner 15-20 (inside Foo)
//	a2.go  FU-A  LOC 300 score 5.0 yellow  function Bar 5-20 with two findings
//	b.go   FU-B  LOC  50 score 3.0 red     function Baz 1-9
//	c.go   none  LOC  50 score 10.0 green
//	a_test.go    test, LOC 80 score 6.0 (owner FU-A, never part of a unit)
func rollupFiles() []FileReport {
	two := []Finding{{Code: CodeComplexMethod, Severity: SeverityHigh}, {Code: CodeLargeMethod, Severity: SeverityWarn}}
	return []FileReport{
		{Path: "a1.go", LOC: 100, Score: 9.0, Band: BandGreen, Functions: []Function{
			{Name: "Foo", StartLine: 10, EndLine: 30, Findings: []Finding{{Code: CodeBumpyRoad, Severity: SeverityWarn}}},
			{Name: "Foo.Inner", StartLine: 15, EndLine: 20},
		}},
		{Path: "a2.go", LOC: 300, Score: 5.0, Band: BandYellow, Functions: []Function{{Name: "Bar", StartLine: 5, EndLine: 20, Findings: two}}},
		{Path: "a_test.go", LOC: 80, Score: 6.0, Band: BandYellow, IsTest: true, Functions: []Function{{Name: "TestBar", StartLine: 3, EndLine: 40}}},
		{Path: "b.go", LOC: 50, Score: 3.0, Band: BandRed, Functions: []Function{{Name: "Baz", StartLine: 1, EndLine: 9}}},
		{Path: "c.go", LOC: 50, Score: 10.0, Band: BandGreen},
	}
}

// Roll-ups on a synthetic model (REQ-EMG-152), computed by hand:
//
//	FU-A: (9.0 x 100 + 5.0 x 300) / 400 = 2400 / 400 = 6.0 yellow, worst a2.go 5.0
//	FU-B: 3.0 red, worst b.go
//	project: (900 + 1500 + 150 + 500) / (100 + 300 + 50 + 50) = 3050 / 500 = 6.1 yellow,
//	         worst b.go 3.0, bands green 2 (a1, c), yellow 1 (a2), red 1 (b)
//	tests: 6.0 yellow, one file
//
// Mutation: an unweighted mean (5.5 for FU-A), counting the test file in the
// unit or project, or attributing c.go to a unit fails.
//
// TRLC-LINKS: REQ-EMG-152
func TestBuildRollupsUnitsProjectAndTests(t *testing.T) {
	owners := map[string]string{"a1.go": "FU-A", "a2.go": "FU-A", "a_test.go": "FU-A", "b.go": "FU-B", "c.go": "unresolved"}
	r := BuildRollups(rollupFiles(), owners, nil)

	mustEqual(t, len(r.Units), 2)
	wantEqual(t, [2]string{r.Units[0].Unit, r.Units[1].Unit}, [2]string{"FU-A", "FU-B"})
	a := r.Units[0]
	wantEqual(t, [5]any{a.Score, a.Band, a.Files, a.LOC, a.Worst.Path}, [5]any{6.0, BandYellow, 2, 400, "a2.go"})
	wantEqual(t, a.Worst.Score, 5.0)
	b := r.Units[1]
	wantEqual(t, [3]any{b.Score, b.Band, b.Worst.Path}, [3]any{3.0, BandRed, "b.go"})
	p := r.Project
	wantEqual(t, [6]any{p.Score, p.Band, p.Files, p.LOC, p.Worst.Path, p.Bands}, [6]any{6.1, BandYellow, 4, 500, "b.go", BandCounts{Green: 2, Yellow: 1, Red: 1}})
	mustEqual(t, r.Tests != nil, true)
	wantEqual(t, [3]any{r.Tests.Score, r.Tests.Files, r.Tests.Worst.Path}, [3]any{6.0, 1, "a_test.go"})
	wantEqual(t, len(r.Requirements), 0) // no links
}

// A requirement reaches the functions that marker-linked symbols lie in, by
// path and line range: the innermost function holds the symbol line, a symbol
// outside every function or in a test file reaches nothing, two symbols in
// one function count it once, and the requirement reports the number of
// functions and findings and the worst score of the files of those functions.
//
//	REQ-1: Foo (line 17 -> Inner, line 12 -> Foo) and Bar (line 7): functions 3, findings 1 + 0 + 2 = 3,
//	       worst file a2.go 5.0 yellow
//	REQ-2: line 100 of b.go is in no function: no row
//	REQ-3: only in a test file: no row
//
// TRLC-LINKS: REQ-EMG-152
func TestBuildRollupsRequirementsJoinSymbolsToFunctions(t *testing.T) {
	links := []SymbolLink{
		{Path: "a1.go", Line: 17, Requirements: []string{"REQ-1"}},
		{Path: "a1.go", Line: 18, Requirements: []string{"REQ-1"}},
		{Path: "a1.go", Line: 12, Requirements: []string{"REQ-1"}},
		{Path: "a2.go", Line: 7, Requirements: []string{"REQ-1", "REQ-4"}},
		{Path: "b.go", Line: 100, Requirements: []string{"REQ-2"}},
		{Path: "a_test.go", Line: 10, Requirements: []string{"REQ-3"}},
		{Path: "missing.go", Line: 1, Requirements: []string{"REQ-5"}},
	}
	got := BuildRollups(rollupFiles(), nil, links).Requirements
	want := []RequirementRollup{
		{Requirement: "REQ-1", Functions: 3, Findings: 3, Score: 5.0, Band: BandYellow, WorstFile: "a2.go"},
		{Requirement: "REQ-4", Functions: 1, Findings: 2, Score: 5.0, Band: BandYellow, WorstFile: "a2.go"},
	}
	wantDeepEqual(t, got, want)
}

// A file without an owner counts for the project only, an empty project has
// score 10.0 green with Files 0 and no worst file, and the tests block is
// absent without test files.
//
// TRLC-LINKS: REQ-EMG-152
func TestBuildRollupsEdgeCases(t *testing.T) {
	r := BuildRollups([]FileReport{{Path: "x.go", LOC: 0, Score: 7.0, Band: BandYellow}}, nil, nil)
	wantEqual(t, [4]any{len(r.Units), r.Project.Files, r.Project.Score, r.Tests != nil}, [4]any{0, 1, 7.0, false}) // an owner-less file counts for the project only
	empty := BuildRollups(nil, nil, nil)
	wantEqual(t, [4]any{empty.Project.Files, empty.Project.Score, empty.Project.Band, empty.Project.Worst != nil}, [4]any{0, 10.0, BandGreen, false})
	wantEqual(t, [2]bool{empty.Units != nil, empty.Requirements != nil}, [2]bool{true, true}) // empty lists, not nil
}

// The innermost function wins, and a line outside every function matches none.
//
// TRLC-LINKS: REQ-EMG-152
func TestInnermostFunction(t *testing.T) {
	fns := []Function{{StartLine: 1, EndLine: 50}, {StartLine: 10, EndLine: 20}}
	for line, want := range map[int]int{5: 0, 10: 1, 20: 1, 21: 0, 50: 0, 51: -1, 0: -1} {
		wantEqual(t, [2]int{line, innermostFunction(fns, line)}, [2]int{line, want})
	}
}

// WorstFunctions ranks production functions by deducted points (weight x
// severity factor), ties by path and line, and leaves test files out.
//
// TRLC-LINKS: REQ-EMG-152
func TestWorstFunctionsRanksByPoints(t *testing.T) {
	r := Report{Files: rollupFiles()}
	// Bar: complex_method high 2.0 + large_method warn 0.8 = 2.8; Foo: bumpy_road warn 1.0.
	got := WorstFunctions(r, 10)
	mustEqual(t, len(got), 2)
	wantEqual(t, [4]any{got[0].Function.Name, got[0].Points, got[1].Function.Name, got[1].Points}, [4]any{"Bar", 2.8, "Foo", 1.0})
	one := WorstFunctions(r, 1)
	mustEqual(t, len(one), 1)
	wantEqual(t, one[0].Function.Name, "Bar")
}
