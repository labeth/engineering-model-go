// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel

import (
	"testing"

	"github.com/labeth/engineering-model-go/model"
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
