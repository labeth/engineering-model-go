// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package engmodel

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func TestGenerateSysMLV2DeterministicAndPreservesStableIDs(t *testing.T) {
	first, err := GenerateSysMLV2FromFile("architecture.yml")
	if err != nil {
		t.Fatalf("first export: %v (%+v)", err, first.Diagnostics)
	}
	second, err := GenerateSysMLV2FromFile("architecture.yml")
	if err != nil {
		t.Fatalf("second export: %v (%+v)", err, second.Diagnostics)
	}
	if first.Text != second.Text {
		t.Fatal("SysML projection is not deterministic")
	}
	for _, stableID := range []string{"FU-SYSML-EXPORTER", "IF-CLI-ENGSYSML", "DO-CANONICAL-SEMANTIC-MODEL", "DO-SYSML-V2-MODEL"} {
		if !strings.Contains(first.Text, stableID) {
			t.Fatalf("output does not preserve stable ID %q", stableID)
		}
	}
	if hasSysMLDiagnostic(first.Diagnostics, "sysml.conformance_unverified") {
		t.Fatalf("toolchain validation is an external gate, not a generator warning: %+v", first.Diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-SYSML-V2-MODEL
func TestGenerateSysMLV2ProjectionRejectsMissingNativeRule(t *testing.T) {
	text, diagnostics := GenerateSysMLV2Projection(model.SemanticModel{
		ID: "MODEL-A",
		Elements: []model.SemanticElement{{
			ID: "X-A", Kind: model.SemanticElementKind("future_kind"),
		}},
	})
	if strings.Contains(text, "item def 'X-A';") {
		t.Fatalf("unsupported element used the forbidden generic-item fallback:\n%s", text)
	}
	if !hasSysMLDiagnostic(diagnostics, "sysml.missing_element_rule") {
		t.Fatalf("expected missing-rule diagnostic, got %+v", diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func TestGenerateSysMLV2ProjectionEmitsStructuralAndBehavioralSemantics(t *testing.T) {
	lower, upper, unique := 1, 2, false
	semantic := model.SemanticModel{
		ID:      "MODEL-A",
		Imports: []model.SemanticImport{{Namespace: "MODEL-A", Imported: "Domain", Visibility: "private", Recursive: true}},
		Elements: []model.SemanticElement{
			{ID: "MODEL-A", Kind: model.ElementPackageDefinition},
			{ID: "PKG-A", Kind: model.ElementPackageDefinition, Namespace: "MODEL-A"},
			{ID: "PART-DEF", Kind: model.ElementPartDefinition, Namespace: "PKG-A"},
			{ID: "PART-A", Kind: model.ElementPartUsage, Namespace: "PKG-A", TypeRef: "PART-DEF", Multiplicity: &model.Multiplicity{Lower: &lower, Upper: &upper}, Ordered: true, Unique: &unique},
			{ID: "ITEM-A", Kind: model.ElementItemDefinition, Namespace: "PKG-A"},
			{ID: "PORT-A", Kind: model.ElementPortUsage, Namespace: "PKG-A", TypeRef: "ITEM-A", Conjugated: true},
			{ID: "ACTION-A", Kind: model.ElementActionDefinition, Namespace: "PKG-A", Features: []model.SemanticFeature{
				{Name: "request", Kind: model.FeatureParameter, Direction: "in", Type: "ITEM-A"},
				{Name: "response", Kind: model.FeatureParameter, Direction: "out", Type: "ITEM-A"},
			}},
			{ID: "CONTROL-A", Kind: model.ElementControlNode, Namespace: "PKG-A", ControlKind: "fork"},
			{ID: "STATE-A", Kind: model.ElementStateDefinition, Namespace: "PKG-A"},
			{ID: "STATE-B", Kind: model.ElementStateDefinition, Namespace: "PKG-A"},
			{ID: "EVENT-A", Kind: model.ElementEventDefinition, Namespace: "PKG-A"},
		},
		Relationships: []model.SemanticRelationship{
			{ID: "CONNECT-A", Kind: model.RelationshipConnection, Source: "PORT-A", Target: "PART-A"},
			{ID: "BIND-A", Kind: model.RelationshipBinding, Source: "PART-A", Target: "ITEM-A"},
			{ID: "TRANSFER-A", Kind: model.RelationshipTransfer, Source: "ACTION-A", Target: "CONTROL-A", ItemRef: "ITEM-A"},
			{ID: "SUCCESSION-A", Kind: model.RelationshipSuccession, Source: "ACTION-A", Target: "CONTROL-A"},
			{ID: "TRANSITION-A", Kind: model.RelationshipTransition, Source: "STATE-A", Target: "STATE-B", Triggers: []string{"EVENT-A"},
				Guard:  &model.SemanticExpression{Language: "expression", Value: "ready"},
				Effect: &model.SemanticExpression{Language: "expression", Value: "notify"}},
			{ID: "ALLOCATE-A", Kind: model.RelationshipAllocation, Source: "PART-A", Target: "ITEM-A"},
		},
	}

	text, diagnostics := GenerateSysMLV2Projection(semantic)
	for _, expected := range []string{
		"private import 'Domain'::*;", "package 'PKG-A' {", "part def 'PART-DEF';",
		"part 'PART-A' : 'PART-DEF' [1..2] ordered nonunique;", `\"typeRef\":\"PART-DEF\"`, `\"multiplicity\":{\"lower\":1,\"upper\":2}`,
		`\"ordered\":true`, `\"unique\":false`, `\"conjugated\":true`,
		`\"name\":\"request\"`, `\"direction\":\"out\"`,
		"connection 'CONNECT-A' connect 'PORT-A' to 'PART-A';", `conceptKind = "connection"`,
		"flow 'TRANSFER-A' of 'ITEM-A'", `\"itemRef\":\"ITEM-A\"`,
		"transition 'TRANSITION-A'", `\"triggers\":[\"EVENT-A\"]`,
		`\"guard\":{\"language\":\"expression\",\"value\":\"ready\"}`,
		"allocation 'ALLOCATE-A'",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in projection:\n%s", expected, text)
		}
	}
	if hasSysMLDiagnostic(diagnostics, "sysml.lossy_detached_behavior") {
		t.Fatalf("owned transition behavior was reported as detached: %+v", diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func TestGenerateSysMLV2ProjectionEmitsRequirementsCasesValuesAndExtensions(t *testing.T) {
	quantity := model.SemanticValue{Kind: "quantity", Quantity: &model.SemanticQuantity{Value: 5, Unit: "UNIT-S"}}
	semantic := model.SemanticModel{
		ID: "MODEL-A",
		Elements: []model.SemanticElement{
			{ID: "MODEL-A", Kind: model.ElementPackageDefinition},
			{ID: "REQ-A", Kind: model.ElementRequirementDefinition, Namespace: "MODEL-A"},
			{ID: "CONCERN-A", Kind: model.ElementConcernDefinition, Namespace: "MODEL-A"},
			{ID: "STAKEHOLDER-A", Kind: model.ElementStakeholderDefinition, Namespace: "MODEL-A"},
			{ID: "CONSTRAINT-A", Kind: model.ElementConstraintDefinition, Namespace: "MODEL-A"},
			{ID: "CALC-A", Kind: model.ElementCalculationDefinition, Namespace: "MODEL-A", Features: []model.SemanticFeature{{
				Name: "duration", Kind: model.FeatureAttribute,
				Value: &model.SemanticExpression{Kind: "literal", TypedValue: &quantity},
			}}},
			{ID: "CASE-A", Kind: model.ElementCaseDefinition, Namespace: "MODEL-A"},
			{ID: "ANALYSIS-A", Kind: model.ElementAnalysisCaseDefinition, Namespace: "MODEL-A"},
			{ID: "VERIFY-A", Kind: model.ElementVerificationCaseUsage, Namespace: "MODEL-A"},
			{ID: "USE-A", Kind: model.ElementUseCaseDefinition, Namespace: "MODEL-A"},
			{ID: "VIEW-A", Kind: model.ElementViewUsage, Namespace: "MODEL-A"},
			{ID: "VIEWPOINT-A", Kind: model.ElementViewpointDefinition, Namespace: "MODEL-A"},
			{ID: "UNIT-S", Kind: model.ElementUnitDefinition, Namespace: "MODEL-A"},
			{ID: "EXT-A", Kind: model.ElementExtension, Namespace: "MODEL-A",
				ExtensionNamespace: "engineering", Extension: "engineering.risk", Targets: []string{"REQ-A"}},
		},
		Relationships: []model.SemanticRelationship{
			{ID: "SAT-A", Kind: model.RelationshipSatisfaction, Source: "REQ-A", Target: "STAKEHOLDER-A"},
			{ID: "VER-A", Kind: model.RelationshipVerification, Source: "VERIFY-A", Target: "REQ-A"},
		},
	}

	text, diagnostics := GenerateSysMLV2Projection(semantic)
	for _, expected := range []string{
		"requirement def 'REQ-A';", "concern def 'CONCERN-A';", "item def 'STAKEHOLDER-A';",
		"constraint def 'CONSTRAINT-A';", "calc def 'CALC-A'", `\"unit\":\"UNIT-S\"`,
		"case def 'CASE-A';", "analysis def 'ANALYSIS-A';", "verification 'VERIFY-A'",
		"use case def 'USE-A';", "view 'VIEW-A'", "viewpoint def 'VIEWPOINT-A';",
		"satisfy requirement 'STAKEHOLDER-A' by 'REQ-A';", `conceptKind = "satisfaction"`,
		"verification 'VER-A' : EngineeringVerification", `conceptKind = "verification"`,
		"conceptKind = \"engineering_extension\";",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("missing %q in projection:\n%s", expected, text)
		}
	}
	if hasSysMLDiagnostic(diagnostics, "sysml.unsupported_element") {
		t.Fatalf("standard requirement/case concepts were treated as unsupported: %+v", diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN
func TestSysMLV2CoverageManifestIsCompleteAndHonest(t *testing.T) {
	manifest := SysMLV2Coverage()
	if manifest.SchemaVersion != "2" {
		t.Fatalf("invalid coverage manifest: %+v", manifest)
	}
	if err := ValidateSysMLV2Coverage(manifest); err != nil {
		t.Fatal(err)
	}
	want := SysMLCoverageTotals{
		Metaclasses: 175, OwnedProperties: 415, EffectiveProperties: 13318,
		InheritanceEdges: 209, DerivedProperties: 328, Relationships: 66,
		CoverageMapped: 262, CoverageDerived: 328, CoverageUnimplemented: 0,
		NativeRendered: 54, AbstractOrImplicit: 121, MissingRenderer: 0,
		ClassifiedRelationships: 66,
	}
	if manifest.Totals != want {
		t.Fatalf("unexpected official totals:\ngot  %+v\nwant %+v", manifest.Totals, want)
	}
}

// TRLC-LINKS: REQ-EMG-036
func hasSysMLDiagnostic(diagnostics []model.SemanticDiagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
