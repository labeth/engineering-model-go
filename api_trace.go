// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel

// Requirement, code and test traceability.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

// TraceMatrix is the consolidated, machine-readable traceability matrix for one model.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type TraceMatrix = engine.TraceMatrix

// TraceRow is one requirement and everything that traces to it.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type TraceRow = engine.TraceRow

// TraceProvenance identifies the selected dependency publication that supplied
// an imported requirement.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type TraceProvenance = engine.TraceProvenance

// TraceDelegatedStatus is the trace status of a delegation's target
// requirement computed in the dependency itself. Status is "implemented",
// "test-linked", "delegated" or "orphan" as in that repository's own matrix,
// "not-available" when the dependency has no code sources, and
// "unknown-requirement" when the target requirement does not exist there.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
type TraceDelegatedStatus = engine.TraceDelegatedStatus

// TraceCodeRef is a code symbol that implements a requirement.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type TraceCodeRef = engine.TraceCodeRef

// TraceDelegation is the downward allocation of a requirement onto a subsystem.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type TraceDelegation = engine.TraceDelegation

// DanglingLink is a code trace link that resolves to nothing in the model.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
type DanglingLink = engine.DanglingLink

// TraceSummary is the headline coverage roll-up.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type TraceSummary = engine.TraceSummary

// TraceOptions configures BuildTraceMatrixFromFilesWith.
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064, REQ-EMG-066
type TraceOptions = engine.TraceOptions

// BuildTraceMatrixFromFiles loads a model, scans its code root, scopes the code to the
// model, validates trace integrity, and returns the matrix together with diagnostics.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-050, REQ-EMG-051
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func BuildTraceMatrixFromFiles(modelPath, requirementsPath, codeRoot string) (TraceMatrix, []Diagnostic, error) {
	return engine.BuildTraceMatrixFromFiles(modelPath, requirementsPath, codeRoot)
}

// BuildTraceMatrixFromFilesWith is BuildTraceMatrixFromFiles with explicit
// resolution options. With composition it composes once, resolves code
// ENGMODEL-LINKS alias::ID against the dependencies' published entities
// (requirements stay local), returns the composition diagnostics, and reports
// each delegated requirement's own status in its target repository.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-050, REQ-EMG-051, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-MODEL-LOADER, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func BuildTraceMatrixFromFilesWith(modelPath, requirementsPath, codeRoot string, options TraceOptions) (TraceMatrix, []Diagnostic, error) {
	return engine.BuildTraceMatrixFromFilesWith(modelPath, requirementsPath, codeRoot, options)
}
