// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

var allowedControlImplementationTypes = map[string]bool{
	"technical":  true,
	"procedural": true,
	"inherited":  true,
	"hybrid":     true,
}

var allowedControlAllocationStatuses = map[string]bool{
	"planned":     true,
	"partial":     true,
	"implemented": true,
	"inherited":   true,
}

var oscalControlIDRe = regexp.MustCompile(`(?i)^[a-z]{1,4}-\d+(\(\d+\))*$`)
var isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var allowedRiskStatus = map[string]bool{
	"open":       true,
	"mitigating": true,
	"accepted":   true,
	"closed":     true,
}

var allowedRiskLevel = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
}

var allowedRiskResponse = map[string]bool{
	"mitigate": true,
	"accept":   true,
	"transfer": true,
	"avoid":    true,
}

var allowedPOAMStatus = map[string]bool{
	"planned":     true,
	"in-progress": true,
	"completed":   true,
	"deferred":    true,
}

var allowedViewKinds = map[string]bool{
	"architecture-intent": true,
	"communication":       true,
	"deployment":          true,
	"security":            true,
	"traceability":        true,
	"state-lifecycle":     true,
	"interaction-flow":    true,
}

var allowedMappingTypes = map[string]bool{
	"contains":       true,
	"depends_on":     true,
	"interacts_with": true,
	"targets":        true,
	"calls":          true,
	"publishes":      true,
	"subscribes":     true,
	"reads":          true,
	"writes":         true,
	"streams":        true,
	"implements":     true,
	"satisfies":      true,
	"allocated_to":   true,
	"verified_by":    true,
	"transitions_to": true,
	"triggered_by":   true,
	"guarded_by":     true,
	"deployed_to":    true,
	"mitigated_by":   true,
	"bounded_by":     true,
}

var allowedViewMappingTypes = map[string]bool{
	"contains":       true,
	"depends_on":     true,
	"interacts_with": true,
	"targets":        true,
	"calls":          true,
	"publishes":      true,
	"subscribes":     true,
	"reads":          true,
	"writes":         true,
	"streams":        true,
	"implements":     true,
	"satisfies":      true,
	"allocated_to":   true,
	"verified_by":    true,
	"transitions_to": true,
	"triggered_by":   true,
	"guarded_by":     true,
	"deployed_to":    true,
	"mitigated_by":   true,
	"bounded_by":     true,
	"flow_next":      true,
	"flow_error":     true,
	"flow_async":     true,
	"flow_ref":       true,
}

var allowedViewEntityKinds = map[string]bool{
	"functional_group":     true,
	"functional_unit":      true,
	"actor":                true,
	"attack_vector":        true,
	"referenced_element":   true,
	"interface":            true,
	"data_object":          true,
	"deployment_target":    true,
	"control":              true,
	"trust_boundary":       true,
	"state":                true,
	"event":                true,
	"flow":                 true,
	"flow_step":            true,
	"threat_scenario":      true,
	"threat_assumption":    true,
	"threat_out_of_scope":  true,
	"threat_mitigation":    true,
	"control_verification": true,
	"risk":                 true,
	"poam_item":            true,
}

var allowedFlowStepKinds = map[string]bool{
	"user_action":          true,
	"system_action":        true,
	"data_move":            true,
	"decision":             true,
	"external_interaction": true,
	"security_check":       true,
	"control_action":       true,
	"deployment_action":    true,
	"notification":         true,
	"integration_call":     true,
}

var allowedFlowKinds = map[string]bool{
	"interaction": true,
	"data":        true,
	"control":     true,
	"deployment":  true,
	"state":       true,
	"event":       true,
	"security":    true,
	"business":    true,
}

var allowedFlowTypes = map[string]bool{
	"interaction":  true,
	"data":         true,
	"control":      true,
	"deployment":   true,
	"state":        true,
	"event":        true,
	"security":     true,
	"notification": true,
}

var allowedThreatScenarioStatus = map[string]bool{
	"identified": true,
	"triaged":    true,
	"mitigating": true,
	"accepted":   true,
	"resolved":   true,
}

var allowedThreatAssumptionStatus = map[string]bool{
	"draft":    true,
	"accepted": true,
	"rejected": true,
	"expired":  true,
}

var allowedThreatOutOfScopeStatus = map[string]bool{
	"proposed": true,
	"approved": true,
	"expired":  true,
}

var allowedThreatMitigationStatus = map[string]bool{
	"planned":     true,
	"partial":     true,
	"implemented": true,
	"verified":    true,
	"deferred":    true,
}

var allowedThreatMitigationEffectiveness = map[string]bool{
	"low":     true,
	"medium":  true,
	"high":    true,
	"unknown": true,
}

var allowedControlVerificationMethod = map[string]bool{
	"test":       true,
	"analysis":   true,
	"inspection": true,
	"exercise":   true,
	"audit":      true,
	"monitoring": true,
}

var allowedControlVerificationStatus = map[string]bool{
	"not-run": true,
	"pass":    true,
	"fail":    true,
	"partial": true,
	"blocked": true,
}

var allowedFlowDirection = map[string]bool{
	"inbound":       true,
	"outbound":      true,
	"bidirectional": true,
	"internal":      true,
}

var allowedFlowFrequency = map[string]bool{
	"realtime":  true,
	"batch":     true,
	"scheduled": true,
	"on-demand": true,
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func Bundle(b model.Bundle) []Diagnostic {
	diags := []Diagnostic{}
	idOwner := map[string]string{}

	addID := func(id, owner string) {
		if strings.TrimSpace(id) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_id", Severity: SeverityError, Message: fmt.Sprintf("missing id for %s", owner), Path: owner})
			return
		}
		if prev, ok := idOwner[id]; ok {
			diags = append(diags, Diagnostic{Code: "model.duplicate_id", Severity: SeverityError, Message: fmt.Sprintf("duplicate id %q (%s, %s)", id, prev, owner), Path: owner})
			return
		}
		idOwner[id] = owner
	}

	groups := map[string]bool{}
	units := map[string]bool{}
	actors := map[string]bool{}
	vectors := map[string]bool{}
	references := map[string]bool{}
	interfaces := map[string]bool{}
	dataObjects := map[string]bool{}
	deploymentTargets := map[string]bool{}
	controls := map[string]bool{}
	trustBoundaries := map[string]bool{}
	states := map[string]bool{}
	events := map[string]bool{}
	risks := map[string]bool{}
	threatScenarios := map[string]bool{}
	threatAssumptions := map[string]bool{}
	threatOutOfScope := map[string]bool{}
	threatMitigations := map[string]bool{}
	controlVerifications := map[string]bool{}
	kindByID := map[string]string{}

	for i, g := range b.Architecture.AuthoredArchitecture.FunctionalGroups {
		addID(g.ID, fmt.Sprintf("authoredArchitecture.functionalGroups[%d]", i))
		groups[g.ID] = true
		kindByID[g.ID] = "functional_group"
	}
	for i, u := range b.Architecture.AuthoredArchitecture.FunctionalUnits {
		path := fmt.Sprintf("authoredArchitecture.functionalUnits[%d]", i)
		addID(u.ID, path)
		units[u.ID] = true
		kindByID[u.ID] = "functional_unit"
		if strings.TrimSpace(u.Group) == "" || !groups[u.Group] {
			diags = append(diags, Diagnostic{Code: "model.invalid_group", Severity: SeverityError, Message: fmt.Sprintf("functional unit %q must reference a valid functional group", u.ID), Path: path})
		}
	}
	for i, a := range b.Architecture.AuthoredArchitecture.Actors {
		addID(a.ID, fmt.Sprintf("authoredArchitecture.actors[%d]", i))
		actors[a.ID] = true
		kindByID[a.ID] = "actor"
	}
	for i, a := range b.Architecture.AuthoredArchitecture.AttackVectors {
		addID(a.ID, fmt.Sprintf("authoredArchitecture.attackVectors[%d]", i))
		vectors[a.ID] = true
		kindByID[a.ID] = "attack_vector"
	}
	for i, r := range b.Architecture.AuthoredArchitecture.ReferencedElements {
		addID(r.ID, fmt.Sprintf("authoredArchitecture.referencedElements[%d]", i))
		references[r.ID] = true
		kindByID[r.ID] = "referenced_element"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.Interfaces {
		path := fmt.Sprintf("authoredArchitecture.interfaces[%d]", i)
		addID(x.ID, path)
		interfaces[x.ID] = true
		kindByID[x.ID] = "interface"
		if owner := strings.TrimSpace(x.Owner); owner != "" && !units[owner] {
			diags = append(diags, Diagnostic{Code: "model.invalid_interface_owner", Severity: SeverityError, Message: fmt.Sprintf("interface %q owner %q must be a functional unit", x.ID, owner), Path: path})
		}
	}
	for i, x := range b.Architecture.AuthoredArchitecture.DataObjects {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.dataObjects[%d]", i))
		dataObjects[x.ID] = true
		kindByID[x.ID] = "data_object"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.DeploymentTargets {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.deploymentTargets[%d]", i))
		deploymentTargets[x.ID] = true
		kindByID[x.ID] = "deployment_target"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.Controls {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.controls[%d]", i))
		controls[x.ID] = true
		kindByID[x.ID] = "control"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.ControlAllocations {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.controlAllocations[%d]", i))
		kindByID[x.ID] = "control_allocation"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.Risks {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.risks[%d]", i))
		risks[x.ID] = true
		kindByID[x.ID] = "risk"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.POAMItems {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.poamItems[%d]", i))
		kindByID[x.ID] = "poam_item"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.TrustBoundaries {
		path := fmt.Sprintf("authoredArchitecture.trustBoundaries[%d]", i)
		addID(x.ID, path)
		trustBoundaries[x.ID] = true
		kindByID[x.ID] = "trust_boundary"
		if parent := strings.TrimSpace(x.ParentRef); parent == strings.TrimSpace(x.ID) && parent != "" {
			diags = append(diags, Diagnostic{Code: "model.invalid_trust_boundary_parent", Severity: SeverityError, Message: fmt.Sprintf("trust boundary %q cannot parent itself", x.ID), Path: path})
		}
	}
	for i, x := range b.Architecture.AuthoredArchitecture.States {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.states[%d]", i))
		states[x.ID] = true
		kindByID[x.ID] = "state"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.Events {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.events[%d]", i))
		events[x.ID] = true
		kindByID[x.ID] = "event"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.Flows {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.flows[%d]", i))
		kindByID[x.ID] = "flow"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.ThreatScenarios {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.threatScenarios[%d]", i))
		threatScenarios[x.ID] = true
		kindByID[x.ID] = "threat_scenario"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.ThreatAssumptions {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.threatAssumptions[%d]", i))
		threatAssumptions[x.ID] = true
		kindByID[x.ID] = "threat_assumption"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.ThreatOutOfScope {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.threatOutOfScope[%d]", i))
		threatOutOfScope[x.ID] = true
		kindByID[x.ID] = "threat_out_of_scope"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.ThreatMitigations {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.threatMitigations[%d]", i))
		threatMitigations[x.ID] = true
		kindByID[x.ID] = "threat_mitigation"
	}
	for i, x := range b.Architecture.AuthoredArchitecture.ControlVerifications {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.controlVerifications[%d]", i))
		controlVerifications[x.ID] = true
		kindByID[x.ID] = "control_verification"
	}

	validID := func(id string) bool {
		_, ok := idOwner[id]
		return ok
	}

	for i, boundary := range b.Architecture.AuthoredArchitecture.TrustBoundaries {
		path := fmt.Sprintf("authoredArchitecture.trustBoundaries[%d]", i)
		if parent := strings.TrimSpace(boundary.ParentRef); parent != "" {
			if !trustBoundaries[parent] {
				diags = append(diags, Diagnostic{Code: "model.invalid_trust_boundary_parent", Severity: SeverityError, Message: fmt.Sprintf("unknown trust boundary parent %q", parent), Path: path})
			}
		}
		for j, member := range boundary.Members {
			member = strings.TrimSpace(member)
			if member == "" || !validID(member) {
				diags = append(diags, Diagnostic{Code: "model.invalid_trust_boundary_member", Severity: SeverityError, Message: fmt.Sprintf("unknown trust boundary member %q", member), Path: fmt.Sprintf("%s.members[%d]", path, j)})
			}
		}
	}

	for i, m := range b.Architecture.AuthoredArchitecture.Mappings {
		path := fmt.Sprintf("authoredArchitecture.mappings[%d]", i)
		if !allowedMappingTypes[m.Type] {
			diags = append(diags, Diagnostic{Code: "model.unknown_mapping_type", Severity: SeverityError, Message: fmt.Sprintf("unknown mapping type %q", m.Type), Path: path})
		}
		if !validID(m.From) {
			diags = append(diags, Diagnostic{Code: "model.broken_reference", Severity: SeverityError, Message: fmt.Sprintf("mapping from %q does not exist", m.From), Path: path})
		}
		if !validID(m.To) {
			diags = append(diags, Diagnostic{Code: "model.broken_reference", Severity: SeverityError, Message: fmt.Sprintf("mapping to %q does not exist", m.To), Path: path})
		}
		if strings.HasPrefix(strings.TrimSpace(m.From), "RT-") || strings.HasPrefix(strings.TrimSpace(m.From), "CODE-") || strings.HasPrefix(strings.TrimSpace(m.To), "RT-") || strings.HasPrefix(strings.TrimSpace(m.To), "CODE-") {
			diags = append(diags, Diagnostic{Code: "model.inferred_id_not_allowed", Severity: SeverityError, Message: "authored mappings must not reference inferred RT-* or CODE-* ids", Path: path})
		}
		if m.Type == "interacts_with" && !(actors[m.From] && units[m.To]) {
			diags = append(diags, Diagnostic{Code: "model.invalid_interaction", Severity: SeverityError, Message: "interacts_with must be actor -> functional unit", Path: path})
		}
		if m.Type == "targets" && !vectors[m.From] {
			diags = append(diags, Diagnostic{Code: "model.invalid_target", Severity: SeverityError, Message: "targets must originate from an attack vector", Path: path})
		}
		if m.Type == "triggered_by" && !events[m.To] {
			diags = append(diags, Diagnostic{Code: "model.invalid_trigger", Severity: SeverityError, Message: "triggered_by must target an event", Path: path})
		}
		if m.Type == "guarded_by" && !(controls[m.To] || trustBoundaries[m.To] || references[m.To]) {
			diags = append(diags, Diagnostic{Code: "model.invalid_guard", Severity: SeverityError, Message: "guarded_by must target a control, trust boundary, or referenced element", Path: path})
		}
		if m.Type == "transitions_to" && !(states[m.From] && states[m.To]) {
			diags = append(diags, Diagnostic{Code: "model.invalid_transition", Severity: SeverityError, Message: "transitions_to must be state -> state", Path: path})
		}
		if fromKind, toKind := kindByID[m.From], kindByID[m.To]; fromKind != "" && toKind != "" && !mappingPairAllowed(m.Type, fromKind, toKind) {
			diags = append(diags, Diagnostic{Code: "model.invalid_mapping_pair", Severity: SeverityError, Message: fmt.Sprintf("mapping type %q is not valid for %s -> %s", m.Type, fromKind, toKind), Path: path})
		}
	}

	for i, a := range b.Architecture.AuthoredArchitecture.ControlAllocations {
		path := fmt.Sprintf("authoredArchitecture.controlAllocations[%d]", i)
		if controlRef := strings.TrimSpace(a.ControlRef); controlRef == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_control_ref", Severity: SeverityError, Message: "control allocation must set controlRef", Path: path})
		} else if !controls[controlRef] {
			diags = append(diags, Diagnostic{Code: "model.invalid_control_ref", Severity: SeverityError, Message: fmt.Sprintf("control allocation references unknown control %q", controlRef), Path: path})
		}
		if len(a.OSCALControlIDs) == 0 {
			diags = append(diags, Diagnostic{Code: "model.empty_oscal_control_ids", Severity: SeverityError, Message: "control allocation must include at least one oscalControlIds entry", Path: path})
		}
		for j, cid := range a.OSCALControlIDs {
			cid = strings.TrimSpace(cid)
			if cid == "" || !oscalControlIDRe.MatchString(cid) {
				diags = append(diags, Diagnostic{Code: "model.invalid_oscal_control_id", Severity: SeverityError, Message: fmt.Sprintf("invalid OSCAL control id %q", cid), Path: fmt.Sprintf("%s.oscalControlIds[%d]", path, j)})
			}
		}
		if len(a.AppliesTo) == 0 {
			diags = append(diags, Diagnostic{Code: "model.empty_control_allocation_scope", Severity: SeverityError, Message: "control allocation must include at least one appliesTo id", Path: path})
		}
		for j, target := range a.AppliesTo {
			target = strings.TrimSpace(target)
			if target == "" || !validID(target) {
				diags = append(diags, Diagnostic{Code: "model.invalid_control_allocation_target", Severity: SeverityError, Message: fmt.Sprintf("unknown appliesTo id %q", target), Path: fmt.Sprintf("%s.appliesTo[%d]", path, j)})
			}
		}
		if it := strings.ToLower(strings.TrimSpace(a.ImplementationType)); it != "" && !allowedControlImplementationTypes[it] {
			diags = append(diags, Diagnostic{Code: "model.invalid_control_implementation_type", Severity: SeverityError, Message: fmt.Sprintf("unknown implementationType %q", a.ImplementationType), Path: path})
		}
		if st := strings.ToLower(strings.TrimSpace(a.Status)); st != "" && !allowedControlAllocationStatuses[st] {
			diags = append(diags, Diagnostic{Code: "model.invalid_control_allocation_status", Severity: SeverityError, Message: fmt.Sprintf("unknown status %q", a.Status), Path: path})
		}
		for j, role := range a.ResponsibleRoles {
			role = strings.TrimSpace(role)
			if role == "" || !actors[role] {
				diags = append(diags, Diagnostic{Code: "model.invalid_control_responsible_role", Severity: SeverityError, Message: fmt.Sprintf("unknown responsible role %q", role), Path: fmt.Sprintf("%s.responsibleRoles[%d]", path, j)})
			}
		}
		for j, ev := range a.Evidence {
			if strings.TrimSpace(ev.Path) == "" {
				diags = append(diags, Diagnostic{Code: "model.empty_control_evidence_path", Severity: SeverityError, Message: "control allocation evidence path is required", Path: fmt.Sprintf("%s.evidence[%d]", path, j)})
			}
		}
	}

	for i, r := range b.Architecture.AuthoredArchitecture.Risks {
		path := fmt.Sprintf("authoredArchitecture.risks[%d]", i)
		if strings.TrimSpace(r.Title) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_risk_title", Severity: SeverityError, Message: "risk title is required", Path: path})
		}
		if strings.TrimSpace(r.Statement) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_risk_statement", Severity: SeverityError, Message: "risk statement is required", Path: path})
		}
		if st := strings.ToLower(strings.TrimSpace(r.Status)); st != "" && !allowedRiskStatus[st] {
			diags = append(diags, Diagnostic{Code: "model.invalid_risk_status", Severity: SeverityError, Message: fmt.Sprintf("unknown risk status %q", r.Status), Path: path})
		}
		if lk := strings.ToLower(strings.TrimSpace(r.Likelihood)); lk != "" && !allowedRiskLevel[lk] {
			diags = append(diags, Diagnostic{Code: "model.invalid_risk_likelihood", Severity: SeverityError, Message: fmt.Sprintf("unknown likelihood %q", r.Likelihood), Path: path})
		}
		if im := strings.ToLower(strings.TrimSpace(r.Impact)); im != "" && !allowedRiskLevel[im] {
			diags = append(diags, Diagnostic{Code: "model.invalid_risk_impact", Severity: SeverityError, Message: fmt.Sprintf("unknown impact %q", r.Impact), Path: path})
		}
		if rr := strings.ToLower(strings.TrimSpace(r.ResidualRisk)); rr != "" && !allowedRiskLevel[rr] {
			diags = append(diags, Diagnostic{Code: "model.invalid_residual_risk", Severity: SeverityError, Message: fmt.Sprintf("unknown residualRisk %q", r.ResidualRisk), Path: path})
		}
		if rsp := strings.ToLower(strings.TrimSpace(r.Response)); rsp != "" && !allowedRiskResponse[rsp] {
			diags = append(diags, Diagnostic{Code: "model.invalid_risk_response", Severity: SeverityError, Message: fmt.Sprintf("unknown response %q", r.Response), Path: path})
		}
		if owner := strings.TrimSpace(r.Owner); owner != "" && !actors[owner] {
			diags = append(diags, Diagnostic{Code: "model.invalid_risk_owner", Severity: SeverityError, Message: fmt.Sprintf("unknown risk owner %q", owner), Path: path})
		}
		for j, id := range r.AppliesTo {
			id = strings.TrimSpace(id)
			if id == "" || !validID(id) {
				diags = append(diags, Diagnostic{Code: "model.invalid_risk_scope_target", Severity: SeverityError, Message: fmt.Sprintf("unknown appliesTo id %q", id), Path: fmt.Sprintf("%s.appliesTo[%d]", path, j)})
			}
		}
		for j, id := range r.RelatedControls {
			id = strings.TrimSpace(id)
			if id == "" || !controls[id] {
				diags = append(diags, Diagnostic{Code: "model.invalid_risk_control_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown related control %q", id), Path: fmt.Sprintf("%s.relatedControls[%d]", path, j)})
			}
		}
		for j, id := range r.AttackVectors {
			id = strings.TrimSpace(id)
			if id == "" || !vectors[id] {
				diags = append(diags, Diagnostic{Code: "model.invalid_risk_attack_vector_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown attack vector %q", id), Path: fmt.Sprintf("%s.attackVectors[%d]", path, j)})
			}
		}
		for j, id := range r.ThreatScenarios {
			id = strings.TrimSpace(id)
			if id == "" || !threatScenarios[id] {
				diags = append(diags, Diagnostic{Code: "model.invalid_risk_threat_scenario_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown threat scenario %q", id), Path: fmt.Sprintf("%s.threatScenarios[%d]", path, j)})
			}
		}
		for j, ev := range r.Evidence {
			if strings.TrimSpace(ev.Path) == "" {
				diags = append(diags, Diagnostic{Code: "model.empty_risk_evidence_path", Severity: SeverityError, Message: "risk evidence path is required", Path: fmt.Sprintf("%s.evidence[%d]", path, j)})
			}
		}
	}

	for i, p := range b.Architecture.AuthoredArchitecture.POAMItems {
		path := fmt.Sprintf("authoredArchitecture.poamItems[%d]", i)
		if riskRef := strings.TrimSpace(p.RiskRef); riskRef == "" || !risks[riskRef] {
			diags = append(diags, Diagnostic{Code: "model.invalid_poam_risk_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown poam riskRef %q", p.RiskRef), Path: path})
		}
		if strings.TrimSpace(p.Milestone) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_poam_milestone", Severity: SeverityError, Message: "poam milestone is required", Path: path})
		}
		if due := strings.TrimSpace(p.DueDate); due != "" && !isoDateRe.MatchString(due) {
			diags = append(diags, Diagnostic{Code: "model.invalid_poam_due_date", Severity: SeverityError, Message: fmt.Sprintf("invalid dueDate %q, expected YYYY-MM-DD", p.DueDate), Path: path})
		}
		if st := strings.ToLower(strings.TrimSpace(p.Status)); st != "" && !allowedPOAMStatus[st] {
			diags = append(diags, Diagnostic{Code: "model.invalid_poam_status", Severity: SeverityError, Message: fmt.Sprintf("unknown poam status %q", p.Status), Path: path})
		}
		if role := strings.TrimSpace(p.ResponsibleRole); role != "" && !actors[role] {
			diags = append(diags, Diagnostic{Code: "model.invalid_poam_responsible_role", Severity: SeverityError, Message: fmt.Sprintf("unknown poam responsibleRole %q", role), Path: path})
		}
		for j, art := range p.Artifacts {
			if strings.TrimSpace(art.Path) == "" {
				diags = append(diags, Diagnostic{Code: "model.empty_poam_artifact_path", Severity: SeverityError, Message: "poam artifact path is required", Path: fmt.Sprintf("%s.artifacts[%d]", path, j)})
			}
		}
	}

	for i, f := range b.Architecture.AuthoredArchitecture.Flows {
		flowPath := fmt.Sprintf("authoredArchitecture.flows[%d]", i)
		if kind := strings.ToLower(strings.TrimSpace(f.Kind)); kind != "" && !allowedFlowKinds[kind] {
			diags = append(diags, Diagnostic{Code: "model.invalid_flow_kind", Severity: SeverityError, Message: fmt.Sprintf("flow %q uses unknown kind %q", strings.TrimSpace(f.ID), f.Kind), Path: flowPath})
		}
		if direction := strings.ToLower(strings.TrimSpace(f.Direction)); direction != "" && !allowedFlowDirection[direction] {
			diags = append(diags, Diagnostic{Code: "model.invalid_flow_direction", Severity: SeverityError, Message: fmt.Sprintf("flow %q uses unknown direction %q", strings.TrimSpace(f.ID), f.Direction), Path: flowPath})
		}
		if frequency := strings.ToLower(strings.TrimSpace(f.Frequency)); frequency != "" && !allowedFlowFrequency[frequency] {
			diags = append(diags, Diagnostic{Code: "model.invalid_flow_frequency", Severity: SeverityError, Message: fmt.Sprintf("flow %q uses unknown frequency %q", strings.TrimSpace(f.ID), f.Frequency), Path: flowPath})
		}
		if source := strings.TrimSpace(f.SourceRef); source != "" && !validID(source) {
			diags = append(diags, Diagnostic{Code: "model.invalid_flow_source_ref", Severity: SeverityError, Message: fmt.Sprintf("flow %q references unknown source %q", strings.TrimSpace(f.ID), source), Path: flowPath})
		}
		if destination := strings.TrimSpace(f.DestinationRef); destination != "" && !validID(destination) {
			diags = append(diags, Diagnostic{Code: "model.invalid_flow_destination_ref", Severity: SeverityError, Message: fmt.Sprintf("flow %q references unknown destination %q", strings.TrimSpace(f.ID), destination), Path: flowPath})
		}
		for j, dataRef := range f.DataRefs {
			dataRef = strings.TrimSpace(dataRef)
			if dataRef == "" || !dataObjects[dataRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_data_ref", Severity: SeverityError, Message: fmt.Sprintf("flow %q references unknown data object %q", strings.TrimSpace(f.ID), dataRef), Path: fmt.Sprintf("%s.dataRefs[%d]", flowPath, j)})
			}
		}
		for j, threatID := range f.Threats {
			threatID = strings.TrimSpace(threatID)
			if threatID == "" || !threatScenarios[threatID] {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_threat_ref", Severity: SeverityError, Message: fmt.Sprintf("flow %q references unknown threat scenario %q", strings.TrimSpace(f.ID), threatID), Path: fmt.Sprintf("%s.threats[%d]", flowPath, j)})
			}
		}
		stepOwner := map[string]string{}
		for j, s := range f.Steps {
			stepPath := fmt.Sprintf("%s.steps[%d]", flowPath, j)
			stepID := strings.TrimSpace(s.ID)
			if stepID == "" {
				diags = append(diags, Diagnostic{Code: "model.missing_flow_step_id", Severity: SeverityError, Message: fmt.Sprintf("flow %q has a step with missing id", strings.TrimSpace(f.ID)), Path: stepPath})
				continue
			}
			if prev, ok := stepOwner[stepID]; ok {
				diags = append(diags, Diagnostic{Code: "model.duplicate_flow_step_id", Severity: SeverityError, Message: fmt.Sprintf("flow %q has duplicate step id %q (%s, %s)", strings.TrimSpace(f.ID), stepID, prev, stepPath), Path: stepPath})
				continue
			}
			stepOwner[stepID] = stepPath
			if kind := strings.TrimSpace(s.Kind); kind != "" && !allowedFlowStepKinds[kind] {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_step_kind", Severity: SeverityError, Message: fmt.Sprintf("flow step %q uses unknown kind %q", stepID, kind), Path: stepPath})
			}
			if flowType := strings.ToLower(strings.TrimSpace(s.FlowType)); flowType != "" && !allowedFlowTypes[flowType] {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_step_type", Severity: SeverityError, Message: fmt.Sprintf("flow step %q uses unknown flowType %q", stepID, s.FlowType), Path: stepPath})
			}
			if direction := strings.ToLower(strings.TrimSpace(s.Direction)); direction != "" && !allowedFlowDirection[direction] {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_step_direction", Severity: SeverityError, Message: fmt.Sprintf("flow step %q uses unknown direction %q", stepID, s.Direction), Path: stepPath})
			}
			if frequency := strings.ToLower(strings.TrimSpace(s.Frequency)); frequency != "" && !allowedFlowFrequency[frequency] {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_step_frequency", Severity: SeverityError, Message: fmt.Sprintf("flow step %q uses unknown frequency %q", stepID, s.Frequency), Path: stepPath})
			}
			if ref := strings.TrimSpace(s.Ref); ref == "" {
				diags = append(diags, Diagnostic{Code: "model.missing_flow_step_ref", Severity: SeverityError, Message: fmt.Sprintf("flow step %q must reference an authored id", stepID), Path: stepPath})
			} else if !validID(ref) {
				diags = append(diags, Diagnostic{Code: "model.broken_flow_reference", Severity: SeverityError, Message: fmt.Sprintf("flow step %q references unknown id %q", stepID, ref), Path: stepPath})
			}
			if source := strings.TrimSpace(s.SourceRef); source != "" && !validID(source) {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_step_source_ref", Severity: SeverityError, Message: fmt.Sprintf("flow step %q references unknown source %q", stepID, source), Path: stepPath})
			}
			if destination := strings.TrimSpace(s.DestinationRef); destination != "" && !validID(destination) {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_step_destination_ref", Severity: SeverityError, Message: fmt.Sprintf("flow step %q references unknown destination %q", stepID, destination), Path: stepPath})
			}
			if interfaceRef := strings.TrimSpace(s.InterfaceRef); interfaceRef != "" && !interfaces[interfaceRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_interface_ref", Severity: SeverityError, Message: fmt.Sprintf("flow step %q references unknown interface %q", stepID, interfaceRef), Path: stepPath})
			}
			if trustBoundaryRef := strings.TrimSpace(s.TrustBoundaryRef); trustBoundaryRef != "" && !trustBoundaries[trustBoundaryRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_flow_trust_boundary_ref", Severity: SeverityError, Message: fmt.Sprintf("flow step %q references unknown trust boundary %q", stepID, trustBoundaryRef), Path: stepPath})
			}
			if s.BoundaryCrossing && strings.TrimSpace(s.TrustBoundaryRef) == "" {
				diags = append(diags, Diagnostic{Code: "model.missing_flow_boundary_ref", Severity: SeverityError, Message: fmt.Sprintf("flow step %q marks boundaryCrossing but has no trustBoundaryRef", stepID), Path: stepPath})
			}
			for k, dataRef := range s.DataRefs {
				dataRef = strings.TrimSpace(dataRef)
				if dataRef == "" || !dataObjects[dataRef] {
					diags = append(diags, Diagnostic{Code: "model.invalid_flow_data_ref", Severity: SeverityError, Message: fmt.Sprintf("flow step %q references unknown data object %q", stepID, dataRef), Path: fmt.Sprintf("%s.dataRefs[%d]", stepPath, k)})
				}
			}
		}
		if len(stepOwner) == 0 {
			diags = append(diags, Diagnostic{Code: "model.empty_flow_steps", Severity: SeverityError, Message: fmt.Sprintf("flow %q must include at least one step", strings.TrimSpace(f.ID)), Path: flowPath})
			continue
		}
		validateStepTarget := func(target, path, edgeKind string) {
			target = strings.TrimSpace(target)
			if target == "" {
				return
			}
			if _, ok := stepOwner[target]; !ok {
				diags = append(diags, Diagnostic{Code: "model.broken_flow_step_link", Severity: SeverityError, Message: fmt.Sprintf("flow %q %s references unknown step %q", strings.TrimSpace(f.ID), edgeKind, target), Path: path})
			}
		}
		for j, s := range f.Steps {
			stepPath := fmt.Sprintf("%s.steps[%d]", flowPath, j)
			for _, next := range s.Next {
				validateStepTarget(next, stepPath, "next")
			}
			for _, onErr := range s.OnError {
				validateStepTarget(onErr, stepPath, "onError")
			}
		}
		if len(f.Entry) == 0 {
			diags = append(diags, Diagnostic{Code: "model.empty_flow_entry", Severity: SeverityError, Message: fmt.Sprintf("flow %q must define at least one entry step", strings.TrimSpace(f.ID)), Path: flowPath})
		} else {
			for _, entry := range f.Entry {
				validateStepTarget(entry, flowPath, "entry")
			}
		}
		if len(f.Exits) > 0 {
			for _, exit := range f.Exits {
				validateStepTarget(exit, flowPath, "exit")
			}
		}
	}

	for i, ts := range b.Architecture.AuthoredArchitecture.ThreatScenarios {
		path := fmt.Sprintf("authoredArchitecture.threatScenarios[%d]", i)
		if strings.TrimSpace(ts.Title) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_threat_scenario_title", Severity: SeverityError, Message: "threat scenario title is required", Path: path})
		}
		if av := strings.TrimSpace(ts.AttackVectorRef); av != "" && !vectors[av] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_attack_vector", Severity: SeverityError, Message: fmt.Sprintf("threat scenario references unknown attack vector %q", av), Path: path})
		}
		if st := strings.ToLower(strings.TrimSpace(ts.Status)); st != "" && !allowedThreatScenarioStatus[st] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_status", Severity: SeverityError, Message: fmt.Sprintf("unknown threat scenario status %q", ts.Status), Path: path})
		}
		if owner := strings.TrimSpace(ts.Owner); owner != "" && !actors[owner] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_owner", Severity: SeverityError, Message: fmt.Sprintf("unknown threat scenario owner %q", owner), Path: path})
		}
		if rr := strings.TrimSpace(ts.RiskRef); rr != "" && !risks[rr] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_risk_ref", Severity: SeverityError, Message: fmt.Sprintf("threat scenario references unknown risk %q", rr), Path: path})
		}
		if lk := strings.ToLower(strings.TrimSpace(ts.Likelihood)); lk != "" && !allowedRiskLevel[lk] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_likelihood", Severity: SeverityError, Message: fmt.Sprintf("unknown likelihood %q", ts.Likelihood), Path: path})
		}
		if im := strings.ToLower(strings.TrimSpace(ts.Impact)); im != "" && !allowedRiskLevel[im] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_impact", Severity: SeverityError, Message: fmt.Sprintf("unknown impact %q", ts.Impact), Path: path})
		}
		if sv := strings.ToLower(strings.TrimSpace(ts.Severity)); sv != "" && !allowedRiskLevel[sv] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_severity", Severity: SeverityError, Message: fmt.Sprintf("unknown severity %q", ts.Severity), Path: path})
		}
		for j, target := range ts.AppliesTo {
			target = strings.TrimSpace(target)
			if target == "" || !validID(target) {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_scope_target", Severity: SeverityError, Message: fmt.Sprintf("unknown appliesTo id %q", target), Path: fmt.Sprintf("%s.appliesTo[%d]", path, j)})
			}
		}
		for j, flowRef := range ts.FlowRefs {
			flowRef = strings.TrimSpace(flowRef)
			if flowRef == "" || kindByID[flowRef] != "flow" {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_flow_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown flow ref %q", flowRef), Path: fmt.Sprintf("%s.flowRefs[%d]", path, j)})
			}
		}
		for j, controlRef := range ts.RelatedControls {
			controlRef = strings.TrimSpace(controlRef)
			if controlRef == "" || !controls[controlRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_control_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown related control %q", controlRef), Path: fmt.Sprintf("%s.relatedControls[%d]", path, j)})
			}
		}
		for j, assumptionRef := range ts.AssumptionRefs {
			assumptionRef = strings.TrimSpace(assumptionRef)
			if assumptionRef == "" || !threatAssumptions[assumptionRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_assumption_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown assumption ref %q", assumptionRef), Path: fmt.Sprintf("%s.assumptionRefs[%d]", path, j)})
			}
		}
		for j, outOfScopeRef := range ts.OutOfScopeRefs {
			outOfScopeRef = strings.TrimSpace(outOfScopeRef)
			if outOfScopeRef == "" || !threatOutOfScope[outOfScopeRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_out_of_scope_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown out-of-scope ref %q", outOfScopeRef), Path: fmt.Sprintf("%s.outOfScopeRefs[%d]", path, j)})
			}
		}
		for j, mitigationRef := range ts.MitigationRefs {
			mitigationRef = strings.TrimSpace(mitigationRef)
			if mitigationRef == "" || !threatMitigations[mitigationRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_mitigation_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown mitigation ref %q", mitigationRef), Path: fmt.Sprintf("%s.mitigationRefs[%d]", path, j)})
			}
		}
		for j, verificationRef := range ts.VerificationRefs {
			verificationRef = strings.TrimSpace(verificationRef)
			if verificationRef == "" || !controlVerifications[verificationRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_scenario_verification_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown verification ref %q", verificationRef), Path: fmt.Sprintf("%s.verificationRefs[%d]", path, j)})
			}
		}
		for j, ev := range ts.Evidence {
			if strings.TrimSpace(ev.Path) == "" {
				diags = append(diags, Diagnostic{Code: "model.empty_threat_scenario_evidence_path", Severity: SeverityError, Message: "threat scenario evidence path is required", Path: fmt.Sprintf("%s.evidence[%d]", path, j)})
			}
		}
		if len(ts.AppliesTo) == 0 && len(ts.FlowRefs) == 0 {
			diags = append(diags, Diagnostic{Code: "model.empty_threat_scenario_scope", Severity: SeverityError, Message: "threat scenario must include appliesTo or flowRefs", Path: path})
		}
	}

	for i, assumption := range b.Architecture.AuthoredArchitecture.ThreatAssumptions {
		path := fmt.Sprintf("authoredArchitecture.threatAssumptions[%d]", i)
		if strings.TrimSpace(assumption.Title) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_threat_assumption_title", Severity: SeverityError, Message: "threat assumption title is required", Path: path})
		}
		if strings.TrimSpace(assumption.Statement) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_threat_assumption_statement", Severity: SeverityError, Message: "threat assumption statement is required", Path: path})
		}
		if st := strings.ToLower(strings.TrimSpace(assumption.Status)); st != "" && !allowedThreatAssumptionStatus[st] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_assumption_status", Severity: SeverityError, Message: fmt.Sprintf("unknown threat assumption status %q", assumption.Status), Path: path})
		}
		if owner := strings.TrimSpace(assumption.Owner); owner != "" && !actors[owner] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_assumption_owner", Severity: SeverityError, Message: fmt.Sprintf("unknown threat assumption owner %q", assumption.Owner), Path: path})
		}
		for j, id := range assumption.AppliesTo {
			id = strings.TrimSpace(id)
			if id == "" || !validID(id) {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_assumption_scope_target", Severity: SeverityError, Message: fmt.Sprintf("unknown appliesTo id %q", id), Path: fmt.Sprintf("%s.appliesTo[%d]", path, j)})
			}
		}
		for j, ev := range assumption.Evidence {
			if strings.TrimSpace(ev.Path) == "" {
				diags = append(diags, Diagnostic{Code: "model.empty_threat_assumption_evidence_path", Severity: SeverityError, Message: "threat assumption evidence path is required", Path: fmt.Sprintf("%s.evidence[%d]", path, j)})
			}
		}
	}

	for i, out := range b.Architecture.AuthoredArchitecture.ThreatOutOfScope {
		path := fmt.Sprintf("authoredArchitecture.threatOutOfScope[%d]", i)
		if strings.TrimSpace(out.Title) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_threat_out_of_scope_title", Severity: SeverityError, Message: "threat out-of-scope title is required", Path: path})
		}
		if strings.TrimSpace(out.Reason) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_threat_out_of_scope_reason", Severity: SeverityError, Message: "threat out-of-scope reason is required", Path: path})
		}
		if st := strings.ToLower(strings.TrimSpace(out.Status)); st != "" && !allowedThreatOutOfScopeStatus[st] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_out_of_scope_status", Severity: SeverityError, Message: fmt.Sprintf("unknown threat out-of-scope status %q", out.Status), Path: path})
		}
		if owner := strings.TrimSpace(out.Owner); owner != "" && !actors[owner] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_out_of_scope_owner", Severity: SeverityError, Message: fmt.Sprintf("unknown threat out-of-scope owner %q", out.Owner), Path: path})
		}
		if expires := strings.TrimSpace(out.ExpiresOn); expires != "" && !isoDateRe.MatchString(expires) {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_out_of_scope_expiry", Severity: SeverityError, Message: fmt.Sprintf("invalid expiresOn %q, expected YYYY-MM-DD", out.ExpiresOn), Path: path})
		}
		for j, id := range out.AppliesTo {
			id = strings.TrimSpace(id)
			if id == "" || !validID(id) {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_out_of_scope_target", Severity: SeverityError, Message: fmt.Sprintf("unknown appliesTo id %q", id), Path: fmt.Sprintf("%s.appliesTo[%d]", path, j)})
			}
		}
		for j, ev := range out.Evidence {
			if strings.TrimSpace(ev.Path) == "" {
				diags = append(diags, Diagnostic{Code: "model.empty_threat_out_of_scope_evidence_path", Severity: SeverityError, Message: "threat out-of-scope evidence path is required", Path: fmt.Sprintf("%s.evidence[%d]", path, j)})
			}
		}
	}

	for i, mitigation := range b.Architecture.AuthoredArchitecture.ThreatMitigations {
		path := fmt.Sprintf("authoredArchitecture.threatMitigations[%d]", i)
		if scenarioRef := strings.TrimSpace(mitigation.ThreatScenarioRef); scenarioRef == "" || !threatScenarios[scenarioRef] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_mitigation_scenario_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown threat scenario ref %q", mitigation.ThreatScenarioRef), Path: path})
		}
		if controlRef := strings.TrimSpace(mitigation.ControlRef); controlRef == "" || !controls[controlRef] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_mitigation_control_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown control ref %q", mitigation.ControlRef), Path: path})
		}
		if st := strings.ToLower(strings.TrimSpace(mitigation.Status)); st != "" && !allowedThreatMitigationStatus[st] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_mitigation_status", Severity: SeverityError, Message: fmt.Sprintf("unknown threat mitigation status %q", mitigation.Status), Path: path})
		}
		if eff := strings.ToLower(strings.TrimSpace(mitigation.Effectiveness)); eff != "" && !allowedThreatMitigationEffectiveness[eff] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_mitigation_effectiveness", Severity: SeverityError, Message: fmt.Sprintf("unknown threat mitigation effectiveness %q", mitigation.Effectiveness), Path: path})
		}
		if owner := strings.TrimSpace(mitigation.Owner); owner != "" && !actors[owner] {
			diags = append(diags, Diagnostic{Code: "model.invalid_threat_mitigation_owner", Severity: SeverityError, Message: fmt.Sprintf("unknown threat mitigation owner %q", mitigation.Owner), Path: path})
		}
		for j, verificationRef := range mitigation.VerificationRefs {
			verificationRef = strings.TrimSpace(verificationRef)
			if verificationRef == "" || !controlVerifications[verificationRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_threat_mitigation_verification_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown control verification ref %q", verificationRef), Path: fmt.Sprintf("%s.verificationRefs[%d]", path, j)})
			}
		}
		for j, ev := range mitigation.Evidence {
			if strings.TrimSpace(ev.Path) == "" {
				diags = append(diags, Diagnostic{Code: "model.empty_threat_mitigation_evidence_path", Severity: SeverityError, Message: "threat mitigation evidence path is required", Path: fmt.Sprintf("%s.evidence[%d]", path, j)})
			}
		}
	}

	for i, verification := range b.Architecture.AuthoredArchitecture.ControlVerifications {
		path := fmt.Sprintf("authoredArchitecture.controlVerifications[%d]", i)
		if controlRef := strings.TrimSpace(verification.ControlRef); controlRef == "" || !controls[controlRef] {
			diags = append(diags, Diagnostic{Code: "model.invalid_control_verification_control_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown control ref %q", verification.ControlRef), Path: path})
		}
		if method := strings.ToLower(strings.TrimSpace(verification.Method)); method != "" && !allowedControlVerificationMethod[method] {
			diags = append(diags, Diagnostic{Code: "model.invalid_control_verification_method", Severity: SeverityError, Message: fmt.Sprintf("unknown control verification method %q", verification.Method), Path: path})
		}
		if status := strings.ToLower(strings.TrimSpace(verification.Status)); status != "" && !allowedControlVerificationStatus[status] {
			diags = append(diags, Diagnostic{Code: "model.invalid_control_verification_status", Severity: SeverityError, Message: fmt.Sprintf("unknown control verification status %q", verification.Status), Path: path})
		}
		if owner := strings.TrimSpace(verification.Owner); owner != "" && !actors[owner] {
			diags = append(diags, Diagnostic{Code: "model.invalid_control_verification_owner", Severity: SeverityError, Message: fmt.Sprintf("unknown control verification owner %q", verification.Owner), Path: path})
		}
		if tested := strings.TrimSpace(verification.LastTested); tested != "" && !isoDateRe.MatchString(tested) {
			diags = append(diags, Diagnostic{Code: "model.invalid_control_verification_last_tested", Severity: SeverityError, Message: fmt.Sprintf("invalid lastTested %q, expected YYYY-MM-DD", verification.LastTested), Path: path})
		}
		for j, scenarioRef := range verification.ThreatScenarioRefs {
			scenarioRef = strings.TrimSpace(scenarioRef)
			if scenarioRef == "" || !threatScenarios[scenarioRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_control_verification_threat_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown threat scenario ref %q", scenarioRef), Path: fmt.Sprintf("%s.threatScenarioRefs[%d]", path, j)})
			}
		}
		for j, riskRef := range verification.RiskRefs {
			riskRef = strings.TrimSpace(riskRef)
			if riskRef == "" || !risks[riskRef] {
				diags = append(diags, Diagnostic{Code: "model.invalid_control_verification_risk_ref", Severity: SeverityError, Message: fmt.Sprintf("unknown risk ref %q", riskRef), Path: fmt.Sprintf("%s.riskRefs[%d]", path, j)})
			}
		}
		for j, ev := range verification.Evidence {
			if strings.TrimSpace(ev.Path) == "" {
				diags = append(diags, Diagnostic{Code: "model.empty_control_verification_evidence_path", Severity: SeverityError, Message: "control verification evidence path is required", Path: fmt.Sprintf("%s.evidence[%d]", path, j)})
			}
		}
	}

	for i, v := range b.Architecture.Views {
		path := fmt.Sprintf("views[%d]", i)
		if strings.TrimSpace(v.ID) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_id", Severity: SeverityError, Message: "view id is required", Path: path})
		}
		if !allowedViewKinds[v.Kind] {
			diags = append(diags, Diagnostic{Code: "model.unknown_view_kind", Severity: SeverityError, Message: fmt.Sprintf("unknown view kind %q", v.Kind), Path: path})
		}
		if len(v.Roots) == 0 {
			diags = append(diags, Diagnostic{Code: "model.empty_roots", Severity: SeverityError, Message: "view must have at least one root", Path: path})
		}
		for _, root := range v.Roots {
			if !validID(root) {
				diags = append(diags, Diagnostic{Code: "model.broken_reference", Severity: SeverityError, Message: fmt.Sprintf("view root %q does not exist", root), Path: path})
			}
		}
		if v.MaxDepth < 0 {
			diags = append(diags, Diagnostic{Code: "model.invalid_view_depth", Severity: SeverityError, Message: "view maxDepth must be >= 0", Path: path})
		}
		for _, kind := range v.IncludeKinds {
			if !allowedViewEntityKinds[strings.TrimSpace(kind)] {
				diags = append(diags, Diagnostic{Code: "model.unknown_view_entity_kind", Severity: SeverityError, Message: fmt.Sprintf("unknown includeKinds value %q", kind), Path: path})
			}
		}
		for _, kind := range v.ExcludeKinds {
			if !allowedViewEntityKinds[strings.TrimSpace(kind)] {
				diags = append(diags, Diagnostic{Code: "model.unknown_view_entity_kind", Severity: SeverityError, Message: fmt.Sprintf("unknown excludeKinds value %q", kind), Path: path})
			}
		}
		for _, rel := range v.IncludeMappings {
			if !allowedViewMappingTypes[strings.TrimSpace(rel)] {
				diags = append(diags, Diagnostic{Code: "model.unknown_view_mapping_type", Severity: SeverityError, Message: fmt.Sprintf("unknown includeMappings value %q", rel), Path: path})
			}
		}
		for _, rel := range v.ExcludeMappings {
			if !allowedViewMappingTypes[strings.TrimSpace(rel)] {
				diags = append(diags, Diagnostic{Code: "model.unknown_view_mapping_type", Severity: SeverityError, Message: fmt.Sprintf("unknown excludeMappings value %q", rel), Path: path})
			}
		}
	}

	return SortDiagnostics(diags)
}

func mappingPairAllowed(mappingType, fromKind, toKind string) bool {
	t := strings.TrimSpace(mappingType)
	from := strings.TrimSpace(fromKind)
	to := strings.TrimSpace(toKind)
	if t == "" || from == "" || to == "" {
		return false
	}
	if t == "contains" {
		if from == "functional_group" && to == "functional_unit" {
			return true
		}
		if from == "functional_unit" {
			return to == "interface" || to == "data_object"
		}
		return false
	}
	allowed := map[string]map[string]bool{
		"depends_on": {
			"functional_unit:functional_unit":    true,
			"functional_unit:referenced_element": true,
			"functional_unit:interface":          true,
			"functional_unit:deployment_target":  true,
		},
		"interacts_with": {
			"actor:functional_unit": true,
		},
		"targets": {
			"attack_vector:functional_unit":    true,
			"attack_vector:referenced_element": true,
			"attack_vector:interface":          true,
			"attack_vector:deployment_target":  true,
		},
		"calls": {
			"functional_unit:functional_unit": true,
			"functional_unit:interface":       true,
		},
		"publishes": {
			"functional_unit:interface":   true,
			"functional_unit:data_object": true,
		},
		"subscribes": {
			"functional_unit:interface":   true,
			"functional_unit:data_object": true,
		},
		"reads": {
			"functional_unit:data_object": true,
		},
		"writes": {
			"functional_unit:data_object": true,
		},
		"streams": {
			"functional_unit:interface":   true,
			"functional_unit:data_object": true,
		},
		"implements": {
			"functional_unit:interface":   true,
			"functional_unit:data_object": true,
		},
		"satisfies": {
			"functional_unit:referenced_element": true,
			"functional_unit:control":            true,
		},
		"allocated_to": {
			"functional_unit:deployment_target": true,
			"interface:deployment_target":       true,
		},
		"verified_by": {
			"functional_unit:referenced_element": true,
			"control:referenced_element":         true,
		},
		"transitions_to": {
			"state:state": true,
		},
		"triggered_by": {
			"state:event": true,
		},
		"guarded_by": {
			"state:control":            true,
			"state:trust_boundary":     true,
			"state:referenced_element": true,
		},
		"deployed_to": {
			"functional_unit:deployment_target": true,
			"interface:deployment_target":       true,
		},
		"mitigated_by": {
			"attack_vector:control": true,
		},
		"bounded_by": {
			"functional_unit:trust_boundary":   true,
			"deployment_target:trust_boundary": true,
			"interface:trust_boundary":         true,
		},
	}
	key := from + ":" + to
	return allowed[t][key]
}
