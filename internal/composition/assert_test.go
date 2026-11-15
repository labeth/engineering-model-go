// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

import (
	"os"
	"reflect"
	"slices"
	"testing"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// requireEqual fails the test unless got deeply equals want.
func requireEqual[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// readText returns the content of a file and fails the test when it cannot be read.
func readText(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

// mustResolve resolves the composition of a manifest and fails the test on an
// error that is not a diagnostic.
func mustResolve(t *testing.T, manifest string, options ResolutionOptions) CompositionResult {
	t.Helper()
	res, err := GenerateCompositionFromFileWith(manifest, options)
	if err != nil {
		t.Fatal(err)
	}
	return res
}

// firstDiag returns the first diagnostic with the code, or nil.
func firstDiag(diags []validate.Diagnostic, code string) *validate.Diagnostic {
	i := slices.IndexFunc(diags, func(d validate.Diagnostic) bool { return d.Code == code })
	if i < 0 {
		return nil
	}
	return &diags[i]
}

// requireNoFile fails the test when the path exists.
func requireNoFile(t *testing.T, path string) {
	t.Helper()
	_, err := os.Lstat(path)
	requireEqual(t, os.IsNotExist(err), true)
}
