// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import "testing"

// Text from a dependency's model is rendered without AsciiDoc passthrough:
// +++…+++, ++…++, $$…$$ and pass macros are broken with {empty}, so no raw
// HTML reaches the document; whitespace is collapsed and other text kept.
// Mutation: dropping a replacement of escapePassthrough, or not applying it
// in oneLineText, fails this test.
// TRLC-LINKS: REQ-EMG-110
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
func TestOneLineTextEscapesPassthrough(t *testing.T) {
	for in, want := range map[string]string{
		"The encoder shall\n publish  one sentence.": "The encoder shall publish one sentence.",
		"+++<script>x</script>+++":                   "+{empty}+{empty}+<script>x</script>+{empty}+{empty}+",
		"a ++<b>++ c":                                "a +{empty}+<b>+{empty}+ c",
		"pass:[<i>raw</i>] and pass:q[x]":            "pass{empty}:[<i>raw</i>] and pass{empty}:q[x]",
		"$$<u>$$":                                    "${empty}$<u>${empty}$",
		"one + two = three":                          "one + two = three",
	} {
		if got := oneLineText(in); got != want {
			t.Errorf("oneLineText(%q) = %q, want %q", in, got, want)
		}
	}
}
