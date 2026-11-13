// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package testutil

import (
	"bytes"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// childMainEnv names the environment variable that makes the test binary run
// a CLI main instead of the test: its value is the newline-separated argument
// list.
const childMainEnv = "ENGFLOW_TEST_CHILD_MAIN_ARGS"

// RunMain runs a CLI main with args in a child process (the test binary
// itself, re-executed for the calling test) and returns its exit code, stdout
// and stderr. The calling test must be a top-level test; RunMain is how a
// test drives a main that calls os.Exit. In the child it runs main and exits.
// TRLC-LINKS: REQ-EMG-168
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func RunMain(t *testing.T, main func(), args ...string) (code int, stdout, stderr string) {
	t.Helper()
	if packed, ok := os.LookupEnv(childMainEnv); ok {
		os.Args = append([]string{filepath.Base(os.Args[0])}, strings.Split(packed, "\n")...)
		if packed == "" {
			os.Args = os.Args[:1]
		}
		main()
		os.Exit(0)
	}
	cmd := exec.Command(os.Args[0], "-test.run=^"+t.Name()+"$")
	cmd.Env = append(os.Environ(), childMainEnv+"="+strings.Join(args, "\n"))
	var out, errOut bytes.Buffer
	cmd.Stdout, cmd.Stderr = &out, &errOut
	err := cmd.Run()
	var exitErr *exec.ExitError
	switch {
	case err == nil:
		code = 0
	case errors.As(err, &exitErr):
		code = exitErr.ExitCode()
	default:
		t.Fatalf("run %s: %v", t.Name(), err)
	}
	return code, out.String(), errOut.String()
}

// NoLifecycleModel writes a small valid schema-v2 module whose manifest has
// no lifecycle and no requirementLinks switch, and returns the path of its
// engmod.yml. The validator reports it with the required-switch error
// lifecycle.missing and its remedy.
// TRLC-LINKS: REQ-EMG-168
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func NoLifecycleModel(t testing.TB) string {
	t.Helper()
	dir := t.TempDir()
	WriteV2ModuleFiles(t, dir, "example.com/nolifecycle@v0", "v0.1.0", "NOLIFECYCLE", "",
		"  functionalGroups:\n    - id: FG-A\n      name: A\n")
	manifest := filepath.Join(dir, "engmod.yml")
	WithoutSwitches(t, manifest)
	return manifest
}

// AssertRefusesWithFix fails the test unless a CLI refused a model without
// lifecycle: a non-zero exit code and an output with the lifecycle.missing
// error and a "fix:" line.
// TRLC-LINKS: REQ-EMG-168
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func AssertRefusesWithFix(t testing.TB, code int, output string) {
	t.Helper()
	if code == 0 {
		t.Errorf("exit code 0, want non-zero; output:\n%s", output)
	}
	if !strings.Contains(output, "lifecycle.missing") {
		t.Errorf("output lacks lifecycle.missing:\n%s", output)
	}
	if !strings.Contains(output, "fix: ") {
		t.Errorf("output lacks a fix: line:\n%s", output)
	}
}
