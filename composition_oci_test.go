// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package engmodel

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/labeth/engineering-model-go/model"
)

// TestCompositionResolvesCUEModule verifies that an exact CUE module version is
// fetched from an OCI registry, materialized in .engmod, and reusable offline.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER, REF-CUE-OCI-MODULE-REGISTRY
func TestCompositionResolvesCUEModule(t *testing.T) {
	root := t.TempDir()
	registryRoot := filepath.Join(root, "registry")
	moduleDir := filepath.Join(registryRoot, "models.atlas.example_shared-edge_v0.1.0")
	writeOCIModuleFixture(t, moduleDir, "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatalf("start registry: %v", err)
	}
	t.Setenv("CUE_REGISTRY", registry.Host()+"+insecure")

	workspace := filepath.Join(root, "workspace")
	writeParentWithOCIModule(t, workspace, "models.atlas.example/shared-edge@v0", "v0.1.0")
	manifest := filepath.Join(workspace, "engmod.yml")
	res, err := GenerateCompositionFromFile(manifest)
	if err != nil {
		registry.Close()
		t.Fatalf("compose OCI module: %v", err)
	}
	if res.Root == nil || len(res.Root.Children) != 1 {
		registry.Close()
		t.Fatalf("expected one module child, got %#v", res.Root)
	}
	assertNoCompositionErrors(t, res.Diagnostics)

	registry.Close()
	res, err = GenerateCompositionFromFile(manifest)
	if err != nil {
		t.Fatalf("compose cached OCI module: %v", err)
	}
	if res.Root == nil || len(res.Root.Children) != 1 {
		t.Fatalf("expected cached module child, got %#v", res.Root)
	}
	assertNoCompositionErrors(t, res.Diagnostics)
}

// TestCompositionUsesWorkspaceModuleReplacement verifies an authored OCI
// coordinate can resolve from a sibling repository during local development.
// TRLC-LINKS: REQ-EMG-048, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-MODEL-LOADER
func TestCompositionUsesWorkspaceModuleReplacement(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "repos", "shared-edge")
	writeOCIModuleFixture(t, moduleDir, "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	if err := os.MkdirAll(filepath.Join(moduleDir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 2
replacements:
  - module: models.atlas.example/shared-edge@v0
    path: ./repos/shared-edge
`)
	product := filepath.Join(root, "repos", "product")
	writeParentWithOCIModule(t, product, "models.atlas.example/shared-edge@v0", "v0.1.0")
	if err := os.MkdirAll(filepath.Join(product, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	res, err := GenerateCompositionFromFile(filepath.Join(product, "engmod.yml"))
	if err != nil {
		t.Fatalf("compose replacement: %v", err)
	}
	if res.Root == nil || len(res.Root.Children) != 1 {
		t.Fatalf("expected replacement child, got %#v", res.Root)
	}
	assertNoCompositionErrors(t, res.Diagnostics)
}

// TestCompositionRejectsModuleIdentityMismatch verifies a workspace replacement
// cannot silently substitute a different published module identity.
// TRLC-LINKS: REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func TestCompositionRejectsModuleIdentityMismatch(t *testing.T) {
	root := t.TempDir()
	moduleDir := filepath.Join(root, "repos", "shared-edge")
	writeOCIModuleFixture(t, moduleDir, "models.atlas.example/different@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), `schemaVersion: 2
replacements:
  - module: models.atlas.example/shared-edge@v0
    path: ./repos/shared-edge
`)
	product := filepath.Join(root, "repos", "product")
	writeParentWithOCIModule(t, product, "models.atlas.example/shared-edge@v0", "v0.1.0")

	res, err := GenerateCompositionFromFile(filepath.Join(product, "engmod.yml"))
	if err != nil {
		t.Fatalf("compose replacement: %v", err)
	}
	assertCompositionDiagnostic(t, res.Diagnostics, "composition.module_identity")
}

// TestCompositionRejectsWorkspaceReplacementEscape verifies local replacements
// cannot cross the workspace boundary.
// TRLC-LINKS: REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, CTRL-MCP-PATH-BOUNDARY
func TestCompositionRejectsWorkspaceReplacementEscape(t *testing.T) {
	root := t.TempDir()
	workspace := filepath.Join(root, "workspace")
	outside := filepath.Join(root, "outside")
	writeOCIModuleFixture(t, outside, "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	if err := os.MkdirAll(workspace, 0o755); err != nil {
		t.Fatal(err)
	}
	writeFile(t, filepath.Join(workspace, model.WorkspaceFileName), `schemaVersion: 2
replacements:
  - module: models.atlas.example/shared-edge@v0
    path: ../outside
`)
	product := filepath.Join(workspace, "product")
	writeParentWithOCIModule(t, product, "models.atlas.example/shared-edge@v0", "v0.1.0")

	res, err := GenerateCompositionFromFile(filepath.Join(product, "engmod.yml"))
	if err != nil {
		t.Fatalf("compose replacement: %v", err)
	}
	assertCompositionDiagnostic(t, res.Diagnostics, "composition.workspace_path_escape")
}

// TestCompositionRejectsNonManifestEntryPoint verifies domain documents cannot
// be used as model entry points.
// TRLC-LINKS: REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func TestCompositionRejectsNonManifestEntryPoint(t *testing.T) {
	_, err := GenerateCompositionFromFile(filepath.Join("model", "architecture.yml"))
	if err == nil {
		t.Fatal("expected architecture domain document entry point to be rejected")
	}
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048
func writeOCIModuleFixture(t *testing.T, dir, modulePath, modelID, capabilityID, requirementID string) {
	t.Helper()
	writeV2CompositionModule(t, dir, modulePath, "v0.1.0", modelID, "public", []string{capabilityID}, []string{requirementID})
}

// TRLC-LINKS: REQ-EMG-047, REQ-EMG-048
func writeParentWithOCIModule(t *testing.T, dir, modulePath, version string) {
	t.Helper()
	writeV2CompositionParent(t, dir, `dependencies:
  - alias: shared
    path: `+modulePath+`
    version: `+version+`
    publications: [public]
`, `  composition:
    subsystems:
      - id: SUB-SHARED
        dependency: shared
        publication: public
    allocations:
      - requirement: REQ-PARENT
        to: SUB-SHARED
        target: shared::CAP-SHARED-EDGE
`)
}
