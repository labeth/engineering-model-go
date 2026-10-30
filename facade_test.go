// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel_test

import (
	"path/filepath"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// The facade is the only public API: an external caller loads and validates a
// model with engmodel.* alone, without importing any internal package.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-044
func TestFacadeLoadsAndValidatesWithoutInternalImports(t *testing.T) {
	bundle, err := engmodel.LoadBundle(filepath.Join("examples", "payments-engineering-sample", "engmod.yml"))
	if err != nil {
		t.Fatalf("LoadBundle: %v", err)
	}
	if len(bundle.Requirements.Requirements) == 0 {
		t.Fatal("LoadBundle returned no requirements")
	}
	var diags []engmodel.Diagnostic = engmodel.ValidateBundle(bundle)
	if engmodel.HasErrors(diags) {
		t.Fatalf("sample model has validation errors: %+v", diags)
	}

	broken := bundle
	broken.Requirements.Requirements = append([]engmodel.Requirement(nil), bundle.Requirements.Requirements...)
	broken.Requirements.Requirements[0].AppliesTo = []string{"FU-DOES-NOT-EXIST"}
	diags = engmodel.SortDiagnostics(engmodel.ValidateBundle(broken))
	if !engmodel.HasErrors(diags) {
		t.Fatal("ValidateBundle accepted a requirement that applies to an unknown ID")
	}
	found := false
	for _, d := range diags {
		if d.Code == "requirement.invalid_applies_to" && d.Severity == engmodel.SeverityError {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected requirement.invalid_applies_to error, got %+v", diags)
	}
}
