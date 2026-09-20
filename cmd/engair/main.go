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
)

// TRLC-LINKS: REQ-EMG-055, REQ-EMG-056
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
	result, err := engmodel.GenerateAirborneAssuranceFromFile(*modelPath, engmodel.AirborneAssuranceOptions{Date: strings.TrimSpace(*date), Version: strings.TrimSpace(*version)})
	if errors.Is(err, engmodel.ErrNoAviationProfile) {
		fmt.Fprintln(os.Stderr, "skip:", err)
		return
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		for _, diagnostic := range result.Diagnostics {
			fmt.Fprintf(os.Stderr, "%s %s: %s\n", diagnostic.Code, diagnostic.Path, diagnostic.Message)
		}
		os.Exit(1)
	}
	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating output directory:", err)
		os.Exit(1)
	}
	names := make([]string, 0, len(result.Files))
	for name := range result.Files {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(*outDir, name), result.Files[name], 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing output:", err)
			os.Exit(1)
		}
	}
}
