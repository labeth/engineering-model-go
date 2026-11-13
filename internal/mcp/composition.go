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
	"encoding/json"
	"fmt"
	"os"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/validate"
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
		"diagnostics":    diagRows(engmodel.ApplyTierPolicy(*s.bundle, res.Diagnostics)),
	}, nil
}

// traceMatrix builds the traceability matrix for the loaded model: per-requirement
// implementation, verification, and delegation status, with orphans and dangling links.
// Delegated rows carry the target requirement's own trace status
// (delegatedStatus). With summaryOnly it returns only the summary and diagnostics.
// A requirement with formal (Quint) evidence also carries formal (each
// result with machine, invariant, mode and sha256) and evidenceMethods.
// TRLC-LINKS: REQ-EMG-026, REQ-EMG-030, REQ-EMG-061, REQ-EMG-067, REQ-EMG-088, REQ-EMG-105, REQ-EMG-148
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
			"delegated":    matrix.Summary.Delegated,
			// Delegated rows whose target requirement is implemented or
			// test linked in the target repository.
			"delegatedImplemented": matrix.Summary.DelegatedImplemented,
			"delegatedTestLinked":  matrix.Summary.DelegatedTestLinked,
			"orphan":               matrix.Summary.Orphan,
			"dangling":             matrix.Summary.DanglingLinks,
			// Test-linked requirements whose every linked test is
			// environment gated (skips unless a variable is set).
			"gatedOnly":             matrix.Summary.GatedOnly,
			"gatedOnlyRequirements": nonNilStrings(matrix.Summary.GatedOnlyRequirements),
			// Requirement links: the links that need review (version
			// mismatch, broken, claimed) and local requirements without an
			// upward trace.
			"linksNeedReview": matrix.Summary.LinksNeedReview, "linksVersionMismatch": matrix.Summary.LinksVersionMismatch,
			"linksBroken":  matrix.Summary.LinksBroken,
			"linksClaimed": matrix.Summary.LinksClaimed,
			"unparented":   matrix.Summary.Unparented,
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
		if r.Gated {
			row["gated"] = true
			row["gates"] = r.Gates
		}
		row["parents"] = nonNilStrings(r.Parents)
		row["children"] = nonNilStrings(r.Children)
		if len(r.Links) > 0 {
			row["links"] = r.Links
		}
		if len(r.Formal) > 0 {
			row["formal"] = r.Formal
		}
		if len(r.EvidenceMethods) > 0 {
			row["evidenceMethods"] = r.EvidenceMethods
		}
		if r.SatisfiedBy != "" {
			row["satisfiedBy"] = r.SatisfiedBy
			row["linksImplemented"] = r.LinksImplemented
			row["linksTestLinked"] = r.LinksTestLinked
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
// TRLC-LINKS: REQ-EMG-016, REQ-EMG-168
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func diagRows(diags []validate.Diagnostic) []map[string]any {
	out := make([]map[string]any, 0, len(diags))
	for _, d := range diags {
		row := map[string]any{
			"code":     d.Code,
			"severity": string(d.Severity),
			"message":  d.Message,
			"path":     d.Path,
		}
		// The life-cycle tier fields are omitted when unset, like their
		// JSON tags, so a model without lifecycle answers as before.
		if d.Deferred {
			row["deferred"] = true
		}
		if d.RequiredFrom != "" {
			row["requiredFrom"] = d.RequiredFrom
		}
		if d.PoamRef != "" {
			row["poamRef"] = d.PoamRef
		}
		// The remedy of a required switch: the skill or command that fixes it.
		if d.Remedy != "" {
			row["remedy"] = d.Remedy
		}
		out = append(out, row)
	}
	return out
}

// nonNilStrings returns list, or an empty list for nil, so JSON renders [].
// TRLC-LINKS: REQ-EMG-088, REQ-EMG-105
func nonNilStrings(list []string) []string {
	if list == nil {
		return []string{}
	}
	return list
}

// tierGap returns the life-cycle tier gap of the loaded model towards
// target (engmodel.TierGapFromFilesWith), as JSON-shaped data (the tool
// response is scrubbed of absolute paths like every other).
// TRLC-LINKS: REQ-EMG-134
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) tierGap(target string) (map[string]any, error) {
	if strings.TrimSpace(target) == "" {
		return nil, fmt.Errorf("tier.gap needs target: poc, prototype, mvp, product or operational")
	}
	report, err := engmodel.TierGapFromFilesWith(s.modelPath, s.requirementsPath, "", target, engmodel.TraceOptions{Resolution: resolution()})
	if err != nil {
		return nil, err
	}
	data, err := json.Marshal(report)
	if err != nil {
		return nil, err
	}
	var out map[string]any
	if err := json.Unmarshal(data, &out); err != nil {
		return nil, err
	}
	return out, nil
}
