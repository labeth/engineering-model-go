// ENGMODEL-OWNER-UNIT: FU-TRLC-EXPORTER
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
)

// TRLC-LINKS: REQ-EMG-006
func main() {
	requirementsPath := flag.String("requirements", "", "path to requirements YAML")
	outDir := flag.String("out-dir", "", "output directory for model.rsl and requirements.trlc")
	packageName := flag.String("package", "", "optional TRLC package name")
	flag.Parse()

	if strings.TrimSpace(*requirementsPath) == "" || strings.TrimSpace(*outDir) == "" {
		fmt.Fprintln(os.Stderr, "usage: engtrlc --requirements <requirements.yml> --out-dir <dir> [--package <Name>]")
		os.Exit(2)
	}

	res, err := engmodel.GenerateTRLCRequirementsFromFile(*requirementsPath, engmodel.TRLCExportOptions{PackageName: strings.TrimSpace(*packageName)})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	out := strings.TrimSpace(*outDir)
	if err := os.MkdirAll(out, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating output directory:", err)
		os.Exit(1)
	}

	modelPath := filepath.Join(out, "model.rsl")
	reqsPath := filepath.Join(out, "requirements.trlc")
	if err := os.WriteFile(modelPath, []byte(res.ModelRSL), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing model.rsl:", err)
		os.Exit(1)
	}
	if err := os.WriteFile(reqsPath, []byte(res.RequirementsTRLC), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing requirements.trlc:", err)
		os.Exit(1)
	}

	fmt.Fprintf(os.Stdout, "Generated TRLC package %q\n", res.PackageName)
	fmt.Fprintf(os.Stdout, "- %s\n", modelPath)
	fmt.Fprintf(os.Stdout, "- %s\n", reqsPath)
}
