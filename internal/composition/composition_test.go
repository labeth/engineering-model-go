// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048, REQ-EMG-049, REQ-EMG-051, REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestCompositionResolvesManifestPublicationAndWritesLock(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	testutil.WriteV2CompositionModule(t, child, testutil.CompositionModule{Path: "example.com/child@v1", Version: "v1.2.3", ModelID: "CHILD", Publication: "public", ArchitectureIDs: []string{"CAP-CHILD"}, RequirementIDs: []string{"REQ-CHILD"}})
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 3
replacements:
  - module: example.com/child@v1
    path: ./child
`)
	parent := filepath.Join(root, "parent")
	testutil.WriteV2CompositionParent(t, parent, testutil.V2Parent{Dependencies: `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
`, Architecture: `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
`})

	manifest := filepath.Join(parent, "engmod.yml")
	result := mustResolve(t, manifest, ResolutionOptions{})
	assertNoCompositionErrors(t, result.Diagnostics)
	requireEqual(t, result.Root != nil && len(result.Root.Children) == 1, true)
	requireEqual(t, len(result.Provenance), 2)
	requireEqual(t, []string{result.Provenance[0].Alias, result.Provenance[0].ResolvedDir}, []string{"child", child})
	// Workspace replacements are active, so the resolved lock is local only.
	lockPath := filepath.Join(parent, CompositionLocalLockFileName)
	first := readText(t, lockPath)
	requireEqual(t, strings.Contains(first, "sha256:") && strings.Contains(first, "alias: child"), true)
	mustResolve(t, manifest, ResolutionOptions{})
	requireEqual(t, readText(t, lockPath), first) // lock generation is deterministic
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func TestCompositionRejectsUnpublishedQualifiedReference(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	testutil.WriteV2CompositionModule(t, child, testutil.CompositionModule{Path: "example.com/child@v1", Version: "v1.2.3", ModelID: "CHILD", Publication: "public", ArchitectureIDs: []string{"CAP-CHILD"}, RequirementIDs: nil})
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 3
replacements:
  - module: example.com/child@v1
    path: ./child
`)
	parent := filepath.Join(root, "parent")
	testutil.WriteV2CompositionParent(t, parent, testutil.V2Parent{Dependencies: `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
`, Architecture: `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
    satisfactions:
      - need: child::CAP-CHILD
        by: child::REQ-CHILD
`})
	result, err := GenerateCompositionFromFile(filepath.Join(parent, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertCompositionDiagnostic(t, result.Diagnostics, "composition.unpublished_reference")
}

// TRLC-LINKS: REQ-EMG-049, REQ-EMG-051
func TestCompositionRejectsUnpublishedBehaviorReference(t *testing.T) {
	root := t.TempDir()
	child := filepath.Join(root, "child")
	testutil.WriteV2CompositionModule(t, child, testutil.CompositionModule{Path: "example.com/child@v1", Version: "v1.2.3", ModelID: "CHILD", Publication: "public", ArchitectureIDs: []string{"CAP-CHILD"}, RequirementIDs: nil})
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 3\nreplacements:\n  - module: example.com/child@v1\n    path: ./child\n")
	parent := filepath.Join(root, "parent")
	testutil.WriteV2CompositionParent(t, parent, testutil.V2Parent{Dependencies: `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
`, Architecture: `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
`})
	testutil.WriteFile(t, filepath.Join(parent, "model", "behavior.yml"), `schemaVersion: 3
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
	testutil.WriteV2CompositionModule(t, child, testutil.CompositionModule{Path: "example.com/child@v1", Version: "v1.2.3", ModelID: "CHILD", Publication: "public", ArchitectureIDs: []string{"CAP-CHILD"}, RequirementIDs: nil})
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 3
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
			testutil.WriteV2CompositionParent(t, parent, testutil.V2Parent{Dependencies: test.dependencies, Architecture: `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: ` + test.publication + `
`})
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
	testutil.WriteV2CompositionModule(t, child, testutil.CompositionModule{Path: "example.com/child@v1", Version: "v1.2.3", ModelID: "CHILD", Publication: "public", ArchitectureIDs: []string{"CAP-CHILD"}, RequirementIDs: nil})
	manifestPath := filepath.Join(child, "engmod.yml")
	manifest, err := os.ReadFile(manifestPath)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, manifestPath, strings.Replace(string(manifest), "architecture: [CAP-CHILD]", "architecture: [REQ-CHILD]", 1))
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 3
replacements:
  - module: example.com/child@v1
    path: ./child
`)
	parent := filepath.Join(root, "parent")
	testutil.WriteV2CompositionParent(t, parent, testutil.V2Parent{Dependencies: `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.2.3
    publications: [public]
`, Architecture: `  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
`})
	result, err := GenerateCompositionFromFile(filepath.Join(parent, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertCompositionDiagnostic(t, result.Diagnostics, "composition.publication_unowned_id")
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
