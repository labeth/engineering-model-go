// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package main

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"strings"
	"testing"
)

// The CLI renders the selected requirements of a real model; every REQ line
// starts with the model text under the engineering-flow consistency check's
// normalization (lower case, collapsed whitespace, no backticks or trailing
// period), and openspec validate --strict accepts the delta when the
// openspec CLI is available (OPENSPEC_BIN or PATH).
// Mutation: writing the title instead of the model text on the REQ line, or
// dropping the scenario, fails this test.
// TRLC-LINKS: REQ-EMG-081
// ENGMODEL-LINKS: IF-CLI-ENGSPEC
func TestRunRendersDeltaThatPassesConsistencyAndOpenSpec(t *testing.T) {
	manifest := filepath.Join("..", "..", "examples", "dal-c-flight-control-sample", "engmod.yml")
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--model", manifest, "--prefix", "REQ-YD-", "--capability", "yaw-damping"}, &stdout, &stderr); code != 0 {
		t.Fatalf("run: %d\n%s", code, stderr.String())
	}
	out := stdout.String()
	if !strings.Contains(out, "## ADDED Requirements\n") || strings.Count(out, "#### Scenario:") < 3 {
		t.Fatalf("delta:\n%s", out)
	}
	model, err := os.ReadFile(filepath.Join("..", "..", "examples", "dal-c-flight-control-sample", "model", "requirements.yml"))
	if err != nil {
		t.Fatal(err)
	}
	texts := map[string]string{}
	for _, m := range regexp.MustCompile(`(?m)^  - id: (REQ-\S+)\n(?:    .*\n)*?    text: (.+)$`).FindAllStringSubmatch(string(model), -1) {
		texts[m[1]] = m[2]
	}
	lines := regexp.MustCompile(`(?m)^(REQ-[A-Z0-9-]+): (.+)$`).FindAllStringSubmatch(out, -1)
	if len(lines) < 3 {
		t.Fatalf("no REQ lines:\n%s", out)
	}
	for _, line := range lines {
		want, ok := texts[line[1]]
		if !ok || !strings.HasPrefix(normalize(line[2]), normalize(want)) {
			t.Fatalf("%s: spec %q does not start with model %q", line[1], line[2], want)
		}
	}
	openspecValidate(t, out)
}

// TRLC-LINKS: REQ-EMG-081
func TestRunUsageAndUnknownID(t *testing.T) {
	var stdout, stderr bytes.Buffer
	if code := run([]string{"--model", "engmod.yml"}, &stdout, &stderr); code != 2 || !strings.Contains(stderr.String(), "usage: engspec") {
		t.Fatalf("no selection: %d %s", code, stderr.String())
	}
	stderr.Reset()
	if code := run([]string{"--model", filepath.Join("..", "..", "engmod.yml"), "--ids", "REQ-EMG-001,REQ-EMG-999"}, &stdout, &stderr); code != 1 || !strings.Contains(stderr.String(), "REQ-EMG-999") {
		t.Fatalf("unknown id: %d %s", code, stderr.String())
	}
}

// normalize is the engineering-flow consistency check's comparison form.
func normalize(text string) string {
	text = strings.ReplaceAll(text, "`", "")
	text = strings.Join(strings.Fields(text), " ")
	return strings.ToLower(strings.TrimRight(text, ". "))
}

// openspecValidate places the delta in a scratch change and runs
// openspec validate --strict, when an openspec binary is available.
func openspecValidate(t *testing.T, delta string) {
	t.Helper()
	bin := os.Getenv("OPENSPEC_BIN")
	if bin == "" {
		bin, _ = exec.LookPath("openspec")
	}
	if bin == "" {
		t.Log("openspec not available (set OPENSPEC_BIN); skipped openspec validate --strict")
		return
	}
	project := t.TempDir()
	change := filepath.Join(project, "openspec", "changes", "add-yaw")
	for path, content := range map[string]string{
		filepath.Join(project, "openspec", "config.yaml"):              "schema: spec-driven\n",
		filepath.Join(change, "proposal.md"):                           "## Why\n\nThe yaw damping software needs a filtered input to stay stable in turbulence.\n\n## What Changes\n\n- Add the filtering requirements.\n",
		filepath.Join(change, "specs", "yaw-damping", "spec.md"):       delta,
		filepath.Join(change, "tasks.md"):                              "## 1. Implement\n\n- [ ] 1.1 Filter the input\n",
		filepath.Join(project, "openspec", "specs", ".keep"):           "",
		filepath.Join(project, "openspec", "changes", "archive", ".k"): "",
	} {
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	cmd := exec.Command(bin, "validate", "add-yaw", "--strict", "--no-interactive")
	cmd.Dir = project
	cmd.Env = append(os.Environ(), "OPENSPEC_TELEMETRY=0", "DO_NOT_TRACK=1")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("openspec validate --strict: %v\n%s\n--- delta\n%s", err, out, delta)
	}
}
