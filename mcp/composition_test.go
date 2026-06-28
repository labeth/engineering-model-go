// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

// TestCompositionAndTraceMatrixTools verifies the composition.resolve and trace.matrix
// MCP tools return real system-of-systems and traceability data for a composed model.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-030
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestCompositionAndTraceMatrixTools(t *testing.T) {
	s := NewServer()
	base := filepath.Join("..", "examples", "coffee-fleet-ota-cloud-sample")
	initResp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath":        filepath.Join(base, "architecture.yml"),
			"requirementsPath": filepath.Join(base, "requirements.yml"),
			"designPath":       filepath.Join(base, "design.yml"),
			"repoRoot":         filepath.Join(".."),
		}},
	})
	if initResp["error"] != nil {
		t.Fatalf("initialize: %+v", initResp["error"])
	}

	comp := callToolData(t, s, "composition.resolve")
	if comp["hasComposition"] != true {
		t.Fatalf("composition.resolve expected hasComposition true: %+v", comp)
	}
	if subs, _ := comp["subsystems"].([]any); len(subs) != 3 {
		t.Fatalf("composition.resolve expected 3 subsystems, got %d", len(subs))
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
