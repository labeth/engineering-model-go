// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-010
func TestBuildCodeReferences_UsesOwnerAndDescriptionFields(t *testing.T) {
	refs := buildCodeReferences([]codescan.CodeItem{{
		Element:     "src/webhook_ingress.go",
		Kind:        "source_file",
		Owner:       "FU-GITHUB-WEBHOOK-INGRESS",
		Description: "validates webhook signatures and routes pull request events",
		Source:      "src/webhook_ingress.go",
	}})

	if len(refs) != 1 {
		t.Fatalf("expected 1 reference, got %d", len(refs))
	}
	if refs[0].Owner != "FU-GITHUB-WEBHOOK-INGRESS" {
		t.Fatalf("unexpected owner: %q", refs[0].Owner)
	}
	wantDesc := "validates webhook signatures and routes pull request events"
	if refs[0].Description != wantDesc {
		t.Fatalf("unexpected description: got %q want %q", refs[0].Description, wantDesc)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestInferCodeItemsDistinctFiles(t *testing.T) {
	root := t.TempDir()
	writeBuildinfoFixture(t, root, "APP-001")
	writeBuildinfoFixture(t, root, "OTA-001")
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}
	items, diags := codescan.InferCodeItems(bundle, "")
	if len(diags) != 0 {
		t.Fatalf("diagnostics: %+v", diags)
	}
	if refs := buildCodeReferences(items); len(refs) != 6 {
		t.Fatalf("publication dropped distinct files: %+v", refs)
	}
	counts, links := countItemKindsAndLinks(items)
	if len(items) != 6 || !reflect.DeepEqual(counts, map[string]int{"symbol": 2, "source_file": 2, "library_stdlib": 2}) {
		t.Fatalf("counts=%v items=%+v", counts, items)
	}
	if links["REQ-APP-001"] != 1 || links["REQ-OTA-001"] != 1 {
		t.Fatalf("links=%v items=%+v", links, items)
	}
}

// writeBuildinfoFixture writes one Go file that implements REQ-<name> under
// root/<name>.
func writeBuildinfoFixture(t *testing.T, root, name string) {
	t.Helper()
	dir := filepath.Join(root, name)
	if err := os.MkdirAll(dir, 0700); err != nil {
		t.Fatal(err)
	}
	src := "// ENGMODEL-OWNER-UNIT: FU-BUILD\npackage buildinfo\nimport \"fmt\"\n// TRLC-LINKS: REQ-" + name + "\nfunc String() string { return fmt.Sprint(1) }\n"
	if err := os.WriteFile(filepath.Join(dir, "buildinfo.go"), []byte(src), 0600); err != nil {
		t.Fatal(err)
	}
}

// countItemKindsAndLinks counts the inferred items per kind and per
// implemented requirement.
func countItemKindsAndLinks(items []codescan.CodeItem) (counts, links map[string]int) {
	counts, links = map[string]int{}, map[string]int{}
	for _, item := range items {
		counts[item.Kind]++
		for _, req := range item.Implements {
			links[req]++
		}
	}
	return counts, links
}
