// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
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
	writeMCPFile(t, filepath.Join(root, "engmod.work.yml"), "schemaVersion: 3\nreplacements:\n  - module: example.com/child@v1\n    path: ./child\n")
	return parent
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func writeMCPModule(t *testing.T, dir, modulePath, version, modelID, manifestExtra, architecture string) {
	t.Helper()
	writeMCPFile(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \""+modulePath+"\"\nlanguage: version: \"v0.17.0\"\n")
	writeMCPFile(t, filepath.Join(dir, "engmod.yml"), `schemaVersion: 3
module:
  path: `+modulePath+`
  version: `+version+`
  modelId: `+modelID+`
  title: `+modelID+`
  introduction: ""
  kind: system
`+manifestExtra+testutil.CompleteSwitches+`documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
`)
	writeMCPFile(t, filepath.Join(dir, "model", "catalog.yml"), "schemaVersion: 3\ncatalog: {}\n")
	requirements := "schemaVersion: 3\nlintRun: {mode: guided}\nrequirements:\n  - id: REQ-PARENT\n    text: Parent requirement.\n  - id: REQ-CHILD\n    text: Child requirement.\n"
	if strings.Contains(architecture, "- id: CAP-CHILD\n") {
		requirements += "    realizes: [CAP-CHILD]\n" // the child requirement realizes its capability (provides.ref was removed)
	}
	writeMCPFile(t, filepath.Join(dir, "model", "requirements.yml"), requirements)
	writeMCPFile(t, filepath.Join(dir, "model", "architecture.yml"), "schemaVersion: 3\narchitecture:\n"+architecture)
	writeMCPFile(t, filepath.Join(dir, "model", "behavior.yml"), "schemaVersion: 3\nbehavior: {}\n")
	writeMCPFile(t, filepath.Join(dir, "model", "assurance.yml"), "schemaVersion: 3\nassurance: {}\n")
	writeMCPFile(t, filepath.Join(dir, "model", "compliance.yml"), "schemaVersion: 3\ncompliance: {}\n")
	writeMCPFile(t, filepath.Join(dir, "model", "views.yml"), "schemaVersion: 3\nviews: []\n")
	writeMCPFile(t, filepath.Join(dir, "model", "decisions.yml"), testutil.TierDecisionPoC)
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

// TestTraceMatrixToolSummaryAndSummaryOnly verifies trace.matrix reports the
// tested/testLinked/passed counts and that summaryOnly (boolean or string)
// drops the per-requirement rows, orphan list and dangling-link details while
// keeping the summary and diagnostics.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-061, REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTraceMatrixToolSummaryAndSummaryOnly(t *testing.T) {
	s := initializedPaymentsServer(t)
	full := callToolPayload(t, s, "trace.matrix", map[string]any{})
	summary, _ := full["summary"].(map[string]any)
	for _, key := range []string{"requirements", "implemented", "tested", "testLinked", "passed", "delegated", "orphan", "dangling"} {
		if _, ok := summary[key]; !ok {
			t.Fatalf("trace.matrix summary missing %q: %+v", key, summary)
		}
	}
	if summary["testLinked"] != summary["tested"] {
		t.Fatalf("testLinked must equal tested: %+v", summary)
	}
	if rows, _ := full["requirements"].([]any); len(rows) == 0 {
		t.Fatalf("full trace.matrix must list requirement rows")
	}
	for _, arg := range []any{true, "true"} {
		short := callToolPayload(t, s, "trace.matrix", map[string]any{"summaryOnly": arg})
		for _, key := range []string{"requirements", "orphans", "danglingLinks"} {
			if _, ok := short[key]; ok {
				t.Fatalf("summaryOnly=%v must omit %q: %+v", arg, key, short)
			}
		}
		if _, ok := short["diagnostics"]; !ok {
			t.Fatalf("summaryOnly=%v must keep diagnostics", arg)
		}
		if got, _ := short["summary"].(map[string]any); got["requirements"] != summary["requirements"] {
			t.Fatalf("summaryOnly=%v summary differs: %+v vs %+v", arg, got, summary)
		}
	}
	if rows, _ := callToolPayload(t, s, "trace.matrix", map[string]any{"summaryOnly": "false"})["requirements"].([]any); len(rows) == 0 {
		t.Fatalf("summaryOnly=false must keep requirement rows")
	}
	schema := inputSchemaForTool("trace.matrix")
	props, _ := schema["properties"].(map[string]any)
	if _, ok := props["summaryOnly"]; !ok {
		t.Fatalf("trace.matrix inputSchema must declare summaryOnly: %+v", schema)
	}
}

// The trace.matrix summary carries no verified and no linksSuspect counter
// (ADR-EMG-073). Mutation: serializing either counter again fails this test.
// TRLC-LINKS: REQ-EMG-182
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTraceMatrixToolSummaryHasNoRemovedCounters(t *testing.T) {
	s := initializedPaymentsServer(t)
	summary, _ := callToolPayload(t, s, "trace.matrix", map[string]any{"summaryOnly": true})["summary"].(map[string]any)
	if _, present := summary["verified"]; present {
		t.Fatalf("summary carries the removed counter verified: %+v", summary)
	}
	if _, present := summary["linksSuspect"]; present {
		t.Fatalf("summary carries the removed counter linksSuspect: %+v", summary)
	}
}
