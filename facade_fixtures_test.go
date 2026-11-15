// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package engmodel_test

import (
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// Fixtures of the facade tests: a copy of an example model in a temporary directory, optionally as a git repository.
// The facade tests call the root package only, so a helper here never reaches into an internal package.

// childA is the example model the facade tests copy; its second capability and four requirements are known to the
// assertions.
const childA = "examples/requirement-links/repos/child-a"

// isolatedModel keeps the process environment from changing what a model resolves: no workspace file and no lock mode.
func isolatedModel(t *testing.T) {
	t.Helper()
	t.Setenv(engmodel.WorkspaceEnv, engmodel.WorkspaceOff)
	t.Setenv("ENGMOD_LOCK", "")
	t.Setenv(engmodel.ModuleCacheEnv, "")
}

// copyModel copies src into dst without generated output, module caches and locks.
func copyModel(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		if d.IsDir() {
			return copyDir(d.Name(), filepath.Join(dst, rel))
		}
		return copyFile(path, filepath.Join(dst, rel))
	})
	if err != nil {
		t.Fatal(err)
	}
}

// copyDir creates a directory of the copy, or skips the directory when it holds generated output or a module cache.
func copyDir(name, dst string) error {
	if name == "generated" || name == ".engmod" {
		return filepath.SkipDir
	}
	return os.MkdirAll(dst, 0o755)
}

// copyFile copies one file of a model, except a lock file.
func copyFile(src, dst string) error {
	if strings.HasPrefix(filepath.Base(src), "engmod.lock") {
		return nil
	}
	data, err := os.ReadFile(src)
	if err != nil {
		return err
	}
	return os.WriteFile(dst, data, 0o644)
}

// writeFile writes content to path, creating its directory.
func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// inFile is a file of a fixture that a test edits.
type inFile string

// replace replaces the first occurrence of old with new in the file, which must contain it.
func (f inFile) replace(t *testing.T, old, new string) {
	t.Helper()
	data, err := os.ReadFile(string(f))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), old) {
		t.Fatalf("%s does not contain %q", f, old)
	}
	writeFile(t, string(f), strings.Replace(string(data), old, new, 1))
}

// expect fails the test when ok is false; what is a format string and its arguments.
func expect(t *testing.T, ok bool, what ...any) {
	t.Helper()
	if !ok {
		t.Errorf(what[0].(string), what[1:]...)
	}
}

// check is the test an equality is checked in.
type check struct{ t *testing.T }

// equal fails the test when got is not deeply equal to want.
func (c check) equal(what string, got, want any) {
	c.t.Helper()
	if !reflect.DeepEqual(got, want) {
		c.t.Errorf("%s: got %+v, want %+v", what, got, want)
	}
}

// git runs git in dir with a fixed identity and fails the test on error.
func git(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir, "-c", "user.name=t", "-c", "user.email=t@example.com", "-c", "commit.gpgsign=false", "-c", "tag.gpgsign=false"}, args...)...)
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// commitAll commits everything in dir and tags the commit.
func commitAll(t *testing.T, dir, tag string) {
	t.Helper()
	git(t, dir, "add", "-A")
	git(t, dir, "commit", "-q", "-m", tag)
	git(t, dir, "tag", tag)
}

// childARepo is the child-a model as its own git repository, committed and tagged v1.0.0.
func childARepo(t *testing.T) string {
	t.Helper()
	isolatedModel(t)
	dir := filepath.Join(t.TempDir(), "child-a")
	copyModel(t, childA, dir)
	git(t, dir, "init", "-q")
	commitAll(t, dir, "v1.0.0")
	return dir
}

// childAReleased is child-a at v1.0.0 and at v1.1.0: the last requirement REQ-A-004 removed, REQ-A-002 with another
// verification criterion, REQ-A-005 added, and a source file and a test file that link REQ-A-001.
func childAReleased(t *testing.T) string {
	t.Helper()
	dir := childARepo(t)
	reqs := filepath.Join(dir, "model", "requirements.yml")
	data, err := os.ReadFile(reqs)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	last := strings.Index(text, "- id: REQ-A-004")
	if last < 0 {
		t.Fatal("the fixture has no REQ-A-004")
	}
	writeFile(t, reqs, text[:last]+"- id: REQ-A-005\n  text: The sentence encoder shall append a checksum to each position sentence.\n  verificationMethods: [test]\n  verificationCriteria: The checksum matches.\n  appliesTo: [FU-ENCODER]\n  realizes: [CAP-A-2]\n")
	inFile(reqs).replace(t, "verificationCriteria: A published sentence carries latitude and longitude.", "verificationCriteria: A published sentence carries latitude and longitude, both in degrees.")
	writeFile(t, filepath.Join(dir, "src", "encoder.go"), "// ENGMODEL-OWNER-UNIT: FU-ENCODER\npackage src\n\n// TRLC-LINKS: REQ-A-001\nfunc Publish() {}\n")
	writeFile(t, filepath.Join(dir, "src", "encoder_test.go"), "// ENGMODEL-OWNER-UNIT: FU-ENCODER\npackage src\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-A-001\nfunc TestPublish(t *testing.T) {}\n")
	commitAll(t, dir, "v1.1.0")
	return dir
}

// ids lists the IDs of entity changes.
func ids(list []engmodel.EntityChange) []string {
	out := []string{}
	for _, c := range list {
		out = append(out, c.ID)
	}
	return out
}
