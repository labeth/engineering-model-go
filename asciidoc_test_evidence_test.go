// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"github.com/labeth/engineering-model-go/model"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-010, REQ-EMG-030
func TestPublicationKeepsTestCodeOutOfRuntimeEvidence(t *testing.T) {
	for _, path := range []string{"api_test.go", "frame.test.cjs", "frame.spec.mjs", "tb_capture.v", "tests/auth_helper.go"} {
		t.Run(path, func(t *testing.T) {
			testCode := []inferredCodeItem{
				{Kind: "source_file", Owner: "FU-A", Source: path},
				{Kind: "symbol", Owner: "FU-A", Source: path + ":17", Implements: []string{"REQ-A"}},
			}
			reqs := []model.Requirement{{ID: "REQ-A", AppliesTo: []string{"FU-A"}}}
			checks := []inferredVerificationCheck{{ID: "VER-A", Verifies: []string{"REQ-A"}, CodeElements: []string{path + ":17"}, Status: "partial"}}
			a := model.AuthoredArchitecture{
				FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A"}},
				FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A"}, {ID: "FU-B"}},
				Mappings:         []model.Mapping{{Type: "depends_on", From: "FU-A", To: "FU-B"}},
			}
			security := []asciidocSecurityPathRow{{AttackVectorID: "AV-A", AttackVector: "Attack", TargetID: "FU-A", Target: "Unit"}}
			requirement := buildRequirementCoverageMermaid(reqs, nil, testCode, checks, nil, nil)
			if !strings.Contains(requirement, "VERCODE_") || !strings.Contains(requirement, "partial") || !strings.Contains(requirement, ": 17") {
				t.Fatalf("verification evidence lost: %s", requirement)
			}
			if strings.Contains(requirement, "code trace") || strings.Contains(requirement, "runtime evidence") {
				t.Fatalf("test-only owner gained implementation evidence: %s", requirement)
			}
			for name, diagram := range map[string]string{
				"group":    buildFunctionalGroupDependencyMermaid(a, "FG-A", nil, testCode),
				"security": buildSecurityPathMermaid(security, nil, testCode),
			} {
				if strings.Contains(diagram, "runtime (inferred)") || strings.Contains(diagram, "CODE_") {
					t.Fatalf("%s retained test runtime: %s", name, diagram)
				}
			}
			if len(buildOwnerEvidence(nil, testCode)) != 0 || len(buildSecurityObservabilityRows(nil, testCode)) != 0 {
				t.Fatal("test code created owner/security evidence")
			}
			mixed := append(append([]inferredCodeItem{}, testCode...), inferredCodeItem{Kind: "symbol", Owner: "FU-A", Source: "production.go:9", Implements: []string{"REQ-A"}})
			requirement = buildRequirementCoverageMermaid(reqs, nil, mixed, checks, nil, nil)
			if !strings.Contains(requirement, "production.go: 9") || !strings.Contains(requirement, "code trace") || !strings.Contains(requirement, "VERCODE_") {
				t.Fatalf("mixed evidence lost production or verification: %s", requirement)
			}
			explicit := buildRequirementCoverageMermaid(reqs, []inferredRuntimeItem{{Owner: "FU-A", Name: "authored-service"}}, testCode, checks, nil, nil)
			if !strings.Contains(explicit, "authored-service") {
				t.Fatal("independent runtime evidence lost")
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-030
func TestVerificationCodeUsesAbsoluteSourceContext(t *testing.T) {
	if !isVerificationCodeItem(inferredCodeItem{Source: "helper.go:3", AbsPath: "/workspace/tests/helper.go"}) {
		t.Fatal("test source root context lost")
	}
	if isVerificationCodeItem(inferredCodeItem{Source: "contest.go:3", AbsPath: "/workspace/src/contest.go"}) {
		t.Fatal("production path classified as test")
	}
	root := t.TempDir()
	dir := filepath.Join(root, "tests")
	if err := os.Mkdir(dir, 0700); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(dir, "helper.go")
	if err := os.WriteFile(path, []byte("// ENGMODEL-OWNER-UNIT: FU-A\npackage fixture\n// TRLC-LINKS: REQ-TEST-001\nfunc helper() {}\n"), 0600); err != nil {
		t.Fatal(err)
	}
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{CodeSources: []string{path}}}}
	code, diags := inferCodeItems(bundle, "")
	if len(diags) != 0 || len(code) != 2 {
		t.Fatalf("inference failed: %+v %+v", code, diags)
	}
	for _, item := range code {
		if item.AbsPath != path || !isVerificationCodeItem(item) {
			t.Fatalf("test file root context lost: %+v", item)
		}
	}
	matrix := buildTraceMatrix(bundle, model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-TEST-001"}}}, code, nil, nil)
	if matrix.Summary.Implemented != 0 {
		t.Fatalf("test-root helper credited as production: %+v", matrix)
	}
}
