package engmodel

import (
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

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

func TestCatalogSystems_AppearInTermsAndRegistry(t *testing.T) {
	doc := model.CatalogDocument{
		Catalog: model.CatalogGroups{
			Systems: []model.CatalogEntry{
				{
					ID:         "SYS-COFFEE-FLEET-SYSTEM",
					Name:       "coffee fleet system",
					Definition: "System-of-interest for connected coffee machines and cloud control.",
					Aliases:    []string{"the coffee fleet system"},
				},
			},
		},
	}

	terms := buildTermsFromCatalog(doc)
	foundTerm := false
	for _, term := range terms {
		if term.ID == "SYS-COFFEE-FLEET-SYSTEM" {
			foundTerm = true
			break
		}
	}
	if !foundTerm {
		t.Fatalf("expected system term in terms and definitions")
	}

	refs := buildCatalogReferences(doc)
	foundSystemRef := false
	foundAliasRef := false
	for _, ref := range refs {
		if ref.ID == "SYS-COFFEE-FLEET-SYSTEM" && ref.Kind == "Catalog System" {
			foundSystemRef = true
		}
		if ref.ID == "the coffee fleet system" && ref.Kind == "Catalog Alias" {
			foundAliasRef = true
		}
	}
	if !foundSystemRef {
		t.Fatalf("expected system entry in catalog registry")
	}
	if !foundAliasRef {
		t.Fatalf("expected system alias entry in catalog registry")
	}
}

func TestBuiltInTerms_IncludeNewAuthoredConcepts(t *testing.T) {
	terms := buildTermsFromCatalog(model.CatalogDocument{})
	seen := map[string]bool{}
	for _, term := range terms {
		seen[term.ID] = true
	}
	for _, id := range []string{
		"EM-INTERFACE",
		"EM-DATA-OBJECT",
		"EM-DEPLOYMENT-TARGET",
		"EM-CONTROL",
		"EM-TRUST-BOUNDARY",
		"EM-STATE",
		"EM-EVENT",
		"EM-FLOW",
		"EM-FLOW-STEP",
	} {
		if !seen[id] {
			t.Fatalf("expected built-in term %s to be present", id)
		}
	}
}
