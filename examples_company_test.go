// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/module"
	"cuelang.org/go/mod/modzip"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TestAtlasIndustriesCompanyExample proves that every simulated repository is a
// valid standalone model and that workspace replacements resolve the complete
// company portfolio without registry access.
// TRLC-LINKS: REQ-EMG-050
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func TestAtlasIndustriesCompanyExample(t *testing.T) {
	root := filepath.Join("examples", "atlas-industries")
	repositories := []string{
		"shared-edge-platform",
		"shared-cloud-platform",
		"shared-security-services",
		"shared-compliance-baseline",
		"aegis-sentinel",
		"orion-relay",
		"company-portfolio",
	}
	for _, repository := range repositories {
		t.Run("load/"+repository, func(t *testing.T) {
			architecture := filepath.Join(root, "repos", repository, "engmod.yml")
			bundle, err := model.LoadBundle(architecture)
			if err != nil {
				t.Fatalf("load %s: %v", repository, err)
			}
			for _, diagnostic := range validate.Bundle(bundle) {
				if diagnostic.Severity == validate.SeverityError {
					t.Fatalf("%s validation error: %s %s", repository, diagnostic.Code, diagnostic.Message)
				}
			}
			modulePath, err := model.ReadCUEModulePath(filepath.Join(root, "repos", repository))
			if err != nil {
				t.Fatalf("read module identity for %s: %v", repository, err)
			}
			version, err := module.NewVersion(modulePath, "v0.1.0")
			if err != nil {
				t.Fatalf("module version for %s: %v", repository, err)
			}
			if report, err := modzip.CheckDir(filepath.Join(root, "repos", repository)); err != nil {
				t.Fatalf("module %s is not publishable: %v (%+v)", version, err, report)
			}
		})
	}

	portfolio := filepath.Join(root, "repos", "company-portfolio", "engmod.yml")
	composition, err := GenerateCompositionFromFile(portfolio)
	if err != nil {
		t.Fatalf("compose portfolio: %v", err)
	}
	for _, diagnostic := range composition.Diagnostics {
		if diagnostic.Severity == validate.SeverityError {
			t.Fatalf("composition error: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
	if composition.Root == nil {
		t.Fatal("portfolio composition has no root")
	}
	if got := len(composition.Root.Children); got != 2 {
		t.Fatalf("expected two product children, got %d", got)
	}

	shared := map[string]int{}
	for _, product := range composition.Root.Children {
		if got := len(product.Children); got != 4 {
			t.Fatalf("product %s: expected four shared children, got %d", product.Bundle.Architecture.Model.ID, got)
		}
		for _, child := range product.Children {
			shared[child.Bundle.Architecture.Model.ID]++
		}
	}
	for _, id := range []string{
		"atlas-shared-edge-platform",
		"atlas-shared-cloud-platform",
		"atlas-shared-security-services",
		"atlas-shared-compliance-baseline",
	} {
		if got := shared[id]; got != 2 {
			t.Fatalf("expected shared module %s below both products, got %d uses", id, got)
		}
	}
	for _, product := range composition.Root.Children {
		foundPeopleBaseline := false
		for _, child := range product.Children {
			if child.Bundle.Architecture.Model.ID != "atlas-shared-compliance-baseline" {
				continue
			}
			for _, unit := range child.Bundle.Architecture.AuthoredArchitecture.FunctionalUnits {
				if unit.ID == "FU-COMP-PEOPLE" {
					foundPeopleBaseline = true
					break
				}
			}
		}
		if !foundPeopleBaseline {
			t.Fatalf("product %s does not inherit the people compliance baseline", product.Bundle.Architecture.Model.ID)
		}
	}
	if got := len(composition.Allocations); got != 10 {
		t.Fatalf("expected ten recursive requirement allocations, got %d", got)
	}
	for _, allocation := range composition.Allocations {
		if !allocation.Resolved || !allocation.TargetRefResolved {
			t.Fatalf("unresolved allocation: %+v", allocation)
		}
	}

	compliance, err := model.LoadBundle(filepath.Join(root, "repos", "shared-compliance-baseline", "engmod.yml"))
	if err != nil {
		t.Fatalf("load compliance baseline: %v", err)
	}
	requiredControls := map[string]bool{
		"CTRL-COMP-PERSONNEL-SCREENING":   false,
		"CTRL-COMP-POLICY-ACKNOWLEDGMENT": false,
		"CTRL-COMP-SECURITY-AWARENESS":    false,
		"CTRL-COMP-ROLE-CHANGE-REVIEW":    false,
		"CTRL-COMP-OFFBOARDING":           false,
	}
	for _, control := range compliance.Architecture.AuthoredArchitecture.Controls {
		if _, required := requiredControls[control.ID]; required {
			requiredControls[control.ID] = true
		}
	}
	for controlID, found := range requiredControls {
		if !found {
			t.Errorf("people compliance baseline is missing control %s", controlID)
		}
	}
	requiredMappings := map[string]bool{
		"IMPL-COMP-NIST-PEOPLE": false,
		"IMPL-COMP-ISO-PEOPLE":  false,
	}
	for _, mapping := range compliance.Architecture.Compliance.Mappings {
		if _, required := requiredMappings[mapping.ID]; required {
			requiredMappings[mapping.ID] = len(mapping.ControlIDs) >= 4 && len(mapping.Evidence) >= 2
		}
	}
	for mappingID, complete := range requiredMappings {
		if !complete {
			t.Errorf("people compliance mapping %s is missing framework controls or evidence", mappingID)
		}
	}
}
