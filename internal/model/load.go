// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

const CurrentSchemaVersion = 2

// ResolveDocumentReferences validates the explicit schema-v2 document map.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-053
func ResolveDocumentReferences(meta ModelMeta) (DocumentReferences, error) {
	refs := meta.Documents
	for kind, path := range map[string]string{
		"catalog": refs.Catalog, "requirements": refs.Requirements,
		"architecture": refs.Architecture, "behavior": refs.Behavior,
		"assurance": refs.Assurance, "compliance": refs.Compliance,
		"views": refs.Views, "decisions": refs.Decisions,
	} {
		if path == "" {
			return DocumentReferences{}, fmt.Errorf("model.documents.%s is required", kind)
		}
	}
	return refs, nil
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func EffectiveSchemaVersion(version int) (int, error) {
	if version != CurrentSchemaVersion {
		return 0, fmt.Errorf("unsupported schemaVersion %d; supported version is %d", version, CurrentSchemaVersion)
	}
	return version, nil
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046, REQ-EMG-049, REQ-EMG-051, REQ-EMG-053, REQ-EMG-054
func LoadBundle(manifestPath string) (Bundle, error) {
	if filepath.Base(filepath.Clean(manifestPath)) != "engmod.yml" {
		return Bundle{}, fmt.Errorf("engmod.yml is the only supported model entry point")
	}
	manifestPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return Bundle{}, fmt.Errorf("resolve manifest path: %w", err)
	}

	var manifest ManifestDocument
	if err := decodeYAMLFile(manifestPath, &manifest); err != nil {
		return Bundle{}, fmt.Errorf("decode manifest file: %w", err)
	}
	manifest.SchemaVersion, err = EffectiveSchemaVersion(manifest.SchemaVersion)
	if err != nil {
		return Bundle{}, fmt.Errorf("manifest schema: %w", err)
	}

	baseDir := filepath.Dir(manifestPath)
	refs := manifest.Documents
	catalogPath := filepath.Join(baseDir, refs.Catalog)
	var catalog CatalogDocument
	if err := decodeYAMLFile(catalogPath, &catalog); err != nil {
		return Bundle{}, fmt.Errorf("decode catalog file: %w", err)
	}

	decisionsPath := filepath.Join(baseDir, refs.Decisions)
	var decisions DecisionsDocument
	if err := decodeYAMLFile(decisionsPath, &decisions); err != nil {
		return Bundle{}, fmt.Errorf("decode decisions file: %w", err)
	}

	requirementsPath := filepath.Join(baseDir, refs.Requirements)
	var requirements RequirementsDocument
	if err := decodeYAMLFile(requirementsPath, &requirements); err != nil {
		return Bundle{}, fmt.Errorf("decode requirements file: %w", err)
	}

	architecturePath := filepath.Join(baseDir, refs.Architecture)
	var architectureInput ArchitectureInputDocument
	if err := decodeYAMLFile(architecturePath, &architectureInput); err != nil {
		return Bundle{}, fmt.Errorf("decode architecture file: %w", err)
	}

	behaviorPath := filepath.Join(baseDir, refs.Behavior)
	var behavior BehaviorDocument
	if err := decodeYAMLFile(behaviorPath, &behavior); err != nil {
		return Bundle{}, fmt.Errorf("decode behavior file: %w", err)
	}

	assurancePath := filepath.Join(baseDir, refs.Assurance)
	var assurance AssuranceDocument
	if err := decodeYAMLFile(assurancePath, &assurance); err != nil {
		return Bundle{}, fmt.Errorf("decode assurance file: %w", err)
	}

	compliancePath := filepath.Join(baseDir, refs.Compliance)
	var compliance ComplianceDocument
	if err := decodeYAMLFile(compliancePath, &compliance); err != nil {
		return Bundle{}, fmt.Errorf("decode compliance file: %w", err)
	}

	viewsPath := filepath.Join(baseDir, refs.Views)
	var views ViewsDocument
	if err := decodeYAMLFile(viewsPath, &views); err != nil {
		return Bundle{}, fmt.Errorf("decode views file: %w", err)
	}

	var aviation *AviationDocument
	aviationPath := ""
	if refs.Aviation != "" {
		aviationPath = filepath.Join(baseDir, refs.Aviation)
		var document AviationDocument
		if err := decodeYAMLFile(aviationPath, &document); err != nil {
			return Bundle{}, fmt.Errorf("decode aviation file: %w", err)
		}
		if _, err := EffectiveSchemaVersion(document.SchemaVersion); err != nil {
			return Bundle{}, fmt.Errorf("aviation schema: %w", err)
		}
		if diagnostics := ValidateAviation(document, requirements, architectureInput); len(diagnostics) > 0 {
			return Bundle{}, &AviationValidationError{Diagnostics: diagnostics}
		}
		aviation = &document
	}

	for kind, version := range map[string]int{
		"catalog": catalog.SchemaVersion, "requirements": requirements.SchemaVersion,
		"architecture": architectureInput.SchemaVersion, "behavior": behavior.SchemaVersion,
		"assurance": assurance.SchemaVersion, "compliance": compliance.SchemaVersion,
		"views": views.SchemaVersion, "decisions": decisions.SchemaVersion,
	} {
		if _, err := EffectiveSchemaVersion(version); err != nil {
			return Bundle{}, fmt.Errorf("%s schema: %w", kind, err)
		}
	}

	input := architectureInput.Architecture
	authored := AuthoredArchitecture{
		FunctionalGroups: input.FunctionalGroups, FunctionalUnits: input.FunctionalUnits,
		Actors: input.Actors, ReferencedElements: input.ReferencedElements,
		Interfaces: input.Interfaces, DataObjects: input.DataObjects,
		DeploymentTargets: input.DeploymentTargets, HardwareItems: input.HardwareItems,
		HardwareInterfaces: input.HardwareInterfaces,
		States:             behavior.Behavior.States, Events: behavior.Behavior.Events,
		Flows: behavior.Behavior.Flows, Mappings: behavior.Behavior.Relationships,
		AttackVectors: assurance.Assurance.AttackVectors, Controls: assurance.Assurance.Controls,
		Risks: assurance.Assurance.Risks, POAMItems: assurance.Assurance.POAMItems,
		TrustBoundaries:      assurance.Assurance.TrustBoundaries,
		ThreatScenarios:      assurance.Assurance.ThreatScenarios,
		ThreatAssumptions:    assurance.Assurance.ThreatAssumptions,
		ThreatOutOfScope:     assurance.Assurance.ThreatOutOfScope,
		ThreatMitigations:    assurance.Assurance.ThreatMitigations,
		ControlVerifications: assurance.Assurance.ControlVerifications,
	}
	arch := ArchitectureDocument{
		SchemaVersion: CurrentSchemaVersion,
		Model: ModelMeta{
			ID: manifest.Module.ModelID, Title: manifest.Module.Title,
			Introduction: manifest.Module.Introduction, Documents: manifest.Documents,
		},
		Decisions: decisions.Decisions, AuthoredArchitecture: authored,
		Semantics:  input.Semantics,
		Compliance: compliance.Compliance, Contract: input.Contract,
		Composition: aggregateComposition(input.Composition), InferenceHints: manifest.InferenceHints,
		NAF: views.NAF, Views: views.Views,
	}
	design := DesignDocument{SchemaVersion: CurrentSchemaVersion, Design: views.Design}

	return Bundle{
		ManifestPath:     manifestPath,
		ArchitecturePath: architecturePath,
		BehaviorPath:     behaviorPath,
		AssurancePath:    assurancePath,
		CompliancePath:   compliancePath,
		ViewsPath:        viewsPath,
		CatalogPath:      catalogPath,
		DecisionsPath:    decisionsPath,
		RequirementsPath: requirementsPath,
		DesignPath:       viewsPath,
		AviationPath:     aviationPath,
		Manifest:         manifest,
		Architecture:     arch,
		Behavior:         behavior,
		Assurance:        assurance,
		Compliance:       compliance,
		Views:            views,
		Catalog:          catalog,
		Decisions:        decisions,
		Requirements:     requirements,
		Design:           design,
		Aviation:         aviation,
	}, nil
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049, REQ-EMG-051
func aggregateComposition(input InputComposition) CompositionModel {
	out := CompositionModel{Allocations: input.Allocations, Satisfactions: input.Satisfactions}
	for _, subsystem := range input.Subsystems {
		out.Subsystems = append(out.Subsystems, Subsystem{
			ID: subsystem.ID, Name: subsystem.Name, Dependency: subsystem.Dependency,
			Publication: subsystem.Publication, Description: subsystem.Description,
		})
	}
	return out
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046
func LoadRequirements(path string) (RequirementsDocument, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return RequirementsDocument{}, fmt.Errorf("resolve requirements path: %w", err)
	}
	var requirements RequirementsDocument
	if err := decodeYAMLFile(absPath, &requirements); err != nil {
		return RequirementsDocument{}, fmt.Errorf("decode requirements file: %w", err)
	}
	requirements.SchemaVersion, err = EffectiveSchemaVersion(requirements.SchemaVersion)
	if err != nil {
		return RequirementsDocument{}, fmt.Errorf("requirements schema: %w", err)
	}
	return requirements, nil
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046
func LoadDesign(path string) (DesignDocument, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return DesignDocument{}, fmt.Errorf("resolve design path: %w", err)
	}
	var views ViewsDocument
	if err := decodeYAMLFile(absPath, &views); err != nil {
		return DesignDocument{}, fmt.Errorf("decode views file: %w", err)
	}
	views.SchemaVersion, err = EffectiveSchemaVersion(views.SchemaVersion)
	if err != nil {
		return DesignDocument{}, fmt.Errorf("views schema: %w", err)
	}
	return DesignDocument{SchemaVersion: views.SchemaVersion, Design: views.Design}, nil
}

// TRLC-LINKS: REQ-EMG-001
func decodeYAMLFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := validateCanonicalYAML(path, b, out); err != nil {
		return fmt.Errorf("validate %s: %w", path, err)
	}
	dec := yaml.NewDecoder(bytes.NewReader(b))
	dec.KnownFields(true)
	if err := dec.Decode(out); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	if err := ensureSingleYAMLDocument(dec); err != nil {
		return fmt.Errorf("decode %s: %w", path, err)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-001
func ensureSingleYAMLDocument(dec *yaml.Decoder) error {
	var extra any
	err := dec.Decode(&extra)
	if err == io.EOF {
		return nil
	}
	if err != nil {
		return err
	}
	return fmt.Errorf("multiple YAML documents are not supported")
}
