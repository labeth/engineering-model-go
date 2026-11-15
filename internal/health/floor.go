// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The floor of a file score (ADR-EMG-087): every source file has to score 10.0,
// the highest score, in the code health scan. A file below the floor is
// reported as health.file_below_floor unless the baseline covers it, so the
// baseline is the explicit, shrinking list of exceptions and a new file, or a
// file with no baseline entry, has to be clean at once. The rule is a function
// of the score and of the baseline entries and reads nothing else, so the same
// tree gives the same diagnostics.
//
// A file is covered when at least one baseline entry (path, qualified function
// name and code) names a finding of it that the scan found: a baselined
// finding, or one that grew past its entry. An entry that matches no finding
// covers nothing (it is health.baseline_stale). A waiver covers a file through
// the score: a waived finding is removed before scoring, so a file whose every
// finding is waived scores 10.0 and is not below the floor.

import (
	"fmt"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// CodeFileBelowFloor is the diagnostic code of a file below the floor that no
// baseline entry covers.
const CodeFileBelowFloor = model.HealthFileBelowFloorCode

// FloorCauseCount is how many deductions a floor message names, the largest
// first; the message says how many more there are.
const FloorCauseCount = 3

// FloorCause is one deduction of a file below the floor, with the line of the
// finding that caused it.
type FloorCause struct {
	Code     string `json:"code"`
	Function string `json:"function"`
	// Line is the first line of the finding; 0 when the function is unknown.
	Line int `json:"line"`
	// Points is what the deduction counts in the score of the file.
	Points float64 `json:"points"`
}

// FloorFile is a file whose score is below the floor, with what deducts.
type FloorFile struct {
	Path  string  `json:"path"`
	Score float64 `json:"score"`
	Floor float64 `json:"floor"`
	// Causes are the largest deductions (at most FloorCauseCount) and More the
	// number of the others.
	Causes []FloorCause `json:"causes"`
	More   int          `json:"more"`
}

// BelowFloor reports whether a score is below the floor. A floor of 0 or less
// is no floor.
//
// TRLC-LINKS: REQ-EMG-209
func BelowFloor(score, floor float64) bool {
	return floor > 0 && score < floor
}

// NewFloorFile describes a file below the floor: the path, the score and the
// largest deductions with the line of their finding (the functions give the
// lines). It does not decide whether the file is below the floor or covered.
//
// TRLC-LINKS: REQ-EMG-209
// ENGMODEL-LINKS: FU-CODE-HEALTH
func NewFloorFile(f FileReport, floor float64) FloorFile {
	causes := make([]FloorCause, 0, len(f.Deductions))
	for _, d := range f.Deductions {
		causes = append(causes, FloorCause{Code: d.Code, Function: d.Function, Line: findingLine(f.Functions, d), Points: d.Counted})
	}
	sort.SliceStable(causes, func(i, j int) bool { return causeBefore(causes[i], causes[j]) })
	file := FloorFile{Path: f.Path, Score: f.Score, Floor: floor, Causes: causes}
	if len(causes) > FloorCauseCount {
		file.More = len(causes) - FloorCauseCount
		file.Causes = causes[:FloorCauseCount]
	}
	return file
}

// causeBefore orders the causes: the larger counted points first, then the
// function and the code.
//
// TRLC-LINKS: REQ-EMG-209
func causeBefore(a, b FloorCause) bool {
	switch {
	case a.Points != b.Points:
		return a.Points > b.Points
	case a.Function != b.Function:
		return a.Function < b.Function
	case a.Code != b.Code:
		return codeOrder(a.Code) < codeOrder(b.Code)
	}
	return a.Line < b.Line
}

// findingLine is the first line of the finding a deduction comes from; 0 when
// no function of that name has a finding of the code.
//
// TRLC-LINKS: REQ-EMG-209
func findingLine(fns []Function, d Deduction) int {
	for _, fn := range fns {
		if line := codeLine(fn, d); line > 0 {
			return line
		}
	}
	return 0
}

// codeLine is the first line of the finding of the deduction's code in fn, when
// fn is the deduction's function; else 0.
//
// TRLC-LINKS: REQ-EMG-209
func codeLine(fn Function, d Deduction) int {
	if fn.Name != d.Function {
		return 0
	}
	for _, f := range fn.Findings {
		if f.Code == d.Code {
			return f.StartLine
		}
	}
	return 0
}

// Message is the diagnostic text of a file below the floor that the baseline
// does not cover: the file, its score, the floor, the deductions with function
// and line, and the fix.
//
// TRLC-LINKS: REQ-EMG-209
func (f FloorFile) Message() string {
	var causes []string
	for _, c := range f.Causes {
		where := c.Function
		if c.Line > 0 {
			where = fmt.Sprintf("%s (line %d)", c.Function, c.Line)
		}
		causes = append(causes, fmt.Sprintf("%s in %s, -%.2f", c.Code, where, c.Points))
	}
	deducts := strings.Join(causes, "; ")
	if f.More > 0 {
		deducts += fmt.Sprintf("; and %d more", f.More)
	}
	return fmt.Sprintf("%s scores %.1f, below the floor of %.1f, and no baseline entry covers it; it is deducted by %s; fix: %s",
		f.Path, f.Score, f.Floor, deducts, validate.ConsistencyRemedy(CodeFileBelowFloor))
}

// BaselineCovered returns the paths of the files that the baseline covers: a
// file with at least one finding that an entry names, whatever its value is
// against the recorded one. The results are not modified.
//
// TRLC-LINKS: REQ-EMG-209, REQ-EMG-210
// ENGMODEL-LINKS: FU-CODE-HEALTH
func BaselineCovered(results []FileResult, entries []model.HealthBaselineEntry) map[string]bool {
	covered := map[string]bool{}
	listed := make(map[baselineKey]bool, len(entries))
	for _, e := range entries {
		listed[baselineKey{e.Path, e.Function, e.Code}] = true
	}
	for _, r := range results {
		if hasListedFinding(r, listed) {
			covered[r.Path] = true
		}
	}
	return covered
}

// hasListedFinding reports whether a finding of the result is an entry.
//
// TRLC-LINKS: REQ-EMG-209
func hasListedFinding(r FileResult, listed map[baselineKey]bool) bool {
	for _, fn := range r.Functions {
		if functionListed(r.Path, fn, listed) {
			return true
		}
	}
	return false
}

// functionListed reports whether a finding of the function is an entry.
//
// TRLC-LINKS: REQ-EMG-209
func functionListed(path string, fn Function, listed map[baselineKey]bool) bool {
	for _, f := range fn.Findings {
		if listed[baselineKey{path, fn.Name, f.Code}] {
			return true
		}
	}
	return false
}

// floorPolicy is what decides whether a file below the floor is reported: the
// floor, the files the baseline covers and the level of the code.
type floorPolicy struct {
	floor   float64
	covered map[string]bool
	level   string
}

// floorDiagnostics returns the diagnostic of each file of a report whose score
// is below the floor and that the baseline does not cover. Test files count like
// any other file.
//
// TRLC-LINKS: REQ-EMG-209
func floorDiagnostics(files []FileReport, p floorPolicy) []Diagnostic {
	var out []Diagnostic
	for _, f := range files {
		if BelowFloor(f.Score, p.floor) && !p.covered[f.Path] {
			out = append(out, Diagnostic{Code: CodeFileBelowFloor, Level: p.level, Message: NewFloorFile(f, p.floor).Message(), Path: f.Path})
		}
	}
	return out
}

// floorCounts counts the files of the report whose score is below its floor and
// those of them that the baseline covers: the others are the diagnostics of
// health.file_below_floor.
//
// TRLC-LINKS: REQ-EMG-209
func (r Report) floorCounts() (below, covered int) {
	below = r.filesBelowFloor()
	return below, max(below-r.floorDiagnosticCount(), 0)
}

// filesBelowFloor counts the files whose score is below the floor of the report.
//
// TRLC-LINKS: REQ-EMG-209
func (r Report) filesBelowFloor() int {
	n := 0
	for _, f := range r.Files {
		if BelowFloor(f.Score, r.Floor) {
			n++
		}
	}
	return n
}

// floorDiagnosticCount counts the diagnostics of health.file_below_floor.
//
// TRLC-LINKS: REQ-EMG-209
func (r Report) floorDiagnosticCount() int {
	n := 0
	for _, d := range r.Diagnostics {
		if d.Code == CodeFileBelowFloor {
			n++
		}
	}
	return n
}
