// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// checkFlows validates each flow, its steps and the links between them.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlows() {
	for i, f := range v.b.Architecture.AuthoredArchitecture.Flows {
		v.checkFlow(indexedPath("authoredArchitecture.flows", i), f)
	}
}

// checkFlow validates one flow: its vocabulary and references, its steps and,
// when it has steps, their links.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlow(path string, f model.Flow) {
	v.checkFlowVocabulary(path, f)
	v.checkFlowRefs(path, f)
	stepOwner := v.checkFlowSteps(path, f)
	if len(stepOwner) == 0 {
		v.fail("model.empty_flow_steps", path, fmt.Sprintf("flow %q must include at least one step", strings.TrimSpace(f.ID)))
		return
	}
	links := stepLinks{v: v, flowID: strings.TrimSpace(f.ID), owner: stepOwner}
	for j, s := range f.Steps {
		links.checkStep(indexedPath(path+".steps", j), s)
	}
	links.checkEntriesAndExits(path, f)
}

// checkFlowVocabulary restricts a flow's kind, direction and frequency to their vocabularies.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlowVocabulary(path string, f model.Flow) {
	lead := []any{strings.TrimSpace(f.ID)}
	v.enumValue(path, f.Kind, enumSpec{allowedFlowKinds, "model.invalid_flow_kind", "flow %q uses unknown kind %q", lead})
	v.enumValue(path, f.Direction, enumSpec{allowedFlowDirection, "model.invalid_flow_direction", "flow %q uses unknown direction %q", lead})
	v.enumValue(path, f.Frequency, enumSpec{allowedFlowFrequency, "model.invalid_flow_frequency", "flow %q uses unknown frequency %q", lead})
}

// checkFlowRefs requires a flow's source, destination, data objects and threats to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlowRefs(path string, f model.Flow) {
	lead := []any{strings.TrimSpace(f.ID)}
	v.optionalRef(path, f.SourceRef, refSpec{ok: v.validID, code: "model.invalid_flow_source_ref", format: "flow %q references unknown source %q", lead: lead})
	v.optionalRef(path, f.DestinationRef, refSpec{ok: v.validID, code: "model.invalid_flow_destination_ref", format: "flow %q references unknown destination %q", lead: lead})
	v.refList(path, f.DataRefs, refSpec{ok: inSet(v.ids.dataObjects), code: "model.invalid_flow_data_ref", format: "flow %q references unknown data object %q", field: "dataRefs", lead: lead})
	v.refList(path, f.Threats, refSpec{ok: inSet(v.ids.threatScenarios), code: "model.invalid_flow_threat_ref", format: "flow %q references unknown threat scenario %q", field: "threats", lead: lead})
}

// checkFlowSteps validates the steps of a flow that have a unique id and returns
// the path of each such step by id.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlowSteps(path string, f model.Flow) map[string]string {
	stepOwner := map[string]string{}
	for j, s := range f.Steps {
		stepPath := indexedPath(path+".steps", j)
		stepID := strings.TrimSpace(s.ID)
		if stepID == "" {
			v.fail("model.missing_flow_step_id", stepPath, fmt.Sprintf("flow %q has a step with missing id", strings.TrimSpace(f.ID)))
			continue
		}
		if prev, ok := stepOwner[stepID]; ok {
			v.fail("model.duplicate_flow_step_id", stepPath, fmt.Sprintf("flow %q has duplicate step id %q (%s, %s)", strings.TrimSpace(f.ID), stepID, prev, stepPath))
			continue
		}
		stepOwner[stepID] = stepPath
		v.checkFlowStep(stepPath, stepID, s)
	}
	return stepOwner
}

// checkFlowStep validates one flow step.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlowStep(stepPath, stepID string, s model.FlowStep) {
	v.checkFlowStepVocabulary(stepPath, stepID, s)
	v.checkFlowStepEnds(stepPath, stepID, s)
	v.checkFlowStepLinks(stepPath, stepID, s)
}

// checkFlowStepVocabulary restricts a step's kind, type, direction and frequency to their vocabularies.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlowStepVocabulary(stepPath, stepID string, s model.FlowStep) {
	lead := []any{stepID}
	if kind := strings.TrimSpace(s.Kind); kind != "" && !allowedFlowStepKinds[kind] {
		v.fail("model.invalid_flow_step_kind", stepPath, fmt.Sprintf("flow step %q uses unknown kind %q", stepID, kind))
	}
	v.enumValue(stepPath, s.FlowType, enumSpec{allowedFlowTypes, "model.invalid_flow_step_type", "flow step %q uses unknown flowType %q", lead})
	v.enumValue(stepPath, s.Direction, enumSpec{allowedFlowDirection, "model.invalid_flow_step_direction", "flow step %q uses unknown direction %q", lead})
	v.enumValue(stepPath, s.Frequency, enumSpec{allowedFlowFrequency, "model.invalid_flow_step_frequency", "flow step %q uses unknown frequency %q", lead})
}

// checkFlowStepEnds requires a step to reference an authored id and its source and destination to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlowStepEnds(stepPath, stepID string, s model.FlowStep) {
	lead := []any{stepID}
	if ref := strings.TrimSpace(s.Ref); ref == "" {
		v.fail("model.missing_flow_step_ref", stepPath, fmt.Sprintf("flow step %q must reference an authored id", stepID))
	} else if !v.validID(ref) {
		v.fail("model.broken_flow_reference", stepPath, fmt.Sprintf("flow step %q references unknown id %q", stepID, ref))
	}
	v.optionalRef(stepPath, s.SourceRef, refSpec{ok: v.validID, code: "model.invalid_flow_step_source_ref", format: "flow step %q references unknown source %q", lead: lead})
	v.optionalRef(stepPath, s.DestinationRef, refSpec{ok: v.validID, code: "model.invalid_flow_step_destination_ref", format: "flow step %q references unknown destination %q", lead: lead})
}

// checkFlowStepLinks requires a step's interface, trust boundary and data objects to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkFlowStepLinks(stepPath, stepID string, s model.FlowStep) {
	lead := []any{stepID}
	v.optionalRef(stepPath, s.InterfaceRef, refSpec{ok: inSet(v.ids.interfaces), code: "model.invalid_flow_interface_ref", format: "flow step %q references unknown interface %q", lead: lead})
	v.optionalRef(stepPath, s.TrustBoundaryRef, refSpec{ok: inSet(v.ids.trustBoundaries), code: "model.invalid_flow_trust_boundary_ref", format: "flow step %q references unknown trust boundary %q", lead: lead})
	if s.BoundaryCrossing {
		v.requireText(stepPath, s.TrustBoundaryRef, textSpec{"model.missing_flow_boundary_ref", fmt.Sprintf("flow step %q marks boundaryCrossing but has no trustBoundaryRef", stepID)})
	}
	v.refList(stepPath, s.DataRefs, refSpec{ok: inSet(v.ids.dataObjects), code: "model.invalid_flow_data_ref", format: "flow step %q references unknown data object %q", field: "dataRefs", lead: lead})
}

// stepLinks checks that the step ids a flow's steps, entries and exits name exist.
type stepLinks struct {
	v      *bundleValidator
	flowID string
	owner  map[string]string
}

// check records an unknown step named by an edge of the given kind, ignoring a blank target.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (l stepLinks) check(target, path, edgeKind string) {
	target = strings.TrimSpace(target)
	if target == "" {
		return
	}
	if _, ok := l.owner[target]; !ok {
		l.v.fail("model.broken_flow_step_link", path, fmt.Sprintf("flow %q %s references unknown step %q", l.flowID, edgeKind, target))
	}
}

// checkStep checks the next and onError targets of a step.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (l stepLinks) checkStep(stepPath string, s model.FlowStep) {
	for _, next := range s.Next {
		l.check(next, stepPath, "next")
	}
	for _, onErr := range s.OnError {
		l.check(onErr, stepPath, "onError")
	}
}

// checkEntriesAndExits requires an entry step and checks the entry and exit targets.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (l stepLinks) checkEntriesAndExits(path string, f model.Flow) {
	if len(f.Entry) == 0 {
		l.v.fail("model.empty_flow_entry", path, fmt.Sprintf("flow %q must define at least one entry step", l.flowID))
	}
	for _, entry := range f.Entry {
		l.check(entry, path, "entry")
	}
	for _, exit := range f.Exits {
		l.check(exit, path, "exit")
	}
}
