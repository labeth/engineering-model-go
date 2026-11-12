// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// completeBundleAt is a model with every required switch written, at the
// tier: a lifecycle {tier, bundles [foundation], tierDecision} with an
// accepted DEC-TIER-<TIER> decision whose Tier: and Bundles: lines agree,
// requirementLinks strict, lintRun.mode strict, schemaVersion 2, a model
// id, a health block, and one derived requirement so that the strict link
// rules are met. It has no error.
func completeBundleAt(tier string) model.Bundle {
	b := behaviorTestBundle(nil, nil)
	id := "DEC-TIER-" + strings.ToUpper(tier)
	b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier, Bundles: []string{"foundation"}, TierDecision: id}
	b.Architecture.Decisions = []model.Decision{tierDecision(id, "accepted", "Tier: "+displayTier(tier)+"\nBundles: foundation")}
	b.Manifest.RequirementLinks = model.RequirementLinksStrict
	b.Requirements.LintRun.Mode = "strict"
	b.Manifest.Health = &model.ManifestHealth{}
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-A", Text: "The unit shall run.", AppliesTo: []string{"FU-A"}, Derived: true, DerivedRationale: "Operability.", Status: "verified"}}
	return b
}

// completeBundle is completeBundleAt mvp, the first tier with strict links.
func completeBundle(t *testing.T) model.Bundle {
	t.Helper()
	return completeBundleAt(model.TierMVP)
}

// errorCodes returns the codes of the error diagnostics.
func errorCodes(diags []Diagnostic) []string {
	var out []string
	for _, d := range diags {
		if d.Severity == SeverityError {
			out = append(out, d.Code)
		}
	}
	return out
}

// A complete model validates without an error at every tier, with the
// health block required only from product.
// Mutation: a required-switch rule that fires on a complete model fails this.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func TestCompleteBundleHasNoErrors(t *testing.T) {
	if got := errorCodes(Bundle(completeBundle(t))); len(got) != 0 {
		t.Fatalf("complete mvp bundle has errors %v", got)
	}
	for _, tier := range model.Tiers {
		b := completeBundleAt(tier)
		if model.TierRank(tier) < model.TierRank(model.TierMVP) {
			b.Manifest.RequirementLinks = model.RequirementLinksGuided
		}
		if got := errorCodes(ApplyTierPolicy(b, Bundle(b))); len(got) != 0 {
			t.Errorf("complete %s bundle has errors %v", tier, got)
		}
	}
}

// requiredCase triggers one required-switch code with a minimal edit of a
// complete bundle.
type requiredCase struct {
	name   string
	code   string
	remedy string // substring of the remedy
	path   string
	edit   func(*model.Bundle)
}

var requiredCases = []requiredCase{
	{"no lifecycle", "lifecycle.missing", "ef-tier-up", "lifecycle",
		func(b *model.Bundle) { b.Manifest.Lifecycle = nil }},
	{"unknown tier", "lifecycle.invalid_tier", "ef-tier-up", "lifecycle.tier",
		func(b *model.Bundle) { b.Manifest.Lifecycle.Tier = "beta" }},
	{"no bundles", "lifecycle.bundles_missing", "ef-tier-up", "lifecycle.bundles",
		func(b *model.Bundle) { b.Manifest.Lifecycle.Bundles = nil }},
	{"unknown bundle", "lifecycle.invalid_bundle", "ef-tier-up", "lifecycle.bundles[0]",
		func(b *model.Bundle) { b.Manifest.Lifecycle.Bundles = []string{"comfort"} }},
	{"no tier decision", "lifecycle.tier_decision_missing", "write engmod.yml lifecycle {tier, bundles, tierDecision}", "lifecycle.tierDecision",
		func(b *model.Bundle) { b.Manifest.Lifecycle.TierDecision = "" }},
	{"decision without Bundles line", "lifecycle.tier_decision_mismatch", "ef-tier-up", "lifecycle.tierDecision",
		func(b *model.Bundle) { b.Architecture.Decisions[0].Decision = "Tier: MVP" }},
	{"no requirementLinks", "model.requirement_links_missing", "requirementLinks: strict", "requirementLinks",
		func(b *model.Bundle) { b.Manifest.RequirementLinks = "" }},
	{"unknown requirementLinks", "model.invalid_requirement_links", "requirementLinks: strict", "requirementLinks",
		func(b *model.Bundle) { b.Manifest.RequirementLinks = "lenient" }},
	{"guided links at mvp", "lifecycle.requirement_links_conflict", "requirementLinks: strict", "requirementLinks",
		func(b *model.Bundle) { b.Manifest.RequirementLinks = "guided" }},
	{"no lint mode", "lint.mode_missing", "lintRun.mode: strict", "lintRun.mode",
		func(b *model.Bundle) { b.Requirements.LintRun.Mode = "" }},
	{"unknown lint mode", "lint.mode_invalid", "lintRun.mode: strict", "lintRun.mode",
		func(b *model.Bundle) { b.Requirements.LintRun.Mode = "lenient" }},
	{"guided lint at mvp", "lint.mode_guided_at_tier", "lintRun.mode: strict", "lintRun.mode",
		func(b *model.Bundle) { b.Requirements.LintRun.Mode = "guided" }},
	{"no health block at product", "health.block_missing", "enghealth baseline write", "health",
		func(b *model.Bundle) {
			*b = completeBundleAt(model.TierProduct)
			b.Manifest.Health = nil
		}},
	{"schemaVersion 0", "model.unsupported_schema_version", "engflow upgrade (engineering-flow) or set schemaVersion: 2", "requirements.schemaVersion",
		func(b *model.Bundle) { b.Requirements.SchemaVersion = 0 }},
	{"schemaVersion 1", "model.unsupported_schema_version", "schemaVersion: 2", "design.schemaVersion",
		func(b *model.Bundle) { b.Design.SchemaVersion = 1 }},
	{"no model id", "model.id_missing", "module.modelId", "model.id",
		func(b *model.Bundle) { b.Architecture.Model.ID = "" }},
}

// TestRequiredSwitchesHaveRemedies is the inventory of required switches
// (ADR-EMG-061): each row removes one switch from a complete model and
// expects its code, an error that no tier changes, and the remedy text.
// Every code of the remedy table has a row and every row's code has a
// remedy, so adding a required code without a remedy fails here.
// Mutation: dropping a remedy from requiredSwitches, or a code from the
// rows, fails the inventory comparison.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func TestRequiredSwitchesHaveRemedies(t *testing.T) {
	rowCodes := map[string]bool{}
	for _, c := range requiredCases {
		rowCodes[c.code] = true
		b := completeBundle(t)
		c.edit(&b)
		got := codesOf(ApplyTierPolicy(b, Bundle(b)), c.code)
		if len(got) == 0 {
			t.Errorf("%s: %s not reported", c.name, c.code)
			continue
		}
		d := got[0]
		if d.Severity != SeverityError || d.Deferred || d.PoamRef != "" {
			t.Errorf("%s: %s must be an undeferred error, got %+v", c.name, c.code, d)
		}
		if d.Path != c.path {
			t.Errorf("%s: path %q, want %q", c.name, d.Path, c.path)
		}
		if !strings.Contains(d.Remedy, c.remedy) {
			t.Errorf("%s: remedy %q lacks %q", c.name, d.Remedy, c.remedy)
		}
		if d.Remedy != remedyFor(c.code) || d.Remedy == "" {
			t.Errorf("%s: remedy %q differs from the table %q", c.name, d.Remedy, remedyFor(c.code))
		}
		if !IsCoreCode(c.code) || TierKeyed(c.code) {
			t.Errorf("%s: %s must be a rank-0 core code and not tier keyed", c.name, c.code)
		}
	}
	tableCodes := map[string]bool{}
	for _, s := range RequiredSwitches() {
		tableCodes[s.Code] = true
		if strings.TrimSpace(s.Switch) == "" || strings.TrimSpace(s.Remedy) == "" {
			t.Errorf("required switch %+v has no name or remedy", s)
		}
		if !rowCodes[s.Code] {
			t.Errorf("remedy table code %s has no inventory row", s.Code)
		}
	}
	for code := range rowCodes {
		if !tableCodes[code] {
			t.Errorf("inventory code %s has no remedy in the table", code)
		}
	}
	if remedyFor("requirement.orphan") != "" {
		t.Error("a code outside the table must have no remedy")
	}
}

// Each required code is rank 0 end to end: an open POA&M item at the
// project tier cannot defer it, lifecycle.raise changes nothing, and the
// POA&M that names it is itself an error.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-129
func TestRequiredSwitchesCannotBeDeferred(t *testing.T) {
	for _, c := range requiredCases {
		b := completeBundle(t)
		c.edit(&b)
		if b.Manifest.Lifecycle == nil {
			b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: "mvp", Bundles: []string{"foundation"}}
		}
		tier := strings.ToLower(b.Manifest.Lifecycle.Tier)
		if model.TierRank(tier) == 0 {
			tier = "mvp"
		}
		b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-X", Milestone: "tier:" + tier, Status: "in-progress", Defers: []model.POAMDefer{{Code: c.code}}}}
		in := []Diagnostic{{Code: c.code, Severity: SeverityError, Path: c.path}}
		out := ApplyTierPolicy(b, in)
		if out[0].Severity != SeverityError || out[0].Deferred || out[0].PoamRef != "" {
			t.Errorf("%s: %s was changed by the tier policy: %+v", c.name, c.code, out[0])
		}
		if got := codesOf(Bundle(b), "lifecycle.poam_defers_core"); len(got) != 1 {
			t.Errorf("%s: a defer naming %s must be lifecycle.poam_defers_core, got %+v", c.name, c.code, got)
		}
	}
}

// A model without lifecycle is lifecycle.missing, an error at the
// lifecycle path; the library keeps loading it (no tier rule applies) and
// every other switch is still checked.
// TRLC-LINKS: REQ-EMG-124
func TestLifecycleMissingIsAnErrorAndTheLibraryStaysLenient(t *testing.T) {
	b := completeBundle(t)
	b.Manifest.Lifecycle = nil
	diags := ApplyTierPolicy(b, Bundle(b))
	if got := errorCodes(diags); len(got) != 1 || got[0] != "lifecycle.missing" {
		t.Fatalf("errors = %v, want only lifecycle.missing", got)
	}
	if _, _, ok := EffectiveTier(b); ok {
		t.Error("EffectiveTier must stay ok=false without lifecycle (library behaviour)")
	}
	in := []Diagnostic{{Code: "requirement.orphan", Severity: SeverityWarning}}
	if out := ApplyTierPolicy(b, in); out[0].Severity != SeverityWarning || out[0].RequiredFrom != "" {
		t.Errorf("ApplyTierPolicy must return diagnostics unchanged without lifecycle: %+v", out[0])
	}
}

// The tier decision must carry a Bundles: line; a decision without one no
// longer means foundation.
// TRLC-LINKS: REQ-EMG-124
func TestTierDecisionNeedsABundlesLine(t *testing.T) {
	b := completeBundle(t)
	if got := codesOf(Bundle(b), "lifecycle.tier_decision_mismatch"); len(got) != 0 {
		t.Fatalf("complete bundle: %+v", got)
	}
	b.Architecture.Decisions[0].Decision = "Tier: MVP"
	got := codesOf(Bundle(b), "lifecycle.tier_decision_mismatch")
	if len(got) != 1 || !strings.Contains(got[0].Message, "Bundles:") {
		t.Fatalf("got %+v, want one mismatch naming the Bundles: line", got)
	}
}

// The health block is required at product and operational and nowhere
// below; an empty block satisfies it.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func TestHealthBlockRequiredFromProduct(t *testing.T) {
	for _, tier := range model.Tiers {
		b := completeBundleAt(tier)
		b.Manifest.Health = nil
		want := model.TierRank(tier) >= model.TierRank(model.TierProduct)
		got := len(codesOf(Bundle(b), "health.block_missing")) == 1
		if got != want {
			t.Errorf("%s without health block: reported=%v, want %v", tier, got, want)
		}
		b.Manifest.Health = &model.ManifestHealth{}
		if len(codesOf(Bundle(b), "health.block_missing")) != 0 {
			t.Errorf("%s with an empty health block still reports health.block_missing", tier)
		}
	}
}

// lintRun.mode guided is allowed below mvp and an error from mvp; strict
// is accepted everywhere and an unknown mode is lint.mode_invalid.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125
func TestLintModeByTier(t *testing.T) {
	for _, tier := range model.Tiers {
		b := completeBundleAt(tier)
		b.Requirements.LintRun.Mode = "Guided"
		want := model.TierRank(tier) >= model.TierRank(model.TierMVP)
		if got := len(codesOf(Bundle(b), "lint.mode_guided_at_tier")) == 1; got != want {
			t.Errorf("guided at %s: reported=%v, want %v", tier, got, want)
		}
		b.Requirements.LintRun.Mode = " STRICT "
		if got := Bundle(b); len(codesOf(got, "lint.mode_guided_at_tier"))+len(codesOf(got, "lint.mode_invalid"))+len(codesOf(got, "lint.mode_missing")) != 0 {
			t.Errorf("strict at %s reported a lint mode error", tier)
		}
	}
}

// requirementLinks may be strict or guided at poc and prototype, must be
// strict from mvp, and empty is model.requirement_links_missing at every
// tier; the schema-version error carries the upgrade remedy in its text.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-127
func TestRequirementLinksRequiredAtEveryTier(t *testing.T) {
	for _, tier := range model.Tiers {
		b := completeBundleAt(tier)
		b.Manifest.RequirementLinks = ""
		if len(codesOf(Bundle(b), "model.requirement_links_missing")) != 1 {
			t.Errorf("%s: empty requirementLinks not reported", tier)
		}
		b.Manifest.RequirementLinks = "guided"
		below := model.TierRank(tier) < model.TierRank(model.TierMVP)
		if got := len(codesOf(Bundle(b), "lifecycle.requirement_links_conflict")) == 0; got != below {
			t.Errorf("guided at %s: allowed=%v, want %v", tier, got, below)
		}
	}
	_, err := model.EffectiveSchemaVersion(1)
	var versionErr *model.SchemaVersionError
	if !errors.As(err, &versionErr) || !strings.Contains(err.Error(), "fix: "+model.SchemaVersionRemedy) {
		t.Fatalf("schema version error = %v", err)
	}
}

// The remedy is a json field and prints as a fix line after the diagnostic;
// a diagnostic without a remedy prints no fix line.
// TRLC-LINKS: REQ-EMG-125
func TestDiagnosticRemedyIsRenderedAndSerialized(t *testing.T) {
	b := completeBundle(t)
	b.Manifest.Lifecycle = nil
	diag := codesOf(Bundle(b), "lifecycle.missing")[0]
	var out bytes.Buffer
	WriteDiagnostic(&out, diag)
	lines := strings.Split(strings.TrimRight(out.String(), "\n"), "\n")
	if len(lines) != 2 || !strings.HasPrefix(lines[0], "lifecycle.missing [error] ") || !strings.HasSuffix(lines[0], "(lifecycle)") || lines[1] != "  fix: "+remedyFor("lifecycle.missing") {
		t.Fatalf("rendered %q", out.String())
	}
	data, err := json.Marshal(diag)
	if err != nil || !strings.Contains(string(data), `"remedy":"run the ef-tier-up skill`) {
		t.Fatalf("json = %s, %v", data, err)
	}
	out.Reset()
	WriteDiagnostic(&out, Diagnostic{Code: "x.y", Severity: SeverityWarning, Message: "m"})
	if out.String() != "x.y [warning] m\n" {
		t.Fatalf("rendered %q", out.String())
	}
	if data, _ := json.Marshal(Diagnostic{Code: "x.y"}); strings.Contains(string(data), "remedy") {
		t.Fatalf("an empty remedy must be omitted: %s", data)
	}
}

// The authoring contract rules list the required switches, and the README
// "Required model switches" table is the current one.
// TRLC-LINKS: REQ-EMG-045, REQ-EMG-124, REQ-EMG-125
func TestRequiredSwitchesAreInTheContractAndTheReadme(t *testing.T) {
	rules := AuthoringRules()
	if !reflect.DeepEqual(rules.Required, RequiredSwitches()) || len(rules.Required) == 0 {
		t.Fatalf("rules.required = %+v", rules.Required)
	}
	for _, r := range rules.Required {
		if r.Switch == "" || r.Code == "" || r.Remedy == "" {
			t.Errorf("incomplete required row %+v", r)
		}
	}
	data, err := json.Marshal(rules)
	if err != nil || !strings.Contains(string(data), `"required":[{"switch":"engmod.yml lifecycle","code":"lifecycle.missing","remedy":"run the ef-tier-up skill`) {
		t.Fatalf("rules json lacks the required section: %v", err)
	}
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if table := RequiredSwitchesMarkdown(); !strings.Contains(string(readme), table) {
		t.Fatalf("README.md does not contain the current RequiredSwitchesMarkdown() table:\n%s", table)
	}
}
