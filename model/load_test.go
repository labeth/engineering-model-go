// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

const validManifest = `schemaVersion: 2
module:
  path: example.com/test@v1
  version: v1.0.0
  modelId: TEST-MODEL
  title: Test model
  introduction: Test introduction
  kind: system
documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
dependencies: []
publications: []
inferenceHints: {}
`

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-051
func TestLoadBundleAggregatesSchemaV2Domains(t *testing.T) {
	dir := writeV2Fixture(t, validManifest)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Architecture.Model.ID != "TEST-MODEL" ||
		len(bundle.Architecture.AuthoredArchitecture.FunctionalUnits) != 1 ||
		len(bundle.Architecture.AuthoredArchitecture.States) != 1 ||
		len(bundle.Architecture.AuthoredArchitecture.Controls) != 1 ||
		len(bundle.Architecture.Compliance.Profiles) != 1 {
		t.Fatalf("domain documents were not aggregated: %+v", bundle.Architecture)
	}
	if got := bundle.Architecture.AuthoredArchitecture.Mappings; len(got) != 1 || got[0].Type != "writes" {
		t.Fatalf("relationships were not aggregated as mappings: %+v", got)
	}
	if got := bundle.Design.Design.FunctionalUnits; len(got) != 1 || got[0].Views["intent"].Narrative != "Narrative" {
		t.Fatalf("view narratives were not aggregated into design: %+v", got)
	}
	if got := bundle.Requirements.Requirements; len(got) != 1 ||
		got[0].Title != "Qualified requirement" ||
		got[0].VerificationMethods[0] != "test" ||
		!got[0].Derived {
		t.Fatalf("requirement qualification metadata was not loaded: %+v", got)
	}
	if got := bundle.Views.Documents; len(got) != 1 ||
		got[0].Control.Identifier != "DOC-001" ||
		got[0].ContentRefs[0] != "REQ-A" {
		t.Fatalf("document definitions were not loaded: %+v", got)
	}
	if _, err := LoadCanonicalBundle(filepath.Join(dir, "engmod.yml")); err != nil {
		t.Fatalf("load canonical bundle from manifest: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func TestLoadBundleRejectsArchitectureEntryPoint(t *testing.T) {
	dir := writeV2Fixture(t, validManifest)
	_, err := LoadBundle(filepath.Join(dir, "model", "architecture.yml"))
	if err == nil || !strings.Contains(err.Error(), "engmod.yml") {
		t.Fatalf("expected manifest-only entry rejection, got %v", err)
	}
	_, err = LoadCanonicalBundle(filepath.Join(dir, "model", "architecture.yml"))
	if err == nil || !strings.Contains(err.Error(), "engmod.yml") {
		t.Fatalf("expected canonical manifest-only entry rejection, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func TestSchemaV2RejectsMissingSchemaVersion(t *testing.T) {
	dir := writeV2Fixture(t, strings.TrimPrefix(validManifest, "schemaVersion: 2\n"))
	_, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err == nil || !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("expected required schemaVersion error, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func TestManifestRequiresEveryExplicitDomainDocument(t *testing.T) {
	fields := []string{"catalog", "requirements", "architecture", "behavior", "assurance", "compliance", "views", "decisions"}
	for _, field := range fields {
		t.Run(field, func(t *testing.T) {
			line := "  " + field + ": model/" + field + ".yml\n"
			dir := writeV2Fixture(t, strings.Replace(validManifest, line, "", 1))
			_, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
			if err == nil || !strings.Contains(err.Error(), "documents."+field) {
				t.Fatalf("expected missing %s path error, got %v", field, err)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func TestLoadBundleUsesOnlyExplicitDocumentPaths(t *testing.T) {
	manifest := strings.Replace(validManifest, "catalog: model/catalog.yml", "catalog: model/vocabulary.yml", 1)
	dir := writeV2Fixture(t, manifest)
	if err := os.Rename(filepath.Join(dir, "model", "catalog.yml"), filepath.Join(dir, "model", "vocabulary.yml")); err != nil {
		t.Fatal(err)
	}
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if filepath.Base(bundle.CatalogPath) != "vocabulary.yml" {
		t.Fatalf("explicit catalog path was not used: %s", bundle.CatalogPath)
	}
}

// TRLC-LINKS: REQ-EMG-046
func TestSchemaV2RejectsUnknownFieldsAndRelationshipKinds(t *testing.T) {
	t.Run("unknown field", func(t *testing.T) {
		dir := writeV2Fixture(t, validManifest)
		path := filepath.Join(dir, "model", "behavior.yml")
		source := readTestFile(t, path) + "unexpected: true\n"
		writeTestFile(t, path, source)
		_, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
		if err == nil || !strings.Contains(err.Error(), "unexpected") {
			t.Fatalf("expected strict unknown-field error, got %v", err)
		}
	})
	t.Run("unsupported relationship", func(t *testing.T) {
		dir := writeV2Fixture(t, validManifest)
		path := filepath.Join(dir, "model", "behavior.yml")
		source := strings.Replace(readTestFile(t, path), "type: writes", "type: arbitrary", 1)
		writeTestFile(t, path, source)
		_, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
		if err == nil || !strings.Contains(err.Error(), "relationships.0.type") {
			t.Fatalf("expected typed relationship error, got %v", err)
		}
	})
}

// TRLC-LINKS: REQ-EMG-045
// ENGMODEL-LINKS: DO-MODEL-AUTHORING-CONTRACT
func TestBuildAuthoringContractDescribesSchemaV2(t *testing.T) {
	dir := writeV2Fixture(t, validManifest)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}

	contract := BuildAuthoringContract(bundle)
	if contract.ContractVersion != AuthoringContractVersion || len(contract.Documents) != 9 || len(contract.Compatibility) != 0 {
		t.Fatalf("unexpected schema-v2 authoring contract: %+v", contract)
	}

	for _, document := range contract.Documents {
		if !document.Required || document.SchemaVersion != 2 || document.Path == "" || document.SchemaPath == "" {
			t.Fatalf("incomplete document contract: %+v", document)
		}
	}
}

// TRLC-LINKS: REQ-EMG-053, REQ-EMG-056
func TestLoadBundleIncludesOptionalAviationDocument(t *testing.T) {
	bundle, err := LoadBundle(filepath.Join("..", "examples", "dal-c-flight-control-sample", "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Aviation == nil || bundle.Aviation.Aviation.Profile.SoftwareLevel != "C" {
		t.Fatalf("aviation document was not loaded: %+v", bundle.Aviation)
	}
	contract := BuildAuthoringContract(bundle)
	if len(contract.Documents) != 10 {
		t.Fatalf("expected optional aviation authoring document, got %+v", contract.Documents)
	}
	last := contract.Documents[len(contract.Documents)-1]
	if last.Kind != "aviation" || last.Required || last.SchemaPath != "model/schema/aviation.cue" {
		t.Fatalf("unexpected aviation authoring contract: %+v", last)
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-051
func writeV2Fixture(t *testing.T, manifest string) string {
	t.Helper()
	dir, err := os.MkdirTemp(".", ".loader-test-")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.RemoveAll(dir) })
	files := map[string]string{
		"engmod.yml":             manifest,
		"model/catalog.yml":      "schemaVersion: 2\ncatalog: {}\n",
		"model/requirements.yml": "schemaVersion: 2\nlintRun: {}\nrequirements:\n  - id: REQ-A\n    title: Qualified requirement\n    text: The system shall provide an outcome.\n    category: functional\n    rationale: The outcome is required.\n    sourceRefs: [REF-DOC]\n    verificationMethods: [test]\n    verificationCriteria: The observable outcome is produced.\n    priority: high\n    criticality: high\n    status: approved\n    derived: true\n    derivedRationale: The implementation allocation introduces this requirement.\n    tags: [baseline]\n    appliesTo: [FU-A]\n",
		"model/architecture.yml": "schemaVersion: 2\narchitecture:\n  functionalUnits:\n    - id: FU-A\n      name: A\n  actors:\n    - id: ACT-A\n      name: Stakeholder A\n  referencedElements:\n    - id: REF-DOC\n      name: Source document\n      kind: document\n      layer: external\n      description: Source description\n      version: \"1.0\"\n      date: 2026-09-20\n      uri: https://example.com/source\n      publisher: Example publisher\n  dataObjects:\n    - id: DO-A\n      name: A\n",
		"model/behavior.yml":     "schemaVersion: 2\nbehavior:\n  states:\n    - id: STATE-A\n      name: A\n  relationships:\n    - type: writes\n      from: FU-A\n      to: DO-A\n",
		"model/assurance.yml":    "schemaVersion: 2\nassurance:\n  controls:\n    - id: CTRL-A\n      name: A\n",
		"model/compliance.yml":   "schemaVersion: 2\ncompliance:\n  profiles:\n    - id: PROFILE-A\n      href: profile.json\n",
		"model/views.yml":        "schemaVersion: 2\nviews:\n  - id: VIEW-A\n    kind: architecture-intent\n    roots: [FU-A]\nnaf:\n  framework: NAF\n  version: \"4.1\"\n  architectureDescription: Test\n  stakeholders: []\n  concerns: []\n  products: []\ndesign:\n  id: DESIGN-A\n  title: Test\n  functionalUnits:\n    - id: FU-A\n      views:\n        intent:\n          narrative: Narrative\ndocuments:\n  - id: DOC-A\n    title: Formal document\n    kind: system-description\n    purpose: Describe the system.\n    audience: [engineering]\n    stakeholderRefs: [ACT-A]\n    referenceRefs: [REF-DOC]\n    contentRefs: [REQ-A, FU-A, VIEW-A]\n    sections:\n      - id: scope\n        title: Scope\n        narrative: Defines the document scope.\n        includeRefs: [FU-A]\n    control:\n      identifier: DOC-001\n      revision: \"1.0\"\n      status: draft\n      issuedBy: Example issuer\n      issueDate: 2026-09-20\n      language: en\n      documentType: specification\n      confidentiality: internal\n      securityClassification: unclassified\n      exportControlled: false\n      countryOfOrigin: Sweden\n      confidentialityStamp: false\n",
		"model/decisions.yml":    "schemaVersion: 2\ndecisions: []\n",
	}
	for name, content := range files {
		writeTestFile(t, filepath.Join(dir, name), content)
	}
	return dir
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func writeTestFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func readTestFile(t *testing.T, path string) string {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(content)
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestArchitectureYAMLDecodesCanonicalStructuralAndBehavioralSemantics(t *testing.T) {
	input := []byte(`
model:
  id: MODEL-A
semantics:
  imports:
    - namespace: MODEL-A
      imported: Systems
      visibility: private
      recursive: true
  elements:
    - id: PKG-DOMAIN
      kind: package_definition
      namespace: MODEL-A
    - id: PART-A
      kind: part_usage
      namespace: PKG-DOMAIN
      typeRef: PART-DEF
      multiplicity: {lower: 1, upper: 4}
      ordered: true
      unique: false
      specializes: [PART-BASE]
      subsets: [PART-SET]
      redefines: [PART-OLD]
      features:
        - name: command
          kind: port
          type: PORT-COMMAND
          conjugated: true
        - name: request
          kind: parameter
          type: ITEM-REQUEST
          direction: in
    - id: CONTROL-FORK
      kind: control_node
      namespace: PKG-DOMAIN
      controlKind: fork
  relationships:
    - id: TRANSITION-A
      kind: transition
      source: STATE-A
      target: STATE-B
      triggers: [EVENT-A]
      guard: {language: expression, value: ready}
      effect: {language: expression, value: notify}
`)
	var document ArchitectureDocument
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("decode canonical semantics: %v", err)
	}

	if len(document.Semantics.Imports) != 1 || len(document.Semantics.Elements) != 3 || len(document.Semantics.Relationships) != 1 {
		t.Fatalf("unexpected semantic content: %+v", document.Semantics)
	}
	part := document.Semantics.Elements[1]
	if part.Multiplicity == nil || part.Multiplicity.Upper == nil || *part.Multiplicity.Upper != 4 || part.Unique == nil || *part.Unique {
		t.Fatalf("structural modifiers were not decoded: %+v", part)
	}
	transition := document.Semantics.Relationships[0]
	if transition.Kind != RelationshipTransition || len(transition.Triggers) != 1 || transition.Guard == nil || transition.Effect == nil {
		t.Fatalf("behavioral semantics were not decoded: %+v", transition)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestArchitectureYAMLDecodesAdvancedCanonicalSemanticsStrictly(t *testing.T) {
	input := []byte(`
model:
  id: MODEL-A
semantics:
  elements:
    - id: REQ-A
      kind: requirement_definition
    - id: UNIT-M
      kind: unit_definition
    - id: CONSTRAINT-A
      kind: constraint_definition
      features:
        - name: maximum
          kind: attribute
          multiplicity: {lower: 1, upper: 1}
          value:
            kind: literal
            typedValue:
              kind: quantity
              quantity: {value: 12.5, unit: UNIT-M}
    - id: OCC-A
      kind: occurrence_usage
      occurrenceId: flight-42
      variation: true
      variants: [OCC-B]
    - id: OCC-B
      kind: snapshot
      occurrenceId: flight-42-at-t1
      portionOf: OCC-A
    - id: EXT-A
      kind: engineering_extension
      extensionNamespace: engineering
      extension: engineering.risk
      targets: [REQ-A]
      metadata:
        - namespace: engineering
          type: engineering.classification
          target: EXT-A
          properties: {level: high}
  relationships:
    - id: VERIFY-A
      kind: verification
      source: OCC-B
      target: REQ-A
`)
	var document ArchitectureDocument
	decoder := yaml.NewDecoder(bytes.NewReader(input))
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err != nil {
		t.Fatalf("decode advanced semantics: %v", err)
	}

	if got := document.Semantics.Elements[2].Features[0].Value.TypedValue.Quantity.Unit; got != "UNIT-M" {
		t.Fatalf("unexpected quantity unit %q", got)
	}

	unknown := bytes.Replace(input, []byte("quantity: {value: 12.5, unit: UNIT-M}"), []byte("quantity: {value: 12.5, unit: UNIT-M, unknown: true}"), 1)
	decoder = yaml.NewDecoder(bytes.NewReader(unknown))
	decoder.KnownFields(true)
	if err := decoder.Decode(&document); err == nil {
		t.Fatal("strict decoding accepted an unknown typed quantity field")
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-052
func TestLoadBundleAggregatesSchemaV2ArchitectureSemantics(t *testing.T) {
	bundle, err := LoadBundle(filepath.Join("..", "examples", "coffee-appliance-six-view", "engmod.yml"))
	if err != nil {
		t.Fatalf("load semantic schema-v2 example: %v", err)
	}
	if len(bundle.Architecture.Semantics.Elements) < 20 {
		t.Fatalf("expected canonical semantic elements in aggregate architecture, got %d", len(bundle.Architecture.Semantics.Elements))
	}
	if len(bundle.Architecture.Semantics.Relationships) < 10 {
		t.Fatalf("expected canonical semantic relationships in aggregate architecture, got %d", len(bundle.Architecture.Semantics.Relationships))
	}
	if bundle.Architecture.Semantics.Relationships[4].ItemRef != "ITEM-BREW-SELECTION" {
		t.Fatalf("typed message was not preserved: %+v", bundle.Architecture.Semantics.Relationships[4])
	}
}
