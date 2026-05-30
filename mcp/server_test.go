// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestToolsListAndAllToolsReturnPayload(t *testing.T) {
	s := NewServer()
	modelPath := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	reqPath := filepath.Join("..", "examples", "payments-engineering-sample", "requirements.yml")
	designPath := filepath.Join("..", "examples", "payments-engineering-sample", "design.yml")
	repoRoot := filepath.Join("..")

	initResp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath":        modelPath,
				"requirementsPath": reqPath,
				"designPath":       designPath,
				"repoRoot":         repoRoot,
			},
		},
	})
	if initResp["error"] != nil {
		t.Fatalf("initialize returned error: %+v", initResp["error"])
	}

	listResp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/list",
		"params":  map[string]any{},
	})
	res, ok := listResp["result"].(map[string]any)
	if !ok {
		t.Fatalf("tools/list missing result")
	}
	tools, ok := res["tools"].([]any)
	if !ok {
		t.Fatalf("tools/list missing tools array")
	}
	if got, want := len(tools), len(s.ToolNames()); got != want {
		t.Fatalf("unexpected tools count, got %d want %d", got, want)
	}

	for i, name := range s.ToolNames() {
		args := argsForTool(name)
		resp := rpcCall(t, s, map[string]any{
			"jsonrpc": "2.0",
			"id":      100 + i,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      name,
				"arguments": args,
			},
		})
		if resp["error"] != nil {
			t.Fatalf("tool %s returned error: %+v", name, resp["error"])
		}
		result, ok := resp["result"].(map[string]any)
		if !ok {
			t.Fatalf("tool %s missing result", name)
		}
		if isErr, _ := result["isError"].(bool); isErr {
			t.Fatalf("tool %s returned isError=true", name)
		}
		content, ok := result["content"].([]any)
		if !ok || len(content) == 0 {
			t.Fatalf("tool %s missing content", name)
		}
		chunk, _ := content[0].(map[string]any)
		text, _ := chunk["text"].(string)
		var payload map[string]any
		if err := json.Unmarshal([]byte(text), &payload); err != nil {
			t.Fatalf("tool %s content not json: %v", name, err)
		}
		if payload["ok"] != true {
			t.Fatalf("tool %s missing ok=true", name)
		}
		if payload["schemaVersion"] == "" {
			t.Fatalf("tool %s missing schemaVersion", name)
		}
		if payload["tool"] != name {
			t.Fatalf("tool %s echo mismatch, got %v", name, payload["tool"])
		}
		assertUsefulToolPayload(t, name, payload, text)
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestHandleJSONRPCValidation(t *testing.T) {
	s := NewServer()

	out, err := s.Handle([]byte(`{"jsonrpc":"2.0","id":1,"method":`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp := decodeResponse(t, out)
	errObj, _ := resp["error"].(map[string]any)
	if got := int(errObj["code"].(float64)); got != -32700 {
		t.Fatalf("expected parse error code, got %d", got)
	}

	out, err = s.Handle([]byte(`{"jsonrpc":"1.0","id":2,"method":"ping"}`))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	resp = decodeResponse(t, out)
	errObj, _ = resp["error"].(map[string]any)
	if got := int(errObj["code"].(float64)); got != -32600 {
		t.Fatalf("expected invalid request code, got %d", got)
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestPathTraversalIsRejected(t *testing.T) {
	s := NewServer()
	rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath": filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml"),
				"repoRoot":  filepath.Join(".."),
			},
		},
	})

	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name": "interfaces.matchFromCode",
			"arguments": map[string]any{
				"path": "../../etc/passwd",
			},
		},
	})
	result, _ := resp["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected tool error for traversal path")
	}
	content, _ := result["content"].([]any)
	chunk, _ := content[0].(map[string]any)
	text, _ := chunk["text"].(string)
	if text == "" {
		t.Fatalf("expected traversal error message")
	}
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("error payload json: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["message"] != "interfaces.matchFromCode path must be inside repoRoot" {
		t.Fatalf("unexpected traversal message: %v", errObj["message"])
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestToolsCallRejectsBadInput(t *testing.T) {
	s := NewServer()

	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "tools/call",
		"params":  "invalid",
	})
	errObj, _ := resp["error"].(map[string]any)
	if got := int(errObj["code"].(float64)); got != -32602 {
		t.Fatalf("expected invalid params code, got %d", got)
	}

	resp = rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      99,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath": filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml"),
				"repoRoot":  filepath.Join(".."),
			},
		},
	})

	resp = rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "requirements.get",
			"arguments": "invalid",
		},
	})
	errObj, _ = resp["error"].(map[string]any)
	if got := int(errObj["code"].(float64)); got != -32602 {
		t.Fatalf("expected invalid params code for arguments, got %d", got)
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestToolMissingRequiredArgument(t *testing.T) {
	s := NewServer()
	rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath": filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml"),
				"repoRoot":  filepath.Join(".."),
			},
		},
	})

	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "requirements.get",
			"arguments": map[string]any{},
		},
	})
	result, _ := resp["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); !isErr {
		t.Fatalf("expected tool error for missing required argument")
	}
	content, _ := result["content"].([]any)
	chunk, _ := content[0].(map[string]any)
	text, _ := chunk["text"].(string)
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("error payload json: %v", err)
	}
	errObj, _ := payload["error"].(map[string]any)
	if errObj["message"] != "requirements.get requires requirementId" {
		t.Fatalf("unexpected error message: %v", errObj["message"])
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func argsForTool(name string) map[string]any {
	switch name {
	case "requirements.get", "requirements.impact", "requirements.supportPath", "requirements.suggestEditPlan", "files.forRequirement", "threats.forRequirement", "flows.forRequirement", "changes.preflight":
		return map[string]any{"requirementId": "REQ-PAY-001"}
	case "files.forControl":
		return map[string]any{"controlId": "CTRL-PAYMENTS-SSO-MFA"}
	case "files.forThreat":
		return map[string]any{"threatId": "TS-PAYMENTS-CHECKOUT-SPOOFING"}
	case "files.owner", "interfaces.matchFromCode":
		return map[string]any{"path": filepath.Join("examples", "payments-engineering-sample", "src", "checkout_api.go")}
	case "verification.status", "ownership.resolve", "runtime.resolve", "confidence.explain":
		return map[string]any{"entityId": "IF-PAYMENTS-CHECKOUT-API"}
	case "flows.diff":
		return map[string]any{"fromFlowId": "FLOW-CUSTOMER-CHECKOUT", "toFlowId": "FLOW-PAYMENTS-MANUAL-REVIEW"}
	case "graph.neighborhood", "graph.search":
		return map[string]any{"query": "payments"}
	case "graph.explainEdge":
		return map[string]any{"from": "FG-PAYMENTS", "to": "FU-CHECKOUT"}
	case "model.list", "entities.list":
		return map[string]any{"query": "checkout"}
	case "model.entity":
		return map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT"}
	case "model.implementations":
		return map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT"}
	case "code.contextForTask":
		return map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT", "requirementId": "REQ-PAY-001"}
	case "views.recommend":
		return map[string]any{"taskType": "security"}
	case "views.renderContext":
		return map[string]any{"viewId": "VIEW-SECURITY"}
	case "governance.checkPatch":
		return map[string]any{"diff": "tests/x_test.go"}
	case "interfaces.resolve", "endpoints.resolve":
		return map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API", "environment": "prod"}
	case "interfaces.implementations":
		return map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API"}
	case "identity.resolve", "policy.resolve", "schema.resolve":
		return map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API"}
	case "environments.resolve":
		return map[string]any{"environment": "prod"}
	case "schema.diff":
		return map[string]any{"fromInterfaceId": "IF-PAYMENTS-CHECKOUT-API", "toInterfaceId": "IF-PAYMENTS-BANK-AUTH"}
	case "flow.detail", "flow.implementations":
		return map[string]any{"flowId": "FLOW-CUSTOMER-CHECKOUT"}
	case "tests.forRequirement":
		return map[string]any{"requirementId": "REQ-PAY-001"}
	case "tests.forEntity":
		return map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT"}
	case "coverage.strictStatus":
		return map[string]any{"path": filepath.Join("examples", "payments-engineering-sample", "src", "checkout_api.go")}
	default:
		return map[string]any{}
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func rpcCall(t *testing.T, s *Server, req map[string]any) map[string]any {
	t.Helper()
	b, err := json.Marshal(req)
	if err != nil {
		t.Fatalf("marshal request: %v", err)
	}
	out, err := s.Handle(b)
	if err != nil {
		t.Fatalf("handle request: %v", err)
	}
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("unmarshal response: %v", err)
	}
	return resp
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func decodeResponse(t *testing.T, out []byte) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func assertUsefulToolPayload(t *testing.T, name string, payload map[string]any, text string) {
	t.Helper()
	for _, forbidden := range []string{"tool not implemented", "not yet semantic", "go run ./cmd/engdoc ...", "go run ./cmd/engdragon ...", "go run ./cmd/engstruct ...", "go run ./cmd/engtrlc ..."} {
		if strings.Contains(text, forbidden) {
			t.Fatalf("tool %s returned placeholder text %q: %s", name, forbidden, text)
		}
	}
	domainKeys := 0
	for k := range payload {
		switch k {
		case "ok", "tool", "schemaVersion", "generatedAt":
		default:
			domainKeys++
		}
	}
	if domainKeys == 0 {
		t.Fatalf("tool %s returned no domain payload keys: %+v", name, payload)
	}
	switch name {
	case "generation.plan":
		rows, _ := payload["commands"].([]any)
		if len(rows) == 0 {
			t.Fatalf("generation.plan returned no commands")
		}
		for _, item := range rows {
			row, _ := item.(map[string]any)
			if strings.TrimSpace(row["command"].(string)) == "" || strings.Contains(row["command"].(string), "<missing>") {
				t.Fatalf("generation.plan returned unusable command: %+v", row)
			}
			outputs, _ := row["outputs"].([]any)
			if len(outputs) == 0 {
				t.Fatalf("generation.plan command has no outputs: %+v", row)
			}
		}
	case "generation.status":
		rows, _ := payload["artifacts"].([]any)
		if len(rows) == 0 {
			t.Fatalf("generation.status returned no artifacts")
		}
	case "flows.diff":
		if len(payload["fieldChanges"].([]any)) == 0 && len(payload["setChanges"].([]any)) == 0 {
			stepChanges, _ := payload["stepChanges"].(map[string]any)
			if len(stepChanges) == 0 {
				t.Fatalf("flows.diff returned no semantic diff content: %+v", payload)
			}
		}
	case "verification.recommend":
		rows, _ := payload["recommendations"].([]any)
		if len(rows) == 0 {
			t.Fatalf("verification.recommend returned no recommendations")
		}
	case "governance.policy":
		policy, _ := payload["policy"].(map[string]any)
		rules, _ := policy["rules"].([]any)
		if len(rules) == 0 {
			t.Fatalf("governance.policy returned no rules: %+v", payload)
		}
	case "tasks.entryPoints":
		rows, _ := payload["entryPoints"].([]any)
		if len(rows) == 0 {
			t.Fatalf("tasks.entryPoints returned no entry points")
		}
	case "tasks.nextBestActions":
		rows, _ := payload["actions"].([]any)
		if len(rows) == 0 {
			t.Fatalf("tasks.nextBestActions returned no actions")
		}
	case "views.recommend":
		rows, _ := payload["recommendedViews"].([]any)
		if len(rows) == 0 {
			t.Fatalf("views.recommend returned no views")
		}
	case "code.contextForTask":
		rows, _ := payload["implementationFiles"].([]any)
		if len(rows) == 0 {
			t.Fatalf("code.contextForTask returned no implementation files")
		}
	case "runtime.resolve":
		rows, _ := payload["deploymentTargets"].([]any)
		if len(rows) == 0 {
			t.Fatalf("runtime.resolve returned no deployment targets")
		}
	case "policy.resolve":
		rows, _ := payload["controls"].([]any)
		if len(rows) == 0 {
			t.Fatalf("policy.resolve returned no controls")
		}
	case "confidence.explain":
		if strings.TrimSpace(payload["confidence"].(string)) == "" {
			t.Fatalf("confidence.explain returned no confidence")
		}
		rows, _ := payload["basis"].([]any)
		if len(rows) == 0 {
			t.Fatalf("confidence.explain returned no basis")
		}
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestInterfaceImplementationsReturnsLinkedSymbols(t *testing.T) {
	s := NewServer()
	rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath":        filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml"),
				"requirementsPath": filepath.Join("..", "examples", "payments-engineering-sample", "requirements.yml"),
				"repoRoot":         filepath.Join(".."),
			},
		},
	})

	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "interfaces.implementations",
			"arguments": map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API"},
		},
	})
	result, _ := resp["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("expected implementations lookup to succeed")
	}
	content, _ := result["content"].([]any)
	chunk, _ := content[0].(map[string]any)
	text, _ := chunk["text"].(string)
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("tool payload json: %v", err)
	}
	items, _ := payload["implementations"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected at least one linked implementation")
	}
	found := false
	for _, item := range items {
		row, _ := item.(map[string]any)
		if strings.Contains(row["path"].(string), "checkout_api.go") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected checkout_api.go in implementations: %+v", items)
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestModelImplementationsReturnsLinkedSymbolsForFlowAndData(t *testing.T) {
	s := NewServer()
	rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath":        filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml"),
				"requirementsPath": filepath.Join("..", "examples", "payments-engineering-sample", "requirements.yml"),
				"repoRoot":         filepath.Join(".."),
			},
		},
	})

	for _, id := range []string{"FLOW-CUSTOMER-CHECKOUT", "DO-PAYMENTS-AUTH-REQUEST"} {
		resp := rpcCall(t, s, map[string]any{
			"jsonrpc": "2.0",
			"id":      id,
			"method":  "tools/call",
			"params": map[string]any{
				"name":      "model.implementations",
				"arguments": map[string]any{"entityId": id},
			},
		})
		result, _ := resp["result"].(map[string]any)
		if isErr, _ := result["isError"].(bool); isErr {
			t.Fatalf("expected model implementation lookup to succeed for %s", id)
		}
		content, _ := result["content"].([]any)
		chunk, _ := content[0].(map[string]any)
		text, _ := chunk["text"].(string)
		var payload map[string]any
		if err := json.Unmarshal([]byte(text), &payload); err != nil {
			t.Fatalf("tool payload json: %v", err)
		}
		items, _ := payload["implementations"].([]any)
		if len(items) == 0 {
			t.Fatalf("expected linked implementations for %s", id)
		}
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestGraphSearchIncludesFlowsAndOtherModelEntities(t *testing.T) {
	s := NewServer()
	rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath":        filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml"),
				"requirementsPath": filepath.Join("..", "examples", "payments-engineering-sample", "requirements.yml"),
				"repoRoot":         filepath.Join(".."),
			},
		},
	})

	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      2,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      "graph.search",
			"arguments": map[string]any{"query": "FLOW-CUSTOMER-CHECKOUT"},
		},
	})
	result, _ := resp["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("expected graph.search to succeed")
	}
	content, _ := result["content"].([]any)
	chunk, _ := content[0].(map[string]any)
	text, _ := chunk["text"].(string)
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("tool payload json: %v", err)
	}
	nodes, _ := payload["nodes"].([]any)
	for _, item := range nodes {
		row, _ := item.(map[string]any)
		if row["kind"] == "flow" && row["id"] == "FLOW-CUSTOMER-CHECKOUT" {
			return
		}
	}
	t.Fatalf("expected flow node in graph.search result: %+v", nodes)
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestDeveloperContextToolsReturnGroupedImplementations(t *testing.T) {
	s := initializedPaymentsServer(t)

	listPayload := callToolPayload(t, s, "model.list", map[string]any{"query": "REQ-PAY-001", "kind": "requirement"})
	entities, _ := listPayload["entities"].([]any)
	if len(entities) == 0 {
		t.Fatalf("expected model.list to return matching requirement")
	}

	entityPayload := callToolPayload(t, s, "model.entity", map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT"})
	if entityPayload["kind"] != "flow" || entityPayload["exists"] != true {
		t.Fatalf("expected model.entity flow detail, got %+v", entityPayload)
	}

	implPayload := callToolPayload(t, s, "flow.implementations", map[string]any{"flowId": "FLOW-CUSTOMER-CHECKOUT"})
	files, _ := implPayload["files"].([]any)
	if len(files) == 0 {
		t.Fatalf("expected grouped flow implementation files")
	}
	foundCheckout := false
	for _, item := range files {
		row, _ := item.(map[string]any)
		if strings.Contains(row["path"].(string), "checkout_api.go") {
			foundCheckout = true
			if strings.TrimSpace(row["lineList"].(string)) == "" {
				t.Fatalf("expected checkout_api.go grouped row to include lineList: %+v", row)
			}
		}
	}
	if !foundCheckout {
		t.Fatalf("expected checkout_api.go in grouped implementation files: %+v", files)
	}

	contextPayload := callToolPayload(t, s, "code.contextForTask", map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT", "requirementId": "REQ-PAY-001"})
	contextFiles, _ := contextPayload["implementationFiles"].([]any)
	if len(contextFiles) == 0 {
		t.Fatalf("expected code.contextForTask to include implementationFiles")
	}

	coveragePayload := callToolPayload(t, s, "coverage.strictStatus", map[string]any{"path": filepath.Join("examples", "payments-engineering-sample", "src", "checkout_api.go")})
	if coveragePayload["status"] == "" {
		t.Fatalf("expected coverage.strictStatus status")
	}
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func initializedPaymentsServer(t *testing.T) *Server {
	t.Helper()
	s := NewServer()
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath":        filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml"),
				"requirementsPath": filepath.Join("..", "examples", "payments-engineering-sample", "requirements.yml"),
				"repoRoot":         filepath.Join(".."),
			},
		},
	})
	if resp["error"] != nil {
		t.Fatalf("initialize returned error: %+v", resp["error"])
	}
	return s
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func callToolPayload(t *testing.T, s *Server, name string, args map[string]any) map[string]any {
	t.Helper()
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      name,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": args,
		},
	})
	result, _ := resp["result"].(map[string]any)
	if isErr, _ := result["isError"].(bool); isErr {
		t.Fatalf("tool %s returned error: %+v", name, result)
	}
	content, _ := result["content"].([]any)
	chunk, _ := content[0].(map[string]any)
	text, _ := chunk["text"].(string)
	var payload map[string]any
	if err := json.Unmarshal([]byte(text), &payload); err != nil {
		t.Fatalf("tool %s payload json: %v", name, err)
	}
	return payload
}
