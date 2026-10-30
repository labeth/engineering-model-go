// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"fmt"
	"reflect"
	"strings"
	"unicode"

	earslint "github.com/labeth/ears-lint-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func lintRequirementsEARS(requirements model.RequirementsDocument, catalogDoc model.CatalogDocument) []validate.Diagnostic {
	return lintRequirementsEARSWithUsage(requirements, catalogDoc, nil)
}

// LintModelRequirementsEARS lints requirements against the bundle's catalog and
// reports catalog.term_unreferenced only for terms the model uses nowhere: a
// term whose id the architecture, behavior, assurance, compliance, views,
// decisions or design documents reference, or whose name appears in any
// authored model text other than its own entry, is in use even when no
// requirement text names it.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func LintModelRequirementsEARS(bundle model.Bundle, requirements model.RequirementsDocument) []validate.Diagnostic {
	used := modelReferencedIDs(bundle)
	for id := range catalogTermsNamedInModel(bundle, requirements) {
		used[id] = true
	}
	diags := lintRequirementsEARSWithUsage(requirements, bundle.Catalog, used)
	for i, d := range diags {
		if d.Code == "catalog.term_unreferenced" {
			diags[i].Message = strings.Replace(d.Message, "is not referenced by any requirement text", "is not referenced anywhere in the model", 1)
		}
	}
	return diags
}

// catalogTermsNamedInModel returns the ids of catalog entries whose name or an
// alias appears, with the EARS lint's word-boundary phrase matching, in any
// authored model text other than the entry itself: requirement text, notes,
// verification criteria and rationale; other catalog entries' names,
// definitions and aliases; and every string of the architecture, behavior,
// assurance, compliance, views, decisions, design and aviation documents.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func catalogTermsNamedInModel(bundle model.Bundle, requirements model.RequirementsDocument) map[string]bool {
	var texts []string
	add := func(ss ...string) {
		for _, s := range ss {
			if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
				texts = append(texts, s)
			}
		}
	}
	for _, r := range requirements.Requirements {
		add(r.Title, r.Text, r.Notes, r.VerificationCriteria, r.Rationale, r.DerivedRationale)
	}
	for _, doc := range []any{bundle.Architecture, bundle.Behavior, bundle.Assurance, bundle.Compliance, bundle.Views, bundle.Decisions, bundle.Design, bundle.Aviation} {
		walkStrings(reflect.ValueOf(doc), 0, func(s string) { add(s) })
	}
	g := bundle.Catalog.Catalog
	groups := [][]model.CatalogEntry{g.Systems, g.FunctionalGroups, g.FunctionalUnits, g.ReferencedElements, g.Actors, g.AttackVectors, g.Events, g.States, g.Features, g.Modes, g.Conditions, g.DataTerms}
	type entryText struct {
		id    string
		texts []string
	}
	var catalogTexts []entryText
	for _, group := range groups {
		for _, e := range group {
			et := entryText{id: strings.TrimSpace(e.ID)}
			for _, s := range append([]string{e.Name, e.Definition}, e.Aliases...) {
				if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
					et.texts = append(et.texts, s)
				}
			}
			catalogTexts = append(catalogTexts, et)
		}
	}
	named := map[string]bool{}
	for _, group := range groups {
		for _, e := range group {
			id := strings.TrimSpace(e.ID)
			if id == "" || named[id] {
				continue
			}
			candidates := []string{}
			for _, c := range append([]string{e.Name}, e.Aliases...) {
				if c = strings.ToLower(strings.TrimSpace(c)); c != "" {
					candidates = append(candidates, c)
				}
			}
			found := anyPhraseIn(texts, candidates)
			for _, other := range catalogTexts {
				if found {
					break
				}
				if other.id != id {
					found = anyPhraseIn(other.texts, candidates)
				}
			}
			if found {
				named[id] = true
			}
		}
	}
	return named
}

// anyPhraseIn reports whether any lowercase phrase occurs in any lowercase text
// on word boundaries.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func anyPhraseIn(texts, phrases []string) bool {
	for _, p := range phrases {
		for _, t := range texts {
			if containsPhrase(t, p) {
				return true
			}
		}
	}
	return false
}

// containsPhrase matches phrase in text only where it is not part of a longer
// word, the same rule the EARS catalog-coverage lint applies.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func containsPhrase(text, phrase string) bool {
	isWord := func(idx int) bool {
		if idx < 0 || idx >= len(text) {
			return false
		}
		b := text[idx]
		return (b >= 'a' && b <= 'z') || (b >= '0' && b <= '9')
	}
	for from := 0; phrase != "" && from < len(text); {
		i := strings.Index(text[from:], phrase)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(phrase)
		if !isWord(start-1) && !isWord(end) {
			return true
		}
		from = start + 1
	}
	return false
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
		walkStrings(reflect.ValueOf(doc), 0, func(s string) {
			for _, tok := range strings.FieldsFunc(s, isNotIDRune) {
				used[tok] = true
			}
		})
	}
	return used
}

// walkStrings calls visit for every string reachable from v through exported
// struct fields, pointers, interfaces, slices, arrays and map keys and values.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func walkStrings(v reflect.Value, depth int, visit func(string)) {
	if depth > 64 || !v.IsValid() {
		return
	}
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		if !v.IsNil() {
			walkStrings(v.Elem(), depth+1, visit)
		}
	case reflect.Struct:
		for i := 0; i < v.NumField(); i++ {
			if v.Type().Field(i).IsExported() {
				walkStrings(v.Field(i), depth+1, visit)
			}
		}
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			walkStrings(v.Index(i), depth+1, visit)
		}
	case reflect.Map:
		iter := v.MapRange()
		for iter.Next() {
			walkStrings(iter.Key(), depth+1, visit)
			walkStrings(iter.Value(), depth+1, visit)
		}
	case reflect.String:
		visit(v.String())
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
