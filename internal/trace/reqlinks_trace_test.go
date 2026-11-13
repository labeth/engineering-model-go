// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// requirement.unsatisfied: in a composing system with requirementLinks
// strict, a requirement without own code or test, without a link naming it
// as parent and not derived is an error; one with code, one with a test, a
// link parent and a derived one are not reported. Guided mode without links
// or contract items reports nothing; a model without subsystems never.
// TRLC-LINKS: REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func TestRequirementUnsatisfied(t *testing.T) {
	var bundle model.Bundle
	bundle.Manifest.RequirementLinks = model.RequirementLinksStrict
	bundle.Architecture.Composition.Subsystems = []model.Subsystem{{ID: "SUB-A", Dependency: "dep", Publication: "public"}}
	bundle.ReqLinks.ReqLinks = []model.ReqLink{{ID: "LNK-1", Parent: "REQ-LINKED", Child: "dep::CAP-A"}}
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{
		{ID: "REQ-CODE"}, {ID: "REQ-TEST"}, {ID: "REQ-LINKED"}, {ID: "REQ-DERIVED", Derived: true}, {ID: "REQ-NONE"},
	}}
	code := []codescan.CodeItem{{Element: "c", Source: "a.go:1", Implements: []string{"REQ-CODE"}}}
	tests := []codescan.VerificationCheck{{ID: "t", Verifies: []string{"REQ-TEST"}}}
	unsatisfied := func(b model.Bundle) []validate.Diagnostic {
		var out []validate.Diagnostic
		for _, d := range ValidateIntegrity(b, reqs, code, tests) {
			if d.Code == "requirement.unsatisfied" {
				out = append(out, d)
			}
		}
		return out
	}
	got := unsatisfied(bundle)
	if len(got) != 1 || got[0].Severity != validate.SeverityError || got[0].Path != "requirements[4]" {
		t.Fatalf("strict: want one error at REQ-NONE, got %+v", got)
	}
	guided := bundle
	guided.Manifest.RequirementLinks = ""
	if got := unsatisfied(guided); len(got) != 1 || got[0].Severity != validate.SeverityWarning {
		t.Fatalf("guided with links: want one warning, got %+v", got)
	}
	guided.ReqLinks.ReqLinks = nil
	if got := unsatisfied(guided); len(got) != 0 {
		t.Fatalf("guided without links or contract: want none, got %+v", got)
	}
	leaf := bundle
	leaf.Architecture.Composition.Subsystems = nil
	if got := unsatisfied(leaf); len(got) != 0 {
		t.Fatalf("model without subsystems: want none, got %+v", got)
	}
}

// The trace follows the upward-trace rule (D22): unparented counts the
// requirements that fail all of (a) realizes a local contract item, (b) the
// parent of a link to a dependency capability and (c) derived with
// derivedRationale; a requirement with only stakeholder sourceRefs is
// unparented. parents is the upward trace only: the realized items, none
// for a link parent (its links are its children) or a sourceRefs entry.
// TRLC-LINKS: REQ-EMG-105
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestUnparentedFollowsUpwardTraceRule(t *testing.T) {
	var bundle model.Bundle
	bundle.Manifest.RequirementLinks = model.RequirementLinksStrict
	bundle.Architecture.Contract.Provides = []model.ContractEntry{{ID: "CAP-A", Kind: "capability"}}
	bundle.ReqLinks.ReqLinks = []model.ReqLink{{ID: "LNK-1", Parent: "REQ-LINKED", Child: "dep::CAP-Z"}}
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{
		{ID: "REQ-REALIZES", Realizes: []string{"CAP-A"}}, {ID: "REQ-LINKED"}, {ID: "REQ-DERIVED", Derived: true, DerivedRationale: "Ops."},
		{ID: "REQ-SOURCE", SourceRefs: []string{"REF-STAKE"}}, {ID: "REQ-NONE"},
	}}
	m := buildTraceMatrix(bundle, reqs, nil, nil)
	if m.Summary.Unparented != 2 {
		t.Fatalf("want REQ-SOURCE and REQ-NONE unparented, got %d", m.Summary.Unparented)
	}
	parents := map[string][]string{}
	for _, r := range m.Requirements {
		parents[r.ID] = r.Parents
	}
	if len(parents["REQ-REALIZES"]) != 1 || parents["REQ-REALIZES"][0] != "CAP-A" || len(parents["REQ-LINKED"]) != 0 || len(parents["REQ-SOURCE"]) != 0 {
		t.Fatalf("parents %v", parents)
	}
}
