// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/export/exportgate"
	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-055, REQ-EMG-056, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGAIR, FU-AIRBORNE-ASSURANCE-EXPORTER
func main() {
	modelPath := flag.String("model", "", "path to engmod.yml manifest")
	outDir := flag.String("out-dir", "", "directory for generated readiness outputs")
	date := flag.String("date", "", "deterministic generated date")
	version := flag.String("version", "", "deterministic package version")
	flag.Parse()
	if strings.TrimSpace(*modelPath) == "" || strings.TrimSpace(*outDir) == "" {
		fmt.Fprintln(os.Stderr, "usage: engair --model <engmod.yml> --out-dir <dir> [--date <date>] [--version <version>]")
		os.Exit(2)
	}
	exitOnFailure(validateModel(*modelPath))
	result, err := engmodel.GenerateAirborneAssuranceFromFile(*modelPath, engmodel.AirborneAssuranceOptions{Date: strings.TrimSpace(*date), Version: strings.TrimSpace(*version)})
	if errors.Is(err, engmodel.ErrNoAviationProfile) {
		fmt.Fprintln(os.Stderr, "skip:", err)
		return
	}
	exitOnFailure(reportGenerateError(err, result.Diagnostics))
	exitOnFailure(writeOutputs(*outDir, result.Files))
}

// exitOnFailure exits with status 1 when failure is non-nil.
//
// TRLC-LINKS: REQ-EMG-055, REQ-EMG-056, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGAIR, FU-AIRBORNE-ASSURANCE-EXPORTER
func exitOnFailure(failure error) {
	if failure != nil {
		os.Exit(1)
	}
}

// validateModel runs the validation gate, which refuses a model with errors
// before the aviation profile check can skip it silently. The returned error
// is non-nil after the message was printed.
//
// TRLC-LINKS: REQ-EMG-055, REQ-EMG-056, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGAIR, FU-AIRBORNE-ASSURANCE-EXPORTER
func validateModel(modelPath string) error {
	_, ok, err := exportgate.ValidationGate(modelPath, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading model:", err)
		return err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "error: model validation failed; no readiness output written")
		return errors.New("model validation failed")
	}
	return nil
}

// reportGenerateError prints a generation error and its diagnostics.
//
// TRLC-LINKS: REQ-EMG-055, REQ-EMG-056, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGAIR, FU-AIRBORNE-ASSURANCE-EXPORTER
func reportGenerateError(err error, diagnostics []model.AviationDiagnostic) error {
	if err == nil {
		return nil
	}
	fmt.Fprintln(os.Stderr, "error:", err)
	for _, diagnostic := range diagnostics {
		fmt.Fprintf(os.Stderr, "%s %s: %s\n", diagnostic.Code, diagnostic.Path, diagnostic.Message)
	}
	return err
}

// writeOutputs writes the generated files in name order into outDir.
//
// TRLC-LINKS: REQ-EMG-055, REQ-EMG-056, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGAIR, FU-AIRBORNE-ASSURANCE-EXPORTER
func writeOutputs(outDir string, files map[string][]byte) error {
	if err := os.MkdirAll(outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating output directory:", err)
		return err
	}
	names := make([]string, 0, len(files))
	for name := range files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(outDir, name), files[name], 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing output:", err)
			return err
		}
	}
	return nil
}
