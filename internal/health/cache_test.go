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
	if err != nil || inner.files != 2 {
		t.Fatalf("first run: %v, engine analysed %d files", err, inner.files)
	}
	second, err := eng.Analyze(inputs, Config{})
	if err != nil || inner.files != 2 || inner.calls != 1 {
		t.Fatalf("second run: %v, engine analysed %d files in %d calls; want it untouched", err, inner.files, inner.calls)
	}
	if first[0].Path != "a.go" || second[1].Path != "b/c.go" || first[1].LOC != second[1].LOC {
		t.Fatalf("results differ: %+v vs %+v", first, second)
	}
	if hits, misses := cache.Stats(); hits != 2 || misses != 2 {
		t.Errorf("Stats = %d hits, %d misses; want 2 and 2", hits, misses)
	}
	// One changed file recomputes only that file.
	inputs[1] = cacheInput("b/c.go", "package c // edited\n")
	if _, err = eng.Analyze(inputs, Config{}); err != nil || inner.files != 3 {
		t.Fatalf("third run: %v, engine analysed %d files; want 3", err, inner.files)
	}
	for _, name := range entryFiles(t, cache) {
		data, err := os.ReadFile(filepath.Join(cache.Dir(), name))
		if err != nil {
			t.Fatal(err)
		}
		if strings.Contains(string(data), "a.go") || strings.Contains(string(data), "c.go") || strings.Contains(string(data), root) {
			t.Errorf("entry %s holds a path: %s", name, data)
		}
		if !strings.HasSuffix(name, ".json") || len(name) != 64+len(".json") {
			t.Errorf("unexpected cache file name %q", name)
		}
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
	if res, ok := cache.Get(key); !ok || res.LOC != 3 || res.Path != "" {
		t.Fatalf("Get after Put = %+v, %v", res, ok)
	}
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
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, ok := cache.Get(key); ok {
			t.Errorf("%s entry was a hit", name)
		}
	}
	cache.Put(key, good)
	if _, ok := cache.Get(key); !ok {
		t.Error("Put did not repair the corrupt entry")
	}
	if _, ok := cache.Get("../../etc/passwd"); ok {
		t.Error("an invalid key must miss")
	}
	cache.Put("../escape", good)
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
		if !ok || res.LOC != i {
			t.Fatalf("round %d: Get = %+v, %v", i, res, ok)
		}
	}
	if names := entryFiles(t, cache); len(names) != 1 || names[0] != key+".json" {
		t.Errorf("cache directory holds %v, want only the entry", names)
	}
	info, err := os.Stat(filepath.Join(cache.Dir(), key+".json"))
	if err != nil || info.Mode().Perm() != 0o600 {
		t.Errorf("entry mode = %v, %v; want 0600", info, err)
	}
}

// A symlinked entry, directory or target is never followed: reads miss and
// writes are dropped.
// Mutation: using Stat instead of Lstat reads through the link and fails.
//
// TRLC-LINKS: REQ-EMG-159
func TestCacheRefusesSymlinks(t *testing.T) {
	key := strings.Repeat("1e", 32)
	t.Run("entry", func(t *testing.T) {
		cache := NewCache(t.TempDir())
		cache.Put(strings.Repeat("22", 32), FileResult{Language: "go", Functions: []Function{}}) // makes the directory
		outside := filepath.Join(t.TempDir(), "secret.json")
		if err := os.WriteFile(outside, []byte(`{"path":"","language":"go","loc":1,"functions":[]}`), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(outside, filepath.Join(cache.Dir(), key+".json")); err != nil {
			t.Skip("symlinks are not available:", err)
		}
		if _, ok := cache.Get(key); ok {
			t.Error("a symlinked entry was read")
		}
		cache.Put(key, FileResult{Language: "go", LOC: 9, Functions: []Function{}})
		data, _ := os.ReadFile(outside)
		if strings.Contains(string(data), `"loc":9`) {
			t.Error("Put wrote through a symlink")
		}
	})
	t.Run("directory", func(t *testing.T) {
		root := t.TempDir()
		elsewhere := t.TempDir()
		if err := os.MkdirAll(filepath.Join(root, ".engmod"), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(elsewhere, filepath.Join(root, ".engmod", "cache")); err != nil {
			t.Skip("symlinks are not available:", err)
		}
		cache := NewCache(root)
		cache.Put(key, FileResult{Language: "go", Functions: []Function{}})
		if list, _ := os.ReadDir(elsewhere); len(list) != 0 {
			t.Errorf("Put wrote through a symlinked directory: %v", list)
		}
		if _, ok := cache.Get(key); ok {
			t.Error("Get read through a symlinked directory")
		}
	})
}

// An unwritable cache location is a silent miss, never an error: the engine
// result is still returned.
// Mutation: returning the write error from Analyze fails this test.
//
// TRLC-LINKS: REQ-EMG-159
func TestCacheUnwritableIsSilent(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".engmod"), []byte("a file, not a directory"), 0o600); err != nil {
		t.Fatal(err)
	}
	inner := &countingEngine{version: "1"}
	eng := CachedEngine{Engine: inner, Cache: NewCache(root)}
	for i := 0; i < 2; i++ {
		res, err := eng.Analyze([]FileInput{cacheInput("a.go", "package a\n")}, Config{})
		if err != nil || len(res) != 1 || res[0].Path != "a.go" {
			t.Fatalf("run %d: %+v, %v", i, res, err)
		}
	}
	if inner.files != 2 {
		t.Errorf("without a usable cache every run analyses; engine ran %d times", inner.files)
	}
	if hits, misses := eng.Cache.Stats(); hits != 0 || misses != 2 {
		t.Errorf("Stats = %d, %d", hits, misses)
	}
	if res, err := (CachedEngine{Engine: inner}).Analyze([]FileInput{cacheInput("a.go", "x")}, Config{}); err != nil || len(res) != 1 {
		t.Errorf("a nil cache passes through: %v", err)
	}
	if os.Geteuid() != 0 {
		ro := t.TempDir()
		if err := os.Chmod(ro, 0o500); err == nil {
			defer os.Chmod(ro, 0o700)
			if _, err := (CachedEngine{Engine: inner, Cache: NewCache(ro)}).Analyze([]FileInput{cacheInput("a.go", "y")}, Config{}); err != nil {
				t.Errorf("a read-only root must not fail the run: %v", err)
			}
		}
	}
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
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(original), `"digest"`) {
		t.Fatalf("an entry must carry its digest: %s", original)
	}
	edits := map[string]string{
		"finding removed": strings.Replace(string(original), `"findings":[{"code":"health.complex_method","severity":"high","startLine":0,"endLine":0,"metric":"cyclomatic","value":30,"message":""}]`, ``, 1),
		"value lowered":   strings.Replace(string(original), `"value":30`, `"value":9`, 1),
		"loc edited":      strings.Replace(string(original), `"loc":3`, `"loc":4`, 1),
		"no digest":       `{"result":{"path":"","language":"go","loc":3,"functions":[]}}`,
		"bare result":     `{"path":"","language":"go","loc":3,"functions":[]}`,
		"digest cleared":  strings.Replace(string(original), `"digest":"`, `"digest":"0`, 1),
	}
	for name, content := range edits {
		if content == string(original) {
			t.Fatalf("%s: the edit changed nothing in %s", name, original)
		}
		if err := os.WriteFile(file, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if _, ok := cache.Get(key); ok {
			t.Errorf("%s: an edited entry was a hit", name)
		}
	}
	// Copied under another key.
	if err := os.WriteFile(filepath.Join(cache.Dir(), other+".json"), original, 0o600); err != nil {
		t.Fatal(err)
	}
	if _, ok := cache.Get(other); ok {
		t.Error("an entry copied to another key was a hit")
	}
	// The untouched entry still hits.
	if err := os.WriteFile(file, original, 0o600); err != nil {
		t.Fatal(err)
	}
	if res, ok := cache.Get(key); !ok || len(res.Functions[0].Findings) != 1 {
		t.Errorf("the intact entry = %+v, %v", res, ok)
	}
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
	env := func(ci string) func(string) string {
		return func(name string) string {
			if name == "CI" {
				return ci
			}
			return ""
		}
	}
	r := NewFixtureRepo(t)
	r.Write("a.go", "package p\n")
	r.Commit("one")

	if c, note := OpenCache(r.Dir, CachePolicy{Getenv: env("")}); c == nil || note != "" {
		t.Errorf("plain work tree: cache %v note %q, want the cache", c, note)
	}
	if c, note := OpenCache(r.Dir, CachePolicy{Getenv: env(""), Disabled: true}); c != nil || note != "" {
		t.Errorf("Disabled: cache %v note %q, want none", c, note)
	}
	if c, note := OpenCache(r.Dir, CachePolicy{Getenv: env("true")}); c != nil || note != "" {
		t.Errorf("CI set: cache %v note %q, want none", c, note)
	}
	if c, _ := OpenCache(r.Dir, CachePolicy{Getenv: env("true"), Force: true}); c == nil {
		t.Error("CI set with Force: want the cache")
	}
	if c, _ := OpenCache(r.Dir, CachePolicy{Getenv: env("true"), Force: true, Disabled: true}); c != nil {
		t.Error("Disabled wins over Force")
	}

	// An untracked cache directory is fine; a tracked entry refuses the cache.
	r.Write(".engmod/cache/health/"+strings.Repeat("ab", 32)+".json", "{}")
	if c, _ := OpenCache(r.Dir, CachePolicy{Getenv: env("")}); c == nil {
		t.Error("an untracked .engmod must not refuse the cache")
	}
	r.Commit("two")
	for name, p := range map[string]CachePolicy{"plain": {Getenv: env("")}, "forced in CI": {Getenv: env("1"), Force: true}} {
		c, note := OpenCache(r.Dir, p)
		if c != nil || !strings.Contains(note, "tracked by git") || !strings.Contains(note, ".engmod/cache/health/") {
			t.Errorf("%s: cache %v note %q, want it refused with a note naming the tracked file", name, c, note)
		}
	}

	// Outside any work tree nothing can be tracked.
	if c, note := OpenCache(t.TempDir(), CachePolicy{Getenv: env("")}); c == nil || note != "" {
		t.Errorf("outside a work tree: cache %v note %q, want the cache", c, note)
	}
}
