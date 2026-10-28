// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"sort"
	"strings"
)

// Rules is the machine-readable subset of the validation rules that agents
// need before authoring YAML: controlled vocabularies, allowed relationship
// kind pairs and cross-reference field rules. It is derived from the same
// tables Bundle uses, so it cannot drift from validation.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
type Rules struct {
	Enums             map[string][]string `json:"enums"`
	RelationshipPairs []RelationshipPair  `json:"relationshipPairs"`
	FieldRules        []string            `json:"fieldRules"`
}

// RelationshipPair is one allowed authoredArchitecture.mappings type with
// the entity kinds of its from and to ends.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
type RelationshipPair struct {
	Type string `json:"type"`
	From string `json:"from"`
	To   string `json:"to"`
}

// enumTables maps each authoring field key to the validation table that checks it.
var enumTables = map[string]map[string]bool{
	"requirement.status":                     allowedRequirementStatus,
	"requirement.verificationMethods":        allowedRequirementVerificationMethods,
	"decision.status":                        allowedDecisionStatus,
	"document.control.status":                allowedDocumentStatus,
	"relationship.type":                      allowedMappingTypes,
	"view.kind":                              allowedViewKinds,
	"view.includeKinds":                      allowedViewEntityKinds,
	"view.includeMappings":                   allowedViewMappingTypes,
	"risk.status":                            allowedRiskStatus,
	"risk.level":                             allowedRiskLevel,
	"risk.response":                          allowedRiskResponse,
	"poamItem.status":                        allowedPOAMStatus,
	"flow.kind":                              allowedFlowKinds,
	"flow.direction":                         allowedFlowDirection,
	"flow.frequency":                         allowedFlowFrequency,
	"flowStep.kind":                          allowedFlowStepKinds,
	"flowStep.flowType":                      allowedFlowTypes,
	"threatScenario.status":                  allowedThreatScenarioStatus,
	"threatAssumption.status":                allowedThreatAssumptionStatus,
	"threatOutOfScope.status":                allowedThreatOutOfScopeStatus,
	"threatMitigation.status":                allowedThreatMitigationStatus,
	"threatMitigation.effectiveness":         allowedThreatMitigationEffectiveness,
	"controlVerification.method":             allowedControlVerificationMethod,
	"controlVerification.status":             allowedControlVerificationStatus,
	"complianceMapping.implementationType":   allowedControlImplementationTypes,
	"complianceMapping.implementationStatus": allowedControlImplementationStatuses,
}

var authoringFieldRules = []string{
	"Enum keys name the field; risk.level applies to risk likelihood, impact and residualRisk and to threatScenario likelihood, impact and severity. view.includeKinds also governs excludeKinds and view.includeMappings also governs excludeMappings.",
	"Status, level, response, method, effectiveness, direction, frequency, flowType and implementation values are trimmed and compared lowercase; relationship.type, view.kind, view entity kinds, view mapping types and flowStep.kind are compared exactly.",
	"Owner fields must be actor (ACT-*) IDs: risk.owner, threatScenario.owner, threatAssumption.owner, threatOutOfScope.owner, threatMitigation.owner, controlVerification.owner, poamItem.responsibleRole and every complianceMapping.responsibleRoles entry. Functional units, systems and free-text team names are rejected.",
	"interface.owner must be a functional unit (FU-*) ID.",
	"appliesTo on risks, threat scenarios, threat assumptions, threat out-of-scope entries and compliance mappings accepts any ID declared as a model element (architecture, behavior, assurance, compliance, views, decisions, requirements or semantics.elements). Catalog-only entries are not elements: SYS-* system IDs from catalog.systems, and catalog terms without a matching authored element, are rejected as unknown appliesTo IDs; scope to the functional units, groups or interfaces instead.",
	"A threat scenario needs at least one appliesTo or flowRefs entry; flowRefs must be FLOW-* IDs, relatedControls CTRL-* IDs.",
	"threatMitigation needs threatScenarioRef (an existing threat scenario) and controlRef (an existing control).",
	"A compliance mapping needs profileRef, modelControlRef, at least one OSCAL-style controlIds entry (for example ac-2 or ac-2(1)) and at least one appliesTo ID.",
	"Dates (decision.date, poamItem.dueDate, threatOutOfScope.expiresOn, controlVerification.lastTested, document control issueDate) use YYYY-MM-DD.",
	"A requirement with verificationMethods must set verificationCriteria; a derived requirement must set derivedRationale. decision.status, decision.date, context and decision text are required.",
	"Authored mappings must not reference inferred RT-* or CODE-* IDs; when both ends are known, the (type, from kind, to kind) triple must appear in relationshipPairs.",
}

// AuthoringRules returns the validation vocabularies and relationship pairs as
// sorted, deterministic data for the model authoring contract.
// TRLC-LINKS: REQ-EMG-045, REQ-EMG-009
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func AuthoringRules() Rules {
	enums := make(map[string][]string, len(enumTables))
	for key, table := range enumTables {
		enums[key] = sortedKeys(table)
	}
	pairs := []RelationshipPair{}
	for mappingType, kinds := range mappingPairs {
		for key, ok := range kinds {
			if !ok {
				continue
			}
			from, to, _ := strings.Cut(key, ":")
			pairs = append(pairs, RelationshipPair{Type: mappingType, From: from, To: to})
		}
	}
	sort.Slice(pairs, func(i, j int) bool {
		if pairs[i].Type != pairs[j].Type {
			return pairs[i].Type < pairs[j].Type
		}
		if pairs[i].From != pairs[j].From {
			return pairs[i].From < pairs[j].From
		}
		return pairs[i].To < pairs[j].To
	})
	return Rules{Enums: enums, RelationshipPairs: pairs, FieldRules: append([]string(nil), authoringFieldRules...)}
}

// TRLC-LINKS: REQ-EMG-045
func sortedKeys(table map[string]bool) []string {
	out := make([]string, 0, len(table))
	for key, ok := range table {
		if ok {
			out = append(out, key)
		}
	}
	sort.Strings(out)
	return out
}
