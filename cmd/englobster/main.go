package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
)

func main() {
	testsDir := flag.String("tests-dir", "", "path to tests directory")
	reqPackage := flag.String("requirements-package", "Requirements", "TRLC requirements package name")
	activityNamespace := flag.String("activity-namespace", "tests", "activity tag namespace")
	outPath := flag.String("out", "", "output .lobster JSON file")
	flag.Parse()

	if strings.TrimSpace(*testsDir) == "" || strings.TrimSpace(*outPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: englobster --tests-dir <dir> --out <activity.lobster> [--requirements-package <pkg>] [--activity-namespace <ns>]")
		os.Exit(2)
	}

	res, err := engmodel.GenerateLobsterActivityTraceFromDir(strings.TrimSpace(*testsDir), engmodel.LobsterActivityExportOptions{
		RequirementsPackage: strings.TrimSpace(*reqPackage),
		ActivityNamespace:   strings.TrimSpace(*activityNamespace),
	})
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	if err := os.WriteFile(strings.TrimSpace(*outPath), []byte(res.JSON), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing output:", err)
		os.Exit(1)
	}
	fmt.Fprintf(os.Stdout, "Generated activity lobster trace: %s\n", strings.TrimSpace(*outPath))
}
