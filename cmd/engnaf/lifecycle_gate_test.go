// ENGMODEL-OWNER-UNIT: FU-NAF-EXPORTER
package main

import (
	"bytes"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engnaf refuses a model without lifecycle and prints the fix of the
// required-switch error.
//
// TRLC-LINKS: REQ-EMG-042, REQ-EMG-168
// ENGMODEL-LINKS: FU-NAF-EXPORTER, IF-CLI-ENGNAF
func TestRunRefusesModelWithoutLifecycle(t *testing.T) {
	var stdout, stderr bytes.Buffer
	code := run([]string{"--model", testutil.NoLifecycleModel(t)}, &stdout, &stderr)
	testutil.AssertRefusesWithFix(t, code, stdout.String()+stderr.String())
}
