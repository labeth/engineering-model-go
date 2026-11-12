// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package main

import (
	"bytes"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

const (
	fixtureDir = "../../internal/export/quint/testdata/payments"
	goldenFile = "../../internal/export/quint/testdata/golden/sm_payments_auth.qnt"
)

func runCLI(args ...string) (int, string, string) {
	var stdout, stderr bytes.Buffer
	code := run(args, &stdout, &stderr)
	return code, stdout.String(), stderr.String()
}

// copyFixture copies the fixture model into a temporary directory and returns
// the path of its engmod.yml.
func copyFixture(t *testing.T) string {
	t.Helper()
	dst := t.TempDir()
	err := filepath.WalkDir(fixtureDir, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(fixtureDir, path)
		target := filepath.Join(dst, rel)
		if entry.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		return os.WriteFile(target, data, 0o644)
	})
	if err != nil {
		t.Fatal(err)
	}
	return filepath.Join(dst, "engmod.yml")
}

func mustRead(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// TRLC-LINKS: REQ-EMG-142
func TestRunDispatch(t *testing.T) {
	code, _, stderr := runCLI()
	if code != 2 || !strings.Contains(stderr, "usage: engquint <command>") || !strings.Contains(stderr, "generate") {
		t.Fatalf("no command: %d %q", code, stderr)
	}
	code, _, stderr = runCLI("frobnicate")
	if code != 2 || !strings.Contains(stderr, `unknown command "frobnicate"`) {
		t.Fatalf("unknown command: %d %q", code, stderr)
	}
	code, stdout, _ := runCLI("--help")
	if code != 0 || !strings.Contains(stdout, "generate") {
		t.Fatalf("help: %d %q", code, stdout)
	}
}

// A later subcommand is one more entry in the commands table.
//
// TRLC-LINKS: REQ-EMG-142
func TestRunDispatchesRegisteredCommands(t *testing.T) {
	commands["probe"] = command{summary: "test probe", run: func(args []string, stdout, stderr io.Writer) int {
		io.WriteString(stdout, strings.Join(args, ","))
		return 7
	}}
	defer delete(commands, "probe")
	code, stdout, _ := runCLI("probe", "a", "--b")
	if code != 7 || stdout != "a,--b" {
		t.Fatalf("got %d %q", code, stdout)
	}
	if _, stdout, _ := runCLI("help"); !strings.Contains(stdout, "probe") || !strings.Contains(stdout, "test probe") {
		t.Fatalf("usage does not list the new command: %q", stdout)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestGenerateUsageErrors(t *testing.T) {
	for _, args := range [][]string{
		{"generate"},
		{"generate", "--bogus"},
		{"generate", "--model", "engmod.yml", "extra"},
	} {
		if code, _, _ := runCLI(args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code, _, _ := runCLI("generate", "-h"); code != 0 {
		t.Errorf("generate -h: exit %d, want 0", code)
	}
	code, _, stderr := runCLI("generate", "--model", filepath.Join(t.TempDir(), "engmod.yml"))
	if code != 2 || !strings.Contains(stderr, "error loading model") {
		t.Errorf("unreadable model: exit %d %q", code, stderr)
	}
}

// TRLC-LINKS: REQ-EMG-141, REQ-EMG-142, REQ-EMG-149
func TestGenerateWritesModuleNextToModel(t *testing.T) {
	manifest := copyFixture(t)
	code, stdout, stderr := runCLI("generate", "--model", manifest)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	path := filepath.Join(filepath.Dir(manifest), "generated", "quint", "sm_payments_auth.qnt")
	if got, want := mustRead(t, path), mustRead(t, goldenFile); got != want {
		t.Fatalf("written module differs from the golden file:\n%s", got)
	}
	if want := "wrote " + path + "\n"; stdout != want {
		t.Fatalf("stdout %q, want %q", stdout, want)
	}
}

// TRLC-LINKS: REQ-EMG-149
func TestGenerateIsIdempotentAndRewritesOnlyChanges(t *testing.T) {
	manifest := copyFixture(t)
	out := filepath.Join(t.TempDir(), "q")
	if code, _, stderr := runCLI("generate", "--model", manifest, "--out", out); code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	path := filepath.Join(out, "sm_payments_auth.qnt")
	old := time.Date(2020, 1, 2, 3, 4, 5, 0, time.UTC)
	if err := os.Chtimes(path, old, old); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runCLI("generate", "--model", manifest, "--out", out)
	if code != 0 || stdout != "unchanged "+path+"\n" {
		t.Fatalf("second run: %d %q", code, stdout)
	}
	if info, err := os.Stat(path); err != nil || !info.ModTime().Equal(old) {
		t.Fatalf("an unchanged module was rewritten: %v %v", info, err)
	}

	// A model change rewrites the module (the drift the freshness gate sees).
	behavior := filepath.Join(filepath.Dir(manifest), "model", "behavior.yml")
	changed := strings.Replace(mustRead(t, behavior), "guard: attempts < 3\n      effect", "guard: attempts < 2\n      effect", 1)
	if err := os.WriteFile(behavior, []byte(changed), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ = runCLI("generate", "--model", manifest, "--out", out)
	if code != 0 || stdout != "wrote "+path+"\n" || !strings.Contains(mustRead(t, path), "v_attempts < 2,") {
		t.Fatalf("after change: %d %q", code, stdout)
	}
}

// A symlink at the module path is refused, not written through, and nothing
// else is removed or changed.
//
// TRLC-LINKS: REQ-EMG-149
func TestGenerateRefusesToWriteThroughASymlink(t *testing.T) {
	manifest := copyFixture(t)
	out := filepath.Join(t.TempDir(), "q")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	victim := filepath.Join(t.TempDir(), "victim.txt")
	if err := os.WriteFile(victim, []byte("precious"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(victim, filepath.Join(out, "sm_payments_auth.qnt")); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	code, _, stderr := runCLI("generate", "--model", manifest, "--out", out)
	if code != 1 || !strings.Contains(stderr, "refusing to write") || !strings.Contains(stderr, "symlink") {
		t.Fatalf("exit %d %q", code, stderr)
	}
	if mustRead(t, victim) != "precious" {
		t.Fatal("the module was written through the symlink")
	}
}

// TRLC-LINKS: REQ-EMG-149
func TestGenerateRemovesOnlyStaleGeneratedModules(t *testing.T) {
	manifest := copyFixture(t)
	out := filepath.Join(t.TempDir(), "q")
	if err := os.MkdirAll(out, 0o755); err != nil {
		t.Fatal(err)
	}
	stale := filepath.Join(out, "sm_gone.qnt")
	handWritten := filepath.Join(out, "notes.qnt")
	other := filepath.Join(out, "readme.txt")
	for path, content := range map[string]string{
		stale:       "// Generated by engineering-model-go from SM-GONE; do not edit.\nmodule sm_gone {}\n",
		handWritten: "// my own spec\nmodule notes {}\n",
		other:       "// Generated by engineering-model-go from SM-GONE; do not edit.\n",
	} {
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	code, stdout, stderr := runCLI("generate", "--model", manifest, "--out", out)
	if code != 0 {
		t.Fatalf("exit %d: %s", code, stderr)
	}
	if !strings.Contains(stdout, "removed "+stale) {
		t.Fatalf("stdout %q", stdout)
	}
	if _, err := os.Stat(stale); !os.IsNotExist(err) {
		t.Error("the stale generated module was kept")
	}
	for _, path := range []string{handWritten, other, filepath.Join(out, "sm_payments_auth.qnt")} {
		if _, err := os.Stat(path); err != nil {
			t.Errorf("%s was removed: %v", path, err)
		}
	}
}

// TRLC-LINKS: REQ-EMG-141, REQ-EMG-142
func TestGenerateErrorExitsOneAndWritesNothing(t *testing.T) {
	manifest := copyFixture(t)
	behavior := filepath.Join(filepath.Dir(manifest), "model", "behavior.yml")
	broken := strings.Replace(mustRead(t, behavior), "guard: attempts < 3\n      effect", "guard: attempts <\n      effect", 1)
	if err := os.WriteFile(behavior, []byte(broken), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "q")
	code, stdout, stderr := runCLI("generate", "--model", manifest, "--out", out)
	if code != 1 || stdout != "" {
		t.Fatalf("exit %d, stdout %q", code, stdout)
	}
	if !strings.Contains(stderr, "semantic.invalid_expression") || !strings.Contains(stderr, "machines[0].transitions[0].guard") {
		t.Fatalf("stderr lacks the precise diagnostic: %q", stderr)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("an output directory was created for a failed generation")
	}
}

// TRLC-LINKS: REQ-EMG-141, REQ-EMG-142
func TestGenerateModelWithoutMachines(t *testing.T) {
	manifest := copyFixture(t)
	behavior := filepath.Join(filepath.Dir(manifest), "model", "behavior.yml")
	if err := os.WriteFile(behavior, []byte("schemaVersion: 2\nbehavior:\n  machines: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	out := filepath.Join(t.TempDir(), "q")
	code, stdout, stderr := runCLI("generate", "--model", manifest, "--out", out)
	if code != 0 || !strings.Contains(stdout, "no machines") {
		t.Fatalf("exit %d %q %q", code, stdout, stderr)
	}
	if _, err := os.Stat(out); !os.IsNotExist(err) {
		t.Error("an output directory was created for a model without machines")
	}
}
