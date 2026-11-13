// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"sort"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// loadedShapeBundle is a bundle as LoadBundle returns it: every document path
// set, aviation and reqLinks present, every schemaVersion current.
func loadedShapeBundle() model.Bundle {
	b := completeBundleAt(model.TierMVP)
	const v = model.CurrentSchemaVersion
	b.ManifestPath, b.BehaviorPath, b.AssurancePath, b.CompliancePath, b.ViewsPath, b.ReqLinksPath = "engmod.yml", "b", "a", "c", "v", "r"
	b.Manifest.SchemaVersion, b.Behavior.SchemaVersion, b.Assurance.SchemaVersion = v, v, v
	b.Compliance.SchemaVersion, b.Views.SchemaVersion, b.ReqLinks.SchemaVersion = v, v, v
	b.Aviation = &model.AviationDocument{SchemaVersion: v}
	return b
}

// Bundle reports model.unsupported_schema_version for every document it
// holds: a bundle in the shape LoadBundle returns names each document whose
// version is not current, by its own path, and none when all are current.
// Mutation: dropping a document from schemaVersions leaves its row unreported.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func TestBundleChecksTheSchemaVersionOfEveryDocument(t *testing.T) {
	if got := codesOf(Bundle(loadedShapeBundle()), "model.unsupported_schema_version"); len(got) != 0 {
		t.Fatalf("current versions reported: %+v", got)
	}
	for _, path := range sortedBreakerPaths() {
		for _, version := range []int{0, 1, 3} {
			assertBrokenVersionReported(t, path, version)
		}
	}
}

// versionBreakers sets the schemaVersion of one document of a bundle, by the
// path Bundle reports for it.
var versionBreakers = map[string]func(*model.Bundle, int){
	"schemaVersion":              func(b *model.Bundle, v int) { b.Architecture.SchemaVersion = v },
	"manifest.schemaVersion":     func(b *model.Bundle, v int) { b.Manifest.SchemaVersion = v },
	"catalog.schemaVersion":      func(b *model.Bundle, v int) { b.Catalog.SchemaVersion = v },
	"requirements.schemaVersion": func(b *model.Bundle, v int) { b.Requirements.SchemaVersion = v },
	"behavior.schemaVersion":     func(b *model.Bundle, v int) { b.Behavior.SchemaVersion = v },
	"assurance.schemaVersion":    func(b *model.Bundle, v int) { b.Assurance.SchemaVersion = v },
	"compliance.schemaVersion":   func(b *model.Bundle, v int) { b.Compliance.SchemaVersion = v },
	"views.schemaVersion":        func(b *model.Bundle, v int) { b.Views.SchemaVersion = v },
	"design.schemaVersion":       func(b *model.Bundle, v int) { b.Design.SchemaVersion = v },
	"decisions.schemaVersion":    func(b *model.Bundle, v int) { b.Decisions.SchemaVersion = v },
	"aviation.schemaVersion":     func(b *model.Bundle, v int) { b.Aviation.SchemaVersion = v },
	"reqLinks.schemaVersion":     func(b *model.Bundle, v int) { b.ReqLinks.SchemaVersion = v },
}

// sortedBreakerPaths returns the document paths of versionBreakers in order.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func sortedBreakerPaths() []string {
	var paths []string
	for path := range versionBreakers {
		paths = append(paths, path)
	}
	sort.Strings(paths)
	return paths
}

// assertBrokenVersionReported checks that a bundle whose document at path has
// the version gets one model.unsupported_schema_version error at that path,
// with the remedy.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func assertBrokenVersionReported(t *testing.T, path string, version int) {
	t.Helper()
	b := loadedShapeBundle()
	versionBreakers[path](&b, version)
	got := codesOf(Bundle(b), "model.unsupported_schema_version")
	if len(got) != 1 {
		t.Errorf("%s = %d: got %+v", path, version, got)
		return
	}
	if got[0].Path != path || got[0].Severity != SeverityError || !strings.Contains(got[0].Message, "supported version is 2") {
		t.Errorf("%s = %d: got %+v", path, version, got)
	}
	if got[0].Remedy != "set schemaVersion: 2 at the top of the file (edit the file; an upgrade command is planned in engineering-flow and does not exist yet)" {
		t.Errorf("%s: remedy %q", path, got[0].Remedy)
	}
}
