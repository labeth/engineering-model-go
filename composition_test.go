// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestCompositionResolvesManifestPublicationAndWritesLock(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	writeV2CompositionModule(t, child, "example.com/child@v1", "v1.2.3", "CHILD", "public", []string{"CAP-CHILD"}, []string{"REQ-CHILD"})
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 2
replacements:
  - module: example.com/child@v1
    path: ./child
`)
	parent := filepath.Join(root, "parent")
	writeV2CompositionParent(t, parent, `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
`, `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
    allocations:
      - requirement: REQ-PARENT
        to: SUB-CHILD
        target: child::CAP-CHILD
`)

	result, err := GenerateCompositionFromFile(filepath.Join(parent, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoCompositionErrors(t, result.Diagnostics)
	if result.Root == nil || len(result.Root.Children) != 1 {
		t.Fatalf("expected one child, got %#v", result.Root)
	}
	if len(result.Allocations) != 1 || !result.Allocations[0].Resolved || result.Allocations[0].TargetRef != "REQ-CHILD" {
		t.Fatalf("allocation did not resolve through publication: %+v", result.Allocations)
	}
	if len(result.Provenance) != 2 || result.Provenance[0].Alias != "child" || result.Provenance[0].ResolvedDir != child {
		t.Fatalf("unexpected provenance: %+v", result.Provenance)
	}
	lockPath := filepath.Join(parent, CompositionLockFileName)
	first, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(first), "sha256:") || !strings.Contains(string(first), "alias: child") {
		t.Fatalf("unexpected lock file:\n%s", first)
	}
	if _, err := GenerateCompositionFromFile(filepath.Join(parent, "engmod.yml")); err != nil {
		t.Fatal(err)
	}
	second, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if string(first) != string(second) {
		t.Fatalf("lock generation is not deterministic:\n%s\n---\n%s", first, second)
	}
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func TestCompositionRejectsUnpublishedQualifiedReference(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	writeV2CompositionModule(t, child, "example.com/child@v1", "v1.2.3", "CHILD", "public", []string{"CAP-CHILD"}, nil)
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 2
replacements:
  - module: example.com/child@v1
    path: ./child
`)
	parent := filepath.Join(root, "parent")
	writeV2CompositionParent(t, parent, `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
`, `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
    allocations:
      - requirement: REQ-PARENT
        to: SUB-CHILD
        target: child::REQ-CHILD
`)
	result, err := GenerateCompositionFromFile(filepath.Join(parent, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertCompositionDiagnostic(t, result.Diagnostics, "composition.allocation_to_internal")
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func TestCompositionRejectsUnpublishedBehaviorReference(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	writeV2CompositionModule(t, child, "example.com/child@v1", "v1.2.3", "CHILD", "public", []string{"CAP-CHILD"}, nil)
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 2\nreplacements:\n  - module: example.com/child@v1\n    path: ./child\n")
	parent := filepath.Join(root, "parent")
	writeV2CompositionParent(t, parent, `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
`, `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
`)
	writeFile(t, filepath.Join(parent, "model", "behavior.yml"), `schemaVersion: 2
behavior:
  relationships:
    - type: depends_on
      from: child::CAP-CHILD
      to: child::INTERNAL
`)
	result, err := GenerateCompositionFromFile(filepath.Join(parent, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertCompositionDiagnostic(t, result.Diagnostics, "composition.unpublished_reference")
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func TestCompositionValidatesDependencyPublicationAgreement(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	writeV2CompositionModule(t, child, "example.com/child@v1", "v1.2.3", "CHILD", "public", []string{"CAP-CHILD"}, nil)
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 2
replacements:
  - module: example.com/child@v1
    path: ./child
`)
	tests := []struct {
		name, dependencies, publication, diagnostic string
	}{
		{"unknown alias", "dependencies: []\n", "public", "composition.unknown_dependency"},
		{"duplicate alias", `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
  - alias: child
    path: example.com/other@v1
    version: v1.0.0
    publications: [public]
`, "public", "composition.duplicate_dependency_alias"},
		{"unselected publication", `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [other]
`, "public", "composition.publication_not_selected"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			parent := filepath.Join(root, strings.ReplaceAll(test.name, " ", "-"))
			writeV2CompositionParent(t, parent, test.dependencies, `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: `+test.publication+`
`)
			result, err := GenerateCompositionFromFile(filepath.Join(parent, "engmod.yml"))
			if err != nil {
				t.Fatal(err)
			}
			assertCompositionDiagnostic(t, result.Diagnostics, test.diagnostic)
		})
	}
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func TestQualifiedReferenceHelpers(t *testing.T) {
	qualified, err := model.QualifyReference("child", "CAP-CHILD")
	if err != nil || qualified != "child::CAP-CHILD" {
		t.Fatalf("qualify: %q %v", qualified, err)
	}
	alias, id, err := model.ParseQualifiedReference(qualified)
	if err != nil || alias != "child" || id != "CAP-CHILD" {
		t.Fatalf("parse: alias=%q id=%q err=%v", alias, id, err)
	}
	if _, _, err := model.ParseQualifiedReference("SUB-CHILD/CAP-CHILD"); err == nil {
		t.Fatal("expected legacy slash-qualified reference to be rejected")
	}
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049
func TestExactModuleCoordinates(t *testing.T) {
	if _, err := model.ParseExactModuleVersion("example.com/child@v1", "v1.2.3"); err != nil {
		t.Fatalf("valid exact module coordinate rejected: %v", err)
	}
	for _, test := range []struct {
		path, version string
	}{
		{"example.com/child", "v1.2.3"},
		{"example.com/child@v1", "latest"},
		{"example.com/child@v1", "v1.2"},
	} {
		if _, err := model.ParseExactModuleVersion(test.path, test.version); err == nil {
			t.Fatalf("expected invalid module coordinate %s@%s to fail", test.path, test.version)
		}
	}
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func TestCompositionRejectsPublicationIDOwnedByAnotherDomain(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	writeV2CompositionModule(t, child, "example.com/child@v1", "v1.2.3", "CHILD", "public", []string{"CAP-CHILD"}, nil)
	manifestPath := filepath.Join(child, "engmod.yml")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, manifestPath, strings.Replace(string(manifest), "architecture: [CAP-CHILD]", "architecture: [REQ-CHILD]", 1))
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 2
replacements:
  - module: example.com/child@v1
    path: ./child
`)
	parent := filepath.Join(root, "parent")
	writeV2CompositionParent(t, parent, `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
`, `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
`)
	result, err := GenerateCompositionFromFile(filepath.Join(parent, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertCompositionDiagnostic(t, result.Diagnostics, "composition.publication_unowned_id")
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func writeV2CompositionModule(t *testing.T, dir, modulePath, version, modelID, publication string, architectureIDs, requirementIDs []string) {
	t.Helper()
	capabilityID := "CAP-CHILD"
	if len(architectureIDs) > 0 {
		capabilityID = architectureIDs[0]
	}
	requirementID := "REQ-CHILD"
	if len(requirementIDs) > 0 {
		requirementID = requirementIDs[0]
	}
	writeV2ModuleFiles(t, dir, modulePath, version, modelID, "", `  contract:
    provides:
      - id: `+capabilityID+`
        kind: capability
        ref: `+requirementID+`
`)
	var publicationYAML strings.Builder
	publicationYAML.WriteString("publications:\n  - id: " + publication + "\n")
	if len(architectureIDs) > 0 {
		publicationYAML.WriteString("    architecture: [" + strings.Join(architectureIDs, ", ") + "]\n")
	}
	if len(requirementIDs) > 0 {
		publicationYAML.WriteString("    requirements: [" + strings.Join(requirementIDs, ", ") + "]\n")
	}
	path := filepath.Join(dir, "engmod.yml")
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, path, strings.Replace(string(data), "documents:\n", publicationYAML.String()+"documents:\n", 1))
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func writeV2CompositionParent(t *testing.T, dir, dependencies, architecture string) {
	t.Helper()
	writeV2ModuleFiles(t, dir, "example.com/parent@v1", "v1.0.0", "PARENT", dependencies, architecture)
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func writeV2ModuleFiles(t *testing.T, dir, modulePath, version, modelID, manifestExtra, architecture string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \""+modulePath+"\"\nlanguage: version: \"v0.17.0\"\n")
	writeFile(t, filepath.Join(dir, "engmod.yml"), `schemaVersion: 2
module:
  path: `+modulePath+`
  version: `+version+`
  modelId: `+modelID+`
  title: `+modelID+`
  introduction: ""
  kind: system
`+manifestExtra+`documents:
  catalog: model/catalog.yml
  requirements: model/requirements.yml
  architecture: model/architecture.yml
  behavior: model/behavior.yml
  assurance: model/assurance.yml
  compliance: model/compliance.yml
  views: model/views.yml
  decisions: model/decisions.yml
`)
	writeFile(t, filepath.Join(dir, "model", "catalog.yml"), "schemaVersion: 2\ncatalog: {}\n")
	writeFile(t, filepath.Join(dir, "model", "requirements.yml"), "schemaVersion: 2\nlintRun: {}\nrequirements:\n  - id: REQ-CHILD\n    text: Child requirement.\n  - id: REQ-SHARED-001\n    text: Shared requirement.\n  - id: REQ-PARENT\n    text: Parent requirement.\n")
	writeFile(t, filepath.Join(dir, "model", "architecture.yml"), "schemaVersion: 2\narchitecture:\n"+architecture)
	writeFile(t, filepath.Join(dir, "model", "behavior.yml"), "schemaVersion: 2\nbehavior: {}\n")
	writeFile(t, filepath.Join(dir, "model", "assurance.yml"), "schemaVersion: 2\nassurance: {}\n")
	writeFile(t, filepath.Join(dir, "model", "compliance.yml"), "schemaVersion: 2\ncompliance: {}\n")
	writeFile(t, filepath.Join(dir, "model", "views.yml"), "schemaVersion: 2\nviews: []\n")
	writeFile(t, filepath.Join(dir, "model", "decisions.yml"), "schemaVersion: 2\ndecisions: []\n")
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write %s: %v", path, err)
	}
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func assertNoCompositionErrors(t *testing.T, diagnostics []validate.Diagnostic) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Severity == validate.SeverityError {
			t.Fatalf("unexpected composition error: %s %s", diagnostic.Code, diagnostic.Message)
		}
	}
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051
func assertCompositionDiagnostic(t *testing.T, diagnostics []validate.Diagnostic, code string) {
	t.Helper()
	for _, diagnostic := range diagnostics {
		if diagnostic.Code == code {
			return
		}
	}
	t.Fatalf("expected %s diagnostic, got %+v", code, diagnostics)
}
