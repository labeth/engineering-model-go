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
// on it (internal/validate.TierPolicy). The schema leaves lifecycle, bundles
// and tierDecision optional so that tools can load and report an incomplete
// model; internal/validate reports each absent one as an error with its
// remedy (lifecycle.missing, lifecycle.bundles_missing,
// lifecycle.tier_decision_missing).
#ManifestLifecycle: {
	tier:          #Tier
	bundles?:      [...#Bundle]    // required by validation; foundation always applies
	tierDecision?: #StableID       // required by validation: an accepted DEC-TIER-* whose Tier:/Bundles: lines agree
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

// #HealthBaseline names the committed baseline of accepted existing findings
// (ADR-EMG-059): a path relative to the manifest directory and the decision
// that justifies it. The path rules and the acceptance of the decision are
// reported by internal/validate.
#HealthBaseline: {
	file:     #NonEmpty
	decision: #NonEmpty
}

// #ManifestHealth configures the optional code health report (ADR-EMG-051):
// without it nothing health related is computed or reported.
#ManifestHealth: {
	exclude?:    [...string]
	thresholds?: [string]: #HealthThreshold
	overrides?:  [...#HealthOverride]
	waivers?:    [...#HealthWaiver]
	baseline?:   #HealthBaseline
	// policyDecision is the accepted decision that records a weakening of
	// this block against the base revision (ADR-EMG-051, amended by the review).
	policyDecision?: #NonEmpty
}

// #CodeExemption exempts one model element from the element to code coverage
// checks (ADR-EMG-069): the reason is at least 20 characters and the decision
// must be accepted (reported by internal/trace as code.exemption_unjustified).
#CodeExemption: {
	id:       #NonEmpty
	reason:   #NonEmpty & =~"(?s)^.{20,}$"
	decision: #NonEmpty
}

// #CodeCoverage configures the element to code coverage checks.
#CodeCoverage: {
	exempt?: [...#CodeExemption]
}

#ManifestDocument: {
	schemaVersion: #SchemaVersion
	module:        #ModuleIdentity
	documents:     #DocumentReferences
	dependencies?: [...#ManifestDependency]
	publications?: [...#ManifestPublication]
	inferenceHints?: #InferenceHints
	requirementLinks?: "strict" | "guided"
	codeCoverage?: #CodeCoverage
	lifecycle?: #ManifestLifecycle
	health?: #ManifestHealth
}
