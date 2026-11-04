// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package reqlinks

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// State is the evaluated state of one requirement link. A link is never
// compared with the content of the item at check time (decision D23): the
// pinned version is the review record, and engws bump compares meanings
// between versions.
type State string

const (
	// StateUnchanged: the pin equals the dependency pin and the item is a
	// published contract item there.
	StateUnchanged State = "unchanged"
	// StateBroken: the child item does not exist at the dependency pin.
	StateBroken State = "broken"
	// StateVersionMismatch: pinned.version differs from the dependency pin:
	// the link was reviewed at another version and needs review at this one.
	StateVersionMismatch State = "version_mismatch"
	// StateChildUnpublished: the child exists but is not a published contract
	// item, or names no declared dependency alias.
	StateChildUnpublished State = "child_unpublished"
	// StateUnresolved: the dependency is declared but was not resolved, so the
	// link cannot be evaluated.
	StateUnresolved State = "unresolved"
	// StateLocal: the dependency resolved from a workspace replacement and
	// the item is not a published contract item of that checkout; a local
	// checkout is work in progress, so nothing is reported (D23).
	StateLocal State = "local"
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
// Local is set when it resolved from a workspace replacement (a checkout,
// not the published version).
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
type Dependency struct {
	Alias        string
	Version      string
	Bundle       model.Bundle
	Publications []model.ManifestPublication
	Local        bool
}

// LinkStatus is the evaluation of one reqLinks entry: its state, review
// status (claimed or accepted), the pinned version, the dependency pin and
// the item at the dependency pin (nil when it is not a published item).
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type LinkStatus struct {
	ID      string
	Parent  string
	Child   string
	Alias   string
	ItemID  string
	State   State
	Status  string // review status: claimed or accepted
	Pinned  string // pinned.version
	Version string // the dependency pin of the alias
	Item    *Item
	// Local is set when the dependency resolved from a workspace replacement.
	Local bool
	// Reason classifies a child_unpublished, unresolved or local link;
	// Detail is its text for people.
	Reason Reason
	Detail string
}

// NeedsReview reports whether a link needs a person: version_mismatch,
// broken or claimed.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-101
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (s LinkStatus) NeedsReview() bool {
	return s.State == StateVersionMismatch || s.State == StateBroken || s.Status == model.ReqLinkClaimed
}

// EvaluateLinks evaluates every reqLinks entry of system against the
// resolved dependencies, in link order, without comparing content (D23). A
// link whose alias is not a declared dependency is child_unpublished; a
// declared but unresolved one is unresolved; an item missing from a
// published dependency is broken and one that is not a published capability
// or assumption of use child_unpublished (local for a workspace replacement,
// which is never reported); otherwise a pin that differs from the manifest
// dependency version or the resolved version is version_mismatch, else
// unchanged.
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
	published := map[string]map[string]Item{}
	out := make([]LinkStatus, 0, len(system.ReqLinks.ReqLinks))
	for _, link := range system.ReqLinks.ReqLinks {
		status := LinkStatus{
			ID: strings.TrimSpace(link.ID), Parent: strings.TrimSpace(link.Parent), Child: strings.TrimSpace(link.Child),
			Status: strings.TrimSpace(link.Status), Pinned: strings.TrimSpace(link.Pinned.Version),
		}
		alias, itemID, err := model.ParseQualifiedReference(status.Child)
		status.Alias, status.ItemID = alias, itemID
		pin, declared := pins[alias]
		status.Version = pin
		dep, resolved := byAlias[alias]
		status.Local = resolved && dep.Local
		switch {
		case err != nil || !declared:
			status.State, status.Reason, status.Detail = StateChildUnpublished, ReasonUndeclaredAlias, fmt.Sprintf("%q is not alias::ID of a declared dependency", status.Child)
		case !resolved:
			status.State, status.Reason, status.Detail = StateUnresolved, ReasonUnresolved, fmt.Sprintf("dependency %q was not resolved", alias)
		default:
			if published[alias] == nil {
				published[alias] = PublishedItems(dep.Bundle, dep.Publications...)
			}
			item, exists := published[alias][itemID]
			switch {
			case exists:
				status.Item = &item
				status.State = StateUnchanged
				if status.Pinned != pin || (dep.Version != "" && status.Pinned != dep.Version) {
					status.State = StateVersionMismatch
				}
			case dep.Local:
				status.Reason = NonItemReason(dep.Bundle, itemID)
				status.State, status.Detail = StateLocal, fmt.Sprintf("%s is not a published contract item of the workspace checkout of %s; links are checked against published versions only", status.Child, alias)
			default:
				switch status.Reason = NonItemReason(dep.Bundle, itemID); status.Reason {
				case ReasonMissing:
					status.State = StateBroken
				case ReasonNotInPublication:
					status.State, status.Detail = StateChildUnpublished, "the item is not in a selected publication"
				default:
					status.State, status.Detail = StateChildUnpublished, "the item is not a capability or assumption of use"
				}
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
