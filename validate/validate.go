package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

var allowedViewKinds = map[string]bool{
	"system-context": true,
	"container":      true,
	"deployment":     true,
}

var allowedRelationTypes = map[string]bool{
	"uses":       true,
	"depends_on": true,
	"part_of":    true,
	"deploys":    true,
	"runs_in":    true,
	"manages":    true,
}

func Bundle(b model.Bundle) []Diagnostic {
	diags := []Diagnostic{}
	idOwner := map[string]string{}
	catalogActors := map[string]bool{}
	catalogSystems := map[string]bool{}
	catalogIDs := map[string]bool{}

	addID := func(id, owner string) {
		if id == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_id", Severity: SeverityError, Message: fmt.Sprintf("missing id for %s", owner), Path: owner})
			return
		}
		if prev, ok := idOwner[id]; ok {
			diags = append(diags, Diagnostic{Code: "model.duplicate_id", Severity: SeverityError, Message: fmt.Sprintf("duplicate id %q (%s, %s)", id, prev, owner), Path: owner})
			return
		}
		idOwner[id] = owner
	}

	addCatalogID := func(id string) {
		if strings.TrimSpace(id) == "" {
			return
		}
		catalogIDs[id] = true
	}

	for _, x := range b.Catalog.Catalog.Actors {
		catalogActors[x.ID] = true
		addCatalogID(x.ID)
	}
	for _, x := range b.Catalog.Catalog.Systems {
		catalogSystems[x.ID] = true
		addCatalogID(x.ID)
	}
	for _, x := range b.Catalog.Catalog.Events {
		addCatalogID(x.ID)
	}
	for _, x := range b.Catalog.Catalog.States {
		addCatalogID(x.ID)
	}
	for _, x := range b.Catalog.Catalog.Features {
		addCatalogID(x.ID)
	}
	for _, x := range b.Catalog.Catalog.Modes {
		addCatalogID(x.ID)
	}
	for _, x := range b.Catalog.Catalog.Conditions {
		addCatalogID(x.ID)
	}
	for _, x := range b.Catalog.Catalog.DataTerms {
		addCatalogID(x.ID)
	}

	for i, p := range b.Architecture.C4.People {
		addID(p.ID, fmt.Sprintf("c4.people[%d]", i))
		if !catalogActors[p.ID] {
			diags = append(diags, Diagnostic{
				Code:     "model.catalog_mapping_missing",
				Severity: SeverityError,
				Message:  fmt.Sprintf("person %q is not mapped to catalog actors", p.ID),
				Path:     fmt.Sprintf("c4.people[%d]", i),
			})
		}
	}
	for i, s := range b.Architecture.C4.SoftwareSystems {
		addID(s.ID, fmt.Sprintf("c4.softwareSystems[%d]", i))
		if !catalogSystems[s.ID] {
			diags = append(diags, Diagnostic{
				Code:     "model.catalog_mapping_missing",
				Severity: SeverityError,
				Message:  fmt.Sprintf("softwareSystem %q is not mapped to catalog systems", s.ID),
				Path:     fmt.Sprintf("c4.softwareSystems[%d]", i),
			})
		}
	}
	for i, c := range b.Architecture.C4.Containers {
		addID(c.ID, fmt.Sprintf("c4.containers[%d]", i))
	}
	for i, c := range b.Architecture.C4.Components {
		addID(c.ID, fmt.Sprintf("c4.components[%d]", i))
	}
	systems := map[string]bool{}
	containers := map[string]bool{}
	for _, s := range b.Architecture.C4.SoftwareSystems {
		systems[s.ID] = true
	}
	for _, c := range b.Architecture.C4.Containers {
		containers[c.ID] = true
	}

	for i, c := range b.Architecture.C4.Containers {
		if c.PartOf == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_part_of", Severity: SeverityError, Message: "container missing partOf", Path: fmt.Sprintf("c4.containers[%d]", i)})
			continue
		}
		if !systems[c.PartOf] {
			diags = append(diags, Diagnostic{Code: "model.invalid_part_of", Severity: SeverityError, Message: fmt.Sprintf("container partOf %q does not exist", c.PartOf), Path: fmt.Sprintf("c4.containers[%d]", i)})
		}
	}

	for i, c := range b.Architecture.C4.Components {
		if c.PartOf == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_part_of", Severity: SeverityError, Message: "component missing partOf", Path: fmt.Sprintf("c4.components[%d]", i)})
			continue
		}
		if !containers[c.PartOf] {
			diags = append(diags, Diagnostic{Code: "model.invalid_part_of", Severity: SeverityError, Message: fmt.Sprintf("component partOf %q does not exist", c.PartOf), Path: fmt.Sprintf("c4.components[%d]", i)})
		}
	}

	for i, rel := range b.Architecture.Relationships {
		path := fmt.Sprintf("relationships[%d]", i)
		if rel.Type == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_relation_type", Severity: SeverityError, Message: "relationship type is required", Path: path})
		} else if !allowedRelationTypes[rel.Type] {
			diags = append(diags, Diagnostic{Code: "model.unknown_relation_type", Severity: SeverityWarning, Message: fmt.Sprintf("unknown relationship type %q", rel.Type), Path: path})
		}
		if _, ok := idOwner[rel.From]; !ok {
			diags = append(diags, Diagnostic{Code: "model.broken_reference", Severity: SeverityError, Message: fmt.Sprintf("relationship from %q does not exist", rel.From), Path: path})
		}
		if _, ok := idOwner[rel.To]; !ok {
			diags = append(diags, Diagnostic{Code: "model.broken_reference", Severity: SeverityError, Message: fmt.Sprintf("relationship to %q does not exist", rel.To), Path: path})
		}
		for _, ref := range rel.CatalogRefs {
			if !catalogIDs[ref] {
				diags = append(diags, Diagnostic{
					Code:     "model.catalog_ref_unknown",
					Severity: SeverityError,
					Message:  fmt.Sprintf("relationship catalogRefs contains unknown id %q", ref),
					Path:     path,
				})
			}
		}
	}

	for i, v := range b.Architecture.Viewpoints {
		path := fmt.Sprintf("viewpoints[%d]", i)
		if strings.TrimSpace(v.ID) == "" {
			diags = append(diags, Diagnostic{Code: "model.missing_id", Severity: SeverityError, Message: "viewpoint id is required", Path: path})
		}
		if !allowedViewKinds[v.Kind] {
			diags = append(diags, Diagnostic{Code: "model.unknown_view_kind", Severity: SeverityError, Message: fmt.Sprintf("unknown viewpoint kind %q", v.Kind), Path: path})
		}
		if len(v.Roots) == 0 {
			diags = append(diags, Diagnostic{Code: "model.empty_roots", Severity: SeverityError, Message: "viewpoint must have at least one root", Path: path})
		}
		for _, root := range v.Roots {
			if v.Kind == "deployment" {
				// Deployment views may root in infra-derived IDs (e.g., ENV-*, EKS-*) that are extracted at render time.
				continue
			}
			if _, ok := idOwner[root]; !ok {
				diags = append(diags, Diagnostic{Code: "model.broken_reference", Severity: SeverityError, Message: fmt.Sprintf("viewpoint root %q does not exist", root), Path: path})
			}
		}
		for _, rel := range v.IncludeRelations {
			if !allowedRelationTypes[rel] {
				diags = append(diags, Diagnostic{Code: "model.unknown_relation_type", Severity: SeverityWarning, Message: fmt.Sprintf("viewpoint includes unknown relation %q", rel), Path: path})
			}
		}
	}

	return SortDiagnostics(diags)
}
