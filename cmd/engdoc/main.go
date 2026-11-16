// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/cliflags"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: IF-CLI-ENGDOC, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VIEW-PROJECTION
type viewFlags []string

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VIEW-PROJECTION
func (v *viewFlags) String() string {
	return strings.Join(*v, ",")
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VIEW-PROJECTION
func (v *viewFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*v = append(*v, value)
	return nil
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-009, REQ-EMG-014, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FU-VIEW-PROJECTION, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// ENGMODEL-LINKS: TR-ARTIFACTS-GENERATION-RUN, TR-ARTIFACTS-GENERATION-BLOCKED, TR-ARTIFACTS-VALIDATED, TR-ARTIFACTS-REGENERATED, EVT-VALIDATION-PASSED
func main() {
	args := parseArgs()
	if !args.complete() {
		fmt.Fprintln(os.Stderr, usageLine)
		os.Exit(2)
	}

	res, err := generate(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printDiagnostics(res.Diagnostics)
		os.Exit(1)
	}
	if err := writeOutputs(args, res); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}

	diagnostics := validate.SortDiagnostics(res.Diagnostics)
	printDiagnostics(diagnostics)
	if validate.HasErrors(diagnostics) {
		os.Exit(1)
	}
}

const usageLine = "usage: engdoc --model <engmod.yml> --requirements <model/requirements.yml> --design <model/views.yml> [--code-root <dir>] [--view <id> ...] [--out <file>] [--decisions-out <file>] [--used-by]"

// docArgs holds the parsed engdoc command line.
type docArgs struct {
	modelPath    string
	reqPath      string
	designPath   string
	codeRoot     string
	outPath      string
	decisionsOut string
	usedBy       bool
	views        viewFlags
}

// complete reports whether the three mandatory paths are set.
//
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-009, REQ-EMG-014, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FU-VIEW-PROJECTION, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func (a *docArgs) complete() bool {
	return strings.TrimSpace(a.modelPath) != "" && strings.TrimSpace(a.reqPath) != "" && strings.TrimSpace(a.designPath) != ""
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-009, REQ-EMG-014, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FU-VIEW-PROJECTION, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func parseArgs() *docArgs {
	a := &docArgs{}
	flag.StringVar(&a.modelPath, "model", "", "path to engmod.yml manifest")
	flag.StringVar(&a.reqPath, "requirements", "", "path to requirements YAML")
	flag.StringVar(&a.designPath, "design", "", "path to design mapping YAML")
	flag.StringVar(&a.codeRoot, "code-root", "", "optional source tree root for TRACE-* code mapping")
	flag.StringVar(&a.outPath, "out", "", "optional output .adoc path; defaults to stdout")
	flag.StringVar(&a.decisionsOut, "decisions-out", "", "optional output path for generated architecture decision records .adoc")
	flag.BoolVar(&a.usedBy, "used-by", false, "add a Used By section listing the requirement links of the other checkouts of the active workspace onto this model's contract items (render-time only; do not commit the output into the child)")
	flag.Var(&a.views, "view", "optional viewpoint ID; repeat to include multiple views")
	modelEnv := cliflags.Register(nil)
	flag.Parse()
	modelEnv.MustApply()
	return a
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-009, REQ-EMG-014, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FU-VIEW-PROJECTION, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func generate(a *docArgs) (engmodel.AsciiDocResult, error) {
	decisionsDocPath := strings.TrimSpace(a.decisionsOut)
	if decisionsDocPath != "" {
		decisionsDocPath = filepath.Base(decisionsDocPath)
	}
	return engmodel.GenerateAsciiDocFromFiles(engmodel.AsciiDocFiles{Architecture: a.modelPath, Requirements: a.reqPath, Design: a.designPath}, engmodel.AsciiDocOptions{
		ViewIDs:          a.views,
		CodeRoot:         strings.TrimSpace(a.codeRoot),
		DecisionsDocPath: decisionsDocPath,
		UsedBy:           a.usedBy,
	})
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-009, REQ-EMG-014, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FU-VIEW-PROJECTION, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func writeOutputs(a *docArgs, res engmodel.AsciiDocResult) error {
	if strings.TrimSpace(a.outPath) == "" {
		_, _ = os.Stdout.WriteString(res.Document)
	} else if err := os.WriteFile(a.outPath, []byte(res.Document), 0o644); err != nil {
		return fmt.Errorf("error writing output: %w", err)
	}
	if strings.TrimSpace(a.decisionsOut) != "" {
		if err := os.WriteFile(a.decisionsOut, []byte(res.DecisionsDocument), 0o644); err != nil {
			return fmt.Errorf("error writing decisions output: %w", err)
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
// ENGMODEL-LINKS: IF-CLI-ENGDOC, FU-CLI-ORCHESTRATION, FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func printDiagnostics(diags []validate.Diagnostic) {
	for _, d := range diags {
		validate.WriteDiagnostic(os.Stderr, d)
	}
}
