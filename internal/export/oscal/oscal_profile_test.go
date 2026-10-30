package oscal

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// TRLC-LINKS: REQ-EMG-013
func TestGenerateOSCALProfileImportsEveryAuthoredProfile(t *testing.T) {
	bundle, err := model.LoadBundle("examples/atlas-industries/repos/shared-compliance-baseline/engmod.yml")
	if err != nil {
		t.Fatal(err)
	}
	doc, err := GenerateProfile(bundle, ProfileOptions{LastModified: "2026-09-19T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	for _, href := range []string{"profiles/iso-27001-atlas.json", "profiles/nist-800-53-atlas.json"} {
		if !strings.Contains(doc, `"href": "`+href+`"`) {
			t.Fatalf("expected aggregate profile to import %s:\n%s", href, doc)
		}
	}
}

// TRLC-LINKS: REQ-EMG-013
func TestGenerateOSCALSSPUsesPackagedAggregateProfile(t *testing.T) {
	bundle, err := model.LoadBundle("examples/atlas-industries/repos/shared-compliance-baseline/engmod.yml")
	if err != nil {
		t.Fatal(err)
	}
	res, err := GenerateSSP(bundle, SSPOptions{
		ImportProfileHref: "./OSCAL-PROFILE.json",
		LastModified:      "2026-09-19T00:00:00Z",
	})
	if err != nil {
		t.Fatal(err)
	}
	if res.Document.SystemSecurityPlan.ImportProfile.Href != "./OSCAL-PROFILE.json" {
		t.Fatalf("unexpected profile import: %q", res.Document.SystemSecurityPlan.ImportProfile.Href)
	}
}
