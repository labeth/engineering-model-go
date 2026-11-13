// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"bytes"
	"encoding/json"
	"sort"
	"strings"
	"testing"
)

// sarifDoc renders the sample report as SARIF and decodes it generically.
func sarifDoc(t *testing.T, r Report) (map[string]any, []byte) {
	t.Helper()
	data, err := SARIF(r)
	if err != nil {
		t.Fatal(err)
	}
	var doc map[string]any
	if err := json.Unmarshal(data, &doc); err != nil {
		t.Fatalf("SARIF is not JSON: %v", err)
	}
	return doc, data
}

// The SARIF export has the required SARIF 2.1.0 structure: version, $schema, one
// run with the tool driver, a rule per health code, and one result per finding
// (plus the waived finding as a suppressed result and the diagnostic) with
// ruleId, ruleIndex, level, message, a physicalLocation with a relative uri and
// a region, and a partial fingerprint. The level is the level at the tier.
//
// This is a minimal structural check, not the full SARIF schema.
//
// Mutation: an absolute uri, a missing rule, a level that ignores the tier or a
// result without region fails.
//
// TRLC-LINKS: REQ-EMG-157
func TestSARIFStructure(t *testing.T) {
	doc, data := sarifDoc(t, BuildReport(sampleResults(), sampleOptions()))
	if doc["version"] != "2.1.0" || !strings.Contains(doc["$schema"].(string), "sarif") {
		t.Fatalf("version/schema = %v %v", doc["version"], doc["$schema"])
	}
	runs := doc["runs"].([]any)
	if len(runs) != 1 {
		t.Fatalf("%d runs, want one", len(runs))
	}
	run := runs[0].(map[string]any)
	driver := run["tool"].(map[string]any)["driver"].(map[string]any)
	if driver["name"] != "enghealth" {
		t.Errorf("driver name = %v", driver["name"])
	}
	rules := driver["rules"].([]any)
	if len(rules) != len(Codes())+1 {
		t.Errorf("%d rules, want %d (the codes and the unused waiver)", len(rules), len(Codes())+1)
	}
	ruleIDs := map[string]int{}
	for i, r := range rules {
		rule := r.(map[string]any)
		id := rule["id"].(string)
		ruleIDs[id] = i
		if rule["shortDescription"].(map[string]any)["text"] == "" || rule["defaultConfiguration"].(map[string]any)["level"] != "error" {
			t.Errorf("rule %s is incomplete: %v", id, rule)
		}
	}
	for _, code := range Codes() {
		if _, ok := ruleIDs[code]; !ok {
			t.Errorf("no rule for %s", code)
		}
	}

	// Findings in the sample after the waiver: complex_method, the test's
	// large_assertion_block; plus the waived excess_arguments and the unused-waiver diagnostic.
	results := run["results"].([]any)
	if len(results) != 4 {
		t.Fatalf("%d results, want 4", len(results))
	}
	var suppressed, withRegion int
	for _, r := range results {
		res := r.(map[string]any)
		id := res["ruleId"].(string)
		if int(res["ruleIndex"].(float64)) != ruleIDs[id] || res["level"] != "error" || res["message"].(map[string]any)["text"] == "" {
			t.Errorf("result %v: bad rule index, level or message", res)
		}
		if res["partialFingerprints"].(map[string]any)[FingerprintKey] == "" {
			t.Errorf("result %v has no fingerprint", res)
		}
		phys := res["locations"].([]any)[0].(map[string]any)["physicalLocation"].(map[string]any)
		uri := phys["artifactLocation"].(map[string]any)["uri"].(string)
		if strings.HasPrefix(uri, "/") || strings.Contains(uri, "..") || strings.Contains(uri, ":") {
			t.Errorf("uri %q is not relative", uri)
		}
		if region, ok := phys["region"].(map[string]any); ok {
			withRegion++
			if region["startLine"].(float64) < 1 || region["endLine"].(float64) < region["startLine"].(float64) {
				t.Errorf("bad region %v", region)
			}
		}
		if sup, ok := res["suppressions"].([]any); ok {
			suppressed++
			if sup[0].(map[string]any)["kind"] != "external" || !strings.Contains(sup[0].(map[string]any)["justification"].(string), "ADR-OK") {
				t.Errorf("bad suppression %v", sup)
			}
		}
	}
	if suppressed != 1 || withRegion != 3 {
		t.Errorf("suppressed = %d, with region = %d; want 1 and 3", suppressed, withRegion)
	}
	for _, banned := range []string{"/home/", "/tmp/", "timestamp", "startTimeUtc"} {
		if strings.Contains(string(data), banned) {
			t.Errorf("SARIF contains %q", banned)
		}
	}
}

// At a tier where debt is a warning the results are warnings, and the SARIF
// output is byte-identical between runs and fingerprints do not depend on the
// line, so moving a function keeps its fingerprint.
//
// TRLC-LINKS: REQ-EMG-157, REQ-EMG-159
func TestSARIFLevelsAndStableFingerprints(t *testing.T) {
	opts := sampleOptions()
	opts.Tier, opts.Levels = "mvp", Levels(bundleAt("mvp", nil, nil))
	r := BuildReport(sampleResults(), opts)
	_, first := sarifDoc(t, r)
	_, second := sarifDoc(t, r)
	if !bytes.Equal(first, second) {
		t.Fatal("SARIF differs between two renderings")
	}
	if strings.Contains(string(first), `"level": "error"`) {
		t.Error("an error result at tier mvp")
	}

	moved := sampleResults()
	moved[1].Functions[0].StartLine += 100
	for i := range moved[1].Functions[0].Findings {
		moved[1].Functions[0].Findings[i].StartLine += 100
		moved[1].Functions[0].Findings[i].EndLine += 100
	}
	fingerprints := func(r Report) []string {
		doc, _ := sarifDoc(t, r)
		var out []string
		for _, res := range doc["runs"].([]any)[0].(map[string]any)["results"].([]any) {
			out = append(out, res.(map[string]any)["partialFingerprints"].(map[string]any)[FingerprintKey].(string))
		}
		return out
	}
	a, b := fingerprints(r), fingerprints(BuildReport(moved, opts))
	if strings.Join(sortedCopy(a), ",") != strings.Join(sortedCopy(b), ",") {
		t.Errorf("fingerprints changed when the lines moved: %v vs %v", a, b)
	}
	if FindingFingerprint(CodeBumpyRoad, "a.go", "f") == FindingFingerprint(CodeBumpyRoad, "a.go", "g") ||
		FindingFingerprint(CodeBumpyRoad, "a.go", "f") == FindingFingerprint(CodeLargeMethod, "a.go", "f") {
		t.Error("fingerprints of different findings collide")
	}
}

// sortedCopy returns a sorted copy of a string slice.
func sortedCopy(in []string) []string {
	out := append([]string(nil), in...)
	sort.Strings(out)
	return out
}

// The artifact uri is a URI reference: each path segment is percent-encoded,
// the slashes stay, so a Next.js route, a name with a space or a hash and a name
// with a percent sign are valid references (Q3 of the second review pass). The
// fingerprint still hashes the raw path.
// Mutation: writing the raw path leaves "[id].tsx" and "a b" in the uri.
//
// TRLC-LINKS: REQ-EMG-157
func TestSARIFURIsArePercentEncoded(t *testing.T) {
	cases := map[string]string{
		"app/[id].tsx": "app/%5Bid%5D.tsx",
		"a b/c#d.go":   "a%20b/c%23d.go",
		"100%.go":      "100%25.go",
		"plain/x.go":   "plain/x.go",
		"c:d/e.go":     "c%3Ad/e.go",
		"q?/\u00e4.go": "q%3F/%C3%A4.go",
	}
	for raw, want := range cases {
		if got := sarifURI(raw); got != want {
			t.Errorf("sarifURI(%q) = %q, want %q", raw, got, want)
		}
	}
	report := BuildReport([]FileResult{{Path: "app/[id].tsx", Language: "tsx", Functions: []Function{{
		Name: "Page", Findings: []Finding{{Code: CodeComplexMethod, Severity: SeverityHigh, StartLine: 1, EndLine: 2, Metric: "cyclomatic", Value: 30, Message: "m"}},
	}}}}, sampleOptions())
	data, err := SARIF(report)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"uri": "app/%5Bid%5D.tsx"`) || strings.Contains(string(data), "[id]") {
		t.Errorf("SARIF uri not encoded:\n%s", data)
	}
	if !strings.Contains(string(data), FindingFingerprint(CodeComplexMethod, "app/[id].tsx", "Page")) {
		t.Errorf("the fingerprint must hash the raw path:\n%s", data)
	}

	delta := DeltaReport{Blocking: []DeltaEntry{{Path: "a b/c#d.go", Function: "F", Code: CodeComplexMethod, Change: ChangeNew, Severity: SeverityHigh}},
		Skipped: []SkippedFile{{Path: "100%.go", Reason: "too large"}}}
	data, err = CheckSARIF(NewCheckReport(delta, map[string]string{"health.regression": "error", CodeFileSkipped: "error"}, nil, PolicyVerdict{}))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(data), `"uri": "a%20b/c%23d.go"`) || !strings.Contains(string(data), `"uri": "100%25.go"`) {
		t.Errorf("CheckSARIF uri not encoded:\n%s", data)
	}
}
