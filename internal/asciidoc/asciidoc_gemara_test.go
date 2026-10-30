// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/export/gemara"
	"github.com/labeth/engineering-model-go/internal/model"
)

// The Gemara GRC chapter of the architecture publication counts the entries of
// the catalogs the Gemara exporter renders for the same bundle.
// TRLC-LINKS: REQ-EMG-015
func TestRenderGemaraAsciiDocChapterCountsExporterCatalogs(t *testing.T) {
	bundle, err := model.LoadBundle(filepath.Join("examples", "payments-engineering-sample", "engmod.yml"))
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	res, err := gemara.Generate(bundle, gemara.Options{})
	if err != nil {
		t.Fatalf("generate gemara: %v", err)
	}
	if len(res.ControlCatalog.Controls) == 0 {
		t.Fatal("sample model renders no Gemara controls")
	}
	chapter := renderGemaraAsciiDocChapter(bundle)
	if !strings.Contains(chapter, "== Gemara GRC Model") {
		t.Fatalf("missing Gemara chapter heading:\n%s", chapter)
	}
	want := fmt.Sprintf("|L2\n|Control Catalog\n|%d\n", len(res.ControlCatalog.Controls))
	if !strings.Contains(chapter, want) {
		t.Fatalf("chapter does not count the exporter's %d controls:\n%s", len(res.ControlCatalog.Controls), chapter)
	}
}
