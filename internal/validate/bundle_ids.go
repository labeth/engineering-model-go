// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"
)

// registerIDs registers every authored id in a fixed order, so a duplicate id
// is reported against the entity that came first.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerIDs() {
	v.registerDecisionsRequirementsGroups()
	v.registerUnits()
	v.registerActorsAndVectors()
	v.registerReferencedElements()
	v.registerInterfaces()
	v.registerAssets()
	v.registerControlsRisksPOAM()
	v.registerTrustBoundaries()
	v.registerMachines()
	v.registerEventsAndFlows()
	v.registerThreatIDs()
	v.registerComplianceAndViewIDs()
}

// registerDecisionsRequirementsGroups registers decisions, requirements and functional groups.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerDecisionsRequirementsGroups() {
	for i, d := range v.b.Architecture.Decisions {
		v.register(idKind{"decision", nil}, d.ID, indexedPath("decisions", i))
	}
	for i, requirement := range v.b.Requirements.Requirements {
		v.register(idKind{"requirement", nil}, requirement.ID, indexedPath("requirements", i))
	}
	for i, g := range v.b.Architecture.AuthoredArchitecture.FunctionalGroups {
		v.register(idKind{"functional_group", v.ids.groups}, g.ID, indexedPath("authoredArchitecture.functionalGroups", i))
	}
}

// registerUnits registers functional units, each of which must name a registered group.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerUnits() {
	for i, u := range v.b.Architecture.AuthoredArchitecture.FunctionalUnits {
		path := indexedPath("authoredArchitecture.functionalUnits", i)
		v.register(idKind{"functional_unit", v.ids.units}, u.ID, path)
		if strings.TrimSpace(u.Group) == "" || !v.ids.groups[u.Group] {
			v.fail("model.invalid_group", path, fmt.Sprintf("functional unit %q must reference a valid functional group, not %q", u.ID, strings.TrimSpace(u.Group)))
		}
	}
}

// registerActorsAndVectors registers actors and attack vectors.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerActorsAndVectors() {
	aa := v.b.Architecture.AuthoredArchitecture
	for i, a := range aa.Actors {
		v.register(idKind{"actor", v.ids.actors}, a.ID, indexedPath("authoredArchitecture.actors", i))
	}
	for i, a := range aa.AttackVectors {
		v.register(idKind{"attack_vector", v.ids.vectors}, a.ID, indexedPath("authoredArchitecture.attackVectors", i))
	}
}

// registerReferencedElements registers referenced elements and checks their uri.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerReferencedElements() {
	for i, r := range v.b.Architecture.AuthoredArchitecture.ReferencedElements {
		path := indexedPath("authoredArchitecture.referencedElements", i)
		v.register(idKind{"referenced_element", v.ids.references}, r.ID, path)
		v.diags = append(v.diags, referencedElementURIDiagnostics(r, path)...)
	}
}

// registerInterfaces registers interfaces, each owned by a unit or a referenced element.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerInterfaces() {
	for i, x := range v.b.Architecture.AuthoredArchitecture.Interfaces {
		path := indexedPath("authoredArchitecture.interfaces", i)
		v.register(idKind{"interface", v.ids.interfaces}, x.ID, path)
		if owner := strings.TrimSpace(x.Owner); invalidInterfaceOwner(owner, v.ids.units, v.ids.references) {
			v.fail("model.invalid_interface_owner", path, fmt.Sprintf("interface %q owner %q must be a functional unit, or a referenced element for an interface an external element offers", x.ID, owner))
		}
	}
}

// registerAssets registers data objects, deployment targets and hardware.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerAssets() {
	aa := v.b.Architecture.AuthoredArchitecture
	for i, x := range aa.DataObjects {
		v.register(idKind{"data_object", v.ids.dataObjects}, x.ID, indexedPath("authoredArchitecture.dataObjects", i))
	}
	for i, x := range aa.DeploymentTargets {
		v.register(idKind{"deployment_target", v.ids.deploymentTargets}, x.ID, indexedPath("authoredArchitecture.deploymentTargets", i))
	}
	for i, x := range aa.HardwareItems {
		v.register(idKind{"hardware_item", nil}, x.ID, indexedPath("authoredArchitecture.hardwareItems", i))
	}
	for i, x := range aa.HardwareInterfaces {
		v.register(idKind{"hardware_interface", nil}, x.ID, indexedPath("authoredArchitecture.hardwareInterfaces", i))
	}
}

// registerControlsRisksPOAM registers controls, risks and POA&M items.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerControlsRisksPOAM() {
	aa := v.b.Architecture.AuthoredArchitecture
	for i, x := range aa.Controls {
		v.register(idKind{"control", v.ids.controls}, x.ID, indexedPath("authoredArchitecture.controls", i))
	}
	for i, x := range aa.Risks {
		v.register(idKind{"risk", v.ids.risks}, x.ID, indexedPath("authoredArchitecture.risks", i))
	}
	for i, x := range aa.POAMItems {
		v.register(idKind{"poam_item", nil}, x.ID, indexedPath("authoredArchitecture.poamItems", i))
	}
}

// registerTrustBoundaries registers trust boundaries; none may parent itself.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerTrustBoundaries() {
	for i, x := range v.b.Architecture.AuthoredArchitecture.TrustBoundaries {
		path := indexedPath("authoredArchitecture.trustBoundaries", i)
		v.register(idKind{"trust_boundary", v.ids.trustBoundaries}, x.ID, path)
		if parent := strings.TrimSpace(x.ParentRef); parent == strings.TrimSpace(x.ID) && parent != "" {
			v.fail("model.invalid_trust_boundary_parent", path, fmt.Sprintf("trust boundary %q cannot parent itself", x.ID))
		}
	}
}

// registerMachines registers machines with their states, transitions and invariants.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerMachines() {
	for i, m := range v.b.Architecture.AuthoredArchitecture.Machines {
		path := indexedPath("authoredArchitecture.machines", i)
		v.register(idKind{"machine", nil}, m.ID, path)
		for j, x := range m.States {
			v.register(idKind{"state", nil}, x.ID, indexedPath(path+".states", j))
		}
		for j, x := range m.Transitions {
			v.register(idKind{"transition", nil}, x.ID, indexedPath(path+".transitions", j))
		}
		for j, x := range m.Invariants {
			v.register(idKind{"invariant", nil}, x.ID, indexedPath(path+".invariants", j))
		}
	}
}

// registerEventsAndFlows registers events and flows.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerEventsAndFlows() {
	aa := v.b.Architecture.AuthoredArchitecture
	for i, x := range aa.Events {
		v.register(idKind{"event", v.ids.events}, x.ID, indexedPath("authoredArchitecture.events", i))
	}
	for i, x := range aa.Flows {
		v.register(idKind{"flow", nil}, x.ID, indexedPath("authoredArchitecture.flows", i))
	}
}

// registerThreatIDs registers the threat entities and control verifications.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerThreatIDs() {
	aa := v.b.Architecture.AuthoredArchitecture
	for i, x := range aa.ThreatScenarios {
		v.register(idKind{"threat_scenario", v.ids.threatScenarios}, x.ID, indexedPath("authoredArchitecture.threatScenarios", i))
	}
	for i, x := range aa.ThreatAssumptions {
		v.register(idKind{"threat_assumption", v.ids.threatAssumptions}, x.ID, indexedPath("authoredArchitecture.threatAssumptions", i))
	}
	for i, x := range aa.ThreatOutOfScope {
		v.register(idKind{"threat_out_of_scope", v.ids.threatOutOfScope}, x.ID, indexedPath("authoredArchitecture.threatOutOfScope", i))
	}
	for i, x := range aa.ThreatMitigations {
		v.register(idKind{"threat_mitigation", v.ids.threatMitigations}, x.ID, indexedPath("authoredArchitecture.threatMitigations", i))
	}
	for i, x := range aa.ControlVerifications {
		v.register(idKind{"control_verification", v.ids.controlVerifications}, x.ID, indexedPath("authoredArchitecture.controlVerifications", i))
	}
}

// registerComplianceAndViewIDs registers compliance profiles and mappings,
// semantic elements, views and documents.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-VALID, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED, FU-VIEW-PROJECTION, IF-GO-PKG-VALIDATE
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011, REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-041, REQ-EMG-043, REQ-EMG-049, REQ-EMG-065, REQ-EMG-070, REQ-EMG-082, REQ-EMG-083, REQ-EMG-085, REQ-EMG-090, REQ-EMG-094, REQ-EMG-124, REQ-EMG-126, REQ-EMG-128, REQ-EMG-132, REQ-EMG-133, REQ-EMG-138, REQ-EMG-139
func (v *bundleValidator) registerComplianceAndViewIDs() {
	arch := v.b.Architecture
	for i, x := range arch.Compliance.Profiles {
		v.register(idKind{"compliance_profile", v.ids.complianceProfiles}, x.ID, indexedPath("compliance.profiles", i))
	}
	for i, x := range arch.Compliance.Mappings {
		v.register(idKind{"compliance_mapping", nil}, x.ID, indexedPath("compliance.mappings", i))
	}
	for i, x := range arch.Semantics.Elements {
		v.register(idKind{string(x.Kind), nil}, x.ID, indexedPath("semantics.elements", i))
	}
	for i, view := range arch.Views {
		v.register(idKind{"view", nil}, view.ID, indexedPath("views", i))
	}
	for i, document := range v.b.Views.Documents {
		v.register(idKind{"document", v.ids.documents}, document.ID, indexedPath("documents", i))
	}
}
