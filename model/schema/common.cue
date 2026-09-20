package model

#SchemaVersion: 2

#StringList: [...string]

#Evidence: {
	path?:        string
	description?: string
}

#CatalogEntry: {
	id?:         string
	name?:       string
	definition?: string
	aliases?:    #StringList
}

#Decision: {
	id?:           string
	title?:        string
	status?:       string
	date?:         string
	context?:      string
	decision?:     string
	consequences?: #StringList
}
