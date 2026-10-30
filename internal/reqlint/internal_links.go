// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package reqlint

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// LintRequirementInternalLinks warns when a requirement links to another
// requirement within the same document. Requirements carry no tiers; a
// requirement is either delegated to a subsystem (declared by a downward
// composition allocation) or implemented here. Linking requirements to one
// another inside a single document recreates tier coupling and is confusing
// across system boundaries, so it is flagged.
// TRLC-LINKS: REQ-EMG-023
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func LintRequirementInternalLinks(reqs model.RequirementsDocument) []validate.Diagnostic {
	ids := map[string]bool{}
	for _, r := range reqs.Requirements {
		ids[strings.TrimSpace(r.ID)] = true
	}
	var diags []validate.Diagnostic
	for _, r := range reqs.Requirements {
		for _, raw := range r.AppliesTo {
			target := strings.TrimSpace(raw)
			if ids[target] {
				diags = append(diags, validate.Diagnostic{
					Code: "requirement.internal_link", Severity: validate.SeverityWarning,
					Message: fmt.Sprintf("requirement %s links to requirement %s in the same document; requirements delegate to subsystems, not to other requirements here", r.ID, target),
					Path:    "requirements",
				})
			}
		}
	}
	return diags
}
