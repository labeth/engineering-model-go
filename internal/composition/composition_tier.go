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
// with a recorded acceptance names the dependency alias in dependencies (or,
// the older form, the subsystem that composes the dependency in appliesTo);
// and one composition.dependency_tier_unknown (a core error, ADR-EMG-073) per
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
	check := tierCheck{
		path: root.Bundle.ManifestPath, tier: tier, rank: rank,
		accepted: validate.AcceptedRiskScopes(root.Bundle), acceptedAliases: validate.AcceptedRiskDependencies(root.Bundle),
		linked: linkedParents(root.Bundle),
	}
	var diags []validate.Diagnostic
	for _, child := range root.Children {
		diags = append(diags, check.child(child)...)
	}
	return diags
}

// tierCheck is what a dependency's tier is compared with: the tier and rank
// of the system, the risks accepted for dependencies and the requirements the
// links of each dependency lead from.
type tierCheck struct {
	path            string
	tier            string
	rank            int
	accepted        map[string]bool
	acceptedAliases map[string]bool
	linked          map[string][]string
}

// linkedParents is, by dependency alias, the requirements whose links point
// into the dependency.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func linkedParents(bundle model.Bundle) map[string][]string {
	linked := map[string][]string{}
	for _, l := range bundle.ReqLinks.ReqLinks {
		alias, _, err := model.ParseQualifiedReference(strings.TrimSpace(l.Child))
		if parent := strings.TrimSpace(l.Parent); err == nil && parent != "" && !containsString(linked[alias], parent) {
			linked[alias] = append(linked[alias], parent)
		}
	}
	return linked
}

// child reports a dependency without a tier or below the system's tier that
// no accepted risk covers.
// TRLC-LINKS: REQ-EMG-126
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-VALIDATION-ENGINE
func (c tierCheck) child(child *ComposedSystem) []validate.Diagnostic {
	alias := child.Dependency
	childTier, childRank, known := validate.EffectiveTier(child.Bundle)
	if !known {
		return []validate.Diagnostic{{
			Code: "composition.dependency_tier_unknown", Severity: validate.SeverityError, Path: c.path, Remedy: validate.ConsistencyRemedy("composition.dependency_tier_unknown"),
			Message: fmt.Sprintf("dependency %q (%s@%s, subsystem %s) declares no life-cycle tier; this system is at tier %s: ask its owner to declare lifecycle in its engmod.yml and release it", alias, child.ModulePath, child.Version, child.SubsystemID, c.tier),
		}}
	}
	if childRank >= c.rank || c.acceptedAliases[alias] || c.accepted[child.SubsystemID] {
		return nil
	}
	parents := c.linked[alias]
	sort.Strings(parents)
	via := ""
	if len(parents) > 0 {
		via = "; its links do not lift the requirements " + strings.Join(parents, ", ") + " to tier " + c.tier
	}
	return []validate.Diagnostic{{
		Code: "composition.dependency_below_tier", Severity: validate.SeverityWarning, Path: c.path,
		Message: fmt.Sprintf("dependency %q (%s@%s, subsystem %s) is at tier %s, below this system's tier %s%s: raise and release the dependency, or record a risk accepted by its owner with dependencies [%s] (or appliesTo [%s])", alias, child.ModulePath, child.Version, child.SubsystemID, childTier, c.tier, via, alias, child.SubsystemID),
	}}
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
