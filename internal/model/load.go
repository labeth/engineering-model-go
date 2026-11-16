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

// CurrentSchemaVersion is the model schema version this release reads and
// writes: the one place Go names the number (the CUE constant #SchemaVersion in
// schema/common.cue must equal it). A breaking release raises it and
// engws release-check reports release.schema_version_not_bumped when it does
// not (ADR-EMG-077, REQ-EMG-186).
const CurrentSchemaVersion = 3

// SchemaVersionLine is the top-level manifest or document line that declares
// the current schema version, for code that writes a document.
var SchemaVersionLine = fmt.Sprintf("schemaVersion: %d", CurrentSchemaVersion)

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

// SchemaVersionRemedy is the fix for a document whose schemaVersion is
// missing or not the current one (ADR-EMG-061). It names only what exists
// today: the manual edit. The upgrade command of engineering-flow is
// planned and not built yet, so no text may tell a user to run it.
var SchemaVersionRemedy = "set " + SchemaVersionLine + " at the top of the file (edit the file; an upgrade command is planned in engineering-flow and does not exist yet)"

// SchemaVersionError reports a schemaVersion that is missing (0) or not the
// current one. Problem is the message without the remedy; Error appends it.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
// ENGMODEL-LINKS: FU-MODEL-LOADER
type SchemaVersionError struct {
	Version int
}

// Problem returns what is wrong, without the remedy.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func (e *SchemaVersionError) Problem() string {
	return fmt.Sprintf("unsupported schemaVersion %d; supported version is %d", e.Version, CurrentSchemaVersion)
}

// Error returns the problem and the fix.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func (e *SchemaVersionError) Error() string {
	return e.Problem() + "; fix: " + SchemaVersionRemedy
}

// EffectiveSchemaVersion returns the version when it is the current schema
// version and a *SchemaVersionError (naming its remedy) otherwise.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func EffectiveSchemaVersion(version int) (int, error) {
	if version != CurrentSchemaVersion {
		return 0, &SchemaVersionError{Version: version}
	}
	return version, nil
}

// ENGMODEL-LINKS: FU-MODEL-LOADER, DO-ARCHITECTURE-MODEL, IF-GO-PKG-MODEL, FLOW-ENGINE-INTEGRATIONS
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046, REQ-EMG-049, REQ-EMG-051, REQ-EMG-053, REQ-EMG-054, REQ-EMG-094, REQ-EMG-132
func LoadBundle(manifestPath string) (Bundle, error) {
	manifestPath, manifest, err := loadManifest(manifestPath)
	if err != nil {
		return Bundle{}, err
	}

	bundle := Bundle{ManifestPath: manifestPath, Manifest: manifest}
	loader := documentLoader{baseDir: filepath.Dir(manifestPath), refs: manifest.Documents}
	architectureInput, err := loader.loadCore(&bundle)
	if err != nil {
		return Bundle{}, err
	}
	if err := loader.loadAviation(&bundle, architectureInput); err != nil {
		return Bundle{}, err
	}
	if err := loader.loadReqLinks(&bundle); err != nil {
		return Bundle{}, err
	}
	bundle.Architecture = assembleArchitecture(manifest, architectureInput, bundle)
	bundle.Design = DesignDocument{SchemaVersion: CurrentSchemaVersion, Design: bundle.Views.Design}
	bundle.DesignPath = bundle.ViewsPath
	return bundle, nil
}

// loadManifest resolves the manifest path and decodes the manifest at the
// current schema version.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046
func loadManifest(manifestPath string) (string, ManifestDocument, error) {
	if filepath.Base(filepath.Clean(manifestPath)) != "engmod.yml" {
		return "", ManifestDocument{}, fmt.Errorf("engmod.yml is the only supported model entry point")
	}
	manifestPath, err := filepath.Abs(manifestPath)
	if err != nil {
		return "", ManifestDocument{}, fmt.Errorf("resolve manifest path: %w", err)
	}

	var manifest ManifestDocument
	if err := decodeYAMLFile(manifestPath, &manifest); err != nil {
		return "", ManifestDocument{}, fmt.Errorf("decode manifest file: %w", err)
	}
	manifest.SchemaVersion, err = EffectiveSchemaVersion(manifest.SchemaVersion)
	if err != nil {
		return "", ManifestDocument{}, fmt.Errorf("manifest schema: %w", err)
	}
	return manifestPath, manifest, nil
}

// documentLoader decodes the documents a manifest references, relative to the
// manifest directory.
type documentLoader struct {
	baseDir string
	refs    DocumentReferences
}

// documentSlot is one referenced document: its manifest path, the label used
// in a decode error, where it decodes to and where its resolved path goes.
type documentSlot struct {
	ref  string
	kind string
	out  any
	path *string
}

// decode resolves and decodes one referenced document and returns its path.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046, REQ-EMG-053
func (l documentLoader) decode(slot documentSlot) error {
	path := filepath.Join(l.baseDir, slot.ref)
	*slot.path = path
	if err := decodeYAMLFile(path, slot.out); err != nil {
		return fmt.Errorf("decode %s file: %w", slot.kind, err)
	}
	return nil
}

// loadCore decodes the required documents in manifest order into the bundle
// and returns the architecture input, which the bundle keeps only in
// assembled form.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044, REQ-EMG-046, REQ-EMG-053
func (l documentLoader) loadCore(b *Bundle) (ArchitectureInputDocument, error) {
	var architectureInput ArchitectureInputDocument
	slots := []documentSlot{
		{l.refs.Catalog, "catalog", &b.Catalog, &b.CatalogPath},
		{l.refs.Decisions, "decisions", &b.Decisions, &b.DecisionsPath},
		{l.refs.Requirements, "requirements", &b.Requirements, &b.RequirementsPath},
		{l.refs.Architecture, "architecture", &architectureInput, &b.ArchitecturePath},
		{l.refs.Behavior, "behavior", &b.Behavior, &b.BehaviorPath},
		{l.refs.Assurance, "assurance", &b.Assurance, &b.AssurancePath},
		{l.refs.Compliance, "compliance", &b.Compliance, &b.CompliancePath},
		{l.refs.Views, "views", &b.Views, &b.ViewsPath},
	}
	for _, slot := range slots {
		if err := l.decode(slot); err != nil {
			return ArchitectureInputDocument{}, err
		}
	}
	return architectureInput, nil
}

// loadAviation decodes and validates the optional aviation document.
// TRLC-LINKS: REQ-EMG-053, REQ-EMG-054, REQ-EMG-094
func (l documentLoader) loadAviation(b *Bundle, architectureInput ArchitectureInputDocument) error {
	if l.refs.Aviation == "" {
		return nil
	}
	var document AviationDocument
	if err := l.decode(documentSlot{l.refs.Aviation, "aviation", &document, &b.AviationPath}); err != nil {
		return err
	}
	diagnostics := ValidateAviation(document, b.Requirements, architectureInput)
	if b.Manifest.Lifecycle != nil {
		// With lifecycle the state and completeness codes are tier keyed
		// (validate reports them); structural codes stay fatal.
		diagnostics, _ = SplitAviationDiagnostics(diagnostics)
	}
	if len(diagnostics) > 0 {
		return &AviationValidationError{Diagnostics: diagnostics}
	}
	b.Aviation = &document
	return nil
}

// loadReqLinks decodes the optional requirement link document.
// TRLC-LINKS: REQ-EMG-132
func (l documentLoader) loadReqLinks(b *Bundle) error {
	if l.refs.ReqLinks == "" {
		return nil
	}
	return l.decode(documentSlot{l.refs.ReqLinks, "reqLinks", &b.ReqLinks, &b.ReqLinksPath})
}

// authoredFromDocuments gathers the authored architecture from the
// architecture, behavior and assurance documents.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func authoredFromDocuments(input ArchitectureInput, behavior BehaviorDocument, assurance AssuranceDocument) AuthoredArchitecture {
	return AuthoredArchitecture{
		FunctionalGroups: input.FunctionalGroups, FunctionalUnits: input.FunctionalUnits,
		Actors: input.Actors, ReferencedElements: input.ReferencedElements,
		Interfaces: input.Interfaces, DataObjects: input.DataObjects,
		DeploymentTargets: input.DeploymentTargets, HardwareItems: input.HardwareItems,
		HardwareInterfaces: input.HardwareInterfaces,
		Events:             behavior.Behavior.Events,
		Flows:              behavior.Behavior.Flows, Mappings: behavior.Behavior.Relationships,
		Machines:      behavior.Behavior.Machines,
		AttackVectors: assurance.Assurance.AttackVectors, Controls: assurance.Assurance.Controls,
		Risks: assurance.Assurance.Risks, POAMItems: assurance.Assurance.POAMItems,
		TrustBoundaries:      assurance.Assurance.TrustBoundaries,
		ThreatScenarios:      assurance.Assurance.ThreatScenarios,
		ThreatAssumptions:    assurance.Assurance.ThreatAssumptions,
		ThreatOutOfScope:     assurance.Assurance.ThreatOutOfScope,
		ThreatMitigations:    assurance.Assurance.ThreatMitigations,
		ControlVerifications: assurance.Assurance.ControlVerifications,
	}
}

// assembleArchitecture builds the architecture document from the manifest,
// the architecture input and the other decoded documents of the bundle.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func assembleArchitecture(manifest ManifestDocument, architectureInput ArchitectureInputDocument, b Bundle) ArchitectureDocument {
	input := architectureInput.Architecture
	return ArchitectureDocument{
		SchemaVersion: CurrentSchemaVersion,
		Model: ModelMeta{
			ID: manifest.Module.ModelID, Title: manifest.Module.Title,
			Introduction: manifest.Module.Introduction, Documents: manifest.Documents,
		},
		Decisions: b.Decisions.Decisions, AuthoredArchitecture: authoredFromDocuments(input, b.Behavior, b.Assurance),
		Semantics:  input.Semantics,
		Compliance: b.Compliance.Compliance, Contract: input.Contract,
		Composition: aggregateComposition(input.Composition), InferenceHints: manifest.InferenceHints,
		NAF: b.Views.NAF, Views: b.Views.Views,
	}
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

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-139
func decodeYAMLFile(path string, out any) error {
	b, err := os.ReadFile(path)
	if err != nil {
		return fmt.Errorf("read %s: %w", path, err)
	}
	if err := checkSchemaVersion(path, b, out); err != nil {
		return err
	}
	if err := rejectLegacyStateVocabulary(path, b, out); err != nil {
		return err
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

// checkSchemaVersion rejects a document whose top-level schemaVersion is
// absent, 0, 1, 3 or anything but the current version, with a
// *SchemaVersionError that carries the fix. It runs before the CUE schema
// check: the CUE schema value is the constant 2, so it would fill in a missing
// version and word a wrong one as an unreadable unification failure. A
// document that does not parse, or whose schemaVersion is not a number, is
// left to the later checks, which report it.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
// ENGMODEL-LINKS: FU-MODEL-LOADER
func checkSchemaVersion(path string, source []byte, out any) error {
	if _, err := cueSchemaName(out); err != nil {
		return nil
	}
	var head struct {
		SchemaVersion int `yaml:"schemaVersion"`
	}
	if err := yaml.Unmarshal(source, &head); err != nil {
		return nil
	}
	if _, err := EffectiveSchemaVersion(head.SchemaVersion); err != nil {
		return fmt.Errorf("%s: %w", path, err)
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
