// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package yamledit

import (
	"strings"
	"testing"
)

const linkFile = `schemaVersion: 2
reqLinks:
# reviewed links
- id: LNK-A
  parent: REQ-A   # the parent
  child: dep::CAP-A
  pinned:
    version: "v0.1.0"
    fingerprint: 'sha256:old'
  status: claimed
- id: LNK-B
  child: dep::AOU-B
  evidence:
  - REQ-A
  pinned: {version: v0.1.0, fingerprint: x}
  status: claimed
`

// SetKey replaces a single-line scalar in place (keeping quotes' absence,
// a trailing comment and every other byte), adds a missing key after the
// mapping's last key, replaces a block value with its lines, and refuses a
// key that would need a flow mapping rewritten.
// Mutation: rewriting the whole line instead of the value, appending a new
// key at the wrong indentation or accepting a flow-mapping insert fails
// this test.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
// ENGMODEL-LINKS: FU-MODEL-CHANGE
func TestSetKey(t *testing.T) {
	linkA, pinA := FindListItem([]string{"reqLinks"}, "id", "LNK-A"), FindListItem([]string{"reqLinks"}, "id", "LNK-A", "pinned")
	linkB, pinB := FindListItem([]string{"reqLinks"}, "id", "LNK-B"), FindListItem([]string{"reqLinks"}, "id", "LNK-B", "pinned")
	for _, tc := range []struct {
		name, key, raw string
		want, err      string
	}{
		{name: "plain in place keeps comment", key: "parent", raw: "REQ-Z",
			want: strings.Replace(linkFile, "  parent: REQ-A   # the parent\n", "  parent: REQ-Z   # the parent\n", 1)},
		{name: "quoted value replaced", key: "version", raw: "v0.2.0",
			want: strings.Replace(linkFile, `    version: "v0.1.0"`, `    version: v0.2.0`, 1)},
		{name: "missing key added at item indent", key: "rationale", raw: "Because.",
			want: strings.Replace(linkFile, "  status: claimed\n- id: LNK-B", "  status: claimed\n  rationale: Because.\n- id: LNK-B", 1)},
		{name: "block value replaced", key: "evidence", raw: "[REQ-C]",
			want: strings.Replace(linkFile, "  evidence:\n  - REQ-A\n", "  evidence: [REQ-C]\n", 1)},
		{name: "flow scalar in place", key: "fingerprint", raw: "y",
			want: strings.Replace(linkFile, "fingerprint: x}", "fingerprint: y}", 1)},
		{name: "flow insert refused", key: "extra", raw: "1", err: "flow mapping"},
		{name: "unknown item refused", key: "status", raw: "accepted", err: "no reqLinks entry with id LNK-NONE"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			find := linkA
			switch tc.key {
			case "version":
				find = pinA
			case "evidence":
				find = linkB
			case "fingerprint", "extra":
				find = pinB
			}
			if tc.name == "unknown item refused" {
				find = FindListItem([]string{"reqLinks"}, "id", "LNK-NONE")
			}
			got, err := SetKey([]byte(linkFile), find, tc.key, tc.raw)
			if tc.err != "" {
				if err == nil || !strings.Contains(err.Error(), tc.err) {
					t.Fatalf("err = %v, want %q", err, tc.err)
				}
				return
			}
			if err != nil || string(got) != tc.want {
				t.Fatalf("err %v\ngot\n%s\nwant\n%s", err, got, tc.want)
			}
		})
	}
}

// SetScalarInPlace replaces a single-line scalar keeping its quote style,
// in block and flow mappings, and refuses a multi-line value, a missing key
// and a value its quote style cannot hold.
// Mutation: dropping the quote style, accepting a value on another line or
// a quote inside a quoted value fails this test.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-MODEL-CHANGE
func TestSetScalarInPlace(t *testing.T) {
	in := "dependencies:\n  - {alias: a, version: 'v1.0.0'}\n  - alias: b\n    version: \"v1.0.0\" # pinned\n  - alias: c\n    version: v1.0.0\n  - alias: d\n    version:\n      v1.0.0\n"
	dep := func(alias string) func(string) []byte {
		return func(v string) []byte {
			out, err := SetScalarInPlace([]byte(in), FindListItem([]string{"dependencies"}, "alias", alias), "version", v)
			if err != nil {
				return []byte("ERR " + err.Error())
			}
			return out
		}
	}
	for name, tc := range map[string]struct {
		got  []byte
		want string
	}{
		"single quoted flow": {dep("a")("v1.1.0"), strings.Replace(in, "'v1.0.0'", "'v1.1.0'", 1)},
		"double quoted":      {dep("b")("v2.0.0"), strings.Replace(in, `"v1.0.0" # pinned`, `"v2.0.0" # pinned`, 1)},
		"plain":              {dep("c")("v3.0.0"), strings.Replace(in, "  - alias: c\n    version: v1.0.0\n", "  - alias: c\n    version: v3.0.0\n", 1)},
		"multi-line refused": {dep("d")("v4.0.0"), "ERR line 8: version is not a single-line scalar"},
		"quote refused":      {dep("a")("v1'0"), `ERR line 2: "v1'0" cannot be written single-quoted in place`},
		"missing alias":      {dep("e")("v1.0.0"), "ERR no dependencies entry with alias e"},
	} {
		if string(tc.got) != tc.want {
			t.Errorf("%s:\ngot\n%s\nwant\n%s", name, tc.got, tc.want)
		}
	}
}

// RemoveKey drops a key with its value lines, is no change for a missing
// key and refuses the first key of a list item and a flow mapping;
// AppendSequenceItems appends at the indentation of the existing items and
// turns an empty flow sequence into a block one; RemoveSequenceItems drops
// the chosen items with their lines and keeps the others byte for byte.
// Mutation: removing the key line only, appending at column 0 or removing
// the neighbouring item fails this test.
// TRLC-LINKS: REQ-EMG-104
// ENGMODEL-LINKS: FU-MODEL-CHANGE
func TestRemoveAndSequenceEdits(t *testing.T) {
	linkB := FindListItem([]string{"reqLinks"}, "id", "LNK-B")
	got, changed, err := RemoveKey([]byte(linkFile), linkB, "evidence")
	if want := strings.Replace(linkFile, "  evidence:\n  - REQ-A\n", "", 1); err != nil || !changed || string(got) != want {
		t.Fatalf("RemoveKey evidence: %v %v\n%s", err, changed, got)
	}
	if got, changed, err := RemoveKey([]byte(linkFile), linkB, "nope"); err != nil || changed || string(got) != linkFile {
		t.Fatalf("RemoveKey missing: %v %v", err, changed)
	}
	if _, _, err := RemoveKey([]byte(linkFile), linkB, "id"); err == nil || !strings.Contains(err.Error(), "first key of a list item") {
		t.Fatalf("RemoveKey id: %v", err)
	}
	if _, _, err := RemoveKey([]byte(linkFile), FindListItem([]string{"reqLinks"}, "id", "LNK-B", "pinned"), "fingerprint"); err == nil || !strings.Contains(err.Error(), "flow mapping") {
		t.Fatalf("RemoveKey flow: %v", err)
	}

	indented := "root:\n  items:\n    - id: A\n      x: 1\n    - id: B\n      x: 2\n  after: true\n"
	got, err = AppendSequenceItems([]byte(indented), FindPath("root"), "items", []string{"- id: C\n  x: 3\n"})
	if want := "root:\n  items:\n    - id: A\n      x: 1\n    - id: B\n      x: 2\n    - id: C\n      x: 3\n  after: true\n"; err != nil || string(got) != want {
		t.Fatalf("AppendSequenceItems:\n%s (%v)", got, err)
	}
	got, err = AppendSequenceItems([]byte("root:\n  items: []\n"), FindPath("root"), "items", []string{"- id: C\n"})
	if want := "root:\n  items:\n  - id: C\n"; err != nil || string(got) != want {
		t.Fatalf("AppendSequenceItems empty flow:\n%q (%v)", got, err)
	}
	got, err = RemoveSequenceItems([]byte(indented), FindPath("root"), "items", []int{0})
	if want := "root:\n  items:\n    - id: B\n      x: 2\n  after: true\n"; err != nil || string(got) != want {
		t.Fatalf("RemoveSequenceItems:\n%s (%v)", got, err)
	}
}

// Scalar encodes one line, plain when safe and quoted otherwise.
// TRLC-LINKS: REQ-EMG-101, REQ-EMG-104
// ENGMODEL-LINKS: FU-MODEL-CHANGE
func TestScalar(t *testing.T) {
	for in, want := range map[string]string{
		"plain text":      "plain text",
		"a: b":            "'a: b'",
		"  two\n lines  ": "two lines",
		"":                `""`,
		"2026-11-03":      `"2026-11-03"`,
	} {
		if got := Scalar(in); got != want {
			t.Errorf("Scalar(%q) = %s, want %s", in, got, want)
		}
	}
}
