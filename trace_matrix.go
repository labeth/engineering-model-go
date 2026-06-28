// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package engmodel

// Trace integrity and the machine-readable traceability matrix. Code trace markers
// (TRLC-LINKS to a requirement, ENGMODEL-LINKS to a model element) are existence-checked
// against the model — not only shape-checked — and code is attributed to the model whose
// architecture.yml is its nearest enclosing root, so a parent model never validates or
// counts a child model's links. The consolidated trace is also emitted as a matrix.

import (
	"bytes"
	"encoding/csv"
	"fmt"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"

	"github.com/labeth/engineering-model-go/model"
	"github.com/labeth/engineering-model-go/validate"
)

// TraceMatrix is the consolidated, machine-readable traceability matrix for one model.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type TraceMatrix struct {
	Model         string         `json:"model"`
	Requirements  []TraceRow     `json:"requirements"`
	DanglingLinks []DanglingLink `json:"danglingLinks"`
	Orphans       []string       `json:"orphanRequirements"`
	Summary       TraceSummary   `json:"summary"`
}

// TraceRow is one requirement and everything that traces to it.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
type TraceRow struct {
	ID            string           `json:"id"`
	Text          string           `json:"text,omitempty"`
	Units         []string         `json:"units,omitempty"`
	Code          []TraceCodeRef   `json:"code,omitempty"`
	Verifications []string         `json:"verifications,omitempty"`
	DelegatedTo   *TraceDelegation `json:"delegatedTo,omitempty"`
	Status        string           `json:"status"`
}

// TraceCodeRef is a code symbol that implements a requirement.
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
type TraceCodeRef struct {
	Symbol string `json:"symbol"`
	Path   string `json:"path"`
	Line   int    `json:"line,omitempty"`
}

// TraceDelegation is the downward allocation of a requirement onto a subsystem.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type TraceDelegation struct {
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

// TraceSummary is the headline coverage roll-up.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
type TraceSummary struct {
	Requirements  int `json:"requirements"`
	Implemented   int `json:"implemented"`
	Verified      int `json:"verified"`
	Delegated     int `json:"delegated"`
	Orphan        int `json:"orphan"`
	DanglingLinks int `json:"danglingLinks"`
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

// knownModelIDs is every id the model defines (catalog, authored architecture, and
// requirements) — the oracle against which an ENGMODEL-LINKS target is resolved.
// TRLC-LINKS: REQ-EMG-028
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-MODEL-LOADER
func knownModelIDs(bundle model.Bundle, requirements model.RequirementsDocument) map[string]bool {
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

// effectiveCodeRoots mirrors inferCodeItems' root derivation: the explicit code-root
// option plus the model's inferenceHints.codeSources. Scoping must use the same roots
// the scan used, otherwise items scanned via inferenceHints (with no --code-root) escape it.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-MODEL-LOADER
func effectiveCodeRoots(bundle model.Bundle, codeRootOption string) []string {
	baseDir := filepath.Dir(bundle.ArchitecturePath)
	roots := make([]string, 0, len(bundle.Architecture.InferenceHints.CodeSources)+1)
	if strings.TrimSpace(codeRootOption) != "" {
		roots = append(roots, codeRootOption)
	}
	for _, src := range bundle.Architecture.InferenceHints.CodeSources {
		roots = append(roots, resolveSourcePath(baseDir, src))
	}
	return uniqueExistingDirs(roots)
}

// scopeCodeToModel keeps only code that belongs to the model rooted at modelDir: a file
// belongs to the model whose architecture.yml is its nearest enclosing directory. This
// stops a parent model (scanned with a broad code root) from claiming a nested model's code.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-SYSTEM-COMPOSITION
func scopeCodeToModel(items []inferredCodeItem, roots []string, modelDir string) []inferredCodeItem {
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
		_ = filepath.WalkDir(ra, func(p string, d fs.DirEntry, werr error) error {
			if werr != nil {
				return nil
			}
			if d.IsDir() {
				if skipScanDir(d.Name()) {
					return filepath.SkipDir
				}
				return nil
			}
			if d.Name() == "architecture.yml" {
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
	out := make([]inferredCodeItem, 0, len(items))
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

// splitSourceLine splits an inferred code item Source ("path:line") into path and line.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
func splitSourceLine(source string) (string, int) {
	idx := strings.LastIndex(source, ":")
	if idx < 0 {
		return source, 0
	}
	if n, err := strconv.Atoi(source[idx+1:]); err == nil {
		return source[:idx], n
	}
	return source, 0
}

// validateTraceIntegrity rejects dangling code trace links (a TRLC-LINKS or ENGMODEL-LINKS
// that resolves to nothing in the model) and reports orphan requirements (traced by no
// code, verification, or delegation). The code is assumed already scoped to this model.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-029
// ENGMODEL-LINKS: FU-VALIDATION-ENGINE, FU-ALLOCATION-TRACE, CTRL-TRACEABILITY-COVERAGE
func validateTraceIntegrity(bundle model.Bundle, requirements model.RequirementsDocument, scopedCode []inferredCodeItem, verification []inferredVerificationCheck, delegationsByReq map[string][]MaterializedAllocation) []validate.Diagnostic {
	var diags []validate.Diagnostic
	reqIDs := requirementIDSet(requirements)
	modelIDs := knownModelIDs(bundle, requirements)
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
func buildTraceMatrix(bundle model.Bundle, requirements model.RequirementsDocument, scopedCode []inferredCodeItem, verification []inferredVerificationCheck, delegationsByReq map[string][]MaterializedAllocation) TraceMatrix {
	reqIDs := requirementIDSet(requirements)
	modelIDs := knownModelIDs(bundle, requirements)
	codeByReq := map[string][]TraceCodeRef{}
	var dangling []DanglingLink
	for _, it := range scopedCode {
		path, line := splitSourceLine(it.Source)
		for _, raw := range it.Implements {
			link := strings.TrimSpace(raw)
			if link == "" {
				continue
			}
			if reqIDs[link] {
				codeByReq[link] = append(codeByReq[link], TraceCodeRef{Symbol: it.Element, Path: path, Line: line})
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

	matrix := TraceMatrix{Model: strings.TrimSpace(bundle.Architecture.Model.ID)}
	for _, r := range requirements.Requirements {
		id := strings.TrimSpace(r.ID)
		row := TraceRow{ID: id, Text: strings.TrimSpace(r.Text), Units: uniqueSorted(r.AppliesTo)}
		row.Code = codeByReq[id]
		row.Verifications = uniqueSorted(verByReq[id])
		if d := delegationsByReq[id]; len(d) > 0 {
			row.DelegatedTo = &TraceDelegation{Subsystem: d[0].Subsystem, Target: d[0].Target, TargetRequirement: d[0].TargetRef}
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
			matrix.Summary.Verified++
		}
		if row.DelegatedTo != nil {
			matrix.Summary.Delegated++
		}
		if row.Status == "orphan" {
			matrix.Summary.Orphan++
		}
	}
	matrix.Summary.DanglingLinks = len(dangling)
	return matrix
}

// BuildTraceMatrixFromFiles loads a model, scans its code root, scopes the code to the
// model, validates trace integrity, and returns the matrix together with diagnostics.
// TRLC-LINKS: REQ-EMG-028, REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-MODEL-LOADER, FU-CODEMAP-INFERENCE
func BuildTraceMatrixFromFiles(modelPath, requirementsPath, codeRoot string) (TraceMatrix, []validate.Diagnostic, error) {
	bundle, err := model.LoadBundle(modelPath)
	if err != nil {
		return TraceMatrix{}, nil, err
	}
	requirements, err := model.LoadRequirements(requirementsPath)
	if err != nil {
		return TraceMatrix{}, nil, err
	}
	resolvedRoot := strings.TrimSpace(codeRoot)
	if resolvedRoot != "" && !filepath.IsAbs(resolvedRoot) {
		resolvedRoot = filepath.Join(filepath.Dir(modelPath), resolvedRoot)
	}
	inferredCode, codeDiags := inferCodeItems(bundle, resolvedRoot)
	inferredVerification, verDiags := inferVerificationChecks(bundle, requirements, inferredCode, resolvedRoot)
	diags := append([]validate.Diagnostic(nil), codeDiags...)
	diags = append(diags, verDiags...)

	delegationsByReq := map[string][]MaterializedAllocation{}
	if HasComposition(bundle) {
		if res, derr := GenerateCompositionFromFile(bundle.ArchitecturePath); derr == nil {
			for _, m := range res.Allocations {
				rid := strings.TrimSpace(m.Requirement)
				delegationsByReq[rid] = append(delegationsByReq[rid], m)
			}
		} else {
			diags = append(diags, validate.Diagnostic{
				Code: "composition.resolve_failed", Severity: validate.SeverityWarning,
				Message: fmt.Sprintf("composition could not be resolved (delegated requirements may appear orphan): %v", derr),
				Path:    bundle.ArchitecturePath,
			})
		}
	}
	scoped := scopeCodeToModel(inferredCode, effectiveCodeRoots(bundle, resolvedRoot), filepath.Dir(bundle.ArchitecturePath))
	diags = append(diags, validateTraceIntegrity(bundle, requirements, scoped, inferredVerification, delegationsByReq)...)
	matrix := buildTraceMatrix(bundle, requirements, scoped, inferredVerification, delegationsByReq)
	return matrix, validate.SortDiagnostics(diags), nil
}

// CSV renders the matrix as one row per requirement for spreadsheet/diff consumption.
// TRLC-LINKS: REQ-EMG-030
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func (m TraceMatrix) CSV() []byte {
	var buf bytes.Buffer
	w := csv.NewWriter(&buf)
	_ = w.Write([]string{"requirement", "status", "units", "code", "verifications", "delegatedTo"})
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
		_ = w.Write([]string{
			r.ID, r.Status, strings.Join(r.Units, " "), strings.Join(code, " "),
			strings.Join(r.Verifications, " "), delegated,
		})
	}
	w.Flush()
	return buf.Bytes()
}
