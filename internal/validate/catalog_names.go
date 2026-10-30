// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// catalogOperatorRe matches what the EARS clause expression parser reads as
// an operator: the standalone words and, or, not, a comma or a parenthesis.
var catalogOperatorRe = regexp.MustCompile(`(?i)(^|[^a-z0-9_-])(and|or|not)([^a-z0-9_-]|$)|[,()]`)

// catalogNameDiagnostics warns once at the catalog field when a clause-term
// name or alias contains an EARS expression operator. Such a term can never
// match in a While/When/Where/If clause: the parser splits it at the
// operator and the requirement reports unresolved-term errors instead.
// Systems, functional groups and units are left out because EARS matches the
// system name whole; referenced elements and attack vectors are not EARS
// terms.
// TRLC-LINKS: REQ-EMG-071
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func catalogNameDiagnostics(c model.CatalogGroups) []Diagnostic {
	var diags []Diagnostic
	for _, g := range []struct {
		key     string
		entries []model.CatalogEntry
	}{
		{"actors", c.Actors}, {"events", c.Events}, {"states", c.States}, {"features", c.Features},
		{"modes", c.Modes}, {"conditions", c.Conditions}, {"dataTerms", c.DataTerms},
	} {
		for i, e := range g.entries {
			check := func(text, field string) {
				m := catalogOperatorRe.FindStringSubmatch(text)
				if m == nil {
					return
				}
				op := strings.ToLower(m[2])
				if op == "" {
					op = m[0]
				}
				diags = append(diags, Diagnostic{Code: "catalog.name_has_operator", Severity: SeverityWarning,
					Message: fmt.Sprintf("catalog %s term %q (%s) contains the EARS operator %q and cannot match in a requirement clause; rename it without and, or, not, commas or parentheses (a positive term such as \"entity is deselected\" instead of \"entity is not selected\")", g.key, strings.TrimSpace(text), strings.TrimSpace(e.ID), op),
					Path:    fmt.Sprintf("catalog.%s[%d].%s", g.key, i, field)})
			}
			check(e.Name, "name")
			for j, alias := range e.Aliases {
				check(alias, fmt.Sprintf("aliases[%d]", j))
			}
		}
	}
	return diags
}
