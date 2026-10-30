// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// Trace integrity and the machine-readable traceability matrix. Code trace markers
// (TRLC-LINKS to a requirement, ENGMODEL-LINKS to a model element) are existence-checked
// against the model — not only shape-checked — and code is attributed to the model whose
// engmod.yml is its nearest enclosing root, so a parent model never validates or
// counts a child model's links. The consolidated trace is also emitted as a matrix.

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// Matrix is the consolidated, machine-readable traceability matrix for one model.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type Matrix struct {
	Model         string         `json:"model"`
	Requirements  []Row          `json:"requirements"`
	DanglingLinks []DanglingLink `json:"danglingLinks"`
	Orphans       []string       `json:"orphanRequirements"`
	Summary       Summary        `json:"summary"`
}

// Row is one requirement and everything that traces to it.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type Row struct {
	ID            string    `json:"id"`
	Text          string    `json:"text,omitempty"`
	Units         []string  `json:"units,omitempty"`
	Code          []CodeRef `json:"code,omitempty"`
	Verifications []string  `json:"verifications,omitempty"`
	// Result is the recorded outcome of the linked tests: "passed" when parsed
	// test-result artifacts report only passes, "failed" when any reports a
	// failure (or a blocked, partial or flaky run), "not-run" otherwise. It is
	// empty when no verification is linked.
	Result      string      `json:"result,omitempty"`
	DelegatedTo *Delegation `json:"delegatedTo,omitempty"`
	// DelegatedStatus is the delegated (or imported) requirement's own trace
	// status in its target repository at the resolved version.
	DelegatedStatus *DelegatedStatus `json:"delegatedStatus,omitempty"`
	Provenance      *Provenance      `json:"provenance,omitempty"`
	Status          string           `json:"status"`
}

// Provenance identifies the selected dependency publication that supplied
// an imported requirement.
// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
type Provenance struct {
	Alias       string `json:"alias"`
	ModulePath  string `json:"modulePath"`
	Version     string `json:"version"`
	SourceModel string `json:"sourceModel"`
	Publication string `json:"publication"`
	SourceID    string `json:"sourceId"`
}

// DelegatedStatus is the trace status of a delegation's target
// requirement computed in the dependency itself. Status is "implemented",
// "test-linked", "delegated" or "orphan" as in that repository's own matrix,
// "not-available" when the dependency has no code sources, and
// "unknown-requirement" when the target requirement does not exist there.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
type DelegatedStatus struct {
	Requirement string `json:"requirement"`
	Status      string `json:"status"`
	Implemented bool   `json:"implemented"`
	TestLinked  bool   `json:"testLinked"`
	Result      string `json:"result,omitempty"`
}

// CodeRef is a code symbol that implements a requirement.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type CodeRef struct {
	Symbol string `json:"symbol"`
	Path   string `json:"path"`
	Line   int    `json:"line,omitempty"`
}

// Delegation is the downward allocation of a requirement onto a subsystem.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type Delegation struct {
	Subsystem         string `json:"subsystem"`
	Target            string `json:"target"`
	TargetRequirement string `json:"targetRequirement,omitempty"`
}

// DanglingLink is a code trace link that resolves to nothing in the model.
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, CTRL-TRACEABILITY-COVERAGE
type DanglingLink struct {
	Kind   string `json:"kind"` // requirement | model
	Target string `json:"target"`
	From   string `json:"from"` // file:line
}

// Summary is the headline coverage roll-up.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type Summary struct {
	Requirements int `json:"requirements"`
	Implemented  int `json:"implemented"`
	// Verified is deprecated: it counts test-linked requirements and always
	// equals TestLinked and Tested. A linked test is not a verification; use
	// TestLinked for links and Passed for requirements with recorded passes.
	Verified int `json:"verified"`
	// Tested counts requirements with at least one linked test function (or
	// verification artifact).
	Tested int `json:"tested"`
	// TestLinked equals Tested under the name that says what it counts: a
	// requirement linked from a test entry point, run or not.
	TestLinked int `json:"testLinked"`
	// Passed counts requirements whose linked verification has a recorded pass
	// and no recorded failure in parsed test-result artifacts.
	Passed    int `json:"passed"`
	Delegated int `json:"delegated"`
	// DelegatedImplemented and DelegatedTestLinked count local delegated rows
	// whose target requirement is implemented, respectively test linked, in
	// the target repository.
	DelegatedImplemented int `json:"delegatedImplemented"`
	DelegatedTestLinked  int `json:"delegatedTestLinked"`
	Orphan               int `json:"orphan"`
	DanglingLinks        int `json:"danglingLinks"`
}

// requirementIDSet is the set of requirement ids defined by the model.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func requirementIDSet(requirements model.RequirementsDocument) map[string]bool {
	ids := map[string]bool{}
	for _, r := range requirements.Requirements {
		ids[strings.TrimSpace(r.ID)] = true
	}
	return ids
}

// KnownModelIDs is every id the model defines (catalog, authored architecture, and
// requirements) — the oracle against which an ENGMODEL-LINKS target is resolved.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-MODEL-LOADER
func KnownModelIDs(bundle model.Bundle, requirements model.RequirementsDocument) map[string]bool {
	ids := map[string]bool{}
	add := func(id string) {
		if id = strings.TrimSpace(id); id != "" {
			ids[id] = true
		}
	}
	a := bundle.Architecture.AuthoredArchitecture
	for _, x := range a.FunctionalGroups {
		add(x.ID)
	}
	for _, x := range a.FunctionalUnits {
		add(x.ID)
	}
	for _, x := range a.Actors {
		add(x.ID)
	}
	for _, x := range a.AttackVectors {
		add(x.ID)
	}
	for _, x := range a.ReferencedElements {
		add(x.ID)
	}
	for _, x := range a.Interfaces {
		add(x.ID)
	}
	for _, x := range a.DataObjects {
		add(x.ID)
	}
	for _, x := range a.DeploymentTargets {
		add(x.ID)
	}
	for _, x := range a.HardwareItems {
		add(x.ID)
	}
	for _, x := range a.HardwareInterfaces {
		add(x.ID)
	}
	for _, x := range a.Controls {
		add(x.ID)
	}
	for _, x := range a.Risks {
		add(x.ID)
	}
	for _, x := range a.TrustBoundaries {
		add(x.ID)
	}
	for _, x := range a.States {
		add(x.ID)
	}
	for _, x := range a.Events {
		add(x.ID)
	}
	for _, x := range a.Flows {
		add(x.ID)
		for _, s := range x.Steps {
			add(x.ID + "::" + strings.TrimSpace(s.ID))
		}
	}
	for _, x := range a.ControlVerifications {
		add(x.ID)
	}
	for _, x := range a.POAMItems {
		add(x.ID)
	}
	for _, x := range a.ThreatScenarios {
		add(x.ID)
	}
	for _, x := range a.ThreatAssumptions {
		add(x.ID)
	}
	for _, x := range a.ThreatOutOfScope {
		add(x.ID)
	}
	for _, x := range a.ThreatMitigations {
		add(x.ID)
	}
	c := bundle.Catalog.Catalog
	for _, group := range [][]model.CatalogEntry{
		c.Systems, c.FunctionalGroups, c.FunctionalUnits, c.ReferencedElements, c.Actors,
		c.AttackVectors, c.Events, c.States, c.Features, c.Modes, c.Conditions, c.DataTerms,
	} {
		for _, x := range group {
			add(x.ID)
		}
	}
	for _, r := range requirements.Requirements {
		add(r.ID)
	}
	return ids
}

// EffectiveCodeRoots mirrors codescan.InferCodeItems' root derivation: the explicit code-root
// option, or the module root when it is not set. Scoping must use the same roots
// the scan used, otherwise items scanned via inferenceHints (with no --code-root) escape it.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-MODEL-LOADER
func EffectiveCodeRoots(bundle model.Bundle, codeRootOption string) []string {
	roots := []string{}
	if strings.TrimSpace(codeRootOption) != "" {
		roots = append(roots, codeRootOption)
	}
	return codescan.WithDefaultCodeRoot(bundle, codescan.UniqueExistingSources(roots))
}

// ScopeCodeToModel keeps only code that belongs to the model rooted at modelDir: a file
// belongs to the model whose engmod.yml is its nearest enclosing directory. This
// stops a parent model (scanned with a broad code root) from claiming a nested model's code.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func ScopeCodeToModel(items []codescan.CodeItem, roots []string, modelDir string) []codescan.CodeItem {
	if len(roots) == 0 {
		return items
	}
	modelDirAbs, _ := filepath.Abs(modelDir)
	modelRoots := map[string]bool{}
	for _, r := range roots {
		ra, err := filepath.Abs(r)
		if err != nil {
			continue
		}
		if info, err := os.Stat(ra); err == nil && !info.IsDir() {
			for dir := filepath.Dir(ra); ; dir = filepath.Dir(dir) {
				if info, err := os.Stat(filepath.Join(dir, "engmod.yml")); err == nil && !info.IsDir() {
					modelRoots[dir] = true
					break
				}
				if filepath.Dir(dir) == dir {
					break
				}
			}
		}
		_ = filepath.WalkDir(ra, func(p string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil
			}
			if d.IsDir() {
				if codescan.SkipScanDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() == "engmod.yml" {
				modelRoots[filepath.Dir(p)] = true
			}
			return nil
		})
	}
	if len(modelRoots) == 0 {
		return items
	}
	nearestRoot := func(fileAbs string) string {
		best := ""
		for r := range modelRoots {
			if fileAbs == r || strings.HasPrefix(fileAbs, r+string(filepath.Separator)) {
				if len(r) > len(best) {
					best = r
				}
			}
		}
		return best
	}
	out := make([]codescan.CodeItem, 0, len(items))
	for _, it := range items {
		// Trace-bearing items carry an unambiguous absolute path captured at scan time;
		// items without one (no trace links) are kept regardless.
		fileAbs := strings.TrimSpace(it.AbsPath)
		if fileAbs == "" {
			out = append(out, it)
			continue
		}
		if nr := nearestRoot(fileAbs); nr == "" || nr == modelDirAbs {
			out = append(out, it)
		}
	}
	return out
}

// ValidateIntegrity rejects dangling code trace links (a TRLC-LINKS or ENGMODEL-LINKS
// that resolves to nothing in the model) and reports orphan requirements (traced by no
// code, verification, or delegation). The code is assumed already scoped to this model.
// It also reports assurance records whose status lags the tests they name
// (verification.status_stale).
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-073
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func ValidateIntegrity(bundle model.Bundle, requirements model.RequirementsDocument, scopedCode []codescan.CodeItem, verification []codescan.VerificationCheck, delegationsByReq map[string][]composition.MaterializedAllocation) []validate.Diagnostic {
	diags := validateIntegrityWithIDs(KnownModelIDs(bundle, requirements), requirements, scopedCode, verification, delegationsByReq)
	return append(diags, staleVerificationStatus(bundle, scopedCode, verification)...)
}

// validateIntegrityWithIDs is ValidateIntegrity against an explicit
// set of model IDs, which may include published alias::ID entities.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029, REQ-EMG-065
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func validateIntegrityWithIDs(modelIDs map[string]bool, requirements model.RequirementsDocument, scopedCode []codescan.CodeItem, verification []codescan.VerificationCheck, delegationsByReq map[string][]composition.MaterializedAllocation) []validate.Diagnostic {
	var diags []validate.Diagnostic
	reqIDs := requirementIDSet(requirements)
	covered := map[string]bool{}
	for _, it := range scopedCode {
		for _, raw := range it.Implements {
			link := strings.TrimSpace(raw)
			if link == "" {
				continue
			}
			if reqIDs[link] {
				covered[link] = true
				continue
			}
			diags = append(diags, validate.Diagnostic{
				Code: "code.dangling_requirement_link", Severity: validate.SeverityError,
				Message: fmt.Sprintf("TRLC-LINKS %q resolves to no requirement in the model", link),
				Path:    it.Source,
			})
		}
		for _, raw := range it.ModelLinks {
			link := strings.TrimSpace(raw)
			if link == "" || modelIDs[link] {
				continue
			}
			diags = append(diags, validate.Diagnostic{
				Code: "code.dangling_model_link", Severity: validate.SeverityError,
				Message: fmt.Sprintf("ENGMODEL-LINKS %q resolves to no model element", link),
				Path:    it.Source,
			})
		}
	}
	// Verification is not path-scoped like code, but this is sound: a requirement id is
	// unique to its model, so a foreign model's verification can only mark its own
	// (differently-prefixed) requirement ids, never one of this model's requirements.
	verified := map[string]bool{}
	for _, v := range verification {
		for _, r := range v.Verifies {
			verified[strings.TrimSpace(r)] = true
		}
	}
	for _, r := range requirements.Requirements {
		id := strings.TrimSpace(r.ID)
		if covered[id] || verified[id] || len(delegationsByReq[id]) > 0 {
			continue
		}
		diags = append(diags, validate.Diagnostic{
			Code: "requirement.orphan", Severity: validate.SeverityWarning,
			Message: fmt.Sprintf("requirement %s is traced by no code, verification, or delegation", id),
			Path:    "requirements",
		})
	}
	return diags
}

// buildTraceMatrix assembles the consolidated traceability matrix for the model.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func buildTraceMatrix(bundle model.Bundle, requirements model.RequirementsDocument, scopedCode []codescan.CodeItem, verification []codescan.VerificationCheck, delegationsByReq map[string][]composition.MaterializedAllocation) Matrix {
	return buildTraceMatrixWithIDs(bundle, KnownModelIDs(bundle, requirements), requirements, scopedCode, verification, delegationsByReq)
}

// buildTraceMatrixWithIDs is buildTraceMatrix resolving ENGMODEL-LINKS against
// an explicit ID set, which may include published alias::ID entities.
// TRLC-LINKS: REQ-EMG-030, REQ-EMG-065
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, CTRL-TRACEABILITY-COVERAGE
func buildTraceMatrixWithIDs(bundle model.Bundle, modelIDs map[string]bool, requirements model.RequirementsDocument, scopedCode []codescan.CodeItem, verification []codescan.VerificationCheck, delegationsByReq map[string][]composition.MaterializedAllocation) Matrix {
	reqIDs := requirementIDSet(requirements)
	codeByReq := map[string][]CodeRef{}
	var dangling []DanglingLink
	for _, it := range scopedCode {
		path, line := codescan.SplitSourceLine(it.Source)
		for _, raw := range it.Implements {
			link := strings.TrimSpace(raw)
			if link == "" {
				continue
			}
			if reqIDs[link] {
				if !codescan.IsVerificationCodeItem(it) {
					codeByReq[link] = append(codeByReq[link], CodeRef{Symbol: it.Element, Path: path, Line: line})
				}
			} else {
				dangling = append(dangling, DanglingLink{Kind: "requirement", Target: link, From: it.Source})
			}
		}
		for _, raw := range it.ModelLinks {
			link := strings.TrimSpace(raw)
			if link != "" && !modelIDs[link] {
				dangling = append(dangling, DanglingLink{Kind: "model", Target: link, From: it.Source})
			}
		}
	}
	verByReq := map[string][]string{}
	for _, v := range verification {
		for _, r := range v.Verifies {
			id := strings.TrimSpace(r)
			verByReq[id] = append(verByReq[id], strings.TrimSpace(v.ID))
		}
	}
	resultByReq := requirementResults(verification)

	matrix := Matrix{Model: strings.TrimSpace(bundle.Architecture.Model.ID)}
	for _, r := range requirements.Requirements {
		id := strings.TrimSpace(r.ID)
		row := Row{ID: id, Text: strings.TrimSpace(r.Text), Units: textutil.UniqueSorted(r.AppliesTo)}
		row.Code = codeByReq[id]
		row.Verifications = textutil.UniqueSorted(verByReq[id])
		if len(row.Verifications) > 0 {
			row.Result = resultByReq[id]
		}
		if d := delegationsByReq[id]; len(d) > 0 {
			row.DelegatedTo = &Delegation{Subsystem: d[0].Subsystem, Target: d[0].Target, TargetRequirement: d[0].TargetRef}
		}
		switch {
		case len(row.Code) > 0:
			row.Status = "implemented"
		case row.DelegatedTo != nil:
			row.Status = "delegated"
		case len(row.Verifications) > 0:
			row.Status = "verified"
		default:
			row.Status = "orphan"
			matrix.Orphans = append(matrix.Orphans, id)
		}
		matrix.Requirements = append(matrix.Requirements, row)
	}
	sort.SliceStable(dangling, func(i, j int) bool {
		if dangling[i].From != dangling[j].From {
			return dangling[i].From < dangling[j].From
		}
		return dangling[i].Target < dangling[j].Target
	})
	matrix.DanglingLinks = dangling
	for _, row := range matrix.Requirements {
		matrix.Summary.Requirements++
		if len(row.Code) > 0 {
			matrix.Summary.Implemented++
		}
		if len(row.Verifications) > 0 {
			matrix.Summary.Tested++
		}
		if row.Result == "passed" {
			matrix.Summary.Passed++
		}
		if row.DelegatedTo != nil {
			matrix.Summary.Delegated++
		}
		if row.Status == "orphan" {
			matrix.Summary.Orphan++
		}
	}
	matrix.Summary.DanglingLinks = len(dangling)
	matrix.Summary.Verified = matrix.Summary.Tested
	matrix.Summary.TestLinked = matrix.Summary.Tested
	return matrix
}

// requirementResults rolls the per-requirement results of every linked
// verification into one row result: failed > passed > not-run.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func requirementResults(verification []codescan.VerificationCheck) map[string]string {
	out := map[string]string{}
	rank := map[string]int{"not-run": 0, "passed": 1, "failed": 2}
	for _, v := range verification {
		for _, req := range v.Verifies {
			id := strings.TrimSpace(req)
			result := "not-run"
			for _, r := range v.Results {
				if strings.TrimSpace(r.Requirement) != id {
					continue
				}
				switch codescan.NormalizeResultStatus(r.Status) {
				case "pass":
					result = "passed"
				case "fail", "blocked", "partial", "flaky":
					result = "failed"
				}
			}
			if current, ok := out[id]; !ok || rank[result] > rank[current] {
				out[id] = result
			}
		}
	}
	return out
}

// Options configures BuildFromFilesWith.
// TRLC-LINKS: REQ-EMG-063, REQ-EMG-064, REQ-EMG-066
type Options struct {
	// Resolution overrides ENGMOD_WORK / ENGMOD_LOCK for the composition run.
	Resolution composition.ResolutionOptions
	// Composition, when set, is a composition already resolved for this
	// manifest; the trace then does not compose again.
	Composition *composition.CompositionResult
}

// BuildFromFiles loads a model, scans its code root, scopes the code to the
// model, validates trace integrity, and returns the matrix together with diagnostics.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-050, REQ-EMG-051
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func BuildFromFiles(modelPath, requirementsPath, codeRoot string) (Matrix, []validate.Diagnostic, error) {
	return BuildFromFilesWith(modelPath, requirementsPath, codeRoot, Options{})
}

// BuildFromFilesWith is BuildFromFiles with explicit
// resolution options. With composition it composes once, resolves code
// ENGMODEL-LINKS alias::ID against the dependencies' published entities
// (requirements stay local), returns the composition diagnostics, and reports
// each delegated requirement's own status in its target repository.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030, REQ-EMG-050, REQ-EMG-051, REQ-EMG-065, REQ-EMG-066, REQ-EMG-067, REQ-EMG-073
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-MODEL-LOADER, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func BuildFromFilesWith(modelPath, requirementsPath, codeRoot string, options Options) (Matrix, []validate.Diagnostic, error) {
	bundle, err := codescan.LoadBundle(modelPath)
	if err != nil {
		return Matrix{}, nil, err
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return Matrix{}, nil, err
	}
	bundle.Requirements = requirements
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return Matrix{}, nil, err
	}
	bundle = canonical.Documents()
	requirements = bundle.Requirements
	resolvedRoot := strings.TrimSpace(codeRoot)
	if resolvedRoot != "" && !filepath.IsAbs(resolvedRoot) {
		resolvedRoot = filepath.Join(filepath.Dir(modelPath), resolvedRoot)
	}
	inferredCode, codeDiags := codescan.InferCodeItems(bundle, resolvedRoot)
	inferredVerification, verDiags := codescan.InferVerificationChecks(bundle, requirements, inferredCode, resolvedRoot)
	diags := append([]validate.Diagnostic(nil), codeDiags...)
	diags = append(diags, verDiags...)

	compositionResult, cerr := composition.ComposeForGeneration(bundle, options.Composition, options.Resolution)
	if cerr != nil {
		diags = append(diags, validate.Diagnostic{
			Code: "composition.resolve_failed", Severity: validate.SeverityWarning,
			Message: fmt.Sprintf("composition could not be resolved (delegated requirements may appear orphan): %v", cerr),
			Path:    bundle.ArchitecturePath,
		})
	}
	diags = append(diags, compositionResult.Diagnostics...)
	compositionProjection := composition.BuildCompositionProjection(compositionResult)
	delegationsByReq := composition.DelegationsByRequirement(compositionResult)
	// Code links may name published dependency entities; requirements stay
	// local so the rows and counts below are this model's own.
	linkBundle := compositionProjection.EnrichBundle(bundle, "architecture", "behavior", "assurance", "compliance")
	linkIDs := KnownModelIDs(linkBundle, requirements)

	scoped := ScopeCodeToModel(inferredCode, EffectiveCodeRoots(bundle, resolvedRoot), codescan.ModelRootDir(bundle))
	diags = append(diags, validateIntegrityWithIDs(linkIDs, requirements, scoped, inferredVerification, delegationsByReq)...)
	diags = append(diags, staleVerificationStatus(bundle, scoped, inferredVerification)...)
	matrix := buildTraceMatrixWithIDs(bundle, linkIDs, requirements, scoped, inferredVerification, delegationsByReq)
	appendPublishedRequirementTraceRows(&matrix, compositionProjection, compositionResult)
	applyDelegatedStatus(&matrix, compositionResult)
	return matrix, validate.SortDiagnostics(diags), nil
}

// dependencyTraceRows computes a resolved dependency's own trace matrix rows
// by requirement id, scanning its module directory (local replacement or
// module cache entry, minus its exclude globs) the way the root model's module
// root is scanned. ok is false when that directory holds no source code.
// TRLC-LINKS: REQ-EMG-067
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func dependencyTraceRows(child *composition.ComposedSystem) (map[string]Row, bool) {
	bundle := child.Bundle
	code, _ := codescan.InferCodeItems(bundle, "")
	verification, _ := codescan.InferVerificationChecks(bundle, bundle.Requirements, code, "")
	scoped := ScopeCodeToModel(code, EffectiveCodeRoots(bundle, ""), codescan.ModelRootDir(bundle))
	if len(scoped) == 0 && len(verification) == 0 {
		return nil, false
	}
	delegations := map[string][]composition.MaterializedAllocation{}
	for _, m := range composition.MaterializeAllocations(&composition.ComposedSystem{Bundle: bundle, Children: child.Children, Requirements: child.Requirements}) {
		if m.System == strings.TrimSpace(bundle.Architecture.Model.ID) {
			delegations[strings.TrimSpace(m.Requirement)] = append(delegations[strings.TrimSpace(m.Requirement)], m)
		}
	}
	matrix := buildTraceMatrix(bundle, bundle.Requirements, scoped, verification, delegations)
	rows := make(map[string]Row, len(matrix.Requirements))
	for _, row := range matrix.Requirements {
		rows[row.ID] = row
	}
	return rows, true
}

// applyDelegatedStatus fills DelegatedStatus for local delegated rows and
// imported requirement rows. Each dependency's trace is computed at most once,
// and only when a row needs it.
// TRLC-LINKS: REQ-EMG-067
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-SYSTEM-COMPOSITION
func applyDelegatedStatus(matrix *Matrix, composed composition.CompositionResult) {
	if composed.Root == nil {
		return
	}
	bySubsystem := map[string]*composition.ComposedSystem{}
	byAlias := map[string]*composition.ComposedSystem{}
	for _, child := range composed.Root.Children {
		bySubsystem[child.SubsystemID] = child
		byAlias[child.Dependency] = child
	}
	type cached struct {
		rows map[string]Row
		ok   bool
	}
	cache := map[*composition.ComposedSystem]cached{}
	status := func(child *composition.ComposedSystem, requirement string) *DelegatedStatus {
		if child == nil || requirement == "" {
			return nil
		}
		entry, seen := cache[child]
		if !seen {
			entry.rows, entry.ok = dependencyTraceRows(child)
			cache[child] = entry
		}
		out := &DelegatedStatus{Requirement: requirement}
		if !entry.ok {
			out.Status = "not-available"
			return out
		}
		row, exists := entry.rows[requirement]
		if !exists {
			out.Status = "unknown-requirement"
			return out
		}
		out.Implemented, out.TestLinked, out.Result = len(row.Code) > 0, len(row.Verifications) > 0, row.Result
		out.Status = row.Status
		if out.Status == "verified" {
			out.Status = "test-linked"
		}
		return out
	}
	for i := range matrix.Requirements {
		row := &matrix.Requirements[i]
		if row.DelegatedTo == nil {
			continue
		}
		if row.Provenance != nil {
			row.DelegatedStatus = status(byAlias[row.Provenance.Alias], row.Provenance.SourceID)
			continue
		}
		row.DelegatedStatus = status(bySubsystem[row.DelegatedTo.Subsystem], row.DelegatedTo.TargetRequirement)
		if row.DelegatedStatus != nil && row.DelegatedStatus.Implemented {
			matrix.Summary.DelegatedImplemented++
		}
		if row.DelegatedStatus != nil && row.DelegatedStatus.TestLinked {
			matrix.Summary.DelegatedTestLinked++
		}
	}
}

// TRLC-LINKS: REQ-EMG-050, REQ-EMG-051
func appendPublishedRequirementTraceRows(matrix *Matrix, projection composition.CompositionProjection, composed composition.CompositionResult) {
	for _, entity := range projection.Domain("requirements") {
		requirement, ok := entity.Value.(model.Requirement)
		if !ok {
			continue
		}
		row := Row{
			ID: entity.QualifiedID, Text: strings.TrimSpace(requirement.Text),
			Units: entity.QualifyList(requirement.AppliesTo),
			Provenance: &Provenance{
				Alias: entity.Provenance.Alias, ModulePath: entity.Provenance.ModulePath,
				Version: entity.Provenance.Version, SourceModel: entity.Provenance.SourceModel,
				Publication: entity.Provenance.Publication, SourceID: entity.SourceID,
			},
		}
		for _, allocation := range composed.Allocations {
			alias, _, err := model.ParseQualifiedReference(allocation.Target)
			if err == nil && alias == entity.Provenance.Alias && allocation.TargetRef == entity.SourceID {
				row.DelegatedTo = &Delegation{
					Subsystem: allocation.Subsystem, Target: allocation.Target,
					TargetRequirement: entity.QualifiedID,
				}
				break
			}
		}
		appliesTo := map[string]bool{}
		for _, ref := range requirement.AppliesTo {
			appliesTo[ref] = true
		}
		for _, candidate := range projection.Domain("assurance") {
			verification, ok := candidate.Value.(model.ControlVerification)
			if ok && appliesTo[verification.ControlRef] && candidate.Provenance.Alias == entity.Provenance.Alias {
				row.Verifications = append(row.Verifications, candidate.QualifiedID)
			}
		}
		row.Verifications = textutil.UniqueSorted(row.Verifications)
		switch {
		case len(row.Verifications) > 0:
			row.Status = "verified"
		case row.DelegatedTo != nil:
			row.Status = "allocated"
		default:
			row.Status = "published"
		}
		matrix.Requirements = append(matrix.Requirements, row)
		matrix.Summary.Requirements++
		if len(row.Verifications) > 0 {
			matrix.Summary.Verified++
			matrix.Summary.Tested++
			matrix.Summary.TestLinked++
		}
		if row.DelegatedTo != nil {
			matrix.Summary.Delegated++
		}
	}
	sort.SliceStable(matrix.Requirements, func(i, j int) bool {
		return matrix.Requirements[i].ID < matrix.Requirements[j].ID
	})
}

// CSV renders the matrix as one row per requirement for spreadsheet/diff consumption.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (m Matrix) CSV() []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"requirement", "status", "units", "code", "verifications", "delegatedTo", "provenance", "result"})
	for _, r := range m.Requirements {
		code := make([]string, 0, len(r.Code))
		for _, c := range r.Code {
			if c.Line > 0 {
				code = append(code, fmt.Sprintf("%s:%d", c.Path, c.Line))
			} else {
				code = append(code, c.Path)
			}
		}
		delegated := ""
		if r.DelegatedTo != nil {
			delegated = r.DelegatedTo.Subsystem + "/" + r.DelegatedTo.TargetRequirement
		}
		provenance := ""
		if r.Provenance != nil {
			provenance = r.Provenance.Alias + ":" + r.Provenance.Publication + "@" + r.Provenance.Version
		}
		_ = w.Write([]string{
			r.ID, r.Status, strings.Join(r.Units, " "), strings.Join(code, " "),
			strings.Join(r.Verifications, " "), delegated, provenance, r.Result,
		})
	}
	w.Flush()
	return buf.Bytes()
}
