// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

const formalQnt = "module sm_x { val inv_x = true }\n"

// formalFixture writes a module and an evidence file under a temporary model
// root and returns the root. entries is the JSON of the results array; %SHA%
// is replaced by the sha256 of the module.
func formalFixture(t *testing.T, entries string) string {
	t.Helper()
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "generated", "quint", "sm_x.qnt"), formalQnt)
	sum := sha256.Sum256([]byte(formalQnt))
	entries = strings.ReplaceAll(entries, "%SHA%", hex.EncodeToString(sum[:]))
	mustWrite(t, filepath.Join(root, "test-results", "quint-formal.json"),
		`{"schema":"engmod-formal-evidence/1","tool":"quint","results":[`+entries+`]}`)
	return root
}

func formalEntry1(req, status string) string {
	return fmt.Sprintf(`{"requirement":%q,"status":%q,"method":"analysis","kind":"formal","machine":"SM-X","invariant":"INV-X","mode":"run","qntFile":"generated/quint/sm_x.qnt","qntSha256":"%%SHA%%","notes":"n"}`, req, status)
}

func inferFormal(t *testing.T, root string) ([]VerificationCheck, []validate.Diagnostic) {
	t.Helper()
	bundle := model.Bundle{ArchitecturePath: filepath.Join(root, "engmod.yml")}
	requirements := model.RequirementsDocument{Requirements: []model.Requirement{{ID: "REQ-X-1"}, {ID: "REQ-X-2"}}}
	return InferVerificationChecks(bundle, requirements, nil, "")
}

func formalCheck(t *testing.T, checks []VerificationCheck) VerificationCheck {
	t.Helper()
	for _, c := range checks {
		if c.Kind == "formal" {
			return c
		}
	}
	t.Fatalf("no formal check in %+v", checks)
	return VerificationCheck{}
}

func resultFor(c VerificationCheck, req string) (VerificationResult, bool) {
	for _, r := range c.Results {
		if r.Requirement == req {
			return r, true
		}
	}
	return VerificationResult{}, false
}

func hasDiag(diags []validate.Diagnostic, code string) bool {
	for _, d := range diags {
		if d.Code == code {
			return true
		}
	}
	return false
}

// TRLC-LINKS: REQ-EMG-146
func TestFormalEvidenceWithMatchingShaIsAnalysisEvidence(t *testing.T) {
	root := formalFixture(t, formalEntry1("REQ-X-1", "pass")+","+formalEntry1("REQ-X-2", "pass"))
	checks, diags := inferFormal(t, root)
	if hasDiag(diags, "verification.formal_stale") {
		t.Fatalf("unexpected stale diagnostic: %+v", diags)
	}
	c := formalCheck(t, checks)
	if c.Method != "analysis" || c.Status != "pass" || len(c.Verifies) != 2 {
		t.Fatalf("check %+v", c)
	}
	for _, req := range []string{"REQ-X-1", "REQ-X-2"} {
		r, ok := resultFor(c, req)
		if !ok || r.Status != "pass" || r.Evidence != "test-results/quint-formal.json" {
			t.Fatalf("%s: %+v %v", req, r, ok)
		}
	}
	if !strings.HasPrefix(c.ID, "VER-INF-FORMAL-") {
		t.Errorf("id %q", c.ID)
	}
}

// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func TestFormalEvidenceWithChangedModuleIsStaleAndNotEvidence(t *testing.T) {
	root := formalFixture(t, formalEntry1("REQ-X-1", "pass"))
	mustWrite(t, filepath.Join(root, "generated", "quint", "sm_x.qnt"), formalQnt+"// edited\n")
	checks, diags := inferFormal(t, root)
	if !hasDiag(diags, "verification.formal_stale") {
		t.Fatalf("no stale diagnostic: %+v", diags)
	}
	for _, d := range diags {
		if d.Code == "verification.formal_stale" && (d.Severity != validate.SeverityWarning || !strings.Contains(d.Message, "INV-X") || !strings.Contains(d.Message, "SM-X")) {
			t.Errorf("diagnostic %+v", d)
		}
	}
	c := formalCheck(t, checks)
	r, ok := resultFor(c, "REQ-X-1")
	if !ok || r.Status == "pass" || r.Status != "not-run" || !strings.Contains(r.Notes, "stale") || c.Status == "pass" {
		t.Fatalf("a stale result must not pass: %+v / check %q", r, c.Status)
	}
}

// TRLC-LINKS: REQ-EMG-145
func TestFormalEvidenceWithMissingModuleOrHashIsStale(t *testing.T) {
	root := formalFixture(t, formalEntry1("REQ-X-1", "pass"))
	if err := os.Remove(filepath.Join(root, "generated", "quint", "sm_x.qnt")); err != nil {
		t.Fatal(err)
	}
	checks, diags := inferFormal(t, root)
	if r, _ := resultFor(formalCheck(t, checks), "REQ-X-1"); r.Status != "not-run" || !hasDiag(diags, "verification.formal_stale") {
		t.Fatalf("missing module: %+v %+v", r, diags)
	}

	noHash := formalFixture(t, `{"requirement":"REQ-X-1","status":"pass","machine":"SM-X","invariant":"INV-X"}`)
	checks, diags = inferFormal(t, noHash)
	if r, _ := resultFor(formalCheck(t, checks), "REQ-X-1"); r.Status != "not-run" || !hasDiag(diags, "verification.formal_stale") {
		t.Fatalf("no hash: %+v %+v", r, diags)
	}
}

// TRLC-LINKS: REQ-EMG-146
func TestFormalEvidenceFailIsFailAndNotMergedIntoTestChecks(t *testing.T) {
	root := formalFixture(t, formalEntry1("REQ-X-1", "fail")+","+formalEntry1("REQ-X-2", "pass"))
	mustWrite(t, filepath.Join(root, "tests", "x_test.go"), "package x\n\n// TRLC-LINKS: REQ-X-2\nfunc TestX(t *testing.T) {}\n")
	checks, _ := inferFormal(t, root)
	c := formalCheck(t, checks)
	if r, _ := resultFor(c, "REQ-X-1"); r.Status != "fail" || c.Status != "fail" {
		t.Fatalf("fail entry: %+v check %q", r, c.Status)
	}
	for _, other := range checks {
		if other.Kind != "formal" && len(other.Evidence) > 0 && strings.Contains(strings.Join(other.Evidence, ","), "quint-formal") {
			t.Fatalf("formal evidence leaked into test check %+v", other)
		}
	}
	if len(checks) != 2 {
		t.Fatalf("want one test check and one formal check, got %d", len(checks))
	}
}

// TRLC-LINKS: REQ-EMG-146
func TestFormalEvidenceUnreadableJSONWarns(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "test-results", "quint-formal.json"), "{not json")
	_, diags := inferFormal(t, root)
	if !hasDiag(diags, "verification.formal_unreadable") {
		t.Fatalf("diagnostics %+v", diags)
	}
}

// Other JSON result files keep the kind and method they always had.
//
// TRLC-LINKS: REQ-EMG-010, REQ-EMG-146
func TestOtherResultFilesAreNotFormal(t *testing.T) {
	root := t.TempDir()
	mustWrite(t, filepath.Join(root, "test-results", "other.json"), `{"results":[{"requirement":"REQ-X-1","status":"pass"}]}`)
	checks, _ := inferFormal(t, root)
	if len(checks) != 1 || checks[0].Kind != "test" || checks[0].Method != "" {
		t.Fatalf("checks %+v", checks)
	}
}
