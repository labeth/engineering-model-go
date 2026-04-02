package view

import (
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	helmv2 "github.com/fluxcd/helm-controller/api/v2"
	kustomizev1 "github.com/fluxcd/kustomize-controller/api/v1"
	sourcev1 "github.com/fluxcd/source-controller/api/v1"
	"github.com/hashicorp/hcl/v2"
	"github.com/hashicorp/hcl/v2/hclparse"
	"github.com/hashicorp/hcl/v2/hclsyntax"
	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
	"github.com/zclconf/go-cty/cty"
	yamlv3 "gopkg.in/yaml.v3"
	"helm.sh/helm/v3/pkg/chart/loader"
	"sigs.k8s.io/yaml"
)

type deploymentGraph struct {
	Nodes map[string]Node
	Edges []Edge
}

func buildDeploymentView(vp model.Viewpoint, idx index, relationships []model.Relationship, b model.Bundle) (ProjectedView, []validate.Diagnostic) {
	graph, diags := extractDeploymentGraph(b)
	allowed := relationSet(vp.IncludeRelations)

	allEdges := append([]Edge(nil), graph.Edges...)
	// Optional architecture dependency overlay.
	for _, rel := range relationships {
		if strings.TrimSpace(rel.Type) != "depends_on" {
			continue
		}
		allEdges = append(allEdges, Edge{
			From:  rel.From,
			To:    rel.To,
			Type:  rel.Type,
			Label: edgeLabel(rel, idx.catalog),
		})
	}

	included := map[string]bool{}
	for _, root := range vp.Roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		if _, ok := graph.Nodes[root]; ok || hasArchitectureNode(root, idx) {
			included[root] = true
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code:     "view.deployment_root_not_found",
			Severity: validate.SeverityWarning,
			Message:  fmt.Sprintf("deployment root %q not found in extracted infra graph", root),
			Path:     "viewpoints",
		})
	}

	if len(included) == 0 {
		for id := range graph.Nodes {
			included[id] = true
		}
	}

	for changed := true; changed; {
		changed = false
		for _, e := range allEdges {
			if !allowed[e.Type] {
				continue
			}
			if included[e.From] || included[e.To] {
				if !included[e.From] {
					included[e.From] = true
					changed = true
				}
				if !included[e.To] {
					included[e.To] = true
					changed = true
				}
			}
		}
	}

	pv := ProjectedView{ID: vp.ID, Kind: vp.Kind, Title: vp.ID}
	for id := range included {
		if n, ok := graph.Nodes[id]; ok {
			pv.Nodes = append(pv.Nodes, n)
			continue
		}
		pv.Nodes = append(pv.Nodes, toNode(id, idx))
	}

	edgeSeen := map[string]bool{}
	for _, e := range allEdges {
		if !allowed[e.Type] {
			continue
		}
		if !included[e.From] || !included[e.To] {
			continue
		}
		key := edgeKey(e)
		if edgeSeen[key] {
			continue
		}
		edgeSeen[key] = true
		pv.Edges = append(pv.Edges, e)
	}

	return sortView(pv), validate.SortDiagnostics(diags)
}

func extractDeploymentGraph(b model.Bundle) (deploymentGraph, []validate.Diagnostic) {
	g := deploymentGraph{
		Nodes: map[string]Node{},
		Edges: []Edge{},
	}
	diags := []validate.Diagnostic{}
	edgeSeen := map[string]bool{}

	addNode := func(n Node) {
		if strings.TrimSpace(n.ID) == "" {
			return
		}
		if strings.TrimSpace(n.Label) == "" {
			n.Label = n.ID
		}
		if strings.TrimSpace(n.Kind) == "" {
			n.Kind = "unknown"
		}
		if _, ok := g.Nodes[n.ID]; ok {
			return
		}
		g.Nodes[n.ID] = n
	}
	addEdge := func(e Edge) {
		if strings.TrimSpace(e.From) == "" || strings.TrimSpace(e.To) == "" || strings.TrimSpace(e.Type) == "" {
			return
		}
		key := edgeKey(e)
		if edgeSeen[key] {
			return
		}
		edgeSeen[key] = true
		g.Edges = append(g.Edges, e)
	}

	modelRoot := filepath.Dir(b.ArchitecturePath)
	infraRoot := filepath.Join(modelRoot, "infra")
	if _, err := os.Stat(infraRoot); err != nil {
		diags = append(diags, validate.Diagnostic{
			Code:     "view.deployment_infra_missing",
			Severity: validate.SeverityWarning,
			Message:  fmt.Sprintf("deployment infra folder not found at %s", infraRoot),
			Path:     "infra",
		})
		return g, diags
	}

	envID, clusterID, nsIDs, tfDiags := extractTerraformNodes(filepath.Join(infraRoot, "terraform"), addNode, addEdge)
	diags = append(diags, tfDiags...)
	_ = nsIDs

	fluxDiags := extractFluxNodes(filepath.Join(infraRoot, "flux"), modelRoot, envID, clusterID, nsIDs, addNode, addEdge)
	diags = append(diags, fluxDiags...)

	return g, validate.SortDiagnostics(diags)
}

func extractTerraformNodes(terraformRoot string, addNode func(Node), addEdge func(Edge)) (envID, clusterID string, namespaceIDs map[string]bool, diags []validate.Diagnostic) {
	namespaceIDs = map[string]bool{}
	namespaceLabels := map[string]string{}
	files, err := collectFiles(terraformRoot, ".tf")
	if err != nil {
		diags = append(diags, validate.Diagnostic{
			Code:     "view.deployment_terraform_scan_failed",
			Severity: validate.SeverityWarning,
			Message:  fmt.Sprintf("failed scanning terraform files: %v", err),
			Path:     terraformRoot,
		})
		return "", "", namespaceIDs, diags
	}
	if len(files) == 0 {
		diags = append(diags, validate.Diagnostic{
			Code:     "view.deployment_terraform_missing",
			Severity: validate.SeverityWarning,
			Message:  "no terraform files found under infra/terraform",
			Path:     terraformRoot,
		})
		return "", "", namespaceIDs, diags
	}

	parser := hclparse.NewParser()
	environment := ""
	clusterName := ""
	clusterNameHint := ""

	for _, p := range files {
		f, parseDiags := parser.ParseHCLFile(p)
		if parseDiags.HasErrors() {
			diags = append(diags, validate.Diagnostic{
				Code:     "view.deployment_terraform_parse_failed",
				Severity: validate.SeverityWarning,
				Message:  fmt.Sprintf("failed parsing %s: %s", p, parseDiags.Error()),
				Path:     p,
			})
			continue
		}
		body, ok := f.Body.(*hclsyntax.Body)
		if !ok {
			continue
		}

		for _, block := range body.Blocks {
			switch block.Type {
			case "locals":
				if environment == "" {
					if v, ok := hclAttributeString(block.Body, "environment"); ok {
						environment = v
					}
				}
				if clusterNameHint == "" {
					if v, ok := hclAttributeString(block.Body, "cluster_name"); ok {
						clusterNameHint = v
					}
				}
			case "resource":
				if len(block.Labels) < 2 {
					continue
				}
				resourceType := strings.TrimSpace(block.Labels[0])
				resourceName := strings.TrimSpace(block.Labels[1])
				switch resourceType {
				case "aws_eks_cluster":
					if clusterName == "" {
						if v, ok := hclAttributeString(block.Body, "name"); ok {
							clusterName = v
						} else if clusterNameHint != "" {
							clusterName = clusterNameHint
						} else {
							clusterName = resourceName
						}
					}
				case "kubernetes_namespace":
					ns := extractTerraformNamespaceName(block.Body, resourceName)
					if ns != "" {
						nsID := deploymentID("NS", ns)
						namespaceIDs[nsID] = true
						namespaceLabels[nsID] = ns
					}
				}
			}
		}
	}

	if environment == "" {
		environment = "prod"
	}
	envID = deploymentID("ENV", environment)
	addNode(Node{ID: envID, Label: strings.ToLower(environment), Kind: "environment"})

	if clusterName == "" {
		diags = append(diags, validate.Diagnostic{
			Code:     "view.deployment_cluster_missing",
			Severity: validate.SeverityWarning,
			Message:  "no aws_eks_cluster resource found in terraform files",
			Path:     terraformRoot,
		})
	} else {
		clusterID = deploymentID("EKS", clusterName)
		addNode(Node{ID: clusterID, Label: clusterName, Kind: "cluster"})
		addEdge(Edge{From: clusterID, To: envID, Type: "part_of", Label: "part_of"})
	}

	nsList := make([]string, 0, len(namespaceIDs))
	for nsID := range namespaceIDs {
		nsList = append(nsList, nsID)
	}
	sort.Strings(nsList)
	for _, nsID := range nsList {
		label := namespaceLabels[nsID]
		if strings.TrimSpace(label) == "" {
			label = strings.ToLower(strings.TrimPrefix(nsID, "NS-"))
		}
		addNode(Node{ID: nsID, Label: label, Kind: "namespace"})
		if clusterID != "" {
			addEdge(Edge{From: nsID, To: clusterID, Type: "part_of", Label: "part_of"})
		}
	}

	return envID, clusterID, namespaceIDs, diags
}

func extractFluxNodes(
	fluxRoot, modelRoot, envID, clusterID string,
	namespaceIDs map[string]bool,
	addNode func(Node),
	addEdge func(Edge),
) []validate.Diagnostic {
	diags := []validate.Diagnostic{}
	files, err := collectFiles(fluxRoot, ".yaml", ".yml")
	if err != nil {
		return []validate.Diagnostic{{
			Code:     "view.deployment_flux_scan_failed",
			Severity: validate.SeverityWarning,
			Message:  fmt.Sprintf("failed scanning flux files: %v", err),
			Path:     fluxRoot,
		}}
	}
	if len(files) == 0 {
		return []validate.Diagnostic{{
			Code:     "view.deployment_flux_missing",
			Severity: validate.SeverityWarning,
			Message:  "no flux yaml files found under infra/flux",
			Path:     fluxRoot,
		}}
	}

	for _, path := range files {
		docs, err := decodeYAMLDocuments(path)
		if err != nil {
			diags = append(diags, validate.Diagnostic{
				Code:     "view.deployment_flux_decode_failed",
				Severity: validate.SeverityWarning,
				Message:  fmt.Sprintf("failed parsing %s: %v", path, err),
				Path:     path,
			})
			continue
		}

		for _, raw := range docs {
			var meta struct {
				APIVersion string `yaml:"apiVersion"`
				Kind       string `yaml:"kind"`
			}
			if err := yaml.Unmarshal(raw, &meta); err != nil {
				diags = append(diags, validate.Diagnostic{
					Code:     "view.deployment_flux_decode_failed",
					Severity: validate.SeverityWarning,
					Message:  fmt.Sprintf("failed decoding flux document metadata in %s: %v", path, err),
					Path:     path,
				})
				continue
			}
			kind := strings.TrimSpace(meta.Kind)
			if kind == "" {
				continue
			}

			switch kind {
			case "GitRepository":
				var obj sourcev1.GitRepository
				if err := yaml.UnmarshalStrict(raw, &obj); err != nil {
					diags = append(diags, validate.Diagnostic{
						Code:     "view.deployment_flux_decode_failed",
						Severity: validate.SeverityWarning,
						Message:  fmt.Sprintf("failed decoding GitRepository in %s: %v", path, err),
						Path:     path,
					})
					continue
				}
				name := strings.TrimSpace(obj.GetName())
				namespace := strings.TrimSpace(obj.GetNamespace())
				if name == "" {
					continue
				}
				srcID := deploymentID("GITSRC", name)
				addNode(Node{ID: srcID, Label: name, Kind: "git_source"})
				if namespace != "" {
					nsID := deploymentID("NS", namespace)
					addNode(Node{ID: nsID, Label: namespace, Kind: "namespace"})
					namespaceIDs[nsID] = true
					if clusterID != "" {
						addEdge(Edge{From: nsID, To: clusterID, Type: "part_of", Label: "part_of"})
					}
				}

			case "Kustomization":
				var obj kustomizev1.Kustomization
				if err := yaml.UnmarshalStrict(raw, &obj); err != nil {
					diags = append(diags, validate.Diagnostic{
						Code:     "view.deployment_flux_decode_failed",
						Severity: validate.SeverityWarning,
						Message:  fmt.Sprintf("failed decoding Kustomization in %s: %v", path, err),
						Path:     path,
					})
					continue
				}
				name := strings.TrimSpace(obj.GetName())
				namespace := strings.TrimSpace(obj.GetNamespace())
				if name == "" {
					continue
				}
				kID := deploymentID("FLUXKUS", name)
				addNode(Node{ID: kID, Label: name, Kind: "flux_kustomization"})
				if namespace != "" {
					nsID := deploymentID("NS", namespace)
					addNode(Node{ID: nsID, Label: namespace, Kind: "namespace"})
					namespaceIDs[nsID] = true
					if clusterID != "" {
						addEdge(Edge{From: nsID, To: clusterID, Type: "part_of", Label: "part_of"})
					}
					addEdge(Edge{From: kID, To: nsID, Type: "runs_in", Label: "runs_in"})
				}
				sourceName := strings.TrimSpace(obj.Spec.SourceRef.Name)
				if sourceName != "" {
					srcID := deploymentID("GITSRC", sourceName)
					addNode(Node{ID: srcID, Label: sourceName, Kind: "git_source"})
					addEdge(Edge{From: kID, To: srcID, Type: "depends_on", Label: "depends_on source"})
				}

			case "HelmRelease":
				var obj helmv2.HelmRelease
				if err := yaml.UnmarshalStrict(raw, &obj); err != nil {
					diags = append(diags, validate.Diagnostic{
						Code:     "view.deployment_flux_decode_failed",
						Severity: validate.SeverityWarning,
						Message:  fmt.Sprintf("failed decoding HelmRelease in %s: %v", path, err),
						Path:     path,
					})
					continue
				}
				name := strings.TrimSpace(obj.GetName())
				namespace := strings.TrimSpace(obj.GetNamespace())
				annotations := obj.GetAnnotations()
				if name == "" {
					continue
				}
				hrID := deploymentID("HELMREL", name)
				addNode(Node{ID: hrID, Label: name, Kind: "helm_release"})

				ns := namespace
				if ns == "" {
					ns = "default"
				}
				nsID := deploymentID("NS", ns)
				addNode(Node{ID: nsID, Label: ns, Kind: "namespace"})
				namespaceIDs[nsID] = true
				addEdge(Edge{From: hrID, To: nsID, Type: "runs_in", Label: "runs_in"})
				if clusterID != "" {
					addEdge(Edge{From: nsID, To: clusterID, Type: "part_of", Label: "part_of"})
				}
				if envID != "" && clusterID != "" {
					addEdge(Edge{From: clusterID, To: envID, Type: "part_of", Label: "part_of"})
				}

				deployTargets := splitCSV(annotations["trace.dev/part-of"])
				if len(deployTargets) == 0 {
					deployTargets = splitCSV(annotations["trace.dev/deploys"])
				}
				for _, target := range deployTargets {
					addEdge(Edge{From: hrID, To: target, Type: "deploys", Label: "deploys"})
				}

				managedBy := strings.TrimSpace(annotations["trace.dev/managed-by"])
				if managedBy != "" {
					mgrID := deploymentID("FLUXKUS", managedBy)
					addNode(Node{ID: mgrID, Label: managedBy, Kind: "flux_kustomization"})
					addEdge(Edge{From: mgrID, To: hrID, Type: "manages", Label: "manages"})
				}

				chartPath := ""
				sourceName := ""
				if obj.Spec.Chart != nil {
					chartPath = strings.TrimSpace(obj.Spec.Chart.Spec.Chart)
					sourceName = strings.TrimSpace(obj.Spec.Chart.Spec.SourceRef.Name)
				}
				if chartPath != "" {
					chartID := deploymentID("HELMCHART", name)
					chartLabel := chartPath
					if chartAbs, ok := resolveHelmChartPath(modelRoot, chartPath); ok {
						ch, err := loader.Load(chartAbs)
						if err != nil {
							diags = append(diags, validate.Diagnostic{
								Code:     "view.deployment_helm_chart_load_failed",
								Severity: validate.SeverityWarning,
								Message:  fmt.Sprintf("failed loading helm chart %q for HelmRelease %q: %v", chartAbs, name, err),
								Path:     path,
							})
						} else if ch != nil && ch.Metadata != nil && strings.TrimSpace(ch.Metadata.Name) != "" {
							chartLabel = ch.Metadata.Name
						}
					}
					addNode(Node{ID: chartID, Label: chartLabel, Kind: "helm_chart"})
					addEdge(Edge{From: hrID, To: chartID, Type: "uses", Label: "uses chart"})
				}

				if sourceName != "" {
					srcID := deploymentID("GITSRC", sourceName)
					addNode(Node{ID: srcID, Label: sourceName, Kind: "git_source"})
					addEdge(Edge{From: hrID, To: srcID, Type: "depends_on", Label: "depends_on source"})
				}
			}
		}
	}

	return diags
}

func hclAttributeString(body *hclsyntax.Body, attrName string) (string, bool) {
	if body == nil {
		return "", false
	}
	attr, ok := body.Attributes[attrName]
	if !ok || attr == nil {
		return "", false
	}
	return hclExpressionString(attr.Expr)
}

func hclExpressionString(expr hclsyntax.Expression) (string, bool) {
	if expr == nil {
		return "", false
	}
	val, diags := expr.Value(&hcl.EvalContext{})
	if diags.HasErrors() || !val.IsKnown() || val.IsNull() {
		return "", false
	}
	if val.Type() == cty.String {
		v := strings.TrimSpace(val.AsString())
		return v, v != ""
	}
	return "", false
}

func extractTerraformNamespaceName(body *hclsyntax.Body, fallback string) string {
	if v, ok := hclAttributeString(body, "name"); ok {
		return v
	}
	if body != nil {
		for _, block := range body.Blocks {
			if block.Type != "metadata" {
				continue
			}
			if v, ok := hclAttributeString(block.Body, "name"); ok {
				return v
			}
		}
	}
	return strings.TrimSpace(fallback)
}

func decodeYAMLDocuments(path string) ([][]byte, error) {
	b, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	dec := yamlv3.NewDecoder(bytes.NewReader(b))
	out := make([][]byte, 0)
	for {
		var node yamlv3.Node
		if err := dec.Decode(&node); err != nil {
			if err == io.EOF {
				break
			}
			return nil, err
		}
		if len(node.Content) == 0 {
			continue
		}
		raw, err := yamlv3.Marshal(&node)
		if err != nil {
			return nil, err
		}
		out = append(out, raw)
	}
	return out, nil
}

func resolveHelmChartPath(modelRoot, chartPath string) (string, bool) {
	p := strings.TrimSpace(chartPath)
	if p == "" {
		return "", false
	}
	if filepath.IsAbs(p) {
		return p, true
	}
	if strings.HasPrefix(p, "./") || strings.HasPrefix(p, "../") {
		return filepath.Clean(filepath.Join(modelRoot, p)), true
	}
	candidate := filepath.Clean(filepath.Join(modelRoot, p))
	if _, err := os.Stat(candidate); err == nil {
		return candidate, true
	}
	return "", false
}

func collectFiles(root string, suffixes ...string) ([]string, error) {
	if _, err := os.Stat(root); err != nil {
		return nil, nil
	}
	list := []string{}
	err := filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			return nil
		}
		for _, s := range suffixes {
			if strings.HasSuffix(strings.ToLower(path), strings.ToLower(s)) {
				list = append(list, path)
				break
			}
		}
		return nil
	})
	sort.Strings(list)
	return list, err
}

func splitCSV(raw string) []string {
	if strings.TrimSpace(raw) == "" {
		return nil
	}
	parts := strings.Split(raw, ",")
	out := make([]string, 0, len(parts))
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			out = append(out, p)
		}
	}
	sort.Strings(out)
	uniq := make([]string, 0, len(out))
	prev := ""
	for _, p := range out {
		if p != prev {
			uniq = append(uniq, p)
			prev = p
		}
	}
	return uniq
}

func edgeKey(e Edge) string {
	return e.From + "|" + e.To + "|" + e.Type + "|" + e.Label
}

func deploymentID(prefix, raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		raw = "item"
	}
	re := regexp.MustCompile(`[^A-Za-z0-9]+`)
	id := strings.Trim(re.ReplaceAllString(strings.ToUpper(raw), "-"), "-")
	if id == "" {
		id = "ITEM"
	}
	return prefix + "-" + id
}

func hasArchitectureNode(id string, idx index) bool {
	if _, ok := idx.people[id]; ok {
		return true
	}
	if _, ok := idx.systems[id]; ok {
		return true
	}
	if _, ok := idx.containers[id]; ok {
		return true
	}
	if _, ok := idx.components[id]; ok {
		return true
	}
	return false
}
