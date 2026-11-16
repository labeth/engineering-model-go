// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package main

import (
	"flag"
	"fmt"
	"io"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
)

const usage = `usage: engspec --model <engmod.yml> (--ids REQ-X-001,REQ-X-002 | --prefix REQ-X-) [--capability name] [--modified] [--out file]

Renders an OpenSpec spec delta (## ADDED Requirements, or MODIFIED with
--modified) from model requirements: per requirement "### Requirement:
<title or ID>", the line "<ID>: <model text>" with shall/must as SHALL/MUST,
and one scenario from verificationCriteria. Paste it into
openspec/changes/<change>/specs/<capability>/spec.md; the REQ line starts with
the model text, so the engineering-flow consistency check accepts it.
Warnings (no SHALL, no criteria) go to stderr. Exit 0 ok, 1 error, 2 usage.
`

// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGSPEC, FLOW-SPEC-DELTA, EVT-SPEC-DELTA-REQUESTED
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses the flags, renders the delta and writes it to stdout or --out.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGSPEC
func run(args []string, stdout, stderr io.Writer) int {
	modelPath, options, outPath, code, ok := parseEngspecArgs(args, stderr)
	if !ok {
		return code
	}
	result, err := engmodel.RenderSpecDeltaFromModel(modelPath, options)
	if err != nil {
		fmt.Fprintln(stderr, "engspec:", err)
		return 1
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(stderr, "engspec: warning:", warning)
	}
	if outPath == "" {
		fmt.Fprint(stdout, result.Markdown)
		return 0
	}
	if err := os.WriteFile(outPath, []byte(result.Markdown), 0o644); err != nil {
		fmt.Fprintln(stderr, "engspec:", err)
		return 1
	}
	return 0
}

// engspecSelectionMissing reports whether neither --ids nor --prefix selects requirements.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGSPEC
func engspecSelectionMissing(ids, prefix string) bool {
	return strings.TrimSpace(ids) == "" && strings.TrimSpace(prefix) == ""
}

// parseEngspecArgs reads the flags into the model path, the delta options and the --out path. ok is false with the
// exit code 2 when the command line is unusable.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGSPEC
func parseEngspecArgs(args []string, stderr io.Writer) (modelPath string, options engmodel.SpecDeltaOptions, outPath string, code int, ok bool) {
	flags := flag.NewFlagSet("engspec", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	model := flags.String("model", "engmod.yml", "")
	ids := flags.String("ids", "", "")
	prefix := flags.String("prefix", "", "")
	capability := flags.String("capability", "", "")
	modified := flags.Bool("modified", false, "")
	out := flags.String("out", "", "")
	err := flags.Parse(args)
	if err != nil || flags.NArg() > 0 || engspecSelectionMissing(*ids, *prefix) {
		if err != nil {
			fmt.Fprintln(stderr, "engspec:", err)
		}
		fmt.Fprint(stderr, usage)
		return "", options, "", 2, false
	}
	options = engmodel.SpecDeltaOptions{Prefix: *prefix, Capability: *capability, IDs: splitEngspecIDs(*ids)}
	if *modified {
		options.Operation = engmodel.SpecDeltaModified
	}
	return *model, options, *out, 0, true
}

// splitEngspecIDs splits the comma separated --ids value, dropping blanks.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGSPEC
func splitEngspecIDs(ids string) []string {
	var out []string
	for _, id := range strings.Split(ids, ",") {
		if id = strings.TrimSpace(id); id != "" {
			out = append(out, id)
		}
	}
	return out
}
