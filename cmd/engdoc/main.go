package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

type viewFlags []string

func (v *viewFlags) String() string {
	return strings.Join(*v, ",")
}

func (v *viewFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*v = append(*v, value)
	return nil
}

func main() {
	modelPath := flag.String("model", "", "path to architecture model YAML")
	reqPath := flag.String("requirements", "", "path to requirements YAML")
	designPath := flag.String("design", "", "path to design mapping YAML")
	codeRoot := flag.String("code-root", "", "optional source tree root for TRACE-* code mapping")
	outPath := flag.String("out", "", "optional output .adoc path; defaults to stdout")
	var views viewFlags
	flag.Var(&views, "view", "optional viewpoint ID; repeat to include multiple views")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" || strings.TrimSpace(*reqPath) == "" || strings.TrimSpace(*designPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: engdoc --model <architecture.yml> --requirements <requirements.yml> --design <design.yml> [--code-root <dir>] [--view <id> ...] [--out <file>]")
		os.Exit(2)
	}

	res, err := engmodel.GenerateAsciiDocFromFiles(*modelPath, *reqPath, *designPath, engmodel.AsciiDocOptions{
		ViewIDs:  views,
		CodeRoot: strings.TrimSpace(*codeRoot),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printDiagnostics(res.Diagnostics)
		os.Exit(1)
	}

	if strings.TrimSpace(*outPath) == "" {
		_, _ = os.Stdout.WriteString(res.Document)
	} else {
		if err := os.WriteFile(*outPath, []byte(res.Document), 0o644); err != nil {
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
