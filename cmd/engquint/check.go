// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"
	"text/tabwriter"

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
	flags := flag.NewFlagSet("engquint check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modelPath := flags.String("model", "", "path to engmod.yml manifest")
	quintPath := flags.String("quint", "", "quint executable; defaults to $ENGMOD_QUINT, else quint on PATH")
	quintHome := flags.String("quint-home", "", "directory with the pre-provisioned Apalache; defaults to $QUINT_HOME (verify mode)")
	mode := flags.String("mode", string(check.ModeRun), "run, verify or both")
	maxSteps := flags.Int("max-steps", check.DefaultMaxSteps, "maximum trace length of quint run")
	maxSamples := flags.Int("max-samples", check.DefaultMaxSamples, "number of traces quint run simulates")
	seed := flags.String("seed", check.DefaultSeed, "random seed of quint run")
	verifySteps := flags.Int("verify-steps", check.DefaultVerifySteps, "maximum depth of quint verify")
	outDir := flags.String("out", "", "directory with the committed modules; defaults to generated/quint next to the model's engmod.yml")
	results := flags.String("results", "", "results file; defaults to <out>/results.json")
	evidence := flags.String("evidence", "", "formal evidence file; defaults to test-results/quint-formal.json next to the model")
	noEvidence := flags.Bool("no-evidence", false, "do not write the evidence file")
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return exitOK
		}
		return exitUsage
	}
	if flags.NArg() > 0 {
		fmt.Fprintf(stderr, "engquint check: unexpected argument %q\n", flags.Arg(0))
		return exitUsage
	}
	if strings.TrimSpace(*modelPath) == "" {
		fmt.Fprintln(stderr, "usage: engquint check --model <engmod.yml> [--mode run|verify|both] [--quint path] [--quint-home dir] [--max-steps N] [--max-samples M] [--seed S] [--verify-steps N] [--out dir] [--results path] [--evidence path] [--no-evidence]")
		return exitUsage
	}

	report, err := check.Check(check.CheckOptions{
		ModelPath: *modelPath, QuintPath: *quintPath, QuintHome: *quintHome, Mode: check.Mode(*mode),
		MaxSteps: *maxSteps, MaxSamples: *maxSamples, Seed: *seed, VerifySteps: *verifySteps,
		OutDir: *outDir, ResultsPath: *results, EvidencePath: *evidence, NoEvidence: *noEvidence,
	})
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

// printReport prints the drift, one table per module and the counterexamples.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143, REQ-EMG-145
func printReport(stdout, stderr io.Writer, report check.CheckReport) {
	for _, msg := range report.GenerationErrors {
		fmt.Fprintln(stderr, "generation error:", msg)
	}
	if len(report.GenerationErrors) > 0 {
		fmt.Fprintln(stdout, "result: FAIL (the model does not generate; fix the diagnostics above)")
		return
	}
	if report.NoMachines {
		fmt.Fprintln(stdout, "model has no machines; nothing to check")
		return
	}
	for _, d := range report.Drift {
		fmt.Fprintf(stdout, "stale: %s is %s; run engquint generate and commit the module\n", d.File, d.Kind)
	}
	for _, m := range report.Modules {
		printModule(stdout, m)
	}
	switch {
	case report.Failed():
		fmt.Fprintln(stdout, "result: FAIL")
	default:
		fmt.Fprintln(stdout, "result: pass")
	}
	if report.ResultsPath != "" {
		fmt.Fprintf(stdout, "results:  %s\n", report.ResultsPath)
	}
	if report.EvidencePath != "" {
		fmt.Fprintf(stdout, "evidence: %s (%d entries)\n", report.EvidencePath, report.EvidenceEntries)
	}
}

// printModule prints the invariant table, the warnings and the counterexamples
// of one module.
//
// TRLC-LINKS: REQ-EMG-142, REQ-EMG-143
func printModule(w io.Writer, m check.ModuleResult) {
	fmt.Fprintf(w, "%s (%s, %s) %s\n", m.Machine, m.QntFile, strings.Join(m.Modes, "+"), strings.ToUpper(m.Status))
	for _, msg := range m.TypeErrors {
		fmt.Fprintf(w, "  type error: %s\n", msg)
	}
	for _, msg := range m.Errors {
		fmt.Fprintf(w, "  error: %s\n", msg)
	}
	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)
	for _, inv := range m.Invariants {
		fmt.Fprintf(tw, "  %s\t%s\t%s\n", inv.ID, strings.Join(inv.Requirements, ", "), stepColumns(inv))
	}
	tw.Flush()
	for _, inv := range m.Invariants {
		for _, named := range []struct {
			mode string
			step *check.StepResult
		}{{"run", inv.Run}, {"verify", inv.Verify}} {
			if named.step == nil {
				continue
			}
			switch named.step.Status {
			case check.StatusViolated:
				fmt.Fprintf(w, "  %s violated: %s (%s; %s)\n", inv.ID, check.FormatCounterexample(named.step.Counterexample), named.mode, strings.Join(inv.Requirements, ", "))
			case check.StatusError:
				fmt.Fprintf(w, "  %s error (%s): %s\n", inv.ID, named.mode, named.step.Message)
			}
		}
	}
	for _, msg := range m.Warnings {
		fmt.Fprintf(w, "  warning: %s\n", msg)
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
