// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// Formal (Quint) evidence in the trace (ADR-EMG-042, REQ-EMG-146). engquint
// check writes test-results/quint-formal.json; the codescan inferred
// verification turns it into one check per file (Method "analysis", Kind
// "formal") whose results are pass, fail or, for a stale entry, not-run with a
// verification.formal_stale warning. That check carries no machine, invariant
// or hash, so this file reads the same evidence files again to give the matrix
// rows that detail (Row.Formal); the MCP tools read it from the rows. The
// reader, the stale rule and the status mapping live once in
// internal/codescan/formal_evidence_read.go; the Gemara exporter and the MCP
// tools use the same one.
//
// The roll-up rule of a requirement (the Gemara log and the MCP tools apply
// the same order):
//   - a failing current result makes the requirement failed (Row.Result
//     "failed", diagnostic verification.failed), whatever else passes: a failing
//     invariant is a failure, never "unverified";
//   - a passing current result is analysis evidence: the formal check is a
//     linked verification (status "verified" for a requirement without code),
//     Row.Result is "passed" unless a test fails, and Row.EvidenceMethods lists
//     "analysis";
//   - a stale (or not-run) result is no evidence: it does not link the formal
//     check, does not make a requirement verified or failed and does not
//     downgrade a requirement that other passing evidence (a test, or another
//     current invariant) verifies. It stays visible in Row.Formal as "stale".

import (
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/codescan"
)

// Formal result values: the requirement-level and per-entry result.
const (
	// formalPass is a current passing result: analysis evidence.
	formalPass = codescan.FormalPass
	// formalFailed is a current failing result.
	formalFailed = codescan.FormalFailed
	// formalStaleResult is a result whose module changed (or is missing, or is
	// not a trusted generated module) since it was produced: no evidence.
	formalStaleResult = codescan.FormalStale
	// formalNotRun is any other recorded status: no evidence.
	formalNotRun = codescan.FormalNotRun
)

// formalResult is one entry of a formal evidence file with its result
// evaluated against the module on disk. JSON tags make it the matrix row's
// formal entry and the entry the MCP tools read from the rows.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-146
type formalResult struct {
	Requirement string `json:"requirement"`
	Machine     string `json:"machine,omitempty"`
	Invariant   string `json:"invariant,omitempty"`
	// Mode is run, verify or run+verify as engquint recorded it.
	Mode string `json:"mode,omitempty"`
	// Result is pass, failed, stale or not-run.
	Result    string `json:"result"`
	QntFile   string `json:"qntFile,omitempty"`
	QntSha256 string `json:"qntSha256,omitempty"`
	// Evidence is the evidence file, relative to the model root.
	Evidence string `json:"evidence,omitempty"`
	// Notes holds the recorded notes, or the reason a result is stale.
	Notes string `json:"notes,omitempty"`
}

// readFormalResults reads the evidence files (paths relative to baseDir) with
// the shared codescan reader (codescan.ReadFormalEvidence) and converts the
// entries to matrix form: a current entry is pass or failed, a stale one is
// stale with the reason in Notes, any other status is not-run. An unreadable
// or invalid file contributes nothing (the codescan reports it as
// verification.formal_unreadable). The result is sorted by requirement,
// machine, invariant and file.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func readFormalResults(baseDir string, files []string) []formalResult {
	var out []formalResult
	for _, e := range codescan.ReadFormalEvidence(baseDir, files) {
		out = append(out, formalResult{Requirement: e.Requirement, Machine: e.Machine, Invariant: e.Invariant, Mode: e.Mode,
			Result: e.Result, QntFile: e.QntFile, QntSha256: e.QntSha256, Evidence: e.Evidence, Notes: e.Notes})
	}
	return out
}

// formalRequirementResult rolls the entries of one requirement into one
// result: failed over pass over stale over not-run; empty without entries.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-146, REQ-EMG-148
func formalRequirementResult(entries []formalResult) string {
	rank := map[string]int{formalNotRun: 1, formalStaleResult: 2, formalPass: 3, formalFailed: 4}
	best := ""
	for _, e := range entries {
		if rank[e.Result] > rank[best] {
			best = e.Result
		}
	}
	return best
}

// formalByRequirement groups results by requirement.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-146
func formalByRequirement(results []formalResult) map[string][]formalResult {
	out := map[string][]formalResult{}
	for _, r := range results {
		out[r.Requirement] = append(out[r.Requirement], r)
	}
	return out
}

// formalCheckFiles returns the formal evidence files a check was built from:
// its Evidence entries named quint-formal.json. Only checks with Method
// "analysis" have any.
// TRLC-LINKS: REQ-EMG-146
func formalCheckFiles(c codescan.VerificationCheck) []string {
	if c.Method != "analysis" {
		return nil
	}
	var files []string
	for _, ev := range c.Evidence {
		if strings.EqualFold(filepath.Base(filepath.ToSlash(ev)), codescan.FormalEvidenceFile) {
			files = append(files, ev)
		}
	}
	return files
}

// formalResultsOf reads the entries of every formal check in verification.
// TRLC-LINKS: REQ-EMG-146
func formalResultsOf(baseDir string, verification []codescan.VerificationCheck) []formalResult {
	var files []string
	seen := map[string]bool{}
	for _, c := range verification {
		for _, f := range formalCheckFiles(c) {
			if !seen[f] {
				seen[f] = true
				files = append(files, f)
			}
		}
	}
	if len(files) == 0 {
		return nil
	}
	sort.Strings(files)
	return readFormalResults(baseDir, files)
}

// effectiveVerification applies the roll-up rule to the inferred checks: a
// requirement whose entries in a formal check are all stale or not-run is
// removed from that check's Verifies and Results, since it is no evidence and
// must not make the requirement verified. Checks without a formal method, and
// models without formal evidence, are returned as they are (the same slice);
// a changed check is copied, never mutated.
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-CODEMAP-INFERENCE
// TRLC-LINKS: REQ-EMG-146
func effectiveVerification(baseDir string, verification []codescan.VerificationCheck) []codescan.VerificationCheck {
	var out []codescan.VerificationCheck
	for i, c := range verification {
		files := formalCheckFiles(c)
		if len(files) == 0 {
			continue
		}
		byReq := formalByRequirement(readFormalResults(baseDir, files))
		drop := map[string]bool{}
		for _, req := range c.Verifies {
			id := strings.TrimSpace(req)
			if r := formalRequirementResult(byReq[id]); r == formalStaleResult || r == formalNotRun {
				drop[id] = true
			}
		}
		if len(drop) == 0 {
			continue
		}
		if out == nil {
			out = append([]codescan.VerificationCheck(nil), verification...)
		}
		kept := c
		kept.Verifies, kept.Results = nil, nil
		for _, req := range c.Verifies {
			if !drop[strings.TrimSpace(req)] {
				kept.Verifies = append(kept.Verifies, req)
			}
		}
		for _, res := range c.Results {
			if !drop[strings.TrimSpace(res.Requirement)] {
				kept.Results = append(kept.Results, res)
			}
		}
		out[i] = kept
	}
	if out == nil {
		return verification
	}
	return out
}
