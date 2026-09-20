// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func TestOfficialMetamodelInventoryIsFullyRepresentable(t *testing.T) {
	metaclasses := 0
	properties := 0
	for metaclass, definition := range OfficialSysMLMetaclasses {
		metaclasses++
		for name, property := range definition.Properties {
			properties++
			value := fixturePropertyValue(property)
			diagnostics := validateMetamodelInstance(
				"FIXTURE", metaclass, map[string]MetamodelPropertyValue{name: value}, "fixture", false,
			)
			if len(diagnostics) != 0 {
				t.Fatalf("%s.%s is not representable: %+v", metaclass, name, diagnostics)
			}
		}
	}
	if metaclasses != 175 {
		t.Fatalf("represented %d metaclasses, want 175", metaclasses)
	}
	if properties != 13318 {
		t.Fatalf("represented %d effective metaclass properties, want 13318", properties)
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func TestGeneratedCUEAcceptsEveryOfficialMetaclassAndProperty(t *testing.T) {
	metaclasses := make([]string, 0, len(OfficialSysMLMetaclasses))
	for metaclass := range OfficialSysMLMetaclasses {
		metaclasses = append(metaclasses, metaclass)
	}
	sort.Strings(metaclasses)
	for _, metaclass := range metaclasses {
		t.Run(metaclass, func(t *testing.T) {
			definition := OfficialSysMLMetaclasses[metaclass]
			properties := make(map[string]MetamodelPropertyValue, len(definition.Properties))
			for name, property := range definition.Properties {
				properties[name] = fixturePropertyValue(property)
			}
			source, err := yaml.Marshal(map[string]any{
				"model": map[string]any{"id": "MODEL-A", "baseCatalogRef": "catalog.yml"},
				"semantics": map[string]any{"elements": []SemanticElement{{
					ID:         "FIXTURE-" + strings.NewReplacer("::", "-", ".", "-").Replace(metaclass),
					Kind:       SemanticElementKind("official_fixture"),
					Metaclass:  metaclass,
					Properties: properties,
				}}},
			})
			if err != nil {
				t.Fatal(err)
			}
			var document ArchitectureDocument
			if err := validateCanonicalYAML("official-fixture.yml", source, &document); err != nil {
				t.Fatalf("generated CUE rejected %s: %v", metaclass, err)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func TestOfficialMetamodelRejectsUnknownReadOnlyMultiplicityAndShape(t *testing.T) {
	tests := []struct {
		name       string
		metaclass  string
		properties map[string]MetamodelPropertyValue
		code       string
	}{
		{
			name: "unknown property", metaclass: "SysML::PartUsage",
			properties: map[string]MetamodelPropertyValue{"notOfficial": {Values: []MetamodelValue{{Kind: "string", String: "x"}}}},
			code:       "semantic.unknown_property",
		},
		{
			name: "read only property", metaclass: "SysML::PartUsage",
			properties: map[string]MetamodelPropertyValue{"partDefinition": {Values: []MetamodelValue{{Kind: "reference", Reference: "PART-DEF"}}}},
			code:       "semantic.read_only_property",
		},
		{
			name: "invalid multiplicity", metaclass: "KerML::Element",
			properties: map[string]MetamodelPropertyValue{"elementId": {}},
			code:       "semantic.invalid_property_multiplicity",
		},
		{
			name: "invalid value shape", metaclass: "KerML::Element",
			properties: map[string]MetamodelPropertyValue{"elementId": {Values: []MetamodelValue{{Kind: "reference", Reference: "X"}}}},
			code:       "semantic.invalid_property_value",
		},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			diagnostics := validateMetamodelInstance("FIXTURE", test.metaclass, test.properties, "fixture", false)
			if !hasSemanticDiagnostic(diagnostics, test.code) {
				t.Fatalf("expected %s, got %+v", test.code, diagnostics)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func TestGeneratedCUEChecksOfficialPropertyNamesReadOnlyMultiplicityAndShapes(t *testing.T) {
	tests := []struct {
		name, property string
	}{
		{"unknown", "notOfficial:\n          values: [{kind: string, string: x}]"},
		{"read-only", "partDefinition:\n          derived: false\n          values: [{kind: reference, reference: PART-DEF}]"},
		{"multiplicity", "elementId:\n          values: []"},
		{"shape", "elementId:\n          values: [{kind: reference, reference: WRONG}]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			metaclass := "KerML::Element"
			if test.name == "read-only" {
				metaclass = "SysML::PartUsage"
			}
			source := []byte(fmt.Sprintf(`model:
  id: MODEL-A
  baseCatalogRef: catalog.yml
semantics:
  elements:
    - id: FIXTURE
      kind: part_usage
      metaclass: %s
      properties:
        %s
`, metaclass, test.property))
			var document ArchitectureDocument
			if err := validateCanonicalYAML("fixture.yml", source, &document); err == nil {
				t.Fatal("expected generated CUE binding to reject invalid official property")
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func TestCanonicalMetamodelPreservesOwnershipNamespacesImportsAndRelationships(t *testing.T) {
	bundle := Bundle{Architecture: ArchitectureDocument{
		Model: ModelMeta{ID: "MODEL-A", Title: "Model"},
		Semantics: SemanticContent{
			Imports: []SemanticImport{
				{Namespace: "PKG-A", Imported: "SysML::Libraries::ScalarValues", Visibility: "private", Recursive: true},
				{Namespace: "PKG-B", Imported: "project://shared/model", Visibility: "public"},
			},
			Elements: []SemanticElement{
				{ID: "PKG-A", Kind: ElementPackageDefinition, Metaclass: "KerML::Package"},
				{ID: "PKG-B", Kind: ElementPackageDefinition, Metaclass: "KerML::Package", Owner: "PKG-A", Namespace: "PKG-A"},
				{ID: "PART-A", Kind: ElementPartUsage, Metaclass: "SysML::PartUsage", Owner: "PKG-B", Namespace: "PKG-B",
					Properties: map[string]MetamodelPropertyValue{
						"elementId": {Values: []MetamodelValue{{Kind: "string", String: "PART-A"}}},
					}},
			},
			Relationships: []SemanticRelationship{{
				ID: "TYPE-A", Kind: RelationshipTyping, Metaclass: "KerML::FeatureTyping",
				Source: "PART-A", Target: "PART-A",
				Properties: map[string]MetamodelPropertyValue{
					"typedFeature": {Values: []MetamodelValue{{Kind: "reference", Reference: "PART-A"}}},
					"type":         {Values: []MetamodelValue{{Kind: "reference", Reference: "PART-A"}}},
				},
			}},
		},
	}}
	canonical, err := NewCanonicalBundle(bundle)
	if err != nil {
		t.Fatal(err)
	}
	got := canonical.Semantic()
	var part *SemanticElement
	for index := range got.Elements {
		if got.Elements[index].ID == "PART-A" {
			part = &got.Elements[index]
			break
		}
	}
	if len(got.Imports) != 2 || part == nil || part.Owner != "PKG-B" || part.Namespace != "PKG-B" {
		t.Fatalf("ownership/import semantics changed: %+v", got)
	}
	if got.Relationships[0].Metaclass != "KerML::FeatureTyping" ||
		len(got.Relationships[0].Properties["typedFeature"].Values) != 1 ||
		len(got.Relationships[0].Properties["type"].Values) != 1 {
		t.Fatalf("relationship properties changed: %+v", got.Relationships[0])
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039, REQ-EMG-040
func TestEngineeringModelCompatibilityDTOsNormalizeToMetaclasses(t *testing.T) {
	bundle, err := LoadBundle("../engmod.yml")
	if err != nil {
		t.Fatal(err)
	}
	semantic, diagnostics := ProjectSemanticModel(bundle)
	if HasSemanticErrors(diagnostics) {
		t.Fatalf("projection failed: %+v", diagnostics)
	}
	for _, element := range semantic.Elements {
		if element.Metaclass == "" {
			t.Fatalf("element %s has no normalized metaclass", element.ID)
		}
		if _, official := OfficialSysMLMetaclasses[element.Metaclass]; !official &&
			!strings.HasPrefix(element.Metaclass, "Engineering::") {
			t.Fatalf("element %s has unstable extension metaclass %q", element.ID, element.Metaclass)
		}
	}
	for _, relationship := range semantic.Relationships {
		if relationship.Metaclass == "" {
			t.Fatalf("relationship %s has no normalized metaclass", relationship.ID)
		}
	}
}

// TRLC-LINKS: REQ-EMG-040
func TestEngineeringExtensionTypedValuesRoundTrip(t *testing.T) {
	integer := int64(7)
	flag := true
	element := SemanticElement{
		ID: "RISK-A", Kind: ElementExtension, Metaclass: "Engineering::Risk",
		ExtensionNamespace: "engineering", Extension: "engineering.risk", Targets: []string{"REQ-A"},
		Properties: map[string]MetamodelPropertyValue{
			"score": {Values: []MetamodelValue{{Kind: "integer", Integer: &integer}}},
			"owner": {Values: []MetamodelValue{{Kind: "reference", Reference: "TEAM-A"}}},
		},
		Metadata: []SemanticMetadata{{
			Namespace: "engineering", Type: "engineering.risk", Target: "RISK-A",
			Properties: map[string]MetamodelValue{
				"accepted": {Kind: "boolean", Boolean: &flag},
				"context": {Kind: "object", Object: map[string]MetamodelValue{
					"repository": {Kind: "string", String: "engineering-model-go"},
				}},
			},
		}},
	}
	data, err := yaml.Marshal(element)
	if err != nil {
		t.Fatal(err)
	}
	var decoded SemanticElement
	if err := yaml.Unmarshal(data, &decoded); err != nil {
		t.Fatal(err)
	}
	want, _ := json.Marshal(element)
	got, _ := json.Marshal(decoded)
	if string(want) != string(got) {
		t.Fatalf("typed extension did not round trip:\nwant %s\ngot  %s\nYAML:\n%s", want, got, data)
	}
	if !strings.HasPrefix(decoded.Metaclass, "Engineering::") {
		t.Fatalf("extension lost namespaced metaclass: %+v", decoded)
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func fixturePropertyValue(property OfficialSysMLProperty) MetamodelPropertyValue {
	count := property.Lower
	if count == 0 {
		count = 1
	}
	values := make([]MetamodelValue, count)
	for index := range values {
		switch metamodelValueKind(property.Kind, property.Type) {
		case "reference":
			values[index] = MetamodelValue{Kind: "reference", Reference: "FIXTURE"}
		case "boolean":
			value := true
			values[index] = MetamodelValue{Kind: "boolean", Boolean: &value}
		case "integer":
			value := int64(index + 1)
			values[index] = MetamodelValue{Kind: "integer", Integer: &value}
		case "real":
			value := float64(index + 1)
			values[index] = MetamodelValue{Kind: "real", Real: &value}
		default:
			values[index] = MetamodelValue{Kind: "string", String: property.ID}
		}
	}
	return MetamodelPropertyValue{Values: values, Derived: property.Derived}
}
