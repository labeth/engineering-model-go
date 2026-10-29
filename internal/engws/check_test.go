// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// planOf builds a plan from (name, local, dependency names) without disk.
func planOf(specs ...struct {
	name  string
	local bool
	deps  []string
}) Plan {
	var plan Plan
	for _, s := range specs {
		e := Entry{Checkout: Checkout{Name: s.name}, Local: s.local}
		e.Manifest.Module.Path = modulePathOf(s.name)
		e.Manifest.Module.Version = "v0.1.0"
		for _, d := range s.deps {
			e.Manifest.Dependencies = append(e.Manifest.Dependencies, model.ManifestDependency{Alias: d, Path: modulePathOf(d), Version: "v0.1.0"})
		}
		plan.Entries = append(plan.Entries, e)
	}
	return plan
}

// check selects the local checkouts, their transitive reverse dependencies
// and the top systems, and nothing else.
// Mutation: dropping the transitive walk (direct consumers only), or omitting
// the top systems, fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestSelectCheckTargetsReverseDependenciesAndTop(t *testing.T) {
	type spec = struct {
		name  string
		local bool
		deps  []string
	}
	plan := planOf(
		spec{"if-1", true, nil},
		spec{"if-2", false, nil},
		spec{"svc-1", false, []string{"if-1"}},
		spec{"svc-2", false, []string{"if-2"}},
		spec{"sys-1", false, []string{"svc-1"}},
		spec{"sys-2", false, []string{"svc-2"}},
		spec{"top", false, []string{"sys-1", "sys-2"}},
		spec{"lone", false, []string{"external"}},
	)
	got := map[string]string{}
	for _, target := range SelectCheckTargets(plan) {
		got[target.Entry.Name] = target.Reason
	}
	want := map[string]string{
		"if-1":  "local",
		"svc-1": "depends on if-1",
		"sys-1": "depends on svc-1",
		"top":   "depends on sys-1",
		"lone":  "top system",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v, want %v", got, want)
	}
	for name, reason := range want {
		if got[name] != reason {
			t.Fatalf("%s: got %q want %q (all %v)", name, got[name], reason, got)
		}
	}

	none := planOf(spec{"if-2", false, nil}, spec{"svc-2", false, []string{"if-2"}})
	targets := SelectCheckTargets(none)
	if len(targets) != 1 || targets[0].Entry.Name != "svc-2" || targets[0].Reason != "top system" {
		t.Fatalf("no local set: want only the top system, got %+v", targets)
	}
}

// check runs the in-process library check in each selected checkout, prints
// one ok|FAIL line per repo with counts and errors, and exits 1 on a failure.
// Mutation: reporting ok regardless of validation errors fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestCheckRunsLibraryCheckAndFailsOnErrors(t *testing.T) {
	root := t.TempDir()
	good := checkout(t, root, "if-a")
	bad := checkout(t, root, "if-b")
	reg := allPublished("if-a", "if-b")
	code, out, errOut := runMain(t, reg, "--root", root, "check")
	if code != 0 || !strings.Contains(out, "ok   if-a (0 errors") || !strings.Contains(out, "ok   if-b") {
		t.Fatalf("clean check: %d\n%s\n%s", code, out, errOut)
	}
	// A dangling trace link in if-b makes it local (dirty) and failing.
	write(t, filepath.Join(bad, "src", "x.go"), "// ENGMODEL-OWNER-UNIT: FU-X\npackage x\n\n// TRLC-LINKS: REQ-NOPE-404\nfunc X() {}\n")
	code, out, _ = runMain(t, reg, "--root", root, "check")
	if code != 1 || !strings.Contains(out, "FAIL if-b (") || !strings.Contains(out, "[error]") {
		t.Fatalf("broken check: %d\n%s", code, out)
	}
	if !strings.Contains(out, "ok   if-a (0 errors, 2 warnings) [library; top system]") {
		t.Fatalf("if-a is a top system and must still pass:\n%s", out)
	}
	code, out, _ = runMain(t, reg, "--root", root, "check", "--summary")
	if code != 1 || strings.Contains(out, "[error]") {
		t.Fatalf("--summary prints only lines: %d\n%s", code, out)
	}
	assertClean(t, good)
}

// A repo's own check is used only when it supports --workspace-root, and
// --summary is passed only when the check declares it.
// Mutation: exec-ing every .github/engmod-check, or always passing
// --summary, fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestPinnedCheckDetection(t *testing.T) {
	dir := t.TempDir()
	if pinnedCheckSupportsWorkspace(dir) {
		t.Fatal("no check dir")
	}
	write(t, filepath.Join(dir, ".github", "engmod-check", "go.mod"), "module example.com/engmod-check\n")
	write(t, filepath.Join(dir, ".github", "engmod-check", "main.go"), "package main\nfunc main() { _ = \"summary\" }\n")
	if pinnedCheckSupportsWorkspace(dir) {
		t.Fatal("check without --workspace-root selected")
	}
	write(t, filepath.Join(dir, ".github", "engmod-check", "main.go"), "package main\nfunc main() { _ = \"workspace-root\" }\n")
	if got := detectPinnedCheck(dir); !got.present || got.summary {
		t.Fatalf("check with --workspace-root only: %+v", got)
	}
	write(t, filepath.Join(dir, ".github", "engmod-check", "main.go"), "package main\nfunc main() { _, _ = \"workspace-root\", \"summary\" }\n")
	if got := detectPinnedCheck(dir); !got.present || !got.summary {
		t.Fatalf("check with both flags: %+v", got)
	}
}

// check warns when engmod.work.yml does not match the local set.
// Mutation: making WorkFileDrift always return "" fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestWorkFileDriftWarnsWhenNotSynced(t *testing.T) {
	root := t.TempDir()
	dir := checkout(t, root, "if-a")
	reg := allPublished("if-a")
	plan, _ := BuildPlan(context.Background(), root, reg, false)
	if d := WorkFileDrift(plan); d != "" {
		t.Fatalf("nothing local, no file: %q", d)
	}
	write(t, filepath.Join(dir, "x.txt"), "dirty\n")
	plan, _ = BuildPlan(context.Background(), root, reg, false)
	if d := WorkFileDrift(plan); !strings.Contains(d, "missing") {
		t.Fatalf("want missing, got %q", d)
	}
	if _, err := Sync(plan, "1.25.0", false); err != nil {
		t.Fatal(err)
	}
	if d := WorkFileDrift(plan); d != "" {
		t.Fatalf("after sync: %q", d)
	}
	os.Remove(filepath.Join(dir, "x.txt"))
	plan, _ = BuildPlan(context.Background(), root, reg, false)
	if d := WorkFileDrift(plan); !strings.Contains(d, "extra ") {
		t.Fatalf("want extra, got %q", d)
	}
}

// check leaves every checkout unchanged: a lock the engine writes from local
// replacements is removed, and an existing lock is restored byte for byte.
// Mutation: dropping guardLock around libraryCheck fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestCheckWritesNothingInsideCheckouts(t *testing.T) {
	root := t.TempDir()
	ifA := checkout(t, root, "if-a")
	svc := checkout(t, root, "svc-b", dep{"ia", "if-a", "v0.1.0"})
	sys := checkout(t, root, "sys-c", dep{"sb", "svc-b", "v0.1.0"})
	lock := "schemaVersion: 1\ndependencies: []\n# committed\n"
	write(t, filepath.Join(sys, "engmod.lock.yml"), lock)
	runGit(t, sys, "add", "engmod.lock.yml")
	runGit(t, sys, "commit", "-q", "-m", "lock")
	runGit(t, sys, "tag", "-f", "v0.1.0")
	reg := allPublished("if-a", "svc-b", "sys-c")
	// Pin everything so composition resolves from the checkouts only.
	for _, name := range []string{"if-a", "svc-b", "sys-c"} {
		if code, _, errOut := runMain(t, reg, "--root", root, "pin", name); code != 0 {
			t.Fatal(errOut)
		}
	}
	code, out, errOut := runMain(t, reg, "--root", root, "check")
	if code != 0 || !strings.Contains(out, "ok   svc-b") || !strings.Contains(out, "ok   sys-c") {
		t.Fatalf("check: %d\n%s\n%s", code, out, errOut)
	}
	for _, dir := range []string{ifA, svc, sys} {
		assertClean(t, dir)
	}
}

// The repo's own check runs with GOWORK=<repo>/.github/engmod-check/go.work
// when that file exists (a deliberately local engine) and GOWORK=off
// otherwise; the root go.work is never inherited.
// Mutation: hard-coding GOWORK=off, or inheriting the root go.work, fails
// this test.
// TRLC-LINKS: REQ-EMG-068
func TestExecCheckSelectsToolGoWork(t *testing.T) {
	root := t.TempDir()
	dir := checkout(t, root, "if-a")
	checkDir := filepath.Join(dir, ".github", "engmod-check")
	write(t, filepath.Join(checkDir, "go.mod"), "module example.com/engmod-check\n\ngo 1.22\n")
	write(t, filepath.Join(checkDir, "main.go"), `package main

import (
	"fmt"
	"os"
)

// Accepts "workspace-root"; reports the GOWORK it was built under.
func main() {
	fmt.Println("tool.gowork [error] GOWORK=" + os.Getenv("GOWORK"))
	os.Exit(1)
}
`)
	write(t, filepath.Join(root, GoWorkFileName), "go 1.22\n\nuse ./nothing-here\n")
	target := CheckTarget{Entry: Entry{Checkout: Checkout{Name: "if-a", Dir: dir}}, Reason: "local"}
	gowork := func() string {
		t.Helper()
		result := execCheck(context.Background(), root, target)
		for _, line := range result.Lines {
			if _, value, ok := strings.Cut(line, "GOWORK="); ok {
				return value
			}
		}
		t.Fatalf("tool did not run: %+v", result)
		return ""
	}
	if got := gowork(); got != "off" {
		t.Fatalf("without the tool go.work: GOWORK=%q, want off", got)
	}
	write(t, filepath.Join(checkDir, GoWorkFileName), "go 1.22\n\nuse .\n")
	if got, want := gowork(), filepath.Join(checkDir, GoWorkFileName); got != want {
		t.Fatalf("with the tool go.work: GOWORK=%q, want %q", got, want)
	}
}

// check --all checks every checkout on purpose, so it does not warn that
// engmod.work.yml lacks the checkouts that are not local; a plain check still
// warns when the work file is out of date.
// Mutation: computing the drift warning from the --all plan fails this test.
// TRLC-LINKS: REQ-EMG-068
func TestCheckAllDoesNotWarnAboutWorkFileDrift(t *testing.T) {
	root := t.TempDir()
	checkout(t, root, "if-a")
	dirty := checkout(t, root, "if-b")
	reg := allPublished("if-a", "if-b")
	if code, _, errOut := runMain(t, reg, "--root", root, "sync"); code != 0 {
		t.Fatal(errOut)
	}
	if _, _, errOut := runMain(t, reg, "--root", root, "check", "--all", "--in-process"); strings.Contains(errOut, "out of date") {
		t.Fatalf("check --all warned about drift:\n%s", errOut)
	}
	write(t, filepath.Join(dirty, "x.txt"), "dirty\n")
	if _, _, errOut := runMain(t, reg, "--root", root, "check", "--in-process"); !strings.Contains(errOut, "out of date") {
		t.Fatalf("plain check must warn about drift:\n%s", errOut)
	}
}

// check --refresh runs every check with ENGMOD_MODULE_CACHE=refresh, so the
// dependencies are re-fetched instead of read from the module cache.
// Mutation: ignoring --refresh (no ENGMOD_MODULE_CACHE in the tool's
// environment) fails this test.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-064
func TestCheckRefreshSetsModuleCacheRefresh(t *testing.T) {
	t.Setenv("ENGMOD_MODULE_CACHE", "")
	root := t.TempDir()
	dir := checkout(t, root, "if-a")
	checkDir := filepath.Join(dir, ".github", "engmod-check")
	write(t, filepath.Join(checkDir, "go.mod"), "module example.com/engmod-check\n\ngo 1.22\n")
	write(t, filepath.Join(checkDir, "main.go"), `package main

import (
	"fmt"
	"os"
)

// Accepts "workspace-root"; reports the module cache mode.
func main() {
	fmt.Println("tool.cache [error] CACHE=" + os.Getenv("ENGMOD_MODULE_CACHE") + ".")
	os.Exit(1)
}
`)
	reg := allPublished("if-a")
	for args, want := range map[string]string{"check": "CACHE=.", "check --refresh": "CACHE=refresh."} {
		_, out, _ := runMain(t, reg, append([]string{"--root", root}, strings.Fields(args)...)...)
		if !strings.Contains(out, want) {
			t.Fatalf("%s: want %s in\n%s", args, want, out)
		}
	}
}
