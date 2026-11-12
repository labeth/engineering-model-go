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
	clean := true
	for _, r := range s {
		if isControl(r) {
			clean = false
			break
		}
	}
	if clean {
		return s
	}
	var b strings.Builder
	for _, r := range s {
		switch {
		case !isControl(r):
			b.WriteRune(r)
		case r < 0x100:
			fmt.Fprintf(&b, "\\x%02x", r)
		default:
			fmt.Fprintf(&b, "\\u%04x", r)
		}
	}
	return b.String()
}

// isControl reports whether r is a control character in the sense of safeText.
//
// TRLC-LINKS: REQ-EMG-158
func isControl(r rune) bool {
	return r < 0x20 || (r >= 0x7f && r <= 0x9f) || r == 0x2028 || r == 0x2029
}

// writeFileAtomic writes data to path through a temporary file in the same
// directory and a rename, so a failure never leaves a half-written file and a
// symbolic link at the target is never followed. An existing target that is not
// a regular file (a symbolic link, a directory, a device, a pipe) is refused.
//
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-158
func writeFileAtomic(path string, data []byte) error {
	if info, err := os.Lstat(path); err == nil && !info.Mode().IsRegular() {
		return fmt.Errorf("refusing to write %s: it exists and is not a regular file", safeText(path))
	} else if err != nil && !errors.Is(err, os.ErrNotExist) {
		return err
	}
	tmp, err := os.CreateTemp(filepath.Dir(path), ".enghealth-*.tmp")
	if err != nil {
		return err
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if err := errors.Join(werr, cerr); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Chmod(tmp.Name(), 0o644); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	if err := os.Rename(tmp.Name(), path); err != nil {
		_ = os.Remove(tmp.Name())
		return err
	}
	return nil
}
