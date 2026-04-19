package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

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
	"functional_group":   true,
	"functional_unit":    true,
	"actor":              true,
	"attack_vector":      true,
	"referenced_element": true,
	"interface":          true,
	"data_object":        true,
	"deployment_target":  true,
	"control":            true,
	"trust_boundary":     true,
	"state":              true,
	"event":              true,
	"flow":               true,
	"flow_step":          true,
}

var allowedFlowStepKinds = map[string]bool{
	"user_action":          true,
	"system_action":        true,
	"data_move":            true,
	"decision":             true,
	"external_interaction": true,
}

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
	for i, x := range b.Architecture.AuthoredArchitecture.TrustBoundaries {
		addID(x.ID, fmt.Sprintf("authoredArchitecture.trustBoundaries[%d]", i))
		trustBoundaries[x.ID] = true
		kindByID[x.ID] = "trust_boundary"
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

	validID := func(id string) bool {
		_, ok := idOwner[id]
		return ok
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

	for i, f := range b.Architecture.AuthoredArchitecture.Flows {
		flowPath := fmt.Sprintf("authoredArchitecture.flows[%d]", i)
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
			if ref := strings.TrimSpace(s.Ref); ref == "" {
				diags = append(diags, Diagnostic{Code: "model.missing_flow_step_ref", Severity: SeverityError, Message: fmt.Sprintf("flow step %q must reference an authored id", stepID), Path: stepPath})
			} else if !validID(ref) {
				diags = append(diags, Diagnostic{Code: "model.broken_flow_reference", Severity: SeverityError, Message: fmt.Sprintf("flow step %q references unknown id %q", stepID, ref), Path: stepPath})
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
