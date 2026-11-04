// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package reqlinks

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// goldenItem is the fixed input of the meaning golden test.
func goldenItem() Item {
	return Item{
		ID: "CAP-NMEA-GPS-OUTPUT", Kind: KindCapability, Text: "  Publish   NMEA\tGPS output. ",
		Realizers: []Realizer{
			{ID: "REQ-NMEA-007", Text: "When NMEA epoch period elapses, the NMEA runtime shall publish one NMEA epoch message.", Status: "approved",
				Notes: "Epoch  period is\nconfigurable.", VerificationMethods: []string{"test", " analysis "}, Criticality: "high"},
			{ID: "REQ-NMEA-002", Text: "The NMEA encoder shall format a GPGGA sentence <with> \"quotes\" & more.", VerificationCriteria: "A GGA\nline is emitted."},
		},
	}
}

// The meaning of a contract item is pinned: compact JSON
// {"item":{id,kind,text,verificationCriteria},"realizers":[{id,text,verificationCriteria}...]}
// with realizers sorted by id, whitespace runs collapsed and trimmed, case
// kept, no HTML escaping and no trailing newline. Notes, status,
// criticality and verification methods are not part of it (D23).
// Mutation: adding status or notes to the meaning, dropping a field or not
// sorting the realizers fails this test.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestItemMeaningGolden(t *testing.T) {
	want := `{"item":{"id":"CAP-NMEA-GPS-OUTPUT","kind":"capability","text":"Publish NMEA GPS output.","verificationCriteria":""},` +
		`"realizers":[{"id":"REQ-NMEA-002","text":"The NMEA encoder shall format a GPGGA sentence <with> \"quotes\" & more.","verificationCriteria":"A GGA line is emitted."},` +
		`{"id":"REQ-NMEA-007","text":"When NMEA epoch period elapses, the NMEA runtime shall publish one NMEA epoch message.","verificationCriteria":""}]}`
	if got := string(ItemMeaning(goldenItem())); got != want {
		t.Fatalf("meaning\n got %s\nwant %s", got, want)
	}
}

// Two versions of an item have the same meaning whatever the order of the
// realizers, whitespace, a realizer's status, notes, criticality or
// verification methods; a change to the item kind, text or criteria, or to
// a realizer's text or criteria, or an added or removed realizer, changes
// it, and MeaningChanges names each change by ID.
// Mutation: comparing status or notes, or not comparing a realizer's
// criteria, fails this test.
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-100
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestSameMeaning(t *testing.T) {
	base := goldenItem()
	same := map[string]func(*Item){
		"realizer order": func(i *Item) { i.Realizers[0], i.Realizers[1] = i.Realizers[1], i.Realizers[0] },
		"whitespace": func(i *Item) {
			i.Text = "Publish NMEA GPS output."
			i.Realizers[1].VerificationCriteria = "A GGA line   is emitted.  "
		},
		"realizer status":      func(i *Item) { i.Realizers[0].Status = "verified" },
		"realizer notes":       func(i *Item) { i.Realizers[0].Notes = "Epoch period is fixed." },
		"realizer criticality": func(i *Item) { i.Realizers[0].Criticality = "low" },
		"realizer methods":     func(i *Item) { i.Realizers[0].VerificationMethods = []string{"inspection"} },
	}
	for name, edit := range same {
		changed := goldenItem()
		edit(&changed)
		if !SameMeaning(base, changed) || len(MeaningChanges(base, changed)) != 0 {
			t.Errorf("%s changed the meaning: %v", name, MeaningChanges(base, changed))
		}
	}
	differ := map[string]struct {
		edit func(*Item)
		want string
	}{
		"item text":         {func(i *Item) { i.Text = "Publish NMEA GPS output!" }, "item text"},
		"item case":         {func(i *Item) { i.Text = "publish NMEA GPS output." }, "item text"},
		"item criteria":     {func(i *Item) { i.VerificationCriteria = "Shown." }, "item verificationCriteria"},
		"item kind":         {func(i *Item) { i.Kind = KindAssumption }, "item kind"},
		"realizer text":     {func(i *Item) { i.Realizers[0].Text += " Again." }, "REQ-NMEA-007 text"},
		"realizer criteria": {func(i *Item) { i.Realizers[0].VerificationCriteria = "Measured." }, "REQ-NMEA-007 verificationCriteria"},
		"realizer added":    {func(i *Item) { i.Realizers = append(i.Realizers, Realizer{ID: "REQ-NMEA-010", Text: "x"}) }, "REQ-NMEA-010 added"},
		"realizer removed":  {func(i *Item) { i.Realizers = i.Realizers[:1] }, "REQ-NMEA-002 removed"},
	}
	for name, tc := range differ {
		changed := goldenItem()
		tc.edit(&changed)
		if got := MeaningChanges(base, changed); SameMeaning(base, changed) || strings.Join(got, "; ") != tc.want {
			t.Errorf("%s: same %v, changes %v, want [%s]", name, SameMeaning(base, changed), got, tc.want)
		}
	}
	// Uncounted requirement fields do not reach the item at all.
	child := childBundle()
	before := ItemsFromBundle(child)["CAP-A"]
	r := &child.Requirements.Requirements[0]
	r.Title, r.Rationale, r.Priority, r.Tags, r.AppliesTo, r.Status = "A1", "Because.", "high", []string{"x"}, []string{"FU-X"}, "verified"
	if !SameMeaning(before, ItemsFromBundle(child)["CAP-A"]) {
		t.Fatal("title, rationale, priority, tags, appliesTo or status changed the meaning")
	}
	r.VerificationCriteria = "Now measured."
	if SameMeaning(before, ItemsFromBundle(child)["CAP-A"]) {
		t.Fatal("ItemsFromBundle does not carry the realizer criteria")
	}
}

// childBundle is a child model with two capabilities, an interface entry,
// two assumptions of use and requirements realizing them; publication public
// exposes CAP-A, AOU-A, IF-A and the realizers REQ-A1 and REQ-A2, not REQ-A3.
func childBundle() model.Bundle {
	var b model.Bundle
	b.Architecture.Contract = model.ContractModel{
		Provides: []model.ContractEntry{
			{ID: "CAP-A", Kind: "capability", Note: "Deliver A."},
			{ID: "CAP-B", Kind: "capability", Ref: "REQ-B", Note: "Deliver B."},
			{ID: "IF-A", Kind: "interface", Note: "Wire A."},
		},
		Assumptions: []model.ContractAssumption{
			{ID: "AOU-A", Text: "Bus is isolated.", VerificationCriteria: "Shown by user."},
			{ID: "AOU-HIDDEN", Text: "Not published."},
		},
	}
	b.Requirements.Requirements = []model.Requirement{
		{ID: "REQ-A1", Text: "The child shall do A1.", Realizes: []string{"CAP-A"}},
		{ID: "REQ-A2", Text: "The child shall do A2.", Realizes: []string{"CAP-A", "AOU-A"}},
		{ID: "REQ-B", Text: "The child shall do B."},
		{ID: "REQ-D", Text: "The child shall log.", Derived: true, DerivedRationale: "Ops."},
		{ID: "REQ-A3", Text: "The child shall do A3 privately.", Realizes: []string{"CAP-A"}},
	}
	b.Manifest.Publications = []model.ManifestPublication{{
		ID: "public", Architecture: []string{"CAP-A", "AOU-A", "IF-A"},
		Requirements: []string{"REQ-A1", "REQ-A2"},
	}}
	return b
}

// ItemsFromBundle returns every capability and assumption of use with its
// realizers (a deprecated contract ref counts as a realizer), never an
// interface entry; PublishedItems keeps only the items and realizers the
// publication exposes.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestItemsFromBundle(t *testing.T) {
	items := ItemsFromBundle(childBundle())
	if len(items) != 4 || items["IF-A"].ID != "" {
		t.Fatalf("items = %+v", items)
	}
	if got := realizerIDs(items["CAP-A"]); got != "REQ-A1,REQ-A2,REQ-A3" {
		t.Fatalf("CAP-A realizers = %s", got)
	}
	if got := realizerIDs(items["CAP-B"]); got != "REQ-B" || items["CAP-B"].Text != "Deliver B." {
		t.Fatalf("CAP-B (ref) = %+v", items["CAP-B"])
	}
	if a := items["AOU-A"]; a.Kind != KindAssumption || a.VerificationCriteria != "Shown by user." || realizerIDs(a) != "REQ-A2" {
		t.Fatalf("AOU-A = %+v", a)
	}
	b := childBundle()
	published := PublishedItems(b, b.Manifest.Publications[0])
	if len(published) != 2 || published["CAP-B"].ID != "" || realizerIDs(published["CAP-A"]) != "REQ-A1,REQ-A2" {
		t.Fatalf("published items = %+v", published)
	}
}

func realizerIDs(item Item) string {
	var ids []string
	for _, r := range item.Realizers {
		ids = append(ids, r.ID)
	}
	return strings.Join(ids, ",")
}

// systemWith returns a system bundle pinning dep at v0.1.0 with the links.
func systemWith(links ...model.ReqLink) model.Bundle {
	var b model.Bundle
	b.Manifest.Dependencies = []model.ManifestDependency{
		{Alias: "dep", Path: "example.com/dep@v0", Version: "v0.1.0", Publications: []string{"public"}},
		{Alias: "other", Path: "example.com/other@v0", Version: "v0.1.0", Publications: []string{"public"}},
	}
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-SYS-1", Text: "The system shall do A."}}
	b.ReqLinks.ReqLinks = links
	return b
}

// dependency is childBundle resolved as alias dep at v0.1.0.
func dependency(child model.Bundle) Dependency {
	return Dependency{Alias: "dep", Version: "v0.1.0", Bundle: child, Publications: child.Manifest.Publications}
}

// link is an accepted link of REQ-SYS-1 pinned to version.
func link(id, target, version string) model.ReqLink {
	return model.ReqLink{ID: id, Parent: "REQ-SYS-1", Child: target, Status: "accepted", Pinned: model.ReqLinkPin{Version: version}}
}

// EvaluateLinks checks each link's pin and item, never the item content
// (D23): unchanged when the pin equals the dependency pin and the item is
// published there, version_mismatch when the pin differs, broken when the
// item is gone at the dependency pin (also behind a stale pin),
// child_unpublished when it is not a published contract item, unresolved
// when the dependency was not resolved. The review status is carried.
// Mutation: comparing content, letting a stale pin hide a broken item or
// dropping the status fails this test.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestEvaluateLinksStates(t *testing.T) {
	child := childBundle()
	claimed := link("LNK-CLAIMED", "dep::CAP-A", "v0.1.0")
	claimed.Status = "claimed"
	system := systemWith(
		link("LNK-OK", "dep::CAP-A", "v0.1.0"),
		claimed,
		link("LNK-BROKEN", "dep::CAP-GONE", "v0.1.0"),
		link("LNK-STALE-GONE", "dep::CAP-GONE", "v0.0.9"),
		link("LNK-IF", "dep::IF-A", "v0.1.0"),
		link("LNK-UNPUB", "dep::CAP-B", "v0.1.0"),
		link("LNK-VERSION", "dep::CAP-A", "v0.0.9"),
		link("LNK-OTHER", "other::CAP-A", "v0.1.0"),
		link("LNK-UNDECLARED", "nope::CAP-A", "v0.1.0"),
		link("LNK-HIDDEN", "dep::AOU-HIDDEN", "v0.1.0"),
	)
	// A realizer text changed at the same pin: no content comparison.
	child.Requirements.Requirements[0].Text = "The child shall do A1 twice."
	got := map[string]LinkStatus{}
	for _, s := range EvaluateLinks(system, []Dependency{dependency(child)}) {
		got[s.ID] = s
	}
	want := map[string]State{
		"LNK-OK": StateUnchanged, "LNK-CLAIMED": StateUnchanged, "LNK-BROKEN": StateBroken, "LNK-STALE-GONE": StateBroken,
		"LNK-IF": StateChildUnpublished, "LNK-UNPUB": StateChildUnpublished,
		"LNK-VERSION": StateVersionMismatch, "LNK-OTHER": StateUnresolved, "LNK-UNDECLARED": StateChildUnpublished,
		"LNK-HIDDEN": StateChildUnpublished,
	}
	for id, state := range want {
		if got[id].State != state {
			t.Errorf("%s state = %q, want %q (%+v)", id, got[id].State, state, got[id])
		}
	}
	if got["LNK-CLAIMED"].Status != "claimed" || got["LNK-OK"].Status != "accepted" {
		t.Fatalf("review status not carried: %+v %+v", got["LNK-CLAIMED"], got["LNK-OK"])
	}
	resolvedElsewhere := dependency(child)
	resolvedElsewhere.Version = "v0.1.1"
	if s := EvaluateLinks(system, []Dependency{resolvedElsewhere}); s[0].State != StateVersionMismatch {
		t.Fatalf("a pin other than the resolved version must be version_mismatch: %+v", s[0])
	}
	if d := got["LNK-HIDDEN"].Detail; !strings.Contains(d, "not in a selected publication") {
		t.Fatalf("LNK-HIDDEN detail = %q", d)
	}
	if s := got["LNK-VERSION"]; s.Item == nil || s.Item.ID != "CAP-A" || s.Pinned != "v0.0.9" || s.Version != "v0.1.0" {
		t.Fatalf("version_mismatch status = %+v", s)
	}
}

// A dependency resolved from a workspace replacement is not checked
// against its local content (D23): an item missing or unpublished there is
// local, never broken or child_unpublished; a present item still carries
// the pin check and the item.
// Mutation: evaluating a local dependency like a published one fails this
// test.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestEvaluateLinksOfLocalDependency(t *testing.T) {
	child := childBundle()
	dep := dependency(child)
	dep.Local = true
	system := systemWith(link("LNK-OK", "dep::CAP-A", "v0.1.0"), link("LNK-GONE", "dep::CAP-GONE", "v0.1.0"),
		link("LNK-UNPUB", "dep::CAP-B", "v0.1.0"), link("LNK-STALE", "dep::CAP-A", "v0.0.9"))
	got := map[string]LinkStatus{}
	for _, s := range EvaluateLinks(system, []Dependency{dep}) {
		got[s.ID] = s
	}
	want := map[string]State{"LNK-OK": StateUnchanged, "LNK-GONE": StateLocal, "LNK-UNPUB": StateLocal, "LNK-STALE": StateVersionMismatch}
	for id, state := range want {
		if got[id].State != state {
			t.Errorf("%s state = %q, want %q", id, got[id].State, state)
		}
	}
	if !strings.Contains(got["LNK-GONE"].Detail, "workspace") || got["LNK-OK"].Item == nil {
		t.Fatalf("local statuses: %+v", got)
	}
}
