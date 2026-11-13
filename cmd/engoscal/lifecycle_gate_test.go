// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package main

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engoscal refuses a model without lifecycle and prints the fix of the
// required-switch error.
//
// TRLC-LINKS: REQ-EMG-013, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FU-OSCAL-EXPORTER
func TestEngoscalRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	code, stdout, stderr := testutil.RunMain(t, main, "--model", manifest)
	testutil.AssertRefusesWithFix(t, code, stdout+stderr)
}
