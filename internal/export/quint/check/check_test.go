// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"context"
	"errors"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/export/quint"
	"github.com/labeth/engineering-model-go/internal/model"
)

const (
	fixtureDir = "../testdata/payments"
)

// env is a copy of the payments fixture with its modules generated and a fake
// quint in place.
type env struct {
	manifest string
	outDir   string
	log      string
	quint    string
	dir      string
}

func newEnv(t *testing.T) env {
	t.Helper()
	dir := t.TempDir()
	copyTree(t, fixtureDir, filepath.Join(dir, "model-root"))
	e := env{dir: dir, manifest: filepath.Join(dir, "model-root", "engmod.yml")}
	e.outDir = filepath.Join(dir, "model-root", "generated", "quint")
	generateInto(t, e.manifest, e.outDir)
	script, err := os.ReadFile("testdata/fake-quint.sh")
	if err != nil {
		t.Fatal(err)
	}
	e.quint = filepath.Join(dir, "bin", "quint")
	writeFile(t, e.quint, string(script), 0o755)
	e.log = filepath.Join(dir, "fake-quint.log")
	t.Setenv("FAKE_QUINT_LOG", e.log)
	for _, name := range []string{"FAKE_QUINT_VIOLATE", "FAKE_QUINT_RUNTIME_ERR", "FAKE_QUINT_TRACE_MAX", "FAKE_QUINT_WIT_ZERO", "FAKE_QUINT_TYPE_ERROR", "FAKE_QUINT_SEQUENCE", "FAKE_QUINT_VERSION"} {
		t.Setenv(name, "")
	}
	return e
}

func (e env) opts() CheckOptions {
	return CheckOptions{ModelPath: e.manifest, QuintPath: e.quint}
}

// calls returns the logged invocations of the fake quint with the given
// subcommand.
func (e env) calls(t *testing.T, sub string) []string {
	t.Helper()
	data, err := os.ReadFile(e.log)
	if errors.Is(err, fs.ErrNotExist) {
		return nil
	}
	if err != nil {
		t.Fatal(err)
	}
	var out []string
	for _, line := range strings.Split(strings.TrimSpace(string(data)), "\n") {
		if strings.HasPrefix(line, sub+" ") {
			out = append(out, line)
		}
	}
	return out
}

func writeFile(t *testing.T, path, content string, mode os.FileMode) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), mode); err != nil {
		t.Fatal(err)
	}
}

func readFile(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func copyTree(t *testing.T, src, dst string) {
	t.Helper()
	err := filepath.WalkDir(src, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(src, path)
		target := filepath.Join(dst, rel)
		if d.IsDir() {
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
}

// generateInto writes the generated modules of a model into dir, the way
// engquint generate does.
func generateInto(t *testing.T, manifest, dir string) quint.Result {
	t.Helper()
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	result, err := quint.Generate(bundle)
	if err != nil {
		t.Fatalf("generate: %v %+v", err, result.Diagnostics)
	}
	for _, f := range result.Files {
		writeFile(t, filepath.Join(dir, f.Name), f.Text, 0o644)
	}
	return result
}

func mustCheck(t *testing.T, opts CheckOptions) CheckReport {
	t.Helper()
	report, err := Check(opts)
	if err != nil {
		t.Fatalf("Check: %v", err)
	}
	return report
}

func invariant(t *testing.T, m ModuleResult, id string) InvariantResult {
	t.Helper()
	for _, inv := range m.Invariants {
		if inv.ID == id {
			return inv
		}
	}
	t.Fatalf("no invariant %s in %+v", id, m.Invariants)
	return InvariantResult{}
}

func evidenceOf(t *testing.T, e env) string {
	t.Helper()
	return readFile(t, filepath.Join(filepath.Dir(e.manifest), "test-results", "quint-formal.json"))
}

// REQ-EMG-143: one witnesses run and one run per invariant, always on the
// typescript backend with the bounds and the seed; the fake quint fails loudly
// when a mandatory flag is missing.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-145, REQ-EMG-146
func TestCheckRunModePassesAndRecordsResults(t *testing.T) {
	e := newEnv(t)
	report := mustCheck(t, e.opts())
	if report.Failed() || report.NoMachines || report.QuintVersion != "0.33.1" {
		t.Fatalf("report %+v", report)
	}
	runs := e.calls(t, "run")
	if len(runs) != 3 {
		t.Fatalf("want 1 witnesses run + 2 invariant runs, got %d: %v", len(runs), runs)
	}
	for _, run := range runs {
		for _, want := range []string{"backend=typescript", "main=sm_payments_auth", "seed=1", "samples=10000", "steps=20"} {
			if !strings.Contains(run, want) {
				t.Errorf("run lacks %s: %s", want, run)
			}
		}
	}
	if !strings.Contains(runs[0], "wits=[wit_state_payments_idle wit_state_payments_authorizing wit_state_payments_authorized wit_state_payments_declined]") {
		t.Errorf("first run is not the witnesses run: %s", runs[0])
	}
	if !strings.Contains(runs[1], "inv=inv_payments_001") || !strings.Contains(runs[2], "inv=inv_payments_002") {
		t.Errorf("invariant runs: %v", runs[1:])
	}
	if len(e.calls(t, "verify")) != 0 {
		t.Error("verify ran in run mode")
	}

	m := report.Modules[0]
	if m.Status != StatusPass || m.Machine != "SM-PAYMENTS-AUTH" || m.Backend != "typescript" || m.QntSha256 == "" || len(m.Witnesses) != 4 {
		t.Fatalf("module %+v", m)
	}
	inv := invariant(t, m, "INV-PAYMENTS-002")
	if inv.Run == nil || inv.Run.Status != StatusPass || *inv.Run.TraceMax != 7 || inv.Verify != nil || len(inv.Requirements) != 2 {
		t.Fatalf("invariant %+v", inv)
	}

	results := readFile(t, report.ResultsPath)
	for _, want := range []string{`"schema": "engmod-quint-results/1"`, `"quintVersion": "0.33.1"`, `"backend": "typescript"`, `"maxSteps": 20`, `"seed": "1"`, `"qntSha256"`, `"witnessed": 17`} {
		if !strings.Contains(results, want) {
			t.Errorf("results.json lacks %s", want)
		}
	}
	for _, banned := range []string{"ms", "timestamp", "duration"} {
		if strings.Contains(results, banned) {
			t.Errorf("results.json contains %q:\n%s", banned, results)
		}
	}
	if report.EvidenceEntries != 3 || !strings.Contains(evidenceOf(t, e), `"status": "pass"`) {
		t.Errorf("evidence: %d entries\n%s", report.EvidenceEntries, evidenceOf(t, e))
	}
}

// TRLC-LINKS: REQ-EMG-145
func TestCheckResultsAreByteIdenticalOnRerun(t *testing.T) {
	e := newEnv(t)
	mustCheck(t, e.opts())
	first, firstEvidence := readFile(t, filepath.Join(e.outDir, "results.json")), evidenceOf(t, e)
	mustCheck(t, e.opts())
	if first != readFile(t, filepath.Join(e.outDir, "results.json")) || firstEvidence != evidenceOf(t, e) {
		t.Fatal("results or evidence changed on a rerun with the same inputs")
	}
}

// TRLC-LINKS: REQ-EMG-143, REQ-EMG-145, REQ-EMG-146
func TestCheckViolationMapsCounterexampleToOrderedTransitionIDs(t *testing.T) {
	e := newEnv(t)
	t.Setenv("FAKE_QUINT_VIOLATE", "inv_payments_001")
	t.Setenv("FAKE_QUINT_SEQUENCE", "TR-PAYMENTS-001 TR-PAYMENTS-003 TR-PAYMENTS-001")
	report := mustCheck(t, e.opts())
	if !report.Failed() {
		t.Fatal("a violation must fail the check")
	}
	m := report.Modules[0]
	if m.Status != StatusViolated {
		t.Fatalf("module status %q", m.Status)
	}
	bad := invariant(t, m, "INV-PAYMENTS-001")
	want := []string{"TR-PAYMENTS-001", "TR-PAYMENTS-003", "TR-PAYMENTS-001"}
	if bad.Status != StatusViolated || strings.Join(bad.Run.Counterexample, ",") != strings.Join(want, ",") || bad.Run.ExitCode != 1 {
		t.Fatalf("violated invariant %+v run %+v", bad, bad.Run)
	}
	if got := FormatCounterexample(bad.Run.Counterexample); got != "TR-PAYMENTS-001 -> TR-PAYMENTS-003 -> TR-PAYMENTS-001" {
		t.Errorf("formatted %q", got)
	}
	if good := invariant(t, m, "INV-PAYMENTS-002"); good.Status != StatusPass {
		t.Errorf("the other invariant must be attributed separately: %+v", good)
	}
	// The violation is evidence: a fail entry for its requirement and pass
	// entries for the passing invariant.
	evidence := evidenceOf(t, e)
	if !strings.Contains(evidence, `"status": "fail"`) || !strings.Contains(evidence, "TR-PAYMENTS-001 -> TR-PAYMENTS-003 -> TR-PAYMENTS-001") {
		t.Errorf("evidence lacks the fail entry:\n%s", evidence)
	}
	if strings.Count(evidence, `"status": "pass"`) != 2 {
		t.Errorf("evidence should hold two pass entries:\n%s", evidence)
	}
}

// TRLC-LINKS: REQ-EMG-143
func TestCheckZeroLengthTraceIsAnErrorNotAPass(t *testing.T) {
	e := newEnv(t)
	t.Setenv("FAKE_QUINT_TRACE_MAX", "0")
	report := mustCheck(t, e.opts())
	m := report.Modules[0]
	if !report.Failed() || m.Status != StatusError {
		t.Fatalf("a vacuous init must fail: %+v", m)
	}
	inv := invariant(t, m, "INV-PAYMENTS-001")
	if inv.Status != StatusError || !strings.Contains(inv.Run.Message, "vacuous") {
		t.Errorf("invariant %+v", inv.Run)
	}
	if report.EvidenceEntries != 0 || strings.Contains(evidenceOf(t, e), `"pass"`) {
		t.Errorf("a vacuous pass must not be evidence:\n%s", evidenceOf(t, e))
	}
}

// TRLC-LINKS: REQ-EMG-143
func TestCheckTraceOfLengthOneWarnsDeadlockButPasses(t *testing.T) {
	e := newEnv(t)
	t.Setenv("FAKE_QUINT_TRACE_MAX", "1")
	report := mustCheck(t, e.opts())
	m := report.Modules[0]
	if report.Failed() || len(m.Warnings) != 1 || !strings.HasPrefix(m.Warnings[0], "deadlock:") {
		t.Fatalf("status %q warnings %v", m.Status, m.Warnings)
	}
}

// TRLC-LINKS: REQ-EMG-143
func TestCheckUnreachedWitnessIsAWarningNotAFailure(t *testing.T) {
	e := newEnv(t)
	t.Setenv("FAKE_QUINT_WIT_ZERO", "wit_state_payments_declined")
	report := mustCheck(t, e.opts())
	m := report.Modules[0]
	if report.Failed() {
		t.Fatalf("an unreachable state must not fail: %+v", m)
	}
	if len(m.Warnings) != 1 || !strings.HasPrefix(m.Warnings[0], "unreachable state STATE-PAYMENTS-DECLINED") {
		t.Fatalf("warnings %v", m.Warnings)
	}
	if got := m.Witnesses[3]; got.Witnessed != 0 || got.Total != 10000 || got.State != "STATE-PAYMENTS-DECLINED" {
		t.Errorf("witness %+v", got)
	}
}

// TRLC-LINKS: REQ-EMG-143
func TestCheckInitialStateIsNeverReportedUnreachable(t *testing.T) {
	e := newEnv(t)
	t.Setenv("FAKE_QUINT_WIT_ZERO", "wit_state_payments_idle")
	report := mustCheck(t, e.opts())
	m := report.Modules[0]
	if report.Failed() || len(m.Warnings) != 0 {
		t.Fatalf("the initial state holds by construction, got status %q warnings %v", m.Status, m.Warnings)
	}
	if got := m.Witnesses[0]; got.Witnessed != 0 || got.State != "STATE-PAYMENTS-IDLE" {
		t.Errorf("the witness count is still recorded as is: %+v", got)
	}
}

// TRLC-LINKS: REQ-EMG-143
func TestCheckRuntimeErrorIsAnErrorWithoutTimings(t *testing.T) {
	e := newEnv(t)
	t.Setenv("FAKE_QUINT_RUNTIME_ERR", "inv_payments_002")
	report := mustCheck(t, e.opts())
	m := report.Modules[0]
	inv := invariant(t, m, "INV-PAYMENTS-002")
	if !report.Failed() || inv.Status != StatusError || !strings.Contains(inv.Run.Message, "QNT502") || strings.Contains(inv.Run.Message, "12ms") {
		t.Fatalf("invariant %+v run %+v", inv, inv.Run)
	}
	if report.EvidenceEntries != 0 {
		t.Errorf("a module with an error has no evidence: %d", report.EvidenceEntries)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckTypeErrorFailsWithLocationAndRunsNothing(t *testing.T) {
	e := newEnv(t)
	t.Setenv("FAKE_QUINT_TYPE_ERROR", "1")
	report := mustCheck(t, e.opts())
	m := report.Modules[0]
	if !report.Failed() || m.Status != StatusError || len(m.TypeErrors) != 1 {
		t.Fatalf("module %+v", m)
	}
	if want := "sm_payments_auth.qnt:3:5: [QNT000] Couldn't unify int and str"; m.TypeErrors[0] != want {
		t.Errorf("type error %q, want %q", m.TypeErrors[0], want)
	}
	if len(e.calls(t, "run")) != 0 {
		t.Error("quint run was called for a module that does not typecheck")
	}
	if report.EvidenceEntries != 0 {
		t.Error("evidence for a module that does not typecheck")
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckMissingQuintIsASetupError(t *testing.T) {
	e := newEnv(t)
	t.Setenv("ENGMOD_QUINT", "")
	t.Setenv("PATH", t.TempDir())
	opts := e.opts()
	opts.QuintPath = ""
	_, err := Check(opts)
	if !IsSetupError(err) || !strings.Contains(err.Error(), "scripts/install-quint-toolchain.sh") {
		t.Fatalf("PATH lookup: %v", err)
	}
	opts.QuintPath = filepath.Join(e.dir, "no-such-quint")
	if _, err := Check(opts); !IsSetupError(err) {
		t.Fatalf("configured path: %v", err)
	}
	t.Setenv("ENGMOD_QUINT", filepath.Join(e.dir, "also-missing"))
	opts.QuintPath = ""
	if _, err := Check(opts); !IsSetupError(err) {
		t.Fatalf("ENGMOD_QUINT: %v", err)
	}
	if _, statErr := os.Stat(filepath.Join(e.outDir, "results.json")); statErr == nil {
		t.Error("a setup error wrote results")
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckUnloadableModelIsASetupError(t *testing.T) {
	_, err := Check(CheckOptions{ModelPath: filepath.Join(t.TempDir(), "engmod.yml")})
	if !IsSetupError(err) {
		t.Fatalf("got %v", err)
	}
	if _, err := Check(CheckOptions{}); !IsSetupError(err) {
		t.Fatalf("no model: %v", err)
	}
	if _, err := Check(CheckOptions{ModelPath: "x", Mode: "sideways"}); !IsSetupError(err) {
		t.Fatalf("bad mode: %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckModelWithoutMachinesWritesNothingAndNeedsNoTool(t *testing.T) {
	e := newEnv(t)
	writeFile(t, filepath.Join(filepath.Dir(e.manifest), "model", "behavior.yml"), "schemaVersion: 2\nbehavior:\n  machines: []\n", 0o644)
	out := filepath.Join(t.TempDir(), "q")
	evidence := filepath.Join(t.TempDir(), "quint-formal.json")
	report := mustCheck(t, CheckOptions{ModelPath: e.manifest, QuintPath: "/nonexistent/quint", OutDir: out, EvidencePath: evidence})
	if !report.NoMachines || report.Failed() {
		t.Fatalf("report %+v", report)
	}
	for _, path := range []string{out, evidence} {
		if _, err := os.Stat(path); err == nil {
			t.Errorf("%s was written for a model without machines", path)
		}
	}
}

// Drift is stale: the committed module is not what the model says, so nothing
// runs for it and it is no evidence.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-149
func TestCheckDriftIsStaleRunsNothingAndWritesNoEvidence(t *testing.T) {
	e := newEnv(t)
	module := filepath.Join(e.outDir, "sm_payments_auth.qnt")
	writeFile(t, module, readFile(t, module)+"// hand edit\n", 0o644)
	report := mustCheck(t, e.opts())
	if !report.Failed() || len(report.Drift) != 1 || report.Drift[0] != (Drift{File: "sm_payments_auth.qnt", Kind: "changed"}) {
		t.Fatalf("drift %+v", report.Drift)
	}
	m := report.Modules[0]
	if m.Status != StatusStale || m.Drift != "changed" || invariant(t, m, "INV-PAYMENTS-001").Status != StatusStale {
		t.Fatalf("module %+v", m)
	}
	if len(e.calls(t, "run"))+len(e.calls(t, "typecheck")) != 0 {
		t.Error("a drifted module was typechecked or run")
	}
	if report.EvidenceEntries != 0 || strings.Contains(evidenceOf(t, e), `"requirement"`) {
		t.Errorf("a drifted module has evidence:\n%s", evidenceOf(t, e))
	}
}

// TRLC-LINKS: REQ-EMG-145, REQ-EMG-149
func TestCheckMissingAndExtraGeneratedModulesAreDrift(t *testing.T) {
	e := newEnv(t)
	extra := filepath.Join(e.outDir, "sm_gone.qnt")
	writeFile(t, extra, quint.GeneratedHeader+"SM-GONE; do not edit.\nmodule sm_gone {}\n", 0o644)
	writeFile(t, filepath.Join(e.outDir, "notes.qnt"), "// hand written\nmodule notes {}\n", 0o644)
	report := mustCheck(t, e.opts())
	if len(report.Drift) != 1 || report.Drift[0] != (Drift{File: "sm_gone.qnt", Kind: "extra"}) || !report.Failed() {
		t.Fatalf("extra: %+v", report.Drift)
	}
	if err := os.Remove(extra); err != nil {
		t.Fatal(err)
	}
	if err := os.Remove(filepath.Join(e.outDir, "sm_payments_auth.qnt")); err != nil {
		t.Fatal(err)
	}
	report = mustCheck(t, e.opts())
	if len(report.Drift) != 1 || report.Drift[0].Kind != "missing" || report.Modules[0].Status != StatusStale || report.Modules[0].QntSha256 != "" {
		t.Fatalf("missing: %+v %+v", report.Drift, report.Modules[0])
	}
}

// Evidence is rewritten on every check, so entries of an earlier pass do not
// outlive a drift.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func TestCheckRewritesEvidenceAndHonoursNoEvidence(t *testing.T) {
	e := newEnv(t)
	mustCheck(t, e.opts())
	if !strings.Contains(evidenceOf(t, e), "REQ-PAYMENTS-002") {
		t.Fatal("first check wrote no evidence")
	}
	module := filepath.Join(e.outDir, "sm_payments_auth.qnt")
	writeFile(t, module, readFile(t, module)+"\n", 0o644)
	mustCheck(t, e.opts())
	if strings.Contains(evidenceOf(t, e), "REQ-PAYMENTS-002") {
		t.Fatalf("stale evidence survived a drift:\n%s", evidenceOf(t, e))
	}

	e = newEnv(t)
	opts := e.opts()
	opts.NoEvidence = true
	opts.EvidencePath = filepath.Join(e.dir, "ev.json")
	report := mustCheck(t, opts)
	if report.EvidencePath != "" {
		t.Error("evidence path reported with NoEvidence")
	}
	if _, err := os.Stat(opts.EvidencePath); err == nil {
		t.Error("evidence written with NoEvidence")
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckBoundsAndPathsAreConfigurable(t *testing.T) {
	e := newEnv(t)
	opts := e.opts()
	opts.MaxSteps, opts.MaxSamples, opts.Seed = 5, 77, "0x2a"
	opts.ResultsPath = filepath.Join(e.dir, "r", "res.json")
	opts.EvidencePath = filepath.Join(e.dir, "r", "ev.json")
	report := mustCheck(t, opts)
	run := e.calls(t, "run")[1]
	for _, want := range []string{"seed=0x2a", "samples=77", "steps=5"} {
		if !strings.Contains(run, want) {
			t.Errorf("run lacks %s: %s", want, run)
		}
	}
	if report.ResultsPath != opts.ResultsPath || !strings.Contains(readFile(t, opts.ResultsPath), `"seed": "0x2a"`) || report.EvidencePath != opts.EvidencePath {
		t.Errorf("paths %+v", report)
	}
}

// REQ-EMG-143 with an injected runner: no process is started.
//
// TRLC-LINKS: REQ-EMG-143
func TestCheckUsesTheInjectedRunner(t *testing.T) {
	e := newEnv(t)
	var seen []string
	opts := e.opts()
	opts.QuintPath = e.quint
	opts.Runner = func(_ context.Context, cmd Command) (Output, error) {
		seen = append(seen, strings.Join(cmd.Args, " "))
		switch cmd.Args[0] {
		case "--version":
			return Output{Stdout: "9.9.9\n"}, nil
		case "run":
			return Output{Stdout: "[ok] No violation found (1ms at 1 traces/second).\nTrace length statistics: max=3, min=2, average=2.50\n"}, nil
		}
		return Output{}, nil
	}
	report := mustCheck(t, opts)
	if report.QuintVersion != "9.9.9" || report.Failed() || len(seen) != 1+1+3 {
		t.Fatalf("version %q failed %v calls %v", report.QuintVersion, report.Failed(), seen)
	}
	if len(e.calls(t, "run")) != 0 {
		t.Error("the real process ran")
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckFailedRunnerStartIsASetupError(t *testing.T) {
	e := newEnv(t)
	opts := e.opts()
	opts.Runner = func(context.Context, Command) (Output, error) { return Output{}, errors.New("exec format error") }
	if _, err := Check(opts); !IsSetupError(err) {
		t.Fatalf("got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestGenerationErrorIsReportedNotRun(t *testing.T) {
	e := newEnv(t)
	behavior := filepath.Join(filepath.Dir(e.manifest), "model", "behavior.yml")
	writeFile(t, behavior, strings.Replace(readFile(t, behavior), "guard: attempts < 3\n      effect", "guard: attempts <\n      effect", 1), 0o644)
	report := mustCheck(t, e.opts())
	if !report.Failed() || len(report.GenerationErrors) == 0 || !strings.Contains(report.GenerationErrors[0], "semantic.invalid_expression") {
		t.Fatalf("report %+v", report)
	}
	if len(e.calls(t, "run"))+len(e.calls(t, "typecheck")) != 0 {
		t.Error("quint ran for a model that does not generate")
	}
}
