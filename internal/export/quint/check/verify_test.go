// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// provision creates a fake pinned Apalache distribution and returns the
// QUINT_HOME that holds it.
func provision(t *testing.T, e env) string {
	t.Helper()
	home := filepath.Join(e.dir, "quint-home")
	writeFile(t, apalacheMC(home), "#!/bin/sh\n", 0o755)
	return home
}

// fakeJava puts a fake java first on PATH; versionOutput is what java -version
// prints on stderr.
func fakeJava(t *testing.T, e env, versionOutput string) {
	t.Helper()
	dir := filepath.Join(e.dir, "javabin")
	writeFile(t, filepath.Join(dir, "java"), "#!/bin/sh\ncat >&2 <<'EOF'\n"+versionOutput+"\nEOF\n", 0o755)
	t.Setenv("PATH", dir+string(os.PathListSeparator)+os.Getenv("PATH"))
}

const java21 = `Picked up JAVA_TOOL_OPTIONS: -Dfoo=bar
openjdk version "21.0.13" 2026-10-20 LTS`

// TRLC-LINKS: REQ-EMG-144
func TestVerifyModePassesThePinnedApalacheAndNoNetworkSettings(t *testing.T) {
	e := newEnv(t)
	home := provision(t, e)
	fakeJava(t, e, java21)
	t.Setenv("HTTPS_PROXY", "http://proxy.example:3128")
	t.Setenv("NO_PROXY", "")
	t.Setenv("no_proxy", "corp.example")
	opts := e.opts()
	opts.Mode, opts.QuintHome = ModeVerify, home
	report := mustCheck(t, opts)
	if report.Failed() {
		t.Fatalf("report %+v", report.Modules)
	}
	if len(e.calls(t, "run")) != 0 {
		t.Error("run mode commands ran in verify mode")
	}
	verifies := e.calls(t, "verify")
	if len(verifies) != 2 {
		t.Fatalf("want one verify per invariant, got %v", verifies)
	}
	ports := map[string]bool{}
	for i, v := range verifies {
		for _, want := range []string{"main=sm_payments_auth", "steps=10", "QUINT_HOME=" + home, "NO_PROXY=localhost,127.0.0.1", "no_proxy=corp.example,localhost,127.0.0.1", "config="} {
			if !strings.Contains(v, want) {
				t.Errorf("verify %d lacks %q: %s", i, want, v)
			}
		}
		if strings.Contains(v, "apalache-version") {
			t.Errorf("--apalache-version passed: %s", v)
		}
		endpoint := v[strings.Index(v, "endpoint="):strings.Index(v, " config=")]
		if ports[endpoint] {
			t.Errorf("port reused: %s", endpoint)
		}
		ports[endpoint] = true
		if !strings.Contains(v, "inv=inv_payments_00"+string(rune('1'+i))) {
			t.Errorf("verify %d invariant: %s", i, v)
		}
	}
	inv := invariant(t, report.Modules[0], "INV-PAYMENTS-001")
	if inv.Verify == nil || inv.Verify.Status != StatusPass || inv.Run != nil {
		t.Fatalf("invariant %+v", inv)
	}
	if got := evidenceOf(t, e); !strings.Contains(got, `"mode": "verify"`) || !strings.Contains(got, "quint verify, apalache 0.62.4") {
		t.Errorf("evidence:\n%s", got)
	}
}

// TRLC-LINKS: REQ-EMG-144
func TestVerifyModeMapsCounterexampleToTransitionIDs(t *testing.T) {
	e := newEnv(t)
	home := provision(t, e)
	fakeJava(t, e, java21)
	t.Setenv("FAKE_QUINT_VIOLATE", "inv_payments_002")
	t.Setenv("FAKE_QUINT_SEQUENCE", "TR-PAYMENTS-001 TR-PAYMENTS-002")
	opts := e.opts()
	opts.Mode, opts.QuintHome = ModeBoth, home
	report := mustCheck(t, opts)
	inv := invariant(t, report.Modules[0], "INV-PAYMENTS-002")
	if !report.Failed() || inv.Verify.Status != StatusViolated || strings.Join(inv.Verify.Counterexample, ">") != "TR-PAYMENTS-001>TR-PAYMENTS-002" {
		t.Fatalf("invariant %+v verify %+v", inv, inv.Verify)
	}
	if inv.Run.Status != StatusViolated || inv.Status != StatusViolated {
		t.Errorf("run %+v status %s", inv.Run, inv.Status)
	}
	if got := report.Modules[0].Modes; strings.Join(got, ",") != "run,verify" {
		t.Errorf("modes %v", got)
	}
}

// TRLC-LINKS: REQ-EMG-142, REQ-EMG-144
func TestVerifyModeMissingApalacheIsASetupErrorAndNothingRuns(t *testing.T) {
	e := newEnv(t)
	fakeJava(t, e, java21)
	opts := e.opts()
	opts.Mode = ModeVerify
	opts.QuintHome = filepath.Join(e.dir, "empty-home")
	if err := os.MkdirAll(opts.QuintHome, 0o755); err != nil {
		t.Fatal(err)
	}
	_, err := Check(opts)
	if !IsSetupError(err) || !strings.Contains(err.Error(), "scripts/install-quint-toolchain.sh") || !strings.Contains(err.Error(), "0.62.4") {
		t.Fatalf("missing dist: %v", err)
	}
	opts.QuintHome = ""
	t.Setenv("QUINT_HOME", "")
	if _, err := Check(opts); !IsSetupError(err) || !strings.Contains(err.Error(), "QUINT_HOME") {
		t.Fatalf("no QUINT_HOME: %v", err)
	}
	if got := e.calls(t, "verify"); len(got) != 0 {
		t.Errorf("quint verify ran without a provisioned Apalache (it would download): %v", got)
	}
	if _, statErr := os.Stat(filepath.Join(e.outDir, "results.json")); statErr == nil {
		t.Error("a setup error wrote results")
	}
}

// TRLC-LINKS: REQ-EMG-142, REQ-EMG-144
func TestVerifyModeJavaTooOldOrMissingIsASetupError(t *testing.T) {
	e := newEnv(t)
	home := provision(t, e)
	opts := e.opts()
	opts.Mode, opts.QuintHome = ModeBoth, home

	fakeJava(t, e, `openjdk version "17.0.9" 2023-10-17`)
	_, err := Check(opts)
	if !IsSetupError(err) || !strings.Contains(err.Error(), "java 21 or later") || !strings.Contains(err.Error(), "found java 17") {
		t.Fatalf("java 17: %v", err)
	}

	fakeJava(t, e, `java version "1.8.0_402"`)
	if _, err := Check(opts); !IsSetupError(err) || !strings.Contains(err.Error(), "found java 8") {
		t.Fatalf("java 8: %v", err)
	}

	t.Setenv("PATH", t.TempDir())
	t.Setenv("JAVA_HOME", "")
	if _, err := Check(opts); !IsSetupError(err) || !strings.Contains(err.Error(), "java 21 or later") {
		t.Fatalf("no java: %v", err)
	}
	if len(e.calls(t, "run"))+len(e.calls(t, "verify"))+len(e.calls(t, "typecheck")) != 0 {
		t.Error("quint ran although java is unusable")
	}
}

// Run mode never needs Apalache or Java.
//
// TRLC-LINKS: REQ-EMG-143, REQ-EMG-144
func TestRunModeNeedsNeitherApalacheNorJava(t *testing.T) {
	e := newEnv(t)
	t.Setenv("PATH", t.TempDir())
	t.Setenv("QUINT_HOME", "")
	if report := mustCheck(t, e.opts()); report.Failed() {
		t.Fatalf("report %+v", report.Modules)
	}
}

// TRLC-LINKS: REQ-EMG-144
func TestJavaMajorParsesModernAndLegacyVersions(t *testing.T) {
	for output, want := range map[string]int{
		`openjdk version "21.0.13" 2026-10-20 LTS`:                   21,
		`openjdk version "21" 2023-09-19`:                            21,
		`java version "1.8.0_402"`:                                   8,
		`openjdk version "17.0.2-ea"`:                                17,
		"Picked up JAVA_TOOL_OPTIONS: x\nopenjdk version \"22.0.1\"": 22,
	} {
		if got, err := javaMajor(output); err != nil || got != want {
			t.Errorf("%q: got %d, %v; want %d", output, got, err, want)
		}
	}
	if _, err := javaMajor("not java"); err == nil {
		t.Error("garbage parsed")
	}
}

// TRLC-LINKS: REQ-EMG-144
func TestVerifyEnvAppendsNoProxyAndSetsScratchDirectories(t *testing.T) {
	base := []string{"PATH=/usr/bin", "HTTP_PROXY=http://p", "NO_PROXY=a.example", "TMPDIR=/tmp"}
	got := map[string]string{}
	for _, kv := range verifyEnv(base, "/q/home", "/scratch", "/jdk/bin") {
		k, v, _ := strings.Cut(kv, "=")
		got[k] = v
	}
	for k, want := range map[string]string{
		"QUINT_HOME": "/q/home", "TMPDIR": "/scratch", "PATH": "/jdk/bin:/usr/bin",
		"NO_PROXY": "a.example,localhost,127.0.0.1", "no_proxy": "localhost,127.0.0.1",
	} {
		if got[k] != want {
			t.Errorf("%s=%q, want %q", k, got[k], want)
		}
	}
	// Without a proxy the environment is left alone.
	for _, kv := range verifyEnv([]string{"PATH=/usr/bin"}, "/h", "/s", "") {
		if strings.HasPrefix(kv, "NO_PROXY=") || strings.HasPrefix(kv, "no_proxy=") {
			t.Errorf("NO_PROXY set without a proxy: %s", kv)
		}
	}
}
