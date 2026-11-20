// assurance.cue validates the assurance document, documents.assurance in engmod.yml
// (#AssuranceDocument), and holds the types only that document uses.
package model

#AttackVector: #Actor

#Control: {
	id?:          string
	name?:        string
	category?:    string
	description?: string
}

#Risk: {
	// alias::ID is reserved for entities imported by composition.
	id?:              #LocalID
	title?:           string
	statement?:       string
	status?:          string
	likelihood?:      string
	impact?:          string
	response?:        string
	owner?:           string
	appliesTo?:       #StringList
	// dependencies names engmod.yml dependency aliases the risk is about.
	dependencies?:    #StringList
	relatedControls?: #StringList
	attackVectors?:   #StringList
	threatScenarios?: #StringList
	evidence?: [...#Evidence]
	residualRisk?: string
	rationale?:    string
	acceptedBy?:   #AcceptanceRecord
}

// #LocalID is an authored entity ID: alias::ID is reserved for imported
// entities, so validation can tell them apart.
#LocalID: string & !~"::"

#AcceptanceRecord: {
	actor?:       string
	date?:        string
	statement?:   string
	decisionRef?: string
}

#POAMArtifact: #Evidence

#POAMItem: {
	id?:              string
	riskRef?:         string
	milestone?:       string
	dueDate?:         string
	status?:          string
	responsibleRole?: string
	artifacts?: [...#POAMArtifact]
	// defers binds the item to the diagnostics it covers: an open item with
	// milestone "tier:<project tier>" defers each named code (at path, when
	// given) to a warning with poamRef (ADR-EMG-033).
	defers?: [...#POAMDefer]
}

#POAMDefer: {
	code:  #NonEmpty
	path?: string
	// ref limits the defer to one requirement by ID; the validator resolves
	// it to the requirement's current path, so it survives reordering.
	ref?: #NonEmpty
}

#TrustBoundary: {
	id?:           string
	name?:         string
	description?:  string
	boundaryType?: string
	parentRef?:    string
	members?:      #StringList
}

#ThreatScenario: {
	id?:               #LocalID
	title?:            string
	summary?:          string
	category?:         string
	stride?:           string
	cwe?:              #StringList
	attackVectorRef?:  string
	appliesTo?:        #StringList
	flowRefs?:         #StringList
	entryPoint?:       string
	preconditions?:    #StringList
	exploitPath?:      #StringList
	likelihood?:       string
	impact?:           string
	severity?:         string
	status?:           string
	owner?:            string
	riskRef?:          string
	relatedControls?:  #StringList
	assumptionRefs?:   #StringList
	outOfScopeRefs?:   #StringList
	mitigationRefs?:   #StringList
	verificationRefs?: #StringList
	detection?:        #StringList
	evidence?: [...#Evidence]
	acceptedBy?: #AcceptanceRecord
}

#ThreatAssumption: {
	id?:        string
	title?:     string
	statement?: string
	status?:    string
	owner?:     string
	appliesTo?: #StringList
	rationale?: string
	evidence?: [...#Evidence]
}

#ThreatOutOfScope: {
	id?:        string
	title?:     string
	reason?:    string
	status?:    string
	owner?:     string
	appliesTo?: #StringList
	expiresOn?: string
	evidence?: [...#Evidence]
}

#ThreatMitigation: {
	id?:                string
	threatScenarioRef?: string
	controlRef?:        string
	status?:            string
	effectiveness?:     string
	owner?:             string
	notes?:             string
	verificationRefs?:  #StringList
	evidence?: [...#Evidence]
}

#ControlVerification: {
	id?:                 string
	controlRef?:         string
	threatScenarioRefs?: #StringList
	riskRefs?:           #StringList
	method?:             string
	status?:             string
	owner?:              string
	lastTested?:         string
	findings?:           #StringList
	evidence?: [...#Evidence]
	verificationRefs?: #StringList
}

#AssuranceModel: {
	attackVectors?:        [...#AttackVector]
	controls?:             [...#Control]
	risks?:                [...#Risk]
	poamItems?:            [...#POAMItem]
	trustBoundaries?:      [...#TrustBoundary]
	threatScenarios?:      [...#ThreatScenario]
	threatAssumptions?:    [...#ThreatAssumption]
	threatOutOfScope?:     [...#ThreatOutOfScope]
	threatMitigations?:    [...#ThreatMitigation]
	controlVerifications?: [...#ControlVerification]
}

#AssuranceDocument: {
	schemaVersion: #SchemaVersion
	assurance:     #AssuranceModel
}
