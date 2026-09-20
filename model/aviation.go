// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package model

import (
	"fmt"
	"sort"
	"strings"
)

// TRLC-LINKS: REQ-EMG-053, REQ-EMG-054
type AviationDocument struct {
	SchemaVersion int             `yaml:"schemaVersion" json:"schemaVersion"`
	Aviation      AviationProfile `yaml:"aviation" json:"aviation"`
}

type AviationProfile struct {
	Profile              AirborneProfile            `yaml:"profile" json:"profile"`
	LifecycleData        []LifecycleDataRecord      `yaml:"lifecycleData" json:"lifecycleData"`
	Objectives           []ObjectiveReference       `yaml:"objectives" json:"objectives"`
	RequirementAssurance []RequirementAssurance     `yaml:"requirementAssurance" json:"requirementAssurance"`
	VerificationCases    []AirborneVerificationCase `yaml:"verificationCases" json:"verificationCases"`
	Procedures           []VerificationProcedure    `yaml:"procedures" json:"procedures"`
	Executions           []VerificationExecution    `yaml:"executions" json:"executions"`
	Baselines            []ConfigurationBaseline    `yaml:"baselines" json:"baselines"`
	Builds               []ConfigurationBuild       `yaml:"builds" json:"builds"`
	Coverage             []StructuralCoverage       `yaml:"coverage" json:"coverage"`
	Tools                []AirborneTool             `yaml:"tools" json:"tools"`
	Reviews              []AirborneReview           `yaml:"reviews" json:"reviews"`
	Approvals            []AirborneApproval         `yaml:"approvals" json:"approvals"`
	Independence         []IndependenceRecord       `yaml:"independence" json:"independence"`
	ProblemReports       []ProblemReport            `yaml:"problemReports" json:"problemReports"`
	SQAAudits            []SQAAudit                 `yaml:"sqaAudits" json:"sqaAudits"`
	Liaison              []CertificationLiaison     `yaml:"liaison" json:"liaison"`
}

type AirborneProfile struct {
	ID                        string   `yaml:"id" json:"id"`
	Standard                  string   `yaml:"standard" json:"standard"`
	Edition                   string   `yaml:"edition" json:"edition"`
	SoftwareLevel             string   `yaml:"softwareLevel" json:"softwareLevel"`
	SoftwareItem              string   `yaml:"softwareItem" json:"softwareItem"`
	Applicant                 string   `yaml:"applicant" json:"applicant"`
	Authority                 string   `yaml:"authority" json:"authority"`
	CertificationBasis        string   `yaml:"certificationBasis" json:"certificationBasis"`
	SafetyAllocationReference string   `yaml:"safetyAllocationReference" json:"safetyAllocationReference"`
	Supplements               []string `yaml:"supplements" json:"supplements"`
	ClaimsDisclaimer          string   `yaml:"claimsDisclaimer" json:"claimsDisclaimer"`
}

type EvidenceReference struct {
	ID      string `yaml:"id" json:"id"`
	Path    string `yaml:"path" json:"path"`
	URI     string `yaml:"uri" json:"uri"`
	Version string `yaml:"version" json:"version"`
	Hash    string `yaml:"hash" json:"hash"`
}

type LifecycleDataRecord struct {
	ID            string            `yaml:"id" json:"id"`
	CategoryCode  string            `yaml:"categoryCode" json:"categoryCode"`
	CategoryName  string            `yaml:"categoryName" json:"categoryName"`
	Source        EvidenceReference `yaml:"source" json:"source"`
	Baseline      string            `yaml:"baseline" json:"baseline"`
	Owner         string            `yaml:"owner" json:"owner"`
	State         string            `yaml:"state" json:"state"`
	Approvals     []string          `yaml:"approvals" json:"approvals"`
	ObjectiveRefs []string          `yaml:"objectiveRefs" json:"objectiveRefs"`
}

type ObjectiveReference struct {
	ID                   string   `yaml:"id" json:"id"`
	SourceReference      string   `yaml:"sourceReference" json:"sourceReference"`
	Applicability        string   `yaml:"applicability" json:"applicability"`
	IndependenceRequired bool     `yaml:"independenceRequired" json:"independenceRequired"`
	Rationale            string   `yaml:"rationale" json:"rationale"`
	EvidenceRefs         []string `yaml:"evidenceRefs" json:"evidenceRefs"`
	State                string   `yaml:"state" json:"state"`
}

type RequirementAssurance struct {
	RequirementID           string   `yaml:"requirementId" json:"requirementId"`
	Level                   string   `yaml:"level" json:"level"`
	Derived                 bool     `yaml:"derived" json:"derived"`
	DerivedRationale        string   `yaml:"derivedRationale" json:"derivedRationale"`
	ParentRefs              []string `yaml:"parentRefs" json:"parentRefs"`
	ArchitectureRefs        []string `yaml:"architectureRefs" json:"architectureRefs"`
	CodeRefs                []string `yaml:"codeRefs" json:"codeRefs"`
	VerificationCriteria    string   `yaml:"verificationCriteria" json:"verificationCriteria"`
	SafetyFeedbackRecipient string   `yaml:"safetyFeedbackRecipient" json:"safetyFeedbackRecipient"`
	SafetyFeedbackStatus    string   `yaml:"safetyFeedbackStatus" json:"safetyFeedbackStatus"`
	SafetyFeedbackEvidence  []string `yaml:"safetyFeedbackEvidence" json:"safetyFeedbackEvidence"`
	Baseline                string   `yaml:"baseline" json:"baseline"`
	State                   string   `yaml:"state" json:"state"`
}

type AirborneVerificationCase struct {
	ID                   string   `yaml:"id" json:"id"`
	Method               string   `yaml:"method" json:"method"`
	Intent               string   `yaml:"intent" json:"intent"`
	RequirementRefs      []string `yaml:"requirementRefs" json:"requirementRefs"`
	ExpectedResult       string   `yaml:"expectedResult" json:"expectedResult"`
	ProcedureRef         string   `yaml:"procedureRef" json:"procedureRef"`
	IndependenceRequired bool     `yaml:"independenceRequired" json:"independenceRequired"`
	State                string   `yaml:"state" json:"state"`
}

type VerificationProcedure struct {
	ID          string   `yaml:"id" json:"id"`
	CaseRefs    []string `yaml:"caseRefs" json:"caseRefs"`
	Steps       []string `yaml:"steps" json:"steps"`
	Environment string   `yaml:"environment" json:"environment"`
	Target      string   `yaml:"target" json:"target"`
	Baseline    string   `yaml:"baseline" json:"baseline"`
	State       string   `yaml:"state" json:"state"`
}

type VerificationExecution struct {
	ID           string   `yaml:"id" json:"id"`
	ProcedureRef string   `yaml:"procedureRef" json:"procedureRef"`
	CaseRefs     []string `yaml:"caseRefs" json:"caseRefs"`
	Baseline     string   `yaml:"baseline" json:"baseline"`
	Build        string   `yaml:"build" json:"build"`
	Environment  string   `yaml:"environment" json:"environment"`
	Target       string   `yaml:"target" json:"target"`
	Producer     string   `yaml:"producer" json:"producer"`
	Verifier     string   `yaml:"verifier" json:"verifier"`
	Status       string   `yaml:"status" json:"status"`
	Result       string   `yaml:"result" json:"result"`
	Evidence     []string `yaml:"evidence" json:"evidence"`
	ApprovalRefs []string `yaml:"approvalRefs" json:"approvalRefs"`
}

type ConfigurationArtifact struct {
	Path string `yaml:"path" json:"path"`
	Hash string `yaml:"hash" json:"hash"`
}

type ConfigurationBaseline struct {
	ID        string                  `yaml:"id" json:"id"`
	Version   string                  `yaml:"version" json:"version"`
	Hash      string                  `yaml:"hash" json:"hash"`
	Artifacts []ConfigurationArtifact `yaml:"artifacts" json:"artifacts"`
	State     string                  `yaml:"state" json:"state"`
}

type ConfigurationBuild struct {
	ID              string                  `yaml:"id" json:"id"`
	Baseline        string                  `yaml:"baseline" json:"baseline"`
	Hash            string                  `yaml:"hash" json:"hash"`
	Artifacts       []ConfigurationArtifact `yaml:"artifacts" json:"artifacts"`
	Compiler        string                  `yaml:"compiler" json:"compiler"`
	CompilerOptions []string                `yaml:"compilerOptions" json:"compilerOptions"`
	Target          string                  `yaml:"target" json:"target"`
	Environment     string                  `yaml:"environment" json:"environment"`
	Reproducibility string                  `yaml:"reproducibility" json:"reproducibility"`
	Status          string                  `yaml:"status" json:"status"`
}

type UncoveredDisposition struct {
	ID          string   `yaml:"id" json:"id"`
	Disposition string   `yaml:"disposition" json:"disposition"`
	Count       int      `yaml:"count" json:"count"`
	Rationale   string   `yaml:"rationale" json:"rationale"`
	Evidence    []string `yaml:"evidence" json:"evidence"`
	Reviewer    string   `yaml:"reviewer" json:"reviewer"`
	Status      string   `yaml:"status" json:"status"`
}

type StructuralCoverage struct {
	ID           string                 `yaml:"id" json:"id"`
	Metric       string                 `yaml:"metric" json:"metric"`
	Baseline     string                 `yaml:"baseline" json:"baseline"`
	Build        string                 `yaml:"build" json:"build"`
	ToolRefs     []string               `yaml:"toolRefs" json:"toolRefs"`
	Total        int                    `yaml:"total" json:"total"`
	Covered      int                    `yaml:"covered" json:"covered"`
	Uncovered    int                    `yaml:"uncovered" json:"uncovered"`
	Dispositions []UncoveredDisposition `yaml:"dispositions" json:"dispositions"`
	Evidence     []string               `yaml:"evidence" json:"evidence"`
	Reviewer     string                 `yaml:"reviewer" json:"reviewer"`
	Status       string                 `yaml:"status" json:"status"`
}

type ToolAssessment struct {
	Criterion               string   `yaml:"criterion" json:"criterion"`
	TQL                     string   `yaml:"tql" json:"tql"`
	OperationalRequirements []string `yaml:"operationalRequirements" json:"operationalRequirements"`
	Evidence                []string `yaml:"evidence" json:"evidence"`
}

type AirborneTool struct {
	ID            string         `yaml:"id" json:"id"`
	Name          string         `yaml:"name" json:"name"`
	Version       string         `yaml:"version" json:"version"`
	IntendedUse   string         `yaml:"intendedUse" json:"intendedUse"`
	CreditClaimed bool           `yaml:"creditClaimed" json:"creditClaimed"`
	Assessment    ToolAssessment `yaml:"assessment" json:"assessment"`
	Environment   string         `yaml:"environment" json:"environment"`
	Baseline      string         `yaml:"baseline" json:"baseline"`
	Status        string         `yaml:"status" json:"status"`
}

type AirborneReview struct {
	ID          string   `yaml:"id" json:"id"`
	SubjectRefs []string `yaml:"subjectRefs" json:"subjectRefs"`
	Reviewer    string   `yaml:"reviewer" json:"reviewer"`
	Result      string   `yaml:"result" json:"result"`
	Evidence    []string `yaml:"evidence" json:"evidence"`
	State       string   `yaml:"state" json:"state"`
}

type AirborneApproval struct {
	ID          string   `yaml:"id" json:"id"`
	SubjectRefs []string `yaml:"subjectRefs" json:"subjectRefs"`
	Approver    string   `yaml:"approver" json:"approver"`
	Evidence    []string `yaml:"evidence" json:"evidence"`
	State       string   `yaml:"state" json:"state"`
}

type IndependenceRecord struct {
	ID          string   `yaml:"id" json:"id"`
	SubjectRef  string   `yaml:"subjectRef" json:"subjectRef"`
	Producer    string   `yaml:"producer" json:"producer"`
	Verifier    string   `yaml:"verifier" json:"verifier"`
	ApprovalRef string   `yaml:"approvalRef" json:"approvalRef"`
	Evidence    []string `yaml:"evidence" json:"evidence"`
	State       string   `yaml:"state" json:"state"`
}

type ProblemReport struct {
	ID               string   `yaml:"id" json:"id"`
	Summary          string   `yaml:"summary" json:"summary"`
	AffectedRefs     []string `yaml:"affectedRefs" json:"affectedRefs"`
	Disposition      string   `yaml:"disposition" json:"disposition"`
	CorrectiveAction string   `yaml:"correctiveAction" json:"correctiveAction"`
	Evidence         []string `yaml:"evidence" json:"evidence"`
	State            string   `yaml:"state" json:"state"`
}

type SQAAudit struct {
	ID                string   `yaml:"id" json:"id"`
	Scope             string   `yaml:"scope" json:"scope"`
	Auditor           string   `yaml:"auditor" json:"auditor"`
	Findings          []string `yaml:"findings" json:"findings"`
	CorrectiveActions []string `yaml:"correctiveActions" json:"correctiveActions"`
	Evidence          []string `yaml:"evidence" json:"evidence"`
	State             string   `yaml:"state" json:"state"`
}

type CertificationLiaison struct {
	ID              string            `yaml:"id" json:"id"`
	Milestone       string            `yaml:"milestone" json:"milestone"`
	SubmissionRefs  []string          `yaml:"submissionRefs" json:"submissionRefs"`
	Findings        []string          `yaml:"findings" json:"findings"`
	AuthorityState  string            `yaml:"authorityState" json:"authorityState"`
	AuthorityRecord EvidenceReference `yaml:"authorityRecord" json:"authorityRecord"`
}

type AviationDiagnostic struct {
	Code    string `json:"code"`
	Path    string `json:"path"`
	Message string `json:"message"`
}

type AviationValidationError struct{ Diagnostics []AviationDiagnostic }

// TRLC-LINKS: REQ-EMG-054
func (e *AviationValidationError) Error() string {
	if len(e.Diagnostics) == 0 {
		return "aviation assurance validation failed"
	}
	return fmt.Sprintf("aviation assurance validation failed: %s: %s", e.Diagnostics[0].Path, e.Diagnostics[0].Message)
}

var lifecycleDataFamilies = []struct{ Code, Name string }{
	{"PLAN-SOFTWARE-ASPECTS-CERTIFICATION", "Plan for Software Aspects of Certification"},
	{"SOFTWARE-DEVELOPMENT-PLAN", "Software Development Plan"},
	{"SOFTWARE-VERIFICATION-PLAN", "Software Verification Plan"},
	{"SOFTWARE-CONFIGURATION-MANAGEMENT-PLAN", "Software Configuration Management Plan"},
	{"SOFTWARE-QUALITY-ASSURANCE-PLAN", "Software Quality Assurance Plan"},
	{"SOFTWARE-REQUIREMENTS-STANDARDS", "Software Requirements Standards"},
	{"SOFTWARE-DESIGN-STANDARDS", "Software Design Standards"},
	{"SOFTWARE-CODE-STANDARDS", "Software Code Standards"},
	{"SOFTWARE-REQUIREMENTS-DATA", "Software Requirements Data"},
	{"DESIGN-DESCRIPTION", "Design Description"},
	{"SOURCE-CODE", "Source Code"},
	{"EXECUTABLE-OBJECT-CODE", "Executable Object Code"},
	{"SOFTWARE-VERIFICATION-CASES-PROCEDURES", "Software Verification Cases and Procedures"},
	{"SOFTWARE-VERIFICATION-RESULTS", "Software Verification Results"},
	{"SOFTWARE-LIFE-CYCLE-ENVIRONMENT-CONFIGURATION-INDEX", "Software Life Cycle Environment Configuration Index"},
	{"SOFTWARE-CONFIGURATION-INDEX", "Software Configuration Index"},
	{"PROBLEM-REPORTS", "Problem Reports"},
	{"SOFTWARE-CONFIGURATION-MANAGEMENT-RECORDS", "Software Configuration Management Records"},
	{"SOFTWARE-QUALITY-ASSURANCE-RECORDS", "Software Quality Assurance Records"},
	{"SOFTWARE-ACCOMPLISHMENT-SUMMARY", "Software Accomplishment Summary"},
	{"PARAMETER-DATA-ITEM-FILE", "Parameter Data Item File"},
	{"TRACEABILITY-DATA", "Traceability Data"},
}

// TRLC-LINKS: REQ-EMG-053, REQ-EMG-055
func RecognizedLifecycleDataFamilies() []LifecycleDataRecord {
	out := make([]LifecycleDataRecord, 0, len(lifecycleDataFamilies))
	for _, family := range lifecycleDataFamilies {
		out = append(out, LifecycleDataRecord{CategoryCode: family.Code, CategoryName: family.Name})
	}
	return out
}

// ValidateAviation performs cross-document and evidence-chain validation.
// TRLC-LINKS: REQ-EMG-054
func ValidateAviation(doc AviationDocument, requirements RequirementsDocument, architectures ...ArchitectureInputDocument) []AviationDiagnostic {
	var d []AviationDiagnostic
	a := doc.Aviation
	add := func(code, path, message string) { d = append(d, AviationDiagnostic{code, path, message}) }
	if a.Profile.SoftwareLevel < "A" || a.Profile.SoftwareLevel > "E" {
		add("aviation.profile.invalid_dal", "aviation.profile.softwareLevel", "softwareLevel must be A, B, C, D, or E")
	}
	ids := map[string]string{}
	register := func(id, path string) {
		if prior, ok := ids[id]; ok {
			add("aviation.duplicate_id", path, "identifier "+id+" duplicates "+prior)
		} else {
			ids[id] = path
		}
	}
	register(a.Profile.ID, "aviation.profile.id")
	familyNames := map[string]string{}
	for _, family := range lifecycleDataFamilies {
		familyNames[family.Code] = family.Name
	}
	seenFamilies := map[string]bool{}
	for i, v := range a.LifecycleData {
		register(v.ID, fmt.Sprintf("aviation.lifecycleData[%d].id", i))
		if v.Source.ID != "" {
			register(v.Source.ID, fmt.Sprintf("aviation.lifecycleData[%d].source.id", i))
		}
		expectedName, recognized := familyNames[v.CategoryCode]
		if !recognized {
			add("aviation.lifecycle.unknown_category", fmt.Sprintf("aviation.lifecycleData[%d].categoryCode", i), "category code is not one of the recognized lifecycle-data families")
		} else if expectedName != v.CategoryName {
			add("aviation.lifecycle.category_name", fmt.Sprintf("aviation.lifecycleData[%d].categoryName", i), "category name does not match the stable category code")
		}
		if seenFamilies[v.CategoryCode] {
			add("aviation.lifecycle.duplicate_category", fmt.Sprintf("aviation.lifecycleData[%d].categoryCode", i), "lifecycle-data category occurs more than once")
		}
		seenFamilies[v.CategoryCode] = true
	}
	for code := range familyNames {
		if !seenFamilies[code] {
			add("aviation.lifecycle.missing_category", "aviation.lifecycleData", "missing recognized lifecycle-data category "+code)
		}
	}
	for i, v := range a.Objectives {
		register(v.ID, fmt.Sprintf("aviation.objectives[%d].id", i))
	}
	for _, v := range a.VerificationCases {
		register(v.ID, "aviation.verificationCases")
	}
	for _, v := range a.Procedures {
		register(v.ID, "aviation.procedures")
	}
	for _, v := range a.Executions {
		register(v.ID, "aviation.executions")
	}
	for _, v := range a.Baselines {
		register(v.ID, "aviation.baselines")
	}
	for _, v := range a.Builds {
		register(v.ID, "aviation.builds")
	}
	for _, v := range a.Coverage {
		register(v.ID, "aviation.coverage")
	}
	for _, v := range a.Tools {
		register(v.ID, "aviation.tools")
	}
	for _, v := range a.Reviews {
		register(v.ID, "aviation.reviews")
	}
	for _, v := range a.Approvals {
		register(v.ID, "aviation.approvals")
	}
	for _, v := range a.Independence {
		register(v.ID, "aviation.independence")
	}
	for _, v := range a.ProblemReports {
		register(v.ID, "aviation.problemReports")
	}
	for _, v := range a.SQAAudits {
		register(v.ID, "aviation.sqaAudits")
	}
	for _, v := range a.Liaison {
		register(v.ID, "aviation.liaison")
		if v.AuthorityRecord.ID != "" {
			register(v.AuthorityRecord.ID, "aviation.liaison."+v.ID+".authorityRecord.id")
		}
	}
	reqs := map[string]bool{}
	for _, r := range requirements.Requirements {
		reqs[r.ID] = true
	}
	architectureIDs := map[string]bool{}
	for _, document := range architectures {
		input := document.Architecture
		for _, item := range input.FunctionalGroups {
			architectureIDs[item.ID] = true
		}
		for _, item := range input.FunctionalUnits {
			architectureIDs[item.ID] = true
		}
		for _, item := range input.Interfaces {
			architectureIDs[item.ID] = true
		}
		for _, item := range input.DataObjects {
			architectureIDs[item.ID] = true
		}
		for _, item := range input.DeploymentTargets {
			architectureIDs[item.ID] = true
		}
		for _, item := range input.HardwareItems {
			architectureIDs[item.ID] = true
		}
	}
	resolves := func(ref string) bool { return ids[ref] != "" || reqs[ref] || architectureIDs[ref] }
	baselines, builds, procedures, cases, approvals, tools := map[string]bool{}, map[string]ConfigurationBuild{}, map[string]bool{}, map[string]AirborneVerificationCase{}, map[string]bool{}, map[string]bool{}
	for _, v := range a.Baselines {
		baselines[v.ID] = true
		if v.Hash == "" {
			add("aviation.baseline.hash_required", "aviation.baselines."+v.ID, "baseline hash is required")
		}
		for _, artifact := range v.Artifacts {
			if artifact.Hash == "" {
				add("aviation.artifact.hash_required", "aviation.baselines."+v.ID, "every baseline artifact requires a hash")
			}
		}
	}
	for _, v := range a.Builds {
		builds[v.ID] = v
		if v.Hash == "" {
			add("aviation.build.hash_required", "aviation.builds."+v.ID, "build hash is required")
		}
		if !baselines[v.Baseline] {
			add("aviation.reference.baseline", "aviation.builds."+v.ID, "baseline reference does not resolve")
		}
		for _, artifact := range v.Artifacts {
			if artifact.Hash == "" {
				add("aviation.artifact.hash_required", "aviation.builds."+v.ID, "every build artifact requires a hash")
			}
		}
	}
	for _, v := range a.Procedures {
		procedures[v.ID] = true
		if !baselines[v.Baseline] {
			add("aviation.reference.baseline", "aviation.procedures."+v.ID, "baseline reference does not resolve")
		}
	}
	for _, v := range a.VerificationCases {
		cases[v.ID] = v
		for _, ref := range v.RequirementRefs {
			if !reqs[ref] {
				add("aviation.reference.requirement", "aviation.verificationCases."+v.ID, "requirement reference "+ref+" does not resolve")
			}
		}
		if !procedures[v.ProcedureRef] {
			add("aviation.reference.procedure", "aviation.verificationCases."+v.ID, "procedure reference does not resolve")
		}
	}
	for _, v := range a.Approvals {
		approvals[v.ID] = len(v.Evidence) > 0
	}
	for _, v := range a.Procedures {
		for _, ref := range v.CaseRefs {
			if _, ok := cases[ref]; !ok {
				add("aviation.reference.case", "aviation.procedures."+v.ID, "verification case reference "+ref+" does not resolve")
			} else if cases[ref].ProcedureRef != v.ID {
				add("aviation.procedure.case_mismatch", "aviation.procedures."+v.ID, "verification case points to a different procedure")
			}
		}
	}
	for _, v := range a.Independence {
		if v.Producer == v.Verifier {
			add("aviation.independence.same_person", "aviation.independence."+v.ID, "producer and verifier must be distinct")
		}
		if !approvals[v.ApprovalRef] || len(v.Evidence) == 0 {
			add("aviation.independence.approval_evidence", "aviation.independence."+v.ID, "independence record requires an approval with evidence and its own evidence")
		}
	}
	for _, v := range a.Tools {
		tools[v.ID] = true
		if !baselines[v.Baseline] {
			add("aviation.reference.baseline", "aviation.tools."+v.ID, "baseline reference does not resolve")
		}
		if v.CreditClaimed && (v.Assessment.Criterion == "" || v.Assessment.TQL == "" || len(v.Assessment.OperationalRequirements) == 0 || len(v.Assessment.Evidence) == 0) {
			add("aviation.tool.incomplete_credit_assessment", "aviation.tools."+v.ID, "credit-claiming tool requires criterion, TQL, operational requirements, and evidence")
		}
	}
	for _, v := range a.RequirementAssurance {
		if !reqs[v.RequirementID] {
			add("aviation.reference.requirement", "aviation.requirementAssurance."+v.RequirementID, "canonical requirement does not resolve")
		}
		if !baselines[v.Baseline] {
			add("aviation.reference.baseline", "aviation.requirementAssurance."+v.RequirementID, "baseline reference does not resolve")
		}
		if v.Derived && (v.DerivedRationale == "" || v.SafetyFeedbackRecipient == "" || v.SafetyFeedbackStatus == "" || len(v.SafetyFeedbackEvidence) == 0) {
			add("aviation.derived.incomplete_safety_feedback", "aviation.requirementAssurance."+v.RequirementID, "derived requirement requires rationale and safety feedback recipient, status, and evidence")
		}
		for _, ref := range v.ParentRefs {
			if !reqs[ref] {
				add("aviation.reference.parent_requirement", "aviation.requirementAssurance."+v.RequirementID, "parent requirement "+ref+" does not resolve")
			}
			for _, ref := range v.ArchitectureRefs {
				if len(architectures) > 0 && !architectureIDs[ref] {
					add("aviation.reference.architecture", "aviation.requirementAssurance."+v.RequirementID, "architecture reference "+ref+" does not resolve")
				}
			}
			for _, ref := range v.SafetyFeedbackEvidence {
				if !resolves(ref) {
					add("aviation.reference.evidence", "aviation.requirementAssurance."+v.RequirementID, "safety feedback evidence "+ref+" does not resolve")
				}
			}
		}
	}
	for _, v := range a.Executions {
		if !procedures[v.ProcedureRef] {
			add("aviation.reference.procedure", "aviation.executions."+v.ID, "procedure reference does not resolve")
		}
		build, ok := builds[v.Build]
		if !ok {
			add("aviation.reference.build", "aviation.executions."+v.ID, "build reference does not resolve")
		} else if build.Baseline != v.Baseline {
			add("aviation.execution.baseline_mismatch", "aviation.executions."+v.ID, "execution baseline does not match build baseline")
		}
		if !baselines[v.Baseline] {
			add("aviation.reference.baseline", "aviation.executions."+v.ID, "baseline reference does not resolve")
		}
		for _, ref := range v.CaseRefs {
			c, ok := cases[ref]
			if !ok {
				add("aviation.reference.case", "aviation.executions."+v.ID, "verification case reference does not resolve")
				continue
			}
			if c.ProcedureRef != v.ProcedureRef {
				add("aviation.execution.procedure_mismatch", "aviation.executions."+v.ID, "execution procedure does not match verification case procedure")
			}
			if c.IndependenceRequired && (v.Producer == v.Verifier || len(v.ApprovalRefs) == 0) {
				add("aviation.independence.missing", "aviation.executions."+v.ID, "independent execution requires distinct producer/verifier and approval evidence")
			}
			for _, approval := range v.ApprovalRefs {
				if !approvals[approval] {
					add("aviation.reference.approval", "aviation.executions."+v.ID, "approval reference lacks evidence or does not resolve")
				}
				for _, ref := range v.Evidence {
					if !resolves(ref) {
						add("aviation.reference.evidence", "aviation.executions."+v.ID, "evidence reference "+ref+" does not resolve")
					}
				}
			}
		}
	}
	for _, v := range a.Coverage {
		if a.Profile.SoftwareLevel == "C" && v.Metric != "statement" {
			add("aviation.coverage.metric", "aviation.coverage."+v.ID, "DAL C coverage metric must be statement")
		}
		if v.Total != v.Covered+v.Uncovered {
			add("aviation.coverage.arithmetic", "aviation.coverage."+v.ID, "total must equal covered plus uncovered")
		}
		sum, open := 0, false
		for _, x := range v.Dispositions {
			sum += x.Count
			if x.Disposition == "open" || x.Status == "declared" || x.Status == "planned" {
				open = true
			}
		}
		if sum != v.Uncovered {
			add("aviation.coverage.disposition_count", "aviation.coverage."+v.ID, "dispositions must fully account for uncovered count")
		}
		if (v.Status == "accepted" || v.Status == "authority-accepted") && open {
			add("aviation.coverage.open_at_closure", "aviation.coverage."+v.ID, "accepted coverage cannot retain open uncovered code")
		}
		if !baselines[v.Baseline] || builds[v.Build].Baseline != v.Baseline {
			add("aviation.coverage.configuration", "aviation.coverage."+v.ID, "coverage must reference an exact matching baseline and build")
		}
		for _, ref := range v.ToolRefs {
			if !tools[ref] {
				add("aviation.reference.tool", "aviation.coverage."+v.ID, "tool reference "+ref+" does not resolve")
			}
			for _, ref := range v.Evidence {
				if !resolves(ref) {
					add("aviation.reference.evidence", "aviation.coverage."+v.ID, "evidence reference "+ref+" does not resolve")
				}
			}
		}
	}
	for _, obj := range a.Objectives {
		for _, ref := range obj.EvidenceRefs {
			if _, ok := ids[ref]; !ok {
				add("aviation.reference.evidence", "aviation.objectives."+obj.ID, "evidence reference "+ref+" does not resolve")
			}
			for _, item := range a.LifecycleData {
				if item.Baseline != "" && !baselines[item.Baseline] {
					add("aviation.reference.baseline", "aviation.lifecycleData."+item.ID, "baseline reference does not resolve")
				}
				for _, ref := range item.Approvals {
					if !approvals[ref] {
						add("aviation.reference.approval", "aviation.lifecycleData."+item.ID, "approval reference lacks evidence or does not resolve")
					}
				}
				for _, ref := range item.ObjectiveRefs {
					if !resolves(ref) {
						add("aviation.reference.objective", "aviation.lifecycleData."+item.ID, "objective reference "+ref+" does not resolve")
					}
				}
			}
			for _, tool := range a.Tools {
				for _, ref := range tool.Assessment.Evidence {
					if !resolves(ref) {
						add("aviation.reference.evidence", "aviation.tools."+tool.ID, "assessment evidence "+ref+" does not resolve")
					}
				}
			}
			for _, review := range a.Reviews {
				for _, ref := range append(append([]string{}, review.SubjectRefs...), review.Evidence...) {
					if !resolves(ref) {
						add("aviation.reference.evidence", "aviation.reviews."+review.ID, "reference "+ref+" does not resolve")
					}
				}
			}
			for _, approval := range a.Approvals {
				for _, ref := range append(append([]string{}, approval.SubjectRefs...), approval.Evidence...) {
					if !resolves(ref) {
						add("aviation.reference.evidence", "aviation.approvals."+approval.ID, "reference "+ref+" does not resolve")
					}
				}
			}
			for _, record := range a.Independence {
				if !resolves(record.SubjectRef) {
					add("aviation.reference.subject", "aviation.independence."+record.ID, "subject reference does not resolve")
				}
				for _, ref := range record.Evidence {
					if !resolves(ref) {
						add("aviation.reference.evidence", "aviation.independence."+record.ID, "evidence reference "+ref+" does not resolve")
					}
				}
			}
			for _, report := range a.ProblemReports {
				for _, ref := range append(append([]string{}, report.AffectedRefs...), report.Evidence...) {
					if !resolves(ref) {
						add("aviation.reference.evidence", "aviation.problemReports."+report.ID, "reference "+ref+" does not resolve")
					}
				}
			}
			for _, audit := range a.SQAAudits {
				for _, ref := range audit.Evidence {
					if !resolves(ref) {
						add("aviation.reference.evidence", "aviation.sqaAudits."+audit.ID, "evidence reference "+ref+" does not resolve")
					}
				}
			}
			for _, liaison := range a.Liaison {
				for _, ref := range liaison.SubmissionRefs {
					if !resolves(ref) {
						add("aviation.reference.submission", "aviation.liaison."+liaison.ID, "submission reference "+ref+" does not resolve")
					}
				}
			}
		}
		if obj.State == "authority-accepted" && !hasAuthorityAcceptance(a.Liaison, obj.ID) {
			add("aviation.authority.acceptance_missing", "aviation.objectives."+obj.ID, "authority-accepted state requires an authored authority record")
		}
	}
	requireAuthority := func(id, state, path string) {
		if state == "authority-accepted" && !hasAuthorityAcceptance(a.Liaison, id) {
			add("aviation.authority.acceptance_missing", path, "authority-accepted state requires an authored authority record")
		}
	}
	for _, v := range a.LifecycleData {
		requireAuthority(v.ID, v.State, "aviation.lifecycleData."+v.ID)
	}
	for _, v := range a.RequirementAssurance {
		requireAuthority(v.RequirementID, v.State, "aviation.requirementAssurance."+v.RequirementID)
	}
	for _, v := range a.VerificationCases {
		requireAuthority(v.ID, v.State, "aviation.verificationCases."+v.ID)
	}
	for _, v := range a.Procedures {
		requireAuthority(v.ID, v.State, "aviation.procedures."+v.ID)
	}
	for _, v := range a.Executions {
		requireAuthority(v.ID, v.Status, "aviation.executions."+v.ID)
	}
	for _, v := range a.Baselines {
		requireAuthority(v.ID, v.State, "aviation.baselines."+v.ID)
	}
	for _, v := range a.Builds {
		requireAuthority(v.ID, v.Status, "aviation.builds."+v.ID)
	}
	for _, v := range a.Coverage {
		requireAuthority(v.ID, v.Status, "aviation.coverage."+v.ID)
	}
	for _, v := range a.Tools {
		requireAuthority(v.ID, v.Status, "aviation.tools."+v.ID)
	}
	for _, v := range a.Reviews {
		requireAuthority(v.ID, v.State, "aviation.reviews."+v.ID)
	}
	for _, v := range a.Approvals {
		requireAuthority(v.ID, v.State, "aviation.approvals."+v.ID)
	}
	for _, v := range a.ProblemReports {
		requireAuthority(v.ID, v.State, "aviation.problemReports."+v.ID)
	}
	for _, v := range a.SQAAudits {
		requireAuthority(v.ID, v.State, "aviation.sqaAudits."+v.ID)
	}
	for _, v := range a.Liaison {
		if v.AuthorityState == "authority-accepted" && v.AuthorityRecord.ID == "" {
			add("aviation.authority.record_missing", "aviation.liaison."+v.ID, "authority-accepted liaison state requires an authored authority record")
		}
	}
	sort.Slice(d, func(i, j int) bool {
		if d[i].Path == d[j].Path {
			return d[i].Code < d[j].Code
		}
		return d[i].Path < d[j].Path
	})
	return d
}

// TRLC-LINKS: REQ-EMG-054
func hasAuthorityAcceptance(records []CertificationLiaison, subject string) bool {
	for _, record := range records {
		if record.AuthorityState == "authority-accepted" && record.AuthorityRecord.ID != "" && (len(record.SubmissionRefs) == 0 || containsString(record.SubmissionRefs, subject)) {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-054
func containsString(values []string, value string) bool {
	for _, candidate := range values {
		if candidate == value {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-055
func IsOpenEvidenceState(state string) bool {
	return state == "declared" || state == "planned" || state == "executed" || state == "reviewed"
}

// TRLC-LINKS: REQ-EMG-055
func NormalizeDisclaimer(value string) string {
	const required = "DRAFT / NOT A COMPLIANCE OR CERTIFICATION DETERMINATION"
	if strings.Contains(strings.ToUpper(value), required) {
		return value
	}
	if strings.TrimSpace(value) == "" {
		return required
	}
	return required + " — " + value
}
