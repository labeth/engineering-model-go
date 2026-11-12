package model

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

#SupportedMappingKind: "contains" | "calls" | "publishes" | "subscribes" |
	"reads" | "writes" | "streams" | "allocated_to" |
	"deployed_to" | "satisfies" | "verified_by" | "depends_on" |
	"interacts_with" | "targets" | "implements" |
	"guarded_by" | "mitigated_by" | "bounded_by"

#TypedMapping: {
	type:        #SupportedMappingKind
	from:        string & !=""
	to:          string & !=""
	description?: string
}

#BehaviorModel: {
	events?:        [...#Event]
	flows?:         [...#Flow]
	relationships?: [...#TypedMapping]
	machines?:      [...#Machine]
}

// #Machine is a first-class state machine: one initial state, typed
// variables, guarded transitions and invariants. Guards, effects and invariant
// expressions are source strings parsed during semantic projection.
#Machine: {
	id:           #NonEmpty & =~"^SM-[A-Z0-9][A-Z0-9-]*$"
	name?:        string
	description?: string
	owner?:       string
	initial?:     string
	variables?: [...#MachineVariable]
	states?: [...#MachineState]
	transitions?: [...#MachineTransition]
	invariants?: [...#MachineInvariant]
}

#MachineVariable: {
	name:    #NonEmpty & =~"^[A-Za-z_][A-Za-z0-9_]*$"
	type:    "Integer" | "Boolean" | "String"
	initial: int | bool | string
}

#MachineState: {
	id:           #StableID
	name?:        string
	description?: string
	final?:       bool
}

#MachineTransition: {
	id:           #StableID
	from:         #NonEmpty
	to:           #NonEmpty
	on?:          #NonEmpty
	guard?:       #NonEmpty
	effect?:      #NonEmpty
	description?: string
}

#MachineInvariant: {
	id:            #StableID
	requirements?: #StringList
	expr:          #NonEmpty
	description?:  string
}

#BehaviorDocument: {
	schemaVersion: #SchemaVersion
	behavior:      #BehaviorModel
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

#ComplianceDocument: {
	schemaVersion: #SchemaVersion
	compliance:    #ComplianceModel
}

#ViewsDocument: {
	schemaVersion: #SchemaVersion
	views:         [...#View]
	naf?:          #NAFProfile
	design?:       #DesignModel
	documents?:    [...#DocumentDefinition]
}
