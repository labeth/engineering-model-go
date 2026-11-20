// behavior.cue validates the behavior document, documents.behavior in engmod.yml
// (#BehaviorDocument), and holds the types only that document uses.
package model

#Event: {
	#Actor
	requirements?: #StringList
}

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
	id:            #StableID
	name?:         string
	description?:  string
	final?:        bool
	requirements?: #StringList
}

#MachineTransition: {
	id:            #StableID
	from:          #NonEmpty
	to:            #NonEmpty
	on?:           #NonEmpty
	guard?:        #NonEmpty
	effect?:       #NonEmpty
	description?:  string
	requirements?: #StringList
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
