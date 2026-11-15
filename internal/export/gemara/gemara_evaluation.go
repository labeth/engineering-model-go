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
	"github.com/labeth/engineering-model-go/internal/validate"
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

// FileInputs names the files a *FromFiles generator loads and the code root
// that locates the test-results directory beside the code.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type FileInputs struct {
	ArchitecturePath string
	RequirementsPath string
	CodeRoot         string
}

// EvalRun configures one evaluation run: codeRoot locates the test-results
// directory beside the code for formal evidence, options stamp the metadata.
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
type EvalRun struct {
	CodeRoot string
	Options  Options
}

// GenerateEvaluationLogFromFiles loads inputs and renders the Evaluation Log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateEvaluationLogFromFiles(in FileInputs, options Options) (EvaluationResult, error) {
	bundle, err := loadEvaluationBundle(in)
	if err != nil {
		return EvaluationResult{}, err
	}
	return GenerateEvaluationLog(bundle, bundle.Requirements, EvalRun{CodeRoot: in.CodeRoot, Options: options})
}

// loadEvaluationBundle loads the architecture, the optional requirements file
// and the composition-enriched bundle of a *FromFiles generator.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func loadEvaluationBundle(in FileInputs) (model.Bundle, error) {
	bundle, err := codescan.LoadBundle(in.ArchitecturePath)
	if err != nil {
		return model.Bundle{}, err
	}
	if strings.TrimSpace(in.RequirementsPath) != "" {
		requirements, err := model.LoadRequirements(in.RequirementsPath)
		if err != nil {
			return model.Bundle{}, err
		}
		bundle.Requirements = requirements
	}
	return composition.EnrichBundleFromComposition(bundle, "architecture", "assurance", "compliance", "requirements")
}

// gemaraInferredStep is a record-only assessment step. Its name is serialized as
// the (non-empty) step string required by the schema; the function is never run.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func gemaraInferredStep(payload interface{}) (gemaraproj.Result, string, gemaraproj.ConfidenceLevel) {
	return gemaraproj.Passed, "evaluated from authored control verification", gemaraproj.Medium
}

// GenerateEvaluationLog builds an L5 Evaluation Log from the model bundle.
// requirements is reserved for inferred-verification augmentation; run.CodeRoot
// locates the test-results directory beside the code for formal evidence. A
// requirement with a formal (Quint) result adds a requirement-keyed entry
// (gemara_formal.go); a model without formal evidence adds none.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-035, REQ-EMG-036, REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func GenerateEvaluationLog(bundle model.Bundle, requirements model.RequirementsDocument, run EvalRun) (EvaluationResult, error) {
	bundle.Requirements = requirements
	modelRoot := evaluationModelRoot(bundle)
	canonical, err := model.NewCanonicalBundle(bundle)
	if err != nil {
		return EvaluationResult{}, err
	}
	bundle = canonical.Documents()
	options, err := resolveGemaraOptions(bundle, run.Options)
	if err != nil {
		return EvaluationResult{}, err
	}
	cfg := newGemaraConfig(bundle, options)
	a := bundle.Architecture.AuthoredArchitecture

	start := gemaraDefaultTimestamp
	if d := normalizeDatetime(options.Date); d != "" {
		start = d
	}
	verifsByControl := groupVerificationsByControl(a.ControlVerifications)
	log := newEvaluationLog(cfg)
	aggregate := gemaraproj.NotRun
	for _, ctrl := range a.Controls {
		ce := buildControlEvaluation(ctrl, verifsByControl[ctrl.ID], start)
		log.Evaluations = append(log.Evaluations, ce)
		aggregate = gemaraproj.UpdateAggregateResult(aggregate, ce.Result)
	}

	// Requirement-keyed formal (Quint) entries; none without formal evidence.
	aggregate = appendFormalEvaluations(&log, aggregate, formalScope{ModelRoot: modelRoot, CodeRoot: run.CodeRoot, Start: start})
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

// evaluationModelRoot is the directory of the model's documents, empty for a
// bundle that was not loaded from files.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func evaluationModelRoot(bundle model.Bundle) string {
	if bundle.ManifestPath == "" && bundle.ArchitecturePath == "" {
		return ""
	}
	return codescan.ModelRootDir(bundle)
}

// groupVerificationsByControl groups control verifications by control id.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func groupVerificationsByControl(verifications []model.ControlVerification) map[string][]model.ControlVerification {
	byControl := map[string][]model.ControlVerification{}
	for _, cv := range verifications {
		byControl[cv.ControlRef] = append(byControl[cv.ControlRef], cv)
	}
	return byControl
}

// newEvaluationLog is the empty log that references the control catalog the
// assessment requirements live in.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func newEvaluationLog(cfg gemaraConfig) gemaraproj.EvaluationLog {
	log := gemaraproj.EvaluationLog{
		Metadata: cfg.newMetadata(cfg.evaluationLogID(), "Control evaluation results for "+cfg.modelTitle, gemaraproj.EvaluationLogArtifact),
		Target: gemaraproj.Resource{
			Id:          cfg.modelID,
			Name:        cfg.modelTitle,
			Type:        gemaraproj.Software,
			Description: "Evaluation target: " + cfg.modelTitle,
		},
	}
	log.Metadata.MappingReferences = []gemaraproj.MappingReference{
		{Id: gemaraRefControlCatalog, Title: "Control Catalog", Version: Version},
	}
	return log
}

// buildControlEvaluation turns the verifications of one control into its
// ControlEvaluation; a control without verifications gets one needs-review log.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func buildControlEvaluation(ctrl model.Control, verifs []model.ControlVerification, start string) *gemaraproj.ControlEvaluation {
	name := textutil.Fallback(ctrl.Name, ctrl.ID)
	var logs []*gemaraproj.AssessmentLog
	if len(verifs) == 0 {
		logs = append(logs, unverifiedAssessment(ctrl.ID, name, start))
	}
	for _, cv := range verifs {
		logs = append(logs, verificationAssessment(cv, name, start))
	}
	ceResult := gemaraproj.NotRun
	for _, al := range logs {
		ceResult = gemaraproj.UpdateAggregateResult(ceResult, al.Result)
	}
	return &gemaraproj.ControlEvaluation{
		Name:           name,
		Result:         ceResult,
		Message:        fmt.Sprintf("%d assessment(s) recorded for control %s.", len(logs), ctrl.ID),
		Control:        gemaraproj.EntryMapping{ReferenceId: gemaraRefControlCatalog, EntryId: ctrl.ID},
		AssessmentLogs: logs,
	}
}

// unverifiedAssessment synthesizes the single needs-review assessment of a
// control without an authored verification.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func unverifiedAssessment(controlID, name, start string) *gemaraproj.AssessmentLog {
	return &gemaraproj.AssessmentLog{
		Requirement:   gemaraproj.EntryMapping{ReferenceId: gemaraRefControlCatalog, EntryId: controlID + "-AR-1"},
		Description:   "No authored verification for control " + name + "; manual review required.",
		Result:        gemaraproj.NeedsReview,
		Message:       "No control verification recorded.",
		Applicability: []string{"all-systems"},
		Steps:         []gemaraproj.AssessmentStep{gemaraInferredStep},
		Start:         gemaraproj.Datetime(start),
	}
}

// verificationAssessment turns one authored control verification into an
// assessment log with its evidence.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func verificationAssessment(cv model.ControlVerification, name, start string) *gemaraproj.AssessmentLog {
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
		al.Evidence = append(al.Evidence, controlVerificationEvidence(fmt.Sprintf("%s-EV-%d", cv.ID, i+1), start, ev))
	}
	if lt := normalizeDatetime(cv.LastTested); lt != "" {
		al.End = gemaraproj.Datetime(lt)
	}
	return al
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

// mapVerificationResult maps an engmod verification status to a Gemara Result
// through validate.ClassifyVerificationStatus, the classification
// gemara.control_needs_review uses.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-083
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func mapVerificationResult(status string) gemaraproj.Result {
	switch validate.ClassifyVerificationStatus(status) {
	case validate.OutcomePassed:
		return gemaraproj.Passed
	case validate.OutcomeFailed:
		return gemaraproj.Failed
	case validate.OutcomeNeedsReview:
		return gemaraproj.NeedsReview
	case validate.OutcomeNotApplicable:
		return gemaraproj.NotApplicable
	case validate.OutcomeNotRun:
		return gemaraproj.NotRun
	default:
		return gemaraproj.Unknown
	}
}
