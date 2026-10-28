// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	earslint "github.com/labeth/ears-lint-go"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func lintRequirementsEARS(requirements model.RequirementsDocument, catalogDoc model.CatalogDocument) []validate.Diagnostic {
	return lintRequirementsEARSWithUsage(requirements, catalogDoc, nil)
}

// lintModelRequirementsEARS lints requirements against the bundle's catalog and
// reports catalog.term_unreferenced only for terms whose id the model uses
// nowhere: a term referenced by the architecture, behavior, assurance,
// compliance, views, decisions or design documents is in use even when no
// requirement text names it.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func lintModelRequirementsEARS(bundle model.Bundle, requirements model.RequirementsDocument) []validate.Diagnostic {
	return lintRequirementsEARSWithUsage(requirements, bundle.Catalog, modelReferencedIDs(bundle))
}

// lintRequirementsEARSWithUsage runs the EARS lint; catalog terms whose id is in
// usedIDs are left out of the coverage check.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func lintRequirementsEARSWithUsage(requirements model.RequirementsDocument, catalogDoc model.CatalogDocument, usedIDs map[string]bool) []validate.Diagnostic {
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
	catalog := toEarsCatalog(catalogDoc)

	results := earslint.LintEarsBatch(items, catalog, &earslint.Options{
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
			severity := mapEarsSeverity(d.Severity)
			if isBlockingCatalogDiagnostic(strings.TrimSpace(d.Code)) {
				severity = validate.SeverityError
			}
			out = append(out, validate.Diagnostic{
				Code:     d.Code,
				Severity: severity,
				Message:  d.Message,
				Path:     path,
			})
		}
	}

	coverageDiags := earslint.LintCatalogCoverage(items, withoutUsedTerms(toEarsCoverageCatalog(catalogDoc), usedIDs), &earslint.Options{
		Mode:       mode,
		CommaAsAnd: requirements.LintRun.CommaAsAnd,
	})
	for _, d := range coverageDiags {
		out = append(out, validate.Diagnostic{
			Code:     d.Code,
			Severity: mapEarsSeverity(d.Severity),
			Message:  d.Message,
			Path:     "requirements",
		})
	}
	waivers := map[string][]string{}
	for _, req := range requirements.Requirements {
		for _, w := range req.LintWaivers {
			if strings.TrimSpace(w.Reason) != "" {
				waivers[strings.TrimSpace(req.ID)] = append(waivers[strings.TrimSpace(req.ID)], strings.TrimSpace(w.Code))
			}
		}
	}
	return validate.SortDiagnostics(SuppressWaivedDiagnostics(out, waivers))
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func toEarsCatalog(doc model.CatalogDocument) earslint.Catalog {
	systems := append([]earslint.CatalogEntry{}, toEarsEntries(doc.Catalog.Systems)...)
	systems = append(systems, toEarsEntries(doc.Catalog.FunctionalGroups)...)
	systems = append(systems, toEarsEntries(doc.Catalog.FunctionalUnits)...)
	return earslint.Catalog{
		Systems:    dedupeEarsEntries(systems),
		Actors:     toEarsEntries(doc.Catalog.Actors),
		Events:     toEarsEntries(doc.Catalog.Events),
		States:     toEarsEntries(doc.Catalog.States),
		Features:   toEarsEntries(doc.Catalog.Features),
		Modes:      toEarsEntries(doc.Catalog.Modes),
		Conditions: toEarsEntries(doc.Catalog.Conditions),
		DataTerms:  toEarsEntries(doc.Catalog.DataTerms),
	}
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func toEarsCoverageCatalog(doc model.CatalogDocument) earslint.Catalog {
	return earslint.Catalog{
		Systems:    toEarsEntries(doc.Catalog.Systems),
		Actors:     toEarsEntries(doc.Catalog.Actors),
		Events:     toEarsEntries(doc.Catalog.Events),
		States:     toEarsEntries(doc.Catalog.States),
		Features:   toEarsEntries(doc.Catalog.Features),
		Modes:      toEarsEntries(doc.Catalog.Modes),
		Conditions: toEarsEntries(doc.Catalog.Conditions),
		DataTerms:  toEarsEntries(doc.Catalog.DataTerms),
	}
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
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

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func dedupeEarsEntries(in []earslint.CatalogEntry) []earslint.CatalogEntry {
	if len(in) == 0 {
		return nil
	}
	out := make([]earslint.CatalogEntry, 0, len(in))
	seen := map[string]bool{}
	for _, e := range in {
		id := strings.TrimSpace(e.ID)
		name := strings.TrimSpace(e.Name)
		if id == "" && name == "" {
			continue
		}
		key := strings.ToLower(id) + "|" + strings.ToLower(name)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, e)
	}
	return out
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
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

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func isBlockingCatalogDiagnostic(code string) bool {
	if code == "expr.unknown_term" {
		return true
	}
	return strings.HasPrefix(code, "catalog.") && strings.HasSuffix(code, "_unresolved")
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func requirementPath(id string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "requirements"
	}
	return fmt.Sprintf("requirements[%s]", id)
}

// withoutUsedTerms drops catalog entries whose id is in used, so the coverage
// lint reports only terms the model uses nowhere.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func withoutUsedTerms(c earslint.Catalog, used map[string]bool) earslint.Catalog {
	if len(used) == 0 {
		return c
	}
	keep := func(in []earslint.CatalogEntry) []earslint.CatalogEntry {
		out := make([]earslint.CatalogEntry, 0, len(in))
		for _, e := range in {
			if !used[strings.TrimSpace(e.ID)] {
				out = append(out, e)
			}
		}
		return out
	}
	return earslint.Catalog{
		Systems: keep(c.Systems), Actors: keep(c.Actors), Events: keep(c.Events), States: keep(c.States),
		Features: keep(c.Features), Modes: keep(c.Modes), Conditions: keep(c.Conditions), DataTerms: keep(c.DataTerms),
	}
}

// modelReferencedIDs collects every id-like token in the model documents other
// than the catalog and the requirements (whose text the EARS lint already checks).
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func modelReferencedIDs(bundle model.Bundle) map[string]bool {
	used := map[string]bool{}
	for _, doc := range []any{bundle.Architecture, bundle.Behavior, bundle.Assurance, bundle.Compliance, bundle.Views, bundle.Decisions, bundle.Design, bundle.Aviation} {
		collectIDTokens(reflect.ValueOf(doc), used, 0)
	}
	return used
}

// collectIDTokens walks v and records each token of every string it holds.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func collectIDTokens(v reflect.Value, used map[string]bool, depth int) {
	if depth > 64 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			collectIDTokens(v.Elem(), used, depth+1)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				collectIDTokens(v.Field(i), used, depth+1)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			collectIDTokens(v.Index(i), used, depth+1)
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			collectIDTokens(iter.Key(), used, depth+1)
			collectIDTokens(iter.Value(), used, depth+1)
		}
	case reflect.String:
		for _, tok := range strings.FieldsFunc(v.String(), isNotIDRune) {
			used[tok] = true
		}
	}
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func isNotIDRune(r rune) bool {
	return !(r == '-' || r == '_' || r == '.' || unicode.IsLetter(r) || unicode.IsDigit(r))
}

// SuppressWaivedDiagnostics removes requirement-scoped diagnostics a model has
// explicitly waived. waivers maps a requirement id to the diagnostic codes
// waived for it (for example "lint.multiple_responses"). Only warnings are
// suppressed: an error is never waived. Diagnostics whose path is not a
// requirement path are kept.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func SuppressWaivedDiagnostics(diags []validate.Diagnostic, waivers map[string][]string) []validate.Diagnostic {
	if len(waivers) == 0 {
		return diags
	}
	waived := map[string]bool{}
	for req, codes := range waivers {
		for _, c := range codes {
			waived[strings.TrimSpace(req)+"|"+strings.TrimSpace(c)] = true
		}
	}
	out := make([]validate.Diagnostic, 0, len(diags))
	for _, d := range diags {
		if d.Severity != validate.SeverityError {
			if id, ok := requirementIDFromPath(d.Path); ok && waived[id+"|"+d.Code] {
				continue
			}
		}
		out = append(out, d)
	}
	return out
}

// requirementIDFromPath extracts REQ from "requirements[REQ]" or "requirements[REQ]@s:e".
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func requirementIDFromPath(path string) (string, bool) {
	rest, ok := strings.CutPrefix(strings.TrimSpace(path), "requirements[")
	if !ok {
		return "", false
	}
	id, _, ok := strings.Cut(rest, "]")
	if !ok || strings.TrimSpace(id) == "" {
		return "", false
	}
	return strings.TrimSpace(id), true
}
