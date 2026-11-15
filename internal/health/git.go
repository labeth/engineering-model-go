// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The health-local git reader of the delta check (ADR-EMG-053). It calls the
// git binary with fixed arguments and no shell, reads files at a revision as
// raw bytes (never a checkout, so the working tree is not touched) and gives
// the child process an allow-listed environment. engws has similar helpers but
// is a top layer package and cannot be imported from here.

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

var (
	// ErrNotInRev reports that a path does not exist at a revision.
	ErrNotInRev = errors.New("path does not exist at the revision")
	// ErrInvalidRev reports a revision that is not safe to hand to git.
	ErrInvalidRev = errors.New("invalid git revision")
	// ErrUnknownRev reports a revision that git cannot resolve to a commit.
	ErrUnknownRev = errors.New("unknown git revision")
	// ErrInvalidPath reports a path that is not a clean repo-relative path, or
	// that holds a control character (a newline or an escape sequence in a file
	// name would reach logs and terminals).
	ErrInvalidPath = errors.New("invalid repo-relative path")
	// ErrNotInWorkTree reports that the directory is not inside a git work tree.
	ErrNotInWorkTree = errors.New("not inside a git work tree")
)

const (
	defaultGitTimeout = 30 * time.Second
	maxGitOutput      = 64 << 20
	// maxGitStderr bounds the stderr that is kept for an error message.
	maxGitStderr = 64 << 10
)

// GitRepo reads one git work tree. Dir is the directory git runs in; every
// path the methods take or return is relative to it, with forward slashes.
type GitRepo struct {
	Dir string
	// Timeout bounds each git call; zero means 30 seconds.
	Timeout time.Duration
}

// ChangedFile is one entry of a name-status listing.
type ChangedFile struct {
	// Status is A (added), M (modified or type change), D (deleted) or R
	// (renamed).
	Status string
	// Score is the similarity percentage of a rename, else 0.
	Score int
	// OldPath is the path before a rename; empty otherwise.
	OldPath string
	// Path is the path after the change (the deleted path for D).
	Path string
}

// NewGitRepo returns the reader for the work tree rooted at dir.
//
// TRLC-LINKS: REQ-EMG-156
func NewGitRepo(dir string) GitRepo { return GitRepo{Dir: dir} }

// gitEnvNames is the allow-list of the git child's environment; nothing else
// of the caller's environment (tokens, GIT_* overrides) reaches git. The user
// and system configuration are switched off (see gitEnv), so a repository
// cannot be steered by them either.
var gitEnvNames = []string{"PATH", "HOME", "LANG", "LC_ALL", "TMPDIR"}

// gitEnvNamesWindows are added on Windows, where git cannot start without
// them: the system directory, the command interpreter and the temp
// directories. Windows names its variables case-insensitively (Path, SystemRoot,
// Temp), so the names match without regard to case there.
var gitEnvNamesWindows = []string{"SYSTEMROOT", "COMSPEC", "TEMP", "TMP"}

// gitEnv builds the environment of a git child from the given process
// environment.
//
// TRLC-LINKS: REQ-EMG-156
func gitEnv(environ []string) []string { return gitEnvFor(runtime.GOOS, environ) }

// gitEnvFor is gitEnv for a named operating system; it is a pure function so
// that the Windows rules can be tested everywhere.
//
// TRLC-LINKS: REQ-EMG-156
func gitEnvFor(goos string, environ []string) []string {
	names := gitEnvNames
	windows := goos == "windows"
	if windows {
		names = append(append([]string{}, names...), gitEnvNamesWindows...)
	}
	env := make([]string, 0, len(names)+3)
	for _, name := range names {
		if kv, ok := lookupEnvEntry(environ, name, windows); ok {
			env = append(env, kv)
		}
	}
	return append(env, "GIT_OPTIONAL_LOCKS=0", "GIT_CONFIG_GLOBAL=/dev/null", "GIT_CONFIG_NOSYSTEM=1")
}

// lookupEnvEntry returns the first KEY=value entry of environ whose key is
// name, compared without regard to case when fold is set.
//
// TRLC-LINKS: REQ-EMG-156
func lookupEnvEntry(environ []string, name string, fold bool) (string, bool) {
	for _, kv := range environ {
		key, _, ok := strings.Cut(kv, "=")
		if ok && envKeyMatches(key, name, fold) {
			return kv, true
		}
	}
	return "", false
}

// envKeyMatches compares an environment key with a wanted name.
//
// TRLC-LINKS: REQ-EMG-156
func envKeyMatches(key, name string, fold bool) bool {
	return key == name || (fold && strings.EqualFold(key, name))
}

// limitedBuffer fails a write that would grow it past its limit.
type limitedBuffer struct {
	buf   bytes.Buffer
	limit int
}

// TRLC-LINKS: REQ-EMG-156
func (b *limitedBuffer) Write(p []byte) (int, error) {
	if b.buf.Len()+len(p) > b.limit {
		return 0, fmt.Errorf("git output exceeds %d bytes", b.limit)
	}
	return b.buf.Write(p)
}

// cappedBuffer keeps the first limit bytes written to it and drops the rest
// without an error, so a chatty child cannot grow it.
type cappedBuffer struct {
	buf   bytes.Buffer
	limit int
}

// TRLC-LINKS: REQ-EMG-156
func (b *cappedBuffer) Write(p []byte) (int, error) {
	if room := b.limit - b.buf.Len(); room > 0 {
		b.buf.Write(p[:min(room, len(p))])
	}
	return len(p), nil
}

// gitArgs prefixes the arguments of one call with the configuration that makes
// git ignore the hooks, the filesystem monitor and signature display of the
// repository.
//
// TRLC-LINKS: REQ-EMG-156
func gitArgs(args []string) []string {
	return append([]string{"-c", "log.showsignature=false", "-c", "core.fsmonitor=false", "-c", "core.hooksPath=/dev/null"}, args...)
}

// run executes git with args and returns the raw stdout and the exit code (-1
// when git did not run to an exit). stderr is part of the returned error.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) run(args ...string) ([]byte, int, error) {
	timeout := g.Timeout
	if timeout <= 0 {
		timeout = defaultGitTimeout
	}
	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()
	cmd := exec.CommandContext(ctx, "git", gitArgs(args)...)
	cmd.Dir = g.Dir
	cmd.Env = gitEnv(os.Environ())
	cmd.WaitDelay = 2 * time.Second
	stdout := &limitedBuffer{limit: maxGitOutput}
	stderr := &cappedBuffer{limit: maxGitStderr}
	cmd.Stdout = stdout
	cmd.Stderr = stderr
	err := cmd.Run()
	if err == nil {
		return stdout.buf.Bytes(), 0, nil
	}
	code := -1
	var exit *exec.ExitError
	if errors.As(err, &exit) {
		code = exit.ExitCode()
	}
	if ctx.Err() != nil {
		err = fmt.Errorf("timed out after %s", timeout)
	}
	return nil, code, fmt.Errorf("git %s: %w: %s", strings.Join(args, " "), err, strings.TrimSpace(stderr.buf.String()))
}

// checkRev refuses a revision that git could read as an option or that holds
// characters no revision needs.
//
// TRLC-LINKS: REQ-EMG-156
func checkRev(rev string) error {
	if rev == "" || strings.HasPrefix(rev, "-") || strings.ContainsAny(rev, ":\x00") {
		return fmt.Errorf("%w %q", ErrInvalidRev, rev)
	}
	for _, r := range rev {
		if r <= ' ' || r == 0x7f {
			return fmt.Errorf("%w %q", ErrInvalidRev, rev)
		}
	}
	return nil
}

// HasControlChar reports whether s holds a control character: below 0x20, 0x7f
// or a C1 control (U+0080 to U+009F).
//
// TRLC-LINKS: REQ-EMG-156
func HasControlChar(s string) bool {
	for _, r := range s {
		if r < 0x20 || (r >= 0x7f && r <= 0x9f) {
			return true
		}
	}
	return false
}

// normalizeRepoPath cleans a caller-supplied path and refuses an empty,
// absolute or parent-escaping one, and one with a control character.
//
// TRLC-LINKS: REQ-EMG-156
func normalizeRepoPath(p string) (string, error) {
	clean := path.Clean(filepath.ToSlash(p))
	if malformedPath(p) || escapingPath(clean) || absolutePath(p, clean) {
		return "", fmt.Errorf("%w %q", ErrInvalidPath, p)
	}
	return clean, nil
}

// malformedPath reports an empty path or one with a control character.
//
// TRLC-LINKS: REQ-EMG-156
func malformedPath(p string) bool { return p == "" || HasControlChar(p) }

// escapingPath reports a cleaned path that is the root or leaves it.
//
// TRLC-LINKS: REQ-EMG-156
func escapingPath(clean string) bool {
	return clean == "." || clean == ".." || strings.HasPrefix(clean, "../")
}

// absolutePath reports a path that is absolute in any notation or names a
// volume.
//
// TRLC-LINKS: REQ-EMG-156
func absolutePath(p, clean string) bool {
	return path.IsAbs(clean) || filepath.IsAbs(p) || filepath.VolumeName(p) != ""
}

// gitPath checks a path that git printed: it must already be clean.
//
// TRLC-LINKS: REQ-EMG-156
func gitPath(p string) (string, error) {
	clean, err := normalizeRepoPath(p)
	if err != nil || clean != p {
		return "", fmt.Errorf("%w %q from git", ErrInvalidPath, p)
	}
	return clean, nil
}

// Resolve returns the commit a revision names as a full object id. An
// unknown revision is ErrUnknownRev.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) Resolve(rev string) (string, error) {
	if err := checkRev(rev); err != nil {
		return "", err
	}
	out, code, err := g.run("rev-parse", "--verify", "--quiet", rev+"^{commit}")
	if err != nil {
		if code == 1 {
			return "", fmt.Errorf("%w %q", ErrUnknownRev, rev)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// MergeBase returns the best common ancestor of two revisions.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) MergeBase(a, b string) (string, error) {
	if err := checkRevs(a, b); err != nil {
		return "", err
	}
	out, code, err := g.run("merge-base", a, b)
	if err != nil {
		if code == 1 {
			return "", fmt.Errorf("%w: %q and %q have no common ancestor", ErrUnknownRev, a, b)
		}
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}

// HasCommits reports whether HEAD names a commit; a repository that was just
// initialised has none.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) HasCommits() (bool, error) {
	_, code, err := g.run("rev-parse", "--verify", "--quiet", "HEAD^{commit}")
	if err == nil {
		return true, nil
	}
	if code == 1 {
		return false, nil
	}
	return false, err
}

// IsShallow reports whether the repository is a shallow clone.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) IsShallow() (bool, error) {
	out, _, err := g.run("rev-parse", "--is-shallow-repository")
	if err != nil {
		return false, err
	}
	return strings.TrimSpace(string(out)) == "true", nil
}

// ShowFile returns the raw bytes of path at rev. It reads the blob itself
// (cat-file, so no textconv driver of the repository is run). A path that is
// not in rev is ErrNotInRev.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) ShowFile(rev, p string) ([]byte, error) {
	if err := checkRev(rev); err != nil {
		return nil, err
	}
	clean, err := normalizeRepoPath(p)
	if err != nil {
		return nil, err
	}
	out, _, err := g.run("cat-file", "blob", rev+":./"+clean)
	if err == nil {
		return out, nil
	}
	if _, rerr := g.Resolve(rev); rerr != nil {
		return nil, rerr
	}
	if _, code, _ := g.run("rev-parse", "--verify", "--quiet", rev+":./"+clean); code == 1 {
		return nil, fmt.Errorf("%w: %s at %s", ErrNotInRev, clean, rev)
	}
	return nil, err
}

// TrackedFiles lists the files git tracks under prefix (a path relative to
// Dir), sorted. A directory that is not inside a git work tree is
// ErrNotInWorkTree; any other failure is returned as it is, so a caller can
// tell "nothing tracked" from "could not ask".
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-159
func (g GitRepo) TrackedFiles(prefix string) ([]string, error) {
	clean, err := normalizeRepoPath(prefix)
	if err != nil {
		return nil, err
	}
	out, code, err := g.run("ls-files", "-z", "--", clean)
	if err != nil {
		return nil, notInWorkTree(code, err)
	}
	files, err := splitGitPaths(out)
	if err != nil {
		return nil, err
	}
	sort.Strings(files)
	return files, nil
}

// notInWorkTree maps the failure of git outside a work tree to
// ErrNotInWorkTree and leaves any other failure as it is.
//
// TRLC-LINKS: REQ-EMG-156
func notInWorkTree(code int, err error) error {
	if code == 128 && strings.Contains(err.Error(), "not a git repository") {
		return ErrNotInWorkTree
	}
	return err
}

// splitGitPaths reads the clean paths of a NUL-separated git listing.
//
// TRLC-LINKS: REQ-EMG-156
func splitGitPaths(out []byte) ([]string, error) {
	var paths []string
	for _, raw := range bytes.Split(out, []byte{0}) {
		if len(raw) == 0 {
			continue
		}
		p, err := gitPath(string(raw))
		if err != nil {
			return nil, err
		}
		paths = append(paths, p)
	}
	return paths, nil
}

// ChangedFiles lists what changed. With head set it is the diff of
// base...head (head against the merge base, rename detection on). With head
// empty it is base against the working tree: tracked changes plus untracked
// files that are not ignored, the latter as A. With base empty (a repository
// without commits) and head empty it is every tracked and untracked file as
// A. The list is sorted by path and holds only clean repo-relative paths.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) ChangedFiles(base, head string) ([]ChangedFile, error) {
	files, err := g.collectChanged(base, head)
	if err != nil {
		return nil, err
	}
	sort.Slice(files, func(i, j int) bool {
		if files[i].Path != files[j].Path {
			return files[i].Path < files[j].Path
		}
		return files[i].OldPath < files[j].OldPath
	})
	return files, nil
}

// collectChanged picks the listing ChangedFiles describes for the two
// revisions, before it is sorted.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) collectChanged(base, head string) ([]ChangedFile, error) {
	switch {
	case base == "" && head == "":
		return g.changedAll()
	case base == "":
		return nil, fmt.Errorf("%w: a head revision needs a base", ErrInvalidRev)
	case head != "":
		return g.changedRange(base, head)
	}
	return g.changedWorkTree(base)
}

// changedAll lists every tracked and untracked file as A.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) changedAll() ([]ChangedFile, error) {
	out, _, err := g.run("ls-files", "-z", "--cached", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return addedFiles(out, nil)
}

// changedRange lists the diff of base...head.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) changedRange(base, head string) ([]ChangedFile, error) {
	if err := checkRevs(base, head); err != nil {
		return nil, err
	}
	return g.nameStatus(base + "..." + head)
}

// changedWorkTree lists base against the working tree: the tracked changes
// plus the untracked files that are not ignored.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) changedWorkTree(base string) ([]ChangedFile, error) {
	if err := checkRev(base); err != nil {
		return nil, err
	}
	files, err := g.nameStatus(base)
	if err != nil {
		return nil, err
	}
	untracked, _, err := g.run("ls-files", "-z", "--others", "--exclude-standard")
	if err != nil {
		return nil, err
	}
	return addedFiles(untracked, files)
}

// nameStatus runs git diff --name-status for one revision argument and parses
// its output.
//
// TRLC-LINKS: REQ-EMG-156
func (g GitRepo) nameStatus(revArg string) ([]ChangedFile, error) {
	out, _, err := g.run("diff", "-M", "--name-status", "-z", "--no-ext-diff", "--no-textconv", "--no-color", "--relative", revArg, "--")
	if err != nil {
		return nil, err
	}
	return parseNameStatus(out)
}

// checkRevs checks each revision in turn.
//
// TRLC-LINKS: REQ-EMG-156
func checkRevs(revs ...string) error {
	for _, rev := range revs {
		if err := checkRev(rev); err != nil {
			return err
		}
	}
	return nil
}

// addedFiles appends one A entry per NUL-separated path that is not already
// listed.
//
// TRLC-LINKS: REQ-EMG-156
func addedFiles(out []byte, files []ChangedFile) ([]ChangedFile, error) {
	paths, err := splitGitPaths(out)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	for _, f := range files {
		seen[f.Path] = true
	}
	for _, p := range paths {
		if !seen[p] {
			seen[p] = true
			files = append(files, ChangedFile{Status: "A", Path: p})
		}
	}
	return files, nil
}

// parseNameStatus reads the output of git diff --name-status -z.
//
// TRLC-LINKS: REQ-EMG-156
func parseNameStatus(out []byte) ([]ChangedFile, error) {
	fields := strings.Split(string(out), "\x00")
	if n := len(fields); n > 0 && fields[n-1] == "" {
		fields = fields[:n-1]
	}
	var files []ChangedFile
	for i := 0; i < len(fields); {
		cf, used, err := parseStatusEntry(fields[i:])
		if err != nil {
			return nil, err
		}
		files = append(files, cf)
		i += used
	}
	return files, nil
}

// parseStatusEntry reads the first entry of the fields of a name-status
// listing and returns it with the number of fields it used.
//
// TRLC-LINKS: REQ-EMG-156
func parseStatusEntry(fields []string) (ChangedFile, int, error) {
	status := fields[0]
	if status == "" {
		return ChangedFile{}, 0, fmt.Errorf("git diff: empty status field")
	}
	kind := status[:1]
	paths := 1
	if kind == "R" || kind == "C" {
		paths = 2
	}
	if 1+paths > len(fields) {
		return ChangedFile{}, 0, fmt.Errorf("git diff: truncated entry for status %q", status)
	}
	cf := ChangedFile{Status: kind}
	if err := readEntryPaths(&cf, status, fields[1:1+paths]); err != nil {
		return ChangedFile{}, 0, err
	}
	normalizeStatus(&cf)
	return cf, 1 + paths, nil
}

// readEntryPaths fills the paths (and the similarity score of a rename or
// copy) of an entry from its path fields.
//
// TRLC-LINKS: REQ-EMG-156
func readEntryPaths(cf *ChangedFile, status string, paths []string) error {
	var err error
	if len(paths) == 2 {
		if cf.OldPath, err = gitPath(paths[0]); err != nil {
			return err
		}
		cf.Score, _ = strconv.Atoi(status[1:])
	}
	cf.Path, err = gitPath(paths[len(paths)-1])
	return err
}

// normalizeStatus reduces a git status letter to A, D, M or R.
//
// TRLC-LINKS: REQ-EMG-156
func normalizeStatus(cf *ChangedFile) {
	switch cf.Status {
	case "A", "D", "M", "R":
	case "C":
		cf.Status, cf.OldPath = "A", ""
	default: // T (type change), U (unmerged), X: the file is there and differs
		cf.Status = "M"
	}
}
