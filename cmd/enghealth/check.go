// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
)

// checkArgs are the parsed arguments of enghealth check.
type checkArgs struct {
	options engmodel.HealthCheckOptions
	format  string
	out     string
}

// parseCheckArgs parses the flags of enghealth check. The exit code is -1
// when the arguments are fine and the check should run.
//
// TRLC-LINKS: REQ-EMG-158
func parseCheckArgs(args []string, stderr io.Writer) (checkArgs, int) {
	flags := flag.NewFlagSet("enghealth check", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var a checkArgs
	flags.StringVar(&a.options.ManifestPath, "model", "", "path to the engmod.yml manifest")
	flags.StringVar(&a.options.Base, "base", "", "revision to compare with, or base...head (default HEAD: the uncommitted work)")
	flags.StringVar(&a.options.Head, "head", "", "revision to check (default: the working tree)")
	flags.StringVar(&a.options.Tier, "tier", "", "check at this life-cycle tier instead of the one the manifest declares")
	flags.StringVar(&a.format, "format", "text", "output format: text, json or sarif")
	flags.StringVar(&a.out, "out", "", "write the output to this file instead of stdout")
	flags.BoolVar(&a.options.NoCache, "no-cache", false, "do not read or write the result cache under .engmod/cache/health")
	flags.BoolVar(&a.options.UseCache, "cache", false, "use the result cache although CI is set (it is off in CI by default)")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: enghealth check --model <engmod.yml> [--base REV | base...head] [--head REV] [--tier t] [--format text|json|sarif] [--out file] [--cache] [--no-cache] [paths...]")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return a, exitOK
		}
		return a, exitUsage
	}
	if a.format != "text" && a.format != "json" && a.format != "sarif" {
		fmt.Fprintf(stderr, "enghealth check: unknown format %q; use text, json or sarif\n", a.format)
		return a, exitUsage
	}
	if strings.TrimSpace(a.options.ManifestPath) == "" {
		flags.Usage()
		return a, exitUsage
	}
	if a.options.UseCache && a.options.NoCache {
		fmt.Fprintln(stderr, "enghealth check: --cache and --no-cache cannot be combined")
		return a, exitUsage
	}
	a.options.Paths = flags.Args()
	a.options.Notes = stderr
	return a, -1
}

// renderCheck renders a report in a format.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func renderCheck(report engmodel.HealthCheckReport, format string) ([]byte, error) {
	switch format {
	case "json":
		return engmodel.EncodeHealthCheck(report)
	case "sarif":
		return engmodel.HealthCheckSARIF(report)
	}
	return []byte(renderCheckText(report)), nil
}

// runCheck compares the changed files with a base revision and fails on a
// regression that is an error at the tier. Exit 0: no regression, or only
// regressions that are warnings at the tier. Exit 1: at least one regression
// that is an error, a changed file that was skipped (too large, not a regular
// file) where a regression is an error, or a weakening of the health policy
// against the base revision that no policyDecision records where
// health.policy_weakened is an error. Exit 2: a setup error (the model does not load, a bad flag,
// tier, path or revision, an unavailable base, no git work tree, an unwritable
// output), which is never a pass.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func runCheck(args []string, stdout, stderr io.Writer) int {
	a, code := parseCheckArgs(args, stderr)
	if code >= 0 {
		return code
	}
	report, err := engmodel.CheckHealth(a.options)
	if err != nil {
		fmt.Fprintln(stderr, "enghealth check:", safeText(err.Error()))
		return exitUsage
	}
	data, err := renderCheck(report, a.format)
	if err != nil {
		fmt.Fprintln(stderr, "enghealth check: render:", safeText(err.Error()))
		return exitUsage
	}
	if err := writeOutput(data, a.out, stdout); err != nil {
		fmt.Fprintln(stderr, "enghealth check: write output:", safeText(err.Error()))
		return exitUsage
	}
	return checkVerdict(report, a.format == "text" && strings.TrimSpace(a.out) == "", stderr)
}

// checkVerdict returns the exit code of a report. When it fails and the text
// summary is not already on stdout, the regressions are printed to stderr, so
// that a log shows what failed whatever the format of the output file.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func checkVerdict(report engmodel.HealthCheckReport, textOnStdout bool, stderr io.Writer) int {
	if !report.Failed {
		return exitOK
	}
	if !textOnStdout {
		fmt.Fprint(stderr, regressionLines(report), failureLines(report))
	}
	fmt.Fprintf(stderr, "enghealth check: %s\n", failureSummary(report))
	return exitFail
}

// failureSummary is the last line of a failed check: the regressions that are
// errors at the tier, and what else failed it.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
func failureSummary(r engmodel.HealthCheckReport) string {
	parts := []string{}
	if len(r.Blocking) > 0 {
		parts = append(parts, fmt.Sprintf("%d regression(s) are errors at tier %s", len(r.Blocking), tierName(r.Tier)))
	}
	if n := skippedErrors(r); n > 0 {
		parts = append(parts, fmt.Sprintf("%d changed file(s) were skipped and were not analysed", n))
	}
	if n := policyErrors(r); n > 0 {
		parts = append(parts, fmt.Sprintf("%d weakening(s) of the health policy against the base revision are not recorded by a policyDecision", n))
	}
	return strings.Join(parts, "; ")
}

// skippedErrors counts the skipped changed files that fail the report: all of
// them where a regression is an error, else none.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func skippedErrors(r engmodel.HealthCheckReport) int {
	if r.RegressionLevel != "error" {
		return 0
	}
	return len(r.Skipped)
}

// policyErrors counts the weakenings of the health policy that fail the report.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func policyErrors(r engmodel.HealthCheckReport) int {
	if r.PolicyLevel != "error" || r.PolicyDecision != "" {
		return 0
	}
	return len(r.Policy)
}

// failureLines lists the skipped files and the policy weakenings, one per line.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
func failureLines(r engmodel.HealthCheckReport) string {
	var b strings.Builder
	for _, s := range r.Skipped {
		fmt.Fprintf(&b, "  skipped %s: %s\n", safeText(s.Path), safeText(s.Reason))
	}
	for _, c := range r.Policy {
		fmt.Fprintf(&b, "  policy weakened (%s): %s\n", safeText(c.Aspect), safeText(c.Message))
	}
	return b.String()
}

// writeOutput writes data to the file out through writeFileAtomic, or to
// stdout when out is empty.
//
// TRLC-LINKS: REQ-EMG-158
func writeOutput(data []byte, out string, stdout io.Writer) error {
	if strings.TrimSpace(out) != "" {
		return writeFileAtomic(out, data)
	}
	_, err := stdout.Write(data)
	return err
}

// shortRev abbreviates a commit id for the text summary.
//
// TRLC-LINKS: REQ-EMG-158
func shortRev(rev string) string {
	if len(rev) > 12 {
		return rev[:12]
	}
	return rev
}

// scoreText renders a file score and band, or "-" for a side that is absent.
//
// TRLC-LINKS: REQ-EMG-158
func scoreText(score *float64, band string) string {
	if score == nil {
		return "-"
	}
	return fmt.Sprintf("%.1f %s", *score, band)
}

// regressionLines lists the regressions, one per entry: file:line, function,
// code, before -> after and the message.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func regressionLines(r engmodel.HealthCheckReport) string {
	var b strings.Builder
	for _, e := range r.Blocking {
		fmt.Fprintf(&b, "  %s:%d %s: %s (%s) %s\n", safeText(e.Path), e.StartLine, safeText(e.Function), e.Code, e.Change, e.Transition())
		fmt.Fprintf(&b, "      %s\n", safeText(e.Message))
	}
	return b.String()
}

// changedFilesText lists the changed files with their score before and after.
//
// TRLC-LINKS: REQ-EMG-158
func changedFilesText(r engmodel.HealthCheckReport) string {
	if len(r.Files) == 0 {
		return "no changed source files\n"
	}
	var b strings.Builder
	fmt.Fprintf(&b, "\nchanged files (%d):\n", len(r.Files))
	for _, f := range r.Files {
		fmt.Fprintf(&b, "  %-40s %s -> %s\n", safeText(f.Path), scoreText(f.ScoreBefore, f.BandBefore), scoreText(f.ScoreAfter, f.BandAfter))
	}
	return b.String()
}

// findingKinds names what the report holds that did not fail it.
//
// TRLC-LINKS: REQ-EMG-158
func findingKinds(r engmodel.HealthCheckReport) []string {
	var kinds []string
	if len(r.Blocking) > 0 {
		kinds = append(kinds, "regressions")
	}
	if len(r.Skipped) > 0 {
		kinds = append(kinds, "skipped files")
	}
	if len(r.Policy) > 0 {
		kinds = append(kinds, "policy weakenings")
	}
	return kinds
}

// verdictText is the last line of the text summary.
//
// TRLC-LINKS: REQ-EMG-158
func verdictText(r engmodel.HealthCheckReport) string {
	switch {
	case r.Failed:
		return "result: FAIL, a regression, a skipped file or an unrecorded policy weakening is an error at this tier\n"
	case len(r.Blocking) > 0 || len(r.Skipped) > 0 || len(r.Policy) > 0:
		return "result: ok, the " + strings.Join(findingKinds(r), " and the ") + " only warn at this tier or are recorded by a decision\n"
	}
	return "result: ok, no regressions\n"
}

// renderCheckText is the short summary of a check: the revisions, the changed
// files with their score before and after, the regressions with the level they
// have at the tier, the counts and the verdict.
//
// TRLC-LINKS: REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func renderCheckText(r engmodel.HealthCheckReport) string {
	var b strings.Builder
	base := shortRev(r.Base)
	if base == "" {
		base = "(no commits)"
	}
	fmt.Fprintf(&b, "enghealth check: %s -> %s, tier %s\n", base, shortRev(r.Head), tierName(r.Tier))
	for _, n := range r.Notes {
		fmt.Fprintf(&b, "note: %s\n", safeText(n))
	}
	b.WriteString(changedFilesText(r))
	if len(r.Blocking) > 0 {
		fmt.Fprintf(&b, "\nregressions (%d, %s at tier %s):\n", len(r.Blocking), r.RegressionLevel, tierName(r.Tier))
		b.WriteString(regressionLines(r))
	}
	if len(r.Skipped) > 0 || len(r.Policy) > 0 {
		fmt.Fprintf(&b, "\nskipped files (%s at tier %s) and policy weakenings (%s):\n", r.RegressionLevel, tierName(r.Tier), r.PolicyLevel)
		b.WriteString(failureLines(r))
		if r.PolicyDecision != "" && len(r.Policy) > 0 {
			fmt.Fprintf(&b, "  the weakening is recorded by decision %s\n", safeText(r.PolicyDecision))
		}
	}
	c := r.Counts
	fmt.Fprintf(&b, "\nfindings in the changed files: %d new, %d worsened, %d existing, %d waived\n", c.New, c.Worsened, c.Existing, len(r.Waived))
	b.WriteString(verdictText(r))
	return b.String()
}
