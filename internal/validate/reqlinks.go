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
	if len(realized) > 0 || linkParents[strings.TrimSpace(r.ID)] {
		return true
	}
	return r.Derived && strings.TrimSpace(r.DerivedRationale) != ""
}

// contractItemKinds maps each local contract item ID (a contract.provides
// capability or a contract.assumptions assumption of use) to its kind.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091
func contractItemKinds(b model.Bundle) map[string]string {
	// Imported alias::ID entries (a bundle enriched with dependency
	// publications) are the dependency's items, not this model's.
	items := map[string]string{}
	for _, entry := range b.Architecture.Contract.Provides {
		if id := strings.TrimSpace(entry.ID); isLocalCapability(id, entry) {
			items[id] = "capability"
		}
	}
	addLocalAssumptions(items, b.Architecture.Contract.Assumptions)
	return items
}

// addLocalAssumptions records the assumptions of use of this model.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091
func addLocalAssumptions(items map[string]string, assumptions []model.ContractAssumption) {
	for _, a := range assumptions {
		if id := strings.TrimSpace(a.ID); !strings.Contains(id, "::") {
			items[id] = "assumption"
		}
	}
}

// isLocalCapability reports whether a contract.provides entry is a capability of this model.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091
func isLocalCapability(id string, entry model.ContractEntry) bool {
	return !strings.Contains(id, "::") && strings.EqualFold(strings.TrimSpace(entry.Kind), "capability")
}

// publishedContractItems maps each contract item a publication exposes to
// the publications that expose it.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-092
func publishedContractItems(b model.Bundle) map[string][]model.ManifestPublication {
	items := contractItemKinds(b)
	out := map[string][]model.ManifestPublication{}
	for _, p := range b.Manifest.Publications {
		for _, id := range publishedItemIDs(p, items) {
			out[id] = append(out[id], p)
		}
	}
	return out
}

// publishedItemIDs lists the trimmed ids of the local contract items a publication names.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-092
func publishedItemIDs(p model.ManifestPublication, items map[string]string) []string {
	var ids []string
	for _, id := range p.Architecture {
		if id = strings.TrimSpace(id); items[id] != "" {
			ids = append(ids, id)
		}
	}
	return ids
}

// requirementLinkCheck holds what the requirement link checks of one repository share.
type requirementLinkCheck struct {
	diagSink
	b            model.Bundle
	items        map[string]string
	published    map[string][]model.ManifestPublication
	requirements map[string]bool
	realized     map[string]bool
	severity     Severity
	applies      bool
	linkParents  map[string]bool
}

// requirementLinkDiagnostics checks the requirement link model of one
// repository: upward traces of requirements (realizes), contract items and
// their realizers, and the requirement link file against the manifest. The
// link rules that need the resolved dependencies (link.suspect, link.broken,
// link.aou_unverified, unpublished items) run at composition.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func requirementLinkDiagnostics(b model.Bundle, validID func(string) bool) []Diagnostic {
	severity, applies := RequirementLinkRules(b)
	c := &requirementLinkCheck{
		b: b, items: contractItemKinds(b), published: publishedContractItems(b),
		requirements: requirementIDSet(b.Requirements.Requirements), realized: map[string]bool{},
		severity: severity, applies: applies, linkParents: CapabilityLinkParents(b),
	}
	c.checkLinksMode()
	c.checkAssumptions()
	for i, r := range b.Requirements.Requirements {
		c.checkRequirement(indexedPath("requirements", i), r)
	}
	c.checkUnrealized()
	diags := append(c.diags, reqLinkFileDiagnostics(b, c.requirements, validID)...)
	return append(diags, RemovedFieldDiagnostics(b)...)
}

// requirementIDSet is the set of the trimmed requirement ids.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func requirementIDSet(reqs []model.Requirement) map[string]bool {
	ids := map[string]bool{}
	for _, r := range reqs {
		ids[strings.TrimSpace(r.ID)] = true
	}
	return ids
}

// checkLinksMode requires requirementLinks, when set, to be strict or guided.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c *requirementLinkCheck) checkLinksMode() {
	mode := strings.TrimSpace(c.b.Manifest.RequirementLinks)
	if mode == "" || mode == model.RequirementLinksStrict || mode == model.RequirementLinksGuided {
		return
	}
	c.fail("model.invalid_requirement_links", "requirementLinks", fmt.Sprintf("requirementLinks %q must be strict or guided", mode))
}

// checkAssumptions requires each local assumption of use to have an AOU-* id and text.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c *requirementLinkCheck) checkAssumptions() {
	for i, a := range c.b.Architecture.Contract.Assumptions {
		if strings.Contains(a.ID, "::") {
			continue
		}
		if !strings.HasPrefix(strings.TrimSpace(a.ID), "AOU-") || strings.TrimSpace(a.Text) == "" {
			c.fail("model.invalid_assumption", indexedPath("contract.assumptions", i), fmt.Sprintf("assumption of use %q needs an AOU-* id and text", a.ID))
		}
	}
}

// checkRequirement checks the realizes targets, the realizers in the
// publications and the upward trace of one requirement. An imported
// requirement (a bundle enriched with dependency publications) traces within
// its own model, checked there.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c *requirementLinkCheck) checkRequirement(path string, r model.Requirement) {
	id := strings.TrimSpace(r.ID)
	if strings.Contains(id, "::") {
		return
	}
	c.checkRealizes(path, id, r)
	for _, target := range r.Realizes {
		c.checkRealizer(path, id, strings.TrimSpace(target))
	}
	if !c.applies || TracesUpward(r, r.Realizes, c.linkParents) {
		return
	}
	c.diags = append(c.diags, Diagnostic{Code: "requirement.no_parent", Severity: c.severity, Path: path,
		Message: fmt.Sprintf("requirement %s traces upward to nothing: (a) add realizes naming a local capability or assumption of use it delivers (publish your own capabilities so a system above can link to them), (b) make it the parent of a reqLinks entry to a dependency's capability, or (c) mark it derived: true with derivedRationale", id)})
}

// checkRealizes requires each realizes target to be a local capability or assumption of use.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c *requirementLinkCheck) checkRealizes(path, id string, r model.Requirement) {
	for j, raw := range r.Realizes {
		target := strings.TrimSpace(raw)
		if c.items[target] == "" {
			c.fail("requirement.realizes_unknown", indexedPath(path+".realizes", j), fmt.Sprintf("requirement %s realizes %q, which %s", id, target, c.unknownRealizesReason(target)))
			continue
		}
		c.realized[target] = true
	}
}

// unknownRealizesReason says why a realizes target is not a local contract item.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c *requirementLinkCheck) unknownRealizesReason(target string) string {
	switch {
	case strings.Contains(target, "::"):
		return "is an imported ID; realizes names this model's own contract items"
	case c.requirements[target]:
		return "is a requirement; requirements realize contract items, not other requirements"
	}
	return "is not a local capability or assumption of use"
}

// checkRealizer requires a requirement that realizes a published item to be listed by the publications that expose it.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c *requirementLinkCheck) checkRealizer(path, id, target string) {
	for _, p := range c.published[target] {
		if !containsTrimmed(p.Requirements, id) {
			c.fail("contract.realizer_unpublished", path,
				fmt.Sprintf("requirement %s realizes %s, which publication %q exposes, but the publication does not list %s under requirements; a using system fingerprints the item with its realizers", id, target, p.ID, id))
		}
	}
}

// checkUnrealized warns about each published capability that no requirement realizes.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-091, REQ-EMG-092, REQ-EMG-093, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func (c *requirementLinkCheck) checkUnrealized() {
	for id, publications := range c.published {
		if c.items[id] == "capability" && !c.realized[id] {
			c.warn("contract.unrealized", "contract.provides",
				fmt.Sprintf("published capability %s (publication %q) is realized by no requirement; add realizes: [%s] to the requirements that deliver it", id, publications[0].ID, id))
		}
	}
}

// linkFileCheck holds what the checks of the requirement link file share.
type linkFileCheck struct {
	diagSink
	pins         map[string]string
	composed     map[string]bool
	actors       map[string]bool
	requirements map[string]bool
	validID      func(string) bool
}

// reqLinkFileDiagnostics checks each reqLinks entry against this model and its
// manifest: unique ids, a local parent requirement or evidence (a parent for
// a capability link), a child that names a composed dependency, the pinned
// version, the review status and, for an accepted link, a reviewer that is a
// model actor and a calendar review date.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112, REQ-EMG-118
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func reqLinkFileDiagnostics(b model.Bundle, requirements map[string]bool, validID func(string) bool) []Diagnostic {
	c := &linkFileCheck{pins: map[string]string{}, composed: map[string]bool{}, actors: map[string]bool{}, requirements: requirements, validID: validID}
	for _, d := range b.Manifest.Dependencies {
		c.pins[strings.TrimSpace(d.Alias)] = strings.TrimSpace(d.Version)
	}
	for _, sub := range b.Architecture.Composition.Subsystems {
		c.composed[strings.TrimSpace(sub.Dependency)] = true
	}
	for _, a := range b.Architecture.AuthoredArchitecture.Actors {
		c.actors[strings.TrimSpace(a.ID)] = true
	}
	seen := map[string]bool{}
	for i, link := range b.ReqLinks.ReqLinks {
		path := indexedPath("reqLinks", i)
		id := strings.TrimSpace(link.ID)
		if seen[id] {
			c.fail("model.duplicate_id", path+".id", fmt.Sprintf("duplicate requirement link id %q", id))
		}
		seen[id] = true
		c.checkLink(path, id, link)
	}
	return c.diags
}

// checkLink checks the parent, evidence, child and review of one requirement link.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112, REQ-EMG-118
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (c *linkFileCheck) checkLink(path, id string, link model.ReqLink) {
	c.checkParent(path, id, link)
	c.checkEvidence(path, id, link)
	c.checkChild(path, id, link)
	c.checkReview(path, id, link)
}

// checkParent requires a capability link to name a local parent requirement and
// any other link a parent or evidence.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112, REQ-EMG-118
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (c *linkFileCheck) checkParent(path, id string, link model.ReqLink) {
	parent := strings.TrimSpace(link.Parent)
	_, childID, _ := model.ParseQualifiedReference(strings.TrimSpace(link.Child))
	switch {
	case parent == "" && strings.HasPrefix(childID, "CAP-"):
		c.fail("link.parent_missing", path+".parent", fmt.Sprintf("link %s to capability %s names no parent requirement; a capability link says the capability satisfies a local requirement: set parent (evidence alone is for assumptions of use)", id, link.Child))
	case parent == "" && len(link.Evidence) == 0:
		c.fail("link.parent_unknown", path, fmt.Sprintf("link %s names neither a parent requirement nor evidence", id))
	case parent != "" && !c.requirements[parent]:
		c.fail("link.parent_unknown", path+".parent", fmt.Sprintf("link %s parent %q is not a requirement of this model", id, parent))
	}
}

// checkEvidence requires each evidence entry to be a local id.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112, REQ-EMG-118
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (c *linkFileCheck) checkEvidence(path, id string, link model.ReqLink) {
	for j, raw := range link.Evidence {
		if e := strings.TrimSpace(raw); e == "" || strings.Contains(e, "::") || !c.validID(e) {
			c.fail("link.parent_unknown", indexedPath(path+".evidence", j), fmt.Sprintf("link %s evidence %q is not a requirement, control, verification or test of this model", id, raw))
		}
	}
}

// checkChild requires the child to be alias::ID of a declared and composed
// dependency at the pinned version.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112, REQ-EMG-118
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (c *linkFileCheck) checkChild(path, id string, link model.ReqLink) {
	alias, _, err := model.ParseQualifiedReference(strings.TrimSpace(link.Child))
	pin, declared := c.pins[alias]
	switch {
	case err != nil || !declared:
		c.fail("link.child_unpublished", path+".child", fmt.Sprintf("link %s child %q must be alias::ID of a declared dependency", id, link.Child))
	case !c.composed[alias]:
		c.fail("link.child_unpublished", path+".child", fmt.Sprintf("link %s child %q: dependency %q is selected by no composition.subsystems entry, so its contract is not resolved", id, link.Child, alias))
	case strings.TrimSpace(link.Pinned.Version) != pin:
		c.fail("link.version_mismatch", path+".pinned.version", fmt.Sprintf("link %s is pinned to %s %s but engmod.yml pins %s; review the link at the new version (engws review-links)", id, alias, link.Pinned.Version, pin))
	}
}

// checkReview requires the status to be claimed or accepted and, for an
// accepted link, a model actor as reviewer and a calendar review date.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112, REQ-EMG-118
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (c *linkFileCheck) checkReview(path, id string, link model.ReqLink) {
	switch strings.TrimSpace(link.Status) {
	case model.ReqLinkClaimed:
		c.warn("link.claimed_only", path+".status", fmt.Sprintf("link %s is claimed, not yet reviewed; review it and set status accepted", id))
	case model.ReqLinkAccepted:
		c.checkAccepted(path, id, link)
	default:
		c.fail("link.invalid_status", path+".status", fmt.Sprintf("link %s status %q must be claimed or accepted", id, link.Status))
	}
}

// checkAccepted requires an accepted link to record its reviewer and review date.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-111, REQ-EMG-112, REQ-EMG-118
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (c *linkFileCheck) checkAccepted(path, id string, link model.ReqLink) {
	by, on := strings.TrimSpace(link.ReviewedBy), strings.TrimSpace(link.ReviewedOn)
	switch {
	case by == "":
		c.fail("link.review_unrecorded", path+".reviewedBy", fmt.Sprintf("link %s is accepted but names no reviewedBy; record the review with engws review-links --accept %s --by ACT-... (or set status claimed)", id, id))
	case !c.actors[by]:
		c.fail("link.review_unrecorded", path+".reviewedBy", fmt.Sprintf("link %s reviewedBy %q is not an actor of this model (architecture.actors)", id, by))
	}
	if on == "" {
		c.fail("link.review_unrecorded", path+".reviewedOn", fmt.Sprintf("link %s is accepted but names no reviewedOn; record the review with engws review-links --accept %s (or set status claimed)", id, id))
	} else if _, err := time.Parse(time.DateOnly, on); err != nil {
		c.fail("link.review_unrecorded", path+".reviewedOn", fmt.Sprintf("link %s reviewedOn %q is not a YYYY-MM-DD calendar date", id, on))
	}
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
