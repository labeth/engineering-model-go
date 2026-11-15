// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// ssoRiskBlock is the mitigating SSO risk of the payments sample and
// ssoRiskAccepted its accepted replacement.
const (
	ssoRiskID       = "ACC-RISK-PAYMENTS-SSO-MFA-GAP"
	ssoRiskBlock    = "    status: mitigating\n    likelihood: medium\n    impact: high\n    response: mitigate\n    owner: ACT-PLATFORM-OPERATOR\n"
	ssoRiskAccepted = "    status: accepted\n    acceptedBy:\n      actor: ACT-PLATFORM-OPERATOR\n      date: 2026-11-02\n      statement: I accept the SSO gap until the cleanup ends.\n    likelihood: medium\n    impact: high\n    response: accept\n    owner: ACT-PLATFORM-OPERATOR\n"
)

// copyPaymentsTree copies the whole payments sample into a temp directory.
// TRLC-LINKS: REQ-EMG-089, REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func copyPaymentsTree(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "payments")
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("..", "..", "examples", "payments-engineering-sample"))); err != nil {
		t.Fatal(err)
	}
	return dir
}

// acceptSSORisk returns the assurance document with the SSO risk accepted.
// TRLC-LINKS: REQ-EMG-089, REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func acceptSSORisk(t *testing.T, doc string) string {
	t.Helper()
	if !strings.Contains(doc, ssoRiskBlock) {
		t.Fatal("fixture changed: SSO risk block not found")
	}
	return strings.Replace(doc, ssoRiskBlock, ssoRiskAccepted, 1)
}

// renderedGemaraDocs renders the server's bundle to a mutable document map.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func renderedGemaraDocs(t *testing.T, s *Server) map[string]string {
	t.Helper()
	res, err := engmodel.GenerateGemara(*s.bundle, engmodel.GemaraExportOptions{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	docs := map[string]string{}
	for k, v := range res.YAML {
		docs[k] = v
	}
	return docs
}

// requireInvalid requires validateGemaraDocuments to report valid=false.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func requireInvalid(t *testing.T, s *Server, docs map[string]string) map[string]any {
	t.Helper()
	out := validateGemaraDocuments(s.bundle, docs)
	if out["valid"] != false {
		t.Fatalf("expected valid=false, got %+v", out)
	}
	return out
}

// gemaraPolicyDocument returns the document of a gemara.policy call.
// TRLC-LINKS: REQ-EMG-089, REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func gemaraPolicyDocument(t *testing.T, s *Server) string {
	t.Helper()
	return s.callToolPayload(t, "gemara.policy", map[string]any{})["document"].(string)
}

// gemara.validate on a consistent model reports no cross-reference errors.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func TestGemaraValidateReportsCrossReferences(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := s.callToolPayload(t, "gemara.validate", map[string]any{})
	if payload["valid"] != true {
		t.Fatalf("expected valid=true, got %+v", payload)
	}
	errs, ok := payload["crossReferenceErrors"].([]any)
	if !ok || len(errs) != 0 {
		t.Fatalf("expected empty crossReferenceErrors, got %+v", payload["crossReferenceErrors"])
	}
}

// A dangling reference injected into the rendered YAML makes validation fail.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func TestGemaraValidateRejectsInjectedDanglingRef(t *testing.T) {
	s := initializedPaymentsServer(t)
	docs := renderedGemaraDocs(t, s)
	mapping := docs["control-threat-mapping"]
	check(t, "fixture changed: mapping document has no threat targets").textHas(mapping, "entry-id: TS-")
	docs["control-threat-mapping"] = strings.Replace(mapping, "entry-id: TS-", "entry-id: TS-MISSING-", 1)
	out := requireInvalid(t, s, docs)
	errs, _ := out["crossReferenceErrors"].([]engmodel.GemaraCrossReferenceError)
	if len(errs) != 1 || errs[0].Artifact != "control-threat-mapping" || !strings.HasPrefix(errs[0].EntryId, "TS-MISSING-") {
		t.Fatalf("unexpected crossReferenceErrors: %+v", errs)
	}
}

// gemara.validate also checks the rendered treatment claims against the model:
// an Enforced action for a POA&M item that is not completed makes it invalid.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func TestGemaraValidateRejectsEnforcedOpenPOAMItem(t *testing.T) {
	// The bedrock example has an open POA&M item whose risk is evaluated, so
	// its enforcement log has an Undetermined action.
	s := initializedServerAt(t, filepath.Join("..", "..", "examples", "bedrock-pr-review-github-app-sample"))
	payload := s.callToolPayload(t, "gemara.validate", map[string]any{})
	if errs, ok := payload["modelConsistencyErrors"].([]any); !ok || len(errs) != 0 {
		t.Fatalf("expected empty modelConsistencyErrors, got %+v", payload["modelConsistencyErrors"])
	}
	docs := renderedGemaraDocs(t, s)
	check(t, "fixture changed: bedrock enforcement log has no open POA&M item").textHas(docs["enforcement-log"], "disposition: Undetermined")
	docs["enforcement-log"] = strings.ReplaceAll(docs["enforcement-log"], "disposition: Undetermined", "disposition: Enforced")
	out := requireInvalid(t, s, docs)
	errs, _ := out["modelConsistencyErrors"].([]engmodel.GemaraCrossReferenceError)
	if len(errs) == 0 || errs[0].Artifact != "enforcement-log" {
		t.Fatalf("unexpected modelConsistencyErrors: %+v", errs)
	}
}

// A tool call renders the model files on disk at call time: editing a risk
// between two gemara.policy calls, without a new initialize, changes the
// second result.
// Mutation: removing the refresh before tools/call returns the old policy.
// TRLC-LINKS: REQ-EMG-089, REQ-EMG-015
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func TestGemaraPolicyReflectsEditsBetweenCalls(t *testing.T) {
	dir := copyPaymentsTree(t)
	s := initializedServerAt(t, dir)
	check(t, "gemara.policy before the edit (fixture changed, the SSO risk is already accepted)").textLacks(gemaraPolicyDocument(t, s), ssoRiskID)
	path := filepath.Join(dir, "model", "assurance.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(acceptSSORisk(t, string(data))), 0o644); err != nil {
		t.Fatal(err)
	}
	check(t, "second gemara.policy call did not reflect the edit").textHas(gemaraPolicyDocument(t, s), ssoRiskID)
}

// A tool call after the model files became invalid returns the reload error
// (JSON-RPC -32001) instead of a result from the previous model, and keeps
// returning it while the files stay broken; once the file is fixed, the next
// call reloads and returns the fixed content.
// Mutation: ignoring the refreshModel error in dispatch serves the old
// policy for the broken model.
// TRLC-LINKS: REQ-EMG-089
// ENGMODEL-LINKS: FU-MCP-SERVER, EVT-MCP-TOOL-CALL-RECEIVED
func TestToolCallReturnsReloadErrorForBrokenModel(t *testing.T) {
	dir := copyPaymentsTree(t)
	s := initializedServerAt(t, dir)
	s.callToolPayload(t, "gemara.policy", map[string]any{})
	path := filepath.Join(dir, "model", "assurance.yml")
	good, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, append(append([]byte(nil), good...), []byte("\n  broken: [unclosed\n")...), 0o644); err != nil {
		t.Fatal(err)
	}
	call := map[string]any{"jsonrpc": "2.0", "id": 7, "method": "tools/call", "params": map[string]any{"name": "gemara.policy", "arguments": map[string]any{}}}
	for attempt := 0; attempt < 2; attempt++ {
		requireReloadError(t, attempt, rpcCall(t, s, call))
	}
	if err := os.WriteFile(path, []byte(acceptSSORisk(t, string(good))), 0o644); err != nil {
		t.Fatal(err)
	}
	check(t, "call after the fix did not return the fixed model").textHas(gemaraPolicyDocument(t, s), ssoRiskID)
}

// reloadErrorWanted reports whether resp is a -32001 reload error with no result.
// TRLC-LINKS: REQ-EMG-089
// ENGMODEL-LINKS: FU-MCP-SERVER, EVT-MCP-TOOL-CALL-RECEIVED
func reloadErrorWanted(resp map[string]any) bool {
	rpcErr, _ := resp["error"].(map[string]any)
	if resp["result"] != nil || rpcErr == nil {
		return false
	}
	message, _ := rpcErr["message"].(string)
	return rpcErr["code"] == float64(-32001) && strings.Contains(message, "reload changed model")
}

// requireReloadError fails unless resp is the reload error.
// TRLC-LINKS: REQ-EMG-089
// ENGMODEL-LINKS: FU-MCP-SERVER, EVT-MCP-TOOL-CALL-RECEIVED
func requireReloadError(t *testing.T, attempt int, resp map[string]any) {
	t.Helper()
	if !reloadErrorWanted(resp) {
		t.Fatalf("attempt %d: want a -32001 reload error and no result, got %+v", attempt, resp)
	}
}
