// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// Requirement-level formal (Quint) results for the verification tools
// (ADR-EMG-042, REQ-EMG-148). The results are read live from the formal
// evidence file engquint check writes (test-results/quint-formal.json), the
// way trace.matrix reads it, so a result turns stale the moment its module
// changes. A model without formal evidence adds nothing to any response.

import (
	"fmt"
	"sort"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/codescan"
)

// formalInvariant is one invariant result of a requirement as the tools
// report it.
type formalInvariant struct {
	Requirement string `json:"requirement"`
	Machine     string `json:"machine,omitempty"`
	Invariant   string `json:"invariant,omitempty"`
	// Mode is run, verify or run+verify as engquint recorded it.
	Mode string `json:"mode,omitempty"`
	// Result is pass, failed, stale or not-run.
	Result    string `json:"result"`
	QntFile   string `json:"qntFile,omitempty"`
	QntSha256 string `json:"qntSha256,omitempty"`
	Evidence  string `json:"evidence,omitempty"`
	Notes     string `json:"notes,omitempty"`
}

// formalRequirement is one requirement's formal status: the rolled-up result
// (failed over pass over stale over not-run, the rule of the trace matrix) and
// every invariant result.
type formalRequirement struct {
	Requirement string            `json:"requirement"`
	Result      string            `json:"result"`
	Invariants  []formalInvariant `json:"invariants"`
}

// hasFormalEvidence reports whether the model has a formal evidence file in
// the directories the verification scan reads (codescan.FormalEvidenceFiles,
// with no code root, as the matrix is built below); a model without one skips
// the trace build.
// TRLC-LINKS: REQ-EMG-148
func (s *Server) hasFormalEvidence() bool {
	if s.bundle == nil {
		return false
	}
	return len(codescan.FormalEvidenceFiles(codescan.ModelRootDir(*s.bundle), formalCodeRoot)) > 0
}

// formalCodeRoot is the code root the tools build the trace matrix with: none,
// so the result directories scanned are the model root's test-results. The
// evidence check and the matrix build use this one value.
const formalCodeRoot = ""

// formalRollUp rolls invariant results into one requirement result: failed
// over pass over stale over not-run.
// TRLC-LINKS: REQ-EMG-148
func formalRollUp(entries []formalInvariant) string {
	rank := map[string]int{"not-run": 1, "stale": 2, "pass": 3, "failed": 4}
	best := ""
	for _, e := range entries {
		if rank[e.Result] > rank[best] {
			best = e.Result
		}
	}
	return best
}

// formalRequirements returns the formal status of every model requirement
// with a formal result, sorted by requirement; nil when the model has no
// formal evidence. The results come from the trace matrix rows, so the stale
// rule is the one the matrix applies (the sha256 of the module is compared at
// call time). A failing matrix build is an error, never an empty result: the
// caller must not report "no formal gaps" when verification could not run.
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func (s *Server) formalRequirements() ([]formalRequirement, error) {
	if !s.hasFormalEvidence() {
		return nil, nil
	}
	matrix, _, err := engmodel.BuildTraceMatrixFromFilesWith(s.modelPath, s.requirementsPath, formalCodeRoot, engmodel.TraceOptions{Resolution: resolution()})
	if err != nil {
		return nil, fmt.Errorf("formal evidence found but the trace matrix could not be built: %w", err)
	}
	var out []formalRequirement
	for _, row := range matrix.Requirements {
		if len(row.Formal) == 0 {
			continue
		}
		fr := formalRequirement{Requirement: row.ID}
		for _, f := range row.Formal {
			fr.Invariants = append(fr.Invariants, formalInvariant{Requirement: f.Requirement, Machine: f.Machine, Invariant: f.Invariant, Mode: f.Mode,
				Result: f.Result, QntFile: f.QntFile, QntSha256: f.QntSha256, Evidence: f.Evidence, Notes: f.Notes})
		}
		fr.Result = formalRollUp(fr.Invariants)
		out = append(out, fr)
	}
	sort.Slice(out, func(i, j int) bool { return out[i].Requirement < out[j].Requirement })
	return out, nil
}

// formalStatus is the formal part of verification.status: every requirement
// with a formal result, narrowed to entityID when it names a requirement, a
// machine or an invariant. It is nil when there is nothing to report; the
// error says verification could not run.
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER
func (s *Server) formalStatus(entityID string) ([]formalRequirement, error) {
	all, err := s.formalRequirements()
	if err != nil {
		return nil, err
	}
	entityID = strings.TrimSpace(entityID)
	if entityID == "" {
		return all, nil
	}
	var out []formalRequirement
	for _, fr := range all {
		keep := fr.Requirement == entityID
		for _, inv := range fr.Invariants {
			keep = keep || inv.Machine == entityID || inv.Invariant == entityID
		}
		if keep {
			out = append(out, fr)
		}
	}
	return out, nil
}

// formalGaps is the formal part of verification.gaps: the requirements whose
// formal result is failed, stale or not-run (no passing current result). The
// error says verification could not run, which is not "no gaps".
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER
func (s *Server) formalGaps() ([]formalRequirement, error) {
	all, err := s.formalRequirements()
	if err != nil {
		return nil, err
	}
	var out []formalRequirement
	for _, fr := range all {
		if fr.Result != "pass" {
			out = append(out, fr)
		}
	}
	return out, nil
}
