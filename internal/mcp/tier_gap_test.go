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
	for _, repo := range []string{"system", "child-a"} {
		manifest := filepath.Join(root, "repos", repo, "engmod.yml")
		data, err := os.ReadFile(manifest)
		if err != nil {
			t.Fatal(err)
		}
		tier := map[string]string{"system": "mvp", "child-a": "prototype"}[repo]
		if err := os.WriteFile(manifest, append(data, []byte("lifecycle:\n  tier: "+tier+"\n")...), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := linksServer(t, root, "system")
	gap := callToolPayload(t, s, "tier.gap", map[string]any{"target": "product"})
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
	if msg := toolError(t, s, "tier.gap", map[string]any{"target": "beta"}); !strings.Contains(msg, "beta") {
		t.Fatalf("unknown target: %q", msg)
	}
	if msg := toolError(t, s, "tier.gap", map[string]any{}); !strings.Contains(msg, "needs target") {
		t.Fatalf("missing target: %q", msg)
	}
	if msg := toolError(t, s, "tier.gap", map[string]any{"target": "product", "x": "1"}); msg == "" {
		t.Fatal("an unknown argument was accepted")
	}
	schema := inputSchemaForTool("tier.gap")
	props := schema["properties"].(map[string]any)
	if props["target"] == nil || props["repo"] == nil || schema["additionalProperties"] != false {
		t.Fatalf("schema %+v", schema)
	}

	matrix := callToolPayload(t, s, "trace.matrix", map[string]any{"summaryOnly": true})
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
