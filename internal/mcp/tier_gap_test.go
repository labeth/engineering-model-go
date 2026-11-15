// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tier.gap {target} returns the tier gap of the model (current and target
// tier, blocking findings with code and requiredFrom, byCode, aviation
// records below the floor and dependencies below the target); trace.matrix
// diagnostics carry deferred and requiredFrom; an unknown target and a
// missing one are refused, and the input schema is strict with repo.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTierGapTool(t *testing.T) {
	root := copyLinksExample(t)
	// The examples declare lifecycle mvp; the case lowers child-a to prototype
	// (the tier decision follows) and leaves child-b without a tier, the kept
	// library behaviour of a dependency without one.
	for repo, tier := range map[string]string{"system": "mvp", "child-a": "prototype", "child-b": ""} {
		setRepoTier(t, filepath.Join(root, "repos", repo), tier)
	}
	s := linksServer(t, root, "system")
	assertTierGapPayload(t, s.callToolPayload(t, "tier.gap", map[string]any{"target": "product"}))
	assertTierGapRefusals(t, s)
	assertTierGapSchema(t)
	assertMatrixTierFields(t, s.callToolPayload(t, "trace.matrix", map[string]any{"summaryOnly": true}))
}

// setRepoTier rewrites the lifecycle of the manifest in dir to tier (none when
// tier is empty) and the tier decision to match.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func setRepoTier(t *testing.T, dir, tier string) {
	t.Helper()
	manifest := filepath.Join(dir, "engmod.yml")
	data, err := os.ReadFile(manifest)
	if err != nil {
		t.Fatal(err)
	}
	text := string(data)
	text = text[:strings.Index(text, "lifecycle:")]
	if tier != "" {
		text += "lifecycle:\n  tier: " + tier + "\n  bundles: [foundation]\n  tierDecision: DEC-TIER-" + strings.ToUpper(tier) + "\n"
		rewriteTierDecision(t, filepath.Join(dir, "model", "decisions.yml"), tier)
	}
	if err := os.WriteFile(manifest, []byte(text), 0o644); err != nil {
		t.Fatal(err)
	}
}

// rewriteTierDecision points the MVP tier decision in the file at tier.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func rewriteTierDecision(t *testing.T, decisions, tier string) {
	t.Helper()
	dec, err := os.ReadFile(decisions)
	if err != nil {
		t.Fatal(err)
	}
	out := strings.NewReplacer("DEC-TIER-MVP", "DEC-TIER-"+strings.ToUpper(tier), "Tier: MVP", "Tier: "+tier).Replace(string(dec))
	if err := os.WriteFile(decisions, []byte(out), 0o644); err != nil {
		t.Fatal(err)
	}
}

// assertTierGapPayload checks the tiers, dependencies, byCode and blocking
// list of a tier.gap payload.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertTierGapPayload(t *testing.T, gap map[string]any) {
	t.Helper()
	if gap["currentTier"] != "mvp" || gap["targetTier"] != "product" {
		t.Fatalf("tier.gap: %+v", gap)
	}
	deps, _ := gap["dependencies"].([]any)
	aliases := []string{}
	for _, d := range deps {
		aliases = append(aliases, d.(map[string]any)["alias"].(string)+":"+d.(map[string]any)["tier"].(string))
	}
	if strings.Join(aliases, ",") != "encoder:prototype,bridge:" {
		t.Fatalf("dependencies %v", aliases)
	}
	byCode, _ := gap["byCode"].(map[string]any)
	if byCode["composition.dependency_below_tier"] != float64(1) {
		t.Fatalf("byCode %+v", byCode)
	}
	if _, ok := gap["blocking"].([]any); !ok {
		t.Fatalf("blocking is not a list: %+v", gap)
	}
}

// assertTierGapRefusals checks the refusals of an unknown target, a missing
// target and an unknown argument.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertTierGapRefusals(t *testing.T, s *Server) {
	t.Helper()
	if msg := s.toolError(t, "tier.gap", map[string]any{"target": "beta"}); !strings.Contains(msg, "beta") {
		t.Fatalf("unknown target: %q", msg)
	}
	if msg := s.toolError(t, "tier.gap", map[string]any{}); !strings.Contains(msg, "needs target") {
		t.Fatalf("missing target: %q", msg)
	}
	if msg := s.toolError(t, "tier.gap", map[string]any{"target": "product", "x": "1"}); msg == "" {
		t.Fatal("an unknown argument was accepted")
	}
}

// assertTierGapSchema checks that the tier.gap input schema is strict with repo.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertTierGapSchema(t *testing.T) {
	t.Helper()
	schema := inputSchemaForTool("tier.gap")
	props := schema["properties"].(map[string]any)
	if props["target"] == nil || props["repo"] == nil || schema["additionalProperties"] != false {
		t.Fatalf("schema %+v", schema)
	}
}

// assertMatrixTierFields checks that the trace.matrix tier diagnostic carries
// requiredFrom and severity.
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertMatrixTierFields(t *testing.T, matrix map[string]any) {
	t.Helper()
	found := false
	for _, d := range matrix["diagnostics"].([]any) {
		row := d.(map[string]any)
		if row["code"] == "composition.dependency_below_tier" {
			found = row["requiredFrom"] == "product" && row["severity"] == "warning"
		}
	}
	if !found {
		t.Fatalf("trace.matrix diagnostics lack the tier fields: %+v", matrix["diagnostics"])
	}
}

// The repository index the file tools search leaves out .engflow
// (engineering-flow's experiment state and variant worktrees): a file
// there naming a requirement is not listed, one in the model's tree is.
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-MCP-SERVER
func TestRepoIndexSkipsEngflowState(t *testing.T) {
	root := copyLinksExample(t)
	system := filepath.Join(root, "repos", "system")
	for path, content := range map[string]string{
		filepath.Join(system, ".engflow", "worktrees", "exp", "v1", "variant.go"): "package v // TRLC-LINKS: REQ-SYS-001\n",
		filepath.Join(system, "notes.md"):                                         "REQ-SYS-001 is satisfied by the encoder.\n",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := linksServer(t, root, "system")
	files := strings.Join(s.filesForRequirement("REQ-SYS-001"), ",")
	if strings.Contains(files, ".engflow") || !strings.Contains(files, "notes.md") {
		t.Fatalf("files for REQ-SYS-001: %s", files)
	}
}
