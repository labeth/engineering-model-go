// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
func main() {
	modelPath := flag.String("model", "", "path to architecture YAML")
	requirementsPath := flag.String("requirements", "", "path to requirements YAML")
	codeRoot := flag.String("code-root", "", "code root to scan for trace links")
	out := flag.String("out", "", "output file (default stdout)")
	format := flag.String("format", "json", "output format: json|csv")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" || strings.TrimSpace(*requirementsPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: engtrace --model <architecture.yml> --requirements <requirements.yml> [--code-root <dir>] [--format json|csv] [--out <file>]")
		os.Exit(2)
	}

	matrix, diags, err := engmodel.BuildTraceMatrixFromFiles(*modelPath, *requirementsPath, *codeRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	var data []byte
	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "csv":
		data = matrix.CSV()
	case "json", "":
		b, merr := json.MarshalIndent(matrix, "", "  ")
		if merr != nil {
			fmt.Fprintln(os.Stderr, "error encoding matrix:", merr)
			os.Exit(1)
		}
		data = append(b, '\n')
	default:
		fmt.Fprintln(os.Stderr, "unknown format:", *format)
		os.Exit(2)
	}

	if o := strings.TrimSpace(*out); o != "" {
		if werr := os.WriteFile(o, data, 0o644); werr != nil {
			fmt.Fprintln(os.Stderr, "error writing output:", werr)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "Wrote traceability matrix: %s (%d requirements, %d implemented, %d verified, %d delegated, %d orphan, %d dangling)\n",
			o, matrix.Summary.Requirements, matrix.Summary.Implemented, matrix.Summary.Verified, matrix.Summary.Delegated, matrix.Summary.Orphan, matrix.Summary.DanglingLinks)
	} else {
		os.Stdout.Write(data)
	}

	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s [%s] %s\n", d.Code, d.Severity, d.Message)
	}
	if validate.HasErrors(diags) {
		os.Exit(1)
	}
}
