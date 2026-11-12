// ENGMODEL-OWNER-UNIT: FU-CODEMAP-INFERENCE
package codescan

import (
	"errors"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/labeth/engineering-model-go/internal/textutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// isFormalEvidencePath reports whether path names a formal evidence file.
//
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-146
func isFormalEvidencePath(path string) bool {
	return strings.EqualFold(filepath.Base(filepath.ToSlash(path)), FormalEvidenceFile)
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

// parseFormalEvidence reads a formal evidence file with the shared reader and
// checks each entry at scan time (see FormalStaleReason): the module named by
// qntFile (relative to the model root) must be a generated Quint module of the
// entry's machine and its sha256 must still match. A stale entry (module
// changed, missing, outside the model root, not generated or no hash recorded)
// is returned as not-run, so it is no evidence and no pass, and yields one
// verification.formal_stale warning per invariant and module. A current entry
// keeps its pass or fail status.
//
// ENGMODEL-LINKS: FU-CODEMAP-INFERENCE, FU-ALLOCATION-TRACE
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func parseFormalEvidence(baseDir, path string) ([]VerificationResult, []validate.Diagnostic) {
	rel, err := filepath.Rel(baseDir, path)
	if err != nil {
		rel = path
	}
	rel = filepath.ToSlash(rel)
	entries, err := readFormalEvidenceFile(newFormalModules(baseDir), path, rel)
	if err != nil {
		message := err.Error()
		if errors.Is(err, errFormalNotJSON) {
			message = "formal evidence is not valid JSON: " + strings.TrimPrefix(err.Error(), errFormalNotJSON.Error()+": ")
		}
		return nil, []validate.Diagnostic{{
			Code: "verification.formal_unreadable", Severity: validate.SeverityWarning,
			Message: message, Path: rel,
		}}
	}
	var results []VerificationResult
	var diags []validate.Diagnostic
	reported := map[string]bool{}
	for _, entry := range entries {
		status := entry.Status
		if entry.StaleReason != "" {
			status = "not-run"
			key := entry.QntFile + "|" + entry.Invariant
			if !reported[key] {
				reported[key] = true
				diags = append(diags, validate.Diagnostic{
					Code: "verification.formal_stale", Severity: validate.SeverityWarning, Path: rel,
					Message: fmt.Sprintf("formal result for invariant %s of %s is stale and counts as no evidence: %s; rerun engquint check",
						textutil.NonEmpty(entry.Invariant, "(unnamed)"), textutil.NonEmpty(entry.Machine, "(unnamed machine)"), entry.StaleReason),
				})
			}
		}
		results = append(results, VerificationResult{Requirement: entry.Requirement, Status: status, Notes: entry.Notes})
	}
	return results, diags
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
