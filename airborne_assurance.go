// ENGMODEL-OWNER-UNIT: FU-AIRBORNE-ASSURANCE-EXPORTER
package engmodel

import (
	"bytes"
	"encoding/csv"
	"encoding/json"
	"errors"
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/model"
)

const AirborneDisclaimer = "DRAFT / NOT A COMPLIANCE OR CERTIFICATION DETERMINATION"

var ErrNoAviationProfile = errors.New("no aviation assurance profile is authored")

type AirborneAssuranceOptions struct {
	Date    string
	Version string
}

type AirborneAssuranceResult struct {
	Files       map[string][]byte
	Diagnostics []model.AviationDiagnostic
}

type airborneEnvelope struct {
	Disclaimer string   `json:"disclaimer"`
	Model      string   `json:"model"`
	Version    string   `json:"version,omitempty"`
	Date       string   `json:"date,omitempty"`
	Baselines  []string `json:"baselines"`
	OpenGaps   int      `json:"openGaps"`
	Data       any      `json:"data"`
}

type gapRecord struct {
	ID       string `json:"id"`
	Area     string `json:"area"`
	State    string `json:"state"`
	Summary  string `json:"summary"`
	Blocking bool   `json:"blocking"`
}

// GenerateAirborneAssuranceFromFile validates and renders the complete
// evidence-readiness package. It does not determine compliance or certification.
// TRLC-LINKS: REQ-EMG-053, REQ-EMG-054, REQ-EMG-055
func GenerateAirborneAssuranceFromFile(manifestPath string, options AirborneAssuranceOptions) (AirborneAssuranceResult, error) {
	bundle, err := model.LoadBundle(manifestPath)
	if err != nil {
		if validation, ok := err.(*model.AviationValidationError); ok {
			return AirborneAssuranceResult{Diagnostics: validation.Diagnostics}, err
		}
		return AirborneAssuranceResult{}, err
	}
	if bundle.Aviation == nil {
		return AirborneAssuranceResult{}, ErrNoAviationProfile
	}
	aviation := bundle.Aviation.Aviation
	aviation.Profile.ClaimsDisclaimer = model.NormalizeDisclaimer(aviation.Profile.ClaimsDisclaimer)
	gaps := airborneGaps(aviation)
	baselines := make([]string, 0, len(aviation.Baselines))
	for _, baseline := range aviation.Baselines {
		baselines = append(baselines, baseline.ID+"@"+baseline.Version+"#"+baseline.Hash)
	}
	sort.Strings(baselines)
	wrap := func(data any) []byte {
		value, marshalErr := json.MarshalIndent(airborneEnvelope{
			Disclaimer: AirborneDisclaimer, Model: bundle.Manifest.Module.ModelID,
			Version: options.Version, Date: options.Date, Baselines: baselines,
			OpenGaps: len(gaps), Data: data,
		}, "", "  ")
		if marshalErr != nil {
			panic(marshalErr)
		}
		return append(value, '\n')
	}

	lifecycle := completeLifecycleData(aviation.LifecycleData)
	files := map[string][]byte{
		"READINESS-SUMMARY.md":                     []byte(readinessSummary(bundle.Manifest.Module.Title, aviation, baselines, gaps, options)),
		"PSAC-OUTLINE-DRAFT.md":                    []byte(psacOutline(aviation, baselines, gaps, options)),
		"LIFECYCLE-DATA-INDEX.json":                wrap(lifecycle),
		"OBJECTIVE-EVIDENCE-MATRIX.csv":            objectiveCSV(aviation.Objectives),
		"REQUIREMENTS-TRACE-MATRIX.csv":            requirementsCSV(aviation),
		"VERIFICATION-SUMMARY.json":                wrap(map[string]any{"cases": aviation.VerificationCases, "procedures": aviation.Procedures, "executions": aviation.Executions}),
		"INDEPENDENCE-REPORT.json":                 wrap(map[string]any{"records": aviation.Independence, "reviews": aviation.Reviews, "approvals": aviation.Approvals}),
		"STRUCTURAL-COVERAGE-SUMMARY.json":         wrap(aviation.Coverage),
		"TOOL-ASSESSMENT-REGISTER.json":            wrap(aviation.Tools),
		"SOFTWARE-CONFIGURATION-INDEX.json":        wrap(map[string]any{"baselines": aviation.Baselines, "builds": aviation.Builds}),
		"PROBLEM-REPORTS.json":                     wrap(aviation.ProblemReports),
		"SQA-RECORDS.json":                         wrap(aviation.SQAAudits),
		"CERTIFICATION-LIAISON-LOG.json":           wrap(aviation.Liaison),
		"OPEN-GAPS.json":                           wrap(gaps),
		"SOFTWARE-ACCOMPLISHMENT-SUMMARY-DRAFT.md": []byte(sasDraft(aviation, baselines, gaps, options)),
	}
	return AirborneAssuranceResult{Files: files}, nil
}

// TRLC-LINKS: REQ-EMG-055
func completeLifecycleData(authored []model.LifecycleDataRecord) []model.LifecycleDataRecord {
	byCode := map[string]model.LifecycleDataRecord{}
	for _, record := range authored {
		byCode[record.CategoryCode] = record
	}
	out := model.RecognizedLifecycleDataFamilies()
	for i := range out {
		if record, ok := byCode[out[i].CategoryCode]; ok {
			out[i] = record
		} else {
			out[i].ID = "MISSING-" + out[i].CategoryCode
			out[i].State = "declared"
			out[i].Owner = "UNASSIGNED"
		}
	}
	return out
}

// TRLC-LINKS: REQ-EMG-055
func airborneGaps(a model.AviationProfile) []gapRecord {
	var gaps []gapRecord
	for _, item := range completeLifecycleData(a.LifecycleData) {
		if item.Source.Path == "" && item.Source.URI == "" {
			gaps = append(gaps, gapRecord{ID: item.ID, Area: "lifecycle-data", State: item.State, Summary: "Missing source for " + item.CategoryName, Blocking: true})
		} else if model.IsOpenEvidenceState(item.State) {
			gaps = append(gaps, gapRecord{ID: item.ID, Area: "lifecycle-data", State: item.State, Summary: item.CategoryName + " is not accepted", Blocking: true})
		}
	}
	for _, objective := range a.Objectives {
		if objective.Applicability == "pending" || model.IsOpenEvidenceState(objective.State) {
			gaps = append(gaps, gapRecord{ID: objective.ID, Area: "objective", State: objective.State, Summary: "Objective applicability or evidence remains open", Blocking: true})
		}
	}
	for _, coverage := range a.Coverage {
		if coverage.Uncovered > 0 || model.IsOpenEvidenceState(coverage.Status) {
			added := false
			for _, disposition := range coverage.Dispositions {
				if disposition.Disposition == "open" || model.IsOpenEvidenceState(disposition.Status) {
					gaps = append(gaps, gapRecord{ID: disposition.ID, Area: "structural-coverage", State: disposition.Status, Summary: disposition.Rationale, Blocking: true})
					added = true
				}
			}
			if !added {
				gaps = append(gaps, gapRecord{ID: coverage.ID, Area: "structural-coverage", State: coverage.Status, Summary: fmt.Sprintf("%d statements remain dispositioned but uncovered", coverage.Uncovered), Blocking: true})
			}
		}
	}
	for _, report := range a.ProblemReports {
		if model.IsOpenEvidenceState(report.State) {
			gaps = append(gaps, gapRecord{ID: report.ID, Area: "problem-report", State: report.State, Summary: report.Summary, Blocking: true})
		}
	}
	for _, audit := range a.SQAAudits {
		if len(audit.Findings) > 0 && model.IsOpenEvidenceState(audit.State) {
			gaps = append(gaps, gapRecord{ID: audit.ID, Area: "sqa", State: audit.State, Summary: strings.Join(audit.Findings, "; "), Blocking: true})
		}
	}
	sort.Slice(gaps, func(i, j int) bool {
		if gaps[i].Area == gaps[j].Area {
			return gaps[i].ID < gaps[j].ID
		}
		return gaps[i].Area < gaps[j].Area
	})
	return gaps
}

// TRLC-LINKS: REQ-EMG-055
func objectiveCSV(objectives []model.ObjectiveReference) []byte {
	rows := [][]string{{"disclaimer", "objective_id", "source_reference", "applicability", "independence_required", "state", "rationale", "evidence_refs"}}
	for _, objective := range objectives {
		rows = append(rows, []string{AirborneDisclaimer, objective.ID, objective.SourceReference, objective.Applicability, fmt.Sprint(objective.IndependenceRequired), objective.State, objective.Rationale, strings.Join(objective.EvidenceRefs, ";")})
	}
	return renderCSV(rows)
}

// TRLC-LINKS: REQ-EMG-055
func requirementsCSV(a model.AviationProfile) []byte {
	executions := map[string][]string{}
	results := map[string][]string{}
	for _, execution := range a.Executions {
		for _, caseRef := range execution.CaseRefs {
			executions[caseRef] = append(executions[caseRef], execution.ID)
			results[caseRef] = append(results[caseRef], execution.Result)
		}
	}
	rows := [][]string{{"disclaimer", "requirement_id", "level", "derived", "parent_refs", "architecture_refs", "code_refs", "verification_cases", "procedures", "executions", "results", "baseline", "state"}}
	for _, overlay := range a.RequirementAssurance {
		var caseRefs, procedureRefs, executionRefs, resultRefs []string
		for _, testCase := range a.VerificationCases {
			if intersects(testCase.RequirementRefs, []string{overlay.RequirementID}) {
				caseRefs = append(caseRefs, testCase.ID)
				procedureRefs = append(procedureRefs, testCase.ProcedureRef)
				executionRefs = append(executionRefs, executions[testCase.ID]...)
				resultRefs = append(resultRefs, results[testCase.ID]...)
			}
		}
		rows = append(rows, []string{AirborneDisclaimer, overlay.RequirementID, overlay.Level, fmt.Sprint(overlay.Derived), strings.Join(overlay.ParentRefs, ";"), strings.Join(overlay.ArchitectureRefs, ";"), strings.Join(overlay.CodeRefs, ";"), strings.Join(caseRefs, ";"), strings.Join(procedureRefs, ";"), strings.Join(executionRefs, ";"), strings.Join(resultRefs, ";"), overlay.Baseline, overlay.State})
	}
	return renderCSV(rows)
}

// TRLC-LINKS: REQ-EMG-055
func renderCSV(rows [][]string) []byte {
	var buffer bytes.Buffer
	writer := csv.NewWriter(&buffer)
	_ = writer.WriteAll(rows)
	writer.Flush()
	return buffer.Bytes()
}

// TRLC-LINKS: REQ-EMG-055
func intersects(left, right []string) bool {
	for _, a := range left {
		for _, b := range right {
			if a == b {
				return true
			}
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-055
func readinessSummary(title string, a model.AviationProfile, baselines []string, gaps []gapRecord, options AirborneAssuranceOptions) string {
	counts := map[string]int{}
	for _, item := range a.LifecycleData {
		counts[item.State]++
	}
	for _, item := range a.Objectives {
		counts[item.State]++
	}
	for _, item := range a.Executions {
		counts[item.Status]++
	}
	states := []string{"declared", "planned", "executed", "reviewed", "accepted", "not-applicable", "authority-accepted"}
	var b strings.Builder
	fmt.Fprintf(&b, "# %s — Airborne Software Evidence Readiness\n\n**%s**\n\n", title, AirborneDisclaimer)
	fmt.Fprintf(&b, "%s\n\nThis package is a readiness aid. It does not reproduce licensed objective text and cannot replace applicant, independent reviewer, configuration-management, quality-assurance, safety, or certification-authority decisions.\n\n", a.Profile.ClaimsDisclaimer)
	fmt.Fprintf(&b, "- Standard profile: %s %s\n- Software level: %s\n- Software item: %s\n- Applicant / authority: %s / %s\n- Generated version/date: %s / %s\n- Baselines: %s\n- Open gaps: %d\n\n## Evidence states\n\n", a.Profile.Standard, a.Profile.Edition, a.Profile.SoftwareLevel, a.Profile.SoftwareItem, a.Profile.Applicant, a.Profile.Authority, options.Version, options.Date, strings.Join(baselines, ", "), len(gaps))
	for _, state := range states {
		fmt.Fprintf(&b, "- %s: %d\n", state, counts[state])
	}
	b.WriteString("\n## Open gaps\n\n")
	for _, gap := range gaps {
		fmt.Fprintf(&b, "- **%s** (%s/%s): %s\n", gap.ID, gap.Area, gap.State, gap.Summary)
	}
	return b.String()
}

// TRLC-LINKS: REQ-EMG-055
func psacOutline(a model.AviationProfile, baselines []string, gaps []gapRecord, options AirborneAssuranceOptions) string {
	return fmt.Sprintf("# PSAC Outline (Draft)\n\n**%s**\n\nGenerated: %s; package version: %s\n\n## Software item and certification basis\n\n- Item: %s\n- Standard/profile: %s %s, Level %s\n- Certification basis reference: %s\n- Safety allocation reference: %s\n- Supplements: %s\n\n## Planned lifecycle and evidence\n\nThe lifecycle-data index identifies the 22 recognized data families and explicitly marks missing sources. Objective applicability is represented only through applicant-supplied opaque identifiers and references.\n\n## Configuration and verification basis\n\nBaselines: %s\n\n## Coordination and open items\n\n%d open readiness gaps remain. Human planning approval, process execution, independence, CM/SQA, safety feedback, and authority acceptance remain external responsibilities.\n", AirborneDisclaimer, options.Date, options.Version, a.Profile.SoftwareItem, a.Profile.Standard, a.Profile.Edition, a.Profile.SoftwareLevel, a.Profile.CertificationBasis, a.Profile.SafetyAllocationReference, strings.Join(a.Profile.Supplements, ", "), strings.Join(baselines, ", "), len(gaps))
}

// TRLC-LINKS: REQ-EMG-055
func sasDraft(a model.AviationProfile, baselines []string, gaps []gapRecord, options AirborneAssuranceOptions) string {
	return fmt.Sprintf("# Software Accomplishment Summary (Draft)\n\n**%s**\n\nGenerated: %s; package version: %s\n\n## Identification\n\n- Software item: %s\n- Profile: %s %s, Level %s\n- Baselines/build basis: %s\n\n## Readiness summary\n\n- Requirement assurance records: %d\n- Verification executions: %d\n- Structural coverage summaries: %d\n- Problem reports: %d\n- SQA audits: %d\n- Open gaps: %d\n\nThis draft summarizes authored evidence only. It is not a statement that lifecycle objectives were satisfied, that independence was adequate, that tools were qualified, or that an authority accepted the software.\n", AirborneDisclaimer, options.Date, options.Version, a.Profile.SoftwareItem, a.Profile.Standard, a.Profile.Edition, a.Profile.SoftwareLevel, strings.Join(baselines, ", "), len(a.RequirementAssurance), len(a.Executions), len(a.Coverage), len(a.ProblemReports), len(a.SQAAudits), len(gaps))
}
