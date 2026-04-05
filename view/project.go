package view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type index struct {
	groups     map[string]model.FunctionalGroup
	units      map[string]model.FunctionalUnit
	actors     map[string]model.Actor
	vectors    map[string]model.AttackVector
	references map[string]model.ReferencedElement
}

func Build(b model.Bundle, viewID string) (ProjectedView, []validate.Diagnostic) {
	idx := buildIndex(b)
	v, ok := findView(b.Architecture.Views, viewID)
	if !ok {
		return ProjectedView{}, []validate.Diagnostic{{
			Code:     "view.not_found",
			Severity: validate.SeverityError,
			Message:  fmt.Sprintf("view %q not found", viewID),
			Path:     "views",
		}}
	}

	included := map[string]bool{}
	for _, root := range v.Roots {
		if strings.TrimSpace(root) != "" {
			included[root] = true
		}
	}

	for changed := true; changed; {
		changed = false
		for _, m := range b.Architecture.AuthoredArchitecture.Mappings {
			if included[m.From] || included[m.To] {
				if !included[m.From] {
					included[m.From] = true
					changed = true
				}
				if !included[m.To] {
					included[m.To] = true
					changed = true
				}
			}
		}
	}

	pv := ProjectedView{ID: v.ID, Kind: v.Kind, Title: v.ID}
	for id := range included {
		if v.Kind == "authored-functional" {
			if _, isVector := idx.vectors[id]; isVector {
				continue
			}
		}
		pv.Nodes = append(pv.Nodes, toNode(id, idx))
	}
	for _, m := range b.Architecture.AuthoredArchitecture.Mappings {
		if v.Kind == "authored-functional" {
			if _, isVector := idx.vectors[m.From]; isVector {
				continue
			}
			if _, isVector := idx.vectors[m.To]; isVector {
				continue
			}
		}
		if included[m.From] && included[m.To] {
			label := strings.TrimSpace(m.Type)
			if d := strings.TrimSpace(m.Description); d != "" {
				label += ": " + d
			}
			pv.Edges = append(pv.Edges, Edge{From: m.From, To: m.To, Type: m.Type, Label: label})
		}
	}

	return sortView(pv), nil
}

func buildIndex(b model.Bundle) index {
	idx := index{
		groups:     map[string]model.FunctionalGroup{},
		units:      map[string]model.FunctionalUnit{},
		actors:     map[string]model.Actor{},
		vectors:    map[string]model.AttackVector{},
		references: map[string]model.ReferencedElement{},
	}
	for _, x := range b.Architecture.AuthoredArchitecture.FunctionalGroups {
		idx.groups[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.FunctionalUnits {
		idx.units[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.Actors {
		idx.actors[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.AttackVectors {
		idx.vectors[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.ReferencedElements {
		idx.references[x.ID] = x
	}
	return idx
}

func findView(views []model.View, id string) (model.View, bool) {
	for _, v := range views {
		if v.ID == id {
			return v, true
		}
	}
	return model.View{}, false
}

func toNode(id string, idx index) Node {
	if g, ok := idx.groups[id]; ok {
		return Node{ID: id, Label: nonEmpty(g.Name, id), Kind: "functional_group"}
	}
	if u, ok := idx.units[id]; ok {
		return Node{ID: id, Label: nonEmpty(u.Name, id), Kind: "functional_unit"}
	}
	if a, ok := idx.actors[id]; ok {
		return Node{ID: id, Label: nonEmpty(a.Name, id), Kind: "actor"}
	}
	if a, ok := idx.vectors[id]; ok {
		return Node{ID: id, Label: nonEmpty(a.Name, id), Kind: "attack_vector"}
	}
	if r, ok := idx.references[id]; ok {
		return Node{ID: id, Label: nonEmpty(r.Name, id), Kind: "referenced_element"}
	}
	return Node{ID: id, Label: id, Kind: "unknown"}
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
