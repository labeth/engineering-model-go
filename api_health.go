// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package engmodel

// Code health: the scan, the report engmod-health/1, its import and the SARIF
// export. Thin facade over internal/health and internal/health/scan;
// TestFacadeReexportsEngine keeps it complete. The scores are this project's
// own and are not CodeScene scores.

import (
	"fmt"
	"io"
	"strings"

	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/health/scan"
	"github.com/labeth/engineering-model-go/internal/model"
)

// HealthReport is the code health report, schema engmod-health/1.
type HealthReport = health.Report

// HealthFileReport is one scored source file of a report.
type HealthFileReport = health.FileReport

// HealthFunction is one function with its metrics and findings.
type HealthFunction = health.Function

// HealthFinding is one smell found in one function.
type HealthFinding = health.Finding

// HealthDeduction is one finding's contribution to a file score.
type HealthDeduction = health.Deduction

// HealthWaivedFinding is a finding moved out of its function by a waiver.
type HealthWaivedFinding = health.WaivedFinding

// HealthDiagnostic is a finding about the analysis itself, such as an unused
// waiver or a file that was not analysed.
type HealthDiagnostic = health.Diagnostic

// HealthRollups holds the project, functional unit, requirement and test
// roll-ups of a report.
type HealthRollups = health.Rollups

// HealthAggregate is the roll-up of a set of files.
type HealthAggregate = health.Aggregate

// HealthSummary counts the findings of a report by level.
type HealthSummary = health.Summary

// HealthRankedFunction is a function with the points its findings deduct.
type HealthRankedFunction = health.RankedFunction

// HealthScanOptions select what ScanHealth analyses.
type HealthScanOptions = scan.Options

// HealthReportSchema is the schema string of the report.
const HealthReportSchema = health.ReportSchema

// HealthScoringVersion is the version of the weights table of the score.
const HealthScoringVersion = health.ScoringVersion

// ScanHealth analyses the code of the model named by the manifest and returns
// the report. A setup error (the model does not load, an unknown tier, a path
// outside the project) wraps ErrHealthSetup.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-151, REQ-EMG-152, REQ-EMG-157, REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func ScanHealth(opts HealthScanOptions) (HealthReport, error) {
	return scan.Scan(opts)
}

// ErrHealthSetup marks a setup error of ScanHealth.
var ErrHealthSetup = scan.ErrSetup

// ImportHealthReport loads and validates a report of another engine that
// conforms to engmod-health/1. A document that does not conform is an error
// wrapping ErrInvalidHealthReport.
//
// TRLC-LINKS: REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func ImportHealthReport(r io.Reader) (HealthReport, error) {
	return health.Import(r)
}

// ErrInvalidHealthReport marks a document that is no conforming
// engmod-health/1 report.
var ErrInvalidHealthReport = health.ErrInvalidReport

// EncodeHealthReport renders a report as canonical JSON.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func EncodeHealthReport(r HealthReport) ([]byte, error) {
	return health.Encode(r)
}

// HealthSARIF renders a report as SARIF 2.1.0.
//
// TRLC-LINKS: REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func HealthSARIF(r HealthReport) ([]byte, error) {
	return health.SARIF(r)
}

// HealthReportSchemaJSON returns the JSON Schema document of engmod-health/1.
//
// TRLC-LINKS: REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH
func HealthReportSchemaJSON() []byte {
	return health.ReportSchemaJSON()
}

// WorstHealthFunctions returns the n functions of production files whose
// findings deduct the most points.
//
// TRLC-LINKS: REQ-EMG-152
// ENGMODEL-LINKS: FU-CODE-HEALTH
func WorstHealthFunctions(r HealthReport, n int) []HealthRankedFunction {
	return health.WorstFunctions(r, n)
}

// RelevelHealthReport reads a report at another tier: the levels of the
// codes (error or warning) come from the model named by modelPath, replaced
// by tier when that is not empty, or from the tier alone when modelPath is
// empty. The scores and findings are unchanged.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func RelevelHealthReport(r HealthReport, modelPath, tier string) (HealthReport, error) {
	var bundle model.Bundle
	if strings.TrimSpace(modelPath) != "" {
		var err error
		if bundle, err = model.LoadBundle(modelPath); err != nil {
			return HealthReport{}, fmt.Errorf("%w: load model: %v", ErrHealthSetup, err)
		}
	}
	if tier = strings.ToLower(strings.TrimSpace(tier)); tier != "" {
		if model.TierRank(tier) == 0 {
			return HealthReport{}, fmt.Errorf("%w: unknown tier %q; use one of %s", ErrHealthSetup, tier, strings.Join(model.Tiers, ", "))
		}
		bundle = health.WithTier(bundle, tier)
	}
	return health.Relevel(r, bundle), nil
}

// HealthCheckOptions select what CheckHealth compares.
type HealthCheckOptions = scan.CheckOptions

// HealthCheckReport is the result of CheckHealth: the findings of the changed
// files classified against a base revision, the level of health.regression at
// the tier and the verdict.
type HealthCheckReport = health.CheckReport

// HealthCheckCounts counts the findings of a check by change class.
type HealthCheckCounts = health.CheckCounts

// HealthDeltaEntry is one finding of a changed file with its change class
// (new, worsened or existing).
type HealthDeltaEntry = health.DeltaEntry

// HealthSkippedFile is a changed file that a check did not analyse (too large
// or not a regular file); it fails the check where a regression does.
type HealthSkippedFile = health.SkippedFile

// HealthPolicyChange is one way in which the health policy of the head is
// weaker than that of the base revision (a higher limit, a new exclude, a lower
// tier); see health.policy_weakened.
type HealthPolicyChange = health.PolicyChange

// HealthFileDelta is the score of one changed file before and after.
type HealthFileDelta = health.FileDelta

// CheckHealth compares the changed files of the model's project with a base
// revision (HEAD by default: the uncommitted work) and classifies each finding
// as new, worsened or existing; the report says whether a regression (a new or
// worsened high finding), a changed file that was skipped, or a weakening of the
// health policy against the base revision that no policyDecision records, is
// an error at the tier of the model (Failed). A problem of the setup (the model does not load, an unknown tier, a path
// outside the project, no git work tree, an unavailable or invalid revision)
// is an error that wraps ErrHealthSetup, ErrHealthBaseUnavailable,
// ErrHealthInvalidRev, ErrHealthInvalidPath or ErrHealthNotGitRepo, never a
// result.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156, REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func CheckHealth(opts HealthCheckOptions) (HealthCheckReport, error) {
	return scan.Check(opts)
}

// Errors of CheckHealth besides ErrHealthSetup.
var (
	// ErrHealthBaseUnavailable: the base revision is unknown or, in a shallow
	// clone, not fetched.
	ErrHealthBaseUnavailable = health.ErrBaseUnavailable
	// ErrHealthInvalidRev: a revision that is not safe to hand to git.
	ErrHealthInvalidRev = health.ErrInvalidRev
	// ErrHealthInvalidPath: a path that is not a clean repository-relative path.
	ErrHealthInvalidPath = health.ErrInvalidPath
	// ErrHealthNotGitRepo: the project is not inside a git work tree.
	ErrHealthNotGitRepo = health.ErrNotGitRepo
)

// EncodeHealthCheck renders a check report as canonical JSON: the delta report
// plus the levels, the counts and the verdict.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158, REQ-EMG-159
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func EncodeHealthCheck(r HealthCheckReport) ([]byte, error) {
	return health.EncodeCheck(r)
}

// HealthCheckSARIF renders the regressions of a check as SARIF 2.1.0 (rule
// health.regression, the smell code in the properties of each result).
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func HealthCheckSARIF(r HealthCheckReport) ([]byte, error) {
	return health.CheckSARIF(r)
}
