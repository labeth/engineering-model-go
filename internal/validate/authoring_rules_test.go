// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"reflect"
	"sort"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
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

// The authoring contract says that state behavior is modeled with machines,
// names the removed vocabulary and the machine fields, expression operators
// and diagnostics, and lists the assurance link diagnostics; the machine
// entity kinds are field sets.
// Mutation: deleting the machines field rule or a machine field set fails this test.
// TRLC-LINKS: REQ-EMG-045, REQ-EMG-087, REQ-EMG-138, REQ-EMG-139, REQ-EMG-140
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func TestAuthoringRulesDefineMachinesAndAssuranceLinks(t *testing.T) {
	text := strings.Join(AuthoringRules().FieldRules, "\n")
	for _, want := range []string{
		"State behavior is modeled with behavior.machines only",
		"behavior.states, transitions_to and the state triggered_by were removed in v0.11.0",
		"guarded_by (protected by a control, trust boundary or referenced element) keeps its meaning",
		"behavior.transition_unknown_event", "behavior.expression_unknown_variable", "behavior.expression_type",
		"behavior.in_state_foreign", "behavior.final_state_has_transitions",
		"behavior.machine_deadlock", "behavior.transition_nondeterministic", "behavior.transition_no_trigger", "behavior.invariant_unlinked",
		"behavior.event_unused counts as a use of an event: the on of a machine transition",
		"threat.verification_missing", "mitigation.verification_control_mismatch", "verification.unlinked", "gemara.control_needs_review",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("authoring field rules must contain %q:\n%s", want, text)
		}
	}
	for _, removed := range []string{"Entering a state is a transitions_to", "first declared, initial one"} {
		if strings.Contains(text, removed) {
			t.Fatalf("authoring field rules still describe removed state vocabulary %q", removed)
		}
	}
	sets := AuthoringRules().FieldSets
	if _, ok := sets["state"]; ok {
		t.Fatal("the legacy state field set was removed with behavior.states")
	}
	want := map[string]string{
		"machine":           "id,name,description,owner,initial,variables,states,transitions,invariants",
		"machineState":      "id,name,description,final",
		"machineTransition": "id,from,to,on,guard,effect,description",
		"machineInvariant":  "id,requirements,expr,description",
		"machineVariable":   "name,type,initial",
	}
	for kind, fields := range want {
		if got := strings.Join(sets[kind], ","); got != fields {
			t.Fatalf("fieldSets[%s] = %s, want %s", kind, got, fields)
		}
	}
	for _, pair := range AuthoringRules().RelationshipPairs {
		if pair.Type == "transitions_to" || pair.Type == "triggered_by" {
			t.Fatalf("removed relationship pair %+v is still allowed", pair)
		}
	}
}

// publicationSections are the yaml fields of model.ManifestPublication after
// id, in declaration order, so a new publication section reaches the
// authoring contract without a second list.
// Mutation: returning a fixed list without "views" fails this test.
// TRLC-LINKS: REQ-EMG-087
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func TestPublicationSectionsAreManifestPublicationFields(t *testing.T) {
	fields := fieldSets()["publication"]
	if len(fields) < 2 || fields[0] != "id" {
		t.Fatalf("publication fields = %v, want id first", fields)
	}
	got := AuthoringRules().PublicationSections
	if strings.Join(got, ",") != strings.Join(fields[1:], ",") || strings.Join(got, ",") != "architecture,behavior,assurance,compliance,requirements,views" {
		t.Fatalf("publicationSections = %v, want the ManifestPublication fields after id %v", got, fields[1:])
	}
}

// The authoring contract describes realizes, assumptions of use, the
// requirement link file with its authoring path (engws link add) and
// version-pinned review, and requirementLinks strictness, and lists the
// fields of an assumption and a link; it no longer tells an agent to keep
// an engine-written fingerprint.
// TRLC-LINKS: REQ-EMG-099, REQ-EMG-119
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func TestAuthoringRulesDescribeRequirementLinks(t *testing.T) {
	rules := AuthoringRules()
	text := strings.Join(rules.FieldRules, "\n")
	for _, want := range []string{
		"realizes", "requirement.no_parent", "parent of a reqLinks entry to a dependency's capability", "A system publishes its own capabilities so a system above can link to them", "contract.assumptions", "contract.realizer_unpublished",
		"documents.reqLinks", "engws link add", "link.version_mismatch", "link.fingerprint_deprecated", "not status or notes", "link.aou_unverified", "requirementLinks in engmod.yml is strict or guided", "pseudo-version",
	} {
		if !strings.Contains(text, want) {
			t.Fatalf("authoring field rules must contain %q", want)
		}
	}
	if strings.Contains(text, "topLevel") {
		t.Fatal("authoring field rules must not mention the removed inferenceHints.topLevel")
	}
	if strings.Contains(text, "link.suspect") || strings.Contains(text, "written by the engine, never by hand") {
		t.Fatal("authoring field rules must not describe the removed fingerprint")
	}
	if got := strings.Join(rules.FieldSets["contractAssumption"], ","); got != "id,text,verificationCriteria" {
		t.Fatalf("contractAssumption fields = %s", got)
	}
	if got := strings.Join(rules.FieldSets["reqLink"], ","); got != "id,parent,child,evidence,pinned,status,rationale,reviewedBy,reviewedOn" {
		t.Fatalf("reqLink fields = %s", got)
	}
	if !strings.Contains(strings.Join(rules.FieldSets["requirement"], ","), "realizes") {
		t.Fatal("requirement fields must list realizes")
	}
}

// fieldTypes gives every field of every fieldSets kind its YAML type, derived
// from the strict-decoding Go types: stride is one string while
// preconditions, exploitPath, detection and findings are lists of strings,
// steps a list of objects of kind flowStep, acceptedBy an object of kind
// acceptanceRecord, boundaryCrossing a bool. The field rules state the flow
// step ref and unique step ID rules, the catalog.term_unreferenced rule and
// the exclude forms.
// Mutation: typing every slice as list<string>, dropping the of kind, or
// removing one of the stated rules fails this test.
// TRLC-LINKS: REQ-EMG-115
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, DO-MODEL-AUTHORING-CONTRACT
func TestAuthoringRulesGiveFieldTypes(t *testing.T) {
	rules := AuthoringRules()
	typeOf := func(kind, field string) string {
		for _, f := range rules.FieldTypes[kind] {
			if f.Name == field {
				if f.Of != "" {
					return f.Type + " of " + f.Of
				}
				return f.Type
			}
		}
		return "missing"
	}
	for _, c := range []struct{ kind, field, want string }{
		{"threatScenario", "stride", "string"},
		{"threatScenario", "preconditions", "list<string>"},
		{"threatScenario", "exploitPath", "list<string>"},
		{"threatScenario", "detection", "list<string>"},
		{"threatScenario", "evidence", "list<object> of evidence"},
		{"threatScenario", "acceptedBy", "object of acceptanceRecord"},
		{"controlVerification", "findings", "list<string>"},
		{"flow", "steps", "list<object> of flowStep"},
		{"flowStep", "boundaryCrossing", "bool"},
		{"flowStep", "ref", "string"},
		{"reqLink", "pinned", "object of reqLinkPin"},
	} {
		if got := typeOf(c.kind, c.field); got != c.want {
			t.Errorf("fieldTypes.%s.%s = %s, want %s", c.kind, c.field, got, c.want)
		}
	}
	for kind, names := range rules.FieldSets {
		if len(rules.FieldTypes[kind]) != len(names) {
			t.Errorf("fieldTypes.%s has %d fields, fieldSets %d", kind, len(rules.FieldTypes[kind]), len(names))
			continue
		}
		for i, f := range rules.FieldTypes[kind] {
			if f.Name != names[i] || f.Type == "" {
				t.Errorf("fieldTypes.%s[%d] = %+v, want %s with a type", kind, i, f, names[i])
			}
		}
	}
	text := strings.Join(rules.FieldRules, "\n")
	for _, want := range []string{
		"every flow step needs ref", "model.missing_flow_step_ref", "step IDs are unique across all flows",
		"semantic.duplicate_identity", "stride is one string", "catalog.term_unreferenced",
		"lintRun.mode: strict", "dir, dir/, dir/* or dir/**", "model.invalid_exclude_pattern", "fieldTypes",
	} {
		if !strings.Contains(text, want) {
			t.Errorf("authoring field rules must contain %q", want)
		}
	}
}

// The authoring contract gives the lifecycle enums, field sets of
// lifecycle and poamDefer, the severity-by-tier table and field rules naming
// tierDecision, raise, defers and the rank-0 codes.
// TRLC-LINKS: REQ-EMG-131
func TestAuthoringRulesDescribeLifecycle(t *testing.T) {
	rules := AuthoringRules()
	if got := strings.Join(rules.Enums["lifecycle.tier"], ","); got != "mvp,operational,poc,product,prototype" {
		t.Errorf("lifecycle.tier enum = %s", got)
	}
	if got := strings.Join(rules.Enums["lifecycle.bundles"], ","); got != "foundation,safety,security" {
		t.Errorf("lifecycle.bundles enum = %s", got)
	}
	if got := strings.Join(rules.FieldSets["lifecycle"], ","); got != "tier,bundles,tierDecision,raise" {
		t.Errorf("lifecycle fields = %s", got)
	}
	if got := strings.Join(rules.FieldSets["poamDefer"], ","); got != "code,path" {
		t.Errorf("poamDefer fields = %s", got)
	}
	if !containsTrimmed(rules.FieldSets["poamItem"], "defers") {
		t.Errorf("poamItem fields lack defers: %v", rules.FieldSets["poamItem"])
	}
	if len(rules.TierPolicy) != len(TierPolicy()) || rules.TierPolicy[0].Code == "" {
		t.Errorf("TierPolicy in rules: %d rows", len(rules.TierPolicy))
	}
	text := strings.Join(rules.FieldRules, "\n")
	for _, want := range []string{"lifecycle.tier_decision_mismatch", "lifecycle.raise", "defers", "tier:<project tier>", "lifecycle.poam_defers_core", "deferred", "requiredFrom", "lifecycle.requirement_links_conflict", "aviation.ai_actor_as_credit"} {
		if !strings.Contains(text, want) {
			t.Errorf("field rules do not mention %s", want)
		}
	}
}
