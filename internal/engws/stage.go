// ENGMODEL-OWNER-UNIT: FU-CLI-ORCHESTRATION
package engws

import (
	"archive/zip"
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"

	"cuelabs.dev/go/oci/ociregistry"
	"cuelabs.dev/go/oci/ociregistry/ocifilter"
	"cuelabs.dev/go/oci/ociregistry/ocimem"
	"cuelabs.dev/go/oci/ociregistry/ociserver"
	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/modregistry"
	cuemodule "cuelang.org/go/mod/module"
	"cuelang.org/go/mod/modzip"
	gomodule "golang.org/x/mod/module"
	gozip "golang.org/x/mod/zip"
)

const (
	// StagingDir is the staging area under the root, owned by engws stage.
	StagingDir = ".engmod/staging"
	// VersionFromManifest stages module.version from engmod.yml (default).
	VersionFromManifest = "manifest"
	// VersionFromTag additionally requires HEAD at tag v<module.version>.
	VersionFromTag = "tag"

	ociRefNameAnnotation = "org.opencontainers.image.ref.name"
)

// StagedModule is one checkout published into the staging area.
type StagedModule struct {
	Name       string // checkout directory name
	ModulePath string // CUE module path with major version
	Version    string
	CUEFiles   int
	LayoutDir  string // OCI image layout of the CUE module
	GoModule   string // staged Go module path, empty without go.mod
	Notes      []string
}

// StageResult is the outcome of Stage.
type StageResult struct {
	Root        string
	RegistryDir string
	GoProxyDir  string
	Modules     []StagedModule // in dependency order
	Warnings    []string
	// Failures name the checkouts that cannot be staged; when any exist
	// nothing is written.
	Failures []string
}

// stageCandidate is a checkout with the facts Stage checks before writing.
type stageCandidate struct {
	checkout   Checkout
	version    string
	mv         cuemodule.Version
	source     string
	revision   string
	commitTime time.Time
	files      []string
	notes      []string
}

// Stage publishes every checkout under root, in dependency order, into
// root/.engmod/staging: the CUE module as cue mod publish would (an OCI
// image layout per module under registry/<module path>, source kind git
// zips committed files only, kind self the directory) and, for a checkout
// with go.mod at its top, the Go module in GOPROXY file layout under
// goproxy/<module>/@v/<version>.{info,mod,zip} with list. It refuses a
// checkout cue mod publish would refuse (dirty tree, no source, module path
// mismatch) and, with versionFrom tag, one whose HEAD lacks tag
// v<module.version>; then nothing is written. It never writes inside a
// checkout.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func Stage(ctx context.Context, root, versionFrom string) (StageResult, error) {
	staging := filepath.Join(root, filepath.FromSlash(StagingDir))
	result := StageResult{Root: root, RegistryDir: filepath.Join(staging, "registry"), GoProxyDir: filepath.Join(staging, "goproxy")}
	checkouts, warnings, err := ScanWithWarnings(root)
	if err != nil {
		return result, err
	}
	result.Warnings = append(result.Warnings, warnings...)
	ordered, err := dependencyOrder(checkouts)
	if err != nil {
		return result, err
	}
	candidates, failures := stagePrepareAll(ctx, ordered, versionFrom)
	result.Failures = append(result.Failures, failures...)
	result.Warnings = append(result.Warnings, stageSkews(ordered)...)
	if len(result.Failures) > 0 {
		return result, nil
	}
	if err := os.RemoveAll(staging); err != nil {
		return result, err
	}
	if err := stageWriteAll(ctx, candidates, &result); err != nil {
		return result, err
	}
	return result, writeGoProxyLists(result.GoProxyDir)
}

// stagePrepareAll prepares every checkout and collects the candidates and
// the failure line of each checkout that cannot be staged.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func stagePrepareAll(ctx context.Context, ordered []Checkout, versionFrom string) (candidates []stageCandidate, failures []string) {
	for _, checkout := range ordered {
		candidate, err := prepareStage(ctx, checkout, versionFrom)
		if err != nil {
			failures = append(failures, fmt.Sprintf("%s: %v", checkout.Name, err))
			continue
		}
		candidates = append(candidates, candidate)
	}
	return candidates, failures
}

// stageWriteAll writes every candidate into the staging area and appends
// the staged modules to result.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func stageWriteAll(ctx context.Context, candidates []stageCandidate, result *StageResult) error {
	dirs := stageDirs{registry: result.RegistryDir, goProxy: result.GoProxyDir}
	for _, c := range candidates {
		staged, err := dirs.write(ctx, c)
		if err != nil {
			return fmt.Errorf("%s: %w", c.checkout.Name, err)
		}
		result.Modules = append(result.Modules, staged)
	}
	return nil
}

// stageOrderer is the depth-first walk of dependencyOrder.
type stageOrderer struct {
	checkouts []Checkout
	byModule  map[string]int
	state     []int // 0 new, 1 visiting, 2 done
	out       []Checkout
}

// dependencyOrder sorts checkouts so every checkout follows the checkouts
// whose module it depends on; ties keep name order. A cycle is an error.
// TRLC-LINKS: REQ-EMG-080
func dependencyOrder(checkouts []Checkout) ([]Checkout, error) {
	o := &stageOrderer{checkouts: checkouts, byModule: map[string]int{}, state: make([]int, len(checkouts))}
	for i, c := range checkouts {
		o.byModule[c.Manifest.Module.Path] = i
	}
	for i := range checkouts {
		if err := o.visit(i, nil); err != nil {
			return nil, err
		}
	}
	return o.out, nil
}

// visit appends checkout i after the checkouts it depends on.
// TRLC-LINKS: REQ-EMG-080
func (o *stageOrderer) visit(i int, path []string) error {
	switch o.state[i] {
	case 1:
		return fmt.Errorf("dependency cycle: %s", strings.Join(append(path, o.checkouts[i].Name), " -> "))
	case 2:
		return nil
	}
	o.state[i] = 1
	for _, j := range o.dependencies(i) {
		if err := o.visit(j, append(path, o.checkouts[i].Name)); err != nil {
			return err
		}
	}
	o.state[i] = 2
	o.out = append(o.out, o.checkouts[i])
	return nil
}

// dependencies returns the indexes of the checkouts checkout i depends on,
// in order.
// TRLC-LINKS: REQ-EMG-080
func (o *stageOrderer) dependencies(i int) []int {
	deps := o.checkouts[i].Manifest.Dependencies
	names := make([]int, 0, len(deps))
	for _, d := range deps {
		if j, ok := o.byModule[strings.TrimSpace(d.Path)]; ok {
			names = append(names, j)
		}
	}
	sort.Ints(names)
	return names
}

// stageSkews warns when a checkout pins a family module (model dependency
// or go.mod require) at another version than the one staged for it.
// TRLC-LINKS: REQ-EMG-080
func stageSkews(checkouts []Checkout) []string {
	byModule, byGo := map[string]Checkout{}, map[string]Checkout{}
	for _, c := range checkouts {
		byModule[c.Manifest.Module.Path] = c
		if c.GoModule != "" {
			byGo[c.GoModule] = c
		}
	}
	var out []string
	for _, c := range checkouts {
		out = append(out, stageModelSkews(c, byModule)...)
		out = append(out, stageGoSkews(c, byGo)...)
	}
	sort.Strings(out)
	return out
}

// stageModelSkews lists the dependencies of c pinned at another version
// than the one staged.
// TRLC-LINKS: REQ-EMG-080
func stageModelSkews(c Checkout, byModule map[string]Checkout) []string {
	var out []string
	for _, d := range c.Manifest.Dependencies {
		if dep, ok := byModule[strings.TrimSpace(d.Path)]; ok && tagFor(d.Version) != tagFor(dep.Manifest.Module.Version) {
			out = append(out, fmt.Sprintf("%s pins %s (%s) at %s but %s stages %s", c.Name, d.Alias, d.Path, d.Version, dep.Name, tagFor(dep.Manifest.Module.Version)))
		}
	}
	return out
}

// stageGoSkews lists the go.mod requires of c at another version than the
// one staged.
// TRLC-LINKS: REQ-EMG-080
func stageGoSkews(c Checkout, byGo map[string]Checkout) []string {
	var out []string
	for path, version := range c.GoRequires {
		if dep, ok := byGo[path]; ok && version != tagFor(dep.Manifest.Module.Version) {
			out = append(out, fmt.Sprintf("%s go.mod requires %s %s but %s stages %s", c.Name, path, version, dep.Name, tagFor(dep.Manifest.Module.Version)))
		}
	}
	return out
}

// prepareStage checks a checkout the way cue mod publish does and collects
// its committed files, revision and commit time.
// TRLC-LINKS: REQ-EMG-080
func prepareStage(_ context.Context, checkout Checkout, versionFrom string) (stageCandidate, error) {
	c := stageCandidate{checkout: checkout, version: tagFor(checkout.Manifest.Module.Version)}
	steps := []func() error{
		func() error { return c.checkTree(versionFrom) },
		c.readModule,
		c.readHistory,
		c.checkGoModule,
	}
	for _, step := range steps {
		if err := step(); err != nil {
			return c, err
		}
	}
	return c, nil
}

// checkTree refuses a checkout outside its own repository, with a dirty
// tree, or (with versionFrom tag) without the tag at HEAD.
// TRLC-LINKS: REQ-EMG-080
func (c *stageCandidate) checkTree(versionFrom string) error {
	dir := c.checkout.Dir
	if own, why := ownGitRepository(dir); !own {
		return errors.New(why)
	}
	out, err := git(dir, "status", "--porcelain")
	if err != nil {
		return fmt.Errorf("git status: %w", err)
	}
	if out != "" {
		return errors.New("working tree has changes; commit them first (cue mod publish refuses a dirty tree)")
	}
	tags, _ := git(dir, "tag", "--points-at", "HEAD")
	if containsLine(tags, c.version) {
		return nil
	}
	if versionFrom == VersionFromTag {
		return fmt.Errorf("HEAD is not at tag %s", c.version)
	}
	c.notes = append(c.notes, "HEAD is not at tag "+c.version+"; staged from HEAD")
	return nil
}

// readModule reads the module version and checks cue.mod/module.cue
// against the manifest.
// TRLC-LINKS: REQ-EMG-080
func (c *stageCandidate) readModule() error {
	mv, err := cuemodule.NewVersion(c.checkout.Manifest.Module.Path, c.version)
	if err != nil {
		return err
	}
	c.mv = mv
	modPath := filepath.Join(c.checkout.Dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(modPath)
	if err != nil {
		return err
	}
	mf, err := modfile.Parse(data, modPath)
	if err != nil {
		return err
	}
	if mf.QualifiedModule() != mv.Path() {
		return fmt.Errorf("cue.mod/module.cue declares %s, engmod.yml %s", mf.QualifiedModule(), mv.Path())
	}
	if mf.Source == nil {
		return errors.New("cue.mod/module.cue has no source field (cue mod edit --source git)")
	}
	c.source = mf.Source.Kind
	return nil
}

// readHistory reads the HEAD revision, its commit time and the committed
// files.
// TRLC-LINKS: REQ-EMG-080
func (c *stageCandidate) readHistory() error {
	dir := c.checkout.Dir
	out, err := git(dir, "-c", "log.showsignature=false", "log", "-1", "--format=%H:%ct")
	if err != nil {
		return fmt.Errorf("git log: %w", err)
	}
	rev, secs, _ := strings.Cut(out, ":")
	unix, err := strconv.ParseInt(secs, 10, 64)
	if err != nil {
		return fmt.Errorf("git log: unexpected output %q", out)
	}
	c.revision, c.commitTime = rev, time.Unix(unix, 0).UTC()
	files, err := git(dir, "ls-files", "-z", "--")
	if err != nil {
		return fmt.Errorf("git ls-files: %w", err)
	}
	for _, f := range strings.Split(strings.TrimSuffix(files, "\x00"), "\x00") {
		if f != "" {
			c.files = append(c.files, f)
		}
	}
	sort.Strings(c.files)
	return nil
}

// checkGoModule checks the Go module path and version of a checkout with
// go.mod.
// TRLC-LINKS: REQ-EMG-080
func (c *stageCandidate) checkGoModule() error {
	if c.checkout.GoModule == "" {
		return nil
	}
	if err := gomodule.Check(c.checkout.GoModule, c.version); err != nil {
		return fmt.Errorf("go module: %w", err)
	}
	return nil
}

// stageDirs are the registry and Go proxy directories of the staging area.
type stageDirs struct{ registry, goProxy string }

// stageZip is the CUE module zip with the metadata it is published with.
type stageZip struct {
	data []byte
	meta *modregistry.Metadata
}

// write writes the CUE module layout and the Go proxy files of c.
// TRLC-LINKS: REQ-EMG-080
func (d stageDirs) write(ctx context.Context, c stageCandidate) (StagedModule, error) {
	staged := StagedModule{Name: c.checkout.Name, ModulePath: c.mv.Path(), Version: c.version, Notes: c.notes,
		LayoutDir: filepath.Join(d.registry, filepath.FromSlash(c.mv.BasePath()))}
	z, err := stageCUEZip(c)
	if err != nil {
		return staged, err
	}
	zr, err := zip.NewReader(bytes.NewReader(z.data), int64(len(z.data)))
	if err != nil {
		return staged, err
	}
	staged.CUEFiles = len(zr.File)
	if err := z.put(ctx, c, staged.LayoutDir); err != nil {
		return staged, err
	}
	if c.checkout.GoModule != "" {
		if err := stageGoModule(c, d.goProxy); err != nil {
			return staged, err
		}
		staged.GoModule = c.checkout.GoModule
	}
	return staged, nil
}

// stageCUEZip zips the CUE module of c: the directory for source kind self,
// else the committed files with the commit as metadata.
// TRLC-LINKS: REQ-EMG-080
func stageCUEZip(c stageCandidate) (stageZip, error) {
	var buf bytes.Buffer
	if c.source == "self" {
		err := modzip.CreateFromDir(&buf, c.mv, c.checkout.Dir)
		return stageZip{data: buf.Bytes()}, err
	}
	if err := modzip.Create(&buf, c.mv, c.files, dirFileIO{root: c.checkout.Dir}); err != nil {
		return stageZip{}, err
	}
	meta := &modregistry.Metadata{VCSType: c.source, VCSCommit: c.revision, VCSCommitTime: c.commitTime}
	return stageZip{data: buf.Bytes(), meta: meta}, nil
}

// put publishes the zip of c into the OCI layout at layoutDir.
// TRLC-LINKS: REQ-EMG-080
func (z stageZip) put(ctx context.Context, c stageCandidate, layoutDir string) error {
	layout := &ociLayoutWriter{dir: layoutDir}
	client := modregistry.NewClientWithResolver(layoutResolver{layout})
	if err := client.PutModuleWithMetadata(ctx, c.mv, bytes.NewReader(z.data), int64(len(z.data)), z.meta); err != nil {
		return fmt.Errorf("put CUE module: %w", err)
	}
	return layout.writeIndex()
}

// stageGoModule writes <module>/@v/<version>.{info,mod,zip} from the
// committed files, as the go command's direct VCS mode would zip the tag.
// TRLC-LINKS: REQ-EMG-080
func stageGoModule(c stageCandidate, goProxyDir string) error {
	escPath, err := gomodule.EscapePath(c.checkout.GoModule)
	if err != nil {
		return err
	}
	escVersion, err := gomodule.EscapeVersion(c.version)
	if err != nil {
		return err
	}
	at := filepath.Join(goProxyDir, filepath.FromSlash(escPath), "@v")
	if err := os.MkdirAll(at, 0o755); err != nil {
		return err
	}
	goMod, err := os.ReadFile(filepath.Join(c.checkout.Dir, "go.mod"))
	if err != nil {
		return err
	}
	goZip, err := stageGoZip(c)
	if err != nil {
		return err
	}
	info, _ := json.Marshal(struct {
		Version string
		Time    time.Time
	}{c.version, c.commitTime})
	return stageWriteFiles(at, map[string][]byte{escVersion + ".info": info, escVersion + ".mod": goMod, escVersion + ".zip": goZip})
}

// stageGoZip zips the committed files of c as its Go module.
// TRLC-LINKS: REQ-EMG-080
func stageGoZip(c stageCandidate) ([]byte, error) {
	var files []gozip.File
	for _, f := range c.files {
		files = append(files, goZipFile{root: c.checkout.Dir, path: f})
	}
	var buf bytes.Buffer
	if err := gozip.Create(&buf, gomodule.Version{Path: c.checkout.GoModule, Version: c.version}, files); err != nil {
		return nil, fmt.Errorf("go module zip: %w", err)
	}
	return buf.Bytes(), nil
}

// stageWriteFiles writes each named file of files into dir.
// TRLC-LINKS: REQ-EMG-080
func stageWriteFiles(dir string, files map[string][]byte) error {
	for name, data := range files {
		if err := os.WriteFile(filepath.Join(dir, name), data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

// writeGoProxyLists writes the list file of every staged Go module.
// TRLC-LINKS: REQ-EMG-080
func writeGoProxyLists(goProxyDir string) error {
	err := filepath.WalkDir(goProxyDir, stageListWalk)
	if errors.Is(err, os.ErrNotExist) {
		return nil
	}
	return err
}

// stageListWalk writes the list file of every @v directory it visits.
// TRLC-LINKS: REQ-EMG-080
func stageListWalk(path string, d os.DirEntry, err error) error {
	if err != nil || !d.IsDir() || d.Name() != "@v" {
		return err
	}
	entries, err := os.ReadDir(path)
	if err != nil {
		return err
	}
	var versions []string
	for _, e := range entries {
		if v, ok := strings.CutSuffix(e.Name(), ".info"); ok {
			versions = append(versions, v)
		}
	}
	return os.WriteFile(filepath.Join(path, "list"), []byte(strings.Join(versions, "\n")+"\n"), 0o644)
}

// StageEnv returns the environment to run the gate against the staging
// area: CUE_REGISTRY routes every staged module to host (merged with the
// current CUE_REGISTRY, replacing routes for the same prefix; omitted when
// host is empty), GOPROXY is the file proxy with no fallback, GONOPROXY=none
// makes private patterns use it, GONOSUMDB names the staged Go modules plus
// the current GONOSUMDB or GOPRIVATE, and ENGMOD_WORK=off with GOWORK=off
// make every dependency resolve from staging instead of the checkouts.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, REF-CUE-OCI-MODULE-REGISTRY
func StageEnv(result StageResult, host string, current map[string]string) []string {
	var env []string
	if host != "" {
		env = append(env, "CUE_REGISTRY="+stageCUERoutes(result, host, current))
	}
	env = append(env, "GOPROXY=file://"+filepath.ToSlash(result.GoProxyDir)+",off", "GONOPROXY=none")
	if noSum := stageNoSumDB(result, current); len(noSum) > 0 {
		env = append(env, "GONOSUMDB="+strings.Join(noSum, ","))
	}
	return append(env, "GOWORK=off", "ENGMOD_WORK=off")
}

// stageCUERoutes returns the CUE_REGISTRY value: a route to host for every
// staged module, then the routes of current that name another prefix.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, REF-CUE-OCI-MODULE-REGISTRY
func stageCUERoutes(result StageResult, host string, current map[string]string) string {
	var routes []string
	staged := map[string]bool{}
	for _, m := range result.Modules {
		base, _, _ := strings.Cut(m.ModulePath, "@")
		staged[base] = true
		routes = append(routes, base+"="+host+"+insecure")
	}
	for _, part := range strings.Split(current["CUE_REGISTRY"], ",") {
		if part = strings.TrimSpace(part); stageKeepRoute(part, staged) {
			routes = append(routes, part)
		}
	}
	return strings.Join(routes, ",")
}

// stageKeepRoute reports whether a CUE_REGISTRY part is kept: not empty and
// not a route for a staged prefix.
// TRLC-LINKS: REQ-EMG-080
func stageKeepRoute(part string, staged map[string]bool) bool {
	key, _, isRoute := strings.Cut(part, "=")
	return part != "" && !(isRoute && staged[key])
}

// stageNoSumDB returns the staged Go modules plus the current GONOSUMDB, or
// GOPRIVATE when that is empty.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, REF-CUE-OCI-MODULE-REGISTRY
func stageNoSumDB(result StageResult, current map[string]string) []string {
	var noSum []string
	for _, m := range result.Modules {
		if m.GoModule != "" {
			noSum = append(noSum, m.GoModule)
		}
	}
	if extra := strings.TrimSpace(current["GONOSUMDB"]); extra != "" {
		noSum = append(noSum, extra)
	} else if extra := strings.TrimSpace(current["GOPRIVATE"]); extra != "" {
		noSum = append(noSum, extra)
	}
	return noSum
}

// ServeOptions says where ServeStaging listens and who is told.
type ServeOptions struct {
	// Addr is the loopback listen address.
	Addr string
	// Ready is called with the listening host:port before serving.
	Ready func(host string)
}

// ServeStaging loads every staged OCI layout into an in-memory read-only
// registry and serves it on opts.Addr (loopback only) until ctx is done.
// opts.Ready is called with the listening host:port before serving.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, REF-CUE-OCI-MODULE-REGISTRY
func ServeStaging(ctx context.Context, result StageResult, opts ServeOptions) error {
	if err := stageCheckLoopback(opts.Addr); err != nil {
		return err
	}
	mem := ocimem.NewWithConfig(&ocimem.Config{ImmutableTags: true})
	for _, m := range result.Modules {
		base, _, _ := strings.Cut(m.ModulePath, "@")
		if err := loadLayout(ctx, mem, ociLayoutRef{Dir: m.LayoutDir, Repo: base}); err != nil {
			return fmt.Errorf("load %s: %w", m.LayoutDir, err)
		}
	}
	listener, err := net.Listen("tcp", opts.Addr)
	if err != nil {
		return err
	}
	server := &http.Server{Handler: ociserver.New(ocifilter.ReadOnly(mem), nil), ReadHeaderTimeout: 10 * time.Second}
	served := make(chan error, 1)
	go func() { served <- server.Serve(listener) }()
	opts.Ready(listener.Addr().String())
	return stageAwaitServer(ctx, server, served)
}

// stageCheckLoopback refuses an address that is not a loopback address.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, REF-CUE-OCI-MODULE-REGISTRY
func stageCheckLoopback(addr string) error {
	host, _, err := net.SplitHostPort(addr)
	if err != nil {
		return err
	}
	if ip := net.ParseIP(host); host != "localhost" && (ip == nil || !ip.IsLoopback()) {
		return fmt.Errorf("--addr %s: the staging registry serves on a loopback address only", addr)
	}
	return nil
}

// stageAwaitServer waits until the server stops or ctx is done, then shuts
// it down.
// TRLC-LINKS: REQ-EMG-080
// ENGMODEL-LINKS: FU-CLI-ORCHESTRATION, REF-CUE-OCI-MODULE-REGISTRY
func stageAwaitServer(ctx context.Context, server *http.Server, served <-chan error) error {
	select {
	case err := <-served:
		return err
	case <-ctx.Done():
	}
	shutdown, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return server.Shutdown(shutdown)
}

// ociIndex is the index.json of an OCI image layout.
type ociIndex struct {
	SchemaVersion int                      `json:"schemaVersion"`
	MediaType     string                   `json:"mediaType"`
	Manifests     []ociregistry.Descriptor `json:"manifests"`
}

// ociLayoutRef names an OCI layout directory and the repository it loads
// into.
type ociLayoutRef struct{ Dir, Repo string }

// layoutLoader pushes the content of one OCI layout into a registry.
type layoutLoader struct {
	r   ociregistry.Interface
	ref ociLayoutRef
}

// loadLayout pushes every tagged manifest of an OCI layout, with its
// config and layers, into the repository of ref in r.
// TRLC-LINKS: REQ-EMG-080
func loadLayout(ctx context.Context, r ociregistry.Interface, ref ociLayoutRef) error {
	data, err := os.ReadFile(filepath.Join(ref.Dir, "index.json"))
	if err != nil {
		return err
	}
	var index ociIndex
	if err := json.Unmarshal(data, &index); err != nil {
		return err
	}
	l := layoutLoader{r: r, ref: ref}
	for _, desc := range index.Manifests {
		if err := l.manifest(ctx, desc); err != nil {
			return err
		}
	}
	return nil
}

// manifest pushes one manifest with its config and layers.
// TRLC-LINKS: REQ-EMG-080
func (l layoutLoader) manifest(ctx context.Context, desc ociregistry.Descriptor) error {
	raw, err := l.blob(desc)
	if err != nil {
		return err
	}
	var manifest ociregistry.Manifest
	if err := json.Unmarshal(raw, &manifest); err != nil {
		return err
	}
	for _, d := range append([]ociregistry.Descriptor{manifest.Config}, manifest.Layers...) {
		if err := l.pushBlob(ctx, d); err != nil {
			return err
		}
	}
	_, err = l.r.PushManifest(ctx, l.ref.Repo, desc.Annotations[ociRefNameAnnotation], raw, desc.MediaType)
	return err
}

// pushBlob pushes one blob of the layout.
// TRLC-LINKS: REQ-EMG-080
func (l layoutLoader) pushBlob(ctx context.Context, d ociregistry.Descriptor) error {
	b, err := l.blob(d)
	if err != nil {
		return err
	}
	_, err = l.r.PushBlob(ctx, l.ref.Repo, d, bytes.NewReader(b))
	return err
}

// blob reads a blob of the layout and checks its size.
// TRLC-LINKS: REQ-EMG-080
func (l layoutLoader) blob(d ociregistry.Descriptor) ([]byte, error) {
	b, err := os.ReadFile(filepath.Join(l.ref.Dir, "blobs", string(d.Digest.Algorithm()), d.Digest.Encoded()))
	if err == nil && int64(len(b)) != d.Size {
		err = fmt.Errorf("blob %s has %d bytes, want %d", d.Digest, len(b), d.Size)
	}
	return b, err
}

// ociLayoutWriter is the write side of an OCI registry that stores what
// modregistry pushes as an OCI image layout, like cue mod publish --out.
type ociLayoutWriter struct {
	dir         string
	descriptors []ociregistry.Descriptor
}

// registry returns the ociregistry.Interface that writes into the layout.
// TRLC-LINKS: REQ-EMG-080
func (w *ociLayoutWriter) registry() ociregistry.Interface {
	return &ociregistry.Funcs{
		PushBlob_: func(_ context.Context, _ string, desc ociregistry.Descriptor, r io.Reader) (ociregistry.Descriptor, error) {
			data, err := io.ReadAll(r)
			if err != nil {
				return desc, err
			}
			return desc, w.writeBlob(desc.Digest, data)
		},
		PushManifest_: func(_ context.Context, _ string, tag string, data []byte, mediaType string) (ociregistry.Descriptor, error) {
			sum := sha256.Sum256(data)
			desc := ociregistry.Descriptor{MediaType: mediaType, Size: int64(len(data)),
				Digest: ociregistry.Digest("sha256:" + hex.EncodeToString(sum[:])), Annotations: map[string]string{ociRefNameAnnotation: tag}}
			w.descriptors = append(w.descriptors, desc)
			return desc, w.writeBlob(desc.Digest, data)
		},
	}
}

// writeBlob stores data under blobs/<algorithm>/<hex> after checking the digest.
// TRLC-LINKS: REQ-EMG-080
func (w *ociLayoutWriter) writeBlob(digest ociregistry.Digest, data []byte) error {
	if err := digest.Validate(); err != nil {
		return err
	}
	verifier := digest.Verifier()
	_, _ = verifier.Write(data)
	if !verifier.Verified() {
		return fmt.Errorf("blob content does not match digest %s", digest)
	}
	path := filepath.Join(w.dir, "blobs", string(digest.Algorithm()), digest.Encoded())
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o644)
}

// writeIndex writes oci-layout and index.json.
// TRLC-LINKS: REQ-EMG-080
func (w *ociLayoutWriter) writeIndex() error {
	data, err := json.MarshalIndent(ociIndex{SchemaVersion: 2, MediaType: "application/vnd.oci.image.index.v1+json", Manifests: w.descriptors}, "", "\t")
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(w.dir, "oci-layout"), []byte("{\n    \"imageLayoutVersion\": \"1.0.0\"\n}\n"), 0o644); err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(w.dir, "index.json"), append(data, '\n'), 0o644)
}

// layoutResolver sends every module to the layout writer under its base path.
type layoutResolver struct{ w *ociLayoutWriter }

// ResolveToRegistry implements modregistry.Resolver.
// TRLC-LINKS: REQ-EMG-080
func (r layoutResolver) ResolveToRegistry(mpath, vers string) (modregistry.RegistryLocation, error) {
	return modregistry.RegistryLocation{Registry: r.w.registry(), Repository: mpath, Tag: vers}, nil
}

// dirFileIO reads module files relative to root for modzip.Create.
type dirFileIO struct{ root string }

// TRLC-LINKS: REQ-EMG-080
func (dirFileIO) Path(f string) string { return f }

// TRLC-LINKS: REQ-EMG-080
func (d dirFileIO) Lstat(f string) (os.FileInfo, error) {
	return os.Lstat(filepath.Join(d.root, filepath.FromSlash(f)))
}

// TRLC-LINKS: REQ-EMG-080
func (d dirFileIO) Open(f string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(d.root, filepath.FromSlash(f)))
}

// goZipFile is one committed file for golang.org/x/mod/zip.Create.
type goZipFile struct{ root, path string }

// TRLC-LINKS: REQ-EMG-080
func (f goZipFile) Path() string { return f.path }

// TRLC-LINKS: REQ-EMG-080
func (f goZipFile) Lstat() (os.FileInfo, error) {
	return os.Lstat(filepath.Join(f.root, filepath.FromSlash(f.path)))
}

// TRLC-LINKS: REQ-EMG-080
func (f goZipFile) Open() (io.ReadCloser, error) {
	return os.Open(filepath.Join(f.root, filepath.FromSlash(f.path)))
}
