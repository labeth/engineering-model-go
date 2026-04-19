package engmodel

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
)

func buildAIEdges(doc AIViewDocument) []AIEdge {
	entityByID := map[string]AIEntity{}
	for _, e := range doc.Entities {
		entityByID[e.ID] = e
	}

	edges := []AIEdge{}
	seen := map[string]bool{}
	addEdge := func(edge AIEdge) {
		edge.FromID = strings.TrimSpace(edge.FromID)
		edge.ToID = strings.TrimSpace(edge.ToID)
		edge.Relation = strings.TrimSpace(edge.Relation)
		if edge.FromID == "" || edge.ToID == "" || edge.Relation == "" {
			return
		}
		edge.SourceRefs = uniqueSorted(edge.SourceRefs)
		key := edge.FromID + "|" + edge.Relation + "|" + edge.ToID + "|" + edge.Origin
		if seen[key] {
			return
		}
		seen[key] = true
		edges = append(edges, edge)
	}

	for _, e := range doc.Entities {
		sourceRefs := e.SourceRefs
		switch e.Kind {
		case "functional_unit":
			if strings.TrimSpace(e.GroupID) != "" {
				addEdge(AIEdge{FromID: e.GroupID, ToID: e.ID, Relation: "contains", Origin: "authored", Confidence: "high", SourceRefs: sourceRefs})
			}
			for _, reqID := range e.RequirementIDs {
				addEdge(AIEdge{FromID: reqID, ToID: e.ID, Relation: "applies_to", Origin: "authored", Confidence: "high", SourceRefs: sourceRefs})
			}
			for _, rtID := range e.RuntimeIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: rtID, Relation: "has_runtime", Origin: "inferred", Confidence: "medium", SourceRefs: combineSourceRefs(sourceRefs, entityByID[rtID].SourceRefs)})
			}
			for _, codeID := range e.CodeIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: codeID, Relation: "has_code", Origin: "inferred", Confidence: "medium", SourceRefs: combineSourceRefs(sourceRefs, entityByID[codeID].SourceRefs)})
			}
			for _, verID := range e.VerificationIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: verID, Relation: "covered_by_verification", Origin: "verification", Confidence: "medium", SourceRefs: combineSourceRefs(sourceRefs, entityByID[verID].SourceRefs)})
			}
			for _, id := range e.InterfaceIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "has_interface", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, id := range e.DataObjectIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "has_data_object", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, id := range e.DeploymentIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "has_deployment_target", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, id := range e.ControlIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "has_control", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, id := range e.TrustBoundaryIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "bounded_by", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, id := range e.StateIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "has_state", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, id := range e.EventIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "has_event", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, id := range e.FlowIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "has_flow", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, id := range e.FlowStepIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: id, Relation: "has_flow_step", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[id].SourceRefs)})
			}
			for _, relatedID := range e.RelatedIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: relatedID, Relation: "depends_on", Origin: "authored", Confidence: "high", SourceRefs: sourceRefs})
			}
		case "requirement":
			for _, verID := range e.VerificationIDs {
				addEdge(AIEdge{FromID: verID, ToID: e.ID, Relation: "verifies", Origin: "verification", Confidence: "medium", SourceRefs: combineSourceRefs(sourceRefs, entityByID[verID].SourceRefs)})
			}
		case "verification":
			for _, reqID := range e.RequirementIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: reqID, Relation: "verifies", Origin: "verification", Confidence: "medium", SourceRefs: sourceRefs})
			}
			for _, codeID := range e.CodeIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: codeID, Relation: "implemented_by", Origin: "verification", Confidence: "medium", SourceRefs: combineSourceRefs(sourceRefs, entityByID[codeID].SourceRefs)})
			}
			for _, ownerID := range e.RelatedIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: ownerID, Relation: "derived_owner", Origin: "inferred", Confidence: "medium", SourceRefs: sourceRefs})
			}
		default:
			if e.Kind == "flow" {
				for _, stepID := range e.FlowStepIDs {
					addEdge(AIEdge{FromID: e.ID, ToID: stepID, Relation: "contains_step", Origin: "authored", Confidence: "high", SourceRefs: combineSourceRefs(sourceRefs, entityByID[stepID].SourceRefs)})
				}
			}
			for _, relatedID := range e.RelatedIDs {
				addEdge(AIEdge{FromID: e.ID, ToID: relatedID, Relation: "related_to", Origin: "authored", Confidence: "high", SourceRefs: sourceRefs})
			}
		}
	}

	for _, sp := range doc.SupportPaths {
		for i := 0; i+1 < len(sp.Path); i++ {
			addEdge(AIEdge{
				FromID:     sp.Path[i],
				ToID:       sp.Path[i+1],
				Relation:   "support_path",
				Origin:     "inferred",
				Confidence: strings.TrimSpace(sp.Confidence),
				SourceRefs: sp.SourceRefs,
			})
		}
	}

	sort.SliceStable(edges, func(i, j int) bool {
		if edges[i].FromID != edges[j].FromID {
			return edges[i].FromID < edges[j].FromID
		}
		if edges[i].Relation != edges[j].Relation {
			return edges[i].Relation < edges[j].Relation
		}
		if edges[i].ToID != edges[j].ToID {
			return edges[i].ToID < edges[j].ToID
		}
		return edges[i].Origin < edges[j].Origin
	})
	return edges
}

func renderAIEdgesNDJSON(edges []AIEdge) (string, error) {
	buf := bytes.NewBuffer(nil)
	enc := json.NewEncoder(buf)
	enc.SetEscapeHTML(false)
	for _, e := range edges {
		if err := enc.Encode(e); err != nil {
			return "", err
		}
	}
	return buf.String(), nil
}

func combineSourceRefs(parts ...[]string) []string {
	out := []string{}
	for _, p := range parts {
		out = append(out, p...)
	}
	return uniqueSorted(out)
}
