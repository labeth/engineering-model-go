// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"crypto/sha256"
	"encoding/hex"
	"net/url"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/model"
)

// SARIF 2.1.0 export (ADR-EMG-050). New code with no precedent in the
// repository: one run, the tool driver "enghealth", one rule per health code,
// one result per finding. Paths are repository-relative, there is no timestamp,
// and the fingerprint of a result is a hash of code, path and function name,
// so it is stable while lines move.

// SARIFVersion and SARIFSchemaURI identify the SARIF flavour written.
const (
	SARIFVersion   = "2.1.0"
	SARIFSchemaURI = "https://json.schemastore.org/sarif-2.1.0.json"
	// FingerprintKey names the partial fingerprint of every result.
	FingerprintKey = "enghealthFinding/v1"
)

type sarifLog struct {
	Schema  string     `json:"$schema"`
	Version string     `json:"version"`
	Runs    []sarifRun `json:"runs"`
}

type sarifRun struct {
	Tool       sarifTool         `json:"tool"`
	Results    []sarifResult     `json:"results"`
	Properties map[string]string `json:"properties,omitempty"`
}

type sarifTool struct {
	Driver sarifDriver `json:"driver"`
}

type sarifDriver struct {
	Name    string      `json:"name"`
	Version string      `json:"version,omitempty"`
	Rules   []sarifRule `json:"rules"`
}

type sarifText struct {
	Text string `json:"text"`
}

type sarifRule struct {
	ID               string      `json:"id"`
	Name             string      `json:"name"`
	ShortDescription sarifText   `json:"shortDescription"`
	DefaultConfig    sarifConfig `json:"defaultConfiguration"`
}

type sarifConfig struct {
	Level string `json:"level"`
}

type sarifResult struct {
	RuleID              string            `json:"ruleId"`
	RuleIndex           int               `json:"ruleIndex"`
	Level               string            `json:"level"`
	Message             sarifText         `json:"message"`
	Locations           []sarifLocation   `json:"locations,omitempty"`
	PartialFingerprints map[string]string `json:"partialFingerprints"`
	Suppressions        []sarifSuppress   `json:"suppressions,omitempty"`
	Properties          map[string]any    `json:"properties,omitempty"`
}

type sarifSuppress struct {
	Kind          string `json:"kind"`
	Status        string `json:"status"`
	Justification string `json:"justification"`
}

type sarifLocation struct {
	PhysicalLocation sarifPhysical `json:"physicalLocation"`
}

type sarifPhysical struct {
	ArtifactLocation sarifArtifact `json:"artifactLocation"`
	Region           *sarifRegion  `json:"region,omitempty"`
}

type sarifArtifact struct {
	URI       string `json:"uri"`
	URIBaseID string `json:"uriBaseId"`
}

// sarifURI turns a repo-relative path into a URI reference: each segment is
// percent-encoded (space, #, %, [, ], ?, non-ASCII bytes and a colon, which would
// read as a scheme in the first segment), the slashes stay.
//
// TRLC-LINKS: REQ-EMG-157
func sarifURI(p string) string {
	segments := strings.Split(p, "/")
	for i, seg := range segments {
		segments[i] = strings.ReplaceAll(url.PathEscape(seg), ":", "%3A")
	}
	return strings.Join(segments, "/")
}

type sarifRegion struct {
	StartLine int `json:"startLine"`
	EndLine   int `json:"endLine"`
}

// ruleDescriptions are the short descriptions of the rules.
var ruleDescriptions = map[string]string{
	CodeComplexMethod:             "Function with high cyclomatic complexity",
	CodeLargeMethod:               "Function with too many logical lines",
	CodeDeepNesting:               "Function with deeply nested control flow",
	CodeBumpyRoad:                 "Function with several separate nested conditional blocks (bumpy road)",
	CodeComplexConditional:        "Condition with many boolean operators",
	CodeExcessArguments:           "Function with many parameters",
	CodePrimitiveObsession:        "Function whose parameters are mostly primitive types",
	CodeBrainMethod:               "Function that is large, complex and deeply nested (brain method)",
	CodeLargeAssertionBlock:       "Test with a long run of consecutive assertions",
	CodeDuplicatedAssertion:       "Identical assertion blocks in several tests of one file",
	model.HealthWaiverUnusedCode:  "Health waiver that matches no finding",
	model.HealthBaselineStaleCode: "Health baseline entry that matches no finding",
	CodeFileBelowFloor:            "Source file whose score is below the floor and that no baseline entry covers",
	"health.file_skipped":         "Source file that was not analysed",
	"health.file_failed":          "Source file whose analysis failed",
	"health.syntax_error":         "Source file the parser could only partly read",
	"health.rollup_unavailable":   "Unit and requirement roll-ups could not be built",
}

// sarifLevel maps a report level to a SARIF level.
//
// TRLC-LINKS: REQ-EMG-157
func sarifLevel(level string) string {
	if level == LevelError {
		return "error"
	}
	return "warning"
}

// FindingFingerprint is the stable identity of a finding: a hash of its code, the
// file path and the qualified function name. It does not change when lines
// move, and it holds no absolute path.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func FindingFingerprint(code, filePath, function string) string {
	sum := sha256.Sum256([]byte(code + "\x00" + filePath + "\x00" + function))
	return hex.EncodeToString(sum[:16])
}

// SARIF renders the report as SARIF 2.1.0: the rules are the health codes in
// report order (plus the codes of the diagnostics present), the results are the
// findings sorted by path, line and code, with the level of the code at the
// report's tier; a waived finding is a result with an external suppression
// that cites its decision, and a diagnostic is a result without a region. The
// output is deterministic: no timestamp, no absolute path.
//
// TRLC-LINKS: REQ-EMG-157
// ENGMODEL-LINKS: FU-CODE-HEALTH, IF-CLI-ENGHEALTH
func SARIF(r Report) ([]byte, error) {
	rules, index := sarifRules(r)
	b := &sarifBuilder{report: r, baselined: r.baselinedSet(), index: index, results: []sarifResult{}}
	b.addFindings()
	b.addWaived()
	b.addDiagnostics()
	sort.SliceStable(b.results, func(i, j int) bool { return lessSarifResult(b.results[i], b.results[j]) })

	log := sarifLog{Schema: SARIFSchemaURI, Version: SARIFVersion, Runs: []sarifRun{{
		Tool:    sarifTool{Driver: sarifDriver{Name: "enghealth", Version: r.Engine.Version, Rules: rules}},
		Results: b.results,
	}}}
	return encodeJSON(log)
}

// sarifRuleCodes lists the codes that are rules: the health codes, the unused
// waiver code and the codes of the diagnostics present, in report order.
//
// TRLC-LINKS: REQ-EMG-157
func sarifRuleCodes(r Report) []string {
	codes := append(Codes(), model.HealthWaiverUnusedCode)
	seen := map[string]bool{}
	for _, c := range codes {
		seen[c] = true
	}
	var extra []string
	for _, d := range r.Diagnostics {
		if !seen[d.Code] {
			seen[d.Code] = true
			extra = append(extra, d.Code)
		}
	}
	sort.Strings(extra)
	return append(codes, extra...)
}

// sarifRules builds the rule descriptors of the report and the index of each
// code among them.
//
// TRLC-LINKS: REQ-EMG-157
func sarifRules(r Report) ([]sarifRule, map[string]int) {
	codes := sarifRuleCodes(r)
	index := map[string]int{}
	rules := make([]sarifRule, len(codes))
	for i, c := range codes {
		index[c] = i
		desc := ruleDescriptions[c]
		if desc == "" {
			desc = c
		}
		rules[i] = sarifRule{ID: c, Name: c, ShortDescription: sarifText{Text: desc}, DefaultConfig: sarifConfig{Level: sarifLevel(r.LevelOf(c))}}
	}
	return rules, index
}

// sarifBuilder collects the results of one report.
type sarifBuilder struct {
	report    Report
	baselined map[baselineKey]bool
	index     map[string]int
	results   []sarifResult
}

// sarifInput is one result before it is added: the finding, the place it is
// in and its suppressions.
type sarifInput struct {
	code, path, function, message, level string
	region                               *sarifRegion
	props                                map[string]any
	suppressions                         []sarifSuppress
}

// add appends the result of one input.
//
// TRLC-LINKS: REQ-EMG-157
func (b *sarifBuilder) add(in sarifInput) {
	res := sarifResult{
		RuleID: in.code, RuleIndex: b.index[in.code], Level: in.level, Message: sarifText{Text: in.message},
		PartialFingerprints: map[string]string{FingerprintKey: FindingFingerprint(in.code, in.path, in.function)},
		Suppressions:        in.suppressions, Properties: in.props,
	}
	if in.path != "" {
		res.Locations = []sarifLocation{{PhysicalLocation: sarifPhysical{
			ArtifactLocation: sarifArtifact{URI: sarifURI(in.path), URIBaseID: "%SRCROOT%"}, Region: in.region,
		}}}
	}
	b.results = append(b.results, res)
}

// addFindings adds the findings of every function, with the baseline as a
// suppression.
//
// TRLC-LINKS: REQ-EMG-157
func (b *sarifBuilder) addFindings() {
	for _, f := range b.report.Files {
		b.addFileFindings(f)
	}
}

// addFileFindings adds the findings of the functions of one file.
//
// TRLC-LINKS: REQ-EMG-157
func (b *sarifBuilder) addFileFindings(f FileReport) {
	for _, fn := range f.Functions {
		for _, finding := range fn.Findings {
			b.add(b.findingInput(f.Path, fn.Name, finding))
		}
	}
}

// findingInput describes a finding of a function, suppressed when the
// baseline accepts it.
//
// TRLC-LINKS: REQ-EMG-157
func (b *sarifBuilder) findingInput(path, function string, finding Finding) sarifInput {
	var sup []sarifSuppress
	if b.baselined[baselineKey{path, function, finding.Code}] {
		sup = []sarifSuppress{{Kind: "external", Status: "accepted", Justification: "accepted in the health baseline"}}
	}
	return sarifInput{
		code: finding.Code, path: path, function: function, message: finding.Message, level: sarifLevel(b.report.LevelOf(finding.Code)),
		region:       &sarifRegion{StartLine: finding.StartLine, EndLine: finding.EndLine},
		props:        map[string]any{"function": function, "metric": finding.Metric, "value": finding.Value, "severity": string(finding.Severity)},
		suppressions: sup,
	}
}

// addWaived adds the waived findings with their external suppression.
//
// TRLC-LINKS: REQ-EMG-157
func (b *sarifBuilder) addWaived() {
	for _, w := range b.report.Waived {
		b.add(sarifInput{
			code: w.Code, path: w.Path, function: w.Function, message: w.Message, level: sarifLevel(b.report.LevelOf(w.Code)),
			region:       &sarifRegion{StartLine: w.StartLine, EndLine: w.EndLine},
			props:        map[string]any{"function": w.Function, "metric": w.Metric, "value": w.Value, "severity": string(w.Severity)},
			suppressions: []sarifSuppress{{Kind: "external", Status: "accepted", Justification: "waived by decision " + w.Decision}},
		})
	}
}

// addDiagnostics adds the diagnostics, which have no region.
//
// TRLC-LINKS: REQ-EMG-157
func (b *sarifBuilder) addDiagnostics() {
	for _, d := range b.report.Diagnostics {
		b.add(sarifInput{code: d.Code, path: d.Path, function: d.Function, message: d.Message, level: sarifLevel(d.Level)})
	}
}

// lessSarifResult orders results by path, line, rule and fingerprint.
//
// TRLC-LINKS: REQ-EMG-157
func lessSarifResult(a, b sarifResult) bool {
	pa, pb := resultKey(a), resultKey(b)
	if pa.path != pb.path {
		return pa.path < pb.path
	}
	if pa.line != pb.line {
		return pa.line < pb.line
	}
	if a.RuleIndex != b.RuleIndex {
		return a.RuleIndex < b.RuleIndex
	}
	return a.PartialFingerprints[FingerprintKey] < b.PartialFingerprints[FingerprintKey]
}

type resultSortKey struct {
	path string
	line int
}

// resultKey is the path and start line of a result for sorting.
//
// TRLC-LINKS: REQ-EMG-157
func resultKey(r sarifResult) resultSortKey {
	if len(r.Locations) == 0 {
		return resultSortKey{}
	}
	loc := r.Locations[0].PhysicalLocation
	k := resultSortKey{path: loc.ArtifactLocation.URI}
	if loc.Region != nil {
		k.line = loc.Region.StartLine
	}
	return k
}
