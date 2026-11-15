// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	"gopkg.in/yaml.v3"
)

// bumpRegistrySpec says which models startBumpRegistry publishes under work.
type bumpRegistrySpec struct {
	work            string
	names, versions []string
}

// startBumpRegistry publishes a model per name and version into a test
// registry and points CUE_REGISTRY at it.
func startBumpRegistry(t *testing.T, spec bumpRegistrySpec) {
	t.Helper()
	registryRoot := filepath.Join(spec.work, "registry")
	for _, v := range spec.versions {
		for _, name := range spec.names {
			writeModel(t, modelSpec{Dir: filepath.Join(registryRoot, strings.ReplaceAll(testFamily, "/", "_")+"_"+name+"_"+v), Name: name, Version: v})
		}
	}
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(registry.Close)
	t.Setenv("CUE_REGISTRY", testFamily+"="+registry.Host()+"+insecure")
}

// bumpWorkspace is a workspace root holding the consumer svc.
type bumpWorkspace struct {
	root, svc string
	manifest  []byte // engmod.yml of svc before the bump
}

// newBumpWorkspace writes if-a (with a go.mod) and the consumer svc, whose
// manifest has hand formatting that a re-marshal would lose and a go.mod
// requiring if-a's Go module.
func newBumpWorkspace(t *testing.T, work string) bumpWorkspace {
	t.Helper()
	root := filepath.Join(work, "ws")
	ifA := checkout(t, repoSpec{Root: root, Name: "if-a"})
	write(t, filepath.Join(ifA, "go.mod"), "module example.com/go/if-a\n\ngo 1.25\n")
	svc := filepath.Join(root, "svc")
	writeModel(t, modelSpec{Dir: svc, Name: "svc", Version: "v0.1.0"}, dep{"ia", "if-a", "v0.1.0"}, dep{"ib", "if-b", "v0.1.0"})
	manifestPath := filepath.Join(svc, "engmod.yml")
	before, _ := os.ReadFile(manifestPath)
	before = []byte(strings.Replace(string(before), "  - alias: ia\n", "  # the interface\n  - alias: ia\n", 1))
	before = []byte(strings.Replace(string(before), "    path: "+modulePathOf("if-a")+"\n    version: v0.1.0", "    path: "+modulePathOf("if-a")+"\n    version: \"v0.1.0\"   # pinned", 1))
	write(t, manifestPath, string(before))
	write(t, filepath.Join(svc, "go.mod"), "module example.com/go/svc\n\ngo 1.25\n\nrequire (\n\texample.com/go/if-a v0.1.0\n\texample.com/other v1.0.0\n)\n")
	return bumpWorkspace{root: root, svc: svc, manifest: before}
}

// bump runs engws bump svc alias version.
func (w bumpWorkspace) bump(t *testing.T, alias, version string) (int, string, string) {
	t.Helper()
	return runMain(t, NewCUERegistry(), "--root", w.root, "bump", "svc", alias, version)
}

// read returns a file of svc.
func (w bumpWorkspace) read(name string) string {
	data, _ := os.ReadFile(filepath.Join(w.svc, name))
	return string(data)
}

// requireVersionLineOnly requires engmod.yml to differ from before in the
// bumped version line only.
func (w bumpWorkspace) requireVersionLineOnly(t *testing.T) {
	t.Helper()
	want := strings.Replace(string(w.manifest), `version: "v0.1.0"   # pinned`, `version: "v0.2.0"   # pinned`, 1)
	if got := w.read("engmod.yml"); got != want {
		t.Fatalf("manifest changed beyond the version line:\n--- got\n%s\n--- want\n%s", got, want)
	}
}

// requireGoFilesBumped requires the go.mod require and the go.sum entry of
// the bumped Go module and the other require unchanged.
func (w bumpWorkspace) requireGoFilesBumped(t *testing.T) {
	t.Helper()
	if goMod := w.read("go.mod"); !strings.Contains(goMod, "example.com/go/if-a v0.2.0") || !strings.Contains(goMod, "example.com/other v1.0.0") {
		t.Fatalf("go.mod:\n%s", goMod)
	}
	if goSum := w.read("go.sum"); !strings.Contains(goSum, "example.com/go/if-a v0.2.0/go.mod h1:") {
		t.Fatalf("go.sum lacks the bumped version:\n%s", goSum)
	}
}

// requireLockVersions requires the lock to pin ia at v0.2.0 and ib at v0.1.0.
func (w bumpWorkspace) requireLockVersions(t *testing.T, out string) {
	t.Helper()
	lockData, err := os.ReadFile(filepath.Join(w.svc, "engmod.lock.yml"))
	if err != nil {
		t.Fatalf("lock not written: %v\n%s", err, out)
	}
	var lock struct {
		Dependencies []struct{ Alias, Version, Digest string } `yaml:"dependencies"`
	}
	if err := yaml.Unmarshal(lockData, &lock); err != nil {
		t.Fatal(err)
	}
	got := map[string]string{}
	for _, d := range lock.Dependencies {
		got[d.Alias] = d.Version
	}
	if got["ia"] != "v0.2.0" || got["ib"] != "v0.1.0" {
		t.Fatalf("lock: %s", lockData)
	}
}

// bumpedFiles are the files of svc a bump changes together or not at all,
// with the message of a failed rollback of each.
var bumpedFiles = []struct{ name, failure string }{
	{"engmod.yml", "engmod.yml not rolled back"},
	{"go.mod", "go.mod not rolled back"},
	{"go.sum", "go.sum not rolled back"},
	{"engmod.lock.yml", "lock changed"},
}

// requireRefusedBumpsRollBack requires a version without a Go release
// (v0.9.0) or without a model release (v0.3.0) to fail and change nothing.
func (w bumpWorkspace) requireRefusedBumpsRollBack(t *testing.T) {
	t.Helper()
	before := map[string]string{}
	for _, f := range bumpedFiles {
		before[f.name] = w.read(f.name)
	}
	for _, v := range []string{"v0.9.0", "v0.3.0"} {
		code, out, errOut := w.bump(t, "ia", v)
		if code != 1 {
			t.Fatalf("bump to %s succeeded:\n%s", v, out)
		}
		requireBumpFailureNamed(t, v, errOut)
		w.requireFiles(t, v, before)
	}
}

// requireFiles requires every bumped file to be as recorded in before.
func (w bumpWorkspace) requireFiles(t *testing.T, version string, before map[string]string) {
	t.Helper()
	for _, f := range bumpedFiles {
		if got := w.read(f.name); got != before[f.name] {
			t.Fatalf("%s: %s:\n%s", version, f.failure, got)
		}
	}
}

// requireBumpFailureNamed requires the failure of a refused bump to say why.
func requireBumpFailureNamed(t *testing.T, version, errOut string) {
	t.Helper()
	if version == "v0.9.0" && !strings.Contains(errOut, "go get") {
		t.Fatalf("go get failure not reported: %s", errOut)
	}
	if version == "v0.3.0" && !strings.Contains(errOut, "svc does not compose against "+strings.TrimSuffix(modulePathOf("if-a"), "@v0")+"@v0.3.0 (") {
		t.Fatalf("composition failure must name the consumer and the module version once: %s", errOut)
	}
}

// bump changes only the dependency's version line in engmod.yml, the go.mod
// require of the dependency's Go module and its go.sum entries (so
// GOWORK=off go builds), and regenerates the lock from the registry.
// Mutation: re-marshalling the manifest (comments and quoting lost), editing
// another dependency's version, skipping the go.mod update, leaving go.sum
// without the new version, or not rolling back engmod.yml, go.mod and go.sum
// when go get or the lock fails fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestBumpEditsVersionLineGoModAndLock(t *testing.T) {
	work := t.TempDir()
	// Go: v0.9.0 has no Go release, v0.3.0 has no model release.
	goProxy(t, "example.com/go/if-a@v0.1.0", "example.com/go/if-a@v0.2.0", "example.com/go/if-a@v0.3.0", "example.com/other@v1.0.0")
	startBumpRegistry(t, bumpRegistrySpec{work: work, names: []string{"if-a", "if-b"}, versions: []string{"v0.1.0", "v0.2.0", "v0.9.0"}})
	ws := newBumpWorkspace(t, work)

	code, out, errOut := ws.bump(t, "ia", "v0.2.0")
	if code != 0 {
		t.Fatalf("bump: %d\n%s\n%s", code, out, errOut)
	}
	ws.requireVersionLineOnly(t)
	ws.requireGoFilesBumped(t)
	ws.requireLockVersions(t, out)
	ws.requireRefusedBumpsRollBack(t)
	if code, _, _ := ws.bump(t, "nope", "v0.2.0"); code != 1 {
		t.Fatal("unknown alias accepted")
	}
	if code, _, _ := ws.bump(t, "ia", "latest"); code != 1 {
		t.Fatal("non-semver version accepted")
	}
}

// The version edit finds the alias in flow style and keeps the quote style.
// Mutation: a regex on "- alias:" lines instead of the YAML node tree fails
// the flow-style case.
// TRLC-LINKS: REQ-EMG-068
func TestSetDependencyVersionByNode(t *testing.T) {
	in := "dependencies:\n  - {alias: a, path: x@v0, version: 'v1.0.0'}\n  - alias: b\n    version: v1.0.0\n"
	out, err := setDependencyVersion([]byte(in), "a", "v1.1.0")
	if err != nil {
		t.Fatal(err)
	}
	if string(out) != strings.Replace(in, "'v1.0.0'", "'v1.1.0'", 1) {
		t.Fatalf("got\n%s", out)
	}
	out, err = setDependencyVersion([]byte(in), "b", "v2.0.0")
	if err != nil || !strings.HasSuffix(string(out), "    version: v2.0.0\n") || !strings.Contains(string(out), "'v1.0.0'") {
		t.Fatalf("got %v\n%s", err, out)
	}
	if _, err := setDependencyVersion([]byte("dependencies:\n  - alias: c\n    version:\n      v1.0.0\n"), "c", "v2.0.0"); err == nil {
		t.Fatal("multi-line value accepted")
	}
}

// requireLocalLockIsRegistryLock requires the lock to equal the
// registry-only reference lock.
func requireLocalLockIsRegistryLock(t *testing.T, svc, reference string) {
	t.Helper()
	lock, _ := os.ReadFile(filepath.Join(svc, "engmod.lock.yml"))
	if string(lock) != reference {
		t.Fatalf("lock pins local content:\n%s\nwant\n%s", lock, reference)
	}
}

// localBump is the consumer svc of a bump with a local replacement and its
// manifest before the bump.
type localBump struct {
	svc      string
	manifest []byte
}

// requireRefused requires a failed bump to leave no lock, the manifest
// unchanged and a message that says why.
func (b localBump) requireRefused(t *testing.T, errOut string) {
	t.Helper()
	if lock, err := os.ReadFile(filepath.Join(b.svc, "engmod.lock.yml")); err == nil {
		t.Fatalf("failed bump left a lock:\n%s", lock)
	}
	if m, _ := os.ReadFile(filepath.Join(b.svc, "engmod.yml")); string(m) != string(b.manifest) {
		t.Fatalf("failed bump changed engmod.yml")
	}
	if !strings.Contains(errOut, "not the registry") {
		t.Fatalf("failure does not say why: %s", errOut)
	}
}

// bump never locks local content: with the dependency replaced locally by the
// root work file, the lock either equals the registry-only lock (engine
// honours ENGMOD_WORK=off) or bump fails and leaves every file unchanged.
// Mutation: dropping the local-resolution guard writes a lock with the local
// digest and fails this test on an engine without ENGMOD_WORK support.
// TRLC-LINKS: REQ-EMG-068
func TestBumpNeverLocksLocalReplacement(t *testing.T) {
	work := t.TempDir()
	startBumpRegistry(t, bumpRegistrySpec{work: work, names: []string{"if-a"}, versions: []string{"v0.1.0", "v0.2.0"}})
	t.Setenv("ENGMOD_WORK", "")

	root := filepath.Join(work, "ws")
	ifA := filepath.Join(root, "if-a")
	writeModel(t, modelSpec{Dir: ifA, Name: "if-a", Version: "v0.2.0"})
	write(t, filepath.Join(ifA, "model", "decisions.yml"), "schemaVersion: 3\ndecisions: []\n# local edit\n")
	svc := filepath.Join(root, "svc")
	writeModel(t, modelSpec{Dir: svc, Name: "svc", Version: "v0.1.0"}, dep{"ia", "if-a", "v0.1.0"})
	manifest, _ := os.ReadFile(filepath.Join(svc, "engmod.yml"))

	// Reference: the registry-only lock, without any work file.
	if code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "ia", "v0.2.0"); code != 0 {
		t.Fatalf("reference bump: %s %s", out, errOut)
	}
	reference, _ := os.ReadFile(filepath.Join(svc, "engmod.lock.yml"))
	write(t, filepath.Join(svc, "engmod.yml"), string(manifest))
	os.Remove(filepath.Join(svc, "engmod.lock.yml"))
	os.RemoveAll(filepath.Join(root, ".engmod"))

	write(t, filepath.Join(root, "engmod.work.yml"), "# "+generatedHeader+"\nschemaVersion: 3\nreplacements:\n  - module: "+modulePathOf("if-a")+"\n    path: ./if-a\n")
	code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "ia", "v0.2.0")
	switch code {
	case 0:
		requireLocalLockIsRegistryLock(t, svc, string(reference))
	case 1:
		localBump{svc: svc, manifest: manifest}.requireRefused(t, errOut)
	default:
		t.Fatalf("bump: %d %s %s", code, out, errOut)
	}
}

// bump refuses a Go module path that is not a valid module path (here one
// starting with "-", which go get would read as a flag) before running go
// get, and leaves every file unchanged.
// Mutation: dropping the module.CheckPath validation of the Go module fails
// this test.
// TRLC-LINKS: REQ-EMG-068
func TestBumpRejectsInvalidGoModulePath(t *testing.T) {
	root := t.TempDir()
	ifA := filepath.Join(root, "if-a")
	writeModel(t, modelSpec{Dir: ifA, Name: "if-a", Version: "v0.1.0"})
	write(t, filepath.Join(ifA, "go.mod"), "module -toolexec=/bin/false\n\ngo 1.25\n")
	svc := filepath.Join(root, "svc")
	writeModel(t, modelSpec{Dir: svc, Name: "svc", Version: "v0.1.0"}, dep{"ia", "if-a", "v0.1.0"})
	goMod := "module example.com/go/svc\n\ngo 1.25\n\nrequire -toolexec=/bin/false v0.1.0\n"
	write(t, filepath.Join(svc, "go.mod"), goMod)
	manifest, _ := os.ReadFile(filepath.Join(svc, "engmod.yml"))
	t.Setenv("CUE_REGISTRY", "127.0.0.1:1+insecure")
	code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "ia", "v0.2.0")
	if code != 1 || !strings.Contains(errOut, "invalid Go module path") || strings.Contains(errOut, "go get") {
		t.Fatalf("want refusal before go get: %d\n%s\n%s", code, out, errOut)
	}
	if m, _ := os.ReadFile(filepath.Join(svc, "engmod.yml")); string(m) != string(manifest) {
		t.Fatal("engmod.yml changed")
	}
	if g, _ := os.ReadFile(filepath.Join(svc, "go.mod")); string(g) != goMod {
		t.Fatal("go.mod changed")
	}
}
