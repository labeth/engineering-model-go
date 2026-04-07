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
}

var allowedMappingTypes = map[string]bool{
	"contains":       true,
	"depends_on":     true,
	"interacts_with": true,
	"targets":        true,
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

	for i, g := range b.Architecture.AuthoredArchitecture.FunctionalGroups {
		addID(g.ID, fmt.Sprintf("authoredArchitecture.functionalGroups[%d]", i))
		groups[g.ID] = true
	}
	for i, u := range b.Architecture.AuthoredArchitecture.FunctionalUnits {
		path := fmt.Sprintf("authoredArchitecture.functionalUnits[%d]", i)
		addID(u.ID, path)
		units[u.ID] = true
		if strings.TrimSpace(u.Group) == "" || !groups[u.Group] {
			diags = append(diags, Diagnostic{Code: "model.invalid_group", Severity: SeverityError, Message: fmt.Sprintf("functional unit %q must reference a valid functional group", u.ID), Path: path})
		}
	}
	for i, a := range b.Architecture.AuthoredArchitecture.Actors {
		addID(a.ID, fmt.Sprintf("authoredArchitecture.actors[%d]", i))
		actors[a.ID] = true
	}
	for i, a := range b.Architecture.AuthoredArchitecture.AttackVectors {
		addID(a.ID, fmt.Sprintf("authoredArchitecture.attackVectors[%d]", i))
		vectors[a.ID] = true
	}
	for i, r := range b.Architecture.AuthoredArchitecture.ReferencedElements {
		addID(r.ID, fmt.Sprintf("authoredArchitecture.referencedElements[%d]", i))
		references[r.ID] = true
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
	}

	return SortDiagnostics(diags)
}
