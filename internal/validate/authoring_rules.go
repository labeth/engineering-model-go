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
	// FieldTypes gives, per entity kind and in the order of FieldSets, each
	// field's YAML type: string, bool, int, list<string>, list<object>,
	// object or map<string,string>; Of names the fieldSets kind of an object
	// or list<object> field when it has one.
	FieldTypes map[string][]FieldType `json:"fieldTypes"`
	// PublicationSections are the manifest publication sections that may
	// export stable IDs. Decisions cannot be published.
	PublicationSections []string `json:"publicationSections"`
	// TierPolicy is the severity of each tier-keyed code per life-cycle
	// tier (engmod.yml lifecycle.tier); engineering-flow renders it as the
	// "Severity by tier" table of MODEL-RULES.md.
	TierPolicy []TierRule `json:"tierPolicy"`
}

// FieldType is the YAML type of one field of an entity kind.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
type FieldType struct {
	Name string `json:"name"`
	Type string `json:"type"`
	Of   string `json:"of,omitempty"`
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
	"lifecycle.tier":                         allowedTiers,
	"lifecycle.bundles":                      allowedBundles,
}

// allowedTiers and allowedBundles are the lifecycle vocabularies.
var (
	allowedTiers   = setOf(model.Tiers)
	allowedBundles = setOf(model.Bundles)
)

// setOf returns a set of the values.
// TRLC-LINKS: REQ-EMG-131
func setOf(values []string) map[string]bool {
	out := make(map[string]bool, len(values))
	for _, v := range values {
		out[v] = true
	}
	return out
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
	"State behavior is modeled with behavior.machines only (ADR-EMG-038). behavior.states, transitions_to and the state triggered_by were removed in v0.11.0: a behavior document that has them fails to load with an error naming the field and machines:. A machine has id (SM-*), owner (a functional unit), initial (one of its states), variables (name, type Integer, Boolean or String, initial literal), states (id STATE-*, optional name and description, optional final: true), transitions (id TR-*, from, to, optional on naming a behavior.events event, optional guard, optional effect) and invariants (id INV-*, requirements, expr). Every state belongs to exactly one machine and keeps its STATE-* ID. A guard and an invariant are Boolean expressions and an effect is a list of assignments separated by semicolons, in the operators and, or, not, implies, ==, !=, <, <=, >, >=, +, -, *, := and in_state(STATE-ID); arithmetic and ordering need Integer operands and == and != need operands of one type. guarded_by (protected by a control, trust boundary or referenced element) keeps its meaning; its source is a machine state or a machine.",
	"Assurance links: a threat scenario's verificationRefs list every control verification that its mitigations reference or that names it in threatScenarioRefs (warning threat.verification_missing); a threatMitigation verificationRefs entry names a control verification of the mitigation's own controlRef (error mitigation.verification_control_mismatch); a controlVerification with method test and status pass or partial names its tests in verificationRefs (warning verification.unlinked); every control has a control verification with its controlRef, else the Gemara evaluation log marks it Needs Review (warning gemara.control_needs_review). Consistency: one (likelihood, impact) pair maps to one severity across the threat scenarios (warning risk.severity_inconsistent; risks have residualRisk, no severity); a risk in a controlVerification riskRefs lists that verification's controlRef in relatedControls (warning controlVerification.risk_control_unlinked); once one threat scenario names attackVectorRef, every one does (warning threat.attack_vector_missing).",
	"fieldSets lists the fields each entity kind accepts; strict decoding rejects any other field (for example control takes only id, name, category and description). evidence applies to every evidence, artifact list entry (path, description). fieldTypes gives each field's YAML type in the same order (string, bool, int, list<string>, list<object>, object, map<string,string>) and, in of, the fieldSets kind of an object or list<object> field: for example threatScenario stride is one string while preconditions, exploitPath and detection, and controlVerification findings, are lists of strings; flow steps is a list<object> of flowStep; risk acceptedBy an object of acceptanceRecord.",
	"Flows: every flow step needs ref, an authored ID the step acts on (model.missing_flow_step_ref, error; model.broken_flow_reference when it does not resolve); a step needs id, and step IDs are unique across all flows, not only within one, because each step becomes a model element (a repeat within a flow is model.duplicate_flow_step_id, across flows semantic.duplicate_identity, both errors): prefix them with the flow, for example checkout-submit.",
	"catalog.term_unreferenced (warning, only with lintRun.mode: strict) names an actor, event, state, feature, mode, condition or data catalog term (and a system) that is referenced nowhere in the model: a term counts as referenced when a requirement text names it (its name or an alias, on word boundaries), when its ID appears in any string of the architecture, behavior, assurance, compliance, views, decisions, design or aviation documents (for example a flow step ref or a state ID), or when its name or alias appears in other authored text (requirement notes or criteria, another catalog entry, any document string). A behavior state with the same ID as the catalog term references it.",
	"Code scope: the module root is scanned minus git-ignored files, nested modules and inferenceHints.exclude. exclude entries are relative to the module root: dir, dir/, dir/* or dir/** each leave the whole directory out, ./dir is the same, a file path leaves that file out, and a glob with * ? [...] matches within one path segment (*.gen.go); an absolute path, .., ** other than a trailing /** and a malformed glob are model.invalid_exclude_pattern (error). The MCP code tools use the same scope.",
	"Publications: a manifest publications entry has an id and the sections architecture, behavior, assurance, compliance, requirements and views (publicationSections), each a list of stable IDs from that document. Decisions cannot be published, so a consumer cannot reference alias::DEC-*; keep a shared decision in the repository that owns it and quote it in the consumer.",
	"A flow step with boundaryCrossing: true needs trustBoundaryRef naming an existing trust boundary (error model.missing_flow_boundary_ref), and should name an onError step (warning behavior.flow_step_no_onerror).",
	"behavior.event_unused counts as a use of an event: the on of a machine transition; the source, target or a triggers entry of an architecture.semantics.relationships entry; the sourceRef or destinationRef of a flow; and the ref, sourceRef or destinationRef of a flow step. Flow or step properties, descriptions and other prose do not count.",
	"Acceptance: a risk with status accepted or response accept, and a threat scenario with status accepted, needs acceptedBy with actor (an ACT-* actor), date (a YYYY-MM-DD calendar date) and statement (the owner's own words) or decisionRef (a DEC-* decision with status accepted whose context quotes the owner); without it validation reports risk.acceptance_unrecorded (error), and an actor, date or decisionRef that does not resolve is model.invalid_acceptance (error). An acceptedBy actor other than the risk's owner is risk.acceptance_not_owner (warning). response accept with status open is risk.accept_while_open (warning). A record with decisionRef but no statement is risk.acceptance_by_decision_only (warning): put the owner's own words accepting this risk in statement. The Gemara policy lists a risk under risks.accepted only with a complete, resolving acceptedBy and status accepted, or response accept with a status other than open. Authored risk and threat scenario IDs must not contain :: (reserved for imported alias::ID entities). Never record an acceptance the owner did not give.",
	"lint.multiple_responses exempts a single response with a field list: the text has exactly one shall and no semicolon, the list opens at the response's first with, including or of, no and, or or comma comes before it, the list holds no then, and no list item after the first has the, a, an, its, their, each, every or all as its second word (that reads as a verb with its object). Passes: 'The flight logger shall publish one record with UTC time, latitude and altitude.' Reported: 'The flight logger shall store the record and notify the operator.'",
	"In strict EARS mode (lintRun.mode: strict) the system term of a requirement (the words before shall) must be the name or an alias of a catalog system, functional group or functional unit, written as in the catalog (for example 'the engineering model go shall ...'); a repository slug such as sim-interface is not a catalog name unless the catalog names it so.",
	"Requirement links, upward trace: a requirement names in realizes the local contract items it exists to deliver or make true, a contract.provides entry of kind capability (CAP-*) or a contract.assumptions assumption of use (AOU-*); never a requirement, an imported alias::ID or an interface entry (requirement.realizes_unknown, error). No model can know it is top level, because any system may later be used by one above, so every requirement traces through a capability: it satisfies the upward trace iff (a) it realizes a local capability or assumption of use, (b) it is the parent of a reqLinks entry to a dependency's capability, or (c) it is derived: true with derivedRationale; otherwise requirement.no_parent. Stakeholder sourceRefs do not count. A system publishes its own capabilities so a system above can link to them. A child never names its users or a system requirement.",
	"Contract items: contract.assumptions entries have id (AOU-*), text and optional verificationCriteria and are published in the architecture list like capabilities, imported as alias::AOU-*. contract.provides ref is deprecated (contract.ref_deprecated, warning): add realizes to the requirement instead. A published capability no requirement realizes is contract.unrealized (warning). Every requirement that realizes a published contract item must be in the requirements list of that publication (contract.realizer_unpublished, error), because a using system compares the item with its realizers between versions.",
	"Requirement link file (documents.reqLinks, usually model/req-links.yml): reqLinks entries {id (LNK-*), parent (a local requirement), child (alias::CAP-* or alias::AOU-* of a composed dependency), evidence (local requirement, control, verification or test IDs; required for an assumption of use), pinned {version (the published dependency version the link was reviewed at; equal to the dependency version in engmod.yml once reviewed)}, status (claimed or accepted), rationale, reviewedBy, reviewedOn (YYYY-MM-DD)}. Write a new link with engws link add (--parent REQ for a capability, --evidence REQ for an assumption of use); record a review with engws review-links --accept. pinned.fingerprint is deprecated (link.fingerprint_deprecated, warning; engws migrate-links removes it). Validation reports link.parent_unknown, link.child_unpublished, link.version_mismatch (errors; review the link at the new version with engws review-links) and link.claimed_only (warning); the composed check reports link.broken (the item does not exist at the published dependency pin) and link.aou_unverified (a dependency assumption of use with no evidence link). No item content is compared at check time and a workspace checkout raises no link finding: engws bump compares each linked item's meaning (item kind, text and verificationCriteria; realizer text and verificationCriteria; not status or notes) between versions and moves the pin only when it is the same; a capability's note is its text, so editing it needs a review in every user. requirement.unsatisfied is a system requirement with no code, test or link and not derived. composition.allocations is deprecated (composition.allocation_deprecated, warning): use reqLinks.",
	"requirementLinks in engmod.yml is strict or guided (default guided; model.invalid_requirement_links otherwise). strict: requirement.no_parent, link.aou_unverified and requirement.unsatisfied are errors in every model. guided: they are warnings, and only in a model that publishes contract items or has reqLinks. The other link and contract rules have the same severity in both modes. A dependency version is a release tag or a Go-style pseudo-version (v1.2.4-0.20261102103400-abcdef123456).",
	"Life-cycle tier: engmod.yml lifecycle {tier (poc < prototype < mvp < product < operational), bundles (foundation, safety, security; default [foundation]), tierDecision, raise} is optional; without it every severity is that of engineering-model-go v0.9.0. With it, tierPolicy gives each tier-keyed code its severity at the project tier: deferred (a warning with deferred: true and requiredFrom, evidence debt that never fails a check), warning or error. tierDecision names an accepted DEC-TIER-* decision whose Tier: line and Bundles: line (comma separated, read case-insensitively, absent means Foundation) agree with lifecycle (lifecycle.tier_decision_mismatch, error). lifecycle.raise lists codes (or family.* patterns) raised to error regardless of tier; it never lowers. From tier mvp requirement links are strict whatever requirementLinks says; requirementLinks: guided at mvp or above is lifecycle.requirement_links_conflict (error). With lifecycle, lifecycle.requirement_unapproved reports requirements with status draft or proposed and lifecycle.requirement_unverified those not verified (rejected and retired excepted). The safety bundle needs documents.aviation (lifecycle.aviation_profile_missing) with aviation.profile.softwareLevel (lifecycle.software_level_missing), and makes aviation.ai_actor_as_credit an error: an aviation review reviewer, independence or execution producer or verifier, or approval approver names a registered engineering-flow agent (people hold assurance credit, AI agents never). Rank-0 core codes (model.duplicate_id, model.broken_reference, model.invalid_*, model.missing_*, code.dangling_*, composition.cycle, composition.lock_*, the link integrity codes, risk.acceptance_unrecorded and the structural lifecycle.* codes) are errors at every tier and are never deferred.",
	"Composition tier: with lifecycle in a composing system, a composed dependency whose own lifecycle tier is below the system tier is composition.dependency_below_tier (tier keyed) and one without lifecycle composition.dependency_tier_unknown. To accept a below-tier dependency, record a risk with status accepted (or response accept) and a complete, resolving acceptedBy that names the dependency alias in dependencies (risk.dependencies: engmod.yml dependencies[].alias values; preferred), or the composition subsystem that composes it in appliesTo (older form). A dependencies entry that is no declared alias is model.invalid_risk_dependency (error).",
	"POA&M binding: a poamItem defers [{code, path?}] binds the item to the diagnostics it covers. While the item's status is not completed and its milestone is tier:<project tier> (for example tier:product), each diagnostic with that code (exact or family.*) and, when path is given, that path is reported as a warning with deferred: true and poamRef naming the item: the finding is still reported, never exempted. Naming a rank-0 core code is lifecycle.poam_defers_core (error) and defers nothing. Only tier-keyed codes and warnings are deferred; an untiered error keeps its severity; the engine has no clock, so an overdue item is the flow's to raise.",
	"Machine errors, never tier keyed: behavior.transition_unknown_event (on names no behavior event), behavior.expression_unknown_variable (a guard, effect or invariant reads or assigns an undeclared variable), behavior.expression_type (an operand or result type the operator table rejects; guards and invariants must be Boolean and := needs the variable's type), behavior.in_state_foreign (in_state names a state of another machine or no state) and behavior.final_state_has_transitions (a final state has an outgoing transition). A missing or unresolved initial state (initial is a single field), unresolved transition ends, duplicate variables and expression syntax errors are reported by projection as semantic.invalid_machine and semantic.invalid_expression. Machine, state, transition and invariant IDs are unique across the model (model.duplicate_id).",
	"Warnings: behavior.event_unused (a behavior event in no machine transition on, transition trigger or flow), behavior.state_unreachable (a machine state no path of transitions reaches from the initial state), behavior.flow_step_no_onerror (a boundaryCrossing step without onError) and catalog.name_has_operator (a clause-term catalog name or alias with a standalone and, or, not, a comma or a parenthesis, which EARS reads as operators). Tier keyed with the hygiene row (deferred at poc and prototype, warning at mvp and product, error at operational): behavior.state_unreachable, behavior.machine_deadlock (a non-final state without outgoing transition), behavior.transition_nondeterministic (two transitions from one state on one event, or both without, with no guards or the same guard), behavior.transition_no_trigger (a transition without on and without guard) and behavior.invariant_unlinked (an invariant linked to no existing requirement).",
	"Code health (optional): engmod.yml health {exclude, thresholds, overrides, waivers, baseline, policyDecision} (fieldSets health, healthThreshold, healthOverride, healthWaiver, healthBaseline; the health section of the contract lists the codes) configures the code health report; without the block nothing about health is validated or reported and every output is unchanged. exclude entries and overrides[].path use the inferenceHints.exclude pattern dialect (model.invalid_health_exclude, model.invalid_health_override). thresholds and overrides[].thresholds are keyed by health.complex_method, health.large_method, health.deep_nesting, health.bumpy_road, health.complex_conditional, health.excess_arguments or health.large_assertion_block with warn at least 1 and high at least warn (model.invalid_health_threshold). A waiver {code, path, function, decision} names a health smell code (model.invalid_health_waiver), the file, the function and a decision: health.waiver_unjustified (error, untiered, never deferred) when that decision does not exist or is not accepted. health.waiver_unused (a waiver that matches no finding) comes from the health report. Baseline: health.baseline {file, decision} accepts existing debt; file is a path relative to the manifest directory (no absolute path, no .., model.invalid_health_baseline) of the JSON file {schema: engmod-health-baseline/1, entries: [{path, function, code, value}]} that enghealth baseline write creates (health.baseline_unreadable, error and untiered, when it is missing, not valid JSON, of another schema or holds an invalid entry); decision must exist and be accepted (health.waiver_unjustified, the same untiered error, naming the baseline when it is not, and then the baseline suppresses nothing). A baselined finding stays in the score but is neither an error nor counted by the gate while its value is at most the recorded value; health.baseline_stale (an entry that matches no finding) is a warning from poc to product and an error at operational, so the baseline only shrinks; a regression is never baselined. Tier keyed: every health smell code and health.waiver_unused are warnings up to mvp and errors at product and operational (from mvp with the safety bundle; lifecycle.raise: [health.*] makes debt an error at any tier); health.regression (a new or worsened high finding) and health.policy_weakened are warnings at poc to mvp and errors from product, and from mvp with the safety bundle. policyDecision names an accepted decision that records a weakening of the block (a higher limit, a new exclude, an override that raises a limit, a lower tier) against the base revision; enghealth check reports the weakening as health.policy_weakened unless the head names a policyDecision the base did not (health.waiver_unjustified when it does not exist or is not accepted).",
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
	"contractAssumption":  model.ContractAssumption{},
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
	"health":              model.ManifestHealth{},
	"healthBaseline":      model.HealthBaseline{},
	"healthOverride":      model.HealthOverride{},
	"healthThreshold":     model.HealthThreshold{},
	"healthWaiver":        model.HealthWaiver{},
	"interface":           model.Interface{},
	"lifecycle":           model.ManifestLifecycle{},
	"lintWaiver":          model.LintWaiver{},
	"mapping":             model.Mapping{},
	"poamDefer":           model.POAMDefer{},
	"poamItem":            model.POAMItem{},
	"publication":         model.ManifestPublication{},
	"referencedElement":   model.ReferencedElement{},
	"reqLink":             model.ReqLink{},
	"reqLinkPin":          model.ReqLinkPin{},
	"requirement":         model.Requirement{},
	"risk":                model.Risk{},
	"machine":             model.Machine{},
	"machineInvariant":    model.MachineInvariant{},
	"machineState":        model.MachineState{},
	"machineTransition":   model.MachineTransition{},
	"machineVariable":     model.MachineVariable{},
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

// yamlType returns the YAML type name of a strict-decoding Go field type and,
// for a struct element, the fieldSets kind of that struct.
// TRLC-LINKS: REQ-EMG-115
func yamlType(t reflect.Type, kinds map[reflect.Type]string) (string, string) {
	for t.Kind() == reflect.Pointer {
		t = t.Elem()
	}
	switch t.Kind() {
	case reflect.String:
		return "string", ""
	case reflect.Bool:
		return "bool", ""
	case reflect.Int, reflect.Int64, reflect.Int32, reflect.Uint, reflect.Uint64, reflect.Uint32:
		return "int", ""
	case reflect.Float32, reflect.Float64:
		return "number", ""
	case reflect.Interface:
		return "any", ""
	case reflect.Struct:
		return "object", kinds[t]
	case reflect.Map:
		elem, of := yamlType(t.Elem(), kinds)
		return "map<string," + elem + ">", of
	case reflect.Slice, reflect.Array:
		elem, of := yamlType(t.Elem(), kinds)
		return "list<" + elem + ">", of
	}
	return t.Kind().String(), ""
}

// fieldTypes returns, per fieldSetTypes kind, each yaml field with its type,
// in declaration order. Every evidence and artifact struct maps to the kind
// evidence.
// TRLC-LINKS: REQ-EMG-115
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func fieldTypes() map[string][]FieldType {
	kinds := map[reflect.Type]string{}
	for kind, value := range fieldSetTypes {
		kinds[reflect.TypeOf(value)] = kind
	}
	evidence := func(t reflect.Type) {
		if _, ok := kinds[t]; !ok && (strings.HasSuffix(t.Name(), "Evidence") || strings.HasSuffix(t.Name(), "Artifact")) {
			kinds[t] = "evidence"
		}
	}
	for _, value := range fieldSetTypes {
		t := reflect.TypeOf(value)
		for i := 0; i < t.NumField(); i++ {
			ft := t.Field(i).Type
			for ft.Kind() == reflect.Pointer || ft.Kind() == reflect.Slice {
				ft = ft.Elem()
			}
			if ft.Kind() == reflect.Struct {
				evidence(ft)
			}
		}
	}
	out := make(map[string][]FieldType, len(fieldSetTypes))
	for kind, value := range fieldSetTypes {
		t := reflect.TypeOf(value)
		fields := []FieldType{}
		for i := 0; i < t.NumField(); i++ {
			name, _, _ := strings.Cut(t.Field(i).Tag.Get("yaml"), ",")
			if name == "" || name == "-" {
				continue
			}
			typ, of := yamlType(t.Field(i).Type, kinds)
			fields = append(fields, FieldType{Name: name, Type: typ, Of: of})
		}
		out[kind] = fields
	}
	return out
}

// AuthoringRules returns the validation vocabularies and relationship pairs as
// sorted, deterministic data for the model authoring contract.
// TRLC-LINKS: REQ-EMG-045, REQ-EMG-009, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-087, REQ-EMG-099, REQ-EMG-131
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
		FieldSets: fieldSets(), FieldTypes: fieldTypes(), PublicationSections: publicationSections(),
		TierPolicy: TierPolicy(),
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
