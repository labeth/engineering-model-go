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

// lifecycleDiagnostics checks engmod.yml lifecycle and the rules it adds:
// tier and bundles (lifecycle.invalid_tier, lifecycle.invalid_bundle, which
// the CUE schema already rejects on load), the tier decision
// (lifecycle.tier_decision_mismatch), requirementLinks guided at mvp or
// above (lifecycle.requirement_links_conflict), the safety bundle's aviation
// profile and software level, requirement status (lifecycle.requirement_*)
// and POA&M defers naming a rank-0 code (lifecycle.poam_defers_core). A
// model without lifecycle gets none of them.
// TRLC-LINKS: REQ-EMG-124, REQ-EMG-127, REQ-EMG-128, REQ-EMG-129
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func lifecycleDiagnostics(b model.Bundle) []Diagnostic {
	l := b.Manifest.Lifecycle
	if l == nil {
		return nil
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
	if ref := strings.TrimSpace(l.TierDecision); ref != "" {
		if problem := tierDecisionProblem(b, *l, ref); problem != "" {
			add("lifecycle.tier_decision_mismatch", SeverityError, "lifecycle.tierDecision", "lifecycle.tierDecision %s: %s", ref, problem)
		}
	}
	if rank >= model.TierRank(model.TierMVP) && strings.EqualFold(strings.TrimSpace(b.Manifest.RequirementLinks), model.RequirementLinksGuided) {
		add("lifecycle.requirement_links_conflict", SeverityError, "requirementLinks",
			"requirementLinks guided at tier %s: the tier requires strict requirement links from mvp; set requirementLinks: strict or remove it", strings.ToLower(l.Tier))
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
// separated; absent means Foundation) must equal lifecycle.tier and the
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
	for _, m := range decisionLine.FindAllStringSubmatch(dec.Decision, -1) {
		switch strings.ToLower(m[1]) {
		case "tier":
			if !tierSeen {
				tier, tierSeen = strings.ToLower(m[2]), true
			}
		case "bundles":
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
