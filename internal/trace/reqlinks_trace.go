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

// capabilityRefs maps each requirement a capability names in its deprecated
// contract ref to that capability, as reqlinks.RealizedItems expects.
// TRLC-LINKS: REQ-EMG-105
func capabilityRefs(bundle model.Bundle) map[string]string {
	refs := map[string]string{}
	for _, entry := range bundle.Architecture.Contract.Provides {
		if ref := strings.TrimSpace(entry.Ref); ref != "" && strings.EqualFold(strings.TrimSpace(entry.Kind), string(reqlinks.KindCapability)) {
			refs[ref] = strings.TrimSpace(entry.ID)
		}
	}
	return refs
}

// publishedCapabilityRefs is capabilityRefs of the resolved dependency alias.
// TRLC-LINKS: REQ-EMG-105
func publishedCapabilityRefs(composed composition.CompositionResult, alias string) map[string]string {
	if composed.Root != nil {
		for _, child := range composed.Root.Children {
			if child.Dependency == alias {
				return capabilityRefs(child.Bundle)
			}
		}
	}
	return nil
}

// requirementParents is the upward trace of a requirement: the contract
// items it realizes, then its stakeholder sourceRefs.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func requirementParents(r model.Requirement, refs map[string]string) []string {
	parents := reqlinks.RealizedItems(r, refs)
	for _, ref := range r.SourceRefs {
		if ref = strings.TrimSpace(ref); ref != "" {
			parents = append(parents, ref)
		}
	}
	return parents
}

// countUnparented counts the requirements that trace upward to nothing
// where the requirement link rules apply: no realizes (or deprecated ref),
// not derived with a rationale, no sourceRefs, in a model that is not a
// top-level system.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func countUnparented(bundle model.Bundle, requirements model.RequirementsDocument) int {
	if _, applies := validate.RequirementLinkRules(bundle); !applies {
		return 0
	}
	if strings.EqualFold(strings.TrimSpace(bundle.Manifest.Module.Kind), "system") && bundle.Manifest.InferenceHints.TopLevel {
		return 0
	}
	refs := capabilityRefs(bundle)
	n := 0
	for _, r := range requirements.Requirements {
		if len(reqlinks.RealizedItems(r, refs)) == 0 && !(r.Derived && strings.TrimSpace(r.DerivedRationale) != "") && len(r.SourceRefs) == 0 {
			n++
		}
	}
	return n
}

// applyRequirementLinks evaluates the links of the composed root against
// its resolved dependencies and fills, for each local row that is the parent
// of a link, Links, Children (the linked item, then its realizing
// requirements) and the roll-up: a row with no code or test of its own whose
// every link resolves to an item with realizing requirements that are all
// implemented (or all test linked) in their own repository is satisfied by
// links. The realizing requirements' published rows get their status in the
// child (DelegatedStatus). It counts suspect, broken and claimed links and
// returns the requirements it satisfied.
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
	type rollUp struct{ implemented, testLinked, resolved bool }
	rolls := map[string]*rollUp{}
	for _, s := range reqlinks.EvaluateLinks(composed.Root.Bundle, composition.LinkDependencies(composed.Root)) {
		switch {
		case s.State == reqlinks.StateSuspect || s.Content == reqlinks.StateSuspect:
			matrix.Summary.LinksSuspect++
		case s.State == reqlinks.StateBroken || s.Content == reqlinks.StateBroken:
			matrix.Summary.LinksBroken++
		}
		if statusByID[s.ID] == model.ReqLinkClaimed {
			matrix.Summary.LinksClaimed++
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
			roll = &rollUp{implemented: true, testLinked: true, resolved: true}
			rolls[s.Parent] = roll
		}
		if s.Item == nil || len(s.Item.Realizers) == 0 {
			roll.resolved = false
			continue
		}
		for _, r := range sortedRealizers(s.Item.Realizers) {
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
			matrix.Summary.Verified++
		}
		satisfied[id] = true
	}
	return satisfied
}

// sortedRealizers returns the realizers sorted by ID.
// TRLC-LINKS: REQ-EMG-105
func sortedRealizers(in []reqlinks.Realizer) []reqlinks.Realizer {
	out := append([]reqlinks.Realizer(nil), in...)
	for i := 1; i < len(out); i++ {
		for j := i; j > 0 && out[j].ID < out[j-1].ID; j-- {
			out[j], out[j-1] = out[j-1], out[j]
		}
	}
	return out
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

// orphanRequirement returns the requirement ID a requirement.orphan
// message names ("requirement X is traced by ...").
// TRLC-LINKS: REQ-EMG-105
func orphanRequirement(message string) string {
	rest, ok := strings.CutPrefix(message, "requirement ")
	if !ok {
		return ""
	}
	id, _, _ := strings.Cut(rest, " ")
	return id
}
