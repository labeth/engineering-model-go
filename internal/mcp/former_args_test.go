// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"strings"
	"testing"
)

// formerArgumentCases pairs each removed alias with the canonical argument the
// rejection must name.
var formerArgumentCases = []struct{ tool, former, canonical string }{
	{"requirements.get", "id", "requirementId"},
	{"requirements.get", "reqId", "requirementId"},
	{"links.review", "id", "linkId"},
	{"files.forControl", "id", "controlId"},
	{"files.forThreat", "id", "threatId"},
	{"files.owner", "file", "path"},
	{"verification.status", "id", "entityId"},
	{"flows.diff", "left", "fromFlowId"},
	{"flows.diff", "right", "toFlowId"},
	{"graph.search", "q", "query"},
	{"graph.search", "name", "query"},
	{"graph.neighborhood", "id", "query"},
	{"model.list", "q", "query"},
	{"model.complete", "query", "prefix"},
	{"model.complete", "q", "prefix"},
	{"model.diagnose", "path", "document"},
	{"model.entity", "id", "entityId"},
	{"code.contextForTask", "reqId", "requirementId"},
	{"code.contextForTask", "file", "path"},
	{"views.recommend", "task", "taskType"},
	{"views.renderContext", "id", "viewId"},
	{"interfaces.resolve", "name", "interfaceId"},
	{"interfaces.resolve", "env", "environment"},
	{"environments.resolve", "env", "environment"},
	{"policy.resolve", "id", "interfaceId"},
	{"schema.diff", "leftSchemaRef", "fromSchemaRef"},
	{"schema.diff", "rightSchemaRef", "toSchemaRef"},
	{"schema.diff", "leftInterfaceId", "fromInterfaceId"},
	{"schema.diff", "rightInterfaceId", "toInterfaceId"},
	{"flow.detail", "id", "flowId"},
	{"tests.forRequirement", "reqId", "requirementId"},
	{"changes.preflight", "id", "requirementId"},
}

// A former argument alias is rejected as an unknown argument and the message
// names the canonical argument; the canonical argument works; the duplicate
// tool entities.list is gone (ADR-EMG-073).
// Mutation: putting an alias back in toolArgsAllowlist, or dropping the
// canonicalArgFor hint, fails this test.
// TRLC-LINKS: REQ-EMG-182
func TestFormerToolArgumentAliasesAreRejectedWithTheCanonicalName(t *testing.T) {
	s := initializedPaymentsServer(t)
	for _, tc := range formerArgumentCases {
		resp := rpcCall(t, s, map[string]any{
			"jsonrpc": "2.0", "id": 1, "method": "tools/call",
			"params": map[string]any{"name": tc.tool, "arguments": map[string]any{tc.former: "X"}},
		})
		rpcErr, _ := resp["error"].(map[string]any)
		message, _ := rpcErr["message"].(string)
		if !strings.Contains(message, "unexpected argument "+`"`+tc.former+`"`) || !strings.Contains(message, `use "`+tc.canonical+`"`) {
			t.Errorf("%s with %q: want rejection naming %q, got %+v", tc.tool, tc.former, tc.canonical, resp)
		}
	}
	if payload := callToolPayload(t, s, "requirements.get", map[string]any{"requirementId": "REQ-PAY-001"}); payload["ok"] != true {
		t.Fatalf("canonical requirementId must work: %+v", payload)
	}
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 2, "method": "tools/call",
		"params": map[string]any{"name": "entities.list", "arguments": map[string]any{}},
	})
	if resp["error"] == nil {
		t.Fatalf("entities.list must be an unknown tool: %+v", resp)
	}
	if _, ok := s.tools["entities.list"]; ok {
		t.Fatal("entities.list must not be listed")
	}
}
