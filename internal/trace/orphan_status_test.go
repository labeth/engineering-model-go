// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// A retired or rejected requirement is not an orphan; an active one with no
// code, test or delegation is.
// Mutation: dropping the status check reports the retired requirement.
//
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029
func TestOrphanIgnoresRetiredAndRejectedRequirements(t *testing.T) {
	doc := model.RequirementsDocument{Requirements: []model.Requirement{
		{ID: "REQ-A", Status: "implemented"},
		{ID: "REQ-B", Status: "Retired"},
		{ID: "REQ-C", Status: "rejected"},
		{ID: "REQ-D"},
	}}
	diags := validateIntegrityWithIDs(map[string]bool{}, doc, nil, nil, nil)
	got := map[string]bool{}
	for _, d := range diags {
		if d.Code == "requirement.orphan" {
			got[d.Path] = true
		}
	}
	if len(got) != 2 || !got[orphanPath("REQ-A")] || !got[orphanPath("REQ-D")] {
		t.Fatalf("orphans = %v, want REQ-A and REQ-D only", got)
	}
}
