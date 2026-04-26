package mcp

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

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
	}
}

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
		return map[string]any{"from": "FU-PAYMENTS-CHECKOUT-API", "to": "IF-PAYMENTS-CHECKOUT-API"}
	case "views.recommend":
		return map[string]any{"taskType": "security"}
	case "views.renderContext":
		return map[string]any{"viewId": "VIEW-SECURITY"}
	case "governance.checkPatch":
		return map[string]any{"diff": "tests/x_test.go"}
	case "interfaces.resolve", "endpoints.resolve":
		return map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API", "environment": "prod"}
	case "identity.resolve", "policy.resolve", "schema.resolve":
		return map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API"}
	case "environments.resolve":
		return map[string]any{"environment": "prod"}
	case "schema.diff":
		return map[string]any{"fromInterfaceId": "IF-PAYMENTS-CHECKOUT-API", "toInterfaceId": "IF-PAYMENTS-BANK-AUTH"}
	default:
		return map[string]any{}
	}
}

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

func decodeResponse(t *testing.T, out []byte) map[string]any {
	t.Helper()
	var resp map[string]any
	if err := json.Unmarshal(out, &resp); err != nil {
		t.Fatalf("decode response: %v", err)
	}
	return resp
}
