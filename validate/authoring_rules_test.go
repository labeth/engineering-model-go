// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"reflect"
	"sort"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-045, REQ-EMG-009
func TestAuthoringRulesEnumsEqualValidationTables(t *testing.T) {
	rules := AuthoringRules()
	if len(rules.Enums) != len(enumTables) {
		t.Fatalf("expected %d enums, got %d", len(enumTables), len(rules.Enums))
	}
	for key, table := range enumTables {
		want := []string{}
		for value := range table {
			want = append(want, value)
		}
		sort.Strings(want)
		if !reflect.DeepEqual(rules.Enums[key], want) {
			t.Fatalf("enum %s = %v, want %v", key, rules.Enums[key], want)
		}
	}
	for key, table := range map[string]map[string]bool{
		"risk.status": allowedRiskStatus, "poamItem.status": allowedPOAMStatus, "relationship.type": allowedMappingTypes,
		"threatOutOfScope.status": allowedThreatOutOfScopeStatus, "flowStep.kind": allowedFlowStepKinds,
	} {
		if len(rules.Enums[key]) != len(table) {
			t.Fatalf("enum %s missing or incomplete: %v", key, rules.Enums[key])
		}
	}
	if len(rules.FieldRules) == 0 {
		t.Fatal("expected field rules")
	}
}

// TRLC-LINKS: REQ-EMG-045, REQ-EMG-001, REQ-EMG-009
func TestAuthoringRulesRelationshipPairsMatchValidation(t *testing.T) {
	rules := AuthoringRules()
	returned := map[RelationshipPair]bool{}
	count := 0
	for _, kinds := range mappingPairs {
		count += len(kinds)
	}
	if len(rules.RelationshipPairs) != count {
		t.Fatalf("expected %d pairs, got %d", count, len(rules.RelationshipPairs))
	}
	for _, pair := range rules.RelationshipPairs {
		returned[pair] = true
		if !allowedMappingTypes[pair.Type] {
			t.Fatalf("pair %+v uses a type validation rejects", pair)
		}
		if !mappingPairAllowed(pair.Type, pair.From, pair.To) {
			t.Fatalf("returned pair %+v rejected by validation", pair)
		}
	}
	for _, pair := range []RelationshipPair{
		{"contains", "functional_group", "functional_unit"},
		{"contains", "hardware_item", "hardware_interface"},
		{"interacts_with", "actor", "functional_unit"},
		{"bounded_by", "interface", "trust_boundary"},
	} {
		if !returned[pair] {
			t.Fatalf("expected pair %+v to be returned", pair)
		}
	}
	for _, pair := range []RelationshipPair{
		{"writes", "actor", "data_object"},
		{"contains", "functional_unit", "functional_unit"},
		{"interacts_with", "functional_unit", "actor"},
		{"mitigated_by", "threat_scenario", "control"},
	} {
		if returned[pair] {
			t.Fatalf("pair %+v must not be returned", pair)
		}
		if mappingPairAllowed(pair.Type, pair.From, pair.To) {
			t.Fatalf("pair %+v not returned but accepted by validation", pair)
		}
	}
}

// TRLC-LINKS: REQ-EMG-045, REQ-EMG-001, REQ-EMG-009
func TestAuthoringRulesPairsAgreeWithBundleValidation(t *testing.T) {
	base := model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}, {ID: "FU-B", Group: "FG-A", Name: "Unit B"}},
		Actors:           []model.Actor{{ID: "ACT-A", Name: "Actor A"}},
		DataObjects:      []model.DataObject{{ID: "DATA-A", Name: "Payload"}},
		TrustBoundaries:  []model.TrustBoundary{{ID: "TB-A", Name: "Boundary"}},
	}
	for _, tc := range []struct {
		mapping model.Mapping
		pair    RelationshipPair
	}{
		{model.Mapping{Type: "contains", From: "FG-A", To: "FU-A"}, RelationshipPair{"contains", "functional_group", "functional_unit"}},
		{model.Mapping{Type: "writes", From: "FU-A", To: "DATA-A"}, RelationshipPair{"writes", "functional_unit", "data_object"}},
		{model.Mapping{Type: "bounded_by", From: "ACT-A", To: "TB-A"}, RelationshipPair{"bounded_by", "actor", "trust_boundary"}},
		{model.Mapping{Type: "writes", From: "ACT-A", To: "DATA-A"}, RelationshipPair{"writes", "actor", "data_object"}},
		{model.Mapping{Type: "contains", From: "FU-A", To: "FU-B"}, RelationshipPair{"contains", "functional_unit", "functional_unit"}},
	} {
		returned := false
		for _, pair := range AuthoringRules().RelationshipPairs {
			if pair == tc.pair {
				returned = true
			}
		}
		arch := base
		arch.Mappings = []model.Mapping{tc.mapping}
		b := schemaV2TestBundle(model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: arch, Views: []model.View{{ID: "V", Kind: "traceability", Roots: []string{"FU-A"}}}}})
		rejected := false
		for _, d := range Bundle(b) {
			if d.Code == "model.invalid_mapping_pair" {
				rejected = true
			}
		}
		if returned == rejected {
			t.Fatalf("%+v: returned=%v but validation rejected=%v", tc.pair, returned, rejected)
		}
	}
}

// TRLC-LINKS: REQ-EMG-045, REQ-EMG-009
func TestAuthoringFieldRuleOutOfScopeRejectsSystemAndNonActorOwner(t *testing.T) {
	b := schemaV2TestBundle(model.Bundle{
		Catalog: model.CatalogDocument{Catalog: model.CatalogGroups{Systems: []model.CatalogEntry{{ID: "SYS-A", Name: "System"}}}},
		Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
			ThreatOutOfScope: []model.ThreatOutOfScope{{ID: "TOS-A", Title: "Out", Reason: "Because", Owner: "FU-A", AppliesTo: []string{"SYS-A", "FU-A"}}},
		}, Views: []model.View{{ID: "V", Kind: "traceability", Roots: []string{"FU-A"}}}},
	})
	codes := map[string]string{}
	for _, d := range Bundle(b) {
		codes[d.Code] = d.Path
	}
	if codes["model.invalid_threat_out_of_scope_target"] != "authoredArchitecture.threatOutOfScope[0].appliesTo[0]" {
		t.Fatalf("expected SYS-A appliesTo rejection, got %+v", codes)
	}
	if _, ok := codes["model.invalid_threat_out_of_scope_owner"]; !ok {
		t.Fatalf("expected non-actor owner rejection, got %+v", codes)
	}
}
