// ENGMODEL-OWNER-UNIT: FU-LOBSTER-EXPORTER
package engmodel

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-006, REQ-EMG-012
func TestGenerateLobsterActivityTraceFromDir(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("mkdir tests dir: %v", err)
	}

	goFile := filepath.Join(testsDir, "foo_test.go")
	if err := os.WriteFile(goFile, []byte("package test\n\n// TRLC-LINKS: REQ-ABC-001, REQ-ABC-002\n"), 0o644); err != nil {
		t.Fatalf("write test file: %v", err)
	}

	res, err := GenerateLobsterActivityTraceFromDir(testsDir, LobsterActivityExportOptions{RequirementsPackage: "ExampleReqs", ActivityNamespace: "tests"})
	if err != nil {
		t.Fatalf("export lobster activity failed: %v", err)
	}

	var doc map[string]any
	if err := json.Unmarshal([]byte(res.JSON), &doc); err != nil {
		t.Fatalf("invalid json: %v", err)
	}
	if doc["schema"] != "lobster-act-trace" {
		t.Fatalf("expected lobster-act-trace schema")
	}
	data, ok := doc["data"].([]any)
	if !ok || len(data) != 1 {
		t.Fatalf("expected one activity item")
	}
	item, ok := data[0].(map[string]any)
	if !ok {
		t.Fatalf("expected object activity item")
	}
	if _, ok := item["tag"].(string); !ok {
		t.Fatalf("expected tag string")
	}
	refs, ok := item["refs"].([]any)
	if !ok || len(refs) != 2 {
		t.Fatalf("expected two refs")
	}
	location, ok := item["location"].(map[string]any)
	if !ok || location["line"] != float64(3) || location["column"] != float64(4) {
		t.Fatalf("expected marker source location 3:4, got %#v", item["location"])
	}
	if location["file"] != "foo_test.go" {
		t.Fatalf("expected portable relative source path, got %#v", location["file"])
	}
}

// TRLC-LINKS: REQ-EMG-006, REQ-EMG-012
func TestGenerateLobsterActivityTraceRejectsEmptySourceSet(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "model.yml"), []byte("schemaVersion: 1\n"), 0o644); err != nil {
		t.Fatalf("write source: %v", err)
	}
	_, err := GenerateLobsterActivityTraceFromDir(dir, LobsterActivityExportOptions{})
	if !errors.Is(err, ErrNoLobsterActivities) {
		t.Fatalf("expected ErrNoLobsterActivities, got %v", err)
	}
}
