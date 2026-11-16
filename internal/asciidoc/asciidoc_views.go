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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
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

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func renderInferredRuntime(in []codescan.RuntimeItem) []asciidocInferredRow {
	out := make([]asciidocInferredRow, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocInferredRow{Name: x.Name, Kind: x.Kind, Owner: x.Owner, Source: x.Source})
	}
	return out
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
// TRLC-LINKS: REQ-EMG-003
func renderInferredCode(in []codescan.CodeItem) []asciidocInferredRow {
	out := make([]asciidocInferredRow, 0, len(in))
	for _, x := range in {
		out = append(out, asciidocInferredRow{Name: codescan.ItemDisplayName(x), Kind: x.Kind, Owner: x.Owner, Source: codescan.SanitizeSourcePath(x.Source)})
	}
	return out
}

// viewHeadings are the headings of the view kinds.
var viewHeadings = map[string]string{
	"architecture-intent": "Architecture Intent View",
	"communication":       "Communication View",
	"deployment":          "Deployment View",
	"security":            "Security View",
	"traceability":        "Traceability View",
	"state-lifecycle":     "State Lifecycle View",
	"interaction-flow":    "Interaction Flow View",
}

// TRLC-LINKS: REQ-EMG-003
func viewHeading(kind string) string {
	if heading, ok := viewHeadings[kind]; ok {
		return heading
	}
	return strings.Title(kind) + " View"
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func resolveViewIDs(bundle model.Bundle, options Options) []string {
	if len(options.ViewIDs) > 0 {
		return append([]string(nil), options.ViewIDs...)
	}
	out := make([]string, 0, len(bundle.Architecture.Views))
	kindByID := map[string]string{}
	for _, v := range bundle.Architecture.Views {
		out = append(out, v.ID)
		kindByID[v.ID] = strings.TrimSpace(v.Kind)
	}
	order := map[string]int{
		"architecture-intent": 0,
		"communication":       1,
		"deployment":          2,
		"security":            3,
		"traceability":        4,
		"state-lifecycle":     5,
		"interaction-flow":    6,
	}
	sort.SliceStable(out, func(i, j int) bool {
		li, ok := order[kindByID[out[i]]]
		if !ok {
			li = 100
		}
		lj, ok := order[kindByID[out[j]]]
		if !ok {
			lj = 100
		}
		if li != lj {
			return li < lj
		}
		return out[i] < out[j]
	})
	return out
}

// inferredDescriptions say what a view of each kind shows.
var inferredDescriptions = map[string]string{
	"architecture-intent": "Show the stable authored architecture intent: capability areas, unit ownership boundaries, and intentional responsibility split.",
	"communication":       "Show who communicates with whom across key request paths, including interface edges and dependency handoffs.",
	"deployment":          "Show inferred deployment artifacts and ownership mapping to authored units. This view emphasizes deployment relationships and platform operations.",
	"security":            "Show the selected security relationships and boundary context. Interpret this projection within its authored scope; membership alone does not establish authentication, enforcement or complete threat coverage.",
	"traceability":        "Show requirement-to-unit-to-evidence traceability, including coverage confidence and explicit evidence gaps.",
	"state-lifecycle":     "Show workflow state transitions, ownership of transitions, and key exceptional paths.",
	"interaction-flow":    "Show user-intent-to-outcome causal flow, including ordered steps, data movement, and error/async branches.",
}

// TRLC-LINKS: REQ-EMG-003
func inferredDescription(kind string) string {
	return textutil.NonEmpty(inferredDescriptions[kind], "Show authored architecture scope for this view.")
}

// viewQuestionsByKind are the questions a view of each kind answers.
var viewQuestionsByKind = map[string][]string{
	"architecture-intent": {
		"What are the main capability areas?",
		"What are the stable units of responsibility and what does each unit own?",
		"How are responsibilities intentionally split?",
	},
	"communication": {
		"Who talks to whom and over which interfaces?",
		"Which edges are synchronous, asynchronous, event-driven, or callback-oriented?",
		"Which communication paths are critical to the main request flow?",
	},
	"deployment": {
		"How does authored intent become deployed runtime?",
		"What control-plane objects manage deployable units?",
		"Where are platform ownership boundaries and missing ownership signals?",
	},
	"traceability": {
		"Which requirements map to which units?",
		"Which authored units have runtime/code evidence and which do not?",
		"Where are confidence or completeness gaps in trace coverage?",
	},
	"security": {
		"What are the main trust boundaries and attack paths?",
		"Which units are exposed and what security evidence exists?",
		"Where is the threat model incomplete?",
	},
	"state-lifecycle": {
		"What states exist and which events trigger transitions?",
		"Which unit owns each state transition?",
		"Where do automatic and manual flows diverge?",
	},
	"interaction-flow": {
		"What happens from user intent to system outcome, step by step?",
		"What data enters/exits each step and where are decisions made?",
		"Where are async and error paths and what completes each path?",
	},
}

// TRLC-LINKS: REQ-EMG-003
func viewQuestions(kind string) []string {
	if questions, ok := viewQuestionsByKind[kind]; ok {
		return append([]string(nil), questions...)
	}
	return []string{"What does this view show?"}
}

// unitGapRules say which evidence gaps a unit has in a view of each kind; a
// rule returns the gap, or empty when the unit has none.
var unitGapRules = map[string][]func(asciidocUnitSection) string{
	"communication": {runtimeDeploymentGap},
	"deployment":    {runtimeDeploymentGap},
	"traceability":  {codeEvidenceGap, runtimeEvidenceGap},
	"security":      {attackVectorGap},
}

// evidenceText is the evidence of a unit, trimmed and lower-cased.
// TRLC-LINKS: REQ-EMG-003
func evidenceText(u asciidocUnitSection) string {
	return strings.ToLower(strings.TrimSpace(u.Evidence))
}

// TRLC-LINKS: REQ-EMG-003
func runtimeDeploymentGap(u asciidocUnitSection) string {
	if strings.Contains(evidenceText(u), "runtime:") {
		return ""
	}
	return fmt.Sprintf("%s has no direct inferred runtime/deployment evidence yet.", u.Name)
}

// TRLC-LINKS: REQ-EMG-003
func codeEvidenceGap(u asciidocUnitSection) string {
	if strings.Contains(evidenceText(u), "code modules:") {
		return ""
	}
	return fmt.Sprintf("%s has no direct inferred code evidence yet.", u.Name)
}

// TRLC-LINKS: REQ-EMG-003
func runtimeEvidenceGap(u asciidocUnitSection) string {
	if strings.Contains(evidenceText(u), "runtime:") {
		return ""
	}
	return fmt.Sprintf("%s has no direct inferred runtime evidence yet.", u.Name)
}

// TRLC-LINKS: REQ-EMG-003
func attackVectorGap(u asciidocUnitSection) string {
	if !strings.Contains(strings.ToLower(u.Threats), "no explicit attack vector") {
		return ""
	}
	return fmt.Sprintf("%s has no explicit authored attack-vector mapping yet; document technical and fraud/abuse threats even when mitigated.", u.Name)
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func viewCoverageGaps(kind string, units []asciidocUnitSection) []string {
	gaps := []string{}
	for _, u := range units {
		gaps = append(gaps, unitGaps(kind, u)...)
	}
	if len(gaps) == 0 {
		return []string{"No major coverage gaps detected from current authored and inferred inputs."}
	}
	sort.Strings(gaps)
	return gaps
}

// unitGaps lists the evidence gaps of a unit in a view of the kind.
// TRLC-LINKS: REQ-EMG-003
func unitGaps(kind string, u asciidocUnitSection) []string {
	var gaps []string
	for _, rule := range unitGapRules[kind] {
		if gap := rule(u); gap != "" {
			gaps = append(gaps, gap)
		}
	}
	return gaps
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func viewCoverageSummary(kind string, units []asciidocUnitSection) string {
	if len(units) == 0 {
		return "No functional units are in scope for this view."
	}
	return fmt.Sprintf("%d/%d units have direct evidence coverage in this view.", viewWithEvidenceCount(kind, units), len(units))
}

// attackChapterScope is what the attack chapters of a security view draw on:
// the units and nodes in scope, the security path rows, the labels and the evidence.
type attackChapterScope struct {
	units        []asciidocUnitSection
	nodeSet      map[string]bool
	securityRows []asciidocSecurityPathRow
	labels       map[string]string
	evidence     realizationEvidence
}

// attackChapterIndex indexes the attack vectors with the units they target.
type attackChapterIndex struct {
	scope               attackChapterScope
	unitByID            map[string]asciidocUnitSection
	attackByID          map[string]model.AttackVector
	unitIDsByAttack     map[string]map[string]bool
	mitigationsByAttack map[string][]string
	boundariesByUnit    map[string][]string
	rowsByAttackID      map[string][]asciidocSecurityPathRow
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-003
func buildSecurityAttackChapters(a model.AuthoredArchitecture, scope attackChapterScope) []asciidocSecurityAttackChapter {
	ix := newAttackChapterIndex(a, scope)
	attackIDs := ix.sortedAttackIDs()
	out := make([]asciidocSecurityAttackChapter, 0, len(attackIDs))
	for _, attackID := range attackIDs {
		out = append(out, ix.chapter(attackID))
	}
	return out
}

// newAttackChapterIndex indexes the units, targets, mitigations, boundaries and path rows.
// TRLC-LINKS: REQ-EMG-003
func newAttackChapterIndex(a model.AuthoredArchitecture, scope attackChapterScope) *attackChapterIndex {
	ix := &attackChapterIndex{
		scope:               scope,
		unitByID:            map[string]asciidocUnitSection{},
		attackByID:          map[string]model.AttackVector{},
		unitIDsByAttack:     map[string]map[string]bool{},
		mitigationsByAttack: map[string][]string{},
		boundariesByUnit:    map[string][]string{},
		rowsByAttackID:      map[string][]asciidocSecurityPathRow{},
	}
	for _, u := range scope.units {
		ix.unitByID[strings.TrimSpace(u.ID)] = u
	}
	for _, av := range a.AttackVectors {
		ix.attackByID[strings.TrimSpace(av.ID)] = av
	}
	for _, row := range scope.securityRows {
		if attackID := strings.TrimSpace(row.AttackVectorID); attackID != "" {
			ix.rowsByAttackID[attackID] = append(ix.rowsByAttackID[attackID], row)
		}
	}
	ix.indexMappings(a.Mappings)
	return ix
}

// indexMappings records which units each attack vector targets, and the
// mitigations of the attack vectors and boundaries of the units.
// TRLC-LINKS: REQ-EMG-003
func (ix *attackChapterIndex) indexMappings(mappings []model.Mapping) {
	for _, m := range mappings {
		from := strings.TrimSpace(m.From)
		to := strings.TrimSpace(m.To)
		switch strings.TrimSpace(m.Type) {
		case "targets":
			ix.indexTarget(from, to)
		case "mitigated_by":
			ix.noteLink(ix.mitigationsByAttack, "AV-", m)
		case "bounded_by":
			ix.noteLink(ix.boundariesByUnit, "FU-", m)
		}
	}
	for attackID, items := range ix.mitigationsByAttack {
		ix.mitigationsByAttack[attackID] = textutil.UniqueSorted(items)
	}
	for unitID, items := range ix.boundariesByUnit {
		ix.boundariesByUnit[unitID] = textutil.UniqueSorted(items)
	}
}

// noteLink adds the label of the target of a mapping under its source, when the
// source has the prefix and the target is not blank.
// TRLC-LINKS: REQ-EMG-003
func (ix *attackChapterIndex) noteLink(links map[string][]string, prefix string, m model.Mapping) {
	from := strings.TrimSpace(m.From)
	to := strings.TrimSpace(m.To)
	if strings.HasPrefix(from, prefix) && to != "" {
		links[from] = append(links[from], labelOrID(to, ix.scope.labels))
	}
}

// indexTarget records that the attack vector targets a unit that is in scope
// and has a section.
// TRLC-LINKS: REQ-EMG-003
func (ix *attackChapterIndex) indexTarget(attackID, unitID string) {
	if attackID == "" || unitID == "" || !ix.scope.nodeSet[unitID] {
		return
	}
	if _, ok := ix.unitByID[unitID]; ok {
		addToSetMap(ix.unitIDsByAttack, attackID, unitID)
	}
}

// sortedAttackIDs lists the attack vectors with targeted units by lower-cased
// name, then ID.
// TRLC-LINKS: REQ-EMG-003
func (ix *attackChapterIndex) sortedAttackIDs() []string {
	ids := sortedKeys(ix.unitIDsByAttack)
	name := func(id string) string {
		return strings.ToLower(textutil.NonEmpty(strings.TrimSpace(ix.attackByID[id].Name), id))
	}
	sort.SliceStable(ids, func(i, j int) bool {
		return cmp.Or(cmp.Compare(name(ids[i]), name(ids[j])), cmp.Compare(ids[i], ids[j])) < 0
	})
	return ids
}

// chapter is the chapter of an attack vector: the units it targets, the
// mitigations and boundaries, and the diagram of its paths.
// TRLC-LINKS: REQ-EMG-003
func (ix *attackChapterIndex) chapter(attackID string) asciidocSecurityAttackChapter {
	unitIDs := sortedKeys(ix.unitIDsByAttack[attackID])
	chapterUnits := make([]asciidocUnitSection, 0, len(unitIDs))
	boundaries := []string{}
	for _, unitID := range unitIDs {
		chapterUnits = append(chapterUnits, ix.unitByID[unitID])
		boundaries = append(boundaries, ix.boundariesByUnit[unitID]...)
	}
	sortUnitSections(chapterUnits)
	attackRows := append([]asciidocSecurityPathRow(nil), ix.rowsByAttackID[attackID]...)
	sortPathRows(attackRows)
	av := ix.attackByID[attackID]
	return asciidocSecurityAttackChapter{
		ID:              attackID,
		Name:            textutil.NonEmpty(strings.TrimSpace(av.Name), attackID),
		Description:     strings.TrimSpace(av.Description),
		MitigatedBy:     joinOrNone(ix.mitigationsByAttack[attackID]),
		TrustBoundaries: joinOrNone(textutil.UniqueSorted(boundaries)),
		Diagram:         buildSecurityPathMermaid(attackRows, ix.scope.evidence.runtime, ix.scope.evidence.code),
		Units:           chapterUnits,
	}
}

// sortUnitSections sorts the units by lower-cased name, then ID.
// TRLC-LINKS: REQ-EMG-003
func sortUnitSections(units []asciidocUnitSection) {
	name := func(u asciidocUnitSection) string {
		return strings.ToLower(textutil.NonEmpty(strings.TrimSpace(u.Name), u.ID))
	}
	sort.SliceStable(units, func(i, j int) bool {
		return cmp.Or(cmp.Compare(name(units[i]), name(units[j])), cmp.Compare(units[i].ID, units[j].ID)) < 0
	})
}

// sortPathRows sorts the rows by target, dependencies, then target ID.
// TRLC-LINKS: REQ-EMG-003
func sortPathRows(rows []asciidocSecurityPathRow) {
	sort.SliceStable(rows, func(i, j int) bool {
		return cmp.Or(cmp.Compare(rows[i].Target, rows[j].Target), cmp.Compare(rows[i].DependsOn, rows[j].DependsOn), cmp.Compare(rows[i].TargetID, rows[j].TargetID)) < 0
	})
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func viewWithEvidenceCount(kind string, units []asciidocUnitSection) int {
	withEvidence := 0
	for _, u := range units {
		if hasDirectEvidence(kind, u) {
			withEvidence++
		}
	}
	return withEvidence
}

// hasDirectEvidence reports whether a unit counts as covered in a view of the kind.
// TRLC-LINKS: REQ-EMG-003
func hasDirectEvidence(kind string, u asciidocUnitSection) bool {
	evidence := evidenceText(u)
	switch kind {
	case "communication", "deployment":
		return strings.Contains(evidence, "runtime:")
	case "traceability":
		return strings.Contains(evidence, "code modules:") || strings.Contains(evidence, "runtime:")
	case "security":
		return !strings.Contains(strings.ToLower(u.Threats), "no explicit attack vector")
	}
	return true
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildHealthRows(views []asciidocViewSection) []asciidocHealthRow {
	rows := make([]asciidocHealthRow, 0, len(views))
	for _, v := range views {
		gapCount := len(v.CoverageGaps)
		if gapCount == 1 && strings.Contains(strings.ToLower(strings.TrimSpace(v.CoverageGaps[0])), "no major coverage gaps") {
			gapCount = 0
		}
		withEvidence := viewWithEvidenceCount(v.Kind, v.Units)
		rows = append(rows, asciidocHealthRow{
			ViewID:                    v.ID,
			ViewHeading:               v.Heading,
			AuthoredStatus:            normalizeAuthoredStatus(v.AuthoredStatus),
			AuthoredStatusExplanation: normalizeAuthoredStatusExplanation(v.AuthoredStatusExplanation),
			UnitsInScope:              len(v.Units),
			WithEvidence:              withEvidence,
			GapCount:                  gapCount,
			Confidence:                viewCoverageConfidence(len(v.Units), withEvidence),
			HeuristicBasisExplanation: viewHeuristicBasis(v.Kind),
		})
	}
	return rows
}

// TRLC-LINKS: REQ-EMG-003
func normalizeAuthoredStatus(s string) string {
	x := strings.TrimSpace(strings.ToLower(s))
	if x == "" {
		return "unspecified"
	}
	return x
}

// TRLC-LINKS: REQ-EMG-003
func normalizeAuthoredStatusExplanation(s string) string {
	x := strings.TrimSpace(s)
	if x == "" {
		return "No authored status explanation provided."
	}
	return x
}

// TRLC-LINKS: REQ-EMG-003
func viewHeuristicBasis(kind string) string {
	switch kind {
	case "communication", "deployment":
		return "Counts a unit as covered when direct inferred runtime evidence exists."
	case "traceability":
		return "Counts a unit as covered when direct inferred runtime or code evidence exists."
	case "security":
		return "Counts a unit as covered when explicit authored attack-vector mapping exists."
	case "architecture-intent":
		return "Defaults authored units to covered; use authoredStatus as the primary publication signal."
	case "state-lifecycle":
		return "Lifecycle heuristic is currently basic; treat authoredStatus as primary signal."
	case "interaction-flow":
		return "Flow heuristic is step/branch-oriented; treat authoredStatus plus projection completeness as the primary signal."
	default:
		return "Coverage heuristic varies by view kind; interpret with authored status and explanation."
	}
}

// TRLC-LINKS: REQ-EMG-003
func viewCoverageConfidence(total, covered int) string {
	if total <= 0 {
		return "n/a"
	}
	ratio := float64(covered) / float64(total)
	switch {
	case ratio >= 0.80:
		return "high"
	case ratio >= 0.50:
		return "medium"
	default:
		return "low"
	}
}

// nextActionsByKind are the actions that close the evidence gaps of a view of each kind.
var nextActionsByKind = map[string][]string{
	"communication": {
		"Add explicit communication metadata (protocol, sync/async/event/callback) to authored mappings where currently implicit.",
		"Expose service protocol/port metadata in Helm or manifest values to enrich interface-edge inference.",
		"Capture missing communicating runtime units by adding discoverable workload artifacts under scanned inference roots.",
	},
	"deployment": {
		"Add owner metadata to Flux/Kustomization/Helm objects for clearer platform ownership mapping.",
		"Ensure deployment control artifacts (source, kustomization, release) are all included in scanned directories.",
		"Add deterministic naming conventions for release and namespace objects to improve cross-artifact linking.",
	},
	"traceability": {
		"Add explicit requirement-to-owner mappings where requirement scope is currently broad or ambiguous.",
		"Add ownership annotations for runtime and code artifacts where coverage is still unresolved.",
		"Prioritize closing gaps for units that are authored but lack both runtime and code evidence.",
	},
	"security": {
		"Add explicit authored attack-vector mappings for units currently marked without threat linkage.",
		"Add security signal ownership metadata (logs/alerts/events) for unresolved observability evidence.",
		"Map additional security-relevant dependencies and exposures into authored mappings for traceable coverage.",
	},
	"interaction-flow": {
		"Add missing step-level data_in/data_out fields for critical path transitions.",
		"Add explicit error branches for validation/retry/fallback behavior where currently implicit.",
		"Link each major step to concrete verification evidence in Traceability and Verification sections.",
	},
}

// TRLC-LINKS: REQ-EMG-003
func viewNextActions(kind string, gaps []string) []string {
	if noMajorGaps(gaps) {
		return []string{"No immediate evidence additions are required for this view."}
	}
	if actions, ok := nextActionsByKind[kind]; ok {
		return append([]string(nil), actions...)
	}
	return []string{"Review listed gaps and add missing authored or inferred evidence so this view reflects intended reality."}
}

// noMajorGaps reports whether the gaps are none or the single note that there are none.
// TRLC-LINKS: REQ-EMG-003
func noMajorGaps(gaps []string) bool {
	return len(gaps) == 0 || (len(gaps) == 1 && strings.Contains(strings.ToLower(gaps[0]), "no major coverage gaps"))
}

// scopedScenarioIDs is the IDs of the threat scenarios that are in scope of the view.
// TRLC-LINKS: REQ-EMG-003
func scopedScenarioIDs(a model.AuthoredArchitecture, nodeSet map[string]bool) map[string]bool {
	ids := map[string]bool{}
	for _, ts := range a.ThreatScenarios {
		if securityScenarioInScope(ts, nodeSet) {
			ids[strings.TrimSpace(ts.ID)] = true
		}
	}
	return ids
}

// controlVerificationRow is the table row of a control verification.
// TRLC-LINKS: REQ-EMG-003
func controlVerificationRow(cv model.ControlVerification, labels map[string]string) asciidocControlVerificationRow {
	return asciidocControlVerificationRow{
		ID:              strings.TrimSpace(cv.ID),
		Control:         labelOrID(strings.TrimSpace(cv.ControlRef), labels),
		ThreatScenarios: joinLabeledIDs(cv.ThreatScenarioRefs, labels),
		Risks:           joinLabeledIDs(cv.RiskRefs, labels),
		Method:          textutil.NonEmpty(strings.TrimSpace(cv.Method), "unknown"),
		Status:          textutil.NonEmpty(strings.TrimSpace(cv.Status), "unknown"),
		Owner:           labelOrID(strings.TrimSpace(cv.Owner), labels),
		LastTested:      textutil.NonEmpty(strings.TrimSpace(cv.LastTested), "unknown"),
		Findings:        textutil.NonEmpty(strings.Join(cv.Findings, "; "), "none"),
	}
}

// verificationsByScenario lists, for each scenario in scope, the rows of the
// control verifications that name it, by ID.
// TRLC-LINKS: REQ-EMG-003
func verificationsByScenario(a model.AuthoredArchitecture, inScope map[string]bool, labels map[string]string) map[string][]asciidocControlVerificationRow {
	byScenario := scenarioRows{}
	for _, cv := range a.ControlVerifications {
		if strings.TrimSpace(cv.ID) != "" {
			byScenario.add(controlVerificationRow(cv, labels), cv.ThreatScenarioRefs, inScope)
		}
	}
	for _, rows := range byScenario {
		sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	}
	return byScenario
}

// scenarioRows lists control verification rows by threat scenario.
type scenarioRows map[string][]asciidocControlVerificationRow

// add adds the row under each of the scenarios in scope.
// TRLC-LINKS: REQ-EMG-003
func (s scenarioRows) add(row asciidocControlVerificationRow, scenarioIDs []string, inScope map[string]bool) {
	for _, scenarioID := range scenarioIDs {
		if scenarioID = strings.TrimSpace(scenarioID); inScope[scenarioID] {
			s[scenarioID] = append(s[scenarioID], row)
		}
	}
}

// threatScenarioRow is the table row of a threat scenario.
// TRLC-LINKS: REQ-EMG-003
func threatScenarioRow(ts model.ThreatScenario, verifications []asciidocControlVerificationRow, labels map[string]string) asciidocThreatScenarioRow {
	id := strings.TrimSpace(ts.ID)
	return asciidocThreatScenarioRow{
		ID:                   id,
		Title:                textutil.NonEmpty(strings.TrimSpace(ts.Title), id),
		Summary:              textutil.NonEmpty(strings.TrimSpace(ts.Summary), "No additional scenario narrative was authored."),
		AttackVector:         labelOrID(strings.TrimSpace(ts.AttackVectorRef), labels),
		Scope:                joinLabeledIDs(ts.AppliesTo, labels),
		Flows:                joinLabeledIDs(ts.FlowRefs, labels),
		Likelihood:           textutil.NonEmpty(strings.TrimSpace(ts.Likelihood), "unknown"),
		Impact:               textutil.NonEmpty(strings.TrimSpace(ts.Impact), "unknown"),
		Severity:             textutil.NonEmpty(strings.TrimSpace(ts.Severity), "unknown"),
		Status:               textutil.NonEmpty(strings.TrimSpace(ts.Status), "unknown"),
		Owner:                labelOrID(strings.TrimSpace(ts.Owner), labels),
		Risk:                 labelOrID(strings.TrimSpace(ts.RiskRef), labels),
		Controls:             joinLabeledIDs(ts.RelatedControls, labels),
		Mitigations:          joinLabeledIDs(ts.MitigationRefs, labels),
		Verifications:        joinLabeledIDs(ts.VerificationRefs, labels),
		ControlVerifications: verifications,
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
// TRLC-LINKS: REQ-EMG-003
func buildSecurityThreatScenarioRows(a model.AuthoredArchitecture, nodeSet map[string]bool, labels map[string]string) []asciidocThreatScenarioRow {
	inScope := scopedScenarioIDs(a, nodeSet)
	verifications := verificationsByScenario(a, inScope, labels)
	rows := make([]asciidocThreatScenarioRow, 0, len(inScope))
	for _, ts := range a.ThreatScenarios {
		if id := strings.TrimSpace(ts.ID); inScope[id] {
			rows = append(rows, threatScenarioRow(ts, verifications[id], labels))
		}
	}
	sort.SliceStable(rows, func(i, j int) bool {
		return cmp.Or(cmp.Compare(rows[j].Severity, rows[i].Severity), cmp.Compare(rows[i].ID, rows[j].ID)) < 0
	})
	return rows
}

// scenarioRefsInScope is the trimmed, non-blank references, of the kind the
// refs function picks, of the threat scenarios that are in scope.
// TRLC-LINKS: REQ-EMG-003
func scenarioRefsInScope(a model.AuthoredArchitecture, nodeSet map[string]bool, refs func(model.ThreatScenario) []string) map[string]bool {
	inScope := map[string]bool{}
	for _, ts := range a.ThreatScenarios {
		if securityScenarioInScope(ts, nodeSet) {
			addTrimmed(inScope, refs(ts))
		}
	}
	return inScope
}

// addTrimmed adds the trimmed, non-blank items to the set.
// TRLC-LINKS: REQ-EMG-003
func addTrimmed(set map[string]bool, items []string) {
	for _, x := range items {
		if x = strings.TrimSpace(x); x != "" {
			set[x] = true
		}
	}
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
// TRLC-LINKS: REQ-EMG-003
func buildSecurityThreatAssumptionRows(a model.AuthoredArchitecture, nodeSet map[string]bool, labels map[string]string) []asciidocThreatAssumptionRow {
	inScope := scenarioRefsInScope(a, nodeSet, func(ts model.ThreatScenario) []string { return ts.AssumptionRefs })
	rows := []asciidocThreatAssumptionRow{}
	for _, x := range a.ThreatAssumptions {
		id := strings.TrimSpace(x.ID)
		if !(inScope[id] || securityAppliesToInScope(x.AppliesTo, nodeSet) || nodeSet[id]) {
			continue
		}
		rows = append(rows, asciidocThreatAssumptionRow{
			ID:        id,
			Title:     textutil.NonEmpty(strings.TrimSpace(x.Title), id),
			Status:    textutil.NonEmpty(strings.TrimSpace(x.Status), "unknown"),
			Owner:     labelOrID(strings.TrimSpace(x.Owner), labels),
			AppliesTo: joinLabeledIDs(x.AppliesTo, labels),
			Rationale: textutil.NonEmpty(strings.TrimSpace(x.Rationale), strings.TrimSpace(x.Statement)),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
// TRLC-LINKS: REQ-EMG-003
func buildSecurityThreatOutOfScopeRows(a model.AuthoredArchitecture, nodeSet map[string]bool, labels map[string]string) []asciidocThreatOutOfScopeRow {
	inScope := scenarioRefsInScope(a, nodeSet, func(ts model.ThreatScenario) []string { return ts.OutOfScopeRefs })
	rows := []asciidocThreatOutOfScopeRow{}
	for _, x := range a.ThreatOutOfScope {
		id := strings.TrimSpace(x.ID)
		if !(inScope[id] || securityAppliesToInScope(x.AppliesTo, nodeSet) || nodeSet[id]) {
			continue
		}
		rows = append(rows, asciidocThreatOutOfScopeRow{
			ID:        id,
			Title:     textutil.NonEmpty(strings.TrimSpace(x.Title), id),
			Status:    textutil.NonEmpty(strings.TrimSpace(x.Status), "unknown"),
			Owner:     labelOrID(strings.TrimSpace(x.Owner), labels),
			AppliesTo: joinLabeledIDs(x.AppliesTo, labels),
			ExpiresOn: textutil.NonEmpty(strings.TrimSpace(x.ExpiresOn), "none"),
			Reason:    strings.TrimSpace(x.Reason),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-THREAT-EXPORTER
// TRLC-LINKS: REQ-EMG-003
func buildSecurityThreatMitigationRows(a model.AuthoredArchitecture, nodeSet map[string]bool, labels map[string]string) []asciidocThreatMitigationRow {
	inScope := scopedScenarioIDs(a, nodeSet)
	rows := []asciidocThreatMitigationRow{}
	for _, m := range a.ThreatMitigations {
		if !(inScope[strings.TrimSpace(m.ThreatScenarioRef)] || nodeSet[strings.TrimSpace(m.ControlRef)] || nodeSet[strings.TrimSpace(m.ID)]) {
			continue
		}
		rows = append(rows, asciidocThreatMitigationRow{
			ID:             strings.TrimSpace(m.ID),
			ThreatScenario: labelOrID(strings.TrimSpace(m.ThreatScenarioRef), labels),
			Control:        labelOrID(strings.TrimSpace(m.ControlRef), labels),
			Status:         textutil.NonEmpty(strings.TrimSpace(m.Status), "unknown"),
			Effectiveness:  textutil.NonEmpty(strings.TrimSpace(m.Effectiveness), "unknown"),
			Owner:          labelOrID(strings.TrimSpace(m.Owner), labels),
			Verifications:  joinLabeledIDs(m.VerificationRefs, labels),
			Notes:          strings.TrimSpace(m.Notes),
		})
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-003
func buildSecurityControlVerificationRows(a model.AuthoredArchitecture, nodeSet map[string]bool, labels map[string]string) []asciidocControlVerificationRow {
	inScope := scopedScenarioIDs(a, nodeSet)
	rows := []asciidocControlVerificationRow{}
	for _, cv := range a.ControlVerifications {
		if !(nodeSet[strings.TrimSpace(cv.ControlRef)] || nodeSet[strings.TrimSpace(cv.ID)] || anyInSet(cv.ThreatScenarioRefs, inScope)) {
			continue
		}
		rows = append(rows, controlVerificationRow(cv, labels))
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func buildSecurityFlowRows(a model.AuthoredArchitecture, nodeSet map[string]bool, labels map[string]string) []asciidocSecurityFlowRow {
	rows := []asciidocSecurityFlowRow{}
	for _, f := range a.Flows {
		if nodeSet[strings.TrimSpace(f.ID)] || nodeSet[strings.TrimSpace(f.SourceRef)] || nodeSet[strings.TrimSpace(f.DestinationRef)] {
			rows = append(rows, securityFlowRow(f, labels))
		}
	}
	sort.SliceStable(rows, func(i, j int) bool { return rows[i].ID < rows[j].ID })
	return rows
}

// flowBoundary is whether a flow crosses a boundary, as true or false, and the
// first trust boundary a crossing step names, or none.
// TRLC-LINKS: REQ-EMG-003
func flowBoundary(f model.Flow, labels map[string]string) (crossing, boundary string) {
	crossing, boundary = "false", "none"
	for _, s := range f.Steps {
		if !s.BoundaryCrossing {
			continue
		}
		crossing = "true"
		if strings.TrimSpace(s.TrustBoundaryRef) != "" {
			return crossing, labelOrID(strings.TrimSpace(s.TrustBoundaryRef), labels)
		}
	}
	return crossing, boundary
}

// securityFlowRow is the table row of a flow.
// TRLC-LINKS: REQ-EMG-003
func securityFlowRow(f model.Flow, labels map[string]string) asciidocSecurityFlowRow {
	id := strings.TrimSpace(f.ID)
	crossing, boundary := flowBoundary(f, labels)
	return asciidocSecurityFlowRow{
		ID:               id,
		Title:            textutil.NonEmpty(strings.TrimSpace(f.Title), id),
		Kind:             textutil.NonEmpty(strings.TrimSpace(f.Kind), "flow"),
		Direction:        textutil.NonEmpty(strings.TrimSpace(f.Direction), "unknown"),
		Frequency:        textutil.NonEmpty(strings.TrimSpace(f.Frequency), "unknown"),
		Source:           labelOrID(strings.TrimSpace(f.SourceRef), labels),
		Destination:      labelOrID(strings.TrimSpace(f.DestinationRef), labels),
		Protocol:         textutil.NonEmpty(strings.TrimSpace(f.Protocol), "n/a"),
		Authentication:   textutil.NonEmpty(strings.TrimSpace(f.Authentication), "n/a"),
		Encryption:       textutil.NonEmpty(strings.TrimSpace(f.EncryptionInTransit), "n/a"),
		Integrity:        textutil.NonEmpty(strings.TrimSpace(f.IntegrityProtection), "n/a"),
		TrustBoundary:    boundary,
		BoundaryCrossing: crossing,
		Threats:          joinLabeledIDs(f.Threats, labels),
		Data:             joinLabeledIDs(f.DataRefs, labels),
	}
}

// TRLC-LINKS: REQ-EMG-003
func securityScenarioInScope(ts model.ThreatScenario, nodeSet map[string]bool) bool {
	if nodeSet[strings.TrimSpace(ts.ID)] || nodeSet[strings.TrimSpace(ts.AttackVectorRef)] || nodeSet[strings.TrimSpace(ts.RiskRef)] {
		return true
	}
	if securityAppliesToInScope(ts.AppliesTo, nodeSet) {
		return true
	}
	if anyInSet(ts.FlowRefs, nodeSet) || anyInSet(ts.RelatedControls, nodeSet) {
		return true
	}
	return false
}

// TRLC-LINKS: REQ-EMG-003
func securityAppliesToInScope(ids []string, nodeSet map[string]bool) bool {
	for _, id := range ids {
		if nodeSet[strings.TrimSpace(id)] {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-003
func anyInSet(ids []string, set map[string]bool) bool {
	for _, id := range ids {
		if set[strings.TrimSpace(id)] {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-003
func joinLabeledIDs(ids []string, labels map[string]string) string {
	items := make([]string, 0, len(ids))
	seen := map[string]bool{}
	for _, id := range ids {
		id = strings.TrimSpace(id)
		if id == "" || seen[id] {
			continue
		}
		seen[id] = true
		items = append(items, labelOrID(id, labels))
	}
	if len(items) == 0 {
		return "none"
	}
	sort.Strings(items)
	return strings.Join(items, ", ")
}

// TRLC-LINKS: REQ-EMG-003
func joinOrNone(items []string) string {
	items = textutil.UniqueSorted(items)
	if len(items) == 0 {
		return "none"
	}
	return strings.Join(items, ", ")
}

// TRLC-LINKS: REQ-EMG-003
func labelOrID(id string, labels map[string]string) string {
	id = strings.TrimSpace(id)
	if id == "" {
		return "none"
	}
	if l := strings.TrimSpace(labels[id]); l != "" {
		return l
	}
	return id
}

// publicationViewHeading distinguishes multiple views of the same kind in a TOC.
// TRLC-LINKS: REQ-EMG-003
func publicationViewHeading(kind, id string, sameKindCount int) string {
	heading := viewHeading(kind)
	if sameKindCount > 1 {
		heading += " (" + strings.TrimSpace(id) + ")"
	}
	return heading
}

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
// TRLC-LINKS: REQ-EMG-003
func functionalPublicationArchitecture(a model.AuthoredArchitecture, selected map[string]bool) model.AuthoredArchitecture {
	owners := interfaceOwners(a)
	selected = withInterfaceOwners(selected, owners)
	include := map[string]bool{}
	for id, chosen := range selected {
		if chosen {
			include[id] = true
		}
	}
	out := model.AuthoredArchitecture{}
	out.FunctionalUnits = selectedUnits(a.FunctionalUnits, selected, include)
	out.FunctionalGroups = includedGroups(a.FunctionalGroups, include)
	out.Actors = selectedActors(a.Actors, selected)
	out.ReferencedElements = selectedReferences(a.ReferencedElements, selected)
	out.Mappings = includedMappings(a.Mappings, owners, include)
	return out
}

// selectedUnits lists the selected units and includes their groups.
// TRLC-LINKS: REQ-EMG-003
func selectedUnits(units []model.FunctionalUnit, selected, include map[string]bool) []model.FunctionalUnit {
	var out []model.FunctionalUnit
	for _, unit := range units {
		if !selected[unit.ID] {
			continue
		}
		out = append(out, unit)
		if unit.Group != "" {
			include[unit.Group] = true
		}
	}
	return out
}

// includedGroups lists the groups in the include set.
// TRLC-LINKS: REQ-EMG-003
func includedGroups(groups []model.FunctionalGroup, include map[string]bool) []model.FunctionalGroup {
	var out []model.FunctionalGroup
	for _, group := range groups {
		if include[group.ID] {
			out = append(out, group)
		}
	}
	return out
}

// selectedActors lists the selected actors.
// TRLC-LINKS: REQ-EMG-003
func selectedActors(actors []model.Actor, selected map[string]bool) []model.Actor {
	var out []model.Actor
	for _, actor := range actors {
		if selected[actor.ID] {
			out = append(out, actor)
		}
	}
	return out
}

// selectedReferences lists the selected referenced elements.
// TRLC-LINKS: REQ-EMG-003
func selectedReferences(refs []model.ReferencedElement, selected map[string]bool) []model.ReferencedElement {
	var out []model.ReferencedElement
	for _, ref := range refs {
		if selected[ref.ID] {
			out = append(out, ref)
		}
	}
	return out
}

// includedMappings lists the mappings between included elements, with
// interfaces replaced by their owners.
// TRLC-LINKS: REQ-EMG-003
func includedMappings(mappings []model.Mapping, owners map[string]string, include map[string]bool) []model.Mapping {
	var out []model.Mapping
	for _, mapping := range mappings {
		mapping = throughOwners(mapping, owners)
		if include[mapping.From] && include[mapping.To] {
			out = append(out, mapping)
		}
	}
	return out
}

// withInterfaceOwners returns the selection with the owner of each selected
// interface added, so a dependency that goes through a selected interface is
// drawn to its owner.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
func withInterfaceOwners(selected map[string]bool, owners map[string]string) map[string]bool {
	out := make(map[string]bool, len(selected))
	for id, chosen := range selected {
		out[id] = chosen
	}
	for id, owner := range owners {
		if selected[id] {
			out[owner] = true
		}
	}
	return out
}

// throughOwners returns a dependency with an interface at an end replaced by
// the interface's owner; any other relationship is returned as it is.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
func throughOwners(m model.Mapping, owners map[string]string) model.Mapping {
	if m.Type == "depends_on" {
		m.From, m.To = dependencyEnd(owners, m.From), dependencyEnd(owners, m.To)
	}
	return m
}

// interfaceOwners maps each interface to its owner: interface.owner, else the
// element that a contains relationship names as its container.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
func interfaceOwners(a model.AuthoredArchitecture) map[string]string {
	owners := containers(a.Mappings)
	for _, i := range a.Interfaces {
		owners = withOwnerField(owners, i)
	}
	return owners
}

// containers maps the target of each contains relationship to its source.
// TRLC-LINKS: REQ-EMG-003
func containers(mappings []model.Mapping) map[string]string {
	owners := map[string]string{}
	for _, m := range mappings {
		if strings.TrimSpace(m.Type) == "contains" {
			owners[strings.TrimSpace(m.To)] = strings.TrimSpace(m.From)
		}
	}
	return owners
}

// withOwnerField sets the owner of an interface from its owner field when
// the field is set.
// TRLC-LINKS: REQ-EMG-003
func withOwnerField(owners map[string]string, i model.Interface) map[string]string {
	if owner := strings.TrimSpace(i.Owner); owner != "" {
		owners[strings.TrimSpace(i.ID)] = owner
	}
	return owners
}

// dependencyEnd returns the end of a dependency as the functional diagrams
// draw it: an interface stands for its owner (a dependency goes through an
// interface, ADR-EMG-085), every other ID is itself.
// TRLC-LINKS: REQ-EMG-003
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR
func dependencyEnd(owners map[string]string, id string) string {
	id = strings.TrimSpace(id)
	if owner := owners[id]; owner != "" && strings.HasPrefix(id, "IF-") {
		return owner
	}
	return id
}
