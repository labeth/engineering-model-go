// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-063
func writeWorkspaceFixture(t *testing.T, path, module string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	content := "schemaVersion: 3\nreplacements:\n  - module: " + module + "\n    path: ./x\n"
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

// An engmod.work.yml found by the upward search that another user could
// have planted (world-writable, or owned by neither the current user nor
// root) is refused with an error naming ENGMOD_WORK; the same file given
// explicitly is used. A file of the current user is found as before.
// Mutation: dropping the trust check of the upward search fails this test.
// TRLC-LINKS: REQ-EMG-108
// ENGMODEL-LINKS: FU-MODEL-LOADER, FU-SYSTEM-COMPOSITION
func TestFindWorkspaceRefusesUntrustedUpwardFile(t *testing.T) {
	root := t.TempDir()
	work := filepath.Join(root, WorkspaceFileName)
	writeWorkspaceFixture(t, work, "example.com/parent@v0")
	project := filepath.Join(root, "repos", "app")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv(WorkspaceEnv, "")
	if _, path, err := FindWorkspace(project); err != nil || path != work {
		t.Fatalf("own file: path=%q err=%v", path, err)
	}
	if err := os.Chmod(work, 0o666); err != nil {
		t.Fatal(err)
	}
	if _, path, err := FindWorkspace(project); err == nil || path != "" || !strings.Contains(err.Error(), "world-writable") || !strings.Contains(err.Error(), WorkspaceEnv) {
		t.Fatalf("world-writable file: path=%q err=%v", path, err)
	}
	if _, path, err := FindWorkspaceWith(project, work); err != nil || path != work {
		t.Fatalf("explicit world-writable file: path=%q err=%v", path, err)
	}
	if err := os.Chmod(work, 0o644); err != nil {
		t.Fatal(err)
	}
	if !ownerCheckSupported {
		return
	}
	old := currentUID
	currentUID = func() int { return old() + 4242 }
	t.Cleanup(func() { currentUID = old })
	if _, path, err := FindWorkspace(project); err == nil || path != "" || !strings.Contains(err.Error(), "not owned") {
		t.Fatalf("file of another user: path=%q err=%v", path, err)
	}
}
