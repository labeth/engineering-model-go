// compliance.cue validates the compliance document, documents.compliance in engmod.yml
// (#ComplianceDocument), and holds the types only that document uses.
package model

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

#ComplianceDocument: {
	schemaVersion: #SchemaVersion
	compliance:    #ComplianceModel
}
