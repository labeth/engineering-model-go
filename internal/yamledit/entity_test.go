// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package yamledit

import (
	"strings"
	"testing"
)

const assuranceFile = `schemaVersion: 3
assurance:
  # risks owned by the security lead
  risks:
    - id: RISK-A
      title: First risk   # keep this comment
      status: open
      appliesTo: [FU-A, FU-B]
      relatedControls:
        - CTRL-A
    - id: RISK-B
      title: Second risk
      status: accepted
      acceptedBy:
        actor: ACT-OWNER
        date: "2026-11-03"
  flows:
    - id: FLOW-X
      steps:
        - id: STEP-1
          ref: FU-A
`

// A stable ID finds its entity anywhere in the document (a risk, a flow step
// nested in a flow); an unknown ID and a duplicated ID are refused.
// Mutation: matching only top-level lists (no STEP-1), or returning the first
// of two duplicates, fails this test.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MODEL-CHANGE
func TestFindByID(t *testing.T) {
	root, err := Parse([]byte(assuranceFile))
	if err != nil {
		t.Fatal(err)
	}
	for id, wantLine := range map[string]int{"RISK-B": 11, "STEP-1": 20} {
		node, err := FindByID(id)(root)
		if err != nil || node.Line != wantLine {
			t.Fatalf("FindByID(%s) = line %v, %v; want line %d", id, node, err, wantLine)
		}
	}
	if _, err := FindByID("RISK-NONE")(root); err == nil || !strings.Contains(err.Error(), "no entity with id RISK-NONE") {
		t.Fatalf("unknown id: %v", err)
	}
	dup, _ := Parse([]byte(strings.Replace(assuranceFile, "- id: RISK-B", "- id: RISK-A", 1)))
	if _, err := FindByID("RISK-A")(dup); err == nil || !strings.Contains(err.Error(), "not unique") {
		t.Fatalf("duplicate id: %v", err)
	}
}

// SetPath replaces a scalar in place (its comment and every other line kept),
// adds a missing key, creates a missing nested mapping (acceptedBy.statement),
// writes typed values (bool, number, "true" as a string) and a list as a block;
// UnsetPath removes a key with its lines; AppendPath and RemovePath edit block
// and flow lists, keeping flow lists flow.
// Mutation: rewriting the whole line, dropping the comment, writing the
// nested key at the entity's indentation, quoting a bool or turning a flow
// list into a block fails this test.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MODEL-CHANGE
func TestEntityPathEdits(t *testing.T) {
	riskA, riskB := FindByID("RISK-A"), FindByID("RISK-B")
	for _, tc := range []struct {
		name string
		edit func([]byte) ([]byte, error)
		old  string
		new  string
	}{
		{"scalar in place keeps comment", func(d []byte) ([]byte, error) { return SetPath(d, riskA, []string{"title"}, "Renamed: risk") },
			"      title: First risk   # keep this comment\n", "      title: 'Renamed: risk'   # keep this comment\n"},
		{"missing key added", func(d []byte) ([]byte, error) { return SetPath(d, riskA, []string{"owner"}, "ACT-OWNER") },
			"        - CTRL-A\n", "        - CTRL-A\n      owner: ACT-OWNER\n"},
		{"nested key in existing mapping", func(d []byte) ([]byte, error) {
			return SetPath(d, riskB, []string{"acceptedBy", "statement"}, "I accept this risk.")
		}, "        date: \"2026-11-03\"\n", "        date: \"2026-11-03\"\n        statement: I accept this risk.\n"},
		{"nested mapping created", func(d []byte) ([]byte, error) {
			return SetPath(d, riskA, []string{"acceptedBy", "statement"}, "Owner words.")
		}, "        - CTRL-A\n", "        - CTRL-A\n      acceptedBy:\n        statement: Owner words.\n"},
		{"bool and string true", func(d []byte) ([]byte, error) {
			d, err := SetPath(d, FindByID("STEP-1"), []string{"boundaryCrossing"}, true)
			if err != nil {
				return nil, err
			}
			return SetPath(d, FindByID("STEP-1"), []string{"action"}, "true")
		}, "          ref: FU-A\n", "          ref: FU-A\n          boundaryCrossing: true\n          action: \"true\"\n"},
		{"list value as block", func(d []byte) ([]byte, error) {
			return SetPath(d, riskA, []string{"relatedControls"}, []any{"CTRL-B", "CTRL-C"})
		}, "        - CTRL-A\n", "        - CTRL-B\n        - CTRL-C\n"},
		{"unset removes lines", func(d []byte) ([]byte, error) { return UnsetPath(d, riskB, []string{"acceptedBy"}) },
			"      acceptedBy:\n        actor: ACT-OWNER\n        date: \"2026-11-03\"\n", ""},
		{"append block", func(d []byte) ([]byte, error) {
			return AppendPath(d, riskA, []string{"relatedControls"}, []any{"CTRL-B"})
		},
			"        - CTRL-A\n", "        - CTRL-A\n        - CTRL-B\n"},
		{"append flow stays flow", func(d []byte) ([]byte, error) { return AppendPath(d, riskA, []string{"appliesTo"}, []any{"FU-C"}) },
			"appliesTo: [FU-A, FU-B]\n", "appliesTo: [FU-A, FU-B, FU-C]\n"},
		{"append object to nested list", func(d []byte) ([]byte, error) {
			return AppendPath(d, FindByID("FLOW-X"), []string{"steps"}, []any{map[string]any{"ref": "FU-B", "id": "STEP-2"}})
		}, "          ref: FU-A\n", "          ref: FU-A\n        - id: STEP-2\n          ref: FU-B\n"},
		{"append creates list", func(d []byte) ([]byte, error) { return AppendPath(d, riskB, []string{"appliesTo"}, []any{"FU-A"}) },
			"        date: \"2026-11-03\"\n", "        date: \"2026-11-03\"\n      appliesTo:\n        - FU-A\n"},
		{"remove from flow", func(d []byte) ([]byte, error) { return RemovePath(d, riskA, []string{"appliesTo"}, []any{"FU-A"}) },
			"appliesTo: [FU-A, FU-B]\n", "appliesTo: [FU-B]\n"},
		{"remove last block item leaves empty list", func(d []byte) ([]byte, error) {
			return RemovePath(d, riskA, []string{"relatedControls"}, []any{"CTRL-A"})
		}, "      relatedControls:\n        - CTRL-A\n", "      relatedControls: []\n"},
		{"remove object by id", func(d []byte) ([]byte, error) {
			return RemovePath(d, FindByID("FLOW-X"), []string{"steps"}, []any{"STEP-1"})
		},
			"      steps:\n        - id: STEP-1\n          ref: FU-A\n", "      steps: []\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			got, err := tc.edit([]byte(assuranceFile))
			want := strings.Replace(assuranceFile, tc.old, tc.new, 1)
			if err != nil || string(got) != want {
				t.Fatalf("err %v\ngot\n%s\nwant\n%s", err, got, want)
			}
		})
	}
	for name, edit := range map[string]func() error{
		"unset missing": func() error { _, err := UnsetPath([]byte(assuranceFile), riskA, []string{"owner"}); return err },
		"remove missing item": func() error {
			_, err := RemovePath([]byte(assuranceFile), riskA, []string{"appliesTo"}, []any{"FU-Z"})
			return err
		},
		"append to scalar": func() error {
			_, err := AppendPath([]byte(assuranceFile), riskA, []string{"title"}, []any{"x"})
			return err
		},
		"set through scalar": func() error { _, err := SetPath([]byte(assuranceFile), riskA, []string{"title", "x"}, "y"); return err },
		"set null":           func() error { _, err := SetPath([]byte(assuranceFile), riskA, []string{"title"}, nil); return err },
	} {
		if err := edit(); err == nil {
			t.Errorf("%s: expected an error", name)
		}
	}
}

// UnifiedDiff renders a patch git apply accepts: file headers, a hunk header
// with line counts and three lines of context.
// Mutation: dropping the hunk header or the context lines fails this test.
// TRLC-LINKS: REQ-EMG-113
// ENGMODEL-LINKS: FU-MODEL-CHANGE
func TestUnifiedDiff(t *testing.T) {
	before := "a\nb\nc\nd\ne\nf\ng\nh\n"
	after := "a\nb\nc\nd\nE\nf\ng\nh\n"
	want := "--- a/m.yml\n+++ b/m.yml\n@@ -2,7 +2,7 @@\n b\n c\n d\n-e\n+E\n f\n g\n h\n"
	if got := UnifiedDiff("m.yml", []byte(before), []byte(after)); got != want {
		t.Fatalf("got\n%s\nwant\n%s", got, want)
	}
	if got := UnifiedDiff("m.yml", []byte(before), []byte(before)); got != "" {
		t.Fatalf("no change must give no patch, got %q", got)
	}
	added := UnifiedDiff("m.yml", []byte("a\n"), []byte("a\nb\n"))
	if added != "--- a/m.yml\n+++ b/m.yml\n@@ -1 +1,2 @@\n a\n+b\n" {
		t.Fatalf("append patch:\n%s", added)
	}
}
