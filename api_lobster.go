// ENGMODEL-OWNER-UNIT: FU-LOBSTER-EXPORTER
package engmodel

// LOBSTER activity trace export.
// Thin facade over internal/export/lobster; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/export/lobster"

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FU-TRLC-EXPORTER
type LobsterActivityExportOptions = lobster.Options

// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type LobsterActivityExportResult = lobster.Result

var ErrNoLobsterActivities = lobster.ErrNoActivities

// TRLC-LINKS: REQ-EMG-006, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-LOBSTER-EXPORTER, CTRL-TRACEABILITY-COVERAGE, FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS, FU-TRLC-EXPORTER, FU-CODEMAP-INFERENCE, DEP-LOCAL-WORKSPACE
func GenerateLobsterActivityTraceFromDir(testsDir string, options LobsterActivityExportOptions) (LobsterActivityExportResult, error) {
	return lobster.GenerateActivityTraceFromDir(testsDir, options)
}
