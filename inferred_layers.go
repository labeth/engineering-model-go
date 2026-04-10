package engmodel

import (
	"bufio"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"unicode"

	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
	"gopkg.in/yaml.v3"
)

type inferredRuntimeItem struct {
	Name        string
	Kind        string
	Owner       string
	Description string
	Source      string
	Ports       []string
}

type inferredCodeItem struct {
	Element     string
	Kind        string
	Owner       string
	Description string
	Source      string
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
					it.Owner = resolveRuntimeOwner(it, bundle.Architecture.AuthoredArchitecture.FunctionalUnits)
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
					it.Owner = resolveRuntimeOwner(it, bundle.Architecture.AuthoredArchitecture.FunctionalUnits)
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
	descHints, hintErr := parseTerraformRuntimeDescriptions(path)
	if hintErr != nil {
		return nil, []validate.Diagnostic{{
			Code:     "runtime.terraform_description_parse_failed",
			Severity: validate.SeverityWarning,
			Message:  fmt.Sprintf("parse terraform runtime description hints: %v", hintErr),
			Path:     path,
		}}
	}

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
			Name:        name,
			Kind:        kind,
			Owner:       inferOwnerByConvention(kind, name),
			Description: strings.TrimSpace(descHints[rtype+"|"+rname]),
			Source:      filepath.ToSlash(path),
		})
	}
	return out, nil
}

var terraformResourceLinePattern = regexp.MustCompile(`^\s*resource\s+"([^"]+)"\s+"([^"]+)"\s*\{`)

func parseTerraformRuntimeDescriptions(path string) (map[string]string, error) {
	f, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	defer f.Close()

	out := map[string]string{}
	scanner := bufio.NewScanner(f)
	pending := ""
	for scanner.Scan() {
		line := scanner.Text()
		if desc, ok := extractRuntimeDescriptionMarker(line); ok {
			pending = desc
			continue
		}

		m := terraformResourceLinePattern.FindStringSubmatch(line)
		if len(m) == 3 {
			if strings.TrimSpace(pending) != "" {
				out[strings.TrimSpace(m[1])+"|"+strings.TrimSpace(m[2])] = strings.TrimSpace(pending)
			}
			pending = ""
			continue
		}

		trimmed := strings.TrimSpace(line)
		if trimmed == "" || strings.HasPrefix(trimmed, "#") || strings.HasPrefix(trimmed, "//") || strings.HasPrefix(trimmed, "/*") || strings.HasPrefix(trimmed, "*") {
			continue
		}
		pending = ""
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

func normalizeTerraformKind(resourceType string) string {
	t := strings.TrimSpace(strings.ToLower(resourceType))
	switch {
	case strings.Contains(t, "lambda_function"):
		return "lambda_function"
	case strings.Contains(t, "cloudwatch_event_rule") || strings.Contains(t, "eventbridge_rule"):
		return "eventbridge_rule"
	case strings.Contains(t, "cloudwatch_event_target") || strings.Contains(t, "eventbridge_target"):
		return "eventbridge_target"
	case strings.Contains(t, "sqs_queue"):
		return "queue"
	case strings.Contains(t, "sns_topic"):
		return "topic"
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

func resolveRuntimeOwner(item inferredRuntimeItem, units []model.FunctionalUnit) string {
	owner := strings.TrimSpace(item.Owner)
	if owner != "" && owner != "unresolved" {
		return owner
	}

	owner = inferOwnerByConvention(item.Kind, item.Name)
	if owner != "unresolved" {
		return owner
	}

	owner = inferOwnerByFunctionalUnits(item.Name, units)
	if owner != "" {
		return owner
	}
	return "unresolved"
}

func inferOwnerByFunctionalUnits(name string, units []model.FunctionalUnit) string {
	nameTokens := normalizeOwnerTokens(name)
	if len(nameTokens) == 0 {
		return ""
	}

	bestID := ""
	bestScore := 0
	bestTokenCount := 0
	for _, fu := range units {
		candidateTokens := normalizeOwnerTokens(fu.ID + " " + fu.Name)
		if len(candidateTokens) == 0 {
			continue
		}
		score := tokenMatchScore(nameTokens, candidateTokens)
		if score < 2 {
			continue
		}
		if score > bestScore || (score == bestScore && len(candidateTokens) > bestTokenCount) || (score == bestScore && len(candidateTokens) == bestTokenCount && strings.TrimSpace(fu.ID) < bestID) {
			bestID = strings.TrimSpace(fu.ID)
			bestScore = score
			bestTokenCount = len(candidateTokens)
		}
	}
	return bestID
}

func tokenMatchScore(left, right []string) int {
	score := 0
	for _, l := range left {
		for _, r := range right {
			if l == r {
				score += 2
				break
			}
			if len(l) >= 5 && len(r) >= 5 && (strings.HasPrefix(l, r) || strings.HasPrefix(r, l)) {
				score++
				break
			}
		}
	}
	return score
}

func normalizeOwnerTokens(s string) []string {
	parts := strings.FieldsFunc(strings.ToLower(strings.TrimSpace(s)), func(r rune) bool {
		return !(unicode.IsLetter(r) || unicode.IsDigit(r))
	})
	out := make([]string, 0, len(parts))
	seen := map[string]bool{}
	for _, part := range parts {
		token := normalizeOwnerToken(part)
		if token == "" {
			continue
		}
		if seen[token] {
			continue
		}
		seen[token] = true
		out = append(out, token)
	}
	return out
}

func normalizeOwnerToken(token string) string {
	x := strings.TrimSpace(token)
	if len(x) < 3 {
		return ""
	}

	switch {
	case strings.HasPrefix(x, "publicat"), strings.HasPrefix(x, "publish"):
		x = "publish"
	case strings.HasPrefix(x, "orchestrat"):
		x = "orchestr"
	case strings.HasPrefix(x, "configur"), strings.HasPrefix(x, "config"), strings.HasPrefix(x, "cfg"):
		x = "config"
	}

	suffixes := []string{"ation", "ition", "tion", "sion", "ment", "ness", "izer", "iser", "ized", "ised", "ing", "ers", "ors", "er", "or", "ed", "es", "s"}
	for _, suffix := range suffixes {
		if strings.HasSuffix(x, suffix) && len(x)-len(suffix) >= 4 {
			x = strings.TrimSuffix(x, suffix)
			break
		}
	}
	return x
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
		description := ""
		if doc.Metadata.Annotations != nil {
			if x := strings.TrimSpace(doc.Metadata.Annotations["engmodel.dev/owner-unit"]); x != "" {
				owner = x
			}
			if x := strings.TrimSpace(doc.Metadata.Annotations["engmodel.dev/runtime-description"]); x != "" {
				description = x
			} else if x := strings.TrimSpace(doc.Metadata.Annotations["engmodel.dev/description"]); x != "" {
				description = x
			}
		}
		if owner == "unresolved" {
			owner = inferOwnerByConvention(strings.ToLower(kind), name)
		}
		out = append(out, inferredRuntimeItem{
			Name:        name,
			Kind:        strings.ToLower(kind),
			Owner:       owner,
			Description: description,
			Source:      filepath.ToSlash(path),
			Ports:       ports,
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

func extractRuntimeDescriptionMarker(line string) (string, bool) {
	markers := []string{
		"engmodel:runtime-description:",
		"engmodel:runtime-description",
		"engmodel.runtime.description:",
		"engmodel.runtime.description",
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
