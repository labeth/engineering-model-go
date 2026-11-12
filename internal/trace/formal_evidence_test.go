// ENGMODEL-OWNER-UNIT: FU-ALLOCATION-TRACE
package trace

// Formal (Quint) evidence in the trace matrix: pass, fail, stale and none
// (REQ-EMG-146). The fixtures are built in temp directories; no committed
// model is loaded.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

const formalQntFile = "generated/quint/sm_demo.qnt"
const formalQntText = "module sm_demo { val x = 1 }\n"

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
	if pass.Status != "verified" || pass.Result != "passed" || len(pass.Verifications) != 1 || strings.Join(pass.EvidenceMethods, ",") != "analysis" {
		t.Fatalf("pass row: %+v", pass)
	}
	if len(pass.Formal) != 1 || pass.Formal[0].Result != formalPass || pass.Formal[0].Machine != "SM-DEMO" || pass.Formal[0].Invariant != "INV-F-001" || len(pass.Formal[0].QntSha256) != 64 {
		t.Fatalf("pass formal entry: %+v", pass.Formal)
	}

	fail := formalRow(t, m, "REQ-F-002")
	if fail.Result != "failed" || fail.Status != "verified" || len(fail.Formal) != 1 || fail.Formal[0].Result != formalFailed {
		t.Fatalf("a failing invariant must fail the requirement: %+v", fail)
	}
	// One invariant passes and another fails: failed wins.
	if mixed := formalRow(t, m, "REQ-F-005"); mixed.Result != "failed" || len(mixed.Formal) != 2 {
		t.Fatalf("pass and fail: %+v", mixed)
	}

	none := formalRow(t, m, "REQ-F-003")
	if none.Status != "orphan" || none.Formal != nil || none.EvidenceMethods != nil {
		t.Fatalf("no evidence: %+v", none)
	}
	// A requirement with a plain test only carries neither new field.
	plain := formalRow(t, m, "REQ-F-004")
	data, _ := json.Marshal(plain)
	if plain.Result == "" || strings.Contains(string(data), "formal") || strings.Contains(string(data), "evidenceMethods") {
		t.Fatalf("test-only row must not gain formal fields: %s", data)
	}
	if m.Summary.Passed != 1 { // only REQ-F-001; the test of REQ-F-004 has no result file
		t.Fatalf("summary %+v", m.Summary)
	}
	formal := 0
	for _, c := range checks {
		if c.Method == "analysis" {
			formal++
		}
	}
	if formal != 1 {
		t.Fatalf("want one formal check, got %d in %+v", formal, checks)
	}
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

	stale := formalRow(t, m, "REQ-F-001")
	if stale.Status != "orphan" || len(stale.Verifications) != 0 || stale.Result != "" || len(stale.EvidenceMethods) != 0 {
		t.Fatalf("a stale-only requirement has no evidence: %+v", stale)
	}
	if len(stale.Formal) != 1 || stale.Formal[0].Result != formalStaleResult || !strings.Contains(stale.Formal[0].Notes, "no longer matches") {
		t.Fatalf("stale formal entry: %+v", stale.Formal)
	}
	other := formalRow(t, m, "REQ-F-002")
	if other.Result != "passed" || other.Status != "verified" || len(other.Verifications) != 1 {
		t.Fatalf("stale evidence must not downgrade a passing test: %+v", other)
	}
	if len(other.Formal) != 1 || other.Formal[0].Result != formalStaleResult {
		t.Fatalf("stale entry stays listed: %+v", other.Formal)
	}
	warned := false
	for _, d := range diags {
		warned = warned || d.Code == "verification.formal_stale"
	}
	if !warned {
		t.Fatalf("missing verification.formal_stale warning: %+v", diags)
	}
}

// ValidateIntegrity reports verification.failed for a failing formal result
// (naming the machine and invariant) in a model with lifecycle, nothing for a
// passing or stale one, and a stale-only requirement stays orphan.
//
// TRLC-LINKS: REQ-EMG-146
// ENGMODEL-LINKS: FU-ALLOCATION-TRACE, FU-VALIDATION-ENGINE
func TestFormalFailureReportsVerificationFailed(t *testing.T) {
	files := formalFixtureFiles([]formalFixtureEntry{
		{"REQ-F-001", "pass", "INV-F-001"},
		{"REQ-F-002", "fail", "INV-F-002"},
		{"REQ-F-003", "pass", "INV-F-003"},
	}, false)
	root := t.TempDir()
	for name, content := range files {
		testutil.WriteFile(t, filepath.Join(root, filepath.FromSlash(name)), content)
	}
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Manifest: model.ManifestDocument{Lifecycle: &model.ManifestLifecycle{Tier: "product"}}}
	reqs := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-F-001"}, {ID: "REQ-F-002"}, {ID: "REQ-F-003"}}}
	checks, _ := codescan.InferVerificationChecks(bundle, reqs, nil, root)

	failed := map[string]string{}
	for _, d := range ValidateIntegrity(bundle, reqs, nil, checks, nil) {
		if d.Code == "verification.failed" {
			failed[d.Path] = d.Message
		}
	}
	if len(failed) != 1 || !strings.Contains(failed["requirements[1]"], "invariant INV-F-002 of machine SM-DEMO") {
		t.Fatalf("verification.failed = %v", failed)
	}
	// The model without lifecycle reports nothing, as for tests.
	bundle.Manifest.Lifecycle = nil
	for _, d := range ValidateIntegrity(bundle, reqs, nil, checks, nil) {
		if d.Code == "verification.failed" {
			t.Fatalf("without lifecycle: %+v", d)
		}
	}

	// Stale evidence: no verification.failed and the requirements are orphans.
	stale := formalFixtureFiles([]formalFixtureEntry{{"REQ-F-002", "fail", "INV-F-002"}}, true)
	root = t.TempDir()
	for name, content := range stale {
		testutil.WriteFile(t, filepath.Join(root, filepath.FromSlash(name)), content)
	}
	bundle = model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml"), Manifest: model.ManifestDocument{Lifecycle: &model.ManifestLifecycle{Tier: "product"}}}
	checks, _ = codescan.InferVerificationChecks(bundle, reqs, nil, root)
	orphans := 0
	for _, d := range ValidateIntegrity(bundle, reqs, nil, checks, nil) {
		if d.Code == "verification.failed" {
			t.Fatalf("a stale failure is no evidence: %+v", d)
		}
		if d.Code == "requirement.orphan" && strings.Contains(d.Message, "REQ-F-002") {
			orphans++
		}
	}
	if orphans != 1 {
		t.Fatalf("stale-only requirement must be orphan, got %d", orphans)
	}
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
	if got := formalRequirementResult(by["REQ-F-001"]); got != formalFailed {
		t.Fatalf("REQ-F-001 = %q", got)
	}
	if got := formalRequirementResult(by["REQ-F-002"]); got != formalNotRun {
		t.Fatalf("REQ-F-002 = %q", got)
	}
	if got := formalRequirementResult(nil); got != "" {
		t.Fatalf("none = %q", got)
	}
	if got := formalRequirementResult([]formalResult{{Result: formalStaleResult}, {Result: formalPass}}); got != formalPass {
		t.Fatalf("pass over stale: %q", got)
	}
	if got := readFormalResults(root, []string{"test-results/missing.json"}); got != nil {
		t.Fatalf("a missing file contributes nothing: %+v", got)
	}
	plain := codescan.VerificationCheck{ID: "T", Evidence: []string{"test-results/quint-formal.json"}}
	if files := formalCheckFiles(plain); files != nil {
		t.Fatalf("only an analysis check names formal files: %v", files)
	}
	plain.Method = "analysis"
	if files := formalCheckFiles(plain); len(files) != 1 {
		t.Fatalf("analysis check files: %v", files)
	}
}
