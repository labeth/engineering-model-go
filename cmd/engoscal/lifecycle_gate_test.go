// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package main

import (
	"strings"
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

// The legacy flag --out is gone: --ssp-out is the one name of the SSP output
// (ADR-EMG-073). Mutation: registering the flag again fails this test.
//
// TRLC-LINKS: REQ-EMG-182
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FU-OSCAL-EXPORTER
func TestEngoscalRejectsTheRemovedOutFlag(t *testing.T) {
	code, stdout, stderr := testutil.RunMain(t, main, "--model", "engmod.yml", "--out", "x.json")
	if code == 0 || !strings.Contains(stdout+stderr, "flag provided but not defined: -out") {
		t.Fatalf("engoscal --out must be rejected as an unknown flag: %d\n%s%s", code, stdout, stderr)
	}
}
