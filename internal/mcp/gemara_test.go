// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// gemara.validate on a consistent model reports no cross-reference errors.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func TestGemaraValidateReportsCrossReferences(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := callToolPayload(t, s, "gemara.validate", map[string]any{})
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
	res, err := engmodel.GenerateGemara(*s.bundle, engmodel.GemaraExportOptions{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	docs := map[string]string{}
	for k, v := range res.YAML {
		docs[k] = v
	}
	mapping := docs["control-threat-mapping"]
	if !strings.Contains(mapping, "entry-id: TS-") {
		t.Fatalf("fixture changed: mapping document has no threat targets")
	}
	docs["control-threat-mapping"] = strings.Replace(mapping, "entry-id: TS-", "entry-id: TS-MISSING-", 1)
	out := validateGemaraDocuments(s.bundle, docs)
	if out["valid"] != false {
		t.Fatalf("expected valid=false, got %+v", out)
	}
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
	payload := callToolPayload(t, s, "gemara.validate", map[string]any{})
	if errs, ok := payload["modelConsistencyErrors"].([]any); !ok || len(errs) != 0 {
		t.Fatalf("expected empty modelConsistencyErrors, got %+v", payload["modelConsistencyErrors"])
	}
	res, err := engmodel.GenerateGemara(*s.bundle, engmodel.GemaraExportOptions{})
	if err != nil {
		t.Fatalf("generate: %v", err)
	}
	docs := map[string]string{}
	for k, v := range res.YAML {
		docs[k] = v
	}
	if !strings.Contains(docs["enforcement-log"], "disposition: Undetermined") {
		t.Fatalf("fixture changed: bedrock enforcement log has no open POA&M item")
	}
	docs["enforcement-log"] = strings.ReplaceAll(docs["enforcement-log"], "disposition: Undetermined", "disposition: Enforced")
	out := validateGemaraDocuments(s.bundle, docs)
	if out["valid"] != false {
		t.Fatalf("expected valid=false, got %+v", out)
	}
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
	dir := filepath.Join(t.TempDir(), "payments")
	if err := os.CopyFS(dir, os.DirFS(filepath.Join("..", "..", "examples", "payments-engineering-sample"))); err != nil {
		t.Fatal(err)
	}
	s := initializedServerAt(t, dir)
	const accepted = "ACC-RISK-PAYMENTS-SSO-MFA-GAP"
	before := callToolPayload(t, s, "gemara.policy", map[string]any{})["document"].(string)
	if strings.Contains(before, accepted) {
		t.Fatalf("fixture changed: the SSO risk is already accepted:\n%s", before)
	}
	path := filepath.Join(dir, "model", "assurance.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	old := "    status: mitigating\n    likelihood: medium\n    impact: high\n    response: mitigate\n    owner: ACT-PLATFORM-OPERATOR\n"
	if !strings.Contains(string(data), old) {
		t.Fatal("fixture changed: SSO risk block not found")
	}
	edited := strings.Replace(string(data), old, "    status: accepted\n    acceptedBy:\n      actor: ACT-PLATFORM-OPERATOR\n      date: 2026-11-02\n      statement: I accept the SSO gap until the cleanup ends.\n    likelihood: medium\n    impact: high\n    response: accept\n    owner: ACT-PLATFORM-OPERATOR\n", 1)
	if err := os.WriteFile(path, []byte(edited), 0o644); err != nil {
		t.Fatal(err)
	}
	after := callToolPayload(t, s, "gemara.policy", map[string]any{})["document"].(string)
	if !strings.Contains(after, accepted) {
		t.Fatalf("second gemara.policy call did not reflect the edit:\n%s", after)
	}
}
