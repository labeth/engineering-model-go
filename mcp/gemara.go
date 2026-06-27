// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// MCP tools that expose the OpenSSF Gemara rendering of the model. Each tool
// renders a Gemara document from the loaded bundle using the engmodel exporter
// (which builds go-gemara SDK structs) and returns the YAML plus a compact
// summary. gemara.validate re-checks every artifact through the SDK's type
// discriminator so an agent can confirm the output is SDK-recognized.

import (
	"fmt"

	gemara "github.com/gemaraproj/go-gemara"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/model"
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
}

// gemaraArtifact renders a single Gemara artifact for the given tool name.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func (s *Server) gemaraArtifact(toolName string) (map[string]any, error) {
	if toolName == "gemara.evaluationLog" {
		var reqs model.RequirementsDocument
		if s.requirements != nil {
			reqs = *s.requirements
		}
		res, err := engmodel.GenerateGemaraEvaluationLog(*s.bundle, reqs, s.repoRoot, engmodel.GemaraExportOptions{})
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

	kind, ok := gemaraToolKind[toolName]
	if !ok {
		return nil, fmt.Errorf("unknown gemara tool %q", toolName)
	}
	res, err := engmodel.GenerateGemara(*s.bundle, engmodel.GemaraExportOptions{})
	if err != nil {
		return nil, err
	}
	doc := res.YAML[kind.short]
	out := map[string]any{"artifactType": kind.typ, "document": doc}
	switch toolName {
	case "gemara.controlCatalog":
		out["controlCount"] = len(res.ControlCatalog.Controls)
		out["groupCount"] = len(res.ControlCatalog.Groups)
	case "gemara.threatCatalog":
		out["threatCount"] = len(res.ThreatCatalog.Threats)
	case "gemara.riskCatalog":
		out["riskCount"] = len(res.RiskCatalog.Risks)
	case "gemara.vectorCatalog":
		out["vectorCount"] = len(res.VectorCatalog.Vectors)
	case "gemara.capabilityCatalog":
		out["capabilityCount"] = len(res.CapabilityCatalog.Capabilities)
	}
	return out, nil
}

// gemaraValidate renders every Gemara artifact and confirms the SDK recognizes
// each one via its metadata.type discriminator.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func (s *Server) gemaraValidate() (map[string]any, error) {
	res, err := engmodel.GenerateGemara(*s.bundle, engmodel.GemaraExportOptions{})
	if err != nil {
		return nil, err
	}
	artifacts := []map[string]any{}
	allValid := true
	for short, expected := range gemaraArtifactTypes {
		doc, present := res.YAML[short]
		if !present || doc == "" {
			continue // conditional artifacts are only produced when the model supports them
		}
		at, derr := gemara.DetectType([]byte(doc))
		valid := derr == nil && at.String() == expected
		if !valid {
			allValid = false
		}
		entry := map[string]any{"artifact": short, "artifactType": expected, "sdkRecognized": valid}
		if derr != nil {
			entry["error"] = derr.Error()
		}
		artifacts = append(artifacts, entry)
	}
	return map[string]any{"allValid": allValid, "artifactCount": len(artifacts), "artifacts": artifacts}, nil
}
