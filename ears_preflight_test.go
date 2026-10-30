// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
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

	eventUnresolvedError := false
	unknownTermError := false
	for _, d := range diags {
		if d.Code == "catalog.event_unresolved" && d.Severity == "error" {
			eventUnresolvedError = true
		}
		if d.Code == "expr.unknown_term" && d.Severity == "error" {
			unknownTermError = true
		}
	}
	if !eventUnresolvedError {
		t.Fatalf("expected catalog.event_unresolved to be blocking error")
	}
	if !unknownTermError {
		t.Fatalf("expected expr.unknown_term to be blocking error")
	}
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
	found := false
	for _, d := range diags {
		if d.Code == "catalog.term_unreferenced" && d.Severity == "warning" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected catalog.term_unreferenced warning, got: %+v", diags)
	}
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
	for _, d := range lintModelRequirementsEARS(bundle, requirements) {
		if d.Code == "catalog.term_unreferenced" {
			for _, id := range []string{"ACT-SECURITY-ANALYST", "ACT-UNUSED", "EVT-REBOOT"} {
				if strings.Contains(d.Message, "("+id+")") {
					reported[id] = true
				}
			}
		}
	}
	if reported["ACT-SECURITY-ANALYST"] || reported["EVT-REBOOT"] {
		t.Fatalf("terms referenced in the model were reported: %v", reported)
	}
	if !reported["ACT-UNUSED"] {
		t.Fatalf("term used nowhere must still be reported: %v", reported)
	}
}

// A catalog term whose name appears in any authored model text other than its
// own entry (requirement notes or verification criteria, another catalog
// entry's definition, an architecture, behavior or assurance description) is
// in use; a term named only in its own definition is still reported, with a
// message that says "anywhere in the model".
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009
func TestLintModelRequirementsEARS_TermNamedInModelTextIsReferenced(t *testing.T) {
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
	reported := map[string]string{}
	for _, d := range lintModelRequirementsEARS(bundle, requirements) {
		if d.Code != "catalog.term_unreferenced" {
			continue
		}
		for _, e := range bundle.Catalog.Catalog.DataTerms {
			if strings.Contains(d.Message, "("+e.ID+")") {
				reported[e.ID] = d.Message
			}
		}
	}
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
	for _, d := range got {
		if d.Path != "requirements" && strings.HasPrefix(d.Path, "requirements[REQ-A]") && d.Code == "lint.multiple_responses" {
			t.Fatalf("waived diagnostic kept: %+v", d)
		}
	}
	if !validate.HasErrors(got) {
		t.Fatalf("an error must never be waived")
	}
	if len(SuppressWaivedDiagnostics(diags, nil)) != len(diags) {
		t.Fatalf("no waivers must keep every diagnostic")
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
	codes := func(doc model.RequirementsDocument) map[string]bool {
		out := map[string]bool{}
		for _, d := range lintRequirementsEARS(doc, catalog) {
			if strings.HasPrefix(d.Path, "requirements[REQ-SIM-001]") {
				out[d.Code] = true
			}
		}
		return out
	}
	before := codes(base)
	if len(before) == 0 {
		t.Skip("fixture text no longer produces a requirement-scoped lint warning")
	}
	waived := base
	waived.Requirements = []model.Requirement{{ID: "REQ-SIM-001", Text: text}}
	for code := range before {
		if strings.HasPrefix(code, "lint.") {
			waived.Requirements[0].LintWaivers = append(waived.Requirements[0].LintWaivers, model.LintWaiver{Code: code, Reason: "two actions are one atomic write"})
		}
	}
	after := codes(waived)
	for code := range before {
		if strings.HasPrefix(code, "lint.") && after[code] {
			t.Fatalf("waived %s still reported", code)
		}
	}
	noReason := base
	noReason.Requirements = []model.Requirement{{ID: "REQ-SIM-001", Text: text, LintWaivers: []model.LintWaiver{{Code: "lint.multiple_responses"}}}}
	diags := validate.Bundle(model.Bundle{Requirements: noReason})
	found := false
	for _, d := range diags {
		if d.Code == "model.missing_lint_waiver_reason" {
			found = true
		}
	}
	if !found {
		t.Fatalf("waiver without reason must be rejected, got %+v", diags)
	}
}
