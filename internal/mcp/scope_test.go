// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// The MCP code tools scan what the check scans: a git-ignored build
// directory holding unmarked JavaScript, and a directory left out with an
// exclude glob, do not make coverage.strictStatus fail or show up in
// files.forRequirement; a tracked unmarked function still fails.
// Mutation: scanning repoRoot without the code scope (codemap.Scan), or
// walking the file index without it, fails this test.
// TRLC-LINKS: REQ-EMG-114
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func TestMCPScanHonoursGitignoreAndExclude(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not installed")
	}
	root := t.TempDir()
	writeMCPModule(t, root, "example.com/scope@v1", "v1.0.0", "SCOPE", "inferenceHints:\n  exclude:\n    - vendor-copy/*\n", "  functionalUnits: []\n")
	writeMCPFile(t, filepath.Join(root, "model", "requirements.yml"), "schemaVersion: 2\nlintRun: {}\nrequirements:\n  - id: REQ-SCOPE-001\n    text: Scope requirement.\n")
	writeMCPFile(t, filepath.Join(root, ".gitignore"), "viewer/dist/\n")
	writeMCPFile(t, filepath.Join(root, "viewer", "dist", "assets", "app.js"), "// REQ-SCOPE-001 bundled\nfunction bundled() { return 1; }\n")
	writeMCPFile(t, filepath.Join(root, "vendor-copy", "lib", "x.js"), "// REQ-SCOPE-001 copied\nfunction copied() { return 1; }\n")
	writeMCPFile(t, filepath.Join(root, "src", "main.js"), "// TRLC-LINKS: REQ-SCOPE-001\nfunction main() { return 1; }\n")
	if out, err := exec.Command("git", "-C", root, "init", "-q").CombinedOutput(); err != nil {
		t.Fatalf("git init: %v %s", err, out)
	}
	s := initializedServerAt(t, root)
	status := callToolPayload(t, s, "coverage.strictStatus", map[string]any{})
	if status["status"] != "pass" {
		t.Fatalf("ignored and excluded files must not fail coverage: %+v", status)
	}
	files := callToolPayload(t, s, "files.forRequirement", map[string]any{"requirementId": "REQ-SCOPE-001"})
	got := strings.Join(strs(files["files"]), " ")
	if !strings.Contains(got, filepath.Join("src", "main.js")) || strings.Contains(got, "dist") || strings.Contains(got, "vendor-copy") {
		t.Fatalf("files.forRequirement must list only in-scope files: %v", got)
	}

	writeMCPFile(t, filepath.Join(root, "src", "extra.js"), "function unmarked() { return 2; }\n")
	s = initializedServerAt(t, root)
	if status := callToolPayload(t, s, "coverage.strictStatus", map[string]any{}); status["status"] != "fail" {
		t.Fatalf("an unmarked in-scope function must still fail coverage: %+v", status)
	}
}
