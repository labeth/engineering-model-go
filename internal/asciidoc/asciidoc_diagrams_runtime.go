// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"cmp"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// runtimeEndpointKinds are the runtime kinds that stand for a unit in the API view.
var runtimeEndpointKinds = map[string]bool{
	"helmrelease": true, "deployment": true, "workload": true, "service": true, "lambda_function": true,
	"terraform_resource": true, "eventbridge_rule": true, "eventbridge_target": true, "queue": true, "topic": true,
}

// runtimeEndpoints indexes the runtime elements that stand for units and the
// ports of the services.
type runtimeEndpoints struct {
	byUnit       map[string]string
	servicePorts map[string]string
}

// newRuntimeEndpoints maps each unit to the first runtime element it owns
// and each service to its ports.
// TRLC-LINKS: REQ-EMG-003
func newRuntimeEndpoints(runtime []codescan.RuntimeItem) runtimeEndpoints {
	ix := runtimeEndpoints{byUnit: map[string]string{}, servicePorts: map[string]string{}}
	for _, r := range runtime {
		name := runtimeShortName(r.Name)
		ix.noteOwner(r, name)
		if (r.Kind == "service" || r.Kind == "helmrelease") && len(r.Ports) > 0 {
			ix.servicePorts[name] = strings.Join(r.Ports, ", ")
		}
	}
	return ix
}

// noteOwner maps the owner of an endpoint element to its name, unless it has one.
// TRLC-LINKS: REQ-EMG-003
func (ix runtimeEndpoints) noteOwner(r codescan.RuntimeItem, name string) {
	owner := strings.TrimSpace(r.Owner)
	if owner == "" || owner == "unresolved" || !runtimeEndpointKinds[strings.TrimSpace(strings.ToLower(r.Kind))] {
		return
	}
	if _, ok := ix.byUnit[r.Owner]; !ok {
		ix.byUnit[r.Owner] = name
	}
}

// row is the API row of a depends_on mapping between units that have a runtime element.
// TRLC-LINKS: REQ-EMG-003
func (ix runtimeEndpoints) row(m model.Mapping) (asciidocRuntimeAPIRow, bool) {
	from := ix.byUnit[m.From]
	to := ix.byUnit[m.To]
	if m.Type != "depends_on" || strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
		return asciidocRuntimeAPIRow{}, false
	}
	ports := textutil.NonEmpty(ix.servicePorts[to], "unknown")
	return asciidocRuntimeAPIRow{Consumer: from, Provider: to, Ports: ports}, true
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildRuntimeAPIRows(runtime []codescan.RuntimeItem, mappings []model.Mapping) []asciidocRuntimeAPIRow {
	ix := newRuntimeEndpoints(runtime)
	out := []asciidocRuntimeAPIRow{}
	seen := map[string]bool{}
	for _, m := range mappings {
		row, ok := ix.row(m)
		key := row.Consumer + "|" + row.Provider + "|" + row.Ports
		if ok && !seen[key] {
			seen[key] = true
			out = append(out, row)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		return cmp.Or(cmp.Compare(out[i].Provider, out[j].Provider), cmp.Compare(out[i].Consumer, out[j].Consumer)) < 0
	})
	return out
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildRuntimeAPIMermaid(rows []asciidocRuntimeAPIRow) string {
	if len(rows) == 0 {
		return ""
	}
	lines := mermaidLines{"flowchart LR"}
	for _, r := range rows {
		cn := "RT_" + codescan.SanitizeNode(r.Consumer)
		pn := "RT_" + codescan.SanitizeNode(r.Provider)
		lines.addf("  %s[\"%s\"]:::runtime_element", cn, escapeMermaidLabel(r.Consumer))
		lines.addf("  %s[\"%s\"]:::runtime_element", pn, escapeMermaidLabel(r.Provider))
		lines.addf("  %s -->|API %s| %s", cn, escapeMermaidLabel(r.Ports), pn)
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

// TRLC-LINKS: REQ-EMG-003
func runtimeShortName(s string) string {
	x := strings.TrimSpace(s)
	if x == "" {
		return x
	}
	if strings.Contains(x, "/") {
		parts := strings.Split(x, "/")
		return strings.TrimSpace(parts[len(parts)-1])
	}
	return x
}

// deploymentKindRoles maps the runtime kinds of a deployment to the role of
// their element in the deployment rows.
var deploymentKindRoles = map[string]string{
	"gitrepository": "source", "kustomization": "kustomization",
	"helmrelease": "release", "eventbridge_rule": "release", "terraform_resource": "release",
	"deployment": "workload", "workload": "workload", "lambda_function": "workload", "service": "workload",
	"queue": "workload", "topic": "workload", "eventbridge_target": "workload",
	"namespace": "namespace", "cluster": "cluster",
}

// deploymentInventory is the deployment elements of the runtime by role.
type deploymentInventory struct {
	sourceName string
	kustomName string
	releases   []string
	workloads  []string
	namespaces []string
	clusters   []string
}

// newDeploymentInventory sorts the runtime elements into their roles; the
// first git repository and kustomization stand for all.
// TRLC-LINKS: REQ-EMG-003
func newDeploymentInventory(runtime []codescan.RuntimeItem) deploymentInventory {
	byRole := map[string][]string{}
	for _, r := range runtime {
		if role, ok := deploymentKindRoles[r.Kind]; ok {
			byRole[role] = append(byRole[role], deploymentNodeName(r))
		}
	}
	return deploymentInventory{
		sourceName: firstNonEmpty(byRole["source"]),
		kustomName: firstNonEmpty(byRole["kustomization"]),
		releases:   textutil.UniqueSorted(byRole["release"]),
		workloads:  textutil.UniqueSorted(byRole["workload"]),
		namespaces: textutil.UniqueSorted(byRole["namespace"]),
		clusters:   textutil.UniqueSorted(byRole["cluster"]),
	}
}

// firstNonEmpty is the first item that is not empty, or empty.
// TRLC-LINKS: REQ-EMG-003
func firstNonEmpty(items []string) string {
	for _, item := range items {
		if item != "" {
			return item
		}
	}
	return ""
}

// deploymentRowSet collects deployment rows, each once.
type deploymentRowSet struct {
	rows []asciidocDeploymentRow
	seen map[string]bool
}

// add records a row when its three parts are not blank and it is new.
// TRLC-LINKS: REQ-EMG-003
func (s *deploymentRowSet) add(from, to, how string) {
	from = strings.TrimSpace(from)
	to = strings.TrimSpace(to)
	how = strings.TrimSpace(how)
	key := from + "|" + to + "|" + how
	if from == "" || to == "" || how == "" || s.seen[key] {
		return
	}
	s.seen[key] = true
	s.rows = append(s.rows, asciidocDeploymentRow{From: from, To: to, How: how})
}

// addFrom adds a row from one element to each of the to elements.
// TRLC-LINKS: REQ-EMG-003
func (s *deploymentRowSet) addFrom(from string, to []string, how string) {
	for _, t := range to {
		s.add(from, t, how)
	}
}

// addTo adds a row from each of the from elements to one element.
// TRLC-LINKS: REQ-EMG-003
func (s *deploymentRowSet) addTo(from []string, to, how string) {
	for _, f := range from {
		s.add(f, to, how)
	}
}

// addMatching adds a row for each pair of from and to elements that match.
// TRLC-LINKS: REQ-EMG-003
func (s *deploymentRowSet) addMatching(from, to []string, how string, matches func(from, to string) bool) {
	for _, f := range from {
		s.addMatchesOf(f, to, how, matches)
	}
}

// addMatchesOf adds a row for each of the to elements that match the from element.
// TRLC-LINKS: REQ-EMG-003
func (s *deploymentRowSet) addMatchesOf(from string, to []string, how string, matches func(from, to string) bool) {
	for _, t := range to {
		if matches(from, t) {
			s.add(from, t, how)
		}
	}
}

// containsFold reports whether s contains substr, ignoring case.
// TRLC-LINKS: REQ-EMG-003
func containsFold(s, substr string) bool {
	return strings.Contains(strings.ToLower(s), strings.ToLower(substr))
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
// TRLC-LINKS: REQ-EMG-003
func buildDeploymentRows(runtime []codescan.RuntimeItem) []asciidocDeploymentRow {
	inv := newDeploymentInventory(runtime)
	set := &deploymentRowSet{rows: []asciidocDeploymentRow{}, seen: map[string]bool{}}
	if inv.sourceName != "" && inv.kustomName != "" {
		set.add(inv.sourceName, inv.kustomName, "reconciles")
	}
	if inv.kustomName != "" {
		set.addFrom(inv.kustomName, inv.releases, "applies")
	}
	set.addMatching(inv.releases, inv.workloads, "deploys", func(release, workload string) bool { return containsFold(workload, release) })
	set.addMatching(inv.releases, inv.namespaces, "targets", func(release, ns string) bool { return containsFold(release, ns) })
	if len(inv.clusters) > 0 {
		set.addTo(inv.namespaces, inv.clusters[0], "part_of")
	}
	// Fallback for runtime sets that do not naturally match the release/namespace
	// heuristics above (for example Terraform Lambda/EventBridge-only samples).
	if len(set.rows) == 0 {
		set.addFallback(runtime)
	}
	sort.SliceStable(set.rows, func(i, j int) bool {
		a, b := set.rows[i], set.rows[j]
		return cmp.Or(cmp.Compare(a.From, b.From), cmp.Compare(a.To, b.To), cmp.Compare(a.How, b.How)) < 0
	})
	return set.rows
}

// addFallback adds a deploys row from the source of each runtime element.
// TRLC-LINKS: REQ-EMG-003
func (s *deploymentRowSet) addFallback(runtime []codescan.RuntimeItem) {
	for _, r := range runtime {
		target := deploymentNodeName(r)
		if target == "" {
			continue
		}
		source := runtimeShortName(strings.ReplaceAll(strings.TrimSpace(r.Source), "\\", "/"))
		s.add(textutil.NonEmpty(source, "inferred-runtime-source"), target, "deploys")
	}
}

// deploymentEdge is a deployment row drawn as an edge.
type deploymentEdge struct {
	From string
	To   string
	How  string
}

// deploymentClassRank orders the node classes; a node keeps the higher class.
var deploymentClassRank = map[string]int{
	"deployment_element": 1,
	"runtime_element":    2,
}

// deploymentDiagram draws the deployment rows as namespaces in clusters, a
// control plane and standalone nodes.
type deploymentDiagram struct {
	lines          mermaidLines
	edges          []deploymentEdge
	nodeClass      map[string]string
	nodeRole       map[string]string
	sourceSet      map[string]bool
	kustomSet      map[string]bool
	releaseSet     map[string]bool
	workloadSet    map[string]bool
	nsSet          map[string]bool
	clusterSet     map[string]bool
	releaseTargets map[string]map[string]bool
	releaseDeploys map[string]map[string]bool
	nsCluster      map[string]string
	releaseNS      map[string]string
	nsReleases     map[string][]string
	nsWorkloads    map[string][]string
	emitted        map[string]bool
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
// TRLC-LINKS: REQ-EMG-003
func buildDeploymentMermaid(rows []asciidocDeploymentRow) string {
	d := newDeploymentDiagram()
	for _, r := range rows {
		d.classify(r)
	}
	d.indexPlacement()
	d.assignRoles()
	d.emitNamespaces()
	d.emitControlPlane()
	d.emitStandalone()
	d.emitEdges()
	d.lines = appendMermaidClassDefs(d.lines)
	return strings.Join(d.lines, "\n")
}

// newDeploymentDiagram is an empty diagram.
// TRLC-LINKS: REQ-EMG-003
func newDeploymentDiagram() *deploymentDiagram {
	return &deploymentDiagram{
		lines:          mermaidLines{"flowchart TB"},
		nodeClass:      map[string]string{},
		nodeRole:       map[string]string{},
		sourceSet:      map[string]bool{},
		kustomSet:      map[string]bool{},
		releaseSet:     map[string]bool{},
		workloadSet:    map[string]bool{},
		nsSet:          map[string]bool{},
		clusterSet:     map[string]bool{},
		releaseTargets: map[string]map[string]bool{},
		releaseDeploys: map[string]map[string]bool{},
		nsCluster:      map[string]string{},
		emitted:        map[string]bool{},
	}
}

// setClass gives the node the class, unless it has one of at least that rank.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) setClass(node, class string) {
	if existing, ok := d.nodeClass[node]; ok && deploymentClassRank[class] <= deploymentClassRank[existing] {
		return
	}
	d.nodeClass[node] = class
}

// deploymentRowKinds notes what a row of each kind tells about its elements.
var deploymentRowKinds = map[string]func(*deploymentDiagram, asciidocDeploymentRow){
	"reconciles": (*deploymentDiagram).noteReconciles,
	"applies":    (*deploymentDiagram).noteApplies,
	"deploys":    (*deploymentDiagram).noteDeploys,
	"targets":    (*deploymentDiagram).noteTargets,
	"part_of":    (*deploymentDiagram).notePartOf,
}

// classify records a row as an edge and what it tells about its elements.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) classify(r asciidocDeploymentRow) {
	if strings.TrimSpace(r.From) == "" || strings.TrimSpace(r.To) == "" || strings.TrimSpace(r.How) == "" {
		return
	}
	d.edges = append(d.edges, deploymentEdge{From: r.From, To: r.To, How: r.How})
	d.setClass(r.From, "deployment_element")
	d.setClass(r.To, "deployment_element")
	if note := deploymentRowKinds[r.How]; note != nil {
		note(d, r)
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) noteReconciles(r asciidocDeploymentRow) {
	d.sourceSet[r.From] = true
	d.kustomSet[r.To] = true
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) noteApplies(r asciidocDeploymentRow) {
	d.kustomSet[r.From] = true
	d.releaseSet[r.To] = true
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) noteDeploys(r asciidocDeploymentRow) {
	d.releaseSet[r.From] = true
	d.workloadSet[r.To] = true
	d.setClass(r.To, "runtime_element")
	addToSetMap(d.releaseDeploys, r.From, r.To)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) noteTargets(r asciidocDeploymentRow) {
	d.releaseSet[r.From] = true
	d.nsSet[r.To] = true
	addToSetMap(d.releaseTargets, r.From, r.To)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) notePartOf(r asciidocDeploymentRow) {
	d.nsSet[r.From] = true
	d.clusterSet[r.To] = true
	if strings.TrimSpace(d.nsCluster[r.From]) == "" {
		d.nsCluster[r.From] = r.To
	}
}

// addToSetMap adds the value to the set under the key.
// TRLC-LINKS: REQ-EMG-003
func addToSetMap(m map[string]map[string]bool, key, value string) {
	if m[key] == nil {
		m[key] = map[string]bool{}
	}
	m[key][value] = true
}

// firstSorted maps each key to the first of its sorted values.
// TRLC-LINKS: REQ-EMG-003
func firstSorted(m map[string]map[string]bool) map[string]string {
	first := map[string]string{}
	for key, values := range m {
		if names := keysSorted(values); len(names) > 0 {
			first[key] = names[0]
		}
	}
	return first
}

// groupBySorted maps each value to the sorted keys that have it.
// TRLC-LINKS: REQ-EMG-003
func groupBySorted(m map[string]string) map[string][]string {
	grouped := map[string][]string{}
	for key, value := range m {
		grouped[value] = append(grouped[value], key)
	}
	for _, keys := range grouped {
		sort.Strings(keys)
	}
	return grouped
}

// workloadNamespaces maps each workload to the namespaces of the releases that deploy it.
// TRLC-LINKS: REQ-EMG-003
func workloadNamespaces(releaseTargets, releaseDeploys map[string]map[string]bool) map[string]map[string]bool {
	byWorkload := map[string]map[string]bool{}
	for release, targets := range releaseTargets {
		for workload := range releaseDeploys[release] {
			for ns := range targets {
				addToSetMap(byWorkload, workload, ns)
			}
		}
	}
	return byWorkload
}

// indexPlacement places each release and workload in its first namespace.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) indexPlacement() {
	d.releaseNS = firstSorted(d.releaseTargets)
	d.nsReleases = groupBySorted(d.releaseNS)
	d.nsWorkloads = groupBySorted(firstSorted(workloadNamespaces(d.releaseTargets, d.releaseDeploys)))
}

// assignRoles gives each node its role; a later role replaces an earlier one,
// and a namespace is never a cluster.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) assignRoles() {
	for _, role := range []struct {
		set  map[string]bool
		name string
	}{{d.sourceSet, "source"}, {d.kustomSet, "kustomization"}, {d.releaseSet, "release"}, {d.workloadSet, "workload"}, {d.nsSet, "namespace"}} {
		for x := range role.set {
			d.nodeRole[x] = role.name
		}
	}
	for x := range d.clusterSet {
		if !d.nsSet[x] {
			d.nodeRole[x] = "cluster"
		}
	}
}

// deploymentLabelPrefixes are the label prefixes of the roles.
var deploymentLabelPrefixes = map[string]string{
	"source": "Source: ", "kustomization": "Kustomization: ", "release": "Release: ", "workload": "Workload: ",
}

// nodeLabel is the label of a node by its role.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) nodeLabel(name string) string {
	role := d.nodeRole[name]
	switch role {
	case "namespace":
		return "ns/" + runtimeShortName(name)
	case "cluster":
		return "cluster/" + runtimeShortName(name)
	}
	return deploymentLabelPrefixes[role] + name
}

// nodeDecl is the declaration of a node with its label and class.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) nodeDecl(name string) string {
	class := textutil.NonEmpty(d.nodeClass[name], "deployment_element")
	return fmt.Sprintf("%s[\"%s\"]:::%s", deploymentNodeID(name), escapeMermaidLabel(d.nodeLabel(name)), class)
}

// deploymentNodeID is the Mermaid node ID of a deployment element.
// TRLC-LINKS: REQ-EMG-003
func deploymentNodeID(name string) string {
	return "DP_" + codescan.SanitizeNode(name)
}

// emit declares a node once, with the indent.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) emit(indent, name string) {
	d.lines.add(indent + d.nodeDecl(name))
	d.emitted[name] = true
}

// emitNamespaceSubgraph draws a namespace with its releases and workloads.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) emitNamespaceSubgraph(indent, ns string) {
	d.lines.addf("%ssubgraph %s[\"Namespace: %s\"]", indent, "NS_"+codescan.SanitizeNode(ns), escapeMermaidLabel(ns))
	d.lines.add(indent + "  direction TB")
	d.emit(indent+"  ", ns)
	for _, release := range d.nsReleases[ns] {
		d.emit(indent+"  ", release)
	}
	for _, workload := range d.nsWorkloads[ns] {
		d.emit(indent+"  ", workload)
	}
	d.lines.add(indent + "end")
}

// emitNamespaces draws the namespaces inside their clusters, then the
// namespaces without a cluster.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) emitNamespaces() {
	clustered := map[string][]string{}
	unclustered := []string{}
	for _, ns := range keysSorted(d.nsSet) {
		if cluster := strings.TrimSpace(d.nsCluster[ns]); cluster == "" {
			unclustered = append(unclustered, ns)
		} else {
			clustered[cluster] = append(clustered[cluster], ns)
		}
	}
	for _, cluster := range keysSorted(d.clusterSet) {
		d.emitCluster(cluster, clustered[cluster])
	}
	for _, ns := range unclustered {
		d.emitNamespaceSubgraph("  ", ns)
	}
}

// emitCluster draws a cluster with its namespaces, when it has any.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) emitCluster(cluster string, namespaces []string) {
	if len(namespaces) == 0 {
		return
	}
	d.lines.addf("  subgraph %s[\"Cluster: %s\"]", "CLUSTER_"+codescan.SanitizeNode(cluster), escapeMermaidLabel(cluster))
	d.lines.add("    direction TB")
	for _, ns := range namespaces {
		d.emitNamespaceSubgraph("    ", ns)
	}
	d.lines.add("  end")
}

// emitControlPlane draws the sources, kustomizations and releases without a namespace.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) emitControlPlane() {
	nodes := map[string]bool{}
	for n := range d.sourceSet {
		nodes[n] = true
	}
	for n := range d.kustomSet {
		nodes[n] = true
	}
	for n := range d.releaseSet {
		if strings.TrimSpace(d.releaseNS[n]) == "" {
			nodes[n] = true
		}
	}
	if len(nodes) == 0 {
		return
	}
	d.lines.add(`  subgraph CONTROL_PLANE["Control Plane"]`)
	d.lines.add("    direction TB")
	for _, n := range keysSorted(nodes) {
		if !d.emitted[n] {
			d.emit("    ", n)
		}
	}
	d.lines.add("  end")
}

// emitStandalone draws the nodes no subgraph holds, and the clusters without namespaces.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) emitStandalone() {
	for _, n := range sortedKeys(d.nodeClass) {
		if !d.clusterSet[n] && !d.emitted[n] {
			d.emit("  ", n)
		}
	}
	for _, cluster := range keysSorted(d.clusterSet) {
		if !d.clusterHasNamespace(cluster) && !d.emitted[cluster] {
			d.emit("  ", cluster)
		}
	}
}

// clusterHasNamespace reports whether a namespace is part of the cluster.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) clusterHasNamespace(cluster string) bool {
	for _, c := range d.nsCluster {
		if strings.TrimSpace(c) == strings.TrimSpace(cluster) {
			return true
		}
	}
	return false
}

// emitEdges draws the rows, except the cluster membership that the subgraphs show.
// TRLC-LINKS: REQ-EMG-003
func (d *deploymentDiagram) emitEdges() {
	for _, r := range d.edges {
		if r.How != "part_of" {
			d.lines.addf("  %s -->|%s| %s", deploymentNodeID(r.From), escapeMermaidLabel(r.How), deploymentNodeID(r.To))
		}
	}
}

// sortedKeys is the sorted keys of a map.
// TRLC-LINKS: REQ-EMG-003
func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TRLC-LINKS: REQ-EMG-003
func deploymentNodeName(r codescan.RuntimeItem) string {
	name := strings.TrimSpace(strings.ReplaceAll(r.Name, "\\", "/"))
	if name == "" {
		return name
	}
	switch strings.ToLower(strings.TrimSpace(r.Kind)) {
	case "helmrelease", "deployment", "workload", "service":
		return name
	default:
		return runtimeShortName(name)
	}
}

// platformGroups are the functional groups whose units operate the platform.
var platformGroups = map[string]bool{"FG-PLATFORM": true, "FG-PLATFORM-OPERATIONS": true}

// platformUnitNames maps each unit of a platform group to its name.
// TRLC-LINKS: REQ-EMG-003
func platformUnitNames(units []model.FunctionalUnit) map[string]string {
	names := map[string]string{}
	for _, u := range units {
		if platformGroups[strings.TrimSpace(strings.ToUpper(u.Group))] {
			names[u.ID] = textutil.NonEmpty(u.Name, u.ID)
		}
	}
	return names
}

// platformUnitTargets lists the runtime elements each unit operates: those it
// owns, and by convention the cluster and GitOps elements of the platform units.
// TRLC-LINKS: REQ-EMG-003
func platformUnitTargets(runtime []codescan.RuntimeItem) map[string][]string {
	unitTargets := map[string][]string{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		if owner != "" && owner != "unresolved" {
			unitTargets[owner] = append(unitTargets[owner], runtimeShortName(r.Name))
		}
	}
	// Convention fallback for platform provisioning artifacts.
	for _, r := range runtime {
		switch r.Kind {
		case "cluster", "namespace":
			unitTargets["FU-CLUSTER-PROVISIONING"] = append(unitTargets["FU-CLUSTER-PROVISIONING"], runtimeShortName(r.Name))
		case "gitrepository", "kustomization", "helmrelease":
			unitTargets["FU-GITOPS-OPERATIONS"] = append(unitTargets["FU-GITOPS-OPERATIONS"], runtimeShortName(r.Name))
		}
	}
	for k, v := range unitTargets {
		unitTargets[k] = textutil.UniqueSorted(v)
	}
	return unitTargets
}

// actorLabel is the name of the actor with the ID, or the ID.
// TRLC-LINKS: REQ-EMG-003
func actorLabel(actors []model.Actor, id string) string {
	for _, x := range actors {
		if x.ID == id {
			return textutil.NonEmpty(x.Name, x.ID)
		}
	}
	return id
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
// TRLC-LINKS: REQ-EMG-003
func buildPlatformOpsRows(a model.AuthoredArchitecture, runtime []codescan.RuntimeItem) []asciidocPlatformOpRow {
	platformUnits := platformUnitNames(a.FunctionalUnits)
	unitTargets := platformUnitTargets(runtime)
	out := []asciidocPlatformOpRow{}
	for _, m := range a.Mappings {
		unitName, ok := platformUnits[m.To]
		if m.Type != "interacts_with" || !ok {
			continue
		}
		actorName := actorLabel(a.Actors, m.From)
		targets := unitTargets[m.To]
		if len(targets) == 0 {
			targets = []string{"platform control operations"}
		}
		for _, t := range targets {
			out = append(out, asciidocPlatformOpRow{Actor: actorName, Unit: unitName, Target: t})
		}
	}
	return out
}

// runtimeTargetWords mark a platform target as a runtime element.
var runtimeTargetWords = []string{"deployment", "workload", "pod", "service"}

// isRuntimeTarget reports whether the target names a runtime element.
// TRLC-LINKS: REQ-EMG-003
func isRuntimeTarget(target string) bool {
	lower := strings.ToLower(target)
	for _, word := range runtimeTargetWords {
		if strings.Contains(lower, word) {
			return true
		}
	}
	return false
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, DEP-LOCAL-WORKSPACE, DEP-CI-PIPELINE
// TRLC-LINKS: REQ-EMG-003
func buildPlatformOpsMermaid(rows []asciidocPlatformOpRow) string {
	lines := mermaidLines{"flowchart LR"}
	for _, r := range rows {
		an := "ACT_" + codescan.SanitizeNode(r.Actor)
		un := "PFU_" + codescan.SanitizeNode(r.Unit)
		tn := "TGT_" + codescan.SanitizeNode(r.Target)
		targetClass := "deployment_element"
		if isRuntimeTarget(r.Target) {
			targetClass = "runtime_element"
		}
		lines.addf("  %s((\"%s\")):::actor", an, escapeMermaidLabel(r.Actor))
		lines.addf("  %s[\"%s\"]:::functional_unit", un, escapeMermaidLabel(r.Unit))
		lines.addf("  %s[\"%s\"]:::%s", tn, escapeMermaidLabel(r.Target), targetClass)
		lines.addf("  %s --> %s", an, un)
		lines.addf("  %s --> %s", un, tn)
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

// dependencySummaries lists the labels of what each unit depends on.
// TRLC-LINKS: REQ-EMG-003
func dependencySummaries(mappings []model.Mapping, labels map[string]string) map[string][]string {
	depsByTarget := map[string][]string{}
	for _, m := range mappings {
		if m.Type == "depends_on" {
			depsByTarget[m.From] = append(depsByTarget[m.From], textutil.NonEmpty(labels[m.To], m.To))
		}
	}
	for k, v := range depsByTarget {
		depsByTarget[k] = textutil.UniqueSorted(v)
	}
	return depsByTarget
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
// TRLC-LINKS: REQ-EMG-003
func buildSecurityPathRows(a model.AuthoredArchitecture, labels map[string]string) []asciidocSecurityPathRow {
	depsByTarget := dependencySummaries(a.Mappings, labels)
	out := []asciidocSecurityPathRow{}
	seen := map[string]bool{}
	for _, m := range a.Mappings {
		if m.Type != "targets" {
			continue
		}
		attack := textutil.NonEmpty(labels[m.From], m.From)
		target := textutil.NonEmpty(labels[m.To], m.To)
		depSummary := textutil.NonEmpty(strings.Join(depsByTarget[m.To], ", "), "none")
		key := attack + "|" + target + "|" + depSummary
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, asciidocSecurityPathRow{
			AttackVectorID: strings.TrimSpace(m.From),
			AttackVector:   attack,
			TargetID:       strings.TrimSpace(m.To),
			Target:         target,
			DependsOn:      depSummary,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return cmp.Or(cmp.Compare(out[i].AttackVector, out[j].AttackVector), cmp.Compare(out[i].Target, out[j].Target)) < 0
	})
	return out
}

// securityPathEvidence is the runtime and code labels of each unit that a
// security path diagram hangs under its targets.
type securityPathEvidence struct {
	runtimeByOwner map[string][]string
	codeByOwner    map[string][]string
}

// newSecurityPathEvidence indexes the runtime and code evidence of the units.
// TRLC-LINKS: REQ-EMG-003
func newSecurityPathEvidence(runtime []codescan.RuntimeItem, code []codescan.CodeItem) securityPathEvidence {
	return securityPathEvidence{runtimeByOwner: securityRuntimeLabels(runtime), codeByOwner: securityCodeLabels(code)}
}

// securityRuntimeLabels lists the sorted runtime element names of each resolved owner.
// TRLC-LINKS: REQ-EMG-003
func securityRuntimeLabels(runtime []codescan.RuntimeItem) map[string][]string {
	byOwner := map[string][]string{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		label := runtimeShortName(strings.TrimSpace(r.Name))
		if owner != "" && owner != "unresolved" && label != "" {
			byOwner[owner] = append(byOwner[owner], label)
		}
	}
	for owner, items := range byOwner {
		byOwner[owner] = textutil.UniqueSorted(items)
	}
	return byOwner
}

// securityCodeLabels lists the grouped code element labels of each resolved owner.
// TRLC-LINKS: REQ-EMG-003
func securityCodeLabels(code []codescan.CodeItem) map[string][]string {
	rawByOwner := map[string][]string{}
	allRaw := []string{}
	for _, c := range code {
		if owner, label, ok := securityCodeElement(c); ok {
			rawByOwner[owner] = append(rawByOwner[owner], label)
			allRaw = append(allRaw, label)
		}
	}
	byOwner := map[string][]string{}
	lookup := codescan.EvidenceLabelLookup(allRaw)
	for owner, items := range rawByOwner {
		byOwner[owner] = codescan.GroupedFileLabels(items, lookup)
	}
	return byOwner
}

// securityCodeElement is the owner and evidence element of a source file or
// symbol of a resolved owner, other than verification code.
// TRLC-LINKS: REQ-EMG-003
func securityCodeElement(c codescan.CodeItem) (owner, label string, ok bool) {
	owner = strings.TrimSpace(c.Owner)
	if codescan.IsVerificationCodeItem(c) || owner == "" || owner == "unresolved" {
		return "", "", false
	}
	if kind := strings.TrimSpace(c.Kind); kind != "source_file" && kind != "symbol" {
		return "", "", false
	}
	if codescan.ItemPath(c) == "" {
		return "", "", false
	}
	label = codescan.ItemEvidenceElement(c)
	return owner, label, strings.TrimSpace(label) != ""
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildSecurityPathMermaid(rows []asciidocSecurityPathRow, runtime []codescan.RuntimeItem, code []codescan.CodeItem) string {
	evidence := newSecurityPathEvidence(runtime, code)
	lines := mermaidLines{"flowchart LR"}
	for _, r := range rows {
		evidence.writeRow(&lines, r)
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

// writeRow draws an attack vector, its target, the dependencies of the target
// and the runtime and code evidence of the target.
// TRLC-LINKS: REQ-EMG-003
func (e securityPathEvidence) writeRow(m *mermaidLines, r asciidocSecurityPathRow) {
	targetID := textutil.NonEmpty(strings.TrimSpace(r.TargetID), r.Target)
	avNode := "AV_" + codescan.SanitizeNode(textutil.NonEmpty(strings.TrimSpace(r.AttackVectorID), r.AttackVector))
	tNode := "SEC_TGT_" + codescan.SanitizeNode(targetID)
	m.add(mermaidTypedNodeLine(avNode, r.AttackVector, "attack_vector"))
	m.add(mermaidTypedNodeLine(tNode, r.Target, "functional_unit"))
	m.addf("  %s -->|targets| %s", avNode, tNode)
	writeSecurityDependencies(m, tNode, r.DependsOn)
	runtimeNodes := e.writeRuntime(m, tNode, targetID, r.Target)
	// Keep code evidence traced via runtime, even when no direct runtime artifact is inferred.
	for _, mod := range e.codeByOwner[targetID] {
		codeNode := "CODE_" + codescan.SanitizeNode(codescan.EvidenceFileKey(mod))
		m.add(mermaidTypedNodeLine(codeNode, mod, "code_element"))
		if len(runtimeNodes) > 0 {
			m.addf("  %s -->|implemented_by| %s", runtimeNodes[0], codeNode)
		}
	}
}

// writeSecurityDependencies draws what the target depends on, from the comma
// separated summary.
// TRLC-LINKS: REQ-EMG-003
func writeSecurityDependencies(m *mermaidLines, tNode, dependsOn string) {
	for _, dep := range strings.Split(dependsOn, ",") {
		dep = strings.TrimSpace(dep)
		if dep == "" || dep == "none" {
			continue
		}
		dNode := "SEC_DEP_" + codescan.SanitizeNode(dep)
		m.add(mermaidTypedNodeLine(dNode, dep, "referenced_element"))
		m.addf("  %s -->|depends_on| %s", tNode, dNode)
	}
}

// writeRuntime draws the runtime elements of the target, or one inferred
// runtime when it has none but code. It returns their nodes.
// TRLC-LINKS: REQ-EMG-003
func (e securityPathEvidence) writeRuntime(m *mermaidLines, tNode, targetID, target string) []string {
	runtimeNodes := make([]string, 0, len(e.runtimeByOwner[targetID]))
	for _, rt := range e.runtimeByOwner[targetID] {
		rtNode := "SEC_RT_" + codescan.SanitizeNode(targetID+"-"+rt)
		m.add(mermaidTypedNodeLine(rtNode, rt, "runtime_element"))
		m.addf("  %s -->|runtime| %s", tNode, rtNode)
		runtimeNodes = append(runtimeNodes, rtNode)
	}
	if len(runtimeNodes) == 0 && len(e.codeByOwner[targetID]) > 0 {
		rtLabel := textutil.NonEmpty(strings.TrimSpace(target), targetID) + " runtime (inferred)"
		rtNode := "SEC_RT_" + codescan.SanitizeNode(targetID+"-runtime")
		m.add(mermaidTypedNodeLine(rtNode, rtLabel, "runtime_element"))
		m.addf("  %s -->|runtime| %s", tNode, rtNode)
		runtimeNodes = append(runtimeNodes, rtNode)
	}
	return runtimeNodes
}

// securityContextDiagram draws the context data flow diagram of the security view.
type securityContextDiagram struct {
	lines     mermaidLines
	added     map[string]bool
	connected map[string]bool
	labels    map[string]string
}

// add appends a line the first time.
// TRLC-LINKS: REQ-EMG-003
func (c *securityContextDiagram) add(line string) {
	if !c.added[line] {
		c.added[line] = true
		c.lines.add(line)
	}
}

// markEdge records that both ends of an edge are connected.
// TRLC-LINKS: REQ-EMG-003
func (c *securityContextDiagram) markEdge(fromID, toID string) {
	for _, id := range []string{fromID, toID} {
		if id = strings.TrimSpace(id); id != "" {
			c.connected[id] = true
		}
	}
}

// node draws a node with its label and class.
// TRLC-LINKS: REQ-EMG-003
func (c *securityContextDiagram) node(id, class string) {
	c.add(mermaidTypedNodeLine("CTX_"+codescan.SanitizeNode(id), textutil.NonEmpty(c.labels[id], id), class))
}

// edge draws an edge between two nodes and marks them connected.
// TRLC-LINKS: REQ-EMG-003
func (c *securityContextDiagram) edge(from, label, to string) {
	c.add(fmt.Sprintf("  CTX_%s -->|%s| CTX_%s", codescan.SanitizeNode(from), label, codescan.SanitizeNode(to)))
	c.markEdge(from, to)
}

// addNodes draws a node of the class for each ID that is not blank.
// TRLC-LINKS: REQ-EMG-003
func (c *securityContextDiagram) addNodes(ids []string, class string) {
	for _, id := range ids {
		if id = strings.TrimSpace(id); id != "" {
			c.node(id, class)
		}
	}
}

// securityContextDataFlowTypes are the mapping types that attach data and interfaces to units.
var securityContextDataFlowTypes = map[string]bool{
	"contains": true, "reads": true, "writes": true, "publishes": true, "subscribes": true, "streams": true,
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, TB-REPO-WORKSPACE, TB-EXTERNAL-VALIDATION-TOOLS
// TRLC-LINKS: REQ-EMG-003
func buildSecurityContextDFDMermaid(a model.AuthoredArchitecture, labels map[string]string) string {
	c := &securityContextDiagram{lines: mermaidLines{"flowchart LR"}, added: map[string]bool{}, connected: map[string]bool{}, labels: labels}
	c.addNodes(interfaceIDs(a.Interfaces), "interface")
	c.addNodes(dataObjectIDs(a.DataObjects), "data_object")
	c.addNodes(trustBoundaryIDs(a.TrustBoundaries), "trust_boundary")
	c.addMappings(a.Mappings)
	// Explicitly include data-flow and ownership links in context diagrams so
	// data/interface nodes are visually connected (not floating).
	c.addDataFlows(a.Mappings)
	// Keep context diagram connected when data/interface nodes are authored but only weakly linked.
	c.addInterfaceOwners(a.Interfaces)
	c.lines = appendMermaidClassDefs(c.lines)
	return strings.Join(uniquePreserve(c.lines), "\n")
}

// interfaceIDs is the IDs of the interfaces.
// TRLC-LINKS: REQ-EMG-003
func interfaceIDs(items []model.Interface) []string {
	ids := make([]string, 0, len(items))
	for _, x := range items {
		ids = append(ids, x.ID)
	}
	return ids
}

// dataObjectIDs is the IDs of the data objects.
// TRLC-LINKS: REQ-EMG-003
func dataObjectIDs(items []model.DataObject) []string {
	ids := make([]string, 0, len(items))
	for _, x := range items {
		ids = append(ids, x.ID)
	}
	return ids
}

// trustBoundaryIDs is the IDs of the trust boundaries.
// TRLC-LINKS: REQ-EMG-003
func trustBoundaryIDs(items []model.TrustBoundary) []string {
	ids := make([]string, 0, len(items))
	for _, x := range items {
		ids = append(ids, x.ID)
	}
	return ids
}

// addMappings draws the mappings of the context types between the IDs they name.
// TRLC-LINKS: REQ-EMG-003
func (c *securityContextDiagram) addMappings(mappings []model.Mapping) {
	for _, m := range mappings {
		t := strings.TrimSpace(m.Type)
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		if !securityContextMappingType(t) || from == "" || to == "" {
			continue
		}
		c.node(from, securityDFDClassForID(from))
		c.node(to, securityDFDClassForID(to))
		c.edge(from, t, to)
	}
}

// addDataFlows draws the data flow and ownership mappings from units to data
// objects, interfaces and trust boundaries.
// TRLC-LINKS: REQ-EMG-003
func (c *securityContextDiagram) addDataFlows(mappings []model.Mapping) {
	for _, m := range mappings {
		t := strings.TrimSpace(m.Type)
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		if !securityContextDataFlowTypes[t] || !strings.HasPrefix(from, "FU-") || !hasAnyPrefix(to, "DO-", "IF-", "TB-") {
			continue
		}
		c.node(from, "functional_unit")
		c.node(to, securityDFDClassForID(to))
		c.edge(from, t, to)
	}
}

// hasAnyPrefix reports whether s starts with one of the prefixes.
// TRLC-LINKS: REQ-EMG-003
func hasAnyPrefix(s string, prefixes ...string) bool {
	for _, p := range prefixes {
		if strings.HasPrefix(s, p) {
			return true
		}
	}
	return false
}

// addInterfaceOwners connects each unconnected interface to the unit that owns it.
// TRLC-LINKS: REQ-EMG-003
func (c *securityContextDiagram) addInterfaceOwners(interfaces []model.Interface) {
	for _, iface := range interfaces {
		id := strings.TrimSpace(iface.ID)
		owner := strings.TrimSpace(iface.Owner)
		if id == "" || c.connected[id] || !strings.HasPrefix(owner, "FU-") {
			continue
		}
		c.node(owner, "functional_unit")
		c.node(id, "interface")
		c.edge(owner, "contains", id)
	}
}

// groupContext is the units of each group that a group context diagram is drawn for.
type groupContext struct {
	unitGroupByID map[string]string
	groupUnits    map[string]map[string]bool
}

// newGroupContext indexes the units that have an ID and a group.
// TRLC-LINKS: REQ-EMG-003
func newGroupContext(units []model.FunctionalUnit) groupContext {
	g := groupContext{unitGroupByID: map[string]string{}, groupUnits: map[string]map[string]bool{}}
	for _, fu := range units {
		id := strings.TrimSpace(fu.ID)
		gid := strings.TrimSpace(fu.Group)
		if id != "" && gid != "" {
			g.unitGroupByID[id] = gid
			addToSetMap(g.groupUnits, gid, id)
		}
	}
	return g
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildSecurityContextDFDMermaidByGroup(a model.AuthoredArchitecture, complianceMappings []model.ComplianceMapping, labels map[string]string) []asciidocSecurityContextDiagram {
	ctx := newGroupContext(a.FunctionalUnits)
	ownedByGroup := securityContextOwnedIDsByGroup(a, complianceMappings, ctx)
	out := []asciidocSecurityContextDiagram{}
	for _, fg := range a.FunctionalGroups {
		groupID := strings.TrimSpace(fg.ID)
		if groupID == "" || len(ctx.groupUnits[groupID]) == 0 {
			continue
		}
		d := &groupContextDiagram{
			groupID: groupID,
			name:    textutil.NonEmpty(labels[groupID], textutil.NonEmpty(fg.Name, groupID)),
			labels:  labels,
			units:   ctx.groupUnits[groupID],
			owned:   unionSets(ownedByGroup[groupID], ctx.groupUnits[groupID]),
		}
		d.included = unionSets(d.owned)
		out = append(out, asciidocSecurityContextDiagram{GroupID: groupID, GroupName: d.name, Mermaid: d.render(a.Mappings)})
	}
	return out
}

// unionSets is a new set of the members of all the sets.
// TRLC-LINKS: REQ-EMG-003
func unionSets(sets ...map[string]bool) map[string]bool {
	union := map[string]bool{}
	for _, set := range sets {
		for id := range set {
			union[id] = true
		}
	}
	return union
}

// groupContextDiagram draws the data flow context of one functional group.
type groupContextDiagram struct {
	groupID  string
	name     string
	labels   map[string]string
	units    map[string]bool
	owned    map[string]bool
	included map[string]bool
}

// render draws the group with the elements it owns, the elements outside it
// that the mappings connect to them, and the edges.
// TRLC-LINKS: REQ-EMG-003
func (d *groupContextDiagram) render(mappings []model.Mapping) string {
	edges := d.collectEdges(mappings)
	lines := mermaidLines{"flowchart LR"}
	boxID := "CTXGROUP_" + codescan.SanitizeNode(d.groupID)
	lines.addf("  subgraph %s[\"%s\"]", boxID, escapeMermaidLabel(d.name))
	lines.add("    direction TB")
	for _, id := range keysSorted(d.owned) {
		lines.add(indentMermaidLine("    ", d.nodeLine(id)))
	}
	lines.add("  end")
	lines.addf("  style %s fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;", boxID)
	for _, id := range d.externalIDs() {
		lines.add(d.nodeLine(id))
	}
	lines = append(lines, edges...)
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

// nodeLine is the typed node of an element, labeled and classed by its ID.
// TRLC-LINKS: REQ-EMG-003
func (d *groupContextDiagram) nodeLine(id string) string {
	return mermaidTypedNodeLine("CTX_"+codescan.SanitizeNode(id), textutil.NonEmpty(d.labels[id], id), securityDFDClassForID(id))
}

// externalIDs is the sorted IDs the edges connect to the group that it does not own.
// TRLC-LINKS: REQ-EMG-003
func (d *groupContextDiagram) externalIDs() []string {
	external := map[string]bool{}
	for id := range d.included {
		if !d.owned[id] && id != d.groupID {
			external[id] = true
		}
	}
	return keysSorted(external)
}

// collectEdges lists the mappings that touch an owned element, once each, and
// includes their ends in the diagram.
// TRLC-LINKS: REQ-EMG-003
func (d *groupContextDiagram) collectEdges(mappings []model.Mapping) []string {
	edges := []string{}
	seen := map[string]bool{}
	for _, m := range mappings {
		t := strings.TrimSpace(m.Type)
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		if !d.includesMapping(t, from, to) {
			continue
		}
		d.included[from] = true
		d.included[to] = true
		edge := fmt.Sprintf("  CTX_%s -->|%s| CTX_%s", codescan.SanitizeNode(from), escapeMermaidLabel(t), codescan.SanitizeNode(to))
		if !seen[edge] {
			seen[edge] = true
			edges = append(edges, edge)
		}
	}
	return edges
}

// includesMapping reports whether a mapping of a context type touches an owned
// element, other than the containment of a unit by the group itself.
// TRLC-LINKS: REQ-EMG-003
func (d *groupContextDiagram) includesMapping(t, from, to string) bool {
	if !securityContextMappingType(t) || from == "" || to == "" {
		return false
	}
	if t == "contains" && from == d.groupID && d.units[to] {
		return false
	}
	return d.owned[from] || d.owned[to]
}

// indentMermaidLine puts a line at the indent, replacing its own.
// TRLC-LINKS: REQ-EMG-003
func indentMermaidLine(indent, line string) string {
	return indent + strings.TrimLeft(line, " ")
}

// securityOwnership collects the elements each group owns in the security context.
type securityOwnership struct {
	ctx   groupContext
	owned map[string]map[string]bool
}

// add makes the group the owner of the element, unless either is blank or the
// element is an actor or referenced element.
// TRLC-LINKS: REQ-EMG-003
func (o *securityOwnership) add(groupID, id string) {
	groupID = strings.TrimSpace(groupID)
	id = strings.TrimSpace(id)
	if groupID == "" || id == "" || hasAnyPrefix(id, "ACT-", "REF-") {
		return
	}
	addToSetMap(o.owned, groupID, id)
}

// securityContextOwnedIDsByGroup maps each group to the units, interfaces,
// controls, trust boundaries and data it owns.
// TRLC-LINKS: REQ-EMG-003
func securityContextOwnedIDsByGroup(a model.AuthoredArchitecture, complianceMappings []model.ComplianceMapping, ctx groupContext) map[string]map[string]bool {
	o := &securityOwnership{ctx: ctx, owned: map[string]map[string]bool{}}
	for _, fu := range a.FunctionalUnits {
		o.add(ctx.unitGroupByID[strings.TrimSpace(fu.ID)], strings.TrimSpace(fu.ID))
	}
	for _, iface := range a.Interfaces {
		o.add(ctx.unitGroupByID[strings.TrimSpace(iface.Owner)], strings.TrimSpace(iface.ID))
	}
	o.addControls(complianceMappings)
	o.addTrustBoundaries(a.TrustBoundaries)
	o.addMappedData(a.Mappings)
	return o.owned
}

// addControls makes the group of each unit or group a compliance mapping
// applies to the owner of the control.
// TRLC-LINKS: REQ-EMG-003
func (o *securityOwnership) addControls(complianceMappings []model.ComplianceMapping) {
	for _, mapping := range complianceMappings {
		controlRef := strings.TrimSpace(mapping.ModelControlRef)
		for _, target := range mapping.AppliesTo {
			o.addControl(strings.TrimSpace(target), controlRef)
		}
	}
}

// addControl makes the group of the target the owner of the control.
// TRLC-LINKS: REQ-EMG-003
func (o *securityOwnership) addControl(target, controlRef string) {
	if strings.HasPrefix(target, "FU-") {
		o.add(o.ctx.unitGroupByID[target], controlRef)
	}
	if strings.HasPrefix(target, "FG-") {
		o.add(target, controlRef)
	}
}

// addTrustBoundaries makes the group the owner of the trust boundaries whose
// members are all units of that group.
// TRLC-LINKS: REQ-EMG-003
func (o *securityOwnership) addTrustBoundaries(boundaries []model.TrustBoundary) {
	for _, tb := range boundaries {
		id := strings.TrimSpace(tb.ID)
		if id == "" || len(tb.Members) == 0 {
			continue
		}
		if groupID, ok := o.localGroup(tb.Members); ok {
			o.add(groupID, id)
		}
	}
}

// localGroup is the group that all the members are units of, if there is one.
// TRLC-LINKS: REQ-EMG-003
func (o *securityOwnership) localGroup(members []string) (string, bool) {
	groupID := ""
	for _, member := range members {
		member = strings.TrimSpace(member)
		memberGroup := o.ctx.unitGroupByID[member]
		if !strings.HasPrefix(member, "FU-") || memberGroup == "" {
			return "", false
		}
		if groupID == "" {
			groupID = memberGroup
		}
		if groupID != memberGroup || !o.ctx.groupUnits[groupID][member] {
			return "", false
		}
	}
	return groupID, true
}

// addMappedData makes the group of a unit the owner of what the unit contains
// and of the data objects and interfaces it writes or publishes.
// TRLC-LINKS: REQ-EMG-003
func (o *securityOwnership) addMappedData(mappings []model.Mapping) {
	for _, m := range mappings {
		to := strings.TrimSpace(m.To)
		if ownsMappedTarget(strings.TrimSpace(m.Type), to) {
			o.add(o.ctx.unitGroupByID[strings.TrimSpace(m.From)], to)
		}
	}
}

// ownsMappedTarget reports whether a mapping of the type makes its source
// own the target: containment, or writing or publishing a data object or interface.
// TRLC-LINKS: REQ-EMG-003
func ownsMappedTarget(mappingType, target string) bool {
	if mappingType == "contains" {
		return true
	}
	return (mappingType == "writes" || mappingType == "publishes") && hasAnyPrefix(target, "DO-", "IF-")
}

// securityContextMappingTypes are the mapping types a security context diagram shows.
var securityContextMappingTypes = map[string]bool{
	"interacts_with": true, "depends_on": true, "calls": true, "bounded_by": true, "contains": true,
	"reads": true, "writes": true, "publishes": true, "subscribes": true, "streams": true,
}

// TRLC-LINKS: REQ-EMG-003
func securityContextMappingType(t string) bool {
	return securityContextMappingTypes[strings.TrimSpace(t)]
}

// securityDFDClasses maps the ID prefix of an element to its diagram class, by
// the first prefix that matches.
var securityDFDClasses = []struct{ prefix, class string }{
	{"ACT-", "actor"}, {"FG-", "functional_group"}, {"FU-", "functional_unit"}, {"AV-", "attack_vector"},
	{"IF-", "interface"}, {"DO-", "data_object"}, {"TB-", "trust_boundary"}, {"CTRL-", "control"},
	{"DEP-", "deployment_target"}, {"STATE-", "state"}, {"EVT-", "event"}, {"FLOW-", "flow"},
}

// TRLC-LINKS: REQ-EMG-003
func securityDFDClassForID(id string) string {
	id = strings.TrimSpace(id)
	for _, c := range securityDFDClasses {
		if strings.HasPrefix(id, c.prefix) {
			return c.class
		}
	}
	return "referenced_element"
}

// mermaidNodeShapes maps a node class to its Mermaid node syntax, filled with
// the node ID, the label and the class.
var mermaidNodeShapes = map[string]string{
	"actor":          "  %s((\"%s\")):::%s",
	"attack_vector":  "  %s((\"%s\")):::%s",
	"interface":      "  %s[/\"%s\"/]:::%s",
	"data_object":    "  %s[(\"%s\")]:::%s",
	"control":        "  %s[[\"%s\"]]:::%s",
	"trust_boundary": "  %s[/\"%s\"\\]:::%s",
	"state":          "  %s([\"%s\"]):::%s",
	"event":          "  %s((\"%s\")):::%s",
	"flow":           "  %s{\"%s\"}:::%s",
	"flow_step":      "  %s>\"%s\"]:::%s",
}

// TRLC-LINKS: REQ-EMG-003
func mermaidTypedNodeLine(nodeID, label, className string) string {
	shape, ok := mermaidNodeShapes[strings.TrimSpace(className)]
	if !ok {
		shape = "  %s[\"%s\"]:::%s"
	}
	return fmt.Sprintf(shape, nodeID, escapeMermaidLabel(label), className)
}

// observabilityKindSignals maps a runtime kind to the signal and layer it yields.
var observabilityKindSignals = map[string]asciidocSecurityObsRow{
	"ingress":            {Signal: "ingress access and suspicious request logs", Layer: "runtime"},
	"service":            {Signal: "runtime request, error, and dependency logs", Layer: "runtime"},
	"deployment":         {Signal: "runtime request, error, and dependency logs", Layer: "runtime"},
	"workload":           {Signal: "runtime request, error, and dependency logs", Layer: "runtime"},
	"pod":                {Signal: "runtime request, error, and dependency logs", Layer: "runtime"},
	"helmrelease":        {Signal: "deployment and platform audit events", Layer: "deployment"},
	"kustomization":      {Signal: "deployment and platform audit events", Layer: "deployment"},
	"gitrepository":      {Signal: "deployment and platform audit events", Layer: "deployment"},
	"cluster":            {Signal: "deployment and platform audit events", Layer: "deployment"},
	"namespace":          {Signal: "deployment and platform audit events", Layer: "deployment"},
	"terraform_resource": {Signal: "deployment and platform audit events", Layer: "deployment"},
}

// observabilityWordSignal says which words in a name or path yield a signal.
type observabilityWordSignal struct {
	words  []string
	signal string
}

var runtimeWordSignals = []observabilityWordSignal{
	{[]string{"auth", "token"}, "authentication and token misuse events"},
	{[]string{"risk", "fraud"}, "abuse and fraud decision logs"},
}

var codeWordSignals = []observabilityWordSignal{
	{[]string{"log", "audit", "trace"}, "application security telemetry hooks"},
	{[]string{"auth", "token"}, "authorization and credential handling checks"},
	{[]string{"risk", "fraud"}, "fraud and abuse detection code paths"},
}

// observabilityRows collects security observability rows, each once.
type observabilityRows struct {
	rows []asciidocSecurityObsRow
	seen map[string]bool
}

// add records a row when it has a signal, a layer and evidence, and is new; a
// blank owner is unresolved.
// TRLC-LINKS: REQ-EMG-003
func (o *observabilityRows) add(row asciidocSecurityObsRow) {
	row.Signal = strings.TrimSpace(row.Signal)
	row.Layer = strings.TrimSpace(row.Layer)
	row.Owner = textutil.NonEmpty(strings.TrimSpace(row.Owner), "unresolved")
	row.Evidence = strings.TrimSpace(row.Evidence)
	key := row.Signal + "|" + row.Layer + "|" + row.Owner + "|" + row.Evidence
	if row.Signal == "" || row.Layer == "" || row.Evidence == "" || o.seen[key] {
		return
	}
	o.seen[key] = true
	o.rows = append(o.rows, row)
}

// addWordSignals adds a row for each signal that has a word in the text.
// TRLC-LINKS: REQ-EMG-003
func (o *observabilityRows) addWordSignals(signals []observabilityWordSignal, text string, row asciidocSecurityObsRow) {
	for _, s := range signals {
		if containsAny(text, s.words) {
			row.Signal = s.signal
			o.add(row)
		}
	}
}

// containsAny reports whether s contains one of the words.
// TRLC-LINKS: REQ-EMG-003
func containsAny(s string, words []string) bool {
	for _, w := range words {
		if strings.Contains(s, w) {
			return true
		}
	}
	return false
}

// addRuntime adds the signals of a runtime element: by its kind and by the
// words in its name.
// TRLC-LINKS: REQ-EMG-003
func (o *observabilityRows) addRuntime(r codescan.RuntimeItem) {
	evidence := r.Kind + " " + runtimeShortName(r.Name)
	if kindRow, ok := observabilityKindSignals[r.Kind]; ok {
		o.add(asciidocSecurityObsRow{Signal: kindRow.Signal, Layer: kindRow.Layer, Owner: r.Owner, Evidence: evidence})
	}
	o.addWordSignals(runtimeWordSignals, strings.ToLower(runtimeShortName(r.Name)), asciidocSecurityObsRow{Layer: "runtime", Owner: r.Owner, Evidence: evidence})
}

// addCode adds the signals of a code item by the words in its path.
// TRLC-LINKS: REQ-EMG-003
func (o *observabilityRows) addCode(c codescan.CodeItem) {
	if codescan.IsVerificationCodeItem(c) {
		return
	}
	o.addWordSignals(codeWordSignals, strings.ToLower(codescan.ItemPath(c)), asciidocSecurityObsRow{Layer: "code", Owner: c.Owner, Evidence: "code " + codescan.ItemPath(c)})
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildSecurityObservabilityRows(runtime []codescan.RuntimeItem, code []codescan.CodeItem) []asciidocSecurityObsRow {
	o := &observabilityRows{rows: []asciidocSecurityObsRow{}, seen: map[string]bool{}}
	for _, r := range runtime {
		o.addRuntime(r)
	}
	for _, c := range code {
		o.addCode(c)
	}
	out := o.rows
	sort.SliceStable(out, func(i, j int) bool {
		return cmp.Or(cmp.Compare(out[i].Layer, out[j].Layer), cmp.Compare(out[i].Signal, out[j].Signal),
			cmp.Compare(out[i].Owner, out[j].Owner), cmp.Compare(out[i].Evidence, out[j].Evidence)) < 0
	})
	if len(out) > 28 {
		return out[:28]
	}
	return out
}
