// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package reqlinks

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
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
	// StateFingerprintOutdated: the recorded fingerprint was written in
	// another canonical form (an older or unknown sha256:vN: version, or the
	// unversioned form), so it cannot be compared (decision D17).
	StateFingerprintOutdated State = "fingerprint_outdated"
)

// Reason says why a link is child_unpublished or unresolved, or why a
// contract ID of a resolved dependency is no link target ("" otherwise).
type Reason string

const (
	// ReasonNone: the state needs no reason.
	ReasonNone Reason = ""
	// ReasonUndeclaredAlias: the child is not alias::ID of a declared dependency.
	ReasonUndeclaredAlias Reason = "undeclared_alias"
	// ReasonUnresolved: the dependency is declared but was not resolved.
	ReasonUnresolved Reason = "unresolved"
	// ReasonNotInPublication: the item exists but no selected publication exposes it.
	ReasonNotInPublication Reason = "not_in_publication"
	// ReasonNotContractItem: the ID is a contract entry that is not a
	// capability or assumption of use (an interface, a requires entry).
	ReasonNotContractItem Reason = "not_contract_item"
	// ReasonMissing: the dependency has no contract entry with the ID.
	ReasonMissing Reason = "missing"
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
	// Reason classifies a child_unpublished or unresolved link; Detail is
	// its text for people.
	Reason Reason
	Detail string
}

// EvaluateLinks evaluates every reqLinks entry of system against the
// resolved dependencies, in link order. A link whose alias is not a declared
// dependency is child_unpublished; a declared but unresolved one is
// unresolved; a pin that differs from the manifest dependency version or
// the resolved version is version_mismatch (Content still holds the content
// state at the resolved version); an item missing from the dependency is broken; an item
// that is not a published capability or assumption of use is
// child_unpublished; otherwise the fingerprints decide unchanged or suspect.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-109
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
			status.State, status.Reason, status.Detail = StateChildUnpublished, ReasonUndeclaredAlias, fmt.Sprintf("%q is not alias::ID of a declared dependency", status.Child)
		case !resolved:
			status.State, status.Reason, status.Detail = StateUnresolved, ReasonUnresolved, fmt.Sprintf("dependency %q was not resolved", alias)
		default:
			if published[alias] == nil {
				published[alias] = PublishedItems(dep.Bundle, dep.Publications...)
			}
			if item, exists := published[alias][itemID]; exists {
				status.Item = &item
				status.Current = Fingerprint(item)
				switch {
				case status.Recorded != "" && FingerprintVersion(status.Recorded) != CanonicalVersion:
					status.State = StateFingerprintOutdated
				case status.Current != status.Recorded:
					status.State = StateSuspect
				default:
					status.State = StateUnchanged
				}
			} else {
				switch status.Reason = NonItemReason(dep.Bundle, itemID); status.Reason {
				case ReasonMissing:
					status.State = StateBroken
				case ReasonNotInPublication:
					status.State, status.Detail = StateChildUnpublished, "the item is not in a selected publication"
				default:
					status.State, status.Detail = StateChildUnpublished, "the item is not a capability or assumption of use"
				}
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

// NonItemReason says why id is no published contract item of a resolved
// dependency: ReasonNotInPublication when it is a capability or assumption
// of use (or a capability entry) no selected publication exposes,
// ReasonNotContractItem when it is a contract entry of another kind (an
// interface, a requires entry), ReasonMissing when the dependency declares
// no contract entry with that ID. Callers have already checked that id is
// not published.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-104
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func NonItemReason(dep model.Bundle, id string) Reason {
	if _, ok := ItemsFromBundle(dep)[id]; ok {
		return ReasonNotInPublication
	}
	for _, entries := range [][]model.ContractEntry{dep.Architecture.Contract.Provides, dep.Architecture.Contract.Requires} {
		for _, e := range entries {
			if strings.TrimSpace(e.ID) == id {
				return ReasonNotContractItem
			}
		}
	}
	return ReasonMissing
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
			for i, r := range SortedRealizers(s.Item.Realizers) {
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

// OutdatedMessage describes a link whose recorded fingerprint uses another
// canonical form than this engine writes (decision D17).
// TRLC-LINKS: REQ-EMG-109
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func OutdatedMessage(s LinkStatus) string {
	form := "the unversioned sha256:<hex> form"
	if v := FingerprintVersion(s.Recorded); v > 0 {
		form = fmt.Sprintf("canonical form v%d", v)
	}
	return fmt.Sprintf("link %s: the recorded fingerprint of %s uses %s, this engine writes v%d, so the two cannot be compared; check that nothing in the item changed since the review (engws review-links), then re-stamp it with engws review-links --accept %s, or use an engine that migrates that form",
		s.ID, s.Child, form, CanonicalVersion, s.ID)
}
