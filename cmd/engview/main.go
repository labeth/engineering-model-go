// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func main() {
	modelPath := flag.String("model", "", "path to engmod.yml manifest")
	viewID := flag.String("view", "", "viewpoint ID to render")
	outPath := flag.String("out", "", "optional output file path; defaults to stdout")
	outDir := flag.String("out-dir", "", "optional directory for rendering every authored view as <view-id>.mmd and <view-id>.svg")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" || (strings.TrimSpace(*viewID) == "" && strings.TrimSpace(*outDir) == "") {
		fmt.Fprintln(os.Stderr, "usage: engview --model <engmod.yml> (--view <id> [--out <file>] | --out-dir <dir>)")
		os.Exit(2)
	}
	if strings.TrimSpace(*outDir) != "" {
		if strings.TrimSpace(*viewID) != "" || strings.TrimSpace(*outPath) != "" {
			fmt.Fprintln(os.Stderr, "--out-dir cannot be combined with --view or --out")
			os.Exit(2)
		}
		renderAllViews(*modelPath, *outDir)
		return
	}

	res, err := engmodel.GenerateFromFile(*modelPath, *viewID)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printDiagnostics(res.Diagnostics)
		os.Exit(1)
	}

	if strings.TrimSpace(*outPath) == "" {
		_, _ = os.Stdout.WriteString(res.Mermaid)
	} else {
		if err := os.WriteFile(*outPath, []byte(res.Mermaid), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing output:", err)
			os.Exit(1)
		}
	}

	printDiagnostics(res.Diagnostics)
	if validate.HasErrors(res.Diagnostics) {
		os.Exit(1)
	}
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION
func renderAllViews(modelPath, outDir string) {
	bundle, err := model.LoadBundle(modelPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating output directory:", err)
		os.Exit(1)
	}
	for _, authoredView := range bundle.Architecture.Views {
		res, err := engmodel.GenerateFromFile(modelPath, authoredView.ID)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(res.Diagnostics)
			os.Exit(1)
		}
		path := filepath.Join(outDir, authoredView.ID+".mmd")
		if err := os.WriteFile(path, []byte(res.Mermaid), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing output:", err)
			os.Exit(1)
		}
		svgPath := filepath.Join(outDir, authoredView.ID+".svg")
		if err := os.WriteFile(svgPath, []byte(res.SVG), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing output:", err)
			os.Exit(1)
		}
		printDiagnostics(res.Diagnostics)
		if validate.HasErrors(res.Diagnostics) {
			os.Exit(1)
		}
	}
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003
// ENGMODEL-LINKS: IF-CLI-ENGVIEW, FU-CLI-ORCHESTRATION, FU-VIEW-PROJECTION, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func printDiagnostics(diags []validate.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s [%s] %s", d.Code, d.Severity, d.Message)
		if strings.TrimSpace(d.Path) != "" {
			fmt.Fprintf(os.Stderr, " (%s)", d.Path)
		}
		fmt.Fprintln(os.Stderr)
	}
}
