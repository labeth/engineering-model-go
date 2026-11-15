// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/validate"
)

func main() {
	os.Exit(runEngchange(os.Args[1:]))
}

// engchangeRequest is the parsed command line of engchange.
type engchangeRequest struct {
	operation string
	root      string
	delta     string
}

// parseEngchangeArgs reads the operation and flags. It returns the exit code 2 when the command line is unusable.
// TRLC-LINKS: REQ-EMG-031
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGCHANGE
func parseEngchangeArgs(args []string) (engchangeRequest, int, bool) {
	if len(args) < 1 {
		usage()
		return engchangeRequest{}, 2, false
	}
	operation := strings.TrimSpace(args[0])
	flags := flag.NewFlagSet("engchange "+operation, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "model root containing engmod.yml and requirements.yml")
	delta := flags.String("delta", "", "requirements delta YAML")
	if err := flags.Parse(args[1:]); err != nil {
		return engchangeRequest{}, 2, false
	}
	if operation != "validate" && operation != "diff" && operation != "apply" {
		usage()
		return engchangeRequest{}, 2, false
	}
	if strings.TrimSpace(*delta) == "" {
		fmt.Fprintln(os.Stderr, "--delta is required")
		return engchangeRequest{}, 2, false
	}
	return engchangeRequest{operation: operation, root: *root, delta: *delta}, 0, true
}

// runEngchange runs one engchange operation and returns the exit code.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-CLI-ORCHESTRATION, IF-CLI-ENGCHANGE, DO-REQUIREMENTS-DELTA, DEP-LOCAL-WORKSPACE, FLOW-ARTIFACT-EXPORT
// ENGMODEL-LINKS: EVT-REQUIREMENTS-DELTA-RECEIVED, EVT-REQUIREMENTS-DELTA-APPLY-REQUESTED, TR-DELTA-REJECT
func runEngchange(args []string) int {
	request, code, ok := parseEngchangeArgs(args)
	if !ok {
		return code
	}
	var (
		result engmodel.RequirementsDeltaResult
		err    error
	)
	if request.operation == "apply" {
		result, err = engmodel.ApplyRequirementsDelta(request.root, request.delta)
	} else {
		result, err = engmodel.PlanRequirementsDelta(request.root, request.delta)
	}
	printDiagnostics(result.Diagnostics)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	if validate.HasErrors(result.Diagnostics) {
		return 1
	}
	printEngchangeResult(request.operation, result)
	return 0
}

// printEngchangeResult prints the outcome of a successful operation.
// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGCHANGE
func printEngchangeResult(operation string, result engmodel.RequirementsDeltaResult) {
	switch operation {
	case "diff":
		fmt.Print(result.Diff)
	case "validate":
		fmt.Printf("requirements delta is valid (%d added, %d updated, %d removed)\n", len(result.Added), len(result.Updated), len(result.Removed))
	case "apply":
		fmt.Printf("applied requirements delta (%d added, %d updated, %d removed)\n", len(result.Added), len(result.Updated), len(result.Removed))
	}
}

// TRLC-LINKS: REQ-EMG-032
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-VALIDATION-ENGINE, IF-CLI-ENGCHANGE
func printDiagnostics(diagnostics []validate.Diagnostic) {
	for _, diagnostic := range diagnostics {
		validate.WriteDiagnostic(os.Stderr, diagnostic)
	}
}

// TRLC-LINKS: REQ-EMG-031
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGCHANGE
func usage() {
	fmt.Fprintln(os.Stderr, "usage: engchange <validate|diff|apply> --root <model-root> --delta <requirements-delta.yml>")
}
