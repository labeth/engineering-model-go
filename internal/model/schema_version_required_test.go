// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"errors"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"
)

// schemaVersion is required in the design and workspace documents as well
// (ADR-EMG-061): the CUE schema declares it required, and because the schema
// value is the constant 3 CUE fills it in, so the loaders reject a missing
// one on the decoded document, with the remedy in the error.
// Mutation: dropping the EffectiveSchemaVersion call of LoadDesign or
// LoadWorkspace accepts the document without schemaVersion.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125
func TestSchemaVersionIsRequiredInDesignAndWorkspaceDocuments(t *testing.T) {
	dir := t.TempDir()
	write := func(name, content string) string {
		path := filepath.Join(dir, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		return path
	}
	if _, err := LoadDesign(write("design-without.yml", "design: {}\n")); err == nil || !strings.Contains(err.Error(), "unsupported schemaVersion 0") || !strings.Contains(err.Error(), SchemaVersionRemedy) {
		t.Errorf("design without schemaVersion: got %v", err)
	}
	if _, err := LoadDesign(write("design-with.yml", "schemaVersion: 3\ndesign: {}\n")); err != nil {
		t.Errorf("design with schemaVersion 3: %v", err)
	}
	if _, err := LoadWorkspace(write("workspace-without.yml", "replacements: []\n")); err == nil || !strings.Contains(err.Error(), "unsupported schemaVersion 0") || !strings.Contains(err.Error(), SchemaVersionRemedy) {
		t.Errorf("workspace without schemaVersion: got %v", err)
	}
	if _, err := LoadWorkspace(write("workspace-with.yml", "schemaVersion: 3\nreplacements: []\n")); err != nil {
		t.Errorf("workspace with schemaVersion 3: %v", err)
	}
}

// A schema version that is missing (0) or old is a *SchemaVersionError whose
// text names the remedy, and the document loaders wrap it.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125
func TestSchemaVersionErrorNamesTheRemedy(t *testing.T) {
	for _, version := range []int{0, 1, 2, 4} {
		assertVersionErrorNamesTheRemedy(t, version)
	}
	if _, err := EffectiveSchemaVersion(CurrentSchemaVersion); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(SchemaVersionRemedy, "schemaVersion: 3") {
		t.Fatalf("remedy = %q", SchemaVersionRemedy)
	}
	assertRemedyNamesNoMissingCommand(t)
}

// assertVersionErrorNamesTheRemedy checks that a bad version is a
// *SchemaVersionError whose text names the remedy and whose problem does not.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125
func assertVersionErrorNamesTheRemedy(t *testing.T, version int) {
	t.Helper()
	_, err := EffectiveSchemaVersion(version)
	var versionErr *SchemaVersionError
	if !errors.As(err, &versionErr) || versionErr.Version != version {
		t.Fatalf("version %d: got %v", version, err)
	}
	if !strings.Contains(err.Error(), "supported version is "+strconv.Itoa(CurrentSchemaVersion)) || !strings.HasSuffix(err.Error(), "fix: "+SchemaVersionRemedy) || strings.Contains(versionErr.Problem(), "fix:") {
		t.Errorf("version %d: error %q, problem %q", version, err, versionErr.Problem())
	}
}

// assertRemedyNamesNoMissingCommand checks that the remedy names only what
// exists: engineering-flow has no upgrade command yet, so no text may tell
// the user to run one.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125
func assertRemedyNamesNoMissingCommand(t *testing.T) {
	t.Helper()
	if strings.Contains(SchemaVersionRemedy, "run engflow") || !strings.Contains(SchemaVersionRemedy, "planned") {
		t.Fatalf("remedy must not name a command that does not exist: %q", SchemaVersionRemedy)
	}
}

// Every document type the bundle loader reads rejects a missing, 0, 1 or 3
// schemaVersion with a *SchemaVersionError that carries the fix, before the
// CUE schema is consulted (which would fill in a missing version and word a
// wrong one as a unification failure).
// Mutation: moving the checkSchemaVersion call of decodeYAMLFile after
// validateCanonicalYAML, or dropping it, makes the absent, 1, 2 and 4 rows
// fail with a CUE error instead.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func TestEveryDocumentTypeRejectsABadSchemaVersionWithTheFix(t *testing.T) {
	for _, kind := range schemaDocumentKinds {
		for _, v := range badSchemaVersions {
			t.Run(kind.name+"/"+v.name, func(t *testing.T) { assertBadSchemaVersionRejected(t, kind, v) })
		}
	}
}

// schemaDocumentKind is a document type of the bundle and its fixture file.
type schemaDocumentKind struct{ name, file string }

var schemaDocumentKinds = []schemaDocumentKind{
	{"manifest", "engmod.yml"}, {"catalog", "model/catalog.yml"}, {"requirements", "model/requirements.yml"},
	{"architecture", "model/architecture.yml"}, {"behavior", "model/behavior.yml"}, {"assurance", "model/assurance.yml"},
	{"compliance", "model/compliance.yml"}, {"views", "model/views.yml"}, {"decisions", "model/decisions.yml"},
	{"aviation", "model/aviation.yml"}, {"reqLinks", "model/req-links.yml"},
}

// badSchemaVersion is a replacement of the "schemaVersion: 3" line and the
// version the loader must report.
type badSchemaVersion struct {
	name    string
	line    string // replacement of the "schemaVersion: 3" line
	version int
}

var badSchemaVersions = []badSchemaVersion{
	{"absent", "", 0}, {"0", "schemaVersion: 0\n", 0}, {"1", "schemaVersion: 1\n", 1}, {"2", "schemaVersion: 2\n", 2}, {"4", "schemaVersion: 4\n", 4},
}

// schemaFixture writes a valid v2 fixture; the aviation and reqLinks
// documents, which the manifest does not name by default, are added.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func schemaFixture(t *testing.T, kind schemaDocumentKind) string {
	t.Helper()
	const anchor = "  decisions: model/decisions.yml\n"
	manifest := validManifest
	extra := ""
	switch kind.name {
	case "aviation":
		manifest = strings.Replace(manifest, anchor, anchor+"  aviation: model/aviation.yml\n", 1)
		extra = "schemaVersion: 3\naviation: {}\n"
	case "reqLinks":
		manifest = strings.Replace(manifest, anchor, anchor+"  reqLinks: model/req-links.yml\n", 1)
		extra = "schemaVersion: 3\nreqLinks: []\n"
	}
	dir := writeV2Fixture(t, manifest)
	if extra != "" {
		writeTestFile(t, filepath.Join(dir, kind.file), extra)
	}
	return dir
}

// replaceSchemaVersionLine replaces the schemaVersion line of a fixture file.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func replaceSchemaVersionLine(t *testing.T, path, line string) {
	t.Helper()
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	mutated := strings.Replace(string(content), "schemaVersion: 3\n", line, 1)
	if mutated == string(content) && line != "schemaVersion: 3\n" {
		t.Fatalf("fixture %s has no schemaVersion line", path)
	}
	writeTestFile(t, path, mutated)
}

// assertBadSchemaVersionRejected checks that the loader rejects the document
// with a *SchemaVersionError that carries the fix and names the file.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
func assertBadSchemaVersionRejected(t *testing.T, kind schemaDocumentKind, v badSchemaVersion) {
	t.Helper()
	dir := schemaFixture(t, kind)
	replaceSchemaVersionLine(t, filepath.Join(dir, kind.file), v.line)
	_, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	var versionErr *SchemaVersionError
	if !errors.As(err, &versionErr) {
		t.Fatalf("got %v, want a *SchemaVersionError", err)
	}
	if versionErr.Version != v.version {
		t.Errorf("version %d, want %d", versionErr.Version, v.version)
	}
	if !strings.Contains(err.Error(), "fix: "+SchemaVersionRemedy) || strings.Contains(err.Error(), "CUE") {
		t.Errorf("error lacks the fix or is a CUE error: %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Base(kind.file)) {
		t.Errorf("error does not name the file %s: %v", kind.file, err)
	}
}

// The previous schema version 2 is no longer read: the loader rejects it with
// a *SchemaVersionError whose message and remedy name the current version 3,
// so a model written for an earlier release fails on its version and not on
// whichever field broke first (ADR-EMG-077).
// Mutation: leaving CurrentSchemaVersion at 2, or a remedy text that names
// another number, fails this test.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-MODEL-LOADER
func TestSchemaVersionTwoIsRejectedWithTheRemedyNamingTheCurrentVersion(t *testing.T) {
	if CurrentSchemaVersion != 3 {
		t.Fatalf("CurrentSchemaVersion = %d, want 3", CurrentSchemaVersion)
	}
	assertBadSchemaVersionRejected(t, schemaDocumentKinds[0], badSchemaVersion{"2", "schemaVersion: 2\n", 2})
	_, err := EffectiveSchemaVersion(2)
	if err == nil || !strings.Contains(err.Error(), "unsupported schemaVersion 2; supported version is 3") || !strings.Contains(err.Error(), "set schemaVersion: 3 at the top") {
		t.Fatalf("got %v", err)
	}
}

// The CUE constant #SchemaVersion equals CurrentSchemaVersion, so the schema
// and the loader cannot disagree about the version a release reads.
// Mutation: raising only one of the two fails this test.
// TRLC-LINKS: REQ-EMG-186
// ENGMODEL-LINKS: FU-MODEL-LOADER
func TestCUESchemaVersionEqualsTheGoConstant(t *testing.T) {
	data, err := os.ReadFile(filepath.Join("schema", "common.cue"))
	if err != nil {
		t.Fatal(err)
	}
	want := "#SchemaVersion: " + strconv.Itoa(CurrentSchemaVersion) + "\n"
	if !strings.Contains(string(data), want) {
		t.Fatalf("schema/common.cue lacks %q", want)
	}
}
