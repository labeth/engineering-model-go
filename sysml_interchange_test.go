// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package engmodel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func TestSysMLProjectSourceReconstructsCanonicalSemantics(t *testing.T) {
	result, err := GenerateSysMLV2FromFile("architecture.yml")
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	path := filepath.Join(t.TempDir(), "model.sysml")
	if err := os.WriteFile(path, []byte(result.Text), 0o644); err != nil {
		t.Fatal(err)
	}
	reconstructed, err := ImportSysMLV2Project(path)
	if err != nil {
		t.Fatalf("import: %v", err)
	}
	canonical, err := model.LoadCanonicalBundle("architecture.yml")
	if err != nil {
		t.Fatalf("load canonical model: %v", err)
	}
	if differences := CompareSysMLV2RoundTrip(canonical.Semantic(), reconstructed); len(differences) != 0 {
		t.Fatalf("round trip differences: %v", differences)
	}
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func TestCompareSysMLV2RoundTripNamesSemanticDifferenceClasses(t *testing.T) {
	expected := model.SemanticModel{
		ID:      "MODEL-A",
		Imports: []model.SemanticImport{{Namespace: "MODEL-A", Imported: "Library"}},
		Elements: []model.SemanticElement{{
			ID: "ELEMENT-A", Kind: model.ElementPartUsage, Owner: "OWNER-A",
			Features: []model.SemanticFeature{{Name: "value", Value: &model.SemanticExpression{Value: "1"}}},
			Metadata: []model.SemanticMetadata{{Namespace: "engineering", Type: "engineering.risk", Target: "ELEMENT-A"}},
		}},
		Relationships: []model.SemanticRelationship{{ID: "REL-A", Kind: model.RelationshipDependency, Source: "ELEMENT-A", Target: "OWNER-A"}},
	}
	actual := expected
	actual.Imports = nil
	actual.Elements = append([]model.SemanticElement(nil), expected.Elements...)
	actual.Elements[0].Owner = "OWNER-B"
	actual.Elements[0].Features = nil
	actual.Elements[0].Metadata = nil
	actual.Relationships = nil

	differences := strings.Join(CompareSysMLV2RoundTrip(expected, actual), ",")
	for _, expectedClass := range []string{"library/project references", "ownership", "typed features and expressions", "typed extensions", "relationships and expressions"} {
		if !strings.Contains(differences, expectedClass) {
			t.Fatalf("missing %q in %q", expectedClass, differences)
		}
	}
}

// TRLC-LINKS: REQ-EMG-040
func TestCompareSysMLV2RoundTripNormalizesDerivedAndImpliedProperties(t *testing.T) {
	expected := model.SemanticModel{
		ID: "MODEL-A",
		Elements: []model.SemanticElement{{
			ID: "ELEMENT-A", Kind: model.ElementItemDefinition,
			Properties: map[string]model.MetamodelPropertyValue{
				"elementId": {Values: []model.MetamodelValue{{Kind: "string", String: "ELEMENT-A"}}},
				"name":      {Derived: true, Values: []model.MetamodelValue{{Kind: "string", String: "Element A"}}},
				"owner":     {Implied: true, Values: []model.MetamodelValue{{Kind: "reference", Reference: "MODEL-A"}}},
			},
		}},
	}
	actual := expected
	actual.Elements = append([]model.SemanticElement(nil), expected.Elements...)
	actual.Elements[0].Properties = map[string]model.MetamodelPropertyValue{
		"elementId": expected.Elements[0].Properties["elementId"],
	}
	if differences := CompareSysMLV2RoundTrip(expected, actual); len(differences) != 0 {
		t.Fatalf("derived/implied properties must be normalized: %v", differences)
	}
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func TestExportSysMLV2ProjectRequiresSysand(t *testing.T) {
	_, err := ExportSysMLV2Project("architecture.yml", filepath.Join(t.TempDir(), "project"), filepath.Join(t.TempDir(), "model.kpar"), filepath.Join(t.TempDir(), "missing-sysand"))
	if err == nil || !strings.Contains(err.Error(), "sysand is unavailable") {
		t.Fatalf("expected clear unavailable tool error, got %v", err)
	}
}
