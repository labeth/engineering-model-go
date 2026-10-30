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
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, e := range entries {
		header := &zip.FileHeader{Name: e.name, Method: zip.Store}
		if e.symlink {
			header.SetMode(os.ModeSymlink | 0o777)
		}
		w, err := zw.CreateHeader(header)
		if err != nil {
			t.Fatal(err)
		}
		if _, err := w.Write([]byte(e.content)); err != nil {
			t.Fatal(err)
		}
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	digestOf := func(b []byte) string { return fmt.Sprintf("sha256:%x", sha256.Sum256(b)) }
	zipBlob := buf.Bytes()
	modFile := []byte("module: \"models.atlas.example/shared-edge@v0\"\nlanguage: version: \"v0.17.0\"\n")
	config := []byte("{}")
	blobs := map[string][]byte{digestOf(zipBlob): zipBlob, digestOf(modFile): modFile, digestOf(config): config}
	served := map[string][]byte{}
	for d, b := range blobs {
		served[d] = b
	}
	if corrupt {
		// A well-formed zip with other content under the original digest.
		var tampered bytes.Buffer
		tw := zip.NewWriter(&tampered)
		for _, e := range entries {
			w, err := tw.Create(e.name)
			if err != nil {
				t.Fatal(err)
			}
			_, _ = w.Write([]byte(e.content + "# tampered\n"))
		}
		if err := tw.Close(); err != nil {
			t.Fatal(err)
		}
		served[digestOf(zipBlob)] = tampered.Bytes()
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
	if err != nil {
		t.Fatal(err)
	}
	const repo = "/v2/models.atlas.example/shared-edge/"
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/v2/":
			w.WriteHeader(http.StatusOK)
		case r.URL.Path == repo+"manifests/v0.1.0" || r.URL.Path == repo+"manifests/"+digestOf(manifest):
			w.Header().Set("Content-Type", "application/vnd.oci.image.manifest.v1+json")
			w.Header().Set("Docker-Content-Digest", digestOf(manifest))
			w.Header().Set("Content-Length", fmt.Sprint(len(manifest)))
			if r.Method != http.MethodHead {
				_, _ = w.Write(manifest)
			}
		case strings.HasPrefix(r.URL.Path, repo+"blobs/"):
			body, ok := served[strings.TrimPrefix(r.URL.Path, repo+"blobs/")]
			if !ok {
				http.NotFound(w, r)
				return
			}
			w.Header().Set("Content-Type", "application/octet-stream")
			w.Header().Set("Content-Length", fmt.Sprint(len(body)))
			if r.Method != http.MethodHead {
				_, _ = w.Write(body)
			}
		default:
			http.NotFound(w, r)
		}
	}))
	t.Cleanup(srv.Close)
	t.Setenv("CUE_REGISTRY", strings.TrimPrefix(srv.URL, "http://")+"+insecure")
}

// TRLC-LINKS: REQ-EMG-047
func validModuleEntries() []rawZipEntry {
	return []rawZipEntry{
		{name: "cue.mod/module.cue", content: "module: \"models.atlas.example/shared-edge@v0\"\nlanguage: version: \"v0.17.0\"\n"},
		{name: "engmod.yml", content: "schemaVersion: 2\n"},
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
			workspace := filepath.Join(root, "workspace")
			testutil.WriteParentWithOCIModule(t, workspace, "models.atlas.example/shared-edge@v0", "v0.1.0")
			res, err := GenerateCompositionFromFile(filepath.Join(workspace, "engmod.yml"))
			if err != nil {
				t.Fatal(err)
			}
			assertCompositionDiagnostic(t, res.Diagnostics, "composition.module_fetch_failed")
			for _, p := range []string{
				filepath.Join(root, "escaped.txt"), filepath.Join(workspace, "escaped.txt"),
				filepath.Join(workspace, ".engmod", "escaped.txt"), filepath.Join(workspace, ".engmod", "modules", "escaped.txt"),
				"/tmp/engmod-zip-absolute-escape.txt",
			} {
				if _, err := os.Lstat(p); err == nil {
					t.Fatalf("crafted zip wrote %s", p)
				}
			}
			entries, _ := os.ReadDir(filepath.Join(workspace, ".engmod", "modules"))
			for _, e := range entries {
				if e.IsDir() {
					t.Fatalf("a rejected module left a cache entry: %s", e.Name())
				}
			}
		})
	}
	// Control: the same registry serves a valid module that is fetched.
	root := t.TempDir()
	serveRawModule(t, validModuleEntries(), false)
	workspace := filepath.Join(root, "workspace")
	testutil.WriteParentWithOCIModule(t, workspace, "models.atlas.example/shared-edge@v0", "v0.1.0")
	res, err := GenerateCompositionFromFile(filepath.Join(workspace, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	for _, d := range res.Diagnostics {
		if d.Code == "composition.module_fetch_failed" {
			t.Fatalf("control module must be fetched: %+v", d)
		}
	}
}
