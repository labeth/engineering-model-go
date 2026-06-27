// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TestTraceIntegrityFlagsDanglingLinks verifies a TRLC-LINKS to a non-existent requirement
// and an ENGMODEL-LINKS to a non-existent element are both hard errors.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE
func TestTraceIntegrityFlagsDanglingLinks(t *testing.T) {
	bundle := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalUnits: []model.FunctionalUnit{{ID: "FU-REAL"}},
	}}}
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-X-001"}}}
	code := []inferredCodeItem{
		{Element: "good", Source: "a.go:1", Implements: []string{"REQ-X-001"}, ModelLinks: []string{"FU-REAL"}},
		{Element: "bad", Source: "b.go:2", Implements: []string{"REQ-X-999"}, ModelLinks: []string{"FU-GHOST"}},
	}
	gotReq, gotModel := false, false
	for _, d := range validateTraceIntegrity(bundle, reqs, code, nil, nil) {
		if d.Code == "code.dangling_requirement_link" && d.Severity == validate.SeverityError {
			gotReq = true
		}
		if d.Code == "code.dangling_model_link" && d.Severity == validate.SeverityError {
			gotModel = true
		}
	}
	if !gotReq || !gotModel {
		t.Fatalf("expected dangling requirement and model errors (gotReq=%v gotModel=%v)", gotReq, gotModel)
	}
}

// TestTraceIntegrityFlagsOrphan verifies a requirement traced by nothing is an orphan
// warning, and a covered requirement is not.
// TRLC-LINKS: REQ-EMG-029
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE
func TestTraceIntegrityFlagsOrphan(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-X-001"}, {ID: "REQ-X-002"}}}
	code := []inferredCodeItem{{Element: "c", Source: "a.go:1", Implements: []string{"REQ-X-001"}}}
	orphan := map[string]bool{}
	for _, d := range validateTraceIntegrity(model.Bundle{}, reqs, code, nil, nil) {
		if d.Code == "requirement.orphan" {
			for _, r := range []string{"REQ-X-001", "REQ-X-002"} {
				if strings.Contains(d.Message, r) {
					orphan[r] = true
				}
			}
		}
	}
	if orphan["REQ-X-001"] {
		t.Fatal("REQ-X-001 is implemented and must not be orphan")
	}
	if !orphan["REQ-X-002"] {
		t.Fatal("REQ-X-002 has no trace and must be orphan")
	}
}

// TestTraceIntegrityDelegationCoversOrphan verifies a delegated requirement is not orphan
// even with no local code or verification.
// TRLC-LINKS: REQ-EMG-029
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestTraceIntegrityDelegationCoversOrphan(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-X-001"}}}
	deleg := map[string][]MaterializedAllocation{"REQ-X-001": {{Requirement: "REQ-X-001", Subsystem: "SUB", Target: "CAP"}}}
	for _, d := range validateTraceIntegrity(model.Bundle{}, reqs, nil, nil, deleg) {
		if d.Code == "requirement.orphan" {
			t.Fatalf("delegated requirement must not be orphan: %s", d.Message)
		}
	}
}

// TestBuildTraceMatrixStatusAndDelegation verifies row status and the delegation link in
// the matrix.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestBuildTraceMatrixStatusAndDelegation(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{
		{ID: "REQ-X-001", AppliesTo: []string{"FU-A"}},
		{ID: "REQ-X-002"},
	}}
	code := []inferredCodeItem{{Element: "impl", Source: "a.go:5", Implements: []string{"REQ-X-001"}}}
	deleg := map[string][]MaterializedAllocation{"REQ-X-002": {{Requirement: "REQ-X-002", Subsystem: "SUB-T", Target: "CAP-T", TargetRef: "REQ-T-001"}}}
	m := buildTraceMatrix(model.Bundle{}, reqs, code, nil, deleg)
	if m.Summary.Requirements != 2 || m.Summary.Implemented != 1 || m.Summary.Delegated != 1 {
		t.Fatalf("unexpected summary: %+v", m.Summary)
	}
	byID := map[string]TraceRow{}
	for _, r := range m.Requirements {
		byID[r.ID] = r
	}
	if byID["REQ-X-001"].Status != "implemented" || len(byID["REQ-X-001"].Code) != 1 {
		t.Fatalf("REQ-X-001 should be implemented with code: %+v", byID["REQ-X-001"])
	}
	d := byID["REQ-X-002"]
	if d.Status != "delegated" || d.DelegatedTo == nil || d.DelegatedTo.TargetRequirement != "REQ-T-001" {
		t.Fatalf("REQ-X-002 should be delegated to REQ-T-001: %+v", d)
	}
}

// TestScopeCodeExcludesNestedModel verifies code under a nested model root (its own
// architecture.yml) is not attributed to the parent model.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func TestScopeCodeExcludesNestedModel(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "architecture.yml"), "model:\n  id: TOP\n")
	if err := os.MkdirAll(filepath.Join(dir, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "child", "architecture.yml"), "model:\n  id: CHILD\n")
	writeFile(t, filepath.Join(dir, "parent.go"), "package x\n")
	writeFile(t, filepath.Join(dir, "child", "kid.go"), "package y\n")
	items := []inferredCodeItem{
		{Element: "p", Source: "parent.go:1", AbsPath: filepath.Join(dir, "parent.go"), Implements: []string{"REQ-TOP-001"}},
		{Element: "k", Source: "child/kid.go:1", AbsPath: filepath.Join(dir, "child", "kid.go"), Implements: []string{"REQ-CHILD-001"}},
	}
	scoped := scopeCodeToModel(items, []string{dir}, dir)
	if len(scoped) != 1 || scoped[0].Element != "p" {
		t.Fatalf("expected only parent code attributed to TOP, got %+v", scoped)
	}
}
