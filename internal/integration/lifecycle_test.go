// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package integration

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
)

// copyExample copies examples/<name> (without generated outputs, module
// caches and local locks) into a temporary directory and returns the copy.
func copyExample(t *testing.T, name string) string {
	t.Helper()
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, "")
	t.Setenv(composition.ModuleCacheEnv, "")
	src := filepath.Join("examples", name)
	root := filepath.Join(t.TempDir(), name)
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			if d.Name() == "generated" || d.Name() == ".engmod" {
				return filepath.SkipDir
			}
			return os.MkdirAll(filepath.Join(root, rel), 0o755)
		}
		if d.Name() == "engmod.lock.local.yml" {
			return nil
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(filepath.Join(root, rel), data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return root
}

// appendLifecycle appends a lifecycle block to dir/engmod.yml.
func appendLifecycle(t *testing.T, dir, block string) string {
	t.Helper()
	manifest := filepath.Join(dir, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(manifest, append(data, []byte("lifecycle:\n"+block)...), 0o644); err != nil {
		t.Fatal(err)
	}
	return manifest
}

// diagnosticsOf returns the diagnostics with the code.
func diagnosticsOf(diags []engmodel.Diagnostic, code string) []engmodel.Diagnostic {
	var out []engmodel.Diagnostic
	for _, d := range diags {
		if d.Code == code {
			out = append(out, d)
		}
	}
	return out
}

// The payments example has requirement.orphan warnings. At tier poc every
// public entry (ValidateBundle, GenerateAsciiDoc, the trace build) reports
// them deferred with requiredFrom product; at tier product they are errors
// that fail the check, but GenerateAsciiDoc still renders the document
// (D15 extended); a rank-0 error still stops the render.
// TRLC-LINKS: REQ-EMG-125, REQ-EMG-130
func TestLifecycleTierAtEveryPublicEntry(t *testing.T) {
	dir := copyExample(t, "payments-engineering-sample")
	manifest := appendLifecycle(t, dir, "  tier: poc\n")
	bundle, err := engmodel.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := engmodel.GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, engmodel.AsciiDocOptions{CodeRoot: dir})
	if err != nil {
		t.Fatalf("render at poc: %v", err)
	}
	_, traceDiags, err := engmodel.BuildTraceMatrixFromFiles(manifest, bundle.RequirementsPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	for name, diags := range map[string][]engmodel.Diagnostic{"generate": doc.Diagnostics, "trace": traceDiags} {
		orphans := diagnosticsOf(diags, "requirement.orphan")
		if len(orphans) == 0 {
			t.Fatalf("%s: no requirement.orphan in %+v", name, diags)
		}
		for _, d := range orphans {
			if d.Severity != engmodel.SeverityWarning || !d.Deferred || d.RequiredFrom != "product" {
				t.Errorf("%s at poc: %+v, want a deferred warning required from product", name, d)
			}
		}
	}
	unapproved := diagnosticsOf(engmodel.ValidateBundle(bundle), "lifecycle.requirement_unverified")
	if len(unapproved) == 0 || !unapproved[0].Deferred {
		t.Fatalf("ValidateBundle at poc: lifecycle.requirement_unverified %+v, want deferred", unapproved)
	}

	data, _ := os.ReadFile(manifest)
	if err := os.WriteFile(manifest, []byte(strings.Replace(string(data), "tier: poc", "tier: product", 1)), 0o644); err != nil {
		t.Fatal(err)
	}
	bundle, err = engmodel.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	doc, err = engmodel.GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, engmodel.AsciiDocOptions{CodeRoot: dir})
	if err != nil {
		t.Fatalf("render at product stopped on tier-raised errors: %v", err)
	}
	if !strings.Contains(doc.Document, "== ") {
		t.Fatal("no document rendered at product")
	}
	orphans := diagnosticsOf(doc.Diagnostics, "requirement.orphan")
	if len(orphans) == 0 || orphans[0].Severity != engmodel.SeverityError || !engmodel.HasErrors(doc.Diagnostics) {
		t.Fatalf("at product: %+v, want requirement.orphan errors that fail the check", orphans)
	}
	// verification.unlinked comes from model validation, which runs before
	// the render gate: as a tier-raised error it does not stop the render.
	if unlinked := diagnosticsOf(doc.Diagnostics, "verification.unlinked"); len(unlinked) == 0 || unlinked[0].Severity != engmodel.SeverityError {
		t.Fatalf("at product: verification.unlinked %+v, want errors", unlinked)
	}

	broken := bundle
	broken.Requirements.Requirements = append([]engmodel.Requirement(nil), bundle.Requirements.Requirements...)
	broken.Requirements.Requirements[0].AppliesTo = []string{"FU-DOES-NOT-EXIST"}
	stopped, err := engmodel.GenerateAsciiDoc(broken, broken.Requirements, broken.Design, engmodel.AsciiDocOptions{CodeRoot: dir})
	if err == nil {
		t.Fatal("a requirement.invalid_applies_to error did not stop the render")
	}
	// The diagnostics of a stopped render carry the tier severities too.
	if got := diagnosticsOf(stopped.Diagnostics, "lifecycle.requirement_unverified"); len(got) == 0 || got[0].RequiredFrom != "operational" {
		t.Fatalf("stopped render: lifecycle.requirement_unverified %+v, want requiredFrom operational", got)
	}
}

// composeSystem composes repos/system of a requirement-links copy (child-a
// and child-b resolve from the workspace) and returns its diagnostics with
// the system's tier policy applied.
func composeSystem(t *testing.T, root string) []engmodel.Diagnostic {
	t.Helper()
	manifest := filepath.Join(root, "repos", "system", "engmod.yml")
	res, err := engmodel.GenerateCompositionFromFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	bundle, err := engmodel.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	return engmodel.ApplyTierPolicy(bundle, res.Diagnostics)
}

// A product-tier system composing a prototype-tier child reports
// composition.dependency_below_tier (error) naming the alias and the system
// requirements linked into it, and a child without lifecycle
// composition.dependency_tier_unknown (warning); a risk with a recorded
// acceptance whose appliesTo names the subsystem composing the alias
// removes the below-tier finding; a system without lifecycle gets neither.
// TRLC-LINKS: REQ-EMG-126
func TestCompositionTierRules(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	if got := append(withCode(composeSystem(t, root), "composition.dependency_below_tier"), withCode(composeSystem(t, root), "composition.dependency_tier_unknown")...); len(got) != 0 {
		t.Fatalf("system without lifecycle: %+v", got)
	}
	appendLifecycle(t, filepath.Join(root, "repos", "child-a"), "  tier: prototype\n")
	appendLifecycle(t, system, "  tier: product\n")
	diags := composeSystem(t, root)
	below := withCode(diags, "composition.dependency_below_tier")
	if len(below) != 1 || below[0].Severity != engmodel.SeverityError || below[0].RequiredFrom != "product" ||
		!strings.Contains(below[0].Message, `"encoder"`) || !strings.Contains(below[0].Message, "prototype") || !strings.Contains(below[0].Message, "REQ-SYS-001, REQ-SYS-002") {
		t.Fatalf("below tier: %+v", below)
	}
	unknown := withCode(diags, "composition.dependency_tier_unknown")
	if len(unknown) != 1 || unknown[0].Severity != engmodel.SeverityWarning || !strings.Contains(unknown[0].Message, `"bridge"`) {
		t.Fatalf("tier unknown: %+v", unknown)
	}

	assurance := "schemaVersion: 2\nassurance:\n  risks:\n  - id: RISK-ENCODER-TIER\n    title: Encoder below tier\n    statement: The encoder is a prototype.\n    status: accepted\n    owner: ACT-SYSTEM-OWNER\n    appliesTo: [SUB-ENCODER]\n" +
		"    acceptedBy: {actor: ACT-SYSTEM-OWNER, date: \"2026-11-05\", statement: I accept the prototype encoder until v1.}\n"
	if err := os.WriteFile(filepath.Join(system, "model", "assurance.yml"), []byte(assurance), 0o644); err != nil {
		t.Fatal(err)
	}
	if got := withCode(composeSystem(t, root), "composition.dependency_below_tier"); len(got) != 0 {
		t.Fatalf("accepted risk on SUB-ENCODER: %+v", got)
	}
}

// writeSystemRisk writes repos/system/model/assurance.yml with one risk on
// the encoder dependency: status, and the field (dependencies or appliesTo)
// with its list.
func writeSystemRisk(t *testing.T, system, status, field, list string) {
	t.Helper()
	assurance := "schemaVersion: 2\nassurance:\n  risks:\n  - id: RISK-ENCODER-TIER\n    title: Encoder below tier\n    statement: The encoder is a prototype.\n    status: " + status + "\n    owner: ACT-SYSTEM-OWNER\n    " + field + ": " + list + "\n" +
		"    acceptedBy: {actor: ACT-SYSTEM-OWNER, date: \"2026-11-05\", statement: I accept the prototype encoder until v1.}\n"
	if err := os.WriteFile(filepath.Join(system, "model", "assurance.yml"), []byte(assurance), 0o644); err != nil {
		t.Fatal(err)
	}
}

// A risk with a recorded acceptance whose dependencies names the alias
// waives composition.dependency_below_tier for that dependency; the finding
// names both ways to accept it, the alias first; a risk that lists the
// alias but is not accepted (status open) waives nothing, and one listing
// another alias waives nothing.
// TRLC-LINKS: REQ-EMG-126
func TestCompositionBelowTierWaivedByRiskDependency(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	appendLifecycle(t, filepath.Join(root, "repos", "child-a"), "  tier: prototype\n")
	appendLifecycle(t, system, "  tier: product\n")

	writeSystemRisk(t, system, "open", "dependencies", "[encoder]")
	below := withCode(composeSystem(t, root), "composition.dependency_below_tier")
	if len(below) != 1 || !strings.Contains(below[0].Message, "dependencies [encoder]") || !strings.Contains(below[0].Message, "appliesTo [SUB-ENCODER]") ||
		strings.Index(below[0].Message, "dependencies [encoder]") > strings.Index(below[0].Message, "appliesTo [SUB-ENCODER]") {
		t.Fatalf("unaccepted risk on encoder: want the finding naming dependencies [encoder] before appliesTo [SUB-ENCODER], got %+v", below)
	}

	writeSystemRisk(t, system, "accepted", "dependencies", "[bridge]")
	if got := withCode(composeSystem(t, root), "composition.dependency_below_tier"); len(got) != 1 {
		t.Fatalf("accepted risk on bridge must not waive encoder: %+v", got)
	}

	writeSystemRisk(t, system, "accepted", "dependencies", "[encoder]")
	diags := composeSystem(t, root)
	if got := withCode(diags, "composition.dependency_below_tier"); len(got) != 0 {
		t.Fatalf("accepted risk with dependencies [encoder]: %+v", got)
	}
	bundle, err := engmodel.LoadBundle(filepath.Join(system, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if got := withCode(engmodel.ValidateBundle(bundle), "model.invalid_risk_dependency"); len(got) != 0 {
		t.Fatalf("a declared alias is valid: %+v", got)
	}
}
