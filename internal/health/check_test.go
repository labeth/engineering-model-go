// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/model"
)

// regressionAt returns the level of health.regression for a tier, with the
// safety bundle and the raised codes of the lifecycle.
//
// TRLC-LINKS: REQ-EMG-155
func regressionAt(tier string, bundles, raise []string) string {
	var b model.Bundle
	b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier, Bundles: bundles, Raise: raise}
	return health.RegressionLevelOf(b)
}

// The level of health.regression comes from the tier policy: warning, warning,
// warning, error, error at poc to operational (row WWWEE), an error from mvp
// with the safety bundle (WWEEE), an error where lifecycle.raise names it, and
// a warning for a model without a tier. CheckLevels carries the level next to
// the levels of the smell codes.
//
// Mutation: a table of its own in place of the tier policy, or ignoring the
// bundle or the raise list, fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func TestRegressionLevelFollowsTheTierPolicy(t *testing.T) {
	want := map[string]string{"poc": "warning", "prototype": "warning", "mvp": "warning", "product": "error", "operational": "error"}
	for tier, level := range want {
		if got := regressionAt(tier, nil, nil); got != level {
			t.Errorf("%s: %s, want %s", tier, got, level)
		}
	}
	if got := regressionAt("mvp", []string{"safety"}, nil); got != "error" {
		t.Errorf("mvp with the safety bundle: %s, want error", got)
	}
	if got := regressionAt("prototype", []string{"safety"}, nil); got != "warning" {
		t.Errorf("prototype with the safety bundle: %s, want warning", got)
	}
	if got := regressionAt("poc", nil, []string{"health.regression"}); got != "error" {
		t.Errorf("poc with lifecycle.raise: %s, want error", got)
	}
	if got := health.RegressionLevelOf(model.Bundle{}); got != "warning" {
		t.Errorf("no tier: %s, want warning", got)
	}
	levels := health.CheckLevels(health.WithTier(model.Bundle{}, "product"))
	if levels["health.regression"] != "error" || levels[health.CodeBumpyRoad] != "error" {
		t.Errorf("levels = %v", levels)
	}
}

// entry builds a delta entry.
//
// TRLC-LINKS: REQ-EMG-156
func entry(change string, sev health.Severity, name string) health.DeltaEntry {
	e := health.DeltaEntry{Path: "pkg/a.go", Function: name, Code: health.CodeComplexMethod, Change: change, Severity: sev,
		Metric: "cyclomatic", ValueAfter: 21, StartLine: 7, EndLine: 40, Message: "cyclomatic 21 in " + name}
	if change != health.ChangeNew {
		e.SeverityBefore, e.ValueBefore = health.SeverityWarn, 12
	}
	return e
}

// The check report counts the entries by class, the waived ones apart, and
// fails only when there is a regression and it is an error at the tier: no
// regression passes at every tier, a regression passes as a warning and fails
// as an error.
//
// Mutation: failing on existing debt, or on a warning regression, fails.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestNewCheckReportVerdict(t *testing.T) {
	regression := entry(health.ChangeNew, health.SeverityHigh, "Big")
	delta := health.DeltaReport{
		Schema: health.DeltaSchema,
		Entries: []health.DeltaEntry{
			regression, entry(health.ChangeWorsened, health.SeverityHigh, "Worse"),
			entry(health.ChangeNew, health.SeverityWarn, "Mild"), entry(health.ChangeExisting, health.SeverityHigh, "Old"),
		},
	}
	delta.Blocking = health.Regressions(delta)
	errLevels := map[string]string{"health.regression": "error"}
	got := health.NewCheckReport(delta, errLevels, nil, health.PolicyVerdict{})
	if got.Counts != (health.CheckCounts{New: 2, Worsened: 1, Existing: 1}) || !got.Failed || got.RegressionLevel != "error" || got.Waived == nil {
		t.Errorf("error level: %+v", got)
	}
	if warn := health.NewCheckReport(delta, map[string]string{"health.regression": "warning"}, nil, health.PolicyVerdict{}); warn.Failed || warn.RegressionLevel != "warning" {
		t.Errorf("warning level must not fail: %+v", warn)
	}
	if missing := health.NewCheckReport(delta, nil, nil, health.PolicyVerdict{}); missing.Failed || missing.Levels == nil {
		t.Errorf("a missing level is a warning: %+v", missing)
	}
	clean := health.DeltaReport{Entries: []health.DeltaEntry{entry(health.ChangeExisting, health.SeverityHigh, "Old")}, Blocking: []health.DeltaEntry{}}
	if c := health.NewCheckReport(clean, errLevels, nil, health.PolicyVerdict{}); c.Failed || c.Counts.Existing != 1 {
		t.Errorf("existing debt must not fail: %+v", c)
	}
}

// Transition reads "(none) -> ..." for a new finding and the old and new
// metric and severity for a worsened one.
//
// TRLC-LINKS: REQ-EMG-156
func TestTransitionText(t *testing.T) {
	if got := entry(health.ChangeNew, health.SeverityHigh, "F").Transition(); got != "(none) -> cyclomatic 21 high" {
		t.Errorf("new: %q", got)
	}
	if got := entry(health.ChangeWorsened, health.SeverityHigh, "F").Transition(); got != "cyclomatic 12 warn -> cyclomatic 21 high" {
		t.Errorf("worsened: %q", got)
	}
}

// The JSON of a check is the delta report object with the levels, the counts,
// the waived findings and the verdict next to it, and is byte-identical for the
// same input.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158, REQ-EMG-159
func TestEncodeCheckExtendsTheDeltaReport(t *testing.T) {
	delta := health.DeltaReport{Schema: health.DeltaSchema, Tier: "product", Base: "abc", Head: health.WorkingTree,
		Notes: []string{}, Files: []health.FileDelta{}, Entries: []health.DeltaEntry{entry(health.ChangeNew, health.SeverityHigh, "Big")}}
	delta.Blocking = health.Regressions(delta)
	rep := health.NewCheckReport(delta, health.CheckLevels(health.WithTier(model.Bundle{}, "product")), nil, health.PolicyVerdict{})
	data, err := health.EncodeCheck(rep)
	if err != nil {
		t.Fatal(err)
	}
	again, _ := health.EncodeCheck(rep)
	if !bytes.Equal(data, again) {
		t.Error("two encodings differ")
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"schema", "tier", "base", "head", "notes", "files", "entries", "blocking", "levels", "regressionLevel", "counts", "waived", "failed"} {
		if _, ok := doc[key]; !ok {
			t.Errorf("the JSON lacks %q:\n%s", key, data)
		}
	}
	if doc["schema"] != health.DeltaSchema || doc["failed"] != true || doc["regressionLevel"] != "error" {
		t.Errorf("doc = %v", doc)
	}
}

// The SARIF of a check has one rule, health.regression, and one result per
// regression with the level of the tier and the smell code in the properties;
// the fingerprint is that of the smell finding in the scan output. A check
// without regressions has an empty result list, not null.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func TestCheckSARIF(t *testing.T) {
	delta := health.DeltaReport{Entries: []health.DeltaEntry{entry(health.ChangeNew, health.SeverityHigh, "Big"), entry(health.ChangeExisting, health.SeverityHigh, "Old")}}
	delta.Blocking = health.Regressions(delta)
	data, err := health.CheckSARIF(health.NewCheckReport(delta, map[string]string{"health.regression": "error"}, nil, health.PolicyVerdict{}))
	if err != nil {
		t.Fatal(err)
	}
	var doc struct {
		Runs []struct {
			Tool struct {
				Driver struct{ Rules []struct{ ID string } }
			}
			Results []struct {
				RuleID              string
				Level               string
				PartialFingerprints map[string]string
				Properties          map[string]any
			}
		}
	}
	if err := json.Unmarshal(data, &doc); err != nil || len(doc.Runs) != 1 {
		t.Fatalf("%v\n%s", err, data)
	}
	run := doc.Runs[0]
	if len(run.Tool.Driver.Rules) != 1 || run.Tool.Driver.Rules[0].ID != "health.regression" || len(run.Results) != 1 {
		t.Fatalf("sarif = %s", data)
	}
	res := run.Results[0]
	if res.RuleID != "health.regression" || res.Level != "error" || res.Properties["code"] != health.CodeComplexMethod ||
		res.PartialFingerprints[health.FingerprintKey] != health.FindingFingerprint(health.CodeComplexMethod, "pkg/a.go", "Big") {
		t.Errorf("result = %+v", res)
	}
	empty, _ := health.CheckSARIF(health.NewCheckReport(health.DeltaReport{Blocking: []health.DeltaEntry{}}, nil, nil, health.PolicyVerdict{}))
	if !strings.Contains(string(empty), `"results": []`) || !strings.Contains(string(empty), `"level": "warning"`) {
		t.Errorf("empty sarif = %s", empty)
	}
}

// fakeEngine returns one function with findings of the given codes.
type fakeEngine struct{ calls int }

// TRLC-LINKS: REQ-EMG-154
func (f *fakeEngine) Name() string { return "fake" }

// TRLC-LINKS: REQ-EMG-154
func (f *fakeEngine) Version() string { return "7" }

// TRLC-LINKS: REQ-EMG-154
func (f *fakeEngine) Analyze(inputs []health.FileInput, _ health.Config) ([]health.FileResult, error) {
	f.calls++
	out := make([]health.FileResult, len(inputs))
	for i, in := range inputs {
		out[i] = health.FileResult{Path: in.Path, Language: in.Language, Functions: []health.Function{{
			Name: "Big", StartLine: 3, EndLine: 9,
			Findings: []health.Finding{
				{Code: health.CodeComplexMethod, Severity: health.SeverityHigh, StartLine: 3, EndLine: 9, Metric: "cyclomatic", Value: 21},
				{Code: health.CodeLargeMethod, Severity: health.SeverityWarn, StartLine: 3, EndLine: 9, Metric: "loc", Value: 80},
			},
		}}}
	}
	return out, nil
}

// The waiving engine removes exactly the waived findings of the function, on
// every call, delegates its name and version, records each waived finding once
// whatever the number of calls, and leaves the inner results alone.
//
// Mutation: waiving by code only, or recording a finding twice, fails.
//
// TRLC-LINKS: REQ-EMG-154, REQ-EMG-156
func TestWaivingEngineRemovesAndRecordsWaivedFindings(t *testing.T) {
	inner := &fakeEngine{}
	e := &health.WaivingEngine{Engine: inner, Waivers: []model.HealthWaiver{
		{Code: health.CodeComplexMethod, Path: "pkg/a.go", Function: "Big", Decision: "ADR-X"},
		{Code: health.CodeLargeMethod, Path: "pkg/a.go", Function: "Other", Decision: "ADR-X"},
	}}
	if e.Name() != "fake" || e.Version() != "7" {
		t.Errorf("identity = %s %s", e.Name(), e.Version())
	}
	for i := 0; i < 2; i++ {
		res, err := e.Analyze([]health.FileInput{{Path: "pkg/a.go", Language: "go"}, {Path: "pkg/b.go", Language: "go"}}, health.Config{})
		if err != nil || len(res) != 2 {
			t.Fatal(err, res)
		}
		if got := res[0].Functions[0].Findings; len(got) != 1 || got[0].Code != health.CodeLargeMethod {
			t.Errorf("a.go findings = %+v, want only large_method", got)
		}
		if got := res[1].Functions[0].Findings; len(got) != 2 {
			t.Errorf("b.go findings = %+v, want both (no waiver for that path)", got)
		}
	}
	waived := e.Waived()
	if len(waived) != 1 || waived[0].Path != "pkg/a.go" || waived[0].Code != health.CodeComplexMethod || waived[0].Decision != "ADR-X" {
		t.Errorf("waived = %+v, want one finding recorded once", waived)
	}
	none := &health.WaivingEngine{Engine: inner}
	res, _ := none.Analyze([]health.FileInput{{Path: "pkg/a.go", Language: "go"}}, health.Config{})
	if len(res[0].Functions[0].Findings) != 2 || len(none.Waived()) != 0 || !reflect.DeepEqual(none.Waived(), []health.WaivedFinding{}) {
		t.Error("an engine without waivers must pass the results through")
	}
}

// RequireWorkTree accepts a git work tree, also one without commits, and names
// a directory outside any repository as ErrNotGitRepo.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func TestRequireWorkTree(t *testing.T) {
	r := health.NewFixtureRepo(t)
	if err := health.RequireWorkTree(r.Repo()); err != nil {
		t.Errorf("a new repository: %v", err)
	}
	err := health.RequireWorkTree(health.NewGitRepo(t.TempDir()))
	if !errors.Is(err, health.ErrNotGitRepo) {
		t.Errorf("a plain directory: %v, want ErrNotGitRepo", err)
	}
}

// A changed file that was skipped fails the report where a regression does
// (health.regression is an error), and only warns below that; a weakening of
// the policy fails where health.policy_weakened is an error unless a decision
// records it. A clean report with neither passes.
//
// Mutation: ignoring Skipped or the policy verdict in NewCheckReport, or
// letting a recorded decision not excuse the weakening, fails.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func TestCheckReportFailsOnSkippedFilesAndUnrecordedPolicyWeakening(t *testing.T) {
	clean := health.DeltaReport{Entries: []health.DeltaEntry{}, Blocking: []health.DeltaEntry{}}
	skipped := clean
	skipped.Skipped = []health.SkippedFile{{Path: "gen/big.go", Reason: "larger than 4194304 bytes"}}
	errLevels := map[string]string{"health.regression": "error"}
	warnLevels := map[string]string{"health.regression": "warning"}

	if r := health.NewCheckReport(skipped, errLevels, nil, health.PolicyVerdict{}); !r.Failed || len(r.Skipped) != 1 {
		t.Errorf("a skipped file where regression is an error must fail: %+v", r)
	}
	if r := health.NewCheckReport(skipped, warnLevels, nil, health.PolicyVerdict{}); r.Failed {
		t.Errorf("a skipped file where regression is a warning must not fail: %+v", r)
	}
	if r := health.NewCheckReport(clean, errLevels, nil, health.PolicyVerdict{}); r.Failed || r.Skipped == nil || r.Policy == nil {
		t.Errorf("a clean report: %+v", r)
	}

	weak := []health.PolicyChange{{Aspect: health.PolicyThreshold, Message: "the limit of health.complex_method is raised"}}
	cases := []struct {
		name    string
		verdict health.PolicyVerdict
		failed  bool
	}{
		{"error level, unrecorded", health.PolicyVerdict{Changes: weak, Level: "error"}, true},
		{"error level, recorded by a decision", health.PolicyVerdict{Changes: weak, Level: "error", Decision: "ADR-POLICY"}, false},
		{"warning level", health.PolicyVerdict{Changes: weak, Level: "warning"}, false},
		{"no change", health.PolicyVerdict{Level: "error"}, false},
	}
	for _, c := range cases {
		r := health.NewCheckReport(clean, warnLevels, nil, c.verdict)
		if r.Failed != c.failed {
			t.Errorf("%s: failed = %v, want %v (%+v)", c.name, r.Failed, c.failed, r)
		}
		if r.PolicyDecision != c.verdict.Decision || len(r.Policy) != len(c.verdict.Changes) {
			t.Errorf("%s: the report must carry the verdict: %+v", c.name, r)
		}
	}
}

// Levels carry file_skipped and file_failed at the level of health.regression,
// computed through the same tier policy: warning up to mvp, error from product
// and from mvp with the safety bundle, error where lifecycle.raise names
// health.regression, and the levels agree with RegressionLevelOf.
//
// Mutation: a fixed warning, or a separate table, fails.
//
// TRLC-LINKS: REQ-EMG-155
func TestFileSkippedAndFailedHaveTheLevelOfRegression(t *testing.T) {
	for _, c := range []struct {
		tier           string
		bundles, raise []string
		want           string
	}{
		{"poc", nil, nil, "warning"}, {"mvp", nil, nil, "warning"}, {"product", nil, nil, "error"}, {"operational", nil, nil, "error"},
		{"mvp", []string{"safety"}, nil, "error"}, {"poc", nil, []string{"health.regression"}, "error"}, {"", nil, nil, "warning"},
	} {
		var b model.Bundle
		if c.tier != "" {
			b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: c.tier, Bundles: c.bundles, Raise: c.raise}
		}
		levels := health.Levels(b)
		if levels[health.CodeFileSkipped] != c.want || levels[health.CodeFileFailed] != c.want || levels[health.CodeFileSkipped] != health.RegressionLevelOf(b) {
			t.Errorf("tier %q: skipped %q failed %q, want %s", c.tier, levels[health.CodeFileSkipped], levels[health.CodeFileFailed], c.want)
		}
	}
}

// The SARIF of a check also carries the skipped files and the unrecorded policy
// weakenings as results, with their own rules added only when there are some,
// so an upload shows what failed the gate. A weakening that a decision records
// is not a result.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-157
func TestCheckSARIFCarriesSkippedFilesAndPolicyWeakenings(t *testing.T) {
	delta := health.DeltaReport{Blocking: []health.DeltaEntry{}, Skipped: []health.SkippedFile{{Path: "gen/big.go", Reason: "larger than 4194304 bytes"}}}
	weak := []health.PolicyChange{{Aspect: "exclude", Message: `health.exclude has a new pattern "pkg/**"`}}
	levels := map[string]string{"health.regression": "error"}
	read := func(rep health.CheckReport) (rules, results []string) {
		data, err := health.CheckSARIF(rep)
		if err != nil {
			t.Fatal(err)
		}
		var doc struct {
			Runs []struct {
				Tool struct {
					Driver struct{ Rules []struct{ ID string } }
				}
				Results []struct {
					RuleID    string
					RuleIndex int
					Level     string
				}
			}
		}
		if err := json.Unmarshal(data, &doc); err != nil {
			t.Fatal(err)
		}
		for _, r := range doc.Runs[0].Tool.Driver.Rules {
			rules = append(rules, r.ID)
		}
		for _, r := range doc.Runs[0].Results {
			if doc.Runs[0].Tool.Driver.Rules[r.RuleIndex].ID != r.RuleID {
				t.Errorf("ruleIndex %d does not name %s", r.RuleIndex, r.RuleID)
			}
			results = append(results, r.RuleID+":"+r.Level)
		}
		return rules, results
	}
	rules, results := read(health.NewCheckReport(delta, levels, nil, health.PolicyVerdict{Changes: weak, Level: "error"}))
	if !reflect.DeepEqual(rules, []string{"health.regression", "health.file_skipped", "health.policy_weakened"}) ||
		!reflect.DeepEqual(results, []string{"health.file_skipped:error", "health.policy_weakened:error"}) {
		t.Errorf("rules %v results %v", rules, results)
	}
	rules, results = read(health.NewCheckReport(delta, levels, nil, health.PolicyVerdict{Changes: weak, Level: "error", Decision: "ADR-POLICY"}))
	if !reflect.DeepEqual(rules, []string{"health.regression", "health.file_skipped"}) || !reflect.DeepEqual(results, []string{"health.file_skipped:error"}) {
		t.Errorf("a recorded weakening: rules %v results %v", rules, results)
	}
	rules, results = read(health.NewCheckReport(health.DeltaReport{}, levels, nil, health.PolicyVerdict{}))
	if !reflect.DeepEqual(rules, []string{"health.regression"}) || len(results) != 0 {
		t.Errorf("a clean check: rules %v results %v", rules, results)
	}
}
