// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"reflect"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

func sampleUsage(t *testing.T) map[string]TermUsage {
	t.Helper()
	var b model.Bundle
	b.Catalog.Catalog.Events = []model.CatalogEntry{{ID: "EVT-X", Name: "model updated event", Aliases: []string{"model refresh"}, Definition: "d"}}
	b.Catalog.Catalog.Features = []model.CatalogEntry{{ID: "FEAT-TRACE", Name: "trace"}, {ID: "FEAT-IDLE", Name: "nothing at all"}}
	b.Requirements.Requirements = []model.Requirement{
		{ID: "REQ-1", Text: "When model updated event is received, the system shall do it."},
		{ID: "REQ-2", Text: "When a model refresh occurs, the system shall do it.", Notes: "relates to EVT-X"},
		{ID: "REQ-3", Text: "The system shall trace everything."},
		{ID: "REQ-4", Text: "The system shall keep a tracer log."}, // "tracer" is a longer word
	}
	usage, err := CatalogUsage(b)
	if err != nil {
		t.Fatal(err)
	}
	out := map[string]TermUsage{}
	for _, u := range usage {
		out[u.ID] = u
	}
	return out
}

// A requirement names a term by its name or an alias on word boundaries, never by a longer word that contains it.
// Mutation: matching substrings or ignoring aliases fails this test.
// TRLC-LINKS: REQ-EMG-193
func TestCatalogUsageRequirementsNameTermsOnWordBoundaries(t *testing.T) {
	usage := sampleUsage(t)
	if got := usage["EVT-X"].Requirements; !reflect.DeepEqual(got, []string{"REQ-1", "REQ-2"}) {
		t.Errorf("EVT-X is named by its name and by an alias: %v", got)
	}
	if got := usage["FEAT-TRACE"].Requirements; !reflect.DeepEqual(got, []string{"REQ-3"}) {
		t.Errorf("only REQ-3 names trace as a word, not REQ-4 (tracer): %v", got)
	}
}

// An entity that holds the ID of a term, or mentions it in prose, references it.
// Mutation: dropping the reference edges fails this test.
// TRLC-LINKS: REQ-EMG-193
func TestCatalogUsageEntitiesReferenceTerms(t *testing.T) {
	usage := sampleUsage(t)
	if got := usage["EVT-X"].ReferencedBy; !reflect.DeepEqual(got, []string{"REQ-2"}) {
		t.Errorf("EVT-X is mentioned in the notes of REQ-2: %v", got)
	}
	idle := usage["FEAT-IDLE"]
	if idle.Requirements == nil || idle.ReferencedBy == nil || len(idle.Requirements)+len(idle.ReferencedBy) != 0 {
		t.Errorf("a term nothing uses has two empty, non-nil lists: %+v", idle)
	}
}

// countTerms returns the number of terms listed twice and the number that requirements name.
func countTerms(usage []TermUsage) (duplicates, named int) {
	seen := map[string]bool{}
	for _, u := range usage {
		if seen[u.ID] {
			duplicates++
		}
		seen[u.ID] = true
		if len(u.Requirements) > 0 {
			named++
		}
	}
	return duplicates, named
}

// The engine's own catalog: every term of the real model is listed once, and requirements name many of them.
// TRLC-LINKS: REQ-EMG-193
func TestCatalogUsageOfThisRepository(t *testing.T) {
	b, err := model.LoadBundle("../../engmod.yml")
	if err != nil {
		t.Fatal(err)
	}
	usage, err := CatalogUsage(b)
	if err != nil {
		t.Fatal(err)
	}
	duplicates, named := countTerms(usage)
	if duplicates != 0 || len(usage) < 50 || named < 20 {
		t.Errorf("%d terms, %d listed twice, %d named by requirements", len(usage), duplicates, named)
	}
}
