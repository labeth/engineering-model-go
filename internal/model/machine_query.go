// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

// MachineStates returns the states of every authored machine in document
// order. Machine states are the only states of a model (ADR-EMG-038), so
// consumers that once listed behavior.states list these.
//
// TRLC-LINKS: REQ-EMG-139
func (a AuthoredArchitecture) MachineStates() []MachineState {
	var out []MachineState
	for _, m := range a.Machines {
		out = append(out, m.States...)
	}
	return out
}

// MachineTransitions returns the transitions of every machine in authored order.
// TRLC-LINKS: REQ-EMG-174
// ENGMODEL-LINKS: FU-MODEL-LOADER
func (a AuthoredArchitecture) MachineTransitions() []MachineTransition {
	var out []MachineTransition
	for _, m := range a.Machines {
		out = append(out, m.Transitions...)
	}
	return out
}
