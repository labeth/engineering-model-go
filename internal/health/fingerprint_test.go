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
	if bodyHashOf("f(\"http://a\") // c", "go", "f") == bodyHashOf("f(\"http:\")", "go", "f") {
		t.Error("a // inside a string must not start a comment")
	}
	if bodyHashOf("x = 1  # note\ny = 2", "python", "f") != bodyHashOf("x = 1\ny = 2  # other", "python", "f") {
		t.Error("python comments must be dropped")
	}
	if bodyHashOf("x = a // b", "python", "f") == bodyHashOf("x = a", "python", "f") {
		t.Error("// is floor division in python, not a comment")
	}
	if bodyHashOf("let x = 1; /* a /* nested */ b */ y()", "rust", "f") != bodyHashOf("let x = 1; y()", "rust", "f") {
		t.Error("rust block comments nest")
	}
	if bodyHashOf("fn f<'a>(x: &'a str, c: char) { let d = '}'; }", "rust", "f") != bodyHashOf("fn f<'a>(x:&'a str,c:char){let d='}';}", "rust", "f") {
		t.Error("rust lifetimes and chars must tokenize without swallowing the body")
	}
	if bodyHashOf("s = '''doc # not a comment'''\nz()", "python", "f") == bodyHashOf("s = '''doc'''\nz()", "python", "f") {
		t.Error("a python triple-quoted string keeps its text")
	}
	if got := BodyHash([]byte("a\nb\n"), Function{Name: "f", StartLine: 5, EndLine: 9}, "go"); got != "" {
		t.Errorf("an out-of-range function hashes to %q, want empty", got)
	}
}
