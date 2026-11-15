// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"errors"
	"reflect"
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

// wantDeepEqual fails the test when got and want are not deeply equal.
func wantDeepEqual(t testing.TB, got, want any) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %+v, want %+v", got, want)
	}
}

// wantErrorIs fails the test when err is not, or does not wrap, target.
func wantErrorIs(t testing.TB, err, target error) {
	t.Helper()
	if !errors.Is(err, target) {
		t.Errorf("error %v, want %v", err, target)
	}
}
