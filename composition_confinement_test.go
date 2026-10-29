// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
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

// A published module whose manifest points a document outside the module
// (documents.catalog: ../../../../secret.yml) or names an absolute code
// source is rejected before any of its documents is read: one
// composition.dependency_path_escape per field naming the alias and field,
// and the content of the target file appears in no diagnostic.
// Mutation: dropping the dependency manifest confinement check in
// resolveDependency fails this test.
// TRLC-LINKS: REQ-EMG-049, REQ-EMG-017
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY, REF-CUE-OCI-MODULE-REGISTRY
func TestRegistryDependencyPathsAreConfinedToTheModule(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	t.Setenv(ModuleCacheEnv, "")
	root := t.TempDir()
	// workspace/.engmod/modules/<entry>/../../../../secret.yml is root/secret.yml.
	writeFile(t, filepath.Join(root, "secret.yml"), "schemaVersion: 2\ncatalog:\n  "+confinementSecret+": leaked\n")
	registryRoot := filepath.Join(root, "registry")
	moduleDir := filepath.Join(registryRoot, "models.atlas.example_shared-edge_v0.1.0")
	writeOCIModuleFixture(t, moduleDir, "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	manifest := filepath.Join(moduleDir, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	escaped := strings.Replace(string(data), "catalog: model/catalog.yml", "catalog: ../../../../secret.yml", 1)
	escaped += "inferenceHints:\n  codeSources:\n    - " + root + "\n"
	writeFile(t, manifest, escaped)
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatalf("start registry: %v", err)
	}
	defer registry.Close()
	t.Setenv("CUE_REGISTRY", registry.Host()+"+insecure")

	workspace := filepath.Join(root, "workspace")
	writeParentWithOCIModule(t, workspace, "models.atlas.example/shared-edge@v0", "v0.1.0")
	res, err := GenerateCompositionFromFile(filepath.Join(workspace, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecret(t, res.Diagnostics)
	got := escapeDiagnostics(res.Diagnostics)
	var fields []string
	for _, d := range got {
		if !strings.Contains(d.Message, `"shared"`) || d.Severity != validate.SeverityError {
			t.Fatalf("escape diagnostic must be an error naming the alias: %+v", d)
		}
		fields = append(fields, d.Message)
	}
	joined := strings.Join(fields, "\n")
	if len(got) != 2 || !strings.Contains(joined, "documents.catalog") || !strings.Contains(joined, "inferenceHints.codeSources[0]") {
		t.Fatalf("want documents.catalog and inferenceHints.codeSources[0] escapes, got %+v", res.Diagnostics)
	}
	if res.Root != nil && len(res.Root.Children) != 0 {
		t.Fatal("a dependency with escaping paths must not be loaded")
	}
	// The trace never follows the rejected code source either.
	_, traceDiags, err := BuildTraceMatrixFromFiles(filepath.Join(workspace, "engmod.yml"), filepath.Join(workspace, "model", "requirements.yml"), "")
	if err != nil {
		t.Fatal(err)
	}
	assertNoSecret(t, traceDiags)
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
	writeFile(t, outside, "schemaVersion: 2\ncatalog:\n  "+confinementSecret+": leaked\n")
	moduleDir := filepath.Join(root, "repos", "shared-edge")
	writeOCIModuleFixture(t, moduleDir, "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	catalog := filepath.Join(moduleDir, "model", "catalog.yml")
	if err := os.Remove(catalog); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, catalog); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 2\nreplacements:\n  - module: models.atlas.example/shared-edge@v0\n    path: ./repos/shared-edge\n")
	product := filepath.Join(root, "repos", "product")
	writeParentWithOCIModule(t, product, "models.atlas.example/shared-edge@v0", "v0.1.0")

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

// The root model keeps its own loader behaviour: a root code source outside
// the model directory (the template's ../ layouts) is not a composition
// error.
// Mutation: applying the dependency confinement to the root model fails
// this test.
// TRLC-LINKS: REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func TestRootModelPathsAreNotDependencyConfined(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	root := t.TempDir()
	writeOCIModuleFixture(t, filepath.Join(root, "repos", "shared-edge"), "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 2\nreplacements:\n  - module: models.atlas.example/shared-edge@v0\n    path: ./repos/shared-edge\n")
	product := filepath.Join(root, "repos", "product")
	writeParentWithOCIModule(t, product, "models.atlas.example/shared-edge@v0", "v0.1.0")
	manifest := filepath.Join(product, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	writeFile(t, manifest, string(data)+"inferenceHints:\n  codeSources:\n    - ../shared-src\n")
	res, err := GenerateCompositionFromFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if got := escapeDiagnostics(res.Diagnostics); len(got) != 0 {
		t.Fatalf("root model paths must not be dependency-confined: %+v", got)
	}
	assertNoCompositionErrors(t, res.Diagnostics)
}
