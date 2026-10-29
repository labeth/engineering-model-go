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

// startLockRegistry serves one published shared-edge module whose requirement
// text is content, so two registries can publish different content under the
// same version (a moved tag).
// TRLC-LINKS: REQ-EMG-064
func startLockRegistry(t *testing.T, root, content string) *modregistrytest.Registry {
	t.Helper()
	registryRoot := filepath.Join(root, "registry-"+content)
	moduleDir := filepath.Join(registryRoot, "models.atlas.example_shared-edge_v0.1.0")
	writeOCIModuleFixture(t, moduleDir, "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	writeFile(t, filepath.Join(moduleDir, "NOTES.md"), content+"\n")
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatalf("start registry: %v", err)
	}
	t.Cleanup(registry.Close)
	t.Setenv("CUE_REGISTRY", registry.Host()+"+insecure")
	return registry
}

// TRLC-LINKS: REQ-EMG-064
func lockCodes(diags []validate.Diagnostic) []string {
	var codes []string
	for _, d := range diags {
		if strings.HasPrefix(d.Code, "composition.lock_") {
			codes = append(codes, d.Code+": "+d.Message)
		}
	}
	return codes
}

// TestCompositionLockCheckMode verifies ENGMOD_LOCK=check compares the
// resolved lock with the committed one, reports missing, invalid, changed
// version, extra entries and a moved tag (same version, other content), and
// never writes the lock.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func TestCompositionLockCheckMode(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, model.WorkspaceOff)
	t.Setenv(LockEnv, "")
	root := t.TempDir()
	startLockRegistry(t, root, "first")
	workspace := filepath.Join(root, "workspace")
	writeParentWithOCIModule(t, workspace, "models.atlas.example/shared-edge@v0", "v0.1.0")
	manifest := filepath.Join(workspace, "engmod.yml")
	lockPath := filepath.Join(workspace, CompositionLockFileName)
	check := ResolutionOptions{Lock: LockModeCheck}

	// No committed lock yet.
	res, err := GenerateCompositionFromFileWith(manifest, check)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockCodes(res.Diagnostics); len(got) != 1 || !strings.HasPrefix(got[0], "composition.lock_missing") {
		t.Fatalf("missing lock: %v", got)
	}
	if _, err := os.Stat(lockPath); !os.IsNotExist(err) {
		t.Fatal("check mode wrote engmod.lock.yml")
	}

	// Write mode writes the lock; check mode then agrees with it.
	if _, err := GenerateCompositionFromFileWith(manifest, ResolutionOptions{Lock: LockModeWrite}); err != nil {
		t.Fatal(err)
	}
	committed, err := os.ReadFile(lockPath)
	if err != nil {
		t.Fatal(err)
	}
	if res, _ = GenerateCompositionFromFileWith(manifest, check); len(lockCodes(res.Diagnostics)) != 0 {
		t.Fatalf("matching lock reported: %v", lockCodes(res.Diagnostics))
	}
	// The environment variable selects check mode too.
	t.Setenv(LockEnv, LockModeCheck)
	writeFile(t, lockPath, "dependencies: [\n")
	if res, _ = GenerateCompositionFromFile(manifest); len(lockCodes(res.Diagnostics)) != 1 || !strings.HasPrefix(lockCodes(res.Diagnostics)[0], "composition.lock_invalid") {
		t.Fatalf("invalid lock: %v", lockCodes(res.Diagnostics))
	}
	t.Setenv(LockEnv, "")

	cases := []struct {
		name, lock, want string
	}{
		{"version changed", strings.Replace(string(committed), "version: v0.1.0", "version: v0.0.9", 1), "lock pins"},
		{"extra entry", string(committed) + "    - alias: gone\n      path: example.com/gone@v0\n      version: v0.1.0\n      publications: [public]\n      digest: sha256:00\n", "no longer resolves"},
		{"missing entry", "schemaVersion: 1\ndependencies: []\n", "missing from the committed lock"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			writeFile(t, lockPath, c.lock)
			res, err := GenerateCompositionFromFileWith(manifest, check)
			if err != nil {
				t.Fatal(err)
			}
			got := lockCodes(res.Diagnostics)
			if len(got) != 1 || !strings.HasPrefix(got[0], "composition.lock_mismatch") || !strings.Contains(got[0], c.want) {
				t.Fatalf("got %v, want one lock_mismatch containing %q", got, c.want)
			}
			if data, _ := os.ReadFile(lockPath); string(data) != c.lock {
				t.Fatal("check mode rewrote engmod.lock.yml")
			}
		})
	}

	// Moved tag: the same version is re-published with other content.
	writeFile(t, lockPath, string(committed))
	if err := os.RemoveAll(filepath.Join(workspace, ".engmod")); err != nil {
		t.Fatal(err)
	}
	startLockRegistry(t, root, "second")
	res, err = GenerateCompositionFromFileWith(manifest, check)
	if err != nil {
		t.Fatal(err)
	}
	if got := lockCodes(res.Diagnostics); len(got) != 1 || !strings.Contains(got[0], "content digest") {
		t.Fatalf("moved tag: %v", got)
	}
}

// TestCompositionLockWorkspaceModeWritesLocalLock verifies that active
// workspace replacements leave engmod.lock.yml byte-identical and write
// engmod.lock.local.yml instead.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, DEP-LOCAL-WORKSPACE
func TestCompositionLockWorkspaceModeWritesLocalLock(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	root := t.TempDir()
	writeOCIModuleFixture(t, filepath.Join(root, "repos", "shared-edge"), "models.atlas.example/shared-edge@v0", "SHARED-EDGE", "CAP-SHARED-EDGE", "REQ-SHARED-001")
	writeFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 2\nreplacements:\n  - module: models.atlas.example/shared-edge@v0\n    path: ./repos/shared-edge\n")
	product := filepath.Join(root, "repos", "product")
	writeParentWithOCIModule(t, product, "models.atlas.example/shared-edge@v0", "v0.1.0")
	committed := "schemaVersion: 1\ndependencies: []\n# committed\n"
	writeFile(t, filepath.Join(product, CompositionLockFileName), committed)

	res, err := GenerateCompositionFromFile(filepath.Join(product, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	assertNoCompositionErrors(t, res.Diagnostics)
	if data, _ := os.ReadFile(filepath.Join(product, CompositionLockFileName)); string(data) != committed {
		t.Fatalf("workspace mode modified engmod.lock.yml:\n%s", data)
	}
	local, err := os.ReadFile(filepath.Join(product, CompositionLocalLockFileName))
	if err != nil || !strings.Contains(string(local), "alias: shared") {
		t.Fatalf("local lock not written: %v\n%s", err, local)
	}
	// Workspace off restores the registry-only resolution (here: a closed
	// local port, so the fetch fails) instead of reading the replacement.
	t.Setenv("CUE_REGISTRY", "127.0.0.1:1+insecure")
	res, err = GenerateCompositionFromFileWith(filepath.Join(product, "engmod.yml"), ResolutionOptions{Workspace: model.WorkspaceOff})
	if err != nil {
		t.Fatal(err)
	}
	assertCompositionDiagnostic(t, res.Diagnostics, "composition.module_fetch_failed")
}
