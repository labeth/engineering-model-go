// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

// Exported parse access for analyses that need the syntax tree itself (the code
// health engine). The scan discards its tree; these entry points hand the tree
// to a callback and release it afterwards, and re-export the test detection of
// testfunc.go so no caller duplicates it.

import (
	"fmt"
	"path/filepath"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tsgo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tsjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tspython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tsrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tstypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"
)

// Language names reported by LanguageForPath and accepted by ParseSource.
const (
	LanguageGo         = "go"
	LanguageRust       = "rust"
	LanguageTypeScript = "typescript"
	LanguageTSX        = "tsx"
	LanguageJavaScript = "javascript"
	LanguagePython     = "python"
)

// LanguageForPath maps a file extension to a language name: .go, .rs, .ts,
// .tsx, .js/.mjs/.cjs and .py. The second result is false for any other path.
// TRLC-LINKS: REQ-EMG-150
func LanguageForPath(path string) (string, bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".go":
		return LanguageGo, true
	case ".rs":
		return LanguageRust, true
	case ".ts":
		return LanguageTypeScript, true
	case ".tsx":
		return LanguageTSX, true
	case ".js", ".mjs", ".cjs":
		return LanguageJavaScript, true
	case ".py":
		return LanguagePython, true
	}
	return "", false
}

// TRLC-LINKS: REQ-EMG-150
func grammarFor(language string) (*sitter.Language, bool) {
	switch language {
	case LanguageGo:
		return sitter.NewLanguage(tsgo.Language()), true
	case LanguageRust:
		return sitter.NewLanguage(tsrust.Language()), true
	case LanguageTypeScript:
		return sitter.NewLanguage(tstypescript.LanguageTypescript()), true
	case LanguageTSX:
		return sitter.NewLanguage(tstypescript.LanguageTSX()), true
	case LanguageJavaScript:
		return sitter.NewLanguage(tsjavascript.Language()), true
	case LanguagePython:
		return sitter.NewLanguage(tspython.Language()), true
	}
	return nil, false
}

// ParseFile parses src with the grammar chosen by the extension of path and
// calls fn with the root node and the language name. The tree is closed when
// ParseFile returns, so fn must not retain any node. An unsupported extension
// is an error; the error of fn is returned unchanged.
// TRLC-LINKS: REQ-EMG-150
func ParseFile(path string, src []byte, fn func(root *sitter.Node, language string) error) error {
	language, ok := LanguageForPath(path)
	if !ok {
		return fmt.Errorf("parse %s: unsupported file extension", path)
	}
	return ParseSource(language, src, fn)
}

// ParseSource is ParseFile for a given language name (see the Language
// constants).
// TRLC-LINKS: REQ-EMG-150
func ParseSource(language string, src []byte, fn func(root *sitter.Node, language string) error) error {
	grammar, ok := grammarFor(language)
	if !ok {
		return fmt.Errorf("parse: unsupported language %q", language)
	}
	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(grammar); err != nil {
		return fmt.Errorf("set parser language %s: %w", language, err)
	}
	tree := parser.Parse(src, nil)
	if tree == nil {
		return fmt.Errorf("parse %s: tree-sitter produced no syntax tree", language)
	}
	defer tree.Close()
	root := tree.RootNode()
	if root == nil {
		return fmt.Errorf("parse %s: tree-sitter produced no syntax tree", language)
	}
	return fn(root, language)
}

// IsGoTestFile reports whether path is a Go _test.go file.
// TRLC-LINKS: REQ-EMG-150
func IsGoTestFile(path string) bool { return isGoTestFile(path) }

// IsPythonTestFile reports whether path is a pytest/unittest file: test_*.py or
// *_test.py.
// TRLC-LINKS: REQ-EMG-150
func IsPythonTestFile(path string) bool { return isPythonTestFile(path) }

// IsGoTestFunction reports whether a function_declaration named name in a
// _test.go file is a `go test` entry point (Test, Benchmark, Fuzz, Example).
// TRLC-LINKS: REQ-EMG-150
func IsGoTestFunction(n *sitter.Node, name string) bool { return goTestFunction(n, name) }

// IsPythonTestFunction reports whether a function_definition is a module-level
// test* function or a test* method directly in a class. The caller decides
// whether the file is a test file (IsPythonTestFile).
// TRLC-LINKS: REQ-EMG-150
func IsPythonTestFunction(n *sitter.Node, src []byte) bool { return pythonTestFunction(n, src) }

// IsRustTestFunction reports whether a function_item carries a test attribute
// (#[test], #[tokio::test], #[rstest], ...).
// TRLC-LINKS: REQ-EMG-150
func IsRustTestFunction(n *sitter.Node, src []byte) bool { return rustTestFunction(n, src) }

// IsRustInTestScope reports whether an item is compiled only for tests: it or an
// enclosing item carries #[cfg(test)], or an enclosing file or module has
// #![cfg(test)].
// TRLC-LINKS: REQ-EMG-150
func IsRustInTestScope(n *sitter.Node, src []byte) bool { return rustInTestScope(n, src) }
