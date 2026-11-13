// ENGMODEL-OWNER-UNIT: FU-LOBSTER-EXPORTER
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// englobster refuses a model without lifecycle, writes nothing and prints the
// fix of the required-switch error.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGLOBSTER, FU-LOBSTER-EXPORTER
func TestRunRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	out := filepath.Join(t.TempDir(), "activity.lobster")
	var stdout, stderr bytes.Buffer
	code := run([]string{"--model", manifest, "--tests-dir", filepath.Dir(manifest), "--out", out}, &stdout, &stderr)
	testutil.AssertRefusesWithFix(t, code, stdout.String()+stderr.String())
	if _, err := os.Stat(out); err == nil {
		t.Errorf("a refused model wrote %s", out)
	}
}
