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
	schemaVersion?: #SchemaVersion
	catalog: #CatalogGroups
}
