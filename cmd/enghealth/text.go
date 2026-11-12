// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"fmt"
	"strings"

	engmodel "github.com/labeth/engineering-model-go"
)

// worstFunctionCount is how many functions the text report lists.
const worstFunctionCount = 10

// renderText is the short summary of a report: the project score and band, the
// score of each functional unit, the worst functions with their findings and
// the counts of errors and warnings at the tier. The scores are this project's
// own and not CodeScene scores.
//
// TRLC-LINKS: REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func renderText(r engmodel.HealthReport) string {
	var b strings.Builder
	p := r.Rollups.Project
	fmt.Fprintf(&b, "enghealth: code health, engine %s %s, scoring version %d, tier %s\n", safeText(r.Engine.Name), safeText(r.Engine.Version), r.Engine.ScoringVersion, tierName(r.Tier))
	fmt.Fprintf(&b, "project: %.1f %s (%d production files, %d lines; %d green, %d yellow, %d red)\n",
		p.Score, p.Band, p.Files, p.LOC, p.Bands.Green, p.Bands.Yellow, p.Bands.Red)
	if p.Worst != nil {
		fmt.Fprintf(&b, "worst file: %s (%.1f)\n", safeText(p.Worst.Path), p.Worst.Score)
	}
	if t := r.Rollups.Tests; t != nil {
		fmt.Fprintf(&b, "tests: %.1f %s (%d test files; %d green, %d yellow, %d red)\n", t.Score, t.Band, t.Files, t.Bands.Green, t.Bands.Yellow, t.Bands.Red)
	}
	if len(r.Rollups.Units) > 0 {
		fmt.Fprintln(&b, "\nfunctional units:")
		for _, u := range r.Rollups.Units {
			fmt.Fprintf(&b, "  %-32s %4.1f %-6s %3d files", safeText(u.Unit), u.Score, u.Band, u.Files)
			if u.Worst != nil {
				fmt.Fprintf(&b, ", worst %s (%.1f)", safeText(u.Worst.Path), u.Worst.Score)
			}
			fmt.Fprintln(&b)
		}
	}
	if worst := engmodel.WorstHealthFunctions(r, worstFunctionCount); len(worst) > 0 {
		fmt.Fprintf(&b, "\nworst %d functions:\n", len(worst))
		for _, w := range worst {
			fmt.Fprintf(&b, "  %s:%d %s (%.1f points)\n", safeText(w.Path), w.Function.StartLine, safeText(w.Function.Name), w.Points)
			for _, f := range w.Function.Findings {
				fmt.Fprintf(&b, "      %-33s %-4s %s [%s]\n", f.Code, f.Severity, safeText(f.Message), findingLevel(r, w.Path, w.Function.Name, f.Code))
			}
		}
	}
	if len(r.Diagnostics) > 0 {
		fmt.Fprintln(&b, "\ndiagnostics:")
		for _, d := range r.Diagnostics {
			fmt.Fprintf(&b, "  %s [%s] %s\n", safeText(d.Code), safeText(d.Level), safeText(d.Message))
		}
	}
	s := r.Summary
	fmt.Fprintf(&b, "\nfindings: %d (%d errors, %d warnings at tier %s), %d waived", s.Findings, s.Errors, s.Warnings, tierName(r.Tier), s.Waived)
	if s.Baselined > 0 {
		fmt.Fprintf(&b, ", %d baselined (counted in the findings and the score, not in the errors or warnings)", s.Baselined)
	}
	fmt.Fprintln(&b)
	if p.Files == 0 {
		fmt.Fprintln(&b, "no production source files were found")
	}
	return b.String()
}

// findingLevel is the level shown for a finding: "baselined" when the baseline
// accepts it, otherwise its level at the tier.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func findingLevel(r engmodel.HealthReport, path, function, code string) string {
	if r.IsBaselined(path, function, code) {
		return "baselined"
	}
	return r.LevelOf(code)
}
