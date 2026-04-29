// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package engmodel

import (
	"fmt"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/codemap"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-010
func inferCodeItems(bundle model.Bundle, codeRootOption string) ([]inferredCodeItem, []validate.Diagnostic) {
	baseDir := filepath.Dir(bundle.ArchitecturePath)
	roots := make([]string, 0, len(bundle.Architecture.InferenceHints.CodeSources)+1)
	if strings.TrimSpace(codeRootOption) != "" {
		roots = append(roots, codeRootOption)
	}
	for _, src := range bundle.Architecture.InferenceHints.CodeSources {
		roots = append(roots, resolveSourcePath(baseDir, src))
	}
	if len(roots) == 0 {
		return nil, nil
	}

	items := []inferredCodeItem{}
	diags := []validate.Diagnostic{}
	seen := map[string]bool{}

	for _, root := range roots {
		abs := root
		if !filepath.IsAbs(abs) {
			abs, _ = filepath.Abs(root)
		}
		metadata := scanCodeMetadata(abs)
		owners := map[string]string{}
		descriptions := map[string]string{}
		for rel, m := range metadata {
			owners[rel] = m.Owner
			if x := strings.TrimSpace(m.Description); x != "" {
				descriptions[rel] = x
			}
		}
		symbols, sDiags, err := codemap.Scan(abs)
		if err != nil {
			diags = append(diags, validate.Diagnostic{
				Code:     "code.scan_failed",
				Severity: validate.SeverityWarning,
				Message:  err.Error(),
				Path:     abs,
			})
			continue
		}
		diags = append(diags, sDiags...)

		for rel, owner := range owners {
			desc := strings.TrimSpace(descriptions[rel])
			key := "source_file|" + rel + "|" + owner
			if seen[key] {
				continue
			}
			seen[key] = true
			items = append(items, inferredCodeItem{
				Element:     rel,
				Kind:        "source_file",
				Owner:       owner,
				Description: desc,
				Source:      rel,
			})

			deps, depDiags := parseCodeDependencies(abs, rel, owner)
			diags = append(diags, depDiags...)
			for _, dep := range deps {
				key := dep.Kind + "|" + dep.Element + "|" + dep.Owner + "|" + dep.Source
				if seen[key] {
					continue
				}
				seen[key] = true
				items = append(items, dep)
			}
		}

		for _, s := range symbols {
			owner := ownerForPath(s.Path, owners)
			label := s.TraceID
			if strings.TrimSpace(label) == "" {
				label = s.Signature
			}
			key := "symbol|" + label + "|" + s.Path + fmt.Sprintf("|%d", s.Line)
			if seen[key] {
				continue
			}
			seen[key] = true
			items = append(items, inferredCodeItem{
				Element: label,
				Kind:    "symbol",
				Owner:   owner,
				Source:  fmt.Sprintf("%s:%d", s.Path, s.Line),
			})
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		if items[i].Element != items[j].Element {
			return items[i].Element < items[j].Element
		}
		return items[i].Source < items[j].Source
	})
	return items, validate.SortDiagnostics(diags)
}

func parseCodeDependencies(root, rel, owner string) ([]inferredCodeItem, []validate.Diagnostic) {
	path := filepath.Join(root, filepath.FromSlash(rel))
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, []validate.Diagnostic{{
			Code:     "code.dependency_read_failed",
			Severity: validate.SeverityWarning,
			Message:  err.Error(),
			Path:     path,
		}}
	}
	switch strings.ToLower(filepath.Ext(rel)) {
	case ".go":
		return parseGoDependencies(rel, data, owner)
	case ".ts", ".tsx":
		return parseTypeScriptDependencies(rel, string(data), owner), nil
	case ".rs":
		return parseRustDependencies(rel, string(data), owner), nil
	default:
		return nil, nil
	}
}

type codeFileMetadata struct {
	Owner       string
	Description string
}

func parseGoDependencies(rel string, src []byte, owner string) ([]inferredCodeItem, []validate.Diagnostic) {
	full := filepath.ToSlash(filepath.Clean(rel))
	fset := token.NewFileSet()
	f, err := parser.ParseFile(fset, full, src, parser.ImportsOnly)
	if err != nil {
		return nil, []validate.Diagnostic{{
			Code:     "code.go_dependency_parse_failed",
			Severity: validate.SeverityWarning,
			Message:  err.Error(),
			Path:     rel,
		}}
	}
	out := []inferredCodeItem{}
	for _, im := range f.Imports {
		path := strings.TrimSpace(strings.Trim(im.Path.Value, "\""))
		if path == "" {
			continue
		}
		kind := goImportKind(path)
		out = append(out, inferredCodeItem{
			Element: path,
			Kind:    kind,
			Owner:   owner,
			Source:  rel,
		})
	}
	return out, nil
}

func goImportKind(path string) string {
	switch {
	case strings.HasPrefix(path, "github.com/labeth/engineering-model-go/"):
		return "library_first_party"
	case strings.Contains(path, "."):
		return "library_external"
	default:
		return "library_stdlib"
	}
}

var (
	tsImportRe  = regexp.MustCompile(`(?m)^\s*import\s+(?:[^'"]+?\s+from\s+)?['"]([^'"]+)['"]`)
	tsRequireRe = regexp.MustCompile(`(?m)require\(\s*['"]([^'"]+)['"]\s*\)`)
	rsUseRe     = regexp.MustCompile(`(?m)^\s*use\s+([A-Za-z0-9_:\{\}]+)\s*;`)
)

func parseTypeScriptDependencies(rel, content, owner string) []inferredCodeItem {
	out := []inferredCodeItem{}
	add := func(dep string) {
		dep = strings.TrimSpace(dep)
		if dep == "" {
			return
		}
		kind := "library_external"
		if strings.HasPrefix(dep, ".") || strings.HasPrefix(dep, "/") {
			kind = "library_first_party"
		}
		out = append(out, inferredCodeItem{
			Element: dep,
			Kind:    kind,
			Owner:   owner,
			Source:  rel,
		})
	}
	for _, m := range tsImportRe.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	for _, m := range tsRequireRe.FindAllStringSubmatch(content, -1) {
		if len(m) > 1 {
			add(m[1])
		}
	}
	return out
}

func parseRustDependencies(rel, content, owner string) []inferredCodeItem {
	out := []inferredCodeItem{}
	for _, m := range rsUseRe.FindAllStringSubmatch(content, -1) {
		if len(m) < 2 {
			continue
		}
		dep := strings.TrimSpace(m[1])
		if dep == "" {
			continue
		}
		kind := "library_external"
		if strings.HasPrefix(dep, "crate::") || strings.HasPrefix(dep, "self::") || strings.HasPrefix(dep, "super::") {
			kind = "library_first_party"
		}
		out = append(out, inferredCodeItem{
			Element: dep,
			Kind:    kind,
			Owner:   owner,
			Source:  rel,
		})
	}
	return out
}

func scanCodeMetadata(root string) map[string]codeFileMetadata {
	out := map[string]codeFileMetadata{}
	_ = filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil || d.IsDir() {
			return nil
		}
		ext := strings.ToLower(filepath.Ext(path))
		if ext != ".go" && ext != ".ts" && ext != ".tsx" && ext != ".rs" {
			return nil
		}
		data, readErr := os.ReadFile(path)
		if readErr != nil {
			return nil
		}
		rel, relErr := filepath.Rel(root, path)
		if relErr != nil {
			rel = path
		}
		rel = filepath.ToSlash(rel)
		owner := "unresolved"
		description := ""
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := extractMarkerValue(strings.TrimSpace(line), "ENGMODEL-OWNER-UNIT:"); ok {
				if x := strings.TrimSpace(v); x != "" {
					owner = x
				}
			}
			if description == "" {
				if desc, ok := extractCodeDescriptionMarker(line); ok {
					description = desc
				}
			}
		}
		out[rel] = codeFileMetadata{
			Owner:       owner,
			Description: description,
		}
		return nil
	})
	return out
}

func extractCodeDescriptionMarker(line string) (string, bool) {
	markers := []string{
		"ENGMODEL-CODE-DESCRIPTION:",
		"engmodel:code-description:",
		"engmodel:code-description",
		"engmodel.code.description:",
		"engmodel.code.description",
	}
	for _, marker := range markers {
		if v, ok := extractMarkerValue(line, marker); ok {
			v = strings.TrimSpace(strings.TrimPrefix(strings.TrimSpace(v), ":"))
			if v != "" {
				return v, true
			}
		}
	}
	return "", false
}

func ownerForPath(path string, owners map[string]string) string {
	if x := strings.TrimSpace(owners[path]); x != "" && x != "unresolved" {
		return x
	}
	best := ""
	for p, owner := range owners {
		if owner == "" || owner == "unresolved" {
			continue
		}
		if strings.HasPrefix(path, p) {
			if len(p) > len(best) {
				best = owner
			}
		}
	}
	if best != "" {
		return best
	}
	return "unresolved"
}
