// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package sysmlmetamodel

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"go/format"
	"io"
	"os"
	"sort"
	"strconv"
	"strings"
)

const InventorySchemaVersion = "1"

type Manifest struct {
	SchemaVersion      string        `json:"schemaVersion"`
	Baseline           string        `json:"baseline"`
	FormalRelease      FormalRelease `json:"formalRelease"`
	MetamodelAuthority string        `json:"metamodelAuthority"`
	Sources            []Source      `json:"sources"`
}

type FormalRelease struct {
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Commit     string `json:"commit"`
	URL        string `json:"url"`
	Authority  string `json:"authority"`
}

type Source struct {
	ID         string `json:"id"`
	Origin     string `json:"origin"`
	Package    string `json:"package"`
	Repository string `json:"repository"`
	Tag        string `json:"tag"`
	Commit     string `json:"commit"`
	Path       string `json:"path"`
	URL        string `json:"url"`
	SHA256     string `json:"sha256"`
}

type Inventory struct {
	SchemaVersion      string            `json:"schemaVersion"`
	Baseline           string            `json:"baseline"`
	Authority          string            `json:"authority"`
	FormalRelease      FormalRelease     `json:"formalRelease"`
	SourceManifestHash string            `json:"sourceManifestSha256"`
	Sources            []InventorySource `json:"sources"`
	Packages           []Package         `json:"packages"`
	Inheritance        []InheritanceEdge `json:"inheritance"`
	Metaclasses        []Metaclass       `json:"metaclasses"`
	Counts             Counts            `json:"counts"`
}

type InventorySource struct {
	ID      string `json:"id"`
	Origin  string `json:"origin"`
	Package string `json:"package"`
	Path    string `json:"path"`
	URL     string `json:"url"`
	SHA256  string `json:"sha256"`
}

type Package struct {
	Name   string `json:"name"`
	Origin string `json:"origin"`
	NSURI  string `json:"nsURI"`
}

type Counts struct {
	Packages            int `json:"packages"`
	Metaclasses         int `json:"metaclasses"`
	AbstractMetaclasses int `json:"abstractMetaclasses"`
	Relationships       int `json:"relationships"`
	OwnedProperties     int `json:"ownedProperties"`
	InheritedProperties int `json:"inheritedProperties"`
	DerivedProperties   int `json:"derivedProperties"`
	InheritanceEdges    int `json:"inheritanceEdges"`
}

type InheritanceEdge struct {
	Sub   string `json:"sub"`
	Super string `json:"super"`
}

type Metaclass struct {
	ID                  string        `json:"id"`
	Name                string        `json:"name"`
	Origin              string        `json:"origin"`
	Package             string        `json:"package"`
	Abstract            bool          `json:"abstract,omitempty"`
	Relationship        bool          `json:"relationship,omitempty"`
	Supertypes          []string      `json:"supertypes,omitempty"`
	OwnedProperties     []Property    `json:"ownedProperties,omitempty"`
	InheritedProperties []string      `json:"inheritedProperties,omitempty"`
	RelationshipEnds    []PropertyRef `json:"relationshipEnds,omitempty"`
}

type Multiplicity struct {
	Lower int `json:"lower"`
	Upper int `json:"upper"`
}

type Property struct {
	ID           string       `json:"id"`
	Name         string       `json:"name"`
	Origin       string       `json:"origin"`
	Kind         string       `json:"kind"`
	Type         string       `json:"type"`
	Multiplicity Multiplicity `json:"multiplicity"`
	Ordered      bool         `json:"ordered"`
	Unique       bool         `json:"unique"`
	Derived      bool         `json:"derived,omitempty"`
	Transient    bool         `json:"transient,omitempty"`
	Volatile     bool         `json:"volatile,omitempty"`
	Containment  bool         `json:"containment,omitempty"`
	Opposite     string       `json:"opposite,omitempty"`
	Subsets      []string     `json:"subsets,omitempty"`
	Redefines    []string     `json:"redefines,omitempty"`
}

type PropertyRef struct {
	DeclaringMetaclass string `json:"declaringMetaclass"`
	Property           string `json:"property"`
	Role               string `json:"role"`
	Type               string `json:"type"`
}

type Coverage struct {
	SchemaVersion string          `json:"schemaVersion"`
	Baseline      string          `json:"baseline"`
	InventoryHash string          `json:"inventorySha256"`
	Entries       []CoverageEntry `json:"entries"`
	Counts        CoverageCounts  `json:"counts"`
}

type CoverageEntry struct {
	ID      string   `json:"id"`
	Kind    string   `json:"kind"`
	Status  string   `json:"status"`
	Aliases []string `json:"aliases,omitempty"`
	Reason  string   `json:"reason,omitempty"`
}

type CoverageCounts struct {
	Mapped        int `json:"mapped"`
	Derived       int `json:"derived"`
	Unimplemented int `json:"unimplemented"`
}

type ePackage struct {
	Name        string        `xml:"name,attr"`
	NSURI       string        `xml:"nsURI,attr"`
	Classifiers []eClassifier `xml:"eClassifiers"`
}

type eClassifier struct {
	Type       string     `xml:"type,attr"`
	Name       string     `xml:"name,attr"`
	Abstract   string     `xml:"abstract,attr"`
	SuperTypes string     `xml:"eSuperTypes,attr"`
	Features   []eFeature `xml:"eStructuralFeatures"`
}

type eFeature struct {
	Type        string        `xml:"type,attr"`
	Name        string        `xml:"name,attr"`
	EType       string        `xml:"eType,attr"`
	Lower       string        `xml:"lowerBound,attr"`
	Upper       string        `xml:"upperBound,attr"`
	Ordered     string        `xml:"ordered,attr"`
	Unique      string        `xml:"unique,attr"`
	Derived     string        `xml:"derived,attr"`
	Transient   string        `xml:"transient,attr"`
	Volatile    string        `xml:"volatile,attr"`
	Containment string        `xml:"containment,attr"`
	Opposite    string        `xml:"eOpposite,attr"`
	Annotations []eAnnotation `xml:"eAnnotations"`
}

type eAnnotation struct {
	Source     string `xml:"source,attr"`
	References string `xml:"references,attr"`
}

type parsedPackage struct {
	source Source
	raw    ePackage
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func LoadManifest(path string) (Manifest, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Manifest{}, nil, err
	}
	var manifest Manifest
	if err := json.Unmarshal(data, &manifest); err != nil {
		return Manifest{}, nil, fmt.Errorf("decode manifest: %w", err)
	}
	if manifest.SchemaVersion != "1" || manifest.Baseline == "" || len(manifest.Sources) == 0 {
		return Manifest{}, nil, errors.New("invalid metamodel source manifest")
	}
	return manifest, data, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func VerifySource(source Source, data []byte) error {
	sum := sha256.Sum256(data)
	actual := hex.EncodeToString(sum[:])
	if actual != source.SHA256 {
		return fmt.Errorf("%s hash mismatch: got %s, want %s", source.ID, actual, source.SHA256)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func Build(manifest Manifest, manifestBytes []byte, sourceData map[string][]byte) (Inventory, error) {
	packages := make([]parsedPackage, 0, len(manifest.Sources))
	classOrigin := map[string]string{}
	enumOrigin := map[string]string{}
	for _, source := range manifest.Sources {
		data, ok := sourceData[source.ID]
		if !ok {
			return Inventory{}, fmt.Errorf("missing source %q", source.ID)
		}
		if err := VerifySource(source, data); err != nil {
			return Inventory{}, err
		}
		var pkg ePackage
		if err := xml.Unmarshal(data, &pkg); err != nil {
			return Inventory{}, fmt.Errorf("decode %s: %w", source.ID, err)
		}
		if pkg.Name != source.Package {
			return Inventory{}, fmt.Errorf("%s package is %q, want %q", source.ID, pkg.Name, source.Package)
		}
		packages = append(packages, parsedPackage{source: source, raw: pkg})
		for _, classifier := range pkg.Classifiers {
			switch classifier.Type {
			case "ecore:EClass":
				if _, exists := classOrigin[classifier.Name]; !exists {
					classOrigin[classifier.Name] = source.Origin
				}
			case "ecore:EEnum":
				if _, exists := enumOrigin[classifier.Name]; !exists {
					enumOrigin[classifier.Name] = source.Origin
				}
			}
		}
	}

	classes := map[string]*Metaclass{}
	for _, pkg := range packages {
		for _, classifier := range pkg.raw.Classifiers {
			if classifier.Type != "ecore:EClass" || classOrigin[classifier.Name] != pkg.source.Origin {
				continue
			}
			id := qualify(pkg.source.Origin, classifier.Name)
			class := &Metaclass{
				ID:       id,
				Name:     classifier.Name,
				Origin:   pkg.source.Origin,
				Package:  pkg.raw.Name,
				Abstract: boolValue(classifier.Abstract, false),
			}
			for _, super := range strings.Fields(classifier.SuperTypes) {
				class.Supertypes = append(class.Supertypes, resolveType(super, classOrigin, enumOrigin, pkg.source.Origin))
			}
			sort.Strings(class.Supertypes)
			for _, feature := range classifier.Features {
				class.OwnedProperties = append(class.OwnedProperties,
					normalizeProperty(id, feature, pkg.source.Origin, classOrigin, enumOrigin))
			}
			sort.Slice(class.OwnedProperties, func(i, j int) bool { return class.OwnedProperties[i].Name < class.OwnedProperties[j].Name })
			classes[id] = class
		}
	}
	// The SysML Ecore is flattened over KerML and can add a SysML property to a
	// KerML metaclass. Merge only additive features; retain KerML as class origin.
	for _, pkg := range packages {
		for _, classifier := range pkg.raw.Classifiers {
			if classifier.Type != "ecore:EClass" || classOrigin[classifier.Name] == pkg.source.Origin {
				continue
			}
			class := classes[qualify(classOrigin[classifier.Name], classifier.Name)]
			owned := map[string]bool{}
			for _, property := range class.OwnedProperties {
				owned[property.Name] = true
			}
			for _, feature := range classifier.Features {
				if !owned[feature.Name] {
					class.OwnedProperties = append(class.OwnedProperties,
						normalizeProperty(class.ID, feature, pkg.source.Origin, classOrigin, enumOrigin))
				}
			}
			sort.Slice(class.OwnedProperties, func(i, j int) bool { return class.OwnedProperties[i].Name < class.OwnedProperties[j].Name })
		}
	}

	for _, class := range classes {
		class.Relationship = isSubclass(class.ID, "KerML::Relationship", classes, map[string]bool{})
		inherited, err := inheritedProperties(class, classes, map[string]bool{})
		if err != nil {
			return Inventory{}, err
		}
		class.InheritedProperties = inherited
		if class.Relationship {
			class.RelationshipEnds = relationshipEnds(class, classes)
		}
	}

	inventory := Inventory{
		SchemaVersion: InventorySchemaVersion,
		Baseline:      manifest.Baseline,
		Authority:     manifest.MetamodelAuthority,
		FormalRelease: manifest.FormalRelease,
	}
	manifestSum := sha256.Sum256(manifestBytes)
	inventory.SourceManifestHash = hex.EncodeToString(manifestSum[:])
	for _, pkg := range packages {
		inventory.Sources = append(inventory.Sources, InventorySource{
			ID: pkg.source.ID, Origin: pkg.source.Origin, Package: pkg.source.Package,
			Path: pkg.source.Path, URL: pkg.source.URL, SHA256: pkg.source.SHA256,
		})
		inventory.Packages = append(inventory.Packages, Package{Name: pkg.raw.Name, Origin: pkg.source.Origin, NSURI: pkg.raw.NSURI})
	}
	sort.Slice(inventory.Sources, func(i, j int) bool { return inventory.Sources[i].ID < inventory.Sources[j].ID })
	sort.Slice(inventory.Packages, func(i, j int) bool { return inventory.Packages[i].Origin < inventory.Packages[j].Origin })
	for _, class := range classes {
		inventory.Metaclasses = append(inventory.Metaclasses, *class)
		for _, super := range class.Supertypes {
			inventory.Inheritance = append(inventory.Inheritance, InheritanceEdge{Sub: class.ID, Super: super})
		}
	}
	sort.Slice(inventory.Metaclasses, func(i, j int) bool { return inventory.Metaclasses[i].ID < inventory.Metaclasses[j].ID })
	sort.Slice(inventory.Inheritance, func(i, j int) bool {
		if inventory.Inheritance[i].Sub == inventory.Inheritance[j].Sub {
			return inventory.Inheritance[i].Super < inventory.Inheritance[j].Super
		}
		return inventory.Inheritance[i].Sub < inventory.Inheritance[j].Sub
	})
	inventory.Counts = countInventory(inventory)
	return inventory, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func Marshal(value any) ([]byte, error) {
	data, err := json.MarshalIndent(value, "", "  ")
	if err != nil {
		return nil, err
	}
	return append(data, '\n'), nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func LoadInventory(path string) (Inventory, []byte, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Inventory{}, nil, err
	}
	var inventory Inventory
	if err := json.Unmarshal(data, &inventory); err != nil {
		return Inventory{}, nil, err
	}
	if inventory.SchemaVersion != InventorySchemaVersion {
		return Inventory{}, nil, fmt.Errorf("unsupported inventory schema %q", inventory.SchemaVersion)
	}
	return inventory, data, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func LoadCoverage(path string) (Coverage, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return Coverage{}, err
	}
	var coverage Coverage
	if err := json.Unmarshal(data, &coverage); err != nil {
		return Coverage{}, err
	}
	return coverage, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func BootstrapCoverage(inventory Inventory, inventoryBytes []byte) Coverage {
	mapped := semanticAliases()
	coverage := Coverage{SchemaVersion: "1", Baseline: inventory.Baseline, InventoryHash: hashBytes(inventoryBytes)}
	for _, class := range inventory.Metaclasses {
		entry := CoverageEntry{ID: class.ID, Kind: "metaclass", Status: "mapped", Aliases: []string{"official:" + class.ID}}
		entry.Aliases = append(entry.Aliases, mapped[class.ID]...)
		sort.Strings(entry.Aliases)
		coverage.Entries = append(coverage.Entries, entry)
		for _, property := range class.OwnedProperties {
			status := "mapped"
			reason := ""
			aliases := []string{"official:" + property.ID}
			if property.Derived {
				status = "derived"
				reason = "official metamodel marks this property derived"
				aliases = nil
			}
			coverage.Entries = append(coverage.Entries, CoverageEntry{
				ID: property.ID, Kind: "property", Status: status, Reason: reason, Aliases: aliases,
			})
		}
	}
	sort.Slice(coverage.Entries, func(i, j int) bool { return coverage.Entries[i].ID < coverage.Entries[j].ID })
	coverage.Counts = countCoverage(coverage.Entries)
	return coverage
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func CheckCoverage(inventory Inventory, inventoryBytes []byte, coverage Coverage) error {
	if coverage.SchemaVersion != "1" {
		return fmt.Errorf("unsupported coverage schema %q", coverage.SchemaVersion)
	}
	if coverage.Baseline != inventory.Baseline {
		return fmt.Errorf("coverage baseline %q does not match inventory %q", coverage.Baseline, inventory.Baseline)
	}
	if coverage.InventoryHash != hashBytes(inventoryBytes) {
		return errors.New("coverage inventory hash is stale")
	}
	expected := map[string]string{}
	for _, class := range inventory.Metaclasses {
		expected[class.ID] = "metaclass"
		for _, property := range class.OwnedProperties {
			expected[property.ID] = "property"
		}
	}
	seen := map[string]bool{}
	aliases := map[string]string{}
	for _, entry := range coverage.Entries {
		kind, ok := expected[entry.ID]
		if !ok {
			return fmt.Errorf("coverage contains unknown inventory entry %q", entry.ID)
		}
		if seen[entry.ID] {
			return fmt.Errorf("coverage contains duplicate entry %q", entry.ID)
		}
		seen[entry.ID] = true
		if entry.Kind != kind {
			return fmt.Errorf("coverage entry %q has kind %q, want %q", entry.ID, entry.Kind, kind)
		}
		switch entry.Status {
		case "mapped":
			if len(entry.Aliases) == 0 {
				return fmt.Errorf("mapped entry %q has no aliases", entry.ID)
			}
		case "derived":
			if kind != "property" {
				return fmt.Errorf("metaclass %q cannot be excluded as derived", entry.ID)
			}
		case "unimplemented":
			return fmt.Errorf("normative coverage entry %q is unimplemented", entry.ID)
		default:
			return fmt.Errorf("coverage entry %q has invalid status %q", entry.ID, entry.Status)
		}
		for _, alias := range entry.Aliases {
			if prior, exists := aliases[alias]; exists {
				return fmt.Errorf("semantic alias %q is mapped by both %s and %s", alias, prior, entry.ID)
			}
			aliases[alias] = entry.ID
		}
	}
	for id := range expected {
		if !seen[id] {
			return fmt.Errorf("official inventory entry %q is unaccounted", id)
		}
	}
	for official, expectedAliases := range semanticAliases() {
		if _, present := expected[official]; !present {
			continue
		}
		for _, alias := range expectedAliases {
			if aliases[alias] != official {
				return fmt.Errorf("semantic alias %q must map to %s", alias, official)
			}
		}
	}
	if got := countCoverage(coverage.Entries); got != coverage.Counts {
		return fmt.Errorf("coverage counts are stale: got %+v, want %+v", coverage.Counts, got)
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func GenerateRegistry(inventory Inventory, coverage Coverage) ([]byte, error) {
	if err := CheckCoverageWithoutHash(inventory, coverage); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString("// Code generated by cmd/sysmlmetamodel; DO NOT EDIT.\n")
	out.WriteString("package model\n\n")
	out.WriteString("type OfficialSysMLProperty struct { ID string; Kind string; Type string; Lower int; Upper int; Ordered bool; Unique bool; Derived bool; ReadOnly bool; Containment bool; Opposite string; Subsets []string; Redefines []string }\n\n")
	out.WriteString("type OfficialSysMLRelationshipEnd struct { Property string; Role string; Type string }\n\n")
	out.WriteString("type OfficialSysMLMetaclass struct { Origin string; Abstract bool; Relationship bool; Supertypes []string; Properties map[string]OfficialSysMLProperty; RelationshipEnds []OfficialSysMLRelationshipEnd }\n\n")
	propertyByID := map[string]Property{}
	for _, class := range inventory.Metaclasses {
		for _, property := range class.OwnedProperties {
			propertyByID[property.ID] = property
		}
	}
	out.WriteString("var OfficialSysMLMetaclasses = map[string]OfficialSysMLMetaclass{\n")
	for _, class := range inventory.Metaclasses {
		fmt.Fprintf(&out, "\t%q: {Origin: %q, Abstract: %t, Relationship: %t", class.ID, class.Origin, class.Abstract, class.Relationship)
		if len(class.Supertypes) > 0 {
			fmt.Fprintf(&out, ", Supertypes: %#v", class.Supertypes)
		}
		out.WriteString(", Properties: map[string]OfficialSysMLProperty{\n")
		for _, property := range effectiveProperties(class, propertyByID) {
			fmt.Fprintf(&out, "\t\t%q: {ID: %q, Kind: %q, Type: %q, Lower: %d, Upper: %d, Ordered: %t, Unique: %t, Derived: %t, ReadOnly: %t, Containment: %t, Opposite: %q, Subsets: %#v, Redefines: %#v},\n",
				property.Name, property.ID, property.Kind, property.Type, property.Multiplicity.Lower, property.Multiplicity.Upper,
				property.Ordered, property.Unique, property.Derived, property.Derived || property.Transient || property.Volatile,
				property.Containment, property.Opposite, property.Subsets, property.Redefines)
		}
		out.WriteString("\t}")
		if len(class.RelationshipEnds) > 0 {
			out.WriteString(", RelationshipEnds: []OfficialSysMLRelationshipEnd{\n")
			for _, end := range class.RelationshipEnds {
				fmt.Fprintf(&out, "\t\t{Property: %q, Role: %q, Type: %q},\n", end.Property, end.Role, end.Type)
			}
			out.WriteString("\t}")
		}
		out.WriteString("},\n")
	}
	out.WriteString("}\n\n")
	out.WriteString("var OfficialSysMLSemanticAliases = map[string]string{\n")
	for _, entry := range coverage.Entries {
		if entry.Status != "mapped" {
			continue
		}
		for _, alias := range entry.Aliases {
			fmt.Fprintf(&out, "\t%q: %q,\n", alias, entry.ID)
		}
	}
	out.WriteString("}\n")
	out.WriteString("\nvar SysMLSemanticAliasExceptions = map[string]string{\n")
	exceptions := semanticAliasExceptions()
	exceptionAliases := make([]string, 0, len(exceptions))
	for alias := range exceptions {
		exceptionAliases = append(exceptionAliases, alias)
	}
	sort.Strings(exceptionAliases)
	for _, alias := range exceptionAliases {
		fmt.Fprintf(&out, "\t%q: %q,\n", alias, exceptions[alias])
	}
	out.WriteString("}\n")
	return format.Source(out.Bytes())
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func GenerateCUE(inventory Inventory, coverage Coverage) ([]byte, error) {
	if err := CheckCoverageWithoutHash(inventory, coverage); err != nil {
		return nil, err
	}
	propertyByID := map[string]Property{}
	for _, class := range inventory.Metaclasses {
		for _, property := range class.OwnedProperties {
			propertyByID[property.ID] = property
		}
	}
	var out bytes.Buffer
	out.WriteString("// Code generated by cmd/sysmlmetamodel; DO NOT EDIT.\n")
	out.WriteString("package model\n\n")
	out.WriteString(`#MetamodelString: string
#MetamodelInteger: int
#MetamodelReal: number
#MetamodelBoolean: bool
#MetamodelStringValue: close({kind: "string", string: #MetamodelString})
#MetamodelIntegerValue: close({kind: "integer", integer: #MetamodelInteger})
#MetamodelRealValue: close({kind: "real", real: #MetamodelReal})
#MetamodelBooleanValue: close({kind: "boolean", boolean: #MetamodelBoolean})
#MetamodelReferenceValue: close({kind: "reference", reference: #MetamodelString & !=""})
#MetamodelNullValue: close({kind: "null"})
#MetamodelObjectValue: close({kind: "object", object: [string]: #MetamodelValue})
#MetamodelListValue: close({kind: "list", list: [...#MetamodelValue]})
#MetamodelValue: #MetamodelStringValue | #MetamodelIntegerValue | #MetamodelRealValue | #MetamodelBooleanValue | #MetamodelReferenceValue | #MetamodelNullValue | #MetamodelObjectValue | #MetamodelListValue
#ExtensionPropertyValue: close({values: [...#MetamodelValue], derived?: bool, implied?: bool})
#ExtensionProperties: [string]: #ExtensionPropertyValue

`)
	out.WriteString("#OfficialMetaclassName: ")
	for index, class := range inventory.Metaclasses {
		if index > 0 {
			out.WriteString(" | ")
		}
		fmt.Fprintf(&out, "%q", class.ID)
	}
	out.WriteString("\n\n#OfficialMetamodelBinding: {\n")
	out.WriteString("\tmetaclass?: #OfficialMetaclassName | (#MetamodelString & =~\"^Engineering::\")\n")
	for _, class := range inventory.Metaclasses {
		fmt.Fprintf(&out, "\tif metaclass == %q {\n\t\tproperties?: close({\n", class.ID)
		for _, property := range effectiveProperties(class, propertyByID) {
			fmt.Fprintf(&out, "\t\t\t%q?: %s\n", property.Name, cuePropertyConstraint(property))
		}
		out.WriteString("\t\t})\n\t}\n")
	}
	out.WriteString("\tif metaclass =~ \"^Engineering::\" {\n\t\tproperties?: #ExtensionProperties\n\t}\n\t...\n}\n")
	return out.Bytes(), nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func effectiveProperties(class Metaclass, propertyByID map[string]Property) []Property {
	byName := map[string]Property{}
	for _, id := range class.InheritedProperties {
		if property, ok := propertyByID[id]; ok {
			byName[property.Name] = property
		}
	}
	for _, property := range class.OwnedProperties {
		byName[property.Name] = property
	}
	names := make([]string, 0, len(byName))
	for name := range byName {
		names = append(names, name)
	}
	sort.Strings(names)
	out := make([]Property, 0, len(names))
	for _, name := range names {
		out = append(out, byName[name])
	}
	return out
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func cuePropertyConstraint(property Property) string {
	shape := "#MetamodelStringValue"
	if property.Kind == "reference" {
		shape = "#MetamodelReferenceValue"
	} else {
		switch property.Type {
		case "Ecore::Boolean", "SysML::EBoolean":
			shape = "#MetamodelBooleanValue"
		case "Ecore::Integer":
			shape = "#MetamodelIntegerValue"
		case "Ecore::Real":
			shape = "#MetamodelRealValue"
		}
	}
	derived := "derived?: false"
	if property.Derived {
		derived = "derived: true"
	}
	var alternatives []string
	if property.Multiplicity.Upper < 0 {
		items := make([]string, property.Multiplicity.Lower)
		for index := range items {
			items[index] = shape
		}
		if len(items) == 0 {
			alternatives = []string{"[..." + shape + "]"}
		} else {
			alternatives = []string{"[" + strings.Join(items, ", ") + ", ..." + shape + "]"}
		}
	} else {
		for count := property.Multiplicity.Lower; count <= property.Multiplicity.Upper; count++ {
			items := make([]string, count)
			for index := range items {
				items[index] = shape
			}
			alternatives = append(alternatives, "["+strings.Join(items, ", ")+"]")
		}
	}
	return fmt.Sprintf("close({values: (%s)\n%s\nimplied?: bool\n})", strings.Join(alternatives, " | "), derived)
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func CheckCoverageWithoutHash(inventory Inventory, coverage Coverage) error {
	copyCoverage := coverage
	copyCoverage.InventoryHash = ""
	data, err := Marshal(inventory)
	if err != nil {
		return err
	}
	copyCoverage.InventoryHash = hashBytes(data)
	return CheckCoverage(inventory, data, copyCoverage)
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func RebindCoverage(inventory Inventory, inventoryBytes []byte, coverage Coverage) (Coverage, error) {
	if err := CheckCoverageWithoutHash(inventory, coverage); err != nil {
		return Coverage{}, err
	}
	coverage.Baseline = inventory.Baseline
	coverage.InventoryHash = hashBytes(inventoryBytes)
	coverage.Counts = countCoverage(coverage.Entries)
	return coverage, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func DecodeInventory(r io.Reader) (Inventory, error) {
	var inventory Inventory
	err := json.NewDecoder(r).Decode(&inventory)
	return inventory, err
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func inheritedProperties(class *Metaclass, classes map[string]*Metaclass, visiting map[string]bool) ([]string, error) {
	if visiting[class.ID] {
		return nil, fmt.Errorf("inheritance cycle at %s", class.ID)
	}
	visiting[class.ID] = true
	defer delete(visiting, class.ID)
	own := map[string]bool{}
	for _, property := range class.OwnedProperties {
		own[property.Name] = true
	}
	seen := map[string]bool{}
	var result []string
	var walk func(string) error
	walk = func(id string) error {
		super, ok := classes[id]
		if !ok {
			return fmt.Errorf("%s inherits unknown metaclass %s", class.ID, id)
		}
		for _, property := range super.OwnedProperties {
			if !own[property.Name] && !seen[property.ID] {
				result = append(result, property.ID)
				seen[property.ID] = true
			}
		}
		for _, next := range super.Supertypes {
			if err := walk(next); err != nil {
				return err
			}
		}
		return nil
	}
	for _, super := range class.Supertypes {
		if err := walk(super); err != nil {
			return nil, err
		}
	}
	sort.Strings(result)
	return result, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func relationshipEnds(class *Metaclass, classes map[string]*Metaclass) []PropertyRef {
	var result []PropertyRef
	add := func(declaring string, property Property) {
		if property.Kind != "reference" || !isRelationshipRole(property) {
			return
		}
		result = append(result, PropertyRef{
			DeclaringMetaclass: declaring, Property: property.ID,
			Role: property.Name, Type: property.Type,
		})
	}
	for _, property := range class.OwnedProperties {
		add(class.ID, property)
	}
	for _, inheritedID := range class.InheritedProperties {
		declaringID := inheritedID[:strings.LastIndex(inheritedID, "::")]
		propertyName := inheritedID[strings.LastIndex(inheritedID, "::")+2:]
		if declaring := classes[declaringID]; declaring != nil {
			for _, property := range declaring.OwnedProperties {
				if property.Name == propertyName {
					add(declaringID, property)
					break
				}
			}
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Property < result[j].Property })
	return result
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func isRelationshipRole(property Property) bool {
	switch property.Name {
	case "source", "target", "relatedElement", "owningRelatedElement", "ownedRelatedElement", "relatedFeature", "connectorEnd":
		return true
	}
	for _, ref := range append(append([]string{}, property.Subsets...), property.Redefines...) {
		for _, suffix := range []string{"::source", "::target", "::relatedElement", "::relatedFeature", "::connectorEnd"} {
			if strings.HasSuffix(ref, suffix) {
				return true
			}
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func isSubclass(id, target string, classes map[string]*Metaclass, seen map[string]bool) bool {
	if id == target {
		return true
	}
	if seen[id] {
		return false
	}
	seen[id] = true
	class := classes[id]
	if class == nil {
		return false
	}
	for _, super := range class.Supertypes {
		if isSubclass(super, target, classes, seen) {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func countInventory(inventory Inventory) Counts {
	counts := Counts{Packages: len(inventory.Packages), Metaclasses: len(inventory.Metaclasses), InheritanceEdges: len(inventory.Inheritance)}
	for _, class := range inventory.Metaclasses {
		if class.Abstract {
			counts.AbstractMetaclasses++
		}
		if class.Relationship {
			counts.Relationships++
		}
		counts.OwnedProperties += len(class.OwnedProperties)
		counts.InheritedProperties += len(class.InheritedProperties)
		for _, property := range class.OwnedProperties {
			if property.Derived {
				counts.DerivedProperties++
			}
		}
	}
	return counts
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func countCoverage(entries []CoverageEntry) CoverageCounts {
	var counts CoverageCounts
	for _, entry := range entries {
		switch entry.Status {
		case "mapped":
			counts.Mapped++
		case "derived":
			counts.Derived++
		case "unimplemented":
			counts.Unimplemented++
		}
	}
	return counts
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func semanticAliases() map[string][]string {
	return map[string][]string{
		"KerML::Dependency":                 {"dependency"},
		"KerML::FeatureTyping":              {"typing"},
		"KerML::OwningMembership":           {"containment"},
		"KerML::Package":                    {"package_definition"},
		"KerML::Redefinition":               {"redefinition"},
		"KerML::ReferenceSubsetting":        {"reference"},
		"KerML::Specialization":             {"specialization"},
		"KerML::Subsetting":                 {"subsetting"},
		"KerML::Succession":                 {"succession"},
		"SysML::ActionDefinition":           {"action_definition"},
		"SysML::ActionUsage":                {"action_usage"},
		"SysML::AllocationUsage":            {"allocation"},
		"SysML::AnalysisCaseDefinition":     {"analysis_case_definition"},
		"SysML::AnalysisCaseUsage":          {"analysis_case_usage"},
		"SysML::AttributeDefinition":        {"attribute_definition"},
		"SysML::AttributeUsage":             {"attribute_usage"},
		"SysML::BindingConnectorAsUsage":    {"binding"},
		"SysML::CalculationDefinition":      {"calculation_definition"},
		"SysML::CalculationUsage":           {"calculation_usage"},
		"SysML::CaseDefinition":             {"case_definition"},
		"SysML::CaseUsage":                  {"case_usage"},
		"SysML::ConcernDefinition":          {"concern_definition"},
		"SysML::ConcernUsage":               {"concern_usage"},
		"SysML::ConnectionDefinition":       {"connection_definition"},
		"SysML::ConnectionUsage":            {"connection", "connection_usage"},
		"SysML::ConstraintDefinition":       {"constraint_definition"},
		"SysML::ConstraintUsage":            {"constraint_usage"},
		"SysML::ControlNode":                {"control_node"},
		"SysML::EventOccurrenceUsage":       {"event_definition", "event_usage"},
		"SysML::FlowUsage":                  {"transfer"},
		"SysML::InterfaceDefinition":        {"interface_definition"},
		"SysML::InterfaceUsage":             {"interface_usage"},
		"SysML::ItemDefinition":             {"item_definition"},
		"SysML::ItemUsage":                  {"item_usage"},
		"SysML::OccurrenceDefinition":       {"individual_definition", "occurrence_definition"},
		"SysML::OccurrenceUsage":            {"individual_usage", "occurrence_usage", "snapshot", "time_slice"},
		"SysML::PartDefinition":             {"part_definition"},
		"SysML::PartUsage":                  {"part_usage"},
		"SysML::PortDefinition":             {"port_definition"},
		"SysML::PortUsage":                  {"port_usage"},
		"SysML::RequirementDefinition":      {"requirement_definition"},
		"SysML::RequirementUsage":           {"requirement_usage", "satisfaction"},
		"SysML::StateDefinition":            {"state_definition"},
		"SysML::StateUsage":                 {"state_usage"},
		"SysML::TransitionUsage":            {"effect", "event_trigger", "guard", "transition"},
		"SysML::UseCaseDefinition":          {"use_case_definition"},
		"SysML::UseCaseUsage":               {"use_case_usage"},
		"SysML::VariantMembership":          {"variant_membership"},
		"SysML::VerificationCaseDefinition": {"verification_case_definition"},
		"SysML::VerificationCaseUsage":      {"verification", "verification_case_usage"},
		"SysML::ViewDefinition":             {"view_definition"},
		"SysML::ViewUsage":                  {"view_usage"},
		"SysML::ViewpointDefinition":        {"viewpoint_definition"},
		"SysML::ViewpointUsage":             {"viewpoint_usage"},
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func semanticAliasExceptions() map[string]string {
	return map[string]string{
		"engineering_extension":  "typed Engineering Model extension outside the official metamodel",
		"quantity_definition":    "standard-library semantic concept, not an Ecore metaclass",
		"stakeholder_definition": "standard-library semantic concept, not an Ecore metaclass",
		"stakeholder_usage":      "standard-library semantic concept, not an Ecore metaclass",
		"unit_definition":        "standard-library semantic concept, not an Ecore metaclass",
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func normalizeProperty(classID string, feature eFeature, origin string, classes, enums map[string]string) Property {
	property := Property{
		ID:           classID + "::" + feature.Name,
		Name:         feature.Name,
		Origin:       origin,
		Kind:         featureKind(feature.Type),
		Type:         resolveType(feature.EType, classes, enums, origin),
		Multiplicity: Multiplicity{Lower: intValue(feature.Lower, 0), Upper: intValue(feature.Upper, 1)},
		Ordered:      boolValue(feature.Ordered, true),
		Unique:       boolValue(feature.Unique, true),
		Derived:      boolValue(feature.Derived, false),
		Transient:    boolValue(feature.Transient, false),
		Volatile:     boolValue(feature.Volatile, false),
		Containment:  boolValue(feature.Containment, false),
		Opposite:     resolvePropertyRef(feature.Opposite, classes, origin),
	}
	for _, annotation := range feature.Annotations {
		switch annotation.Source {
		case "subsets":
			property.Subsets = append(property.Subsets, resolvePropertyRefs(annotation.References, classes, origin)...)
		case "redefines":
			property.Redefines = append(property.Redefines, resolvePropertyRefs(annotation.References, classes, origin)...)
		}
	}
	sort.Strings(property.Subsets)
	sort.Strings(property.Redefines)
	return property
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func resolveType(raw string, classes, enums map[string]string, fallback string) string {
	name := fragmentName(raw)
	if name == "" {
		return raw
	}
	if origin, ok := classes[name]; ok {
		return qualify(origin, name)
	}
	if origin, ok := enums[name]; ok {
		return qualify(origin, name)
	}
	if strings.Contains(raw, "Types.ecore") {
		return "Ecore::" + name
	}
	return qualify(fallback, name)
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func resolvePropertyRefs(raw string, classes map[string]string, fallback string) []string {
	var result []string
	for _, ref := range strings.Fields(raw) {
		if resolved := resolvePropertyRef(ref, classes, fallback); resolved != "" {
			result = append(result, resolved)
		}
	}
	return result
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func resolvePropertyRef(raw string, classes map[string]string, fallback string) string {
	if raw == "" {
		return ""
	}
	fragment := raw
	if index := strings.LastIndex(fragment, "#//"); index >= 0 {
		fragment = fragment[index+3:]
	} else {
		fragment = strings.TrimPrefix(fragment, "#//")
	}
	parts := strings.Split(fragment, "/")
	if len(parts) != 2 {
		return raw
	}
	origin := fallback
	if known, ok := classes[parts[0]]; ok {
		origin = known
	}
	return qualify(origin, parts[0]) + "::" + parts[1]
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func fragmentName(raw string) string {
	if index := strings.LastIndex(raw, "#//"); index >= 0 {
		return raw[index+3:]
	}
	return strings.TrimPrefix(raw, "#//")
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func qualify(origin, name string) string {
	return origin + "::" + name
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func featureKind(raw string) string {
	if raw == "ecore:EReference" {
		return "reference"
	}
	return "attribute"
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func boolValue(raw string, defaultValue bool) bool {
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.ParseBool(raw)
	return err == nil && value
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func intValue(raw string, defaultValue int) int {
	if raw == "" {
		return defaultValue
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return defaultValue
	}
	return value
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func hashBytes(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}
