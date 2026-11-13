// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package main

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engair refuses a model without lifecycle and prints the fix of the
// required-switch error.
//
// TRLC-LINKS: REQ-EMG-055, REQ-EMG-168
// ENGMODEL-LINKS: FU-AIRBORNE-ASSURANCE-EXPORTER, FU-CLI-ORCHESTRATION
func TestEngairRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	code, stdout, stderr := testutil.RunMain(t, main, "--model", manifest, "--out-dir", t.TempDir())
	testutil.AssertRefusesWithFix(t, code, stdout+stderr)
}
