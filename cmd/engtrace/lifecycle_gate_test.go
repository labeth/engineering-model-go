// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package main

import (
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engtrace refuses a model without lifecycle and prints the fix of the
// required-switch error.
//
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-168
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CLI-ORCHESTRATION
func TestEngtraceRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	code, stdout, stderr := testutil.RunMain(t, main, "--model", manifest, "--requirements", filepath.Join(filepath.Dir(manifest), "model", "requirements.yml"))
	testutil.AssertRefusesWithFix(t, code, stdout+stderr)
}
