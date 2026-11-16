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
	// Rationale explains a not-applicable state; only with it does
	// not-applicable satisfy a life-cycle tier evidence floor.
	Rationale string `yaml:"rationale,omitempty" json:"rationale,omitempty"`
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

// aviationValidator holds the indexes ValidateAviation builds while it walks
// the document. The checks run in a fixed order because later checks resolve
// against indexes that earlier checks fill.
type aviationValidator struct {
	a               AviationProfile
	diags           []AviationDiagnostic
	ids             map[string]string
	reqs            map[string]bool
	architectureIDs map[string]bool
	haveArchitects  bool
	baselines       map[string]bool
	procedures      map[string]bool
	approvals       map[string]bool
	tools           map[string]bool
	builds          map[string]ConfigurationBuild
	cases           map[string]AirborneVerificationCase
}

// aviationRefRule describes the diagnostic of a reference that does not
// resolve: the message is "<label> <ref> does not resolve".
type aviationRefRule struct{ code, path, label string }

// aviationIDGroup is a list of identifiers registered under one path.
type aviationIDGroup struct {
	path string
	ids  []string
}

// aviationAuthorityTarget is a record whose state may claim authority
// acceptance.
type aviationAuthorityTarget struct{ id, state, path string }

// ValidateAviation performs cross-document and evidence-chain validation.
// TRLC-LINKS: REQ-EMG-054
func ValidateAviation(doc AviationDocument, requirements RequirementsDocument, architectures ...ArchitectureInputDocument) []AviationDiagnostic {
	v := newAviationValidator(doc.Aviation, requirements, architectures)
	v.checkSoftwareLevel()
	v.registerIdentifiers()
	v.checkBaselines()
	v.checkBuilds()
	v.checkProcedureBaselines()
	v.checkVerificationCases()
	v.indexApprovals()
	v.checkProcedureCases()
	v.checkIndependence()
	v.checkTools()
	v.checkRequirementAssurance()
	v.checkExecutions()
	v.checkCoverage()
	v.checkObjectives()
	v.checkAuthorityStates()
	v.checkLiaisonRecords()
	sortAviationDiagnostics(v.diags)
	return v.diags
}

// TRLC-LINKS: REQ-EMG-054
func newAviationValidator(a AviationProfile, requirements RequirementsDocument, architectures []ArchitectureInputDocument) *aviationValidator {
	reqs := map[string]bool{}
	for _, r := range requirements.Requirements {
		reqs[r.ID] = true
	}
	return &aviationValidator{
		a:               a,
		ids:             map[string]string{},
		reqs:            reqs,
		architectureIDs: collectArchitectureIDs(architectures),
		haveArchitects:  len(architectures) > 0,
		baselines:       map[string]bool{},
		procedures:      map[string]bool{},
		approvals:       map[string]bool{},
		tools:           map[string]bool{},
		builds:          map[string]ConfigurationBuild{},
		cases:           map[string]AirborneVerificationCase{},
	}
}

// TRLC-LINKS: REQ-EMG-054
func collectArchitectureIDs(documents []ArchitectureInputDocument) map[string]bool {
	out := map[string]bool{}
	for _, document := range documents {
		for _, id := range architectureElementIDs(document.Architecture) {
			out[id] = true
		}
	}
	return out
}

// architectureElementIDs lists the IDs of the elements ArchitectureInput
// declares that an aviation record may reference.
//
// TRLC-LINKS: REQ-EMG-054
func architectureElementIDs(in ArchitectureInput) []string {
	var out []string
	out = append(out, idsOf(in.FunctionalGroups, func(x FunctionalGroup) string { return x.ID })...)
	out = append(out, idsOf(in.FunctionalUnits, func(x FunctionalUnit) string { return x.ID })...)
	out = append(out, idsOf(in.Interfaces, func(x Interface) string { return x.ID })...)
	out = append(out, idsOf(in.DataObjects, func(x DataObject) string { return x.ID })...)
	out = append(out, idsOf(in.DeploymentTargets, func(x DeploymentTarget) string { return x.ID })...)
	out = append(out, idsOf(in.HardwareItems, func(x HardwareItem) string { return x.ID })...)
	return out
}

// TRLC-LINKS: REQ-EMG-054
func idsOf[T any](items []T, id func(T) string) []string {
	out := make([]string, 0, len(items))
	for _, item := range items {
		out = append(out, id(item))
	}
	return out
}

// TRLC-LINKS: REQ-EMG-054
func sortAviationDiagnostics(d []AviationDiagnostic) {
	sort.Slice(d, func(i, j int) bool {
		if d[i].Path == d[j].Path {
			return d[i].Code < d[j].Code
		}
		return d[i].Path < d[j].Path
	})
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) add(code, path, message string) {
	v.diags = append(v.diags, AviationDiagnostic{code, path, message})
}

// resolves reports whether ref is a registered identifier, requirement or
// architecture element.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) resolves(ref string) bool {
	return v.ids[ref] != "" || v.reqs[ref] || v.architectureIDs[ref]
}

// checkRefs reports every ref that ok does not accept.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkRefs(refs []string, ok func(string) bool, rule aviationRefRule) {
	for _, ref := range refs {
		if !ok(ref) {
			v.add(rule.code, rule.path, rule.label+" "+ref+" does not resolve")
		}
	}
}

// evidenceRule is the rule of an evidence reference that does not resolve.
//
// TRLC-LINKS: REQ-EMG-054
func evidenceRule(path, label string) aviationRefRule {
	return aviationRefRule{"aviation.reference.evidence", path, label}
}

// requireBaseline reports a baseline reference that does not resolve.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) requireBaseline(path, baseline string) {
	if !v.baselines[baseline] {
		v.add("aviation.reference.baseline", path, "baseline reference does not resolve")
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) register(id, path string) {
	if prior, ok := v.ids[id]; ok {
		v.add("aviation.duplicate_id", path, "identifier "+id+" duplicates "+prior)
	} else {
		v.ids[id] = path
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkSoftwareLevel() {
	if level := v.a.Profile.SoftwareLevel; level < "A" || level > "E" {
		v.add("aviation.profile.invalid_dal", "aviation.profile.softwareLevel", "softwareLevel must be A, B, C, D, or E")
	}
}

// registerIdentifiers registers every identifier of the document and reports
// duplicates and lifecycle-data category problems.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) registerIdentifiers() {
	a := v.a
	v.register(a.Profile.ID, "aviation.profile.id")
	v.checkLifecycleData()
	for i, o := range a.Objectives {
		v.register(o.ID, fmt.Sprintf("aviation.objectives[%d].id", i))
	}
	v.registerGroups()
	for _, l := range a.Liaison {
		v.register(l.ID, "aviation.liaison")
		if l.AuthorityRecord.ID != "" {
			v.register(l.AuthorityRecord.ID, "aviation.liaison."+l.ID+".authorityRecord.id")
		}
	}
}

// registerGroups registers the identifiers of the evidence record lists.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) registerGroups() {
	a := v.a
	groups := []aviationIDGroup{
		{"aviation.verificationCases", idsOf(a.VerificationCases, func(x AirborneVerificationCase) string { return x.ID })},
		{"aviation.procedures", idsOf(a.Procedures, func(x VerificationProcedure) string { return x.ID })},
		{"aviation.executions", idsOf(a.Executions, func(x VerificationExecution) string { return x.ID })},
		{"aviation.baselines", idsOf(a.Baselines, func(x ConfigurationBaseline) string { return x.ID })},
		{"aviation.builds", idsOf(a.Builds, func(x ConfigurationBuild) string { return x.ID })},
		{"aviation.coverage", idsOf(a.Coverage, func(x StructuralCoverage) string { return x.ID })},
		{"aviation.tools", idsOf(a.Tools, func(x AirborneTool) string { return x.ID })},
		{"aviation.reviews", idsOf(a.Reviews, func(x AirborneReview) string { return x.ID })},
		{"aviation.approvals", idsOf(a.Approvals, func(x AirborneApproval) string { return x.ID })},
		{"aviation.independence", idsOf(a.Independence, func(x IndependenceRecord) string { return x.ID })},
		{"aviation.problemReports", idsOf(a.ProblemReports, func(x ProblemReport) string { return x.ID })},
		{"aviation.sqaAudits", idsOf(a.SQAAudits, func(x SQAAudit) string { return x.ID })},
	}
	for _, group := range groups {
		for _, id := range group.ids {
			v.register(id, group.path)
		}
	}
}

// checkLifecycleData registers the lifecycle-data identifiers and checks that
// every recognized category occurs once with its stable name.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkLifecycleData() {
	familyNames := map[string]string{}
	for _, family := range lifecycleDataFamilies {
		familyNames[family.Code] = family.Name
	}
	seen := map[string]bool{}
	for i, item := range v.a.LifecycleData {
		v.registerLifecycleIDs(i, item)
		v.checkLifecycleCategory(i, item, familyNames)
		v.checkLifecycleDuplicate(i, item, seen)
	}
	v.checkMissingCategories(familyNames, seen)
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) registerLifecycleIDs(i int, item LifecycleDataRecord) {
	v.register(item.ID, fmt.Sprintf("aviation.lifecycleData[%d].id", i))
	if item.Source.ID != "" {
		v.register(item.Source.ID, fmt.Sprintf("aviation.lifecycleData[%d].source.id", i))
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkMissingCategories(familyNames map[string]string, seen map[string]bool) {
	for code := range familyNames {
		if !seen[code] {
			v.add("aviation.lifecycle.missing_category", "aviation.lifecycleData", "missing recognized lifecycle-data category "+code)
		}
	}
}

// checkLifecycleDuplicate reports a repeated category and marks it seen.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkLifecycleDuplicate(i int, item LifecycleDataRecord, seen map[string]bool) {
	if seen[item.CategoryCode] {
		v.add("aviation.lifecycle.duplicate_category", fmt.Sprintf("aviation.lifecycleData[%d].categoryCode", i), "lifecycle-data category occurs more than once")
	}
	seen[item.CategoryCode] = true
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkLifecycleCategory(i int, item LifecycleDataRecord, familyNames map[string]string) {
	expectedName, recognized := familyNames[item.CategoryCode]
	if !recognized {
		v.add("aviation.lifecycle.unknown_category", fmt.Sprintf("aviation.lifecycleData[%d].categoryCode", i), "category code is not one of the recognized lifecycle-data families")
	} else if expectedName != item.CategoryName {
		v.add("aviation.lifecycle.category_name", fmt.Sprintf("aviation.lifecycleData[%d].categoryName", i), "category name does not match the stable category code")
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkArtifactHashes(artifacts []ConfigurationArtifact, path, message string) {
	for _, artifact := range artifacts {
		if artifact.Hash == "" {
			v.add("aviation.artifact.hash_required", path, message)
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkBaselines() {
	for _, b := range v.a.Baselines {
		v.baselines[b.ID] = true
		path := "aviation.baselines." + b.ID
		if b.Hash == "" {
			v.add("aviation.baseline.hash_required", path, "baseline hash is required")
		}
		v.checkArtifactHashes(b.Artifacts, path, "every baseline artifact requires a hash")
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkBuilds() {
	for _, b := range v.a.Builds {
		v.builds[b.ID] = b
		path := "aviation.builds." + b.ID
		if b.Hash == "" {
			v.add("aviation.build.hash_required", path, "build hash is required")
		}
		v.requireBaseline(path, b.Baseline)
		v.checkArtifactHashes(b.Artifacts, path, "every build artifact requires a hash")
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkProcedureBaselines() {
	for _, p := range v.a.Procedures {
		v.procedures[p.ID] = true
		v.requireBaseline("aviation.procedures."+p.ID, p.Baseline)
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkVerificationCases() {
	for _, c := range v.a.VerificationCases {
		v.cases[c.ID] = c
		path := "aviation.verificationCases." + c.ID
		v.checkRefs(c.RequirementRefs, func(ref string) bool { return v.reqs[ref] }, aviationRefRule{"aviation.reference.requirement", path, "requirement reference"})
		if !v.procedures[c.ProcedureRef] {
			v.add("aviation.reference.procedure", path, "procedure reference does not resolve")
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) indexApprovals() {
	for _, approval := range v.a.Approvals {
		v.approvals[approval.ID] = len(approval.Evidence) > 0
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkProcedureCases() {
	for _, p := range v.a.Procedures {
		for _, ref := range p.CaseRefs {
			v.checkProcedureCase(p, ref)
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkProcedureCase(p VerificationProcedure, ref string) {
	path := "aviation.procedures." + p.ID
	c, ok := v.cases[ref]
	if !ok {
		v.add("aviation.reference.case", path, "verification case reference "+ref+" does not resolve")
	} else if c.ProcedureRef != p.ID {
		v.add("aviation.procedure.case_mismatch", path, "verification case points to a different procedure")
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkIndependence() {
	for _, r := range v.a.Independence {
		path := "aviation.independence." + r.ID
		if r.Producer == r.Verifier {
			v.add("aviation.independence.same_person", path, "producer and verifier must be distinct")
		}
		if !v.approvals[r.ApprovalRef] || len(r.Evidence) == 0 {
			v.add("aviation.independence.approval_evidence", path, "independence record requires an approval with evidence and its own evidence")
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkTools() {
	for _, t := range v.a.Tools {
		v.tools[t.ID] = true
		path := "aviation.tools." + t.ID
		v.requireBaseline(path, t.Baseline)
		if t.CreditClaimed && toolAssessmentIncomplete(t.Assessment) {
			v.add("aviation.tool.incomplete_credit_assessment", path, "credit-claiming tool requires criterion, TQL, operational requirements, and evidence")
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func toolAssessmentIncomplete(a ToolAssessment) bool {
	if a.Criterion == "" || a.TQL == "" {
		return true
	}
	return len(a.OperationalRequirements) == 0 || len(a.Evidence) == 0
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkRequirementAssurance() {
	for _, r := range v.a.RequirementAssurance {
		path := "aviation.requirementAssurance." + r.RequirementID
		if !v.reqs[r.RequirementID] {
			v.add("aviation.reference.requirement", path, "canonical requirement does not resolve")
		}
		v.requireBaseline(path, r.Baseline)
		if r.Derived && derivedFeedbackIncomplete(r) {
			v.add("aviation.derived.incomplete_safety_feedback", path, "derived requirement requires rationale and safety feedback recipient, status, and evidence")
		}
		v.checkParentRefs(r, path)
	}
}

// TRLC-LINKS: REQ-EMG-054
func derivedFeedbackIncomplete(r RequirementAssurance) bool {
	if r.DerivedRationale == "" || r.SafetyFeedbackRecipient == "" {
		return true
	}
	return r.SafetyFeedbackStatus == "" || len(r.SafetyFeedbackEvidence) == 0
}

// checkParentRefs checks the parent requirements of r. The architecture and
// safety-feedback references are checked once per parent, so a record with
// several parents reports their problems that many times.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkParentRefs(r RequirementAssurance, path string) {
	for _, parent := range r.ParentRefs {
		if !v.reqs[parent] {
			v.add("aviation.reference.parent_requirement", path, "parent requirement "+parent+" does not resolve")
		}
		architectureOK := func(ref string) bool { return !v.haveArchitects || v.architectureIDs[ref] }
		v.checkRefs(r.ArchitectureRefs, architectureOK, aviationRefRule{"aviation.reference.architecture", path, "architecture reference"})
		v.checkRefs(r.SafetyFeedbackEvidence, v.resolves, evidenceRule(path, "safety feedback evidence"))
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkExecutions() {
	for _, e := range v.a.Executions {
		path := "aviation.executions." + e.ID
		if !v.procedures[e.ProcedureRef] {
			v.add("aviation.reference.procedure", path, "procedure reference does not resolve")
		}
		v.checkExecutionBuild(e, path)
		v.requireBaseline(path, e.Baseline)
		for _, ref := range e.CaseRefs {
			v.checkExecutionCase(e, ref)
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkExecutionBuild(e VerificationExecution, path string) {
	build, ok := v.builds[e.Build]
	if !ok {
		v.add("aviation.reference.build", path, "build reference does not resolve")
	} else if build.Baseline != e.Baseline {
		v.add("aviation.execution.baseline_mismatch", path, "execution baseline does not match build baseline")
	}
}

// checkExecutionCase checks one verification case of e.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkExecutionCase(e VerificationExecution, ref string) {
	path := "aviation.executions." + e.ID
	c, ok := v.cases[ref]
	if !ok {
		v.add("aviation.reference.case", path, "verification case reference does not resolve")
		return
	}
	if c.ProcedureRef != e.ProcedureRef {
		v.add("aviation.execution.procedure_mismatch", path, "execution procedure does not match verification case procedure")
	}
	if c.IndependenceRequired && (e.Producer == e.Verifier || len(e.ApprovalRefs) == 0) {
		v.add("aviation.independence.missing", path, "independent execution requires distinct producer/verifier and approval evidence")
	}
	v.checkExecutionApprovals(e, path)
}

// checkExecutionApprovals checks the approval references of e; the evidence
// is checked once per approval.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkExecutionApprovals(e VerificationExecution, path string) {
	for _, approval := range e.ApprovalRefs {
		if !v.approvals[approval] {
			v.add("aviation.reference.approval", path, "approval reference lacks evidence or does not resolve")
		}
		v.checkRefs(e.Evidence, v.resolves, evidenceRule(path, "evidence reference"))
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkCoverage() {
	for _, c := range v.a.Coverage {
		path := "aviation.coverage." + c.ID
		if v.a.Profile.SoftwareLevel == "C" && c.Metric != "statement" {
			v.add("aviation.coverage.metric", path, "DAL C coverage metric must be statement")
		}
		if c.Total != c.Covered+c.Uncovered {
			v.add("aviation.coverage.arithmetic", path, "total must equal covered plus uncovered")
		}
		v.checkCoverageDispositions(c, path)
		if !v.baselines[c.Baseline] || v.builds[c.Build].Baseline != c.Baseline {
			v.add("aviation.coverage.configuration", path, "coverage must reference an exact matching baseline and build")
		}
		v.checkCoverageTools(c, path)
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkCoverageDispositions(c StructuralCoverage, path string) {
	sum, open := 0, false
	for _, x := range c.Dispositions {
		sum += x.Count
		open = open || isOpenDisposition(x)
	}
	if sum != c.Uncovered {
		v.add("aviation.coverage.disposition_count", path, "dispositions must fully account for uncovered count")
	}
	if (c.Status == "accepted" || c.Status == "authority-accepted") && open {
		v.add("aviation.coverage.open_at_closure", path, "accepted coverage cannot retain open uncovered code")
	}
}

// TRLC-LINKS: REQ-EMG-054
func isOpenDisposition(x UncoveredDisposition) bool {
	return x.Disposition == "open" || x.Status == "declared" || x.Status == "planned"
}

// checkCoverageTools checks the tool references of c. The evidence is checked
// once per tool reference.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkCoverageTools(c StructuralCoverage, path string) {
	for _, ref := range c.ToolRefs {
		if !v.tools[ref] {
			v.add("aviation.reference.tool", path, "tool reference "+ref+" does not resolve")
		}
		v.checkRefs(c.Evidence, v.resolves, evidenceRule(path, "evidence reference"))
	}
}

// checkObjectives checks the evidence of every objective and its authority
// state. The document-wide reference checks run once per objective evidence
// reference, so the diagnostics they report repeat accordingly.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkObjectives() {
	for _, obj := range v.a.Objectives {
		v.checkObjectiveEvidence(obj)
		if obj.State == "authority-accepted" && !hasAuthorityAcceptance(v.a.Liaison, obj.ID) {
			v.add("aviation.authority.acceptance_missing", "aviation.objectives."+obj.ID, "authority-accepted state requires an authored authority record")
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkObjectiveEvidence(obj ObjectiveReference) {
	for _, ref := range obj.EvidenceRefs {
		if _, ok := v.ids[ref]; !ok {
			v.add("aviation.reference.evidence", "aviation.objectives."+obj.ID, "evidence reference "+ref+" does not resolve")
		}
		v.checkRecordReferences()
	}
}

// checkRecordReferences checks the references of the lifecycle data, tools,
// reviews, approvals, independence records, problem reports, audits and
// liaison records.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkRecordReferences() {
	a := v.a
	for _, item := range a.LifecycleData {
		v.checkLifecycleReferences(item)
	}
	for _, t := range a.Tools {
		v.checkRefs(t.Assessment.Evidence, v.resolves, evidenceRule("aviation.tools."+t.ID, "assessment evidence"))
	}
	for _, r := range a.Reviews {
		v.checkRefs(joinRefs(r.SubjectRefs, r.Evidence), v.resolves, evidenceRule("aviation.reviews."+r.ID, "reference"))
	}
	for _, approval := range a.Approvals {
		v.checkRefs(joinRefs(approval.SubjectRefs, approval.Evidence), v.resolves, evidenceRule("aviation.approvals."+approval.ID, "reference"))
	}
	v.checkClosingReferences()
}

// checkClosingReferences checks the references of the independence records,
// problem reports, audits and liaison records.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkClosingReferences() {
	a := v.a
	for _, r := range a.Independence {
		v.checkIndependenceReferences(r)
	}
	for _, report := range a.ProblemReports {
		v.checkRefs(joinRefs(report.AffectedRefs, report.Evidence), v.resolves, evidenceRule("aviation.problemReports."+report.ID, "reference"))
	}
	for _, audit := range a.SQAAudits {
		v.checkRefs(audit.Evidence, v.resolves, evidenceRule("aviation.sqaAudits."+audit.ID, "evidence reference"))
	}
	for _, l := range a.Liaison {
		v.checkRefs(l.SubmissionRefs, v.resolves, aviationRefRule{"aviation.reference.submission", "aviation.liaison." + l.ID, "submission reference"})
	}
}

// TRLC-LINKS: REQ-EMG-054
func joinRefs(first, second []string) []string {
	return append(append([]string{}, first...), second...)
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkLifecycleReferences(item LifecycleDataRecord) {
	path := "aviation.lifecycleData." + item.ID
	if item.Baseline != "" {
		v.requireBaseline(path, item.Baseline)
	}
	for _, ref := range item.Approvals {
		if !v.approvals[ref] {
			v.add("aviation.reference.approval", path, "approval reference lacks evidence or does not resolve")
		}
	}
	v.checkRefs(item.ObjectiveRefs, v.resolves, aviationRefRule{"aviation.reference.objective", path, "objective reference"})
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkIndependenceReferences(r IndependenceRecord) {
	path := "aviation.independence." + r.ID
	if !v.resolves(r.SubjectRef) {
		v.add("aviation.reference.subject", path, "subject reference does not resolve")
	}
	v.checkRefs(r.Evidence, v.resolves, evidenceRule(path, "evidence reference"))
}

// checkAuthorityStates reports every record in the authority-accepted state
// that has no authored authority record.
//
// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkAuthorityStates() {
	a := v.a
	var targets []aviationAuthorityTarget
	targets = append(targets, authorityTargets(a.LifecycleData, func(x LifecycleDataRecord) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.State, "aviation.lifecycleData." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.RequirementAssurance, func(x RequirementAssurance) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.RequirementID, x.State, "aviation.requirementAssurance." + x.RequirementID}
	})...)
	targets = append(targets, authorityTargets(a.VerificationCases, func(x AirborneVerificationCase) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.State, "aviation.verificationCases." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.Procedures, func(x VerificationProcedure) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.State, "aviation.procedures." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.Executions, func(x VerificationExecution) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.Status, "aviation.executions." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.Baselines, func(x ConfigurationBaseline) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.State, "aviation.baselines." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.Builds, func(x ConfigurationBuild) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.Status, "aviation.builds." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.Coverage, func(x StructuralCoverage) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.Status, "aviation.coverage." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.Tools, func(x AirborneTool) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.Status, "aviation.tools." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.Reviews, func(x AirborneReview) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.State, "aviation.reviews." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.Approvals, func(x AirborneApproval) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.State, "aviation.approvals." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.ProblemReports, func(x ProblemReport) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.State, "aviation.problemReports." + x.ID}
	})...)
	targets = append(targets, authorityTargets(a.SQAAudits, func(x SQAAudit) aviationAuthorityTarget {
		return aviationAuthorityTarget{x.ID, x.State, "aviation.sqaAudits." + x.ID}
	})...)
	for _, t := range targets {
		if t.state == "authority-accepted" && !hasAuthorityAcceptance(a.Liaison, t.id) {
			v.add("aviation.authority.acceptance_missing", t.path, "authority-accepted state requires an authored authority record")
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func authorityTargets[T any](items []T, target func(T) aviationAuthorityTarget) []aviationAuthorityTarget {
	out := make([]aviationAuthorityTarget, 0, len(items))
	for _, item := range items {
		out = append(out, target(item))
	}
	return out
}

// TRLC-LINKS: REQ-EMG-054
func (v *aviationValidator) checkLiaisonRecords() {
	for _, l := range v.a.Liaison {
		if l.AuthorityState == "authority-accepted" && l.AuthorityRecord.ID == "" {
			v.add("aviation.authority.record_missing", "aviation.liaison."+l.ID, "authority-accepted liaison state requires an authored authority record")
		}
	}
}

// TRLC-LINKS: REQ-EMG-054
func hasAuthorityAcceptance(records []CertificationLiaison, subject string) bool {
	for _, record := range records {
		if authorityAccepts(record, subject) {
			return true
		}
	}
	return false
}

// authorityAccepts reports whether record carries an authority acceptance
// that covers subject (a record without submission references covers all).
//
// TRLC-LINKS: REQ-EMG-054
func authorityAccepts(record CertificationLiaison, subject string) bool {
	if record.AuthorityState != "authority-accepted" || record.AuthorityRecord.ID == "" {
		return false
	}
	return len(record.SubmissionRefs) == 0 || containsString(record.SubmissionRefs, subject)
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

// aviationStateCodes are the evidence state and completeness codes of
// ValidateAviation: with engmod.yml lifecycle they are validation
// diagnostics keyed by the life-cycle tier, not load failures (ADR-EMG-035).
var aviationStateCodes = map[string]bool{
	"aviation.lifecycle.missing_category":         true,
	"aviation.tool.incomplete_credit_assessment":  true,
	"aviation.derived.incomplete_safety_feedback": true,
}

// AviationStateCode reports whether an aviation diagnostic code is a state
// or completeness code (tier keyed with lifecycle) rather than a structural
// code (references, duplicates, hashes, arithmetic, mismatches), which stays
// load-fatal at every tier.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-AIRBORNE-ASSURANCE-EXPORTER, FU-MODEL-LOADER
func AviationStateCode(code string) bool {
	return aviationStateCodes[code]
}

// SplitAviationDiagnostics returns the structural (load-fatal) and the state
// diagnostics of ValidateAviation.
// TRLC-LINKS: REQ-EMG-132
// ENGMODEL-LINKS: FU-AIRBORNE-ASSURANCE-EXPORTER, FU-MODEL-LOADER
func SplitAviationDiagnostics(diagnostics []AviationDiagnostic) (structural, state []AviationDiagnostic) {
	for _, d := range diagnostics {
		if AviationStateCode(d.Code) {
			state = append(state, d)
		} else {
			structural = append(structural, d)
		}
	}
	return structural, state
}

// TRLC-LINKS: REQ-EMG-055
func IsOpenEvidenceState(state string) bool { return openEvidenceStates[state] }

var openEvidenceStates = map[string]bool{"declared": true, "planned": true, "executed": true, "reviewed": true}

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
