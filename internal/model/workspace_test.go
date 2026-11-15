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

// requireWorkspaceAt fails unless the search from project finds the workspace
// file at want without error.
func requireWorkspaceAt(t *testing.T, label, project, want string) WorkspaceDocument {
	t.Helper()
	ws, path, err := FindWorkspace(project)
	if err != nil || path != want {
		t.Fatalf("%s: path=%q err=%v", label, path, err)
	}
	return ws
}

// requireNoWorkspace fails unless the search from project finds no workspace.
func requireNoWorkspace(t *testing.T, label, project string) {
	t.Helper()
	ws, path, err := FindWorkspace(project)
	if err != nil || path != "" || len(ws.Replacements) != 0 {
		t.Fatalf("%s: ws=%+v path=%q err=%v", label, ws, path, err)
	}
}

// requireWorkspaceRefused fails unless the search from project is refused with
// an error that names every one of the given fragments.
func requireWorkspaceRefused(t *testing.T, label, project string, fragments ...string) {
	t.Helper()
	_, path, err := FindWorkspace(project)
	if err == nil || path != "" {
		t.Fatalf("%s: path=%q err=%v", label, path, err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			t.Fatalf("%s: error %q does not name %q", label, err, fragment)
		}
	}
}

// requireExplicitWorkspace fails unless FindWorkspaceWith resolves mode to want.
func requireExplicitWorkspace(t *testing.T, label, project, mode, want string) {
	t.Helper()
	if _, path, err := FindWorkspaceWith(project, mode); err != nil || path != want {
		t.Fatalf("%s: path=%q err=%v", label, path, err)
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
	requireWorkspaceAt(t, "unset", project, filepath.Join(root, WorkspaceFileName))

	t.Setenv(WorkspaceEnv, "off")
	requireNoWorkspace(t, "off", project)

	t.Setenv(WorkspaceEnv, explicit)
	ws := requireWorkspaceAt(t, "path", project, explicit)
	if len(ws.Replacements) != 1 || ws.Replacements[0].Module != "example.com/explicit@v0" {
		t.Fatalf("path: ws=%+v", ws)
	}

	t.Setenv(WorkspaceEnv, filepath.Join(root, "missing.yml"))
	if _, _, err := FindWorkspace(project); err == nil {
		t.Fatal("expected a missing ENGMOD_WORK file to be an error")
	}

	// The explicit mode wins over the environment.
	requireExplicitWorkspace(t, "FindWorkspaceWith off", project, "off", "")
	t.Setenv(WorkspaceEnv, "off")
	requireExplicitWorkspace(t, "FindWorkspaceWith path", project, explicit, explicit)
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
	requireWorkspaceAt(t, "own file", project, work)
	if err := os.Chmod(work, 0o666); err != nil {
		t.Fatal(err)
	}
	requireWorkspaceRefused(t, "world-writable file", project, "world-writable", WorkspaceEnv)
	requireExplicitWorkspace(t, "explicit world-writable file", project, work, work)
	if err := os.Chmod(work, 0o644); err != nil {
		t.Fatal(err)
	}
	if !ownerCheckSupported {
		return
	}
	old := currentUID
	currentUID = func() int { return old() + 4242 }
	t.Cleanup(func() { currentUID = old })
	requireWorkspaceRefused(t, "file of another user", project, "not owned")
}
