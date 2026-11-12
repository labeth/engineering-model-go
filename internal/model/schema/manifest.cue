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

// #HealthThreshold is the warn and high limit of one code health smell.
// Value rules (warn >= 1, high >= warn, known codes) are reported by
// internal/validate with a path, not by the schema.
#HealthThreshold: {
	warn: int
	high: int
}

// #HealthOverride replaces thresholds for the paths it names (the exclude
// pattern dialect of inferenceHints.exclude).
#HealthOverride: {
	path:        #NonEmpty
	thresholds?: [string]: #HealthThreshold
}

// #HealthWaiver waives one finding (code, path, function) on the strength
// of an accepted decision.
#HealthWaiver: {
	code:     #NonEmpty
	path:     #NonEmpty
	function: #NonEmpty
	decision: #NonEmpty
}

// #ManifestHealth configures the optional code health report (ADR-EMG-051):
// without it nothing health related is computed or reported.
#ManifestHealth: {
	exclude?:    [...string]
	thresholds?: [string]: #HealthThreshold
	overrides?:  [...#HealthOverride]
	waivers?:    [...#HealthWaiver]
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
	health?: #ManifestHealth
}
