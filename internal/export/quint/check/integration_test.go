// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
)

// realEnv copies the payments fixture into a temporary directory and
// generates its module. It skips the test unless ENGMOD_QUINT names a quint
// executable.
func realEnv(t *testing.T) env {
	t.Helper()
	quintPath := os.Getenv(QuintEnv)
	if quintPath == "" {
		t.Skipf("set %s to run against the real quint", QuintEnv)
	}
	dir := t.TempDir()
	copyTree(t, fixtureDir, filepath.Join(dir, "model-root"))
	e := env{dir: dir, quint: quintPath, manifest: filepath.Join(dir, "model-root", "engmod.yml")}
	e.outDir = filepath.Join(dir, "model-root", "generated", "quint")
	generateInto(t, e.manifest, e.outDir)
	return e
}

// breakInvariant weakens INV-PAYMENTS-001 so that the machine violates it, and
// regenerates the module.
func breakInvariant(t *testing.T, e env) {
	t.Helper()
	behavior := filepath.Join(filepath.Dir(e.manifest), "model", "behavior.yml")
	text := readFile(t, behavior)
	broken := strings.Replace(text, "expr: attempts >= 0 and attempts <= 3", "expr: attempts >= 0 and attempts <= 2", 1)
	if broken == text {
		t.Fatal("fixture invariant not found")
	}
	writeFile(t, behavior, broken, 0o644)
	generateInto(t, e.manifest, e.outDir)
}

// realVerifyEnv additionally needs a provisioned Apalache and Java 21.
func realVerifyEnv(t *testing.T) (env, string) {
	t.Helper()
	e := realEnv(t)
	home := os.Getenv(QuintHomeEnv)
	if home == "" {
		t.Skipf("set %s to run verify mode against the real Apalache", QuintHomeEnv)
	}
	if _, err := os.Stat(apalacheMC(home)); err != nil {
		t.Skipf("Apalache %s is not provisioned under %s", ApalacheVersion, home)
	}
	if _, err := resolveJava(context.Background(), ExecRunner); err != nil {
		t.Skipf("no usable Java: %v", err)
	}
	return e, home
}

// TRLC-LINKS: REQ-EMG-143, REQ-EMG-145, REQ-EMG-146
func TestRealQuintRunModeOnTheFixture(t *testing.T) {
	e := realEnv(t)
	opts := CheckOptions{ModelPath: e.manifest, QuintPath: e.quint, MaxSamples: 1000}
	report := mustCheck(t, opts)
	if report.Failed() {
		t.Fatalf("the fixture must pass: %+v", report.Modules)
	}
	m := report.Modules[0]
	if m.QuintVersion == "" || len(m.Witnesses) != 4 || report.EvidenceEntries != 3 {
		t.Fatalf("module %+v entries %d", m, report.EvidenceEntries)
	}
	for _, w := range m.Witnesses {
		if w.Witnessed == 0 {
			t.Errorf("state %s is not reached", w.State)
		}
	}
	for _, inv := range m.Invariants {
		if inv.Run.Status != StatusPass || inv.Run.TraceMax == nil || *inv.Run.TraceMax < 2 {
			t.Errorf("%s: %+v", inv.ID, inv.Run)
		}
	}
	again := mustCheck(t, opts)
	if readFile(t, report.ResultsPath) != readFile(t, again.ResultsPath) {
		t.Error("results.json differs between two identical runs")
	}
}

// TRLC-LINKS: REQ-EMG-143, REQ-EMG-145
func TestRealQuintRunModeReportsABrokenInvariantWithItsTransitions(t *testing.T) {
	e := realEnv(t)
	breakInvariant(t, e)
	report := mustCheck(t, CheckOptions{ModelPath: e.manifest, QuintPath: e.quint, MaxSamples: 1000})
	if !report.Failed() {
		t.Fatal("a broken invariant must fail")
	}
	m := report.Modules[0]
	bad := invariant(t, m, "INV-PAYMENTS-001")
	if bad.Status != StatusViolated || len(bad.Run.Counterexample) == 0 {
		t.Fatalf("invariant %+v run %+v", bad, bad.Run)
	}
	for _, id := range bad.Run.Counterexample {
		if !strings.HasPrefix(id, "TR-PAYMENTS-00") {
			t.Errorf("counterexample step %q is not a transition ID", id)
		}
	}
	if bad.Run.Counterexample[0] != "TR-PAYMENTS-001" {
		t.Errorf("the first step must leave the initial state: %v", bad.Run.Counterexample)
	}
	if good := invariant(t, m, "INV-PAYMENTS-002"); good.Status != StatusPass {
		t.Errorf("the intact invariant must still pass: %+v", good)
	}
}

// TRLC-LINKS: REQ-EMG-144, REQ-EMG-145
func TestRealQuintVerifyModePassesOnTheFixture(t *testing.T) {
	e, home := realVerifyEnv(t)
	report := mustCheck(t, CheckOptions{ModelPath: e.manifest, QuintPath: e.quint, QuintHome: home, Mode: ModeBoth, MaxSamples: 500})
	if report.Failed() {
		t.Fatalf("the fixture must pass in both modes: %+v", report.Modules)
	}
	for _, inv := range report.Modules[0].Invariants {
		if inv.Verify == nil || inv.Verify.Status != StatusPass || inv.Run.Status != StatusPass {
			t.Errorf("%s: run %+v verify %+v", inv.ID, inv.Run, inv.Verify)
		}
	}
}

// TRLC-LINKS: REQ-EMG-144
func TestRealQuintVerifyModeFindsTheCounterexample(t *testing.T) {
	e, home := realVerifyEnv(t)
	breakInvariant(t, e)
	report := mustCheck(t, CheckOptions{ModelPath: e.manifest, QuintPath: e.quint, QuintHome: home, Mode: ModeVerify})
	bad := invariant(t, report.Modules[0], "INV-PAYMENTS-001")
	if !report.Failed() || bad.Verify.Status != StatusViolated || len(bad.Verify.Counterexample) == 0 {
		t.Fatalf("invariant %+v verify %+v", bad, bad.Verify)
	}
	for _, id := range bad.Verify.Counterexample {
		if !strings.HasPrefix(id, "TR-PAYMENTS-00") {
			t.Errorf("counterexample step %q is not a transition ID", id)
		}
	}
	if good := invariant(t, report.Modules[0], "INV-PAYMENTS-002"); good.Status != StatusPass {
		t.Errorf("the intact invariant must pass (a terminal state is not a deadlock violation): %+v", good)
	}
}

// The evidence file engquint writes is read by the trace pipeline: a pass is
// analysis evidence, and changing the module afterwards makes it stale.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func TestEvidenceFileIsReadByTheCodescanAndTurnsStaleWhenTheModuleChanges(t *testing.T) {
	e := newEnv(t)
	mustCheck(t, e.opts())
	bundle, err := model.LoadBundle(e.manifest)
	if err != nil {
		t.Fatal(err)
	}
	formal := func() (codescan.VerificationCheck, bool) {
		checks, diags := codescan.InferVerificationChecks(bundle, bundle.Requirements, nil, "")
		stale := false
		for _, d := range diags {
			stale = stale || d.Code == "verification.formal_stale"
		}
		for _, c := range checks {
			if c.Kind == "formal" {
				return c, stale
			}
		}
		t.Fatalf("no formal check in %+v", checks)
		return codescan.VerificationCheck{}, false
	}

	c, stale := formal()
	if stale || c.Method != "analysis" || c.Status != "pass" {
		t.Fatalf("fresh evidence: stale=%v check %+v", stale, c)
	}
	passed := 0
	for _, r := range c.Results {
		if r.Status == "pass" && strings.HasPrefix(r.Requirement, "REQ-PAYMENTS-00") {
			passed++
		}
	}
	if passed != 2 {
		t.Fatalf("want evidence for both requirements: %+v", c.Results)
	}

	module := filepath.Join(e.outDir, "sm_payments_auth.qnt")
	writeFile(t, module, readFile(t, module)+"// changed after the check\n", 0o644)
	c, stale = formal()
	if !stale || c.Status == "pass" {
		t.Fatalf("a changed module must make the result stale: stale=%v check %+v", stale, c)
	}
	for _, r := range c.Results {
		if r.Status == "pass" {
			t.Errorf("stale result counted as a pass: %+v", r)
		}
	}
}
