// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"path/filepath"
	"testing"
)

// trace.matrix reports each delegated row's delegatedStatus and the summary's
// delegatedImplemented and delegatedTestLinked counts, also with summaryOnly.
// Mutation: dropping delegatedStatus from the row map, or the two summary
// counters from the summary map, fails this test.
// TRLC-LINKS: REQ-EMG-067, REQ-EMG-061
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTraceMatrixToolReportsDelegatedStatus(t *testing.T) {
	base := writeMCPCompositionFixture(t)
	s := NewServer()
	if resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath":        filepath.Join(base, "engmod.yml"),
			"requirementsPath": filepath.Join(base, "model", "requirements.yml"),
			"repoRoot":         base,
		}},
	}); resp["error"] != nil {
		t.Fatalf("initialize: %+v", resp["error"])
	}
	full := callToolPayload(t, s, "trace.matrix", map[string]any{})
	var delegated map[string]any
	for _, item := range full["requirements"].([]any) {
		row := item.(map[string]any)
		if row["id"] == "REQ-PARENT" {
			delegated = row
		}
	}
	if delegated == nil || delegated["delegatedTo"] == nil {
		t.Fatalf("REQ-PARENT must be delegated: %+v", full["requirements"])
	}
	status, ok := delegated["delegatedStatus"].(map[string]any)
	if !ok || status["requirement"] != "REQ-CHILD" || status["status"] == nil {
		t.Fatalf("delegated row must carry delegatedStatus for REQ-CHILD: %+v", delegated)
	}
	for _, args := range []map[string]any{{}, {"summaryOnly": true}} {
		summary := callToolPayload(t, s, "trace.matrix", args)["summary"].(map[string]any)
		for _, key := range []string{"delegatedImplemented", "delegatedTestLinked"} {
			if _, ok := summary[key]; !ok {
				t.Fatalf("summary %v missing %q: %+v", args, key, summary)
			}
		}
	}
}
