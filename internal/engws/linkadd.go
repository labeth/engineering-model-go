// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"fmt"
	"io"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/linkreview"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// LinkAddOptions name the link engws link add writes: the parent
// requirement or the evidence IDs, the child alias::ID and a rationale.
// TRLC-LINKS: REQ-EMG-119
type LinkAddOptions struct {
	Parent    string
	Evidence  []string
	Child     string
	Rationale string
}

// AddLink writes a claimed requirement link into the model in repo, pinned
// to the dependency version of the child's alias (decision D23). It checks
// before writing: the parent is a requirement of the model (a capability
// link needs one, an assumption link needs parent or evidence), each
// evidence ID validates like a link's, the child is alias::ID of a declared
// dependency and a published capability or assumption of use at the pinned
// version (resolved with the workspace off, the published content), and no
// link already joins the same parent or evidence to the child. The link is
// appended with every other line kept; a model without a link file gets
// model/req-links.yml and documents.reqLinks. The link id is
// LNK-<REQ>-<ALIAS>, made unique.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ADD
func AddLink(repo string, opts LinkAddOptions) (model.ReqLink, error) {
	manifest := filepath.Join(repo, "engmod.yml")
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		return model.ReqLink{}, err
	}
	parent, child := strings.TrimSpace(opts.Parent), strings.TrimSpace(opts.Child)
	evidence := []string{}
	for _, e := range opts.Evidence {
		if e = strings.TrimSpace(e); e != "" {
			evidence = append(evidence, e)
		}
	}
	alias, itemID, err := model.ParseQualifiedReference(child)
	if err != nil {
		return model.ReqLink{}, fmt.Errorf("child %q must be alias::CAP-... or alias::AOU-...: %v", child, err)
	}
	var pin string
	for _, d := range bundle.Manifest.Dependencies {
		if strings.TrimSpace(d.Alias) == alias {
			pin = strings.TrimSpace(d.Version)
		}
	}
	switch {
	case pin == "":
		return model.ReqLink{}, fmt.Errorf("%s names no dependency alias %q", filepath.Base(manifest), alias)
	case parent == "" && len(evidence) == 0:
		return model.ReqLink{}, fmt.Errorf("give --parent REQ or --evidence REQ[,REQ]")
	case parent == "" && strings.HasPrefix(itemID, "CAP-"):
		return model.ReqLink{}, fmt.Errorf("a link to capability %s needs --parent: the capability satisfies a local requirement (evidence alone is for assumptions of use)", child)
	}
	for _, l := range bundle.ReqLinks.ReqLinks {
		if strings.TrimSpace(l.Child) == child && strings.TrimSpace(l.Parent) == parent && sameStrings(l.Evidence, evidence) {
			from := parent
			if from == "" {
				from = "evidence " + strings.Join(evidence, ",")
			}
			return model.ReqLink{}, fmt.Errorf("link %s already links %s to %s", l.ID, from, child)
		}
	}
	used := map[string]bool{}
	for _, l := range bundle.ReqLinks.ReqLinks {
		used[strings.TrimSpace(l.ID)] = true
	}
	idSource := parent
	if idSource == "" {
		idSource = evidence[0]
	}
	link := model.ReqLink{
		ID: uniqueLinkID(linkID(idSource, alias), used), Parent: parent, Child: child, Evidence: evidence,
		Pinned: model.ReqLinkPin{Version: pin}, Status: model.ReqLinkClaimed, Rationale: oneLine(opts.Rationale),
	}
	if len(link.Evidence) == 0 {
		link.Evidence = nil
	}

	// Validate the model with the link added, then resolve the child at the
	// pin from the published dependency.
	edited := bundle
	edited.ReqLinks.ReqLinks = append(append([]model.ReqLink(nil), bundle.ReqLinks.ReqLinks...), link)
	index := len(edited.ReqLinks.ReqLinks) - 1
	prefix := fmt.Sprintf("reqLinks[%d]", index)
	for _, d := range validate.Bundle(edited) {
		if d.Severity == validate.SeverityError && (d.Path == prefix || strings.HasPrefix(d.Path, prefix+".")) {
			return model.ReqLink{}, fmt.Errorf("%s: %s", d.Code, d.Message)
		}
	}
	ctx, _, err := linkreview.EvaluateBundle(manifest, edited, linkreview.Published(reviewResolution()))
	if err != nil {
		return model.ReqLink{}, fmt.Errorf("a link is checked against the published dependency at its pin, composed with the workspace off, and that failed: %w", err)
	}
	for _, s := range ctx.Statuses {
		if s.ID != link.ID {
			continue
		}
		if s.Item == nil {
			detail := s.Detail
			if detail == "" {
				detail = string(s.State)
			}
			return model.ReqLink{}, fmt.Errorf("%s is not a published capability or assumption of use of %s at %s (%s)", child, alias, pin, detail)
		}
		if s.Item.Kind == reqlinks.KindAssumption && len(evidence) == 0 {
			return model.ReqLink{}, fmt.Errorf("%s is an assumption of use: give --evidence naming the local requirement, test or control that shows it holds", child)
		}
	}

	files := newLinkFiles(manifest)
	if err := files.appendLinks(bundle, []model.ReqLink{link}); err != nil {
		return model.ReqLink{}, err
	}
	paths := make([]string, 0, len(files.writes))
	for path := range files.writes {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	if err := writeAllOrNothing(paths, files.writes); err != nil {
		return model.ReqLink{}, err
	}
	if _, err := model.LoadBundle(manifest); err != nil {
		return link, fmt.Errorf("wrote %s, but the model no longer loads: %w", link.ID, err)
	}
	return link, nil
}

// sameStrings reports whether a and b hold the same trimmed values in order.
// TRLC-LINKS: REQ-EMG-119
func sameStrings(a, b []string) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if strings.TrimSpace(a[i]) != strings.TrimSpace(b[i]) {
			return false
		}
	}
	return true
}

// runLinkAdd adds one link and prints it.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: EVT-LINK-ADD-REQUESTED
func runLinkAdd(repo string, opts LinkAddOptions, w io.Writer) (int, error) {
	link, err := AddLink(repo, opts)
	if err != nil {
		return 1, err
	}
	from := link.Parent
	if from == "" {
		from = "evidence " + strings.Join(link.Evidence, ",")
	}
	fmt.Fprintf(w, "engws: added %s %s -> %s (%s, %s); review it with engws review-links --accept %s --by ACT-... --rationale TEXT\n", link.ID, from, link.Child, link.Pinned.Version, link.Status, link.ID)
	return 0, nil
}
