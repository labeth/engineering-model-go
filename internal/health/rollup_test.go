// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"reflect"
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

	if len(r.Units) != 2 || r.Units[0].Unit != "FU-A" || r.Units[1].Unit != "FU-B" {
		t.Fatalf("units = %+v", r.Units)
	}
	a := r.Units[0]
	if a.Score != 6.0 || a.Band != BandYellow || a.Files != 2 || a.LOC != 400 || a.Worst.Path != "a2.go" || a.Worst.Score != 5.0 {
		t.Errorf("FU-A = %+v worst %+v", a, a.Worst)
	}
	if b := r.Units[1]; b.Score != 3.0 || b.Band != BandRed || b.Worst.Path != "b.go" {
		t.Errorf("FU-B = %+v", b)
	}
	p := r.Project
	if p.Score != 6.1 || p.Band != BandYellow || p.Files != 4 || p.LOC != 500 || p.Worst.Path != "b.go" ||
		p.Bands != (BandCounts{Green: 2, Yellow: 1, Red: 1}) {
		t.Errorf("project = %+v worst %+v", p, p.Worst)
	}
	if r.Tests == nil || r.Tests.Score != 6.0 || r.Tests.Files != 1 || r.Tests.Worst.Path != "a_test.go" {
		t.Errorf("tests = %+v", r.Tests)
	}
	if len(r.Requirements) != 0 {
		t.Errorf("no links, but requirements = %+v", r.Requirements)
	}
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
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("requirements = %+v, want %+v", got, want)
	}
}

// A file without an owner counts for the project only, an empty project has
// score 10.0 green with Files 0 and no worst file, and the tests block is
// absent without test files.
//
// TRLC-LINKS: REQ-EMG-152
func TestBuildRollupsEdgeCases(t *testing.T) {
	r := BuildRollups([]FileReport{{Path: "x.go", LOC: 0, Score: 7.0, Band: BandYellow}}, nil, nil)
	if len(r.Units) != 0 || r.Project.Files != 1 || r.Project.Score != 7.0 || r.Tests != nil {
		t.Errorf("owner-less file: %+v", r)
	}
	empty := BuildRollups(nil, nil, nil)
	if empty.Project.Files != 0 || empty.Project.Score != 10.0 || empty.Project.Band != BandGreen || empty.Project.Worst != nil {
		t.Errorf("empty = %+v", empty.Project)
	}
	if empty.Units == nil || empty.Requirements == nil {
		t.Error("units and requirements must be empty lists, not nil")
	}
}

// The innermost function wins, and a line outside every function matches none.
//
// TRLC-LINKS: REQ-EMG-152
func TestInnermostFunction(t *testing.T) {
	fns := []Function{{StartLine: 1, EndLine: 50}, {StartLine: 10, EndLine: 20}}
	for line, want := range map[int]int{5: 0, 10: 1, 20: 1, 21: 0, 50: 0, 51: -1, 0: -1} {
		if got := innermostFunction(fns, line); got != want {
			t.Errorf("line %d -> %d, want %d", line, got, want)
		}
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
	if len(got) != 2 || got[0].Function.Name != "Bar" || got[0].Points != 2.8 || got[1].Function.Name != "Foo" || got[1].Points != 1.0 {
		t.Fatalf("got %+v", got)
	}
	if one := WorstFunctions(r, 1); len(one) != 1 || one[0].Function.Name != "Bar" {
		t.Errorf("limit 1: %+v", one)
	}
}
