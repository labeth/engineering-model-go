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

// subject is a text a test makes assertions about, and the whole document or
// diagram it is part of, which the failure messages may show.
type subject struct {
	t     *testing.T
	text  string
	whole string
}

// about is a text that is its own whole.
func about(t *testing.T, text string) subject {
	return subject{t: t, text: text, whole: text}
}

// inside is a block of a whole document or diagram.
func inside(t *testing.T, block, whole string) subject {
	return subject{t: t, text: block, whole: whole}
}

// has fails the test with the message of the first check whose text is missing.
func (s subject) has(checks ...docCheck) {
	s.t.Helper()
	for _, c := range checks {
		if !strings.Contains(s.text, c.text) {
			s.t.Fatalf("%s", c.msg)
		}
	}
}

// lacks fails the test with the message of the first check whose text is present.
func (s subject) lacks(checks ...docCheck) {
	s.t.Helper()
	for _, c := range checks {
		if strings.Contains(s.text, c.text) {
			s.t.Fatalf("%s", c.msg)
		}
	}
}

// hasAll fails the test, with the format filled with the text, for each text
// that is missing.
func (s subject) hasAll(format string, wants ...string) {
	s.t.Helper()
	for _, want := range wants {
		if !strings.Contains(s.text, want) {
			s.t.Fatalf(format, want)
		}
	}
}

// hasNone fails the test, with the format filled with the text, for each text
// that is present.
func (s subject) hasNone(format string, unwanted ...string) {
	s.t.Helper()
	for _, text := range unwanted {
		if strings.Contains(s.text, text) {
			s.t.Fatalf(format, text)
		}
	}
}

// hasShowing fails the test, with the format filled with the text and the
// whole, for each text that is missing.
func (s subject) hasShowing(format string, wants ...string) {
	s.t.Helper()
	for _, want := range wants {
		if !strings.Contains(s.text, want) {
			s.t.Fatalf(format, want, s.whole)
		}
	}
}

// lacksShowing fails the test, with the format filled with the text and the
// whole, for each text that is present.
func (s subject) lacksShowing(format string, unwanted ...string) {
	s.t.Helper()
	for _, text := range unwanted {
		if strings.Contains(s.text, text) {
			s.t.Fatalf(format, text, s.whole)
		}
	}
}

// count fails the test, with the format filled with the count and the whole,
// unless the text occurs the number of times.
func (s subject) count(text string, want int, format string) {
	s.t.Helper()
	if got := strings.Count(s.text, text); got != want {
		s.t.Fatalf(format, got, s.whole)
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
	return Generate(Inputs{Bundle: schemaV2TestBundle(s.bundle), Requirements: schemaV2TestRequirements(s.requirements), Design: schemaV2TestDesign(s.design)}, Options{})
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
	res, err := Generate(Inputs{Bundle: schemaV2TestBundle(bundle), Requirements: schemaV2TestRequirements(model.RequirementsDocument{}), Design: schemaV2TestDesign(model.DesignDocument{})}, options)
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

// numberedLabels are the labels of the format for 1 up to count.
func numberedLabels(format string, count int) []string {
	labels := make([]string, 0, count)
	for i := 1; i <= count; i++ {
		labels = append(labels, fmt.Sprintf(format, i))
	}
	return labels
}
