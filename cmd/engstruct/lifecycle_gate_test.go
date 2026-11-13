// ENGMODEL-OWNER-UNIT: FU-STRUCTURIZR-EXPORTER
package main

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engstruct refuses a model without lifecycle and prints the fix of the
// required-switch error.
//
// TRLC-LINKS: REQ-EMG-168
// ENGMODEL-LINKS: FU-STRUCTURIZR-EXPORTER, FU-CLI-ORCHESTRATION
func TestEngstructRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	code, stdout, stderr := testutil.RunMain(t, main, "--model", manifest)
	testutil.AssertRefusesWithFix(t, code, stdout+stderr)
}
