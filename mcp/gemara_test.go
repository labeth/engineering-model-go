// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
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
	s := initializedServerAt(t, filepath.Join("..", "examples", "bedrock-pr-review-github-app-sample"))
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
