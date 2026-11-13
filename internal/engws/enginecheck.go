// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"runtime/debug"
	"strings"

	"golang.org/x/mod/modfile"
	"golang.org/x/mod/module"
	"golang.org/x/mod/semver"

	"github.com/labeth/engineering-model-go/internal/validate"
)

// The codes the check reports instead of checking a repository by another
// route (ADR-EMG-072, removal plan phase 1). Both are errors that no tier
// changes; their remedies are in validate.RequiredSwitches.
const (
	codeCheckTemplateStale = "engine.check_template_stale"
	codePinOutdated        = "engine.pin_outdated"
	checkTemplateDir       = ".github/engmod-check"
)

// checkRoute is how RunChecks checks one target.
type checkRoute int

const (
	routeLibrary checkRoute = iota // in-process library, only by explicit request
	routeExec                      // the repository's own .github/engmod-check
	routeStale                     // not checked: its template cannot take the workspace root
)

// routeOf decides how a repository is checked: the library only when the
// in-process option is given, otherwise its own check template, and a
// repository whose template cannot take the workspace root (or has none) is
// stale instead of being checked by the library.
// TRLC-LINKS: REQ-EMG-180
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func routeOf(dir string, opts CheckOptions) checkRoute {
	if opts.InProcess {
		return routeLibrary
	}
	if pinnedCheckSupportsWorkspace(dir) {
		return routeExec
	}
	return routeStale
}

// staleResult is the result of a repository that is not checked because its
// check template cannot take the workspace root: one error diagnostic with
// the remedy to run engflow upgrade for that directory.
// TRLC-LINKS: REQ-EMG-180
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func staleResult(target CheckTarget) CheckResult {
	d := validate.Diagnostic{
		Code: codeCheckTemplateStale, Severity: validate.SeverityError, Path: checkTemplateDir,
		Message: "the repository has no " + checkTemplateDir + " that takes --workspace-root (missing, or its source lacks the flag), so it is not checked by another route",
		Remedy:  strings.ReplaceAll(validate.RequiredRemedy(codeCheckTemplateStale), "<dir>", target.Entry.Dir),
	}
	result := CheckResult{Target: target, Method: "not checked (" + codeCheckTemplateStale + ")"}
	addErrorDiagnostics(&result, []validate.Diagnostic{d})
	return result
}

// addErrorDiagnostics adds error diagnostics to a result: the error count,
// the failure, and the diagnostic line with its fix line (validate.WriteDiagnostic).
// TRLC-LINKS: REQ-EMG-180, REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func addErrorDiagnostics(result *CheckResult, diags []validate.Diagnostic) {
	for _, d := range diags {
		var out bytes.Buffer
		validate.WriteDiagnostic(&out, d)
		result.Errors++
		result.Failed = true
		result.Lines = append(result.Lines, strings.Split(strings.TrimRight(out.String(), "\n"), "\n")...)
	}
}

// engineVersion returns the version of the running engine; a test replaces it.
var engineVersion = buildEngineVersion

// buildEngineVersion reads the engineering-model-go version from the build
// information of the running binary: the main module when this engine is the
// main module, else its dependency entry. A devel build has no release
// version and returns whatever the toolchain stamped.
// TRLC-LINKS: REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func buildEngineVersion() string {
	info, ok := debug.ReadBuildInfo()
	if !ok {
		return ""
	}
	if info.Main.Path == engineModule {
		return info.Main.Version
	}
	for _, dep := range info.Deps {
		if dep.Path == engineModule {
			return dep.Version
		}
	}
	return ""
}

// comparableVersion reports whether a version of the running engine can be
// compared with a pin: a valid semantic version that is neither a
// pseudo-version nor carries build metadata such as +dirty (a devel build).
// TRLC-LINKS: REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func comparableVersion(v string) bool {
	return semver.IsValid(v) && !module.IsPseudoVersion(v) && semver.Build(v) == ""
}

// enginePinOf returns the engineering-model-go version the go.mod of
// dir/.github/engmod-check requires. It reports false for a missing or
// unparsable go.mod (ParseLax would drop the replace directives), a missing requirement, a replace directive for the
// engine (a local engine is not a pin) and a go.work in that directory (the
// check is deliberately built against a local engine).
// TRLC-LINKS: REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func enginePinOf(dir string) (string, bool) {
	checkDir := filepath.Join(dir, ".github", "engmod-check")
	data, err := os.ReadFile(filepath.Join(checkDir, "go.mod"))
	if err != nil || fileExists(filepath.Join(checkDir, GoWorkFileName)) {
		return "", false
	}
	file, err := modfile.Parse("go.mod", data, nil)
	if err != nil || replacesEngine(file) {
		return "", false
	}
	for _, r := range file.Require {
		if r.Mod.Path == engineModule {
			return r.Mod.Version, true
		}
	}
	return "", false
}

// replacesEngine reports whether a go.mod replaces the engine module.
// TRLC-LINKS: REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func replacesEngine(file *modfile.File) bool {
	for _, r := range file.Replace {
		if r.Old.Path == engineModule {
			return true
		}
	}
	return false
}

// pinOutdatedDiagnostics reports the pin of dir's check template when it is a
// valid version older than the running engine. A devel running engine, an
// unparsable or absent pin and an equal or newer pin report nothing.
// TRLC-LINKS: REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func pinOutdatedDiagnostics(dir string) []validate.Diagnostic {
	running := engineVersion()
	if !comparableVersion(running) {
		return nil
	}
	pin, ok := enginePinOf(dir)
	if !ok || !semver.IsValid(pin) || semver.Compare(pin, running) >= 0 {
		return nil
	}
	return []validate.Diagnostic{{
		Code: codePinOutdated, Severity: validate.SeverityError, Path: checkTemplateDir + "/go.mod",
		Message: "the check template pins " + engineModule + " " + pin + ", older than the running engine " + running,
		Remedy:  strings.ReplaceAll(validate.RequiredRemedy(codePinOutdated), "<version>", running),
	}}
}

// pinnedExec runs the repository's own check with the lock guard and adds
// the pin diagnostic to its result.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func pinnedExec(ctx context.Context, root string, target CheckTarget) CheckResult {
	restore := guardLock(target.Entry.Dir)
	result := execCheck(ctx, root, target)
	addErrorDiagnostics(&result, pinOutdatedDiagnostics(target.Entry.Dir))
	restore(&result)
	return result
}
