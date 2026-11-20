// architecture.cue validates the architecture document, documents.architecture in engmod.yml
// (#ArchitectureInputDocument), and the #ArchitectureDocument the loader assembles.
package model

#ModelMeta: {
	id:            string & !=""
	title?:        string
	introduction?: string
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
	events?: [...#Event]
	flows?: [...#Flow]
	threatScenarios?: [...#ThreatScenario]
	threatAssumptions?: [...#ThreatAssumption]
	threatOutOfScope?: [...#ThreatOutOfScope]
	threatMitigations?: [...#ThreatMitigation]
	controlVerifications?: [...#ControlVerification]
	mappings?: [...#Mapping]
}

#ContractEntry: {
	id?:   string
	kind?: string
	// On a provides entry ref is removed (ADR-EMG-075): decoded only so
	// validation reports model.provides_ref_removed with its remedy.
	ref?:  string
	note?: string
}

#ContractAssumption: {
	id:                    #NonEmpty & =~"^AOU-[A-Z0-9][A-Z0-9-]*$"
	text:                  #NonEmpty
	verificationCriteria?: string
}

#ContractModel: {
	provides?: [...#ContractEntry]
	requires?: [...#ContractEntry]
	assumptions?: [...#ContractAssumption]
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

// #Allocation is removed (ADR-EMG-078): it decodes only so validation reports
// model.allocations_removed with its remedy.
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
	schemaVersion:         #SchemaVersion
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

#InputSubsystem: {
	id:          string & !=""
	name?:       string
	dependency:  string & !=""
	publication: string & !=""
	description?: string
}

#InputComposition: {
	subsystems?:    [...#InputSubsystem]
	allocations?:   [...#Allocation]
	satisfactions?: [...#Satisfaction]
}

#ArchitectureInput: {
	functionalGroups?:   [...#FunctionalGroup]
	functionalUnits?:    [...#FunctionalUnit]
	actors?:             [...#Actor]
	referencedElements?: [...#ReferencedElement]
	interfaces?:         [...#Interface]
	dataObjects?:        [...#DataObject]
	deploymentTargets?:  [...#DeploymentTarget]
	hardwareItems?:      [...#HardwareItem]
	hardwareInterfaces?: [...#HardwareInterface]
	contract?:            #ContractModel
	composition?:         #InputComposition
	semantics?:           #SemanticContent
}

#ArchitectureInputDocument: {
	schemaVersion: #SchemaVersion
	architecture:  #ArchitectureInput
}

#Multiplicity: {
	lower?: int & >=0
	upper?: int & >=-1
	if lower != _|_ && upper != _|_ && upper >= 0 {
		upper: >=lower
	}
}

#SemanticQuantity: {
	value: number
	unit:  #MetamodelString & !=""
}

#SemanticValue: {
	kind:       #MetamodelString
	string?:    #MetamodelString
	integer?:   #MetamodelInteger
	real?:      #MetamodelReal
	boolean?:   #MetamodelBoolean
	reference?: #MetamodelString
	quantity?:  #SemanticQuantity
	if kind == "quantity" {
		quantity: #SemanticQuantity
	}
}

#SemanticExpression: {
	kind?:       #MetamodelString
	language?:   #MetamodelString
	operator?:   #MetamodelString
	value?:      #MetamodelString
	typedValue?: #SemanticValue
	operands?: [...#SemanticExpression]
}

#SemanticFeature: {
	name:          #MetamodelString
	kind?:         "attribute" | "port" | "parameter"
	type?:         #MetamodelString
	direction?:    #MetamodelString
	multiplicity?: #Multiplicity
	ordered?:      bool
	unique?:       bool
	conjugated?:   bool
	specializes?:  #StringList
	subsets?:      #StringList
	redefines?:    #StringList
	value?:        #SemanticExpression
	references?:   #StringList
}

#SemanticMetadata: {
	namespace: #MetamodelString
	type:      #MetamodelString
	target:    #MetamodelString
	properties?: [string]: #MetamodelValue
}

#SemanticElementBase: {
	id:            #MetamodelString & !=""
	name?:         #MetamodelString
	kind:          #MetamodelString & !=""
	metaclass?:    #MetamodelString & !=""
	properties?:   #ExtensionProperties
	namespace?:    #MetamodelString
	owner?:        #MetamodelString
	typeRef?:      #MetamodelString
	multiplicity?: #Multiplicity
	ordered?:      bool
	unique?:       bool
	conjugated?:   bool
	specializes?:  #StringList
	subsets?:      #StringList
	redefines?:    #StringList
	controlKind?:  #MetamodelString
	occurrenceId?: #MetamodelString
	portionOf?:    #MetamodelString
	variation?:    bool
	variants?:     #StringList
	references?:   #StringList
	features?: [...#SemanticFeature]
	metadata?: [...#SemanticMetadata]
	extensionNamespace?: #MetamodelString
	extension?:          #MetamodelString
	targets?:            #StringList
	if kind == "engineering_extension" {
		extensionNamespace: #MetamodelString & !=""
		extension:          #MetamodelString & !=""
		targets: [#MetamodelString, ...#MetamodelString]
	}
}

#SemanticElement: #SemanticElementBase & #OfficialMetamodelBinding

#SemanticRelationshipBase: {
	id:          #MetamodelString & !=""
	kind:        #MetamodelString & !=""
	metaclass?:  #MetamodelString & !=""
	properties?: #ExtensionProperties
	source:      #MetamodelString & !=""
	target:      #MetamodelString & !=""
	owner?:      #MetamodelString
	sourceType?: #MetamodelString
	itemRef?:    #MetamodelString
	triggers?:   #StringList
	guard?:      #SemanticExpression
	effect?:     #SemanticExpression
	metadata?: [...#SemanticMetadata]
}

#SemanticRelationship: #SemanticRelationshipBase & #OfficialMetamodelBinding

#SemanticImport: {
	namespace:   #MetamodelString
	imported:    #MetamodelString
	visibility?: #MetamodelString
	recursive?:  bool
}

#SemanticContent: {
	imports?: [...#SemanticImport]
	elements?: [...#SemanticElement]
	relationships?: [...#SemanticRelationship]
}
