// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// copyLinksExample copies examples/requirement-links (with engmod.work.yml,
// without generated output and locks) into a temporary directory.
func copyLinksExample(t *testing.T) string {
	t.Helper()
	t.Setenv(engmodel.WorkspaceEnv, "")
	t.Setenv(engmodel.LockEnv, "")
	t.Setenv(engmodel.ModuleCacheEnv, "")
	src := filepath.Join("..", "..", "examples", "requirement-links")
	root := t.TempDir()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			if d.Name() == "generated" || d.Name() == ".engmod" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(root, rel), 0o755)
		}
		if strings.HasPrefix(d.Name(), "engmod.lock") {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// linksServer initializes a server on repos/<repo> of a copied fixture.
func linksServer(t *testing.T, root, repo string) *Server {
	t.Helper()
	dir := filepath.Join(root, "repos", repo)
	s := NewServer()
	if resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath": filepath.Join(dir, "engmod.yml"), "repoRoot": dir,
		}},
	}); resp["error"] != nil {
		t.Fatalf("initialize %s: %+v", repo, resp["error"])
	}
	return s
}

// payloadRow returns the requirement row with id of a trace.matrix payload.
func payloadRow(t *testing.T, payload map[string]any, id string) map[string]any {
	t.Helper()
	for _, item := range payload["requirements"].([]any) {
		if row := item.(map[string]any); row["id"] == id {
			return row
		}
	}
	t.Fatalf("no row %s", id)
	return nil
}

// strs converts a JSON array to strings.
func strs(v any) []string {
	var out []string
	list, _ := v.([]any)
	for _, x := range list {
		s, _ := x.(string)
		out = append(out, s)
	}
	return out
}

// trace.matrix returns parents, children and links per row and the link
// counts in the summary, also with summaryOnly.
// Mutation: dropping parents or children from the row map, or a link
// counter from the summary map, fails this test.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTraceMatrixToolReportsRequirementLinks(t *testing.T) {
	root := copyLinksExample(t)
	s := linksServer(t, root, "system")
	for _, args := range []map[string]any{{}, {"summaryOnly": true}} {
		summary := callToolPayload(t, s, "trace.matrix", args)["summary"].(map[string]any)
		for _, key := range []string{"linksSuspect", "linksBroken", "linksClaimed", "unparented"} {
			if summary[key] != float64(0) {
				t.Fatalf("summary %v: %s = %v, want 0: %+v", args, key, summary[key], summary)
			}
		}
	}
	full := callToolPayload(t, s, "trace.matrix", map[string]any{})
	sys := payloadRow(t, full, "REQ-SYS-002")
	if got := strings.Join(strs(sys["children"]), ","); got != "encoder::CAP-A-2,encoder::REQ-A-003" {
		t.Fatalf("REQ-SYS-002 children %q: %+v", got, sys)
	}
	if links, _ := sys["links"].([]any); len(links) != 1 {
		t.Fatalf("REQ-SYS-002 links: %+v", sys)
	}
	if got := strs(payloadRow(t, full, "encoder::REQ-A-003")["parents"]); len(got) != 1 || got[0] != "encoder::CAP-A-2" {
		t.Fatalf("encoder::REQ-A-003 parents %v", got)
	}
}
