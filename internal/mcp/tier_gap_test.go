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
	if msg := toolError(t, s, "tier.gap", map[string]any{}); !strings.Contains(msg, "target") {
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
