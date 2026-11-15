// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"fmt"
	"reflect"
	"regexp"
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
	diags := append(lintRequirementsEARSWithUsage(requirements, bundle.Catalog, used), lintRequirementsModeled(bundle, requirements)...)
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
	for _, r := range requirements.Requirements {
		texts = append(texts, lowerTrimmed(r.Title, r.Text, r.Notes, r.VerificationCriteria, r.Rationale, r.DerivedRationale)...)
	}
	walkModelDocuments(bundle, func(s string) { texts = append(texts, lowerTrimmed(s)...) })
	groups := entryGroups(bundle.Catalog)
	lookup := namedLookup{texts: texts, catalogTexts: catalogEntryTexts(groups)}
	named := map[string]bool{}
	for _, group := range groups {
		lookup.collectNamed(group, named)
	}
	return named
}

// lowerTrimmed is the lowercased, trimmed, non-empty strings.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func lowerTrimmed(ss ...string) []string {
	var out []string
	for _, s := range ss {
		if s = strings.ToLower(strings.TrimSpace(s)); s != "" {
			out = append(out, s)
		}
	}
	return out
}

// walkModelDocuments calls visit for every string of the model documents other
// than the catalog and the requirements.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func walkModelDocuments(bundle model.Bundle, visit func(string)) {
	for _, doc := range []any{bundle.Architecture, bundle.Behavior, bundle.Assurance, bundle.Compliance, bundle.Views, bundle.Decisions, bundle.Design, bundle.Aviation} {
		walkStrings(reflect.ValueOf(doc), 0, visit)
	}
}

// entryGroups lists the entry groups of the catalog the model texts are matched against.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func entryGroups(doc model.CatalogDocument) [][]model.CatalogEntry {
	g := doc.Catalog
	return [][]model.CatalogEntry{g.Systems, g.FunctionalGroups, g.FunctionalUnits, g.ReferencedElements, g.Actors, g.AttackVectors, g.Events, g.States, g.Features, g.Modes, g.Conditions, g.DataTerms}
}

// entryText is the lowercased name, definition and aliases of one catalog entry.
type entryText struct {
	id    string
	texts []string
}

// catalogEntryTexts is the texts of every catalog entry.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func catalogEntryTexts(groups [][]model.CatalogEntry) []entryText {
	var out []entryText
	for _, group := range groups {
		for _, e := range group {
			out = append(out, entryText{id: strings.TrimSpace(e.ID), texts: lowerTrimmed(append([]string{e.Name, e.Definition}, e.Aliases...)...)})
		}
	}
	return out
}

// namedLookup finds catalog terms in the authored texts of the model.
type namedLookup struct {
	texts        []string
	catalogTexts []entryText
}

// collectNamed adds the ids of the entries whose name or alias the model texts name.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func (l namedLookup) collectNamed(group []model.CatalogEntry, named map[string]bool) {
	for _, e := range group {
		id := strings.TrimSpace(e.ID)
		if id != "" && !named[id] && l.names(id, lowerTrimmed(append([]string{e.Name}, e.Aliases...)...)) {
			named[id] = true
		}
	}
}

// names reports whether any candidate phrase appears in the model texts or in
// the texts of a catalog entry other than the entry id itself.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func (l namedLookup) names(id string, candidates []string) bool {
	if anyPhraseIn(l.texts, candidates) {
		return true
	}
	for _, other := range l.catalogTexts {
		if other.id != id && anyPhraseIn(other.texts, candidates) {
			return true
		}
	}
	return false
}

// anyPhraseIn reports whether any lowercase phrase occurs in any lowercase text
// on word boundaries.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func anyPhraseIn(texts, phrases []string) bool {
	for _, p := range phrases {
		if phraseInAny(texts, p) {
			return true
		}
	}
	return false
}

// phraseInAny reports whether the lowercase phrase occurs in any lowercase text on word boundaries.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func phraseInAny(texts []string, phrase string) bool {
	for _, t := range texts {
		if containsPhrase(t, phrase) {
			return true
		}
	}
	return false
}

// containsPhrase matches phrase in text only where it is not part of a longer
// word, the same rule the EARS catalog-coverage lint applies.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func containsPhrase(text, phrase string) bool {
	for from := 0; phrase != "" && from < len(text); {
		i := strings.Index(text[from:], phrase)
		if i < 0 {
			return false
		}
		start := from + i
		end := start + len(phrase)
		if !isWordByteAt(text, start-1) && !isWordByteAt(text, end) {
			return true
		}
		from = start + 1
	}
	return false
}

// isWordByteAt reports whether text has a lowercase letter or digit at idx.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func isWordByteAt(text string, idx int) bool {
	if idx < 0 || idx >= len(text) {
		return false
	}
	b := text[idx]
	return b-'a' < 26 || b-'0' < 10
}

// lintRequirementsEARSWithUsage runs the EARS lint; catalog terms whose id is in
// usedIDs are left out of the coverage check.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-072, REQ-EMG-123
func lintRequirementsEARSWithUsage(requirements model.RequirementsDocument, catalogDoc model.CatalogDocument, usedIDs map[string]bool) []validate.Diagnostic {
	items := make([][2]string, 0, len(requirements.Requirements))
	for _, req := range requirements.Requirements {
		items = append(items, [2]string{req.ID, req.Text})
	}
	if len(items) == 0 {
		return nil
	}
	results := earslint.LintEarsBatch(items, toEarsCatalog(catalogDoc), earsOptions(requirements))
	out := lintResultDiagnostics(results, requirements)
	coverageDiags := earslint.LintCatalogCoverage(items, withoutUsedTerms(toEarsCoverageCatalog(catalogDoc), usedIDs), earsOptions(requirements))
	for _, d := range coverageDiags {
		out = append(out, validate.Diagnostic{
			Code:     d.Code,
			Severity: mapEarsSeverity(d.Severity),
			Message:  d.Message,
			Path:     "requirements",
		})
	}
	return validate.SortDiagnostics(SuppressWaivedDiagnostics(out, requirementWaivers(requirements)))
}

// earsOptions are the lint options of the requirements document: strict unless
// the lint run asks for guided mode.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-072, REQ-EMG-123
func earsOptions(requirements model.RequirementsDocument) *earslint.Options {
	mode := earslint.ModeStrict
	if strings.EqualFold(strings.TrimSpace(requirements.LintRun.Mode), string(earslint.ModeGuided)) {
		mode = earslint.ModeGuided
	}
	return &earslint.Options{Mode: mode, CommaAsAnd: requirements.LintRun.CommaAsAnd}
}

// lintResultDiagnostics converts the lint results to diagnostics: a multiple
// responses finding that only lists fields is dropped and an unresolved
// catalog term is blocking.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-072, REQ-EMG-123
func lintResultDiagnostics(results []earslint.LintResult, requirements model.RequirementsDocument) []validate.Diagnostic {
	textByID := map[string]string{}
	for _, req := range requirements.Requirements {
		textByID[req.ID] = req.Text
	}
	out := make([]validate.Diagnostic, 0)
	for _, r := range results {
		out = append(out, resultDiagnostics(r, textByID[r.ID])...)
	}
	return out
}

// resultDiagnostics converts the diagnostics of one requirement's lint result.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-072, REQ-EMG-123
func resultDiagnostics(r earslint.LintResult, text string) []validate.Diagnostic {
	var out []validate.Diagnostic
	reqPath := requirementPath(r.ID)
	sections := unresolvedSections(r.Diagnostics)
	for _, d := range r.Diagnostics {
		if d.Code == "lint.multiple_responses" && responseConjunctionsOnlyInFieldList(text) {
			continue
		}
		out = append(out, validate.Diagnostic{
			Code:     d.Code,
			Severity: diagnosticSeverity(d),
			Message:  unknownTermMessage(d, text, sections),
			Path:     diagnosticPath(reqPath, d),
		})
	}
	return out
}

// diagnosticSeverity maps the lint severity; an unresolved catalog term is an error.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func diagnosticSeverity(d earslint.Diagnostic) validate.Severity {
	if isBlockingCatalogDiagnostic(strings.TrimSpace(d.Code)) {
		return validate.SeverityError
	}
	return mapEarsSeverity(d.Severity)
}

// diagnosticPath is the requirement path, with the span of the finding when it has one.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func diagnosticPath(reqPath string, d earslint.Diagnostic) string {
	if d.Span == nil {
		return reqPath
	}
	return fmt.Sprintf("%s@%d:%d", reqPath, d.Span.Start, d.Span.End)
}

// requirementWaivers maps each requirement id to the lint codes it waives with a reason.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func requirementWaivers(requirements model.RequirementsDocument) map[string][]string {
	waivers := map[string][]string{}
	for _, req := range requirements.Requirements {
		if codes := reasonedWaiverCodes(req); len(codes) > 0 {
			id := strings.TrimSpace(req.ID)
			waivers[id] = append(waivers[id], codes...)
		}
	}
	return waivers
}

// reasonedWaiverCodes are the codes of a requirement's lint waivers that give a reason.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func reasonedWaiverCodes(req model.Requirement) []string {
	var codes []string
	for _, w := range req.LintWaivers {
		if strings.TrimSpace(w.Reason) != "" {
			codes = append(codes, strings.TrimSpace(w.Code))
		}
	}
	return codes
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

// catalogSections maps the role of an ears-lint catalog.<role>_unresolved
// code to the catalog section its term belongs in.
var catalogSections = map[string]string{
	"system": "systems", "actor": "actors", "event": "events", "state": "states", "feature": "features",
	"mode": "modes", "condition": "conditions", "data_term": "dataTerms",
}

// unresolvedSections returns, per span of an unresolved term, the catalog
// section its clause position asks for: ears-lint reports
// catalog.<role>_unresolved at the same span as expr.unknown_term.
// TRLC-LINKS: REQ-EMG-123
func unresolvedSections(diags []earslint.Diagnostic) map[earslint.Span]string {
	out := map[earslint.Span]string{}
	for _, d := range diags {
		role, ok := strings.CutSuffix(strings.TrimPrefix(d.Code, "catalog."), "_unresolved")
		if !ok || !strings.HasPrefix(d.Code, "catalog.") || d.Span == nil {
			continue
		}
		if section, known := catalogSections[role]; known {
			out[*d.Span] = section
		}
	}
	return out
}

// unknownTermMessage names the term of an expr.unknown_term or
// catalog.<role>_unresolved diagnostic and the catalog section it belongs
// in (from sections); other diagnostics keep their message.
// TRLC-LINKS: REQ-EMG-123
func unknownTermMessage(d earslint.Diagnostic, text string, sections map[earslint.Span]string) string {
	unresolved := isUnresolvedCode(d.Code)
	if (d.Code != "expr.unknown_term" && !unresolved) || d.Span == nil {
		return d.Message
	}
	section, ok := sections[*d.Span]
	if !ok || !spanWithin(*d.Span, len(text)) {
		return d.Message
	}
	term := strings.TrimSpace(text[d.Span.Start:d.Span.End])
	hint := fmt.Sprintf("; add it to catalog.%s (or an alias of an existing entry)", section)
	if unresolved {
		return fmt.Sprintf("unresolved catalog term %q%s", term, hint)
	}
	return fmt.Sprintf("unknown term %q in expression%s", term, hint)
}

// isUnresolvedCode reports whether a lint code is catalog.<role>_unresolved.
// TRLC-LINKS: REQ-EMG-123
func isUnresolvedCode(code string) bool {
	return strings.HasPrefix(code, "catalog.") && strings.HasSuffix(code, "_unresolved")
}

// spanWithin reports whether a span is a non-empty range inside a text of the given length.
// TRLC-LINKS: REQ-EMG-123
func spanWithin(span earslint.Span, length int) bool {
	return span.Start >= 0 && span.End <= length && span.Start < span.End
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func isBlockingCatalogDiagnostic(code string) bool {
	if code == "expr.unknown_term" {
		return true
	}
	return isUnresolvedCode(code)
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
	walkModelDocuments(bundle, func(s string) {
		for _, tok := range strings.FieldsFunc(s, isNotIDRune) {
			used[tok] = true
		}
	})
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
	if v.Kind() == reflect.String {
		visit(v.String())
		return
	}
	for _, child := range childValues(v) {
		walkStrings(child, depth+1, visit)
	}
}

// childValues are the values one level below v that walkStrings descends into.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func childValues(v reflect.Value) []reflect.Value {
	switch v.Kind() {
	case reflect.Pointer, reflect.Interface:
		return indirectValues(v)
	case reflect.Struct:
		return exportedFieldValues(v)
	case reflect.Slice, reflect.Array:
		return elementValues(v)
	case reflect.Map:
		return mapEntryValues(v)
	}
	return nil
}

// indirectValues is the target of a non-nil pointer or interface.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func indirectValues(v reflect.Value) []reflect.Value {
	if v.IsNil() {
		return nil
	}
	return []reflect.Value{v.Elem()}
}

// exportedFieldValues are the exported fields of a struct.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func exportedFieldValues(v reflect.Value) []reflect.Value {
	var out []reflect.Value
	for i := 0; i < v.NumField(); i++ {
		if v.Type().Field(i).IsExported() {
			out = append(out, v.Field(i))
		}
	}
	return out
}

// elementValues are the elements of a slice or array.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func elementValues(v reflect.Value) []reflect.Value {
	out := make([]reflect.Value, 0, v.Len())
	for i := 0; i < v.Len(); i++ {
		out = append(out, v.Index(i))
	}
	return out
}

// mapEntryValues are the keys and values of a map, each key before its value.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func mapEntryValues(v reflect.Value) []reflect.Value {
	var out []reflect.Value
	iter := v.MapRange()
	for iter.Next() {
		out = append(out, iter.Key(), iter.Value())
	}
	return out
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func isNotIDRune(r rune) bool {
	return !(isIDPunctuation(r) || unicode.IsLetter(r) || unicode.IsDigit(r))
}

// isIDPunctuation reports whether r is the hyphen, underscore or dot of an id.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func isIDPunctuation(r rune) bool {
	return r == '-' || r == '_' || r == '.'
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
	waived := waivedKeys(waivers)
	out := make([]validate.Diagnostic, 0, len(diags))
	for _, d := range diags {
		if !isWaivedWarning(d, waived) {
			out = append(out, d)
		}
	}
	return out
}

// waivedKeys is the set of "requirement|code" keys of the waivers.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func waivedKeys(waivers map[string][]string) map[string]bool {
	waived := map[string]bool{}
	for req, codes := range waivers {
		for _, c := range codes {
			waived[strings.TrimSpace(req)+"|"+strings.TrimSpace(c)] = true
		}
	}
	return waived
}

// isWaivedWarning reports whether a non-error diagnostic of a requirement path is waived.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func isWaivedWarning(d validate.Diagnostic, waived map[string]bool) bool {
	if d.Severity == validate.SeverityError {
		return false
	}
	id, ok := requirementIDFromPath(d.Path)
	return ok && waived[id+"|"+d.Code]
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

// fieldListPrepositionRe finds the preposition that opens a field list in a
// response: with, including or of as a whole word.
var fieldListPrepositionRe = regexp.MustCompile(`\b(with|including|of)\b`)

// fieldListSplitRe splits a field list into its items at commas and at the
// whole words and and or.
var fieldListSplitRe = regexp.MustCompile(`\s*,\s*|\s+\b(?:and|or)\b\s+`)

// fieldListDeterminers open a noun phrase. A list item whose second word is
// one ("notify the operator", "close a valve") reads as a verb with its
// object, that is a second response.
var fieldListDeterminers = map[string]bool{"the": true, "a": true, "an": true, "its": true, "their": true, "each": true, "every": true, "all": true}

// responseConjunctionsOnlyInFieldList reports whether every "and", "or" and
// comma of a requirement's single response sits in a field list after one
// response verb ("shall publish one message with UTC time, latitude and
// altitude"), so lint.multiple_responses is a false positive. The rule: the
// text has exactly one "shall" and no ";"; the response's first with,
// including or of opens the list; no "and", "or" or comma comes before it;
// the list holds no "then"; and no list item after the first reads
// as a verb with its object (second word the, a, an, its, their, each, every
// or all). "with the timestamp and notify the operator" and "stop all of the
// motors and close the valve" stay reported.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-072
func responseConjunctionsOnlyInFieldList(text string) bool {
	before, list, ok := splitFieldList(text)
	if !ok || conjunctionBeforeList(before) || containsPhrase(list, "then") {
		return false
	}
	return !hasVerbItem(list)
}

// splitFieldList splits the single response of a requirement at the preposition
// that opens its field list; false when the text has not exactly one "shall" or
// has a ";" or no preposition.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-072
func splitFieldList(text string) (string, string, bool) {
	lower := strings.ToLower(strings.TrimSpace(text))
	if strings.Count(" "+lower+" ", " shall ") != 1 || strings.Contains(lower, ";") {
		return "", "", false
	}
	_, response, _ := strings.Cut(lower, " shall ")
	response = strings.TrimSuffix(strings.TrimSpace(response), ".")
	loc := fieldListPrepositionRe.FindStringIndex(response)
	if loc == nil {
		return "", "", false
	}
	return response[:loc[0]], strings.TrimSpace(response[loc[1]:]), true
}

// conjunctionBeforeList reports whether a comma, "and" or "or" comes before the field list.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-072
func conjunctionBeforeList(before string) bool {
	return strings.Contains(before, ",") || containsPhrase(before, "and") || containsPhrase(before, "or")
}

// hasVerbItem reports whether a list item after the first reads as a verb with
// its object: its second word is a determiner.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
// TRLC-LINKS: REQ-EMG-072
func hasVerbItem(list string) bool {
	for i, item := range fieldListSplitRe.Split(list, -1) {
		if words := strings.Fields(item); i > 0 && len(words) > 1 && fieldListDeterminers[words[1]] {
			return true
		}
	}
	return false
}
