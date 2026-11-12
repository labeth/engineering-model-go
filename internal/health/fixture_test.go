// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// FixtureRepo is a throw-away git repository in a temp directory. It is
// exported so that the external test package can use it; it only exists in
// test builds.
type FixtureRepo struct {
	T   *testing.T
	Dir string
}

// NewFixtureRepo runs git init in a new temp directory. The test must not be
// parallel: it points HOME at an empty directory so that no configuration of
// the machine reaches git.
//
// TRLC-LINKS: REQ-EMG-156
func NewFixtureRepo(t *testing.T) *FixtureRepo {
	t.Helper()
	if _, err := exec.LookPath("git"); err != nil {
		t.Skip("git is not installed")
	}
	t.Setenv("HOME", t.TempDir())
	r := &FixtureRepo{T: t, Dir: t.TempDir()}
	r.Git("init", "-q", "-b", "main")
	return r
}

// Git runs git in the repository with a fixed identity and no machine
// configuration, and returns its stdout.
//
// TRLC-LINKS: REQ-EMG-156
func (r *FixtureRepo) Git(args ...string) string {
	return gitIn(r.T, r.Dir, args...)
}

// gitIn runs git in dir for a test.
//
// TRLC-LINKS: REQ-EMG-156
func gitIn(t *testing.T, dir string, args ...string) string {
	t.Helper()
	full := append([]string{
		"-c", "user.name=Test", "-c", "user.email=test@example.com", "-c", "commit.gpgsign=false",
		"-c", "protocol.file.allow=always", "-c", "init.defaultBranch=main",
	}, args...)
	cmd := exec.Command("git", full...)
	cmd.Dir = dir
	cmd.Env = []string{
		"PATH=" + os.Getenv("PATH"), "HOME=" + os.Getenv("HOME"), "GIT_CONFIG_GLOBAL=/dev/null",
		"GIT_CONFIG_NOSYSTEM=1", "GIT_AUTHOR_DATE=2026-01-01T00:00:00Z", "GIT_COMMITTER_DATE=2026-01-01T00:00:00Z",
	}
	var stdout, stderr bytes.Buffer
	cmd.Stdout, cmd.Stderr = &stdout, &stderr
	if err := cmd.Run(); err != nil {
		t.Fatalf("git %s: %v: %s", strings.Join(args, " "), err, stderr.String())
	}
	return stdout.String()
}

// Write writes a file (creating directories) without staging it.
//
// TRLC-LINKS: REQ-EMG-156
func (r *FixtureRepo) Write(rel, content string) {
	r.T.Helper()
	full := filepath.Join(r.Dir, filepath.FromSlash(rel))
	if err := os.MkdirAll(filepath.Dir(full), 0o755); err != nil {
		r.T.Fatal(err)
	}
	if err := os.WriteFile(full, []byte(content), 0o644); err != nil {
		r.T.Fatal(err)
	}
}

// Remove deletes a file from the working tree.
//
// TRLC-LINKS: REQ-EMG-156
func (r *FixtureRepo) Remove(rel string) {
	r.T.Helper()
	if err := os.Remove(filepath.Join(r.Dir, filepath.FromSlash(rel))); err != nil {
		r.T.Fatal(err)
	}
}

// Commit stages everything, commits and returns the new commit id.
//
// TRLC-LINKS: REQ-EMG-156
func (r *FixtureRepo) Commit(msg string) string {
	r.T.Helper()
	r.Git("add", "-A")
	r.Git("commit", "-q", "--allow-empty", "-m", msg)
	return strings.TrimSpace(r.Git("rev-parse", "HEAD"))
}

// Repo is the reader for the fixture.
//
// TRLC-LINKS: REQ-EMG-156
func (r *FixtureRepo) Repo() GitRepo { return NewGitRepo(r.Dir) }
