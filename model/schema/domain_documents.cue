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
	"reads" | "writes" | "streams" | "transitions_to" | "allocated_to" |
	"deployed_to" | "satisfies" | "verified_by" | "depends_on" |
	"interacts_with" | "targets" | "implements" | "triggered_by" |
	"guarded_by" | "mitigated_by" | "bounded_by"

#TypedMapping: {
	type:        #SupportedMappingKind
	from:        string & !=""
	to:          string & !=""
	description?: string
}

#BehaviorModel: {
	states?:        [...#State]
	events?:        [...#Event]
	flows?:         [...#Flow]
	relationships?: [...#TypedMapping]
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
}
