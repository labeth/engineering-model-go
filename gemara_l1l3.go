// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

// Gemara L1 Principle/Guidance catalogs and the L3 Policy, derived from the model:
//   - L1 Principle Catalog : a standard set of governance principles
//   - L1 Guidance Catalog  : guidelines derived from control categories
//   - L3 Policy            : scope, imports, risk treatment, and adherence derived
//                            from compliance mappings, risks, and controls

import (
	"strings"

	gemara "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/model"
)

const (
	gemaraRefPrincipleCatalog = "ENGMOD-PRINCIPLE-CATALOG"
	gemaraRefGuidanceCatalog  = "ENGMOD-GUIDANCE-CATALOG"
)

// -------------------- L1 Principle Catalog --------------------

// buildPrincipleCatalog emits a standard set of governance principles.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildPrincipleCatalog(bundle model.Bundle, cfg gemaraConfig) gemara.PrincipleCatalog {
	const group = "security-principles"
	cat := gemara.PrincipleCatalog{
		Title:    cfg.modelTitle + " - Principles",
		Metadata: cfg.newMetadata(gemaraRefPrincipleCatalog, "Governing principles for "+cfg.modelTitle, gemara.PrincipleCatalogArtifact),
		Groups:   []gemara.Group{{Id: group, Title: "Security Principles", Description: "Foundational values guiding governance, design, and operations."}},
		Principles: []gemara.Principle{
			{Id: "PRIN-LEAST-PRIVILEGE", Title: "Least Privilege", Group: group, Description: "Grant only the access required to perform a function.", Rationale: "Minimizes blast radius of compromise or error."},
			{Id: "PRIN-DEFENSE-IN-DEPTH", Title: "Defense in Depth", Group: group, Description: "Layer independent controls so no single failure is catastrophic.", Rationale: "Redundant controls contain failures of any one layer."},
			{Id: "PRIN-SECURE-BY-DEFAULT", Title: "Secure by Default", Group: group, Description: "Default configurations must be the safe configurations.", Rationale: "Most deployments never change defaults."},
			{Id: "PRIN-TRACEABILITY", Title: "Traceability and Evidence", Group: group, Description: "Every control must be verifiable and produce evidence.", Rationale: "Assurance requires objective, machine-checkable evidence."},
			{Id: "PRIN-DATA-PROTECTION", Title: "Data Protection", Group: group, Description: "Protect data confidentiality, integrity, and availability commensurate with its classification.", Rationale: "Data sensitivity drives required safeguards."},
		},
	}
	return cat
}

// -------------------- L1 Guidance Catalog --------------------

// buildGuidanceCatalog derives a guidance catalog from the distinct control categories.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildGuidanceCatalog(bundle model.Bundle, cfg gemaraConfig) (gemara.GuidanceCatalog, bool) {
	a := bundle.Architecture.AuthoredArchitecture
	if len(a.Controls) == 0 {
		return gemara.GuidanceCatalog{}, false
	}
	const group = "control-guidance"
	cat := gemara.GuidanceCatalog{
		Title:        cfg.modelTitle + " - Guidance",
		Metadata:     cfg.newMetadata(gemaraRefGuidanceCatalog, "Control guidance for "+cfg.modelTitle, gemara.GuidanceCatalogArtifact),
		GuidanceType: gemara.GuidanceFramework,
		FrontMatter:  "Guidance derived from the control families of " + cfg.modelTitle + ".",
		Groups:       []gemara.Group{{Id: group, Title: "Control Guidance", Description: "Guidance organized by control family."}},
	}

	cat.Guidelines = guidanceGuidelines(a.Controls, group)
	cat.Metadata.MappingReferences = []gemara.MappingReference{
		{Id: gemaraRefPrincipleCatalog, Title: "Principle Catalog", Version: GemaraVersion},
	}
	return cat, true
}

// guidanceGuidelines builds one guideline per distinct control category.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func guidanceGuidelines(controls []model.Control, group string) []gemara.Guideline {
	seen := map[string]bool{}
	var out []gemara.Guideline
	for _, ctrl := range controls {
		category := strings.TrimSpace(ctrl.Category)
		if category == "" {
			category = "general"
		}
		gid := guidelineIDForCategory(category)
		if seen[gid] {
			continue
		}
		seen[gid] = true
		out = append(out, gemara.Guideline{
			Id:        gid,
			Title:     strings.Title(category) + " Guidance",
			Objective: "Design, implement, and verify " + category + " controls so their objectives are demonstrably met.",
			Group:     group,
			Recommendations: []string{
				"Define a verifiable assessment requirement for each control objective.",
				"Capture objective evidence for every assessment.",
			},
			Principles: []gemara.MultiEntryMapping{{
				ReferenceId: gemaraRefPrincipleCatalog,
				Entries:     []gemara.ArtifactMapping{{ReferenceId: "PRIN-TRACEABILITY"}, {ReferenceId: "PRIN-DEFENSE-IN-DEPTH"}},
			}},
			State: gemara.LifecycleActive,
		})
	}
	return out
}

// guidelineIDForCategory returns the stable guideline id for a control category.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func guidelineIDForCategory(category string) string {
	c := strings.TrimSpace(category)
	if c == "" {
		c = "general"
	}
	return "GL-" + slug(c)
}

// -------------------- L3 Policy --------------------

// buildPolicy derives an organizational policy from compliance, risks, and controls.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func buildPolicy(bundle model.Bundle, cfg gemaraConfig, date string) (gemara.Policy, bool) {
	a := bundle.Architecture.AuthoredArchitecture
	if len(a.Controls) == 0 {
		return gemara.Policy{}, false
	}
	start := normalizeDatetime(date)
	if start == "" {
		start = gemaraDefaultTimestamp
	}
	contact := []gemara.Contact{{Name: cfg.options.AuthorName}}

	pol := gemara.Policy{
		Title:    cfg.modelTitle + " - Security Policy",
		Metadata: cfg.newMetadata(cfg.modelID+"-POLICY", "Security policy for "+cfg.modelTitle, gemara.PolicyArtifact),
		Contacts: gemara.RACI{Responsible: contact, Accountable: contact},
		Scope: gemara.Scope{In: gemara.Dimensions{
			Technologies: policyTechnologies(a),
			Sensitivity:  policySensitivities(a),
			Users:        policyUsers(a),
		}},
		Imports: gemara.Imports{
			Catalogs: []gemara.CatalogImport{{ReferenceId: gemaraRefControlCatalog}},
			Guidance: []gemara.GuidanceImport{{ReferenceId: gemaraRefGuidanceCatalog}},
		},
		ImplementationPlan: gemara.ImplementationPlan{
			EvaluationTimeline:  gemara.ImplementationDetails{Start: gemara.Datetime(start), Notes: "Controls are evaluated continuously from inferred verification."},
			EnforcementTimeline: gemara.ImplementationDetails{Start: gemara.Datetime(start), Notes: "Non-compliant changes are gated and remediated via POA&M."},
		},
		Adherence: gemara.Adherence{
			EvaluationMethods: []gemara.AcceptedMethod{
				{Id: "EM-INTENT", Type: gemara.MethodIntent, Mode: gemara.ModeAutomated, Required: true, Description: "Static analysis of code, configuration, and test evidence."},
				{Id: "EM-BEHAVIORAL", Type: gemara.MethodBehavioral, Mode: gemara.ModeManual, Required: false, Description: "Exercise and review of runtime behavior."},
			},
			EnforcementMethods: []gemara.AcceptedMethod{
				{Id: "ENM-GATE", Type: gemara.MethodGate, Mode: gemara.ModeAutomated, Required: true, Description: "Preventive change-control gate."},
				{Id: "ENM-REMEDIATION", Type: gemara.MethodRemediation, Mode: gemara.ModeManual, Required: false, Description: "Remediation tracked via POA&M."},
			},
			AssessmentPlans: policyAssessmentPlans(a),
			NonCompliance:   "Non-compliant findings are tracked as POA&M items with an owner and due date; unresolved high-severity findings block release.",
		},
	}

	pol.Risks = policyRisks(a)

	pol.Metadata.MappingReferences = []gemara.MappingReference{
		{Id: gemaraRefControlCatalog, Title: "Control Catalog", Version: GemaraVersion},
		{Id: gemaraRefGuidanceCatalog, Title: "Guidance Catalog", Version: GemaraVersion},
		{Id: gemaraRefRiskCatalog, Title: "Risk Catalog", Version: GemaraVersion},
	}
	return pol, true
}

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func policyTechnologies(a model.AuthoredArchitecture) []string {
	var out []string
	for _, i := range a.Interfaces {
		if p := strings.TrimSpace(i.Protocol); p != "" {
			out = appendUnique(out, p)
		}
	}
	if len(out) == 0 {
		out = []string{"software"}
	}
	return out
}

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func policySensitivities(a model.AuthoredArchitecture) []string {
	var out []string
	for _, d := range a.DataObjects {
		if c := strings.TrimSpace(d.Classification); c != "" {
			out = appendUnique(out, c)
		} else if s := strings.TrimSpace(d.Sensitivity); s != "" {
			out = appendUnique(out, s)
		}
	}
	if len(out) == 0 {
		out = []string{"internal"}
	}
	return out
}

// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func policyUsers(a model.AuthoredArchitecture) []string {
	var out []string
	for _, ac := range a.Actors {
		out = appendUnique(out, fallback(ac.Name, ac.ID))
	}
	if len(out) == 0 {
		out = []string{"system operators"}
	}
	return out
}

// policyAssessmentPlans maps each control verification to an assessment plan.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func policyAssessmentPlans(a model.AuthoredArchitecture) []gemara.AssessmentPlan {
	var out []gemara.AssessmentPlan
	for _, cv := range a.ControlVerifications {
		out = append(out, gemara.AssessmentPlan{
			Id:            "AP-" + cv.ID,
			RequirementId: cv.ID,
			Frequency:     "continuous",
			EvaluationMethods: []gemara.AcceptedMethod{
				{Id: "EM-INTENT", Type: gemara.MethodIntent, Mode: gemara.ModeAutomated, Required: true},
			},
			EvidenceRequirements: "Verification evidence linked to the control.",
		})
	}
	return out
}

// policyRisks splits model risks into mitigated and accepted treatments.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
// TRLC-LINKS: REQ-EMG-015
func policyRisks(a model.AuthoredArchitecture) gemara.Risks {
	var risks gemara.Risks
	for _, r := range a.Risks {
		ref := gemara.EntryMapping{ReferenceId: gemaraRefRiskCatalog, EntryId: r.ID}
		if strings.Contains(strings.ToLower(r.Response), "accept") {
			risks.Accepted = append(risks.Accepted, gemara.AcceptedRisk{
				Id:            "ACC-" + r.ID,
				Risk:          ref,
				Justification: fallback(r.Rationale, fallback(r.ResidualRisk, "Accepted by policy owner.")),
			})
		} else {
			risks.Mitigated = append(risks.Mitigated, gemara.MitigatedRisk{Id: "MIT-" + r.ID, Risk: ref})
		}
	}
	return risks
}
