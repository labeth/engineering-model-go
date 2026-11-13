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
	if r.Schema != "engmod-health/1" || r.Engine.ScoringVersion != ScoringVersion || r.Engine.Weights[CodeComplexMethod] != 1.0 {
		t.Fatalf("header = %+v %+v", r.Schema, r.Engine)
	}
	var paths []string
	for _, f := range r.Files {
		paths = append(paths, f.Path)
	}
	if strings.Join(paths, ",") != "pkg/a.go,pkg/b.go,pkg/z_test.go" {
		t.Fatalf("files not sorted by path: %v", paths)
	}
	b := r.Files[1]
	if b.Score != 8.0 || b.Band != BandGreen || len(b.Deductions) != 1 || b.Deductions[0].Code != CodeComplexMethod {
		t.Errorf("pkg/b.go = %v %s %+v, want 8.0 green with only complex_method", b.Score, b.Band, b.Deductions)
	}
	if len(r.Waived) != 1 || r.Waived[0].Decision != "ADR-OK" || r.Waived[0].Code != CodeExcessArguments {
		t.Errorf("waived = %+v", r.Waived)
	}
	if len(r.Diagnostics) != 1 || r.Diagnostics[0].Code != model.HealthWaiverUnusedCode || r.Diagnostics[0].Level != LevelError {
		t.Errorf("diagnostics = %+v, want one unused waiver at level error", r.Diagnostics)
	}
	if want := (Summary{Files: 3, Findings: 2, Waived: 1, Errors: 3, Warnings: 0}); r.Summary != want {
		t.Errorf("summary = %+v, want %+v", r.Summary, want)
	}
	if r.Rollups.Project.Files != 2 || r.Rollups.Tests == nil || len(r.Rollups.Units) != 1 || r.Rollups.Units[0].Unit != "FU-A" ||
		len(r.Rollups.Requirements) != 1 || r.Rollups.Requirements[0].Requirement != "REQ-1" {
		t.Errorf("rollups = %+v", r.Rollups)
	}
}

// At a tier where the debt row is a warning the same findings count as warnings;
// a report without levels counts everything as a warning.
//
// TRLC-LINKS: REQ-EMG-155
func TestBuildReportWarningsBelowProduct(t *testing.T) {
	opts := sampleOptions()
	opts.Tier, opts.Levels = "mvp", Levels(bundleAt("mvp", nil, nil))
	if s := BuildReport(sampleResults(), opts).Summary; s.Errors != 0 || s.Warnings != 3 {
		t.Errorf("mvp summary = %+v, want 0 errors and 3 warnings", s)
	}
	opts.Levels = nil
	if s := BuildReport(sampleResults(), opts).Summary; s.Errors != 0 {
		t.Errorf("no levels: %+v", s)
	}
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
	if err != nil {
		t.Fatal(err)
	}
	reversed := sampleResults()
	reversed[0], reversed[2] = reversed[2], reversed[0]
	for i := 0; i < 5; i++ {
		again, err := Encode(BuildReport(reversed, sampleOptions()))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(first, again) {
			t.Fatal("two reports of the same results differ")
		}
	}
	text := string(first)
	if !strings.HasSuffix(text, "}\n") || !strings.HasPrefix(text, "{\n  \"schema\": \"engmod-health/1\",\n") {
		t.Errorf("not canonical JSON: %.80q ... %q", text, text[len(text)-4:])
	}
	for _, banned := range []string{"timestamp", "generatedAt", "\"time\"", "/home/", "/tmp/", "\\u003c"} {
		if strings.Contains(text, banned) {
			t.Errorf("report contains %q", banned)
		}
	}
	if !strings.Contains(text, `"waived": [`) || !strings.Contains(text, `"diagnostics": [`) {
		t.Error("waived and diagnostics must be lists, never absent")
	}
	if empty, _ := Encode(BuildReport(nil, Options{})); !strings.Contains(string(empty), `"files": []`) {
		t.Errorf("an empty report must hold an empty files list: %s", empty)
	}
}

// LevelOf treats a missing code as a warning.
//
// TRLC-LINKS: REQ-EMG-155
func TestLevelOf(t *testing.T) {
	r := Report{Levels: map[string]string{CodeBumpyRoad: LevelError}}
	if r.LevelOf(CodeBumpyRoad) != LevelError || r.LevelOf(CodeLargeMethod) != LevelWarning {
		t.Error("LevelOf is wrong")
	}
}

// Relevel recomputes the tier, bundles, levels, diagnostic levels and summary
// and leaves scores and findings alone.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-157
func TestRelevelRecomputesLevelsOnly(t *testing.T) {
	opts := sampleOptions()
	opts.Tier, opts.Levels = "mvp", Levels(bundleAt("mvp", nil, nil))
	mvp := BuildReport(sampleResults(), opts)
	if mvp.Summary.Errors != 0 {
		t.Fatalf("mvp errors = %d", mvp.Summary.Errors)
	}
	op := Relevel(mvp, bundleAt("operational", []string{"safety"}, nil))
	if op.Tier != "operational" || op.Summary.Errors != 3 || op.Diagnostics[0].Level != LevelError || len(op.Bundles) != 2 {
		t.Errorf("operational = %+v %+v", op.Tier, op.Summary)
	}
	if op.Files[1].Score != mvp.Files[1].Score || mvp.Diagnostics[0].Level != LevelWarning {
		t.Error("Relevel changed the scores or the original report")
	}
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
	if len(got) != 2 || got[0].Path != "b.go" || got[1].Path != "c.go" || got[0].Level != LevelError ||
		!strings.Contains(got[0].Message, "4 syntax errors") || !strings.Contains(got[1].Message, "at least") {
		t.Fatalf("diagnostics = %+v", got)
	}
	if r.Files[1].SyntaxErrors != 4 || r.Files[0].SyntaxErrors != 0 || r.Summary.Errors != 2 {
		t.Errorf("files %+v summary %+v", r.Files, r.Summary)
	}
	data, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateReportJSON(data); err != nil {
		t.Errorf("the report does not validate: %v", err)
	}
	if !strings.Contains(string(data), `"syntaxErrors": 4`) {
		t.Errorf("the count is not in the JSON:\n%s", data)
	}
}
