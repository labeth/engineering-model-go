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

// workspaceProbe searches for the workspace file from one project directory
// and fails its test on an unexpected outcome.
type workspaceProbe struct {
	t       *testing.T
	project string
}

// at fails unless the search finds the workspace file at want without error.
func (p workspaceProbe) at(label, want string) WorkspaceDocument {
	p.t.Helper()
	ws, path, err := FindWorkspace(p.project)
	if err != nil || path != want {
		p.t.Fatalf("%s: path=%q err=%v", label, path, err)
	}
	return ws
}

// none fails unless the search finds no workspace.
func (p workspaceProbe) none(label string) {
	p.t.Helper()
	ws, path, err := FindWorkspace(p.project)
	if err != nil || path != "" || len(ws.Replacements) != 0 {
		p.t.Fatalf("%s: ws=%+v path=%q err=%v", label, ws, path, err)
	}
}

// refused fails unless the search is refused with an error that names every
// one of the fragments.
func (p workspaceProbe) refused(label string, fragments ...string) {
	p.t.Helper()
	_, path, err := FindWorkspace(p.project)
	if err == nil || path != "" {
		p.t.Fatalf("%s: path=%q err=%v", label, path, err)
	}
	for _, fragment := range fragments {
		if !strings.Contains(err.Error(), fragment) {
			p.t.Fatalf("%s: error %q does not name %q", label, err, fragment)
		}
	}
}

// explicit fails unless FindWorkspaceWith resolves mode to want.
func (p workspaceProbe) explicit(mode, want string) {
	p.t.Helper()
	if _, path, err := FindWorkspaceWith(p.project, mode); err != nil || path != want {
		p.t.Fatalf("FindWorkspaceWith %q: path=%q err=%v", mode, path, err)
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

	probe := workspaceProbe{t: t, project: project}
	t.Setenv(WorkspaceEnv, "")
	probe.at("unset", filepath.Join(root, WorkspaceFileName))

	t.Setenv(WorkspaceEnv, "off")
	probe.none("off")

	t.Setenv(WorkspaceEnv, explicit)
	ws := probe.at("path", explicit)
	if len(ws.Replacements) != 1 || ws.Replacements[0].Module != "example.com/explicit@v0" {
		t.Fatalf("path: ws=%+v", ws)
	}

	t.Setenv(WorkspaceEnv, filepath.Join(root, "missing.yml"))
	if _, _, err := FindWorkspace(project); err == nil {
		t.Fatal("expected a missing ENGMOD_WORK file to be an error")
	}

	// The explicit mode wins over the environment.
	probe.explicit("off", "")
	t.Setenv(WorkspaceEnv, "off")
	probe.explicit(explicit, explicit)
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
	probe := workspaceProbe{t: t, project: project}
	t.Setenv(WorkspaceEnv, "")
	probe.at("own file", work)
	if err := os.Chmod(work, 0o666); err != nil {
		t.Fatal(err)
	}
	probe.refused("world-writable file", "world-writable", WorkspaceEnv)
	probe.explicit(work, work)
	if err := os.Chmod(work, 0o644); err != nil {
		t.Fatal(err)
	}
	if !ownerCheckSupported {
		return
	}
	old := currentUID
	currentUID = func() int { return old() + 4242 }
	t.Cleanup(func() { currentUID = old })
	probe.refused("file of another user", "not owned")
}
