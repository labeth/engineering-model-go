// ENGMODEL-OWNER-UNIT: FU-AI-VIEW-BUILDER
package engmodel

import (
	"encoding/json"
	"fmt"
	"hash/fnv"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TRLC-LINKS: REQ-EMG-002, REQ-EMG-010, REQ-EMG-012
func GenerateAIViewFromFiles(architecturePath, requirementsPath, designPath string, options AIViewOptions) (AIViewResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return AIViewResult{}, err
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return AIViewResult{}, err
	}
	design, err := model.LoadDesign(designPath)
	if err != nil {
		return AIViewResult{}, err
	}
	if strings.TrimSpace(options.CodeRoot) != "" && !filepath.IsAbs(options.CodeRoot) {
		baseDir := filepath.Dir(architecturePath)
		options.CodeRoot = filepath.Join(baseDir, options.CodeRoot)
	}
	reqPath, _ := filepath.Abs(requirementsPath)
	designAbsPath, _ := filepath.Abs(designPath)
	return generateAIView(bundle, requirements, design, reqPath, designAbsPath, options)
}

func GenerateAIView(bundle model.Bundle, requirements model.RequirementsDocument, design model.DesignDocument, options AIViewOptions) (AIViewResult, error) {
	baseDir := filepath.Dir(bundle.ArchitecturePath)
	reqPath := filepath.Join(baseDir, "requirements.yml")
	designPath := filepath.Join(baseDir, "design.yml")
	return generateAIView(bundle, requirements, design, reqPath, designPath, options)
}

func generateAIView(bundle model.Bundle, requirements model.RequirementsDocument, design model.DesignDocument, requirementsPath, designPath string, options AIViewOptions) (AIViewResult, error) {
	diags := validate.Bundle(bundle)
	diags = append(diags, validateCatalogDescriptions(bundle.Catalog)...)
	diags = append(diags, lintRequirementsEARS(requirements, bundle.Catalog)...)
	if validate.HasErrors(diags) {
		return AIViewResult{Diagnostics: validate.SortDiagnostics(diags)}, fmt.Errorf("validation failed")
	}

	if strings.TrimSpace(options.CodeRoot) != "" && !filepath.IsAbs(options.CodeRoot) {
		baseDir := filepath.Dir(bundle.ArchitecturePath)
		options.CodeRoot = filepath.Join(baseDir, options.CodeRoot)
	}

	inferredRuntime, runtimeDiags := inferRuntimeItems(bundle)
	inferredCode, codeDiags := inferCodeItems(bundle, options.CodeRoot)
	inferredVerification, verificationDiags := inferVerificationChecks(bundle, requirements, inferredCode, options.CodeRoot)
	diags = append(diags, runtimeDiags...)
	diags = append(diags, codeDiags...)
	diags = append(diags, verificationDiags...)
	diags = validate.SortDiagnostics(diags)

	doc := buildAIViewDocument(bundle, requirements, design, inferredRuntime, inferredCode, inferredVerification, requirementsPath, designPath, options)

	jsonBytes, err := json.MarshalIndent(doc, "", "  ")
	if err != nil {
		return AIViewResult{Diagnostics: diags}, fmt.Errorf("marshal ai json: %w", err)
	}
	jsonOut := string(jsonBytes) + "\n"
	mdOut := renderAIViewMarkdown(doc)
	edges := buildAIEdges(doc)
	edgesOut, err := renderAIEdgesNDJSON(edges)
	if err != nil {
		return AIViewResult{Diagnostics: diags}, fmt.Errorf("render ai edges: %w", err)
	}

	return AIViewResult{
		Document:    doc,
		JSON:        jsonOut,
		Markdown:    mdOut,
		EdgesNDJSON: edgesOut,
		Diagnostics: diags,
	}, nil
}

type aiBuildContext struct {
	bundle             model.Bundle
	requirements       model.RequirementsDocument
	design             model.DesignDocument
	runtime            []inferredRuntimeItem
	code               []inferredCodeItem
	verification       []inferredVerificationCheck
	requirementsPath   string
	designPath         string
	selectedViewIDs    []string
	codeRoots          []string
	sourceByEntityID   map[string]map[string]bool
	sourceBlocksByKey  map[string]*AISourceBlock
	runtimeEntityIDFor map[string]string
	codeEntityIDFor    map[string]string
}

func buildAIViewDocument(bundle model.Bundle, requirements model.RequirementsDocument, design model.DesignDocument, inferredRuntime []inferredRuntimeItem, inferredCode []inferredCodeItem, inferredVerification []inferredVerificationCheck, requirementsPath, designPath string, options AIViewOptions) AIViewDocument {
	ctx := aiBuildContext{
		bundle:             bundle,
		requirements:       requirements,
		design:             design,
		runtime:            inferredRuntime,
		code:               inferredCode,
		verification:       inferredVerification,
		requirementsPath:   requirementsPath,
		designPath:         designPath,
		selectedViewIDs:    resolveViewIDs(bundle, AsciiDocOptions{ViewIDs: options.ViewIDs}),
		sourceByEntityID:   map[string]map[string]bool{},
		sourceBlocksByKey:  map[string]*AISourceBlock{},
		runtimeEntityIDFor: map[string]string{},
		codeEntityIDFor:    map[string]string{},
	}
	ctx.codeRoots = resolveAICodeRoots(bundle, options.CodeRoot)

	a := bundle.Architecture.AuthoredArchitecture
	labelByID := buildLabelIndex(a)
	reqByUnit := requirementsByUnit(requirements.Requirements)
	attackByTarget := attackVectorsByTarget(a.Mappings, labelByID)

	fuToRuntime := map[string][]string{}
	fuToCode := map[string][]string{}
	reqToVerification := map[string][]string{}
	fuToVerification := map[string][]string{}
	fuToInterface := map[string][]string{}
	fuToData := map[string][]string{}
	fuToDeployment := map[string][]string{}
	fuToControl := map[string][]string{}
	fuToBoundary := map[string][]string{}
	fuToState := map[string][]string{}
	fuToEvent := map[string][]string{}
	fuToFlow := map[string][]string{}
	fuToFlowStep := map[string][]string{}

	entities := make([]AIEntity, 0, len(a.FunctionalGroups)+len(a.FunctionalUnits)+len(requirements.Requirements)+len(inferredRuntime)+len(inferredCode)+len(inferredVerification))

	for _, fg := range a.FunctionalGroups {
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, fg.ID, "functional_group", fmt.Sprintf("authored functional group %s", fg.ID), fg.ID)
		if strings.TrimSpace(designPath) != "" {
			ctx.addAuthoredYAMLSource(designPath, fg.ID, "design_yaml", fmt.Sprintf("design narrative for %s", fg.ID), fg.ID)
		}
		related := []string{}
		for _, fu := range a.FunctionalUnits {
			if strings.TrimSpace(fu.Group) == strings.TrimSpace(fg.ID) {
				related = append(related, fu.ID)
			}
		}
		related = uniqueSorted(related)
		entities = append(entities, AIEntity{
			ID:         fg.ID,
			Kind:       "functional_group",
			Title:      nonEmpty(strings.TrimSpace(fg.Name), strings.TrimSpace(fg.ID)),
			Summary:    nonEmpty(strings.TrimSpace(fg.Description), strings.TrimSpace(fg.Prose)),
			Origin:     "authored",
			Status:     "stable",
			RelatedIDs: related,
			SourceRefs: uniqueSorted([]string{sid}),
		})
	}

	for _, fu := range a.FunctionalUnits {
		triggers := splitCSVOrNone(reqByUnit[fu.ID])
		consumes := dependsOnTargets(fu.ID, a.Mappings)
		produces := unitProducedRelations(fu.ID, a.Mappings)
		threats := splitCSVOrNone(attackByTarget[fu.ID])
		runtimeIDs := []string{}
		codeIDs := []string{}
		verificationIDs := []string{}

		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, fu.ID, "functional_unit", fmt.Sprintf("authored functional unit %s", fu.ID), fu.ID)
		if strings.TrimSpace(designPath) != "" {
			ctx.addAuthoredYAMLSource(designPath, fu.ID, "design_yaml", fmt.Sprintf("design narrative for %s", fu.ID), fu.ID)
		}

		entities = append(entities, AIEntity{
			ID:      fu.ID,
			Kind:    "functional_unit",
			Title:   nonEmpty(strings.TrimSpace(fu.Name), strings.TrimSpace(fu.ID)),
			Summary: strings.TrimSpace(fu.Prose),
			Origin:  "authored",
			Status:  "stable",
			GroupID: strings.TrimSpace(fu.Group),
			Fields: AIEntityFields{
				Triggers: triggers,
				Consumes: consumes,
				Produces: produces,
				Threats:  threats,
			},
			RuntimeIDs:      runtimeIDs,
			CodeIDs:         codeIDs,
			VerificationIDs: verificationIDs,
			RelatedIDs:      uniqueSorted(append(append([]string{}, consumes...), fu.Group)),
			SourceRefs:      uniqueSorted([]string{sid}),
		})
	}

	for _, req := range requirements.Requirements {
		sid := ctx.addAuthoredYAMLSource(requirementsPath, req.ID, "requirement", fmt.Sprintf("authored requirement %s", req.ID), req.ID)
		entities = append(entities, AIEntity{
			ID:              strings.TrimSpace(req.ID),
			Kind:            "requirement",
			Title:           strings.TrimSpace(req.ID),
			Summary:         strings.TrimSpace(req.Text),
			Origin:          "authored",
			Status:          "stable",
			RelatedIDs:      uniqueSorted(req.AppliesTo),
			VerificationIDs: nil,
			SourceRefs:      uniqueSorted([]string{sid}),
		})
	}

	for _, itf := range a.Interfaces {
		id := strings.TrimSpace(itf.ID)
		if id == "" {
			continue
		}
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, id, "interface", fmt.Sprintf("authored interface %s", id), id)
		related := []string{}
		if owner := strings.TrimSpace(itf.Owner); owner != "" {
			related = append(related, owner)
			fuToInterface[owner] = append(fuToInterface[owner], id)
		}
		entities = append(entities, AIEntity{
			ID:         id,
			Kind:       "interface",
			Title:      nonEmpty(strings.TrimSpace(itf.Name), id),
			Summary:    strings.TrimSpace(itf.Protocol + " " + itf.Endpoint),
			Origin:     "authored",
			Status:     "stable",
			RelatedIDs: uniqueSorted(related),
			SourceRefs: []string{sid},
		})
	}

	for _, obj := range a.DataObjects {
		id := strings.TrimSpace(obj.ID)
		if id == "" {
			continue
		}
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, id, "data_object", fmt.Sprintf("authored data object %s", id), id)
		entities = append(entities, AIEntity{
			ID:         id,
			Kind:       "data_object",
			Title:      nonEmpty(strings.TrimSpace(obj.Name), id),
			Summary:    strings.TrimSpace(obj.Sensitivity),
			Origin:     "authored",
			Status:     "stable",
			RelatedIDs: nil,
			SourceRefs: []string{sid},
		})
	}

	for _, target := range a.DeploymentTargets {
		id := strings.TrimSpace(target.ID)
		if id == "" {
			continue
		}
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, id, "deployment_target", fmt.Sprintf("authored deployment target %s", id), id)
		entities = append(entities, AIEntity{
			ID:         id,
			Kind:       "deployment_target",
			Title:      nonEmpty(strings.TrimSpace(target.Name), id),
			Summary:    strings.TrimSpace(target.Environment + " " + target.Cluster + " " + target.Namespace),
			Origin:     "authored",
			Status:     "stable",
			RelatedIDs: nil,
			SourceRefs: []string{sid},
		})
	}

	for _, ctrl := range a.Controls {
		id := strings.TrimSpace(ctrl.ID)
		if id == "" {
			continue
		}
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, id, "control", fmt.Sprintf("authored control %s", id), id)
		entities = append(entities, AIEntity{
			ID:         id,
			Kind:       "control",
			Title:      nonEmpty(strings.TrimSpace(ctrl.Name), id),
			Summary:    nonEmpty(strings.TrimSpace(ctrl.Description), strings.TrimSpace(ctrl.Category)),
			Origin:     "authored",
			Status:     "stable",
			RelatedIDs: nil,
			SourceRefs: []string{sid},
		})
	}

	for _, boundary := range a.TrustBoundaries {
		id := strings.TrimSpace(boundary.ID)
		if id == "" {
			continue
		}
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, id, "trust_boundary", fmt.Sprintf("authored trust boundary %s", id), id)
		entities = append(entities, AIEntity{
			ID:         id,
			Kind:       "trust_boundary",
			Title:      nonEmpty(strings.TrimSpace(boundary.Name), id),
			Summary:    strings.TrimSpace(boundary.Description),
			Origin:     "authored",
			Status:     "stable",
			RelatedIDs: nil,
			SourceRefs: []string{sid},
		})
	}

	for _, state := range a.States {
		id := strings.TrimSpace(state.ID)
		if id == "" {
			continue
		}
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, id, "state", fmt.Sprintf("authored state %s", id), id)
		entities = append(entities, AIEntity{
			ID:         id,
			Kind:       "state",
			Title:      nonEmpty(strings.TrimSpace(state.Name), id),
			Summary:    strings.TrimSpace(state.Description),
			Origin:     "authored",
			Status:     "stable",
			RelatedIDs: nil,
			SourceRefs: []string{sid},
		})
	}

	for _, event := range a.Events {
		id := strings.TrimSpace(event.ID)
		if id == "" {
			continue
		}
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, id, "event", fmt.Sprintf("authored event %s", id), id)
		entities = append(entities, AIEntity{
			ID:         id,
			Kind:       "event",
			Title:      nonEmpty(strings.TrimSpace(event.Name), id),
			Summary:    strings.TrimSpace(event.Description),
			Origin:     "authored",
			Status:     "stable",
			RelatedIDs: nil,
			SourceRefs: []string{sid},
		})
	}

	for _, flow := range a.Flows {
		flowID := strings.TrimSpace(flow.ID)
		if flowID == "" {
			continue
		}
		sid := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, flowID, "flow", fmt.Sprintf("authored flow %s", flowID), flowID)
		stepIDs := []string{}
		for _, step := range flow.Steps {
			stepID := strings.TrimSpace(step.ID)
			if stepID == "" {
				continue
			}
			entityID := flowID + "::" + stepID
			stepIDs = append(stepIDs, entityID)
			related := []string{flowID}
			if ref := strings.TrimSpace(step.Ref); ref != "" {
				related = append(related, ref)
				if strings.HasPrefix(ref, "FU-") {
					fuToFlow[ref] = append(fuToFlow[ref], flowID)
					fuToFlowStep[ref] = append(fuToFlowStep[ref], entityID)
				}
			}
			for _, next := range step.Next {
				next = strings.TrimSpace(next)
				if next == "" {
					continue
				}
				related = append(related, flowID+"::"+next)
			}
			for _, onErr := range step.OnError {
				onErr = strings.TrimSpace(onErr)
				if onErr == "" {
					continue
				}
				related = append(related, flowID+"::"+onErr)
			}
			stepSource := ctx.addAuthoredYAMLSource(bundle.ArchitecturePath, entityID, "flow_step", fmt.Sprintf("flow step %s", entityID), entityID)
			action := strings.TrimSpace(step.Action)
			if action == "" {
				action = stepID
			}
			summary := action
			if len(step.DataIn) > 0 || len(step.DataOut) > 0 {
				summary = fmt.Sprintf("%s (in: %s; out: %s)", action, strings.Join(step.DataIn, ", "), strings.Join(step.DataOut, ", "))
			}
			entities = append(entities, AIEntity{
				ID:         entityID,
				Kind:       "flow_step",
				Title:      action,
				Summary:    strings.TrimSpace(summary),
				Origin:     "authored",
				Status:     "stable",
				RelatedIDs: uniqueSorted(related),
				SourceRefs: []string{stepSource},
			})
		}
		entities = append(entities, AIEntity{
			ID:          flowID,
			Kind:        "flow",
			Title:       nonEmpty(strings.TrimSpace(flow.Title), flowID),
			Summary:     fmt.Sprintf("entry: %s; exits: %s; steps: %d", strings.Join(flow.Entry, ", "), strings.Join(flow.Exits, ", "), len(stepIDs)),
			Origin:      "authored",
			Status:      "stable",
			RelatedIDs:  uniqueSorted(stepIDs),
			FlowStepIDs: uniqueSorted(stepIDs),
			SourceRefs:  []string{sid},
		})
	}

	for _, r := range inferredRuntime {
		id := aiRuntimeEntityID(r)
		ctx.runtimeEntityIDFor[aiRuntimeItemKey(r)] = id
		if strings.TrimSpace(r.Owner) != "" && strings.TrimSpace(r.Owner) != "unresolved" {
			fuToRuntime[r.Owner] = append(fuToRuntime[r.Owner], id)
		}

		sid := ctx.addArtifactSource(r.Source, "inferred_runtime", id, fmt.Sprintf("runtime evidence for %s", id), strings.TrimSpace(r.Name))
		entity := AIEntity{
			ID:      id,
			Kind:    "runtime_element",
			Title:   nonEmpty(strings.TrimSpace(r.Name), id),
			Summary: fmt.Sprintf("Inferred runtime %s owned by %s", strings.TrimSpace(r.Kind), nonEmpty(strings.TrimSpace(r.Owner), "unresolved")),
			Origin:  "inferred",
			Status:  runtimeStatus(r),
			RelatedIDs: func() []string {
				if strings.TrimSpace(r.Owner) == "" || strings.TrimSpace(r.Owner) == "unresolved" {
					return nil
				}
				return []string{strings.TrimSpace(r.Owner)}
			}(),
			SourceRefs: []string{sid},
		}
		entity.FieldProvenance = append(entity.FieldProvenance, AIFieldProvenance{
			Field:      "owner",
			Origin:     "inferred",
			Confidence: runtimeConfidence(r),
			SourceRefs: entity.SourceRefs,
		})
		entities = append(entities, entity)
	}

	for _, c := range inferredCode {
		id := aiCodeEntityID(c)
		ctx.codeEntityIDFor[aiCodeItemKey(c)] = id
		if strings.TrimSpace(c.Owner) != "" && strings.TrimSpace(c.Owner) != "unresolved" {
			fuToCode[c.Owner] = append(fuToCode[c.Owner], id)
		}
		sid := ctx.addArtifactSource(c.Source, "inferred_code", id, fmt.Sprintf("code evidence for %s", id), strings.TrimSpace(c.Element))
		entity := AIEntity{
			ID:      id,
			Kind:    "code_element",
			Title:   nonEmpty(strings.TrimSpace(c.Element), id),
			Summary: fmt.Sprintf("Inferred code %s owned by %s", strings.TrimSpace(c.Kind), nonEmpty(strings.TrimSpace(c.Owner), "unresolved")),
			Origin:  "inferred",
			Status:  codeStatus(c),
			RelatedIDs: func() []string {
				if strings.TrimSpace(c.Owner) == "" || strings.TrimSpace(c.Owner) == "unresolved" {
					return nil
				}
				return []string{strings.TrimSpace(c.Owner)}
			}(),
			SourceRefs: []string{sid},
		}
		entity.FieldProvenance = append(entity.FieldProvenance, AIFieldProvenance{
			Field:      "owner",
			Origin:     "inferred",
			Confidence: codeConfidence(c),
			SourceRefs: entity.SourceRefs,
		})
		entities = append(entities, entity)
	}

	for _, v := range inferredVerification {
		id := strings.TrimSpace(v.ID)
		for _, reqID := range v.Verifies {
			reqID = strings.TrimSpace(reqID)
			if reqID == "" {
				continue
			}
			reqToVerification[reqID] = append(reqToVerification[reqID], id)
		}
		for _, owner := range v.DerivedOwners {
			owner = strings.TrimSpace(owner)
			if owner == "" {
				continue
			}
			fuToVerification[owner] = append(fuToVerification[owner], id)
		}

		sourceRefs := []string{}
		for _, ev := range v.Evidence {
			sid := ctx.addArtifactSource(ev, "verification_artifact", id, fmt.Sprintf("verification evidence for %s", id), strings.TrimSpace(v.Name))
			sourceRefs = append(sourceRefs, sid)
		}
		if len(sourceRefs) == 0 {
			sourceRefs = []string{ctx.addSourceBlock("verification_artifact", "none", 0, 0, fmt.Sprintf("verification evidence for %s", id), []string{id})}
		}

		codeIDs := []string{}
		for _, ce := range v.CodeElements {
			ce = strings.TrimSpace(ce)
			if ce == "" {
				continue
			}
			if strings.HasPrefix(ce, "CODE-") {
				codeIDs = append(codeIDs, ce)
				continue
			}
			mapped := findAIEntityCodeIDByElement(ce, inferredCode)
			if mapped != "" {
				codeIDs = append(codeIDs, mapped)
			}
		}

		entity := AIEntity{
			ID:              id,
			Kind:            "verification",
			Title:           nonEmpty(strings.TrimSpace(v.Name), id),
			Summary:         nonEmpty(strings.TrimSpace(v.Description), "Inferred verification check."),
			Origin:          "verification",
			Status:          strings.TrimSpace(v.Status),
			RequirementIDs:  uniqueSorted(v.Verifies),
			CodeIDs:         uniqueSorted(codeIDs),
			RelatedIDs:      uniqueSorted(v.DerivedOwners),
			FieldProvenance: []AIFieldProvenance{{Field: "requirement_ids", Origin: "inferred", Confidence: verificationConfidence(v), SourceRefs: uniqueSorted(sourceRefs)}},
			SourceRefs:      uniqueSorted(sourceRefs),
		}
		entities = append(entities, entity)
	}

	entityByID := map[string]*AIEntity{}
	for i := range entities {
		entityByID[entities[i].ID] = &entities[i]
	}
	for _, m := range a.Mappings {
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		if from == "" || to == "" {
			continue
		}
		if fromEntity := entityByID[from]; fromEntity != nil {
			fromEntity.RelatedIDs = uniqueSorted(append(fromEntity.RelatedIDs, to))
		}
		if toEntity := entityByID[to]; toEntity != nil {
			toEntity.RelatedIDs = uniqueSorted(append(toEntity.RelatedIDs, from))
		}
		if strings.HasPrefix(from, "FU-") {
			if toEntity := entityByID[to]; toEntity != nil {
				switch toEntity.Kind {
				case "interface":
					fuToInterface[from] = append(fuToInterface[from], to)
				case "data_object":
					fuToData[from] = append(fuToData[from], to)
				case "deployment_target":
					fuToDeployment[from] = append(fuToDeployment[from], to)
				case "control":
					fuToControl[from] = append(fuToControl[from], to)
				case "trust_boundary":
					fuToBoundary[from] = append(fuToBoundary[from], to)
				case "state":
					fuToState[from] = append(fuToState[from], to)
				case "event":
					fuToEvent[from] = append(fuToEvent[from], to)
				case "flow":
					fuToFlow[from] = append(fuToFlow[from], to)
				case "flow_step":
					fuToFlowStep[from] = append(fuToFlowStep[from], to)
				}
			}
		}
	}

	for i := range entities {
		if entities[i].Kind != "functional_unit" {
			continue
		}
		id := entities[i].ID
		entities[i].RequirementIDs = uniqueSorted(splitCSVOrNone(reqByUnit[id]))
		entities[i].RuntimeIDs = uniqueSorted(fuToRuntime[id])
		entities[i].CodeIDs = uniqueSorted(fuToCode[id])
		entities[i].VerificationIDs = uniqueSorted(fuToVerification[id])
		entities[i].InterfaceIDs = uniqueSorted(fuToInterface[id])
		entities[i].DataObjectIDs = uniqueSorted(fuToData[id])
		entities[i].DeploymentIDs = uniqueSorted(fuToDeployment[id])
		entities[i].ControlIDs = uniqueSorted(fuToControl[id])
		entities[i].TrustBoundaryIDs = uniqueSorted(fuToBoundary[id])
		entities[i].StateIDs = uniqueSorted(fuToState[id])
		entities[i].EventIDs = uniqueSorted(fuToEvent[id])
		entities[i].FlowIDs = uniqueSorted(fuToFlow[id])
		entities[i].FlowStepIDs = uniqueSorted(fuToFlowStep[id])
		if len(entities[i].RuntimeIDs) > 0 {
			entities[i].FieldProvenance = append(entities[i].FieldProvenance, AIFieldProvenance{
				Field:      "runtime_ids",
				Origin:     "inferred",
				Confidence: "medium",
				SourceRefs: ctx.sourceRefsForEntityIDs(entities[i].RuntimeIDs),
			})
		}
		if len(entities[i].CodeIDs) > 0 {
			entities[i].FieldProvenance = append(entities[i].FieldProvenance, AIFieldProvenance{
				Field:      "code_ids",
				Origin:     "inferred",
				Confidence: "medium",
				SourceRefs: ctx.sourceRefsForEntityIDs(entities[i].CodeIDs),
			})
		}
		allEvidenceIDs := []string{}
		allEvidenceIDs = append(allEvidenceIDs, entities[i].RuntimeIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].CodeIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].InterfaceIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].DataObjectIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].DeploymentIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].ControlIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].TrustBoundaryIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].StateIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].EventIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].FlowIDs...)
		allEvidenceIDs = append(allEvidenceIDs, entities[i].FlowStepIDs...)
		entities[i].SourceRefs = uniqueSorted(append(entities[i].SourceRefs, ctx.sourceRefsForEntityIDs(allEvidenceIDs)...))
	}

	for i := range entities {
		if entities[i].Kind != "requirement" {
			continue
		}
		id := entities[i].ID
		entities[i].VerificationIDs = uniqueSorted(reqToVerification[id])
		if len(entities[i].VerificationIDs) > 0 {
			entities[i].FieldProvenance = append(entities[i].FieldProvenance, AIFieldProvenance{
				Field:      "verification_ids",
				Origin:     "inferred",
				Confidence: "medium",
				SourceRefs: ctx.sourceRefsForEntityIDs(entities[i].VerificationIDs),
			})
		}
		entities[i].SourceRefs = uniqueSorted(append(entities[i].SourceRefs, ctx.sourceRefsForEntityIDs(entities[i].VerificationIDs)...))
		if reqEntity, ok := entityByID[id]; ok {
			_ = reqEntity
		}
	}

	for i := range entities {
		entities[i].SourceRefs = uniqueSorted(entities[i].SourceRefs)
		sort.SliceStable(entities[i].FieldProvenance, func(left, right int) bool {
			return entities[i].FieldProvenance[left].Field < entities[i].FieldProvenance[right].Field
		})
	}

	sort.SliceStable(entities, func(left, right int) bool {
		lr := aiEntityKindRank(entities[left].Kind)
		rr := aiEntityKindRank(entities[right].Kind)
		if lr != rr {
			return lr < rr
		}
		return entities[left].ID < entities[right].ID
	})

	index := AIEntityIndex{
		FunctionalGroupIDs:  collectEntityIDsByKind(entities, "functional_group"),
		FunctionalUnitIDs:   collectEntityIDsByKind(entities, "functional_unit"),
		RequirementIDs:      collectEntityIDsByKind(entities, "requirement"),
		RuntimeIDs:          collectEntityIDsByKind(entities, "runtime_element"),
		CodeIDs:             collectEntityIDsByKind(entities, "code_element"),
		VerificationIDs:     collectEntityIDsByKind(entities, "verification"),
		InterfaceIDs:        collectEntityIDsByKind(entities, "interface"),
		DataObjectIDs:       collectEntityIDsByKind(entities, "data_object"),
		DeploymentTargetIDs: collectEntityIDsByKind(entities, "deployment_target"),
		ControlIDs:          collectEntityIDsByKind(entities, "control"),
		TrustBoundaryIDs:    collectEntityIDsByKind(entities, "trust_boundary"),
		StateIDs:            collectEntityIDsByKind(entities, "state"),
		EventIDs:            collectEntityIDsByKind(entities, "event"),
		FlowIDs:             collectEntityIDsByKind(entities, "flow"),
		FlowStepIDs:         collectEntityIDsByKind(entities, "flow_step"),
		Lookup:              make([]AIEntityLookup, 0, len(entities)),
	}
	for _, e := range entities {
		index.Lookup = append(index.Lookup, AIEntityLookup{ID: e.ID, Kind: e.Kind, Title: e.Title})
	}

	supportPaths := buildAISupportPaths(entities)
	entryPoints := buildAIEntryPoints(entities, supportPaths)
	implementationPaths := buildAIImplementationPaths(entities, supportPaths)
	gaps := buildAIGapReport(entities, supportPaths)

	modelSummary := AIModelSummary{
		ID:    strings.TrimSpace(bundle.Architecture.Model.ID),
		Title: strings.TrimSpace(bundle.Architecture.Model.Title),
		Counts: AIModelCounts{
			FunctionalGroups:  len(index.FunctionalGroupIDs),
			FunctionalUnits:   len(index.FunctionalUnitIDs),
			Requirements:      len(index.RequirementIDs),
			Runtime:           len(index.RuntimeIDs),
			Code:              len(index.CodeIDs),
			Verification:      len(index.VerificationIDs),
			Interfaces:        len(index.InterfaceIDs),
			DataObjects:       len(index.DataObjectIDs),
			DeploymentTargets: len(index.DeploymentTargetIDs),
			Controls:          len(index.ControlIDs),
			TrustBoundaries:   len(index.TrustBoundaryIDs),
			States:            len(index.StateIDs),
			Events:            len(index.EventIDs),
			Flows:             len(index.FlowIDs),
			FlowSteps:         len(index.FlowStepIDs),
			Views:             len(ctx.selectedViewIDs),
		},
	}
	for _, ep := range entryPoints {
		modelSummary.EntryPointIDs = append(modelSummary.EntryPointIDs, ep.ID)
	}
	modelSummary.EntryPointIDs = uniqueSorted(modelSummary.EntryPointIDs)

	sourceBlocks := ctx.finalizeSourceBlocks()

	return AIViewDocument{
		SchemaVersion:       "ai-view/v1",
		Model:               modelSummary,
		EntryPoints:         entryPoints,
		Gaps:                gaps,
		EntityIndex:         index,
		SupportPaths:        supportPaths,
		ImplementationPaths: implementationPaths,
		Entities:            entities,
		SourceBlocks:        sourceBlocks,
	}
}

func resolveAICodeRoots(bundle model.Bundle, codeRoot string) []string {
	baseDir := filepath.Dir(bundle.ArchitecturePath)
	roots := []string{}
	if strings.TrimSpace(codeRoot) != "" {
		roots = append(roots, strings.TrimSpace(codeRoot))
	}
	for _, src := range bundle.Architecture.InferenceHints.CodeSources {
		roots = append(roots, resolveSourcePath(baseDir, src))
	}
	out := []string{}
	seen := map[string]bool{}
	for _, root := range roots {
		if strings.TrimSpace(root) == "" {
			continue
		}
		abs := root
		if !filepath.IsAbs(abs) {
			if x, err := filepath.Abs(abs); err == nil {
				abs = x
			}
		}
		if seen[abs] {
			continue
		}
		seen[abs] = true
		out = append(out, abs)
	}
	sort.Strings(out)
	return out
}

func splitCSVOrNone(s string) []string {
	t := strings.TrimSpace(s)
	if t == "" || t == "none" || strings.HasPrefix(strings.ToLower(t), "no explicit") {
		return nil
	}
	parts := strings.Split(t, ",")
	out := []string{}
	for _, p := range parts {
		x := strings.TrimSpace(p)
		if x != "" {
			out = append(out, x)
		}
	}
	return uniqueSorted(out)
}

func dependsOnTargets(unitID string, mappings []model.Mapping) []string {
	out := []string{}
	for _, m := range mappings {
		if strings.TrimSpace(m.Type) == "depends_on" && strings.TrimSpace(m.From) == strings.TrimSpace(unitID) {
			out = append(out, strings.TrimSpace(m.To))
		}
	}
	return uniqueSorted(out)
}

func unitProducedRelations(unitID string, mappings []model.Mapping) []string {
	out := []string{}
	for _, m := range mappings {
		if strings.TrimSpace(m.From) != strings.TrimSpace(unitID) {
			continue
		}
		if strings.TrimSpace(m.Type) == "contains" {
			continue
		}
		out = append(out, strings.TrimSpace(m.Type)+":"+strings.TrimSpace(m.To))
	}
	return uniqueSorted(out)
}

func aiRuntimeItemKey(r inferredRuntimeItem) string {
	return strings.TrimSpace(r.Kind) + "|" + strings.TrimSpace(r.Name) + "|" + strings.TrimSpace(r.Source)
}

func aiCodeItemKey(c inferredCodeItem) string {
	return strings.TrimSpace(c.Kind) + "|" + strings.TrimSpace(c.Element) + "|" + strings.TrimSpace(c.Source)
}

func aiRuntimeEntityID(r inferredRuntimeItem) string {
	return "RT-" + strings.ToUpper(sanitizeNode(strings.TrimSpace(r.Kind)+"-"+strings.TrimSpace(r.Name)+"-"+shortHash(strings.TrimSpace(r.Source))))
}

func aiCodeEntityID(c inferredCodeItem) string {
	return "CODE-" + strings.ToUpper(sanitizeNode(strings.TrimSpace(c.Kind)+"-"+strings.TrimSpace(c.Element)+"-"+shortHash(strings.TrimSpace(c.Source))))
}

func shortHash(in string) string {
	h := fnv.New32a()
	_, _ = h.Write([]byte(in))
	return fmt.Sprintf("%06X", h.Sum32())[:6]
}

func findAIEntityCodeIDByElement(element string, inferredCode []inferredCodeItem) string {
	element = strings.TrimSpace(element)
	for _, c := range inferredCode {
		if strings.TrimSpace(c.Element) == element {
			return aiCodeEntityID(c)
		}
	}
	return ""
}

func runtimeStatus(r inferredRuntimeItem) string {
	if strings.TrimSpace(r.Owner) == "" || strings.TrimSpace(r.Owner) == "unresolved" {
		return "owner-unresolved"
	}
	return "inferred"
}

func runtimeConfidence(r inferredRuntimeItem) string {
	if strings.TrimSpace(r.Owner) == "" || strings.TrimSpace(r.Owner) == "unresolved" {
		return "low"
	}
	switch strings.ToLower(strings.TrimSpace(r.Kind)) {
	case "lambda_function", "deployment", "workload", "service", "helmrelease":
		return "high"
	case "terraform_resource", "eventbridge_rule", "eventbridge_target", "queue", "topic":
		return "medium"
	default:
		return "medium"
	}
}

func codeStatus(c inferredCodeItem) string {
	if strings.TrimSpace(c.Owner) == "" || strings.TrimSpace(c.Owner) == "unresolved" {
		return "owner-unresolved"
	}
	return "inferred"
}

func codeConfidence(c inferredCodeItem) string {
	if strings.TrimSpace(c.Owner) == "" || strings.TrimSpace(c.Owner) == "unresolved" {
		return "low"
	}
	switch strings.ToLower(strings.TrimSpace(c.Kind)) {
	case "source_file":
		return "high"
	case "symbol", "library_first_party", "library_external", "library_stdlib":
		return "medium"
	default:
		return "medium"
	}
}

func verificationConfidence(v inferredVerificationCheck) string {
	if len(v.Results) == 0 {
		return "low"
	}
	status := strings.ToLower(strings.TrimSpace(v.Status))
	switch status {
	case "pass":
		return "high"
	case "fail", "partial", "blocked":
		return "medium"
	default:
		return "low"
	}
}

func collectEntityIDsByKind(entities []AIEntity, kind string) []string {
	out := []string{}
	for _, e := range entities {
		if e.Kind == kind {
			out = append(out, e.ID)
		}
	}
	return uniqueSorted(out)
}

func aiEntityKindRank(kind string) int {
	switch kind {
	case "functional_group":
		return 1
	case "functional_unit":
		return 2
	case "requirement":
		return 3
	case "interface", "data_object", "deployment_target", "control", "trust_boundary", "state", "event", "flow", "flow_step":
		return 4
	case "runtime_element":
		return 5
	case "code_element":
		return 6
	case "verification":
		return 7
	default:
		return 99
	}
}

func buildAISupportPaths(entities []AIEntity) []AISupportPath {
	entityByID := map[string]AIEntity{}
	for _, e := range entities {
		entityByID[e.ID] = e
	}

	support := []AISupportPath{}
	for _, e := range entities {
		if e.Kind != "requirement" {
			continue
		}
		fus := uniqueSorted(e.RelatedIDs)
		verIDs := uniqueSorted(e.VerificationIDs)
		runtimeIDs := []string{}
		codeIDs := []string{}
		interfaceIDs := []string{}
		dataIDs := []string{}
		deploymentIDs := []string{}
		controlIDs := []string{}
		boundaryIDs := []string{}
		stateIDs := []string{}
		eventIDs := []string{}
		flowIDs := []string{}
		flowStepIDs := []string{}
		for _, fuID := range fus {
			fu, ok := entityByID[fuID]
			if !ok || fu.Kind != "functional_unit" {
				continue
			}
			runtimeIDs = append(runtimeIDs, fu.RuntimeIDs...)
			codeIDs = append(codeIDs, fu.CodeIDs...)
			interfaceIDs = append(interfaceIDs, fu.InterfaceIDs...)
			dataIDs = append(dataIDs, fu.DataObjectIDs...)
			deploymentIDs = append(deploymentIDs, fu.DeploymentIDs...)
			controlIDs = append(controlIDs, fu.ControlIDs...)
			boundaryIDs = append(boundaryIDs, fu.TrustBoundaryIDs...)
			stateIDs = append(stateIDs, fu.StateIDs...)
			eventIDs = append(eventIDs, fu.EventIDs...)
			flowIDs = append(flowIDs, fu.FlowIDs...)
			flowStepIDs = append(flowStepIDs, fu.FlowStepIDs...)
		}
		runtimeIDs = uniqueSorted(runtimeIDs)
		codeIDs = uniqueSorted(codeIDs)
		interfaceIDs = uniqueSorted(interfaceIDs)
		dataIDs = uniqueSorted(dataIDs)
		deploymentIDs = uniqueSorted(deploymentIDs)
		controlIDs = uniqueSorted(controlIDs)
		boundaryIDs = uniqueSorted(boundaryIDs)
		stateIDs = uniqueSorted(stateIDs)
		eventIDs = uniqueSorted(eventIDs)
		flowIDs = uniqueSorted(flowIDs)
		flowStepIDs = uniqueSorted(flowStepIDs)

		path := []string{e.ID}
		if len(fus) > 0 {
			path = append(path, fus[0])
		}
		if len(runtimeIDs) > 0 {
			path = append(path, runtimeIDs[0])
		}
		if len(codeIDs) > 0 {
			path = append(path, codeIDs[0])
		}
		for _, extras := range [][]string{interfaceIDs, dataIDs, deploymentIDs, controlIDs, boundaryIDs, stateIDs, eventIDs, flowIDs, flowStepIDs} {
			if len(extras) > 0 {
				path = append(path, extras[0])
			}
		}
		if len(verIDs) > 0 {
			path = append(path, verIDs[0])
		}
		path = uniquePreserve(path)

		confidence := "low"
		hasImplementation := len(runtimeIDs)+len(codeIDs)+len(interfaceIDs)+len(dataIDs)+len(deploymentIDs)+len(controlIDs)+len(boundaryIDs)+len(stateIDs)+len(eventIDs)+len(flowIDs)+len(flowStepIDs) > 0
		switch {
		case len(verIDs) > 0 && hasImplementation:
			confidence = "high"
		case len(verIDs) > 0 || hasImplementation:
			confidence = "medium"
		}

		sourceRefs := []string{}
		sourceRefs = append(sourceRefs, e.SourceRefs...)
		if len(fus) > 0 {
			if fu, ok := entityByID[fus[0]]; ok {
				sourceRefs = append(sourceRefs, fu.SourceRefs...)
			}
		}
		if len(verIDs) > 0 {
			if v, ok := entityByID[verIDs[0]]; ok {
				sourceRefs = append(sourceRefs, v.SourceRefs...)
			}
		}

		support = append(support, AISupportPath{
			ID:                "PATH-" + strings.ToUpper(sanitizeNode(e.ID)),
			FromID:            e.ID,
			ToVerificationIDs: verIDs,
			Path:              path,
			Summary:           fmt.Sprintf("Support path for %s from authored scope to inferred evidence and verification.", e.ID),
			Confidence:        confidence,
			SourceRefs:        uniqueSorted(sourceRefs),
		})
	}
	sort.SliceStable(support, func(i, j int) bool {
		return support[i].ID < support[j].ID
	})
	return support
}

func buildAIEntryPoints(entities []AIEntity, supportPaths []AISupportPath) []AIEntryPoint {
	requirementsWithSupport := []string{}
	requirementsWithGaps := []string{}
	fuWithEvidence := []string{}
	lowConfidenceInferred := []string{}
	verificationFailures := []string{}

	for _, sp := range supportPaths {
		switch sp.Confidence {
		case "high", "medium":
			requirementsWithSupport = append(requirementsWithSupport, sp.FromID)
		default:
			requirementsWithGaps = append(requirementsWithGaps, sp.FromID)
		}
	}
	for _, e := range entities {
		switch e.Kind {
		case "functional_unit":
			if len(e.RuntimeIDs)+len(e.CodeIDs)+len(e.VerificationIDs)+len(e.InterfaceIDs)+len(e.DataObjectIDs)+len(e.DeploymentIDs)+len(e.ControlIDs)+len(e.TrustBoundaryIDs)+len(e.StateIDs)+len(e.EventIDs)+len(e.FlowIDs)+len(e.FlowStepIDs) > 0 {
				fuWithEvidence = append(fuWithEvidence, e.ID)
			}
		case "runtime_element", "code_element":
			low := false
			for _, p := range e.FieldProvenance {
				if strings.ToLower(strings.TrimSpace(p.Confidence)) == "low" {
					low = true
					break
				}
			}
			if low {
				lowConfidenceInferred = append(lowConfidenceInferred, e.ID)
			}
		case "verification":
			st := strings.ToLower(strings.TrimSpace(e.Status))
			if st == "fail" || st == "blocked" || st == "partial" {
				verificationFailures = append(verificationFailures, e.ID)
			}
		}
	}

	entryPoints := []AIEntryPoint{
		{
			ID:        "EP-REQ-COVERAGE",
			Kind:      "requirements",
			Title:     "Requirements with direct support paths",
			EntityIDs: uniqueSorted(requirementsWithSupport),
		},
		{
			ID:        "EP-REQ-GAPS",
			Kind:      "requirements",
			Title:     "Requirements with low-confidence support",
			EntityIDs: uniqueSorted(requirementsWithGaps),
		},
		{
			ID:        "EP-FU-EVIDENCE",
			Kind:      "functional_units",
			Title:     "Functional units with runtime/code/verification evidence",
			EntityIDs: uniqueSorted(fuWithEvidence),
		},
		{
			ID:        "EP-LOW-CONFIDENCE-INFERRED",
			Kind:      "inferred",
			Title:     "Low-confidence inferred entities",
			EntityIDs: uniqueSorted(lowConfidenceInferred),
		},
		{
			ID:        "EP-VERIFICATION-FAILURES",
			Kind:      "verification",
			Title:     "Verification checks with failing/partial status",
			EntityIDs: uniqueSorted(verificationFailures),
		},
	}
	sort.SliceStable(entryPoints, func(i, j int) bool {
		return entryPoints[i].ID < entryPoints[j].ID
	})
	return entryPoints
}

func buildAIGapReport(entities []AIEntity, supportPaths []AISupportPath) AIGapReport {
	entityByID := map[string]AIEntity{}
	for _, e := range entities {
		entityByID[e.ID] = e
	}

	reqWithoutVerification := []string{}
	reqLowConfidence := []string{}
	for _, sp := range supportPaths {
		if strings.TrimSpace(sp.FromID) == "" {
			continue
		}
		reqEntity, ok := entityByID[sp.FromID]
		if !ok || reqEntity.Kind != "requirement" {
			continue
		}
		if len(reqEntity.VerificationIDs) == 0 {
			reqWithoutVerification = append(reqWithoutVerification, reqEntity.ID)
		}
		if strings.ToLower(strings.TrimSpace(sp.Confidence)) == "low" {
			reqLowConfidence = append(reqLowConfidence, reqEntity.ID)
		}
	}

	fuWithoutCode := []string{}
	fuWithoutRuntime := []string{}
	fuWithoutTests := []string{}
	verificationFailures := []string{}
	lowConfidenceInferred := []string{}

	for _, e := range entities {
		switch e.Kind {
		case "functional_unit":
			if len(e.CodeIDs) == 0 {
				fuWithoutCode = append(fuWithoutCode, e.ID)
			}
			if len(e.RuntimeIDs) == 0 {
				fuWithoutRuntime = append(fuWithoutRuntime, e.ID)
			}
			if len(e.VerificationIDs) == 0 {
				fuWithoutTests = append(fuWithoutTests, e.ID)
			}
		case "verification":
			st := strings.ToLower(strings.TrimSpace(e.Status))
			if st == "fail" || st == "blocked" || st == "partial" {
				verificationFailures = append(verificationFailures, e.ID)
			}
		case "runtime_element", "code_element":
			for _, p := range e.FieldProvenance {
				if strings.ToLower(strings.TrimSpace(p.Confidence)) == "low" {
					lowConfidenceInferred = append(lowConfidenceInferred, e.ID)
					break
				}
			}
		}
	}

	return AIGapReport{
		RequirementsWithoutVerification: uniqueSorted(reqWithoutVerification),
		RequirementsLowConfidence:       uniqueSorted(reqLowConfidence),
		FunctionalUnitsWithoutCode:      uniqueSorted(fuWithoutCode),
		FunctionalUnitsWithoutRuntime:   uniqueSorted(fuWithoutRuntime),
		FunctionalUnitsWithoutTests:     uniqueSorted(fuWithoutTests),
		VerificationFailures:            uniqueSorted(verificationFailures),
		LowConfidenceInferred:           uniqueSorted(lowConfidenceInferred),
	}
}

func buildAIImplementationPaths(entities []AIEntity, supportPaths []AISupportPath) []AIImplementationPath {
	entityByID := map[string]AIEntity{}
	for _, e := range entities {
		entityByID[e.ID] = e
	}

	paths := []AIImplementationPath{}
	for _, sp := range supportPaths {
		req, ok := entityByID[sp.FromID]
		if !ok || req.Kind != "requirement" {
			continue
		}
		fuIDs := []string{}
		for _, id := range req.RelatedIDs {
			if strings.HasPrefix(strings.TrimSpace(id), "FU-") {
				fuIDs = append(fuIDs, strings.TrimSpace(id))
			}
		}
		fuIDs = uniqueSorted(fuIDs)

		impacted := []string{req.ID}
		impacted = append(impacted, fuIDs...)
		steps := []AIImplementationStep{{
			Order:      1,
			Action:     "Confirm requirement intent and acceptance criteria before code changes",
			EntityID:   req.ID,
			EntityKind: req.Kind,
			SourceRefs: req.SourceRefs,
		}}

		stepOrder := 2
		for _, fuID := range fuIDs {
			fu, ok := entityByID[fuID]
			if !ok {
				continue
			}
			impacted = append(impacted, fu.InterfaceIDs...)
			impacted = append(impacted, fu.DataObjectIDs...)
			impacted = append(impacted, fu.ControlIDs...)
			impacted = append(impacted, fu.FlowIDs...)
			impacted = append(impacted, fu.FlowStepIDs...)
			impacted = append(impacted, fu.CodeIDs...)
			impacted = append(impacted, fu.VerificationIDs...)

			steps = append(steps, AIImplementationStep{
				Order:      stepOrder,
				Action:     "Implement or update behavior in functional unit scope and dependent interfaces/flows",
				EntityID:   fu.ID,
				EntityKind: fu.Kind,
				SourceRefs: fu.SourceRefs,
			})
			stepOrder++

			if len(fu.CodeIDs) > 0 {
				steps = append(steps, AIImplementationStep{
					Order:      stepOrder,
					Action:     "Update code evidence linked to this unit",
					EntityID:   fu.CodeIDs[0],
					EntityKind: "code_element",
					SourceRefs: sourceRefsForEntities(entityByID, fu.CodeIDs),
				})
				stepOrder++
			}
		}

		if len(req.VerificationIDs) > 0 {
			steps = append(steps, AIImplementationStep{
				Order:      stepOrder,
				Action:     "Update or add tests so verification stays passing for this requirement",
				EntityID:   req.VerificationIDs[0],
				EntityKind: "verification",
				SourceRefs: sourceRefsForEntities(entityByID, req.VerificationIDs),
			})
		} else {
			steps = append(steps, AIImplementationStep{
				Order:      stepOrder,
				Action:     "Add new verification coverage (tests/checks) for this requirement",
				EntityKind: "verification",
				SourceRefs: req.SourceRefs,
			})
		}

		priority := "medium"
		if strings.EqualFold(sp.Confidence, "low") || len(req.VerificationIDs) == 0 {
			priority = "high"
		}

		allRefs := []string{}
		allRefs = append(allRefs, req.SourceRefs...)
		allRefs = append(allRefs, sourceRefsForEntities(entityByID, impacted)...)
		paths = append(paths, AIImplementationPath{
			ID:                "IMPL-" + strings.TrimPrefix(req.ID, "REQ-"),
			RequirementID:     req.ID,
			Goal:              req.Summary,
			Priority:          priority,
			Confidence:        sp.Confidence,
			ImpactedEntityIDs: uniqueSorted(impacted),
			VerificationIDs:   uniqueSorted(req.VerificationIDs),
			Steps:             steps,
			SourceRefs:        uniqueSorted(allRefs),
		})
	}

	sort.SliceStable(paths, func(i, j int) bool {
		return paths[i].ID < paths[j].ID
	})
	return paths
}

func sourceRefsForEntities(entityByID map[string]AIEntity, ids []string) []string {
	refs := []string{}
	for _, id := range ids {
		if e, ok := entityByID[strings.TrimSpace(id)]; ok {
			refs = append(refs, e.SourceRefs...)
		}
	}
	return uniqueSorted(refs)
}

func parsePathAndLine(source string) (string, int) {
	s := filepath.ToSlash(strings.TrimSpace(source))
	if s == "" {
		return "", 0
	}
	parts := strings.Split(s, ":")
	if len(parts) >= 2 {
		last := strings.TrimSpace(parts[len(parts)-1])
		if n, err := strconv.Atoi(last); err == nil && n > 0 {
			return strings.Join(parts[:len(parts)-1], ":"), n
		}
	}
	return s, 0
}

func (ctx *aiBuildContext) addAuthoredYAMLSource(path, id, kind, summary, entityID string) string {
	line := findLineForYAMLID(path, id)
	if line == 0 {
		line = findLineContaining(path, id)
	}
	return ctx.addSourceBlock(kind, path, line, line, summary, []string{entityID})
}

func (ctx *aiBuildContext) addArtifactSource(source, kind, entityID, summary, hint string) string {
	path, line := parsePathAndLine(source)
	absPath, displayPath := ctx.resolveReadablePath(path)
	if line == 0 {
		line = findLineContaining(absPath, hint)
	}
	return ctx.addSourceBlock(kind, displayPath, line, line, summary, []string{entityID})
}

func (ctx *aiBuildContext) addSourceBlock(kind, path string, lineStart, lineEnd int, summary string, entityIDs []string) string {
	p := sanitizeSourcePath(path)
	if strings.TrimSpace(p) == "" {
		p = "none"
	}
	if lineStart > 0 && lineEnd == 0 {
		lineEnd = lineStart
	}
	key := strings.Join([]string{kind, p, strconv.Itoa(lineStart), strconv.Itoa(lineEnd), summary}, "|")
	if existing, ok := ctx.sourceBlocksByKey[key]; ok {
		existing.EntityIDs = uniqueSorted(append(existing.EntityIDs, entityIDs...))
		for _, entityID := range entityIDs {
			ctx.linkEntitySource(entityID, existing.ID)
		}
		return existing.ID
	}
	id := "SRC-" + strings.ToUpper(sanitizeNode(kind+"-"+p+"-"+strconv.Itoa(lineStart)+"-"+shortHash(key)))
	block := &AISourceBlock{
		ID:        id,
		Path:      p,
		LineStart: lineStart,
		LineEnd:   lineEnd,
		Kind:      kind,
		EntityIDs: uniqueSorted(entityIDs),
		Summary:   strings.TrimSpace(summary),
	}
	ctx.sourceBlocksByKey[key] = block
	for _, entityID := range entityIDs {
		ctx.linkEntitySource(entityID, id)
	}
	return id
}

func (ctx *aiBuildContext) linkEntitySource(entityID, sourceID string) {
	entityID = strings.TrimSpace(entityID)
	sourceID = strings.TrimSpace(sourceID)
	if entityID == "" || sourceID == "" {
		return
	}
	if ctx.sourceByEntityID[entityID] == nil {
		ctx.sourceByEntityID[entityID] = map[string]bool{}
	}
	ctx.sourceByEntityID[entityID][sourceID] = true
}

func (ctx *aiBuildContext) sourceRefsForEntityIDs(entityIDs []string) []string {
	out := []string{}
	for _, entityID := range entityIDs {
		for sourceID := range ctx.sourceByEntityID[strings.TrimSpace(entityID)] {
			out = append(out, sourceID)
		}
	}
	return uniqueSorted(out)
}

func (ctx *aiBuildContext) finalizeSourceBlocks() []AISourceBlock {
	out := make([]AISourceBlock, 0, len(ctx.sourceBlocksByKey))
	for _, block := range ctx.sourceBlocksByKey {
		block.EntityIDs = uniqueSorted(block.EntityIDs)
		out = append(out, *block)
	}
	sort.SliceStable(out, func(i, j int) bool {
		if out[i].Path != out[j].Path {
			return out[i].Path < out[j].Path
		}
		if out[i].LineStart != out[j].LineStart {
			return out[i].LineStart < out[j].LineStart
		}
		return out[i].ID < out[j].ID
	})
	return out
}

func (ctx *aiBuildContext) resolveReadablePath(path string) (string, string) {
	p := strings.TrimSpace(path)
	if p == "" {
		return "", "none"
	}
	if filepath.IsAbs(p) {
		return p, p
	}
	baseDir := filepath.Dir(ctx.bundle.ArchitecturePath)
	candidates := []string{filepath.Join(baseDir, p)}
	for _, root := range ctx.codeRoots {
		candidates = append(candidates, filepath.Join(root, p))
	}
	for _, cand := range candidates {
		if info, err := os.Stat(cand); err == nil && !info.IsDir() {
			return cand, cand
		}
	}
	return p, p
}

func findLineForYAMLID(path, id string) int {
	id = strings.TrimSpace(id)
	if id == "" {
		return 0
	}
	lines := readFileLines(path)
	needle := "id: " + id
	for idx, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == needle || strings.Trim(trimmed, "\"'") == needle {
			return idx + 1
		}
		if strings.Contains(trimmed, "id:") && strings.Contains(trimmed, id) {
			return idx + 1
		}
	}
	return 0
}

func findLineContaining(path, token string) int {
	token = strings.TrimSpace(token)
	if token == "" {
		return 0
	}
	lines := readFileLines(path)
	for idx, line := range lines {
		if strings.Contains(line, token) {
			return idx + 1
		}
	}
	return 0
}

func readFileLines(path string) []string {
	if strings.TrimSpace(path) == "" {
		return nil
	}
	b, err := os.ReadFile(path)
	if err != nil {
		return nil
	}
	return strings.Split(string(b), "\n")
}
