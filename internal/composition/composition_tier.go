// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

// Composition tier rules (ADR-EMG-034): a system with a life-cycle tier
// compares each composed dependency's own tier, read from the dependency's
// manifest at its pin.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// TierDiagnostics returns, for a root system with lifecycle, one
// composition.dependency_below_tier (warning; the tier policy sets its
// severity) per composed dependency whose tier is below the system's, naming
// the system requirements whose links point into it, unless a local risk
// with a recorded acceptance names the subsystem that composes the
// dependency in appliesTo; and one composition.dependency_tier_unknown per
// dependency without a (valid) lifecycle tier. A system without lifecycle
// gets neither.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func TierDiagnostics(root *ComposedSystem) []validate.Diagnostic {
	if root == nil {
		return nil
	}
	tier, rank, ok := validate.EffectiveTier(root.Bundle)
	if !ok {
		return nil
	}
	accepted := validate.AcceptedRiskScopes(root.Bundle)
	linked := map[string][]string{}
	for _, l := range root.Bundle.ReqLinks.ReqLinks {
		alias, _, err := model.ParseQualifiedReference(strings.TrimSpace(l.Child))
		if parent := strings.TrimSpace(l.Parent); err == nil && parent != "" && !containsString(linked[alias], parent) {
			linked[alias] = append(linked[alias], parent)
		}
	}
	path := root.Bundle.ManifestPath
	var diags []validate.Diagnostic
	for _, child := range root.Children {
		alias := child.Dependency
		childTier, childRank, known := validate.EffectiveTier(child.Bundle)
		if !known {
			diags = append(diags, validate.Diagnostic{
				Code: "composition.dependency_tier_unknown", Severity: validate.SeverityWarning, Path: path,
				Message: fmt.Sprintf("dependency %q (%s@%s, subsystem %s) declares no life-cycle tier; this system is at tier %s: ask its owner to declare lifecycle in its engmod.yml and release it", alias, child.ModulePath, child.Version, child.SubsystemID, tier),
			})
			continue
		}
		if childRank >= rank || accepted[child.SubsystemID] {
			continue
		}
		parents := linked[alias]
		sort.Strings(parents)
		via := ""
		if len(parents) > 0 {
			via = "; its links do not lift the requirements " + strings.Join(parents, ", ") + " to tier " + tier
		}
		diags = append(diags, validate.Diagnostic{
			Code: "composition.dependency_below_tier", Severity: validate.SeverityWarning, Path: path,
			Message: fmt.Sprintf("dependency %q (%s@%s, subsystem %s) is at tier %s, below this system's tier %s%s: raise and release the dependency, or record a risk accepted by its owner with appliesTo [%s]", alias, child.ModulePath, child.Version, child.SubsystemID, childTier, tier, via, child.SubsystemID),
		})
	}
	return diags
}

// containsString reports whether list holds value.
// TRLC-LINKS: REQ-EMG-126
func containsString(list []string, value string) bool {
	for _, v := range list {
		if v == value {
			return true
		}
	}
	return false
}
