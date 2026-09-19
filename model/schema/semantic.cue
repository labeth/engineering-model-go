package model

#Multiplicity: {
	lower?: int & >=0
	upper?: int & >=-1
	if lower != _|_ && upper != _|_ && upper >= 0 {
		upper: >=lower
	}
}

#SemanticQuantity: {
	value: number
	unit:  string & !=""
}

#SemanticValue: {
	kind:       string
	string?:    string
	integer?:   int
	real?:      number
	boolean?:   bool
	reference?: string
	quantity?:  #SemanticQuantity
	if kind == "quantity" {
		quantity: #SemanticQuantity
	}
}

#SemanticExpression: {
	kind?:       string
	language?:   string
	operator?:   string
	value?:      string
	typedValue?: #SemanticValue
	operands?: [...#SemanticExpression]
}

#SemanticFeature: {
	name:          string
	kind?:         "attribute" | "port" | "parameter"
	type?:         string
	direction?:    string
	multiplicity?: #Multiplicity
	ordered?:      bool
	unique?:       bool
	conjugated?:   bool
	specializes?:  #StringList
	subsets?:      #StringList
	redefines?:    #StringList
	value?:        #SemanticExpression
	references?:   #StringList
}

#SemanticMetadata: {
	namespace: string
	type:      string
	target:    string
	properties?: [string]: #MetamodelValue
}

#SemanticElementBase: {
	id:            string & !=""
	name?:         string
	kind:          string & !=""
	metaclass?:    string & !=""
	properties?:   #ExtensionProperties
	namespace?:    string
	owner?:        string
	typeRef?:      string
	multiplicity?: #Multiplicity
	ordered?:      bool
	unique?:       bool
	conjugated?:   bool
	specializes?:  #StringList
	subsets?:      #StringList
	redefines?:    #StringList
	controlKind?:  string
	occurrenceId?: string
	portionOf?:    string
	variation?:    bool
	variants?:     #StringList
	references?:   #StringList
	features?: [...#SemanticFeature]
	metadata?: [...#SemanticMetadata]
	extensionNamespace?: string
	extension?:          string
	targets?:            #StringList
	if kind == "engineering_extension" {
		extensionNamespace: string & !=""
		extension:          string & !=""
		targets: [string, ...string]
	}
}

#SemanticElement: #SemanticElementBase & #OfficialMetamodelBinding

#SemanticRelationshipBase: {
	id:          string & !=""
	kind:        string & !=""
	metaclass?:  string & !=""
	properties?: #ExtensionProperties
	source:      string & !=""
	target:      string & !=""
	owner?:      string
	sourceType?: string
	itemRef?:    string
	triggers?:   #StringList
	guard?:      #SemanticExpression
	effect?:     #SemanticExpression
	metadata?: [...#SemanticMetadata]
}

#SemanticRelationship: #SemanticRelationshipBase & #OfficialMetamodelBinding

#SemanticImport: {
	namespace:   string
	imported:    string
	visibility?: string
	recursive?:  bool
}

#SemanticContent: {
	imports?: [...#SemanticImport]
	elements?: [...#SemanticElement]
	relationships?: [...#SemanticRelationship]
}
