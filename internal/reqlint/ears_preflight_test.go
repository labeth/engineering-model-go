// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func TestLintRequirementsEARS_UnresolvedCatalogTermsAreBlocking(t *testing.T) {
	requirements := model.RequirementsDocument{
		LintRun: model.LintRun{
			ID:         "lint-blocking-check",
			Mode:       "guided",
			CommaAsAnd: true,
		},
		Requirements: []model.Requirement{
			{
				ID:   "REQ-TST-001",
				Text: "If unknown outage event is received, then the coffee fleet system shall publish brew telemetry.",
			},
		},
	}
	catalog := model.CatalogDocument{
		Catalog: model.CatalogGroups{
			Systems: []model.CatalogEntry{
				{ID: "SYS-COFFEE-FLEET-SYSTEM", Name: "coffee fleet system"},
			},
			DataTerms: []model.CatalogEntry{
				{ID: "DATA-BREW-TELEMETRY", Name: "brew telemetry"},
			},
		},
	}

	diags := lintRequirementsEARS(requirements, catalog)
	if len(diags) == 0 {
		t.Fatalf("expected diagnostics")
	}

	if !hasCodeAtSeverity(diags, "catalog.event_unresolved", validate.SeverityError) {
		t.Fatalf("expected catalog.event_unresolved to be blocking error")
	}
	if !hasCodeAtSeverity(diags, "expr.unknown_term", validate.SeverityError) {
		t.Fatalf("expected expr.unknown_term to be blocking error")
	}
}

// hasCodeAtSeverity reports whether a diagnostic has the code and severity.
func hasCodeAtSeverity(diags []validate.Diagnostic, code string, severity validate.Severity) bool {
	for _, d := range diags {
		if d.Code == code && d.Severity == severity {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func TestLintRequirementsEARS_UnreferencedCatalogTermsWarnInStrict(t *testing.T) {
	requirements := model.RequirementsDocument{
		LintRun: model.LintRun{
			ID:         "lint-coverage-strict",
			Mode:       "strict",
			CommaAsAnd: true,
		},
		Requirements: []model.Requirement{
			{
				ID:   "REQ-TST-002",
				Text: "When telemetry ingested event is received, the coffee fleet system shall publish brew telemetry.",
			},
		},
	}
	catalog := model.CatalogDocument{
		Catalog: model.CatalogGroups{
			Systems: []model.CatalogEntry{
				{ID: "SYS-COFFEE-FLEET-SYSTEM", Name: "coffee fleet system"},
			},
			Actors: []model.CatalogEntry{
				{ID: "ACT-SECURITY-ANALYST", Name: "security analyst"},
			},
			Events: []model.CatalogEntry{
				{ID: "EVT-TELEMETRY-INGESTED", Name: "telemetry ingested event is received"},
			},
			DataTerms: []model.CatalogEntry{
				{ID: "DATA-BREW-TELEMETRY", Name: "brew telemetry"},
			},
		},
	}

	diags := lintRequirementsEARS(requirements, catalog)
	if !hasCodeAtSeverity(diags, "catalog.term_unreferenced", validate.SeverityWarning) {
		t.Fatalf("expected catalog.term_unreferenced warning, got: %+v", diags)
	}
}

// catalog.term_unreferenced is a tier-keyed code (ADR-EMG-073): the lint emits a
// warning and the tier policy keeps it a warning below product and makes it an
// error from product; it is never an untiered error, which would stop the
// AsciiDoc render.
// Mutation: dropping its row from tierTable leaves it a warning at product;
// forcing an error in the lint makes it an error at prototype.
// TRLC-LINKS: REQ-EMG-183
func TestLintRequirementsEARS_UnreferencedCatalogTermIsTierKeyed(t *testing.T) {
	requirements := model.RequirementsDocument{
		LintRun: model.LintRun{ID: "lint-tier", Mode: "strict", CommaAsAnd: true},
		Requirements: []model.Requirement{
			{ID: "REQ-TST-003", Text: "When telemetry ingested event is received, the coffee fleet system shall publish brew telemetry."},
		},
	}
	catalog := model.CatalogDocument{Catalog: model.CatalogGroups{
		Systems:   []model.CatalogEntry{{ID: "SYS-COFFEE-FLEET-SYSTEM", Name: "coffee fleet system"}},
		Actors:    []model.CatalogEntry{{ID: "ACT-SECURITY-ANALYST", Name: "security analyst"}},
		Events:    []model.CatalogEntry{{ID: "EVT-TELEMETRY-INGESTED", Name: "telemetry ingested event is received"}},
		DataTerms: []model.CatalogEntry{{ID: "DATA-BREW-TELEMETRY", Name: "brew telemetry"}},
	}}
	diags := lintRequirementsEARS(requirements, catalog)
	for _, c := range []struct {
		tier string
		want validate.Severity
	}{
		{"poc", validate.SeverityWarning}, {"prototype", validate.SeverityWarning}, {"mvp", validate.SeverityWarning},
		{"product", validate.SeverityError}, {"operational", validate.SeverityError},
	} {
		if got := termSeveritiesAtTier(diags, c.tier); !allSeverity(got, c.want) {
			t.Errorf("at %s: severities %v, want one or more %s", c.tier, got, c.want)
		}
	}
}

// allSeverity reports whether got is not empty and every severity is want.
// TRLC-LINKS: REQ-EMG-183
func allSeverity(got []validate.Severity, want validate.Severity) bool {
	for _, sev := range got {
		if sev != want {
			return false
		}
	}
	return len(got) > 0
}

// termSeveritiesAtTier returns the severity of each catalog.term_unreferenced
// diagnostic after the tier policy of a model at the tier.
// TRLC-LINKS: REQ-EMG-183
func termSeveritiesAtTier(diags []validate.Diagnostic, tier string) []validate.Severity {
	var b model.Bundle
	b.Manifest.Lifecycle = &model.ManifestLifecycle{Tier: tier}
	var out []validate.Severity
	for _, d := range validate.ApplyTierPolicy(b, diags) {
		if d.Code == "catalog.term_unreferenced" {
			out = append(out, d.Severity)
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func TestLintRequirementsEARS_UnreferencedCatalogTermsNotEmittedInGuided(t *testing.T) {
	requirements := model.RequirementsDocument{
		LintRun: model.LintRun{
			ID:         "lint-coverage-guided",
			Mode:       "guided",
			CommaAsAnd: true,
		},
		Requirements: []model.Requirement{
			{
				ID:   "REQ-TST-003",
				Text: "When telemetry ingested event is received, the coffee fleet system shall publish brew telemetry.",
			},
		},
	}
	catalog := model.CatalogDocument{
		Catalog: model.CatalogGroups{
			Systems: []model.CatalogEntry{
				{ID: "SYS-COFFEE-FLEET-SYSTEM", Name: "coffee fleet system"},
			},
			Actors: []model.CatalogEntry{
				{ID: "ACT-SECURITY-ANALYST", Name: "security analyst"},
			},
			Events: []model.CatalogEntry{
				{ID: "EVT-TELEMETRY-INGESTED", Name: "telemetry ingested event is received"},
			},
			DataTerms: []model.CatalogEntry{
				{ID: "DATA-BREW-TELEMETRY", Name: "brew telemetry"},
			},
		},
	}

	diags := lintRequirementsEARS(requirements, catalog)
	for _, d := range diags {
		if d.Code == "catalog.term_unreferenced" {
			t.Fatalf("did not expect catalog.term_unreferenced in guided mode, got: %+v", diags)
		}
	}
}

// A catalog term the model references by id (here an actor owning a risk and
// an event in behavior) is in use even when no requirement text names it; only
// terms used nowhere are reported.
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func TestLintModelRequirementsEARS_SuppressesTermsUsedInModel(t *testing.T) {
	var bundle model.Bundle
	bundle.Catalog = model.CatalogDocument{Catalog: model.CatalogGroups{
		Systems:   []model.CatalogEntry{{ID: "SYS-COFFEE-FLEET-SYSTEM", Name: "coffee fleet system"}},
		Actors:    []model.CatalogEntry{{ID: "ACT-SECURITY-ANALYST", Name: "security analyst"}, {ID: "ACT-UNUSED", Name: "unused person"}},
		Events:    []model.CatalogEntry{{ID: "EVT-TELEMETRY-INGESTED", Name: "telemetry ingested event is received"}, {ID: "EVT-REBOOT", Name: "reboot event"}},
		DataTerms: []model.CatalogEntry{{ID: "DATA-BREW-TELEMETRY", Name: "brew telemetry"}},
	}}
	bundle.Assurance.Assurance.Risks = []model.Risk{{ID: "RISK-1", Owner: "ACT-SECURITY-ANALYST"}}
	bundle.Behavior.Behavior.Events = []model.Event{{ID: "EVT-REBOOT", Name: "Reboot"}}
	requirements := model.RequirementsDocument{
		LintRun: model.LintRun{ID: "lint-coverage-model", Mode: "strict", CommaAsAnd: true},
		Requirements: []model.Requirement{{
			ID:   "REQ-TST-004",
			Text: "When telemetry ingested event is received, the coffee fleet system shall publish brew telemetry.",
		}},
	}

	reported := map[string]bool{}
	for id := range unreferencedMessages(LintModelRequirementsEARS(bundle, requirements), "ACT-SECURITY-ANALYST", "ACT-UNUSED", "EVT-REBOOT") {
		reported[id] = true
	}
	if reported["ACT-SECURITY-ANALYST"] || reported["EVT-REBOOT"] {
		t.Fatalf("terms referenced in the model were reported: %v", reported)
	}
	if !reported["ACT-UNUSED"] {
		t.Fatalf("term used nowhere must still be reported: %v", reported)
	}
}

// unreferencedMessages maps each of the ids to the message of the
// catalog.term_unreferenced diagnostic that names it.
func unreferencedMessages(diags []validate.Diagnostic, ids ...string) map[string]string {
	out := map[string]string{}
	for _, d := range diags {
		if d.Code != "catalog.term_unreferenced" {
			continue
		}
		for _, id := range idsNamedIn(d.Message, ids) {
			out[id] = d.Message
		}
	}
	return out
}

// idsNamedIn lists the ids the message names in parentheses.
func idsNamedIn(message string, ids []string) []string {
	var named []string
	for _, id := range ids {
		if strings.Contains(message, "("+id+")") {
			named = append(named, id)
		}
	}
	return named
}

// A catalog term whose name appears in any authored model text other than its
// own entry (requirement notes or verification criteria, another catalog
// entry's definition, an architecture, behavior or assurance description) is
// in use; a term named only in its own definition is still reported, with a
// message that says "anywhere in the model".
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func TestLintModelRequirementsEARS_TermNamedInModelTextIsReferenced(t *testing.T) {
	bundle, requirements := termNamedFixture()
	var ids []string
	for _, e := range bundle.Catalog.Catalog.DataTerms {
		ids = append(ids, e.ID)
	}
	reported := unreferencedMessages(LintModelRequirementsEARS(bundle, requirements), ids...)
	for _, id := range []string{"DATA-OUTPUT-RATE", "DATA-NOTES-ONLY", "DATA-CRITERIA-ONLY", "DATA-ARCH-ONLY", "DATA-BEHAVIOR-ONLY", "DATA-ASSURANCE-ONLY"} {
		if msg, ok := reported[id]; ok {
			t.Errorf("%s is named in model text but reported: %s", id, msg)
		}
	}
	msg, ok := reported["DATA-SELF-ONLY"]
	if !ok {
		t.Fatalf("term named only in its own entry must be reported: %v", reported)
	}
	if !strings.Contains(msg, "anywhere in the model") || strings.Contains(msg, "requirement text") {
		t.Fatalf("message = %q, want it to say anywhere in the model", msg)
	}
}

// termNamedFixture is a model whose catalog terms are named in different model
// texts, with the requirements that publish brew telemetry.
func termNamedFixture() (model.Bundle, model.RequirementsDocument) {
	var bundle model.Bundle
	bundle.Catalog = model.CatalogDocument{Catalog: model.CatalogGroups{
		Systems: []model.CatalogEntry{{ID: "SYS-COFFEE-FLEET-SYSTEM", Name: "coffee fleet system"}},
		Events:  []model.CatalogEntry{{ID: "EVT-TELEMETRY-INGESTED", Name: "telemetry ingested event is received", Definition: "One output period, the inverse of the output rate, has passed."}},
		DataTerms: []model.CatalogEntry{
			{ID: "DATA-BREW-TELEMETRY", Name: "brew telemetry"},
			{ID: "DATA-OUTPUT-RATE", Name: "output rate"},
			{ID: "DATA-NOTES-ONLY", Name: "grind size"},
			{ID: "DATA-CRITERIA-ONLY", Name: "water hardness"},
			{ID: "DATA-ARCH-ONLY", Name: "boiler pressure"},
			{ID: "DATA-BEHAVIOR-ONLY", Name: "descale cycle"},
			{ID: "DATA-ASSURANCE-ONLY", Name: "tamper seal"},
			{ID: "DATA-SELF-ONLY", Name: "milk froth", Definition: "The milk froth level."},
		},
	}}
	bundle.Architecture.AuthoredArchitecture.FunctionalUnits = []model.FunctionalUnit{{ID: "FU-BOILER", Name: "Boiler", Prose: "Holds the Boiler Pressure."}}
	bundle.Behavior.Behavior.Events = []model.Event{{ID: "EVT-X", Name: "Starts the descale cycle"}}
	bundle.Assurance.Assurance.Risks = []model.Risk{{ID: "RISK-1", Statement: "A broken tamper seal goes unnoticed."}}
	requirements := model.RequirementsDocument{
		LintRun: model.LintRun{ID: "lint-coverage-text", Mode: "strict", CommaAsAnd: true},
		Requirements: []model.Requirement{{
			ID:                   "REQ-TST-005",
			Text:                 "When telemetry ingested event is received, the coffee fleet system shall publish brew telemetry.",
			Notes:                "The grind size is fixed.",
			VerificationCriteria: "Measured at nominal water hardness.",
		}},
	}
	return bundle, requirements
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func TestSuppressWaivedDiagnostics(t *testing.T) {
	diags := []validate.Diagnostic{
		{Code: "lint.multiple_responses", Severity: validate.SeverityWarning, Path: "requirements[REQ-A]"},
		{Code: "lint.multiple_responses", Severity: validate.SeverityWarning, Path: "requirements[REQ-A]@3:9"},
		{Code: "lint.multiple_responses", Severity: validate.SeverityWarning, Path: "requirements[REQ-B]"},
		{Code: "lint.other", Severity: validate.SeverityWarning, Path: "requirements[REQ-A]"},
		{Code: "expr.unknown_term", Severity: validate.SeverityError, Path: "requirements[REQ-A]"},
		{Code: "lint.multiple_responses", Severity: validate.SeverityWarning, Path: "requirements"},
	}
	waivers := map[string][]string{"REQ-A": {"lint.multiple_responses", "expr.unknown_term"}}
	got := SuppressWaivedDiagnostics(diags, waivers)
	if len(got) != 4 {
		t.Fatalf("got %d diagnostics, want 4: %+v", len(got), got)
	}
	assertWaivedRemoved(t, got)
	if !validate.HasErrors(got) {
		t.Fatalf("an error must never be waived")
	}
	if len(SuppressWaivedDiagnostics(diags, nil)) != len(diags) {
		t.Fatalf("no waivers must keep every diagnostic")
	}
}

// assertWaivedRemoved checks that no waived multiple-responses diagnostic of REQ-A is kept.
func assertWaivedRemoved(t *testing.T, got []validate.Diagnostic) {
	t.Helper()
	for _, d := range got {
		if d.Path != "requirements" && strings.HasPrefix(d.Path, "requirements[REQ-A]") && d.Code == "lint.multiple_responses" {
			t.Fatalf("waived diagnostic kept: %+v", d)
		}
	}
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func TestLintRequirementsEARS_HonorsLintWaiversWithReason(t *testing.T) {
	catalog := model.CatalogDocument{Catalog: model.CatalogGroups{
		Systems:   []model.CatalogEntry{{ID: "SYS-SIM", Name: "simulator"}},
		DataTerms: []model.CatalogEntry{{ID: "DATA-SENTENCE", Name: "sentence"}},
	}}
	text := "The simulator shall write the sentence and then the simulator shall flush the sentence."
	base := model.RequirementsDocument{LintRun: model.LintRun{Mode: "strict", CommaAsAnd: true},
		Requirements: []model.Requirement{{ID: "REQ-SIM-001", Text: text}}}
	before := simulatorCodes(base, catalog)
	if len(before) == 0 {
		t.Skip("fixture text no longer produces a requirement-scoped lint warning")
	}
	waived := base
	waived.Requirements = []model.Requirement{{ID: "REQ-SIM-001", Text: text, LintWaivers: lintWaiversFor(before)}}
	after := simulatorCodes(waived, catalog)
	for code := range before {
		if strings.HasPrefix(code, "lint.") && after[code] {
			t.Fatalf("waived %s still reported", code)
		}
	}
	assertWaiverNeedsReason(t, text)
}

// lintWaiversFor waives each lint.* code of the set with a reason.
func lintWaiversFor(codes map[string]bool) []model.LintWaiver {
	var waivers []model.LintWaiver
	for code := range codes {
		if strings.HasPrefix(code, "lint.") {
			waivers = append(waivers, model.LintWaiver{Code: code, Reason: "two actions are one atomic write"})
		}
	}
	return waivers
}

// simulatorCodes is the set of lint codes reported for requirement REQ-SIM-001.
func simulatorCodes(doc model.RequirementsDocument, catalog model.CatalogDocument) map[string]bool {
	out := map[string]bool{}
	for _, d := range lintRequirementsEARS(doc, catalog) {
		if strings.HasPrefix(d.Path, "requirements[REQ-SIM-001]") {
			out[d.Code] = true
		}
	}
	return out
}

// assertWaiverNeedsReason checks that validation rejects a lint waiver without a reason.
func assertWaiverNeedsReason(t *testing.T, text string) {
	t.Helper()
	noReason := model.RequirementsDocument{LintRun: model.LintRun{Mode: "strict", CommaAsAnd: true},
		Requirements: []model.Requirement{{ID: "REQ-SIM-001", Text: text, LintWaivers: []model.LintWaiver{{Code: "lint.multiple_responses"}}}}}
	diags := validate.Bundle(model.Bundle{Requirements: noReason})
	for _, d := range diags {
		if d.Code == "model.missing_lint_waiver_reason" {
			return
		}
	}
	t.Fatalf("waiver without reason must be rejected, got %+v", diags)
}

// lint.multiple_responses is not reported for one response verb whose
// conjunctions all sit in a trailing with, of or including field list; a
// conjunction before the list, a second shall, or a "then" clause in the list
// is still reported.
// Mutation: dropping the post-filter, the single-shall check, the
// before-list check, the then check or the verb-with-object item check fails
// this test.
// TRLC-LINKS: REQ-EMG-072
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestLintMultipleResponsesIgnoresFieldListAfterOneVerb(t *testing.T) {
	catalog := model.CatalogDocument{Catalog: model.CatalogGroups{
		Systems: []model.CatalogEntry{{ID: "SYS-SIM", Name: "simulator"}},
	}}
	cases := map[string]bool{
		"The simulator shall publish one world state message with UTC time, latitude, longitude and altitude.": false,
		"The simulator shall return the result including the schema version and the error fields.":             false,
		"The simulator shall publish a report of errors, warnings and notes.":                                  false,
		"The simulator shall log the event with the timestamp and notify the operator.":                        true,
		"The simulator shall log the event and notify the operator.":                                           true,
		"The simulator shall log the event, then notify the operator with the alarm text.":                     true,
		"The simulator shall send the message with the time and then flush the queue.":                         true,
		"The simulator shall send the message with the time and the simulator shall flush the queue.":          true,
		"The simulator shall send the message with the time and then the simulator shall flush the queue.":     true,
		"The simulator shall send the message with the time; the simulator shall flush the queue.":             true,
	}
	for text, want := range cases {
		doc := model.RequirementsDocument{LintRun: model.LintRun{Mode: "strict"}, Requirements: []model.Requirement{{ID: "REQ-SIM-001", Text: text}}}
		if got := reportsMultipleResponses(doc, catalog); got != want {
			t.Errorf("%q: lint.multiple_responses reported=%v, want %v", text, got, want)
		}
	}
}

// reportsMultipleResponses reports whether the lint of the document reports lint.multiple_responses.
func reportsMultipleResponses(doc model.RequirementsDocument, catalog model.CatalogDocument) bool {
	for _, d := range lintRequirementsEARS(doc, catalog) {
		if d.Code == "lint.multiple_responses" {
			return true
		}
	}
	return false
}

// An unknown term in an EARS clause names the term and the catalog section
// it belongs in, derived from its clause position (the paired
// catalog.<role>_unresolved at the same span): a When clause term in
// catalog.events, a While clause term in catalog.states, both on
// expr.unknown_term and on the catalog diagnostic, so the author knows
// where to add it.
// Mutation: dropping the section or the term from either message fails
// this test.
// TRLC-LINKS: REQ-EMG-123
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestUnknownTermNamesItsCatalogSection(t *testing.T) {
	requirements := model.RequirementsDocument{
		LintRun: model.LintRun{Mode: "strict", CommaAsAnd: true},
		Requirements: []model.Requirement{
			{ID: "REQ-A", Text: "When pump starts, the coffee fleet system shall record brew telemetry."},
			{ID: "REQ-B", Text: "While brewing mode is active, the coffee fleet system shall record brew telemetry."},
		},
	}
	catalog := model.CatalogDocument{Catalog: model.CatalogGroups{
		Systems:   []model.CatalogEntry{{ID: "SYS-C", Name: "coffee fleet system"}},
		DataTerms: []model.CatalogEntry{{ID: "DATA-B", Name: "brew telemetry"}},
	}}
	got := map[string]string{}
	for _, d := range lintRequirementsEARS(requirements, catalog) {
		got[d.Code+" "+d.Path] = d.Message
	}
	for key, want := range map[string]string{
		"expr.unknown_term requirements[REQ-A]@5:16":        `unknown term "pump starts" in expression; add it to catalog.events (or an alias of an existing entry)`,
		"catalog.event_unresolved requirements[REQ-A]@5:16": `unresolved catalog term "pump starts"; add it to catalog.events (or an alias of an existing entry)`,
		"expr.unknown_term requirements[REQ-B]@6:28":        `unknown term "brewing mode is active" in expression; add it to catalog.states (or an alias of an existing entry)`,
	} {
		if got[key] != want {
			t.Errorf("%s = %q, want %q", key, got[key], want)
		}
	}
}
