// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package textutil

import (
	"reflect"
	"testing"
)

// TRLC-LINKS: REQ-EMG-003
func TestNonEmpty(t *testing.T) {
	cases := []struct{ v, fb, want string }{
		{"a", "x", "a"},
		{" a ", "x", " a "},
		{"", "x", "x"},
		{" \t\n", "x", "x"},
		{"", "", ""},
	}
	for _, c := range cases {
		if got := NonEmpty(c.v, c.fb); got != c.want {
			t.Errorf("NonEmpty(%q, %q) = %q, want %q", c.v, c.fb, got, c.want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-015
func TestFallback(t *testing.T) {
	cases := []struct{ s, def, want string }{
		{"a", "d", "a"},
		{" a", "d", " a"},
		{"", "d", "d"},
		{"   ", "d", "d"},
	}
	for _, c := range cases {
		if got := Fallback(c.s, c.def); got != c.want {
			t.Errorf("Fallback(%q, %q) = %q, want %q", c.s, c.def, got, c.want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestUniqueSorted(t *testing.T) {
	cases := []struct {
		in, want []string
	}{
		{nil, []string{}},
		{[]string{" ", ""}, []string{}},
		{[]string{"b", " a ", "b", "c", "a"}, []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		got := UniqueSorted(c.in)
		if got == nil || !reflect.DeepEqual(got, c.want) {
			t.Errorf("UniqueSorted(%q) = %#v, want %#v", c.in, got, c.want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestAppendUnique(t *testing.T) {
	cases := []struct {
		in   []string
		x    string
		want []string
	}{
		{nil, "a", []string{"a"}},
		{nil, "  ", nil},
		{[]string{"a"}, " a ", []string{"a"}},
		{[]string{"a"}, " b ", []string{"a", "b"}},
	}
	for _, c := range cases {
		if got := AppendUnique(c.in, c.x); !reflect.DeepEqual(got, c.want) {
			t.Errorf("AppendUnique(%q, %q) = %#v, want %#v", c.in, c.x, got, c.want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-010
func TestAppendUniqueSlice(t *testing.T) {
	cases := []struct {
		in, items, want []string
	}{
		{nil, nil, nil},
		{[]string{"a"}, []string{"b", "a", " ", "b ", "c"}, []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		if got := AppendUniqueSlice(c.in, c.items); !reflect.DeepEqual(got, c.want) {
			t.Errorf("AppendUniqueSlice(%q, %q) = %#v, want %#v", c.in, c.items, got, c.want)
		}
	}
	in := make([]string, 1, 4)
	in[0] = "a"
	_ = AppendUniqueSlice(in, []string{"b"})
	if in[:2][1] != "" {
		t.Fatalf("AppendUniqueSlice must not write into the input's backing array")
	}
}

// TRLC-LINKS: REQ-EMG-015
func TestSortedKeys(t *testing.T) {
	cases := []struct {
		m    map[string]int
		want []string
	}{
		{nil, []string{}},
		{map[string]int{"b": 1, "a": 2, "c": 3}, []string{"a", "b", "c"}},
	}
	for _, c := range cases {
		if got := SortedKeys(c.m); !reflect.DeepEqual(got, c.want) {
			t.Errorf("SortedKeys(%v) = %#v, want %#v", c.m, got, c.want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-013
func TestDeterministicUUID(t *testing.T) {
	cases := []struct{ seed, want string }{
		{"", "da39a3ee-5e6b-5b0d-b255-bfef95601890"},
		{"ssp:system", "ce381b05-24ab-53d7-a6a6-2ee6800a3cce"},
	}
	for _, c := range cases {
		if got := DeterministicUUID(c.seed); got != c.want {
			t.Errorf("DeterministicUUID(%q) = %q, want %q", c.seed, got, c.want)
		}
	}
	if DeterministicUUID("a") == DeterministicUUID("b") {
		t.Fatal("distinct seeds must give distinct UUIDs")
	}
}
