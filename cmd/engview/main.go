// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FLOW-ARTIFACT-EXPORT, DO-VIEW-DIAGRAM
func main() {
	os.Exit(run(parseArgs()))
}

// viewArgs holds the parsed engview command line.
type viewArgs struct {
	modelPath string
	viewID    string
	outPath   string
	outDir    string
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FLOW-ARTIFACT-EXPORT, DO-VIEW-DIAGRAM
func parseArgs() *viewArgs {
	a := &viewArgs{}
	flag.StringVar(&a.modelPath, "model", "", "path to engmod.yml manifest")
	flag.StringVar(&a.viewID, "view", "", "viewpoint ID to render")
	flag.StringVar(&a.outPath, "out", "", "optional output file path; defaults to stdout")
	flag.StringVar(&a.outDir, "out-dir", "", "optional directory for rendering every authored view as <view-id>.mmd and <view-id>.svg")
	flag.Parse()
	return a
}

// run executes the command and returns the process exit code.
//
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FLOW-ARTIFACT-EXPORT, DO-VIEW-DIAGRAM
func run(a *viewArgs) int {
	if strings.TrimSpace(a.modelPath) == "" || (strings.TrimSpace(a.viewID) == "" && strings.TrimSpace(a.outDir) == "") {
		fmt.Fprintln(os.Stderr, "usage: engview --model <engmod.yml> (--view <id> [--out <file>] | --out-dir <dir>)")
		return 2
	}
	if strings.TrimSpace(a.outDir) == "" {
		return a.renderOne()
	}
	if strings.TrimSpace(a.viewID) != "" || strings.TrimSpace(a.outPath) != "" {
		fmt.Fprintln(os.Stderr, "--out-dir cannot be combined with --view or --out")
		return 2
	}
	return a.renderAllViews()
}

// renderOne renders the single requested view to --out or stdout.
//
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FLOW-ARTIFACT-EXPORT, DO-VIEW-DIAGRAM
func (a *viewArgs) renderOne() int {
	res, err := engmodel.GenerateFromFile(a.modelPath, a.viewID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printDiagnostics(res.Diagnostics)
		return 1
	}

	if strings.TrimSpace(a.outPath) == "" {
		_, _ = os.Stdout.WriteString(res.Mermaid)
	} else if err := writeOutput(a.outPath, res.Mermaid); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	printDiagnostics(res.Diagnostics)
	if validate.HasErrors(res.Diagnostics) {
		return 1
	}
	return 0
}

// writeOutput writes content to path and wraps a failure as the command's
// output error.
//
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FLOW-ARTIFACT-EXPORT, DO-VIEW-DIAGRAM
func writeOutput(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("error writing output: %w", err)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION
func (a *viewArgs) renderAllViews() int {
	bundle, err := model.LoadBundle(a.modelPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if err := os.MkdirAll(a.outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating output directory:", err)
		return 1
	}
	for _, authoredView := range bundle.Architecture.Views {
		if code := a.renderView(authoredView.ID); code != 0 {
			return code
		}
	}
	return 0
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION
func (a *viewArgs) renderView(id string) int {
	res, err := engmodel.GenerateFromFile(a.modelPath, id)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printDiagnostics(res.Diagnostics)
		return 1
	}
	if err := writeOutput(filepath.Join(a.outDir, id+".mmd"), res.Mermaid); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	if err := writeOutput(filepath.Join(a.outDir, id+".svg"), res.SVG); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	printDiagnostics(res.Diagnostics)
	if validate.HasErrors(res.Diagnostics) {
		return 1
	}
	return 0
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func printDiagnostics(diags []validate.Diagnostic) {
	for _, d := range diags {
		validate.WriteDiagnostic(os.Stderr, d)
	}
}
