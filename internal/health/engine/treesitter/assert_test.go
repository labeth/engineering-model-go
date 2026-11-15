// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package treesitter

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
