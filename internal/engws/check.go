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
	byModule := map[string]int{}
	for i, entry := range plan.Entries {
		byModule[entry.Manifest.Module.Path] = i
	}
	// consumers[i] lists the checkouts that depend on checkout i.
	consumers := make([][]int, len(plan.Entries))
	dependedOn := make([]bool, len(plan.Entries))
	for i, entry := range plan.Entries {
		for _, dep := range entry.Manifest.Dependencies {
			j, ok := byModule[strings.TrimSpace(dep.Path)]
			if !ok || j == i {
				continue
			}
			consumers[j] = append(consumers[j], i)
			dependedOn[j] = true
		}
	}
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
	// InProcess forces the library check even when a repo pins its own.
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
// never captures the check module); otherwise the library in this binary checks it
// in-process. Exec checks run concurrently; in-process checks run one at a
// time because they share process state (environment, caches).
// TRLC-LINKS: REQ-EMG-068
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION
func RunChecks(ctx context.Context, root string, targets []CheckTarget, opts CheckOptions) []CheckResult {
	results := make([]CheckResult, len(targets))
	jobs := opts.Jobs
	if jobs <= 0 {
		jobs = runtime.GOMAXPROCS(0)
	}
	if opts.Refresh {
		restore := setEnv(engmodel.ModuleCacheEnv, engmodel.ModuleCacheRefresh)
		defer restore()
	}
	sem := make(chan struct{}, jobs)
	var wg sync.WaitGroup
	var inProcess []int
	for i, target := range targets {
		if !opts.InProcess && pinnedCheckSupportsWorkspace(target.Entry.Dir) {
			wg.Add(1)
			go func(i int, target CheckTarget) {
				defer wg.Done()
				sem <- struct{}{}
				defer func() { <-sem }()
				restore := guardLock(target.Entry.Dir)
				results[i] = execCheck(ctx, root, target)
				restore(&results[i])
			}(i, target)
			continue
		}
		inProcess = append(inProcess, i)
	}
	for _, i := range inProcess {
		restore := guardLock(targets[i].Entry.Dir)
		results[i] = libraryCheck(targets[i])
		restore(&results[i])
	}
	wg.Wait()
	return results
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
			if os.Remove(path) == nil {
				result.Lines = append(result.Lines, "note: removed "+filepath.Base(path)+" written by the check")
			}
		case existed && (err != nil || !bytes.Equal(before, after)):
			if writeFileAtomic(path, before) == nil {
				result.Lines = append(result.Lines, "note: restored "+filepath.Base(path)+" changed by the check")
			}
		}
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
func execCheck(ctx context.Context, root string, target CheckTarget) CheckResult {
	result := CheckResult{Target: target, Method: "exec .github/engmod-check"}
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
	out, err := cmd.CombinedOutput()
	for _, line := range strings.Split(string(out), "\n") {
		switch {
		case errorLine.MatchString(line):
			result.Errors++
			result.Lines = append(result.Lines, line)
		case warningLine.MatchString(line):
			n := 1
			if m := warningLine.FindStringSubmatch(line); m[1] != "" {
				n, _ = strconv.Atoi(m[1])
			}
			result.Warnings += n
		}
	}
	if err != nil {
		result.Failed = true
		if result.Errors == 0 {
			// Build, flag or trace failure without diagnostic lines: show the tail.
			result.Lines = append(result.Lines, tail(strings.TrimSpace(string(out)), 8)...)
		}
	}
	return result
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
	fail := func(format string, args ...any) CheckResult {
		result.Failed = true
		result.Errors++
		result.Lines = append(result.Lines, fmt.Sprintf(format, args...))
		return result
	}
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		return fail("load: %v", err)
	}
	// GenerateAsciiDoc validates alias::ID references against the
	// dependencies' publications (REQ-EMG-065) and returns the composition
	// diagnostics (REQ-EMG-066).
	doc, err := engmodel.GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, engmodel.AsciiDocOptions{})
	diagnostics := doc.Diagnostics
	if err == nil && engmodel.HasComposition(bundle) {
		comp, cerr := engmodel.GenerateCompositionFromFile(manifest)
		if cerr != nil {
			return fail("composition: %v", cerr)
		}
		diagnostics = append(diagnostics, comp.Diagnostics...)
	}
	dangling := 0
	if err == nil {
		matrix, traceDiags, terr := engmodel.BuildTraceMatrixFromFiles(manifest, bundle.RequirementsPath, target.Entry.Dir)
		if terr != nil {
			return fail("trace: %v", terr)
		}
		diagnostics = append(diagnostics, traceDiags...)
		dangling = matrix.Summary.DanglingLinks
	}
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
		case validate.SeverityWarning:
			result.Warnings++
		}
	}
	if err != nil && result.Errors == 0 {
		return fail("generate: %v", err)
	}
	if dangling > 0 {
		result.Lines = append(result.Lines, fmt.Sprintf("trace: %d dangling links", dangling))
	}
	result.Failed = err != nil || result.Errors > 0 || dangling > 0
	return result
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
	if len(diffs) == 0 {
		return ""
	}
	sort.Strings(diffs)
	return fmt.Sprintf("%s is out of date (%s); run engws sync", model.WorkspaceFileName, strings.Join(diffs, ", "))
}
