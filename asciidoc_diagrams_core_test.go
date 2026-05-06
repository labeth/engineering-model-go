// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-003
func TestBuildFunctionalContextMermaid_LinksExternalActorsAndReferences(t *testing.T) {
	a := model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Name: "Unit A", Group: "FG-A"}},
		Actors:           []model.Actor{{ID: "ACT-A", Name: "Actor A"}},
		ReferencedElements: []model.ReferencedElement{
			{ID: "REF-GO-TOOLCHAIN", Name: "Go Toolchain"},
		},
		Mappings: []model.Mapping{
			{Type: "interacts_with", From: "ACT-A", To: "FU-A", Description: "Uses unit A."},
			{Type: "depends_on", From: "FU-A", To: "REF-GO-TOOLCHAIN", Description: "Runs Go commands."},
		},
	}

	out := buildFunctionalContextMermaid(a)

	for _, want := range []string{
		`ACT_ACT_A -->|Uses unit A.| FU_FU_A`,
		`FU_FU_A -->|Runs Go commands.| REF_REF_GO_TOOLCHAIN`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("system boundary diagram missing edge %q:\n%s", want, out)
		}
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuildFunctionalGroupDependencyMermaid_ShowsOutgoingDependenciesOnly(t *testing.T) {
	a := model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{
			{ID: "FG-A", Name: "Group A"},
			{ID: "FG-B", Name: "Group B"},
		},
		FunctionalUnits: []model.FunctionalUnit{
			{ID: "FU-A", Name: "Unit A", Group: "FG-A"},
			{ID: "FU-B", Name: "Unit B", Group: "FG-B"},
			{ID: "FU-C", Name: "Unit C", Group: "FG-B"},
		},
		ReferencedElements: []model.ReferencedElement{{ID: "REF-GO-TOOLCHAIN", Name: "Go Toolchain"}},
		Mappings: []model.Mapping{
			{Type: "depends_on", From: "FU-A", To: "FU-B"},
			{Type: "depends_on", From: "FU-A", To: "REF-GO-TOOLCHAIN"},
			{Type: "depends_on", From: "FU-C", To: "FU-A"},
		},
	}

	runtime := []inferredRuntimeItem{{Name: "unit-a-runtime", Kind: "service", Owner: "FU-A"}}
	code := []inferredCodeItem{
		{Kind: "symbol", Owner: "FU-A", Source: "src/unit_a.go:12"},
		{Kind: "source_file", Owner: "FU-B", Source: "src/unit_b.go"},
	}

	out := buildFunctionalGroupDependencyMermaid(a, "FG-A", runtime, code)

	for _, want := range []string{
		`subgraph FGDEP_FG_A["Group A"]`,
		`subgraph FGDEP_TARGET_FG_B["Group B"]`,
		`FU_FU_A["Unit A"]:::functional_unit`,
		`FU_FU_B["Unit B"]:::functional_unit`,
		`REF_REF_GO_TOOLCHAIN["Go Toolchain"]:::referenced_element`,
		`FU_FU_A -->|depends_on| FU_FU_B`,
		`FU_FU_A -->|depends_on| REF_REF_GO_TOOLCHAIN`,
		`subgraph RT_FU_A_UNIT_A_RUNTIME["unit-a-runtime"]`,
		`FU_FU_A -->|runtime evidence| RT_FU_A_UNIT_A_RUNTIME`,
		`CODE_UNIT_A_GO["unit_a.go"]:::code_element`,
	} {
		if !strings.Contains(out, want) {
			t.Fatalf("functional group dependency diagram missing %q:\n%s", want, out)
		}
	}
	for _, unwanted := range []string{
		`unit_a.go:12`,
		`implemented_by`,
		`code evidence`,
		`FU_FU_B -->|runtime evidence| RT_FU_B_RUNTIME`,
		`CODE_UNIT_B_GO["unit_b.go"]:::code_element`,
	} {
		if strings.Contains(out, unwanted) {
			t.Fatalf("functional group dependency diagram should not include %q:\n%s", unwanted, out)
		}
	}
	if strings.Contains(out, "FU_FU_C -->|depends_on| FU_FU_A") {
		t.Fatalf("functional group dependency diagram should not include incoming dependencies:\n%s", out)
	}
	localStart := strings.Index(out, `subgraph FGDEP_FG_A["Group A"]`)
	targetStart := strings.Index(out, `subgraph FGDEP_TARGET_FG_B["Group B"]`)
	if localStart < 0 || targetStart < 0 || targetStart <= localStart {
		t.Fatalf("expected local FG subgraph before target FG subgraph:\n%s", out)
	}
	localBlock := out[localStart:targetStart]
	for _, want := range []string{
		`subgraph RT_FU_A_UNIT_A_RUNTIME["unit-a-runtime"]`,
		`CODE_UNIT_A_GO["unit_a.go"]:::code_element`,
	} {
		if !strings.Contains(localBlock, want) {
			t.Fatalf("expected local FG subgraph to contain %q:\n%s", want, out)
		}
	}
	runtimeStart := strings.Index(localBlock, `subgraph RT_FU_A_UNIT_A_RUNTIME["unit-a-runtime"]`)
	if runtimeStart < 0 {
		t.Fatalf("expected runtime subgraph inside local FG subgraph:\n%s", out)
	}
	runtimeEnd := strings.Index(localBlock[runtimeStart:], "\n    end")
	if runtimeEnd < 0 {
		t.Fatalf("expected runtime subgraph to close inside local FG subgraph:\n%s", out)
	}
	runtimeBlock := localBlock[runtimeStart : runtimeStart+runtimeEnd]
	if !strings.Contains(runtimeBlock, `CODE_UNIT_A_GO["unit_a.go"]:::code_element`) {
		t.Fatalf("expected code box inside runtime subgraph:\n%s", out)
	}
	targetEnd := strings.Index(out[targetStart:], "\n  end")
	if targetEnd < 0 {
		t.Fatalf("expected target FG subgraph to close:\n%s", out)
	}
	targetBlock := out[targetStart : targetStart+targetEnd]
	if strings.Contains(targetBlock, ":::runtime_element") || strings.Contains(targetBlock, ":::code_element") {
		t.Fatalf("target FG subgraph should only contain dependency FUs:\n%s", out)
	}
}

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
func TestBuildFunctionalManhattanTable_MultiBandIncludesAllColumnsAndUnits(t *testing.T) {
	a := testMatrixArchitecture(19, 20)

	out := buildFunctionalManhattanTable(a)

	if !strings.Contains(out, "*FG Columns 1-8*") {
		t.Fatalf("missing first multi-band heading")
	}
	if !strings.Contains(out, "*FG Columns 9-16*") {
		t.Fatalf("missing second multi-band heading")
	}
	if !strings.Contains(out, "*FG Columns 17-19*") {
		t.Fatalf("missing third multi-band heading")
	}
	if got := strings.Count(out, `<table style="width:100%;table-layout:fixed;border-collapse:collapse;border-spacing:0;margin:0;`); got != 3 {
		t.Fatalf("expected three html band tables, got %d", got)
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

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
func TestCodeEvidenceLabelsPreserveFileExtensionsAndSymbolLines(t *testing.T) {
	file := inferredCodeItem{Kind: "source_file", Element: "src/checkout_api.go", Source: "src/checkout_api.go"}
	if got := codeItemEvidenceLabel(file); got != "checkout_api.go" {
		t.Fatalf("expected file label with extension, got %q", got)
	}

	symbol := inferredCodeItem{Kind: "symbol", Element: "CODE-AUTHORIZE", Source: "src/payment_engine.rs:25"}
	if got := codeItemDisplayName(symbol); got != "payment_engine.rs:25" {
		t.Fatalf("expected function-level symbol display, got %q", got)
	}
	if got := codeItemEvidenceLabel(symbol); got != "payment_engine.rs:25" {
		t.Fatalf("expected symbol evidence label with file extension and line, got %q", got)
	}
	if got := sanitizeNode(codeItemDisplayName(symbol)); strings.ContainsAny(got, "()[]") {
		t.Fatalf("expected mermaid-safe symbol node id, got %q", got)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestGroupedCodeElementEvidenceLabels_OneLabelPerFileWithCommaLines(t *testing.T) {
	got := groupedCodeElementEvidenceLabels([]string{
		"src/payment_engine.rs:25",
		"src/payment_engine.rs:11",
		"src/payment_engine.rs:25",
		"src/checkout_api.go",
	}, nil)

	want := []string{"checkout_api.go", "payment_engine.rs:11,25"}
	if len(got) != len(want) {
		t.Fatalf("unexpected grouped labels: got %+v want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected grouped label at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuildRequirementCoverageMermaid_GroupsCodeNodesByFile(t *testing.T) {
	reqs := []model.Requirement{{ID: "REQ-A", AppliesTo: []string{"FU-A"}}}
	code := []inferredCodeItem{
		{Kind: "symbol", Owner: "FU-A", Source: "src/payment_engine.rs:25", Implements: []string{"REQ-A"}},
		{Kind: "symbol", Owner: "FU-A", Source: "src/payment_engine.rs:11", Implements: []string{"REQ-A"}},
		{Kind: "source_file", Owner: "FU-A", Source: "src/ai_view_schema.go"},
	}
	verification := []inferredVerificationCheck{{
		ID:           "VER-A",
		Status:       "pass",
		Verifies:     []string{"REQ-A"},
		CodeElements: []string{"tests/unit/payment_engine_test.go:30", "tests/unit/payment_engine_test.go:12"},
	}}

	out := buildRequirementCoverageMermaid(reqs, nil, code, verification, map[string]string{"FU-A": "Unit A"})

	if got := strings.Count(out, `["payment_engine.rs:11,25"]:::code_element`); got != 1 {
		t.Fatalf("expected one implementation code box with comma-separated lines, got %d:\n%s", got, out)
	}
	if got := strings.Count(out, `["payment_engine_test.go:12,30"]:::code_element`); got != 1 {
		t.Fatalf("expected one verification code box with comma-separated lines, got %d:\n%s", got, out)
	}
	if strings.Contains(out, `["payment_engine.rs:11"]`) || strings.Contains(out, `["payment_engine.rs:25"]`) {
		t.Fatalf("did not expect separate implementation boxes per line:\n%s", out)
	}
	if strings.Contains(out, "ai_view_schema.go") {
		t.Fatalf("did not expect owner-only source files without TRLC-LINKS in requirement coverage:\n%s", out)
	}
	if strings.Contains(out, "CODE_PAYMENT_ENGINE_RS_") {
		t.Fatalf("did not expect line numbers in Mermaid code node IDs:\n%s", out)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestBuildRequirementAlignmentCompactTable_BandsFunctionalUnitColumns(t *testing.T) {
	reqs := []model.Requirement{
		{ID: "REQ-A", AppliesTo: []string{"FU-01", "FU-02", "FU-03", "FU-04", "FU-05", "FU-06", "FU-07", "FU-08", "FU-09", "FU-10"}},
		{ID: "REQ-B", AppliesTo: []string{"FU-02", "FU-09"}},
	}

	out := buildRequirementAlignmentCompactTable(reqs)

	if !strings.Contains(out, "*Functional Unit Columns 1-7*") {
		t.Fatalf("missing first requirement mapping band heading:\n%s", out)
	}
	if !strings.Contains(out, "*Functional Unit Columns 8-10*") {
		t.Fatalf("missing second requirement mapping band heading:\n%s", out)
	}
	if got := strings.Count(out, "[cols=\""); got != 2 {
		t.Fatalf("expected two requirement mapping tables, got %d:\n%s", got, out)
	}
	if strings.Contains(out, "[cols=\"2,1,1,1,1,1,1,1,1") {
		t.Fatalf("requirement mapping table exceeded 8 total columns:\n%s", out)
	}
	for _, label := range []string{"FU-01", "FU-07", "FU-08", "FU-10"} {
		if !strings.Contains(out, "|"+label) {
			t.Fatalf("missing functional unit %s in banded requirement table:\n%s", label, out)
		}
	}
}

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
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
