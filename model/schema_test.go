// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"cuelang.org/go/cue"
)

// TRLC-LINKS: REQ-EMG-041, REQ-EMG-043
func TestCanonicalCUESchemaRejectsUnsupportedNAFVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "architecture.yml")
	source := `model:
  id: test-model
  baseCatalogRef: ./catalog.yml
naf:
  framework: NAF
  version: "4.2"
  architectureDescription: Test
  stakeholders: []
  concerns: []
  products: []
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	var document ArchitectureDocument
	err := decodeYAMLFile(path, &document)
	if err == nil {
		t.Fatal("expected CUE validation to reject unsupported NAF version")
	}
	if !strings.Contains(err.Error(), "naf.version") {
		t.Fatalf("expected NAF version path, got: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046
func TestCanonicalCUESchemaRejectsUnsupportedDocumentSchemaVersion(t *testing.T) {
	path := filepath.Join(t.TempDir(), "architecture.yml")
	source := `schemaVersion: 2
model:
  id: test-model
  baseCatalogRef: ./catalog.yml
`
	if err := os.WriteFile(path, []byte(source), 0o644); err != nil {
		t.Fatal(err)
	}
	var document ArchitectureDocument
	err := decodeYAMLFile(path, &document)
	if err == nil {
		t.Fatal("expected CUE validation to reject unsupported schema version")
	}
	if !strings.Contains(err.Error(), "schemaVersion") {
		t.Fatalf("expected schemaVersion path, got: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-CANONICAL-SEMANTIC-MODEL
func TestCanonicalCUESchemaLoadsRootAndExamples(t *testing.T) {
	architectures := []string{
		filepath.Join("..", "architecture.yml"),
		filepath.Join("..", "examples", "bedrock-pr-review-github-app-sample", "architecture.yml"),
		filepath.Join("..", "examples", "coffee-fleet-ota-cloud-sample", "architecture.yml"),
		filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml"),
	}
	for _, architecture := range architectures {
		t.Run(architecture, func(t *testing.T) {
			if _, err := LoadBundle(architecture); err != nil {
				t.Fatalf("load CUE-validated bundle: %v", err)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-CANONICAL-SEMANTIC-MODEL
func TestCanonicalCUESchemaRejectsUnknownFieldWithPath(t *testing.T) {
	var document ArchitectureDocument
	err := decodeYAMLFile("testdata/invalid-unknown-architecture.yml", &document)
	if err == nil {
		t.Fatal("expected CUE validation to reject an unknown field")
	}
	if !strings.Contains(err.Error(), "model.unexpectedField") {
		t.Fatalf("expected path-aware error, got: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-CANONICAL-SEMANTIC-MODEL
func TestCanonicalCUESchemaRejectsInvalidMultiplicityBounds(t *testing.T) {
	var document ArchitectureDocument
	err := decodeYAMLFile("testdata/invalid-semantic-multiplicity.yml", &document)
	if err == nil {
		t.Fatal("expected CUE validation to reject upper multiplicity below lower")
	}
	if !strings.Contains(err.Error(), "semantics.elements.0.multiplicity.upper") {
		t.Fatalf("expected path-aware multiplicity error, got: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-035
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-VALIDATION-ENGINE, DO-CANONICAL-SEMANTIC-MODEL
func TestCanonicalCUESchemaMatchesGoRuntimeRepresentations(t *testing.T) {
	documents := []any{
		(*ArchitectureDocument)(nil),
		(*CatalogDocument)(nil),
		(*DecisionsDocument)(nil),
		(*RequirementsDocument)(nil),
		(*DesignDocument)(nil),
	}
	for _, document := range documents {
		schema, err := cueSchemaForDocument(document)
		if err != nil {
			t.Fatalf("load schema for %T: %v", document, err)
		}
		assertGoTypeMatchesCUESchema(t, reflect.TypeOf(document), schema, cueSchemaNameForTest(document), map[reflect.Type]bool{})
	}
}

// TRLC-LINKS: REQ-EMG-035
func cueSchemaNameForTest(document any) string {
	name, err := cueSchemaName(document)
	if err != nil {
		panic(err)
	}
	return "#" + name
}

// TRLC-LINKS: REQ-EMG-035
func assertGoTypeMatchesCUESchema(t *testing.T, goType reflect.Type, schema cue.Value, path string, visiting map[reflect.Type]bool) {
	t.Helper()
	for goType.Kind() == reflect.Pointer || goType.Kind() == reflect.Slice {
		if goType.Kind() == reflect.Slice {
			schema = schema.LookupPath(cue.MakePath(cue.AnyIndex))
		}
		goType = goType.Elem()
	}
	if goType.Kind() == reflect.Map || goType.Kind() == reflect.Interface || goType.Kind() != reflect.Struct {
		return
	}
	if visiting[goType] {
		return
	}
	visiting[goType] = true
	defer delete(visiting, goType)

	cueFields := make(map[string]cue.Value)
	fields, err := schema.Fields(cue.Optional(true))
	if err != nil {
		t.Fatalf("inspect CUE schema %s: %v", path, err)
	}
	for fields.Next() {
		cueFields[fields.Selector().Unquoted()] = fields.Value()
	}

	goFields := make(map[string]reflect.Type)
	for i := 0; i < goType.NumField(); i++ {
		field := goType.Field(i)
		name := strings.Split(field.Tag.Get("yaml"), ",")[0]
		if name == "" {
			name = field.Name
		}
		if name == "-" {
			continue
		}
		goFields[name] = field.Type
		cueField, ok := cueFields[name]
		if !ok {
			t.Errorf("%s.%s exists in Go but not CUE", path, name)
			continue
		}
		assertGoTypeMatchesCUESchema(t, field.Type, cueField, path+"."+name, visiting)
	}

	for name := range cueFields {
		if _, ok := goFields[name]; !ok {
			t.Errorf("%s.%s exists in CUE but not Go", path, name)
		}
	}
}
