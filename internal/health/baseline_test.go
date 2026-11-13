// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// entry is a baseline entry that accepts any value up to 1000.
func entry(path, function, code string) model.HealthBaselineEntry {
	return entryAt(path, function, code, 1000)
}

// entryAt is a baseline entry that recorded a value.
func entryAt(path, function, code string, value int) model.HealthBaselineEntry {
	return model.HealthBaselineEntry{Path: path, Function: function, Code: code, Value: value}
}

// baselineOptions are the sample options without waivers and with the given
// baseline, at the given tier.
func baselineOptions(tier string, entries ...model.HealthBaselineEntry) Options {
	o := sampleOptions()
	o.Tier, o.Levels, o.Waivers, o.Baseline = tier, Levels(bundleAt(tier, nil, nil)), nil, entries
	return o
}

// ApplyBaseline marks exactly the findings whose path, qualified function name
// and code are an entry, in report order, and returns the entries that match
// nothing in canonical order. The same function name in another file, another
// code of the same function and the input are untouched.
//
// Mutation: matching on the function name alone, or on the code alone, or
// dropping the finding from the results, fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
func TestApplyBaselineMatchesPathFunctionAndCode(t *testing.T) {
	in := waiverResults()
	entries := []model.HealthBaselineEntry{
		entry("internal/a.go", "Parser.parse", CodeComplexMethod),
		entry("internal/zzz.go", "gone", CodeLargeMethod),
		entry("internal/a.go", "Parser.parse", CodeBumpyRoad),
	}
	baselined, stale := ApplyBaseline(in, entries)
	if len(baselined) != 1 || baselined[0].Path != "internal/a.go" || baselined[0].Function != "Parser.parse" ||
		baselined[0].Code != CodeComplexMethod || baselined[0].Value != 25 {
		t.Fatalf("baselined = %+v", baselined)
	}
	wantStale := []model.HealthBaselineEntry{entry("internal/a.go", "Parser.parse", CodeBumpyRoad), entry("internal/zzz.go", "gone", CodeLargeMethod)}
	if !reflect.DeepEqual(stale, wantStale) {
		t.Errorf("stale = %+v, want %+v", stale, wantStale)
	}
	if !reflect.DeepEqual(in, waiverResults()) {
		t.Error("ApplyBaseline modified its input")
	}
	if b, s := ApplyBaseline(in, nil); b != nil || s != nil {
		t.Errorf("an empty baseline marked %+v / %+v", b, s)
	}
}

// A baselined finding stays in the report and counts toward the score, but it
// is neither an error nor a warning of the summary: the report lists it under
// baselined, the score and the deductions equal those of a scan without the
// baseline, Findings counts it, and Errors drops by exactly the baselined
// errors.
//
// pkg/b.go: Big has complex_method high and excess_arguments warn; 2.0 + 0.5 =
// 2.5 -> 7.5 either way. At product the debt codes are errors: without the
// baseline 3 errors (the two of Big and the test's large_assertion_block);
// baselining complex_method leaves 2.
//
// Mutation: removing the finding from the function before scoring (as a
// waiver does) changes the score and fails.
//
// TRLC-LINKS: REQ-EMG-151, REQ-EMG-155
func TestBaselinedFindingsCountTowardTheScoreButNotTheErrors(t *testing.T) {
	plain := BuildReport(sampleResults(), baselineOptions("product"))
	based := BuildReport(sampleResults(), baselineOptions("product", entry("pkg/b.go", "Big", CodeComplexMethod)))

	if !reflect.DeepEqual(plain.Files, based.Files) || !reflect.DeepEqual(plain.Rollups, based.Rollups) {
		t.Fatal("a baseline changed the files, the scores or the roll-ups")
	}
	if len(based.Baselined) != 1 || based.Baselined[0].Code != CodeComplexMethod || !based.IsBaselined("pkg/b.go", "Big", CodeComplexMethod) ||
		based.IsBaselined("pkg/b.go", "Big", CodeExcessArguments) {
		t.Fatalf("baselined = %+v", based.Baselined)
	}
	if plain.Summary.Baselined != 0 || based.Summary.Baselined != 1 || based.Summary.Findings != plain.Summary.Findings {
		t.Errorf("summaries: plain %+v baselined %+v", plain.Summary, based.Summary)
	}
	if based.Summary.Errors != plain.Summary.Errors-1 {
		t.Errorf("errors = %d, want %d (one baselined error)", based.Summary.Errors, plain.Summary.Errors-1)
	}
	if got := based.Summary.Errors + based.Summary.Warnings + based.Summary.Baselined; got != based.Summary.Findings+len(based.Diagnostics) {
		t.Errorf("errors + warnings + baselined = %d, want findings %d + diagnostics %d", got, based.Summary.Findings, len(based.Diagnostics))
	}
	if len(based.Diagnostics) != len(plain.Diagnostics) {
		t.Errorf("an entry that matched must not add a diagnostic: %+v", based.Diagnostics)
	}
}

// An entry that matches no finding is health.baseline_stale, at the level of
// the tier row WWWWE: a warning at poc, prototype, mvp and product and an error
// at operational, where it counts in the errors of the summary; a matching
// entry raises none.
//
// Mutation: a different row, or no diagnostic, fails.
//
// TRLC-LINKS: REQ-EMG-155
func TestStaleBaselineEntryFollowsTheTierRow(t *testing.T) {
	want := map[string]string{"poc": LevelWarning, "prototype": LevelWarning, "mvp": LevelWarning, "product": LevelWarning, "operational": LevelError}
	stale := entry("pkg/b.go", "Gone", CodeLargeMethod)
	for tier, level := range want {
		o := baselineOptions(tier, entry("pkg/b.go", "Big", CodeComplexMethod), stale)
		r := BuildReport(sampleResults(), o)
		var found []Diagnostic
		for _, d := range r.Diagnostics {
			if d.Code == model.HealthBaselineStaleCode {
				found = append(found, d)
			}
		}
		if len(found) != 1 || found[0].Level != level || found[0].Path != "pkg/b.go" || found[0].Function != "Gone" ||
			!strings.Contains(found[0].Message, CodeLargeMethod) {
			t.Errorf("%s: stale diagnostics = %+v, want one %s", tier, found, level)
		}
		if r.Levels[model.HealthBaselineStaleCode] != level {
			t.Errorf("%s: levels[baseline_stale] = %q", tier, r.Levels[model.HealthBaselineStaleCode])
		}
	}
}

// A manifest without a baseline gives the report it gave before the feature: an
// empty baseline is the same report, byte for byte, and the encoding has no
// baselined key and no summary count; a report with a baseline lists them.
//
// TRLC-LINKS: REQ-EMG-160, REQ-EMG-157
func TestNoBaselineLeavesTheReportUnchanged(t *testing.T) {
	plain, err := Encode(BuildReport(sampleResults(), baselineOptions("product")))
	if err != nil {
		t.Fatal(err)
	}
	empty, err := Encode(BuildReport(sampleResults(), baselineOptions("product", []model.HealthBaselineEntry{}...)))
	if err != nil || !bytes.Equal(plain, empty) {
		t.Fatal("an empty baseline changed the report")
	}
	if bytes.Contains(plain, []byte("baselined")) {
		t.Errorf("a report without baseline must not mention baselined:\n%s", plain)
	}
	based, err := Encode(BuildReport(sampleResults(), baselineOptions("product", entry("pkg/b.go", "Big", CodeComplexMethod))))
	if err != nil || !bytes.Contains(based, []byte(`"baselined": [`)) || !bytes.Contains(based, []byte(`"baselined": 1`)) {
		t.Fatalf("a baselined report must list and count them: %v", err)
	}
}

// The schema describes the baselined list: a report with one validates, imports
// and re-encodes byte-identical, keeps the count after a re-level and rejects an
// unclean path.
//
// TRLC-LINKS: REQ-EMG-157
func TestBaselinedReportConformsToTheSchemaAndImports(t *testing.T) {
	r := BuildReport(sampleResults(), baselineOptions("product", entry("pkg/b.go", "Big", CodeComplexMethod), entry("pkg/b.go", "Gone", CodeLargeMethod)))
	data, err := Encode(r)
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateReportJSON(data); err != nil {
		t.Fatalf("schema: %v", err)
	}
	back, err := Import(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	again, err := Encode(back)
	if err != nil || !bytes.Equal(data, again) {
		t.Fatalf("import does not round-trip: %v", err)
	}
	if relevel := Relevel(back, bundleAt("operational", nil, nil)); relevel.Summary.Baselined != 1 || len(relevel.Baselined) != 1 {
		t.Errorf("relevel lost the baselined findings: %+v", relevel.Summary)
	}
	bad := bytes.Replace(data, []byte(`"path": "pkg/b.go",
      "function": "Big"`), []byte(`"path": "../b.go",
      "function": "Big"`), 1)
	if _, err := Import(bytes.NewReader(bad)); err == nil {
		t.Error("an unclean baselined path was imported")
	}
}

// SARIF marks a baselined finding as externally suppressed, so an upload does
// not show accepted debt as an open error; the others keep no suppression.
//
// TRLC-LINKS: REQ-EMG-157
func TestSARIFSuppressesBaselinedFindings(t *testing.T) {
	r := BuildReport(sampleResults(), baselineOptions("product", entry("pkg/b.go", "Big", CodeComplexMethod)))
	data, err := SARIF(r)
	if err != nil {
		t.Fatal(err)
	}
	if n := strings.Count(string(data), "accepted in the health baseline"); n != 1 {
		t.Fatalf("suppressions = %d, want 1:\n%s", n, data)
	}
}

// BaselineEntries lists one entry per distinct finding, sorted, and honours the
// filter; BaselineDiff names the stale and the missing entries.
//
// TRLC-LINKS: REQ-EMG-153
func TestBaselineEntriesAndDiff(t *testing.T) {
	r := BuildReport(sampleResults(), baselineOptions("product"))
	all := BaselineEntries(r, nil)
	want := []model.HealthBaselineEntry{
		entryAt("pkg/b.go", "Big", CodeComplexMethod, 25), entryAt("pkg/b.go", "Big", CodeExcessArguments, 6),
		entryAt("pkg/z_test.go", "TestZ", CodeLargeAssertionBlock, 7),
	}
	if !reflect.DeepEqual(all, want) {
		t.Fatalf("entries = %+v, want %+v", all, want)
	}
	only := BaselineEntries(r, func(code string) bool { return code == CodeComplexMethod })
	if len(only) != 1 || only[0].Code != CodeComplexMethod {
		t.Errorf("filtered = %+v", only)
	}
	have := []model.HealthBaselineEntry{entryAt("pkg/b.go", "Big", CodeComplexMethod, 25), entry("pkg/old.go", "Old", CodeLargeMethod)}
	stale, missing := BaselineDiff(have, all)
	if !reflect.DeepEqual(stale, []model.HealthBaselineEntry{entry("pkg/old.go", "Old", CodeLargeMethod)}) || len(missing) != 2 {
		t.Errorf("stale %+v missing %+v", stale, missing)
	}
	// An entry whose recorded value differs from the current one is both stale
	// and missing, so baseline write --check fails until it is rewritten.
	stale, missing = BaselineDiff([]model.HealthBaselineEntry{entryAt("pkg/b.go", "Big", CodeComplexMethod, 30)}, only)
	if len(stale) != 1 || stale[0].Value != 30 || len(missing) != 1 || missing[0].Value != 25 {
		t.Errorf("a changed value: stale %+v missing %+v", stale, missing)
	}
}

// The baseline records the metric value of each finding and accepts it only
// while the finding is at most that value: equal or lower is baselined; a
// finding that grew is not baselined, counts at the level of its code (an
// error at product), says in its message what the baseline accepted, and its
// entry is not stale. Small increases cannot ratchet in silently.
//
// pkg/b.go Big has complex_method 25 (high) and excess_arguments 6 (warn).
//
// Mutation: comparing the entry without its value (baselining by key alone), or
// using >= instead of > for the comparison, fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
func TestBaselinedFindingMustNotGrowPastItsRecordedValue(t *testing.T) {
	cases := []struct {
		recorded  int
		baselined bool
	}{{26, true}, {25, true}, {24, false}, {1, false}}
	for _, c := range cases {
		r := BuildReport(sampleResults(), baselineOptions("product", entryAt("pkg/b.go", "Big", CodeComplexMethod, c.recorded)))
		if got := r.IsBaselined("pkg/b.go", "Big", CodeComplexMethod); got != c.baselined {
			t.Errorf("recorded %d: baselined = %v, want %v", c.recorded, got, c.baselined)
		}
		wantErrors := 3
		if c.baselined {
			wantErrors = 2
		}
		if r.Summary.Errors != wantErrors {
			t.Errorf("recorded %d: errors = %d, want %d", c.recorded, r.Summary.Errors, wantErrors)
		}
		for _, d := range r.Diagnostics {
			if d.Code == model.HealthBaselineStaleCode {
				t.Errorf("recorded %d: an entry that matched a finding must not be stale: %+v", c.recorded, d)
			}
		}
		message := r.Files[1].Functions[0].Findings[0].Message
		if grew := strings.Contains(message, "the baseline accepts cyclomatic"); grew == c.baselined {
			t.Errorf("recorded %d: message %q", c.recorded, message)
		}
	}
	grown := BuildReport(sampleResults(), baselineOptions("product", entryAt("pkg/b.go", "Big", CodeComplexMethod, 24)))
	if want := "cyclomatic 25 in Big (the baseline accepts cyclomatic 24, now 25: debt that grows is not baselined)"; grown.Files[1].Functions[0].Findings[0].Message != want {
		t.Errorf("message = %q, want %q", grown.Files[1].Functions[0].Findings[0].Message, want)
	}
	if sampleResults()[1].Functions[0].Findings[0].Message != "cyclomatic 25 in Big" {
		t.Error("annotating a grown finding modified the input results")
	}
}
