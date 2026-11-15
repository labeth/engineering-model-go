// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"regexp"
	"sort"
	"strings"
	"time"

	"github.com/labeth/engineering-model-go/internal/model"
)

var allowedControlImplementationTypes = map[string]bool{
	"technical":  true,
	"procedural": true,
	"inherited":  true,
	"hybrid":     true,
}

var allowedControlImplementationStatuses = map[string]bool{
	"planned":     true,
	"partial":     true,
	"implemented": true,
	"inherited":   true,
}

var oscalControlIDRe = regexp.MustCompile(`(?i)^[a-z]{1,4}-\d+((\(\d+\))|(\.\d+))*$`)
var isoDateRe = regexp.MustCompile(`^\d{4}-\d{2}-\d{2}$`)

var allowedRiskStatus = map[string]bool{
	"open":       true,
	"mitigating": true,
	"accepted":   true,
	"closed":     true,
}

var allowedRiskLevel = map[string]bool{
	"low":    true,
	"medium": true,
	"high":   true,
}

var allowedRiskResponse = map[string]bool{
	"mitigate": true,
	"accept":   true,
	"transfer": true,
	"avoid":    true,
}

var allowedPOAMStatus = map[string]bool{
	"planned":     true,
	"in-progress": true,
	"completed":   true,
	"deferred":    true,
}

var allowedDecisionStatus = map[string]bool{
	"proposed":   true,
	"accepted":   true,
	"superseded": true,
	"deprecated": true,
	"rejected":   true,
}

var allowedRequirementVerificationMethods = map[string]bool{
	"analysis":      true,
	"demonstration": true,
	"inspection":    true,
	"review":        true,
	"test":          true,
}

var allowedRequirementStatus = map[string]bool{
	"draft":       true,
	"proposed":    true,
	"approved":    true,
	"implemented": true,
	"verified":    true,
	"rejected":    true,
	"retired":     true,
}

var allowedDocumentStatus = map[string]bool{
	"draft":      true,
	"in-review":  true,
	"approved":   true,
	"released":   true,
	"superseded": true,
	"withdrawn":  true,
}

var allowedViewKinds = map[string]bool{
	"architecture-intent": true,
	"communication":       true,
	"deployment":          true,
	"security":            true,
	"traceability":        true,
	"state-lifecycle":     true,
	"interaction-flow":    true,
	"use-case":            true,
	"logical":             true,
	"process":             true,
	"physical":            true,
	"implementation":      true,
}

var allowedMappingTypes = map[string]bool{
	"contains":       true,
	"depends_on":     true,
	"interacts_with": true,
	"targets":        true,
	"calls":          true,
	"publishes":      true,
	"subscribes":     true,
	"reads":          true,
	"writes":         true,
	"streams":        true,
	"implements":     true,
	"satisfies":      true,
	"allocated_to":   true,
	"verified_by":    true,
	"guarded_by":     true,
	"deployed_to":    true,
	"mitigated_by":   true,
	"bounded_by":     true,
}

var allowedViewMappingTypes = map[string]bool{
	"contains":       true,
	"depends_on":     true,
	"interacts_with": true,
	"targets":        true,
	"calls":          true,
	"publishes":      true,
	"subscribes":     true,
	"reads":          true,
	"writes":         true,
	"streams":        true,
	"implements":     true,
	"satisfies":      true,
	"allocated_to":   true,
	"verified_by":    true,
	"guarded_by":     true,
	"deployed_to":    true,
	"mitigated_by":   true,
	"bounded_by":     true,
	"transition":     true,
	"on_event":       true,
	"flow_next":      true,
	"flow_error":     true,
	"flow_async":     true,
	"flow_ref":       true,
}

var allowedViewEntityKinds = map[string]bool{
	"hardware_item":        true,
	"hardware_interface":   true,
	"functional_group":     true,
	"functional_unit":      true,
	"actor":                true,
	"attack_vector":        true,
	"referenced_element":   true,
	"interface":            true,
	"data_object":          true,
	"deployment_target":    true,
	"control":              true,
	"trust_boundary":       true,
	"machine":              true,
	"state":                true,
	"event":                true,
	"flow":                 true,
	"flow_step":            true,
	"threat_scenario":      true,
	"threat_assumption":    true,
	"threat_out_of_scope":  true,
	"threat_mitigation":    true,
	"control_verification": true,
	"risk":                 true,
	"poam_item":            true,
}

var allowedFlowStepKinds = map[string]bool{
	"user_action":          true,
	"system_action":        true,
	"data_move":            true,
	"decision":             true,
	"external_interaction": true,
	"security_check":       true,
	"control_action":       true,
	"deployment_action":    true,
	"notification":         true,
	"integration_call":     true,
}

var allowedFlowKinds = map[string]bool{
	"interaction": true,
	"data":        true,
	"control":     true,
	"deployment":  true,
	"state":       true,
	"event":       true,
	"security":    true,
	"business":    true,
}

var allowedFlowTypes = map[string]bool{
	"interaction":  true,
	"data":         true,
	"control":      true,
	"deployment":   true,
	"state":        true,
	"event":        true,
	"security":     true,
	"notification": true,
}

var allowedThreatScenarioStatus = map[string]bool{
	"identified": true,
	"triaged":    true,
	"mitigating": true,
	"accepted":   true,
	"resolved":   true,
}

var allowedThreatAssumptionStatus = map[string]bool{
	"draft":    true,
	"accepted": true,
	"rejected": true,
	"expired":  true,
}

var allowedThreatOutOfScopeStatus = map[string]bool{
	"proposed": true,
	"approved": true,
	"expired":  true,
}

var allowedThreatMitigationStatus = map[string]bool{
	"planned":     true,
	"partial":     true,
	"implemented": true,
	"verified":    true,
	"deferred":    true,
}

var allowedThreatMitigationEffectiveness = map[string]bool{
	"low":     true,
	"medium":  true,
	"high":    true,
	"unknown": true,
}

var allowedControlVerificationMethod = map[string]bool{
	"test":       true,
	"analysis":   true,
	"inspection": true,
	"exercise":   true,
	"audit":      true,
	"monitoring": true,
}

var allowedControlVerificationStatus = map[string]bool{
	"not-run": true,
	"pass":    true,
	"fail":    true,
	"partial": true,
	"blocked": true,
}

var allowedFlowDirection = map[string]bool{
	"inbound":       true,
	"outbound":      true,
	"bidirectional": true,
	"internal":      true,
}

var allowedFlowFrequency = map[string]bool{
	"realtime":  true,
	"batch":     true,
	"scheduled": true,
	"on-demand": true,
}

// documentVersion is the schemaVersion of one document the bundle holds.
type documentVersion struct {
	path    string
	version int
}

// schemaVersions lists the schemaVersion of every document the bundle holds:
// the five that are always present, and the manifest, behavior, assurance,
// compliance, views, aviation and reqLinks documents of a bundle that was
// loaded from files (their path is set) or that carries them. A bundle built
// in code without those documents is not judged on documents it does not hold.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func schemaVersions(b model.Bundle) []documentVersion {
	out := []documentVersion{
		{"schemaVersion", b.Architecture.SchemaVersion},
		{"catalog.schemaVersion", b.Catalog.SchemaVersion},
		{"requirements.schemaVersion", b.Requirements.SchemaVersion},
		{"design.schemaVersion", b.Design.SchemaVersion},
		{"decisions.schemaVersion", b.Decisions.SchemaVersion},
	}
	optional := []struct {
		held bool
		documentVersion
	}{
		{b.ManifestPath != "", documentVersion{"manifest.schemaVersion", b.Manifest.SchemaVersion}},
		{b.BehaviorPath != "", documentVersion{"behavior.schemaVersion", b.Behavior.SchemaVersion}},
		{b.AssurancePath != "", documentVersion{"assurance.schemaVersion", b.Assurance.SchemaVersion}},
		{b.CompliancePath != "", documentVersion{"compliance.schemaVersion", b.Compliance.SchemaVersion}},
		{b.ViewsPath != "", documentVersion{"views.schemaVersion", b.Views.SchemaVersion}},
		{b.Aviation != nil, documentVersion{"aviation.schemaVersion", aviationSchemaVersion(b)}},
		{b.ReqLinksPath != "", documentVersion{"reqLinks.schemaVersion", b.ReqLinks.SchemaVersion}},
	}
	for _, o := range optional {
		if o.held {
			out = append(out, o.documentVersion)
		}
	}
	return out
}

// aviationSchemaVersion is the schemaVersion of the aviation document, 0
// without one.
// TRLC-LINKS: REQ-EMG-044, REQ-EMG-046, REQ-EMG-125, REQ-EMG-166
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func aviationSchemaVersion(b model.Bundle) int {
	if b.Aviation == nil {
		return 0
	}
	return b.Aviation.SchemaVersion
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func Bundle(b model.Bundle) []Diagnostic {
	v := newBundleValidator(b)
	v.checkSchemaVersions()
	v.checkSubsystems()
	v.registerIDs()
	v.checkInferenceHints()
	v.collectAppliesToTargets()
	v.checkHardwareInterfaces()
	v.checkRequirements()
	v.checkDecisions()
	v.checkTrustBoundaries()
	v.checkMappings()
	v.checkComplianceProfiles()
	v.checkComplianceMappings()
	v.checkRisks()
	v.checkPOAMItems()
	v.checkFlows()
	v.checkThreatScenarios()
	v.checkThreatAssumptions()
	v.checkThreatOutOfScope()
	v.checkThreatMitigations()
	v.checkControlVerifications()
	v.checkViews()
	v.checkDocuments()
	asOf := AsOf()
	v.appendPackageDiagnostics(asOf)
	v.appendSemanticDiagnostics()
	v.appendAmbiguousCodeReferences()

	// Only the lifecycle.* entries are judged here: those diagnostics are
	// emitted by Bundle alone. Entries for codes the scanners emit later
	// (health, code, verification, composition) are judged by UnusedDefers on
	// the merged list.
	v.diags = append(v.diags, unusedDefers(b, v.diags, deferScope{asOf: asOf, inScope: func(pattern string) bool { return strings.HasPrefix(pattern, "lifecycle.") }})...)

	return SortDiagnostics(withRemedies(v.diags))
}

// appendPackageDiagnostics adds the diagnostics of the checks that live in their own files: NAF
// profile, behavior, interfaces, catalog names, assurance, acceptance, requirement links,
// life-cycle, required switches, health and aviation.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) appendPackageDiagnostics(asOf time.Time) {
	b := v.b
	authored := b.Architecture.AuthoredArchitecture
	v.diags = append(v.diags, validateNAFProfile(b.Architecture, v.ids.actors)...)
	v.diags = append(v.diags, append(behaviorDiagnostics(authored, b.Architecture.Semantics, v.requirementIDs()), interfaceDiagnostics(authored, v.kindByID)...)...)
	v.diags = append(v.diags, catalogNameDiagnostics(b.Catalog.Catalog)...)
	v.diags = append(v.diags, assuranceLinkDiagnostics(authored)...)
	v.diags = append(v.diags, assuranceConsistencyDiagnostics(authored)...)
	v.diags = append(v.diags, acceptanceDiagnostics(b)...)
	v.diags = append(v.diags, requirementLinkDiagnostics(b, v.validID)...)
	v.diags = append(v.diags, lifecycleDiagnostics(b, asOf)...)
	v.diags = append(v.diags, requiredSwitchDiagnostics(b)...)
	v.diags = append(v.diags, healthDiagnostics(b)...)
	v.diags = append(v.diags, aviationTierDiagnostics(b)...)
}

// requirementIDs is the set of the trimmed requirement ids.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) requirementIDs() map[string]bool {
	ids := map[string]bool{}
	for _, requirement := range v.b.Requirements.Requirements {
		ids[strings.TrimSpace(requirement.ID)] = true
	}
	return ids
}

// appendSemanticDiagnostics adds the errors of the semantic projection and its validation, except a missing id.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) appendSemanticDiagnostics() {
	semantic, projectionDiagnostics := model.ProjectSemanticModel(v.b)
	projectionDiagnostics = append(projectionDiagnostics, model.ValidateSemanticModel(semantic)...)
	for _, diagnostic := range projectionDiagnostics {
		if diagnostic.Severity == model.SemanticSeverityError && diagnostic.Code != "semantic.missing_id" {
			v.fail(diagnostic.Code, diagnostic.Path, diagnostic.Message)
		}
	}
}

// appendAmbiguousCodeReferences reports each CODE id the checks found to be produced more than once.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) appendAmbiguousCodeReferences() {
	for id := range v.ambiguousCodeIDs {
		v.fail("model.ambiguous_code_reference", id, fmt.Sprintf("CODE id %q is produced by more than one file or declaration; rename one or exclude it from the code scope", id))
	}
}

// mappingPairs is the single table of allowed relationship kind pairs; it is
// read by mappingPairAllowed and exported through AuthoringRules.
var mappingPairs = map[string]map[string]bool{
	"contains": {
		"hardware_item:hardware_item":      true,
		"hardware_item:hardware_interface": true,
		"functional_group:functional_unit": true,
		"functional_unit:interface":        true,
		"functional_unit:data_object":      true,
	},
	"depends_on": {
		"functional_unit:functional_unit":    true,
		"functional_unit:referenced_element": true,
		"functional_unit:interface":          true,
		"functional_unit:deployment_target":  true,
	},
	"interacts_with": {
		"hardware_item:hardware_item": true,
		"actor:functional_unit":       true,
		"actor:interface":             true,
	},
	"targets": {
		"attack_vector:functional_unit":    true,
		"attack_vector:referenced_element": true,
		"attack_vector:interface":          true,
		"attack_vector:deployment_target":  true,
	},
	"calls": {
		"functional_unit:functional_unit": true,
		"functional_unit:interface":       true,
	},
	"publishes": {
		"functional_unit:interface":   true,
		"functional_unit:data_object": true,
	},
	"subscribes": {
		"functional_unit:interface":   true,
		"functional_unit:data_object": true,
	},
	"reads": {
		"functional_unit:data_object": true,
	},
	"writes": {
		"functional_unit:data_object": true,
	},
	"streams": {
		"functional_unit:interface":   true,
		"functional_unit:data_object": true,
	},
	"implements": {
		"functional_unit:interface":   true,
		"functional_unit:data_object": true,
	},
	"satisfies": {
		"functional_unit:referenced_element": true,
		"functional_unit:control":            true,
	},
	"allocated_to": {
		"deployment_target:hardware_item":   true,
		"functional_unit:hardware_item":     true,
		"functional_unit:deployment_target": true,
		"interface:deployment_target":       true,
	},
	"verified_by": {
		"functional_unit:referenced_element": true,
		"control:referenced_element":         true,
	},
	"guarded_by": {
		"state:control":              true,
		"state:trust_boundary":       true,
		"state:referenced_element":   true,
		"machine:control":            true,
		"machine:trust_boundary":     true,
		"machine:referenced_element": true,
	},
	"deployed_to": {
		"functional_unit:deployment_target": true,
		"interface:deployment_target":       true,
	},
	"mitigated_by": {
		"attack_vector:control": true,
	},
	"bounded_by": {
		"actor:trust_boundary":             true,
		"hardware_item:trust_boundary":     true,
		"data_object:trust_boundary":       true,
		"functional_unit:trust_boundary":   true,
		"deployment_target:trust_boundary": true,
		"interface:trust_boundary":         true,
	},
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func mappingPairAllowed(mappingType, fromKind, toKind string) bool {
	t := strings.TrimSpace(mappingType)
	from := strings.TrimSpace(fromKind)
	to := strings.TrimSpace(toKind)
	if t == "" || from == "" || to == "" {
		return false
	}
	return mappingPairs[t][from+":"+to]
}

// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func firstNonEmpty(values ...string) string {
	for _, value := range values {
		if strings.TrimSpace(value) != "" {
			return value
		}
	}
	return ""
}

// unreferenceableInferredID reports inferred identifiers authored links must not use:
// runtime RT-* ids and line-based AUTO-* ids. Path-derived CODE-* ids are resolved instead.
// TRLC-LINKS: REQ-EMG-059
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func unreferenceableInferredID(id string) bool {
	id = strings.TrimSpace(id)
	return strings.HasPrefix(id, "RT-") || strings.HasPrefix(id, "AUTO-")
}

var uriSchemeRe = regexp.MustCompile(`^[A-Za-z][A-Za-z0-9+.-]+:`)
var commitIDRe = regexp.MustCompile(`(^|[^0-9a-fA-F])[0-9a-fA-F]{40}([^0-9a-fA-F]|$)`)

// referencedElementURIDiagnostics keeps referenced elements for artifacts outside the
// repository: a local path must be a CODE reference, and a mutable uri needs a version.
// TRLC-LINKS: REQ-EMG-060
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func referencedElementURIDiagnostics(r model.ReferencedElement, path string) []Diagnostic {
	uri := strings.TrimSpace(r.URI)
	if uri == "" {
		return nil
	}
	scheme := strings.ToLower(uriSchemeRe.FindString(uri))
	if scheme == "" || scheme == "file:" {
		return []Diagnostic{{Code: "model.referenced_element_local_uri", Severity: SeverityError, Message: fmt.Sprintf("referenced element %q points at local path %q; reference the file's CODE id instead", r.ID, uri), Path: path + ".uri"}}
	}
	immutable := scheme == "doi:" || commitIDRe.MatchString(uri)
	if !immutable && strings.TrimSpace(r.Version) == "" {
		return []Diagnostic{{Code: "model.referenced_element_missing_version", Severity: SeverityError, Message: fmt.Sprintf("referenced element %q uri %q can change; set version or pin the uri to a commit or doi", r.ID, uri), Path: path + ".version"}}
	}
	return nil
}

// declaredAliasList returns the engmod.yml dependency aliases, sorted and
// comma separated, or "none".
// TRLC-LINKS: REQ-EMG-126
func declaredAliasList(b model.Bundle) string {
	var aliases []string
	for _, dep := range b.Manifest.Dependencies {
		if alias := strings.TrimSpace(dep.Alias); alias != "" {
			aliases = append(aliases, alias)
		}
	}
	if len(aliases) == 0 {
		return "none"
	}
	sort.Strings(aliases)
	return strings.Join(aliases, ", ")
}

// invalidInterfaceOwner reports whether an interface owner is set and is
// neither a functional unit nor a referenced element (the element an external
// interface belongs to, ADR-EMG-085).
// TRLC-LINKS: REQ-EMG-207
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func invalidInterfaceOwner(owner string, units, references map[string]bool) bool {
	return owner != "" && !units[owner] && !references[owner]
}

// validInteractionEnds reports whether an interacts_with relationship goes
// from an actor to a functional unit or to an interface, or from a hardware
// item to a hardware item.
// TRLC-LINKS: REQ-EMG-207
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func validInteractionEnds(m model.Mapping, kindByID map[string]string) bool {
	from, to := kindByID[m.From], kindByID[m.To]
	if from == "actor" {
		return to == "functional_unit" || to == "interface"
	}
	return from == "hardware_item" && to == "hardware_item"
}
