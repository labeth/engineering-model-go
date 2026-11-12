// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"bytes"
	"fmt"

	"gopkg.in/yaml.v3"
)

// LegacyStateRemovedMessage is the tail of every diagnostic for the state
// vocabulary retired in v0.11.0 (ADR-EMG-038).
const LegacyStateRemovedMessage = "was removed in v0.11.0: model state behavior with behavior.machines (see ADR-EMG-038)"

// legacyStateMappingTypes are the relationship types that only described
// free-floating states: transitions_to (state -> state) and triggered_by
// (state -> event, handled in a state). guarded_by is not listed; it stays.
var legacyStateMappingTypes = map[string]bool{"transitions_to": true, "triggered_by": true}

// LegacyStateVocabulary reports the first use of the removed state
// vocabulary in a behavior document source: a behavior.states list, or a
// behavior.relationships entry of type transitions_to or triggered_by. The
// returned path names the offending field, for example "behavior.states" or
// "behavior.relationships[2].type"; found is false for a document that
// models state behavior with machines only or that cannot be parsed (the
// strict decoder reports parse errors).
//
// TRLC-LINKS: REQ-EMG-139
func LegacyStateVocabulary(source []byte) (path, mappingType string, found bool) {
	var document struct {
		Behavior struct {
			States        yaml.Node `yaml:"states"`
			Relationships []struct {
				Type string `yaml:"type"`
			} `yaml:"relationships"`
		} `yaml:"behavior"`
	}
	if err := yaml.NewDecoder(bytes.NewReader(source)).Decode(&document); err != nil {
		return "", "", false
	}
	if !document.Behavior.States.IsZero() {
		return "behavior.states", "", true
	}
	for i, relationship := range document.Behavior.Relationships {
		if legacyStateMappingTypes[relationship.Type] {
			return fmt.Sprintf("behavior.relationships[%d].type", i), relationship.Type, true
		}
	}
	return "", "", false
}

// rejectLegacyStateVocabulary turns the removed state vocabulary of a
// behavior document into one precise error before the strict decoder reports
// an unknown field or an unsupported relationship kind.
//
// TRLC-LINKS: REQ-EMG-139
func rejectLegacyStateVocabulary(path string, source []byte, out any) error {
	if _, ok := out.(*BehaviorDocument); !ok {
		return nil
	}
	field, mappingType, found := LegacyStateVocabulary(source)
	if !found {
		return nil
	}
	if mappingType != "" {
		return fmt.Errorf("%s: %s (%s) %s", path, field, mappingType, LegacyStateRemovedMessage)
	}
	return fmt.Errorf("%s: %s %s", path, field, LegacyStateRemovedMessage)
}
