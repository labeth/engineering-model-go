// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"path/filepath"
	"sort"
	"strings"
)

// The roll-ups of ADR-EMG-049: file scores reach the elements the model
// knows. Production files and test files are rolled up apart; the unit,
// requirement and project levels see production code only, and the tests block
// reports the test files.

// BandCounts is the number of files in each band.
type BandCounts struct {
	Green  int `json:"green"`
	Yellow int `json:"yellow"`
	Red    int `json:"red"`
}

// WorstFile names the file with the lowest score of an aggregate.
type WorstFile struct {
	Path  string  `json:"path"`
	Score float64 `json:"score"`
}

// Aggregate is the roll-up of a set of files: the LOC-weighted mean score
// (a file counts with its LOC, at least 1), the worst file and the number of
// files per band. A set without files has score 10.0, band green and no worst
// file; Files = 0 tells it apart.
type Aggregate struct {
	Files int        `json:"files"`
	LOC   int        `json:"loc"`
	Score float64    `json:"score"`
	Band  string     `json:"band"`
	Worst *WorstFile `json:"worst,omitempty"`
	Bands BandCounts `json:"bands"`
}

// UnitRollup is the roll-up of the production files attributed to one
// functional unit by the file-level ENGMODEL-OWNER-UNIT marker.
type UnitRollup struct {
	Unit string `json:"unit"`
	Aggregate
}

// RequirementRollup is the roll-up of one requirement: the functions that
// marker-linked symbols with TRLC-LINKS to the requirement lie in. Score is
// the worst score of the files of those functions.
type RequirementRollup struct {
	Requirement string  `json:"requirement"`
	Functions   int     `json:"functions"`
	Findings    int     `json:"findings"`
	Score       float64 `json:"score"`
	Band        string  `json:"band"`
	WorstFile   string  `json:"worstFile"`
}

// Rollups holds every roll-up of a report.
type Rollups struct {
	Project      Aggregate           `json:"project"`
	Units        []UnitRollup        `json:"units"`
	Requirements []RequirementRollup `json:"requirements"`
	// Tests rolls up the test files; it is absent when there are none.
	Tests *Aggregate `json:"tests,omitempty"`
}

// SymbolLink is a marker-linked symbol of the code map: the declaration line
// of a symbol and the requirements its TRLC-LINKS name.
type SymbolLink struct {
	Path         string
	Line         int
	Requirements []string
}

// aggregate rolls a set of files up.
//
// TRLC-LINKS: REQ-EMG-152
func aggregate(files []FileReport) Aggregate {
	if len(files) == 0 {
		return Aggregate{Score: MaxScore, Band: BandGreen}
	}
	var a Aggregate
	var weighted float64
	var weight int
	for i, f := range files {
		w := f.LOC
		if w < 1 {
			w = 1
		}
		weighted += f.Score * float64(w)
		weight += w
		a.Files++
		a.LOC += f.LOC
		switch f.Band {
		case BandGreen:
			a.Bands.Green++
		case BandYellow:
			a.Bands.Yellow++
		default:
			a.Bands.Red++
		}
		// files are sorted by path, so the first of equal scores wins.
		if i == 0 || f.Score < a.Worst.Score {
			a.Worst = &WorstFile{Path: f.Path, Score: f.Score}
		}
	}
	a.Score = RoundTo(weighted/float64(weight), 1)
	a.Band = BandOf(a.Score)
	return a
}

// BuildRollups computes the roll-ups of a set of file reports sorted by path.
// owners maps a file path to its functional unit (a file without an entry, or
// with the empty value or "unresolved", counts for the project only) and
// links are the marker-linked symbols used to reach the requirements.
//
// TRLC-LINKS: REQ-EMG-152
// ENGMODEL-LINKS: FU-CODE-HEALTH
func BuildRollups(files []FileReport, owners map[string]string, links []SymbolLink) Rollups {
	var production, tests []FileReport
	byUnit := map[string][]FileReport{}
	byPath := map[string]*FileReport{}
	for i := range files {
		f := files[i]
		byPath[f.Path] = &files[i]
		if f.IsTest {
			tests = append(tests, f)
			continue
		}
		production = append(production, f)
		if unit := strings.TrimSpace(owners[f.Path]); unit != "" && unit != "unresolved" {
			byUnit[unit] = append(byUnit[unit], f)
		}
	}
	r := Rollups{Project: aggregate(production), Units: []UnitRollup{}, Requirements: requirementRollups(byPath, links)}
	if len(tests) > 0 {
		t := aggregate(tests)
		r.Tests = &t
	}
	units := make([]string, 0, len(byUnit))
	for unit := range byUnit {
		units = append(units, unit)
	}
	sort.Strings(units)
	for _, unit := range units {
		r.Units = append(r.Units, UnitRollup{Unit: unit, Aggregate: aggregate(byUnit[unit])})
	}
	return r
}

// requirementRollups joins the functions of production files to the symbols
// by path and line range (the innermost function whose range holds the symbol
// line) and rolls them up per requirement, sorted by requirement ID.
//
// TRLC-LINKS: REQ-EMG-152
func requirementRollups(byPath map[string]*FileReport, links []SymbolLink) []RequirementRollup {
	type fnKey struct {
		path string
		fn   int
	}
	perReq := map[string]map[fnKey]bool{}
	for _, link := range links {
		f := byPath[filepath.ToSlash(link.Path)]
		if f == nil || f.IsTest {
			continue
		}
		fn := innermostFunction(f.Functions, link.Line)
		if fn < 0 {
			continue
		}
		for _, req := range link.Requirements {
			if perReq[req] == nil {
				perReq[req] = map[fnKey]bool{}
			}
			perReq[req][fnKey{f.Path, fn}] = true
		}
	}
	reqs := make([]string, 0, len(perReq))
	for req := range perReq {
		reqs = append(reqs, req)
	}
	sort.Strings(reqs)
	out := make([]RequirementRollup, 0, len(reqs))
	for _, req := range reqs {
		keys := make([]fnKey, 0, len(perReq[req]))
		for k := range perReq[req] {
			keys = append(keys, k)
		}
		sort.Slice(keys, func(a, b int) bool {
			if keys[a].path != keys[b].path {
				return keys[a].path < keys[b].path
			}
			return keys[a].fn < keys[b].fn
		})
		row := RequirementRollup{Requirement: req, Functions: len(keys)}
		for i, k := range keys {
			f := byPath[k.path]
			row.Findings += len(f.Functions[k.fn].Findings)
			if i == 0 || f.Score < row.Score {
				row.Score, row.WorstFile = f.Score, f.Path
			}
		}
		row.Band = BandOf(row.Score)
		out = append(out, row)
	}
	return out
}

// innermostFunction returns the index of the function with the smallest line
// range that holds line, or -1.
//
// TRLC-LINKS: REQ-EMG-152
func innermostFunction(fns []Function, line int) int {
	best := -1
	for i, f := range fns {
		if line < f.StartLine || line > f.EndLine {
			continue
		}
		if best < 0 || f.EndLine-f.StartLine < fns[best].EndLine-fns[best].StartLine {
			best = i
		}
	}
	return best
}

// RankedFunction is a function with findings and the points they deduct.
type RankedFunction struct {
	Path     string
	Function Function
	Points   float64
}

// WorstFunctions returns the n functions of production files with the most
// deducted points (the sum of weight x severity factor of their findings),
// the worst first; ties go to the path and the line. It ranks functions, not
// files, and is the list the text report shows.
//
// TRLC-LINKS: REQ-EMG-152
// ENGMODEL-LINKS: FU-CODE-HEALTH
func WorstFunctions(r Report, n int) []RankedFunction {
	var ranked []RankedFunction
	for _, f := range r.Files {
		if f.IsTest {
			continue
		}
		for _, fn := range f.Functions {
			points := 0.0
			for _, finding := range fn.Findings {
				points += weights[finding.Code] * SeverityFactor(finding.Severity)
			}
			if points > 0 {
				ranked = append(ranked, RankedFunction{Path: f.Path, Function: fn, Points: RoundTo(points, 4)})
			}
		}
	}
	sort.SliceStable(ranked, func(i, j int) bool {
		a, b := ranked[i], ranked[j]
		if a.Points != b.Points {
			return a.Points > b.Points
		}
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		return a.Function.StartLine < b.Function.StartLine
	})
	if n >= 0 && len(ranked) > n {
		ranked = ranked[:n]
	}
	return ranked
}
