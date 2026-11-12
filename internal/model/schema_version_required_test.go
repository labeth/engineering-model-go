// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// schemaVersion is required in the design and workspace documents as well
// (ADR-EMG-061): the CUE schema declares it required, and because the schema
// value is the constant 2 CUE fills it in, so the loaders reject a missing
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
	if _, err := LoadDesign(write("design-with.yml", "schemaVersion: 2\ndesign: {}\n")); err != nil {
		t.Errorf("design with schemaVersion 2: %v", err)
	}
	if _, err := LoadWorkspace(write("workspace-without.yml", "replacements: []\n")); err == nil || !strings.Contains(err.Error(), "unsupported schemaVersion 0") || !strings.Contains(err.Error(), SchemaVersionRemedy) {
		t.Errorf("workspace without schemaVersion: got %v", err)
	}
	if _, err := LoadWorkspace(write("workspace-with.yml", "schemaVersion: 2\nreplacements: []\n")); err != nil {
		t.Errorf("workspace with schemaVersion 2: %v", err)
	}
}

// A schema version that is missing (0) or old is a *SchemaVersionError whose
// text names the remedy, and the document loaders wrap it.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125
func TestSchemaVersionErrorNamesTheRemedy(t *testing.T) {
	for _, version := range []int{0, 1, 3} {
		_, err := EffectiveSchemaVersion(version)
		var versionErr *SchemaVersionError
		if !errors.As(err, &versionErr) || versionErr.Version != version {
			t.Fatalf("version %d: got %v", version, err)
		}
		if !strings.Contains(err.Error(), "supported version is 2") || !strings.HasSuffix(err.Error(), "fix: "+SchemaVersionRemedy) || strings.Contains(versionErr.Problem(), "fix:") {
			t.Errorf("version %d: error %q, problem %q", version, err, versionErr.Problem())
		}
	}
	if _, err := EffectiveSchemaVersion(CurrentSchemaVersion); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(SchemaVersionRemedy, "engflow upgrade") || !strings.Contains(SchemaVersionRemedy, "schemaVersion: 2") {
		t.Fatalf("remedy = %q", SchemaVersionRemedy)
	}
}
