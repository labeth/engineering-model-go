// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package testutil

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// withCompleteSwitches adds the lifecycle exactly when the manifest block has
// no top-level lifecycle key: a requirementLinks key or text that merely
// contains "lifecycle:" does not skip it.
// TRLC-LINKS: REQ-EMG-162, REQ-EMG-163, REQ-EMG-164
func TestWithCompleteSwitchesChecksLifecycleKey(t *testing.T) {
	cases := []struct {
		name, extra     string
		wantLifecycle   bool
		wantGuidedAdded bool
	}{
		{"empty", "", true, true},
		{"requirementLinks only", "requirementLinks: strict\n", true, false},
		{"lifecycle in a comment", "# lifecycle: none\n", true, true},
		{"own lifecycle", "lifecycle:\n  tier: mvp\n", false, false},
	}
	for _, c := range cases {
		got, added := withCompleteSwitches(c.extra)
		if added != c.wantLifecycle {
			t.Errorf("%s: lifecycle added = %v, want %v", c.name, added, c.wantLifecycle)
		}
		if has := strings.Contains(got, "tierDecision: DEC-TIER-POC"); has != c.wantLifecycle {
			t.Errorf("%s: output has the poc lifecycle = %v, want %v:\n%s", c.name, has, c.wantLifecycle, got)
		}
		if has := strings.Contains(got, "requirementLinks: guided"); has != c.wantGuidedAdded {
			t.Errorf("%s: output has requirementLinks guided = %v, want %v:\n%s", c.name, has, c.wantGuidedAdded, got)
		}
	}
}

// A manifest with its own lifecycle gets no stray DEC-TIER-POC decision.
// TRLC-LINKS: REQ-EMG-162, REQ-EMG-163, REQ-EMG-164
func TestWriteV2ModuleFilesWritesTierDecisionOnlyForItsLifecycle(t *testing.T) {
	own := t.TempDir()
	WriteV2ModuleFiles(t, own, "example.com/own@v0", "v0.1.0", "OWN", "requirementLinks: strict\nlifecycle:\n  tier: mvp\n  bundles: [foundation]\n  tierDecision: DEC-TIER-MVP\n", "  functionalGroups: []\n")
	data, err := os.ReadFile(filepath.Join(own, "model", "decisions.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(data), "DEC-TIER-POC") {
		t.Errorf("a manifest with its own lifecycle got a stray DEC-TIER-POC:\n%s", data)
	}
	added := t.TempDir()
	WriteV2ModuleFiles(t, added, "example.com/added@v0", "v0.1.0", "ADDED", "", "  functionalGroups: []\n")
	data, err = os.ReadFile(filepath.Join(added, "model", "decisions.yml"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), "DEC-TIER-POC") {
		t.Errorf("the added lifecycle has no DEC-TIER-POC decision:\n%s", data)
	}
}
