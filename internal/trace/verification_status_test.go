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

	stale := diagsOf(ValidateIntegrity(bundle, Evidence{reqs, code, checks}), "verification.status_stale")
	messages := map[string]string{}
	for _, d := range stale {
		messages[d.Path] = d.Message
	}
	paths := []string{
		"authoredArchitecture.controlVerifications[0].status",
		"authoredArchitecture.controlVerifications[2].status",
		"authoredArchitecture.threatMitigations[0].status",
	}
	requireEqual(t, pathsOf(stale, "verification.status_stale"), paths)
	requireEqual(t, severitiesByCode(stale), map[string]validate.Severity{"verification.status_stale": validate.SeverityWarning})
	for _, path := range paths {
		requireEqual(t, strings.Contains(messages[path], "update the status"), true)
	}
	// The messages name the existing tests.
	requireEqual(t, strings.Contains(messages[paths[0]], testDecl) && strings.Contains(messages[paths[1]], "REQ-B"), true)
	requireEqual(t, strings.Contains(messages[paths[2]], "CV-PASSED"), true)
}
