// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/cliflags"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-030, REQ-EMG-063, REQ-EMG-064, REQ-EMG-134
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION
func main() {
	modelPath := flag.String("model", "", "path to engmod.yml manifest")
	requirementsPath := flag.String("requirements", "", "path to requirements YAML")
	codeRoot := flag.String("code-root", "", "code root to scan for trace links")
	out := flag.String("out", "", "output file (default stdout)")
	format := flag.String("format", "json", "output format: json|csv")
	tierGap := flag.String("tier-gap", "", "print, as JSON, what the target tier (poc|prototype|mvp|product|operational) requires that the model does not meet, instead of the matrix")
	modelEnv := cliflags.Register(nil)
	flag.Parse()
	modelEnv.MustApply()

	if strings.TrimSpace(*modelPath) == "" || strings.TrimSpace(*requirementsPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: engtrace --model <engmod.yml> --requirements <model/requirements.yml> [--code-root <dir>] [--format json|csv] [--out <file>] [--tier-gap <tier>]")
		os.Exit(2)
	}
	if target := strings.TrimSpace(*tierGap); target != "" {
		os.Exit(printTierGap(*modelPath, *requirementsPath, *codeRoot, target, *out))
	}

	matrix, diags, err := engmodel.BuildTraceMatrixFromFiles(*modelPath, *requirementsPath, *codeRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	var data []byte
	switch strings.ToLower(strings.TrimSpace(*format)) {
	case "csv":
		data = matrix.CSV()
	case "json", "":
		b, merr := json.MarshalIndent(matrix, "", "  ")
		if merr != nil {
			fmt.Fprintln(os.Stderr, "error encoding matrix:", merr)
			os.Exit(1)
		}
		data = append(b, '\n')
	default:
		fmt.Fprintln(os.Stderr, "unknown format:", *format)
		os.Exit(2)
	}

	if o := strings.TrimSpace(*out); o != "" {
		if werr := os.WriteFile(o, data, 0o644); werr != nil {
			fmt.Fprintln(os.Stderr, "error writing output:", werr)
			os.Exit(1)
		}
		fmt.Fprintf(os.Stdout, "Wrote traceability matrix: %s (%d requirements, %d implemented, %d verified, %d delegated, %d orphan, %d dangling)\n",
			o, matrix.Summary.Requirements, matrix.Summary.Implemented, matrix.Summary.Verified, matrix.Summary.Delegated, matrix.Summary.Orphan, matrix.Summary.DanglingLinks)
	} else {
		os.Stdout.Write(data)
	}

	for _, d := range diags {
		fmt.Fprintf(os.Stderr, "%s [%s] %s\n", d.Code, d.Severity, d.Message)
	}
	if validate.HasErrors(diags) {
		os.Exit(1)
	}
}

// printTierGap writes the tier gap report as JSON to out (default stdout);
// it exits 0, 1 on an error and 2 for an unknown tier.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CLI-ORCHESTRATION
func printTierGap(modelPath, requirementsPath, codeRoot, target, out string) int {
	if engmodel.TierRank(target) == 0 {
		fmt.Fprintf(os.Stderr, "unknown tier %q: want poc, prototype, mvp, product or operational\n", target)
		return 2
	}
	report, err := engmodel.TierGapFromFiles(modelPath, requirementsPath, codeRoot, target)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		return 1
	}
	data, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		fmt.Fprintln(os.Stderr, "error encoding tier gap:", err)
		return 1
	}
	data = append(data, '\n')
	if o := strings.TrimSpace(out); o != "" {
		if err := os.WriteFile(o, data, 0o644); err != nil {
			fmt.Fprintln(os.Stderr, "error writing output:", err)
			return 1
		}
		return 0
	}
	_, _ = os.Stdout.Write(data)
	return 0
}
