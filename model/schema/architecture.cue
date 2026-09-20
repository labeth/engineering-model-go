package model

#ModelMeta: {
	id:             string & !=""
	title?:         string
	introduction?:  string
	baseCatalogRef?: string & !=""
}

#FunctionalGroup: {
	id?:          string
	name?:        string
	description?: string
	prose?:       string
	tags?:        #StringList
}

#FunctionalUnit: {
	id?:    string
	name?:  string
	group?: string
	tags?:  #StringList
	prose?: string
}

#Actor: {
	id?:          string
	name?:        string
	description?: string
}

#AttackVector: #Actor

#ReferencedElement: {
	id?:          string
	kind?:        string
	layer?:       string
	name?:        string
	description?: string
	version?:     string
	date?:        string
	uri?:         string
	publisher?:   string
}

#Mapping: {
	type?:        string
	from?:        string
	to?:          string
	description?: string
}

#Interface: {
	id?:        string
	name?:      string
	protocol?:  string
	endpoint?:  string
	schemaRef?: string
	owner?:     string
}

#DataObject: {
	id?:              string
	name?:            string
	termRef?:         string
	schemaRef?:       string
	sensitivity?:     string
	classification?:  string
	regulatoryTags?:  #StringList
	retention?:       string
	confidentiality?: string
	integrity?:       string
	availability?:    string
}

#DeploymentTarget: {
	id?:          string
	name?:        string
	environment?: string
	region?:      string
	account?:     string
	cluster?:     string
	namespace?:   string
	trustZone?:   string
}

#HardwareItem: {
	id?:          string
	name?:        string
	kind?:        string
	partNumber?:  string
	supplier?:    string
	safetyLevel?: string
	description?: string
	hosts?:       #StringList
}

#HardwareInterface: {
	id?:                   string
	name?:                 string
	busType?:              string
	from?:                 string
	to?:                   string
	direction?:            string
	softwareInterfaceRef?: string
	description?:          string
}

#Control: {
	id?:          string
	name?:        string
	category?:    string
	description?: string
}

#ComplianceProfile: {
	id?:          string
	href?:        string
	catalogHref?: string
}

#ComplianceMapping: {
	id?:                   string
	profileRef?:           string
	controlIds?:           #StringList
	modelControlRef?:      string
	appliesTo?:            #StringList
	implementationType?:   string
	implementationStatus?: string
	status?:               string
	narrative?:            string
	rationale?:            string
	evidence?: [...#Evidence]
	inheritedFrom?:    #StringList
	responsibleRoles?: #StringList
}

#ComplianceModel: {
	profiles?: [...#ComplianceProfile]
	mappings?: [...#ComplianceMapping]
}

#Risk: {
	id?:              string
	title?:           string
	statement?:       string
	status?:          string
	likelihood?:      string
	impact?:          string
	response?:        string
	owner?:           string
	appliesTo?:       #StringList
	relatedControls?: #StringList
	attackVectors?:   #StringList
	threatScenarios?: #StringList
	evidence?: [...#Evidence]
	residualRisk?: string
	rationale?:    string
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
}

#TrustBoundary: {
	id?:           string
	name?:         string
	description?:  string
	boundaryType?: string
	parentRef?:    string
	members?:      #StringList
}

#State: #Actor
#Event: #Actor

#FlowStep: {
	id?:                  string
	ref?:                 string
	kind?:                string
	flowType?:            string
	direction?:           string
	frequency?:           string
	sourceRef?:           string
	destinationRef?:      string
	action?:              string
	channel?:             string
	protocol?:            string
	dataIn?:              #StringList
	dataOut?:             #StringList
	dataRefs?:            #StringList
	interfaceRef?:        string
	trustBoundaryRef?:    string
	boundaryCrossing?:    bool
	authentication?:      string
	encryptionInTransit?: string
	integrityProtection?: string
	next?:                #StringList
	onError?:             #StringList
	async?:               bool
	optional?:            bool
}

#Flow: {
	id?:                  string
	title?:               string
	kind?:                string
	methodology?:         string
	direction?:           string
	frequency?:           string
	sourceRef?:           string
	destinationRef?:      string
	protocol?:            string
	channel?:             string
	authentication?:      string
	encryptionInTransit?: string
	integrityProtection?: string
	dataRefs?:            #StringList
	description?:         string
	criticality?:         string
	threats?:             #StringList
	entry?:               #StringList
	exits?:               #StringList
	steps?: [...#FlowStep]
	properties?: [string]: string
}

#ThreatScenario: {
	id?:               string
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
}

#AuthoredArchitecture: {
	functionalGroups?: [...#FunctionalGroup]
	functionalUnits?: [...#FunctionalUnit]
	actors?: [...#Actor]
	attackVectors?: [...#AttackVector]
	referencedElements?: [...#ReferencedElement]
	interfaces?: [...#Interface]
	dataObjects?: [...#DataObject]
	deploymentTargets?: [...#DeploymentTarget]
	hardwareItems?: [...#HardwareItem]
	hardwareInterfaces?: [...#HardwareInterface]
	controls?: [...#Control]
	risks?: [...#Risk]
	poamItems?: [...#POAMItem]
	trustBoundaries?: [...#TrustBoundary]
	states?: [...#State]
	events?: [...#Event]
	flows?: [...#Flow]
	threatScenarios?: [...#ThreatScenario]
	threatAssumptions?: [...#ThreatAssumption]
	threatOutOfScope?: [...#ThreatOutOfScope]
	threatMitigations?: [...#ThreatMitigation]
	controlVerifications?: [...#ControlVerification]
	mappings?: [...#Mapping]
}

#InferenceHints: {
	runtimeSources?:           #StringList
	codeSources?:              #StringList
	expectedRuntimeKinds?:     #StringList
	ownershipResolutionOrder?: #StringList
}

#View: {
	id?:                        string
	kind?:                      string
	roots?:                     #StringList
	authoredStatus?:            string
	authoredStatusExplanation?: string
	includeKinds?:              #StringList
	excludeKinds?:              #StringList
	includeMappings?:           #StringList
	excludeMappings?:           #StringList
	maxDepth?:                  int
	audience?:                  string
	abstraction?:               string
}

#NAFStakeholder: {
	actorRef: string & !=""
	role:     string & !=""
}

#NAFConcern: {
	id:              string & !=""
	name:            string & !=""
	description?:    string
	stakeholderRefs: [...string & !=""]
}

#NAFViewpoint: "C1" | "C2" | "C3" | "C4" | "C5" | "C7" | "C8" | "Cr" |
	"S1" | "S2" | "S3" | "S4" | "S5" | "S6" | "S7" | "S8" | "Sr" | "C1-S1" |
	"L1" | "L2" | "L2-L3" | "L3" | "L4" | "L5" | "L6" | "L7" | "L8" | "Lr" |
	"P1" | "P2" | "P3" | "P4" | "L4-P4" | "P5" | "P6" | "P7" | "P8" | "Pr" |
	"A1" | "A2" | "A3" | "A4" | "A5" | "A6" | "A7" | "A8" | "Ar"

#NAFProduct: {
	id:          string & !=""
	viewpoint:   #NAFViewpoint
	title:       string & !=""
	viewRef:     string & !=""
	concernRefs: [...string & !=""]
}

#NAFProfile: {
	framework:               "NAF"
	version:                 "4.1"
	architectureDescription: string & !=""
	stakeholders:            [...#NAFStakeholder]
	concerns:                [...#NAFConcern]
	products:                [...#NAFProduct]
}

#ContractEntry: {
	id?:   string
	kind?: string
	ref?:  string
	note?: string
}

#ContractModel: {
	provides?: [...#ContractEntry]
	requires?: [...#ContractEntry]
}

#Subsystem: {
	id?:          string
	name?:        string
	dependency?:  string
	publication?: string
	source?:      #SubsystemSource
	ref?:         string
	git?:         string
	rev?:         string
	path?:        string
	description?: string
}

#SubsystemSource: {
	local?:  #LocalSubsystemSource
	git?:    #GitSubsystemSource
	module?: #ModuleSubsystemSource
}

#LocalSubsystemSource: {
	path: string & !=""
}

#GitSubsystemSource: {
	url:       string & !=""
	revision?: string
	path?:     string
}

#ModuleSubsystemSource: {
	path:       string & !=""
	version:    string & !=""
	directory?: string
}

#Allocation: {
	requirement?: string
	to?:          string
	target?:      string
	rationale?:   string
}

#Satisfaction: {
	need?: string
	by?:   string
}

#CompositionModel: {
	subsystems?: [...#Subsystem]
	allocations?: [...#Allocation]
	satisfactions?: [...#Satisfaction]
}

#ArchitectureDocument: {
	schemaVersion?:        #SchemaVersion
	model:                 #ModelMeta
	semantics?:            #SemanticContent
	authoredArchitecture?: #AuthoredArchitecture
	compliance?:           #ComplianceModel
	contract?:             #ContractModel
	composition?:          #CompositionModel
	inferenceHints?:       #InferenceHints
	naf?:                  #NAFProfile
	views?: [...#View]
}
