// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// A clause-term catalog name or alias holding a standalone and, or, not, a
// comma or a parenthesis is one warning at that catalog field; a word that
// only contains an operator (android, notify, order) and a system name, which
// EARS matches whole, are not reported.
// Mutation: dropping the word-boundary check reports "notify operator";
// dropping the comma or parenthesis check misses those entries.
// TRLC-LINKS: REQ-EMG-071
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestCatalogNameHasOperatorWarnsAtTheCatalogEntry(t *testing.T) {
	b := schemaV2TestBundle(model.Bundle{Catalog: model.CatalogDocument{Catalog: model.CatalogGroups{
		Systems: []model.CatalogEntry{{ID: "SYS-A", Name: "validation and analysis", Definition: "d"}},
		States: []model.CatalogEntry{
			{ID: "STATE-A", Name: "entity is not selected", Definition: "d"},
			{ID: "STATE-B", Name: "android order notify operator", Definition: "d"},
		},
		Events: []model.CatalogEntry{
			{ID: "EVT-A", Name: "tick (periodic) is received", Definition: "d"},
			{ID: "EVT-B", Name: "start is received", Definition: "d", Aliases: []string{"start", "start, restart"}},
		},
		Features: []model.CatalogEntry{{ID: "FEAT-A", Name: "Import OR export is enabled", Definition: "d"}},
	}}})
	var got []Diagnostic
	for _, d := range Bundle(b) {
		if d.Code == "catalog.name_has_operator" {
			got = append(got, d)
		}
	}
	want := map[string]bool{
		"catalog.states[0].name":       true,
		"catalog.events[0].name":       true,
		"catalog.events[1].aliases[1]": true,
		"catalog.features[0].name":     true,
	}
	if len(got) != len(want) {
		t.Fatalf("want %d catalog.name_has_operator warnings, got %+v", len(want), got)
	}
	for _, d := range got {
		if !want[d.Path] || d.Severity != SeverityWarning {
			t.Fatalf("unexpected catalog.name_has_operator %+v", d)
		}
	}
}
