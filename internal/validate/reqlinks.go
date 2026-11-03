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
// requirement links.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-098
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func RequirementLinkRules(b model.Bundle) (Severity, bool) {
	if model.RequirementLinkMode(b.Manifest) == model.RequirementLinksStrict {
		return SeverityError, true
	}
	return SeverityWarning, len(b.ReqLinks.ReqLinks) > 0 || len(publishedContractItems(b)) > 0
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

	// Contract: deprecated refs, unrealized published capabilities.
	realized := map[string]bool{}
	refRealizers := map[string]string{}
	for i, entry := range b.Architecture.Contract.Provides {
		if strings.Contains(entry.ID, "::") {
			continue
		}
		if ref := strings.TrimSpace(entry.Ref); ref != "" {
			add("contract.ref_deprecated", SeverityWarning, fmt.Sprintf("contract.provides[%d].ref", i),
				"contract entry %s names requirement %s in ref; ref is deprecated: add realizes: [%s] to %s instead", entry.ID, ref, entry.ID, ref)
			if items[strings.TrimSpace(entry.ID)] != "" {
				refRealizers[ref] = strings.TrimSpace(entry.ID)
				realized[strings.TrimSpace(entry.ID)] = true
			}
		}
	}
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
	topLevel := strings.EqualFold(strings.TrimSpace(b.Manifest.Module.Kind), "system") && b.Manifest.InferenceHints.TopLevel
	for i, r := range b.Requirements.Requirements {
		path := fmt.Sprintf("requirements[%d]", i)
		id := strings.TrimSpace(r.ID)
		if strings.Contains(id, "::") {
			// An imported requirement (a bundle enriched with dependency
			// publications) traces within its own model, checked there.
			continue
		}
		targets := append([]string(nil), r.Realizes...)
		if item := refRealizers[id]; item != "" {
			targets = append(targets, item)
		}
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
		if !applies || len(targets) > 0 || (r.Derived && strings.TrimSpace(r.DerivedRationale) != "") || len(r.SourceRefs) > 0 || topLevel {
			continue
		}
		add("requirement.no_parent", severity, path,
			"requirement %s traces upward to nothing: add realizes naming the contract item it serves, mark it derived with derivedRationale, name its stakeholder sourceRefs, or (in a top-level system) set inferenceHints.topLevel", id)
	}
	for id, publications := range published {
		if items[id] == "capability" && !realized[id] {
			add("contract.unrealized", SeverityWarning, "contract.provides",
				"published capability %s (publication %q) is realized by no requirement; add realizes: [%s] to the requirements that deliver it", id, publications[0].ID, id)
		}
	}

	// Composition: allocations are deprecated in favour of reqLinks.
	for i, a := range b.Architecture.Composition.Allocations {
		add("composition.allocation_deprecated", SeverityWarning, fmt.Sprintf("composition.allocations[%d]", i),
			"allocation of %s to %s is deprecated: record it as a reqLinks entry (parent %s, child %s) in the requirement link file", a.Requirement, a.Target, a.Requirement, a.Target)
	}

	diags = append(diags, reqLinkFileDiagnostics(b, requirements, validID)...)
	return diags
}

// reqLinkFileDiagnostics checks each reqLinks entry against this model and its
// manifest: unique ids, a local parent requirement or evidence (a parent for
// a capability link), a child that names a composed dependency, the pinned
// version, the review status and, for an accepted link, a reviewer that is a
// model actor and a calendar review date.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112
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
