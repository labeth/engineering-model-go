// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

// Requirement link chapters. A model with requirement links gets a
// "Requirement Links" chapter: per local requirement, the tree requirement →
// link (state) → linked child item → its realizing requirements with their
// trace status in the child, and the assumptions of use with their
// evidence. A model that publishes contract items gets a "Contract Items"
// chapter: each item with its realizing requirements, and,
// only when the caller opts in (Options.UsedBy) and the render runs in a
// workspace whose sibling checkouts link the items, a "Used By" section
// computed at render time.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/trace"
)

// linkStateLabel names an evaluated link state; every state but unchanged
// (and local, a workspace checkout) is marked.
// TRLC-LINKS: REQ-EMG-107
func linkStateLabel(s reqlinks.LinkStatus) string {
	switch s.State {
	case reqlinks.StateUnchanged:
		return "unchanged"
	case reqlinks.StateLocal:
		return "local: workspace checkout, not checked"
	case reqlinks.StateBroken:
		return "*BROKEN*: the child item does not exist at " + s.Version
	case reqlinks.StateVersionMismatch:
		return fmt.Sprintf("*VERSION MISMATCH*: reviewed at %s, dependency %s; run engws review-links", s.Pinned, s.Version)
	case reqlinks.StateChildUnpublished:
		return "*UNPUBLISHED*: " + s.Detail
	default:
		return "*" + strings.ToUpper(string(s.State)) + "*: " + s.Detail
	}
}

// reviewLabel names the review status; a claimed link is marked.
// TRLC-LINKS: REQ-EMG-107
func reviewLabel(status string) string {
	if strings.TrimSpace(status) == model.ReqLinkClaimed {
		return "*CLAIMED*: not yet reviewed"
	}
	return strings.TrimSpace(status)
}

// realizerTraceLabel describes a realizing requirement's trace status in
// its own repository.
// TRLC-LINKS: REQ-EMG-107
func realizerTraceLabel(s *trace.DelegatedStatus) string {
	if s == nil || s.Status == "not-available" {
		return "status not available"
	}
	if s.Status == "unknown-requirement" {
		return "unknown requirement"
	}
	switch {
	case s.Implemented && s.TestLinked:
		return "implemented, test linked"
	case s.Implemented:
		return "implemented, not test linked"
	case s.TestLinked:
		return "test linked, not implemented"
	default:
		return "not implemented, not test linked"
	}
}

// oneLineText collapses whitespace for a list item and neutralizes AsciiDoc
// passthrough syntax, so text from a dependency's model cannot inject raw
// output into the document.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-110
func oneLineText(s string) string { return escapePassthrough(textutil.CollapseSpace(s)) }

// escapePassthrough breaks the AsciiDoc passthrough forms (+++…+++, ++…++,
// pass:[…] and other pass macros, $$…$$) by inserting a {empty} attribute
// reference, so the text renders literally instead of emitting raw HTML.
// TRLC-LINKS: REQ-EMG-110
func escapePassthrough(s string) string {
	s = strings.ReplaceAll(s, "+++", "+{empty}+{empty}+")
	s = strings.ReplaceAll(s, "++", "+{empty}+")
	s = strings.ReplaceAll(s, "$$", "${empty}$")
	s = strings.ReplaceAll(s, "pass:", "pass{empty}:")
	return s
}

// requirementLinkIndex indexes the evaluated requirement links of a composed model.
type requirementLinkIndex struct {
	statuses       []reqlinks.LinkStatus
	reviewStatus   map[string]string
	evidence       map[string][]string
	byParent       map[string][]reqlinks.LinkStatus
	realizerStatus func(alias, id string) *trace.DelegatedStatus
}

// newRequirementLinkIndex evaluates the links of the model against the dependencies.
// TRLC-LINKS: REQ-EMG-107
func newRequirementLinkIndex(bundle model.Bundle, composed composition.CompositionResult) requirementLinkIndex {
	ix := requirementLinkIndex{
		statuses:       reqlinks.EvaluateLinks(composed.Root.Bundle, composition.LinkDependencies(composed.Root)),
		reviewStatus:   map[string]string{},
		evidence:       map[string][]string{},
		byParent:       map[string][]reqlinks.LinkStatus{},
		realizerStatus: trace.DependencyRequirementStatus(composed),
	}
	for _, l := range bundle.ReqLinks.ReqLinks {
		ix.reviewStatus[strings.TrimSpace(l.ID)] = strings.TrimSpace(l.Status)
		ix.evidence[strings.TrimSpace(l.ID)] = l.Evidence
	}
	for _, s := range ix.statuses {
		if s.Parent != "" {
			ix.byParent[s.Parent] = append(ix.byParent[s.Parent], s)
		}
	}
	return ix
}

// renderRequirementLinksChapter renders the requirement link tree of a
// composed model with requirement links ("" otherwise).
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE
func renderRequirementLinksChapter(bundle model.Bundle, composed composition.CompositionResult) string {
	if len(bundle.ReqLinks.ReqLinks) == 0 || composed.Root == nil {
		return ""
	}
	ix := newRequirementLinkIndex(bundle, composed)
	var b strings.Builder
	b.WriteString("\n<<<\n== Requirement Links\n\n")
	b.WriteString("Each requirement of this model with the requirement links (`model/req-links.yml`) that satisfy it: the linked contract item of a dependency at the pinned version, ")
	b.WriteString("and the dependency requirements that realize the item with their implementation and test status in the dependency's own repository. ")
	b.WriteString("A link pinned to another version than the dependency, a broken and a claimed link are marked; review it with `engws review-links`.\n\n")
	for _, r := range bundle.Requirements.Requirements {
		ix.writeRequirement(&b, r)
	}
	b.WriteString("\n")
	// Assumptions of use of the resolved dependencies and the links that
	// evidence them.
	ix.writeAssumptions(&b, dependencyAssumptions(composed))
	return b.String()
}

// writeRequirement writes a requirement with its links, or why it has none.
// TRLC-LINKS: REQ-EMG-107
func (ix requirementLinkIndex) writeRequirement(b *strings.Builder, r model.Requirement) {
	id := strings.TrimSpace(r.ID)
	b.WriteString(fmt.Sprintf("* *%s*: %s\n", id, oneLineText(r.Text)))
	links := ix.byParent[id]
	if len(links) == 0 {
		b.WriteString(noLinkLine(r))
	}
	for _, s := range links {
		ix.writeLink(b, s)
	}
}

// noLinkLine says that a requirement is derived, or has no requirement link.
// TRLC-LINKS: REQ-EMG-107
func noLinkLine(r model.Requirement) string {
	if r.Derived {
		return "** derived: " + oneLineText(r.DerivedRationale) + "\n"
	}
	return "** no requirement link\n"
}

// writeLink writes a link with its linked item and the realizers of the item.
// TRLC-LINKS: REQ-EMG-107
func (ix requirementLinkIndex) writeLink(b *strings.Builder, s reqlinks.LinkStatus) {
	b.WriteString(fmt.Sprintf("** link %s (%s, %s) to `%s`", s.ID, linkStateLabel(s), reviewLabel(ix.reviewStatus[s.ID]), s.Child))
	if s.Item == nil {
		b.WriteString("\n")
		return
	}
	b.WriteString(": " + oneLineText(s.Item.Text) + "\n")
	for _, rz := range reqlinks.SortedRealizers(s.Item.Realizers) {
		b.WriteString(fmt.Sprintf("*** `%s::%s` (%s): %s\n", s.Alias, rz.ID, realizerTraceLabel(ix.realizerStatus(s.Alias, rz.ID)), oneLineText(rz.Text)))
	}
}

// linkAssumption is an assumption of use that a dependency publishes.
type linkAssumption struct {
	id   string
	text string
}

// dependencyAssumptions lists the assumptions of use of the dependencies, by qualified ID.
// TRLC-LINKS: REQ-EMG-107
func dependencyAssumptions(composed composition.CompositionResult) []linkAssumption {
	var assumptions []linkAssumption
	for _, dep := range composition.LinkDependencies(composed.Root) {
		assumptions = append(assumptions, publishedAssumptions(dep)...)
	}
	sort.Slice(assumptions, func(i, j int) bool { return assumptions[i].id < assumptions[j].id })
	return assumptions
}

// publishedAssumptions lists the assumptions of use a dependency publishes.
// TRLC-LINKS: REQ-EMG-107
func publishedAssumptions(dep reqlinks.Dependency) []linkAssumption {
	var assumptions []linkAssumption
	for id, item := range reqlinks.PublishedItems(dep.Bundle, dep.Publications...) {
		if item.Kind == reqlinks.KindAssumption {
			assumptions = append(assumptions, linkAssumption{dep.Alias + "::" + id, oneLineText(item.Text)})
		}
	}
	return assumptions
}

// writeAssumptions writes the assumptions of use with the links that evidence
// them.
// TRLC-LINKS: REQ-EMG-107
func (ix requirementLinkIndex) writeAssumptions(b *strings.Builder, assumptions []linkAssumption) {
	if len(assumptions) == 0 {
		return
	}
	b.WriteString("=== Assumptions of Use\n\n")
	b.WriteString("Conditions a dependency relies on, with the local requirements, controls or tests that show they hold in this model.\n\n")
	for _, a := range assumptions {
		b.WriteString(fmt.Sprintf("* `%s`: %s\n", a.id, a.text))
		ix.writeEvidence(b, a.id)
	}
	b.WriteString("\n")
}

// writeEvidence writes the links with evidence that name the assumption, or
// that it is unverified.
// TRLC-LINKS: REQ-EMG-107
func (ix requirementLinkIndex) writeEvidence(b *strings.Builder, assumptionID string) {
	found := false
	for _, s := range ix.statuses {
		if s.Child != assumptionID || len(ix.evidence[s.ID]) == 0 {
			continue
		}
		found = true
		b.WriteString(fmt.Sprintf("** evidence %s (link %s, %s, %s)\n", strings.Join(ix.evidence[s.ID], ", "), s.ID, linkStateLabel(s), reviewLabel(ix.reviewStatus[s.ID])))
	}
	if !found {
		b.WriteString("** *UNVERIFIED*: no link with evidence names this assumption (link.aou_unverified)\n")
	}
}

// renderContractItemsChapter renders the published contract items of the
// model with their realizing requirements, and, with
// usedBy, the users of the active workspace that link them ("" when it
// publishes none).
// TRLC-LINKS: REQ-EMG-107
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE
func renderContractItemsChapter(bundle model.Bundle, workspaceMode string, usedBy bool) string {
	items := reqlinks.PublishedItems(bundle, bundle.Manifest.Publications...)
	if len(items) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("\n<<<\n== Contract Items\n\n")
	b.WriteString("The capabilities and assumptions of use this model publishes, each with the requirements that realize it. ")
	b.WriteString("A using system pins the version it reviewed in its requirement links; a release that changes the text or verification criteria of an item or a realizing requirement needs a new review in every user.\n\n")
	for _, id := range sortedKeys(items) {
		writeContractItem(&b, id, items[id])
	}
	if usedBy {
		writeContractUsers(&b, composition.WorkspaceLinkUsers(bundle, workspaceMode))
	}
	return b.String()
}

// writeContractItem writes a published item with its verification criteria and realizers.
// TRLC-LINKS: REQ-EMG-107
func writeContractItem(b *strings.Builder, id string, item reqlinks.Item) {
	b.WriteString(fmt.Sprintf("=== %s (%s)\n\n%s\n\n", id, item.Kind, oneLineText(item.Text)))
	if v := oneLineText(item.VerificationCriteria); v != "" {
		b.WriteString("Verification criteria: " + v + "\n\n")
	}
	realizers := reqlinks.SortedRealizers(item.Realizers)
	if len(realizers) == 0 {
		b.WriteString("Realized by no published requirement.\n\n")
		return
	}
	b.WriteString("Realized by:\n\n")
	for _, r := range realizers {
		b.WriteString(fmt.Sprintf("* `%s`: %s\n", r.ID, oneLineText(r.Text)))
	}
	b.WriteString("\n")
}

// writeContractUsers writes the Used By table of the requirement links of the
// other models in the workspace.
// TRLC-LINKS: REQ-EMG-107
func writeContractUsers(b *strings.Builder, uses []composition.LinkUse) {
	if len(uses) == 0 {
		return
	}
	b.WriteString("=== Used By\n\n")
	b.WriteString("Requirement links of the other models in this workspace onto these items, found at render time because the render opted in (engdoc --used-by). This section is not part of the model; do not commit a document that contains it into this repository.\n\n")
	b.WriteString("[cols=\"2,2,2,2,1\",options=\"header\"]\n|===\n|Item |Model |Link |Parent requirement |Status\n")
	cell := func(v string) string { return escapeTableCell(oneLineText(v)) }
	for _, u := range uses {
		b.WriteString(fmt.Sprintf("|%s\n|%s\n|%s\n|%s\n|%s\n", cell(u.Item), cell(u.Model), cell(u.Link), cell(nonEmptyDash(u.Parent)), cell(u.Status)))
	}
	b.WriteString("|===\n\n")
}

// nonEmptyDash returns s, or "-" when it is empty.
// TRLC-LINKS: REQ-EMG-107
func nonEmptyDash(s string) string {
	if strings.TrimSpace(s) == "" {
		return "-"
	}
	return s
}
