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
	cases := []struct {
		name   string
		health model.ManifestHealth
		code   string
		path   string
		in     string // a fragment of the message
	}{
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
		{"waiver absolute path", waivers(model.HealthWaiver{Code: "health.large_method", Path: "/a.go", Function: "f", Decision: "ADR-OK"}), "model.invalid_health_waiver", "health.waivers[0].path", "is absolute"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			h := c.health
			got := healthCodes(healthBundle(&h))
			if len(got) != 1 {
				t.Fatalf("want one health diagnostic, got %+v", got)
			}
			d := got[0]
			if d.Code != c.code || d.Path != c.path || d.Severity != SeverityError || !strings.Contains(d.Message, c.in) {
				t.Fatalf("got %+v, want %s at %s containing %q", d, c.code, c.path, c.in)
			}
			if !IsCoreCode(d.Code) {
				t.Fatalf("%s must be a core code: a broken configuration is never deferred", d.Code)
			}
		})
	}
}

// A waiver is justified only by an existing, accepted decision: accepted
// passes; a proposed and a missing decision are health.waiver_unjustified
// (error) naming the waiver and the decision.
// TRLC-LINKS: REQ-EMG-154
func TestHealthWaiverNeedsAcceptedDecision(t *testing.T) {
	waiver := func(decision string) *model.ManifestHealth {
		w := okWaiver()
		w.Decision = decision
		return &model.ManifestHealth{Waivers: []model.HealthWaiver{w}}
	}
	if got := healthCodes(healthBundle(waiver("ADR-OK"))); len(got) != 0 {
		t.Fatalf("accepted decision: got %+v", got)
	}
	for decision, problem := range map[string]string{"ADR-DRAFT": `has status "proposed", not accepted`, "ADR-NONE": "does not exist"} {
		got := healthCodes(healthBundle(waiver(decision)))
		if len(got) != 1 || got[0].Code != "health.waiver_unjustified" || got[0].Severity != SeverityError || got[0].Path != "health.waivers[0].decision" {
			t.Fatalf("%s: got %+v", decision, got)
		}
		for _, want := range []string{"health.complex_method", "parse", "internal/a.go", decision, problem} {
			if !strings.Contains(got[0].Message, want) {
				t.Errorf("%s: message %q lacks %q", decision, got[0].Message, want)
			}
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
	w := okWaiver()
	w.Decision = "ADR-NONE"
	for _, tier := range model.Tiers {
		b := healthBundle(&model.ManifestHealth{Waivers: []model.HealthWaiver{w}})
		b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier}
		b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-1", Milestone: "tier:" + tier, Status: "open", Defers: []model.POAMDefer{{Code: "health.*"}}}}
		var found bool
		for _, d := range ApplyTierPolicy(b, Bundle(b)) {
			if d.Code == "health.waiver_unjustified" {
				found = true
				if d.Severity != SeverityError || d.Deferred || d.PoamRef != "" || d.RequiredFrom != "" {
					t.Errorf("%s: untiered error changed: %+v", tier, d)
				}
			}
		}
		if !found {
			t.Errorf("%s: no health.waiver_unjustified", tier)
		}
	}
}

// The health tier rows end to end through ApplyTierPolicy with lifecycle at
// each tier: debt and unused waivers are warnings everywhere; the regression
// code is W,W,W,E,E, and W,W,E,E,E with the safety bundle; lifecycle.raise
// health.* makes debt an error; without lifecycle nothing is keyed.
// TRLC-LINKS: REQ-EMG-155
func TestHealthTierRowsEndToEnd(t *testing.T) {
	debt := model.HealthSmellCodes()
	rows := []struct {
		code    string
		bundles []string
		want    string // E or W per tier
	}{
		{"health.regression", nil, "WWWEE"},
		{"health.regression", []string{"safety"}, "WWEEE"},
		{"health.waiver_unused", nil, "WWWWW"},
		{"health.waiver_unused", []string{"safety"}, "WWWWW"},
	}
	for _, code := range debt {
		rows = append(rows, struct {
			code    string
			bundles []string
			want    string
		}{code, nil, "WWWWW"}, struct {
			code    string
			bundles []string
			want    string
		}{code, []string{"safety"}, "WWWWW"})
	}
	for _, r := range rows {
		var got strings.Builder
		for _, tier := range model.Tiers {
			d := severityAt(t, tierBundle(tier, r.bundles...), r.code, "p", SeverityWarning)
			if d.Deferred {
				t.Errorf("%s at %s must not be deferred", r.code, tier)
			}
			got.WriteString(map[Severity]string{SeverityWarning: "W", SeverityError: "E"}[d.Severity])
		}
		if got.String() != r.want {
			t.Errorf("%s %v: %s, want %s", r.code, r.bundles, got.String(), r.want)
		}
		if d := severityAt(t, tierBundle(""), r.code, "p", SeverityWarning); d.Severity != SeverityWarning || d.RequiredFrom != "" {
			t.Errorf("%s without lifecycle changed: %+v", r.code, d)
		}
	}
	// regression RequiredFrom follows the row.
	if d := severityAt(t, tierBundle("poc"), "health.regression", "p", SeverityWarning); d.RequiredFrom != "product" {
		t.Errorf("regression requiredFrom = %q", d.RequiredFrom)
	}
	if d := severityAt(t, tierBundle("poc", "safety"), "health.regression", "p", SeverityWarning); d.RequiredFrom != "mvp" {
		t.Errorf("regression with safety requiredFrom = %q", d.RequiredFrom)
	}
	// The escape hatch: lifecycle.raise health.* makes debt blocking.
	b := tierBundle("poc")
	b.Manifest.Lifecycle.Raise = []string{"health.*"}
	for _, code := range debt {
		if d := severityAt(t, b, code, "p", SeverityWarning); d.Severity != SeverityError || d.Deferred {
			t.Errorf("%s raised: %+v", code, d)
		}
	}
}

// Every health smell code, the report codes and the table agree: all are tier
// keyed, none is a core code, the markdown table carries the rows and the
// authoring rules list them.
// TRLC-LINKS: REQ-EMG-155
func TestHealthCodesAreInTheTierTable(t *testing.T) {
	keyed := map[string]TierRule{}
	for _, r := range TierPolicy() {
		keyed[r.Code] = r
	}
	for _, code := range append(model.HealthSmellCodes(), "health.waiver_unused", "health.regression") {
		r, ok := keyed[code]
		if !ok || IsCoreCode(code) {
			t.Errorf("%s: tier keyed=%v core=%v", code, ok, IsCoreCode(code))
		}
		if r.Bundle != "" {
			t.Errorf("%s must not be bundle specific", code)
		}
	}
	for _, code := range model.HealthThresholdCodes() {
		if !model.IsHealthSmellCode(code) {
			t.Errorf("threshold code %s is no smell code", code)
		}
	}
	md := TierPolicyMarkdown()
	for _, row := range []string{
		"| `health.complex_method` | W | W | W | W | W |",
		"| `health.regression` | W | W | W | E | E |",
		"| `health.regression` with safety | W | W | E | E | E |",
		"| `health.waiver_unused` | W | W | W | W | W |",
	} {
		if !strings.Contains(md, row) {
			t.Errorf("table lacks %s", row)
		}
	}
	for _, code := range []string{"health.waiver_unjustified"} {
		if TierKeyed(code) {
			t.Errorf("%s must be untiered", code)
		}
	}
}

// The authoring rules list the health field sets and describe the block and
// the waiver rule; the nested limits are typed healthThreshold.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154
func TestAuthoringRulesDescribeHealth(t *testing.T) {
	rules := AuthoringRules()
	for kind, want := range map[string]string{
		"health":          "exclude,thresholds,overrides,waivers",
		"healthThreshold": "warn,high",
		"healthOverride":  "path,thresholds",
		"healthWaiver":    "code,path,function,decision",
	} {
		if got := strings.Join(rules.FieldSets[kind], ","); got != want {
			t.Errorf("fieldSets[%s] = %s, want %s", kind, got, want)
		}
	}
	var thresholds FieldType
	for _, f := range rules.FieldTypes["health"] {
		if f.Name == "thresholds" {
			thresholds = f
		}
	}
	if thresholds.Type != "map<string,object>" || thresholds.Of != "healthThreshold" {
		t.Errorf("health.thresholds type = %+v", thresholds)
	}
	text := strings.Join(rules.FieldRules, "\n")
	for _, want := range []string{"Code health (optional)", "health.waiver_unjustified", "model.invalid_health_threshold", "lifecycle.raise: [health.*]", "health.regression"} {
		if !strings.Contains(text, want) {
			t.Errorf("field rules lack %q", want)
		}
	}
	for _, code := range model.HealthThresholdCodes() {
		if !strings.Contains(text, code) {
			t.Errorf("field rules lack threshold code %s", code)
		}
	}
}

// waivers is a health block with the given waivers.
func waivers(w ...model.HealthWaiver) model.ManifestHealth {
	return model.ManifestHealth{Waivers: w}
}
