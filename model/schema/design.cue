package model

#DesignView: {
	title?:     string
	narrative?: string
}

#DesignFunctionalGroup: {
	id?: string
	views?: [string]: #DesignView
}

#DesignFunctionalUnit: {
	id?:    string
	group?: string
	views?: [string]: #DesignView
}

#DesignModel: {
	id?:    string
	title?: string
	functionalGroups?: [...#DesignFunctionalGroup]
	functionalUnits?: [...#DesignFunctionalUnit]
}

#DesignDocument: {
	schemaVersion?: #SchemaVersion
	design: #DesignModel
}
