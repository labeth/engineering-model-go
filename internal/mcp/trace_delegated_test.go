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

// trace.matrix reports gatedOnly and gatedOnlyRequirements in its summary,
// also with summaryOnly, and gated with the gates on a row whose every linked
// test skips unless an environment variable is set.
// Mutation: dropping gatedOnly from the summary map fails this test.
// TRLC-LINKS: REQ-EMG-088
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTraceMatrixToolReportsGatedOnly(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "solo")
	writeMCPModule(t, dir, "example.com/solo@v1", "v1.0.0", "SOLO", "", "  contract: {}\n")
	writeMCPFile(t, filepath.Join(dir, "e2e_test.go"), "package solo\n\nimport (\n\t\"os\"\n\t\"testing\"\n)\n\n// TRLC-LINKS: REQ-PARENT\nfunc TestE2E(t *testing.T) {\n\tif os.Getenv(\"SOLO_E2E\") == \"\" {\n\t\tt.Skip(\"set SOLO_E2E\")\n\t}\n}\n\n// TRLC-LINKS: REQ-CHILD\nfunc TestUnit(t *testing.T) {}\n")
	s := NewServer()
	if resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath": filepath.Join(dir, "engmod.yml"), "repoRoot": dir,
		}},
	}); resp["error"] != nil {
		t.Fatalf("initialize: %+v", resp["error"])
	}
	for _, args := range []map[string]any{{}, {"summaryOnly": true}} {
		summary := callToolPayload(t, s, "trace.matrix", args)["summary"].(map[string]any)
		list, _ := summary["gatedOnlyRequirements"].([]any)
		if summary["gatedOnly"] != float64(1) || len(list) != 1 || list[0] != "REQ-PARENT" || summary["testLinked"] != float64(2) {
			t.Fatalf("summary %v: want gatedOnly 1 [REQ-PARENT] and testLinked 2, got %+v", args, summary)
		}
	}
	for _, item := range callToolPayload(t, s, "trace.matrix", map[string]any{})["requirements"].([]any) {
		row := item.(map[string]any)
		gated, _ := row["gated"].(bool)
		gates, _ := row["gates"].([]any)
		switch row["id"] {
		case "REQ-PARENT":
			if !gated || len(gates) != 1 || gates[0] != "SOLO_E2E" {
				t.Fatalf("REQ-PARENT row must be gated by SOLO_E2E: %+v", row)
			}
		case "REQ-CHILD":
			if gated {
				t.Fatalf("REQ-CHILD row is not gated: %+v", row)
			}
		}
	}
}
