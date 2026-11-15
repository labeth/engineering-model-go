// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"os"
	"path/filepath"
	"regexp"
	"slices"
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

// probe is a diagnostic of the code and severity at the path, to run through the policy.
func probe(code, path string, in Severity) Diagnostic {
	return Diagnostic{Code: code, Severity: in, Message: "m", Path: path}
}

// severityAt returns the severity and deferred flag the policy gives one
// diagnostic of code at path in a model at the tier.
func severityAt(t *testing.T, b model.Bundle, in Diagnostic) Diagnostic {
	t.Helper()
	out := ApplyTierPolicy(b, []Diagnostic{in})
	if len(out) != 1 {
		t.Fatalf("ApplyTierPolicy returned %d diagnostics, want 1", len(out))
	}
	return out[0]
}

// tierSeverityCase is one tier-keyed code with its expected severity per tier.
type tierSeverityCase struct {
	code             string
	tiers            [5]string // poc..operational: D, W or E
	requiredFrom     string
	bundles          []string
	emittedSeverity  Severity
	withoutLifecycle Severity
}

// tierSeverityCases is the expected severity of each tier-keyed code.
var tierSeverityCases = []tierSeverityCase{
	{"requirement.orphan", [5]string{"D", "D", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
	{"verification.failed", [5]string{"D", "D", "D", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.event_unused", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.machine_deadlock", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.transition_nondeterministic", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.transition_no_trigger", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.invariant_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.state_unreachable", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.missing_trlc_link", [5]string{"W", "W", "W", "E", "E"}, "product", nil, SeverityError, SeverityError},
	{"code.owner_unresolved", [5]string{"W", "W", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
	{"code.model_link_kind_mismatch", [5]string{"W", "W", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
	{"code.trace_unattached", [5]string{"W", "W", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
	{"behavior.requirement_unmodeled", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.machine_unverified", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.state_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.transition_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"behavior.event_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"gemara.control_needs_review", [5]string{"D", "D", "D", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.unit_unlinked", [5]string{"D", "D", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
	{"code.machine_unlinked", [5]string{"D", "D", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
	{"code.interface_unlinked", [5]string{"D", "D", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
	{"code.state_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.transition_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.event_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.flow_unlinked", [5]string{"D", "D", "D", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.data_unlinked", [5]string{"D", "D", "D", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.exemption_unused", [5]string{"W", "W", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.state_candidate_unlinked", [5]string{"D", "D", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"code.state_suppression_unused", [5]string{"W", "W", "W", "W", "E"}, "operational", nil, SeverityWarning, SeverityWarning},
	{"aviation.tool.incomplete_credit_assessment", [5]string{"D", "D", "E", "E", "E"}, "mvp", nil, SeverityWarning, SeverityWarning},
	{"aviation.derived.incomplete_safety_feedback", [5]string{"D", "D", "E", "E", "E"}, "mvp", nil, SeverityWarning, SeverityWarning},
	{"catalog.term_unreferenced", [5]string{"W", "W", "W", "E", "E"}, "product", nil, SeverityWarning, SeverityWarning},
	{"lifecycle.aviation_profile_missing", [5]string{"D", "D", "E", "E", "E"}, "mvp", []string{"safety"}, SeverityWarning, SeverityWarning},
}

// The policy gives each tier-keyed code its severity at the project tier:
// deferred (a warning with deferred set), warning or error, with the tier
// it is required from; without lifecycle nothing changes.
// TRLC-LINKS: REQ-EMG-125
func TestApplyTierPolicySeverityByTier(t *testing.T) {
	for _, c := range tierSeverityCases {
		t.Run(c.code, func(t *testing.T) {
			requireSeverityAtEveryTier(t, c)
			requireUnchangedWithoutLifecycle(t, c)
		})
	}
}

// requireSeverityAtEveryTier checks the severity, deferral and required-from
// tier of the code at each tier.
// TRLC-LINKS: REQ-EMG-125
func requireSeverityAtEveryTier(t *testing.T, c tierSeverityCase) {
	t.Helper()
	for i, tier := range model.Tiers {
		got := severityAt(t, tierBundle(tier, c.bundles...), probe(c.code, "p", c.emittedSeverity))
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
}

// requireUnchangedWithoutLifecycle checks that a model without lifecycle keeps
// the emitted severity.
// TRLC-LINKS: REQ-EMG-125
func requireUnchangedWithoutLifecycle(t *testing.T, c tierSeverityCase) {
	t.Helper()
	if got := severityAt(t, tierBundle(""), probe(c.code, "p", c.emittedSeverity)); got.Severity != c.withoutLifecycle || got.Deferred || got.RequiredFrom != "" {
		t.Errorf("%s without lifecycle: got %+v, want %s unchanged", c.code, got, c.withoutLifecycle)
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
		if got := severityAt(t, b, probe(code, "p", SeverityWarning)); got.Severity != SeverityError || got.Deferred {
			t.Errorf("%s raised at poc: got %+v, want a plain error", code, got)
		}
	}
	if got := severityAt(t, b, probe("lint.multiple_responses", "p", SeverityWarning)); got.Severity != SeverityWarning {
		t.Errorf("unraised code changed: %+v", got)
	}
}

// Every row of the table is non-decreasing from poc to operational
// (deferred < warning < error), with the safety bundle's row no lower than
// the base row: a higher tier or an added bundle never lowers a severity.
// TRLC-LINKS: REQ-EMG-125
func TestTierPolicyMonotonic(t *testing.T) {
	rules := TierPolicy()
	if len(rules) < 20 {
		t.Fatalf("TierPolicy has %d rules", len(rules))
	}
	for _, r := range rules {
		assertRowNonDecreasing(t, r.Code, r.Severities)
		assertRowNonDecreasing(t, r.Code, r.SafetySeverities)
		assertSafetyNotLower(t, r)
	}
}

// severityRank orders the severities of a tier row.
var severityRank = map[string]int{"deferred": 0, "warning": 1, "error": 2}

// assertRowNonDecreasing checks that a row never lowers its severity from one tier to the next.
func assertRowNonDecreasing(t *testing.T, code string, row [5]string) {
	t.Helper()
	for i := 1; i < 5; i++ {
		if severityRank[row[i]] < severityRank[row[i-1]] {
			t.Errorf("%s decreases from %s to %s: %v", code, model.Tiers[i-1], model.Tiers[i], row)
		}
	}
}

// assertSafetyNotLower checks that the safety row is no lower than the base row at any tier.
func assertSafetyNotLower(t *testing.T, r TierRule) {
	t.Helper()
	for i := 0; i < 5; i++ {
		if severityRank[r.SafetySeverities[i]] < severityRank[r.Severities[i]] {
			t.Errorf("%s: the safety row is lower than the base row at %s", r.Code, model.Tiers[i])
		}
	}
}

// codeLiteral finds diagnostic code literals in Go source.
var codeLiteral = regexp.MustCompile(`"((?:model|code|composition|link|risk|requirement|verification|threat|behavior|contract|lifecycle|aviation|health|lint|gemara|catalog|architecture)\.[a-z_.]+)"`)

// emittedCodes returns every code literal in the module's non-test Go
// source.
func emittedCodes(t *testing.T) map[string]bool {
	t.Helper()
	codes := map[string]bool{}
	err := filepath.WalkDir(filepath.Join("..", ".."), func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return skipUnscannedDir(d.Name())
		}
		if !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
			return nil
		}
		return addCodeLiterals(codes, path)
	})
	if err != nil {
		t.Fatal(err)
	}
	return codes
}

// skipUnscannedDir is filepath.SkipDir for the directories whose source is not scanned for codes.
func skipUnscannedDir(name string) error {
	if name == "examples" || name == ".git" || name == "generated" {
		return filepath.SkipDir
	}
	return nil
}

// addCodeLiterals adds the diagnostic code literals of a Go file.
func addCodeLiterals(codes map[string]bool, path string) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	for _, m := range codeLiteral.FindAllStringSubmatch(string(data), -1) {
		codes[m[1]] = true
	}
	return nil
}

// The rank-0 core codes are a closed list: exactly these patterns, each
// matching at least one code the engine emits, none keyed by tier, and a
// core diagnostic keeps its severity at every tier. Every tier-keyed code
// is emitted by the engine.
// TRLC-LINKS: REQ-EMG-125
func TestCoreCodesAreClosedAndNeverTierKeyed(t *testing.T) {
	got := CoreCodes()
	if want := wantCoreCodes(); strings.Join(got, ",") != strings.Join(want, ",") {
		t.Fatalf("CoreCodes() = %v\nwant %v", got, want)
	}
	emitted := emittedCodes(t)
	assertCorePatternsEmitted(t, got, emitted)
	assertTierKeyedCodesEmitted(t, emitted)
	for _, tier := range model.Tiers {
		if got := severityAt(t, tierBundle(tier), probe("model.invalid_group", "p", SeverityError)); got.Severity != SeverityError || got.Deferred {
			t.Errorf("core code changed at %s: %+v", tier, got)
		}
	}
}

// assertCorePatternsEmitted checks that each core pattern matches a code the engine emits.
func assertCorePatternsEmitted(t *testing.T, patterns []string, emitted map[string]bool) {
	t.Helper()
	for _, pattern := range patterns {
		if !matchesEmittedCode(pattern, emitted) && pattern != "composition.cycle" {
			t.Errorf("core pattern %s matches no emitted code", pattern)
		}
	}
}

// assertTierKeyedCodesEmitted checks that every tier-keyed code is emitted and none is core.
func assertTierKeyedCodesEmitted(t *testing.T, emitted map[string]bool) {
	t.Helper()
	for _, r := range TierPolicy() {
		if IsCoreCode(r.Code) {
			t.Errorf("%s is both core and tier keyed", r.Code)
		}
		if !emitted[r.Code] {
			t.Errorf("tier-keyed code %s is not emitted by the engine", r.Code)
		}
	}
}

// matchesEmittedCode reports whether the core pattern matches some emitted code.
func matchesEmittedCode(pattern string, emitted map[string]bool) bool {
	for code := range emitted {
		if matchCodePattern(pattern, code) {
			return true
		}
	}
	return false
}

// wantCoreCodes is the closed list of rank-0 core code patterns.
func wantCoreCodes() []string {
	return []string{
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
		"composition.dependency_tier_unknown",
	}
}

// composition.dependency_tier_unknown is a core error (ADR-EMG-073 item 4):
// an error at every tier, with and without the safety bundle, never tier
// keyed and never deferred by a POA&M item.
// Mutation: moving it back to tierTable, or out of coreCodes, makes it a
// warning at a low tier.
// TRLC-LINKS: REQ-EMG-183
func TestDependencyTierUnknownIsCore(t *testing.T) {
	const code = "composition.dependency_tier_unknown"
	if !IsCoreCode(code) || TierKeyed(code) {
		t.Fatalf("%s: core=%v tierKeyed=%v, want core and untiered", code, IsCoreCode(code), TierKeyed(code))
	}
	for _, tier := range model.Tiers {
		requireCoreError(t, code, tierBundle(tier))
		requireCoreError(t, code, tierBundle(tier, model.BundleSafety))
	}
}

// requireCoreError checks that a core code is a plain error in the model.
// TRLC-LINKS: REQ-EMG-183
func requireCoreError(t *testing.T, code string, b model.Bundle) {
	t.Helper()
	if got := severityAt(t, b, probe(code, "p", SeverityError)); got.Severity != SeverityError || got.Deferred {
		t.Errorf("%s at %v: got %+v, want a plain error", code, b.Manifest.Lifecycle, got)
	}
}

// A requirement link rule follows the effective requirementLinks mode: with
// requirementLinks strict it stays an error even at poc, where the table
// defers it in guided mode.
// TRLC-LINKS: REQ-EMG-127
func TestTierPolicyRequirementLinkRows(t *testing.T) {
	guided := tierBundle("poc")
	if got := severityAt(t, guided, probe("requirement.no_parent", "p", SeverityWarning)); !got.Deferred {
		t.Errorf("guided poc: got %+v, want deferred", got)
	}
	strict := tierBundle("poc")
	strict.Manifest.RequirementLinks = "strict"
	if got := severityAt(t, strict, probe("requirement.no_parent", "p", SeverityError)); got.Severity != SeverityError || got.Deferred {
		t.Errorf("strict poc: got %+v, want error", got)
	}
	if got := severityAt(t, tierBundle("mvp"), probe("link.aou_unverified", "p", SeverityError)); got.Severity != SeverityError {
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
	for i, tier := range model.Tiers {
		b := machineAdvisoryBundle(tier)
		diags := ApplyTierPolicy(b, Bundle(b))
		want := [5]string{"D", "D", "W", "W", "E"}[i]
		for _, code := range advisoryMachineCodes {
			assertAdvisoryLetter(t, diags, advisoryProbe{code, tier, want})
		}
		assertOneStructuralError(t, diags, tier)
	}
	assertStructuralCodesUntiered(t)
	assertPlainKeepsAdvisoryWarnings(t)
}

// assertStructuralCodesUntiered checks that the structural machine codes are not tier keyed.
func assertStructuralCodesUntiered(t *testing.T) {
	t.Helper()
	assertHealthCodesUntiered(t, "behavior.transition_unknown_event", "behavior.expression_type", "behavior.final_state_has_transitions")
}

// assertPlainKeepsAdvisoryWarnings checks that without lifecycle the advisory codes keep their authored warning.
func assertPlainKeepsAdvisoryWarnings(t *testing.T) {
	t.Helper()
	plain := machineAdvisoryBundle("")
	for _, d := range ApplyTierPolicy(plain, Bundle(plain)) {
		if isAdvisoryMachineCode(d.Code) && !keepsAuthoredWarning(d) {
			t.Errorf("%s without lifecycle changed: %+v", d.Code, d)
		}
	}
}

// advisoryProbe is an advisory code with the letter (D, W or E) expected at a tier.
type advisoryProbe struct{ code, tier, want string }

// advisoryMachineCodes are the advisory machine codes the hygiene row keys.
var advisoryMachineCodes = []string{"behavior.machine_deadlock", "behavior.transition_nondeterministic", "behavior.transition_no_trigger", "behavior.invariant_unlinked"}

// isAdvisoryMachineCode reports whether the code is one of the advisory machine codes.
func isAdvisoryMachineCode(code string) bool {
	return slices.Contains(advisoryMachineCodes, code)
}

// keepsAuthoredWarning reports whether a diagnostic is still the plain warning it was authored as.
func keepsAuthoredWarning(d Diagnostic) bool {
	return d.Severity == SeverityWarning && !d.Deferred && d.RequiredFrom == ""
}

// machineAdvisoryBundle is a model at the tier with a deadlocked,
// nondeterministic, trigger-less and unlinked machine and one structural error
// (an unknown event).
func machineAdvisoryBundle(tier string) model.Bundle {
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

// assertAdvisoryLetter checks that the code is reported at the tier as D (deferred), W or E.
func assertAdvisoryLetter(t *testing.T, diags []Diagnostic, p advisoryProbe) {
	t.Helper()
	found := diagnosticsWithCode(diags, p.code)
	if len(found) == 0 {
		t.Errorf("%s not reported at %s", p.code, p.tier)
	}
	for _, d := range found {
		if got := severityLetter(d); got != p.want {
			t.Errorf("%s at %s: got %s (%+v), want %s", p.code, p.tier, got, d, p.want)
		}
	}
}

// severityLetter is D for a deferred diagnostic, else W or E by severity.
func severityLetter(d Diagnostic) string {
	if d.Deferred {
		return "D"
	}
	return map[Severity]string{SeverityWarning: "W", SeverityError: "E"}[d.Severity]
}

// assertOneStructuralError checks that the structural machine error stays one plain error at the tier.
func assertOneStructuralError(t *testing.T, diags []Diagnostic, tier string) {
	t.Helper()
	structural := diagnosticsWithCode(diags, "behavior.transition_unknown_event")
	if len(structural) != 1 {
		t.Errorf("want one structural machine error at %s, got %d", tier, len(structural))
	}
	for _, d := range structural {
		if d.Severity != SeverityError || d.Deferred {
			t.Errorf("structural machine error changed at %s: %+v", tier, d)
		}
	}
}

// Each consistency check (ADR-EMG-069) names its remedy: the skill that fixes
// it. Any other code has none.
// Mutation: dropping a code from consistencyRemedies, or naming another skill,
// fails its row.
// TRLC-LINKS: REQ-EMG-125
func TestConsistencyRemedies(t *testing.T) {
	cases := map[string]string{
		"code.owner_unresolved":                     "ef-code-linking",
		"code.model_link_kind_mismatch":             "ef-code-linking",
		"behavior.requirement_unmodeled":            "ef-behavior",
		"behavior.machine_unverified":               "ef-formal-behavior",
		"behavior.state_unlinked":                   "ef-behavior",
		"behavior.transition_unlinked":              "ef-behavior",
		"behavior.event_unlinked":                   "ef-behavior",
		"code.unit_unlinked":                        "ef-code-linking",
		"code.machine_unlinked":                     "ef-code-linking",
		"code.interface_unlinked":                   "ef-code-linking",
		"code.state_unlinked":                       "ef-code-linking",
		"code.transition_unlinked":                  "ef-code-linking",
		"code.event_unlinked":                       "ef-code-linking",
		"code.flow_unlinked":                        "ef-code-linking",
		"code.data_unlinked":                        "ef-code-linking",
		"composition.dependency_tier_unknown":       "owner of the dependency",
		"code.exemption_unused":                     "ef-tier-up",
		"code.state_candidate_unlinked":             "ef-behavior",
		"code.state_suppression_unreasoned":         "ENGMODEL-NOT-A-STATE",
		"code.state_suppression_unused":             "ENGMODEL-NOT-A-STATE",
		"code.exemption_unjustified":                "ef-tier-up",
		"model.invalid_code_exemption":              "ef-tier-up",
		"requirement.orphan":                        "",
		"architecture.dependency_without_interface": "name the interface the dependency goes through",
		"architecture.interface_without_data":       "name the data type sent over the interface",
	}
	for code, skill := range cases {
		got := ConsistencyRemedy(code)
		if (skill == "") != (got == "") || !strings.Contains(got, skill) {
			t.Errorf("%s: remedy %q, want one naming %q", code, got, skill)
		}
	}
}

// code.exemption_unjustified is an untiered error like
// health.waiver_unjustified and model.invalid_code_exemption is a core code:
// both stay errors at every tier and no POA&M item defers them; the exemption
// decision rule is the health waiver rule.
// Mutation: keying either code in the tier table, or listing it as deferrable,
// lowers it at a low tier; accepting a proposed decision fails the rule.
// TRLC-LINKS: REQ-EMG-176
func TestCodeExemptionCodesAreUntieredErrors(t *testing.T) {
	if TierKeyed("code.exemption_unjustified") || TierKeyed("model.invalid_code_exemption") {
		t.Fatal("the exemption error codes must be untiered")
	}
	if !IsCoreCode("model.invalid_code_exemption") || IsCoreCode("code.exemption_unjustified") {
		t.Fatal("model.invalid_code_exemption is core, code.exemption_unjustified is untiered but not core")
	}
	for _, tier := range model.Tiers {
		requireExemptionErrorsAt(t, tier)
	}
	requireExemptionDecisionProblems(t)
}

// requireExemptionErrorsAt checks that both exemption codes stay undeferred
// errors at the tier.
// TRLC-LINKS: REQ-EMG-176
func requireExemptionErrorsAt(t *testing.T, tier string) {
	t.Helper()
	for _, code := range []string{"code.exemption_unjustified", "model.invalid_code_exemption"} {
		if got := severityAt(t, tierBundle(tier), probe(code, "p", SeverityError)); got.Severity != SeverityError || got.Deferred {
			t.Errorf("%s at %s: %+v", code, tier, got)
		}
	}
}

// requireExemptionDecisionProblems checks the exemption decision rule.
// TRLC-LINKS: REQ-EMG-176
func requireExemptionDecisionProblems(t *testing.T) {
	t.Helper()
	b := tierBundle("poc")
	b.Architecture.Decisions = []model.Decision{{ID: "DEC-OK", Status: "accepted"}, {ID: "DEC-NO", Status: "proposed"}}
	for decision, want := range map[string]string{"DEC-OK": "", "DEC-NO": `has status "proposed", not accepted`, "DEC-GONE": "does not exist", " ": "has no decision"} {
		if got := CodeExemptionDecisionProblem(b, decision); got != want {
			t.Errorf("decision %q: problem %q, want %q", decision, got, want)
		}
	}
}

// The state-like code discovery (ADR-EMG-070): the candidate follows the
// hygiene row (deferred below mvp, a warning at mvp and product, an error at
// operational), the unused suppression the row WWWWE, and the unreasoned
// suppression is an untiered error that no tier lowers.
// Mutation: keying code.state_suppression_unreasoned in the tier table, or
// moving the candidate to rowDebtToProduct, fails a tier.
// TRLC-LINKS: REQ-EMG-188
func TestStateDiscoveryCodesFollowTheirTierRows(t *testing.T) {
	for code, want := range stateDiscoveryRows {
		t.Run(code, func(t *testing.T) { assertTierRow(t, code, want) })
	}
	t.Run("unreasoned suppression is untiered", func(t *testing.T) { assertUnreasonedIsUntiered(t) })
}

// stateRow is the severity and deferral of a code at each of the five tiers.
type stateRow struct {
	severity [5]Severity
	deferred [5]bool
}

var stateDiscoveryRows = map[string]stateRow{
	"code.state_candidate_unlinked": {
		[5]Severity{SeverityWarning, SeverityWarning, SeverityWarning, SeverityWarning, SeverityError},
		[5]bool{true, true, false, false, false},
	},
	"code.state_suppression_unused": {
		[5]Severity{SeverityWarning, SeverityWarning, SeverityWarning, SeverityWarning, SeverityError},
		[5]bool{},
	},
}

// assertTierRow checks one code at every tier against its expected row.
// TRLC-LINKS: REQ-EMG-188
func assertTierRow(t *testing.T, code string, want stateRow) {
	t.Helper()
	for i, tier := range model.Tiers {
		got := severityAt(t, tierBundle(tier), probe(code, "p", SeverityWarning))
		if got.Severity != want.severity[i] || got.Deferred != want.deferred[i] {
			t.Errorf("%s at %s: got %s deferred=%v", code, tier, got.Severity, got.Deferred)
		}
	}
}

// assertUnreasonedIsUntiered checks that no tier lowers or defers the unreasoned suppression.
// TRLC-LINKS: REQ-EMG-188
func assertUnreasonedIsUntiered(t *testing.T) {
	t.Helper()
	if TierKeyed("code.state_suppression_unreasoned") {
		t.Fatal("code.state_suppression_unreasoned must be untiered")
	}
	for _, tier := range model.Tiers {
		if got := severityAt(t, tierBundle(tier), probe("code.state_suppression_unreasoned", "p", SeverityError)); got.Severity != SeverityError || got.Deferred {
			t.Errorf("unreasoned suppression at %s: %+v", tier, got)
		}
	}
}
