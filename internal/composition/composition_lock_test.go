// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

import (
	"os"
	"path/filepath"
	"reflect"
	"strconv"
	"strings"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// startLockRegistry serves one published shared-edge module whose requirement
// text is content, so two registries can publish different content under the
// same version (a moved tag).
// TRLC-LINKS: REQ-EMG-064
func startLockRegistry(t *testing.T, root, content string) *modregistrytest.Registry {
	t.Helper()
	registryRoot := filepath.Join(root, "registry-"+content)
	moduleDir := filepath.Join(registryRoot, "models.atlas.example_shared-edge_v0.1.0")
	testutil.WriteOCIModuleFixture(t, moduleDir, testutil.OCIModule{Path: "models.atlas.example/shared-edge@v0", ModelID: "SHARED-EDGE", CapabilityID: "CAP-SHARED-EDGE", RequirementID: "REQ-SHARED-001"})
	testutil.WriteFile(t, filepath.Join(moduleDir, "NOTES.md"), content+"\n")
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
	testutil.WriteParentWithOCIModule(t, workspace, testutil.OCIParent{Path: "models.atlas.example/shared-edge@v0", Version: "v0.1.0"})
	manifest := filepath.Join(workspace, "engmod.yml")
	lockPath := filepath.Join(workspace, CompositionLockFileName)
	check := ResolutionOptions{Lock: LockModeCheck}

	// No committed lock yet.
	codes := lockCodes(mustResolve(t, manifest, check).Diagnostics)
	requireEqual(t, len(codes), 1)
	requireEqual(t, strings.HasPrefix(codes[0], "composition.lock_missing"), true)
	requireNoFile(t, lockPath) // check mode never writes the lock

	// Write mode writes the lock; check mode then agrees with it.
	mustResolve(t, manifest, ResolutionOptions{Lock: LockModeWrite})
	committed := readText(t, lockPath)
	requireEqual(t, lockCodes(mustResolve(t, manifest, check).Diagnostics), []string(nil))
	// The environment variable selects check mode too.
	t.Setenv(LockEnv, LockModeCheck)
	testutil.WriteFile(t, lockPath, "dependencies: [\n")
	res, _ := GenerateCompositionFromFile(manifest)
	codes = lockCodes(res.Diagnostics)
	requireEqual(t, len(codes), 1)
	requireEqual(t, strings.HasPrefix(codes[0], "composition.lock_invalid"), true)
	t.Setenv(LockEnv, "")

	for _, c := range lockMismatchCases(committed) {
		t.Run(c.name, func(t *testing.T) { checkLockMismatch(t, manifest, c) })
	}
	checkMovedTag(t, root, committed)
}

// lockMismatchCase is a committed lock content that must give one lock_mismatch
// naming the text in want.
type lockMismatchCase struct{ name, lock, want string }

// lockMismatchCases lists the ways a committed lock differs from the resolved one.
func lockMismatchCases(committed string) []lockMismatchCase {
	return []lockMismatchCase{
		{"version changed", strings.Replace(committed, "version: v0.1.0", "version: v0.0.9", 1), "lock pins"},
		{"extra entry", committed + "    - alias: gone\n      path: example.com/gone@v0\n      version: v0.1.0\n      publications: [public]\n      digest: sha256:00\n", "no longer resolves"},
		{"missing entry", "schemaVersion: 1\ndependencies: []\n", "missing from the committed lock"},
	}
}

// checkLockMismatch checks that check mode reports the one mismatch and does
// not rewrite the lock.
func checkLockMismatch(t *testing.T, manifest string, c lockMismatchCase) {
	lockPath := filepath.Join(filepath.Dir(manifest), CompositionLockFileName)
	testutil.WriteFile(t, lockPath, c.lock)
	got := lockCodes(mustResolve(t, manifest, ResolutionOptions{Lock: LockModeCheck}).Diagnostics)
	requireEqual(t, len(got), 1)
	requireEqual(t, strings.HasPrefix(got[0], "composition.lock_mismatch") && strings.Contains(got[0], c.want), true)
	requireEqual(t, readText(t, lockPath), c.lock) // check mode did not rewrite the lock
}

// checkMovedTag checks that a version re-published with other content is
// reported as a changed content digest.
func checkMovedTag(t *testing.T, root, committed string) {
	workspace := filepath.Join(root, "workspace")
	testutil.WriteFile(t, filepath.Join(workspace, CompositionLockFileName), committed)
	requireEqual(t, os.RemoveAll(filepath.Join(workspace, ".engmod")), nil)
	startLockRegistry(t, root, "second")
	got := lockCodes(mustResolve(t, filepath.Join(workspace, "engmod.yml"), ResolutionOptions{Lock: LockModeCheck}).Diagnostics)
	requireEqual(t, len(got), 1)
	requireEqual(t, strings.Contains(got[0], "content digest"), true)
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
	testutil.WriteOCIModuleFixture(t, filepath.Join(root, "repos", "shared-edge"), testutil.OCIModule{Path: "models.atlas.example/shared-edge@v0", ModelID: "SHARED-EDGE", CapabilityID: "CAP-SHARED-EDGE", RequirementID: "REQ-SHARED-001"})
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 3\nreplacements:\n  - module: models.atlas.example/shared-edge@v0\n    path: ./repos/shared-edge\n")
	product := filepath.Join(root, "repos", "product")
	testutil.WriteParentWithOCIModule(t, product, testutil.OCIParent{Path: "models.atlas.example/shared-edge@v0", Version: "v0.1.0"})
	committed := "schemaVersion: 1\ndependencies: []\n# committed\n"
	testutil.WriteFile(t, filepath.Join(product, CompositionLockFileName), committed)

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

// lock_mismatch messages name a dependency as module@version without the
// major suffix twice (shared@v0.2.0, not shared@v0@v0.2.0).
// Mutation: formatting Path+"@"+Version again fails this test.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func TestLockMismatchMessageNamesModuleOnce(t *testing.T) {
	dir := t.TempDir()
	lockPath := filepath.Join(dir, CompositionLockFileName)
	testutil.WriteFile(t, lockPath, "schemaVersion: 1\ndependencies:\n  - alias: s\n    path: example.com/shared@v0\n    version: v0.1.0\n    publications: [public]\n    digest: sha256:aa\n")
	resolved := func(version, digest string) CompositionLockDocument {
		return CompositionLockDocument{SchemaVersion: 1, Dependencies: []CompositionLockDependency{{Alias: "s", Path: "example.com/shared@v0", Version: version, Publications: []string{"public"}, Digest: digest}}}
	}
	for _, lock := range []CompositionLockDocument{resolved("v0.2.0", "sha256:aa"), resolved("v0.1.0", "sha256:bb"), {SchemaVersion: 1, Dependencies: append(resolved("v0.1.0", "sha256:aa").Dependencies, CompositionLockDependency{Alias: "t", Path: "example.com/t@v1", Version: "v1.0.0"})}} {
		diags := checkCompositionLock(lockPath, lock)
		requireEqual(t, len(diags) > 0, true)
		for _, d := range diags {
			requireNamesModuleOnce(t, d.Message)
		}
	}
}

// requireNamesModuleOnce fails the test unless a message names the module as
// module@version without a duplicated major.
func requireNamesModuleOnce(t *testing.T, message string) {
	t.Helper()
	requireEqual(t, strings.Contains(message, "@v0@") || strings.Contains(message, "@v1@"), false)
	requireEqual(t, strings.Contains(message, "example.com/shared@v0.") || strings.Contains(message, "example.com/t@v1.0.0"), true)
}

// A tag moved in the registry is not masked by the module cache: check mode
// verifies each cached module against the registry's current manifest and
// re-fetches it when it changed (lock_mismatch); write mode with the
// workspace off (lock regeneration) and ENGMOD_MODULE_CACHE=refresh re-fetch,
// so the regenerated lock records the registry's content; the default write
// mode reuses the cache offline.
// Mutation: reusing the cache entry unconditionally (no verification in
// check mode, no refresh for workspace off or ENGMOD_MODULE_CACHE=refresh)
// fails this test.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func TestModuleCacheDoesNotMaskMovedTag(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	t.Setenv(ModuleCacheEnv, "")
	root := t.TempDir()
	startLockRegistry(t, root, "first")
	workspace := filepath.Join(root, "workspace")
	testutil.WriteParentWithOCIModule(t, workspace, testutil.OCIParent{Path: "models.atlas.example/shared-edge@v0", Version: "v0.1.0"})
	manifest := filepath.Join(workspace, "engmod.yml")
	digest := func() string { return lockDigest(t, filepath.Join(workspace, CompositionLockFileName)) }
	write := func(options ResolutionOptions) string {
		options.Lock = LockModeWrite
		mustResolve(t, manifest, options)
		return digest()
	}
	first := write(ResolutionOptions{})

	// The tag moves: the same version now has other content.
	startLockRegistry(t, root, "second")
	got := lockCodes(mustResolve(t, manifest, ResolutionOptions{Lock: LockModeCheck}).Diagnostics)
	requireEqual(t, len(got), 1) // check mode with a warm cache sees the moved tag
	requireEqual(t, strings.Contains(got[0], "content digest"), true)

	// Default write mode reuses the (now refreshed) cache; restore the stale
	// entry first by fetching from the first registry again.
	startLockRegistry(t, root, "first")
	requireEqual(t, write(ResolutionOptions{ModuleCache: ModuleCacheRefresh}), first) // refresh restores the first content
	startLockRegistry(t, root, "second")
	requireEqual(t, write(ResolutionOptions{}), first) // default write mode reuses the cache offline
	t.Setenv(ModuleCacheEnv, ModuleCacheRefresh)
	requireEqual(t, write(ResolutionOptions{}) != first, true) // ENGMOD_MODULE_CACHE=refresh re-fetches
	t.Setenv(ModuleCacheEnv, "")

	startLockRegistry(t, root, "first")
	requireEqual(t, write(ResolutionOptions{Workspace: model.WorkspaceOff}), first) // lock regeneration with the workspace off re-fetches
}

// lockDigest is the first content digest recorded in a lock file.
func lockDigest(t *testing.T, lockPath string) string {
	t.Helper()
	_, after, _ := strings.Cut(readText(t, lockPath), "digest: ")
	return strings.TrimSpace(after)
}

// A diagnostic located in a fetched dependency names it as module@version
// and its path inside the module, not the .engmod/modules cache directory.
// Mutation: returning the cache path unchanged fails this test.
// TRLC-LINKS: REQ-EMG-066
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func TestDependencyDiagnosticPathNamesModuleVersion(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, model.WorkspaceOff)
	t.Setenv(LockEnv, "")
	root := t.TempDir()
	startLockRegistry(t, root, "first")
	workspace := filepath.Join(root, "workspace")
	testutil.WriteV2CompositionParent(t, workspace, testutil.V2Parent{Dependencies: `dependencies:
  - alias: shared
    path: models.atlas.example/shared-edge@v0
    version: v0.1.0
    publications: [public, nope]
`, Architecture: `  composition:
    subsystems:
      - id: SUB-SHARED
        dependency: shared
        publication: public
`})
	res := mustResolve(t, filepath.Join(workspace, "engmod.yml"), ResolutionOptions{Lock: LockModeCheck})
	found := firstDiag(res.Diagnostics, "composition.unknown_publication")
	requireEqual(t, found != nil, true)
	requireEqual(t, found.Path, "models.atlas.example/shared-edge@v0.1.0:engmod.yml")
	requireEqual(t, strings.Contains(found.Message, ".engmod"), false)
}

// With ENGMOD_LOCK=check and workspace replacements active the committed
// lock is not compared; the resolution says so with a
// composition.lock_check_skipped warning instead of passing silently.
// Mutation: dropping the lock_check_skipped warning fails this test.
// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, DEP-LOCAL-WORKSPACE
func TestLockCheckWithReplacementsWarnsSkipped(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	root := t.TempDir()
	testutil.WriteOCIModuleFixture(t, filepath.Join(root, "repos", "shared-edge"), testutil.OCIModule{Path: "models.atlas.example/shared-edge@v0", ModelID: "SHARED-EDGE", CapabilityID: "CAP-SHARED-EDGE", RequirementID: "REQ-SHARED-001"})
	testutil.WriteFile(t, filepath.Join(root, model.WorkspaceFileName), "schemaVersion: 3\nreplacements:\n  - module: models.atlas.example/shared-edge@v0\n    path: ./repos/shared-edge\n")
	product := filepath.Join(root, "repos", "product")
	testutil.WriteParentWithOCIModule(t, product, testutil.OCIParent{Path: "models.atlas.example/shared-edge@v0", Version: "v0.1.0"})
	res := mustResolve(t, filepath.Join(product, "engmod.yml"), ResolutionOptions{Lock: LockModeCheck})
	assertNoCompositionErrors(t, res.Diagnostics)
	skipped := firstDiag(res.Diagnostics, "composition.lock_check_skipped")
	requireEqual(t, skipped != nil, true)
	requireEqual(t, skipped.Severity, validate.SeverityWarning)
	requireEqual(t, strings.Contains(skipped.Message, "ENGMOD_WORK=off"), true)
	requireNoFile(t, filepath.Join(product, CompositionLockFileName)) // check mode wrote no lock
}

// The module digest leaves out .engflow (engineering-flow's local
// experiment state and worktrees) like .git and .engmod: adding files under
// it does not change the digest, while a file elsewhere does.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func TestModuleDigestSkipsEngflowState(t *testing.T) {
	dir := t.TempDir()
	testutil.WriteFile(t, filepath.Join(dir, "engmod.yml"), "x")
	before, err := digestModuleDirectory(dir)
	requireEqual(t, err, nil)
	testutil.WriteFile(t, filepath.Join(dir, ".engflow", "worktrees", "exp", "v1", "main.go"), "package x")
	after, _ := digestModuleDirectory(dir)
	requireEqual(t, after, before) // a file under .engflow does not change the digest
	testutil.WriteFile(t, filepath.Join(dir, "main.go"), "package x")
	after, _ = digestModuleDirectory(dir)
	requireEqual(t, after != before, true) // a module file does
}

// lockCase is one committed lock content and the one error expected for it.
type lockCase struct {
	name    string
	content string // "" writes no lock file
	want    string // code of the one expected error, "" for none
}

// diagKey is the part of a diagnostic the lock tests compare.
type diagKey struct {
	code     string
	severity validate.Severity
	path     string
}

// A committed lock whose schemaVersion is missing or not the current one is
// composition.lock_schema_outdated at the lock with a remedy that names
// engws bump and engws sync, and reports nothing else; a lock at the current
// version is not reported, and a missing lock stays composition.lock_missing.
// Mutation: dropping the lockSchemaDiagnostics call in checkCompositionLock,
// or comparing with > instead of !=, fails a row.
// TRLC-LINKS: REQ-EMG-179
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION
func TestLockSchemaVersionIsChecked(t *testing.T) {
	deps := "dependencies: []\n"
	current := "schemaVersion: " + strconv.Itoa(CompositionLockSchemaVersion) + "\n"
	for _, c := range []lockCase{
		{"missing schemaVersion", deps, "composition.lock_schema_outdated"},
		{"schemaVersion 0", "schemaVersion: 0\n" + deps, "composition.lock_schema_outdated"},
		{"older schemaVersion", "schemaVersion: -1\n" + deps, "composition.lock_schema_outdated"},
		{"newer schemaVersion", "schemaVersion: 99\n" + deps, "composition.lock_schema_outdated"},
		{"current schemaVersion", current + deps, ""},
		{"no lock file", "", "composition.lock_missing"},
	} {
		t.Run(c.name, func(t *testing.T) { assertLockCase(t, c) })
	}
}

// assertLockCase checks one lock content against the current resolved lock.
// TRLC-LINKS: REQ-EMG-179
func assertLockCase(t *testing.T, c lockCase) {
	t.Helper()
	path := filepath.Join(t.TempDir(), CompositionLockFileName)
	if c.content != "" {
		testutil.WriteFile(t, path, c.content)
	}
	diags := checkCompositionLock(path, CompositionLockDocument{SchemaVersion: CompositionLockSchemaVersion})
	var got, want []diagKey
	for _, d := range diags {
		got = append(got, diagKey{d.Code, d.Severity, d.Path})
	}
	if c.want != "" {
		want = []diagKey{{c.want, validate.SeverityError, path}}
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("diagnostics %+v, want %+v", diags, want)
	}
	if c.want == "composition.lock_schema_outdated" {
		assertLockRemedy(t, diags[0])
	}
}

// assertLockRemedy checks that the remedy is the table's and names engws
// bump and engws sync.
// TRLC-LINKS: REQ-EMG-179
func assertLockRemedy(t *testing.T, d validate.Diagnostic) {
	t.Helper()
	if d.Remedy != validate.RequiredRemedy(d.Code) || !strings.Contains(d.Remedy, "engws bump") || !strings.Contains(d.Remedy, "engws sync") {
		t.Fatalf("remedy %q must be the table's and name engws bump and engws sync", d.Remedy)
	}
}
