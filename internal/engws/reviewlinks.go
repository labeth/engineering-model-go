// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/linkreview"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// now is the clock of review-links (reviewedOn); tests replace it.
var now = time.Now

// LinkReview is one link that needs review (linkreview.Review).
// TRLC-LINKS: REQ-EMG-101
type LinkReview = linkreview.Review

// reviewResolution composes with the lock checked and each cached module
// verified against the registry's manifest digest (refetched when it
// differs, reused offline), so a re-staged or moved version is seen; nothing
// is written but the module cache.
// TRLC-LINKS: REQ-EMG-101
func reviewResolution() composition.ResolutionOptions {
	return composition.ResolutionOptions{Lock: composition.LockModeCheck}
}

// oneLine collapses whitespace like the meaning serialization.
// TRLC-LINKS: REQ-EMG-101
func oneLine(s string) string { return linkreview.OneLine(s) }

// ReviewLinks returns every link of the model in repo that needs review
// (pinned to another version than the dependency, broken or claimed), with
// the item at the pinned version and at the dependency pin, both resolved
// with the workspace off from the module cache or the registry (never from
// a workspace checkout, whose content is not that version's).
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func ReviewLinks(repo string) ([]LinkReview, error) {
	ctx, err := linkreview.EvaluateForAccept(filepath.Join(repo, "engmod.yml"), reviewResolution())
	if err != nil {
		return nil, err
	}
	return linkreview.Reviews(ctx, reviewResolution()), nil
}

// AcceptLink records the review of link id in the model in repo:
// pinned.version set to the dependency pin (the published dependency,
// composed with the workspace off, decision D9), reviewedBy, reviewedOn
// (today), the rationale and status accepted, a deprecated fingerprint
// removed. Only the touched lines of the requirement link file change. It
// refuses an actor the model does not define, an unknown link, a broken
// link, one whose item cannot be resolved and a model that does not compose
// with the workspace off. It then evaluates the link again and fails when
// it still does not pass (the file stays written; the error says why).
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ACCEPT, TR-LINK-REACCEPT
func AcceptLink(repo, id, by, rationale string) (reqlinks.LinkStatus, error) {
	manifest := filepath.Join(repo, "engmod.yml")
	ctx, err := linkreview.EvaluateForAccept(manifest, reviewResolution())
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
	if err := writeFileAtomic(a.Path, a.After); err != nil {
		return a.Status, err
	}
	return a.Status, verifyAccepted(manifest, id)
}

// verifyAccepted evaluates link id of manifest again (workspace off) and
// returns an error naming why it does not pass: a state other than
// unchanged, a status other than accepted, or a link.* validation error at
// the link.
// TRLC-LINKS: REQ-EMG-101
func verifyAccepted(manifest, id string) error {
	ctx, err := linkreview.EvaluateForAccept(manifest, reviewResolution())
	if err != nil {
		return fmt.Errorf("accepted %s, but the model no longer evaluates: %w", id, err)
	}
	for _, s := range ctx.Statuses {
		if s.ID != id {
			continue
		}
		if s.State != reqlinks.StateUnchanged || s.Status != "accepted" {
			return fmt.Errorf("accepted %s, but it still does not pass: %s, status %s (pinned %s, dependency %s)", id, s.State, s.Status, s.Pinned, s.Version)
		}
	}
	for i, l := range ctx.Bundle.ReqLinks.ReqLinks {
		if l.ID != id {
			continue
		}
		prefix := fmt.Sprintf("reqLinks[%d]", i)
		for _, d := range validate.Bundle(ctx.Bundle) {
			if d.Severity == validate.SeverityError && strings.HasPrefix(d.Code, "link.") && (d.Path == prefix || strings.HasPrefix(d.Path, prefix+".")) {
				return fmt.Errorf("accepted %s, but it still does not pass: %s %s", id, d.Code, d.Message)
			}
		}
	}
	return nil
}

// printReviews prints one line per link that needs review: ID, child,
// state and review status, the pinned and dependency version and what
// changed in meaning by ID; with show, the parent text and the item before
// and after, changed lines marked - and +.
// TRLC-LINKS: REQ-EMG-101
func printReviews(w io.Writer, reviews []LinkReview, show bool) {
	if len(reviews) == 0 {
		fmt.Fprintln(w, "engws: no requirement links need review")
		return
	}
	for _, r := range reviews {
		fmt.Fprintf(w, "%s %s: %s\n", r.ID, r.Child, reviewSummary(r))
		if !show {
			continue
		}
		if r.Parent != "" {
			fmt.Fprintf(w, "  parent %s: %s\n", r.Parent, r.ParentText)
		}
		var before, after []string
		if r.Before != nil {
			fmt.Fprintf(w, "  before (%s)\n", r.Pinned)
			before = r.Before.Lines()
		} else {
			fmt.Fprintf(w, "  %s\n", r.BeforeNote)
		}
		if r.After != nil {
			fmt.Fprintf(w, "  after (%s)\n", r.Version)
			after = r.After.Lines()
		} else {
			detail := r.Detail
			if detail == "" {
				detail = "the item does not exist"
			}
			fmt.Fprintf(w, "  after (%s): %s\n", r.Version, detail)
		}
		for _, line := range linkreview.DiffLines(before, after, r.Before != nil) {
			fmt.Fprintln(w, "  "+line)
		}
	}
	fmt.Fprintln(w, "engws: review each link (--show prints the text), then engws review-links --accept LNK-... --by ACT-... --rationale TEXT")
}

// reviewSummary is the one-line state of a review: state, status, versions
// and the meaning changes (or why they are unknown).
// TRLC-LINKS: REQ-EMG-101
func reviewSummary(r LinkReview) string {
	parts := []string{r.State}
	if r.Status == "claimed" {
		parts = append(parts, "claimed")
	}
	line := strings.Join(parts, ", ") + fmt.Sprintf(" (pinned %s, dependency %s)", r.Pinned, r.Version)
	switch {
	case r.State == string(reqlinks.StateBroken):
		line += ": the item does not exist at " + r.Version
	case r.After == nil && r.Detail != "":
		line += ": " + r.Detail
	case r.Before == nil && r.After != nil:
		line += ": " + strings.TrimPrefix(r.BeforeNote, "before: ")
	case r.Before != nil && r.After != nil && len(r.Changes) == 0 && r.Pinned != r.Version:
		line += ": same meaning"
	case len(r.Changes) > 0:
		line += ": changed " + strings.Join(r.Changes, ", ")
	}
	return line
}

// runReviewLinks lists the links that need review (exit 1 when any) or,
// with accept, records one review.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: EVT-LINK-REVIEW-REQUESTED
func runReviewLinks(repo string, asJSON, show bool, accept, by, rationale string, w io.Writer) (int, error) {
	if accept != "" {
		s, err := AcceptLink(repo, accept, by, rationale)
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(w, "engws: accepted %s %s at %s by %s on %s\n", s.ID, s.Child, s.Version, by, now().Format("2006-01-02"))
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
		printReviews(w, reviews, show)
	}
	if len(reviews) > 0 {
		return 1, nil
	}
	return 0, nil
}
