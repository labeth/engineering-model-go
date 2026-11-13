// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// diagRows carries the remedy of a diagnostic as the remedy key and leaves
// the key out of a diagnostic without one.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER
func TestDiagRowsCarryRemedy(t *testing.T) {
	rows := diagRows([]validate.Diagnostic{
		{Code: "lifecycle.missing", Severity: validate.SeverityError, Message: "m", Remedy: "run the ef-tier-up skill"},
		{Code: "other.code", Severity: validate.SeverityWarning, Message: "m"},
	})
	if got := rows[0]["remedy"]; got != "run the ef-tier-up skill" {
		t.Errorf("remedy = %v, want the remedy of the diagnostic", got)
	}
	if _, present := rows[1]["remedy"]; present {
		t.Errorf("a diagnostic without remedy has a remedy key: %v", rows[1])
	}
}

// model.diagnose answers lifecycle.missing with its remedy in the
// diagnostics JSON.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER
func TestModelDiagnoseReportsRemedyOfRequiredSwitch(t *testing.T) {
	t.Setenv("CUE_REGISTRY", "none")
	t.Setenv("ENGMOD_LOCK", "")
	t.Setenv("ENGMOD_WORK", "")
	dir := filepath.Dir(testutil.NoLifecycleModel(t))
	content, err := os.ReadFile(filepath.Join(dir, "model", "requirements.yml"))
	if err != nil {
		t.Fatal(err)
	}
	s := initializedServerAt(t, dir)
	result := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "requirements", "content": string(content)})
	other, _ := result["otherDiagnostics"].(map[string]any)
	remedies, _ := other["remedies"].(map[string]any)
	if remedy, _ := remedies["lifecycle.missing"].(string); remedy == "" {
		t.Fatalf("otherDiagnostics has no remedy of lifecycle.missing: %s", fmt.Sprint(result))
	}
}
