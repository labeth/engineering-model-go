// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
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
	sizes := []int{len(scoped.FunctionalUnits), len(scoped.FunctionalGroups), len(scoped.Actors), len(scoped.ReferencedElements), len(scoped.Mappings)}
	if !slices.Equal(sizes, []int{1, 1, 1, 1, 2}) || scoped.FunctionalGroups[0].ID != "FG-A" {
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
	result, err := Generate(schemaV2TestBundle(bundle), schemaV2TestRequirements(model.RequirementsDocument{}), schemaV2TestDesign(model.DesignDocument{}), Options{})
	if err != nil {
		t.Fatalf("generation: %v %+v", err, result.Diagnostics)
	}
	doc := result.Document
	about(t, doc).hasAll("missing %q", ":showtitle:", "== Deployment View", "== Architecture Intent View (VIEW-LOGIC)", "=== Authored View Diagram", "ADC hardware")
	hardware := sectionByHeading(doc, "== Deployment View")
	about(t, hardware).has(
		docCheck{"ADC hardware", "hardware projection missing from its deployment chapter"},
		docCheck{"=== Authored View Diagram", "hardware projection missing from its deployment chapter"},
	)
	about(t, hardware).lacks(docCheck{"=== System Decomposition Diagram", "hardware-only view gained unrelated functional decomposition"})
	logic := sectionByHeading(doc, "== Architecture Intent View (VIEW-LOGIC)")
	about(t, logic).lacks(docCheck{"FU_SIBLING[", "incorrect logical diagram scope"})
	about(t, logic).has(docCheck{"FU_FU_A[", "incorrect logical diagram scope"})
	about(t, doc).has(docCheck{"Unselected sibling", "complete reference material was lost"})
	if publicationViewHeading("deployment", "VIEW-ONLY", 1) != "Deployment View" {
		t.Fatal("single-view heading compatibility changed")
	}
}

// scopeBeforeDiagram reports whether the explanation appears in the section
// before its first diagram, together with the authored status.
func scopeBeforeDiagram(section, explanation, status string) bool {
	position := strings.Index(section, explanation)
	return position >= 0 && position <= strings.Index(section, "=== Authored View Diagram") && strings.Contains(section, status)
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
	result, err := Generate(schemaV2TestBundle(bundle), schemaV2TestRequirements(model.RequirementsDocument{}), schemaV2TestDesign(model.DesignDocument{}), Options{})
	if err != nil {
		t.Fatalf("generation: %v", err)
	}
	section := sectionByHeading(result.Document, "== Security View (VIEW-MEMBERS)")
	if !scopeBeforeDiagram(section, "*Authored scope and limitations:* Membership only; enforcement remains unverified.", "*Authored status:* in-review") {
		t.Fatalf("scope missing before diagram: %s", section)
	}
	other := sectionByHeading(result.Document, "== Security View (VIEW-OTHER)")
	leaked := "scope leaked or fallback missing: " + other
	about(t, other).lacks(docCheck{"Membership only;", leaked})
	about(t, other).has(docCheck{"*Authored status:* unspecified", leaked}, docCheck{"No authored status explanation provided.", leaked})
	about(t, section).lacks(docCheck{"focused on attack paths and security evidence", "generic prose overstates projected evidence"})
}
