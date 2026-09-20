// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"reflect"
	"strings"
)

const AuthoringContractVersion = "engineering-model.authoring.v2"

// AuthoringContract is compact machine context for editing the canonical YAML set.
type AuthoringContract struct {
	ContractVersion   string                `json:"contractVersion"`
	CanonicalFormat   string                `json:"canonicalFormat"`
	SchemaAuthority   string                `json:"schemaAuthority"`
	Documents         []AuthoringDocument   `json:"documents"`
	StableIdentifiers StableIdentifierGuide `json:"stableIdentifiers"`
	Dependencies      DependencyGuide       `json:"dependencies"`
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

type DependencyGuide struct {
	PreferredSourceField string   `json:"preferredSourceField"`
	SourceKinds          []string `json:"sourceKinds"`
	WorkspaceFile        string   `json:"workspaceFile"`
	WorkspaceSchemaPath  string   `json:"workspaceSchemaPath"`
	RegistryEnvironment  string   `json:"registryEnvironment"`
	CacheDirectory       string   `json:"cacheDirectory"`
	Rules                []string `json:"rules"`
}

// BuildAuthoringContract describes the loaded input model without copying its domain graph.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-045, REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-053
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func BuildAuthoringContract(bundle Bundle) AuthoringContract {
	baseDir := filepath.Dir(bundle.ManifestPath)
	relative := func(path string) string {
		if rel, err := filepath.Rel(baseDir, path); err == nil {
			return filepath.ToSlash(rel)
		}
		return filepath.ToSlash(path)
	}
	documents := []AuthoringDocument{
		{Kind: "manifest", Path: relative(bundle.ManifestPath), SchemaVersion: contractSchemaVersion(bundle.Manifest.SchemaVersion), SchemaPath: "model/schema/manifest.cue", Required: true, TopLevelFields: yamlTopLevelFields(ManifestDocument{})},
		{Kind: "catalog", Path: relative(bundle.CatalogPath), SchemaVersion: contractSchemaVersion(bundle.Catalog.SchemaVersion), SchemaPath: "model/schema/catalog.cue", Required: true, TopLevelFields: yamlTopLevelFields(CatalogDocument{})},
		{Kind: "requirements", Path: relative(bundle.RequirementsPath), SchemaVersion: contractSchemaVersion(bundle.Requirements.SchemaVersion), SchemaPath: "model/schema/requirements.cue", Required: true, TopLevelFields: yamlTopLevelFields(RequirementsDocument{})},
		{Kind: "architecture", Path: relative(bundle.ArchitecturePath), SchemaVersion: contractSchemaVersion(bundle.Architecture.SchemaVersion), SchemaPath: "model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(ArchitectureInputDocument{})},
		{Kind: "behavior", Path: relative(bundle.BehaviorPath), SchemaVersion: contractSchemaVersion(bundle.Behavior.SchemaVersion), SchemaPath: "model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(BehaviorDocument{})},
		{Kind: "assurance", Path: relative(bundle.AssurancePath), SchemaVersion: contractSchemaVersion(bundle.Assurance.SchemaVersion), SchemaPath: "model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(AssuranceDocument{})},
		{Kind: "compliance", Path: relative(bundle.CompliancePath), SchemaVersion: contractSchemaVersion(bundle.Compliance.SchemaVersion), SchemaPath: "model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(ComplianceDocument{})},
		{Kind: "views", Path: relative(bundle.ViewsPath), SchemaVersion: contractSchemaVersion(bundle.Views.SchemaVersion), SchemaPath: "model/schema/domain_documents.cue", Required: true, TopLevelFields: yamlTopLevelFields(ViewsDocument{})},
		{Kind: "decisions", Path: relative(bundle.DecisionsPath), SchemaVersion: contractSchemaVersion(bundle.Decisions.SchemaVersion), SchemaPath: "model/schema/decisions.cue", Required: true, TopLevelFields: yamlTopLevelFields(DecisionsDocument{})},
	}
	if bundle.Aviation != nil {
		documents = append(documents, AuthoringDocument{Kind: "aviation", Path: relative(bundle.AviationPath), SchemaVersion: contractSchemaVersion(bundle.Aviation.SchemaVersion), SchemaPath: "model/schema/aviation.cue", Required: false, TopLevelFields: yamlTopLevelFields(AviationDocument{})})
	}
	return AuthoringContract{
		ContractVersion: AuthoringContractVersion,
		CanonicalFormat: "YAML",
		SchemaAuthority: "CUE",
		Documents:       documents,
		StableIdentifiers: StableIdentifierGuide{
			Pattern: "^[A-Z][A-Z0-9-]*$",
			Prefixes: map[string]string{
				"REQ": "requirement", "FU": "functional unit", "FG": "functional group",
				"IF": "interface", "DO": "data object", "FLOW": "flow", "ACT": "actor",
				"CTRL": "control", "STATE": "state", "EVT": "event", "ADR": "decision",
			},
		},
		Dependencies: DependencyGuide{
			PreferredSourceField: "dependencies[]",
			SourceKinds:          []string{"cue-module"},
			WorkspaceFile:        WorkspaceFileName,
			WorkspaceSchemaPath:  "model/schema/workspace.cue",
			RegistryEnvironment:  "CUE_REGISTRY",
			CacheDirectory:       ".engmod/modules",
			Rules: []string{
				"Declare dependencies only in engmod.yml with a unique alias.",
				"Use a major-qualified CUE module path, exact semantic version, and at least one selected publication.",
				"Use engmod.work.yml only for local module-to-repository replacements.",
				"Reference imported stable identifiers as alias::ID.",
				"Only identifiers selected through dependency publications are visible.",
			},
		},
		AuthoringOrder: []string{"manifest", "catalog", "requirements", "architecture", "behavior", "assurance", "compliance", "aviation", "views", "decisions"},
		Invariants: []string{
			"Define controlled vocabulary and stable IDs before referencing them.",
			"Represent each semantic concept once; outputs are projections of the canonical graph.",
			"Use requirements deltas through engflow instead of rewriting requirements.yml directly.",
			"Run CUE validation before strict Go decoding and cross-document validation.",
		},
		Compatibility: []string{},
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-045
func contractSchemaVersion(version int) int {
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
