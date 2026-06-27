// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

// Gemara L5 Evaluation Log generation. The log records the opinionated result of
// assessing each control, sourced from authored ControlVerifications (which carry
// pass AND fail outcomes) so the log is a faithful Evaluation Log rather than a
// failures-only assessment. Each control verification becomes one AssessmentLog
// under its control's ControlEvaluation.

import (
	"fmt"
	"strings"

	gemara "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/model"
)

// gemaraDefaultTimestamp is a deterministic placeholder used when no date is
// supplied, keeping generated artifacts reproducible.
const gemaraDefaultTimestamp = "1970-01-01T00:00:00Z"

// GemaraEvaluationResult holds the L5 Evaluation Log and its serialization.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type GemaraEvaluationResult struct {
	EvaluationLog gemara.EvaluationLog
	YAML          string
	HasContent    bool // false when there is nothing to evaluate (no controls)
}

// GenerateGemaraEvaluationLogFromFiles loads inputs and renders the Evaluation Log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraEvaluationLogFromFiles(architecturePath, requirementsPath, codeRoot string, options GemaraExportOptions) (GemaraEvaluationResult, error) {
	bundle, err := model.LoadBundle(architecturePath)
	if err != nil {
		return GemaraEvaluationResult{}, err
	}
	var requirements model.RequirementsDocument
	if strings.TrimSpace(requirementsPath) != "" {
		requirements, err = model.LoadRequirements(requirementsPath)
		if err != nil {
			return GemaraEvaluationResult{}, err
		}
	}
	return GenerateGemaraEvaluationLog(bundle, requirements, codeRoot, options)
}

// gemaraInferredStep is a record-only assessment step. Its name is serialized as
// the (non-empty) step string required by the schema; the function is never run.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func gemaraInferredStep(payload interface{}) (gemara.Result, string, gemara.ConfidenceLevel) {
	return gemara.Passed, "evaluated from authored control verification", gemara.Medium
}

// GenerateGemaraEvaluationLog builds an L5 Evaluation Log from the model bundle.
// requirements and codeRoot are reserved for inferred-verification augmentation.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateGemaraEvaluationLog(bundle model.Bundle, requirements model.RequirementsDocument, codeRoot string, options GemaraExportOptions) (GemaraEvaluationResult, error) {
	cfg := newGemaraConfig(bundle, options)
	a := bundle.Architecture.AuthoredArchitecture

	start := gemaraDefaultTimestamp
	if d := normalizeDatetime(options.Date); d != "" {
		start = d
	}

	// Group control verifications by control id.
	verifsByControl := map[string][]model.ControlVerification{}
	for _, cv := range a.ControlVerifications {
		verifsByControl[cv.ControlRef] = append(verifsByControl[cv.ControlRef], cv)
	}

	log := gemara.EvaluationLog{
		Metadata: cfg.newMetadata(cfg.modelID+"-EVALUATION-LOG", "Control evaluation results for "+cfg.modelTitle, gemara.EvaluationLogArtifact),
		Target: gemara.Resource{
			Id:          cfg.modelID,
			Name:        cfg.modelTitle,
			Type:        gemara.Software,
			Description: "Evaluation target: " + cfg.modelTitle,
		},
	}
	// Reference the control catalog the assessment requirements live in.
	log.Metadata.MappingReferences = []gemara.MappingReference{
		{Id: gemaraRefControlCatalog, Title: "Control Catalog", Version: GemaraVersion},
	}

	aggregate := gemara.NotRun
	for _, ctrl := range a.Controls {
		verifs := verifsByControl[ctrl.ID]
		name := fallback(ctrl.Name, ctrl.ID)

		var logs []*gemara.AssessmentLog
		ceResult := gemara.NotRun

		if len(verifs) == 0 {
			// Synthesize a single not-run assessment for the default requirement.
			res := gemara.NeedsReview
			logs = append(logs, &gemara.AssessmentLog{
				Requirement:   gemara.EntryMapping{ReferenceId: gemaraRefControlCatalog, EntryId: ctrl.ID + "-AR-1"},
				Description:   "No authored verification for control " + name + "; manual review required.",
				Result:        res,
				Message:       "No control verification recorded.",
				Applicability: []string{"all-systems"},
				Steps:         []gemara.AssessmentStep{gemaraInferredStep},
				Start:         gemara.Datetime(start),
			})
			ceResult = gemara.UpdateAggregateResult(ceResult, res)
		} else {
			for _, cv := range verifs {
				res := mapVerificationResult(cv.Status)
				msg := strings.TrimSpace(strings.Join(cv.Findings, "; "))
				if msg == "" {
					msg = "Status: " + fallback(cv.Status, "unknown")
				}
				al := &gemara.AssessmentLog{
					Requirement:     gemara.EntryMapping{ReferenceId: gemaraRefControlCatalog, EntryId: cv.ID},
					Description:     fmt.Sprintf("Assessment of %q via %s.", name, fallback(cv.Method, "verification")),
					Result:          res,
					Message:         msg,
					Applicability:   []string{"all-systems"},
					Steps:           []gemara.AssessmentStep{gemaraInferredStep},
					Start:           gemara.Datetime(start),
					ConfidenceLevel: confidenceForResult(res),
				}
				if res == gemara.Failed && len(cv.Findings) > 0 {
					al.Recommendation = "Address findings: " + strings.Join(cv.Findings, "; ")
				}
				for i, ev := range cv.Evidence {
					al.Evidence = append(al.Evidence, gemara.Evidence{
						Id:          fmt.Sprintf("%s-EV-%d", cv.ID, i+1),
						Type:        gemara.EvidenceType("ControlVerification"),
						CollectedAt: gemara.Datetime(start),
						Description: fallback(ev.Description, ev.Path),
					})
				}
				if lt := normalizeDatetime(cv.LastTested); lt != "" {
					al.End = gemara.Datetime(lt)
				}
				logs = append(logs, al)
				ceResult = gemara.UpdateAggregateResult(ceResult, res)
			}
		}

		ce := &gemara.ControlEvaluation{
			Name:           name,
			Result:         ceResult,
			Message:        fmt.Sprintf("%d assessment(s) recorded for control %s.", len(logs), ctrl.ID),
			Control:        gemara.EntryMapping{ReferenceId: gemaraRefControlCatalog, EntryId: ctrl.ID},
			AssessmentLogs: logs,
		}
		log.Evaluations = append(log.Evaluations, ce)
		aggregate = gemara.UpdateAggregateResult(aggregate, ceResult)
	}

	log.Result = aggregate

	if len(log.Evaluations) == 0 {
		return GemaraEvaluationResult{HasContent: false}, nil
	}

	out, err := marshalGemara(log)
	if err != nil {
		return GemaraEvaluationResult{}, fmt.Errorf("marshal gemara evaluation log: %w", err)
	}
	return GemaraEvaluationResult{EvaluationLog: log, YAML: out, HasContent: true}, nil
}

// confidenceForResult maps an assessment result to an evaluator confidence level.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func confidenceForResult(res gemara.Result) gemara.ConfidenceLevel {
	switch res {
	case gemara.Passed, gemara.Failed:
		return gemara.High
	case gemara.NeedsReview:
		return gemara.Medium
	default:
		return gemara.Low
	}
}

// mapVerificationResult maps an engmod verification status to a Gemara Result.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func mapVerificationResult(status string) gemara.Result {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass", "passed", "satisfied", "implemented", "ok", "success":
		return gemara.Passed
	case "fail", "failed", "not-satisfied", "notsatisfied", "error":
		return gemara.Failed
	case "partial", "needs-review", "review", "in-progress", "planned":
		return gemara.NeedsReview
	case "n/a", "na", "not-applicable", "notapplicable":
		return gemara.NotApplicable
	case "", "unknown", "not-run", "notrun":
		return gemara.NotRun
	default:
		return gemara.Unknown
	}
}
