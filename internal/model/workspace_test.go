// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"os"
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-063
func writeWorkspaceFixture(t *testing.T, path, module string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "schemaVersion: 2\nreplacements:\n  - module: " + module + "\n    path: ./x\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// TestFindWorkspaceHonoursEngmodWork verifies ENGMOD_WORK=off ignores a parent
// workspace, ENGMOD_WORK=<path> uses that file without the parent search, a
// missing path is an error, and the programmatic mode overrides the variable.
// TRLC-LINKS: REQ-EMG-063
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
func TestFindWorkspaceHonoursEngmodWork(t *testing.T) {
	root := t.TempDir()
	writeWorkspaceFixture(t, filepath.Join(root, WorkspaceFileName), "example.com/parent@v0")
	explicit := filepath.Join(root, "elsewhere", "ws.yml")
	writeWorkspaceFixture(t, explicit, "example.com/explicit@v0")
	project := filepath.Join(root, "repos", "app")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}

	t.Setenv(WorkspaceEnv, "")
	if _, path, err := FindWorkspace(project); err != nil || path != filepath.Join(root, WorkspaceFileName) {
		t.Fatalf("unset: path=%q err=%v", path, err)
	}

	t.Setenv(WorkspaceEnv, "off")
	if ws, path, err := FindWorkspace(project); err != nil || path != "" || len(ws.Replacements) != 0 {
		t.Fatalf("off: ws=%+v path=%q err=%v", ws, path, err)
	}

	t.Setenv(WorkspaceEnv, explicit)
	ws, path, err := FindWorkspace(project)
	if err != nil || path != explicit || len(ws.Replacements) != 1 || ws.Replacements[0].Module != "example.com/explicit@v0" {
		t.Fatalf("path: ws=%+v path=%q err=%v", ws, path, err)
	}

	t.Setenv(WorkspaceEnv, filepath.Join(root, "missing.yml"))
	if _, _, err := FindWorkspace(project); err == nil {
		t.Fatal("expected a missing ENGMOD_WORK file to be an error")
	}

	// The explicit mode wins over the environment.
	if _, path, err := FindWorkspaceWith(project, "off"); err != nil || path != "" {
		t.Fatalf("FindWorkspaceWith off: path=%q err=%v", path, err)
	}
	t.Setenv(WorkspaceEnv, "off")
	if _, path, err := FindWorkspaceWith(project, explicit); err != nil || path != explicit {
		t.Fatalf("FindWorkspaceWith path: path=%q err=%v", path, err)
	}
}
