// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE

// Package linkreview evaluates the requirement links of one model and
// prepares their review: the links with their state, the child item before
// (at the link's pinned version, resolved with the workspace off) and after
// (at the dependency pin), and the line-preserving edit that records an
// accepted review in the requirement link file. engws review-links writes
// the edit; the MCP server only shows it.
package linkreview

import (
	"bytes"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/yamledit"
	"gopkg.in/yaml.v3"
)

// ItemView is a contract item as a review shows it: the canonical fields of
// the item and its realizing requirements, sorted by ID.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
type ItemView struct {
	ID                   string         `json:"id"`
	Kind                 string         `json:"kind"`
	Text                 string         `json:"text"`
	VerificationCriteria string         `json:"verificationCriteria,omitempty"`
	Fingerprint          string         `json:"fingerprint"`
	Realizers            []RealizerView `json:"realizers"`
}

// RealizerView is one realizing requirement of an ItemView.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
type RealizerView struct {
	ID                   string `json:"id"`
	Text                 string `json:"text"`
	VerificationCriteria string `json:"verificationCriteria,omitempty"`
	Status               string `json:"status,omitempty"`
}

// Review is one link that needs review: its state, the parent requirement
// and the child item at the reviewed version (Before, nil with BeforeNote
// when it cannot be resolved) and at the dependency pin (After, nil when
// the item is gone).
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
type Review struct {
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

// NewItemView converts an item; nil stays nil.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
func NewItemView(item *reqlinks.Item) *ItemView {
	if item == nil {
		return nil
	}
	view := &ItemView{ID: item.ID, Kind: string(item.Kind), Text: OneLine(item.Text), VerificationCriteria: OneLine(item.VerificationCriteria), Fingerprint: reqlinks.Fingerprint(*item), Realizers: []RealizerView{}}
	for _, r := range item.Realizers {
		view.Realizers = append(view.Realizers, RealizerView{ID: r.ID, Text: OneLine(r.Text), VerificationCriteria: OneLine(r.VerificationCriteria), Status: strings.TrimSpace(r.Status)})
	}
	sort.SliceStable(view.Realizers, func(i, j int) bool { return view.Realizers[i].ID < view.Realizers[j].ID })
	return view
}

// OneLine collapses whitespace like the canonical serialization.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
func OneLine(s string) string { return strings.Join(strings.Fields(s), " ") }

// Lines renders an item view as one line for the item and one per realizer.
// TRLC-LINKS: REQ-EMG-101
func (v *ItemView) Lines() []string {
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

// NeedsReview reports whether a link is listed for review: suspect, broken
// or version_mismatch.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
func NeedsReview(s reqlinks.LinkStatus) bool {
	switch s.State {
	case reqlinks.StateSuspect, reqlinks.StateBroken, reqlinks.StateVersionMismatch:
		return true
	}
	return false
}

// Context is a model with its links evaluated at the current pins.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
type Context struct {
	Manifest string
	Bundle   model.Bundle
	Statuses []reqlinks.LinkStatus
}

// Evaluate loads the model of manifest, composes it with options and
// evaluates its links. A model without documents.reqLinks has no statuses
// and is not composed.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func Evaluate(manifest string, options composition.ResolutionOptions) (Context, error) {
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		return Context{}, err
	}
	if bundle.ReqLinksPath == "" {
		return Context{Manifest: manifest, Bundle: bundle}, nil
	}
	comp, err := composition.GenerateCompositionFromFileWith(manifest, options)
	if err != nil {
		return Context{}, err
	}
	if comp.Root == nil {
		return Context{}, fmt.Errorf("%s does not compose: %s", filepath.Base(manifest), firstError(comp))
	}
	return Context{Manifest: manifest, Bundle: bundle, Statuses: reqlinks.EvaluateLinks(comp.Root.Bundle, composition.LinkDependencies(comp.Root))}, nil
}

// firstError returns the first error diagnostic of a composition as text.
// TRLC-LINKS: REQ-EMG-101
func firstError(comp composition.CompositionResult) string {
	for _, d := range comp.Diagnostics {
		if d.Severity == "error" {
			return d.Code + ": " + d.Message
		}
	}
	return "unknown error"
}

// Reviews returns every link of ctx that needs review, with the item at the
// reviewed version resolved through options with the workspace off (module
// cache, else the registry; never a workspace checkout, whose content is
// not that version's).
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func Reviews(ctx Context, options composition.ResolutionOptions) []Review {
	parents := map[string]string{}
	for _, r := range ctx.Bundle.Requirements.Requirements {
		parents[strings.TrimSpace(r.ID)] = OneLine(r.Text)
	}
	befores := map[string]map[string]reqlinks.Item{} // alias@version -> items
	notes := map[string]string{}
	reviews := []Review{}
	for _, s := range ctx.Statuses {
		if !NeedsReview(s) {
			continue
		}
		review := Review{
			ID: s.ID, Child: s.Child, State: string(s.State), Content: string(s.Content), Parent: s.Parent, ParentText: parents[s.Parent],
			Pinned: s.Pinned, Version: s.Version, Recorded: s.Recorded, Current: s.Current, After: NewItemView(s.Item), Detail: s.Detail,
		}
		key := s.Alias + "@" + s.Pinned
		if _, done := befores[key]; !done && s.Pinned != "" {
			befores[key], notes[key] = ItemsAtVersion(ctx.Bundle, s.Alias, s.Pinned, options)
		}
		if item, ok := befores[key][s.ItemID]; ok {
			if fp := reqlinks.Fingerprint(item); fp != s.Recorded {
				review.BeforeNote = fmt.Sprintf("before: the item at %s has fingerprint %s, not the recorded %s (the tag moved or the fingerprint was not stamped from %s)", s.Pinned, fp, s.Recorded, s.Pinned)
			} else {
				review.Before = NewItemView(&item)
			}
		} else if notes[key] != "" {
			review.BeforeNote = notes[key]
		} else {
			review.BeforeNote = fmt.Sprintf("before: %s does not exist at %s", s.Child, s.Pinned)
		}
		reviews = append(reviews, review)
	}
	return reviews
}

// ItemsAtVersion resolves dependency alias of system at version with the
// workspace off (options otherwise; use lock check so no lock is written)
// and returns its published contract items, or a note when the version
// cannot be resolved.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
func ItemsAtVersion(system model.Bundle, alias, version string, options composition.ResolutionOptions) (map[string]reqlinks.Item, string) {
	moved := system
	moved.Manifest.Dependencies = append([]model.ManifestDependency(nil), system.Manifest.Dependencies...)
	for i := range moved.Manifest.Dependencies {
		if moved.Manifest.Dependencies[i].Alias == alias {
			moved.Manifest.Dependencies[i].Version = version
		}
	}
	unavailable := fmt.Sprintf("before: version %s not available offline", version)
	options.Workspace = model.WorkspaceOff
	comp, err := composition.GenerateCompositionForBundle(moved, options)
	if err != nil || comp.Root == nil {
		return nil, unavailable
	}
	for _, dep := range composition.LinkDependencies(comp.Root) {
		if dep.Alias == alias && dep.Version == version {
			return reqlinks.PublishedItems(dep.Bundle, dep.Publications...), ""
		}
	}
	return nil, unavailable
}

// DiffLines marks lines only in before with "- ", only in after with "+ "
// and common lines with "  ", in document order (a longest common
// subsequence walk). With no before, every after line is shown unmarked.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
func DiffLines(before, after []string, haveBefore bool) []string {
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

// Acceptance is the edit that records the review of one link: the link
// status it accepts and the requirement link file content before and after.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
type Acceptance struct {
	Status reqlinks.LinkStatus
	Path   string
	Before []byte
	After  []byte
}

// Accept prepares the review of link id in ctx: the current fingerprint,
// the dependency pin, reviewedBy, reviewedOn (date, YYYY-MM-DD), the
// rationale and status accepted, editing only the touched lines of data
// (the requirement link file). It refuses an actor the model does not
// define, an unknown link, a broken link and one whose item cannot be
// evaluated. Nothing is written.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-106
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func Accept(ctx Context, data []byte, id, by, rationale, date string) (Acceptance, error) {
	if ctx.Bundle.ReqLinksPath == "" {
		return Acceptance{}, fmt.Errorf("%s names no documents.reqLinks", filepath.Base(ctx.Manifest))
	}
	actor := false
	for _, a := range ctx.Bundle.Architecture.AuthoredArchitecture.Actors {
		actor = actor || strings.TrimSpace(a.ID) == by
	}
	if !actor {
		return Acceptance{}, fmt.Errorf("--by %s is not an actor of the model (architecture.actors)", by)
	}
	var status *reqlinks.LinkStatus
	for i := range ctx.Statuses {
		if ctx.Statuses[i].ID == id {
			status = &ctx.Statuses[i]
		}
	}
	switch {
	case status == nil:
		return Acceptance{}, fmt.Errorf("no requirement link %s in %s", id, filepath.Base(ctx.Bundle.ReqLinksPath))
	case status.State == reqlinks.StateBroken || status.Content == reqlinks.StateBroken:
		return Acceptance{Status: *status}, fmt.Errorf("link %s is broken: %s no longer exists at %s; remove the link or link another contract item", id, status.Child, status.Version)
	case status.Current == "":
		return Acceptance{Status: *status}, fmt.Errorf("link %s cannot be accepted: %s (%s)", id, status.State, status.Detail)
	}
	path := ctx.Bundle.ReqLinksPath
	link := yamledit.FindListItem([]string{"reqLinks"}, "id", id)
	pinned := yamledit.FindListItem([]string{"reqLinks"}, "id", id, "pinned")
	edits := []struct {
		find  func(*yaml.Node) (*yaml.Node, error)
		key   string
		value string
	}{
		{pinned, "version", status.Version},
		{pinned, "fingerprint", status.Current},
		{link, "status", model.ReqLinkAccepted},
		{link, "rationale", OneLine(rationale)},
		{link, "reviewedBy", by},
		{link, "reviewedOn", date},
	}
	edited := data
	var err error
	for _, e := range edits {
		if current, ok := scalarAt(edited, e.find, e.key); ok && current == e.value {
			continue
		}
		if edited, err = yamledit.SetKey(edited, e.find, e.key, yamledit.Scalar(e.value)); err != nil {
			return Acceptance{Status: *status}, fmt.Errorf("%s: link %s: %w", filepath.Base(path), id, err)
		}
	}
	var check model.ReqLinksDocument
	dec := yaml.NewDecoder(bytes.NewReader(edited))
	dec.KnownFields(true)
	if err := dec.Decode(&check); err != nil {
		return Acceptance{Status: *status}, fmt.Errorf("%s: edit produced invalid YAML (nothing written): %w", filepath.Base(path), err)
	}
	return Acceptance{Status: *status, Path: path, Before: data, After: edited}, nil
}

// Patch renders the acceptance as a unified-style line diff of the link
// file under name: "--- a/name", "+++ b/name", then the changed lines
// marked "-" and "+" with the unchanged lines around them omitted.
// TRLC-LINKS: REQ-EMG-106
func (a Acceptance) Patch(name string) string {
	before := strings.Split(strings.TrimSuffix(string(a.Before), "\n"), "\n")
	after := strings.Split(strings.TrimSuffix(string(a.After), "\n"), "\n")
	var b strings.Builder
	fmt.Fprintf(&b, "--- a/%s\n+++ b/%s\n", name, name)
	for _, line := range DiffLines(before, after, true) {
		if strings.HasPrefix(line, "- ") || strings.HasPrefix(line, "+ ") {
			b.WriteString(line[:1] + line[2:] + "\n")
		}
	}
	return b.String()
}

// scalarAt returns the scalar value of key in the mapping found by find.
// TRLC-LINKS: REQ-EMG-101
func scalarAt(data []byte, find func(*yaml.Node) (*yaml.Node, error), key string) (string, bool) {
	root, err := yamledit.Parse(data)
	if err != nil {
		return "", false
	}
	mapping, err := find(root)
	if err != nil {
		return "", false
	}
	value := yamledit.MappingValue(mapping, key)
	if value == nil || value.Kind != yaml.ScalarNode {
		return "", false
	}
	return value.Value, true
}
