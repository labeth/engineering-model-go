// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"

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
// and POA&M defers naming a rank-0 code (lifecycle.poam_defers_core).
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-125, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129, REQ-EMG-162, REQ-EMG-163
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func lifecycleDiagnostics(b model.Bundle) []Diagnostic {
	l := b.Manifest.Lifecycle
	if l == nil {
		return []Diagnostic{{Code: "lifecycle.missing", Severity: SeverityError, Path: "lifecycle",
			Message: "engmod.yml has no lifecycle block: the model has no tier, so no tier rule applies and the project is not at any stage"}}
	}
	var diags []Diagnostic
	add := func(code string, severity Severity, path, format string, args ...any) {
		diags = append(diags, Diagnostic{Code: code, Severity: severity, Message: fmt.Sprintf(format, args...), Path: path})
	}
	rank := model.TierRank(l.Tier)
	if rank == 0 {
		add("lifecycle.invalid_tier", SeverityError, "lifecycle.tier", "lifecycle.tier %q must be one of %s", l.Tier, strings.Join(model.Tiers, ", "))
	}
	for i, bundle := range l.Bundles {
		if !containsTrimmed(model.Bundles, strings.ToLower(bundle)) {
			add("lifecycle.invalid_bundle", SeverityError, fmt.Sprintf("lifecycle.bundles[%d]", i), "lifecycle bundle %q must be one of %s", bundle, strings.Join(model.Bundles, ", "))
		}
	}
	if len(l.Bundles) == 0 {
		add("lifecycle.bundles_missing", SeverityError, "lifecycle.bundles", "lifecycle.bundles is empty: write the assurance bundles (foundation, safety, security) explicitly; there is no default")
	}
	if ref := strings.TrimSpace(l.TierDecision); ref == "" {
		add("lifecycle.tier_decision_missing", SeverityError, "lifecycle.tierDecision", "lifecycle.tierDecision is empty: name the accepted DEC-TIER-* decision that records the tier")
	} else if problem := tierDecisionProblem(b, *l, ref); problem != "" {
		add("lifecycle.tier_decision_mismatch", SeverityError, "lifecycle.tierDecision", "lifecycle.tierDecision %s: %s", ref, problem)
	}
	if rank >= model.TierRank(model.TierMVP) && strings.EqualFold(strings.TrimSpace(b.Manifest.RequirementLinks), model.RequirementLinksGuided) {
		add("lifecycle.requirement_links_conflict", SeverityError, "requirementLinks",
			"requirementLinks guided at tier %s: the tier requires strict requirement links from mvp; set requirementLinks: strict", strings.ToLower(l.Tier))
	}
	if l.HasBundle(model.BundleSafety) {
		if b.Aviation == nil {
			add("lifecycle.aviation_profile_missing", SeverityWarning, "documents.aviation",
				"the safety bundle needs the DO-178C aviation profile: add documents.aviation (model/aviation.yml) with the software level and lifecycle data")
		}
		if b.Aviation == nil || strings.TrimSpace(b.Aviation.Aviation.Profile.SoftwareLevel) == "" {
			add("lifecycle.software_level_missing", SeverityWarning, "aviation.profile.softwareLevel",
				"the safety bundle needs aviation.profile.softwareLevel (A to E) from the system safety assessment")
		}
	}
	for i, r := range b.Requirements.Requirements {
		if strings.Contains(r.ID, "::") {
			continue
		}
		status := strings.ToLower(strings.TrimSpace(r.Status))
		path := fmt.Sprintf("requirements[%d].status", i)
		shown := status
		if shown == "" {
			shown = "unset"
		}
		if status == "draft" || status == "proposed" {
			add("lifecycle.requirement_unapproved", SeverityWarning, path, "requirement %s has status %s; the tier requires approved requirements", r.ID, shown)
		}
		if status != "verified" && status != "rejected" && status != "retired" {
			add("lifecycle.requirement_unverified", SeverityWarning, path, "requirement %s has status %s, not verified; record verified once its verification passed (engws status-suggest)", r.ID, shown)
		}
	}
	for i, p := range b.Architecture.AuthoredArchitecture.POAMItems {
		for j, d := range p.Defers {
			if code := strings.TrimSpace(d.Code); code != "" && deferNamesCore(code) {
				add("lifecycle.poam_defers_core", SeverityError, fmt.Sprintf("authoredArchitecture.poamItems[%d].defers[%d].code", i, j),
					"POA&M item %s defers %s, a rank-0 core code that holds at every tier and cannot be deferred; fix the finding instead", p.ID, code)
			}
		}
	}
	return diags
}

// deferNamesCore reports whether a defers code (exact or family.*) names
// a rank-0 core code.
// TRLC-LINKS: REQ-EMG-129
func deferNamesCore(code string) bool {
	if IsCoreCode(code) {
		return true
	}
	if prefix, ok := strings.CutSuffix(code, "*"); ok {
		for _, p := range coreCodes {
			if strings.HasPrefix(strings.TrimSuffix(p, "*"), prefix) {
				return true
			}
		}
	}
	return false
}

// tierDecisionProblem returns why the decision named by tierDecision does
// not record the lifecycle, or "" when it does: it must exist, be accepted,
// and its Tier: line (one tier, case-insensitive) and Bundles: line (comma
// separated; required, there is no default) must equal lifecycle.tier and the
// effective bundles.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func tierDecisionProblem(b model.Bundle, l model.ManifestLifecycle, ref string) string {
	var dec *model.Decision
	for i := range b.Architecture.Decisions {
		if strings.TrimSpace(b.Architecture.Decisions[i].ID) == ref {
			dec = &b.Architecture.Decisions[i]
		}
	}
	if dec == nil {
		return "names no decision of this model"
	}
	if !strings.EqualFold(strings.TrimSpace(dec.Status), "accepted") {
		return fmt.Sprintf("has status %q; the tier decision must be accepted", dec.Status)
	}
	var tier string
	var bundles []string
	tierSeen := false
	bundlesSeen := false
	for _, m := range decisionLine.FindAllStringSubmatch(dec.Decision, -1) {
		switch strings.ToLower(m[1]) {
		case "tier":
			if !tierSeen {
				tier, tierSeen = strings.ToLower(m[2]), true
			}
		case "bundles":
			bundlesSeen = true
			for _, part := range strings.Split(m[2], ",") {
				if p := strings.ToLower(strings.TrimSpace(part)); p != "" {
					bundles = append(bundles, p)
				}
			}
		}
	}
	if !tierSeen {
		return "the decision text has no Tier: line"
	}
	if !bundlesSeen || len(bundles) == 0 {
		return "the decision text has no Bundles: line naming at least one bundle"
	}
	if tier != strings.ToLower(strings.TrimSpace(l.Tier)) {
		return fmt.Sprintf("the decision records Tier: %s, lifecycle.tier is %s", displayTier(tier), l.Tier)
	}
	want := l.EffectiveBundles()
	got := model.ManifestLifecycle{Bundles: bundles}.EffectiveBundles()
	sort.Strings(want)
	sort.Strings(got)
	if strings.Join(want, ",") != strings.Join(got, ",") {
		return fmt.Sprintf("the decision records Bundles: %s, lifecycle.bundles give %s", strings.Join(got, ", "), strings.Join(want, ", "))
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
