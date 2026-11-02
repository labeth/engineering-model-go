// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel

// Requirement links across composed models: contract item fingerprints and
// link evaluation. Thin facade over internal/reqlinks; TestFacadeReexportsEngine
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
	LinkStateSuspect          = reqlinks.StateSuspect
	LinkStateBroken           = reqlinks.StateBroken
	LinkStateVersionMismatch  = reqlinks.StateVersionMismatch
	LinkStateChildUnpublished = reqlinks.StateChildUnpublished
	LinkStateUnresolved       = reqlinks.StateUnresolved
)

// LinkDependency is one resolved dependency as requirement link evaluation sees it.
// TRLC-LINKS: REQ-EMG-097
type LinkDependency = reqlinks.Dependency

// LinkStatus is the evaluation of one requirement link.
// TRLC-LINKS: REQ-EMG-097
type LinkStatus = reqlinks.LinkStatus

// CanonicalContractItem returns the canonical serialization a fingerprint hashes.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func CanonicalContractItem(item ContractItem) []byte { return reqlinks.CanonicalItem(item) }

// ContractItemFingerprint returns the sha256:<hex> fingerprint of a contract item.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ContractItemFingerprint(item ContractItem) string { return reqlinks.Fingerprint(item) }

// ContractItemsFromBundle returns every contract item of a model with its realizers.
// TRLC-LINKS: REQ-EMG-095
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ContractItemsFromBundle(bundle Bundle) map[string]ContractItem {
	return reqlinks.ItemsFromBundle(bundle)
}

// RealizedContractItems returns the contract item IDs a requirement realizes.
// TRLC-LINKS: REQ-EMG-090
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func RealizedContractItems(r Requirement, refs map[string]string) []string {
	return reqlinks.RealizedItems(r, refs)
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
	return reqlinks.Diagnostics(system, deps)
}

// SuspectLinkMessage describes a suspect link with fingerprints and current texts.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func SuspectLinkMessage(status LinkStatus) string { return reqlinks.SuspectMessage(status) }

// LinkDependencies returns the resolved direct dependencies of a composed system.
// TRLC-LINKS: REQ-EMG-097
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func LinkDependencies(sys *ComposedSystem) []LinkDependency { return composition.LinkDependencies(sys) }
