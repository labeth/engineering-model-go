// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"errors"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TRLC-LINKS: REQ-EMG-150
func TestLanguageForPath(t *testing.T) {
	for path, want := range map[string]string{
		"a.go": "go", "dir/a.RS": "rust", "a.ts": "typescript", "a.tsx": "tsx",
		"a.js": "javascript", "a.mjs": "javascript", "a.cjs": "javascript", "a.py": "python",
	} {
		if got, ok := LanguageForPath(path); !ok || got != want {
			t.Errorf("LanguageForPath(%q) = %q, %v; want %q", path, got, ok, want)
		}
	}
	for _, path := range []string{"a.v", "a.txt", "Makefile", "a.go.txt"} {
		if got, ok := LanguageForPath(path); ok {
			t.Errorf("LanguageForPath(%q) = %q, want unsupported", path, got)
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestParseFileHandsOverTheTree(t *testing.T) {
	for path, tc := range map[string]struct{ src, root, language string }{
		"a.go":  {"package p\n", "source_file", "go"},
		"a.rs":  {"fn a() {}\n", "source_file", "rust"},
		"a.ts":  {"let a: number = 1;\n", "program", "typescript"},
		"a.tsx": {"const a = <b/>;\n", "program", "tsx"},
		"a.js":  {"let a = 1;\n", "program", "javascript"},
		"a.py":  {"a = 1\n", "module", "python"},
	} {
		called := false
		err := ParseFile(path, []byte(tc.src), func(root *sitter.Node, language string) error {
			called = true
			if root.Kind() != tc.root || language != tc.language {
				t.Errorf("%s: root %q language %q, want %q %q", path, root.Kind(), language, tc.root, tc.language)
			}
			if root.HasError() {
				t.Errorf("%s: unexpected syntax error", path)
			}
			return nil
		})
		if err != nil || !called {
			t.Errorf("%s: err %v called %v", path, err, called)
		}
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestParseFileErrors(t *testing.T) {
	noop := func(*sitter.Node, string) error { return nil }
	if err := ParseFile("a.txt", nil, noop); err == nil {
		t.Error("an unsupported extension is an error")
	}
	if err := ParseSource("cobol", nil, noop); err == nil {
		t.Error("an unsupported language is an error")
	}
	sentinel := errors.New("stop")
	if err := ParseFile("a.go", []byte("package p\n"), func(*sitter.Node, string) error { return sentinel }); !errors.Is(err, sentinel) {
		t.Errorf("the callback error is returned unchanged, got %v", err)
	}
	// Syntax errors do not fail the parse: the tree is handed over with its error flag.
	err := ParseFile("a.go", []byte("package p\nfunc ("), func(root *sitter.Node, _ string) error {
		if !root.HasError() {
			t.Error("expected a syntax error flag")
		}
		return nil
	})
	if err != nil {
		t.Errorf("a syntax error is not a parse failure: %v", err)
	}
}

// withFunctions parses src and calls check with the root and the nodes of the given kind
// in source order. The tree is only valid inside the
// callback, so check runs there.
func withFunctions(t *testing.T, path, src, kind string, check func(root *sitter.Node, fns []*sitter.Node)) {
	t.Helper()
	err := ParseFile(path, []byte(src), func(root *sitter.Node, _ string) error {
		var fns []*sitter.Node
		var visit func(n *sitter.Node)
		visit = func(n *sitter.Node) {
			if n.Kind() == kind {
				fns = append(fns, n)
			}
			for i := uint(0); i < n.NamedChildCount(); i++ {
				visit(n.NamedChild(i))
			}
		}
		visit(root)
		check(root, fns)
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

// TRLC-LINKS: REQ-EMG-150
func TestExportedTestDetection(t *testing.T) {
	if !IsGoTestFile("pkg/a_test.go") || IsGoTestFile("pkg/a.go") {
		t.Error("IsGoTestFile")
	}
	if !IsPythonTestFile("test_a.py") || !IsPythonTestFile("a_test.py") || IsPythonTestFile("a.py") {
		t.Error("IsPythonTestFile")
	}

	goSrc := "package p\nfunc TestA() {}\nfunc Testa() {}\nfunc TestMain() {}\nfunc helper() {}\nfunc BenchmarkB() {}\n"
	withFunctions(t, "a_test.go", goSrc, "function_declaration", func(_ *sitter.Node, fns []*sitter.Node) {
		want := map[string]bool{"TestA": true, "Testa": false, "TestMain": false, "helper": false, "BenchmarkB": true}
		for _, fn := range fns {
			name := fn.ChildByFieldName("name").Utf8Text([]byte(goSrc))
			if got := IsGoTestFunction(fn, name); got != want[name] {
				t.Errorf("IsGoTestFunction(%s) = %v", name, got)
			}
		}
	})

	pySrc := "def test_a():\n    pass\n\ndef helper():\n    pass\n\nclass T:\n    def test_b(self):\n        pass\n    def other(self):\n        pass\n"
	withFunctions(t, "test_a.py", pySrc, "function_definition", func(_ *sitter.Node, fns []*sitter.Node) {
		want := map[string]bool{"test_a": true, "helper": false, "test_b": true, "other": false}
		for _, fn := range fns {
			name := fn.ChildByFieldName("name").Utf8Text([]byte(pySrc))
			if got := IsPythonTestFunction(fn, []byte(pySrc)); got != want[name] {
				t.Errorf("IsPythonTestFunction(%s) = %v", name, got)
			}
		}
	})

	rsSrc := "fn prod() {}\n#[cfg(test)]\nmod tests {\n    #[test]\n    fn a() {}\n    #[tokio::test]\n    async fn b() {}\n    fn helper() {}\n}\n"
	withFunctions(t, "lib.rs", rsSrc, "function_item", func(_ *sitter.Node, fns []*sitter.Node) {
		src := []byte(rsSrc)
		want := map[string][2]bool{"prod": {false, false}, "a": {true, true}, "b": {true, true}, "helper": {false, true}}
		for _, fn := range fns {
			name := fn.ChildByFieldName("name").Utf8Text(src)
			if got := [2]bool{IsRustTestFunction(fn, src), IsRustInTestScope(fn, src)}; got != want[name] {
				t.Errorf("rust %s: test %v scope %v, want %v", name, got[0], got[1], want[name])
			}
		}
	})
	inner := "#![cfg(test)]\nfn a() {}\n"
	withFunctions(t, "lib.rs", inner, "source_file", func(root *sitter.Node, _ []*sitter.Node) {
		if !IsRustInTestScope(root, []byte(inner)) {
			t.Error("#![cfg(test)] marks the file as test scope")
		}
	})
}
