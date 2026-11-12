// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package gemara

// Requirement-keyed formal entries of the Evaluation Log (ADR-EMG-042,
// REQ-EMG-147). The model has no requirement-to-control mapping for machine
// invariants, so a Quint result is not filed under a control: each requirement
// with formal evidence gets one ControlEvaluation whose control entry is the
// requirement itself (reference ENGMOD-REQUIREMENTS) and one AssessmentLog per
// invariant result. The schema fields used are the ones control evaluations
// already use; no field is added.
//
//   - a passing, non-stale result is Passed and a failing one Failed;
//   - a stale result (the module's sha256 no longer matches) or any other
//     status is NotRun, with the reason in the message: it is not evaluated and
//     never reported as passed;
//   - the machine, invariant, mode and the result hash (qntSha256) are in the
//     message and in the evidence payload.
//
// A model without formal evidence adds nothing, so its log is unchanged.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	gemaraproj "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// gemaraRefRequirements is the mapping reference of the requirement-keyed
// formal entries.
const gemaraRefRequirements = "ENGMOD-REQUIREMENTS"

// gemaraFormalStep is a record-only assessment step for a formal result; its
// function name is serialized as the (non-empty) step string.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func gemaraFormalStep(payload interface{}) (gemaraproj.Result, string, gemaraproj.ConfidenceLevel) {
	return gemaraproj.Passed, "evaluated from a sha-verified formal result", gemaraproj.Medium
}

// formalResultToGemara maps a trace formal result to a Gemara result: pass
// and failed map directly; stale and not-run are NotRun.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func formalResultToGemara(result string) gemaraproj.Result {
	switch result {
	case formalPass:
		return gemaraproj.Passed
	case formalFailed:
		return gemaraproj.Failed
	default:
		return gemaraproj.NotRun
	}
}

// appendFormalEvaluations appends one requirement-keyed ControlEvaluation for
// each requirement with a formal result found under baseDir (and beside
// codeRoot), registers the ENGMOD-REQUIREMENTS mapping reference and returns
// the updated aggregate result. It does nothing when there is no formal
// evidence.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, FU-ALLOCATION-TRACE
func appendFormalEvaluations(log *gemaraproj.EvaluationLog, aggregate gemaraproj.Result, baseDir, codeRoot, start string) gemaraproj.Result {
	if baseDir == "" {
		return aggregate
	}
	results := readFormalResults(baseDir, formalEvidenceFiles(baseDir, codeRoot))
	if len(results) == 0 {
		return aggregate
	}
	log.Metadata.MappingReferences = append(log.Metadata.MappingReferences,
		gemaraproj.MappingReference{Id: gemaraRefRequirements, Title: "Requirements", Version: Version})
	byReq := map[string][]formalResult{}
	for _, r := range results {
		byReq[r.Requirement] = append(byReq[r.Requirement], r)
	}
	var order []string
	seen := map[string]bool{}
	for _, r := range results {
		if !seen[r.Requirement] {
			seen[r.Requirement] = true
			order = append(order, r.Requirement)
		}
	}
	for _, req := range order {
		ceResult := gemaraproj.NotRun
		var logs []*gemaraproj.AssessmentLog
		for i, r := range byReq[req] {
			res := formalResultToGemara(r.Result)
			al := &gemaraproj.AssessmentLog{
				Requirement:     gemaraproj.EntryMapping{ReferenceId: gemaraRefRequirements, EntryId: textutil.Fallback(r.Invariant, req)},
				Description:     fmt.Sprintf("Assessment of %s via analysis: Quint invariant %s of machine %s.", req, textutil.Fallback(r.Invariant, "(unnamed)"), textutil.Fallback(r.Machine, "(unnamed)")),
				Result:          res,
				Message:         formalMessage(r),
				Applicability:   []string{"all-systems"},
				Steps:           []gemaraproj.AssessmentStep{gemaraFormalStep},
				Start:           gemaraproj.Datetime(start),
				ConfidenceLevel: confidenceForResult(res),
				Evidence: []gemaraproj.Evidence{{
					Id:          fmt.Sprintf("%s-FORMAL-%d", req, i+1),
					Type:        gemaraproj.EvidenceType("FormalVerification"),
					CollectedAt: gemaraproj.Datetime(start),
					Description: fmt.Sprintf("Quint result for %s: %s, sha256 %s.", textutil.Fallback(r.Invariant, req), r.Result, textutil.Fallback(r.QntSha256, "(none)")),
					Payload: map[string]string{
						"machine": r.Machine, "invariant": r.Invariant, "mode": r.Mode, "method": "analysis",
						"qntFile": r.QntFile, "qntSha256": r.QntSha256, "result": r.Result,
					},
				}},
			}
			if res == gemaraproj.Failed {
				al.Recommendation = "Fix the model or the invariant and rerun engquint check."
			}
			if res == gemaraproj.NotRun {
				al.Recommendation = "Rerun engquint check to refresh the result."
			}
			logs = append(logs, al)
			ceResult = gemaraproj.UpdateAggregateResult(ceResult, res)
		}
		log.Evaluations = append(log.Evaluations, &gemaraproj.ControlEvaluation{
			Name:           "Formal verification of " + req,
			Result:         ceResult,
			Message:        fmt.Sprintf("%d formal result(s) recorded for requirement %s.", len(logs), req),
			Control:        gemaraproj.EntryMapping{ReferenceId: gemaraRefRequirements, EntryId: req},
			AssessmentLogs: logs,
		})
		aggregate = gemaraproj.UpdateAggregateResult(aggregate, ceResult)
	}
	return aggregate
}

// formalMessage is the assessment message of a formal result: machine,
// invariant, mode, the result hash and, for a stale or not-run result, the
// reason it was not evaluated.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func formalMessage(r formalResult) string {
	parts := []string{fmt.Sprintf("machine %s, invariant %s", textutil.Fallback(r.Machine, "(unnamed)"), textutil.Fallback(r.Invariant, "(unnamed)"))}
	if r.Mode != "" {
		parts = append(parts, "mode "+r.Mode)
	}
	if r.QntSha256 != "" {
		parts = append(parts, "result hash "+r.QntSha256)
	}
	msg := strings.Join(parts, ", ")
	switch r.Result {
	case formalStale:
		return "Not evaluated, " + r.Notes + " (" + msg + ")."
	case formalNotRun:
		return "Not evaluated: the recorded status is not a pass or a fail (" + msg + ")."
	}
	return "Formal result " + r.Result + ": " + msg + "."
}

// Formal result values, as the trace matrix reports them.
const (
	formalPass   = "pass"
	formalFailed = "failed"
	formalStale  = "stale"
	formalNotRun = "not-run"
)

// formalEvidenceName is the file engquint check writes into test-results/.
const formalEvidenceName = "quint-formal.json"

// formalResult is one evidence entry evaluated against the module on disk.
// internal/export/gemara may not import internal/trace (the layering), so
// this mirrors the reader of internal/trace/formal_evidence.go: the same
// files, the same stale rule, the same four results.
type formalResult struct {
	Requirement, Machine, Invariant, Mode string
	// Result is pass, failed, stale or not-run.
	Result, QntFile, QntSha256, Notes string
}

// formalEvidenceFiles lists the formal evidence files of a model, relative to
// baseDir and sorted: under <baseDir>/test-results and, when codeRoot is set,
// under test-results beside it (the result roots the verification inference
// reads).
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func formalEvidenceFiles(baseDir, codeRoot string) []string {
	roots := []string{filepath.Join(baseDir, "test-results")}
	if root := strings.TrimSpace(codeRoot); root != "" {
		if !filepath.IsAbs(root) {
			root = filepath.Join(baseDir, root)
		}
		roots = append(roots, filepath.Join(filepath.Dir(root), "test-results"))
	}
	seen := map[string]bool{}
	var files []string
	for _, root := range roots {
		_ = filepath.WalkDir(root, func(path string, d os.DirEntry, err error) error {
			if err != nil || d.IsDir() || !strings.EqualFold(d.Name(), formalEvidenceName) {
				return nil
			}
			rel, relErr := filepath.Rel(baseDir, path)
			if relErr != nil {
				rel = path
			}
			if rel = filepath.ToSlash(rel); !seen[rel] {
				seen[rel] = true
				files = append(files, rel)
			}
			return nil
		})
	}
	sort.Strings(files)
	return files
}

// readFormalResults reads the evidence files and evaluates each entry: a
// current entry is pass or failed, one whose module hash no longer matches (or
// whose module, path or hash is missing) is stale with the reason in Notes,
// any other status is not-run. An unreadable file contributes nothing. The
// result is sorted by requirement, machine and invariant.
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func readFormalResults(baseDir string, files []string) []formalResult {
	var out []formalResult
	hashes := map[string]string{}
	for _, rel := range files {
		data, err := os.ReadFile(filepath.Join(baseDir, filepath.FromSlash(rel)))
		if err != nil {
			continue
		}
		var file struct {
			Results []struct {
				Requirement, Status, Machine, Invariant, Mode, Notes string
				QntFile                                              string `json:"qntFile"`
				QntSha256                                            string `json:"qntSha256"`
			} `json:"results"`
		}
		if json.Unmarshal(data, &file) != nil {
			continue
		}
		for _, e := range file.Results {
			req := strings.TrimSpace(e.Requirement)
			if req == "" {
				continue
			}
			r := formalResult{Requirement: req, Machine: strings.TrimSpace(e.Machine), Invariant: strings.TrimSpace(e.Invariant),
				Mode: strings.TrimSpace(e.Mode), QntFile: strings.TrimSpace(e.QntFile),
				QntSha256: strings.ToLower(strings.TrimSpace(e.QntSha256)), Notes: strings.TrimSpace(e.Notes)}
			switch codescan.NormalizeResultStatus(e.Status) {
			case "pass":
				r.Result = formalPass
			case "fail", "blocked", "partial", "flaky":
				r.Result = formalFailed
			default:
				r.Result = formalNotRun
			}
			if reason := formalStaleReason(baseDir, r, hashes); reason != "" {
				r.Result, r.Notes = formalStale, "stale: "+reason+"; rerun engquint check"
			}
			out = append(out, r)
		}
	}
	sort.SliceStable(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Requirement != b.Requirement {
			return a.Requirement < b.Requirement
		}
		if a.Machine != b.Machine {
			return a.Machine < b.Machine
		}
		return a.Invariant < b.Invariant
	})
	return out
}

// formalStaleReason returns why a result is stale, or "" when the sha256 of
// its module still matches (the rule of internal/codescan).
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-147
func formalStaleReason(baseDir string, r formalResult, hashes map[string]string) string {
	if r.QntFile == "" || r.QntSha256 == "" {
		return "the entry records no qntFile or qntSha256"
	}
	if filepath.IsAbs(r.QntFile) {
		return fmt.Sprintf("qntFile %s is not relative to the model root", r.QntFile)
	}
	got, seen := hashes[r.QntFile]
	if !seen {
		got = "missing"
		if data, err := os.ReadFile(filepath.Join(baseDir, filepath.FromSlash(r.QntFile))); err == nil {
			sum := sha256.Sum256(data)
			got = hex.EncodeToString(sum[:])
		}
		hashes[r.QntFile] = got
	}
	switch {
	case got == "missing":
		return fmt.Sprintf("module %s is missing", r.QntFile)
	case got != r.QntSha256:
		return fmt.Sprintf("the sha256 of %s no longer matches the one the result was produced from", r.QntFile)
	}
	return ""
}
