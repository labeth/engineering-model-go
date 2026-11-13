// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"reflect"
	"testing"
)

// The facts walk counts the constants of a named type in every form Go writes
// them: an explicit type, an iota block that repeats the type, a conversion
// value and several names in one spec; an untyped constant is not counted.
// Mutation: dropping the implicit-repetition branch of constType (return ""
// instead of previous) makes the iota block count 1.
// TRLC-LINKS: REQ-EMG-188
func TestGoStateFactsCountConstants(t *testing.T) {
	src := `package p
type S int
type C string
const (
	A S = iota
	B
	_
	D
)
const E, F S = 1, 2
const G = S(7)
const H = 5
const (
	I C = "i"
	J = "j"
	K = C("k")
)
`
	facts, err := ExtractGoStateFacts("p.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]int{"S": 6, "C": 2}
	if !reflect.DeepEqual(facts.Consts, want) {
		t.Fatalf("constants %v, want %v", facts.Consts, want)
	}
}

// The facts walk records struct fields of a plain named type, assignments to
// selector fields (a composite literal, a short declaration and a plain
// variable are not assignments to a field) and the type declaration lines.
// Mutation: treating a composite literal key as an assignment, or recording
// pointer-typed fields, changes the sets.
// TRLC-LINKS: REQ-EMG-188
func TestGoStateFactsFieldsAndAssignments(t *testing.T) {
	src := `package p
type S int
type (
	T int
)
type H struct {
	state, other S
	p *S
	embedded
	n int
}
func f(h *H) {
	h.state = 1
	h.count += 2
	x := H{other: 1}
	var y S
	y = 3
	_, _ = x, y
}
`
	facts, err := ExtractGoStateFacts("p.go", []byte(src))
	if err != nil {
		t.Fatal(err)
	}
	if want := []GoFieldDecl{{"state", "S"}, {"other", "S"}, {"n", "int"}}; !reflect.DeepEqual(facts.Fields, want) {
		t.Errorf("fields %v, want %v", facts.Fields, want)
	}
	if want := map[string]bool{"state": true, "count": true}; !reflect.DeepEqual(facts.Assigned, want) {
		t.Errorf("assigned %v, want %v", facts.Assigned, want)
	}
	wantTypes := []GoTypeDecl{{"S", 2, 2}, {"T", 4, 3}, {"H", 6, 6}}
	if !reflect.DeepEqual(facts.Types, wantTypes) {
		t.Errorf("types %v, want %v", facts.Types, wantTypes)
	}
}

// A suppression is a comment that starts with the marker; the reason is the
// text after the colon on that line, and a mention inside a sentence is not a
// suppression.
// Mutation: matching the marker anywhere in the comment makes the sentence
// case a suppression.
// TRLC-LINKS: REQ-EMG-188
func TestGoStateSuppressionComments(t *testing.T) {
	cases := []struct {
		comment string
		reason  string
		ok      bool
	}{
		{"// ENGMODEL-NOT-A-STATE: a colour of the legend", "a colour of the legend", true},
		{"//ENGMODEL-NOT-A-STATE:   padded   ", "padded", true},
		{"// ENGMODEL-NOT-A-STATE", "", true},
		{"// ENGMODEL-NOT-A-STATE reason without colon", "reason without colon", true},
		{"/* ENGMODEL-NOT-A-STATE: block form */", "block form", true},
		{"// see ENGMODEL-NOT-A-STATE: in the docs", "", false},
		{"// ENGMODEL-NOT-A-STATE-EXTRA: nope", "", false},
		{"// something else", "", false},
	}
	for _, c := range cases {
		got, ok := suppressionReason(c.comment)
		if got != c.reason || ok != c.ok {
			t.Errorf("%q: got (%q, %v), want (%q, %v)", c.comment, got, ok, c.reason, c.ok)
		}
	}
}

// The excluded suffixes are case sensitive and end the name.
// Mutation: dropping a suffix from goStateExcludedSuffixes, or comparing
// case-insensitively, fails a row.
// TRLC-LINKS: REQ-EMG-188
func TestGoStateExcludedName(t *testing.T) {
	cases := map[string]bool{
		"ElementKind": true, "Severity": true, "ExportFormat": true, "MachineType": true,
		"Reason": true, "RunMode": true, "FlagOption": true,
		"State": false, "Status": false, "Phase": false, "Kinds": false, "kindly": false, "typed": false,
	}
	for name, want := range cases {
		if got := GoStateExcludedName(name); got != want {
			t.Errorf("%s: excluded %v, want %v", name, got, want)
		}
	}
}

// The candidate rule needs all of: three constants, a field of the type and
// an assignment to a field of that name, summed over the files of a package;
// a type of another directory does not count.
// Mutation: lowering GoStateMinConstants to 2, or merging by file instead of
// by directory, changes the result.
// TRLC-LINKS: REQ-EMG-188
func TestGoStateCandidatesRule(t *testing.T) {
	typeFile := "package p\ntype Phase int\ntype Lonely int\n"
	constFile := "package p\nconst (\n\tA Phase = iota\n\tB\n\tC\n)\nconst (\n\tL1 Lonely = iota\n\tL2\n\tL3\n)\n"
	useFile := "package p\ntype H struct{ phase Phase }\nfunc f(h *H) { h.phase = A }\n"
	other := "package q\nfunc g(h *H) { h.phase = 1 }\n"
	var facts []GoStateFacts
	for path, src := range map[string]string{"d/a.go": typeFile, "d/b.go": constFile, "d/c.go": useFile, "e/q.go": other} {
		f, err := ExtractGoStateFacts(path, []byte(src))
		if err != nil {
			t.Fatal(err)
		}
		facts = append(facts, f)
	}
	got := GoStateCandidates(facts)
	want := []GoStateCandidate{{File: "d/a.go", Type: "Phase", Line: 2, DeclLine: 2, Constants: 3, Field: "phase"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("candidates %+v, want %+v", got, want)
	}
}
