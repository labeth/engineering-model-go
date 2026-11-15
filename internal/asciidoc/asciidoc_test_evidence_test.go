// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
)

// testCodeCase is a test file that implements and verifies a requirement of a
// unit, with the diagrams inputs around it.
type testCodeCase struct {
	testCode []codescan.CodeItem
	reqs     []model.Requirement
	checks   []codescan.VerificationCheck
	arch     model.AuthoredArchitecture
	security []asciidocSecurityPathRow
}

// newTestCodeCase is the case of a test file at the path.
func newTestCodeCase(path string) testCodeCase {
	return testCodeCase{
		testCode: []codescan.CodeItem{
			{Kind: "source_file", Owner: "FU-A", Source: path},
			{Kind: "symbol", Owner: "FU-A", Source: path + ":17", Implements: []string{"REQ-A"}},
		},
		reqs:   []model.Requirement{{ID: "REQ-A", AppliesTo: []string{"FU-A"}}},
		checks: []codescan.VerificationCheck{{ID: "VER-A", Verifies: []string{"REQ-A"}, CodeElements: []string{path + ":17"}, Status: "partial"}},
		arch: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A"}, {ID: "FU-B"}},
			Mappings:         []model.Mapping{{Type: "depends_on", From: "FU-A", To: "FU-B"}},
		},
		security: []asciidocSecurityPathRow{{AttackVectorID: "AV-A", AttackVector: "Attack", TargetID: "FU-A", Target: "Unit"}},
	}
}

// requireVerificationOnly fails the test unless test code shows as verification
// evidence and not as implementation or runtime evidence.
func (c testCodeCase) requireVerificationOnly(t *testing.T) {
	t.Helper()
	requirement := buildRequirementCoverageMermaid(c.reqs, coverageSources{code: c.testCode, verification: c.checks})
	lost := "verification evidence lost: " + requirement
	requireContains(t, requirement, docCheck{"VERCODE_", lost}, docCheck{"partial", lost}, docCheck{": 17", lost})
	gained := "test-only owner gained implementation evidence: " + requirement
	requireNotContains(t, requirement, docCheck{"code trace", gained}, docCheck{"runtime evidence", gained})
}

// requireNoInferredRuntime fails the test when a diagram infers a runtime from
// test code, or test code creates owner or security evidence.
func (c testCodeCase) requireNoInferredRuntime(t *testing.T) {
	t.Helper()
	diagrams := map[string]string{
		"group":    buildFunctionalGroupDependencyMermaid(c.arch, "FG-A", realizationEvidence{code: c.testCode}),
		"security": buildSecurityPathMermaid(c.security, nil, c.testCode),
	}
	for name, diagram := range diagrams {
		requireNotContains(t, diagram, docCheck{"runtime (inferred)", name + " retained test runtime: " + diagram}, docCheck{"CODE_", name + " retained test runtime: " + diagram})
	}
	if len(buildOwnerEvidence(nil, c.testCode)) != 0 || len(buildSecurityObservabilityRows(nil, c.testCode)) != 0 {
		t.Fatal("test code created owner/security evidence")
	}
}

// requireMixedEvidence fails the test unless production code next to test code
// keeps its implementation evidence, and independent runtime evidence is kept.
func (c testCodeCase) requireMixedEvidence(t *testing.T) {
	t.Helper()
	mixed := append(append([]codescan.CodeItem{}, c.testCode...), codescan.CodeItem{Kind: "symbol", Owner: "FU-A", Source: "production.go:9", Implements: []string{"REQ-A"}})
	requirement := buildRequirementCoverageMermaid(c.reqs, coverageSources{code: mixed, verification: c.checks})
	lost := "mixed evidence lost production or verification: " + requirement
	requireContains(t, requirement, docCheck{"production.go: 9", lost}, docCheck{"code trace", lost}, docCheck{"VERCODE_", lost})
	explicit := buildRequirementCoverageMermaid(c.reqs, coverageSources{runtime: []codescan.RuntimeItem{{Owner: "FU-A", Name: "authored-service"}}, code: c.testCode, verification: c.checks})
	requireContains(t, explicit, docCheck{"authored-service", "independent runtime evidence lost"})
}

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-010, REQ-EMG-030
func TestPublicationKeepsTestCodeOutOfRuntimeEvidence(t *testing.T) {
	for _, path := range []string{"api_test.go", "frame.test.cjs", "frame.spec.mjs", "tb_capture.v", "tests/auth_helper.go"} {
		t.Run(path, func(t *testing.T) {
			c := newTestCodeCase(path)
			c.requireVerificationOnly(t)
			c.requireNoInferredRuntime(t)
			c.requireMixedEvidence(t)
		})
	}
}
