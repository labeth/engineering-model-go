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
	for _, r := range runtime {
		name := runtimeShortName(r.Name)
		if strings.TrimSpace(r.Owner) != "" && strings.TrimSpace(r.Owner) != "unresolved" && (r.Kind == "helmrelease" || r.Kind == "deployment" || r.Kind == "workload") {
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
	var sourceName, kustomName, clusterName string
	releases := []string{}
	workloads := []string{}
	namespaces := []string{}

	for _, r := range runtime {
		n := runtimeShortName(r.Name)
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
		case "deployment", "workload":
			workloads = append(workloads, n)
		case "namespace":
			namespaces = append(namespaces, n)
		case "cluster":
			if clusterName == "" {
				clusterName = n
			}
		}
	}
	releases = uniqueSorted(releases)
	workloads = uniqueSorted(workloads)
	namespaces = uniqueSorted(namespaces)

	out := []asciidocDeploymentRow{}
	if sourceName != "" && kustomName != "" {
		out = append(out, asciidocDeploymentRow{From: sourceName, To: kustomName, How: "reconciles"})
	}
	for _, r := range releases {
		if kustomName != "" {
			out = append(out, asciidocDeploymentRow{From: kustomName, To: r, How: "applies"})
		}
	}
	for _, r := range releases {
		for _, w := range workloads {
			if strings.Contains(strings.ToLower(w), strings.ToLower(r)) {
				out = append(out, asciidocDeploymentRow{From: r, To: w, How: "deploys"})
			}
		}
	}
	for _, r := range releases {
		for _, ns := range namespaces {
			if strings.Contains(strings.ToLower(r), strings.ToLower(ns)) {
				out = append(out, asciidocDeploymentRow{From: r, To: ns, How: "targets"})
			}
		}
	}
	if clusterName != "" {
		for _, ns := range namespaces {
			out = append(out, asciidocDeploymentRow{From: ns, To: clusterName, How: "part_of"})
		}
	}
	return out
}

func buildDeploymentMermaid(rows []asciidocDeploymentRow) string {
	lines := []string{"flowchart TB"}
	nodeClass := map[string]string{}
	classRank := map[string]int{
		"deployment_element": 1,
		"runtime_element":    2,
	}
	setClass := func(node, class string) {
		if existing, ok := nodeClass[node]; ok {
			if classRank[class] <= classRank[existing] {
				return
			}
		}
		nodeClass[node] = class
	}
	for _, r := range rows {
		setClass(r.From, "deployment_element")
		setClass(r.To, "deployment_element")
		if r.How == "deploys" {
			setClass(r.To, "runtime_element")
		}
	}
	for _, r := range rows {
		fn := "DP_" + sanitizeNode(r.From)
		tn := "DP_" + sanitizeNode(r.To)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::%s", fn, escapeMermaidLabel(r.From), nodeClass[r.From]))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::%s", tn, escapeMermaidLabel(r.To), nodeClass[r.To]))
		lines = append(lines, fmt.Sprintf("  %s -->|%s| %s", fn, escapeMermaidLabel(r.How), tn))
	}
	lines = appendMermaidClassDefs(lines)
	return strings.Join(uniquePreserve(lines), "\n")
}

func buildPlatformOpsRows(a model.AuthoredArchitecture, runtime []inferredRuntimeItem) []asciidocPlatformOpRow {
	platformUnits := map[string]string{}
	for _, u := range a.FunctionalUnits {
		if strings.TrimSpace(u.Group) == "FG-PLATFORM" {
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
			AttackVector: attack,
			Target:       target,
			DependsOn:    depSummary,
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

func buildSecurityPathMermaid(rows []asciidocSecurityPathRow) string {
	lines := []string{"flowchart LR"}
	for _, r := range rows {
		avNode := "AV_" + sanitizeNode(r.AttackVector)
		tNode := "SEC_TGT_" + sanitizeNode(r.Target)
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
