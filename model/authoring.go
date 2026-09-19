// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"reflect"
	"strings"
)

const AuthoringContractVersion = "engineering-model.authoring.v1"

// AuthoringContract is compact machine context for editing the canonical YAML set.
type AuthoringContract struct {
	ContractVersion   string                `json:"contractVersion"`
	CanonicalFormat   string                `json:"canonicalFormat"`
	SchemaAuthority   string                `json:"schemaAuthority"`
	Documents         []AuthoringDocument   `json:"documents"`
	StableIdentifiers StableIdentifierGuide `json:"stableIdentifiers"`
	AuthoringOrder    []string              `json:"authoringOrder"`
	Invariants        []string              `json:"invariants"`
	Compatibility     []string              `json:"compatibility"`
}

type AuthoringDocument struct {
	Kind           string   `json:"kind"`
	Path           string   `json:"path"`
	SchemaVersion  int      `json:"schemaVersion"`
	SchemaPath     string   `json:"schemaPath"`
	Required       bool     `json:"required"`
	TopLevelFields []string `json:"topLevelFields"`
}

type StableIdentifierGuide struct {
	Pattern  string            `json:"pattern"`
	Prefixes map[string]string `json:"prefixes"`
}

// BuildAuthoringContract describes the loaded input model without copying its domain graph.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-045
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func BuildAuthoringContract(bundle Bundle) AuthoringContract {
	baseDir := filepath.Dir(bundle.ArchitecturePath)
	relative := func(path string) string {
		if rel, err := filepath.Rel(baseDir, path); err == nil {
			return filepath.ToSlash(rel)
		}
		return filepath.ToSlash(path)
	}
	explicit := bundle.Architecture.Model.Documents
	return AuthoringContract{
		ContractVersion: AuthoringContractVersion,
		CanonicalFormat: "YAML",
		SchemaAuthority: "CUE",
		Documents: []AuthoringDocument{
			{Kind: "architecture", Path: relative(bundle.ArchitecturePath), SchemaVersion: contractSchemaVersion(bundle.Architecture.SchemaVersion), SchemaPath: "model/schema/architecture.cue", Required: true, TopLevelFields: yamlTopLevelFields(ArchitectureDocument{})},
			{Kind: "catalog", Path: relative(bundle.CatalogPath), SchemaVersion: contractSchemaVersion(bundle.Catalog.SchemaVersion), SchemaPath: "model/schema/catalog.cue", Required: true, TopLevelFields: yamlTopLevelFields(CatalogDocument{})},
			{Kind: "requirements", Path: relative(bundle.RequirementsPath), SchemaVersion: contractSchemaVersion(bundle.Requirements.SchemaVersion), SchemaPath: "model/schema/requirements.cue", Required: strings.TrimSpace(explicit.Requirements) != "", TopLevelFields: yamlTopLevelFields(RequirementsDocument{})},
			{Kind: "design", Path: relative(bundle.DesignPath), SchemaVersion: contractSchemaVersion(bundle.Design.SchemaVersion), SchemaPath: "model/schema/design.cue", Required: strings.TrimSpace(explicit.Design) != "", TopLevelFields: yamlTopLevelFields(DesignDocument{})},
			{Kind: "decisions", Path: relative(bundle.DecisionsPath), SchemaVersion: contractSchemaVersion(bundle.Decisions.SchemaVersion), SchemaPath: "model/schema/decisions.cue", Required: strings.TrimSpace(explicit.Decisions) != "", TopLevelFields: yamlTopLevelFields(DecisionsDocument{})},
		},
		StableIdentifiers: StableIdentifierGuide{
			Pattern: "^[A-Z][A-Z0-9-]*$",
			Prefixes: map[string]string{
				"REQ": "requirement", "FU": "functional unit", "FG": "functional group",
				"IF": "interface", "DO": "data object", "FLOW": "flow", "ACT": "actor",
				"CTRL": "control", "STATE": "state", "EVT": "event", "ADR": "decision",
			},
		},
		AuthoringOrder: []string{"catalog", "architecture", "requirements", "design", "decisions"},
		Invariants: []string{
			"Define controlled vocabulary and stable IDs before referencing them.",
			"Represent each semantic concept once; outputs are projections of the canonical graph.",
			"Use requirements deltas through engflow instead of rewriting requirements.yml directly.",
			"Run CUE validation before strict Go decoding and cross-document validation.",
		},
		Compatibility: []string{
			"Missing schemaVersion means version 1.",
			"model.baseCatalogRef remains accepted when model.documents.catalog is absent.",
			"requirements.yml, design.yml, and decisions.yml remain default companion paths.",
			"All existing generated output formats remain unchanged.",
		},
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-045
func contractSchemaVersion(version int) int {
	if version == 0 {
		return CurrentSchemaVersion
	}
	return version
}

// TRLC-LINKS: REQ-EMG-045
func yamlTopLevelFields(value any) []string {
	t := reflect.TypeOf(value)
	fields := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		name := strings.Split(t.Field(i).Tag.Get("yaml"), ",")[0]
		if name != "" && name != "-" {
			fields = append(fields, name)
		}
	}
	return fields
}
