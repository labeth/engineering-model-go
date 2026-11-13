// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"
)

// The committed baseline of accepted code health debt (ADR-EMG-059): a JSON
// file that lists existing findings by path, qualified function name and code
// with the metric value each had when the baseline was written.
// It holds no timestamp and no absolute path and is canonical (sorted, two-space
// indent, trailing newline), so a re-written baseline of unchanged debt is
// byte-identical and a diff of it shows exactly the debt that was paid or
// added. This package owns the format because internal/validate reads the file
// to report health.baseline_unreadable and may not import internal/health.

// HealthBaselineSchema is the schema string of a baseline file.
const HealthBaselineSchema = "engmod-health-baseline/1"

// maxHealthBaselineBytes bounds what is read of a baseline file.
const maxHealthBaselineBytes = 64 << 20

// HealthBaselineEntry is one accepted finding. Path is relative to the code
// root (the form of a waiver path), Function is the qualified function name
// of the engine and Code a health smell code. Value is the metric value of the
// finding when the baseline was written: a finding is baselined only while its
// current value is at most Value, so debt that grows is no longer accepted.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-161
type HealthBaselineEntry struct {
	Path     string `json:"path"`
	Function string `json:"function"`
	Code     string `json:"code"`
	Value    int    `json:"value"`
}

// HealthBaselineKey identifies a baseline entry without its value: the finding
// it accepts.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
type HealthBaselineKey struct{ Path, Function, Code string }

// Key returns the finding the entry names.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func (e HealthBaselineEntry) Key() HealthBaselineKey {
	return HealthBaselineKey{e.Path, e.Function, e.Code}
}

// HealthBaselineFile is the content of a baseline file.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-154, REQ-EMG-161
type HealthBaselineFile struct {
	Schema  string                `json:"schema"`
	Entries []HealthBaselineEntry `json:"entries"`
}

// ErrHealthBaseline marks a baseline file that cannot be used: missing, not
// JSON, not engmod-health-baseline/1 or holding an invalid entry.
var ErrHealthBaseline = errors.New("health baseline")

// HealthBaselinePathProblem returns why a baseline file path of the manifest
// is not acceptable (empty, absolute, leaving the manifest directory or using
// a backslash), or "" when it is a clean relative path.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func HealthBaselinePathProblem(file string) string {
	file = strings.TrimSpace(file)
	switch {
	case file == "":
		return "is empty"
	case strings.Contains(file, "\\"):
		return "contains a backslash; use forward slashes"
	case strings.HasPrefix(file, "/") || filepath.IsAbs(file):
		return "is absolute"
	}
	clean := path.Clean(file)
	if clean == ".." || strings.HasPrefix(clean, "../") {
		return "leaves the manifest directory"
	}
	if clean == "." || strings.HasSuffix(file, "/") {
		return "names a directory, not a file"
	}
	return ""
}

// ResolveHealthBaselinePath returns the path of the baseline file next to the
// manifest. A path that is not a clean relative one is an error.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func ResolveHealthBaselinePath(manifestPath, file string) (string, error) {
	if problem := HealthBaselinePathProblem(file); problem != "" {
		return "", fmt.Errorf("%w: file %q %s", ErrHealthBaseline, file, problem)
	}
	return filepath.Join(filepath.Dir(manifestPath), filepath.FromSlash(path.Clean(strings.TrimSpace(file)))), nil
}

// LoadHealthBaseline reads and parses the baseline file named relative to the
// directory of the manifest.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func LoadHealthBaseline(manifestPath, file string) (HealthBaselineFile, error) {
	full, err := ResolveHealthBaselinePath(manifestPath, file)
	if err != nil {
		return HealthBaselineFile{}, err
	}
	f, err := os.Open(full)
	if err != nil {
		return HealthBaselineFile{}, fmt.Errorf("%w: %s cannot be read: %v", ErrHealthBaseline, file, err)
	}
	defer f.Close()
	buf := new(bytes.Buffer)
	if _, err := buf.ReadFrom(io.LimitReader(f, maxHealthBaselineBytes+1)); err != nil {
		return HealthBaselineFile{}, fmt.Errorf("%w: %s cannot be read: %v", ErrHealthBaseline, file, err)
	}
	if buf.Len() > maxHealthBaselineBytes {
		return HealthBaselineFile{}, fmt.Errorf("%w: %s is larger than %d bytes", ErrHealthBaseline, file, maxHealthBaselineBytes)
	}
	parsed, err := ParseHealthBaseline(buf.Bytes())
	if err != nil {
		return HealthBaselineFile{}, fmt.Errorf("%w (%s)", err, file)
	}
	return parsed, nil
}

// ParseHealthBaseline decodes a baseline file strictly: the schema string must
// be engmod-health-baseline/1, no unknown field is accepted, every entry needs
// a clean relative path, a function, a health smell code and a value of at
// least 1, and a finding (path, function, code) may not be listed twice. The
// entries are returned in canonical order.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func ParseHealthBaseline(data []byte) (HealthBaselineFile, error) {
	var file HealthBaselineFile
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	if err := dec.Decode(&file); err != nil {
		return HealthBaselineFile{}, fmt.Errorf("%w: not a valid baseline document: %v", ErrHealthBaseline, err)
	}
	if dec.More() {
		return HealthBaselineFile{}, fmt.Errorf("%w: trailing content after the baseline document", ErrHealthBaseline)
	}
	if file.Schema != HealthBaselineSchema {
		return HealthBaselineFile{}, fmt.Errorf("%w: schema %q is not %q", ErrHealthBaseline, file.Schema, HealthBaselineSchema)
	}
	seen := map[HealthBaselineKey]bool{}
	for i, e := range file.Entries {
		if problem := healthBaselineEntryProblem(e); problem != "" {
			return HealthBaselineFile{}, fmt.Errorf("%w: entries[%d] %s", ErrHealthBaseline, i, problem)
		}
		if seen[e.Key()] {
			return HealthBaselineFile{}, fmt.Errorf("%w: entries[%d] repeats %s %s in %s", ErrHealthBaseline, i, e.Code, e.Function, e.Path)
		}
		seen[e.Key()] = true
	}
	file.Entries = SortHealthBaselineEntries(file.Entries)
	return file, nil
}

// healthBaselineEntryProblem returns why an entry is invalid, or "".
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func healthBaselineEntryProblem(e HealthBaselineEntry) string {
	switch {
	case e.Path == "" || strings.HasPrefix(e.Path, "/") || strings.Contains(e.Path, "\\") || path.Clean(e.Path) != e.Path || e.Path == ".." || strings.HasPrefix(e.Path, "../"):
		return fmt.Sprintf("path %q is not a clean repository-relative path", e.Path)
	case strings.TrimSpace(e.Function) == "":
		return "has no function"
	case !IsHealthSmellCode(e.Code):
		return fmt.Sprintf("code %q is not a health smell code", e.Code)
	case e.Value < 1:
		return fmt.Sprintf("value %d must be at least 1: the metric value of the finding when the baseline was written", e.Value)
	}
	return ""
}

// SortHealthBaselineEntries returns the entries without duplicates, sorted by
// path, function and code: the canonical order of a baseline file. Of several
// entries for one finding the largest value is kept, since a function name that
// occurs twice in a file is one entry that must cover both.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func SortHealthBaselineEntries(entries []HealthBaselineEntry) []HealthBaselineEntry {
	out := make([]HealthBaselineEntry, 0, len(entries))
	at := make(map[HealthBaselineKey]int, len(entries))
	for _, e := range entries {
		i, seen := at[e.Key()]
		switch {
		case !seen:
			at[e.Key()] = len(out)
			out = append(out, e)
		case e.Value > out[i].Value:
			out[i].Value = e.Value
		}
	}
	sort.Slice(out, func(i, j int) bool {
		a, b := out[i], out[j]
		if a.Path != b.Path {
			return a.Path < b.Path
		}
		if a.Function != b.Function {
			return a.Function < b.Function
		}
		return a.Code < b.Code
	})
	return out
}

// EncodeHealthBaseline renders the canonical baseline file of the entries:
// sorted and de-duplicated, two-space indent, no HTML escaping, a trailing
// newline and no timestamp.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-161
func EncodeHealthBaseline(entries []HealthBaselineEntry) ([]byte, error) {
	file := HealthBaselineFile{Schema: HealthBaselineSchema, Entries: SortHealthBaselineEntries(entries)}
	var buf bytes.Buffer
	enc := json.NewEncoder(&buf)
	enc.SetEscapeHTML(false)
	enc.SetIndent("", "  ")
	if err := enc.Encode(file); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
