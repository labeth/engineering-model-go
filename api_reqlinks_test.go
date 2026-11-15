// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel_test

import (
	"path/filepath"
	"reflect"
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

// ContractItemsFromBundle lists the capabilities and the assumptions of use of a model with the requirements that realize
// each, in requirement order and the same on every call.
// Mutation: dropping an item kind, a realizer or the requirement order fails this test.
// TRLC-LINKS: REQ-EMG-203
func TestContractItemsFromBundleListsItemsWithTheirRealizers(t *testing.T) {
	cases := []struct {
		model, item string
		kind        engmodel.ContractItemKind
		realizers   []string
	}{
		{"child-a", "CAP-A-1", engmodel.ContractItemCapability, []string{"REQ-A-001", "REQ-A-002"}},
		{"child-a", "CAP-A-2", engmodel.ContractItemCapability, []string{"REQ-A-003"}},
		{"child-b", "CAP-B-1", engmodel.ContractItemCapability, []string{"REQ-B-001"}},
		{"child-b", "AOU-B-1", engmodel.ContractItemAssumption, []string{"REQ-B-001"}},
	}
	for _, c := range cases {
		t.Run(c.item, func(t *testing.T) {
			item, ok := engmodel.ContractItemsFromBundle(linksModel(t, c.model))[c.item]
			if !ok {
				t.Fatalf("%s is not a contract item of %s", c.item, c.model)
			}
			if item.ID != c.item || item.Kind != c.kind || item.Text == "" || !reflect.DeepEqual(realizerIDs(item), c.realizers) {
				t.Errorf("item %+v: kind %q, realizers %v, want kind %q and %v", item.ID, item.Kind, realizerIDs(item), c.kind, c.realizers)
			}
		})
	}
	if items := engmodel.ContractItemsFromBundle(linksModel(t, "child-a")); len(items) != 2 {
		t.Errorf("child-a has two contract items, got %d", len(items))
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
	if len(want) < 13 {
		t.Fatalf("the model has %d realizers of CAP-EMG-GO-API", len(want))
	}
	for call := 0; call < 5; call++ {
		item, ok := engmodel.ContractItemsFromBundle(bundle)["CAP-EMG-GO-API"]
		if !ok || item.Kind != engmodel.ContractItemCapability {
			t.Fatalf("call %d: CAP-EMG-GO-API is %+v", call, item)
		}
		if got := realizerIDs(item); !reflect.DeepEqual(got, want) {
			t.Fatalf("call %d: realizers %v, want %v", call, got, want)
		}
	}
}

// EvaluateRequirementLinks gives each link of a system one status against its dependencies, with the reason where one applies.
// Mutation: a wrong state for any row, a lost reason or a reordered result fails this test.
// TRLC-LINKS: REQ-EMG-204
func TestEvaluateRequirementLinksGivesEachLinkOneState(t *testing.T) {
	childA, childB := linksModel(t, "child-a"), linksModel(t, "child-b")
	withInterface := childA
	withInterface.Architecture.Contract.Provides = append([]engmodel.ContractEntry{{ID: "IF-A-1", Kind: "interface"}}, childA.Architecture.Contract.Provides...)
	dep := func(alias, version string, b engmodel.Bundle, local bool) engmodel.LinkDependency {
		return engmodel.LinkDependency{Alias: alias, Version: version, Bundle: b, Publications: b.Manifest.Publications, Local: local}
	}
	hidden := dep("encoder", "v0.1.0", childA, false)
	hidden.Publications = nil
	cases := []struct {
		name    string
		child   string
		pinned  string
		deps    []engmodel.LinkDependency
		state   engmodel.LinkState
		reason  engmodel.LinkReason
		hasItem bool
	}{
		{"unchanged capability", "encoder::CAP-A-1", "v0.1.0", []engmodel.LinkDependency{dep("encoder", "v0.1.0", childA, false)}, engmodel.LinkStateUnchanged, engmodel.LinkReasonNone, true},
		{"unchanged assumption", "bridge::AOU-B-1", "v0.1.0", []engmodel.LinkDependency{dep("bridge", "v0.1.0", childB, false)}, engmodel.LinkStateUnchanged, engmodel.LinkReasonNone, true},
		{"reviewed at another version", "encoder::CAP-A-1", "v0.0.9", []engmodel.LinkDependency{dep("encoder", "v0.1.0", childA, false)}, engmodel.LinkStateVersionMismatch, engmodel.LinkReasonNone, true},
		{"resolved without a version, pin differs from the manifest", "encoder::CAP-A-1", "v0.0.9", []engmodel.LinkDependency{dep("encoder", "", childA, false)}, engmodel.LinkStateVersionMismatch, engmodel.LinkReasonNone, true},
		{"resolved to another version", "encoder::CAP-A-1", "v0.1.0", []engmodel.LinkDependency{dep("encoder", "v0.2.0", childA, false)}, engmodel.LinkStateVersionMismatch, engmodel.LinkReasonNone, true},
		{"item missing", "encoder::CAP-A-9", "v0.1.0", []engmodel.LinkDependency{dep("encoder", "v0.1.0", childA, false)}, engmodel.LinkStateBroken, engmodel.LinkReasonMissing, false},
		{"contract entry that is no item", "encoder::IF-A-1", "v0.1.0", []engmodel.LinkDependency{dep("encoder", "v0.1.0", withInterface, false)}, engmodel.LinkStateChildUnpublished, engmodel.LinkReasonNotContractItem, false},
		{"item outside every selected publication", "encoder::CAP-A-1", "v0.1.0", []engmodel.LinkDependency{hidden}, engmodel.LinkStateChildUnpublished, engmodel.LinkReasonNotInPublication, false},
		{"alias that is no dependency", "ghost::CAP-A-1", "v0.1.0", []engmodel.LinkDependency{dep("encoder", "v0.1.0", childA, false)}, engmodel.LinkStateChildUnpublished, engmodel.LinkReasonUndeclaredAlias, false},
		{"dependency that was not resolved", "encoder::CAP-A-1", "v0.1.0", nil, engmodel.LinkStateUnresolved, engmodel.LinkReasonUnresolved, false},
		{"unpublished item of a workspace checkout", "encoder::CAP-A-9", "v0.1.0", []engmodel.LinkDependency{dep("encoder", "v0.1.0", childA, true)}, engmodel.LinkStateLocal, engmodel.LinkReasonMissing, false},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			system := linksModel(t, "system")
			system.ReqLinks.ReqLinks = []engmodel.ReqLink{{ID: "LNK-1", Parent: "REQ-SYS-001", Child: c.child, Pinned: engmodel.ReqLinkPin{Version: c.pinned}, Status: engmodel.ReqLinkAccepted}}
			got := engmodel.EvaluateRequirementLinks(system, c.deps)
			if len(got) != 1 {
				t.Fatalf("%d statuses for one link", len(got))
			}
			s := got[0]
			if s.ID != "LNK-1" || s.Parent != "REQ-SYS-001" || s.Child != c.child || s.Status != engmodel.ReqLinkAccepted || s.Pinned != c.pinned {
				t.Errorf("the link is not carried over: %+v", s)
			}
			if s.State != c.state || s.Reason != c.reason || (s.Item != nil) != c.hasItem {
				t.Errorf("state %q reason %q item %v; want %q, %q, item %v", s.State, s.Reason, s.Item, c.state, c.reason, c.hasItem)
			}
			explained := s.State == engmodel.LinkStateChildUnpublished || s.State == engmodel.LinkStateUnresolved || s.State == engmodel.LinkStateLocal
			if explained != (s.Detail != "") {
				t.Errorf("detail %q for state %q: only unpublished, unresolved and local links carry one", s.Detail, s.State)
			}
		})
	}
}

// The links of the example system are evaluated in link order against its two resolved dependencies and each reports the item at the pin.
// Mutation: skipping a link or evaluating against the wrong dependency fails this test.
// TRLC-LINKS: REQ-EMG-204
func TestEvaluateRequirementLinksOfTheExampleSystemKeepsLinkOrder(t *testing.T) {
	system, childA, childB := linksModel(t, "system"), linksModel(t, "child-a"), linksModel(t, "child-b")
	deps := []engmodel.LinkDependency{
		{Alias: "encoder", Version: "v0.1.0", Bundle: childA, Publications: childA.Manifest.Publications},
		{Alias: "bridge", Version: "v0.1.0", Bundle: childB, Publications: childB.Manifest.Publications},
	}
	got := engmodel.EvaluateRequirementLinks(system, deps)
	want := []string{"LNK-SYS-001-ENCODER:CAP-A-1", "LNK-SYS-001-BRIDGE:CAP-B-1", "LNK-SYS-002-ENCODER:CAP-A-2", "LNK-AOU-B-1:AOU-B-1"}
	var ids []string
	for _, s := range got {
		if s.State != engmodel.LinkStateUnchanged || s.Item == nil || s.NeedsReview() {
			t.Errorf("%s is %q and needs no review: %+v", s.ID, s.State, s)
		}
		ids = append(ids, s.ID+":"+s.ItemID)
	}
	if !reflect.DeepEqual(ids, want) {
		t.Errorf("links %v, want %v", ids, want)
	}
}

// The realizers of an item are sorted by ID in a copy, and the input is not changed.
// Mutation: sorting in place, sorting descending or returning the input fails this test.
// TRLC-LINKS: REQ-EMG-203
func TestSortedContractItemRealizersReturnsASortedCopy(t *testing.T) {
	in := []engmodel.ContractItemRealizer{{ID: "REQ-3"}, {ID: "REQ-1"}, {ID: "REQ-2"}}
	got := engmodel.SortedContractItemRealizers(in)
	if !reflect.DeepEqual([]string{got[0].ID, got[1].ID, got[2].ID}, []string{"REQ-1", "REQ-2", "REQ-3"}) {
		t.Errorf("sorted %+v", got)
	}
	if in[0].ID != "REQ-3" {
		t.Errorf("the input was sorted in place: %+v", in)
	}
}
