// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package exportgate

import (
	"bytes"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// The gate refuses a model without lifecycle and prints the fix; it passes a
// complete model and reports a model that does not load as an error.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-168
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-VALIDATION-ENGINE
func TestValidationGate(t *testing.T) {
	var out bytes.Buffer
	_, ok, err := ValidationGate(testutil.NoLifecycleModel(t), &out)
	if err != nil || ok {
		t.Fatalf("model without lifecycle: ok=%v err=%v, want refused", ok, err)
	}
	testutil.AssertRefusesWithFix(t, 1, out.String())

	dir := t.TempDir()
	testutil.WriteV2ModuleFiles(t, dir, "example.com/complete@v0", "v0.1.0", "COMPLETE", "", "  functionalGroups:\n    - id: FG-A\n      name: A\n")
	out.Reset()
	if _, ok, err := ValidationGate(filepath.Join(dir, "engmod.yml"), &out); err != nil || !ok {
		t.Fatalf("complete model: ok=%v err=%v\n%s", ok, err, out.String())
	}

	if _, _, err := ValidationGate(filepath.Join(dir, "missing.yml"), &out); err == nil {
		t.Fatal("a manifest that does not exist loaded")
	}
}
