// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package engmodel_test

import (
	"fmt"
	"path/filepath"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
)

// unknownIDFindings are the requirement.invalid_applies_to findings of a bundle, sorted.
func unknownIDFindings(bundle engmodel.Bundle) []engmodel.Diagnostic {
	var found []engmodel.Diagnostic
	for _, d := range engmodel.SortDiagnostics(engmodel.ValidateBundle(bundle)) {
		if d.Code == "requirement.invalid_applies_to" {
			found = append(found, d)
		}
	}
	return found
}

// withAppliesTo is the bundle with REQ-A-001 applying to the given IDs.
func withAppliesTo(bundle engmodel.Bundle, ids []string) engmodel.Bundle {
	bundle.Requirements.Requirements = append([]engmodel.Requirement(nil), bundle.Requirements.Requirements...)
	bundle.Requirements.Requirements[0].AppliesTo = ids
	return bundle
}

// ValidateBundle returns, for each finding, a stable code, a severity, a message and the model path of the finding;
// the bundle of a valid model has no error and one requirement that applies to an unknown ID has exactly that finding.
// Mutation: a renamed code, a lower severity, a lost path or message, or a check that flags a known ID fails this test.
// TRLC-LINKS: REQ-EMG-195
func TestValidateBundleReturnsCodeSeverityMessageAndPathOfEachFinding(t *testing.T) {
	isolatedModel(t)
	bundle, err := engmodel.LoadBundle(filepath.Join(childA, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	expect(t, !engmodel.HasErrors(engmodel.ValidateBundle(bundle)), "the valid model has errors")
	cases := map[string][]string{
		"unknown unit":                  {"FU-DOES-NOT-EXIST"},
		"known unit and an unknown one": {"FU-ENCODER", "FU-DOES-NOT-EXIST"},
		"empty ID":                      {""},
	}
	for name, ids := range cases {
		t.Run(name, func(t *testing.T) {
			found := unknownIDFindings(withAppliesTo(bundle, ids))
			if len(found) != 1 {
				t.Fatalf("%d findings, want 1: %+v", len(found), found)
			}
			unknown, d := ids[len(ids)-1], found[0]
			check{t}.equal("severity and path", [2]string{string(d.Severity), d.Path}, [2]string{"error", fmt.Sprintf("requirements[0].appliesTo[%d]", len(ids)-1)})
			expect(t, strings.Contains(d.Message, "REQ-A-001") && strings.Contains(d.Message, unknown), "message %q names REQ-A-001 and %q", d.Message, unknown)
		})
	}
}

// A known unit and a qualified ID of a dependency are no finding.
// Mutation: a check that flags a known ID fails this test.
// TRLC-LINKS: REQ-EMG-195
func TestValidateBundleAcceptsKnownAndQualifiedIDs(t *testing.T) {
	isolatedModel(t)
	bundle, err := engmodel.LoadBundle(filepath.Join(childA, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, ids := range [][]string{{"FU-ENCODER"}, {"encoder::FU-ENCODER"}} {
		check{t}.equal(fmt.Sprintf("findings for %v", ids), len(unknownIDFindings(withAppliesTo(bundle, ids))), 0)
	}
}
