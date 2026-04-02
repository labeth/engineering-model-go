package view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type index struct {
	people     map[string]model.Person
	systems    map[string]model.SoftwareSystem
	containers map[string]model.Container
	components map[string]model.Component
	catalog    map[string]string
}

func Build(b model.Bundle, viewpointID string) (ProjectedView, []validate.Diagnostic) {
	idx := buildIndex(b)
	vp, ok := findViewpoint(b.Architecture.Viewpoints, viewpointID)
	if !ok {
		return ProjectedView{}, []validate.Diagnostic{{
			Code:     "view.not_found",
			Severity: validate.SeverityError,
			Message:  fmt.Sprintf("viewpoint %q not found", viewpointID),
			Path:     "viewpoints",
		}}
	}

	switch vp.Kind {
	case "system-context":
		return buildSystemContext(vp, idx, b.Architecture.Relationships), nil
	case "container":
		return buildContainerView(vp, idx, b.Architecture.Relationships), nil
	case "deployment":
		return buildDeploymentView(vp, idx, b.Architecture.Relationships, b)
	default:
		return ProjectedView{}, []validate.Diagnostic{{
			Code:     "view.unsupported_kind",
			Severity: validate.SeverityError,
			Message:  fmt.Sprintf("unsupported viewpoint kind %q", vp.Kind),
			Path:     "viewpoints",
		}}
	}
}

func buildIndex(b model.Bundle) index {
	idx := index{
		people:     map[string]model.Person{},
		systems:    map[string]model.SoftwareSystem{},
		containers: map[string]model.Container{},
		components: map[string]model.Component{},
		catalog:    map[string]string{},
	}
	for _, x := range b.Architecture.C4.People {
		idx.people[x.ID] = x
	}
	for _, x := range b.Architecture.C4.SoftwareSystems {
		idx.systems[x.ID] = x
	}
	for _, x := range b.Architecture.C4.Containers {
		idx.containers[x.ID] = x
	}
	for _, x := range b.Architecture.C4.Components {
		idx.components[x.ID] = x
	}
	addCatalog := func(entries []model.CatalogEntry) {
		for _, e := range entries {
			label := strings.TrimSpace(e.Name)
			if label == "" {
				label = e.ID
			}
			idx.catalog[e.ID] = label
		}
	}
	addCatalog(b.Catalog.Catalog.Systems)
	addCatalog(b.Catalog.Catalog.Actors)
	addCatalog(b.Catalog.Catalog.Events)
	addCatalog(b.Catalog.Catalog.States)
	addCatalog(b.Catalog.Catalog.Features)
	addCatalog(b.Catalog.Catalog.Modes)
	addCatalog(b.Catalog.Catalog.Conditions)
	addCatalog(b.Catalog.Catalog.DataTerms)
	return idx
}

func findViewpoint(viewpoints []model.Viewpoint, id string) (model.Viewpoint, bool) {
	for _, v := range viewpoints {
		if v.ID == id {
			return v, true
		}
	}
	return model.Viewpoint{}, false
}

func buildSystemContext(vp model.Viewpoint, idx index, relationships []model.Relationship) ProjectedView {
	included := map[string]bool{}
	allowed := relationSet(vp.IncludeRelations)

	for _, root := range vp.Roots {
		included[root] = true
	}

	// Include internal structure below root systems to seed context projection.
	for _, c := range idx.containers {
		if included[c.PartOf] {
			included[c.ID] = true
		}
	}
	for _, comp := range idx.components {
		if included[comp.PartOf] {
			included[comp.ID] = true
		}
	}

	// Expand inclusion by allowed relations until stable.
	for changed := true; changed; {
		changed = false
		for _, rel := range relationships {
			if !allowed[rel.Type] {
				continue
			}
			if included[rel.From] || included[rel.To] {
				if !included[rel.From] {
					included[rel.From] = true
					changed = true
				}
				if !included[rel.To] {
					included[rel.To] = true
					changed = true
				}
			}
		}
	}

	pv := ProjectedView{ID: vp.ID, Kind: vp.Kind, Title: vp.ID}
	pv.Nodes = materializeNodes(included, idx)
	for _, rel := range relationships {
		if !allowed[rel.Type] {
			continue
		}
		if included[rel.From] && included[rel.To] {
			pv.Edges = append(pv.Edges, Edge{From: rel.From, To: rel.To, Type: rel.Type, Label: edgeLabel(rel, idx.catalog)})
		}
	}
	return sortView(pv)
}

func buildContainerView(vp model.Viewpoint, idx index, relationships []model.Relationship) ProjectedView {
	included := map[string]bool{}
	allowed := relationSet(vp.IncludeRelations)
	includePartOf := allowed["part_of"]

	for _, root := range vp.Roots {
		included[root] = true
	}
	for _, c := range idx.containers {
		if included[c.PartOf] {
			included[c.ID] = true
		}
	}

	for changed := true; changed; {
		changed = false
		for _, rel := range relationships {
			if !allowed[rel.Type] {
				continue
			}
			if included[rel.From] || included[rel.To] {
				if !included[rel.From] {
					included[rel.From] = true
					changed = true
				}
				if !included[rel.To] {
					included[rel.To] = true
					changed = true
				}
			}
		}
	}

	pv := ProjectedView{ID: vp.ID, Kind: vp.Kind, Title: vp.ID}
	pv.Nodes = materializeNodes(included, idx)
	for _, rel := range relationships {
		if !allowed[rel.Type] {
			continue
		}
		if included[rel.From] && included[rel.To] {
			pv.Edges = append(pv.Edges, Edge{From: rel.From, To: rel.To, Type: rel.Type, Label: edgeLabel(rel, idx.catalog)})
		}
	}
	if includePartOf {
		for _, c := range idx.containers {
			if included[c.ID] && included[c.PartOf] {
				pv.Edges = append(pv.Edges, Edge{From: c.ID, To: c.PartOf, Type: "part_of", Label: "part_of"})
			}
		}
	}
	return sortView(pv)
}

func relationSet(in []string) map[string]bool {
	m := map[string]bool{}
	for _, x := range in {
		m[x] = true
	}
	return m
}

func materializeNodes(included map[string]bool, idx index) []Node {
	nodes := make([]Node, 0, len(included))
	for id := range included {
		nodes = append(nodes, toNode(id, idx))
	}
	return nodes
}

func toNode(id string, idx index) Node {
	if p, ok := idx.people[id]; ok {
		return Node{ID: id, Label: nonEmpty(p.Name, id), Kind: "person"}
	}
	if s, ok := idx.systems[id]; ok {
		kind := "system"
		if strings.EqualFold(strings.TrimSpace(s.Kind), "external_system") {
			kind = "external_system"
		}
		return Node{ID: id, Label: nonEmpty(s.Name, id), Kind: kind}
	}
	if c, ok := idx.containers[id]; ok {
		return Node{ID: id, Label: nonEmpty(c.Name, id), Kind: "container"}
	}
	if c, ok := idx.components[id]; ok {
		return Node{ID: id, Label: nonEmpty(c.Name, id), Kind: "component"}
	}
	return Node{ID: id, Label: id, Kind: "unknown"}
}

func edgeLabel(rel model.Relationship, catalog map[string]string) string {
	label := rel.Type
	if strings.TrimSpace(rel.Description) != "" {
		label = rel.Type + ": " + strings.TrimSpace(rel.Description)
	}
	if len(rel.CatalogRefs) > 0 {
		items := make([]string, 0, len(rel.CatalogRefs))
		for _, ref := range rel.CatalogRefs {
			if resolved, ok := catalog[ref]; ok && strings.TrimSpace(resolved) != "" {
				items = append(items, resolved)
				continue
			}
			items = append(items, ref)
		}
		label = label + " (catalog: " + strings.Join(items, ", ") + ")"
	}
	return label
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func sortView(v ProjectedView) ProjectedView {
	sort.SliceStable(v.Nodes, func(i, j int) bool {
		if v.Nodes[i].Kind != v.Nodes[j].Kind {
			return v.Nodes[i].Kind < v.Nodes[j].Kind
		}
		return v.Nodes[i].ID < v.Nodes[j].ID
	})
	sort.SliceStable(v.Edges, func(i, j int) bool {
		a := v.Edges[i]
		b := v.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.Label < b.Label
	})
	return v
}
