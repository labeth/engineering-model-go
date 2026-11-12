// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// formalEvidenceFile is the name of the evidence file engquint check writes
// into test-results/. The codescan package must not import the exporter, so the
// layout is read here by name: a results array whose entries carry requirement,
// status, invariant, machine, qntFile (relative to the model root) and
// qntSha256.
const formalEvidenceFile = "quint-formal.json"

// formalEntry is one entry of the formal evidence file.
type formalEntry struct {
	Requirement string `json:"requirement"`
	Status      string `json:"status"`
	Machine     string `json:"machine"`
	Invariant   string `json:"invariant"`
	QntFile     string `json:"qntFile"`
	QntSha256   string `json:"qntSha256"`
	Notes       string `json:"notes"`
}

// isFormalEvidencePath reports whether path names a formal evidence file.
//
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-146
func isFormalEvidencePath(path string) bool {
	return strings.EqualFold(filepath.Base(filepath.ToSlash(path)), formalEvidenceFile)
}

// verificationMethodFromPath is the verification method of a result file:
// analysis for formal evidence and empty (the default test method) otherwise.
//
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-146
func verificationMethodFromPath(path string) string {
	if isFormalEvidencePath(path) {
		return "analysis"
	}
	return ""
}

// parseFormalEvidence reads a formal evidence file and checks each entry at
// scan time: the sha256 of the module named by qntFile (relative to the model
// root) must still match. A stale entry (module changed, missing or no hash
// recorded) is returned as not-run, so it is no evidence and no pass, and
// yields one verification.formal_stale warning per invariant and module. A
// current entry keeps its pass or fail status.
//
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func parseFormalEvidence(baseDir, path string) ([]VerificationResult, []validate.Diagnostic) {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, []validate.Diagnostic{{
			Code: "verification.formal_unreadable", Severity: validate.SeverityWarning,
			Message: err.Error(), Path: rel,
		}}
	}
	var file struct {
		Results []formalEntry `json:"results"`
	}
	if err := json.Unmarshal(data, &file); err != nil {
		return nil, []validate.Diagnostic{{
			Code: "verification.formal_unreadable", Severity: validate.SeverityWarning,
			Message: "formal evidence is not valid JSON: " + err.Error(), Path: rel,
		}}
	}
	var results []VerificationResult
	var diags []validate.Diagnostic
	reported := map[string]bool{}
	hashes := map[string]string{}
	for _, entry := range file.Results {
		req := strings.TrimSpace(entry.Requirement)
		if req == "" {
			continue
		}
		status := NormalizeResultStatus(entry.Status)
		notes := strings.TrimSpace(entry.Notes)
		if reason := formalStaleReason(baseDir, entry, hashes); reason != "" {
			status = "not-run"
			notes = "stale: " + reason + "; rerun engquint check"
			key := entry.QntFile + "|" + entry.Invariant
			if !reported[key] {
				reported[key] = true
				diags = append(diags, validate.Diagnostic{
					Code: "verification.formal_stale", Severity: validate.SeverityWarning, Path: rel,
					Message: fmt.Sprintf("formal result for invariant %s of %s is stale and counts as no evidence: %s; rerun engquint check",
						textutil.NonEmpty(entry.Invariant, "(unnamed)"), textutil.NonEmpty(entry.Machine, "(unnamed machine)"), reason),
				})
			}
		}
		results = append(results, VerificationResult{Requirement: req, Status: status, Notes: notes})
	}
	return results, diags
}

// formalStaleReason returns why an entry is stale, or "" when the sha256 of its
// module still matches. hashes caches the hash of each module file.
//
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
// TRLC-LINKS: REQ-EMG-145
func formalStaleReason(baseDir string, entry formalEntry, hashes map[string]string) string {
	file := strings.TrimSpace(entry.QntFile)
	want := strings.ToLower(strings.TrimSpace(entry.QntSha256))
	if file == "" || want == "" {
		return "the entry records no qntFile or qntSha256"
	}
	if filepath.IsAbs(file) {
		return fmt.Sprintf("qntFile %s is not relative to the model root", file)
	}
	got, seen := hashes[file]
	if !seen {
		data, err := os.ReadFile(filepath.Join(baseDir, filepath.FromSlash(file)))
		if err != nil {
			got = "missing"
		} else {
			sum := sha256.Sum256(data)
			got = hex.EncodeToString(sum[:])
		}
		hashes[file] = got
	}
	switch {
	case got == "missing":
		return fmt.Sprintf("module %s is missing", file)
	case got != want:
		return fmt.Sprintf("the sha256 of %s no longer matches the one the result was produced from", file)
	}
	return ""
}

// verificationDescriptionForArtifact describes the check inferred from a result
// file: formal evidence gets its own text, other files keep the description
// found in the artifact.
//
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE
// TRLC-LINKS: REQ-EMG-146
func verificationDescriptionForArtifact(baseDir, relPath string) string {
	if isFormalEvidencePath(relPath) {
		return "Formal verification of machine invariants with Quint (analysis); a result counts only while the sha256 of its module matches."
	}
	return verificationDescriptionFromResultArtifact(baseDir, relPath)
}
