// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"encoding/json"
	"sort"
	"strconv"

	"github.com/labeth/engineering-model-go/internal/model"
)

// The report of the code health analysis, schema "engmod-health/1"
// (ADR-EMG-050). It holds no timestamp and no absolute path, and every list is
// sorted, so two runs on one tree give byte-identical reports. It is the only
// thing the roll-ups, the exports and the delta check read; no engine internals
// leak into it. docs/health-report.schema.json describes it.

// EngineInfo names the engine that produced the report and the scoring it
// applied.
type EngineInfo struct {
	Name           string `json:"name"`
	Version        string `json:"version"`
	ScoringVersion int    `json:"scoringVersion"`
	// Weights is the weights table of the scoring version, so a reader can
	// recompute the deductions of a report.
	Weights map[string]float64 `json:"weights,omitempty"`
}

// FileReport is one scored source file.
type FileReport struct {
	Path        string      `json:"path"`
	Language    string      `json:"language"`
	LOC         int         `json:"loc"`
	IsTest      bool        `json:"isTest,omitempty"`
	Score       float64     `json:"score"`
	Band        string      `json:"band"`
	Deductions  []Deduction `json:"deductions,omitempty"`
	Functions   []Function  `json:"functions"`
	Unsupported []string    `json:"unsupported,omitempty"`
	// SyntaxErrors counts the syntax errors the parser found (see FileResult).
	SyntaxErrors int `json:"syntaxErrors,omitempty"`
}

// Diagnostic is a finding of the analysis about the analysis itself: an
// unused waiver, a file that could not be analysed. Level is "error" or
// "warning".
type Diagnostic struct {
	Code     string `json:"code"`
	Level    string `json:"level"`
	Message  string `json:"message"`
	Path     string `json:"path,omitempty"`
	Function string `json:"function,omitempty"`
}

// Summary counts the findings of a report by their level at the project tier.
// Errors and Warnings include the diagnostics.
type Summary struct {
	Files    int `json:"files"`
	Findings int `json:"findings"`
	Waived   int `json:"waived"`
	// Baselined counts the findings the baseline accepts; they are in Findings
	// and in neither Errors nor Warnings. Absent when there are none.
	Baselined int `json:"baselined,omitempty"`
	Errors    int `json:"errors"`
	Warnings  int `json:"warnings"`
	// BelowFloor counts the files whose score is below the floor of the
	// report and FloorCovered those of them that the baseline covers
	// (ADR-EMG-087); the others are health.file_below_floor diagnostics. Both
	// are absent without a floor, and FloorCovered is the number of baselined
	// files that can still be burned down to the floor.
	BelowFloor   int `json:"belowFloor,omitempty"`
	FloorCovered int `json:"floorCovered,omitempty"`
}

// Report is the code health report.
type Report struct {
	Schema string     `json:"schema"`
	Engine EngineInfo `json:"engine"`
	// Tier is the project life-cycle tier the report was made for; it is
	// absent for a project without one. Bundles are its assurance bundles.
	Tier    string   `json:"tier,omitempty"`
	Bundles []string `json:"bundles,omitempty"`
	// Floor is the lowest file score that needs no baseline cover
	// (ADR-EMG-087); absent when the report was made without one.
	Floor float64 `json:"floor,omitempty"`
	// Levels is the level ("error" or "warning") of each code at Tier, from
	// the tier policy of the model. A code missing from it is a warning.
	Levels map[string]string `json:"levels,omitempty"`
	Files  []FileReport      `json:"files"`
	Waived []WaivedFinding   `json:"waived"`
	// Baselined marks the findings the baseline of the manifest accepts
	// (ADR-EMG-059). They also stay in Files, so the score counts them. Absent
	// when the manifest has no baseline or nothing matches.
	Baselined   []BaselinedFinding `json:"baselined,omitempty"`
	Diagnostics []Diagnostic       `json:"diagnostics"`
	Rollups     Rollups            `json:"rollups"`
	Summary     Summary            `json:"summary"`
}

// Options are the inputs of BuildReport besides the engine results.
type Options struct {
	EngineName    string
	EngineVersion string
	Tier          string
	Bundles       []string
	// Levels is the level of each code at the tier (see Levels).
	Levels map[string]string
	// Floor is the score every file needs unless the baseline covers it
	// (ADR-EMG-087); 0 asks for no floor, which a caller that builds a report
	// by hand may want and the scan never does (see model.EffectiveHealthFloor).
	Floor float64
	// Waivers are applied to the results before scoring.
	Waivers []model.HealthWaiver
	// Baseline are the entries of the baseline file, already limited to the
	// files in scope of a path-restricted scan; see ApplyBaseline.
	Baseline []model.HealthBaselineEntry
	// Owners maps a file path to its functional unit; Links are the
	// marker-linked symbols (see BuildRollups).
	Owners map[string]string
	Links  []SymbolLink
	// Diagnostics are added to those of the analysis (for example files that
	// were skipped).
	Diagnostics []Diagnostic
	// Scorer defaults to ScoreFile.
	Scorer Scorer
}

// LevelOf returns the level of a code in the report: "error" or "warning".
//
// TRLC-LINKS: REQ-EMG-155
func (r Report) LevelOf(code string) string {
	if r.Levels[code] == LevelError {
		return LevelError
	}
	return LevelWarning
}

// BuildReport applies the waivers to the engine results, scores every file,
// marks the baselined findings, rolls the scores up and counts the findings by
// their level. The result is sorted by path and holds no timestamp.
//
// TRLC-LINKS: REQ-EMG-151, REQ-EMG-152, REQ-EMG-154, REQ-EMG-155, REQ-EMG-157, REQ-EMG-159, REQ-EMG-209
// ENGMODEL-LINKS: FU-CODE-HEALTH
func BuildReport(results []FileResult, opts Options) Report {
	kept, waived, unused := ApplyWaivers(results, opts.Waivers)
	sort.SliceStable(kept, func(i, j int) bool { return kept[i].Path < kept[j].Path })

	r := newReport(opts, waived, fileReports(kept, opts.Scorer))
	applied := applyBaseline(kept, opts.Baseline)
	r.Baselined = applied.baselined
	annotateGrown(r.Files, applied.grown)
	r.Diagnostics = append(append(r.Diagnostics, opts.Diagnostics...), floorDiagnostics(r.Files, floorPolicy{opts.Floor, BaselineCovered(kept, opts.Baseline), r.LevelOf(CodeFileBelowFloor)})...)
	r.Diagnostics = append(r.Diagnostics, r.syntaxDiagnostics(kept)...)
	r.Diagnostics = append(r.Diagnostics, r.staleDiagnostics(applied.stale)...)
	r.Diagnostics = append(r.Diagnostics, r.unusedWaiverDiagnostics(unused)...)
	sort.SliceStable(r.Diagnostics, func(i, j int) bool { return lessDiagnostic(r.Diagnostics[i], r.Diagnostics[j]) })
	r.Rollups = BuildRollups(r.Files, opts.Owners, opts.Links)
	r.Summary = r.Summarize()
	return r
}

// newReport is the report of the options with its files and waived findings
// and no diagnostics yet.
//
// TRLC-LINKS: REQ-EMG-151, REQ-EMG-155, REQ-EMG-159
func newReport(opts Options, waived []WaivedFinding, files []FileReport) Report {
	r := Report{
		Schema: ReportSchema,
		Engine: EngineInfo{Name: opts.EngineName, Version: opts.EngineVersion, ScoringVersion: ScoringVersion, Weights: Weights()},
		Tier:   opts.Tier, Bundles: opts.Bundles, Floor: opts.Floor, Levels: opts.Levels,
		Files: files, Waived: waived, Diagnostics: []Diagnostic{},
	}
	if r.Waived == nil {
		r.Waived = []WaivedFinding{}
	}
	return r
}

// fileReports scores every file; the default scorer is ScoreFile.
//
// TRLC-LINKS: REQ-EMG-151
func fileReports(kept []FileResult, scorer Scorer) []FileReport {
	if scorer == nil {
		scorer = ScoreFile
	}
	files := make([]FileReport, 0, len(kept))
	for _, res := range kept {
		score := scorer(res)
		fns := res.Functions
		if fns == nil {
			fns = []Function{}
		}
		files = append(files, FileReport{
			Path: res.Path, Language: res.Language, LOC: res.LOC, IsTest: res.IsTest,
			Score: score.Score, Band: score.Band, Deductions: score.Deductions,
			Functions: fns, Unsupported: res.Unsupported, SyntaxErrors: res.SyntaxErrors,
		})
	}
	return files
}

// syntaxDiagnostics lists the diagnostic of each file the parser only partly
// read.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func (r Report) syntaxDiagnostics(kept []FileResult) []Diagnostic {
	var out []Diagnostic
	for _, res := range kept {
		if res.SyntaxErrors > 0 {
			out = append(out, syntaxDiagnostic(res, r.LevelOf(CodeSyntaxError)))
		}
	}
	return out
}

// staleDiagnostics lists the baseline entries that match no finding.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-159
func (r Report) staleDiagnostics(stale []model.HealthBaselineEntry) []Diagnostic {
	var out []Diagnostic
	for _, e := range stale {
		out = append(out, Diagnostic{
			Code: model.HealthBaselineStaleCode, Level: r.LevelOf(model.HealthBaselineStaleCode),
			Message: "baseline entry for " + e.Code + " in " + e.Function + " of " + e.Path + " matches no finding and can be removed",
			Path:    e.Path, Function: e.Function,
		})
	}
	return out
}

// unusedWaiverDiagnostics lists the waivers that match no finding.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-159
func (r Report) unusedWaiverDiagnostics(unused []model.HealthWaiver) []Diagnostic {
	var out []Diagnostic
	for _, w := range unused {
		out = append(out, Diagnostic{
			Code: model.HealthWaiverUnusedCode, Level: r.LevelOf(model.HealthWaiverUnusedCode),
			Message: "waiver for " + w.Code + " in " + w.Function + " of " + w.Path + " (" + w.Decision + ") matches no finding and can be removed",
			Path:    w.Path, Function: w.Function,
		})
	}
	return out
}

// syntaxDiagnostic is the diagnostic of a file the parser only partly read: the
// functions it lost are not measured, so the file is not clean.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func syntaxDiagnostic(res FileResult, level string) Diagnostic {
	count := strconv.Itoa(res.SyntaxErrors)
	if res.SyntaxErrors >= SyntaxErrorCap {
		count = "at least " + count
	}
	return Diagnostic{
		Code: CodeSyntaxError, Level: level, Path: res.Path,
		Message: res.Path + " has " + count + " syntax errors: the functions the parser lost are not measured, so the file may be worse than it scores",
	}
}

// Summarize counts the findings and diagnostics of the report by their level
// at the project tier. A baselined finding is counted in Findings and
// Baselined and in neither Errors nor Warnings, so Findings is Errors plus
// Warnings plus Baselined for the findings (the diagnostics add to the two
// level counts only).
//
// TRLC-LINKS: REQ-EMG-155
func (r Report) Summarize() Summary {
	z := summarizer{r: r, baselined: r.baselinedSet(), s: Summary{Files: len(r.Files), Waived: len(r.Waived)}}
	for _, f := range r.Files {
		z.addFile(f)
	}
	for _, d := range r.Diagnostics {
		if d.Level == LevelError {
			z.s.Errors++
		} else {
			z.s.Warnings++
		}
	}
	z.s.BelowFloor, z.s.FloorCovered = r.floorCounts()
	return z.s
}

// summarizer counts the findings of a report.
type summarizer struct {
	r         Report
	baselined map[baselineKey]bool
	s         Summary
}

// addFile counts the findings of the functions of one file.
//
// TRLC-LINKS: REQ-EMG-155
func (z *summarizer) addFile(f FileReport) {
	for _, fn := range f.Functions {
		for _, finding := range fn.Findings {
			z.add(baselineKey{f.Path, fn.Name, finding.Code})
		}
	}
}

// add counts one finding by whether it is baselined or its level.
//
// TRLC-LINKS: REQ-EMG-155
func (z *summarizer) add(k baselineKey) {
	z.s.Findings++
	switch {
	case z.baselined[k]:
		z.s.Baselined++
	case z.r.LevelOf(k.code) == LevelError:
		z.s.Errors++
	default:
		z.s.Warnings++
	}
}

// Encode renders the report as canonical JSON: two-space indent, keys in
// struct order (maps sorted), no HTML escaping and a trailing newline.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
// ENGMODEL-LINKS: FU-CODE-HEALTH, DO-HEALTH-REPORT
func Encode(r Report) ([]byte, error) {
	return encodeJSON(r)
}

// encodeJSON is the canonical JSON encoding shared by the report and SARIF.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func encodeJSON(v any) ([]byte, error) {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
