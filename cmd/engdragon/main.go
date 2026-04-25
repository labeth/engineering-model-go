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
	modelPath := flag.String("model", "", "path to architecture model YAML")
	format := flag.String("format", string(engmodel.ThreatModelFormatThreatDragonV2), "export format: threat-dragon-v2 or open-otm")
	outPath := flag.String("out", "", "optional output .json path; defaults to stdout")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: engdragon --model <architecture.yml> [--format threat-dragon-v2|open-otm] [--out <file>]")
		os.Exit(2)
	}

	res, err := engmodel.GenerateThreatModelExportFromFile(*modelPath, engmodel.ThreatModelExportOptions{Format: engmodel.ThreatModelFormat(strings.TrimSpace(*format))})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printDiagnostics(res.Diagnostics)
		os.Exit(1)
	}

	if strings.TrimSpace(*outPath) == "" {
		_, _ = os.Stdout.WriteString(res.JSON)
	} else {
		if err := os.WriteFile(*outPath, []byte(res.JSON), 0o644); err != nil {
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
