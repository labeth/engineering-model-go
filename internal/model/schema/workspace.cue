// workspace.cue validates the local workspace file engmod.work.yml
// (#WorkspaceDocument) and holds the types only that document uses.
package model

#WorkspaceReplacement: {
	module: string & !=""
	path:   string & !=""
}

#WorkspaceDocument: {
	schemaVersion: #SchemaVersion
	replacements: [...#WorkspaceReplacement]
}
