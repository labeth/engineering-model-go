// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"sort"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
)

// Exit codes shared by every subcommand: 0 success, 1 blocking findings (scan
// with --fail-on-error, check on a regression that is an error at the tier),
// 2 a setup or usage error (never a pass). A source file that was skipped or
// failed has the level of a regression at the tier, so it counts as an error
// of scan --fail-on-error from product and makes check exit 1 there.
const (
	exitOK    = 0
	exitFail  = 1
	exitUsage = 2
)

// command is one enghealth subcommand. A new subcommand (such as check) is one
// more entry in commands; run receives the arguments after the command name.
type command struct {
	summary string
	run     func(args []string, stdout, stderr io.Writer) int
}

var commands = map[string]command{
	"scan":     {summary: "score the code of a model and write the health report", run: runScan},
	"check":    {summary: "compare the changed files with a base revision and fail on a regression", run: runCheck},
	"baseline": {summary: "write or check the committed baseline of accepted debt", run: runBaseline},
}

// TRLC-LINKS: REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH, FLOW-ARTIFACT-EXPORT
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run dispatches to the subcommand named by args[0].
//
// TRLC-LINKS: REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func run(args []string, stdout, stderr io.Writer) int {
	if len(args) == 0 {
		usage(stderr)
		return exitUsage
	}
	switch args[0] {
	case "-h", "--help", "help":
		usage(stdout)
		return exitOK
	}
	cmd, ok := commands[args[0]]
	if !ok {
		fmt.Fprintf(stderr, "enghealth: unknown command %q\n", args[0])
		usage(stderr)
		return exitUsage
	}
	return cmd.run(args[1:], stdout, stderr)
}

// usage prints the command list.
//
// TRLC-LINKS: REQ-EMG-158
func usage(w io.Writer) {
	names := make([]string, 0, len(commands))
	for name := range commands {
		names = append(names, name)
	}
	sort.Strings(names)
	fmt.Fprintln(w, "usage: enghealth <command> [flags]")
	fmt.Fprintln(w)
	fmt.Fprintln(w, "commands:")
	for _, name := range names {
		fmt.Fprintf(w, "  %-10s %s\n", name, commands[name].summary)
	}
	fmt.Fprintln(w)
	fmt.Fprintln(w, "run enghealth <command> -h for the flags of a command")
	fmt.Fprintln(w, "exit codes: 0 ok, 1 blocking findings (scan --fail-on-error; check: a regression, a skipped changed file or an unrecorded policy weakening that is an error at the tier), 2 setup error")
}

// runScan scores the code of a model, or re-renders an imported report.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func runScan(args []string, stdout, stderr io.Writer) int {
	a, code := parseScanArgs(args, stderr)
	if code >= 0 {
		return code
	}
	report, code := a.report(stderr)
	if code >= 0 {
		return code
	}
	return finishScan(a, report, streams{stdout, stderr})
}

// scanArgs are the parsed arguments of enghealth scan.
type scanArgs struct {
	modelPath, format, out, tier, importPath string
	failOnError, useCache, noCache           bool
	paths                                    []string
	usage                                    func()
}

// parseScanArgs parses and checks the flags of enghealth scan. The exit code is
// -1 when the arguments are fine and the scan should run.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func parseScanArgs(args []string, stderr io.Writer) (scanArgs, int) {
	flags := flag.NewFlagSet("enghealth scan", flag.ContinueOnError)
	flags.SetOutput(stderr)
	var a scanArgs
	flags.StringVar(&a.modelPath, "model", "", "path to the engmod.yml manifest")
	flags.StringVar(&a.format, "format", "text", "output format: text, json or sarif")
	flags.StringVar(&a.out, "out", "", "write the output to this file instead of stdout")
	flags.StringVar(&a.tier, "tier", "", "scan at this life-cycle tier instead of the one the manifest declares")
	flags.BoolVar(&a.failOnError, "fail-on-error", false, "exit 1 when a finding is an error at the tier")
	flags.StringVar(&a.importPath, "import", "", "re-render this engmod-health/1 report (from another engine) instead of scanning")
	flags.BoolVar(&a.useCache, "cache", false, "use the result cache under .engmod/cache/health although CI is set (it is off in CI by default)")
	flags.BoolVar(&a.noCache, "no-cache", false, "do not read or write the result cache")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "usage: enghealth scan --model <engmod.yml> [--format text|json|sarif] [--out file] [--tier t] [--fail-on-error] [--cache] [--no-cache] [paths...]")
		fmt.Fprintln(stderr, "       enghealth scan --import <report.json> [--model <engmod.yml>] [--tier t] [--format text|json|sarif] [--out file] [--fail-on-error]")
		flags.PrintDefaults()
	}
	a.usage = flags.Usage
	if err := flags.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return a, exitOK
		}
		return a, exitUsage
	}
	a.paths = flags.Args()
	if code := a.validate(stderr); code >= 0 {
		return a, code
	}
	return a, -1
}

// validFormat reports an output format of the commands.
//
// TRLC-LINKS: REQ-EMG-158
func validFormat(format string) bool {
	return format == "text" || format == "json" || format == "sarif"
}

// validate checks the parsed flags: the exit code is -1 when they are fine.
//
// TRLC-LINKS: REQ-EMG-158
func (a scanArgs) validate(stderr io.Writer) int {
	if !validFormat(a.format) {
		fmt.Fprintf(stderr, "enghealth scan: unknown format %q; use text, json or sarif\n", a.format)
		return exitUsage
	}
	if a.useCache && a.noCache {
		fmt.Fprintln(stderr, "enghealth scan: --cache and --no-cache cannot be combined")
		return exitUsage
	}
	return -1
}

// report scans the model, or reads the imported report. The exit code is -1
// when there is a report.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func (a scanArgs) report(stderr io.Writer) (engmodel.HealthReport, int) {
	if strings.TrimSpace(a.importPath) != "" {
		return a.importedReport(stderr)
	}
	if strings.TrimSpace(a.modelPath) == "" {
		a.usage()
		return engmodel.HealthReport{}, exitUsage
	}
	report, err := engmodel.ScanHealth(engmodel.HealthScanOptions{
		ManifestPath: a.modelPath, Tier: a.tier, Paths: a.paths,
		NoCache: a.noCache, UseCache: a.useCache, Notes: stderr,
	})
	if err != nil {
		fmt.Fprintln(stderr, "enghealth scan:", safeText(err.Error()))
		return engmodel.HealthReport{}, exitUsage
	}
	return report, -1
}

// importedReport reads the report of --import. The exit code is -1 when there
// is a report.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func (a scanArgs) importedReport(stderr io.Writer) (engmodel.HealthReport, int) {
	if len(a.paths) > 0 {
		fmt.Fprintln(stderr, "enghealth scan: paths cannot be combined with --import")
		return engmodel.HealthReport{}, exitUsage
	}
	if a.failOnError && strings.TrimSpace(a.modelPath) == "" && strings.TrimSpace(a.tier) == "" {
		fmt.Fprintln(stderr, "enghealth scan: --fail-on-error with --import needs --model or --tier: the levels of the imported report are not trusted")
		return engmodel.HealthReport{}, exitUsage
	}
	report, err := importReport(a.importPath, a.modelPath, a.tier)
	if err != nil {
		fmt.Fprintln(stderr, "enghealth scan:", safeText(err.Error()))
		return engmodel.HealthReport{}, exitUsage
	}
	fmt.Fprintf(stderr, "enghealth scan: read a report of scoring version %d; its scores are taken as they are and not recomputed\n", report.Engine.ScoringVersion)
	return report, -1
}

// finishScan renders the report, writes it and sets the exit code; with
// --fail-on-error a finding, a file below the floor or a stale baseline entry
// at the tier is exit 1, which is how this repository is held at 10.0 for
// every file (REQ-EMG-211).
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158, REQ-EMG-211
func finishScan(a scanArgs, report engmodel.HealthReport, s streams) int {
	if a.failOnError && len(report.Levels) == 0 {
		fmt.Fprintln(s.stderr, "enghealth scan: --fail-on-error needs the levels of the codes, and the report has none")
		return exitUsage
	}
	data, err := renderScan(report, a.format)
	if err != nil {
		fmt.Fprintln(s.stderr, "enghealth scan: render:", err)
		return exitUsage
	}
	if err := writeOutput(data, a.out, s.stdout); err != nil {
		fmt.Fprintln(s.stderr, "enghealth scan: write output:", safeText(err.Error()))
		return exitUsage
	}
	if a.failOnError && report.Summary.Errors > 0 {
		fmt.Fprintf(s.stderr, "enghealth scan: %d finding(s) and diagnostic(s) are errors at tier %s\n", report.Summary.Errors, safeText(tierName(report.Tier)))
		return exitFail
	}
	return exitOK
}

// renderScan renders a report in a format.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func renderScan(report engmodel.HealthReport, format string) ([]byte, error) {
	switch format {
	case "json":
		return engmodel.EncodeHealthReport(report)
	case "sarif":
		return engmodel.HealthSARIF(report)
	}
	return []byte(renderText(report)), nil
}

// importReport reads a report from another engine. With a model or a tier the
// levels of the codes are recomputed at that tier; otherwise the report keeps
// its own.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func importReport(path, modelPath, tier string) (engmodel.HealthReport, error) {
	f, err := os.Open(path)
	if err != nil {
		return engmodel.HealthReport{}, fmt.Errorf("import: %w", err)
	}
	defer f.Close()
	report, err := engmodel.ImportHealthReport(f)
	if err != nil {
		return engmodel.HealthReport{}, err
	}
	if strings.TrimSpace(modelPath) == "" && strings.TrimSpace(tier) == "" {
		return report, nil
	}
	return engmodel.RelevelHealthReport(report, modelPath, tier)
}

// tierName is the tier for messages: "none" for a project without one.
//
// TRLC-LINKS: REQ-EMG-158
func tierName(tier string) string {
	if tier == "" {
		return "(none)"
	}
	return safeText(tier)
}
