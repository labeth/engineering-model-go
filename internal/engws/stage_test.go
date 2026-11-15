// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"bufio"
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/validate"
)

// lockedBuffer is a goroutine-safe stdout for a foreground engws run.
type lockedBuffer struct {
	mu  sync.Mutex
	buf bytes.Buffer
}

func (b *lockedBuffer) Write(p []byte) (int, error) {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.Write(p)
}

func (b *lockedBuffer) String() string {
	b.mu.Lock()
	defer b.mu.Unlock()
	return b.buf.String()
}

// stageSpec describes a checkout written by stageCheckout.
type stageSpec struct {
	Root, Name, GoModule string
	Deps                 []dep
}

// stageCheckout writes a model whose CUE module has source kind git, an
// optional Go module, a git-ignored scratch file, commits and tags v0.1.0.
func stageCheckout(t *testing.T, spec stageSpec) string {
	t.Helper()
	dir := filepath.Join(spec.Root, spec.Name)
	writeModel(t, dir, spec.Name, "v0.1.0", spec.Deps...)
	write(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \""+modulePathOf(spec.Name)+"\"\nlanguage: version: \"v0.17.0\"\nsource: kind: \"git\"\n")
	write(t, filepath.Join(dir, ".gitignore"), ".engmod/\nengmod.lock.local.yml\nscratch.txt\n")
	write(t, filepath.Join(dir, "scratch.txt"), "not committed\n")
	if spec.GoModule != "" {
		write(t, filepath.Join(dir, "go.mod"), "module "+spec.GoModule+"\n\ngo 1.21\n")
		write(t, filepath.Join(dir, "name.go"), "package lib\n\n// Name names the module.\nfunc Name() string { return \""+spec.Name+"\" }\n")
	}
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")
	runGit(t, dir, "tag", "v0.1.0")
	return dir
}

// envKey reports whether a printed key is an environment variable name.
func envKey(key string) bool {
	return key != "" && key == strings.ToUpper(key) && !strings.ContainsAny(key, " :")
}

// envLines parses the KEY=value lines engws prints.
func envLines(out string) map[string]string {
	env := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, ok := strings.Cut(line, "=")
		if ok && envKey(key) {
			env[key] = strings.Trim(value, "'")
		}
	}
	return env
}

func noErrors(t *testing.T, what string, diags []validate.Diagnostic) {
	t.Helper()
	for _, d := range diags {
		if d.Severity == validate.SeverityError {
			t.Fatalf("%s: %s %s (%s)", what, d.Code, d.Message, d.Path)
		}
	}
}

// stageFamily is the three checkouts interface <- service <- system.
type stageFamily struct{ root, ifZ, svc, sys string }

// newStageFamily writes the family and a workspace that replaces every
// checkout; ENGMOD_WORK=off must ignore it.
func newStageFamily(t *testing.T) stageFamily {
	t.Helper()
	root := t.TempDir()
	fam := stageFamily{root: root}
	fam.ifZ = stageCheckout(t, stageSpec{Root: root, Name: "z-if", GoModule: "example.com/go/z-if"})
	fam.svc = stageCheckout(t, stageSpec{Root: root, Name: "m-svc", Deps: []dep{{"ia", "z-if", "v0.1.0"}}})
	fam.sys = stageCheckout(t, stageSpec{Root: root, Name: "a-sys", Deps: []dep{{"sv", "m-svc", "v0.1.0"}}})
	if code, out, errOut := runMain(t, allPublished("z-if", "m-svc", "a-sys"), "--root", root, "sync", "--all"); code != 0 {
		t.Fatalf("sync: %d\n%s\n%s", code, out, errOut)
	}
	return fam
}

// gitStatuses records the porcelain status, ignored files included, of
// every checkout.
func (f stageFamily) gitStatuses(t *testing.T) map[string]string {
	t.Helper()
	statuses := map[string]string{}
	for _, dir := range []string{f.ifZ, f.svc, f.sys} {
		statuses[dir] = runGit(t, dir, "status", "--porcelain", "--ignored")
	}
	return statuses
}

// stageServe is a foreground engws stage --serve run.
type stageServe struct {
	cancel         context.CancelFunc
	done           chan int
	stdout, stderr lockedBuffer
}

// startStageServe runs stage --serve on root and waits until it serves.
func startStageServe(t *testing.T, root string) *stageServe {
	t.Helper()
	ctx, cancel := context.WithCancel(context.Background())
	s := &stageServe{cancel: cancel, done: make(chan int, 1)}
	go func() {
		s.done <- Main(ctx, []string{"--root", root, "stage", "--serve"}, CLI{Stdout: &s.stdout, Stderr: &s.stderr, Registry: fakeRegistry{unknown: true}})
	}()
	deadline := time.Now().Add(60 * time.Second)
	for !strings.Contains(s.stdout.String(), "engws: serving") {
		s.failIfExited(t)
		if time.Now().After(deadline) {
			t.Fatalf("stage --serve did not start\n%s\n%s", s.stdout.String(), s.stderr.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	return s
}

// failIfExited fails the test when the run ended before serving.
func (s *stageServe) failIfExited(t *testing.T) {
	t.Helper()
	select {
	case code := <-s.done:
		t.Fatalf("stage exited %d before serving\n%s\n%s", code, s.stdout.String(), s.stderr.String())
	default:
	}
}

// stop cancels the run and requires a clean, timely exit.
func (s *stageServe) stop(t *testing.T) {
	t.Helper()
	s.cancel()
	select {
	case code := <-s.done:
		if code != 0 {
			t.Fatalf("serve exit %d\n%s", code, s.stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop on cancel")
	}
}

// requireStagedInOrder requires the staged lines in dependency order.
func requireStagedInOrder(t *testing.T, out string) {
	t.Helper()
	iA, iS, iY := strings.Index(out, "staged z-if "), strings.Index(out, "staged m-svc "), strings.Index(out, "staged a-sys ")
	if !(iA >= 0 && iA < iS && iS < iY) {
		t.Fatalf("not staged in dependency order:\n%s", out)
	}
}

// requireUnchanged requires every checkout status to be as recorded.
func requireUnchanged(t *testing.T, before map[string]string) {
	t.Helper()
	for dir, status := range before {
		if got := runGit(t, dir, "status", "--porcelain", "--ignored"); got != status {
			t.Fatalf("stage wrote inside %s:\n%s", dir, got)
		}
	}
}

// stagePrinted is what a stage --serve run printed and where it staged.
type stagePrinted struct {
	out, staging string
	env          map[string]string
}

// requireLayout requires the registry and Go proxy files in staging.
func (p stagePrinted) requireLayout(t *testing.T) {
	t.Helper()
	staging, out := p.staging, p.out
	for _, path := range []string{
		filepath.Join(staging, "registry", "example.com", "fam", "z-if", "oci-layout"),
		filepath.Join(staging, "registry", "example.com", "fam", "a-sys", "index.json"),
		filepath.Join(staging, "goproxy", "example.com", "go", "z-if", "@v", "v0.1.0.zip"),
		filepath.Join(staging, "goproxy", "example.com", "go", "z-if", "@v", "v0.1.0.info"),
	} {
		if _, err := os.Stat(path); err != nil {
			t.Fatalf("staging layout: %v\n%s", err, out)
		}
	}
}

// requireEnv requires the printed CUE and Go environment.
func (p stagePrinted) requireEnv(t *testing.T) {
	t.Helper()
	env, staging, out := p.env, p.staging, p.out
	if reg := env["CUE_REGISTRY"]; !strings.Contains(reg, testFamily+"/z-if=127.0.0.1:") || !strings.Contains(reg, testFamily+"/a-sys=") {
		t.Fatalf("CUE_REGISTRY: %q\n%s", env["CUE_REGISTRY"], out)
	}
	if env["GOPROXY"] != "file://"+filepath.ToSlash(filepath.Join(staging, "goproxy"))+",off" || env["GONOPROXY"] != "none" || !strings.Contains(env["GONOSUMDB"], "example.com/go/z-if") {
		t.Fatalf("Go env: %v\n%s", env, out)
	}
}

// composeOffline composes the model with the workspace off and the given
// lock mode ("" composes, "check" also checks the committed lock).
func composeOffline(t *testing.T, manifest, lock string) composition.CompositionResult {
	t.Helper()
	res, err := composition.GenerateCompositionFromFileWith(manifest, composition.ResolutionOptions{Workspace: "off", Lock: lock})
	if err != nil {
		t.Fatal(err)
	}
	what := "compose"
	if lock != "" {
		what = "lock check"
	}
	noErrors(t, what, res.Diagnostics)
	return res
}

// requireComposeFromStaging requires the system to compose from the staged
// registry alone, every dependency from the module cache without an
// uncommitted file, and its committed lock check to pass.
func requireComposeFromStaging(t *testing.T, sys string, env map[string]string) {
	t.Helper()
	t.Setenv("CUE_REGISTRY", env["CUE_REGISTRY"])
	manifest := filepath.Join(sys, "engmod.yml")
	res := composeOffline(t, manifest, "")
	if res.Root == nil || len(res.Root.Children) != 1 || len(res.Root.Children[0].Children) != 1 {
		t.Fatalf("composition tree: %#v", res.Root)
	}
	cache := filepath.Join(sys, ".engmod", "modules")
	for _, child := range []string{res.Root.Children[0].Dir, res.Root.Children[0].Children[0].Dir} {
		requireStagedDependency(t, child, cache)
	}
	composeOffline(t, manifest, "check")
}

// requireStagedDependency requires a dependency from the module cache that
// holds no uncommitted file.
func requireStagedDependency(t *testing.T, child, cache string) {
	t.Helper()
	if !strings.HasPrefix(child, cache) {
		t.Fatalf("dependency resolved from %s, not the staged registry", child)
	}
	if _, err := os.Stat(filepath.Join(child, "scratch.txt")); err == nil {
		t.Fatalf("staged module %s holds an uncommitted file", child)
	}
}

// requireGoConsumer requires an outside consumer to resolve the staged Go
// module through GOPROXY.
func requireGoConsumer(t *testing.T, env map[string]string) {
	t.Helper()
	consumer := filepath.Join(t.TempDir(), "consumer")
	write(t, filepath.Join(consumer, "go.mod"), "module example.com/consumer\n\ngo 1.21\n\nrequire example.com/go/z-if v0.1.0\n")
	write(t, filepath.Join(consumer, "main.go"), "package main\n\nimport (\n\t\"fmt\"\n\n\tlib \"example.com/go/z-if\"\n)\n\nfunc main() { fmt.Print(lib.Name()) }\n")
	goEnv := append(os.Environ(), "GOWORK=off", "GOFLAGS=-modcacherw", "GOTOOLCHAIN=local", "GOMODCACHE="+filepath.Join(t.TempDir(), "modcache"),
		"GOPROXY="+env["GOPROXY"], "GONOPROXY="+env["GONOPROXY"], "GONOSUMDB="+env["GONOSUMDB"], "GOPRIVATE=")
	// @latest outside any module reads the proxy's list file.
	latest := exec.Command("go", "list", "-m", "example.com/go/z-if@latest")
	latest.Dir, latest.Env = t.TempDir(), goEnv
	if b, err := latest.CombinedOutput(); err != nil || strings.TrimSpace(string(b)) != "example.com/go/z-if v0.1.0" {
		t.Fatalf("go list @latest: %v\n%s", err, b)
	}
	for _, args := range [][]string{{"mod", "tidy"}, {"build", "-o", "consumer", "."}} {
		cmd := exec.Command("go", args...)
		cmd.Dir, cmd.Env = consumer, goEnv
		if b, err := cmd.CombinedOutput(); err != nil {
			t.Fatalf("go %v: %v\n%s", args, err, b)
		}
	}
	if b, err := exec.Command(filepath.Join(consumer, "consumer")).Output(); err != nil || string(b) != "z-if" {
		t.Fatalf("consumer: %q %v", b, err)
	}
}

// stage publishes a family of three (interface <- service <- system) in
// dependency order into <root>/.engmod/staging without touching any
// checkout; with --serve the printed CUE_REGISTRY lets the system compose
// with the workspace off from the staged registry alone (no replacements)
// and its committed lock check pass, and the printed GOPROXY lets an outside
// consumer go get and build the staged Go module offline. The staged CUE
// module holds committed files only.
// Mutation: dropping the dependency sort, writing no Go proxy list, routing
// CUE_REGISTRY to a wrong host, or zipping ignored files fails this test.
// TRLC-LINKS: REQ-EMG-080
func TestStageServesFamilyForCompositionAndGo(t *testing.T) {
	fam := newStageFamily(t)
	before := fam.gitStatuses(t)
	t.Setenv("CUE_REGISTRY", "")
	t.Setenv("GONOSUMDB", "")
	t.Setenv("GOPRIVATE", "")

	serve := startStageServe(t, fam.root)
	defer serve.cancel()
	out := serve.stdout.String()
	requireStagedInOrder(t, out)
	requireUnchanged(t, before)
	printed := stagePrinted{out: out, staging: filepath.Join(fam.root, ".engmod", "staging"), env: envLines(out)}
	printed.requireLayout(t)
	printed.requireEnv(t)
	requireComposeFromStaging(t, fam.sys, printed.env)
	requireGoConsumer(t, printed.env)
	serve.stop(t)
}

// requireStageSkewWarning requires a plain stage to succeed, warn about the
// consumer pinning another version and print the Go env only.
func requireStageSkewWarning(t *testing.T, root string) {
	t.Helper()
	code, out, errOut := runMain(t, fakeRegistry{unknown: true}, "--root", root, "stage")
	if code != 0 || !strings.Contains(out, "warning: svc pins ia ("+modulePathOf("if-a")+") at v0.0.9 but if-a stages v0.1.0") {
		t.Fatalf("stage: %d\n%s\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "engws stage --serve") || strings.Contains(out, "CUE_REGISTRY=") {
		t.Fatalf("without --serve the CUE env needs the server:\n%s", out)
	}
}

// requireStageRefusesUntagged requires --version-from tag to refuse a
// checkout whose HEAD lacks the tag.
func requireStageRefusesUntagged(t *testing.T, root string) {
	t.Helper()
	code, out, errOut := runMain(t, fakeRegistry{unknown: true}, "--root", root, "stage", "--version-from", "tag")
	if code != 1 || !strings.Contains(out, "FAIL if-a: HEAD is not at tag v0.1.0") {
		t.Fatalf("untagged: %d\n%s\n%s", code, out, errOut)
	}
}

// requireStageRefusesDirty requires stage --serve to refuse a dirty
// checkout and serve nothing.
func requireStageRefusesDirty(t *testing.T, root string) {
	t.Helper()
	// --serve would block if staging went ahead; the deadline stops it.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var serveOut, serveErr bytes.Buffer
	code := Main(ctx, []string{"--root", root, "stage", "--serve"}, CLI{Stdout: &serveOut, Stderr: &serveErr, Registry: fakeRegistry{unknown: true}})
	out, errOut := serveOut.String(), serveErr.String()
	if code != 1 || !strings.Contains(out, "FAIL if-a: working tree has changes") || strings.Contains(out, "engws: serving") {
		t.Fatalf("dirty: %d\n%s\n%s", code, out, errOut)
	}
}

// stage refuses a checkout cue mod publish would refuse (dirty tree) or,
// with --version-from tag, one whose HEAD lacks tag v<module.version>; it
// reports each failing checkout, serves nothing and exits 1. It warns when a
// consumer pins a family module at another version than the staged one.
// Mutation: skipping the clean-tree check, or ignoring --version-from tag,
// fails this test.
// TRLC-LINKS: REQ-EMG-080
func TestStageRefusesDirtyOrUntaggedCheckouts(t *testing.T) {
	root := t.TempDir()
	ifA := stageCheckout(t, stageSpec{Root: root, Name: "if-a"})
	stageCheckout(t, stageSpec{Root: root, Name: "svc", Deps: []dep{{"ia", "if-a", "v0.0.9"}}})
	requireStageSkewWarning(t, root)

	runGit(t, ifA, "commit", "-q", "--allow-empty", "-m", "after tag")
	requireStageRefusesUntagged(t, root)

	write(t, filepath.Join(ifA, "model", "notes.txt"), "uncommitted\n")
	requireStageRefusesDirty(t, root)
	if code, _, errOut := runMain(t, fakeRegistry{unknown: true}, "--root", root, "stage", "--version-from", "branch"); code != 2 || !strings.Contains(errOut, "--version-from") {
		t.Fatalf("bad --version-from: %d %s", code, errOut)
	}
}

// The printed environment keeps the user's other registry routes and
// private module patterns: CUE_REGISTRY adds one route per staged module and
// drops an existing route for the same prefix, and GONOSUMDB keeps
// GOPRIVATE so no private module name reaches the checksum database.
// Mutation: replacing CUE_REGISTRY instead of merging, or dropping GOPRIVATE
// from GONOSUMDB, fails this test.
// TRLC-LINKS: REQ-EMG-080
func TestStageEnvMergesExistingSettings(t *testing.T) {
	result := StageResult{GoProxyDir: "/w/.engmod/staging/goproxy", Modules: []StagedModule{
		{Name: "if-a", ModulePath: "example.com/fam/if-a@v0", Version: "v0.1.0", GoModule: "example.com/go/if-a"},
		{Name: "svc", ModulePath: "example.com/fam/svc@v0", Version: "v0.1.0"},
	}}
	env := StageEnv(result, "127.0.0.1:5000", map[string]string{
		"CUE_REGISTRY": "example.com/fam/svc=ghcr.io/old,example.com/other=r.example.com,registry.example.com",
		"GOPRIVATE":    "corp.example.com/*",
	})
	got := strings.Join(env, "\n")
	for _, want := range []string{
		"CUE_REGISTRY=example.com/fam/if-a=127.0.0.1:5000+insecure,example.com/fam/svc=127.0.0.1:5000+insecure,example.com/other=r.example.com,registry.example.com",
		"GOPROXY=file:///w/.engmod/staging/goproxy,off",
		"GONOPROXY=none",
		"GONOSUMDB=example.com/go/if-a,corp.example.com/*",
	} {
		if !strings.Contains(got, want+"\n") && !strings.HasSuffix(got, want) {
			t.Fatalf("missing %q in\n%s", want, got)
		}
	}
	if strings.Contains(got, "ghcr.io/old") {
		t.Fatalf("stale route kept:\n%s", got)
	}
	if noHost := strings.Join(StageEnv(result, "", nil), "\n"); strings.Contains(noHost, "CUE_REGISTRY=") || !strings.Contains(noHost, "GOPROXY=") {
		t.Fatalf("without a server only the Go env is printed:\n%s", noHost)
	}
}

// The staging registry has no authentication, so it serves on a loopback
// address only; another address is refused before anything listens.
// Mutation: dropping the loopback check fails this test (the serve would
// start on 0.0.0.0 and block until the deadline).
// TRLC-LINKS: REQ-EMG-080
func TestServeStagingLoopbackOnly(t *testing.T) {
	ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
	defer cancel()
	started := false
	err := ServeStaging(ctx, StageResult{}, ServeOptions{Addr: "0.0.0.0:0", Ready: func(string) { started = true }})
	if err == nil || started || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("non-loopback address: started=%v err=%v", started, err)
	}
}
