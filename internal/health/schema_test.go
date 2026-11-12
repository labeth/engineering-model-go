// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
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
	doc := sampleDocument(t)
	if empty, _ := Encode(BuildReport(nil, Options{EngineName: "x", EngineVersion: "1"})); ValidateReportJSON(empty) != nil {
		t.Errorf("an empty report is rejected: %v", ValidateReportJSON(empty))
	}

	file := func() map[string]any { return at(t, doc, "files", 1).(map[string]any) }
	cases := []struct {
		name   string
		mutate func()
	}{
		{"another schema string", func() { doc["schema"] = "engmod-health/2" }},
		{"missing engine", func() { delete(doc, "engine") }},
		{"missing rollups", func() { delete(doc, "rollups") }},
		{"missing files", func() { delete(doc, "files") }},
		{"unknown top-level field", func() { doc["generatedAt"] = "2026-01-01" }},
		{"score above 10", func() { file()["score"] = 10.5 }},
		{"score below 1", func() { file()["score"] = 0.5 }},
		{"unknown band", func() { file()["band"] = "purple" }},
		{"absolute path", func() { file()["path"] = "/etc/passwd" }},
		{"backslash path", func() { file()["path"] = `a\b.go` }},
		{"missing functions", func() { delete(file(), "functions") }},
		{"unknown code", func() {
			at(t, doc, "files", 1, "functions", 0, "findings", 0).(map[string]any)["code"] = "health.other"
		}},
		{"unknown severity", func() {
			at(t, doc, "files", 1, "functions", 0, "findings", 0).(map[string]any)["severity"] = "fatal"
		}},
		{"finding without metric", func() { delete(at(t, doc, "files", 1, "functions", 0, "findings", 0).(map[string]any), "metric") }},
		{"unknown finding field", func() { at(t, doc, "files", 1, "functions", 0, "findings", 0).(map[string]any)["extra"] = 1 }},
		{"negative loc", func() { file()["loc"] = -1 }},
		{"negative syntax errors", func() { file()["syntaxErrors"] = -1 }},
		{"unknown tier", func() { doc["tier"] = "gold" }},
		{"unknown level", func() { doc["levels"].(map[string]any)[CodeBumpyRoad] = "fatal" }},
		{"project band", func() { at(t, doc, "rollups", "project").(map[string]any)["band"] = "blue" }},
		{"unknown project field", func() { at(t, doc, "rollups", "project").(map[string]any)["median"] = 5 }},
		{"waived without decision", func() { delete(at(t, doc, "waived", 0).(map[string]any), "decision") }},
		{"waived with unknown field", func() { at(t, doc, "waived", 0).(map[string]any)["extra"] = true }},
		{"diagnostic level", func() { at(t, doc, "diagnostics", 0).(map[string]any)["level"] = "info" }},
	}
	for _, c := range cases {
		doc = sampleDocument(t)
		c.mutate()
		data, err := json.Marshal(doc)
		if err != nil {
			t.Fatal(err)
		}
		if err := ValidateReportJSON(data); err == nil {
			t.Errorf("%s: the broken report validates", c.name)
		} else if !strings.Contains(err.Error(), "engmod-health/1") {
			t.Errorf("%s: error %q does not name the schema", c.name, err)
		}
	}
	if err := ValidateReportJSON([]byte("[1,2")); err == nil {
		t.Error("a document that is not JSON validates")
	}
}
