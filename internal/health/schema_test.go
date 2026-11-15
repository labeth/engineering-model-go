// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// The published schema document and the copy this package embeds are the same
// bytes.
//
// TRLC-LINKS: REQ-EMG-157
func TestPublishedSchemaMatchesEmbeddedCopy(t *testing.T) {
	published, err := os.ReadFile(filepath.Join("..", "..", "docs", "health-report.schema.json"))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(published, ReportSchemaJSON()) {
		t.Fatal("docs/health-report.schema.json differs from internal/health/health-report.schema.json; copy one over the other")
	}
	if !json.Valid(published) {
		t.Fatal("the schema is not valid JSON")
	}
}

// sampleDocument is the generic JSON form of the sample report.
func sampleDocument(t *testing.T) map[string]any {
	t.Helper()
	data, err := Encode(BuildReport(sampleResults(), sampleOptions()))
	if err != nil {
		t.Fatal(err)
	}
	if err := ValidateReportJSON(data); err != nil {
		t.Fatalf("a generated report does not validate: %v", err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatal(err)
	}
	return doc
}

// first descends to the first element of a list under keys.
func at(t *testing.T, doc any, keys ...any) any {
	t.Helper()
	for _, k := range keys {
		switch key := k.(type) {
		case string:
			doc = doc.(map[string]any)[key]
		case int:
			doc = doc.([]any)[key]
		}
	}
	return doc
}

// A real generated report, an empty report and one with every optional block
// validate; broken variants are rejected with the schema as the reason.
//
// Mutation: loosening the schema (dropping a required list, the band enum, the
// score range, the path pattern, additionalProperties) lets one of these
// through and fails this test.
//
// TRLC-LINKS: REQ-EMG-157
func TestReportSchemaAcceptsGeneratedAndRejectsBroken(t *testing.T) {
	empty, _ := Encode(BuildReport(nil, Options{EngineName: "x", EngineVersion: "1"}))
	wantEqual(t, ValidateReportJSON(empty), nil) // an empty report validates
	for _, c := range schemaMutations(t) {
		doc := sampleDocument(t)
		c.mutate(doc)
		data, err := json.Marshal(doc)
		mustEqual(t, err, nil)
		err = ValidateReportJSON(data)
		mustEqual(t, [2]any{c.name, err != nil}, [2]any{c.name, true}) // the broken report is rejected
		wantContains(t, err.Error(), "engmod-health/1")                // with the schema as the reason
	}
	wantEqual(t, ValidateReportJSON([]byte("[1,2")) != nil, true) // a document that is not JSON
}

// fileOf is the second file of a document.
func fileOf(t *testing.T, doc map[string]any) map[string]any {
	return at(t, doc, "files", 1).(map[string]any)
}

// schemaMutation is a change of the sample document that the schema rejects.
type schemaMutation struct {
	name   string
	mutate func(doc map[string]any)
}

// schemaMutations are the broken variants of the sample document.
func schemaMutations(t *testing.T) []schemaMutation {
	return []schemaMutation{
		{"another schema string", func(doc map[string]any) { doc["schema"] = "engmod-health/2" }},
		{"missing engine", func(doc map[string]any) { delete(doc, "engine") }},
		{"missing rollups", func(doc map[string]any) { delete(doc, "rollups") }},
		{"missing files", func(doc map[string]any) { delete(doc, "files") }},
		{"unknown top-level field", func(doc map[string]any) { doc["generatedAt"] = "2026-01-01" }},
		{"score above 10", func(doc map[string]any) { fileOf(t, doc)["score"] = 10.5 }},
		{"score below 1", func(doc map[string]any) { fileOf(t, doc)["score"] = 0.5 }},
		{"unknown band", func(doc map[string]any) { fileOf(t, doc)["band"] = "purple" }},
		{"absolute path", func(doc map[string]any) { fileOf(t, doc)["path"] = "/etc/passwd" }},
		{"backslash path", func(doc map[string]any) { fileOf(t, doc)["path"] = `a\b.go` }},
		{"missing functions", func(doc map[string]any) { delete(fileOf(t, doc), "functions") }},
		{"unknown code", func(doc map[string]any) {
			at(t, doc, "files", 1, "functions", 0, "findings", 0).(map[string]any)["code"] = "health.other"
		}},
		{"unknown severity", func(doc map[string]any) {
			at(t, doc, "files", 1, "functions", 0, "findings", 0).(map[string]any)["severity"] = "fatal"
		}},
		{"finding without metric", func(doc map[string]any) {
			delete(at(t, doc, "files", 1, "functions", 0, "findings", 0).(map[string]any), "metric")
		}},
		{"unknown finding field", func(doc map[string]any) {
			at(t, doc, "files", 1, "functions", 0, "findings", 0).(map[string]any)["extra"] = 1
		}},
		{"negative loc", func(doc map[string]any) { fileOf(t, doc)["loc"] = -1 }},
		{"negative syntax errors", func(doc map[string]any) { fileOf(t, doc)["syntaxErrors"] = -1 }},
		{"unknown tier", func(doc map[string]any) { doc["tier"] = "gold" }},
		{"unknown level", func(doc map[string]any) { doc["levels"].(map[string]any)[CodeBumpyRoad] = "fatal" }},
		{"project band", func(doc map[string]any) { at(t, doc, "rollups", "project").(map[string]any)["band"] = "blue" }},
		{"unknown project field", func(doc map[string]any) { at(t, doc, "rollups", "project").(map[string]any)["median"] = 5 }},
		{"waived without decision", func(doc map[string]any) { delete(at(t, doc, "waived", 0).(map[string]any), "decision") }},
		{"waived with unknown field", func(doc map[string]any) { at(t, doc, "waived", 0).(map[string]any)["extra"] = true }},
		{"diagnostic level", func(doc map[string]any) { at(t, doc, "diagnostics", 0).(map[string]any)["level"] = "info" }},
	}
}
