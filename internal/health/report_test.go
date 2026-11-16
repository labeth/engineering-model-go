// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// sampleResults are three files given in unsorted order: a production file
// with findings in two functions, a clean one and a test file.
func sampleResults() []FileResult {
	return []FileResult{
		{Path: "pkg/z_test.go", Language: "go", LOC: 40, IsTest: true, Functions: []Function{
			{Name: "TestZ", Kind: "function", StartLine: 3, EndLine: 30, LOC: 25, IsTest: true, Assertions: 7, Findings: []Finding{
				{Code: CodeLargeAssertionBlock, Severity: SeverityWarn, StartLine: 3, EndLine: 30, Metric: "assertions", Value: 7, Message: "7 consecutive assertions in TestZ"},
			}},
		}},
		{Path: "pkg/b.go", Language: "go", LOC: 200, Unsupported: []string{CodePrimitiveObsession}, Functions: []Function{
			{Name: "Big", Kind: "function", StartLine: 10, EndLine: 120, LOC: 100, Parameters: 6, Cyclomatic: 25, MaxNesting: 3, Bumps: 2, MaxBoolOps: 1, Findings: []Finding{
				{Code: CodeComplexMethod, Severity: SeverityHigh, StartLine: 10, EndLine: 120, Metric: "cyclomatic", Value: 25, Message: "cyclomatic 25 in Big"},
				{Code: CodeExcessArguments, Severity: SeverityWarn, StartLine: 10, EndLine: 120, Metric: "parameters", Value: 6, Message: "6 parameters in Big"},
			}},
			{Name: "T.Small", Kind: "method", StartLine: 130, EndLine: 140, LOC: 8, Cyclomatic: 2},
		}},
		{Path: "pkg/a.go", Language: "go", LOC: 50, Functions: []Function{
			{Name: "Tiny", Kind: "function", StartLine: 1, EndLine: 5, LOC: 4, Cyclomatic: 1},
		}},
	}
}

// sampleOptions are the options of the sample report: product tier, so the
// debt codes are errors.
func sampleOptions() Options {
	return Options{
		EngineName: "treesitter", EngineVersion: "1", Tier: "product", Bundles: []string{"foundation"},
		Levels: Levels(bundleAt("product", nil, nil)),
		Waivers: []model.HealthWaiver{
			{Code: CodeExcessArguments, Path: "pkg/b.go", Function: "Big", Decision: "ADR-OK"},
			{Code: CodeBumpyRoad, Path: "pkg/b.go", Function: "Big", Decision: "ADR-OLD"},
		},
		Owners: map[string]string{"pkg/a.go": "FU-A", "pkg/b.go": "FU-A"},
		Links:  []SymbolLink{{Path: "pkg/b.go", Line: 12, Requirements: []string{"REQ-1"}}},
	}
}

// BuildReport sorts the files, applies the waivers before scoring, rolls up and
// counts the findings at the level of the tier.
//
// pkg/b.go after the waiver: Big has complex_method high only, 2.0, so the
// score is 10.0 - 2.0 = 8.0 green (without the waiver: 2.0 + 0.5 = 2.5 -> 7.5).
// Findings: complex_method (error at product) and the test's
// large_assertion_block (error at product) = 2 errors, 1 waived, plus the
// unused waiver as a diagnostic that is an error at product: 3 errors.
//
// Mutation: scoring before the waiver, or leaving the unused waiver out,
// fails.
//
// TRLC-LINKS: REQ-EMG-151, REQ-EMG-152, REQ-EMG-154, REQ-EMG-155, REQ-EMG-157
func TestBuildReportScoresAfterWaiversAndCountsAtTheTier(t *testing.T) {
	r := BuildReport(sampleResults(), sampleOptions())
	wantEqual(t, [3]any{r.Schema, r.Engine.ScoringVersion, r.Engine.Weights[CodeComplexMethod]}, [3]any{"engmod-health/1", ScoringVersion, 1.0})
	var paths []string
	for _, f := range r.Files {
		paths = append(paths, f.Path)
	}
	mustEqual(t, strings.Join(paths, ","), "pkg/a.go,pkg/b.go,pkg/z_test.go") // sorted by path
	b := r.Files[1]
	wantEqual(t, [3]any{b.Score, b.Band, len(b.Deductions)}, [3]any{8.0, BandGreen, 1})
	wantEqual(t, b.Deductions[0].Code, CodeComplexMethod)
	mustEqual(t, len(r.Waived), 1)
	wantEqual(t, [2]string{r.Waived[0].Decision, r.Waived[0].Code}, [2]string{"ADR-OK", CodeExcessArguments})
	mustEqual(t, len(r.Diagnostics), 1)
	wantEqual(t, [2]string{r.Diagnostics[0].Code, r.Diagnostics[0].Level}, [2]string{model.HealthWaiverUnusedCode, LevelError}) // the unused waiver
	wantEqual(t, r.Summary, Summary{Files: 3, Findings: 2, Waived: 1, Errors: 3, Warnings: 0})
	requireSampleRollups(t, r.Rollups)
}

// requireSampleRollups checks the roll-ups of the sample report: two production
// files, the tests, one unit and one requirement.
func requireSampleRollups(t *testing.T, ro Rollups) {
	t.Helper()
	wantEqual(t, [3]any{ro.Project.Files, ro.Tests != nil, len(ro.Units)}, [3]any{2, true, 1})
	wantEqual(t, ro.Units[0].Unit, "FU-A")
	mustEqual(t, len(ro.Requirements), 1)
	wantEqual(t, ro.Requirements[0].Requirement, "REQ-1")
}

// At a tier where the debt row is a warning the same findings count as warnings;
// a report without levels counts everything as a warning.
//
// TRLC-LINKS: REQ-EMG-155
func TestBuildReportWarningsBelowProduct(t *testing.T) {
	opts := sampleOptions()
	opts.Tier, opts.Levels = "mvp", Levels(bundleAt("mvp", nil, nil))
	s := BuildReport(sampleResults(), opts).Summary
	wantEqual(t, [2]int{s.Errors, s.Warnings}, [2]int{0, 3})
	opts.Levels = nil
	wantEqual(t, BuildReport(sampleResults(), opts).Summary.Errors, 0)
}

// The encoded report is canonical and deterministic: two builds from the
// results in any order give the same bytes, with two-space indent and a
// trailing newline, and no timestamp, no absolute path and no HTML escaping.
//
// Mutation: iterating a map in the output, or adding a generated-at field,
// fails this test.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func TestEncodeIsCanonicalAndDeterministic(t *testing.T) {
	first, err := Encode(BuildReport(sampleResults(), sampleOptions()))
	mustEqual(t, err, nil)
	reversed := sampleResults()
	reversed[0], reversed[2] = reversed[2], reversed[0]
	for i := 0; i < 5; i++ {
		again, err := Encode(BuildReport(reversed, sampleOptions()))
		mustEqual(t, err, nil)
		mustEqual(t, bytes.Equal(first, again), true) // two reports of the same results are the same
	}
	text := string(first)
	wantEqual(t, [2]bool{strings.HasSuffix(text, "}\n"), strings.HasPrefix(text, "{\n  \"schema\": \"engmod-health/1\",\n")}, [2]bool{true, true})
	for _, banned := range []string{"timestamp", "generatedAt", "\"time\"", "/home/", "/tmp/", "\\u003c"} {
		wantNone(t, text, banned)
	}
	wantAll(t, text, `"waived": [`, `"diagnostics": [`) // lists, never absent
	empty, _ := Encode(BuildReport(nil, Options{}))
	wantContains(t, string(empty), `"files": []`)
}

// LevelOf treats a missing code as a warning.
//
// TRLC-LINKS: REQ-EMG-155
func TestLevelOf(t *testing.T) {
	r := Report{Levels: map[string]string{CodeBumpyRoad: LevelError}}
	wantEqual(t, [2]string{r.LevelOf(CodeBumpyRoad), r.LevelOf(CodeLargeMethod)}, [2]string{LevelError, LevelWarning})
}

// Relevel recomputes the tier, bundles, levels, diagnostic levels and summary
// and leaves scores and findings alone.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-157
func TestRelevelRecomputesLevelsOnly(t *testing.T) {
	opts := sampleOptions()
	opts.Tier, opts.Levels = "mvp", Levels(bundleAt("mvp", nil, nil))
	mvp := BuildReport(sampleResults(), opts)
	mustEqual(t, mvp.Summary.Errors, 0)
	op := Relevel(mvp, bundleAt("operational", []string{"safety"}, nil))
	wantEqual(t, [4]any{op.Tier, op.Summary.Errors, op.Diagnostics[0].Level, len(op.Bundles)}, [4]any{"operational", 3, LevelError, 2})
	// The scores and the original report are not changed.
	wantEqual(t, [2]any{op.Files[1].Score, mvp.Diagnostics[0].Level}, [2]any{mvp.Files[1].Score, LevelWarning})
}

// A file with syntax errors gets a health.syntax_error diagnostic at the level of
// the code, its count is in the file report and the JSON of the report still
// validates against the schema; a clean file gets none.
// Mutation: dropping the diagnostic lets a broken file read as clean.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155, REQ-EMG-157
func TestBuildReportFlagsSyntaxErrors(t *testing.T) {
	opts := sampleOptions()
	opts.Levels = map[string]string{CodeSyntaxError: LevelError}
	r := BuildReport([]FileResult{
		{Path: "a.go", Language: "go", LOC: 3, Functions: []Function{}},
		{Path: "b.go", Language: "go", LOC: 3, SyntaxErrors: 4, Functions: []Function{}},
		{Path: "c.go", Language: "go", LOC: 3, SyntaxErrors: SyntaxErrorCap, Functions: []Function{}},
	}, opts)
	var got []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Code == CodeSyntaxError {
			got = append(got, d)
		}
	}
	mustEqual(t, len(got), 2)
	wantEqual(t, [3]string{got[0].Path, got[1].Path, got[0].Level}, [3]string{"b.go", "c.go", LevelError})
	wantContains(t, got[0].Message, "4 syntax errors")
	wantContains(t, got[1].Message, "at least")
	wantEqual(t, [3]int{r.Files[1].SyntaxErrors, r.Files[0].SyntaxErrors, r.Summary.Errors}, [3]int{4, 0, 2})
	data, err := Encode(r)
	mustEqual(t, err, nil)
	wantEqual(t, ValidateReportJSON(data), nil) // the report validates against the schema
	wantContains(t, string(data), `"syntaxErrors": 4`)
}
