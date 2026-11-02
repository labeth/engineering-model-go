// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package reqlinks

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// State is the evaluated state of one requirement link.
type State string

const (
	// StateUnchanged: the recorded fingerprint equals the current one.
	StateUnchanged State = "unchanged"
	// StateSuspect: the item or a realizing requirement changed since review.
	StateSuspect State = "suspect"
	// StateBroken: the child item no longer exists in the pinned version.
	StateBroken State = "broken"
	// StateVersionMismatch: pinned.version differs from the dependency pin.
	StateVersionMismatch State = "version_mismatch"
	// StateChildUnpublished: the child exists but is not a published contract
	// item, or names no declared dependency alias.
	StateChildUnpublished State = "child_unpublished"
	// StateUnresolved: the dependency is declared but was not resolved, so the
	// link cannot be evaluated.
	StateUnresolved State = "unresolved"
)

// Dependency is one resolved dependency as a using system sees it: its alias,
// the version it resolved, its model and the publications selected from it.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
type Dependency struct {
	Alias        string
	Version      string
	Bundle       model.Bundle
	Publications []model.ManifestPublication
}

// LinkStatus is the evaluation of one reqLinks entry. Recorded is the pinned
// fingerprint, Current the fingerprint of the item in the resolved
// dependency (empty when it cannot be computed), Item the current item.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type LinkStatus struct {
	ID     string
	Parent string
	Child  string
	Alias  string
	ItemID string
	State  State
	// Content is the state of the item content alone: unchanged, suspect or
	// broken, also for a version_mismatch link (evaluated at the resolved
	// version against the recorded fingerprint); empty when the item cannot
	// be evaluated.
	Content  State
	Pinned   string // pinned.version
	Version  string // the dependency pin of the alias
	Recorded string
	Current  string
	Item     *Item
	Detail   string
}

// EvaluateLinks evaluates every reqLinks entry of system against the
// resolved dependencies, in link order. A link whose alias is not a declared
// dependency is child_unpublished; a declared but unresolved one is
// unresolved; a pin that differs from the manifest dependency version or
// the resolved version is version_mismatch (Content still holds the content
// state at the resolved version); an item missing from the dependency is broken; an item
// that is not a published capability or assumption of use is
// child_unpublished; otherwise the fingerprints decide unchanged or suspect.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func EvaluateLinks(system model.Bundle, deps []Dependency) []LinkStatus {
	pins := map[string]string{}
	for _, d := range system.Manifest.Dependencies {
		pins[strings.TrimSpace(d.Alias)] = strings.TrimSpace(d.Version)
	}
	byAlias := map[string]Dependency{}
	for _, d := range deps {
		if prev, ok := byAlias[d.Alias]; ok {
			prev.Publications = append(prev.Publications, d.Publications...)
			byAlias[d.Alias] = prev
			continue
		}
		byAlias[d.Alias] = d
	}
	allItems := map[string]map[string]Item{}
	published := map[string]map[string]Item{}
	out := make([]LinkStatus, 0, len(system.ReqLinks.ReqLinks))
	for _, link := range system.ReqLinks.ReqLinks {
		status := LinkStatus{
			ID: strings.TrimSpace(link.ID), Parent: strings.TrimSpace(link.Parent), Child: strings.TrimSpace(link.Child),
			Pinned: strings.TrimSpace(link.Pinned.Version), Recorded: strings.TrimSpace(link.Pinned.Fingerprint),
		}
		alias, itemID, err := model.ParseQualifiedReference(status.Child)
		status.Alias, status.ItemID = alias, itemID
		pin, declared := pins[alias]
		status.Version = pin
		dep, resolved := byAlias[alias]
		switch {
		case err != nil || !declared:
			status.State, status.Detail = StateChildUnpublished, fmt.Sprintf("%q is not alias::ID of a declared dependency", status.Child)
		case !resolved:
			status.State, status.Detail = StateUnresolved, fmt.Sprintf("dependency %q was not resolved", alias)
		default:
			if allItems[alias] == nil {
				allItems[alias] = ItemsFromBundle(dep.Bundle)
				published[alias] = PublishedItems(dep.Bundle, dep.Publications...)
			}
			item, exists := published[alias][itemID]
			switch {
			case exists:
				status.Item = &item
				status.Current = Fingerprint(item)
				status.State = StateUnchanged
				if status.Current != status.Recorded {
					status.State = StateSuspect
				}
			case allItems[alias][itemID].ID != "":
				status.State, status.Detail = StateChildUnpublished, "the item is not in a selected publication"
			case declaredElsewhere(dep.Bundle, itemID):
				status.State, status.Detail = StateChildUnpublished, "the item is not a capability or assumption of use"
			default:
				status.State = StateBroken
			}
			if status.State != StateChildUnpublished {
				status.Content = status.State
			}
			if status.Pinned != pin || (dep.Version != "" && status.Pinned != dep.Version) {
				// The fingerprint is computed from the resolved version; a pin
				// that differs from the manifest or the resolved module is
				// stale, whatever its content state.
				status.State = StateVersionMismatch
			}
		}
		out = append(out, status)
	}
	return out
}

// declaredElsewhere reports whether id is a contract entry of the bundle
// that is not a contract item (an interface or other provides/requires entry).
// TRLC-LINKS: REQ-EMG-097
func declaredElsewhere(bundle model.Bundle, id string) bool {
	for _, entries := range [][]model.ContractEntry{bundle.Architecture.Contract.Provides, bundle.Architecture.Contract.Requires} {
		for _, e := range entries {
			if strings.TrimSpace(e.ID) == id {
				return true
			}
		}
	}
	return false
}

// Diagnostics evaluates the links and returns the composition-time link
// diagnostics: link.suspect, link.broken and link.child_unpublished (errors;
// a child naming an undeclared or uncomposed dependency is reported by
// validation, and version_mismatch too) and link.aou_unverified for every
// published assumption of use of a resolved dependency that no link with
// evidence covers (error in strict mode, warning in guided mode where the
// rules apply, see validate.RequirementLinkRules).
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func Diagnostics(system model.Bundle, deps []Dependency) []validate.Diagnostic {
	path := system.ReqLinksPath
	if path == "" {
		path = "reqLinks"
	}
	var diags []validate.Diagnostic
	errAt := func(code, msg string) {
		diags = append(diags, validate.Diagnostic{Code: code, Severity: validate.SeverityError, Message: msg, Path: path})
	}
	for _, s := range EvaluateLinks(system, deps) {
		switch s.State {
		case StateSuspect:
			errAt("link.suspect", SuspectMessage(s))
		case StateVersionMismatch:
			// Validation reports the stale pin; the content state tells the
			// reviewer what changed at the pinned dependency version.
			switch s.Content {
			case StateSuspect:
				errAt("link.suspect", SuspectMessage(s))
			case StateBroken:
				errAt("link.broken", fmt.Sprintf("link %s: %s no longer exists in %s at %s; remove the link or link another contract item", s.ID, s.Child, s.Alias, s.Version))
			}
		case StateBroken:
			errAt("link.broken", fmt.Sprintf("link %s: %s no longer exists in %s at %s; remove the link or link another contract item", s.ID, s.Child, s.Alias, s.Version))
		case StateChildUnpublished:
			if s.Detail == "" || strings.HasPrefix(s.Detail, "the item") {
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
		for id, item := range PublishedItems(dep.Bundle, dep.Publications...) {
			qualified := dep.Alias + "::" + id
			if item.Kind != KindAssumption || evidenced[qualified] || seen[qualified] {
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

// SuspectMessage describes a suspect link: the recorded and current
// fingerprints and the current text of the item and its realizing
// requirements. The text at the reviewed version is not part of the pinned
// dependency; engws review-links resolves it for a before/after diff.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func SuspectMessage(s LinkStatus) string {
	var b strings.Builder
	fmt.Fprintf(&b, "link %s: %s at %s changed since review (recorded fingerprint %s, current %s)", s.ID, s.Child, s.Version, s.Recorded, s.Current)
	if s.Item != nil {
		fmt.Fprintf(&b, "; now %s %q", s.Item.Kind, normalize(s.Item.Text))
		if len(s.Item.Realizers) > 0 {
			b.WriteString(", realized by")
			for i, r := range sortedRealizers(s.Item.Realizers) {
				if i > 0 {
					b.WriteString(";")
				}
				fmt.Fprintf(&b, " %s: %s", r.ID, normalize(r.Text))
			}
		}
	}
	b.WriteString("; review the change (engws review-links) and re-stamp the fingerprint")
	return b.String()
}

// TRLC-LINKS: REQ-EMG-097
func sortedRealizers(in []Realizer) []Realizer {
	out := append([]Realizer(nil), in...)
	sort.SliceStable(out, func(i, j int) bool { return out[i].ID < out[j].ID })
	return out
}
