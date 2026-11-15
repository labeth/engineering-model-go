// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// semanticConcernViews are the view kinds that project the canonical semantic model.
var semanticConcernViews = map[string]bool{
	"use-case": true, "logical": true, "process": true, "physical": true, "implementation": true, "deployment": true,
}

// TRLC-LINKS: REQ-EMG-052
func isSemanticConcernView(kind string) bool {
	return semanticConcernViews[strings.TrimSpace(kind)]
}

// TRLC-LINKS: REQ-EMG-052
func hasAuthoredSemanticRoot(v model.View, semantics model.SemanticContent) bool {
	authored := make(map[string]bool, len(semantics.Elements))
	for _, element := range semantics.Elements {
		authored[element.ID] = true
	}
	for _, root := range v.Roots {
		if authored[strings.TrimSpace(root)] {
			return true
		}
	}
	return false
}

// semanticProjection walks the canonical graph outwards from the roots of a
// view.
type semanticProjection struct {
	view     model.View
	semantic model.SemanticModel
	elements map[string]model.SemanticElement
	included map[string]bool
	depth    map[string]int
	queue    []string
}

// buildSemanticConcernView projects the canonical graph directly. Legacy
// authoring reaches this path only through ProjectSemanticModel, so view logic
// never creates a second set of architecture concepts.
// TRLC-LINKS: REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION, DO-CANONICAL-SEMANTIC-MODEL
func buildSemanticConcernView(v model.View, semantic model.SemanticModel) ProjectedView {
	p := &semanticProjection{
		view:     v,
		semantic: semantic,
		elements: make(map[string]model.SemanticElement, len(semantic.Elements)),
		included: map[string]bool{},
		depth:    map[string]int{},
	}
	for _, element := range semantic.Elements {
		p.elements[element.ID] = element
	}
	p.seedRoots()
	p.run(effectiveMaxDepth(v))
	return sortSemanticView(p.project())
}

// allowedElement reports whether the view kind shows the element.
// TRLC-LINKS: REQ-EMG-052
func (p *semanticProjection) allowedElement(element model.SemanticElement) bool {
	return semanticElementAllowed(p.view.Kind, element)
}

// allowedRelationship reports whether the view kind shows the relationship.
// TRLC-LINKS: REQ-EMG-052
func (p *semanticProjection) allowedRelationship(relationship model.SemanticRelationship) bool {
	return semanticRelationshipAllowed(p.view.Kind, relationship)
}

// seedRoots includes the roots that are shown elements, or every shown element
// when none is.
// TRLC-LINKS: REQ-EMG-052
func (p *semanticProjection) seedRoots() {
	for _, root := range p.view.Roots {
		root = strings.TrimSpace(root)
		if element, ok := p.elements[root]; ok && p.allowedElement(element) {
			p.included[root], p.depth[root] = true, 0
			p.queue = append(p.queue, root)
		}
	}
	if len(p.queue) > 0 {
		return
	}
	for _, element := range p.semantic.Elements {
		if p.allowedElement(element) {
			p.included[element.ID] = true
		}
	}
}

// run includes the elements the shown relationships lead to from the queued
// ones, down to the depth.
// TRLC-LINKS: REQ-EMG-052
func (p *semanticProjection) run(maxDepth int) {
	for len(p.queue) > 0 {
		current := p.queue[0]
		p.queue = p.queue[1:]
		if p.depth[current] < maxDepth {
			p.expand(current)
		}
	}
}

// expand includes the shown elements at the other end of the shown relationships of the element.
// TRLC-LINKS: REQ-EMG-052
func (p *semanticProjection) expand(current string) {
	for _, relationship := range p.semantic.Relationships {
		if p.allowedRelationship(relationship) {
			p.reach(relationshipNeighbor(relationship, current), current)
		}
	}
}

// relationshipNeighbor is the other end of the relationship, or empty when the
// element is neither end.
// TRLC-LINKS: REQ-EMG-052
func relationshipNeighbor(relationship model.SemanticRelationship, current string) string {
	switch current {
	case relationship.Source:
		return relationship.Target
	case relationship.Target:
		return relationship.Source
	}
	return ""
}

// reach includes the neighbor, one step beyond the current element, when it is a
// shown element that is not included yet.
// TRLC-LINKS: REQ-EMG-052
func (p *semanticProjection) reach(neighbor, current string) {
	element, ok := p.elements[neighbor]
	if !ok || !p.allowedElement(element) || p.included[neighbor] {
		return
	}
	p.included[neighbor], p.depth[neighbor] = true, p.depth[current]+1
	p.queue = append(p.queue, neighbor)
}

// project is the included shown elements and the shown relationships between them.
// TRLC-LINKS: REQ-EMG-052
func (p *semanticProjection) project() ProjectedView {
	projected := ProjectedView{ID: p.view.ID, Kind: p.view.Kind, Title: p.view.ID}
	for _, element := range p.semantic.Elements {
		if p.included[element.ID] && p.allowedElement(element) {
			projected.Nodes = append(projected.Nodes, semanticNode(element))
		}
	}
	for sequence, relationship := range p.semantic.Relationships {
		if p.allowedRelationship(relationship) && p.included[relationship.Source] && p.included[relationship.Target] {
			projected.Edges = append(projected.Edges, semanticEdge(relationship, sequence))
		}
	}
	return projected
}

// semanticNode is the node of an element with its features.
// TRLC-LINKS: REQ-EMG-052
func semanticNode(element model.SemanticElement) Node {
	features := make([]string, 0, len(element.Features))
	for _, feature := range element.Features {
		features = append(features, featureDetail(feature))
	}
	return Node{
		ID: element.ID, Label: nonEmpty(element.Name, element.ID),
		Kind: semanticViewKind(element), TypeRef: element.TypeRef, Features: features,
	}
}

// featureDetail is a feature as name, type and quantity value.
// TRLC-LINKS: REQ-EMG-052
func featureDetail(feature model.SemanticFeature) string {
	detail := strings.TrimSpace(feature.Name)
	if feature.Type != "" {
		detail += ": " + feature.Type
	}
	if feature.Value != nil && feature.Value.TypedValue != nil && feature.Value.TypedValue.Quantity != nil {
		q := feature.Value.TypedValue.Quantity
		detail += " = " + formatNumber(q.Value) + " " + q.Unit
	}
	return detail
}

// semanticEdge is the edge of a relationship, at its sequence in the model.
// TRLC-LINKS: REQ-EMG-052
func semanticEdge(relationship model.SemanticRelationship, sequence int) Edge {
	label := strings.TrimSpace(relationship.SourceType)
	if label == "" {
		label = string(relationship.Kind)
	}
	return Edge{
		ID: relationship.ID, From: relationship.Source, To: relationship.Target,
		Type: string(relationship.Kind), Label: label, ItemRef: relationship.ItemRef,
		Sequence: sequence,
	}
}

// semanticElementKinds are the element kinds each semantic view kind shows.
var semanticElementKinds = map[string][]string{
	"use-case":       {"actor", "use_case"},
	"logical":        {"capability", "logical_component", "interface"},
	"process":        {"actor", "use_case", "capability", "logical_component", "software_component"},
	"physical":       {"hardware", "port", "connector", "quantity"},
	"implementation": {"software_component", "interface", "port", "data"},
	"deployment":     {"software_component", "hardware", "deployment_target"},
}

// TRLC-LINKS: REQ-EMG-052
func semanticElementAllowed(viewKind string, element model.SemanticElement) bool {
	return slices.Contains(semanticElementKinds[strings.TrimSpace(viewKind)], semanticViewKind(element))
}

// semanticRelationshipKinds are the relationship kinds each semantic view kind
// shows, other than the use-case view, which also depends on the dependency type.
var semanticRelationshipKinds = map[string][]model.SemanticRelationshipKind{
	"logical":        {model.RelationshipContainment, model.RelationshipSpecialization, model.RelationshipDependency, model.RelationshipConnection},
	"process":        {model.RelationshipTransfer},
	"physical":       {model.RelationshipContainment, model.RelationshipConnection, model.RelationshipBinding, model.RelationshipDependency},
	"implementation": {model.RelationshipContainment, model.RelationshipDependency, model.RelationshipTyping, model.RelationshipConnection},
	"deployment":     {model.RelationshipAllocation, model.RelationshipContainment, model.RelationshipDependency},
}

// TRLC-LINKS: REQ-EMG-052
func semanticRelationshipAllowed(viewKind string, relationship model.SemanticRelationship) bool {
	kind := strings.TrimSpace(viewKind)
	if kind == "use-case" {
		return useCaseRelationshipAllowed(relationship)
	}
	return slices.Contains(semanticRelationshipKinds[kind], relationship.Kind)
}

// useCaseRelationshipAllowed reports whether the use-case view shows the
// relationship: a reference, or an include or participation dependency.
// TRLC-LINKS: REQ-EMG-052
func useCaseRelationshipAllowed(relationship model.SemanticRelationship) bool {
	sourceType := strings.TrimSpace(relationship.SourceType)
	if relationship.Kind == model.RelationshipReference {
		return true
	}
	return relationship.Kind == model.RelationshipDependency && (sourceType == "include" || sourceType == "participates")
}

// semanticExtensionKinds are the view kinds of the engineering extensions.
var semanticExtensionKinds = map[string]string{
	"engineering.actor":              "actor",
	"engineering.capability":         "capability",
	"engineering.hardware":           "hardware",
	"engineering.hardware_item":      "hardware",
	"engineering.software_component": "software_component",
	"engineering.deployment_target":  "deployment_target",
	"engineering.connector":          "connector",
	"engineering.data_object":        "data",
}

// semanticElementViewKinds are the view kinds of the element kinds.
var semanticElementViewKinds = map[model.SemanticElementKind]string{
	model.ElementStakeholderDefinition: "actor", model.ElementStakeholderUsage: "actor",
	model.ElementUseCaseDefinition: "use_case", model.ElementUseCaseUsage: "use_case",
	model.ElementInterfaceDefinition: "interface", model.ElementInterfaceUsage: "interface",
	model.ElementPortDefinition: "port", model.ElementPortUsage: "port",
	model.ElementConnectionDefinition: "connector", model.ElementConnectionUsage: "connector",
	model.ElementQuantityDefinition: "quantity", model.ElementUnitDefinition: "quantity",
	model.ElementAttributeDefinition: "quantity", model.ElementAttributeUsage: "quantity",
	model.ElementItemDefinition: "data", model.ElementItemUsage: "data",
	model.ElementPartDefinition: "logical_component", model.ElementPartUsage: "logical_component",
}

// TRLC-LINKS: REQ-EMG-052
func semanticViewKind(element model.SemanticElement) string {
	if kind, ok := semanticExtensionKinds[strings.TrimSpace(element.Extension)]; ok {
		return kind
	}
	if kind, ok := semanticElementViewKinds[element.Kind]; ok {
		return kind
	}
	return string(element.Kind)
}

// TRLC-LINKS: REQ-EMG-052
func sortSemanticView(projected ProjectedView) ProjectedView {
	sort.SliceStable(projected.Nodes, func(i, j int) bool { return projected.Nodes[i].ID < projected.Nodes[j].ID })
	if projected.Kind != "process" {
		sort.SliceStable(projected.Edges, func(i, j int) bool {
			if projected.Edges[i].From != projected.Edges[j].From {
				return projected.Edges[i].From < projected.Edges[j].From
			}
			if projected.Edges[i].To != projected.Edges[j].To {
				return projected.Edges[i].To < projected.Edges[j].To
			}
			return projected.Edges[i].ID < projected.Edges[j].ID
		})
	}
	return projected
}

// TRLC-LINKS: REQ-EMG-052
func formatNumber(value float64) string {
	return strings.TrimRight(strings.TrimRight(fmt.Sprintf("%.6f", value), "0"), ".")
}
