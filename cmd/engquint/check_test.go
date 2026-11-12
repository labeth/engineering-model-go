// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const fakeQuintScript = "../../internal/export/quint/check/testdata/fake-quint.sh"

// fakeQuint installs the fake quint of the check package and points
// ENGMOD_QUINT at it.
func fakeQuint(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "quint")
	if err := os.WriteFile(path, []byte(mustRead(t, fakeQuintScript)), 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENGMOD_QUINT", path)
	t.Setenv("FAKE_QUINT_LOG", filepath.Join(dir, "log"))
	for _, name := range []string{"FAKE_QUINT_VIOLATE", "FAKE_QUINT_RUNTIME_ERR", "FAKE_QUINT_TRACE_MAX", "FAKE_QUINT_WIT_ZERO", "FAKE_QUINT_TYPE_ERROR", "FAKE_QUINT_SEQUENCE"} {
		t.Setenv(name, "")
	}
	return path
}

// generated returns a fixture model whose module is generated and committed.
func generated(t *testing.T) string {
	t.Helper()
	manifest := copyFixture(t)
	if code, _, stderr := runCLI("generate", "--model", manifest); code != 0 {
		t.Fatalf("generate: %d %s", code, stderr)
	}
	return manifest
}

// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-146
func TestCheckPassesExitsZeroAndWritesResultsAndEvidence(t *testing.T) {
	fakeQuint(t)
	manifest := generated(t)
	code, stdout, stderr := runCLI("check", "--model", manifest)
	if code != 0 {
		t.Fatalf("exit %d\n%s\n%s", code, stdout, stderr)
	}
	for _, want := range []string{
		"SM-PAYMENTS-AUTH (generated/quint/sm_payments_auth.qnt, run) PASS",
		"INV-PAYMENTS-001", "REQ-PAYMENTS-001", "run: pass",
		"INV-PAYMENTS-002", "REQ-PAYMENTS-001, REQ-PAYMENTS-002",
		"result: pass", "(3 entries)",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout lacks %q:\n%s", want, stdout)
		}
	}
	root := filepath.Dir(manifest)
	if !strings.Contains(mustRead(t, filepath.Join(root, "generated", "quint", "results.json")), `"backend": "typescript"`) {
		t.Error("results.json not written next to the modules")
	}
	if !strings.Contains(mustRead(t, filepath.Join(root, "test-results", "quint-formal.json")), `"method": "analysis"`) {
		t.Error("evidence not written to test-results")
	}
}

// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143
func TestCheckViolationExitsOneAndNamesTheCounterexample(t *testing.T) {
	fakeQuint(t)
	t.Setenv("FAKE_QUINT_VIOLATE", "inv_payments_002")
	t.Setenv("FAKE_QUINT_SEQUENCE", "TR-PAYMENTS-001 TR-PAYMENTS-002")
	code, stdout, _ := runCLI("check", "--model", generated(t))
	if code != 1 {
		t.Fatalf("exit %d", code)
	}
	for _, want := range []string{
		"INV-PAYMENTS-002 violated: TR-PAYMENTS-001 -> TR-PAYMENTS-002 (run; REQ-PAYMENTS-001, REQ-PAYMENTS-002)",
		"run: violated", "result: FAIL",
	} {
		if !strings.Contains(stdout, want) {
			t.Errorf("stdout lacks %q:\n%s", want, stdout)
		}
	}
}

// TRLC-LINKS: REQ-EMG-142, REQ-EMG-145
func TestCheckStaleModuleExitsOneAndWritesNoEvidenceEntries(t *testing.T) {
	fakeQuint(t)
	manifest := generated(t)
	module := filepath.Join(filepath.Dir(manifest), "generated", "quint", "sm_payments_auth.qnt")
	if err := os.WriteFile(module, []byte(mustRead(t, module)+"// edit\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	code, stdout, _ := runCLI("check", "--model", manifest)
	if code != 1 || !strings.Contains(stdout, "stale: sm_payments_auth.qnt is changed; run engquint generate") || !strings.Contains(stdout, "(0 entries)") {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
}

// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143
func TestCheckVacuousInitExitsOneWithAnExplanation(t *testing.T) {
	fakeQuint(t)
	t.Setenv("FAKE_QUINT_TRACE_MAX", "0")
	code, stdout, _ := runCLI("check", "--model", generated(t))
	if code != 1 || !strings.Contains(stdout, "vacuous") {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
}

// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143
func TestCheckWarningsDoNotFailTheCheck(t *testing.T) {
	fakeQuint(t)
	t.Setenv("FAKE_QUINT_WIT_ZERO", "wit_state_payments_declined")
	code, stdout, _ := runCLI("check", "--model", generated(t))
	if code != 0 || !strings.Contains(stdout, "warning: unreachable state STATE-PAYMENTS-DECLINED") {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckTypeErrorExitsOne(t *testing.T) {
	fakeQuint(t)
	t.Setenv("FAKE_QUINT_TYPE_ERROR", "1")
	code, stdout, _ := runCLI("check", "--model", generated(t))
	if code != 1 || !strings.Contains(stdout, "type error: sm_payments_auth.qnt:3:5") {
		t.Fatalf("exit %d\n%s", code, stdout)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckSetupErrorsExitTwo(t *testing.T) {
	manifest := generated(t)
	t.Setenv("ENGMOD_QUINT", filepath.Join(t.TempDir(), "no-quint"))
	code, _, stderr := runCLI("check", "--model", manifest)
	if code != 2 || !strings.Contains(stderr, "quint is not available") {
		t.Fatalf("missing quint: %d %q", code, stderr)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(manifest), "test-results")); err == nil {
		t.Error("a setup error wrote evidence")
	}

	fakeQuint(t)
	code, _, stderr = runCLI("check", "--model", manifest, "--mode", "verify", "--quint-home", t.TempDir())
	if code != 2 || !strings.Contains(stderr, "scripts/install-quint-toolchain.sh") {
		t.Fatalf("missing Apalache: %d %q", code, stderr)
	}
	code, _, stderr = runCLI("check", "--model", filepath.Join(t.TempDir(), "engmod.yml"))
	if code != 2 || !strings.Contains(stderr, "load model") {
		t.Fatalf("unreadable model: %d %q", code, stderr)
	}
	if code, _, stderr = runCLI("check", "--model", manifest, "--mode", "sideways"); code != 2 || !strings.Contains(stderr, "unknown mode") {
		t.Fatalf("bad mode: %d %q", code, stderr)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckUsageErrors(t *testing.T) {
	for _, args := range [][]string{{"check"}, {"check", "--bogus"}, {"check", "--model", "engmod.yml", "extra"}} {
		if code, _, _ := runCLI(args...); code != 2 {
			t.Errorf("%v: exit %d, want 2", args, code)
		}
	}
	if code, _, _ := runCLI("check", "-h"); code != 0 {
		t.Errorf("check -h: exit %d, want 0", code)
	}
	if _, stdout, _ := runCLI("help"); !strings.Contains(stdout, "check") {
		t.Errorf("usage does not list check: %q", stdout)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckModelWithoutMachinesPrintsNoMachinesAndWritesNothing(t *testing.T) {
	manifest := copyFixture(t)
	behavior := filepath.Join(filepath.Dir(manifest), "model", "behavior.yml")
	if err := os.WriteFile(behavior, []byte("schemaVersion: 2\nbehavior:\n  machines: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("ENGMOD_QUINT", "") // no tool is needed
	code, stdout, stderr := runCLI("check", "--model", manifest)
	if code != 0 || !strings.Contains(stdout, "no machines") {
		t.Fatalf("exit %d %q %q", code, stdout, stderr)
	}
	for _, dir := range []string{"generated", "test-results"} {
		if _, err := os.Stat(filepath.Join(filepath.Dir(manifest), dir)); err == nil {
			t.Errorf("%s was written for a model without machines", dir)
		}
	}
}

// TRLC-LINKS: REQ-EMG-142, REQ-EMG-146
func TestCheckFlagsSelectPathsAndSuppressEvidence(t *testing.T) {
	fakeQuint(t)
	manifest := generated(t)
	results := filepath.Join(t.TempDir(), "res.json")
	code, stdout, stderr := runCLI("check", "--model", manifest, "--results", results, "--no-evidence", "--max-steps", "4", "--max-samples", "9", "--seed", "7")
	if code != 0 {
		t.Fatalf("exit %d %s", code, stderr)
	}
	if got := mustRead(t, results); !strings.Contains(got, `"maxSamples": 9`) || !strings.Contains(got, `"seed": "7"`) {
		t.Errorf("results:\n%s", got)
	}
	if strings.Contains(stdout, "evidence:") {
		t.Errorf("evidence reported with --no-evidence:\n%s", stdout)
	}
	if _, err := os.Stat(filepath.Join(filepath.Dir(manifest), "test-results")); err == nil {
		t.Error("evidence written with --no-evidence")
	}
}
