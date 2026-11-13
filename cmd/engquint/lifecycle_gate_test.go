// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engquint generate refuses a model without lifecycle, writes no module and
// prints the fix of the required-switch error.
//
// TRLC-LINKS: REQ-EMG-141, REQ-EMG-168
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func TestGenerateRefusesModelWithoutLifecycle(t *testing.T) {
	out := filepath.Join(t.TempDir(), "quint")
	code, stdout, stderr := runCLI("generate", "--model", testutil.NoLifecycleModel(t), "--out", out)
	testutil.AssertRefusesWithFix(t, code, stdout+stderr)
	if _, err := os.Stat(out); err == nil {
		t.Errorf("a refused model created the output directory %s", out)
	}
}
