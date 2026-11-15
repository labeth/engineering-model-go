// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/internal/model"
)

// decisionLine finds a labelled line (Tier: or Bundles:) of a decision text.
var decisionLine = regexp.MustCompile(`(?im)^\s*(tier|bundles)\s*:\s*(.*?)\s*$`)

// lifecycleDiagnostics checks engmod.yml lifecycle and the rules it adds. A
// manifest without lifecycle is lifecycle.missing (ADR-EMG-061): the model
// has no tier. With one: tier and bundles (lifecycle.invalid_tier,
// lifecycle.invalid_bundle, which the CUE schema already rejects on load),
// the bundles that must be written (lifecycle.bundles_missing), the tier
// decision that must be named (lifecycle.tier_decision_missing) and agree
// (lifecycle.tier_decision_mismatch), requirementLinks guided at mvp or
// above (lifecycle.requirement_links_conflict), the safety bundle's aviation
// profile and software level, requirement status (lifecycle.requirement_*)
// and POA&M defers naming a rank-0 or non-deferrable code
// (lifecycle.poam_defers_core) or an item past its dueDate at asOf
// (lifecycle.poam_overdue). The POA&M checks run first, so a model without
// lifecycle still reports a defer of lifecycle.missing.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func lifecycleDiagnostics(b model.Bundle, asOf time.Time) []Diagnostic {
	l := b.Manifest.Lifecycle
	diags := poamDiagnostics(b, asOf)
	if l == nil {
		return append(diags, Diagnostic{Code: "lifecycle.missing", Severity: SeverityError, Path: "lifecycle",
			Message: "engmod.yml has no lifecycle block: the model has no tier, so no tier rule applies and the project is not at any stage"})
	}
	diags = append(diags, lifecycleTierDiagnostics(b, *l)...)
	diags = append(diags, lifecycleSafetyDiagnostics(b, *l)...)
	return append(diags, lifecycleRequirementDiagnostics(b)...)
}

// lifecycleTierDiagnostics checks the tier, the bundles, the tier decision and
// the requirement links mode of a lifecycle block.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func lifecycleTierDiagnostics(b model.Bundle, l model.ManifestLifecycle) []Diagnostic {
	var s diagSink
	rank := model.TierRank(l.Tier)
	if rank == 0 {
		s.fail("lifecycle.invalid_tier", "lifecycle.tier", fmt.Sprintf("lifecycle.tier %q must be one of %s", l.Tier, strings.Join(model.Tiers, ", ")))
	}
	s.lifecycleBundles(l)
	s.lifecycleTierDecision(b, l)
	if rank >= model.TierRank(model.TierMVP) && strings.EqualFold(strings.TrimSpace(b.Manifest.RequirementLinks), model.RequirementLinksGuided) {
		s.fail("lifecycle.requirement_links_conflict", "requirementLinks",
			fmt.Sprintf("requirementLinks guided at tier %q: the tier requires strict requirement links from mvp; set requirementLinks: strict", strings.ToLower(l.Tier)))
	}
	return s.diags
}

// lifecycleBundles requires the lifecycle bundles to be written and known.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s *diagSink) lifecycleBundles(l model.ManifestLifecycle) {
	for i, bundle := range l.Bundles {
		if !containsTrimmed(model.Bundles, strings.ToLower(bundle)) {
			s.fail("lifecycle.invalid_bundle", indexedPath("lifecycle.bundles", i), fmt.Sprintf("lifecycle bundle %q must be one of %s", bundle, strings.Join(model.Bundles, ", ")))
		}
	}
	if len(l.Bundles) == 0 {
		s.fail("lifecycle.bundles_missing", "lifecycle.bundles", "lifecycle.bundles is empty: write the assurance bundles (foundation, safety, security) explicitly; there is no default")
	}
}

// lifecycleTierDecision requires the tier decision to be named and to agree with the lifecycle.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s *diagSink) lifecycleTierDecision(b model.Bundle, l model.ManifestLifecycle) {
	if ref := strings.TrimSpace(l.TierDecision); ref == "" {
		s.fail("lifecycle.tier_decision_missing", "lifecycle.tierDecision", "lifecycle.tierDecision is empty: name the accepted DEC-TIER-* decision that records the tier")
	} else if problem := tierDecisionProblem(b, l, ref); problem != "" {
		s.fail("lifecycle.tier_decision_mismatch", "lifecycle.tierDecision", fmt.Sprintf("lifecycle.tierDecision %q: %s", ref, problem))
	}
}

// lifecycleSafetyDiagnostics warns when the safety bundle lacks the aviation profile or software level.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func lifecycleSafetyDiagnostics(b model.Bundle, l model.ManifestLifecycle) []Diagnostic {
	if !l.HasBundle(model.BundleSafety) {
		return nil
	}
	var s diagSink
	if b.Aviation == nil {
		s.warn("lifecycle.aviation_profile_missing", "documents.aviation",
			"the safety bundle needs the DO-178C aviation profile: add documents.aviation (model/aviation.yml) with the software level and lifecycle data")
	}
	if b.Aviation == nil || strings.TrimSpace(b.Aviation.Aviation.Profile.SoftwareLevel) == "" {
		s.warn("lifecycle.software_level_missing", "aviation.profile.softwareLevel",
			"the safety bundle needs aviation.profile.softwareLevel (A to E) from the system safety assessment")
	}
	return s.diags
}

// lifecycleRequirementDiagnostics warns about requirements that are not approved or not verified.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func lifecycleRequirementDiagnostics(b model.Bundle) []Diagnostic {
	var s diagSink
	for i, r := range b.Requirements.Requirements {
		if !strings.Contains(r.ID, "::") {
			s.requirementStatus(indexedPath("requirements", i)+".status", r)
		}
	}
	return s.diags
}

// requirementStatus warns about a draft or proposed requirement and about one not yet verified, rejected or retired.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (s *diagSink) requirementStatus(path string, r model.Requirement) {
	status := strings.ToLower(strings.TrimSpace(r.Status))
	shown := status
	if shown == "" {
		shown = "unset"
	}
	if status == "draft" || status == "proposed" {
		s.warn("lifecycle.requirement_unapproved", path, fmt.Sprintf("requirement %q has status %q; the tier requires approved requirements", r.ID, shown))
	}
	if !closedRequirementStatus(status) {
		s.warn("lifecycle.requirement_unverified", path, fmt.Sprintf("requirement %q has status %q, not verified; record verified once its verification passed (engws status-suggest)", r.ID, shown))
	}
}

// closedRequirementStatus reports whether a lowercase status is verified, rejected or retired.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func closedRequirementStatus(status string) bool {
	return status == "verified" || status == "rejected" || status == "retired"
}

// poamDiagnostics checks the POA&M items independently of the lifecycle
// block: a defers entry that names a rank-0 or non-deferrable code, or a
// family pattern that would cover one (lifecycle.poam_defers_core, an
// error), and an open item past its dueDate at asOf (lifecycle.poam_overdue:
// an error at tier product and operational, a warning below and without a
// tier). An overdue item defers nothing (ApplyTierPolicyAt). The zero asOf
// reports no overdue item.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func poamDiagnostics(b model.Bundle, asOf time.Time) []Diagnostic {
	var diags []Diagnostic
	overdue := poamOverdueSeverity(b)
	for i, p := range b.Architecture.AuthoredArchitecture.POAMItems {
		item := activePoamItem{index: i, item: p}
		diags = append(diags, poamDefersCoreDiagnostics(item)...)
		if poamOverdue(p, asOf) {
			diags = append(diags, poamOverdueDiagnostic(item, overdue, asOf))
		}
	}
	return diags
}

// poamOverdueSeverity is the severity of lifecycle.poam_overdue: an error at
// tier product and operational, a warning below and without a tier.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func poamOverdueSeverity(b model.Bundle) Severity {
	if _, rank, ok := EffectiveTier(b); ok && rank >= model.TierRank(model.TierProduct) {
		return SeverityError
	}
	return SeverityWarning
}

// poamDefersCoreDiagnostics reports each defers entry of an item that names
// or would cover a protected code (lifecycle.poam_defers_core).
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func poamDefersCoreDiagnostics(item activePoamItem) []Diagnostic {
	var diags []Diagnostic
	for j, d := range item.item.Defers {
		code := strings.TrimSpace(d.Code)
		if code == "" {
			continue
		}
		covered := deferCoverage(code)
		if covered == "" {
			continue
		}
		diags = append(diags, Diagnostic{Code: "lifecycle.poam_defers_core", Severity: SeverityError,
			Path:    fmt.Sprintf("authoredArchitecture.poamItems[%d].defers[%d].code", item.index, j),
			Message: fmt.Sprintf("POA&M item %q defers %q, which covers %s that holds at every tier and cannot be deferred; fix the finding instead", item.item.ID, code, covered)})
	}
	return diags
}

// poamOverdueDiagnostic is the lifecycle.poam_overdue diagnostic of an item.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func poamOverdueDiagnostic(item activePoamItem, severity Severity, asOf time.Time) Diagnostic {
	return Diagnostic{Code: "lifecycle.poam_overdue", Severity: severity,
		Path: fmt.Sprintf("authoredArchitecture.poamItems[%d].dueDate", item.index),
		Message: fmt.Sprintf("POA&M item %q is overdue: dueDate %q is before %s and the item is not completed; it defers nothing until it is completed or its dueDate is renewed by a recorded decision",
			item.item.ID, item.item.DueDate, asOf.UTC().Format(time.DateOnly))}
}

// UnusedDefers returns lifecycle.poam_defers_unused for each defers entry of
// an honoured POA&M item (open, milestone tier:<project tier>, not overdue)
// that matches none of diags: a stale positional path, a ref to a requirement
// that does not exist, or a finding that was fixed. It is a warning at every
// tier and an error at operational, where an entry that covers nothing is a
// defer waiting to hide the next finding at that path. Pass the merged list
// of every diagnostic source (Bundle, trace, health, verification): an entry
// is unused only against the complete list. Bundle itself reports the
// lifecycle.* entries, whose diagnostics it alone emits.
// TRLC-LINKS: REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func UnusedDefers(b model.Bundle, diags []Diagnostic) []Diagnostic {
	return unusedDefers(b, diags, deferScope{asOf: AsOf(), inScope: func(string) bool { return true }})
}

// deferScope is the date and the code patterns unusedDefers looks at.
type deferScope struct {
	asOf    time.Time
	inScope func(pattern string) bool
}

// unusedDefers is UnusedDefers at a date, limited to the entries whose code
// pattern in scope accepts.
// TRLC-LINKS: REQ-EMG-129
func unusedDefers(b model.Bundle, diags []Diagnostic, scope deferScope) []Diagnostic {
	tier, rank, ok := EffectiveTier(b)
	if !ok {
		return nil
	}
	run := unusedDeferRun{b: b, diags: diags, scope: scope, severity: SeverityWarning}
	if rank == model.TierRank(model.TierOperational) {
		run.severity = SeverityError
	}
	var out []Diagnostic
	for _, p := range activePoamItems(b, tier, scope.asOf) {
		out = append(out, run.item(p)...)
	}
	return out
}

// unusedDeferRun is the context unusedDefers applies to each honoured item.
type unusedDeferRun struct {
	b        model.Bundle
	diags    []Diagnostic
	scope    deferScope
	severity Severity
}

// item reports the in-scope defers entries of an item that match no
// diagnostic.
// TRLC-LINKS: REQ-EMG-129
func (r unusedDeferRun) item(p activePoamItem) []Diagnostic {
	var out []Diagnostic
	for j, d := range p.item.Defers {
		if r.scope.inScope(strings.TrimSpace(d.Code)) && !r.used(d) {
			out = append(out, unusedDeferDiagnostic(p, j, r.severity))
		}
	}
	return out
}

// used reports whether a defers entry matches some diagnostic.
// TRLC-LINKS: REQ-EMG-129
func (r unusedDeferRun) used(d model.POAMDefer) bool {
	for _, x := range r.diags {
		if deferMatches(r.b, d, x) {
			return true
		}
	}
	return false
}

// unusedDeferDiagnostic is the lifecycle.poam_defers_unused diagnostic of
// the j-th defers entry of an item.
// TRLC-LINKS: REQ-EMG-129
func unusedDeferDiagnostic(p activePoamItem, j int, severity Severity) Diagnostic {
	d := p.item.Defers[j]
	return Diagnostic{Code: "lifecycle.poam_defers_unused", Severity: severity,
		Path: fmt.Sprintf("authoredArchitecture.poamItems[%d].defers[%d]", p.index, j),
		Message: fmt.Sprintf("POA&M item %q defers %q%s but no diagnostic matches it: remove the entry, or name the finding by ref: <requirement ID> instead of a positional path",
			p.item.ID, d.Code, deferTarget(d))}
}

// deferTarget renders the path or ref of a defers entry for a message.
// TRLC-LINKS: REQ-EMG-129
func deferTarget(d model.POAMDefer) string {
	var parts string
	if p := strings.TrimSpace(d.Path); p != "" {
		parts += fmt.Sprintf(" at path %q", p)
	}
	if r := strings.TrimSpace(d.Ref); r != "" {
		parts += fmt.Sprintf(" for ref %q", r)
	}
	return parts
}

// deferCoverage says which protected code a defers code (exact or family.*)
// names or would cover: a rank-0 core pattern or a non-deferrable code. It
// returns "" for a code a POA&M item may defer, such as the exact
// health.complex_method or lifecycle.requirement_unverified. A wildcard is
// rejected when any protected code starts with its prefix.
// TRLC-LINKS: REQ-EMG-129
func deferCoverage(code string) string {
	if covered := coreCoverage(code); covered != "" {
		return covered
	}
	return nonDeferrableCoverage(code)
}

// coreCoverage names the rank-0 core pattern a defers code names or covers.
// TRLC-LINKS: REQ-EMG-129
func coreCoverage(code string) string {
	prefix, wildcard := strings.CutSuffix(code, "*")
	for _, p := range coreCodes {
		if matchCodePattern(p, code) || (wildcard && strings.HasPrefix(strings.TrimSuffix(p, "*"), prefix)) {
			return fmt.Sprintf("the core code %s", p)
		}
	}
	return ""
}

// nonDeferrableCoverage names the non-deferrable code a defers code names or
// covers.
// TRLC-LINKS: REQ-EMG-129
func nonDeferrableCoverage(code string) string {
	for _, c := range nonDeferrable {
		if matchCodePattern(code, c) {
			return fmt.Sprintf("the non-deferrable code %s", c)
		}
	}
	return ""
}

// tierDecisionProblem returns why the decision named by tierDecision does
// not record the lifecycle, or "" when it does: it must exist, be accepted,
// and its Tier: line (one tier, case-insensitive) and Bundles: line (comma
// separated; required, there is no default) must equal lifecycle.tier and the
// effective bundles.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func tierDecisionProblem(b model.Bundle, l model.ManifestLifecycle, ref string) string {
	dec := findDecision(b.Architecture.Decisions, ref)
	if dec == nil {
		return "names no decision of this model"
	}
	if !strings.EqualFold(strings.TrimSpace(dec.Status), "accepted") {
		return fmt.Sprintf("has status %q; the tier decision must be accepted", dec.Status)
	}
	rec := parseDecisionRecord(dec.Decision)
	if !rec.tierSeen {
		return "the decision text has no Tier: line"
	}
	if !rec.bundlesSeen || len(rec.bundles) == 0 {
		return "the decision text has no Bundles: line naming at least one bundle"
	}
	if rec.tier != strings.ToLower(strings.TrimSpace(l.Tier)) {
		return fmt.Sprintf("the decision records Tier: %q, lifecycle.tier is %q", displayTier(rec.tier), l.Tier)
	}
	return bundlesProblem(l, rec.bundles)
}

// findDecision is the last decision whose trimmed id is ref, or nil.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func findDecision(decisions []model.Decision, ref string) *model.Decision {
	var dec *model.Decision
	for i := range decisions {
		if strings.TrimSpace(decisions[i].ID) == ref {
			dec = &decisions[i]
		}
	}
	return dec
}

// decisionRecord is the Tier: and Bundles: lines of a decision text.
type decisionRecord struct {
	tier        string
	tierSeen    bool
	bundles     []string
	bundlesSeen bool
}

// parseDecisionRecord reads the first Tier: line and every Bundles: line of a decision text.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func parseDecisionRecord(text string) decisionRecord {
	var rec decisionRecord
	for _, m := range decisionLine.FindAllStringSubmatch(text, -1) {
		rec.add(m[1], m[2])
	}
	return rec
}

// add records one labelled line: only the first Tier: line counts.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (r *decisionRecord) add(label, value string) {
	switch strings.ToLower(label) {
	case "tier":
		if !r.tierSeen {
			r.tier, r.tierSeen = strings.ToLower(value), true
		}
	case "bundles":
		r.bundlesSeen = true
		r.bundles = append(r.bundles, splitBundleList(value)...)
	}
}

// splitBundleList splits a comma separated bundle list into lowercase names.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func splitBundleList(value string) []string {
	var out []string
	for _, part := range strings.Split(value, ",") {
		if p := strings.ToLower(strings.TrimSpace(part)); p != "" {
			out = append(out, p)
		}
	}
	return out
}

// bundlesProblem says how the bundles a decision records differ from the effective bundles of the lifecycle, or "".
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func bundlesProblem(l model.ManifestLifecycle, recorded []string) string {
	want := l.EffectiveBundles()
	got := model.ManifestLifecycle{Bundles: recorded}.EffectiveBundles()
	sort.Strings(want)
	sort.Strings(got)
	if strings.Join(want, ",") != strings.Join(got, ",") {
		return fmt.Sprintf("the decision records Bundles: %q, lifecycle.bundles give %q", strings.Join(got, ", "), strings.Join(want, ", "))
	}
	return ""
}

// displayTier returns a tier as decision text writes it (MVP, PoC, ...).
// TRLC-LINKS: REQ-EMG-124
func displayTier(tier string) string {
	switch tier {
	case model.TierMVP:
		return "MVP"
	case model.TierPoC:
		return "PoC"
	}
	if tier == "" {
		return tier
	}
	return strings.ToUpper(tier[:1]) + tier[1:]
}

// AcceptedRiskScopes returns the appliesTo IDs of the local risks recorded
// as accepted (status accepted or response accept) with a complete,
// resolving acceptedBy: the scopes an owner accepted the risk for.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func AcceptedRiskScopes(b model.Bundle) map[string]bool {
	return acceptedRiskValues(b, func(r model.Risk) []string { return r.AppliesTo })
}

// AcceptedRiskDependencies returns the dependencies aliases of the local
// risks recorded as accepted with a complete, resolving acceptedBy: the
// dependencies an owner accepted below the system's life-cycle tier.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func AcceptedRiskDependencies(b model.Bundle) map[string]bool {
	return acceptedRiskValues(b, func(r model.Risk) []string { return r.Dependencies })
}

// acceptedRiskValues collects field of every local risk recorded as accepted
// with a complete, resolving acceptedBy.
// TRLC-LINKS: REQ-EMG-126
func acceptedRiskValues(b model.Bundle, field func(model.Risk) []string) map[string]bool {
	refs := NewAcceptanceRefs(b)
	out := map[string]bool{}
	for _, r := range b.Architecture.AuthoredArchitecture.Risks {
		if strings.Contains(r.ID, "::") || !riskRecordedAccepted(r) || !refs.Resolves(r.AcceptedBy) {
			continue
		}
		for _, v := range field(r) {
			out[strings.TrimSpace(v)] = true
		}
	}
	return out
}
