package model

#LintRun: {
	id?:         string
	mode?:       string
	commaAsAnd?: bool
	catalogRef?: string
}

#Requirement: {
	id?:                   string
	title?:                string
	text?:                 string
	notes?:                string
	category?:             string
	rationale?:            string
	sourceRefs?:           #StringList
	verificationMethods?:  [...("analysis" | "demonstration" | "inspection" | "review" | "test")]
	verificationCriteria?: string
	priority?:             string
	criticality?:          string
	status?:               "draft" | "proposed" | "approved" | "implemented" | "verified" | "rejected" | "retired"
	derived?:              bool
	derivedRationale?:     string
	tags?:                 #StringList
	appliesTo?:            #StringList
}

#Expected: {
	id?:      string
	pattern?: string
}

#RequirementsDocument: {
	schemaVersion: #SchemaVersion
	lintRun: #LintRun
	requirements: [...#Requirement]
	expected?: [...#Expected]
}
