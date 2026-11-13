// ENGMODEL-OWNER-UNIT: FU-TRLC-EXPORTER
package main

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/testutil"
)

// engtrlc refuses a model without lifecycle, writes nothing and prints the
// fix of the required-switch error.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGTRLC, FU-TRLC-EXPORTER
func TestRunRefusesModelWithoutLifecycle(t *testing.T) {
	manifest := testutil.NoLifecycleModel(t)
	out := filepath.Join(t.TempDir(), "trlc")
	var stdout, stderr bytes.Buffer
	code := run([]string{"--model", manifest, "--requirements", filepath.Join(filepath.Dir(manifest), "model", "requirements.yml"), "--out-dir", out}, &stdout, &stderr)
	testutil.AssertRefusesWithFix(t, code, stdout.String()+stderr.String())
	if _, err := os.Stat(out); err == nil {
		t.Errorf("a refused model created the output directory %s", out)
	}
}
