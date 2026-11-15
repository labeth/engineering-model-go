// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// severityRow is the severity of a code at the five tiers as letters, for a
// model with the bundles.
func severityRow(t *testing.T, code string, bundles ...string) string {
	t.Helper()
	var row strings.Builder
	for _, tier := range model.Tiers {
		d := severityAt(t, tierBundle(tier, bundles...), probe(code, "p", SeverityWarning))
		row.WriteString(map[Severity]string{SeverityWarning: "W", SeverityError: "E"}[d.Severity])
	}
	return row.String()
}

// health.file_below_floor (ADR-EMG-087) is a warning at poc, prototype and mvp
// and an error at product and operational, an error from mvp with the safety
// bundle, a warning without a tier, and lifecycle.raise makes it an error at any
// tier. It is tier keyed, not core.
//
// Mutation: another row, or a code that the tier policy does not key, fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestFileBelowFloorRow(t *testing.T) {
	code := model.HealthFileBelowFloorCode
	if got := severityRow(t, code); got != "WWWEE" {
		t.Errorf("%s = %s, want WWWEE", code, got)
	}
	if got := severityRow(t, code, "safety"); got != "WWEEE" {
		t.Errorf("%s with the safety bundle = %s, want WWEEE", code, got)
	}
	if d := severityAt(t, tierBundle(""), probe(code, "p", SeverityWarning)); d.Severity != SeverityWarning {
		t.Errorf("%s without a tier = %+v, want a warning", code, d)
	}
	if !TierKeyed(code) || IsCoreCode(code) {
		t.Errorf("%s must be tier keyed and not core", code)
	}
}

// The code is a gate, like health.regression: a POA&M item can not defer it, and
// it has a remedy that names the fix.
//
// Mutation: leaving the code deferrable or without a remedy fails.
//
// TRLC-LINKS: REQ-EMG-209
func TestFileBelowFloorIsAGateWithARemedy(t *testing.T) {
	code := model.HealthFileBelowFloorCode
	if !IsNonDeferrable(code) {
		t.Errorf("%s must not be deferrable", code)
	}
	if remedy := ConsistencyRemedy(code); !strings.Contains(remedy, "10.0") || !strings.Contains(remedy, "baseline") {
		t.Errorf("remedy = %q, want the fix with the floor and the baseline", remedy)
	}
}
