// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package integration

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/asciidoc"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/trace"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// A control verification that names a test declaration by inferred CODE id
// and still says not-run gets one verification.status_stale warning from the
// AsciiDoc generation and from the trace, the two calls a model check makes;
// with status pass it is quiet.
// Mutation: not calling staleVerificationStatus from ValidateIntegrity (the
// generation path) fails this test.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-ASCIIDOC-GENERATOR
func TestModelCheckReportsStaleControlVerificationStatus(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteV2ModuleFiles(t, dir, "example.com/cv@v0", "v0.1.0", "cv", "", `  functionalGroups:
    - id: FG-A
      name: Group
  functionalUnits:
    - id: FU-A
      name: Unit
      group: FG-A
`)
	testutil.WriteFile(t, filepath.Join(dir, "model", "catalog.yml"), "schemaVersion: 2\ncatalog:\n  systems:\n    - id: SYS-UNIT\n      name: unit\n      definition: The decoding unit.\n")
	testutil.WriteFile(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 2\nlintRun: {}\nrequirements:\n  - id: REQ-CV-001\n    text: The unit shall decode.\n")
	assurance := `schemaVersion: 2
assurance:
  controls:
    - id: CTRL-A
      name: Decoder rejects bad input
  controlVerifications:
    - id: CV-A
      controlRef: CTRL-A
      method: test
      status: not-run
      verificationRefs: [CODE-STATE-STATE-TEST-GO--TESTDECODEREJECTS]
`
	testutil.WriteFile(t, filepath.Join(dir, "model", "assurance.yml"), assurance)
	testutil.WriteFile(t, filepath.Join(dir, "state", "state.go"), "// ENGMODEL-OWNER-UNIT: FU-A\npackage state\n\n// TRLC-LINKS: REQ-CV-001\nfunc Decode() {}\n")
	testutil.WriteFile(t, filepath.Join(dir, "state", "state_test.go"), "// ENGMODEL-OWNER-UNIT: FU-A\npackage state\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-CV-001\nfunc TestDecodeRejects(t *testing.T) { Decode() }\n")
	manifest := filepath.Join(dir, "engmod.yml")
	requirements := filepath.Join(dir, "model", "requirements.yml")
	views := filepath.Join(dir, "model", "views.yml")

	stale := func() (doc, tr int) {
		t.Helper()
		result, err := asciidoc.GenerateFromFiles(manifest, requirements, views, asciidoc.Options{})
		if err != nil {
			t.Fatalf("generate: %v %+v", err, result.Diagnostics)
		}
		_, traceDiags, err := trace.BuildFromFiles(manifest, requirements, "")
		if err != nil {
			t.Fatal(err)
		}
		count := func(diags []validate.Diagnostic) int {
			n := 0
			for _, d := range diags {
				if d.Code != "verification.status_stale" {
					continue
				}
				if d.Path != "authoredArchitecture.controlVerifications[0].status" || !strings.Contains(d.Message, "CODE-STATE-STATE-TEST-GO--TESTDECODEREJECTS") {
					t.Fatalf("unexpected %+v", d)
				}
				n++
			}
			return n
		}
		doc, tr = count(result.Diagnostics), count(traceDiags)
		return doc, tr
	}
	if doc, tr := stale(); doc != 1 || tr != 1 {
		t.Fatalf("want one verification.status_stale from generation and from the trace, got %d and %d", doc, tr)
	}
	if err := os.WriteFile(filepath.Join(dir, "model", "assurance.yml"), []byte(strings.Replace(assurance, "status: not-run", "status: pass", 1)), 0o600); err != nil {
		t.Fatal(err)
	}
	if doc, tr := stale(); doc != 0 || tr != 0 {
		t.Fatalf("a recorded pass must be quiet, got %d and %d", doc, tr)
	}
}
