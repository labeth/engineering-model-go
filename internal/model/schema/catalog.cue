// catalog.cue validates the catalog document, documents.catalog in engmod.yml
// (#CatalogDocument), and holds the types only that document uses.
package model

#CatalogGroups: {
	systems?: [...#CatalogEntry]
	functionalGroups?: [...#CatalogEntry]
	functionalUnits?: [...#CatalogEntry]
	referencedElements?: [...#CatalogEntry]
	actors?: [...#CatalogEntry]
	attackVectors?: [...#CatalogEntry]
	events?: [...#CatalogEntry]
	states?: [...#CatalogEntry]
	features?: [...#CatalogEntry]
	modes?: [...#CatalogEntry]
	conditions?: [...#CatalogEntry]
	dataTerms?: [...#CatalogEntry]
}

#CatalogDocument: {
	schemaVersion: #SchemaVersion
	catalog: #CatalogGroups
}

#CatalogEntry: {
	id?:         string
	name?:       string
	definition?: string
	aliases?:    #StringList
}
