package engmodel

import (
	"fmt"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

type AsciiDocOptions struct {
	ViewIDs  []string
	CodeRoot string
}

type AsciiDocResult struct {
	Document    string
	Diagnostics []validate.Diagnostic
}

func GenerateAsciiDocFromFiles(architecturePath, requirementsPath, designPath string, options AsciiDocOptions) (AsciiDocResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return AsciiDocResult{}, err
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return AsciiDocResult{}, err
	}
	design, err := model.LoadDesign(designPath)
	if err != nil {
		return AsciiDocResult{}, err
	}
	if strings.TrimSpace(options.CodeRoot) != "" && !filepath.IsAbs(options.CodeRoot) {
		baseDir := filepath.Dir(architecturePath)
		options.CodeRoot = filepath.Join(baseDir, options.CodeRoot)
	}
	return GenerateAsciiDoc(bundle, requirements, design, options)
}

func GenerateAsciiDoc(bundle model.Bundle, requirements model.RequirementsDocument, design model.DesignDocument, options AsciiDocOptions) (AsciiDocResult, error) {
	diags := validate.Bundle(bundle)
	diags = append(diags, lintRequirementsEARS(requirements, bundle.Catalog)...)
	if validate.HasErrors(diags) {
		return AsciiDocResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}

	viewIDs := resolveViewIDs(bundle, options)
	viewSections := make([]asciidocViewSection, 0, len(viewIDs))
	viewNodeIDs := map[string]map[string]bool{}
	viewByID := map[string]model.View{}
	for _, v := range bundle.Architecture.Views {
		viewByID[v.ID] = v
	}
	for _, viewID := range viewIDs {
		res, err := Generate(bundle, viewID)
		diags = append(diags, res.Diagnostics...)
		if err != nil {
			return AsciiDocResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("generate view %s: %w", viewID, err)
		}
		viewSections = append(viewSections, asciidocViewSection{
			ID:      viewID,
			Kind:    res.View.Kind,
			Heading: viewHeading(res.View.Kind),
			Mermaid: strings.TrimSpace(res.Mermaid),
			Inf:     inferredDescription(res.View.Kind),
		})
		nodes := map[string]bool{}
		for _, n := range res.View.Nodes {
			nodes[n.ID] = true
		}
		viewNodeIDs[viewID] = nodes
	}

	designGroups := mapDesignGroups(design)
	designUnits := mapDesignUnits(design)
	inferredRuntime, runtimeDiags := inferRuntimeItems(bundle)
	inferredCode, codeDiags := inferCodeItems(bundle, options.CodeRoot)
	diags = append(diags, runtimeDiags...)
	diags = append(diags, codeDiags...)

	fgSections := make([]asciidocEntitySection, 0, len(bundle.Architecture.AuthoredArchitecture.FunctionalGroups))
	for _, g := range bundle.Architecture.AuthoredArchitecture.FunctionalGroups {
		details := buildDesignDetails(g.ID, g.Prose, designGroups[g.ID], bundle.Architecture.Views)
		fgSections = append(fgSections, asciidocEntitySection{
			Anchor:      referenceAnchor("idx-fg", g.ID),
			ID:          g.ID,
			Name:        nonEmpty(g.Name, g.ID),
			Description: strings.TrimSpace(g.Description),
			Tags:        strings.Join(g.Tags, ", "),
			Intro:       strings.TrimSpace(g.Prose),
			Details:     details,
		})
	}

	labelByID := buildLabelIndex(bundle.Architecture.AuthoredArchitecture)
	reqByUnit := requirementsByUnit(requirements.Requirements)
	attackByTarget := attackVectorsByTarget(bundle.Architecture.AuthoredArchitecture.Mappings, labelByID)

	fuSections := make([]asciidocUnitSection, 0, len(bundle.Architecture.AuthoredArchitecture.FunctionalUnits))
	for _, u := range bundle.Architecture.AuthoredArchitecture.FunctionalUnits {
		details := buildDesignDetails(u.ID, u.Prose, designUnits[u.ID], bundle.Architecture.Views)
		deps := unitDependencies(u.ID, bundle.Architecture.AuthoredArchitecture.Mappings, labelByID)
		inputs := reqByUnit[u.ID]
		if strings.TrimSpace(inputs) == "" {
			inputs = "inferred from mapped interactions and runtime context"
		}
		outputs := unitOutputs(u.ID, bundle.Architecture.AuthoredArchitecture.Mappings, labelByID)
		if strings.TrimSpace(outputs) == "" {
			outputs = "derived unit outcomes from authored mappings"
		}
		failureModes := attackByTarget[u.ID]
		if strings.TrimSpace(failureModes) == "" {
			failureModes = "no explicit attack vector targeting this unit"
		}
		intro := strings.TrimSpace(u.Prose)
		fuSections = append(fuSections, asciidocUnitSection{
			Anchor:       referenceAnchor("idx-fu", u.ID),
			GroupAnchor:  referenceAnchor("idx-fg", u.Group),
			ID:           u.ID,
			Name:         nonEmpty(u.Name, u.ID),
			Group:        u.Group,
			Tags:         strings.Join(u.Tags, ", "),
			Intro:        intro,
			Details:      details,
			WhatOwns:     unitOwnershipSummary(u, bundle.Architecture.AuthoredArchitecture.Mappings, reqByUnit, labelByID),
			Inputs:       inputs,
			Outputs:      outputs,
			Dependencies: deps,
			FailureModes: failureModes,
		})
	}
	sort.SliceStable(fuSections, func(i, j int) bool {
		if fuSections[i].Group != fuSections[j].Group {
			return fuSections[i].Group < fuSections[j].Group
		}
		return fuSections[i].ID < fuSections[j].ID
	})

	for i := range viewSections {
		v := viewByID[viewSections[i].ID]
		nodeSet := viewNodeIDs[viewSections[i].ID]
		gs := make([]asciidocEntitySection, 0, len(fgSections))
		for _, g := range fgSections {
			if !nodeSet[g.ID] {
				continue
			}
			detail := detailForView(g.Details, v.ID)
			gs = append(gs, asciidocEntitySection{
				Anchor:      g.Anchor,
				ID:          g.ID,
				Name:        g.Name,
				Description: g.Description,
				Tags:        g.Tags,
				Intro:       g.Intro,
				Details: []asciidocDesignDetail{
					detail,
				},
			})
		}
		us := make([]asciidocUnitSection, 0, len(fuSections))
		for _, u := range fuSections {
			if !nodeSet[u.ID] {
				continue
			}
			detail := detailForView(u.Details, v.ID)
			us = append(us, asciidocUnitSection{
				Anchor:       u.Anchor,
				GroupAnchor:  u.GroupAnchor,
				ID:           u.ID,
				Name:         u.Name,
				Group:        u.Group,
				Tags:         u.Tags,
				Intro:        u.Intro,
				Details:      []asciidocDesignDetail{detail},
				WhatOwns:     u.WhatOwns,
				Inputs:       u.Inputs,
				Outputs:      u.Outputs,
				Dependencies: u.Dependencies,
				FailureModes: u.FailureModes,
			})
		}
		viewSections[i].Groups = gs
		viewSections[i].Units = us
		switch v.Kind {
		case "runtime":
			apiRows := buildRuntimeAPIRows(inferredRuntime, bundle.Architecture.AuthoredArchitecture.Mappings)
			viewSections[i].RuntimeAPIRows = apiRows
			viewSections[i].RuntimeAPIGraph = buildRuntimeAPIMermaid(apiRows)
		case "deployment":
			depRows := buildDeploymentRows(inferredRuntime)
			viewSections[i].DeploymentRows = depRows
			viewSections[i].DeploymentGraph = buildDeploymentMermaid(depRows)
			opRows := buildPlatformOpsRows(bundle.Architecture.AuthoredArchitecture, inferredRuntime)
			viewSections[i].PlatformOpsRows = opRows
			viewSections[i].PlatformOpsGraph = buildPlatformOpsMermaid(opRows)
		case "security":
			secRows := buildSecurityPathRows(bundle.Architecture.AuthoredArchitecture, labelByID)
			viewSections[i].SecurityRows = secRows
			viewSections[i].SecurityGraph = buildSecurityPathMermaid(secRows)
			viewSections[i].SecurityObsRows = buildSecurityObservabilityRows(inferredRuntime, inferredCode)
		case "code-ownership":
			codeRows := buildCodeOwnershipRows(inferredCode)
			viewSections[i].InferredGraph = buildCodeOwnershipMermaid(codeRows, bundle.Architecture.AuthoredArchitecture)
			viewSections[i].InferredRows = codeRows
		}
	}

	reqSections := make([]asciidocRequirementSection, 0, len(requirements.Requirements))
	for _, r := range requirements.Requirements {
		reqSections = append(reqSections, asciidocRequirementSection{Anchor: referenceAnchor("req", r.ID), ID: r.ID, Text: strings.TrimSpace(r.Text), Notes: strings.TrimSpace(r.Notes)})
	}
	sort.SliceStable(reqSections, func(i, j int) bool { return reqSections[i].ID < reqSections[j].ID })
	reqMermaid := buildRequirementAlignmentMermaid(requirements.Requirements, labelByID)
	refIndex := buildReferenceIndex(bundle, requirements, inferredRuntime, inferredCode)
	linkTargets := buildLinkTargets(refIndex)
	terms := buildTermsFromCatalog(bundle.Catalog)
	sort.SliceStable(terms, func(i, j int) bool {
		leftName := strings.ToLower(strings.TrimSpace(terms[i].Name))
		rightName := strings.ToLower(strings.TrimSpace(terms[j].Name))
		if leftName != rightName {
			return leftName < rightName
		}
		return terms[i].ID < terms[j].ID
	})

	for i := range viewSections {
		for j := range viewSections[i].Groups {
			viewSections[i].Groups[j].Description = linkifyText(viewSections[i].Groups[j].Description, linkTargets)
			viewSections[i].Groups[j].Intro = linkifyText(viewSections[i].Groups[j].Intro, linkTargets)
			for k := range viewSections[i].Groups[j].Details {
				viewSections[i].Groups[j].Details[k].Narrative = linkifyText(viewSections[i].Groups[j].Details[k].Narrative, linkTargets)
			}
		}
		for j := range viewSections[i].Units {
			viewSections[i].Units[j].Intro = linkifyText(viewSections[i].Units[j].Intro, linkTargets)
			viewSections[i].Units[j].WhatOwns = linkifyText(viewSections[i].Units[j].WhatOwns, linkTargets)
			viewSections[i].Units[j].Inputs = linkifyText(viewSections[i].Units[j].Inputs, linkTargets)
			viewSections[i].Units[j].Outputs = linkifyText(viewSections[i].Units[j].Outputs, linkTargets)
			viewSections[i].Units[j].Dependencies = linkifyText(viewSections[i].Units[j].Dependencies, linkTargets)
			viewSections[i].Units[j].FailureModes = linkifyText(viewSections[i].Units[j].FailureModes, linkTargets)
			for k := range viewSections[i].Units[j].Details {
				viewSections[i].Units[j].Details[k].Narrative = linkifyText(viewSections[i].Units[j].Details[k].Narrative, linkTargets)
			}
		}
		if viewSections[i].Inf != "" {
			viewSections[i].Inf = linkifyText(viewSections[i].Inf, linkTargets)
		}
	}
	for i := range reqSections {
		reqSections[i].Text = linkifyText(reqSections[i].Text, linkTargets)
		reqSections[i].Notes = linkifyText(reqSections[i].Notes, linkTargets)
	}

	doc, err := renderAsciiDocTemplate(asciidocTemplateData{
		Title:        nonEmpty(design.Design.Title, bundle.Architecture.Model.Title),
		Introduction: linkifyText(strings.TrimSpace(bundle.Architecture.Model.Introduction), linkTargets),
		Terms:        terms,
		Purpose:      "This architecture description is generated from authored structure and inferred realization layers.",
		ReaderTracks: []string{"Product/domain engineers: Functional + Runtime", "Platform/SRE engineers: Deployment + Runtime", "Implementation engineers: Realization + Functional", "Security engineers: Security + Functional"},
		Legend:       []string{"Authored: intentional functional architecture", "Inferred: discovered runtime or code realization", "realizes: runtime to functional unit", "owned_by: code to functional unit", "traces_to: symbol to requirement"},
		ModelMeta: asciidocModelMeta{
			ID:             strings.TrimSpace(bundle.Architecture.Model.ID),
			Title:          strings.TrimSpace(bundle.Architecture.Model.Title),
			BaseCatalogRef: strings.TrimSpace(bundle.Architecture.Model.BaseCatalogRef),
		},
		LintRun: asciidocLintRun{
			ID:         strings.TrimSpace(requirements.LintRun.ID),
			Mode:       strings.TrimSpace(requirements.LintRun.Mode),
			CommaAsAnd: requirements.LintRun.CommaAsAnd,
			CatalogRef: strings.TrimSpace(requirements.LintRun.CatalogRef),
		},
		ViewConfig: renderViewConfig(bundle.Architecture.Views),
		InferenceHints: asciidocInferenceHints{
			RuntimeSources:           strings.Join(bundle.Architecture.InferenceHints.RuntimeSources, ", "),
			CodeSources:              strings.Join(bundle.Architecture.InferenceHints.CodeSources, ", "),
			ExpectedRuntimeKinds:     strings.Join(bundle.Architecture.InferenceHints.ExpectedRuntimeKinds, ", "),
			OwnershipResolutionOrder: strings.Join(bundle.Architecture.InferenceHints.OwnershipResolutionOrder, ", "),
		},
		Actors:             renderActors(bundle.Architecture.AuthoredArchitecture.Actors),
		AttackVectors:      renderAttackVectors(bundle.Architecture.AuthoredArchitecture.AttackVectors),
		ReferencedElements: renderReferencedElements(bundle.Architecture.AuthoredArchitecture.ReferencedElements),
		Mappings:           renderMappings(bundle.Architecture.AuthoredArchitecture.Mappings),
		InferredRuntime:    renderInferredRuntime(inferredRuntime),
		InferredCode:       renderInferredCode(inferredCode),
		Summary: asciidocSummary{
			FunctionalGroups:   listNamesFG(bundle.Architecture.AuthoredArchitecture.FunctionalGroups),
			FunctionalUnits:    listNamesFU(bundle.Architecture.AuthoredArchitecture.FunctionalUnits),
			Actors:             listNamesActors(bundle.Architecture.AuthoredArchitecture.Actors),
			AttackVectors:      listNamesVectors(bundle.Architecture.AuthoredArchitecture.AttackVectors),
			ReferencedElements: listNamesRefs(bundle.Architecture.AuthoredArchitecture.ReferencedElements),
		},
		Views:              viewSections,
		RequirementMermaid: reqMermaid,
		RequirementInf:     "Show requirement-to-unit mappings inferred from appliesTo and authored architecture ownership boundaries.",
		Requirements:       reqSections,
		ReferenceIndex:     refIndex,
	})
	if err != nil {
		return AsciiDocResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return AsciiDocResult{Document: doc, Diagnostics: validate.SortDiagnostics(diags)}, nil
}

func renderViewConfig(in []model.View) []asciidocViewConfig {
	out := make([]asciidocViewConfig, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocViewConfig{
			ID:    strings.TrimSpace(x.ID),
			Kind:  strings.TrimSpace(x.Kind),
			Roots: strings.Join(x.Roots, ", "),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func renderActors(in []model.Actor) []asciidocActorSection {
	out := make([]asciidocActorSection, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocActorSection{
			ID:          strings.TrimSpace(x.ID),
			Name:        strings.TrimSpace(x.Name),
			Description: strings.TrimSpace(x.Description),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func renderAttackVectors(in []model.AttackVector) []asciidocAttackVectorSection {
	out := make([]asciidocAttackVectorSection, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocAttackVectorSection{
			ID:          strings.TrimSpace(x.ID),
			Name:        strings.TrimSpace(x.Name),
			Description: strings.TrimSpace(x.Description),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func renderReferencedElements(in []model.ReferencedElement) []asciidocReferencedSection {
	out := make([]asciidocReferencedSection, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocReferencedSection{
			ID:    strings.TrimSpace(x.ID),
			Name:  strings.TrimSpace(x.Name),
			Kind:  strings.TrimSpace(x.Kind),
			Layer: strings.TrimSpace(x.Layer),
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

func renderMappings(in []model.Mapping) []asciidocMappingSection {
	out := make([]asciidocMappingSection, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocMappingSection{
			Type:        strings.TrimSpace(x.Type),
			From:        strings.TrimSpace(x.From),
			To:          strings.TrimSpace(x.To),
			Description: strings.TrimSpace(x.Description),
		})
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Type != out[j].Type {
			return out[i].Type < out[j].Type
		}
		if out[i].From != out[j].From {
			return out[i].From < out[j].From
		}
		return out[i].To < out[j].To
	})
	return out
}

func renderInferredRuntime(in []inferredRuntimeItem) []asciidocInferredRow {
	out := make([]asciidocInferredRow, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocInferredRow{Name: x.Name, Kind: x.Kind, Owner: x.Owner, Source: x.Source})
	}
	return out
}

func renderInferredCode(in []inferredCodeItem) []asciidocInferredRow {
	out := make([]asciidocInferredRow, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocInferredRow{Name: x.Element, Kind: x.Kind, Owner: x.Owner, Source: x.Source})
	}
	return out
}

func viewHeading(kind string) string {
	switch kind {
	case "authored-functional":
		return "Functional View"
	case "runtime":
		return "Runtime View"
	case "deployment":
		return "Deployment View"
	case "code-ownership":
		return "Realization View"
	case "security":
		return "Security View"
	default:
		return strings.Title(kind) + " View"
	}
}

func resolveViewIDs(bundle model.Bundle, options AsciiDocOptions) []string {
	if len(options.ViewIDs) > 0 {
		return append([]string(nil), options.ViewIDs...)
	}
	out := make([]string, 0, len(bundle.Architecture.Views))
	for _, v := range bundle.Architecture.Views {
		out = append(out, v.ID)
	}
	return out
}

func inferredDescription(kind string) string {
	switch kind {
	case "runtime":
		return "Show inferred runtime interaction and containment mapped to authored unit boundaries."
	case "deployment":
		return "Show inferred deployment artifacts and ownership mapping to authored units."
	case "code-ownership":
		return "Show inferred implementation structure and dependencies mapped to authored architecture."
	case "security":
		return "Show inferred exposure and dependency risk points aligned to unit boundaries."
	default:
		return "Show authored architecture scope for this view."
	}
}

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
	case "authored-functional":
		return "functional"
	case "runtime":
		return "runtime"
	case "deployment":
		return "deployment"
	case "code-ownership":
		return "code_ownership"
	case "security":
		return "security"
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
	return out
}

func referenceAnchor(kind, id string) string {
	return "REF_" + strings.ToUpper(strings.TrimSpace(kind)) + "_" + sanitizeNode(id)
}

func buildReferenceIndex(bundle model.Bundle, requirements model.RequirementsDocument, runtime []inferredRuntimeItem, code []inferredCodeItem) asciidocReferenceIndex {
	authored := []asciidocReferenceEntry{}
	addAuthored := func(anchorKind, kind, id, name, desc string) {
		authored = append(authored, asciidocReferenceEntry{
			Anchor:      referenceAnchor(anchorKind, id),
			ID:          strings.TrimSpace(id),
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

	return asciidocReferenceIndex{
		Authored: authored,
		Catalog:  catalog,
		Runtime:  runtimeRefs,
		Code:     codeRefs,
	}
}

func buildCatalogReferences(doc model.CatalogDocument) []asciidocReferenceEntry {
	out := []asciidocReferenceEntry{}
	add := func(kind string, entries []model.CatalogEntry) {
		for _, e := range entries {
			canonical := referenceAnchor("catalog", e.ID)
			out = append(out, asciidocReferenceEntry{
				Anchor:       referenceAnchor("idx-catalog", e.ID),
				TargetAnchor: canonical,
				ID:           e.ID,
				Name:         nonEmpty(e.Name, e.ID),
				Kind:         "Catalog " + kind,
				Description:  strings.TrimSpace(e.Definition),
			})
			for _, a := range e.Aliases {
				alias := strings.TrimSpace(a)
				if alias == "" {
					continue
				}
				out = append(out, asciidocReferenceEntry{
					Anchor:       referenceAnchor("catalog-alias", e.ID+"-"+alias),
					TargetAnchor: canonical,
					ID:           alias,
					Name:         nonEmpty(e.Name, e.ID),
					Kind:         "Catalog Alias",
					Description:  "Alias of " + e.ID,
				})
			}
		}
	}
	c := doc.Catalog
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
		out = append(out, asciidocReferenceEntry{
			Anchor:      referenceAnchor("rt", kind+"-"+id),
			ID:          id,
			Name:        id,
			Kind:        "Runtime " + kind,
			Description: "Owner: " + nonEmpty(r.Owner, "unresolved"),
			Source:      r.Source,
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
		out = append(out, asciidocReferenceEntry{
			Anchor:      referenceAnchor("code", c.Kind+"-"+id),
			ID:          id,
			Name:        id,
			Kind:        "Code " + c.Kind,
			Description: "Owner: " + nonEmpty(c.Owner, "unresolved"),
			Source:      c.Source,
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

func buildTermsFromCatalog(doc model.CatalogDocument) []asciidocTerm {
	out := []asciidocTerm{}
	out = append(out, builtInEngineeringModelTerms()...)
	add := func(entries []model.CatalogEntry) {
		for _, e := range entries {
			out = append(out, asciidocTerm{
				Anchor:     referenceAnchor("catalog", e.ID),
				ID:         strings.TrimSpace(e.ID),
				Name:       nonEmpty(strings.TrimSpace(e.Name), strings.TrimSpace(e.ID)),
				Definition: strings.TrimSpace(e.Definition),
				Aliases:    uniqueSorted(e.Aliases),
			})
		}
	}
	c := doc.Catalog
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
			Anchor:     referenceAnchor("engmodel", "EM-FUNCTIONAL-GROUP"),
			ID:         "EM-FUNCTIONAL-GROUP",
			Name:       "functional group",
			Definition: "A major authored capability area that groups related functional units.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-FUNCTIONAL-UNIT"),
			ID:         "EM-FUNCTIONAL-UNIT",
			Name:       "functional unit",
			Definition: "An authored working unit inside a functional group that owns specific behavior.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-RUNTIME-ELEMENT"),
			ID:         "EM-RUNTIME-ELEMENT",
			Name:       "runtime element",
			Definition: "An inferred runtime realization element discovered from infrastructure and deployment sources.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-CODE-ELEMENT"),
			ID:         "EM-CODE-ELEMENT",
			Name:       "code element",
			Definition: "An inferred code structure or ownership element discovered from source trees and build metadata.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-REFERENCED-ELEMENT"),
			ID:         "EM-REFERENCED-ELEMENT",
			Name:       "referenced element",
			Definition: "An architecture-relevant external, platform, or third-party dependency represented by role.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-ACTOR"),
			ID:         "EM-ACTOR",
			Name:       "actor",
			Definition: "A person or operational role that interacts with functional units.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-ATTACK-VECTOR"),
			ID:         "EM-ATTACK-VECTOR",
			Name:       "attack vector",
			Definition: "A technical misuse or attack path that targets functional, referenced, or runtime elements.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-AUTHORED-MAPPING"),
			ID:         "EM-AUTHORED-MAPPING",
			Name:       "authored mapping",
			Definition: "An explicit relationship declared in architecture inputs between authored or referenced elements.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-INFERRED-MAPPING"),
			ID:         "EM-INFERRED-MAPPING",
			Name:       "inferred mapping",
			Definition: "A discovered relationship that links inferred runtime/code elements upward to authored design.",
		},
		{
			Anchor:     referenceAnchor("engmodel", "EM-UPWARD-LINKING"),
			ID:         "EM-UPWARD-LINKING",
			Name:       "upward linking",
			Definition: "Rule where runtime and code elements point to stable functional groups/units; authored architecture does not depend on inferred IDs.",
		},
	}
}

type linkTarget struct {
	Anchor string
	Label  string
}

func buildLinkTargets(ref asciidocReferenceIndex) map[string]linkTarget {
	out := map[string]linkTarget{}
	add := func(token, anchor, label string) {
		token = strings.TrimSpace(token)
		anchor = strings.TrimSpace(anchor)
		label = strings.TrimSpace(label)
		if token == "" || anchor == "" {
			return
		}
		if label == "" {
			label = token
		}
		if _, exists := out[token]; exists {
			return
		}
		out[token] = linkTarget{Anchor: anchor, Label: label}
	}
	// Priority order matters; first match wins.
	for _, e := range ref.Authored {
		target := e.Anchor
		if strings.TrimSpace(e.TargetAnchor) != "" {
			target = strings.TrimSpace(e.TargetAnchor)
		}
		add(e.ID, target, e.ID)
		add(e.Name, target, e.Name)
	}
	for _, e := range ref.Catalog {
		target := e.Anchor
		if strings.TrimSpace(e.TargetAnchor) != "" {
			target = strings.TrimSpace(e.TargetAnchor)
		}
		add(e.ID, target, e.ID)
		// Avoid over-linking with short/common alias words.
		if strings.Contains(e.ID, "-") || strings.Contains(e.ID, " ") || len(strings.TrimSpace(e.ID)) >= 10 {
			add(e.ID, target, e.ID)
		}
		if len(strings.Fields(e.Name)) >= 2 {
			add(e.Name, target, e.Name)
		}
	}
	// For inferred entries, only link explicit IDs to avoid prose noise.
	for _, e := range ref.Runtime {
		target := e.Anchor
		if strings.TrimSpace(e.TargetAnchor) != "" {
			target = strings.TrimSpace(e.TargetAnchor)
		}
		add(e.ID, target, e.ID)
	}
	for _, e := range ref.Code {
		target := e.Anchor
		if strings.TrimSpace(e.TargetAnchor) != "" {
			target = strings.TrimSpace(e.TargetAnchor)
		}
		add(e.ID, target, e.ID)
	}
	return out
}

func linkifyText(text string, targets map[string]linkTarget) string {
	in := strings.TrimSpace(text)
	if in == "" || strings.Contains(in, "<<") {
		return text
	}
	type tokenInfo struct {
		Token string
		Link  linkTarget
	}
	items := make([]tokenInfo, 0, len(targets))
	for t, l := range targets {
		t = strings.TrimSpace(t)
		if len(t) < 4 {
			continue
		}
		items = append(items, tokenInfo{Token: t, Link: l})
	}
	sort.SliceStable(items, func(i, j int) bool { return len(items[i].Token) > len(items[j].Token) })

	type span struct {
		start int
		end   int
		repl  string
	}
	spans := []span{}
	used := make([]bool, len(text))
	isWordBound := func(s string, idx int) bool {
		if idx <= 0 || idx >= len(s) {
			return true
		}
		ch := s[idx]
		return !((ch >= 'A' && ch <= 'Z') || (ch >= 'a' && ch <= 'z') || (ch >= '0' && ch <= '9') || ch == '_' || ch == '-')
	}

	for _, it := range items {
		token := it.Token
		if token == "" {
			continue
		}
		link := "<<" + it.Link.Anchor + "," + it.Link.Label + ">>"
		start := 0
		for {
			pos := strings.Index(text[start:], token)
			if pos < 0 {
				break
			}
			s := start + pos
			e := s + len(token)
			ok := true
			if !(strings.Contains(token, " ") || strings.ContainsAny(token, "/:.") || regexp.MustCompile(`[A-Z]{2,}|-`).MatchString(token)) {
				ok = isWordBound(text, s-1) && isWordBound(text, e)
			}
			if ok {
				for i := s; i < e; i++ {
					if used[i] {
						ok = false
						break
					}
				}
			}
			if ok {
				for i := s; i < e; i++ {
					used[i] = true
				}
				spans = append(spans, span{start: s, end: e, repl: link})
			}
			start = e
		}
	}
	if len(spans) == 0 {
		return text
	}
	sort.SliceStable(spans, func(i, j int) bool { return spans[i].start < spans[j].start })
	var b strings.Builder
	last := 0
	for _, sp := range spans {
		if sp.start < last {
			continue
		}
		b.WriteString(text[last:sp.start])
		b.WriteString(sp.repl)
		last = sp.end
	}
	b.WriteString(text[last:])
	return b.String()
}

func requirementsByUnit(reqs []model.Requirement) map[string]string {
	set := map[string][]string{}
	for _, r := range reqs {
		for _, u := range r.AppliesTo {
			set[u] = append(set[u], r.ID)
		}
	}
	out := map[string]string{}
	for u, ids := range set {
		ids = uniqueSorted(ids)
		out[u] = strings.Join(ids, ", ")
	}
	return out
}

func unitDependencies(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.Type == "depends_on" && m.From == unitID {
			out = append(out, nonEmpty(labels[m.To], m.To))
		}
	}
	out = uniqueSorted(out)
	if len(out) == 0 {
		return "none"
	}
	return strings.Join(out, ", ")
}

func unitOutputs(unitID string, mappings []model.Mapping, labels map[string]string) string {
	out := []string{}
	for _, m := range mappings {
		if m.From == unitID && m.Type != "contains" {
			out = append(out, m.Type+" -> "+nonEmpty(labels[m.To], m.To))
		}
	}
	out = uniqueSorted(out)
	if len(out) == 0 {
		return ""
	}
	return strings.Join(out, "; ")
}

func unitOwnershipSummary(u model.FunctionalUnit, mappings []model.Mapping, reqByUnit map[string]string, labels map[string]string) string {
	areas := []string{}
	for _, m := range mappings {
		if m.From != u.ID {
			continue
		}
		switch m.Type {
		case "depends_on":
			areas = append(areas, "decision and orchestration flow to "+nonEmpty(labels[m.To], m.To))
		case "interacts_with":
			areas = append(areas, "interaction handling with "+nonEmpty(labels[m.To], m.To))
		}
	}
	areas = uniqueSorted(areas)
	if len(areas) > 2 {
		areas = areas[:2]
	}

	base := "functional responsibility for " + strings.ToLower(nonEmpty(u.Name, u.ID))
	if len(areas) > 0 {
		base = base + "; includes " + strings.Join(areas, " and ")
	}
	if req := strings.TrimSpace(reqByUnit[u.ID]); req != "" {
		base = base + "; primary requirement scope: " + req
	}
	return base
}

func attackVectorsByTarget(mappings []model.Mapping, labels map[string]string) map[string]string {
	set := map[string][]string{}
	for _, m := range mappings {
		if m.Type == "targets" {
			set[m.To] = append(set[m.To], nonEmpty(labels[m.From], m.From))
		}
	}
	out := map[string]string{}
	for k, v := range set {
		v = uniqueSorted(v)
		out[k] = strings.Join(v, ", ")
	}
	return out
}

func listNamesFG(in []model.FunctionalGroup) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func listNamesFU(in []model.FunctionalUnit) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func listNamesActors(in []model.Actor) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func listNamesVectors(in []model.AttackVector) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func listNamesRefs(in []model.ReferencedElement) string {
	out := make([]string, 0, len(in))
	for _, x := range in {
		out = append(out, nonEmpty(x.Name, x.ID))
	}
	sort.Strings(out)
	return strings.Join(out, ", ")
}

func uniqueSorted(in []string) []string {
	set := map[string]bool{}
	for _, s := range in {
		s = strings.TrimSpace(s)
		if s != "" {
			set[s] = true
		}
	}
	out := make([]string, 0, len(set))
	for s := range set {
		out = append(out, s)
	}
	sort.Strings(out)
	return out
}

func buildRequirementAlignmentMermaid(reqs []model.Requirement, labels map[string]string) string {
	lines := []string{"flowchart LR"}
	for _, r := range reqs {
		reqNode := "REQ_" + sanitizeNode(r.ID)
		lines = append(lines, "  "+reqNode+"[\""+escapeMermaidLabel(r.ID)+"\"]")
		for _, u := range uniqueSorted(r.AppliesTo) {
			target := "UNIT_" + sanitizeNode(u)
			label := nonEmpty(labels[u], u)
			lines = append(lines, "  "+target+"[\""+escapeMermaidLabel(label)+"\"]")
			lines = append(lines, "  "+reqNode+" --> "+target)
		}
	}
	return strings.Join(lines, "\n")
}

func sanitizeNode(s string) string {
	repl := strings.NewReplacer("-", "_", " ", "_", ".", "_", "/", "_", ":", "_", "\\", "_")
	out := repl.Replace(strings.ToUpper(strings.TrimSpace(s)))
	if out == "" {
		out = "NODE"
	}
	return out
}

func escapeMermaidLabel(s string) string {
	s = strings.ReplaceAll(s, "\"", "\\\"")
	s = strings.ReplaceAll(s, "\n", " ")
	return strings.TrimSpace(s)
}

func limitRuntimeRows(in []inferredRuntimeItem, n int) []inferredRuntimeItem {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

func limitCodeRows(in []inferredCodeItem, n int) []inferredCodeItem {
	if len(in) <= n {
		return in
	}
	return in[:n]
}

func buildRuntimeInferenceMermaid(in []inferredRuntimeItem) string {
	lines := []string{"flowchart LR"}
	ownerNode := map[string]string{}
	kindNode := map[string]string{}
	ownerIdx := 0
	kindIdx := 0
	for _, x := range in {
		owner := nonEmpty(strings.TrimSpace(x.Owner), "unresolved")
		kind := nonEmpty(strings.TrimSpace(x.Kind), "runtime")
		on, ok := ownerNode[owner]
		if !ok {
			on = fmt.Sprintf("OR%d", ownerIdx)
			ownerIdx++
			ownerNode[owner] = on
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", on, escapeMermaidLabel(owner)))
		}
		kn, ok := kindNode[kind]
		if !ok {
			kn = fmt.Sprintf("RK%d", kindIdx)
			kindIdx++
			kindNode[kind] = kn
			lines = append(lines, fmt.Sprintf("  %s((\"%s\"))", kn, escapeMermaidLabel(kind)))
		}
		lines = append(lines, fmt.Sprintf("  %s --> %s", on, kn))
	}
	return strings.Join(uniquePreserve(lines), "\n")
}

func buildCodeInferenceMermaid(in []inferredCodeItem) string {
	lines := []string{"flowchart LR"}
	ownerNode := map[string]string{}
	kindNode := map[string]string{}
	ownerIdx := 0
	kindIdx := 0
	for _, x := range in {
		owner := nonEmpty(strings.TrimSpace(x.Owner), "unresolved")
		kind := nonEmpty(strings.TrimSpace(x.Kind), "code")
		on, ok := ownerNode[owner]
		if !ok {
			on = fmt.Sprintf("CO%d", ownerIdx)
			ownerIdx++
			ownerNode[owner] = on
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", on, escapeMermaidLabel(owner)))
		}
		kn, ok := kindNode[kind]
		if !ok {
			kn = fmt.Sprintf("CK%d", kindIdx)
			kindIdx++
			kindNode[kind] = kn
			lines = append(lines, fmt.Sprintf("  %s((\"%s\"))", kn, escapeMermaidLabel(kind)))
		}
		lines = append(lines, fmt.Sprintf("  %s --> %s", on, kn))
	}
	return strings.Join(uniquePreserve(lines), "\n")
}

func buildCodeOwnershipRows(in []inferredCodeItem) []asciidocInferredRow {
	type moduleBucket struct {
		owner string
		mod   string
		langs map[string]bool
		files map[string]bool
	}
	moduleBuckets := map[string]*moduleBucket{}
	libraryRows := []asciidocInferredRow{}
	librarySeen := map[string]bool{}

	for _, c := range in {
		if c.Kind == "source_file" || c.Kind == "symbol" {
			path := codeItemPath(c)
			if path == "" {
				continue
			}
			mod := moduleFromPath(path)
			owner := nonEmpty(strings.TrimSpace(c.Owner), "unresolved")
			key := owner + "|" + mod
			b, ok := moduleBuckets[key]
			if !ok {
				b = &moduleBucket{
					owner: owner,
					mod:   mod,
					langs: map[string]bool{},
					files: map[string]bool{},
				}
				moduleBuckets[key] = b
			}
			if lg := languageFromPath(path); lg != "" {
				b.langs[lg] = true
			}
			b.files[path] = true
			continue
		}
		if strings.HasPrefix(c.Kind, "library_") {
			owner := nonEmpty(strings.TrimSpace(c.Owner), "unresolved")
			module := moduleFromPath(codeItemPath(c))
			libType := strings.TrimPrefix(c.Kind, "library_")
			kind := "library (" + strings.ReplaceAll(libType, "_", "-") + ")"
			key := owner + "|" + module + "|" + c.Element + "|" + kind
			if librarySeen[key] {
				continue
			}
			librarySeen[key] = true
			libraryRows = append(libraryRows, asciidocInferredRow{
				Name:   c.Element,
				Kind:   kind,
				Owner:  owner,
				Source: "module: " + module,
			})
		}
	}

	rows := make([]asciidocInferredRow, 0, len(moduleBuckets)+len(libraryRows))
	for _, b := range moduleBuckets {
		files := setToSortedSlice(b.files)
		display := files
		if len(display) > 4 {
			display = append(display[:4], fmt.Sprintf("+%d more", len(files)-4))
		}
		langs := setToSortedSlice(b.langs)
		kind := "module"
		if len(langs) > 0 {
			kind = "module (" + strings.Join(langs, ", ") + ")"
		}
		rows = append(rows, asciidocInferredRow{
			Name:   b.mod,
			Kind:   kind,
			Owner:  b.owner,
			Source: strings.Join(display, ", "),
		})
	}
	rows = append(rows, libraryRows...)
	sort.SliceStable(rows, func(i, j int) bool {
		if rows[i].Owner != rows[j].Owner {
			return rows[i].Owner < rows[j].Owner
		}
		if rows[i].Kind != rows[j].Kind {
			return rows[i].Kind < rows[j].Kind
		}
		return rows[i].Name < rows[j].Name
	})
	if len(rows) > 28 {
		return rows[:28]
	}
	return rows
}

func buildCodeOwnershipMermaid(rows []asciidocInferredRow, a model.AuthoredArchitecture) string {
	lines := []string{"flowchart TB"}
	fuToGroup := map[string]string{}
	for _, u := range a.FunctionalUnits {
		fuToGroup[u.ID] = strings.TrimSpace(u.Group)
	}
	fgLabel := map[string]string{}
	for _, g := range a.FunctionalGroups {
		fgLabel[g.ID] = nonEmpty(g.Name, g.ID)
	}

	seenFU := map[string]bool{}
	seenFG := map[string]bool{}
	seenFUEdge := map[string]bool{}
	seenLibEdge := map[string]bool{}

	for _, r := range rows {
		if !strings.HasPrefix(r.Kind, "library") {
			continue
		}
		fu := strings.TrimSpace(r.Owner)
		if fu == "" || fu == "unresolved" {
			continue
		}

		fun := "FU_" + sanitizeNode(fu)
		if !seenFU[fun] {
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_unit", fun, escapeMermaidLabel(fu)))
			seenFU[fun] = true
		}

		if fgID := fuToGroup[fu]; strings.TrimSpace(fgID) != "" {
			fgn := "FG_" + sanitizeNode(fgID)
			fg := nonEmpty(fgLabel[fgID], fgID)
			if !seenFG[fgn] {
				lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::functional_group", fgn, escapeMermaidLabel(fg)))
				seenFG[fgn] = true
			}
			edgeKey := fgn + "|" + fun
			if !seenFUEdge[edgeKey] {
				lines = append(lines, fmt.Sprintf("  %s -->|contains| %s", fgn, fun))
				seenFUEdge[edgeKey] = true
			}
		}

		ln := "LIB_" + sanitizeNode(r.Name)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]:::referenced_element", ln, escapeMermaidLabel(shortLibraryLabel(r.Name))))
		edgeKey := fun + "|" + ln
		if !seenLibEdge[edgeKey] {
			lines = append(lines, fmt.Sprintf("  %s -->|uses_library| %s", fun, ln))
			seenLibEdge[edgeKey] = true
		}
	}
	lines = append(lines,
		"  classDef functional_group fill:#e8f5e9,stroke:#1b5e20,stroke-width:1px;",
		"  classDef functional_unit fill:#e3f2fd,stroke:#0d47a1,stroke-width:1px;",
		"  classDef referenced_element fill:#f3e5f5,stroke:#6a1b9a,stroke-width:1px;",
	)
	return strings.Join(uniquePreserve(lines), "\n")
}

func shortLibraryLabel(lib string) string {
	x := strings.TrimSpace(lib)
	if x == "" {
		return x
	}
	x = strings.TrimPrefix(x, "./")
	x = strings.TrimPrefix(x, "crate::")
	if strings.Contains(x, "/") {
		parts := strings.Split(x, "/")
		if len(parts) > 2 {
			x = strings.Join(parts[len(parts)-2:], "/")
		}
	}
	if strings.Contains(x, "::") {
		parts := strings.Split(x, "::")
		if len(parts) > 3 {
			x = strings.Join(parts[len(parts)-3:], "::")
		}
	}
	return x
}

func codeItemPath(c inferredCodeItem) string {
	src := strings.TrimSpace(c.Source)
	if src == "" {
		return ""
	}
	if idx := strings.Index(src, ":"); idx > 0 {
		src = src[:idx]
	}
	return filepath.ToSlash(strings.TrimSpace(src))
}

func moduleFromPath(p string) string {
	p = filepath.ToSlash(strings.TrimSpace(p))
	if p == "" {
		return "root"
	}
	dir := filepath.ToSlash(filepath.Dir(p))
	if dir == "." || dir == "/" || dir == "" {
		base := filepath.Base(p)
		ext := filepath.Ext(base)
		return strings.TrimSuffix(base, ext)
	}
	return dir
}

func languageFromPath(p string) string {
	switch strings.ToLower(filepath.Ext(p)) {
	case ".go":
		return "go"
	case ".ts", ".tsx":
		return "typescript"
	case ".rs":
		return "rust"
	default:
		return ""
	}
}

func setToSortedSlice(in map[string]bool) []string {
	out := make([]string, 0, len(in))
	for x := range in {
		if strings.TrimSpace(x) != "" {
			out = append(out, strings.TrimSpace(x))
		}
	}
	sort.Strings(out)
	return out
}

func uniquePreserve(in []string) []string {
	seen := map[string]bool{}
	out := make([]string, 0, len(in))
	for _, s := range in {
		if seen[s] {
			continue
		}
		seen[s] = true
		out = append(out, s)
	}
	return out
}

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
		if r.Kind == "service" && len(r.Ports) > 0 {
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
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", cn, escapeMermaidLabel(r.Consumer)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", pn, escapeMermaidLabel(r.Provider)))
		lines = append(lines, fmt.Sprintf("  %s -->|API %s| %s", cn, escapeMermaidLabel(r.Ports), pn))
	}
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
		out = append(out, asciidocDeploymentRow{From: sourceName, To: kustomName, How: "source drives kustomization reconciliation"})
	}
	for _, r := range releases {
		if kustomName != "" {
			out = append(out, asciidocDeploymentRow{From: kustomName, To: r, How: "kustomization applies helm release"})
		}
	}
	for _, r := range releases {
		for _, w := range workloads {
			if strings.Contains(strings.ToLower(w), strings.ToLower(r)) {
				out = append(out, asciidocDeploymentRow{From: r, To: w, How: "release deploys runtime workload"})
			}
		}
	}
	for _, r := range releases {
		for _, ns := range namespaces {
			if strings.Contains(strings.ToLower(r), strings.ToLower(ns)) {
				out = append(out, asciidocDeploymentRow{From: r, To: ns, How: "release targets namespace"})
			}
		}
	}
	if clusterName != "" {
		for _, ns := range namespaces {
			out = append(out, asciidocDeploymentRow{From: ns, To: clusterName, How: "namespace part of cluster"})
		}
	}
	return out
}

func buildDeploymentMermaid(rows []asciidocDeploymentRow) string {
	lines := []string{"flowchart LR"}
	for _, r := range rows {
		fn := "DP_" + sanitizeNode(r.From)
		tn := "DP_" + sanitizeNode(r.To)
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", fn, escapeMermaidLabel(r.From)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", tn, escapeMermaidLabel(r.To)))
		lines = append(lines, fmt.Sprintf("  %s -->|%s| %s", fn, escapeMermaidLabel(r.How), tn))
	}
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
		lines = append(lines, fmt.Sprintf("  %s((\"%s\"))", an, escapeMermaidLabel(r.Actor)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", un, escapeMermaidLabel(r.Unit)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", tn, escapeMermaidLabel(r.Target)))
		lines = append(lines, fmt.Sprintf("  %s --> %s", an, un))
		lines = append(lines, fmt.Sprintf("  %s --> %s", un, tn))
	}
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
		lines = append(lines, fmt.Sprintf("  %s((\"%s\"))", avNode, escapeMermaidLabel(r.AttackVector)))
		lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", tNode, escapeMermaidLabel(r.Target)))
		lines = append(lines, fmt.Sprintf("  %s -->|targets| %s", avNode, tNode))
		for _, dep := range strings.Split(r.DependsOn, ",") {
			dep = strings.TrimSpace(dep)
			if dep == "" || dep == "none" {
				continue
			}
			dNode := "SEC_DEP_" + sanitizeNode(dep)
			lines = append(lines, fmt.Sprintf("  %s[\"%s\"]", dNode, escapeMermaidLabel(dep)))
			lines = append(lines, fmt.Sprintf("  %s -->|depends_on| %s", tNode, dNode))
		}
	}
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
