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

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044
func TestLoadBundle(t *testing.T) {
	p := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	b, err := LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	if b.Architecture.Model.ID != "sample-payments-layered-model" {
		t.Fatalf("unexpected model id: %q", b.Architecture.Model.ID)
	}
	if len(b.Architecture.Views) != 7 {
		t.Fatalf("expected 7 views, got %d", len(b.Architecture.Views))
	}
	if b.Architecture.SchemaVersion != CurrentSchemaVersion ||
		b.Catalog.SchemaVersion != CurrentSchemaVersion ||
		b.Requirements.SchemaVersion != CurrentSchemaVersion ||
		b.Design.SchemaVersion != CurrentSchemaVersion ||
		b.Decisions.SchemaVersion != CurrentSchemaVersion {
		t.Fatalf("legacy documents were not normalized to schema version %d", CurrentSchemaVersion)
	}
}

// TRLC-LINKS: REQ-EMG-001
func TestLoadBundle_Decisions(t *testing.T) {
	p := filepath.Join("..", "architecture.yml")
	b, err := LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}

	if len(b.Architecture.Decisions) == 0 {
		t.Fatalf("expected root model decisions")
	}
	if filepath.Base(b.DecisionsPath) != "decisions.yml" {
		t.Fatalf("unexpected decisions path: %q", b.DecisionsPath)
	}
	if len(b.Decisions.Decisions) != len(b.Architecture.Decisions) {
		t.Fatalf("expected decisions document and architecture decisions to match")
	}
	d := b.Architecture.Decisions[0]
	if d.ID != "ADR-EMG-001" {
		t.Fatalf("unexpected decision id: %q", d.ID)
	}
	if d.Status != "accepted" {
		t.Fatalf("unexpected decision status: %q", d.Status)
	}
	if len(d.Consequences) == 0 {
		t.Fatalf("expected decision consequences")
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func TestLoadBundleUsesExplicitDocumentReferences(t *testing.T) {
	bundle, err := LoadBundle(filepath.Join("..", "architecture.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if bundle.Architecture.Model.BaseCatalogRef != "" {
		t.Fatalf("root model should use explicit document references, got legacy ref %q", bundle.Architecture.Model.BaseCatalogRef)
	}
	refs := bundle.Architecture.Model.Documents
	if refs.Catalog != "./catalog.yml" || refs.Requirements != "./requirements.yml" || refs.Design != "./design.yml" || refs.Decisions != "./decisions.yml" {
		t.Fatalf("unexpected explicit references: %+v", refs)
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func TestResolveDocumentReferencesRejectsConflictingCatalogAliases(t *testing.T) {
	_, err := ResolveDocumentReferences(ModelMeta{
		BaseCatalogRef: "./legacy-catalog.yml",
		Documents:      DocumentReferences{Catalog: "./catalog.yml"},
	})
	if err == nil || !strings.Contains(err.Error(), "conflicts") {
		t.Fatalf("expected conflicting catalog reference error, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func TestLoadBundleRejectsMissingExplicitCompanionDocument(t *testing.T) {
	dir := t.TempDir()
	architecture := `schemaVersion: 1
model:
  id: TEST-MODEL
  documents:
    catalog: ./catalog.yml
    requirements: ./missing-requirements.yml
`
	catalog := "schemaVersion: 1\ncatalog: {}\n"
	if err := os.WriteFile(filepath.Join(dir, "architecture.yml"), []byte(architecture), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "catalog.yml"), []byte(catalog), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := LoadBundle(filepath.Join(dir, "architecture.yml"))
	if err == nil || !strings.Contains(err.Error(), "explicit requirements document") {
		t.Fatalf("expected missing explicit companion error, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-044
func TestLoadBundleUsesCustomCompanionPaths(t *testing.T) {
	dir := t.TempDir()
	files := map[string]string{
		"architecture.yml": `schemaVersion: 1
model:
  id: TEST-MODEL
  documents:
    catalog: ./vocabulary.yml
    requirements: ./needs.yml
    design: ./solution.yml
    decisions: ./adrs.yml
`,
		"vocabulary.yml": "schemaVersion: 1\ncatalog: {}\n",
		"needs.yml":      "schemaVersion: 1\nlintRun: {}\nrequirements: []\n",
		"solution.yml":   "schemaVersion: 1\ndesign:\n  id: TEST-DESIGN\n  title: Test\n",
		"adrs.yml":       "schemaVersion: 1\ndecisions: []\n",
	}
	for name, content := range files {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	bundle, err := LoadBundle(filepath.Join(dir, "architecture.yml"))
	if err != nil {
		t.Fatalf("load explicit document set: %v", err)
	}
	if filepath.Base(bundle.CatalogPath) != "vocabulary.yml" ||
		filepath.Base(bundle.RequirementsPath) != "needs.yml" ||
		filepath.Base(bundle.DesignPath) != "solution.yml" ||
		filepath.Base(bundle.DecisionsPath) != "adrs.yml" {
		t.Fatalf("explicit paths were not resolved: %+v", bundle)
	}
}

// TRLC-LINKS: REQ-EMG-045
// ENGMODEL-LINKS: DO-MODEL-AUTHORING-CONTRACT
func TestBuildAuthoringContractDescribesCanonicalDocuments(t *testing.T) {
	bundle, err := LoadBundle(filepath.Join("..", "architecture.yml"))
	if err != nil {
		t.Fatal(err)
	}
	contract := BuildAuthoringContract(bundle)
	if contract.ContractVersion != AuthoringContractVersion || contract.CanonicalFormat != "YAML" || contract.SchemaAuthority != "CUE" {
		t.Fatalf("unexpected contract header: %+v", contract)
	}
	if len(contract.Documents) != 5 {
		t.Fatalf("expected five canonical documents, got %d", len(contract.Documents))
	}
	for _, document := range contract.Documents {
		if document.SchemaVersion != CurrentSchemaVersion || document.Path == "" || document.SchemaPath == "" || len(document.TopLevelFields) == 0 {
			t.Fatalf("incomplete document contract: %+v", document)
		}
	}
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
