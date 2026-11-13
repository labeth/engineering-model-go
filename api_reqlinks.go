// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel

// Requirement links across composed models: link evaluation and the
// meaning of a contract item compared between versions (D23). Thin facade over internal/reqlinks; TestFacadeReexportsEngine
// keeps it complete.

import (
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/reqlinks"
)

// ContractItemKind is the kind of a contract item: capability or assumption.
type ContractItemKind = reqlinks.Kind

const (
	// ContractItemCapability is a contract.provides entry of kind capability.
	ContractItemCapability = reqlinks.KindCapability
	// ContractItemAssumption is a contract.assumptions assumption of use.
	ContractItemAssumption = reqlinks.KindAssumption
)

// ContractItemRealizer is a requirement that realizes a contract item.
// TRLC-LINKS: REQ-EMG-095
type ContractItemRealizer = reqlinks.Realizer

// ContractItem is a contract item with its realizing requirements.
// TRLC-LINKS: REQ-EMG-095
type ContractItem = reqlinks.Item

// LinkState is the evaluated state of one requirement link.
type LinkState = reqlinks.State

const (
	LinkStateUnchanged        = reqlinks.StateUnchanged
	LinkStateBroken           = reqlinks.StateBroken
	LinkStateVersionMismatch  = reqlinks.StateVersionMismatch
	LinkStateChildUnpublished = reqlinks.StateChildUnpublished
	LinkStateUnresolved       = reqlinks.StateUnresolved
	// LinkStateLocal: the dependency is a workspace checkout and the item is not published there.
	LinkStateLocal = reqlinks.StateLocal
)

// LinkReason classifies a child_unpublished or unresolved link.
type LinkReason = reqlinks.Reason

const (
	LinkReasonNone             = reqlinks.ReasonNone
	LinkReasonUndeclaredAlias  = reqlinks.ReasonUndeclaredAlias
	LinkReasonUnresolved       = reqlinks.ReasonUnresolved
	LinkReasonNotInPublication = reqlinks.ReasonNotInPublication
	LinkReasonNotContractItem  = reqlinks.ReasonNotContractItem
	LinkReasonMissing          = reqlinks.ReasonMissing
)

// ContractNonItemReason says why an ID is no published contract item of a dependency model.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ContractNonItemReason(dep Bundle, id string) LinkReason { return reqlinks.NonItemReason(dep, id) }

// SortedContractItemRealizers returns a copy of realizers sorted by ID.
// TRLC-LINKS: REQ-EMG-097
func SortedContractItemRealizers(in []ContractItemRealizer) []ContractItemRealizer {
	return reqlinks.SortedRealizers(in)
}

// LinkDependency is one resolved dependency as requirement link evaluation sees it.
// TRLC-LINKS: REQ-EMG-097
type LinkDependency = reqlinks.Dependency

// LinkStatus is the evaluation of one requirement link.
// TRLC-LINKS: REQ-EMG-097
type LinkStatus = reqlinks.LinkStatus

// ContractItemMeaning returns the meaning of a contract item as compact JSON.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ContractItemMeaning(item ContractItem) []byte { return reqlinks.ItemMeaning(item) }

// SameContractItemMeaning reports whether two versions of an item mean the same.
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-100
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func SameContractItemMeaning(before, after ContractItem) bool {
	return reqlinks.SameMeaning(before, after)
}

// ContractItemMeaningChanges names each difference in meaning between two versions of an item.
// TRLC-LINKS: REQ-EMG-095, REQ-EMG-100
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ContractItemMeaningChanges(before, after ContractItem) []string {
	return reqlinks.MeaningChanges(before, after)
}

// ContractItemsFromBundle returns every contract item of a model with its realizers.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ContractItemsFromBundle(bundle Bundle) map[string]ContractItem {
	return reqlinks.ItemsFromBundle(bundle)
}

// RealizedContractItems returns the contract item IDs a requirement realizes.
// TRLC-LINKS: REQ-EMG-090
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func RealizedContractItems(r Requirement) []string {
	return reqlinks.RealizedItems(r)
}

// PublishedContractItems returns the contract items and realizers the publications expose.
// TRLC-LINKS: REQ-EMG-092, REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func PublishedContractItems(bundle Bundle, publications ...ManifestPublication) map[string]ContractItem {
	return reqlinks.PublishedItems(bundle, publications...)
}

// EvaluateRequirementLinks evaluates every requirement link of a system
// against its resolved dependencies.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func EvaluateRequirementLinks(system Bundle, deps []LinkDependency) []LinkStatus {
	return reqlinks.EvaluateLinks(system, deps)
}

// RequirementLinkDiagnostics returns the composition-time link diagnostics.
// TRLC-LINKS: REQ-EMG-097, REQ-EMG-098
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func RequirementLinkDiagnostics(system Bundle, deps []LinkDependency) []Diagnostic {
	return composition.LinkDiagnostics(system, deps)
}

// LinkDependencies returns the resolved direct dependencies of a composed system.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func LinkDependencies(sys *ComposedSystem) []LinkDependency { return composition.LinkDependencies(sys) }
