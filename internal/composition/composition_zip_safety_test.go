// ENGMODEL-OWNER-UNIT: FU-SYSTEM-COMPOSITION
package composition

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
	"github.com/labeth/engineering-model-go/internal/testutil"
)

// rawZipEntry is one crafted entry of a module zip.
type rawZipEntry struct {
	name, content string
	symlink       bool
}

// serveRawModule serves models.atlas.example/shared-edge@v0 v0.1.0 from a
// hand-written OCI registry whose module zip holds exactly entries (no
// modzip normalization). With corrupt the zip blob is served with other
// bytes than its digest names.
// TRLC-LINKS: REQ-EMG-047
func serveRawModule(t *testing.T, entries []rawZipEntry, corrupt bool) {
	t.Helper()
	zipBlob := zipOf(t, entries, func(e rawZipEntry) rawZipEntry { return e })
	modFile := []byte("module: \"models.atlas.example/shared-edge@v0\"\nlanguage: version: \"v0.17.0\"\n")
	config := []byte("{}")
	reg := rawRegistry{served: map[string][]byte{digestOf(zipBlob): zipBlob, digestOf(modFile): modFile, digestOf(config): config}}
	if corrupt {
		// A well-formed zip with other content under the original digest.
		reg.served[digestOf(zipBlob)] = zipOf(t, entries, func(e rawZipEntry) rawZipEntry {
			e.content += "# tampered\n"
			e.symlink = false
			return e
		})
	}
	manifest, err := json.Marshal(map[string]any{
		"schemaVersion": 2,
		"mediaType":     "application/vnd.oci.image.manifest.v1+json",
		"config":        map[string]any{"mediaType": "application/vnd.cue.module.v1+json", "digest": digestOf(config), "size": len(config)},
		"layers": []map[string]any{
			{"mediaType": "application/zip", "digest": digestOf(zipBlob), "size": len(zipBlob)},
			{"mediaType": "application/vnd.cue.modulefile.v1", "digest": digestOf(modFile), "size": len(modFile)},
		},
	})
	requireEqual(t, err, nil)
	reg.manifest = manifest
	srv := httptest.NewServer(reg)
	t.Cleanup(srv.Close)
	t.Setenv("CUE_REGISTRY", strings.TrimPrefix(srv.URL, "http://")+"+insecure")
}

// digestOf is the OCI digest of a blob.
func digestOf(b []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(b)) }

// zipOf writes the entries, each as adapt changes it, into a stored zip.
func zipOf(t *testing.T, entries []rawZipEntry, adapt func(rawZipEntry) rawZipEntry) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		e = adapt(e)
		header := &zip.FileHeader{Name: e.name, Method: zip.Store}
		if e.symlink {
			header.SetMode(os.ModeSymlink | 0o777)
		}
		w, err := zw.CreateHeader(header)
		requireEqual(t, err, nil)
		_, err = w.Write([]byte(e.content))
		requireEqual(t, err, nil)
	}
	requireEqual(t, zw.Close(), nil)
	return buf.Bytes()
}

// rawRegistryRepo is the repository the raw registry serves.
const rawRegistryRepo = "/v2/models.atlas.example/shared-edge/"

// rawRegistry is a hand-written OCI registry for one module version: its
// manifest and the blobs it names.
type rawRegistry struct {
	manifest []byte
	served   map[string][]byte
}

// ServeHTTP serves the registry root, the manifest and the blobs.
func (reg rawRegistry) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch {
	case r.URL.Path == "/v2/":
		w.WriteHeader(http.StatusOK)
	case r.URL.Path == rawRegistryRepo+"manifests/v0.1.0" || r.URL.Path == rawRegistryRepo+"manifests/"+digestOf(reg.manifest):
		w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
		w.Header().Set("Docker-Content-Digest", digestOf(reg.manifest))
		writeBody(w, r, reg.manifest)
	case strings.HasPrefix(r.URL.Path, rawRegistryRepo+"blobs/"):
		reg.serveBlob(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveBlob serves the blob the path names, or 404.
func (reg rawRegistry) serveBlob(w http.ResponseWriter, r *http.Request) {
	body, ok := reg.served[strings.TrimPrefix(r.URL.Path, rawRegistryRepo+"blobs/")]
	if !ok {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", "application/octet-stream")
	writeBody(w, r, body)
}

// writeBody sets the length of the body and writes it unless the request is a HEAD.
func writeBody(w http.ResponseWriter, r *http.Request, body []byte) {
	w.Header().Set("Content-Length", fmt.Sprint(len(body)))
	if r.Method != http.MethodHead {
		_, _ = w.Write(body)
	}
}

// TRLC-LINKS: REQ-EMG-047
func validModuleEntries() []rawZipEntry {
	return []rawZipEntry{
		{name: "cue.mod/module.cue", content: "module: \"models.atlas.example/shared-edge@v0\"\nlanguage: version: \"v0.17.0\"\n"},
		{name: "engmod.yml", content: "schemaVersion: 3\n"},
	}
}

// A registry module zip with a ../ entry, an absolute entry or a symlink, or
// a zip blob whose bytes do not match its digest, fails the fetch
// (composition.module_fetch_failed) and nothing is written outside the
// module cache; the cache keeps no entry for the module.
// Mutations: ignoring the modzip.Unzip error (dotdot, absolute), ignoring the blob
// read error that carries the digest mismatch (digest), or dropping
// checkModuleZipEntries (symlink) fails this test.
// TRLC-LINKS: REQ-EMG-047, REQ-EMG-049
// ENGMODEL-LINKS: FU-SYSTEM-COMPOSITION, REF-CUE-OCI-MODULE-REGISTRY
func TestCraftedModuleZipIsRejected(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(LockEnv, "")
	t.Setenv(ModuleCacheEnv, "")
	for name, tc := range map[string]struct {
		extra   []rawZipEntry
		corrupt bool
	}{
		"dotdot":   {extra: []rawZipEntry{{name: "../escaped.txt", content: "x"}, {name: "../../escaped.txt", content: "x"}}},
		"absolute": {extra: []rawZipEntry{{name: "/tmp/engmod-zip-absolute-escape.txt", content: "x"}}},
		"symlink":  {extra: []rawZipEntry{{name: "link", content: "/etc/passwd", symlink: true}}},
		"digest":   {corrupt: true},
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			serveRawModule(t, append(validModuleEntries(), tc.extra...), tc.corrupt)
			workspace := fetchWorkspace(t, root)
			res, err := GenerateCompositionFromFile(filepath.Join(workspace, "engmod.yml"))
			requireEqual(t, err, nil)
			assertCompositionDiagnostic(t, res.Diagnostics, "composition.module_fetch_failed")
			requireNothingEscaped(t, root, workspace)
		})
	}
	// Control: the same registry serves a valid module that is fetched.
	serveRawModule(t, validModuleEntries(), false)
	workspace := fetchWorkspace(t, t.TempDir())
	res, err := GenerateCompositionFromFile(filepath.Join(workspace, "engmod.yml"))
	requireEqual(t, err, nil)
	requireEqual(t, firstDiag(res.Diagnostics, "composition.module_fetch_failed") == nil, true)
}

// fetchWorkspace writes a parent model that depends on the shared-edge module.
func fetchWorkspace(t *testing.T, root string) string {
	workspace := filepath.Join(root, "workspace")
	testutil.WriteParentWithOCIModule(t, workspace, testutil.OCIParent{Path: "models.atlas.example/shared-edge@v0", Version: "v0.1.0"})
	return workspace
}

// requireNothingEscaped fails the test when a crafted zip wrote outside the
// module cache or the cache kept an entry for the rejected module.
func requireNothingEscaped(t *testing.T, root, workspace string) {
	t.Helper()
	for _, p := range []string{
		filepath.Join(root, "escaped.txt"), filepath.Join(workspace, "escaped.txt"),
		filepath.Join(workspace, ".engmod", "escaped.txt"), filepath.Join(workspace, ".engmod", "modules", "escaped.txt"),
		"/tmp/engmod-zip-absolute-escape.txt",
	} {
		requireNoFile(t, p)
	}
	entries, _ := os.ReadDir(filepath.Join(workspace, ".engmod", "modules"))
	for _, e := range entries {
		requireEqual(t, e.IsDir(), false) // a rejected module left a cache entry
	}
}
