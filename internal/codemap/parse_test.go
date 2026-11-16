// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"errors"
	"slices"
	"testing"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// TRLC-LINKS: REQ-EMG-150
func TestLanguageForPath(t *testing.T) {
	supported := map[string]string{
		"a.go": "go", "dir/a.RS": "rust", "a.ts": "typescript", "a.tsx": "tsx",
		"a.js": "javascript", "a.mjs": "javascript", "a.cjs": "javascript", "a.py": "python",
	}
	got := map[string]string{}
	for path := range supported {
		got[path], _ = LanguageForPath(path)
	}
	requireEqual(t, got, supported)
	unsupported := slices.ContainsFunc([]string{"a.v", "a.txt", "Makefile", "a.go.txt"}, func(path string) bool {
		_, ok := LanguageForPath(path)
		return ok
	})
	requireEqual(t, unsupported, false)
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

// parseCase is a source to parse and the kind of syntax node to collect.
type parseCase struct {
	path, src, kind string
}

// withFunctions parses the case and calls check with the root and the nodes of
// its kind in source order. The tree is only valid inside the callback, so
// check runs there.
func withFunctions(t *testing.T, c parseCase, check func(root *sitter.Node, fns []*sitter.Node)) {
	t.Helper()
	err := ParseFile(c.path, []byte(c.src), func(root *sitter.Node, _ string) error {
		var fns []*sitter.Node
		var visit func(n *sitter.Node)
		visit = func(n *sitter.Node) {
			if n.Kind() == c.kind {
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
	requireEqual(t, []bool{IsGoTestFile("pkg/a_test.go"), IsGoTestFile("pkg/a.go")}, []bool{true, false})
	requireEqual(t, []bool{IsPythonTestFile("test_a.py"), IsPythonTestFile("a_test.py"), IsPythonTestFile("a.py")}, []bool{true, true, false})
	t.Run("go", checkGoTestDetection)
	t.Run("python", checkPythonTestDetection)
	t.Run("rust", checkRustTestDetection)
}

// checkGoTestDetection checks IsGoTestFunction on Go test entry points and helpers.
func checkGoTestDetection(t *testing.T) {
	src := "package p\nfunc TestA() {}\nfunc Testa() {}\nfunc TestMain() {}\nfunc helper() {}\nfunc BenchmarkB() {}\n"
	got := map[string]bool{}
	withFunctions(t, parseCase{"a_test.go", src, "function_declaration"}, func(_ *sitter.Node, fns []*sitter.Node) {
		for _, fn := range fns {
			name := fn.ChildByFieldName("name").Utf8Text([]byte(src))
			got[name] = IsGoTestFunction(fn, name)
		}
	})
	requireEqual(t, got, map[string]bool{"TestA": true, "Testa": false, "TestMain": false, "helper": false, "BenchmarkB": true})
}

// checkPythonTestDetection checks IsPythonTestFunction on functions and methods.
func checkPythonTestDetection(t *testing.T) {
	src := "def test_a():\n    pass\n\ndef helper():\n    pass\n\nclass T:\n    def test_b(self):\n        pass\n    def other(self):\n        pass\n"
	got := map[string]bool{}
	withFunctions(t, parseCase{"test_a.py", src, "function_definition"}, func(_ *sitter.Node, fns []*sitter.Node) {
		for _, fn := range fns {
			got[fn.ChildByFieldName("name").Utf8Text([]byte(src))] = IsPythonTestFunction(fn, []byte(src))
		}
	})
	requireEqual(t, got, map[string]bool{"test_a": true, "helper": false, "test_b": true, "other": false})
}

// checkRustTestDetection checks IsRustTestFunction and IsRustInTestScope.
func checkRustTestDetection(t *testing.T) {
	src := "fn prod() {}\n#[cfg(test)]\nmod tests {\n    #[test]\n    fn a() {}\n    #[tokio::test]\n    async fn b() {}\n    fn helper() {}\n}\n"
	got := map[string][2]bool{}
	withFunctions(t, parseCase{"lib.rs", src, "function_item"}, func(_ *sitter.Node, fns []*sitter.Node) {
		for _, fn := range fns {
			got[fn.ChildByFieldName("name").Utf8Text([]byte(src))] = [2]bool{IsRustTestFunction(fn, []byte(src)), IsRustInTestScope(fn, []byte(src))}
		}
	})
	requireEqual(t, got, map[string][2]bool{"prod": {false, false}, "a": {true, true}, "b": {true, true}, "helper": {false, true}})
	inner := "#![cfg(test)]\nfn a() {}\n"
	withFunctions(t, parseCase{"lib.rs", inner, "source_file"}, func(root *sitter.Node, _ []*sitter.Node) {
		requireEqual(t, IsRustInTestScope(root, []byte(inner)), true)
	})
}
