// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

// A model-reading tool takes an optional repo argument: a checkout inside
// the workspace root, the directory of the trusted engmod.work.yml found
// above the server's own model, re-resolved on every call. So a server
// started before engws sync wrote engmod.work.yml answers for a sibling as
// soon as the file exists, and one server answers for every member of a
// family. repo is relative to the workspace root (or absolute inside it),
// must hold engmod.yml and resolve inside the root after symlinks; without
// a workspace only the server's own checkout is allowed; a world-writable
// workspace file is refused like the upward search refuses it. Every
// model-reading tool lists repo in its strict input schema.
// Mutation: resolving the workspace once at startup, skipping the
// confinement or the trust check, or ignoring repo fails this test.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func TestRepoArgumentAnswersForFamilyMembers(t *testing.T) {
	f := newRepoArgFixture(t)
	f.withoutWorkspace(t)
	// engws sync writes the workspace after the server started.
	if err := os.WriteFile(f.work, f.workData, 0o644); err != nil {
		t.Fatal(err)
	}
	f.answersPerMember(t)
	f.defaultsToOwnCheckout(t)
	f.rejectsEscapes(t)
	f.refusesWorldWritableWorkspace(t)
	requireRepoInSchemas(t)
}

// repoArgFixture is the links example family with its workspace file removed
// and a server started on the system checkout.
type repoArgFixture struct {
	s        *Server
	root     string
	work     string
	workData []byte
}

// newRepoArgFixture copies the family, removes engmod.work.yml (remembering
// its content) and starts a server before the workspace exists.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func newRepoArgFixture(t *testing.T) *repoArgFixture {
	t.Helper()
	root := copyLinksExample(t)
	work := filepath.Join(root, "engmod.work.yml")
	workData, err := os.ReadFile(work)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(work); err != nil {
		t.Fatal(err)
	}
	return &repoArgFixture{s: linksServer(t, root, "system"), root: root, work: work, workData: workData}
}

// requirementCount returns the trace.matrix requirements count for the args.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (f *repoArgFixture) requirementCount(t *testing.T, args map[string]any) any {
	t.Helper()
	return f.s.callToolPayload(t, "trace.matrix", args)["summary"].(map[string]any)["requirements"]
}

// withoutWorkspace allows only the server's own checkout.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (f *repoArgFixture) withoutWorkspace(t *testing.T) {
	t.Helper()
	own := f.requirementCount(t, map[string]any{})
	if got := f.requirementCount(t, map[string]any{"repo": "."}); got != own {
		t.Fatalf("repo . without a workspace is the server's own checkout: %v, want %v", got, own)
	}
	if msg := f.s.toolError(t, "trace.matrix", map[string]any{"repo": "../child-a"}); !strings.Contains(msg, "outside") {
		t.Fatalf("a sibling without a workspace: %q", msg)
	}
}

// answersPerMember answers for each family member named by repo.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (f *repoArgFixture) answersPerMember(t *testing.T) {
	t.Helper()
	if got := f.requirementCount(t, map[string]any{"repo": "repos/child-a"}); got != float64(4) {
		t.Fatalf("trace.matrix for repos/child-a: %v requirements, want 4", got)
	}
	if got := f.s.callToolPayload(t, "requirements.get", map[string]any{"repo": "repos/child-a", "requirementId": "REQ-A-001"}); got["exists"] != true {
		t.Fatalf("requirements.get in child-a: %+v", got)
	}
	if links, _ := f.s.callToolPayload(t, "links.list", map[string]any{"repo": filepath.Join(f.root, "repos", "child-b")})["links"].([]any); len(links) != 0 {
		t.Fatalf("links.list in child-b: %+v", links)
	}
}

// defaultsToOwnCheckout answers for the server's own checkout without repo
// and diagnoses a document of the named member.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (f *repoArgFixture) defaultsToOwnCheckout(t *testing.T) {
	t.Helper()
	if got, sys := f.requirementCount(t, map[string]any{}), f.requirementCount(t, map[string]any{"repo": "repos/system"}); got != sys || got == float64(4) {
		t.Fatalf("without repo the server answers for its own checkout: %v, repos/system %v", got, sys)
	}
	diag := f.s.callToolPayload(t, "model.diagnose", map[string]any{"repo": "repos/child-a", "document": "requirements", "content": "schemaVersion: 3\nrequirements: [\n"})
	if diag["valid"] != false || diag["path"] != "model/requirements.yml" {
		t.Fatalf("model.diagnose in child-a: %+v", diag)
	}
}

// rejectsEscapes refuses a valid model outside the workspace, reachable
// through a symlink, and the other repo values that leave the root.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (f *repoArgFixture) rejectsEscapes(t *testing.T) {
	t.Helper()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.Rename(filepath.Join(f.root, "repos", "child-b"), outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(f.root, "repos", "child-b")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(f.root, "repos", "escape")); err != nil {
		t.Fatal(err)
	}
	for name, repo := range map[string]string{"dot-dot": "../", "absolute outside": outside, "symlink": "repos/escape", "symlinked member": "repos/child-b", "no model": "repos"} {
		if msg := f.s.toolError(t, "links.list", map[string]any{"repo": repo}); msg == "" {
			t.Errorf("%s: repo %q accepted", name, repo)
		}
	}
}

// refusesWorldWritableWorkspace refuses a world-writable workspace file like
// the upward search does.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func (f *repoArgFixture) refusesWorldWritableWorkspace(t *testing.T) {
	t.Helper()
	if runtime.GOOS == "windows" {
		return
	}
	if err := os.Chmod(f.work, 0o666); err != nil {
		t.Fatal(err)
	}
	if msg := f.s.toolError(t, "trace.matrix", map[string]any{"repo": "repos/child-a"}); !strings.Contains(msg, "world-writable") {
		t.Fatalf("an untrusted workspace: %q", msg)
	}
}

// requireRepoInSchemas requires every model-reading tool to list repo in its
// input schema.
// TRLC-LINKS: REQ-EMG-120
// ENGMODEL-LINKS: FU-MCP-SERVER, CTRL-MCP-PATH-BOUNDARY
func requireRepoInSchemas(t *testing.T) {
	t.Helper()
	for _, tool := range []string{"model.diagnose", "model.patch", "trace.matrix", "links.list", "links.suspect", "links.review", "requirements.get", "requirements.parents", "gemara.validate", "gemara.controlCatalog", "model.list", "composition.resolve"} {
		props, _ := inputSchemaForTool(tool)["properties"].(map[string]any)
		if _, ok := props["repo"]; !ok {
			t.Errorf("%s input schema lacks repo", tool)
		}
	}
}
