// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// requireEqual fails the test unless got deeply equals want.
func requireEqual[T any](t *testing.T, got, want T) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
}

// diagsOf returns the diagnostics with one of the codes.
func diagsOf(diags []validate.Diagnostic, codes ...string) []validate.Diagnostic {
	var out []validate.Diagnostic
	for _, d := range diags {
		if slices.Contains(codes, d.Code) {
			out = append(out, d)
		}
	}
	return out
}

// severitiesByCode maps each diagnostic code to the severity of its last
// diagnostic.
func severitiesByCode(diags []validate.Diagnostic) map[string]validate.Severity {
	out := map[string]validate.Severity{}
	for _, d := range diags {
		out[d.Code] = d.Severity
	}
	return out
}

// traceFixture is a model directory of files with the code and verification
// inferred from it.
type traceFixture struct {
	t        *testing.T
	root     string
	bundle   model.Bundle
	reqs     model.RequirementsDocument
	code     []codescan.CodeItem
	checks   []codescan.VerificationCheck
	codeDiag []validate.Diagnostic
	verDiag  []validate.Diagnostic
}

// newTraceFixture writes the files under a fresh model root.
func newTraceFixture(t *testing.T, files map[string]string) *traceFixture {
	t.Helper()
	root := t.TempDir()
	for name, text := range files {
		testutil.WriteFile(t, filepath.Join(root, filepath.FromSlash(name)), text)
	}
	return &traceFixture{t: t, root: root, bundle: model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}}
}

// infer infers the code and verification checks for requirements with the
// given ids and, when given, the requirements as they are.
func (f *traceFixture) infer(reqIDs ...string) {
	for _, id := range reqIDs {
		f.reqs.Requirements = append(f.reqs.Requirements, model.Requirement{ID: id})
	}
	f.inferred()
}

// inferred infers the code and verification checks for f.reqs.
func (f *traceFixture) inferred() {
	f.code, f.codeDiag = codescan.InferCodeItems(f.bundle, "")
	f.checks, f.verDiag = codescan.InferVerificationChecks(f.bundle, f.reqs, f.code, "")
}

// matrix is the trace matrix of the fixture.
func (f *traceFixture) matrix() Matrix {
	return buildTraceMatrix(f.bundle, Evidence{f.reqs, f.code, f.checks})
}

// diagnostics is the integrity diagnostics of the fixture.
func (f *traceFixture) diagnostics() []validate.Diagnostic {
	return ValidateIntegrity(f.bundle, Evidence{f.reqs, f.code, f.checks})
}

// requireInferred fails the test unless inference found the given numbers of
// code items and verification checks without diagnostics.
func (f *traceFixture) requireInferred(code, checks int) {
	f.t.Helper()
	requireEqual(f.t, []int{len(f.codeDiag), len(f.code), len(f.verDiag), len(f.checks)}, []int{0, code, 0, checks})
}

// requireFilesExist fails the test unless every symbol of the code items has
// an absolute source path that exists.
func requireFilesExist(t *testing.T, code []codescan.CodeItem) {
	t.Helper()
	for _, item := range code {
		if item.Kind == "symbol" {
			_, err := os.Stat(item.AbsPath)
			requireEqual(t, err, nil)
		}
	}
}

// countOf is the number of items equal to want.
func countOf[T comparable](items []T, want T) int {
	return len(slices.DeleteFunc(slices.Clone(items), func(item T) bool { return item != want }))
}
