// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"fmt"
	"sort"

	"github.com/labeth/engineering-model-go/internal/model"
)

// The committed baseline of accepted debt (ADR-EMG-059). The file format and
// its reader live in internal/model; this file applies a loaded baseline to
// the findings of a scan and derives a baseline from them.
//
// Semantics, in one place:
//   - A finding whose (path, qualified function name, code) is an entry of the
//     baseline is baselined. It stays in the file's functions, so the score,
//     the deductions, the roll-ups, the SARIF results and the delta check see it
//     exactly as without a baseline; the report lists it in addition under
//     "baselined" and the summary counts it separately: it is neither an error
//     nor a warning of the gate. (A waived finding, by contrast, is removed
//     before scoring.)
//   - A finding is baselined only while its metric value is at most the value
//     the entry recorded; one that grew past it is an ordinary finding at the
//     level of its code, with a message that says what the baseline allowed.
//   - A regression is never baselined: the delta check compares the working
//     tree with a base revision and does not read the baseline.
//   - An entry that matches no finding is stale and reported as
//     health.baseline_stale, so the baseline can only shrink.

// BaselinedFinding is a finding that the baseline of the manifest accepts. The
// finding itself stays in its function; this entry marks it.
type BaselinedFinding struct {
	Path     string `json:"path"`
	Function string `json:"function"`
	Finding
}

// baselineKey identifies a finding the way a baseline entry does.
type baselineKey struct{ path, function, code string }

// baselineResult is what applying a baseline found.
type baselineResult struct {
	baselined []BaselinedFinding
	stale     []model.HealthBaselineEntry
	// grown maps the key of an entry to its recorded value for every entry
	// that matched a finding whose value is above it.
	grown map[baselineKey]int
}

// ApplyBaseline marks the findings of the results that an entry of the
// baseline covers (same file, qualified function name and code) and whose
// current value is at most the value the entry recorded. It returns the
// baselined findings sorted by path, line, function and code, and the entries
// that match no finding, in canonical order. A finding above the recorded value
// is not baselined, so debt that grows is not accepted. The results are not
// modified and an entry matches every function of that name in the file.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155, REQ-EMG-161
// ENGMODEL-LINKS: FU-CODE-HEALTH
func ApplyBaseline(results []FileResult, entries []model.HealthBaselineEntry) ([]BaselinedFinding, []model.HealthBaselineEntry) {
	r := applyBaseline(results, entries)
	return r.baselined, r.stale
}

// applyBaseline is ApplyBaseline with the entries whose finding grew.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155, REQ-EMG-161
func applyBaseline(results []FileResult, entries []model.HealthBaselineEntry) baselineResult {
	if len(entries) == 0 {
		return baselineResult{}
	}
	want := make(map[baselineKey]*baselineWant, len(entries))
	for _, e := range entries {
		want[baselineKey{e.Path, e.Function, e.Code}] = &baselineWant{value: e.Value}
	}
	out := baselineResult{baselined: collectBaselined(results, want)}
	sort.SliceStable(out.baselined, func(a, b int) bool {
		return lessFindingKey(out.baselined[a].sortKey(), out.baselined[b].sortKey())
	})
	stale, grown := staleAndGrown(want)
	out.stale = model.SortHealthBaselineEntries(stale)
	out.grown = grown
	return out
}

// sortKey is the order of the baselined findings of a report.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155, REQ-EMG-161
func (b BaselinedFinding) sortKey() findingKey {
	return findingKey{path: b.Path, function: b.Function, line: b.StartLine, code: codeOrder(b.Code)}
}

// staleAndGrown reads the matched entries: the entries no finding matched, and
// the recorded value of each entry whose finding grew.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155, REQ-EMG-161
func staleAndGrown(want map[baselineKey]*baselineWant) ([]model.HealthBaselineEntry, map[baselineKey]int) {
	var stale []model.HealthBaselineEntry
	grown := map[baselineKey]int{}
	for k, w := range want {
		switch {
		case !w.matched:
			stale = append(stale, model.HealthBaselineEntry{Path: k.path, Function: k.function, Code: k.code, Value: w.value})
		case w.grown:
			grown[k] = w.value
		}
	}
	return stale, grown
}

// baselineWant is the state of one entry while the findings are matched.
type baselineWant struct {
	value          int
	matched, grown bool
}

// collectBaselined returns the findings of the results that an entry accepts
// and records on each entry whether a finding matched it and whether one of
// them is above the recorded value.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func collectBaselined(results []FileResult, want map[baselineKey]*baselineWant) []BaselinedFinding {
	var baselined []BaselinedFinding
	for _, r := range results {
		for _, fn := range r.Functions {
			baselined = append(baselined, collectFunction(r.Path, fn, want)...)
		}
	}
	return baselined
}

// collectFunction returns the findings of one function that an entry accepts.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func collectFunction(path string, fn Function, want map[baselineKey]*baselineWant) []BaselinedFinding {
	var baselined []BaselinedFinding
	for _, f := range fn.Findings {
		w, ok := want[baselineKey{path, fn.Name, f.Code}]
		if !ok {
			continue
		}
		w.matched = true
		if f.Value > w.value {
			w.grown = true
			continue
		}
		baselined = append(baselined, BaselinedFinding{Path: path, Function: fn.Name, Finding: f})
	}
	return baselined
}

// annotateGrown adds to the message of every finding that grew past its
// baseline entry what the baseline allowed, so a reader sees why a function
// that is listed in the baseline still counts. It copies what it changes.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func annotateGrown(files []FileReport, grown map[baselineKey]int) {
	if len(grown) == 0 {
		return
	}
	for i := range files {
		annotateFile(&files[i], grown)
	}
}

// annotateFile annotates the grown findings of the functions of one file.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func annotateFile(file *FileReport, grown map[baselineKey]int) {
	for j, fn := range file.Functions {
		if copied := annotateFunction(file.Path, fn, grown); copied != nil {
			file.Functions[j].Findings = copied
		}
	}
}

// annotateFunction returns a copy of the findings of fn with the grown ones
// annotated, or nil when none grew.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func annotateFunction(path string, fn Function, grown map[baselineKey]int) []Finding {
	var copied []Finding
	for k, f := range fn.Findings {
		recorded, ok := grown[baselineKey{path, fn.Name, f.Code}]
		if !ok || f.Value <= recorded {
			continue
		}
		if copied == nil {
			copied = append([]Finding(nil), fn.Findings...)
		}
		copied[k].Message = fmt.Sprintf("%s (the baseline accepts %s %d, now %d: debt that grows is not baselined)", f.Message, f.Metric, recorded, f.Value)
	}
	return copied
}

// baselinedSet is the set of findings the report marks as baselined.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func (r Report) baselinedSet() map[baselineKey]bool {
	if len(r.Baselined) == 0 {
		return nil
	}
	set := make(map[baselineKey]bool, len(r.Baselined))
	for _, b := range r.Baselined {
		set[baselineKey{b.Path, b.Function, b.Code}] = true
	}
	return set
}

// IsBaselined reports whether the finding of a code in a function of a file is
// marked as baselined by the report.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-161
func (r Report) IsBaselined(path, function, code string) bool {
	return r.baselinedSet()[baselineKey{path, function, code}]
}

// BaselineEntries derives the baseline of the findings that remain in a
// report: one entry per (path, function, code) with the metric value of the
// finding (the largest when a name occurs twice), sorted, for every finding that
// keep accepts (a nil keep accepts all). A report made without a baseline holds
// every unwaived finding; a waived finding was removed before and is not
// baselined, because a waiver already accepts it.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
// ENGMODEL-LINKS: FU-CODE-HEALTH
func BaselineEntries(r Report, keep func(code string) bool) []model.HealthBaselineEntry {
	var entries []model.HealthBaselineEntry
	for _, f := range r.Files {
		for _, fn := range f.Functions {
			entries = append(entries, functionBaselineEntries(f.Path, fn, keep)...)
		}
	}
	return model.SortHealthBaselineEntries(entries)
}

// functionBaselineEntries derives the entries of the findings of one function
// that keep accepts.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func functionBaselineEntries(path string, fn Function, keep func(code string) bool) []model.HealthBaselineEntry {
	var entries []model.HealthBaselineEntry
	for _, finding := range fn.Findings {
		if keep == nil || keep(finding.Code) {
			entries = append(entries, model.HealthBaselineEntry{Path: path, Function: fn.Name, Code: finding.Code, Value: finding.Value})
		}
	}
	return entries
}

// BaselineDiff compares a baseline file with the entries a scan would write.
// stale are the entries of the file that no longer match a finding (or whose
// recorded value differs from the current one) and missing the findings that
// the file does not list with their current value. Both are sorted.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func BaselineDiff(have, want []model.HealthBaselineEntry) (stale, missing []model.HealthBaselineEntry) {
	inHave, inWant := entrySet(have), entrySet(want)
	return model.SortHealthBaselineEntries(entriesNotIn(have, inWant)), model.SortHealthBaselineEntries(entriesNotIn(want, inHave))
}

// entrySet is the set of the entries of a list.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func entrySet(entries []model.HealthBaselineEntry) map[model.HealthBaselineEntry]bool {
	set := make(map[model.HealthBaselineEntry]bool, len(entries))
	for _, e := range entries {
		set[e] = true
	}
	return set
}

// entriesNotIn lists the entries that are not in the set, in order.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func entriesNotIn(entries []model.HealthBaselineEntry, set map[model.HealthBaselineEntry]bool) []model.HealthBaselineEntry {
	var out []model.HealthBaselineEntry
	for _, e := range entries {
		if !set[e] {
			out = append(out, e)
		}
	}
	return out
}
