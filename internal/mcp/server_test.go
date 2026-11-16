// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"fmt"
	"maps"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestToolsListAndAllToolsReturnPayload(t *testing.T) {
	s := initializedServerWithDesign(t)
	assertToolsList(t, s)
	for i, name := range s.ToolNames() {
		if name == "links.review" {
			// Needs a model with requirement links; the payments sample has
			// none (TestLinksReviewToolReturnsCommandAndPatch covers it).
			continue
		}
		s.assertToolReturnsPayload(t, i, name)
	}
}

// initializedServerWithDesign starts the server on the payments sample with
// its views document as design path.
func initializedServerWithDesign(t *testing.T) *Server {
	t.Helper()
	s := NewServer()
	sample := filepath.Join("..", "..", "examples", "payments-engineering-sample")
	initResp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath":        filepath.Join(sample, "engmod.yml"),
				"requirementsPath": filepath.Join(sample, "model", "requirements.yml"),
				"designPath":       filepath.Join(sample, "model", "views.yml"),
				"repoRoot":         filepath.Join("..", ".."),
			},
		},
	})
	if initResp["error"] != nil {
		t.Fatalf("initialize returned error: %+v", initResp["error"])
	}
	return s
}

// assertToolsList checks that tools/list lists every tool of the server.
func assertToolsList(t *testing.T, s *Server) {
	t.Helper()
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
}

// assertToolReturnsPayload calls one tool with its sample arguments and
// checks the envelope and the domain payload of the result.
func (s *Server) assertToolReturnsPayload(t *testing.T, index int, name string) {
	t.Helper()
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      100 + index,
		"method":  "tools/call",
		"params": map[string]any{
			"name":      name,
			"arguments": argsForTool(name),
		},
	})
	text := toolResultText(t, name, resp)
	payload := decodeToolEnvelope(t, name, text)
	assertUsefulToolPayload(t, toolOutput{name: name, payload: payload, text: text})
}

// toolResultText returns the text of the first content chunk of a successful
// tools/call response.
func toolResultText(t *testing.T, name string, resp map[string]any) string {
	t.Helper()
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
	return text
}

// decodeToolEnvelope parses the tool text and checks ok, schemaVersion and
// the tool name echo.
func decodeToolEnvelope(t *testing.T, name, text string) map[string]any {
	t.Helper()
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
	return payload
}

// TRLC-LINKS: REQ-EMG-045, REQ-EMG-047, REQ-EMG-048, REQ-EMG-182
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func TestModelAuthoringContractIsCompactAndActionable(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := s.callToolPayload(t, "model.authoringContract", map[string]any{})
	contract, ok := payload["contract"].(map[string]any)
	if !ok {
		t.Fatalf("missing authoring contract: %+v", payload)
	}
	if contract["contractVersion"] != "engineering-model.authoring.v8" {
		t.Fatalf("unexpected contract version: %v", contract["contractVersion"])
	}
	assertContractHealthGuide(t, contract)
	assertContractRequiredRules(t, payload)
	assertContractDocuments(t, contract)
}

// assertContractHealthGuide checks the health guide of the contract: 16 codes
// and the waiver rule.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func assertContractHealthGuide(t *testing.T, contract map[string]any) {
	t.Helper()
	health, _ := contract["health"].(map[string]any)
	if codes, _ := health["codes"].([]any); len(codes) != 17 {
		t.Fatalf("the v7 contract must carry the health guide with 17 codes: %+v", health)
	}
	if rules, _ := health["rules"].([]any); len(rules) == 0 || !strings.Contains(fmt.Sprint(rules...), "health.waiver_unjustified") {
		t.Fatalf("the health guide must state the waiver rule: %+v", health)
	}
}

// assertContractRequiredRules checks that the rules list the required
// switches with their remedies.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func assertContractRequiredRules(t *testing.T, payload map[string]any) {
	t.Helper()
	rules, _ := payload["rules"].(map[string]any)
	required, _ := rules["required"].([]any)
	if len(required) == 0 || !strings.Contains(fmt.Sprint(required...), "lifecycle.missing") || !strings.Contains(fmt.Sprint(required...), "ef-tier-up") {
		t.Fatalf("the v7 rules must list the required switches with their remedies: %+v", rules["required"])
	}
}

// assertContractDocuments checks the document count, the absent compatibility
// field and the dependency authoring guidance of the contract.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func assertContractDocuments(t *testing.T, contract map[string]any) {
	t.Helper()
	documents, _ := contract["documents"].([]any)
	if len(documents) != 9 {
		t.Fatalf("expected manifest plus eight domain documents, got %d", len(documents))
	}
	if _, present := contract["compatibility"]; present {
		t.Fatalf("the authoring contract must carry no compatibility field (ADR-EMG-073): %+v", contract)
	}
	dependencies, ok := contract["dependencies"].(map[string]any)
	if !ok || dependencies["registryEnvironment"] != "CUE_REGISTRY" || dependencies["workspaceFile"] != model.WorkspaceFileName {
		t.Fatalf("missing dependency authoring guidance: %+v", contract)
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
				"modelPath": filepath.Join("..", "..", "examples", "payments-engineering-sample", "engmod.yml"),
				"repoRoot":  filepath.Join("..", ".."),
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
				"modelPath": filepath.Join("..", "..", "examples", "payments-engineering-sample", "engmod.yml"),
				"repoRoot":  filepath.Join("..", ".."),
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
				"modelPath": filepath.Join("..", "..", "examples", "payments-engineering-sample", "engmod.yml"),
				"repoRoot":  filepath.Join("..", ".."),
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
	if name == "model.diagnose" {
		content, _ := os.ReadFile(filepath.Join("..", "..", "examples", "payments-engineering-sample", "model", "decisions.yml"))
		return map[string]any{"document": "decisions", "content": string(content)}
	}
	for _, entry := range toolArgsTable {
		if slices.Contains(entry.names, name) {
			return maps.Clone(entry.args)
		}
	}
	return map[string]any{}
}

// toolArgs gives the sample arguments of the tools named by names.
type toolArgs struct {
	names []string
	args  map[string]any
}

// toolArgsTable lists the sample arguments each tool is called with against
// the payments sample; a tool not listed is called without arguments.
var toolArgsTable = []toolArgs{
	{[]string{"requirements.get", "requirements.parents", "requirements.impact", "requirements.supportPath", "requirements.suggestEditPlan", "files.forRequirement", "threats.forRequirement", "flows.forRequirement", "changes.preflight", "tests.forRequirement"}, map[string]any{"requirementId": "REQ-PAY-001"}},
	{[]string{"files.forControl"}, map[string]any{"controlId": "CTRL-PAYMENTS-SSO-MFA"}},
	{[]string{"files.forThreat"}, map[string]any{"threatId": "TS-PAYMENTS-CHECKOUT-SPOOFING"}},
	{[]string{"files.owner", "interfaces.matchFromCode", "coverage.strictStatus"}, map[string]any{"path": filepath.Join("examples", "payments-engineering-sample", "src", "checkout_api.go")}},
	{[]string{"verification.status", "ownership.resolve", "runtime.resolve", "confidence.explain"}, map[string]any{"entityId": "IF-PAYMENTS-CHECKOUT-API"}},
	{[]string{"flows.diff"}, map[string]any{"fromFlowId": "FLOW-CUSTOMER-CHECKOUT", "toFlowId": "FLOW-PAYMENTS-MANUAL-REVIEW"}},
	{[]string{"graph.neighborhood", "graph.search"}, map[string]any{"query": "payments"}},
	{[]string{"graph.explainEdge"}, map[string]any{"from": "FG-PAYMENTS", "to": "FU-CHECKOUT"}},
	{[]string{"model.list"}, map[string]any{"query": "checkout"}},
	{[]string{"model.entity", "model.implementations", "tests.forEntity"}, map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT"}},
	{[]string{"code.contextForTask"}, map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT", "requirementId": "REQ-PAY-001"}},
	{[]string{"views.recommend"}, map[string]any{"taskType": "security"}},
	{[]string{"tier.gap"}, map[string]any{"target": "product"}},
	{[]string{"views.renderContext"}, map[string]any{"viewId": "VIEW-SECURITY"}},
	{[]string{"governance.checkPatch"}, map[string]any{"diff": "tests/x_test.go"}},
	{[]string{"interfaces.resolve", "endpoints.resolve"}, map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API", "environment": "prod"}},
	{[]string{"interfaces.implementations", "identity.resolve", "policy.resolve", "schema.resolve"}, map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API"}},
	{[]string{"environments.resolve"}, map[string]any{"environment": "prod"}},
	{[]string{"schema.diff"}, map[string]any{"fromInterfaceId": "IF-PAYMENTS-CHECKOUT-API", "toInterfaceId": "IF-PAYMENTS-BANK-AUTH"}},
	{[]string{"flow.detail", "flow.implementations"}, map[string]any{"flowId": "FLOW-CUSTOMER-CHECKOUT"}},
	{[]string{"model.complete"}, map[string]any{"prefix": "FU-", "field": "appliesTo"}},
	{[]string{"model.patch"}, map[string]any{"document": "assurance", "id": "RISK-PAYMENTS-SSO-MFA-GAP", "set": map[string]any{"title": "Renamed"}}},
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
func assertUsefulToolPayload(t *testing.T, out toolOutput) {
	t.Helper()
	assertNoPlaceholderText(t, out)
	assertHasDomainKeys(t, out)
	if spec, ok := nonEmptyListChecks[out.name]; ok {
		spec.tool = out.name
		assertNonEmptyList(t, out.payload, spec)
	}
	if check, ok := customPayloadChecks[out.name]; ok {
		check(t, out.payload)
	}
}

// toolOutput is the result of one tool call: its name, the decoded payload and
// the raw text.
type toolOutput struct {
	name    string
	payload map[string]any
	text    string
}

// listSpec names a list a tool payload must carry non-empty: the tool, the
// payload key and the plural noun the failure message uses.
type listSpec struct {
	tool, key, what string
}

// nonEmptyListChecks lists the tools whose payload must carry a non-empty list.
var nonEmptyListChecks = map[string]listSpec{
	"generation.status":      {key: "artifacts", what: "artifacts"},
	"verification.recommend": {key: "recommendations", what: "recommendations"},
	"tasks.entryPoints":      {key: "entryPoints", what: "entry points"},
	"tasks.nextBestActions":  {key: "actions", what: "actions"},
	"views.recommend":        {key: "recommendedViews", what: "views"},
	"code.contextForTask":    {key: "implementationFiles", what: "implementation files"},
	"runtime.resolve":        {key: "deploymentTargets", what: "deployment targets"},
	"policy.resolve":         {key: "controls", what: "controls"},
}

// customPayloadChecks lists the tools with a payload check of their own.
var customPayloadChecks = map[string]func(*testing.T, map[string]any){
	"generation.plan":    assertGenerationPlanPayload,
	"flows.diff":         assertFlowsDiffPayload,
	"governance.policy":  assertGovernancePolicyPayload,
	"confidence.explain": assertConfidencePayload,
}

// assertNoPlaceholderText fails when the tool text carries placeholder text.
func assertNoPlaceholderText(t *testing.T, out toolOutput) {
	t.Helper()
	for _, forbidden := range []string{"tool not implemented", "not yet semantic", "go run ./cmd/engdoc ...", "go run ./cmd/engdragon ...", "go run ./cmd/engstruct ...", "go run ./cmd/engtrlc ..."} {
		if strings.Contains(out.text, forbidden) {
			t.Fatalf("tool %s returned placeholder text %q: %s", out.name, forbidden, out.text)
		}
	}
}

// assertHasDomainKeys fails when the payload carries only envelope keys.
func assertHasDomainKeys(t *testing.T, out toolOutput) {
	t.Helper()
	domainKeys := 0
	for k := range out.payload {
		switch k {
		case "ok", "tool", "schemaVersion", "generatedAt":
		default:
			domainKeys++
		}
	}
	if domainKeys == 0 {
		t.Fatalf("tool %s returned no domain payload keys: %+v", out.name, out.payload)
	}
}

// assertNonEmptyList fails when the payload list named by the spec is empty.
func assertNonEmptyList(t *testing.T, payload map[string]any, spec listSpec) {
	t.Helper()
	rows, _ := payload[spec.key].([]any)
	if len(rows) == 0 {
		t.Fatalf("%s returned no %s", spec.tool, spec.what)
	}
}

// assertGenerationPlanPayload checks that every planned command is usable and
// has outputs.
func assertGenerationPlanPayload(t *testing.T, payload map[string]any) {
	t.Helper()
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
}

// assertFlowsDiffPayload checks that the diff has field, set or step changes.
func assertFlowsDiffPayload(t *testing.T, payload map[string]any) {
	t.Helper()
	if len(payload["fieldChanges"].([]any)) != 0 || len(payload["setChanges"].([]any)) != 0 {
		return
	}
	if stepChanges, _ := payload["stepChanges"].(map[string]any); len(stepChanges) == 0 {
		t.Fatalf("flows.diff returned no semantic diff content: %+v", payload)
	}
}

// assertGovernancePolicyPayload checks that the policy carries rules.
func assertGovernancePolicyPayload(t *testing.T, payload map[string]any) {
	t.Helper()
	policy, _ := payload["policy"].(map[string]any)
	rules, _ := policy["rules"].([]any)
	if len(rules) == 0 {
		t.Fatalf("governance.policy returned no rules: %+v", payload)
	}
}

// assertConfidencePayload checks that the explanation carries a confidence and
// its basis.
func assertConfidencePayload(t *testing.T, payload map[string]any) {
	t.Helper()
	if strings.TrimSpace(payload["confidence"].(string)) == "" {
		t.Fatalf("confidence.explain returned no confidence")
	}
	assertNonEmptyList(t, payload, listSpec{tool: "confidence.explain", key: "basis", what: "basis"})
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func TestInterfaceImplementationsReturnsLinkedSymbols(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := s.callToolPayload(t, "interfaces.implementations", map[string]any{"interfaceId": "IF-PAYMENTS-CHECKOUT-API"})
	items, _ := payload["implementations"].([]any)
	if len(items) == 0 {
		t.Fatalf("expected at least one linked implementation")
	}
	if !hasPathContaining(items, "checkout_api.go") {
		t.Fatalf("expected checkout_api.go in implementations: %+v", items)
	}
}

// hasPathContaining reports whether a row of items has a path containing sub.
func hasPathContaining(items []any, sub string) bool {
	for _, item := range items {
		row, _ := item.(map[string]any)
		if strings.Contains(row["path"].(string), sub) {
			return true
		}
	}
	return false
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
				"modelPath":        filepath.Join("..", "..", "examples", "payments-engineering-sample", "engmod.yml"),
				"requirementsPath": filepath.Join("..", "..", "examples", "payments-engineering-sample", "model", "requirements.yml"),
				"repoRoot":         filepath.Join("..", ".."),
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
				"modelPath":        filepath.Join("..", "..", "examples", "payments-engineering-sample", "engmod.yml"),
				"requirementsPath": filepath.Join("..", "..", "examples", "payments-engineering-sample", "model", "requirements.yml"),
				"repoRoot":         filepath.Join("..", ".."),
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

	listPayload := s.callToolPayload(t, "model.list", map[string]any{"query": "REQ-PAY-001", "kind": "requirement"})
	entities, _ := listPayload["entities"].([]any)
	if len(entities) == 0 {
		t.Fatalf("expected model.list to return matching requirement")
	}

	entityPayload := s.callToolPayload(t, "model.entity", map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT"})
	if entityPayload["kind"] != "flow" || entityPayload["exists"] != true {
		t.Fatalf("expected model.entity flow detail, got %+v", entityPayload)
	}

	assertGroupedFlowImplementations(t, s.callToolPayload(t, "flow.implementations", map[string]any{"flowId": "FLOW-CUSTOMER-CHECKOUT"}))

	contextPayload := s.callToolPayload(t, "code.contextForTask", map[string]any{"entityId": "FLOW-CUSTOMER-CHECKOUT", "requirementId": "REQ-PAY-001"})
	contextFiles, _ := contextPayload["implementationFiles"].([]any)
	if len(contextFiles) == 0 {
		t.Fatalf("expected code.contextForTask to include implementationFiles")
	}

	coveragePayload := s.callToolPayload(t, "coverage.strictStatus", map[string]any{"path": filepath.Join("examples", "payments-engineering-sample", "src", "checkout_api.go")})
	if coveragePayload["status"] == "" {
		t.Fatalf("expected coverage.strictStatus status")
	}
}

// assertGroupedFlowImplementations checks that the grouped flow files carry
// checkout_api.go with its lineList.
func assertGroupedFlowImplementations(t *testing.T, implPayload map[string]any) {
	t.Helper()
	files, _ := implPayload["files"].([]any)
	if len(files) == 0 {
		t.Fatalf("expected grouped flow implementation files")
	}
	foundCheckout := false
	for _, item := range files {
		row, _ := item.(map[string]any)
		if strings.Contains(row["path"].(string), "checkout_api.go") {
			foundCheckout = true
			assertHasLineList(t, row)
		}
	}
	if !foundCheckout {
		t.Fatalf("expected checkout_api.go in grouped implementation files: %+v", files)
	}
}

// assertHasLineList fails when a grouped implementation row has no lineList.
func assertHasLineList(t *testing.T, row map[string]any) {
	t.Helper()
	if strings.TrimSpace(row["lineList"].(string)) == "" {
		t.Fatalf("expected checkout_api.go grouped row to include lineList: %+v", row)
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
				"modelPath":        filepath.Join("..", "..", "examples", "payments-engineering-sample", "engmod.yml"),
				"requirementsPath": filepath.Join("..", "..", "examples", "payments-engineering-sample", "model", "requirements.yml"),
				"repoRoot":         filepath.Join("..", ".."),
			},
		},
	})
	if resp["error"] != nil {
		t.Fatalf("initialize returned error: %+v", resp["error"])
	}
	return s
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func (s *Server) callToolPayload(t *testing.T, name string, args map[string]any) map[string]any {
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
