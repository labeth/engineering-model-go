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
)

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-GEMARA-EXPORTER
func main() {
	modelPath := flag.String("model", "", "path to architecture YAML")
	requirementsPath := flag.String("requirements", "", "path to requirements YAML (for the evaluation log)")
	codeRoot := flag.String("code-root", "", "code root for inferred verification (for the evaluation log)")
	outDir := flag.String("out-dir", "", "output directory for Gemara YAML documents")
	author := flag.String("author", "", "metadata.author.name")
	authorID := flag.String("author-id", "", "metadata.author.id")
	version := flag.String("version", "", "metadata.version")
	date := flag.String("date", "", "metadata.date (ISO 8601)")
	oscalCatalogOut := flag.String("oscal-catalog-out", "", "also emit an OSCAL Catalog (JSON) derived from the Gemara control catalog")
	oscalAROut := flag.String("oscal-ar-out", "", "also emit OSCAL Assessment Results (JSON) derived from the Gemara evaluation log")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" || strings.TrimSpace(*outDir) == "" {
		fmt.Fprintln(os.Stderr, "usage: enggemara --model <architecture.yml> --out-dir <dir> [--requirements <reqs.yml>] [--code-root <dir>] [--author <name>] [--version <v>]")
		os.Exit(2)
	}

	opts := engmodel.GemaraExportOptions{
		AuthorID:   strings.TrimSpace(*authorID),
		AuthorName: strings.TrimSpace(*author),
		Version:    strings.TrimSpace(*version),
		Date:       strings.TrimSpace(*date),
	}

	res, err := engmodel.GenerateGemaraFromFile(*modelPath, opts)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	// Evaluation log is optional and requires requirements + code root.
	evalYAML := ""
	if strings.TrimSpace(*requirementsPath) != "" {
		evalRes, err := engmodel.GenerateGemaraEvaluationLogFromFiles(*modelPath, *requirementsPath, strings.TrimSpace(*codeRoot), opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error generating evaluation log:", err)
			os.Exit(1)
		}
		evalYAML = evalRes.YAML
	}

	if err := os.MkdirAll(*outDir, 0o755); err != nil {
		fmt.Fprintln(os.Stderr, "error creating out-dir:", err)
		os.Exit(1)
	}

	// Write every produced artifact (catalogs, logs, mapping document, lexicon).
	files := map[string]string{}
	for name, content := range res.YAML {
		files[name+".yaml"] = content
	}
	if evalYAML != "" {
		files["evaluation-log.yaml"] = evalYAML
	}

	for name, content := range files {
		path := filepath.Join(*outDir, name)
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing", path, ":", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "wrote %s\n", path)
	}

	// Optional Gemara -> OSCAL bridge outputs.
	if out := strings.TrimSpace(*oscalCatalogOut); out != "" {
		js, err := engmodel.GenerateGemaraOSCALCatalogFromFile(*modelPath, opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error generating OSCAL catalog:", err)
			os.Exit(1)
		}
		if err := os.WriteFile(out, []byte(js), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing", out, ":", err)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "wrote %s\n", out)
	}
	if out := strings.TrimSpace(*oscalAROut); out != "" {
		js, err := engmodel.GenerateGemaraOSCALAssessmentResultsFromFiles(*modelPath, strings.TrimSpace(*requirementsPath), strings.TrimSpace(*codeRoot), opts)
		if err != nil {
			fmt.Fprintln(os.Stderr, "error generating OSCAL assessment results:", err)
			os.Exit(1)
		}
		if js == "" {
			fmt.Fprintln(os.Stderr, "no controls to evaluate; skipping OSCAL assessment results")
		} else if err := os.WriteFile(out, []byte(js), 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing", out, ":", err)
			os.Exit(1)
		} else {
			fmt.Fprintf(os.Stdout, "wrote %s\n", out)
		}
	}
}
