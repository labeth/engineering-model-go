// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package gemara

// Gemara L5 Evaluation Log generation. The log records the opinionated result of
// assessing each control, sourced from authored ControlVerifications (which carry
// pass AND fail outcomes) so the log is a faithful Evaluation Log rather than a
// failures-only assessment. Each control verification becomes one AssessmentLog
// under its control's ControlEvaluation.

import (
	"fmt"
	"strings"

	gemaraproj "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/textutil"
)

// gemaraDefaultTimestamp is used only for empty documents that contain no
// timeline-bearing authored content.
const gemaraDefaultTimestamp = "1970-01-01T00:00:00Z"

// EvaluationResult holds the L5 Evaluation Log and its serialization.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type EvaluationResult struct {
	EvaluationLog gemaraproj.EvaluationLog
	YAML          string
	HasContent    bool // false when there is nothing to evaluate (no controls)
}

// GenerateEvaluationLogFromFiles loads inputs and renders the Evaluation Log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateEvaluationLogFromFiles(architecturePath, requirementsPath, codeRoot string, options Options) (EvaluationResult, error) {
	bundle, err := codescan.LoadBundle(architecturePath)
	if err != nil {
		return EvaluationResult{}, err
	}
	var requirements model.RequirementsDocument
	if strings.TrimSpace(requirementsPath) != "" {
		requirements, err = model.LoadRequirements(requirementsPath)
		if err != nil {
			return EvaluationResult{}, err
		}
		bundle.Requirements = requirements
	}
	bundle, err = composition.EnrichBundleFromComposition(bundle, "architecture", "assurance", "compliance", "requirements")
	if err != nil {
		return EvaluationResult{}, err
	}
	requirements = bundle.Requirements
	return GenerateEvaluationLog(bundle, requirements, codeRoot, options)
}

// gemaraInferredStep is a record-only assessment step. Its name is serialized as
// the (non-empty) step string required by the schema; the function is never run.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func gemaraInferredStep(payload interface{}) (gemaraproj.Result, string, gemaraproj.ConfidenceLevel) {
	return gemaraproj.Passed, "evaluated from authored control verification", gemaraproj.Medium
}

// GenerateEvaluationLog builds an L5 Evaluation Log from the model bundle.
// requirements and codeRoot are reserved for inferred-verification augmentation.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateEvaluationLog(bundle model.Bundle, requirements model.RequirementsDocument, codeRoot string, options Options) (EvaluationResult, error) {
	bundle.Requirements = requirements
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return EvaluationResult{}, err
	}
	bundle = canonical.Documents()
	requirements = bundle.Requirements
	options, err = resolveGemaraOptions(bundle, options)
	if err != nil {
		return EvaluationResult{}, err
	}
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

	log := gemaraproj.EvaluationLog{
		Metadata: cfg.newMetadata(cfg.evaluationLogID(), "Control evaluation results for "+cfg.modelTitle, gemaraproj.EvaluationLogArtifact),
		Target: gemaraproj.Resource{
			Id:          cfg.modelID,
			Name:        cfg.modelTitle,
			Type:        gemaraproj.Software,
			Description: "Evaluation target: " + cfg.modelTitle,
		},
	}
	// Reference the control catalog the assessment requirements live in.
	log.Metadata.MappingReferences = []gemaraproj.MappingReference{
		{Id: gemaraRefControlCatalog, Title: "Control Catalog", Version: Version},
	}

	aggregate := gemaraproj.NotRun
	for _, ctrl := range a.Controls {
		verifs := verifsByControl[ctrl.ID]
		name := textutil.Fallback(ctrl.Name, ctrl.ID)

		var logs []*gemaraproj.AssessmentLog
		ceResult := gemaraproj.NotRun

		if len(verifs) == 0 {
			// Synthesize a single not-run assessment for the default requirement.
			res := gemaraproj.NeedsReview
			logs = append(logs, &gemaraproj.AssessmentLog{
				Requirement:   gemaraproj.EntryMapping{ReferenceId: gemaraRefControlCatalog, EntryId: ctrl.ID + "-AR-1"},
				Description:   "No authored verification for control " + name + "; manual review required.",
				Result:        res,
				Message:       "No control verification recorded.",
				Applicability: []string{"all-systems"},
				Steps:         []gemaraproj.AssessmentStep{gemaraInferredStep},
				Start:         gemaraproj.Datetime(start),
			})
			ceResult = gemaraproj.UpdateAggregateResult(ceResult, res)
		} else {
			for _, cv := range verifs {
				res := mapVerificationResult(cv.Status)
				msg := strings.TrimSpace(strings.Join(cv.Findings, "; "))
				if msg == "" {
					msg = "Status: " + textutil.Fallback(cv.Status, "unknown")
				}
				al := &gemaraproj.AssessmentLog{
					Requirement:     gemaraproj.EntryMapping{ReferenceId: gemaraRefControlCatalog, EntryId: cv.ID},
					Description:     fmt.Sprintf("Assessment of %q via %s.", name, textutil.Fallback(cv.Method, "verification")),
					Result:          res,
					Message:         msg,
					Applicability:   []string{"all-systems"},
					Steps:           []gemaraproj.AssessmentStep{gemaraInferredStep},
					Start:           gemaraproj.Datetime(start),
					ConfidenceLevel: confidenceForResult(res),
				}
				if res == gemaraproj.Failed && len(cv.Findings) > 0 {
					al.Recommendation = "Address findings: " + strings.Join(cv.Findings, "; ")
				}
				for i, ev := range cv.Evidence {
					al.Evidence = append(al.Evidence, controlVerificationEvidence(
						fmt.Sprintf("%s-EV-%d", cv.ID, i+1),
						start,
						ev,
					))
				}
				if lt := normalizeDatetime(cv.LastTested); lt != "" {
					al.End = gemaraproj.Datetime(lt)
				}
				logs = append(logs, al)
				ceResult = gemaraproj.UpdateAggregateResult(ceResult, res)
			}
		}

		ce := &gemaraproj.ControlEvaluation{
			Name:           name,
			Result:         ceResult,
			Message:        fmt.Sprintf("%d assessment(s) recorded for control %s.", len(logs), ctrl.ID),
			Control:        gemaraproj.EntryMapping{ReferenceId: gemaraRefControlCatalog, EntryId: ctrl.ID},
			AssessmentLogs: logs,
		}
		log.Evaluations = append(log.Evaluations, ce)
		aggregate = gemaraproj.UpdateAggregateResult(aggregate, ceResult)
	}

	log.Result = aggregate

	if len(log.Evaluations) == 0 {
		return EvaluationResult{HasContent: false}, nil
	}

	out, err := marshalGemara(log)
	if err != nil {
		return EvaluationResult{}, fmt.Errorf("marshal gemara evaluation log: %w", err)
	}
	return EvaluationResult{EvaluationLog: log, YAML: out, HasContent: true}, nil
}

// confidenceForResult maps an assessment result to an evaluator confidence level.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func confidenceForResult(res gemaraproj.Result) gemaraproj.ConfidenceLevel {
	switch res {
	case gemaraproj.Passed, gemaraproj.Failed:
		return gemaraproj.High
	case gemaraproj.NeedsReview:
		return gemaraproj.Medium
	default:
		return gemaraproj.Low
	}
}

// mapVerificationResult maps an engmod verification status to a Gemara Result.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func mapVerificationResult(status string) gemaraproj.Result {
	switch strings.ToLower(strings.TrimSpace(status)) {
	case "pass", "passed", "satisfied", "implemented", "ok", "success":
		return gemaraproj.Passed
	case "fail", "failed", "not-satisfied", "notsatisfied", "error":
		return gemaraproj.Failed
	case "partial", "needs-review", "review", "in-progress", "planned":
		return gemaraproj.NeedsReview
	case "n/a", "na", "not-applicable", "notapplicable":
		return gemaraproj.NotApplicable
	case "", "unknown", "not-run", "notrun":
		return gemaraproj.NotRun
	default:
		return gemaraproj.Unknown
	}
}
