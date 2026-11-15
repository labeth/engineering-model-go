// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package integration

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// coffeeView is one view of the coffee fixture with the Mermaid text it must
// contain.
type coffeeView struct {
	modelPath string
	id        string
	anchors   []string
}

// coffeeSVG is the part of a generated SVG document that carries its
// accessibility and dimensions.
type coffeeSVG struct {
	XMLName xml.Name `xml:"svg"`
	Role    string   `xml:"role,attr"`
	Label   string   `xml:"aria-labelledby,attr"`
	ViewBox string   `xml:"viewBox,attr"`
	Title   string   `xml:"title"`
}

func (d coffeeSVG) accessible() bool {
	return d.Role == "img" && d.Label != "" && d.Title != ""
}

func (d coffeeSVG) hasDimensions() bool {
	return d.ViewBox != "" && d.ViewBox != "0 0 0 0"
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func TestCoffeeApplianceSixViewFixture(t *testing.T) {
	root := filepath.Join("examples", "coffee-appliance-six-view")
	modelPath := filepath.Join(root, "engmod.yml")
	expected := map[string][]string{
		"VIEW-COFFEE-USE-CASE":       {"flowchart LR", "Coffee Drinker", "include", "Prepare Coffee"},
		"VIEW-COFFEE-LOGICAL":        {"flowchart TB", "Make Drink", "Brew Beverage", "Control Water Temperature"},
		"VIEW-COFFEE-PROCESS":        {"sequenceDiagram", "autonumber", "ITEM-BREW-SELECTION", "N_SW_TEMPERATURE_CONTROLLER->>N_SW_TEMPERATURE_CONTROLLER"},
		"VIEW-COFFEE-PHYSICAL":       {"flowchart LR", "Thermoblock Heater", "Pump to Heater Line", "Brew Temperature"},
		"VIEW-COFFEE-IMPLEMENTATION": {"classDiagram", "Appliance Controller", "commandPort", "Brew Coordinator"},
		"VIEW-COFFEE-DEPLOYMENT":     {"flowchart LR", "Countertop Device", "deployed to", "Appliance Controller"},
	}
	for viewID, anchors := range expected {
		assertCoffeeView(t, coffeeView{modelPath: modelPath, id: viewID, anchors: anchors})
	}
	assertCoffeeSysML(t, modelPath)
	assertNoAuthoredSysML(t, root)
}

// assertCoffeeView checks the Mermaid anchors, the SVG determinism, the SVG
// accessibility and the provenance of one view.
func assertCoffeeView(t *testing.T, view coffeeView) {
	t.Helper()
	result, err := engmodel.GenerateFromFile(view.modelPath, view.id)
	if err != nil {
		t.Fatalf("%s: %v", view.id, err)
	}
	for _, anchor := range view.anchors {
		if !strings.Contains(result.Mermaid, anchor) {
			t.Fatalf("%s Mermaid missing %q:\n%s", view.id, anchor, result.Mermaid)
		}
	}
	if result.SVG != engmodelSVGAgain(t, view.modelPath, view.id) {
		t.Fatalf("%s SVG bytes are not deterministic", view.id)
	}
	assertCoffeeSVG(t, view, result.SVG)
}

// assertCoffeeSVG checks the accessibility, dimensions and provenance of the
// SVG of one view.
func assertCoffeeSVG(t *testing.T, view coffeeView, svg string) {
	t.Helper()
	var document coffeeSVG
	if err := xml.Unmarshal([]byte(svg), &document); err != nil {
		t.Fatalf("%s invalid SVG: %v", view.id, err)
	}
	if !document.accessible() || !document.hasDimensions() {
		t.Fatalf("%s SVG lacks accessibility or dimensions: %+v", view.id, document)
	}
	if !strings.Contains(svg, `data-source="`+view.modelPath+`"`) || !strings.Contains(svg, `data-view="`+view.id+`"`) {
		t.Fatalf("%s SVG lacks source/view provenance", view.id)
	}
}

// assertCoffeeSysML checks that the generated SysML holds the fixture anchors.
func assertCoffeeSysML(t *testing.T, modelPath string) {
	t.Helper()
	sysml, err := engmodel.GenerateSysMLV2FromFile(modelPath)
	if err != nil {
		t.Fatalf("generate SysML: %v", err)
	}
	for _, anchor := range []string{"use case def 'UC-PREPARE-COFFEE'", "part def 'HW-APPLIANCE-CHASSIS'", "flow 'MSG-03-CONTROL-LOOP'", "attribute def 'QTY-BREW-TEMPERATURE'"} {
		if !strings.Contains(sysml.Text, anchor) {
			t.Fatalf("SysML missing %q", anchor)
		}
	}
}

// assertNoAuthoredSysML fails on a SysML file outside a generated directory.
func assertNoAuthoredSysML(t *testing.T, root string) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() || !strings.EqualFold(filepath.Ext(path), ".sysml") {
			return nil
		}
		if !strings.Contains(filepath.ToSlash(path), "/generated/") {
			t.Fatalf("authored SysML input found: %s", path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TRLC-LINKS: REQ-EMG-012, REQ-EMG-052
func engmodelSVGAgain(t *testing.T, modelPath, viewID string) string {
	t.Helper()
	result, err := engmodel.GenerateFromFile(modelPath, viewID)
	if err != nil {
		t.Fatal(err)
	}
	return result.SVG
}

// TRLC-LINKS: REQ-EMG-052
func TestCoffeeApplianceGeneratedArtifactsMatch(t *testing.T) {
	root := filepath.Join("examples", "coffee-appliance-six-view")
	for _, viewID := range []string{"VIEW-COFFEE-USE-CASE", "VIEW-COFFEE-LOGICAL", "VIEW-COFFEE-PROCESS", "VIEW-COFFEE-PHYSICAL", "VIEW-COFFEE-IMPLEMENTATION", "VIEW-COFFEE-DEPLOYMENT"} {
		assertGeneratedArtifactsCurrent(t, root, viewID)
	}
}

// assertGeneratedArtifactsCurrent checks that the committed .mmd and .svg of a
// view equal the freshly generated ones.
func assertGeneratedArtifactsCurrent(t *testing.T, root, viewID string) {
	t.Helper()
	result, err := engmodel.GenerateFromFile(filepath.Join(root, "engmod.yml"), viewID)
	if err != nil {
		t.Fatal(err)
	}
	for extension, want := range map[string]string{".mmd": result.Mermaid, ".svg": result.SVG} {
		got, err := os.ReadFile(filepath.Join(root, "generated", viewID+extension))
		if err != nil {
			t.Fatal(err)
		}
		if !bytes.Equal(got, []byte(want)) {
			t.Fatalf("%s%s is stale", viewID, extension)
		}
	}
}
