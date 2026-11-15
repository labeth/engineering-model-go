// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// Formal (Quint) evidence in the trace matrix: pass, fail, stale and none
// (REQ-EMG-146). The fixtures are built in temp directories; no committed
// model is loaded.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"path/filepath"
	"slices"
	"strconv"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

const formalQntFile = "generated/quint/sm_demo.qnt"
const formalQntText = codescan.FormalGeneratedHeader + "SM-DEMO; do not edit.\nmodule sm_demo { val x = 1 }\n"

// formalFixtureEntry is one entry of a fixture evidence file.
type formalFixtureEntry struct{ req, status, invariant string }

// formalFixtureFiles returns the files of a model directory with a module and an
// evidence file; staleSha records a hash that does not match the module.
//
// TRLC-LINKS: REQ-EMG-146
func formalFixtureFiles(entries []formalFixtureEntry, staleSha bool) map[string]string {
	sum := sha256.Sum256([]byte(formalQntText))
	sha := hex.EncodeToString(sum[:])
	if staleSha {
		sha = strings.Repeat("0", 64)
	}
	var results []map[string]string
	for _, e := range entries {
		results = append(results, map[string]string{
			"requirement": e.req, "status": e.status, "method": "analysis", "kind": "formal",
			"machine": "SM-DEMO", "invariant": e.invariant, "mode": "run",
			"qntFile": formalQntFile, "qntSha256": sha, "notes": "fixture",
		})
	}
	data, _ := json.Marshal(map[string]any{"schema": "engmod-quint-results/1", "results": results})
	return map[string]string{formalQntFile: formalQntText, "test-results/quint-formal.json": string(data)}
}

// TRLC-LINKS: REQ-EMG-146
func formalRow(t *testing.T, m Matrix, id string) Row {
	t.Helper()
	row, ok := rowsByID(m)[id]
	if !ok {
		t.Fatalf("no row %s", id)
	}
	return row
}

// A passing current result is analysis evidence: the row is verified, passed,
// lists the method analysis and the formal entry; a failing one fails the
// requirement (not unverified); a stale one is no evidence; a requirement
// without formal evidence has neither new field in its JSON.
//
// TRLC-LINKS: REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestFormalEvidenceRollsUpPassFailAndNone(t *testing.T) {
	files := formalFixtureFiles([]formalFixtureEntry{
		{"REQ-F-001", "pass", "INV-F-001"},
		{"REQ-F-002", "fail", "INV-F-002"},
		{"REQ-F-005", "pass", "INV-F-001"},
		{"REQ-F-005", "fail", "INV-F-002"},
	}, false)
	files["tests/plain_test.go"] = "package p\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-F-004\nfunc TestPlain(t *testing.T) {}\n"
	m, checks, _ := verificationFixture(t, files, "REQ-F-001", "REQ-F-002", "REQ-F-003", "REQ-F-004", "REQ-F-005")

	pass := formalRow(t, m, "REQ-F-001")
	requireEqual(t, rowFacts(pass), []string{"verified", "passed", "1", "analysis"})
	requireEqual(t, formalFacts(pass.Formal), []string{formalPass + "|SM-DEMO|INV-F-001|64"})
	fail := formalRow(t, m, "REQ-F-002") // a failing invariant fails the requirement
	requireEqual(t, rowFacts(fail)[:2], []string{"verified", "failed"})
	requireEqual(t, formalFacts(fail.Formal), []string{formalFailed + "|SM-DEMO|INV-F-002|64"})
	mixed := formalRow(t, m, "REQ-F-005") // one invariant passes and another fails: failed wins
	requireEqual(t, []string{mixed.Result, strconv.Itoa(len(mixed.Formal))}, []string{"failed", "2"})
	none := formalRow(t, m, "REQ-F-003")
	requireEqual(t, []bool{none.Status == "orphan", none.Formal == nil, none.EvidenceMethods == nil}, []bool{true, true, true})
	// A requirement with a plain test only carries neither new field.
	plain := formalRow(t, m, "REQ-F-004")
	data, _ := json.Marshal(plain)
	requireEqual(t, []bool{plain.Result != "", strings.Contains(string(data), "formal"), strings.Contains(string(data), "evidenceMethods")}, []bool{true, false, false})
	requireEqual(t, m.Summary.Passed, 1) // only REQ-F-001; the test of REQ-F-004 has no result file
	requireEqual(t, len(diagsOfMethod(checks, "analysis")), 1)
}

// rowFacts is the status, result, number of verifications and evidence
// methods of a row.
func rowFacts(r Row) []string {
	return []string{r.Status, r.Result, strconv.Itoa(len(r.Verifications)), strings.Join(r.EvidenceMethods, ",")}
}

// formalFacts renders each formal result as result|machine|invariant|sha length.
func formalFacts(results []formalResult) []string {
	var out []string
	for _, f := range results {
		out = append(out, fmt.Sprintf("%s|%s|%s|%d", f.Result, f.Machine, f.Invariant, len(f.QntSha256)))
	}
	return out
}

// diagsOfMethod returns the checks of a method.
func diagsOfMethod(checks []codescan.VerificationCheck, method string) []codescan.VerificationCheck {
	return slices.DeleteFunc(slices.Clone(checks), func(c codescan.VerificationCheck) bool { return c.Method != method })
}

// A stale result is no evidence: the requirement is orphan (nothing else
// verifies it) and the entry shows as stale; another passing test of the
// requirement keeps it passed.
//
// TRLC-LINKS: REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestFormalStaleResultIsNoEvidenceAndDoesNotDowngrade(t *testing.T) {
	files := formalFixtureFiles([]formalFixtureEntry{
		{"REQ-F-001", "pass", "INV-F-001"},
		{"REQ-F-002", "pass", "INV-F-002"},
	}, true)
	files["tests/two_test.go"] = "package p\n\nimport \"testing\"\n\n// TRLC-LINKS: REQ-F-002\nfunc TestTwo(t *testing.T) {}\n"
	files["test-results/two.json"] = `{"results":[{"requirement":"REQ-F-002","status":"pass"}]}`
	m, _, diags := verificationFixture(t, files, "REQ-F-001", "REQ-F-002")

	stale := formalRow(t, m, "REQ-F-001") // a stale-only requirement has no evidence
	requireEqual(t, rowFacts(stale), []string{"orphan", "", "0", ""})
	requireEqual(t, formalFacts(stale.Formal), []string{formalStaleResult + "|SM-DEMO|INV-F-001|64"})
	requireEqual(t, strings.Contains(stale.Formal[0].Notes, "no longer matches"), true)
	other := formalRow(t, m, "REQ-F-002") // stale evidence does not downgrade a passing test
	requireEqual(t, rowFacts(other), []string{"verified", "passed", "1", ""})
	requireEqual(t, formalFacts(other.Formal), []string{formalStaleResult + "|SM-DEMO|INV-F-002|64"})
	requireEqual(t, len(diagsOf(diags, "verification.formal_stale")) > 0, true)
}

// ValidateIntegrity reports verification.failed for a failing formal result
// (naming the machine and invariant) in a model with lifecycle, nothing for a
// passing or stale one, and a stale-only requirement stays orphan.
//
// TRLC-LINKS: REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func TestFormalFailureReportsVerificationFailed(t *testing.T) {
	reqs := formalThreeRequirements()
	bundle, checks := formalLifecycleFixture(t, []formalFixtureEntry{
		{"REQ-F-001", "pass", "INV-F-001"},
		{"REQ-F-002", "fail", "INV-F-002"},
		{"REQ-F-003", "pass", "INV-F-003"},
	}, false)
	failed := diagsOf(ValidateIntegrity(bundle, Evidence{reqs, nil, checks}), "verification.failed")
	requireEqual(t, pathsOf(failed, "verification.failed"), []string{"requirements[1]"})
	requireEqual(t, strings.Contains(failed[0].Message, "invariant INV-F-002 of machine SM-DEMO"), true)
	// The model without lifecycle reports nothing, as for tests.
	bundle.Manifest.Lifecycle = nil
	requireEqual(t, len(diagsOf(ValidateIntegrity(bundle, Evidence{reqs, nil, checks}), "verification.failed")), 0)

	// Stale evidence: no verification.failed and the requirement is an orphan.
	bundle, checks = formalLifecycleFixture(t, []formalFixtureEntry{{"REQ-F-002", "fail", "INV-F-002"}}, true)
	diags := ValidateIntegrity(bundle, Evidence{reqs, nil, checks})
	requireEqual(t, len(diagsOf(diags, "verification.failed")), 0)
	requireEqual(t, countOf(pathsOf(diags, "requirement.orphan"), "requirements[REQ-F-002]"), 1)
}

// formalLifecycleFixture writes the formal evidence files of the entries into
// a model with lifecycle and infers the verification checks of
// formalThreeRequirements.
func formalLifecycleFixture(t *testing.T, entries []formalFixtureEntry, staleSha bool) (model.Bundle, []codescan.VerificationCheck) {
	t.Helper()
	root := t.TempDir()
	for name, content := range formalFixtureFiles(entries, staleSha) {
		testutil.WriteFile(t, filepath.Join(root, filepath.FromSlash(name)), content)
	}
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Manifest: model.ManifestDocument{Lifecycle: &model.ManifestLifecycle{Tier: "product"}}}
	checks, _ := codescan.InferVerificationChecks(bundle, formalThreeRequirements(), nil, root)
	return bundle, checks
}

// formalThreeRequirements is the requirements REQ-F-001 to REQ-F-003.
func formalThreeRequirements() model.RequirementsDocument {
	return model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-F-001"}, {ID: "REQ-F-002"}, {ID: "REQ-F-003"}}}
}

// The reading helper and the roll-up (failed over pass over stale over
// not-run): per-entry results, the requirement result and the evidence files a
// formal check names.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestReadFormalResultsAndRollUp(t *testing.T) {
	root := t.TempDir()
	for name, content := range formalFixtureFiles([]formalFixtureEntry{
		{"REQ-F-001", "pass", "INV-A"}, {"REQ-F-001", "fail", "INV-B"}, {"REQ-F-002", "skipped", "INV-C"},
	}, false) {
		testutil.WriteFile(t, filepath.Join(root, filepath.FromSlash(name)), content)
	}
	by := formalByRequirement(readFormalResults(root, []string{"test-results/quint-formal.json"}))
	requireEqual(t, []string{formalRequirementResult(by["REQ-F-001"]), formalRequirementResult(by["REQ-F-002"]), formalRequirementResult(nil)}, []string{formalFailed, formalNotRun, ""})
	requireEqual(t, formalRequirementResult([]formalResult{{Result: formalStaleResult}, {Result: formalPass}}), formalPass) // pass over stale
	requireEqual(t, readFormalResults(root, []string{"test-results/missing.json"}) == nil, true)                            // a missing file contributes nothing
	plain := codescan.VerificationCheck{ID: "T", Evidence: []string{"test-results/quint-formal.json"}}
	requireEqual(t, formalCheckFiles(plain) == nil, true) // only an analysis check names formal files
	plain.Method = "analysis"
	requireEqual(t, len(formalCheckFiles(plain)), 1)
}

// A crafted evidence file is no evidence: a module outside the model root or a
// file that is not a generated Quint module is stale (the requirement stays
// orphan) whatever hash is recorded, while a module checked out with CRLF line
// endings still matches the hash of its LF form.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE
func TestFormalEvidenceRejectsOutsideAndNonGeneratedModulesAndAcceptsCRLF(t *testing.T) {
	plain := "module plain { val x = 1 }\n"
	entry := func(req, file, text string) map[string]string {
		sum := sha256.Sum256([]byte(text))
		return map[string]string{"requirement": req, "status": "pass", "machine": "SM-DEMO", "invariant": "INV-" + req,
			"qntFile": file, "qntSha256": hex.EncodeToString(sum[:])}
	}
	data, _ := json.Marshal(map[string]any{"results": []map[string]string{
		entry("REQ-F-001", "../../../../dev/zero", ""),
		entry("REQ-F-002", "generated/plain.qnt", plain),
		entry("REQ-F-003", "generated/crlf.qnt", formalQntText),
	}})
	files := map[string]string{
		"generated/plain.qnt":            plain,
		"generated/crlf.qnt":             strings.ReplaceAll(formalQntText, "\n", "\r\n"),
		"test-results/quint-formal.json": string(data),
	}
	m, _, diags := verificationFixture(t, files, "REQ-F-001", "REQ-F-002", "REQ-F-003")
	for id, want := range map[string]string{"REQ-F-001": "not a local path", "REQ-F-002": "not a generated Quint module"} {
		row := formalRow(t, m, id)
		requireEqual(t, []string{row.Status, row.Formal[0].Result}, []string{"orphan", formalStaleResult})
		requireEqual(t, strings.Contains(row.Formal[0].Notes, want), true)
	}
	crlf := formalRow(t, m, "REQ-F-003") // a CRLF checkout stays current
	requireEqual(t, []string{crlf.Status, crlf.Result, crlf.Formal[0].Result}, []string{"verified", "passed", formalPass})
	requireEqual(t, len(diagsOf(diags, "verification.formal_stale")), 2)
}
