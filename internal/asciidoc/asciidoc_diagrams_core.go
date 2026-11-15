// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"cmp"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/render/diagramstyle"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// TRLC-LINKS: REQ-EMG-003
func appendMermaidClassDefs(lines []string) []string {
	return append(lines, diagramstyle.MermaidClassDefsWithIndent("  ")...)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func buildRequirementAlignmentMermaid(reqs []model.Requirement, labels map[string]string) string {
	lines := []string{"flowchart LR"}
	for _, r := range reqs {
		reqNode := "REQ_" + codescan.SanitizeNode(r.ID)
		lines = append(lines, "  "+reqNode+"[\""+escapeMermaidLabel(r.ID)+"\"]:::requirement")
		for _, u := range textutil.UniqueSorted(r.AppliesTo) {
			target := "UNIT_" + codescan.SanitizeNode(u)
			label := textutil.NonEmpty(labels[u], u)
			lines = append(lines, "  "+target+"[\""+escapeMermaidLabel(label)+"\"]:::functional_unit")
			lines = append(lines, "  "+reqNode+" --> "+target)
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

// mermaidLines collects the lines of a Mermaid diagram.
type mermaidLines []string

// add appends a line.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (m *mermaidLines) add(line string) {
	*m = append(*m, line)
}

// addf appends a formatted line.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (m *mermaidLines) addf(format string, args ...any) {
	*m = append(*m, fmt.Sprintf(format, args...))
}

// requirementTargets are the functional units a requirement applies to.
type requirementTargets struct {
	ID      string
	Targets map[string]bool
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildRequirementAlignmentCompactTable(reqs []model.Requirement) string {
	byReq := requirementTargetSets(reqs)
	fuIDs := targetUnitIDs(byReq)
	if len(byReq) == 0 || len(fuIDs) == 0 {
		return "No authored requirement-to-unit mappings were found."
	}
	rows := [][]string{append([]string{"Requirement"}, fuIDs...)}
	for _, r := range byReq {
		rows = append(rows, r.matrixRow(fuIDs))
	}
	return publicationMatrixSVG(rows, false)
}

// targetUnitIDs is the sorted union of the units the requirements apply to.
// TRLC-LINKS: REQ-EMG-003
func targetUnitIDs(byReq []requirementTargets) []string {
	fuSet := map[string]bool{}
	for _, r := range byReq {
		for fu := range r.Targets {
			fuSet[fu] = true
		}
	}
	return textutil.UniqueSorted(keysFromSet(fuSet))
}

// matrixRow is the requirement ID followed by a mark for each unit it applies to.
// TRLC-LINKS: REQ-EMG-003
func (r requirementTargets) matrixRow(fuIDs []string) []string {
	row := []string{r.ID}
	for _, fu := range fuIDs {
		row = append(row, markIf(r.Targets[fu], "X"))
	}
	return row
}

// markIf is the mark when the condition holds, else empty.
// TRLC-LINKS: REQ-EMG-003
func markIf(condition bool, mark string) string {
	if condition {
		return mark
	}
	return ""
}

// requirementTargetSets lists each distinct requirement ID once, sorted, with
// the units it applies to.
// TRLC-LINKS: REQ-EMG-003
func requirementTargetSets(reqs []model.Requirement) []requirementTargets {
	seen := map[string]bool{}
	byReq := make([]requirementTargets, 0, len(reqs))
	for _, r := range reqs {
		rid := strings.TrimSpace(r.ID)
		if rid == "" || seen[rid] {
			continue
		}
		seen[rid] = true
		targets := map[string]bool{}
		for _, fu := range textutil.UniqueSorted(r.AppliesTo) {
			targets[fu] = true
		}
		byReq = append(byReq, requirementTargets{ID: rid, Targets: targets})
	}
	sort.SliceStable(byReq, func(i, j int) bool { return byReq[i].ID < byReq[j].ID })
	return byReq
}

// groupBox is a functional group with its units, drawn as one Mermaid subgraph.
type groupBox struct {
	id    string
	label string
	units []model.FunctionalUnit
}

// groupBoxStyle is how a diagram draws its group boxes.
type groupBoxStyle struct {
	idPrefix  string
	indent    string
	direction string
}

// functionalGroupBoxes lists the groups that have units, in authored order,
// then the units without a group. The label of a group is looked up under the
// key labelKey makes of its ID.
// TRLC-LINKS: REQ-EMG-003
func functionalGroupBoxes(a model.AuthoredArchitecture, labelKey func(string) string) []groupBox {
	labelByID := map[string]string{}
	order := make([]string, 0, len(a.FunctionalGroups))
	for _, fg := range a.FunctionalGroups {
		labelByID[labelKey(fg.ID)] = textutil.NonEmpty(fg.Name, fg.ID)
		order = append(order, strings.TrimSpace(fg.ID))
	}
	unitsByGroup := map[string][]model.FunctionalUnit{}
	for _, fu := range a.FunctionalUnits {
		gid := strings.TrimSpace(fu.Group)
		unitsByGroup[gid] = append(unitsByGroup[gid], fu)
	}
	if len(unitsByGroup[""]) > 0 {
		order = append(order, "")
		labelByID[""] = "Unassigned Functional Units"
	}
	boxes := make([]groupBox, 0, len(order))
	for _, gid := range order {
		if units := unitsByGroup[gid]; len(units) > 0 {
			boxes = append(boxes, groupBox{id: gid, label: labelByID[gid], units: units})
		}
	}
	return boxes
}

// writeGroupBoxes draws each group box with its units.
// TRLC-LINKS: REQ-EMG-003
func writeGroupBoxes(m *mermaidLines, boxes []groupBox, style groupBoxStyle) {
	for _, box := range boxes {
		boxID := style.idPrefix + codescan.SanitizeNode(box.id)
		if box.id == "" {
			boxID = style.idPrefix + "UNASSIGNED"
		}
		m.addf("%ssubgraph %s[\"%s\"]", style.indent, boxID, escapeMermaidLabel(box.label))
		m.addf("%s  direction %s", style.indent, style.direction)
		for _, fu := range box.units {
			m.addf("%s  %s[\"%s\"]:::functional_unit", style.indent, "FU_"+codescan.SanitizeNode(fu.ID), escapeMermaidLabel(textutil.NonEmpty(fu.Name, fu.ID)))
		}
		m.add(style.indent + "end")
		m.addf("%sstyle %s fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;", style.indent, boxID)
	}
}

// writeReferencedElements draws the referenced elements as nodes.
// TRLC-LINKS: REQ-EMG-003
func writeReferencedElements(m *mermaidLines, refs []model.ReferencedElement) {
	for _, ref := range refs {
		m.addf("  %s[\"%s\"]:::referenced_element", "REF_"+codescan.SanitizeNode(ref.ID), escapeMermaidLabel(textutil.NonEmpty(ref.Name, ref.ID)))
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalContextMermaid(a model.AuthoredArchitecture) string {
	lines := mermaidLines{"flowchart TB"}
	for _, act := range a.Actors {
		lines.addf("  %s((\"%s\")):::actor", "ACT_"+codescan.SanitizeNode(act.ID), escapeMermaidLabel(textutil.NonEmpty(act.Name, act.ID)))
	}
	writeReferencedElements(&lines, a.ReferencedElements)
	lines.add("  subgraph SYSTEM_BOUNDARY[\"System Boundary (Internal)\"]")
	lines.add("    direction TB")
	writeGroupBoxes(&lines, functionalGroupBoxes(a, strings.TrimSpace), groupBoxStyle{idPrefix: "FGCTX_", indent: "    ", direction: "LR"})
	lines.add("  end")
	lines.add("  style SYSTEM_BOUNDARY fill:#f5f5f5,stroke:#424242,stroke-width:1px;")
	writeContextEdges(&lines, a)
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

// contextNodes classifies the IDs of a context diagram as internal or external.
type contextNodes struct {
	internal map[string]bool
	external map[string]string
}

// newContextNodes classifies the groups and units as internal, the actors and
// referenced elements as external (with their node ID prefix).
// TRLC-LINKS: REQ-EMG-003
func newContextNodes(a model.AuthoredArchitecture) contextNodes {
	n := contextNodes{internal: map[string]bool{}, external: map[string]string{}}
	for _, fg := range a.FunctionalGroups {
		n.internal[strings.TrimSpace(fg.ID)] = true
	}
	for _, fu := range a.FunctionalUnits {
		n.internal[strings.TrimSpace(fu.ID)] = true
	}
	for _, act := range a.Actors {
		n.external[strings.TrimSpace(act.ID)] = "ACT_"
	}
	for _, ref := range a.ReferencedElements {
		n.external[strings.TrimSpace(ref.ID)] = "REF_"
	}
	return n
}

// edge is the diagram edge of a mapping between an external and an internal
// node, in either direction.
// TRLC-LINKS: REQ-EMG-003
func (n contextNodes) edge(m model.Mapping) (string, bool) {
	from := strings.TrimSpace(m.From)
	to := strings.TrimSpace(m.To)
	fromPrefix, fromExternal := n.external[from]
	toPrefix, toExternal := n.external[to]
	var fromNode, toNode string
	switch {
	case fromExternal && n.internal[to]:
		fromNode, toNode = fromPrefix+codescan.SanitizeNode(from), functionalContextInternalNodeID(to)
	case n.internal[from] && toExternal:
		fromNode, toNode = functionalContextInternalNodeID(from), toPrefix+codescan.SanitizeNode(to)
	default:
		return "", false
	}
	label := textutil.NonEmpty(strings.TrimSpace(m.Description), strings.TrimSpace(m.Type))
	return fmt.Sprintf("  %s -->|%s| %s", fromNode, escapeMermaidLabel(label), toNode), true
}

// writeContextEdges draws the mappings between external and internal nodes, each edge once.
// TRLC-LINKS: REQ-EMG-003
func writeContextEdges(m *mermaidLines, a model.AuthoredArchitecture) {
	nodes := newContextNodes(a)
	seen := map[string]bool{}
	for _, mapping := range a.Mappings {
		if edge, ok := nodes.edge(mapping); ok && !seen[edge] {
			m.add(edge)
			seen[edge] = true
		}
	}
}

// TRLC-LINKS: REQ-EMG-003
func keysFromSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

// TRLC-LINKS: REQ-EMG-003
func escapeTableCell(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

// TRLC-LINKS: REQ-EMG-003
func functionalContextInternalNodeID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "FG-") {
		return "FGCTX_" + codescan.SanitizeNode(id)
	}
	return "FU_" + codescan.SanitizeNode(id)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalDecompositionMermaid(a model.AuthoredArchitecture) string {
	// Left-to-right ranks stack sibling units vertically rather than forming a wide strip.
	lines := []string{"flowchart LR"}
	lines = append(lines, "  SYS[\"System\"]:::system_boundary")
	for _, fg := range a.FunctionalGroups {
		gn := "FG_" + codescan.SanitizeNode(fg.ID)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_group", gn, escapeMermaidLabel(textutil.NonEmpty(fg.Name, fg.ID))))
		lines = append(lines, fmt.Sprintf("  SYS -->|contains| %s", gn))
	}
	for _, fu := range a.FunctionalUnits {
		un := "FU_" + codescan.SanitizeNode(fu.ID)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_unit", un, escapeMermaidLabel(textutil.NonEmpty(fu.Name, fu.ID))))
		if strings.TrimSpace(fu.Group) != "" {
			lines = append(lines, fmt.Sprintf("  FG_%s -->|contains| %s", codescan.SanitizeNode(fu.Group), un))
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

// fgColumn is a functional group with its units, a column of the matrix.
type fgColumn struct {
	ID    string
	Name  string
	Units []model.FunctionalUnit
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalManhattanTable(a model.AuthoredArchitecture) string {
	cols := functionalColumns(a)
	maxRows := 0
	for _, c := range cols {
		maxRows = max(maxRows, len(c.Units))
	}
	rows := make([][]string, maxRows+1)
	for row := range rows {
		rows[row] = make([]string, len(cols))
		for col, c := range cols {
			rows[row][col] = manhattanCell(c, row, maxRows)
		}
	}
	return publicationMatrixSVG(rows, true)
}

// functionalColumns is one column per group, with its units sorted by name.
// TRLC-LINKS: REQ-EMG-003
func functionalColumns(a model.AuthoredArchitecture) []fgColumn {
	cols := make([]fgColumn, 0, len(a.FunctionalGroups))
	for _, fg := range a.FunctionalGroups {
		cols = append(cols, fgColumn{
			ID:   strings.TrimSpace(fg.ID),
			Name: textutil.NonEmpty(strings.TrimSpace(fg.Name), strings.TrimSpace(fg.ID)),
		})
	}
	indexByFG := map[string]int{}
	for i, c := range cols {
		indexByFG[c.ID] = i
	}
	for _, fu := range a.FunctionalUnits {
		if idx, ok := indexByFG[strings.TrimSpace(fu.Group)]; ok {
			cols[idx].Units = append(cols[idx].Units, fu)
		}
	}
	for i := range cols {
		sortUnitsByName(cols[i].Units)
	}
	return cols
}

// sortUnitsByName sorts the units by name (or ID when unnamed), then ID.
// TRLC-LINKS: REQ-EMG-003
func sortUnitsByName(units []model.FunctionalUnit) {
	label := func(u model.FunctionalUnit) string {
		return textutil.NonEmpty(strings.TrimSpace(u.Name), strings.TrimSpace(u.ID))
	}
	sort.SliceStable(units, func(left, right int) bool {
		return cmp.Or(cmp.Compare(label(units[left]), label(units[right])), cmp.Compare(units[left].ID, units[right].ID)) < 0
	})
}

// manhattanCell is the text of a column in a row: the units stack up from the
// bottom, above the group name in the last row.
// TRLC-LINKS: REQ-EMG-003
func manhattanCell(c fgColumn, row, maxRows int) string {
	if row == maxRows {
		return c.Name
	}
	unit := row - (maxRows - len(c.Units))
	if unit < 0 {
		return ""
	}
	return textutil.NonEmpty(c.Units[unit].Name, c.Units[unit].ID)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalCollaborationMermaid(a model.AuthoredArchitecture) string {
	lines := mermaidLines{"flowchart LR"}
	writeGroupBoxes(&lines, functionalGroupBoxes(a, func(id string) string { return id }), groupBoxStyle{idPrefix: "FGBOX_", indent: "  ", direction: "TB"})
	writeReferencedElements(&lines, a.ReferencedElements)
	seen := map[string]bool{}
	for _, m := range a.Mappings {
		if edge, ok := dependsOnEdge(m); ok && !seen[edge] {
			seen[edge] = true
			lines.add(edge)
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

// dependsOnEdge is the diagram edge of a depends_on mapping from a unit to a
// unit or a referenced element.
// TRLC-LINKS: REQ-EMG-003
func dependsOnEdge(m model.Mapping) (string, bool) {
	if m.Type != "depends_on" {
		return "", false
	}
	return unitDependencyEdge(strings.TrimSpace(m.From), strings.TrimSpace(m.To))
}

// unitDependencyEdge is the depends_on edge from a unit to a unit or a
// referenced element, by their IDs.
// TRLC-LINKS: REQ-EMG-003
func unitDependencyEdge(from, to string) (string, bool) {
	if !strings.HasPrefix(from, "FU-") {
		return "", false
	}
	switch {
	case strings.HasPrefix(to, "FU-"):
		return fmt.Sprintf("  FU_%s -->|depends_on| FU_%s", codescan.SanitizeNode(from), codescan.SanitizeNode(to)), true
	case strings.HasPrefix(to, "REF-"):
		return fmt.Sprintf("  FU_%s -->|depends_on| REF_%s", codescan.SanitizeNode(from), codescan.SanitizeNode(to)), true
	}
	return "", false
}

// realizationEvidence is the inferred runtime and code elements of a model.
type realizationEvidence struct {
	runtime []codescan.RuntimeItem
	code    []codescan.CodeItem
}

// groupDependencies is what a group dependency diagram shows: the units of
// the group, the units and referenced elements they depend on, and the edges.
type groupDependencies struct {
	groupID        string
	groupLabelByID map[string]string
	unitLabelByID  map[string]string
	unitGroupByID  map[string]string
	refLabelByID   map[string]string
	localUnits     []model.FunctionalUnit
	localUnitSet   map[string]bool
	externalFUs    map[string]bool
	refs           map[string]bool
	edges          []string
}

// newGroupDependencies indexes the labels and the units of the group.
// TRLC-LINKS: REQ-EMG-003
func newGroupDependencies(a model.AuthoredArchitecture, groupID string) *groupDependencies {
	d := &groupDependencies{
		groupID:        groupID,
		groupLabelByID: map[string]string{},
		unitLabelByID:  map[string]string{},
		unitGroupByID:  map[string]string{},
		refLabelByID:   map[string]string{},
		localUnits:     []model.FunctionalUnit{},
		localUnitSet:   map[string]bool{},
		externalFUs:    map[string]bool{},
		refs:           map[string]bool{},
	}
	for _, fg := range a.FunctionalGroups {
		d.groupLabelByID[strings.TrimSpace(fg.ID)] = textutil.NonEmpty(fg.Name, fg.ID)
	}
	for _, fu := range a.FunctionalUnits {
		id := strings.TrimSpace(fu.ID)
		d.unitLabelByID[id] = textutil.NonEmpty(fu.Name, fu.ID)
		d.unitGroupByID[id] = strings.TrimSpace(fu.Group)
		if strings.TrimSpace(fu.Group) == groupID {
			d.localUnits = append(d.localUnits, fu)
			d.localUnitSet[id] = true
		}
	}
	sort.SliceStable(d.localUnits, func(i, j int) bool {
		return d.localUnits[i].ID < d.localUnits[j].ID
	})
	for _, ref := range a.ReferencedElements {
		d.refLabelByID[strings.TrimSpace(ref.ID)] = textutil.NonEmpty(ref.Name, ref.ID)
	}
	return d
}

// collectEdges records the depends_on edges from the units of the group.
// TRLC-LINKS: REQ-EMG-003
func (d *groupDependencies) collectEdges(mappings []model.Mapping) {
	seen := map[string]bool{}
	for _, m := range mappings {
		from := strings.TrimSpace(m.From)
		if strings.TrimSpace(m.Type) != "depends_on" || !d.localUnitSet[from] {
			continue
		}
		to := strings.TrimSpace(m.To)
		edge, ok := unitDependencyEdge(from, to)
		if !ok || seen[edge] {
			continue
		}
		d.noteTarget(to)
		seen[edge] = true
		d.edges = append(d.edges, edge)
	}
}

// noteTarget records a dependency target outside the group as an external
// unit or a referenced element.
// TRLC-LINKS: REQ-EMG-003
func (d *groupDependencies) noteTarget(to string) {
	if strings.HasPrefix(to, "FU-") && !d.localUnitSet[to] {
		d.externalFUs[to] = true
	}
	if strings.HasPrefix(to, "REF-") {
		d.refs[to] = true
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalGroupDependencyMermaid(a model.AuthoredArchitecture, groupID string, evidence realizationEvidence) string {
	deps := newGroupDependencies(a, strings.TrimSpace(groupID))
	if len(deps.localUnits) == 0 {
		return ""
	}
	deps.collectEdges(a.Mappings)
	if len(deps.edges) == 0 {
		return ""
	}
	diagram := &groupDependencyDiagram{
		deps:           deps,
		runtimeByOwner: runtimeEvidenceByOwner(evidence.runtime, deps.localUnitSet),
		codeByOwner:    codeEvidenceByOwner(evidence.code, deps.localUnitSet),
		lines:          mermaidLines{"flowchart LR"},
		definedCode:    map[string]bool{},
	}
	diagram.writeLocalGroup()
	diagram.writeExternalGroups()
	for _, id := range keysSorted(deps.refs) {
		diagram.lines.addf("  REF_%s[\"%s\"]:::referenced_element", codescan.SanitizeNode(id), escapeMermaidLabel(textutil.NonEmpty(deps.refLabelByID[id], id)))
	}
	diagram.lines = append(diagram.lines, deps.edges...)
	diagram.lines = appendMermaidClassDefs(diagram.lines)
	return strings.Join(diagram.lines, "\n")
}

// groupDependencyDiagram draws a group dependency diagram.
type groupDependencyDiagram struct {
	deps           *groupDependencies
	runtimeByOwner map[string][]string
	codeByOwner    map[string][]string
	lines          mermaidLines
	definedCode    map[string]bool
}

// runtimeEvidenceByOwner lists the runtime element labels of each unit in scope.
// TRLC-LINKS: REQ-EMG-003
func runtimeEvidenceByOwner(runtime []codescan.RuntimeItem, scope map[string]bool) map[string][]string {
	byOwner := map[string][]string{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		label := textutil.NonEmpty(strings.TrimSpace(r.Name), strings.TrimSpace(r.Kind))
		if scope[owner] && label != "" {
			byOwner[owner] = append(byOwner[owner], label)
		}
	}
	for owner, items := range byOwner {
		byOwner[owner] = textutil.UniqueSorted(items)
	}
	return byOwner
}

// codeEvidenceByOwner lists the grouped code element labels of each unit in scope.
// TRLC-LINKS: REQ-EMG-003
func codeEvidenceByOwner(code []codescan.CodeItem, scope map[string]bool) map[string][]string {
	rawByOwner := map[string][]string{}
	allRaw := []string{}
	for _, c := range code {
		owner, elem, ok := scopedCodeElement(c, scope)
		if ok {
			rawByOwner[owner] = append(rawByOwner[owner], elem)
			allRaw = append(allRaw, elem)
		}
	}
	byOwner := map[string][]string{}
	lookup := codescan.EvidenceLabelLookup(allRaw)
	for owner, items := range rawByOwner {
		byOwner[owner] = codescan.GroupedFileLabels(items, lookup)
	}
	return byOwner
}

// scopedCodeElement is the owner and evidence element of a source file or
// symbol of a unit in scope, other than verification code.
// TRLC-LINKS: REQ-EMG-003
func scopedCodeElement(c codescan.CodeItem, scope map[string]bool) (owner, elem string, ok bool) {
	owner = strings.TrimSpace(c.Owner)
	if codescan.IsVerificationCodeItem(c) || !scope[owner] {
		return "", "", false
	}
	if kind := strings.TrimSpace(c.Kind); kind != "source_file" && kind != "symbol" {
		return "", "", false
	}
	elem = codescan.ItemEvidenceElement(c)
	return owner, elem, strings.TrimSpace(elem) != ""
}

// writeLocalGroup draws the group box with its units and their evidence.
// TRLC-LINKS: REQ-EMG-003
func (g *groupDependencyDiagram) writeLocalGroup() {
	d := g.deps
	boxID := "FGDEP_" + codescan.SanitizeNode(d.groupID)
	if d.groupID == "" {
		boxID = "FGDEP_UNASSIGNED"
	}
	g.lines.addf("  subgraph %s[\"%s\"]", boxID, escapeMermaidLabel(textutil.NonEmpty(d.groupLabelByID[d.groupID], d.groupID)))
	g.lines.add("    direction TB")
	for _, fu := range d.localUnits {
		g.lines.addf("    %s[\"%s\"]:::functional_unit", "FU_"+codescan.SanitizeNode(fu.ID), escapeMermaidLabel(textutil.NonEmpty(fu.Name, fu.ID)))
	}
	for _, owner := range keysSorted(d.localUnitSet) {
		g.writeOwnerEvidence(owner)
	}
	g.lines.add("  end")
	g.lines.addf("  style %s fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;", boxID)
}

// writeOwnerEvidence draws a runtime box for each runtime element of a unit,
// or an inferred one when only code evidence exists.
// TRLC-LINKS: REQ-EMG-003
func (g *groupDependencyDiagram) writeOwnerEvidence(owner string) {
	runtimeLabels := append([]string(nil), g.runtimeByOwner[owner]...)
	if len(runtimeLabels) == 0 && len(g.codeByOwner[owner]) > 0 {
		runtimeLabels = append(runtimeLabels, textutil.NonEmpty(g.deps.unitLabelByID[owner], owner)+" runtime (inferred)")
	}
	for idx, rt := range runtimeLabels {
		rtBox := "RT_" + codescan.SanitizeNode(owner+"-"+rt)
		g.lines.addf("    subgraph %s[\"%s\"]", rtBox, escapeMermaidLabel(rt))
		g.lines.add("      direction TB")
		if idx == 0 {
			g.writeCodeNodes(owner)
		}
		if idx > 0 || len(g.codeByOwner[owner]) == 0 {
			g.lines.addf("      %s[\"runtime evidence\"]:::runtime_element", "RTNODE_"+codescan.SanitizeNode(owner+"-"+rt))
		}
		g.lines.add("    end")
		g.lines.addf("    style %s fill:#b2ebf2,stroke:#00838f,stroke-width:1px,color:#006064;", rtBox)
		g.lines.addf("    FU_%s -->|runtime evidence| %s", codescan.SanitizeNode(owner), rtBox)
	}
}

// writeCodeNodes draws the code elements of a unit that no earlier box drew.
// TRLC-LINKS: REQ-EMG-003
func (g *groupDependencyDiagram) writeCodeNodes(owner string) {
	for _, label := range g.codeByOwner[owner] {
		codeNode := "CODE_" + codescan.SanitizeNode(codescan.EvidenceFileKey(label))
		if g.definedCode[codeNode] {
			continue
		}
		g.lines.addf("      %s[\"%s\"]:::code_element", codeNode, escapeMermaidLabel(label))
		g.definedCode[codeNode] = true
	}
}

// writeExternalGroups draws the units outside the group, in a box per group.
// TRLC-LINKS: REQ-EMG-003
func (g *groupDependencyDiagram) writeExternalGroups() {
	d := g.deps
	unitsByGroup := map[string][]string{}
	for _, id := range keysSorted(d.externalFUs) {
		gid := strings.TrimSpace(d.unitGroupByID[id])
		unitsByGroup[gid] = append(unitsByGroup[gid], id)
	}
	for _, gid := range keysSortedStringSlices(unitsByGroup) {
		boxID := "FGDEP_TARGET_" + codescan.SanitizeNode(gid)
		if gid == "" {
			boxID = "FGDEP_TARGET_UNASSIGNED"
		}
		g.lines.addf("  subgraph %s[\"%s\"]", boxID, escapeMermaidLabel(textutil.NonEmpty(d.groupLabelByID[gid], "External Functional Units")))
		g.lines.add("    direction TB")
		for _, id := range textutil.UniqueSorted(unitsByGroup[gid]) {
			g.lines.addf("    FU_%s[\"%s\"]:::functional_unit", codescan.SanitizeNode(id), escapeMermaidLabel(textutil.NonEmpty(d.unitLabelByID[id], id)))
		}
		g.lines.add("  end")
		g.lines.addf("  style %s fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;", boxID)
	}
}

// TRLC-LINKS: REQ-EMG-003
func keysSorted(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// TRLC-LINKS: REQ-EMG-003
func keysSortedStringSlices(m map[string][]string) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	sort.Strings(out)
	return out
}

// coverageSources are the inferred layers and labels a requirement coverage diagram draws from.
type coverageSources struct {
	runtime      []codescan.RuntimeItem
	code         []codescan.CodeItem
	verification []codescan.VerificationCheck
	labels       map[string]string
}

// coverageEvidence indexes the evidence of the requirements by unit and by requirement.
type coverageEvidence struct {
	rtByOwner      map[string][]string
	codeByOwnerReq map[string]map[string][]string
	checksByReq    map[string][]codescan.VerificationCheck
	labels         map[string]string
}

// coverageRequirement is a requirement with the node that draws it.
type coverageRequirement struct {
	req  model.Requirement
	node string
}

// coverageUnit is a unit a requirement applies to.
type coverageUnit struct {
	id      string
	label   string
	hasCode bool
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func buildRequirementCoverageMermaid(reqs []model.Requirement, src coverageSources) string {
	// Stack the evidence chain to fit portrait publication pages.
	lines := mermaidLines{`%%{init: {"flowchart": {"nodeSpacing": 70, "rankSpacing": 100}}}%%`, "flowchart TB"}
	evidence := coverageEvidence{
		rtByOwner:      runtimeNamesByOwner(src.runtime),
		codeByOwnerReq: codeByOwnerAndRequirement(src.code),
		checksByReq:    checksByRequirement(src.verification),
		labels:         src.labels,
	}
	for _, r := range reqs {
		evidence.writeRequirement(&lines, r)
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

// runtimeNamesByOwner lists the runtime element names of each resolved owner.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func runtimeNamesByOwner(runtime []codescan.RuntimeItem) map[string][]string {
	byOwner := map[string][]string{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		if owner == "" || owner == "unresolved" {
			continue
		}
		byOwner[owner] = append(byOwner[owner], textutil.NonEmpty(strings.TrimSpace(r.Name), strings.TrimSpace(r.Kind)))
	}
	return byOwner
}

// codeByOwnerAndRequirement lists the labels of the code symbols that
// implement each requirement, by owner, keeping the globally disambiguated
// file names but only the lines of the requirement.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func codeByOwnerAndRequirement(code []codescan.CodeItem) map[string]map[string][]string {
	rawByOwnerReq := map[string]map[string][]string{}
	allRaw := []string{}
	for _, c := range code {
		owner, elem, ok := implementingCodeElement(c)
		if ok {
			allRaw = append(allRaw, addImplementingElement(rawByOwnerReq, owner, elem, c.Implements)...)
		}
	}
	globalLookup := codescan.EvidenceLabelLookup(allRaw)
	byOwnerReq := map[string]map[string][]string{}
	for owner, byReq := range rawByOwnerReq {
		byOwnerReq[owner] = labelRequirementCode(byReq, globalLookup)
	}
	return byOwnerReq
}

// addImplementingElement adds the element under each requirement it implements
// and returns it once per requirement.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func addImplementingElement(rawByOwnerReq map[string]map[string][]string, owner, elem string, implements []string) []string {
	added := []string{}
	for _, reqID := range textutil.UniqueSorted(implements) {
		if rawByOwnerReq[owner] == nil {
			rawByOwnerReq[owner] = map[string][]string{}
		}
		rawByOwnerReq[owner][reqID] = append(rawByOwnerReq[owner][reqID], elem)
		added = append(added, elem)
	}
	return added
}

// labelRequirementCode labels the code elements of each requirement.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func labelRequirementCode(byReq map[string][]string, globalLookup map[string]string) map[string][]string {
	labeled := map[string][]string{}
	for reqID, items := range byReq {
		labeled[reqID] = codescan.GroupedEvidenceLabels(items, requirementLabelLookup(items, globalLookup))
	}
	return labeled
}

// implementingCodeElement is the owner and evidence element of a code symbol
// with a resolved owner, other than verification code.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func implementingCodeElement(c codescan.CodeItem) (owner, elem string, ok bool) {
	if codescan.IsVerificationCodeItem(c) || strings.TrimSpace(c.Kind) != "symbol" {
		return "", "", false
	}
	owner = strings.TrimSpace(c.Owner)
	if owner == "" || owner == "unresolved" {
		return "", "", false
	}
	elem = codescan.ItemEvidenceElement(c)
	return owner, elem, strings.TrimSpace(elem) != ""
}

// requirementLabelLookup labels the items with the path the global lookup
// disambiguated, and only the line numbers of these items.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func requirementLabelLookup(items []string, globalLookup map[string]string) map[string]string {
	local := codescan.EvidenceLabelLookup(items)
	for key, label := range local {
		path, _, ok := codescan.SplitEvidencePathLines(globalLookup[key])
		if !ok {
			continue
		}
		_, lineNumbers, _ := codescan.SplitEvidencePathLines(label)
		local[key] = path
		if len(lineNumbers) > 0 {
			local[key] += ":" + codescan.JoinLineNumbers(lineNumbers)
		}
	}
	return local
}

// checksByRequirement lists the verification checks of each requirement they
// verify, by ID, then name.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func checksByRequirement(verification []codescan.VerificationCheck) map[string][]codescan.VerificationCheck {
	byReq := map[string][]codescan.VerificationCheck{}
	for _, v := range verification {
		if strings.TrimSpace(v.ID) == "" {
			continue
		}
		for _, reqID := range textutil.UniqueSorted(v.Verifies) {
			byReq[reqID] = append(byReq[reqID], v)
		}
	}
	for _, checks := range byReq {
		sort.SliceStable(checks, func(i, j int) bool {
			return cmp.Or(cmp.Compare(checks[i].ID, checks[j].ID), cmp.Compare(checks[i].Name, checks[j].Name)) < 0
		})
	}
	return byReq
}

// writeRequirement draws a requirement with its units, their evidence and its verification checks.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func (e coverageEvidence) writeRequirement(m *mermaidLines, r model.Requirement) {
	req := coverageRequirement{req: r, node: "REQC_" + codescan.SanitizeNode(r.ID)}
	m.add("  " + req.node + "[\"" + escapeMermaidLabel(r.ID) + "\"]:::requirement")
	for _, fu := range textutil.UniqueSorted(r.AppliesTo) {
		e.writeUnit(m, req, fu)
	}
	e.writeChecks(m, req)
}

// writeUnit draws a unit the requirement applies to, its runtime evidence and
// the code that implements the requirement.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func (e coverageEvidence) writeUnit(m *mermaidLines, req coverageRequirement, fu string) {
	reqCode := e.codeByOwnerReq[fu][strings.TrimSpace(req.req.ID)]
	unit := coverageUnit{id: fu, label: textutil.NonEmpty(e.labels[fu], fu), hasCode: len(reqCode) > 0}
	fuNode := "FU_" + codescan.SanitizeNode(fu)
	m.add("  " + fuNode + "[\"" + escapeMermaidLabel(unit.label) + "\"]:::functional_unit")
	m.add("  " + req.node + " -->|applies_to| " + fuNode)
	primaryRTNode := ""
	if rtNodes := e.writeRuntimeNodes(m, unit); len(rtNodes) > 0 {
		primaryRTNode = rtNodes[0]
	}
	for _, cm := range textutil.UniqueSorted(reqCode) {
		cmNode := "CODE_" + codescan.SanitizeNode(req.req.ID) + "_" + codescan.SanitizeNode(codescan.EvidenceFileKey(cm))
		m.add("  " + cmNode + "[\"" + escapeMermaidLabel(diagramCodeEvidenceLabel(cm)) + "\"]:::code_element")
		source, edge := fuNode, " -->|code evidence| "
		if primaryRTNode != "" {
			source, edge = primaryRTNode, " --->|implemented_by| "
		}
		m.add("  " + source + edge + cmNode)
		m.add("  " + req.node + " -->|code trace| " + cmNode)
	}
}

// writeRuntimeNodes draws the runtime elements of a unit, or one inferred
// runtime when the unit has none but implementing code. It returns their nodes.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func (e coverageEvidence) writeRuntimeNodes(m *mermaidLines, unit coverageUnit) []string {
	fuNode := "FU_" + codescan.SanitizeNode(unit.id)
	rtNodes := []string{}
	for _, rt := range textutil.UniqueSorted(e.rtByOwner[unit.id]) {
		rtNode := "RTC_" + codescan.SanitizeNode(unit.id+"-"+rt)
		m.add("  " + rtNode + "[\"" + escapeMermaidLabel(rt) + "\"]:::runtime_element")
		m.add("  " + fuNode + " -->|runtime evidence| " + rtNode)
		rtNodes = append(rtNodes, rtNode)
	}
	if len(rtNodes) == 0 && unit.hasCode {
		rtNode := "RTC_" + codescan.SanitizeNode(unit.id+"-runtime")
		m.add("  " + rtNode + "[\"" + escapeMermaidLabel(unit.label+" runtime (inferred)") + "\"]:::runtime_element")
		m.add("  " + fuNode + " -->|runtime evidence| " + rtNode)
		rtNodes = append(rtNodes, rtNode)
	}
	return rtNodes
}

// writeChecks draws the verification checks of the requirement with their test code.
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-026
func (e coverageEvidence) writeChecks(m *mermaidLines, req coverageRequirement) {
	checks := e.checksByReq[strings.TrimSpace(req.req.ID)]
	allCode := []string{}
	for _, v := range checks {
		allCode = append(allCode, v.CodeElements...)
	}
	lookup := codescan.EvidenceLabelLookup(allCode)
	for _, v := range checks {
		checkLabel := strings.TrimSpace(v.ID)
		if status := strings.TrimSpace(v.Status); status != "" {
			checkLabel += " (" + status + ")"
		}
		verNode := "VERC_" + codescan.SanitizeNode(v.ID)
		m.add("  " + verNode + "[\"" + escapeMermaidLabel(checkLabel) + "\"]:::verification")
		m.add("  " + verNode + " -->|verifies| " + req.node)
		for _, label := range codescan.GroupedEvidenceLabels(v.CodeElements, lookup) {
			ceNode := "VERCODE_" + codescan.SanitizeNode(req.req.ID+"_"+v.ID) + "_" + codescan.SanitizeNode(codescan.EvidenceFileKey(label))
			m.add("  " + ceNode + "[\"" + escapeMermaidLabel(diagramCodeEvidenceLabel(label)) + "\"]:::code_element")
			m.add("  " + verNode + " --->|implemented_by| " + ceNode)
		}
	}
}

// TRLC-LINKS: REQ-EMG-003
func escapeMermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// moduleBucket collects the files and languages of one owner's module.
type moduleBucket struct {
	owner string
	mod   string
	langs map[string]bool
	files map[string]bool
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildCodeOwnershipRows(in []codescan.CodeItem) []asciidocInferredRow {
	moduleBuckets := map[string]*moduleBucket{}
	libraryRows := []asciidocInferredRow{}
	librarySeen := map[string]bool{}
	for _, c := range in {
		if isModuleItem(c) {
			addToModuleBucket(moduleBuckets, c)
		} else if row, ok := libraryRow(c); ok && !librarySeen[libraryRowKey(row)] {
			librarySeen[libraryRowKey(row)] = true
			libraryRows = append(libraryRows, row)
		}
	}
	rows := make([]asciidocInferredRow, 0, len(moduleBuckets)+len(libraryRows))
	for _, b := range moduleBuckets {
		rows = append(rows, moduleRow(b))
	}
	rows = append(rows, libraryRows...)
	sort.SliceStable(rows, func(i, j int) bool {
		return cmp.Or(cmp.Compare(rows[i].Owner, rows[j].Owner), cmp.Compare(rows[i].Kind, rows[j].Kind), cmp.Compare(rows[i].Name, rows[j].Name)) < 0
	})
	if len(rows) > 28 {
		return rows[:28]
	}
	return rows
}

// isModuleItem reports whether the item is a source file or a symbol.
// TRLC-LINKS: REQ-EMG-003
func isModuleItem(c codescan.CodeItem) bool {
	return c.Kind == "source_file" || c.Kind == "symbol"
}

// addToModuleBucket adds the file of a source file or symbol to the bucket of
// its owner and module.
// TRLC-LINKS: REQ-EMG-003
func addToModuleBucket(buckets map[string]*moduleBucket, c codescan.CodeItem) {
	path := codescan.ItemPath(c)
	if path == "" {
		return
	}
	mod := moduleFromPath(path)
	owner := textutil.NonEmpty(strings.TrimSpace(c.Owner), "unresolved")
	key := owner + "|" + mod
	b, ok := buckets[key]
	if !ok {
		b = &moduleBucket{owner: owner, mod: mod, langs: map[string]bool{}, files: map[string]bool{}}
		buckets[key] = b
	}
	if lg := codescan.LanguageFromPath(path); lg != "" {
		b.langs[lg] = true
	}
	b.files[path] = true
}

// libraryRow is the row of a library dependency, or false for another kind of item.
// TRLC-LINKS: REQ-EMG-003
func libraryRow(c codescan.CodeItem) (asciidocInferredRow, bool) {
	if !strings.HasPrefix(c.Kind, "library_") {
		return asciidocInferredRow{}, false
	}
	libType := strings.TrimPrefix(c.Kind, "library_")
	return asciidocInferredRow{
		Name:   c.Element,
		Kind:   "library (" + strings.ReplaceAll(libType, "_", "-") + ")",
		Owner:  textutil.NonEmpty(strings.TrimSpace(c.Owner), "unresolved"),
		Source: "module: " + moduleFromPath(codescan.ItemPath(c)),
	}, true
}

// libraryRowKey identifies a library row: owner, module, library and kind.
// TRLC-LINKS: REQ-EMG-003
func libraryRowKey(row asciidocInferredRow) string {
	return row.Owner + "|" + strings.TrimPrefix(row.Source, "module: ") + "|" + row.Name + "|" + row.Kind
}

// moduleRow is the row of a module: its languages and the first files.
// TRLC-LINKS: REQ-EMG-003
func moduleRow(b *moduleBucket) asciidocInferredRow {
	files := setToSortedSlice(b.files)
	display := files
	if len(display) > 4 {
		display = append(display[:4], fmt.Sprintf("+%d more", len(files)-4))
	}
	kind := "module"
	if langs := setToSortedSlice(b.langs); len(langs) > 0 {
		kind = "module (" + strings.Join(langs, ", ") + ")"
	}
	return asciidocInferredRow{Name: b.mod, Kind: kind, Owner: b.owner, Source: strings.Join(display, ", ")}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildCodeOwnershipMermaid(rows []asciidocInferredRow, a model.AuthoredArchitecture) string {
	d := &ownershipDiagram{
		lines:       mermaidLines{"flowchart TB"},
		fuToGroup:   map[string]string{},
		fgLabel:     map[string]string{},
		seenNodes:   map[string]bool{},
		seenEdges:   map[string]bool{},
		seenLibrary: map[string]bool{},
	}
	for _, u := range a.FunctionalUnits {
		d.fuToGroup[u.ID] = strings.TrimSpace(u.Group)
	}
	for _, g := range a.FunctionalGroups {
		d.fgLabel[g.ID] = textutil.NonEmpty(g.Name, g.ID)
	}
	for _, r := range rows {
		d.addLibraryRow(r)
	}
	d.lines = appendMermaidClassDefs(d.lines)
	return strings.Join(uniquePreserve(d.lines), "\n")
}

// ownershipDiagram draws the libraries each unit uses, with the group of the unit.
type ownershipDiagram struct {
	lines       mermaidLines
	fuToGroup   map[string]string
	fgLabel     map[string]string
	seenNodes   map[string]bool
	seenEdges   map[string]bool
	seenLibrary map[string]bool
}

// addLibraryRow draws the unit that owns a library row, its group and the
// library it uses.
// TRLC-LINKS: REQ-EMG-003
func (d *ownershipDiagram) addLibraryRow(r asciidocInferredRow) {
	fu := strings.TrimSpace(r.Owner)
	if !strings.HasPrefix(r.Kind, "library") || fu == "" || fu == "unresolved" {
		return
	}
	fun := "FU_" + codescan.SanitizeNode(fu)
	if !d.seenNodes[fun] {
		d.lines.addf("  %s[\"%s\"]:::functional_unit", fun, escapeMermaidLabel(fu))
		d.seenNodes[fun] = true
	}
	d.addGroup(fu, fun)
	ln := "LIB_" + codescan.SanitizeNode(r.Name)
	d.lines.addf("  %s[\"%s\"]:::code_element", ln, escapeMermaidLabel(shortLibraryLabel(r.Name)))
	if edgeKey := fun + "|" + ln; !d.seenLibrary[edgeKey] {
		d.lines.addf("  %s -->|uses_library| %s", fun, ln)
		d.seenLibrary[edgeKey] = true
	}
}

// addGroup draws the group of a unit and its containment edge, each once.
// TRLC-LINKS: REQ-EMG-003
func (d *ownershipDiagram) addGroup(fu, fun string) {
	fgID := d.fuToGroup[fu]
	if strings.TrimSpace(fgID) == "" {
		return
	}
	fgn := "FG_" + codescan.SanitizeNode(fgID)
	if !d.seenNodes[fgn] {
		d.lines.addf("  %s[\"%s\"]:::functional_group", fgn, escapeMermaidLabel(textutil.NonEmpty(d.fgLabel[fgID], fgID)))
		d.seenNodes[fgn] = true
	}
	if edgeKey := fgn + "|" + fun; !d.seenEdges[edgeKey] {
		d.lines.addf("  %s -->|contains| %s", fgn, fun)
		d.seenEdges[edgeKey] = true
	}
}

// TRLC-LINKS: REQ-EMG-003
func shortLibraryLabel(lib string) string {
	x := strings.TrimSpace(lib)
	if x == "" {
		return x
	}
	x = strings.TrimPrefix(x, "./")
	x = strings.TrimPrefix(x, "crate::")
	x = lastSegments(x, "/", 2)
	return lastSegments(x, "::", 3)
}

// lastSegments keeps the last n segments of a path that has more than n.
// TRLC-LINKS: REQ-EMG-003
func lastSegments(path, sep string, n int) string {
	parts := strings.Split(path, sep)
	if len(parts) <= n {
		return path
	}
	return strings.Join(parts[len(parts)-n:], sep)
}

// TRLC-LINKS: REQ-EMG-003
func moduleFromPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	if p == "" {
		return "root"
	}
	dir := filepath.ToSlash(filepath.Dir(p))
	if dir == "." || dir == "/" || dir == "" {
		base := filepath.Base(p)
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}
	return dir
}

// TRLC-LINKS: REQ-EMG-003
func setToSortedSlice(in map[string]bool) []string {
	out := make([]string, 0, len(in))
	for x := range in {
		if strings.TrimSpace(x) != "" {
			out = append(out, strings.TrimSpace(x))
		}
	}
	sort.Strings(out)
	return out
}

// TRLC-LINKS: REQ-EMG-003
func uniquePreserve(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}
