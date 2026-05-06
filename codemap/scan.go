// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
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
	Name         string
	Kind         string
	Line         int
	Signature    string
	RequiresTRLC bool
}

type languageSpec struct {
	Language          *sitter.Language
	DeclarationKind   map[string]bool
	TraceRequiredKind map[string]bool
}

var supportedExt = map[string]bool{
	".go":  true,
	".ts":  true,
	".tsx": true,
	".rs":  true,
}

// TRLC-LINKS: REQ-EMG-010
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

// TRLC-LINKS: REQ-EMG-010
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

	decls, commentLines, parseDiags, err := extractDeclarations(path, src)
	if err != nil {
		return nil, nil, err
	}
	diags := append([]validate.Diagnostic(nil), parseDiags...)

	declByLine := map[int]declaration{}
	traceRequiredByLine := map[int]declaration{}
	for _, d := range decls {
		if _, exists := declByLine[d.Line]; !exists {
			declByLine[d.Line] = d
		}
		if d.RequiresTRLC {
			traceRequiredByLine[d.Line] = d
		}
	}

	symbols := []Symbol{}
	linkedLines := map[int]bool{}
	var pending pendingTrace

	scanner := bufio.NewScanner(strings.NewReader(string(src)))
	lineNo := 0
	for scanner.Scan() {
		lineNo++
		line := scanner.Text()
		trimmed := strings.TrimSpace(line)

		if commentLines[lineNo] && isCommentLike(trimmed) {
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
				for _, link := range pending.implements {
					if !validRequirementID(link) {
						diags = append(diags, validate.Diagnostic{
							Code:     "code.invalid_trlc_link",
							Severity: validate.SeverityError,
							Message:  fmt.Sprintf("TRLC-LINKS value %q is not a requirement id", link),
							Path:     fmt.Sprintf("%s:%d", relPath, lineNo),
						})
					}
				}
				if pending.firstLine == 0 {
					pending.firstLine = lineNo
				}
				continue
			}
		}

		if pending.firstLine == 0 {
			continue
		}

		if d, ok := declByLine[lineNo]; ok {
			implements := append([]string(nil), pending.implements...)
			symbols = append(symbols, symbolForDeclaration(d, relPath, lineNo, trimmed, pending.traceID, pending.partOf, implements))
			if len(implements) > 0 {
				linkedLines[lineNo] = true
			}
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
	missingLines := make([]int, 0, len(traceRequiredByLine))
	for line := range traceRequiredByLine {
		if !linkedLines[line] {
			missingLines = append(missingLines, line)
		}
	}
	sort.Ints(missingLines)
	if len(missingLines) > 0 {
		lineList := joinLineNumbers(missingLines)
		diags = append(diags, validate.Diagnostic{
			Code:     "code.missing_trlc_link",
			Severity: validate.SeverityError,
			Message:  fmt.Sprintf("functions missing TRLC-LINKS at lines %s", strings.ReplaceAll(lineList, ",", ", ")),
			Path:     fmt.Sprintf("%s:%s", relPath, lineList),
		})
	}

	return symbols, diags, nil
}

// TRLC-LINKS: REQ-EMG-010
func symbolForDeclaration(d declaration, relPath string, lineNo int, line string, traceID string, partOf []string, implements []string) Symbol {
	traceID = strings.TrimSpace(traceID)
	if traceID == "" {
		traceID = autoTraceID(d.Name, relPath, lineNo)
	}
	signature := strings.TrimSpace(d.Signature)
	if signature == "" {
		signature = strings.TrimSpace(line)
	}
	return Symbol{
		TraceID:    traceID,
		PartOf:     append([]string(nil), partOf...),
		Implements: append([]string(nil), implements...),
		Path:       relPath,
		Line:       lineNo,
		Signature:  signature,
	}
}

// TRLC-LINKS: REQ-EMG-010
func extractDeclarations(path string, src []byte) ([]declaration, map[int]bool, []validate.Diagnostic, error) {
	ext := strings.ToLower(filepath.Ext(path))
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

	out := []declaration{}
	commentLines := map[int]bool{}
	var visit func(n *sitter.Node)
	visit = func(n *sitter.Node) {
		if n == nil {
			return
		}
		if n.IsNamed() && strings.Contains(n.Kind(), "comment") {
			start := int(n.StartPosition().Row) + 1
			end := int(n.EndPosition().Row) + 1
			for line := start; line <= end; line++ {
				commentLines[line] = true
			}
		}
		if n.IsNamed() && spec.DeclarationKind[n.Kind()] {
			name := declarationName(n, src)
			line := int(n.StartPosition().Row) + 1
			signature := firstLine(strings.TrimSpace(n.Utf8Text(src)))
			if name != "" {
				out = append(out, declaration{
					Name:         name,
					Kind:         n.Kind(),
					Line:         line,
					Signature:    signature,
					RequiresTRLC: spec.TraceRequiredKind[n.Kind()],
				})
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
	return out, commentLines, nil, nil
}

// TRLC-LINKS: REQ-EMG-010
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
			TraceRequiredKind: map[string]bool{
				"function_declaration": true,
				"method_declaration":   true,
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
			TraceRequiredKind: map[string]bool{
				"function_declaration": true,
				"method_definition":    true,
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
			TraceRequiredKind: map[string]bool{
				"function_declaration": true,
				"method_definition":    true,
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
			TraceRequiredKind: map[string]bool{
				"function_item": true,
			},
		}, true
	default:
		return languageSpec{}, false
	}
}

// TRLC-LINKS: REQ-EMG-010
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

// TRLC-LINKS: REQ-EMG-010
func joinLineNumbers(lines []int) string {
	parts := make([]string, 0, len(lines))
	for _, line := range lines {
		parts = append(parts, fmt.Sprintf("%d", line))
	}
	return strings.Join(parts, ",")
}

// TRLC-LINKS: REQ-EMG-010
func validRequirementID(id string) bool {
	return requirementIDPattern.MatchString(strings.TrimSpace(id))
}

// TRLC-LINKS: REQ-EMG-010
func isCommentLike(trimmed string) bool {
	return strings.HasPrefix(trimmed, "//") ||
		strings.HasPrefix(trimmed, "#") ||
		strings.HasPrefix(trimmed, "/*") ||
		strings.HasPrefix(trimmed, "*") ||
		strings.HasPrefix(trimmed, "--")
}

// TRLC-LINKS: REQ-EMG-010
func isAttributeLike(trimmed string) bool {
	return strings.HasPrefix(trimmed, "@") || strings.HasPrefix(trimmed, "#[")
}

// TRLC-LINKS: REQ-EMG-010
func autoTraceID(symbolName, relPath string, lineNo int) string {
	name := strings.TrimSpace(symbolName)
	if name != "" {
		return "CODE-" + slugUpper(name)
	}
	return fmt.Sprintf("AUTO-%s:%d", relPath, lineNo)
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
