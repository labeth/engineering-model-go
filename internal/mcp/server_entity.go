// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"fmt"
	"path/filepath"
	"reflect"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
type policyFacts struct {
	controls           []string
	mappings           []model.Mapping
	complianceMappings []model.ComplianceMapping
	threats            []string
	risks              []string
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
func (s *Server) policyForEntity(id string) map[string]any {
	a := s.bundle.Architecture.AuthoredArchitecture
	facts := &policyFacts{
		controls:           []string{},
		mappings:           []model.Mapping{},
		complianceMappings: []model.ComplianceMapping{},
		threats:            []string{},
		risks:              []string{},
	}
	facts.collectMappings(a.Mappings, id)
	facts.collectCompliance(s.bundle.Architecture.Compliance.Mappings, id)
	facts.collectThreats(a.ThreatScenarios, id)
	facts.collectRisks(a.Risks, id)
	sort.Strings(facts.threats)
	sort.Strings(facts.risks)
	return map[string]any{"entityId": id, "controls": uniqueStrings(facts.controls), "complianceMappings": facts.complianceMappings, "threatScenarios": uniqueStrings(facts.threats), "risks": uniqueStrings(facts.risks), "mappings": facts.mappings}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
func (p *policyFacts) collectMappings(in []model.Mapping, id string) {
	for _, m := range in {
		if (m.Type == "guarded_by" || m.Type == "mitigated_by") && m.From == id {
			p.controls = append(p.controls, m.To)
			p.mappings = append(p.mappings, m)
		}
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
func (p *policyFacts) collectCompliance(in []model.ComplianceMapping, id string) {
	for _, cm := range in {
		if containsString(cm.AppliesTo, id) {
			p.controls = append(p.controls, cm.ModelControlRef)
			p.complianceMappings = append(p.complianceMappings, cm)
		}
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
func (p *policyFacts) collectThreats(in []model.ThreatScenario, id string) {
	for _, ts := range in {
		if containsString(ts.AppliesTo, id) || ts.EntryPoint == id || containsString(ts.FlowRefs, id) {
			p.threats = append(p.threats, ts.ID)
			p.controls = append(p.controls, ts.RelatedControls...)
		}
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE
func (p *policyFacts) collectRisks(in []model.Risk, id string) {
	for _, risk := range in {
		if containsString(risk.AppliesTo, id) || containsString(risk.ThreatScenarios, id) {
			p.risks = append(p.risks, risk.ID)
			p.controls = append(p.controls, risk.RelatedControls...)
		}
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) runtimeForEntity(id string) map[string]any {
	a := s.bundle.Architecture.AuthoredArchitecture
	idsToCheck := []string{id}
	if iface := s.findInterface(id); iface != nil && iface.Owner != "" {
		idsToCheck = append(idsToCheck, iface.Owner)
	}
	targetIDs, mappings := deployedMappings(a.Mappings, id, idsToCheck)
	targetIDs = uniqueStrings(targetIDs)
	targets := []model.DeploymentTarget{}
	for _, dep := range a.DeploymentTargets {
		if containsString(targetIDs, dep.ID) {
			targets = append(targets, dep)
		}
	}
	return map[string]any{"entityId": id, "runtimeTargets": targetIDs, "deploymentTargets": targets, "mappings": mappings, "runtimeSources": s.bundle.Architecture.InferenceHints.RuntimeSources}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func deployedMappings(in []model.Mapping, id string, idsToCheck []string) ([]string, []model.Mapping) {
	targetIDs := []string{}
	mappings := []model.Mapping{}
	for _, m := range in {
		if m.Type == "deployed_to" && (id == "" || containsString(idsToCheck, m.From)) {
			targetIDs = append(targetIDs, m.To)
			mappings = append(mappings, m)
		}
	}
	return targetIDs, mappings
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
type confidenceEvidence struct {
	exists          bool
	mappings        int
	requirements    int
	implementations int
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (e confidenceEvidence) basis() []string {
	basis := []string{}
	if e.exists {
		basis = append(basis, "authored model entity")
	}
	if e.mappings > 0 {
		basis = append(basis, "authored mappings")
	}
	if e.requirements > 0 {
		basis = append(basis, "requirement appliesTo links")
	}
	if e.implementations > 0 {
		basis = append(basis, "source ENGMODEL-LINKS")
	}
	return basis
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (e confidenceEvidence) level() string {
	if !e.exists {
		return "low"
	}
	if e.mappings > 0 || e.implementations > 0 || e.requirements > 0 {
		return "high"
	}
	return "medium"
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) confidenceForEntity(id string) map[string]any {
	kind, exists := s.entityKindOrView(id)
	implementations, diagnostics := s.implementationsForModelLink(id)
	mappings := []model.Mapping{}
	for _, m := range s.bundle.Architecture.AuthoredArchitecture.Mappings {
		if m.From == id || m.To == id {
			mappings = append(mappings, m)
		}
	}
	reqs := s.requirementsForEntity(id)
	evidence := confidenceEvidence{exists: exists, mappings: len(mappings), requirements: len(reqs), implementations: len(implementations)}
	return map[string]any{"id": id, "kind": kind, "exists": exists, "confidence": evidence.level(), "basis": evidence.basis(), "mappingCount": len(mappings), "requirementIds": reqs, "implementationFiles": groupImplementationsByFile(implementations), "diagnostics": diagnostics}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-TRACEABILITY-COVERAGE, FU-CODEMAP-INFERENCE
func (s *Server) entityKindOrView(id string) (string, bool) {
	kind, _, exists := s.modelEntity(id)
	if exists {
		return kind, true
	}
	for _, view := range s.bundle.Architecture.Views {
		if view.ID == id {
			return "view", true
		}
	}
	return kind, false
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) stalenessForPath(path string) map[string]any {
	originalPath := strings.TrimSpace(path)
	out := s.fileStatus(s.stalenessCheckPath(originalPath))
	out["requestedPath"] = originalPath
	out["staleness"] = stalenessLabel(out)
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func (s *Server) stalenessCheckPath(path string) string {
	if path == "" || s.repoRoot == "" {
		return path
	}
	if resolved, ok := s.resolvePathInRepo(path); ok {
		return resolved
	}
	return path
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, DEP-LOCAL-WORKSPACE
func stalenessLabel(status map[string]any) string {
	if status["exists"] != true {
		return "missing"
	}
	ageHours, _ := status["ageHours"].(int)
	switch {
	case ageHours < 24:
		return "fresh"
	case ageHours < 168:
		return "stale-soon"
	default:
		return "stale"
	}
}

// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type implementationGroup struct {
	path      string
	lines     []int
	symbols   []map[string]any
	lineSeen  map[int]bool
	traceSeen map[string]bool
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func newImplementationGroup(path string) *implementationGroup {
	return &implementationGroup{path: path, lineSeen: map[int]bool{}, traceSeen: map[string]bool{}}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (g *implementationGroup) add(row map[string]any) {
	line, ok := row["line"].(int)
	if ok && line > 0 && !g.lineSeen[line] {
		g.lineSeen[line] = true
		g.lines = append(g.lines, line)
	}
	trace, _ := row["traceId"].(string)
	key := fmt.Sprintf("%s:%d", trace, line)
	if !g.traceSeen[key] {
		g.traceSeen[key] = true
		g.symbols = append(g.symbols, map[string]any{
			"line":      line,
			"signature": row["signature"],
			"traceId":   row["traceId"],
		})
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (g *implementationGroup) row() map[string]any {
	sort.Ints(g.lines)
	sort.SliceStable(g.symbols, func(i, j int) bool {
		li, _ := iLine(g.symbols[i])
		lj, _ := iLine(g.symbols[j])
		return li < lj
	})
	return map[string]any{
		"path":     g.path,
		"lines":    g.lines,
		"lineList": joinLineNumbers(g.lines),
		"symbols":  g.symbols,
	}
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func groupImplementationsByFile(rows []map[string]any) []map[string]any {
	byPath := map[string]*implementationGroup{}
	paths := []string{}
	for _, row := range rows {
		path, _ := row["path"].(string)
		if path == "" {
			continue
		}
		g := byPath[path]
		if g == nil {
			g = newImplementationGroup(path)
			byPath[path] = g
			paths = append(paths, path)
		}
		g.add(row)
	}
	sort.Strings(paths)
	out := make([]map[string]any, 0, len(paths))
	for _, path := range paths {
		out = append(out, byPath[path].row())
	}
	return out
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func isTestPath(path string) bool {
	p := strings.ToLower(filepath.ToSlash(path))
	base := filepath.Base(p)
	return anyString(testPathDirs, func(dir string) bool { return strings.Contains(p, dir) }) ||
		anyString(testPathSuffixes, func(suffix string) bool { return strings.HasSuffix(base, suffix) })
}

var testPathDirs = []string{"/test/", "/tests/"}

var testPathSuffixes = []string{"_test.go", ".test.ts", ".spec.ts", ".test.tsx", ".spec.tsx"}

// anyString reports whether match holds for any of items.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func anyString(items []string, match func(string) bool) bool {
	for _, item := range items {
		if match(item) {
			return true
		}
	}
	return false
}

// authoredKind describes one slice of the authored architecture: the entity
// kind name, the slice itself and the struct fields that form its display name.
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type authoredKind struct {
	kind   string
	items  func(model.AuthoredArchitecture) any
	labels []string
}

// authoredKinds lists the authored entity kinds in lookup order: the first
// slice holding an ID wins, and graph nodes are emitted in this order.
var authoredKinds = []authoredKind{
	{"functional_group", func(a model.AuthoredArchitecture) any { return a.FunctionalGroups }, []string{"Name"}},
	{"functional_unit", func(a model.AuthoredArchitecture) any { return a.FunctionalUnits }, []string{"Name"}},
	{"actor", func(a model.AuthoredArchitecture) any { return a.Actors }, []string{"Name"}},
	{"attack_vector", func(a model.AuthoredArchitecture) any { return a.AttackVectors }, []string{"Name"}},
	{"referenced_element", func(a model.AuthoredArchitecture) any { return a.ReferencedElements }, []string{"Name"}},
	{"interface", func(a model.AuthoredArchitecture) any { return a.Interfaces }, []string{"Name"}},
	{"data_object", func(a model.AuthoredArchitecture) any { return a.DataObjects }, []string{"Name"}},
	{"deployment_target", func(a model.AuthoredArchitecture) any { return a.DeploymentTargets }, []string{"Name"}},
	{"hardware_item", func(a model.AuthoredArchitecture) any { return a.HardwareItems }, []string{"Name"}},
	{"hardware_interface", func(a model.AuthoredArchitecture) any { return a.HardwareInterfaces }, []string{"Name"}},
	{"control", func(a model.AuthoredArchitecture) any { return a.Controls }, []string{"Name"}},
	{"trust_boundary", func(a model.AuthoredArchitecture) any { return a.TrustBoundaries }, []string{"Name"}},
	{"machine", func(a model.AuthoredArchitecture) any { return a.Machines }, []string{"Name"}},
	{"state", func(a model.AuthoredArchitecture) any { return a.MachineStates() }, []string{"Name"}},
	{"event", func(a model.AuthoredArchitecture) any { return a.Events }, []string{"Name"}},
	{"flow", func(a model.AuthoredArchitecture) any { return a.Flows }, []string{"Title"}},
	{"threat_scenario", func(a model.AuthoredArchitecture) any { return a.ThreatScenarios }, []string{"Title"}},
	{"threat_assumption", func(a model.AuthoredArchitecture) any { return a.ThreatAssumptions }, []string{"Title"}},
	{"threat_out_of_scope", func(a model.AuthoredArchitecture) any { return a.ThreatOutOfScope }, []string{"Title"}},
	{"threat_mitigation", func(a model.AuthoredArchitecture) any { return a.ThreatMitigations }, []string{"ThreatScenarioRef", "ControlRef"}},
	{"control_verification", func(a model.AuthoredArchitecture) any { return a.ControlVerifications }, []string{"ControlRef", "Method"}},
	{"risk", func(a model.AuthoredArchitecture) any { return a.Risks }, []string{"Title"}},
	{"poam_item", func(a model.AuthoredArchitecture) any { return a.POAMItems }, []string{"RiskRef"}},
}

// structField reads a string struct field of one slice element.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func structField(item reflect.Value, name string) string {
	return item.FieldByName(name).String()
}

// label is the display name of one slice element: a single field as is, or
// several fields joined by a space and trimmed.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (k authoredKind) label(item reflect.Value) string {
	parts := make([]string, 0, len(k.labels))
	for _, name := range k.labels {
		parts = append(parts, structField(item, name))
	}
	if len(parts) == 1 {
		return parts[0]
	}
	return strings.TrimSpace(strings.Join(parts, " "))
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) modelEntity(id string) (string, any, bool) {
	id = strings.TrimSpace(id)
	if id == "" {
		return "", nil, false
	}
	if kind, entity, ok := s.requirementEntity(id); ok {
		return kind, entity, true
	}
	if s.bundle == nil {
		return "", nil, false
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	for _, k := range authoredKinds {
		if entity, ok := entityByID(k.items(a), id); ok {
			return k.kind, entity, true
		}
	}
	return "", nil, false
}

// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func (s *Server) requirementEntity(id string) (string, any, bool) {
	if s.requirements == nil {
		return "", nil, false
	}
	for _, x := range s.requirements.Requirements {
		if x.ID == id {
			return "requirement", x, true
		}
	}
	return "", nil, false
}

// entityByID returns the element of the struct slice items whose ID field
// equals id.
// TRLC-LINKS: REQ-EMG-007
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
func entityByID(items any, id string) (any, bool) {
	list := reflect.ValueOf(items)
	for i := 0; i < list.Len(); i++ {
		if list.Index(i).FieldByName("ID").String() == id {
			return list.Index(i).Interface(), true
		}
	}
	return nil, false
}

// ownerFor helper: the Owner of the element of items whose ID equals id.
// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func ownerByID(items any, id string) (string, bool) {
	entity, ok := entityByID(items, id)
	if !ok {
		return "", false
	}
	return reflect.ValueOf(entity).FieldByName("Owner").String(), true
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) graphNodes(query string, max int) []map[string]any {
	if s.bundle == nil {
		return nil
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	out := []map[string]any{}
	for _, k := range authoredKinds {
		out = append(out, k.nodes(a, query)...)
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i]["id"].(string) < out[j]["id"].(string) })
	if len(out) > max {
		out = out[:max]
	}
	return out
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (k authoredKind) nodes(a model.AuthoredArchitecture, query string) []map[string]any {
	out := []map[string]any{}
	list := reflect.ValueOf(k.items(a))
	for i := 0; i < list.Len(); i++ {
		item := list.Index(i)
		id := structField(item, "ID")
		name := k.label(item)
		if graphNodeMatches(query, id, name) {
			out = append(out, map[string]any{"kind": k.kind, "id": id, "name": name})
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func graphNodeMatches(query, id, name string) bool {
	if query == "" {
		return true
	}
	q := strings.ToLower(query)
	return strings.Contains(strings.ToLower(id), q) || strings.Contains(strings.ToLower(name), q)
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func (s *Server) ownerFor(id string) string {
	if s.bundle == nil || strings.TrimSpace(id) == "" {
		return ""
	}
	a := s.bundle.Architecture.AuthoredArchitecture
	for _, items := range []any{a.Interfaces, a.Risks, a.ThreatScenarios, a.ControlVerifications} {
		if owner, ok := ownerByID(items, id); ok {
			return owner
		}
	}
	return ""
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func (s *Server) graphEdges(query string, max int) []map[string]any {
	if s.bundle == nil {
		return nil
	}
	out := []map[string]any{}
	for _, m := range s.bundle.Architecture.AuthoredArchitecture.Mappings {
		if query == "" || mappingMatches(m, strings.ToLower(query)) {
			out = append(out, map[string]any{"type": m.Type, "from": m.From, "to": m.To, "description": m.Description})
		}
	}
	if len(out) > max {
		out = out[:max]
	}
	return out
}

// TRLC-LINKS: REQ-EMG-008
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY, CTRL-STRICT-MCP-INPUT-SCHEMA, EVT-MCP-TOOL-CALL-RECEIVED
func mappingMatches(m model.Mapping, q string) bool {
	for _, field := range []string{m.From, m.To, m.Type} {
		if strings.Contains(strings.ToLower(field), q) {
			return true
		}
	}
	return false
}
