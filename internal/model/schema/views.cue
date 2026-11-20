// views.cue validates the views document, documents.views in engmod.yml (#ViewsDocument),
// and the #DesignDocument the loader builds from its design section.
package model

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

#DesignView: {
	title?:     string
	narrative?: string
}

#DesignFunctionalGroup: {
	id?: string
	views?: [string]: #DesignView
}

#DesignFunctionalUnit: {
	id?:    string
	group?: string
	views?: [string]: #DesignView
}

#DesignModel: {
	id?:    string
	title?: string
	functionalGroups?: [...#DesignFunctionalGroup]
	functionalUnits?: [...#DesignFunctionalUnit]
}

#DesignDocument: {
	schemaVersion: #SchemaVersion
	design: #DesignModel
}

#DocumentControl: {
	identifier:             #NonEmpty
	revision:               #NonEmpty
	status:                 "draft" | "in-review" | "approved" | "released" | "superseded" | "withdrawn"
	issuedBy:               #NonEmpty
	issueDate:              #NonEmpty & =~"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"
	language:               #NonEmpty
	documentType:           #NonEmpty
	confidentiality:        #NonEmpty
	securityClassification: #NonEmpty
	exportControlled:       bool
	countryOfOrigin:        #NonEmpty
	confidentialityStamp:   bool
}

#DocumentSection: {
	id:           #NonEmpty
	title:        #NonEmpty
	narrative:    string
	includeRefs?: #StringList
}

#DocumentDefinition: {
	id:               #StableID
	title:            #NonEmpty
	kind:             #NonEmpty
	purpose:          #NonEmpty
	audience?:        #StringList
	stakeholderRefs?: #StringList
	referenceRefs?:   #StringList
	contentRefs?:     #StringList
	sections?:        [...#DocumentSection]
	control:          #DocumentControl
}

#ViewsDocument: {
	schemaVersion: #SchemaVersion
	views:         [...#View]
	naf?:          #NAFProfile
	design?:       #DesignModel
	documents?:    [...#DocumentDefinition]
}
