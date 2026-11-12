// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"context"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"
	"unicode/utf8"
)

const okRunOutput = "[ok] No violation found (1ms at 1 traces/second).\nTrace length statistics: max=3, min=2, average=2.50\n"

// stubRunner answers --version and typecheck and hands every other command to
// handle.
func stubRunner(handle func(ctx context.Context, cmd Command) Output) Runner {
	return func(ctx context.Context, cmd Command) (Output, error) {
		switch cmd.Args[0] {
		case "--version":
			return Output{Stdout: "9.9.9\n"}, nil
		case "typecheck":
			return Output{}, nil
		}
		return handle(ctx, cmd), nil
	}
}

// evidenceOnDisk returns how many entries the evidence file holds.
func evidenceOnDisk(t *testing.T, e env) int {
	t.Helper()
	var parsed struct {
		Results []json.RawMessage `json:"results"`
	}
	if err := json.Unmarshal([]byte(evidenceOf(t, e)), &parsed); err != nil {
		t.Fatalf("evidence is not JSON: %v\n%s", err, evidenceOf(t, e))
	}
	if parsed.Results == nil {
		t.Fatalf("evidence has no results array:\n%s", evidenceOf(t, e))
	}
	return len(parsed.Results)
}

// REQ-EMG-143: the overall deadline kills the running tool and ends the check
// with a timeout error that is neither a setup error nor a pass, and writes
// nothing.
//
// TRLC-LINKS: REQ-EMG-143
func TestCheckOverallDeadlineEndsTheCheckWithATimeout(t *testing.T) {
	e := newEnv(t)
	opts := e.opts()
	opts.Timeout = 150 * time.Millisecond
	var calls int
	opts.Runner = stubRunner(func(ctx context.Context, cmd Command) Output {
		calls++
		<-ctx.Done()
		return Output{ExitCode: -1, TimedOut: true}
	})
	start := time.Now()
	_, err := Check(opts)
	if !errors.Is(err, ErrTimeout) || IsSetupError(err) || !strings.Contains(err.Error(), "--timeout") {
		t.Fatalf("err %v", err)
	}
	if calls != 1 || time.Since(start) > 3*time.Second {
		t.Fatalf("calls %d after %s: the check went on after the deadline", calls, time.Since(start))
	}
	if _, statErr := os.Stat(filepath.Join(e.outDir, "results.json")); statErr == nil {
		t.Error("results written after a timeout")
	}
}

// TRLC-LINKS: REQ-EMG-143
func TestCheckStopsWhenItsContextIsCancelled(t *testing.T) {
	e := newEnv(t)
	opts := e.opts()
	ctx, cancel := context.WithCancel(context.Background())
	opts.Context = ctx
	opts.Runner = stubRunner(func(context.Context, Command) Output {
		cancel()
		return Output{ExitCode: -1}
	})
	_, err := Check(opts)
	if !errors.Is(err, context.Canceled) || errors.Is(err, ErrTimeout) {
		t.Fatalf("err %v", err)
	}
}

// A call that hits its own time limit is a verdict (an error step with a clear
// message and no pass), the limits reach the runner, and the next call goes on.
//
// TRLC-LINKS: REQ-EMG-143
func TestCheckCallTimeoutAndOutputLimitReachTheRunnerAndFailTheStep(t *testing.T) {
	e := newEnv(t)
	opts := e.opts()
	opts.CallTimeout, opts.MaxOutput = 7*time.Second, 4096
	var mu sync.Mutex
	var limits []Command
	opts.Runner = stubRunner(func(_ context.Context, cmd Command) Output {
		mu.Lock()
		defer mu.Unlock()
		limits = append(limits, cmd)
		switch len(limits) {
		case 1: // the witnesses run
			return Output{Stdout: okRunOutput}
		case 2:
			return Output{ExitCode: -1, TimedOut: true, Stderr: "killed: context deadline exceeded"}
		}
		return Output{Stdout: okRunOutput, Truncated: true}
	})
	report := mustCheck(t, opts)
	for _, cmd := range limits {
		if cmd.Timeout != 7*time.Second || cmd.MaxOutput != 4096 {
			t.Errorf("limits not passed on: %+v", cmd)
		}
	}
	m := report.Modules[0]
	first, second := invariant(t, m, "INV-PAYMENTS-001"), invariant(t, m, "INV-PAYMENTS-002")
	if !report.Failed() || first.Status != StatusError || !strings.Contains(first.Run.Message, "time limit") {
		t.Fatalf("timed out step %+v", first.Run)
	}
	if second.Status != StatusError || !strings.Contains(second.Run.Message, "capture limit") {
		t.Fatalf("truncated step must not pass although it printed the ok line: %+v", second.Run)
	}
	if report.EvidenceEntries != 0 {
		t.Errorf("an error is no evidence: %d", report.EvidenceEntries)
	}
}

// TRLC-LINKS: REQ-EMG-142
func TestCheckRejectsAbsurdBounds(t *testing.T) {
	for name, mutate := range map[string]func(*CheckOptions){
		"max-steps":    func(o *CheckOptions) { o.MaxSteps = MaxStepsLimit + 1 },
		"max-samples":  func(o *CheckOptions) { o.MaxSamples = MaxSamplesLimit + 1 },
		"verify-steps": func(o *CheckOptions) { o.VerifySteps = MaxVerifyStepsLimit + 1 },
	} {
		opts := CheckOptions{ModelPath: "engmod.yml"}
		mutate(&opts)
		if _, err := Check(opts); !IsSetupError(err) || !strings.Contains(err.Error(), name) {
			t.Errorf("%s: %v", name, err)
		}
	}
	at := CheckOptions{ModelPath: "engmod.yml", MaxSteps: MaxStepsLimit, MaxSamples: MaxSamplesLimit, VerifySteps: MaxVerifyStepsLimit}
	if _, err := withDefaults(at); err != nil {
		t.Errorf("the maxima themselves are allowed: %v", err)
	}
	if _, err := withDefaults(CheckOptions{ModelPath: "engmod.yml", Timeout: -1}); !IsSetupError(err) {
		t.Errorf("negative timeout: %v", err)
	}
}

// Secrets of the caller's environment never reach the tools: only the
// allow-list and the names PassEnv asks for do; proxy variables reach verify
// calls only.
//
// TRLC-LINKS: REQ-EMG-143, REQ-EMG-144
func TestCheckPassesOnlyAllowListedEnvironment(t *testing.T) {
	e := newEnv(t)
	t.Setenv("GITHUB_TOKEN", "ghp_secret")
	t.Setenv("AWS_SECRET_ACCESS_KEY", "aws_secret")
	t.Setenv("NODE_OPTIONS", "--require /tmp/evil.js")
	t.Setenv("JAVA_TOOL_OPTIONS", "-javaagent:/tmp/evil.jar")
	t.Setenv("HTTPS_PROXY", "http://proxy.example:3128")
	t.Setenv("CUSTOM_ASKED_FOR", "yes")
	t.Setenv("LANG", "C.UTF-8")
	home := provision(t, e)
	fakeJava(t, e, java21)

	var mu sync.Mutex
	envs := map[string][]string{}
	opts := e.opts()
	opts.Mode, opts.QuintHome = ModeBoth, home
	opts.PassEnv = append(opts.PassEnv, "CUSTOM_ASKED_FOR")
	opts.Runner = func(_ context.Context, cmd Command) (Output, error) {
		mu.Lock()
		defer mu.Unlock()
		sub := cmd.Args[0]
		if strings.HasSuffix(cmd.Path, "java") {
			sub = "java"
		}
		envs[sub] = cmd.Env
		switch sub {
		case "java":
			return Output{Stderr: java21}, nil
		case "--version":
			return Output{Stdout: "9.9.9\n"}, nil
		case "run", "verify":
			return Output{Stdout: okRunOutput}, nil
		}
		return Output{}, nil
	}
	mustCheck(t, opts)

	for _, sub := range []string{"java", "--version", "typecheck", "run", "verify"} {
		env := strings.Join(envs[sub], "\n")
		if envs[sub] == nil {
			t.Fatalf("no %s call recorded", sub)
		}
		for _, banned := range []string{"GITHUB_TOKEN", "AWS_SECRET", "NODE_OPTIONS", "JAVA_TOOL_OPTIONS", "ghp_secret"} {
			if strings.Contains(env, banned) {
				t.Errorf("%s call inherited %s:\n%s", sub, banned, env)
			}
		}
		for _, want := range []string{"PATH=", "LANG=C.UTF-8"} {
			if !strings.Contains(env, want) {
				t.Errorf("%s call lacks %s", sub, want)
			}
		}
	}
	for _, sub := range []string{"run", "typecheck"} {
		if env := strings.Join(envs[sub], "\n"); !strings.Contains(env, "CUSTOM_ASKED_FOR=yes") || strings.Contains(env, "PROXY") {
			t.Errorf("%s env should hold PassEnv names and no proxy:\n%s", sub, env)
		}
	}
	verify := strings.Join(envs["verify"], "\n")
	for _, want := range []string{"HTTPS_PROXY=http://proxy.example:3128", "NO_PROXY=localhost,127.0.0.1", "QUINT_HOME=" + home} {
		if !strings.Contains(verify, want) {
			t.Errorf("verify env lacks %s:\n%s", want, verify)
		}
	}
}

// TRLC-LINKS: REQ-EMG-143
func TestAllowEnvKeepsOnlyNamedVariables(t *testing.T) {
	environ := []string{"PATH=/bin", "HOME=/h", "TOKEN=x", "NO_PROXY=a", "HTTP_PROXY=p", "EXTRA=1", "BROKEN"}
	got := strings.Join(allowEnv(environ, []string{"EXTRA"}, false), " ")
	if got != "PATH=/bin HOME=/h EXTRA=1" {
		t.Errorf("without proxies: %q", got)
	}
	got = strings.Join(allowEnv(environ, nil, true), " ")
	if got != "PATH=/bin HOME=/h NO_PROXY=a HTTP_PROXY=p" {
		t.Errorf("with proxies: %q", got)
	}
}

// PATH is found under any spelling on Windows, Java goes first, an unset PATH
// never gets an empty element, and the Java launcher is java.exe there.
//
// TRLC-LINKS: REQ-EMG-144
func TestVerifyEnvPathHandling(t *testing.T) {
	sep := string(os.PathListSeparator)
	got := strings.Join(buildVerifyEnv(true, []string{"Path=C:\\bin", "http_proxy=p", "No_Proxy=x"}, "H", "T", "C:\\jdk"), "|")
	for _, want := range []string{"Path=C:\\jdk" + sep + "C:\\bin", "No_Proxy=x,localhost,127.0.0.1"} {
		if !strings.Contains(got, want) {
			t.Errorf("windows env lacks %q: %s", want, got)
		}
	}
	if strings.Count(strings.ToUpper(got), "PATH=") != 1 || strings.Count(strings.ToUpper(got), "NO_PROXY=") != 1 {
		t.Errorf("a name appears under two spellings: %s", got)
	}
	got = strings.Join(buildVerifyEnv(false, []string{"HOME=/h"}, "H", "T", "/jdk/bin"), "|")
	if !strings.Contains(got, "PATH=/jdk/bin|") && !strings.HasSuffix(got, "PATH=/jdk/bin") {
		t.Errorf("unset PATH: %s", got)
	}
	if strings.Contains(got, "PATH=/jdk/bin"+sep) {
		t.Errorf("empty PATH element: %s", got)
	}
	if javaBinary("windows") != "java.exe" || javaBinary("linux") != "java" {
		t.Error("java launcher name")
	}
}

// REQ-EMG-146: the evidence of an earlier pass must not outlive the moment the
// model stops generating, drifts, fails to typecheck or runs vacuously.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func TestCheckClearsEvidenceWhenNothingCanBeTrusted(t *testing.T) {
	passFirst := func(t *testing.T) env {
		e := newEnv(t)
		mustCheck(t, e.opts())
		if evidenceOnDisk(t, e) != 3 {
			t.Fatalf("setup: want 3 entries\n%s", evidenceOf(t, e))
		}
		return e
	}
	behavior := func(e env) string { return filepath.Join(filepath.Dir(e.manifest), "model", "behavior.yml") }

	t.Run("generation error", func(t *testing.T) {
		e := passFirst(t)
		writeFile(t, behavior(e), strings.Replace(readFile(t, behavior(e)), "guard: attempts < 3\n      effect", "guard: attempts <\n      effect", 1), 0o644)
		report := mustCheck(t, e.opts())
		if len(report.GenerationErrors) == 0 || evidenceOnDisk(t, e) != 0 || report.EvidencePath == "" {
			t.Fatalf("report %+v\n%s", report, evidenceOf(t, e))
		}
		if !strings.Contains(evidenceOf(t, e), `"quintVersion": "0.33.1"`) {
			t.Errorf("the tool version of the cleared file is kept:\n%s", evidenceOf(t, e))
		}
	})
	t.Run("drift", func(t *testing.T) {
		e := passFirst(t)
		module := filepath.Join(e.outDir, "sm_payments_auth.qnt")
		writeFile(t, module, readFile(t, module)+"// edit\n", 0o644)
		mustCheck(t, e.opts())
		if evidenceOnDisk(t, e) != 0 {
			t.Fatalf("stale pass survived drift:\n%s", evidenceOf(t, e))
		}
	})
	t.Run("drift with no module left to check", func(t *testing.T) {
		e := passFirst(t)
		writeFile(t, behavior(e), "schemaVersion: 2\nbehavior:\n  machines: []\n", 0o644)
		report := mustCheck(t, e.opts())
		if len(report.Drift) != 1 || report.Drift[0].Kind != "extra" || evidenceOnDisk(t, e) != 0 {
			t.Fatalf("drift %+v\n%s", report.Drift, evidenceOf(t, e))
		}
	})
	t.Run("model without machines", func(t *testing.T) {
		e := passFirst(t)
		writeFile(t, behavior(e), "schemaVersion: 2\nbehavior:\n  machines: []\n", 0o644)
		if err := os.Remove(filepath.Join(e.outDir, "sm_payments_auth.qnt")); err != nil {
			t.Fatal(err)
		}
		report := mustCheck(t, e.opts())
		if !report.NoMachines || evidenceOnDisk(t, e) != 0 {
			t.Fatalf("report %+v\n%s", report, evidenceOf(t, e))
		}
		// A second run has nothing left to clear.
		if report = mustCheck(t, e.opts()); report.EvidencePath != "" {
			t.Errorf("an already empty file was reported as rewritten")
		}
	})
	t.Run("type error", func(t *testing.T) {
		e := passFirst(t)
		t.Setenv("FAKE_QUINT_TYPE_ERROR", "1")
		mustCheck(t, e.opts())
		if evidenceOnDisk(t, e) != 0 {
			t.Fatalf("stale pass survived a type error:\n%s", evidenceOf(t, e))
		}
	})
	t.Run("vacuous run", func(t *testing.T) {
		e := passFirst(t)
		t.Setenv("FAKE_QUINT_TRACE_MAX", "0")
		mustCheck(t, e.opts())
		if evidenceOnDisk(t, e) != 0 {
			t.Fatalf("stale pass survived a vacuous run:\n%s", evidenceOf(t, e))
		}
	})
	t.Run("no evidence file is created for a model that never had one", func(t *testing.T) {
		e := newEnv(t)
		writeFile(t, behavior(e), strings.Replace(readFile(t, behavior(e)), "guard: attempts < 3\n      effect", "guard: attempts <\n      effect", 1), 0o644)
		mustCheck(t, e.opts())
		if _, err := os.Stat(filepath.Join(filepath.Dir(e.manifest), "test-results", "quint-formal.json")); err == nil {
			t.Error("evidence created for a model that does not generate")
		}
	})
	t.Run("NoEvidence leaves the file alone", func(t *testing.T) {
		e := passFirst(t)
		writeFile(t, behavior(e), "schemaVersion: 2\nbehavior:\n  machines: []\n", 0o644)
		opts := e.opts()
		opts.NoEvidence = true
		mustCheck(t, opts)
		if evidenceOnDisk(t, e) != 3 {
			t.Error("--no-evidence rewrote the evidence file")
		}
	})
}

// A checkout that converted the module to CRLF is neither drift nor a
// different hash: both use the LF text, the rule the trace's evidence reader
// applies too.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func TestCheckNormalisesCRLFBeforeDriftAndHash(t *testing.T) {
	e := newEnv(t)
	lf := mustCheck(t, e.opts())
	module := filepath.Join(e.outDir, "sm_payments_auth.qnt")
	text := readFile(t, module)
	writeFile(t, module, strings.ReplaceAll(text, "\n", "\r\n"), 0o644)
	crlf := mustCheck(t, e.opts())
	if len(crlf.Drift) != 0 || crlf.Failed() {
		t.Fatalf("CRLF read as drift: %+v", crlf.Drift)
	}
	if crlf.Modules[0].QntSha256 == "" || crlf.Modules[0].QntSha256 != lf.Modules[0].QntSha256 || crlf.Modules[0].QntSha256 != sha256Hex([]byte(text)) {
		t.Errorf("hash %s differs from the LF hash %s", crlf.Modules[0].QntSha256, lf.Modules[0].QntSha256)
	}
	if got := string(normalizeEOL([]byte("a\r\nb\rc\n\r\n"))); got != "a\nb\rc\n\n" {
		t.Errorf("normalizeEOL = %q: only CRLF changes", got)
	}
	// A real edit is still drift.
	writeFile(t, module, strings.ReplaceAll(text+"// edit\n", "\n", "\r\n"), 0o644)
	if edited := mustCheck(t, e.opts()); len(edited.Drift) != 1 {
		t.Errorf("an edit hidden by CRLF: %+v", edited.Drift)
	}
}

// A violation whose trace file cannot be read says so; only a readable trace
// that holds just the initial state blames the initial state.
//
// TRLC-LINKS: REQ-EMG-143
func TestCounterexampleDistinguishesAnUnreadableTraceFromTheInitialState(t *testing.T) {
	violation := Output{ExitCode: 1, Stdout: "[violation] Found an issue (8ms).\n"}
	dir := t.TempDir()

	missing := stepFromOutput(violation, filepath.Join(dir, "no-such.itf.json"))
	if missing.Status != StatusViolated || !missing.TraceUnavailable || !strings.Contains(missing.Message, "readable trace") {
		t.Fatalf("missing trace %+v", missing)
	}
	if got := FormatCounterexample(missing); !strings.Contains(got, "trace unavailable") || strings.Contains(got, "initial state") {
		t.Errorf("format %q", got)
	}

	for name, body := range map[string]string{
		"garbage":            "not json",
		"no states":          `{"states":[]}`,
		"steps without ids":  `{"states":[{"a":1},{"a":2}]}`,
		"transition not str": `{"states":[{"a":1},{"lastTransition":5}]}`,
	} {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".json")
		writeFile(t, path, body, 0o644)
		if step := stepFromOutput(violation, path); !step.TraceUnavailable {
			t.Errorf("%s: %+v", name, step)
		}
	}

	path := filepath.Join(dir, "initial.json")
	writeFile(t, path, `{"states":[{"lastTransition":"init"}]}`, 0o644)
	initial := stepFromOutput(violation, path)
	if initial.TraceUnavailable || len(initial.Counterexample) != 0 || initial.Message != "" {
		t.Fatalf("initial-only trace %+v", initial)
	}
	if got := FormatCounterexample(initial); !strings.Contains(got, "initial state violates") {
		t.Errorf("format %q", got)
	}

	// The flag reaches results.json and the evidence note.
	data, _ := json.Marshal(missing)
	if !strings.Contains(string(data), `"traceUnavailable":true`) {
		t.Errorf("json %s", data)
	}
	notes := evidenceNotes(ModuleResult{Machine: "SM-X"}, InvariantResult{ID: "INV-X", Status: StatusViolated, Run: &missing})
	if !strings.Contains(notes, "trace unavailable") {
		t.Errorf("notes %q", notes)
	}
}

// TRLC-LINKS: REQ-EMG-143
func TestErrorMessageTruncatesOnARuneBoundary(t *testing.T) {
	for _, filler := range []string{"é", "日", "😀", "a"} {
		long := strings.Repeat(filler, 300)
		msg := errorMessage(Output{ExitCode: 1, Stderr: "error: " + long})
		if !utf8.ValidString(msg) {
			t.Errorf("%q cut inside a rune", filler)
		}
		prefix := "quint exited with code 1: "
		if body := strings.TrimPrefix(msg, prefix); len(body) > maxMessageBytes || len(body) < maxMessageBytes-3 {
			t.Errorf("%q: kept %d bytes", filler, len(body))
		}
	}
	if got := truncateUTF8("short", 200); got != "short" {
		t.Errorf("short text changed: %q", got)
	}
}

// Writes never follow a symlink: a link planted at results.json, at the
// evidence file or in place of a module is refused, and the file it points at
// stays as it was.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-149
func TestWritesRefuseSymlinksAndReplaceAtomically(t *testing.T) {
	dir := t.TempDir()
	victim := filepath.Join(dir, "victim.txt")
	writeFile(t, victim, "precious", 0o644)
	link := filepath.Join(dir, "link.json")
	if err := os.Symlink(victim, link); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	if _, err := WriteFileIfChanged(link, []byte("pwned")); err == nil || !strings.Contains(err.Error(), "symlink") || !strings.Contains(err.Error(), "refusing") {
		t.Fatalf("symlink target: %v", err)
	}
	if readFile(t, victim) != "precious" {
		t.Fatal("the write went through the symlink")
	}
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteFileIfChanged(filepath.Join(dir, "sub"), []byte("x")); err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("directory target: %v", err)
	}

	target := filepath.Join(dir, "out.json")
	if wrote, err := WriteFileIfChanged(target, []byte("one")); err != nil || !wrote {
		t.Fatalf("create: %v %v", wrote, err)
	}
	if info, _ := os.Stat(target); info.Mode().Perm() != 0o644 {
		t.Errorf("mode %v", info.Mode())
	}
	before, _ := os.Stat(target)
	if wrote, err := WriteFileIfChanged(target, []byte("one")); err != nil || wrote {
		t.Fatalf("an identical file is not rewritten: %v %v", wrote, err)
	}
	after, _ := os.Stat(target)
	if !os.SameFile(before, after) {
		t.Error("identical content replaced the file")
	}
	if wrote, err := WriteFileIfChanged(target, []byte("two")); err != nil || !wrote || readFile(t, target) != "two" {
		t.Fatalf("update: %v %v", wrote, err)
	}
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", entry.Name())
		}
	}
}

// TRLC-LINKS: REQ-EMG-145, REQ-EMG-149
func TestCheckRefusesToWriteResultsThroughASymlinkAndToReadALinkedModule(t *testing.T) {
	e := newEnv(t)
	victim := filepath.Join(e.dir, "victim.txt")
	writeFile(t, victim, "precious", 0o644)
	results := filepath.Join(e.outDir, "results.json")
	if err := os.Symlink(victim, results); err != nil {
		t.Skipf("no symlinks here: %v", err)
	}
	_, err := Check(e.opts())
	if err == nil || IsSetupError(err) || !strings.Contains(err.Error(), "refusing to write") {
		t.Fatalf("err %v", err)
	}
	if readFile(t, victim) != "precious" {
		t.Fatal("results were written through the symlink")
	}
	os.Remove(results)
	os.Remove(e.log)

	module := filepath.Join(e.outDir, "sm_payments_auth.qnt")
	real := filepath.Join(e.dir, "real.qnt")
	if err := os.Rename(module, real); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(real, module); err != nil {
		t.Skip(err)
	}
	if _, err := Check(e.opts()); !IsSetupError(err) || !strings.Contains(err.Error(), "not a regular file") {
		t.Fatalf("a linked module must be refused: %v", err)
	}
	if len(e.calls(t, "run"))+len(e.calls(t, "typecheck")) != 0 {
		t.Error("quint ran on a symlinked module")
	}
}

// One helper decides what a generated module is, for the check and the CLI.
//
// TRLC-LINKS: REQ-EMG-149
func TestIsGeneratedModule(t *testing.T) {
	if !IsGeneratedModule([]byte("// Generated by engineering-model-go from SM-X; do not edit.\nmodule x {}\n")) {
		t.Error("generated header not recognised")
	}
	for _, text := range []string{"", "// hand written\nmodule x {}\n", " // Generated by engineering-model-go from SM-X"} {
		if IsGeneratedModule([]byte(text)) {
			t.Errorf("%q recognised", text)
		}
	}
}
