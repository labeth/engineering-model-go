package engmodel

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func TestGenerateLobsterActivityTraceFromDir(t *testing.T) {
	dir := t.TempDir()
	testsDir := filepath.Join(dir, "tests")
	if err := os.MkdirAll(testsDir, 0o755); err != nil {
		t.Fatalf("mkdir tests dir: %v", err)
	}
	goFile := filepath.Join(testsDir, "foo_test.go")
	if err := os.WriteFile(goFile, []byte("// TRLC-LINKS: REQ-ABC-001, REQ-ABC-002\npackage test\n"), 0o644); err != nil {
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
}
