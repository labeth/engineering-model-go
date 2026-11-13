// ENGMODEL-OWNER-UNIT: FU-LOBSTER-EXPORTER
package main

import (
	"errors"
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/export/exportgate"
)

// TRLC-LINKS: REQ-EMG-006
// ENGMODEL-LINKS: IF-CLI-ENGLOBSTER, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-TRLC-EXPORTER
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run validates the model, then exports the activity trace of the tests: a
// model with errors (a missing lifecycle among them) is refused with its
// diagnostics and the fix of each required-switch error, and nothing is
// written.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGLOBSTER, FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("englobster", flag.ContinueOnError)
	flags.SetOutput(stderr)
	modelPath := flags.String("model", "", "path to engmod.yml manifest of the model the tests trace to")
	testsDir := flags.String("tests-dir", "", "path to tests directory")
	reqPackage := flags.String("requirements-package", "Requirements", "TRLC requirements package name")
	activityNamespace := flags.String("activity-namespace", "tests", "activity tag namespace")
	outPath := flags.String("out", "", "output .lobster JSON file")
	if err := flags.Parse(args); err != nil {
		return 2
	}

	if anyBlank(*modelPath, *testsDir, *outPath) {
		fmt.Fprintln(stderr, "usage: englobster --model <engmod.yml> --tests-dir <dir> --out <activity.lobster> [--requirements-package <pkg>] [--activity-namespace <ns>]")
		return 2
	}

	if code := gateExit(*modelPath, stderr); code != 0 {
		return code
	}

	res, err := engmodel.GenerateLobsterActivityTraceFromDir(strings.TrimSpace(*testsDir), engmodel.LobsterActivityExportOptions{
		RequirementsPackage: strings.TrimSpace(*reqPackage),
		ActivityNamespace:   strings.TrimSpace(*activityNamespace),
	})
	if errors.Is(err, engmodel.ErrNoLobsterActivities) {
		return skipInapplicable(strings.TrimSpace(*outPath), err, stderr)
	}
	if err != nil {
		fmt.Fprintln(stderr, "error:", err)
		return 1
	}

	if err := os.WriteFile(strings.TrimSpace(*outPath), []byte(res.JSON), 0o644); err != nil {
		fmt.Fprintln(stderr, "error writing output:", err)
		return 1
	}
	fmt.Fprintf(stdout, "Generated activity lobster trace: %s\n", strings.TrimSpace(*outPath))
	return 0
}

// anyBlank reports whether any of the values is empty after trimming.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGLOBSTER, FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func anyBlank(values ...string) bool {
	for _, v := range values {
		if strings.TrimSpace(v) == "" {
			return true
		}
	}
	return false
}

// gateExit runs the validation gate and returns 0 when the model passes, 1
// after printing why it does not.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGLOBSTER, FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func gateExit(modelPath string, stderr io.Writer) int {
	_, ok, err := exportgate.ValidationGate(modelPath, stderr)
	if err != nil {
		fmt.Fprintln(stderr, "error loading model:", err)
		return 1
	}
	if !ok {
		fmt.Fprintln(stderr, "error: model validation failed; no LOBSTER activity output written")
		return 1
	}
	return 0
}

// skipInapplicable removes a stale output when the tests have no activities.
//
// TRLC-LINKS: REQ-EMG-006, REQ-EMG-168
// ENGMODEL-LINKS: IF-CLI-ENGLOBSTER, FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func skipInapplicable(outPath string, cause error, stderr io.Writer) int {
	if removeErr := os.Remove(outPath); removeErr != nil && !os.IsNotExist(removeErr) {
		fmt.Fprintln(stderr, "error removing inapplicable output:", removeErr)
		return 1
	}
	fmt.Fprintln(stderr, "skip LOBSTER activity output:", cause)
	return 0
}
