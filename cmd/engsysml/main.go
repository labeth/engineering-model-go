// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN, FLOW-ARTIFACT-EXPORT
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// sysmlArgs holds the parsed engsysml command line.
type sysmlArgs struct {
	modelPath        string
	outPath          string
	coverage         bool
	validateCoverage string
	projectOut       string
	kparOut          string
	verifyKPAR       string
	reopenOut        string
	sysandPath       string
}

// wantsProject reports whether a Sysand project or KPAR output was requested.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (a *sysmlArgs) wantsProject() bool {
	return strings.TrimSpace(a.projectOut) != "" || strings.TrimSpace(a.kparOut) != ""
}

// sysmlCLI is one engsysml invocation with its output streams.
type sysmlCLI struct {
	args   sysmlArgs
	stdout io.Writer
	stderr io.Writer
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func parseArgs(args []string, stderr io.Writer) (sysmlArgs, error) {
	var a sysmlArgs
	flags := flag.NewFlagSet("engsysml", flag.ContinueOnError)
	flags.SetOutput(stderr)
	flags.StringVar(&a.modelPath, "model", "", "path to engmod.yml manifest")
	flags.StringVar(&a.outPath, "out", "", "optional output file path; defaults to stdout")
	flags.BoolVar(&a.coverage, "coverage", false, "emit the SysML v2 coverage manifest as JSON")
	flags.StringVar(&a.validateCoverage, "validate-coverage", "", "validate a generated SysML v2 coverage manifest and its freshness")
	flags.StringVar(&a.projectOut, "project-out", "", "create a Sysand interchange project in this directory")
	flags.StringVar(&a.kparOut, "kpar-out", "", "build a normative KPAR at this path using Sysand")
	flags.StringVar(&a.verifyKPAR, "verify-kpar", "", "reopen this KPAR with Sysand and compare it to --model")
	flags.StringVar(&a.reopenOut, "reopen-out", "", "empty directory for --verify-kpar")
	flags.StringVar(&a.sysandPath, "sysand", "", "path to the pinned Sysand executable; defaults to PATH")
	return a, flags.Parse(args)
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func run(args []string, stdout, stderr io.Writer) int {
	parsed, err := parseArgs(args, stderr)
	if err != nil {
		return 2
	}
	c := &sysmlCLI{args: parsed, stdout: stdout, stderr: stderr}
	return c.dispatch()
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (c *sysmlCLI) dispatch() int {
	a := c.args
	if a.coverage {
		return c.emitCoverage()
	}
	if strings.TrimSpace(a.validateCoverage) != "" {
		return c.checkCoverage()
	}
	if strings.TrimSpace(a.modelPath) == "" {
		fmt.Fprintln(c.stderr, "usage: engsysml --model <engmod.yml> [--out <file>] [--project-out <dir> --kpar-out <file>] | --model <engmod.yml> --verify-kpar <file> --reopen-out <dir> | --coverage | --validate-coverage <file>")
		return 2
	}
	if strings.TrimSpace(a.verifyKPAR) != "" {
		return c.verifyRoundTrip()
	}
	if a.wantsProject() {
		return c.exportProject()
	}
	return c.exportSource()
}

// encodeCoverage renders a coverage manifest as indented JSON.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func encodeCoverage(manifest engmodel.SysMLCoverageManifest) ([]byte, error) {
	data, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (c *sysmlCLI) emitCoverage() int {
	manifest := engmodel.SysMLV2Coverage()
	if err := engmodel.ValidateSysMLV2Coverage(manifest); err != nil {
		fmt.Fprintln(c.stderr, "error validating coverage manifest:", err)
		return 1
	}
	data, err := encodeCoverage(manifest)
	if err != nil {
		fmt.Fprintln(c.stderr, "error encoding coverage manifest:", err)
		return 1
	}
	return c.writeOutput(strings.TrimSpace(c.args.outPath), data)
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (c *sysmlCLI) checkCoverage() int {
	data, err := os.ReadFile(strings.TrimSpace(c.args.validateCoverage))
	if err != nil {
		fmt.Fprintln(c.stderr, "error reading coverage manifest:", err)
		return 1
	}
	var manifest engmodel.SysMLCoverageManifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		fmt.Fprintln(c.stderr, "error decoding coverage manifest:", err)
		return 1
	}
	if err := engmodel.ValidateSysMLV2Coverage(manifest); err != nil {
		fmt.Fprintln(c.stderr, "error validating coverage manifest:", err)
		return 1
	}
	expected, err := encodeCoverage(engmodel.SysMLV2Coverage())
	if err != nil {
		fmt.Fprintln(c.stderr, "error encoding coverage manifest:", err)
		return 1
	}
	if !bytes.Equal(data, expected) {
		fmt.Fprintln(c.stderr, "coverage manifest is stale; regenerate with go run ./cmd/engsysml --coverage --out generated/SYSML-COVERAGE.json")
		return 1
	}
	fmt.Fprintln(c.stdout, "SysML coverage manifest is complete and current")
	return 0
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (c *sysmlCLI) verifyRoundTrip() int {
	a := c.args
	if strings.TrimSpace(a.reopenOut) == "" {
		fmt.Fprintln(c.stderr, "error: --verify-kpar requires --reopen-out")
		return 2
	}
	if _, err := engmodel.VerifySysMLV2RoundTrip(a.modelPath, a.verifyKPAR, a.reopenOut, a.sysandPath); err != nil {
		fmt.Fprintln(c.stderr, "error:", err)
		return 1
	}
	fmt.Fprintln(c.stdout, "SysML KPAR native source round trip verified")
	return 0
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (c *sysmlCLI) exportProject() int {
	a := c.args
	if strings.TrimSpace(a.projectOut) == "" || strings.TrimSpace(a.kparOut) == "" {
		fmt.Fprintln(c.stderr, "error: --project-out and --kpar-out must be used together")
		return 2
	}
	result, err := engmodel.ExportSysMLV2Project(a.modelPath, a.projectOut, a.kparOut, a.sysandPath)
	printDiagnostics(c.stderr, result)
	if err != nil {
		fmt.Fprintln(c.stderr, "error:", err)
		return 1
	}
	return 0
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func (c *sysmlCLI) exportSource() int {
	result, err := engmodel.GenerateSysMLV2FromFile(c.args.modelPath)
	printDiagnostics(c.stderr, result)
	if err != nil {
		fmt.Fprintln(c.stderr, "error:", err)
		return 1
	}
	return c.writeOutput(strings.TrimSpace(c.args.outPath), []byte(result.Text))
}

// TRLC-LINKS: REQ-EMG-036
func (c *sysmlCLI) writeOutput(path string, data []byte) int {
	if path == "" {
		_, _ = c.stdout.Write(data)
		return 0
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintln(c.stderr, "error writing output:", err)
		return 1
	}
	return 0
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-168
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML
func printDiagnostics(stderr io.Writer, result engmodel.SysMLExportResult) {
	for _, diagnostic := range result.PrintableDiagnostics() {
		validate.WriteDiagnostic(stderr, diagnostic)
	}
}
