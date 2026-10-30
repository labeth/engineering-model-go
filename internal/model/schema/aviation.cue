package model

#EvidenceState: "declared" | "planned" | "executed" | "reviewed" | "accepted" | "not-applicable" | "authority-accepted"
#EvidenceRef: {
	id: #NonEmpty
	path?: string
	uri?: string
	version?: string
	hash?: string
}
#AviationProfile: {
	id: #StableID
	standard: "DO-178C" | "ED-12C"
	edition: #NonEmpty
	softwareLevel: "A" | "B" | "C" | "D" | "E"
	softwareItem: #NonEmpty
	applicant: #NonEmpty
	authority: #NonEmpty
	certificationBasis: #NonEmpty
	safetyAllocationReference: #NonEmpty
	supplements?: [...#NonEmpty]
	claimsDisclaimer: #NonEmpty
}
#LifecycleData: {
	id: #StableID
	categoryCode: #NonEmpty
	categoryName: #NonEmpty
	source?: #EvidenceRef
	baseline?: #StableID
	owner: #NonEmpty
	state: #EvidenceState
	approvals?: [...#StableID]
	objectiveRefs?: [...#StableID]
}
#ObjectiveRef: {
	id: #StableID
	sourceReference: #NonEmpty
	applicability: "applicable" | "not-applicable" | "pending"
	independenceRequired: bool
	rationale: #NonEmpty
	evidenceRefs?: [...#StableID]
	state: #EvidenceState
}
#RequirementAssurance: {
	requirementId: #StableID
	level: "system-allocated" | "HLR" | "LLR"
	derived: bool
	derivedRationale?: string
	parentRefs?: [...#StableID]
	architectureRefs?: [...#StableID]
	codeRefs?: [...#NonEmpty]
	verificationCriteria: #NonEmpty
	safetyFeedbackRecipient?: string
	safetyFeedbackStatus?: #EvidenceState
	safetyFeedbackEvidence?: [...#StableID]
	baseline: #StableID
	state: #EvidenceState
}
#VerificationCase: {
	id: #StableID
	method: "review" | "analysis" | "test"
	intent: "normal" | "robustness"
	requirementRefs: [#StableID, ...#StableID]
	expectedResult: #NonEmpty
	procedureRef: #StableID
	independenceRequired: bool
	state: #EvidenceState
}
#Procedure: {
	id: #StableID
	caseRefs: [#StableID, ...#StableID]
	steps: [#NonEmpty, ...#NonEmpty]
	environment: #NonEmpty
	target: #NonEmpty
	baseline: #StableID
	state: #EvidenceState
}
#Execution: {
	id: #StableID
	procedureRef: #StableID
	caseRefs: [#StableID, ...#StableID]
	baseline: #StableID
	build: #StableID
	environment: #NonEmpty
	target: #NonEmpty
	producer: #NonEmpty
	verifier: #NonEmpty
	status: #EvidenceState
	result: #NonEmpty
	evidence: [#StableID, ...#StableID]
	approvalRefs?: [...#StableID]
}
#Artifact: {path: #NonEmpty, hash: #NonEmpty}
#Baseline: {
	id: #StableID
	version: #NonEmpty
	hash: #NonEmpty
	artifacts: [#Artifact, ...#Artifact]
	state: #EvidenceState
}
#Build: {
	id: #StableID
	baseline: #StableID
	hash: #NonEmpty
	artifacts: [#Artifact, ...#Artifact]
	compiler: #NonEmpty
	compilerOptions?: [...string]
	target: #NonEmpty
	environment: #NonEmpty
	reproducibility: #NonEmpty
	status: #EvidenceState
}
#Disposition: {
	id: #StableID
	disposition: "additional-test" | "dead" | "deactivated" | "unintended" | "open"
	count: int & >=0
	rationale: #NonEmpty
	evidence?: [...#StableID]
	reviewer: #NonEmpty
	status: #EvidenceState
}
#Coverage: {
	id: #StableID
	metric: "statement" | "decision" | "mcdc"
	baseline: #StableID
	build: #StableID
	toolRefs: [#StableID, ...#StableID]
	total: int & >=0
	covered: int & >=0
	uncovered: int & >=0
	dispositions?: [...#Disposition]
	evidence: [#StableID, ...#StableID]
	reviewer: #NonEmpty
	status: #EvidenceState
}
#ToolAssessment: {
	criterion?: string
	tql?: string
	operationalRequirements?: [...#NonEmpty]
	evidence?: [...#StableID]
}
#Tool: {
	id: #StableID
	name: #NonEmpty
	version: #NonEmpty
	intendedUse: #NonEmpty
	creditClaimed: bool
	assessment?: #ToolAssessment
	environment: #NonEmpty
	baseline: #StableID
	status: #EvidenceState
}
#Review: {id: #StableID, subjectRefs: [#StableID, ...#StableID], reviewer: #NonEmpty, result: #NonEmpty, evidence: [#StableID, ...#StableID], state: #EvidenceState}
#Approval: {id: #StableID, subjectRefs: [#StableID, ...#StableID], approver: #NonEmpty, evidence: [#StableID, ...#StableID], state: #EvidenceState}
#Independence: {id: #StableID, subjectRef: #StableID, producer: #NonEmpty, verifier: #NonEmpty, approvalRef: #StableID, evidence: [#StableID, ...#StableID], state: #EvidenceState}
#Problem: {id: #StableID, summary: #NonEmpty, affectedRefs?: [...#StableID], disposition: #NonEmpty, correctiveAction: #NonEmpty, evidence?: [...#StableID], state: #EvidenceState}
#SQAAudit: {id: #StableID, scope: #NonEmpty, auditor: #NonEmpty, findings?: [...#NonEmpty], correctiveActions?: [...#NonEmpty], evidence: [#StableID, ...#StableID], state: #EvidenceState}
#Liaison: {id: #StableID, milestone: #NonEmpty, submissionRefs?: [...#StableID], findings?: [...#NonEmpty], authorityState: #EvidenceState, authorityRecord?: #EvidenceRef}
#AviationDocument: {
	schemaVersion: #SchemaVersion
	aviation: {
		profile: #AviationProfile
		lifecycleData?: [...#LifecycleData]
		objectives?: [...#ObjectiveRef]
		requirementAssurance?: [...#RequirementAssurance]
		verificationCases?: [...#VerificationCase]
		procedures?: [...#Procedure]
		executions?: [...#Execution]
		baselines?: [...#Baseline]
		builds?: [...#Build]
		coverage?: [...#Coverage]
		tools?: [...#Tool]
		reviews?: [...#Review]
		approvals?: [...#Approval]
		independence?: [...#Independence]
		problemReports?: [...#Problem]
		sqaAudits?: [...#SQAAudit]
		liaison?: [...#Liaison]
	}
}
