// ENGMODEL-OWNER-UNIT: FU-VIEW-PROJECTION
package view

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
type index struct {
	nodes    map[string]Node
	machines map[string]model.Machine
	flows    map[string]model.Flow
}

// TRLC-LINKS: REQ-EMG-003, REQ-EMG-035, REQ-EMG-036, REQ-EMG-139
// ENGMODEL-LINKS: FU-VIEW-PROJECTION, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, DO-VIEW-PROJECTION, IF-GO-PKG-VIEW
func Build(b model.Bundle, viewID string) (ProjectedView, []validate.Diagnostic) {
	canonical, err := model.NewCanonicalBundle(b)
	if err != nil {
		return ProjectedView{}, canonicalDiagnostics(canonical)
	}
	b = canonical.Documents()
	v, ok := findView(b.Architecture.Views, viewID)
	if !ok {
		return ProjectedView{}, []validate.Diagnostic{{
			Code:     "view.not_found",
			Severity: validate.SeverityError,
			Message:  fmt.Sprintf("view %q not found", viewID),
			Path:     "views",
		}}
	}
	if isSemanticConcernView(v.Kind) && hasAuthoredSemanticRoot(v, b.Architecture.Semantics) {
		return buildSemanticConcernView(v, canonical.Semantic()), nil
	}
	idx := buildIndex(b)
	filter := resolveViewSemantics(v)
	if strings.TrimSpace(v.Kind) == "interaction-flow" {
		return buildInteractionFlowView(flowViewInput{view: v, idx: idx, mappings: b.Architecture.AuthoredArchitecture.Mappings, filter: filter, maxDepth: effectiveMaxDepth(v)}), nil
	}
	walk := &viewWalk{idx: idx, filter: filter, mappings: viewMappings(b.Architecture.AuthoredArchitecture), included: map[string]bool{}, depth: map[string]int{}}
	return projectGraph(v, walk), nil
}

// canonicalDiagnostics are the diagnostics of a canonical bundle that failed,
// sorted; an error of the canonical model is an error, any other a warning.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func canonicalDiagnostics(canonical model.CanonicalBundle) []validate.Diagnostic {
	diags := make([]validate.Diagnostic, 0, len(canonical.Diagnostics()))
	for _, diagnostic := range canonical.Diagnostics() {
		severity := validate.SeverityWarning
		if diagnostic.Severity == model.SemanticSeverityError {
			severity = validate.SeverityError
		}
		diags = append(diags, validate.Diagnostic{
			Code:     diagnostic.Code,
			Severity: severity,
			Message:  diagnostic.Message,
			Path:     diagnostic.Path,
		})
	}
	return validate.SortDiagnostics(diags)
}

// effectiveMaxDepth is the depth limit of a view, or 99 when it sets none.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func effectiveMaxDepth(v model.View) int {
	if v.MaxDepth <= 0 {
		return 99
	}
	return v.MaxDepth
}

// viewWalk walks the mappings outwards from the roots of a view.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
type viewWalk struct {
	idx      index
	filter   viewFilter
	mappings []model.Mapping
	included map[string]bool
	depth    map[string]int
	queue    []string
}

// projectGraph projects the nodes within the depth of the roots of a view, and
// the mappings between them.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func projectGraph(v model.View, w *viewWalk) ProjectedView {
	w.seedRoots(v.Roots)
	w.run(effectiveMaxDepth(v))
	return sortView(w.project(ProjectedView{ID: v.ID, Kind: v.Kind, Title: v.ID}))
}

// seed includes the node at depth 0, unless it is included.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) seed(id string) {
	if !w.included[id] {
		w.included[id] = true
		w.depth[id] = 0
		w.queue = append(w.queue, id)
	}
}

// seedRoots includes the roots; a machine root stands for all its states.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) seedRoots(roots []string) {
	for _, root := range roots {
		root = strings.TrimSpace(root)
		if root == "" {
			continue
		}
		w.seed(root)
		w.seedMachineStates(root)
	}
}

// seedMachineStates includes the states of the machine with the ID.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) seedMachineStates(machineID string) {
	for _, state := range w.idx.machines[machineID].States {
		if id := strings.TrimSpace(state.ID); id != "" {
			w.seed(id)
		}
	}
}

// run includes the neighbors of the queued nodes, down to the depth.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) run(maxDepth int) {
	for len(w.queue) > 0 {
		current := w.queue[0]
		w.queue = w.queue[1:]
		if w.depth[current] < maxDepth {
			w.expand(current)
		}
	}
}

// expand includes the neighbors of the node through the allowed mappings.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) expand(current string) {
	for _, m := range w.mappings {
		if !w.filter.mappingAllowed(strings.TrimSpace(m.Type)) {
			continue
		}
		for _, n := range mappingNeighbors(m, current) {
			w.reach(strings.TrimSpace(n), current)
		}
	}
}

// mappingNeighbors are the ends of the mapping that are not the node, as the
// node is the source, the target or both.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func mappingNeighbors(m model.Mapping, current string) []string {
	neighbors := []string{}
	if m.From == current {
		neighbors = append(neighbors, m.To)
	}
	if m.To == current {
		neighbors = append(neighbors, m.From)
	}
	return neighbors
}

// reach includes the node one step beyond the current node, when it is not blank,
// its kind is allowed and it is not included yet.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) reach(n, current string) {
	if n == "" || !w.filter.kindAllowed(kindForID(n, w.idx)) || w.included[n] {
		return
	}
	w.included[n] = true
	w.depth[n] = w.depth[current] + 1
	w.queue = append(w.queue, n)
}

// nodeVisible reports whether the node is included and its kind allowed.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) nodeVisible(id string) bool {
	return w.included[id] && w.filter.kindAllowed(kindForID(id, w.idx))
}

// project adds the visible nodes and the allowed mappings between them.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) project(pv ProjectedView) ProjectedView {
	pv.Nodes = w.visibleNodes()
	pv.Edges = w.visibleEdges()
	return pv
}

// visibleNodes are the included nodes of an allowed kind.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) visibleNodes() []Node {
	var nodes []Node
	for id := range w.included {
		if w.filter.kindAllowed(kindForID(id, w.idx)) {
			nodes = append(nodes, toNode(id, w.idx))
		}
	}
	return nodes
}

// visibleEdges are the allowed mappings between visible nodes.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (w *viewWalk) visibleEdges() []Edge {
	var edges []Edge
	for _, m := range w.mappings {
		if w.filter.mappingAllowed(strings.TrimSpace(m.Type)) && w.nodeVisible(m.From) && w.nodeVisible(m.To) {
			edges = append(edges, Edge{From: m.From, To: m.To, Type: m.Type, Label: mappingLabel(m)})
		}
	}
	return edges
}

// mappingLabel is the type of a mapping, with its description.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func mappingLabel(m model.Mapping) string {
	label := strings.TrimSpace(m.Type)
	if d := strings.TrimSpace(m.Description); d != "" {
		label += ": " + d
	}
	return label
}

// Edge types the machine transitions project to: MappingTransition runs
// state -> state, MappingOnEvent state -> event for the event a transition
// from the state waits for. They are view-only mapping types; authored
// relationships cannot use them.
const (
	MappingTransition = "transition"
	MappingOnEvent    = "on_event"
)

// viewMappings returns the authored relationships followed by the mappings
// that the machine transitions project to, so the state lifecycle view is
// driven by machines (ADR-EMG-038).
//
// TRLC-LINKS: REQ-EMG-139
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func viewMappings(a model.AuthoredArchitecture) []model.Mapping {
	out := append([]model.Mapping(nil), a.Mappings...)
	seen := map[string]bool{}
	for _, m := range a.Machines {
		out = appendTransitionMappings(out, m, seen)
	}
	return out
}

// appendTransitionMappings adds a transition mapping for each transition of
// the machine, and an on_event mapping for each event a state waits for, once.
//
// TRLC-LINKS: REQ-EMG-139
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func appendTransitionMappings(out []model.Mapping, m model.Machine, seen map[string]bool) []model.Mapping {
	for _, t := range m.Transitions {
		from, to, on := strings.TrimSpace(t.From), strings.TrimSpace(t.To), strings.TrimSpace(t.On)
		out = append(out, model.Mapping{Type: MappingTransition, From: from, To: to, Description: transitionDescription(t)})
		if key := from + "|" + on; on != "" && !seen[key] {
			seen[key] = true
			out = append(out, model.Mapping{Type: MappingOnEvent, From: from, To: on})
		}
	}
	return out
}

// transitionDescription renders a transition as "TR-ID on EVT [guard] / effect".
//
// TRLC-LINKS: REQ-EMG-139
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func transitionDescription(t model.MachineTransition) string {
	var parts []string
	parts = append(parts, strings.TrimSpace(t.ID))
	if on := strings.TrimSpace(t.On); on != "" {
		parts = append(parts, "on "+on)
	}
	if guard := strings.TrimSpace(t.Guard); guard != "" {
		parts = append(parts, "["+guard+"]")
	}
	if effect := strings.TrimSpace(t.Effect); effect != "" {
		parts = append(parts, "/ "+effect)
	}
	return strings.Join(parts, " ")
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func buildIndex(b model.Bundle) index {
	a := b.Architecture.AuthoredArchitecture
	idx := index{nodes: map[string]Node{}, machines: map[string]model.Machine{}, flows: map[string]model.Flow{}}
	for _, category := range nodeCategories(a) {
		idx.addNodes(category)
	}
	for _, x := range a.Machines {
		idx.machines[x.ID] = x
	}
	for _, x := range a.Flows {
		idx.flows[x.ID] = x
	}
	return idx
}

// addNodes adds the nodes of a category, the last of a repeated ID winning,
// unless an earlier category has the ID.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (idx index) addNodes(category []Node) {
	byID := map[string]Node{}
	for _, n := range category {
		byID[n.ID] = n
	}
	for id, n := range byID {
		if _, taken := idx.nodes[id]; !taken {
			idx.nodes[id] = n
		}
	}
}

// nodesOf are the nodes of the items, of the kind, labeled by the name of each
// item or its ID when it has none.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func nodesOf[T any](kind string, items []T, describe func(T) (id, name string)) []Node {
	nodes := make([]Node, 0, len(items))
	for _, item := range items {
		id, name := describe(item)
		nodes = append(nodes, Node{ID: id, Label: nonEmpty(name, id), Kind: kind})
	}
	return nodes
}

// nodeCategories are the nodes of every kind of element, in the order in which
// an ID that several kinds have takes its first kind.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func nodeCategories(a model.AuthoredArchitecture) [][]Node {
	return [][]Node{
		nodesOf("hardware_item", a.HardwareItems, func(x model.HardwareItem) (string, string) { return x.ID, x.Name }),
		nodesOf("hardware_interface", a.HardwareInterfaces, func(x model.HardwareInterface) (string, string) { return x.ID, x.Name }),
		nodesOf("functional_group", a.FunctionalGroups, func(x model.FunctionalGroup) (string, string) { return x.ID, x.Name }),
		nodesOf("functional_unit", a.FunctionalUnits, func(x model.FunctionalUnit) (string, string) { return x.ID, x.Name }),
		nodesOf("actor", a.Actors, func(x model.Actor) (string, string) { return x.ID, x.Name }),
		nodesOf("attack_vector", a.AttackVectors, func(x model.AttackVector) (string, string) { return x.ID, x.Name }),
		nodesOf("referenced_element", a.ReferencedElements, func(x model.ReferencedElement) (string, string) { return x.ID, x.Name }),
		nodesOf("interface", a.Interfaces, func(x model.Interface) (string, string) { return x.ID, x.Name }),
		nodesOf("data_object", a.DataObjects, func(x model.DataObject) (string, string) { return x.ID, x.Name }),
		nodesOf("deployment_target", a.DeploymentTargets, func(x model.DeploymentTarget) (string, string) { return x.ID, x.Name }),
		nodesOf("control", a.Controls, func(x model.Control) (string, string) { return x.ID, x.Name }),
		nodesOf("trust_boundary", a.TrustBoundaries, func(x model.TrustBoundary) (string, string) { return x.ID, x.Name }),
		nodesOf("machine", a.Machines, func(x model.Machine) (string, string) { return x.ID, x.Name }),
		nodesOf("state", a.MachineStates(), func(x model.MachineState) (string, string) { return x.ID, x.Name }),
		nodesOf("event", a.Events, func(x model.Event) (string, string) { return x.ID, x.Name }),
		nodesOf("flow", a.Flows, func(x model.Flow) (string, string) { return x.ID, x.Title }),
		nodesOf("threat_scenario", a.ThreatScenarios, func(x model.ThreatScenario) (string, string) { return x.ID, x.Title }),
		nodesOf("threat_assumption", a.ThreatAssumptions, func(x model.ThreatAssumption) (string, string) { return x.ID, x.Title }),
		nodesOf("threat_out_of_scope", a.ThreatOutOfScope, func(x model.ThreatOutOfScope) (string, string) { return x.ID, x.Title }),
		nodesOf("threat_mitigation", a.ThreatMitigations, func(x model.ThreatMitigation) (string, string) { return x.ID, "" }),
		nodesOf("control_verification", a.ControlVerifications, func(x model.ControlVerification) (string, string) { return x.ID, "" }),
	}
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func findView(views []model.View, id string) (model.View, bool) {
	for _, v := range views {
		if v.ID == id {
			return v, true
		}
	}
	return model.View{}, false
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func toNode(id string, idx index) Node {
	if n, ok := idx.nodes[id]; ok {
		return n
	}
	return Node{ID: id, Label: id, Kind: "unknown"}
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func kindForID(id string, idx index) string {
	return toNode(id, idx).Kind
}

// viewFilter says which node kinds and mappings a view includes and excludes.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
type viewFilter struct {
	includeKinds    map[string]bool
	excludeKinds    map[string]bool
	includeMappings map[string]bool
	excludeMappings map[string]bool
}

// allowed reports whether the name passes the filter: it is not blank, it is
// included when there is a list of inclusions, and it is not excluded.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func allowed(name string, includes, excludes map[string]bool) bool {
	name = strings.TrimSpace(name)
	if name == "" || excludes[name] {
		return false
	}
	return len(includes) == 0 || includes[name]
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (f viewFilter) mappingAllowed(mapping string) bool {
	return allowed(mapping, f.includeMappings, f.excludeMappings)
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (f viewFilter) kindAllowed(kind string) bool {
	return allowed(kind, f.includeKinds, f.excludeKinds)
}

// viewKindDefault is what a view of a kind includes and excludes when it
// authors no kind or mapping inclusion.
type viewKindDefault struct {
	excludeKinds    []string
	includeMappings []string
	includeKinds    []string
}

// viewKindDefaults are the defaults of the view kinds.
var viewKindDefaults = map[string]viewKindDefault{
	"architecture-intent": {excludeKinds: []string{"attack_vector"}},
	"communication":       {includeMappings: []string{"interacts_with", "calls", "publishes", "subscribes", "reads", "writes", "streams", "depends_on"}},
	"deployment": {
		includeMappings: []string{"deployed_to", "allocated_to", "depends_on", "contains"},
		includeKinds:    []string{"functional_group", "functional_unit", "deployment_target", "interface", "referenced_element", "trust_boundary"},
	},
	"security": {
		includeMappings: []string{"targets", "mitigated_by", "bounded_by", "guarded_by", "depends_on"},
		includeKinds:    []string{"actor", "hardware_item", "data_object", "functional_group", "functional_unit", "attack_vector", "control", "trust_boundary", "referenced_element", "interface", "deployment_target", "flow", "threat_scenario", "threat_assumption", "threat_out_of_scope", "threat_mitigation", "control_verification"},
	},
	"traceability": {
		includeMappings: []string{"implements", "satisfies", "verified_by", "allocated_to", "deployed_to", "depends_on"},
		includeKinds:    []string{"functional_group", "functional_unit", "interface", "data_object", "deployment_target", "control", "referenced_element"},
	},
	"state-lifecycle": {
		includeMappings: []string{MappingTransition, MappingOnEvent, "guarded_by"},
		includeKinds:    []string{"state", "event", "control", "trust_boundary", "referenced_element"},
	},
	"interaction-flow": {
		includeMappings: []string{"flow_next", "flow_error", "flow_async", "flow_ref"},
		includeKinds:    []string{"flow", "flow_step", "actor", "functional_group", "functional_unit", "interface", "data_object", "deployment_target", "control", "trust_boundary", "state", "event", "referenced_element"},
	},
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func resolveViewSemantics(v model.View) viewFilter {
	f := viewFilter{
		includeKinds:    setFromSlice(v.IncludeKinds),
		excludeKinds:    setFromSlice(v.ExcludeKinds),
		includeMappings: setFromSlice(v.IncludeMappings),
		excludeMappings: setFromSlice(v.ExcludeMappings),
	}
	if len(f.includeKinds) != 0 || len(f.includeMappings) != 0 {
		return f
	}
	d := viewKindDefaults[strings.TrimSpace(v.Kind)]
	for _, kind := range d.excludeKinds {
		f.excludeKinds[kind] = true
	}
	if d.includeMappings != nil {
		f.includeMappings = setFromSlice(d.includeMappings)
	}
	if d.includeKinds != nil {
		f.includeKinds = setFromSlice(d.includeKinds)
	}
	return f
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func setFromSlice(in []string) map[string]bool {
	out := map[string]bool{}
	for _, x := range in {
		if x = strings.TrimSpace(x); x != "" {
			out[x] = true
		}
	}
	return out
}

// flowViewInput is what the interaction flow view is projected from.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
type flowViewInput struct {
	view     model.View
	idx      index
	mappings []model.Mapping
	filter   viewFilter
	maxDepth int
}

// flowViewBuilder collects the nodes and edges of the interaction flow view.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
type flowViewBuilder struct {
	in        flowViewInput
	nodesByID map[string]Node
	edges     []Edge
	edgeSeen  map[string]bool
	adj       map[string][]string
}

// flowStep is a step of a flow, by the flow ID and the node ID of the step.
type flowStep struct {
	flowID string
	nodeID string
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func buildInteractionFlowView(in flowViewInput) ProjectedView {
	b := &flowViewBuilder{in: in, nodesByID: map[string]Node{}, edgeSeen: map[string]bool{}, adj: map[string][]string{}}
	flowIDs := b.selectedFlowIDs()
	for _, flowID := range flowIDs {
		b.addFlow(flowID)
	}
	return sortView(b.project(b.reachable(b.roots(flowIDs))))
}

// stepNodeID is the node ID of a step of a flow.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func stepNodeID(flowID, stepID string) string {
	return flowID + "::" + stepID
}

// addNode adds the node unless its ID is blank or its kind is not allowed.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addNode(n Node) {
	if strings.TrimSpace(n.ID) != "" && b.in.filter.kindAllowed(n.Kind) {
		b.nodesByID[n.ID] = n
	}
}

// addEdge adds the edge, once, when its mapping is allowed and both its nodes exist.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addEdge(e Edge) {
	edgeKey := e.From + "|" + e.To + "|" + e.Type
	if !b.edgeWanted(e) || b.edgeSeen[edgeKey] {
		return
	}
	b.edgeSeen[edgeKey] = true
	b.edges = append(b.edges, e)
	b.adj[e.From] = append(b.adj[e.From], e.To)
}

// edgeWanted reports whether the edge has an allowed mapping and both its nodes exist.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) edgeWanted(e Edge) bool {
	return b.in.filter.mappingAllowed(e.Type) && b.hasNode(e.From) && b.hasNode(e.To)
}

// hasNode reports whether the ID is a node.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) hasNode(id string) bool {
	_, ok := b.nodesByID[id]
	return ok
}

// selectedFlowIDs are the flows named by the roots, or all flows when no root
// names one, sorted.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) selectedFlowIDs() []string {
	ids := b.flowRoots()
	if len(ids) == 0 {
		ids = sortedFlowIDs(b.in.idx.flows)
	}
	sort.Strings(ids)
	return ids
}

// flowRoots are the roots of the view that are flows.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) flowRoots() []string {
	ids := []string{}
	for _, r := range b.in.view.Roots {
		if id := strings.TrimSpace(r); id != "" && b.isFlow(id) {
			ids = append(ids, id)
		}
	}
	return ids
}

// sortedFlowIDs are the IDs of the flows.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func sortedFlowIDs(flows map[string]model.Flow) []string {
	ids := make([]string, 0, len(flows))
	for id := range flows {
		ids = append(ids, id)
	}
	return ids
}

// isFlow reports whether the ID is a flow.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) isFlow(id string) bool {
	_, ok := b.in.idx.flows[id]
	return ok
}

// addFlow adds the flow, its steps, the elements the steps refer to and the
// activities between them.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addFlow(flowID string) {
	flow := b.in.idx.flows[flowID]
	b.addNode(Node{ID: flowID, Label: nonEmpty(strings.TrimSpace(flow.Title), flowID), Kind: "flow"})
	b.addStepNodes(flowID, flow)
	b.addEntryEdges(flowID, flow)
	b.addActivityEdges(b.addStepEdges(flowID, flow))
}

// addStepNodes adds a node for each step that has an ID.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addStepNodes(flowID string, flow model.Flow) {
	for _, step := range flow.Steps {
		if stepID := strings.TrimSpace(step.ID); stepID != "" {
			b.addNode(Node{ID: stepNodeID(flowID, stepID), Label: nonEmpty(strings.TrimSpace(step.Action), stepID), Kind: "flow_step"})
		}
	}
}

// addEntryEdges adds an edge from the flow to each of its entry steps.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addEntryEdges(flowID string, flow model.Flow) {
	for _, entry := range flow.Entry {
		if sid := strings.TrimSpace(entry); sid != "" {
			b.addEdge(Edge{From: flowID, To: stepNodeID(flowID, sid), Type: "flow_next", Label: "flow_next"})
		}
	}
}

// addStepEdges adds the edges from each step to the element it refers to and
// to its next and error steps. It returns the elements the steps refer to.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addStepEdges(flowID string, flow model.Flow) map[string]bool {
	participants := map[string]bool{}
	for _, step := range flow.Steps {
		at := flowStep{flowID: flowID, nodeID: stepNodeID(flowID, strings.TrimSpace(step.ID))}
		b.addRefEdge(at, step.Ref, participants)
		edgeType := "flow_next"
		if step.Async {
			edgeType = "flow_async"
		}
		b.addTargetEdges(at, step.Next, edgeType)
		b.addTargetEdges(at, step.OnError, "flow_error")
	}
	return participants
}

// addRefEdge adds the element a step refers to, when it is known, and an edge to it.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addRefEdge(at flowStep, ref string, participants map[string]bool) {
	if strings.TrimSpace(ref) == "" {
		return
	}
	refNode := toNode(strings.TrimSpace(ref), b.in.idx)
	if strings.TrimSpace(refNode.ID) == "" || refNode.Kind == "unknown" {
		return
	}
	b.addNode(refNode)
	b.addEdge(Edge{From: at.nodeID, To: refNode.ID, Type: "flow_ref", Label: "flow_ref"})
	participants[refNode.ID] = true
}

// addTargetEdges adds an edge of the type from the step to each of the steps of its flow.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addTargetEdges(at flowStep, targets []string, edgeType string) {
	for _, target := range targets {
		b.addEdge(Edge{From: at.nodeID, To: stepNodeID(at.flowID, strings.TrimSpace(target)), Type: edgeType, Label: edgeType})
	}
}

// interactionActivityTypes are the mapping types that show what the elements a
// flow refers to do.
var interactionActivityTypes = map[string]bool{
	"contains": true, "calls": true, "reads": true, "writes": true, "publishes": true, "subscribes": true, "streams": true,
}

// addActivityEdges adds the mappings of the activity types from the elements
// the flow refers to, with their ends.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) addActivityEdges(participants map[string]bool) {
	for _, m := range b.in.mappings {
		if fromNode, toEnd, ok := b.activityEnds(m, participants); ok {
			t := strings.TrimSpace(m.Type)
			b.addNode(fromNode)
			b.addNode(toEnd)
			b.addEdge(Edge{From: fromNode.ID, To: toEnd.ID, Type: t, Label: t})
		}
	}
}

// activityEnds are the known nodes at the ends of a mapping of an activity
// type that starts at an element the flow refers to.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) activityEnds(m model.Mapping, participants map[string]bool) (from, to Node, ok bool) {
	fromID := strings.TrimSpace(m.From)
	toID := strings.TrimSpace(m.To)
	if !interactionActivityTypes[strings.TrimSpace(m.Type)] || !participants[fromID] || toID == "" {
		return Node{}, Node{}, false
	}
	from, to = toNode(fromID, b.in.idx), toNode(toID, b.in.idx)
	return from, to, from.Kind != "unknown" && to.Kind != "unknown"
}

// roots are the roots of the view that are nodes, or the selected flows that are.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) roots(flowIDs []string) []string {
	roots := b.existing(b.in.view.Roots)
	if len(roots) == 0 {
		roots = b.existing(flowIDs)
	}
	return roots
}

// existing is the trimmed non-blank IDs that are nodes.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) existing(ids []string) []string {
	var out []string
	for _, id := range ids {
		if id = strings.TrimSpace(id); b.hasNode(id) {
			out = append(out, id)
		}
	}
	return out
}

// reachable are the roots and the nodes the edges lead to within the depth.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) reachable(roots []string) map[string]bool {
	included := map[string]bool{}
	depth := map[string]int{}
	queue := []string{}
	for _, r := range roots {
		queue = visitNode(included, queue, r)
	}
	for len(queue) > 0 {
		cur := queue[0]
		queue = queue[1:]
		for _, nxt := range b.in.nextWithin(b.adj[cur], depth[cur]) {
			if !included[nxt] {
				depth[nxt] = depth[cur] + 1
				queue = visitNode(included, queue, nxt)
			}
		}
	}
	return included
}

// visitNode includes the node and queues it, unless it is included.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func visitNode(included map[string]bool, queue []string, id string) []string {
	if included[id] {
		return queue
	}
	included[id] = true
	return append(queue, id)
}

// nextWithin are the nodes one more step away, when the depth has not reached the limit.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (in flowViewInput) nextWithin(next []string, depth int) []string {
	if depth >= in.maxDepth {
		return nil
	}
	return next
}

// project is the included nodes and the edges between them.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) project(included map[string]bool) ProjectedView {
	pv := ProjectedView{ID: b.in.view.ID, Kind: b.in.view.Kind, Title: b.in.view.ID}
	pv.Nodes = b.includedNodes(included)
	pv.Edges = b.includedEdges(included)
	return pv
}

// includedNodes are the nodes in the set.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) includedNodes(included map[string]bool) []Node {
	var nodes []Node
	for id, n := range b.nodesByID {
		if included[id] {
			nodes = append(nodes, n)
		}
	}
	return nodes
}

// includedEdges are the edges between nodes in the set.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func (b *flowViewBuilder) includedEdges(included map[string]bool) []Edge {
	var edges []Edge
	for _, e := range b.edges {
		if included[e.From] && included[e.To] {
			edges = append(edges, e)
		}
	}
	return edges
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-VIEW-PROJECTION
func sortView(v ProjectedView) ProjectedView {
	sort.SliceStable(v.Nodes, func(i, j int) bool {
		if v.Nodes[i].Kind != v.Nodes[j].Kind {
			return v.Nodes[i].Kind < v.Nodes[j].Kind
		}
		return v.Nodes[i].ID < v.Nodes[j].ID
	})
	sort.SliceStable(v.Edges, func(i, j int) bool {
		a := v.Edges[i]
		b := v.Edges[j]
		if a.From != b.From {
			return a.From < b.From
		}
		if a.To != b.To {
			return a.To < b.To
		}
		if a.Type != b.Type {
			return a.Type < b.Type
		}
		return a.Label < b.Label
	})
	return v
}
