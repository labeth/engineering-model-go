package main

import (
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

func main() {
	modelPath := flag.String("model", "", "path to architecture model YAML")
	outPath := flag.String("out", "", "legacy SSP output path; equivalent to --ssp-out")
	sspOut := flag.String("ssp-out", "", "optional output path for OSCAL SSP JSON")
	arOut := flag.String("ar-out", "", "optional output path for OSCAL Assessment Results JSON")
	poamOut := flag.String("poam-out", "", "optional output path for OSCAL POA&M JSON")
	profile := flag.String("profile", "", "optional OSCAL profile href for SSP")
	systemName := flag.String("system-name", "", "optional SSP system-name override")
	systemDesc := flag.String("system-description", "", "optional SSP description override")
	reqPath := flag.String("requirements", "", "optional requirements YAML path for assessment-results generation")
	codeRoot := flag.String("code-root", "", "optional source tree root for verification/code inference in assessment-results")
	apHref := flag.String("ap-href", "", "optional assessment plan href for assessment-results import-ap")
	sspHref := flag.String("ssp-href", "", "optional SSP href for POA&M import-ssp")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: engoscal --model <architecture.yml> [--ssp-out <ssp.json>] [--ar-out <ar.json>] [--poam-out <poam.json>] [--requirements <requirements.yml>] [--code-root <dir>] [--profile <href>] [--system-name <name>] [--system-description <text>] [--ap-href <assessment-plan.json>] [--ssp-href <ssp.json>]")
		os.Exit(2)
	}
	if strings.TrimSpace(*sspOut) == "" && strings.TrimSpace(*outPath) != "" {
		*sspOut = strings.TrimSpace(*outPath)
	}

	emitSSP := strings.TrimSpace(*sspOut) != "" || (strings.TrimSpace(*sspOut) == "" && strings.TrimSpace(*arOut) == "" && strings.TrimSpace(*poamOut) == "")
	emitAR := strings.TrimSpace(*arOut) != ""
	emitPOAM := strings.TrimSpace(*poamOut) != ""

	allDiags := []validate.Diagnostic{}

	if emitSSP {
		res, err := engmodel.GenerateOSCALSSPFromFile(*modelPath, engmodel.OSCALSSPOptions{
			ProfileHref:       strings.TrimSpace(*profile),
			SystemName:        strings.TrimSpace(*systemName),
			SystemDescription: strings.TrimSpace(*systemDesc),
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(res.Diagnostics)
			os.Exit(1)
		}
		allDiags = append(allDiags, res.Diagnostics...)
		if strings.TrimSpace(*sspOut) == "" {
			_, _ = os.Stdout.WriteString(res.JSON)
			_, _ = os.Stdout.WriteString("\n")
		} else {
			if err := os.WriteFile(*sspOut, []byte(res.JSON), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error writing ssp output:", err)
				os.Exit(1)
			}
		}
	}

	if emitAR {
		res, err := engmodel.GenerateOSCALAssessmentResultsFromFile(*modelPath, engmodel.OSCALAROptions{
			AssessmentPlanHref: strings.TrimSpace(*apHref),
			RequirementsPath:   strings.TrimSpace(*reqPath),
			CodeRoot:           strings.TrimSpace(*codeRoot),
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(res.Diagnostics)
			os.Exit(1)
		}
		allDiags = append(allDiags, res.Diagnostics...)
		if err := os.WriteFile(*arOut, []byte(res.JSON), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing assessment-results output:", err)
			os.Exit(1)
		}
	}

	if emitPOAM {
		res, err := engmodel.GenerateOSCALPOAMFromFile(*modelPath, engmodel.OSCALPOAMOptions{SSPHref: strings.TrimSpace(*sspHref)})
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(res.Diagnostics)
			os.Exit(1)
		}
		allDiags = append(allDiags, res.Diagnostics...)
		if err := os.WriteFile(*poamOut, []byte(res.JSON), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing poam output:", err)
			os.Exit(1)
		}
	}

	allDiags = validate.SortDiagnostics(allDiags)
	printDiagnostics(allDiags)
	if validate.HasErrors(allDiags) {
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
