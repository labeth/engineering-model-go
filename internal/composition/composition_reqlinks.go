// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

// Composition-time requirement link diagnostics: the link rules that need
// the resolved dependencies. internal/reqlinks evaluates the links; this
// file turns the evaluation into diagnostics.

import (
	"fmt"
	"slices"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// LinkDiagnostics evaluates the links and returns the composition-time link
// diagnostics: link.broken (the item does not exist at the dependency pin)
// and link.child_unpublished (errors; a child naming an undeclared or
// uncomposed dependency and a stale pin, link.version_mismatch, are
// reported by validation) and link.aou_unverified for every published
// assumption of use of a resolved dependency that no link with evidence
// covers (error in strict mode, warning in guided mode where the rules
// apply, see validate.RequirementLinkRules). No item content is compared
// (decision D23), and a dependency resolved from a workspace replacement
// raises no link finding: a checkout is work in progress.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func LinkDiagnostics(system model.Bundle, deps []reqlinks.Dependency) []validate.Diagnostic {
	path := system.ReqLinksPath
	if path == "" {
		path = "reqLinks"
	}
	var diags []validate.Diagnostic
	for _, s := range reqlinks.EvaluateLinks(system, deps) {
		if d, ok := linkProblem(s, path); ok {
			diags = append(diags, d)
		}
	}
	severity, applies := validate.RequirementLinkRules(system)
	if !applies {
		return diags
	}
	return append(diags, unverifiedAssumptionDiagnostics(system, deps, unverifiedSeverity{severity: severity, path: path})...)
}

// linkProblem is the error of a link that is broken or names a child that is
// not a published contract item.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func linkProblem(s reqlinks.LinkStatus, path string) (validate.Diagnostic, bool) {
	switch {
	case s.State == reqlinks.StateBroken:
		return errDiag("link.broken", fmt.Sprintf("link %s: %s does not exist in %s at %s; remove the link or link another contract item", s.ID, s.Child, s.Alias, s.Version), path), true
	case s.State == reqlinks.StateChildUnpublished && (s.Reason == reqlinks.ReasonNotInPublication || s.Reason == reqlinks.ReasonNotContractItem):
		// An undeclared or uncomposed alias is reported by validation.
		return errDiag("link.child_unpublished", fmt.Sprintf("link %s: %s is not a published contract item of dependency %q (%s)", s.ID, s.Child, s.Alias, s.Detail), path), true
	}
	return validate.Diagnostic{}, false
}

// unverifiedSeverity is how link.aou_unverified is reported: its severity and path.
type unverifiedSeverity struct {
	severity validate.Severity
	path     string
}

// unverifiedAssumptionDiagnostics reports link.aou_unverified for every
// published assumption of use of a resolved dependency that no link with
// evidence covers, in qualified id order.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func unverifiedAssumptionDiagnostics(system model.Bundle, deps []reqlinks.Dependency, report unverifiedSeverity) []validate.Diagnostic {
	evidenced := map[string]bool{}
	for _, link := range system.ReqLinks.ReqLinks {
		if len(link.Evidence) > 0 {
			evidenced[strings.TrimSpace(link.Child)] = true
		}
	}
	var unverified []string
	for _, dep := range deps {
		unverified = append(unverified, unverifiedAssumptions(dep, evidenced)...)
	}
	sort.Strings(unverified)
	var diags []validate.Diagnostic
	for _, q := range slices.Compact(unverified) {
		diags = append(diags, validate.Diagnostic{
			Code: "link.aou_unverified", Severity: report.severity, Path: report.path,
			Message: fmt.Sprintf("assumption of use %s has no requirement link with evidence in this model; add a reqLinks entry with child %s and evidence naming the local requirement, test or control that shows it holds", q, q),
		})
	}
	return diags
}

// unverifiedAssumptions is the qualified ids of the assumptions of use a
// resolved dependency publishes that no evidenced link covers. A dependency
// resolved from a workspace replacement has none: a checkout is work in
// progress.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func unverifiedAssumptions(dep reqlinks.Dependency, evidenced map[string]bool) []string {
	if dep.Local {
		return nil
	}
	var out []string
	for id, item := range reqlinks.PublishedItems(dep.Bundle, dep.Publications...) {
		qualified := dep.Alias + "::" + id
		if item.Kind == reqlinks.KindAssumption && !evidenced[qualified] {
			out = append(out, qualified)
		}
	}
	return out
}
