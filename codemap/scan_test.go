package codemap

import (
	"path/filepath"
	"testing"
)

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
