// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"testing"
)

// loadSelfSemanticModel projects the repository's own engineering model.
func loadSelfSemanticModel(t *testing.T) (SemanticModel, []SemanticDiagnostic) {
	t.Helper()
	bundle, err := LoadBundle(filepath.Join("..", "..", "engmod.yml"))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	return ProjectSemanticModel(bundle)
}

// failOnSemanticError fails the test on the first error diagnostic.
func failOnSemanticError(t *testing.T, diagnostics []SemanticDiagnostic, label string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SemanticSeverityError {
			t.Fatalf("%s: %+v", label, diagnostic)
		}
	}
}

// findSemanticElement returns the element with the given ID, or nil.
func findSemanticElement(elements []SemanticElement, id string) *SemanticElement {
	for i := range elements {
		if elements[i].ID == id {
			return &elements[i]
		}
	}
	return nil
}

// assertUniqueElementIDs fails the test when two elements share an ID.
func assertUniqueElementIDs(t *testing.T, elements []SemanticElement) {
	t.Helper()
	seen := map[string]bool{}
	for _, element := range elements {
		if seen[element.ID] {
			t.Fatalf("duplicate semantic element %q", element.ID)
		}
		seen[element.ID] = true
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelStableUniqueElements(t *testing.T) {
	semantic, diagnostics := loadSelfSemanticModel(t)
	failOnSemanticError(t, diagnostics, "unexpected projection error")

	assertUniqueElementIDs(t, semantic.Elements)
	unit := findSemanticElement(semantic.Elements, "FU-SYSML-EXPORTER")
	if unit == nil {
		t.Fatal("missing stable ID FU-SYSML-EXPORTER")
	}
	if unit.Owner != "FG-ARTIFACT-GENERATION" {
		t.Fatalf("unexpected owner: %q", unit.Owner)
	}
	if len(unit.Metadata) < 2 {
		t.Fatalf("expected source and catalog metadata, got %+v", unit.Metadata)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelRejectsDuplicateIDs(t *testing.T) {
	bundle := Bundle{Architecture: ArchitectureDocument{
		Model: ModelMeta{ID: "MODEL-A"},
		AuthoredArchitecture: AuthoredArchitecture{
			FunctionalGroups: []FunctionalGroup{
				{ID: "FG-A", Name: "First"},
				{ID: "FG-A", Name: "Second"},
			},
		},
	}}

	semantic, diagnostics := ProjectSemanticModel(bundle)
	count := 0
	for _, element := range semantic.Elements {
		if element.ID == "FG-A" {
			count++
		}
	}
	if count != 1 {
		t.Fatalf("expected one canonical element for duplicate ID, got %d", count)
	}
	if !hasSemanticDiagnostic(diagnostics, "semantic.duplicate_identity") {
		t.Fatalf("expected duplicate diagnostic, got %+v", diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelReportsUnsupportedAndLossyRelationships(t *testing.T) {
	bundle := Bundle{Architecture: ArchitectureDocument{
		Model: ModelMeta{ID: "MODEL-A"},
		AuthoredArchitecture: AuthoredArchitecture{
			FunctionalGroups: []FunctionalGroup{{ID: "FG-A"}},
			Mappings: []Mapping{
				{Type: "future_relation", From: "FG-A", To: "FG-A"},
				{Type: "depends_on", From: "", To: "FG-A"},
			},
		},
	}}

	_, diagnostics := ProjectSemanticModel(bundle)
	if !hasSemanticDiagnostic(diagnostics, "semantic.unsupported_relationship") {
		t.Fatalf("expected unsupported relationship diagnostic, got %+v", diagnostics)
	}
	if !hasSemanticDiagnostic(diagnostics, "semantic.lossy_relationship") {
		t.Fatalf("expected lossy relationship diagnostic, got %+v", diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func TestValidateSemanticModelIdentityRelationshipsAndOwnership(t *testing.T) {
	semantic := SemanticModel{
		ID: "MODEL-A",
		Elements: []SemanticElement{
			{ID: "MODEL-A", Kind: ElementPackageDefinition, Namespace: "MODEL-A"},
			{ID: "FU-A", Kind: ElementPartUsage, Namespace: "MODEL-A", Owner: "FG-MISSING"},
			{ID: "FU-A", Kind: ElementPartUsage, Namespace: "MODEL-A"},
		},
		Relationships: []SemanticRelationship{{
			ID: "REL-A", Kind: RelationshipDependency, Source: "FU-A", Target: "FU-MISSING", Owner: "OWNER-MISSING",
		}},
	}
	diagnostics := ValidateSemanticModel(semantic)
	for _, code := range []string{
		"semantic.duplicate_identity",
		"semantic.dangling_relationship",
		"semantic.invalid_ownership",
	} {
		if !hasSemanticDiagnostic(diagnostics, code) {
			t.Fatalf("expected %s diagnostic, got %+v", code, diagnostics)
		}
	}
}

func transferSuccessionFixtureBundle() Bundle {
	return Bundle{Architecture: ArchitectureDocument{
		Model: ModelMeta{ID: "MODEL-A"},
		AuthoredArchitecture: AuthoredArchitecture{
			FunctionalGroups: []FunctionalGroup{{ID: "FG-A"}},
			FunctionalUnits:  []FunctionalUnit{{ID: "FU-A", Group: "FG-A"}, {ID: "FU-B", Group: "FG-A"}},
			DataObjects:      []DataObject{{ID: "DO-A"}},
			Controls:         []Control{{ID: "CTRL-A"}},
			Machines: []Machine{{
				ID: "SM-A", Owner: "FU-A", Initial: "STATE-A",
				States:      []MachineState{{ID: "STATE-A"}, {ID: "STATE-B"}},
				Transitions: []MachineTransition{{ID: "TR-A", From: "STATE-A", To: "STATE-B", On: "EVT-A"}},
			}},
			Events: []Event{{ID: "EVT-A"}},
			Flows: []Flow{{
				ID: "FLOW-A", SourceRef: "FU-A", DestinationRef: "FU-B", DataRefs: []string{"DO-A"},
				Steps: []FlowStep{
					{ID: "STEP-A", Ref: "FU-A", DataOut: []string{"DO-A"}, Next: []string{"STEP-B"}},
					{ID: "STEP-B", Ref: "FU-B", DataIn: []string{"DO-A"}},
				},
			}},
			Mappings: []Mapping{
				{Type: "contains", From: "FG-A", To: "FU-A"},
				{Type: "guarded_by", From: "STATE-A", To: "CTRL-A"},
			},
		},
	}}
}

// countRelationships counts the relationships with the kind, source and target of want.
func countRelationships(relationships []SemanticRelationship, want SemanticRelationship) int {
	count := 0
	for _, relationship := range relationships {
		if relationship.Kind == want.Kind && relationship.Source == want.Source && relationship.Target == want.Target {
			count++
		}
	}
	return count
}

func assertSeparatedRelationshipKinds(t *testing.T, relationships []SemanticRelationship) {
	t.Helper()
	if !hasRelationshipKind(relationships, RelationshipTransfer) {
		t.Fatalf("flow endpoints were not projected as a transfer: %+v", relationships)
	}
	if !hasRelationshipKind(relationships, RelationshipSuccession) {
		t.Fatalf("flow step ordering was not projected as succession: %+v", relationships)
	}
	if !hasRelationshipKind(relationships, RelationshipTransition) || !hasRelationshipKind(relationships, RelationshipGuard) {
		t.Fatalf("machine transitions and guarded_by mappings were not projected: %+v", relationships)
	}
	if hasRelationshipKind(relationships, SemanticRelationshipKind("flow")) {
		t.Fatalf("ambiguous flow relationship remains: %+v", relationships)
	}
}

func assertFlowParameterFeatures(t *testing.T, elements []SemanticElement) {
	t.Helper()
	flow := findSemanticElement(elements, "FLOW-A")
	if flow == nil || len(flow.Features) != 1 || flow.Features[0].Kind != FeatureParameter {
		t.Fatalf("flow parameters not projected: %+v", flow)
	}
	step := findSemanticElement(elements, "STEP-A")
	if step == nil || len(step.Features) != 1 || step.Features[0].Direction != "out" {
		t.Fatalf("step output parameter not projected: %+v", step)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelSeparatesTransfersSuccessionsAndTransitions(t *testing.T) {
	semantic, diagnostics := ProjectSemanticModel(transferSuccessionFixtureBundle())
	assertSeparatedRelationshipKinds(t, semantic.Relationships)
	if containmentCount := countRelationships(semantic.Relationships, SemanticRelationship{Kind: RelationshipContainment, Source: "FG-A", Target: "FU-A"}); containmentCount != 1 {
		t.Fatalf("legacy ownership aliases produced %d canonical containment relationships", containmentCount)
	}
	if hasSemanticDiagnostic(diagnostics, "semantic.legacy_lifecycle_attachment") || hasSemanticDiagnostic(diagnostics, "semantic.unsupported_relationship") {
		t.Fatalf("guarded_by from a machine state must project without legacy diagnostics, got %+v", diagnostics)
	}
	assertFlowParameterFeatures(t, semantic.Elements)
}

func canonicalContentFixtureBundle() Bundle {
	return Bundle{Architecture: ArchitectureDocument{
		Model: ModelMeta{ID: "MODEL-A"},
		Semantics: SemanticContent{
			Imports: []SemanticImport{{Namespace: "MODEL-A", Imported: "Domain", Visibility: "public"}},
			Elements: []SemanticElement{
				{ID: "PKG-A", Kind: ElementPackageDefinition},
				{ID: "PART-A", Kind: ElementPartUsage, Namespace: "PKG-A", TypeRef: "PART-DEF", Conjugated: false,
					Specializes: []string{"PART-BASE"}, Subsets: []string{"PART-SET"}, Redefines: []string{"PART-OLD"}},
				{ID: "ACTION-A", Kind: ElementActionDefinition, Namespace: "PKG-A", Features: []SemanticFeature{{
					Name: "input", Kind: FeatureParameter, Direction: "in", Type: "ITEM-A",
				}}},
				{ID: "STATE-A", Kind: ElementStateDefinition, Namespace: "PKG-A"},
				{ID: "STATE-B", Kind: ElementStateDefinition, Namespace: "PKG-A"},
				{ID: "EVENT-A", Kind: ElementEventDefinition, Namespace: "PKG-A"},
			},
			Relationships: []SemanticRelationship{
				{ID: "BIND-A", Kind: RelationshipBinding, Source: "PART-A", Target: "ACTION-A"},
				{ID: "TRANSITION-A", Kind: RelationshipTransition, Source: "STATE-A", Target: "STATE-B", Triggers: []string{"EVENT-A"},
					Guard:  &SemanticExpression{Language: "expression", Value: "ready"},
					Effect: &SemanticExpression{Language: "expression", Value: "notify"}},
			},
		},
	}}
}

// partKeepsStructure reports whether part preserved the namespace, type and
// specialization lists the canonical content declared.
func partKeepsStructure(part *SemanticElement) bool {
	if part == nil {
		return false
	}
	if part.Namespace != "PKG-A" || part.TypeRef != "PART-DEF" {
		return false
	}
	return len(part.Specializes) == 1 && len(part.Subsets) == 1 && len(part.Redefines) == 1
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelPreservesCanonicalStructuralAndBehavioralContent(t *testing.T) {
	semantic, diagnostics := ProjectSemanticModel(canonicalContentFixtureBundle())
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected canonical projection diagnostics: %+v", diagnostics)
	}
	if len(semantic.Imports) != 1 || !hasRelationshipKind(semantic.Relationships, RelationshipBinding) || !hasRelationshipKind(semantic.Relationships, RelationshipTransition) {
		t.Fatalf("canonical semantic content was not preserved: %+v", semantic)
	}
	part := findSemanticElement(semantic.Elements, "PART-A")
	if !partKeepsStructure(part) {
		t.Fatalf("structural semantics were not preserved: %+v", part)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func TestValidateSemanticModelStructuralAndBehavioralInvariants(t *testing.T) {
	lower, upper := 2, 1
	semantic := SemanticModel{
		ID:      "MODEL-A",
		Imports: []SemanticImport{{Namespace: "MISSING", Imported: ""}},
		Elements: []SemanticElement{
			{ID: "MODEL-A", Kind: ElementPackageDefinition},
			{ID: "ACTION-A", Kind: ElementActionDefinition, Namespace: "MODEL-A", Features: []SemanticFeature{{
				Name: "bad", Kind: FeatureParameter, Direction: "sideways", Multiplicity: &Multiplicity{Lower: &lower, Upper: &upper},
			}}},
		},
		Relationships: []SemanticRelationship{{
			ID: "TRANSITION-A", Kind: RelationshipTransition, Source: "ACTION-A", Target: "ACTION-A", Triggers: []string{"EVT-MISSING"},
		}},
	}
	diagnostics := ValidateSemanticModel(semantic)
	for _, code := range []string{"semantic.invalid_import", "semantic.invalid_parameter_direction", "semantic.invalid_multiplicity", "semantic.dangling_relationship"} {
		if !hasSemanticDiagnostic(diagnostics, code) {
			t.Fatalf("expected %s diagnostic, got %+v", code, diagnostics)
		}
	}
}

// anySemanticElement reports whether match holds for any element.
func anySemanticElement(elements []SemanticElement, match func(SemanticElement) bool) bool {
	for _, element := range elements {
		if match(element) {
			return true
		}
	}
	return false
}

func assertMappedElementKinds(t *testing.T, elements []SemanticElement) {
	t.Helper()
	expectKinds := map[string]SemanticElementKind{
		"REQ-EMG-035":              ElementRequirementDefinition,
		"ACT-ARCHITECTURE-AUTHOR":  ElementStakeholderDefinition,
		"VER-MCP-PATH-BOUNDARY":    ElementVerificationCaseUsage,
		"VIEW-ARCHITECTURE-INTENT": ElementViewUsage,
		"RISK-MCP-PATH-ESCAPE":     ElementExtension,
		"ADR-EMG-011":              ElementExtension,
	}
	seen := map[string]SemanticElement{}
	for _, element := range elements {
		seen[element.ID] = element
	}
	for id, kind := range expectKinds {
		if element, ok := seen[id]; !ok || element.Kind != kind {
			t.Fatalf("expected %s to map to %s, got %+v", id, kind, element)
		}
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelMapsRequirementsCasesViewsAndExtensions(t *testing.T) {
	semantic, diagnostics := loadSelfSemanticModel(t)
	diagnostics = append(diagnostics, ValidateSemanticModel(semantic)...)
	failOnSemanticError(t, diagnostics, "unexpected semantic error")

	assertMappedElementKinds(t, semantic.Elements)
	hasConcern := anySemanticElement(semantic.Elements, func(e SemanticElement) bool { return e.Kind == ElementConcernUsage })
	hasEvidence := anySemanticElement(semantic.Elements, func(e SemanticElement) bool { return e.Extension == "engineering.verification_evidence" })
	hasOwnershipPolicy := anySemanticElement(semantic.Elements, func(e SemanticElement) bool { return e.Extension == "engineering.code_ownership_policy" })
	if !hasConcern || !hasEvidence || !hasOwnershipPolicy {
		t.Fatalf("missing mapped concern/evidence/ownership extensions: concern=%t evidence=%t ownership=%t", hasConcern, hasEvidence, hasOwnershipPolicy)
	}
	if !hasRelationshipKind(semantic.Relationships, RelationshipVerification) {
		t.Fatal("control verification did not produce a verification relationship")
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL, FU-VALIDATION-ENGINE
func TestValidateSemanticModelAdvancedValueOccurrenceVariationAndMetadata(t *testing.T) {
	zero := 0
	semantic := SemanticModel{
		ID: "MODEL-A",
		Elements: []SemanticElement{
			{ID: "MODEL-A", Kind: ElementPackageDefinition},
			{ID: "REQ-A", Kind: ElementRequirementDefinition, Namespace: "MODEL-A"},
			{ID: "OCC-A", Kind: ElementOccurrenceUsage, Namespace: "MODEL-A", OccurrenceID: "same"},
			{ID: "OCC-B", Kind: ElementSnapshot, Namespace: "MODEL-A", OccurrenceID: "same", PortionOf: "MISSING"},
			{ID: "VAR-A", Kind: ElementPartDefinition, Namespace: "MODEL-A", Variants: []string{"MISSING"}},
			{ID: "EXT-A", Kind: ElementExtension, Namespace: "MODEL-A", Authored: true,
				Extension: "engineering.risk", Targets: []string{"MISSING"},
				Metadata: []SemanticMetadata{{Type: "engineering.note", Target: "MISSING"}}},
			{ID: "CALC-A", Kind: ElementCalculationDefinition, Namespace: "MODEL-A", Features: []SemanticFeature{{
				Name: "result", Kind: FeatureAttribute, Multiplicity: &Multiplicity{Upper: &zero},
				Value: &SemanticExpression{Operator: "+", TypedValue: &SemanticValue{
					Kind: "quantity", Quantity: &SemanticQuantity{Value: 1},
				}},
			}}},
		},
	}
	diagnostics := ValidateSemanticModel(semantic)
	for _, code := range []string{
		"semantic.duplicate_occurrence_identity",
		"semantic.dangling_reference",
		"semantic.invalid_variant",
		"semantic.invalid_extension_type",
		"semantic.invalid_extension_target",
		"semantic.invalid_metadata_type",
		"semantic.invalid_metadata_target",
		"semantic.invalid_expression",
		"semantic.invalid_quantity",
		"semantic.value_multiplicity_mismatch",
	} {
		if !hasSemanticDiagnostic(diagnostics, code) {
			t.Fatalf("expected %s diagnostic, got %+v", code, diagnostics)
		}
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
func hasRelationshipKind(relationships []SemanticRelationship, kind SemanticRelationshipKind) bool {
	for _, relationship := range relationships {
		if relationship.Kind == kind {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-035
func hasSemanticDiagnostic(diagnostics []SemanticDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
