// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package reqdelta

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033
func TestPlanRequirementsDelta_AddUpdateRemoveAndPreserveComments(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  add:
    - id: REQ-TST-003
      text: "When request received event is received, the test system shall store audit record."
      notes: "Added requirement."
      appliesTo: [FU-STORE]
  update:
    - id: REQ-TST-001
      text: "When request received event is received, the test system shall persist audit record."
      notes: "Updated requirement."
      appliesTo: [FU-STORE, FU-STORE]
  remove: [REQ-TST-002]
`)

	result, err := PlanRequirementsDelta(root, deltaPath)
	if err != nil {
		t.Fatalf("plan delta: %v", err)
	}
	if validate.HasErrors(result.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	assertPlannedLists(t, result)
	assertPlannedYAML(t, result)
	assertPlannedDiff(t, result)
	original, err := os.ReadFile(filepath.Join(root, "model", "requirements.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(original), "REQ-TST-003") {
		t.Fatal("plan modified canonical requirements")
	}
}

func assertPlannedLists(t *testing.T, result RequirementsDeltaResult) {
	t.Helper()
	if got, want := strings.Join(result.Added, ","), "REQ-TST-003"; got != want {
		t.Fatalf("added = %q, want %q", got, want)
	}
	if got, want := strings.Join(result.Updated, ","), "REQ-TST-001"; got != want {
		t.Fatalf("updated = %q, want %q", got, want)
	}
	if got, want := strings.Join(result.Removed, ","), "REQ-TST-002"; got != want {
		t.Fatalf("removed = %q, want %q", got, want)
	}
}

func assertPlannedYAML(t *testing.T, result RequirementsDeltaResult) {
	t.Helper()
	if !strings.Contains(string(result.YAML), "# keep this requirement comment") {
		t.Fatalf("expected updated requirement comment to be preserved:\n%s", result.YAML)
	}
	if !strings.Contains(string(result.YAML), "# keep this field comment") {
		t.Fatalf("expected updated field comment to be preserved:\n%s", result.YAML)
	}
	if !strings.Contains(string(result.YAML), "# keep this appliesTo comment") {
		t.Fatalf("expected updated sequence item comment to be preserved:\n%s", result.YAML)
	}
	if !strings.Contains(string(result.YAML), "# keep this duplicate appliesTo comment") {
		t.Fatalf("expected duplicate sequence item comment to be preserved:\n%s", result.YAML)
	}
	if strings.Contains(string(result.YAML), "REQ-TST-002") {
		t.Fatalf("removed requirement remains in candidate:\n%s", result.YAML)
	}
}

func assertPlannedDiff(t *testing.T, result RequirementsDeltaResult) {
	t.Helper()
	if !strings.Contains(result.Diff, `+ REQ-TST-003:`) ||
		!strings.Contains(result.Diff, `~ REQ-TST-001`) ||
		!strings.Contains(result.Diff, `- REQ-TST-002:`) {
		t.Fatalf("unexpected diff:\n%s", result.Diff)
	}
}

// TRLC-LINKS: REQ-EMG-031
func TestPlanRequirementsDelta_RejectsUnnormalizedIDs(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  add:
    - id: " REQ-TST-003 "
      text: "When request received event is received, the test system shall store audit record."
`)

	_, err := PlanRequirementsDelta(root, deltaPath)
	if err == nil || !strings.Contains(err.Error(), "contains surrounding whitespace") {
		t.Fatalf("expected unnormalized id error, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-031
func TestPlanRequirementsDelta_RejectsUnnormalizedRemoveID(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  remove: [" REQ-TST-002 "]
`)

	_, err := PlanRequirementsDelta(root, deltaPath)
	if err == nil || !strings.Contains(err.Error(), "contains surrounding whitespace") {
		t.Fatalf("expected unnormalized id error, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-033
func TestApplyRequirementsDelta_RejectsSymlink(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	target := filepath.Join(root, "model", "requirements.yml")
	realTarget := filepath.Join(root, "requirements-real.yml")
	if err := os.Rename(target, realTarget); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(realTarget, target); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  remove: [REQ-TST-002]
`)

	_, err := ApplyRequirementsDelta(root, deltaPath)
	if err == nil || !strings.Contains(err.Error(), "must not be a symbolic link") {
		t.Fatalf("expected symlink rejection, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-033, REQ-EMG-185
func TestApplyRequirementsDelta_RejectsConcurrentWriterLock(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	release, err := acquireRequirementsLock(filepath.Join(root, "model", "requirements.yml"))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  remove: [REQ-TST-002]
`)

	_, err = ApplyRequirementsDelta(root, deltaPath)
	if err == nil || !strings.Contains(err.Error(), "already in progress") {
		t.Fatalf("expected concurrent writer rejection, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-033
func TestApplyRequirementsDelta_SharesLockAcrossDirectorySymlinks(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	alias := filepath.Join(t.TempDir(), "model")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}
	release, err := acquireRequirementsLock(filepath.Join(root, "model", "requirements.yml"))
	if err != nil {
		t.Fatal(err)
	}
	defer release()
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  remove: [REQ-TST-002]
`)

	_, err = ApplyRequirementsDelta(alias, deltaPath)
	if err == nil || !strings.Contains(err.Error(), "already in progress") {
		t.Fatalf("expected shared concurrent writer lock, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-033
func TestRequirementsPath_CanonicalizesDirectorySymlink(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	alias := filepath.Join(t.TempDir(), "model")
	if err := os.Symlink(root, alias); err != nil {
		t.Skipf("directory symlink unavailable: %v", err)
	}

	target, err := requirementsPath(alias)
	if err != nil {
		t.Fatal(err)
	}
	canonicalRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if want := filepath.Join(canonicalRoot, "model", "requirements.yml"); target != want {
		t.Fatalf("requirements path = %q, want canonical path %q", target, want)
	}
}

// TRLC-LINKS: REQ-EMG-033
func TestApplyRequirementsDelta_ReusesReleasedLockFile(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	target := filepath.Join(root, "model", "requirements.yml")
	release, err := acquireRequirementsLock(target)
	if err != nil {
		t.Fatal(err)
	}
	if err := release(); err != nil {
		t.Fatal(err)
	}
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  remove: [REQ-TST-002]
`)

	if _, err := ApplyRequirementsDelta(root, deltaPath); err != nil {
		t.Fatalf("expected released lock to be reusable, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033
func TestApplyRequirementsDelta_WritesValidatedCandidate(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	target := filepath.Join(root, "model", "requirements.yml")
	before, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  update:
    - id: REQ-TST-001
      text: "When request received event is received, the test system shall persist audit record."
      notes: "Applied."
      appliesTo: [FU-STORE]
`)

	result, err := ApplyRequirementsDelta(root, deltaPath)
	if err != nil {
		t.Fatalf("apply delta: %v", err)
	}
	if validate.HasErrors(result.Diagnostics) {
		t.Fatalf("unexpected diagnostics: %+v", result.Diagnostics)
	}
	assertAppliedNotes(t, target)
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != before.Mode().Perm() {
		t.Fatalf("mode changed from %v to %v", before.Mode().Perm(), info.Mode().Perm())
	}
}

func assertAppliedNotes(t *testing.T, target string) {
	t.Helper()
	after, err := model.LoadRequirements(target)
	if err != nil {
		t.Fatalf("load applied requirements: %v", err)
	}
	if got := after.Requirements[0].Notes; got != "Applied." {
		t.Fatalf("notes = %q, want Applied.", got)
	}
}

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032
func TestPlanRequirementsDelta_RejectsConflictingOperations(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  update:
    - id: REQ-TST-001
      text: "When request received event is received, the test system shall persist audit record."
  remove: [REQ-TST-001]
`)

	_, err := PlanRequirementsDelta(root, deltaPath)
	if err == nil || !strings.Contains(err.Error(), "appears in both update and remove") {
		t.Fatalf("expected conflicting operation error, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033, REQ-EMG-185
func TestApplyRequirementsDelta_InvalidCandidateDoesNotWrite(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	target := filepath.Join(root, "model", "requirements.yml")
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  add:
    - id: REQ-TST-003
      text: ""
      appliesTo: [FU-STORE]
`)

	result, err := ApplyRequirementsDelta(root, deltaPath)
	if err == nil {
		t.Fatal("expected invalid candidate error")
	}
	if !validate.HasErrors(result.Diagnostics) {
		t.Fatalf("expected blocking diagnostics, got %+v", result.Diagnostics)
	}
	after, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(before) {
		t.Fatal("invalid apply modified canonical requirements")
	}
}

// TRLC-LINKS: REQ-EMG-032, REQ-EMG-033
func TestApplyRequirementsDelta_DanglingAppliesToDoesNotWrite(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	target := filepath.Join(root, "model", "requirements.yml")
	before, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  add:
    - id: REQ-TST-003
      text: "When request received event is received, the test system shall store audit record."
      appliesTo: [FU-MISSING]
`)

	result, err := ApplyRequirementsDelta(root, deltaPath)
	if err == nil {
		t.Fatal("expected invalid candidate error")
	}
	if !hasDiagnosticCode(result.Diagnostics, "requirement.invalid_applies_to") {
		t.Fatalf("expected invalid appliesTo diagnostic, got %+v", result.Diagnostics)
	}
	after, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(after) != string(before) {
		t.Fatal("invalid apply modified canonical requirements")
	}
}

// TRLC-LINKS: REQ-EMG-031
func TestPlanRequirementsDelta_RejectsUnknownFields(t *testing.T) {
	root := writeRequirementsDeltaFixture(t)
	deltaPath := writeDelta(t, root, `
version: 1
requirements:
  add: []
  unexpected: true
`)

	_, err := PlanRequirementsDelta(root, deltaPath)
	if err == nil || !strings.Contains(err.Error(), "field unexpected not found") {
		t.Fatalf("expected strict decode error, got %v", err)
	}
}

var fixtureManifest = `
schemaVersion: 3
module:
  path: example.com/test-model@v0
  version: v0.1.0
  modelId: test-model
  title: Test Model
  introduction: ""
  kind: system
documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
dependencies: []
publications: []
inferenceHints: {}
` + testutil.CompleteSwitches

const fixtureArchitecture = `
schemaVersion: 3
architecture:
  functionalGroups:
    - id: FG-TEST
      name: Test
      description: Test group.
  functionalUnits:
    - id: FU-STORE
      name: Store
      group: FG-TEST
`

const fixtureBehavior = `
schemaVersion: 3
behavior:
  relationships:
    - type: contains
      from: FG-TEST
      to: FU-STORE
`

const fixtureCatalog = `
schemaVersion: 3
catalog:
  systems:
    - id: SYS-TEST
      name: test system
      definition: System used to test requirements changes.
  functionalGroups:
    - id: FG-TEST
      name: test
      definition: Test group.
  functionalUnits:
    - id: FU-STORE
      name: store
      definition: Stores records.
  events:
    - id: EVT-REQUEST
      name: request received event is received
      definition: A request is ready to process.
  dataTerms:
    - id: DATA-AUDIT
      name: audit record
      definition: Record retained for audit.
`

const fixtureRequirements = `
schemaVersion: 3
lintRun:
  id: test-lint
  mode: guided
  commaAsAnd: true
  catalogRef: ./catalog.yml

requirements:
  # keep this requirement comment
  - id: REQ-TST-001
    text: "When request received event is received, the test system shall store audit record." # keep this field comment
    notes: "Original."
    appliesTo:
      - FU-STORE # keep this appliesTo comment
      - FU-STORE # keep this duplicate appliesTo comment
  - id: REQ-TST-002
    text: "When request received event is received, the test system shall retain audit record."
    appliesTo: [FU-STORE]

expected: []
`

const fixtureAssurance = "schemaVersion: 3\nassurance: {}\n"

const fixtureCompliance = "schemaVersion: 3\ncompliance: {}\n"

const fixtureViews = "schemaVersion: 3\nviews: []\n"

var fixtureDecisions = testutil.TierDecisionPoC

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033
func writeRequirementsDeltaFixture(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for path, content := range map[string]string{
		"engmod.yml": fixtureManifest,
		filepath.Join("model", "architecture.yml"): fixtureArchitecture,
		filepath.Join("model", "behavior.yml"):     fixtureBehavior,
		filepath.Join("model", "catalog.yml"):      fixtureCatalog,
		filepath.Join("model", "requirements.yml"): fixtureRequirements,
		filepath.Join("model", "assurance.yml"):    fixtureAssurance,
		filepath.Join("model", "compliance.yml"):   fixtureCompliance,
		filepath.Join("model", "views.yml"):        fixtureViews,
		filepath.Join("model", "decisions.yml"):    fixtureDecisions,
	} {
		writeFixtureFile(t, filepath.Join(root, path), content)
	}
	return root
}

// TRLC-LINKS: REQ-EMG-031
func writeDelta(t *testing.T, root, content string) string {
	t.Helper()
	path := filepath.Join(root, "requirements-delta.yml")
	writeFixtureFile(t, path, content)
	return path
}

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-033
func writeFixtureFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(strings.TrimLeft(content, "\n")), 0o640); err != nil {
		t.Fatal(err)
	}
}

func hasDiagnosticCode(diagnostics []validate.Diagnostic, code string) bool {
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return true
		}
	}
	return false
}
