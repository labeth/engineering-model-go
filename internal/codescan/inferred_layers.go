// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

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
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
	"gopkg.in/yaml.v3"
)

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type RuntimeItem struct {
	Name        string
	Kind        string
	Owner       string
	Description string
	Source      string
	Ports       []string
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type CodeItem struct {
	Element      string
	Kind         string
	Owner        string
	Description  string
	Source       string
	AbsPath      string // absolute file path (symbol items), for unambiguous model-boundary scoping
	EvidencePath string // model-relative path for portable publication evidence
	Implements   []string
	ModelLinks   []string
}

// TRLC-LINKS: REQ-EMG-010
func ModelRootDir(bundle model.Bundle) string {
	if bundle.ManifestPath != "" {
		return filepath.Dir(bundle.ManifestPath)
	}
	return filepath.Dir(bundle.ArchitecturePath)
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func InferRuntimeItems(bundle model.Bundle) ([]RuntimeItem, []validate.Diagnostic) {
	inf := &runtimeInference{
		baseDir: ModelRootDir(bundle),
		units:   bundle.Architecture.AuthoredArchitecture.FunctionalUnits,
		seen:    map[string]bool{},
		items:   []RuntimeItem{},
		diags:   []validate.Diagnostic{},
	}
	for _, src := range bundle.Architecture.InferenceHints.RuntimeSources {
		inf.walkSource(resolveSourcePath(inf.baseDir, src))
	}
	sortRuntimeItems(inf.items)
	return inf.items, validate.SortDiagnostics(inf.diags)
}

// runtimeInference accumulates the runtime items and diagnostics of the runtime sources.
type runtimeInference struct {
	baseDir string
	units   []model.FunctionalUnit
	seen    map[string]bool
	items   []RuntimeItem
	diags   []validate.Diagnostic
}

// walkSource walks one runtime source; a failed walk is a warning.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (r *runtimeInference) walkSource(root string) {
	parsedBySource := map[string]bool{}
	walkErr := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if !d.IsDir() {
			r.visitFile(path, parsedBySource)
		}
		return nil
	})
	if walkErr != nil {
		r.diags = append(r.diags, validate.Diagnostic{
			Code:     "runtime.source_walk_failed",
			Severity: validate.SeverityWarning,
			Message:  fmt.Sprintf("failed to walk runtime source %q: %v", root, walkErr),
			Path:     "inferenceHints.runtimeSources",
		})
	}
}

// visitFile infers the runtime items of one file by its extension: Terraform
// resources, Kubernetes manifests (and, when no manifest item names the file,
// a deployment artifact) or a JSON deployment artifact.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (r *runtimeInference) visitFile(path string, parsedBySource map[string]bool) {
	switch strings.ToLower(filepath.Ext(path)) {
	case ".tf":
		tfItems, tfDiags := parseTerraformRuntime(path)
		r.diags = append(r.diags, tfDiags...)
		r.addParsed(tfItems, parsedBySource)
	case ".yaml", ".yml":
		yItems, yDiags := parseManifestRuntime(path)
		r.diags = append(r.diags, yDiags...)
		r.addParsed(yItems, parsedBySource)
		if !parsedBySource[filepath.ToSlash(path)] {
			r.addAll(inferDeploymentArtifactItems(r.baseDir, path))
		}
	case ".json":
		r.addAll(inferDeploymentArtifactItems(r.baseDir, path))
	}
}

// addParsed adds parsed items and records their sources as parsed.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (r *runtimeInference) addParsed(items []RuntimeItem, parsedBySource map[string]bool) {
	for _, it := range items {
		parsedBySource[filepath.ToSlash(strings.TrimSpace(it.Source))] = true
		r.add(it)
	}
}

// addAll adds every item.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (r *runtimeInference) addAll(items []RuntimeItem) {
	for _, it := range items {
		r.add(it)
	}
}

// add resolves the owner of an item and adds it unless an equal item exists.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (r *runtimeInference) add(it RuntimeItem) {
	it.Owner = resolveRuntimeOwner(it, r.units)
	key := it.Source + "|" + it.Kind + "|" + it.Name + "|" + it.Owner
	if r.seen[key] {
		return
	}
	r.seen[key] = true
	r.items = append(r.items, it)
}

// sortRuntimeItems orders items by kind, name and source.
// TRLC-LINKS: REQ-EMG-010
func sortRuntimeItems(items []RuntimeItem) {
	sort.SliceStable(items, func(i, j int) bool {
		if items[i].Kind != items[j].Kind {
			return items[i].Kind < items[j].Kind
		}
		if items[i].Name != items[j].Name {
			return items[i].Name < items[j].Name
		}
		return items[i].Source < items[j].Source
	})
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
// TRLC-LINKS: REQ-EMG-010
func parseTerraformRuntime(path string) ([]RuntimeItem, []validate.Diagnostic) {
	descHints, hintErr := parseTerraformRuntimeDescriptions(path)
	if hintErr != nil {
		return nil, []validate.Diagnostic{runtimeWarning("runtime.terraform_description_parse_failed", fmt.Sprintf("parse terraform runtime description hints: %v", hintErr), path)}
	}
	content, failed := terraformResourceBlocks(path)
	if failed != nil {
		return nil, []validate.Diagnostic{*failed}
	}
	out := []RuntimeItem{}
	for _, b := range content.Blocks {
		if item, ok := terraformRuntimeItem(b, descHints, path); ok {
			out = append(out, item)
		}
	}
	return out, nil
}

// runtimeWarning is a warning diagnostic of runtime inference.
// TRLC-LINKS: REQ-EMG-010
func runtimeWarning(code, message, path string) validate.Diagnostic {
	return validate.Diagnostic{Code: code, Severity: validate.SeverityWarning, Message: message, Path: path}
}

// terraformResourceBlocks parses a Terraform file and returns its resource blocks, or
// the warning when the file does not parse.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func terraformResourceBlocks(path string) (*hcl.BodyContent, *validate.Diagnostic) {
	file, diags := hclparse.NewParser().ParseHCLFile(path)
	if diags.HasErrors() {
		d := runtimeWarning("runtime.terraform_parse_failed", diags.Error(), path)
		return nil, &d
	}
	schema := &hcl.BodySchema{
		Blocks: []hcl.BlockHeaderSchema{
			{Type: "resource", LabelNames: []string{"type", "name"}},
		},
	}
	content, _, cDiags := file.Body.PartialContent(schema)
	if cDiags.HasErrors() {
		d := runtimeWarning("runtime.terraform_schema_failed", cDiags.Error(), path)
		return nil, &d
	}
	return content, nil
}

// terraformRuntimeItem is the runtime item of one resource block; false when it has no labels.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func terraformRuntimeItem(b *hcl.Block, descHints map[string]string, path string) (RuntimeItem, bool) {
	rtype, rname := "", ""
	if len(b.Labels) > 0 {
		rtype = strings.TrimSpace(b.Labels[0])
	}
	if len(b.Labels) > 1 {
		rname = strings.TrimSpace(b.Labels[1])
	}
	if rtype == "" && rname == "" {
		return RuntimeItem{}, false
	}
	kind := normalizeTerraformKind(rtype)
	name := rname
	if name == "" {
		name = rtype
	}
	return RuntimeItem{
		Name:        name,
		Kind:        kind,
		Owner:       inferOwnerByConvention(kind, name),
		Description: strings.TrimSpace(descHints[rtype+"|"+rname]),
		Source:      filepath.ToSlash(path),
	}, true
}

var terraformResourceLinePattern = regexp.MustCompile(`^\s*resource\s+"([^"]+)"\s+"([^"]+)"\s*\{`)

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
// TRLC-LINKS: REQ-EMG-010
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
		pending = recordTerraformDescription(scanner.Text(), pending, out)
	}
	if err := scanner.Err(); err != nil {
		return nil, err
	}
	return out, nil
}

// recordTerraformDescription handles one line: a description marker becomes the
// pending description, a resource line takes it, a comment or blank line keeps it
// and any other line drops it. It returns the pending description.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func recordTerraformDescription(line, pending string, out map[string]string) string {
	if desc, ok := extractRuntimeDescriptionMarker(line); ok {
		return desc
	}
	if m := terraformResourceLinePattern.FindStringSubmatch(line); len(m) == 3 {
		if strings.TrimSpace(pending) != "" {
			out[strings.TrimSpace(m[1])+"|"+strings.TrimSpace(m[2])] = strings.TrimSpace(pending)
		}
		return ""
	}
	if isCommentOrBlank(strings.TrimSpace(line)) {
		return pending
	}
	return ""
}

// isCommentOrBlank reports whether a trimmed line is empty or starts a comment.
// TRLC-LINKS: REQ-EMG-010
func isCommentOrBlank(trimmed string) bool {
	if trimmed == "" {
		return true
	}
	for _, prefix := range []string{"#", "//", "/*", "*"} {
		if strings.HasPrefix(trimmed, prefix) {
			return true
		}
	}
	return false
}

// terraformKindRule maps a resource type to a runtime kind when it contains,
// for any one alternative, every term of that alternative.
type terraformKindRule struct {
	kind         string
	alternatives [][]string
}

// terraformKindRules are matched in order against the lowercased resource type.
var terraformKindRules = []terraformKindRule{
	{"lambda_function", [][]string{{"lambda_function"}}},
	{"eventbridge_rule", [][]string{{"cloudwatch_event_rule"}, {"eventbridge_rule"}}},
	{"eventbridge_target", [][]string{{"cloudwatch_event_target"}, {"eventbridge_target"}}},
	{"queue", [][]string{{"sqs_queue"}}},
	{"topic", [][]string{{"sns_topic"}}},
	{"cluster", [][]string{{"eks", "cluster"}}},
	{"namespace", [][]string{{"namespace"}}},
}

// TRLC-LINKS: REQ-EMG-010
func normalizeTerraformKind(resourceType string) string {
	t := strings.TrimSpace(strings.ToLower(resourceType))
	for _, rule := range terraformKindRules {
		if rule.matches(t) {
			return rule.kind
		}
	}
	return "terraform_resource"
}

// matches reports whether the type contains all terms of some alternative.
// TRLC-LINKS: REQ-EMG-010
func (r terraformKindRule) matches(t string) bool {
	for _, terms := range r.alternatives {
		if containsAll(t, terms) {
			return true
		}
	}
	return false
}

// containsAll reports whether s contains every term.
// TRLC-LINKS: REQ-EMG-010
func containsAll(s string, terms []string) bool {
	for _, term := range terms {
		if !strings.Contains(s, term) {
			return false
		}
	}
	return true
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func inferOwnerByConvention(kind, name string) string {
	k := strings.ToLower(strings.TrimSpace(kind))
	n := strings.ToLower(strings.TrimSpace(name))
	switch k {
	case "cluster":
		return "FU-CLUSTER-PROVISIONING"
	case "gitrepository", "kustomization":
		return "FU-GITOPS-OPERATIONS"
	case "namespace":
		return namespaceOwner(n)
	default:
		return "unresolved"
	}
}

// namespaceOwner is the conventional owner of a namespace by its lowercased name.
// TRLC-LINKS: REQ-EMG-010
func namespaceOwner(name string) string {
	switch name {
	case "flux-system", "flux_system":
		return "FU-GITOPS-OPERATIONS"
	case "payments":
		return "FU-CHECKOUT"
	case "risk":
		return "FU-RISK-SCORING"
	default:
		return "unresolved"
	}
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func resolveRuntimeOwner(item RuntimeItem, units []model.FunctionalUnit) string {
	owner := strings.TrimSpace(item.Owner)
	declared := owner != "" && owner != "unresolved"
	if declared && functionalUnitExists(owner, units) {
		return owner
	}
	if conventional := inferOwnerByConvention(item.Kind, item.Name); existingUnitOwner(conventional, units) {
		return conventional
	}
	inferred := inferOwnerByFunctionalUnits(item.Name+" "+item.Source, units)
	if inferred != "" || !declared {
		return inferred
	}
	return owner
}

// existingUnitOwner reports whether an owner is resolved and names a functional unit.
// TRLC-LINKS: REQ-EMG-010
func existingUnitOwner(owner string, units []model.FunctionalUnit) bool {
	return owner != "unresolved" && functionalUnitExists(owner, units)
}

// TRLC-LINKS: REQ-EMG-010
func functionalUnitExists(id string, units []model.FunctionalUnit) bool {
	id = strings.TrimSpace(id)
	if id == "" {
		return false
	}
	for _, unit := range units {
		if strings.TrimSpace(unit.ID) == id {
			return true
		}
	}
	return false
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
// TRLC-LINKS: REQ-EMG-010
func inferDeploymentArtifactItems(baseDir, path string) []RuntimeItem {
	if !isDeploymentEvidencePath(path) {
		return nil
	}
	rel := deploymentArtifactDisplayPath(baseDir, path)
	return []RuntimeItem{{
		Name:        rel,
		Kind:        "deployment_artifact",
		Owner:       "unresolved",
		Description: "Inferred deployment configuration evidence artifact.",
		Source:      filepath.ToSlash(path),
	}}
}

// TRLC-LINKS: REQ-EMG-010
func isDeploymentEvidencePath(path string) bool {
	p := filepath.ToSlash(strings.ToLower(strings.TrimSpace(path)))
	if p == "" {
		return false
	}
	ext := strings.ToLower(filepath.Ext(p))
	if ext != ".yaml" && ext != ".yml" && ext != ".json" {
		return false
	}
	return strings.Contains(p, "/deploy/oci/") || strings.Contains(p, "/deploy/kairos/") || strings.Contains(p, "/infra/flux/")
}

// TRLC-LINKS: REQ-EMG-010
func deploymentArtifactDisplayPath(baseDir, path string) string {
	abs := strings.TrimSpace(path)
	if abs == "" {
		return "deployment-artifact"
	}
	if rel, err := filepath.Rel(baseDir, abs); err == nil {
		rel = filepath.ToSlash(strings.TrimSpace(rel))
		if rel != "" && rel != "." && !strings.HasPrefix(rel, "../") {
			return rel
		}
	}
	return filepath.ToSlash(abs)
}

// ownerCandidate is a functional unit scored against a runtime item name.
type ownerCandidate struct {
	id     string
	score  int
	tokens int
}

// better reports whether the candidate outranks best: by score, then by token
// count, then by the smaller id.
// TRLC-LINKS: REQ-EMG-010
func (c ownerCandidate) better(best ownerCandidate) bool {
	if c.score != best.score {
		return c.score > best.score
	}
	if c.tokens != best.tokens {
		return c.tokens > best.tokens
	}
	return c.id < best.id
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func inferOwnerByFunctionalUnits(name string, units []model.FunctionalUnit) string {
	nameTokens := normalizeOwnerTokens(name)
	if len(nameTokens) == 0 {
		return ""
	}
	best := ownerCandidate{}
	for _, fu := range units {
		cand, ok := scoreOwnerCandidate(nameTokens, fu)
		if ok && cand.better(best) {
			best = cand
		}
	}
	return best.id
}

// scoreOwnerCandidate scores a functional unit against the name tokens; false
// when it has no tokens or scores below two.
// TRLC-LINKS: REQ-EMG-010
func scoreOwnerCandidate(nameTokens []string, fu model.FunctionalUnit) (ownerCandidate, bool) {
	candidateTokens := normalizeOwnerTokens(fu.ID + " " + fu.Name)
	if len(candidateTokens) == 0 {
		return ownerCandidate{}, false
	}
	score := tokenMatchScore(nameTokens, candidateTokens)
	return ownerCandidate{id: strings.TrimSpace(fu.ID), score: score, tokens: len(candidateTokens)}, score >= 2
}

// TRLC-LINKS: REQ-EMG-010
func tokenMatchScore(left, right []string) int {
	score := 0
	for _, l := range left {
		score += tokenScore(l, right)
	}
	return score
}

// tokenScore is 2 when the token equals one of the candidates, else 1 when it
// shares a prefix with one of them (both of five characters or more), else 0.
// The first candidate that matches either way decides.
// TRLC-LINKS: REQ-EMG-010
func tokenScore(l string, right []string) int {
	for _, r := range right {
		if l == r {
			return 2
		}
		if sharesLongPrefix(l, r) {
			return 1
		}
	}
	return 0
}

// sharesLongPrefix reports whether two tokens of five characters or more are
// prefixes of each other.
// TRLC-LINKS: REQ-EMG-010
func sharesLongPrefix(l, r string) bool {
	long := len(l) >= 5 && len(r) >= 5
	return long && (strings.HasPrefix(l, r) || strings.HasPrefix(r, l))
}

// TRLC-LINKS: REQ-EMG-010
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

// TRLC-LINKS: REQ-EMG-010
func normalizeOwnerToken(token string) string {
	x := strings.TrimSpace(token)
	if len(x) < 3 {
		return ""
	}
	return stripOwnerSuffix(canonicalOwnerStem(x))
}

// canonicalOwnerStem folds the publish, orchestrate and configure families to one stem.
// TRLC-LINKS: REQ-EMG-010
func canonicalOwnerStem(x string) string {
	switch {
	case hasAnyPrefix(x, "publicat", "publish"):
		return "publish"
	case hasAnyPrefix(x, "orchestrat"):
		return "orchestr"
	case hasAnyPrefix(x, "configur", "config", "cfg"):
		return "config"
	}
	return x
}

// hasAnyPrefix reports whether s starts with one of the prefixes.
// TRLC-LINKS: REQ-EMG-010
func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, prefix := range prefixes {
		if strings.HasPrefix(s, prefix) {
			return true
		}
	}
	return false
}

// ownerSuffixes are tried in order; the first that leaves four characters is stripped.
var ownerSuffixes = []string{"ation", "ition", "tion", "sion", "ment", "ness", "izer", "iser", "ized", "ised", "ing", "ers", "ors", "er", "or", "ed", "es", "s"}

// stripOwnerSuffix removes the first matching suffix that leaves at least four characters.
// TRLC-LINKS: REQ-EMG-010
func stripOwnerSuffix(x string) string {
	for _, suffix := range ownerSuffixes {
		if strings.HasSuffix(x, suffix) && len(x)-len(suffix) >= 4 {
			return strings.TrimSuffix(x, suffix)
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

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
// TRLC-LINKS: REQ-EMG-010
func parseManifestRuntime(path string) ([]RuntimeItem, []validate.Diagnostic) {
	if isUnrenderedTemplate(path) {
		return nil, nil
	}
	f, err := os.Open(path)
	if err != nil {
		return nil, []validate.Diagnostic{runtimeWarning("runtime.manifest_open_failed", fmt.Sprintf("open manifest: %v", err), path)}
	}
	defer f.Close()

	dec := yaml.NewDecoder(f)
	out := []RuntimeItem{}
	for {
		var doc manifestObject
		err := dec.Decode(&doc)
		if err == io.EOF {
			return out, nil
		}
		if err != nil {
			return out, []validate.Diagnostic{runtimeWarning("runtime.manifest_parse_failed", fmt.Sprintf("decode manifest: %v", err), path)}
		}
		if item, ok := manifestRuntimeItem(doc, path); ok {
			out = append(out, item)
		}
	}
}

// isUnrenderedTemplate reports whether a readable file holds Helm template
// braces; such files are not concrete runtime manifests.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func isUnrenderedTemplate(path string) bool {
	raw, err := os.ReadFile(path)
	if err != nil {
		return false
	}
	txt := string(raw)
	return strings.Contains(txt, "{{") && strings.Contains(txt, "}}")
}

// manifestRuntimeItem is the runtime item of one manifest document; false when
// it lacks a kind or name.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func manifestRuntimeItem(doc manifestObject, path string) (RuntimeItem, bool) {
	kind := strings.TrimSpace(doc.Kind)
	name := strings.TrimSpace(doc.Metadata.Name)
	if kind == "" || name == "" {
		return RuntimeItem{}, false
	}
	ports := normalizePorts(doc.Spec.Ports)
	if len(ports) == 0 && strings.EqualFold(kind, "HelmRelease") {
		ports = portsFromHelmValues(doc.Spec.Values)
	}
	if ns := strings.TrimSpace(doc.Metadata.Namespace); ns != "" {
		name = fmt.Sprintf("%s/%s", ns, name)
	}
	owner, description := manifestAnnotations(doc.Metadata.Annotations)
	if owner == "unresolved" {
		owner = inferOwnerByConvention(strings.ToLower(kind), name)
	}
	return RuntimeItem{
		Name:        name,
		Kind:        strings.ToLower(kind),
		Owner:       owner,
		Description: description,
		Source:      filepath.ToSlash(path),
		Ports:       ports,
	}, true
}

// manifestAnnotations reads the owner unit ("unresolved" without one) and the
// description (the runtime description, else the description) annotations.
// TRLC-LINKS: REQ-EMG-010
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
func manifestAnnotations(annotations map[string]string) (string, string) {
	owner := "unresolved"
	if x := strings.TrimSpace(annotations["engmodel.dev/owner-unit"]); x != "" {
		owner = x
	}
	description := strings.TrimSpace(annotations["engmodel.dev/runtime-description"])
	if description == "" {
		description = strings.TrimSpace(annotations["engmodel.dev/description"])
	}
	return owner, description
}

// TRLC-LINKS: REQ-EMG-010
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

// TRLC-LINKS: REQ-EMG-010
func portsFromHelmValues(values map[string]any) []string {
	service, ok := values["service"].(map[string]any)
	if !ok {
		return nil
	}
	portRaw, ok := service["port"]
	if !ok {
		return nil
	}
	port := helmPortNumber(portRaw)
	if port <= 0 {
		return nil
	}
	return []string{fmt.Sprintf("%d/TCP", port)}
}

// helmPortNumber is the number of an int, int64, float64 or numeric string; 0 otherwise.
// TRLC-LINKS: REQ-EMG-010
func helmPortNumber(raw any) int {
	port := 0
	switch v := raw.(type) {
	case int:
		port = v
	case int64:
		port = int(v)
	case float64:
		port = int(v)
	case string:
		fmt.Sscanf(strings.TrimSpace(v), "%d", &port)
	}
	return port
}

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
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

// TRLC-LINKS: REQ-EMG-010
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

// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-010
func extractRuntimeDescriptionMarker(line string) (string, bool) {
	markers := []string{
		"engmodel:runtime-description:",
		"engmodel:runtime-description",
		"engmodel.runtime.description:",
		"engmodel.runtime.description",
	}
	for _, marker := range markers {
		if v := markerDescription(line, marker); v != "" {
			return v, true
		}
	}
	return "", false
}
