// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package engmodel

// Airborne assurance package generation.
// Thin facade over internal/export/airborne; TestFacadeReexportsEngine keeps it complete.

import "github.com/labeth/engineering-model-go/internal/export/airborne"

type AirborneAssuranceOptions = airborne.Options

type AirborneAssuranceResult = airborne.Result

const AirborneDisclaimer = airborne.Disclaimer

var ErrNoAviationProfile = airborne.ErrNoAviationProfile

// GenerateAirborneAssuranceFromFile validates and renders the complete
// evidence-readiness package. It does not determine compliance or certification.
// TRLC-LINKS: REQ-EMG-053, REQ-EMG-054, REQ-EMG-055
func GenerateAirborneAssuranceFromFile(manifestPath string, options AirborneAssuranceOptions) (AirborneAssuranceResult, error) {
	return airborne.GenerateFromFile(manifestPath, options)
}
