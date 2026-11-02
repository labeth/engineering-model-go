// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"reflect"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
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
	// FieldSets lists, per entity kind, the YAML fields strict decoding
	// accepts, in declaration order; any other field is rejected.
	FieldSets map[string][]string `json:"fieldSets"`
	// PublicationSections are the manifest publication sections that may
	// export stable IDs. Decisions cannot be published.
	PublicationSections []string `json:"publicationSections"`
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
	"controlVerification.verificationRefs entries name the tests that verify the control: inferred CODE-<PATH>--<NAME> IDs of test declarations or REQ-* IDs whose linked tests verify it. verification.status_stale warns while a referenced test exists and the status is unset or not-run (threatMitigation: unset or planned); the status itself stays authored.",
	"triggered_by (state -> event) means the event is handled within the state: the state accepts or reacts to it without being entered by it. Entering a state is a transitions_to (state -> state); name the causing event in that relationship's description, or, to record it structurally, author an architecture.semantics.relationships transition with triggers: kind transition, source and target states and triggers listing the event. Do not use triggered_by for the event that enters a state; the SysML export renders it as an accept within the source state.",
	"Assurance links: a threat scenario's verificationRefs list every control verification that its mitigations reference or that names it in threatScenarioRefs (warning threat.verification_missing); a threatMitigation verificationRefs entry names a control verification of the mitigation's own controlRef (error mitigation.verification_control_mismatch); a controlVerification with method test and status pass or partial names its tests in verificationRefs (warning verification.unlinked); every control has a control verification with its controlRef, else the Gemara evaluation log marks it Needs Review (warning gemara.control_needs_review).",
	"fieldSets lists the fields each entity kind accepts; strict decoding rejects any other field (for example control takes only id, name, category and description). evidence applies to every evidence, artifact list entry (path, description).",
	"Publications: a manifest publications entry has an id and the sections architecture, behavior, assurance, compliance, requirements and views (publicationSections), each a list of stable IDs from that document. Decisions cannot be published, so a consumer cannot reference alias::DEC-*; keep a shared decision in the repository that owns it and quote it in the consumer.",
	"A flow step with boundaryCrossing: true needs trustBoundaryRef naming an existing trust boundary (error model.missing_flow_boundary_ref), and should name an onError step (warning behavior.flow_step_no_onerror).",
	"behavior.event_unused counts as a use of an event: the to end of a triggered_by mapping; the source, target or a triggers entry of an architecture.semantics.relationships entry; the sourceRef or destinationRef of a flow; and the ref, sourceRef or destinationRef of a flow step. Flow or step properties, descriptions and other prose do not count.",
	"Acceptance: a risk with status accepted or response accept, and a threat scenario with status accepted, needs acceptedBy with actor (an ACT-* actor), date (a YYYY-MM-DD calendar date) and statement (the owner's own words) or decisionRef (a DEC-* decision with status accepted whose context quotes the owner); without it validation reports risk.acceptance_unrecorded (error), and an actor, date or decisionRef that does not resolve is model.invalid_acceptance (error). An acceptedBy actor other than the risk's owner is risk.acceptance_not_owner (warning). response accept with status open is risk.accept_while_open (warning). The Gemara policy lists a risk under risks.accepted only with a complete, resolving acceptedBy and status accepted, or response accept with a status other than open. Authored risk and threat scenario IDs must not contain :: (reserved for imported alias::ID entities). Never record an acceptance the owner did not give.",
	"lint.multiple_responses exempts a single response with a field list: the text has exactly one shall and no semicolon, the list opens at the response's first with, including or of, no and, or or comma comes before it, the list holds no then, and no list item after the first has the, a, an, its, their, each, every or all as its second word (that reads as a verb with its object). Passes: 'The flight logger shall publish one record with UTC time, latitude and altitude.' Reported: 'The flight logger shall store the record and notify the operator.'",
	"In strict EARS mode (lintRun.mode: strict) the system term of a requirement (the words before shall) must be the name or an alias of a catalog system, functional group or functional unit, written as in the catalog (for example 'the engineering model go shall ...'); a repository slug such as sim-interface is not a catalog name unless the catalog names it so.",
	"Warnings: behavior.event_unused (a behavior event in no triggered_by relationship, transition trigger or flow), behavior.state_unreachable (with transitions in the model, a state other than the first declared, initial one that no transitions_to enters), behavior.flow_step_no_onerror (a boundaryCrossing step without onError) and catalog.name_has_operator (a clause-term catalog name or alias with a standalone and, or, not, a comma or a parenthesis, which EARS reads as operators).",
}

// fieldSetTypes maps each entity kind of fieldSets to the strict-decoding Go
// type whose yaml tags are the accepted fields.
var fieldSetTypes = map[string]any{
	"acceptanceRecord":    model.AcceptanceRecord{},
	"actor":               model.Actor{},
	"attackVector":        model.AttackVector{},
	"catalogEntry":        model.CatalogEntry{},
	"complianceMapping":   model.ComplianceMapping{},
	"complianceProfile":   model.ComplianceProfile{},
	"contractEntry":       model.ContractEntry{},
	"control":             model.Control{},
	"controlVerification": model.ControlVerification{},
	"dataObject":          model.DataObject{},
	"decision":            model.Decision{},
	"dependency":          model.ManifestDependency{},
	"deploymentTarget":    model.DeploymentTarget{},
	"event":               model.Event{},
	"evidence":            model.RiskEvidence{},
	"flow":                model.Flow{},
	"flowStep":            model.FlowStep{},
	"functionalGroup":     model.FunctionalGroup{},
	"functionalUnit":      model.FunctionalUnit{},
	"hardwareInterface":   model.HardwareInterface{},
	"hardwareItem":        model.HardwareItem{},
	"interface":           model.Interface{},
	"lintWaiver":          model.LintWaiver{},
	"mapping":             model.Mapping{},
	"poamItem":            model.POAMItem{},
	"publication":         model.ManifestPublication{},
	"referencedElement":   model.ReferencedElement{},
	"requirement":         model.Requirement{},
	"risk":                model.Risk{},
	"state":               model.State{},
	"subsystem":           model.Subsystem{},
	"threatAssumption":    model.ThreatAssumption{},
	"threatMitigation":    model.ThreatMitigation{},
	"threatOutOfScope":    model.ThreatOutOfScope{},
	"threatScenario":      model.ThreatScenario{},
	"trustBoundary":       model.TrustBoundary{},
	"view":                model.View{},
}

// publicationSections returns the manifest publication sections in schema
// order: the yaml fields of model.ManifestPublication after id.
// TRLC-LINKS: REQ-EMG-087
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func publicationSections() []string {
	var out []string
	for _, field := range yamlFields(reflect.TypeOf(model.ManifestPublication{})) {
		if field != "id" {
			out = append(out, field)
		}
	}
	return out
}

// yamlFields returns the yaml field names of a struct type in declaration
// order.
// TRLC-LINKS: REQ-EMG-087
func yamlFields(t reflect.Type) []string {
	fields := make([]string, 0, t.NumField())
	for i := 0; i < t.NumField(); i++ {
		name, _, _ := strings.Cut(t.Field(i).Tag.Get("yaml"), ",")
		if name != "" && name != "-" {
			fields = append(fields, name)
		}
	}
	return fields
}

// fieldSets returns the yaml field names of every fieldSetTypes entry.
// TRLC-LINKS: REQ-EMG-087
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func fieldSets() map[string][]string {
	out := make(map[string][]string, len(fieldSetTypes))
	for kind, value := range fieldSetTypes {
		out[kind] = yamlFields(reflect.TypeOf(value))
	}
	return out
}

// AuthoringRules returns the validation vocabularies and relationship pairs as
// sorted, deterministic data for the model authoring contract.
// TRLC-LINKS: REQ-EMG-045, REQ-EMG-009, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-087
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
	return Rules{
		Enums: enums, RelationshipPairs: pairs, FieldRules: append([]string(nil), authoringFieldRules...),
		FieldSets: fieldSets(), PublicationSections: publicationSections(),
	}
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
