// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package cliflags

import (
	"flag"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// recorder captures the environment assignments Apply makes.
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
type recorder map[string]string

// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
func (r recorder) set(key, value string) error {
	r[key] = value
	return nil
}

// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
func parse(t *testing.T, args ...string) *Options {
	t.Helper()
	fs := flag.NewFlagSet("test", flag.ContinueOnError)
	opts := Register(fs)
	if err := fs.Parse(args); err != nil {
		t.Fatalf("parse %v: %v", args, err)
	}
	return opts
}

// Unset flags leave ENGMOD_WORK / ENGMOD_LOCK alone so the environment (or the
// library default) still decides.
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestApplyWithoutFlagsKeepsEnvironment(t *testing.T) {
	rec := recorder{}
	if err := parse(t).apply(rec.set); err != nil {
		t.Fatal(err)
	}
	if len(rec) != 0 {
		t.Fatalf("no flag given, expected no environment change, got %v", rec)
	}
}

// TRLC-LINKS: REQ-EMG-063
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestWorkspaceOffSetsEnvironment(t *testing.T) {
	rec := recorder{}
	if err := parse(t, "--workspace", "OFF").apply(rec.set); err != nil {
		t.Fatal(err)
	}
	if rec[WorkspaceEnv] != "off" {
		t.Fatalf("--workspace=OFF must export %s=off, got %v", WorkspaceEnv, rec)
	}
}

// TRLC-LINKS: REQ-EMG-063
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestWorkspacePathIsExportedAbsolute(t *testing.T) {
	dir := t.TempDir()
	work := filepath.Join(dir, "engmod.work.yml")
	if err := os.WriteFile(work, []byte("schemaVersion: 2\nreplacements: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Chdir(dir)
	rec := recorder{}
	if err := parse(t, "--workspace=engmod.work.yml").apply(rec.set); err != nil {
		t.Fatal(err)
	}
	got := rec[WorkspaceEnv]
	if !filepath.IsAbs(got) || filepath.Base(got) != "engmod.work.yml" {
		t.Fatalf("expected an absolute engmod.work.yml path, got %q", got)
	}
	if resolved, _ := filepath.EvalSymlinks(got); resolved != mustEval(t, work) {
		t.Fatalf("exported %q does not point at %q", got, work)
	}
}

// TRLC-LINKS: REQ-EMG-063
func mustEval(t *testing.T, path string) string {
	t.Helper()
	resolved, err := filepath.EvalSymlinks(path)
	if err != nil {
		t.Fatal(err)
	}
	return resolved
}

// TRLC-LINKS: REQ-EMG-063
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestWorkspaceRejectsMissingWrongNameAndDirectory(t *testing.T) {
	dir := t.TempDir()
	other := filepath.Join(dir, "other.yml")
	if err := os.WriteFile(other, []byte("x: 1\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	asDir := filepath.Join(dir, "sub", "engmod.work.yml")
	if err := os.MkdirAll(asDir, 0o755); err != nil {
		t.Fatal(err)
	}
	for name, value := range map[string]string{
		"missing":   filepath.Join(dir, "engmod.work.yml"),
		"wrongName": other,
		"directory": asDir,
	} {
		rec := recorder{}
		err := parse(t, "--workspace", value).apply(rec.set)
		if err == nil || !strings.Contains(err.Error(), "--workspace") {
			t.Fatalf("%s: expected a --workspace error, got %v", name, err)
		}
		if len(rec) != 0 {
			t.Fatalf("%s: rejected value must not change the environment: %v", name, rec)
		}
	}
}

// TRLC-LINKS: REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestLockAcceptsOnlyCheckOrWrite(t *testing.T) {
	for _, mode := range []string{"check", "Write"} {
		rec := recorder{}
		if err := parse(t, "--lock", mode).apply(rec.set); err != nil {
			t.Fatalf("--lock=%s: %v", mode, err)
		}
		if rec[LockEnv] != strings.ToLower(mode) {
			t.Fatalf("--lock=%s must export %s=%s, got %v", mode, LockEnv, strings.ToLower(mode), rec)
		}
	}
	rec := recorder{}
	if err := parse(t, "--lock", "update").apply(rec.set); err == nil || !strings.Contains(err.Error(), "--lock") {
		t.Fatalf("--lock=update must be rejected, got %v", err)
	}
	if len(rec) != 0 {
		t.Fatalf("rejected --lock must not change the environment: %v", rec)
	}
}

// Apply writes the process environment the library reads.
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func TestApplySetsProcessEnvironment(t *testing.T) {
	t.Setenv(WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	if err := parse(t, "--workspace=off", "--lock=check").Apply(); err != nil {
		t.Fatal(err)
	}
	if os.Getenv(WorkspaceEnv) != "off" || os.Getenv(LockEnv) != "check" {
		t.Fatalf("environment not set: %s=%q %s=%q", WorkspaceEnv, os.Getenv(WorkspaceEnv), LockEnv, os.Getenv(LockEnv))
	}
}
