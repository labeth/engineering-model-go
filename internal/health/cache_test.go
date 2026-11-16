// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// countingEngine is a fake engine that counts how many files it analysed.
type countingEngine struct {
	version string
	files   int
	calls   int
}

// TRLC-LINKS: REQ-EMG-159
func (e *countingEngine) Name() string { return "counting" }

// TRLC-LINKS: REQ-EMG-159
func (e *countingEngine) Version() string { return e.version }

// TRLC-LINKS: REQ-EMG-159
func (e *countingEngine) Analyze(inputs []FileInput, _ Config) ([]FileResult, error) {
	e.calls++
	out := make([]FileResult, 0, len(inputs))
	for _, in := range inputs {
		e.files++
		out = append(out, FileResult{
			Path: in.Path, Language: in.Language, LOC: len(in.Source),
			Functions: []Function{{Name: "F", StartLine: 1, EndLine: 2, Findings: []Finding{{Code: CodeComplexMethod, Severity: SeverityWarn, Metric: "cyclomatic", Value: 11}}}},
		})
	}
	return out, nil
}

// TRLC-LINKS: REQ-EMG-159
func cacheInput(path, src string) FileInput {
	return FileInput{Path: path, Language: "go", Source: []byte(src)}
}

// entryFiles lists the files in the cache directory.
//
// TRLC-LINKS: REQ-EMG-159
func entryFiles(t *testing.T, c *Cache) []string {
	t.Helper()
	list, err := os.ReadDir(c.Dir())
	if err != nil {
		t.Fatal(err)
	}
	var names []string
	for _, e := range list {
		names = append(names, e.Name())
	}
	return names
}

// The second run on unchanged files is served from the cache, with the same
// results and the paths of the inputs; the entries hold no path.
// Mutation: a Get that always misses fails the call count; storing the path
// fails the entry check.
//
// TRLC-LINKS: REQ-EMG-159
func TestCachedEngineHitsAndMisses(t *testing.T) {
	root := t.TempDir()
	inner := &countingEngine{version: "1"}
	cache := NewCache(root)
	eng := CachedEngine{Engine: inner, Cache: cache}
	inputs := []FileInput{cacheInput("a.go", "package a\n"), cacheInput("b/c.go", "package c\n")}

	first, err := eng.Analyze(inputs, Config{})
	mustEqual(t, [2]any{err, inner.files}, [2]any{nil, 2})
	second, err := eng.Analyze(inputs, Config{})
	mustEqual(t, [3]any{err, inner.files, inner.calls}, [3]any{nil, 2, 1}) // the engine is untouched
	mustEqual(t, [3]any{first[0].Path, second[1].Path, first[1].LOC == second[1].LOC}, [3]any{"a.go", "b/c.go", true})
	hits, misses := cache.Stats()
	wantEqual(t, [2]int{hits, misses}, [2]int{2, 2})
	// One changed file recomputes only that file.
	inputs[1] = cacheInput("b/c.go", "package c // edited\n")
	_, err = eng.Analyze(inputs, Config{})
	mustEqual(t, [2]any{err, inner.files}, [2]any{nil, 3})
	requireEntriesHoldNoPath(t, cache, root)
}

// requireEntriesHoldNoPath checks that no entry of the cache holds a path and
// that each has a file name of a key.
func requireEntriesHoldNoPath(t *testing.T, cache *Cache, root string) {
	t.Helper()
	for _, name := range entryFiles(t, cache) {
		data, err := os.ReadFile(filepath.Join(cache.Dir(), name))
		mustEqual(t, err, nil)
		wantEqual(t, [2]any{name, strings.Contains(string(data), "a.go") || strings.Contains(string(data), "c.go") || strings.Contains(string(data), root)}, [2]any{name, false})
		wantEqual(t, [2]any{name, strings.HasSuffix(name, ".json") && len(name) == 64+len(".json")}, [2]any{name, true})
	}
}

// Everything that decides a result is in the key: a changed threshold, engine
// version, language, content or path is a miss; the order of the threshold map
// is not.
// Mutation: leaving the thresholds or the engine version out of the key fails.
//
// TRLC-LINKS: REQ-EMG-159
func TestCacheKeyCoversEverything(t *testing.T) {
	eng := &countingEngine{version: "1"}
	in := cacheInput("a.go", "package a\n")
	cfg := Config{Thresholds: map[string]Threshold{CodeComplexMethod: {Warn: 5, High: 9}, CodeLargeMethod: {Warn: 40, High: 80}}}
	key := CacheKey(eng, in, cfg)
	if !validKey(key) {
		t.Fatalf("key %q is not 64 hex digits", key)
	}
	same := Config{Thresholds: map[string]Threshold{CodeLargeMethod: {Warn: 40, High: 80}, CodeComplexMethod: {Warn: 5, High: 9}}}
	if CacheKey(eng, in, same) != key {
		t.Error("the key must not depend on map order")
	}
	if CacheKey(&countingEngine{version: "1"}, in, Config{Thresholds: nil}) != CacheKey(eng, in, Config{Thresholds: map[string]Threshold{}}) {
		t.Error("nil and empty thresholds are the same config")
	}
	other := Config{Thresholds: map[string]Threshold{CodeComplexMethod: {Warn: 5, High: 10}, CodeLargeMethod: {Warn: 40, High: 80}}}
	lang := in
	lang.Language = "rust"
	for name, k := range map[string]string{
		"threshold":      CacheKey(eng, in, other),
		"engine version": CacheKey(&countingEngine{version: "2"}, in, cfg),
		"language":       CacheKey(eng, lang, cfg),
		"content":        CacheKey(eng, cacheInput("a.go", "package a // x\n"), cfg),
		"path":           CacheKey(eng, cacheInput("b.go", "package a\n"), cfg),
	} {
		if k == key {
			t.Errorf("a changed %s must change the key", name)
		}
	}
}

// A different threshold or engine version recomputes through the engine.
// Mutation: a key without them serves stale results and fails the counts.
//
// TRLC-LINKS: REQ-EMG-159
func TestCachedEngineInvalidation(t *testing.T) {
	root := t.TempDir()
	inner := &countingEngine{version: "1"}
	cache := NewCache(root)
	in := []FileInput{cacheInput("a.go", "package a\n")}
	run := func(e Engine, cfg Config) {
		t.Helper()
		if _, err := (CachedEngine{Engine: e, Cache: cache}).Analyze(in, cfg); err != nil {
			t.Fatal(err)
		}
	}
	run(inner, Config{})
	run(inner, Config{})
	if inner.files != 1 {
		t.Fatalf("engine ran %d times for an unchanged file", inner.files)
	}
	run(inner, Config{Thresholds: map[string]Threshold{CodeComplexMethod: {Warn: 3, High: 4}}})
	if inner.files != 2 {
		t.Errorf("a changed threshold must recompute; engine ran %d times", inner.files)
	}
	bumped := &countingEngine{version: "2"}
	run(bumped, Config{})
	if bumped.files != 1 {
		t.Errorf("a new engine version must recompute; engine ran %d times", bumped.files)
	}
	if got := (CachedEngine{Engine: bumped}).Name() + (CachedEngine{Engine: bumped}).Version(); got != "counting2" {
		t.Errorf("CachedEngine identity = %q", got)
	}
}

// A corrupt or truncated entry, one with an unknown field, or one that holds a
// path is a miss, and the next Put repairs it.
// Mutation: accepting a partial decode fails the truncated case.
//
// TRLC-LINKS: REQ-EMG-159
func TestCacheCorruptEntryIsMiss(t *testing.T) {
	cache := NewCache(t.TempDir())
	key := strings.Repeat("ab", 32)
	good := FileResult{Language: "go", LOC: 3, Functions: []Function{{Name: "F"}}}
	cache.Put(key, good)
	res, ok := cache.Get(key)
	mustEqual(t, [3]any{ok, res.LOC, res.Path}, [3]any{true, 3, ""})
	file := filepath.Join(cache.Dir(), key+".json")
	for name, content := range map[string]string{
		"empty":     "",
		"garbage":   "not json",
		"truncated": `{"path":"","language":"go","loc":3,"functions":[{"name":"F"`,
		"trailing":  `{"path":"","language":"go","loc":3,"functions":[]} {"x":1}`,
		"unknown":   `{"path":"","language":"go","loc":3,"functions":[],"extra":true}`,
		"has path":  `{"path":"/home/x/a.go","language":"go","loc":3,"functions":[]}`,
		"no list":   `{"path":"","language":"go","loc":3}`,
		"null":      `null`,
	} {
		writeFileOrFail(t, file, content)
		_, hit := cache.Get(key)
		wantEqual(t, [2]any{name, hit}, [2]any{name, false})
	}
	cache.Put(key, good)
	_, ok = cache.Get(key)
	wantEqual(t, ok, true) // Put repairs the corrupt entry
	_, ok = cache.Get("../../etc/passwd")
	wantEqual(t, ok, false) // an invalid key misses
	cache.Put("../escape", good)
}

// writeFileOrFail writes a file (mode 0600) or stops the test.
func writeFileOrFail(t *testing.T, path, content string) {
	t.Helper()
	if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
}

// Put leaves no temporary file behind and replaces an entry whole.
//
// TRLC-LINKS: REQ-EMG-159
func TestCachePutIsAtomic(t *testing.T) {
	cache := NewCache(t.TempDir())
	key := strings.Repeat("0f", 32)
	for i := 0; i < 5; i++ {
		cache.Put(key, FileResult{Language: "go", LOC: i, Functions: []Function{}})
		res, ok := cache.Get(key)
		mustEqual(t, [2]any{ok, res.LOC}, [2]any{true, i})
	}
	wantDeepEqual(t, entryFiles(t, cache), []string{key + ".json"}) // only the entry
	info, err := os.Stat(filepath.Join(cache.Dir(), key+".json"))
	mustEqual(t, err, nil)
	wantEqual(t, info.Mode().Perm(), os.FileMode(0o600))
}

// A symlinked entry, directory or target is never followed: reads miss and
// writes are dropped.
// Mutation: using Stat instead of Lstat reads through the link and fails.
//
// TRLC-LINKS: REQ-EMG-159
func TestCacheRefusesSymlinks(t *testing.T) {
	key := strings.Repeat("1e", 32)
	t.Run("entry", func(t *testing.T) { requireSymlinkedEntryRefused(t, key) })
	t.Run("directory", func(t *testing.T) { requireSymlinkedDirectoryRefused(t, key) })
}

// requireSymlinkedEntryRefused checks that a symlinked entry is not read and
// that Put does not write through it.
func requireSymlinkedEntryRefused(t *testing.T, key string) {
	t.Helper()
	cache := NewCache(t.TempDir())
	cache.Put(strings.Repeat("22", 32), FileResult{Language: "go", Functions: []Function{}}) // makes the directory
	outside := filepath.Join(t.TempDir(), "secret.json")
	writeFileOrFail(t, outside, `{"path":"","language":"go","loc":1,"functions":[]}`)
	if err := os.Symlink(outside, filepath.Join(cache.Dir(), key+".json")); err != nil {
		t.Skip("symlinks are not available:", err)
	}
	_, ok := cache.Get(key)
	wantEqual(t, ok, false)
	cache.Put(key, FileResult{Language: "go", LOC: 9, Functions: []Function{}})
	data, _ := os.ReadFile(outside)
	wantNone(t, string(data), `"loc":9`)
}

// requireSymlinkedDirectoryRefused checks that Put does not write through a
// symlinked cache directory and that Get does not read through it.
func requireSymlinkedDirectoryRefused(t *testing.T, key string) {
	t.Helper()
	root := t.TempDir()
	elsewhere := t.TempDir()
	mustEqual(t, os.MkdirAll(filepath.Join(root, ".engmod"), 0o700), nil)
	if err := os.Symlink(elsewhere, filepath.Join(root, ".engmod", "cache")); err != nil {
		t.Skip("symlinks are not available:", err)
	}
	cache := NewCache(root)
	cache.Put(key, FileResult{Language: "go", Functions: []Function{}})
	list, _ := os.ReadDir(elsewhere)
	wantEqual(t, len(list), 0)
	_, ok := cache.Get(key)
	wantEqual(t, ok, false)
}

// An unwritable cache location is a silent miss, never an error: the engine
// result is still returned.
// Mutation: returning the write error from Analyze fails this test.
//
// TRLC-LINKS: REQ-EMG-159
func TestCacheUnwritableIsSilent(t *testing.T) {
	root := t.TempDir()
	writeFileOrFail(t, filepath.Join(root, ".engmod"), "a file, not a directory")
	inner := &countingEngine{version: "1"}
	eng := CachedEngine{Engine: inner, Cache: NewCache(root)}
	for i := 0; i < 2; i++ {
		res, err := eng.Analyze([]FileInput{cacheInput("a.go", "package a\n")}, Config{})
		mustEqual(t, [2]any{err, len(res)}, [2]any{nil, 1})
		wantEqual(t, res[0].Path, "a.go")
	}
	wantEqual(t, inner.files, 2) // without a usable cache every run analyses
	hits, misses := eng.Cache.Stats()
	wantEqual(t, [2]int{hits, misses}, [2]int{0, 2})
	res, err := (CachedEngine{Engine: inner}).Analyze([]FileInput{cacheInput("a.go", "x")}, Config{})
	wantEqual(t, [2]any{err, len(res)}, [2]any{nil, 1}) // a nil cache passes through
	requireReadOnlyRootIsHarmless(t, inner)
}

// requireReadOnlyRootIsHarmless checks that a read-only cache root does not
// fail the run.
func requireReadOnlyRootIsHarmless(t *testing.T, inner Engine) {
	t.Helper()
	ro := t.TempDir()
	if os.Geteuid() == 0 || os.Chmod(ro, 0o500) != nil {
		return
	}
	defer os.Chmod(ro, 0o700)
	_, err := (CachedEngine{Engine: inner, Cache: NewCache(ro)}).Analyze([]FileInput{cacheInput("a.go", "y")}, Config{})
	wantEqual(t, err, nil)
}

// An entry that was edited (a finding removed, a value lowered), copied under
// another key, or written in the old format without a digest is a miss, so a
// stray edit of the cache can not turn a finding into a pass. The digest is no
// signature: an entry forged with the key rule and its own digest is a hit,
// which is why OpenCache refuses a tracked cache and a CI run.
//
// Mutation: skipping the digest comparison in read fails the edited cases.
//
// TRLC-LINKS: REQ-EMG-159
func TestCacheEntryDigestDetectsEdits(t *testing.T) {
	cache := NewCache(t.TempDir())
	key := strings.Repeat("ab", 32)
	other := strings.Repeat("cd", 32)
	good := FileResult{Language: "go", LOC: 3, Functions: []Function{{Name: "F", Findings: []Finding{{Code: CodeComplexMethod, Severity: SeverityHigh, Metric: "cyclomatic", Value: 30}}}}}
	cache.Put(key, good)
	file := filepath.Join(cache.Dir(), key+".json")
	original, err := os.ReadFile(file)
	mustEqual(t, err, nil)
	wantContains(t, string(original), `"digest"`) // an entry carries its digest
	edits := map[string]string{
		"finding removed": strings.Replace(string(original), `"findings":[{"code":"health.complex_method","severity":"high","startLine":0,"endLine":0,"metric":"cyclomatic","value":30,"message":""}]`, ``, 1),
		"value lowered":   strings.Replace(string(original), `"value":30`, `"value":9`, 1),
		"loc edited":      strings.Replace(string(original), `"loc":3`, `"loc":4`, 1),
		"no digest":       `{"result":{"path":"","language":"go","loc":3,"functions":[]}}`,
		"bare result":     `{"path":"","language":"go","loc":3,"functions":[]}`,
		"digest cleared":  strings.Replace(string(original), `"digest":"`, `"digest":"0`, 1),
	}
	for name, content := range edits {
		mustEqual(t, [2]any{name, content != string(original)}, [2]any{name, true}) // the edit changed something
		writeFileOrFail(t, file, content)
		_, hit := cache.Get(key)
		wantEqual(t, [2]any{name, hit}, [2]any{name, false})
	}
	// Copied under another key.
	writeFileOrFail(t, filepath.Join(cache.Dir(), other+".json"), string(original))
	_, hit := cache.Get(other)
	wantEqual(t, hit, false)
	// The untouched entry still hits.
	writeFileOrFail(t, file, string(original))
	res, ok := cache.Get(key)
	mustEqual(t, ok, true)
	wantEqual(t, len(res.Functions[0].Findings), 1)
}

// OpenCache decides whether a run may use the cache: off with Disabled; off
// where CI is non-empty unless Force; refused, with a note, while git tracks
// any file under .engmod, even with Force; on in a plain work tree and outside
// a git work tree.
//
// Mutation: dropping the CI test, the Force override or the tracked-file test
// fails the matching case.
//
// TRLC-LINKS: REQ-EMG-159
func TestOpenCacheRules(t *testing.T) {
	r := NewFixtureRepo(t)
	r.Write("a.go", "package p\n")
	r.Commit("one")

	requireOpen(t, r.Dir, openWant{policy: CachePolicy{Getenv: ciEnv("")}, open: true}) // a plain work tree
	requireOpen(t, r.Dir, openWant{policy: CachePolicy{Getenv: ciEnv(""), Disabled: true}})
	requireOpen(t, r.Dir, openWant{policy: CachePolicy{Getenv: ciEnv("true")}})                          // CI set
	requireOpen(t, r.Dir, openWant{policy: CachePolicy{Getenv: ciEnv("true"), Force: true}, open: true}) // CI set with Force
	requireOpen(t, r.Dir, openWant{policy: CachePolicy{Getenv: ciEnv("true"), Force: true, Disabled: true}})

	// An untracked cache directory is fine; a tracked entry refuses the cache.
	r.Write(".engmod/cache/health/"+strings.Repeat("ab", 32)+".json", "{}")
	requireOpen(t, r.Dir, openWant{policy: CachePolicy{Getenv: ciEnv("")}, open: true})
	r.Commit("two")
	requireOpen(t, r.Dir, openWant{policy: CachePolicy{Getenv: ciEnv("")}, note: "tracked by git"})
	requireOpen(t, r.Dir, openWant{policy: CachePolicy{Getenv: ciEnv("1"), Force: true}, note: ".engmod/cache/health/"})

	// Outside any work tree nothing can be tracked.
	requireOpen(t, t.TempDir(), openWant{policy: CachePolicy{Getenv: ciEnv("")}, open: true})
}

// ciEnv is an environment where CI has a value and nothing else is set.
func ciEnv(ci string) func(string) string {
	return func(name string) string {
		if name == "CI" {
			return ci
		}
		return ""
	}
}

// openWant is the outcome of OpenCache for a policy: whether it returns a cache
// and the text its note holds (empty: no note).
type openWant struct {
	policy CachePolicy
	open   bool
	note   string
}

// requireOpen checks the outcome of OpenCache in a directory.
func requireOpen(t *testing.T, dir string, w openWant) {
	t.Helper()
	c, note := OpenCache(dir, w.policy)
	wantEqual(t, c != nil, w.open)
	wantEqual(t, strings.Contains(note, w.note), true)
	wantEqual(t, w.note != "" || note == "", true)
}
