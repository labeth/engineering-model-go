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

const confinementSecret = "TOP-SECRET-CREDENTIAL-7f3a"

// TRLC-LINKS: REQ-EMG-049
func assertNoSecret(t *testing.T, diags []validate.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if strings.Contains(d.Message, confinementSecret) || strings.Contains(d.Path, confinementSecret) {
			t.Fatalf("diagnostic leaks the content of a file outside the module: %+v", d)
		}
	}
}

// TRLC-LINKS: REQ-EMG-049
func escapeDiagnostics(diags []validate.Diagnostic) []validate.Diagnostic {
	var out []validate.Diagnostic
	for _, d := range diags {
		if d.Code == "composition.dependency_path_escape" {
			out = append(out, d)
		}
	}
	return out
}

// A workspace replacement whose document is a symlink to a file outside the
// replacement checkout is rejected the same way, after resolving symlinks.
// Mutation: comparing the lexical path instead of the symlink-resolved path
// fails this test.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, DEP-LOCAL-WORKSPACE
func TestWorkspaceReplacementSymlinkedDocumentIsConfined(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "secret.yml")
	testutil.WriteFile(t, outside, "schemaVersion: 2\ncatalog:\n  "+confinementSecret+": leaked\n")
	moduleDir := filepath.Join(root, "repos", "shared-edge")
	testutil.WriteOCIModuleFixture(t, moduleDir, "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	catalog := filepath.Join(moduleDir, "model", "catalog.yml")
	if err := os.Remove(catalog); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, catalog); err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 2\nreplacements:\n  - module: models.atlas.example/shared-edge@v0\n    path: ./repos/shared-edge\n")
	product := filepath.Join(root, "repos", "product")
	testutil.WriteParentWithOCIModule(t, product, "models.atlas.example/shared-edge@v0", "v0.1.0")

	res, err := GenerateCompositionFromFile(filepath.Join(product, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecret(t, res.Diagnostics)
	got := escapeDiagnostics(res.Diagnostics)
	if len(got) != 1 || !strings.Contains(got[0].Message, "documents.catalog") {
		t.Fatalf("want one documents.catalog escape, got %+v", res.Diagnostics)
	}
}

// The root model keeps its own loader behaviour: a root exclude glob that
// leaves the model directory is not a composition error.
// Mutation: applying the dependency confinement to the root model fails
// this test.
// TRLC-LINKS: REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func TestRootModelPathsAreNotDependencyConfined(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	root := t.TempDir()
	testutil.WriteOCIModuleFixture(t, filepath.Join(root, "repos", "shared-edge"), "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 2\nreplacements:\n  - module: models.atlas.example/shared-edge@v0\n    path: ./repos/shared-edge\n")
	product := filepath.Join(root, "repos", "product")
	testutil.WriteParentWithOCIModule(t, product, "models.atlas.example/shared-edge@v0", "v0.1.0")
	manifest := filepath.Join(product, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	testutil.WriteFile(t, manifest, string(data)+"inferenceHints:\n  exclude:\n    - ../shared-src/**\n")
	res, err := GenerateCompositionFromFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := escapeDiagnostics(res.Diagnostics); len(got) != 0 {
		t.Fatalf("root model paths must not be dependency-confined: %+v", got)
	}
	assertNoCompositionErrors(t, res.Diagnostics)
}
