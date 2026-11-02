// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"time"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"gopkg.in/yaml.v3"
)

// now is the clock of review-links (reviewedOn); tests replace it.
var now = time.Now

// ItemView is a contract item as review-links shows it: the canonical
// fields of the item and its realizing requirements, sorted by ID.
// TRLC-LINKS: REQ-EMG-101
type ItemView struct {
	ID                   string         `json:"id"`
	Kind                 string         `json:"kind"`
	Text                 string         `json:"text"`
	VerificationCriteria string         `json:"verificationCriteria,omitempty"`
	Fingerprint          string         `json:"fingerprint"`
	Realizers            []RealizerView `json:"realizers"`
}

// RealizerView is one realizing requirement of an ItemView.
// TRLC-LINKS: REQ-EMG-101
type RealizerView struct {
	ID                   string `json:"id"`
	Text                 string `json:"text"`
	VerificationCriteria string `json:"verificationCriteria,omitempty"`
	Status               string `json:"status,omitempty"`
}

// LinkReview is one link that needs review: its state, the parent
// requirement and the child item at the reviewed version (Before, nil with
// BeforeNote when it cannot be resolved) and at the dependency pin (After,
// nil when the item is gone).
// TRLC-LINKS: REQ-EMG-101
type LinkReview struct {
	ID         string    `json:"id"`
	Child      string    `json:"child"`
	State      string    `json:"state"`
	Content    string    `json:"content,omitempty"`
	Parent     string    `json:"parent,omitempty"`
	ParentText string    `json:"parentText,omitempty"`
	Pinned     string    `json:"pinnedVersion"`
	Version    string    `json:"dependencyVersion"`
	Recorded   string    `json:"recordedFingerprint"`
	Current    string    `json:"currentFingerprint,omitempty"`
	Before     *ItemView `json:"before,omitempty"`
	BeforeNote string    `json:"beforeNote,omitempty"`
	After      *ItemView `json:"after,omitempty"`
	Detail     string    `json:"detail,omitempty"`
}

// itemView converts an item; nil stays nil.
// TRLC-LINKS: REQ-EMG-101
func itemView(item *reqlinks.Item) *ItemView {
	if item == nil {
		return nil
	}
	view := &ItemView{ID: item.ID, Kind: string(item.Kind), Text: oneLine(item.Text), VerificationCriteria: oneLine(item.VerificationCriteria), Fingerprint: reqlinks.Fingerprint(*item), Realizers: []RealizerView{}}
	for _, r := range item.Realizers {
		view.Realizers = append(view.Realizers, RealizerView{ID: r.ID, Text: oneLine(r.Text), VerificationCriteria: oneLine(r.VerificationCriteria), Status: strings.TrimSpace(r.Status)})
	}
	sort.SliceStable(view.Realizers, func(i, j int) bool { return view.Realizers[i].ID < view.Realizers[j].ID })
	return view
}

// oneLine collapses whitespace like the canonical serialization.
// TRLC-LINKS: REQ-EMG-101
func oneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// lines renders an item view as one line for the item and one per realizer.
// TRLC-LINKS: REQ-EMG-101
func (v *ItemView) lines() []string {
	head := fmt.Sprintf("%s %s: %s", v.Kind, v.ID, v.Text)
	if v.VerificationCriteria != "" {
		head += " (verification: " + v.VerificationCriteria + ")"
	}
	out := []string{head}
	for _, r := range v.Realizers {
		status := r.Status
		if status == "" {
			status = "-"
		}
		line := fmt.Sprintf("%s [status %s]: %s", r.ID, status, r.Text)
		if r.VerificationCriteria != "" {
			line += " (verification: " + r.VerificationCriteria + ")"
		}
		out = append(out, line)
	}
	return out
}

// needsReview reports whether review-links lists a link.
// TRLC-LINKS: REQ-EMG-101
func needsReview(s reqlinks.LinkStatus) bool {
	switch s.State {
	case reqlinks.StateSuspect, reqlinks.StateBroken, reqlinks.StateVersionMismatch:
		return true
	}
	return false
}

// linkContext is a model with its links evaluated at the current pins.
type linkContext struct {
	manifest string
	bundle   model.Bundle
	statuses []reqlinks.LinkStatus
}

// evaluateRepoLinks composes the model in repo (lock check, module cache
// reuse: nothing is written but the module cache) and evaluates its links.
// TRLC-LINKS: REQ-EMG-101
func evaluateRepoLinks(repo string) (linkContext, error) {
	manifest := filepath.Join(repo, "engmod.yml")
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		return linkContext{}, err
	}
	if bundle.ReqLinksPath == "" {
		return linkContext{manifest: manifest, bundle: bundle}, nil
	}
	comp, err := engmodel.GenerateCompositionFromFileWith(manifest, engmodel.ResolutionOptions{Lock: "check", ModuleCache: "reuse"})
	if err != nil {
		return linkContext{}, err
	}
	if comp.Root == nil {
		return linkContext{}, fmt.Errorf("%s does not compose: %s", manifest, firstError(comp.Diagnostics))
	}
	return linkContext{manifest: manifest, bundle: bundle, statuses: reqlinks.EvaluateLinks(comp.Root.Bundle, engmodel.LinkDependencies(comp.Root))}, nil
}

// firstError returns the first error diagnostic as text.
// TRLC-LINKS: REQ-EMG-101
func firstError(diags []engmodel.Diagnostic) string {
	for _, d := range diags {
		if d.Severity == "error" {
			return d.Code + ": " + d.Message
		}
	}
	return "unknown error"
}

// ReviewLinks returns every link of the model in repo that needs review,
// with the item at the reviewed version resolved from the module cache or
// the registry (never from a workspace checkout, whose content is not that
// version's).
// TRLC-LINKS: REQ-EMG-101
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
func ReviewLinks(repo string) ([]LinkReview, error) {
	ctx, err := evaluateRepoLinks(repo)
	if err != nil {
		return nil, err
	}
	parents := map[string]string{}
	for _, r := range ctx.bundle.Requirements.Requirements {
		parents[strings.TrimSpace(r.ID)] = oneLine(r.Text)
	}
	befores := map[string]map[string]reqlinks.Item{} // alias@version -> items
	notes := map[string]string{}
	reviews := []LinkReview{}
	for _, s := range ctx.statuses {
		if !needsReview(s) {
			continue
		}
		review := LinkReview{
			ID: s.ID, Child: s.Child, State: string(s.State), Content: string(s.Content), Parent: s.Parent, ParentText: parents[s.Parent],
			Pinned: s.Pinned, Version: s.Version, Recorded: s.Recorded, Current: s.Current, After: itemView(s.Item), Detail: s.Detail,
		}
		key := s.Alias + "@" + s.Pinned
		if _, done := befores[key]; !done && s.Pinned != "" {
			befores[key], notes[key] = itemsAtVersion(ctx.bundle, s.Alias, s.Pinned)
		}
		if item, ok := befores[key][s.ItemID]; ok {
			if fp := reqlinks.Fingerprint(item); fp != s.Recorded {
				review.BeforeNote = fmt.Sprintf("before: the item at %s has fingerprint %s, not the recorded %s (the tag moved or the fingerprint was not stamped from %s)", s.Pinned, fp, s.Recorded, s.Pinned)
			} else {
				review.Before = itemView(&item)
			}
		} else if notes[key] != "" {
			review.BeforeNote = notes[key]
		} else {
			review.BeforeNote = fmt.Sprintf("before: %s does not exist at %s", s.Child, s.Pinned)
		}
		reviews = append(reviews, review)
	}
	return reviews, nil
}

// itemsAtVersion resolves dependency alias of system at version with the
// workspace off (module cache, else registry; lock check, so no lock is
// written) and returns its published contract items, or a note when the
// version cannot be resolved.
// TRLC-LINKS: REQ-EMG-101
func itemsAtVersion(system model.Bundle, alias, version string) (map[string]reqlinks.Item, string) {
	moved := system
	moved.Manifest.Dependencies = append([]model.ManifestDependency(nil), system.Manifest.Dependencies...)
	for i := range moved.Manifest.Dependencies {
		if moved.Manifest.Dependencies[i].Alias == alias {
			moved.Manifest.Dependencies[i].Version = version
		}
	}
	unavailable := fmt.Sprintf("before: version %s not available offline", version)
	comp, err := engmodel.GenerateCompositionForBundle(moved, engmodel.ResolutionOptions{Workspace: model.WorkspaceOff, Lock: "check", ModuleCache: "reuse"})
	if err != nil || comp.Root == nil {
		return nil, unavailable
	}
	for _, dep := range engmodel.LinkDependencies(comp.Root) {
		if dep.Alias == alias && dep.Version == version {
			return reqlinks.PublishedItems(dep.Bundle, dep.Publications...), ""
		}
	}
	return nil, unavailable
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
			before = r.Before.lines()
		} else {
			fmt.Fprintf(w, "  %s\n", r.BeforeNote)
		}
		if r.After != nil {
			fmt.Fprintf(w, "  after (%s): %s\n", r.Version, r.After.Fingerprint)
			after = r.After.lines()
		} else {
			detail := r.Detail
			if detail == "" {
				detail = "the item no longer exists"
			}
			fmt.Fprintf(w, "  after (%s): %s\n", r.Version, detail)
		}
		for _, line := range diffLines(before, after, r.Before != nil) {
			fmt.Fprintln(w, "  "+line)
		}
	}
	fmt.Fprintln(w, "engws: review each link, then engws review-links --accept LNK-... --by ACT-... --rationale TEXT")
}

// diffLines marks lines only in before with "- ", only in after with "+ "
// and common lines with "  ", in document order (a longest common
// subsequence walk). With no before, every after line is shown unmarked.
// TRLC-LINKS: REQ-EMG-101
func diffLines(before, after []string, haveBefore bool) []string {
	var out []string
	if !haveBefore {
		for _, l := range after {
			out = append(out, "  "+l)
		}
		return out
	}
	lcs := make([][]int, len(before)+1)
	for i := range lcs {
		lcs[i] = make([]int, len(after)+1)
	}
	for i := len(before) - 1; i >= 0; i-- {
		for j := len(after) - 1; j >= 0; j-- {
			if before[i] == after[j] {
				lcs[i][j] = lcs[i+1][j+1] + 1
			} else {
				lcs[i][j] = max(lcs[i+1][j], lcs[i][j+1])
			}
		}
	}
	i, j := 0, 0
	for i < len(before) || j < len(after) {
		switch {
		case i < len(before) && j < len(after) && before[i] == after[j]:
			out = append(out, "  "+after[j])
			i++
			j++
		case j == len(after) || (i < len(before) && lcs[i+1][j] >= lcs[i][j+1]):
			out = append(out, "- "+before[i])
			i++
		default:
			out = append(out, "+ "+after[j])
			j++
		}
	}
	return out
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
	ctx, err := evaluateRepoLinks(repo)
	if err != nil {
		return reqlinks.LinkStatus{}, err
	}
	if ctx.bundle.ReqLinksPath == "" {
		return reqlinks.LinkStatus{}, fmt.Errorf("%s names no documents.reqLinks", ctx.manifest)
	}
	actor := false
	for _, a := range ctx.bundle.Architecture.AuthoredArchitecture.Actors {
		actor = actor || strings.TrimSpace(a.ID) == by
	}
	if !actor {
		return reqlinks.LinkStatus{}, fmt.Errorf("--by %s is not an actor of the model (architecture.actors)", by)
	}
	var status *reqlinks.LinkStatus
	for i := range ctx.statuses {
		if ctx.statuses[i].ID == id {
			status = &ctx.statuses[i]
		}
	}
	switch {
	case status == nil:
		return reqlinks.LinkStatus{}, fmt.Errorf("no requirement link %s in %s", id, ctx.bundle.ReqLinksPath)
	case status.State == reqlinks.StateBroken || status.Content == reqlinks.StateBroken:
		return *status, fmt.Errorf("link %s is broken: %s no longer exists at %s; remove the link or link another contract item", id, status.Child, status.Version)
	case status.Current == "":
		return *status, fmt.Errorf("link %s cannot be accepted: %s (%s)", id, status.State, status.Detail)
	}
	path := ctx.bundle.ReqLinksPath
	data, err := os.ReadFile(path)
	if err != nil {
		return *status, err
	}
	link := findListItem([]string{"reqLinks"}, "id", id)
	edits := []struct {
		find  func(*yaml.Node) (*yaml.Node, error)
		key   string
		value string
	}{
		{findListItem([]string{"reqLinks"}, "id", id, "pinned"), "version", status.Version},
		{findListItem([]string{"reqLinks"}, "id", id, "pinned"), "fingerprint", status.Current},
		{link, "status", model.ReqLinkAccepted},
		{link, "rationale", oneLine(rationale)},
		{link, "reviewedBy", by},
		{link, "reviewedOn", now().Format("2006-01-02")},
	}
	edited := data
	for _, e := range edits {
		if current, ok := scalarAt(edited, e.find, e.key); ok && current == e.value {
			continue
		}
		if edited, err = setKey(edited, e.find, e.key, yamlScalar(e.value)); err != nil {
			return *status, fmt.Errorf("%s: link %s: %w", path, id, err)
		}
	}
	var check model.ReqLinksDocument
	dec := yaml.NewDecoder(bytes.NewReader(edited))
	dec.KnownFields(true)
	if err := dec.Decode(&check); err != nil {
		return *status, fmt.Errorf("%s: edit produced invalid YAML (nothing written): %w", path, err)
	}
	return *status, writeFileAtomic(path, edited)
}

// scalarAt returns the scalar value of key in the mapping found by find.
// TRLC-LINKS: REQ-EMG-101
func scalarAt(data []byte, find func(*yaml.Node) (*yaml.Node, error), key string) (string, bool) {
	root, err := parseYAML(data)
	if err != nil {
		return "", false
	}
	mapping, err := find(root)
	if err != nil {
		return "", false
	}
	value := mappingValue(mapping, key)
	if value == nil || value.Kind != yaml.ScalarNode {
		return "", false
	}
	return value.Value, true
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
