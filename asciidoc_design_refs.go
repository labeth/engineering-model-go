// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func mapDesignGroups(d model.DesignDocument) map[string]model.DesignFunctionalGroup {
	out := map[string]model.DesignFunctionalGroup{}
	for _, x := range d.Design.FunctionalGroups {
		out[x.ID] = x
	}
	return out
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func mapDesignUnits(d model.DesignDocument) map[string]model.DesignFunctionalUnit {
	out := map[string]model.DesignFunctionalUnit{}
	for _, x := range d.Design.FunctionalUnits {
		out[x.ID] = x
	}
	return out
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
func nonEmpty(v, fallback string) string {
	if strings.TrimSpace(v) == "" {
		return fallback
	}
	return v
}

// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func referenceAnchor(kind, id string) string {
	return "REF_" + strings.ToUpper(strings.TrimSpace(kind)) + "_" + sanitizeNode(id)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
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

// TRLC-LINKS: REQ-EMG-003
func aliasDescription(e model.CatalogEntry) string {
	name := nonEmpty(strings.TrimSpace(e.Name), strings.TrimSpace(e.ID))
	def := strings.TrimSpace(e.Definition)
	if def == "" {
		return "Same meaning as " + name + "."
	}
	return def
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func buildCodeReferences(in []inferredCodeItem) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	seen := map[string]bool{}
	for _, c := range in {
		id := codeItemDisplayName(c)
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
			Anchor:      referenceAnchor("code", c.Kind+"-"+id+"-"+source),
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func builtInEngineeringModelTerms() []asciidocTerm {
	return []asciidocTerm{
		{
			Anchor:      referenceAnchor("engmodel", "TERM-FUNCTIONAL-GROUP"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-FUNCTIONAL-GROUP"),
			ID:          "TERM-FUNCTIONAL-GROUP",
			Name:        "functional group",
			Definition:  "A major authored capability area that groups related functional units.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-FUNCTIONAL-UNIT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-FUNCTIONAL-UNIT"),
			ID:          "TERM-FUNCTIONAL-UNIT",
			Name:        "functional unit",
			Definition:  "An authored working unit inside a functional group that owns specific behavior.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-RUNTIME-ELEMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-RUNTIME-ELEMENT"),
			ID:          "TERM-RUNTIME-ELEMENT",
			Name:        "runtime element",
			Definition:  "An inferred runtime realization element discovered from infrastructure and deployment sources.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-CODE-ELEMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-CODE-ELEMENT"),
			ID:          "TERM-CODE-ELEMENT",
			Name:        "code element",
			Definition:  "An inferred code structure or ownership element discovered from source trees and build metadata.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-TRACE-MARKER"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-TRACE-MARKER"),
			ID:          "TERM-TRACE-MARKER",
			Name:        "trace marker",
			Definition:  "Source-level marker such as TRLC-LINKS or ENGMODEL-LINKS used to connect declarations to requirements and model entities.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-MODEL"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-MODEL"),
			ID:          "TERM-MODEL",
			Name:        "model",
			Definition:  "The authored architecture model root that composes functional structure, relationships, inference hints, and views.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-BUNDLE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-BUNDLE"),
			ID:          "TERM-BUNDLE",
			Name:        "model bundle",
			Definition:  "Loaded architecture, catalog, and companion documents resolved as one working model context.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-CATALOG"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-CATALOG"),
			ID:          "TERM-CATALOG",
			Name:        "catalog",
			Definition:  "Controlled vocabulary document used to normalize model, requirement, and generated-document terminology.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-CATALOG-ENTRY"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-CATALOG-ENTRY"),
			ID:          "TERM-CATALOG-ENTRY",
			Name:        "catalog entry",
			Definition:  "A stable catalog term with a definition and aliases for linting, linking, and generated references.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-LINT-RUN"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-LINT-RUN"),
			ID:          "TERM-LINT-RUN",
			Name:        "lint run",
			Definition:  "Requirement lint configuration that controls parsing and quality checks for requirement text.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-VALIDATION"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-VALIDATION"),
			ID:          "TERM-VALIDATION",
			Name:        "validation",
			Definition:  "Model quality gate that checks authored documents, references, relationship semantics, and requirement lint results.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-VALIDATION-DIAGNOSTIC"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-VALIDATION-DIAGNOSTIC"),
			ID:          "TERM-VALIDATION-DIAGNOSTIC",
			Name:        "validation diagnostic",
			Definition:  "Structured validation finding with code, severity, message, and optional source path.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-DESIGN"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-DESIGN"),
			ID:          "TERM-DESIGN",
			Name:        "design document",
			Definition:  "Authored design narrative organized by model entities and architecture views.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-DESIGN-VIEW"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-DESIGN-VIEW"),
			ID:          "TERM-DESIGN-VIEW",
			Name:        "design view",
			Definition:  "View-scoped design narrative attached to an authored functional group or functional unit.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-DECISION"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-DECISION"),
			ID:          "TERM-DECISION",
			Name:        "architecture decision",
			Definition:  "Authored architecture decision record with status, context, decision text, and consequences.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-INFERENCE-HINT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-INFERENCE-HINT"),
			ID:          "TERM-INFERENCE-HINT",
			Name:        "inference hint",
			Definition:  "Authored source and ownership configuration used to discover runtime, code, and verification evidence.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-VIEW"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-VIEW"),
			ID:          "TERM-VIEW",
			Name:        "view",
			Definition:  "Authored projection configuration that selects roots, entity kinds, mappings, audience, and abstraction.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-ASCIIDOC-DOCUMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-ASCIIDOC-DOCUMENT"),
			ID:          "TERM-ASCIIDOC-DOCUMENT",
			Name:        "AsciiDoc document",
			Definition:  "Human-readable generated architecture publication document assembled from authored model, inferred evidence, diagrams, references, and decisions.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-ASCIIDOC-SECTION"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-ASCIIDOC-SECTION"),
			ID:          "TERM-ASCIIDOC-SECTION",
			Name:        "AsciiDoc section",
			Definition:  "Structured generated document section or row model used to render authored and inferred architecture content.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-ASCIIDOC-DIAGRAM"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-ASCIIDOC-DIAGRAM"),
			ID:          "TERM-ASCIIDOC-DIAGRAM",
			Name:        "AsciiDoc diagram",
			Definition:  "Generated diagram block, usually Mermaid, that visualizes architecture relationships in the AsciiDoc publication.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-REFERENCE-INDEX"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-REFERENCE-INDEX"),
			ID:          "TERM-REFERENCE-INDEX",
			Name:        "reference index",
			Definition:  "Generated index of authored, catalog, runtime, code, and verification references with stable anchors and backlinks.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-SECURITY-CONTEXT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-SECURITY-CONTEXT"),
			ID:          "TERM-SECURITY-CONTEXT",
			Name:        "security context",
			Definition:  "Security-focused generated context view that groups owned functional units and shows external actors, references, flows, controls, and trust boundaries.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-THREAT-MODEL-EXPORT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-THREAT-MODEL-EXPORT"),
			ID:          "TERM-THREAT-MODEL-EXPORT",
			Name:        "threat model export",
			Definition:  "Generated security artifact that translates authored architecture, flows, trust boundaries, threats, and mitigations into an external threat-model schema.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-THREAT-DRAGON-DOCUMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-THREAT-DRAGON-DOCUMENT"),
			ID:          "TERM-THREAT-DRAGON-DOCUMENT",
			Name:        "Threat Dragon document",
			Definition:  "Threat Dragon JSON representation generated from the engineering model for STRIDE threat-model review.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-OPEN-OTM-DOCUMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-OPEN-OTM-DOCUMENT"),
			ID:          "TERM-OPEN-OTM-DOCUMENT",
			Name:        "Open OTM document",
			Definition:  "Open Threat Model JSON representation generated from the engineering model for interoperable threat-model exchange.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-STRUCTURIZR-DSL"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-STRUCTURIZR-DSL"),
			ID:          "TERM-STRUCTURIZR-DSL",
			Name:        "Structurizr DSL",
			Definition:  "Generated Structurizr DSL workspace that projects authored architecture, relationships, dynamic views, and deployment metadata.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-TRLC-PACKAGE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-TRLC-PACKAGE"),
			ID:          "TERM-TRLC-PACKAGE",
			Name:        "TRLC package",
			Definition:  "Generated TRLC model and requirements package used for formal requirement traceability processing.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-LOBSTER-ACTIVITY-TRACE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-LOBSTER-ACTIVITY-TRACE"),
			ID:          "TERM-LOBSTER-ACTIVITY-TRACE",
			Name:        "LOBSTER activity trace",
			Definition:  "Generated LOBSTER activity JSON that links verification evidence back to formal requirement identifiers.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-OSCAL-SSP"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-OSCAL-SSP"),
			ID:          "TERM-OSCAL-SSP",
			Name:        "OSCAL SSP",
			Definition:  "Generated OSCAL system security plan that maps authored system metadata, components, controls, allocations, and evidence into SSP JSON.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-OSCAL-ASSESSMENT-RESULTS"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-OSCAL-ASSESSMENT-RESULTS"),
			ID:          "TERM-OSCAL-ASSESSMENT-RESULTS",
			Name:        "OSCAL assessment results",
			Definition:  "Generated OSCAL assessment results that summarize reviewed controls, verification findings, and modeled risks.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-OSCAL-POAM"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-OSCAL-POAM"),
			ID:          "TERM-OSCAL-POAM",
			Name:        "OSCAL POA&M",
			Definition:  "Generated OSCAL plan of action and milestones that maps modeled POA&M items and related risks into compliance JSON.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-MCP-SERVER"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-MCP-SERVER"),
			ID:          "TERM-MCP-SERVER",
			Name:        "MCP server",
			Definition:  "Model Context Protocol server that exposes engineering-model operations to AI agents through JSON-RPC tools.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-MCP-TOOL"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-MCP-TOOL"),
			ID:          "TERM-MCP-TOOL",
			Name:        "MCP tool",
			Definition:  "Named MCP operation with a constrained input schema and structured response payload.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-MCP-TOOL-RESPONSE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-MCP-TOOL-RESPONSE"),
			ID:          "TERM-MCP-TOOL-RESPONSE",
			Name:        "MCP tool response",
			Definition:  "Structured MCP tool payload that includes success state, schema version, generated timestamp, and result data or validation error.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-REPO-INDEX"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-REPO-INDEX"),
			ID:          "TERM-REPO-INDEX",
			Name:        "repository index",
			Definition:  "Bounded first-party source index used to answer model-aware file, requirement, control, and threat lookup queries.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-MCP-STDIO-TRANSPORT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-MCP-STDIO-TRANSPORT"),
			ID:          "TERM-MCP-STDIO-TRANSPORT",
			Name:        "MCP stdio transport",
			Definition:  "Content-Length framed standard input/output transport used by the MCP command-line server.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-CLI-COMMAND"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-CLI-COMMAND"),
			ID:          "TERM-CLI-COMMAND",
			Name:        "CLI command",
			Definition:  "Command-line entrypoint that coordinates loading, validation, generation, and file output for an engineering-model workflow.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-GENERATION-WORKFLOW"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-GENERATION-WORKFLOW"),
			ID:          "TERM-GENERATION-WORKFLOW",
			Name:        "generation workflow",
			Definition:  "Orchestrated run that combines model loading, validation, projection, rendering, and artifact emission.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-SOURCE-BLOCK"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-SOURCE-BLOCK"),
			ID:          "TERM-SOURCE-BLOCK",
			Name:        "source block",
			Definition:  "Stable source reference block that records file path, optional line span, kind, summary, and linked entity IDs.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-REFERENCED-ELEMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-REFERENCED-ELEMENT"),
			ID:          "TERM-REFERENCED-ELEMENT",
			Name:        "referenced element",
			Definition:  "An architecture-relevant external, platform, or third-party dependency represented by role.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-ACTOR"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-ACTOR"),
			ID:          "TERM-ACTOR",
			Name:        "actor",
			Definition:  "A person or operational role that interacts with functional units.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-INTERFACE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-INTERFACE"),
			ID:          "TERM-INTERFACE",
			Name:        "interface",
			Definition:  "An authored interface boundary (for example API, channel, endpoint, or contract) owned by a functional unit.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-DATA-OBJECT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-DATA-OBJECT"),
			ID:          "TERM-DATA-OBJECT",
			Name:        "data object",
			Definition:  "An authored data artifact or contract shape traced across flow, interface, and implementation boundaries.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-DEPLOYMENT-TARGET"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-DEPLOYMENT-TARGET"),
			ID:          "TERM-DEPLOYMENT-TARGET",
			Name:        "deployment target",
			Definition:  "An authored deployment destination such as environment, cluster, namespace, or registry scope.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-CONTROL"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-CONTROL"),
			ID:          "TERM-CONTROL",
			Name:        "control",
			Definition:  "An authored security, policy, or operational control used to constrain behavior and reduce risk.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-COMPLIANCE-MAPPING"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-COMPLIANCE-MAPPING"),
			ID:          "TERM-COMPLIANCE-MAPPING",
			Name:        "compliance mapping",
			Definition:  "Authored mapping that links a model control to selected compliance controls, model scope, implementation status, evidence, and responsible roles.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-RISK"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-RISK"),
			ID:          "TERM-RISK",
			Name:        "risk",
			Definition:  "Authored risk record with likelihood, impact, response, scope, controls, and evidence.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-POAM-ITEM"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-POAM-ITEM"),
			ID:          "TERM-POAM-ITEM",
			Name:        "POA&M item",
			Definition:  "Plan of action and milestones item linked to a modeled risk and supporting artifacts.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-EVIDENCE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-EVIDENCE"),
			ID:          "TERM-EVIDENCE",
			Name:        "evidence",
			Definition:  "A file, artifact, or observation used to support control, risk, threat, or verification claims.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-TRUST-BOUNDARY"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-TRUST-BOUNDARY"),
			ID:          "TERM-TRUST-BOUNDARY",
			Name:        "trust boundary",
			Definition:  "An authored trust separation zone that marks policy, identity, or security control boundaries.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-STATE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-STATE"),
			ID:          "TERM-STATE",
			Name:        "state",
			Definition:  "An authored lifecycle state used to model transition behavior, guards, and event-driven progression.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-EVENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-EVENT"),
			ID:          "TERM-EVENT",
			Name:        "event",
			Definition:  "An authored trigger signal that drives transitions, flow progress, or asynchronous outcomes.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-FLOW"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-FLOW"),
			ID:          "TERM-FLOW",
			Name:        "flow",
			Definition:  "An authored causal interaction sequence from user/system intent to outcome, represented as ordered steps.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-FLOW-STEP"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-FLOW-STEP"),
			ID:          "TERM-FLOW-STEP",
			Name:        "flow step",
			Definition:  "A single authored step in a flow that captures action, data in/out, references, and normal/error/async transitions.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-ATTACK-VECTOR"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-ATTACK-VECTOR"),
			ID:          "TERM-ATTACK-VECTOR",
			Name:        "attack vector",
			Definition:  "A technical misuse or attack path that targets functional, referenced, or runtime elements.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-THREAT-SCENARIO"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-THREAT-SCENARIO"),
			ID:          "TERM-THREAT-SCENARIO",
			Name:        "threat scenario",
			Definition:  "Authored threat narrative that connects attack vectors, scope, flows, controls, risks, and verification evidence.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-THREAT-ASSUMPTION"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-THREAT-ASSUMPTION"),
			ID:          "TERM-THREAT-ASSUMPTION",
			Name:        "threat assumption",
			Definition:  "Authored security assumption that records scope, rationale, owner, status, and evidence.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-THREAT-OUT-OF-SCOPE"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-THREAT-OUT-OF-SCOPE"),
			ID:          "TERM-THREAT-OUT-OF-SCOPE",
			Name:        "threat out-of-scope decision",
			Definition:  "Authored decision that excludes a threat concern from current scope with reason, owner, expiry, and evidence.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-THREAT-MITIGATION"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-THREAT-MITIGATION"),
			ID:          "TERM-THREAT-MITIGATION",
			Name:        "threat mitigation",
			Definition:  "Authored mitigation record that links a threat scenario to a control, effectiveness, verification, and evidence.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-AUTHORED-MAPPING"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-AUTHORED-MAPPING"),
			ID:          "TERM-AUTHORED-MAPPING",
			Name:        "authored mapping",
			Definition:  "An explicit relationship declared in architecture inputs between authored or referenced elements.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-INFERRED-MAPPING"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-INFERRED-MAPPING"),
			ID:          "TERM-INFERRED-MAPPING",
			Name:        "inferred mapping",
			Definition:  "A discovered relationship that links inferred runtime/code elements upward to authored design.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-UPWARD-LINKING"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-UPWARD-LINKING"),
			ID:          "TERM-UPWARD-LINKING",
			Name:        "upward linking",
			Definition:  "Rule where runtime and code elements point to stable functional groups/units; authored architecture does not depend on inferred IDs.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-REQUIREMENT"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-REQUIREMENT"),
			ID:          "TERM-REQUIREMENT",
			Name:        "requirement",
			Definition:  "A structured requirement statement that defines expected system behavior and maps to functional ownership.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-VERIFICATION-CHECK"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-VERIFICATION-CHECK"),
			ID:          "TERM-VERIFICATION-CHECK",
			Name:        "verification check",
			Definition:  "An inferred or authored verification artifact that validates requirement behavior with test evidence.",
		},
		{
			Anchor:      referenceAnchor("engmodel", "TERM-CONTROL-VERIFICATION"),
			IndexAnchor: referenceAnchor("idx-engmodel", "TERM-CONTROL-VERIFICATION"),
			ID:          "TERM-CONTROL-VERIFICATION",
			Name:        "control verification",
			Definition:  "Authored control verification record with method, status, threat/risk scope, findings, and evidence.",
		},
	}
}
