// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
)

// linkChildBundle is a child model with two capabilities, an interface
// entry, two assumptions of use and requirements realizing them; publication
// public exposes CAP-A, AOU-A, IF-A and the realizers REQ-A1 and REQ-A2.
func linkChildBundle() model.Bundle {
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
		{ID: "REQ-A3", Text: "The child shall do A3 privately.", Realizes: []string{"CAP-A"}},
	}
	b.Manifest.Publications = []model.ManifestPublication{{
		ID: "public", Architecture: []string{"CAP-A", "AOU-A", "IF-A"},
		Requirements: []string{"REQ-A1", "REQ-A2"},
	}}
	return b
}

// systemWith returns a system bundle pinning dep at v0.1.0 with the links.
func systemWith(links ...model.ReqLink) model.Bundle {
	var b model.Bundle
	b.Manifest.Dependencies = []model.ManifestDependency{
		{Alias: "dep", Path: "example.com/dep@v0", Version: "v0.1.0", Publications: []string{"public"}},
	}
	b.Requirements.Requirements = []model.Requirement{{ID: "REQ-SYS-1", Text: "The system shall do A."}}
	b.ReqLinks.ReqLinks = links
	return b
}

// dependency is linkChildBundle resolved as alias dep at v0.1.0.
func dependency(child model.Bundle) reqlinks.Dependency {
	return reqlinks.Dependency{Alias: "dep", Version: "v0.1.0", Bundle: child, Publications: child.Manifest.Publications}
}

// Diagnostics reports link.suspect (with the current item and realizer texts
// and both fingerprints), link.broken, link.child_unpublished and
// link.fingerprint_outdated (an older or unknown canonical form) as errors,
// and an assumption of use of a dependency without an evidence link as
// link.aou_unverified: an error in strict mode, a warning in guided mode
// where links apply, nothing in a guided model without links or contract.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-095, REQ-EMG-097, REQ-EMG-109
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func TestLinkDiagnostics(t *testing.T) {
	child := linkChildBundle()
	deps := []reqlinks.Dependency{dependency(child)}
	zero := "sha256:v1:" + strings.Repeat("0", 64)
	system := systemWith(
		model.ReqLink{ID: "LNK-SUSPECT", Parent: "REQ-SYS-1", Child: "dep::CAP-A", Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: zero}},
		model.ReqLink{ID: "LNK-BROKEN", Parent: "REQ-SYS-1", Child: "dep::CAP-GONE", Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: zero}},
		model.ReqLink{ID: "LNK-IF", Parent: "REQ-SYS-1", Child: "dep::IF-A", Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: zero}},
	)
	system.Manifest.RequirementLinks = model.RequirementLinksStrict
	codes := map[string]string{}
	for _, d := range LinkDiagnostics(system, deps) {
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

	// A fingerprint of an older or unknown canonical form is not compared:
	// link.fingerprint_outdated (error), never link.suspect.
	outdated := systemWith(model.ReqLink{ID: "LNK-OLD", Parent: "REQ-SYS-1", Child: "dep::CAP-A", Status: "accepted",
		Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: "sha256:" + strings.Repeat("0", 64)}})
	var old []string
	for _, d := range LinkDiagnostics(outdated, deps) {
		if strings.HasPrefix(d.Code, "link.") && d.Code != "link.aou_unverified" {
			old = append(old, d.Code+"|"+string(d.Severity)+"|"+d.Message)
		}
	}
	if len(old) != 1 || !strings.HasPrefix(old[0], "link.fingerprint_outdated|error|") ||
		!strings.Contains(old[0], "engws review-links --accept LNK-OLD") || !strings.Contains(old[0], "unversioned") {
		t.Fatalf("outdated fingerprint: %+v", old)
	}

	evidenced := systemWith(model.ReqLink{ID: "LNK-AOU", Child: "dep::AOU-A", Evidence: []string{"REQ-SYS-1"}, Status: "accepted",
		Pinned: model.ReqLinkPin{Version: "v0.1.0", Fingerprint: reqlinks.Fingerprint(reqlinks.PublishedItems(child, child.Manifest.Publications[0])["AOU-A"])}})
	if d := LinkDiagnostics(evidenced, deps); len(d) != 0 {
		t.Fatalf("evidenced assumption: %+v", d)
	}
	noEvidence := evidenced
	noEvidence.ReqLinks.ReqLinks = []model.ReqLink{evidenced.ReqLinks.ReqLinks[0]}
	noEvidence.ReqLinks.ReqLinks[0].Evidence = nil
	noEvidence.ReqLinks.ReqLinks[0].Parent = "REQ-SYS-1"
	if d := LinkDiagnostics(noEvidence, deps); len(d) != 1 || d[0].Code != "link.aou_unverified" || d[0].Severity != "warning" {
		t.Fatalf("guided link without evidence: %+v", d)
	}
	if d := LinkDiagnostics(systemWith(), deps); len(d) != 0 {
		t.Fatalf("guided model without links or contract: %+v", d)
	}
}

// The composed diagnostics of a stale pin (version_mismatch) report the
// content state at the resolved version: link.suspect with the current text,
// link.broken, nothing for unchanged content.
// Mutation: not reporting the suspect or broken content of a
// version_mismatch link fails this test.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-100
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestLinkDiagnosticsOfStaleLinks(t *testing.T) {
	child := linkChildBundle()
	current := reqlinks.Fingerprint(reqlinks.PublishedItems(child, child.Manifest.Publications[0])["CAP-A"])
	zero := "sha256:v1:" + strings.Repeat("0", 64)
	stale := func(id, target, fingerprint string) model.ReqLink {
		return model.ReqLink{ID: id, Parent: "REQ-SYS-1", Child: target, Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.0.9", Fingerprint: fingerprint}}
	}
	system := systemWith(stale("LNK-SAME", "dep::CAP-A", current), stale("LNK-CHANGED", "dep::CAP-A", zero), stale("LNK-GONE", "dep::CAP-GONE", current))
	codes := map[string]string{}
	for _, d := range LinkDiagnostics(system, []reqlinks.Dependency{dependency(child)}) {
		codes[d.Code] += d.Message + "\n"
	}
	if !strings.Contains(codes["link.suspect"], "LNK-CHANGED") || !strings.Contains(codes["link.suspect"], "REQ-A1: The child shall do A1.") || strings.Contains(codes["link.suspect"], "LNK-SAME") {
		t.Fatalf("link.suspect = %q", codes["link.suspect"])
	}
	if !strings.Contains(codes["link.broken"], "LNK-GONE") {
		t.Fatalf("link.broken = %q", codes["link.broken"])
	}
}
