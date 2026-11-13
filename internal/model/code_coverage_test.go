// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const codeCoverageBlock = `codeCoverage:
  exempt:
    - id: FU-A
      reason: implemented by an external system
      decision: ADR-A
`

// A manifest with a codeCoverage block loads into the typed CodeCoverage;
// without the block it is nil. The block sits before lifecycle and health in
// the manifest field list.
// Mutation: dropping the CodeCoverage field, or its yaml tag, loses the list.
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-MODEL-LOADER
func TestManifestCodeCoverageLoads(t *testing.T) {
	dir := writeV2Fixture(t, validManifest+codeCoverageBlock)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := &CodeCoverage{Exempt: []CodeExemption{{ID: "FU-A", Reason: "implemented by an external system", Decision: "ADR-A"}}}
	if !reflect.DeepEqual(bundle.Manifest.CodeCoverage, want) {
		t.Fatalf("codeCoverage = %+v, want %+v", bundle.Manifest.CodeCoverage, want)
	}
	plain, err := LoadBundle(filepath.Join(writeV2Fixture(t, validManifest), "engmod.yml"))
	if err != nil || plain.Manifest.CodeCoverage != nil {
		t.Fatalf("a manifest without codeCoverage must leave it nil: %v %+v", err, plain.Manifest.CodeCoverage)
	}
}

// An unknown key, a missing field and a reason shorter than 20 characters in
// a codeCoverage block fail loading.
// Mutation: opening #CodeExemption, dropping the reason pattern or making a
// field optional in manifest.cue loads a block that must be rejected.
// TRLC-LINKS: REQ-EMG-176
// ENGMODEL-LINKS: FU-MODEL-LOADER
func TestManifestCodeCoverageRejectsInvalidBlocks(t *testing.T) {
	cases := []struct{ name, block string }{
		{"unknown key in an entry", strings.Replace(codeCoverageBlock, "decision: ADR-A", "decision: ADR-A\n      owner: me", 1)},
		{"unknown key in the block", codeCoverageBlock + "  skip: [FU-A]\n"},
		{"short reason", strings.Replace(codeCoverageBlock, "implemented by an external system", "too short", 1)},
		{"nineteen characters", strings.Replace(codeCoverageBlock, "implemented by an external system", "abcdefghijklmnopqrs", 1)},
		{"missing reason", strings.Replace(codeCoverageBlock, "      reason: implemented by an external system\n", "", 1)},
		{"empty decision", strings.Replace(codeCoverageBlock, "decision: ADR-A", `decision: ""`, 1)},
		{"empty id", strings.Replace(codeCoverageBlock, "id: FU-A", `id: ""`, 1)},
		{"missing id", strings.Replace(codeCoverageBlock, "    - id: FU-A\n      reason", "    - reason", 1)},
	}
	for _, c := range cases {
		dir := writeV2Fixture(t, validManifest+c.block)
		if _, err := LoadBundle(filepath.Join(dir, "engmod.yml")); err == nil {
			t.Errorf("%s: the block must fail loading", c.name)
		}
	}
	ok := strings.Replace(codeCoverageBlock, "implemented by an external system", "abcdefghijklmnopqrst", 1)
	if _, err := LoadBundle(filepath.Join(writeV2Fixture(t, validManifest+ok), "engmod.yml")); err != nil {
		t.Errorf("a reason of exactly 20 characters must load: %v", err)
	}
}
