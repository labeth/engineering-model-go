// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelStableUniqueElements(t *testing.T) {
	bundle, err := LoadBundle(filepath.Join("..", "engmod.yml"))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	semantic, diagnostics := ProjectSemanticModel(bundle)
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SemanticSeverityError {
			t.Fatalf("unexpected projection error: %+v", diagnostic)
		}
	}

	seen := map[string]bool{}
	var unit *SemanticElement
	for i := range semantic.Elements {
		element := &semantic.Elements[i]
		if seen[element.ID] {
			t.Fatalf("duplicate semantic element %q", element.ID)
		}
		seen[element.ID] = true
		if element.ID == "FU-SYSML-EXPORTER" {
			unit = element
		}
	}
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

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelSeparatesTransfersSuccessionsAndTransitions(t *testing.T) {
	bundle := Bundle{Architecture: ArchitectureDocument{
		Model: ModelMeta{ID: "MODEL-A"},
		AuthoredArchitecture: AuthoredArchitecture{
			FunctionalGroups: []FunctionalGroup{{ID: "FG-A"}},
			FunctionalUnits:  []FunctionalUnit{{ID: "FU-A", Group: "FG-A"}, {ID: "FU-B", Group: "FG-A"}},
			DataObjects:      []DataObject{{ID: "DO-A"}},
			States:           []State{{ID: "STATE-A"}, {ID: "STATE-B"}},
			Events:           []Event{{ID: "EVT-A"}},
			Flows: []Flow{{
				ID: "FLOW-A", SourceRef: "FU-A", DestinationRef: "FU-B", DataRefs: []string{"DO-A"},
				Steps: []FlowStep{
					{ID: "STEP-A", Ref: "FU-A", DataOut: []string{"DO-A"}, Next: []string{"STEP-B"}},
					{ID: "STEP-B", Ref: "FU-B", DataIn: []string{"DO-A"}},
				},
			}},
			Mappings: []Mapping{
				{Type: "contains", From: "FG-A", To: "FU-A"},
				{Type: "transitions_to", From: "STATE-A", To: "STATE-B"},
				{Type: "triggered_by", From: "STATE-A", To: "EVT-A"},
			},
		},
	}}

	semantic, diagnostics := ProjectSemanticModel(bundle)
	if !hasRelationshipKind(semantic.Relationships, RelationshipTransfer) {
		t.Fatalf("flow endpoints were not projected as a transfer: %+v", semantic.Relationships)
	}
	if !hasRelationshipKind(semantic.Relationships, RelationshipSuccession) {
		t.Fatalf("flow step ordering was not projected as succession: %+v", semantic.Relationships)
	}
	if !hasRelationshipKind(semantic.Relationships, RelationshipTransition) || !hasRelationshipKind(semantic.Relationships, RelationshipEventTrigger) {
		t.Fatalf("lifecycle mappings were not preserved: %+v", semantic.Relationships)
	}
	if hasRelationshipKind(semantic.Relationships, SemanticRelationshipKind("flow")) {
		t.Fatalf("ambiguous flow relationship remains: %+v", semantic.Relationships)
	}
	containmentCount := 0
	for _, relationship := range semantic.Relationships {
		if relationship.Kind == RelationshipContainment && relationship.Source == "FG-A" && relationship.Target == "FU-A" {
			containmentCount++
		}
	}
	if containmentCount != 1 {
		t.Fatalf("legacy ownership aliases produced %d canonical containment relationships", containmentCount)
	}
	if !hasSemanticDiagnostic(diagnostics, "semantic.legacy_lifecycle_attachment") {
		t.Fatalf("expected explicit legacy transition attachment diagnostic, got %+v", diagnostics)
	}
	var flow, step *SemanticElement
	for i := range semantic.Elements {
		switch semantic.Elements[i].ID {
		case "FLOW-A":
			flow = &semantic.Elements[i]
		case "STEP-A":
			step = &semantic.Elements[i]
		}
	}
	if flow == nil || len(flow.Features) != 1 || flow.Features[0].Kind != FeatureParameter {
		t.Fatalf("flow parameters not projected: %+v", flow)
	}
	if step == nil || len(step.Features) != 1 || step.Features[0].Direction != "out" {
		t.Fatalf("step output parameter not projected: %+v", step)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelPreservesCanonicalStructuralAndBehavioralContent(t *testing.T) {
	bundle := Bundle{Architecture: ArchitectureDocument{
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

	semantic, diagnostics := ProjectSemanticModel(bundle)
	if len(diagnostics) != 0 {
		t.Fatalf("unexpected canonical projection diagnostics: %+v", diagnostics)
	}
	if len(semantic.Imports) != 1 || !hasRelationshipKind(semantic.Relationships, RelationshipBinding) || !hasRelationshipKind(semantic.Relationships, RelationshipTransition) {
		t.Fatalf("canonical semantic content was not preserved: %+v", semantic)
	}
	var part *SemanticElement
	for i := range semantic.Elements {
		if semantic.Elements[i].ID == "PART-A" {
			part = &semantic.Elements[i]
			break
		}
	}
	if part == nil || part.Namespace != "PKG-A" || part.TypeRef != "PART-DEF" || len(part.Specializes) != 1 || len(part.Subsets) != 1 || len(part.Redefines) != 1 {
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

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: DO-CANONICAL-SEMANTIC-MODEL
func TestProjectSemanticModelMapsRequirementsCasesViewsAndExtensions(t *testing.T) {
	bundle, err := LoadBundle(filepath.Join("..", "engmod.yml"))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	semantic, diagnostics := ProjectSemanticModel(bundle)
	diagnostics = append(diagnostics, ValidateSemanticModel(semantic)...)
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == SemanticSeverityError {
			t.Fatalf("unexpected semantic error: %+v", diagnostic)
		}
	}

	expectKinds := map[string]SemanticElementKind{
		"REQ-EMG-035":              ElementRequirementDefinition,
		"ACT-ARCHITECTURE-AUTHOR":  ElementStakeholderDefinition,
		"VER-MCP-PATH-BOUNDARY":    ElementVerificationCaseUsage,
		"VIEW-ARCHITECTURE-INTENT": ElementViewUsage,
		"RISK-MCP-PATH-ESCAPE":     ElementExtension,
		"ADR-EMG-011":              ElementExtension,
	}
	seen := map[string]SemanticElement{}
	hasConcern := false
	hasEvidence := false
	hasOwnershipPolicy := false
	for _, element := range semantic.Elements {
		seen[element.ID] = element
		hasConcern = hasConcern || element.Kind == ElementConcernUsage
		hasEvidence = hasEvidence || element.Extension == "engineering.verification_evidence"
		hasOwnershipPolicy = hasOwnershipPolicy || element.Extension == "engineering.code_ownership_policy"
	}
	for id, kind := range expectKinds {
		if element, ok := seen[id]; !ok || element.Kind != kind {
			t.Fatalf("expected %s to map to %s, got %+v", id, kind, element)
		}
	}
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
