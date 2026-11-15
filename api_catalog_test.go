// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel_test

import (
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
		_, again := byID[u.ID]
		expect(t, !again, "term %s is listed twice", u.ID)
		byID[u.ID] = u
	}
	return byID, list
}

// catalogWithUsage is child-a with an alias, two more features and three more requirements: two name a term, one
// names it only inside a longer word.
func catalogWithUsage(t *testing.T) engmodel.Bundle {
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
	return bundle
}

// termUse is the requirements that name a term and the entities that reference it.
type termUse struct{ requirements, referencedBy []string }

// A requirement names a term by its name or an alias and never by a longer word, an entity that holds the ID of a
// term references it, and a term nothing uses has two empty lists.
// Mutation: matching substrings, ignoring aliases or dropping the reference list fails this test.
// TRLC-LINKS: REQ-EMG-193, REQ-EMG-201
func TestCatalogUsageListsRequirementsAndReferencesOfEachTerm(t *testing.T) {
	usage, list := usageByID(t, catalogWithUsage(t))
	cases := map[string]termUse{
		"EVT-EPOCH":     {[]string{"REQ-A-001", "REQ-A-100"}, []string{}},
		"EVT-LIFECYCLE": {[]string{"REQ-A-004"}, []string{"REQ-A-101"}},
		"FU-ENCODER":    {[]string{"REQ-A-002", "REQ-A-003"}, []string{"REQ-A-001", "REQ-A-002", "REQ-A-003", "REQ-A-004"}},
		"FEAT-COUNTER":  {[]string{"REQ-A-102"}, []string{}},
		"FEAT-IDLE":     {[]string{}, []string{}},
	}
	for id, want := range cases {
		t.Run(id, func(t *testing.T) {
			u := usage[id]
			check{t}.equal("requirements", u.Requirements, want.requirements)
			check{t}.equal("referenced by", u.ReferencedBy, want.referencedBy)
		})
	}
	idle := usage["FEAT-IDLE"]
	check{t}.equal("the term with its group, name and definition", [3]string{idle.Group, idle.Name, idle.Definition}, [3]string{"features", "nothing at all", "Used by no one."})
	check{t}.equal("number of terms, one per catalog entry", len(list), 8)
}

// The catalog of this repository: each term is listed once, requirements name many of them and the list is the same on every call.
// Mutation: listing a term twice or in another order fails this test.
// TRLC-LINKS: REQ-EMG-193, REQ-EMG-201
func TestCatalogUsageOfThisRepositoryListsEachTermOnce(t *testing.T) {
	bundle := selfBundle(t)
	usage, list := usageByID(t, bundle)
	named := 0
	for _, u := range list {
		named += min(1, len(u.Requirements))
	}
	expect(t, len(usage) >= 50 && named >= 20, "%d terms, %d named by requirements", len(usage), named)
	for _, id := range []string{"FU-MODEL-LOADER", "SYS-ENGINEERING-MODEL-GO"} {
		expect(t, usage[id].Name != "" && usage[id].Group != "", "%s: %+v", id, usage[id])
	}
	again, err := engmodel.CatalogUsage(bundle)
	expect(t, err == nil, "second call: %v", err)
	check{t}.equal("a second call", again, list)
}
