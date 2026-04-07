package engmodel

import (
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
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
