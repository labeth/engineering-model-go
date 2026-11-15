// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// MCP tools that expose the OpenSSF Gemara rendering of the model. Each tool
// renders a Gemara document from the loaded bundle using the engmodel exporter
// (which builds go-gemara SDK structs) and returns the YAML plus a compact
// summary. gemara.validate re-checks every artifact through the SDK's type
// discriminator so an agent can confirm the output is SDK-recognized.

import (
	"fmt"
	"sort"

	gemara "github.com/gemaraproj/go-gemara"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
)

// gemaraToolKind maps an MCP tool name to its artifact short-name + type label.
var gemaraToolKind = map[string]struct {
	short string
	typ   string
}{
	"gemara.controlCatalog":    {"control-catalog", "ControlCatalog"},
	"gemara.threatCatalog":     {"threat-catalog", "ThreatCatalog"},
	"gemara.riskCatalog":       {"risk-catalog", "RiskCatalog"},
	"gemara.vectorCatalog":     {"vector-catalog", "VectorCatalog"},
	"gemara.capabilityCatalog": {"capability-catalog", "CapabilityCatalog"},
	"gemara.principleCatalog":  {"principle-catalog", "PrincipleCatalog"},
	"gemara.guidanceCatalog":   {"guidance-catalog", "GuidanceCatalog"},
	"gemara.policy":            {"policy", "Policy"},
	"gemara.lexicon":           {"lexicon", "Lexicon"},
	"gemara.mappingDocument":   {"control-threat-mapping", "MappingDocument"},
	"gemara.auditLog":          {"audit-log", "AuditLog"},
	"gemara.enforcementLog":    {"enforcement-log", "EnforcementLog"},
}

// gemaraArtifactTypes maps every produced artifact short-name to its Gemara type,
// used by gemara.validate.
var gemaraArtifactTypes = map[string]string{
	"vector-catalog":         "VectorCatalog",
	"capability-catalog":     "CapabilityCatalog",
	"control-catalog":        "ControlCatalog",
	"threat-catalog":         "ThreatCatalog",
	"risk-catalog":           "RiskCatalog",
	"principle-catalog":      "PrincipleCatalog",
	"guidance-catalog":       "GuidanceCatalog",
	"policy":                 "Policy",
	"lexicon":                "Lexicon",
	"control-threat-mapping": "MappingDocument",
	"audit-log":              "AuditLog",
	"enforcement-log":        "EnforcementLog",
	"evaluation-log":         "EvaluationLog",
}

// gemaraCountFields adds the artifact specific counts of a rendered result.
var gemaraCountFields = map[string]func(res engmodel.GemaraExportResult, out map[string]any){
	"gemara.controlCatalog": func(res engmodel.GemaraExportResult, out map[string]any) {
		out["controlCount"] = len(res.ControlCatalog.Controls)
		out["groupCount"] = len(res.ControlCatalog.Groups)
	},
	"gemara.threatCatalog": func(res engmodel.GemaraExportResult, out map[string]any) {
		out["threatCount"] = len(res.ThreatCatalog.Threats)
	},
	"gemara.riskCatalog": func(res engmodel.GemaraExportResult, out map[string]any) {
		out["riskCount"] = len(res.RiskCatalog.Risks)
	},
	"gemara.vectorCatalog": func(res engmodel.GemaraExportResult, out map[string]any) {
		out["vectorCount"] = len(res.VectorCatalog.Vectors)
	},
	"gemara.capabilityCatalog": func(res engmodel.GemaraExportResult, out map[string]any) {
		out["capabilityCount"] = len(res.CapabilityCatalog.Capabilities)
	},
}

// gemaraArtifact renders a single Gemara artifact for the given tool name.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func (s *Server) gemaraArtifact(toolName string) (map[string]any, error) {
	if toolName == "gemara.evaluationLog" {
		return s.gemaraEvaluationLogArtifact()
	}
	kind, ok := gemaraToolKind[toolName]
	if !ok {
		return nil, fmt.Errorf("unknown gemara tool %q", toolName)
	}
	res, err := engmodel.GenerateGemara(*s.bundle, engmodel.GemaraExportOptions{})
	if err != nil {
		return nil, err
	}
	out := map[string]any{"artifactType": kind.typ, "document": res.YAML[kind.short]}
	if addCounts, ok := gemaraCountFields[toolName]; ok {
		addCounts(res, out)
	}
	return out, nil
}

// gemaraEvaluationLogArtifact renders the Gemara evaluation log of the model.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func (s *Server) gemaraEvaluationLogArtifact() (map[string]any, error) {
	res, err := engmodel.GenerateGemaraEvaluationLog(*s.bundle, s.requirementsOrEmpty(), s.repoRoot, engmodel.GemaraExportOptions{})
	if err != nil {
		return nil, err
	}
	if !res.HasContent {
		return map[string]any{"artifactType": "EvaluationLog", "document": "", "note": "no controls to evaluate"}, nil
	}
	return map[string]any{
		"artifactType":    "EvaluationLog",
		"document":        res.YAML,
		"evaluationCount": len(res.EvaluationLog.Evaluations),
		"aggregateResult": res.EvaluationLog.Result.String(),
	}, nil
}

// requirementsOrEmpty returns the loaded requirements, or an empty document.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func (s *Server) requirementsOrEmpty() model.RequirementsDocument {
	if s.requirements == nil {
		return model.RequirementsDocument{}
	}
	return *s.requirements
}

// gemaraValidate renders every Gemara artifact, confirms the SDK recognizes
// each one via its metadata.type discriminator, and checks that every
// cross-artifact reference resolves within the rendered set.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func (s *Server) gemaraValidate() (map[string]any, error) {
	res, err := engmodel.GenerateGemara(*s.bundle, engmodel.GemaraExportOptions{})
	if err != nil {
		return nil, err
	}
	docs := map[string]string{}
	for k, v := range res.YAML {
		docs[k] = v
	}
	evalRes, err := engmodel.GenerateGemaraEvaluationLog(*s.bundle, s.requirementsOrEmpty(), s.repoRoot, engmodel.GemaraExportOptions{})
	if err != nil {
		return nil, err
	}
	if evalRes.HasContent {
		docs["evaluation-log"] = evalRes.YAML
	}
	return validateGemaraDocuments(s.bundle, docs), nil
}

// validateGemaraDocuments type-checks each rendered artifact, resolves its
// cross references and checks its treatment claims (dispositions, mitigated
// and accepted risks) against the model; valid is false when any check fails.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func validateGemaraDocuments(bundle *model.Bundle, docs map[string]string) map[string]any {
	artifacts, allValid := checkGemaraArtifactTypes(docs)
	xrefErrs := engmodel.CheckGemaraCrossReferences(docs)
	if xrefErrs == nil {
		xrefErrs = []engmodel.GemaraCrossReferenceError{}
	}
	consistencyErrs := []engmodel.GemaraCrossReferenceError{}
	if bundle != nil {
		consistencyErrs = append(consistencyErrs, engmodel.CheckGemaraModelConsistency(*bundle, docs)...)
	}
	return map[string]any{
		"valid":                  len(xrefErrs) == 0 && len(consistencyErrs) == 0 && allValid,
		"allValid":               allValid,
		"artifactCount":          len(artifacts),
		"artifacts":              artifacts,
		"crossReferenceErrors":   xrefErrs,
		"modelConsistencyErrors": consistencyErrs,
	}
}

// checkGemaraArtifactTypes checks, in artifact name order, that the SDK
// recognizes each present artifact as its expected type; the bool is false
// when one is not recognized.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func checkGemaraArtifactTypes(docs map[string]string) ([]map[string]any, bool) {
	artifacts := []map[string]any{}
	allValid := true
	names := make([]string, 0, len(gemaraArtifactTypes))
	for short := range gemaraArtifactTypes {
		names = append(names, short)
	}
	sort.Strings(names)
	for _, short := range names {
		expected := gemaraArtifactTypes[short]
		doc, present := docs[short]
		if !present || doc == "" {
			continue // conditional artifacts are only produced when the model supports them
		}
		entry, valid := gemaraArtifactEntry(short, expected, doc)
		allValid = allValid && valid
		artifacts = append(artifacts, entry)
	}
	return artifacts, allValid
}

// gemaraArtifactEntry reports whether the SDK recognizes doc as the expected
// artifact type, as an entry of the gemara.validate artifact list.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func gemaraArtifactEntry(short, expected, doc string) (map[string]any, bool) {
	at, derr := gemara.DetectType([]byte(doc))
	valid := derr == nil && at.String() == expected
	entry := map[string]any{"artifact": short, "artifactType": expected, "sdkRecognized": valid}
	if derr != nil {
		entry["error"] = derr.Error()
	}
	return entry, valid
}
