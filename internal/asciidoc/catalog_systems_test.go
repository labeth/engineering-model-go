// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package asciidoc

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// hasTermID reports whether a term has the ID.
func hasTermID(terms []asciidocTerm, id string) bool {
	for _, term := range terms {
		if term.ID == id {
			return true
		}
	}
	return false
}

// hasReference reports whether a reference entry has the ID and the kind.
func hasReference(refs []asciidocReferenceEntry, id, kind string) bool {
	for _, ref := range refs {
		if ref.ID == id && ref.Kind == kind {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-001
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

	if !hasTermID(buildTermsFromCatalog(doc), "SYS-COFFEE-FLEET-SYSTEM") {
		t.Fatalf("expected system term in terms and definitions")
	}
	refs := buildCatalogReferences(doc)
	if !hasReference(refs, "SYS-COFFEE-FLEET-SYSTEM", "Catalog System") {
		t.Fatalf("expected system entry in catalog registry")
	}
	if !hasReference(refs, "the coffee fleet system", "Catalog Alias") {
		t.Fatalf("expected system alias entry in catalog registry")
	}
}

// TRLC-LINKS: REQ-EMG-001
func TestBuiltInTerms_IncludeNewAuthoredConcepts(t *testing.T) {
	terms := buildTermsFromCatalog(model.CatalogDocument{})
	seen := map[string]bool{}
	for _, term := range terms {
		seen[term.ID] = true
	}
	for _, id := range []string{
		"TERM-INTERFACE",
		"TERM-DATA-OBJECT",
		"TERM-DEPLOYMENT-TARGET",
		"TERM-CONTROL",
		"TERM-TRUST-BOUNDARY",
		"TERM-STATE",
		"TERM-EVENT",
		"TERM-FLOW",
		"TERM-FLOW-STEP",
	} {
		if !seen[id] {
			t.Fatalf("expected built-in term %s to be present", id)
		}
	}
}
