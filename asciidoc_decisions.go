// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

func buildDecisionSections(in []model.Decision) []asciidocDecisionSection {
	out := make([]asciidocDecisionSection, 0, len(in))
	for _, d := range in {
		consequences := make([]string, 0, len(d.Consequences))
		for _, c := range d.Consequences {
			c = strings.TrimSpace(c)
			if c != "" {
				consequences = append(consequences, c)
			}
		}
		out = append(out, asciidocDecisionSection{
			ID:           strings.TrimSpace(d.ID),
			Title:        strings.TrimSpace(d.Title),
			Status:       strings.TrimSpace(d.Status),
			Date:         strings.TrimSpace(d.Date),
			Context:      strings.TrimSpace(d.Context),
			Decision:     strings.TrimSpace(d.Decision),
			Consequences: consequences,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func renderDecisionsDocument(decisions []asciidocDecisionSection) string {
	if len(decisions) == 0 {
		return ""
	}
	var b strings.Builder
	b.WriteString("= Architecture Decision Records\n")
	b.WriteString(":toc:\n")
	b.WriteString(":toclevels: 2\n")
	b.WriteString(":sectnums:\n\n")
	for i, d := range decisions {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString("[[")
		b.WriteString(d.ID)
		b.WriteString("]]\n")
		b.WriteString("== ")
		b.WriteString(d.ID)
		b.WriteString(": ")
		b.WriteString(d.Title)
		b.WriteString("\n\n")
		b.WriteString("Status:: ")
		b.WriteString(d.Status)
		b.WriteString("\n")
		b.WriteString("Date:: ")
		b.WriteString(d.Date)
		b.WriteString("\n\n")
		b.WriteString("=== Context\n\n")
		b.WriteString(d.Context)
		b.WriteString("\n\n")
		b.WriteString("=== Decision\n\n")
		b.WriteString(d.Decision)
		b.WriteString("\n\n")
		b.WriteString("=== Consequences\n\n")
		for _, c := range d.Consequences {
			b.WriteString("- ")
			b.WriteString(c)
			b.WriteString("\n")
		}
	}
	return b.String()
}
