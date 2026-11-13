// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// The element to code coverage checks of ADR-EMG-069 (second step): a
// functional unit, machine, state, transition, event, software interface,
// flow or data object that no scanned non-test source names is reported, and
// the codeCoverage.exempt list of engmod.yml records the elements that are
// deliberately not implemented here. An element is named by an
// ENGMODEL-OWNER-UNIT marker (the owner of a file) or by an ENGMODEL-LINKS
// marker. A model with no scanned source is not asked, so an empty code scope
// reports nothing. Actors, attack vectors, trust boundaries, threat scenarios,
// assumptions, out-of-scope entries, risks, POA&M items, deployment targets,
// decisions, catalog terms, views and imported alias::ID entities are never
// asked.

import (
	"fmt"
	"strings"
	"unicode/utf8"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// The untiered codes of the exemption list. code.exemption_unjustified is an
// error at every tier, like health.waiver_unjustified; model.invalid_* is a
// core code.
const (
	codeExemptionUnjustified = "code.exemption_unjustified"
	codeExemptionUnused      = "code.exemption_unused"
	codeExemptionInvalid     = "model.invalid_code_exemption"
)

// coverageLabels maps each element to code coverage code to the noun of its
// message. It is the list of codes the check emits for an unlinked element.
var coverageLabels = map[string]string{
	"code.unit_unlinked":       "functional unit",
	"code.machine_unlinked":    "machine",
	"code.interface_unlinked":  "software interface",
	"code.state_unlinked":      "machine state",
	"code.transition_unlinked": "machine transition",
	"code.event_unlinked":      "event",
	"code.flow_unlinked":       "flow",
	"code.data_unlinked":       "data object",
}

// coverageElement is one model element the check asks about.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type coverageElement struct {
	id   string
	code string
	path string
	// aliases are other identifiers whose naming counts as naming the element
	// (a machine is named by its owner unit, its states and its transitions).
	aliases []string
	// parent is the machine whose exemption also exempts the element.
	parent string
}

// coverageScope is the part of the model the builders read: the local
// functional units and the software interfaces owned by one of them.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type coverageScope struct {
	arch   model.AuthoredArchitecture
	units  map[string]bool
	ifaces map[string]bool
}

// localID reports whether an ID names an element of this model: it is not
// empty and not an imported alias::ID.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func localID(id string) bool {
	id = strings.TrimSpace(id)
	return id != "" && !strings.Contains(id, "::")
}

// newCoverageScope reads the local units and the software interfaces whose
// owner is a local unit (an interface with another owner is not asked).
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func newCoverageScope(arch model.AuthoredArchitecture) coverageScope {
	s := coverageScope{arch: arch, units: map[string]bool{}, ifaces: map[string]bool{}}
	s.addUnits()
	s.addInterfaces()
	return s
}

// addUnits records the local functional units.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (s coverageScope) addUnits() {
	for _, u := range s.arch.FunctionalUnits {
		if localID(u.ID) {
			s.units[strings.TrimSpace(u.ID)] = true
		}
	}
}

// addInterfaces records the software interfaces owned by a local unit.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (s coverageScope) addInterfaces() {
	for _, i := range s.arch.Interfaces {
		if localID(i.ID) && s.units[strings.TrimSpace(i.Owner)] {
			s.ifaces[strings.TrimSpace(i.ID)] = true
		}
	}
}

// codeNames returns the identifiers that scanned non-test source names, from
// ENGMODEL-OWNER-UNIT and ENGMODEL-LINKS markers, and whether any non-test
// source file was scanned at all.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func codeNames(scoped []codescan.CodeItem) (map[string]bool, bool) {
	names := map[string]bool{}
	add := func(id string) {
		if id = strings.TrimSpace(id); id != "" {
			names[id] = true
		}
	}
	scanned := false
	for _, it := range scoped {
		if codescan.IsVerificationCodeItem(it) {
			continue
		}
		scanned = scanned || it.Kind == "source_file"
		add(it.Owner)
		for _, link := range it.ModelLinks {
			add(link)
		}
	}
	return names, scanned
}

// named reports whether the source names the element or one of its aliases.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (el coverageElement) named(names map[string]bool) bool {
	if names[el.id] {
		return true
	}
	for _, alias := range el.aliases {
		if names[alias] {
			return true
		}
	}
	return false
}

// unitElements asks about every local functional unit.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func unitElements(s coverageScope) []coverageElement {
	var out []coverageElement
	for i, u := range s.arch.FunctionalUnits {
		if localID(u.ID) {
			out = append(out, coverageElement{id: strings.TrimSpace(u.ID), code: "code.unit_unlinked", path: fmt.Sprintf("authoredArchitecture.functionalUnits[%d]", i)})
		}
	}
	return out
}

// machineElements asks about every local machine. The machine is named by
// its owner unit, its own ID, or any of its state and transition IDs.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func machineElements(s coverageScope) []coverageElement {
	var out []coverageElement
	for i, m := range s.arch.Machines {
		if !localID(m.ID) {
			continue
		}
		aliases := append([]string{strings.TrimSpace(m.Owner)}, append(stateIDs(m), transitionIDs(m)...)...)
		out = append(out, coverageElement{id: strings.TrimSpace(m.ID), code: "code.machine_unlinked", path: fmt.Sprintf("authoredArchitecture.machines[%d]", i), aliases: aliases})
	}
	return out
}

// interfaceElements asks about every software interface owned by a local unit.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func interfaceElements(s coverageScope) []coverageElement {
	var out []coverageElement
	for i, in := range s.arch.Interfaces {
		if id := strings.TrimSpace(in.ID); s.ifaces[id] {
			out = append(out, coverageElement{id: id, code: "code.interface_unlinked", path: fmt.Sprintf("authoredArchitecture.interfaces[%d]", i)})
		}
	}
	return out
}

// stateIDs returns the trimmed state IDs of a machine.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func stateIDs(m model.Machine) []string {
	ids := make([]string, 0, len(m.States))
	for _, st := range m.States {
		ids = append(ids, strings.TrimSpace(st.ID))
	}
	return ids
}

// transitionIDs returns the trimmed transition IDs of a machine.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func transitionIDs(m model.Machine) []string {
	ids := make([]string, 0, len(m.Transitions))
	for _, tr := range m.Transitions {
		ids = append(ids, strings.TrimSpace(tr.ID))
	}
	return ids
}

// childKind is a kind of machine child the check asks about: the code, the
// path field of the machine and how to list the child IDs.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type childKind struct {
	code  string
	field string
	ids   func(model.Machine) []string
}

// machineChildren asks about the children (states or transitions) of every
// machine whose owner is a local functional unit.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func machineChildren(s coverageScope, k childKind) []coverageElement {
	var out []coverageElement
	for i, m := range s.arch.Machines {
		if !s.units[strings.TrimSpace(m.Owner)] {
			continue
		}
		for j, id := range k.ids(m) {
			out = append(out, coverageElement{id: id, code: k.code, path: fmt.Sprintf("authoredArchitecture.machines[%d].%s[%d]", i, k.field, j), parent: strings.TrimSpace(m.ID)})
		}
	}
	return out
}

// stateElements asks about the states of locally owned machines.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func stateElements(s coverageScope) []coverageElement {
	return machineChildren(s, childKind{"code.state_unlinked", "states", stateIDs})
}

// transitionElements asks about the transitions of locally owned machines.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func transitionElements(s coverageScope) []coverageElement {
	return machineChildren(s, childKind{"code.transition_unlinked", "transitions", transitionIDs})
}

// triggeringEvents returns the events that a transition of a locally owned
// machine triggers on.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (s coverageScope) triggeringEvents() map[string]bool {
	used := map[string]bool{}
	for _, m := range s.arch.Machines {
		if s.units[strings.TrimSpace(m.Owner)] {
			addTransitionEvents(used, m)
		}
	}
	return used
}

// addTransitionEvents records the events the transitions of a machine trigger on.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func addTransitionEvents(used map[string]bool, m model.Machine) {
	for _, tr := range m.Transitions {
		used[strings.TrimSpace(tr.On)] = true
	}
}

// eventElements asks about the events that a transition of a locally owned
// machine triggers on.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func eventElements(s coverageScope) []coverageElement {
	used := s.triggeringEvents()
	var out []coverageElement
	for i, e := range s.arch.Events {
		if id := strings.TrimSpace(e.ID); used[id] && localID(id) {
			out = append(out, coverageElement{id: id, code: "code.event_unlinked", path: fmt.Sprintf("authoredArchitecture.events[%d]", i)})
		}
	}
	return out
}

// flowRefs returns the elements a flow and its steps point at: the source,
// the destination, and the ref and interface of each step.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func flowRefs(f model.Flow) []string {
	refs := []string{f.SourceRef, f.DestinationRef}
	for _, st := range f.Steps {
		refs = append(refs, st.Ref, st.InterfaceRef, st.SourceRef, st.DestinationRef)
	}
	return refs
}

// flowData returns the data object IDs a flow and its steps carry.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func flowData(f model.Flow) []string {
	data := append([]string(nil), f.DataRefs...)
	for _, st := range f.Steps {
		data = append(data, st.DataIn...)
		data = append(data, st.DataOut...)
		data = append(data, st.DataRefs...)
	}
	return data
}

// touchesLocalInterface reports whether a flow points at a software interface
// owned by a local unit.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (s coverageScope) touchesLocalInterface(f model.Flow) bool {
	for _, ref := range flowRefs(f) {
		if s.ifaces[strings.TrimSpace(ref)] {
			return true
		}
	}
	return false
}

// flowElements asks about the flows that touch a local software interface.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func flowElements(s coverageScope) []coverageElement {
	var out []coverageElement
	for i, f := range s.arch.Flows {
		if s.touchesLocalInterface(f) && localID(f.ID) {
			out = append(out, coverageElement{id: strings.TrimSpace(f.ID), code: "code.flow_unlinked", path: fmt.Sprintf("authoredArchitecture.flows[%d]", i)})
		}
	}
	return out
}

// carriedData returns the data object IDs that a flow touching a local
// software interface carries.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (s coverageScope) carriedData() map[string]bool {
	carried := map[string]bool{}
	for _, f := range s.arch.Flows {
		if s.touchesLocalInterface(f) {
			addTrimmed(carried, flowData(f))
		}
	}
	return carried
}

// addTrimmed records the trimmed IDs in the set.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func addTrimmed(set map[string]bool, ids []string) {
	for _, id := range ids {
		set[strings.TrimSpace(id)] = true
	}
}

// dataElements asks about the data objects that a flow touching a local
// software interface carries.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func dataElements(s coverageScope) []coverageElement {
	carried := s.carriedData()
	var out []coverageElement
	for i, d := range s.arch.DataObjects {
		if id := strings.TrimSpace(d.ID); carried[id] && localID(id) {
			out = append(out, coverageElement{id: id, code: "code.data_unlinked", path: fmt.Sprintf("authoredArchitecture.dataObjects[%d]", i)})
		}
	}
	return out
}

// coverageBuilders is the table of the element kinds that are asked, one
// builder each. A kind that is not in it is never asked.
var coverageBuilders = []func(coverageScope) []coverageElement{
	unitElements, machineElements, interfaceElements, stateElements,
	transitionElements, eventElements, flowElements, dataElements,
}

// unnamedElements returns the elements that the names do not name.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func unnamedElements(els []coverageElement, names map[string]bool) []coverageElement {
	var out []coverageElement
	for _, el := range els {
		if !el.named(names) {
			out = append(out, el)
		}
	}
	return out
}

// unlinkedElements returns the asked elements that scanned source does not
// name, in model order.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func unlinkedElements(s coverageScope, names map[string]bool) []coverageElement {
	var out []coverageElement
	for _, build := range coverageBuilders {
		out = append(out, unnamedElements(build(s), names)...)
	}
	return out
}

// exemptionContext is what the exemption checks read.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type exemptionContext struct {
	bundle model.Bundle
	kinds  map[string]string
}

// problems returns the diagnostics of one exemption entry: an ID that is no
// model element or a reason that is too short (model.invalid_code_exemption),
// and a decision that does not exist or is not accepted
// (code.exemption_unjustified).
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (c exemptionContext) problems(i int, e model.CodeExemption) []validate.Diagnostic {
	path := fmt.Sprintf("codeCoverage.exempt[%d]", i)
	var diags []validate.Diagnostic
	add := func(code, field, message string) {
		diags = append(diags, validate.Diagnostic{
			Code: code, Severity: validate.SeverityError, Path: path + "." + field, Message: message,
			Remedy: validate.ConsistencyRemedy(code),
		})
	}
	if id := strings.TrimSpace(e.ID); c.kinds[id] != kindElement {
		add(codeExemptionInvalid, "id", fmt.Sprintf("codeCoverage.exempt[%d] id %q is not a model element", i, e.ID))
	}
	if utf8.RuneCountInString(strings.TrimSpace(e.Reason)) < model.CodeExemptionMinReason {
		add(codeExemptionInvalid, "reason", fmt.Sprintf("codeCoverage.exempt[%d] reason needs at least %d characters that say why no code names %s", i, model.CodeExemptionMinReason, e.ID))
	}
	if problem := validate.CodeExemptionDecisionProblem(c.bundle, e.Decision); problem != "" {
		add(codeExemptionUnjustified, "decision", fmt.Sprintf("codeCoverage exemption of %s is unjustified: decision %s %s; an exemption needs an accepted decision", e.ID, e.Decision, problem))
	}
	return diags
}

// exemptionDiagnostics checks every codeCoverage.exempt entry and returns the
// IDs of the entries that are valid and justified (only those exempt an
// element) with the diagnostics of the others.
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func exemptionDiagnostics(bundle model.Bundle) (map[string]bool, []validate.Diagnostic) {
	cc := bundle.Manifest.CodeCoverage
	if cc == nil {
		return nil, nil
	}
	ctx := exemptionContext{bundle: bundle, kinds: knownModelKinds(bundle, bundle.Requirements)}
	valid := map[string]bool{}
	var diags []validate.Diagnostic
	for i, e := range cc.Exempt {
		problems := ctx.problems(i, e)
		if len(problems) == 0 {
			valid[strings.TrimSpace(e.ID)] = true
		}
		diags = append(diags, problems...)
	}
	return valid, diags
}

// exemptedBy returns the ID of the valid exemption that covers an element:
// its own, or that of its machine, or "".
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func exemptedBy(valid map[string]bool, el coverageElement) string {
	switch {
	case valid[el.id]:
		return el.id
	case el.parent != "" && valid[el.parent]:
		return el.parent
	}
	return ""
}

// unlinkedDiagnostic reports one element that no scanned source names.
// TRLC-LINKS: REQ-EMG-175
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func unlinkedDiagnostic(el coverageElement) validate.Diagnostic {
	return validate.Diagnostic{
		Code: el.code, Severity: validate.SeverityWarning, Path: el.path, Remedy: validate.ConsistencyRemedy(el.code),
		Message: fmt.Sprintf("%s %s is named by no ENGMODEL-OWNER-UNIT or ENGMODEL-LINKS marker in scanned non-test source", coverageLabels[el.code], el.id),
	}
}

// unusedExemptions reports each valid exemption that exempted no element: it
// can be removed.
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func unusedExemptions(bundle model.Bundle, valid, used map[string]bool) []validate.Diagnostic {
	var diags []validate.Diagnostic
	for i, e := range bundle.Manifest.CodeCoverage.Exempt {
		id := strings.TrimSpace(e.ID)
		if !valid[id] || used[id] {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: codeExemptionUnused, Severity: validate.SeverityWarning, Path: fmt.Sprintf("codeCoverage.exempt[%d]", i),
			Message: fmt.Sprintf("codeCoverage exemption of %s matches no element that scanned source leaves unnamed; remove it", e.ID),
			Remedy:  validate.ConsistencyRemedy(codeExemptionUnused),
		})
	}
	return diags
}

// coverageDiagnostics is the element to code coverage check: the exemption
// entries are validated always, and the unlinked elements and unused entries
// only when non-test source was scanned.
// TRLC-LINKS: REQ-EMG-175, REQ-EMG-176
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func coverageDiagnostics(bundle model.Bundle, scopedCode []codescan.CodeItem) []validate.Diagnostic {
	valid, diags := exemptionDiagnostics(bundle)
	names, scanned := codeNames(scopedCode)
	if !scanned {
		return diags
	}
	used := map[string]bool{}
	for _, el := range unlinkedElements(newCoverageScope(bundle.Architecture.AuthoredArchitecture), names) {
		if id := exemptedBy(valid, el); id != "" {
			used[id] = true
			continue
		}
		diags = append(diags, unlinkedDiagnostic(el))
	}
	if bundle.Manifest.CodeCoverage == nil {
		return diags
	}
	return append(diags, unusedExemptions(bundle, valid, used)...)
}
