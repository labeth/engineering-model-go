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
			{ID: "REQ-NMEA-007", Text: "When NMEA epoch period elapses, the NMEA runtime shall publish one NMEA epoch message.", Status: "approved"},
			{ID: "REQ-NMEA-002", Text: "The NMEA encoder shall format a GPGGA sentence <with> \"quotes\" & more.", VerificationCriteria: "A GGA\nline is emitted."},
		},
	}
}

// The fingerprint algorithm is pinned: sha256 over the canonical JSON
// {"item":{id,kind,text,verificationCriteria},"realizers":[{id,text,verificationCriteria,status}...]}
// with realizers sorted by id, whitespace runs collapsed and trimmed, case
// kept, no HTML escaping and no trailing newline. The expected hash was
// checked independently: printf '%s' '<canonical>' | sha256sum.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestFingerprintGolden(t *testing.T) {
	item := goldenItem()
	wantCanonical := `{"item":{"id":"CAP-NMEA-GPS-OUTPUT","kind":"capability","text":"Publish NMEA GPS output.","verificationCriteria":""},` +
		`"realizers":[{"id":"REQ-NMEA-002","text":"The NMEA encoder shall format a GPGGA sentence <with> \"quotes\" & more.","verificationCriteria":"A GGA line is emitted.","status":""},` +
		`{"id":"REQ-NMEA-007","text":"When NMEA epoch period elapses, the NMEA runtime shall publish one NMEA epoch message.","verificationCriteria":"","status":"approved"}]}`
	if got := string(CanonicalItem(item)); got != wantCanonical {
		t.Fatalf("canonical item\n got %s\nwant %s", got, wantCanonical)
	}
	const want = "sha256:66687f1f595d7eab6b81db14bd758a3abdcf8afdb78dfde0ccdbb711692cb658"
	if got := Fingerprint(item); got != want {
		t.Fatalf("Fingerprint = %s, want %s", got, want)
	}
}

// The same content gives the same fingerprint whatever the order of the
// realizers; a change to a realizer's text, criteria or status, or to the
// item, changes it; whitespace-only and case changes behave as specified.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestFingerprintIsDeterministic(t *testing.T) {
	base := Fingerprint(goldenItem())
	shuffled := goldenItem()
	shuffled.Realizers[0], shuffled.Realizers[1] = shuffled.Realizers[1], shuffled.Realizers[0]
	if Fingerprint(shuffled) != base {
		t.Fatal("realizer order changed the fingerprint")
	}
	spaced := goldenItem()
	spaced.Text = "Publish NMEA GPS output."
	spaced.Realizers[1].VerificationCriteria = "A GGA line   is emitted.  "
	if Fingerprint(spaced) != base {
		t.Fatal("whitespace-only changes changed the fingerprint")
	}
	for name, edit := range map[string]func(*Item){
		"item text":      func(i *Item) { i.Text = "Publish NMEA GPS output!" },
		"item case":      func(i *Item) { i.Text = "publish NMEA GPS output." },
		"item criteria":  func(i *Item) { i.VerificationCriteria = "Shown." },
		"realizer text":  func(i *Item) { i.Realizers[0].Text += " Again." },
		"realizer state": func(i *Item) { i.Realizers[1].Status = "retired" },
		"realizer added": func(i *Item) { i.Realizers = append(i.Realizers, Realizer{ID: "REQ-NMEA-010", Text: "x"}) },
	} {
		changed := goldenItem()
		edit(&changed)
		if Fingerprint(changed) == base {
			t.Errorf("%s did not change the fingerprint", name)
		}
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
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestEvaluateLinksStates(t *testing.T) {
	child := childBundle()
	current := Fingerprint(PublishedItems(child, child.Manifest.Publications[0])["CAP-A"])
	link := func(id, target, version, fingerprint string) model.ReqLink {
		return model.ReqLink{ID: id, Parent: "REQ-SYS-1", Child: target, Status: "accepted", Pinned: model.ReqLinkPin{Version: version, Fingerprint: fingerprint}}
	}
	system := systemWith(
		link("LNK-OK", "dep::CAP-A", "v0.1.0", current),
		link("LNK-SUSPECT", "dep::CAP-A", "v0.1.0", "sha256:"+strings.Repeat("0", 64)),
		link("LNK-BROKEN", "dep::CAP-GONE", "v0.1.0", current),
		link("LNK-IF", "dep::IF-A", "v0.1.0", current),
		link("LNK-UNPUB", "dep::CAP-B", "v0.1.0", current),
		link("LNK-VERSION", "dep::CAP-A", "v0.0.9", current),
		link("LNK-OTHER", "other::CAP-A", "v0.1.0", current),
		link("LNK-UNDECLARED", "nope::CAP-A", "v0.1.0", current),
		link("LNK-HIDDEN", "dep::AOU-HIDDEN", "v0.1.0", current),
	)
	got := map[string]LinkStatus{}
	for _, s := range EvaluateLinks(system, []Dependency{dependency(child)}) {
		got[s.ID] = s
	}
	want := map[string]State{
		"LNK-OK": StateUnchanged, "LNK-SUSPECT": StateSuspect, "LNK-BROKEN": StateBroken,
		"LNK-IF": StateChildUnpublished, "LNK-UNPUB": StateChildUnpublished,
		"LNK-VERSION": StateVersionMismatch, "LNK-OTHER": StateUnresolved, "LNK-UNDECLARED": StateChildUnpublished,
		"LNK-HIDDEN": StateChildUnpublished,
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

// Diagnostics reports link.suspect (with the current item and realizer texts
// and both fingerprints), link.broken and link.child_unpublished as errors,
// and an assumption of use of a dependency without an evidence link as
// link.aou_unverified: an error in strict mode, a warning in guided mode
// where links apply, nothing in a guided model without links or contract.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-095, REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func TestDiagnostics(t *testing.T) {
	child := childBundle()
	deps := []Dependency{dependency(child)}
	zero := "sha256:" + strings.Repeat("0", 64)
	system := systemWith(
		model.ReqLink{ID: "LNK-SUSPECT", Parent: "REQ-SYS-1", Child: "dep::CAP-A", Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: zero}},
		model.ReqLink{ID: "LNK-BROKEN", Parent: "REQ-SYS-1", Child: "dep::CAP-GONE", Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: zero}},
		model.ReqLink{ID: "LNK-IF", Parent: "REQ-SYS-1", Child: "dep::IF-A", Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: zero}},
	)
	system.Manifest.RequirementLinks = model.RequirementLinksStrict
	codes := map[string]string{}
	for _, d := range Diagnostics(system, deps) {
		codes[d.Code] += string(d.Severity) + "|" + d.Message + "\n"
	}
	suspect := codes["link.suspect"]
	for _, want := range []string{"error|", "LNK-SUSPECT", "dep::CAP-A", zero, "Deliver A.", "REQ-A1: The child shall do A1.", "REQ-A2: The child shall do A2."} {
		if !strings.Contains(suspect, want) {
			t.Fatalf("link.suspect lacks %q: %q", want, suspect)
		}
	}
	if !strings.Contains(codes["link.broken"], "error|") || !strings.Contains(codes["link.broken"], "LNK-BROKEN") {
		t.Fatalf("link.broken = %q", codes["link.broken"])
	}
	if !strings.Contains(codes["link.child_unpublished"], "LNK-IF") {
		t.Fatalf("link.child_unpublished = %q", codes["link.child_unpublished"])
	}
	if !strings.Contains(codes["link.aou_unverified"], "error|") || !strings.Contains(codes["link.aou_unverified"], "dep::AOU-A") {
		t.Fatalf("strict link.aou_unverified = %q", codes["link.aou_unverified"])
	}

	evidenced := systemWith(model.ReqLink{ID: "LNK-AOU", Child: "dep::AOU-A", Evidence: []string{"REQ-SYS-1"}, Status: "accepted",
		Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: Fingerprint(PublishedItems(child, child.Manifest.Publications[0])["AOU-A"])}})
	if d := Diagnostics(evidenced, deps); len(d) != 0 {
		t.Fatalf("evidenced assumption: %+v", d)
	}
	noEvidence := evidenced
	noEvidence.ReqLinks.ReqLinks = []model.ReqLink{evidenced.ReqLinks.ReqLinks[0]}
	noEvidence.ReqLinks.ReqLinks[0].Evidence = nil
	noEvidence.ReqLinks.ReqLinks[0].Parent = "REQ-SYS-1"
	if d := Diagnostics(noEvidence, deps); len(d) != 1 || d[0].Code != "link.aou_unverified" || d[0].Severity != "warning" {
		t.Fatalf("guided link without evidence: %+v", d)
	}
	if d := Diagnostics(systemWith(), deps); len(d) != 0 {
		t.Fatalf("guided model without links or contract: %+v", d)
	}
}

// A link whose pin is stale (version_mismatch) is still evaluated against
// the item at the resolved version: Content is unchanged, suspect or broken
// and Current is that version's fingerprint, and the composed diagnostics
// report link.suspect (with the current text) or link.broken for it, so a
// bump that kept the old pin shows what changed.
// Mutation: skipping the content evaluation of a version_mismatch link, or
// not reporting its suspect or broken content, fails this test.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-100
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestEvaluateStaleLinkContent(t *testing.T) {
	child := childBundle()
	current := Fingerprint(PublishedItems(child, child.Manifest.Publications[0])["CAP-A"])
	zero := "sha256:" + strings.Repeat("0", 64)
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
	codes := map[string]string{}
	for _, d := range Diagnostics(system, []Dependency{dependency(child)}) {
		codes[d.Code] += d.Message + "\n"
	}
	if !strings.Contains(codes["link.suspect"], "LNK-CHANGED") || !strings.Contains(codes["link.suspect"], "REQ-A1: The child shall do A1.") || strings.Contains(codes["link.suspect"], "LNK-SAME") {
		t.Fatalf("link.suspect = %q", codes["link.suspect"])
	}
	if !strings.Contains(codes["link.broken"], "LNK-GONE") {
		t.Fatalf("link.broken = %q", codes["link.broken"])
	}
}
