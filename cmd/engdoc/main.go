// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

type viewFlags []string

func (v *viewFlags) String() string {
	return strings.Join(*v, ",")
}

func (v *viewFlags) Set(value string) error {
	value = strings.TrimSpace(value)
	if value == "" {
		return nil
	}
	*v = append(*v, value)
	return nil
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-002, REQ-EMG-003, REQ-EMG-009, REQ-EMG-012, REQ-EMG-014
func main() {
	modelPath := flag.String("model", "", "path to architecture model YAML")
	reqPath := flag.String("requirements", "", "path to requirements YAML")
	designPath := flag.String("design", "", "path to design mapping YAML")
	codeRoot := flag.String("code-root", "", "optional source tree root for TRACE-* code mapping")
	outPath := flag.String("out", "", "optional output .adoc path; defaults to stdout")
	decisionsOut := flag.String("decisions-out", "", "optional output path for generated architecture decision records .adoc")
	aiJSONOut := flag.String("ai-json-out", "", "optional output path for AI JSON export")
	aiMarkdownOut := flag.String("ai-md-out", "", "optional output path for derived AI markdown export")
	aiEdgesOut := flag.String("ai-edges-out", "", "optional output path for AI edge stream NDJSON export")
	var views viewFlags
	flag.Var(&views, "view", "optional viewpoint ID; repeat to include multiple views")
	flag.Parse()

	if strings.TrimSpace(*modelPath) == "" || strings.TrimSpace(*reqPath) == "" || strings.TrimSpace(*designPath) == "" {
		fmt.Fprintln(os.Stderr, "usage: engdoc --model <architecture.yml> --requirements <requirements.yml> --design <design.yml> [--code-root <dir>] [--view <id> ...] [--out <file>] [--decisions-out <file>] [--ai-json-out <file>] [--ai-md-out <file>] [--ai-edges-out <file>]")
		os.Exit(2)
	}

	useAI := strings.TrimSpace(*aiJSONOut) != "" || strings.TrimSpace(*aiMarkdownOut) != "" || strings.TrimSpace(*aiEdgesOut) != ""
	useAsciiDoc := strings.TrimSpace(*outPath) != "" || strings.TrimSpace(*decisionsOut) != "" || !useAI
	allDiagnostics := []validate.Diagnostic{}

	if useAsciiDoc {
		decisionsDocPath := strings.TrimSpace(*decisionsOut)
		if decisionsDocPath != "" {
			decisionsDocPath = filepath.Base(decisionsDocPath)
		}
		res, err := engmodel.GenerateAsciiDocFromFiles(*modelPath, *reqPath, *designPath, engmodel.AsciiDocOptions{
			ViewIDs:          views,
			CodeRoot:         strings.TrimSpace(*codeRoot),
			DecisionsDocPath: decisionsDocPath,
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(res.Diagnostics)
			os.Exit(1)
		}
		allDiagnostics = append(allDiagnostics, res.Diagnostics...)

		if strings.TrimSpace(*outPath) == "" {
			_, _ = os.Stdout.WriteString(res.Document)
		} else {
			if err := os.WriteFile(*outPath, []byte(res.Document), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error writing output:", err)
				os.Exit(1)
			}
		}
		if strings.TrimSpace(*decisionsOut) != "" {
			if err := os.WriteFile(*decisionsOut, []byte(res.DecisionsDocument), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error writing decisions output:", err)
				os.Exit(1)
			}
		}
	}

	if useAI {
		aiRes, err := engmodel.GenerateAIViewFromFiles(*modelPath, *reqPath, *designPath, engmodel.AIViewOptions{
			ViewIDs:  views,
			CodeRoot: strings.TrimSpace(*codeRoot),
		})
		if err != nil {
			fmt.Fprintln(os.Stderr, "error:", err)
			printDiagnostics(aiRes.Diagnostics)
			os.Exit(1)
		}
		allDiagnostics = append(allDiagnostics, aiRes.Diagnostics...)

		if strings.TrimSpace(*aiJSONOut) != "" {
			if err := os.WriteFile(*aiJSONOut, []byte(aiRes.JSON), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error writing ai json output:", err)
				os.Exit(1)
			}
		}
		if strings.TrimSpace(*aiMarkdownOut) != "" {
			if err := os.WriteFile(*aiMarkdownOut, []byte(aiRes.Markdown), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error writing ai markdown output:", err)
				os.Exit(1)
			}
		}
		if strings.TrimSpace(*aiEdgesOut) != "" {
			if err := os.WriteFile(*aiEdgesOut, []byte(aiRes.EdgesNDJSON), 0o644); err != nil {
				fmt.Fprintln(os.Stderr, "error writing ai edges output:", err)
				os.Exit(1)
			}
		}
	}

	allDiagnostics = validate.SortDiagnostics(allDiagnostics)
	printDiagnostics(allDiagnostics)
	if validate.HasErrors(allDiagnostics) {
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
