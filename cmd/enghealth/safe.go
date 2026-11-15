// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

// safeText makes a string safe to print to a log or a terminal: a control
// character (below 0x20, 0x7f, or a C1 control) becomes a visible \x or \u
// escape. A file name or a message that came from the repository can hold a
// newline or an escape sequence, which in a GitHub Actions log would start a
// workflow command (::error::, ::stop-commands::) and on a terminal would move
// the cursor or rewrite the screen. The backslash is left alone.
//
// TRLC-LINKS: REQ-EMG-158
func safeText(s string) string {
	if !strings.ContainsFunc(s, isControl) {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		b.WriteString(escapeControl(r))
	}
	return b.String()
}

// escapeControl is a rune as safeText prints it: itself, or a visible escape
// when it is a control character.
//
// TRLC-LINKS: REQ-EMG-158
func escapeControl(r rune) string {
	switch {
	case !isControl(r):
		return string(r)
	case r < 0x100:
		return fmt.Sprintf("\\x%02x", r)
	}
	return fmt.Sprintf("\\u%04x", r)
}

// isControl reports whether r is a control character in the sense of safeText.
//
// TRLC-LINKS: REQ-EMG-158
func isControl(r rune) bool {
	return isC0OrC1(r) || isLineSeparator(r)
}

// isC0OrC1 reports the C0 and C1 control characters and DEL.
//
// TRLC-LINKS: REQ-EMG-158
func isC0OrC1(r rune) bool { return r < 0x20 || (r >= 0x7f && r <= 0x9f) }

// isLineSeparator reports the Unicode line and paragraph separators.
//
// TRLC-LINKS: REQ-EMG-158
func isLineSeparator(r rune) bool { return r == 0x2028 || r == 0x2029 }

// writeFileAtomic writes data to path through a temporary file in the same
// directory and a rename, so a failure never leaves a half-written file and a
// symbolic link at the target is never followed. An existing target that is not
// a regular file (a symbolic link, a directory, a device, a pipe) is refused.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func writeFileAtomic(path string, data []byte) error {
	if err := checkAtomicTarget(path); err != nil {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".enghealth-*.tmp")
	if err != nil {
		return err
	}
	if err := fillTemp(tmp, data); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}

// fillTemp writes the data to the temporary file, closes it and makes it
// readable.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func fillTemp(tmp *os.File, data []byte) error {
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		return err
	}
	return os.Chmod(tmp.Name(), 0o644)
}

// checkAtomicTarget refuses an existing target that is not a regular file.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func checkAtomicTarget(path string) error {
	info, err := os.Lstat(path)
	if err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to write %s: it exists and is not a regular file", safeText(path))
	}
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	return nil
}
