// ENGMODEL-OWNER-UNIT: FU-TRLC-EXPORTER
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/export/exportgate"
)

// TRLC-LINKS: REQ-EMG-006
// ENGMODEL-LINKS: IF-CLI-ENGTRLC, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run validates the model, then exports its requirements: a model with
// errors (a missing lifecycle among them) is refused with its diagnostics and
// the fix of each required-switch error, and nothing is written.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGTRLC, FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("engtrlc", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modelPath := flags.String("model", "", "path to engmod.yml manifest of the model the requirements belong to")
	requirementsPath := flags.String("requirements", "", "path to requirements YAML")
	outDir := flags.String("out-dir", "", "output directory for model.rsl and requirements.trlc")
	packageName := flags.String("package", "", "optional TRLC package name")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if anyBlank(*modelPath, *requirementsPath, *outDir) {
		fmt.Fprintln(stderr, "usage: engtrlc --model <engmod.yml> --requirements <model/requirements.yml> --out-dir <dir> [--package <Name>]")
		return 2
	}

	if code := gateExit(*modelPath, stderr); code != 0 {
		return code
	}

	res, err := engmodel.GenerateTRLCRequirementsFromFile(*requirementsPath, engmodel.TRLCExportOptions{PackageName: strings.TrimSpace(*packageName)})
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	out := strings.TrimSpace(*outDir)
	rslPath := filepath.Join(out, "model.rsl")
	reqsPath := filepath.Join(out, "requirements.trlc")
	if code := writeTRLC(out, res, stderr); code != 0 {
		return code
	}

	fmt.Fprintf(stdout, "Generated TRLC package %q\n", res.PackageName)
	fmt.Fprintf(stdout, "- %s\n", rslPath)
	fmt.Fprintf(stdout, "- %s\n", reqsPath)
	return 0
}

// anyBlank reports whether any of the values is empty after trimming.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGTRLC, FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func anyBlank(values ...string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return true
		}
	}
	return false
}

// gateExit runs the validation gate and returns 0 when the model passes, 1
// after printing why it does not.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGTRLC, FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func gateExit(modelPath string, stderr io.Writer) int {
	_, ok, err := exportgate.ValidationGate(modelPath, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "error loading model:", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(stderr, "error: model validation failed; no TRLC written")
		return 1
	}
	return 0
}

// writeTRLC creates out and writes model.rsl and requirements.trlc into it,
// returning 0 or 1 after printing the failure.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGTRLC, FU-TRLC-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func writeTRLC(out string, res engmodel.TRLCExportResult, stderr io.Writer) int {
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintln(stderr, "error creating output directory:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(out, "model.rsl"), []byte(res.ModelRSL), 0o644); err != nil {
		fmt.Fprintln(stderr, "error writing model.rsl:", err)
		return 1
	}
	if err := os.WriteFile(filepath.Join(out, "requirements.trlc"), []byte(res.RequirementsTRLC), 0o644); err != nil {
		fmt.Fprintln(stderr, "error writing requirements.trlc:", err)
		return 1
	}
	return 0
}
