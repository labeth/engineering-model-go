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

// stageCheckout writes a model whose CUE module has source kind git, an
// optional Go module, a git-ignored scratch file, commits and tags v0.1.0.
func stageCheckout(t *testing.T, root, name, goModule string, deps ...dep) string {
	t.Helper()
	dir := filepath.Join(root, name)
	writeModel(t, dir, name, "v0.1.0", deps...)
	write(t, filepath.Join(dir, "cue.mod", "module.cue"), "module: \""+modulePathOf(name)+"\"\nlanguage: version: \"v0.17.0\"\nsource: kind: \"git\"\n")
	write(t, filepath.Join(dir, ".gitignore"), ".engmod/\nengmod.lock.local.yml\nscratch.txt\n")
	write(t, filepath.Join(dir, "scratch.txt"), "not committed\n")
	if goModule != "" {
		write(t, filepath.Join(dir, "go.mod"), "module "+goModule+"\n\ngo 1.21\n")
		write(t, filepath.Join(dir, "name.go"), "package lib\n\n// Name names the module.\nfunc Name() string { return \""+name+"\" }\n")
	}
	runGit(t, dir, "init", "-q")
	runGit(t, dir, "add", ".")
	runGit(t, dir, "commit", "-q", "-m", "init")
	runGit(t, dir, "tag", "v0.1.0")
	return dir
}

// envLines parses the KEY=value lines engws prints.
func envLines(out string) map[string]string {
	env := map[string]string{}
	scanner := bufio.NewScanner(strings.NewReader(out))
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		key, value, ok := strings.Cut(line, "=")
		if ok && key != "" && key == strings.ToUpper(key) && !strings.ContainsAny(key, " :") {
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
	root := t.TempDir()
	ifZ := stageCheckout(t, root, "z-if", "example.com/go/z-if")
	svc := stageCheckout(t, root, "m-svc", "", dep{"ia", "z-if", "v0.1.0"})
	sys := stageCheckout(t, root, "a-sys", "", dep{"sv", "m-svc", "v0.1.0"})
	// A workspace that replaces every checkout; ENGMOD_WORK=off must ignore it.
	if code, out, errOut := runMain(t, allPublished("z-if", "m-svc", "a-sys"), "--root", root, "sync", "--all"); code != 0 {
		t.Fatalf("sync: %d\n%s\n%s", code, out, errOut)
	}
	before := map[string]string{}
	for _, dir := range []string{ifZ, svc, sys} {
		before[dir] = runGit(t, dir, "status", "--porcelain", "--ignored")
	}
	t.Setenv("CUE_REGISTRY", "")
	t.Setenv("GONOSUMDB", "")
	t.Setenv("GOPRIVATE", "")

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()
	var stdout, stderr lockedBuffer
	done := make(chan int, 1)
	go func() {
		done <- Main(ctx, []string{"--root", root, "stage", "--serve"}, &stdout, &stderr, fakeRegistry{unknown: true})
	}()
	deadline := time.Now().Add(60 * time.Second)
	for !strings.Contains(stdout.String(), "engws: serving") {
		select {
		case code := <-done:
			t.Fatalf("stage exited %d before serving\n%s\n%s", code, stdout.String(), stderr.String())
		default:
		}
		if time.Now().After(deadline) {
			t.Fatalf("stage --serve did not start\n%s\n%s", stdout.String(), stderr.String())
		}
		time.Sleep(20 * time.Millisecond)
	}
	out := stdout.String()
	iA, iS, iY := strings.Index(out, "staged z-if "), strings.Index(out, "staged m-svc "), strings.Index(out, "staged a-sys ")
	if iA < 0 || iS < 0 || iY < 0 || !(iA < iS && iS < iY) {
		t.Fatalf("not staged in dependency order:\n%s", out)
	}
	for dir, status := range before {
		if got := runGit(t, dir, "status", "--porcelain", "--ignored"); got != status {
			t.Fatalf("stage wrote inside %s:\n%s", dir, got)
		}
	}
	staging := filepath.Join(root, ".engmod", "staging")
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
	env := envLines(out)
	if !strings.Contains(env["CUE_REGISTRY"], testFamily+"/z-if=127.0.0.1:") || !strings.Contains(env["CUE_REGISTRY"], testFamily+"/a-sys=") {
		t.Fatalf("CUE_REGISTRY: %q\n%s", env["CUE_REGISTRY"], out)
	}
	if env["GOPROXY"] != "file://"+filepath.ToSlash(filepath.Join(staging, "goproxy"))+",off" || env["GONOPROXY"] != "none" || !strings.Contains(env["GONOSUMDB"], "example.com/go/z-if") {
		t.Fatalf("Go env: %v\n%s", env, out)
	}

	// Composition from the staged registry alone.
	t.Setenv("CUE_REGISTRY", env["CUE_REGISTRY"])
	manifest := filepath.Join(sys, "engmod.yml")
	res, err := composition.GenerateCompositionFromFileWith(manifest, composition.ResolutionOptions{Workspace: "off"})
	if err != nil {
		t.Fatal(err)
	}
	noErrors(t, "compose", res.Diagnostics)
	if res.Root == nil || len(res.Root.Children) != 1 || len(res.Root.Children[0].Children) != 1 {
		t.Fatalf("composition tree: %#v", res.Root)
	}
	cache := filepath.Join(sys, ".engmod", "modules")
	for _, child := range []string{res.Root.Children[0].Dir, res.Root.Children[0].Children[0].Dir} {
		if !strings.HasPrefix(child, cache) {
			t.Fatalf("dependency resolved from %s, not the staged registry", child)
		}
		if _, err := os.Stat(filepath.Join(child, "scratch.txt")); err == nil {
			t.Fatalf("staged module %s holds an uncommitted file", child)
		}
	}
	res, err = composition.GenerateCompositionFromFileWith(manifest, composition.ResolutionOptions{Workspace: "off", Lock: "check"})
	if err != nil {
		t.Fatal(err)
	}
	noErrors(t, "lock check", res.Diagnostics)

	// Go: an outside consumer resolves the staged module through GOPROXY.
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

	cancel()
	select {
	case code := <-done:
		if code != 0 {
			t.Fatalf("serve exit %d\n%s", code, stderr.String())
		}
	case <-time.After(10 * time.Second):
		t.Fatal("serve did not stop on cancel")
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
	ifA := stageCheckout(t, root, "if-a", "")
	stageCheckout(t, root, "svc", "", dep{"ia", "if-a", "v0.0.9"})
	code, out, errOut := runMain(t, fakeRegistry{unknown: true}, "--root", root, "stage")
	if code != 0 || !strings.Contains(out, "warning: svc pins ia ("+modulePathOf("if-a")+") at v0.0.9 but if-a stages v0.1.0") {
		t.Fatalf("stage: %d\n%s\n%s", code, out, errOut)
	}
	if !strings.Contains(out, "engws stage --serve") || strings.Contains(out, "CUE_REGISTRY=") {
		t.Fatalf("without --serve the CUE env needs the server:\n%s", out)
	}

	runGit(t, ifA, "commit", "-q", "--allow-empty", "-m", "after tag")
	code, out, errOut = runMain(t, fakeRegistry{unknown: true}, "--root", root, "stage", "--version-from", "tag")
	if code != 1 || !strings.Contains(out, "FAIL if-a: HEAD is not at tag v0.1.0") {
		t.Fatalf("untagged: %d\n%s\n%s", code, out, errOut)
	}

	write(t, filepath.Join(ifA, "model", "notes.txt"), "uncommitted\n")
	// --serve would block if staging went ahead; the deadline stops it.
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	var serveOut, serveErr bytes.Buffer
	code = Main(ctx, []string{"--root", root, "stage", "--serve"}, &serveOut, &serveErr, fakeRegistry{unknown: true})
	out, errOut = serveOut.String(), serveErr.String()
	if code != 1 || !strings.Contains(out, "FAIL if-a: working tree has changes") || strings.Contains(out, "engws: serving") {
		t.Fatalf("dirty: %d\n%s\n%s", code, out, errOut)
	}
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
	err := ServeStaging(ctx, StageResult{}, "0.0.0.0:0", func(string) { started = true })
	if err == nil || started || !strings.Contains(err.Error(), "loopback") {
		t.Fatalf("non-loopback address: started=%v err=%v", started, err)
	}
}
