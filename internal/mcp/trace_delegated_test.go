// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"path/filepath"
	"testing"
)

// initializeMCPAt starts the MCP server on the model in base.
// TRLC-LINKS: REQ-EMG-067
func initializeMCPAt(t *testing.T, base string) *Server {
	t.Helper()
	s := NewServer()
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath":        filepath.Join(base, "engmod.yml"),
			"requirementsPath": filepath.Join(base, "model", "requirements.yml"),
			"repoRoot":         base,
		}},
	})
	if resp["error"] != nil {
		t.Fatalf("initialize: %+v", resp["error"])
	}
	return s
}

// traceRowsByID returns the requirement rows of a trace.matrix payload by id.
// TRLC-LINKS: REQ-EMG-067
func traceRowsByID(payload map[string]any) map[string]map[string]any {
	rows := map[string]map[string]any{}
	for _, item := range payload["requirements"].([]any) {
		row := item.(map[string]any)
		rows[row["id"].(string)] = row
	}
	return rows
}

// assertNoAllocationDelegation fails when a row still carries delegatedTo.
// TRLC-LINKS: REQ-EMG-187
func assertNoAllocationDelegation(t *testing.T, rows map[string]map[string]any) {
	t.Helper()
	for _, row := range rows {
		if _, has := row["delegatedTo"]; has {
			t.Fatalf("no row is delegated by an allocation: %+v", row)
		}
	}
}

// assertLinkSummary checks the link counter of one trace.matrix summary and
// that the allocation counters are gone.
// TRLC-LINKS: REQ-EMG-067, REQ-EMG-187
func assertLinkSummary(t *testing.T, summary map[string]any) {
	t.Helper()
	if summary["linksClaimed"] != float64(1) {
		t.Fatalf("want linksClaimed 1, got %+v", summary)
	}
	for _, key := range []string{"delegatedImplemented", "delegatedTestLinked"} {
		if _, has := summary[key]; has {
			t.Fatalf("summary still reports %q: %+v", key, summary)
		}
	}
}

// trace.matrix reports the delegatedStatus of the requirement a requirement
// link reaches (the imported row of the realizing requirement) and the link
// counters of the summary, also with summaryOnly; the allocation counters and
// the delegatedTo field are gone (ADR-EMG-078).
// Mutation: dropping delegatedStatus from the row map, linksClaimed from the
// summary map, or putting delegatedTo or delegatedImplemented back fails this
// test.
// TRLC-LINKS: REQ-EMG-067, REQ-EMG-061, REQ-EMG-187
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTraceMatrixToolReportsDelegatedStatus(t *testing.T) {
	s := initializeMCPAt(t, writeMCPCompositionFixture(t))
	rows := traceRowsByID(callToolPayload(t, s, "trace.matrix", map[string]any{}))
	status, _ := rows["child::REQ-CHILD"]["delegatedStatus"].(map[string]any)
	if status["requirement"] != "REQ-CHILD" || status["status"] == nil {
		t.Fatalf("the linked row must carry delegatedStatus for REQ-CHILD: %+v", rows)
	}
	if links, _ := rows["REQ-PARENT"]["links"].([]any); len(links) != 1 {
		t.Fatalf("REQ-PARENT must carry its link: %+v", rows["REQ-PARENT"])
	}
	assertNoAllocationDelegation(t, rows)
	for _, args := range []map[string]any{{}, {"summaryOnly": true}} {
		assertLinkSummary(t, callToolPayload(t, s, "trace.matrix", args)["summary"].(map[string]any))
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
