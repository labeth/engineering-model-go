// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package exportutil

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// An explicit timestamp wins and is normalized to UTC RFC3339; otherwise the
// latest authored decision or verification date is used, so regeneration is
// reproducible without a wall clock.
// TRLC-LINKS: REQ-EMG-012
func TestResolveGeneratedTimestamp(t *testing.T) {
	var bundle model.Bundle
	bundle.Decisions.Decisions = []model.Decision{{Date: "2026-03-01"}, {Date: "2026-05-02"}}
	bundle.Architecture.AuthoredArchitecture.ControlVerifications = []model.ControlVerification{{LastTested: "2026-04-01T12:00:00+02:00"}}

	got, err := ResolveGeneratedTimestamp(bundle, "", "test")
	if err != nil || got != "2026-05-02T00:00:00Z" {
		t.Fatalf("authored latest = %q, %v; want 2026-05-02T00:00:00Z", got, err)
	}
	got, err = ResolveGeneratedTimestamp(bundle, "2026-09-19T10:00:00+02:00", "test")
	if err != nil || got != "2026-09-19T08:00:00Z" {
		t.Fatalf("explicit = %q, %v; want 2026-09-19T08:00:00Z", got, err)
	}
	if _, err := ResolveGeneratedTimestamp(bundle, "yesterday", "test"); err == nil {
		t.Fatal("accepted an invalid explicit timestamp")
	}
	if _, err := ResolveGeneratedTimestamp(model.Bundle{}, "", "test"); err == nil {
		t.Fatal("accepted a model without any authored date")
	}
	if got := NormalizeGeneratedTimestamp("2026-01-02"); got != "2026-01-02T00:00:00Z" {
		t.Fatalf("NormalizeGeneratedTimestamp(date) = %q", got)
	}
}
