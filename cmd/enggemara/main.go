// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
// Command enggemara renders the engineering model into OpenSSF Gemara documents
// (L1 Vector/Principle/Guidance, L2 Capability/Threat/Control, L3 Risk/Policy,
// L5 Evaluation Log, L6 Enforcement Log, L7 Audit Log) using the official
// go-gemara SDK types, with an optional OSCAL catalog and assessment-results bridge.
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/cliflags"
)

// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
func main() {
	args := parseArgs()
	if strings.TrimSpace(args.modelPath) == "" || strings.TrimSpace(args.outDir) == "" {
		fmt.Fprintln(os.Stderr, "usage: enggemara --model <engmod.yml> --out-dir <dir> [--requirements <model/requirements.yml>] [--code-root <dir>] [--author <name>] [--version <v>]")
		os.Exit(2)
	}
	if err := run(args); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

// gemaraArgs holds the parsed enggemara command line.
type gemaraArgs struct {
	modelPath       string
	requirements    string
	codeRoot        string
	outDir          string
	author          string
	authorID        string
	version         string
	date            string
	oscalCatalogOut string
	oscalAROut      string
	oscalAPHref     string
}

// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
func parseArgs() *gemaraArgs {
	a := &gemaraArgs{}
	flag.StringVar(&a.modelPath, "model", "", "path to engmod.yml manifest")
	flag.StringVar(&a.requirements, "requirements", "", "path to requirements YAML (for the evaluation log)")
	flag.StringVar(&a.codeRoot, "code-root", "", "code root for inferred verification (for the evaluation log)")
	flag.StringVar(&a.outDir, "out-dir", "", "output directory for Gemara YAML documents")
	flag.StringVar(&a.author, "author", "", "metadata.author.name")
	flag.StringVar(&a.authorID, "author-id", "", "metadata.author.id")
	flag.StringVar(&a.version, "version", "", "metadata.version")
	flag.StringVar(&a.date, "date", "", "metadata.date (ISO 8601)")
	flag.StringVar(&a.oscalCatalogOut, "oscal-catalog-out", "", "also emit an OSCAL Catalog (JSON) derived from the Gemara control catalog")
	flag.StringVar(&a.oscalAROut, "oscal-ar-out", "", "also emit OSCAL Assessment Results (JSON) derived from the Gemara evaluation log")
	flag.StringVar(&a.oscalAPHref, "oscal-ap-href", "", "assessment-plan href imported by the optional OSCAL Assessment Results")
	modelEnv := cliflags.Register(nil)
	flag.Parse()
	modelEnv.MustApply()
	return a
}

// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
func (a *gemaraArgs) exportOptions() engmodel.GemaraExportOptions {
	return engmodel.GemaraExportOptions{
		AuthorID:           strings.TrimSpace(a.authorID),
		AuthorName:         strings.TrimSpace(a.author),
		Version:            strings.TrimSpace(a.version),
		Date:               strings.TrimSpace(a.date),
		AssessmentPlanHref: strings.TrimSpace(a.oscalAPHref),
	}
}

// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
func run(a *gemaraArgs) error {
	opts := a.exportOptions()
	res, err := engmodel.GenerateGemaraFromFile(a.modelPath, opts)
	if err != nil {
		return fmt.Errorf("error: %w", err)
	}

	// Evaluation log is optional and requires requirements + code root.
	evalYAML, err := evaluationLog(a, opts)
	if err != nil {
		return err
	}
	if err := os.MkdirAll(a.outDir, 0o755); err != nil {
		return fmt.Errorf("error creating out-dir: %w", err)
	}
	if err := writeArtifacts(a.outDir, artifactFiles(res.YAML, evalYAML)); err != nil {
		return err
	}
	return writeOSCALOutputs(a, opts)
}

// artifactFiles names every produced artifact (catalogs, logs, mapping
// document, lexicon) by its file name.
//
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
func artifactFiles(documents map[string]string, evalYAML string) map[string]string {
	files := map[string]string{}
	for name, content := range documents {
		files[name+".yaml"] = content
	}
	if evalYAML != "" {
		files["evaluation-log.yaml"] = evalYAML
	}
	return files
}

// writeArtifacts writes every file into dir.
//
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
func writeArtifacts(dir string, files map[string]string) error {
	for name, content := range files {
		if err := writeArtifact(filepath.Join(dir, name), content); err != nil {
			return err
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
func evaluationLog(a *gemaraArgs, opts engmodel.GemaraExportOptions) (string, error) {
	if strings.TrimSpace(a.requirements) == "" {
		return "", nil
	}
	evalRes, err := engmodel.GenerateGemaraEvaluationLogFromFiles(a.modelPath, a.requirements, strings.TrimSpace(a.codeRoot), opts)
	if err != nil {
		return "", fmt.Errorf("error generating evaluation log: %w", err)
	}
	return evalRes.YAML, nil
}

// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
func writeArtifact(path, content string) error {
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		return fmt.Errorf("error writing %s : %w", path, err)
	}
	fmt.Fprintf(os.Stdout, "wrote %s\n", path)
	return nil
}

// TRLC-LINKS: REQ-EMG-015, REQ-EMG-063, REQ-EMG-064
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER, IF-CLI-ENGGEMARA, FLOW-ARTIFACT-EXPORT
// writeOSCALOutputs writes the optional Gemara -> OSCAL bridge outputs.
func writeOSCALOutputs(a *gemaraArgs, opts engmodel.GemaraExportOptions) error {
	if out := strings.TrimSpace(a.oscalCatalogOut); out != "" {
		js, err := engmodel.GenerateGemaraOSCALCatalogFromFile(a.modelPath, opts)
		if err != nil {
			return fmt.Errorf("error generating OSCAL catalog: %w", err)
		}
		if err := writeArtifact(out, js); err != nil {
			return err
		}
	}
	out := strings.TrimSpace(a.oscalAROut)
	if out == "" {
		return nil
	}
	js, err := engmodel.GenerateGemaraOSCALAssessmentResultsFromFiles(a.modelPath, strings.TrimSpace(a.requirements), strings.TrimSpace(a.codeRoot), opts)
	if err != nil {
		return fmt.Errorf("error generating OSCAL assessment results: %w", err)
	}
	if js == "" {
		fmt.Fprintln(os.Stderr, "no controls to evaluate; skipping OSCAL assessment results")
		return nil
	}
	return writeArtifact(out, js)
}
