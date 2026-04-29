// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

func TestBuildFunctionalManhattanTable_SingleBand(t *testing.T) {
	a := testMatrixArchitecture(4, 9)

	out := buildFunctionalManhattanTable(a)

	if strings.Contains(out, "*FG Columns ") {
		t.Fatalf("single-band layout should not include band headings")
	}
	if got := strings.Count(out, `<table style="width:100%;table-layout:fixed;border-collapse:collapse;border-spacing:0;margin:0;`); got != 1 {
		t.Fatalf("expected exactly one html table, got %d", got)
	}
	for i := 1; i <= 4; i++ {
		label := fmt.Sprintf("Capability FG-%02d", i)
		if !strings.Contains(out, label) {
			t.Fatalf("missing functional group label %q", label)
		}
	}
	for i := 1; i <= 9; i++ {
		label := fmt.Sprintf("Unit FU-%02d", i)
		if !strings.Contains(out, label) {
			t.Fatalf("missing functional unit label %q", label)
		}
	}
}

func TestBuildFunctionalManhattanTable_MultiBandIncludesAllColumnsAndUnits(t *testing.T) {
	a := testMatrixArchitecture(19, 20)

	out := buildFunctionalManhattanTable(a)

	if !strings.Contains(out, "*FG Columns 1-10*") {
		t.Fatalf("missing first multi-band heading")
	}
	if !strings.Contains(out, "*FG Columns 11-19*") {
		t.Fatalf("missing second multi-band heading")
	}
	if got := strings.Count(out, `<table style="width:100%;table-layout:fixed;border-collapse:collapse;border-spacing:0;margin:0;`); got != 2 {
		t.Fatalf("expected two html band tables, got %d", got)
	}

	for i := 1; i <= 19; i++ {
		label := fmt.Sprintf("Capability FG-%02d", i)
		if !strings.Contains(out, label) {
			t.Fatalf("missing functional group label %q", label)
		}
	}
	for i := 1; i <= 20; i++ {
		label := fmt.Sprintf("Unit FU-%02d", i)
		if !strings.Contains(out, label) {
			t.Fatalf("missing functional unit label %q", label)
		}
		if got := strings.Count(out, label); got != 1 {
			t.Fatalf("expected functional unit label %q exactly once, got %d", label, got)
		}
	}
}

func TestBuildFunctionalManhattanTable_RandomizedCoverage(t *testing.T) {
	r := rand.New(rand.NewSource(20260407))
	for caseIdx := 0; caseIdx < 10; caseIdx++ {
		fgCount := 3 + r.Intn(18) // 3..20
		fuCount := fgCount + r.Intn(21-fgCount)
		a := testRandomMatrixArchitecture(r, fgCount, fuCount)
		out := buildFunctionalManhattanTable(a)

		t.Run(fmt.Sprintf("case_%02d_fg_%02d_fu_%02d", caseIdx+1, fgCount, fuCount), func(t *testing.T) {
			for i := 1; i <= fgCount; i++ {
				label := fmt.Sprintf("Capability FG-%02d", i)
				if !strings.Contains(out, label) {
					t.Fatalf("missing functional group label %q", label)
				}
			}
			for i := 1; i <= fuCount; i++ {
				label := fmt.Sprintf("Unit FU-%02d", i)
				if !strings.Contains(out, label) {
					t.Fatalf("missing functional unit label %q", label)
				}
			}
		})
	}
}

func TestBuildFunctionalManhattanTable_BottomAlignedAndNoRenderedEmptyBlocks(t *testing.T) {
	a := model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{
			{ID: "FG-A", Name: "Capability A"},
			{ID: "FG-B", Name: "Capability B"},
		},
		FunctionalUnits: []model.FunctionalUnit{
			{ID: "FU-A1", Name: "Unit A1", Group: "FG-A"},
			{ID: "FU-B1", Name: "Unit B1", Group: "FG-B"},
			{ID: "FU-B2", Name: "Unit B2", Group: "FG-B"},
			{ID: "FU-B3", Name: "Unit B3", Group: "FG-B"},
		},
	}

	out := buildFunctionalManhattanTable(a)

	if strings.Contains(out, "&nbsp;") {
		t.Fatalf("did not expect styled placeholder blocks for empty Manhattan cells")
	}
	idxA1 := strings.Index(out, "Unit A1")
	idxB1 := strings.Index(out, "Unit B1")
	idxB2 := strings.Index(out, "Unit B2")
	if idxA1 < 0 || idxB1 < 0 || idxB2 < 0 {
		t.Fatalf("expected all unit labels to exist in output")
	}
	if idxA1 < idxB1 || idxA1 < idxB2 {
		t.Fatalf("expected short column unit to be bottom-aligned below higher-column upper rows")
	}
}

func testMatrixArchitecture(fgCount, fuCount int) model.AuthoredArchitecture {
	groups := make([]model.FunctionalGroup, 0, fgCount)
	for i := 1; i <= fgCount; i++ {
		id := fmt.Sprintf("FG-%02d", i)
		groups = append(groups, model.FunctionalGroup{
			ID:   id,
			Name: fmt.Sprintf("Capability %s", id),
		})
	}

	units := make([]model.FunctionalUnit, 0, fuCount)
	for i := 1; i <= fuCount; i++ {
		unitID := fmt.Sprintf("FU-%02d", i)
		groupID := fmt.Sprintf("FG-%02d", ((i-1)%fgCount)+1)
		units = append(units, model.FunctionalUnit{
			ID:    unitID,
			Name:  fmt.Sprintf("Unit %s", unitID),
			Group: groupID,
		})
	}

	return model.AuthoredArchitecture{
		FunctionalGroups: groups,
		FunctionalUnits:  units,
	}
}

func testRandomMatrixArchitecture(r *rand.Rand, fgCount, fuCount int) model.AuthoredArchitecture {
	groups := make([]model.FunctionalGroup, 0, fgCount)
	for i := 1; i <= fgCount; i++ {
		id := fmt.Sprintf("FG-%02d", i)
		groups = append(groups, model.FunctionalGroup{
			ID:   id,
			Name: fmt.Sprintf("Capability %s", id),
		})
	}

	units := make([]model.FunctionalUnit, 0, fuCount)
	for i := 1; i <= fuCount; i++ {
		unitID := fmt.Sprintf("FU-%02d", i)
		groupID := fmt.Sprintf("FG-%02d", 1+r.Intn(fgCount))
		units = append(units, model.FunctionalUnit{
			ID:    unitID,
			Name:  fmt.Sprintf("Unit %s", unitID),
			Group: groupID,
		})
	}

	return model.AuthoredArchitecture{
		FunctionalGroups: groups,
		FunctionalUnits:  units,
	}
}
