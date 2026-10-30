package model

#NonEmpty: string & !=""
#StableID: #NonEmpty & =~"^[A-Z][A-Z0-9-]*$"
#ModulePath: #NonEmpty & =~"@v[0-9]+$"
#ExactVersion: #NonEmpty & =~"^v(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)\\.(0|[1-9][0-9]*)(-[0-9A-Za-z.-]+)?(\\+[0-9A-Za-z.-]+)?$"

#ModuleIdentity: {
	path:         #ModulePath
	version:      #ExactVersion
	modelId:      #NonEmpty
	title:        #NonEmpty
	introduction: string
	kind:         #NonEmpty
}

#DocumentReferences: {
	catalog:      #NonEmpty
	requirements: #NonEmpty
	architecture: #NonEmpty
	behavior:     #NonEmpty
	assurance:    #NonEmpty
	compliance:   #NonEmpty
	views:        #NonEmpty
	decisions:    #NonEmpty
	aviation?:    #NonEmpty
}

#ManifestDependency: {
	alias:        #NonEmpty & =~"^[a-z][a-z0-9-]*$"
	path:         #ModulePath
	version:      #ExactVersion
	publications: [#NonEmpty, ...#NonEmpty]
}

#ManifestPublication: {
	id:           #NonEmpty
	architecture?: [...#StableID]
	behavior?:     [...#StableID]
	assurance?:    [...#StableID]
	compliance?:   [...#StableID]
	requirements?: [...#StableID]
	views?:        [...#StableID]
}

#ManifestDocument: {
	schemaVersion: #SchemaVersion
	module:        #ModuleIdentity
	documents:     #DocumentReferences
	dependencies?: [...#ManifestDependency]
	publications?: [...#ManifestPublication]
	inferenceHints?: #InferenceHints
}
