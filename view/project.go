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
	interfaces map[string]model.Interface
	data       map[string]model.DataObject
	targets    map[string]model.DeploymentTarget
	controls   map[string]model.Control
	boundaries map[string]model.TrustBoundary
	states     map[string]model.State
	events     map[string]model.Event
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

	includeKinds, excludeKinds, includeMappings, excludeMappings := resolveViewSemantics(v)
	maxDepth := v.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 99
	}

	included := map[string]bool{}
	depth := map[string]int{}
	queue := []string{}
	for _, root := range v.Roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if !included[root] {
			included[root] = true
			depth[root] = 0
			queue = append(queue, root)
		}
	}

	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if depth[current] >= maxDepth {
			continue
		}
		for _, m := range b.Architecture.AuthoredArchitecture.Mappings {
			if !mappingAllowed(strings.TrimSpace(m.Type), includeMappings, excludeMappings) {
				continue
			}
			neighbors := []string{}
			if m.From == current {
				neighbors = append(neighbors, m.To)
			}
			if m.To == current {
				neighbors = append(neighbors, m.From)
			}
			for _, n := range neighbors {
				n = strings.TrimSpace(n)
				if n == "" {
					continue
				}
				if !nodeKindAllowed(kindForID(n, idx), includeKinds, excludeKinds) {
					continue
				}
				if !included[n] {
					included[n] = true
					depth[n] = depth[current] + 1
					queue = append(queue, n)
				}
			}
		}
	}

	pv := ProjectedView{ID: v.ID, Kind: v.Kind, Title: v.ID}
	for id := range included {
		kind := kindForID(id, idx)
		if !nodeKindAllowed(kind, includeKinds, excludeKinds) {
			continue
		}
		pv.Nodes = append(pv.Nodes, toNode(id, idx))
	}
	for _, m := range b.Architecture.AuthoredArchitecture.Mappings {
		if !mappingAllowed(strings.TrimSpace(m.Type), includeMappings, excludeMappings) {
			continue
		}
		if included[m.From] && included[m.To] && nodeKindAllowed(kindForID(m.From, idx), includeKinds, excludeKinds) && nodeKindAllowed(kindForID(m.To, idx), includeKinds, excludeKinds) {
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
		interfaces: map[string]model.Interface{},
		data:       map[string]model.DataObject{},
		targets:    map[string]model.DeploymentTarget{},
		controls:   map[string]model.Control{},
		boundaries: map[string]model.TrustBoundary{},
		states:     map[string]model.State{},
		events:     map[string]model.Event{},
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
	for _, x := range b.Architecture.AuthoredArchitecture.Interfaces {
		idx.interfaces[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.DataObjects {
		idx.data[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.DeploymentTargets {
		idx.targets[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.Controls {
		idx.controls[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.TrustBoundaries {
		idx.boundaries[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.States {
		idx.states[x.ID] = x
	}
	for _, x := range b.Architecture.AuthoredArchitecture.Events {
		idx.events[x.ID] = x
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
	if x, ok := idx.interfaces[id]; ok {
		return Node{ID: id, Label: nonEmpty(x.Name, id), Kind: "interface"}
	}
	if x, ok := idx.data[id]; ok {
		return Node{ID: id, Label: nonEmpty(x.Name, id), Kind: "data_object"}
	}
	if x, ok := idx.targets[id]; ok {
		return Node{ID: id, Label: nonEmpty(x.Name, id), Kind: "deployment_target"}
	}
	if x, ok := idx.controls[id]; ok {
		return Node{ID: id, Label: nonEmpty(x.Name, id), Kind: "control"}
	}
	if x, ok := idx.boundaries[id]; ok {
		return Node{ID: id, Label: nonEmpty(x.Name, id), Kind: "trust_boundary"}
	}
	if x, ok := idx.states[id]; ok {
		return Node{ID: id, Label: nonEmpty(x.Name, id), Kind: "state"}
	}
	if x, ok := idx.events[id]; ok {
		return Node{ID: id, Label: nonEmpty(x.Name, id), Kind: "event"}
	}
	return Node{ID: id, Label: id, Kind: "unknown"}
}

func kindForID(id string, idx index) string {
	return toNode(id, idx).Kind
}

func mappingAllowed(mapping string, includes, excludes map[string]bool) bool {
	mapping = strings.TrimSpace(mapping)
	if mapping == "" {
		return false
	}
	if len(includes) > 0 && !includes[mapping] {
		return false
	}
	if excludes[mapping] {
		return false
	}
	return true
}

func nodeKindAllowed(kind string, includes, excludes map[string]bool) bool {
	kind = strings.TrimSpace(kind)
	if kind == "" {
		return false
	}
	if len(includes) > 0 && !includes[kind] {
		return false
	}
	if excludes[kind] {
		return false
	}
	return true
}

func resolveViewSemantics(v model.View) (map[string]bool, map[string]bool, map[string]bool, map[string]bool) {
	includeKinds := setFromSlice(v.IncludeKinds)
	excludeKinds := setFromSlice(v.ExcludeKinds)
	includeMappings := setFromSlice(v.IncludeMappings)
	excludeMappings := setFromSlice(v.ExcludeMappings)

	if len(includeKinds) == 0 && len(includeMappings) == 0 {
		switch strings.TrimSpace(v.Kind) {
		case "architecture-intent":
			excludeKinds["attack_vector"] = true
		case "communication":
			includeMappings = setFromSlice([]string{"interacts_with", "calls", "publishes", "subscribes", "reads", "writes", "streams", "depends_on"})
		case "deployment":
			includeMappings = setFromSlice([]string{"deployed_to", "allocated_to", "depends_on", "contains"})
			includeKinds = setFromSlice([]string{"functional_group", "functional_unit", "deployment_target", "interface", "referenced_element", "trust_boundary"})
		case "security":
			includeMappings = setFromSlice([]string{"targets", "mitigated_by", "bounded_by", "guarded_by", "depends_on"})
			includeKinds = setFromSlice([]string{"functional_group", "functional_unit", "attack_vector", "control", "trust_boundary", "referenced_element", "interface", "deployment_target"})
		case "traceability":
			includeMappings = setFromSlice([]string{"implements", "satisfies", "verified_by", "allocated_to", "deployed_to", "depends_on"})
			includeKinds = setFromSlice([]string{"functional_group", "functional_unit", "interface", "data_object", "deployment_target", "control", "referenced_element"})
		case "state-lifecycle":
			includeMappings = setFromSlice([]string{"transitions_to", "triggered_by", "guarded_by"})
			includeKinds = setFromSlice([]string{"state", "event", "control", "trust_boundary", "referenced_element"})
		}
	}

	return includeKinds, excludeKinds, includeMappings, excludeMappings
}

func setFromSlice(in []string) map[string]bool {
	out := map[string]bool{}
	for _, x := range in {
		x = strings.TrimSpace(x)
		if x == "" {
			continue
		}
		out[x] = true
	}
	return out
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
