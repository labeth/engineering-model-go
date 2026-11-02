// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"time"

	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/linkreview"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
)

// now is the clock of review-links (reviewedOn); tests replace it.
var now = time.Now

// LinkReview is one link that needs review (linkreview.Review).
// TRLC-LINKS: REQ-EMG-101
type LinkReview = linkreview.Review

// reviewResolution composes with the lock checked and the module cache
// reused: nothing is written but the module cache.
// TRLC-LINKS: REQ-EMG-101
func reviewResolution() composition.ResolutionOptions {
	return composition.ResolutionOptions{Lock: composition.LockModeCheck, ModuleCache: composition.ModuleCacheReuse}
}

// oneLine collapses whitespace like the canonical serialization.
// TRLC-LINKS: REQ-EMG-101
func oneLine(s string) string { return linkreview.OneLine(s) }

// ReviewLinks returns every link of the model in repo that needs review,
// with the item at the reviewed version resolved from the module cache or
// the registry (never from a workspace checkout, whose content is not that
// version's).
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func ReviewLinks(repo string) ([]LinkReview, error) {
	ctx, err := linkreview.Evaluate(filepath.Join(repo, "engmod.yml"), reviewResolution())
	if err != nil {
		return nil, err
	}
	return linkreview.Reviews(ctx, reviewResolution()), nil
}

// AcceptLink records the review of link id in the model in repo: the
// current fingerprint, the dependency pin, reviewedBy, reviewedOn (today),
// the rationale and status accepted. Only the touched lines of the
// requirement link file change. It refuses an actor the model does not
// define, an unknown link, a broken link and one whose item cannot be
// evaluated.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func AcceptLink(repo, id, by, rationale string) (reqlinks.LinkStatus, error) {
	ctx, err := linkreview.Evaluate(filepath.Join(repo, "engmod.yml"), reviewResolution())
	if err != nil {
		return reqlinks.LinkStatus{}, err
	}
	var data []byte
	if ctx.Bundle.ReqLinksPath != "" {
		if data, err = os.ReadFile(ctx.Bundle.ReqLinksPath); err != nil {
			return reqlinks.LinkStatus{}, err
		}
	}
	a, err := linkreview.Accept(ctx, data, id, by, rationale, now().Format("2006-01-02"))
	if err != nil {
		return a.Status, err
	}
	return a.Status, writeFileAtomic(a.Path, a.After)
}

// printReviews prints each review: state, parent, fingerprints and the item
// before and after, realizer lines that changed marked - and +.
// TRLC-LINKS: REQ-EMG-101
func printReviews(w io.Writer, reviews []LinkReview) {
	if len(reviews) == 0 {
		fmt.Fprintln(w, "engws: no requirement links need review")
		return
	}
	for _, r := range reviews {
		state := r.State
		if r.State == string(reqlinks.StateVersionMismatch) && r.Content != "" && r.Content != string(reqlinks.StateUnchanged) {
			state = r.Content + ", " + r.State
		}
		fmt.Fprintf(w, "%s %s: %s (pinned %s, dependency %s)\n", r.ID, r.Child, state, r.Pinned, r.Version)
		if r.Parent != "" {
			fmt.Fprintf(w, "  parent %s: %s\n", r.Parent, r.ParentText)
		}
		var before, after []string
		if r.Before != nil {
			fmt.Fprintf(w, "  before (%s): %s\n", r.Pinned, r.Before.Fingerprint)
			before = r.Before.Lines()
		} else {
			fmt.Fprintf(w, "  %s\n", r.BeforeNote)
		}
		if r.After != nil {
			fmt.Fprintf(w, "  after (%s): %s\n", r.Version, r.After.Fingerprint)
			after = r.After.Lines()
		} else {
			detail := r.Detail
			if detail == "" {
				detail = "the item no longer exists"
			}
			fmt.Fprintf(w, "  after (%s): %s\n", r.Version, detail)
		}
		for _, line := range linkreview.DiffLines(before, after, r.Before != nil) {
			fmt.Fprintln(w, "  "+line)
		}
	}
	fmt.Fprintln(w, "engws: review each link, then engws review-links --accept LNK-... --by ACT-... --rationale TEXT")
}

// runReviewLinks lists the links that need review (exit 1 when any) or,
// with accept, records one review.
// TRLC-LINKS: REQ-EMG-101
func runReviewLinks(repo string, asJSON bool, accept, by, rationale string, w io.Writer) (int, error) {
	if accept != "" {
		s, err := AcceptLink(repo, accept, by, rationale)
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(w, "engws: accepted %s %s at %s (%s) by %s on %s\n", s.ID, s.Child, s.Version, s.Current, by, now().Format("2006-01-02"))
		return 0, nil
	}
	reviews, err := ReviewLinks(repo)
	if err != nil {
		return 1, err
	}
	if asJSON {
		enc := json.NewEncoder(w)
		enc.SetIndent("", "  ")
		if err := enc.Encode(reviews); err != nil {
			return 1, err
		}
	} else {
		printReviews(w, reviews)
	}
	if len(reviews) > 0 {
		return 1, nil
	}
	return 0, nil
}
