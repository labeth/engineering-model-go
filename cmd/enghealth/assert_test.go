// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"strings"
	"testing"
)

// wantEqual fails the test when got differs from want.
func wantEqual[T comparable](t testing.TB, got, want T) {
	t.Helper()
	if got != want {
		t.Errorf("got %v, want %v", got, want)
	}
}

// mustEqual stops the test when got differs from want.
func mustEqual[T comparable](t testing.TB, got, want T) {
	t.Helper()
	if got != want {
		t.Fatalf("got %v, want %v", got, want)
	}
}

// wantContains fails the test when text lacks want.
func wantContains(t testing.TB, text, want string) {
	t.Helper()
	if !strings.Contains(text, want) {
		t.Errorf("%q lacks %q", text, want)
	}
}

// wantAll fails the test for each of the wants that text lacks.
func wantAll(t testing.TB, text string, wants ...string) {
	t.Helper()
	for _, want := range wants {
		wantContains(t, text, want)
	}
}

// wantNone fails the test when text holds unwanted.
func wantNone(t testing.TB, text, unwanted string) {
	t.Helper()
	if strings.Contains(text, unwanted) {
		t.Errorf("%q holds %q", text, unwanted)
	}
}

// cliCase is a run of the command and what it must do: its exit code and a text
// that each of its outputs must hold (empty: no requirement).
type cliCase struct {
	args           []string
	code           int
	stdout, stderr string
}

// requireCLI runs the command and checks the case; it returns the outputs.
func requireCLI(t *testing.T, c cliCase) (stdout, stderr string) {
	t.Helper()
	code, stdout, stderr := runCLI(c.args...)
	wantEqual(t, code, c.code)
	wantContains(t, stdout, c.stdout)
	wantContains(t, stderr, c.stderr)
	return stdout, stderr
}

// expect runs enghealth check on the project with the arguments of the case
// and checks it; it returns the outputs.
func (r *checkRepo) expect(c cliCase) (stdout, stderr string) {
	r.t.Helper()
	c.args = append([]string{"check", "--model", r.manifest}, c.args...)
	return requireCLI(r.t, c)
}
