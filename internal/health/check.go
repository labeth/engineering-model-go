// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The gate of the delta check (ADR-EMG-053, ADR-EMG-059): the delta report of
// delta.go plus what the tier makes of it. A regression is a new or worsened
// high finding; it is an error where the tier policy says health.regression is
// an error (from product, from mvp with the safety bundle, or where
// lifecycle.raise names it) and a warning below, so a gate fails on a
// regression only where the project has asked for that. Debt that was already
// there never fails the gate, and neither does a finding that a waiver covers.
// Two more things fail it at the same level: a changed file that was skipped
// (its findings are unknown) and a weakening of the health policy against the
// base revision that no accepted policyDecision records (policy.go).

import (
	"errors"
	"fmt"
	"sort"
	"strings"
	"sync"

	"github.com/labeth/engineering-model-go/internal/model"
)

// ErrNotGitRepo reports that the project is not inside a git work tree; the
// check needs one. It is a setup error (the command exits 2).
var ErrNotGitRepo = errors.New("not a git repository")

// CheckCounts counts the findings of the changed files by change class. The
// waived findings are not in it.
type CheckCounts struct {
	New      int `json:"new"`
	Worsened int `json:"worsened"`
	Existing int `json:"existing"`
}

// CheckReport is the delta report with the levels at the project tier. The
// JSON encoding is the DeltaReport object with the extra keys below.
type CheckReport struct {
	DeltaReport
	// Levels is the level (error or warning) of health.regression and of every
	// health code at the tier.
	Levels map[string]string `json:"levels"`
	// RegressionLevel is the level of health.regression at the tier.
	RegressionLevel string      `json:"regressionLevel"`
	Counts          CheckCounts `json:"counts"`
	// Waived are the findings of the changed files that a waiver covers. They
	// are left out of both sides of the comparison, so they are existing and
	// never a regression.
	Waived []WaivedFinding `json:"waived"`
	// Policy lists how the change weakens the health policy against the base
	// revision (see PolicyWeakenings); PolicyLevel is the level of
	// health.policy_weakened and PolicyDecision the accepted decision that
	// records the weakening, if the head names one the base did not.
	Policy         []PolicyChange `json:"policyWeakened"`
	PolicyLevel    string         `json:"policyLevel"`
	PolicyDecision string         `json:"policyDecision,omitempty"`
	// Failed is true when there is a regression and it is an error at the tier,
	// when a changed file was skipped and that is an error at the tier (the
	// level of health.regression), or when the policy is weakened without a
	// recorded decision and that is an error at the tier.
	Failed bool `json:"failed"`
}

// RegressionLevelOf returns the level of health.regression at the project tier
// of the bundle, from the same tier policy the scan uses (ApplyTierPolicy: the
// row of the code, the safety bundle, lifecycle.raise). A bundle without a
// life-cycle tier gets "warning".
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
// ENGMODEL-LINKS: FU-CODE-HEALTH
func RegressionLevelOf(b model.Bundle) string {
	return levelOfCode(b, model.HealthRegressionCode)
}

// CheckLevels is Levels with health.regression and health.policy_weakened
// added (the latter at the tier of the bundle; the report holds the stricter
// level of the comparison, see PolicyLevelOf).
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-158
func CheckLevels(b model.Bundle) map[string]string {
	levels := Levels(b)
	levels[model.HealthRegressionCode] = RegressionLevelOf(b)
	levels[model.HealthPolicyWeakenedCode] = levelOfCode(b, model.HealthPolicyWeakenedCode)
	return levels
}

// countEntries counts the entries by change class.
//
// TRLC-LINKS: REQ-EMG-156
func countEntries(entries []DeltaEntry) CheckCounts {
	var c CheckCounts
	for _, e := range entries {
		switch e.Change {
		case ChangeNew:
			c.New++
		case ChangeWorsened:
			c.Worsened++
		default:
			c.Existing++
		}
	}
	return c
}

// PolicyVerdict is what the comparison of the policy with the base revision
// contributes to a check report.
type PolicyVerdict struct {
	// Changes are the weakenings found; Level is the level of
	// health.policy_weakened (the stricter of the base and the head tier);
	// Decision is the accepted policyDecision of the head that the base did not
	// already name, which records the weakening.
	Changes  []PolicyChange
	Level    string
	Decision string
}

// Blocks reports whether the weakening fails the check: there is one, it is
// not recorded by a decision and it is an error at the tier.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-156
func (v PolicyVerdict) Blocks() bool {
	return len(v.Changes) > 0 && v.Decision == "" && v.Level == LevelError
}

// NewCheckReport adds the levels, the counts and the verdict to a delta
// report. levels must hold health.regression (see CheckLevels); a missing level
// is a warning, so a missing level never fails a gate. A changed file that was
// skipped fails the report where a regression does.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func NewCheckReport(d DeltaReport, levels map[string]string, waived []WaivedFinding, policy PolicyVerdict) CheckReport {
	if levels == nil {
		levels = map[string]string{}
	}
	if waived == nil {
		waived = []WaivedFinding{}
	}
	if d.Skipped == nil {
		d.Skipped = []SkippedFile{}
	}
	level := LevelWarning
	if levels[model.HealthRegressionCode] == LevelError {
		level = LevelError
	}
	if policy.Changes == nil {
		policy.Changes = []PolicyChange{}
	}
	if policy.Level == "" {
		policy.Level = LevelWarning
	}
	failed := (len(d.Blocking) > 0 || len(d.Skipped) > 0) && level == LevelError
	return CheckReport{
		DeltaReport: d, Levels: levels, RegressionLevel: level, Counts: countEntries(d.Entries), Waived: waived,
		Policy: policy.Changes, PolicyLevel: policy.Level, PolicyDecision: policy.Decision,
		Failed: failed || policy.Blocks(),
	}
}

// Transition describes the change of a finding: "(none) -> cyclomatic 21 high"
// for a new one, "cyclomatic 12 warn -> cyclomatic 21 high" for a worsened one.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func (e DeltaEntry) Transition() string {
	after := fmt.Sprintf("%s %d %s", e.Metric, e.ValueAfter, e.Severity)
	if e.SeverityBefore == "" {
		return "(none) -> " + after
	}
	return fmt.Sprintf("%s %d %s -> %s", e.Metric, e.ValueBefore, e.SeverityBefore, after)
}

// EncodeCheck renders the check report as canonical JSON: the same encoding as
// the health report, sorted, with no timestamp and no absolute path.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158, REQ-EMG-159
func EncodeCheck(c CheckReport) ([]byte, error) {
	return encodeJSON(c)
}

// CheckSARIF renders the regressions of a check as SARIF 2.1.0 with the
// writer's types: one rule, health.regression, with the level it has at the
// tier, and one result per regression (the blocking entries). The underlying
// smell code is in the properties of the result. The fingerprint is that of the
// smell code, file and function, so it equals the fingerprint of the same
// finding in the scan output.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-158
func CheckSARIF(c CheckReport) ([]byte, error) {
	level := sarifLevel(c.RegressionLevel)
	results := []sarifResult{}
	for _, e := range c.Blocking {
		results = append(results, sarifResult{
			RuleID: model.HealthRegressionCode, RuleIndex: 0, Level: level,
			Message: sarifText{Text: fmt.Sprintf("%s in %s: %s (%s)", e.Change, e.Function, e.Message, e.Transition())},
			Locations: []sarifLocation{{PhysicalLocation: sarifPhysical{
				ArtifactLocation: sarifArtifact{URI: sarifURI(e.Path), URIBaseID: "%SRCROOT%"},
				Region:           &sarifRegion{StartLine: e.StartLine, EndLine: e.EndLine},
			}}},
			PartialFingerprints: map[string]string{FingerprintKey: FindingFingerprint(e.Code, e.Path, e.Function)},
			Properties: map[string]any{
				"code": e.Code, "function": e.Function, "change": e.Change, "metric": e.Metric,
				"valueBefore": e.ValueBefore, "valueAfter": e.ValueAfter, "severity": string(e.Severity),
			},
		})
	}
	rules := []sarifRule{{
		ID: model.HealthRegressionCode, Name: model.HealthRegressionCode,
		ShortDescription: sarifText{Text: "New or worsened high code health finding against the base revision"},
		DefaultConfig:    sarifConfig{Level: level},
	}}
	results, rules = appendSkippedResults(results, rules, c)
	results, rules = appendPolicyResults(results, rules, c)
	run := sarifRun{
		Tool:    sarifTool{Driver: sarifDriver{Name: "enghealth", Rules: rules}},
		Results: results,
	}
	return encodeJSON(sarifLog{Schema: SARIFSchemaURI, Version: SARIFVersion, Runs: []sarifRun{run}})
}

// appendSkippedResults adds the rule health.file_skipped and one result per
// changed file that was not analysed, at the level of health.regression. The
// rule is added only when there is such a file.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-157
func appendSkippedResults(results []sarifResult, rules []sarifRule, c CheckReport) ([]sarifResult, []sarifRule) {
	if len(c.Skipped) == 0 {
		return results, rules
	}
	level := sarifLevel(c.RegressionLevel)
	index := len(rules)
	rules = append(rules, sarifRule{ID: CodeFileSkipped, Name: CodeFileSkipped,
		ShortDescription: sarifText{Text: "Changed source file that was not analysed"}, DefaultConfig: sarifConfig{Level: level}})
	for _, f := range c.Skipped {
		results = append(results, sarifResult{
			RuleID: CodeFileSkipped, RuleIndex: index, Level: level,
			Message: sarifText{Text: f.Path + " was not analysed: " + f.Reason},
			Locations: []sarifLocation{{PhysicalLocation: sarifPhysical{
				ArtifactLocation: sarifArtifact{URI: sarifURI(f.Path), URIBaseID: "%SRCROOT%"},
			}}},
			PartialFingerprints: map[string]string{FingerprintKey: FindingFingerprint(CodeFileSkipped, f.Path, "")},
		})
	}
	return results, rules
}

// appendPolicyResults adds the rule health.policy_weakened and one result per
// weakening of the policy that no decision records. The rule is added only
// when there is such a weakening.
//
// TRLC-LINKS: REQ-EMG-155, REQ-EMG-157
func appendPolicyResults(results []sarifResult, rules []sarifRule, c CheckReport) ([]sarifResult, []sarifRule) {
	if len(c.Policy) == 0 || c.PolicyDecision != "" {
		return results, rules
	}
	level := sarifLevel(c.PolicyLevel)
	index := len(rules)
	rules = append(rules, sarifRule{ID: model.HealthPolicyWeakenedCode, Name: model.HealthPolicyWeakenedCode,
		ShortDescription: sarifText{Text: "Change that weakens the code health policy against the base revision"}, DefaultConfig: sarifConfig{Level: level}})
	for _, p := range c.Policy {
		results = append(results, sarifResult{
			RuleID: model.HealthPolicyWeakenedCode, RuleIndex: index, Level: level,
			Message:             sarifText{Text: p.Message},
			PartialFingerprints: map[string]string{FingerprintKey: FindingFingerprint(model.HealthPolicyWeakenedCode, p.Aspect, p.Message)},
			Properties:          map[string]any{"aspect": p.Aspect},
		})
	}
	return results, rules
}

// WaivingEngine wraps an Engine and removes the findings that a waiver covers
// (ApplyWaivers), on every file it analyses. The delta check runs on it, so
// the same waivers apply to the base and to the head and a waived finding is
// never new or worsened. It records the findings it waived.
type WaivingEngine struct {
	Engine  Engine
	Waivers []model.HealthWaiver

	mu     sync.Mutex
	waived map[string]WaivedFinding
}

var _ Engine = (*WaivingEngine)(nil)

// Name is the name of the inner engine.
//
// TRLC-LINKS: REQ-EMG-154
func (e *WaivingEngine) Name() string { return e.Engine.Name() }

// Version is the version of the inner engine.
//
// TRLC-LINKS: REQ-EMG-154
func (e *WaivingEngine) Version() string { return e.Engine.Version() }

// Analyze runs the inner engine and applies the waivers to its results.
//
// TRLC-LINKS: REQ-EMG-154, REQ-EMG-156
func (e *WaivingEngine) Analyze(inputs []FileInput, cfg Config) ([]FileResult, error) {
	results, err := e.Engine.Analyze(inputs, cfg)
	if err != nil || len(e.Waivers) == 0 {
		return results, err
	}
	kept, waived, _ := ApplyWaivers(results, e.Waivers)
	e.mu.Lock()
	defer e.mu.Unlock()
	if e.waived == nil {
		e.waived = map[string]WaivedFinding{}
	}
	for _, w := range waived {
		key := w.Path + "\x00" + w.Function + "\x00" + w.Code
		if _, seen := e.waived[key]; !seen {
			e.waived[key] = w
		}
	}
	return kept, nil
}

// Waived lists the findings waived so far, one per file, function and code,
// sorted by path, line and code.
//
// TRLC-LINKS: REQ-EMG-154
func (e *WaivingEngine) Waived() []WaivedFinding {
	e.mu.Lock()
	defer e.mu.Unlock()
	out := make([]WaivedFinding, 0, len(e.waived))
	for _, w := range e.waived {
		out = append(out, w)
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		switch {
		case a.Path != b.Path:
			return a.Path < b.Path
		case a.StartLine != b.StartLine:
			return a.StartLine < b.StartLine
		case a.Function != b.Function:
			return a.Function < b.Function
		}
		return codeOrder(a.Code) < codeOrder(b.Code)
	})
	return out
}

// RequireWorkTree returns ErrNotGitRepo (wrapped, with git's message) when the
// directory of the repo is not inside a git work tree. The delta check calls
// it first, so that a missing repository is named as such and not as a failed
// git command.
//
// TRLC-LINKS: REQ-EMG-156, REQ-EMG-158
func RequireWorkTree(g GitRepo) error {
	out, code, err := g.run("rev-parse", "--is-inside-work-tree")
	if err == nil && strings.TrimSpace(string(out)) == "true" {
		return nil
	}
	if err != nil && code != 128 {
		return err
	}
	return fmt.Errorf("%w: %s is not inside a git work tree; the check compares against a base revision", ErrNotGitRepo, g.Dir)
}
