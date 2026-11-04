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

// Diagnostics report link.broken and link.child_unpublished as errors, never
// a content comparison (D23: a changed realizer at the same pin raises
// nothing), and an assumption of use of a dependency without an evidence
// link as link.aou_unverified: an error in strict mode, a warning in guided
// mode where links apply, nothing in a guided model without links or
// contract. A dependency resolved from a workspace replacement raises no
// link finding at all.
// Mutation: comparing content, reporting a local dependency or dropping
// link.aou_unverified fails this test.
// TRLC-LINKS: REQ-EMG-094, REQ-EMG-097, REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func TestLinkDiagnostics(t *testing.T) {
	child := linkChildBundle()
	child.Requirements.Requirements[0].Text = "The child shall do A1 twice."
	deps := []reqlinks.Dependency{dependency(child)}
	link := func(id, target string) model.ReqLink {
		return model.ReqLink{ID: id, Parent: "REQ-SYS-1", Child: target, Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.1.0"}}
	}
	system := systemWith(link("LNK-SAME-PIN", "dep::CAP-A"), link("LNK-BROKEN", "dep::CAP-GONE"), link("LNK-IF", "dep::IF-A"))
	system.Manifest.RequirementLinks = model.RequirementLinksStrict
	codes := map[string]string{}
	for _, d := range LinkDiagnostics(system, deps) {
		codes[d.Code] += string(d.Severity) + "|" + d.Message + "\n"
	}
	if strings.Contains(codes["link.broken"]+codes["link.child_unpublished"], "LNK-SAME-PIN") || codes["link.suspect"] != "" {
		t.Fatalf("a link at its pin was compared with content: %v", codes)
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

	local := dependency(child)
	local.Local = true
	if d := LinkDiagnostics(system, []reqlinks.Dependency{local}); len(d) != 0 {
		t.Fatalf("a workspace replacement raised link findings: %+v", d)
	}

	evidenced := systemWith(model.ReqLink{ID: "LNK-AOU", Child: "dep::AOU-A", Evidence: []string{"REQ-SYS-1"}, Status: "accepted",
		Pinned: model.ReqLinkPin{Version: "v0.1.0"}})
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

// A stale pin (version_mismatch, reported by validation) whose item is gone
// at the dependency pin is link.broken; one whose item exists raises nothing
// at composition.
// Mutation: letting the stale pin hide the broken item fails this test.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-100
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func TestLinkDiagnosticsOfStaleLinks(t *testing.T) {
	child := linkChildBundle()
	stale := func(id, target string) model.ReqLink {
		return model.ReqLink{ID: id, Parent: "REQ-SYS-1", Child: target, Status: "accepted", Pinned: model.ReqLinkPin{Version: "v0.0.9"}}
	}
	system := systemWith(stale("LNK-SAME", "dep::CAP-A"), stale("LNK-GONE", "dep::CAP-GONE"))
	codes := map[string]string{}
	for _, d := range LinkDiagnostics(system, []reqlinks.Dependency{dependency(child)}) {
		codes[d.Code] += d.Message + "\n"
	}
	if !strings.Contains(codes["link.broken"], "LNK-GONE") || strings.Contains(codes["link.broken"], "LNK-SAME") {
		t.Fatalf("link.broken = %q", codes["link.broken"])
	}
}
