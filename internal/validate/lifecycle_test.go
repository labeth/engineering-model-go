// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// tierDecision is an accepted DEC-TIER-* decision with Tier and Bundles lines.
func tierDecision(id, status, text string) model.Decision {
	return model.Decision{ID: id, Title: "Life-cycle tier", Status: status, Date: "2026-11-04", Context: "Tier.", Decision: text, Consequences: []string{"Applies."}}
}

// tierDecisionBundle is a model at tier with tierDecision naming dec.
func tierDecisionBundle(tier string, bundles []string, dec model.Decision) model.Bundle {
	b := tierBundle(tier, bundles...)
	b.Manifest.Lifecycle.TierDecision = dec.ID
	b.Architecture.Decisions = []model.Decision{dec}
	return b
}

// tierDecision must name an accepted decision whose Tier: and Bundles: lines
// agree with lifecycle (read case-insensitively; the Bundles: line is required,
// there is no default); anything else is lifecycle.tier_decision_mismatch
// (error).
// TRLC-LINKS: REQ-EMG-124
func TestLifecycleTierDecisionMustAgree(t *testing.T) {
	ok := tierDecision("DEC-TIER-MVP", "accepted", "Tier: MVP\nPrevious tier: none\nBundles: Foundation, Safety\nDecider: @owner")
	assertDecisionAgrees(t, tierDecisionBundle("mvp", []string{"foundation", "safety"}, ok))
	assertDecisionAgrees(t, tierDecisionBundle("poc", []string{"foundation"}, tierDecision("DEC-TIER-POC", "accepted", "Tier: PoC\nBundles: foundation")))
	for _, c := range tierDecisionCases(ok) {
		b := tierDecisionBundle(c.tier, c.bundles, c.dec)
		if c.ref != "" {
			b.Manifest.Lifecycle.TierDecision = c.ref
		}
		got := codesOf(Bundle(b), "lifecycle.tier_decision_mismatch")
		if len(got) != 1 || !isErrorAt(got[0], "lifecycle.tier_decision_mismatch", "lifecycle.tierDecision") || !strings.Contains(got[0].Message, c.want) {
			t.Errorf("%s: got %+v, want one error mentioning %q", c.name, got, c.want)
		}
	}
}

// assertDecisionAgrees checks that a decision agreeing with the lifecycle is not reported.
func assertDecisionAgrees(t *testing.T, b model.Bundle) {
	t.Helper()
	if got := codesOf(Bundle(b), "lifecycle.tier_decision_mismatch"); len(got) != 0 {
		t.Fatalf("agreeing decision reported: %+v", got)
	}
}

// tierDecisionCase is a tier decision that does not agree with the lifecycle,
// with the fragment its mismatch message holds.
type tierDecisionCase struct {
	name    string
	tier    string
	bundles []string
	dec     model.Decision
	ref     string
	want    string
}

// tierDecisionCases lists the ways a tier decision can disagree with the lifecycle.
func tierDecisionCases(ok model.Decision) []tierDecisionCase {
	return []tierDecisionCase{
		{"tier differs", "product", []string{"foundation", "safety"}, ok, "", `Tier: "MVP"`},
		{"bundles differ", "mvp", nil, ok, "", "Bundles"},
		{"proposed", "mvp", []string{"safety"}, tierDecision("DEC-TIER-MVP", "proposed", "Tier: MVP\nBundles: Foundation, Safety"), "", "accepted"},
		{"no bundles line", "poc", []string{"foundation"}, tierDecision("DEC-TIER-POC", "accepted", "Tier: PoC"), "", "Bundles:"},
		{"empty bundles line", "poc", []string{"foundation"}, tierDecision("DEC-TIER-POC", "accepted", "Tier: PoC\nBundles:"), "", "Bundles:"},
		{"no tier line", "mvp", nil, tierDecision("DEC-TIER-MVP", "accepted", "We go to MVP."), "", "Tier:"},
		{"unknown decision", "mvp", nil, ok, "DEC-TIER-NONE", "DEC-TIER-NONE"},
	}
}

// An unknown tier or bundle in a Go-built manifest is an error; requirementLinks
// guided at mvp or above is lifecycle.requirement_links_conflict (error),
// below mvp it is allowed.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-127
func TestLifecycleStructuralRules(t *testing.T) {
	b := tierBundle("beta", "comfort")
	diags := Bundle(b)
	if len(codesOf(diags, "lifecycle.invalid_tier")) != 1 || len(codesOf(diags, "lifecycle.invalid_bundle")) != 1 {
		t.Fatalf("got %+v", diags)
	}
	for tier, want := range map[string]int{"prototype": 0, "mvp": 1, "operational": 1} {
		assertLinksConflict(t, tier, want)
	}
}

// assertLinksConflict checks that requirementLinks guided at the tier is
// reported want times, as an error at requirementLinks.
func assertLinksConflict(t *testing.T, tier string, want int) {
	t.Helper()
	b := tierBundle(tier)
	b.Manifest.RequirementLinks = "guided"
	got := codesOf(Bundle(b), "lifecycle.requirement_links_conflict")
	if len(got) != want {
		t.Errorf("guided at %s: got %+v, want %d errors", tier, got, want)
		return
	}
	for _, d := range got {
		if !isErrorAt(d, "lifecycle.requirement_links_conflict", "requirementLinks") {
			t.Errorf("guided at %s: got %+v", tier, d)
		}
	}
}

// With lifecycle a requirement of status draft or proposed is
// lifecycle.requirement_unapproved and one not verified (rejected and
// retired excepted) lifecycle.requirement_unverified, at requirements[i].status;
// without lifecycle neither is reported.
// TRLC-LINKS: REQ-EMG-128
func TestLifecycleRequirementStatusRules(t *testing.T) {
	b := tierBundle("product")
	b.Requirements.Requirements = []model.Requirement{
		{ID: "REQ-1", Text: "The unit shall a.", AppliesTo: []string{"FU-A"}, Status: "draft"},
		{ID: "REQ-2", Text: "The unit shall b.", AppliesTo: []string{"FU-A"}, Status: "Proposed"},
		{ID: "REQ-3", Text: "The unit shall c.", AppliesTo: []string{"FU-A"}, Status: "approved"},
		{ID: "REQ-4", Text: "The unit shall d.", AppliesTo: []string{"FU-A"}, Status: "verified"},
		{ID: "REQ-5", Text: "The unit shall e.", AppliesTo: []string{"FU-A"}, Status: "retired"},
		{ID: "REQ-6", Text: "The unit shall f.", AppliesTo: []string{"FU-A"}},
	}
	paths := func(diags []Diagnostic) string {
		var p []string
		for _, d := range diags {
			p = append(p, d.Path)
		}
		return strings.Join(p, ",")
	}
	diags := Bundle(b)
	if got := paths(codesOf(diags, "lifecycle.requirement_unapproved")); got != "requirements[0].status,requirements[1].status" {
		t.Errorf("unapproved at %s", got)
	}
	if got := paths(codesOf(diags, "lifecycle.requirement_unverified")); got != "requirements[0].status,requirements[1].status,requirements[2].status,requirements[5].status" {
		t.Errorf("unverified at %s", got)
	}
	b.Manifest.Lifecycle = nil
	if got := append(codesOf(Bundle(b), "lifecycle.requirement_unapproved"), codesOf(Bundle(b), "lifecycle.requirement_unverified")...); len(got) != 0 {
		t.Errorf("without lifecycle: %+v", got)
	}
}

// The safety bundle needs documents.aviation (lifecycle.aviation_profile_missing)
// and a softwareLevel (lifecycle.software_level_missing); the foundation
// bundle needs neither.
// TRLC-LINKS: REQ-EMG-128
func TestLifecycleSafetyBundleNeedsAviation(t *testing.T) {
	b := tierBundle("mvp", "safety")
	diags := ApplyTierPolicy(b, Bundle(b))
	for _, code := range []string{"lifecycle.aviation_profile_missing", "lifecycle.software_level_missing"} {
		if got := codesOf(diags, code); len(got) != 1 || got[0].Severity != SeverityError {
			t.Errorf("%s at mvp with safety: got %+v, want one error", code, got)
		}
	}
	b.Aviation = &model.AviationDocument{Aviation: model.AviationProfile{Profile: model.AirborneProfile{SoftwareLevel: "C"}}}
	b.Manifest.Documents.Aviation = "model/aviation.yml"
	diags = Bundle(b)
	if got := append(codesOf(diags, "lifecycle.aviation_profile_missing"), codesOf(diags, "lifecycle.software_level_missing")...); len(got) != 0 {
		t.Errorf("with aviation: %+v", got)
	}
	plain := tierBundle("operational")
	if got := codesOf(Bundle(plain), "lifecycle.aviation_profile_missing"); len(got) != 0 {
		t.Errorf("foundation bundle: %+v", got)
	}
}

// An open POA&M item with milestone tier:<project tier> defers the
// diagnostics its defers entries name (code and, when given, path) to a
// warning with poamRef; another tier's item, a completed item and another
// path do not; an untiered error is never deferred; naming a rank-0 code,
// error or warning (composition.lock_check_skipped), is
// lifecycle.poam_defers_core (error) and defers nothing.
// TRLC-LINKS: REQ-EMG-129
func TestPOAMDefersBindDiagnostics(t *testing.T) {
	b := tierBundle("product")
	b.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{
		{ID: "POAM-TIER-PRODUCT-01", Milestone: "tier:product", Status: "in-progress", Defers: []model.POAMDefer{{Code: "requirement.orphan", Path: "requirements[REQ-1]"}}},
		{ID: "POAM-TIER-PRODUCT-02", Milestone: "tier:product", Status: "completed", Defers: []model.POAMDefer{{Code: "verification.unlinked"}}},
		{ID: "POAM-TIER-MVP-01", Milestone: "tier:mvp", Status: "planned", Defers: []model.POAMDefer{{Code: "link.claimed_only"}}},
		{ID: "POAM-TIER-PRODUCT-03", Milestone: "tier:product", Status: "planned", Defers: []model.POAMDefer{{Code: "model.duplicate_id"}, {Code: "lint.multiple_*"}, {Code: "composition.lock_check_skipped"}}},
	}
	in := []Diagnostic{
		{Code: "requirement.orphan", Severity: SeverityWarning, Path: "requirements[REQ-1]"},
		{Code: "requirement.orphan", Severity: SeverityWarning, Path: "requirements[REQ-2]"},
		{Code: "verification.unlinked", Severity: SeverityWarning, Path: "v"},
		{Code: "link.claimed_only", Severity: SeverityWarning, Path: "l"},
		{Code: "model.duplicate_id", Severity: SeverityError, Path: "d"},
		{Code: "lint.multiple_responses", Severity: SeverityWarning, Path: "x"},
		{Code: "lint.other", Severity: SeverityError, Path: "y"},
		{Code: "composition.lock_check_skipped", Severity: SeverityWarning, Path: "z"},
	}
	out := ApplyTierPolicy(b, in)
	for i, w := range []deferOutcome{
		{SeverityWarning, true, "POAM-TIER-PRODUCT-01"},
		{SeverityError, false, ""},
		{SeverityError, false, ""},
		{SeverityError, false, ""},
		{SeverityError, false, ""},
		{SeverityWarning, true, "POAM-TIER-PRODUCT-03"},
		{SeverityError, false, ""},
		{SeverityWarning, false, ""},
	} {
		if got := (deferOutcome{out[i].Severity, out[i].Deferred, out[i].PoamRef}); got != w {
			t.Errorf("%s at %s: got %+v, want %+v", in[i].Code, in[i].Path, got, w)
		}
	}
	got := codesOf(Bundle(b), "lifecycle.poam_defers_core")
	if len(got) != 2 || !isErrorAt(got[0], "lifecycle.poam_defers_core", "authoredArchitecture.poamItems[3].defers[0].code") {
		t.Fatalf("poam_defers_core: got %+v", got)
	}
}

// deferOutcome is the severity, deferral and POA&M item a diagnostic is expected to end with.
type deferOutcome struct {
	severity Severity
	deferred bool
	ref      string
}
