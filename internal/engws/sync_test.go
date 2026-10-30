// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// sync writes engmod.work.yml at the root with only the local checkouts and
// go.work with every checkout that has go.mod (a non-local checkout is at its
// published tag, so go test and go run work in it), and git status stays
// clean in every checkout.
// Mutation: writing the work file into a checkout, listing every checkout in
// engmod.work.yml, or listing only local checkouts in go.work fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestSyncWritesRootFilesOnlyForLocalCheckouts(t *testing.T) {
	root := t.TempDir()
	a := checkout(t, root, "if-a")
	write(t, filepath.Join(a, "go.mod"), "module example.com/go/if-a\n\ngo 1.25\n")
	runGit(t, a, "add", "go.mod")
	runGit(t, a, "commit", "-q", "-m", "go")
	runGit(t, a, "tag", "-f", "v0.1.0") // still tagged -> not local
	b := checkout(t, root, "svc-b", dep{"ia", "if-a", "v0.1.0"})
	write(t, filepath.Join(b, "go.mod"), "module example.com/go/svc-b\n\ngo 1.25\n")
	runGit(t, b, "add", "go.mod")
	runGit(t, b, "commit", "-q", "-m", "go") // untagged -> local
	code, out, errOut := runMain(t, allPublished("if-a", "svc-b"), "--root", root, "sync")
	if code != 0 {
		t.Fatalf("sync: %d\n%s\n%s", code, out, errOut)
	}
	work, err := os.ReadFile(filepath.Join(root, model.WorkspaceFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(string(work), "# "+generatedHeader+"\nschemaVersion: 2\n") ||
		!strings.Contains(string(work), "module: "+modulePathOf("svc-b")+"\n    path: ./svc-b") ||
		strings.Contains(string(work), "if-a") {
		t.Fatalf("work file:\n%s", work)
	}
	if _, err := model.LoadWorkspace(filepath.Join(root, model.WorkspaceFileName)); err != nil {
		t.Fatalf("library rejects work file: %v", err)
	}
	goWork, err := os.ReadFile(filepath.Join(root, GoWorkFileName))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(goWork), "\t./svc-b\n") || !strings.Contains(string(goWork), "\t./if-a\n") || !strings.Contains(string(goWork), "\ngo 1.") {
		t.Fatalf("go.work:\n%s", goWork)
	}
	if !strings.Contains(out, "1 local of 2") {
		t.Fatalf("sync output:\n%s", out)
	}
	assertClean(t, a)
	assertClean(t, b)
}

// go.work is removed when no checkout has go.mod, and the work file then
// lists no replacements.
// Mutation: leaving a stale go.work fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestSyncRemovesGoWorkWithoutLocalGoModules(t *testing.T) {
	root := t.TempDir()
	checkout(t, root, "if-a")
	write(t, filepath.Join(root, GoWorkFileName), "// "+generatedHeader+"\n\ngo 1.25\n\nuse ./old\n")
	if code, _, errOut := runMain(t, allPublished("if-a"), "--root", root, "sync"); code != 0 {
		t.Fatal(errOut)
	}
	if _, err := os.Stat(filepath.Join(root, GoWorkFileName)); !os.IsNotExist(err) {
		t.Fatalf("go.work still present: %v", err)
	}
	work, _ := os.ReadFile(filepath.Join(root, model.WorkspaceFileName))
	if !strings.Contains(string(work), "replacements: []") {
		t.Fatalf("work file:\n%s", work)
	}
}

// --all lists every checkout.
// Mutation: ignoring --all fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestSyncAllListsEveryCheckout(t *testing.T) {
	root := t.TempDir()
	checkout(t, root, "if-a")
	checkout(t, root, "if-b")
	if code, _, errOut := runMain(t, allPublished("if-a", "if-b"), "--root", root, "sync", "--all"); code != 0 {
		t.Fatal(errOut)
	}
	work, _ := os.ReadFile(filepath.Join(root, model.WorkspaceFileName))
	if !strings.Contains(string(work), "./if-a") || !strings.Contains(string(work), "./if-b") {
		t.Fatalf("work file:\n%s", work)
	}
}

// A hand-written engmod.work.yml or go.work is not overwritten without --force.
// Mutation: removing checkOwned fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestSyncRefusesHandWrittenWorkFiles(t *testing.T) {
	for _, name := range []string{model.WorkspaceFileName, GoWorkFileName} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			checkout(t, root, "if-a")
			hand := "# mine\nschemaVersion: 2\nreplacements: []\n"
			write(t, filepath.Join(root, name), hand)
			plan, err := BuildPlan(context.Background(), root, allPublished("if-a"), false)
			if err != nil {
				t.Fatal(err)
			}
			if _, err := Sync(plan, "1.25.0", false); !errors.Is(err, ErrHandWritten) {
				t.Fatalf("want ErrHandWritten, got %v", err)
			}
			if data, _ := os.ReadFile(filepath.Join(root, name)); string(data) != hand {
				t.Fatalf("%s overwritten", name)
			}
			if code, _, _ := runMain(t, allPublished("if-a"), "--root", root, "sync"); code != 1 {
				t.Fatalf("CLI sync over a hand-written file: exit %d", code)
			}
			if _, err := Sync(plan, "1.25.0", true); err != nil {
				t.Fatalf("--force: %v", err)
			}
		})
	}
}

// A local checkout at another version than a consumer pins is SKEW; status
// and skew exit 1 and sync still exits 0.
// Mutation: comparing pins with the registry instead of the local version, or
// returning 0 from status on skew, fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestSkewReportsConsumerAliasPinnedLocal(t *testing.T) {
	root := t.TempDir()
	checkout(t, root, "if-a")
	checkout(t, root, "svc-b", dep{"ia", "if-a", "v0.2.1"})
	checkout(t, root, "svc-c", dep{"ia", "if-a", "v0.1.0"})
	reg := allPublished("if-a", "svc-b", "svc-c")
	if code, out, _ := runMain(t, reg, "--root", root, "skew"); code != 0 || out != "" {
		t.Fatalf("no local if-a: want no skew, got %d %q", code, out)
	}
	if code, _, errOut := runMain(t, reg, "--root", root, "pin", "if-a"); code != 0 {
		t.Fatal(errOut)
	}
	code, out, _ := runMain(t, reg, "--root", root, "skew")
	if code != 1 || !strings.HasPrefix(out, "SKEW svc-b ia v0.2.1 v0.1.0 ") || strings.Contains(out, "svc-c") {
		t.Fatalf("skew: %d\n%s", code, out)
	}
	if code, out, _ := runMain(t, reg, "--root", root, "status"); code != 1 || !strings.Contains(out, "SKEW svc-b") {
		t.Fatalf("status: %d\n%s", code, out)
	}
	if code, out, _ := runMain(t, reg, "--root", root, "sync"); code != 0 || !strings.Contains(out, "SKEW svc-b") {
		t.Fatalf("sync: %d\n%s", code, out)
	}
}

// The go directive comes from GOVERSION without the go prefix or suffix.
// Mutation: writing GOVERSION verbatim fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestParseGoVersion(t *testing.T) {
	for in, want := range map[string]string{"go1.27.3": "1.27.3", "go1.26": "1.26", "go1.27rc1": "1.27", "go1.27.3 X:boringcrypto": "1.27.3"} {
		if got, err := parseGoVersion(in); err != nil || got != want {
			t.Fatalf("parseGoVersion(%q) = %q, %v", in, got, err)
		}
	}
	if _, err := parseGoVersion("devel"); err == nil {
		t.Fatal("devel accepted")
	}
}

// Usage errors exit 2 and unknown commands are rejected.
// Mutation: returning 0 or 1 for usage errors fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestMainUsageErrors(t *testing.T) {
	for _, args := range [][]string{{}, {"frob"}, {"pin"}, {"bump", "a"}, {"sync", "--nope"}} {
		if code, _, errOut := runMain(t, fakeRegistry{}, args...); code != 2 || !strings.Contains(errOut, "usage: engws") {
			t.Fatalf("%v: exit %d\n%s", args, code, errOut)
		}
	}
	if code, out, _ := runMain(t, fakeRegistry{}, "--help"); code != 0 || !strings.Contains(out, "bump <repo> <alias> <version>") {
		t.Fatalf("--help: %d %s", code, out)
	}
}

// A consumer's go.mod require of a workspace checkout's Go module at another
// version than the checkout's version tag is SKEW, because go.work makes the
// checkout replace that require; an equal version is not.
// Mutation: comparing only engmod.yml pins (no Go requires), or reporting a
// require that equals the checkout's tag, fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestSkewReportsGoRequireAgainstCheckoutVersion(t *testing.T) {
	root := t.TempDir()
	ifA := checkout(t, root, "if-a")
	write(t, filepath.Join(ifA, "go.mod"), "module example.com/go/if-a\n\ngo 1.25\n")
	svcB := checkout(t, root, "svc-b", dep{"ia", "if-a", "v0.1.0"})
	write(t, filepath.Join(svcB, "go.mod"), "module example.com/go/svc-b\n\ngo 1.25\n\nrequire example.com/go/if-a v0.0.9\n")
	svcC := checkout(t, root, "svc-c", dep{"ia", "if-a", "v0.1.0"})
	write(t, filepath.Join(svcC, "go.mod"), "module example.com/go/svc-c\n\ngo 1.25\n\nrequire example.com/go/if-a v0.1.0\n")
	for _, dir := range []string{ifA, svcB, svcC} {
		runGit(t, dir, "add", "go.mod")
		runGit(t, dir, "commit", "-q", "-m", "go")
		runGit(t, dir, "tag", "-f", "v0.1.0")
	}
	code, out, _ := runMain(t, allPublished("if-a", "svc-b", "svc-c"), "--root", root, "skew")
	if code != 1 || !strings.Contains(out, "SKEW svc-b go:example.com/go/if-a v0.0.9 v0.1.0 ") || strings.Contains(out, "svc-c") {
		t.Fatalf("skew: %d\n%s", code, out)
	}
}

// sync skips, with a warning, a checkout whose module.path is not a valid
// CUE module path (here one carrying a YAML line break) or whose directory
// name is unsafe, so neither reaches engmod.work.yml or go.work.
// Mutation: dropping the module path or directory name check in Scan fails
// this test.
// TRLC-LINKS: REQ-EMG-068
func TestSyncSkipsUnsafeModulePathsAndDirectoryNames(t *testing.T) {
	root := t.TempDir()
	good := filepath.Join(root, "if-a")
	writeModel(t, good, "if-a", "v0.1.0")
	write(t, filepath.Join(good, "go.mod"), "module example.com/go/if-a\n\ngo 1.25\n")
	badPath := filepath.Join(root, "if-b")
	writeModel(t, badPath, "if-b", "v0.1.0")
	manifest, _ := os.ReadFile(filepath.Join(badPath, "engmod.yml"))
	write(t, filepath.Join(badPath, "engmod.yml"), strings.Replace(string(manifest), "  path: "+modulePathOf("if-b")+"\n", "  path: \"x@v0\\n    path: /etc\"\n", 1))
	badDir := filepath.Join(root, "if c")
	writeModel(t, badDir, "if-c", "v0.1.0")
	write(t, filepath.Join(badDir, "go.mod"), "module example.com/go/if-c\n\ngo 1.25\n")
	code, out, errOut := runMain(t, allPublished(), "--root", root, "sync", "--all")
	if code != 0 {
		t.Fatalf("sync: %d\n%s\n%s", code, out, errOut)
	}
	work, _ := os.ReadFile(filepath.Join(root, model.WorkspaceFileName))
	goWork, _ := os.ReadFile(filepath.Join(root, GoWorkFileName))
	if strings.Contains(string(work), "/etc") || strings.Contains(string(work), "if c") || strings.Contains(string(goWork), "if c") || !strings.Contains(string(work), "./if-a") {
		t.Fatalf("unsafe checkout written:\n%s\n%s", work, goWork)
	}
	if _, err := model.LoadWorkspace(filepath.Join(root, model.WorkspaceFileName)); err != nil {
		t.Fatalf("work file invalid: %v", err)
	}
	if !strings.Contains(out+errOut, "warning: if-b: module.path") || !strings.Contains(out+errOut, `warning: skipping "if c"`) {
		t.Fatalf("missing warnings:\n%s\n%s", out, errOut)
	}
}
