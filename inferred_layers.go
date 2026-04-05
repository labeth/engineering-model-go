package engmodel

import (
	"fmt"
	"go/parser"
	"go/token"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/labeth/engineering-model-go/codemap"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
	"gopkg.in/yaml.v3"
)

type inferredRuntimeItem struct {
	Name   string
	Kind   string
	Owner  string
	Source string
	Ports  []string
}

type inferredCodeItem struct {
	Element string
	Kind    string
	Owner   string
	Source  string
}

func inferRuntimeItems(bundle model.Bundle) ([]inferredRuntimeItem, []validate.Diagnostic) {
	baseDir := filepath.Dir(bundle.ArchitecturePath)
	items := []inferredRuntimeItem{}
	diags := []validate.Diagnostic{}
	seen := map[string]bool{}

	for _, src := range bundle.Architecture.InferenceHints.RuntimeSources {
		root := resolveSourcePath(baseDir, src)
		walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
			if err != nil {
				return err
			}
			if d.IsDir() {
				return nil
			}
			ext := strings.ToLower(filepath.Ext(path))
			switch ext {
			case ".tf":
				tfItems, tfDiags := parseTerraformRuntime(path)
				diags = append(diags, tfDiags...)
				for _, it := range tfItems {
					key := it.Source + "|" + it.Kind + "|" + it.Name + "|" + it.Owner
					if !seen[key] {
						seen[key] = true
						items = append(items, it)
					}
				}
			case ".yaml", ".yml":
				yItems, yDiags := parseManifestRuntime(path)
				diags = append(diags, yDiags...)
				for _, it := range yItems {
					key := it.Source + "|" + it.Kind + "|" + it.Name + "|" + it.Owner
					if !seen[key] {
						seen[key] = true
						items = append(items, it)
					}
				}
			}
			return nil
		})
		if walkErr != nil {
			diags = append(diags, validate.Diagnostic{
				Code:     "runtime.source_walk_failed",
				Severity: validate.SeverityWarning,
				Message:  fmt.Sprintf("failed to walk runtime source %q: %v", root, walkErr),
				Path:     "inferenceHints.runtimeSources",
			})
		}
	}

	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].Source < items[j].Source
	})
	return items, validate.SortDiagnostics(diags)
}

func parseTerraformRuntime(path string) ([]inferredRuntimeItem, []validate.Diagnostic) {
	parser := hclparse.NewParser()
	file, diags := parser.ParseHCLFile(path)
	if diags.HasErrors() {
		return nil, []validate.Diagnostic{{
			Code:     "runtime.terraform_parse_failed",
			Severity: validate.SeverityWarning,
			Message:  diags.Error(),
			Path:     path,
		}}
	}

	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
		},
	}
	content, _, cDiags := file.Body.PartialContent(schema)
	if cDiags.HasErrors() {
		return nil, []validate.Diagnostic{{
			Code:     "runtime.terraform_schema_failed",
			Severity: validate.SeverityWarning,
			Message:  cDiags.Error(),
			Path:     path,
		}}
	}

	out := []inferredRuntimeItem{}
	for _, b := range content.Blocks {
		rtype := ""
		rname := ""
		if len(b.Labels) > 0 {
			rtype = strings.TrimSpace(b.Labels[0])
		}
		if len(b.Labels) > 1 {
			rname = strings.TrimSpace(b.Labels[1])
		}
		if rtype == "" && rname == "" {
			continue
		}
		kind := normalizeTerraformKind(rtype)
		name := rname
		if name == "" {
			name = rtype
		}
		out = append(out, inferredRuntimeItem{
			Name:   name,
			Kind:   kind,
			Owner:  inferOwnerByConvention(kind, name),
			Source: filepath.ToSlash(path),
		})
	}
	return out, nil
}

func normalizeTerraformKind(resourceType string) string {
	t := strings.TrimSpace(strings.ToLower(resourceType))
	switch {
	case strings.Contains(t, "eks") && strings.Contains(t, "cluster"):
		return "cluster"
	case strings.Contains(t, "namespace"):
		return "namespace"
	default:
		return "terraform_resource"
	}
}

func inferOwnerByConvention(kind, name string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	n := strings.ToLower(strings.TrimSpace(name))
	switch {
	case k == "cluster":
		return "FU-CLUSTER-PROVISIONING"
	case k == "gitrepository" || k == "kustomization":
		return "FU-GITOPS-OPERATIONS"
	case k == "namespace" && (n == "flux-system" || n == "flux_system"):
		return "FU-GITOPS-OPERATIONS"
	case (k == "gitrepository" || k == "kustomization") && strings.HasPrefix(n, "flux-system/"):
		return "FU-GITOPS-OPERATIONS"
	case k == "namespace" && n == "payments":
		return "FU-CHECKOUT"
	case k == "namespace" && n == "risk":
		return "FU-RISK-SCORING"
	default:
		return "unresolved"
	}
}

type manifestMetadata struct {
	Name        string            `yaml:"name"`
	Namespace   string            `yaml:"namespace"`
	Annotations map[string]string `yaml:"annotations"`
}

type manifestObject struct {
	Kind     string           `yaml:"kind"`
	Metadata manifestMetadata `yaml:"metadata"`
	Spec     manifestSpec     `yaml:"spec"`
}

type manifestSpec struct {
	Ports  []manifestPort `yaml:"ports"`
	Values map[string]any `yaml:"values"`
}

type manifestPort struct {
	Name       string `yaml:"name"`
	Protocol   string `yaml:"protocol"`
	Port       int    `yaml:"port"`
	TargetPort any    `yaml:"targetPort"`
}

func parseManifestRuntime(path string) ([]inferredRuntimeItem, []validate.Diagnostic) {
	raw, readErr := os.ReadFile(path)
	if readErr == nil {
		txt := string(raw)
		// Skip unrendered Helm templates; they are not concrete runtime manifests.
		if strings.Contains(txt, "{{") && strings.Contains(txt, "}}") {
			return nil, nil
		}
	}

	f, err := os.Open(path)
	if err != nil {
		return nil, []validate.Diagnostic{{
			Code:     "runtime.manifest_open_failed",
			Severity: validate.SeverityWarning,
			Message:  fmt.Sprintf("open manifest: %v", err),
			Path:     path,
		}}
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	out := []inferredRuntimeItem{}
	for {
		var doc manifestObject
		err := dec.Decode(&doc)
		if err != nil {
			if err == io.EOF {
				break
			}
			return out, []validate.Diagnostic{{
				Code:     "runtime.manifest_parse_failed",
				Severity: validate.SeverityWarning,
				Message:  fmt.Sprintf("decode manifest: %v", err),
				Path:     path,
			}}
		}

		kind := strings.TrimSpace(doc.Kind)
		name := strings.TrimSpace(doc.Metadata.Name)
		if kind == "" || name == "" {
			continue
		}
		ports := normalizePorts(doc.Spec.Ports)
		if len(ports) == 0 && strings.EqualFold(kind, "HelmRelease") {
			ports = portsFromHelmValues(doc.Spec.Values)
		}
		if ns := strings.TrimSpace(doc.Metadata.Namespace); ns != "" {
			name = fmt.Sprintf("%s/%s", ns, name)
		}
		owner := "unresolved"
		if doc.Metadata.Annotations != nil {
			if x := strings.TrimSpace(doc.Metadata.Annotations["engmodel.dev/owner-unit"]); x != "" {
				owner = x
			}
		}
		if owner == "unresolved" {
			owner = inferOwnerByConvention(strings.ToLower(kind), name)
		}
		out = append(out, inferredRuntimeItem{
			Name:   name,
			Kind:   strings.ToLower(kind),
			Owner:  owner,
			Source: filepath.ToSlash(path),
			Ports:  ports,
		})
	}
	return out, nil
}

func normalizePorts(in []manifestPort) []string {
	out := make([]string, 0, len(in))
	for _, p := range in {
		if p.Port <= 0 {
			continue
		}
		proto := strings.TrimSpace(p.Protocol)
		if proto == "" {
			proto = "TCP"
		}
		name := strings.TrimSpace(p.Name)
		if name != "" {
			out = append(out, fmt.Sprintf("%s:%d/%s", name, p.Port, proto))
		} else {
			out = append(out, fmt.Sprintf("%d/%s", p.Port, proto))
		}
	}
	return out
}

func portsFromHelmValues(values map[string]any) []string {
	if values == nil {
		return nil
	}
	serviceRaw, ok := values["service"]
	if !ok {
		return nil
	}
	service, ok := serviceRaw.(map[string]any)
	if !ok {
		return nil
	}
	portRaw, ok := service["port"]
	if !ok {
		return nil
	}
	port := 0
	switch v := portRaw.(type) {
	case int:
		port = v
	case int64:
		port = int(v)
	case float64:
		port = int(v)
	case string:
		fmt.Sscanf(strings.TrimSpace(v), "%d", &port)
	}
	if port <= 0 {
		return nil
	}
	return []string{fmt.Sprintf("%d/TCP", port)}
}

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
		owners := scanCodeOwners(abs)
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
			key := "source_file|" + rel + "|" + owner
			if seen[key] {
				continue
			}
			seen[key] = true
			items = append(items, inferredCodeItem{
				Element: rel,
				Kind:    "source_file",
				Owner:   owner,
				Source:  rel,
			})

			deps, depDiags := parseCodeDependencies(abs, rel)
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

func parseCodeDependencies(root, rel string) ([]inferredCodeItem, []validate.Diagnostic) {
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
	owner := "unresolved"
	for _, line := range strings.Split(string(data), "\n") {
		if v, ok := extractMarkerValue(strings.TrimSpace(line), "ENGMODEL-OWNER-UNIT:"); ok {
			if x := strings.TrimSpace(v); x != "" {
				owner = x
				break
			}
		}
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

func scanCodeOwners(root string) map[string]string {
	out := map[string]string{}
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
		for _, line := range strings.Split(string(data), "\n") {
			if v, ok := extractMarkerValue(strings.TrimSpace(line), "ENGMODEL-OWNER-UNIT:"); ok {
				if x := strings.TrimSpace(v); x != "" {
					owner = x
					break
				}
			}
		}
		out[rel] = owner
		return nil
	})
	return out
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

func resolveSourcePath(baseDir, source string) string {
	s := strings.TrimSpace(source)
	if s == "" {
		return baseDir
	}
	if filepath.IsAbs(s) {
		return s
	}
	return filepath.Join(baseDir, s)
}

func extractMarkerValue(line, marker string) (string, bool) {
	x := strings.TrimSpace(line)
	x = strings.TrimPrefix(x, "//")
	x = strings.TrimPrefix(x, "#")
	x = strings.TrimPrefix(x, "/*")
	x = strings.TrimPrefix(x, "*")
	x = strings.TrimSpace(strings.TrimSuffix(x, "*/"))
	if strings.HasPrefix(strings.ToUpper(x), strings.ToUpper(marker)) {
		return strings.TrimSpace(x[len(marker):]), true
	}
	return "", false
}
