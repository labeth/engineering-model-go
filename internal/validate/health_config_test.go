// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// healthBundle is a valid model with the health block and two decisions: an
// accepted ADR-OK and a proposed ADR-DRAFT.
func healthBundle(h *model.ManifestHealth) model.Bundle {
	b := behaviorTestBundle(nil, nil)
	b.Architecture.Decisions = []model.Decision{
		{ID: "ADR-OK", Status: "accepted"},
		{ID: "ADR-DRAFT", Status: "Proposed"},
	}
	b.Manifest.Health = h
	return b
}

// healthCodes returns the diagnostics of a model whose code is one of the
// health configuration codes.
func healthCodes(b model.Bundle) []Diagnostic {
	var out []Diagnostic
	for _, d := range Bundle(b) {
		if strings.HasPrefix(d.Code, "health.") || strings.HasPrefix(d.Code, "model.invalid_health_") {
			out = append(out, d)
		}
	}
	return out
}

// floorOf is a health block with health.floor set.
func floorOf(v float64) model.ManifestHealth { return model.ManifestHealth{Floor: &v} }

// A floor from 1.0 to 10.0 is valid, the default (no floor) too, and a floor
// outside it is the core error model.invalid_health_floor, never deferred
// (ADR-EMG-087). The valid values are the bounds and one in between.
//
// Mutation: a range check that excludes 1.0 or 10.0, or one that accepts 10.1,
// fails here and in the cases of TestHealthConfigurationErrors.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-209
func TestHealthFloorRange(t *testing.T) {
	for _, v := range []float64{1.0, 5.5, 9.9, 10.0} {
		h := floorOf(v)
		if got := healthCodes(healthBundle(&h)); len(got) != 0 {
			t.Errorf("floor %v got %+v", v, got)
		}
	}
	if got := healthCodes(healthBundle(&model.ManifestHealth{})); len(got) != 0 {
		t.Errorf("no floor got %+v", got)
	}
	if !IsCoreCode("model.invalid_health_floor") {
		t.Error("model.invalid_health_floor must be core: a broken floor is never deferred")
	}
}

func okWaiver() model.HealthWaiver {
	return model.HealthWaiver{Code: "health.complex_method", Path: "internal/a.go", Function: "parse", Decision: "ADR-OK"}
}

// A model without a health block gets no health diagnostic and the same
// diagnostics as one with an empty block: the feature is opt-in and changes
// no output (REQ-EMG-160; the golden test pins the examples byte for byte).
// TRLC-LINKS: REQ-EMG-160
func TestHealthValidationIsOptIn(t *testing.T) {
	plain := healthBundle(nil)
	if got := healthCodes(plain); len(got) != 0 {
		t.Fatalf("a model without health got %+v", got)
	}
	if !reflect.DeepEqual(Bundle(plain), Bundle(healthBundle(&model.ManifestHealth{}))) {
		t.Fatal("an empty health block changed the diagnostics")
	}
	full := &model.ManifestHealth{
		Exclude:    []string{"vendor/**", "*.gen.go"},
		Thresholds: map[string]model.HealthThreshold{"health.complex_method": {Warn: 9, High: 9}},
		Overrides:  []model.HealthOverride{{Path: "internal/legacy/**", Thresholds: map[string]model.HealthThreshold{"health.large_method": {Warn: 1, High: 100}}}},
		Waivers:    []model.HealthWaiver{okWaiver()},
	}
	if got := healthCodes(healthBundle(full)); len(got) != 0 {
		t.Fatalf("a valid health block got %+v", got)
	}
	if !reflect.DeepEqual(Bundle(plain), Bundle(healthBundle(full))) {
		t.Fatal("a valid health block changed the other diagnostics")
	}
}

// Each configuration error has a negative fixture (the valid block above) and
// a positive one here that reports exactly its code at its path.
// TRLC-LINKS: REQ-EMG-153
func TestHealthConfigurationErrors(t *testing.T) {
	ok := model.HealthThreshold{Warn: 5, High: 10}
	cases := []healthCase{
		{"exclude absolute", model.ManifestHealth{Exclude: []string{"/etc"}}, "model.invalid_health_exclude", "health.exclude[0]", "is absolute"},
		{"exclude parent", model.ManifestHealth{Exclude: []string{"a", "../x"}}, "model.invalid_health_exclude", "health.exclude[1]", "leaves the module root"},
		{"exclude double star", model.ManifestHealth{Exclude: []string{"a/**/b"}}, "model.invalid_health_exclude", "health.exclude[0]", "** other than"},
		{"exclude negation glob", model.ManifestHealth{Exclude: []string{"[x"}}, "model.invalid_health_exclude", "health.exclude[0]", "not a valid glob"},
		{"threshold unknown code", model.ManifestHealth{Thresholds: map[string]model.HealthThreshold{"health.nope": ok}}, "model.invalid_health_threshold", `health.thresholds["health.nope"]`, "takes no numeric limits"},
		{"threshold structural code", model.ManifestHealth{Thresholds: map[string]model.HealthThreshold{"health.brain_method": ok}}, "model.invalid_health_threshold", `health.thresholds["health.brain_method"]`, "takes no numeric limits"},
		{"threshold warn zero", model.ManifestHealth{Thresholds: map[string]model.HealthThreshold{"health.large_method": {Warn: 0, High: 5}}}, "model.invalid_health_threshold", `health.thresholds["health.large_method"]`, "warn 0 must be at least 1"},
		{"threshold high below warn", model.ManifestHealth{Thresholds: map[string]model.HealthThreshold{"health.deep_nesting": {Warn: 6, High: 5}}}, "model.invalid_health_threshold", `health.thresholds["health.deep_nesting"]`, "high 5 must be at least warn 6"},
		{"override without path", model.ManifestHealth{Overrides: []model.HealthOverride{{}}}, "model.invalid_health_override", "health.overrides[0].path", "needs a path"},
		{"override bad path", model.ManifestHealth{Overrides: []model.HealthOverride{{Path: "../x"}}}, "model.invalid_health_override", "health.overrides[0].path", "leaves the module root"},
		{"override bad threshold", model.ManifestHealth{Overrides: []model.HealthOverride{{Path: "a", Thresholds: map[string]model.HealthThreshold{"health.bumpy_road": {Warn: 3, High: 2}}}}}, "model.invalid_health_threshold", `health.overrides[0].thresholds["health.bumpy_road"]`, "high 2 must be at least warn 3"},
		{"waiver without code", waivers(model.HealthWaiver{Path: "a.go", Function: "f", Decision: "ADR-OK"}), "model.invalid_health_waiver", "health.waivers[0].code", "needs code"},
		{"waiver without path", waivers(model.HealthWaiver{Code: "health.large_method", Function: "f", Decision: "ADR-OK"}), "model.invalid_health_waiver", "health.waivers[0].path", "needs path"},
		{"waiver without function", waivers(model.HealthWaiver{Code: "health.large_method", Path: "a.go", Decision: "ADR-OK"}), "model.invalid_health_waiver", "health.waivers[0].function", "needs function"},
		{"waiver without decision", waivers(model.HealthWaiver{Code: "health.large_method", Path: "a.go", Function: "f"}), "model.invalid_health_waiver", "health.waivers[0].decision", "needs decision"},
		{"waiver not a smell code", waivers(model.HealthWaiver{Code: "health.regression", Path: "a.go", Function: "f", Decision: "ADR-OK"}), "model.invalid_health_waiver", "health.waivers[0].code", "not a health smell code"},
		{"floor above the highest score", floorOf(10.1), "model.invalid_health_floor", "health.floor", "outside 1.0 to 10.0"},
		{"floor below the lowest score", floorOf(0.9), "model.invalid_health_floor", "health.floor", "outside 1.0 to 10.0"},
		{"floor zero", floorOf(0), "model.invalid_health_floor", "health.floor", "outside 1.0 to 10.0"},
		{"floor negative", floorOf(-4), "model.invalid_health_floor", "health.floor", "outside 1.0 to 10.0"},
		{"waiver absolute path", waivers(model.HealthWaiver{Code: "health.large_method", Path: "/a.go", Function: "f", Decision: "ADR-OK"}), "model.invalid_health_waiver", "health.waivers[0].path", "is absolute"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) { assertHealthCase(t, c) })
	}
}

// healthCase is a health block with the one diagnostic it must report.
type healthCase struct {
	name   string
	health model.ManifestHealth
	code   string
	path   string
	in     string // a fragment of the message
}

// assertHealthCase checks that the case reports exactly its code at its path
// as a core error.
func assertHealthCase(t *testing.T, c healthCase) {
	t.Helper()
	h := c.health
	got := healthCodes(healthBundle(&h))
	if len(got) != 1 {
		t.Fatalf("want one health diagnostic, got %+v", got)
	}
	d := got[0]
	if d.Code != c.code || d.Path != c.path || d.Severity != SeverityError {
		t.Fatalf("got %+v, want %s at %s", d, c.code, c.path)
	}
	if !strings.Contains(d.Message, c.in) {
		t.Fatalf("message %q lacks %q", d.Message, c.in)
	}
	if !IsCoreCode(d.Code) {
		t.Fatalf("%s must be a core code: a broken configuration is never deferred", d.Code)
	}
}

// A waiver is justified only by an existing, accepted decision: accepted
// passes; a proposed and a missing decision are health.waiver_unjustified
// (error) naming the waiver and the decision.
// TRLC-LINKS: REQ-EMG-154
func TestHealthWaiverNeedsAcceptedDecision(t *testing.T) {
	if got := healthCodes(healthBundle(waiverWithDecision("ADR-OK"))); len(got) != 0 {
		t.Fatalf("accepted decision: got %+v", got)
	}
	for decision, problem := range map[string]string{"ADR-DRAFT": `has status "proposed", not accepted`, "ADR-NONE": "does not exist"} {
		assertWaiverUnjustified(t, decision, problem)
	}
}

// waiverWithDecision is a health block with the valid waiver naming the decision.
func waiverWithDecision(decision string) *model.ManifestHealth {
	w := okWaiver()
	w.Decision = decision
	return &model.ManifestHealth{Waivers: []model.HealthWaiver{w}}
}

// assertWaiverUnjustified checks that a waiver naming the decision is reported as unjustified with the problem.
func assertWaiverUnjustified(t *testing.T, decision, problem string) {
	t.Helper()
	got := healthCodes(healthBundle(waiverWithDecision(decision)))
	if len(got) != 1 || !isErrorAt(got[0], "health.waiver_unjustified", "health.waivers[0].decision") {
		t.Fatalf("%s: got %+v", decision, got)
	}
	assertMessageHas(t, got[0], "health.complex_method", "parse", "internal/a.go", decision, problem)
}

// isErrorAt reports whether a diagnostic is an error with the code at the path.
func isErrorAt(d Diagnostic, code, path string) bool {
	return d.Code == code && d.Severity == SeverityError && d.Path == path
}

// assertMessageHas checks that the message of the diagnostic holds every fragment.
func assertMessageHas(t *testing.T, d Diagnostic, fragments ...string) {
	t.Helper()
	for _, want := range fragments {
		if !strings.Contains(d.Message, want) {
			t.Errorf("message %q lacks %q", d.Message, want)
		}
	}
}

// health.waiver_unjustified is untiered: not in the table, not a core code,
// and at no tier is it lowered or deferred, not even by an open POA&M item
// that names it.
// TRLC-LINKS: REQ-EMG-154, REQ-EMG-155
func TestHealthWaiverUnjustifiedIsUntieredAndNeverDeferred(t *testing.T) {
	if TierKeyed("health.waiver_unjustified") || IsCoreCode("health.waiver_unjustified") {
		t.Fatal("health.waiver_unjustified must be untiered and not core")
	}
	for _, tier := range model.Tiers {
		assertUnjustifiedStaysAnError(t, tier)
	}
}

// assertUnjustifiedStaysAnError checks that at the tier the unjustified waiver is
// an untiered error that an open POA&M item naming health.* does not defer.
func assertUnjustifiedStaysAnError(t *testing.T, tier string) {
	t.Helper()
	b := healthBundle(&model.ManifestHealth{Waivers: []model.HealthWaiver{unjustifiedWaiver()}})
	b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier}
	b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-1", Milestone: "tier:" + tier, Status: "open", Defers: []model.POAMDefer{{Code: "health.*"}}}}
	found := diagnosticsWithCode(ApplyTierPolicy(b, Bundle(b)), "health.waiver_unjustified")
	if len(found) == 0 {
		t.Errorf("%s: no health.waiver_unjustified", tier)
	}
	for _, d := range found {
		if !isUntieredError(d) {
			t.Errorf("%s: untiered error changed: %+v", tier, d)
		}
	}
}

// diagnosticsWithCode lists the diagnostics with the code.
func diagnosticsWithCode(diags []Diagnostic, code string) []Diagnostic {
	var out []Diagnostic
	for _, d := range diags {
		if d.Code == code {
			out = append(out, d)
		}
	}
	return out
}

// isUntieredError reports whether a diagnostic is an error that no tier
// keyed, deferred or required from a tier.
func isUntieredError(d Diagnostic) bool {
	if d.Severity != SeverityError {
		return false
	}
	return !d.Deferred && d.PoamRef == "" && d.RequiredFrom == ""
}

// unjustifiedWaiver is the valid waiver with a decision that does not exist.
func unjustifiedWaiver() model.HealthWaiver {
	w := okWaiver()
	w.Decision = "ADR-NONE"
	return w
}

// The health tier rows end to end through ApplyTierPolicy with lifecycle at
// each tier: debt, unused waivers, regressions and policy weakenings are
// W,W,W,E,E (warnings up to mvp, errors at product and operational), and
// W,W,E,E,E with the safety bundle; a stale baseline entry is W,W,W,W,E;
// lifecycle.raise health.* makes debt an error; without lifecycle nothing is
// keyed.
// TRLC-LINKS: REQ-EMG-155
func TestHealthTierRowsEndToEnd(t *testing.T) {
	debt := model.HealthSmellCodes()
	for _, r := range healthTierRows(debt) {
		assertHealthRow(t, r)
	}
	// regression RequiredFrom follows the row.
	if d := severityAt(t, tierBundle("poc"), probe("health.regression", "p", SeverityWarning)); d.RequiredFrom != "product" {
		t.Errorf("regression requiredFrom = %q", d.RequiredFrom)
	}
	if d := severityAt(t, tierBundle("poc", "safety"), probe("health.regression", "p", SeverityWarning)); d.RequiredFrom != "mvp" {
		t.Errorf("regression with safety requiredFrom = %q", d.RequiredFrom)
	}
	assertRaiseMakesDebtBlocking(t, debt)
}

// healthTierRow is a health code with its severity per tier, E or W, with the bundles.
type healthTierRow struct {
	code    string
	bundles []string
	want    string // E or W per tier
}

// healthTierRows lists the rows of the health tier policy, with a row for each debt code with and without safety.
func healthTierRows(debt []string) []healthTierRow {
	rows := []healthTierRow{
		{"health.regression", nil, "WWWEE"},
		{"health.regression", []string{"safety"}, "WWEEE"},
		{model.HealthPolicyWeakenedCode, nil, "WWWEE"},
		{model.HealthPolicyWeakenedCode, []string{"safety"}, "WWEEE"},
		{"health.waiver_unused", nil, "WWWEE"},
		{"health.waiver_unused", []string{"safety"}, "WWEEE"},
		// A stale baseline entry is a warning below operational and an error at
		// operational, with or without the safety bundle (ADR-EMG-059).
		{model.HealthBaselineStaleCode, nil, "WWWWE"},
		{model.HealthBaselineStaleCode, []string{"safety"}, "WWWWE"},
	}
	for _, code := range debt {
		rows = append(rows, healthTierRow{code, nil, "WWWEE"}, healthTierRow{code, []string{"safety"}, "WWEEE"})
	}
	return rows
}

// assertHealthRow checks the severity of the row's code at every tier, that none
// is deferred and that without lifecycle the diagnostic is unchanged.
func assertHealthRow(t *testing.T, r healthTierRow) {
	t.Helper()
	var got strings.Builder
	for _, tier := range model.Tiers {
		d := severityAt(t, tierBundle(tier, r.bundles...), probe(r.code, "p", SeverityWarning))
		if d.Deferred {
			t.Errorf("%s at %s must not be deferred", r.code, tier)
		}
		got.WriteString(map[Severity]string{SeverityWarning: "W", SeverityError: "E"}[d.Severity])
	}
	if got.String() != r.want {
		t.Errorf("%s %v: %s, want %s", r.code, r.bundles, got.String(), r.want)
	}
	if d := severityAt(t, tierBundle(""), probe(r.code, "p", SeverityWarning)); d.Severity != SeverityWarning || d.RequiredFrom != "" {
		t.Errorf("%s without lifecycle changed: %+v", r.code, d)
	}
}

// assertRaiseMakesDebtBlocking checks the escape hatch: lifecycle.raise health.* makes debt an error.
func assertRaiseMakesDebtBlocking(t *testing.T, debt []string) {
	t.Helper()
	b := tierBundle("poc")
	b.Manifest.Lifecycle.Raise = []string{"health.*"}
	for _, code := range debt {
		if d := severityAt(t, b, probe(code, "p", SeverityWarning)); d.Severity != SeverityError || d.Deferred {
			t.Errorf("%s raised: %+v", code, d)
		}
	}
}

// Every health smell code, the report codes and the table agree: all are tier
// keyed, none is a core code, the markdown table carries the rows and the
// authoring rules list them.
// TRLC-LINKS: REQ-EMG-155
func TestHealthCodesAreInTheTierTable(t *testing.T) {
	assertHealthCodesKeyed(t)
	assertThresholdCodesAreSmellCodes(t)
	assertHealthRowsInTheTable(t, TierPolicyMarkdown())
	assertHealthCodesUntiered(t, "health.waiver_unjustified", model.HealthBaselineUnreadableCode)
}

// assertThresholdCodesAreSmellCodes checks that every threshold code is a smell code.
func assertThresholdCodesAreSmellCodes(t *testing.T) {
	t.Helper()
	for _, code := range model.HealthThresholdCodes() {
		if !model.IsHealthSmellCode(code) {
			t.Errorf("threshold code %s is no smell code", code)
		}
	}
}

// assertHealthCodesUntiered checks that none of the codes is tier keyed.
func assertHealthCodesUntiered(t *testing.T, codes ...string) {
	t.Helper()
	for _, code := range codes {
		if TierKeyed(code) {
			t.Errorf("%s must be untiered", code)
		}
	}
}

// assertHealthCodesKeyed checks that every health code is tier keyed, none is core and none is bundle specific.
func assertHealthCodesKeyed(t *testing.T) {
	t.Helper()
	keyed := map[string]TierRule{}
	for _, r := range TierPolicy() {
		keyed[r.Code] = r
	}
	for _, code := range append(model.HealthSmellCodes(), "health.waiver_unused", "health.regression", model.HealthPolicyWeakenedCode, model.HealthBaselineStaleCode, model.HealthFileBelowFloorCode) {
		r, ok := keyed[code]
		if !ok || IsCoreCode(code) {
			t.Errorf("%s: tier keyed=%v core=%v", code, ok, IsCoreCode(code))
		}
		if r.Bundle != "" {
			t.Errorf("%s must not be bundle specific", code)
		}
	}
}

// assertHealthRowsInTheTable checks the markdown rows of the health codes.
func assertHealthRowsInTheTable(t *testing.T, md string) {
	t.Helper()
	for _, row := range []string{
		"| `health.complex_method` | W | W | W | E | E |",
		"| `health.complex_method` with safety | W | W | E | E | E |",
		"| `health.regression` | W | W | W | E | E |",
		"| `health.regression` with safety | W | W | E | E | E |",
		"| `health.policy_weakened` | W | W | W | E | E |",
		"| `health.policy_weakened` with safety | W | W | E | E | E |",
		"| `health.waiver_unused` | W | W | W | E | E |",
		"| `health.file_below_floor` | W | W | W | E | E |",
		"| `health.file_below_floor` with safety | W | W | E | E | E |",
		"| `health.baseline_stale` | W | W | W | W | E |",
	} {
		if !strings.Contains(md, row) {
			t.Errorf("table lacks %s", row)
		}
	}
}

// The authoring rules list the health field sets and describe the block and
// the waiver rule; the nested limits are typed healthThreshold.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154
func TestAuthoringRulesDescribeHealth(t *testing.T) {
	rules := AuthoringRules()
	for kind, want := range map[string]string{
		"health":          "exclude,thresholds,overrides,waivers,baseline,policyDecision,floor",
		"healthBaseline":  "file,decision",
		"healthThreshold": "warn,high",
		"healthOverride":  "path,thresholds",
		"healthWaiver":    "code,path,function,decision",
	} {
		if got := strings.Join(rules.FieldSets[kind], ","); got != want {
			t.Errorf("fieldSets[%s] = %s, want %s", kind, got, want)
		}
	}
	if thresholds := fieldTypeNamed(rules.FieldTypes["health"], "thresholds"); thresholds.Type != "map<string,object>" || thresholds.Of != "healthThreshold" {
		t.Errorf("health.thresholds type = %+v", thresholds)
	}
	assertHealthFieldRules(t, strings.Join(rules.FieldRules, "\n"))
}

// fieldTypeNamed is the field with the name, or the zero field.
func fieldTypeNamed(fields []FieldType, name string) FieldType {
	for _, f := range fields {
		if f.Name == name {
			return f
		}
	}
	return FieldType{}
}

// assertHealthFieldRules checks that the field rules text names the health terms.
func assertHealthFieldRules(t *testing.T, text string) {
	t.Helper()
	assertTextHas(t, text, []string{"Code health: engmod.yml health", "health.block_missing", "health.waiver_unjustified", "model.invalid_health_threshold", "lifecycle.raise: [health.*]", "health.regression", "health.policy_weakened", "policyDecision", "health.file_below_floor", "health.floor", "model.invalid_health_floor"})
	assertTextHas(t, text, []string{"health.baseline", "health.baseline_stale", "health.baseline_unreadable", "model.invalid_health_baseline", "engmod-health-baseline/1", "never baselined", "suppresses nothing", "path, function, code, value"})
	assertTextHas(t, text, model.HealthThresholdCodes())
}

// assertTextHas checks that the text holds every fragment.
func assertTextHas(t *testing.T, text string, fragments []string) {
	t.Helper()
	for _, want := range fragments {
		if !strings.Contains(text, want) {
			t.Errorf("field rules lack %q", want)
		}
	}
}

// waivers is a health block with the given waivers.
func waivers(w ...model.HealthWaiver) model.ManifestHealth {
	return model.ManifestHealth{Waivers: w}
}

// health.policyDecision records a weakening of the health policy and is
// justified like a waiver: an accepted decision passes; a proposed and a
// missing decision are health.waiver_unjustified (error, untiered) at
// health.policyDecision, naming the policy decision; without it nothing is
// reported (REQ-EMG-160).
//
// Mutation: not checking the decision, or reporting another code, fails.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-160
func TestHealthPolicyDecisionNeedsAnAcceptedDecision(t *testing.T) {
	if got := healthCodes(healthBundle(&model.ManifestHealth{PolicyDecision: "ADR-OK"})); len(got) != 0 {
		t.Fatalf("accepted decision: got %+v", got)
	}
	if got := healthCodes(healthBundle(&model.ManifestHealth{})); len(got) != 0 {
		t.Fatalf("no policy decision: got %+v", got)
	}
	for decision, problem := range map[string]string{"ADR-DRAFT": `has status "proposed", not accepted`, "ADR-NONE": "does not exist"} {
		assertPolicyDecisionProblem(t, decision, problem)
	}
	if got := AcceptedHealthPolicyDecision(healthBundle(&model.ManifestHealth{PolicyDecision: " ADR-OK "})); got != "ADR-OK" {
		t.Errorf("AcceptedHealthPolicyDecision = %q", got)
	}
	if got := AcceptedHealthPolicyDecision(healthBundle(nil)); got != "" {
		t.Errorf("no health block = %q", got)
	}
	if TierKeyed("health.waiver_unjustified") {
		t.Error("the diagnostic must stay untiered")
	}
}

// assertPolicyDecisionProblem checks that a policyDecision naming the decision
// is reported as unjustified with the problem and records nothing.
func assertPolicyDecisionProblem(t *testing.T, decision, problem string) {
	t.Helper()
	got := healthCodes(healthBundle(&model.ManifestHealth{PolicyDecision: decision}))
	if len(got) != 1 || !isErrorAt(got[0], model.HealthWaiverUnjustifiedCode, "health.policyDecision") {
		t.Fatalf("%s: got %+v", decision, got)
	}
	assertMessageHas(t, got[0], "policy decision", decision, problem)
	if AcceptedHealthPolicyDecision(healthBundle(&model.ManifestHealth{PolicyDecision: decision})) != "" {
		t.Errorf("%s must record nothing", decision)
	}
}

// HealthBaselineDecisionProblem says why a baseline is unjustified and is empty
// for an accepted decision and for a model without a baseline.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154
func TestHealthBaselineDecisionProblem(t *testing.T) {
	with := func(decision string) model.Bundle {
		return healthBundle(&model.ManifestHealth{Baseline: &model.HealthBaseline{File: "model/b.json", Decision: decision}})
	}
	for decision, want := range map[string]string{"ADR-OK": "", "ADR-DRAFT": "not accepted", "ADR-NONE": "does not exist", "": "no decision"} {
		got := HealthBaselineDecisionProblem(with(decision))
		if (want == "") != (got == "") || !strings.Contains(got, want) {
			t.Errorf("decision %q: problem %q, want it to contain %q", decision, got, want)
		}
	}
	if HealthBaselineDecisionProblem(healthBundle(&model.ManifestHealth{})) != "" || HealthBaselineDecisionProblem(healthBundle(nil)) != "" {
		t.Error("a model without a baseline has no problem")
	}
}
