// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engdoc refuses a model without lifecycle and prints the fix of the
// required-switch error.
//
// TRLC-LINKS: REQ-EMG-003, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FU-CLI-ORCHESTRATION
func TestEngdocRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	code, stdout, stderr := testutil.RunMain(t, main, "--model", manifest, "--requirements", filepath.Join(filepath.Dir(manifest), "model", "requirements.yml"), "--design", filepath.Join(filepath.Dir(manifest), "model", "views.yml"))
	testutil.AssertRefusesWithFix(t, code, stdout+stderr)
}
