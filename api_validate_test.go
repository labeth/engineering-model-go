// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// ValidateBundle returns, for each finding, a stable code, a severity, a message and the model path of the finding;
// the bundle of a valid model has no error and one requirement that applies to an unknown ID has exactly that finding.
// Mutation: a renamed code, a lower severity, a lost path or message, or a check that flags a known ID fails this test.
// TRLC-LINKS: REQ-EMG-195
func TestValidateBundleReturnsCodeSeverityMessageAndPathOfEachFinding(t *testing.T) {
	isolatedModel(t)
	bundle, err := engmodel.LoadBundle(filepath.Join(childA, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if diags := engmodel.ValidateBundle(bundle); engmodel.HasErrors(diags) {
		t.Fatalf("the valid model has errors: %+v", diags)
	}
	cases := []struct {
		name      string
		appliesTo []string
		findings  int
	}{
		{"unknown unit", []string{"FU-DOES-NOT-EXIST"}, 1},
		{"known unit and an unknown one", []string{"FU-ENCODER", "FU-DOES-NOT-EXIST"}, 1},
		{"empty ID", []string{""}, 1},
		{"known unit", []string{"FU-ENCODER"}, 0},
		{"qualified ID of a dependency", []string{"encoder::FU-ENCODER"}, 0},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			edited := bundle
			edited.Requirements.Requirements = append([]engmodel.Requirement(nil), bundle.Requirements.Requirements...)
			edited.Requirements.Requirements[0].AppliesTo = c.appliesTo
			var found []engmodel.Diagnostic
			for _, d := range engmodel.SortDiagnostics(engmodel.ValidateBundle(edited)) {
				if d.Code == "requirement.invalid_applies_to" {
					found = append(found, d)
				}
			}
			if len(found) != c.findings {
				t.Fatalf("%d findings, want %d: %+v", len(found), c.findings, found)
			}
			for _, d := range found {
				unknown := c.appliesTo[len(c.appliesTo)-1]
				wantPath := fmt.Sprintf("requirements[0].appliesTo[%d]", len(c.appliesTo)-1)
				if d.Severity != engmodel.SeverityError || d.Path != wantPath || !strings.Contains(d.Message, "REQ-A-001") || !strings.Contains(d.Message, unknown) {
					t.Errorf("finding %+v, want an error at %s that names REQ-A-001 and %q", d, wantPath, unknown)
				}
			}
		})
	}
}
