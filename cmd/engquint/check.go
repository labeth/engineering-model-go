// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"text/tabwriter"
	"time"

	"github.com/labeth/engineering-model-go/internal/export/quint/check"
)

// runCheck generates the modules in memory, compares them with the committed
// ones and runs or verifies each invariant with Quint. Exit 0: everything
// passed (warnings allowed); 1: a violation, a stale result, a vacuous run or
// a type error; 2: a setup error.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-144, REQ-EMG-145, REQ-EMG-146
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func runCheck(args []string, stdout, stderr io.Writer) int {
	a, code, done := parseCheckFlags(args, stderr)
	if done {
		return code
	}

	// An interrupt or a termination signal cancels the check, which kills the
	// tool's whole process group instead of leaving it behind.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()
	report, err := check.Check(a.options(ctx))
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		if check.IsSetupError(err) {
			return exitUsage
		}
		return exitFail
	}
	printReport(stdout, stderr, report)
	if report.Failed() {
		return exitFail
	}
	return exitOK
}

// checkArgs holds the parsed engquint check flags.
type checkArgs struct {
	modelPath   string
	quintPath   string
	quintHome   string
	mode        string
	maxSteps    int
	maxSamples  int
	seed        string
	verifySteps int
	timeout     time.Duration
	passEnv     string
	outDir      string
	results     string
	evidence    string
	noEvidence  bool
}

// register declares the check flags on flags.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-144, REQ-EMG-145, REQ-EMG-146
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func (a *checkArgs) register(flags *flag.FlagSet) {
	flags.StringVar(&a.modelPath, "model", "", "path to engmod.yml manifest")
	flags.StringVar(&a.quintPath, "quint", "", "quint executable; defaults to $ENGMOD_QUINT, else quint on PATH")
	flags.StringVar(&a.quintHome, "quint-home", "", "directory with the pre-provisioned Apalache; defaults to $QUINT_HOME (verify mode)")
	flags.StringVar(&a.mode, "mode", string(check.ModeRun), "run, verify or both")
	flags.IntVar(&a.maxSteps, "max-steps", check.DefaultMaxSteps, fmt.Sprintf("maximum trace length of quint run (at most %d)", check.MaxStepsLimit))
	flags.IntVar(&a.maxSamples, "max-samples", check.DefaultMaxSamples, fmt.Sprintf("number of traces quint run simulates (at most %d)", check.MaxSamplesLimit))
	flags.StringVar(&a.seed, "seed", check.DefaultSeed, "random seed of quint run")
	flags.IntVar(&a.verifySteps, "verify-steps", check.DefaultVerifySteps, fmt.Sprintf("maximum depth of quint verify (at most %d)", check.MaxVerifyStepsLimit))
	flags.DurationVar(&a.timeout, "timeout", check.DefaultTimeout, "overall time limit of the check; the running tool is killed when it expires (one call is limited to "+check.DefaultCallTimeout.String()+")")
	flags.StringVar(&a.passEnv, "pass-env", "", "comma-separated names of extra environment variables to pass to the tools; by default only PATH, HOME, LANG, LC_ALL, TMPDIR, TEMP, TMP, TERM, JAVA_HOME, QUINT_HOME (and proxy variables in verify mode) are")
	flags.StringVar(&a.outDir, "out", "", "directory with the committed modules; defaults to generated/quint next to the model's engmod.yml")
	flags.StringVar(&a.results, "results", "", "results file; defaults to <out>/results.json")
	flags.StringVar(&a.evidence, "evidence", "", "formal evidence file; defaults to test-results/quint-formal.json next to the model")
	flags.BoolVar(&a.noEvidence, "no-evidence", false, "do not write the evidence file")
}

// parseCheckFlags parses and validates the check flags. done is true when the
// command ends with code.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-144, REQ-EMG-145, REQ-EMG-146
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func parseCheckFlags(args []string, stderr io.Writer) (a *checkArgs, code int, done bool) {
	a = &checkArgs{}
	flags := flag.NewFlagSet("engquint check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	a.register(flags)
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return a, exitOK, true
		}
		return a, exitUsage, true
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "engquint check: unexpected argument %q\n", flags.Arg(0))
		return a, exitUsage, true
	}
	if strings.TrimSpace(a.modelPath) == "" {
		fmt.Fprintln(stderr, "usage: engquint check --model <engmod.yml> [--mode run|verify|both] [--quint path] [--quint-home dir] [--max-steps N] [--max-samples M] [--seed S] [--verify-steps N] [--timeout 30m] [--pass-env NAME,...] [--out dir] [--results path] [--evidence path] [--no-evidence]")
		return a, exitUsage, true
	}
	return a, exitOK, false
}

// options maps the flags to the check options.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-144, REQ-EMG-145, REQ-EMG-146
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func (a *checkArgs) options(ctx context.Context) check.CheckOptions {
	return check.CheckOptions{
		ModelPath: a.modelPath, QuintPath: a.quintPath, QuintHome: a.quintHome, Mode: check.Mode(a.mode),
		MaxSteps: a.maxSteps, MaxSamples: a.maxSamples, Seed: a.seed, VerifySteps: a.verifySteps,
		OutDir: a.outDir, ResultsPath: a.results, EvidencePath: a.evidence, NoEvidence: a.noEvidence,
		Context: ctx, Timeout: a.timeout, PassEnv: splitNames(a.passEnv),
	}
}

// printReport prints the drift, one table per module and the counterexamples.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-145
func printReport(stdout, stderr io.Writer, report check.CheckReport) {
	for _, msg := range report.GenerationErrors {
		fmt.Fprintln(stderr, "generation error:", msg)
	}
	switch {
	case len(report.GenerationErrors) > 0:
		fmt.Fprintln(stdout, "result: FAIL (the model does not generate; fix the diagnostics above)")
		printCleared(stdout, report)
	case report.NoMachines:
		fmt.Fprintln(stdout, "model has no machines; nothing to check")
		printCleared(stdout, report)
	default:
		printChecked(stdout, report)
	}
}

// printChecked prints the drift, the modules, the verdict and the output paths
// of a report that was checked.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-145
func printChecked(stdout io.Writer, report check.CheckReport) {
	for _, d := range report.Drift {
		fmt.Fprintf(stdout, "stale: %s is %s; run engquint generate and commit the module\n", d.File, d.Kind)
	}
	for _, m := range report.Modules {
		printModule(stdout, m)
	}
	if report.Failed() {
		fmt.Fprintln(stdout, "result: FAIL")
	} else {
		fmt.Fprintln(stdout, "result: pass")
	}
	if report.ResultsPath != "" {
		fmt.Fprintf(stdout, "results:  %s\n", report.ResultsPath)
	}
	if report.EvidencePath != "" {
		fmt.Fprintf(stdout, "evidence: %s (%d entries)\n", report.EvidencePath, report.EvidenceEntries)
	}
}

// printCleared says that the evidence file lost its entries because the model
// no longer generates or has no machines.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func printCleared(stdout io.Writer, report check.CheckReport) {
	if report.EvidencePath != "" {
		fmt.Fprintf(stdout, "evidence: %s (cleared: its entries no longer apply)\n", report.EvidencePath)
	}
}

// splitNames splits a comma-separated list of names and drops empty ones.
//
// TRLC-LINKS: REQ-EMG-142
func splitNames(list string) []string {
	var names []string
	for _, name := range strings.Split(list, ",") {
		if name = strings.TrimSpace(name); name != "" {
			names = append(names, name)
		}
	}
	return names
}

// printModule prints the invariant table, the warnings and the counterexamples
// of one module.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143
func printModule(w io.Writer, m check.ModuleResult) {
	fmt.Fprintf(w, "%s (%s, %s) %s\n", m.Machine, m.QntFile, strings.Join(m.Modes, "+"), strings.ToUpper(m.Status))
	printMessages(w, "type error", m.TypeErrors)
	printMessages(w, "error", m.Errors)
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, inv := range m.Invariants {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", inv.ID, strings.Join(inv.Requirements, ", "), stepColumns(inv))
	}
	tw.Flush()
	for _, inv := range m.Invariants {
		printStepProblems(w, inv)
	}
	printMessages(w, "warning", m.Warnings)
}

// printMessages prints each message of a module with its label.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143
func printMessages(w io.Writer, label string, msgs []string) {
	for _, msg := range msgs {
		fmt.Fprintf(w, "  %s: %s\n", label, msg)
	}
}

// namedStep is one verdict of an invariant with the mode that produced it.
type namedStep struct {
	mode string
	step *check.StepResult
}

// printStepProblems prints the violations and errors of the run and verify
// steps of an invariant.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143
func printStepProblems(w io.Writer, inv check.InvariantResult) {
	for _, named := range []namedStep{{"run", inv.Run}, {"verify", inv.Verify}} {
		if named.step != nil {
			printStepProblem(w, inv, named)
		}
	}
}

// printStepProblem prints the counterexample or error of one step.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143
func printStepProblem(w io.Writer, inv check.InvariantResult, named namedStep) {
	switch named.step.Status {
	case check.StatusViolated:
		fmt.Fprintf(w, "  %s violated: %s (%s; %s)\n", inv.ID, check.FormatCounterexample(*named.step), named.mode, strings.Join(inv.Requirements, ", "))
		if named.step.TraceUnavailable {
			fmt.Fprintf(w, "    %s\n", named.step.Message)
		}
	case check.StatusError:
		fmt.Fprintf(w, "  %s error (%s): %s\n", inv.ID, named.mode, named.step.Message)
	}
}

// stepColumns renders the run and verify verdicts of an invariant.
//
// TRLC-LINKS: REQ-EMG-142
func stepColumns(inv check.InvariantResult) string {
	if inv.Run == nil && inv.Verify == nil {
		return inv.Status
	}
	var cols []string
	if inv.Run != nil {
		cols = append(cols, "run: "+inv.Run.Status)
	}
	if inv.Verify != nil {
		cols = append(cols, "verify: "+inv.Verify.Status)
	}
	return strings.Join(cols, "  ")
}
