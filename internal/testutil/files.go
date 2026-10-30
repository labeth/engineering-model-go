// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION

// Package testutil holds test fixtures shared by the internal packages' tests.
// It is imported only from _test.go files.
package testutil

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"
)

// ChdirRepoRoot changes the working directory to the repository root (the
// directory holding go.mod above the current one), so package tests use the
// repository-relative paths (examples/..., engmod.yml, model/..., go run
// ./cmd/...) they used when the implementation lived in the root package.
// TRLC-LINKS: REQ-EMG-001
func ChdirRepoRoot() error {
	dir, err := os.Getwd()
	if err != nil {
		return err
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return os.Chdir(dir)
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return fmt.Errorf("no go.mod above %s", dir)
		}
		dir = parent
	}
}

// WriteFile writes content to path, creating its parent directories.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func WriteFile(t testing.TB, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
