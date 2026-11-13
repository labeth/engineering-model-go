// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// The codes of the fields that were deprecated and are removed (ADR-EMG-075
// and ADR-EMG-078, removal plan phase 4). Each is an untiered error: no tier
// or POA&M item changes it. The fields still decode so that the error can
// name them.
const (
	codePinnedFingerprintRemoved = "model.pinned_fingerprint_removed"
	codeProvidesRefRemoved       = "model.provides_ref_removed"
	codeAllocationsRemoved       = "model.allocations_removed"
)

// RemovedFieldDiagnostics reports each requirement link that still carries
// pinned.fingerprint, each local contract.provides entry that still carries
// ref and a manifest that still carries composition.allocations, as the errors
// model.pinned_fingerprint_removed, model.provides_ref_removed and
// model.allocations_removed with their remedy.
// TRLC-LINKS: REQ-EMG-184, REQ-EMG-187
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func RemovedFieldDiagnostics(b model.Bundle) []Diagnostic {
	diags := append(fingerprintRemovedDiagnostics(b), providesRefRemovedDiagnostics(b)...)
	return append(diags, allocationsRemovedDiagnostics(b)...)
}

// fingerprintRemovedDiagnostics reports model.pinned_fingerprint_removed at
// the pinned.fingerprint field of every link that carries one.
// TRLC-LINKS: REQ-EMG-184
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func fingerprintRemovedDiagnostics(b model.Bundle) []Diagnostic {
	var diags []Diagnostic
	for i, link := range b.ReqLinks.ReqLinks {
		if link.Pinned.Fingerprint == "" {
			continue
		}
		diags = append(diags, Diagnostic{
			Code: codePinnedFingerprintRemoved, Severity: SeverityError,
			Path:    fmt.Sprintf("reqLinks[%d].pinned.fingerprint", i),
			Message: fmt.Sprintf("link %s records pinned.fingerprint, which is no longer supported: the pinned version is the review record", strings.TrimSpace(link.ID)),
			Remedy:  ConsistencyRemedy(codePinnedFingerprintRemoved),
		})
	}
	return diags
}

// providesRefRemovedDiagnostics reports model.provides_ref_removed at the ref
// of every local contract.provides entry that carries one. An imported
// alias::ID entry is skipped: its dependency reports its own entry.
// TRLC-LINKS: REQ-EMG-184
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func providesRefRemovedDiagnostics(b model.Bundle) []Diagnostic {
	var diags []Diagnostic
	for i, entry := range b.Architecture.Contract.Provides {
		ref := strings.TrimSpace(entry.Ref)
		if ref == "" || strings.Contains(entry.ID, "::") {
			continue
		}
		diags = append(diags, Diagnostic{
			Code: codeProvidesRefRemoved, Severity: SeverityError,
			Path:    fmt.Sprintf("contract.provides[%d].ref", i),
			Message: fmt.Sprintf("contract entry %s names requirement %s in ref, which is no longer supported", strings.TrimSpace(entry.ID), ref),
			Remedy:  ConsistencyRemedy(codeProvidesRefRemoved),
		})
	}
	return diags
}

// allocationsRemovedDiagnostics reports model.allocations_removed once, at
// composition.allocations, when the manifest carries any allocation: the
// field still decodes so that the error can carry the remedy, and nothing
// reads it (ADR-EMG-078).
// TRLC-LINKS: REQ-EMG-187
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE
func allocationsRemovedDiagnostics(b model.Bundle) []Diagnostic {
	count := len(b.Architecture.Composition.Allocations)
	if count == 0 {
		return nil
	}
	return []Diagnostic{{
		Code: codeAllocationsRemoved, Severity: SeverityError,
		Path:    "composition.allocations",
		Message: fmt.Sprintf("composition.allocations carries %d allocation(s), which is no longer supported: requirement links are the only trace to a requirement of another model", count),
		Remedy:  ConsistencyRemedy(codeAllocationsRemoved),
	}}
}
