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

// AcceptLink records the review of link req.ID in the model in repo:
// pinned.version set to the dependency pin (the published dependency,
// composed with the workspace off, decision D9), reviewedBy, reviewedOn
// (today), the rationale and status accepted, a deprecated fingerprint
// removed. Only the touched lines of the requirement link file change. It
// refuses an actor the model does not define, an unknown link, a broken
// link, one whose item cannot be resolved and a model that does not compose
// with the workspace off. It then evaluates the link again and fails when
// it still does not pass (the file stays written; the error says why).
// The request's Date is set to today.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ACCEPT, TR-LINK-REACCEPT
func AcceptLink(repo string, req linkreview.AcceptRequest) (reqlinks.LinkStatus, error) {
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
	req.Date = now().Format("2006-01-02")
	a, err := linkreview.AcceptReview(ctx, data, req)
	if err != nil {
		return a.Status, err
	}
	if err := writeFileAtomic(a.Path, a.After); err != nil {
		return a.Status, err
	}
	return a.Status, verifyAccepted(manifest, req.ID)
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
	if err := verifyAcceptedStatus(ctx, id); err != nil {
		return err
	}
	return verifyAcceptedValidation(ctx, id)
}

// verifyAcceptedStatus fails when link id is not unchanged and accepted.
// TRLC-LINKS: REQ-EMG-101
func verifyAcceptedStatus(ctx linkreview.Context, id string) error {
	for _, s := range ctx.Statuses {
		if s.ID != id {
			continue
		}
		if s.State != reqlinks.StateUnchanged || s.Status != "accepted" {
			return fmt.Errorf("accepted %s, but it still does not pass: %s, status %s (pinned %s, dependency %s)", id, s.State, s.Status, s.Pinned, s.Version)
		}
	}
	return nil
}

// verifyAcceptedValidation fails on a link.* validation error at link id.
// TRLC-LINKS: REQ-EMG-101
func verifyAcceptedValidation(ctx linkreview.Context, id string) error {
	for i, l := range ctx.Bundle.ReqLinks.ReqLinks {
		if l.ID != id {
			continue
		}
		if d, ok := linkValidationError(ctx, fmt.Sprintf("reqLinks[%d]", i)); ok {
			return fmt.Errorf("accepted %s, but it still does not pass: %s %s", id, d.Code, d.Message)
		}
	}
	return nil
}

// linkValidationError returns the first link.* validation error at prefix.
// TRLC-LINKS: REQ-EMG-101
func linkValidationError(ctx linkreview.Context, prefix string) (validate.Diagnostic, bool) {
	for _, d := range validate.Bundle(ctx.Bundle) {
		if d.Severity == validate.SeverityError && strings.HasPrefix(d.Code, "link.") && (d.Path == prefix || strings.HasPrefix(d.Path, prefix+".")) {
			return d, true
		}
	}
	return validate.Diagnostic{}, false
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
		if show {
			printReviewDetail(w, r)
		}
	}
	fmt.Fprintln(w, "engws: review each link (--show prints the text), then engws review-links --accept LNK-... --by ACT-... --rationale TEXT")
}

// printReviewDetail prints the parent, the item before and after and their
// line diff of one review.
// TRLC-LINKS: REQ-EMG-101
func printReviewDetail(w io.Writer, r LinkReview) {
	if r.Parent != "" {
		fmt.Fprintf(w, "  parent %s: %s\n", r.Parent, r.ParentText)
	}
	before := printReviewBefore(w, r)
	after := printReviewAfter(w, r)
	for _, line := range linkreview.DiffLines(before, after, r.Before != nil) {
		fmt.Fprintln(w, "  "+line)
	}
}

// printReviewBefore prints the before heading and returns its lines.
// TRLC-LINKS: REQ-EMG-101
func printReviewBefore(w io.Writer, r LinkReview) []string {
	if r.Before == nil {
		fmt.Fprintf(w, "  %s\n", r.BeforeNote)
		return nil
	}
	fmt.Fprintf(w, "  before (%s)\n", r.Pinned)
	return r.Before.Lines()
}

// printReviewAfter prints the after heading and returns its lines.
// TRLC-LINKS: REQ-EMG-101
func printReviewAfter(w io.Writer, r LinkReview) []string {
	if r.After != nil {
		fmt.Fprintf(w, "  after (%s)\n", r.Version)
		return r.After.Lines()
	}
	detail := r.Detail
	if detail == "" {
		detail = "the item does not exist"
	}
	fmt.Fprintf(w, "  after (%s): %s\n", r.Version, detail)
	return nil
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
	return line + reviewSuffix(r)
}

// reviewSuffixRules are tried in order; the first that applies gives the
// text after the versions.
var reviewSuffixRules = []func(LinkReview) (string, bool){
	func(r LinkReview) (string, bool) {
		return ": the item does not exist at " + r.Version, r.State == string(reqlinks.StateBroken)
	},
	func(r LinkReview) (string, bool) { return ": " + r.Detail, r.After == nil && r.Detail != "" },
	func(r LinkReview) (string, bool) {
		return ": " + strings.TrimPrefix(r.BeforeNote, "before: "), r.Before == nil && r.After != nil
	},
	func(r LinkReview) (string, bool) { return ": same meaning", reviewSameMeaning(r) },
	func(r LinkReview) (string, bool) {
		return ": changed " + strings.Join(r.Changes, ", "), len(r.Changes) > 0
	},
}

// reviewSameMeaning reports a changed pin whose item meaning is the same.
// TRLC-LINKS: REQ-EMG-101
func reviewSameMeaning(r LinkReview) bool {
	known := r.Before != nil && r.After != nil
	return known && len(r.Changes) == 0 && r.Pinned != r.Version
}

// reviewSuffix returns the text of the first applicable rule, or "".
// TRLC-LINKS: REQ-EMG-101
func reviewSuffix(r LinkReview) string {
	for _, rule := range reviewSuffixRules {
		if text, ok := rule(r); ok {
			return text
		}
	}
	return ""
}

// reviewLinksOptions are the flags of engws review-links: the repository,
// the output form and, when Accept.ID is set, the review to record.
type reviewLinksOptions struct {
	Repo   string
	JSON   bool
	Show   bool
	Accept linkreview.AcceptRequest
}

// runReviewLinks lists the links that need review (exit 1 when any) or,
// with an accept ID, records one review.
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: EVT-LINK-REVIEW-REQUESTED
func runReviewLinks(opts reviewLinksOptions, w io.Writer) (int, error) {
	if opts.Accept.ID != "" {
		s, err := AcceptLink(opts.Repo, opts.Accept)
		if err != nil {
			return 1, err
		}
		fmt.Fprintf(w, "engws: accepted %s %s at %s by %s on %s\n", s.ID, s.Child, s.Version, opts.Accept.By, now().Format("2006-01-02"))
		return 0, nil
	}
	reviews, err := ReviewLinks(opts.Repo)
	if err != nil {
		return 1, err
	}
	if err := printReviewList(w, reviews, opts); err != nil {
		return 1, err
	}
	if len(reviews) > 0 {
		return 1, nil
	}
	return 0, nil
}

// printReviewList prints reviews as JSON or text.
// TRLC-LINKS: REQ-EMG-101
func printReviewList(w io.Writer, reviews []LinkReview, opts reviewLinksOptions) error {
	if !opts.JSON {
		printReviews(w, reviews, opts.Show)
		return nil
	}
	enc := json.NewEncoder(w)
	enc.SetIndent("", "  ")
	return enc.Encode(reviews)
}
