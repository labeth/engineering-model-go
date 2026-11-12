// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"path"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// DefaultExcludes are the paths every scan leaves out in addition to
// health.exclude of the manifest: generated and third-party code. They use the
// dialect of inferenceHints.exclude and are anchored at the module root.
// testdata/ is not excluded; its fixtures are named *.txt and are therefore no
// source files.
//
// TRLC-LINKS: REQ-EMG-153
func DefaultExcludes() []string {
	return []string{"generated/**", "vendor/**", "node_modules/**", ".engmod/**", ".git/**"}
}

// PathMatches reports whether a module-relative path matches a pattern of the
// inferenceHints.exclude dialect: path.Match against the path and each of its
// ancestor directories (so dir, dir/, dir/* and dir/** all match everything
// below dir), anchored at the module root. A pattern the dialect refuses
// (validate.ExcludePatternProblem) matches nothing. The same dialect as the
// code scan's exclusion in internal/codescan.
//
// TRLC-LINKS: REQ-EMG-153
func PathMatches(pattern, rel string) bool {
	if validate.ExcludePatternProblem(pattern) != "" {
		return false
	}
	p := strings.TrimPrefix(filepath.ToSlash(strings.TrimSpace(pattern)), "./")
	p = strings.TrimSuffix(strings.TrimSuffix(p, "/**"), "/")
	if p == "" {
		return false
	}
	for prefix := rel; prefix != "." && prefix != "/" && prefix != ""; prefix = path.Dir(prefix) {
		if matched, _ := path.Match(p, prefix); matched {
			return true
		}
	}
	return false
}

// Excluded reports whether a module-relative file path is left out of the
// scan: it matches health.exclude of the manifest (h may be nil) or a default
// exclude.
//
// TRLC-LINKS: REQ-EMG-153
func Excluded(h *model.ManifestHealth, rel string) bool {
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	for _, pattern := range DefaultExcludes() {
		if PathMatches(pattern, rel) {
			return true
		}
	}
	if h == nil {
		return false
	}
	for _, pattern := range h.Exclude {
		if PathMatches(pattern, rel) {
			return true
		}
	}
	return false
}

// ConfigFor builds the engine Config of one file. The base is
// DefaultThresholdsForTier(tier) (the tier is empty without a life-cycle
// tier); health.thresholds of the manifest replace the limits of a code; the
// first health.overrides entry whose path matches the file replaces the limits
// of its codes again. A threshold always replaces both limits of its code.
// The result holds the full effective limits, so it is also the part of a
// cache key that depends on the configuration.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-155
// ENGMODEL-LINKS: FU-CODE-HEALTH
func ConfigFor(tier string, h *model.ManifestHealth, rel string) Config {
	thresholds := DefaultThresholdsForTier(tier)
	if h == nil {
		return Config{Thresholds: thresholds}
	}
	for code, t := range h.Thresholds {
		thresholds[code] = Threshold{Warn: t.Warn, High: t.High}
	}
	rel = strings.TrimPrefix(filepath.ToSlash(rel), "./")
	for _, o := range h.Overrides {
		if !PathMatches(o.Path, rel) {
			continue
		}
		for code, t := range o.Thresholds {
			thresholds[code] = Threshold{Warn: t.Warn, High: t.High}
		}
		break
	}
	return Config{Thresholds: thresholds}
}

// WithTier returns the bundle with the life-cycle tier replaced, keeping the
// bundles and the raised codes of its lifecycle block. It is how a caller
// scans a model at another tier than the one it declares.
//
// TRLC-LINKS: REQ-EMG-155
func WithTier(b model.Bundle, tier string) model.Bundle {
	var lifecycle model.ManifestLifecycle
	if b.Manifest.Lifecycle != nil {
		lifecycle = *b.Manifest.Lifecycle
	}
	lifecycle.Tier = tier
	b.Manifest.Lifecycle = &lifecycle
	return b
}

// Levels returns the level of every health code at the project tier of the
// bundle: "error" or "warning". It asks the tier policy of internal/validate
// (the row of each code, the safety bundle, lifecycle.raise), so a bundle
// without a lifecycle tier gets "warning" for every code. The codes
// health.file_skipped, health.file_failed, health.syntax_error and
// health.rollup_unavailable (what the analysis could not see) have the level
// of health.regression.
//
// TRLC-LINKS: REQ-EMG-155
// ENGMODEL-LINKS: FU-CODE-HEALTH
func Levels(b model.Bundle) map[string]string {
	codes := append(Codes(), model.HealthWaiverUnusedCode, model.HealthBaselineStaleCode)
	diags := make([]validate.Diagnostic, len(codes))
	for i, code := range codes {
		diags[i] = validate.Diagnostic{Code: code, Severity: validate.SeverityWarning}
	}
	out := make(map[string]string, len(codes)+2)
	for _, d := range validate.ApplyTierPolicy(b, diags) {
		out[d.Code] = LevelWarning
		if d.Severity == validate.SeverityError {
			out[d.Code] = LevelError
		}
	}
	// A file that was skipped or failed hides its findings, so it blocks where
	// a regression blocks, through the same tier-policy path.
	for _, code := range []string{CodeFileSkipped, CodeFileFailed, CodeSyntaxError, CodeRollupUnavailable} {
		out[code] = RegressionLevelOf(b)
	}
	return out
}

// Levels of a finding at the project tier.
const (
	LevelError   = "error"
	LevelWarning = "warning"
)

// Relevel returns the report with the tier, the bundles and the level of
// every code taken from the bundle, and the summary recomputed. It is how a
// report of another engine, or a report made at another tier, is read at the
// tier of a model. Diagnostics of the health codes get the level of their code.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH
func Relevel(r Report, b model.Bundle) Report {
	r.Tier, _, _ = validate.EffectiveTier(b)
	r.Bundles = nil
	if b.Manifest.Lifecycle != nil {
		r.Bundles = b.Manifest.Lifecycle.EffectiveBundles()
	}
	r.Levels = Levels(b)
	diags := make([]Diagnostic, len(r.Diagnostics))
	for i, d := range r.Diagnostics {
		if _, ok := r.Levels[d.Code]; ok {
			d.Level = r.LevelOf(d.Code)
		}
		diags[i] = d
	}
	r.Diagnostics = diags
	r.Summary = r.Summarize()
	return r
}
