// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// linkTestBundle is a child model with a published capability CAP-A, an
// assumption of use AOU-A and requirements: REQ-A realizes CAP-A, REQ-D is
// derived, REQ-S has only a stakeholder source and REQ-N has nothing.
func linkTestBundle(mode string) model.Bundle {
	b := behaviorTestBundle(nil, nil)
	b.Manifest.RequirementLinks = mode
	b.Manifest.Module.Kind = "system"
	b.Architecture.AuthoredArchitecture.ReferencedElements = []model.ReferencedElement{{ID: "REF-STAKE", Name: "Stakeholder needs"}}
	b.Architecture.Contract = model.ContractModel{
		Provides:    []model.ContractEntry{{ID: "CAP-A", Kind: "capability", Note: "A."}},
		Assumptions: []model.ContractAssumption{{ID: "AOU-A", Text: "Lab bus only."}},
	}
	b.Requirements.Requirements = []model.Requirement{
		{ID: "REQ-A", Text: "The unit shall do A.", AppliesTo: []string{"FU-A"}, Realizes: []string{"CAP-A", "AOU-A"}},
		{ID: "REQ-D", Text: "The unit shall log.", AppliesTo: []string{"FU-A"}, Derived: true, DerivedRationale: "Operability."},
		{ID: "REQ-S", Text: "The unit shall start.", AppliesTo: []string{"FU-A"}, SourceRefs: []string{"REF-STAKE"}},
		{ID: "REQ-N", Text: "The unit shall stop.", AppliesTo: []string{"FU-A"}},
	}
	b.Manifest.Publications = []model.ManifestPublication{{ID: "public", Architecture: []string{"CAP-A", "AOU-A"}, Requirements: []string{"REQ-A"}}}
	return b
}

// codesOf returns the diagnostics with the code.
func codesOf(diags []Diagnostic, code string) []Diagnostic {
	var out []Diagnostic
	for _, d := range diags {
		if d.Code == code {
			out = append(out, d)
		}
	}
	return out
}

// requirement.no_parent (D22): in strict mode a requirement that does not
// realize a local contract item (a), is not the parent of a link to a
// dependency capability (b) and is not derived with derivedRationale (c) is
// an error, with only the child repository present; guided mode makes it a
// warning where the model publishes contract items or has links and drops it
// in a standalone model. Stakeholder sourceRefs alone do not trace upward.
// The message names exactly the three options.
// TRLC-LINKS: REQ-EMG-090
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func TestRequirementNoParent(t *testing.T) {
	got := codesOf(Bundle(linkTestBundle("strict")), "requirement.no_parent")
	if len(got) != 2 || got[0].Severity != SeverityError || got[0].Path != "requirements[2]" || !strings.Contains(got[0].Message, "REQ-S") ||
		got[1].Path != "requirements[3]" || !strings.Contains(got[1].Message, "REQ-N") {
		t.Fatalf("strict: want errors for REQ-S (sourceRefs only) and REQ-N, got %+v", got)
	}
	for _, want := range []string{"realizes", "parent of a reqLinks entry", "derived: true with derivedRationale"} {
		if !strings.Contains(got[1].Message, want) {
			t.Fatalf("message %q must name %q", got[1].Message, want)
		}
	}
	if strings.Contains(got[1].Message, "sourceRefs") || strings.Contains(got[1].Message, "topLevel") {
		t.Fatalf("message %q names an option that no longer exists", got[1].Message)
	}
	got = codesOf(Bundle(linkTestBundle("")), "requirement.no_parent")
	if len(got) != 2 || got[0].Severity != SeverityWarning {
		t.Fatalf("guided with contract: want two warnings, got %+v", got)
	}
	standalone := linkTestBundle("")
	standalone.Manifest.Publications = nil
	if got := codesOf(Bundle(standalone), "requirement.no_parent"); len(got) != 0 {
		t.Fatalf("guided standalone: want none, got %+v", got)
	}
}

// The three upward traces of D22 each satisfy requirement.no_parent on
// their own: (a) realizes a local capability or assumption of use, (b) the
// parent of a reqLinks entry to a dependency capability, (c) derived with
// derivedRationale. A link to an assumption of use with the requirement as
// evidence, or derived without a rationale, is no upward trace.
// TRLC-LINKS: REQ-EMG-090
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
func TestRequirementUpwardTraceOptions(t *testing.T) {
	only := func(r model.Requirement, links ...model.ReqLink) []Diagnostic {
		b := linkTestBundle("strict")
		b.Requirements.Requirements = []model.Requirement{b.Requirements.Requirements[0], r}
		b.ReqLinks.ReqLinks = links
		return codesOf(Bundle(b), "requirement.no_parent")
	}
	if got := only(model.Requirement{ID: "REQ-X", Text: "The unit shall x.", AppliesTo: []string{"FU-A"}, Realizes: []string{"AOU-A"}}); len(got) != 0 {
		t.Fatalf("(a) realizes an assumption of use: want none, got %+v", got)
	}
	if got := only(model.Requirement{ID: "REQ-X", Text: "The unit shall x.", AppliesTo: []string{"FU-A"}},
		model.ReqLink{ID: "LNK-X", Parent: "REQ-X", Child: "dep::CAP-Z"}); len(got) != 0 {
		t.Fatalf("(b) parent of a capability link: want none, got %+v", got)
	}
	if got := only(model.Requirement{ID: "REQ-X", Text: "The unit shall x.", AppliesTo: []string{"FU-A"}, Derived: true, DerivedRationale: "Operability."}); len(got) != 0 {
		t.Fatalf("(c) derived with rationale: want none, got %+v", got)
	}
	if got := only(model.Requirement{ID: "REQ-X", Text: "The unit shall x.", AppliesTo: []string{"FU-A"}},
		model.ReqLink{ID: "LNK-X", Parent: "REQ-X", Child: "dep::AOU-Z"}, model.ReqLink{ID: "LNK-Y", Child: "dep::AOU-Z", Evidence: []string{"REQ-X"}}); len(got) != 1 {
		t.Fatalf("a link to an assumption of use is no upward trace: want one, got %+v", got)
	}
	if got := only(model.Requirement{ID: "REQ-X", Text: "The unit shall x.", AppliesTo: []string{"FU-A"}, Derived: true}); len(got) != 1 {
		t.Fatalf("derived without derivedRationale: want one, got %+v", got)
	}
	if got := only(model.Requirement{ID: "REQ-X", Text: "The unit shall x.", AppliesTo: []string{"FU-A"}, SourceRefs: []string{"REF-STAKE"}}); len(got) != 1 {
		t.Fatalf("sourceRefs only: want one, got %+v", got)
	}
}

// requirement.realizes_unknown: a realizes target that is a requirement, an
// imported alias::ID, an interface entry or unknown is an error.
// TRLC-LINKS: REQ-EMG-090
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestRequirementRealizesUnknown(t *testing.T) {
	b := linkTestBundle("strict")
	b.Architecture.Contract.Provides = append(b.Architecture.Contract.Provides, model.ContractEntry{ID: "IF-X", Kind: "interface"})
	b.Requirements.Requirements[3].Realizes = []string{"REQ-A", "dep::CAP-Z", "IF-X", "CAP-NONE"}
	got := codesOf(Bundle(b), "requirement.realizes_unknown")
	if len(got) != 4 {
		t.Fatalf("want four realizes_unknown errors, got %+v", got)
	}
	for i, want := range []string{"is a requirement", "imported ID", "not a local capability", "not a local capability"} {
		if got[i].Severity != SeverityError || !strings.Contains(got[i].Message, want) {
			t.Fatalf("diagnostic %d = %+v, want %q", i, got[i], want)
		}
	}
	for _, n := range codesOf(Bundle(b), "requirement.no_parent") {
		if n.Path == "requirements[3]" {
			t.Fatalf("invalid targets are reported once, as realizes_unknown, not also as no_parent; got %+v", n)
		}
	}
}

// contract.unrealized warns at a published capability no requirement
// realizes; a deprecated ref counts and warns contract.ref_deprecated.
// contract.realizer_unpublished is an error at a realizer the publication of
// its item does not list.
// TRLC-LINKS: REQ-EMG-091, REQ-EMG-092, REQ-EMG-093
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestContractItemRules(t *testing.T) {
	b := linkTestBundle("strict")
	b.Architecture.Contract.Provides = append(b.Architecture.Contract.Provides,
		model.ContractEntry{ID: "CAP-B", Kind: "capability", Note: "B."},
		model.ContractEntry{ID: "CAP-C", Kind: "capability", Ref: "REQ-S", Note: "C."})
	b.Manifest.Publications[0].Architecture = append(b.Manifest.Publications[0].Architecture, "CAP-B", "CAP-C")
	b.Manifest.Publications[0].Requirements = append(b.Manifest.Publications[0].Requirements, "REQ-S")
	diags := Bundle(b)
	if got := codesOf(diags, "contract.unrealized"); len(got) != 1 || got[0].Severity != SeverityWarning || !strings.Contains(got[0].Message, "CAP-B") {
		t.Fatalf("contract.unrealized = %+v", got)
	}
	if got := codesOf(diags, "contract.ref_deprecated"); len(got) != 1 || got[0].Severity != SeverityWarning || got[0].Path != "contract.provides[2].ref" {
		t.Fatalf("contract.ref_deprecated = %+v", got)
	}
	if got := codesOf(diags, "contract.realizer_unpublished"); len(got) != 0 {
		t.Fatalf("all realizers published, got %+v", got)
	}
	b.Manifest.Publications[0].Requirements = []string{"REQ-S"}
	got := codesOf(Bundle(b), "contract.realizer_unpublished")
	if len(got) != 2 || got[0].Severity != SeverityError || got[0].Path != "requirements[0]" || !strings.Contains(got[0].Message, "REQ-A") {
		t.Fatalf("contract.realizer_unpublished = %+v", got)
	}
}

// The requirement link file: a parent that is no local requirement or a
// link without parent and evidence is link.parent_unknown, a child of an
// undeclared or uncomposed dependency is link.child_unpublished, a pin other
// than the manifest's is link.version_mismatch (errors), and a claimed link
// is link.claimed_only (warning). A requirementLinks value other than strict
// or guided is model.invalid_requirement_links; an allocation warns
// composition.allocation_deprecated.
// TRLC-LINKS: REQ-EMG-090, REQ-EMG-094, REQ-EMG-099
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestRequirementLinkFileRules(t *testing.T) {
	b := linkTestBundle("strict")
	b.Manifest.Dependencies = []model.ManifestDependency{{Alias: "dep", Version: "v0.2.0"}, {Alias: "loose", Version: "v1.0.0"}}
	b.Architecture.Composition.Subsystems = []model.Subsystem{{ID: "SUB-DEP", Dependency: "dep", Publication: "public"}}
	b.Architecture.Composition.Allocations = []model.Allocation{{Requirement: "REQ-A", To: "SUB-DEP", Target: "dep::CAP-X"}}
	pin := model.ReqLinkPin{Version: "v0.2.0", Fingerprint: "sha256:v1:" + strings.Repeat("a", 64)}
	b.ReqLinks.ReqLinks = []model.ReqLink{
		{ID: "LNK-OK", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: pin, Status: "accepted"},
		{ID: "LNK-PARENT", Parent: "REQ-NONE", Child: "dep::CAP-X", Pinned: pin, Status: "accepted"},
		{ID: "LNK-EMPTY", Child: "dep::AOU-X", Pinned: pin, Status: "accepted"},
		{ID: "LNK-EVIDENCE", Child: "dep::AOU-X", Evidence: []string{"REQ-A", "CTRL-NONE"}, Pinned: pin, Status: "accepted"},
		{ID: "LNK-UNDECLARED", Parent: "REQ-A", Child: "nope::CAP-X", Pinned: pin, Status: "accepted"},
		{ID: "LNK-UNCOMPOSED", Parent: "REQ-A", Child: "loose::CAP-X", Pinned: model.ReqLinkPin{Version: "v1.0.0"}, Status: "accepted"},
		{ID: "LNK-VERSION", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: model.ReqLinkPin{Version: "v0.1.0"}, Status: "claimed"},
	}
	diags := Bundle(b)
	paths := func(code string) string {
		var p []string
		for _, d := range codesOf(diags, code) {
			p = append(p, string(d.Severity)+" "+d.Path)
		}
		return strings.Join(p, ",")
	}
	for code, want := range map[string]string{
		"link.parent_unknown":               "error reqLinks[1].parent,error reqLinks[2],error reqLinks[3].evidence[1]",
		"link.child_unpublished":            "error reqLinks[4].child,error reqLinks[5].child",
		"link.version_mismatch":             "error reqLinks[6].pinned.version",
		"link.claimed_only":                 "warning reqLinks[6].status",
		"composition.allocation_deprecated": "warning composition.allocations[0]",
		"model.invalid_requirement_links":   "",
	} {
		if got := paths(code); got != want {
			t.Errorf("%s = %q, want %q", code, got, want)
		}
	}
	b.Manifest.RequirementLinks = "lenient"
	if got := codesOf(Bundle(b), "model.invalid_requirement_links"); len(got) != 1 || got[0].Severity != SeverityError {
		t.Fatalf("model.invalid_requirement_links = %+v", got)
	}
}

// linkFileBundle is linkTestBundle composing dep at v0.2.0 with the links
// and an actor ACT-OWNER.
func linkFileBundle(links ...model.ReqLink) model.Bundle {
	b := linkTestBundle("strict")
	b.Manifest.Dependencies = []model.ManifestDependency{{Alias: "dep", Version: "v0.2.0"}}
	b.Architecture.Composition.Subsystems = []model.Subsystem{{ID: "SUB-DEP", Dependency: "dep", Publication: "public"}}
	b.Architecture.AuthoredArchitecture.Actors = []model.Actor{{ID: "ACT-OWNER", Name: "Owner"}}
	b.ReqLinks.ReqLinks = links
	return b
}

// linkPin is a valid pin of dep at v0.2.0.
var linkPin = model.ReqLinkPin{Version: "v0.2.0", Fingerprint: "sha256:v1:" + strings.Repeat("a", 64)}

// An accepted link records who reviewed it and when: no reviewedBy or no
// reviewedOn, a reviewedBy that is no actor and a reviewedOn that is no
// calendar date are each link.review_unrecorded (error) at that field; a
// complete record and a claimed link without one report nothing.
// Mutation: dropping the missing-field check, the actor check or the date
// check fails this test.
// TRLC-LINKS: REQ-EMG-111
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestAcceptedLinkReviewIsRecorded(t *testing.T) {
	accepted := func(id, by, on string) model.ReqLink {
		return model.ReqLink{ID: id, Parent: "REQ-A", Child: "dep::CAP-X", Pinned: linkPin, Status: "accepted", ReviewedBy: by, ReviewedOn: on}
	}
	b := linkFileBundle(
		accepted("LNK-OK", "ACT-OWNER", "2026-11-02"),
		accepted("LNK-NO-BY", "", "2026-11-02"),
		accepted("LNK-NO-ON", "ACT-OWNER", ""),
		accepted("LNK-ACTOR", "ACT-NOBODY", "2026-11-02"),
		accepted("LNK-DATE", "ACT-OWNER", "2026-02-30"),
		model.ReqLink{ID: "LNK-CLAIMED", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: linkPin, Status: "claimed"},
	)
	var got []string
	for _, d := range codesOf(Bundle(b), "link.review_unrecorded") {
		got = append(got, string(d.Severity)+" "+d.Path+" "+d.Message)
	}
	want := []string{
		"error reqLinks[1].reviewedBy link LNK-NO-BY is accepted but names no reviewedBy; record the review with engws review-links --accept LNK-NO-BY --by ACT-... (or set status claimed)",
		"error reqLinks[2].reviewedOn link LNK-NO-ON is accepted but names no reviewedOn; record the review with engws review-links --accept LNK-NO-ON (or set status claimed)",
		`error reqLinks[3].reviewedBy link LNK-ACTOR reviewedBy "ACT-NOBODY" is not an actor of this model (architecture.actors)`,
		`error reqLinks[4].reviewedOn link LNK-DATE reviewedOn "2026-02-30" is not a YYYY-MM-DD calendar date`,
	}
	if strings.Join(got, "\n") != strings.Join(want, "\n") {
		t.Fatalf("link.review_unrecorded =\n%s\nwant\n%s", strings.Join(got, "\n"), strings.Join(want, "\n"))
	}
}

// A link to a capability needs a parent requirement (link.parent_missing),
// even when it carries evidence; a link to an assumption of use with
// evidence only is complete.
// Mutation: accepting evidence instead of a parent for a capability link
// fails this test.
// TRLC-LINKS: REQ-EMG-112
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestCapabilityLinkNeedsParent(t *testing.T) {
	b := linkFileBundle(
		model.ReqLink{ID: "LNK-CAP", Child: "dep::CAP-X", Evidence: []string{"REQ-A"}, Pinned: linkPin, Status: "claimed"},
		model.ReqLink{ID: "LNK-AOU", Child: "dep::AOU-X", Evidence: []string{"REQ-A"}, Pinned: linkPin, Status: "claimed"},
	)
	got := codesOf(Bundle(b), "link.parent_missing")
	if len(got) != 1 || got[0].Path != "reqLinks[0].parent" || got[0].Severity != SeverityError || !strings.Contains(got[0].Message, "LNK-CAP") {
		t.Fatalf("link.parent_missing = %+v", got)
	}
}

// The remaining link file rules: a duplicate link id is model.duplicate_id,
// a status other than claimed or accepted is link.invalid_status (both
// errors), and an assumption of use without an AOU-* id or text is
// model.invalid_assumption.
// Mutation: dropping the duplicate check, the status check or the
// assumption check fails this test.
// TRLC-LINKS: REQ-EMG-091, REQ-EMG-094
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func TestLinkFileIntegrityRules(t *testing.T) {
	b := linkFileBundle(
		model.ReqLink{ID: "LNK-A", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: linkPin, Status: "claimed"},
		model.ReqLink{ID: "LNK-A", Parent: "REQ-A", Child: "dep::CAP-Y", Pinned: linkPin, Status: "claimed"},
		model.ReqLink{ID: "LNK-B", Parent: "REQ-A", Child: "dep::CAP-X", Pinned: linkPin, Status: "maybe"},
	)
	b.Architecture.Contract.Assumptions = append(b.Architecture.Contract.Assumptions,
		model.ContractAssumption{ID: "ASSUME-X", Text: "Wrong prefix."}, model.ContractAssumption{ID: "AOU-EMPTY"})
	diags := Bundle(b)
	check := func(code, want string) {
		t.Helper()
		var got []string
		for _, d := range codesOf(diags, code) {
			got = append(got, string(d.Severity)+" "+d.Path)
		}
		if strings.Join(got, ",") != want {
			t.Errorf("%s = %q, want %q", code, strings.Join(got, ","), want)
		}
	}
	check("model.duplicate_id", "error reqLinks[1].id")
	check("link.invalid_status", "error reqLinks[2].status")
	check("model.invalid_assumption", "error contract.assumptions[1],error contract.assumptions[2]")
}
