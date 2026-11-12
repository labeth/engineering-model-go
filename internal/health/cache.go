// ENGMODEL-OWNER-UNIT: FU-CODE-HEALTH
package health

// The per-file result cache (ADR-EMG-054). Entries live under
// <root>/.engmod/cache/health/<key>.json, keyed by the sha256 of everything
// that decides the result. The cache is a build product: a missing, corrupt or
// unreadable entry is a miss and an unwritable directory loses the entry
// silently, so the cache can make a run faster but never fail it or change its
// output.
//
// What the cache does not do is authenticate an entry. The key is computable
// from public inputs (the file content, the limits and the engine version), so
// whoever can commit a file under .engmod/cache/health can forge an entry for
// it. Each entry carries a structural digest (sha256 over the key and the
// canonical JSON of the result), which turns an edited or truncated entry into
// a miss but not a forged one made with the key rule. That is why the cache is
// not trusted by default: OpenCache leaves it off where CI is set unless the
// caller asks for it, and refuses it while any file under .engmod is tracked
// by git.

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync/atomic"
)

// cacheSubdir is the cache directory below the project root.
var cacheSubdir = []string{".engmod", "cache", "health"}

// cacheKeyVersion changes when the layout of the entries or of the key changes.
const cacheKeyVersion = 2

// maxCacheEntry bounds the size of an entry that is read back.
const maxCacheEntry = 16 << 20

// Cache stores the FileResult of a file by content key.
type Cache struct {
	root         string
	hits, misses atomic.Int64
}

// NewCache returns the cache of the project at root. It touches no file
// until the first Put.
//
// TRLC-LINKS: REQ-EMG-159
func NewCache(root string) *Cache { return &Cache{root: root} }

// Dir is the cache directory.
//
// TRLC-LINKS: REQ-EMG-159
func (c *Cache) Dir() string {
	return filepath.Join(append([]string{c.root}, cacheSubdir...)...)
}

// Stats returns the number of Get calls that found an entry and that did not.
//
// TRLC-LINKS: REQ-EMG-159
func (c *Cache) Stats() (hits, misses int) {
	return int(c.hits.Load()), int(c.misses.Load())
}

// cacheKeyInput is what decides a result. The path is in the key because the
// engine's test detection reads the file name; it is not in the entry.
type cacheKeyInput struct {
	Version        int                  `json:"version"`
	Engine         string               `json:"engine"`
	EngineVersion  string               `json:"engineVersion"`
	ScoringVersion int                  `json:"scoringVersion"`
	Thresholds     map[string]Threshold `json:"thresholds"`
	Language       string               `json:"language"`
	Path           string               `json:"path"`
	ContentSHA256  string               `json:"contentSha256"`
}

// CacheKey returns the key of the result of analysing in with cfg on the
// engine. The thresholds are marshalled as canonical JSON (sorted keys), so
// the key does not depend on map order.
//
// TRLC-LINKS: REQ-EMG-159
func CacheKey(engine Engine, in FileInput, cfg Config) string {
	content := sha256.Sum256(in.Source)
	thresholds := cfg.Thresholds
	if thresholds == nil {
		thresholds = map[string]Threshold{}
	}
	raw, err := json.Marshal(cacheKeyInput{
		Version: cacheKeyVersion, Engine: engine.Name(), EngineVersion: engine.Version(),
		ScoringVersion: ScoringVersion, Thresholds: thresholds, Language: in.Language,
		Path: in.Path, ContentSHA256: hex.EncodeToString(content[:]),
	})
	if err != nil { // a map of ints and strings always marshals
		panic(fmt.Sprintf("health: cache key: %v", err))
	}
	sum := sha256.Sum256(raw)
	return hex.EncodeToString(sum[:])
}

// validKey reports whether key is a 64-digit lower-case hex string, so a key
// can never name a path outside the cache directory.
//
// TRLC-LINKS: REQ-EMG-159
func validKey(key string) bool {
	if len(key) != 64 {
		return false
	}
	for _, r := range key {
		if (r < '0' || r > '9') && (r < 'a' || r > 'f') {
			return false
		}
	}
	return true
}

// usable checks that every component of the cache directory that exists is a
// real directory and not a symlink; with create it makes the missing ones.
//
// TRLC-LINKS: REQ-EMG-159
func (c *Cache) usable(create bool) bool {
	dir := c.root
	for _, part := range cacheSubdir {
		dir = filepath.Join(dir, part)
		info, err := os.Lstat(dir)
		switch {
		case err == nil:
			if !info.IsDir() {
				return false
			}
		case os.IsNotExist(err) && create:
			if os.Mkdir(dir, 0o700) != nil {
				return false
			}
		default:
			return false
		}
	}
	return true
}

// Get returns the cached result for key. The returned result has an empty
// Path; the caller sets it. A missing, corrupt or symlinked entry is a miss.
//
// TRLC-LINKS: REQ-EMG-159
func (c *Cache) Get(key string) (FileResult, bool) {
	res, ok := c.read(key)
	if ok {
		c.hits.Add(1)
	} else {
		c.misses.Add(1)
	}
	return res, ok
}

// TRLC-LINKS: REQ-EMG-159
func (c *Cache) read(key string) (FileResult, bool) {
	if !validKey(key) || !c.usable(false) {
		return FileResult{}, false
	}
	file := filepath.Join(c.Dir(), key+".json")
	info, err := os.Lstat(file)
	if err != nil || !info.Mode().IsRegular() || info.Size() > maxCacheEntry {
		return FileResult{}, false
	}
	data, err := os.ReadFile(file)
	if err != nil {
		return FileResult{}, false
	}
	dec := json.NewDecoder(bytes.NewReader(data))
	dec.DisallowUnknownFields()
	var entry cacheEntry
	if dec.Decode(&entry) != nil || dec.More() {
		return FileResult{}, false
	}
	res := entry.Result
	if res.Path != "" || res.Functions == nil || entry.Digest != entryDigest(key, res) {
		return FileResult{}, false
	}
	return res, true
}

// cacheEntry is the content of an entry file: the result and its digest.
type cacheEntry struct {
	Digest string     `json:"digest"`
	Result FileResult `json:"result"`
}

// entryDigest is the structural digest of an entry: sha256 over the key, a
// newline and the canonical JSON of the result. It detects an entry that was
// edited, truncated or copied under another key; it is no signature.
//
// TRLC-LINKS: REQ-EMG-159
func entryDigest(key string, res FileResult) string {
	raw, err := json.Marshal(res)
	if err != nil { // a result of ints and strings always marshals
		panic(fmt.Sprintf("health: cache digest: %v", err))
	}
	sum := sha256.Sum256(append([]byte(key+"\n"), raw...))
	return hex.EncodeToString(sum[:])
}

// Put stores res under key without its path or any other host detail. It
// writes a temporary file and renames it, so a reader sees a whole entry or
// none; any failure drops the entry silently.
//
// TRLC-LINKS: REQ-EMG-159
func (c *Cache) Put(key string, res FileResult) {
	if !validKey(key) || !c.usable(true) {
		return
	}
	res.Path = ""
	if res.Functions == nil {
		res.Functions = []Function{}
	}
	data, err := json.Marshal(cacheEntry{Digest: entryDigest(key, res), Result: res})
	if err != nil {
		return
	}
	final := filepath.Join(c.Dir(), key+".json")
	if info, err := os.Lstat(final); err == nil && !info.Mode().IsRegular() {
		return // never write through a symlink or into a directory
	}
	tmp, err := os.CreateTemp(c.Dir(), ".tmp-*")
	if err != nil {
		return
	}
	_, werr := tmp.Write(data)
	cerr := tmp.Close()
	if werr != nil || cerr != nil || os.Rename(tmp.Name(), final) != nil {
		_ = os.Remove(tmp.Name())
	}
}

// CachedEngine wraps an Engine with a Cache. It names and versions itself as
// the inner engine, so a result is the same with and without the cache.
type CachedEngine struct {
	Engine Engine
	Cache  *Cache
}

var _ Engine = CachedEngine{}

// Name is the name of the inner engine.
//
// TRLC-LINKS: REQ-EMG-159
func (e CachedEngine) Name() string { return e.Engine.Name() }

// Version is the version of the inner engine.
//
// TRLC-LINKS: REQ-EMG-159
func (e CachedEngine) Version() string { return e.Engine.Version() }

// Analyze serves each input from the cache when it can and sends the rest to
// the inner engine in one call; results are in input order.
//
// TRLC-LINKS: REQ-EMG-159
func (e CachedEngine) Analyze(inputs []FileInput, cfg Config) ([]FileResult, error) {
	if e.Cache == nil {
		return e.Engine.Analyze(inputs, cfg)
	}
	results := make([]FileResult, len(inputs))
	keys := make([]string, len(inputs))
	var missing []FileInput
	var missingAt []int
	for i, in := range inputs {
		keys[i] = CacheKey(e.Engine, in, cfg)
		if res, ok := e.Cache.Get(keys[i]); ok {
			res.Path = in.Path
			results[i] = res
			continue
		}
		missing = append(missing, in)
		missingAt = append(missingAt, i)
	}
	if len(missing) == 0 {
		return results, nil
	}
	fresh, err := e.Engine.Analyze(missing, cfg)
	if err != nil {
		return nil, err
	}
	if len(fresh) != len(missing) {
		return nil, fmt.Errorf("health: engine %s returned %d results for %d files", e.Engine.Name(), len(fresh), len(missing))
	}
	for j, res := range fresh {
		i := missingAt[j]
		e.Cache.Put(keys[i], res)
		results[i] = res
	}
	return results, nil
}

// CachePolicy decides whether a run may use the result cache.
type CachePolicy struct {
	// Disabled turns the cache off (--no-cache).
	Disabled bool
	// Force keeps the cache on where CI is set (--cache).
	Force bool
	// Getenv reads the environment; nil means os.Getenv.
	Getenv func(string) string
}

// OpenCache returns the cache of the project at root, or nil when this run
// must not use it, and a note that says why when the reason is not obvious to
// the caller. It is off when the policy disables it, and off where the
// environment variable CI is non-empty unless the policy forces it. It is
// refused, with a note, when git lists any tracked file under .engmod of the
// project: a committed cache entry could turn a finding into a pass. When git
// cannot be asked (it is missing or fails) the cache is refused as well; a
// directory that is not inside a git work tree has nothing tracked.
//
// TRLC-LINKS: REQ-EMG-159
func OpenCache(root string, policy CachePolicy) (*Cache, string) {
	getenv := policy.Getenv
	if getenv == nil {
		getenv = os.Getenv
	}
	if policy.Disabled || (getenv("CI") != "" && !policy.Force) {
		return nil, ""
	}
	tracked, err := NewGitRepo(root).TrackedFiles(".engmod")
	switch {
	case errors.Is(err, ErrNotInWorkTree):
	case err != nil:
		return nil, "the result cache is not used: git could not say whether .engmod is tracked (" + err.Error() + ")"
	case len(tracked) > 0:
		return nil, fmt.Sprintf("the result cache is not used: %d file(s) under .engmod are tracked by git (for example %s); a committed cache entry could hide a finding, so run without them or untrack them", len(tracked), tracked[0])
	}
	return NewCache(root), ""
}
