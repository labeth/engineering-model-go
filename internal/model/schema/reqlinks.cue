package model

#ReqLinkPin: {
	// The dependency version the link was reviewed at; equal to the
	// dependency pin in engmod.yml once reviewed (decision D23).
	version: #ExactVersion
	// Removed (ADR-EMG-075): decoded only so validation reports the error
	// model.pinned_fingerprint_removed with its remedy.
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
