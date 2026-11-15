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
		t.Run(tier, func(t *testing.T) { requireRollupDiagnostic(t, tier, want) })
	}
	// No failure, no diagnostic.
	dir, manifest := fixture(t, "lifecycle:\n  tier: product\n")
	r := mustScan(t, Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, NoCache: true})
	wantEqual(t, len(diagnosticsWithCode(r, health.CodeRollupUnavailable)), 0)
}

// diagnosticsWithCode lists the diagnostics of a code.
func diagnosticsWithCode(r health.Report, code string) []health.Diagnostic {
	var found []health.Diagnostic
	for _, d := range r.Diagnostics {
		if d.Code == code {
			found = append(found, d)
		}
	}
	return found
}

// requireRollupDiagnostic scans with a code scan that fails and checks the one
// roll-up diagnostic: its level, no absolute path, the reason.
func requireRollupDiagnostic(t *testing.T, tier, want string) {
	t.Helper()
	dir, manifest := fixture(t, "lifecycle:\n  tier: "+tier+"\n")
	old := inferCodeItems
	inferCodeItems = func(model.Bundle, string) ([]codescan.CodeItem, []validate.Diagnostic) {
		return nil, []validate.Diagnostic{{Code: "code.scan_failed", Severity: validate.SeverityWarning, Message: "open " + dir + "/pkg: denied", Path: dir}}
	}
	r, err := Scan(Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, NoCache: true})
	inferCodeItems = old
	mustEqual(t, err, nil)
	found := diagnosticsWithCode(r, health.CodeRollupUnavailable)
	mustEqual(t, len(found), 1)
	wantEqual(t, [2]any{found[0].Level, strings.Contains(found[0].Message, dir)}, [2]any{want, false})
	wantContains(t, found[0].Message, "code.scan_failed")
	wantEqual(t, r.LevelOf(health.CodeRollupUnavailable), want)
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

	r := mustScan(t, Options{ManifestPath: manifest, NoCache: true})
	d, ok := diagnosticFor(r, "locked", CodeFileSkipped)
	wantEqual(t, [3]any{ok, d.Level, strings.Contains(d.Message, dir)}, [3]any{true, health.LevelError, false})
	wantContains(t, d.Message, "cannot be read")
	r = mustScan(t, Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, NoCache: true})
	_, ok = diagnosticFor(r, "locked", CodeFileSkipped)
	wantEqual(t, ok, false) // a directory outside the scope is not reported
}

// A file the parser could only partly read reports health.syntax_error with the
// level of health.file_failed, whatever its other findings are.
// Mutation: not reading the error count leaves the broken file clean.
//
// TRLC-LINKS: REQ-EMG-150, REQ-EMG-155
func TestScanReportsSyntaxErrors(t *testing.T) {
	for tier, want := range map[string]string{"poc": health.LevelWarning, "product": health.LevelError} {
		t.Run(tier, func(t *testing.T) { requireSyntaxDiagnostic(t, tier, want) })
	}
}

// requireSyntaxDiagnostic scans a broken file at a tier and checks that the
// syntax diagnostic has the level, that a clean file has none and that the file
// report holds the count.
func requireSyntaxDiagnostic(t *testing.T, tier, want string) {
	t.Helper()
	dir, manifest := fixture(t, "lifecycle:\n  tier: "+tier+"\n")
	testutil.WriteFile(t, filepath.Join(dir, "pkg", "broken.go"), "package pkg\n\nfunc Ok() int { return 1 }\n\nfunc Lost( {\n\tif x {\n")
	r := mustScan(t, Options{ManifestPath: manifest, Paths: []string{filepath.Join(dir, "pkg")}, NoCache: true})
	d, ok := diagnosticFor(r, "pkg/broken.go", health.CodeSyntaxError)
	wantEqual(t, [2]any{ok, d.Level}, [2]any{true, want})
	wantContains(t, d.Message, "syntax errors")
	_, clean := diagnosticFor(r, "pkg/clean.go", health.CodeSyntaxError)
	wantEqual(t, clean, false)
	for _, f := range r.Files {
		wantEqual(t, [2]any{f.Path, f.Path == "pkg/broken.go" && f.SyntaxErrors == 0}, [2]any{f.Path, false})
	}
}
