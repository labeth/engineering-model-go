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
	return entryAt(baselineKey{path, function, code}, 1000)
}

// entryAt is a baseline entry that recorded a value.
func entryAt(k baselineKey, value int) model.HealthBaselineEntry {
	return model.HealthBaselineEntry{Path: k.path, Function: k.function, Code: k.code, Value: value}
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
	mustEqual(t, len(baselined), 1)
	wantEqual(t, [4]any{baselined[0].Path, baselined[0].Function, baselined[0].Code, baselined[0].Value}, [4]any{"internal/a.go", "Parser.parse", CodeComplexMethod, 25})
	wantStale := []model.HealthBaselineEntry{entry("internal/a.go", "Parser.parse", CodeBumpyRoad), entry("internal/zzz.go", "gone", CodeLargeMethod)}
	wantDeepEqual(t, stale, wantStale)
	wantDeepEqual(t, in, waiverResults()) // the input is not modified
	b, s := ApplyBaseline(in, nil)
	wantEqual(t, [2]bool{b != nil, s != nil}, [2]bool{false, false}) // an empty baseline marks nothing
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

	wantDeepEqual(t, based.Files, plain.Files)
	wantDeepEqual(t, based.Rollups, plain.Rollups)
	mustEqual(t, len(based.Baselined), 1)
	wantEqual(t, based.Baselined[0].Code, CodeComplexMethod)
	wantEqual(t, [2]bool{based.IsBaselined("pkg/b.go", "Big", CodeComplexMethod), based.IsBaselined("pkg/b.go", "Big", CodeExcessArguments)}, [2]bool{true, false})
	wantEqual(t, [3]int{plain.Summary.Baselined, based.Summary.Baselined, based.Summary.Findings}, [3]int{0, 1, plain.Summary.Findings})
	wantEqual(t, based.Summary.Errors, plain.Summary.Errors-1) // one baselined error
	total := based.Summary.Errors + based.Summary.Warnings + based.Summary.Baselined
	wantEqual(t, total, based.Summary.Findings+len(based.Diagnostics))
	wantEqual(t, len(based.Diagnostics), len(plain.Diagnostics)) // a matching entry adds no diagnostic
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
		found := diagnosticsOfCode(r, model.HealthBaselineStaleCode)
		mustEqual(t, [2]any{tier, len(found)}, [2]any{tier, 1})
		wantEqual(t, [4]string{tier, found[0].Level, found[0].Path, found[0].Function}, [4]string{tier, level, "pkg/b.go", "Gone"})
		wantContains(t, found[0].Message, CodeLargeMethod)
		wantEqual(t, [2]string{tier, r.Levels[model.HealthBaselineStaleCode]}, [2]string{tier, level})
	}
}

// diagnosticsOfCode lists the diagnostics of a report with a code.
func diagnosticsOfCode(r Report, code string) []Diagnostic {
	var found []Diagnostic
	for _, d := range r.Diagnostics {
		if d.Code == code {
			found = append(found, d)
		}
	}
	return found
}

// A manifest without a baseline gives the report it gave before the feature: an
// empty baseline is the same report, byte for byte, and the encoding has no
// baselined key and no summary count; a report with a baseline lists them.
//
// TRLC-LINKS: REQ-EMG-160, REQ-EMG-157
func TestNoBaselineLeavesTheReportUnchanged(t *testing.T) {
	plain, err := Encode(BuildReport(sampleResults(), baselineOptions("product")))
	mustEqual(t, err, nil)
	empty, err := Encode(BuildReport(sampleResults(), baselineOptions("product", []model.HealthBaselineEntry{}...)))
	mustEqual(t, err, nil)
	mustEqual(t, bytes.Equal(plain, empty), true) // an empty baseline changes nothing
	wantNone(t, string(plain), "baselined")
	based, err := Encode(BuildReport(sampleResults(), baselineOptions("product", entry("pkg/b.go", "Big", CodeComplexMethod))))
	mustEqual(t, err, nil)
	wantAll(t, string(based), `"baselined": [`, `"baselined": 1`) // a baselined report lists and counts them
}

// The schema describes the baselined list: a report with one validates, imports
// and re-encodes byte-identical, keeps the count after a re-level and rejects an
// unclean path.
//
// TRLC-LINKS: REQ-EMG-157
func TestBaselinedReportConformsToTheSchemaAndImports(t *testing.T) {
	r := BuildReport(sampleResults(), baselineOptions("product", entry("pkg/b.go", "Big", CodeComplexMethod), entry("pkg/b.go", "Gone", CodeLargeMethod)))
	data, err := Encode(r)
	mustEqual(t, err, nil)
	mustEqual(t, ValidateReportJSON(data), nil)
	back, err := Import(bytes.NewReader(data))
	mustEqual(t, err, nil)
	again, err := Encode(back)
	mustEqual(t, err, nil)
	wantEqual(t, bytes.Equal(data, again), true) // import round-trips
	relevel := Relevel(back, bundleAt("operational", nil, nil))
	wantEqual(t, [2]int{relevel.Summary.Baselined, len(relevel.Baselined)}, [2]int{1, 1})
	bad := bytes.Replace(data, []byte(`"path": "pkg/b.go",
      "function": "Big"`), []byte(`"path": "../b.go",
      "function": "Big"`), 1)
	_, err = Import(bytes.NewReader(bad))
	wantEqual(t, err != nil, true) // an unclean baselined path is refused
}

// SARIF marks a baselined finding as externally suppressed, so an upload does
// not show accepted debt as an open error; the others keep no suppression.
//
// TRLC-LINKS: REQ-EMG-157
func TestSARIFSuppressesBaselinedFindings(t *testing.T) {
	r := BuildReport(sampleResults(), baselineOptions("product", entry("pkg/b.go", "Big", CodeComplexMethod)))
	data, err := SARIF(r)
	mustEqual(t, err, nil)
	wantEqual(t, strings.Count(string(data), "accepted in the health baseline"), 1)
}

// BaselineEntries lists one entry per distinct finding, sorted, and honours the
// filter; BaselineDiff names the stale and the missing entries.
//
// TRLC-LINKS: REQ-EMG-153
func TestBaselineEntriesAndDiff(t *testing.T) {
	r := BuildReport(sampleResults(), baselineOptions("product"))
	all := BaselineEntries(r, nil)
	want := []model.HealthBaselineEntry{
		entryAt(baselineKey{"pkg/b.go", "Big", CodeComplexMethod}, 25), entryAt(baselineKey{"pkg/b.go", "Big", CodeExcessArguments}, 6),
		entryAt(baselineKey{"pkg/z_test.go", "TestZ", CodeLargeAssertionBlock}, 7),
	}
	mustEqual(t, reflect.DeepEqual(all, want), true)
	only := BaselineEntries(r, func(code string) bool { return code == CodeComplexMethod })
	mustEqual(t, len(only), 1)
	wantEqual(t, only[0].Code, CodeComplexMethod)
	have := []model.HealthBaselineEntry{entryAt(baselineKey{"pkg/b.go", "Big", CodeComplexMethod}, 25), entry("pkg/old.go", "Old", CodeLargeMethod)}
	stale, missing := BaselineDiff(have, all)
	wantDeepEqual(t, stale, []model.HealthBaselineEntry{entry("pkg/old.go", "Old", CodeLargeMethod)})
	wantEqual(t, len(missing), 2)
	// An entry whose recorded value differs from the current one is both stale
	// and missing, so baseline write --check fails until it is rewritten.
	stale, missing = BaselineDiff([]model.HealthBaselineEntry{entryAt(baselineKey{"pkg/b.go", "Big", CodeComplexMethod}, 30)}, only)
	mustEqual(t, [2]int{len(stale), len(missing)}, [2]int{1, 1})
	wantEqual(t, [2]int{stale[0].Value, missing[0].Value}, [2]int{30, 25})
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
		requireRecordedValue(t, c.recorded, c.baselined)
	}
	grown := BuildReport(sampleResults(), baselineOptions("product", entryAt(baselineKey{"pkg/b.go", "Big", CodeComplexMethod}, 24)))
	wantEqual(t, grown.Files[1].Functions[0].Findings[0].Message, "cyclomatic 25 in Big (the baseline accepts cyclomatic 24, now 25: debt that grows is not baselined)")
	wantEqual(t, sampleResults()[1].Functions[0].Findings[0].Message, "cyclomatic 25 in Big") // the input is not modified
}

// requireRecordedValue checks the report of a baseline entry with a recorded
// value: whether the finding is baselined, the errors, no stale entry and the
// message of a grown finding.
func requireRecordedValue(t *testing.T, recorded int, baselined bool) {
	t.Helper()
	r := BuildReport(sampleResults(), baselineOptions("product", entryAt(baselineKey{"pkg/b.go", "Big", CodeComplexMethod}, recorded)))
	wantEqual(t, [2]any{recorded, r.IsBaselined("pkg/b.go", "Big", CodeComplexMethod)}, [2]any{recorded, baselined})
	wantErrors := 3
	if baselined {
		wantErrors = 2
	}
	wantEqual(t, [2]int{recorded, r.Summary.Errors}, [2]int{recorded, wantErrors})
	wantEqual(t, len(diagnosticsOfCode(r, model.HealthBaselineStaleCode)), 0) // a matching entry is not stale
	message := r.Files[1].Functions[0].Findings[0].Message
	wantEqual(t, [2]any{recorded, strings.Contains(message, "the baseline accepts cyclomatic")}, [2]any{recorded, !baselined})
}
