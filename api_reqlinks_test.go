// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel_test

import (
	"path/filepath"
	"slices"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// linksModel loads one model of the requirement links example.
func linksModel(t *testing.T, name string) engmodel.Bundle {
	t.Helper()
	isolatedModel(t)
	bundle, err := engmodel.LoadBundle(filepath.Join("examples", "requirement-links", "repos", name, "engmod.yml"))
	if err != nil {
		t.Fatalf("LoadBundle %s: %v", name, err)
	}
	return bundle
}

// realizerIDs lists the IDs of the requirements that realize an item.
func realizerIDs(item engmodel.ContractItem) []string {
	out := []string{}
	for _, r := range item.Realizers {
		out = append(out, r.ID)
	}
	return out
}

// contractItem is the item a model is expected to publish: its kind and the requirements that realize it.
type contractItem struct {
	kind      engmodel.ContractItemKind
	realizers []string
}

// ContractItemsFromBundle lists the capabilities and the assumptions of use of a model with the requirements that realize
// each, in requirement order.
// Mutation: dropping an item kind, a realizer or the requirement order fails this test.
// TRLC-LINKS: REQ-EMG-203
func TestContractItemsFromBundleListsItemsWithTheirRealizers(t *testing.T) {
	cases := map[string]map[string]contractItem{
		"child-a": {
			"CAP-A-1": {engmodel.ContractItemCapability, []string{"REQ-A-001", "REQ-A-002"}},
			"CAP-A-2": {engmodel.ContractItemCapability, []string{"REQ-A-003"}},
		},
		"child-b": {
			"CAP-B-1": {engmodel.ContractItemCapability, []string{"REQ-B-001"}},
			"AOU-B-1": {engmodel.ContractItemAssumption, []string{"REQ-B-001"}},
		},
	}
	for model, want := range cases {
		t.Run(model, func(t *testing.T) {
			items := engmodel.ContractItemsFromBundle(linksModel(t, model))
			check{t}.equal("number of items", len(items), len(want))
			for id, w := range want {
				item := items[id]
				check{t}.equal(id+" ID, kind and realizers", [3]any{item.ID, item.Kind, realizerIDs(item)}, [3]any{id, w.kind, w.realizers})
				expect(t, item.Text != "", "%s has no text", id)
			}
		})
	}
}

// The capability CAP-EMG-GO-API of this repository lists the requirements that realize it, in requirement order, on every call.
// Mutation: an unordered result or a missing realizer fails this test.
// TRLC-LINKS: REQ-EMG-203
func TestContractItemsFromBundleOfThisRepositoryIsStable(t *testing.T) {
	bundle := selfBundle(t)
	var want []string
	for _, r := range bundle.Requirements.Requirements {
		if slices.Contains(r.Realizes, "CAP-EMG-GO-API") {
			want = append(want, r.ID)
		}
	}
	expect(t, len(want) >= 13, "the model has %d realizers of CAP-EMG-GO-API", len(want))
	for call := 0; call < 5; call++ {
		item := engmodel.ContractItemsFromBundle(bundle)["CAP-EMG-GO-API"]
		check{t}.equal("kind and realizers", [2]any{item.Kind, realizerIDs(item)}, [2]any{engmodel.ContractItemCapability, want})
	}
}

// The realizers of an item are sorted by ID in a copy, and the input is not changed.
// Mutation: sorting in place, sorting descending or returning the input fails this test.
// TRLC-LINKS: REQ-EMG-203
func TestSortedContractItemRealizersReturnsASortedCopy(t *testing.T) {
	in := []engmodel.ContractItemRealizer{{ID: "REQ-3"}, {ID: "REQ-1"}, {ID: "REQ-2"}}
	got := engmodel.SortedContractItemRealizers(in)
	check{t}.equal("sorted copy", []string{got[0].ID, got[1].ID, got[2].ID}, []string{"REQ-1", "REQ-2", "REQ-3"})
	check{t}.equal("first of the input", in[0].ID, "REQ-3")
}

// linkCase is one link, the dependencies it is evaluated against and the status it must get.
type linkCase struct {
	child, pinned string
	deps          []engmodel.LinkDependency
	want          linkWant
}

// linkWant is the state and reason of a link, and whether it carries the item it points at.
type linkWant struct {
	state   engmodel.LinkState
	reason  engmodel.LinkReason
	hasItem bool
}

// resolved is a dependency that resolved to a published version of a model.
func resolved(alias, version string, b engmodel.Bundle) engmodel.LinkDependency {
	return engmodel.LinkDependency{Alias: alias, Version: version, Bundle: b, Publications: b.Manifest.Publications}
}

// linkCases are the ten states a requirement link can be in.
func linkCases(t *testing.T) map[string]linkCase {
	childA, childB := linksModel(t, "child-a"), linksModel(t, "child-b")
	withInterface := childA
	withInterface.Architecture.Contract.Provides = append([]engmodel.ContractEntry{{ID: "IF-A-1", Kind: "interface"}}, childA.Architecture.Contract.Provides...)
	hidden := resolved("encoder", "v0.1.0", childA)
	hidden.Publications = nil
	local := resolved("encoder", "v0.1.0", childA)
	local.Local = true
	encoder := []engmodel.LinkDependency{resolved("encoder", "v0.1.0", childA)}
	return map[string]linkCase{
		"unchanged capability":                     {"encoder::CAP-A-1", "v0.1.0", encoder, linkWant{engmodel.LinkStateUnchanged, engmodel.LinkReasonNone, true}},
		"unchanged assumption":                     {"bridge::AOU-B-1", "v0.1.0", []engmodel.LinkDependency{resolved("bridge", "v0.1.0", childB)}, linkWant{engmodel.LinkStateUnchanged, engmodel.LinkReasonNone, true}},
		"reviewed at another version":              {"encoder::CAP-A-1", "v0.0.9", encoder, linkWant{engmodel.LinkStateVersionMismatch, engmodel.LinkReasonNone, true}},
		"resolved without a version":               {"encoder::CAP-A-1", "v0.0.9", []engmodel.LinkDependency{resolved("encoder", "", childA)}, linkWant{engmodel.LinkStateVersionMismatch, engmodel.LinkReasonNone, true}},
		"resolved to another version":              {"encoder::CAP-A-1", "v0.1.0", []engmodel.LinkDependency{resolved("encoder", "v0.2.0", childA)}, linkWant{engmodel.LinkStateVersionMismatch, engmodel.LinkReasonNone, true}},
		"item missing":                             {"encoder::CAP-A-9", "v0.1.0", encoder, linkWant{engmodel.LinkStateBroken, engmodel.LinkReasonMissing, false}},
		"contract entry that is no item":           {"encoder::IF-A-1", "v0.1.0", []engmodel.LinkDependency{resolved("encoder", "v0.1.0", withInterface)}, linkWant{engmodel.LinkStateChildUnpublished, engmodel.LinkReasonNotContractItem, false}},
		"item outside every selected publication":  {"encoder::CAP-A-1", "v0.1.0", []engmodel.LinkDependency{hidden}, linkWant{engmodel.LinkStateChildUnpublished, engmodel.LinkReasonNotInPublication, false}},
		"alias that is no dependency":              {"ghost::CAP-A-1", "v0.1.0", encoder, linkWant{engmodel.LinkStateChildUnpublished, engmodel.LinkReasonUndeclaredAlias, false}},
		"dependency that was not resolved":         {"encoder::CAP-A-1", "v0.1.0", nil, linkWant{engmodel.LinkStateUnresolved, engmodel.LinkReasonUnresolved, false}},
		"unpublished item of a workspace checkout": {"encoder::CAP-A-9", "v0.1.0", []engmodel.LinkDependency{local}, linkWant{engmodel.LinkStateLocal, engmodel.LinkReasonMissing, false}},
	}
}

// explained reports whether a status of this state carries a detail for people.
func explained(state engmodel.LinkState) bool {
	return slices.Contains([]engmodel.LinkState{engmodel.LinkStateChildUnpublished, engmodel.LinkStateUnresolved, engmodel.LinkStateLocal}, state)
}

// EvaluateRequirementLinks gives each link of a system one status against its dependencies, with the reason where one applies.
// Mutation: a wrong state for any row, a lost reason or a reordered result fails this test.
// TRLC-LINKS: REQ-EMG-204
func TestEvaluateRequirementLinksGivesEachLinkOneState(t *testing.T) {
	for name, c := range linkCases(t) {
		t.Run(name, func(t *testing.T) {
			system := linksModel(t, "system")
			system.ReqLinks.ReqLinks = []engmodel.ReqLink{{ID: "LNK-1", Parent: "REQ-SYS-001", Child: c.child, Pinned: engmodel.ReqLinkPin{Version: c.pinned}, Status: engmodel.ReqLinkAccepted}}
			got := engmodel.EvaluateRequirementLinks(system, c.deps)
			if len(got) != 1 {
				t.Fatalf("%d statuses for one link", len(got))
			}
			s := got[0]
			check{t}.equal("the link carried over", [5]string{s.ID, s.Parent, s.Child, s.Status, s.Pinned}, [5]string{"LNK-1", "REQ-SYS-001", c.child, engmodel.ReqLinkAccepted, c.pinned})
			check{t}.equal("state, reason and item", linkWant{s.State, s.Reason, s.Item != nil}, c.want)
			check{t}.equal("detail only for unpublished, unresolved and local links", s.Detail != "", explained(s.State))
		})
	}
}

// The links of the example system are evaluated in link order against its two resolved dependencies and each reports the item at the pin.
// Mutation: skipping a link or evaluating against the wrong dependency fails this test.
// TRLC-LINKS: REQ-EMG-204
func TestEvaluateRequirementLinksOfTheExampleSystemKeepsLinkOrder(t *testing.T) {
	childA, childB := linksModel(t, "child-a"), linksModel(t, "child-b")
	got := engmodel.EvaluateRequirementLinks(linksModel(t, "system"), []engmodel.LinkDependency{resolved("encoder", "v0.1.0", childA), resolved("bridge", "v0.1.0", childB)})
	var ids []string
	for _, s := range got {
		expect(t, s.State == engmodel.LinkStateUnchanged && s.Item != nil && !s.NeedsReview(), "%s is %q and needs no review: %+v", s.ID, s.State, s)
		ids = append(ids, s.ID+":"+s.ItemID)
	}
	check{t}.equal("links in order", ids, []string{"LNK-SYS-001-ENCODER:CAP-A-1", "LNK-SYS-001-BRIDGE:CAP-B-1", "LNK-SYS-002-ENCODER:CAP-A-2", "LNK-AOU-B-1:AOU-B-1"})
}
