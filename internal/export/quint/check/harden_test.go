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
	checkLimitVerdicts(t, report)
}

// checkLimitVerdicts checks that a timed out and a truncated call are error
// steps, never a pass, and give no evidence.
func checkLimitVerdicts(t *testing.T, report CheckReport) {
	t.Helper()
	m := report.Modules[0]
	first, second := invariant(t, m, "INV-PAYMENTS-001"), invariant(t, m, "INV-PAYMENTS-002")
	if !report.Failed() || first.Status != StatusError {
		t.Fatalf("timed out step %+v", first.Run)
	}
	if !strings.Contains(first.Run.Message, "time limit") {
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

	envs := &recordedEnvs{byCall: map[string][]string{}}
	opts := e.opts()
	opts.Mode, opts.QuintHome = ModeBoth, home
	opts.PassEnv = append(opts.PassEnv, "CUSTOM_ASKED_FOR")
	opts.Runner = envs.runner()
	mustCheck(t, opts)

	for _, sub := range []string{"java", "--version", "typecheck", "run", "verify"} {
		checkNoSecretsIn(t, sub, envs.byCall[sub])
	}
	for _, sub := range []string{"run", "typecheck"} {
		env := strings.Join(envs.byCall[sub], "\n")
		requireAll(t, env, "CUSTOM_ASKED_FOR=yes")
		requireNone(t, env, "PROXY")
	}
	requireAll(t, strings.Join(envs.byCall["verify"], "\n"), "HTTPS_PROXY=http://proxy.example:3128", "NO_PROXY=localhost,127.0.0.1", "QUINT_HOME="+home)
}

// recordedEnvs records the environment of the last call of each kind.
type recordedEnvs struct {
	mu     sync.Mutex
	byCall map[string][]string
}

// runner answers the calls of a check in both modes and records their
// environments under java, --version, typecheck, run or verify.
func (r *recordedEnvs) runner() Runner {
	return func(_ context.Context, cmd Command) (Output, error) {
		r.mu.Lock()
		defer r.mu.Unlock()
		sub := cmd.Args[0]
		if strings.HasSuffix(cmd.Path, "java") {
			sub = "java"
		}
		r.byCall[sub] = cmd.Env
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
}

// checkNoSecretsIn checks that a recorded call ran with the allow-listed
// environment only.
func checkNoSecretsIn(t *testing.T, sub string, environ []string) {
	t.Helper()
	if environ == nil {
		t.Fatalf("no %s call recorded", sub)
	}
	env := strings.Join(environ, "\n")
	requireNone(t, env, "GITHUB_TOKEN", "AWS_SECRET", "NODE_OPTIONS", "JAVA_TOOL_OPTIONS", "ghp_secret")
	requireAll(t, env, "PATH=", "LANG=C.UTF-8")
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
	checkWindowsVerifyEnv(t, sep)
	got := strings.Join(buildVerifyEnv(false, []string{"HOME=/h"}, verifyDirs{quintHome: "H", tmpDir: "T", javaDir: "/jdk/bin"}), "|")
	if !strings.Contains(got, "PATH=/jdk/bin|") && !strings.HasSuffix(got, "PATH=/jdk/bin") {
		t.Errorf("unset PATH: %s", got)
	}
	requireNone(t, got, "PATH=/jdk/bin"+sep)
	if javaBinary("windows") != "java.exe" || javaBinary("linux") != "java" {
		t.Error("java launcher name")
	}
}

func checkWindowsVerifyEnv(t *testing.T, sep string) {
	t.Helper()
	got := strings.Join(buildVerifyEnv(true, []string{"Path=C:\\bin", "http_proxy=p", "No_Proxy=x"}, verifyDirs{quintHome: "H", tmpDir: "T", javaDir: "C:\\jdk"}), "|")
	requireAll(t, got, "Path=C:\\jdk"+sep+"C:\\bin", "No_Proxy=x,localhost,127.0.0.1")
	if strings.Count(strings.ToUpper(got), "PATH=") != 1 || strings.Count(strings.ToUpper(got), "NO_PROXY=") != 1 {
		t.Errorf("a name appears under two spellings: %s", got)
	}
}

// REQ-EMG-146: the evidence of an earlier pass must not outlive the moment the
// model stops generating, drifts, fails to typecheck or runs vacuously.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func TestCheckClearsEvidenceWhenNothingCanBeTrusted(t *testing.T) {
	for _, tc := range clearEvidenceCases {
		t.Run(tc.name, tc.run)
	}
}

var clearEvidenceCases = []struct {
	name string
	run  func(t *testing.T)
}{
	{"generation error", evidenceClearedByGenerationError},
	{"drift", evidenceClearedByDrift},
	{"drift with no module left to check", evidenceClearedByDriftWithoutModules},
	{"model without machines", evidenceClearedWithoutMachines},
	{"type error", evidenceClearedByTypeError},
	{"vacuous run", evidenceClearedByVacuousRun},
	{"no evidence file is created for a model that never had one", evidenceNotCreatedWhenNeverThere},
	{"NoEvidence leaves the file alone", evidenceKeptWithNoEvidence},
}

const noMachinesBehavior = "schemaVersion: 3\nbehavior:\n  machines: []\n"

// passFirst runs a passing check, so the evidence file holds three entries.
func passFirst(t *testing.T) env {
	t.Helper()
	e := newEnv(t)
	mustCheck(t, e.opts())
	if evidenceOnDisk(t, e) != 3 {
		t.Fatalf("setup: want 3 entries\n%s", evidenceOf(t, e))
	}
	return e
}

func behaviorPath(e env) string {
	return filepath.Join(filepath.Dir(e.manifest), "model", "behavior.yml")
}

// breakGuard gives a transition guard a syntax error, so the model no longer
// generates.
func breakGuard(t *testing.T, e env) {
	t.Helper()
	path := behaviorPath(e)
	writeFile(t, path, strings.Replace(readFile(t, path), "guard: attempts < 3\n      effect", "guard: attempts <\n      effect", 1))
}

func evidenceClearedByGenerationError(t *testing.T) {
	e := passFirst(t)
	breakGuard(t, e)
	report := mustCheck(t, e.opts())
	if len(report.GenerationErrors) == 0 || evidenceOnDisk(t, e) != 0 || report.EvidencePath == "" {
		t.Fatalf("report %+v\n%s", report, evidenceOf(t, e))
	}
	if !strings.Contains(evidenceOf(t, e), `"quintVersion": "0.33.1"`) {
		t.Errorf("the tool version of the cleared file is kept:\n%s", evidenceOf(t, e))
	}
}

func evidenceClearedByDrift(t *testing.T) {
	e := passFirst(t)
	module := filepath.Join(e.outDir, "sm_payments_auth.qnt")
	writeFile(t, module, readFile(t, module)+"// edit\n")
	mustCheck(t, e.opts())
	if evidenceOnDisk(t, e) != 0 {
		t.Fatalf("stale pass survived drift:\n%s", evidenceOf(t, e))
	}
}

func evidenceClearedByDriftWithoutModules(t *testing.T) {
	e := passFirst(t)
	writeFile(t, behaviorPath(e), noMachinesBehavior)
	report := mustCheck(t, e.opts())
	if len(report.Drift) != 1 || report.Drift[0].Kind != "extra" || evidenceOnDisk(t, e) != 0 {
		t.Fatalf("drift %+v\n%s", report.Drift, evidenceOf(t, e))
	}
}

func evidenceClearedWithoutMachines(t *testing.T) {
	e := passFirst(t)
	writeFile(t, behaviorPath(e), noMachinesBehavior)
	removeFile(t, filepath.Join(e.outDir, "sm_payments_auth.qnt"))
	report := mustCheck(t, e.opts())
	if !report.NoMachines || evidenceOnDisk(t, e) != 0 {
		t.Fatalf("report %+v\n%s", report, evidenceOf(t, e))
	}
	// A second run has nothing left to clear.
	if report = mustCheck(t, e.opts()); report.EvidencePath != "" {
		t.Errorf("an already empty file was reported as rewritten")
	}
}

func evidenceClearedByTypeError(t *testing.T) {
	e := passFirst(t)
	t.Setenv("FAKE_QUINT_TYPE_ERROR", "1")
	mustCheck(t, e.opts())
	if evidenceOnDisk(t, e) != 0 {
		t.Fatalf("stale pass survived a type error:\n%s", evidenceOf(t, e))
	}
}

func evidenceClearedByVacuousRun(t *testing.T) {
	e := passFirst(t)
	t.Setenv("FAKE_QUINT_TRACE_MAX", "0")
	mustCheck(t, e.opts())
	if evidenceOnDisk(t, e) != 0 {
		t.Fatalf("stale pass survived a vacuous run:\n%s", evidenceOf(t, e))
	}
}

func evidenceNotCreatedWhenNeverThere(t *testing.T) {
	e := newEnv(t)
	breakGuard(t, e)
	mustCheck(t, e.opts())
	if _, err := os.Stat(filepath.Join(filepath.Dir(e.manifest), "test-results", "quint-formal.json")); err == nil {
		t.Error("evidence created for a model that does not generate")
	}
}

func evidenceKeptWithNoEvidence(t *testing.T) {
	e := passFirst(t)
	writeFile(t, behaviorPath(e), noMachinesBehavior)
	opts := e.opts()
	opts.NoEvidence = true
	mustCheck(t, opts)
	if evidenceOnDisk(t, e) != 3 {
		t.Error("--no-evidence rewrote the evidence file")
	}
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
	writeFile(t, module, strings.ReplaceAll(text, "\n", "\r\n"))
	crlf := mustCheck(t, e.opts())
	if len(crlf.Drift) != 0 || crlf.Failed() {
		t.Fatalf("CRLF read as drift: %+v", crlf.Drift)
	}
	hash := crlf.Modules[0].QntSha256
	if hash == "" || hash != lf.Modules[0].QntSha256 || hash != sha256Hex([]byte(text)) {
		t.Errorf("hash %s differs from the LF hash %s", hash, lf.Modules[0].QntSha256)
	}
	// A real edit is still drift.
	writeFile(t, module, strings.ReplaceAll(text+"// edit\n", "\n", "\r\n"))
	if edited := mustCheck(t, e.opts()); len(edited.Drift) != 1 {
		t.Errorf("an edit hidden by CRLF: %+v", edited.Drift)
	}
}

// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func TestNormalizeEOLChangesOnlyCRLF(t *testing.T) {
	if got := string(normalizeEOL([]byte("a\r\nb\rc\n\r\n"))); got != "a\nb\rc\n\n" {
		t.Errorf("normalizeEOL = %q: only CRLF changes", got)
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
	checkMissingTrace(t, missing)
	for name, body := range map[string]string{
		"garbage":            "not json",
		"no states":          `{"states":[]}`,
		"steps without ids":  `{"states":[{"a":1},{"a":2}]}`,
		"transition not str": `{"states":[{"a":1},{"lastTransition":5}]}`,
	} {
		path := filepath.Join(dir, strings.ReplaceAll(name, " ", "-")+".json")
		writeFile(t, path, body)
		if step := stepFromOutput(violation, path); !step.TraceUnavailable {
			t.Errorf("%s: %+v", name, step)
		}
	}
	checkInitialOnlyTrace(t, violation, dir)
	checkUnavailableFlagReachesOutputs(t, missing)
}

func checkMissingTrace(t *testing.T, missing StepResult) {
	t.Helper()
	if missing.Status != StatusViolated || !missing.TraceUnavailable || !strings.Contains(missing.Message, "readable trace") {
		t.Fatalf("missing trace %+v", missing)
	}
	if got := FormatCounterexample(missing); !strings.Contains(got, "trace unavailable") || strings.Contains(got, "initial state") {
		t.Errorf("format %q", got)
	}
}

func checkInitialOnlyTrace(t *testing.T, violation Output, dir string) {
	t.Helper()
	path := filepath.Join(dir, "initial.json")
	writeFile(t, path, `{"states":[{"lastTransition":"init"}]}`)
	initial := stepFromOutput(violation, path)
	if initial.TraceUnavailable || len(initial.Counterexample) != 0 || initial.Message != "" {
		t.Fatalf("initial-only trace %+v", initial)
	}
	if got := FormatCounterexample(initial); !strings.Contains(got, "initial state violates") {
		t.Errorf("format %q", got)
	}
}

// checkUnavailableFlagReachesOutputs checks that the flag reaches results.json
// and the evidence note.
func checkUnavailableFlagReachesOutputs(t *testing.T, missing StepResult) {
	t.Helper()
	data, _ := json.Marshal(missing)
	requireAll(t, string(data), `"traceUnavailable":true`)
	notes := evidenceNotes(ModuleResult{Machine: "SM-X"}, InvariantResult{ID: "INV-X", Status: StatusViolated, Run: &missing})
	requireAll(t, notes, "trace unavailable")
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
	writeFile(t, victim, "precious")
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
	checkDirectoryTargetRefused(t, dir)
	checkAtomicReplace(t, filepath.Join(dir, "out.json"))
	requireNoTempFiles(t, dir)
}

func requireNoTempFiles(t *testing.T, dir string) {
	t.Helper()
	entries, _ := os.ReadDir(dir)
	for _, entry := range entries {
		if strings.Contains(entry.Name(), ".tmp-") {
			t.Errorf("temp file left behind: %s", entry.Name())
		}
	}
}

func checkDirectoryTargetRefused(t *testing.T, dir string) {
	t.Helper()
	if err := os.Mkdir(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := WriteFileIfChanged(filepath.Join(dir, "sub"), []byte("x")); err == nil || !strings.Contains(err.Error(), "directory") {
		t.Fatalf("directory target: %v", err)
	}
}

// checkAtomicReplace checks create, rewrite-free no-op and update of a target.
func checkAtomicReplace(t *testing.T, target string) {
	t.Helper()
	if wrote, err := WriteFileIfChanged(target, []byte("one")); err != nil || !wrote {
		t.Fatalf("create: %v %v", wrote, err)
	}
	if info, _ := os.Stat(target); info.Mode().Perm() != 0o644 {
		t.Errorf("mode %v", info.Mode())
	}
	checkIdenticalWriteIsANoOp(t, target)
	if wrote, err := WriteFileIfChanged(target, []byte("two")); err != nil || !wrote || readFile(t, target) != "two" {
		t.Fatalf("update: %v %v", wrote, err)
	}
}

func checkIdenticalWriteIsANoOp(t *testing.T, target string) {
	t.Helper()
	before, _ := os.Stat(target)
	if wrote, err := WriteFileIfChanged(target, []byte("one")); err != nil || wrote {
		t.Fatalf("an identical file is not rewritten: %v %v", wrote, err)
	}
	after, _ := os.Stat(target)
	if !os.SameFile(before, after) {
		t.Error("identical content replaced the file")
	}
}

// TRLC-LINKS: REQ-EMG-145, REQ-EMG-149
func TestCheckRefusesToWriteResultsThroughASymlinkAndToReadALinkedModule(t *testing.T) {
	e := newEnv(t)
	victim := filepath.Join(e.dir, "victim.txt")
	writeFile(t, victim, "precious")
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
	checkLinkedModuleRefused(t, e)
}

func checkLinkedModuleRefused(t *testing.T, e env) {
	t.Helper()
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
