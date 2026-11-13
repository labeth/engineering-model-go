// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"strings"
	"testing"
)

const linkedBehaviorYAML = `schemaVersion: 3
behavior:
  events:
    - {id: EVT-A, name: Requested, requirements: [REQ-A, REQ-B]}
  machines:
    - id: SM-PAY
      owner: FU-A
      initial: STATE-PAY-IDLE
      states:
        - {id: STATE-PAY-IDLE, name: Idle, requirements: [REQ-A]}
        - {id: STATE-PAY-DONE, name: Done, final: true}
      transitions:
        - {id: TR-PAY-001, from: STATE-PAY-IDLE, to: STATE-PAY-DONE, on: EVT-A, requirements: [REQ-B]}
`

// A state, a transition and an event load their optional requirements list,
// and an element without the field loads with none.
// Mutation: removing the requirements field from a Go struct or the CUE
// definition fails the load or the decoded value.
// TRLC-LINKS: REQ-EMG-177
func TestLoadBundleCarriesElementRequirements(t *testing.T) {
	bundle, err := loadMachineFixture(t, linkedBehaviorYAML)
	if err != nil {
		t.Fatal(err)
	}
	a := bundle.Architecture.AuthoredArchitecture
	m := a.Machines[0]
	got := strings.Join(m.States[0].Requirements, ",") + "|" + strings.Join(m.Transitions[0].Requirements, ",") + "|" + strings.Join(a.Events[0].Requirements, ",")
	if got != "REQ-A|REQ-B|REQ-A,REQ-B" {
		t.Fatalf("requirements decoded as %q", got)
	}
	if len(m.States[1].Requirements) != 0 {
		t.Fatalf("a state without the field must list none: %+v", m.States[1])
	}
}

// The requirements field is a list of strings on states, transitions and
// events, and the definitions stay closed to other keys.
// Mutation: opening an element definition to any key, or typing the field as
// a plain string, fails a row.
// TRLC-LINKS: REQ-EMG-177
func TestLoadBundleRejectsInvalidElementRequirements(t *testing.T) {
	cases := map[string]struct{ old, new, want string }{
		"state requirements not a list":      {"name: Idle, requirements: [REQ-A]", "name: Idle, requirements: REQ-A", "requirements"},
		"state requirement not a string":     {"name: Idle, requirements: [REQ-A]", "name: Idle, requirements: [1]", "requirements"},
		"transition requirements not a list": {"requirements: [REQ-B]}", "requirements: REQ-B}", "requirements"},
		"event requirements not a list":      {"requirements: [REQ-A, REQ-B]", "requirements: REQ-A", "requirements"},
		"event requirement not a string":     {"requirements: [REQ-A, REQ-B]", "requirements: [{x: 1}]", "requirements"},
		"unknown state key":                  {"name: Idle, requirements", "name: Idle, covers: [REQ-A], requirements", "covers"},
		"unknown transition key":             {"on: EVT-A, requirements", "on: EVT-A, covers: [REQ-A], requirements", "covers"},
		"unknown event key":                  {"name: Requested, requirements", "name: Requested, covers: [REQ-A], requirements", "covers"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			if !strings.Contains(linkedBehaviorYAML, c.old) {
				t.Fatalf("fixture lacks %q", c.old)
			}
			_, err := loadMachineFixture(t, strings.Replace(linkedBehaviorYAML, c.old, c.new, 1))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected rejection mentioning %q, got %v", c.want, err)
			}
		})
	}
}

// An event without requirements projects the same semantic metadata as
// before the field existed, and one with requirements carries them.
// Mutation: dropping json omitempty from Event.Requirements adds a null
// Requirements property to every event.
// TRLC-LINKS: REQ-EMG-177
func TestEventRequirementsLeaveUnsetMetadataUnchanged(t *testing.T) {
	cases := map[string]struct {
		requirements []string
		want         bool
	}{"unset": {nil, false}, "set": {[]string{"REQ-A"}, true}}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			var b Bundle
			b.Architecture.AuthoredArchitecture.Events = []Event{{ID: "EVT-A", Name: "A", Requirements: c.requirements}}
			semantic, _ := ProjectSemanticModel(b)
			_, has := semantic.Elements[0].Metadata[0].Properties["Requirements"]
			if has != c.want {
				t.Fatalf("Requirements property present = %v, want %v", has, c.want)
			}
		})
	}
}
