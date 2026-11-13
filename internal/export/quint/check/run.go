// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/export/quint"
)

// apalacheConfigJSON switches off Apalache's deadlock check. A machine with a
// final state has no enabled transition there, which is normal behavior and
// not an invariant violation; run mode reports a machine that deadlocks in its
// initial state as a warning instead.
const apalacheConfigJSON = `{"checker":{"no-deadlock":true}}` + "\n"

// runArgs builds the arguments of quint run. The backend flag is mandatory:
// the default Rust backend downloads its binary at run time.
//
// TRLC-LINKS: REQ-EMG-143
func (c *checker) runArgs(path, module string, extra ...string) []string {
	args := []string{
		"run", path,
		"--backend=" + Backend,
		"--main=" + module,
		fmt.Sprintf("--max-steps=%d", c.opts.MaxSteps),
		fmt.Sprintf("--max-samples=%d", c.opts.MaxSamples),
		"--seed=" + c.opts.Seed,
	}
	return append(args, extra...)
}

// runModule runs the reachability witnesses of one machine and then quint run
// once per invariant, so a violation names one invariant and its requirements.
//
// TRLC-LINKS: REQ-EMG-143
func (c *checker) runModule(m *ModuleResult, file quint.File, path string) error {
	if len(file.Witnesses) > 0 {
		args := append(c.runArgs(path, m.Module), "--witnesses")
		args = append(args, file.Witnesses...)
		out, err := c.call(Command{Path: c.quint, Args: args, Dir: c.scratch, Env: c.env})
		if err != nil {
			return err
		}
		c.recordWitnesses(m, file, out)
	}
	for i, inv := range file.Invariants {
		itf := filepath.Join(c.scratch, fmt.Sprintf("%s.%s.run.itf.json", m.Module, inv.Name))
		args := c.runArgs(path, m.Module, "--invariant="+inv.Name, "--out-itf="+itf)
		out, err := c.call(Command{Path: c.quint, Args: args, Dir: c.scratch, Env: c.env})
		if err != nil {
			return err
		}
		step := stepFromOutput(out, itf)
		if step.Status == StatusPass && step.TraceMax != nil {
			switch *step.TraceMax {
			case 0:
				step.Status = StatusError
				step.Message = "vacuous: the longest trace has length 0, the initial state is unsatisfiable or deadlocked, so nothing was checked"
			case 1:
				m.addWarning(deadlockWarning)
			}
		}
		m.Invariants[i].Run = &step
	}
	return nil
}

const deadlockWarning = "deadlock: no transition is enabled in the initial state, so every trace has length 1 and only the initial state was checked"

// recordWitnesses reads the result of the witnesses run: a vacuous initial
// state is an error, a machine that deadlocks at once and a state that no
// trace reaches are warnings.
//
// TRLC-LINKS: REQ-EMG-143
func (c *checker) recordWitnesses(m *ModuleResult, file quint.File, out Output) {
	p := parseQuintOutput(out)
	if p.Status != StatusPass {
		msg := p.Message
		if msg == "" {
			msg = "unexpected violation"
		}
		m.Errors = append(m.Errors, "witnesses run failed: "+msg)
		return
	}
	if p.TraceMax != nil {
		switch *p.TraceMax {
		case 0:
			m.Errors = append(m.Errors, "vacuous: the longest trace has length 0, the initial state is unsatisfiable or deadlocked")
		case 1:
			m.addWarning(deadlockWarning)
		}
	}
	counts := parseWitnesses(out)
	for i, name := range file.Witnesses {
		state := ""
		if i < len(file.States) {
			state = file.States[i]
		}
		count, ok := counts[name]
		if !ok {
			m.addWarning(fmt.Sprintf("no witness result for state %s (%s)", state, name))
			continue
		}
		m.Witnesses = append(m.Witnesses, WitnessResult{Name: name, State: state, Witnessed: count.Witnessed, Total: count.Total})
		// The initial state holds by construction; Quint's witness count does
		// not include the state a trace starts in, so a machine that never
		// re-enters its initial state would be reported as unreachable.
		if count.Witnessed == 0 && state != file.Initial {
			m.addWarning(fmt.Sprintf("unreachable state %s: %s was witnessed in 0 of %d traces", state, name, count.Total))
		}
	}
}

// stepFromOutput turns the output of one quint call into a StepResult; the
// counterexample of a violation comes from the ITF trace file.
//
// TRLC-LINKS: REQ-EMG-143, REQ-EMG-144
func stepFromOutput(out Output, itfPath string) StepResult {
	p := parseQuintOutput(out)
	step := StepResult{Status: p.Status, ExitCode: p.ExitCode, TraceMin: p.TraceMin, TraceMax: p.TraceMax, Message: p.Message}
	if p.Status == StatusViolated {
		ids, err := counterexampleFromITF(itfPath)
		if err != nil {
			step.Message = "violation without a readable trace: " + err.Error()
			step.TraceUnavailable = true
		}
		step.Counterexample = ids
	}
	return step
}

// verifyModule runs quint verify once per invariant against the pre-provisioned
// Apalache. Each call gets its own free server port, the scratch directory as
// working and temporary directory, and an environment that keeps localhost
// out of any proxy.
//
// TRLC-LINKS: REQ-EMG-144
func (c *checker) verifyModule(m *ModuleResult, path string) error {
	env := verifyEnv(c.verifyBaseEnv, c.opts.QuintHome, c.scratch, c.javaDir)
	for i, inv := range m.Invariants {
		port, err := freePort()
		if err != nil {
			return setupError(err, "allocate a port for the Apalache server")
		}
		itf := filepath.Join(c.scratch, fmt.Sprintf("%s.%s.verify.itf.json", m.Module, inv.Name))
		args := []string{
			"verify", path,
			"--main=" + m.Module,
			"--invariant=" + inv.Name,
			fmt.Sprintf("--max-steps=%d", c.opts.VerifySteps),
			"--out-itf=" + itf,
			fmt.Sprintf("--server-endpoint=localhost:%d", port),
			"--apalache-config=" + c.apalacheConfig,
		}
		out, err := c.call(Command{Path: c.quint, Args: args, Dir: c.scratch, Env: env})
		if err != nil {
			return err
		}
		step := stepFromOutput(out, itf)
		step.TraceMin, step.TraceMax = nil, nil
		m.Invariants[i].Verify = &step
	}
	return nil
}

// FormatCounterexample renders the counterexample of a violated step as
// "TR-A -> TR-B -> TR-C". A violation whose trace could not be read says so
// instead of blaming the initial state, which only a readable trace without
// any transition proves.
//
// TRLC-LINKS: REQ-EMG-143
func FormatCounterexample(step StepResult) string {
	switch {
	case step.TraceUnavailable:
		return "(trace unavailable: the violation has no readable counterexample)"
	case len(step.Counterexample) == 0:
		return "(no transition: the initial state violates the invariant)"
	}
	return strings.Join(step.Counterexample, " -> ")
}
