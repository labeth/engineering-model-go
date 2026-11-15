// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// CheckTarget is a checkout selected for the model check and why.
type CheckTarget struct {
	Entry  Entry
	Reason string // local, depends on <checkout>, top system
}

// SelectCheckTargets returns the checkouts affected by the local set: the
// local checkouts, every checkout that depends on one of them directly or
// transitively inside the workspace, and the top systems (checkouts no other
// checkout depends on). The result is sorted by checkout name.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func SelectCheckTargets(plan Plan) []CheckTarget {
	consumers, dependedOn := checkConsumers(plan)
	reason := checkReasons(plan, consumers)
	for i := range plan.Entries {
		if _, seen := reason[i]; !seen && !dependedOn[i] {
			reason[i] = "top system"
		}
	}
	out := make([]CheckTarget, 0, len(reason))
	for i, why := range reason {
		out = append(out, CheckTarget{Entry: plan.Entries[i], Reason: why})
	}
	sort.Slice(out, func(a, b int) bool { return out[a].Entry.Name < out[b].Entry.Name })
	return out
}

// checkConsumers returns, per checkout index, the checkouts that depend on it
// and whether any checkout does.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func checkConsumers(plan Plan) ([][]int, []bool) {
	byModule := map[string]int{}
	for i, entry := range plan.Entries {
		byModule[entry.Manifest.Module.Path] = i
	}
	consumers := make([][]int, len(plan.Entries))
	dependedOn := make([]bool, len(plan.Entries))
	for i, entry := range plan.Entries {
		for _, dep := range entry.Manifest.Dependencies {
			j, ok := byModule[strings.TrimSpace(dep.Path)]
			if ok && j != i {
				consumers[j] = append(consumers[j], i)
				dependedOn[j] = true
			}
		}
	}
	return consumers, dependedOn
}

// checkReasons marks the local checkouts and, breadth first, their consumers.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func checkReasons(plan Plan, consumers [][]int) map[int]string {
	reason := map[int]string{}
	var queue []int
	for i, entry := range plan.Entries {
		if entry.Local {
			reason[i] = "local"
			queue = append(queue, i)
		}
	}
	for len(queue) > 0 {
		j := queue[0]
		queue = queue[1:]
		for _, i := range consumers[j] {
			if _, seen := reason[i]; !seen {
				reason[i] = "depends on " + plan.Entries[j].Name
				queue = append(queue, i)
			}
		}
	}
	return reason
}

// CheckResult is the outcome of the model check in one checkout.
type CheckResult struct {
	Target   CheckTarget
	Method   string // exec: the repo's pinned .github/engmod-check; library: in-process
	Errors   int
	Warnings int
	Lines    []string // error lines and other failure output
	Failed   bool
}

// Line is the one-line summary: ok|FAIL repo (n errors, m warnings).
// TRLC-LINKS: REQ-EMG-068
func (r CheckResult) Line() string {
	status := "ok"
	if r.Failed {
		status = "FAIL"
	}
	return fmt.Sprintf("%-4s %s (%d errors, %d warnings) [%s; %s]", status, r.Target.Entry.Name, r.Errors, r.Warnings, r.Method, r.Target.Reason)
}

// CheckOptions configures RunChecks.
type CheckOptions struct {
	// Root is the workspace root the checks run against.
	Root string
	// InProcess forces the library check even when a repo pins its own; without
	// it a repository whose check template cannot take the workspace root is
	// engine.check_template_stale, not checked by the library (ADR-EMG-072).
	InProcess bool
	// Jobs bounds the concurrent exec checks; 0 means GOMAXPROCS.
	Jobs int
	// Refresh re-fetches every registry dependency instead of reusing the
	// module cache (ENGMOD_MODULE_CACHE=refresh), so a moved tag is seen.
	Refresh bool
}

// RunChecks runs the model check in every target. A repository that ships
// .github/engmod-check with --workspace-root support is checked with its
// own pinned engine (go run with GOWORK from ToolGoWork, so the root go.work
// never captures the check module), and its result carries engine.pin_outdated
// when that pin is older than the running engine (ADR-EMG-072). A repository
// without such a template is not checked by another route: its result is the
// error engine.check_template_stale with the remedy engflow upgrade, unless
// opts.InProcess asks for the library in this binary, which then checks every
// target in-process. Exec checks run concurrently; in-process checks run one
// at a time because they share process state (environment, caches).
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-180, REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func RunChecks(ctx context.Context, targets []CheckTarget, opts CheckOptions) []CheckResult {
	results := make([]CheckResult, len(targets))
	if opts.Refresh {
		restore := setEnv(engmodel.ModuleCacheEnv, engmodel.ModuleCacheRefresh)
		defer restore()
	}
	job := &checkJob{ctx: ctx, opts: opts, results: results}
	for _, i := range job.dispatch(targets) {
		restore := guardLock(targets[i].Entry.Dir)
		results[i] = libraryCheck(targets[i])
		restore(&results[i])
	}
	job.wg.Wait()
	return results
}

// checkJob is the shared state of one RunChecks call.
type checkJob struct {
	ctx     context.Context
	opts    CheckOptions
	results []CheckResult
	wg      sync.WaitGroup
}

// dispatch starts the exec checks, records the stale ones and returns the
// indexes of the targets to check in-process.
// TRLC-LINKS: REQ-EMG-068, REQ-EMG-180, REQ-EMG-181
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func (j *checkJob) dispatch(targets []CheckTarget) []int {
	jobs := j.opts.Jobs
	if jobs <= 0 {
		jobs = runtime.GOMAXPROCS(0)
	}
	sem := make(chan struct{}, jobs)
	var inProcess []int
	for i, target := range targets {
		switch routeOf(target.Entry.Dir, j.opts) {
		case routeExec:
			j.wg.Add(1)
			go func(i int, target CheckTarget) {
				defer j.wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				j.results[i] = pinnedExec(j.ctx, j.opts.Root, target)
			}(i, target)
		case routeStale:
			j.results[i] = staleResult(target)
		default:
			inProcess = append(inProcess, i)
		}
	}
	return inProcess
}

// guardLock snapshots dir/engmod.lock.yml and returns a function that puts it
// back byte for byte (or removes a lock the check created). Engines without
// the workspace lock policy (REQ-EMG-064) write engmod.lock.yml from local
// replacements; a check must not change a checkout.
// TRLC-LINKS: REQ-EMG-068
func guardLock(dir string) func(*CheckResult) {
	restoreLocal := guardLockFile(filepath.Join(dir, engmodel.CompositionLocalLockFileName))
	restore := guardLockFile(filepath.Join(dir, engmodel.CompositionLockFileName))
	return func(result *CheckResult) {
		restore(result)
		restoreLocal(result)
	}
}

// guardLockFile restores or removes one lock file after the check.
// TRLC-LINKS: REQ-EMG-068
func guardLockFile(path string) func(*CheckResult) {
	before, err := os.ReadFile(path)
	existed := err == nil
	return func(result *CheckResult) {
		after, err := os.ReadFile(path)
		switch {
		case !existed && err == nil:
			lockRemoveCreated(path, result)
		case existed && (err != nil || !bytes.Equal(before, after)):
			lockRestoreChanged(path, before, result)
		}
	}
}

// lockRemoveCreated removes a lock file the check created.
// TRLC-LINKS: REQ-EMG-068
func lockRemoveCreated(path string, result *CheckResult) {
	if os.Remove(path) == nil {
		result.Lines = append(result.Lines, "note: removed "+filepath.Base(path)+" written by the check")
	}
}

// lockRestoreChanged writes back a lock file the check changed or removed.
// TRLC-LINKS: REQ-EMG-068
func lockRestoreChanged(path string, before []byte, result *CheckResult) {
	if writeFileAtomic(path, before) == nil {
		result.Lines = append(result.Lines, "note: restored "+filepath.Base(path)+" changed by the check")
	}
}

// pinnedCheck describes a repository's own .github/engmod-check.
type pinnedCheck struct {
	present bool // a Go check that accepts --workspace-root
	summary bool // it also accepts --summary
}

// detectPinnedCheck inspects dir/.github/engmod-check. Without
// --workspace-root a template check refuses the workspace file above the
// project, so such a check is not used.
// TRLC-LINKS: REQ-EMG-068
func detectPinnedCheck(dir string) pinnedCheck {
	checkDir := filepath.Join(dir, ".github", "engmod-check")
	if _, err := os.Stat(filepath.Join(checkDir, "go.mod")); err != nil {
		return pinnedCheck{}
	}
	var found pinnedCheck
	files, _ := filepath.Glob(filepath.Join(checkDir, "*.go"))
	for _, file := range files {
		data, err := os.ReadFile(file)
		if err != nil {
			continue
		}
		found.present = found.present || bytes.Contains(data, []byte(`"workspace-root"`))
		found.summary = found.summary || bytes.Contains(data, []byte(`"summary"`))
	}
	if !found.present {
		return pinnedCheck{}
	}
	return found
}

// pinnedCheckSupportsWorkspace reports whether the repo's own check is used.
// TRLC-LINKS: REQ-EMG-068
func pinnedCheckSupportsWorkspace(dir string) bool {
	return detectPinnedCheck(dir).present
}

var (
	errorLine   = regexp.MustCompile(`^\S+ \[error\] `)
	warningLine = regexp.MustCompile(`^\S+ \[warning\](?: x(\d+))?`)
)

// ToolGoWork is the GOWORK value for running a repository's own
// .github/engmod-check: its go.work when the repository deliberately builds
// the check against a local engine, else "off". The workspace root go.work
// lists the checkouts, not the tool module, and is never inherited.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func ToolGoWork(repoDir string) string {
	if path := filepath.Join(repoDir, ".github", "engmod-check", GoWorkFileName); fileExists(path) {
		if abs, err := filepath.Abs(path); err == nil {
			return abs
		}
		return path
	}
	return "off"
}

// execCheck runs the repository's own pinned check with summary output.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FLOW-ENGINE-INTEGRATIONS
func execCheck(ctx context.Context, root string, target CheckTarget) CheckResult {
	result := CheckResult{Target: target, Method: "exec .github/engmod-check"}
	out, err := execCheckCommand(ctx, root, target).CombinedOutput()
	execCountLines(&result, string(out))
	if err != nil {
		result.Failed = true
		if result.Errors == 0 {
			// Build, flag or trace failure without diagnostic lines: show the tail.
			result.Lines = append(result.Lines, tail(strings.TrimSpace(string(out)), 8)...)
		}
	}
	return result
}

// execCheckCommand builds the go run command of the repository's own check.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FLOW-ENGINE-INTEGRATIONS
func execCheckCommand(ctx context.Context, root string, target CheckTarget) *exec.Cmd {
	args := []string{"-C", ".github/engmod-check", "run", ".",
		"--model", "../../engmod.yml", "--code-root", ".", "--workspace-root", root}
	if detectPinnedCheck(target.Entry.Dir).summary {
		args = append(args, "--summary")
	}
	cmd := exec.CommandContext(ctx, "go", args...)
	cmd.Dir = target.Entry.Dir
	cmd.Env = append(checkEnvironment(os.Environ()), "GOWORK="+ToolGoWork(target.Entry.Dir))
	if work := filepath.Join(root, model.WorkspaceFileName); fileExists(work) && os.Getenv("ENGMOD_WORK") == "" {
		// Pin the check to the root work file; engines without ENGMOD_WORK
		// support ignore it and find the same file by parent search.
		cmd.Env = append(cmd.Env, "ENGMOD_WORK="+work)
	}
	return cmd
}

// execCountLines counts the error and warning lines of the check output and
// keeps the error lines.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FLOW-ENGINE-INTEGRATIONS
func execCountLines(result *CheckResult, out string) {
	for _, line := range strings.Split(out, "\n") {
		switch {
		case errorLine.MatchString(line):
			result.Errors++
			result.Lines = append(result.Lines, line)
		case warningLine.MatchString(line):
			result.Warnings += warningCount(line)
		}
	}
}

// warningCount is the repeat count of a warning line (x<n>), else 1.
// TRLC-LINKS: REQ-EMG-068
func warningCount(line string) int {
	if m := warningLine.FindStringSubmatch(line); m[1] != "" {
		n, _ := strconv.Atoi(m[1])
		return n
	}
	return 1
}

// checkEnvAllowlist names the variables a repository's own check receives;
// ENGMOD_* variables pass as well. Everything else (provider API keys, cloud
// credentials, tokens) stays with the caller: sibling checkout code must not
// see it.
var checkEnvAllowlist = map[string]bool{
	"PATH": true, "HOME": true, "USER": true, "LOGNAME": true, "LANG": true, "LC_ALL": true, "TERM": true,
	"GOPATH": true, "GOMODCACHE": true, "GOCACHE": true, "GOENV": true, "GOROOT": true,
	"GOPROXY": true, "GOSUMDB": true, "GOPRIVATE": true, "GONOSUMDB": true, "GONOPROXY": true,
	"GOFLAGS": true, "GOTOOLCHAIN": true, "GOINSECURE": true,
	"TMPDIR": true, "SSL_CERT_FILE": true, "SSL_CERT_DIR": true,
	"HTTP_PROXY": true, "HTTPS_PROXY": true, "NO_PROXY": true, "http_proxy": true, "https_proxy": true, "no_proxy": true,
	"CUE_REGISTRY": true, "CUE_CONFIG_DIR": true, "CUE_CACHE_DIR": true,
	"XDG_CONFIG_HOME": true, "XDG_CACHE_HOME": true, "XDG_RUNTIME_DIR": true,
	"DOCKER_CONFIG": true, "CI": true,
}

// checkEnvironment filters environ to the allowlist and ENGMOD_*; GOWORK is
// dropped because the caller sets it per checkout.
// TRLC-LINKS: REQ-EMG-068
func checkEnvironment(environ []string) []string {
	out := make([]string, 0, len(checkEnvAllowlist))
	for _, kv := range environ {
		key, _, _ := strings.Cut(kv, "=")
		if checkEnvAllowlist[key] || strings.HasPrefix(key, "ENGMOD_") {
			out = append(out, kv)
		}
	}
	return out
}

// fileExists reports whether path names an existing file.
// TRLC-LINKS: REQ-EMG-068
func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// tail returns the last n lines of text.
// TRLC-LINKS: REQ-EMG-068
func tail(text string, n int) []string {
	lines := strings.Split(text, "\n")
	if len(lines) > n {
		lines = lines[len(lines)-n:]
	}
	return lines
}

// libraryCheck validates the model in-process like the template check: load
// the bundle, generate the document (validation, EARS lint, trace integrity),
// add composition diagnostics, and build the trace matrix with the checkout
// as code root; dangling trace links fail.
// TRLC-LINKS: REQ-EMG-068
func libraryCheck(target CheckTarget) CheckResult {
	result := CheckResult{Target: target, Method: "library"}
	manifest := filepath.Join(target.Entry.Dir, "engmod.yml")
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		return libraryFail(result, "load: %v", err)
	}
	run, failure := libraryGather(bundle, manifest, target.Entry.Dir)
	if failure != "" {
		return libraryFail(result, "%s", failure)
	}
	libraryTally(&result, engmodel.ApplyTierPolicy(bundle, run.diagnostics))
	return libraryFinish(result, run)
}

// libraryFinish adds the generation error and the dangling trace links to the
// tallied result and decides whether it failed.
// TRLC-LINKS: REQ-EMG-068
func libraryFinish(result CheckResult, run libraryRun) CheckResult {
	if run.genErr != nil && result.Errors == 0 {
		return libraryFail(result, "generate: %v", run.genErr)
	}
	if run.dangling > 0 {
		result.Lines = append(result.Lines, fmt.Sprintf("trace: %d dangling links", run.dangling))
	}
	result.Failed = run.genErr != nil || result.Errors > 0 || run.dangling > 0
	return result
}

// libraryFail records one failure line and marks the result failed.
// TRLC-LINKS: REQ-EMG-068
func libraryFail(result CheckResult, format string, args ...any) CheckResult {
	result.Failed = true
	result.Errors++
	result.Lines = append(result.Lines, fmt.Sprintf(format, args...))
	return result
}

// libraryRun is what the in-process check gathered about one model.
type libraryRun struct {
	diagnostics []validate.Diagnostic
	dangling    int
	genErr      error // the document generation error, reported after the diagnostics
}

// libraryGather generates the document, adds the composition diagnostics and
// builds the trace matrix. A composition or trace error is the failure text.
// TRLC-LINKS: REQ-EMG-068
func libraryGather(bundle model.Bundle, manifest, codeRoot string) (libraryRun, string) {
	// GenerateAsciiDoc validates alias::ID references against the
	// dependencies' publications (REQ-EMG-065) and returns the composition
	// diagnostics (REQ-EMG-066).
	doc, err := engmodel.GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, engmodel.AsciiDocOptions{})
	run := libraryRun{diagnostics: doc.Diagnostics, genErr: err}
	if err != nil {
		return run, ""
	}
	if engmodel.HasComposition(bundle) {
		comp, cerr := engmodel.GenerateCompositionFromFile(manifest)
		if cerr != nil {
			return run, fmt.Sprintf("composition: %v", cerr)
		}
		run.diagnostics = append(run.diagnostics, comp.Diagnostics...)
	}
	matrix, traceDiags, terr := engmodel.BuildTraceMatrixFromFiles(manifest, bundle.RequirementsPath, codeRoot)
	if terr != nil {
		return run, fmt.Sprintf("trace: %v", terr)
	}
	run.diagnostics = append(run.diagnostics, traceDiags...)
	run.dangling = matrix.Summary.DanglingLinks
	return run, ""
}

// libraryTally counts the distinct error and warning diagnostics into result.
// TRLC-LINKS: REQ-EMG-068
func libraryTally(result *CheckResult, diagnostics []validate.Diagnostic) {
	seen := map[string]bool{}
	for _, d := range validate.SortDiagnostics(diagnostics) {
		key := d.Code + "|" + d.Path + "|" + d.Message
		if seen[key] {
			continue
		}
		seen[key] = true
		switch d.Severity {
		case validate.SeverityError:
			result.Errors++
			result.Lines = append(result.Lines, fmt.Sprintf("%s [error] %s (%s)", d.Code, d.Message, d.Path))
			if fix := d.FixLine(); fix != "" {
				result.Lines = append(result.Lines, fix)
			}
		case validate.SeverityWarning:
			result.Warnings++
		}
	}
}

// WorkFileDrift reports how root/engmod.work.yml differs from the plan's
// local set, so check can warn that sync was not run.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func WorkFileDrift(plan Plan) string {
	return workFileDrift(plan, true)
}

// CheckAllDrift is WorkFileDrift for check --all, whose plan marks every
// checkout local on purpose: it warns only when a changed or pinned checkout
// is missing from engmod.work.yml, never about the unchanged ones --all adds
// or about extra replacements.
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION
func CheckAllDrift(plan Plan) string {
	needed := Plan{Root: plan.Root, Notes: plan.Notes, Entries: make([]Entry, len(plan.Entries))}
	for i, entry := range plan.Entries {
		entry.Local = entry.Pinned || len(entry.Reasons) > 0
		needed.Entries[i] = entry
	}
	return workFileDrift(needed, false)
}

// workFileDrift compares engmod.work.yml with the plan's local set; extra
// replacements are reported when withExtra is set.
// TRLC-LINKS: REQ-EMG-068
func workFileDrift(plan Plan, withExtra bool) string {
	path := filepath.Join(plan.Root, model.WorkspaceFileName)
	if _, err := os.Stat(path); err != nil {
		if len(plan.Local()) == 0 {
			return ""
		}
		return model.WorkspaceFileName + " is missing; run engws sync"
	}
	workspace, err := model.LoadWorkspace(path)
	if err != nil {
		return fmt.Sprintf("%s is invalid: %v", model.WorkspaceFileName, err)
	}
	diffs := workFileDiffs(plan, workspace, withExtra)
	if len(diffs) == 0 {
		return ""
	}
	sort.Strings(diffs)
	return fmt.Sprintf("%s is out of date (%s); run engws sync", model.WorkspaceFileName, strings.Join(diffs, ", "))
}

// workFileDiffs lists the local checkouts missing from the work file and, with
// withExtra, the replacements no local checkout wants.
// TRLC-LINKS: REQ-EMG-068
func workFileDiffs(plan Plan, workspace model.WorkspaceDocument, withExtra bool) []string {
	have := map[string]string{}
	for _, r := range workspace.Replacements {
		have[r.Module] = filepath.Clean(r.Path)
	}
	want := map[string]string{}
	for _, entry := range plan.Local() {
		want[entry.Manifest.Module.Path] = entry.Name
	}
	var diffs []string
	for module, name := range want {
		if have[module] != name {
			diffs = append(diffs, "missing "+name)
		}
	}
	for module, path := range have {
		if _, ok := want[module]; !ok && withExtra {
			diffs = append(diffs, "extra "+path)
		}
	}
	return diffs
}
