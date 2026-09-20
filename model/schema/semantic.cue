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
	unit:  #MetamodelString & !=""
}

#SemanticValue: {
	kind:       #MetamodelString
	string?:    #MetamodelString
	integer?:   #MetamodelInteger
	real?:      #MetamodelReal
	boolean?:   #MetamodelBoolean
	reference?: #MetamodelString
	quantity?:  #SemanticQuantity
	if kind == "quantity" {
		quantity: #SemanticQuantity
	}
}

#SemanticExpression: {
	kind?:       #MetamodelString
	language?:   #MetamodelString
	operator?:   #MetamodelString
	value?:      #MetamodelString
	typedValue?: #SemanticValue
	operands?: [...#SemanticExpression]
}

#SemanticFeature: {
	name:          #MetamodelString
	kind?:         "attribute" | "port" | "parameter"
	type?:         #MetamodelString
	direction?:    #MetamodelString
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
	namespace: #MetamodelString
	type:      #MetamodelString
	target:    #MetamodelString
	properties?: [string]: #MetamodelValue
}

#SemanticElementBase: {
	id:            #MetamodelString & !=""
	name?:         #MetamodelString
	kind:          #MetamodelString & !=""
	metaclass?:    #MetamodelString & !=""
	properties?:   #ExtensionProperties
	namespace?:    #MetamodelString
	owner?:        #MetamodelString
	typeRef?:      #MetamodelString
	multiplicity?: #Multiplicity
	ordered?:      bool
	unique?:       bool
	conjugated?:   bool
	specializes?:  #StringList
	subsets?:      #StringList
	redefines?:    #StringList
	controlKind?:  #MetamodelString
	occurrenceId?: #MetamodelString
	portionOf?:    #MetamodelString
	variation?:    bool
	variants?:     #StringList
	references?:   #StringList
	features?: [...#SemanticFeature]
	metadata?: [...#SemanticMetadata]
	extensionNamespace?: #MetamodelString
	extension?:          #MetamodelString
	targets?:            #StringList
	if kind == "engineering_extension" {
		extensionNamespace: #MetamodelString & !=""
		extension:          #MetamodelString & !=""
		targets: [#MetamodelString, ...#MetamodelString]
	}
}

#SemanticElement: #SemanticElementBase & #OfficialMetamodelBinding

#SemanticRelationshipBase: {
	id:          #MetamodelString & !=""
	kind:        #MetamodelString & !=""
	metaclass?:  #MetamodelString & !=""
	properties?: #ExtensionProperties
	source:      #MetamodelString & !=""
	target:      #MetamodelString & !=""
	owner?:      #MetamodelString
	sourceType?: #MetamodelString
	itemRef?:    #MetamodelString
	triggers?:   #StringList
	guard?:      #SemanticExpression
	effect?:     #SemanticExpression
	metadata?: [...#SemanticMetadata]
}

#SemanticRelationship: #SemanticRelationshipBase & #OfficialMetamodelBinding

#SemanticImport: {
	namespace:   #MetamodelString
	imported:    #MetamodelString
	visibility?: #MetamodelString
	recursive?:  bool
}

#SemanticContent: {
	imports?: [...#SemanticImport]
	elements?: [...#SemanticElement]
	relationships?: [...#SemanticRelationship]
}
