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

// bump changes only the dependency's version line in engmod.yml, the go.mod
// require of the dependency's Go module, and regenerates the lock from the
// registry.
// Mutation: re-marshalling the manifest (comments and quoting lost), editing
// another dependency's version, skipping go mod edit, or not rolling back
// when the lock cannot be written fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestBumpEditsVersionLineGoModAndLock(t *testing.T) {
	work := t.TempDir()
	registryRoot := filepath.Join(work, "registry")
	for _, v := range []string{"v0.1.0", "v0.2.0"} {
		writeModel(t, filepath.Join(registryRoot, strings.ReplaceAll(testFamily, "/", "_")+"_if-a_"+v), "if-a", v)
		writeModel(t, filepath.Join(registryRoot, strings.ReplaceAll(testFamily, "/", "_")+"_if-b_"+v), "if-b", v)
	}
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	t.Setenv("CUE_REGISTRY", testFamily+"="+registry.Host()+"+insecure")

	root := filepath.Join(work, "ws")
	ifA := checkout(t, root, "if-a")
	write(t, filepath.Join(ifA, "go.mod"), "module example.com/go/if-a\n\ngo 1.25\n")
	svc := filepath.Join(root, "svc")
	writeModel(t, svc, "svc", "v0.1.0", dep{"ia", "if-a", "v0.1.0"}, dep{"ib", "if-b", "v0.1.0"})
	manifestPath := filepath.Join(svc, "engmod.yml")
	before, _ := os.ReadFile(manifestPath)
	// Hand formatting that a re-marshal would lose.
	before = []byte(strings.Replace(string(before), "  - alias: ia\n", "  # the interface\n  - alias: ia\n", 1))
	before = []byte(strings.Replace(string(before), "    path: "+modulePathOf("if-a")+"\n    version: v0.1.0", "    path: "+modulePathOf("if-a")+"\n    version: \"v0.1.0\"   # pinned", 1))
	write(t, manifestPath, string(before))
	goModBefore := "module example.com/go/svc\n\ngo 1.25\n\nrequire (\n\texample.com/go/if-a v0.1.0\n\texample.com/other v1.0.0\n)\n"
	write(t, filepath.Join(svc, "go.mod"), goModBefore)

	code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "ia", "v0.2.0")
	if code != 0 {
		t.Fatalf("bump: %d\n%s\n%s", code, out, errOut)
	}
	after, _ := os.ReadFile(manifestPath)
	want := strings.Replace(string(before), `version: "v0.1.0"   # pinned`, `version: "v0.2.0"   # pinned`, 1)
	if string(after) != want {
		t.Fatalf("manifest changed beyond the version line:\n--- got\n%s\n--- want\n%s", after, want)
	}
	goMod, _ := os.ReadFile(filepath.Join(svc, "go.mod"))
	if !strings.Contains(string(goMod), "example.com/go/if-a v0.2.0") || !strings.Contains(string(goMod), "example.com/other v1.0.0") {
		t.Fatalf("go.mod:\n%s", goMod)
	}
	lockData, err := os.ReadFile(filepath.Join(svc, "engmod.lock.yml"))
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

	// An unpublished version cannot be locked: nothing changes.
	manifestNow, _ := os.ReadFile(manifestPath)
	goModNow, _ := os.ReadFile(filepath.Join(svc, "go.mod"))
	if code, _, _ := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "ia", "v0.9.0"); code != 1 {
		t.Fatal("bump to an unpublished version succeeded")
	}
	if m, _ := os.ReadFile(manifestPath); string(m) != string(manifestNow) {
		t.Fatalf("engmod.yml not rolled back:\n%s", m)
	}
	if g, _ := os.ReadFile(filepath.Join(svc, "go.mod")); string(g) != string(goModNow) {
		t.Fatalf("go.mod not rolled back:\n%s", g)
	}
	if code, _, _ := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "nope", "v0.2.0"); code != 1 {
		t.Fatal("unknown alias accepted")
	}
	if code, _, _ := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "ia", "latest"); code != 1 {
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

// bump never locks local content: with the dependency replaced locally by the
// root work file, the lock either equals the registry-only lock (engine
// honours ENGMOD_WORK=off) or bump fails and leaves every file unchanged.
// Mutation: dropping the local-resolution guard writes a lock with the local
// digest and fails this test on an engine without ENGMOD_WORK support.
// TRLC-LINKS: REQ-EMG-068
func TestBumpNeverLocksLocalReplacement(t *testing.T) {
	work := t.TempDir()
	registryRoot := filepath.Join(work, "registry")
	for _, v := range []string{"v0.1.0", "v0.2.0"} {
		writeModel(t, filepath.Join(registryRoot, strings.ReplaceAll(testFamily, "/", "_")+"_if-a_"+v), "if-a", v)
	}
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	t.Setenv("CUE_REGISTRY", testFamily+"="+registry.Host()+"+insecure")
	t.Setenv("ENGMOD_WORK", "")

	root := filepath.Join(work, "ws")
	ifA := filepath.Join(root, "if-a")
	writeModel(t, ifA, "if-a", "v0.2.0")
	write(t, filepath.Join(ifA, "model", "decisions.yml"), "schemaVersion: 2\ndecisions: []\n# local edit\n")
	svc := filepath.Join(root, "svc")
	writeModel(t, svc, "svc", "v0.1.0", dep{"ia", "if-a", "v0.1.0"})
	manifest, _ := os.ReadFile(filepath.Join(svc, "engmod.yml"))

	// Reference: the registry-only lock, without any work file.
	if code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "ia", "v0.2.0"); code != 0 {
		t.Fatalf("reference bump: %s %s", out, errOut)
	}
	reference, _ := os.ReadFile(filepath.Join(svc, "engmod.lock.yml"))
	write(t, filepath.Join(svc, "engmod.yml"), string(manifest))
	os.Remove(filepath.Join(svc, "engmod.lock.yml"))
	os.RemoveAll(filepath.Join(root, ".engmod"))

	write(t, filepath.Join(root, "engmod.work.yml"), "# "+generatedHeader+"\nschemaVersion: 2\nreplacements:\n  - module: "+modulePathOf("if-a")+"\n    path: ./if-a\n")
	code, out, errOut := runMain(t, NewCUERegistry(), "--root", root, "bump", "svc", "ia", "v0.2.0")
	lock, lockErr := os.ReadFile(filepath.Join(svc, "engmod.lock.yml"))
	switch code {
	case 0:
		if string(lock) != string(reference) {
			t.Fatalf("lock pins local content:\n%s\nwant\n%s", lock, reference)
		}
	case 1:
		if lockErr == nil {
			t.Fatalf("failed bump left a lock:\n%s", lock)
		}
		if m, _ := os.ReadFile(filepath.Join(svc, "engmod.yml")); string(m) != string(manifest) {
			t.Fatalf("failed bump changed engmod.yml")
		}
		if !strings.Contains(errOut, "not the registry") {
			t.Fatalf("failure does not say why: %s", errOut)
		}
	default:
		t.Fatalf("bump: %d %s %s", code, out, errOut)
	}
}
