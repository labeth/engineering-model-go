// ENGMODEL-OWNER-UNIT: FU-LOBSTER-EXPORTER
package engmodel

// LOBSTER activity trace export.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-TRLC-EXPORTER
type LobsterActivityExportOptions = engine.LobsterActivityExportOptions

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type LobsterActivityExportResult = engine.LobsterActivityExportResult

var ErrNoLobsterActivities = engine.ErrNoLobsterActivities

// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-TRLC-EXPORTER, FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func GenerateLobsterActivityTraceFromDir(testsDir string, options LobsterActivityExportOptions) (LobsterActivityExportResult, error) {
	return engine.GenerateLobsterActivityTraceFromDir(testsDir, options)
}
