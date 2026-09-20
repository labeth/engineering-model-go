// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-052
func isSemanticConcernView(kind string) bool {
	switch strings.TrimSpace(kind) {
	case "use-case", "logical", "process", "physical", "implementation", "deployment":
		return true
	default:
		return false
	}
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

// buildSemanticConcernView projects the canonical graph directly. Legacy
// authoring reaches this path only through ProjectSemanticModel, so view logic
// never creates a second set of architecture concepts.
// TRLC-LINKS: REQ-EMG-052
// ENGMODEL-LINKS: FU-VIEW-PROJECTION, DO-CANONICAL-SEMANTIC-MODEL
func buildSemanticConcernView(v model.View, semantic model.SemanticModel) ProjectedView {
	elements := make(map[string]model.SemanticElement, len(semantic.Elements))
	for _, element := range semantic.Elements {
		elements[element.ID] = element
	}
	allowedElement := func(element model.SemanticElement) bool {
		return semanticElementAllowed(v.Kind, element)
	}
	allowedRelationship := func(relationship model.SemanticRelationship) bool {
		return semanticRelationshipAllowed(v.Kind, relationship)
	}

	included := map[string]bool{}
	depth := map[string]int{}
	queue := make([]string, 0, len(v.Roots))
	for _, root := range v.Roots {
		root = strings.TrimSpace(root)
		if element, ok := elements[root]; ok && allowedElement(element) {
			included[root], depth[root] = true, 0
			queue = append(queue, root)
		}
	}
	if len(queue) == 0 {
		for _, element := range semantic.Elements {
			if allowedElement(element) {
				included[element.ID] = true
			}
		}
	}
	maxDepth := v.MaxDepth
	if maxDepth <= 0 {
		maxDepth = 99
	}
	for len(queue) > 0 {
		current := queue[0]
		queue = queue[1:]
		if depth[current] >= maxDepth {
			continue
		}
		for _, relationship := range semantic.Relationships {
			if !allowedRelationship(relationship) {
				continue
			}
			var neighbor string
			switch current {
			case relationship.Source:
				neighbor = relationship.Target
			case relationship.Target:
				neighbor = relationship.Source
			}
			element, ok := elements[neighbor]
			if !ok || !allowedElement(element) || included[neighbor] {
				continue
			}
			included[neighbor], depth[neighbor] = true, depth[current]+1
			queue = append(queue, neighbor)
		}
	}

	projected := ProjectedView{ID: v.ID, Kind: v.Kind, Title: v.ID}
	for _, element := range semantic.Elements {
		if !included[element.ID] || !allowedElement(element) {
			continue
		}
		features := make([]string, 0, len(element.Features))
		for _, feature := range element.Features {
			detail := strings.TrimSpace(feature.Name)
			if feature.Type != "" {
				detail += ": " + feature.Type
			}
			if feature.Value != nil && feature.Value.TypedValue != nil && feature.Value.TypedValue.Quantity != nil {
				q := feature.Value.TypedValue.Quantity
				detail += " = " + formatNumber(q.Value) + " " + q.Unit
			}
			features = append(features, detail)
		}
		projected.Nodes = append(projected.Nodes, Node{
			ID: element.ID, Label: nonEmpty(element.Name, element.ID),
			Kind: semanticViewKind(element), TypeRef: element.TypeRef, Features: features,
		})
	}
	for sequence, relationship := range semantic.Relationships {
		if !allowedRelationship(relationship) || !included[relationship.Source] || !included[relationship.Target] {
			continue
		}
		label := strings.TrimSpace(relationship.SourceType)
		if label == "" {
			label = string(relationship.Kind)
		}
		projected.Edges = append(projected.Edges, Edge{
			ID: relationship.ID, From: relationship.Source, To: relationship.Target,
			Type: string(relationship.Kind), Label: label, ItemRef: relationship.ItemRef,
			Sequence: sequence,
		})
	}
	return sortSemanticView(projected)
}

// TRLC-LINKS: REQ-EMG-052
func semanticElementAllowed(viewKind string, element model.SemanticElement) bool {
	kind := semanticViewKind(element)
	switch strings.TrimSpace(viewKind) {
	case "use-case":
		return kind == "actor" || kind == "use_case"
	case "logical":
		return kind == "capability" || kind == "logical_component" || kind == "interface"
	case "process":
		return kind == "actor" || kind == "use_case" || kind == "capability" ||
			kind == "logical_component" || kind == "software_component"
	case "physical":
		return kind == "hardware" || kind == "port" || kind == "connector" || kind == "quantity"
	case "implementation":
		return kind == "software_component" || kind == "interface" || kind == "port" || kind == "data"
	case "deployment":
		return kind == "software_component" || kind == "hardware" || kind == "deployment_target"
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-052
func semanticRelationshipAllowed(viewKind string, relationship model.SemanticRelationship) bool {
	sourceType := strings.TrimSpace(relationship.SourceType)
	switch strings.TrimSpace(viewKind) {
	case "use-case":
		return relationship.Kind == model.RelationshipReference ||
			(relationship.Kind == model.RelationshipDependency &&
				(sourceType == "include" || sourceType == "participates"))
	case "logical":
		return relationship.Kind == model.RelationshipContainment ||
			relationship.Kind == model.RelationshipSpecialization ||
			relationship.Kind == model.RelationshipDependency ||
			relationship.Kind == model.RelationshipConnection
	case "process":
		return relationship.Kind == model.RelationshipTransfer
	case "physical":
		return relationship.Kind == model.RelationshipContainment ||
			relationship.Kind == model.RelationshipConnection ||
			relationship.Kind == model.RelationshipBinding ||
			relationship.Kind == model.RelationshipDependency
	case "implementation":
		return relationship.Kind == model.RelationshipContainment ||
			relationship.Kind == model.RelationshipDependency ||
			relationship.Kind == model.RelationshipTyping ||
			relationship.Kind == model.RelationshipConnection
	case "deployment":
		return relationship.Kind == model.RelationshipAllocation ||
			relationship.Kind == model.RelationshipContainment ||
			relationship.Kind == model.RelationshipDependency
	default:
		return false
	}
}

// TRLC-LINKS: REQ-EMG-052
func semanticViewKind(element model.SemanticElement) string {
	extension := strings.TrimSpace(element.Extension)
	switch extension {
	case "engineering.actor":
		return "actor"
	case "engineering.capability":
		return "capability"
	case "engineering.hardware", "engineering.hardware_item":
		return "hardware"
	case "engineering.software_component":
		return "software_component"
	case "engineering.deployment_target":
		return "deployment_target"
	case "engineering.connector":
		return "connector"
	case "engineering.data_object":
		return "data"
	}
	switch element.Kind {
	case model.ElementStakeholderDefinition, model.ElementStakeholderUsage:
		return "actor"
	case model.ElementUseCaseDefinition, model.ElementUseCaseUsage:
		return "use_case"
	case model.ElementInterfaceDefinition, model.ElementInterfaceUsage:
		return "interface"
	case model.ElementPortDefinition, model.ElementPortUsage:
		return "port"
	case model.ElementConnectionDefinition, model.ElementConnectionUsage:
		return "connector"
	case model.ElementQuantityDefinition, model.ElementUnitDefinition, model.ElementAttributeDefinition, model.ElementAttributeUsage:
		return "quantity"
	case model.ElementItemDefinition, model.ElementItemUsage:
		return "data"
	case model.ElementPartDefinition, model.ElementPartUsage:
		return "logical_component"
	default:
		return string(element.Kind)
	}
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
