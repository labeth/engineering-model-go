package engmodel

import (
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

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
