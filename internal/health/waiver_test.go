// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// waiverResults is two files: parse has two findings and an unrelated one.
func waiverResults() []FileResult {
	return []FileResult{
		{Path: "internal/a.go", Language: "go", Functions: []Function{
			{Name: "Parser.parse", StartLine: 10, EndLine: 90, Findings: []Finding{
				{Code: CodeComplexMethod, Severity: SeverityHigh, StartLine: 10, EndLine: 90, Metric: "cyclomatic", Value: 25},
				{Code: CodeLargeMethod, Severity: SeverityWarn, StartLine: 10, EndLine: 90, Metric: "loc", Value: 80},
			}},
			{Name: "other", StartLine: 100, EndLine: 120, Findings: []Finding{
				{Code: CodeComplexMethod, Severity: SeverityWarn, StartLine: 100, EndLine: 120, Metric: "cyclomatic", Value: 11},
			}},
		}},
		{Path: "internal/b.go", Language: "go", Functions: []Function{
			{Name: "Parser.parse", StartLine: 5, EndLine: 50, Findings: []Finding{
				{Code: CodeComplexMethod, Severity: SeverityWarn, StartLine: 5, EndLine: 50, Metric: "cyclomatic", Value: 12},
			}},
		}},
	}
}

// A waiver moves exactly the finding with its code, file and qualified function
// name to the waived list with its decision; every other finding, including the
// same function name in another file, stays. An unused waiver is returned, and
// the input is not modified.
//
// Mutation: matching on the function name alone, or on the code alone, or
// dropping the finding without listing it, fails.
//
// TRLC-LINKS: REQ-EMG-154
func TestApplyWaiversMovesOnlyTheMatchingFinding(t *testing.T) {
	in := waiverResults()
	waivers := []model.HealthWaiver{
		{Code: CodeComplexMethod, Path: "internal/a.go", Function: "Parser.parse", Decision: "ADR-OK"},
		{Code: CodeBumpyRoad, Path: "internal/a.go", Function: "Parser.parse", Decision: "ADR-NEVER"},
		{Code: CodeComplexMethod, Path: "internal/zzz.go", Function: "gone", Decision: "ADR-GONE"},
	}
	out, waived, unused := ApplyWaivers(in, waivers)

	mustEqual(t, len(waived), 1)
	w := waived[0]
	wantEqual(t, [5]any{w.Path, w.Function, w.Code, w.Decision, w.Value}, [5]any{"internal/a.go", "Parser.parse", CodeComplexMethod, "ADR-OK", 25})
	got := out[0].Functions[0].Findings // only large_method is left
	mustEqual(t, len(got), 1)
	wantEqual(t, got[0].Code, CodeLargeMethod)
	wantEqual(t, [2]int{len(out[0].Functions[1].Findings), len(out[1].Functions[0].Findings)}, [2]int{1, 1}) // no other function or file was waived
	wantDeepEqual(t, in, waiverResults())                                                                    // the input is not modified
	mustEqual(t, len(unused), 2)                                                                             // the two waivers that match nothing, in order
	wantEqual(t, [2]string{unused[0].Decision, unused[1].Decision}, [2]string{"ADR-NEVER", "ADR-GONE"})
}

// A path glob of the exclude dialect names several files, a waived list is
// sorted by path and line, and the last finding of a function leaves it
// without findings.
//
// TRLC-LINKS: REQ-EMG-154
func TestApplyWaiversGlobAndOrder(t *testing.T) {
	waivers := []model.HealthWaiver{{Code: CodeComplexMethod, Path: "internal/*.go", Function: "Parser.parse", Decision: "ADR-OK"}}
	out, waived, unused := ApplyWaivers(waiverResults(), waivers)
	mustEqual(t, [2]int{len(waived), len(unused)}, [2]int{2, 0})
	wantEqual(t, [2]string{waived[0].Path, waived[1].Path}, [2]string{"internal/a.go", "internal/b.go"})
	wantEqual(t, out[1].Functions[0].Findings == nil, true) // the last finding leaves the function without any
	out2, w2, u2 := ApplyWaivers(waiverResults(), nil)
	wantEqual(t, [2]int{len(w2), len(u2)}, [2]int{0, 0})
	wantDeepEqual(t, out2, waiverResults()) // no waivers change nothing
}

// WaiversForFiles keeps only the waivers for files in the scan, so a scan of
// some paths does not report the other waivers as unused.
//
// TRLC-LINKS: REQ-EMG-154
func TestWaiversForFiles(t *testing.T) {
	ws := []model.HealthWaiver{{Path: "a.go"}, {Path: "b/*.go"}, {Path: "c.go"}}
	got := WaiversForFiles(ws, []string{"a.go", "b/x.go"})
	mustEqual(t, len(got), 2)
	wantEqual(t, [2]string{got[0].Path, got[1].Path}, [2]string{"a.go", "b/*.go"})
}
