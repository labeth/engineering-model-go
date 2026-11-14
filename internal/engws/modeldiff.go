// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
	"flag"
	"fmt"
	"io"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
)

// printModelDiff prints a summary line per kind of change and one line per entity.
// TRLC-LINKS: REQ-EMG-190
func printModelDiff(w io.Writer, d engmodel.ModelDiff) {
	to := d.To
	if to == "" {
		to = "working tree"
	}
	fmt.Fprintf(w, "model %s..%s\n", d.From, to)
	printEntityDiff(w, d.Entities)
	switch {
	case d.Trace != nil:
		printTraceDelta(w, d.Trace)
	case d.TraceError != "":
		fmt.Fprintf(w, "  trace: not computed: %s\n", d.TraceError)
	}
}

// TRLC-LINKS: REQ-EMG-190
func printEntityDiff(w io.Writer, e engmodel.EntityDiff) {
	fmt.Fprintf(w, "  entities: %d added, %d removed, %d changed; references: %d added, %d removed\n",
		len(e.Added), len(e.Removed), len(e.Changed), len(e.ReferencesAdded), len(e.ReferencesRemoved))
	for _, c := range e.Added {
		fmt.Fprintf(w, "  added    %s %s: %s\n", c.Kind, c.ID, c.Title)
	}
	for _, c := range e.Changed {
		fmt.Fprintf(w, "  changed  %s %s (%s)\n", c.Kind, c.ID, strings.Join(c.Fields, ", "))
	}
	for _, c := range e.Removed {
		fmt.Fprintf(w, "  removed  %s %s: %s\n", c.Kind, c.ID, c.Title)
	}
}

// TRLC-LINKS: REQ-EMG-191
func printTraceDelta(w io.Writer, t *engmodel.TraceDelta) {
	fmt.Fprintf(w, "  trace: %d gained, %d lost, %d tests gained, %d new orphans, %d resolved orphans, %d status changes\n",
		len(t.Gained), len(t.Lost), len(t.TestsGained), len(t.NewOrphans), len(t.ResolvedOrphans), len(t.StatusChanged))
	for _, id := range t.NewOrphans {
		fmt.Fprintf(w, "  orphan   %s\n", id)
	}
}

// runModelDiff prints the model diff (text or JSON); a git, IO or model error exits 2.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-191
// ENGMODEL-LINKS: EVT-CONTRACT-DIFF-REQUESTED
func runModelDiff(req engmodel.ModelDiffRequest, asJSON bool, w io.Writer) (int, error) {
	d, err := engmodel.DiffModels(req)
	if err != nil {
		return 2, err
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		return 0, enc.Encode(d)
	}
	printModelDiff(w, d)
	return 0, nil
}

// runModelDiffCommand is engws model-diff <from-ref> [<to-ref>] [--repo DIR] [--json] [--no-trace]: it parses its own
// flags, so the other commands do not carry them. It returns the exit code: 0, or 2 for a usage, git, IO or model error.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-191
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func runModelDiffCommand(args []string, stdout, stderr io.Writer) int {
	sub := flag.NewFlagSet("engws model-diff", flag.ContinueOnError)
	sub.SetOutput(io.Discard)
	repo := sub.String("repo", ".", "")
	asJSON := sub.Bool("json", false, "")
	noTrace := sub.Bool("no-trace", false, "")
	positional, err := parseInterspersed(sub, args)
	if err != nil {
		fmt.Fprintf(stderr, "engws model-diff: %v\n%s", err, Usage)
		return 2
	}
	if len(positional) < 1 || len(positional) > 2 {
		fmt.Fprintf(stderr, "engws model-diff: expected 1 or 2 arguments, got %d\n%s", len(positional), Usage)
		return 2
	}
	req := engmodel.ModelDiffRequest{Repo: *repo, From: positional[0], Trace: !*noTrace}
	if len(positional) == 2 {
		req.To = positional[1]
	}
	code, err := runModelDiff(req, *asJSON, stdout)
	if err != nil {
		fmt.Fprintf(stderr, "engws model-diff: %v\n", err)
	}
	return code
}
