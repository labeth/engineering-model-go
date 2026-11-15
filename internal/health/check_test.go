// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health_test

import (
	"bytes"
	"encoding/json"
	"errors"
	"reflect"
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
		wantEqual(t, [2]string{tier, regressionAt(tier, nil, nil)}, [2]string{tier, level})
	}
	wantEqual(t, regressionAt("mvp", []string{"safety"}, nil), "error")
	wantEqual(t, regressionAt("prototype", []string{"safety"}, nil), "warning")
	wantEqual(t, regressionAt("poc", nil, []string{"health.regression"}), "error") // lifecycle.raise
	wantEqual(t, health.RegressionLevelOf(model.Bundle{}), "warning")              // no tier
	levels := health.CheckLevels(health.WithTier(model.Bundle{}, "product"))
	wantEqual(t, [2]string{levels["health.regression"], levels[health.CodeBumpyRoad]}, [2]string{"error", "error"})
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
	got := health.NewCheckReport(delta, health.CheckInput{Levels: errLevels})
	wantEqual(t, got.Counts, health.CheckCounts{New: 2, Worsened: 1, Existing: 1})
	wantEqual(t, [3]any{got.Failed, got.RegressionLevel, got.Waived != nil}, [3]any{true, "error", true})
	warn := health.NewCheckReport(delta, health.CheckInput{Levels: map[string]string{"health.regression": "warning"}})
	wantEqual(t, [2]any{warn.Failed, warn.RegressionLevel}, [2]any{false, "warning"}) // a warning does not fail
	missing := health.NewCheckReport(delta, health.CheckInput{})
	wantEqual(t, [2]bool{missing.Failed, missing.Levels != nil}, [2]bool{false, true}) // a missing level is a warning
	clean := health.DeltaReport{Entries: []health.DeltaEntry{entry(health.ChangeExisting, health.SeverityHigh, "Old")}, Blocking: []health.DeltaEntry{}}
	c := health.NewCheckReport(clean, health.CheckInput{Levels: errLevels})
	wantEqual(t, [2]any{c.Failed, c.Counts.Existing}, [2]any{false, 1}) // existing debt does not fail
}

// Transition reads "(none) -> ..." for a new finding and the old and new
// metric and severity for a worsened one.
//
// TRLC-LINKS: REQ-EMG-156
func TestTransitionText(t *testing.T) {
	wantEqual(t, entry(health.ChangeNew, health.SeverityHigh, "F").Transition(), "(none) -> cyclomatic 21 high")
	wantEqual(t, entry(health.ChangeWorsened, health.SeverityHigh, "F").Transition(), "cyclomatic 12 warn -> cyclomatic 21 high")
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
	rep := health.NewCheckReport(delta, health.CheckInput{Levels: health.CheckLevels(health.WithTier(model.Bundle{}, "product"))})
	data, err := health.EncodeCheck(rep)
	mustEqual(t, err, nil)
	again, _ := health.EncodeCheck(rep)
	wantEqual(t, bytes.Equal(data, again), true)
	var doc map[string]any
	mustEqual(t, json.Unmarshal(data, &doc), nil)
	for _, key := range []string{"schema", "tier", "base", "head", "notes", "files", "entries", "blocking", "levels", "regressionLevel", "counts", "waived", "failed"} {
		_, ok := doc[key]
		wantEqual(t, [2]any{key, ok}, [2]any{key, true})
	}
	wantEqual(t, [3]any{doc["schema"], doc["failed"], doc["regressionLevel"]}, [3]any{health.DeltaSchema, true, "error"})
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
	data, err := health.CheckSARIF(health.NewCheckReport(delta, health.CheckInput{Levels: map[string]string{"health.regression": "error"}}))
	mustEqual(t, err, nil)
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
	mustEqual(t, json.Unmarshal(data, &doc), nil)
	mustEqual(t, len(doc.Runs), 1)
	run := doc.Runs[0]
	mustEqual(t, [2]int{len(run.Tool.Driver.Rules), len(run.Results)}, [2]int{1, 1})
	wantEqual(t, run.Tool.Driver.Rules[0].ID, "health.regression")
	res := run.Results[0]
	wantEqual(t, [4]any{res.RuleID, res.Level, res.Properties["code"], res.PartialFingerprints[health.FingerprintKey]},
		[4]any{"health.regression", "error", health.CodeComplexMethod, health.FindingFingerprint(health.CodeComplexMethod, "pkg/a.go", "Big")})
	empty, _ := health.CheckSARIF(health.NewCheckReport(health.DeltaReport{Blocking: []health.DeltaEntry{}}, health.CheckInput{}))
	wantAll(t, string(empty), `"results": []`, `"level": "warning"`) // an empty list, not null
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
	wantEqual(t, [2]string{e.Name(), e.Version()}, [2]string{"fake", "7"})
	for i := 0; i < 2; i++ {
		res, err := e.Analyze([]health.FileInput{{Path: "pkg/a.go", Language: "go"}, {Path: "pkg/b.go", Language: "go"}}, health.Config{})
		mustEqual(t, [2]any{err, len(res)}, [2]any{nil, 2})
		got := res[0].Functions[0].Findings // only large_method
		mustEqual(t, len(got), 1)
		wantEqual(t, got[0].Code, health.CodeLargeMethod)
		wantEqual(t, len(res[1].Functions[0].Findings), 2) // no waiver for that path
	}
	waived := e.Waived()
	mustEqual(t, len(waived), 1) // recorded once
	wantEqual(t, [3]string{waived[0].Path, waived[0].Code, waived[0].Decision}, [3]string{"pkg/a.go", health.CodeComplexMethod, "ADR-X"})
	none := &health.WaivingEngine{Engine: inner}
	res, _ := none.Analyze([]health.FileInput{{Path: "pkg/a.go", Language: "go"}}, health.Config{})
	wantEqual(t, len(res[0].Functions[0].Findings), 2) // an engine without waivers passes the results through
	wantDeepEqual(t, none.Waived(), []health.WaivedFinding{})
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

	r := health.NewCheckReport(skipped, health.CheckInput{Levels: errLevels})
	wantEqual(t, [2]any{r.Failed, len(r.Skipped)}, [2]any{true, 1}) // a skipped file where regression is an error fails
	r = health.NewCheckReport(skipped, health.CheckInput{Levels: warnLevels})
	wantEqual(t, r.Failed, false)
	r = health.NewCheckReport(clean, health.CheckInput{Levels: errLevels})
	wantEqual(t, [3]bool{r.Failed, r.Skipped != nil, r.Policy != nil}, [3]bool{false, true, true})
	requirePolicyVerdicts(t, clean, warnLevels)
}

// requirePolicyVerdicts checks the failure and the carried verdict for each
// way a policy weakening can be recorded.
func requirePolicyVerdicts(t *testing.T, clean health.DeltaReport, levels map[string]string) {
	t.Helper()
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
		r := health.NewCheckReport(clean, health.CheckInput{Levels: levels, Policy: c.verdict})
		wantEqual(t, [2]any{c.name, r.Failed}, [2]any{c.name, c.failed})
		wantEqual(t, [3]any{c.name, r.PolicyDecision, len(r.Policy)}, [3]any{c.name, c.verdict.Decision, len(c.verdict.Changes)})
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
	rules, results := read(health.NewCheckReport(delta, health.CheckInput{Levels: levels, Policy: health.PolicyVerdict{Changes: weak, Level: "error"}}))
	if !reflect.DeepEqual(rules, []string{"health.regression", "health.file_skipped", "health.policy_weakened"}) ||
		!reflect.DeepEqual(results, []string{"health.file_skipped:error", "health.policy_weakened:error"}) {
		t.Errorf("rules %v results %v", rules, results)
	}
	rules, results = read(health.NewCheckReport(delta, health.CheckInput{Levels: levels, Policy: health.PolicyVerdict{Changes: weak, Level: "error", Decision: "ADR-POLICY"}}))
	if !reflect.DeepEqual(rules, []string{"health.regression", "health.file_skipped"}) || !reflect.DeepEqual(results, []string{"health.file_skipped:error"}) {
		t.Errorf("a recorded weakening: rules %v results %v", rules, results)
	}
	rules, results = read(health.NewCheckReport(health.DeltaReport{}, health.CheckInput{Levels: levels}))
	if !reflect.DeepEqual(rules, []string{"health.regression"}) || len(results) != 0 {
		t.Errorf("a clean check: rules %v results %v", rules, results)
	}
}
