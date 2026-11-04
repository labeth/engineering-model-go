package model

#ReqLinkPin: {
	// The dependency version the link was reviewed at; equal to the
	// dependency pin in engmod.yml once reviewed (decision D23).
	version: #ExactVersion
	// Deprecated (D23): accepted for one release, ignored, and reported as
	// link.fingerprint_deprecated; engws migrate-links removes it.
	fingerprint?: string
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
