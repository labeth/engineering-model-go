// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

// Composition-time requirement link diagnostics: the link rules that need
// the resolved dependencies. internal/reqlinks evaluates the links; this
// file turns the evaluation into diagnostics.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// LinkDiagnostics evaluates the links and returns the composition-time link
// diagnostics: link.suspect, link.fingerprint_outdated, link.broken and
// link.child_unpublished (errors;
// a child naming an undeclared or uncomposed dependency is reported by
// validation, and version_mismatch too) and link.aou_unverified for every
// published assumption of use of a resolved dependency that no link with
// evidence covers (error in strict mode, warning in guided mode where the
// rules apply, see validate.RequirementLinkRules).
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-098, REQ-EMG-109
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func LinkDiagnostics(system model.Bundle, deps []reqlinks.Dependency) []validate.Diagnostic {
	path := system.ReqLinksPath
	if path == "" {
		path = "reqLinks"
	}
	var diags []validate.Diagnostic
	errAt := func(code, msg string) {
		diags = append(diags, validate.Diagnostic{Code: code, Severity: validate.SeverityError, Message: msg, Path: path})
	}
	for _, s := range reqlinks.EvaluateLinks(system, deps) {
		content := s.State
		if s.State == reqlinks.StateVersionMismatch {
			// Validation reports the stale pin; the content state tells the
			// reviewer what changed at the pinned dependency version.
			content = s.Content
		}
		switch content {
		case reqlinks.StateSuspect:
			errAt("link.suspect", reqlinks.SuspectMessage(s))
		case reqlinks.StateFingerprintOutdated:
			errAt("link.fingerprint_outdated", reqlinks.OutdatedMessage(s))
		case reqlinks.StateBroken:
			errAt("link.broken", fmt.Sprintf("link %s: %s no longer exists in %s at %s; remove the link or link another contract item", s.ID, s.Child, s.Alias, s.Version))
		case reqlinks.StateChildUnpublished:
			// An undeclared or uncomposed alias is reported by validation.
			if s.Reason == reqlinks.ReasonNotInPublication || s.Reason == reqlinks.ReasonNotContractItem {
				errAt("link.child_unpublished", fmt.Sprintf("link %s: %s is not a published contract item of dependency %q (%s)", s.ID, s.Child, s.Alias, s.Detail))
			}
		}
	}
	severity, applies := validate.RequirementLinkRules(system)
	if !applies {
		return diags
	}
	evidenced := map[string]bool{}
	for _, link := range system.ReqLinks.ReqLinks {
		if len(link.Evidence) > 0 {
			evidenced[strings.TrimSpace(link.Child)] = true
		}
	}
	var unverified []string
	seen := map[string]bool{}
	for _, dep := range deps {
		for id, item := range reqlinks.PublishedItems(dep.Bundle, dep.Publications...) {
			qualified := dep.Alias + "::" + id
			if item.Kind != reqlinks.KindAssumption || evidenced[qualified] || seen[qualified] {
				continue
			}
			seen[qualified] = true
			unverified = append(unverified, qualified)
		}
	}
	sort.Strings(unverified)
	for _, q := range unverified {
		diags = append(diags, validate.Diagnostic{
			Code: "link.aou_unverified", Severity: severity, Path: path,
			Message: fmt.Sprintf("assumption of use %s has no requirement link with evidence in this model; add a reqLinks entry with child %s and evidence naming the local requirement, test or control that shows it holds", q, q),
		})
	}
	return diags
}
