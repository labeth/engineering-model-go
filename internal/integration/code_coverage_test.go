// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package integration

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
)

// The payments example, built through the trace the way the CLIs do, reports
// the element to code coverage diagnostics of ADR-EMG-069: each as a warning
// at its declared tier (mvp), with a remedy, and counted exactly, so the
// golden of the examples reflects them.
// Mutation: removing the coverageDiagnostics call from integrityTail, or
// widening what is asked, changes the counts.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestPaymentsExampleReportsCodeCoverage(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, composition.LockModeCheck)
	t.Setenv(composition.ModuleCacheEnv, composition.ModuleCacheReuse)
	manifest := filepath.Join("examples", "payments-engineering-sample", "engmod.yml")
	bundle, err := engmodel.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	_, diags, err := engmodel.BuildTraceMatrixFromFiles(manifest, bundle.RequirementsPath, filepath.Dir(manifest))
	if err != nil {
		t.Fatal(err)
	}
	got := unlinkedCounts(t, diags)
	want := map[string]int{"code.unit_unlinked": 2, "code.state_unlinked": 4, "code.transition_unlinked": 4, "code.event_unlinked": 2}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("coverage diagnostics %v, want %v", got, want)
	}
}

// unlinkedCounts counts the code.*_unlinked diagnostics by code and fails for
// each that is not a warning with the ef-code-linking remedy.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func unlinkedCounts(t *testing.T, diags []engmodel.Diagnostic) map[string]int {
	t.Helper()
	got := map[string]int{}
	for _, d := range diags {
		if !strings.HasPrefix(d.Code, "code.") || !strings.HasSuffix(d.Code, "_unlinked") {
			continue
		}
		got[d.Code]++
		if d.Severity != engmodel.SeverityWarning || d.Remedy == "" || !strings.Contains(d.Remedy, "ef-code-linking") {
			t.Errorf("want a warning at mvp with the ef-code-linking remedy: %+v", d)
		}
	}
	return got
}
