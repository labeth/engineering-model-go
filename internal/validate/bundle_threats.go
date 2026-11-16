// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// rawRefSpec describes a reference whose message names the value as written;
// a blank value is an error only when required.
type rawRefSpec struct {
	set          map[string]bool
	code, format string
	required     bool
}

// rawRef records the spec's error at path when the trimmed reference is not in the set.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) rawRef(path, value string, spec rawRefSpec) {
	ref := strings.TrimSpace(value)
	bad := !spec.set[ref]
	if ref == "" {
		bad = spec.required
	}
	if bad {
		v.fail(spec.code, path, fmt.Sprintf(spec.format, value))
	}
}

// checkThreatScenarios validates each threat scenario.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkThreatScenarios() {
	for i, ts := range v.b.Architecture.AuthoredArchitecture.ThreatScenarios {
		path := indexedPath("authoredArchitecture.threatScenarios", i)
		v.checkThreatScenarioFields(path, ts)
		v.checkThreatScenarioTargets(path, ts)
		v.checkThreatScenarioLinks(path, ts)
	}
}

// checkThreatScenarioFields requires a title and restricts status and levels to their vocabularies.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkThreatScenarioFields(path string, ts model.ThreatScenario) {
	v.requireText(path, ts.Title, textSpec{"model.missing_threat_scenario_title", "threat scenario title is required"})
	v.enumValue(path, ts.Status, enumSpec{allowedThreatScenarioStatus, "model.invalid_threat_scenario_status", "unknown threat scenario status %q", nil})
	v.enumValue(path, ts.Likelihood, enumSpec{allowedRiskLevel, "model.invalid_threat_scenario_likelihood", "unknown likelihood %q", nil})
	v.enumValue(path, ts.Impact, enumSpec{allowedRiskLevel, "model.invalid_threat_scenario_impact", "unknown impact %q", nil})
	v.enumValue(path, ts.Severity, enumSpec{allowedRiskLevel, "model.invalid_threat_scenario_severity", "unknown severity %q", nil})
}

// checkThreatScenarioTargets requires the attack vector, owner, risk, scope and flows of a scenario to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkThreatScenarioTargets(path string, ts model.ThreatScenario) {
	v.optionalRef(path, ts.AttackVectorRef, refSpec{ok: inSet(v.ids.vectors), code: "model.invalid_threat_scenario_attack_vector", format: "threat scenario references unknown attack vector %q"})
	v.optionalRef(path, ts.Owner, refSpec{ok: inSet(v.ids.actors), code: "model.invalid_threat_scenario_owner", format: "unknown threat scenario owner %q"})
	v.optionalRef(path, ts.RiskRef, refSpec{ok: inSet(v.ids.risks), code: "model.invalid_threat_scenario_risk_ref", format: "threat scenario references unknown risk %q"})
	v.refList(path, ts.AppliesTo, refSpec{ok: v.validID, code: "model.invalid_threat_scenario_scope_target", format: "unknown appliesTo id %q", field: "appliesTo"})
	v.refList(path, ts.FlowRefs, refSpec{ok: v.isFlow, code: "model.invalid_threat_scenario_flow_ref", format: "unknown flow ref %q", field: "flowRefs"})
	if len(ts.AppliesTo) == 0 && len(ts.FlowRefs) == 0 {
		v.fail("model.empty_threat_scenario_scope", path, "threat scenario must include appliesTo or flowRefs")
	}
}

// isFlow reports whether an id is a registered flow.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) isFlow(id string) bool {
	return v.kindByID[id] == "flow"
}

// checkThreatScenarioLinks requires the controls, assumptions, exclusions, mitigations and verifications of a scenario to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkThreatScenarioLinks(path string, ts model.ThreatScenario) {
	v.refList(path, ts.RelatedControls, refSpec{ok: inSet(v.ids.controls), code: "model.invalid_threat_scenario_control_ref", format: "unknown related control %q", field: "relatedControls"})
	v.refList(path, ts.AssumptionRefs, refSpec{ok: inSet(v.ids.threatAssumptions), code: "model.invalid_threat_scenario_assumption_ref", format: "unknown assumption ref %q", field: "assumptionRefs"})
	v.refList(path, ts.OutOfScopeRefs, refSpec{ok: inSet(v.ids.threatOutOfScope), code: "model.invalid_threat_scenario_out_of_scope_ref", format: "unknown out-of-scope ref %q", field: "outOfScopeRefs"})
	v.refList(path, ts.MitigationRefs, refSpec{ok: inSet(v.ids.threatMitigations), code: "model.invalid_threat_scenario_mitigation_ref", format: "unknown mitigation ref %q", field: "mitigationRefs"})
	v.refList(path, ts.VerificationRefs, refSpec{ok: inSet(v.ids.controlVerifications), code: "model.invalid_threat_scenario_verification_ref", format: "unknown verification ref %q", field: "verificationRefs"})
	v.blankPaths(path, evidencePathsOf(ts.Evidence, func(e model.ThreatScenarioEvidence) string { return e.Path }), evidenceSpec{"model.empty_threat_scenario_evidence_path", "threat scenario evidence path is required"})
}

// checkThreatAssumptions validates each threat assumption.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkThreatAssumptions() {
	for i, a := range v.b.Architecture.AuthoredArchitecture.ThreatAssumptions {
		path := indexedPath("authoredArchitecture.threatAssumptions", i)
		v.requireText(path, a.Title, textSpec{"model.missing_threat_assumption_title", "threat assumption title is required"})
		v.requireText(path, a.Statement, textSpec{"model.missing_threat_assumption_statement", "threat assumption statement is required"})
		v.enumValue(path, a.Status, enumSpec{allowedThreatAssumptionStatus, "model.invalid_threat_assumption_status", "unknown threat assumption status %q", nil})
		v.rawRef(path, a.Owner, rawRefSpec{set: v.ids.actors, code: "model.invalid_threat_assumption_owner", format: "unknown threat assumption owner %q"})
		v.refList(path, a.AppliesTo, refSpec{ok: v.validID, code: "model.invalid_threat_assumption_scope_target", format: "unknown appliesTo id %q", field: "appliesTo"})
		v.blankPaths(path, evidencePathsOf(a.Evidence, func(e model.ThreatAssumptionEvidence) string { return e.Path }), evidenceSpec{"model.empty_threat_assumption_evidence_path", "threat assumption evidence path is required"})
	}
}

// checkThreatOutOfScope validates each out-of-scope declaration.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkThreatOutOfScope() {
	for i, o := range v.b.Architecture.AuthoredArchitecture.ThreatOutOfScope {
		path := indexedPath("authoredArchitecture.threatOutOfScope", i)
		v.requireText(path, o.Title, textSpec{"model.missing_threat_out_of_scope_title", "threat out-of-scope title is required"})
		v.requireText(path, o.Reason, textSpec{"model.missing_threat_out_of_scope_reason", "threat out-of-scope reason is required"})
		v.enumValue(path, o.Status, enumSpec{allowedThreatOutOfScopeStatus, "model.invalid_threat_out_of_scope_status", "unknown threat out-of-scope status %q", nil})
		v.rawRef(path, o.Owner, rawRefSpec{set: v.ids.actors, code: "model.invalid_threat_out_of_scope_owner", format: "unknown threat out-of-scope owner %q"})
		v.isoDateValue(path, o.ExpiresOn, textSpec{"model.invalid_threat_out_of_scope_expiry", "invalid expiresOn %q, expected YYYY-MM-DD"})
		v.refList(path, o.AppliesTo, refSpec{ok: v.validID, code: "model.invalid_threat_out_of_scope_target", format: "unknown appliesTo id %q", field: "appliesTo"})
		v.blankPaths(path, evidencePathsOf(o.Evidence, func(e model.ThreatOutOfScopeEvidence) string { return e.Path }), evidenceSpec{"model.empty_threat_out_of_scope_evidence_path", "threat out-of-scope evidence path is required"})
	}
}

// checkThreatMitigations validates each threat mitigation.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkThreatMitigations() {
	for i, m := range v.b.Architecture.AuthoredArchitecture.ThreatMitigations {
		path := indexedPath("authoredArchitecture.threatMitigations", i)
		v.rawRef(path, m.ThreatScenarioRef, rawRefSpec{set: v.ids.threatScenarios, code: "model.invalid_threat_mitigation_scenario_ref", format: "unknown threat scenario ref %q", required: true})
		v.rawRef(path, m.ControlRef, rawRefSpec{set: v.ids.controls, code: "model.invalid_threat_mitigation_control_ref", format: "unknown control ref %q", required: true})
		v.enumValue(path, m.Status, enumSpec{allowedThreatMitigationStatus, "model.invalid_threat_mitigation_status", "unknown threat mitigation status %q", nil})
		v.enumValue(path, m.Effectiveness, enumSpec{allowedThreatMitigationEffectiveness, "model.invalid_threat_mitigation_effectiveness", "unknown threat mitigation effectiveness %q", nil})
		v.rawRef(path, m.Owner, rawRefSpec{set: v.ids.actors, code: "model.invalid_threat_mitigation_owner", format: "unknown threat mitigation owner %q"})
		v.refList(path, m.VerificationRefs, refSpec{ok: inSet(v.ids.controlVerifications), code: "model.invalid_threat_mitigation_verification_ref", format: "unknown control verification ref %q", field: "verificationRefs"})
		v.blankPaths(path, evidencePathsOf(m.Evidence, func(e model.ThreatMitigationEvidence) string { return e.Path }), evidenceSpec{"model.empty_threat_mitigation_evidence_path", "threat mitigation evidence path is required"})
	}
}

// checkControlVerifications validates each control verification.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkControlVerifications() {
	for i, c := range v.b.Architecture.AuthoredArchitecture.ControlVerifications {
		path := indexedPath("authoredArchitecture.controlVerifications", i)
		v.checkControlVerificationFields(path, c)
		v.checkControlVerificationRefs(path, c)
	}
}

// checkControlVerificationFields checks a verification's control, vocabularies, owner and date.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkControlVerificationFields(path string, c model.ControlVerification) {
	v.rawRef(path, c.ControlRef, rawRefSpec{set: v.ids.controls, code: "model.invalid_control_verification_control_ref", format: "unknown control ref %q", required: true})
	v.enumValue(path, c.Method, enumSpec{allowedControlVerificationMethod, "model.invalid_control_verification_method", "unknown control verification method %q", nil})
	v.enumValue(path, c.Status, enumSpec{allowedControlVerificationStatus, "model.invalid_control_verification_status", "unknown control verification status %q", nil})
	v.rawRef(path, c.Owner, rawRefSpec{set: v.ids.actors, code: "model.invalid_control_verification_owner", format: "unknown control verification owner %q"})
	v.isoDateValue(path, c.LastTested, textSpec{"model.invalid_control_verification_last_tested", "invalid lastTested %q, expected YYYY-MM-DD"})
}

// checkControlVerificationRefs requires the scenarios, risks and verification references of a verification to exist.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkControlVerificationRefs(path string, c model.ControlVerification) {
	v.refList(path, c.ThreatScenarioRefs, refSpec{ok: inSet(v.ids.threatScenarios), code: "model.invalid_control_verification_threat_ref", format: "unknown threat scenario ref %q", field: "threatScenarioRefs"})
	v.refList(path, c.RiskRefs, refSpec{ok: inSet(v.ids.risks), code: "model.invalid_control_verification_risk_ref", format: "unknown risk ref %q", field: "riskRefs"})
	v.checkVerificationRefs(path, c)
	v.blankPaths(path, evidencePathsOf(c.Evidence, func(e model.ControlVerificationEvidence) string { return e.Path }), evidenceSpec{"model.empty_control_verification_evidence_path", "control verification evidence path is required"})
}

// checkVerificationRefs requires each verification ref to name a requirement or
// a CODE identifier of a test declaration; an alias::ID entry is checked by composition.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) checkVerificationRefs(path string, c model.ControlVerification) {
	for j, raw := range c.VerificationRefs {
		ref := strings.TrimSpace(raw)
		if strings.Contains(ref, "::") {
			continue
		}
		if ref == "" || !v.isVerificationTarget(ref) {
			v.fail("model.invalid_control_verification_ref", indexedPath(path+".verificationRefs", j), fmt.Sprintf("verification ref %q must name a requirement or an inferred CODE identifier of a test declaration", ref))
		}
	}
}

// isVerificationTarget reports whether a verification ref is a requirement or a valid CODE id.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) isVerificationTarget(ref string) bool {
	return v.kindByID[ref] == "requirement" || model.IsCodeID(ref) && v.validID(ref)
}
