// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package reqlinks

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// goldenItem is the fixed input of the fingerprint golden test.
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

// The fingerprint algorithm is pinned: "sha256:v1:" and the hex sha256 over
// the canonical JSON
// {"item":{id,kind,text,verificationCriteria},"realizers":[{id,text,notes,verificationCriteria,verificationMethods,criticality,status}...]}
// with realizers sorted by id, verificationMethods sorted, whitespace runs
// collapsed and trimmed, case kept, no HTML escaping and no trailing
// newline. The expected hash was checked independently:
// printf '%s' '<canonical>' | sha256sum.
// Mutation: dropping a canonical field, not sorting verificationMethods or
// dropping the version from the prefix fails this test.
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-109
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestFingerprintGolden(t *testing.T) {
	item := goldenItem()
	wantCanonical := `{"item":{"id":"CAP-NMEA-GPS-OUTPUT","kind":"capability","text":"Publish NMEA GPS output.","verificationCriteria":""},` +
		`"realizers":[{"id":"REQ-NMEA-002","text":"The NMEA encoder shall format a GPGGA sentence <with> \"quotes\" & more.","notes":"","verificationCriteria":"A GGA line is emitted.","verificationMethods":[],"criticality":"","status":""},` +
		`{"id":"REQ-NMEA-007","text":"When NMEA epoch period elapses, the NMEA runtime shall publish one NMEA epoch message.","notes":"Epoch period is configurable.","verificationCriteria":"","verificationMethods":["analysis","test"],"criticality":"high","status":"approved"}]}`
	if got := string(CanonicalItem(item)); got != wantCanonical {
		t.Fatalf("canonical item\n got %s\nwant %s", got, wantCanonical)
	}
	const want = "sha256:v1:cde6842da74be3e1bf8bc92a9218537f78a31a36f29130cacfd7a4d964f199cd"
	if got := Fingerprint(item); got != want {
		t.Fatalf("Fingerprint = %s, want %s", got, want)
	}
}

// The same content gives the same fingerprint whatever the order of the
// realizers or their verification methods; a change to a realizer's text,
// notes, criteria, methods, criticality or status, or to the item, changes
// it; whitespace-only and case changes behave as specified. Fields the
// fingerprint does not cover (title, rationale, appliesTo, priority, tags)
// do not change it.
// Mutation: dropping notes, verificationMethods or criticality from the
// canonical realizer fails this test.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestFingerprintIsDeterministic(t *testing.T) {
	base := Fingerprint(goldenItem())
	shuffled := goldenItem()
	shuffled.Realizers[0], shuffled.Realizers[1] = shuffled.Realizers[1], shuffled.Realizers[0]
	shuffled.Realizers[1].VerificationMethods = []string{"analysis", "test"}
	if Fingerprint(shuffled) != base {
		t.Fatal("realizer or method order changed the fingerprint")
	}
	spaced := goldenItem()
	spaced.Text = "Publish NMEA GPS output."
	spaced.Realizers[1].VerificationCriteria = "A GGA line   is emitted.  "
	spaced.Realizers[0].Notes = " Epoch period is configurable. "
	if Fingerprint(spaced) != base {
		t.Fatal("whitespace-only changes changed the fingerprint")
	}
	for name, edit := range map[string]func(*Item){
		"item text":            func(i *Item) { i.Text = "Publish NMEA GPS output!" },
		"item case":            func(i *Item) { i.Text = "publish NMEA GPS output." },
		"item criteria":        func(i *Item) { i.VerificationCriteria = "Shown." },
		"item kind":            func(i *Item) { i.Kind = KindAssumption },
		"realizer text":        func(i *Item) { i.Realizers[0].Text += " Again." },
		"realizer notes":       func(i *Item) { i.Realizers[0].Notes = "Epoch period is fixed." },
		"realizer methods":     func(i *Item) { i.Realizers[0].VerificationMethods = []string{"test"} },
		"realizer criticality": func(i *Item) { i.Realizers[0].Criticality = "low" },
		"realizer state":       func(i *Item) { i.Realizers[1].Status = "retired" },
		"realizer added":       func(i *Item) { i.Realizers = append(i.Realizers, Realizer{ID: "REQ-NMEA-010", Text: "x"}) },
	} {
		changed := goldenItem()
		edit(&changed)
		if Fingerprint(changed) == base {
			t.Errorf("%s did not change the fingerprint", name)
		}
	}
	// Uncounted requirement fields do not reach the item at all.
	child := childBundle()
	before := Fingerprint(ItemsFromBundle(child)["CAP-A"])
	r := &child.Requirements.Requirements[0]
	r.Title, r.Rationale, r.Priority, r.Tags, r.AppliesTo = "A1", "Because.", "high", []string{"x"}, []string{"FU-X"}
	if Fingerprint(ItemsFromBundle(child)["CAP-A"]) != before {
		t.Fatal("title, rationale, priority, tags or appliesTo changed the fingerprint")
	}
	r.Notes = "Now with a note."
	if Fingerprint(ItemsFromBundle(child)["CAP-A"]) == before {
		t.Fatal("ItemsFromBundle does not carry the realizer notes")
	}
}

// FingerprintVersion reads the canonical-form version of a recorded
// fingerprint: 1 for "sha256:v1:<hex>", 0 for the unversioned
// "sha256:<hex>" of v0.8.0 pre-releases, the number of any other
// "sha256:vN:" prefix, and -1 for anything else.
// TRLC-LINKS: REQ-EMG-109
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestFingerprintVersion(t *testing.T) {
	hex := strings.Repeat("a", 64)
	for in, want := range map[string]int{
		"sha256:v1:" + hex: 1, "sha256:" + hex: 0, "sha256:v2:" + hex: 2, "sha256:v12:" + hex: 12,
		"": -1, "sha256:v1:abc": -1, "md5:" + hex: -1, "sha256:vx:" + hex: -1,
	} {
		if got := FingerprintVersion(in); got != want {
			t.Errorf("FingerprintVersion(%q) = %d, want %d", in, got, want)
		}
	}
	if FingerprintVersion(Fingerprint(goldenItem())) != CanonicalVersion {
		t.Fatal("Fingerprint does not write the current canonical version")
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

// EvaluateLinks compares each link's pin with the dependency it resolves to:
// unchanged when the fingerprint matches, suspect when it differs, broken
// when the item is gone, child_unpublished when it is not a published
// contract item, version_mismatch when the pin differs from the
// dependency, unresolved when the dependency was not resolved.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-095, REQ-EMG-109
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestEvaluateLinksStates(t *testing.T) {
	child := childBundle()
	current := Fingerprint(PublishedItems(child, child.Manifest.Publications[0])["CAP-A"])
	link := func(id, target, version, fingerprint string) model.ReqLink {
		return model.ReqLink{ID: id, Parent: "REQ-SYS-1", Child: target, Status: "accepted", Pinned: model.ReqLinkPin{Version: version, Fingerprint: fingerprint}}
	}
	system := systemWith(
		link("LNK-OK", "dep::CAP-A", "v0.1.0", current),
		link("LNK-SUSPECT", "dep::CAP-A", "v0.1.0", "sha256:v1:"+strings.Repeat("0", 64)),
		link("LNK-BROKEN", "dep::CAP-GONE", "v0.1.0", current),
		link("LNK-IF", "dep::IF-A", "v0.1.0", current),
		link("LNK-UNPUB", "dep::CAP-B", "v0.1.0", current),
		link("LNK-VERSION", "dep::CAP-A", "v0.0.9", current),
		link("LNK-OTHER", "other::CAP-A", "v0.1.0", current),
		link("LNK-UNDECLARED", "nope::CAP-A", "v0.1.0", current),
		link("LNK-HIDDEN", "dep::AOU-HIDDEN", "v0.1.0", current),
		link("LNK-OLD-FORM", "dep::CAP-A", "v0.1.0", "sha256:"+strings.Repeat("0", 64)),
		link("LNK-NEW-FORM", "dep::CAP-A", "v0.1.0", "sha256:v9:"+strings.Repeat("0", 64)),
	)
	got := map[string]LinkStatus{}
	for _, s := range EvaluateLinks(system, []Dependency{dependency(child)}) {
		got[s.ID] = s
	}
	want := map[string]State{
		"LNK-OK": StateUnchanged, "LNK-SUSPECT": StateSuspect, "LNK-BROKEN": StateBroken,
		"LNK-IF": StateChildUnpublished, "LNK-UNPUB": StateChildUnpublished,
		"LNK-VERSION": StateVersionMismatch, "LNK-OTHER": StateUnresolved, "LNK-UNDECLARED": StateChildUnpublished,
		"LNK-HIDDEN": StateChildUnpublished, "LNK-OLD-FORM": StateFingerprintOutdated, "LNK-NEW-FORM": StateFingerprintOutdated,
	}
	for id, state := range want {
		if got[id].State != state {
			t.Errorf("%s state = %q, want %q (%+v)", id, got[id].State, state, got[id])
		}
	}
	resolvedElsewhere := dependency(child)
	resolvedElsewhere.Version = "v0.1.1"
	if s := EvaluateLinks(system, []Dependency{resolvedElsewhere}); s[0].State != StateVersionMismatch {
		t.Fatalf("a pin other than the resolved version must be version_mismatch: %+v", s[0])
	}
	if d := got["LNK-HIDDEN"].Detail; !strings.Contains(d, "not in a selected publication") {
		t.Fatalf("LNK-HIDDEN detail = %q", d)
	}
	if s := got["LNK-SUSPECT"]; s.Current != current || s.Recorded == current || s.Item == nil || s.Item.ID != "CAP-A" {
		t.Fatalf("suspect status = %+v", s)
	}
}

// A change to a realizing requirement of a linked item makes the link
// suspect; a change to a requirement that realizes no linked item leaves it
// unchanged.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestEvaluateLinksSeesOnlyRelevantChanges(t *testing.T) {
	child := childBundle()
	pin := Fingerprint(PublishedItems(child, child.Manifest.Publications[0])["CAP-A"])
	system := systemWith(model.ReqLink{ID: "LNK-A", Parent: "REQ-SYS-1", Child: "dep::CAP-A", Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: pin}})

	unrelated := childBundle()
	unrelated.Requirements.Requirements[3].Text = "The child shall log more."
	if s := EvaluateLinks(system, []Dependency{dependency(unrelated)}); s[0].State != StateUnchanged {
		t.Fatalf("unrelated change: %+v", s)
	}
	related := childBundle()
	related.Requirements.Requirements[1].Text = "The child shall do A2 faster."
	if s := EvaluateLinks(system, []Dependency{dependency(related)}); s[0].State != StateSuspect {
		t.Fatalf("realizer change: %+v", s)
	}
}

// A link whose pin is stale (version_mismatch) is still evaluated against
// the item at the resolved version: Content is unchanged, suspect or broken
// and Current is that version's fingerprint, so a bump that kept the old pin
// shows what changed.
// Mutation: skipping the content evaluation of a version_mismatch link
// fails this test.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-100
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestEvaluateStaleLinkContent(t *testing.T) {
	child := childBundle()
	current := Fingerprint(PublishedItems(child, child.Manifest.Publications[0])["CAP-A"])
	zero := "sha256:v1:" + strings.Repeat("0", 64)
	stale := func(id, target, fingerprint string) model.ReqLink {
		return model.ReqLink{ID: id, Parent: "REQ-SYS-1", Child: target, Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.0.9", Fingerprint: fingerprint}}
	}
	system := systemWith(stale("LNK-SAME", "dep::CAP-A", current), stale("LNK-CHANGED", "dep::CAP-A", zero), stale("LNK-GONE", "dep::CAP-GONE", current))
	got := map[string]LinkStatus{}
	for _, s := range EvaluateLinks(system, []Dependency{dependency(child)}) {
		got[s.ID] = s
	}
	want := map[string]State{"LNK-SAME": StateUnchanged, "LNK-CHANGED": StateSuspect, "LNK-GONE": StateBroken}
	for id, content := range want {
		if s := got[id]; s.State != StateVersionMismatch || s.Content != content {
			t.Errorf("%s = %s/%s, want version_mismatch/%s", id, s.State, s.Content, content)
		}
	}
	if s := got["LNK-CHANGED"]; s.Current != current || s.Item == nil {
		t.Fatalf("stale suspect link lacks the current fingerprint and item: %+v", s)
	}
}
