// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package asciidoc

import (
	"cmp"
	"fmt"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/generate"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/render/diagramstyle"
	"github.com/labeth/engineering-model-go/internal/reqlint"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/trace"
	"github.com/labeth/engineering-model-go/internal/validate"
	"github.com/labeth/engineering-model-go/internal/view"
)

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type Options struct {
	ViewIDs          []string
	CodeRoot         string
	DecisionsDocPath string
	// Resolution overrides ENGMOD_WORK / ENGMOD_LOCK for the composition run.
	Resolution composition.ResolutionOptions
	// Composition, when set, is a composition already resolved for this
	// bundle's manifest; Generate then does not compose again.
	Composition *composition.CompositionResult
	// UsedBy adds the "Used By" section to the Contract Items chapter: the
	// requirement links of the other checkouts of the active workspace onto
	// this model's items (D16). Off by default, also when a workspace is
	// found, so a committed render of a child never names its users; a
	// document rendered with it must not be committed into the child.
	UsedBy bool
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type Result struct {
	Document          string
	DecisionsDocument string
	Diagnostics       []validate.Diagnostic
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014
func GenerateFromFiles(architecturePath, requirementsPath, designPath string, options Options) (Result, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return Result{}, err
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return Result{}, err
	}
	design, err := model.LoadDesign(designPath)
	if err != nil {
		return Result{}, err
	}
	if strings.TrimSpace(options.CodeRoot) != "" && !filepath.IsAbs(options.CodeRoot) {
		baseDir := filepath.Dir(architecturePath)
		options.CodeRoot = filepath.Join(baseDir, options.CodeRoot)
	}
	return Generate(bundle, requirements, design, options)
}

// traceFindingCodes are the requirement link trace findings that strict
// mode makes errors; like link.*, they do not stop the render (D15).
var traceFindingCodes = map[string]bool{
	"requirement.no_parent": true, "requirement.unsatisfied": true, "contract.realizer_unpublished": true,
}

// hasBlockingErrors reports an error that stops the render. Requirement link
// findings (link.* and the strict-mode trace findings requirement.no_parent,
// requirement.unsatisfied and contract.realizer_unpublished) are errors of
// the model check but do not stop the render: the document shows the
// marked links and the trace a reviewer needs (D15). With lifecycle, an
// error the tier policy sets (validate.RendersDespite) does not stop it
// either: only rank-0 and untiered errors do (D15 extended, ADR-EMG-032).
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-130
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE
func hasBlockingErrors(bundle model.Bundle, diags []validate.Diagnostic) bool {
	for _, d := range diags {
		if d.Severity == validate.SeverityError && blocksRender(bundle, d.Code) {
			return true
		}
	}
	return false
}

// blocksRender reports whether an error with the code stops the render.
// TRLC-LINKS: REQ-EMG-107, REQ-EMG-130
func blocksRender(bundle model.Bundle, code string) bool {
	return !strings.HasPrefix(code, "link.") && !traceFindingCodes[code] && !validate.RendersDespite(bundle, code)
}

// validationBundle enriches the bundle with the dependencies' published
// architecture, behavior, assurance and compliance entities, so
// alias::ID references validate. Requirements are not imported. It returns the
// composition diagnostics to report: those of the resolution, or, without
// composition, an unpublished_reference for every alias::ID reference.
// TRLC-LINKS: REQ-EMG-065, REQ-EMG-066
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func validationBundle(bundle model.Bundle, composed composition.CompositionResult, compositionErr error) (model.Bundle, []validate.Diagnostic) {
	if !composition.HasComposition(bundle) {
		return bundle, composition.QualifiedReferenceDiagnostics(bundle, composition.CompositionResult{})
	}
	if compositionErr != nil {
		return bundle, []validate.Diagnostic{{
			Code: "composition.resolve_failed", Severity: validate.SeverityError,
			Message: fmt.Sprintf("composition could not be resolved: %v", compositionErr),
			Path:    bundle.ManifestPath,
		}}
	}
	projection := composition.BuildCompositionProjection(composed)
	enriched := projection.EnrichBundle(bundle, "architecture", "behavior", "assurance", "compliance")
	return enriched, composed.Diagnostics
}

// excludeUnpublishedReferenceEchoes drops the generic validation errors that
// repeat a composition.unpublished_reference (the same alias::ID reported as a
// broken reference, unknown flow target, unknown owner, ...), so each
// unpublished reference is reported once with its field path.
// TRLC-LINKS: REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-SYSTEM-COMPOSITION
func excludeUnpublishedReferenceEchoes(diags, compositionDiags []validate.Diagnostic) []validate.Diagnostic {
	unpublished := unpublishedReferences(compositionDiags)
	if len(unpublished) == 0 {
		return diags
	}
	out := diags[:0:0]
	for _, d := range diags {
		if !echoesUnpublishedReference(d, unpublished) {
			out = append(out, d)
		}
	}
	return out
}

// unpublishedReferences is the set of alias::ID references that the
// composition diagnostics report as unpublished.
// TRLC-LINKS: REQ-EMG-065
func unpublishedReferences(compositionDiags []validate.Diagnostic) map[string]bool {
	unpublished := map[string]bool{}
	for _, d := range compositionDiags {
		if d.Code != "composition.unpublished_reference" {
			continue
		}
		if ref, ok := firstQuoted(d.Message); ok {
			unpublished[ref] = true
		}
	}
	return unpublished
}

// echoesUnpublishedReference reports whether a generic validation error
// repeats one of the unpublished references.
// TRLC-LINKS: REQ-EMG-065
func echoesUnpublishedReference(d validate.Diagnostic, unpublished map[string]bool) bool {
	if d.Severity != validate.SeverityError || strings.HasPrefix(d.Code, "composition.") {
		return false
	}
	for ref := range unpublished {
		if strings.Contains(d.Message, strconv.Quote(ref)) {
			return true
		}
	}
	return false
}

// firstQuoted returns the first Go-quoted string in a diagnostic message.
// TRLC-LINKS: REQ-EMG-065
func firstQuoted(message string) (string, bool) {
	start := strings.Index(message, `"`)
	if start < 0 {
		return "", false
	}
	for end := start + 1; end < len(message); end++ {
		if message[end] == '"' && message[end-1] != '\\' {
			value, err := strconv.Unquote(message[start : end+1])
			return value, err == nil
		}
	}
	return "", false
}

// generation carries what one Generate call threads through its stages: the
// inputs, the validated bundle, the diagnostics collected so far and the
// sections that feed the template.
type generation struct {
	bundle           model.Bundle
	validated        model.Bundle
	requirements     model.RequirementsDocument
	design           model.DesignDocument
	options          Options
	composed         composition.CompositionResult
	compositionErr   error
	compositionDiags []validate.Diagnostic
	diags            []validate.Diagnostic

	viewByID    map[string]model.View
	viewNodeIDs map[string]map[string]bool
	views       []asciidocViewSection

	inferredRuntime      []codescan.RuntimeItem
	inferredCode         []codescan.CodeItem
	inferredVerification []codescan.VerificationCheck
	evidenceByOwner      map[string]string

	labelByID      map[string]string
	reqByUnit      map[string]string
	attackByTarget map[string]string
	groups         []asciidocEntitySection
	units          []asciidocUnitSection

	linkTargets  map[string]linkTarget
	terms        []asciidocTerm
	reqSections  []asciidocRequirementSection
	verification []asciidocVerificationSection
	verResults   []asciidocVerificationResultRow
	refIndex     asciidocReferenceIndex
	decisions    []asciidocDecisionSection
}

// Generate validates and renders the model. With composition it
// resolves the dependencies once, validates against the bundle enriched with
// their published entities, and returns every composition diagnostic; a
// composition error fails the call like any other validation error.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func Generate(bundle model.Bundle, requirements model.RequirementsDocument, design model.DesignDocument, options Options) (Result, error) {
	g := &generation{bundle: bundle, requirements: requirements, design: design, options: options}
	if err := g.canonicalize(); err != nil {
		return Result{}, err
	}
	if err := g.validateModel(); err != nil {
		return g.fail(err)
	}
	if err := g.projectViews(); err != nil {
		return g.fail(err)
	}
	g.inferRealization()
	g.buildEntitySections()
	g.fillViews()
	g.buildDocumentSections()
	g.linkifySections()
	data := g.templateData()
	doc, err := g.renderDocument(data)
	if err != nil {
		return g.fail(err)
	}
	return Result{Document: doc, DecisionsDocument: renderDecisionsDocument(g.decisions), Diagnostics: validate.SortDiagnostics(g.diags)}, nil
}

// fail returns the diagnostics collected so far with the error.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) fail(err error) (Result, error) {
	return Result{Diagnostics: validate.SortDiagnostics(g.diags)}, err
}

// authored is the authored architecture of the model under generation.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) authored() model.AuthoredArchitecture {
	return g.bundle.Architecture.AuthoredArchitecture
}

// canonicalize builds the canonical bundle with its code index.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) canonicalize() error {
	g.bundle.Requirements = g.requirements
	g.bundle.Design = g.design
	canonical, err := model.NewCanonicalBundle(g.bundle)
	if err != nil {
		return err
	}
	g.bundle = codescan.WithCodeIndex(canonical.Documents())
	g.requirements = g.bundle.Requirements
	g.design = g.bundle.Design
	return nil
}

// validateModel composes, validates and lints; it fails when an error stops
// the render.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) validateModel() error {
	g.composed, g.compositionErr = composition.ComposeForGeneration(g.bundle, g.options.Composition, g.options.Resolution)
	g.validated, g.compositionDiags = validationBundle(g.bundle, g.composed, g.compositionErr)
	diags := excludeUnpublishedReferenceEchoes(validate.Bundle(g.validated), g.compositionDiags)
	diags = append(diags, g.compositionDiags...)
	diags = append(diags, reqlint.ValidateCatalogDescriptions(g.bundle.Catalog)...)
	diags = append(diags, reqlint.LintModelRequirementsEARS(g.bundle, g.requirements)...)
	diags = append(diags, reqlint.LintRequirementInternalLinks(g.requirements)...)
	g.diags = validate.ApplyTierPolicy(g.bundle, diags)
	if hasBlockingErrors(g.bundle, g.diags) {
		return fmt.Errorf("validation failed")
	}
	return nil
}

// projectViews projects every selected view into a view section and records
// its node IDs.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) projectViews() error {
	viewIDs := resolveViewIDs(g.bundle, g.options)
	g.viewByID = map[string]model.View{}
	for _, v := range g.bundle.Architecture.Views {
		g.viewByID[v.ID] = v
	}
	kindCounts := map[string]int{}
	for _, viewID := range viewIDs {
		kindCounts[g.viewByID[viewID].Kind]++
	}
	g.viewNodeIDs = map[string]map[string]bool{}
	g.views = make([]asciidocViewSection, 0, len(viewIDs))
	for _, viewID := range viewIDs {
		res, err := generate.Generate(g.validated, viewID)
		g.diags = append(g.diags, excludeUnpublishedReferenceEchoes(res.Diagnostics, g.compositionDiags)...)
		if err != nil {
			return fmt.Errorf("generate view %s: %w", viewID, err)
		}
		g.views = append(g.views, g.viewSection(viewID, res, kindCounts[res.View.Kind]))
		g.viewNodeIDs[viewID] = projectedNodeIDs(res.View.Nodes)
	}
	return nil
}

// viewSection is the section of one projected view.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) viewSection(viewID string, res generate.Result, kindCount int) asciidocViewSection {
	cfg := g.viewByID[viewID]
	graph, edgeKey := publicationEdgeKey(res.View.Kind, strings.TrimSpace(res.Mermaid))
	return asciidocViewSection{
		ID:                        viewID,
		Kind:                      res.View.Kind,
		Heading:                   publicationViewHeading(res.View.Kind, viewID, kindCount),
		AuthoredStatus:            normalizeAuthoredStatus(cfg.AuthoredStatus),
		AuthoredStatusExplanation: normalizeAuthoredStatusExplanation(cfg.AuthoredStatusExplanation),
		Mermaid:                   graph,
		EdgeKey:                   edgeKey,
		Inf:                       inferredDescription(res.View.Kind),
		ViewQuestions:             viewQuestions(res.View.Kind),
		ProjectedNodes:            sortedProjectedNodes(res.View.Nodes),
		ProjectedMappings:         sortedProjectedMappings(res.View.Edges),
	}
}

// projectedNodeIDs is the set of node IDs of a projected view.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func projectedNodeIDs(nodes []view.Node) map[string]bool {
	ids := map[string]bool{}
	for _, n := range nodes {
		ids[n.ID] = true
	}
	return ids
}

// sortedProjectedNodes lists the nodes by kind, then ID.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func sortedProjectedNodes(nodes []view.Node) []asciidocProjectedNode {
	out := make([]asciidocProjectedNode, 0, len(nodes))
	for _, n := range nodes {
		out = append(out, asciidocProjectedNode{ID: strings.TrimSpace(n.ID), Kind: strings.TrimSpace(n.Kind), Label: strings.TrimSpace(n.Label)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return cmp.Or(cmp.Compare(out[i].Kind, out[j].Kind), cmp.Compare(out[i].ID, out[j].ID)) < 0
	})
	return out
}

// sortedProjectedMappings lists the edges by type, then source, then target.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func sortedProjectedMappings(edges []view.Edge) []asciidocMappingSection {
	out := make([]asciidocMappingSection, 0, len(edges))
	for _, e := range edges {
		out = append(out, asciidocMappingSection{Type: strings.TrimSpace(e.Type), From: strings.TrimSpace(e.From), To: strings.TrimSpace(e.To), Description: strings.TrimSpace(e.Label)})
	}
	sort.SliceStable(out, func(i, j int) bool {
		return cmp.Or(cmp.Compare(out[i].Type, out[j].Type), cmp.Compare(out[i].From, out[j].From), cmp.Compare(out[i].To, out[j].To)) < 0
	})
	return out
}

// inferRealization infers the runtime, code and verification layers and
// checks the trace integrity against them.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) inferRealization() {
	runtime, runtimeDiags := codescan.InferRuntimeItems(g.bundle)
	code, codeDiags := codescan.InferCodeItems(g.bundle, g.options.CodeRoot)
	verification, verificationDiags := codescan.InferVerificationChecks(g.bundle, g.requirements, code, g.options.CodeRoot)
	g.inferredRuntime, g.inferredCode, g.inferredVerification = runtime, code, verification
	g.diags = append(g.diags, runtimeDiags...)
	g.diags = append(g.diags, codeDiags...)
	g.diags = append(g.diags, verificationDiags...)
	g.evidenceByOwner = buildOwnerEvidence(runtime, code)
	scopedCode := trace.ScopeCodeToModel(code, trace.EffectiveCodeRoots(g.bundle, g.options.CodeRoot), codescan.ModelRootDir(g.bundle))
	g.diags = append(g.diags, trace.ValidateIntegrity(g.validated, g.requirements, scopedCode, verification)...)
	g.diags = validate.ApplyTierPolicy(g.bundle, g.diags)
}

// buildEntitySections builds the functional group and unit sections.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) buildEntitySections() {
	a := g.authored()
	g.labelByID = buildLabelIndex(a)
	g.reqByUnit = requirementsByUnit(g.requirements.Requirements)
	g.attackByTarget = attackVectorsByTarget(a.Mappings, g.labelByID)
	g.groups = g.groupSections()
	g.units = g.unitSections()
}

// groupSections builds the section of every functional group.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) groupSections() []asciidocEntitySection {
	designs := mapDesignGroups(g.design)
	groups := g.authored().FunctionalGroups
	out := make([]asciidocEntitySection, 0, len(groups))
	for _, fg := range groups {
		out = append(out, asciidocEntitySection{
			Anchor:      referenceAnchor("idx-fg", fg.ID),
			ID:          fg.ID,
			Name:        textutil.NonEmpty(fg.Name, fg.ID),
			Description: strings.TrimSpace(fg.Description),
			Tags:        strings.Join(fg.Tags, ", "),
			Intro:       strings.TrimSpace(fg.Prose),
			Details:     buildDesignDetails(fg.Prose, designs[fg.ID].Views, g.bundle.Architecture.Views),
		})
	}
	return out
}

// unitSections builds the section of every functional unit, by group and ID.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) unitSections() []asciidocUnitSection {
	designs := mapDesignUnits(g.design)
	units := g.authored().FunctionalUnits
	out := make([]asciidocUnitSection, 0, len(units))
	for _, u := range units {
		out = append(out, g.unitSection(u, designs[u.ID]))
	}
	sort.SliceStable(out, func(i, j int) bool {
		return cmp.Or(cmp.Compare(out[i].Group, out[j].Group), cmp.Compare(out[i].ID, out[j].ID)) < 0
	})
	return out
}

// unitSection builds the section of one functional unit.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) unitSection(u model.FunctionalUnit, design model.DesignFunctionalUnit) asciidocUnitSection {
	maps := g.authored().Mappings
	return asciidocUnitSection{
		Anchor:      referenceAnchor("idx-fu", u.ID),
		GroupAnchor: referenceAnchor("idx-fg", u.Group),
		ID:          u.ID,
		Name:        textutil.NonEmpty(u.Name, u.ID),
		Group:       u.Group,
		Tags:        strings.Join(u.Tags, ", "),
		Intro:       strings.TrimSpace(u.Prose),
		Details:     buildDesignDetails(u.Prose, design.Views, g.bundle.Architecture.Views),
		WhatOwns:    unitOwnershipSummary(u, maps, g.labelByID),
		Triggers:    textutil.NonEmpty(g.reqByUnit[u.ID], "no explicit authored requirement trigger"),
		Consumes:    textutil.NonEmpty(unitConsumers(u.ID, maps, g.labelByID), "none"),
		Produces:    textutil.NonEmpty(unitOutputs(u.ID, maps, g.labelByID), "none"),
		DependsOn:   unitDependencies(u.ID, maps, g.labelByID),
		Threats:     textutil.NonEmpty(g.attackByTarget[u.ID], "no explicit attack vector targeting this unit"),
		Evidence:    textutil.NonEmpty(g.evidenceByOwner[u.ID], "authored unit with no direct derived runtime/code evidence yet"),
	}
}

// scopedGroups is the groups that appear in a view, with the detail of that view.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func scopedGroups(groups []asciidocEntitySection, nodeSet map[string]bool, viewID string) []asciidocEntitySection {
	out := make([]asciidocEntitySection, 0, len(groups))
	for _, fg := range groups {
		if !nodeSet[fg.ID] {
			continue
		}
		scoped := fg
		scoped.Details = []asciidocDesignDetail{detailForView(fg.Details, viewID)}
		out = append(out, scoped)
	}
	return out
}

// scopedUnits is the units that appear in a view, with the detail of that view.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func scopedUnits(units []asciidocUnitSection, nodeSet map[string]bool, viewID string) []asciidocUnitSection {
	out := make([]asciidocUnitSection, 0, len(units))
	for _, u := range units {
		if !nodeSet[u.ID] {
			continue
		}
		scoped := u
		scoped.Details = []asciidocDesignDetail{detailForView(u.Details, viewID)}
		out = append(out, scoped)
	}
	return out
}

// fillViews scopes the entities to each view and builds its kind specific
// diagrams and tables.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) fillViews() {
	for i := range g.views {
		g.fillView(&g.views[i])
	}
}

// viewFillers builds the kind specific content of a view section.
var viewFillers = map[string]func(*generation, *asciidocViewSection){
	"architecture-intent": (*generation).fillArchitectureIntent,
	"communication":       (*generation).fillCommunication,
	"deployment":          (*generation).fillDeployment,
	"security":            (*generation).fillSecurity,
	"traceability":        (*generation).fillTraceability,
}

// fillView scopes the entities to one view and builds its coverage and its
// kind specific content.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) fillView(vs *asciidocViewSection) {
	cfg := g.viewByID[vs.ID]
	kind := cfg.Kind
	nodeSet := g.viewNodeIDs[vs.ID]
	vs.Groups = scopedGroups(g.groups, nodeSet, cfg.ID)
	vs.Units = scopedUnits(g.units, nodeSet, cfg.ID)
	if kind == "communication" {
		g.describeCommunicationUnits(vs.Units)
	}
	vs.CoverageGaps = viewCoverageGaps(kind, vs.Units)
	vs.CoverageSummary = viewCoverageSummary(kind, vs.Units)
	vs.NextActions = viewNextActions(kind, vs.CoverageGaps)
	if fill := viewFillers[kind]; fill != nil {
		fill(g, vs)
	}
}

// describeCommunicationUnits replaces the unit flows of a communication view
// with its interfaces and messages.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) describeCommunicationUnits(units []asciidocUnitSection) {
	maps := g.authored().Mappings
	for j := range units {
		units[j].Consumes = unitInboundInterfacesDetailed(units[j].ID, maps, g.labelByID)
		units[j].Produces = unitOutboundInterfacesDetailed(units[j].ID, maps, g.labelByID)
		units[j].Triggers = unitMessagesConsumed(units[j].ID, maps, g.labelByID)
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) fillArchitectureIntent(vs *asciidocViewSection) {
	scoped := functionalPublicationArchitecture(g.authored(), g.viewNodeIDs[vs.ID])
	if len(scoped.FunctionalUnits) > 0 || len(scoped.FunctionalGroups) > 0 {
		vs.FuncContextGraph = buildFunctionalContextMermaid(scoped)
		vs.FuncDecompGraph = buildFunctionalDecompositionMermaid(scoped)
		vs.FuncMatrixTable = buildFunctionalManhattanTable(scoped)
	}
	for j := range vs.Groups {
		vs.Groups[j].DependencyGraph = buildFunctionalGroupDependencyMermaid(scoped, vs.Groups[j].ID, realizationEvidence{g.inferredRuntime, g.inferredCode})
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) fillCommunication(vs *asciidocViewSection) {
	apiRows := buildRuntimeAPIRows(g.inferredRuntime, g.authored().Mappings)
	vs.RuntimeAPIRows = apiRows
	vs.RuntimeAPIGraph = buildRuntimeAPIMermaid(apiRows)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) fillDeployment(vs *asciidocViewSection) {
	depRows := buildDeploymentRows(g.inferredRuntime)
	vs.DeploymentRows = depRows
	vs.DeploymentGraph = buildDeploymentMermaid(depRows)
	opRows := buildPlatformOpsRows(g.authored(), g.inferredRuntime)
	vs.PlatformOpsRows = opRows
	vs.PlatformOpsGraph = buildPlatformOpsMermaid(opRows)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) fillSecurity(vs *asciidocViewSection) {
	a := g.authored()
	nodeSet := g.viewNodeIDs[vs.ID]
	secRows := buildSecurityPathRows(a, g.labelByID)
	vs.SecurityRows = secRows
	vs.SecurityThreatScenarios = buildSecurityThreatScenarioRows(a, nodeSet, g.labelByID)
	vs.SecurityThreatAssumptions = buildSecurityThreatAssumptionRows(a, nodeSet, g.labelByID)
	vs.SecurityThreatOutOfScope = buildSecurityThreatOutOfScopeRows(a, nodeSet, g.labelByID)
	vs.SecurityThreatMitigations = buildSecurityThreatMitigationRows(a, nodeSet, g.labelByID)
	vs.SecurityControlChecks = buildSecurityControlVerificationRows(a, nodeSet, g.labelByID)
	vs.SecurityFlowRows = buildSecurityFlowRows(a, nodeSet, g.labelByID)
	vs.SecurityGraph = buildSecurityPathMermaid(secRows, g.inferredRuntime, g.inferredCode)
	vs.SecurityContextDiagrams = buildSecurityContextDFDMermaidByGroup(a, g.bundle.Architecture.Compliance.Mappings, g.labelByID)
	vs.SecurityObsRows = buildSecurityObservabilityRows(g.inferredRuntime, g.inferredCode)
	vs.SecurityAttackChapters = buildSecurityAttackChapters(a, vs.Units, nodeSet, secRows, g.labelByID, g.inferredRuntime, g.inferredCode)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) fillTraceability(vs *asciidocViewSection) {
	codeRows := buildCodeOwnershipRows(g.inferredCode)
	vs.InferredGraph = buildCodeOwnershipMermaid(codeRows, g.authored())
	vs.InferredRows = codeRows
}

// buildDocumentSections builds the requirement, verification, reference
// index, term and decision sections.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) buildDocumentSections() {
	g.reqSections = g.requirementSections()
	g.verification, g.verResults = g.verificationSections()
	g.refIndex = buildReferenceIndex(referenceSources{g.bundle, g.requirements, g.inferredRuntime, g.inferredCode, g.inferredVerification})
	g.linkTargets = buildLinkTargets(g.refIndex)
	g.terms = buildTermsFromCatalog(g.bundle.Catalog)
	g.decisions = buildDecisionSections(g.bundle.Architecture.Decisions)
	sort.SliceStable(g.terms, func(i, j int) bool {
		left := strings.ToLower(strings.TrimSpace(g.terms[i].Name))
		right := strings.ToLower(strings.TrimSpace(g.terms[j].Name))
		return cmp.Or(cmp.Compare(left, right), cmp.Compare(g.terms[i].ID, g.terms[j].ID)) < 0
	})
}

// requirementSections builds the section of every requirement, by ID.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) requirementSections() []asciidocRequirementSection {
	reqs := g.requirements.Requirements
	out := make([]asciidocRequirementSection, 0, len(reqs))
	for _, r := range reqs {
		out = append(out, asciidocRequirementSection{
			Anchor:               referenceAnchor("req", r.ID),
			ID:                   r.ID,
			Text:                 strings.TrimSpace(r.Text),
			Notes:                strings.TrimSpace(r.Notes),
			AlignmentMermaid:     buildRequirementAlignmentMermaid([]model.Requirement{r}, g.labelByID),
			CoverageMermaid:      buildRequirementCoverageMermaid([]model.Requirement{r}, coverageSources{g.inferredRuntime, g.inferredCode, g.inferredVerification, g.labelByID}),
			AlignmentExplanation: "What this diagram shows: direct authored mapping from this requirement to the functional units it applies to.",
			CoverageExplanation:  "What this diagram shows: this requirement-to-unit mapping extended with inferred runtime/code plus verification evidence attached to each requirement and unit.",
		})
	}
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}

// verificationSections builds the section and the result rows of every
// inferred verification check.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) verificationSections() ([]asciidocVerificationSection, []asciidocVerificationResultRow) {
	sections := make([]asciidocVerificationSection, 0, len(g.inferredVerification))
	rows := make([]asciidocVerificationResultRow, 0)
	for _, v := range g.inferredVerification {
		sections = append(sections, verificationSection(v))
		rows = append(rows, verificationResultRows(v)...)
	}
	sort.SliceStable(sections, func(i, j int) bool {
		return cmp.Or(cmp.Compare(sections[i].ID, sections[j].ID), cmp.Compare(sections[i].Name, sections[j].Name)) < 0
	})
	sort.SliceStable(rows, func(i, j int) bool {
		return cmp.Or(cmp.Compare(rows[i].Requirement, rows[j].Requirement), cmp.Compare(rows[i].CheckID, rows[j].CheckID), cmp.Compare(rows[i].Status, rows[j].Status)) < 0
	})
	return sections, rows
}

// verificationSection is the section of one verification check.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func verificationSection(v codescan.VerificationCheck) asciidocVerificationSection {
	return asciidocVerificationSection{
		Anchor:        referenceAnchor("verify", strings.TrimSpace(v.ID)),
		IndexAnchor:   referenceAnchor("idx-ver", strings.TrimSpace(v.ID)),
		ID:            strings.TrimSpace(v.ID),
		Name:          strings.TrimSpace(v.Name),
		Kind:          strings.TrimSpace(v.Kind),
		Status:        strings.TrimSpace(v.Status),
		Verifies:      joinList(v.Verifies),
		TestCode:      publicationTestCode(v.CodeElements),
		DerivedOwners: joinList(v.DerivedOwners),
		Evidence:      joinList(v.Evidence),
		ResultSummary: summarizeVerificationResults(v.Results),
		Description:   strings.TrimSpace(v.Description),
	}
}

// verificationResultRows is the result rows of one verification check.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func verificationResultRows(v codescan.VerificationCheck) []asciidocVerificationResultRow {
	rows := make([]asciidocVerificationResultRow, 0, len(v.Results))
	for _, result := range v.Results {
		rows = append(rows, asciidocVerificationResultRow{
			CheckID:     strings.TrimSpace(v.ID),
			CheckName:   textutil.NonEmpty(strings.TrimSpace(v.Name), strings.TrimSpace(v.ID)),
			Requirement: strings.TrimSpace(result.Requirement),
			Status:      strings.TrimSpace(result.Status),
			Evidence:    textutil.NonEmpty(strings.TrimSpace(result.Evidence), "none"),
			Notes:       strings.TrimSpace(result.Notes),
		})
	}
	return rows
}

// joinList joins the non-blank items with commas, or says none.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func joinList(items []string) string {
	parts := make([]string, 0, len(items))
	for _, x := range items {
		if t := strings.TrimSpace(x); t != "" {
			parts = append(parts, t)
		}
	}
	if len(parts) == 0 {
		return "none"
	}
	return strings.Join(parts, ", ")
}

// summarizeVerificationResults counts the results by status, sorted.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func summarizeVerificationResults(results []codescan.VerificationResult) string {
	if len(results) == 0 {
		return "none"
	}
	counts := map[string]int{}
	for _, r := range results {
		counts[textutil.NonEmpty(strings.ToLower(strings.TrimSpace(r.Status)), "unknown")]++
	}
	parts := make([]string, 0, len(counts))
	for status, count := range counts {
		parts = append(parts, fmt.Sprintf("%s:%d", status, count))
	}
	sort.Strings(parts)
	return strings.Join(parts, ", ")
}

// templateData assembles the data the AsciiDoc template renders.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) templateData() asciidocTemplateData {
	data := g.documentHeader()
	g.addAuthoredContent(&data)
	data.MermaidClassDefs = diagramstyle.MermaidClassDefsBlock("  ")
	data.Views = g.views
	data.RequirementMermaid = buildRequirementAlignmentCompactTable(g.requirements.Requirements)
	data.RequirementInf = "Show requirement-to-unit mappings inferred from appliesTo and authored architecture ownership boundaries."
	data.Requirements = g.reqSections
	data.Verifications = g.verification
	data.VerificationResults = g.verResults
	data.ReferenceIndex = g.refIndex
	data.Decisions = g.decisions
	data.DecisionsDocPath = textutil.NonEmpty(strings.TrimSpace(g.options.DecisionsDocPath), "DECISIONS.adoc")
	return data
}

// documentHeader is the front matter of the document.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) documentHeader() asciidocTemplateData {
	arch := g.bundle.Architecture
	lint := g.requirements.LintRun
	return asciidocTemplateData{
		Title:        textutil.NonEmpty(g.design.Design.Title, arch.Model.Title),
		Introduction: linkifyText(strings.TrimSpace(arch.Model.Introduction), g.linkTargets),
		HealthRows:   buildHealthRows(g.views),
		Terms:        g.terms,
		Purpose:      "This architecture description is generated from authored structure and inferred realization layers.",
		ReaderTracks: []string{"Product/domain engineers: Functional + Runtime", "Platform/SRE engineers: Deployment + Runtime", "Implementation engineers: Realization + Functional", "Security engineers: Security + Functional"},
		Legend:       []string{"Authored: intentional functional architecture", "Inferred: discovered runtime or code realization", "realizes: runtime to functional unit", "implemented_by: verification to test code", "verifies: verification to requirement"},
		ModelMeta:    asciidocModelMeta{ID: strings.TrimSpace(arch.Model.ID), Title: strings.TrimSpace(arch.Model.Title)},
		LintRun: asciidocLintRun{
			ID:         strings.TrimSpace(lint.ID),
			Mode:       strings.TrimSpace(lint.Mode),
			CommaAsAnd: lint.CommaAsAnd,
			CatalogRef: strings.TrimSpace(lint.CatalogRef),
		},
		ViewConfig: renderViewConfig(arch.Views),
		InferenceHints: asciidocInferenceHints{
			RuntimeSources:           strings.Join(arch.InferenceHints.RuntimeSources, ", "),
			ExpectedRuntimeKinds:     strings.Join(arch.InferenceHints.ExpectedRuntimeKinds, ", "),
			OwnershipResolutionOrder: strings.Join(arch.InferenceHints.OwnershipResolutionOrder, ", "),
		},
	}
}

// addAuthoredContent adds the authored lists, the inferred layers and the
// summary to the template data.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) addAuthoredContent(data *asciidocTemplateData) {
	a := g.authored()
	data.Actors = renderActors(a.Actors)
	data.AttackVectors = renderAttackVectors(a.AttackVectors)
	data.ReferencedElements = renderReferencedElements(a.ReferencedElements)
	data.Mappings = renderMappings(a.Mappings)
	data.InferredRuntime = renderInferredRuntime(g.inferredRuntime)
	data.InferredCode = renderInferredCode(g.inferredCode)
	data.Summary = asciidocSummary{
		FunctionalGroups:   listNamesFG(a.FunctionalGroups),
		FunctionalUnits:    listNamesFU(a.FunctionalUnits),
		Actors:             listNamesActors(a.Actors),
		AttackVectors:      listNamesVectors(a.AttackVectors),
		ReferencedElements: listNamesRefs(a.ReferencedElements),
	}
}

// renderDocument renders the template twice, the second time with the
// backlinks of the first.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) renderDocument(data asciidocTemplateData) (string, error) {
	doc, err := renderAsciiDocTemplate(data)
	if err != nil {
		return "", err
	}
	data.ReferenceIndex = applyReferenceBacklinks(doc, data.ReferenceIndex)
	doc, err = renderAsciiDocTemplate(data)
	if err != nil {
		return "", err
	}
	return g.insertTrailingChapters(doc), nil
}

// insertTrailingChapters adds the Gemara, Composition, requirement link and
// contract item chapters before the Reference Index, so that remains the
// closing chapter.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, DO-ARCHITECTURE-DOCUMENT, IF-GO-PKG-ASCIIDOC
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066, REQ-EMG-107, REQ-EMG-125, REQ-EMG-130
func (g *generation) insertTrailingChapters(doc string) string {
	trailing := renderGemaraAsciiDocChapter(g.bundle) + renderCompositionAsciiDocChapterFor(g.bundle, g.composed, g.compositionErr) +
		renderRequirementLinksChapter(g.bundle, g.composed) + renderContractItemsChapter(g.bundle, g.options.Resolution.Workspace, g.options.UsedBy)
	if trailing == "" {
		return doc
	}
	if idx := strings.Index(doc, "<<<\n== Reference Index"); idx >= 0 {
		return doc[:idx] + strings.TrimPrefix(trailing, "\n") + doc[idx:]
	}
	return doc + trailing
}
