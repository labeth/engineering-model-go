// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package engmodel

// Airborne assurance package generation.
// Thin facade over internal/engine; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/engine"

type AirborneAssuranceOptions = engine.AirborneAssuranceOptions

type AirborneAssuranceResult = engine.AirborneAssuranceResult

const AirborneDisclaimer = engine.AirborneDisclaimer

var ErrNoAviationProfile = engine.ErrNoAviationProfile

// GenerateAirborneAssuranceFromFile validates and renders the complete
// evidence-readiness package. It does not determine compliance or certification.
// TRLC-LINKS: REQ-EMG-053, REQ-EMG-054, REQ-EMG-055
func GenerateAirborneAssuranceFromFile(manifestPath string, options AirborneAssuranceOptions) (AirborneAssuranceResult, error) {
	return engine.GenerateAirborneAssuranceFromFile(manifestPath, options)
}
