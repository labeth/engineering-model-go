// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-003
func TestFunctionalPublicationScopePreservesParentsWithoutSiblings(t *testing.T) {
	a := model.AuthoredArchitecture{
		FunctionalGroups:   []model.FunctionalGroup{{ID: "FG-A"}, {ID: "FG-B"}},
		FunctionalUnits:    []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A"}, {ID: "FU-SIBLING", Group: "FG-A"}, {ID: "FU-B", Group: "FG-B"}},
		Actors:             []model.Actor{{ID: "ACT-A"}, {ID: "ACT-B"}},
		ReferencedElements: []model.ReferencedElement{{ID: "REF-A"}, {ID: "REF-B"}},
		Mappings:           []model.Mapping{{From: "ACT-A", To: "FU-A"}, {From: "FU-A", To: "REF-A"}, {From: "FU-A", To: "FU-SIBLING"}, {From: "ACT-B", To: "FU-B"}},
	}
	selected := map[string]bool{"FU-A": true, "ACT-A": true, "REF-A": true}
	scoped := functionalPublicationArchitecture(a, selected)
	if len(scoped.FunctionalUnits) != 1 || len(scoped.FunctionalGroups) != 1 || scoped.FunctionalGroups[0].ID != "FG-A" || len(scoped.Actors) != 1 || len(scoped.ReferencedElements) != 1 || len(scoped.Mappings) != 2 {
		t.Fatalf("incorrect scope: %+v", scoped)
	}
	if len(a.FunctionalUnits) != 3 || len(selected) != 3 {
		t.Fatal("scoping mutated original model or projection")
	}
	if strings.Contains(buildFunctionalDecompositionMermaid(scoped), "SIBLING") {
		t.Fatal("sibling leaked into decomposition")
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestArchitecturePublicationIncludesHardwareProjectionAndDistinctHeadings(t *testing.T) {
	bundle := model.Bundle{ArchitecturePath: filepath.Join(t.TempDir(), "engmod.yml"), Architecture: model.ArchitectureDocument{
		Model: model.ModelMeta{ID: "scope", Title: "Scope Example"},
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Name: "Visible unit", Group: "FG-A"}, {ID: "FU-SIBLING", Name: "Unselected sibling", Group: "FG-A"}},
			HardwareItems:    []model.HardwareItem{{ID: "HW-ADC", Name: "ADC hardware", Kind: "sensor"}},
		},
		Views: []model.View{{ID: "VIEW-LOGIC", Kind: "architecture-intent", Roots: []string{"FU-A"}}, {ID: "VIEW-HARDWARE", Kind: "deployment", Roots: []string{"HW-ADC"}, IncludeKinds: []string{"hardware_item"}}, {ID: "VIEW-OTHER", Kind: "architecture-intent", Roots: []string{"FU-SIBLING"}}},
	}}
	result, err := GenerateAsciiDoc(schemaV2TestBundle(bundle), schemaV2TestRequirements(model.RequirementsDocument{}), schemaV2TestDesign(model.DesignDocument{}), AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generation: %v %+v", err, result.Diagnostics)
	}
	doc := result.Document
	for _, want := range []string{":showtitle:", "== Deployment View", "== Architecture Intent View (VIEW-LOGIC)", "=== Authored View Diagram", "ADC hardware"} {
		if !strings.Contains(doc, want) {
			t.Fatalf("missing %q", want)
		}
	}
	hardware := sectionByHeading(doc, "== Deployment View")
	if !strings.Contains(hardware, "ADC hardware") || !strings.Contains(hardware, "=== Authored View Diagram") {
		t.Fatal("hardware projection missing from its deployment chapter")
	}
	if strings.Contains(hardware, "=== System Decomposition Diagram") {
		t.Fatal("hardware-only view gained unrelated functional decomposition")
	}
	logic := sectionByHeading(doc, "== Architecture Intent View (VIEW-LOGIC)")
	if strings.Contains(logic, "FU_SIBLING[") || !strings.Contains(logic, "FU_FU_A[") {
		t.Fatalf("incorrect logical diagram scope")
	}
	if !strings.Contains(doc, "Unselected sibling") {
		t.Fatal("complete reference material was lost")
	}
	if publicationViewHeading("deployment", "VIEW-ONLY", 1) != "Deployment View" {
		t.Fatal("single-view heading compatibility changed")
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestPublicationKeepsAuthoredScopeWithEachView(t *testing.T) {
	bundle := model.Bundle{ArchitecturePath: filepath.Join(t.TempDir(), "engmod.yml"), Architecture: model.ArchitectureDocument{
		Model:                model.ModelMeta{ID: "scope", Title: "Authored scope"},
		AuthoredArchitecture: model.AuthoredArchitecture{TrustBoundaries: []model.TrustBoundary{{ID: "TB-A", Name: "Boundary"}}},
		Views: []model.View{
			{ID: "VIEW-MEMBERS", Kind: "security", Roots: []string{"TB-A"}, AuthoredStatus: "in-review", AuthoredStatusExplanation: "Membership only; enforcement remains unverified."},
			{ID: "VIEW-OTHER", Kind: "security", Roots: []string{"TB-A"}},
		},
	}}
	result, err := GenerateAsciiDoc(schemaV2TestBundle(bundle), schemaV2TestRequirements(model.RequirementsDocument{}), schemaV2TestDesign(model.DesignDocument{}), AsciiDocOptions{})
	if err != nil {
		t.Fatalf("generation: %v", err)
	}
	section := sectionByHeading(result.Document, "== Security View (VIEW-MEMBERS)")
	explanation := "*Authored scope and limitations:* Membership only; enforcement remains unverified."
	position := strings.Index(section, explanation)
	if position < 0 || position > strings.Index(section, "=== Authored View Diagram") || !strings.Contains(section, "*Authored status:* in-review") {
		t.Fatalf("scope missing before diagram: %s", section)
	}
	other := sectionByHeading(result.Document, "== Security View (VIEW-OTHER)")
	if strings.Contains(other, "Membership only;") || !strings.Contains(other, "*Authored status:* unspecified") || !strings.Contains(other, "No authored status explanation provided.") {
		t.Fatalf("scope leaked or fallback missing: %s", other)
	}
	if strings.Contains(section, "focused on attack paths and security evidence") {
		t.Fatal("generic prose overstates projected evidence")
	}
}
