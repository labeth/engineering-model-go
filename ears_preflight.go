package engmodel

import (
	"fmt"
	"strings"

	earslint "github.com/labeth/ears-lint-go"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

func lintRequirementsEARS(requirements model.RequirementsDocument, catalogDoc model.CatalogDocument) []validate.Diagnostic {
	items := make([][2]string, 0, len(requirements.Requirements))
	for _, req := range requirements.Requirements {
		items = append(items, [2]string{req.ID, req.Text})
	}
	if len(items) == 0 {
		return nil
	}

	mode := earslint.ModeStrict
	if strings.EqualFold(strings.TrimSpace(requirements.LintRun.Mode), string(earslint.ModeGuided)) {
		mode = earslint.ModeGuided
	}

	results := earslint.LintEarsBatch(items, toEarsCatalog(catalogDoc), &earslint.Options{
		Mode:       mode,
		CommaAsAnd: requirements.LintRun.CommaAsAnd,
	})

	out := make([]validate.Diagnostic, 0)
	for _, r := range results {
		reqPath := requirementPath(r.ID)
		for _, d := range r.Diagnostics {
			path := reqPath
			if d.Span != nil {
				path = fmt.Sprintf("%s@%d:%d", reqPath, d.Span.Start, d.Span.End)
			}
			out = append(out, validate.Diagnostic{
				Code:     d.Code,
				Severity: mapEarsSeverity(d.Severity),
				Message:  d.Message,
				Path:     path,
			})
		}
	}
	return validate.SortDiagnostics(out)
}

func toEarsCatalog(doc model.CatalogDocument) earslint.Catalog {
	return earslint.Catalog{
		Systems:    append(toEarsEntries(doc.Catalog.FunctionalGroups), toEarsEntries(doc.Catalog.FunctionalUnits)...),
		Actors:     toEarsEntries(doc.Catalog.Actors),
		Events:     toEarsEntries(doc.Catalog.Events),
		States:     toEarsEntries(doc.Catalog.States),
		Features:   toEarsEntries(doc.Catalog.Features),
		Modes:      toEarsEntries(doc.Catalog.Modes),
		Conditions: toEarsEntries(doc.Catalog.Conditions),
		DataTerms:  toEarsEntries(doc.Catalog.DataTerms),
	}
}

func toEarsEntries(in []model.CatalogEntry) []earslint.CatalogEntry {
	out := make([]earslint.CatalogEntry, 0, len(in))
	for _, e := range in {
		out = append(out, earslint.CatalogEntry{
			ID:      strings.TrimSpace(e.ID),
			Name:    strings.TrimSpace(e.Name),
			Aliases: append([]string(nil), e.Aliases...),
		})
	}
	return out
}

func mapEarsSeverity(in earslint.Severity) validate.Severity {
	switch in {
	case earslint.SeverityError:
		return validate.SeverityError
	case earslint.SeverityWarning:
		return validate.SeverityWarning
	default:
		// Keep the host model binary (error/warning) severity scale.
		return validate.SeverityWarning
	}
}

func requirementPath(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "requirements"
	}
	return fmt.Sprintf("requirements[%s]", id)
}
