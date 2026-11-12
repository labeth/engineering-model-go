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

// StripLegacyStateVocabulary returns a behavior document source without the
// state vocabulary retired in v0.11.0: the behavior.states list and the
// behavior.relationships of type transitions_to or triggered_by. It exists
// for tools that read the contract of a model committed before v0.11.0
// (contract-diff, release-check): the contract is requirements and
// interfaces, never state behavior, and the current loader refuses the
// retired vocabulary in a model that is being worked on. A source without the
// vocabulary is returned unchanged.
//
// TRLC-LINKS: REQ-EMG-139
func StripLegacyStateVocabulary(source []byte) ([]byte, error) {
	if _, _, found := LegacyStateVocabulary(source); !found {
		return source, nil
	}
	var root yaml.Node
	if err := yaml.Unmarshal(source, &root); err != nil {
		return nil, err
	}
	if root.Kind != yaml.DocumentNode || len(root.Content) != 1 {
		return source, nil
	}
	behavior := mappingValue(root.Content[0], "behavior")
	if behavior == nil {
		return source, nil
	}
	removeMappingKey(behavior, "states")
	if relationships := mappingValue(behavior, "relationships"); relationships != nil && relationships.Kind == yaml.SequenceNode {
		kept := relationships.Content[:0]
		for _, item := range relationships.Content {
			if typeNode := mappingValue(item, "type"); typeNode != nil && legacyStateMappingTypes[typeNode.Value] {
				continue
			}
			kept = append(kept, item)
		}
		relationships.Content = kept
	}
	var out bytes.Buffer
	encoder := yaml.NewEncoder(&out)
	encoder.SetIndent(2)
	if err := encoder.Encode(&root); err != nil {
		return nil, err
	}
	if err := encoder.Close(); err != nil {
		return nil, err
	}
	return out.Bytes(), nil
}

// mappingValue returns the value node of key in a mapping node, or nil.
//
// TRLC-LINKS: REQ-EMG-139
func mappingValue(node *yaml.Node, key string) *yaml.Node {
	if node == nil || node.Kind != yaml.MappingNode {
		return nil
	}
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1]
		}
	}
	return nil
}

// removeMappingKey deletes key and its value from a mapping node.
//
// TRLC-LINKS: REQ-EMG-139
func removeMappingKey(node *yaml.Node, key string) {
	for i := 0; i+1 < len(node.Content); i += 2 {
		if node.Content[i].Value == key {
			node.Content = append(node.Content[:i], node.Content[i+2:]...)
			return
		}
	}
}
