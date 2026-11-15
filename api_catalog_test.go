// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel_test

import (
	"reflect"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// usageByID is the catalog usage of a bundle by term ID; a term listed twice fails the test.
func usageByID(t *testing.T, bundle engmodel.Bundle) (map[string]engmodel.CatalogTermUsage, []engmodel.CatalogTermUsage) {
	t.Helper()
	list, err := engmodel.CatalogUsage(bundle)
	if err != nil {
		t.Fatalf("CatalogUsage: %v", err)
	}
	byID := map[string]engmodel.CatalogTermUsage{}
	for _, u := range list {
		if _, again := byID[u.ID]; again {
			t.Errorf("term %s is listed twice", u.ID)
		}
		byID[u.ID] = u
	}
	return byID, list
}

// A requirement names a term by its name or an alias and never by a longer word, an entity that holds the ID of a
// term references it, and a term nothing uses has two empty lists.
// Mutation: matching substrings, ignoring aliases or dropping the reference list fails this test.
// TRLC-LINKS: REQ-EMG-193, REQ-EMG-201
func TestCatalogUsageListsRequirementsAndReferencesOfEachTerm(t *testing.T) {
	bundle := linksModel(t, "child-a")
	bundle.Catalog.Catalog.Events = append([]engmodel.CatalogEntry(nil), bundle.Catalog.Catalog.Events...)
	bundle.Catalog.Catalog.Events[0].Aliases = []string{"epoch tick"}
	bundle.Catalog.Catalog.Features = []engmodel.CatalogEntry{{ID: "FEAT-IDLE", Name: "nothing at all", Definition: "Used by no one."}, {ID: "FEAT-COUNTER", Name: "counter"}}
	bundle.Requirements.Requirements = append([]engmodel.Requirement(nil), bundle.Requirements.Requirements...)
	bundle.Requirements.Requirements = append(bundle.Requirements.Requirements,
		engmodel.Requirement{ID: "REQ-A-100", Text: "When the epoch tick occurs, the position encoder shall publish one position sentence."},
		engmodel.Requirement{ID: "REQ-A-101", Text: "The position encoder shall keep a countersign log.", Notes: "relates to EVT-LIFECYCLE"},
		engmodel.Requirement{ID: "REQ-A-102", Text: "The position encoder shall increment a counter."},
	)
	usage, list := usageByID(t, bundle)
	cases := []struct {
		id           string
		requirements []string
		referencedBy []string
	}{
		{"EVT-EPOCH", []string{"REQ-A-001", "REQ-A-100"}, []string{}},
		{"EVT-LIFECYCLE", []string{"REQ-A-004"}, []string{"REQ-A-101"}},
		{"FU-ENCODER", []string{"REQ-A-002", "REQ-A-003"}, []string{"REQ-A-001", "REQ-A-002", "REQ-A-003", "REQ-A-004"}},
		{"FEAT-COUNTER", []string{"REQ-A-102"}, []string{}},
		{"FEAT-IDLE", []string{}, []string{}},
	}
	for _, c := range cases {
		t.Run(c.id, func(t *testing.T) {
			u, ok := usage[c.id]
			if !ok {
				t.Fatalf("%s is not listed", c.id)
			}
			if u.Requirements == nil || u.ReferencedBy == nil {
				t.Errorf("a list is nil, it must encode as []: %+v", u)
			}
			if !reflect.DeepEqual(u.Requirements, c.requirements) || !reflect.DeepEqual(u.ReferencedBy, c.referencedBy) {
				t.Errorf("requirements %v, referenced by %v; want %v and %v", u.Requirements, u.ReferencedBy, c.requirements, c.referencedBy)
			}
		})
	}
	if idle := usage["FEAT-IDLE"]; idle.Group != "features" || idle.Name != "nothing at all" || idle.Definition != "Used by no one." {
		t.Errorf("the term is listed with its group, name and definition: %+v", idle)
	}
	if got := len(list); got != 8 {
		t.Errorf("%d terms listed, want one per catalog entry of the model (8)", got)
	}
}

// The catalog of this repository: each term is listed once, requirements name many of them and the list is the same on every call.
// Mutation: listing a term twice or in another order fails this test.
// TRLC-LINKS: REQ-EMG-193, REQ-EMG-201
func TestCatalogUsageOfThisRepositoryListsEachTermOnce(t *testing.T) {
	bundle := selfBundle(t)
	usage, list := usageByID(t, bundle)
	named := 0
	for _, u := range list {
		if len(u.Requirements) > 0 {
			named++
		}
	}
	if len(usage) < 50 || named < 20 {
		t.Errorf("%d terms, %d named by requirements", len(usage), named)
	}
	for _, id := range []string{"FU-MODEL-LOADER", "SYS-ENGINEERING-MODEL-GO"} {
		if usage[id].Name == "" || usage[id].Group == "" {
			t.Errorf("%s: %+v", id, usage[id])
		}
	}
	again, err := engmodel.CatalogUsage(bundle)
	if err != nil || !reflect.DeepEqual(list, again) {
		t.Errorf("a second call returned another list (error %v)", err)
	}
}
