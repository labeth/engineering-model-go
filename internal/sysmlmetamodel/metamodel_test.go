package sysmlmetamodel

import (
	"crypto/sha256"
	"encoding/hex"
	"strings"
	"testing"
)

const testEcore = `<?xml version="1.0" encoding="UTF-8"?>
<ecore:EPackage xmlns:ecore="http://www.eclipse.org/emf/2002/Ecore"
 xmlns:xsi="http://www.w3.org/2001/XMLSchema-instance" name="kerml" nsURI="urn:test">
 <eClassifiers xsi:type="ecore:EClass" name="Relationship" abstract="true">
  <eStructuralFeatures xsi:type="ecore:EReference" name="source" upperBound="-1" eType="#//Base"/>
  <eStructuralFeatures xsi:type="ecore:EReference" name="target" upperBound="-1" eType="#//Base"/>
 </eClassifiers>
 <eClassifiers xsi:type="ecore:EClass" name="Base">
  <eStructuralFeatures xsi:type="ecore:EAttribute" name="name" lowerBound="1"
   eType="ecore:EDataType Types.ecore#//String"/>
  <eStructuralFeatures xsi:type="ecore:EReference" name="derivedItems" upperBound="-1"
   eType="#//Base" derived="true" transient="true" volatile="true"/>
 </eClassifiers>
 <eClassifiers xsi:type="ecore:EClass" name="Child" eSuperTypes="#//Base"/>
 <eClassifiers xsi:type="ecore:EClass" name="Link" eSuperTypes="#//Relationship">
  <eStructuralFeatures xsi:type="ecore:EReference" name="linked" lowerBound="1" upperBound="2"
   eType="#//Child" containment="true">
   <eAnnotations source="subsets" references="#//Relationship/target"/>
  </eStructuralFeatures>
 </eClassifiers>
</ecore:EPackage>`

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func TestBuildIsDeterministicAndExtractsSemantics(t *testing.T) {
	manifest, manifestBytes, sources := fixture()
	first, err := Build(manifest, manifestBytes, sources)
	if err != nil {
		t.Fatal(err)
	}
	second, err := Build(manifest, manifestBytes, sources)
	if err != nil {
		t.Fatal(err)
	}
	firstJSON, _ := Marshal(first)
	secondJSON, _ := Marshal(second)
	if string(firstJSON) != string(secondJSON) {
		t.Fatal("inventory generation is not deterministic")
	}
	checkInheritedProperties(t, first)
	checkDerivedProperty(t, first)
	checkRelationshipSemantics(t, first)
}

// checkInheritedProperties checks that Child inherits the properties of Base,
// sorted.
func checkInheritedProperties(t *testing.T, inventory Inventory) {
	t.Helper()
	child := findClass(t, inventory, "KerML::Child")
	if len(child.InheritedProperties) != 2 ||
		child.InheritedProperties[0] != "KerML::Base::derivedItems" ||
		child.InheritedProperties[1] != "KerML::Base::name" {
		t.Fatalf("unexpected inherited properties: %#v", child.InheritedProperties)
	}
}

// checkDerivedProperty checks that a derived property is normalized to an
// optional unbounded one.
func checkDerivedProperty(t *testing.T, inventory Inventory) {
	t.Helper()
	derived := findProperty(t, findClass(t, inventory, "KerML::Base"), "derivedItems")
	if !derived.Derived || derived.Multiplicity.Lower != 0 || derived.Multiplicity.Upper != -1 {
		t.Fatalf("derived property was not normalized: %#v", derived)
	}
}

// checkRelationshipSemantics checks that Link is a relationship whose
// containment property has its multiplicity and that it has three ends.
func checkRelationshipSemantics(t *testing.T, inventory Inventory) {
	t.Helper()
	link := findClass(t, inventory, "KerML::Link")
	linked := findProperty(t, link, "linked")
	if !link.Relationship || len(link.RelationshipEnds) != 3 {
		t.Fatalf("relationship semantics were not normalized: class=%#v property=%#v", link, linked)
	}
	if linked.Multiplicity != (Multiplicity{Lower: 1, Upper: 2}) || !linked.Containment {
		t.Fatalf("relationship property was not normalized: class=%#v property=%#v", link, linked)
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func TestVerifySourceRejectsHashMismatch(t *testing.T) {
	source := Source{ID: "test", SHA256: strings.Repeat("0", 64)}
	if err := VerifySource(source, []byte(testEcore)); err == nil || !strings.Contains(err.Error(), "hash mismatch") {
		t.Fatalf("expected hash mismatch, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func TestCoverageRejectsUnaccountedInventoryEntry(t *testing.T) {
	manifest, manifestBytes, sources := fixture()
	inventory, err := Build(manifest, manifestBytes, sources)
	if err != nil {
		t.Fatal(err)
	}

	inventoryBytes, _ := Marshal(inventory)
	for _, kind := range []string{"metaclass", "property"} {
		t.Run(kind, func(t *testing.T) {
			coverage := BootstrapCoverage(inventory, inventoryBytes)
			coverage.Entries = withoutFirstOfKind(coverage.Entries, kind)
			if err := CheckCoverage(inventory, inventoryBytes, coverage); err == nil || !strings.Contains(err.Error(), "unaccounted") {
				t.Fatalf("expected unaccounted %s error, got %v", kind, err)
			}
		})
	}
}

// withoutFirstOfKind returns the entries without the first one of kind.
func withoutFirstOfKind(entries []CoverageEntry, kind string) []CoverageEntry {
	for index, entry := range entries {
		if entry.Kind == kind {
			return append(entries[:index], entries[index+1:]...)
		}
	}
	return entries
}

// TRLC-LINKS: REQ-EMG-039
func TestCoverageRejectsUnimplementedNormativeEntry(t *testing.T) {
	manifest, manifestBytes, sources := fixture()
	inventory, err := Build(manifest, manifestBytes, sources)
	if err != nil {
		t.Fatal(err)
	}
	inventoryBytes, _ := Marshal(inventory)
	coverage := BootstrapCoverage(inventory, inventoryBytes)
	coverage.Entries[0].Status = "unimplemented"
	coverage.Counts = countCoverage(coverage.Entries)
	if err := CheckCoverage(inventory, inventoryBytes, coverage); err == nil || !strings.Contains(err.Error(), "unimplemented") {
		t.Fatalf("expected unimplemented error, got %v", err)
	}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func fixture() (Manifest, []byte, map[string][]byte) {
	data := []byte(testEcore)
	sum := sha256.Sum256(data)
	manifest := Manifest{
		SchemaVersion:      "1",
		Baseline:           "test",
		MetamodelAuthority: "test fixture",
		Sources: []Source{{
			ID: "kerml", Origin: "KerML", Package: "kerml",
			Path: "kerml.ecore", URL: "https://example.invalid/kerml.ecore",
			SHA256: hex.EncodeToString(sum[:]),
		}},
	}
	manifestBytes, _ := Marshal(manifest)
	return manifest, manifestBytes, map[string][]byte{"kerml": data}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func findClass(t *testing.T, inventory Inventory, id string) Metaclass {
	t.Helper()
	for _, class := range inventory.Metaclasses {
		if class.ID == id {
			return class
		}
	}
	t.Fatalf("missing class %s", id)
	return Metaclass{}
}

// TRLC-LINKS: REQ-EMG-038, REQ-EMG-039
func findProperty(t *testing.T, class Metaclass, name string) Property {
	t.Helper()
	for _, property := range class.OwnedProperties {
		if property.Name == name {
			return property
		}
	}
	t.Fatalf("missing property %s on %s", name, class.ID)
	return Property{}
}
