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

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/validate"
)

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
	res, err := engmodel.GenerateCompositionFromFile(s.bundle.ArchitecturePath)
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
				"id":       c.SubsystemID,
				"model":    c.Bundle.Architecture.Model.ID,
				"provides": provides,
				"requires": requires,
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
	rootID := ""
	if res.Root != nil {
		rootID = res.Root.Bundle.Architecture.Model.ID
	}
	return map[string]any{
		"hasComposition": true,
		"root":           rootID,
		"subsystems":     subsystems,
		"allocations":    allocations,
		"diagnostics":    diagRows(res.Diagnostics),
	}, nil
}

// traceMatrix builds the traceability matrix for the loaded model: per-requirement
// implementation, verification, and delegation status, with orphans and dangling links.
// TRLC-LINKS: REQ-EMG-026, REQ-EMG-030
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) traceMatrix() (map[string]any, error) {
	matrix, diags, err := engmodel.BuildTraceMatrixFromFiles(s.modelPath, s.requirementsPath, "")
	if err != nil {
		return nil, err
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
		rows = append(rows, row)
	}
	dangling := make([]map[string]any, 0, len(matrix.DanglingLinks))
	for _, d := range matrix.DanglingLinks {
		dangling = append(dangling, map[string]any{"kind": d.Kind, "target": d.Target, "from": d.From})
	}
	return map[string]any{
		"model": matrix.Model,
		"summary": map[string]any{
			"requirements": matrix.Summary.Requirements,
			"implemented":  matrix.Summary.Implemented,
			"verified":     matrix.Summary.Verified,
			"delegated":    matrix.Summary.Delegated,
			"orphan":       matrix.Summary.Orphan,
			"dangling":     matrix.Summary.DanglingLinks,
		},
		"requirements":  rows,
		"orphans":       matrix.Orphans,
		"danglingLinks": dangling,
		"diagnostics":   diagRows(diags),
	}, nil
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
