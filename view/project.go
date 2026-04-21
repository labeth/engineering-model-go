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
	flows      map[string]model.Flow
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

	if strings.TrimSpace(v.Kind) == "interaction-flow" {
		return buildInteractionFlowView(v, idx, b.Architecture.AuthoredArchitecture.Mappings, includeKinds, excludeKinds, includeMappings, excludeMappings, maxDepth), nil
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
		flows:      map[string]model.Flow{},
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
	for _, x := range b.Architecture.AuthoredArchitecture.Flows {
		idx.flows[x.ID] = x
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
	if x, ok := idx.flows[id]; ok {
		return Node{ID: id, Label: nonEmpty(x.Title, id), Kind: "flow"}
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
		case "interaction-flow":
			includeMappings = setFromSlice([]string{"flow_next", "flow_error", "flow_async", "flow_ref"})
			includeKinds = setFromSlice([]string{"flow", "flow_step", "actor", "functional_group", "functional_unit", "interface", "data_object", "deployment_target", "control", "trust_boundary", "state", "event", "referenced_element"})
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

func buildInteractionFlowView(v model.View, idx index, mappings []model.Mapping, includeKinds, excludeKinds, includeMappings, excludeMappings map[string]bool, maxDepth int) ProjectedView {
	pv := ProjectedView{ID: v.ID, Kind: v.Kind, Title: v.ID}
	nodesByID := map[string]Node{}
	edges := []Edge{}
	edgeSeen := map[string]bool{}
	adj := map[string][]string{}
	isInteractionActivityType := func(mappingType string) bool {
		switch strings.TrimSpace(mappingType) {
		case "contains", "calls", "reads", "writes", "publishes", "subscribes", "streams":
			return true
		default:
			return false
		}
	}

	addNode := func(n Node) {
		if strings.TrimSpace(n.ID) == "" {
			return
		}
		if !nodeKindAllowed(n.Kind, includeKinds, excludeKinds) {
			return
		}
		nodesByID[n.ID] = n
	}
	addEdge := func(e Edge) {
		if !mappingAllowed(e.Type, includeMappings, excludeMappings) {
			return
		}
		if _, ok := nodesByID[e.From]; !ok {
			return
		}
		if _, ok := nodesByID[e.To]; !ok {
			return
		}
		edgeKey := e.From + "|" + e.To + "|" + e.Type
		if edgeSeen[edgeKey] {
			return
		}
		edgeSeen[edgeKey] = true
		edges = append(edges, e)
		adj[e.From] = append(adj[e.From], e.To)
	}

	selectedFlowIDs := []string{}
	for _, r := range v.Roots {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if _, ok := idx.flows[r]; ok {
			selectedFlowIDs = append(selectedFlowIDs, r)
		}
	}
	if len(selectedFlowIDs) == 0 {
		for id := range idx.flows {
			selectedFlowIDs = append(selectedFlowIDs, id)
		}
	}
	sort.Strings(selectedFlowIDs)

	stepNodeID := func(flowID, stepID string) string {
		return flowID + "::" + stepID
	}

	for _, flowID := range selectedFlowIDs {
		flow := idx.flows[flowID]
		flowNode := Node{ID: flowID, Label: nonEmpty(strings.TrimSpace(flow.Title), flowID), Kind: "flow"}
		addNode(flowNode)
		participants := map[string]bool{}

		for _, step := range flow.Steps {
			stepID := strings.TrimSpace(step.ID)
			if stepID == "" {
				continue
			}
			label := strings.TrimSpace(step.Action)
			if label == "" {
				label = stepID
			}
			nID := stepNodeID(flowID, stepID)
			addNode(Node{ID: nID, Label: label, Kind: "flow_step"})
		}

		for _, entry := range flow.Entry {
			sid := strings.TrimSpace(entry)
			if sid == "" {
				continue
			}
			to := stepNodeID(flowID, sid)
			addEdge(Edge{From: flowID, To: to, Type: "flow_next", Label: "flow_next"})
		}

		for _, step := range flow.Steps {
			from := stepNodeID(flowID, strings.TrimSpace(step.ID))
			if strings.TrimSpace(step.Ref) != "" {
				if refNode := toNode(strings.TrimSpace(step.Ref), idx); strings.TrimSpace(refNode.ID) != "" && refNode.Kind != "unknown" {
					addNode(refNode)
					addEdge(Edge{From: from, To: refNode.ID, Type: "flow_ref", Label: "flow_ref"})
					participants[refNode.ID] = true
				}
			}
			edgeType := "flow_next"
			if step.Async {
				edgeType = "flow_async"
			}
			for _, next := range step.Next {
				to := stepNodeID(flowID, strings.TrimSpace(next))
				addEdge(Edge{From: from, To: to, Type: edgeType, Label: edgeType})
			}
			for _, onErr := range step.OnError {
				to := stepNodeID(flowID, strings.TrimSpace(onErr))
				addEdge(Edge{From: from, To: to, Type: "flow_error", Label: "flow_error"})
			}
		}

		for _, m := range mappings {
			t := strings.TrimSpace(m.Type)
			if !isInteractionActivityType(t) {
				continue
			}
			from := strings.TrimSpace(m.From)
			to := strings.TrimSpace(m.To)
			if from == "" || to == "" || !participants[from] {
				continue
			}
			fromNode := toNode(from, idx)
			toNode := toNode(to, idx)
			if fromNode.Kind == "unknown" || toNode.Kind == "unknown" {
				continue
			}
			addNode(fromNode)
			addNode(toNode)
			addEdge(Edge{From: fromNode.ID, To: toNode.ID, Type: t, Label: t})
		}
	}

	roots := []string{}
	for _, r := range v.Roots {
		r = strings.TrimSpace(r)
		if r == "" {
			continue
		}
		if _, ok := nodesByID[r]; ok {
			roots = append(roots, r)
		}
	}
	if len(roots) == 0 {
		for _, flowID := range selectedFlowIDs {
			if _, ok := nodesByID[flowID]; ok {
				roots = append(roots, flowID)
			}
		}
	}
	included := map[string]bool{}
	depth := map[string]int{}
	queue := []string{}
	for _, r := range roots {
		if !included[r] {
			included[r] = true
			depth[r] = 0
			queue = append(queue, r)
		}
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		if depth[cur] >= maxDepth {
			continue
		}
		for _, nxt := range adj[cur] {
			if included[nxt] {
				continue
			}
			included[nxt] = true
			depth[nxt] = depth[cur] + 1
			queue = append(queue, nxt)
		}
	}
	for id, n := range nodesByID {
		if included[id] {
			pv.Nodes = append(pv.Nodes, n)
		}
	}
	for _, e := range edges {
		if included[e.From] && included[e.To] {
			pv.Edges = append(pv.Edges, e)
		}
	}

	return sortView(pv)
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
