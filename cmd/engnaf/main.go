// ENGMODEL-OWNER-UNIT: FU-NAF-EXPORTER
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
)

// TRLC-LINKS: REQ-EMG-042, REQ-EMG-043
// ENGMODEL-LINKS: FU-NAF-EXPORTER, IF-CLI-ENGNAF, DO-NAF-V4-ARCHITECTURE
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// TRLC-LINKS: REQ-EMG-042, REQ-EMG-043
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("engnaf", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modelPath := flags.String("model", "", "path to engmod.yml manifest")
	outPath := flags.String("out", "", "optional output file path; defaults to stdout")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if strings.TrimSpace(*modelPath) == "" {
		fmt.Fprintln(stderr, "usage: engnaf --model <engmod.yml> [--out <file>]")
		return 2
	}
	result, err := engmodel.GenerateNAFV41FromFile(*modelPath)
	for _, diagnostic := range result.Diagnostics {
		fmt.Fprintf(stderr, "%s [%s] %s", diagnostic.Code, diagnostic.Severity, diagnostic.Message)
		if diagnostic.Path != "" {
			fmt.Fprintf(stderr, " (%s)", diagnostic.Path)
		}
		fmt.Fprintln(stderr)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	if strings.TrimSpace(*outPath) == "" {
		_, _ = io.WriteString(stdout, result.Document)
		return 0
	}
	if err := os.WriteFile(strings.TrimSpace(*outPath), []byte(result.Document), 0o644); err != nil {
		fmt.Fprintln(stderr, "error writing output:", err)
		return 1
	}
	return 0
}
