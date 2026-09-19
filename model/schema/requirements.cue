package model

#LintRun: {
	id?:         string
	mode?:       string
	commaAsAnd?: bool
	catalogRef?: string
}

#Requirement: {
	id?:        string
	text?:      string
	notes?:     string
	appliesTo?: #StringList
}

#Expected: {
	id?:      string
	pattern?: string
}

#RequirementsDocument: {
	schemaVersion?: #SchemaVersion
	lintRun: #LintRun
	requirements: [...#Requirement]
	expected?: [...#Expected]
}
