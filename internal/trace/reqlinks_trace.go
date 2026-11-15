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

// requirementParents is the upward trace of a requirement: the contract
// items it realizes (D22: stakeholder sourceRefs are no upward trace, and a
// requirement that is the parent of links has them as children).
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func requirementParents(r model.Requirement) []string {
	return reqlinks.RealizedItems(r)
}

// countUnparented counts the requirements that fail the upward-trace rule
// (validate.TracesUpward) where the requirement link rules apply.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func countUnparented(bundle model.Bundle, requirements model.RequirementsDocument) int {
	if _, applies := validate.RequirementLinkRules(bundle); !applies {
		return 0
	}
	linkParents := validate.CapabilityLinkParents(bundle)
	n := 0
	for _, r := range requirements.Requirements {
		if !validate.TracesUpward(r, reqlinks.RealizedItems(r), linkParents) {
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
	applier := newLinkApplier(matrix, composed.Root, traces)
	for _, s := range reqlinks.EvaluateLinks(composed.Root.Bundle, composition.LinkDependencies(composed.Root)) {
		applier.countReview(s)
		applier.apply(s)
	}
	return applier.satisfy()
}

// rollUp is what the links of one requirement add up to: whether every
// realizing requirement is implemented, or test linked, in its own
// repository, whether every link resolves to realizers, and whether every
// link is accepted and unchanged.
type rollUp struct{ implemented, testLinked, resolved, reviewed bool }

// rolledUp reports whether the links could satisfy the requirement.
func (r rollUp) rolledUp() bool {
	return r.resolved && (r.implemented || r.testLinked)
}

// linkApplier applies the evaluated links of a composed root to the rows of
// a matrix.
type linkApplier struct {
	matrix     *Matrix
	byAlias    map[string]*composition.ComposedSystem
	statusByID map[string]string
	rowIndex   map[string]int
	rolls      map[string]*rollUp
	traces     dependencyTraces
}

// newLinkApplier indexes the dependencies by alias, the link review status by
// id and the matrix rows by requirement id.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func newLinkApplier(matrix *Matrix, root *composition.ComposedSystem, traces dependencyTraces) *linkApplier {
	a := &linkApplier{matrix: matrix, byAlias: childrenByAlias(root), statusByID: map[string]string{},
		rowIndex: map[string]int{}, rolls: map[string]*rollUp{}, traces: traces}
	for _, l := range root.Bundle.ReqLinks.ReqLinks {
		a.statusByID[strings.TrimSpace(l.ID)] = strings.TrimSpace(l.Status)
	}
	for i, r := range matrix.Requirements {
		a.rowIndex[r.ID] = i
	}
	return a
}

// childrenByAlias is the resolved dependencies of a composed root by alias.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func childrenByAlias(root *composition.ComposedSystem) map[string]*composition.ComposedSystem {
	byAlias := map[string]*composition.ComposedSystem{}
	if root != nil {
		for _, child := range root.Children {
			byAlias[child.Dependency] = child
		}
	}
	return byAlias
}

// countReview counts a link in the summary where it needs review.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (a *linkApplier) countReview(s reqlinks.LinkStatus) {
	switch s.State {
	case reqlinks.StateVersionMismatch:
		a.matrix.Summary.LinksVersionMismatch++
	case reqlinks.StateBroken:
		a.matrix.Summary.LinksBroken++
	}
	if a.statusByID[s.ID] == model.ReqLinkClaimed {
		a.matrix.Summary.LinksClaimed++
	}
	if s.State != reqlinks.StateLocal && s.NeedsReview() {
		a.matrix.Summary.LinksNeedReview++
	}
}

// apply records a link on the row of its local parent requirement: the link,
// the linked item and its realizing requirements as children, and their
// status in the dependency for the roll-up.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (a *linkApplier) apply(s reqlinks.LinkStatus) {
	i, local := a.rowIndex[s.Parent]
	if s.Parent == "" || !local {
		return
	}
	row := &a.matrix.Requirements[i]
	row.Links = append(row.Links, LinkRef{ID: s.ID, Child: s.Child, State: string(s.State), Status: a.statusByID[s.ID]})
	row.Children = appendUnique(row.Children, s.Child)
	roll := a.rollOf(s.Parent)
	// Only a reviewed link whose item is the reviewed one counts.
	roll.reviewed = roll.reviewed && s.State == reqlinks.StateUnchanged && a.statusByID[s.ID] == model.ReqLinkAccepted
	if s.Item == nil || len(s.Item.Realizers) == 0 {
		roll.resolved = false
		return
	}
	a.addRealizers(i, s, roll)
}

// rollOf is the roll-up of a parent requirement, started with every link
// assumed good.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (a *linkApplier) rollOf(parent string) *rollUp {
	roll := a.rolls[parent]
	if roll == nil {
		roll = &rollUp{implemented: true, testLinked: true, resolved: true, reviewed: true}
		a.rolls[parent] = roll
	}
	return roll
}

// fold narrows the roll-up by the status of one realizing requirement.
func (r *rollUp) fold(status *DelegatedStatus) {
	r.implemented = r.implemented && status != nil && status.Implemented
	r.testLinked = r.testLinked && status != nil && status.TestLinked
}

// addRealizers adds the realizing requirements of a link's item as children
// of the row at i and folds their status in the dependency into the roll-up.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (a *linkApplier) addRealizers(i int, s reqlinks.LinkStatus, roll *rollUp) {
	for _, r := range reqlinks.SortedRealizers(s.Item.Realizers) {
		qualified := s.Alias + "::" + r.ID
		row := &a.matrix.Requirements[i]
		row.Children = appendUnique(row.Children, qualified)
		status := a.traces.status(a.byAlias[s.Alias], r.ID)
		a.noteDelegated(qualified, status)
		roll.fold(status)
	}
}

// noteDelegated sets the status of a published requirement row in its
// dependency unless the row has one.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (a *linkApplier) noteDelegated(qualified string, status *DelegatedStatus) {
	if j, ok := a.rowIndex[qualified]; ok && a.matrix.Requirements[j].DelegatedStatus == nil {
		a.matrix.Requirements[j].DelegatedStatus = status
	}
}

// satisfy marks the requirements their links satisfy and returns them.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (a *linkApplier) satisfy() map[string]bool {
	satisfied := map[string]bool{}
	for id, roll := range a.rolls {
		if a.satisfyRow(id, roll) {
			satisfied[id] = true
		}
	}
	return satisfied
}

// satisfyRow rolls the links of requirement id up into its row when it has no
// code or test of its own: a reviewed roll-up satisfies it; one that is not
// reviewed marks it "links-unreviewed" and does not count.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (a *linkApplier) satisfyRow(id string, roll *rollUp) bool {
	row := &a.matrix.Requirements[a.rowIndex[id]]
	if !roll.rolledUp() || len(row.Code) > 0 || len(row.Verifications) > 0 {
		return false
	}
	if !roll.reviewed {
		row.SatisfiedBy = "links-unreviewed"
		return false
	}
	a.markSatisfied(row, id, roll)
	return true
}

// markSatisfied rolls the links up into the row: it is no longer an orphan
// and counts as implemented or test linked.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func (a *linkApplier) markSatisfied(row *Row, id string, roll *rollUp) {
	row.SatisfiedBy, row.LinksImplemented, row.LinksTestLinked = "links", roll.implemented, roll.testLinked
	if row.Status == "orphan" {
		a.matrix.Summary.Orphan--
		a.matrix.Orphans = removeString(a.matrix.Orphans, id)
	}
	row.Status = "verified"
	if roll.implemented {
		row.Status = "implemented"
		a.matrix.Summary.Implemented++
	}
	if roll.testLinked {
		a.matrix.Summary.Tested++
		a.matrix.Summary.TestLinked++
	}
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
	byAlias := childrenByAlias(composed.Root)
	traces := dependencyTraces{}
	return func(alias, requirement string) *DelegatedStatus {
		return traces.status(byAlias[alias], requirement)
	}
}
