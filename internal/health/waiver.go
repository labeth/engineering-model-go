// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// WaivedFinding is a finding that a waiver of the manifest moved out of the
// function it was found in: it stays in the report with the decision that
// justifies it and is never silently dropped (ADR-EMG-051, REQ-EMG-154).
type WaivedFinding struct {
	Path     string `json:"path"`
	Function string `json:"function"`
	// Decision is the accepted decision that justifies the waiver.
	Decision string `json:"decision"`
	Finding
}

// waiverMatchesPath reports whether the path of a waiver names a file: the
// exact path, or a glob of the exclude dialect within the path segments.
//
// TRLC-LINKS: REQ-EMG-154
func waiverMatchesPath(pattern, rel string) bool {
	pattern = strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(pattern)), "./")
	if pattern == rel {
		return true
	}
	matched, _ := path.Match(pattern, rel)
	return matched
}

// ApplyWaivers moves every finding that a waiver covers (same code, the file
// of the waiver path and the qualified function name) out of the results. It
// returns the results without those findings, the waived findings sorted by
// path, line and code, and the waivers that matched no finding. Whether the
// decision of a waiver is accepted is checked by internal/validate and not
// again here. The input is not modified.
//
// TRLC-LINKS: REQ-EMG-154
// ENGMODEL-LINKS: FU-CODE-HEALTH
func ApplyWaivers(results []FileResult, waivers []model.HealthWaiver) ([]FileResult, []WaivedFinding, []model.HealthWaiver) {
	run := &waiverRun{waivers: waivers, used: make([]bool, len(waivers))}
	out := make([]FileResult, len(results))
	for i, r := range results {
		out[i] = run.applyFile(r)
	}
	sort.SliceStable(run.waived, func(a, b int) bool { return lessFindingKey(run.waived[a].sortKey(), run.waived[b].sortKey()) })
	return out, run.waived, run.unused()
}

// waiverRun is the state of one ApplyWaivers call: which waivers matched and
// the findings they took out.
type waiverRun struct {
	waivers []model.HealthWaiver
	used    []bool
	waived  []WaivedFinding
}

// sortKey is the order of the waived findings of a report.
//
// TRLC-LINKS: REQ-EMG-154
func (w WaivedFinding) sortKey() findingKey {
	return findingKey{path: w.Path, function: w.Function, line: w.StartLine, code: codeOrder(w.Code)}
}

// applyFile returns a copy of a result without the findings the waivers cover.
//
// TRLC-LINKS: REQ-EMG-154
func (w *waiverRun) applyFile(r FileResult) FileResult {
	out := r
	out.Functions = make([]Function, len(r.Functions))
	for j, f := range r.Functions {
		out.Functions[j] = f
		if len(f.Findings) > 0 {
			out.Functions[j].Findings = w.applyFunction(r.Path, f)
		}
	}
	return out
}

// applyFunction returns the findings of a function that no waiver covers (nil
// when there are none) and records the others as waived.
//
// TRLC-LINKS: REQ-EMG-154
func (w *waiverRun) applyFunction(path string, f Function) []Finding {
	kept := make([]Finding, 0, len(f.Findings))
	for _, finding := range f.Findings {
		idx := matchWaiver(w.waivers, baselineKey{path, f.Name, finding.Code})
		if idx < 0 {
			kept = append(kept, finding)
			continue
		}
		w.used[idx] = true
		w.waived = append(w.waived, WaivedFinding{Path: path, Function: f.Name, Decision: strings.TrimSpace(w.waivers[idx].Decision), Finding: finding})
	}
	if len(kept) == 0 {
		return nil
	}
	return kept
}

// unused lists the waivers that matched no finding.
//
// TRLC-LINKS: REQ-EMG-154
func (w *waiverRun) unused() []model.HealthWaiver {
	var unused []model.HealthWaiver
	for i, waiver := range w.waivers {
		if !w.used[i] {
			unused = append(unused, waiver)
		}
	}
	return unused
}

// matchWaiver returns the index of the first waiver for a finding (the key
// holds the file, the function and the code), or -1.
//
// TRLC-LINKS: REQ-EMG-154
func matchWaiver(waivers []model.HealthWaiver, k baselineKey) int {
	for i, w := range waivers {
		if strings.TrimSpace(w.Code) == k.code && strings.TrimSpace(w.Function) == k.function && waiverMatchesPath(w.Path, k.path) {
			return i
		}
	}
	return -1
}

// WaiversForFiles keeps the waivers whose path names one of the given files.
// A scan restricted to some paths uses it, so that a waiver for a file outside
// the scan is not reported as unused.
//
// TRLC-LINKS: REQ-EMG-154
func WaiversForFiles(waivers []model.HealthWaiver, files []string) []model.HealthWaiver {
	var out []model.HealthWaiver
	for _, w := range waivers {
		if namesAny(w.Path, files) {
			out = append(out, w)
		}
	}
	return out
}

// namesAny reports whether a waiver path names one of the files.
//
// TRLC-LINKS: REQ-EMG-154
func namesAny(waiverPath string, files []string) bool {
	for _, f := range files {
		if waiverMatchesPath(waiverPath, f) {
			return true
		}
	}
	return false
}
