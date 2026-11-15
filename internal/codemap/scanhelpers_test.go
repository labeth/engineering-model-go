// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// scanRoot scans root and fails the test when the scan fails.
func scanRoot(t *testing.T, root string) ([]Symbol, []validate.Diagnostic) {
	t.Helper()
	symbols, diags, err := Scan(root)
	if err != nil {
		t.Fatalf("scan failed: %v", err)
	}
	return symbols, diags
}

// scanSource writes one file into a fresh directory and scans it.
func scanSource(t *testing.T, name, content string) ([]Symbol, []validate.Diagnostic) {
	t.Helper()
	return scanRoot(t, writeTree(t, map[string]string{name: content}))
}

// requireNoDiags fails the test when the scan reported diagnostics.
func requireNoDiags(t *testing.T, diags []validate.Diagnostic) {
	t.Helper()
	if len(diags) != 0 {
		t.Fatalf("expected no diagnostics, got %+v", diags)
	}
}

// requireDiag returns the first diagnostic with the code and fails the test
// when there is none.
func requireDiag(t *testing.T, diags []validate.Diagnostic, code string) validate.Diagnostic {
	t.Helper()
	i := slices.IndexFunc(diags, func(d validate.Diagnostic) bool { return d.Code == code })
	if i < 0 {
		t.Fatalf("expected %s diagnostic, got %+v", code, diags)
	}
	return diags[i]
}

// requireSymbol returns the symbol with the trace identifier and fails the
// test when there is none.
func requireSymbol(t *testing.T, symbols []Symbol, id string) Symbol {
	t.Helper()
	i := slices.IndexFunc(symbols, func(s Symbol) bool { return s.TraceID == id })
	if i < 0 {
		t.Fatalf("missing symbol %s in %+v", id, symbols)
	}
	return symbols[i]
}

// requireEqual fails the test unless got deeply equals want.
func requireEqual[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// symbolLinks is the identity and the links of a symbol.
type symbolLinks struct {
	TraceID    string
	Implements []string
	ModelLinks []string
}

// linksOf returns the identity and the links of a symbol.
func linksOf(s Symbol) symbolLinks {
	return symbolLinks{TraceID: s.TraceID, Implements: s.Implements, ModelLinks: s.ModelLinks}
}

// writeSource writes one file into a fresh directory and returns its path.
func writeSource(t *testing.T, name, content string) string {
	t.Helper()
	return filepath.Join(writeTree(t, map[string]string{name: content}), name)
}

// requireErrorDiag fails the test unless diags holds an error with the code.
func requireErrorDiag(t *testing.T, diags []validate.Diagnostic, code string) {
	t.Helper()
	requireEqual(t, string(requireDiag(t, diags, code).Severity), "error")
}

// declarationNames returns the names of declarations in order.
func declarationNames(decls []declaration) []string {
	names := make([]string, 0, len(decls))
	for _, d := range decls {
		names = append(names, d.Name)
	}
	return names
}

// implementedBy returns the requirements all symbols implement.
func implementedBy(symbols []Symbol) []string {
	var all []string
	for _, s := range symbols {
		all = append(all, s.Implements...)
	}
	return all
}
