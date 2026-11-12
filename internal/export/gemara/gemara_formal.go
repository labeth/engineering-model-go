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
	"fmt"
	"strings"

	gemaraproj "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// gemaraRefRequirements is the mapping reference of the requirement-keyed
// formal entries.
const gemaraRefRequirements = "ENGMOD-REQUIREMENTS"

// The record-only assessment steps of a formal result. The schema requires a
// non-empty step and the SDK serializes only the function name (the exporter
// never runs a step and leaves steps-executed unset, so a consumer reads the
// entry as examined, not tested). One step per outcome keeps the step
// consistent with the result of its entry: a failed entry never names a
// passing step, and a stale one never claims to be evaluated.

// gemaraFormalStepPassed records a current passing formal result.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func gemaraFormalStepPassed(payload interface{}) (gemaraproj.Result, string, gemaraproj.ConfidenceLevel) {
	return gemaraproj.Passed, "evaluated from a sha-verified formal result", gemaraproj.Medium
}

// gemaraFormalStepFailed records a current failing formal result.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func gemaraFormalStepFailed(payload interface{}) (gemaraproj.Result, string, gemaraproj.ConfidenceLevel) {
	return gemaraproj.Failed, "evaluated from a sha-verified formal result that failed", gemaraproj.Medium
}

// gemaraFormalStepNotRun records a stale or unrecognised formal result: it was
// not evaluated.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func gemaraFormalStepNotRun(payload interface{}) (gemaraproj.Result, string, gemaraproj.ConfidenceLevel) {
	return gemaraproj.NotRun, "not evaluated: the formal result is stale or has no pass or fail status", gemaraproj.Undetermined
}

// gemaraFormalStepFor returns the record-only step matching a Gemara result.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func gemaraFormalStepFor(result gemaraproj.Result) gemaraproj.AssessmentStep {
	switch result {
	case gemaraproj.Passed:
		return gemaraFormalStepPassed
	case gemaraproj.Failed:
		return gemaraFormalStepFailed
	}
	return gemaraFormalStepNotRun
}

// formalResultToGemara maps a formal result to a Gemara result: pass and
// failed map directly; stale and not-run are NotRun.
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func formalResultToGemara(result string) gemaraproj.Result {
	switch result {
	case codescan.FormalPass:
		return gemaraproj.Passed
	case codescan.FormalFailed:
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
	results := codescan.ReadFormalEvidence(baseDir, codescan.FormalEvidenceFiles(baseDir, codeRoot))
	if len(results) == 0 {
		return aggregate
	}
	log.Metadata.MappingReferences = append(log.Metadata.MappingReferences,
		gemaraproj.MappingReference{Id: gemaraRefRequirements, Title: "Requirements", Version: Version})
	byReq := map[string][]codescan.FormalEntry{}
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
				Steps:           []gemaraproj.AssessmentStep{gemaraFormalStepFor(res)},
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
func formalMessage(r codescan.FormalEntry) string {
	parts := []string{fmt.Sprintf("machine %s, invariant %s", textutil.Fallback(r.Machine, "(unnamed)"), textutil.Fallback(r.Invariant, "(unnamed)"))}
	if r.Mode != "" {
		parts = append(parts, "mode "+r.Mode)
	}
	if r.QntSha256 != "" {
		parts = append(parts, "result hash "+r.QntSha256)
	}
	msg := strings.Join(parts, ", ")
	switch r.Result {
	case codescan.FormalStale:
		return "Not evaluated, " + r.Notes + " (" + msg + ")."
	case codescan.FormalNotRun:
		return "Not evaluated: the recorded status is not a pass or a fail (" + msg + ")."
	}
	return "Formal result " + r.Result + ": " + msg + "."
}
