// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engview refuses a model without lifecycle and prints the fix of the
// required-switch error.
//
// TRLC-LINKS: REQ-EMG-168
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION
func TestEngviewRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	code, stdout, stderr := testutil.RunMain(t, main, "--model", manifest, "--view", "VIEW-X")
	testutil.AssertRefusesWithFix(t, code, stdout+stderr)
}
