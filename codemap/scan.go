package codemap

import (
	"bufio"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	sitter "github.com/tree-sitter/go-tree-sitter"
	tsgo "github.com/tree-sitter/tree-sitter-go/bindings/go"
	tsrust "github.com/tree-sitter/tree-sitter-rust/bindings/go"
	tstypescript "github.com/tree-sitter/tree-sitter-typescript/bindings/go"

	"github.com/labeth/engineering-model-go/validate"
)

type Symbol struct {
	TraceID    string
	PartOf     []string
	Implements []string
	Path       string
	Line       int
	Signature  string
}

type pendingTrace struct {
	traceID    string
	partOf     []string
	implements []string
	firstLine  int
}

type declaration struct {
	Name      string
	Line      int
	Signature string
}

type languageSpec struct {
	Language        *sitter.Language
	DeclarationKind map[string]bool
}

var supportedExt = map[string]bool{
	".go":  true,
	".ts":  true,
	".tsx": true,
	".rs":  true,
}

func Scan(root string) ([]Symbol, []validate.Diagnostic, error) {
	absRoot, err := filepath.Abs(root)
	if err != nil {
		return nil, nil, fmt.Errorf("resolve code root: %w", err)
	}

	symbols := []Symbol{}
	diags := []validate.Diagnostic{}

	err = filepath.WalkDir(absRoot, func(path string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || name == "node_modules" || name == "vendor" || name == ".idea" || name == ".vscode" {
				return filepath.SkipDir
			}
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if !supportedExt[ext] {
			return nil
		}
		fileSymbols, fileDiags, err := scanFile(absRoot, path)
		if err != nil {
			return err
		}
		symbols = append(symbols, fileSymbols...)
		diags = append(diags, fileDiags...)
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("walk code root: %w", err)
	}

	sort.SliceStable(symbols, func(i, j int) bool {
		if symbols[i].Path != symbols[j].Path {
			return symbols[i].Path < symbols[j].Path
		}
		if symbols[i].Line != symbols[j].Line {
			return symbols[i].Line < symbols[j].Line
		}
		return symbols[i].TraceID < symbols[j].TraceID
	})
	diags = validate.SortDiagnostics(diags)
	return symbols, diags, nil
}

func scanFile(root, path string) ([]Symbol, []validate.Diagnostic, error) {
	src, err := os.ReadFile(path)
	if err != nil {
		return nil, nil, fmt.Errorf("read %s: %w", path, err)
	}

	relPath, err := filepath.Rel(root, path)
	if err != nil {
		relPath = path
	}
	relPath = filepath.ToSlash(relPath)

	decls, parseDiags, err := extractDeclarations(path, src)
	if err != nil {
		return nil, nil, err
	}
	diags := append([]validate.Diagnostic(nil), parseDiags...)

	declByLine := map[int]declaration{}
	for _, d := range decls {
		if _, exists := declByLine[d.Line]; !exists {
			declByLine[d.Line] = d
		}
	}

	symbols := []Symbol{}
	var pending pendingTrace

	scanner := bufio.NewScanner(strings.NewReader(string(src)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if v, ok := markerValue(trimmed, "TRACE-ID:"); ok {
			pending.traceID = strings.TrimSpace(v)
			if pending.firstLine == 0 {
				pending.firstLine = lineNo
			}
			continue
		}
		if v, ok := markerValue(trimmed, "TRACE-PART-OF:"); ok {
			pending.partOf = splitCSV(v)
			if pending.firstLine == 0 {
				pending.firstLine = lineNo
			}
			continue
		}
		if v, ok := markerValue(trimmed, "TRLC-LINKS:"); ok {
			pending.implements = splitCSV(v)
			if pending.firstLine == 0 {
				pending.firstLine = lineNo
			}
			continue
		}

		if pending.firstLine == 0 {
			continue
		}

		if d, ok := declByLine[lineNo]; ok {
			traceID := strings.TrimSpace(pending.traceID)
			if traceID == "" {
				traceID = autoTraceID(d.Name, relPath, lineNo)
			}
			signature := strings.TrimSpace(d.Signature)
			if signature == "" {
				signature = strings.TrimSpace(trimmed)
			}
			symbols = append(symbols, Symbol{
				TraceID:    traceID,
				PartOf:     append([]string(nil), pending.partOf...),
				Implements: append([]string(nil), pending.implements...),
				Path:       relPath,
				Line:       lineNo,
				Signature:  signature,
			})
			pending = pendingTrace{}
			continue
		}

		// Allow blank lines, comments, and annotation/decorator lines between marker and declaration.
		if trimmed == "" || isCommentLike(trimmed) || isAttributeLike(trimmed) {
			continue
		}

		diags = append(diags, validate.Diagnostic{
			Code:     "code.trace_unattached",
			Severity: validate.SeverityWarning,
			Message:  "trace marker block not attached to any declaration",
			Path:     fmt.Sprintf("%s:%d", relPath, pending.firstLine),
		})
		pending = pendingTrace{}
	}
	if err := scanner.Err(); err != nil {
		return nil, nil, fmt.Errorf("scan %s: %w", path, err)
	}
	if pending.firstLine != 0 {
		diags = append(diags, validate.Diagnostic{
			Code:     "code.trace_unattached",
			Severity: validate.SeverityWarning,
			Message:  "trace marker block not attached to any declaration",
			Path:     fmt.Sprintf("%s:%d", relPath, pending.firstLine),
		})
	}

	return symbols, diags, nil
}

func extractDeclarations(path string, src []byte) ([]declaration, []validate.Diagnostic, error) {
	ext := strings.ToLower(filepath.Ext(path))
	spec, ok := treeSitterSpec(ext)
	if !ok {
		return nil, nil, nil
	}

	parser := sitter.NewParser()
	defer parser.Close()
	if err := parser.SetLanguage(spec.Language); err != nil {
		return nil, nil, fmt.Errorf("set parser language for %s: %w", path, err)
	}
	tree := parser.Parse(src, nil)
	defer tree.Close()
	root := tree.RootNode()
	if root == nil {
		return nil, []validate.Diagnostic{{
			Code:     "code.parse_error",
			Severity: validate.SeverityWarning,
			Message:  "tree-sitter produced no syntax tree",
			Path:     path,
		}}, nil
	}

	out := []declaration{}
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.IsNamed() && spec.DeclarationKind[n.Kind()] {
			name := declarationName(n, src)
			line := int(n.StartPosition().Row) + 1
			signature := firstLine(strings.TrimSpace(n.Utf8Text(src)))
			if name != "" {
				out = append(out, declaration{Name: name, Line: line, Signature: signature})
			}
		}
		count := n.NamedChildCount()
		for i := uint(0); i < count; i++ {
			child := n.NamedChild(i)
			visit(child)
		}
	}
	visit(root)

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Line != out[j].Line {
			return out[i].Line < out[j].Line
		}
		return out[i].Name < out[j].Name
	})
	return out, nil, nil
}

func treeSitterSpec(ext string) (languageSpec, bool) {
	switch ext {
	case ".go":
		return languageSpec{
			Language: sitter.NewLanguage(tsgo.Language()),
			DeclarationKind: map[string]bool{
				"function_declaration": true,
				"method_declaration":   true,
				"type_declaration":     true,
			},
		}, true
	case ".ts":
		return languageSpec{
			Language: sitter.NewLanguage(tstypescript.LanguageTypescript()),
			DeclarationKind: map[string]bool{
				"function_declaration": true,
				"method_definition":    true,
				"class_declaration":    true,
			},
		}, true
	case ".tsx":
		return languageSpec{
			Language: sitter.NewLanguage(tstypescript.LanguageTSX()),
			DeclarationKind: map[string]bool{
				"function_declaration": true,
				"method_definition":    true,
				"class_declaration":    true,
			},
		}, true
	case ".rs":
		return languageSpec{
			Language: sitter.NewLanguage(tsrust.Language()),
			DeclarationKind: map[string]bool{
				"function_item": true,
				"struct_item":   true,
				"impl_item":     true,
			},
		}, true
	default:
		return languageSpec{}, false
	}
}

func declarationName(n *sitter.Node, src []byte) string {
	if n == nil {
		return ""
	}
	if name := n.ChildByFieldName("name"); name != nil {
		return strings.TrimSpace(name.Utf8Text(src))
	}
	count := n.NamedChildCount()
	for i := uint(0); i < count; i++ {
		child := n.NamedChild(i)
		if child == nil {
			continue
		}
		k := child.Kind()
		if k == "identifier" || k == "field_identifier" || k == "type_identifier" || k == "property_identifier" {
			return strings.TrimSpace(child.Utf8Text(src))
		}
	}
	return ""
}

func firstLine(s string) string {
	if s == "" {
		return ""
	}
	if idx := strings.IndexByte(s, '\n'); idx >= 0 {
		return strings.TrimSpace(s[:idx])
	}
	return strings.TrimSpace(s)
}

func markerValue(line, marker string) (string, bool) {
	idx := strings.Index(line, marker)
	if idx < 0 {
		return "", false
	}
	return strings.TrimSpace(line[idx+len(marker):]), true
}

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

func isCommentLike(trimmed string) bool {
	return strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "--")
}

func isAttributeLike(trimmed string) bool {
	return strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "#[")
}

func autoTraceID(symbolName, relPath string, lineNo int) string {
	name := strings.TrimSpace(symbolName)
	if name != "" {
		return "CODE-" + slugUpper(name)
	}
	return fmt.Sprintf("AUTO-%s:%d", relPath, lineNo)
}

var nonAlnum = regexp.MustCompile(`[^A-Za-z0-9]+`)

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
