// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engchange refuses a model without lifecycle and prints the fix of the
// required-switch error.
//
// TRLC-LINKS: REQ-EMG-032, REQ-EMG-168
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-CLI-ORCHESTRATION, IF-CLI-ENGCHANGE
func TestEngchangeRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	root := filepath.Dir(manifest)
	delta := filepath.Join(root, "delta.yml")
	if err := os.WriteFile(delta, []byte("version: 1\nrequirements:\n  remove: [REQ-SHARED-001]\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, stderr := testutil.RunMain(t, main, "validate", "--root", root, "--delta", delta)
	testutil.AssertRefusesWithFix(t, code, stdout+stderr)
}
