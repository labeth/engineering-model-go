// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"reflect"
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

	if len(waived) != 1 || waived[0].Path != "internal/a.go" || waived[0].Function != "Parser.parse" ||
		waived[0].Code != CodeComplexMethod || waived[0].Decision != "ADR-OK" || waived[0].Value != 25 {
		t.Fatalf("waived = %+v", waived)
	}
	if got := out[0].Functions[0].Findings; len(got) != 1 || got[0].Code != CodeLargeMethod {
		t.Errorf("parse findings after waiver = %+v, want only large_method", got)
	}
	if len(out[0].Functions[1].Findings) != 1 || len(out[1].Functions[0].Findings) != 1 {
		t.Error("a finding of another function or file was waived")
	}
	if !reflect.DeepEqual(in, waiverResults()) {
		t.Error("ApplyWaivers modified its input")
	}
	if len(unused) != 2 || unused[0].Decision != "ADR-NEVER" || unused[1].Decision != "ADR-GONE" {
		t.Errorf("unused = %+v, want the two waivers that match nothing, in order", unused)
	}
}

// A path glob of the exclude dialect names several files, a waived list is
// sorted by path and line, and the last finding of a function leaves it
// without findings.
//
// TRLC-LINKS: REQ-EMG-154
func TestApplyWaiversGlobAndOrder(t *testing.T) {
	waivers := []model.HealthWaiver{{Code: CodeComplexMethod, Path: "internal/*.go", Function: "Parser.parse", Decision: "ADR-OK"}}
	out, waived, unused := ApplyWaivers(waiverResults(), waivers)
	if len(waived) != 2 || waived[0].Path != "internal/a.go" || waived[1].Path != "internal/b.go" || len(unused) != 0 {
		t.Fatalf("waived = %+v unused = %+v", waived, unused)
	}
	if out[1].Functions[0].Findings != nil {
		t.Errorf("b.go parse still has findings: %+v", out[1].Functions[0].Findings)
	}
	if out2, w2, u2 := ApplyWaivers(waiverResults(), nil); len(w2) != 0 || len(u2) != 0 || !reflect.DeepEqual(out2, waiverResults()) {
		t.Error("no waivers must change nothing")
	}
}

// WaiversForFiles keeps only the waivers for files in the scan, so a scan of
// some paths does not report the other waivers as unused.
//
// TRLC-LINKS: REQ-EMG-154
func TestWaiversForFiles(t *testing.T) {
	ws := []model.HealthWaiver{{Path: "a.go"}, {Path: "b/*.go"}, {Path: "c.go"}}
	got := WaiversForFiles(ws, []string{"a.go", "b/x.go"})
	if len(got) != 2 || got[0].Path != "a.go" || got[1].Path != "b/*.go" {
		t.Fatalf("got %+v", got)
	}
}
