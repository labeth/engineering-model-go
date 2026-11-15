// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codemap

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"slices"
	"sort"
	"strings"
	"unsafe"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tsgo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tsjavascript "github.com/tree-sitter/tree-sitter-javascript/bindings/go"
	tspython "github.com/tree-sitter/tree-sitter-python/bindings/go"
	tsrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tstypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type Symbol struct {
	TraceID    string
	PartOf     []string
	Implements []string
	ModelLinks []string
	Path       string
	Line       int
	Signature  string
	// IsTest marks a test entry point the language's runner executes (Go TestX/
	// BenchmarkX/FuzzX/ExampleX in _test.go, Rust #[test]-like functions, Python
	// test* functions in test files). Only these verify the requirements they link.
	IsTest bool
	// TestOnly marks code compiled or collected only for tests: every declaration
	// in a Go _test.go or Python test file and Rust items under #[cfg(test)]. It
	// never counts as production implementation.
	TestOnly bool
	// Gates names what an environment-gated Go test waits for: each
	// variable read by os.Getenv or os.LookupEnv, or testing.Short, in an if
	// among its first statements whose body calls t.Skip, t.Skipf or
	// t.SkipNow. It is empty for an ungated test and for non-tests.
	Gates []string
	// Benchmark marks a test entry point that is a benchmark (Go BenchmarkX,
	// Rust #[bench]): it measures and asserts nothing by itself.
	Benchmark bool
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type pendingTrace struct {
	traceID    string
	partOf     []string
	implements []string
	modelLinks []string
	firstLine  int
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type declaration struct {
	Name         string
	Kind         string
	Line         int
	Signature    string
	RequiresTRLC bool
	LeadingLine  int
	IsTest       bool
	TestOnly     bool
	Gates        []string
	Benchmark    bool
	// EndLine is the last line of the declaration (0 when the language reader does not know it).
	EndLine int
	// Further holds the other names a grouped Go declaration (type, var or const block) declares, each at its own line.
	Further []declaredName
}

// declaredName is one more name that a grouped declaration declares.
type declaredName struct {
	Name    string
	Line    int
	EndLine int
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type languageSpec struct {
	Language          *sitter.Language
	DeclarationKind   map[string]bool
	TraceRequiredKind map[string]bool
}

var supportedExt = map[string]bool{
	".js":  true,
	".mjs": true,
	".cjs": true,
	".go":  true,
	".ts":  true,
	".tsx": true,
	".rs":  true,
	".v":   true,
	".py":  true,
}

// ScanOptions controls identifier derivation and path filtering for ScanWithOptions.
// IDBase is the directory symbol identifiers are made relative to (the module root);
// it defaults to the scan root. Skip, when set, excludes files and directories.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type ScanOptions struct {
	IDBase string
	Skip   func(path string, isDir bool) bool
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func Scan(root string) ([]Symbol, []validate.Diagnostic, error) {
	return ScanWithOptions(root, ScanOptions{})
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-057, REQ-EMG-058, REQ-EMG-084
func ScanWithOptions(root string, opts ScanOptions) ([]Symbol, []validate.Diagnostic, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve code root: %w", err)
	}
	walker := &codeWalker{absRoot: absRoot, opts: opts, scanBase: scanBaseOf(absRoot)}
	walker.idBase = walker.scanBase
	if strings.TrimSpace(opts.IDBase) != "" {
		if walker.idBase, err = filepath.Abs(opts.IDBase); err != nil {
			return nil, nil, fmt.Errorf("resolve identifier base: %w", err)
		}
	}
	walker.symbols, walker.diags = []Symbol{}, []validate.Diagnostic{}
	if err := filepath.WalkDir(absRoot, walker.visit); err != nil {
		return nil, nil, fmt.Errorf("walk code root: %w", err)
	}
	sortSymbols(walker.symbols)
	return walker.symbols, validate.SortDiagnostics(walker.diags), nil
}

// scanBaseOf is the directory a scan root's paths are made relative to: the
// root itself, or its directory when the root is a single file.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-057, REQ-EMG-058, REQ-EMG-084
func scanBaseOf(absRoot string) string {
	if info, err := os.Stat(absRoot); err == nil && !info.IsDir() {
		return filepath.Dir(absRoot)
	}
	return absRoot
}

// sortSymbols orders symbols by path, line and trace identifier.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-057, REQ-EMG-058, REQ-EMG-084
func sortSymbols(symbols []Symbol) {
	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].Path != symbols[j].Path {
			return symbols[i].Path < symbols[j].Path
		}
		if symbols[i].Line != symbols[j].Line {
			return symbols[i].Line < symbols[j].Line
		}
		return symbols[i].TraceID < symbols[j].TraceID
	})
}

// codeWalker collects the symbols and diagnostics of every scanned file under
// a code root.
type codeWalker struct {
	absRoot  string
	scanBase string
	idBase   string
	opts     ScanOptions
	symbols  []Symbol
	diags    []validate.Diagnostic
}

// fileScanner scans one file into symbols and diagnostics.
type fileScanner func(root, idBase, path string) ([]Symbol, []validate.Diagnostic, error)

// visit is the filepath.WalkDirFunc of a scan.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-057, REQ-EMG-058, REQ-EMG-084
func (w *codeWalker) visit(path string, d fs.DirEntry, walkErr error) error {
	if walkErr != nil {
		return walkErr
	}
	if d.IsDir() {
		return w.visitDirectory(path, d.Name())
	}
	scan := w.scannerFor(path)
	if scan == nil {
		return nil
	}
	fileSymbols, fileDiags, err := scan(w.scanBase, w.idBase, path)
	if err != nil {
		return err
	}
	w.symbols = append(w.symbols, fileSymbols...)
	w.diags = append(w.diags, fileDiags...)
	return nil
}

// visitDirectory skips dependency caches and dot-directories (tool caches,
// .engmod) so the scan reflects only committed source and is reproducible
// across environments, and any directory the options skip.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-057, REQ-EMG-058, REQ-EMG-084
func (w *codeWalker) visitDirectory(path, name string) error {
	if skipDirectory(name) {
		return filepath.SkipDir
	}
	if path != w.absRoot && w.opts.Skip != nil && w.opts.Skip(path, true) {
		return filepath.SkipDir
	}
	return nil
}

// scannerFor returns the scanner of a file, or nil when the scan skips it.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-057, REQ-EMG-058, REQ-EMG-084
func (w *codeWalker) scannerFor(path string) fileScanner {
	artifact := IsArtifactFile(path)
	if !supportedExt[strings.ToLower(filepath.Ext(path))] && !artifact {
		return nil
	}
	if w.opts.Skip != nil && w.opts.Skip(path, false) {
		return nil
	}
	if !artifact {
		return scanFile
	}
	if !ReadableArtifact(path) {
		return nil
	}
	return scanArtifactFile
}

// readScanFile reads a scanned file and returns its scan-relative path and
// its module-relative identifier path.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-084
func readScanFile(root, idBase, path string) (src []byte, relPath, idPath string, err error) {
	src, err = os.ReadFile(path)
	if err != nil {
		return nil, "", "", fmt.Errorf("read %s: %w", path, err)
	}
	relPath, err = filepath.Rel(root, path)
	if err != nil {
		relPath = path
	}
	relPath = filepath.ToSlash(relPath)
	// Identifiers are module-relative; files outside the identifier base keep scan-relative ones.
	idPath = relPath
	if rel, relErr := filepath.Rel(idBase, path); relErr == nil && !strings.HasPrefix(filepath.ToSlash(rel), "../") {
		idPath = filepath.ToSlash(rel)
	}
	return src, relPath, idPath, nil
}

// scanArtifactFile scans a container build file or Markdown document for
// file-level trace markers.
// TRLC-LINKS: REQ-EMG-084
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func scanArtifactFile(root, idBase, path string) ([]Symbol, []validate.Diagnostic, error) {
	src, relPath, idPath, err := readScanFile(root, idBase, path)
	if err != nil {
		return nil, nil, err
	}
	symbols, diags := scanArtifact(relPath, idPath, src)
	return symbols, diags, nil
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func scanFile(root, idBase, path string) ([]Symbol, []validate.Diagnostic, error) {
	src, relPath, idPath, err := readScanFile(root, idBase, path)
	if err != nil {
		return nil, nil, err
	}
	decls, commentLines, parseDiags, err := extractDeclarations(path, src)
	if err != nil {
		return nil, nil, err
	}
	scan := newFileScan(symbolSite{relPath: relPath, idPath: idPath}, markTestContext(relPath, decls), commentLines)
	scan.diags = append([]validate.Diagnostic(nil), parseDiags...)
	if err := scan.scanLines(src); err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", path, err)
	}
	scan.finish(path)
	return scan.symbols, scan.diags, nil
}

// symbolSite is where a symbol is declared: the scan-relative path, the
// module-relative identifier path, the line number and its trimmed text.
type symbolSite struct {
	relPath string
	idPath  string
	lineNo  int
	line    string
}

// fileScan is the state of scanning one file's lines for trace markers.
type fileScan struct {
	site                symbolSite
	commentLines        map[int]bool
	declByLine          map[int]declaration
	traceRequiredByLine map[int]declaration
	leadingLines        map[int]bool
	linkedLines         map[int]bool
	pending             pendingTrace
	symbols             []Symbol
	diags               []validate.Diagnostic
}

// newFileScan indexes a file's declarations by line for the marker scan.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func newFileScan(site symbolSite, decls []declaration, commentLines map[int]bool) *fileScan {
	scan := &fileScan{
		site:                site,
		commentLines:        commentLines,
		declByLine:          map[int]declaration{},
		traceRequiredByLine: map[int]declaration{},
		leadingLines:        map[int]bool{},
		linkedLines:         map[int]bool{},
		symbols:             []Symbol{},
	}
	for _, d := range decls {
		scan.index(d)
	}
	return scan
}

// index records one declaration under its line.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) index(d declaration) {
	for line := d.LeadingLine; line > 0 && line < d.Line; line++ {
		s.leadingLines[line] = true
	}
	if _, exists := s.declByLine[d.Line]; !exists {
		s.declByLine[d.Line] = d
	}
	if d.RequiresTRLC {
		s.traceRequiredByLine[d.Line] = d
	}
}

// scanLines reads the source line by line.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) scanLines(src []byte) error {
	scanner := bufio.NewScanner(strings.NewReader(string(src)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		s.scanLine(lineNo, strings.TrimSpace(scanner.Text()))
	}
	return scanner.Err()
}

// scanLine handles one trimmed line: a marker line, the declaration a marker
// block attaches to, a line allowed between them, or the end of an unattached
// marker block.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) scanLine(lineNo int, trimmed string) {
	if s.commentLines[lineNo] && isCommentLike(trimmed) && s.readMarker(lineNo, trimmed) {
		return
	}
	if s.pending.firstLine == 0 {
		return
	}
	s.continueBlock(lineNo, trimmed)
}

// continueBlock handles a line that is not a marker while a marker block is
// pending.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) continueBlock(lineNo int, trimmed string) {
	if d, ok := s.declByLine[lineNo]; ok {
		s.attach(d, symbolSite{relPath: s.site.relPath, idPath: s.site.idPath, lineNo: lineNo, line: trimmed})
		return
	}
	// Allow blank lines, comments, and annotation/decorator lines between marker and declaration.
	if isFiller(trimmed) || s.leadingLines[lineNo] {
		return
	}
	s.reportUnattached()
	s.pending = pendingTrace{}
}

// isFiller reports whether a line may sit between a marker block and its
// declaration.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func isFiller(trimmed string) bool {
	return trimmed == "" || isCommentLike(trimmed) || isAttributeLike(trimmed)
}

// readMarker records a TRACE-ID, TRACE-PART-OF, TRLC-LINKS or ENGMODEL-LINKS
// line in the pending block and reports whether the line was one.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) readMarker(lineNo int, trimmed string) bool {
	if v, ok := markerValue(trimmed, "TRACE-ID:"); ok {
		s.pending.traceID = strings.TrimSpace(v)
	} else if v, ok := markerValue(trimmed, "TRACE-PART-OF:"); ok {
		s.pending.partOf = splitCSV(v)
	} else if v, ok := markerValue(trimmed, "TRLC-LINKS:"); ok {
		s.readTRLCLinks(lineNo, v)
	} else if v, ok := markerValue(trimmed, "ENGMODEL-LINKS:"); ok {
		// Stacked ENGMODEL-LINKS lines above one declaration accumulate (REQ-EMG-174).
		s.pending.modelLinks = appendUnique(s.pending.modelLinks, splitCSV(v))
	} else {
		return false
	}
	if s.pending.firstLine == 0 {
		s.pending.firstLine = lineNo
	}
	return true
}

// readTRLCLinks accumulates the requirement ids of a TRLC-LINKS line and
// reports each that is not a requirement id.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) readTRLCLinks(lineNo int, value string) {
	links := splitCSV(value)
	// Stacked TRLC-LINKS lines above one declaration accumulate.
	s.pending.implements = append(s.pending.implements, links...)
	for _, link := range links {
		if !validRequirementID(link) {
			s.diags = append(s.diags, validate.Diagnostic{
				Code:     "code.invalid_trlc_link",
				Severity: validate.SeverityError,
				Message:  fmt.Sprintf("TRLC-LINKS value %q is not a requirement id", link),
				Path:     fmt.Sprintf("%s:%d", s.site.relPath, lineNo),
			})
		}
	}
}

// attach turns the pending marker block into the symbol of the declaration
// at site.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) attach(d declaration, site symbolSite) {
	s.symbols = append(s.symbols, symbolForDeclaration(d, site, s.pending))
	if len(s.pending.implements) > 0 {
		s.linkedLines[site.lineNo] = true
	}
	s.pending = pendingTrace{}
}

// reportUnattached reports the pending marker block as attached to no
// declaration.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) reportUnattached() {
	s.diags = append(s.diags, validate.Diagnostic{
		Code:     "code.trace_unattached",
		Severity: validate.SeverityWarning,
		Message:  "trace marker block not attached to any declaration",
		Path:     fmt.Sprintf("%s:%d", s.site.relPath, s.pending.firstLine),
	})
}

// finish reports a marker block left open at the end of the file and the
// functions that carry no TRLC-LINKS.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (s *fileScan) finish(path string) {
	if s.pending.firstLine != 0 {
		s.reportUnattached()
	}
	missingLines := make([]int, 0, len(s.traceRequiredByLine))
	for line := range s.traceRequiredByLine {
		if !s.linkedLines[line] {
			missingLines = append(missingLines, line)
		}
	}
	sort.Ints(missingLines)
	if len(missingLines) == 0 {
		return
	}
	lineList := joinLineNumbers(missingLines)
	label := "functions"
	if strings.EqualFold(filepath.Ext(path), ".v") {
		label = "modules"
	}
	s.diags = append(s.diags, validate.Diagnostic{
		Code:     "code.missing_trlc_link",
		Severity: validate.SeverityError,
		Message:  fmt.Sprintf("%s missing TRLC-LINKS at lines %s", label, strings.ReplaceAll(lineList, ",", ", ")),
		Path:     fmt.Sprintf("%s:%s", s.site.relPath, lineList),
	})
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func symbolForDeclaration(d declaration, site symbolSite, pending pendingTrace) Symbol {
	traceID := strings.TrimSpace(pending.traceID)
	if traceID == "" {
		traceID = autoTraceID(d.Name, site)
	}
	signature := strings.TrimSpace(d.Signature)
	if signature == "" {
		signature = strings.TrimSpace(site.line)
	}
	return Symbol{
		TraceID:    traceID,
		PartOf:     append([]string(nil), pending.partOf...),
		Implements: append([]string(nil), pending.implements...),
		ModelLinks: append([]string(nil), pending.modelLinks...),
		Path:       site.relPath,
		Line:       site.lineNo,
		Signature:  signature,
		IsTest:     d.IsTest,
		TestOnly:   d.TestOnly || d.IsTest,
		Gates:      append([]string(nil), d.Gates...),
		Benchmark:  d.Benchmark,
	}
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func extractDeclarations(path string, src []byte) ([]declaration, map[int]bool, []validate.Diagnostic, error) {
	ext := strings.ToLower(filepath.Ext(path))
	if ext == ".v" {
		return extractVerilogDeclarations(path, src)
	}
	spec, ok := treeSitterSpec(ext)
	if !ok {
		return nil, nil, nil, nil
	}

	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(spec.Language); err != nil {
		return nil, nil, nil, fmt.Errorf("set parser language for %s: %w", path, err)
	}
	tree := parser.Parse(src, nil)
	defer tree.Close()
	root := tree.RootNode()
	if root == nil {
		return nil, nil, []validate.Diagnostic{{
			Code:     "code.parse_error",
			Severity: validate.SeverityWarning,
			Message:  "tree-sitter produced no syntax tree",
			Path:     path,
		}}, nil
	}
	reader := declarationReader{ext: ext, path: path, src: src, spec: spec}
	out, commentLines := reader.collect(root)
	return out, commentLines, reader.diagnostics(root, out), nil
}

// collect walks the syntax tree and returns its declarations, ordered by line
// and name, with the lines that hold comments.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (r declarationReader) collect(root *sitter.Node) ([]declaration, map[int]bool) {
	out := []declaration{}
	commentLines := map[int]bool{}
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		r.noteComment(n, commentLines)
		if d, ok := r.read(n); ok {
			out = append(out, d)
		}
		count := n.NamedChildCount()
		for i := uint(0); i < count; i++ {
			visit(n.NamedChild(i))
		}
	}
	visit(root)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Name < out[j].Name
	})
	return out, commentLines
}

// diagnostics reports the syntax problems of a JavaScript or Python tree.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func (r declarationReader) diagnostics(root *sitter.Node, out []declaration) []validate.Diagnostic {
	if isJavaScript(r.ext) {
		return javascriptDiagnostics(r.path, root, out)
	}
	if r.ext == ".py" && root.HasError() {
		return []validate.Diagnostic{{Code: "code.parse_error", Severity: validate.SeverityError, Path: r.path, Message: "Python syntax tree contains errors; declaration coverage is incomplete"}}
	}
	return nil
}

// specBuilders builds the tree-sitter spec of each supported extension.
var specBuilders = map[string]func() languageSpec{
	".py": func() languageSpec { return languageSpec{Language: sitter.NewLanguage(tspython.Language())} },
	".js": javascriptSpec, ".mjs": javascriptSpec, ".cjs": javascriptSpec,
	".go": func() languageSpec {
		return declarationSpec(tsgo.Language(),
			[]string{"function_declaration", "method_declaration", "type_declaration", "var_declaration", "const_declaration"},
			[]string{"function_declaration", "method_declaration"})
	},
	".ts":  func() languageSpec { return typescriptSpec(tstypescript.LanguageTypescript()) },
	".tsx": func() languageSpec { return typescriptSpec(tstypescript.LanguageTSX()) },
	".rs": func() languageSpec {
		return declarationSpec(tsrust.Language(),
			[]string{"function_item", "struct_item", "impl_item"},
			[]string{"function_item"})
	},
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func treeSitterSpec(ext string) (languageSpec, bool) {
	build, ok := specBuilders[ext]
	if !ok {
		return languageSpec{}, false
	}
	return build(), true
}

// javascriptSpec is the spec of JavaScript, whose declarations the
// JavaScript reader finds itself.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func javascriptSpec() languageSpec {
	return languageSpec{Language: sitter.NewLanguage(tsjavascript.Language())}
}

// typescriptSpec is the spec of a TypeScript dialect.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func typescriptSpec(language unsafe.Pointer) languageSpec {
	return declarationSpec(language,
		[]string{"function_declaration", "method_definition", "class_declaration"},
		[]string{"function_declaration", "method_definition"})
}

// declarationSpec is the spec of a language whose declarations are the given
// syntax node kinds, of which the required kinds must carry TRLC-LINKS.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func declarationSpec(language unsafe.Pointer, kinds, required []string) languageSpec {
	return languageSpec{
		Language:          sitter.NewLanguage(language),
		DeclarationKind:   kindSet(kinds),
		TraceRequiredKind: kindSet(required),
	}
}

// kindSet is the set of the given syntax node kinds.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func kindSet(kinds []string) map[string]bool {
	set := make(map[string]bool, len(kinds))
	for _, kind := range kinds {
		set[kind] = true
	}
	return set
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func declarationName(n *sitter.Node, src []byte) string {
	if n == nil {
		return ""
	}
	if name := n.ChildByFieldName("name"); name != nil {
		return strings.TrimSpace(name.Utf8Text(src))
	}
	switch n.Kind() {
	case "type_declaration":
		if name, ok := specName(n, src, []string{"type_spec", "type_alias"}); ok {
			return name
		}
	case "var_declaration", "const_declaration":
		// Package-level var/const declarations carry their name on a nested spec.
		// They attach trace/model-link markers (full linking) but do not require them.
		if name, ok := specName(n, src, []string{"var_spec", "const_spec"}); ok {
			return name
		}
	}
	return firstIdentifier(n, src)
}

// specName is the name of the first child of one of the spec kinds that has
// one.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func specName(n *sitter.Node, src []byte, kinds []string) (string, bool) {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		child := n.NamedChild(i)
		if child == nil || !slices.Contains(kinds, child.Kind()) {
			continue
		}
		if name := child.ChildByFieldName("name"); name != nil {
			return strings.TrimSpace(name.Utf8Text(src)), true
		}
	}
	return "", false
}

// firstIdentifier is the text of the first identifier child of n.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func firstIdentifier(n *sitter.Node, src []byte) string {
	for i := uint(0); i < n.NamedChildCount(); i++ {
		child := n.NamedChild(i)
		if child == nil {
			continue
		}
		switch child.Kind() {
		case "identifier", "field_identifier", "type_identifier", "property_identifier":
			return strings.TrimSpace(child.Utf8Text(src))
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-010
func firstLine(s string) string {
	if s == "" {
		return ""
	}
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func markerValue(line, marker string) (string, bool) {
	idx := strings.Index(line, marker)
	if idx < 0 {
		return "", false
	}
	return strings.TrimSpace(line[idx+len(marker):]), true
}

// TRLC-LINKS: REQ-EMG-010
func splitCSV(v string) []string {
	parts := strings.Split(v, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	return out
}

// appendUnique appends the values of more that are not yet in values, keeping
// the order written.
// TRLC-LINKS: REQ-EMG-174
func appendUnique(values, more []string) []string {
	for _, v := range more {
		if !slices.Contains(values, v) {
			values = append(values, v)
		}
	}
	return values
}

// TRLC-LINKS: REQ-EMG-010
func joinLineNumbers(lines []int) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d", line))
	}
	return strings.Join(parts, ",")
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func validRequirementID(id string) bool {
	return requirementIDPattern.MatchString(strings.TrimSpace(id))
}

// TRLC-LINKS: REQ-EMG-010
func isCommentLike(trimmed string) bool {
	return slices.ContainsFunc([]string{"//", "#", "/*", "*", "--"}, func(prefix string) bool {
		return strings.HasPrefix(trimmed, prefix)
	})
}

// TRLC-LINKS: REQ-EMG-010
func isAttributeLike(trimmed string) bool {
	return strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "#[")
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-058
func autoTraceID(symbolName string, site symbolSite) string {
	name := strings.TrimSpace(symbolName)
	if name != "" {
		return FileID(site.idPath) + "--" + slugUpper(name)
	}
	return fmt.Sprintf("AUTO-%s:%d", site.relPath, site.lineNo)
}

// FileID returns the CODE identifier of a module-relative file path: the path upper-cased
// with each run of non-alphanumeric characters folded to one hyphen.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-058
func FileID(relPath string) string {
	return "CODE-" + strings.ToUpper(strings.Trim(nonAlnum.ReplaceAllString(filepath.ToSlash(relPath), "-"), "-"))
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)
var requirementIDPattern = regexp.MustCompile(`^REQ-[A-Z0-9]+-[0-9]+$`)

// TRLC-LINKS: REQ-EMG-010
func slugUpper(s string) string {
	s = strings.TrimSpace(s)
	if s == "" {
		return "SYMBOL"
	}
	normalized := nonAlnum.ReplaceAllString(s, "")
	normalized = strings.ToUpper(normalized)
	if normalized == "" {
		return "SYMBOL"
	}
	return normalized
}

// namedChildren returns the named children of n that exist.
// TRLC-LINKS: REQ-EMG-010
func namedChildren(n *sitter.Node) []*sitter.Node {
	count := n.NamedChildCount()
	out := make([]*sitter.Node, 0, count)
	for i := uint(0); i < count; i++ {
		if child := n.NamedChild(i); child != nil {
			out = append(out, child)
		}
	}
	return out
}
