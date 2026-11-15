// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-010
func TestScan(t *testing.T) {
	symbols, diags := scanRoot(t, filepath.Join("..", "..", "examples", "payments-engineering-sample", "src"))
	requireNoDiags(t, diags)
	for _, id := range []string{
		"CODE-CHECKOUT-API-GO--STARTSESSION",
		"CODE-PAYMENT-ENGINE-RS--AUTHORIZEPAYMENT",
		"CODE-RISK-SCORER-TS--CALCULATERISKSCORE",
		"CODE-PAYMENT-ENGINE-RS--NOTIFYSUPPORT",
	} {
		requireSymbol(t, symbols, id)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_IgnoresMarkerTextInsideStringLiterals(t *testing.T) {
	content := `package sample

// TRLC-LINKS: REQ-SAMPLE-001
func TestFixture(t interface{}) {
	marker := "// TRLC-LINKS: REQ-IGNORED-001\n"
	_ = marker
}
`
	_, diags := scanSource(t, "sample_test.go", content)
	requireNoDiags(t, diags)
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_RequiresTraceMarkersOnDeclarationLevel(t *testing.T) {
	content := `package sample

// TRLC-LINKS: REQ-SAMPLE-002
func TestFixture(t interface{}) {
}

// TRLC-LINKS: REQ-SAMPLE-001
`
	symbols, diags := scanSource(t, "sample_test.go", content)
	requireEqual(t, len(symbols), 1)
	requireEqual(t, symbols[0].Implements, []string{"REQ-SAMPLE-002"})
	requireDiag(t, diags, "code.trace_unattached")
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_AttachesTRLCLinksToGoTypeDeclarations(t *testing.T) {
	content := `package sample

// TRLC-LINKS: REQ-SAMPLE-001
type Document struct {
	ID string
}

// TRLC-LINKS: REQ-SAMPLE-002
type Alias = string
`
	symbols, diags := scanSource(t, "schema.go", content)
	requireNoDiags(t, diags)
	requireEqual(t, len(symbols), 2)
	requireEqual(t, linksOf(symbols[0]), symbolLinks{TraceID: "CODE-SCHEMA-GO--DOCUMENT", Implements: []string{"REQ-SAMPLE-001"}})
	requireEqual(t, linksOf(symbols[1]), symbolLinks{TraceID: "CODE-SCHEMA-GO--ALIAS", Implements: []string{"REQ-SAMPLE-002"}})
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_AttachesEngmodelLinksToDeclarations(t *testing.T) {
	content := `package sample

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-SAMPLE-001
type Document struct {
	ID string
}
`
	symbols, diags := scanSource(t, "schema.go", content)
	requireNoDiags(t, diags)
	requireEqual(t, len(symbols), 1)
	requireEqual(t, linksOf(symbols[0]), symbolLinks{
		TraceID:    "CODE-SCHEMA-GO--DOCUMENT",
		Implements: []string{"REQ-SAMPLE-001"},
		ModelLinks: []string{"FU-CODEMAP-INFERENCE", "CTRL-TRACEABILITY-COVERAGE", "DEP-LOCAL-WORKSPACE"},
	})
}

// Stacked ENGMODEL-LINKS lines above one declaration accumulate in the order
// written and a repeated identifier is kept once.
// Mutation: assigning instead of appending the identifiers of a line leaves only the last line.
// TRLC-LINKS: REQ-EMG-174
func TestScan_StackedEngmodelLinksAccumulate(t *testing.T) {
	content := `package sample

// ENGMODEL-LINKS: FU-ONE, FU-TWO
// ENGMODEL-LINKS: FU-TWO, FU-THREE
// TRLC-LINKS: REQ-SAMPLE-001
type Document struct{}
`
	symbols, _ := scanSource(t, "schema.go", content)
	requireEqual(t, len(symbols), 1)
	requireEqual(t, symbols[0].ModelLinks, []string{"FU-ONE", "FU-TWO", "FU-THREE"})
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_RequiresTRLCLinksOnEveryFunction(t *testing.T) {
	content := `package sample

type Fixture struct{}

func missingOne() {}

// TRLC-LINKS: REQ-SAMPLE-001
func linked() {}

func missingTwo() {}
`
	_, diags := scanSource(t, "sample.go", content)
	missing := requireDiag(t, diags, "code.missing_trlc_link")
	requireEqual(t, string(missing.Severity), "error")
	requireEqual(t, missing.Path, "sample.go:5,10")
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_RejectsMalformedTRLCLinks(t *testing.T) {
	content := `package sample

// TRLC-` + `LINKS: "); ok {
func broken() {}
`
	_, diags := scanSource(t, "sample.go", content)
	requireEqual(t, string(requireDiag(t, diags, "code.invalid_trlc_link").Severity), "error")
}
