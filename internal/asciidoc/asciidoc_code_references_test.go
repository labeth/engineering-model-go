// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"os"
	"path/filepath"
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
func TestInferCodeItemsDistinctFilesAndOverlappingRoots(t *testing.T) {
	root := t.TempDir()
	var files []string
	for _, name := range []string{"APP-001", "OTA-001"} {
		dir := filepath.Join(root, name)
		if err := os.MkdirAll(dir, 0700); err != nil {
			t.Fatal(err)
		}
		file := filepath.Join(dir, "buildinfo.go")
		src := "// ENGMODEL-OWNER-UNIT: FU-BUILD\npackage buildinfo\nimport \"fmt\"\n// TRLC-LINKS: REQ-" + name + "\nfunc String() string { return fmt.Sprint(1) }\n"
		if err := os.WriteFile(file, []byte(src), 0600); err != nil {
			t.Fatal(err)
		}
		files = append(files, file)
	}
	for _, overlap := range []bool{false, true} {
		sources := append([]string{}, files...)
		if overlap {
			sources = append(sources, root)
		}
		bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Architecture: model.ArchitectureDocument{InferenceHints: model.InferenceHints{CodeSources: sources}}}
		items, diags := codescan.InferCodeItems(bundle, "")
		if len(diags) != 0 {
			t.Fatalf("diagnostics: %+v", diags)
		}
		if refs := buildCodeReferences(items); len(refs) != 6 {
			t.Fatalf("publication dropped distinct files: %+v", refs)
		}
		counts := map[string]int{}
		links := map[string]int{}
		for _, item := range items {
			counts[item.Kind]++
			for _, req := range item.Implements {
				links[req]++
			}
		}
		if counts["symbol"] != 2 || counts["source_file"] != 2 || counts["library_stdlib"] != 2 || len(items) != 6 || links["REQ-APP-001"] != 1 || links["REQ-OTA-001"] != 1 {
			t.Fatalf("overlap=%v: counts=%v links=%v items=%+v", overlap, counts, links, items)
		}
	}
}
