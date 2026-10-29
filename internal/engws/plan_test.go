// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/mod/modregistrytest"
	"github.com/labeth/engineering-model-go/model"
)

// A clean checkout at its tag whose version is published resolves from OCI.
// Mutation: making every checkout local (Local = true) fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestPlanCleanTaggedPublishedResolvesFromOCI(t *testing.T) {
	root := t.TempDir()
	checkout(t, root, "if-a")
	plan, err := BuildPlan(context.Background(), root, allPublished("if-a"), false)
	if err != nil {
		t.Fatal(err)
	}
	e := entry(t, plan, "if-a")
	if e.Local || e.State != StatePublished || len(e.Reasons) != 0 {
		t.Fatalf("want oci/published, got local=%v state=%s reasons=%v", e.Local, e.State, e.Reasons)
	}
}

// A dirty working tree (including an untracked file) makes the checkout local.
// Mutation: dropping the git status --porcelain check fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestPlanDirtyCheckoutIsLocal(t *testing.T) {
	root := t.TempDir()
	dir := checkout(t, root, "if-a")
	write(t, filepath.Join(dir, "model", "notes.yml"), "x: 1\n")
	plan, err := BuildPlan(context.Background(), root, allPublished("if-a"), false)
	if err != nil {
		t.Fatal(err)
	}
	e := entry(t, plan, "if-a")
	if !e.Local || e.State != StateDirty {
		t.Fatalf("want local/dirty, got local=%v state=%s reasons=%v", e.Local, e.State, e.Reasons)
	}
}

// A commit after the release tag makes the checkout local even when clean.
// Mutation: dropping the HEAD-at-tag check fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestPlanUntaggedCommitIsLocal(t *testing.T) {
	root := t.TempDir()
	dir := checkout(t, root, "if-a")
	write(t, filepath.Join(dir, "README.md"), "change\n")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "wip")
	plan, err := BuildPlan(context.Background(), root, allPublished("if-a"), false)
	if err != nil {
		t.Fatal(err)
	}
	e := entry(t, plan, "if-a")
	if !e.Local || e.State != StateUntagged || !strings.Contains(e.Reasons[0], "v0.1.0") {
		t.Fatalf("want local/untagged, got local=%v state=%s reasons=%v", e.Local, e.State, e.Reasons)
	}
}

// The tag is v<version> whether or not module.version has a leading v.
// Mutation: comparing the tag with the raw manifest version fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestTagForAcceptsVersionWithOrWithoutV(t *testing.T) {
	for _, in := range []string{"v0.1.0", "0.1.0", " v0.1.0 "} {
		if got := tagFor(in); got != "v0.1.0" {
			t.Fatalf("tagFor(%q) = %q", in, got)
		}
	}
	root := t.TempDir()
	dir := checkout(t, root, "if-a")
	data, _ := os.ReadFile(filepath.Join(dir, "engmod.yml"))
	write(t, filepath.Join(dir, "engmod.yml"), strings.Replace(string(data), "version: v0.1.0\n  modelId", "version: 0.1.0\n  modelId", 1))
	runGit(t, dir, "commit", "-q", "-am", "unprefixed")
	runGit(t, dir, "tag", "-f", "v0.1.0")
	plan, err := BuildPlan(context.Background(), root, allPublished("if-a"), false)
	if err != nil {
		t.Fatal(err)
	}
	if e := entry(t, plan, "if-a"); e.Local {
		t.Fatalf("0.1.0 at tag v0.1.0 should be oci, got %v", e.Reasons)
	}
}

// A tagged clean checkout whose version is not in the registry is local; the
// registry is a real in-process CUE OCI registry reached through CUE_REGISTRY.
// Mutation: ignoring PublicationMissing in classify fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestPlanUnpublishedVersionIsLocalAgainstOCIRegistry(t *testing.T) {
	work := t.TempDir()
	registryRoot := filepath.Join(work, "registry")
	writeModel(t, filepath.Join(registryRoot, strings.ReplaceAll(testFamily, "/", "_")+"_if-a_v0.1.0"), "if-a", "v0.1.0")
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	t.Setenv("CUE_REGISTRY", testFamily+"="+registry.Host()+"+insecure")

	root := filepath.Join(work, "ws")
	checkout(t, root, "if-a") // v0.1.0 published
	checkout(t, root, "if-b") // never published
	plan, err := BuildPlan(context.Background(), root, NewCUERegistry(), false)
	if err != nil {
		t.Fatal(err)
	}
	if e := entry(t, plan, "if-a"); e.Local || e.State != StatePublished {
		t.Fatalf("if-a: want published/oci, got %s %v %s", e.State, e.Reasons, e.Note)
	}
	if e := entry(t, plan, "if-b"); !e.Local || e.State != StateUnpublished {
		t.Fatalf("if-b: want unpublished/local, got %s %v %s", e.State, e.Reasons, e.Note)
	}
}

// CUE_REGISTRY routing follows CUE: the longest prefix wins, "none" and an
// unset variable mean unknown (never changed), and a bare host is the default.
// Mutation: treating PublicationUnknown as missing fails the none/unset cases.
// TRLC-LINKS: REQ-EMG-068
func TestCUERegistryRoutingAndOfflineIsUnknown(t *testing.T) {
	registryRoot := t.TempDir()
	writeModel(t, filepath.Join(registryRoot, strings.ReplaceAll(testFamily, "/", "_")+"_if-a_v0.1.0"), "if-a", "v0.1.0")
	registry, err := modregistrytest.New(os.DirFS(registryRoot), "")
	if err != nil {
		t.Fatal(err)
	}
	defer registry.Close()
	ctx := context.Background()
	cases := []struct {
		env  string
		want Publication
	}{
		{testFamily + "=" + registry.Host() + "+insecure", PublicationPublished},
		{"example.com=none," + testFamily + "=" + registry.Host() + "+insecure", PublicationPublished},
		{registry.Host() + "+insecure," + testFamily + "=none", PublicationUnknown},
		{registry.Host() + "+insecure", PublicationPublished},
		{"none", PublicationUnknown},
		{"", PublicationUnknown},
	}
	for _, c := range cases {
		t.Setenv("CUE_REGISTRY", c.env)
		got, why := NewCUERegistry().Published(ctx, modulePathOf("if-a"), "v0.1.0")
		if got != c.want {
			t.Fatalf("CUE_REGISTRY=%q: got %v (%s), want %v", c.env, got, why, c.want)
		}
		if got == PublicationUnknown && why == "" {
			t.Fatalf("CUE_REGISTRY=%q: unknown without a reason", c.env)
		}
	}
	t.Setenv("CUE_REGISTRY", registry.Host()+"+insecure")
	if got, _ := NewCUERegistry().Published(ctx, modulePathOf("if-a"), "v0.9.0"); got != PublicationMissing {
		t.Fatalf("unpublished version: got %v", got)
	}

	root := t.TempDir()
	checkout(t, root, "if-a")
	plan, err := BuildPlan(ctx, root, fakeRegistry{unknown: true}, false)
	if err != nil {
		t.Fatal(err)
	}
	if e := entry(t, plan, "if-a"); e.Local || e.State != StateUnknown || len(plan.Notes) == 0 {
		t.Fatalf("offline: want oci/unknown with a note, got %s local=%v notes=%v", e.State, e.Local, plan.Notes)
	}
}

// A directory that is not its own git repository is local with the reason,
// also when it sits inside a parent repository.
// Mutation: accepting a parent repository's work tree fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestPlanNonGitCheckoutIsLocal(t *testing.T) {
	root := t.TempDir()
	runGit(t, root, "init", "-q")
	writeModel(t, filepath.Join(root, "if-a"), "if-a", "v0.1.0")
	plan, err := BuildPlan(context.Background(), root, allPublished("if-a"), false)
	if err != nil {
		t.Fatal(err)
	}
	e := entry(t, plan, "if-a")
	if !e.Local || e.State != StateNoGit || !strings.Contains(e.Reasons[0], "own git repository") {
		t.Fatalf("want local/nogit, got local=%v state=%s reasons=%v", e.Local, e.State, e.Reasons)
	}
}

// pin forces a clean published checkout local and unpin returns it to OCI.
// Mutation: ignoring the pins in classify fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestPinAndUnpinForceLocal(t *testing.T) {
	root := t.TempDir()
	dir := checkout(t, root, "if-a")
	reg := allPublished("if-a")
	if code, out, errOut := runMain(t, reg, "--root", root, "pin", "if-a"); code != 0 {
		t.Fatalf("pin: %d %s %s", code, out, errOut)
	}
	plan, _ := BuildPlan(context.Background(), root, reg, false)
	if e := entry(t, plan, "if-a"); !e.Local || e.State != StatePinned {
		t.Fatalf("pinned: got %s local=%v", e.State, e.Local)
	}
	work, _ := os.ReadFile(filepath.Join(root, "engmod.work.yml"))
	if !strings.Contains(string(work), "path: ./if-a") {
		t.Fatalf("pin did not sync:\n%s", work)
	}
	if code, _, errOut := runMain(t, reg, "--root", root, "unpin", "if-a"); code != 0 {
		t.Fatal(errOut)
	}
	plan, _ = BuildPlan(context.Background(), root, reg, false)
	if e := entry(t, plan, "if-a"); e.Local {
		t.Fatalf("unpinned: still local")
	}
	if code, _, _ := runMain(t, reg, "--root", root, "pin", "nope"); code == 0 {
		t.Fatal("pin of a missing checkout succeeded")
	}
	assertClean(t, dir)
}

// The root is the nearest ancestor holding engmod.work.yml or .engws.yml.
// Mutation: returning start unconditionally fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestFindRootSearchesAncestors(t *testing.T) {
	root := t.TempDir()
	deep := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if got, _ := FindRoot(deep); got != deep {
		t.Fatalf("no marker: got %s want %s", got, deep)
	}
	write(t, filepath.Join(root, ".engws.yml"), "pins: []\n")
	if got, _ := FindRoot(deep); got != root {
		t.Fatalf("marker: got %s want %s", got, root)
	}
}

// The ancestor search never returns $HOME or an ancestor of it: a stray
// .engws.yml or engmod.work.yml there does not make it the root. The search
// below $HOME still finds a marker.
// Mutation: searching up to / without stopping at $HOME fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestFindRootStopsAtHome(t *testing.T) {
	top := t.TempDir()
	home := filepath.Join(top, "home")
	deep := filepath.Join(home, "src", "ws", "repo")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", home)
	write(t, filepath.Join(home, ".engws.yml"), "pins: []\n")
	write(t, filepath.Join(top, model.WorkspaceFileName), "schemaVersion: 2\nreplacements: []\n")
	if got, err := FindRoot(deep); err != nil || got != deep {
		t.Fatalf("marker in $HOME and above it: got %s, %v; want %s", got, err, deep)
	}
	write(t, filepath.Join(home, "src", "ws", ".engws.yml"), "pins: []\n")
	if got, err := FindRoot(deep); err != nil || got != filepath.Join(home, "src", "ws") {
		t.Fatalf("marker below $HOME: got %s, %v", got, err)
	}
	// $HOME given as a symlink to the real home: the search from the real
	// path still stops at the home directory.
	if err := os.Remove(filepath.Join(home, "src", "ws", ".engws.yml")); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(top, "home-link")
	if err := os.Symlink(home, link); err != nil {
		t.Fatal(err)
	}
	t.Setenv("HOME", link)
	if got, err := FindRoot(deep); err != nil || got != deep {
		t.Fatalf("symlinked $HOME: got %s, %v; want %s", got, err, deep)
	}
}

// A world-writable root found by the search is refused; --root accepts it.
// Mutation: dropping the world-writable check fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestWorldWritableRootNeedsExplicitRoot(t *testing.T) {
	t.Setenv("HOME", filepath.Join(t.TempDir(), "home"))
	shared := filepath.Join(t.TempDir(), "shared")
	deep := filepath.Join(shared, "repo")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	write(t, filepath.Join(shared, ".engws.yml"), "pins: []\n")
	if err := os.Chmod(shared, 0o777); err != nil {
		t.Fatal(err)
	}
	if got, err := FindRoot(deep); err == nil || !strings.Contains(err.Error(), "world-writable") {
		t.Fatalf("world-writable root must be refused: %s, %v", got, err)
	}
	if code, out, errOut := runMain(t, allPublished(), "--root", shared, "status"); code != 0 || !strings.HasPrefix(out, "workspace "+shared) {
		t.Fatalf("explicit --root accepts it: %d\n%s\n%s", code, out, errOut)
	}
}

// Every command names the chosen root on its first output line (skew on
// stderr, so its stdout stays the SKEW lines).
// Mutation: not printing the root for bump or skew fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestEveryCommandPrintsRootFirst(t *testing.T) {
	root := t.TempDir()
	checkout(t, root, "if-a")
	reg := allPublished("if-a")
	for _, args := range [][]string{{"status"}, {"sync"}, {"skew"}, {"check", "--in-process"}, {"pin", "if-a"}, {"unpin", "if-a"}, {"bump", "if-a", "nope", "v0.2.0"}} {
		_, out, errOut := runMain(t, reg, append([]string{"--root", root}, args...)...)
		first := out
		if args[0] == "skew" {
			first = errOut
		}
		line, _, _ := strings.Cut(first, "\n")
		if !strings.Contains(line, root) {
			t.Fatalf("%v: first line %q does not name the root\nstdout:\n%s\nstderr:\n%s", args, line, out, errOut)
		}
	}
}
