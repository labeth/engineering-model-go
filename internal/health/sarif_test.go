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
	wantEqual(t, doc["version"], "2.1.0")
	wantContains(t, doc["$schema"].(string), "sarif")
	runs := list(doc["runs"])
	mustEqual(t, len(runs), 1)
	run := obj(runs[0])
	driver := obj(obj(run["tool"])["driver"])
	wantEqual(t, driver["name"], "enghealth")
	ruleIDs := requireRules(t, list(driver["rules"]))

	// Findings in the sample after the waiver: complex_method, the test's
	// large_assertion_block; plus the waived excess_arguments and the unused-waiver diagnostic.
	results := list(run["results"])
	mustEqual(t, len(results), 4)
	var counts [2]int // suppressed results and results with a region
	for _, r := range results {
		c := requireResult(t, obj(r), ruleIDs)
		counts[0] += c[0]
		counts[1] += c[1]
	}
	wantEqual(t, counts, [2]int{1, 3})
	for _, banned := range []string{"/home/", "/tmp/", "timestamp", "startTimeUtc"} {
		wantNone(t, string(data), banned)
	}
}

// obj reads a decoded JSON object.
func obj(v any) map[string]any { return v.(map[string]any) }

// list reads a decoded JSON array.
func list(v any) []any { return v.([]any) }

// requireRules checks that every health code and the unused waiver is a
// complete rule, and returns the index of each rule by its id.
func requireRules(t *testing.T, rules []any) map[string]int {
	t.Helper()
	wantEqual(t, len(rules), len(Codes())+1) // the codes and the unused waiver
	ids := map[string]int{}
	for i, r := range rules {
		rule := obj(r)
		id := rule["id"].(string)
		ids[id] = i
		wantEqual(t, [3]any{id, obj(rule["shortDescription"])["text"] != "", obj(rule["defaultConfiguration"])["level"]}, [3]any{id, true, "error"})
	}
	for _, code := range Codes() {
		_, ok := ids[code]
		wantEqual(t, [2]any{code, ok}, [2]any{code, true})
	}
	return ids
}

// requireResult checks one result: the rule index, the level, the message, the
// fingerprint and the relative uri of its location; it returns whether it is
// suppressed and whether it has a region, as 0 or 1.
func requireResult(t *testing.T, res map[string]any, ruleIDs map[string]int) [2]int {
	t.Helper()
	id := res["ruleId"].(string)
	wantEqual(t, [4]any{id, int(res["ruleIndex"].(float64)), res["level"], obj(res["message"])["text"] != ""}, [4]any{id, ruleIDs[id], "error", true})
	wantEqual(t, obj(res["partialFingerprints"])[FingerprintKey] != "", true)
	phys := obj(obj(list(res["locations"])[0])["physicalLocation"])
	uri := obj(phys["artifactLocation"])["uri"].(string)
	wantEqual(t, [2]any{uri, strings.HasPrefix(uri, "/") || strings.Contains(uri, "..") || strings.Contains(uri, ":")}, [2]any{uri, false})
	return [2]int{requireSuppression(t, res), requireRegion(t, phys)}
}

// requireRegion checks the region of a location, when it has one, and returns
// 1 for a region and 0 for none.
func requireRegion(t *testing.T, phys map[string]any) int {
	t.Helper()
	region, ok := phys["region"].(map[string]any)
	if !ok {
		return 0
	}
	start, end := region["startLine"].(float64), region["endLine"].(float64)
	wantEqual(t, [2]bool{start >= 1, end >= start}, [2]bool{true, true})
	return 1
}

// requireSuppression checks the suppression of a result, when it has one, and
// returns 1 for a suppression and 0 for none.
func requireSuppression(t *testing.T, res map[string]any) int {
	t.Helper()
	sup, ok := res["suppressions"].([]any)
	if !ok {
		return 0
	}
	first := obj(sup[0])
	wantEqual(t, first["kind"], "external")
	wantContains(t, first["justification"].(string), "ADR-OK")
	return 1
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
		wantEqual(t, sarifURI(raw), want)
	}
	report := BuildReport([]FileResult{{Path: "app/[id].tsx", Language: "tsx", Functions: []Function{{
		Name: "Page", Findings: []Finding{{Code: CodeComplexMethod, Severity: SeverityHigh, StartLine: 1, EndLine: 2, Metric: "cyclomatic", Value: 30, Message: "m"}},
	}}}}, sampleOptions())
	data, err := SARIF(report)
	mustEqual(t, err, nil)
	wantContains(t, string(data), `"uri": "app/%5Bid%5D.tsx"`)
	wantNone(t, string(data), "[id]")
	wantContains(t, string(data), FindingFingerprint(CodeComplexMethod, "app/[id].tsx", "Page")) // the fingerprint hashes the raw path
}

// The uri of a check result and of a skipped file is percent-encoded too.
//
// TRLC-LINKS: REQ-EMG-157
func TestCheckSARIFURIsArePercentEncoded(t *testing.T) {
	delta := DeltaReport{Blocking: []DeltaEntry{{Path: "a b/c#d.go", Function: "F", Code: CodeComplexMethod, Change: ChangeNew, Severity: SeverityHigh}},
		Skipped: []SkippedFile{{Path: "100%.go", Reason: "too large"}}}
	data, err := CheckSARIF(NewCheckReport(delta, CheckInput{Levels: map[string]string{"health.regression": "error", CodeFileSkipped: "error"}}))
	mustEqual(t, err, nil)
	wantAll(t, string(data), `"uri": "a%20b/c%23d.go"`, `"uri": "100%25.go"`)
}
