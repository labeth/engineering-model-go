// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package engmodel

// Quint specification export: one deterministic module per behavior machine.
// Thin facade over internal/export/quint; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/export/quint"

// QuintExportResult holds the generated Quint modules and the diagnostics
// found while generating them.
type QuintExportResult = quint.Result

// QuintFile is one generated Quint module with the machine, invariant and
// witness names a checker needs.
type QuintFile = quint.File

// QuintInvariant is one invariant val of a generated module and the
// requirements it protects.
type QuintInvariant = quint.Invariant

// QuintGeneratedHeader starts the first line of every generated Quint module;
// the machine ID follows. Tools use it to tell generated files from
// hand-written ones.
const QuintGeneratedHeader = quint.GeneratedHeader

// GenerateQuint renders one Quint module for each machine of the bundle. A
// parse error or an unresolved reference fails generation with diagnostics
// and no file for that machine.
//
// TRLC-LINKS: REQ-EMG-141, REQ-EMG-142
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func GenerateQuint(bundle Bundle) (QuintExportResult, error) {
	return quint.Generate(bundle)
}

// GenerateQuintFromFile loads the model named by its engmod.yml manifest and
// generates its Quint modules.
//
// TRLC-LINKS: REQ-EMG-141, REQ-EMG-142
// ENGMODEL-LINKS: FU-QUINT-EXPORTER, IF-CLI-ENGQUINT
func GenerateQuintFromFile(manifestPath string) (QuintExportResult, error) {
	return quint.GenerateFromFile(manifestPath)
}
