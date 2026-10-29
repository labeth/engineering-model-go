// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// MCP tools that expose system-of-systems composition and the traceability matrix to
// an agent. composition.resolve surfaces subsystems, their provides/requires contract,
// the materialized parent->subsystem allocations (with the specific delegated subsystem
// requirement), and composition diagnostics. trace.matrix surfaces, per requirement, the
// implementation / verification / delegation status, plus orphan requirements and
// dangling code trace links — the same data the engtrace gate uses, so an agent can see
// exactly what is unimplemented or mis-linked before editing.

import (
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

// resolution is how the MCP server's read tools resolve composition: they
// never write engmod.lock.yml (lock check unless ENGMOD_LOCK is set) and
// work offline from the local module cache (reuse unless ENGMOD_MODULE_CACHE
// or ENGMOD_LOCK is set), so a tool call makes no registry request when the
// cache is warm. Verifying the cache against the registry is the CLI and CI
// check path.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-064
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func resolution() engmodel.ResolutionOptions {
	options := engmodel.ResolutionOptions{}
	if strings.TrimSpace(os.Getenv(engmodel.LockEnv)) == "" {
		options.Lock = engmodel.LockModeCheck
	}
	if strings.TrimSpace(os.Getenv(engmodel.ModuleCacheEnv)) == "" {
		options.ModuleCache = engmodel.ModuleCacheReuse
	}
	return options
}

// ApplyReadOnlyResolutionDefaults sets the process defaults of a model
// server: the lock is checked, never written (ENGMOD_LOCK=check), and cached
// modules are reused without registry lookups (ENGMOD_MODULE_CACHE=reuse),
// unless the environment already sets them. It covers every composing tool,
// the Gemara and OSCAL exporters included. Call it once at startup.
// TRLC-LINKS: REQ-EMG-069, REQ-EMG-064
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func ApplyReadOnlyResolutionDefaults() {
	for key, value := range map[string]string{engmodel.LockEnv: engmodel.LockModeCheck, engmodel.ModuleCacheEnv: engmodel.ModuleCacheReuse} {
		if strings.TrimSpace(os.Getenv(key)) == "" {
			_ = os.Setenv(key, value)
		}
	}
}

// compositionResolve resolves the system-of-systems rooted at the loaded model and
// returns its subsystems, materialized requirement allocations, and diagnostics.
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-020
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION, FU-ALLOCATION-TRACE
func (s *Server) compositionResolve() (map[string]any, error) {
	if !engmodel.HasComposition(*s.bundle) {
		return map[string]any{
			"hasComposition": false,
			"note":           "this model declares no subsystems (composition.subsystems is empty)",
		}, nil
	}
	res, err := engmodel.GenerateCompositionFromFileWith(s.bundle.ManifestPath, resolution())
	if err != nil {
		return nil, err
	}
	subsystems := []map[string]any{}
	if res.Root != nil {
		for _, c := range res.Root.Children {
			provides := []string{}
			for _, p := range c.Bundle.Architecture.Contract.Provides {
				provides = append(provides, p.ID)
			}
			requires := []string{}
			for _, r := range c.Bundle.Architecture.Contract.Requires {
				requires = append(requires, r.ID)
			}
			subsystems = append(subsystems, map[string]any{
				"id":                c.SubsystemID,
				"model":             c.Bundle.Architecture.Model.ID,
				"dependency":        c.Dependency,
				"publication":       c.Publication,
				"modulePath":        c.ModulePath,
				"version":           c.Version,
				"resolvedDirectory": res.DisplayDir(c.Dir),
				"provides":          provides,
				"requires":          requires,
			})
		}
	}
	allocations := []map[string]any{}
	for _, a := range res.Allocations {
		allocations = append(allocations, map[string]any{
			"requirement":       a.Requirement,
			"subsystem":         a.Subsystem,
			"target":            a.Target,
			"targetRequirement": a.TargetRef,
			"resolved":          a.Resolved,
			"note":              a.Note,
		})
	}
	provenance := make([]engmodel.ImportedEntityProvenance, len(res.Provenance))
	for i, p := range res.Provenance {
		p.ResolvedDir = res.DisplayDir(p.ResolvedDir)
		provenance[i] = p
	}
	rootID := ""
	if res.Root != nil {
		rootID = res.Root.Bundle.Architecture.Model.ID
	}
	return map[string]any{
		"hasComposition": true,
		"root":           rootID,
		"subsystems":     subsystems,
		"allocations":    allocations,
		"provenance":     provenance,
		"lock":           res.Lock,
		"diagnostics":    diagRows(res.Diagnostics),
	}, nil
}

// traceMatrix builds the traceability matrix for the loaded model: per-requirement
// implementation, verification, and delegation status, with orphans and dangling links.
// Delegated rows carry the target requirement's own trace status
// (delegatedStatus). With summaryOnly it returns only the summary and diagnostics.
// TRLC-LINKS: REQ-EMG-026, REQ-EMG-030, REQ-EMG-061, REQ-EMG-067
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) traceMatrix(summaryOnly bool) (map[string]any, error) {
	matrix, diags, err := engmodel.BuildTraceMatrixFromFilesWith(s.modelPath, s.requirementsPath, "", engmodel.TraceOptions{Resolution: resolution()})
	if err != nil {
		return nil, err
	}
	out := map[string]any{
		"model": matrix.Model,
		"summary": map[string]any{
			"requirements": matrix.Summary.Requirements,
			"implemented":  matrix.Summary.Implemented,
			"tested":       matrix.Summary.Tested,
			"testLinked":   matrix.Summary.TestLinked,
			"passed":       matrix.Summary.Passed,
			"verified":     matrix.Summary.Verified, // deprecated alias of testLinked
			"delegated":    matrix.Summary.Delegated,
			// Delegated rows whose target requirement is implemented or
			// test linked in the target repository.
			"delegatedImplemented": matrix.Summary.DelegatedImplemented,
			"delegatedTestLinked":  matrix.Summary.DelegatedTestLinked,
			"orphan":               matrix.Summary.Orphan,
			"dangling":             matrix.Summary.DanglingLinks,
		},
		"diagnostics": diagRows(diags),
	}
	if summaryOnly {
		return out, nil
	}
	rows := make([]map[string]any, 0, len(matrix.Requirements))
	for _, r := range matrix.Requirements {
		code := make([]string, 0, len(r.Code))
		for _, c := range r.Code {
			if c.Line > 0 {
				code = append(code, fmt.Sprintf("%s:%d", c.Path, c.Line))
			} else {
				code = append(code, c.Path)
			}
		}
		row := map[string]any{
			"id":            r.ID,
			"status":        r.Status,
			"units":         r.Units,
			"code":          code,
			"verifications": r.Verifications,
		}
		if r.DelegatedTo != nil {
			row["delegatedTo"] = map[string]any{
				"subsystem":         r.DelegatedTo.Subsystem,
				"target":            r.DelegatedTo.Target,
				"targetRequirement": r.DelegatedTo.TargetRequirement,
			}
		}
		if r.DelegatedStatus != nil {
			row["delegatedStatus"] = r.DelegatedStatus
		}
		rows = append(rows, row)
	}
	dangling := make([]map[string]any, 0, len(matrix.DanglingLinks))
	for _, d := range matrix.DanglingLinks {
		dangling = append(dangling, map[string]any{"kind": d.Kind, "target": d.Target, "from": d.From})
	}
	out["requirements"] = rows
	out["orphans"] = matrix.Orphans
	out["danglingLinks"] = dangling
	return out, nil
}

// diagRows renders diagnostics as compact maps for MCP tool responses.
// TRLC-LINKS: REQ-EMG-016
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func diagRows(diags []validate.Diagnostic) []map[string]any {
	out := make([]map[string]any, 0, len(diags))
	for _, d := range diags {
		out = append(out, map[string]any{
			"code":     d.Code,
			"severity": string(d.Severity),
			"message":  d.Message,
			"path":     d.Path,
		})
	}
	return out
}
