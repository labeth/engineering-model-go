// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/render/diagramstyle"
	"github.com/labeth/engineering-model-go/validate"
)

type AsciiDocOptions struct {
	ViewIDs          []string
	CodeRoot         string
	DecisionsDocPath string
}

type AsciiDocResult struct {
	Document          string
	DecisionsDocument string
	Diagnostics       []validate.Diagnostic
}

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014
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
	diags = append(diags, validateCatalogDescriptions(bundle.Catalog)...)
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
		viewCfg := viewByID[viewID]
		res, err := Generate(bundle, viewID)
		diags = append(diags, res.Diagnostics...)
		if err != nil {
			return AsciiDocResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("generate view %s: %w", viewID, err)
		}
		projectedNodes := make([]asciidocProjectedNode, 0, len(res.View.Nodes))
		for _, n := range res.View.Nodes {
			projectedNodes = append(projectedNodes, asciidocProjectedNode{ID: strings.TrimSpace(n.ID), Kind: strings.TrimSpace(n.Kind), Label: strings.TrimSpace(n.Label)})
		}
		sort.SliceStable(projectedNodes, func(i, j int) bool {
			if projectedNodes[i].Kind != projectedNodes[j].Kind {
				return projectedNodes[i].Kind < projectedNodes[j].Kind
			}
			return projectedNodes[i].ID < projectedNodes[j].ID
		})

		projectedMappings := make([]asciidocMappingSection, 0, len(res.View.Edges))
		for _, e := range res.View.Edges {
			projectedMappings = append(projectedMappings, asciidocMappingSection{Type: strings.TrimSpace(e.Type), From: strings.TrimSpace(e.From), To: strings.TrimSpace(e.To), Description: strings.TrimSpace(e.Label)})
		}
		sort.SliceStable(projectedMappings, func(i, j int) bool {
			if projectedMappings[i].Type != projectedMappings[j].Type {
				return projectedMappings[i].Type < projectedMappings[j].Type
			}
			if projectedMappings[i].From != projectedMappings[j].From {
				return projectedMappings[i].From < projectedMappings[j].From
			}
			return projectedMappings[i].To < projectedMappings[j].To
		})

		viewSections = append(viewSections, asciidocViewSection{
			ID:                        viewID,
			Kind:                      res.View.Kind,
			Heading:                   viewHeading(res.View.Kind),
			AuthoredStatus:            normalizeAuthoredStatus(viewCfg.AuthoredStatus),
			AuthoredStatusExplanation: normalizeAuthoredStatusExplanation(viewCfg.AuthoredStatusExplanation),
			Mermaid:                   strings.TrimSpace(res.Mermaid),
			Inf:                       inferredDescription(res.View.Kind),
			ViewQuestions:             viewQuestions(res.View.Kind),
			ProjectedNodes:            projectedNodes,
			ProjectedMappings:         projectedMappings,
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
	inferredVerification, verificationDiags := inferVerificationChecks(bundle, requirements, inferredCode, options.CodeRoot)
	diags = append(diags, runtimeDiags...)
	diags = append(diags, codeDiags...)
	diags = append(diags, verificationDiags...)
	evidenceByOwner := buildOwnerEvidence(inferredRuntime, inferredCode)

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
			inputs = "no explicit authored requirement trigger"
		}
		consumes := unitConsumers(u.ID, bundle.Architecture.AuthoredArchitecture.Mappings, labelByID)
		if strings.TrimSpace(consumes) == "" {
			consumes = "none"
		}
		produces := unitOutputs(u.ID, bundle.Architecture.AuthoredArchitecture.Mappings, labelByID)
		if strings.TrimSpace(produces) == "" {
			produces = "none"
		}
		threats := attackByTarget[u.ID]
		if strings.TrimSpace(threats) == "" {
			threats = "no explicit attack vector targeting this unit"
		}
		intro := strings.TrimSpace(u.Prose)
		fuSections = append(fuSections, asciidocUnitSection{
			Anchor:      referenceAnchor("idx-fu", u.ID),
			GroupAnchor: referenceAnchor("idx-fg", u.Group),
			ID:          u.ID,
			Name:        nonEmpty(u.Name, u.ID),
			Group:       u.Group,
			Tags:        strings.Join(u.Tags, ", "),
			Intro:       intro,
			Details:     details,
			WhatOwns:    unitOwnershipSummary(u, bundle.Architecture.AuthoredArchitecture.Mappings, reqByUnit, labelByID),
			Triggers:    inputs,
			Consumes:    consumes,
			Produces:    produces,
			DependsOn:   deps,
			Threats:     threats,
			Evidence:    nonEmpty(evidenceByOwner[u.ID], "authored unit with no direct derived runtime/code evidence yet"),
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
				Anchor:      u.Anchor,
				GroupAnchor: u.GroupAnchor,
				ID:          u.ID,
				Name:        u.Name,
				Group:       u.Group,
				Tags:        u.Tags,
				Intro:       u.Intro,
				Details:     []asciidocDesignDetail{detail},
				WhatOwns:    u.WhatOwns,
				Triggers:    u.Triggers,
				Consumes:    u.Consumes,
				Produces:    u.Produces,
				DependsOn:   u.DependsOn,
				Threats:     u.Threats,
				Evidence:    u.Evidence,
			})
		}
		if v.Kind == "communication" {
			for j := range us {
				us[j].Consumes = unitInboundInterfacesDetailed(us[j].ID, bundle.Architecture.AuthoredArchitecture.Mappings, labelByID)
				us[j].Produces = unitOutboundInterfacesDetailed(us[j].ID, bundle.Architecture.AuthoredArchitecture.Mappings, labelByID)
				us[j].Triggers = unitMessagesConsumed(us[j].ID, bundle.Architecture.AuthoredArchitecture.Mappings, labelByID)
			}
		}
		viewSections[i].Groups = gs
		viewSections[i].Units = us
		viewSections[i].CoverageGaps = viewCoverageGaps(v.Kind, us)
		viewSections[i].CoverageSummary = viewCoverageSummary(v.Kind, us)
		viewSections[i].NextActions = viewNextActions(v.Kind, viewSections[i].CoverageGaps)
		switch v.Kind {
		case "architecture-intent":
			viewSections[i].FuncContextGraph = buildFunctionalContextMermaid(bundle.Architecture.AuthoredArchitecture)
			viewSections[i].FuncDecompGraph = buildFunctionalDecompositionMermaid(bundle.Architecture.AuthoredArchitecture)
			viewSections[i].FuncMatrixTable = buildFunctionalManhattanTable(bundle.Architecture.AuthoredArchitecture)
			viewSections[i].FuncCollabGraph = buildFunctionalCollaborationMermaid(bundle.Architecture.AuthoredArchitecture)
		case "communication":
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
			viewSections[i].SecurityThreatScenarios = buildSecurityThreatScenarioRows(bundle.Architecture.AuthoredArchitecture, nodeSet, labelByID)
			viewSections[i].SecurityThreatAssumptions = buildSecurityThreatAssumptionRows(bundle.Architecture.AuthoredArchitecture, nodeSet, labelByID)
			viewSections[i].SecurityThreatOutOfScope = buildSecurityThreatOutOfScopeRows(bundle.Architecture.AuthoredArchitecture, nodeSet, labelByID)
			viewSections[i].SecurityThreatMitigations = buildSecurityThreatMitigationRows(bundle.Architecture.AuthoredArchitecture, nodeSet, labelByID)
			viewSections[i].SecurityControlChecks = buildSecurityControlVerificationRows(bundle.Architecture.AuthoredArchitecture, nodeSet, labelByID)
			viewSections[i].SecurityFlowRows = buildSecurityFlowRows(bundle.Architecture.AuthoredArchitecture, nodeSet, labelByID)
			viewSections[i].SecurityGraph = buildSecurityPathMermaid(secRows, inferredRuntime, inferredCode)
			viewSections[i].SecurityContextDFD = buildSecurityContextDFDMermaid(bundle.Architecture.AuthoredArchitecture, labelByID)
			viewSections[i].SecurityDataFlowDFD = buildSecurityDataFlowDFDMermaid(bundle.Architecture.AuthoredArchitecture, labelByID)
			viewSections[i].SecurityThreatOverlayDFD = buildSecurityThreatOverlayMermaid(bundle.Architecture.AuthoredArchitecture, labelByID)
			viewSections[i].SecurityObsRows = buildSecurityObservabilityRows(inferredRuntime, inferredCode)
			viewSections[i].SecurityAttackChapters = buildSecurityAttackChapters(bundle.Architecture.AuthoredArchitecture, us, nodeSet, secRows, inferredRuntime, inferredCode)
		case "traceability":
			codeRows := buildCodeOwnershipRows(inferredCode)
			viewSections[i].InferredGraph = buildCodeOwnershipMermaid(codeRows, bundle.Architecture.AuthoredArchitecture)
			viewSections[i].InferredRows = codeRows
		}
	}

	reqSections := make([]asciidocRequirementSection, 0, len(requirements.Requirements))
	for _, r := range requirements.Requirements {
		alignmentMermaid := buildRequirementAlignmentMermaid([]model.Requirement{r}, labelByID)
		coverageMermaid := buildRequirementCoverageMermaid([]model.Requirement{r}, inferredRuntime, inferredCode, inferredVerification, labelByID)
		reqSections = append(reqSections, asciidocRequirementSection{
			Anchor:               referenceAnchor("req", r.ID),
			ID:                   r.ID,
			Text:                 strings.TrimSpace(r.Text),
			Notes:                strings.TrimSpace(r.Notes),
			AlignmentMermaid:     alignmentMermaid,
			CoverageMermaid:      coverageMermaid,
			AlignmentExplanation: "What this diagram shows: direct authored mapping from this requirement to the functional units it applies to.",
			CoverageExplanation:  "What this diagram shows: this requirement-to-unit mapping extended with inferred runtime/code plus verification evidence attached to each requirement and unit.",
		})
	}
	sort.SliceStable(reqSections, func(i, j int) bool { return reqSections[i].ID < reqSections[j].ID })
	joinList := func(items []string) string {
		parts := make([]string, 0, len(items))
		for _, x := range items {
			t := strings.TrimSpace(x)
			if t == "" {
				continue
			}
			parts = append(parts, t)
		}
		if len(parts) == 0 {
			return "none"
		}
		return strings.Join(parts, ", ")
	}
	summarizeResults := func(results []inferredVerificationResult) string {
		if len(results) == 0 {
			return "none"
		}
		counts := map[string]int{}
		for _, r := range results {
			status := strings.ToLower(strings.TrimSpace(r.Status))
			if status == "" {
				status = "unknown"
			}
			counts[status]++
		}
		order := []string{"pass", "fail", "partial", "blocked", "not-run", "flaky", "unknown"}
		parts := make([]string, 0, len(order))
		for _, status := range order {
			if counts[status] == 0 {
				continue
			}
			parts = append(parts, fmt.Sprintf("%s:%d", status, counts[status]))
			delete(counts, status)
		}
		for status, count := range counts {
			parts = append(parts, fmt.Sprintf("%s:%d", status, count))
		}
		sort.Strings(parts)
		return strings.Join(parts, ", ")
	}
	verificationSections := make([]asciidocVerificationSection, 0, len(inferredVerification))
	verificationResultRows := make([]asciidocVerificationResultRow, 0)
	for _, v := range inferredVerification {
		verificationSections = append(verificationSections, asciidocVerificationSection{
			Anchor:        referenceAnchor("verify", strings.TrimSpace(v.ID)),
			IndexAnchor:   referenceAnchor("idx-ver", strings.TrimSpace(v.ID)),
			ID:            strings.TrimSpace(v.ID),
			Name:          strings.TrimSpace(v.Name),
			Kind:          strings.TrimSpace(v.Kind),
			Status:        strings.TrimSpace(v.Status),
			Verifies:      joinList(v.Verifies),
			TestCode:      joinList(v.CodeElements),
			DerivedOwners: joinList(v.DerivedOwners),
			Evidence:      joinList(v.Evidence),
			ResultSummary: summarizeResults(v.Results),
			Description:   strings.TrimSpace(v.Description),
		})
		for _, result := range v.Results {
			verificationResultRows = append(verificationResultRows, asciidocVerificationResultRow{
				CheckID:     strings.TrimSpace(v.ID),
				CheckName:   nonEmpty(strings.TrimSpace(v.Name), strings.TrimSpace(v.ID)),
				Requirement: strings.TrimSpace(result.Requirement),
				Status:      strings.TrimSpace(result.Status),
				Evidence:    nonEmpty(strings.TrimSpace(result.Evidence), "none"),
				Notes:       strings.TrimSpace(result.Notes),
			})
		}
	}
	sort.SliceStable(verificationSections, func(i, j int) bool {
		if verificationSections[i].ID != verificationSections[j].ID {
			return verificationSections[i].ID < verificationSections[j].ID
		}
		return verificationSections[i].Name < verificationSections[j].Name
	})
	sort.SliceStable(verificationResultRows, func(i, j int) bool {
		if verificationResultRows[i].Requirement != verificationResultRows[j].Requirement {
			return verificationResultRows[i].Requirement < verificationResultRows[j].Requirement
		}
		if verificationResultRows[i].CheckID != verificationResultRows[j].CheckID {
			return verificationResultRows[i].CheckID < verificationResultRows[j].CheckID
		}
		return verificationResultRows[i].Status < verificationResultRows[j].Status
	})

	reqMermaid := buildRequirementAlignmentCompactTable(requirements.Requirements)
	refIndex := buildReferenceIndex(bundle, requirements, inferredRuntime, inferredCode, inferredVerification)
	linkTargets := buildLinkTargets(refIndex)
	terms := buildTermsFromCatalog(bundle.Catalog)
	decisionSections := buildDecisionSections(bundle.Architecture.Decisions)
	decisionsDocPath := strings.TrimSpace(options.DecisionsDocPath)
	if decisionsDocPath == "" {
		decisionsDocPath = "DECISIONS.adoc"
	}
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
			viewSections[i].Units[j].Triggers = linkifyText(viewSections[i].Units[j].Triggers, linkTargets)
			viewSections[i].Units[j].Consumes = linkifyText(viewSections[i].Units[j].Consumes, linkTargets)
			viewSections[i].Units[j].Produces = linkifyText(viewSections[i].Units[j].Produces, linkTargets)
			viewSections[i].Units[j].DependsOn = linkifyText(viewSections[i].Units[j].DependsOn, linkTargets)
			viewSections[i].Units[j].Threats = linkifyText(viewSections[i].Units[j].Threats, linkTargets)
			viewSections[i].Units[j].Evidence = linkifyText(viewSections[i].Units[j].Evidence, linkTargets)
			for k := range viewSections[i].Units[j].Details {
				viewSections[i].Units[j].Details[k].Narrative = linkifyText(viewSections[i].Units[j].Details[k].Narrative, linkTargets)
			}
		}
		for j := range viewSections[i].SecurityAttackChapters {
			viewSections[i].SecurityAttackChapters[j].Name = linkifyText(viewSections[i].SecurityAttackChapters[j].Name, linkTargets)
			viewSections[i].SecurityAttackChapters[j].Description = linkifyText(viewSections[i].SecurityAttackChapters[j].Description, linkTargets)
			for k := range viewSections[i].SecurityAttackChapters[j].Units {
				viewSections[i].SecurityAttackChapters[j].Units[k].Intro = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].Intro, linkTargets)
				viewSections[i].SecurityAttackChapters[j].Units[k].WhatOwns = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].WhatOwns, linkTargets)
				viewSections[i].SecurityAttackChapters[j].Units[k].Triggers = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].Triggers, linkTargets)
				viewSections[i].SecurityAttackChapters[j].Units[k].Consumes = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].Consumes, linkTargets)
				viewSections[i].SecurityAttackChapters[j].Units[k].Produces = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].Produces, linkTargets)
				viewSections[i].SecurityAttackChapters[j].Units[k].DependsOn = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].DependsOn, linkTargets)
				viewSections[i].SecurityAttackChapters[j].Units[k].Threats = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].Threats, linkTargets)
				viewSections[i].SecurityAttackChapters[j].Units[k].Evidence = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].Evidence, linkTargets)
				for d := range viewSections[i].SecurityAttackChapters[j].Units[k].Details {
					viewSections[i].SecurityAttackChapters[j].Units[k].Details[d].Narrative = linkifyText(viewSections[i].SecurityAttackChapters[j].Units[k].Details[d].Narrative, linkTargets)
				}
			}
		}
		for j := range viewSections[i].SecurityThreatScenarios {
			viewSections[i].SecurityThreatScenarios[j].ID = linkifyText(viewSections[i].SecurityThreatScenarios[j].ID, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Title = linkifyText(viewSections[i].SecurityThreatScenarios[j].Title, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].AttackVector = linkifyText(viewSections[i].SecurityThreatScenarios[j].AttackVector, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Scope = linkifyText(viewSections[i].SecurityThreatScenarios[j].Scope, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Flows = linkifyText(viewSections[i].SecurityThreatScenarios[j].Flows, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Likelihood = linkifyText(viewSections[i].SecurityThreatScenarios[j].Likelihood, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Impact = linkifyText(viewSections[i].SecurityThreatScenarios[j].Impact, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Severity = linkifyText(viewSections[i].SecurityThreatScenarios[j].Severity, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Status = linkifyText(viewSections[i].SecurityThreatScenarios[j].Status, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Owner = linkifyText(viewSections[i].SecurityThreatScenarios[j].Owner, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Risk = linkifyText(viewSections[i].SecurityThreatScenarios[j].Risk, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Controls = linkifyText(viewSections[i].SecurityThreatScenarios[j].Controls, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Mitigations = linkifyText(viewSections[i].SecurityThreatScenarios[j].Mitigations, linkTargets)
			viewSections[i].SecurityThreatScenarios[j].Verifications = linkifyText(viewSections[i].SecurityThreatScenarios[j].Verifications, linkTargets)
		}
		for j := range viewSections[i].SecurityThreatAssumptions {
			viewSections[i].SecurityThreatAssumptions[j].ID = linkifyText(viewSections[i].SecurityThreatAssumptions[j].ID, linkTargets)
			viewSections[i].SecurityThreatAssumptions[j].Title = linkifyText(viewSections[i].SecurityThreatAssumptions[j].Title, linkTargets)
			viewSections[i].SecurityThreatAssumptions[j].Status = linkifyText(viewSections[i].SecurityThreatAssumptions[j].Status, linkTargets)
			viewSections[i].SecurityThreatAssumptions[j].Owner = linkifyText(viewSections[i].SecurityThreatAssumptions[j].Owner, linkTargets)
			viewSections[i].SecurityThreatAssumptions[j].AppliesTo = linkifyText(viewSections[i].SecurityThreatAssumptions[j].AppliesTo, linkTargets)
			viewSections[i].SecurityThreatAssumptions[j].Rationale = linkifyText(viewSections[i].SecurityThreatAssumptions[j].Rationale, linkTargets)
		}
		for j := range viewSections[i].SecurityThreatOutOfScope {
			viewSections[i].SecurityThreatOutOfScope[j].ID = linkifyText(viewSections[i].SecurityThreatOutOfScope[j].ID, linkTargets)
			viewSections[i].SecurityThreatOutOfScope[j].Title = linkifyText(viewSections[i].SecurityThreatOutOfScope[j].Title, linkTargets)
			viewSections[i].SecurityThreatOutOfScope[j].Status = linkifyText(viewSections[i].SecurityThreatOutOfScope[j].Status, linkTargets)
			viewSections[i].SecurityThreatOutOfScope[j].Owner = linkifyText(viewSections[i].SecurityThreatOutOfScope[j].Owner, linkTargets)
			viewSections[i].SecurityThreatOutOfScope[j].AppliesTo = linkifyText(viewSections[i].SecurityThreatOutOfScope[j].AppliesTo, linkTargets)
			viewSections[i].SecurityThreatOutOfScope[j].ExpiresOn = linkifyText(viewSections[i].SecurityThreatOutOfScope[j].ExpiresOn, linkTargets)
			viewSections[i].SecurityThreatOutOfScope[j].Reason = linkifyText(viewSections[i].SecurityThreatOutOfScope[j].Reason, linkTargets)
		}
		for j := range viewSections[i].SecurityThreatMitigations {
			viewSections[i].SecurityThreatMitigations[j].ID = linkifyText(viewSections[i].SecurityThreatMitigations[j].ID, linkTargets)
			viewSections[i].SecurityThreatMitigations[j].ThreatScenario = linkifyText(viewSections[i].SecurityThreatMitigations[j].ThreatScenario, linkTargets)
			viewSections[i].SecurityThreatMitigations[j].Control = linkifyText(viewSections[i].SecurityThreatMitigations[j].Control, linkTargets)
			viewSections[i].SecurityThreatMitigations[j].Status = linkifyText(viewSections[i].SecurityThreatMitigations[j].Status, linkTargets)
			viewSections[i].SecurityThreatMitigations[j].Effectiveness = linkifyText(viewSections[i].SecurityThreatMitigations[j].Effectiveness, linkTargets)
			viewSections[i].SecurityThreatMitigations[j].Owner = linkifyText(viewSections[i].SecurityThreatMitigations[j].Owner, linkTargets)
			viewSections[i].SecurityThreatMitigations[j].Verifications = linkifyText(viewSections[i].SecurityThreatMitigations[j].Verifications, linkTargets)
			viewSections[i].SecurityThreatMitigations[j].Notes = linkifyText(viewSections[i].SecurityThreatMitigations[j].Notes, linkTargets)
		}
		for j := range viewSections[i].SecurityControlChecks {
			viewSections[i].SecurityControlChecks[j].ID = linkifyText(viewSections[i].SecurityControlChecks[j].ID, linkTargets)
			viewSections[i].SecurityControlChecks[j].Control = linkifyText(viewSections[i].SecurityControlChecks[j].Control, linkTargets)
			viewSections[i].SecurityControlChecks[j].ThreatScenarios = linkifyText(viewSections[i].SecurityControlChecks[j].ThreatScenarios, linkTargets)
			viewSections[i].SecurityControlChecks[j].Risks = linkifyText(viewSections[i].SecurityControlChecks[j].Risks, linkTargets)
			viewSections[i].SecurityControlChecks[j].Method = linkifyText(viewSections[i].SecurityControlChecks[j].Method, linkTargets)
			viewSections[i].SecurityControlChecks[j].Status = linkifyText(viewSections[i].SecurityControlChecks[j].Status, linkTargets)
			viewSections[i].SecurityControlChecks[j].Owner = linkifyText(viewSections[i].SecurityControlChecks[j].Owner, linkTargets)
			viewSections[i].SecurityControlChecks[j].LastTested = linkifyText(viewSections[i].SecurityControlChecks[j].LastTested, linkTargets)
			viewSections[i].SecurityControlChecks[j].Findings = linkifyText(viewSections[i].SecurityControlChecks[j].Findings, linkTargets)
		}
		for j := range viewSections[i].SecurityFlowRows {
			viewSections[i].SecurityFlowRows[j].ID = linkifyText(viewSections[i].SecurityFlowRows[j].ID, linkTargets)
			viewSections[i].SecurityFlowRows[j].Title = linkifyText(viewSections[i].SecurityFlowRows[j].Title, linkTargets)
			viewSections[i].SecurityFlowRows[j].Kind = linkifyText(viewSections[i].SecurityFlowRows[j].Kind, linkTargets)
			viewSections[i].SecurityFlowRows[j].Direction = linkifyText(viewSections[i].SecurityFlowRows[j].Direction, linkTargets)
			viewSections[i].SecurityFlowRows[j].Frequency = linkifyText(viewSections[i].SecurityFlowRows[j].Frequency, linkTargets)
			viewSections[i].SecurityFlowRows[j].Source = linkifyText(viewSections[i].SecurityFlowRows[j].Source, linkTargets)
			viewSections[i].SecurityFlowRows[j].Destination = linkifyText(viewSections[i].SecurityFlowRows[j].Destination, linkTargets)
			viewSections[i].SecurityFlowRows[j].Protocol = linkifyText(viewSections[i].SecurityFlowRows[j].Protocol, linkTargets)
			viewSections[i].SecurityFlowRows[j].Authentication = linkifyText(viewSections[i].SecurityFlowRows[j].Authentication, linkTargets)
			viewSections[i].SecurityFlowRows[j].Encryption = linkifyText(viewSections[i].SecurityFlowRows[j].Encryption, linkTargets)
			viewSections[i].SecurityFlowRows[j].Integrity = linkifyText(viewSections[i].SecurityFlowRows[j].Integrity, linkTargets)
			viewSections[i].SecurityFlowRows[j].TrustBoundary = linkifyText(viewSections[i].SecurityFlowRows[j].TrustBoundary, linkTargets)
			viewSections[i].SecurityFlowRows[j].BoundaryCrossing = linkifyText(viewSections[i].SecurityFlowRows[j].BoundaryCrossing, linkTargets)
			viewSections[i].SecurityFlowRows[j].Threats = linkifyText(viewSections[i].SecurityFlowRows[j].Threats, linkTargets)
			viewSections[i].SecurityFlowRows[j].Data = linkifyText(viewSections[i].SecurityFlowRows[j].Data, linkTargets)
		}
		if viewSections[i].Inf != "" {
			viewSections[i].Inf = linkifyText(viewSections[i].Inf, linkTargets)
		}
		for j := range viewSections[i].ViewQuestions {
			viewSections[i].ViewQuestions[j] = linkifyText(viewSections[i].ViewQuestions[j], linkTargets)
		}
		for j := range viewSections[i].CoverageGaps {
			viewSections[i].CoverageGaps[j] = linkifyText(viewSections[i].CoverageGaps[j], linkTargets)
		}
		for j := range viewSections[i].NextActions {
			viewSections[i].NextActions[j] = linkifyText(viewSections[i].NextActions[j], linkTargets)
		}
		for j := range viewSections[i].ProjectedNodes {
			viewSections[i].ProjectedNodes[j].ID = linkifyText(viewSections[i].ProjectedNodes[j].ID, linkTargets)
			viewSections[i].ProjectedNodes[j].Kind = linkifyText(viewSections[i].ProjectedNodes[j].Kind, linkTargets)
			viewSections[i].ProjectedNodes[j].Label = linkifyText(viewSections[i].ProjectedNodes[j].Label, linkTargets)
		}
		for j := range viewSections[i].ProjectedMappings {
			viewSections[i].ProjectedMappings[j].Type = linkifyText(viewSections[i].ProjectedMappings[j].Type, linkTargets)
			viewSections[i].ProjectedMappings[j].From = linkifyText(viewSections[i].ProjectedMappings[j].From, linkTargets)
			viewSections[i].ProjectedMappings[j].To = linkifyText(viewSections[i].ProjectedMappings[j].To, linkTargets)
			viewSections[i].ProjectedMappings[j].Description = linkifyText(viewSections[i].ProjectedMappings[j].Description, linkTargets)
		}
		viewSections[i].CoverageSummary = linkifyText(viewSections[i].CoverageSummary, linkTargets)
	}
	for i := range reqSections {
		reqSections[i].Text = linkifyText(reqSections[i].Text, linkTargets)
		reqSections[i].Notes = linkifyText(reqSections[i].Notes, linkTargets)
	}
	for i := range verificationSections {
		verificationSections[i].Kind = linkifyText(verificationSections[i].Kind, linkTargets)
		verificationSections[i].Status = linkifyText(verificationSections[i].Status, linkTargets)
		verificationSections[i].Verifies = linkifyText(verificationSections[i].Verifies, linkTargets)
		verificationSections[i].TestCode = linkifyText(verificationSections[i].TestCode, linkTargets)
		verificationSections[i].DerivedOwners = linkifyText(verificationSections[i].DerivedOwners, linkTargets)
		verificationSections[i].Evidence = linkifyText(verificationSections[i].Evidence, linkTargets)
		verificationSections[i].ResultSummary = linkifyText(verificationSections[i].ResultSummary, linkTargets)
		verificationSections[i].Description = linkifyText(verificationSections[i].Description, linkTargets)
	}
	for i := range verificationResultRows {
		verificationResultRows[i].CheckID = linkifyText(verificationResultRows[i].CheckID, linkTargets)
		verificationResultRows[i].CheckName = linkifyText(verificationResultRows[i].CheckName, linkTargets)
		verificationResultRows[i].Requirement = linkifyText(verificationResultRows[i].Requirement, linkTargets)
		verificationResultRows[i].Status = linkifyText(verificationResultRows[i].Status, linkTargets)
		verificationResultRows[i].Evidence = linkifyText(verificationResultRows[i].Evidence, linkTargets)
		verificationResultRows[i].Notes = linkifyText(verificationResultRows[i].Notes, linkTargets)
	}

	templateData := asciidocTemplateData{
		Title:        nonEmpty(design.Design.Title, bundle.Architecture.Model.Title),
		Introduction: linkifyText(strings.TrimSpace(bundle.Architecture.Model.Introduction), linkTargets),
		HealthRows:   buildHealthRows(viewSections),
		Terms:        terms,
		Purpose:      "This architecture description is generated from authored structure and inferred realization layers.",
		ReaderTracks: []string{"Product/domain engineers: Functional + Runtime", "Platform/SRE engineers: Deployment + Runtime", "Implementation engineers: Realization + Functional", "Security engineers: Security + Functional"},
		Legend:       []string{"Authored: intentional functional architecture", "Inferred: discovered runtime or code realization", "realizes: runtime to functional unit", "implemented_by: verification to test code", "verifies: verification to requirement"},
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
		MermaidClassDefs:    diagramstyle.MermaidClassDefsBlock("  "),
		Views:               viewSections,
		RequirementMermaid:  reqMermaid,
		RequirementInf:      "Show requirement-to-unit mappings inferred from appliesTo and authored architecture ownership boundaries.",
		Requirements:        reqSections,
		Verifications:       verificationSections,
		VerificationResults: verificationResultRows,
		ReferenceIndex:      refIndex,
		Decisions:           decisionSections,
		DecisionsDocPath:    decisionsDocPath,
	}

	doc, err := renderAsciiDocTemplate(templateData)
	if err != nil {
		return AsciiDocResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	templateData.ReferenceIndex = applyReferenceBacklinks(doc, templateData.ReferenceIndex)
	doc, err = renderAsciiDocTemplate(templateData)
	if err != nil {
		return AsciiDocResult{Diagnostics: validate.SortDiagnostics(diags)}, err
	}

	return AsciiDocResult{Document: doc, DecisionsDocument: renderDecisionsDocument(decisionSections), Diagnostics: validate.SortDiagnostics(diags)}, nil
}
