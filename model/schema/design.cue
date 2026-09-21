package model

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
	schemaVersion?: #SchemaVersion
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
