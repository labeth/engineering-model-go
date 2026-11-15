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
	req, err := newLinkAddRequest(repo, opts)
	if err != nil {
		return model.ReqLink{}, err
	}
	if err := req.checkInputs(); err != nil {
		return model.ReqLink{}, err
	}
	if err := req.checkDuplicate(); err != nil {
		return model.ReqLink{}, err
	}
	link := req.newLink()
	if err := req.validate(link); err != nil {
		return model.ReqLink{}, err
	}
	return req.write(link)
}

// linkAddRequest is one engws link add after its inputs are normalized.
type linkAddRequest struct {
	manifest       string
	bundle         model.Bundle
	parent, child  string
	evidence       []string
	alias, itemID  string
	pin, rationale string
}

// newLinkAddRequest loads the model of repo and normalizes the options.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ADD
func newLinkAddRequest(repo string, opts LinkAddOptions) (*linkAddRequest, error) {
	manifest := filepath.Join(repo, "engmod.yml")
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		return nil, err
	}
	req := &linkAddRequest{manifest: manifest, bundle: bundle, rationale: opts.Rationale,
		parent: strings.TrimSpace(opts.Parent), child: strings.TrimSpace(opts.Child), evidence: linkAddEvidence(opts.Evidence)}
	req.alias, req.itemID, err = model.ParseQualifiedReference(req.child)
	if err != nil {
		return nil, fmt.Errorf("child %q must be alias::CAP-... or alias::AOU-...: %v", req.child, err)
	}
	req.pin = linkAddPin(bundle, req.alias)
	return req, nil
}

// linkAddEvidence returns the trimmed, non-empty evidence IDs.
// TRLC-LINKS: REQ-EMG-119
func linkAddEvidence(given []string) []string {
	evidence := []string{}
	for _, e := range given {
		if e = strings.TrimSpace(e); e != "" {
			evidence = append(evidence, e)
		}
	}
	return evidence
}

// linkAddPin returns the version of the last dependency with the alias, or
// "" when there is none.
// TRLC-LINKS: REQ-EMG-119
func linkAddPin(bundle model.Bundle, alias string) string {
	pin := ""
	for _, d := range bundle.Manifest.Dependencies {
		if strings.TrimSpace(d.Alias) == alias {
			pin = strings.TrimSpace(d.Version)
		}
	}
	return pin
}

// checkInputs refuses a child alias that is no dependency and a parent or
// evidence the child kind does not allow.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ADD
func (q *linkAddRequest) checkInputs() error {
	switch {
	case q.pin == "":
		return fmt.Errorf("%s names no dependency alias %q", filepath.Base(q.manifest), q.alias)
	case q.parent == "" && len(q.evidence) == 0:
		return fmt.Errorf("give --parent REQ or --evidence REQ[,REQ]")
	case q.parent == "" && strings.HasPrefix(q.itemID, "CAP-"):
		return fmt.Errorf("a link to capability %s needs --parent: the capability satisfies a local requirement (evidence alone is for assumptions of use)", q.child)
	}
	return nil
}

// checkDuplicate refuses a link that already joins the same parent or
// evidence to the child.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ADD
func (q *linkAddRequest) checkDuplicate() error {
	for _, l := range q.bundle.ReqLinks.ReqLinks {
		if strings.TrimSpace(l.Child) == q.child && strings.TrimSpace(l.Parent) == q.parent && sameStrings(l.Evidence, q.evidence) {
			return fmt.Errorf("link %s already links %s to %s", l.ID, linkSource(q.parent, q.evidence), q.child)
		}
	}
	return nil
}

// linkSource names what a link starts from: its parent, else its evidence.
// TRLC-LINKS: REQ-EMG-119
func linkSource(parent string, evidence []string) string {
	if parent == "" {
		return "evidence " + strings.Join(evidence, ",")
	}
	return parent
}

// newLink builds the claimed link with a unique ID.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ADD
func (q *linkAddRequest) newLink() model.ReqLink {
	used := map[string]bool{}
	for _, l := range q.bundle.ReqLinks.ReqLinks {
		used[strings.TrimSpace(l.ID)] = true
	}
	idSource := q.parent
	if idSource == "" {
		idSource = q.evidence[0]
	}
	link := model.ReqLink{
		ID: uniqueLinkID(linkID(idSource, q.alias), used), Parent: q.parent, Child: q.child, Evidence: q.evidence,
		Pinned: model.ReqLinkPin{Version: q.pin}, Status: model.ReqLinkClaimed, Rationale: oneLine(q.rationale),
	}
	if len(link.Evidence) == 0 {
		link.Evidence = nil
	}
	return link
}

// validate checks the model with the link added, then resolves the child at
// the pin from the published dependency.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ADD
func (q *linkAddRequest) validate(link model.ReqLink) error {
	edited := q.bundle
	edited.ReqLinks.ReqLinks = append(append([]model.ReqLink(nil), q.bundle.ReqLinks.ReqLinks...), link)
	if err := linkAddSchemaError(edited); err != nil {
		return err
	}
	ctx, _, err := linkreview.EvaluateBundle(q.manifest, edited, linkreview.Published(reviewResolution()))
	if err != nil {
		return fmt.Errorf("a link is checked against the published dependency at its pin, composed with the workspace off, and that failed: %w", err)
	}
	for _, s := range ctx.Statuses {
		if s.ID != link.ID {
			continue
		}
		if err := q.checkStatus(s); err != nil {
			return err
		}
	}
	return nil
}

// linkAddSchemaError returns the first validation error of the added
// (last) link of the bundle.
// TRLC-LINKS: REQ-EMG-119
func linkAddSchemaError(edited model.Bundle) error {
	prefix := fmt.Sprintf("reqLinks[%d]", len(edited.ReqLinks.ReqLinks)-1)
	for _, d := range validate.Bundle(edited) {
		if d.Severity == validate.SeverityError && (d.Path == prefix || strings.HasPrefix(d.Path, prefix+".")) {
			return fmt.Errorf("%s: %s", d.Code, d.Message)
		}
	}
	return nil
}

// checkStatus refuses a child that is no published contract item and an
// assumption of use without evidence.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ADD
func (q *linkAddRequest) checkStatus(s reqlinks.LinkStatus) error {
	if s.Item == nil {
		detail := s.Detail
		if detail == "" {
			detail = string(s.State)
		}
		return fmt.Errorf("%s is not a published capability or assumption of use of %s at %s (%s)", q.child, q.alias, q.pin, detail)
	}
	if s.Item.Kind == reqlinks.KindAssumption && len(q.evidence) == 0 {
		return fmt.Errorf("%s is an assumption of use: give --evidence naming the local requirement, test or control that shows it holds", q.child)
	}
	return nil
}

// write appends the link to the link file, all or nothing, and checks the
// model still loads.
// TRLC-LINKS: REQ-EMG-119
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-ALLOCATION-TRACE
// ENGMODEL-LINKS: TR-LINK-ADD
func (q *linkAddRequest) write(link model.ReqLink) (model.ReqLink, error) {
	files := newLinkFiles(q.manifest)
	if err := files.appendLinks(q.bundle, []model.ReqLink{link}); err != nil {
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
	if _, err := model.LoadBundle(q.manifest); err != nil {
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
	fmt.Fprintf(w, "engws: added %s %s -> %s (%s, %s); review it with engws review-links --accept %s --by ACT-... --rationale TEXT\n", link.ID, linkSource(link.Parent, link.Evidence), link.Child, link.Pinned.Version, link.Status, link.ID)
	return 0, nil
}
