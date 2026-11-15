// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// checkHardwareInterfaces requires each hardware interface to connect hardware items.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkHardwareInterfaces() {
	for i, x := range v.b.Architecture.AuthoredArchitecture.HardwareInterfaces {
		v.checkHardwareInterface(indexedPath("authoredArchitecture.hardwareInterfaces", i), x)
	}
}

// checkHardwareInterface requires both ends to be hardware items and the software interface to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkHardwareInterface(path string, x model.HardwareInterface) {
	for _, end := range []struct{ field, endpoint string }{{"from", x.From}, {"to", x.To}} {
		if v.kindByID[end.endpoint] != "hardware_item" {
			v.fail("model.invalid_hardware_endpoint", path+"."+end.field, fmt.Sprintf("hardware interface %q %s must reference a hardware item", x.ID, end.field))
		}
	}
	if x.SoftwareInterfaceRef != "" && !v.ids.interfaces[x.SoftwareInterfaceRef] {
		v.fail("model.invalid_hardware_software_interface", path+".softwareInterfaceRef", fmt.Sprintf("hardware interface %q references unknown software interface %q", x.ID, x.SoftwareInterfaceRef))
	}
}

// checkTrustBoundaries requires each parent and member of a trust boundary to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkTrustBoundaries() {
	for i, boundary := range v.b.Architecture.AuthoredArchitecture.TrustBoundaries {
		path := indexedPath("authoredArchitecture.trustBoundaries", i)
		if parent := strings.TrimSpace(boundary.ParentRef); parent != "" && !v.ids.trustBoundaries[parent] {
			v.fail("model.invalid_trust_boundary_parent", path, fmt.Sprintf("unknown trust boundary parent %q", parent))
		}
		v.refList(path, boundary.Members, refSpec{ok: v.validID, code: "model.invalid_trust_boundary_member", format: "unknown trust boundary member %q", field: "members"})
	}
}

// checkMappings validates each authored mapping.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkMappings() {
	for i, m := range v.b.Architecture.AuthoredArchitecture.Mappings {
		path := indexedPath("authoredArchitecture.mappings", i)
		v.checkMappingType(path, m)
		v.checkMappingEnds(path, m)
		v.checkMappingRules(path, m)
		v.checkMappingPair(path, m)
	}
}

// removedStateMapping reports whether a mapping uses a state relationship type that was removed.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func removedStateMapping(m model.Mapping) bool {
	return m.Type == "transitions_to" || m.Type == "triggered_by"
}

// checkMappingType requires a known mapping type.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkMappingType(path string, m model.Mapping) {
	if allowedMappingTypes[m.Type] {
		return
	}
	message := fmt.Sprintf("unknown mapping type %q", m.Type)
	if removedStateMapping(m) {
		message = fmt.Sprintf("mapping type %q %s", m.Type, model.LegacyStateRemovedMessage)
	}
	v.fail("model.unknown_mapping_type", path, message)
}

// checkMappingEnds requires both ends of a mapping to exist and not to be inferred ids.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkMappingEnds(path string, m model.Mapping) {
	if !v.validID(m.From) {
		v.fail("model.broken_reference", path, fmt.Sprintf("mapping from %q does not exist", m.From))
	}
	if !v.validID(m.To) {
		v.fail("model.broken_reference", path, fmt.Sprintf("mapping to %q does not exist", m.To))
	}
	if unreferenceableInferredID(m.From) || unreferenceableInferredID(m.To) {
		v.fail("model.inferred_id_not_allowed", path, "authored mappings must not reference inferred RT-* or AUTO-* ids")
	}
}

// checkMappingRules checks what the interacts_with, targets and guarded_by relationships may connect.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkMappingRules(path string, m model.Mapping) {
	if m.Type == "interacts_with" && !validInteractionEnds(m, v.kindByID) {
		v.fail("model.invalid_interaction", path, "interacts_with must be actor -> functional unit, actor -> interface or hardware item -> hardware item")
	}
	if m.Type == "targets" && !v.ids.vectors[m.From] {
		v.fail("model.invalid_target", path, "targets must originate from an attack vector")
	}
	if m.Type == "guarded_by" && !v.isGuard(m.To) {
		v.fail("model.invalid_guard", path, "guarded_by must target a control, trust boundary, or referenced element")
	}
}

// isGuard reports whether an id is a control, a trust boundary or a referenced element.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) isGuard(id string) bool {
	return v.ids.controls[id] || v.ids.trustBoundaries[id] || v.ids.references[id]
}

// checkMappingPair requires the kinds of both ends to be an allowed pair of the mapping type.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkMappingPair(path string, m model.Mapping) {
	fromKind, toKind := v.kindByID[m.From], v.kindByID[m.To]
	if removedStateMapping(m) || fromKind == "" || toKind == "" {
		return
	}
	if !mappingPairAllowed(m.Type, fromKind, toKind) {
		v.fail("model.invalid_mapping_pair", path, fmt.Sprintf("mapping type %q is not valid for %s -> %s", m.Type, fromKind, toKind))
	}
}

// checkComplianceProfiles requires an id and a source of each compliance profile.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkComplianceProfiles() {
	for i, p := range v.b.Architecture.Compliance.Profiles {
		path := indexedPath("compliance.profiles", i)
		v.requireText(path, p.ID, textSpec{"model.missing_compliance_profile_id", "compliance profile id is required"})
		if strings.TrimSpace(p.Href) == "" && strings.TrimSpace(p.CatalogHref) == "" {
			v.fail("model.missing_compliance_profile_source", path, "compliance profile must set href or catalogHref")
		}
	}
}

// checkComplianceMappings validates each compliance mapping.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkComplianceMappings() {
	for i, m := range v.b.Architecture.Compliance.Mappings {
		path := indexedPath("compliance.mappings", i)
		v.checkComplianceRefs(path, m)
		v.checkComplianceControlIDs(path, m)
		v.checkComplianceScope(path, m)
		v.checkComplianceImplementation(path, m)
	}
}

// checkComplianceRefs requires the profile and the model control a mapping names to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkComplianceRefs(path string, m model.ComplianceMapping) {
	if profileRef := strings.TrimSpace(m.ProfileRef); profileRef == "" {
		v.fail("model.missing_compliance_profile_ref", path, "compliance mapping must set profileRef")
	} else if !v.ids.complianceProfiles[profileRef] {
		v.fail("model.invalid_compliance_profile_ref", path, fmt.Sprintf("compliance mapping references unknown profile %q", profileRef))
	}
	if controlRef := strings.TrimSpace(m.ModelControlRef); controlRef == "" {
		v.fail("model.missing_compliance_model_control_ref", path, "compliance mapping must set modelControlRef")
	} else if !v.ids.controls[controlRef] {
		v.fail("model.invalid_compliance_model_control_ref", path, fmt.Sprintf("compliance mapping references unknown model control %q", controlRef))
	}
}

// checkComplianceControlIDs requires at least one well-formed control id.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkComplianceControlIDs(path string, m model.ComplianceMapping) {
	if len(m.ControlIDs) == 0 {
		v.fail("model.empty_compliance_control_ids", path, "compliance mapping must include at least one controlIds entry")
	}
	v.refList(path, m.ControlIDs, refSpec{ok: oscalControlIDRe.MatchString, code: "model.invalid_compliance_control_id", format: "invalid compliance control id %q", field: "controlIds"})
}

// checkComplianceScope requires a scope of existing ids.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkComplianceScope(path string, m model.ComplianceMapping) {
	if len(m.AppliesTo) == 0 {
		v.fail("model.empty_compliance_mapping_scope", path, "compliance mapping must include at least one appliesTo id")
	}
	v.refList(path, m.AppliesTo, refSpec{ok: v.validID, code: "model.invalid_compliance_mapping_target", format: "unknown appliesTo id %q", field: "appliesTo"})
}

// checkComplianceImplementation checks the implementation type, status, roles and evidence.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkComplianceImplementation(path string, m model.ComplianceMapping) {
	v.enumValue(path, m.ImplementationType, enumSpec{allowedControlImplementationTypes, "model.invalid_compliance_implementation_type", "unknown implementationType %q", nil})
	status := firstNonEmpty(strings.ToLower(strings.TrimSpace(m.ImplementationStatus)), strings.ToLower(strings.TrimSpace(m.Status)))
	v.enumValue(path, status, enumSpec{allowedControlImplementationStatuses, "model.invalid_compliance_mapping_status", "unknown implementation status %q", nil})
	v.refList(path, m.ResponsibleRoles, refSpec{ok: inSet(v.ids.actors), code: "model.invalid_compliance_responsible_role", format: "unknown responsible role %q", field: "responsibleRoles"})
	v.blankPaths(path, evidencePathsOf(m.Evidence, func(e model.ControlEvidence) string { return e.Path }), evidenceSpec{"model.empty_compliance_evidence_path", "compliance mapping evidence path is required"})
}

// collectDependencyAliases lists the dependency aliases of the manifest.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) collectDependencyAliases() {
	for _, dep := range v.b.Manifest.Dependencies {
		if alias := strings.TrimSpace(dep.Alias); alias != "" {
			v.dependencyAliases[alias] = true
		}
	}
}

// checkRisks validates each risk.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkRisks() {
	v.collectDependencyAliases()
	for i, r := range v.b.Architecture.AuthoredArchitecture.Risks {
		path := indexedPath("authoredArchitecture.risks", i)
		v.checkRiskFields(path, r)
		v.checkRiskRefs(path, r)
	}
}

// checkRiskFields requires a risk's title and statement and its status and level vocabularies.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkRiskFields(path string, r model.Risk) {
	v.requireText(path, r.Title, textSpec{"model.missing_risk_title", "risk title is required"})
	v.requireText(path, r.Statement, textSpec{"model.missing_risk_statement", "risk statement is required"})
	v.enumValue(path, r.Status, enumSpec{allowedRiskStatus, "model.invalid_risk_status", "unknown risk status %q", nil})
	v.enumValue(path, r.Likelihood, enumSpec{allowedRiskLevel, "model.invalid_risk_likelihood", "unknown likelihood %q", nil})
	v.enumValue(path, r.Impact, enumSpec{allowedRiskLevel, "model.invalid_risk_impact", "unknown impact %q", nil})
	v.enumValue(path, r.ResidualRisk, enumSpec{allowedRiskLevel, "model.invalid_residual_risk", "unknown residualRisk %q", nil})
	v.enumValue(path, r.Response, enumSpec{allowedRiskResponse, "model.invalid_risk_response", "unknown response %q", nil})
}

// checkRiskRefs requires a risk's owner, scope, dependencies and links to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkRiskRefs(path string, r model.Risk) {
	if owner := strings.TrimSpace(r.Owner); owner != "" && !v.ids.actors[owner] {
		v.fail("model.invalid_risk_owner", path, fmt.Sprintf("unknown risk owner %q", owner))
	}
	v.refList(path, r.AppliesTo, refSpec{ok: v.validID, code: "model.invalid_risk_scope_target", format: "unknown appliesTo id %q", field: "appliesTo"})
	v.checkRiskDependencies(path, r)
	v.refList(path, r.RelatedControls, refSpec{ok: inSet(v.ids.controls), code: "model.invalid_risk_control_ref", format: "unknown related control %q", field: "relatedControls"})
	v.refList(path, r.AttackVectors, refSpec{ok: inSet(v.ids.vectors), code: "model.invalid_risk_attack_vector_ref", format: "unknown attack vector %q", field: "attackVectors"})
	v.refList(path, r.ThreatScenarios, refSpec{ok: inSet(v.ids.threatScenarios), code: "model.invalid_risk_threat_scenario_ref", format: "unknown threat scenario %q", field: "threatScenarios"})
	v.blankPaths(path, evidencePathsOf(r.Evidence, func(e model.RiskEvidence) string { return e.Path }), evidenceSpec{"model.empty_risk_evidence_path", "risk evidence path is required"})
}

// checkRiskDependencies requires each dependency a risk names to be an alias declared in engmod.yml.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkRiskDependencies(path string, r model.Risk) {
	for j, raw := range r.Dependencies {
		alias := strings.TrimSpace(raw)
		if alias == "" || !v.dependencyAliases[alias] {
			v.fail("model.invalid_risk_dependency", indexedPath(path+".dependencies", j), fmt.Sprintf("risk %q names %q in dependencies, which is no dependency alias in engmod.yml (declared: %s); name an alias from engmod.yml dependencies", r.ID, alias, declaredAliasList(v.b)))
		}
	}
}

// checkPOAMItems validates each POA&M item.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkPOAMItems() {
	for i, p := range v.b.Architecture.AuthoredArchitecture.POAMItems {
		path := indexedPath("authoredArchitecture.poamItems", i)
		if riskRef := strings.TrimSpace(p.RiskRef); riskRef == "" || !v.ids.risks[riskRef] {
			v.fail("model.invalid_poam_risk_ref", path, fmt.Sprintf("unknown poam riskRef %q", p.RiskRef))
		}
		v.requireText(path, p.Milestone, textSpec{"model.missing_poam_milestone", "poam milestone is required"})
		v.isoDateValue(path, p.DueDate, textSpec{"model.invalid_poam_due_date", "invalid dueDate %q, expected YYYY-MM-DD"})
		v.enumValue(path, p.Status, enumSpec{allowedPOAMStatus, "model.invalid_poam_status", "unknown poam status %q", nil})
		if role := strings.TrimSpace(p.ResponsibleRole); role != "" && !v.ids.actors[role] {
			v.fail("model.invalid_poam_responsible_role", path, fmt.Sprintf("unknown poam responsibleRole %q", role))
		}
		v.checkPOAMArtifacts(path, p)
	}
}

// checkPOAMArtifacts requires a path of each POA&M artifact.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkPOAMArtifacts(path string, p model.POAMItem) {
	for j, art := range p.Artifacts {
		v.requireText(indexedPath(path+".artifacts", j), art.Path, textSpec{"model.empty_poam_artifact_path", "poam artifact path is required"})
	}
}
