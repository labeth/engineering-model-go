// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engine

import (
	"fmt"
	"math/rand"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
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

	runtime := []codescan.RuntimeItem{{Name: "unit-a-runtime", Kind: "service", Owner: "FU-A"}}
	code := []codescan.CodeItem{
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
	if got := strings.Count(out, `<svg `); got != 1 {
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
func TestBuildFunctionalManhattanTable_CompleteOverviewIncludesAllColumnsAndUnits(t *testing.T) {
	a := testMatrixArchitecture(19, 20)

	out := buildFunctionalManhattanTable(a)

	if strings.Contains(out, "*FG Columns") || strings.Count(out, "<svg ") != 1 {
		t.Fatal("all groups must share one complete vector overview")
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
func TestBuildRequirementCoverageMermaid_GroupsCodeNodesByFile(t *testing.T) {
	reqs := []model.Requirement{{ID: "REQ-A", AppliesTo: []string{"FU-A"}}}
	code := []codescan.CodeItem{
		{Kind: "symbol", Owner: "FU-A", Source: "src/payment_engine.rs:25", Implements: []string{"REQ-A"}},
		{Kind: "symbol", Owner: "FU-A", Source: "src/payment_engine.rs:11", Implements: []string{"REQ-A"}},
		{Kind: "source_file", Owner: "FU-A", Source: "src/ai_view_schema.go"},
	}
	verification := []codescan.VerificationCheck{{
		ID:           "VER-A",
		Status:       "pass",
		Verifies:     []string{"REQ-A"},
		CodeElements: []string{"tests/unit/payment_engine_test.go:30", "tests/unit/payment_engine_test.go:12"},
	}}

	out := buildRequirementCoverageMermaid(reqs, nil, code, verification, map[string]string{"FU-A": "Unit A"}, nil)

	if got := strings.Count(out, `["payment_engine.rs: 11, 25"]:::code_element`); got != 1 {
		t.Fatalf("expected one implementation code box with comma-separated lines, got %d:\n%s", got, out)
	}
	if got := strings.Count(out, `["payment_engine_test.go: 12, 30"]:::code_element`); got != 1 {
		t.Fatalf("expected one verification code box with comma-separated lines, got %d:\n%s", got, out)
	}
	if strings.Contains(out, `["payment_engine.rs: 11"]`) || strings.Contains(out, `["payment_engine.rs: 25"]`) {
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
func TestBuildRequirementAlignmentCompactTable_KeepsAllFunctionalUnitColumnsTogether(t *testing.T) {
	reqs := []model.Requirement{
		{ID: "REQ-A", AppliesTo: []string{"FU-01", "FU-02", "FU-03", "FU-04", "FU-05", "FU-06", "FU-07", "FU-08", "FU-09", "FU-10"}},
		{ID: "REQ-B", AppliesTo: []string{"FU-02", "FU-09"}},
	}

	out := buildRequirementAlignmentCompactTable(reqs)

	if strings.Contains(out, "*Functional Unit Columns") || strings.Count(out, "<svg ") != 1 {
		t.Fatal("all mappings must share one complete vector overview")
	}
	if strings.Count(out, ">X</text>") != 12 {
		t.Fatal("mapping lost or duplicated")
	}

	for _, label := range []string{"FU-01", "FU-07", "FU-08", "FU-10"} {
		if !strings.Contains(out, ">"+label+"</text>") {
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

// TRLC-LINKS: REQ-EMG-003
func TestCodeEvidenceAcrossIndependentSourceRoots(t *testing.T) {
	code := []codescan.CodeItem{
		{Kind: "symbol", Owner: "FU-APP", Source: "server.go:412", AbsPath: "/workspace/app/server.go", Implements: []string{"REQ-APP"}},
		{Kind: "symbol", Owner: "FU-BROKER", Source: "server.go:16", AbsPath: "/workspace/ota/server.go", Implements: []string{"REQ-BROKER"}},
	}

	diagram := buildRequirementCoverageMermaid([]model.Requirement{{ID: "REQ-BROKER", AppliesTo: []string{"FU-BROKER"}}}, nil, code, nil, nil, nil)
	if !strings.Contains(diagram, "ota/server.go: 16") || strings.Contains(diagram, "412") {
		t.Fatalf("independent source roots merged in requirement graph: %s", diagram)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestRequirementEvidenceExcludesOtherRequirementLines(t *testing.T) {
	code := []codescan.CodeItem{
		{Kind: "symbol", Owner: "FU-AGENT", Source: "rpc.go:39", AbsPath: "/workspace/agent/rpc.go", Implements: []string{"REQ-RPC"}},
		{Kind: "symbol", Owner: "FU-AGENT", Source: "rpc.go:95", AbsPath: "/workspace/agent/rpc.go", Implements: []string{"REQ-TAKEOVER"}},
		{Kind: "symbol", Owner: "FU-OTHER", Source: "rpc.go:12", AbsPath: "/workspace/other/rpc.go", Implements: []string{"REQ-OTHER"}},
	}
	diagram := buildRequirementCoverageMermaid([]model.Requirement{{ID: "REQ-RPC", AppliesTo: []string{"FU-AGENT"}}}, nil, code, nil, nil, nil)
	if !strings.Contains(diagram, "agent/rpc.go: 39") || strings.Contains(diagram, "95") || strings.Contains(diagram, "rpc.go: 12") {
		t.Fatalf("requirement evidence contains unrelated lines: %s", diagram)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestRequirementAndVerificationSourceNodesDoNotOverwrite(t *testing.T) {
	code := []codescan.CodeItem{
		{Kind: "symbol", Owner: "FU-A", Source: "rpc.go:39", Implements: []string{"REQ-A"}},
		{Kind: "symbol", Owner: "FU-A", Source: "rpc.go:95", Implements: []string{"REQ-B"}},
	}
	checks := []codescan.VerificationCheck{{ID: "VER-A", Verifies: []string{"REQ-A"}, CodeElements: []string{"rpc_test.go:39", "rpc_test.go:95"}}}
	got := buildRequirementCoverageMermaid([]model.Requirement{{ID: "REQ-A", AppliesTo: []string{"FU-A"}}, {ID: "REQ-B", AppliesTo: []string{"FU-A"}}}, nil, code, checks, nil, nil)
	for _, want := range []string{`CODE_REQ_A_RPC_GO["rpc.go: 39"]`, `CODE_REQ_B_RPC_GO["rpc.go: 95"]`, `VERCODE_REQ_A_VER_A_RPC_TEST_GO["rpc_test.go: 39, 95"]`} {
		if !strings.Contains(got, want) {
			t.Fatalf("missing separate evidence node %s: %s", want, got)
		}
	}
}
