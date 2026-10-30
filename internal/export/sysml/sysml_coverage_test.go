// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package sysml

import (
	"encoding/json"
	"strings"
	"testing"
)

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func TestValidateSysMLV2CoverageRejectsEveryZeroGapFailureMode(t *testing.T) {
	tests := []struct {
		name   string
		mutate func(*CoverageManifest)
		want   string
	}{
		{"missing metaclass renderer", func(m *CoverageManifest) { m.Metaclasses = m.Metaclasses[1:] }, "missing metaclass renderer"},
		{"unaccounted property", func(m *CoverageManifest) { m.Properties = m.Properties[1:] }, "unaccounted official property"},
		{"unclassified relationship", func(m *CoverageManifest) { m.Relationships = m.Relationships[1:] }, "unclassified relationship"},
		{"partial normative status", func(m *CoverageManifest) { m.Metaclasses[0].Status = "partial" }, "invalid metaclass coverage"},
		{"unsupported normative status", func(m *CoverageManifest) { m.Properties[0].Status = "unsupported" }, "invalid official property coverage"},
		{"inventory hash drift", func(m *CoverageManifest) { m.Authority.InventorySHA256 = strings.Repeat("0", 64) }, "inventory hash drift"},
		{"source hash drift", func(m *CoverageManifest) { m.Authority.SourceManifestSHA256 = strings.Repeat("0", 64) }, "source manifest hash drift"},
		{"parser failure", func(m *CoverageManifest) { setCoverageCheck(m, "official-parser", "failed") }, "official-parser"},
		{"stale artifact", func(m *CoverageManifest) { setCoverageCheck(m, "artifact-freshness", "failed") }, "artifact-freshness"},
		{"native KPAR round-trip difference", func(m *CoverageManifest) { setCoverageCheck(m, "native-kpar-round-trip", "failed") }, "native-kpar-round-trip"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manifest := cloneCoverageManifest(t, V2Coverage())
			test.mutate(&manifest)
			if err := ValidateV2Coverage(manifest); err == nil || !strings.Contains(err.Error(), test.want) {
				t.Fatalf("expected %q failure, got %v", test.want, err)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func cloneCoverageManifest(t *testing.T, manifest CoverageManifest) CoverageManifest {
	t.Helper()
	data, err := json.Marshal(manifest)
	if err != nil {
		t.Fatal(err)
	}
	var cloned CoverageManifest
	if err := json.Unmarshal(data, &cloned); err != nil {
		t.Fatal(err)
	}
	return cloned
}

// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func setCoverageCheck(manifest *CoverageManifest, id, status string) {
	for index := range manifest.Checks {
		if manifest.Checks[index].ID == id {
			manifest.Checks[index].Status = status
			return
		}
	}
}
