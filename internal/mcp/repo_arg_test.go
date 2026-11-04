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
	root := copyLinksExample(t)
	work := filepath.Join(root, "engmod.work.yml")
	workData, err := os.ReadFile(work)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(work); err != nil {
		t.Fatal(err)
	}
	s := linksServer(t, root, "system")
	requirementCount := func(args map[string]any) any {
		return callToolPayload(t, s, "trace.matrix", args)["summary"].(map[string]any)["requirements"]
	}
	own := requirementCount(map[string]any{})
	if got := requirementCount(map[string]any{"repo": "."}); got != own {
		t.Fatalf("repo . without a workspace is the server's own checkout: %v, want %v", got, own)
	}
	if msg := toolError(t, s, "trace.matrix", map[string]any{"repo": "../child-a"}); !strings.Contains(msg, "outside") {
		t.Fatalf("a sibling without a workspace: %q", msg)
	}

	// engws sync writes the workspace after the server started.
	if err := os.WriteFile(work, workData, 0o644); err != nil {
		t.Fatal(err)
	}
	if got := requirementCount(map[string]any{"repo": "repos/child-a"}); got != float64(4) {
		t.Fatalf("trace.matrix for repos/child-a: %v requirements, want 4", got)
	}
	if got := callToolPayload(t, s, "requirements.get", map[string]any{"repo": "repos/child-a", "requirementId": "REQ-A-001"}); got["exists"] != true {
		t.Fatalf("requirements.get in child-a: %+v", got)
	}
	if links, _ := callToolPayload(t, s, "links.list", map[string]any{"repo": filepath.Join(root, "repos", "child-b")})["links"].([]any); len(links) != 0 {
		t.Fatalf("links.list in child-b: %+v", links)
	}
	if got, sys := requirementCount(map[string]any{}), requirementCount(map[string]any{"repo": "repos/system"}); got != sys || got == float64(4) {
		t.Fatalf("without repo the server answers for its own checkout: %v, repos/system %v", got, sys)
	}
	diag := callToolPayload(t, s, "model.diagnose", map[string]any{"repo": "repos/child-a", "document": "requirements", "content": "schemaVersion: 2\nrequirements: [\n"})
	if diag["valid"] != false || diag["path"] != "model/requirements.yml" {
		t.Fatalf("model.diagnose in child-a: %+v", diag)
	}

	// A valid model outside the workspace, reachable through a symlink.
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.Rename(filepath.Join(root, "repos", "child-b"), outside); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "repos", "child-b")); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(root, "repos", "escape")); err != nil {
		t.Fatal(err)
	}
	for name, repo := range map[string]string{"dot-dot": "../", "absolute outside": outside, "symlink": "repos/escape", "symlinked member": "repos/child-b", "no model": "repos"} {
		if msg := toolError(t, s, "links.list", map[string]any{"repo": repo}); msg == "" {
			t.Errorf("%s: repo %q accepted", name, repo)
		}
	}
	if runtime.GOOS != "windows" {
		if err := os.Chmod(work, 0o666); err != nil {
			t.Fatal(err)
		}
		if msg := toolError(t, s, "trace.matrix", map[string]any{"repo": "repos/child-a"}); !strings.Contains(msg, "world-writable") {
			t.Fatalf("an untrusted workspace: %q", msg)
		}
	}

	for _, tool := range []string{"model.diagnose", "model.patch", "trace.matrix", "links.list", "links.suspect", "links.review", "requirements.get", "requirements.parents", "gemara.validate", "gemara.controlCatalog", "model.list", "composition.resolve"} {
		props, _ := inputSchemaForTool(tool)["properties"].(map[string]any)
		if _, ok := props["repo"]; !ok {
			t.Errorf("%s input schema lacks repo", tool)
		}
	}
}
