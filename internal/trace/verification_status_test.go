// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// A control verification whose verificationRefs name a test declaration that
// exists, or a requirement that a test links, while its status is unset or
// not-run warns at its status; so does a threat mitigation still planned
// whose referenced verification has existing tests. Recorded statuses,
// missing tests, production declarations and imported records are quiet.
// Mutation: treating a production declaration as a test, or ignoring the
// recorded status, fails this test.
// TRLC-LINKS: REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func TestStaleVerificationStatusWarnsWhenTestsExist(t *testing.T) {
	const testDecl = "CODE-STATE-STATE-TEST-GO--TESTDECODEREJECTS"
	bundle := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		ControlVerifications: []model.ControlVerification{
			{ID: "CV-TESTED", Status: "not-run", VerificationRefs: []string{testDecl}},
			{ID: "CV-PASSED", Status: "pass", VerificationRefs: []string{testDecl}},
			{ID: "CV-BY-REQ", VerificationRefs: []string{"REQ-B"}},
			{ID: "CV-MISSING", Status: "not-run", VerificationRefs: []string{"CODE-STATE-STATE-TEST-GO--TESTGONE", "REQ-UNTESTED"}},
			{ID: "CV-PROD", Status: "not-run", VerificationRefs: []string{"CODE-STATE-STATE-GO--DECODE"}},
			{ID: "dep::CV-IMPORTED", Status: "not-run", VerificationRefs: []string{testDecl}},
		},
		ThreatMitigations: []model.ThreatMitigation{
			{ID: "MIT-PLANNED", Status: "planned", VerificationRefs: []string{"CV-PASSED"}},
			{ID: "MIT-DONE", Status: "implemented", VerificationRefs: []string{"CV-PASSED"}},
			{ID: "MIT-UNTESTED", Status: "planned", VerificationRefs: []string{"CV-MISSING"}},
		},
	}}}
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-A"}, {ID: "REQ-B"}, {ID: "REQ-UNTESTED"}}}
	code := []codescan.CodeItem{
		{Element: testDecl, Kind: "symbol", Source: "state/state_test.go:12", AbsPath: "/m/state/state_test.go", Implements: []string{"REQ-A"}},
		{Element: "CODE-STATE-STATE-GO--DECODE", Kind: "symbol", Source: "state/state.go:4", AbsPath: "/m/state/state.go", Implements: []string{"REQ-A", "REQ-UNTESTED"}},
	}
	checks := []codescan.VerificationCheck{{ID: "VER-STATE", Verifies: []string{"REQ-A", "REQ-B"}}}

	got := map[string]validate.Diagnostic{}
	for _, d := range ValidateIntegrity(bundle, reqs, code, checks, nil) {
		if d.Code == "verification.status_stale" {
			got[d.Path] = d
		}
	}
	want := []string{
		"authoredArchitecture.controlVerifications[0].status",
		"authoredArchitecture.controlVerifications[2].status",
		"authoredArchitecture.threatMitigations[0].status",
	}
	if len(got) != len(want) {
		t.Fatalf("want verification.status_stale at %v, got %+v", want, got)
	}
	for _, path := range want {
		d, ok := got[path]
		if !ok || d.Severity != validate.SeverityWarning || !strings.Contains(d.Message, "update the status") {
			t.Fatalf("want a verification.status_stale warning at %s, got %+v", path, got)
		}
	}
	if !strings.Contains(got[want[0]].Message, testDecl) || !strings.Contains(got[want[1]].Message, "REQ-B") || !strings.Contains(got[want[2]].Message, "CV-PASSED") {
		t.Fatalf("messages must name the existing tests: %+v", got)
	}
}
