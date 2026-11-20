// decisions.cue validates the decisions document, documents.decisions in engmod.yml
// (#DecisionsDocument), and holds the types only that document uses.
package model

#DecisionsDocument: {
	schemaVersion: #SchemaVersion
	decisions: [...#Decision]
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
