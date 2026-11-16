// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"fmt"
	"strings"
	"testing"
)

// requireWrappedCoordinates fails the test unless the evidence label keeps its
// file name and line coordinates and gains a wrap point after each comma.
func requireWrappedCoordinates(t *testing.T, original string, lines []string) {
	t.Helper()
	input := []string{original}
	got := publicationTestCode(input)
	if input[0] != original {
		t.Fatal("canonical evidence was mutated")
	}
	if !strings.HasPrefix(got, "path/with,comma_test.go:") {
		t.Fatal("filename punctuation changed")
	}
	suffix := strings.TrimPrefix(got, "path/with,comma_test.go:")
	if strings.ReplaceAll(suffix, " ", "") != strings.Join(lines, ",") {
		t.Fatal("line coordinates changed")
	}
	if len(strings.Fields(suffix)) != 200 {
		t.Fatal("long table token lacks wrap points")
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestPublicationEvidenceLineListsWrapWithoutChangingCoordinates(t *testing.T) {
	lines := make([]string, 200)
	for i := range lines {
		lines[i] = fmt.Sprint(i + 1)
	}
	requireWrappedCoordinates(t, "path/with,comma_test.go:"+strings.Join(lines, ","), lines)
	for _, value := range []string{"file.go:17", "path/a,b.go", "file.go:label,other", "file.go:1,,2"} {
		if publicationTestCode([]string{value}) != value {
			t.Fatalf("unexpected rewrite of %q", value)
		}
	}
}

// requireAtomicCoordinates fails the test unless the label keeps its file name
// and coordinates and has a wrap opportunity between the coordinates.
func requireAtomicCoordinates(t *testing.T, original string) {
	t.Helper()
	got := diagramCodeEvidenceLabel(original)
	colon := strings.LastIndexByte(original, ':')
	if !strings.HasPrefix(got, original[:colon+1]+" ") {
		t.Fatalf("filename changed: %q", got)
	}
	if strings.ReplaceAll(got[colon+1:], " ", "") != original[colon+1:] {
		t.Fatalf("coordinates changed: %q", got)
	}
	if len(strings.Fields(got[colon+1:])) != len(strings.Split(original[colon+1:], ",")) {
		t.Fatalf("missing wrap opportunities: %q", got)
	}
}

// TRLC-LINKS: REQ-EMG-003
func TestDiagramCoordinatesHaveAtomicNumericTokens(t *testing.T) {
	for _, original := range []string{"bode_test.go:12,28,40,92,122,148", "path/with,comma.go:1,110,9999", "C:/src/file.go:123"} {
		requireAtomicCoordinates(t, original)
	}
	for _, original := range []string{"path/a,b.go", "file.go:label,other", "file.go:1,,2"} {
		if diagramCodeEvidenceLabel(original) != original {
			t.Fatalf("invalid coordinate suffix rewritten: %q", original)
		}
	}
}
