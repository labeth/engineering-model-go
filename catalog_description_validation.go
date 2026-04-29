// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func validateCatalogDescriptions(doc model.CatalogDocument) []validate.Diagnostic {
	type catalogGroup struct {
		pathKey string
		entries []model.CatalogEntry
	}

	groups := []catalogGroup{
		{pathKey: "systems", entries: doc.Catalog.Systems},
		{pathKey: "functionalGroups", entries: doc.Catalog.FunctionalGroups},
		{pathKey: "functionalUnits", entries: doc.Catalog.FunctionalUnits},
		{pathKey: "referencedElements", entries: doc.Catalog.ReferencedElements},
		{pathKey: "actors", entries: doc.Catalog.Actors},
		{pathKey: "attackVectors", entries: doc.Catalog.AttackVectors},
		{pathKey: "events", entries: doc.Catalog.Events},
		{pathKey: "states", entries: doc.Catalog.States},
		{pathKey: "features", entries: doc.Catalog.Features},
		{pathKey: "modes", entries: doc.Catalog.Modes},
		{pathKey: "conditions", entries: doc.Catalog.Conditions},
		{pathKey: "dataTerms", entries: doc.Catalog.DataTerms},
	}

	out := make([]validate.Diagnostic, 0)
	for _, g := range groups {
		for i, e := range g.entries {
			if strings.TrimSpace(e.Definition) != "" {
				continue
			}
			id := strings.TrimSpace(e.ID)
			if id == "" {
				id = "<missing-id>"
			}
			out = append(out, validate.Diagnostic{
				Code:     "catalog.missing_description",
				Severity: validate.SeverityError,
				Message:  fmt.Sprintf("catalog entry %q must include a non-empty definition/description", id),
				Path:     fmt.Sprintf("catalog.%s[%d]", g.pathKey, i),
			})
		}
	}
	return validate.SortDiagnostics(out)
}
