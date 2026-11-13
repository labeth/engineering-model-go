// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"fmt"
	"strings"

	earslint "github.com/labeth/ears-lint-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// requirementPatterns returns the EARS pattern the lint classifies each
// requirement text as (ubiquitous, state-driven, event-driven, optional-feature,
// unwanted-behaviour or complex), by requirement ID; a text the lint cannot
// parse has no entry. It runs the same lint as the EARS preflight and leaves
// its diagnostics to it.
// TRLC-LINKS: REQ-EMG-009, REQ-EMG-172
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func requirementPatterns(requirements model.RequirementsDocument, catalogDoc model.CatalogDocument) map[string]string {
	items := make([][2]string, 0, len(requirements.Requirements))
	for _, req := range requirements.Requirements {
		items = append(items, [2]string{req.ID, req.Text})
	}
	mode := earslint.ModeStrict
	if strings.EqualFold(strings.TrimSpace(requirements.LintRun.Mode), string(earslint.ModeGuided)) {
		mode = earslint.ModeGuided
	}
	results := earslint.LintEarsBatch(items, toEarsCatalog(catalogDoc), &earslint.Options{Mode: mode, CommaAsAnd: requirements.LintRun.CommaAsAnd})
	patterns := map[string]string{}
	for _, r := range results {
		if r.Pattern != "" {
			patterns[r.ID] = string(r.Pattern)
		}
	}
	return patterns
}

// modeledElements is what a requirement can be modeled by: the requirements
// that machine invariants list and the IDs of the behavior elements.
type modeledElements struct {
	invariantRequirements map[string]bool
	elements              map[string]bool
}

// modelingOf collects the requirements the machine invariants list and the
// IDs of machines, states, transitions, events and flows.
// TRLC-LINKS: REQ-EMG-172
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func modelingOf(a model.AuthoredArchitecture) modeledElements {
	m := modeledElements{invariantRequirements: map[string]bool{}, elements: map[string]bool{}}
	for _, machine := range a.Machines {
		m.addMachine(machine)
	}
	for _, e := range a.Events {
		m.elements[strings.TrimSpace(e.ID)] = true
	}
	for _, f := range a.Flows {
		m.elements[strings.TrimSpace(f.ID)] = true
	}
	return m
}

// addMachine records the machine, its states and transitions as elements and
// the requirements its invariants list.
// TRLC-LINKS: REQ-EMG-172
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (m modeledElements) addMachine(machine model.Machine) {
	m.elements[strings.TrimSpace(machine.ID)] = true
	for _, s := range machine.States {
		m.elements[strings.TrimSpace(s.ID)] = true
	}
	for _, t := range machine.Transitions {
		m.elements[strings.TrimSpace(t.ID)] = true
	}
	for _, inv := range machine.Invariants {
		for _, r := range inv.Requirements {
			m.invariantRequirements[strings.TrimSpace(r)] = true
		}
	}
}

// modeled reports whether a requirement has behavior behind it: a machine
// invariant lists it, or its appliesTo names a machine, state, transition,
// event or flow.
// TRLC-LINKS: REQ-EMG-172
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func (m modeledElements) modeled(r model.Requirement) bool {
	if m.invariantRequirements[strings.TrimSpace(r.ID)] {
		return true
	}
	for _, target := range r.AppliesTo {
		if m.elements[strings.TrimSpace(target)] {
			return true
		}
	}
	return false
}

// behaviorPattern reports whether an EARS pattern describes behavior a machine
// models: event-driven or state-driven.
// TRLC-LINKS: REQ-EMG-172
func behaviorPattern(pattern string) bool {
	return pattern == string(earslint.PatternEventDriven) || pattern == string(earslint.PatternStateDriven)
}

// behaviorExempt reports whether a requirement is not expected to have
// behavior behind it: retired, rejected or derived.
// TRLC-LINKS: REQ-EMG-172
func behaviorExempt(r model.Requirement) bool {
	status := strings.ToLower(strings.TrimSpace(r.Status))
	return r.Derived || status == "retired" || status == "rejected"
}

// lintRequirementsModeled reports behavior.requirement_unmodeled (warning;
// the tier policy sets the severity) at each event-driven or state-driven
// requirement that no machine invariant lists and whose appliesTo names no
// machine, state, transition, event or flow. A model that declares no machine
// is not asked to model behavior, and retired, rejected and derived
// requirements are exempt.
// TRLC-LINKS: REQ-EMG-172
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func lintRequirementsModeled(bundle model.Bundle, requirements model.RequirementsDocument) []validate.Diagnostic {
	a := bundle.Architecture.AuthoredArchitecture
	if len(a.Machines) == 0 {
		return nil
	}
	patterns := requirementPatterns(requirements, bundle.Catalog)
	modeling := modelingOf(a)
	var diags []validate.Diagnostic
	for _, r := range requirements.Requirements {
		pattern := patterns[r.ID]
		if !behaviorPattern(pattern) || behaviorExempt(r) || modeling.modeled(r) {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "behavior.requirement_unmodeled", Severity: validate.SeverityWarning, Path: requirementPath(r.ID),
			Message: fmt.Sprintf("%s requirement %s is modeled by no machine: no invariant lists it and its appliesTo names no machine, state, transition, event or flow", pattern, r.ID),
			Remedy:  validate.ConsistencyRemedy("behavior.requirement_unmodeled"),
		})
	}
	return diags
}
