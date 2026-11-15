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
	"maps"
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

// loadedPackages is the decoded Ecore packages of a manifest and the origin
// that first declares each class and enum name.
type loadedPackages struct {
	packages    []parsedPackage
	classOrigin originMap
	enumOrigin  originMap
}

// originMap maps a classifier name to the origin that declares it first.
type originMap map[string]string

// addFirst records origin for name unless a first declaration is known.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (m originMap) addFirst(name, origin string) {
	if _, exists := m[name]; !exists {
		m[name] = origin
	}
}

// resolver resolves Ecore references found in the package of one origin.
type resolver struct {
	classes originMap
	enums   originMap
	origin  string
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func Build(manifest Manifest, manifestBytes []byte, sourceData map[string][]byte) (Inventory, error) {
	loaded, err := loadPackages(manifest, sourceData)
	if err != nil {
		return Inventory{}, err
	}
	classes := loaded.ownClasses()
	loaded.mergeForeignFeatures(classes)
	if err := resolveInheritance(classes); err != nil {
		return Inventory{}, err
	}
	inventory := newInventory(manifest, manifestBytes)
	inventory.addPackages(loaded.packages)
	inventory.addMetaclasses(classes)
	inventory.Counts = countInventory(inventory)
	return inventory, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func loadPackages(manifest Manifest, sourceData map[string][]byte) (loadedPackages, error) {
	loaded := loadedPackages{
		packages:    make([]parsedPackage, 0, len(manifest.Sources)),
		classOrigin: originMap{},
		enumOrigin:  originMap{},
	}
	for _, source := range manifest.Sources {
		pkg, err := decodeSource(source, sourceData)
		if err != nil {
			return loadedPackages{}, err
		}
		loaded.packages = append(loaded.packages, parsedPackage{source: source, raw: pkg})
		loaded.recordOrigins(source.Origin, pkg)
	}
	return loaded, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func decodeSource(source Source, sourceData map[string][]byte) (ePackage, error) {
	data, ok := sourceData[source.ID]
	if !ok {
		return ePackage{}, fmt.Errorf("missing source %q", source.ID)
	}
	if err := VerifySource(source, data); err != nil {
		return ePackage{}, err
	}
	var pkg ePackage
	if err := xml.Unmarshal(data, &pkg); err != nil {
		return ePackage{}, fmt.Errorf("decode %s: %w", source.ID, err)
	}
	if pkg.Name != source.Package {
		return ePackage{}, fmt.Errorf("%s package is %q, want %q", source.ID, pkg.Name, source.Package)
	}
	return pkg, nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (l loadedPackages) recordOrigins(origin string, pkg ePackage) {
	for _, classifier := range pkg.Classifiers {
		switch classifier.Type {
		case "ecore:EClass":
			l.classOrigin.addFirst(classifier.Name, origin)
		case "ecore:EEnum":
			l.enumOrigin.addFirst(classifier.Name, origin)
		}
	}
}

// eachClassifier calls visit for every classifier of every package.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (l loadedPackages) eachClassifier(visit func(pkg parsedPackage, classifier eClassifier)) {
	for _, pkg := range l.packages {
		for _, classifier := range pkg.raw.Classifiers {
			visit(pkg, classifier)
		}
	}
}

// resolver returns the reference resolver of the packages of origin.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (l loadedPackages) resolver(origin string) resolver {
	return resolver{classes: l.classOrigin, enums: l.enumOrigin, origin: origin}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (l loadedPackages) ownClasses() map[string]*Metaclass {
	classes := map[string]*Metaclass{}
	l.eachClassifier(func(pkg parsedPackage, classifier eClassifier) {
		if classifier.Type != "ecore:EClass" || l.classOrigin[classifier.Name] != pkg.source.Origin {
			return
		}
		class := l.newClass(pkg, classifier)
		classes[class.ID] = class
	})
	return classes
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (l loadedPackages) newClass(pkg parsedPackage, classifier eClassifier) *Metaclass {
	r := l.resolver(pkg.source.Origin)
	id := qualify(pkg.source.Origin, classifier.Name)
	class := &Metaclass{
		ID:       id,
		Name:     classifier.Name,
		Origin:   pkg.source.Origin,
		Package:  pkg.raw.Name,
		Abstract: boolValue(classifier.Abstract, false),
	}
	for _, super := range strings.Fields(classifier.SuperTypes) {
		class.Supertypes = append(class.Supertypes, r.resolveType(super))
	}
	sort.Strings(class.Supertypes)
	for _, feature := range classifier.Features {
		class.OwnedProperties = append(class.OwnedProperties, r.normalizeProperty(id, feature))
	}
	sortOwnedProperties(class)
	return class
}

// sortOwnedProperties orders the owned properties of class by name.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func sortOwnedProperties(class *Metaclass) {
	sort.Slice(class.OwnedProperties, func(i, j int) bool { return class.OwnedProperties[i].Name < class.OwnedProperties[j].Name })
}

// mergeForeignFeatures adds the properties that a package of another origin
// declares on a class. The SysML Ecore is flattened over KerML and can add a
// SysML property to a KerML metaclass. Merge only additive features; retain
// KerML as class origin.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (l loadedPackages) mergeForeignFeatures(classes map[string]*Metaclass) {
	l.eachClassifier(func(pkg parsedPackage, classifier eClassifier) {
		if classifier.Type != "ecore:EClass" || l.classOrigin[classifier.Name] == pkg.source.Origin {
			return
		}
		class := classes[qualify(l.classOrigin[classifier.Name], classifier.Name)]
		l.addMissingFeatures(class, pkg, classifier)
	})
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (l loadedPackages) addMissingFeatures(class *Metaclass, pkg parsedPackage, classifier eClassifier) {
	r := l.resolver(pkg.source.Origin)
	owned := map[string]bool{}
	for _, property := range class.OwnedProperties {
		owned[property.Name] = true
	}
	for _, feature := range classifier.Features {
		if !owned[feature.Name] {
			class.OwnedProperties = append(class.OwnedProperties, r.normalizeProperty(class.ID, feature))
		}
	}
	sortOwnedProperties(class)
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func resolveInheritance(classes map[string]*Metaclass) error {
	for _, class := range classes {
		class.Relationship = isSubclass(class.ID, "KerML::Relationship", classes)
		inherited, err := inheritedProperties(class, classes, map[string]bool{})
		if err != nil {
			return err
		}
		class.InheritedProperties = inherited
		if class.Relationship {
			class.RelationshipEnds = relationshipEnds(class, classes)
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func newInventory(manifest Manifest, manifestBytes []byte) Inventory {
	return Inventory{
		SchemaVersion:      InventorySchemaVersion,
		Baseline:           manifest.Baseline,
		Authority:          manifest.MetamodelAuthority,
		FormalRelease:      manifest.FormalRelease,
		SourceManifestHash: hashBytes(manifestBytes),
	}
}

// addPackages records the sources and packages, sorted by id and origin.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (inv *Inventory) addPackages(packages []parsedPackage) {
	for _, pkg := range packages {
		inv.Sources = append(inv.Sources, InventorySource{
			ID: pkg.source.ID, Origin: pkg.source.Origin, Package: pkg.source.Package,
			Path: pkg.source.Path, URL: pkg.source.URL, SHA256: pkg.source.SHA256,
		})
		inv.Packages = append(inv.Packages, Package{Name: pkg.raw.Name, Origin: pkg.source.Origin, NSURI: pkg.raw.NSURI})
	}
	sort.Slice(inv.Sources, func(i, j int) bool { return inv.Sources[i].ID < inv.Sources[j].ID })
	sort.Slice(inv.Packages, func(i, j int) bool { return inv.Packages[i].Origin < inv.Packages[j].Origin })
}

// addMetaclasses records the classes and their inheritance edges, sorted.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (inv *Inventory) addMetaclasses(classes map[string]*Metaclass) {
	for _, class := range classes {
		inv.Metaclasses = append(inv.Metaclasses, *class)
		for _, super := range class.Supertypes {
			inv.Inheritance = append(inv.Inheritance, InheritanceEdge{Sub: class.ID, Super: super})
		}
	}
	sort.Slice(inv.Metaclasses, func(i, j int) bool { return inv.Metaclasses[i].ID < inv.Metaclasses[j].ID })
	sort.Slice(inv.Inheritance, func(i, j int) bool {
		a, b := inv.Inheritance[i], inv.Inheritance[j]
		if a.Sub == b.Sub {
			return a.Super < b.Super
		}
		return a.Sub < b.Sub
	})
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
		coverage.Entries = append(coverage.Entries, propertyEntries(class)...)
	}
	sort.Slice(coverage.Entries, func(i, j int) bool { return coverage.Entries[i].ID < coverage.Entries[j].ID })
	coverage.Counts = countCoverage(coverage.Entries)
	return coverage
}

// propertyEntries returns the coverage entries of the owned properties of
// class: derived properties are excluded, the others are mapped.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func propertyEntries(class Metaclass) []CoverageEntry {
	var entries []CoverageEntry
	for _, property := range class.OwnedProperties {
		entry := CoverageEntry{ID: property.ID, Kind: "property", Status: "mapped", Aliases: []string{"official:" + property.ID}}
		if property.Derived {
			entry.Status = "derived"
			entry.Reason = "official metamodel marks this property derived"
			entry.Aliases = nil
		}
		entries = append(entries, entry)
	}
	return entries
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func CheckCoverage(inventory Inventory, inventoryBytes []byte, coverage Coverage) error {
	if err := checkCoverageHeader(inventory, inventoryBytes, coverage); err != nil {
		return err
	}
	expected := expectedKinds(inventory)
	aliases, err := checkEntries(expected, coverage.Entries)
	if err != nil {
		return err
	}
	if err := checkSemanticAliases(expected, aliases); err != nil {
		return err
	}
	if got := countCoverage(coverage.Entries); got != coverage.Counts {
		return fmt.Errorf("coverage counts are stale: got %+v, want %+v", coverage.Counts, got)
	}
	return nil
}

// checkCoverageHeader checks the schema, baseline and inventory hash of a
// coverage binding.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func checkCoverageHeader(inventory Inventory, inventoryBytes []byte, coverage Coverage) error {
	if coverage.SchemaVersion != "1" {
		return fmt.Errorf("unsupported coverage schema %q", coverage.SchemaVersion)
	}
	if coverage.Baseline != inventory.Baseline {
		return fmt.Errorf("coverage baseline %q does not match inventory %q", coverage.Baseline, inventory.Baseline)
	}
	if coverage.InventoryHash != hashBytes(inventoryBytes) {
		return errors.New("coverage inventory hash is stale")
	}
	return nil
}

// expectedKinds maps every metaclass and property id of the inventory to its
// coverage kind.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func expectedKinds(inventory Inventory) map[string]string {
	expected := map[string]string{}
	for _, class := range inventory.Metaclasses {
		expected[class.ID] = "metaclass"
		for _, property := range class.OwnedProperties {
			expected[property.ID] = "property"
		}
	}
	return expected
}

// entryCheck accumulates the coverage entries seen so far.
type entryCheck struct {
	expected map[string]string
	seen     map[string]bool
	aliases  map[string]string
}

// checkEntries checks every coverage entry against the expected inventory
// ids and returns the alias to id map; every inventory id must be covered.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func checkEntries(expected map[string]string, entries []CoverageEntry) (map[string]string, error) {
	check := entryCheck{expected: expected, seen: map[string]bool{}, aliases: map[string]string{}}
	for _, entry := range entries {
		if err := check.add(entry); err != nil {
			return nil, err
		}
	}
	if err := check.unaccounted(); err != nil {
		return nil, err
	}
	return check.aliases, nil
}

// unaccounted returns an error for an expected id that no entry covers.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (c entryCheck) unaccounted() error {
	for id := range c.expected {
		if !c.seen[id] {
			return fmt.Errorf("official inventory entry %q is unaccounted", id)
		}
	}
	return nil
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (c entryCheck) add(entry CoverageEntry) error {
	kind, ok := c.expected[entry.ID]
	if !ok {
		return fmt.Errorf("coverage contains unknown inventory entry %q", entry.ID)
	}
	if c.seen[entry.ID] {
		return fmt.Errorf("coverage contains duplicate entry %q", entry.ID)
	}
	c.seen[entry.ID] = true
	if entry.Kind != kind {
		return fmt.Errorf("coverage entry %q has kind %q, want %q", entry.ID, entry.Kind, kind)
	}
	if err := checkStatus(entry, kind); err != nil {
		return err
	}
	for _, alias := range entry.Aliases {
		if prior, exists := c.aliases[alias]; exists {
			return fmt.Errorf("semantic alias %q is mapped by both %s and %s", alias, prior, entry.ID)
		}
		c.aliases[alias] = entry.ID
	}
	return nil
}

// checkStatus checks that the status of a coverage entry of kind is valid.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func checkStatus(entry CoverageEntry, kind string) error {
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
	return nil
}

// checkSemanticAliases checks that every semantic alias of an inventory
// metaclass is mapped to it.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func checkSemanticAliases(expected, aliases map[string]string) error {
	for official, expectedAliases := range semanticAliases() {
		if _, present := expected[official]; !present {
			continue
		}
		if err := checkAliasesOf(official, expectedAliases, aliases); err != nil {
			return err
		}
	}
	return nil
}

// checkAliasesOf checks that every alias in expectedAliases maps to official.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func checkAliasesOf(official string, expectedAliases []string, aliases map[string]string) error {
	for _, alias := range expectedAliases {
		if aliases[alias] != official {
			return fmt.Errorf("semantic alias %q must map to %s", alias, official)
		}
	}
	return nil
}

// registryHeader is the fixed start of the generated Go registry.
const registryHeader = "// Code generated by cmd/sysmlmetamodel; DO NOT EDIT.\n" +
	"package model\n\n" +
	"type OfficialSysMLProperty struct { ID string; Kind string; Type string; Lower int; Upper int; Ordered bool; Unique bool; Derived bool; ReadOnly bool; Containment bool; Opposite string; Subsets []string; Redefines []string }\n\n" +
	"type OfficialSysMLRelationshipEnd struct { Property string; Role string; Type string }\n\n" +
	"type OfficialSysMLMetaclass struct { Origin string; Abstract bool; Relationship bool; Supertypes []string; Properties map[string]OfficialSysMLProperty; RelationshipEnds []OfficialSysMLRelationshipEnd }\n\n"

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func GenerateRegistry(inventory Inventory, coverage Coverage) ([]byte, error) {
	if err := CheckCoverageWithoutHash(inventory, coverage); err != nil {
		return nil, err
	}
	var out bytes.Buffer
	out.WriteString(registryHeader)
	propertyByID := propertyIndex(inventory)
	out.WriteString("var OfficialSysMLMetaclasses = map[string]OfficialSysMLMetaclass{\n")
	for _, class := range inventory.Metaclasses {
		writeRegistryMetaclass(&out, class, propertyByID)
	}
	out.WriteString("}\n\n")
	writeSemanticAliases(&out, coverage)
	writeAliasExceptions(&out)
	return format.Source(out.Bytes())
}

// propertyIndex maps every owned property id of the inventory to the
// property.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func propertyIndex(inventory Inventory) map[string]Property {
	propertyByID := map[string]Property{}
	for _, class := range inventory.Metaclasses {
		for _, property := range class.OwnedProperties {
			propertyByID[property.ID] = property
		}
	}
	return propertyByID
}

// writeRegistryMetaclass writes the registry entry of one metaclass.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func writeRegistryMetaclass(out *bytes.Buffer, class Metaclass, propertyByID map[string]Property) {
	fmt.Fprintf(out, "\t%q: {Origin: %q, Abstract: %t, Relationship: %t", class.ID, class.Origin, class.Abstract, class.Relationship)
	if len(class.Supertypes) > 0 {
		fmt.Fprintf(out, ", Supertypes: %#v", class.Supertypes)
	}
	out.WriteString(", Properties: map[string]OfficialSysMLProperty{\n")
	for _, property := range effectiveProperties(class, propertyByID) {
		writeRegistryProperty(out, property)
	}
	out.WriteString("\t}")
	if len(class.RelationshipEnds) > 0 {
		out.WriteString(", RelationshipEnds: []OfficialSysMLRelationshipEnd{\n")
		for _, end := range class.RelationshipEnds {
			fmt.Fprintf(out, "\t\t{Property: %q, Role: %q, Type: %q},\n", end.Property, end.Role, end.Type)
		}
		out.WriteString("\t}")
	}
	out.WriteString("},\n")
}

// writeRegistryProperty writes the registry entry of one property.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func writeRegistryProperty(out *bytes.Buffer, property Property) {
	fmt.Fprintf(out, "\t\t%q: {ID: %q, Kind: %q, Type: %q, Lower: %d, Upper: %d, Ordered: %t, Unique: %t, Derived: %t, ReadOnly: %t, Containment: %t, Opposite: %q, Subsets: %#v, Redefines: %#v},\n",
		property.Name, property.ID, property.Kind, property.Type, property.Multiplicity.Lower, property.Multiplicity.Upper,
		property.Ordered, property.Unique, property.Derived, property.Derived || property.Transient || property.Volatile,
		property.Containment, property.Opposite, property.Subsets, property.Redefines)
}

// writeSemanticAliases writes the alias table of the mapped coverage entries.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func writeSemanticAliases(out *bytes.Buffer, coverage Coverage) {
	out.WriteString("var OfficialSysMLSemanticAliases = map[string]string{\n")
	for _, entry := range coverage.Entries {
		if entry.Status != "mapped" {
			continue
		}
		for _, alias := range entry.Aliases {
			fmt.Fprintf(out, "\t%q: %q,\n", alias, entry.ID)
		}
	}
	out.WriteString("}\n")
}

// writeAliasExceptions writes the table of aliases outside the official
// metamodel, sorted by alias.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func writeAliasExceptions(out *bytes.Buffer) {
	out.WriteString("\nvar SysMLSemanticAliasExceptions = map[string]string{\n")
	exceptions := semanticAliasExceptions()
	exceptionAliases := make([]string, 0, len(exceptions))
	for alias := range exceptions {
		exceptionAliases = append(exceptionAliases, alias)
	}
	sort.Strings(exceptionAliases)
	for _, alias := range exceptionAliases {
		fmt.Fprintf(out, "\t%q: %q,\n", alias, exceptions[alias])
	}
	out.WriteString("}\n")
}

// cueHeader is the fixed start of the generated CUE binding.
const cueHeader = "// Code generated by cmd/sysmlmetamodel; DO NOT EDIT.\n" + "package model\n\n" + `#MetamodelString: string
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

`

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func GenerateCUE(inventory Inventory, coverage Coverage) ([]byte, error) {
	if err := CheckCoverageWithoutHash(inventory, coverage); err != nil {
		return nil, err
	}
	propertyByID := propertyIndex(inventory)
	var out bytes.Buffer
	out.WriteString(cueHeader)
	out.WriteString("#OfficialMetaclassName: ")
	for index, class := range inventory.Metaclasses {
		if index > 0 {
			out.WriteString(" | ")
		}
		fmt.Fprintf(&out, "%q", class.ID)
	}
	out.WriteString("\n\n#OfficialMetamodelBinding: {\n")
	out.WriteString("\tmetaclass: *\"Engineering::Element\" | #OfficialMetaclassName | (#MetamodelString & =~\"^Engineering::\")\n")
	for _, class := range inventory.Metaclasses {
		writeCUEMetaclass(&out, class, propertyByID)
	}
	out.WriteString("\tif metaclass =~ \"^Engineering::\" {\n\t\tproperties?: #ExtensionProperties\n\t}\n\t...\n}\n")
	return out.Bytes(), nil
}

// writeCUEMetaclass writes the property constraints of one metaclass.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func writeCUEMetaclass(out *bytes.Buffer, class Metaclass, propertyByID map[string]Property) {
	fmt.Fprintf(out, "\tif metaclass == %q {\n\t\tproperties?: close({\n", class.ID)
	for _, property := range effectiveProperties(class, propertyByID) {
		fmt.Fprintf(out, "\t\t\t%q?: %s\n", property.Name, cuePropertyConstraint(property))
	}
	out.WriteString("\t\t})\n\t}\n")
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
	derived := "derived?: false"
	if property.Derived {
		derived = "derived: true"
	}
	alternatives := cueAlternatives(property.Multiplicity, cueShape(property))
	return fmt.Sprintf("close({values: (%s)\n%s\nimplied?: bool\n})", strings.Join(alternatives, " | "), derived)
}

// cueShape returns the CUE value type that a property holds.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func cueShape(property Property) string {
	if property.Kind == "reference" {
		return "#MetamodelReferenceValue"
	}
	switch property.Type {
	case "Ecore::Boolean", "SysML::EBoolean":
		return "#MetamodelBooleanValue"
	case "Ecore::Integer":
		return "#MetamodelIntegerValue"
	case "Ecore::Real":
		return "#MetamodelRealValue"
	}
	return "#MetamodelStringValue"
}

// cueAlternatives returns the CUE list shapes a value of the multiplicity
// can take: one per allowed count, or an open list for an unbounded upper
// bound.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func cueAlternatives(m Multiplicity, shape string) []string {
	if m.Upper < 0 {
		if m.Lower == 0 {
			return []string{"[..." + shape + "]"}
		}
		return []string{"[" + strings.Join(repeatShape(shape, m.Lower), ", ") + ", ..." + shape + "]"}
	}
	var alternatives []string
	for count := m.Lower; count <= m.Upper; count++ {
		alternatives = append(alternatives, "["+strings.Join(repeatShape(shape, count), ", ")+"]")
	}
	return alternatives
}

// repeatShape returns count copies of shape.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func repeatShape(shape string, count int) []string {
	items := make([]string, count)
	for index := range items {
		items[index] = shape
	}
	return items
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

// inheritWalk collects the properties a class inherits from its supertypes.
type inheritWalk struct {
	class   *Metaclass
	classes map[string]*Metaclass
	own     map[string]bool
	seen    map[string]bool
	result  []string
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func inheritedProperties(class *Metaclass, classes map[string]*Metaclass, visiting map[string]bool) ([]string, error) {
	if visiting[class.ID] {
		return nil, fmt.Errorf("inheritance cycle at %s", class.ID)
	}
	visiting[class.ID] = true
	defer delete(visiting, class.ID)
	w := &inheritWalk{class: class, classes: classes, own: map[string]bool{}, seen: map[string]bool{}}
	for _, property := range class.OwnedProperties {
		w.own[property.Name] = true
	}
	for _, super := range class.Supertypes {
		if err := w.walk(super); err != nil {
			return nil, err
		}
	}
	sort.Strings(w.result)
	return w.result, nil
}

// walk adds the properties of the class id and of its supertypes.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (w *inheritWalk) walk(id string) error {
	super, ok := w.classes[id]
	if !ok {
		return fmt.Errorf("%s inherits unknown metaclass %s", w.class.ID, id)
	}
	w.collect(super)
	for _, next := range super.Supertypes {
		if err := w.walk(next); err != nil {
			return err
		}
	}
	return nil
}

// collect adds the properties of super that the class does not own and that
// were not added yet.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (w *inheritWalk) collect(super *Metaclass) {
	for _, property := range super.OwnedProperties {
		if !w.own[property.Name] && !w.seen[property.ID] {
			w.result = append(w.result, property.ID)
			w.seen[property.ID] = true
		}
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func relationshipEnds(class *Metaclass, classes map[string]*Metaclass) []PropertyRef {
	var result []PropertyRef
	for _, property := range class.OwnedProperties {
		result = appendRoleEnd(result, class.ID, property)
	}
	for _, inheritedID := range class.InheritedProperties {
		if declaringID, property, ok := declaredProperty(inheritedID, classes); ok {
			result = appendRoleEnd(result, declaringID, property)
		}
	}
	sort.Slice(result, func(i, j int) bool { return result[i].Property < result[j].Property })
	return result
}

// appendRoleEnd appends the relationship end that property declared by the
// metaclass declaring forms, if it is a relationship role reference.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func appendRoleEnd(result []PropertyRef, declaring string, property Property) []PropertyRef {
	if property.Kind != "reference" || !isRelationshipRole(property) {
		return result
	}
	return append(result, PropertyRef{
		DeclaringMetaclass: declaring, Property: property.ID,
		Role: property.Name, Type: property.Type,
	})
}

// declaredProperty finds the declaring metaclass id and the property of an
// inherited property id.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func declaredProperty(inheritedID string, classes map[string]*Metaclass) (string, Property, bool) {
	cut := strings.LastIndex(inheritedID, "::")
	declaringID, propertyName := inheritedID[:cut], inheritedID[cut+2:]
	declaring := classes[declaringID]
	if declaring == nil {
		return "", Property{}, false
	}
	for _, property := range declaring.OwnedProperties {
		if property.Name == propertyName {
			return declaringID, property, true
		}
	}
	return "", Property{}, false
}

// relationshipRoleSuffixes are the endings of the property references that
// make a property a relationship role.
var relationshipRoleSuffixes = []string{"::source", "::target", "::relatedElement", "::relatedFeature", "::connectorEnd"}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func isRelationshipRole(property Property) bool {
	switch property.Name {
	case "source", "target", "relatedElement", "owningRelatedElement", "ownedRelatedElement", "relatedFeature", "connectorEnd":
		return true
	}
	for _, ref := range append(append([]string{}, property.Subsets...), property.Redefines...) {
		if hasRoleSuffix(ref) {
			return true
		}
	}
	return false
}

// hasRoleSuffix reports whether ref ends with a relationship role suffix.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func hasRoleSuffix(ref string) bool {
	for _, suffix := range relationshipRoleSuffixes {
		if strings.HasSuffix(ref, suffix) {
			return true
		}
	}
	return false
}

// lineage walks the supertypes of metaclasses looking for a target.
type lineage struct {
	target  string
	classes map[string]*Metaclass
	seen    map[string]bool
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func isSubclass(id, target string, classes map[string]*Metaclass) bool {
	return (&lineage{target: target, classes: classes, seen: map[string]bool{}}).reaches(id)
}

// reaches reports whether the class id is the target or inherits from it.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (l *lineage) reaches(id string) bool {
	if id == l.target {
		return true
	}
	if l.seen[id] {
		return false
	}
	l.seen[id] = true
	class := l.classes[id]
	if class == nil {
		return false
	}
	for _, super := range class.Supertypes {
		if l.reaches(super) {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func countInventory(inventory Inventory) Counts {
	counts := Counts{Packages: len(inventory.Packages), Metaclasses: len(inventory.Metaclasses), InheritanceEdges: len(inventory.Inheritance)}
	for _, class := range inventory.Metaclasses {
		counts.addClass(class)
	}
	return counts
}

// addClass adds the counts of one metaclass.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (c *Counts) addClass(class Metaclass) {
	if class.Abstract {
		c.AbstractMetaclasses++
	}
	if class.Relationship {
		c.Relationships++
	}
	c.OwnedProperties += len(class.OwnedProperties)
	c.InheritedProperties += len(class.InheritedProperties)
	for _, property := range class.OwnedProperties {
		if property.Derived {
			c.DerivedProperties++
		}
	}
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
	aliases := kermlAliases()
	maps.Copy(aliases, sysmlDefinitionAliases())
	maps.Copy(aliases, sysmlUsageAliases())
	return aliases
}

// kermlAliases returns the semantic aliases of the KerML metaclasses.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func kermlAliases() map[string][]string {
	return map[string][]string{
		"KerML::Dependency":          {"dependency"},
		"KerML::FeatureTyping":       {"typing"},
		"KerML::OwningMembership":    {"containment"},
		"KerML::Package":             {"package_definition"},
		"KerML::Redefinition":        {"redefinition"},
		"KerML::ReferenceSubsetting": {"reference"},
		"KerML::Specialization":      {"specialization"},
		"KerML::Subsetting":          {"subsetting"},
		"KerML::Succession":          {"succession"},
	}
}

// sysmlDefinitionAliases returns the semantic aliases of the first half of the SysML
// metaclasses.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func sysmlDefinitionAliases() map[string][]string {
	return map[string][]string{
		"SysML::ActionDefinition":        {"action_definition"},
		"SysML::ActionUsage":             {"action_usage"},
		"SysML::AllocationUsage":         {"allocation"},
		"SysML::AnalysisCaseDefinition":  {"analysis_case_definition"},
		"SysML::AnalysisCaseUsage":       {"analysis_case_usage"},
		"SysML::AttributeDefinition":     {"attribute_definition"},
		"SysML::AttributeUsage":          {"attribute_usage"},
		"SysML::BindingConnectorAsUsage": {"binding"},
		"SysML::CalculationDefinition":   {"calculation_definition"},
		"SysML::CalculationUsage":        {"calculation_usage"},
		"SysML::CaseDefinition":          {"case_definition"},
		"SysML::CaseUsage":               {"case_usage"},
		"SysML::ConcernDefinition":       {"concern_definition"},
		"SysML::ConcernUsage":            {"concern_usage"},
		"SysML::ConnectionDefinition":    {"connection_definition"},
		"SysML::ConnectionUsage":         {"connection", "connection_usage"},
		"SysML::ConstraintDefinition":    {"constraint_definition"},
		"SysML::ConstraintUsage":         {"constraint_usage"},
		"SysML::ControlNode":             {"control_node"},
		"SysML::EventOccurrenceUsage":    {"event_definition", "event_usage"},
		"SysML::FlowUsage":               {"transfer"},
		"SysML::InterfaceDefinition":     {"interface_definition"},
	}
}

// sysmlUsageAliases returns the semantic aliases of the second half of the SysML
// metaclasses.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func sysmlUsageAliases() map[string][]string {
	return map[string][]string{
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

// normalizeProperty converts an Ecore feature of the class classID to a
// property.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (r resolver) normalizeProperty(classID string, feature eFeature) Property {
	property := Property{
		ID:           classID + "::" + feature.Name,
		Name:         feature.Name,
		Origin:       r.origin,
		Kind:         featureKind(feature.Type),
		Type:         r.resolveType(feature.EType),
		Multiplicity: Multiplicity{Lower: intValue(feature.Lower, 0), Upper: intValue(feature.Upper, 1)},
		Ordered:      boolValue(feature.Ordered, true),
		Unique:       boolValue(feature.Unique, true),
		Derived:      boolValue(feature.Derived, false),
		Transient:    boolValue(feature.Transient, false),
		Volatile:     boolValue(feature.Volatile, false),
		Containment:  boolValue(feature.Containment, false),
		Opposite:     r.resolvePropertyRef(feature.Opposite),
	}
	for _, annotation := range feature.Annotations {
		switch annotation.Source {
		case "subsets":
			property.Subsets = append(property.Subsets, r.resolvePropertyRefs(annotation.References)...)
		case "redefines":
			property.Redefines = append(property.Redefines, r.resolvePropertyRefs(annotation.References)...)
		}
	}
	sort.Strings(property.Subsets)
	sort.Strings(property.Redefines)
	return property
}

// resolveType resolves an Ecore type reference to a qualified name.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (r resolver) resolveType(raw string) string {
	name := fragmentName(raw)
	if name == "" {
		return raw
	}
	if origin, ok := r.classes[name]; ok {
		return qualify(origin, name)
	}
	if origin, ok := r.enums[name]; ok {
		return qualify(origin, name)
	}
	if strings.Contains(raw, "Types.ecore") {
		return "Ecore::" + name
	}
	return qualify(r.origin, name)
}

// resolvePropertyRefs resolves a list of property references.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (r resolver) resolvePropertyRefs(raw string) []string {
	var result []string
	for _, ref := range strings.Fields(raw) {
		if resolved := r.resolvePropertyRef(ref); resolved != "" {
			result = append(result, resolved)
		}
	}
	return result
}

// resolvePropertyRef resolves a property reference to a qualified name.
// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func (r resolver) resolvePropertyRef(raw string) string {
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
	origin := r.origin
	if known, ok := r.classes[parts[0]]; ok {
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
