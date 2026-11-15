// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
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
	code := []codescan.CodeItem{
		{Element: "good", Source: "a.go:1", Implements: []string{"REQ-X-001"}, ModelLinks: []string{"FU-REAL"}},
		{Element: "bad", Source: "b.go:2", Implements: []string{"REQ-X-999"}, ModelLinks: []string{"FU-GHOST"}},
	}
	dangling := diagsOf(ValidateIntegrity(bundle, Evidence{reqs, code, nil}), "code.dangling_requirement_link", "code.dangling_model_link")
	requireEqual(t, severitiesByCode(dangling), map[string]validate.Severity{
		"code.dangling_requirement_link": validate.SeverityError,
		"code.dangling_model_link":       validate.SeverityError,
	})
}

// TestTraceIntegrityFlagsOrphan verifies a requirement traced by nothing is an orphan
// warning, and a covered requirement is not.
// TRLC-LINKS: REQ-EMG-029
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE
func TestTraceIntegrityFlagsOrphan(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-X-001"}, {ID: "REQ-X-002"}}}
	code := []codescan.CodeItem{{Element: "c", Source: "a.go:1", Implements: []string{"REQ-X-001"}}}
	orphans := pathsOf(ValidateIntegrity(model.Bundle{}, Evidence{reqs, code, nil}), "requirement.orphan")
	requireEqual(t, orphans, []string{"requirements[REQ-X-002]"})
}

// TestBuildTraceMatrixStatus verifies row status in the matrix: code gives
// implemented, nothing gives orphan, and no row is delegated by an allocation
// (ADR-EMG-078: requirement links are the only trace to another model).
// Mutation: giving the unlinked requirement the status delegated, or counting
// it in Summary.Delegated, fails this test.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-187
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestBuildTraceMatrixStatus(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{
		{ID: "REQ-X-001", AppliesTo: []string{"FU-A"}},
		{ID: "REQ-X-002"},
	}}
	code := []codescan.CodeItem{{Element: "impl", Source: "a.go:5", Implements: []string{"REQ-X-001"}}}
	m := buildTraceMatrix(model.Bundle{}, Evidence{reqs, code, nil})
	if got := [4]int{m.Summary.Requirements, m.Summary.Implemented, m.Summary.Orphan, m.Summary.Delegated}; got != [4]int{2, 1, 1, 0} {
		t.Fatalf("unexpected summary: %+v", m.Summary)
	}
	statuses := map[string]string{}
	for _, r := range m.Requirements {
		statuses[r.ID] = r.Status
	}
	if statuses["REQ-X-001"] != "implemented" || len(m.Requirements[0].Code) != 1 {
		t.Fatalf("REQ-X-001 should be implemented with code: %+v", m.Requirements[0])
	}
	if statuses["REQ-X-002"] != "orphan" {
		t.Fatalf("REQ-X-002 has no trace and should be orphan: %+v", m.Requirements[1])
	}
}

// TestScopeCodeExcludesNestedModel verifies code under a nested model root (its own
// engmod.yml) is not attributed to the parent model.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func TestScopeCodeExcludesNestedModel(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "engmod.yml"), "model:\n  id: TOP\n")
	if err := os.MkdirAll(filepath.Join(dir, "child"), 0o755); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(dir, "child", "engmod.yml"), "model:\n  id: CHILD\n")
	testutil.WriteFile(t, filepath.Join(dir, "parent.go"), "package x\n")
	testutil.WriteFile(t, filepath.Join(dir, "child", "kid.go"), "package y\n")
	items := []codescan.CodeItem{
		{Element: "p", Source: "parent.go:1", AbsPath: filepath.Join(dir, "parent.go"), Implements: []string{"REQ-TOP-001"}},
		{Element: "k", Source: "child/kid.go:1", AbsPath: filepath.Join(dir, "child", "kid.go"), Implements: []string{"REQ-CHILD-001"}},
	}
	scoped := ScopeCodeToModel(items, []string{dir}, dir)
	if len(scoped) != 1 || scoped[0].Element != "p" {
		t.Fatalf("expected only parent code attributed to TOP, got %+v", scoped)
	}
}

// TestTraceSummaryJSONReportsTestLinked verifies the summary serializes
// testLinked (equal to tested) and carries no verified or linksSuspect key
// (ADR-EMG-073).
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-061, REQ-EMG-182
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestTraceSummaryJSONReportsTestLinked(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-X-001"}, {ID: "REQ-X-002"}}}
	checks := []codescan.VerificationCheck{{ID: "VC-X", Name: "TestX", Verifies: []string{"REQ-X-001"}}}
	m := buildTraceMatrix(model.Bundle{}, Evidence{reqs, nil, checks})
	raw, err := json.Marshal(m.Summary)
	if err != nil {
		t.Fatal(err)
	}
	var got map[string]any
	if err := json.Unmarshal(raw, &got); err != nil {
		t.Fatal(err)
	}
	for _, key := range []string{"testLinked", "tested"} {
		if got[key] != float64(1) {
			t.Fatalf("summary %s = %v, want 1 (%s)", key, got[key], raw)
		}
	}
	if _, present := got["verified"]; present {
		t.Fatalf("summary must not carry the removed counter verified: %s", raw)
	}
	if _, present := got["linksSuspect"]; present {
		t.Fatalf("summary must not carry the removed counter linksSuspect: %s", raw)
	}
}

// With lifecycle the trace reports verification.failed at a requirement
// whose linked test result failed and verification.gated_only at one whose
// every linked test is environment gated; a passing ungated requirement, one
// with a gated and an ungated test, and a model without lifecycle report
// neither.
// TRLC-LINKS: REQ-EMG-128
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestTraceTierVerificationRules(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-X-001"}, {ID: "REQ-X-002"}, {ID: "REQ-X-003"}, {ID: "REQ-X-004"}}}
	verification := []codescan.VerificationCheck{
		{ID: "V1", Verifies: []string{"REQ-X-001"}, Results: []codescan.VerificationResult{{Requirement: "REQ-X-001", Status: "fail"}}},
		{ID: "V2", Verifies: []string{"REQ-X-002", "REQ-X-004"}, Gates: map[string][]string{"REQ-X-002": {"HW_RIG"}, "REQ-X-004": {"HW_RIG"}}},
		{ID: "V3", Verifies: []string{"REQ-X-003", "REQ-X-004"}, Results: []codescan.VerificationResult{{Requirement: "REQ-X-003", Status: "pass"}}},
	}
	with := model.Bundle{Manifest: model.ManifestDocument{Lifecycle: &model.ManifestLifecycle{Tier: "product"}}}
	codes := []string{"verification.failed", "verification.gated_only"}
	diags := ValidateIntegrity(with, Evidence{reqs, nil, verification})
	requireEqual(t, [][]string{pathsOf(diags, codes[0]), pathsOf(diags, codes[1])}, [][]string{{"requirements[0]"}, {"requirements[1]"}})
	requireEqual(t, len(diagsOf(ValidateIntegrity(model.Bundle{}, Evidence{reqs, nil, verification}), codes...)), 0)
}
