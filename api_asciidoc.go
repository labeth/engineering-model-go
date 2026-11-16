// ENGMODEL-OWNER-UNIT: FU-ASCIIDOC-GENERATOR
package engmodel

// AsciiDoc architecture document generation.
// Thin facade over internal/asciidoc; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/asciidoc"

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type AsciiDocOptions = asciidoc.Options

// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE, STATE-MODEL-INVALID, EVT-VALIDATION-FAILED
type AsciiDocResult = asciidoc.Result

// AsciiDocInputs are the loaded model documents GenerateAsciiDoc renders.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE, DEP-LOCAL-WORKSPACE
type AsciiDocInputs = asciidoc.Inputs

// AsciiDocFiles are the architecture, requirements and design paths
// GenerateAsciiDocFromFiles loads.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FU-MODEL-LOADER
type AsciiDocFiles = asciidoc.FilePaths

// GenerateAsciiDocFromFiles loads the model files and renders the document.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014
func GenerateAsciiDocFromFiles(files AsciiDocFiles, options AsciiDocOptions) (AsciiDocResult, error) {
	return asciidoc.GenerateFromFiles(files, options)
}

// GenerateAsciiDoc validates and renders the model. With composition it
// resolves the dependencies once, validates against the bundle enriched with
// their published entities, and returns every composition diagnostic; a
// composition error fails the call like any other validation error.
// ENGMODEL-LINKS: FU-ASCIIDOC-GENERATOR, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS
// TRLC-LINKS: REQ-EMG-001, REQ-EMG-003, REQ-EMG-014, REQ-EMG-035, REQ-EMG-036, REQ-EMG-065, REQ-EMG-066
func GenerateAsciiDoc(in AsciiDocInputs, options AsciiDocOptions) (AsciiDocResult, error) {
	return asciidoc.Generate(in, options)
}
