// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/internal/model"
)

// RequirementLinkRules returns the severity of the strictness-dependent
// requirement link rules (requirement.no_parent, link.aou_unverified,
// requirement.unsatisfied) and whether they apply to the model: in strict
// mode they are errors in every model; in guided mode (the default) they are
// warnings, and only in a model that publishes a contract item or has
// requirement links. From life-cycle tier mvp the effective mode is strict
// (ADR-EMG-031): the rules apply in every model, at the declared mode's
// severity, and ApplyTierPolicy sets the tier's severity.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-098, REQ-EMG-127
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func RequirementLinkRules(b model.Bundle) (Severity, bool) {
	if model.RequirementLinkMode(b.Manifest) == model.RequirementLinksStrict {
		return SeverityError, true
	}
	if EffectiveRequirementLinkMode(b) == model.RequirementLinksStrict {
		return SeverityWarning, true
	}
	return SeverityWarning, len(b.ReqLinks.ReqLinks) > 0 || len(publishedContractItems(b)) > 0
}

// CapabilityLinkParents returns the local requirements that are the parent
// of a reqLinks entry whose child is a dependency capability (alias::CAP-*):
// upward trace (b) of D22. A link to an assumption of use is evidence, not
// an upward trace.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-105
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func CapabilityLinkParents(b model.Bundle) map[string]bool {
	parents := map[string]bool{}
	for _, link := range b.ReqLinks.ReqLinks {
		parent := strings.TrimSpace(link.Parent)
		if _, child, err := model.ParseQualifiedReference(strings.TrimSpace(link.Child)); err == nil && parent != "" && strings.HasPrefix(child, "CAP-") {
			parents[parent] = true
		}
	}
	return parents
}

// TracesUpward reports whether a requirement satisfies the upward-trace
// rule (D22), given the contract items it realizes (realizes and a
// deprecated ref, valid or not: an invalid target is reported as
// requirement.realizes_unknown instead) and the capability link parents:
// (a) it realizes a local capability or assumption of use, (b) it is the
// parent of a link to a dependency capability, or (c) it is derived with a
// derivedRationale. Stakeholder sourceRefs do not count.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-105
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TracesUpward(r model.Requirement, realized []string, linkParents map[string]bool) bool {
	return len(realized) > 0 || linkParents[strings.TrimSpace(r.ID)] || (r.Derived && strings.TrimSpace(r.DerivedRationale) != "")
}

// contractItemKinds maps each local contract item ID (a contract.provides
// capability or a contract.assumptions assumption of use) to its kind.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091
func contractItemKinds(b model.Bundle) map[string]string {
	// Imported alias::ID entries (a bundle enriched with dependency
	// publications) are the dependency's items, not this model's.
	items := map[string]string{}
	for _, entry := range b.Architecture.Contract.Provides {
		if id := strings.TrimSpace(entry.ID); !strings.Contains(id, "::") && strings.EqualFold(strings.TrimSpace(entry.Kind), "capability") {
			items[id] = "capability"
		}
	}
	for _, a := range b.Architecture.Contract.Assumptions {
		if id := strings.TrimSpace(a.ID); !strings.Contains(id, "::") {
			items[id] = "assumption"
		}
	}
	return items
}

// publishedContractItems maps each contract item a publication exposes to
// the publications that expose it.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-092
func publishedContractItems(b model.Bundle) map[string][]model.ManifestPublication {
	items := contractItemKinds(b)
	out := map[string][]model.ManifestPublication{}
	for _, p := range b.Manifest.Publications {
		for _, id := range p.Architecture {
			if id = strings.TrimSpace(id); items[id] != "" {
				out[id] = append(out[id], p)
			}
		}
	}
	return out
}

// requirementLinkDiagnostics checks the requirement link model of one
// repository: upward traces of requirements (realizes), contract items and
// their realizers, and the requirement link file against the manifest. The
// link rules that need the resolved dependencies (link.suspect, link.broken,
// link.aou_unverified, unpublished items) run at composition.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func requirementLinkDiagnostics(b model.Bundle, validID func(string) bool) []Diagnostic {
	var diags []Diagnostic
	add := func(code string, severity Severity, path, format string, args ...any) {
		diags = append(diags, Diagnostic{Code: code, Severity: severity, Message: fmt.Sprintf(format, args...), Path: path})
	}
	if mode := strings.TrimSpace(b.Manifest.RequirementLinks); mode != "" && mode != model.RequirementLinksStrict && mode != model.RequirementLinksGuided {
		add("model.invalid_requirement_links", SeverityError, "requirementLinks", "requirementLinks %q must be strict or guided", mode)
	}
	items := contractItemKinds(b)
	published := publishedContractItems(b)
	requirements := map[string]bool{}
	for _, r := range b.Requirements.Requirements {
		requirements[strings.TrimSpace(r.ID)] = true
	}

	// Contract: unrealized published capabilities.
	realized := map[string]bool{}
	for i, a := range b.Architecture.Contract.Assumptions {
		if strings.Contains(a.ID, "::") {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(a.ID), "AOU-") || strings.TrimSpace(a.Text) == "" {
			add("model.invalid_assumption", SeverityError, fmt.Sprintf("contract.assumptions[%d]", i), "assumption of use %q needs an AOU-* id and text", a.ID)
		}
	}

	// Requirements: realizes targets, realizers in the publication, upward trace.
	severity, applies := RequirementLinkRules(b)
	linkParents := CapabilityLinkParents(b)
	for i, r := range b.Requirements.Requirements {
		path := fmt.Sprintf("requirements[%d]", i)
		id := strings.TrimSpace(r.ID)
		if strings.Contains(id, "::") {
			// An imported requirement (a bundle enriched with dependency
			// publications) traces within its own model, checked there.
			continue
		}
		targets := r.Realizes
		for j, raw := range r.Realizes {
			target := strings.TrimSpace(raw)
			if items[target] == "" {
				reason := "is not a local capability or assumption of use"
				switch {
				case strings.Contains(target, "::"):
					reason = "is an imported ID; realizes names this model's own contract items"
				case requirements[target]:
					reason = "is a requirement; requirements realize contract items, not other requirements"
				}
				add("requirement.realizes_unknown", SeverityError, fmt.Sprintf("%s.realizes[%d]", path, j), "requirement %s realizes %q, which %s", id, target, reason)
				continue
			}
			realized[target] = true
		}
		for _, target := range targets {
			for _, p := range published[strings.TrimSpace(target)] {
				if !containsTrimmed(p.Requirements, id) {
					add("contract.realizer_unpublished", SeverityError, path,
						"requirement %s realizes %s, which publication %q exposes, but the publication does not list %s under requirements; a using system fingerprints the item with its realizers", id, target, p.ID, id)
				}
			}
		}
		if !applies || TracesUpward(r, targets, linkParents) {
			continue
		}
		add("requirement.no_parent", severity, path,
			"requirement %s traces upward to nothing: (a) add realizes naming a local capability or assumption of use it delivers (publish your own capabilities so a system above can link to them), (b) make it the parent of a reqLinks entry to a dependency's capability, or (c) mark it derived: true with derivedRationale", id)
	}
	for id, publications := range published {
		if items[id] == "capability" && !realized[id] {
			add("contract.unrealized", SeverityWarning, "contract.provides",
				"published capability %s (publication %q) is realized by no requirement; add realizes: [%s] to the requirements that deliver it", id, publications[0].ID, id)
		}
	}

	diags = append(diags, reqLinkFileDiagnostics(b, requirements, validID)...)
	diags = append(diags, RemovedFieldDiagnostics(b)...)
	return diags
}

// reqLinkFileDiagnostics checks each reqLinks entry against this model and its
// manifest: unique ids, a local parent requirement or evidence (a parent for
// a capability link), a child that names a composed dependency, the pinned
// version, the review status and, for an accepted link, a reviewer that is a
// model actor and a calendar review date.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112, REQ-EMG-118
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func reqLinkFileDiagnostics(b model.Bundle, requirements map[string]bool, validID func(string) bool) []Diagnostic {
	var diags []Diagnostic
	add := func(code string, severity Severity, path, format string, args ...any) {
		diags = append(diags, Diagnostic{Code: code, Severity: severity, Message: fmt.Sprintf(format, args...), Path: path})
	}
	pins := map[string]string{}
	for _, d := range b.Manifest.Dependencies {
		pins[strings.TrimSpace(d.Alias)] = strings.TrimSpace(d.Version)
	}
	composed := map[string]bool{}
	for _, s := range b.Architecture.Composition.Subsystems {
		composed[strings.TrimSpace(s.Dependency)] = true
	}
	actors := map[string]bool{}
	for _, a := range b.Architecture.AuthoredArchitecture.Actors {
		actors[strings.TrimSpace(a.ID)] = true
	}
	seen := map[string]bool{}
	for i, link := range b.ReqLinks.ReqLinks {
		path := fmt.Sprintf("reqLinks[%d]", i)
		id := strings.TrimSpace(link.ID)
		if seen[id] {
			add("model.duplicate_id", SeverityError, path+".id", "duplicate requirement link id %q", id)
		}
		seen[id] = true
		parent := strings.TrimSpace(link.Parent)
		_, childID, _ := model.ParseQualifiedReference(strings.TrimSpace(link.Child))
		switch {
		case parent == "" && strings.HasPrefix(childID, "CAP-"):
			add("link.parent_missing", SeverityError, path+".parent", "link %s to capability %s names no parent requirement; a capability link says the capability satisfies a local requirement: set parent (evidence alone is for assumptions of use)", id, link.Child)
		case parent == "" && len(link.Evidence) == 0:
			add("link.parent_unknown", SeverityError, path, "link %s names neither a parent requirement nor evidence", id)
		case parent != "" && !requirements[parent]:
			add("link.parent_unknown", SeverityError, path+".parent", "link %s parent %q is not a requirement of this model", id, parent)
		}
		for j, raw := range link.Evidence {
			if e := strings.TrimSpace(raw); e == "" || strings.Contains(e, "::") || !validID(e) {
				add("link.parent_unknown", SeverityError, fmt.Sprintf("%s.evidence[%d]", path, j), "link %s evidence %q is not a requirement, control, verification or test of this model", id, raw)
			}
		}
		alias, _, err := model.ParseQualifiedReference(strings.TrimSpace(link.Child))
		pin, declared := pins[alias]
		switch {
		case err != nil || !declared:
			add("link.child_unpublished", SeverityError, path+".child", "link %s child %q must be alias::ID of a declared dependency", id, link.Child)
		case !composed[alias]:
			add("link.child_unpublished", SeverityError, path+".child", "link %s child %q: dependency %q is selected by no composition.subsystems entry, so its contract is not resolved", id, link.Child, alias)
		case strings.TrimSpace(link.Pinned.Version) != pin:
			add("link.version_mismatch", SeverityError, path+".pinned.version", "link %s is pinned to %s %s but engmod.yml pins %s; review the link at the new version (engws review-links)", id, alias, link.Pinned.Version, pin)
		}
		switch strings.TrimSpace(link.Status) {
		case model.ReqLinkClaimed:
			add("link.claimed_only", SeverityWarning, path+".status", "link %s is claimed, not yet reviewed; review it and set status accepted", id)
		case model.ReqLinkAccepted:
			by, on := strings.TrimSpace(link.ReviewedBy), strings.TrimSpace(link.ReviewedOn)
			switch {
			case by == "":
				add("link.review_unrecorded", SeverityError, path+".reviewedBy", "link %s is accepted but names no reviewedBy; record the review with engws review-links --accept %s --by ACT-... (or set status claimed)", id, id)
			case !actors[by]:
				add("link.review_unrecorded", SeverityError, path+".reviewedBy", "link %s reviewedBy %q is not an actor of this model (architecture.actors)", id, by)
			}
			if on == "" {
				add("link.review_unrecorded", SeverityError, path+".reviewedOn", "link %s is accepted but names no reviewedOn; record the review with engws review-links --accept %s (or set status claimed)", id, id)
			} else if _, err := time.Parse(time.DateOnly, on); err != nil {
				add("link.review_unrecorded", SeverityError, path+".reviewedOn", "link %s reviewedOn %q is not a YYYY-MM-DD calendar date", id, on)
			}
		default:
			add("link.invalid_status", SeverityError, path+".status", "link %s status %q must be claimed or accepted", id, link.Status)
		}
	}
	return diags
}

// TRLC-LINKS: REQ-EMG-092
func containsTrimmed(values []string, want string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == want {
			return true
		}
	}
	return false
}
