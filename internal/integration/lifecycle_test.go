// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package integration

import (
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
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
	copying := exampleCopy{
		src:       filepath.Join("examples", name),
		dst:       filepath.Join(t.TempDir(), name),
		skipsFile: func(file string) bool { return file == "engmod.lock.local.yml" },
	}
	if err := filepath.WalkDir(copying.src, copying.visit); err != nil {
		t.Fatal(err)
	}
	return copying.dst
}

// exampleCopy copies the files of one example directory tree, leaving out the
// files skipsFile names.
type exampleCopy struct {
	src, dst  string
	skipsFile func(name string) bool
}

// visit copies one entry of the walk, skipping generated outputs, module
// caches and the files skipsFile names.
func (c exampleCopy) visit(path string, d fs.DirEntry, err error) error {
	if err != nil {
		return err
	}
	rel, _ := filepath.Rel(c.src, path)
	if d.IsDir() {
		return c.visitDir(d, rel)
	}
	if c.skipsFile(d.Name()) {
		return nil
	}
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(c.dst, rel), data, 0o644)
}

// visitDir creates the copy of a directory, or skips a generated or cache one.
func (c exampleCopy) visitDir(d fs.DirEntry, rel string) error {
	if d.Name() == "generated" || d.Name() == ".engmod" {
		return filepath.SkipDir
	}
	return os.MkdirAll(filepath.Join(c.dst, rel), 0o755)
}

// tierDecisionLine matches the Tier: and Bundles: lines of a tier decision.
var tierDecisionLine = regexp.MustCompile(`(?m)^(\s*)(Tier|Bundles): .*$`)

// lifecycleChoice is a tier with the bundles that apply at it.
type lifecycleChoice struct {
	tier    string
	bundles []string
}

// decisionID is the ID of the accepted decision that records the tier.
func (c lifecycleChoice) decisionID() string {
	return "DEC-TIER-" + strings.ToUpper(c.tier)
}

// setLifecycle gives the example copy in dir the lifecycle tier with the
// foundation bundle; see setLifecycleWith.
func setLifecycle(t *testing.T, dir, tier string) string {
	t.Helper()
	return setLifecycleWith(t, dir, lifecycleChoice{tier: tier, bundles: []string{"foundation"}})
}

// setLifecycleWith gives the example copy in dir the lifecycle tier with the
// bundles, the matching accepted DEC-TIER-<TIER> decision (the examples carry
// DEC-TIER-MVP; its id, Tier: and Bundles: lines are rewritten) and, from
// product, the health block that tier requires. It returns the manifest path.
func setLifecycleWith(t *testing.T, dir string, choice lifecycleChoice) string {
	t.Helper()
	id := choice.decisionID()
	manifest := filepath.Join(dir, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if i := strings.Index(string(data), "lifecycle:"); i >= 0 {
		data = data[:i]
	}
	block := "lifecycle:\n  tier: " + choice.tier + "\n  bundles: [" + strings.Join(choice.bundles, ", ") + "]\n  tierDecision: " + id + "\n"
	if model.TierRank(choice.tier) >= model.TierRank(model.TierProduct) {
		block += "health: {}\n"
	}
	if err := os.WriteFile(manifest, append(data, []byte(block)...), 0o644); err != nil {
		t.Fatal(err)
	}
	rewriteTierDecision(t, filepath.Join(dir, "model", "decisions.yml"), choice)
	return manifest
}

// rewriteTierDecision gives the tier decision in the decisions document the
// ID and the tier and bundles of choice.
func rewriteTierDecision(t *testing.T, decisions string, choice lifecycleChoice) {
	t.Helper()
	text, err := os.ReadFile(decisions)
	if err != nil {
		t.Fatal(err)
	}
	out := regexp.MustCompile(`id: DEC-TIER-[A-Z]+`).ReplaceAllString(string(text), "id: "+choice.decisionID())
	out = tierDecisionLine.ReplaceAllStringFunc(out, func(line string) string {
		m := tierDecisionLine.FindStringSubmatch(line)
		if m[2] == "Tier" {
			return m[1] + "Tier: " + choice.tier
		}
		return m[1] + "Bundles: " + strings.Join(choice.bundles, ", ")
	})
	if err := os.WriteFile(decisions, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
}

// removeLifecycle cuts the lifecycle block (the last key the examples write,
// with a health block after it) from dir/engmod.yml: the model without a tier
// that the kept library behaviour (composition tier rules, the tier gap of a
// dependency) is tested against. The validator reports such a model as
// lifecycle.missing. It returns the manifest path.
func removeLifecycle(t *testing.T, dir string) string {
	t.Helper()
	manifest := filepath.Join(dir, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	i := strings.Index(text, "lifecycle:")
	if i < 0 {
		t.Fatalf("%s declares no lifecycle", manifest)
	}
	if err := os.WriteFile(manifest, []byte(text[:i]), 0o644); err != nil {
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
	assertOrphansDeferredAtPoc(t, dir)
	bundle := renderAtProductTier(t, dir)

	broken := bundle
	broken.Requirements.Requirements = append([]engmodel.Requirement(nil), bundle.Requirements.Requirements...)
	broken.Requirements.Requirements[0].AppliesTo = []string{"FU-DOES-NOT-EXIST"}
	stopped, err := engmodel.GenerateAsciiDoc(engmodel.AsciiDocInputs{Bundle: broken, Requirements: broken.Requirements, Design: broken.Design}, engmodel.AsciiDocOptions{CodeRoot: dir})
	if err == nil {
		t.Fatal("a requirement.invalid_applies_to error did not stop the render")
	}
	// The diagnostics of a stopped render carry the tier severities too.
	if got := diagnosticsOf(stopped.Diagnostics, "lifecycle.requirement_unverified"); len(got) == 0 || got[0].RequiredFrom != "operational" {
		t.Fatalf("stopped render: lifecycle.requirement_unverified %+v, want requiredFrom operational", got)
	}
}

// assertOrphansDeferredAtPoc checks that at tier poc the generate and trace
// entries report requirement.orphan as a warning deferred to product and
// ValidateBundle reports lifecycle.requirement_unverified as deferred.
func assertOrphansDeferredAtPoc(t *testing.T, dir string) {
	t.Helper()
	manifest := setLifecycle(t, dir, "poc")
	bundle, err := engmodel.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	doc, err := engmodel.GenerateAsciiDoc(engmodel.AsciiDocInputs{Bundle: bundle, Requirements: bundle.Requirements, Design: bundle.Design}, engmodel.AsciiDocOptions{CodeRoot: dir})
	if err != nil {
		t.Fatalf("render at poc: %v", err)
	}
	_, traceDiags, err := engmodel.BuildTraceMatrixFromFiles(manifest, bundle.RequirementsPath, dir)
	if err != nil {
		t.Fatal(err)
	}
	for name, diags := range map[string][]engmodel.Diagnostic{"generate": doc.Diagnostics, "trace": traceDiags} {
		assertOrphansDeferred(t, name, diags)
	}
	unverified := diagnosticsOf(engmodel.ValidateBundle(bundle), "lifecycle.requirement_unverified")
	if len(unverified) == 0 || !unverified[0].Deferred {
		t.Fatalf("ValidateBundle at poc: lifecycle.requirement_unverified %+v, want deferred", unverified)
	}
}

// assertOrphansDeferred checks that diags hold requirement.orphan only as
// warnings deferred to product.
func assertOrphansDeferred(t *testing.T, name string, diags []engmodel.Diagnostic) {
	t.Helper()
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

// renderAtProductTier checks that at tier product the orphans and unlinked
// verifications are errors that fail the check although the document is still
// rendered, and returns the loaded bundle.
func renderAtProductTier(t *testing.T, dir string) engmodel.Bundle {
	t.Helper()
	bundle, err := engmodel.LoadBundle(setLifecycle(t, dir, "product"))
	if err != nil {
		t.Fatal(err)
	}
	doc, err := engmodel.GenerateAsciiDoc(engmodel.AsciiDocInputs{Bundle: bundle, Requirements: bundle.Requirements, Design: bundle.Design}, engmodel.AsciiDocOptions{CodeRoot: dir})
	if err != nil {
		t.Fatalf("render at product stopped on tier-raised errors: %v", err)
	}
	if !strings.Contains(doc.Document, "== ") {
		t.Fatal("no document rendered at product")
	}
	assertTierRaisedErrors(t, doc.Diagnostics)
	return bundle
}

// assertTierRaisedErrors checks that the orphans and unlinked verifications
// are errors at product, and that the orphans fail the check.
func assertTierRaisedErrors(t *testing.T, diags []engmodel.Diagnostic) {
	t.Helper()
	orphans := diagnosticsOf(diags, "requirement.orphan")
	if len(orphans) == 0 || orphans[0].Severity != engmodel.SeverityError || !engmodel.HasErrors(diags) {
		t.Fatalf("at product: %+v, want requirement.orphan errors that fail the check", orphans)
	}
	// verification.unlinked comes from model validation, which runs before
	// the render gate: as a tier-raised error it does not stop the render.
	if unlinked := diagnosticsOf(diags, "verification.unlinked"); len(unlinked) == 0 || unlinked[0].Severity != engmodel.SeverityError {
		t.Fatalf("at product: verification.unlinked %+v, want errors", unlinked)
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
// composition.dependency_tier_unknown (a core error at every tier, ADR-EMG-073,
// with a remedy that asks the dependency's owner); a risk with a recorded
// acceptance whose appliesTo names the subsystem composing the alias
// removes the below-tier finding; a system without lifecycle gets neither.
// TRLC-LINKS: REQ-EMG-126, REQ-EMG-183
func TestCompositionTierRules(t *testing.T) {
	root := copyRequirementLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	// The examples declare a tier; the library behaviour without one is kept,
	// so the case strips it from all three repositories first.
	for _, name := range []string{"system", "child-a", "child-b"} {
		removeLifecycle(t, filepath.Join(root, "repos", name))
	}
	diags := composeSystem(t, root)
	if got := append(withCode(diags, "composition.dependency_below_tier"), withCode(diags, "composition.dependency_tier_unknown")...); len(got) != 0 {
		t.Fatalf("system without lifecycle: %+v", got)
	}
	setLifecycle(t, filepath.Join(root, "repos", "child-a"), "prototype")
	setLifecycle(t, system, "product")
	assertBelowTierNamesTheEncoder(t, withCode(composeSystem(t, root), "composition.dependency_below_tier"))
	requireTierUnknownIsCoreError(t, root, system)

	writeSystemRisk(t, system, systemRisk{status: "accepted", field: "appliesTo", list: "[SUB-ENCODER]"})
	if got := withCode(composeSystem(t, root), "composition.dependency_below_tier"); len(got) != 0 {
		t.Fatalf("accepted risk on SUB-ENCODER: %+v", got)
	}
}

// assertBelowTierNamesTheEncoder checks that below is one error required from
// product naming the alias, the tier and the linked system requirements.
func assertBelowTierNamesTheEncoder(t *testing.T, below []engmodel.Diagnostic) {
	t.Helper()
	if len(below) != 1 || below[0].Severity != engmodel.SeverityError || below[0].RequiredFrom != "product" {
		t.Fatalf("below tier: %+v", below)
	}
	for _, want := range []string{`"encoder"`, "prototype", "REQ-SYS-001, REQ-SYS-002"} {
		if !strings.Contains(below[0].Message, want) {
			t.Fatalf("below tier: %+v", below)
		}
	}
}

// isPlainCoreError reports whether a composition diagnostic is an error that
// no tier lowers or defers.
func isPlainCoreError(d engmodel.Diagnostic) bool {
	return d.Severity == engmodel.SeverityError && d.RequiredFrom == "" && !d.Deferred
}

// requireTierUnknownIsCoreError checks that composition.dependency_tier_unknown
// is a core error with the owner remedy at the system's tier and at poc, and
// leaves the system at product.
// TRLC-LINKS: REQ-EMG-183
func requireTierUnknownIsCoreError(t *testing.T, root, system string) {
	t.Helper()
	const code = "composition.dependency_tier_unknown"
	unknown := withCode(composeSystem(t, root), code)
	if len(unknown) != 1 || !isPlainCoreError(unknown[0]) || !strings.Contains(unknown[0].Message, `"bridge"`) {
		t.Fatalf("tier unknown: %+v", unknown)
	}
	if !strings.Contains(unknown[0].Remedy, "owner of the dependency") {
		t.Fatalf("tier unknown remedy: %q", unknown[0].Remedy)
	}
	setLifecycle(t, system, "poc")
	if low := withCode(composeSystem(t, root), code); len(low) != 1 || !isPlainCoreError(low[0]) {
		t.Fatalf("tier unknown at poc: %+v", low)
	}
	setLifecycle(t, system, "product")
}

// systemRisk is the risk of the encoder dependency: its status, and the field
// (dependencies or appliesTo) with its list.
type systemRisk struct{ status, field, list string }

// writeSystemRisk writes repos/system/model/assurance.yml with one risk on
// the encoder dependency.
func writeSystemRisk(t *testing.T, system string, risk systemRisk) {
	t.Helper()
	assurance := "schemaVersion: 3\nassurance:\n  risks:\n  - id: RISK-ENCODER-TIER\n    title: Encoder below tier\n    statement: The encoder is a prototype.\n    status: " + risk.status + "\n    owner: ACT-SYSTEM-OWNER\n    " + risk.field + ": " + risk.list + "\n" +
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
	setLifecycle(t, filepath.Join(root, "repos", "child-a"), "prototype")
	setLifecycle(t, filepath.Join(root, "repos", "child-b"), "product")
	setLifecycle(t, system, "product")

	writeSystemRisk(t, system, systemRisk{status: "open", field: "dependencies", list: "[encoder]"})
	assertFindingNamesBothAcceptances(t, withCode(composeSystem(t, root), "composition.dependency_below_tier"))

	writeSystemRisk(t, system, systemRisk{status: "accepted", field: "dependencies", list: "[bridge]"})
	if got := withCode(composeSystem(t, root), "composition.dependency_below_tier"); len(got) != 1 {
		t.Fatalf("accepted risk on bridge must not waive encoder: %+v", got)
	}

	writeSystemRisk(t, system, systemRisk{status: "accepted", field: "dependencies", list: "[encoder]"})
	if got := withCode(composeSystem(t, root), "composition.dependency_below_tier"); len(got) != 0 {
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

// assertFindingNamesBothAcceptances checks that below is one finding that names
// dependencies [encoder] before appliesTo [SUB-ENCODER].
func assertFindingNamesBothAcceptances(t *testing.T, below []engmodel.Diagnostic) {
	t.Helper()
	if len(below) != 1 {
		t.Fatalf("unaccepted risk on encoder: want one finding, got %+v", below)
	}
	byDependencies := strings.Index(below[0].Message, "dependencies [encoder]")
	byApplies := strings.Index(below[0].Message, "appliesTo [SUB-ENCODER]")
	if byDependencies < 0 || byApplies < 0 || byDependencies > byApplies {
		t.Fatalf("unaccepted risk on encoder: want the finding naming dependencies [encoder] before appliesTo [SUB-ENCODER], got %+v", below)
	}
}
