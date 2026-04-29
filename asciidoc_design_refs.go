// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

func mapDesignGroups(d model.DesignDocument) map[string]model.DesignFunctionalGroup {
	out := map[string]model.DesignFunctionalGroup{}
	for _, x := range d.Design.FunctionalGroups {
		out[x.ID] = x
	}
	return out
}

func mapDesignUnits(d model.DesignDocument) map[string]model.DesignFunctionalUnit {
	out := map[string]model.DesignFunctionalUnit{}
	for _, x := range d.Design.FunctionalUnits {
		out[x.ID] = x
	}
	return out
}

func viewDesignKey(kind string) string {
	switch kind {
	case "architecture-intent":
		return "architecture_intent"
	case "communication":
		return "communication"
	case "deployment":
		return "deployment"
	case "security":
		return "security"
	case "traceability":
		return "traceability"
	case "state-lifecycle":
		return "state_lifecycle"
	default:
		return kind
	}
}

func buildDesignDetails(entityID, intro string, source interface{}, views []model.View) []asciidocDesignDetail {
	out := make([]asciidocDesignDetail, 0, len(views))
	for _, v := range views {
		key := viewDesignKey(v.Kind)
		var dv model.DesignView
		var ok bool
		switch s := source.(type) {
		case model.DesignFunctionalGroup:
			dv, ok = s.Views[key]
		case model.DesignFunctionalUnit:
			dv, ok = s.Views[key]
		}
		title := strings.TrimSpace(v.Kind)
		narr := strings.TrimSpace(intro)
		if ok {
			if strings.TrimSpace(dv.Title) != "" {
				title = strings.TrimSpace(dv.Title)
			}
			if strings.TrimSpace(dv.Narrative) != "" {
				narr = strings.TrimSpace(dv.Narrative)
			}
		}
		out = append(out, asciidocDesignDetail{ViewID: v.ID, Title: title, Narrative: narr})
	}
	_ = entityID
	return out
}

func detailForView(details []asciidocDesignDetail, viewID string) asciidocDesignDetail {
	for _, d := range details {
		if d.ViewID == viewID {
			return d
		}
	}
	if len(details) > 0 {
		return details[0]
	}
	return asciidocDesignDetail{ViewID: viewID, Title: "Design", Narrative: ""}
}

func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

func buildLabelIndex(a model.AuthoredArchitecture) map[string]string {
	out := map[string]string{}
	for _, x := range a.FunctionalGroups {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.FunctionalUnits {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.Actors {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.AttackVectors {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.ReferencedElements {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.Interfaces {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.DataObjects {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.DeploymentTargets {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.Controls {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.TrustBoundaries {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.States {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, x := range a.Events {
		out[x.ID] = nonEmpty(x.Name, x.ID)
	}
	for _, f := range a.Flows {
		out[f.ID] = nonEmpty(f.Title, f.ID)
		for _, s := range f.Steps {
			sid := strings.TrimSpace(s.ID)
			if sid == "" {
				continue
			}
			composite := strings.TrimSpace(f.ID) + "::" + sid
			label := strings.TrimSpace(s.Action)
			if label == "" {
				label = sid
			}
			out[composite] = label
		}
	}
	return out
}

func referenceAnchor(kind, id string) string {
	return "REF_" + strings.ToUpper(strings.TrimSpace(kind)) + "_" + sanitizeNode(id)
}

func buildReferenceIndex(bundle model.Bundle, requirements model.RequirementsDocument, runtime []inferredRuntimeItem, code []inferredCodeItem, verification []inferredVerificationCheck) asciidocReferenceIndex {
	authored := []asciidocReferenceEntry{}
	catalogIDs := catalogEntryIDSet(bundle.Catalog)
	addAuthored := func(anchorKind, kind, id, name, desc string) {
		id = strings.TrimSpace(id)
		if id == "" {
			return
		}
		if catalogIDs[strings.ToUpper(id)] {
			return
		}
		authored = append(authored, asciidocReferenceEntry{
			Anchor:      referenceAnchor(anchorKind, id),
			ID:          id,
			Name:        strings.TrimSpace(name),
			Kind:        strings.TrimSpace(kind),
			Description: strings.TrimSpace(desc),
		})
	}

	for _, x := range bundle.Architecture.AuthoredArchitecture.FunctionalGroups {
		addAuthored("idx-fg", "Functional Group", x.ID, nonEmpty(x.Name, x.ID), x.Description)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.FunctionalUnits {
		addAuthored("idx-fu", "Functional Unit", x.ID, nonEmpty(x.Name, x.ID), x.Prose)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.Actors {
		addAuthored("idx-actor", "Actor", x.ID, nonEmpty(x.Name, x.ID), x.Description)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.AttackVectors {
		addAuthored("idx-attack", "Attack Vector", x.ID, nonEmpty(x.Name, x.ID), x.Description)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.ReferencedElements {
		addAuthored("idx-ref", "Referenced Element", x.ID, nonEmpty(x.Name, x.ID), x.Kind+" / "+x.Layer)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.Interfaces {
		desc := fmt.Sprintf("protocol=%s; endpoint=%s; schemaRef=%s; owner=%s", nonEmpty(strings.TrimSpace(x.Protocol), "n/a"), nonEmpty(strings.TrimSpace(x.Endpoint), "n/a"), nonEmpty(strings.TrimSpace(x.SchemaRef), "n/a"), nonEmpty(strings.TrimSpace(x.Owner), "n/a"))
		addAuthored("idx-if", "Interface", x.ID, nonEmpty(x.Name, x.ID), desc)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.DataObjects {
		desc := fmt.Sprintf("termRef=%s; schemaRef=%s; sensitivity=%s", nonEmpty(strings.TrimSpace(x.TermRef), "n/a"), nonEmpty(strings.TrimSpace(x.SchemaRef), "n/a"), nonEmpty(strings.TrimSpace(x.Sensitivity), "n/a"))
		addAuthored("idx-do", "Data Object", x.ID, nonEmpty(x.Name, x.ID), desc)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.DeploymentTargets {
		desc := fmt.Sprintf("environment=%s; region=%s; account=%s; cluster=%s; namespace=%s; trustZone=%s", nonEmpty(strings.TrimSpace(x.Environment), "n/a"), nonEmpty(strings.TrimSpace(x.Region), "n/a"), nonEmpty(strings.TrimSpace(x.Account), "n/a"), nonEmpty(strings.TrimSpace(x.Cluster), "n/a"), nonEmpty(strings.TrimSpace(x.Namespace), "n/a"), nonEmpty(strings.TrimSpace(x.TrustZone), "n/a"))
		addAuthored("idx-dep", "Deployment Target", x.ID, nonEmpty(x.Name, x.ID), desc)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.Controls {
		desc := fmt.Sprintf("category=%s; description=%s", nonEmpty(strings.TrimSpace(x.Category), "n/a"), nonEmpty(strings.TrimSpace(x.Description), "n/a"))
		addAuthored("idx-ctrl", "Control", x.ID, nonEmpty(x.Name, x.ID), desc)
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.TrustBoundaries {
		addAuthored("idx-tb", "Trust Boundary", x.ID, nonEmpty(x.Name, x.ID), nonEmpty(strings.TrimSpace(x.Description), "n/a"))
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.States {
		addAuthored("idx-state", "State", x.ID, nonEmpty(x.Name, x.ID), nonEmpty(strings.TrimSpace(x.Description), "n/a"))
	}
	for _, x := range bundle.Architecture.AuthoredArchitecture.Events {
		addAuthored("idx-evt", "Event", x.ID, nonEmpty(x.Name, x.ID), nonEmpty(strings.TrimSpace(x.Description), "n/a"))
	}
	for _, f := range bundle.Architecture.AuthoredArchitecture.Flows {
		addAuthored("idx-flow", "Flow", f.ID, nonEmpty(f.Title, f.ID), fmt.Sprintf("entry=%s; exits=%s; steps=%d", strings.Join(f.Entry, ", "), strings.Join(f.Exits, ", "), len(f.Steps)))
		for _, s := range f.Steps {
			sid := strings.TrimSpace(s.ID)
			if sid == "" {
				continue
			}
			id := strings.TrimSpace(f.ID) + "::" + sid
			desc := fmt.Sprintf("kind=%s; ref=%s; action=%s; dataIn=%s; dataOut=%s; async=%t", nonEmpty(strings.TrimSpace(s.Kind), "n/a"), nonEmpty(strings.TrimSpace(s.Ref), "n/a"), nonEmpty(strings.TrimSpace(s.Action), "n/a"), strings.Join(s.DataIn, ", "), strings.Join(s.DataOut, ", "), s.Async)
			addAuthored("idx-flow-step", "Flow Step", id, nonEmpty(strings.TrimSpace(s.Action), sid), desc)
		}
	}
	for _, x := range requirements.Requirements {
		authored = append(authored, asciidocReferenceEntry{
			Anchor:       referenceAnchor("idx-req", x.ID),
			TargetAnchor: referenceAnchor("req", x.ID),
			ID:           x.ID,
			Name:         x.ID,
			Kind:         "Requirement",
			Description:  strings.TrimSpace(x.Text),
		})
	}
	sort.SliceStable(authored, func(i, j int) bool {
		if authored[i].Kind != authored[j].Kind {
			return authored[i].Kind < authored[j].Kind
		}
		return authored[i].ID < authored[j].ID
	})

	catalog := buildCatalogReferences(bundle.Catalog)
	runtimeRefs := buildRuntimeReferences(runtime)
	codeRefs := buildCodeReferences(code)
	verificationRefs := buildVerificationReferences(verification)

	return asciidocReferenceIndex{
		Authored:     authored,
		Catalog:      catalog,
		Runtime:      runtimeRefs,
		Code:         codeRefs,
		Verification: verificationRefs,
	}
}

func buildCatalogReferences(doc model.CatalogDocument) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	seen := map[string]bool{}
	for _, term := range builtInEngineeringModelTerms() {
		id := strings.TrimSpace(term.ID)
		if id == "" {
			continue
		}
		key := strings.ToUpper(id)
		if seen[key] {
			continue
		}
		seen[key] = true
		out = append(out, asciidocReferenceEntry{
			Anchor:       referenceAnchor("idx-engmodel", term.ID),
			TargetAnchor: term.Anchor,
			ID:           id,
			Name:         term.Name,
			Kind:         "Engineering Model Term",
			Description:  strings.TrimSpace(term.Definition),
		})
	}
	add := func(kind string, entries []model.CatalogEntry) {
		for _, e := range entries {
			id := strings.TrimSpace(e.ID)
			if id == "" {
				continue
			}
			key := strings.ToUpper(id)
			if seen[key] {
				continue
			}
			seen[key] = true
			canonical := referenceAnchor("catalog", e.ID)
			out = append(out, asciidocReferenceEntry{
				Anchor:       referenceAnchor("idx-catalog", e.ID),
				TargetAnchor: canonical,
				ID:           id,
				Name:         nonEmpty(e.Name, e.ID),
				Kind:         "Catalog " + strings.TrimSpace(kind),
				Aliases:      uniqueSorted(e.Aliases),
				Description:  strings.TrimSpace(e.Definition),
			})
			for _, alias := range uniqueSorted(e.Aliases) {
				alias = strings.TrimSpace(alias)
				if alias == "" {
					continue
				}
				aliasKey := strings.ToUpper(alias)
				if seen[aliasKey] {
					continue
				}
				seen[aliasKey] = true
				out = append(out, asciidocReferenceEntry{
					Anchor:       referenceAnchor("idx-catalog", alias),
					TargetAnchor: canonical,
					ID:           alias,
					Name:         alias,
					Kind:         "Catalog Alias",
					Description:  aliasDescription(e),
				})
			}
		}
	}
	c := doc.Catalog
	add("System", c.Systems)
	add("Functional Group", c.FunctionalGroups)
	add("Functional Unit", c.FunctionalUnits)
	add("Referenced Element", c.ReferencedElements)
	add("Actor", c.Actors)
	add("Attack Vector", c.AttackVectors)
	add("Event", c.Events)
	add("State", c.States)
	add("Feature", c.Features)
	add("Mode", c.Modes)
	add("Condition", c.Conditions)
	add("Data Term", c.DataTerms)
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func catalogEntryIDSet(doc model.CatalogDocument) map[string]bool {
	out := map[string]bool{}
	add := func(entries []model.CatalogEntry) {
		for _, e := range entries {
			id := strings.ToUpper(strings.TrimSpace(e.ID))
			if id == "" {
				continue
			}
			out[id] = true
		}
	}
	c := doc.Catalog
	add(c.Systems)
	add(c.FunctionalGroups)
	add(c.FunctionalUnits)
	add(c.ReferencedElements)
	add(c.Actors)
	add(c.AttackVectors)
	add(c.Events)
	add(c.States)
	add(c.Features)
	add(c.Modes)
	add(c.Conditions)
	add(c.DataTerms)
	return out
}

func aliasDescription(e model.CatalogEntry) string {
	name := nonEmpty(strings.TrimSpace(e.Name), strings.TrimSpace(e.ID))
	def := strings.TrimSpace(e.Definition)
	if def == "" {
		return "Same meaning as " + name + "."
	}
	return def
}

func buildRuntimeReferences(in []inferredRuntimeItem) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	seen := map[string]bool{}
	for _, r := range in {
		id := strings.TrimSpace(r.Name)
		kind := strings.TrimSpace(r.Kind)
		key := kind + "|" + id + "|" + r.Source
		if id == "" || seen[key] {
			continue
		}
		seen[key] = true
		owner := nonEmpty(strings.TrimSpace(r.Owner), "unresolved")
		source := sanitizeSourcePath(r.Source)
		desc := strings.TrimSpace(r.Description)
		if desc == "" {
			desc = fmt.Sprintf("Inferred runtime %s owned by %s from %s.", nonEmpty(kind, "element"), owner, nonEmpty(source, "unknown source"))
		}
		out = append(out, asciidocReferenceEntry{
			Anchor:      referenceAnchor("rt", kind+"-"+id),
			ID:          id,
			Name:        id,
			Kind:        "Runtime " + kind,
			Owner:       owner,
			Description: desc,
			Source:      source,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > 80 {
		return out[:80]
	}
	return out
}

func buildCodeReferences(in []inferredCodeItem) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	seen := map[string]bool{}
	for _, c := range in {
		id := strings.TrimSpace(c.Element)
		key := c.Kind + "|" + id + "|" + c.Source
		if id == "" || seen[key] {
			continue
		}
		seen[key] = true
		owner := nonEmpty(strings.TrimSpace(c.Owner), "unresolved")
		source := sanitizeSourcePath(c.Source)
		desc := strings.TrimSpace(c.Description)
		if desc == "" {
			switch strings.TrimSpace(c.Kind) {
			case "symbol":
				desc = fmt.Sprintf("Inferred code symbol owned by %s from %s.", owner, nonEmpty(source, "unknown source"))
			case "source_file":
				desc = fmt.Sprintf("Inferred source file owned by %s from %s.", owner, nonEmpty(source, "unknown source"))
			default:
				desc = fmt.Sprintf("Inferred %s dependency owned by %s from %s.", nonEmpty(strings.TrimSpace(c.Kind), "code"), owner, nonEmpty(source, "unknown source"))
			}
		}
		out = append(out, asciidocReferenceEntry{
			Anchor:      referenceAnchor("code", c.Kind+"-"+id),
			ID:          id,
			Name:        id,
			Kind:        "Code " + c.Kind,
			Owner:       owner,
			Description: desc,
			Source:      source,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Kind != out[j].Kind {
			return out[i].Kind < out[j].Kind
		}
		return out[i].ID < out[j].ID
	})
	if len(out) > 120 {
		return out[:120]
	}
	return out
}

func buildVerificationReferences(in []inferredVerificationCheck) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	seen := map[string]bool{}
	for _, v := range in {
		id := strings.TrimSpace(v.ID)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		source := "n/a"
		if len(v.Evidence) > 0 {
			source = sanitizeSourcePath(strings.TrimSpace(v.Evidence[0]))
		}
		if len(v.Evidence) > 1 {
			source = fmt.Sprintf("%s (+%d)", source, len(v.Evidence)-1)
		}
		desc := strings.TrimSpace(v.Description)
		if desc == "" {
			desc = fmt.Sprintf("Inferred %s verification check with status %s.", nonEmpty(strings.TrimSpace(v.Kind), "test"), nonEmpty(strings.TrimSpace(v.Status), "not-run"))
		}
		out = append(out, asciidocReferenceEntry{
			Anchor:       referenceAnchor("idx-ver", id),
			TargetAnchor: referenceAnchor("verify", id),
			ID:           id,
			Name:         nonEmpty(strings.TrimSpace(v.Name), id),
			Kind:         nonEmpty(strings.TrimSpace(v.Kind), "test"),
			Status:       nonEmpty(strings.TrimSpace(v.Status), "not-run"),
			Description:  desc,
			Source:       source,
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return out[i].ID < out[j].ID
	})
	return out
}

func buildTermsFromCatalog(doc model.CatalogDocument) []asciidocTerm {
	out := []asciidocTerm{}
	out = append(out, builtInEngineeringModelTerms()...)
	add := func(entries []model.CatalogEntry) {
		for _, e := range entries {
			out = append(out, asciidocTerm{
				Anchor:      referenceAnchor("catalog", e.ID),
				IndexAnchor: referenceAnchor("idx-catalog", e.ID),
				ID:          strings.TrimSpace(e.ID),
				Name:        nonEmpty(strings.TrimSpace(e.Name), strings.TrimSpace(e.ID)),
				Definition:  strings.TrimSpace(e.Definition),
				Aliases:     uniqueSorted(e.Aliases),
			})
		}
	}
	c := doc.Catalog
	add(c.Systems)
	add(c.FunctionalGroups)
	add(c.FunctionalUnits)
	add(c.ReferencedElements)
	add(c.Actors)
	add(c.AttackVectors)
	add(c.Events)
	add(c.States)
	add(c.Features)
	add(c.Modes)
	add(c.Conditions)
	add(c.DataTerms)
	sort.SliceStable(out, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(out[i].Name))
		rightName := strings.ToLower(strings.TrimSpace(out[j].Name))
		if leftName != rightName {
			return leftName < rightName
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func builtInEngineeringModelTerms() []asciidocTerm {
	return []asciidocTerm{
		{
			Anchor:      referenceAnchor("engmodel", "EM-FUNCTIONAL-GROUP"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-FUNCTIONAL-GROUP"),
			ID:          "EM-FUNCTIONAL-GROUP",
			Name:        "functional group",
			Definition:  "A major authored capability area that groups related functional units.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-FUNCTIONAL-UNIT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-FUNCTIONAL-UNIT"),
			ID:          "EM-FUNCTIONAL-UNIT",
			Name:        "functional unit",
			Definition:  "An authored working unit inside a functional group that owns specific behavior.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-RUNTIME-ELEMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-RUNTIME-ELEMENT"),
			ID:          "EM-RUNTIME-ELEMENT",
			Name:        "runtime element",
			Definition:  "An inferred runtime realization element discovered from infrastructure and deployment sources.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-CODE-ELEMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-CODE-ELEMENT"),
			ID:          "EM-CODE-ELEMENT",
			Name:        "code element",
			Definition:  "An inferred code structure or ownership element discovered from source trees and build metadata.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-REFERENCED-ELEMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-REFERENCED-ELEMENT"),
			ID:          "EM-REFERENCED-ELEMENT",
			Name:        "referenced element",
			Definition:  "An architecture-relevant external, platform, or third-party dependency represented by role.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-ACTOR"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-ACTOR"),
			ID:          "EM-ACTOR",
			Name:        "actor",
			Definition:  "A person or operational role that interacts with functional units.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-INTERFACE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-INTERFACE"),
			ID:          "EM-INTERFACE",
			Name:        "interface",
			Definition:  "An authored interface boundary (for example API, channel, endpoint, or contract) owned by a functional unit.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-DATA-OBJECT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-DATA-OBJECT"),
			ID:          "EM-DATA-OBJECT",
			Name:        "data object",
			Definition:  "An authored data artifact or contract shape traced across flow, interface, and implementation boundaries.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-DEPLOYMENT-TARGET"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-DEPLOYMENT-TARGET"),
			ID:          "EM-DEPLOYMENT-TARGET",
			Name:        "deployment target",
			Definition:  "An authored deployment destination such as environment, cluster, namespace, or registry scope.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-CONTROL"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-CONTROL"),
			ID:          "EM-CONTROL",
			Name:        "control",
			Definition:  "An authored security, policy, or operational control used to constrain behavior and reduce risk.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-TRUST-BOUNDARY"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-TRUST-BOUNDARY"),
			ID:          "EM-TRUST-BOUNDARY",
			Name:        "trust boundary",
			Definition:  "An authored trust separation zone that marks policy, identity, or security control boundaries.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-STATE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-STATE"),
			ID:          "EM-STATE",
			Name:        "state",
			Definition:  "An authored lifecycle state used to model transition behavior, guards, and event-driven progression.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-EVENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-EVENT"),
			ID:          "EM-EVENT",
			Name:        "event",
			Definition:  "An authored trigger signal that drives transitions, flow progress, or asynchronous outcomes.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-FLOW"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-FLOW"),
			ID:          "EM-FLOW",
			Name:        "flow",
			Definition:  "An authored causal interaction sequence from user/system intent to outcome, represented as ordered steps.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-FLOW-STEP"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-FLOW-STEP"),
			ID:          "EM-FLOW-STEP",
			Name:        "flow step",
			Definition:  "A single authored step in a flow that captures action, data in/out, references, and normal/error/async transitions.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-ATTACK-VECTOR"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-ATTACK-VECTOR"),
			ID:          "EM-ATTACK-VECTOR",
			Name:        "attack vector",
			Definition:  "A technical misuse or attack path that targets functional, referenced, or runtime elements.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-AUTHORED-MAPPING"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-AUTHORED-MAPPING"),
			ID:          "EM-AUTHORED-MAPPING",
			Name:        "authored mapping",
			Definition:  "An explicit relationship declared in architecture inputs between authored or referenced elements.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-INFERRED-MAPPING"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-INFERRED-MAPPING"),
			ID:          "EM-INFERRED-MAPPING",
			Name:        "inferred mapping",
			Definition:  "A discovered relationship that links inferred runtime/code elements upward to authored design.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-UPWARD-LINKING"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-UPWARD-LINKING"),
			ID:          "EM-UPWARD-LINKING",
			Name:        "upward linking",
			Definition:  "Rule where runtime and code elements point to stable functional groups/units; authored architecture does not depend on inferred IDs.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-REQUIREMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-REQUIREMENT"),
			ID:          "EM-REQUIREMENT",
			Name:        "requirement",
			Definition:  "A structured requirement statement that defines expected system behavior and maps to functional ownership.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "EM-VERIFICATION-CHECK"),
			IndexAnchor: referenceAnchor("idx-engmodel", "EM-VERIFICATION-CHECK"),
			ID:          "EM-VERIFICATION-CHECK",
			Name:        "verification check",
			Definition:  "An inferred or authored verification artifact that validates requirement behavior with test evidence.",
		},
	}
}
