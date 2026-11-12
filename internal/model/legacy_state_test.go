// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"path/filepath"
	"strings"
	"testing"
)

const legacyBehaviorHead = "schemaVersion: 2\nbehavior:\n"

const legacyMachinesBlock = `  machines:
    - id: SM-A
      owner: FU-A
      initial: STATE-A
      states:
        - {id: STATE-A}
        - {id: STATE-B}
      transitions:
        - {id: TR-A, from: STATE-A, to: STATE-B}
`

// A behavior document with a states list, a transitions_to relationship or a
// triggered_by relationship fails to load with one message that names the
// field and machines:, not with a strict-decode unknown-field error.
// Mutation: dropping the pre-check yields a CUE or unknown-field error that
// lacks the removal message.
// TRLC-LINKS: REQ-EMG-139
// ENGMODEL-LINKS: FU-MODEL-LOADER
func TestLoadBundleRejectsRemovedStateVocabularyWithAPreciseMessage(t *testing.T) {
	cases := map[string]struct {
		behavior string
		want     string
	}{
		"states list": {
			legacyBehaviorHead + "  states:\n    - {id: STATE-A, name: A}\n" + legacyMachinesBlock,
			"behavior.states was removed in v0.11.0: model state behavior with behavior.machines (see ADR-EMG-038)",
		},
		"empty states list": {
			legacyBehaviorHead + "  states: []\n",
			"behavior.states was removed in v0.11.0: model state behavior with behavior.machines (see ADR-EMG-038)",
		},
		"transitions_to": {
			legacyBehaviorHead + legacyMachinesBlock + "  relationships:\n    - {type: writes, from: FU-A, to: DO-A}\n    - {type: transitions_to, from: STATE-A, to: STATE-B}\n",
			"behavior.relationships[1].type (transitions_to) was removed in v0.11.0: model state behavior with behavior.machines (see ADR-EMG-038)",
		},
		"triggered_by": {
			legacyBehaviorHead + legacyMachinesBlock + "  relationships:\n    - {type: triggered_by, from: STATE-A, to: EVT-A}\n",
			"behavior.relationships[0].type (triggered_by) was removed in v0.11.0: model state behavior with behavior.machines (see ADR-EMG-038)",
		},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := loadMachineFixture(t, c.behavior)
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("want an error containing %q, got %v", c.want, err)
			}
			if !strings.Contains(err.Error(), "model/behavior.yml") {
				t.Fatalf("the error must name the file, got %v", err)
			}
			if strings.Contains(err.Error(), "CUE schema validation") || strings.Contains(err.Error(), "not found in type") {
				t.Fatalf("the removal message must replace the schema error, got %v", err)
			}
		})
	}
}

// A document with machines, guarded_by and the other supported relationship
// kinds loads; so does a document with only a machines block.
// Mutation: rejecting every relationship whose from is a state loads neither.
// TRLC-LINKS: REQ-EMG-139
// ENGMODEL-LINKS: FU-MODEL-LOADER
func TestLoadBundleAcceptsMachinesAndGuardedBy(t *testing.T) {
	behavior := legacyBehaviorHead + legacyMachinesBlock +
		"  relationships:\n    - {type: writes, from: FU-A, to: DO-A}\n    - {type: guarded_by, from: STATE-B, to: CTRL-A}\n"
	bundle, err := loadMachineFixture(t, behavior)
	if err != nil {
		t.Fatal(err)
	}
	if got := bundle.Architecture.AuthoredArchitecture.MachineStates(); len(got) != 2 {
		t.Fatalf("machine states = %+v", got)
	}
	if _, err := loadMachineFixture(t, legacyBehaviorHead+legacyMachinesBlock); err != nil {
		t.Fatal(err)
	}
}

// The schema no longer lists transitions_to and triggered_by as relationship
// kinds, and a document handed to the strict decoder directly (without the
// pre-check) rejects the old fields.
// Mutation: restoring either kind to #SupportedMappingKind fails the first case.
// TRLC-LINKS: REQ-EMG-139
func TestSchemaDoesNotListRemovedStateVocabulary(t *testing.T) {
	dir := writeV2Fixture(t, validManifest)
	path := filepath.Join(dir, "model", "behavior.yml")
	for _, source := range []string{
		legacyBehaviorHead + "  relationships:\n    - {type: transitions_to, from: FU-A, to: FU-A}\n",
		legacyBehaviorHead + "  states:\n    - {id: STATE-A}\n",
	} {
		writeTestFile(t, path, source)
		var document BehaviorDocument
		err := validateCanonicalYAML(path, []byte(source), &document)
		if err == nil || !strings.Contains(err.Error(), "CUE schema validation failed") {
			t.Fatalf("the schema must reject %q, got %v", source, err)
		}
	}
}

// LegacyStateVocabulary finds the first removed field, ignores documents it
// cannot parse and documents without the removed vocabulary.
// TRLC-LINKS: REQ-EMG-139
func TestLegacyStateVocabulary(t *testing.T) {
	for source, want := range map[string]string{
		legacyBehaviorHead + "  states: []\n": "behavior.states",
		legacyBehaviorHead + "  relationships:\n    - {type: guarded_by, from: A, to: B}\n    - {type: triggered_by}\n": "behavior.relationships[1].type",
		legacyBehaviorHead + legacyMachinesBlock: "",
		"behavior: [unterminated":                "",
		"":                                       "",
	} {
		if got, _, found := LegacyStateVocabulary([]byte(source)); got != want || found != (want != "") {
			t.Fatalf("LegacyStateVocabulary(%q) = %q, %v; want %q", source, got, found, want)
		}
	}
}

// TRLC-LINKS: REQ-EMG-139
func TestStripLegacyStateVocabularyKeepsTheRestOfTheDocument(t *testing.T) {
	source := []byte(`schemaVersion: 2
behavior:
  states:
  - id: STATE-A
    name: A
  events:
  - id: EVT-X
    name: X
  relationships:
  - type: transitions_to
    from: STATE-A
    to: STATE-A
  - type: triggered_by
    from: STATE-A
    to: EVT-X
  - type: guarded_by
    from: STATE-A
    to: CTRL-Y
`)
	out, err := StripLegacyStateVocabulary(source)
	if err != nil {
		t.Fatal(err)
	}
	if _, _, found := LegacyStateVocabulary(out); found {
		t.Fatalf("legacy vocabulary survived:\n%s", out)
	}
	text := string(out)
	for _, want := range []string{"EVT-X", "guarded_by", "CTRL-Y"} {
		if !strings.Contains(text, want) {
			t.Errorf("stripped document lost %q:\n%s", want, text)
		}
	}
	for _, gone := range []string{"name: A", "transitions_to", "triggered_by"} {
		if strings.Contains(text, gone) {
			t.Errorf("stripped document still has %q:\n%s", gone, text)
		}
	}
}

// TRLC-LINKS: REQ-EMG-139
func TestStripLegacyStateVocabularyLeavesMachineDocumentsUntouched(t *testing.T) {
	source := []byte("schemaVersion: 2\nbehavior:\n  machines: []\n")
	out, err := StripLegacyStateVocabulary(source)
	if err != nil || string(out) != string(source) {
		t.Fatalf("got %q, %v", out, err)
	}
}
