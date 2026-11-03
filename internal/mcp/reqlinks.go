// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// MCP tools for requirement links: links.list, links.suspect, links.review
// and requirements.parents, and the parents and children of requirements.get.
// They read only the model the server was initialized with and the
// dependencies it composes, so a child-only server never shows a user. The
// server writes no model file: links.review returns the engws command and
// the patch it would write.

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/linkreview"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
)

// linkContext evaluates the links of the loaded model with the server's
// read-only resolution (lock check, module cache reuse).
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) linkContext() (linkreview.Context, error) {
	if s.bundle == nil || s.bundle.ManifestPath == "" {
		return linkreview.Context{}, fmt.Errorf("model not loaded from an engmod.yml")
	}
	return linkreview.Evaluate(s.bundle.ManifestPath, resolution())
}

// linksList returns every requirement link with its state, the pinned and
// the dependency version and the recorded and current fingerprint.
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) linksList() (map[string]any, error) {
	ctx, err := s.linkContext()
	if err != nil {
		return nil, err
	}
	status := map[string]model.ReqLink{}
	for _, l := range ctx.Bundle.ReqLinks.ReqLinks {
		status[strings.TrimSpace(l.ID)] = l
	}
	links := []map[string]any{}
	for _, st := range ctx.Statuses {
		l := status[st.ID]
		row := map[string]any{
			"id": st.ID, "parent": st.Parent, "child": st.Child, "evidence": nonNilStrings(l.Evidence),
			"state": string(st.State), "content": string(st.Content), "status": strings.TrimSpace(l.Status),
			"pinnedVersion": st.Pinned, "dependencyVersion": st.Version,
			"recordedFingerprint": st.Recorded, "currentFingerprint": st.Current,
			"reviewedBy": l.ReviewedBy, "reviewedOn": l.ReviewedOn,
		}
		if st.Detail != "" {
			row["detail"] = st.Detail
		}
		links = append(links, row)
	}
	return map[string]any{"links": links, "count": len(links)}, nil
}

// linksSuspect returns the links that need review (suspect, broken,
// version_mismatch) with the parent text and the child item after the
// change and, when the pinned version resolves offline, before it.
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) linksSuspect() (map[string]any, error) {
	ctx, err := s.linkContext()
	if err != nil {
		return nil, err
	}
	reviews := linkreview.Reviews(ctx, resolution())
	return map[string]any{"links": reviews, "count": len(reviews)}, nil
}

// linksReview prepares the acceptance of one link without writing it: the
// engws review-links command that records it and the patch of the
// requirement link file that command writes.
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) linksReview(id, by, rationale string) (map[string]any, error) {
	if err := requireStableID(id, "LNK-"); err != nil {
		return nil, err
	}
	if err := requireStableID(by, "ACT-"); err != nil {
		return nil, err
	}
	if strings.TrimSpace(rationale) == "" {
		rationale = "<why the changed item still satisfies the parent requirement>"
	}
	if s.bundle == nil || s.bundle.ManifestPath == "" {
		return nil, fmt.Errorf("model not loaded from an engmod.yml")
	}
	// Like engws review-links --accept: the published dependency at its pin.
	ctx, err := linkreview.EvaluateForAccept(s.bundle.ManifestPath, resolution())
	if err != nil {
		return nil, err
	}
	var data []byte
	if ctx.Bundle.ReqLinksPath != "" {
		if data, err = os.ReadFile(ctx.Bundle.ReqLinksPath); err != nil {
			return nil, err
		}
	}
	date := s.now().Format("2006-01-02")
	acceptance, err := linkreview.Accept(ctx, data, id, by, rationale, date)
	if err != nil {
		return nil, err
	}
	rel := filepath.Base(acceptance.Path)
	if r, err := filepath.Rel(filepath.Dir(ctx.Manifest), acceptance.Path); err == nil && filepath.IsLocal(r) {
		rel = filepath.ToSlash(r)
	}
	command := fmt.Sprintf("engws review-links --accept %s --by %s --rationale %s", id, by, shellQuote(linkreview.OneLine(rationale)))
	return map[string]any{
		"link": id, "child": acceptance.Status.Child, "state": string(acceptance.Status.State),
		"command": command, "path": rel, "patch": acceptance.Patch(rel), "reviewedOn": date, "written": false,
		"note": "the MCP server writes no model file: review the change (links.suspect), then run the command in the repository",
	}, nil
}

// shellQuote quotes s for a POSIX shell.
// TRLC-LINKS: REQ-EMG-106
func shellQuote(s string) string {
	return "'" + strings.ReplaceAll(s, "'", `'"'"'`) + "'"
}

// requirementTrace is the upward and downward trace of one local
// requirement: the contract items it realizes and its stakeholder
// sourceRefs (parents), and for each requirement link it is the parent of,
// the link with its state and the linked child item with its realizing
// requirements (children: the item, then each realizer, qualified).
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) requirementTrace(r model.Requirement) (realizes []string, parents []string, links []map[string]any, children []string, err error) {
	refs := map[string]string{}
	for _, entry := range s.bundle.Architecture.Contract.Provides {
		if ref := strings.TrimSpace(entry.Ref); ref != "" && strings.EqualFold(strings.TrimSpace(entry.Kind), string(reqlinks.KindCapability)) {
			refs[ref] = strings.TrimSpace(entry.ID)
		}
	}
	realizes = nonNilStrings(reqlinks.RealizedItems(r, refs))
	parents = append([]string{}, realizes...)
	for _, ref := range r.SourceRefs {
		if ref = strings.TrimSpace(ref); ref != "" {
			parents = append(parents, ref)
		}
	}
	links, children = []map[string]any{}, []string{}
	if s.bundle.ReqLinksPath == "" {
		return realizes, parents, links, children, nil
	}
	ctx, err := s.linkContext()
	if err != nil {
		return realizes, parents, links, children, err
	}
	id := strings.TrimSpace(r.ID)
	status := map[string]string{}
	for _, l := range ctx.Bundle.ReqLinks.ReqLinks {
		status[strings.TrimSpace(l.ID)] = strings.TrimSpace(l.Status)
	}
	seen := map[string]bool{}
	add := func(v string) {
		if !seen[v] {
			seen[v] = true
			children = append(children, v)
		}
	}
	for _, st := range ctx.Statuses {
		if st.Parent != id {
			continue
		}
		link := map[string]any{"id": st.ID, "child": st.Child, "state": string(st.State), "status": status[st.ID]}
		add(st.Child)
		if st.Item != nil {
			view := linkreview.NewItemView(st.Item)
			realizers := []map[string]any{}
			for _, rz := range view.Realizers {
				q := st.Alias + "::" + rz.ID
				add(q)
				realizers = append(realizers, map[string]any{"id": q, "text": rz.Text, "status": rz.Status})
			}
			link["item"] = map[string]any{"id": st.Child, "kind": view.Kind, "text": view.Text, "realizers": realizers}
		}
		links = append(links, link)
	}
	return realizes, parents, links, children, nil
}

// requirementsParents returns the trace of one requirement for
// requirements.parents.
// TRLC-LINKS: REQ-EMG-106
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) requirementsParents(id string) (map[string]any, error) {
	r := s.findRequirement(id)
	if r == nil {
		return map[string]any{"requirementId": id, "exists": false}, nil
	}
	realizes, parents, links, children, err := s.requirementTrace(*r)
	if err != nil {
		return nil, err
	}
	return map[string]any{
		"requirementId": id, "exists": true, "realizes": realizes, "parents": parents,
		"sourceRefs": nonNilStrings(r.SourceRefs), "derived": r.Derived, "derivedRationale": r.DerivedRationale,
		"links": links, "children": children,
	}, nil
}
