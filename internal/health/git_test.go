// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"errors"
	"os"
	"reflect"
	"strings"
	"testing"
)

// The git child gets the allow-listed variables, GIT_OPTIONAL_LOCKS and the
// switches that turn the user and system configuration off; tokens and GIT_*
// overrides of the caller never reach it.
// Mutation: passing os.Environ() through fails this test.
//
// TRLC-LINKS: REQ-EMG-156
func TestGitEnvIsAllowListed(t *testing.T) {
	got := gitEnv([]string{
		"PATH=/bin", "HOME=/h", "LANG=C", "LC_ALL=C", "TMPDIR=/t",
		"GITHUB_TOKEN=secret", "GIT_DIR=/elsewhere", "GIT_CONFIG_GLOBAL=/x", "SSH_AUTH_SOCK=/s",
	})
	want := []string{"PATH=/bin", "HOME=/h", "LANG=C", "LC_ALL=C", "TMPDIR=/t", "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1"}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("env = %v, want %v", got, want)
	}
}

// On Windows the variable names match without regard to case (Path, SystemRoot)
// and git also gets SYSTEMROOT, COMSPEC, TEMP and TMP; elsewhere names match
// exactly and those four are not passed (Q6 of the second review pass). No home
// fallback is invented and the global configuration stays off.
// Mutation: a case-sensitive match drops Path, so git finds no program on Windows.
//
// TRLC-LINKS: REQ-EMG-156
func TestGitEnvForWindowsAndOthers(t *testing.T) {
	environ := []string{
		`Path=C:\Windows\System32`, `SystemRoot=C:\Windows`, `ComSpec=C:\Windows\System32\cmd.exe`,
		`Temp=C:\tmp`, `TMP=C:\tmp2`, `USERPROFILE=C:\Users\me`, `GITHUB_TOKEN=secret`, `lang=C`,
	}
	tail := []string{"GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1"}
	win := gitEnvFor("windows", environ)
	wantWin := append([]string{
		`Path=C:\Windows\System32`, `lang=C`, `SystemRoot=C:\Windows`, `ComSpec=C:\Windows\System32\cmd.exe`,
		`Temp=C:\tmp`, `TMP=C:\tmp2`,
	}, tail...)
	if !reflect.DeepEqual(win, wantWin) {
		t.Errorf("windows env = %v\nwant %v", win, wantWin)
	}
	if got := gitEnvFor("linux", environ); !reflect.DeepEqual(got, tail) {
		t.Errorf("linux env = %v, want only %v (exact names, no Windows variables)", got, tail)
	}
	if got := gitEnvFor("linux", []string{"PATH=/bin", "SYSTEMROOT=/x"}); !reflect.DeepEqual(got, append([]string{"PATH=/bin"}, tail...)) {
		t.Errorf("linux env = %v", got)
	}
}

// Revisions that git could read as options, or that name something other than
// a commit, are refused before git runs; paths must be clean and relative.
// Mutation: dropping the leading "-" check fails the first case.
//
// TRLC-LINKS: REQ-EMG-156
func TestRevAndPathValidation(t *testing.T) {
	for _, rev := range []string{"", "-p", "--output=x", "a b", "a\nb", "HEAD:file", "a\x00b"} {
		if err := checkRev(rev); !errors.Is(err, ErrInvalidRev) {
			t.Errorf("checkRev(%q) = %v, want ErrInvalidRev", rev, err)
		}
	}
	for _, rev := range []string{"HEAD", "HEAD~1", "HEAD^", "main", "origin/main", "v1.0", "abc123", "main@{1}"} {
		if err := checkRev(rev); err != nil {
			t.Errorf("checkRev(%q) = %v", rev, err)
		}
	}
	for _, p := range []string{"", ".", "..", "../x", "a/../../x", "/etc/passwd", "a\x00b", "a\nb.go", "::error::x\n.go", "a\x1b[2Jb.go", "a\x7fb", "a\u009bb"} {
		if _, err := normalizeRepoPath(p); !errors.Is(err, ErrInvalidPath) {
			t.Errorf("normalizeRepoPath(%q) = %v, want ErrInvalidPath", p, err)
		}
	}
	if got, err := normalizeRepoPath("./a//b/../c.go"); err != nil || got != "a/c.go" {
		t.Errorf("normalizeRepoPath = %q, %v", got, err)
	}
	if _, err := gitPath("a/./b"); err == nil {
		t.Error("gitPath accepted an unclean path from git")
	}
}

// ShowFile returns the raw bytes (no trimming, CRLF kept), tells a missing
// path from a missing revision, and refuses unsafe arguments.
// Mutation: TrimSpace on the output fails the raw-bytes check.
//
// TRLC-LINKS: REQ-EMG-156
func TestShowFile(t *testing.T) {
	r := NewFixtureRepo(t)
	raw := "line one\r\nline two\r\n\n\n"
	r.Write("pkg/a.go", raw)
	sha := r.Commit("one")
	repo := r.Repo()

	got, err := repo.ShowFile("HEAD", "pkg/a.go")
	if err != nil || string(got) != raw {
		t.Fatalf("ShowFile = %q, %v; want the raw bytes", got, err)
	}
	if got, err = repo.ShowFile(sha, "./pkg//a.go"); err != nil || string(got) != raw {
		t.Fatalf("ShowFile by sha = %q, %v", got, err)
	}
	if _, err = repo.ShowFile("HEAD", "pkg/missing.go"); !errors.Is(err, ErrNotInRev) {
		t.Errorf("missing path: %v, want ErrNotInRev", err)
	}
	if _, err = repo.ShowFile("nosuchbranch", "pkg/a.go"); !errors.Is(err, ErrUnknownRev) {
		t.Errorf("unknown rev: %v, want ErrUnknownRev", err)
	}
	if _, err = repo.ShowFile("--output=x", "pkg/a.go"); !errors.Is(err, ErrInvalidRev) {
		t.Errorf("option-like rev: %v, want ErrInvalidRev", err)
	}
	if _, err = repo.ShowFile("HEAD", "../outside"); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("traversal: %v, want ErrInvalidPath", err)
	}
	if _, err = repo.ShowFile("HEAD", "pkg"); err == nil || errors.Is(err, ErrNotInRev) {
		t.Errorf("a directory is not a file at the revision: %v", err)
	}
}

// ChangedFiles against the working tree lists added, modified, deleted and
// renamed files and the untracked ones, skips ignored files, and returns clean
// sorted paths.
// Mutation: dropping the untracked listing fails the "new.go" check.
//
// TRLC-LINKS: REQ-EMG-156
func TestChangedFilesWorkingTree(t *testing.T) {
	r := NewFixtureRepo(t)
	body := strings.Repeat("a line of code that is long enough to be matched as a rename\n", 12)
	r.Write(".gitignore", "ignored.go\n")
	r.Write("keep.go", "package keep\n")
	r.Write("mod.go", "package mod\n")
	r.Write("gone.go", "package gone\n")
	r.Write("old name.go", body)
	r.Commit("base")

	r.Write("mod.go", "package mod // changed\n")
	r.Remove("gone.go")
	if err := os.MkdirAll(r.Dir+"/dir", 0o755); err != nil {
		t.Fatal(err)
	}
	r.Git("mv", "old name.go", "dir/new name.go")
	r.Write("dir/new name.go", body+"one more line\n")
	r.Write("staged.go", "package staged\n")
	r.Git("add", "staged.go")
	r.Write("new.go", "package n\n")
	r.Write("ignored.go", "package ignored\n")

	got, err := r.Repo().ChangedFiles("HEAD", "")
	if err != nil {
		t.Fatal(err)
	}
	want := []ChangedFile{
		{Status: "R", Score: 95, OldPath: "old name.go", Path: "dir/new name.go"},
		{Status: "D", Path: "gone.go"},
		{Status: "M", Path: "mod.go"},
		{Status: "A", Path: "new.go"},
		{Status: "A", Path: "staged.go"},
	}
	if len(got) != len(want) {
		t.Fatalf("ChangedFiles = %+v, want %d entries", got, len(want))
	}
	for i := range want {
		if got[i].Status != want[i].Status || got[i].Path != want[i].Path || got[i].OldPath != want[i].OldPath {
			t.Errorf("entry %d = %+v, want %+v", i, got[i], want[i])
		}
	}
	if got[0].Score < 50 || got[0].Score > 100 {
		t.Errorf("rename score %d out of range", got[0].Score)
	}
}

// Against a revision range the diff is base...head, so a change that landed
// on the base branch after the fork is not listed.
// Mutation: a two-dot diff lists main-only.go and fails this test.
//
// TRLC-LINKS: REQ-EMG-156
func TestChangedFilesRange(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", "package a\n")
	fork := r.Commit("base")
	r.Git("checkout", "-q", "-b", "feature")
	r.Write("feature.go", "package f\n")
	r.Write("a.go", "package a // x\n")
	r.Commit("feature")
	r.Git("checkout", "-q", "main")
	r.Write("main-only.go", "package m\n")
	r.Commit("main moves on")

	repo := r.Repo()
	got, err := repo.ChangedFiles("main", "feature")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path != "a.go" || got[0].Status != "M" || got[1].Path != "feature.go" || got[1].Status != "A" {
		t.Fatalf("ChangedFiles(main, feature) = %+v", got)
	}
	mb, err := repo.MergeBase("main", "feature")
	if err != nil || mb != fork {
		t.Fatalf("MergeBase = %q, %v; want %q", mb, err, fork)
	}
	if _, err = repo.ChangedFiles("main", "--bad"); !errors.Is(err, ErrInvalidRev) {
		t.Errorf("option-like head: %v", err)
	}
	if _, err = repo.ChangedFiles("", "feature"); !errors.Is(err, ErrInvalidRev) {
		t.Errorf("head without base: %v", err)
	}
}

// Paths are relative to the directory the reader runs in, for the diff, the
// untracked listing and ShowFile alike.
//
// TRLC-LINKS: REQ-EMG-156
func TestPathsAreRelativeToDir(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("sub/a.go", "package a\n")
	r.Write("other/b.go", "package b\n")
	r.Commit("base")
	r.Write("sub/a.go", "package a // x\n")
	r.Write("sub/new.go", "package n\n")
	r.Write("other/b.go", "package b // x\n")

	repo := NewGitRepo(r.Dir + "/sub")
	got, err := repo.ChangedFiles("HEAD", "")
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Path != "a.go" || got[1].Path != "new.go" {
		t.Fatalf("ChangedFiles in sub = %+v", got)
	}
	if b, err := repo.ShowFile("HEAD", "a.go"); err != nil || string(b) != "package a\n" {
		t.Fatalf("ShowFile in sub = %q, %v", b, err)
	}
}

// An empty repository has no commits and lists every file as added; a
// repository with commits says so.
//
// TRLC-LINKS: REQ-EMG-156
func TestHasCommitsAndEmptyListing(t *testing.T) {
	r := NewFixtureRepo(t)
	repo := r.Repo()
	if has, err := repo.HasCommits(); err != nil || has {
		t.Fatalf("HasCommits on an empty repository = %v, %v", has, err)
	}
	r.Write("a.go", "package a\n")
	r.Git("add", "a.go")
	r.Write("b.go", "package b\n")
	got, err := repo.ChangedFiles("", "")
	if err != nil || len(got) != 2 || got[0].Status != "A" || got[1].Path != "b.go" {
		t.Fatalf("ChangedFiles on an empty repository = %+v, %v", got, err)
	}
	if _, err = repo.Resolve("HEAD"); !errors.Is(err, ErrUnknownRev) {
		t.Errorf("Resolve(HEAD) before the first commit: %v", err)
	}
	r.Commit("first")
	if has, err := repo.HasCommits(); err != nil || !has {
		t.Fatalf("HasCommits = %v, %v", has, err)
	}
	if sha, err := repo.Resolve("HEAD"); err != nil || len(sha) != 40 {
		t.Fatalf("Resolve = %q, %v", sha, err)
	}
	if _, err := NewGitRepo(t.TempDir()).HasCommits(); err == nil {
		t.Error("HasCommits outside a repository must be an error")
	}
}

// A clone made with --depth 1 is shallow and does not know the earlier commit.
//
// TRLC-LINKS: REQ-EMG-156
func TestShallowClone(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", "package a\n")
	r.Commit("one")
	r.Write("a.go", "package a // two\n")
	r.Commit("two")
	if shallow, err := r.Repo().IsShallow(); err != nil || shallow {
		t.Fatalf("IsShallow on the source = %v, %v", shallow, err)
	}
	dst := t.TempDir() + "/clone"
	gitIn(t, t.TempDir(), "clone", "-q", "--depth", "1", "file://"+r.Dir, dst)
	clone := NewGitRepo(dst)
	if shallow, err := clone.IsShallow(); err != nil || !shallow {
		t.Fatalf("IsShallow on the clone = %v, %v", shallow, err)
	}
	if _, err := clone.Resolve("HEAD"); err != nil {
		t.Fatalf("HEAD is available: %v", err)
	}
	if _, err := clone.Resolve("HEAD~1"); err == nil {
		t.Fatal("HEAD~1 must not resolve in a depth-1 clone")
	}
}

// Malformed name-status output is an error, not a silent skip.
//
// TRLC-LINKS: REQ-EMG-156
func TestParseNameStatus(t *testing.T) {
	got, err := parseNameStatus([]byte("M\x00a.go\x00R087\x00old.go\x00new.go\x00T\x00t.go\x00A\x00x/y.go\x00"))
	if err != nil {
		t.Fatal(err)
	}
	want := []ChangedFile{
		{Status: "M", Path: "a.go"},
		{Status: "R", Score: 87, OldPath: "old.go", Path: "new.go"},
		{Status: "M", Path: "t.go"},
		{Status: "A", Path: "x/y.go"},
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("parseNameStatus = %+v, want %+v", got, want)
	}
	for _, bad := range []string{"R100\x00only-one\x00", "M\x00../escape.go\x00", "M\x00/abs.go\x00", "\x00a.go\x00"} {
		if _, err := parseNameStatus([]byte(bad)); err == nil {
			t.Errorf("parseNameStatus(%q) accepted malformed output", bad)
		}
	}
}

// A git file name with a control character (a newline, an escape sequence) is
// refused as ErrInvalidPath, both when the caller supplies it and when git
// prints it, so it can never reach a log or a terminal (a newline would start a
// GitHub Actions workflow command). Ordinary names, including non-ASCII ones, pass.
//
// Mutation: dropping the control-character test of normalizeRepoPath fails.
//
// TRLC-LINKS: REQ-EMG-156
func TestControlCharactersInPathsAreRefused(t *testing.T) {
	if !HasControlChar("a\nb") || !HasControlChar("\x1b[31m") || !HasControlChar("a\x7f") || !HasControlChar("\u0085") {
		t.Error("HasControlChar missed a control character")
	}
	if HasControlChar("pkg/ünï/main.go") || HasControlChar("a b.go") {
		t.Error("HasControlChar refused an ordinary name")
	}
	if _, err := gitPath("evil\n::error::x.go"); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("gitPath = %v, want ErrInvalidPath", err)
	}
	if got, err := normalizeRepoPath("pkg/ünï.go"); err != nil || got != "pkg/ünï.go" {
		t.Errorf("a non-ASCII name = %q, %v", got, err)
	}
	r := NewFixtureRepo(t)
	r.Write("ok.go", "package p\n")
	r.Write("bad\nname.go", "package p\n")
	r.Commit("one")
	r.Write("ok.go", "package p\n// changed\n")
	if _, err := r.Repo().ChangedFiles("", ""); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("a listing with a newline in a name: %v, want ErrInvalidPath", err)
	}
	if _, err := r.Repo().ChangedFiles("HEAD", ""); err != nil {
		t.Errorf("a change that does not touch the odd name is fine: %v", err)
	}
	if _, err := r.Repo().ShowFile("HEAD", "bad\nname.go"); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("ShowFile = %v, want ErrInvalidPath", err)
	}
}

// The stderr of git is kept up to 64 KiB, the rest is dropped without an error,
// so a chatty child cannot grow the error message of a failed call.
//
// Mutation: an unbounded buffer fails the length check.
//
// TRLC-LINKS: REQ-EMG-156
func TestGitStderrIsCapped(t *testing.T) {
	b := &cappedBuffer{limit: maxGitStderr}
	chunk := []byte(strings.Repeat("x", 1000))
	for i := 0; i < 200; i++ {
		if n, err := b.Write(chunk); n != len(chunk) || err != nil {
			t.Fatalf("Write = %d, %v; a write past the cap must still succeed", n, err)
		}
	}
	if b.buf.Len() != maxGitStderr || maxGitStderr != 64<<10 {
		t.Errorf("kept %d bytes, want %d", b.buf.Len(), 64<<10)
	}
}

// Every call carries the configuration that switches off the repository's hooks,
// its filesystem monitor and signature display, and the diff calls refuse
// external diff and textconv drivers.
//
// TRLC-LINKS: REQ-EMG-156
func TestGitArgsDisableRepositoryHelpers(t *testing.T) {
	got := strings.Join(gitArgs([]string{"status"}), " ")
	for _, want := range []string{"core.hooksPath=/dev/null", "core.fsmonitor=false", "log.showsignature=false"} {
		if !strings.Contains(got, want) {
			t.Errorf("git arguments %q lack %s", got, want)
		}
	}
	src, err := os.ReadFile("git.go")
	if err != nil {
		t.Fatal(err)
	}
	if strings.Count(string(src), `"diff"`) != strings.Count(string(src), `"--no-ext-diff", "--no-textconv"`) {
		t.Error("a git diff call lacks --no-ext-diff and --no-textconv")
	}
}

// TrackedFiles lists what git tracks under a prefix, tells a directory outside
// any work tree apart from a failure, and ignores untracked files.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-159
func TestTrackedFiles(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", "package p\n")
	r.Commit("one")
	got, err := r.Repo().TrackedFiles(".engmod")
	if err != nil || len(got) != 0 {
		t.Fatalf("nothing tracked: %v, %v", got, err)
	}
	r.Write(".engmod/cache/health/x.json", "{}")
	if got, err = r.Repo().TrackedFiles(".engmod"); err != nil || len(got) != 0 {
		t.Fatalf("an untracked file is not tracked: %v, %v", got, err)
	}
	r.Commit("two")
	if got, err = r.Repo().TrackedFiles(".engmod"); err != nil || !reflect.DeepEqual(got, []string{".engmod/cache/health/x.json"}) {
		t.Fatalf("tracked = %v, %v", got, err)
	}
	if _, err = NewGitRepo(t.TempDir()).TrackedFiles(".engmod"); !errors.Is(err, ErrNotInWorkTree) {
		t.Errorf("outside a work tree: %v, want ErrNotInWorkTree", err)
	}
	if _, err = r.Repo().TrackedFiles("../x"); !errors.Is(err, ErrInvalidPath) {
		t.Errorf("an escaping prefix: %v, want ErrInvalidPath", err)
	}
}
