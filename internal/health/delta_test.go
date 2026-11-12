// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"strings"
	"testing"
)

// fakeEngine analyses a toy format so that the delta rules can be tested with
// exact metrics: a function starts at a line "fn NAME" and runs to the next
// such line; its cyclomatic complexity is 1 plus the lines that start with
// "if". The complex_method limits come from the Config (default 10 and 20).
type fakeEngine struct{}

// TRLC-LINKS: REQ-EMG-156
func (fakeEngine) Name() string { return "fake" }

// TRLC-LINKS: REQ-EMG-156
func (fakeEngine) Version() string { return "1" }

// TRLC-LINKS: REQ-EMG-156
func (fakeEngine) Analyze(inputs []FileInput, cfg Config) ([]FileResult, error) {
	limit := DefaultThresholds()[CodeComplexMethod]
	if t, ok := cfg.Thresholds[CodeComplexMethod]; ok {
		limit = t
	}
	var out []FileResult
	for _, in := range inputs {
		lines := strings.Split(strings.ReplaceAll(string(in.Source), "\r\n", "\n"), "\n")
		if lines[len(lines)-1] == "" {
			lines = lines[:len(lines)-1]
		}
		res := FileResult{Path: in.Path, Language: in.Language, LOC: len(lines), Functions: []Function{}}
		for i := 0; i < len(lines); i++ {
			name, ok := strings.CutPrefix(lines[i], "fn ")
			if !ok {
				continue
			}
			end, cyc := i+1, 1
			for end < len(lines) && !strings.HasPrefix(lines[end], "fn ") {
				if strings.HasPrefix(lines[end], "if") {
					cyc++
				}
				end++
			}
			f := Function{Name: name, Kind: "function", StartLine: i + 1, EndLine: end, Cyclomatic: cyc}
			if cyc >= limit.Warn {
				sev := SeverityWarn
				if cyc >= limit.High {
					sev = SeverityHigh
				}
				f.Findings = []Finding{{
					Code: CodeComplexMethod, Severity: sev, StartLine: f.StartLine, EndLine: f.EndLine,
					Metric: "cyclomatic", Value: cyc, Message: fmt.Sprintf("%s has complexity %d", name, cyc),
				}}
			}
			res.Functions = append(res.Functions, f)
		}
		out = append(out, res)
	}
	return out, nil
}

// toyFunc renders a function of the toy format with the given cyclomatic complexity.
//
// TRLC-LINKS: REQ-EMG-156
func toyFunc(name string, cyc int) string {
	return "fn " + name + "\n" + strings.Repeat("if x\n", cyc-1) + "ret\n"
}

// scoreByFindings is a stand-in for the real weights table.
//
// TRLC-LINKS: REQ-EMG-156
func scoreByFindings(r FileResult) FileScore {
	n := 0
	for _, f := range r.Functions {
		n += len(f.Findings)
	}
	score := float64(10 - n)
	band := BandGreen
	if score < 8 {
		band = BandYellow
	}
	return FileScore{Score: score, Band: band}
}

// deltaOptions are the options of the tests: warn at 4 and high at 7.
//
// TRLC-LINKS: REQ-EMG-156
func deltaOptions(r *FixtureRepo) CheckOptions {
	return CheckOptions{
		Repo: r.Repo(), Engine: fakeEngine{}, Scorer: scoreByFindings,
		ConfigFor: func(string) Config {
			return Config{Thresholds: map[string]Threshold{CodeComplexMethod: {Warn: 4, High: 7}}}
		},
	}
}

// byFunction indexes the entries of a path by function name.
//
// TRLC-LINKS: REQ-EMG-156
func byFunction(t *testing.T, rep DeltaReport, path string) map[string]DeltaEntry {
	t.Helper()
	out := map[string]DeltaEntry{}
	for _, e := range rep.Entries {
		if e.Path != path {
			continue
		}
		if _, dup := out[e.Function]; dup {
			t.Fatalf("two entries for %s %s", path, e.Function)
		}
		out[e.Function] = e
	}
	return out
}

// wantEntry checks the change class, severity and values of one entry.
//
// TRLC-LINKS: REQ-EMG-156
func wantEntry(t *testing.T, got map[string]DeltaEntry, name, change string, sev Severity, before, after int) {
	t.Helper()
	e, ok := got[name]
	if !ok {
		t.Errorf("%s: no entry", name)
		return
	}
	if e.Change != change || e.Severity != sev || e.ValueBefore != before || e.ValueAfter != after {
		t.Errorf("%s = %s/%s %d->%d, want %s/%s %d->%d", name, e.Change, e.Severity, e.ValueBefore, e.ValueAfter, change, sev, before, after)
	}
}

// The worked example: one function got worse, one better, one is unchanged,
// one is new, one grew by a quarter inside warn, and one was deleted. Only
// the high new and worsened entries block.
// Mutation: classing every entry new fails the existing cases; blocking on
// warn fails the blocking list.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckClassifiesModifiedFile(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("worse", 5)+toyFunc("better", 8)+toyFunc("same", 5)+toyFunc("grow", 4)+toyFunc("gone", 9)+toyFunc("plain", 2))
	baseSha := r.Commit("base")
	r.Write("a.go", toyFunc("worse", 8)+toyFunc("better", 5)+toyFunc("same", 5)+toyFunc("grow", 5)+toyFunc("fresh", 10)+toyFunc("freshWarn", 5)+toyFunc("plain", 3))

	rep, err := Check(deltaOptions(r))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Schema != DeltaSchema || rep.Base != baseSha || rep.Head != WorkingTree {
		t.Errorf("header = %q %q %q", rep.Schema, rep.Base, rep.Head)
	}
	got := byFunction(t, rep, "a.go")
	wantEntry(t, got, "worse", ChangeWorsened, SeverityHigh, 5, 8)
	wantEntry(t, got, "better", ChangeExisting, SeverityWarn, 8, 5)
	wantEntry(t, got, "same", ChangeExisting, SeverityWarn, 5, 5)
	wantEntry(t, got, "grow", ChangeWorsened, SeverityWarn, 4, 5)
	wantEntry(t, got, "fresh", ChangeNew, SeverityHigh, 0, 10)
	wantEntry(t, got, "freshWarn", ChangeNew, SeverityWarn, 0, 5)
	if len(got) != 6 {
		t.Errorf("entries = %v; deleted and clean functions must produce none", got)
	}
	if got["worse"].SeverityBefore != SeverityWarn || got["fresh"].SeverityBefore != "" {
		t.Errorf("severityBefore: worse %q, fresh %q", got["worse"].SeverityBefore, got["fresh"].SeverityBefore)
	}
	blocking := Regressions(rep)
	if len(blocking) != 2 || blocking[0].Function != "fresh" || blocking[1].Function != "worse" {
		t.Errorf("Regressions = %+v, want fresh and worse", blocking)
	}
	if fmt.Sprint(blocking) != fmt.Sprint(rep.Blocking) {
		t.Errorf("Blocking %+v differs from Regressions %+v", rep.Blocking, blocking)
	}
	if len(rep.Files) != 1 || rep.Files[0].Path != "a.go" || rep.Files[0].ScoreBefore == nil || rep.Files[0].ScoreAfter == nil {
		t.Fatalf("Files = %+v", rep.Files)
	}
	if *rep.Files[0].ScoreBefore != 5 || *rep.Files[0].ScoreAfter != 4 || rep.Files[0].BandBefore != BandYellow {
		t.Errorf("file delta = %+v", rep.Files[0])
	}
}

// The 20 percent rule: value >= ceil(base * 1.2) is worsened, anything less
// is existing, and a higher severity is worsened at any growth.
// Mutation: using a strict > or floor instead of the rounded-up limit fails
// the boundaries.
//
// TRLC-LINKS: REQ-EMG-156
func TestClassifyRule(t *testing.T) {
	warn := func(v int) Finding { return Finding{Severity: SeverityWarn, Value: v} }
	high := func(v int) Finding { return Finding{Severity: SeverityHigh, Value: v} }
	for _, tc := range []struct {
		name          string
		before, after Finding
		want          string
	}{
		{"same value", warn(10), warn(10), ChangeExisting},
		{"smaller", warn(10), warn(9), ChangeExisting},
		{"+10 percent", warn(10), warn(11), ChangeExisting},
		{"exactly +20 percent", warn(10), warn(12), ChangeWorsened},
		{"rounded up 4 -> 5", warn(4), warn(5), ChangeWorsened},
		{"12 -> 14 is below 14.4", warn(12), warn(14), ChangeExisting},
		{"12 -> 15 reaches 14.4", warn(12), warn(15), ChangeWorsened},
		{"warn to high by one step", warn(9), high(10), ChangeWorsened},
		{"high stays high, small growth", high(20), high(21), ChangeExisting},
		{"high stays high, 20 percent", high(20), high(24), ChangeWorsened},
		{"high to warn", high(20), warn(9), ChangeExisting},
	} {
		if got := classify(tc.before, tc.after); got != tc.want {
			t.Errorf("%s: classify = %s, want %s", tc.name, got, tc.want)
		}
	}
}

// A new file has only new entries and no score before; an unsupported file
// and an excluded one are skipped.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckNewFileAndSkips(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("keep.go", toyFunc("k", 1))
	r.Commit("base")
	r.Write("added.go", toyFunc("big", 9)+toyFunc("mid", 5)+toyFunc("small", 1))
	r.Write("notes.txt", toyFunc("txt", 9))
	r.Write("README.md", toyFunc("md", 9))
	r.Write("gen/skip.go", toyFunc("generated", 9))
	opts := deltaOptions(r)
	opts.Exclude = func(p string) bool { return strings.HasPrefix(p, "gen/") }

	rep, err := Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Entries) != 2 {
		t.Fatalf("entries = %+v, want big and mid", rep.Entries)
	}
	got := byFunction(t, rep, "added.go")
	wantEntry(t, got, "big", ChangeNew, SeverityHigh, 0, 9)
	wantEntry(t, got, "mid", ChangeNew, SeverityWarn, 0, 5)
	if len(rep.Files) != 1 || rep.Files[0].Status != "A" || rep.Files[0].ScoreBefore != nil || rep.Files[0].BandBefore != "" || rep.Files[0].ScoreAfter == nil {
		t.Errorf("Files = %+v", rep.Files)
	}
	if len(Regressions(rep)) != 1 {
		t.Errorf("Regressions = %+v", Regressions(rep))
	}

	opts.Exclude = nil
	opts.Paths = []string{"./gen"}
	rep, err = Check(opts)
	if err != nil || len(rep.Entries) != 1 || rep.Entries[0].Path != "gen/skip.go" {
		t.Errorf("Paths restricted to gen: %+v, %v", rep.Entries, err)
	}
	opts.Paths = []string{"../outside"}
	if _, err = Check(opts); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("a Paths entry outside the repository: %v", err)
	}
}

// A renamed file with edits is compared with its old self: the unchanged
// smell stays existing and only the edited function is worsened or new.
// Mutation: not mapping the old path makes every finding new.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckRenameWithEdits(t *testing.T) {
	r := NewFixtureRepo(t)
	stable := strings.Repeat("padding line that keeps the similarity of the rename high\n", 10)
	r.Write("old.go", stable+toyFunc("keep", 6)+toyFunc("edit", 5))
	r.Commit("base")
	r.Git("mv", "old.go", "new.go")
	r.Write("new.go", stable+toyFunc("keep", 6)+toyFunc("edit", 8)+toyFunc("added", 8))

	rep, err := Check(deltaOptions(r))
	if err != nil {
		t.Fatal(err)
	}
	got := byFunction(t, rep, "new.go")
	wantEntry(t, got, "keep", ChangeExisting, SeverityWarn, 6, 6)
	wantEntry(t, got, "edit", ChangeWorsened, SeverityHigh, 5, 8)
	wantEntry(t, got, "added", ChangeNew, SeverityHigh, 0, 8)
	if len(rep.Files) != 1 || rep.Files[0].Status != "R" || rep.Files[0].OldPath != "old.go" || rep.Files[0].ScoreBefore == nil {
		t.Errorf("Files = %+v", rep.Files)
	}
	for _, e := range rep.Entries {
		if e.Path == "old.go" {
			t.Errorf("entry for the old path: %+v", e)
		}
	}
}

// A function moved to another file, or renamed in place, with the same body
// is not new; a copy next to the original is, and so is a moved function whose
// body changed.
// Mutation: matching on the name only, or reusing a base function for two
// head functions, fails.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckMovedAndRenamedFunctions(t *testing.T) {
	r := NewFixtureRepo(t)
	// body renders a complexity-8 function; the tag keeps the bodies of the
	// functions apart, so that only the intended pairs share a hash.
	body := func(name, tag string, extra int) string {
		return "fn " + name + "\n" + strings.Repeat("if x\n", 7) + "ret " + tag + "\n" + strings.Repeat("extra\n", extra)
	}
	r.Write("a.go", body("moved", "m", 0)+body("renamed", "r", 0)+body("copied", "c", 0)+body("edited", "e", 0)+toyFunc("stay", 2))
	r.Write("b.go", toyFunc("other", 2))
	r.Commit("base")
	// moved leaves a.go for b.go unchanged; renamed changes its name; copied
	// gains a twin; edited moves with a changed body.
	r.Write("a.go", body("renamedNow", "r", 0)+body("copied", "c", 0)+toyFunc("stay", 2))
	r.Write("b.go", toyFunc("other", 2)+body("moved", "m", 0)+body("copied2", "c", 0)+body("edited", "e", 1))

	rep, err := Check(deltaOptions(r))
	if err != nil {
		t.Fatal(err)
	}
	a, b := byFunction(t, rep, "a.go"), byFunction(t, rep, "b.go")
	wantEntry(t, a, "renamedNow", ChangeExisting, SeverityHigh, 8, 8)
	wantEntry(t, a, "copied", ChangeExisting, SeverityHigh, 8, 8)
	wantEntry(t, b, "moved", ChangeExisting, SeverityHigh, 8, 8)
	wantEntry(t, b, "copied2", ChangeNew, SeverityHigh, 0, 8)
	// "edited" moved to b.go and its body changed, so the hash differs: new.
	wantEntry(t, b, "edited", ChangeNew, SeverityHigh, 0, 8)
	if got := Regressions(rep); len(got) != 2 {
		t.Errorf("Regressions = %+v, want copied2 and edited", got)
	}
}

// A repository without a commit has no base: every finding is new, a note says
// so and there is no error; an explicit base is a setup error.
// Mutation: failing on the missing HEAD fails this test.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckRepositoryWithoutCommits(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("big", 9)+toyFunc("mid", 5))
	r.Git("add", "a.go")
	r.Write("untracked.go", toyFunc("more", 8))

	rep, err := Check(deltaOptions(r))
	if err != nil {
		t.Fatal(err)
	}
	if rep.Base != "" || len(rep.Notes) != 1 || !strings.Contains(rep.Notes[0], "no commits") {
		t.Errorf("Base %q, Notes %v", rep.Base, rep.Notes)
	}
	if len(rep.Entries) != 3 {
		t.Fatalf("entries = %+v", rep.Entries)
	}
	for _, e := range rep.Entries {
		if e.Change != ChangeNew {
			t.Errorf("%s is %s, want new", e.Function, e.Change)
		}
	}
	if len(rep.Blocking) != 2 {
		t.Errorf("Blocking = %+v", rep.Blocking)
	}
	opts := deltaOptions(r)
	opts.Base = "origin/main"
	if _, err := Check(opts); !errors.Is(err, ErrBaseUnavailable) {
		t.Errorf("explicit base without commits: %v", err)
	}
}

// In a shallow clone that lacks the base the check is a setup error that names
// the base and tells what to do; the base that is present still works.
// Mutation: treating a missing base as "no findings" fails this test.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckShallowCloneWithoutBase(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("f", 2))
	r.Commit("one")
	r.Write("a.go", toyFunc("f", 9))
	r.Commit("two")
	dst := t.TempDir() + "/clone"
	gitIn(t, t.TempDir(), "clone", "-q", "--depth", "1", "file://"+r.Dir, dst)

	opts := deltaOptions(r)
	opts.Repo = NewGitRepo(dst)
	opts.Base = "HEAD~1"
	_, err := Check(opts)
	if !errors.Is(err, ErrBaseUnavailable) {
		t.Fatalf("err = %v, want ErrBaseUnavailable", err)
	}
	for _, want := range []string{"HEAD~1", "shallow", "fetch"} {
		if !strings.Contains(err.Error(), want) {
			t.Errorf("message %q does not mention %q", err.Error(), want)
		}
	}
	opts.Base = "HEAD...origin/gone"
	if _, err = Check(opts); err == nil {
		t.Error("an unknown head must be an error")
	}
	opts.Base = ""
	if rep, err := Check(opts); err != nil || len(rep.Entries) != 0 {
		t.Errorf("clean shallow clone against HEAD: %+v, %v", rep.Entries, err)
	}
	// In a full repository an unknown base is also a setup error.
	opts = deltaOptions(r)
	opts.Base = "nosuchbranch"
	if _, err = Check(opts); !errors.Is(err, ErrBaseUnavailable) || strings.Contains(err.Error(), "shallow") {
		t.Errorf("unknown base in a full repository: %v", err)
	}
	opts.Base = "--output=x"
	if _, err = Check(opts); !errors.Is(err, ErrInvalidRev) {
		t.Errorf("option-like base: %v", err)
	}
}

// A base that equals the head, and a clean working tree, give no entries and no
// files.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckBaseEqualsHead(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("big", 9))
	sha := r.Commit("one")
	for name, mod := range map[string]func(*CheckOptions){
		"clean working tree": func(o *CheckOptions) {},
		"same revision":      func(o *CheckOptions) { o.Base, o.Head = "HEAD", "HEAD" },
		"range form":         func(o *CheckOptions) { o.Base = sha + "..." + sha },
	} {
		opts := deltaOptions(r)
		mod(&opts)
		rep, err := Check(opts)
		if err != nil || len(rep.Entries) != 0 || len(rep.Files) != 0 || len(rep.Blocking) != 0 {
			t.Errorf("%s: %+v, %v", name, rep, err)
		}
	}
	opts := deltaOptions(r)
	opts.Base, opts.Head = "HEAD...HEAD", "HEAD"
	if _, err := Check(opts); !errors.Is(err, ErrInvalidRev) {
		t.Errorf("range plus head: %v", err)
	}
}

// jsonEqual compares two values by their JSON.
//
// TRLC-LINKS: REQ-EMG-159
func jsonEqual(t *testing.T, a, b any) bool {
	t.Helper()
	x, err1 := json.Marshal(a)
	y, err2 := json.Marshal(b)
	if err1 != nil || err2 != nil {
		t.Fatal(err1, err2)
	}
	return string(x) == string(y)
}

// With a head revision the comparison is against the merge base, reading both
// sides with git; the working tree is not consulted.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckRevisionRange(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("f", 5)+toyFunc("g", 5))
	fork := r.Commit("base")
	r.Git("checkout", "-q", "-b", "feature")
	r.Write("a.go", toyFunc("f", 9)+toyFunc("g", 5))
	r.Commit("feature")
	r.Git("checkout", "-q", "main")
	r.Write("a.go", toyFunc("f", 5)+toyFunc("g", 9))
	r.Commit("main moves on")
	r.Write("a.go", "uncommitted garbage "+toyFunc("f", 5))

	opts := deltaOptions(r)
	opts.Base = "main...feature"
	rep, err := Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Base != fork || rep.Head != strings.TrimSpace(r.Git("rev-parse", "feature")) {
		t.Errorf("Base %q Head %q; want the merge base and the feature head", rep.Base, rep.Head)
	}
	got := byFunction(t, rep, "a.go")
	wantEntry(t, got, "f", ChangeWorsened, SeverityHigh, 5, 9)
	wantEntry(t, got, "g", ChangeExisting, SeverityWarn, 5, 5)

	opts.Base, opts.Head = "main", "feature"
	again, err := Check(opts)
	if err != nil || !jsonEqual(t, rep, again) {
		t.Errorf("Base+Head and the range form differ: %v", err)
	}
}

// A branch that is behind its target is compared with the merge base, also for
// the working tree: a smell that the target fixed after the fork is not a
// regression of the branch (Q1 of the second review pass).
// Mutation: diffing against the tip of the target makes f appear worsened.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckWorkingTreeUsesMergeBaseOfDivergedBranch(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("f", 9)+toyFunc("g", 2))
	fork := r.Commit("base: f is a high smell")
	r.Git("checkout", "-q", "-b", "feature")
	r.Write("b.go", toyFunc("h", 2))
	r.Commit("feature adds a file")
	r.Git("checkout", "-q", "main")
	r.Write("a.go", toyFunc("f", 2)+toyFunc("g", 2))
	tip := r.Commit("main fixes f")
	r.Git("checkout", "-q", "feature")
	// The working tree of the branch still holds the old f, and a new edit.
	r.Write("a.go", toyFunc("f", 9)+toyFunc("g", 3))

	opts := deltaOptions(r)
	opts.Base = "main"
	rep, err := Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Base != fork {
		t.Errorf("Base %q, want the merge base %q", rep.Base, fork)
	}
	if len(rep.Blocking) != 0 {
		t.Errorf("a fixed smell of main must not block the stale branch: %+v", rep.Blocking)
	}
	wantEntry(t, byFunction(t, rep, "a.go"), "f", ChangeExisting, SeverityHigh, 9, 9)
	found := false
	for _, n := range rep.Notes {
		found = found || (strings.Contains(n, "merge base") && strings.Contains(n, tip[:12]))
	}
	if !found {
		t.Errorf("notes %q do not say that the merge base differs from the tip", rep.Notes)
	}

	// With the tip itself (the base equals the merge base) there is no note.
	r.Git("checkout", "-q", "-f", "-b", "fresh", "main")
	r.Write("a.go", toyFunc("f", 9)+toyFunc("g", 2))
	opts.Base = "main"
	rep, err = Check(opts)
	if err != nil || rep.Base != tip || len(rep.Notes) != 0 {
		t.Errorf("up-to-date branch: base %q notes %q err %v", rep.Base, rep.Notes, err)
	}
	if len(rep.Blocking) != 1 || rep.Blocking[0].Function != "f" {
		t.Errorf("a real regression must still block: %+v", rep.Blocking)
	}
}

// Without a common ancestor the resolved base is kept and a note says so.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckWorkingTreeWithoutCommonAncestor(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("f", 2))
	r.Commit("main")
	r.Git("checkout", "-q", "--orphan", "island")
	r.Write("a.go", toyFunc("f", 9))
	root := r.Commit("unrelated history")
	r.Git("checkout", "-q", "main")
	r.Git("checkout", "-q", "-b", "work")
	opts := deltaOptions(r)
	opts.Base = root
	rep, err := Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	if rep.Base != root {
		t.Errorf("Base %q, want the resolved base %q", rep.Base, root)
	}
	if len(rep.Notes) != 1 || !strings.Contains(rep.Notes[0], "no common ancestor") {
		t.Errorf("notes %q", rep.Notes)
	}
}

// A deleted file produces nothing and is not listed.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckDeletedFile(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("big", 9))
	r.Commit("one")
	r.Remove("a.go")
	rep, err := Check(deltaOptions(r))
	if err != nil || len(rep.Entries) != 0 || len(rep.Files) != 0 {
		t.Errorf("%+v, %v", rep, err)
	}
}

// Two runs, and runs with the paths given in another order, produce the same
// JSON; the report holds no timestamp and no absolute path and is sorted.
// Mutation: iterating a map to build the entries fails over repeated runs.
//
// TRLC-LINKS: REQ-EMG-159
func TestCheckIsDeterministic(t *testing.T) {
	r := NewFixtureRepo(t)
	names := []string{"z.go", "a.go", "m/n.go", "b.go", "m/a.go"}
	for _, n := range names {
		r.Write(n, toyFunc("x", 5)+toyFunc("y", 5)+toyFunc("dup", 5)+toyFunc("dup", 5))
	}
	r.Commit("base")
	for i, n := range names {
		r.Write(n, toyFunc("x", 5+i)+toyFunc("y", 8)+toyFunc("fresh", 9)+toyFunc("dup", 8)+toyFunc("dup", 9)+toyFunc("w", 5))
	}
	opts := deltaOptions(r)
	first, err := Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	for i := 0; i < 5; i++ {
		opts.Paths = []string{names[(i+1)%5], names[(i+2)%5], names[(i+3)%5], names[(i+4)%5], names[i%5]}
		again, err := Check(opts)
		if err != nil || !jsonEqual(t, first, again) {
			t.Fatalf("run %d differs (%v)", i, err)
		}
	}
	data, _ := json.Marshal(first)
	if strings.Contains(string(data), r.Dir) || strings.Contains(strings.ToLower(string(data)), "time") {
		t.Error("report holds an absolute path or a timestamp")
	}
	for i := 1; i < len(first.Entries); i++ {
		p, q := first.Entries[i-1], first.Entries[i]
		if p.Path > q.Path || (p.Path == q.Path && p.Function > q.Function) {
			t.Errorf("entries out of order: %+v before %+v", p, q)
		}
	}
	// Functions that share a name pair in order: the first dup was 5, now 8.
	var dups []DeltaEntry
	for _, e := range first.Entries {
		if e.Path == "a.go" && e.Function == "dup" {
			dups = append(dups, e)
		}
	}
	if len(dups) != 2 || dups[0].Change != ChangeWorsened || dups[1].Change != ChangeWorsened {
		t.Errorf("dup entries = %+v", dups)
	}
}

// The config of the head path is used for both sides, and the tier selects the
// default thresholds when no ConfigFor is given.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckConfigAndTier(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", toyFunc("f", 8))
	r.Commit("base")
	r.Write("a.go", toyFunc("f", 8)+"// touched\n")
	opts := deltaOptions(r)
	var seen []string
	opts.ConfigFor = func(p string) Config {
		seen = append(seen, p)
		return Config{Thresholds: map[string]Threshold{CodeComplexMethod: {Warn: 3, High: 99}}}
	}
	rep, err := Check(opts)
	if err != nil || len(rep.Entries) != 1 || rep.Entries[0].Change != ChangeExisting || rep.Entries[0].Severity != SeverityWarn {
		t.Fatalf("%+v, %v", rep.Entries, err)
	}
	if fmt.Sprint(seen) != "[a.go]" {
		t.Errorf("ConfigFor called for %v", seen)
	}

	opts.ConfigFor = nil
	opts.Tier = "product" // complex_method 7 / 12 applies
	if rep, err = Check(opts); err != nil || len(rep.Entries) != 1 || rep.Tier != "product" || rep.Entries[0].Severity != SeverityWarn {
		t.Errorf("tier product: %+v, %v", rep.Entries, err)
	}
	opts.Tier = ""
	if rep, err = Check(opts); err != nil || len(rep.Entries) != 0 {
		t.Errorf("default thresholds (warn 10): %+v, %v", rep.Entries, err)
	}
	opts.Engine = nil
	if _, err = Check(opts); err == nil {
		t.Error("a missing engine must be an error")
	}
}

// A CRLF working-tree copy of an LF base is the same code for the fake engine
// and for the body hash: nothing becomes new.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckCRLFWorkingTree(t *testing.T) {
	r := NewFixtureRepo(t)
	src := toyFunc("big", 9) + toyFunc("mid", 5)
	r.Write("a.go", src)
	r.Commit("lf")
	r.Write("a.go", strings.ReplaceAll(src, "\n", "\r\n"))
	rep, err := Check(deltaOptions(r))
	if err != nil || len(rep.Entries) != 2 {
		t.Fatalf("%+v, %v", rep.Entries, err)
	}
	for _, e := range rep.Entries {
		if e.Change != ChangeExisting {
			t.Errorf("%s is %s after a line-ending change", e.Function, e.Change)
		}
	}
	if len(Regressions(rep)) != 0 {
		t.Errorf("Regressions = %+v", Regressions(rep))
	}
}

// A changed head file that is larger than the 4 MiB limit, or not a regular
// file, is skipped, listed in Skipped with its reason and noted: the findings of
// a file the check cannot read are unknown, so the report must be able to fail
// on it. The limit is the same for a file of the working tree and for one read
// from a revision with Head set (readHead had no cap there), and the limit's
// edge (exactly 4 MiB) is analysed.
//
// Mutation: dropping the length test of the revision branch of readHead, or the
// Skipped record, fails.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckSkipsOversizedAndIrregularHeadFiles(t *testing.T) {
	pad := func(n int) string {
		return toyFunc("big", 9) + "#" + strings.Repeat("x", n-len(toyFunc("big", 9))-2) + "\n"
	}
	const limit = 4 << 20
	r := NewFixtureRepo(t)
	r.Write("keep.go", toyFunc("k", 1))
	base := r.Commit("base")
	r.Write("huge.go", pad(limit+1))
	r.Write("edge.go", pad(limit))
	r.Write("small.go", toyFunc("s", 9))
	r.Commit("head")

	check := func(t *testing.T, headRev bool) DeltaReport {
		t.Helper()
		opts := deltaOptions(r)
		opts.Base = base
		if headRev {
			opts.Head = "HEAD"
		}
		rep, err := Check(opts)
		if err != nil {
			t.Fatal(err)
		}
		return rep
	}
	for name, headRev := range map[string]bool{"revision head": true, "working tree": false} {
		rep := check(t, headRev)
		if len(rep.Skipped) != 1 || rep.Skipped[0].Path != "huge.go" || !strings.Contains(rep.Skipped[0].Reason, "larger than 4194304") {
			t.Errorf("%s: Skipped = %+v, want huge.go only", name, rep.Skipped)
		}
		if !strings.Contains(strings.Join(rep.Notes, "\n"), "skipped huge.go") {
			t.Errorf("%s: notes = %v", name, rep.Notes)
		}
		paths := map[string]bool{}
		for _, e := range rep.Entries {
			paths[e.Path] = true
		}
		if paths["huge.go"] || !paths["edge.go"] || !paths["small.go"] {
			t.Errorf("%s: entries for %v, want edge.go and small.go but not huge.go", name, paths)
		}
	}

	// Not a regular file: a symbolic link at head (working tree).
	r2 := NewFixtureRepo(t)
	r2.Write("keep.go", toyFunc("k", 1))
	r2.Commit("base")
	if err := os.Symlink("keep.go", r2.Dir+"/link.go"); err != nil {
		t.Skip("symbolic links are not available")
	}
	rep, err := Check(deltaOptions(r2))
	if err != nil {
		t.Fatal(err)
	}
	if len(rep.Skipped) != 1 || rep.Skipped[0].Path != "link.go" || rep.Skipped[0].Reason != "not a regular file" {
		t.Errorf("Skipped = %+v, want link.go as not a regular file", rep.Skipped)
	}
	r2.Remove("link.go")
	if clean, err := Check(deltaOptions(r2)); err != nil || clean.Skipped == nil || len(clean.Skipped) != 0 {
		t.Errorf("a check without skips must report an empty list: %+v, %v", clean.Skipped, err)
	}
}

// failingFake fails the analysis of the files whose path contains "bad" or
// whose source contains FAILME.
type failingFake struct{ fakeEngine }

// TRLC-LINKS: REQ-EMG-156
func (e failingFake) Analyze(inputs []FileInput, cfg Config) ([]FileResult, error) {
	if strings.Contains(inputs[0].Path, "bad") || strings.Contains(string(inputs[0].Source), "FAILME") {
		return nil, errors.New("boom")
	}
	return e.fakeEngine.Analyze(inputs, cfg)
}

// A changed file the engine cannot analyse is a skipped file with the reason,
// not a failure of the whole check: the other files are still compared. A base
// side that cannot be analysed leaves a note and makes every finding of the
// file new.
//
// Mutation: returning the engine error from load ends the check and fails.
//
// TRLC-LINKS: REQ-EMG-156
func TestCheckListsFilesTheEngineCannotAnalyse(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("good.go", toyFunc("g", 1))
	r.Write("bad_base.go", toyFunc("b", 9))
	r.Commit("base")
	r.Write("good.go", toyFunc("g", 9))
	r.Write("bad_new.go", toyFunc("n", 9))
	opts := deltaOptions(r)
	opts.Engine = failingFake{}
	rep, err := Check(opts)
	if err != nil {
		t.Fatalf("an engine failure on one file must not end the check: %v", err)
	}
	if len(rep.Skipped) != 1 || rep.Skipped[0].Path != "bad_new.go" || !strings.Contains(rep.Skipped[0].Reason, "the analysis failed:") || !strings.Contains(rep.Skipped[0].Reason, "boom") {
		t.Errorf("Skipped = %+v", rep.Skipped)
	}
	if got := byFunction(t, rep, "good.go"); len(got) != 1 {
		t.Errorf("the other file was not compared: %+v", rep.Entries)
	}
	// The base side fails: the head is compared with nothing, every finding is
	// new, and a note says so.
	r2 := NewFixtureRepo(t)
	r2.Write("a.go", "FAILME\n"+toyFunc("f", 9))
	r2.Commit("base")
	r2.Write("a.go", toyFunc("f", 9))
	opts = deltaOptions(r2)
	opts.Engine = failingFake{}
	if rep, err = Check(opts); err != nil || len(rep.Skipped) != 0 {
		t.Fatalf("base failure: %+v, %v", rep.Skipped, err)
	}
	wantEntry(t, byFunction(t, rep, "a.go"), "f", ChangeNew, SeverityHigh, 0, 9)
	if !strings.Contains(strings.Join(rep.Notes, "\n"), "could not analyse the base of a.go") {
		t.Errorf("notes = %v", rep.Notes)
	}
}
