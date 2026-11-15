// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package exportutil

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

func timestampBundle() model.Bundle {
	var bundle model.Bundle
	bundle.Decisions.Decisions = []model.Decision{{Date: "2026-03-01"}, {Date: "2026-05-02"}}
	bundle.Architecture.AuthoredArchitecture.ControlVerifications = []model.ControlVerification{{LastTested: "2026-04-01T12:00:00+02:00"}}
	return bundle
}

// The latest authored decision or verification date is used, so regeneration
// is reproducible without a wall clock.
// TRLC-LINKS: REQ-EMG-012
func TestResolveGeneratedTimestampAuthoredLatest(t *testing.T) {
	got, err := ResolveGeneratedTimestamp(timestampBundle(), "", "test")
	if err != nil || got != "2026-05-02T00:00:00Z" {
		t.Fatalf("authored latest = %q, %v; want 2026-05-02T00:00:00Z", got, err)
	}
}

// An explicit timestamp wins and is normalized to UTC RFC3339.
// TRLC-LINKS: REQ-EMG-012
func TestResolveGeneratedTimestampExplicit(t *testing.T) {
	got, err := ResolveGeneratedTimestamp(timestampBundle(), "2026-09-19T10:00:00+02:00", "test")
	if err != nil || got != "2026-09-19T08:00:00Z" {
		t.Fatalf("explicit = %q, %v; want 2026-09-19T08:00:00Z", got, err)
	}
}

// TRLC-LINKS: REQ-EMG-012
func TestResolveGeneratedTimestampRejections(t *testing.T) {
	if _, err := ResolveGeneratedTimestamp(timestampBundle(), "yesterday", "test"); err == nil {
		t.Fatal("accepted an invalid explicit timestamp")
	}
	if _, err := ResolveGeneratedTimestamp(model.Bundle{}, "", "test"); err == nil {
		t.Fatal("accepted a model without any authored date")
	}
}

// TRLC-LINKS: REQ-EMG-012
func TestNormalizeGeneratedTimestampDate(t *testing.T) {
	if got := NormalizeGeneratedTimestamp("2026-01-02"); got != "2026-01-02T00:00:00Z" {
		t.Fatalf("NormalizeGeneratedTimestamp(date) = %q", got)
	}
}
