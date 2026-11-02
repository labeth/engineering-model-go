package model

#ReqLinkPin: {
	version:     #ExactVersion
	fingerprint: string & =~"^sha256:[0-9a-f]{64}$"
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
