package model

#SchemaVersion: 3

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
