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
		out = append(out, analyzeToy(in, limit))
	}
	return out, nil
}

// analyzeToy analyses one file of the toy format.
func analyzeToy(in FileInput, limit Threshold) FileResult {
	lines := strings.Split(strings.ReplaceAll(string(in.Source), "\r\n", "\n"), "\n")
	if lines[len(lines)-1] == "" {
		lines = lines[:len(lines)-1]
	}
	res := FileResult{Path: in.Path, Language: in.Language, LOC: len(lines), Functions: []Function{}}
	for i := 0; i < len(lines); i++ {
		if strings.HasPrefix(lines[i], "fn ") {
			res.Functions = append(res.Functions, toyFunction(lines, i, limit))
		}
	}
	return res
}

// toyFunction is the function that starts at line i of the toy format, with a
// complex_method finding when it reaches the limit.
func toyFunction(lines []string, i int, limit Threshold) Function {
	name := strings.TrimPrefix(lines[i], "fn ")
	end, cyc := i+1, 1
	for end < len(lines) && !strings.HasPrefix(lines[end], "fn ") {
		if strings.HasPrefix(lines[end], "if") {
			cyc++
		}
		end++
	}
	f := Function{Name: name, Kind: "function", StartLine: i + 1, EndLine: end, Cyclomatic: cyc}
	if cyc < limit.Warn {
		return f
	}
	sev := SeverityWarn
	if cyc >= limit.High {
		sev = SeverityHigh
	}
	f.Findings = []Finding{{
		Code: CodeComplexMethod, Severity: sev, StartLine: f.StartLine, EndLine: f.EndLine,
		Metric: "cyclomatic", Value: cyc, Message: fmt.Sprintf("%s has complexity %d", name, cyc),
	}}
	return f
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

// entryWant is the expected change class, severity and values of one entry.
type entryWant struct {
	name, change  string
	sev           Severity
	before, after int
}

// wantEntry checks the change class, severity and values of one entry.
//
// TRLC-LINKS: REQ-EMG-156
func wantEntry(t *testing.T, got map[string]DeltaEntry, w entryWant) {
	t.Helper()
	e, ok := got[w.name]
	mustEqual(t, [2]any{w.name, ok}, [2]any{w.name, true})
	wantEqual(t, [5]any{w.name, e.Change, e.Severity, e.ValueBefore, e.ValueAfter}, [5]any{w.name, w.change, w.sev, w.before, w.after})
}

// mustCheck runs the check or stops the test.
func mustCheck(t *testing.T, opts CheckOptions) DeltaReport {
	t.Helper()
	rep, err := Check(opts)
	if err != nil {
		t.Fatal(err)
	}
	return rep
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

	rep := mustCheck(t, deltaOptions(r))
	wantEqual(t, [3]string{rep.Schema, rep.Base, rep.Head}, [3]string{DeltaSchema, baseSha, WorkingTree})
	got := byFunction(t, rep, "a.go")
	wantEntry(t, got, entryWant{"worse", ChangeWorsened, SeverityHigh, 5, 8})
	wantEntry(t, got, entryWant{"better", ChangeExisting, SeverityWarn, 8, 5})
	wantEntry(t, got, entryWant{"same", ChangeExisting, SeverityWarn, 5, 5})
	wantEntry(t, got, entryWant{"grow", ChangeWorsened, SeverityWarn, 4, 5})
	wantEntry(t, got, entryWant{"fresh", ChangeNew, SeverityHigh, 0, 10})
	wantEntry(t, got, entryWant{"freshWarn", ChangeNew, SeverityWarn, 0, 5})
	wantEqual(t, len(got), 6) // deleted and clean functions produce no entry
	wantEqual(t, [2]Severity{got["worse"].SeverityBefore, got["fresh"].SeverityBefore}, [2]Severity{SeverityWarn, ""})
	requireBlocking(t, rep)
	requireFileDelta(t, rep)
}

// requireBlocking checks that the regressions of the worked example are the
// high new and worsened entries, sorted, and the Blocking list of the report.
func requireBlocking(t *testing.T, rep DeltaReport) {
	t.Helper()
	blocking := Regressions(rep)
	mustEqual(t, len(blocking), 2)
	wantEqual(t, [2]string{blocking[0].Function, blocking[1].Function}, [2]string{"fresh", "worse"})
	wantEqual(t, fmt.Sprint(rep.Blocking), fmt.Sprint(blocking))
}

// requireFileDelta checks the one file delta of the worked example.
func requireFileDelta(t *testing.T, rep DeltaReport) {
	t.Helper()
	mustEqual(t, len(rep.Files), 1)
	f := rep.Files[0]
	mustEqual(t, [3]bool{f.Path == "a.go", f.ScoreBefore != nil, f.ScoreAfter != nil}, [3]bool{true, true, true})
	wantEqual(t, [3]any{*f.ScoreBefore, *f.ScoreAfter, f.BandBefore}, [3]any{5.0, 4.0, BandYellow})
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

	rep := mustCheck(t, opts)
	mustEqual(t, len(rep.Entries), 2) // big and mid
	got := byFunction(t, rep, "added.go")
	wantEntry(t, got, entryWant{"big", ChangeNew, SeverityHigh, 0, 9})
	wantEntry(t, got, entryWant{"mid", ChangeNew, SeverityWarn, 0, 5})
	mustEqual(t, len(rep.Files), 1)
	f := rep.Files[0]
	wantEqual(t, [4]any{f.Status, f.ScoreBefore == nil, f.BandBefore, f.ScoreAfter != nil}, [4]any{"A", true, "", true})
	wantEqual(t, len(Regressions(rep)), 1)

	opts.Exclude = nil
	opts.Paths = []string{"./gen"}
	rep = mustCheck(t, opts)
	mustEqual(t, len(rep.Entries), 1) // restricted to gen
	wantEqual(t, rep.Entries[0].Path, "gen/skip.go")
	opts.Paths = []string{"../outside"}
	_, err := Check(opts)
	wantEqual(t, errors.Is(err, ErrInvalidPath), true) // a Paths entry outside the repository
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

	rep := mustCheck(t, deltaOptions(r))
	got := byFunction(t, rep, "new.go")
	wantEntry(t, got, entryWant{"keep", ChangeExisting, SeverityWarn, 6, 6})
	wantEntry(t, got, entryWant{"edit", ChangeWorsened, SeverityHigh, 5, 8})
	wantEntry(t, got, entryWant{"added", ChangeNew, SeverityHigh, 0, 8})
	mustEqual(t, len(rep.Files), 1)
	wantEqual(t, [3]any{rep.Files[0].Status, rep.Files[0].OldPath, rep.Files[0].ScoreBefore != nil}, [3]any{"R", "old.go", true})
	wantEqual(t, len(byFunction(t, rep, "old.go")), 0) // no entry for the old path
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
	wantEntry(t, a, entryWant{"renamedNow", ChangeExisting, SeverityHigh, 8, 8})
	wantEntry(t, a, entryWant{"copied", ChangeExisting, SeverityHigh, 8, 8})
	wantEntry(t, b, entryWant{"moved", ChangeExisting, SeverityHigh, 8, 8})
	wantEntry(t, b, entryWant{"copied2", ChangeNew, SeverityHigh, 0, 8})
	// "edited" moved to b.go and its body changed, so the hash differs: new.
	wantEntry(t, b, entryWant{"edited", ChangeNew, SeverityHigh, 0, 8})
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

	rep := mustCheck(t, deltaOptions(r))
	wantEqual(t, [2]any{rep.Base, len(rep.Notes)}, [2]any{"", 1})
	wantContains(t, rep.Notes[0], "no commits")
	mustEqual(t, len(rep.Entries), 3)
	for _, e := range rep.Entries {
		wantEqual(t, [2]string{e.Function, e.Change}, [2]string{e.Function, ChangeNew})
	}
	wantEqual(t, len(rep.Blocking), 2)
	opts := deltaOptions(r)
	opts.Base = "origin/main"
	_, err := Check(opts)
	wantEqual(t, errors.Is(err, ErrBaseUnavailable), true) // an explicit base without commits
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
	mustEqual(t, errors.Is(err, ErrBaseUnavailable), true)
	wantAll(t, err.Error(), "HEAD~1", "shallow", "fetch")
	opts.Base = "HEAD...origin/gone"
	_, err = Check(opts)
	wantEqual(t, err != nil, true) // an unknown head is an error
	opts.Base = ""
	wantEqual(t, len(mustCheck(t, opts).Entries), 0) // a clean shallow clone against HEAD
	// In a full repository an unknown base is also a setup error.
	opts = deltaOptions(r)
	opts.Base = "nosuchbranch"
	_, err = Check(opts)
	wantEqual(t, [2]bool{errors.Is(err, ErrBaseUnavailable), strings.Contains(err.Error(), "shallow")}, [2]bool{true, false})
	opts.Base = "--output=x"
	_, err = Check(opts)
	wantEqual(t, errors.Is(err, ErrInvalidRev), true) // an option-like base
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
		rep := mustCheck(t, opts)
		wantEqual(t, [4]any{name, len(rep.Entries), len(rep.Files), len(rep.Blocking)}, [4]any{name, 0, 0, 0})
	}
	opts := deltaOptions(r)
	opts.Base, opts.Head = "HEAD...HEAD", "HEAD"
	_, err := Check(opts)
	wantEqual(t, errors.Is(err, ErrInvalidRev), true) // range plus head
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
	wantEntry(t, got, entryWant{"f", ChangeWorsened, SeverityHigh, 5, 9})
	wantEntry(t, got, entryWant{"g", ChangeExisting, SeverityWarn, 5, 5})

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
	rep := mustCheck(t, opts)
	wantEqual(t, rep.Base, fork)       // the merge base
	wantEqual(t, len(rep.Blocking), 0) // a fixed smell of main must not block the stale branch
	wantEntry(t, byFunction(t, rep, "a.go"), entryWant{"f", ChangeExisting, SeverityHigh, 9, 9})
	wantEqual(t, noteSaysMergeBase(rep.Notes, tip), true)

	// With the tip itself (the base equals the merge base) there is no note.
	r.Git("checkout", "-q", "-f", "-b", "fresh", "main")
	r.Write("a.go", toyFunc("f", 9)+toyFunc("g", 2))
	rep = mustCheck(t, opts)
	wantEqual(t, [2]any{rep.Base, len(rep.Notes)}, [2]any{tip, 0})
	mustEqual(t, len(rep.Blocking), 1) // a real regression still blocks
	wantEqual(t, rep.Blocking[0].Function, "f")
}

// noteSaysMergeBase reports whether a note says that the merge base, not the tip
// of the target, was compared.
func noteSaysMergeBase(notes []string, tip string) bool {
	for _, n := range notes {
		if strings.Contains(n, "merge base") && strings.Contains(n, tip[:12]) {
			return true
		}
	}
	return false
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
	first := mustCheck(t, opts)
	for i := 0; i < 5; i++ {
		opts.Paths = []string{names[(i+1)%5], names[(i+2)%5], names[(i+3)%5], names[(i+4)%5], names[i%5]}
		wantEqual(t, jsonEqual(t, first, mustCheck(t, opts)), true)
	}
	data, _ := json.Marshal(first)
	wantNone(t, string(data), r.Dir)
	wantNone(t, strings.ToLower(string(data)), "time") // no absolute path and no timestamp
	wantEqual(t, entriesAreSorted(first.Entries), true)
	// Functions that share a name pair in order: the first dup was 5, now 8.
	var dups []DeltaEntry
	for _, e := range first.Entries {
		if e.Path == "a.go" && e.Function == "dup" {
			dups = append(dups, e)
		}
	}
	mustEqual(t, len(dups), 2)
	wantEqual(t, [2]string{dups[0].Change, dups[1].Change}, [2]string{ChangeWorsened, ChangeWorsened})
}

// entriesAreSorted reports whether the entries are in the order of path and
// function.
func entriesAreSorted(entries []DeltaEntry) bool {
	for i := 1; i < len(entries); i++ {
		p, q := entries[i-1], entries[i]
		if p.Path > q.Path || (p.Path == q.Path && p.Function > q.Function) {
			return false
		}
	}
	return true
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
	rep := mustCheck(t, opts)
	mustEqual(t, len(rep.Entries), 1)
	wantEqual(t, [2]string{rep.Entries[0].Change, string(rep.Entries[0].Severity)}, [2]string{ChangeExisting, string(SeverityWarn)})
	wantEqual(t, fmt.Sprint(seen), "[a.go]")

	opts.ConfigFor = nil
	opts.Tier = "product" // complex_method 7 / 12 applies
	rep = mustCheck(t, opts)
	mustEqual(t, len(rep.Entries), 1)
	wantEqual(t, [2]string{rep.Tier, string(rep.Entries[0].Severity)}, [2]string{"product", string(SeverityWarn)})
	opts.Tier = ""
	wantEqual(t, len(mustCheck(t, opts).Entries), 0) // default thresholds: warn 10
	opts.Engine = nil
	_, err := Check(opts)
	wantEqual(t, err != nil, true) // a missing engine is an error
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
	const limit = 4 << 20
	r := NewFixtureRepo(t)
	r.Write("keep.go", toyFunc("k", 1))
	base := r.Commit("base")
	r.Write("huge.go", padToy(limit+1))
	r.Write("edge.go", padToy(limit))
	r.Write("small.go", toyFunc("s", 9))
	r.Commit("head")
	for name, headRev := range map[string]bool{"revision head": true, "working tree": false} {
		opts := deltaOptions(r)
		opts.Base = base
		if headRev {
			opts.Head = "HEAD"
		}
		t.Run(name, func(t *testing.T) { requireOnlyHugeSkipped(t, opts) })
	}
	requireSymlinkSkipped(t)
}

// padToy is a toy file of n bytes with a high function and a comment.
func padToy(n int) string {
	return toyFunc("big", 9) + "#" + strings.Repeat("x", n-len(toyFunc("big", 9))-2) + "\n"
}

// requireOnlyHugeSkipped checks that the file above the limit is the only one
// skipped, with its reason and a note, and that the file at the limit is
// analysed.
func requireOnlyHugeSkipped(t *testing.T, opts CheckOptions) {
	t.Helper()
	rep := mustCheck(t, opts)
	mustEqual(t, len(rep.Skipped), 1)
	wantEqual(t, rep.Skipped[0].Path, "huge.go")
	wantContains(t, rep.Skipped[0].Reason, "larger than 4194304")
	wantContains(t, strings.Join(rep.Notes, "\n"), "skipped huge.go")
	paths := map[string]bool{}
	for _, e := range rep.Entries {
		paths[e.Path] = true
	}
	wantEqual(t, [3]bool{paths["huge.go"], paths["edge.go"], paths["small.go"]}, [3]bool{false, true, true})
}

// requireSymlinkSkipped checks that a symbolic link at head is skipped as not a
// regular file and that a check without skips reports an empty list.
func requireSymlinkSkipped(t *testing.T) {
	t.Helper()
	r := NewFixtureRepo(t)
	r.Write("keep.go", toyFunc("k", 1))
	r.Commit("base")
	if err := os.Symlink("keep.go", r.Dir+"/link.go"); err != nil {
		t.Skip("symbolic links are not available")
	}
	rep := mustCheck(t, deltaOptions(r))
	mustEqual(t, len(rep.Skipped), 1)
	wantEqual(t, [2]string{rep.Skipped[0].Path, rep.Skipped[0].Reason}, [2]string{"link.go", "not a regular file"})
	r.Remove("link.go")
	clean := mustCheck(t, deltaOptions(r))
	wantEqual(t, [2]bool{clean.Skipped != nil, len(clean.Skipped) == 0}, [2]bool{true, true})
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
	rep := mustCheck(t, opts) // an engine failure on one file must not end the check
	mustEqual(t, len(rep.Skipped), 1)
	wantEqual(t, rep.Skipped[0].Path, "bad_new.go")
	wantAll(t, rep.Skipped[0].Reason, "the analysis failed:", "boom")
	wantEqual(t, len(byFunction(t, rep, "good.go")), 1) // the other file was compared
	requireFailingBaseIsNew(t)
}

// requireFailingBaseIsNew checks that a base side that cannot be analysed
// leaves a note and makes every finding of the file new.
func requireFailingBaseIsNew(t *testing.T) {
	t.Helper()
	r := NewFixtureRepo(t)
	r.Write("a.go", "FAILME\n"+toyFunc("f", 9))
	r.Commit("base")
	r.Write("a.go", toyFunc("f", 9))
	opts := deltaOptions(r)
	opts.Engine = failingFake{}
	rep := mustCheck(t, opts)
	wantEqual(t, len(rep.Skipped), 0)
	wantEntry(t, byFunction(t, rep, "a.go"), entryWant{"f", ChangeNew, SeverityHigh, 0, 9})
	wantContains(t, strings.Join(rep.Notes, "\n"), "could not analyse the base of a.go")
}
