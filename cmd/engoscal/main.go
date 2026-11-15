// ENGMODEL-OWNER-UNIT: FU-OSCAL-EXPORTER
package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/cliflags"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func main() {
	args := parseArgs()
	if strings.TrimSpace(args.modelPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: engoscal --model <engmod.yml> [--profile-out <profile.json>] [--ssp-out <ssp.json>] [--ap-out <ap.json>] [--ar-out <ar.json>] [--poam-out <poam.json>] [--requirements <model/requirements.yml>] [--code-root <dir>] [--profile <href>] [--import-profile-href <href>] [--catalog <href>] [--system-name <name>] [--system-description <text>] [--ap-href <assessment-plan.json>] [--ssp-href <ssp.json>]")
		os.Exit(2)
	}

	run := &oscalRun{args: args}
	run.emitAll()

	diags := validate.SortDiagnostics(run.diags)
	printDiagnostics(diags)
	if validate.HasErrors(diags) {
		os.Exit(1)
	}
}

// oscalArgs holds the parsed engoscal command line.
type oscalArgs struct {
	modelPath     string
	profileOut    string
	sspOut        string
	apOut         string
	arOut         string
	poamOut       string
	profile       string
	importProfile string
	systemName    string
	systemDesc    string
	reqPath       string
	codeRoot      string
	apHref        string
	sspHref       string
	catalog       string
	lastModified  string
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func parseArgs() *oscalArgs {
	a := &oscalArgs{}
	flag.StringVar(&a.modelPath, "model", "", "path to engmod.yml manifest")
	flag.StringVar(&a.profileOut, "profile-out", "", "optional output path for aggregate OSCAL Profile JSON")
	flag.StringVar(&a.sspOut, "ssp-out", "", "optional output path for OSCAL SSP JSON")
	flag.StringVar(&a.apOut, "ap-out", "", "optional output path for OSCAL Assessment Plan JSON")
	flag.StringVar(&a.arOut, "ar-out", "", "optional output path for OSCAL Assessment Results JSON")
	flag.StringVar(&a.poamOut, "poam-out", "", "optional output path for OSCAL POA&M JSON")
	flag.StringVar(&a.profile, "profile", "", "optional OSCAL profile href for SSP")
	flag.StringVar(&a.importProfile, "import-profile-href", "", "optional packaged aggregate profile href emitted by the SSP")
	flag.StringVar(&a.systemName, "system-name", "", "optional SSP system-name override")
	flag.StringVar(&a.systemDesc, "system-description", "", "optional SSP description override")
	flag.StringVar(&a.reqPath, "requirements", "", "optional requirements YAML path for assessment-results generation")
	flag.StringVar(&a.codeRoot, "code-root", "", "optional source tree root for verification/code inference in assessment-results")
	flag.StringVar(&a.apHref, "ap-href", "", "optional assessment plan href for assessment-results import-ap")
	flag.StringVar(&a.sspHref, "ssp-href", "", "optional SSP href for POA&M import-ssp")
	flag.StringVar(&a.catalog, "catalog", "", "optional OSCAL catalog href used with --profile for validation and control selection")
	flag.StringVar(&a.lastModified, "last-modified", "", "optional RFC3339 timestamp for reproducible OSCAL metadata")
	modelEnv := cliflags.Register(nil)
	flag.Parse()
	modelEnv.MustApply()
	return a
}

// requested reports whether an output path flag is set.
//
// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func requested(path string) bool {
	return strings.TrimSpace(path) != ""
}

// wantsSSP reports whether the SSP is emitted: explicitly, or by default when
// no output at all was requested.
//
// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (a *oscalArgs) wantsSSP() bool {
	return requested(a.sspOut) || !anyRequested(a.profileOut, a.apOut, a.arOut, a.poamOut)
}

// anyRequested reports whether one of the output path flags is set.
//
// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func anyRequested(paths ...string) bool {
	for _, path := range paths {
		if requested(path) {
			return true
		}
	}
	return false
}

// generated is the JSON document and diagnostics of one OSCAL generator.
type generated struct {
	json  string
	diags []validate.Diagnostic
}

// outputSpec names one output file and the sentinel error that means the model
// has nothing to emit for it.
type outputSpec struct {
	label   string
	path    string
	skipErr error
}

// oscalRun is one engoscal invocation collecting diagnostics.
type oscalRun struct {
	args  *oscalArgs
	diags []validate.Diagnostic
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (r *oscalRun) emitAll() {
	a := r.args
	sspAvailable := !a.wantsSSP()
	if requested(a.profileOut) {
		r.emitFile(outputSpec{"profile", a.profileOut, engmodel.ErrNoOSCALProfiles}, a.generateProfile)
	}
	if a.wantsSSP() {
		sspAvailable = r.emitFile(outputSpec{"ssp", a.sspOut, engmodel.ErrNoOSCALCompliance}, a.generateSSP)
	}
	if requested(a.apOut) {
		r.emitFile(outputSpec{"assessment-plan", a.apOut, engmodel.ErrNoOSCALCompliance}, a.generateAP)
	}
	if requested(a.arOut) {
		r.emitFile(outputSpec{"assessment-results", a.arOut, engmodel.ErrNoOSCALCompliance}, a.generateAR)
	}
	if requested(a.poamOut) {
		r.emitPOAM(sspAvailable)
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (r *oscalRun) emitPOAM(sspAvailable bool) {
	spec := outputSpec{"poam", r.args.poamOut, engmodel.ErrNoPOAMItems}
	if sspAvailable {
		r.emitFile(spec, r.args.generatePOAM)
		return
	}
	removeInapplicable(spec)
	fmt.Fprintln(os.Stderr, "skip poam output: no applicable SSP was generated")
}

// emitFile generates one output and writes it, or removes the stale file when
// the model has nothing to emit. It reports whether a file was written.
//
// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (r *oscalRun) emitFile(spec outputSpec, generate func() (generated, error)) bool {
	res, err := generate()
	if errors.Is(err, spec.skipErr) {
		removeInapplicable(spec)
		fmt.Fprintf(os.Stderr, "skip %s output: %v\n", spec.label, err)
		return false
	}
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		printDiagnostics(res.diags)
		os.Exit(1)
	}
	r.diags = append(r.diags, res.diags...)
	if !requested(spec.path) {
		_, _ = os.Stdout.WriteString(res.json)
		_, _ = os.Stdout.WriteString("\n")
		return false
	}
	if err := os.WriteFile(spec.path, []byte(res.json), 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing "+spec.label+" output:", err)
		os.Exit(1)
	}
	return true
}

// removeInapplicable deletes a stale output file the model no longer produces.
//
// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func removeInapplicable(spec outputSpec) {
	removeErr := os.Remove(strings.TrimSpace(spec.path))
	if removeErr != nil && !os.IsNotExist(removeErr) {
		fmt.Fprintln(os.Stderr, "error removing inapplicable "+spec.label+" output:", removeErr)
		os.Exit(1)
	}
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (a *oscalArgs) generateProfile() (generated, error) {
	doc, err := engmodel.GenerateOSCALProfileFromFile(a.modelPath, engmodel.OSCALProfileOptions{
		LastModified: strings.TrimSpace(a.lastModified),
	})
	return generated{json: doc}, err
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (a *oscalArgs) generateSSP() (generated, error) {
	res, err := engmodel.GenerateOSCALSSPFromFile(a.modelPath, engmodel.OSCALSSPOptions{
		ProfileHref:       strings.TrimSpace(a.profile),
		ImportProfileHref: strings.TrimSpace(a.importProfile),
		CatalogHref:       strings.TrimSpace(a.catalog),
		SystemName:        strings.TrimSpace(a.systemName),
		SystemDescription: strings.TrimSpace(a.systemDesc),
		LastModified:      strings.TrimSpace(a.lastModified),
	})
	return generated{json: res.JSON, diags: res.Diagnostics}, err
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (a *oscalArgs) generateAP() (generated, error) {
	res, err := engmodel.GenerateOSCALAssessmentPlanFromFile(a.modelPath, engmodel.OSCALAPOptions{
		SSPHref:      strings.TrimSpace(a.sspHref),
		ProfileHref:  strings.TrimSpace(a.profile),
		CatalogHref:  strings.TrimSpace(a.catalog),
		LastModified: strings.TrimSpace(a.lastModified),
	})
	return generated{json: res.JSON, diags: res.Diagnostics}, err
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (a *oscalArgs) generateAR() (generated, error) {
	res, err := engmodel.GenerateOSCALAssessmentResultsFromFile(a.modelPath, engmodel.OSCALAROptions{
		AssessmentPlanHref: strings.TrimSpace(a.apHref),
		RequirementsPath:   strings.TrimSpace(a.reqPath),
		CodeRoot:           strings.TrimSpace(a.codeRoot),
		ProfileHref:        strings.TrimSpace(a.profile),
		CatalogHref:        strings.TrimSpace(a.catalog),
		LastModified:       strings.TrimSpace(a.lastModified),
	})
	return generated{json: res.JSON, diags: res.Diagnostics}, err
}

// TRLC-LINKS: REQ-EMG-013, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-CLI-ORCHESTRATION, FLOW-ARTIFACT-EXPORT
func (a *oscalArgs) generatePOAM() (generated, error) {
	res, err := engmodel.GenerateOSCALPOAMFromFile(a.modelPath, engmodel.OSCALPOAMOptions{
		SSPHref:      strings.TrimSpace(a.sspHref),
		LastModified: strings.TrimSpace(a.lastModified),
	})
	return generated{json: res.JSON, diags: res.Diagnostics}, err
}

// TRLC-LINKS: REQ-EMG-013
// ENGMODEL-LINKS: IF-CLI-ENGOSCAL, FU-OSCAL-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-VALIDATION-ENGINE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
func printDiagnostics(diags []validate.Diagnostic) {
	for _, d := range diags {
		validate.WriteDiagnostic(os.Stderr, d)
	}
}
