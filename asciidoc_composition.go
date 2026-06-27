// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

// Renders a "System Composition & Hardware" chapter: hardware items and interfaces,
// referenced subsystems, the requirement allocation matrix materialized across the
// composed system-of-systems, a hardware/software interface view, and findings.

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// renderCompositionAsciiDocChapter renders hardware and composition content when present.
// TRLC-LINKS: REQ-EMG-024, REQ-EMG-025
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func renderCompositionAsciiDocChapter(bundle model.Bundle) string {
	a := bundle.Architecture.AuthoredArchitecture
	hasHW := len(a.HardwareItems) > 0 || len(a.HardwareInterfaces) > 0
	hasComp := len(bundle.Architecture.Composition.Subsystems) > 0
	if !hasHW && !hasComp {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n<<<\n== System Composition & Hardware\n\n")
	b.WriteString("This chapter shows the system architecture across hardware and software: hardware items and their ")
	b.WriteString("interfaces, the subsystems this system composes, and the allocation of this system's requirements onto them. ")
	b.WriteString("Subsystem references are downward-only and resolved from local subdirectories.\n\n")

	if len(a.HardwareItems) > 0 {
		b.WriteString("=== Hardware Items\n\n[cols=\"2,3,1,3,1\",options=\"header\"]\n|===\n|ID |Name |Kind |Hosts |Safety\n")
		for _, h := range a.HardwareItems {
			b.WriteString(fmt.Sprintf("|%s\n|%s\n|%s\n|%s\n|%s\n",
				escapeTableCell(h.ID), escapeTableCell(fallback(h.Name, h.ID)), escapeTableCell(h.Kind),
				escapeTableCell(strings.Join(h.Hosts, ", ")), escapeTableCell(fallback(h.SafetyLevel, "-"))))
		}
		b.WriteString("|===\n\n")
	}

	if len(a.HardwareInterfaces) > 0 {
		b.WriteString("=== Hardware Interfaces\n\n[cols=\"2,2,3,1,2\",options=\"header\"]\n|===\n|ID |Bus |From → To |Dir |Software IF\n")
		for _, hi := range a.HardwareInterfaces {
			b.WriteString(fmt.Sprintf("|%s\n|%s\n|%s → %s\n|%s\n|%s\n",
				escapeTableCell(hi.ID), escapeTableCell(hi.BusType),
				escapeTableCell(hi.From), escapeTableCell(hi.To),
				escapeTableCell(fallback(hi.Direction, "-")), escapeTableCell(fallback(hi.SoftwareInterfaceRef, "-"))))
		}
		b.WriteString("|===\n\n")
		if mmd := compositionHardwareMermaid(bundle); mmd != "" {
			b.WriteString("==== Hardware/Software Interface View\n\n[source,mermaid]\n----\n")
			b.WriteString(mmd)
			b.WriteString("\n----\n\n")
		}
	}

	if !hasComp {
		return b.String()
	}

	res, err := GenerateCompositionFromFile(bundle.ArchitecturePath)
	if err != nil {
		b.WriteString("NOTE: composition could not be resolved: " + escapeTableCell(err.Error()) + "\n\n")
		return b.String()
	}

	b.WriteString("=== Subsystems\n\n[cols=\"2,3,3\",options=\"header\"]\n|===\n|ID |Name |Reference\n")
	for _, sub := range bundle.Architecture.Composition.Subsystems {
		b.WriteString(fmt.Sprintf("|%s\n|%s\n|`%s`\n", escapeTableCell(sub.ID), escapeTableCell(fallback(sub.Name, sub.ID)), escapeTableCell(sub.Ref)))
	}
	b.WriteString("|===\n\n")

	if len(res.Allocations) > 0 {
		b.WriteString("=== Requirement Allocation Matrix\n\n[cols=\"2,2,2,1\",options=\"header\"]\n|===\n|Requirement |Subsystem |Target |Resolved\n")
		for _, m := range res.Allocations {
			mark := "yes"
			if !m.Resolved {
				mark = "NO — " + m.Note
			}
			b.WriteString(fmt.Sprintf("|%s\n|%s\n|%s\n|%s\n",
				escapeTableCell(m.Requirement), escapeTableCell(m.Subsystem), escapeTableCell(m.Target), escapeTableCell(mark)))
		}
		b.WriteString("|===\n\n")
		if mmd := compositionDelegationMermaid(bundle, res); mmd != "" {
			b.WriteString("==== Requirement Delegation View\n\n")
			b.WriteString("Each delegated requirement of this system links downward to the subsystem contract it binds. ")
			b.WriteString("Requirements not shown here carry no delegation link and are implemented in this system.\n\n")
			b.WriteString("[source,mermaid]\n----\n")
			b.WriteString(mmd)
			b.WriteString("\n----\n\n")
		}
	}

	var findings []validate.Diagnostic
	for _, d := range res.Diagnostics {
		if strings.HasPrefix(d.Code, "composition.") {
			findings = append(findings, d)
		}
	}
	if len(findings) > 0 {
		b.WriteString("=== Composition Findings\n\n[cols=\"2,1,4\",options=\"header\"]\n|===\n|Code |Severity |Message\n")
		for _, d := range findings {
			b.WriteString(fmt.Sprintf("|%s\n|%s\n|%s\n", escapeTableCell(d.Code), escapeTableCell(string(d.Severity)), escapeTableCell(d.Message)))
		}
		b.WriteString("|===\n\n")
	} else {
		b.WriteString("All subsystem required interfaces are satisfied and all allocations resolve to published contract identifiers.\n\n")
	}

	return b.String()
}

// compositionHardwareMermaid renders hardware items as nodes and hardware interfaces as edges.
// TRLC-LINKS: REQ-EMG-024
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VIEW-PROJECTION
func compositionHardwareMermaid(bundle model.Bundle) string {
	a := bundle.Architecture.AuthoredArchitecture
	if len(a.HardwareInterfaces) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("flowchart LR")
	for _, h := range a.HardwareItems {
		label := fallback(h.Name, h.ID)
		if len(h.Hosts) > 0 {
			label += " (" + strings.Join(h.Hosts, ", ") + ")"
		}
		b.WriteString(fmt.Sprintf("\n  %s[\"%s\"]", mermaidID(h.ID), mermaidLabel(label)))
	}
	for _, hi := range a.HardwareInterfaces {
		b.WriteString(fmt.Sprintf("\n  %s -->|%s| %s", mermaidID(hi.From), mermaidLabel(fallback(hi.BusType, "link")), mermaidID(hi.To)))
	}
	return b.String()
}

// compositionDelegationMermaid renders this system's delegated requirements on the
// left and each subsystem's bound contract target on the right, with one edge per
// allocation. It makes the downward delegation links visible as a graph, not only a
// table; requirements with no edge are implemented in this system.
// TRLC-LINKS: REQ-EMG-025, REQ-EMG-026
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-ALLOCATION-TRACE, FU-VIEW-PROJECTION
func compositionDelegationMermaid(bundle model.Bundle, res CompositionResult) string {
	if len(res.Allocations) == 0 {
		return ""
	}
	subName := map[string]string{}
	for _, sub := range bundle.Architecture.Composition.Subsystems {
		subName[sub.ID] = fallback(sub.Name, sub.ID)
	}
	// The delegation resolves to the specific subsystem requirement (contract ref)
	// when one is declared, so the reader sees exactly what satisfies it; otherwise
	// it falls back to the published contract target.
	targetKey := func(m MaterializedAllocation) string {
		if m.TargetRef != "" {
			return m.TargetRef
		}
		return m.Target
	}
	targetNode := func(m MaterializedAllocation) string { return mermaidID(m.Subsystem + "__" + targetKey(m)) }
	targetLabel := func(m MaterializedAllocation) string {
		if m.TargetRef != "" {
			return m.TargetRef + " (" + m.Target + ")"
		}
		return m.Target
	}

	var b strings.Builder
	b.WriteString("flowchart LR")

	// This system's delegated requirements.
	title := fallback(bundle.Architecture.Model.Title, fallback(bundle.Architecture.Model.ID, "This System"))
	b.WriteString(fmt.Sprintf("\n  subgraph SYS[\"%s\"]", mermaidLabel(title)))
	seenReq := map[string]bool{}
	for _, m := range res.Allocations {
		if !seenReq[m.Requirement] {
			seenReq[m.Requirement] = true
			b.WriteString(fmt.Sprintf("\n    %s[\"%s\"]", mermaidID(m.Requirement), mermaidLabel(m.Requirement)))
		}
	}
	b.WriteString("\n  end")

	// Each subsystem's bound contract targets.
	var subOrder []string
	subSeen := map[string]bool{}
	for _, m := range res.Allocations {
		if !subSeen[m.Subsystem] {
			subSeen[m.Subsystem] = true
			subOrder = append(subOrder, m.Subsystem)
		}
	}
	targetSeen := map[string]bool{}
	for _, sub := range subOrder {
		b.WriteString(fmt.Sprintf("\n  subgraph %s[\"%s\"]", mermaidID(sub), mermaidLabel(fallback(subName[sub], sub))))
		for _, m := range res.Allocations {
			if m.Subsystem != sub {
				continue
			}
			node := targetNode(m)
			if targetSeen[node] {
				continue
			}
			targetSeen[node] = true
			b.WriteString(fmt.Sprintf("\n    %s([\"%s\"])", node, mermaidLabel(targetLabel(m))))
		}
		b.WriteString("\n  end")
	}

	// Downward delegation edges to the specific requirement that satisfies each one.
	for _, m := range res.Allocations {
		label := "delegates to"
		if !m.Resolved {
			label = "unresolved"
		} else if m.TargetRef == "" {
			label = "delegates (no specific requirement)"
		} else if !m.TargetRefResolved {
			label = "delegates to (unresolved)"
		}
		b.WriteString(fmt.Sprintf("\n  %s -->|%s| %s", mermaidID(m.Requirement), mermaidLabel(label), targetNode(m)))
	}
	return b.String()
}

// lintRequirementInternalLinks warns when a requirement links to another
// requirement within the same document. Requirements carry no tiers; a
// requirement is either delegated to a subsystem (declared by a downward
// composition allocation) or implemented here. Linking requirements to one
// another inside a single document recreates tier coupling and is confusing
// across system boundaries, so it is flagged.
// TRLC-LINKS: REQ-EMG-023
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func lintRequirementInternalLinks(reqs model.RequirementsDocument) []validate.Diagnostic {
	ids := map[string]bool{}
	for _, r := range reqs.Requirements {
		ids[strings.TrimSpace(r.ID)] = true
	}
	var diags []validate.Diagnostic
	for _, r := range reqs.Requirements {
		for _, raw := range r.AppliesTo {
			target := strings.TrimSpace(raw)
			if ids[target] {
				diags = append(diags, validate.Diagnostic{
					Code: "requirement.internal_link", Severity: validate.SeverityWarning,
					Message: fmt.Sprintf("requirement %s links to requirement %s in the same document; requirements delegate to subsystems, not to other requirements here", r.ID, target),
					Path:    "requirements",
				})
			}
		}
	}
	return diags
}

// mermaidID sanitizes an id for use as a Mermaid node id.
// TRLC-LINKS: REQ-EMG-024
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
func mermaidID(s string) string {
	var b strings.Builder
	for _, r := range s {
		if (r >= 'a' && r <= 'z') || (r >= 'A' && r <= 'Z') || (r >= '0' && r <= '9') || r == '_' {
			b.WriteRune(r)
		} else {
			b.WriteRune('_')
		}
	}
	out := b.String()
	if out == "" {
		return "n"
	}
	return out
}

// mermaidLabel escapes a label for a Mermaid node.
// TRLC-LINKS: REQ-EMG-024
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
func mermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "'")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}
