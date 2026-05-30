// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"html"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/render/diagramstyle"
)

// TRLC-LINKS: REQ-EMG-003
func appendMermaidClassDefs(lines []string) []string {
	return append(lines, diagramstyle.MermaidClassDefsWithIndent("  ")...)
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-REQUIREMENT, EM-FUNCTIONAL-UNIT
// TRLC-LINKS: REQ-EMG-003
func buildRequirementAlignmentMermaid(reqs []model.Requirement, labels map[string]string) string {
	lines := []string{"flowchart LR"}
	for _, r := range reqs {
		reqNode := "REQ_" + sanitizeNode(r.ID)
		lines = append(lines, "  "+reqNode+"[\""+escapeMermaidLabel(r.ID)+"\"]:::requirement")
		for _, u := range uniqueSorted(r.AppliesTo) {
			target := "UNIT_" + sanitizeNode(u)
			label := nonEmpty(labels[u], u)
			lines = append(lines, "  "+target+"[\""+escapeMermaidLabel(label)+"\"]:::functional_unit")
			lines = append(lines, "  "+reqNode+" --> "+target)
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-REQUIREMENT, EM-FUNCTIONAL-UNIT
// TRLC-LINKS: REQ-EMG-003
func buildRequirementAlignmentCompactTable(reqs []model.Requirement) string {
	appliesByReq := map[string]map[string]bool{}
	reqIDs := make([]string, 0, len(reqs))
	seenReq := map[string]bool{}
	fuSet := map[string]bool{}

	type reqTargets struct {
		ID      string
		Targets []string
	}
	byReq := make([]reqTargets, 0, len(reqs))

	for _, r := range reqs {
		rid := strings.TrimSpace(r.ID)
		if rid == "" || seenReq[rid] {
			continue
		}
		seenReq[rid] = true
		targets := uniqueSorted(r.AppliesTo)
		for _, fu := range targets {
			if strings.TrimSpace(fu) == "" {
				continue
			}
			fuSet[fu] = true
		}
		byReq = append(byReq, reqTargets{ID: rid, Targets: targets})
	}
	sort.SliceStable(byReq, func(i, j int) bool { return byReq[i].ID < byReq[j].ID })
	for _, r := range byReq {
		reqIDs = append(reqIDs, r.ID)
		if appliesByReq[r.ID] == nil {
			appliesByReq[r.ID] = map[string]bool{}
		}
		for _, fu := range r.Targets {
			fu = strings.TrimSpace(fu)
			if fu == "" {
				continue
			}
			appliesByReq[r.ID][fu] = true
		}
	}
	fuIDs := uniqueSorted(keysFromSet(fuSet))
	if len(reqIDs) == 0 || len(fuIDs) == 0 {
		return "No authored requirement-to-unit mappings were found."
	}

	const maxTableColumns = 8
	const maxFUColumnsPerBand = maxTableColumns - 1

	renderBand := func(band []string) string {
		colSpec := make([]string, 0, len(band)+1)
		colSpec = append(colSpec, "2")
		for range band {
			colSpec = append(colSpec, "1")
		}
		lines := []string{
			"[cols=\"" + strings.Join(colSpec, ",") + "\",options=\"header\"]",
			"|===",
			"|Requirement",
		}
		for _, fu := range band {
			lines = append(lines, "|"+escapeTableCell(fu))
		}
		for _, reqID := range reqIDs {
			lines = append(lines, "|"+escapeTableCell(reqID))
			for _, fu := range band {
				mark := ""
				if appliesByReq[reqID][fu] {
					mark = "X"
				}
				lines = append(lines, "|"+mark)
			}
		}
		lines = append(lines, "|===")
		return strings.Join(lines, "\n")
	}

	if len(fuIDs) <= maxFUColumnsPerBand {
		return renderBand(fuIDs)
	}

	sections := make([]string, 0, (len(fuIDs)+maxFUColumnsPerBand-1)/maxFUColumnsPerBand)
	for start := 0; start < len(fuIDs); start += maxFUColumnsPerBand {
		end := start + maxFUColumnsPerBand
		if end > len(fuIDs) {
			end = len(fuIDs)
		}
		sections = append(sections,
			fmt.Sprintf("*Functional Unit Columns %d-%d*", start+1, end),
			renderBand(fuIDs[start:end]),
		)
	}
	return strings.Join(sections, "\n\n")
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

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-FUNCTIONAL-GROUP, EM-FUNCTIONAL-UNIT, EM-ACTOR, EM-REFERENCED-ELEMENT
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalContextMermaid(a model.AuthoredArchitecture) string {
	lines := []string{"flowchart LR"}
	for _, act := range a.Actors {
		an := "ACT_" + sanitizeNode(act.ID)
		lines = append(lines, fmt.Sprintf("  %s((\"%s\")):::actor", an, escapeMermaidLabel(nonEmpty(act.Name, act.ID))))
	}
	for _, ref := range a.ReferencedElements {
		rn := "REF_" + sanitizeNode(ref.ID)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::referenced_element", rn, escapeMermaidLabel(nonEmpty(ref.Name, ref.ID))))
	}

	lines = append(lines, "  subgraph SYSTEM_BOUNDARY[\"System Boundary (Internal)\"]")
	lines = append(lines, "    direction TB")
	groupLabelByID := map[string]string{}
	groupOrder := make([]string, 0, len(a.FunctionalGroups))
	for _, fg := range a.FunctionalGroups {
		gid := strings.TrimSpace(fg.ID)
		groupLabelByID[gid] = nonEmpty(fg.Name, fg.ID)
		groupOrder = append(groupOrder, gid)
	}
	unitsByGroup := map[string][]model.FunctionalUnit{}
	for _, fu := range a.FunctionalUnits {
		gid := strings.TrimSpace(fu.Group)
		unitsByGroup[gid] = append(unitsByGroup[gid], fu)
	}
	if len(unitsByGroup[""]) > 0 {
		groupOrder = append(groupOrder, "")
		groupLabelByID[""] = "Unassigned Functional Units"
	}
	for _, gid := range groupOrder {
		units := unitsByGroup[gid]
		if len(units) == 0 {
			continue
		}
		boxID := "FGCTX_" + sanitizeNode(gid)
		if gid == "" {
			boxID = "FGCTX_UNASSIGNED"
		}
		lines = append(lines, fmt.Sprintf("    subgraph %s[\"%s\"]", boxID, escapeMermaidLabel(groupLabelByID[gid])))
		lines = append(lines, "      direction TB")
		for _, fu := range units {
			un := "FU_" + sanitizeNode(fu.ID)
			lines = append(lines, fmt.Sprintf("      %s[\"%s\"]:::functional_unit", un, escapeMermaidLabel(nonEmpty(fu.Name, fu.ID))))
		}
		lines = append(lines, "    end")
		lines = append(lines, fmt.Sprintf("    style %s fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;", boxID))
	}
	lines = append(lines, "  end")
	lines = append(lines, "  style SYSTEM_BOUNDARY fill:#f5f5f5,stroke:#424242,stroke-width:1px;")

	internalNodes := map[string]bool{}
	for _, fg := range a.FunctionalGroups {
		internalNodes[strings.TrimSpace(fg.ID)] = true
	}
	for _, fu := range a.FunctionalUnits {
		internalNodes[strings.TrimSpace(fu.ID)] = true
	}
	externalNodes := map[string]string{}
	for _, act := range a.Actors {
		externalNodes[strings.TrimSpace(act.ID)] = "ACT_"
	}
	for _, ref := range a.ReferencedElements {
		externalNodes[strings.TrimSpace(ref.ID)] = "REF_"
	}
	seenEdges := map[string]bool{}
	for _, m := range a.Mappings {
		label := strings.TrimSpace(m.Description)
		if label == "" {
			label = strings.TrimSpace(m.Type)
		}
		label = escapeMermaidLabel(label)
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		fromExternalPrefix, fromExternal := externalNodes[from]
		toExternalPrefix, toExternal := externalNodes[to]
		fromInternal := internalNodes[from]
		toInternal := internalNodes[to]
		switch {
		case fromExternal && toInternal:
			fromNode := fromExternalPrefix + sanitizeNode(from)
			toNode := functionalContextInternalNodeID(to)
			edge := fmt.Sprintf("  %s -->|%s| %s", fromNode, label, toNode)
			if !seenEdges[edge] {
				lines = append(lines, edge)
				seenEdges[edge] = true
			}
		case fromInternal && toExternal:
			fromNode := functionalContextInternalNodeID(from)
			toNode := toExternalPrefix + sanitizeNode(to)
			edge := fmt.Sprintf("  %s -->|%s| %s", fromNode, label, toNode)
			if !seenEdges[edge] {
				lines = append(lines, edge)
				seenEdges[edge] = true
			}
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

// TRLC-LINKS: REQ-EMG-003
func functionalContextInternalNodeID(id string) string {
	id = strings.TrimSpace(id)
	if strings.HasPrefix(id, "FG-") {
		return "FGCTX_" + sanitizeNode(id)
	}
	return "FU_" + sanitizeNode(id)
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-FUNCTIONAL-GROUP, EM-FUNCTIONAL-UNIT
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalDecompositionMermaid(a model.AuthoredArchitecture) string {
	lines := []string{"flowchart TB"}
	lines = append(lines, "  SYS[\"System\"]:::system_boundary")
	for _, fg := range a.FunctionalGroups {
		gn := "FG_" + sanitizeNode(fg.ID)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_group", gn, escapeMermaidLabel(nonEmpty(fg.Name, fg.ID))))
		lines = append(lines, fmt.Sprintf("  SYS -->|contains| %s", gn))
	}
	for _, fu := range a.FunctionalUnits {
		un := "FU_" + sanitizeNode(fu.ID)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_unit", un, escapeMermaidLabel(nonEmpty(fu.Name, fu.ID))))
		if strings.TrimSpace(fu.Group) != "" {
			lines = append(lines, fmt.Sprintf("  FG_%s -->|contains| %s", sanitizeNode(fu.Group), un))
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

// ENGMODEL-LINKS: EM-ASCIIDOC-SECTION, EM-FUNCTIONAL-GROUP, EM-FUNCTIONAL-UNIT
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalManhattanTable(a model.AuthoredArchitecture) string {
	type fgColumn struct {
		ID    string
		Name  string
		Units []model.FunctionalUnit
	}
	cols := make([]fgColumn, 0, len(a.FunctionalGroups))
	for _, fg := range a.FunctionalGroups {
		cols = append(cols, fgColumn{
			ID:   strings.TrimSpace(fg.ID),
			Name: nonEmpty(strings.TrimSpace(fg.Name), strings.TrimSpace(fg.ID)),
		})
	}
	indexByFG := map[string]int{}
	for i, c := range cols {
		indexByFG[c.ID] = i
	}
	for _, fu := range a.FunctionalUnits {
		gid := strings.TrimSpace(fu.Group)
		idx, ok := indexByFG[gid]
		if !ok {
			continue
		}
		cols[idx].Units = append(cols[idx].Units, fu)
	}
	for i := range cols {
		sort.SliceStable(cols[i].Units, func(left, right int) bool {
			l := nonEmpty(strings.TrimSpace(cols[i].Units[left].Name), strings.TrimSpace(cols[i].Units[left].ID))
			r := nonEmpty(strings.TrimSpace(cols[i].Units[right].Name), strings.TrimSpace(cols[i].Units[right].ID))
			if l != r {
				return l < r
			}
			return cols[i].Units[left].ID < cols[i].Units[right].ID
		})
	}

	cell := func(text, bg, border, fg string) string {
		escaped := html.EscapeString(strings.TrimSpace(text))
		if escaped == "" {
			return `<td style="padding:0;border:none;vertical-align:bottom;background:#f5f5f5;"></td>`
		}
		return fmt.Sprintf(`<td style="padding:0;border:none;vertical-align:bottom;"><div style="background:%s;border:1px solid %s;color:%s;padding:1px 3px;margin:0;display:block;text-align:center;min-height:10px;font-size:0.82em;line-height:1.02;white-space:normal;">%s</div></td>`, bg, border, fg, escaped)
	}

	renderBand := func(band []fgColumn) string {
		maxRows := 0
		for _, c := range band {
			if len(c.Units) > maxRows {
				maxRows = len(c.Units)
			}
		}
		lines := []string{
			"++++",
			`<div style="background:#f5f5f5;padding:12px;">`,
			`<table style="width:100%;table-layout:fixed;border-collapse:collapse;border-spacing:0;margin:0;border:0;outline:0;">`,
		}
		for row := 0; row < maxRows; row++ {
			lines = append(lines, "<tr>")
			for _, c := range band {
				label := ""
				// Bottom-align units within each FG column so the lowest FU row is populated first.
				offset := maxRows - len(c.Units)
				unitIdx := row - offset
				if unitIdx >= 0 && unitIdx < len(c.Units) {
					label = nonEmpty(strings.TrimSpace(c.Units[unitIdx].Name), strings.TrimSpace(c.Units[unitIdx].ID))
				}
				lines = append(lines, cell(label, "#e3f2fd", "#0d47a1", "#0d47a1"))
			}
			lines = append(lines, "</tr>")
		}
		lines = append(lines, "<tr>")
		for _, c := range band {
			lines = append(lines, cell(c.Name, "#e8f5e9", "#1b5e20", "#1b5e20"))
		}
		lines = append(lines, "</tr>", "</table>", "</div>", "++++")
		return strings.Join(lines, "\n")
	}

	const maxColumnsPerBand = 8
	if len(cols) <= maxColumnsPerBand {
		return renderBand(cols)
	}

	sections := make([]string, 0, (len(cols)+maxColumnsPerBand-1)/maxColumnsPerBand)
	for start := 0; start < len(cols); start += maxColumnsPerBand {
		end := start + maxColumnsPerBand
		if end > len(cols) {
			end = len(cols)
		}
		band := cols[start:end]
		sections = append(sections,
			fmt.Sprintf("*FG Columns %d-%d*", start+1, end),
			renderBand(band),
		)
	}
	return strings.Join(sections, "\n\n")
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-FUNCTIONAL-UNIT, EM-AUTHORED-MAPPING
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalCollaborationMermaid(a model.AuthoredArchitecture) string {
	lines := []string{"flowchart LR"}
	groupLabelByID := map[string]string{}
	for _, fg := range a.FunctionalGroups {
		groupLabelByID[fg.ID] = nonEmpty(fg.Name, fg.ID)
	}
	unitsByGroup := map[string][]model.FunctionalUnit{}
	groupOrder := make([]string, 0, len(a.FunctionalGroups))
	for _, fg := range a.FunctionalGroups {
		groupID := strings.TrimSpace(fg.ID)
		groupOrder = append(groupOrder, groupID)
	}
	for _, fu := range a.FunctionalUnits {
		groupID := strings.TrimSpace(fu.Group)
		unitsByGroup[groupID] = append(unitsByGroup[groupID], fu)
	}
	if len(unitsByGroup[""]) > 0 {
		groupOrder = append(groupOrder, "")
		groupLabelByID[""] = "Unassigned Functional Units"
	}
	for _, gid := range groupOrder {
		units := unitsByGroup[gid]
		if len(units) == 0 {
			continue
		}
		boxID := "FGBOX_" + sanitizeNode(gid)
		if gid == "" {
			boxID = "FGBOX_UNASSIGNED"
		}
		lines = append(lines, fmt.Sprintf("  subgraph %s[\"%s\"]", boxID, escapeMermaidLabel(groupLabelByID[gid])))
		lines = append(lines, "    direction TB")
		for _, fu := range units {
			un := "FU_" + sanitizeNode(fu.ID)
			lines = append(lines, fmt.Sprintf("    %s[\"%s\"]:::functional_unit", un, escapeMermaidLabel(nonEmpty(fu.Name, fu.ID))))
		}
		lines = append(lines, "  end")
		lines = append(lines, fmt.Sprintf("  style %s fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;", boxID))
	}

	for _, ref := range a.ReferencedElements {
		rn := "REF_" + sanitizeNode(ref.ID)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::referenced_element", rn, escapeMermaidLabel(nonEmpty(ref.Name, ref.ID))))
	}
	seenEdges := map[string]bool{}
	addEdge := func(edge string) {
		if seenEdges[edge] {
			return
		}
		seenEdges[edge] = true
		lines = append(lines, edge)
	}
	for _, m := range a.Mappings {
		if m.Type != "depends_on" {
			continue
		}
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		switch {
		case strings.HasPrefix(from, "FU-") && strings.HasPrefix(to, "FU-"):
			addEdge(fmt.Sprintf("  FU_%s -->|depends_on| FU_%s", sanitizeNode(from), sanitizeNode(to)))
		case strings.HasPrefix(from, "FU-") && strings.HasPrefix(to, "REF-"):
			addEdge(fmt.Sprintf("  FU_%s -->|depends_on| REF_%s", sanitizeNode(from), sanitizeNode(to)))
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-FUNCTIONAL-GROUP, EM-FUNCTIONAL-UNIT, EM-RUNTIME-ELEMENT, EM-CODE-ELEMENT
// TRLC-LINKS: REQ-EMG-003
func buildFunctionalGroupDependencyMermaid(a model.AuthoredArchitecture, groupID string, runtime []inferredRuntimeItem, code []inferredCodeItem) string {
	groupID = strings.TrimSpace(groupID)
	groupLabelByID := map[string]string{}
	unitLabelByID := map[string]string{}
	unitGroupByID := map[string]string{}
	refLabelByID := map[string]string{}
	for _, fg := range a.FunctionalGroups {
		groupLabelByID[strings.TrimSpace(fg.ID)] = nonEmpty(fg.Name, fg.ID)
	}
	localUnits := []model.FunctionalUnit{}
	localUnitSet := map[string]bool{}
	for _, fu := range a.FunctionalUnits {
		id := strings.TrimSpace(fu.ID)
		unitLabelByID[id] = nonEmpty(fu.Name, fu.ID)
		unitGroupByID[id] = strings.TrimSpace(fu.Group)
		if strings.TrimSpace(fu.Group) == groupID {
			localUnits = append(localUnits, fu)
			localUnitSet[id] = true
		}
	}
	if len(localUnits) == 0 {
		return ""
	}
	sort.SliceStable(localUnits, func(i, j int) bool {
		return localUnits[i].ID < localUnits[j].ID
	})
	for _, ref := range a.ReferencedElements {
		refLabelByID[strings.TrimSpace(ref.ID)] = nonEmpty(ref.Name, ref.ID)
	}

	externalFUs := map[string]bool{}
	refs := map[string]bool{}
	edgeSet := map[string]bool{}
	edges := []string{}
	addEdge := func(edge string) {
		if edgeSet[edge] {
			return
		}
		edgeSet[edge] = true
		edges = append(edges, edge)
	}
	for _, m := range a.Mappings {
		if strings.TrimSpace(m.Type) != "depends_on" {
			continue
		}
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		if !localUnitSet[from] {
			continue
		}
		switch {
		case strings.HasPrefix(from, "FU-") && strings.HasPrefix(to, "FU-"):
			if !localUnitSet[to] {
				externalFUs[to] = true
			}
			addEdge(fmt.Sprintf("  FU_%s -->|depends_on| FU_%s", sanitizeNode(from), sanitizeNode(to)))
		case strings.HasPrefix(from, "FU-") && strings.HasPrefix(to, "REF-"):
			refs[to] = true
			addEdge(fmt.Sprintf("  FU_%s -->|depends_on| REF_%s", sanitizeNode(from), sanitizeNode(to)))
		}
	}
	if len(edges) == 0 {
		return ""
	}

	evidenceScopeUnits := map[string]bool{}
	for id := range localUnitSet {
		evidenceScopeUnits[id] = true
	}
	runtimeByOwner := map[string][]string{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		if !evidenceScopeUnits[owner] {
			continue
		}
		label := nonEmpty(strings.TrimSpace(r.Name), strings.TrimSpace(r.Kind))
		if label != "" {
			runtimeByOwner[owner] = append(runtimeByOwner[owner], label)
		}
	}
	for owner, items := range runtimeByOwner {
		runtimeByOwner[owner] = uniqueSorted(items)
	}
	codeRawByOwner := map[string][]string{}
	allCodeRaw := []string{}
	for _, c := range code {
		owner := strings.TrimSpace(c.Owner)
		if !evidenceScopeUnits[owner] {
			continue
		}
		switch strings.TrimSpace(c.Kind) {
		case "source_file", "symbol":
			elem := codeItemEvidenceElement(c)
			if strings.TrimSpace(elem) == "" {
				continue
			}
			codeRawByOwner[owner] = append(codeRawByOwner[owner], elem)
			allCodeRaw = append(allCodeRaw, elem)
		}
	}
	codeByOwner := map[string][]string{}
	codeLabelLookup := codeElementEvidenceLabelLookup(allCodeRaw)
	for owner, items := range codeRawByOwner {
		codeByOwner[owner] = groupedCodeElementFileLabels(items, codeLabelLookup)
	}

	lines := []string{"flowchart LR"}
	boxID := "FGDEP_" + sanitizeNode(groupID)
	if groupID == "" {
		boxID = "FGDEP_UNASSIGNED"
	}
	lines = append(lines, fmt.Sprintf("  subgraph %s[\"%s\"]", boxID, escapeMermaidLabel(nonEmpty(groupLabelByID[groupID], groupID))))
	lines = append(lines, "    direction TB")
	for _, fu := range localUnits {
		un := "FU_" + sanitizeNode(fu.ID)
		lines = append(lines, fmt.Sprintf("    %s[\"%s\"]:::functional_unit", un, escapeMermaidLabel(nonEmpty(fu.Name, fu.ID))))
	}
	definedCodeNodes := map[string]bool{}
	for _, owner := range keysSorted(evidenceScopeUnits) {
		runtimeLabels := append([]string(nil), runtimeByOwner[owner]...)
		if len(runtimeLabels) == 0 && len(codeByOwner[owner]) > 0 {
			runtimeLabels = append(runtimeLabels, nonEmpty(unitLabelByID[owner], owner)+" runtime (inferred)")
		}
		for idx, rt := range runtimeLabels {
			rtBox := "RT_" + sanitizeNode(owner+"-"+rt)
			lines = append(lines, fmt.Sprintf("    subgraph %s[\"%s\"]", rtBox, escapeMermaidLabel(rt)))
			lines = append(lines, "      direction TB")
			if idx == 0 {
				for _, label := range codeByOwner[owner] {
					codeNode := "CODE_" + sanitizeNode(codeElementEvidenceFileKey(label))
					if definedCodeNodes[codeNode] {
						continue
					}
					lines = append(lines, fmt.Sprintf("      %s[\"%s\"]:::code_element", codeNode, escapeMermaidLabel(label)))
					definedCodeNodes[codeNode] = true
				}
			}
			if idx > 0 || len(codeByOwner[owner]) == 0 {
				rtNode := "RTNODE_" + sanitizeNode(owner+"-"+rt)
				lines = append(lines, fmt.Sprintf("      %s[\"runtime evidence\"]:::runtime_element", rtNode))
			}
			lines = append(lines, "    end")
			lines = append(lines, fmt.Sprintf("    style %s fill:#b2ebf2,stroke:#00838f,stroke-width:1px,color:#006064;", rtBox))
			lines = append(lines, fmt.Sprintf("    FU_%s -->|runtime evidence| %s", sanitizeNode(owner), rtBox))
		}
	}
	lines = append(lines, "  end")
	lines = append(lines, fmt.Sprintf("  style %s fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;", boxID))

	externalUnitsByGroup := map[string][]string{}
	for _, id := range keysSorted(externalFUs) {
		gid := strings.TrimSpace(unitGroupByID[id])
		externalUnitsByGroup[gid] = append(externalUnitsByGroup[gid], id)
	}
	for _, gid := range keysSortedStringSlices(externalUnitsByGroup) {
		boxID := "FGDEP_TARGET_" + sanitizeNode(gid)
		if gid == "" {
			boxID = "FGDEP_TARGET_UNASSIGNED"
		}
		lines = append(lines, fmt.Sprintf("  subgraph %s[\"%s\"]", boxID, escapeMermaidLabel(nonEmpty(groupLabelByID[gid], "External Functional Units"))))
		lines = append(lines, "    direction TB")
		for _, id := range uniqueSorted(externalUnitsByGroup[gid]) {
			lines = append(lines, fmt.Sprintf("    FU_%s[\"%s\"]:::functional_unit", sanitizeNode(id), escapeMermaidLabel(nonEmpty(unitLabelByID[id], id))))
		}
		lines = append(lines, "  end")
		lines = append(lines, fmt.Sprintf("  style %s fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;", boxID))
	}
	for _, id := range keysSorted(refs) {
		lines = append(lines, fmt.Sprintf("  REF_%s[\"%s\"]:::referenced_element", sanitizeNode(id), escapeMermaidLabel(nonEmpty(refLabelByID[id], id))))
	}
	lines = append(lines, edges...)
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
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

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-REQUIREMENT, EM-RUNTIME-ELEMENT, EM-CODE-ELEMENT, EM-VERIFICATION-CHECK
// TRLC-LINKS: REQ-EMG-003
func buildRequirementCoverageMermaid(reqs []model.Requirement, runtime []inferredRuntimeItem, code []inferredCodeItem, verification []inferredVerificationCheck, labels map[string]string) string {
	lines := []string{"flowchart LR"}
	rtByOwner := map[string][]string{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		if owner == "" || owner == "unresolved" {
			continue
		}
		rtByOwner[owner] = append(rtByOwner[owner], nonEmpty(strings.TrimSpace(r.Name), strings.TrimSpace(r.Kind)))
	}
	codeRawByOwnerReq := map[string]map[string][]string{}
	allCodeRaw := []string{}
	for _, c := range code {
		owner := strings.TrimSpace(c.Owner)
		if owner == "" || owner == "unresolved" || strings.TrimSpace(c.Kind) != "symbol" {
			continue
		}
		elem := codeItemEvidenceElement(c)
		if strings.TrimSpace(elem) == "" {
			continue
		}
		for _, reqID := range uniqueSorted(c.Implements) {
			reqID = strings.TrimSpace(reqID)
			if reqID == "" {
				continue
			}
			if codeRawByOwnerReq[owner] == nil {
				codeRawByOwnerReq[owner] = map[string][]string{}
			}
			codeRawByOwnerReq[owner][reqID] = append(codeRawByOwnerReq[owner][reqID], elem)
			allCodeRaw = append(allCodeRaw, elem)
		}
	}
	codeByOwnerReq := map[string]map[string][]string{}
	codeLabelLookup := codeElementEvidenceLabelLookup(allCodeRaw)
	for owner, byReq := range codeRawByOwnerReq {
		codeByOwnerReq[owner] = map[string][]string{}
		for reqID, items := range byReq {
			codeByOwnerReq[owner][reqID] = groupedCodeElementEvidenceLabels(items, codeLabelLookup)
		}
	}
	checksByReq := map[string][]inferredVerificationCheck{}
	for _, v := range verification {
		id := strings.TrimSpace(v.ID)
		if id == "" {
			continue
		}
		for _, reqID := range uniqueSorted(v.Verifies) {
			reqID = strings.TrimSpace(reqID)
			if reqID == "" {
				continue
			}
			checksByReq[reqID] = append(checksByReq[reqID], v)
		}
	}
	for reqID := range checksByReq {
		sort.SliceStable(checksByReq[reqID], func(i, j int) bool {
			if checksByReq[reqID][i].ID != checksByReq[reqID][j].ID {
				return checksByReq[reqID][i].ID < checksByReq[reqID][j].ID
			}
			return checksByReq[reqID][i].Name < checksByReq[reqID][j].Name
		})
	}

	for _, r := range reqs {
		reqNode := "REQC_" + sanitizeNode(r.ID)
		lines = append(lines, "  "+reqNode+"[\""+escapeMermaidLabel(r.ID)+"\"]:::requirement")
		for _, fu := range uniqueSorted(r.AppliesTo) {
			reqID := strings.TrimSpace(r.ID)
			reqCode := codeByOwnerReq[fu][reqID]
			fuNode := "FU_" + sanitizeNode(fu)
			fuLabel := nonEmpty(labels[fu], fu)
			lines = append(lines, "  "+fuNode+"[\""+escapeMermaidLabel(fuLabel)+"\"]:::functional_unit")
			lines = append(lines, "  "+reqNode+" -->|applies_to| "+fuNode)

			rtNodes := []string{}
			for _, rt := range uniqueSorted(rtByOwner[fu]) {
				rtNode := "RTC_" + sanitizeNode(fu+"-"+rt)
				lines = append(lines, "  "+rtNode+"[\""+escapeMermaidLabel(rt)+"\"]:::runtime_element")
				lines = append(lines, "  "+fuNode+" -->|runtime evidence| "+rtNode)
				rtNodes = append(rtNodes, rtNode)
			}
			if len(rtNodes) == 0 && len(reqCode) > 0 {
				rtLabel := fuLabel + " runtime (inferred)"
				rtNode := "RTC_" + sanitizeNode(fu+"-runtime")
				lines = append(lines, "  "+rtNode+"[\""+escapeMermaidLabel(rtLabel)+"\"]:::runtime_element")
				lines = append(lines, "  "+fuNode+" -->|runtime evidence| "+rtNode)
				rtNodes = append(rtNodes, rtNode)
			}

			primaryRTNode := ""
			if len(rtNodes) > 0 {
				primaryRTNode = rtNodes[0]
			}
			for _, cm := range uniqueSorted(reqCode) {
				cmNode := "CODE_" + sanitizeNode(codeElementEvidenceFileKey(cm))
				lines = append(lines, "  "+cmNode+"[\""+escapeMermaidLabel(cm)+"\"]:::code_element")
				if primaryRTNode != "" {
					lines = append(lines, "  "+primaryRTNode+" -->|implemented_by| "+cmNode)
				} else {
					lines = append(lines, "  "+fuNode+" -->|code evidence| "+cmNode)
				}
				lines = append(lines, "  "+reqNode+" -->|code trace| "+cmNode)
			}
		}
		allVerificationCode := []string{}
		for _, v := range checksByReq[strings.TrimSpace(r.ID)] {
			allVerificationCode = append(allVerificationCode, v.CodeElements...)
		}
		verificationCodeLabelLookup := codeElementEvidenceLabelLookup(allVerificationCode)
		for _, v := range checksByReq[strings.TrimSpace(r.ID)] {
			status := strings.TrimSpace(v.Status)
			checkLabel := strings.TrimSpace(v.ID)
			if status != "" {
				checkLabel = checkLabel + " (" + status + ")"
			}
			verNode := "VERC_" + sanitizeNode(v.ID)
			lines = append(lines, "  "+verNode+"[\""+escapeMermaidLabel(checkLabel)+"\"]:::verification")
			lines = append(lines, "  "+verNode+" -->|verifies| "+reqNode)
			for _, label := range groupedCodeElementEvidenceLabels(v.CodeElements, verificationCodeLabelLookup) {
				ceNode := "CODE_" + sanitizeNode(codeElementEvidenceFileKey(label))
				lines = append(lines, "  "+ceNode+"[\""+escapeMermaidLabel(label)+"\"]:::code_element")
				lines = append(lines, "  "+verNode+" -->|implemented_by| "+ceNode)
			}
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

// TRLC-LINKS: REQ-EMG-003
func sanitizeNode(s string) string {
	repl := strings.NewReplacer("-", "_", " ", "_", ".", "_", "/", "_", ":", "_", ",", "_", "\\", "_", "(", "_", ")", "_", "[", "_", "]", "_")
	out := repl.Replace(strings.ToUpper(strings.TrimSpace(s)))
	if out == "" {
		out = "NODE"
	}
	return out
}

// TRLC-LINKS: REQ-EMG-003
func escapeMermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

// ENGMODEL-LINKS: EM-CODE-ELEMENT, EM-ASCIIDOC-SECTION
// TRLC-LINKS: REQ-EMG-003
func buildCodeOwnershipRows(in []inferredCodeItem) []asciidocInferredRow {
	type moduleBucket struct {
		owner string
		mod   string
		langs map[string]bool
		files map[string]bool
	}
	moduleBuckets := map[string]*moduleBucket{}
	libraryRows := []asciidocInferredRow{}
	librarySeen := map[string]bool{}

	for _, c := range in {
		if c.Kind == "source_file" || c.Kind == "symbol" {
			path := codeItemPath(c)
			if path == "" {
				continue
			}
			mod := moduleFromPath(path)
			owner := nonEmpty(strings.TrimSpace(c.Owner), "unresolved")
			key := owner + "|" + mod
			b, ok := moduleBuckets[key]
			if !ok {
				b = &moduleBucket{
					owner: owner,
					mod:   mod,
					langs: map[string]bool{},
					files: map[string]bool{},
				}
				moduleBuckets[key] = b
			}
			if lg := languageFromPath(path); lg != "" {
				b.langs[lg] = true
			}
			b.files[path] = true
			continue
		}
		if strings.HasPrefix(c.Kind, "library_") {
			owner := nonEmpty(strings.TrimSpace(c.Owner), "unresolved")
			module := moduleFromPath(codeItemPath(c))
			libType := strings.TrimPrefix(c.Kind, "library_")
			kind := "library (" + strings.ReplaceAll(libType, "_", "-") + ")"
			key := owner + "|" + module + "|" + c.Element + "|" + kind
			if librarySeen[key] {
				continue
			}
			librarySeen[key] = true
			libraryRows = append(libraryRows, asciidocInferredRow{
				Name:   c.Element,
				Kind:   kind,
				Owner:  owner,
				Source: "module: " + module,
			})
		}
	}

	rows := make([]asciidocInferredRow, 0, len(moduleBuckets)+len(libraryRows))
	for _, b := range moduleBuckets {
		files := setToSortedSlice(b.files)
		display := files
		if len(display) > 4 {
			display = append(display[:4], fmt.Sprintf("+%d more", len(files)-4))
		}
		langs := setToSortedSlice(b.langs)
		kind := "module"
		if len(langs) > 0 {
			kind = "module (" + strings.Join(langs, ", ") + ")"
		}
		rows = append(rows, asciidocInferredRow{
			Name:   b.mod,
			Kind:   kind,
			Owner:  b.owner,
			Source: strings.Join(display, ", "),
		})
	}
	rows = append(rows, libraryRows...)
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Owner != rows[j].Owner {
			return rows[i].Owner < rows[j].Owner
		}
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].Name < rows[j].Name
	})
	if len(rows) > 28 {
		return rows[:28]
	}
	return rows
}

// ENGMODEL-LINKS: EM-ASCIIDOC-DIAGRAM, EM-CODE-ELEMENT, EM-FUNCTIONAL-UNIT
// TRLC-LINKS: REQ-EMG-003
func buildCodeOwnershipMermaid(rows []asciidocInferredRow, a model.AuthoredArchitecture) string {
	lines := []string{"flowchart TB"}
	fuToGroup := map[string]string{}
	for _, u := range a.FunctionalUnits {
		fuToGroup[u.ID] = strings.TrimSpace(u.Group)
	}
	fgLabel := map[string]string{}
	for _, g := range a.FunctionalGroups {
		fgLabel[g.ID] = nonEmpty(g.Name, g.ID)
	}

	seenFU := map[string]bool{}
	seenFG := map[string]bool{}
	seenFUEdge := map[string]bool{}
	seenLibEdge := map[string]bool{}

	for _, r := range rows {
		if !strings.HasPrefix(r.Kind, "library") {
			continue
		}
		fu := strings.TrimSpace(r.Owner)
		if fu == "" || fu == "unresolved" {
			continue
		}

		fun := "FU_" + sanitizeNode(fu)
		if !seenFU[fun] {
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_unit", fun, escapeMermaidLabel(fu)))
			seenFU[fun] = true
		}

		if fgID := fuToGroup[fu]; strings.TrimSpace(fgID) != "" {
			fgn := "FG_" + sanitizeNode(fgID)
			fg := nonEmpty(fgLabel[fgID], fgID)
			if !seenFG[fgn] {
				lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_group", fgn, escapeMermaidLabel(fg)))
				seenFG[fgn] = true
			}
			edgeKey := fgn + "|" + fun
			if !seenFUEdge[edgeKey] {
				lines = append(lines, fmt.Sprintf("  %s -->|contains| %s", fgn, fun))
				seenFUEdge[edgeKey] = true
			}
		}

		ln := "LIB_" + sanitizeNode(r.Name)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::code_element", ln, escapeMermaidLabel(shortLibraryLabel(r.Name))))
		edgeKey := fun + "|" + ln
		if !seenLibEdge[edgeKey] {
			lines = append(lines, fmt.Sprintf("  %s -->|uses_library| %s", fun, ln))
			seenLibEdge[edgeKey] = true
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

// TRLC-LINKS: REQ-EMG-003
func shortLibraryLabel(lib string) string {
	x := strings.TrimSpace(lib)
	if x == "" {
		return x
	}
	x = strings.TrimPrefix(x, "./")
	x = strings.TrimPrefix(x, "crate::")
	if strings.Contains(x, "/") {
		parts := strings.Split(x, "/")
		if len(parts) > 2 {
			x = strings.Join(parts[len(parts)-2:], "/")
		}
	}
	if strings.Contains(x, "::") {
		parts := strings.Split(x, "::")
		if len(parts) > 3 {
			x = strings.Join(parts[len(parts)-3:], "::")
		}
	}
	return x
}

// TRLC-LINKS: REQ-EMG-003
func codeItemPath(c inferredCodeItem) string {
	src := strings.TrimSpace(c.Source)
	if src == "" {
		return ""
	}
	if idx := strings.Index(src, ":"); idx > 0 {
		src = src[:idx]
	}
	return filepath.ToSlash(strings.TrimSpace(src))
}

// TRLC-LINKS: REQ-EMG-003
func codeItemDisplayName(c inferredCodeItem) string {
	elem := strings.TrimSpace(c.Element)
	source := sanitizeSourcePath(c.Source)
	switch strings.TrimSpace(c.Kind) {
	case "symbol":
		if label := codeItemEvidenceLabel(c); label != "" {
			return label
		}
		return nonEmpty(elem, "code symbol")
	case "source_file":
		if label := codeItemEvidenceLabel(c); label != "" {
			return label
		}
		return nonEmpty(source, elem)
	default:
		if source == "" {
			return elem
		}
		sourceLabel := codeElementEvidenceLabel(source)
		if sourceLabel == "" {
			sourceLabel = source
		}
		return elem + " (" + sourceLabel + ")"
	}
}

// TRLC-LINKS: REQ-EMG-003
func codeItemEvidenceLabel(c inferredCodeItem) string {
	return codeElementEvidenceLabel(codeItemEvidenceElement(c))
}

// TRLC-LINKS: REQ-EMG-003
func codeItemEvidenceElement(c inferredCodeItem) string {
	switch strings.TrimSpace(c.Kind) {
	case "source_file":
		return codeItemPath(c)
	case "symbol":
		return sanitizeSourcePath(c.Source)
	default:
		return strings.TrimSpace(c.Element)
	}
}

// ENGMODEL-LINKS: EM-CODE-ELEMENT, EM-SOURCE-BLOCK
// TRLC-LINKS: REQ-EMG-003
func codeElementEvidenceLabel(elem string) string {
	elem = sanitizeSourcePath(elem)
	if elem == "" {
		return ""
	}
	path := elem
	line := ""
	if idx := strings.LastIndex(path, ":"); idx > 0 && idx < len(path)-1 {
		suffix := path[idx+1:]
		if lines, ok := parseLineNumberList(suffix); ok {
			path = path[:idx]
			line = ":" + joinEvidenceLineNumbers(lines)
		}
	}
	if filepath.Ext(path) != "" {
		return filepath.Base(path) + line
	}
	return elem
}

// ENGMODEL-LINKS: EM-CODE-ELEMENT, EM-SOURCE-BLOCK
// TRLC-LINKS: REQ-EMG-003
func groupedCodeElementEvidenceLabels(elems []string, lookup map[string]string) []string {
	if lookup == nil {
		lookup = codeElementEvidenceLabelLookup(elems)
	}
	out := []string{}
	for _, elem := range elems {
		key := codeElementEvidenceFileKey(elem)
		if key != "" {
			if label := strings.TrimSpace(lookup[key]); label != "" {
				out = appendUnique(out, label)
				continue
			}
		}
		label := codeElementEvidenceLabel(elem)
		if strings.TrimSpace(label) == "" {
			label = strings.TrimSpace(elem)
		}
		if label != "" {
			out = appendUnique(out, label)
		}
	}
	return uniqueSorted(out)
}

// TRLC-LINKS: REQ-EMG-003
func groupedCodeElementFileLabels(elems []string, lookup map[string]string) []string {
	if lookup == nil {
		lookup = codeElementEvidenceLabelLookup(elems)
	}
	out := []string{}
	for _, elem := range elems {
		label := ""
		if key := codeElementEvidenceFileKey(elem); key != "" {
			if full := strings.TrimSpace(lookup[key]); full != "" {
				label = strings.TrimSpace(strings.Split(full, ":")[0])
			}
		}
		if label == "" {
			path, _, ok := splitCodeEvidencePathLines(elem)
			if ok {
				label = filepath.Base(path)
			}
		}
		if label == "" {
			label = strings.TrimSpace(strings.Split(codeElementEvidenceLabel(elem), ":")[0])
		}
		if label != "" {
			out = appendUnique(out, label)
		}
	}
	return uniqueSorted(out)
}

// TRLC-LINKS: REQ-EMG-003
func codeElementEvidenceLabelLookup(elems []string) map[string]string {
	type bucket struct {
		base  string
		lines map[int]bool
	}
	buckets := map[string]*bucket{}
	for _, elem := range elems {
		path, lines, ok := splitCodeEvidencePathLines(elem)
		if !ok {
			continue
		}
		base := filepath.Base(path)
		key := strings.ToLower(base)
		b, ok := buckets[key]
		if !ok {
			b = &bucket{base: base, lines: map[int]bool{}}
			buckets[key] = b
		}
		for _, line := range lines {
			if line > 0 {
				b.lines[line] = true
			}
		}
	}
	out := map[string]string{}
	for key, b := range buckets {
		lines := make([]int, 0, len(b.lines))
		for line := range b.lines {
			lines = append(lines, line)
		}
		sort.Ints(lines)
		if len(lines) == 0 {
			out[key] = b.base
			continue
		}
		parts := make([]string, 0, len(lines))
		for _, line := range lines {
			parts = append(parts, fmt.Sprintf("%d", line))
		}
		out[key] = b.base + ":" + strings.Join(parts, ",")
	}
	return out
}

// TRLC-LINKS: REQ-EMG-003
func codeElementEvidenceFileKey(elem string) string {
	path, _, ok := splitCodeEvidencePathLines(elem)
	if !ok {
		return ""
	}
	return strings.ToLower(filepath.Base(path))
}

// TRLC-LINKS: REQ-EMG-003
func splitCodeEvidencePathLines(elem string) (string, []int, bool) {
	elem = sanitizeSourcePath(elem)
	if elem == "" {
		return "", nil, false
	}
	path := elem
	lines := []int{}
	if idx := strings.LastIndex(path, ":"); idx > 0 && idx < len(path)-1 {
		suffix := path[idx+1:]
		if parsed, ok := parseLineNumberList(suffix); ok {
			path = path[:idx]
			lines = parsed
		}
	}
	if filepath.Ext(path) == "" {
		return "", nil, false
	}
	return filepath.ToSlash(path), lines, true
}

// TRLC-LINKS: REQ-EMG-003
func parseLineNumberList(s string) ([]int, bool) {
	s = strings.TrimSpace(s)
	if s == "" {
		return nil, false
	}
	seen := map[int]bool{}
	out := []int{}
	for _, part := range strings.Split(s, ",") {
		part = strings.TrimSpace(part)
		if !allDigits(part) {
			return nil, false
		}
		n := 0
		for _, r := range part {
			n = n*10 + int(r-'0')
		}
		if n <= 0 || seen[n] {
			continue
		}
		seen[n] = true
		out = append(out, n)
	}
	if len(out) == 0 {
		return nil, false
	}
	sort.Ints(out)
	return out, true
}

// TRLC-LINKS: REQ-EMG-003
func joinEvidenceLineNumbers(lines []int) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d", line))
	}
	return strings.Join(parts, ",")
}

// TRLC-LINKS: REQ-EMG-003
func allDigits(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
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
func languageFromPath(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".rs":
		return "rust"
	default:
		return ""
	}
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
