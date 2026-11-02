// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008
func paymentsServerWithRepoRoot(t *testing.T, repoRoot string) *Server {
	t.Helper()
	s := NewServer()
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0",
		"id":      1,
		"method":  "initialize",
		"params": map[string]any{
			"initializationOptions": map[string]any{
				"modelPath":        filepath.Join("..", "..", "examples", "payments-engineering-sample", "engmod.yml"),
				"requirementsPath": filepath.Join("..", "..", "examples", "payments-engineering-sample", "model", "requirements.yml"),
				"repoRoot":         repoRoot,
			},
		},
	})
	if resp["error"] != nil {
		t.Fatalf("initialize returned error: %+v", resp["error"])
	}
	return s
}

// TRLC-LINKS: REQ-EMG-007, REQ-EMG-008, REQ-EMG-010
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func TestFilesOwnerReadsOnlyOwnerMarker(t *testing.T) {
	root := t.TempDir()
	write := func(name, content string) {
		t.Helper()
		if err := os.WriteFile(filepath.Join(root, name), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	// FU-CHECKOUT appears first in a doc comment; the marker names FU-RISK-SCORING.
	write("doc_mention.go", "// Package sample talks to FU-CHECKOUT over HTTP.\n// ENGMODEL-OWNER-UNIT: FU-RISK-SCORING\npackage sample\n")
	write("python_owner.py", "\"\"\"Calls FU-CHECKOUT.\"\"\"\n# ENGMODEL-OWNER-UNIT: FU-SUPPORT-REVIEW\n")
	write("unknown_owner.go", "// ENGMODEL-OWNER-UNIT: FU-NOT-IN-MODEL\npackage sample\n")
	write("no_marker.go", "// Mentions FU-CHECKOUT and FU-RISK-SCORING only in prose.\npackage sample\n")

	s := paymentsServerWithRepoRoot(t, root)
	for _, tc := range []struct {
		file  string
		owner string
		known bool
		line  float64
	}{
		{"doc_mention.go", "FU-RISK-SCORING", true, 2},
		{"python_owner.py", "FU-SUPPORT-REVIEW", true, 2},
		{"unknown_owner.go", "FU-NOT-IN-MODEL", false, 1},
		{"no_marker.go", "", false, 0},
	} {
		t.Run(tc.file, func(t *testing.T) {
			payload := callToolPayload(t, s, "files.owner", map[string]any{"path": tc.file})
			if payload["ownerFunctionalUnit"] != tc.owner {
				t.Fatalf("owner = %v, want %q", payload["ownerFunctionalUnit"], tc.owner)
			}
			if payload["ownerKnown"] != tc.known {
				t.Fatalf("ownerKnown = %v, want %v", payload["ownerKnown"], tc.known)
			}
			if payload["markerLine"] != tc.line {
				t.Fatalf("markerLine = %v, want %v", payload["markerLine"], tc.line)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-045, REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func TestModelAuthoringContractIncludesValidationRules(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := callToolPayload(t, s, "model.authoringContract", map[string]any{})
	rules, ok := payload["rules"].(map[string]any)
	if !ok {
		t.Fatalf("missing rules: %+v", payload)
	}
	want := validate.AuthoringRules()
	enums, _ := rules["enums"].(map[string]any)
	if len(enums) != len(want.Enums) {
		t.Fatalf("enums = %d, want %d", len(enums), len(want.Enums))
	}
	riskStatus, _ := enums["risk.status"].([]any)
	if len(riskStatus) != len(want.Enums["risk.status"]) {
		t.Fatalf("risk.status = %v", riskStatus)
	}
	pairs, _ := rules["relationshipPairs"].([]any)
	if len(pairs) != len(want.RelationshipPairs) || len(pairs) == 0 {
		t.Fatalf("relationshipPairs = %d, want %d", len(pairs), len(want.RelationshipPairs))
	}
	first, _ := pairs[0].(map[string]any)
	if first["type"] == "" || first["from"] == "" || first["to"] == "" {
		t.Fatalf("pair shape: %+v", first)
	}
	if fieldRules, _ := rules["fieldRules"].([]any); len(fieldRules) != len(want.FieldRules) {
		t.Fatalf("fieldRules = %v", fieldRules)
	}
}

// The authoring contract rules list the fields each entity kind accepts, the
// manifest publication sections (decisions cannot be published) and the
// field rules for boundaryCrossing, behavior.event_unused, the
// lint.multiple_responses field-list exemption and the strict EARS system term.
// Mutation: dropping fieldSets from validate.Rules fails this test.
// TRLC-LINKS: REQ-EMG-087, REQ-EMG-045
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func TestModelAuthoringContractListsFieldSetsAndPublicationSections(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := callToolPayload(t, s, "model.authoringContract", map[string]any{})
	rules, _ := payload["rules"].(map[string]any)
	sets, _ := rules["fieldSets"].(map[string]any)
	fields := func(kind string) string {
		list, _ := sets[kind].([]any)
		out := make([]string, 0, len(list))
		for _, f := range list {
			out = append(out, f.(string))
		}
		return strings.Join(out, ",")
	}
	if got := fields("control"); got != "id,name,category,description" {
		t.Fatalf("fieldSets.control = %q", got)
	}
	risk := "," + fields("risk") + ","
	for _, f := range []string{"id", "status", "response", "owner", "evidence", "residualRisk", "rationale", "acceptedBy"} {
		if !strings.Contains(risk, ","+f+",") {
			t.Fatalf("fieldSets.risk lacks %s: %s", f, risk)
		}
	}
	for _, kind := range []string{"threatScenario", "flowStep", "flow", "functionalUnit", "requirement", "decision", "acceptanceRecord", "controlVerification", "publication"} {
		if fields(kind) == "" {
			t.Fatalf("fieldSets has no %s: %v", kind, sets)
		}
	}
	if !strings.Contains(","+fields("flowStep")+",", ",trustBoundaryRef,") {
		t.Fatalf("fieldSets.flowStep = %s", fields("flowStep"))
	}
	sections, _ := rules["publicationSections"].([]any)
	var got []string
	for _, x := range sections {
		got = append(got, x.(string))
	}
	if strings.Join(got, ",") != "architecture,behavior,assurance,compliance,requirements,views" {
		t.Fatalf("publicationSections = %v", got)
	}
	fieldRules, _ := rules["fieldRules"].([]any)
	all := ""
	for _, r := range fieldRules {
		all += r.(string) + "\n"
	}
	for _, want := range []string{"boundaryCrossing: true needs trustBoundaryRef", "behavior.event_unused counts", "properties", "lint.multiple_responses", "with, including or of", "strict EARS", "catalog system", "Decisions cannot be published", "acceptedBy"} {
		if !strings.Contains(all, want) {
			t.Fatalf("fieldRules lack %q:\n%s", want, all)
		}
	}
}
