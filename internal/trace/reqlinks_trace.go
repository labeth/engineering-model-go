// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// The requirement link view of the trace: parents (the upward trace of each
// requirement), children (the downward trace of a composing system's
// requirements through its links) and the roll-up of a system requirement
// from the trace of the child requirements that realize its linked items.

import (
	"strings"

	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// publishedCapabilityRefs is reqlinks.CapabilityRefs of the resolved dependency alias.
// TRLC-LINKS: REQ-EMG-105
func publishedCapabilityRefs(composed composition.CompositionResult, alias string) map[string]string {
	if composed.Root != nil {
		for _, child := range composed.Root.Children {
			if child.Dependency == alias {
				return reqlinks.CapabilityRefs(child.Bundle)
			}
		}
	}
	return nil
}

// requirementParents is the upward trace of a requirement: the contract
// items it realizes (D22: stakeholder sourceRefs are no upward trace, and a
// requirement that is the parent of links has them as children).
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func requirementParents(r model.Requirement, refs map[string]string) []string {
	return reqlinks.RealizedItems(r, refs)
}

// countUnparented counts the requirements that fail the upward-trace rule
// (validate.TracesUpward) where the requirement link rules apply.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func countUnparented(bundle model.Bundle, requirements model.RequirementsDocument) int {
	if _, applies := validate.RequirementLinkRules(bundle); !applies {
		return 0
	}
	refs := reqlinks.CapabilityRefs(bundle)
	linkParents := validate.CapabilityLinkParents(bundle)
	n := 0
	for _, r := range requirements.Requirements {
		if !validate.TracesUpward(r, reqlinks.RealizedItems(r, refs), linkParents) {
			n++
		}
	}
	return n
}

// applyRequirementLinks evaluates the links of the composed root against
// its resolved dependencies and fills, for each local row that is the parent
// of a link, Links, Children (the linked item, then its realizing
// requirements) and the roll-up: a row with no code or test of its own whose
// every link is accepted, evaluates unchanged and resolves to an item with
// realizing requirements that are all implemented (or all test linked) in
// their own repository is satisfied by links; when a link would roll it up
// but is claimed or not unchanged it is "links-unreviewed" and not counted. The realizing requirements' published rows get their status in the
// child (DelegatedStatus). It counts the links that need review
// (version-mismatched, broken or claimed) and returns the requirements it
// satisfied.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func applyRequirementLinks(matrix *Matrix, composed composition.CompositionResult, traces dependencyTraces) map[string]bool {
	if composed.Root == nil || len(composed.Root.Bundle.ReqLinks.ReqLinks) == 0 {
		return nil
	}
	byAlias := map[string]*composition.ComposedSystem{}
	for _, child := range composed.Root.Children {
		byAlias[child.Dependency] = child
	}
	statusByID := map[string]string{}
	for _, l := range composed.Root.Bundle.ReqLinks.ReqLinks {
		statusByID[strings.TrimSpace(l.ID)] = strings.TrimSpace(l.Status)
	}
	rowIndex := map[string]int{}
	for i, r := range matrix.Requirements {
		rowIndex[r.ID] = i
	}
	type rollUp struct{ implemented, testLinked, resolved, reviewed bool }
	rolls := map[string]*rollUp{}
	for _, s := range reqlinks.EvaluateLinks(composed.Root.Bundle, composition.LinkDependencies(composed.Root)) {
		switch s.State {
		case reqlinks.StateVersionMismatch:
			matrix.Summary.LinksVersionMismatch++
		case reqlinks.StateBroken:
			matrix.Summary.LinksBroken++
		}
		if statusByID[s.ID] == model.ReqLinkClaimed {
			matrix.Summary.LinksClaimed++
		}
		if s.State != reqlinks.StateLocal && s.NeedsReview() {
			matrix.Summary.LinksNeedReview++
		}
		i, local := rowIndex[s.Parent]
		if s.Parent == "" || !local {
			continue
		}
		row := &matrix.Requirements[i]
		row.Links = append(row.Links, LinkRef{ID: s.ID, Child: s.Child, State: string(s.State), Status: statusByID[s.ID]})
		row.Children = appendUnique(row.Children, s.Child)
		roll := rolls[s.Parent]
		if roll == nil {
			roll = &rollUp{implemented: true, testLinked: true, resolved: true, reviewed: true}
			rolls[s.Parent] = roll
		}
		// Only a reviewed link whose item is the reviewed one counts.
		roll.reviewed = roll.reviewed && s.State == reqlinks.StateUnchanged && statusByID[s.ID] == model.ReqLinkAccepted
		if s.Item == nil || len(s.Item.Realizers) == 0 {
			roll.resolved = false
			continue
		}
		for _, r := range reqlinks.SortedRealizers(s.Item.Realizers) {
			qualified := s.Alias + "::" + r.ID
			row = &matrix.Requirements[i]
			row.Children = appendUnique(row.Children, qualified)
			status := traces.status(byAlias[s.Alias], r.ID)
			if j, ok := rowIndex[qualified]; ok && matrix.Requirements[j].DelegatedStatus == nil {
				matrix.Requirements[j].DelegatedStatus = status
			}
			roll.implemented = roll.implemented && status != nil && status.Implemented
			roll.testLinked = roll.testLinked && status != nil && status.TestLinked
		}
	}
	satisfied := map[string]bool{}
	for id, roll := range rolls {
		row := &matrix.Requirements[rowIndex[id]]
		if !roll.resolved || len(row.Code) > 0 || len(row.Verifications) > 0 || (!roll.implemented && !roll.testLinked) {
			continue
		}
		if !roll.reviewed {
			row.SatisfiedBy = "links-unreviewed"
			continue
		}
		row.SatisfiedBy, row.LinksImplemented, row.LinksTestLinked = "links", roll.implemented, roll.testLinked
		if row.Status == "orphan" {
			matrix.Summary.Orphan--
			matrix.Orphans = removeString(matrix.Orphans, id)
		}
		row.Status = "verified"
		if roll.implemented {
			row.Status = "implemented"
			matrix.Summary.Implemented++
		}
		if roll.testLinked {
			matrix.Summary.Tested++
			matrix.Summary.TestLinked++
		}
		satisfied[id] = true
	}
	return satisfied
}

// appendUnique appends value unless list holds it.
// TRLC-LINKS: REQ-EMG-105
func appendUnique(list []string, value string) []string {
	for _, v := range list {
		if v == value {
			return list
		}
	}
	return append(list, value)
}

// removeString returns list without value.
// TRLC-LINKS: REQ-EMG-105
func removeString(list []string, value string) []string {
	out := list[:0]
	for _, v := range list {
		if v != value {
			out = append(out, v)
		}
	}
	return out
}

// orphanPath is the diagnostic path of requirement.orphan for id.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-105
func orphanPath(id string) string { return "requirements[" + id + "]" }

// orphanRequirement returns the requirement ID of a requirement.orphan
// diagnostic path (orphanPath), "" for another path.
// TRLC-LINKS: REQ-EMG-105
func orphanRequirement(path string) string {
	rest, ok := strings.CutPrefix(path, "requirements[")
	if !ok {
		return ""
	}
	id, ok := strings.CutSuffix(rest, "]")
	if !ok {
		return ""
	}
	return id
}

// DependencyRequirementStatus returns a lookup of the trace status of a
// requirement of a resolved dependency in its own repository (by dependency
// alias and local requirement ID), each dependency's trace computed at most
// once; nil when the alias is not a resolved dependency.
// TRLC-LINKS: REQ-EMG-105, REQ-EMG-107
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func DependencyRequirementStatus(composed composition.CompositionResult) func(alias, requirement string) *DelegatedStatus {
	byAlias := map[string]*composition.ComposedSystem{}
	if composed.Root != nil {
		for _, child := range composed.Root.Children {
			byAlias[child.Dependency] = child
		}
	}
	traces := dependencyTraces{}
	return func(alias, requirement string) *DelegatedStatus {
		return traces.status(byAlias[alias], requirement)
	}
}
