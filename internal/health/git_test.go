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
		wantErrorIs(t, checkRev(rev), ErrInvalidRev)
	}
	for _, rev := range []string{"HEAD", "HEAD~1", "HEAD^", "main", "origin/main", "v1.0", "abc123", "main@{1}"} {
		wantEqual(t, checkRev(rev), nil)
	}
	for _, p := range []string{"", ".", "..", "../x", "a/../../x", "/etc/passwd", "a\x00b", "a\nb.go", "::error::x\n.go", "a\x1b[2Jb.go", "a\x7fb", "a\u009bb"} {
		_, err := normalizeRepoPath(p)
		wantErrorIs(t, err, ErrInvalidPath)
	}
	got, err := normalizeRepoPath("./a//b/../c.go")
	wantEqual(t, [2]any{got, err}, [2]any{"a/c.go", nil})
	_, err = gitPath("a/./b")
	wantEqual(t, err != nil, true) // an unclean path from git is refused
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
	mustEqual(t, [2]any{string(got), err}, [2]any{raw, nil}) // the raw bytes
	got, err = repo.ShowFile(sha, "./pkg//a.go")
	mustEqual(t, [2]any{string(got), err}, [2]any{raw, nil})
	for _, c := range []struct {
		rev, path string
		want      error
	}{
		{"HEAD", "pkg/missing.go", ErrNotInRev},
		{"nosuchbranch", "pkg/a.go", ErrUnknownRev},
		{"--output=x", "pkg/a.go", ErrInvalidRev},
		{"HEAD", "../outside", ErrInvalidPath},
	} {
		_, err = repo.ShowFile(c.rev, c.path)
		wantErrorIs(t, err, c.want)
	}
	_, err = repo.ShowFile("HEAD", "pkg")
	wantEqual(t, [2]bool{err != nil, errors.Is(err, ErrNotInRev)}, [2]bool{true, false}) // a directory is not a file
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
	mustEqual(t, os.MkdirAll(r.Dir+"/dir", 0o755), nil)
	r.Git("mv", "old name.go", "dir/new name.go")
	r.Write("dir/new name.go", body+"one more line\n")
	r.Write("staged.go", "package staged\n")
	r.Git("add", "staged.go")
	r.Write("new.go", "package n\n")
	r.Write("ignored.go", "package ignored\n")

	got, err := r.Repo().ChangedFiles("HEAD", "")
	mustEqual(t, err, nil)
	want := []ChangedFile{
		{Status: "R", Score: 95, OldPath: "old name.go", Path: "dir/new name.go"},
		{Status: "D", Path: "gone.go"},
		{Status: "M", Path: "mod.go"},
		{Status: "A", Path: "new.go"},
		{Status: "A", Path: "staged.go"},
	}
	mustEqual(t, len(got), len(want))
	for i := range want {
		wantEqual(t, [3]string{got[i].Status, got[i].Path, got[i].OldPath}, [3]string{want[i].Status, want[i].Path, want[i].OldPath})
	}
	wantEqual(t, got[0].Score >= 50 && got[0].Score <= 100, true) // the rename score is in range
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
	mustEqual(t, [2]any{err, len(got)}, [2]any{nil, 2})
	wantEqual(t, [4]string{got[0].Path, got[0].Status, got[1].Path, got[1].Status}, [4]string{"a.go", "M", "feature.go", "A"})
	mb, err := repo.MergeBase("main", "feature")
	mustEqual(t, [2]any{mb, err}, [2]any{fork, nil})
	_, err = repo.ChangedFiles("main", "--bad")
	wantErrorIs(t, err, ErrInvalidRev) // an option-like head
	_, err = repo.ChangedFiles("", "feature")
	wantErrorIs(t, err, ErrInvalidRev) // a head without a base
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
	has, err := repo.HasCommits()
	mustEqual(t, [2]any{has, err}, [2]any{false, nil})
	r.Write("a.go", "package a\n")
	r.Git("add", "a.go")
	r.Write("b.go", "package b\n")
	got, err := repo.ChangedFiles("", "")
	mustEqual(t, [2]any{err, len(got)}, [2]any{nil, 2})
	wantEqual(t, [2]string{got[0].Status, got[1].Path}, [2]string{"A", "b.go"})
	_, err = repo.Resolve("HEAD")
	wantErrorIs(t, err, ErrUnknownRev) // before the first commit
	r.Commit("first")
	has, err = repo.HasCommits()
	mustEqual(t, [2]any{has, err}, [2]any{true, nil})
	sha, err := repo.Resolve("HEAD")
	mustEqual(t, [2]any{len(sha), err}, [2]any{40, nil})
	_, err = NewGitRepo(t.TempDir()).HasCommits()
	wantEqual(t, err != nil, true) // outside a repository
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
	for _, s := range []string{"a\nb", "\x1b[31m", "a\x7f", "\u0085"} {
		wantEqual(t, [2]any{s, HasControlChar(s)}, [2]any{s, true})
	}
	for _, s := range []string{"pkg/ünï/main.go", "a b.go"} {
		wantEqual(t, [2]any{s, HasControlChar(s)}, [2]any{s, false})
	}
	_, err := gitPath("evil\n::error::x.go")
	wantErrorIs(t, err, ErrInvalidPath)
	got, err := normalizeRepoPath("pkg/ünï.go")
	wantEqual(t, [2]any{got, err}, [2]any{"pkg/ünï.go", nil}) // a non-ASCII name
	r := NewFixtureRepo(t)
	r.Write("ok.go", "package p\n")
	r.Write("bad\nname.go", "package p\n")
	r.Commit("one")
	r.Write("ok.go", "package p\n// changed\n")
	_, err = r.Repo().ChangedFiles("", "")
	wantErrorIs(t, err, ErrInvalidPath) // a listing with a newline in a name
	_, err = r.Repo().ChangedFiles("HEAD", "")
	wantEqual(t, err, nil) // a change that does not touch the odd name is fine
	_, err = r.Repo().ShowFile("HEAD", "bad\nname.go")
	wantErrorIs(t, err, ErrInvalidPath)
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
	mustEqual(t, [2]any{len(got), err}, [2]any{0, nil}) // nothing tracked
	r.Write(".engmod/cache/health/x.json", "{}")
	got, err = r.Repo().TrackedFiles(".engmod")
	mustEqual(t, [2]any{len(got), err}, [2]any{0, nil}) // an untracked file is not tracked
	r.Commit("two")
	got, err = r.Repo().TrackedFiles(".engmod")
	mustEqual(t, err, nil)
	wantDeepEqual(t, got, []string{".engmod/cache/health/x.json"})
	_, err = NewGitRepo(t.TempDir()).TrackedFiles(".engmod")
	wantErrorIs(t, err, ErrNotInWorkTree)
	_, err = r.Repo().TrackedFiles("../x")
	wantErrorIs(t, err, ErrInvalidPath)
}
