// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health_test

// The delta check with the real tree-sitter engine. The external test package
// may import the engine: the layering test only restricts production imports.

import (
	"encoding/json"
	"os"
	"strings"
	"testing"
	"time"

	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/health/engine/treesitter"
)

// goFunc renders a Go function with the given number of if statements, so its
// cyclomatic complexity is ifs+1.
//
// TRLC-LINKS: REQ-EMG-156
func goFunc(name string, ifs int) string {
	return "func " + name + "(x int) int {\n" + strings.Repeat("\tif x > 0 {\n\t\tx++\n\t}\n", ifs) + "\treturn x\n}\n\n"
}

// realOptions checks with the real engine; complex_method warns at 4 and is
// high at 7.
//
// TRLC-LINKS: REQ-EMG-156
func realOptions(r *health.FixtureRepo) health.CheckOptions {
	return health.CheckOptions{
		Repo: r.Repo(), Engine: treesitter.New(),
		ConfigFor: func(string) health.Config {
			return health.Config{Thresholds: map[string]health.Threshold{health.CodeComplexMethod: {Warn: 4, High: 7}}}
		},
	}
}

// complexOnly keeps the complex_method entries of a report.
//
// TRLC-LINKS: REQ-EMG-156
func complexOnly(rep health.DeltaReport) map[string]health.DeltaEntry {
	out := map[string]health.DeltaEntry{}
	for _, e := range rep.Entries {
		if e.Code == health.CodeComplexMethod {
			out[e.Path+" "+e.Function] = e
		}
	}
	return out
}

// Converting a file to CRLF changes no finding: nothing is new or worsened.
// Mutation: an engine or hash that reads \r as code fails this test.
//
// TRLC-LINKS: REQ-EMG-156
func TestRealEngineCRLF(t *testing.T) {
	r := health.NewFixtureRepo(t)
	src := "package p\n\n" + goFunc("Big", 9) + goFunc("Mid", 5) + goFunc("Small", 1)
	r.Write("p.go", src)
	r.Commit("lf")
	r.Write("p.go", strings.ReplaceAll(src, "\n", "\r\n"))

	rep, err := health.Check(realOptions(r))
	if err != nil {
		t.Fatal(err)
	}
	got := complexOnly(rep)
	if len(got) != 2 || got["p.go Big"].Change != health.ChangeExisting || got["p.go Mid"].Change != health.ChangeExisting {
		t.Fatalf("entries = %+v", got)
	}
	if len(health.Regressions(rep)) != 0 {
		t.Errorf("Regressions = %+v", health.Regressions(rep))
	}
}

// A file that does not parse is still analysed as far as the parser recovered
// it: the complex function before the error is reported.
//
// TRLC-LINKS: REQ-EMG-156
func TestRealEngineSyntaxError(t *testing.T) {
	r := health.NewFixtureRepo(t)
	r.Write("keep.go", "package k\n")
	r.Commit("base")
	r.Write("p.go", "package p\n\n"+goFunc("Big", 9)+"func broken( {\n\tif x {\n")

	rep, err := health.Check(realOptions(r))
	if err != nil {
		t.Fatalf("a syntax error must not fail the check: %v", err)
	}
	big, ok := complexOnly(rep)["p.go Big"]
	if !ok || big.Change != health.ChangeNew || big.Severity != health.SeverityHigh || big.ValueAfter != 10 {
		t.Fatalf("entries = %+v", rep.Entries)
	}
	if len(rep.Blocking) != 1 {
		t.Errorf("Blocking = %+v", rep.Blocking)
	}
}

// A head file the parser only partly reads is listed in Skipped when the change
// brought the errors in, since the functions it lost are not measured; a file
// that was as broken at base is not listed again.
// Mutation: not listing it lets a broken file read as clean in the gate.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func TestRealEngineListsNewSyntaxErrors(t *testing.T) {
	r := health.NewFixtureRepo(t)
	r.Write("old.go", "package o\n\nfunc broken( {\n")
	r.Write("fine.go", "package f\n")
	r.Commit("base")
	r.Write("old.go", "package o\n\nfunc broken( {\n\n// edited\n")
	r.Write("fine.go", "package f\n\nfunc lost( {\n")
	r.Write("new.go", "package n\n\nfunc also( {\n")

	rep, err := health.Check(realOptions(r))
	if err != nil {
		t.Fatal(err)
	}
	var paths []string
	for _, s := range rep.Skipped {
		paths = append(paths, s.Path)
		if !strings.Contains(s.Reason, "syntax errors") {
			t.Errorf("reason %q", s.Reason)
		}
	}
	if strings.Join(paths, ",") != "fine.go,new.go" {
		t.Errorf("skipped = %v, want fine.go and new.go (old.go was broken before)", paths)
	}
}

// A renamed file with edits, a function renamed in place and a function moved
// to another file, with the real engine and real Go sources.
// Mutation: not mapping the old path, or not matching by body hash, fails.
//
// TRLC-LINKS: REQ-EMG-156
func TestRealEngineRenameAndMove(t *testing.T) {
	r := health.NewFixtureRepo(t)
	pad := strings.Repeat("// padding line that keeps the rename similarity high\n", 12)
	r.Write("old.go", "package p\n\n"+pad+goFunc("Keep", 6)+goFunc("Edit", 5)+goFunc("Renamed", 8)+goFunc("Mover", 9))
	r.Write("other.go", "package p\n\n"+goFunc("Other", 1))
	r.Commit("base")
	r.Git("mv", "old.go", "new.go")
	r.Write("new.go", "package p\n\n"+pad+goFunc("Keep", 6)+goFunc("Edit", 8)+goFunc("RenamedNow", 8))
	r.Write("other.go", "package p\n\n"+goFunc("Other", 1)+goFunc("Mover", 9))

	rep, err := health.Check(realOptions(r))
	if err != nil {
		t.Fatal(err)
	}
	got := complexOnly(rep)
	want := map[string]string{
		"new.go Keep":       health.ChangeExisting,
		"new.go Edit":       health.ChangeWorsened,
		"new.go RenamedNow": health.ChangeExisting,
		"other.go Mover":    health.ChangeExisting,
	}
	for key, change := range want {
		if got[key].Change != change {
			t.Errorf("%s = %q, want %q (all: %+v)", key, got[key].Change, change, got)
		}
	}
	if edit := got["new.go Edit"]; edit.ValueBefore != 6 || edit.ValueAfter != 9 {
		t.Errorf("Edit values = %d -> %d", edit.ValueBefore, edit.ValueAfter)
	}
}

// With the cache in front of the real engine two checks give the same JSON
// and the second is served from the cache.
//
// TRLC-LINKS: REQ-EMG-159
func TestRealEngineWithCache(t *testing.T) {
	r := health.NewFixtureRepo(t)
	r.Write("p.go", "package p\n\n"+goFunc("Big", 5))
	r.Commit("base")
	r.Write("p.go", "package p\n\n"+goFunc("Big", 9)+goFunc("New", 7))
	cache := health.NewCache(r.Dir)
	opts := realOptions(r)
	opts.Engine = health.CachedEngine{Engine: treesitter.New(), Cache: cache}

	first, err := health.Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	if hits, misses := cache.Stats(); hits != 0 || misses != 2 {
		t.Errorf("first run Stats = %d, %d; want 0 hits and 2 misses", hits, misses)
	}
	second, err := health.Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	if hits, misses := cache.Stats(); hits != 2 || misses != 2 {
		t.Errorf("second run Stats = %d, %d; want 2 hits and 2 misses in total", hits, misses)
	}
	a, _ := json.Marshal(first)
	b, _ := json.Marshal(second)
	if string(a) != string(b) {
		t.Errorf("cached run differs:\n%s\n%s", a, b)
	}
	plain := realOptions(r)
	third, err := health.Check(plain)
	if c, _ := json.Marshal(third); err != nil || string(c) != string(a) {
		t.Errorf("a run without the cache differs: %v", err)
	}
}

// The warm check of a single changed file with the real engine and the cache
// has a target of 300 ms (ADR-EMG-054). It is a measurement, not a gate, and
// runs only with ENGMODEL_HEALTH_PERF=1.
//
// TRLC-LINKS: REQ-EMG-159
func TestWarmSingleFileCheckTime(t *testing.T) {
	if os.Getenv("ENGMODEL_HEALTH_PERF") != "1" {
		t.Skip("set ENGMODEL_HEALTH_PERF=1 to measure the warm single-file check")
	}
	opts := perfFixture(t)
	cold := time.Now()
	if _, err := health.Check(opts); err != nil { // cold: fills the cache
		t.Fatal(err)
	}
	t.Logf("cold single-file check: %s", time.Since(cold))
	best := time.Hour
	for i := 0; i < 5; i++ {
		start := time.Now()
		if _, err := health.Check(opts); err != nil {
			t.Fatal(err)
		}
		if d := time.Since(start); d < best {
			best = d
		}
	}
	t.Logf("warm single-file check of %s: best of 5 = %s (target < 300ms)", opts.Paths[0], best)
}

// perfFixture commits a real 1000-line Go source of this module, edits it and
// returns options for a warm check of that one file.
//
// TRLC-LINKS: REQ-EMG-159
func perfFixture(t *testing.T) health.CheckOptions {
	t.Helper()
	src, err := os.ReadFile("../model/semantic.go")
	if err != nil {
		t.Skip("source for the measurement not found:", err)
	}
	r := health.NewFixtureRepo(t)
	r.Write("semantic.go", string(src))
	r.Commit("base")
	r.Write("semantic.go", string(src)+"\n"+goFunc("Added", 12))
	opts := health.CheckOptions{
		Repo:   r.Repo(),
		Engine: health.CachedEngine{Engine: treesitter.New(), Cache: health.NewCache(r.Dir)},
		Paths:  []string{"semantic.go"},
	}
	t.Logf("fixture: %d lines", strings.Count(string(src), "\n"))
	return opts
}
