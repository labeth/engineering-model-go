package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

func main() {
	modelPath := flag.String("model", "", "path to architecture YAML (e.g. 03-architecture-model.yml)")
	viewID := flag.String("view", "", "viewpoint ID to render")
	outPath := flag.String("out", "", "optional output file path; defaults to stdout")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" || strings.TrimSpace(*viewID) == "" {
		fmt.Fprintln(os.Stderr, "usage: engview --model <path> --view <id> [--out <file>]")
		os.Exit(2)
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

func printDiagnostics(diags []validate.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s [%s] %s", d.Code, d.Severity, d.Message)
		if strings.TrimSpace(d.Path) != "" {
			fmt.Fprintf(os.Stderr, " (%s)", d.Path)
		}
		fmt.Fprintln(os.Stderr)
	}
}
