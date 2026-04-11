package engmodel

import (
	"fmt"
	"html"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

func appendMermaidClassDefs(lines []string) []string {
	return append(lines,
		"  classDef system_boundary fill:#f5f5f5,stroke:#424242,stroke-width:2px,color:#212121;",
		"  classDef functional_group fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px,color:#1b5e20;",
		"  classDef functional_unit fill:#e3f2fd,stroke:#0d47a1,stroke-width:1px,color:#0d47a1;",
		"  classDef actor fill:#fff8e1,stroke:#ef6c00,stroke-width:1px,color:#bf360c;",
		"  classDef referenced_element fill:#f3e5f5,stroke:#6a1b9a,stroke-width:1px,color:#4a148c;",
		"  classDef attack_vector fill:#ffebee,stroke:#b71c1c,stroke-width:1px,color:#7f0000;",
		"  classDef requirement fill:#fffde7,stroke:#f9a825,stroke-width:1px,color:#7f6000;",
		"  classDef verification fill:#fce4ec,stroke:#ad1457,stroke-width:1px,color:#880e4f;",
		"  classDef runtime_element fill:#b2ebf2,stroke:#00838f,stroke-width:1px,color:#006064;",
		"  classDef deployment_element fill:#d7ccc8,stroke:#4e342e,stroke-width:1px,color:#2f1b14;",
		"  classDef code_element fill:#eceff1,stroke:#37474f,stroke-width:1px,color:#263238;",
	)
}

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

	colSpec := make([]string, 0, len(fuIDs)+1)
	colSpec = append(colSpec, "2")
	for range fuIDs {
		colSpec = append(colSpec, "1")
	}
	lines := []string{
		"[cols=\"" + strings.Join(colSpec, ",") + "\",options=\"header\"]",
		"|===",
		"|Requirement",
	}
	for _, fu := range fuIDs {
		lines = append(lines, "|"+escapeTableCell(fu))
	}
	for _, reqID := range reqIDs {
		lines = append(lines, "|"+escapeTableCell(reqID))
		for _, fu := range fuIDs {
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

func keysFromSet(m map[string]bool) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}

func escapeTableCell(s string) string {
	s = strings.TrimSpace(s)
	s = strings.ReplaceAll(s, "|", "\\|")
	return s
}

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

	for _, m := range a.Mappings {
		label := strings.TrimSpace(m.Description)
		if label == "" {
			label = strings.TrimSpace(m.Type)
		}
		label = escapeMermaidLabel(label)
		switch m.Type {
		case "interacts_with":
			lines = append(lines, fmt.Sprintf("  ACT_%s -->|%s| FU_%s", sanitizeNode(m.From), label, sanitizeNode(m.To)))
		case "depends_on":
			if strings.HasPrefix(m.To, "REF-") {
				lines = append(lines, fmt.Sprintf("  FU_%s -->|%s| REF_%s", sanitizeNode(m.From), label, sanitizeNode(m.To)))
			}
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

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

	const maxColumnsPerBand = 10
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
	codeByOwner := map[string][]string{}
	for _, c := range code {
		owner := strings.TrimSpace(c.Owner)
		if owner == "" || owner == "unresolved" || c.Kind != "source_file" {
			continue
		}
		codeByOwner[owner] = append(codeByOwner[owner], moduleFromPath(codeItemPath(c)))
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
			if len(rtNodes) == 0 && len(codeByOwner[fu]) > 0 {
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
			for _, cm := range uniqueSorted(codeByOwner[fu]) {
				cmNode := "CDC_" + sanitizeNode(fu+"-"+cm)
				lines = append(lines, "  "+cmNode+"[\""+escapeMermaidLabel(cm)+"\"]:::code_element")
				if primaryRTNode != "" {
					lines = append(lines, "  "+primaryRTNode+" -->|implemented_by| "+cmNode)
				} else {
					lines = append(lines, "  "+fuNode+" -->|code evidence| "+cmNode)
				}
				lines = append(lines, "  "+reqNode+" -->|code trace| "+cmNode)
			}
		}
		for _, v := range checksByReq[strings.TrimSpace(r.ID)] {
			status := strings.TrimSpace(v.Status)
			checkLabel := strings.TrimSpace(v.ID)
			if status != "" {
				checkLabel = checkLabel + " (" + status + ")"
			}
			verNode := "VERC_" + sanitizeNode(v.ID)
			lines = append(lines, "  "+verNode+"[\""+escapeMermaidLabel(checkLabel)+"\"]:::verification")
			lines = append(lines, "  "+verNode+" -->|verifies| "+reqNode)
			for _, ce := range uniqueSorted(v.CodeElements) {
				elem := strings.TrimSpace(ce)
				if elem == "" {
					continue
				}
				label := moduleFromPath(elem)
				if strings.TrimSpace(label) == "" {
					label = elem
				}
				ceNode := "VEC_" + sanitizeNode(v.ID+"-"+elem)
				lines = append(lines, "  "+ceNode+"[\""+escapeMermaidLabel(label)+"\"]:::code_element")
				lines = append(lines, "  "+verNode+" -->|implemented_by| "+ceNode)
			}
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

func sanitizeNode(s string) string {
	repl := strings.NewReplacer("-", "_", " ", "_", ".", "_", "/", "_", ":", "_", "\\", "_")
	out := repl.Replace(strings.ToUpper(strings.TrimSpace(s)))
	if out == "" {
		out = "NODE"
	}
	return out
}

func escapeMermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

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
