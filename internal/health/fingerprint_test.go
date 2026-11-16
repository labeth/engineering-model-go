// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import "testing"

// hashOf hashes the whole text as one function.
//
// TRLC-LINKS: REQ-EMG-156
func bodyHashOf(src, lang, name string) string {
	lines := 1
	for _, r := range src {
		if r == '\n' {
			lines++
		}
	}
	return BodyHash([]byte(src), Function{Name: name, StartLine: 1, EndLine: lines}, lang)
}

// The hash ignores white space, comments, line endings and the function's own
// name, and notices a change of any real token.
// Mutation: hashing the raw text fails the equal cases; ignoring string
// content fails the last case.
//
// TRLC-LINKS: REQ-EMG-156
func TestBodyHashNormalization(t *testing.T) {
	base := "func Old(a int) int {\n\tif a > 1 { // big\n\t\treturn Old(a - 1)\n\t}\n\treturn a\n}\n"
	cases := []struct {
		name, src, lang, fn string
		same                bool
	}{
		{"whitespace and comments", "func Old(a int) int { /* x */ if a>1 {\n return Old(a-1) }\n\n return a }", "go", "Old", true},
		{"crlf", "func Old(a int) int {\r\n\tif a > 1 { // big\r\n\t\treturn Old(a - 1)\r\n\t}\r\n\treturn a\r\n}\r\n", "go", "Old", true},
		{"renamed with recursion", "func Fresh(a int) int {\n\tif a > 1 {\n\t\treturn Fresh(a - 1)\n\t}\n\treturn a\n}\n", "go", "Fresh", true},
		{"method of another type, same name part", "func Old(a int) int {\n\tif a > 1 {\n\t\treturn Old(a - 1)\n\t}\n\treturn a\n}", "go", "T.Old", true},
		{"changed operator", "func Old(a int) int {\n\tif a >= 1 {\n\t\treturn Old(a - 1)\n\t}\n\treturn a\n}\n", "go", "Old", false},
		{"extra statement", "func Old(a int) int {\n\tif a > 1 {\n\t\treturn Old(a - 1)\n\t}\n\ta++\n\treturn a\n}\n", "go", "Old", false},
		{"changed string", "func Old(a int) int {\n\tprintln(\"x\")\n\treturn a\n}\n", "go", "Old", false},
	}
	want := bodyHashOf(base, "go", "Old")
	for _, tc := range cases {
		got := bodyHashOf(tc.src, tc.lang, tc.fn)
		if (got == want) != tc.same {
			t.Errorf("%s: same=%v, want %v", tc.name, got == want, tc.same)
		}
	}
	a := bodyHashOf("func Old(a int) int {\n\tprintln(\"x\")\n\treturn a\n}\n", "go", "Old")
	b := bodyHashOf("func Old(a int) int {\n\tprintln(\"y\")\n\treturn a\n}\n", "go", "Old")
	if a == b {
		t.Error("a changed string literal must change the hash")
	}
}

// A comment marker inside a string is not a comment, and each language has its
// own comment syntax.
//
// TRLC-LINKS: REQ-EMG-156
func TestBodyHashLanguages(t *testing.T) {
	differ := func(a, b, lang string) bool { return bodyHashOf(a, lang, "f") != bodyHashOf(b, lang, "f") }
	wantEqual(t, differ("f(\"http://a\") // c", "f(\"http:\")", "go"), true)                       // a // inside a string does not start a comment
	wantEqual(t, differ("x = 1  # note\ny = 2", "x = 1\ny = 2  # other", "python"), false)         // python comments are dropped
	wantEqual(t, differ("x = a // b", "x = a", "python"), true)                                    // // is floor division in python
	wantEqual(t, differ("let x = 1; /* a /* nested */ b */ y()", "let x = 1; y()", "rust"), false) // rust block comments nest
	wantEqual(t, differ("fn f<'a>(x: &'a str, c: char) { let d = '}'; }", "fn f<'a>(x:&'a str,c:char){let d='}';}", "rust"), false)
	wantEqual(t, differ("s = '''doc # not a comment'''\nz()", "s = '''doc'''\nz()", "python"), true)  // a triple-quoted string keeps its text
	wantEqual(t, BodyHash([]byte("a\nb\n"), Function{Name: "f", StartLine: 5, EndLine: 9}, "go"), "") // an out-of-range function hashes to nothing
}
