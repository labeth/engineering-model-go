// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
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
	e := d.Entities
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
	switch {
	case d.Trace != nil:
		t := d.Trace
		fmt.Fprintf(w, "  trace: %d gained, %d lost, %d tests gained, %d new orphans, %d resolved orphans, %d status changes\n",
			len(t.Gained), len(t.Lost), len(t.TestsGained), len(t.NewOrphans), len(t.ResolvedOrphans), len(t.StatusChanged))
		for _, id := range t.NewOrphans {
			fmt.Fprintf(w, "  orphan   %s\n", id)
		}
	case d.TraceError != "":
		fmt.Fprintf(w, "  trace: not computed: %s\n", d.TraceError)
	}
}

// runModelDiff prints the model diff (text or JSON); a git, IO or model error exits 2.
// TRLC-LINKS: REQ-EMG-190, REQ-EMG-191
// ENGMODEL-LINKS: EVT-CONTRACT-DIFF-REQUESTED
func runModelDiff(repo, fromRef, toRef string, asJSON, noTrace bool, w io.Writer) (int, error) {
	d, err := engmodel.DiffModels(repo, fromRef, toRef, engmodel.ModelDiffOptions{Trace: !noTrace})
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
