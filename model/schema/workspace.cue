package model

#WorkspaceReplacement: {
	module: string & !=""
	path:   string & !=""
}

#WorkspaceDocument: {
	schemaVersion?: #SchemaVersion
	replacements: [...#WorkspaceReplacement]
}
