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
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGSPEC
func main() {
	os.Exit(run(os.Args[1:], os.Stdout, os.Stderr))
}

// run parses the flags, renders the delta and writes it to stdout or --out.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: FU-MODEL-CHANGE, IF-CLI-ENGSPEC
func run(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("engspec", flag.ContinueOnError)
	flags.SetOutput(io.Discard)
	modelPath := flags.String("model", "engmod.yml", "")
	ids := flags.String("ids", "", "")
	prefix := flags.String("prefix", "", "")
	capability := flags.String("capability", "", "")
	modified := flags.Bool("modified", false, "")
	outPath := flags.String("out", "", "")
	if err := flags.Parse(args); err != nil || flags.NArg() > 0 || (strings.TrimSpace(*ids) == "" && strings.TrimSpace(*prefix) == "") {
		if err != nil {
			fmt.Fprintln(stderr, "engspec:", err)
		}
		fmt.Fprint(stderr, usage)
		return 2
	}
	options := engmodel.SpecDeltaOptions{Prefix: *prefix, Capability: *capability}
	for _, id := range strings.Split(*ids, ",") {
		if id = strings.TrimSpace(id); id != "" {
			options.IDs = append(options.IDs, id)
		}
	}
	if *modified {
		options.Operation = engmodel.SpecDeltaModified
	}
	result, err := engmodel.RenderSpecDeltaFromModel(*modelPath, options)
	if err != nil {
		fmt.Fprintln(stderr, "engspec:", err)
		return 1
	}
	for _, warning := range result.Warnings {
		fmt.Fprintln(stderr, "engspec: warning:", warning)
	}
	if *outPath == "" {
		fmt.Fprint(stdout, result.Markdown)
		return 0
	}
	if err := os.WriteFile(*outPath, []byte(result.Markdown), 0o644); err != nil {
		fmt.Fprintln(stderr, "engspec:", err)
		return 1
	}
	return 0
}
