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
	out := make([]FileResult, len(results))
	var waived []WaivedFinding
	used := make([]bool, len(waivers))
	for i, r := range results {
		out[i] = r
		out[i].Functions = make([]Function, len(r.Functions))
		for j, f := range r.Functions {
			out[i].Functions[j] = f
			if len(f.Findings) == 0 {
				continue
			}
			kept := make([]Finding, 0, len(f.Findings))
			for _, finding := range f.Findings {
				w := matchWaiver(waivers, r.Path, f.Name, finding.Code)
				if w < 0 {
					kept = append(kept, finding)
					continue
				}
				used[w] = true
				waived = append(waived, WaivedFinding{Path: r.Path, Function: f.Name, Decision: strings.TrimSpace(waivers[w].Decision), Finding: finding})
			}
			if len(kept) == 0 {
				kept = nil
			}
			out[i].Functions[j].Findings = kept
		}
	}
	sort.SliceStable(waived, func(a, b int) bool {
		x, y := waived[a], waived[b]
		if x.Path != y.Path {
			return x.Path < y.Path
		}
		if x.StartLine != y.StartLine {
			return x.StartLine < y.StartLine
		}
		if x.Function != y.Function {
			return x.Function < y.Function
		}
		return codeOrder(x.Code) < codeOrder(y.Code)
	})
	var unused []model.HealthWaiver
	for i, w := range waivers {
		if !used[i] {
			unused = append(unused, w)
		}
	}
	return out, waived, unused
}

// matchWaiver returns the index of the first waiver for a finding, or -1.
//
// TRLC-LINKS: REQ-EMG-154
func matchWaiver(waivers []model.HealthWaiver, filePath, function, code string) int {
	for i, w := range waivers {
		if strings.TrimSpace(w.Code) == code && strings.TrimSpace(w.Function) == function && waiverMatchesPath(w.Path, filePath) {
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
		for _, f := range files {
			if waiverMatchesPath(w.Path, f) {
				out = append(out, w)
				break
			}
		}
	}
	return out
}
