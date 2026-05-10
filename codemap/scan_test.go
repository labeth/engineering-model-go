// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"os"
	"path/filepath"
	"testing"
)

// TRLC-LINKS: REQ-EMG-010
func TestScan(t *testing.T) {
	root := filepath.Join("..", "examples", "payments-engineering-sample", "src")
	symbols, diags, err := Scan(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got: %+v", diags)
	}
	if len(symbols) < 2 {
		t.Fatalf("expected at least 2 symbols, got %d", len(symbols))
	}

	foundCheckout := false
	foundPaymentEngine := false
	foundTS := false
	foundRustReview := false
	for _, s := range symbols {
		if s.TraceID == "CODE-STARTSESSION" {
			foundCheckout = true
		}
		if s.TraceID == "CODE-AUTHORIZEPAYMENT" {
			foundPaymentEngine = true
		}
		if s.TraceID == "CODE-CALCULATERISKSCORE" {
			foundTS = true
		}
		if s.TraceID == "CODE-NOTIFYSUPPORT" {
			foundRustReview = true
		}
	}
	if !foundCheckout || !foundPaymentEngine || !foundTS || !foundRustReview {
		t.Fatalf("missing expected symbols in scan result: %+v", symbols)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_IgnoresMarkerTextInsideStringLiterals(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample_test.go")
	content := `package sample

// TRLC-LINKS: REQ-SAMPLE-001
func TestFixture(t interface{}) {
	marker := "// TRLC-LINKS: REQ-IGNORED-001\n"
	_ = marker
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, diags, err := Scan(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got: %+v", diags)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_RequiresTraceMarkersOnDeclarationLevel(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample_test.go")
	content := `package sample

// TRLC-LINKS: REQ-SAMPLE-001
var fixture = "not a declaration-level trace target"

// TRLC-LINKS: REQ-SAMPLE-002
func TestFixture(t interface{}) {
	_ = fixture
}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	symbols, diags, err := Scan(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(symbols) != 1 {
		t.Fatalf("expected one symbol for attached function marker, got %+v", symbols)
	}
	if len(symbols[0].Implements) != 1 || symbols[0].Implements[0] != "REQ-SAMPLE-002" {
		t.Fatalf("expected attached function marker to be used, got %+v", symbols[0])
	}
	found := false
	for _, d := range diags {
		if d.Code == "code.trace_unattached" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected code.trace_unattached diagnostic, got %+v", diags)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_AttachesTRLCLinksToGoTypeDeclarations(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "schema.go")
	content := `package sample

// TRLC-LINKS: REQ-SAMPLE-001
type Document struct {
	ID string
}

// TRLC-LINKS: REQ-SAMPLE-002
type Alias = string
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	symbols, diags, err := Scan(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diags)
	}
	if len(symbols) != 2 {
		t.Fatalf("expected two type symbols, got %+v", symbols)
	}
	if symbols[0].TraceID != "CODE-DOCUMENT" || symbols[0].Implements[0] != "REQ-SAMPLE-001" {
		t.Fatalf("unexpected first type symbol: %+v", symbols[0])
	}
	if symbols[1].TraceID != "CODE-ALIAS" || symbols[1].Implements[0] != "REQ-SAMPLE-002" {
		t.Fatalf("unexpected second type symbol: %+v", symbols[1])
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_RequiresTRLCLinksOnEveryFunction(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	content := `package sample

type Fixture struct{}

func missingOne() {}

// TRLC-LINKS: REQ-SAMPLE-001
func linked() {}

func missingTwo() {}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, diags, err := Scan(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	var missing *validateDiagnostic
	for i := range diags {
		if diags[i].Code == "code.missing_trlc_link" {
			missing = &validateDiagnostic{
				Severity: string(diags[i].Severity),
				Message:  diags[i].Message,
				Path:     diags[i].Path,
			}
			break
		}
	}
	if missing == nil {
		t.Fatalf("expected code.missing_trlc_link diagnostic, got %+v", diags)
	}
	if missing.Severity != "error" {
		t.Fatalf("expected error severity, got %+v", missing)
	}
	if missing.Path != "sample.go:5,10" {
		t.Fatalf("expected file path with comma-listed lines, got %+v", missing)
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestScan_RejectsMalformedTRLCLinks(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "sample.go")
	content := `package sample

// TRLC-` + `LINKS: "); ok {
func broken() {}
`
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}

	_, diags, err := Scan(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	foundInvalid := false
	for _, d := range diags {
		if d.Code == "code.invalid_trlc_link" && d.Severity == "error" {
			foundInvalid = true
			break
		}
	}
	if !foundInvalid {
		t.Fatalf("expected code.invalid_trlc_link diagnostic, got %+v", diags)
	}
}

type validateDiagnostic struct {
	Severity string
	Message  string
	Path     string
}
