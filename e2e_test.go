// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engmodel

import (
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-001
func TestGenerateFromFile_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	views := []string{"VIEW-ARCHITECTURE-INTENT", "VIEW-COMMUNICATION", "VIEW-DEPLOYMENT", "VIEW-SECURITY", "VIEW-TRACEABILITY"}

	for _, v := range views {
		v := v
		t.Run(v, func(t *testing.T) {
			res, err := GenerateFromFile(modelPath, v)
			if err != nil {
				t.Fatalf("generate failed: %v", err)
			}
			if validate.HasErrors(res.Diagnostics) {
				t.Fatalf("unexpected diagnostics: %+v", res.Diagnostics)
			}
			if !strings.Contains(res.Mermaid, "flowchart LR") {
				t.Fatalf("mermaid output missing flowchart header")
			}
			if !strings.Contains(res.Mermaid, "view: "+v) {
				t.Fatalf("mermaid output missing view marker for %s", v)
			}
		})
	}
}

func TestCLI_EndToEnd(t *testing.T) {
	modelPath := filepath.Join("examples", "payments-engineering-sample", "architecture.yml")
	cmd := exec.Command("go", "run", "./cmd/engview", "--model", modelPath, "--view", "VIEW-ARCHITECTURE-INTENT")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("cli failed: %v\noutput:\n%s", err, string(out))
	}
	text := string(out)
	if !strings.Contains(text, "flowchart LR") {
		t.Fatalf("cli output missing mermaid flowchart header")
	}
	if !strings.Contains(text, "VIEW-ARCHITECTURE-INTENT") {
		t.Fatalf("cli output missing expected view id marker")
	}
}
