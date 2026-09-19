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
)

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("engsysml", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modelPath := flags.String("model", "", "path to architecture YAML")
	outPath := flags.String("out", "", "optional output file path; defaults to stdout")
	coverage := flags.Bool("coverage", false, "emit the SysML v2 coverage manifest as JSON")
	validateCoverage := flags.String("validate-coverage", "", "validate a generated SysML v2 coverage manifest and its freshness")
	projectOut := flags.String("project-out", "", "create a Sysand interchange project in this directory")
	kparOut := flags.String("kpar-out", "", "build a normative KPAR at this path using Sysand")
	verifyKPAR := flags.String("verify-kpar", "", "reopen this KPAR with Sysand and compare it to --model")
	reopenOut := flags.String("reopen-out", "", "empty directory for --verify-kpar")
	sysandPath := flags.String("sysand", "", "path to the pinned Sysand executable; defaults to PATH")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if *coverage {
		manifest := engmodel.SysMLV2Coverage()
		if err := engmodel.ValidateSysMLV2Coverage(manifest); err != nil {
			fmt.Fprintln(stderr, "error validating coverage manifest:", err)
			return 1
		}
		data, err := json.MarshalIndent(manifest, "", "  ")
		if err != nil {
			fmt.Fprintln(stderr, "error encoding coverage manifest:", err)
			return 1
		}
		data = append(data, '\n')
		return writeOutput(strings.TrimSpace(*outPath), data, stdout, stderr)
	}

	if strings.TrimSpace(*validateCoverage) != "" {
		data, err := os.ReadFile(strings.TrimSpace(*validateCoverage))
		if err != nil {
			fmt.Fprintln(stderr, "error reading coverage manifest:", err)
			return 1
		}
		var manifest engmodel.SysMLCoverageManifest
		if err := json.Unmarshal(data, &manifest); err != nil {
			fmt.Fprintln(stderr, "error decoding coverage manifest:", err)
			return 1
		}
		if err := engmodel.ValidateSysMLV2Coverage(manifest); err != nil {
			fmt.Fprintln(stderr, "error validating coverage manifest:", err)
			return 1
		}
		expected, err := json.MarshalIndent(engmodel.SysMLV2Coverage(), "", "  ")
		if err != nil {
			fmt.Fprintln(stderr, "error encoding coverage manifest:", err)
			return 1
		}
		expected = append(expected, '\n')
		if !bytes.Equal(data, expected) {
			fmt.Fprintln(stderr, "coverage manifest is stale; regenerate with go run ./cmd/engsysml --coverage --out generated/SYSML-COVERAGE.json")
			return 1
		}
		fmt.Fprintln(stdout, "SysML coverage manifest is complete and current")
		return 0
	}

	if strings.TrimSpace(*modelPath) == "" {
		fmt.Fprintln(stderr, "usage: engsysml --model <path> [--out <file>] [--project-out <dir> --kpar-out <file>] | --model <path> --verify-kpar <file> --reopen-out <dir> | --coverage | --validate-coverage <file>")
		return 2
	}

	if strings.TrimSpace(*verifyKPAR) != "" {
		if strings.TrimSpace(*reopenOut) == "" {
			fmt.Fprintln(stderr, "error: --verify-kpar requires --reopen-out")
			return 2
		}
		if _, err := engmodel.VerifySysMLV2RoundTrip(*modelPath, *verifyKPAR, *reopenOut, *sysandPath); err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		fmt.Fprintln(stdout, "SysML KPAR semantic round trip verified")
		return 0
	}

	if strings.TrimSpace(*projectOut) != "" || strings.TrimSpace(*kparOut) != "" {
		if strings.TrimSpace(*projectOut) == "" || strings.TrimSpace(*kparOut) == "" {
			fmt.Fprintln(stderr, "error: --project-out and --kpar-out must be used together")
			return 2
		}
		result, err := engmodel.ExportSysMLV2Project(*modelPath, *projectOut, *kparOut, *sysandPath)
		printDiagnostics(stderr, result)
		if err != nil {
			fmt.Fprintln(stderr, "error:", err)
			return 1
		}
		return 0
	}

	result, err := engmodel.GenerateSysMLV2FromFile(*modelPath)
	printDiagnostics(stderr, result)
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}
	return writeOutput(strings.TrimSpace(*outPath), []byte(result.Text), stdout, stderr)
}

// TRLC-LINKS: REQ-EMG-036
func writeOutput(path string, data []byte, stdout, stderr io.Writer) int {
	if path == "" {
		_, _ = stdout.Write(data)
		return 0
	}
	if err := os.WriteFile(path, data, 0o644); err != nil {
		fmt.Fprintln(stderr, "error writing output:", err)
		return 1
	}
	return 0
}

// TRLC-LINKS: REQ-EMG-036
func printDiagnostics(stderr io.Writer, result engmodel.SysMLExportResult) {
	for _, diagnostic := range result.Diagnostics {
		fmt.Fprintf(stderr, "%s [%s] %s", diagnostic.Code, diagnostic.Severity, diagnostic.Message)
		if diagnostic.Path != "" {
			fmt.Fprintf(stderr, " (%s)", diagnostic.Path)
		}
		fmt.Fprintln(stderr)
	}
}
