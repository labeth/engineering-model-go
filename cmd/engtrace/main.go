// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/cliflags"
	"github.com/labeth/engineering-model-go/internal/export/exportgate"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-030, REQ-EMG-063, REQ-EMG-064, REQ-EMG-134, REQ-EMG-168
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

	// The matrix builder reports trace findings only; the validation gate
	// refuses a model with errors (a missing lifecycle among them) first.
	exitOnFailure(gateModel(*modelPath))

	matrix, diags, err := engmodel.BuildTraceMatrixFromFiles(*modelPath, *requirementsPath, *codeRoot)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}

	data, code := encodeMatrix(matrix, *format)
	exitOnCode(code)
	exitOnCode(emitMatrix(data, matrix, strings.TrimSpace(*out)))

	for _, d := range diags {
		validate.WriteDiagnostic(os.Stderr, d)
	}
	if validate.HasErrors(diags) {
		os.Exit(1)
	}
}

// exitOnFailure exits with status 1 when failure is non-nil.
//
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-063, REQ-EMG-064, REQ-EMG-134, REQ-EMG-168
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION
func exitOnFailure(failure error) {
	if failure != nil {
		os.Exit(1)
	}
}

// exitOnCode exits with code when it is not 0.
//
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-063, REQ-EMG-064, REQ-EMG-134, REQ-EMG-168
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION
func exitOnCode(code int) {
	if code != 0 {
		os.Exit(code)
	}
}

// gateModel runs the validation gate; the returned error is non-nil after the
// message was printed.
//
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-063, REQ-EMG-064, REQ-EMG-134, REQ-EMG-168
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION
func gateModel(modelPath string) error {
	_, ok, err := exportgate.ValidationGate(modelPath, os.Stderr)
	if err != nil {
		fmt.Fprintln(os.Stderr, "error loading model:", err)
		return err
	}
	if !ok {
		fmt.Fprintln(os.Stderr, "error: model validation failed; no traceability matrix written")
		return errors.New("model validation failed")
	}
	return nil
}

// encodeMatrix renders the matrix in format and returns the exit code to end
// with (0 when the matrix was encoded).
//
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-063, REQ-EMG-064, REQ-EMG-134, REQ-EMG-168
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION
func encodeMatrix(matrix engmodel.TraceMatrix, format string) ([]byte, int) {
	switch strings.ToLower(strings.TrimSpace(format)) {
	case "csv":
		return matrix.CSV(), 0
	case "json", "":
		b, err := json.MarshalIndent(matrix, "", "  ")
		if err != nil {
			fmt.Fprintln(os.Stderr, "error encoding matrix:", err)
			return nil, 1
		}
		return append(b, '\n'), 0
	}
	fmt.Fprintln(os.Stderr, "unknown format:", format)
	return nil, 2
}

// emitMatrix writes the encoded matrix to out, or to stdout when out is empty,
// and returns the exit code to end with (0 on success).
//
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-063, REQ-EMG-064, REQ-EMG-134, REQ-EMG-168
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-CLI-ORCHESTRATION
func emitMatrix(data []byte, matrix engmodel.TraceMatrix, out string) int {
	if out == "" {
		os.Stdout.Write(data)
		return 0
	}
	if err := os.WriteFile(out, data, 0o644); err != nil {
		fmt.Fprintln(os.Stderr, "error writing output:", err)
		return 1
	}
	fmt.Fprintf(os.Stdout, "Wrote traceability matrix: %s (%d requirements, %d implemented, %d verified, %d delegated, %d orphan, %d dangling)\n",
		out, matrix.Summary.Requirements, matrix.Summary.Implemented, matrix.Summary.Verified, matrix.Summary.Delegated, matrix.Summary.Orphan, matrix.Summary.DanglingLinks)
	return 0
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
