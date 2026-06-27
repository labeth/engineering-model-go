// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TestCompositionResolvesCoffeeFleet verifies the coffee-fleet system-of-systems
// resolves its subsystems, materializes all allocations, and reports no errors.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-020
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func TestCompositionResolvesCoffeeFleet(t *testing.T) {
	path := filepath.Join("examples", "coffee-fleet-ota-cloud-sample", "architecture.yml")
	res, err := GenerateCompositionFromFile(path)
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	if res.Root == nil {
		t.Fatal("nil composition root")
	}
	if len(res.Root.Children) != 3 {
		t.Fatalf("expected 3 resolved subsystems, got %d", len(res.Root.Children))
	}
	if len(res.Allocations) != 3 {
		t.Fatalf("expected 3 allocations, got %d", len(res.Allocations))
	}
	for _, a := range res.Allocations {
		if !a.Resolved {
			t.Fatalf("allocation %s -> %s/%s did not resolve: %s", a.Requirement, a.Subsystem, a.Target, a.Note)
		}
	}
	for _, d := range res.Diagnostics {
		if d.Severity == validate.SeverityError {
			t.Fatalf("unexpected composition error: %s %s", d.Code, d.Message)
		}
	}
}

// TestCompositionRejectsEscapingRef verifies a subsystem ref outside the workspace
// boundary is rejected.
// TRLC-LINKS: REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY
func TestCompositionRejectsEscapingRef(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "catalog.yml"), "catalog:\n  systems:\n    - id: SYS-X\n      name: x\n      definition: x.\n")
	writeFile(t, filepath.Join(dir, "architecture.yml"),
		"model:\n  id: TOP\n  title: Top\n  baseCatalogRef: ./catalog.yml\ncomposition:\n  subsystems:\n    - id: SUB-ESCAPE\n      ref: ../../etc\n")
	res, err := GenerateCompositionFromFile(filepath.Join(dir, "architecture.yml"))
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "composition.out_of_workspace" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected composition.out_of_workspace diagnostic, got %+v", res.Diagnostics)
	}
}

// TestRequirementInternalLinkWarns verifies that a requirement pointing at another
// requirement in the same document is flagged: requirements carry no tiers and
// delegate to subsystems, not to one another within a document.
// TRLC-LINKS: REQ-EMG-023
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE
func TestRequirementInternalLinkWarns(t *testing.T) {
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{
		{ID: "REQ-A", AppliesTo: []string{"REQ-B"}},
		{ID: "REQ-B", AppliesTo: []string{"FU-OK"}},
	}}
	found := false
	for _, d := range lintRequirementInternalLinks(reqs) {
		if d.Code == "requirement.internal_link" && d.Severity == validate.SeverityWarning {
			found = true
		}
	}
	if !found {
		t.Fatal("expected requirement.internal_link warning for a requirement linking to another requirement")
	}
}

// TestDelegationResolvesToSubsystemRequirement verifies a parent allocation resolves
// through the subsystem contract ref to the specific subsystem requirement that
// satisfies it, so the delegation names exactly what solves it.
// TRLC-LINKS: REQ-EMG-022, REQ-EMG-025
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestDelegationResolvesToSubsystemRequirement(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(child, "catalog.yml"), "catalog:\n  systems:\n    - id: SYS-C\n      name: c\n      definition: c.\n")
	writeFile(t, filepath.Join(child, "architecture.yml"), "model:\n  id: CHILD\n  title: Child\n  baseCatalogRef: ./catalog.yml\ncontract:\n  provides:\n    - id: CAP-C\n      kind: capability\n      ref: REQ-C-001\n")
	writeFile(t, filepath.Join(child, "requirements.yml"), "requirements:\n  - id: REQ-C-001\n    text: realizes CAP-C\n")
	writeFile(t, filepath.Join(dir, "catalog.yml"), "catalog:\n  systems:\n    - id: SYS-P\n      name: p\n      definition: p.\n")
	writeFile(t, filepath.Join(dir, "architecture.yml"), "model:\n  id: TOP\n  title: Top\n  baseCatalogRef: ./catalog.yml\ncomposition:\n  subsystems:\n    - id: SUB-C\n      ref: ./child\n  allocations:\n    - requirement: REQ-P-001\n      to: SUB-C\n      target: CAP-C\n")

	res, err := GenerateCompositionFromFile(filepath.Join(dir, "architecture.yml"))
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	var found bool
	for _, m := range res.Allocations {
		if m.Target == "CAP-C" {
			found = true
			if m.TargetRef != "REQ-C-001" || !m.TargetRefResolved {
				t.Fatalf("expected delegation to resolve to REQ-C-001, got ref=%q resolved=%v", m.TargetRef, m.TargetRefResolved)
			}
		}
	}
	if !found {
		t.Fatal("allocation to CAP-C was not materialized")
	}
	for _, d := range res.Diagnostics {
		if d.Code == "composition.unknown_target_requirement" || d.Code == "composition.untraceable_delegation" {
			t.Fatalf("unexpected diagnostic %s: %s", d.Code, d.Message)
		}
	}
}

// TestDelegationUnknownTargetRequirementErrors verifies that a contract ref pointing
// at a requirement the subsystem does not define is a hard error.
// TRLC-LINKS: REQ-EMG-022
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func TestDelegationUnknownTargetRequirementErrors(t *testing.T) {
	dir := t.TempDir()
	child := filepath.Join(dir, "child")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(child, "catalog.yml"), "catalog:\n  systems:\n    - id: SYS-C\n      name: c\n      definition: c.\n")
	writeFile(t, filepath.Join(child, "architecture.yml"), "model:\n  id: CHILD\n  title: Child\n  baseCatalogRef: ./catalog.yml\ncontract:\n  provides:\n    - id: CAP-C\n      kind: capability\n      ref: REQ-MISSING\n")
	writeFile(t, filepath.Join(child, "requirements.yml"), "requirements:\n  - id: REQ-C-001\n    text: realizes CAP-C\n")
	writeFile(t, filepath.Join(dir, "catalog.yml"), "catalog:\n  systems:\n    - id: SYS-P\n      name: p\n      definition: p.\n")
	writeFile(t, filepath.Join(dir, "architecture.yml"), "model:\n  id: TOP\n  title: Top\n  baseCatalogRef: ./catalog.yml\ncomposition:\n  subsystems:\n    - id: SUB-C\n      ref: ./child\n  allocations:\n    - requirement: REQ-P-001\n      to: SUB-C\n      target: CAP-C\n")

	res, err := GenerateCompositionFromFile(filepath.Join(dir, "architecture.yml"))
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "composition.unknown_target_requirement" && d.Severity == validate.SeverityError {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected composition.unknown_target_requirement error, got %+v", res.Diagnostics)
	}
}

// TestCompositionResolvesGitSubsystem verifies an external git subsystem is cloned
// into the workspace .engmod cache and resolved into the composed view like a local
// subsystem. The "external" repo is a local git repo, so the test stays hermetic.
// TRLC-LINKS: REQ-EMG-027
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func TestCompositionResolvesGitSubsystem(t *testing.T) {
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git not available")
	}
	root := t.TempDir()

	// Build an external repository holding a subsystem model.
	remote := filepath.Join(root, "remote")
	if err := os.MkdirAll(remote, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(remote, "catalog.yml"), "catalog:\n  systems:\n    - id: SYS-EXT\n      name: ext\n      definition: ext.\n")
	writeFile(t, filepath.Join(remote, "architecture.yml"), "model:\n  id: EXT\n  title: External\n  baseCatalogRef: ./catalog.yml\ncontract:\n  provides:\n    - id: CAP-EXT\n      kind: capability\n      ref: REQ-EXT-001\n")
	writeFile(t, filepath.Join(remote, "requirements.yml"), "requirements:\n  - id: REQ-EXT-001\n    text: realizes CAP-EXT\n")
	for _, args := range [][]string{
		{"init", "--quiet"},
		{"-c", "user.email=t@t", "-c", "user.name=t", "add", "."},
		{"-c", "user.email=t@t", "-c", "user.name=t", "commit", "--quiet", "-m", "init"},
	} {
		if out, err := runGit(remote, args...); err != nil {
			t.Fatalf("git %v: %v: %s", args, err, out)
		}
	}

	// Top model in a separate workspace references the repo via git:.
	ws := filepath.Join(root, "ws")
	if err := os.MkdirAll(ws, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(ws, "catalog.yml"), "catalog:\n  systems:\n    - id: SYS-TOP\n      name: top\n      definition: top.\n")
	writeFile(t, filepath.Join(ws, "architecture.yml"), "model:\n  id: TOP\n  title: Top\n  baseCatalogRef: ./catalog.yml\ncomposition:\n  subsystems:\n    - id: SUB-EXT\n      git: "+remote+"\n  allocations:\n    - requirement: REQ-TOP-001\n      to: SUB-EXT\n      target: CAP-EXT\n")

	res, err := GenerateCompositionFromFile(filepath.Join(ws, "architecture.yml"))
	if err != nil {
		t.Fatalf("compose: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(ws, ".engmod", "subsystems", "SUB-EXT", "architecture.yml")); statErr != nil {
		t.Fatalf("expected external subsystem cloned into .engmod, got %v", statErr)
	}
	if res.Root == nil || len(res.Root.Children) != 1 {
		t.Fatalf("expected 1 resolved external subsystem, got %#v", res.Root)
	}
	var resolved bool
	for _, m := range res.Allocations {
		if m.Target == "CAP-EXT" {
			resolved = m.Resolved && m.TargetRef == "REQ-EXT-001" && m.TargetRefResolved
		}
	}
	if !resolved {
		t.Fatalf("expected allocation to the cloned subsystem to resolve to REQ-EXT-001, got %+v", res.Allocations)
	}
	for _, d := range res.Diagnostics {
		if d.Severity == validate.SeverityError {
			t.Fatalf("unexpected composition error: %s %s", d.Code, d.Message)
		}
	}
}

// writeFile is a test helper.
// TRLC-LINKS: REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}
