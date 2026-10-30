// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-003
func TestCodeEvidenceLabelsPreserveFileExtensionsAndSymbolLines(t *testing.T) {
	file := CodeItem{Kind: "source_file", Element: "src/checkout_api.go", Source: "src/checkout_api.go"}
	if got := codeItemEvidenceLabel(file); got != "checkout_api.go" {
		t.Fatalf("expected file label with extension, got %q", got)
	}

	symbol := CodeItem{Kind: "symbol", Element: "CODE-AUTHORIZE", Source: "src/payment_engine.rs:25"}
	if got := ItemDisplayName(symbol); got != "payment_engine.rs:25" {
		t.Fatalf("expected function-level symbol display, got %q", got)
	}
	if got := codeItemEvidenceLabel(symbol); got != "payment_engine.rs:25" {
		t.Fatalf("expected symbol evidence label with file extension and line, got %q", got)
	}
	if got := SanitizeNode(ItemDisplayName(symbol)); strings.ContainsAny(got, "()[]") {
		t.Fatalf("expected mermaid-safe symbol node id, got %q", got)
	}
	if got := SanitizeNode("parse(input) [v2]"); got != "PARSE_INPUT___V2_" {
		t.Fatalf("expected brackets folded into a mermaid-safe node id, got %q", got)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestGroupedCodeElementEvidenceLabels_OneLabelPerFileWithCommaLines(t *testing.T) {
	got := GroupedEvidenceLabels([]string{
		"src/payment_engine.rs:25",
		"src/payment_engine.rs:11",
		"src/payment_engine.rs:25",
		"src/checkout_api.go",
	}, nil)

	want := []string{"checkout_api.go", "payment_engine.rs:11,25"}
	if len(got) != len(want) {
		t.Fatalf("unexpected grouped labels: got %+v want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("unexpected grouped label at %d: got %q want %q", i, got[i], want[i])
		}
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestCodeEvidenceLabelsKeepDistinctSourcePaths(t *testing.T) {
	inputs := []string{"app/server.go:412", "ota/server.go:16", "ota/./server.go:20", "ota/Server.go:7"}
	lookup := EvidenceLabelLookup(inputs)
	for _, tc := range []struct{ source, want string }{
		{"app/server.go:412", "app/server.go:412"},
		{"ota/server.go:16", "ota/server.go:16,20"},
		{"ota/Server.go:7", "Server.go:7"},
	} {
		got := GroupedEvidenceLabels([]string{tc.source}, lookup)
		if len(got) != 1 || got[0] != tc.want {
			t.Fatalf("source %q: got %v, want %q", tc.source, got, tc.want)
		}
	}
	got := GroupedFileLabels(inputs, lookup)
	if strings.Join(got, ",") != "Server.go,app/server.go,ota/server.go" {
		t.Fatalf("distinct source files collapsed: %v", got)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestAbsoluteCodeItemEvidenceKeepsSourceIdentity(t *testing.T) {
	items := []CodeItem{
		{Kind: "symbol", Source: "/workspace/app/server.go:412"},
		{Kind: "symbol", Source: "/workspace/ota/server.go:16"},
	}
	elements := []string{ItemEvidenceElement(items[0]), ItemEvidenceElement(items[1])}
	lookup := EvidenceLabelLookup(elements)
	for i, item := range items {
		got := GroupedEvidenceLabels(elements[i:i+1], lookup)
		if len(got) != 1 || got[0] != strings.TrimPrefix(item.Source, "/workspace/") {
			t.Fatalf("source %q: got %v", item.Source, got)
		}
	}
}

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-010
func TestInferredCodeEvidenceIsWorktreeIndependent(t *testing.T) {
	var publications []string
	for _, name := range []string{"checkout-a", "checkout-b"} {
		root := filepath.Join(t.TempDir(), name)
		source := filepath.Join(root, "internal", "demo.go")
		if err := os.MkdirAll(filepath.Dir(source), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(source, []byte("// ENGMODEL-OWNER-UNIT: FU-DEMO\npackage demo\n\n// TRLC-LINKS: REQ-EMG-003\nfunc Run() {}\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		items, diagnostics := InferCodeItems(model.Bundle{ManifestPath: filepath.Join(root, "engmod.yml")}, root)
		if len(diagnostics) != 0 {
			t.Fatalf("scan %s: %v", name, diagnostics)
		}
		elements := make([]string, 0, len(items))
		for _, item := range items {
			if item.Kind == "source_file" || item.Kind == "symbol" {
				elements = append(elements, ItemEvidenceElement(item))
			}
		}
		publications = append(publications, strings.Join(GroupedFileLabels(elements, nil), ","))
		if strings.Contains(publications[len(publications)-1], name) {
			t.Fatalf("checkout name leaked into code evidence: %s", publications[len(publications)-1])
		}
	}
	if publications[0] != publications[1] {
		t.Fatalf("code evidence depends on checkout: %q != %q", publications[0], publications[1])
	}
}
