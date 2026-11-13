// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package scan

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/health"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// A code scan of the model that fails leaves the unit and requirement roll-ups
// empty; the report says so with health.rollup_unavailable at the level of
// health.file_failed (an error from product), and names no absolute path (Q4 of
// the second review pass).
// Mutation: dropping the scan diagnostics leaves the report silent.
//
// TRLC-LINKS: REQ-EMG-152, REQ-EMG-155
func TestScanReportsUnavailableRollups(t *testing.T) {
	for tier, want := range map[string]string{"poc": health.LevelWarning, "product": health.LevelError} {
		dir, manifest := fixture(t, "lifecycle:\n  tier: "+tier+"\n")
		old := inferCodeItems
		inferCodeItems = func(model.Bundle, string) ([]codescan.CodeItem, []validate.Diagnostic) {
			return nil, []validate.Diagnostic{{Code: "code.scan_failed", Severity: validate.SeverityWarning, Message: "open " + dir + "/pkg: denied", Path: dir}}
		}
		r, err := Scan(Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, NoCache: true})
		inferCodeItems = old
		if err != nil {
			t.Fatal(err)
		}
		var found []health.Diagnostic
		for _, d := range r.Diagnostics {
			if d.Code == health.CodeRollupUnavailable {
				found = append(found, d)
			}
		}
		if len(found) != 1 || found[0].Level != want || strings.Contains(found[0].Message, dir) || !strings.Contains(found[0].Message, "code.scan_failed") {
			t.Errorf("%s: diagnostics %+v, want one %s without the absolute path", tier, found, want)
		}
		if r.LevelOf(health.CodeRollupUnavailable) != want {
			t.Errorf("%s: level %s", tier, r.LevelOf(health.CodeRollupUnavailable))
		}
	}
	// No failure, no diagnostic.
	dir, manifest := fixture(t, "lifecycle:\n  tier: product\n")
	r, err := Scan(Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range r.Diagnostics {
		if d.Code == health.CodeRollupUnavailable {
			t.Errorf("unexpected %+v", d)
		}
	}
}

// A directory the walk cannot read is reported as health.file_skipped, in scope
// or on the way to a scoped path, and never when the scan is restricted to
// another directory.
// Mutation: ignoring the walk error leaves the report without the entry.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155, REQ-EMG-158
func TestScanReportsUnreadableDirectories(t *testing.T) {
	if os.Geteuid() == 0 {
		t.Skip("root reads every directory")
	}
	dir, manifest := fixture(t, "lifecycle:\n  tier: product\n")
	locked := filepath.Join(dir, "locked")
	testutil.WriteFile(t, filepath.Join(locked, "x.go"), "package locked\n")
	if err := os.Chmod(locked, 0); err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() { _ = os.Chmod(locked, 0o755) })
	if _, err := os.ReadDir(locked); err == nil {
		t.Skip("directory permissions are not enforced here")
	}

	r, err := Scan(Options{ManifestPath: manifest, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	d, ok := diagnosticFor(r, "locked", CodeFileSkipped)
	if !ok || d.Level != health.LevelError || strings.Contains(d.Message, dir) || !strings.Contains(d.Message, "cannot be read") {
		t.Errorf("whole-project scan: %+v %v", d, ok)
	}
	r, err = Scan(Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, NoCache: true})
	if err != nil {
		t.Fatal(err)
	}
	if _, ok := diagnosticFor(r, "locked", CodeFileSkipped); ok {
		t.Error("a directory outside the scope must not be reported")
	}
}

// A file the parser could only partly read reports health.syntax_error with the
// level of health.file_failed, whatever its other findings are.
// Mutation: not reading the error count leaves the broken file clean.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestScanReportsSyntaxErrors(t *testing.T) {
	for tier, want := range map[string]string{"poc": health.LevelWarning, "product": health.LevelError} {
		dir, manifest := fixture(t, "lifecycle:\n  tier: "+tier+"\n")
		testutil.WriteFile(t, filepath.Join(dir, "pkg", "broken.go"), "package pkg\n\nfunc Ok() int { return 1 }\n\nfunc Lost( {\n\tif x {\n")
		r, err := Scan(Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, NoCache: true})
		if err != nil {
			t.Fatal(err)
		}
		d, ok := diagnosticFor(r, "pkg/broken.go", health.CodeSyntaxError)
		if !ok || d.Level != want || !strings.Contains(d.Message, "syntax errors") {
			t.Errorf("%s: %+v %v, want level %s", tier, d, ok, want)
		}
		if _, clean := diagnosticFor(r, "pkg/clean.go", health.CodeSyntaxError); clean {
			t.Error("a clean file got a syntax diagnostic")
		}
		for _, f := range r.Files {
			if f.Path == "pkg/broken.go" && f.SyntaxErrors == 0 {
				t.Errorf("the file report holds no syntax error count: %+v", f)
			}
		}
	}
}
