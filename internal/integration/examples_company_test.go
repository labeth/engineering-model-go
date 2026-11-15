// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package integration

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/module"
	"cuelang.org/go/mod/modzip"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TestAtlasIndustriesCompanyExample proves that every simulated repository is a
// valid standalone model and that workspace replacements resolve the complete
// company portfolio without registry access.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-097
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
			assertAtlasRepository(t, root, repository)
		})
	}

	composed := composeAtlasPortfolio(t, root)
	assertSharedModulesBelowBothProducts(t, composed)
	for _, product := range composed.Children {
		if !inheritsPeopleBaseline(product) {
			t.Fatalf("product %s does not inherit the people compliance baseline", product.Bundle.Architecture.Model.ID)
		}
	}
	assertAtlasRequirementLinks(t, composed)
	assertPeopleComplianceBaseline(t, root)
}

// assertAtlasRepository checks that one repository loads, validates and is a
// publishable module.
func assertAtlasRepository(t *testing.T, root, repository string) {
	t.Helper()
	dir := filepath.Join(root, "repos", repository)
	bundle, err := model.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatalf("load %s: %v", repository, err)
	}
	for _, diagnostic := range validate.Bundle(bundle) {
		if diagnostic.Severity == validate.SeverityError {
			t.Fatalf("%s validation error: %s %s", repository, diagnostic.Code, diagnostic.Message)
		}
	}
	modulePath, err := model.ReadCUEModulePath(dir)
	if err != nil {
		t.Fatalf("read module identity for %s: %v", repository, err)
	}
	version, err := module.NewVersion(modulePath, "v0.1.0")
	if err != nil {
		t.Fatalf("module version for %s: %v", repository, err)
	}
	if report, err := modzip.CheckDir(dir); err != nil {
		t.Fatalf("module %s is not publishable: %v (%+v)", version, err, report)
	}
}

// composeAtlasPortfolio composes the company portfolio and returns its root,
// which has the two product children.
func composeAtlasPortfolio(t *testing.T, root string) *composition.ComposedSystem {
	t.Helper()
	composed, err := composition.GenerateCompositionFromFile(filepath.Join(root, "repos", "company-portfolio", "engmod.yml"))
	if err != nil {
		t.Fatalf("compose portfolio: %v", err)
	}
	for _, diagnostic := range composed.Diagnostics {
		if diagnostic.Severity == validate.SeverityError {
			t.Fatalf("composition error: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
	if composed.Root == nil {
		t.Fatal("portfolio composition has no root")
	}
	if got := len(composed.Root.Children); got != 2 {
		t.Fatalf("expected two product children, got %d", got)
	}
	return composed.Root
}

// assertSharedModulesBelowBothProducts checks that each product composes the
// four shared modules.
func assertSharedModulesBelowBothProducts(t *testing.T, portfolio *composition.ComposedSystem) {
	t.Helper()
	shared := sharedModuleUses(t, portfolio)
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
}

// sharedModuleUses counts, per model ID, how many products compose the module.
func sharedModuleUses(t *testing.T, portfolio *composition.ComposedSystem) map[string]int {
	t.Helper()
	shared := map[string]int{}
	for _, product := range portfolio.Children {
		if got := len(product.Children); got != 4 {
			t.Fatalf("product %s: expected four shared children, got %d", product.Bundle.Architecture.Model.ID, got)
		}
		for _, child := range product.Children {
			shared[child.Bundle.Architecture.Model.ID]++
		}
	}
	return shared
}

// inheritsPeopleBaseline reports whether a product composes the compliance
// baseline that holds the people functional unit.
func inheritsPeopleBaseline(product *composition.ComposedSystem) bool {
	for _, child := range product.Children {
		if child.Bundle.Architecture.Model.ID == "atlas-shared-compliance-baseline" && hasFunctionalUnit(child, "FU-COMP-PEOPLE") {
			return true
		}
	}
	return false
}

// hasFunctionalUnit reports whether a composed module authors the unit.
func hasFunctionalUnit(system *composition.ComposedSystem, id string) bool {
	for _, unit := range system.Bundle.Architecture.AuthoredArchitecture.FunctionalUnits {
		if unit.ID == id {
			return true
		}
	}
	return false
}

// assertAtlasRequirementLinks checks that product requirements link to
// shared-module capabilities and portfolio requirements to product
// capabilities (model/req-links.yml); every link resolves to an unchanged
// published item at the pinned version.
func assertAtlasRequirementLinks(t *testing.T, portfolio *composition.ComposedSystem) {
	t.Helper()
	links := reqlinks.EvaluateLinks(portfolio.Bundle, composition.LinkDependencies(portfolio))
	for _, product := range portfolio.Children {
		links = append(links, reqlinks.EvaluateLinks(product.Bundle, composition.LinkDependencies(product))...)
	}
	if got := len(links); got != 10 {
		t.Fatalf("expected ten requirement links, got %d", got)
	}
	for _, link := range links {
		if link.State != reqlinks.StateUnchanged {
			t.Fatalf("link %s is %s: %+v", link.ID, link.State, link)
		}
	}
}

// assertPeopleComplianceBaseline checks that the compliance baseline holds the
// people controls and complete framework mappings.
func assertPeopleComplianceBaseline(t *testing.T, root string) {
	t.Helper()
	compliance, err := model.LoadBundle(filepath.Join(root, "repos", "shared-compliance-baseline", "engmod.yml"))
	if err != nil {
		t.Fatalf("load compliance baseline: %v", err)
	}
	assertPeopleControls(t, compliance)
	assertPeopleMappings(t, compliance)
}

func assertPeopleControls(t *testing.T, compliance model.Bundle) {
	t.Helper()
	authored := map[string]bool{}
	for _, control := range compliance.Architecture.AuthoredArchitecture.Controls {
		authored[control.ID] = true
	}
	for _, controlID := range []string{
		"CTRL-COMP-PERSONNEL-SCREENING",
		"CTRL-COMP-POLICY-ACKNOWLEDGMENT",
		"CTRL-COMP-SECURITY-AWARENESS",
		"CTRL-COMP-ROLE-CHANGE-REVIEW",
		"CTRL-COMP-OFFBOARDING",
	} {
		if !authored[controlID] {
			t.Errorf("people compliance baseline is missing control %s", controlID)
		}
	}
}

func assertPeopleMappings(t *testing.T, compliance model.Bundle) {
	t.Helper()
	complete := map[string]bool{}
	for _, mapping := range compliance.Architecture.Compliance.Mappings {
		complete[mapping.ID] = len(mapping.ControlIDs) >= 4 && len(mapping.Evidence) >= 2
	}
	for _, mappingID := range []string{"IMPL-COMP-NIST-PEOPLE", "IMPL-COMP-ISO-PEOPLE"} {
		if !complete[mappingID] {
			t.Errorf("people compliance mapping %s is missing framework controls or evidence", mappingID)
		}
	}
}
