// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
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
	out := validateGemaraDocuments(docs)
	if out["valid"] != false {
		t.Fatalf("expected valid=false, got %+v", out)
	}
	errs, _ := out["crossReferenceErrors"].([]engmodel.GemaraCrossReferenceError)
	if len(errs) != 1 || errs[0].Artifact != "control-threat-mapping" || !strings.HasPrefix(errs[0].EntryId, "TS-MISSING-") {
		t.Fatalf("unexpected crossReferenceErrors: %+v", errs)
	}
}
