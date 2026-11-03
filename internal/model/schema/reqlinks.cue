package model

#ReqLinkPin: {
	version:     #ExactVersion
	// "" until the item could be resolved (engws migrate-links offline);
	// the composed check then reports the link as suspect. The engine writes
	// sha256:v<N>:<hex> (N the canonical-form version); the unversioned
	// sha256:<hex> of the v0.8.0 pre-releases and other versions load and are
	// reported as link.fingerprint_outdated.
	fingerprint: "" | (string & =~"^sha256:(v[0-9]+:)?[0-9a-f]{64}$")
}

#ReqLink: {
	id:          #StableID
	parent?:     #StableID
	child:       #NonEmpty & =~"^[a-z][a-z0-9-]*::[A-Z][A-Z0-9-]*$"
	evidence?:   [...#NonEmpty]
	pinned:      #ReqLinkPin
	status:      "claimed" | "accepted"
	rationale?:  string
	reviewedBy?: string
	reviewedOn?: string & =~"^[0-9]{4}-[0-9]{2}-[0-9]{2}$"
}

#ReqLinksDocument: {
	schemaVersion: #SchemaVersion
	reqLinks: [...#ReqLink]
}
