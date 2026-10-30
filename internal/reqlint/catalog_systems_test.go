// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-001
func TestToEarsCatalog_IncludesCatalogSystems(t *testing.T) {
	doc := model.CatalogDocument{
		Catalog: model.CatalogGroups{
			Systems: []model.CatalogEntry{
				{ID: "SYS-COFFEE", Name: "coffee fleet system"},
			},
			FunctionalGroups: []model.CatalogEntry{
				{ID: "FG-EDGE", Name: "machine edge"},
			},
			FunctionalUnits: []model.CatalogEntry{
				{ID: "FU-INGEST", Name: "fleet ingestion api"},
			},
		},
	}

	c := toEarsCatalog(doc)
	if len(c.Systems) != 3 {
		t.Fatalf("expected 3 system entries (system + fg + fu), got %d", len(c.Systems))
	}
	if c.Systems[0].ID != "SYS-COFFEE" {
		t.Fatalf("expected explicit catalog system first, got %q", c.Systems[0].ID)
	}
}
