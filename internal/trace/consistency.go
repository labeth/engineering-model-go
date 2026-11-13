// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// The consistency checks of ADR-EMG-069 that read the code index and the
// formal evidence: code.owner_unresolved, code.model_link_kind_mismatch and
// behavior.machine_unverified. Each reports a warning; the tier policy sets
// the severity.

import (
	"fmt"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// The kinds knownModelKinds assigns to an ID.
const (
	// kindElement is an element the architecture, behavior, assurance or
	// compliance documents author.
	kindElement = "element"
	// kindRequirement is a requirement ID.
	kindRequirement = "requirement"
	// kindTerm is a catalog term that no authored element shares its ID with.
	kindTerm = "catalog term"
)

// knownModelKinds is KnownModelIDs with the kind of each ID: kindElement for
// an authored model element, kindRequirement for a requirement and kindTerm
// for a catalog entry that is neither. An ID that is an element and also a
// catalog entry or requirement is an element. It names the same IDs as
// KnownModelIDs.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func knownModelKinds(bundle model.Bundle, requirements model.RequirementsDocument) map[string]string {
	withoutCatalog := bundle
	withoutCatalog.Catalog = model.CatalogDocument{}
	elements := KnownModelIDs(withoutCatalog, model.RequirementsDocument{})
	kinds := map[string]string{}
	for id := range KnownModelIDs(bundle, model.RequirementsDocument{}) {
		kinds[id] = kindTerm
	}
	for id := range requirementIDSet(requirements) {
		kinds[id] = kindRequirement
	}
	for id := range elements {
		kinds[id] = kindElement
	}
	return kinds
}

// integrityTail is the part of the trace integrity check that follows the
// requirement rules: the tier verification diagnostics, the stale
// verification status and the consistency checks. The bundle carries the
// requirements.
// TRLC-LINKS: REQ-EMG-073, REQ-EMG-128, REQ-EMG-146, REQ-EMG-170, REQ-EMG-171, REQ-EMG-173, REQ-EMG-175, REQ-EMG-176, REQ-EMG-188
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func integrityTail(bundle model.Bundle, scopedCode []codescan.CodeItem, verification []codescan.VerificationCheck) []validate.Diagnostic {
	diags := tierVerificationDiagnostics(bundle, bundle.Requirements, verification)
	diags = append(diags, staleVerificationStatus(bundle, scopedCode, verification)...)
	diags = append(diags, ownerDiagnostics(bundle, scopedCode)...)
	diags = append(diags, modelLinkKindDiagnostics(bundle, scopedCode)...)
	diags = append(diags, coverageDiagnostics(bundle, scopedCode)...)
	diags = append(diags, stateDiscoveryDiagnostics(bundle, scopedCode)...)
	return append(diags, machineVerificationDiagnostics(bundle, verification)...)
}

// ownerDiagnostics reports code.owner_unresolved once per scoped source file
// whose ENGMODEL-OWNER-UNIT marker is missing or names no functional unit of
// the model. The owner is known per file, so the diagnostic is per file. Test
// files and generated files are left out, as are owners that name an
// imported alias::ID unit, which only the composition can resolve.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-170
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func ownerDiagnostics(bundle model.Bundle, scopedCode []codescan.CodeItem) []validate.Diagnostic {
	units := map[string]bool{}
	for _, u := range bundle.Architecture.AuthoredArchitecture.FunctionalUnits {
		units[strings.TrimSpace(u.ID)] = true
	}
	var diags []validate.Diagnostic
	for _, it := range scopedCode {
		message := ownerProblem(it, units)
		if message == "" {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "code.owner_unresolved", Severity: validate.SeverityWarning, Message: message, Path: it.Source,
			Remedy: validate.ConsistencyRemedy("code.owner_unresolved"),
		})
	}
	return diags
}

// ownerProblem describes why a code item's file has no resolved owner, or ""
// when it has one or is exempt.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-170
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ownerProblem(it codescan.CodeItem, units map[string]bool) string {
	if ownerExempt(it) {
		return ""
	}
	owner := strings.TrimSpace(it.Owner)
	if strings.Contains(owner, "::") || units[owner] {
		return ""
	}
	if owner == "" || owner == "unresolved" {
		return fmt.Sprintf("file %s has no ENGMODEL-OWNER-UNIT marker naming the functional unit that owns it", it.Source)
	}
	return fmt.Sprintf("ENGMODEL-OWNER-UNIT %q of file %s is not a functional unit of the model", owner, it.Source)
}

// ownerExempt reports whether a code item is outside the owner check: not a
// source file, a test file or a generated file.
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-170
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func ownerExempt(it codescan.CodeItem) bool {
	return it.Kind != "source_file" || codescan.IsVerificationCodeItem(it) || codescan.IsGeneratedSource(it.AbsPath)
}

// modelLinkKindDiagnostics reports code.model_link_kind_mismatch at each
// ENGMODEL-LINKS target that exists in the model but is a requirement or a
// catalog term, not a model element: a requirement belongs in TRLC-LINKS. A
// target that exists nowhere is code.dangling_model_link.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-171
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func modelLinkKindDiagnostics(bundle model.Bundle, scopedCode []codescan.CodeItem) []validate.Diagnostic {
	kinds := knownModelKinds(bundle, bundle.Requirements)
	var diags []validate.Diagnostic
	for _, it := range scopedCode {
		diags = append(diags, kindMismatches(it, kinds)...)
	}
	return diags
}

// kindMismatches reports the ENGMODEL-LINKS targets of one code item that are
// a requirement or a catalog term.
// TRLC-LINKS: REQ-EMG-171
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
func kindMismatches(it codescan.CodeItem, kinds map[string]string) []validate.Diagnostic {
	var diags []validate.Diagnostic
	for _, raw := range it.ModelLinks {
		link := strings.TrimSpace(raw)
		kind := kinds[link]
		if kind != kindRequirement && kind != kindTerm {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "code.model_link_kind_mismatch", Severity: validate.SeverityWarning, Path: it.Source,
			Message: fmt.Sprintf("ENGMODEL-LINKS %q is a %s, not a model element", link, kind),
			Remedy:  validate.ConsistencyRemedy("code.model_link_kind_mismatch"),
		})
	}
	return diags
}

// formalEvidence is what the formal evidence files say about invariants:
// whether any file exists and which machine and invariant pairs have a
// current (passing or failing, not stale) result.
type formalEvidence struct {
	exists  bool
	current map[string]bool
}

// readFormalEvidence reads the formal evidence of the verification checks.
// TRLC-LINKS: REQ-EMG-146, REQ-EMG-173
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func readFormalEvidence(baseDir string, verification []codescan.VerificationCheck) formalEvidence {
	ev := formalEvidence{current: map[string]bool{}}
	for _, c := range verification {
		ev.exists = ev.exists || len(formalCheckFiles(c)) > 0
	}
	for _, r := range formalResultsOf(baseDir, verification) {
		if r.Result == formalPass || r.Result == formalFailed {
			ev.current[r.Machine+"|"+r.Invariant] = true
		}
	}
	return ev
}

// machineVerificationDiagnostics reports behavior.machine_unverified at each
// machine without an invariant and, when a formal evidence file exists, at
// each invariant without a current formal result.
// TRLC-LINKS: REQ-EMG-146, REQ-EMG-173
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-QUINT-EXPORTER
func machineVerificationDiagnostics(bundle model.Bundle, verification []codescan.VerificationCheck) []validate.Diagnostic {
	machines := bundle.Architecture.AuthoredArchitecture.Machines
	if len(machines) == 0 {
		return nil
	}
	ev := readFormalEvidence(codescan.ModelRootDir(bundle), verification)
	var diags []validate.Diagnostic
	for i, m := range machines {
		diags = append(diags, machineUnverified(m, i, ev)...)
	}
	return diags
}

// machineUnverified reports the unverified part of one machine.
// TRLC-LINKS: REQ-EMG-146, REQ-EMG-173
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func machineUnverified(m model.Machine, index int, ev formalEvidence) []validate.Diagnostic {
	remedy := validate.ConsistencyRemedy("behavior.machine_unverified")
	path := fmt.Sprintf("authoredArchitecture.machines[%d]", index)
	if len(m.Invariants) == 0 {
		return []validate.Diagnostic{{
			Code: "behavior.machine_unverified", Severity: validate.SeverityWarning, Path: path, Remedy: remedy,
			Message: fmt.Sprintf("machine %s has no invariant, so no formal check states what it must always satisfy; add an invariant that lists the requirement it protects", m.ID),
		}}
	}
	var diags []validate.Diagnostic
	for j, inv := range m.Invariants {
		if !ev.exists || ev.current[strings.TrimSpace(m.ID)+"|"+strings.TrimSpace(inv.ID)] {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "behavior.machine_unverified", Severity: validate.SeverityWarning, Remedy: remedy,
			Path:    fmt.Sprintf("%s.invariants[%d]", path, j),
			Message: fmt.Sprintf("invariant %s of machine %s has no current formal result; generate the Quint module and rerun engquint check", inv.ID, m.ID),
		})
	}
	return diags
}
