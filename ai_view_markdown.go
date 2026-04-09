package engmodel

import (
	"strconv"
	"strings"
)

func renderAIViewMarkdown(doc AIViewDocument) string {
	var b strings.Builder
	b.WriteString("# AI View\n\n")
	b.WriteString("Schema: `")
	b.WriteString(doc.SchemaVersion)
	b.WriteString("`\n\n")

	b.WriteString("## Model\n\n")
	b.WriteString("- ID: `")
	b.WriteString(doc.Model.ID)
	b.WriteString("`\n")
	b.WriteString("- Title: ")
	b.WriteString(doc.Model.Title)
	b.WriteString("\n")
	b.WriteString("- Counts: FG=")
	b.WriteString(intToString(doc.Model.Counts.FunctionalGroups))
	b.WriteString(", FU=")
	b.WriteString(intToString(doc.Model.Counts.FunctionalUnits))
	b.WriteString(", REQ=")
	b.WriteString(intToString(doc.Model.Counts.Requirements))
	b.WriteString(", RT=")
	b.WriteString(intToString(doc.Model.Counts.Runtime))
	b.WriteString(", CODE=")
	b.WriteString(intToString(doc.Model.Counts.Code))
	b.WriteString(", VER=")
	b.WriteString(intToString(doc.Model.Counts.Verification))
	b.WriteString(", VIEWS=")
	b.WriteString(intToString(doc.Model.Counts.Views))
	b.WriteString("\n\n")

	b.WriteString("## Entry Points\n\n")
	for _, ep := range doc.EntryPoints {
		b.WriteString("### ")
		b.WriteString(ep.ID)
		b.WriteString("\n\n")
		b.WriteString("- Kind: `")
		b.WriteString(ep.Kind)
		b.WriteString("`\n")
		b.WriteString("- Title: ")
		b.WriteString(ep.Title)
		b.WriteString("\n")
		b.WriteString("- Entities: ")
		b.WriteString(strings.Join(ep.EntityIDs, ", "))
		b.WriteString("\n\n")
	}

	b.WriteString("## Support Paths\n\n")
	for _, sp := range doc.SupportPaths {
		b.WriteString("- `")
		b.WriteString(sp.ID)
		b.WriteString("`: ")
		b.WriteString(strings.Join(sp.Path, " -> "))
		b.WriteString(" (confidence: ")
		b.WriteString(sp.Confidence)
		b.WriteString(")\n")
	}
	b.WriteString("\n")

	b.WriteString("## Entities\n\n")
	for _, e := range doc.Entities {
		b.WriteString("### ")
		b.WriteString(e.ID)
		b.WriteString("\n\n")
		b.WriteString("- Kind: `")
		b.WriteString(e.Kind)
		b.WriteString("`\n")
		b.WriteString("- Origin: `")
		b.WriteString(e.Origin)
		b.WriteString("`\n")
		if strings.TrimSpace(e.Status) != "" {
			b.WriteString("- Status: `")
			b.WriteString(e.Status)
			b.WriteString("`\n")
		}
		b.WriteString("- Title: ")
		b.WriteString(e.Title)
		b.WriteString("\n")
		if strings.TrimSpace(e.Summary) != "" {
			b.WriteString("- Summary: ")
			b.WriteString(e.Summary)
			b.WriteString("\n")
		}
		if len(e.RequirementIDs) > 0 {
			b.WriteString("- Requirements: ")
			b.WriteString(strings.Join(e.RequirementIDs, ", "))
			b.WriteString("\n")
		}
		if len(e.RuntimeIDs) > 0 {
			b.WriteString("- Runtime: ")
			b.WriteString(strings.Join(e.RuntimeIDs, ", "))
			b.WriteString("\n")
		}
		if len(e.CodeIDs) > 0 {
			b.WriteString("- Code: ")
			b.WriteString(strings.Join(e.CodeIDs, ", "))
			b.WriteString("\n")
		}
		if len(e.VerificationIDs) > 0 {
			b.WriteString("- Verification: ")
			b.WriteString(strings.Join(e.VerificationIDs, ", "))
			b.WriteString("\n")
		}
		if len(e.SourceRefs) > 0 {
			b.WriteString("- Source Refs: ")
			b.WriteString(strings.Join(e.SourceRefs, ", "))
			b.WriteString("\n")
		}
		b.WriteString("\n")
	}

	b.WriteString("## Source Blocks\n\n")
	for _, sb := range doc.SourceBlocks {
		b.WriteString("- `")
		b.WriteString(sb.ID)
		b.WriteString("` ")
		b.WriteString(sb.Path)
		if sb.LineStart > 0 {
			b.WriteString(":")
			b.WriteString(intToString(sb.LineStart))
		}
		b.WriteString(" [")
		b.WriteString(sb.Kind)
		b.WriteString("] entities=")
		b.WriteString(strings.Join(sb.EntityIDs, ", "))
		b.WriteString("\n")
	}

	return b.String()
}

func intToString(n int) string {
	return strconv.Itoa(n)
}
