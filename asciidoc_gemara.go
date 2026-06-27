// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

// Renders a first-class "Gemara GRC Model" chapter into the architecture
// publication. The chapter summarizes the OpenSSF Gemara catalogs produced by
// the Gemara exporter (the same typed documents validated against the Gemara
// schemas), giving the human-readable document a GRC view of the model.

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func renderGemaraAsciiDocChapter(bundle model.Bundle) string {
	res, err := GenerateGemara(bundle, GemaraExportOptions{})
	if err != nil {
		return ""
	}

	var b strings.Builder
	b.WriteString("\n<<<\n== Gemara GRC Model\n\n")
	b.WriteString("This chapter renders the model as OpenSSF Gemara (https://gemara.openssf.org) documents. ")
	b.WriteString("Each artifact is built with the official `go-gemara` SDK types and validated against the published Gemara CUE schemas. ")
	b.WriteString("Generate the full YAML set with `enggemara`; query it via the `gemara.*` MCP tools.\n\n")

	// Layer coverage summary across all produced artifact types.
	b.WriteString("[cols=\"1,3,1\",options=\"header\"]\n|===\n|Gemara Layer |Artifact |Entries\n")
	b.WriteString(gemaraRow("L1", "Vector Catalog", len(res.VectorCatalog.Vectors)))
	b.WriteString(gemaraRow("L1", "Principle Catalog", len(res.PrincipleCatalog.Principles)))
	b.WriteString(gemaraRow("L1", "Guidance Catalog", len(res.GuidanceCatalog.Guidelines)))
	b.WriteString(gemaraRow("L2", "Capability Catalog", len(res.CapabilityCatalog.Capabilities)))
	b.WriteString(gemaraRow("L2", "Control Catalog", len(res.ControlCatalog.Controls)))
	b.WriteString(gemaraRow("L2", "Threat Catalog", len(res.ThreatCatalog.Threats)))
	b.WriteString(gemaraRow("L3", "Risk Catalog", len(res.RiskCatalog.Risks)))
	if res.HasPolicy {
		b.WriteString(gemaraRow("L3", "Policy", len(res.Policy.Adherence.AssessmentPlans)))
	}
	if res.HasMapping {
		b.WriteString(gemaraRow("-", "Control→Threat Mapping", len(res.ControlThreatMapping.Mappings)))
	}
	b.WriteString(gemaraRow("-", "Lexicon", len(res.Lexicon.Terms)))
	if res.HasAudit {
		b.WriteString(gemaraRow("L7", "Audit Log", len(res.AuditLog.Results)))
	}
	if res.HasEnforcement {
		b.WriteString(gemaraRow("L6", "Enforcement Log", len(res.EnforcementLog.Actions)))
	}
	b.WriteString("|===\n\n")

	// Controls (L2) with assessment requirement counts.
	if len(res.ControlCatalog.Controls) > 0 {
		b.WriteString("=== Controls (L2)\n\n")
		b.WriteString("[cols=\"2,4,2,1\",options=\"header\"]\n|===\n|Control |Objective |Group |Assessment Requirements\n")
		for _, c := range res.ControlCatalog.Controls {
			b.WriteString(fmt.Sprintf("|%s\n|%s\n|%s\n|%d\n",
				escapeTableCell(c.Id),
				escapeTableCell(gemaraTruncate(c.Objective, 140)),
				escapeTableCell(c.Group),
				len(c.AssessmentRequirements)))
		}
		b.WriteString("|===\n\n")
	}

	// Threats (L2).
	if len(res.ThreatCatalog.Threats) > 0 {
		b.WriteString("=== Threats (L2)\n\n")
		b.WriteString("[cols=\"2,4,2\",options=\"header\"]\n|===\n|Threat |Title |Group\n")
		for _, t := range res.ThreatCatalog.Threats {
			b.WriteString(fmt.Sprintf("|%s\n|%s\n|%s\n",
				escapeTableCell(t.Id),
				escapeTableCell(gemaraTruncate(t.Title, 120)),
				escapeTableCell(t.Group)))
		}
		b.WriteString("|===\n\n")
	}

	// Risks (L3) with derived severity.
	if len(res.RiskCatalog.Risks) > 0 {
		b.WriteString("=== Risks (L3)\n\n")
		b.WriteString("[cols=\"2,4,1\",options=\"header\"]\n|===\n|Risk |Title |Severity\n")
		for _, r := range res.RiskCatalog.Risks {
			b.WriteString(fmt.Sprintf("|%s\n|%s\n|%s\n",
				escapeTableCell(r.Id),
				escapeTableCell(gemaraTruncate(r.Title, 120)),
				escapeTableCell(r.Severity.String())))
		}
		b.WriteString("|===\n\n")
	}

	return b.String()
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func gemaraRow(layer, artifact string, count int) string {
	return fmt.Sprintf("|%s\n|%s\n|%d\n", layer, artifact, count)
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func gemaraTruncate(s string, n int) string {
	s = strings.TrimSpace(strings.ReplaceAll(s, "\n", " "))
	if len(s) <= n {
		return s
	}
	return strings.TrimSpace(s[:n]) + "…"
}
