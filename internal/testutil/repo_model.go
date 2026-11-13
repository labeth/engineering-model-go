// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package testutil

import (
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// RepoRoot returns the repository root: the directory holding go.mod at or
// above the current working directory. It does not change the directory.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func RepoRoot(t testing.TB) string {
	t.Helper()
	dir, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	for {
		if _, err := os.Stat(filepath.Join(dir, "go.mod")); err == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			t.Fatalf("no go.mod above the working directory")
		}
		dir = parent
	}
}

// repoModelTrees are the repository directories the test copy carries next
// to the root files: the model documents and the source trees the validator
// scans to resolve inferred CODE identifiers (control verification refs name
// test declarations). Large unrelated trees (.git, .engmod, examples,
// generated) are left out.
var repoModelTrees = []string{"model", "cmd", "internal", "scripts", ".github/workflows"}

// acceptedRepoDecisions are the repository decisions the test copy must carry
// as accepted: AcceptedRepoModel rewrites a proposed status and keeps an accepted one.
var acceptedRepoDecisions = []string{"DEC-TIER-OPERATIONAL", "DEC-HEALTH-BASELINE"}

// asOfEnv is the environment variable the validator reads as the as-of date.
const asOfEnv = "ENGMOD_AS_OF"

// repoModelAsOf is the as-of date of the test copy: before the due dates of
// the repository's POA&M items, so none is overdue.
const repoModelAsOf = "2026-01-01"

// AcceptedRepoModel copies the repository's own engmod.yml and model/
// directory (which holds every document the manifest lists, including the
// health baseline file) into a temporary directory and returns the path of
// the copied manifest. Besides the manifest and model/ it carries the root
// files and the source trees the validator scans (repoModelTrees). In the copy, the status of DEC-TIER-OPERATIONAL and
// DEC-HEALTH-BASELINE is rewritten to accepted, and the test's as-of date
// (ENGMOD_AS_OF, restored when the test ends) is set to 2026-01-01 so the
// repository's POA&M items are not overdue.
//
// This is a TEST-ONLY copy so exporter and CLI tests, which only need a
// rich, valid model, do not depend on the owner accepting those decisions or
// renewing the POA&M items. It is never used by production code and never
// writes into the repository. The repository's own gate state is asserted
// on purpose in internal/integration TestRepositoryModelGate. Tests that
// need source files point their code root at RepoRoot, not at the copy.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func AcceptedRepoModel(t testing.TB) string {
	t.Helper()
	root := RepoRoot(t)
	dst := t.TempDir()
	copyRootFiles(t, root, dst)
	paths := repoPaths{t: t, root: root, dst: dst}
	for _, dir := range repoModelTrees {
		copyTree(t, paths, dir)
	}
	acceptDecisions(t, filepath.Join(dst, "model", "decisions.yml"))
	t.Setenv(asOfEnv, repoModelAsOf)
	return filepath.Join(dst, "engmod.yml")
}

// copyRootFiles copies the regular files of the repository root.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func copyRootFiles(t testing.TB, root, dst string) {
	t.Helper()
	entries, err := os.ReadDir(root)
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		if e.Type().IsRegular() {
			copyRepoFile(t, filepath.Join(root, e.Name()), filepath.Join(dst, e.Name()))
		}
	}
}

// repoPaths names the repository root and the test copy it is copied into.
type repoPaths struct {
	t         testing.TB
	root, dst string
}

// copyTree copies one repository directory below the root into the copy.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func copyTree(t testing.TB, paths repoPaths, dir string) {
	t.Helper()
	src := filepath.Join(paths.root, filepath.FromSlash(dir))
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		return copyTreeEntry(paths, path, d)
	})
	if err != nil {
		t.Fatal(err)
	}
}

// copyTreeEntry copies one directory or file found while walking a tree.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func copyTreeEntry(paths repoPaths, path string, d fs.DirEntry) error {
	rel, err := filepath.Rel(paths.root, path)
	if err != nil {
		return err
	}
	if d.IsDir() {
		return os.MkdirAll(filepath.Join(paths.dst, rel), 0o755)
	}
	if d.Type().IsRegular() {
		copyRepoFile(paths.t, path, filepath.Join(paths.dst, rel))
	}
	return nil
}

// acceptDecisions rewrites the status of acceptedRepoDecisions in the copied
// decisions document.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func acceptDecisions(t testing.TB, decisions string) {
	t.Helper()
	data, err := os.ReadFile(decisions)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	for _, id := range acceptedRepoDecisions {
		text = acceptDecision(t, text, id)
	}
	if err := os.WriteFile(decisions, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// acceptDecision rewrites the first "status: proposed" line after the
// "- id: <id>" line of a decisions document to "status: accepted". A decision
// that is already accepted is left as it is; a decision that is missing, or
// has another status, fails the test.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func acceptDecision(t testing.TB, text, id string) string {
	t.Helper()
	lines := strings.Split(text, "\n")
	inDecision := false
	for i, line := range lines {
		switch {
		case strings.HasPrefix(line, "- id: "):
			inDecision = strings.TrimSpace(strings.TrimPrefix(line, "- id: ")) == id
		case inDecision && strings.TrimSpace(line) == "status: accepted":
			return text
		case inDecision && strings.TrimSpace(line) == "status: proposed":
			lines[i] = strings.Replace(line, "proposed", "accepted", 1)
			return strings.Join(lines, "\n")
		}
	}
	t.Fatalf("decision %s with status proposed or accepted not found", id)
	return text
}

// copyRepoFile copies one file, creating its parent directory.
// TRLC-LINKS: REQ-EMG-124
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func copyRepoFile(t testing.TB, from, to string) {
	t.Helper()
	data, err := os.ReadFile(from)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(to, data, 0o644); err != nil {
		t.Fatal(err)
	}
}
