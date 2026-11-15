// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"cmp"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func mapDesignGroups(d model.DesignDocument) map[string]model.DesignFunctionalGroup {
	out := map[string]model.DesignFunctionalGroup{}
	for _, x := range d.Design.FunctionalGroups {
		out[x.ID] = x
	}
	return out
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func mapDesignUnits(d model.DesignDocument) map[string]model.DesignFunctionalUnit {
	out := map[string]model.DesignFunctionalUnit{}
	for _, x := range d.Design.FunctionalUnits {
		out[x.ID] = x
	}
	return out
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func viewDesignKey(kind string) string {
	switch kind {
	case "architecture-intent":
		return "architecture_intent"
	case "communication":
		return "communication"
	case "deployment":
		return "deployment"
	case "security":
		return "security"
	case "traceability":
		return "traceability"
	case "state-lifecycle":
		return "state_lifecycle"
	default:
		return kind
	}
}

// buildDesignDetails is the design detail of an entity for each view: the
// title and narrative the design document gives for the view, else the view kind
// and the entity's own introduction.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildDesignDetails(intro string, designViews map[string]model.DesignView, views []model.View) []asciidocDesignDetail {
	out := make([]asciidocDesignDetail, 0, len(views))
	for _, v := range views {
		detail := asciidocDesignDetail{ViewID: v.ID, Title: strings.TrimSpace(v.Kind), Narrative: strings.TrimSpace(intro)}
		if dv, ok := designViews[viewDesignKey(v.Kind)]; ok {
			detail.Title = trimmedOr(dv.Title, detail.Title)
			detail.Narrative = trimmedOr(dv.Narrative, detail.Narrative)
		}
		out = append(out, detail)
	}
	return out
}

// trimmedOr is the trimmed value, or the fallback when it is blank.
// TRLC-LINKS: REQ-EMG-003
func trimmedOr(value, fallback string) string {
	if t := strings.TrimSpace(value); t != "" {
		return t
	}
	return fallback
}

// TRLC-LINKS: REQ-EMG-003
func detailForView(details []asciidocDesignDetail, viewID string) asciidocDesignDetail {
	for _, d := range details {
		if d.ViewID == viewID {
			return d
		}
	}
	if len(details) > 0 {
		return details[0]
	}
	return asciidocDesignDetail{ViewID: viewID, Title: "Design", Narrative: ""}
}

// buildLabelIndex maps the ID of every authored element to its label.
// TRLC-LINKS: REQ-EMG-003
func buildLabelIndex(a model.AuthoredArchitecture) map[string]string {
	out := map[string]string{}
	labelStructure(out, a)
	labelInterfaces(out, a)
	labelBehavior(out, a)
	labelFlows(out, a.Flows)
	return out
}

// labelStructure labels the groups, units, actors, attack vectors and referenced elements.
// TRLC-LINKS: REQ-EMG-003
func labelStructure(out map[string]string, a model.AuthoredArchitecture) {
	for _, x := range a.FunctionalGroups {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.FunctionalUnits {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.Actors {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.AttackVectors {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.ReferencedElements {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
}

// labelInterfaces labels the interfaces, data objects, deployment targets, controls and trust boundaries.
// TRLC-LINKS: REQ-EMG-003
func labelInterfaces(out map[string]string, a model.AuthoredArchitecture) {
	for _, x := range a.Interfaces {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.DataObjects {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.DeploymentTargets {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.Controls {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.TrustBoundaries {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
}

// labelBehavior labels the machines, their states and the events.
// TRLC-LINKS: REQ-EMG-003
func labelBehavior(out map[string]string, a model.AuthoredArchitecture) {
	for _, m := range a.Machines {
		out[m.ID] = textutil.NonEmpty(m.Name, m.ID)
	}
	for _, x := range a.MachineStates() {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
	for _, x := range a.Events {
		out[x.ID] = textutil.NonEmpty(x.Name, x.ID)
	}
}

// labelFlows labels the flows and, under flowID::stepID, their steps.
// TRLC-LINKS: REQ-EMG-003
func labelFlows(out map[string]string, flows []model.Flow) {
	for _, f := range flows {
		out[f.ID] = textutil.NonEmpty(f.Title, f.ID)
		for _, s := range f.Steps {
			if sid := strings.TrimSpace(s.ID); sid != "" {
				out[strings.TrimSpace(f.ID)+"::"+sid] = trimmedOr(s.Action, sid)
			}
		}
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func referenceAnchor(kind, id string) string {
	return "REF_" + strings.ToUpper(strings.TrimSpace(kind)) + "_" + codescan.SanitizeNode(id)
}

// referenceSources are the model and the inferred layers that the reference index lists.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type referenceSources struct {
	bundle       model.Bundle
	requirements model.RequirementsDocument
	runtime      []codescan.RuntimeItem
	code         []codescan.CodeItem
	verification []codescan.VerificationCheck
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildReferenceIndex(src referenceSources) asciidocReferenceIndex {
	return asciidocReferenceIndex{
		Authored:     buildAuthoredReferences(src.bundle, src.requirements),
		Catalog:      buildCatalogReferences(src.bundle.Catalog),
		Runtime:      buildRuntimeReferences(src.runtime),
		Code:         buildCodeReferences(src.code),
		Verification: buildVerificationReferences(src.verification),
	}
}

// authoredRef is one authored element to list in the reference index.
type authoredRef struct {
	anchorKind string
	kind       string
	id         string
	name       string
	desc       string
}

// authoredReferences collects the authored entries that are not catalog entries.
type authoredReferences struct {
	catalogIDs map[string]bool
	entries    []asciidocReferenceEntry
}

// add lists the element, unless its ID is blank or a catalog ID.
// TRLC-LINKS: REQ-EMG-003
func (r *authoredReferences) add(ref authoredRef) {
	id := strings.TrimSpace(ref.id)
	if id == "" || r.catalogIDs[strings.ToUpper(id)] {
		return
	}
	r.entries = append(r.entries, asciidocReferenceEntry{
		Anchor:      referenceAnchor(ref.anchorKind, id),
		ID:          id,
		Name:        strings.TrimSpace(ref.name),
		Kind:        strings.TrimSpace(ref.kind),
		Description: strings.TrimSpace(ref.desc),
	})
}

// buildAuthoredReferences lists the authored elements and the requirements, by kind then ID.
// TRLC-LINKS: REQ-EMG-003
func buildAuthoredReferences(bundle model.Bundle, requirements model.RequirementsDocument) []asciidocReferenceEntry {
	refs := &authoredReferences{catalogIDs: catalogEntryIDSet(bundle.Catalog), entries: []asciidocReferenceEntry{}}
	a := bundle.Architecture.AuthoredArchitecture
	refs.addStructure(a)
	refs.addInterfaces(a)
	refs.addBehavior(a)
	refs.addFlows(a.Flows)
	for _, x := range requirements.Requirements {
		refs.entries = append(refs.entries, asciidocReferenceEntry{
			Anchor:       referenceAnchor("idx-req", x.ID),
			TargetAnchor: referenceAnchor("req", x.ID),
			ID:           x.ID,
			Name:         x.ID,
			Kind:         "Requirement",
			Description:  strings.TrimSpace(x.Text),
		})
	}
	sortReferenceEntries(refs.entries)
	return refs.entries
}

// sortReferenceEntries sorts the entries by kind, then ID.
// TRLC-LINKS: REQ-EMG-003
func sortReferenceEntries(entries []asciidocReferenceEntry) {
	sort.SliceStable(entries, func(i, j int) bool {
		return cmp.Or(cmp.Compare(entries[i].Kind, entries[j].Kind), cmp.Compare(entries[i].ID, entries[j].ID)) < 0
	})
}

// notAvailable is the text n/a for a blank value.
// TRLC-LINKS: REQ-EMG-003
func notAvailable(value string) string {
	return textutil.NonEmpty(strings.TrimSpace(value), "n/a")
}

// addStructure lists the groups, units, actors, attack vectors and referenced elements.
// TRLC-LINKS: REQ-EMG-003
func (r *authoredReferences) addStructure(a model.AuthoredArchitecture) {
	for _, x := range a.FunctionalGroups {
		r.add(authoredRef{"idx-fg", "Functional Group", x.ID, textutil.NonEmpty(x.Name, x.ID), x.Description})
	}
	for _, x := range a.FunctionalUnits {
		r.add(authoredRef{"idx-fu", "Functional Unit", x.ID, textutil.NonEmpty(x.Name, x.ID), x.Prose})
	}
	for _, x := range a.Actors {
		r.add(authoredRef{"idx-actor", "Actor", x.ID, textutil.NonEmpty(x.Name, x.ID), x.Description})
	}
	for _, x := range a.AttackVectors {
		r.add(authoredRef{"idx-attack", "Attack Vector", x.ID, textutil.NonEmpty(x.Name, x.ID), x.Description})
	}
	for _, x := range a.ReferencedElements {
		r.add(authoredRef{"idx-ref", "Referenced Element", x.ID, textutil.NonEmpty(x.Name, x.ID), x.Kind + " / " + x.Layer})
	}
}

// addInterfaces lists the interfaces, data objects, deployment targets, controls and trust boundaries.
// TRLC-LINKS: REQ-EMG-003
func (r *authoredReferences) addInterfaces(a model.AuthoredArchitecture) {
	for _, x := range a.Interfaces {
		desc := fmt.Sprintf("protocol=%s; endpoint=%s; schemaRef=%s; owner=%s", notAvailable(x.Protocol), notAvailable(x.Endpoint), notAvailable(x.SchemaRef), notAvailable(x.Owner))
		r.add(authoredRef{"idx-if", "Interface", x.ID, textutil.NonEmpty(x.Name, x.ID), desc})
	}
	for _, x := range a.DataObjects {
		desc := fmt.Sprintf("termRef=%s; schemaRef=%s; sensitivity=%s", notAvailable(x.TermRef), notAvailable(x.SchemaRef), notAvailable(x.Sensitivity))
		r.add(authoredRef{"idx-do", "Data Object", x.ID, textutil.NonEmpty(x.Name, x.ID), desc})
	}
	for _, x := range a.DeploymentTargets {
		desc := fmt.Sprintf("environment=%s; region=%s; account=%s; cluster=%s; namespace=%s; trustZone=%s", notAvailable(x.Environment), notAvailable(x.Region), notAvailable(x.Account), notAvailable(x.Cluster), notAvailable(x.Namespace), notAvailable(x.TrustZone))
		r.add(authoredRef{"idx-dep", "Deployment Target", x.ID, textutil.NonEmpty(x.Name, x.ID), desc})
	}
	for _, x := range a.Controls {
		desc := fmt.Sprintf("category=%s; description=%s", notAvailable(x.Category), notAvailable(x.Description))
		r.add(authoredRef{"idx-ctrl", "Control", x.ID, textutil.NonEmpty(x.Name, x.ID), desc})
	}
	for _, x := range a.TrustBoundaries {
		r.add(authoredRef{"idx-tb", "Trust Boundary", x.ID, textutil.NonEmpty(x.Name, x.ID), notAvailable(x.Description)})
	}
}

// addBehavior lists the machines, their states and the events.
// TRLC-LINKS: REQ-EMG-003
func (r *authoredReferences) addBehavior(a model.AuthoredArchitecture) {
	for _, m := range a.Machines {
		desc := fmt.Sprintf("owner=%s; initial=%s; states=%d; transitions=%d", notAvailable(m.Owner), notAvailable(m.Initial), len(m.States), len(m.Transitions))
		r.add(authoredRef{"idx-machine", "Machine", m.ID, textutil.NonEmpty(m.Name, m.ID), desc})
	}
	for _, x := range a.MachineStates() {
		r.add(authoredRef{"idx-state", "State", x.ID, textutil.NonEmpty(x.Name, x.ID), notAvailable(x.Description)})
	}
	for _, x := range a.Events {
		r.add(authoredRef{"idx-evt", "Event", x.ID, textutil.NonEmpty(x.Name, x.ID), notAvailable(x.Description)})
	}
}

// addFlows lists the flows and, as flowID::stepID, their steps.
// TRLC-LINKS: REQ-EMG-003
func (r *authoredReferences) addFlows(flows []model.Flow) {
	for _, f := range flows {
		desc := fmt.Sprintf("entry=%s; exits=%s; steps=%d", strings.Join(f.Entry, ", "), strings.Join(f.Exits, ", "), len(f.Steps))
		r.add(authoredRef{"idx-flow", "Flow", f.ID, textutil.NonEmpty(f.Title, f.ID), desc})
		for _, s := range f.Steps {
			r.addFlowStep(f.ID, s)
		}
	}
}

// addFlowStep lists a flow step that has an ID.
// TRLC-LINKS: REQ-EMG-003
func (r *authoredReferences) addFlowStep(flowID string, s model.FlowStep) {
	sid := strings.TrimSpace(s.ID)
	if sid == "" {
		return
	}
	desc := fmt.Sprintf("kind=%s; ref=%s; action=%s; dataIn=%s; dataOut=%s; async=%t", notAvailable(s.Kind), notAvailable(s.Ref), notAvailable(s.Action), strings.Join(s.DataIn, ", "), strings.Join(s.DataOut, ", "), s.Async)
	r.add(authoredRef{"idx-flow-step", "Flow Step", strings.TrimSpace(flowID) + "::" + sid, textutil.NonEmpty(strings.TrimSpace(s.Action), sid), desc})
}

// catalogReferences collects the catalog entries, their aliases and the
// built-in terms, each ID once.
type catalogReferences struct {
	seen    map[string]bool
	entries []asciidocReferenceEntry
}

// claim reports whether the ID is new, and marks it seen.
// TRLC-LINKS: REQ-EMG-003
func (r *catalogReferences) claim(id string) bool {
	key := strings.ToUpper(id)
	if r.seen[key] {
		return false
	}
	r.seen[key] = true
	return true
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildCatalogReferences(doc model.CatalogDocument) []asciidocReferenceEntry {
	refs := &catalogReferences{seen: map[string]bool{}, entries: []asciidocReferenceEntry{}}
	refs.addBuiltInTerms()
	c := doc.Catalog
	refs.addEntries("System", c.Systems)
	refs.addEntries("Functional Group", c.FunctionalGroups)
	refs.addEntries("Functional Unit", c.FunctionalUnits)
	refs.addEntries("Referenced Element", c.ReferencedElements)
	refs.addEntries("Actor", c.Actors)
	refs.addEntries("Attack Vector", c.AttackVectors)
	refs.addEntries("Event", c.Events)
	refs.addEntries("State", c.States)
	refs.addEntries("Feature", c.Features)
	refs.addEntries("Mode", c.Modes)
	refs.addEntries("Condition", c.Conditions)
	refs.addEntries("Data Term", c.DataTerms)
	sortReferenceEntries(refs.entries)
	return refs.entries
}

// addBuiltInTerms lists the terms of the engineering model itself.
// TRLC-LINKS: REQ-EMG-003
func (r *catalogReferences) addBuiltInTerms() {
	for _, term := range builtInEngineeringModelTerms() {
		id := strings.TrimSpace(term.ID)
		if id == "" || !r.claim(id) {
			continue
		}
		r.entries = append(r.entries, asciidocReferenceEntry{
			Anchor:       referenceAnchor("idx-engmodel", term.ID),
			TargetAnchor: term.Anchor,
			ID:           id,
			Name:         term.Name,
			Kind:         "Engineering Model Term",
			Description:  strings.TrimSpace(term.Definition),
		})
	}
}

// addEntries lists the catalog entries of one kind, then their aliases.
// TRLC-LINKS: REQ-EMG-003
func (r *catalogReferences) addEntries(kind string, entries []model.CatalogEntry) {
	for _, e := range entries {
		id := strings.TrimSpace(e.ID)
		if id == "" || !r.claim(id) {
			continue
		}
		canonical := referenceAnchor("catalog", e.ID)
		r.entries = append(r.entries, asciidocReferenceEntry{
			Anchor:       referenceAnchor("idx-catalog", e.ID),
			TargetAnchor: canonical,
			ID:           id,
			Name:         textutil.NonEmpty(e.Name, e.ID),
			Kind:         "Catalog " + strings.TrimSpace(kind),
			Aliases:      textutil.UniqueSorted(e.Aliases),
			Description:  strings.TrimSpace(e.Definition),
		})
		r.addAliases(e, canonical)
	}
}

// addAliases lists the aliases of a catalog entry that no entry has claimed.
// TRLC-LINKS: REQ-EMG-003
func (r *catalogReferences) addAliases(e model.CatalogEntry, canonical string) {
	for _, alias := range textutil.UniqueSorted(e.Aliases) {
		alias = strings.TrimSpace(alias)
		if alias == "" || !r.claim(alias) {
			continue
		}
		r.entries = append(r.entries, asciidocReferenceEntry{
			Anchor:       referenceAnchor("idx-catalog", alias),
			TargetAnchor: canonical,
			ID:           alias,
			Name:         alias,
			Kind:         "Catalog Alias",
			Description:  aliasDescription(e),
		})
	}
}

// TRLC-LINKS: REQ-EMG-003
func catalogEntryIDSet(doc model.CatalogDocument) map[string]bool {
	out := map[string]bool{}
	for _, entries := range catalogEntryLists(doc) {
		for _, e := range entries {
			if id := strings.ToUpper(strings.TrimSpace(e.ID)); id != "" {
				out[id] = true
			}
		}
	}
	return out
}

// catalogEntryLists is every list of catalog entries, in catalog order.
// TRLC-LINKS: REQ-EMG-003
func catalogEntryLists(doc model.CatalogDocument) [][]model.CatalogEntry {
	c := doc.Catalog
	return [][]model.CatalogEntry{c.Systems, c.FunctionalGroups, c.FunctionalUnits, c.ReferencedElements, c.Actors,
		c.AttackVectors, c.Events, c.States, c.Features, c.Modes, c.Conditions, c.DataTerms}
}

// TRLC-LINKS: REQ-EMG-003
func aliasDescription(e model.CatalogEntry) string {
	name := textutil.NonEmpty(strings.TrimSpace(e.Name), strings.TrimSpace(e.ID))
	def := strings.TrimSpace(e.Definition)
	if def == "" {
		return "Same meaning as " + name + "."
	}
	return def
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildRuntimeReferences(in []codescan.RuntimeItem) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	seen := map[string]bool{}
	for _, r := range in {
		id := strings.TrimSpace(r.Name)
		kind := strings.TrimSpace(r.Kind)
		key := kind + "|" + id + "|" + r.Source
		if id == "" || seen[key] {
			continue
		}
		seen[key] = true
		owner := textutil.NonEmpty(strings.TrimSpace(r.Owner), "unresolved")
		source := codescan.SanitizeSourcePath(r.Source)
		desc := trimmedOr(r.Description, fmt.Sprintf("Inferred runtime %s owned by %s from %s.", textutil.NonEmpty(kind, "element"), owner, textutil.NonEmpty(source, "unknown source")))
		out = append(out, asciidocReferenceEntry{
			Anchor:      referenceAnchor("rt", kind+"-"+id),
			ID:          id,
			Name:        id,
			Kind:        "Runtime " + kind,
			Owner:       owner,
			Description: desc,
			Source:      source,
		})
	}
	sortReferenceEntries(out)
	return firstEntries(out, 80)
}

// firstEntries is at most the first limit entries.
// TRLC-LINKS: REQ-EMG-003
func firstEntries(entries []asciidocReferenceEntry, limit int) []asciidocReferenceEntry {
	if len(entries) > limit {
		return entries[:limit]
	}
	return entries
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildCodeReferences(in []codescan.CodeItem) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	seen := map[string]bool{}
	for _, c := range in {
		id := codescan.ItemDisplayName(c)
		key := c.Kind + "|" + id + "|" + c.Source
		if id == "" || seen[key] {
			continue
		}
		seen[key] = true
		owner := textutil.NonEmpty(strings.TrimSpace(c.Owner), "unresolved")
		source := codescan.SanitizeSourcePath(c.Source)
		out = append(out, asciidocReferenceEntry{
			Anchor:      referenceAnchor("code", c.Kind+"-"+id+"-"+source),
			ID:          id,
			Name:        id,
			Kind:        "Code " + c.Kind,
			Owner:       owner,
			Description: codeReferenceDescription(c, owner, source),
			Source:      source,
		})
	}
	sortReferenceEntries(out)
	return firstEntries(out, 120)
}

// codeReferenceDescription is the description of a code item, or the sentence
// inferred from its kind, owner and source.
// TRLC-LINKS: REQ-EMG-003
func codeReferenceDescription(c codescan.CodeItem, owner, source string) string {
	if desc := strings.TrimSpace(c.Description); desc != "" {
		return desc
	}
	from := textutil.NonEmpty(source, "unknown source")
	switch strings.TrimSpace(c.Kind) {
	case "symbol":
		return fmt.Sprintf("Inferred code symbol owned by %s from %s.", owner, from)
	case "source_file":
		return fmt.Sprintf("Inferred source file owned by %s from %s.", owner, from)
	default:
		return fmt.Sprintf("Inferred %s dependency owned by %s from %s.", textutil.NonEmpty(strings.TrimSpace(c.Kind), "code"), owner, from)
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-003
func buildVerificationReferences(in []codescan.VerificationCheck) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	seen := map[string]bool{}
	for _, v := range in {
		id := strings.TrimSpace(v.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		out = append(out, asciidocReferenceEntry{
			Anchor:       referenceAnchor("idx-ver", id),
			TargetAnchor: referenceAnchor("verify", id),
			ID:           id,
			Name:         textutil.NonEmpty(strings.TrimSpace(v.Name), id),
			Kind:         textutil.NonEmpty(strings.TrimSpace(v.Kind), "test"),
			Status:       textutil.NonEmpty(strings.TrimSpace(v.Status), "not-run"),
			Description:  verificationReferenceDescription(v),
			Source:       verificationReferenceSource(v.Evidence),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

// verificationReferenceSource is the first evidence path, with the count of
// the others, or n/a without evidence.
// TRLC-LINKS: REQ-EMG-003
func verificationReferenceSource(evidence []string) string {
	source := "n/a"
	if len(evidence) > 0 {
		source = codescan.SanitizeSourcePath(strings.TrimSpace(evidence[0]))
	}
	if len(evidence) > 1 {
		source = fmt.Sprintf("%s (+%d)", source, len(evidence)-1)
	}
	return source
}

// verificationReferenceDescription is the description of a check, or the
// sentence inferred from its kind and status.
// TRLC-LINKS: REQ-EMG-003
func verificationReferenceDescription(v codescan.VerificationCheck) string {
	if desc := strings.TrimSpace(v.Description); desc != "" {
		return desc
	}
	return fmt.Sprintf("Inferred %s verification check with status %s.", textutil.NonEmpty(strings.TrimSpace(v.Kind), "test"), textutil.NonEmpty(strings.TrimSpace(v.Status), "not-run"))
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildTermsFromCatalog(doc model.CatalogDocument) []asciidocTerm {
	out := []asciidocTerm{}
	out = append(out, builtInEngineeringModelTerms()...)
	for _, entries := range catalogEntryLists(doc) {
		for _, e := range entries {
			out = append(out, catalogTerm(e))
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(out[i].Name))
		rightName := strings.ToLower(strings.TrimSpace(out[j].Name))
		return cmp.Or(cmp.Compare(leftName, rightName), cmp.Compare(out[i].ID, out[j].ID)) < 0
	})
	return out
}

// catalogTerm is the term of a catalog entry.
// TRLC-LINKS: REQ-EMG-003
func catalogTerm(e model.CatalogEntry) asciidocTerm {
	return asciidocTerm{
		Anchor:      referenceAnchor("catalog", e.ID),
		IndexAnchor: referenceAnchor("idx-catalog", e.ID),
		ID:          strings.TrimSpace(e.ID),
		Name:        textutil.NonEmpty(strings.TrimSpace(e.Name), strings.TrimSpace(e.ID)),
		Definition:  strings.TrimSpace(e.Definition),
		Aliases:     textutil.UniqueSorted(e.Aliases),
	}
}

// engineeringModelTerms are the terms of the engineering model itself, defined
// once as the ID, the name and the definition.
var engineeringModelTerms = []struct{ id, name, definition string }{
	{"TERM-FUNCTIONAL-GROUP", "functional group",
		"A major authored capability area that groups related functional units."},
	{"TERM-FUNCTIONAL-UNIT", "functional unit",
		"An authored working unit inside a functional group that owns specific behavior."},
	{"TERM-RUNTIME-ELEMENT", "runtime element",
		"An inferred runtime realization element discovered from infrastructure and deployment sources."},
	{"TERM-CODE-ELEMENT", "code element",
		"An inferred code structure or ownership element discovered from source trees and build metadata."},
	{"TERM-TRACE-MARKER", "trace marker",
		"Source-level marker such as TRLC-LINKS or ENGMODEL-LINKS used to connect declarations to requirements and model entities."},
	{"TERM-MODEL", "model",
		"The authored architecture model root that composes functional structure, relationships, inference hints, and views."},
	{"TERM-BUNDLE", "model bundle",
		"Loaded architecture, catalog, and companion documents resolved as one working model context."},
	{"TERM-CATALOG", "catalog",
		"Controlled vocabulary document used to normalize model, requirement, and generated-document terminology."},
	{"TERM-CATALOG-ENTRY", "catalog entry",
		"A stable catalog term with a definition and aliases for linting, linking, and generated references."},
	{"TERM-LINT-RUN", "lint run",
		"Requirement lint configuration that controls parsing and quality checks for requirement text."},
	{"TERM-VALIDATION", "validation",
		"Model quality gate that checks authored documents, references, relationship semantics, and requirement lint results."},
	{"TERM-VALIDATION-DIAGNOSTIC", "validation diagnostic",
		"Structured validation finding with code, severity, message, and optional source path."},
	{"TERM-DESIGN", "design document",
		"Authored design narrative organized by model entities and architecture views."},
	{"TERM-DESIGN-VIEW", "design view",
		"View-scoped design narrative attached to an authored functional group or functional unit."},
	{"TERM-DECISION", "architecture decision",
		"Authored architecture decision record with status, context, decision text, and consequences."},
	{"TERM-INFERENCE-HINT", "inference hint",
		"Authored source and ownership configuration used to discover runtime, code, and verification evidence."},
	{"TERM-VIEW", "view",
		"Authored projection configuration that selects roots, entity kinds, mappings, audience, and abstraction."},
	{"TERM-ASCIIDOC-DOCUMENT", "AsciiDoc document",
		"Human-readable generated architecture publication document assembled from authored model, inferred evidence, diagrams, references, and decisions."},
	{"TERM-ASCIIDOC-SECTION", "AsciiDoc section",
		"Structured generated document section or row model used to render authored and inferred architecture content."},
	{"TERM-ASCIIDOC-DIAGRAM", "AsciiDoc diagram",
		"Generated diagram block, usually Mermaid, that visualizes architecture relationships in the AsciiDoc publication."},
	{"TERM-REFERENCE-INDEX", "reference index",
		"Generated index of authored, catalog, runtime, code, and verification references with stable anchors and backlinks."},
	{"TERM-SECURITY-CONTEXT", "security context",
		"Security-focused generated context view that groups owned functional units and shows external actors, references, flows, controls, and trust boundaries."},
	{"TERM-THREAT-MODEL-EXPORT", "threat model export",
		"Generated security artifact that translates authored architecture, flows, trust boundaries, threats, and mitigations into an external threat-model schema."},
	{"TERM-THREAT-DRAGON-DOCUMENT", "Threat Dragon document",
		"Threat Dragon JSON representation generated from the engineering model for STRIDE threat-model review."},
	{"TERM-OPEN-OTM-DOCUMENT", "Open OTM document",
		"Open Threat Model JSON representation generated from the engineering model for interoperable threat-model exchange."},
	{"TERM-STRUCTURIZR-DSL", "Structurizr DSL",
		"Generated Structurizr DSL workspace that projects authored architecture, relationships, dynamic views, and deployment metadata."},
	{"TERM-TRLC-PACKAGE", "TRLC package",
		"Generated TRLC model and requirements package used for formal requirement traceability processing."},
	{"TERM-LOBSTER-ACTIVITY-TRACE", "LOBSTER activity trace",
		"Generated LOBSTER activity JSON that links verification evidence back to formal requirement identifiers."},
	{"TERM-OSCAL-SSP", "OSCAL SSP",
		"Generated OSCAL system security plan that maps authored system metadata, components, controls, allocations, and evidence into SSP JSON."},
	{"TERM-OSCAL-ASSESSMENT-RESULTS", "OSCAL assessment results",
		"Generated OSCAL assessment results that summarize reviewed controls, verification findings, and modeled risks."},
	{"TERM-OSCAL-POAM", "OSCAL POA&M",
		"Generated OSCAL plan of action and milestones that maps modeled POA&M items and related risks into compliance JSON."},
	{"TERM-MCP-SERVER", "MCP server",
		"Model Context Protocol server that exposes engineering-model operations to AI agents through JSON-RPC tools."},
	{"TERM-MCP-TOOL", "MCP tool",
		"Named MCP operation with a constrained input schema and structured response payload."},
	{"TERM-MCP-TOOL-RESPONSE", "MCP tool response",
		"Structured MCP tool payload that includes success state, schema version, generated timestamp, and result data or validation error."},
	{"TERM-REPO-INDEX", "repository index",
		"Bounded first-party source index used to answer model-aware file, requirement, control, and threat lookup queries."},
	{"TERM-MCP-STDIO-TRANSPORT", "MCP stdio transport",
		"Content-Length framed standard input/output transport used by the MCP command-line server."},
	{"TERM-CLI-COMMAND", "CLI command",
		"Command-line entrypoint that coordinates loading, validation, generation, and file output for an engineering-model workflow."},
	{"TERM-GENERATION-WORKFLOW", "generation workflow",
		"Orchestrated run that combines model loading, validation, projection, rendering, and artifact emission."},
	{"TERM-SOURCE-BLOCK", "source block",
		"Stable source reference block that records file path, optional line span, kind, summary, and linked entity IDs."},
	{"TERM-REFERENCED-ELEMENT", "referenced element",
		"An architecture-relevant external, platform, or third-party dependency represented by role."},
	{"TERM-ACTOR", "actor",
		"A person or operational role that interacts with functional units."},
	{"TERM-INTERFACE", "interface",
		"An authored interface boundary (for example API, channel, endpoint, or contract) owned by a functional unit."},
	{"TERM-DATA-OBJECT", "data object",
		"An authored data artifact or contract shape traced across flow, interface, and implementation boundaries."},
	{"TERM-DEPLOYMENT-TARGET", "deployment target",
		"An authored deployment destination such as environment, cluster, namespace, or registry scope."},
	{"TERM-CONTROL", "control",
		"An authored security, policy, or operational control used to constrain behavior and reduce risk."},
	{"TERM-COMPLIANCE-MAPPING", "compliance mapping",
		"Authored mapping that links a model control to selected compliance controls, model scope, implementation status, evidence, and responsible roles."},
	{"TERM-RISK", "risk",
		"Authored risk record with likelihood, impact, response, scope, controls, and evidence."},
	{"TERM-POAM-ITEM", "POA&M item",
		"Plan of action and milestones item linked to a modeled risk and supporting artifacts."},
	{"TERM-EVIDENCE", "evidence",
		"A file, artifact, or observation used to support control, risk, threat, or verification claims."},
	{"TERM-TRUST-BOUNDARY", "trust boundary",
		"An authored trust separation zone that marks policy, identity, or security control boundaries."},
	{"TERM-STATE", "state",
		"An authored lifecycle state used to model transition behavior, guards, and event-driven progression."},
	{"TERM-EVENT", "event",
		"An authored trigger signal that drives transitions, flow progress, or asynchronous outcomes."},
	{"TERM-FLOW", "flow",
		"An authored causal interaction sequence from user/system intent to outcome, represented as ordered steps."},
	{"TERM-FLOW-STEP", "flow step",
		"A single authored step in a flow that captures action, data in/out, references, and normal/error/async transitions."},
	{"TERM-ATTACK-VECTOR", "attack vector",
		"A technical misuse or attack path that targets functional, referenced, or runtime elements."},
	{"TERM-THREAT-SCENARIO", "threat scenario",
		"Authored threat narrative that connects attack vectors, scope, flows, controls, risks, and verification evidence."},
	{"TERM-THREAT-ASSUMPTION", "threat assumption",
		"Authored security assumption that records scope, rationale, owner, status, and evidence."},
	{"TERM-THREAT-OUT-OF-SCOPE", "threat out-of-scope decision",
		"Authored decision that excludes a threat concern from current scope with reason, owner, expiry, and evidence."},
	{"TERM-THREAT-MITIGATION", "threat mitigation",
		"Authored mitigation record that links a threat scenario to a control, effectiveness, verification, and evidence."},
	{"TERM-AUTHORED-MAPPING", "authored mapping",
		"An explicit relationship declared in architecture inputs between authored or referenced elements."},
	{"TERM-INFERRED-MAPPING", "inferred mapping",
		"A discovered relationship that links inferred runtime/code elements upward to authored design."},
	{"TERM-UPWARD-LINKING", "upward linking",
		"Rule where runtime and code elements point to stable functional groups/units; authored architecture does not depend on inferred IDs."},
	{"TERM-REQUIREMENT", "requirement",
		"A structured requirement statement that defines expected system behavior and maps to functional ownership."},
	{"TERM-VERIFICATION-CHECK", "verification check",
		"An inferred or authored verification artifact that validates requirement behavior with test evidence."},
	{"TERM-CONTROL-VERIFICATION", "control verification",
		"Authored control verification record with method, status, threat/risk scope, findings, and evidence."},
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func builtInEngineeringModelTerms() []asciidocTerm {
	terms := make([]asciidocTerm, 0, len(engineeringModelTerms))
	for _, t := range engineeringModelTerms {
		terms = append(terms, asciidocTerm{
			Anchor:      referenceAnchor("engmodel", t.id),
			IndexAnchor: referenceAnchor("idx-engmodel", t.id),
			ID:          t.id,
			Name:        t.name,
			Definition:  t.definition,
		})
	}
	return terms
}
