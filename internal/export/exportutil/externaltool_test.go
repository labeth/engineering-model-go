// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package exportutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// writeFakeTool writes an executable stub named name in a new temp dir and
// returns the directory and the file path.
func writeFakeTool(t *testing.T, name string) (dir, tool string) {
	t.Helper()
	dir = t.TempDir()
	tool = filepath.Join(dir, name)
	if err := os.WriteFile(tool, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}
	return dir, tool
}

// wantErrContaining fails unless err is non-nil and mentions every fragment.
func wantErrContaining(t *testing.T, err error, fragments ...string) {
	t.Helper()
	if err == nil {
		t.Fatal("expected an error")
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("error %q does not contain %q", err, fragment)
		}
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestResolveExternalToolConfiguredPath(t *testing.T) {
	dir, tool := writeFakeTool(t, "tool")
	got, err := ResolveExternalTool(tool, "tool", "hint")
	if err != nil {
		t.Fatal(err)
	}
	if got != tool {
		t.Fatalf("got %q, want %q", got, tool)
	}

	_, err = ResolveExternalTool(dir, "tool", "hint")
	wantErrContaining(t, err, "is a directory")
	_, err = ResolveExternalTool(filepath.Join(dir, "missing"), "tool", "hint")
	wantErrContaining(t, err, "is unavailable")
}

// TRLC-LINKS: REQ-EMG-142
func TestResolveExternalToolPathLookupAndHint(t *testing.T) {
	dir, tool := writeFakeTool(t, "engmod-fake-tool")
	t.Setenv("PATH", dir)
	got, err := ResolveExternalTool("", "engmod-fake-tool", "hint")
	if err != nil {
		t.Fatal(err)
	}
	if got != tool {
		t.Fatalf("got %q, want %q", got, tool)
	}

	_, err = ResolveExternalTool("", "engmod-absent-tool", "install it with scripts/install.sh")
	wantErrContaining(t, err, "install it with scripts/install.sh", "engmod-absent-tool is required")
	_, err = ResolveExternalTool("  ", "engmod-absent-tool", "")
	wantErrContaining(t, err, "engmod-absent-tool is required")
	if strings.Contains(err.Error(), "; :") {
		t.Fatalf("absent tool without hint: got %v", err)
	}
}
