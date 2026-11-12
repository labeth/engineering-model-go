// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package exportutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-142
func TestResolveExternalToolConfiguredPath(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := ResolveExternalTool(tool, "tool", "hint")
	if err != nil {
		t.Fatal(err)
	}
	if got != tool {
		t.Fatalf("got %q, want %q", got, tool)
	}

	if _, err := ResolveExternalTool(dir, "tool", "hint"); err == nil || !strings.Contains(err.Error(), "is a directory") {
		t.Fatalf("directory: got %v", err)
	}
	if _, err := ResolveExternalTool(filepath.Join(dir, "missing"), "tool", "hint"); err == nil || !strings.Contains(err.Error(), "is unavailable") {
		t.Fatalf("missing file: got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestResolveExternalToolPathLookupAndHint(t *testing.T) {
	dir := t.TempDir()
	tool := filepath.Join(dir, "engmod-fake-tool")
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("PATH", dir)
	got, err := ResolveExternalTool("", "engmod-fake-tool", "hint")
	if err != nil {
		t.Fatal(err)
	}
	if got != tool {
		t.Fatalf("got %q, want %q", got, tool)
	}

	_, err = ResolveExternalTool("", "engmod-absent-tool", "install it with scripts/install.sh")
	if err == nil || !strings.Contains(err.Error(), "install it with scripts/install.sh") || !strings.Contains(err.Error(), "engmod-absent-tool is required") {
		t.Fatalf("absent tool: got %v", err)
	}
	_, err = ResolveExternalTool("  ", "engmod-absent-tool", "")
	if err == nil || strings.Contains(err.Error(), "; :") {
		t.Fatalf("absent tool without hint: got %v", err)
	}
}
