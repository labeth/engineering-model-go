// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func main() {
	modelPath := flag.String("model", "", "path to engmod.yml manifest")
	outPath := flag.String("out", "", "legacy SSP output path; equivalent to --ssp-out")
	profileOut := flag.String("profile-out", "", "optional output path for aggregate OSCAL Profile JSON")
	sspOut := flag.String("ssp-out", "", "optional output path for OSCAL SSP JSON")
	apOut := flag.String("ap-out", "", "optional output path for OSCAL Assessment Plan JSON")
	arOut := flag.String("ar-out", "", "optional output path for OSCAL Assessment Results JSON")
	poamOut := flag.String("poam-out", "", "optional output path for OSCAL POA&M JSON")
	profile := flag.String("profile", "", "optional OSCAL profile href for SSP")
	importProfile := flag.String("import-profile-href", "", "optional packaged aggregate profile href emitted by the SSP")
	systemName := flag.String("system-name", "", "optional SSP system-name override")
	systemDesc := flag.String("system-description", "", "optional SSP description override")
	reqPath := flag.String("requirements", "", "optional requirements YAML path for assessment-results generation")
	codeRoot := flag.String("code-root", "", "optional source tree root for verification/code inference in assessment-results")
	apHref := flag.String("ap-href", "", "optional assessment plan href for assessment-results import-ap")
	sspHref := flag.String("ssp-href", "", "optional SSP href for POA&M import-ssp")
	catalog := flag.String("catalog", "", "optional OSCAL catalog href used with --profile for validation and control selection")
	lastModified := flag.String("last-modified", "", "optional RFC3339 timestamp for reproducible OSCAL metadata")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: engoscal --model <engmod.yml> [--profile-out <profile.json>] [--ssp-out <ssp.json>] [--ap-out <ap.json>] [--ar-out <ar.json>] [--poam-out <poam.json>] [--requirements <model/requirements.yml>] [--code-root <dir>] [--profile <href>] [--import-profile-href <href>] [--catalog <href>] [--system-name <name>] [--system-description <text>] [--ap-href <assessment-plan.json>] [--ssp-href <ssp.json>]")
		os.Exit(2)
	}
	if strings.TrimSpace(*sspOut) == "" && strings.TrimSpace(*outPath) != "" {
		*sspOut = strings.TrimSpace(*outPath)
	}

	emitProfile := strings.TrimSpace(*profileOut) != ""
	emitSSP := strings.TrimSpace(*sspOut) != "" || (strings.TrimSpace(*profileOut) == "" && strings.TrimSpace(*sspOut) == "" && strings.TrimSpace(*apOut) == "" && strings.TrimSpace(*arOut) == "" && strings.TrimSpace(*poamOut) == "")
	emitAP := strings.TrimSpace(*apOut) != ""
	emitAR := strings.TrimSpace(*arOut) != ""
	emitPOAM := strings.TrimSpace(*poamOut) != ""

	allDiags := []validate.Diagnostic{}
	sspAvailable := !emitSSP

	if emitProfile {
		doc, err := engmodel.GenerateOSCALProfileFromFile(*modelPath, engmodel.OSCALProfileOptions{
			LastModified: strings.TrimSpace(*lastModified),
		})
		if errors.Is(err, engmodel.ErrNoOSCALProfiles) {
			if removeErr := os.Remove(strings.TrimSpace(*profileOut)); removeErr != nil && !os.IsNotExist(removeErr) {
				fmt.Fprintln(os.Stderr, "error removing inapplicable profile output:", removeErr)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "skip profile output:", err)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			os.Exit(1)
		} else if err := os.WriteFile(*profileOut, []byte(doc), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing profile output:", err)
			os.Exit(1)
		}
	}

	if emitSSP {
		res, err := engmodel.GenerateOSCALSSPFromFile(*modelPath, engmodel.OSCALSSPOptions{
			ProfileHref:       strings.TrimSpace(*profile),
			ImportProfileHref: strings.TrimSpace(*importProfile),
			CatalogHref:       strings.TrimSpace(*catalog),
			SystemName:        strings.TrimSpace(*systemName),
			SystemDescription: strings.TrimSpace(*systemDesc),
			LastModified:      strings.TrimSpace(*lastModified),
		})
		if errors.Is(err, engmodel.ErrNoOSCALCompliance) {
			if removeErr := os.Remove(strings.TrimSpace(*sspOut)); removeErr != nil && !os.IsNotExist(removeErr) {
				fmt.Fprintln(os.Stderr, "error removing inapplicable ssp output:", removeErr)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "skip ssp output:", err)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(res.Diagnostics)
			os.Exit(1)
		} else if strings.TrimSpace(*sspOut) == "" {
			allDiags = append(allDiags, res.Diagnostics...)
			_, _ = os.Stdout.WriteString(res.JSON)
			_, _ = os.Stdout.WriteString("\n")
		} else {
			allDiags = append(allDiags, res.Diagnostics...)
			if err := os.WriteFile(*sspOut, []byte(res.JSON), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error writing ssp output:", err)
				os.Exit(1)
			}
			sspAvailable = true
		}
	}

	if emitAP {
		res, err := engmodel.GenerateOSCALAssessmentPlanFromFile(*modelPath, engmodel.OSCALAPOptions{
			SSPHref:      strings.TrimSpace(*sspHref),
			ProfileHref:  strings.TrimSpace(*profile),
			CatalogHref:  strings.TrimSpace(*catalog),
			LastModified: strings.TrimSpace(*lastModified),
		})
		if errors.Is(err, engmodel.ErrNoOSCALCompliance) {
			if removeErr := os.Remove(strings.TrimSpace(*apOut)); removeErr != nil && !os.IsNotExist(removeErr) {
				fmt.Fprintln(os.Stderr, "error removing inapplicable assessment-plan output:", removeErr)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "skip assessment-plan output:", err)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(res.Diagnostics)
			os.Exit(1)
		} else if err := os.WriteFile(*apOut, []byte(res.JSON), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing assessment-plan output:", err)
			os.Exit(1)
		} else {
			allDiags = append(allDiags, res.Diagnostics...)
		}
	}

	if emitAR {
		res, err := engmodel.GenerateOSCALAssessmentResultsFromFile(*modelPath, engmodel.OSCALAROptions{
			AssessmentPlanHref: strings.TrimSpace(*apHref),
			RequirementsPath:   strings.TrimSpace(*reqPath),
			CodeRoot:           strings.TrimSpace(*codeRoot),
			ProfileHref:        strings.TrimSpace(*profile),
			CatalogHref:        strings.TrimSpace(*catalog),
			LastModified:       strings.TrimSpace(*lastModified),
		})
		if errors.Is(err, engmodel.ErrNoOSCALCompliance) {
			if removeErr := os.Remove(strings.TrimSpace(*arOut)); removeErr != nil && !os.IsNotExist(removeErr) {
				fmt.Fprintln(os.Stderr, "error removing inapplicable assessment-results output:", removeErr)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "skip assessment-results output:", err)
		} else if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(res.Diagnostics)
			os.Exit(1)
		} else if err := os.WriteFile(*arOut, []byte(res.JSON), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing assessment-results output:", err)
			os.Exit(1)
		} else {
			allDiags = append(allDiags, res.Diagnostics...)
		}
	}

	if emitPOAM {
		if !sspAvailable {
			if removeErr := os.Remove(strings.TrimSpace(*poamOut)); removeErr != nil && !os.IsNotExist(removeErr) {
				fmt.Fprintln(os.Stderr, "error removing inapplicable poam output:", removeErr)
				os.Exit(1)
			}
			fmt.Fprintln(os.Stderr, "skip poam output: no applicable SSP was generated")
		} else {
			res, err := engmodel.GenerateOSCALPOAMFromFile(*modelPath, engmodel.OSCALPOAMOptions{
				SSPHref:      strings.TrimSpace(*sspHref),
				LastModified: strings.TrimSpace(*lastModified),
			})
			if errors.Is(err, engmodel.ErrNoPOAMItems) {
				if removeErr := os.Remove(strings.TrimSpace(*poamOut)); removeErr != nil && !os.IsNotExist(removeErr) {
					fmt.Fprintln(os.Stderr, "error removing inapplicable poam output:", removeErr)
					os.Exit(1)
				}
				fmt.Fprintln(os.Stderr, "skip poam output:", err)
			} else if err != nil {
				fmt.Fprintln(os.Stderr, "error:", err)
				printDiagnostics(res.Diagnostics)
				os.Exit(1)
			} else if err := os.WriteFile(*poamOut, []byte(res.JSON), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error writing poam output:", err)
				os.Exit(1)
			} else {
				allDiags = append(allDiags, res.Diagnostics...)
			}
		}
	}

	allDiags = validate.SortDiagnostics(allDiags)
	printDiagnostics(allDiags)
	if validate.HasErrors(allDiags) {
		os.Exit(1)
	}
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func printDiagnostics(diags []validate.Diagnostic) {
	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s [%s] %s", d.Code, d.Severity, d.Message)
		if strings.TrimSpace(d.Path) != "" {
			fmt.Fprintf(os.Stderr, " (%s)", d.Path)
		}
		fmt.Fprintln(os.Stderr)
	}
}
