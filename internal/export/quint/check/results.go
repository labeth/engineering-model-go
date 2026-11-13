// ENGMODEL-OWNER-UNIT: FU-QUINT-EXPORTER
package check

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/labeth/engineering-model-go/internal/export/quint"
)

const (
	// ResultsSchema names the layout of results.json.
	ResultsSchema = "engmod-quint-results/1"
	// EvidenceSchema names the layout of the formal evidence file.
	EvidenceSchema = "engmod-formal-evidence/1"
)

// sha256Hex is the lower-case hex sha256 of data.
//
// TRLC-LINKS: REQ-EMG-145
func sha256Hex(data []byte) string {
	sum := sha256.Sum256(data)
	return hex.EncodeToString(sum[:])
}

// normalizeEOL is the one text-normalisation rule for generated modules: every
// CRLF becomes LF (a lone CR is kept). A module is compared with a fresh
// generation, and its sha256 is recorded in results.json and in the evidence
// file, on this normalised text, so a checkout that converted line endings
// neither reads as drift nor changes the hash. The shared evidence reader of
// the trace applies exactly this rule before it hashes a module.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func normalizeEOL(data []byte) []byte {
	return bytes.ReplaceAll(data, []byte("\r\n"), []byte("\n"))
}

// IsGeneratedModule reports whether the text of a .qnt file starts with the
// header engquint writes, which is what marks a file engquint may report as
// extra and remove. A hand-written module never has it.
//
// TRLC-LINKS: REQ-EMG-149
func IsGeneratedModule(data []byte) bool {
	return bytes.HasPrefix(data, []byte(quint.GeneratedHeader))
}

// WriteFileIfChanged writes data to path and reports whether it did. It never
// follows a symlink at the target: an existing target that is not a regular
// file is refused with an error, an unchanged file is not rewritten, and a
// change is written to a temporary file in the same directory and renamed over
// the target, so a reader never sees a partial file and a symlink planted at
// the path is replaced, not written through. A new file gets mode 0644, an
// existing one keeps its permission bits.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-149
func WriteFileIfChanged(path string, data []byte) (bool, error) {
	perm := os.FileMode(0o644)
	info, err := os.Lstat(path)
	switch {
	case err == nil:
		if !info.Mode().IsRegular() {
			return false, fmt.Errorf("refusing to write %s: it exists and is a %s, not a regular file; remove it first", path, fileKind(info.Mode()))
		}
		perm = info.Mode().Perm()
		if existing, readErr := os.ReadFile(path); readErr == nil && bytes.Equal(existing, data) {
			return false, nil
		}
	case !errors.Is(err, fs.ErrNotExist):
		return false, err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), "."+filepath.Base(path)+".tmp-*")
	if err != nil {
		return false, err
	}
	tmpName := tmp.Name()
	_, err = tmp.Write(data)
	if err == nil {
		err = tmp.Chmod(perm)
	}
	if closeErr := tmp.Close(); err == nil {
		err = closeErr
	}
	if err == nil {
		err = os.Rename(tmpName, path)
	}
	if err != nil {
		os.Remove(tmpName)
		return false, err
	}
	return true, nil
}

// fileKind names the kind of a non-regular file for an error message.
//
// TRLC-LINKS: REQ-EMG-149
func fileKind(mode fs.FileMode) string {
	switch {
	case mode&fs.ModeSymlink != 0:
		return "symlink"
	case mode.IsDir():
		return "directory"
	}
	return "special file"
}

// compareWithCommitted compares a fresh generation with the modules committed
// in dir. A generated module that is missing or differs is drift, and so is a
// file with the generated header that no machine produces any more (hand
// written .qnt files are ignored). It returns the drift list and, per module
// file name, the drift kind of a changed or missing module.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-149
// ENGMODEL-LINKS: STATE-ARTIFACTS-FRESH, STATE-ARTIFACTS-STALE
func compareWithCommitted(dir string, files []quint.File) ([]Drift, map[string]string, error) {
	var drifts []Drift
	perModule := map[string]string{}
	produced := map[string]bool{}
	for _, file := range files {
		produced[file.Name] = true
		path := filepath.Join(dir, file.Name)
		info, statErr := os.Lstat(path)
		if statErr == nil && !info.Mode().IsRegular() {
			return nil, nil, fmt.Errorf("%s is a %s, not a regular file", path, fileKind(info.Mode()))
		}
		committed, err := os.ReadFile(path)
		switch {
		case errors.Is(err, fs.ErrNotExist):
			drifts = append(drifts, Drift{File: file.Name, Kind: "missing"})
			perModule[file.Name] = "missing"
		case err != nil:
			return nil, nil, err
		case !bytes.Equal(normalizeEOL(committed), []byte(file.Text)):
			drifts = append(drifts, Drift{File: file.Name, Kind: "changed"})
			perModule[file.Name] = "changed"
		}
	}
	entries, err := os.ReadDir(dir)
	if err != nil && !errors.Is(err, fs.ErrNotExist) {
		return nil, nil, err
	}
	for _, entry := range entries {
		name := entry.Name()
		if !entry.Type().IsRegular() || filepath.Ext(name) != ".qnt" || produced[name] {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			return nil, nil, err
		}
		if IsGeneratedModule(data) {
			drifts = append(drifts, Drift{File: name, Kind: "extra"})
		}
	}
	return drifts, perModule, nil
}

// resultsFile is the layout of results.json.
type resultsFile struct {
	Schema  string         `json:"schema"`
	Modules []ModuleResult `json:"modules"`
}

// writeResults writes results.json: struct-ordered keys, authored module
// order, no timestamps and no durations, so a rerun on the same inputs is
// byte-identical.
//
// TRLC-LINKS: REQ-EMG-145
func writeResults(path string, modules []ModuleResult) error {
	return writeJSON(path, resultsFile{Schema: ResultsSchema, Modules: modules})
}

// writeJSON writes v as indented JSON with a trailing newline, creating the
// directory, through WriteFileIfChanged (no symlink is followed).
//
// TRLC-LINKS: REQ-EMG-145
func writeJSON(path string, v any) error {
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(v); err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	_, err := WriteFileIfChanged(path, buf.Bytes())
	return err
}

// EvidenceEntry is one requirement-keyed entry of the formal evidence file.
// The trace pipeline reads the requirement and status keys; the sha256 of the
// named module lets it detect a stale result at scan time.
type EvidenceEntry struct {
	Requirement string `json:"requirement"`
	// Status is pass or fail.
	Status string `json:"status"`
	// Method is always analysis.
	Method string `json:"method"`
	// Kind is always formal.
	Kind      string `json:"kind"`
	Machine   string `json:"machine"`
	Invariant string `json:"invariant"`
	// Mode is run, verify or run+verify.
	Mode string `json:"mode"`
	// QntFile is the module path relative to the model directory.
	QntFile   string `json:"qntFile"`
	QntSha256 string `json:"qntSha256"`
	Notes     string `json:"notes"`
}

// evidenceFile is the layout of the formal evidence file.
type evidenceFile struct {
	Schema       string          `json:"schema"`
	Tool         string          `json:"tool"`
	QuintVersion string          `json:"quintVersion"`
	Backend      string          `json:"backend"`
	Results      []EvidenceEntry `json:"results"`
}

// evidenceEntries builds the evidence of the modules that have no drift and no
// error: a pass entry for each requirement of a passing invariant and a fail
// entry for each requirement of a violated one. An invariant without a
// requirement evidences nothing. The entries are sorted.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func evidenceEntries(modules []ModuleResult) []EvidenceEntry {
	entries := []EvidenceEntry{}
	for _, m := range modules {
		if m.Status != StatusPass && m.Status != StatusViolated {
			continue
		}
		for _, inv := range m.Invariants {
			if inv.Status != StatusPass && inv.Status != StatusViolated {
				continue
			}
			status := "pass"
			if inv.Status == StatusViolated {
				status = "fail"
			}
			for _, req := range inv.Requirements {
				entries = append(entries, EvidenceEntry{
					Requirement: req, Status: status, Method: "analysis", Kind: "formal",
					Machine: m.Machine, Invariant: inv.ID, Mode: evidenceMode(m.Modes),
					QntFile: m.QntFile, QntSha256: m.QntSha256, Notes: evidenceNotes(m, inv),
				})
			}
		}
	}
	sort.SliceStable(entries, func(i, j int) bool {
		a, b := entries[i], entries[j]
		if a.Requirement != b.Requirement {
			return a.Requirement < b.Requirement
		}
		if a.Machine != b.Machine {
			return a.Machine < b.Machine
		}
		return a.Invariant < b.Invariant
	})
	return entries
}

// evidenceMode joins the modes with a plus sign.
//
// TRLC-LINKS: REQ-EMG-146
func evidenceMode(modes []string) string { return strings.Join(modes, "+") }

// evidenceNotes describes how an entry was obtained, without timings.
//
// TRLC-LINKS: REQ-EMG-146
func evidenceNotes(m ModuleResult, inv InvariantResult) string {
	if inv.Status == StatusViolated {
		for _, step := range []*StepResult{inv.Run, inv.Verify} {
			if step != nil && step.Status == StatusViolated {
				return fmt.Sprintf("%s of %s violated: %s", inv.ID, m.Machine, FormatCounterexample(*step))
			}
		}
	}
	var parts []string
	if inv.Run != nil {
		parts = append(parts, fmt.Sprintf("quint run, %s backend, max-steps %d, max-samples %d, seed %s", m.Backend, m.Bounds.MaxSteps, m.Bounds.MaxSamples, m.Bounds.Seed))
	}
	if inv.Verify != nil {
		parts = append(parts, fmt.Sprintf("quint verify, apalache %s, max-steps %d", ApalacheVersion, m.Bounds.VerifySteps))
	}
	return fmt.Sprintf("%s of %s holds: %s; quint %s", inv.ID, m.Machine, strings.Join(parts, "; "), m.QuintVersion)
}

// writeEvidence writes the formal evidence file the trace reads from
// test-results/. Check rewrites it whenever it runs a module, and clears it
// (clearEvidence) when the model stops generating, so an entry of a module
// that has since drifted, failed or disappeared does not linger.
//
// TRLC-LINKS: REQ-EMG-146
func writeEvidence(path, quintVersion string, entries []EvidenceEntry) error {
	if entries == nil {
		entries = []EvidenceEntry{}
	}
	return writeJSON(path, evidenceFile{
		Schema: EvidenceSchema, Tool: "quint", QuintVersion: quintVersion, Backend: Backend, Results: entries,
	})
}

// clearEvidence rewrites an existing evidence file without entries and reports
// whether it changed the file. A missing file stays missing (a model that never
// had formal evidence gets none), and a file without entries is left as is. An
// unreadable evidence file is replaced too: the trace treats a file without
// entries like no evidence.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-146
func clearEvidence(path string) (bool, error) {
	data, err := os.ReadFile(path)
	if errors.Is(err, fs.ErrNotExist) {
		return false, nil
	}
	if err != nil {
		return false, err
	}
	var previous evidenceFile
	if json.Unmarshal(data, &previous) == nil && len(previous.Results) == 0 {
		return false, nil
	}
	if err := writeEvidence(path, previous.QuintVersion, nil); err != nil {
		return false, err
	}
	return true, nil
}
