// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
)

// Declaration is one named declaration of a source file as the syntax tree reads it, whether or not a trace marker
// is attached to it: what a reader needs to jump from a use to the definition.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type Declaration struct {
	Name string
	// Kind is function, method, type, var, const, class, struct, impl, module or declaration.
	Kind     string
	Language string
	// Path is relative to the scanned root, with forward slashes.
	Path string
	Line int
	// EndLine is the last line, or 0 when the language reader does not know it.
	EndLine   int
	Signature string
	// TestOnly marks code that is compiled or collected only for tests.
	TestOnly bool
}

// ListDeclarations returns every named declaration under root, in path and line order. It reads files only and
// follows the same skipping rules as ScanWithOptions.
// TRLC-LINKS: REQ-EMG-192
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func ListDeclarations(root string, opts ScanOptions) ([]Declaration, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, fmt.Errorf("resolve code root: %w", err)
	}
	w := &declarationWalk{root: absRoot, base: absRoot, opts: opts}
	if info, statErr := os.Stat(absRoot); statErr == nil && !info.IsDir() {
		w.base = filepath.Dir(absRoot)
	}
	if err := filepath.WalkDir(absRoot, w.visit); err != nil {
		return nil, fmt.Errorf("walk code root: %w", err)
	}
	return append([]Declaration{}, w.found...), nil
}

// declarationWalk is one walk of ListDeclarations; paths are made relative to base.
type declarationWalk struct {
	root, base string
	opts       ScanOptions
	found      []Declaration
}

// TRLC-LINKS: REQ-EMG-192
func (w *declarationWalk) visit(path string, d fs.DirEntry, walkErr error) error {
	switch {
	case walkErr != nil:
		return walkErr
	case d.IsDir() && w.skipsDirectory(path, d.Name()):
		return filepath.SkipDir
	case d.IsDir() || !w.reads(path):
		return nil
	}
	decls, err := declarationsOfFile(w.base, path)
	w.found = append(w.found, decls...)
	return err
}

// skipsDirectory is the scan's rule for a directory it does not enter: dependency caches, dot-directories and what
// the code scope leaves out.
// TRLC-LINKS: REQ-EMG-192
func (w *declarationWalk) skipsDirectory(path, name string) bool {
	if skipDirectory(name) {
		return true
	}
	return path != w.root && w.opts.Skip != nil && w.opts.Skip(path, true)
}

// reads reports whether the scan reads the file: a language it knows and not skipped by the code scope.
// TRLC-LINKS: REQ-EMG-192
func (w *declarationWalk) reads(path string) bool {
	if !supportedExt[strings.ToLower(filepath.Ext(path))] {
		return false
	}
	return w.opts.Skip == nil || !w.opts.Skip(path, false)
}

// declarationsOfFile reads one file and lists its declarations with a path relative to base.
// TRLC-LINKS: REQ-EMG-192
func declarationsOfFile(base, path string) ([]Declaration, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	rel, relErr := filepath.Rel(base, path)
	if relErr != nil {
		rel = path
	}
	return DeclarationsOf(filepath.ToSlash(rel), src)
}

// DeclarationsOf lists the declarations of one source file. path decides the language (its extension) and is
// recorded as given; an unsupported extension has no declarations.
// TRLC-LINKS: REQ-EMG-192
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func DeclarationsOf(path string, src []byte) ([]Declaration, error) {
	decls, _, _, err := extractDeclarations(path, src)
	if err != nil {
		return nil, err
	}
	language, _ := LanguageForPath(path)
	out := []Declaration{}
	for _, d := range markTestContext(path, decls) {
		out = append(out, publicDeclarations(d, language, path)...)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Name < out[j].Name
	})
	return out, nil
}

// publicDeclarations is the declaration and each further name of a grouped declaration.
// TRLC-LINKS: REQ-EMG-192
func publicDeclarations(d declaration, language, path string) []Declaration {
	base := Declaration{
		Name: d.Name, Kind: friendlyKind(d.Kind), Language: language, Path: path, Line: d.Line, EndLine: d.EndLine,
		Signature: strings.TrimSpace(d.Signature), TestOnly: d.TestOnly || d.IsTest,
	}
	out := []Declaration{base}
	for _, f := range d.Further {
		more := base
		more.Name, more.Line, more.EndLine = f.Name, f.Line, f.EndLine
		out = append(out, more)
	}
	return out
}

// skipDirectory is the rule of the scan for directories that are never read: dependency caches and dot-directories.
// TRLC-LINKS: REQ-EMG-192
func skipDirectory(name string) bool {
	if name == "node_modules" || name == "vendor" {
		return true
	}
	return strings.HasPrefix(name, ".") && len(name) > 1
}

// goSpecKinds maps a grouped Go declaration to the kinds of the specs inside it.
var goSpecKinds = map[string][]string{
	"type_declaration":  {"type_spec", "type_alias"},
	"var_declaration":   {"var_spec"},
	"const_declaration": {"const_spec"},
}

// goFurtherNames returns the names of a grouped Go declaration other than the first, which is the declaration's own.
// TRLC-LINKS: REQ-EMG-192
func goFurtherNames(n *sitter.Node, src []byte, first string) []declaredName {
	var out []declaredName
	for _, spec := range descendantsOfKind(n, goSpecKinds[n.Kind()]) {
		out = append(out, specFurtherNames(spec, src, first)...)
	}
	return out
}

// specFurtherNames are the names one spec declares, except the first name of the whole declaration.
// TRLC-LINKS: REQ-EMG-192
func specFurtherNames(spec *sitter.Node, src []byte, first string) []declaredName {
	var out []declaredName
	for _, id := range specNames(spec) {
		name := strings.TrimSpace(id.Utf8Text(src))
		if name != "" && name != "_" && name != first {
			out = append(out, declaredName{Name: name, Line: int(id.StartPosition().Row) + 1, EndLine: int(spec.EndPosition().Row) + 1})
		}
	}
	return out
}

// descendantsOfKind lists the nodes below n whose kind is one of kinds, outermost first, without looking inside a match.
// TRLC-LINKS: REQ-EMG-192
func descendantsOfKind(n *sitter.Node, kinds []string) []*sitter.Node {
	var out []*sitter.Node
	for i := uint(0); i < n.NamedChildCount(); i++ {
		c := n.NamedChild(i)
		if slices.Contains(kinds, c.Kind()) {
			out = append(out, c)
		} else {
			out = append(out, descendantsOfKind(c, kinds)...)
		}
	}
	return out
}

// specNames returns the identifiers a spec declares: they come first, and the first other child (a type, a value) ends them.
// TRLC-LINKS: REQ-EMG-192
func specNames(spec *sitter.Node) []*sitter.Node {
	var out []*sitter.Node
	for i := uint(0); i < spec.NamedChildCount(); i++ {
		id := spec.NamedChild(i)
		if id.Kind() != "identifier" && id.Kind() != "type_identifier" {
			break
		}
		out = append(out, id)
	}
	return out
}

var friendlyKinds = map[string]string{
	"function_declaration": "function", "function_item": "function", "function_definition": "function",
	"generator_function_declaration": "function", "method_declaration": "method", "method_definition": "method",
	"type_declaration": "type", "var_declaration": "var", "variable_declarator": "var", "const_declaration": "const",
	"class_declaration": "class", "class_definition": "class", "struct_item": "struct", "impl_item": "impl",
	"module_declaration": "module",
}

// friendlyKind names a syntax node kind for a reader; a kind it does not know is a plain declaration.
// TRLC-LINKS: REQ-EMG-192
func friendlyKind(kind string) string {
	if k, ok := friendlyKinds[kind]; ok {
		return k
	}
	return "declaration"
}

// declarationReader reads the declaration a syntax node makes and the comment lines of one file.
type declarationReader struct {
	ext, path string
	src       []byte
	spec      languageSpec
}

// noteComment records the lines a comment node covers.
// TRLC-LINKS: REQ-EMG-010
func (r declarationReader) noteComment(n *sitter.Node, lines map[int]bool) {
	if !r.isComment(n) {
		return
	}
	for line := int(n.StartPosition().Row) + 1; line <= int(n.EndPosition().Row)+1; line++ {
		lines[line] = true
	}
}

// isComment reports whether the node is a comment that has lines of its own (a Python comment after code does not).
// TRLC-LINKS: REQ-EMG-010
func (r declarationReader) isComment(n *sitter.Node) bool {
	if !n.IsNamed() || !strings.Contains(n.Kind(), "comment") {
		return false
	}
	return r.ext != ".py" || pythonStandaloneComment(n, r.src)
}

// read returns the declaration the node makes, with its last line, when it makes one.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-192
func (r declarationReader) read(n *sitter.Node) (declaration, bool) {
	var d declaration
	var ok bool
	switch {
	case isJavaScript(r.ext):
		d, ok = javascriptDeclaration(n, r.src)
	case r.ext == ".py":
		d, ok = r.python(n)
	case n.IsNamed() && r.spec.DeclarationKind[n.Kind()]:
		d, ok = r.named(n)
	}
	if ok {
		d.EndLine = int(n.EndPosition().Row) + 1
	}
	return d, ok
}

// TRLC-LINKS: REQ-EMG-010
func (r declarationReader) python(n *sitter.Node) (declaration, bool) {
	d, ok := pythonDeclaration(n, r.src)
	if ok && isPythonTestFile(r.path) {
		d.TestOnly = true
		d.IsTest = pythonTestFunction(n, r.src)
	}
	return d, ok
}

// named reads a Go, TypeScript or Rust declaration that has a name.
// TRLC-LINKS: REQ-EMG-010
func (r declarationReader) named(n *sitter.Node) (declaration, bool) {
	name := declarationName(n, r.src)
	if name == "" {
		return declaration{}, false
	}
	d := declaration{
		Name:         name,
		Kind:         n.Kind(),
		Line:         int(n.StartPosition().Row) + 1,
		Signature:    firstLine(strings.TrimSpace(n.Utf8Text(r.src))),
		RequiresTRLC: r.spec.TraceRequiredKind[n.Kind()],
	}
	switch r.ext {
	case ".go":
		r.goFacts(&d, n)
	case ".rs":
		d.IsTest = rustTestFunction(n, r.src)
		d.Benchmark = d.IsTest && rustBenchFunction(n, r.src)
		d.TestOnly = d.IsTest || rustInTestScope(n, r.src)
	}
	return d, true
}

// goFacts adds the other names of a grouped declaration and what a Go test file says about it.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-192
func (r declarationReader) goFacts(d *declaration, n *sitter.Node) {
	d.Further = goFurtherNames(n, r.src, d.Name)
	if !isGoTestFile(r.path) {
		return
	}
	d.TestOnly = true
	d.IsTest = goTestFunction(n, d.Name)
	if d.IsTest {
		d.Gates = goTestGates(n, r.src)
		d.Benchmark = strings.HasPrefix(d.Name, "Benchmark")
	}
}
