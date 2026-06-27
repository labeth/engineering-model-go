// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

import (
	"encoding/json"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TestGemaraOSCALBridge confirms the Gemara->OSCAL SDK bridge emits well-formed
// OSCAL Catalog and Assessment Results documents from the model.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraOSCALBridge(t *testing.T) {
	opts := GemaraExportOptions{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"}
	modelPath := "examples/payments-engineering-sample/architecture.yml"
	bundle, err := model.LoadBundle(modelPath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}

	catJSON, err := GenerateGemaraOSCALCatalog(bundle, opts)
	if err != nil {
		t.Fatalf("oscal catalog: %v", err)
	}
	var cat map[string]json.RawMessage
	if err := json.Unmarshal([]byte(catJSON), &cat); err != nil {
		t.Fatalf("oscal catalog not valid JSON: %v", err)
	}
	if _, ok := cat["catalog"]; !ok {
		t.Fatalf("oscal catalog missing 'catalog' root: keys=%v", keysOf(cat))
	}

	arJSON, err := GenerateGemaraOSCALAssessmentResults(bundle, model.RequirementsDocument{}, "", opts)
	if err != nil {
		t.Fatalf("oscal assessment results: %v", err)
	}
	if arJSON != "" {
		var ar map[string]json.RawMessage
		if err := json.Unmarshal([]byte(arJSON), &ar); err != nil {
			t.Fatalf("oscal AR not valid JSON: %v", err)
		}
		if _, ok := ar["assessment-results"]; !ok {
			t.Fatalf("oscal AR missing 'assessment-results' root: keys=%v", keysOf(ar))
		}
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func keysOf(m map[string]json.RawMessage) []string {
	out := make([]string, 0, len(m))
	for k := range m {
		out = append(out, k)
	}
	return out
}
