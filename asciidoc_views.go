package engmodel

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

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
	case "architecture-intent":
		return "Architecture Intent View"
	case "communication":
		return "Communication View"
	case "deployment":
		return "Deployment View"
	case "security":
		return "Security View"
	case "traceability":
		return "Traceability View"
	case "state-lifecycle":
		return "State Lifecycle View"
	default:
		return strings.Title(kind) + " View"
	}
}

func resolveViewIDs(bundle model.Bundle, options AsciiDocOptions) []string {
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

func inferredDescription(kind string) string {
	switch kind {
	case "architecture-intent":
		return "Show the stable authored architecture intent: capability areas, unit ownership boundaries, and intentional responsibility split."
	case "communication":
		return "Show who communicates with whom across key request paths, including interface edges and dependency handoffs."
	case "deployment":
		return "Show inferred deployment artifacts and ownership mapping to authored units. This view emphasizes deployment relationships and platform operations."
	case "security":
		return "Show inferred exposure and dependency risk points aligned to unit boundaries, focused on attack paths and security evidence."
	case "traceability":
		return "Show requirement-to-unit-to-evidence traceability, including coverage confidence and explicit evidence gaps."
	case "state-lifecycle":
		return "Show workflow state transitions, ownership of transitions, and key exceptional paths."
	default:
		return "Show authored architecture scope for this view."
	}
}

func viewQuestions(kind string) []string {
	switch kind {
	case "architecture-intent":
		return []string{
			"What are the main capability areas?",
			"What are the stable units of responsibility and what does each unit own?",
			"How are responsibilities intentionally split?",
		}
	case "communication":
		return []string{
			"Who talks to whom and over which interfaces?",
			"Which edges are synchronous, asynchronous, event-driven, or callback-oriented?",
			"Which communication paths are critical to the main request flow?",
		}
	case "deployment":
		return []string{
			"How does authored intent become deployed runtime?",
			"What control-plane objects manage deployable units?",
			"Where are platform ownership boundaries and missing ownership signals?",
		}
	case "traceability":
		return []string{
			"Which requirements map to which units?",
			"Which authored units have runtime/code evidence and which do not?",
			"Where are confidence or completeness gaps in trace coverage?",
		}
	case "security":
		return []string{
			"What are the main trust boundaries and attack paths?",
			"Which units are exposed and what security evidence exists?",
			"Where is the threat model incomplete?",
		}
	case "state-lifecycle":
		return []string{
			"What states exist and which events trigger transitions?",
			"Which unit owns each state transition?",
			"Where do automatic and manual flows diverge?",
		}
	default:
		return []string{"What does this view show?"}
	}
}

func viewCoverageGaps(kind string, units []asciidocUnitSection) []string {
	gaps := []string{}
	for _, u := range units {
		evidence := strings.ToLower(strings.TrimSpace(u.Evidence))
		switch kind {
		case "communication", "deployment":
			if !strings.Contains(evidence, "runtime:") {
				gaps = append(gaps, fmt.Sprintf("%s has no direct inferred runtime/deployment evidence yet.", u.Name))
			}
		case "traceability":
			if !strings.Contains(evidence, "code modules:") {
				gaps = append(gaps, fmt.Sprintf("%s has no direct inferred code evidence yet.", u.Name))
			}
			if !strings.Contains(evidence, "runtime:") {
				gaps = append(gaps, fmt.Sprintf("%s has no direct inferred runtime evidence yet.", u.Name))
			}
		case "security":
			if strings.Contains(strings.ToLower(u.Threats), "no explicit attack vector") {
				gaps = append(gaps, fmt.Sprintf("%s has no explicit authored attack-vector mapping yet; document technical and fraud/abuse threats even when mitigated.", u.Name))
			}
		}
	}
	if len(gaps) == 0 {
		return []string{"No major coverage gaps detected from current authored and inferred inputs."}
	}
	sort.Strings(gaps)
	return gaps
}

func viewCoverageSummary(kind string, units []asciidocUnitSection) string {
	if len(units) == 0 {
		return "No functional units are in scope for this view."
	}
	withEvidence := 0
	for _, u := range units {
		evidence := strings.ToLower(strings.TrimSpace(u.Evidence))
		switch kind {
		case "communication", "deployment":
			if strings.Contains(evidence, "runtime:") {
				withEvidence++
			}
		case "traceability":
			if strings.Contains(evidence, "code modules:") || strings.Contains(evidence, "runtime:") {
				withEvidence++
			}
		case "security":
			if !strings.Contains(strings.ToLower(u.Threats), "no explicit attack vector") {
				withEvidence++
			}
		default:
			withEvidence++
		}
	}
	return fmt.Sprintf("%d/%d units have direct evidence coverage in this view.", withEvidence, len(units))
}

func buildSecurityAttackChapters(a model.AuthoredArchitecture, units []asciidocUnitSection, nodeSet map[string]bool, securityRows []asciidocSecurityPathRow, runtime []inferredRuntimeItem, code []inferredCodeItem) []asciidocSecurityAttackChapter {
	unitByID := map[string]asciidocUnitSection{}
	for _, u := range units {
		unitByID[strings.TrimSpace(u.ID)] = u
	}
	unitIDsByAttack := map[string]map[string]bool{}
	for _, m := range a.Mappings {
		if strings.TrimSpace(m.Type) != "targets" {
			continue
		}
		attackID := strings.TrimSpace(m.From)
		unitID := strings.TrimSpace(m.To)
		if attackID == "" || unitID == "" {
			continue
		}
		if !nodeSet[unitID] {
			continue
		}
		if _, ok := unitByID[unitID]; !ok {
			continue
		}
		if unitIDsByAttack[attackID] == nil {
			unitIDsByAttack[attackID] = map[string]bool{}
		}
		unitIDsByAttack[attackID][unitID] = true
	}

	attackByID := map[string]model.AttackVector{}
	for _, av := range a.AttackVectors {
		attackByID[strings.TrimSpace(av.ID)] = av
	}
	rowsByAttackID := map[string][]asciidocSecurityPathRow{}
	for _, row := range securityRows {
		attackID := strings.TrimSpace(row.AttackVectorID)
		if attackID == "" {
			continue
		}
		rowsByAttackID[attackID] = append(rowsByAttackID[attackID], row)
	}

	attackIDs := make([]string, 0, len(unitIDsByAttack))
	for attackID := range unitIDsByAttack {
		attackIDs = append(attackIDs, attackID)
	}
	sort.SliceStable(attackIDs, func(i, j int) bool {
		left := attackByID[attackIDs[i]]
		right := attackByID[attackIDs[j]]
		leftName := strings.ToLower(nonEmpty(strings.TrimSpace(left.Name), attackIDs[i]))
		rightName := strings.ToLower(nonEmpty(strings.TrimSpace(right.Name), attackIDs[j]))
		if leftName != rightName {
			return leftName < rightName
		}
		return attackIDs[i] < attackIDs[j]
	})

	out := make([]asciidocSecurityAttackChapter, 0, len(attackIDs))
	for _, attackID := range attackIDs {
		unitIDs := keysFromSet(unitIDsByAttack[attackID])
		sort.Strings(unitIDs)
		chapterUnits := make([]asciidocUnitSection, 0, len(unitIDs))
		for _, unitID := range unitIDs {
			chapterUnits = append(chapterUnits, unitByID[unitID])
		}
		sort.SliceStable(chapterUnits, func(i, j int) bool {
			left := strings.ToLower(nonEmpty(strings.TrimSpace(chapterUnits[i].Name), chapterUnits[i].ID))
			right := strings.ToLower(nonEmpty(strings.TrimSpace(chapterUnits[j].Name), chapterUnits[j].ID))
			if left != right {
				return left < right
			}
			return chapterUnits[i].ID < chapterUnits[j].ID
		})
		av := attackByID[attackID]
		attackRows := append([]asciidocSecurityPathRow(nil), rowsByAttackID[attackID]...)
		sort.SliceStable(attackRows, func(i, j int) bool {
			if attackRows[i].Target != attackRows[j].Target {
				return attackRows[i].Target < attackRows[j].Target
			}
			if attackRows[i].DependsOn != attackRows[j].DependsOn {
				return attackRows[i].DependsOn < attackRows[j].DependsOn
			}
			return attackRows[i].TargetID < attackRows[j].TargetID
		})
		out = append(out, asciidocSecurityAttackChapter{
			ID:          attackID,
			Name:        nonEmpty(strings.TrimSpace(av.Name), attackID),
			Description: strings.TrimSpace(av.Description),
			Diagram:     buildSecurityPathMermaid(attackRows, runtime, code),
			Units:       chapterUnits,
		})
	}
	return out
}

func viewWithEvidenceCount(kind string, units []asciidocUnitSection) int {
	withEvidence := 0
	for _, u := range units {
		evidence := strings.ToLower(strings.TrimSpace(u.Evidence))
		switch kind {
		case "communication", "deployment":
			if strings.Contains(evidence, "runtime:") {
				withEvidence++
			}
		case "traceability":
			if strings.Contains(evidence, "code modules:") || strings.Contains(evidence, "runtime:") {
				withEvidence++
			}
		case "security":
			if !strings.Contains(strings.ToLower(u.Threats), "no explicit attack vector") {
				withEvidence++
			}
		default:
			withEvidence++
		}
	}
	return withEvidence
}

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

func normalizeAuthoredStatus(s string) string {
	x := strings.TrimSpace(strings.ToLower(s))
	if x == "" {
		return "unspecified"
	}
	return x
}

func normalizeAuthoredStatusExplanation(s string) string {
	x := strings.TrimSpace(s)
	if x == "" {
		return "No authored status explanation provided."
	}
	return x
}

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
	default:
		return "Coverage heuristic varies by view kind; interpret with authored status and explanation."
	}
}

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

func viewNextActions(kind string, gaps []string) []string {
	if len(gaps) == 0 || (len(gaps) == 1 && strings.Contains(strings.ToLower(gaps[0]), "no major coverage gaps")) {
		return []string{"No immediate evidence additions are required for this view."}
	}
	switch kind {
	case "communication":
		return []string{
			"Add explicit communication metadata (protocol, sync/async/event/callback) to authored mappings where currently implicit.",
			"Expose service protocol/port metadata in Helm or manifest values to enrich interface-edge inference.",
			"Capture missing communicating runtime units by adding discoverable workload artifacts under scanned inference roots.",
		}
	case "deployment":
		return []string{
			"Add owner metadata to Flux/Kustomization/Helm objects for clearer platform ownership mapping.",
			"Ensure deployment control artifacts (source, kustomization, release) are all included in scanned directories.",
			"Add deterministic naming conventions for release and namespace objects to improve cross-artifact linking.",
		}
	case "traceability":
		return []string{
			"Add explicit requirement-to-owner mappings where requirement scope is currently broad or ambiguous.",
			"Add ownership annotations for runtime and code artifacts where coverage is still unresolved.",
			"Prioritize closing gaps for units that are authored but lack both runtime and code evidence.",
		}
	case "security":
		return []string{
			"Add explicit authored attack-vector mappings for units currently marked without threat linkage.",
			"Add security signal ownership metadata (logs/alerts/events) for unresolved observability evidence.",
			"Map additional security-relevant dependencies and exposures into authored mappings for traceable coverage.",
		}
	default:
		return []string{
			"Review listed gaps and add missing authored or inferred evidence so this view reflects intended reality.",
		}
	}
}
