// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER

// Package check runs the generated Quint modules of a model with Quint and
// records the verdicts. It compares the committed modules with a fresh
// generation (drift is stale), typechecks each module, runs Quint once per
// invariant on the TypeScript backend, optionally verifies each invariant with
// the pre-provisioned Apalache, and writes a deterministic results file and the
// formal evidence file the trace reads. It never downloads a tool.
package check

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/internal/export/quint"
	"github.com/labeth/engineering-model-go/internal/model"
)

// Mode selects which Quint commands check runs.
type Mode string

const (
	// ModeRun simulates with quint run, once per invariant.
	ModeRun Mode = "run"
	// ModeVerify model-checks with quint verify (Apalache), once per invariant.
	ModeVerify Mode = "verify"
	// ModeBoth does both.
	ModeBoth Mode = "both"
)

// Verdicts of an invariant, a module and a Quint call.
const (
	StatusPass     = "pass"
	StatusViolated = "violated"
	StatusError    = "error"
	StatusStale    = "stale"
)

// Defaults of CheckOptions.
const (
	DefaultMaxSteps    = 20
	DefaultMaxSamples  = 10000
	DefaultSeed        = "1"
	DefaultVerifySteps = 10
)

// Largest values of the search bounds. A larger value is a setup error: the
// cost of a run grows with steps times samples, and a typo must not turn a
// check into a multi-hour job (the time limits would kill it anyway).
const (
	MaxStepsLimit       = 1000
	MaxSamplesLimit     = 1000000
	MaxVerifyStepsLimit = 100
)

// Backend is the only Quint run backend: the default Rust backend downloads
// its binary at run time.
const Backend = "typescript"

// CheckOptions configures Check. Zero values select the documented defaults.
type CheckOptions struct {
	// ModelPath is the engmod.yml manifest.
	ModelPath string
	// QuintPath is the quint executable; default ENGMOD_QUINT, else quint on PATH.
	QuintPath string
	// QuintHome is the directory with the pre-provisioned Apalache distribution;
	// default QUINT_HOME. Only verify mode needs it.
	QuintHome string
	// Mode is run (default), verify or both.
	Mode Mode
	// MaxSteps bounds the length of a simulated trace (default 20).
	MaxSteps int
	// MaxSamples is the number of simulated traces (default 10000).
	MaxSamples int
	// Seed is the fixed random seed (default "1").
	Seed string
	// VerifySteps bounds the depth of quint verify (default 10).
	VerifySteps int
	// OutDir holds the committed modules (default <model dir>/generated/quint).
	OutDir string
	// ResultsPath is the results file (default <OutDir>/results.json).
	ResultsPath string
	// EvidencePath is the formal evidence file the trace reads (default
	// <model dir>/test-results/quint-formal.json).
	EvidencePath string
	// NoEvidence suppresses the evidence file.
	NoEvidence bool
	// Runner starts external processes; default ExecRunner. Tests inject a fake.
	Runner Runner
	// Context bounds the whole check; default context.Background(). Cancelling
	// it kills the running tool and ends the check with an error.
	Context context.Context
	// Timeout is the overall deadline of the check, all calls together (default
	// DefaultTimeout, 30 minutes). When it expires the running tool is killed
	// and Check returns an error that wraps ErrTimeout.
	Timeout time.Duration
	// CallTimeout is the limit of one Quint, Apalache or Java call (default
	// DefaultCallTimeout, 10 minutes).
	CallTimeout time.Duration
	// MaxOutput is the number of bytes captured from each output stream of one
	// call (default DefaultMaxOutput, 8 MiB).
	MaxOutput int
	// PassEnv names environment variables passed to the tools in addition to
	// the built-in allow-list (PATH, HOME, LANG, LC_ALL, TMPDIR, TEMP, TMP,
	// TERM, JAVA_HOME, QUINT_HOME; proxy variables in verify mode). Nothing
	// else of the caller's environment is inherited.
	PassEnv []string
}

// Bounds are the search bounds recorded with a result.
type Bounds struct {
	MaxSteps    int    `json:"maxSteps"`
	MaxSamples  int    `json:"maxSamples"`
	Seed        string `json:"seed"`
	VerifySteps int    `json:"verifySteps"`
}

// StepResult is the verdict of one Quint call on one invariant.
type StepResult struct {
	// Status is pass, violated or error.
	Status   string `json:"status"`
	ExitCode int    `json:"exitCode"`
	// TraceMin and TraceMax are the trace length statistics of a run.
	TraceMin *int `json:"traceMin,omitempty"`
	TraceMax *int `json:"traceMax,omitempty"`
	// Counterexample is the ordered TR-* IDs of a violating trace.
	Counterexample []string `json:"counterexample,omitempty"`
	// TraceUnavailable is set when the step violated the invariant but its
	// counterexample trace could not be read, so Counterexample is empty
	// without the initial state being the violation.
	TraceUnavailable bool `json:"traceUnavailable,omitempty"`
	// Message explains an error.
	Message string `json:"message,omitempty"`
}

// InvariantResult is the verdict on one invariant.
type InvariantResult struct {
	Name         string      `json:"name"`
	ID           string      `json:"id"`
	Requirements []string    `json:"requirements,omitempty"`
	Status       string      `json:"status"`
	Run          *StepResult `json:"run,omitempty"`
	Verify       *StepResult `json:"verify,omitempty"`
}

// WitnessResult is the reachability of one state.
type WitnessResult struct {
	Name      string `json:"name"`
	State     string `json:"state"`
	Witnessed int    `json:"witnessed"`
	Total     int    `json:"total"`
}

// ModuleResult is the outcome for one generated module.
type ModuleResult struct {
	Machine      string   `json:"machine"`
	Module       string   `json:"module"`
	QntFile      string   `json:"qntFile"`
	QntSha256    string   `json:"qntSha256,omitempty"`
	QuintVersion string   `json:"quintVersion"`
	Backend      string   `json:"backend"`
	Modes        []string `json:"modes"`
	Bounds       Bounds   `json:"bounds"`
	// Status is stale (drift), error, violated or pass.
	Status string `json:"status"`
	// Drift is changed or missing when the committed module differs from a
	// fresh generation.
	Drift      string            `json:"drift,omitempty"`
	TypeErrors []string          `json:"typeErrors,omitempty"`
	Invariants []InvariantResult `json:"invariants"`
	Witnesses  []WitnessResult   `json:"witnesses"`
	Warnings   []string          `json:"warnings"`
	Errors     []string          `json:"errors,omitempty"`
}

// Drift is one generated-module file that disagrees with a fresh generation.
type Drift struct {
	File string `json:"file"`
	// Kind is changed, missing or extra.
	Kind string `json:"kind"`
}

// CheckReport is the outcome of Check.
type CheckReport struct {
	QuintVersion     string
	NoMachines       bool
	GenerationErrors []string
	Drift            []Drift
	Modules          []ModuleResult
	// ResultsPath and EvidencePath are the files written; empty when not written.
	ResultsPath  string
	EvidencePath string
	// EvidenceEntries is the number of entries in the evidence file.
	EvidenceEntries int
}

// Failed reports whether the check found a violation, a stale result, a type
// error, a vacuous or erroring run, or a generation error: the CLI exits 1.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-145
func (r CheckReport) Failed() bool {
	if len(r.GenerationErrors) > 0 || len(r.Drift) > 0 {
		return true
	}
	for _, m := range r.Modules {
		if m.Status != StatusPass {
			return true
		}
	}
	return false
}

// checker carries the state of one Check call.
type checker struct {
	ctx       context.Context
	opts      CheckOptions
	run       Runner
	quint     string
	version   string
	scratch   string
	javaDir   string
	apalache  string
	modelDir  string
	verifying bool
	running   bool
	// env and verifyBaseEnv are the allow-listed environments of a call and of
	// a verify call before verifyEnv completes it.
	env           []string
	verifyBaseEnv []string
	// apalacheConfig is the Apalache configuration file verify passes.
	apalacheConfig string
}

// Check generates the modules of the model in memory, compares them with the
// committed ones in OutDir (any difference is drift and so stale), typechecks
// and runs or verifies every module, and writes the results file and the
// evidence file. A problem with the environment is returned as a *SetupError.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-144, REQ-EMG-145, REQ-EMG-146
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func Check(opts CheckOptions) (CheckReport, error) {
	var report CheckReport
	opts, err := withDefaults(opts)
	if err != nil {
		return report, err
	}
	bundle, err := model.LoadBundle(opts.ModelPath)
	if err != nil {
		return report, setupError(err, "load model %s", opts.ModelPath)
	}
	generated, genErr := quint.Generate(bundle)
	if genErr != nil {
		for _, d := range generated.Diagnostics {
			if d.Severity == model.SemanticSeverityError {
				report.GenerationErrors = append(report.GenerationErrors, fmt.Sprintf("%s: %s (%s)", d.Code, d.Message, d.Path))
			}
		}
		if len(report.GenerationErrors) == 0 {
			report.GenerationErrors = []string{genErr.Error()}
		}
		return report, dropEvidence(opts, &report)
	}

	drifts, moduleDrift, err := compareWithCommitted(opts.OutDir, generated.Files)
	if err != nil {
		return report, setupError(err, "read generated modules in %s", opts.OutDir)
	}
	report.Drift = drifts
	if len(generated.Files) == 0 {
		report.NoMachines = len(drifts) == 0
		return report, dropEvidence(opts, &report)
	}

	ctx, cancel := context.WithTimeout(opts.Context, opts.Timeout)
	defer cancel()
	c := &checker{ctx: ctx, opts: opts, run: opts.Runner, modelDir: filepath.Dir(opts.ModelPath)}
	c.running = opts.Mode == ModeRun || opts.Mode == ModeBoth
	c.verifying = opts.Mode == ModeVerify || opts.Mode == ModeBoth
	c.env = allowEnv(os.Environ(), opts.PassEnv, false)
	c.verifyBaseEnv = allowEnv(os.Environ(), opts.PassEnv, true)
	if err := c.prepare(); err != nil {
		return report, err
	}
	defer os.RemoveAll(c.scratch)
	report.QuintVersion = c.version

	for _, file := range generated.Files {
		module, err := c.checkModule(file, moduleDrift[file.Name])
		if err != nil {
			return report, err
		}
		report.Modules = append(report.Modules, module)
	}

	if err := writeResults(opts.ResultsPath, report.Modules); err != nil {
		return report, fmt.Errorf("write results: %w", err)
	}
	report.ResultsPath = opts.ResultsPath
	if !opts.NoEvidence {
		entries := evidenceEntries(report.Modules)
		if err := writeEvidence(opts.EvidencePath, c.version, entries); err != nil {
			return report, fmt.Errorf("write evidence: %w", err)
		}
		report.EvidencePath = opts.EvidencePath
		report.EvidenceEntries = len(entries)
	}
	return report, nil
}

// dropEvidence is the evidence rule of a check that ends before any module is
// run (the model does not generate, it has no machines, or only extra
// generated modules drifted): an existing evidence file loses its entries, so
// the trace stops showing a pass for a model that no longer generates. A file
// that does not exist is not created.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func dropEvidence(opts CheckOptions, report *CheckReport) error {
	if opts.NoEvidence {
		return nil
	}
	cleared, err := clearEvidence(opts.EvidencePath)
	if err != nil {
		return fmt.Errorf("write evidence: %w", err)
	}
	if cleared {
		report.EvidencePath = opts.EvidencePath
	}
	return nil
}

// withDefaults fills the zero values of opts.
//
// TRLC-LINKS: REQ-EMG-142
func withDefaults(opts CheckOptions) (CheckOptions, error) {
	if strings.TrimSpace(opts.ModelPath) == "" {
		return opts, setupError(nil, "no model path given")
	}
	abs, err := filepath.Abs(opts.ModelPath)
	if err != nil {
		return opts, setupError(err, "resolve model path")
	}
	opts.ModelPath = abs
	modelDir := filepath.Dir(abs)
	switch opts.Mode {
	case "":
		opts.Mode = ModeRun
	case ModeRun, ModeVerify, ModeBoth:
	default:
		return opts, setupError(nil, "unknown mode %q (run, verify or both)", opts.Mode)
	}
	if opts.MaxSteps <= 0 {
		opts.MaxSteps = DefaultMaxSteps
	}
	if opts.MaxSamples <= 0 {
		opts.MaxSamples = DefaultMaxSamples
	}
	if opts.VerifySteps <= 0 {
		opts.VerifySteps = DefaultVerifySteps
	}
	for _, bound := range []struct {
		name         string
		value, limit int
	}{{"max-steps", opts.MaxSteps, MaxStepsLimit}, {"max-samples", opts.MaxSamples, MaxSamplesLimit}, {"verify-steps", opts.VerifySteps, MaxVerifyStepsLimit}} {
		if bound.value > bound.limit {
			return opts, setupError(nil, "%s %d is above the maximum %d", bound.name, bound.value, bound.limit)
		}
	}
	if strings.TrimSpace(opts.Seed) == "" {
		opts.Seed = DefaultSeed
	}
	if opts.OutDir == "" {
		opts.OutDir = filepath.Join(modelDir, "generated", "quint")
	}
	if opts.OutDir, err = filepath.Abs(opts.OutDir); err != nil {
		return opts, setupError(err, "resolve output directory")
	}
	if opts.ResultsPath == "" {
		opts.ResultsPath = filepath.Join(opts.OutDir, "results.json")
	}
	if opts.EvidencePath == "" {
		opts.EvidencePath = filepath.Join(modelDir, "test-results", "quint-formal.json")
	}
	if opts.QuintHome == "" {
		opts.QuintHome = os.Getenv(QuintHomeEnv)
	}
	if opts.QuintHome != "" {
		if opts.QuintHome, err = filepath.Abs(opts.QuintHome); err != nil {
			return opts, setupError(err, "resolve Quint home")
		}
	}
	if opts.Runner == nil {
		opts.Runner = ExecRunner
	}
	if opts.Context == nil {
		opts.Context = context.Background()
	}
	if opts.Timeout < 0 || opts.CallTimeout < 0 || opts.MaxOutput < 0 {
		return opts, setupError(nil, "timeouts and the output limit must not be negative")
	}
	if opts.Timeout == 0 {
		opts.Timeout = DefaultTimeout
	}
	if opts.CallTimeout == 0 {
		opts.CallTimeout = DefaultCallTimeout
	}
	if opts.MaxOutput == 0 {
		opts.MaxOutput = DefaultMaxOutput
	}
	return opts, nil
}

// prepare resolves the tools, checks the verify prerequisites and creates the
// scratch directory. Every failure is a setup error.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-144
func (c *checker) prepare() error {
	path, err := resolveQuint(c.opts.QuintPath)
	if err != nil {
		return err
	}
	c.quint = path
	if c.verifying {
		if strings.TrimSpace(c.opts.QuintHome) == "" {
			return setupError(nil, "verify mode needs the pre-provisioned Apalache: set %s or pass --quint-home (%s)", QuintHomeEnv, installHint)
		}
		c.apalache = apalacheMC(c.opts.QuintHome)
		if info, statErr := os.Stat(c.apalache); statErr != nil || info.IsDir() {
			return setupError(statErr, "Apalache %s is not provisioned under %s (%s); engquint never downloads it", ApalacheVersion, c.opts.QuintHome, installHint)
		}
		if c.javaDir, err = resolveJava(c.ctx, c.run); err != nil {
			return err
		}
	}
	if c.scratch, err = os.MkdirTemp("", "engquint-"); err != nil {
		return setupError(err, "create scratch directory")
	}
	if c.verifying {
		c.apalacheConfig = filepath.Join(c.scratch, "apalache.json")
		if err := os.WriteFile(c.apalacheConfig, []byte(apalacheConfigJSON), 0o644); err != nil {
			os.RemoveAll(c.scratch)
			return setupError(err, "write the Apalache configuration")
		}
	}
	out, err := c.call(Command{Path: c.quint, Args: []string{"--version"}, Dir: c.scratch, Env: c.env})
	if err != nil {
		os.RemoveAll(c.scratch)
		return err
	}
	c.version = strings.TrimSpace(firstLine(out.Stdout))
	if out.ExitCode != 0 || c.version == "" {
		os.RemoveAll(c.scratch)
		return setupError(nil, "%s --version failed (exit %d)", c.quint, out.ExitCode)
	}
	return nil
}

// modes lists the modes the check runs, in report order.
//
// TRLC-LINKS: REQ-EMG-145
func (c *checker) modes() []string {
	var modes []string
	if c.running {
		modes = append(modes, string(ModeRun))
	}
	if c.verifying {
		modes = append(modes, string(ModeVerify))
	}
	return modes
}

// bounds returns the search bounds in use.
//
// TRLC-LINKS: REQ-EMG-145
func (c *checker) bounds() Bounds {
	return Bounds{MaxSteps: c.opts.MaxSteps, MaxSamples: c.opts.MaxSamples, Seed: c.opts.Seed, VerifySteps: c.opts.VerifySteps}
}

// checkModule checks one generated module. A stale module (drift) is reported
// but not run: its committed text is not what the model says. A Quint process
// that cannot be started is returned as a setup error.
//
// TRLC-LINKS: REQ-EMG-143, REQ-EMG-144, REQ-EMG-145
func (c *checker) checkModule(file quint.File, drift string) (ModuleResult, error) {
	path := filepath.Join(c.opts.OutDir, file.Name)
	rel, err := filepath.Rel(c.modelDir, path)
	if err != nil {
		rel = path
	}
	m := ModuleResult{
		Machine: file.Machine, Module: file.Module, QntFile: filepath.ToSlash(rel),
		QuintVersion: c.version, Backend: Backend, Modes: c.modes(), Bounds: c.bounds(),
		Drift: drift, Invariants: []InvariantResult{}, Witnesses: []WitnessResult{}, Warnings: []string{},
	}
	if data, readErr := os.ReadFile(path); readErr == nil {
		m.QntSha256 = sha256Hex(normalizeEOL(data))
	}
	for _, inv := range file.Invariants {
		m.Invariants = append(m.Invariants, InvariantResult{Name: inv.Name, ID: inv.ID, Requirements: inv.Requirements})
	}
	if drift != "" {
		m.Status = StatusStale
		for i := range m.Invariants {
			m.Invariants[i].Status = StatusStale
		}
		return m, nil
	}

	ok, err := c.typecheck(&m, path)
	if err != nil {
		return m, err
	}
	if !ok {
		m.Status = StatusError
		for i := range m.Invariants {
			m.Invariants[i].Status = StatusError
		}
		return m, nil
	}
	if c.running {
		if err := c.runModule(&m, file, path); err != nil {
			return m, err
		}
	}
	if c.verifying {
		if err := c.verifyModule(&m, path); err != nil {
			return m, err
		}
	}
	finishModule(&m)
	return m, nil
}

// finishModule derives the invariant and module statuses from the step
// results. An error beats a violation, which beats a pass.
//
// TRLC-LINKS: REQ-EMG-145
func finishModule(m *ModuleResult) {
	status := StatusPass
	for i := range m.Invariants {
		inv := &m.Invariants[i]
		inv.Status = worst(stepStatus(inv.Run), stepStatus(inv.Verify))
		status = worst(status, inv.Status)
	}
	if len(m.Errors) > 0 {
		status = StatusError
	}
	m.Status = status
}

// stepStatus is the status of an optional step; a step that did not run is
// a pass for the combination.
//
// TRLC-LINKS: REQ-EMG-145
func stepStatus(s *StepResult) string {
	if s == nil {
		return StatusPass
	}
	return s.Status
}

// worst returns the more severe of two statuses: error, violated, pass.
//
// TRLC-LINKS: REQ-EMG-145
func worst(a, b string) string {
	rank := map[string]int{StatusPass: 0, StatusViolated: 1, StatusError: 2}
	if rank[b] > rank[a] {
		return b
	}
	return a
}

// addWarning appends a warning once.
//
// TRLC-LINKS: REQ-EMG-143
func (m *ModuleResult) addWarning(text string) {
	for _, w := range m.Warnings {
		if w == text {
			return
		}
	}
	m.Warnings = append(m.Warnings, text)
}

// call runs one external command under the limits of the check. A process that
// cannot start is a setup error; an expired overall deadline or a cancelled
// context ends the check with an error that wraps ErrTimeout or the context
// error, because the tool was killed and its output means nothing.
//
// TRLC-LINKS: REQ-EMG-143
func (c *checker) call(cmd Command) (Output, error) {
	if err := c.ctxError(); err != nil {
		return Output{}, err
	}
	cmd.Timeout, cmd.MaxOutput = c.opts.CallTimeout, c.opts.MaxOutput
	out, err := c.run(c.ctx, cmd)
	if ctxErr := c.ctxError(); ctxErr != nil {
		return out, ctxErr
	}
	if err != nil {
		return out, setupError(err, "cannot start %s", cmd.Path)
	}
	return out, nil
}

// ctxError reports the expiry or cancellation of the check's context.
//
// TRLC-LINKS: REQ-EMG-143
func (c *checker) ctxError() error {
	err := c.ctx.Err()
	switch {
	case err == nil:
		return nil
	case errors.Is(err, context.DeadlineExceeded):
		return fmt.Errorf("%w: the check exceeded its overall limit of %s (raise it with --timeout)", ErrTimeout, c.opts.Timeout)
	default:
		return fmt.Errorf("check cancelled: %w", err)
	}
}

// typecheck runs quint typecheck on the module. Type errors are recorded and
// return false; only a process that cannot start is an error.
//
// TRLC-LINKS: REQ-EMG-142
func (c *checker) typecheck(m *ModuleResult, path string) (bool, error) {
	jsonPath := filepath.Join(c.scratch, m.Module+".typecheck.json")
	out, err := c.call(Command{Path: c.quint, Args: []string{"typecheck", path, "--out=" + jsonPath}, Dir: c.scratch, Env: c.env})
	if err != nil {
		return false, err
	}
	if out.ExitCode == 0 {
		return true, nil
	}
	m.TypeErrors = typeErrors(jsonPath, out)
	return false, nil
}
