// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engmodel_test

import (
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/codescan"
)

// internal/codescan recognises a generated Quint module by the header the
// exporter writes, but may not import the exporter, so it holds a copy of the
// prefix. This test fails if the two differ.
//
// TRLC-LINKS: REQ-EMG-145
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func TestFormalGeneratedHeaderMatchesGenerator(t *testing.T) {
	if codescan.FormalGeneratedHeader != engmodel.QuintGeneratedHeader {
		t.Fatalf("codescan.FormalGeneratedHeader = %q, the Quint generator writes %q", codescan.FormalGeneratedHeader, engmodel.QuintGeneratedHeader)
	}
}
