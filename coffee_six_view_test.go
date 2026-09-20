// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package engmodel_test

import (
	"bytes"
	"encoding/xml"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

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
		result, err := engmodel.GenerateFromFile(modelPath, viewID)
		if err != nil {
			t.Fatalf("%s: %v", viewID, err)
		}
		for _, anchor := range anchors {
			if !strings.Contains(result.Mermaid, anchor) {
				t.Fatalf("%s Mermaid missing %q:\n%s", viewID, anchor, result.Mermaid)
			}
		}
		if result.SVG != engmodelSVGAgain(t, modelPath, viewID) {
			t.Fatalf("%s SVG bytes are not deterministic", viewID)
		}
		var document struct {
			XMLName xml.Name `xml:"svg"`
			Role    string   `xml:"role,attr"`
			Label   string   `xml:"aria-labelledby,attr"`
			ViewBox string   `xml:"viewBox,attr"`
			Title   string   `xml:"title"`
		}
		if err := xml.Unmarshal([]byte(result.SVG), &document); err != nil {
			t.Fatalf("%s invalid SVG: %v", viewID, err)
		}
		if document.Role != "img" || document.Label == "" || document.ViewBox == "" || document.ViewBox == "0 0 0 0" || document.Title == "" {
			t.Fatalf("%s SVG lacks accessibility or dimensions: %+v", viewID, document)
		}
		if !strings.Contains(result.SVG, `data-source="`+modelPath+`"`) || !strings.Contains(result.SVG, `data-view="`+viewID+`"`) {
			t.Fatalf("%s SVG lacks source/view provenance", viewID)
		}
	}

	sysml, err := engmodel.GenerateSysMLV2FromFile(modelPath)
	if err != nil {
		t.Fatalf("generate SysML: %v", err)
	}
	for _, anchor := range []string{"use case def 'UC-PREPARE-COFFEE'", "part def 'HW-APPLIANCE-CHASSIS'", "flow 'MSG-03-CONTROL-LOOP'", "attribute def 'QTY-BREW-TEMPERATURE'"} {
		if !strings.Contains(sysml.Text, anchor) {
			t.Fatalf("SysML missing %q", anchor)
		}
	}
	err = filepath.WalkDir(root, func(path string, entry os.DirEntry, walkErr error) error {
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
}
