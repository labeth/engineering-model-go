// common.cue validates no model file on its own: it holds the types that two
// or more model files use (schema version, string lists, IDs, evidence, actors).
package model

#SchemaVersion: 3

#StringList: [...string]

#Evidence: {
	path?:        string
	description?: string
}

#Actor: {
	id?:          string
	name?:        string
	description?: string
}

#NonEmpty: string & !=""

#StableID: #NonEmpty & =~"^[A-Z][A-Z0-9-]*$"

#ExactVersion: #NonEmpty & =~"^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\\+[0-9A-Za-z.-]+)?$"
