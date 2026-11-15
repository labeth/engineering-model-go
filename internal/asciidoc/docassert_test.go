// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// docCheck is a text and the failure message of a document that lacks it, or
// has it, according to the helper it is passed to.
type docCheck struct {
	text string
	msg  string
}

// requireContains fails the test with the message of the first check whose
// text the document lacks.
func requireContains(t *testing.T, doc string, checks ...docCheck) {
	t.Helper()
	for _, c := range checks {
		if !strings.Contains(doc, c.text) {
			t.Fatalf("%s", c.msg)
		}
	}
}

// requireNotContains fails the test with the message of the first check whose
// text the document has.
func requireNotContains(t *testing.T, doc string, checks ...docCheck) {
	t.Helper()
	for _, c := range checks {
		if strings.Contains(doc, c.text) {
			t.Fatalf("%s", c.msg)
		}
	}
}

// requireAllContain fails the test, with the format filled with the text, for
// each text the document lacks.
func requireAllContain(t *testing.T, doc, format string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(doc, want) {
			t.Fatalf(format, want)
		}
	}
}

// requireNoneContain fails the test, with the format filled with the text, for
// each text the document has.
func requireNoneContain(t *testing.T, doc, format string, unwanted ...string) {
	t.Helper()
	for _, text := range unwanted {
		if strings.Contains(doc, text) {
			t.Fatalf(format, text)
		}
	}
}

// requireReferenceEntries fails the test for each ID without a reference index entry.
func requireReferenceEntries(t *testing.T, doc string, ids ...string) {
	t.Helper()
	for _, id := range ids {
		if referenceBlockByID(doc, id) == "" {
			t.Fatalf("missing reference index entry for %s", id)
		}
	}
}

// requireNoBlankPaddedLines fails the test for a line that holds only whitespace.
func requireNoBlankPaddedLines(t *testing.T, doc string) {
	t.Helper()
	for lineNumber, line := range strings.Split(doc, "\n") {
		if line != "" && strings.TrimSpace(line) == "" {
			t.Fatalf("line %d contains only whitespace", lineNumber+1)
		}
	}
}

// sampleModel is a model loaded from an example directory.
type sampleModel struct {
	bundle       model.Bundle
	requirements model.RequirementsDocument
	design       model.DesignDocument
}

// loadSample loads the manifest, requirements and views of the example directory.
func loadSample(t *testing.T, dir string) sampleModel {
	t.Helper()
	bundle, err := model.LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	requirements, err := model.LoadRequirements(filepath.Join(dir, "model", "requirements.yml"))
	if err != nil {
		t.Fatalf("load requirements failed: %v", err)
	}
	design, err := model.LoadDesign(filepath.Join(dir, "model", "views.yml"))
	if err != nil {
		t.Fatalf("load design failed: %v", err)
	}
	return sampleModel{bundle: bundle, requirements: requirements, design: design}
}

// generate renders the sample as schema v2 documents.
func (s sampleModel) generate() (Result, error) {
	return Generate(schemaV2TestBundle(s.bundle), schemaV2TestRequirements(s.requirements), schemaV2TestDesign(s.design), Options{})
}

// mustGenerate renders the sample and fails the test on an error.
func (s sampleModel) mustGenerate(t *testing.T) Result {
	t.Helper()
	res, err := s.generate()
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	return res
}

// mustGenerateBundle renders a bundle without requirements or views as schema
// v2 documents and fails the test on an error.
func mustGenerateBundle(t *testing.T, bundle model.Bundle, options Options) Result {
	t.Helper()
	res, err := Generate(schemaV2TestBundle(bundle), schemaV2TestRequirements(model.RequirementsDocument{}), schemaV2TestDesign(model.DesignDocument{}), options)
	if err != nil {
		t.Fatalf("generate asciidoc failed: %v", err)
	}
	return res
}

// writeTree writes the files, by path relative to root, creating directories.
func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		path := filepath.Join(root, filepath.FromSlash(rel))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatalf("create directory of %s: %v", rel, err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture %s: %v", rel, err)
		}
	}
}

// hasDiagnostic reports whether a diagnostic has the code.
func hasDiagnostic(diags []validate.Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// requireIn fails the test, with the format filled with the text and the whole
// diagram, for each text the block lacks.
func requireIn(t *testing.T, block, whole, format string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		if !strings.Contains(block, want) {
			t.Fatalf(format, want, whole)
		}
	}
}

// requireNotIn fails the test, with the format filled with the text and the
// whole diagram, for each text the block has.
func requireNotIn(t *testing.T, block, whole, format string, unwanted ...string) {
	t.Helper()
	for _, text := range unwanted {
		if strings.Contains(block, text) {
			t.Fatalf(format, text, whole)
		}
	}
}

// requireCount fails the test, with the format filled with the count and the
// document, unless the text occurs the number of times.
func requireCount(t *testing.T, doc, text string, want int, format string) {
	t.Helper()
	if got := strings.Count(doc, text); got != want {
		t.Fatalf(format, got, doc)
	}
}

// numberedLabels are the labels of the format for 1 up to count.
func numberedLabels(format string, count int) []string {
	labels := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		labels = append(labels, fmt.Sprintf(format, i))
	}
	return labels
}

// requireDiagramHas fails the test, with the format filled with the text and
// the diagram, for each text the diagram lacks.
func requireDiagramHas(t *testing.T, diagram, format string, wants ...string) {
	t.Helper()
	requireIn(t, diagram, diagram, format, wants...)
}

// requireDiagramLacks fails the test, with the format filled with the text and
// the diagram, for each text the diagram has.
func requireDiagramLacks(t *testing.T, diagram, format string, unwanted ...string) {
	t.Helper()
	requireNotIn(t, diagram, diagram, format, unwanted...)
}
