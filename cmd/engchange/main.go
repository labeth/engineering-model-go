// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-031, REQ-EMG-032, REQ-EMG-033
// ENGMODEL-LINKS: FU-MODEL-CHANGE, FU-CLI-ORCHESTRATION, IF-CLI-ENGCHANGE, DO-REQUIREMENTS-DELTA, DEP-LOCAL-WORKSPACE
func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	operation := strings.TrimSpace(os.Args[1])
	flags := flag.NewFlagSet("engchange "+operation, flag.ContinueOnError)
	flags.SetOutput(os.Stderr)
	root := flags.String("root", ".", "model root containing engmod.yml and requirements.yml")
	delta := flags.String("delta", "", "requirements delta YAML")
	if err := flags.Parse(os.Args[2:]); err != nil {
		os.Exit(2)
	}
	if operation != "validate" && operation != "diff" && operation != "apply" {
		usage()
		os.Exit(2)
	}
	if strings.TrimSpace(*delta) == "" {
		fmt.Fprintln(os.Stderr, "--delta is required")
		os.Exit(2)
	}

	var (
		result engmodel.RequirementsDeltaResult
		err    error
	)
	if operation == "apply" {
		result, err = engmodel.ApplyRequirementsDelta(*root, *delta)
	} else {
		result, err = engmodel.PlanRequirementsDelta(*root, *delta)
	}
	printDiagnostics(result.Diagnostics)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
	if validate.HasErrors(result.Diagnostics) {
		os.Exit(1)
	}

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
		fmt.Fprintf(os.Stderr, "%s [%s] %s", diagnostic.Code, diagnostic.Severity, diagnostic.Message)
		if diagnostic.Path != "" {
			fmt.Fprintf(os.Stderr, " (%s)", diagnostic.Path)
		}
		fmt.Fprintln(os.Stderr)
	}
}

// TRLC-LINKS: REQ-EMG-031
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGCHANGE
func usage() {
	fmt.Fprintln(os.Stderr, "usage: engchange <validate|diff|apply> --root <model-root> --delta <requirements-delta.yml>")
}
