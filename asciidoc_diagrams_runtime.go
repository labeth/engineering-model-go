package engmodel

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

func buildRuntimeAPIRows(runtime []inferredRuntimeItem, mappings []model.Mapping) []asciidocRuntimeAPIRow {
	fuToRuntime := map[string]string{}
	servicePorts := map[string]string{}
	isRuntimeEndpoint := func(kind string) bool {
		switch strings.TrimSpace(strings.ToLower(kind)) {
		case "helmrelease", "deployment", "workload", "service", "lambda_function", "terraform_resource", "eventbridge_rule", "eventbridge_target", "queue", "topic":
			return true
		default:
			return false
		}
	}
	for _, r := range runtime {
		name := runtimeShortName(r.Name)
		if strings.TrimSpace(r.Owner) != "" && strings.TrimSpace(r.Owner) != "unresolved" && isRuntimeEndpoint(r.Kind) {
			if _, ok := fuToRuntime[r.Owner]; !ok {
				fuToRuntime[r.Owner] = name
			}
		}
		if (r.Kind == "service" || r.Kind == "helmrelease") && len(r.Ports) > 0 {
			servicePorts[name] = strings.Join(r.Ports, ", ")
		}
	}

	out := []asciidocRuntimeAPIRow{}
	seen := map[string]bool{}
	for _, m := range mappings {
		if m.Type != "depends_on" {
			continue
		}
		from := fuToRuntime[m.From]
		to := fuToRuntime[m.To]
		if strings.TrimSpace(from) == "" || strings.TrimSpace(to) == "" {
			continue
		}
		ports := servicePorts[to]
		if strings.TrimSpace(ports) == "" {
			ports = "unknown"
		}
		key := from + "|" + to + "|" + ports
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, asciidocRuntimeAPIRow{
			Consumer: from,
			Provider: to,
			Ports:    ports,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Provider != out[j].Provider {
			return out[i].Provider < out[j].Provider
		}
		return out[i].Consumer < out[j].Consumer
	})
	return out
}

func buildRuntimeAPIMermaid(rows []asciidocRuntimeAPIRow) string {
	if len(rows) == 0 {
		return ""
	}
	lines := []string{"flowchart LR"}
	for _, r := range rows {
		cn := "RT_" + sanitizeNode(r.Consumer)
		pn := "RT_" + sanitizeNode(r.Provider)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::runtime_element", cn, escapeMermaidLabel(r.Consumer)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::runtime_element", pn, escapeMermaidLabel(r.Provider)))
		lines = append(lines, fmt.Sprintf("  %s -->|API %s| %s", cn, escapeMermaidLabel(r.Ports), pn))
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

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

func buildDeploymentRows(runtime []inferredRuntimeItem) []asciidocDeploymentRow {
	var sourceName, kustomName string
	releases := []string{}
	workloads := []string{}
	namespaces := []string{}
	clusters := []string{}
	out := []asciidocDeploymentRow{}
	seen := map[string]bool{}
	addRow := func(from, to, how string) {
		from = strings.TrimSpace(from)
		to = strings.TrimSpace(to)
		how = strings.TrimSpace(how)
		if from == "" || to == "" || how == "" {
			return
		}
		key := from + "|" + to + "|" + how
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, asciidocDeploymentRow{From: from, To: to, How: how})
	}

	for _, r := range runtime {
		n := deploymentNodeName(r)
		switch r.Kind {
		case "gitrepository":
			if sourceName == "" {
				sourceName = n
			}
		case "kustomization":
			if kustomName == "" {
				kustomName = n
			}
		case "helmrelease":
			releases = append(releases, n)
		case "deployment", "workload", "lambda_function", "service", "queue", "topic", "eventbridge_target":
			workloads = append(workloads, n)
		case "eventbridge_rule", "terraform_resource":
			releases = append(releases, n)
		case "namespace":
			namespaces = append(namespaces, n)
		case "cluster":
			clusters = append(clusters, n)
		}
	}
	releases = uniqueSorted(releases)
	workloads = uniqueSorted(workloads)
	namespaces = uniqueSorted(namespaces)
	clusters = uniqueSorted(clusters)

	if sourceName != "" && kustomName != "" {
		addRow(sourceName, kustomName, "reconciles")
	}
	for _, r := range releases {
		if kustomName != "" {
			addRow(kustomName, r, "applies")
		}
	}
	for _, r := range releases {
		for _, w := range workloads {
			if strings.Contains(strings.ToLower(w), strings.ToLower(r)) {
				addRow(r, w, "deploys")
			}
		}
	}
	for _, r := range releases {
		for _, ns := range namespaces {
			if strings.Contains(strings.ToLower(r), strings.ToLower(ns)) {
				addRow(r, ns, "targets")
			}
		}
	}
	if len(clusters) > 0 {
		clusterName := clusters[0]
		for _, ns := range namespaces {
			addRow(ns, clusterName, "part_of")
		}
	}

	// Fallback for runtime sets that do not naturally match the release/namespace
	// heuristics above (for example Terraform Lambda/EventBridge-only samples).
	if len(out) == 0 {
		for _, r := range runtime {
			target := deploymentNodeName(r)
			if target == "" {
				continue
			}
			source := runtimeShortName(strings.ReplaceAll(strings.TrimSpace(r.Source), "\\", "/"))
			if source == "" {
				source = "inferred-runtime-source"
			}
			addRow(source, target, "deploys")
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		if out[i].To != out[j].To {
			return out[i].To < out[j].To
		}
		return out[i].How < out[j].How
	})
	return out
}

func buildDeploymentMermaid(rows []asciidocDeploymentRow) string {
	lines := []string{"flowchart TB"}

	type edge struct {
		From string
		To   string
		How  string
	}
	edges := make([]edge, 0, len(rows))

	nodeClass := map[string]string{}
	classRank := map[string]int{
		"deployment_element": 1,
		"runtime_element":    2,
	}
	sourceSet := map[string]bool{}
	kustomSet := map[string]bool{}
	releaseSet := map[string]bool{}
	workloadSet := map[string]bool{}
	nsSet := map[string]bool{}
	clusterSet := map[string]bool{}
	releaseTargets := map[string]map[string]bool{}
	releaseDeploys := map[string]map[string]bool{}
	nsCluster := map[string]string{}

	setClass := func(node, class string) {
		if existing, ok := nodeClass[node]; ok {
			if classRank[class] <= classRank[existing] {
				return
			}
		}
		nodeClass[node] = class
	}
	for _, r := range rows {
		if strings.TrimSpace(r.From) == "" || strings.TrimSpace(r.To) == "" || strings.TrimSpace(r.How) == "" {
			continue
		}
		edges = append(edges, edge{From: r.From, To: r.To, How: r.How})
		setClass(r.From, "deployment_element")
		setClass(r.To, "deployment_element")
		switch r.How {
		case "reconciles":
			sourceSet[r.From] = true
			kustomSet[r.To] = true
		case "applies":
			kustomSet[r.From] = true
			releaseSet[r.To] = true
		case "deploys":
			releaseSet[r.From] = true
			workloadSet[r.To] = true
			setClass(r.To, "runtime_element")
			if releaseDeploys[r.From] == nil {
				releaseDeploys[r.From] = map[string]bool{}
			}
			releaseDeploys[r.From][r.To] = true
		case "targets":
			releaseSet[r.From] = true
			nsSet[r.To] = true
			if releaseTargets[r.From] == nil {
				releaseTargets[r.From] = map[string]bool{}
			}
			releaseTargets[r.From][r.To] = true
		case "part_of":
			nsSet[r.From] = true
			clusterSet[r.To] = true
			if strings.TrimSpace(nsCluster[r.From]) == "" {
				nsCluster[r.From] = r.To
			}
		}
	}

	releasePrimaryNS := map[string]string{}
	for release, targets := range releaseTargets {
		names := make([]string, 0, len(targets))
		for ns := range targets {
			names = append(names, ns)
		}
		sort.Strings(names)
		if len(names) > 0 {
			releasePrimaryNS[release] = names[0]
		}
	}
	nsReleases := map[string][]string{}
	for release, ns := range releasePrimaryNS {
		nsReleases[ns] = append(nsReleases[ns], release)
	}
	for ns := range nsReleases {
		sort.Strings(nsReleases[ns])
	}

	workloadNamespaces := map[string]map[string]bool{}
	for release, targets := range releaseTargets {
		for ns := range targets {
			for workload := range releaseDeploys[release] {
				if workloadNamespaces[workload] == nil {
					workloadNamespaces[workload] = map[string]bool{}
				}
				workloadNamespaces[workload][ns] = true
			}
		}
	}
	workloadPrimaryNS := map[string]string{}
	for workload, nsCandidates := range workloadNamespaces {
		names := make([]string, 0, len(nsCandidates))
		for ns := range nsCandidates {
			names = append(names, ns)
		}
		sort.Strings(names)
		if len(names) > 0 {
			workloadPrimaryNS[workload] = names[0]
		}
	}
	nsWorkloads := map[string][]string{}
	for workload, ns := range workloadPrimaryNS {
		nsWorkloads[ns] = append(nsWorkloads[ns], workload)
	}
	for ns := range nsWorkloads {
		sort.Strings(nsWorkloads[ns])
	}

	nodeRole := map[string]string{}
	for x := range sourceSet {
		nodeRole[x] = "source"
	}
	for x := range kustomSet {
		nodeRole[x] = "kustomization"
	}
	for x := range releaseSet {
		nodeRole[x] = "release"
	}
	for x := range workloadSet {
		nodeRole[x] = "workload"
	}
	for x := range nsSet {
		nodeRole[x] = "namespace"
	}
	for x := range clusterSet {
		if nsSet[x] {
			continue
		}
		nodeRole[x] = "cluster"
	}

	nodeID := func(name string) string {
		return "DP_" + sanitizeNode(name)
	}
	nodeLabel := func(name string) string {
		switch nodeRole[name] {
		case "source":
			return "Source: " + name
		case "kustomization":
			return "Kustomization: " + name
		case "release":
			return "Release: " + name
		case "workload":
			return "Workload: " + name
		case "namespace":
			return "ns/" + runtimeShortName(name)
		case "cluster":
			return "cluster/" + runtimeShortName(name)
		default:
			return name
		}
	}
	nodeDecl := func(name string) string {
		class := nodeClass[name]
		if strings.TrimSpace(class) == "" {
			class = "deployment_element"
		}
		return fmt.Sprintf("%s[\"%s\"]:::%s", nodeID(name), escapeMermaidLabel(nodeLabel(name)), class)
	}
	sortedNodeNames := func(set map[string]bool) []string {
		out := make([]string, 0, len(set))
		for k := range set {
			out = append(out, k)
		}
		sort.Strings(out)
		return out
	}
	emitted := map[string]bool{}
	emitStandalone := func(name string) {
		if emitted[name] {
			return
		}
		lines = append(lines, "  "+nodeDecl(name))
		emitted[name] = true
	}

	namespaces := sortedNodeNames(nsSet)
	if len(namespaces) > 0 {
		clusteredNamespaces := map[string][]string{}
		unclusteredNamespaces := []string{}
		for _, ns := range namespaces {
			cluster := strings.TrimSpace(nsCluster[ns])
			if cluster == "" {
				unclusteredNamespaces = append(unclusteredNamespaces, ns)
				continue
			}
			clusteredNamespaces[cluster] = append(clusteredNamespaces[cluster], ns)
		}
		for cluster := range clusteredNamespaces {
			sort.Strings(clusteredNamespaces[cluster])
		}
		sort.Strings(unclusteredNamespaces)

		emitNamespaceSubgraph := func(indent, ns string) {
			subgraphID := "NS_" + sanitizeNode(ns)
			lines = append(lines, fmt.Sprintf("%ssubgraph %s[\"Namespace: %s\"]", indent, subgraphID, escapeMermaidLabel(ns)))
			lines = append(lines, indent+"  direction TB")
			lines = append(lines, indent+"  "+nodeDecl(ns))
			emitted[ns] = true
			for _, release := range nsReleases[ns] {
				lines = append(lines, indent+"  "+nodeDecl(release))
				emitted[release] = true
			}
			for _, workload := range nsWorkloads[ns] {
				lines = append(lines, indent+"  "+nodeDecl(workload))
				emitted[workload] = true
			}
			lines = append(lines, indent+"end")
		}

		clusterNames := sortedNodeNames(clusterSet)
		for _, cluster := range clusterNames {
			namesInCluster := clusteredNamespaces[cluster]
			if len(namesInCluster) == 0 {
				continue
			}
			clusterID := "CLUSTER_" + sanitizeNode(cluster)
			lines = append(lines, fmt.Sprintf("  subgraph %s[\"Cluster: %s\"]", clusterID, escapeMermaidLabel(cluster)))
			lines = append(lines, "    direction TB")
			for _, ns := range namesInCluster {
				emitNamespaceSubgraph("    ", ns)
			}
			lines = append(lines, "  end")
		}
		for _, ns := range unclusteredNamespaces {
			emitNamespaceSubgraph("  ", ns)
		}
	}

	controlPlaneNodes := map[string]bool{}
	for n := range sourceSet {
		controlPlaneNodes[n] = true
	}
	for n := range kustomSet {
		controlPlaneNodes[n] = true
	}
	for n := range releaseSet {
		if strings.TrimSpace(releasePrimaryNS[n]) == "" {
			controlPlaneNodes[n] = true
		}
	}
	if len(controlPlaneNodes) > 0 {
		lines = append(lines, `  subgraph CONTROL_PLANE["Control Plane"]`)
		lines = append(lines, "    direction TB")
		for _, n := range sortedNodeNames(controlPlaneNodes) {
			if emitted[n] {
				continue
			}
			lines = append(lines, "    "+nodeDecl(n))
			emitted[n] = true
		}
		lines = append(lines, "  end")
	}

	allNodes := map[string]bool{}
	for n := range nodeClass {
		allNodes[n] = true
	}
	for _, n := range sortedNodeNames(allNodes) {
		if clusterSet[n] {
			continue
		}
		emitStandalone(n)
	}
	for _, cluster := range sortedNodeNames(clusterSet) {
		hasNamespace := false
		for _, c := range nsCluster {
			if strings.TrimSpace(c) == strings.TrimSpace(cluster) {
				hasNamespace = true
				break
			}
		}
		if !hasNamespace {
			emitStandalone(cluster)
		}
	}

	for _, r := range edges {
		if r.How == "part_of" {
			continue
		}
		lines = append(lines, fmt.Sprintf("  %s -->|%s| %s", nodeID(r.From), escapeMermaidLabel(r.How), nodeID(r.To)))
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(lines, "\n")
}

func deploymentNodeName(r inferredRuntimeItem) string {
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

func buildPlatformOpsRows(a model.AuthoredArchitecture, runtime []inferredRuntimeItem) []asciidocPlatformOpRow {
	platformUnits := map[string]string{}
	for _, u := range a.FunctionalUnits {
		groupID := strings.TrimSpace(strings.ToUpper(u.Group))
		if groupID == "FG-PLATFORM" || groupID == "FG-PLATFORM-OPERATIONS" {
			platformUnits[u.ID] = nonEmpty(u.Name, u.ID)
		}
	}

	unitTargets := map[string][]string{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		if owner != "" && owner != "unresolved" {
			unitTargets[owner] = append(unitTargets[owner], runtimeShortName(r.Name))
		}
	}
	// Convention fallback for platform provisioning artifacts.
	for _, r := range runtime {
		n := runtimeShortName(r.Name)
		switch r.Kind {
		case "cluster", "namespace":
			unitTargets["FU-CLUSTER-PROVISIONING"] = append(unitTargets["FU-CLUSTER-PROVISIONING"], n)
		case "gitrepository", "kustomization", "helmrelease":
			unitTargets["FU-GITOPS-OPERATIONS"] = append(unitTargets["FU-GITOPS-OPERATIONS"], n)
		}
	}
	for k, v := range unitTargets {
		unitTargets[k] = uniqueSorted(v)
	}

	out := []asciidocPlatformOpRow{}
	for _, m := range a.Mappings {
		if m.Type != "interacts_with" {
			continue
		}
		unitName, ok := platformUnits[m.To]
		if !ok {
			continue
		}
		actorName := m.From
		for _, x := range a.Actors {
			if x.ID == m.From {
				actorName = nonEmpty(x.Name, x.ID)
				break
			}
		}
		targets := unitTargets[m.To]
		if len(targets) == 0 {
			out = append(out, asciidocPlatformOpRow{Actor: actorName, Unit: unitName, Target: "platform control operations"})
			continue
		}
		for _, t := range targets {
			out = append(out, asciidocPlatformOpRow{Actor: actorName, Unit: unitName, Target: t})
		}
	}
	return out
}

func buildPlatformOpsMermaid(rows []asciidocPlatformOpRow) string {
	lines := []string{"flowchart LR"}
	for _, r := range rows {
		an := "ACT_" + sanitizeNode(r.Actor)
		un := "PFU_" + sanitizeNode(r.Unit)
		tn := "TGT_" + sanitizeNode(r.Target)
		targetClass := "deployment_element"
		lowerTarget := strings.ToLower(r.Target)
		if strings.Contains(lowerTarget, "deployment") || strings.Contains(lowerTarget, "workload") || strings.Contains(lowerTarget, "pod") || strings.Contains(lowerTarget, "service") {
			targetClass = "runtime_element"
		}
		lines = append(lines, fmt.Sprintf("  %s((\"%s\")):::actor", an, escapeMermaidLabel(r.Actor)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_unit", un, escapeMermaidLabel(r.Unit)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::%s", tn, escapeMermaidLabel(r.Target), targetClass))
		lines = append(lines, fmt.Sprintf("  %s --> %s", an, un))
		lines = append(lines, fmt.Sprintf("  %s --> %s", un, tn))
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

func buildSecurityPathRows(a model.AuthoredArchitecture, labels map[string]string) []asciidocSecurityPathRow {
	depsByTarget := map[string][]string{}
	for _, m := range a.Mappings {
		if m.Type != "depends_on" {
			continue
		}
		depsByTarget[m.From] = append(depsByTarget[m.From], nonEmpty(labels[m.To], m.To))
	}
	for k, v := range depsByTarget {
		depsByTarget[k] = uniqueSorted(v)
	}

	out := []asciidocSecurityPathRow{}
	seen := map[string]bool{}
	for _, m := range a.Mappings {
		if m.Type != "targets" {
			continue
		}
		attack := nonEmpty(labels[m.From], m.From)
		target := nonEmpty(labels[m.To], m.To)
		deps := depsByTarget[m.To]
		depSummary := "none"
		if len(deps) > 0 {
			depSummary = strings.Join(deps, ", ")
		}
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
		if out[i].AttackVector != out[j].AttackVector {
			return out[i].AttackVector < out[j].AttackVector
		}
		return out[i].Target < out[j].Target
	})
	return out
}

func buildSecurityPathMermaid(rows []asciidocSecurityPathRow, runtime []inferredRuntimeItem, code []inferredCodeItem) string {
	runtimeByOwner := map[string][]string{}
	for _, r := range runtime {
		owner := strings.TrimSpace(r.Owner)
		if owner == "" || owner == "unresolved" {
			continue
		}
		label := runtimeShortName(strings.TrimSpace(r.Name))
		if label == "" {
			continue
		}
		runtimeByOwner[owner] = append(runtimeByOwner[owner], label)
	}
	for owner, items := range runtimeByOwner {
		runtimeByOwner[owner] = uniqueSorted(items)
	}

	codeByOwner := map[string][]string{}
	for _, c := range code {
		owner := strings.TrimSpace(c.Owner)
		if owner == "" || owner == "unresolved" {
			continue
		}
		label := ""
		switch strings.TrimSpace(c.Kind) {
		case "source_file", "symbol":
			path := codeItemPath(c)
			if path == "" {
				continue
			}
			label = moduleFromPath(path)
			if strings.TrimSpace(label) == "" {
				label = path
			}
		default:
			label = strings.TrimSpace(c.Element)
		}
		if strings.TrimSpace(label) == "" {
			continue
		}
		codeByOwner[owner] = append(codeByOwner[owner], label)
	}
	for owner, items := range codeByOwner {
		codeByOwner[owner] = uniqueSorted(items)
	}

	lines := []string{"flowchart LR"}
	for _, r := range rows {
		attackID := strings.TrimSpace(r.AttackVectorID)
		if attackID == "" {
			attackID = r.AttackVector
		}
		targetID := strings.TrimSpace(r.TargetID)
		if targetID == "" {
			targetID = r.Target
		}
		avNode := "AV_" + sanitizeNode(attackID)
		tNode := "SEC_TGT_" + sanitizeNode(targetID)
		lines = append(lines, fmt.Sprintf("  %s((\"%s\")):::attack_vector", avNode, escapeMermaidLabel(r.AttackVector)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_unit", tNode, escapeMermaidLabel(r.Target)))
		lines = append(lines, fmt.Sprintf("  %s -->|targets| %s", avNode, tNode))
		for _, dep := range strings.Split(r.DependsOn, ",") {
			dep = strings.TrimSpace(dep)
			if dep == "" || dep == "none" {
				continue
			}
			dNode := "SEC_DEP_" + sanitizeNode(dep)
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::referenced_element", dNode, escapeMermaidLabel(dep)))
			lines = append(lines, fmt.Sprintf("  %s -->|depends_on| %s", tNode, dNode))
		}
		runtimeNodes := make([]string, 0, len(runtimeByOwner[targetID]))
		for _, rt := range runtimeByOwner[targetID] {
			rtNode := "SEC_RT_" + sanitizeNode(targetID+"-"+rt)
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::runtime_element", rtNode, escapeMermaidLabel(rt)))
			lines = append(lines, fmt.Sprintf("  %s -->|runtime| %s", tNode, rtNode))
			runtimeNodes = append(runtimeNodes, rtNode)
		}
		// Keep code evidence traced via runtime, even when no direct runtime artifact is inferred.
		if len(runtimeNodes) == 0 && len(codeByOwner[targetID]) > 0 {
			rtLabel := nonEmpty(strings.TrimSpace(r.Target), targetID) + " runtime (inferred)"
			rtNode := "SEC_RT_" + sanitizeNode(targetID+"-runtime")
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::runtime_element", rtNode, escapeMermaidLabel(rtLabel)))
			lines = append(lines, fmt.Sprintf("  %s -->|runtime| %s", tNode, rtNode))
			runtimeNodes = append(runtimeNodes, rtNode)
		}
		for _, mod := range codeByOwner[targetID] {
			codeNode := "SEC_CODE_" + sanitizeNode(targetID+"-"+mod)
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::code_element", codeNode, escapeMermaidLabel(mod)))
			if len(runtimeNodes) > 0 {
				lines = append(lines, fmt.Sprintf("  %s -->|implemented_by| %s", runtimeNodes[0], codeNode))
			}
		}
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

func buildSecurityObservabilityRows(runtime []inferredRuntimeItem, code []inferredCodeItem) []asciidocSecurityObsRow {
	out := []asciidocSecurityObsRow{}
	seen := map[string]bool{}

	add := func(signal, layer, owner, evidence string) {
		signal = strings.TrimSpace(signal)
		layer = strings.TrimSpace(layer)
		owner = strings.TrimSpace(owner)
		evidence = strings.TrimSpace(evidence)
		if signal == "" || layer == "" || evidence == "" {
			return
		}
		if owner == "" {
			owner = "unresolved"
		}
		key := signal + "|" + layer + "|" + owner + "|" + evidence
		if seen[key] {
			return
		}
		seen[key] = true
		out = append(out, asciidocSecurityObsRow{
			Signal:   signal,
			Layer:    layer,
			Owner:    owner,
			Evidence: evidence,
		})
	}

	for _, r := range runtime {
		name := strings.ToLower(runtimeShortName(r.Name))
		owner := r.Owner
		switch r.Kind {
		case "ingress":
			add("ingress access and suspicious request logs", "runtime", owner, r.Kind+" "+runtimeShortName(r.Name))
		case "service", "deployment", "workload", "pod":
			add("runtime request, error, and dependency logs", "runtime", owner, r.Kind+" "+runtimeShortName(r.Name))
		case "helmrelease", "kustomization", "gitrepository", "cluster", "namespace", "terraform_resource":
			add("deployment and platform audit events", "deployment", owner, r.Kind+" "+runtimeShortName(r.Name))
		}
		if strings.Contains(name, "auth") || strings.Contains(name, "token") {
			add("authentication and token misuse events", "runtime", owner, r.Kind+" "+runtimeShortName(r.Name))
		}
		if strings.Contains(name, "risk") || strings.Contains(name, "fraud") {
			add("abuse and fraud decision logs", "runtime", owner, r.Kind+" "+runtimeShortName(r.Name))
		}
	}

	for _, c := range code {
		path := strings.ToLower(codeItemPath(c))
		owner := c.Owner
		if strings.Contains(path, "log") || strings.Contains(path, "audit") || strings.Contains(path, "trace") {
			add("application security telemetry hooks", "code", owner, "code "+codeItemPath(c))
		}
		if strings.Contains(path, "auth") || strings.Contains(path, "token") {
			add("authorization and credential handling checks", "code", owner, "code "+codeItemPath(c))
		}
		if strings.Contains(path, "risk") || strings.Contains(path, "fraud") {
			add("fraud and abuse detection code paths", "code", owner, "code "+codeItemPath(c))
		}
	}

	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Layer != out[j].Layer {
			return out[i].Layer < out[j].Layer
		}
		if out[i].Signal != out[j].Signal {
			return out[i].Signal < out[j].Signal
		}
		if out[i].Owner != out[j].Owner {
			return out[i].Owner < out[j].Owner
		}
		return out[i].Evidence < out[j].Evidence
	})
	if len(out) > 28 {
		return out[:28]
	}
	return out
}
