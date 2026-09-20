// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestCompositionAndTraceMatrixTools verifies the composition.resolve and trace.matrix
// MCP tools return real system-of-systems and traceability data for a composed model.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-030
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestCompositionAndTraceMatrixTools(t *testing.T) {
	s := NewServer()
	base := writeMCPCompositionFixture(t)
	initResp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath":        filepath.Join(base, "engmod.yml"),
			"requirementsPath": filepath.Join(base, "model", "requirements.yml"),
			"designPath":       filepath.Join(base, "model", "views.yml"),
			"repoRoot":         base,
		}},
	})
	if initResp["error"] != nil {
		t.Fatalf("initialize: %+v", initResp["error"])
	}

	comp := callToolData(t, s, "composition.resolve")
	if comp["hasComposition"] != true {
		t.Fatalf("composition.resolve expected hasComposition true: %+v", comp)
	}
	if subs, _ := comp["subsystems"].([]any); len(subs) != 1 {
		t.Fatalf("composition.resolve expected 1 subsystem, got %d", len(subs))
	}

	if allocs, _ := comp["allocations"].([]any); len(allocs) == 0 {
		t.Fatal("composition.resolve expected allocations")
	}

	tm := callToolData(t, s, "trace.matrix")
	summary, ok := tm["summary"].(map[string]any)
	if !ok {
		t.Fatalf("trace.matrix missing summary: %+v", tm)
	}
	if summary["requirements"] == nil {
		t.Fatal("trace.matrix summary missing requirements count")
	}
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func writeMCPCompositionFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	child := filepath.Join(root, "child")
	parent := filepath.Join(root, "parent")
	writeMCPModule(t, child, "example.com/child@v1", "v1.0.0", "CHILD", `publications:
  - id: public
    architecture: [CAP-CHILD]
`, `  contract:
    provides:
      - id: CAP-CHILD
        kind: capability
        ref: REQ-CHILD
`)
	writeMCPModule(t, parent, "example.com/parent@v1", "v1.0.0", "PARENT", `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.0.0
    publications: [public]
`, `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
    allocations:
      - requirement: REQ-PARENT
        to: SUB-CHILD
        target: child::CAP-CHILD
`)
	writeMCPFile(t, filepath.Join(root, "engmod.work.yml"), "schemaVersion: 2\nreplacements:\n  - module: example.com/child@v1\n    path: ./child\n")
	return parent
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func writeMCPModule(t *testing.T, dir, modulePath, version, modelID, manifestExtra, architecture string) {
	t.Helper()
	writeMCPFile(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \""+modulePath+"\"\nlanguage: version: \"v0.17.0\"\n")
	writeMCPFile(t, filepath.Join(dir, "engmod.yml"), `schemaVersion: 2
module:
  path: `+modulePath+`
  version: `+version+`
  modelId: `+modelID+`
  title: `+modelID+`
  introduction: ""
  kind: system
`+manifestExtra+`documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
`)
	writeMCPFile(t, filepath.Join(dir, "model", "catalog.yml"), "schemaVersion: 2\ncatalog: {}\n")
	writeMCPFile(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 2\nlintRun: {}\nrequirements:\n  - id: REQ-PARENT\n    text: Parent requirement.\n  - id: REQ-CHILD\n    text: Child requirement.\n")
	writeMCPFile(t, filepath.Join(dir, "model", "architecture.yml"), "schemaVersion: 2\narchitecture:\n"+architecture)
	writeMCPFile(t, filepath.Join(dir, "model", "behavior.yml"), "schemaVersion: 2\nbehavior: {}\n")
	writeMCPFile(t, filepath.Join(dir, "model", "assurance.yml"), "schemaVersion: 2\nassurance: {}\n")
	writeMCPFile(t, filepath.Join(dir, "model", "compliance.yml"), "schemaVersion: 2\ncompliance: {}\n")
	writeMCPFile(t, filepath.Join(dir, "model", "views.yml"), "schemaVersion: 2\nviews: []\n")
	writeMCPFile(t, filepath.Join(dir, "model", "decisions.yml"), "schemaVersion: 2\ndecisions: []\n")
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func writeMCPFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// callToolData calls a no-argument MCP tool and returns its parsed JSON payload.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER
func callToolData(t *testing.T, s *Server, name string) map[string]any {
	t.Helper()
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 99, "method": "tools/call",
		"params": map[string]any{"name": name, "arguments": map[string]any{}},
	})
	if resp["error"] != nil {
		t.Fatalf("tool %s error: %+v", name, resp["error"])
	}
	result, _ := resp["result"].(map[string]any)
	content, _ := result["content"].([]any)
	if len(content) == 0 {
		t.Fatalf("tool %s missing content", name)
	}
	chunk, _ := content[0].(map[string]any)
	text, _ := chunk["text"].(string)
	var data map[string]any
	if err := json.Unmarshal([]byte(text), &data); err != nil {
		t.Fatalf("tool %s content not json: %v", name, err)
	}
	return data
}
