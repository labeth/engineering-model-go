// ENGMODEL-OWNER-UNIT: FU-SYSML-EXPORTER
package engmodel

// SysML v2 textual export, coverage and KPAR interchange.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

type SysMLCoverageManifest = engine.SysMLCoverageManifest

type SysMLCoverageAuthority = engine.SysMLCoverageAuthority

type SysMLCoverageTotals = engine.SysMLCoverageTotals

type SysMLMetaclassCoverage = engine.SysMLMetaclassCoverage

type SysMLPropertyCoverage = engine.SysMLPropertyCoverage

type SysMLRelationshipCoverage = engine.SysMLRelationshipCoverage

type SysMLExtensionCoverage = engine.SysMLExtensionCoverage

type SysMLCoverageCheckEvidence = engine.SysMLCoverageCheckEvidence

// SysMLExportResult is a deterministic textual projection plus explicit
// projection and conformance diagnostics.
type SysMLExportResult = engine.SysMLExportResult

type SysMLRendererCoverage = engine.SysMLRendererCoverage

type SysMLRendererEvidence = engine.SysMLRendererEvidence

// SysMLRoundTripResult captures the normative KPAR path and reopened native
// SysML source used for source-integrity comparison.
type SysMLRoundTripResult = engine.SysMLRoundTripResult

// SysMLV2Coverage returns metaclass/property-level evidence generated from the
// pinned official inventory and strict binding.
//
// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func SysMLV2Coverage() SysMLCoverageManifest {
	return engine.SysMLV2Coverage()
}

// ValidateSysMLV2Coverage rejects any gap or stale/tampered evidence.
//
// TRLC-LINKS: REQ-EMG-039, REQ-EMG-040
func ValidateSysMLV2Coverage(manifest SysMLCoverageManifest) error {
	return engine.ValidateSysMLV2Coverage(manifest)
}

// GenerateSysMLV2FromFile loads the canonical YAML input and generates a SysML
// v2 textual projection.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036, REQ-EMG-037, REQ-EMG-047, REQ-EMG-050
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN
func GenerateSysMLV2FromFile(architecturePath string) (SysMLExportResult, error) {
	return engine.GenerateSysMLV2FromFile(architecturePath)
}

// GenerateSysMLV2 validates and projects a Bundle before rendering.
//
// TRLC-LINKS: REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func GenerateSysMLV2(bundle Bundle) (SysMLExportResult, error) {
	return engine.GenerateSysMLV2(bundle)
}

// GenerateSysMLV2Projection renders a pre-built semantic graph. Missing
// rendering rules are errors; official concepts are never replaced by a
// generic item declaration.
//
// TRLC-LINKS: REQ-EMG-036
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, DO-CANONICAL-SEMANTIC-MODEL, DO-SYSML-V2-MODEL
func GenerateSysMLV2Projection(semantic SemanticModel) (string, []SemanticDiagnostic) {
	return engine.GenerateSysMLV2Projection(semantic)
}

// ExportSysMLV2Project creates a Sysand interchange project and builds its KPAR.
// Sysand, rather than a custom ZIP writer, owns all normative KPAR packaging.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
// ENGMODEL-LINKS: FU-SYSML-EXPORTER, IF-CLI-ENGSYSML, DO-SYSML-V2-MODEL, REF-SYSML-V2-TOOLCHAIN
func ExportSysMLV2Project(architecturePath, projectDir, kparPath, sysandPath string) (SysMLExportResult, error) {
	return engine.ExportSysMLV2Project(architecturePath, projectDir, kparPath, sysandPath)
}

// ReopenSysMLV2Project clones a KPAR with Sysand and returns its native SysML
// project source.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func ReopenSysMLV2Project(kparPath, reopenedDir, sysandPath string) (SysMLRoundTripResult, error) {
	return engine.ReopenSysMLV2Project(kparPath, reopenedDir, sysandPath)
}

// VerifySysMLV2RoundTrip compares the reopened KPAR source to a fresh native
// SysML projection of the authored Engineering Model.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func VerifySysMLV2RoundTrip(architecturePath, kparPath, reopenedDir, sysandPath string) (SysMLRoundTripResult, error) {
	return engine.VerifySysMLV2RoundTrip(architecturePath, kparPath, reopenedDir, sysandPath)
}

// ImportSysMLV2Project reconstructs the canonical semantic model from legacy
// projects that embedded an Engineering Model payload. Native projections do
// not embed non-SysML canonical data.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func ImportSysMLV2Project(sourcePath string) (SemanticModel, error) {
	return engine.ImportSysMLV2Project(sourcePath)
}

// CompareSysMLV2RoundTrip reports differences in stable identity, ownership,
// relationships, expressions, references, and typed extensions.
//
// TRLC-LINKS: REQ-EMG-036, REQ-EMG-037
func CompareSysMLV2RoundTrip(expected, actual SemanticModel) []string {
	return engine.CompareSysMLV2RoundTrip(expected, actual)
}
