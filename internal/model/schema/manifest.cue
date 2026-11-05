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
	reqLinks?:    #NonEmpty
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

// #Tier is the life-cycle tier of a module, ascending rank 1..5 (ADR-EMG-031).
#Tier: "poc" | "prototype" | "mvp" | "product" | "operational"

// #Bundle is an assurance track; foundation always applies.
#Bundle: "foundation" | "safety" | "security"

// #ManifestLifecycle declares the project tier: the bar for the default
// branch and every release. Validation keys the severity of the tier rules
// on it (internal/validate.TierPolicy); without it every severity is that
// of engineering-model-go v0.9.0.
#ManifestLifecycle: {
	tier:          #Tier
	bundles?:      [...#Bundle]    // default [foundation]
	tierDecision?: #StableID       // an accepted DEC-TIER-* whose Tier:/Bundles: lines agree
	raise?:        [...#NonEmpty]  // codes raised to error regardless of tier; never lowers
}

#ManifestDocument: {
	schemaVersion: #SchemaVersion
	module:        #ModuleIdentity
	documents:     #DocumentReferences
	dependencies?: [...#ManifestDependency]
	publications?: [...#ManifestPublication]
	inferenceHints?: #InferenceHints
	requirementLinks?: "strict" | "guided"
	lifecycle?: #ManifestLifecycle
}
