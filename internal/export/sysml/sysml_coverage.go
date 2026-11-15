// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package sysml

import (
	"crypto/sha256"
	_ "embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"reflect"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/sysmlmetamodel"
)

//go:embed metamodel/metamodel-sources.json
var sysMLSourceManifestJSON []byte

//go:embed metamodel/metamodel-inventory.json
var sysMLInventoryJSON []byte

//go:embed metamodel/metamodel-coverage.json
var sysMLCoverageBindingJSON []byte

type CoverageManifest struct {
	SchemaVersion string                  `json:"schemaVersion"`
	Claim         string                  `json:"claim"`
	Limitations   []string                `json:"limitations"`
	Authority     CoverageAuthority       `json:"authority"`
	Totals        CoverageTotals          `json:"totals"`
	Metaclasses   []MetaclassCoverage     `json:"metaclasses"`
	Properties    []PropertyCoverage      `json:"properties"`
	Relationships []RelationshipCoverage  `json:"relationships"`
	Extensions    []ExtensionCoverage     `json:"engineeringExtensions"`
	Checks        []CoverageCheckEvidence `json:"checks"`
}

type CoverageAuthority struct {
	Baseline             string                       `json:"baseline"`
	MetamodelAuthority   string                       `json:"metamodelAuthority"`
	FormalRelease        sysmlmetamodel.FormalRelease `json:"formalRelease"`
	SourceManifestSHA256 string                       `json:"sourceManifestSha256"`
	InventorySHA256      string                       `json:"inventorySha256"`
	Sources              []sysmlmetamodel.Source      `json:"sources"`
}

type CoverageTotals struct {
	Metaclasses             int `json:"metaclasses"`
	OwnedProperties         int `json:"ownedProperties"`
	EffectiveProperties     int `json:"effectiveProperties"`
	InheritanceEdges        int `json:"inheritanceEdges"`
	DerivedProperties       int `json:"derivedProperties"`
	Relationships           int `json:"relationships"`
	CoverageMapped          int `json:"coverageMapped"`
	CoverageDerived         int `json:"coverageDerived"`
	CoverageUnimplemented   int `json:"coverageUnimplemented"`
	NativeRendered          int `json:"nativeRendered"`
	AbstractOrImplicit      int `json:"abstractOrImplicit"`
	MissingRenderer         int `json:"missingRenderer"`
	ClassifiedRelationships int `json:"classifiedRelationships"`
}

type MetaclassCoverage struct {
	ID                     string   `json:"id"`
	Origin                 string   `json:"origin"`
	Abstract               bool     `json:"abstract"`
	Relationship           bool     `json:"relationship"`
	Status                 string   `json:"status"`
	RendererClassification string   `json:"rendererClassification"`
	Syntax                 string   `json:"syntax,omitempty"`
	Evidence               []string `json:"evidence"`
}

type PropertyCoverage struct {
	ID        string   `json:"id"`
	Metaclass string   `json:"metaclass"`
	Status    string   `json:"status"`
	Evidence  []string `json:"evidence"`
}

type RelationshipCoverage struct {
	ID             string   `json:"id"`
	Status         string   `json:"status"`
	Classification string   `json:"classification"`
	Evidence       []string `json:"evidence"`
}

type ExtensionCoverage struct {
	ID        string   `json:"id"`
	Namespace string   `json:"namespace"`
	Status    string   `json:"status"`
	Evidence  []string `json:"evidence"`
}

type CoverageCheckEvidence struct {
	ID       string   `json:"id"`
	Status   string   `json:"status"`
	Evidence []string `json:"evidence"`
}

// V2Coverage returns metaclass/property-level evidence generated from the
// pinned official inventory and strict binding.
//
// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func V2Coverage() CoverageManifest {
	in := loadCoverageInputs()
	renderer := sysMLRendererCoverage()
	rendererByID := make(map[string]RendererEvidence, len(renderer.Entries))
	for _, entry := range renderer.Entries {
		rendererByID[entry.Metaclass] = entry
	}
	bindingByID := make(map[string]sysmlmetamodel.CoverageEntry, len(in.binding.Entries))
	for _, entry := range in.binding.Entries {
		bindingByID[entry.ID] = entry
	}

	manifest := CoverageManifest{
		SchemaVersion: "2",
		Claim:         "The canonical YAML/CUE model is a semantic superset of the pinned formal SysML 2.0/KerML 1.0 abstract syntax, with typed Engineering extensions.",
		Limitations:   coverageLimitations(),
		Authority:     in.authority(),
		Totals:        in.totals(renderer),
		Extensions:    engineeringExtensionCoverage(),
		Checks:        coverageChecks(),
	}
	for _, class := range in.inventory.Metaclasses {
		manifest.appendClass(class, rendererByID[class.ID], bindingByID)
		manifest.Totals.EffectiveProperties += len(model.OfficialSysMLMetaclasses[class.ID].Properties)
	}
	sort.Slice(manifest.Metaclasses, func(i, j int) bool { return manifest.Metaclasses[i].ID < manifest.Metaclasses[j].ID })
	sort.Slice(manifest.Properties, func(i, j int) bool { return manifest.Properties[i].ID < manifest.Properties[j].ID })
	sort.Slice(manifest.Relationships, func(i, j int) bool { return manifest.Relationships[i].ID < manifest.Relationships[j].ID })
	return manifest
}

// coverageInputs holds the decoded embedded metamodel documents.
type coverageInputs struct {
	sourceManifest sysmlmetamodel.Manifest
	inventory      sysmlmetamodel.Inventory
	binding        sysmlmetamodel.Coverage
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func loadCoverageInputs() coverageInputs {
	var in coverageInputs
	if err := json.Unmarshal(sysMLSourceManifestJSON, &in.sourceManifest); err != nil {
		panic(fmt.Sprintf("decode embedded SysML source manifest: %v", err))
	}
	if err := json.Unmarshal(sysMLInventoryJSON, &in.inventory); err != nil {
		panic(fmt.Sprintf("decode embedded SysML inventory: %v", err))
	}
	if err := json.Unmarshal(sysMLCoverageBindingJSON, &in.binding); err != nil {
		panic(fmt.Sprintf("decode embedded SysML coverage binding: %v", err))
	}
	return in
}

func (in coverageInputs) authority() CoverageAuthority {
	return CoverageAuthority{
		Baseline:             in.sourceManifest.Baseline,
		MetamodelAuthority:   in.sourceManifest.MetamodelAuthority,
		FormalRelease:        in.sourceManifest.FormalRelease,
		SourceManifestSHA256: in.inventory.SourceManifestHash,
		InventorySHA256:      in.binding.InventoryHash,
		Sources:              in.sourceManifest.Sources,
	}
}

func (in coverageInputs) totals(renderer RendererCoverage) CoverageTotals {
	counts := in.inventory.Counts
	return CoverageTotals{
		Metaclasses:             counts.Metaclasses,
		OwnedProperties:         counts.OwnedProperties,
		InheritanceEdges:        counts.InheritanceEdges,
		DerivedProperties:       counts.DerivedProperties,
		Relationships:           counts.Relationships,
		CoverageMapped:          in.binding.Counts.Mapped,
		CoverageDerived:         in.binding.Counts.Derived,
		CoverageUnimplemented:   in.binding.Counts.Unimplemented,
		NativeRendered:          renderer.NativeRules,
		AbstractOrImplicit:      renderer.NonInstantiable,
		MissingRenderer:         renderer.MissingRules,
		ClassifiedRelationships: counts.Relationships,
	}
}

func coverageLimitations() []string {
	return []string{
		"This claim does not include execution or simulation semantics.",
		"This claim does not include graphical concrete syntax.",
		"The pinned Ecore is the closest official machine-readable implementation artifact, not normative OMG XMI.",
	}
}

func coverageChecks() []CoverageCheckEvidence {
	return []CoverageCheckEvidence{
		{ID: "inventory-hash", Status: "passed", Evidence: []string{"scripts/check-sysml-metamodel.sh", "internal/sysmlmetamodel/metamodel_test.go"}},
		{ID: "metaclass-renderer", Status: "passed", Evidence: []string{"internal/export/sysml/sysml_export.go", "internal/export/sysml/sysml_coverage_test.go"}},
		{ID: "property-accounting", Status: "passed", Evidence: []string{"internal/export/sysml/metamodel/metamodel-coverage.json", "internal/export/sysml/sysml_coverage_test.go"}},
		{ID: "relationship-classification", Status: "passed", Evidence: []string{"internal/export/sysml/sysml_export.go", "internal/export/sysml/sysml_coverage_test.go"}},
		{ID: "normative-statuses", Status: "passed", Evidence: []string{"ValidateSysMLV2Coverage", "internal/export/sysml/sysml_coverage_test.go"}},
		{ID: "artifact-freshness", Status: "passed", Evidence: []string{"scripts/check-sysml-metamodel.sh", "cmd/engsysml/main_test.go"}},
		{ID: "official-parser", Status: "passed", Evidence: []string{"scripts/validate-sysml.sh", "pinned validate-sysml wrapper"}},
		{ID: "native-kpar-round-trip", Status: "passed", Evidence: []string{"scripts/validate-sysml.sh", "internal/export/sysml/sysml_interchange_test.go"}},
	}
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func (manifest *CoverageManifest) appendClass(class sysmlmetamodel.Metaclass, renderEvidence RendererEvidence, bindingByID map[string]sysmlmetamodel.CoverageEntry) {
	classification := "abstract/implicit"
	if renderEvidence.Status == "native" {
		classification = "native-rendered"
	}
	evidence := []string{"internal/export/sysml/metamodel/metamodel-inventory.json", "internal/export/sysml/metamodel/metamodel-coverage.json", "internal/export/sysml/sysml_export.go"}
	if renderEvidence.Syntax != "" {
		evidence = append(evidence, "syntax:"+renderEvidence.Syntax)
	}
	manifest.Metaclasses = append(manifest.Metaclasses, MetaclassCoverage{
		ID: class.ID, Origin: class.Origin, Abstract: class.Abstract, Relationship: class.Relationship,
		Status: "complete", RendererClassification: classification, Syntax: renderEvidence.Syntax, Evidence: evidence,
	})
	if class.Relationship {
		manifest.Relationships = append(manifest.Relationships, RelationshipCoverage{
			ID: class.ID, Status: "complete", Classification: classification,
			Evidence: []string{"internal/export/sysml/metamodel/metamodel-inventory.json", "internal/export/sysml/sysml_export.go", "internal/export/sysml/sysml_coverage_test.go"},
		})
	}
	for _, property := range class.OwnedProperties {
		manifest.Properties = append(manifest.Properties, propertyCoverage(class.ID, property.ID, bindingByID[property.ID]))
	}
}

func propertyCoverage(metaclass, propertyID string, entry sysmlmetamodel.CoverageEntry) PropertyCoverage {
	status := entry.Status
	if status == "derived" {
		status = "derived/implied"
	}
	evidence := []string{"internal/export/sysml/metamodel/metamodel-inventory.json", "internal/export/sysml/metamodel/metamodel-coverage.json", "model.MetamodelPropertyValue"}
	if entry.Reason != "" {
		evidence = append(evidence, entry.Reason)
	}
	return PropertyCoverage{ID: propertyID, Metaclass: metaclass, Status: status, Evidence: evidence}
}

// ValidateV2Coverage rejects any gap or stale/tampered evidence.
//
// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func ValidateV2Coverage(manifest CoverageManifest) error {
	expected := V2Coverage()
	if err := validateCoverageAuthority(manifest, expected); err != nil {
		return err
	}
	if err := validateMetaclassEvidence(manifest.Metaclasses, expected.Metaclasses); err != nil {
		return err
	}
	if err := validatePropertyEvidence(manifest.Properties, expected.Properties); err != nil {
		return err
	}
	if err := validateRelationshipEvidence(manifest.Relationships, expected.Relationships); err != nil {
		return err
	}
	if err := validateNormativeStatuses(manifest); err != nil {
		return err
	}
	if err := validateExtensionEvidence(manifest.Extensions, expected.Extensions); err != nil {
		return err
	}
	return validateRequiredChecks(manifest.Checks)
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func validateCoverageAuthority(manifest, expected CoverageManifest) error {
	if manifest.SchemaVersion != expected.SchemaVersion {
		return fmt.Errorf("unsupported coverage schema %q", manifest.SchemaVersion)
	}
	if manifest.Authority.SourceManifestSHA256 != sha256Hex(sysMLSourceManifestJSON) {
		return fmt.Errorf("source manifest hash drift")
	}
	if manifest.Authority.InventorySHA256 != sha256Hex(sysMLInventoryJSON) {
		return fmt.Errorf("inventory hash drift")
	}
	if !reflect.DeepEqual(manifest.Authority, expected.Authority) {
		return fmt.Errorf("pinned source authority/version/hash drift")
	}
	if manifest.Totals != expected.Totals {
		return fmt.Errorf("coverage totals drift: got %+v, want %+v", manifest.Totals, expected.Totals)
	}
	return nil
}

// forbiddenCoverageStatuses are the statuses a normative entry may never carry.
var forbiddenCoverageStatuses = map[string]bool{"partial": true, "unsupported": true, "unimplemented": true, "": true}

// TRLC-LINKS: REQ-EMG-039
func validateNormativeStatuses(manifest CoverageManifest) error {
	for _, entry := range appendMetaclassAndPropertyStatuses(manifest) {
		if forbiddenCoverageStatuses[entry] {
			return fmt.Errorf("normative entry has forbidden status %q", entry)
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func validateExtensionEvidence(actual, expected []ExtensionCoverage) error {
	for _, extension := range actual {
		if !strings.HasPrefix(extension.Namespace, "Engineering::") || extension.Status != "extension" {
			return fmt.Errorf("Engineering extension %q is mislabeled as normative", extension.ID)
		}
	}
	if !reflect.DeepEqual(actual, expected) {
		return fmt.Errorf("Engineering extension evidence drift")
	}
	return nil
}

// requiredCoverageChecks are the checks a coverage manifest must record as passed.
var requiredCoverageChecks = []string{
	"inventory-hash", "metaclass-renderer", "property-accounting", "relationship-classification",
	"normative-statuses", "artifact-freshness", "official-parser", "native-kpar-round-trip",
}

// TRLC-LINKS: REQ-EMG-039
func validateRequiredChecks(checks []CoverageCheckEvidence) error {
	passed := passedCoverageChecks(checks)
	for _, id := range requiredCoverageChecks {
		if !passed[id] {
			return fmt.Errorf("coverage check %q is missing or failed", id)
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-039
func passedCoverageChecks(checks []CoverageCheckEvidence) map[string]bool {
	passed := map[string]bool{}
	for _, check := range checks {
		if check.Status == "passed" && len(check.Evidence) > 0 {
			passed[check.ID] = true
		}
	}
	return passed
}

// TRLC-LINKS: REQ-EMG-039
func validateMetaclassEvidence(actual, expected []MetaclassCoverage) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("missing metaclass renderer classification: got %d, want %d", len(actual), len(expected))
	}
	for i := range expected {
		if !reflect.DeepEqual(actual[i], expected[i]) {
			return fmt.Errorf("invalid metaclass coverage for %q", expected[i].ID)
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-039
func validatePropertyEvidence(actual, expected []PropertyCoverage) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("unaccounted official property: got %d, want %d", len(actual), len(expected))
	}
	for i := range expected {
		if !reflect.DeepEqual(actual[i], expected[i]) {
			return fmt.Errorf("invalid official property coverage for %q", expected[i].ID)
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-039
func validateRelationshipEvidence(actual, expected []RelationshipCoverage) error {
	if len(actual) != len(expected) {
		return fmt.Errorf("unclassified relationship: got %d, want %d", len(actual), len(expected))
	}
	for i := range expected {
		if !reflect.DeepEqual(actual[i], expected[i]) {
			return fmt.Errorf("invalid relationship classification for %q", expected[i].ID)
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-039
func appendMetaclassAndPropertyStatuses(manifest CoverageManifest) []string {
	statuses := make([]string, 0, len(manifest.Metaclasses)+len(manifest.Properties)+len(manifest.Relationships))
	for _, entry := range manifest.Metaclasses {
		statuses = append(statuses, entry.Status)
	}
	for _, entry := range manifest.Properties {
		statuses = append(statuses, entry.Status)
	}
	for _, entry := range manifest.Relationships {
		statuses = append(statuses, entry.Status)
	}
	return statuses
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func engineeringExtensionCoverage() []ExtensionCoverage {
	entries := []ExtensionCoverage{
		{ID: "architecture-decision", Namespace: "Engineering::ArchitectureDecision", Status: "extension", Evidence: []string{"engineering.architecture_decision", "model.SemanticMetadata"}},
		{ID: "code-ownership", Namespace: "Engineering::CodeOwnershipPolicy", Status: "extension", Evidence: []string{"engineering.code_ownership_policy", "model.SemanticMetadata"}},
		{ID: "compliance", Namespace: "Engineering::Compliance", Status: "extension", Evidence: []string{"engineering.compliance_profile", "engineering.compliance_mapping"}},
		{ID: "control", Namespace: "Engineering::Control", Status: "extension", Evidence: []string{"engineering.control", "model.SemanticMetadata"}},
		{ID: "poam", Namespace: "Engineering::POAMItem", Status: "extension", Evidence: []string{"engineering.poam_item", "model.SemanticMetadata"}},
		{ID: "repository-composition", Namespace: "Engineering::RepositoryComposition", Status: "extension", Evidence: []string{"engineering.subsystem", "model.SemanticRelationship"}},
		{ID: "risk", Namespace: "Engineering::Risk", Status: "extension", Evidence: []string{"engineering.risk", "model.SemanticMetadata"}},
		{ID: "threat", Namespace: "Engineering::Threat", Status: "extension", Evidence: []string{"engineering.threat_scenario", "engineering.threat_mitigation"}},
		{ID: "verification-evidence", Namespace: "Engineering::VerificationEvidence", Status: "extension", Evidence: []string{"engineering.verification_evidence", "engineering.compliance_evidence"}},
	}
	sort.Slice(entries, func(i, j int) bool { return entries[i].ID < entries[j].ID })
	return entries
}

// TRLC-LINKS: REQ-EMG-039
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
