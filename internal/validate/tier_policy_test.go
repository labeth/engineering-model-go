// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// tierBundle is a valid model at the tier, with the bundles, or without
// lifecycle when tier is empty.
func tierBundle(tier string, bundles ...string) model.Bundle {
	b := behaviorTestBundle(nil, nil)
	if tier != "" {
		b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier, Bundles: bundles}
		if model.TierRank(tier) >= model.TierRank(model.TierMVP) {
			b.Manifest.RequirementLinks = model.RequirementLinksStrict
		}
	}
	return b
}

// severityAt returns the severity and deferred flag the policy gives one
// diagnostic of code at path in a model at the tier.
func severityAt(t *testing.T, b model.Bundle, code, path string, in Severity) Diagnostic {
	t.Helper()
	out := ApplyTierPolicy(b, []Diagnostic{{Code: code, Severity: in, Message: "m", Path: path}})
	if len(out) != 1 {
		t.Fatalf("ApplyTierPolicy returned %d diagnostics, want 1", len(out))
	}
	return out[0]
}

// The policy gives each tier-keyed code its severity at the project tier:
// deferred (a warning with deferred set), warning or error, with the tier
// it is required from; without lifecycle nothing changes.
// TRLC-LINKS: REQ-EMG-125
func TestApplyTierPolicySeverityByTier(t *testing.T) {
	cases := []struct {
		code             string
		tiers            [5]string // poc..operational: D, W or E
		requiredFrom     string
		bundles          []string
		emittedSeverity  Severity
		withoutLifecycle Severity
	}{
		{"requirement.orphan", [5]string{"D", "D", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
		{"verification.failed", [5]string{"D", "D", "D", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
		{"behavior.event_unused", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
		{"behavior.machine_deadlock", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
		{"behavior.transition_nondeterministic", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
		{"behavior.transition_no_trigger", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
		{"behavior.invariant_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
		{"behavior.state_unreachable", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
		{"code.missing_trlc_link", [5]string{"W", "W", "W", "E", "E"}, "product", nil, SeverityError, SeverityError},
		{"composition.dependency_tier_unknown", [5]string{"W", "W", "W", "W", "W"}, "", nil, SeverityWarning, SeverityWarning},
		{"composition.dependency_tier_unknown", [5]string{"W", "W", "W", "E", "E"}, "product", []string{"safety"}, SeverityWarning, SeverityWarning},
		{"lifecycle.aviation_profile_missing", [5]string{"D", "D", "E", "E", "E"}, "mvp", []string{"safety"}, SeverityWarning, SeverityWarning},
	}
	for _, c := range cases {
		for i, tier := range model.Tiers {
			got := severityAt(t, tierBundle(tier, c.bundles...), c.code, "p", c.emittedSeverity)
			want := map[string]Diagnostic{
				"D": {Severity: SeverityWarning, Deferred: true},
				"W": {Severity: SeverityWarning},
				"E": {Severity: SeverityError},
			}[c.tiers[i]]
			if got.Severity != want.Severity || got.Deferred != want.Deferred || got.RequiredFrom != c.requiredFrom {
				t.Errorf("%s at %s %v: got %s deferred=%v requiredFrom=%q, want %s deferred=%v requiredFrom=%q",
					c.code, tier, c.bundles, got.Severity, got.Deferred, got.RequiredFrom, want.Severity, want.Deferred, c.requiredFrom)
			}
		}
		if got := severityAt(t, tierBundle(""), c.code, "p", c.emittedSeverity); got.Severity != c.withoutLifecycle || got.Deferred || got.RequiredFrom != "" {
			t.Errorf("%s without lifecycle: got %+v, want %s unchanged", c.code, got, c.withoutLifecycle)
		}
	}
}

// A code the table does not key keeps its severity at every tier, and the
// policy is idempotent: applying it to its own output changes nothing.
// TRLC-LINKS: REQ-EMG-125
func TestApplyTierPolicyKeepsUnkeyedCodesAndIsIdempotent(t *testing.T) {
	b := tierBundle("poc")
	in := []Diagnostic{
		{Code: "model.duplicate_id", Severity: SeverityError, Message: "x", Path: "a"},
		{Code: "requirement.invalid_applies_to", Severity: SeverityError, Message: "x", Path: "b"},
		{Code: "requirement.orphan", Severity: SeverityWarning, Message: "x", Path: "c"},
	}
	once := ApplyTierPolicy(b, in)
	if once[0].Severity != SeverityError || once[1].Severity != SeverityError || !once[2].Deferred {
		t.Fatalf("got %+v", once)
	}
	twice := ApplyTierPolicy(b, once)
	for i := range once {
		if once[i] != twice[i] {
			t.Fatalf("not idempotent at %d: %+v then %+v", i, once[i], twice[i])
		}
	}
	if in[2].Deferred {
		t.Fatal("ApplyTierPolicy modified its input")
	}
}

// lifecycle.raise makes a code an error regardless of tier and never
// lowers one; a family pattern family.* raises every code of the family.
// TRLC-LINKS: REQ-EMG-125
func TestTierPolicyRaise(t *testing.T) {
	b := tierBundle("poc")
	b.Manifest.Lifecycle.Raise = []string{"requirement.orphan", "behavior.*", "catalog.term_unreferenced"}
	for _, code := range []string{"requirement.orphan", "behavior.state_unreachable", "catalog.term_unreferenced"} {
		if got := severityAt(t, b, code, "p", SeverityWarning); got.Severity != SeverityError || got.Deferred {
			t.Errorf("%s raised at poc: got %+v, want a plain error", code, got)
		}
	}
	if got := severityAt(t, b, "lint.multiple_responses", "p", SeverityWarning); got.Severity != SeverityWarning {
		t.Errorf("unraised code changed: %+v", got)
	}
}

// Every row of the table is non-decreasing from poc to operational
// (deferred < warning < error), with the safety bundle's row no lower than
// the base row: a higher tier or an added bundle never lowers a severity.
// TRLC-LINKS: REQ-EMG-125
func TestTierPolicyMonotonic(t *testing.T) {
	rank := map[string]int{"deferred": 0, "warning": 1, "error": 2}
	rules := TierPolicy()
	if len(rules) < 20 {
		t.Fatalf("TierPolicy has %d rules", len(rules))
	}
	for _, r := range rules {
		for _, row := range [][5]string{r.Severities, r.SafetySeverities} {
			for i := 1; i < 5; i++ {
				if rank[row[i]] < rank[row[i-1]] {
					t.Errorf("%s decreases from %s to %s: %v", r.Code, model.Tiers[i-1], model.Tiers[i], row)
				}
			}
		}
		for i := 0; i < 5; i++ {
			if rank[r.SafetySeverities[i]] < rank[r.Severities[i]] {
				t.Errorf("%s: the safety row is lower than the base row at %s", r.Code, model.Tiers[i])
			}
		}
	}
}

// codeLiteral finds diagnostic code literals in Go source.
var codeLiteral = regexp.MustCompile(`"((?:model|code|composition|link|risk|requirement|verification|threat|behavior|contract|lifecycle|aviation|health|lint)\.[a-z_.]+)"`)

// emittedCodes returns every code literal in the module's non-test Go
// source.
func emittedCodes(t *testing.T) map[string]bool {
	t.Helper()
	codes := map[string]bool{}
	err := filepath.WalkDir(filepath.Join("..", ".."), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "examples" || d.Name() == ".git" || d.Name() == "generated") {
			return filepath.SkipDir
		}
		if d.IsDir() || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		for _, m := range codeLiteral.FindAllStringSubmatch(string(data), -1) {
			codes[m[1]] = true
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return codes
}

// The rank-0 core codes are a closed list: exactly these patterns, each
// matching at least one code the engine emits, none keyed by tier, and a
// core diagnostic keeps its severity at every tier. Every tier-keyed code
// is emitted by the engine.
// TRLC-LINKS: REQ-EMG-125
func TestCoreCodesAreClosedAndNeverTierKeyed(t *testing.T) {
	want := []string{
		"model.duplicate_id", "model.broken_reference", "model.invalid_*", "model.missing_*",
		"code.dangling_*", "composition.cycle", "composition.lock_*",
		"link.version_mismatch", "link.broken", "link.parent_unknown", "link.parent_missing", "link.child_unpublished",
		"link.review_unrecorded", "link.invalid_status", "contract.realizer_unpublished", "requirement.realizes_unknown",
		"risk.acceptance_unrecorded",
		"lifecycle.invalid_tier", "lifecycle.invalid_bundle", "lifecycle.tier_decision_mismatch",
		"lifecycle.requirement_links_conflict", "lifecycle.poam_defers_core",
		"lifecycle.missing", "lifecycle.bundles_missing", "lifecycle.tier_decision_missing",
		"model.requirement_links_missing", "model.unsupported_schema_version", "model.id_missing",
		"lint.mode_missing", "lint.mode_invalid", "lint.mode_guided_at_tier", "health.block_missing",
		"aviation.ai_actor_as_credit", "aviation.authority_state_unevidenced",
	}
	got := CoreCodes()
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("CoreCodes() = %v\nwant %v", got, want)
	}
	emitted := emittedCodes(t)
	for _, pattern := range got {
		matched := false
		for code := range emitted {
			if matchCodePattern(pattern, code) {
				matched = true
			}
		}
		if !matched && pattern != "composition.cycle" {
			t.Errorf("core pattern %s matches no emitted code", pattern)
		}
	}
	for _, r := range TierPolicy() {
		if IsCoreCode(r.Code) {
			t.Errorf("%s is both core and tier keyed", r.Code)
		}
		if !emitted[r.Code] {
			t.Errorf("tier-keyed code %s is not emitted by the engine", r.Code)
		}
	}
	for _, tier := range model.Tiers {
		if got := severityAt(t, tierBundle(tier), "model.invalid_group", "p", SeverityError); got.Severity != SeverityError || got.Deferred {
			t.Errorf("core code changed at %s: %+v", tier, got)
		}
	}
}

// A requirement link rule follows the effective requirementLinks mode: with
// requirementLinks strict it stays an error even at poc, where the table
// defers it in guided mode.
// TRLC-LINKS: REQ-EMG-127
func TestTierPolicyRequirementLinkRows(t *testing.T) {
	guided := tierBundle("poc")
	if got := severityAt(t, guided, "requirement.no_parent", "p", SeverityWarning); !got.Deferred {
		t.Errorf("guided poc: got %+v, want deferred", got)
	}
	strict := tierBundle("poc")
	strict.Manifest.RequirementLinks = "strict"
	if got := severityAt(t, strict, "requirement.no_parent", "p", SeverityError); got.Severity != SeverityError || got.Deferred {
		t.Errorf("strict poc: got %+v, want error", got)
	}
	if got := severityAt(t, tierBundle("mvp"), "link.aou_unverified", "p", SeverityError); got.Severity != SeverityError {
		t.Errorf("mvp: got %+v, want error", got)
	}
}

// TierPolicyMarkdown renders the table the README "Life-cycle tier" section
// shows; the README copy is current.
// TRLC-LINKS: REQ-EMG-125
func TestTierPolicyMarkdownIsInReadme(t *testing.T) {
	table := TierPolicyMarkdown()
	if !strings.Contains(table, "| `requirement.orphan` | D | D | W | E | E |") {
		t.Fatalf("table lacks the requirement.orphan row:\n%s", table)
	}
	readme, err := os.ReadFile(filepath.Join("..", "..", "README.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(readme), table) {
		t.Fatalf("README.md does not contain the current TierPolicyMarkdown() table:\n%s", table)
	}
}

// The four advisory machine codes follow the hygiene row end to end: a model
// with a deadlocked, nondeterministic, trigger-less and unlinked machine
// reports each as deferred at poc and prototype, a warning at mvp and
// product and an error at operational, while a structural machine error
// (an unknown event) stays a plain error at poc and the tier policy does not
// key it.
// Mutation: dropping a code from the table leaves it a warning at operational;
// keying a structural code defers it at poc.
// TRLC-LINKS: REQ-EMG-140, REQ-EMG-138
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestMachineAdvisoryCodesFollowTheHygieneRowEndToEnd(t *testing.T) {
	advisory := []string{"behavior.machine_deadlock", "behavior.transition_nondeterministic", "behavior.transition_no_trigger", "behavior.invariant_unlinked"}
	build := func(tier string) model.Bundle {
		b := tierBundle(tier)
		m := baseTestMachine()
		m.States[2].Final = false // STATE-STOP deadlocks
		m.Transitions = append(m.Transitions,
			model.MachineTransition{ID: "TR-DUP", From: "STATE-IDLE", To: "STATE-RUN", On: "EVT-START"}, // same event as TR-START, no guards
			model.MachineTransition{ID: "TR-AUTO", From: "STATE-RUN", To: "STATE-IDLE"})                 // no event, no guard
		m.Invariants = []model.MachineInvariant{{ID: "INV-A", Expr: "count >= 0"}} // no requirement
		b.Architecture.AuthoredArchitecture.Machines = []model.Machine{m}
		b.Architecture.AuthoredArchitecture.Machines[0].Transitions[1].On = "EVT-NOPE" // a structural error
		return b
	}
	want := map[string][5]string{}
	for _, code := range advisory {
		want[code] = [5]string{"D", "D", "W", "W", "E"}
	}
	for i, tier := range model.Tiers {
		b := build(tier)
		diags := ApplyTierPolicy(b, Bundle(b))
		for _, code := range advisory {
			found := false
			for _, d := range diags {
				if d.Code != code {
					continue
				}
				found = true
				gotLetter := map[bool]string{true: "D", false: map[Severity]string{SeverityWarning: "W", SeverityError: "E"}[d.Severity]}[d.Deferred]
				if gotLetter != want[code][i] {
					t.Errorf("%s at %s: got %s (%+v), want %s", code, tier, gotLetter, d, want[code][i])
				}
			}
			if !found {
				t.Errorf("%s not reported at %s", code, tier)
			}
		}
		structural := 0
		for _, d := range diags {
			if d.Code == "behavior.transition_unknown_event" {
				structural++
				if d.Severity != SeverityError || d.Deferred {
					t.Errorf("structural machine error changed at %s: %+v", tier, d)
				}
			}
		}
		if structural != 1 {
			t.Errorf("want one structural machine error at %s, got %d", tier, structural)
		}
	}
	if TierKeyed("behavior.transition_unknown_event") || TierKeyed("behavior.expression_type") || TierKeyed("behavior.final_state_has_transitions") {
		t.Error("structural machine codes must not be tier keyed")
	}
	// Without lifecycle the advisory codes keep their authored warning.
	plain := build("")
	for _, d := range ApplyTierPolicy(plain, Bundle(plain)) {
		for _, code := range advisory {
			if d.Code == code && (d.Severity != SeverityWarning || d.Deferred || d.RequiredFrom != "") {
				t.Errorf("%s without lifecycle changed: %+v", code, d)
			}
		}
	}
}
