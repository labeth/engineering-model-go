// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"fmt"
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// floorRow is one function of a floorFile: the name, the code of its one
// finding and whether the finding is high (else warn). A warn complex_method
// deducts 1.0 and a high one 2.0.
type floorRow struct {
	name string
	code string
	high bool
}

// floorFile is a file result with one function per row, 20 lines apart from
// line 10.
func floorFile(path string, isTest bool, rows ...floorRow) FileResult {
	res := FileResult{Path: path, Language: "go", LOC: 100, IsTest: isTest}
	for i, r := range rows {
		line := 10 + 20*i
		severity := SeverityWarn
		if r.high {
			severity = SeverityHigh
		}
		finding := Finding{Code: r.code, Severity: severity, StartLine: line, EndLine: line + 15, Metric: "cyclomatic", Value: 20}
		res.Functions = append(res.Functions, Function{Name: r.name, Kind: "function", StartLine: line, EndLine: line + 15, Findings: []Finding{finding}})
	}
	return res
}

// warnFile and highFile are one-function files with a complex_method finding.
func warnFile(path string) FileResult {
	return floorFile(path, false, floorRow{name: "Warn", code: CodeComplexMethod})
}

func highFile(path string) FileResult {
	return floorFile(path, false, floorRow{name: "High", code: CodeComplexMethod, high: true})
}

// floorCase is a small set of files, a baseline, waivers and a floor at tier
// operational, with the paths of the files that are reported.
type floorCase struct {
	name     string
	results  []FileResult
	baseline []model.HealthBaselineEntry
	waivers  []model.HealthWaiver
	floor    float64
	want     []string
}

// options are the report options of the case at tier operational.
func (c floorCase) options() Options {
	return Options{EngineName: "treesitter", EngineVersion: "1", Tier: "operational", Levels: Levels(bundleAt("operational", nil, nil)), Floor: c.floor, Baseline: c.baseline, Waivers: c.waivers}
}

// floorDiagnosticPaths lists the paths of the health.file_below_floor
// diagnostics of a report, in report order.
func floorDiagnosticPaths(r Report) []string {
	out := []string{}
	for _, d := range r.Diagnostics {
		if d.Code == CodeFileBelowFloor {
			out = append(out, d.Path)
		}
	}
	return out
}

// floorDiagnostic returns the one health.file_below_floor diagnostic of a
// report, or false when there is none or several.
func floorDiagnostic(r Report) (Diagnostic, bool) {
	var found []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Code == CodeFileBelowFloor {
			found = append(found, d)
		}
	}
	if len(found) != 1 {
		return Diagnostic{}, false
	}
	return found[0], true
}

// coverCases are the cases of the baseline cover.
func coverCases() []floorCase {
	entryFor := func(path, function, code string) []model.HealthBaselineEntry {
		return []model.HealthBaselineEntry{entry(path, function, code)}
	}
	return []floorCase{
		{name: "a file at 10.0 passes", results: []FileResult{floorFile("pkg/clean.go", false)}, floor: 10.0, want: []string{}},
		{name: "below the floor and uncovered fails", results: []FileResult{warnFile("pkg/warn.go")}, floor: 10.0, want: []string{"pkg/warn.go"}},
		{name: "a file of the baseline is covered", results: []FileResult{warnFile("pkg/warn.go")}, baseline: entryFor("pkg/warn.go", "Warn", CodeComplexMethod), floor: 10.0, want: []string{}},
		{name: "an entry the finding outgrew still covers the file", results: []FileResult{warnFile("pkg/warn.go")},
			baseline: []model.HealthBaselineEntry{entryAt(baselineKey{"pkg/warn.go", "Warn", CodeComplexMethod}, 3)}, floor: 10.0, want: []string{}},
		{name: "a stale entry covers nothing", results: []FileResult{warnFile("pkg/warn.go")}, baseline: entryFor("pkg/warn.go", "Gone", CodeComplexMethod), floor: 10.0, want: []string{"pkg/warn.go"}},
		{name: "an entry of another file covers nothing", results: []FileResult{warnFile("pkg/warn.go")}, baseline: entryFor("pkg/other.go", "Warn", CodeComplexMethod), floor: 10.0, want: []string{"pkg/warn.go"}},
		{name: "an entry of another code covers nothing", results: []FileResult{warnFile("pkg/warn.go")}, baseline: entryFor("pkg/warn.go", "Warn", CodeLargeMethod), floor: 10.0, want: []string{"pkg/warn.go"}},
		{name: "a new file is not covered by the entries of the others", results: []FileResult{highFile("pkg/high.go"), warnFile("pkg/warn.go")},
			baseline: entryFor("pkg/high.go", "High", CodeComplexMethod), floor: 10.0, want: []string{"pkg/warn.go"}},
		{name: "an empty baseline covers nothing", results: []FileResult{highFile("pkg/high.go"), warnFile("pkg/warn.go")}, baseline: []model.HealthBaselineEntry{}, floor: 10.0, want: []string{"pkg/high.go", "pkg/warn.go"}},
		{name: "a test file counts like any file", results: []FileResult{floorFile("pkg/x_test.go", true, floorRow{name: "TestX", code: CodeComplexMethod})}, floor: 10.0, want: []string{"pkg/x_test.go"}},
	}
}

// waiverAndFloorCases are the cases of the waivers and of the value of the
// floor.
func waiverAndFloorCases() []floorCase {
	two := floorFile("pkg/two.go", false, floorRow{name: "One", code: CodeComplexMethod}, floorRow{name: "Two", code: CodeComplexMethod})
	waive := func(path, function string) []model.HealthWaiver {
		return []model.HealthWaiver{{Code: CodeComplexMethod, Path: path, Function: function, Decision: "ADR-OK"}}
	}
	return []floorCase{
		{name: "a waiver of every finding leaves the file at 10.0", results: []FileResult{warnFile("pkg/warn.go")}, waivers: waive("pkg/warn.go", "Warn"), floor: 10.0, want: []string{}},
		{name: "a waiver of some findings leaves the file below the floor", results: []FileResult{two}, waivers: waive("pkg/two.go", "One"), floor: 10.0, want: []string{"pkg/two.go"}},
		{name: "a score equal to the floor passes", results: []FileResult{warnFile("pkg/warn.go")}, floor: 9.0, want: []string{}},
		{name: "a floor above the score fails", results: []FileResult{warnFile("pkg/warn.go")}, floor: 9.1, want: []string{"pkg/warn.go"}},
		{name: "a lowered floor lets only the high file fail", results: []FileResult{warnFile("pkg/warn.go"), highFile("pkg/high.go")}, floor: 8.5, want: []string{"pkg/high.go"}},
		{name: "no floor asks for nothing", results: []FileResult{warnFile("pkg/warn.go"), highFile("pkg/high.go")}, floor: 0, want: []string{}},
	}
}

// health.file_below_floor is reported for every file whose score is below the
// floor and that no baseline entry covers (ADR-EMG-087). Each case is a small
// set of files, a baseline, waivers and a floor, and says which files are
// reported.
//
// Mutation: skipping the baseline cover, covering by path alone, letting a
// stale entry cover, counting a waived finding, leaving test files out or
// comparing with <= instead of < fails the matching row.
//
// TRLC-LINKS: REQ-EMG-209
func TestFileBelowFloorIsReportedUnlessCovered(t *testing.T) {
	for _, c := range append(coverCases(), waiverAndFloorCases()...) {
		t.Run(c.name, func(t *testing.T) {
			if got := floorDiagnosticPaths(BuildReport(c.results, c.options())); !reflect.DeepEqual(got, c.want) {
				t.Errorf("health.file_below_floor at %v, want %v", got, c.want)
			}
		})
	}
}

// levelCase is a tier (with bundles and raise) and the level of the code there.
type levelCase struct {
	name  string
	b     model.Bundle
	level string
}

// levelCases are the five tiers, no tier, the safety bundle and a raise.
func levelCases() []levelCase {
	return []levelCase{
		{"no tier", model.Bundle{}, LevelWarning},
		{"poc", bundleAt("poc", nil, nil), LevelWarning},
		{"prototype", bundleAt("prototype", nil, nil), LevelWarning},
		{"mvp", bundleAt("mvp", nil, nil), LevelWarning},
		{"product", bundleAt("product", nil, nil), LevelError},
		{"operational", bundleAt("operational", nil, nil), LevelError},
		{"mvp with the safety bundle", bundleAt("mvp", []string{"safety"}, nil), LevelError},
		{"prototype with the safety bundle", bundleAt("prototype", []string{"safety"}, nil), LevelWarning},
		{"poc raised", bundleAt("poc", nil, []string{"health.*"}), LevelError},
	}
}

// The level of the diagnostic is the level of the code at the tier: a warning at
// poc, prototype and mvp, an error at product and operational, an error from
// mvp with the safety bundle and with lifecycle.raise, a warning without a tier.
// The summary counts the finding and the file at that level.
//
// Mutation: a level that ignores the tier, the bundle or the raise, or a
// diagnostic that is left out of the summary, fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestFileBelowFloorLevelFollowsTheTier(t *testing.T) {
	for _, c := range levelCases() {
		t.Run(c.name, func(t *testing.T) {
			r := BuildReport([]FileResult{warnFile("pkg/warn.go")}, Options{Floor: 10.0, Levels: Levels(c.b)})
			d, ok := floorDiagnostic(r)
			if !ok || d.Level != c.level {
				t.Fatalf("diagnostics = %+v, want one at %s", r.Diagnostics, c.level)
			}
			wantErrors := map[string]int{LevelError: 2, LevelWarning: 0}[c.level]
			if r.Summary.Errors != wantErrors || r.Summary.Errors+r.Summary.Warnings != 2 {
				t.Errorf("summary = %+v, want %d errors of 2", r.Summary, wantErrors)
			}
		})
	}
}

// requireInOrder fails for each part that the text lacks or that stands before
// the one named earlier.
func requireInOrder(t *testing.T, text string, parts ...string) {
	t.Helper()
	last := -1
	for _, part := range parts {
		at := strings.Index(text, part)
		if at <= last {
			t.Errorf("%q is missing or out of order in:\n%s", part, text)
		}
		last = max(last, at)
	}
}

// manyCauses is a file with five deductions: Worst (2.0), Second (1.0 x 0.5),
// Mid (0.8 x 0.25), Tail (0.6 x 0.25) and Small (0.5 x 0.25).
func manyCauses() FileResult {
	return floorFile("pkg/many.go", false,
		floorRow{name: "Small", code: CodeExcessArguments},
		floorRow{name: "Mid", code: CodeLargeMethod},
		floorRow{name: "Worst", code: CodeComplexMethod, high: true},
		floorRow{name: "Second", code: CodeDeepNesting},
		floorRow{name: "Tail", code: CodeComplexConditional},
	)
}

// The message names the file, its score, the floor, the three largest
// deductions with function and line, the number of the others and the fix, the
// largest first.
//
// Mutation: naming all deductions, naming the first three in source order,
// leaving the line or the floor out, or dropping the fix fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestFileBelowFloorMessageNamesTheCauses(t *testing.T) {
	r := BuildReport([]FileResult{manyCauses()}, floorCase{floor: 10.0}.options())
	d, ok := floorDiagnostic(r)
	if !ok {
		t.Fatalf("diagnostics = %+v", r.Diagnostics)
	}
	requireInOrder(t, d.Message, fmt.Sprintf("pkg/many.go scores %.1f, below the floor of 10.0, and no baseline entry covers it", r.Files[0].Score),
		"health.complex_method in Worst (line 50), -2.00", "health.deep_nesting in Second (line 70), -0.50",
		"health.large_method in Mid (line 30), -0.20", "and 2 more", "fix: raise the file to 10.0")
	if strings.Contains(d.Message, "Small") || strings.Contains(d.Message, "Tail") {
		t.Errorf("message names a deduction below the three largest:\n%s", d.Message)
	}
}

// burnDownResults are three files below the floor and one at 10.0.
func burnDownResults() []FileResult {
	return []FileResult{warnFile("pkg/a.go"), warnFile("pkg/b.go"), warnFile("pkg/c.go"), floorFile("pkg/d.go", false)}
}

// burnDownBaseline covers a.go and b.go.
func burnDownBaseline() []model.HealthBaselineEntry {
	return []model.HealthBaselineEntry{entry("pkg/a.go", "Warn", CodeComplexMethod), entry("pkg/b.go", "Warn", CodeComplexMethod)}
}

// The summary counts the files below the floor and the ones the baseline
// covers, so the number left to burn down is visible; the report names its
// floor; Relevel keeps both counts.
//
// Mutation: counting only the uncovered files, or counting the files at 10.0,
// fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestFloorSummaryCountsTheBurnDown(t *testing.T) {
	c := floorCase{floor: 10.0, baseline: burnDownBaseline()}
	r := BuildReport(burnDownResults(), c.options())
	if r.Floor != 10.0 || r.Summary.BelowFloor != 3 || r.Summary.FloorCovered != 2 {
		t.Errorf("floor %v summary %+v, want 3 below and 2 covered", r.Floor, r.Summary)
	}
	if got := floorDiagnosticPaths(r); !reflect.DeepEqual(got, []string{"pkg/c.go"}) {
		t.Errorf("reported %v, want c.go", got)
	}
	if re := Relevel(r, bundleAt("mvp", nil, nil)); re.Summary.BelowFloor != 3 || re.Summary.FloorCovered != 2 {
		t.Errorf("relevel at mvp: %+v", re.Summary)
	}
}

// A report made without a floor has no floor, no count and no diagnostic.
//
// Mutation: a default floor in BuildReport fails here and in every older report
// test.
//
// TRLC-LINKS: REQ-EMG-209
func TestReportWithoutAFloorHasNone(t *testing.T) {
	c := floorCase{floor: 0, baseline: burnDownBaseline()}
	r := BuildReport(burnDownResults(), c.options())
	if r.Floor != 0 || r.Summary.BelowFloor != 0 || r.Summary.FloorCovered != 0 {
		t.Errorf("floor %v summary %+v, want none", r.Floor, r.Summary)
	}
	if got := floorDiagnosticPaths(r); len(got) != 0 {
		t.Errorf("reported %v without a floor", got)
	}
}

// BaselineCovered returns exactly the files with a finding that an entry names;
// the input is untouched and an empty baseline covers nothing.
//
// Mutation: covering by path alone or by function name alone fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestBaselineCoveredNamesFilesWithAMatchedFinding(t *testing.T) {
	in := waiverResults()
	got := BaselineCovered(in, []model.HealthBaselineEntry{
		entry("internal/a.go", "other", CodeComplexMethod),
		entry("internal/b.go", "Parser.parse", CodeLargeMethod),
		entry("internal/zzz.go", "gone", CodeComplexMethod),
	})
	if want := map[string]bool{"internal/a.go": true}; !reflect.DeepEqual(got, want) {
		t.Errorf("covered = %v, want %v", got, want)
	}
	if got := BaselineCovered(in, nil); len(got) != 0 {
		t.Errorf("an empty baseline covered %v", got)
	}
	if !reflect.DeepEqual(in, waiverResults()) {
		t.Error("BaselineCovered modified its input")
	}
}

// The same tree gives the same report: the order of the input files and of the
// baseline entries does not matter, the encoding is identical and it validates
// against the schema.
//
// Mutation: iterating a map to build the diagnostics, or taking the order of
// the input, fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestFloorReportIsDeterministic(t *testing.T) {
	results := []FileResult{warnFile("pkg/c.go"), highFile("pkg/a.go"), warnFile("pkg/b.go")}
	forward := floorCase{floor: 10.0, baseline: []model.HealthBaselineEntry{entry("pkg/a.go", "High", CodeComplexMethod), entry("pkg/b.go", "Warn", CodeComplexMethod)}}
	backward := floorCase{floor: 10.0, baseline: []model.HealthBaselineEntry{forward.baseline[1], forward.baseline[0]}}
	first, err := Encode(BuildReport(results, forward.options()))
	if err != nil {
		t.Fatal(err)
	}
	second, err := Encode(BuildReport([]FileResult{results[2], results[1], results[0]}, backward.options()))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(first, second) {
		t.Errorf("the reports differ:\n%s\n%s", first, second)
	}
	if err := ValidateReportJSON(first); err != nil {
		t.Errorf("the report with a floor does not validate: %v", err)
	}
}
