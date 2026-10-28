// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

// Test entry-point detection. A declaration is a test (Symbol.IsTest) only when
// the language's test runner would execute it; helpers, fakes and fixtures in
// test files are test-only code (Symbol.TestOnly) but not tests. JavaScript and
// TypeScript have no detection yet: their callers keep treating every linked
// declaration in a test file as verification.

import (
	"path/filepath"
	"strings"
	"unicode"
	"unicode/utf8"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// markTestContext marks declarations in Rust and Python files under a tests/
// directory as test-only, and drops the TRLC-LINKS requirement from test-only
// declarations that are not tests: a helper or fake verifies nothing, so its
// link is optional (and reported as verification.helper_link when present).
// Languages without test detection keep their requirement unchanged.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func markTestContext(relPath string, decls []declaration) []declaration {
	ext := strings.ToLower(filepath.Ext(relPath))
	inTestsDir := false
	for _, part := range strings.Split(filepath.ToSlash(relPath), "/") {
		if strings.EqualFold(part, "tests") {
			inTestsDir = true
		}
	}
	for i := range decls {
		if (ext == ".rs" || ext == ".py") && inTestsDir {
			decls[i].TestOnly = true
		}
		if decls[i].TestOnly && !decls[i].IsTest {
			decls[i].RequiresTRLC = false
		}
	}
	return decls
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func isGoTestFile(path string) bool {
	return strings.HasSuffix(strings.ToLower(filepath.Base(path)), "_test.go")
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func isPythonTestFile(path string) bool {
	base := strings.ToLower(filepath.Base(path))
	return strings.HasSuffix(base, ".py") && (strings.HasPrefix(base, "test_") || strings.HasSuffix(base, "_test.py"))
}

// goTestFunction reports whether a Go function declaration in a _test.go file is
// a `go test` entry point: TestX, BenchmarkX, FuzzX or ExampleX, where the
// suffix does not start with a lower-case letter. TestMain is a harness hook.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func goTestFunction(n *sitter.Node, name string) bool {
	if n == nil || n.Kind() != "function_declaration" || name == "TestMain" {
		return false
	}
	for _, prefix := range []string{"Test", "Benchmark", "Fuzz", "Example"} {
		if !strings.HasPrefix(name, prefix) {
			continue
		}
		rest := name[len(prefix):]
		if rest == "" {
			return true
		}
		r, _ := utf8.DecodeRuneInString(rest)
		return !unicode.IsLower(r)
	}
	return false
}

// pythonTestFunction reports whether a function is collected by pytest/unittest:
// a module-level `test*` function or a `test*` method defined directly in a class.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func pythonTestFunction(n *sitter.Node, src []byte) bool {
	if n == nil || n.Kind() != "function_definition" {
		return false
	}
	name := n.ChildByFieldName("name")
	if name == nil || !strings.HasPrefix(name.Utf8Text(src), "test") {
		return false
	}
	parent := n.Parent()
	if parent != nil && parent.Kind() == "decorated_definition" {
		parent = parent.Parent()
	}
	if parent == nil {
		return false
	}
	if parent.Kind() == "module" {
		return true
	}
	return parent.Kind() == "block" && parent.Parent() != nil && parent.Parent().Kind() == "class_definition"
}

// rustTestFunction reports whether a function_item carries a test attribute:
// #[test], #[rstest], #[test_case(..)], #[wasm_bindgen_test] or #[path::test]
// such as #[tokio::test].
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func rustTestFunction(n *sitter.Node, src []byte) bool {
	if n == nil || n.Kind() != "function_item" {
		return false
	}
	for _, attr := range rustOuterAttributes(n) {
		path := rustAttributePath(attr, src)
		switch {
		case path == "test", path == "rstest", path == "test_case", path == "wasm_bindgen_test":
			return true
		case strings.HasSuffix(path, "::test"):
			return true
		}
	}
	return false
}

// rustInTestScope reports whether a Rust item is compiled only for tests: it or an
// enclosing item (typically `mod tests`) carries `#[cfg(test)]`, or it sits in a
// file or module body with `#![cfg(test)]`.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func rustInTestScope(n *sitter.Node, src []byte) bool {
	for cur := n; cur != nil; cur = cur.Parent() {
		if strings.HasSuffix(cur.Kind(), "_item") && cur.Kind() != "attribute_item" {
			for _, attr := range rustOuterAttributes(cur) {
				if rustCfgTest(attr, src) {
					return true
				}
			}
		}
		if cur.Kind() == "source_file" || cur.Kind() == "declaration_list" {
			count := cur.NamedChildCount()
			for i := uint(0); i < count; i++ {
				child := cur.NamedChild(i)
				if child != nil && child.Kind() == "inner_attribute_item" && rustCfgTest(child, src) {
					return true
				}
			}
		}
	}
	return false
}

// rustOuterAttributes returns the attribute_item siblings directly preceding an item,
// skipping interleaved comments.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func rustOuterAttributes(n *sitter.Node) []*sitter.Node {
	out := []*sitter.Node{}
	for prev := n.PrevNamedSibling(); prev != nil; prev = prev.PrevNamedSibling() {
		switch {
		case prev.Kind() == "attribute_item":
			out = append(out, prev)
		case strings.Contains(prev.Kind(), "comment"):
		default:
			return out
		}
	}
	return out
}

// rustAttributePath returns the attribute path of an attribute item, e.g. "tokio::test".
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func rustAttributePath(item *sitter.Node, src []byte) string {
	attr := rustAttribute(item)
	if attr == nil || attr.NamedChildCount() == 0 {
		return ""
	}
	return strings.Join(strings.Fields(attr.NamedChild(0).Utf8Text(src)), "")
}

// rustCfgTest reports whether an attribute item is `cfg(test)` or `cfg(all(test, ..))`.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func rustCfgTest(item *sitter.Node, src []byte) bool {
	attr := rustAttribute(item)
	if attr == nil || rustAttributePath(item, src) != "cfg" {
		return false
	}
	args := attr.ChildByFieldName("arguments")
	if args == nil {
		return false
	}
	text := strings.Join(strings.Fields(args.Utf8Text(src)), "")
	if text == "(test)" {
		return true
	}
	if !strings.HasPrefix(text, "(all(") {
		return false
	}
	for _, part := range strings.FieldsFunc(strings.TrimSuffix(strings.TrimPrefix(text, "(all("), "))"), func(r rune) bool { return r == ',' }) {
		if part == "test" {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-010, REQ-EMG-061
func rustAttribute(item *sitter.Node) *sitter.Node {
	if item == nil {
		return nil
	}
	count := item.NamedChildCount()
	for i := uint(0); i < count; i++ {
		if child := item.NamedChild(i); child != nil && child.Kind() == "attribute" {
			return child
		}
	}
	return nil
}
