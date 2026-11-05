// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package integration

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	engmodel "github.com/labeth/engineering-model-go"
	"github.com/labeth/engineering-model-go/internal/composition"
	"github.com/labeth/engineering-model-go/internal/model"
)

// goldenNoLifecycle is the record of every example model's diagnostics and
// generated outputs, captured with engineering-model-go v0.9.0 (before the
// life-cycle tier existed). ENGMOD_UPDATE_GOLDEN=1 rewrites it. The
// repository's own model is compared by scripts/lifecycle-golden-self.sh,
// which runs the v0.9.0 and the current binaries on the same tree: its trace
// changes with every code edit, so a stored record cannot hold for it.
const goldenNoLifecycle = "internal/integration/testdata/no-lifecycle-golden.json"

// goldenModel is one model's observable result: the diagnostics of model
// validation, document generation and the trace build as JSON, and the
// SHA-256 of the generated document, decisions document and trace matrix.
type goldenModel struct {
	Validate  json.RawMessage `json:"validate"`
	Generate  json.RawMessage `json:"generate"`
	GenError  string          `json:"generateError,omitempty"`
	Trace     json.RawMessage `json:"trace"`
	Document  string          `json:"document"`
	Decisions string          `json:"decisions"`
	Matrix    string          `json:"matrix"`
}

// A manifest without lifecycle gives byte-identical diagnostics and outputs
// to v0.9.0 for every example model: validation, the AsciiDoc document and
// its diagnostics, and the trace matrix and its diagnostics.
// TRLC-LINKS: REQ-EMG-124
func TestNoLifecycleGoldenOverExamples(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, composition.LockModeCheck)
	t.Setenv(composition.ModuleCacheEnv, composition.ModuleCacheReuse)
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	var manifests []string
	err = filepath.WalkDir("examples", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() && (d.Name() == "generated" || d.Name() == ".engmod") {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == "engmod.yml" {
			manifests = append(manifests, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(manifests)
	scrub := func(s string) string { return strings.ReplaceAll(s, root, "<root>") }
	raw := func(v any) json.RawMessage {
		data, err := json.Marshal(v)
		if err != nil {
			t.Fatal(err)
		}
		return json.RawMessage(scrub(string(data)))
	}
	hash := func(s string) string { return fmt.Sprintf("%x", sha256.Sum256([]byte(scrub(s)))) }
	got := map[string]goldenModel{}
	for _, manifest := range manifests {
		bundle, err := engmodel.LoadBundle(manifest)
		if err != nil {
			t.Fatalf("load %s: %v", manifest, err)
		}
		if bundle.Manifest.Lifecycle != nil {
			t.Fatalf("%s declares lifecycle; the golden covers models without it", manifest)
		}
		var g goldenModel
		g.Validate = raw(engmodel.SortDiagnostics(engmodel.ValidateBundle(bundle)))
		doc, gerr := engmodel.GenerateAsciiDoc(bundle, bundle.Requirements, bundle.Design, engmodel.AsciiDocOptions{CodeRoot: filepath.Dir(manifest)})
		if gerr != nil {
			g.GenError = gerr.Error()
		}
		g.Generate, g.Document, g.Decisions = raw(doc.Diagnostics), hash(doc.Document), hash(doc.DecisionsDocument)
		matrix, diags, terr := engmodel.BuildTraceMatrixFromFiles(manifest, bundle.RequirementsPath, filepath.Dir(manifest))
		if terr != nil {
			t.Fatalf("trace %s: %v", manifest, terr)
		}
		g.Trace = raw(diags)
		data, _ := json.Marshal(matrix)
		g.Matrix = hash(string(data))
		got[filepath.ToSlash(manifest)] = g
	}
	if os.Getenv("ENGMOD_UPDATE_GOLDEN") == "1" {
		data, _ := json.MarshalIndent(got, "", " ")
		if err := os.MkdirAll(filepath.Dir(goldenNoLifecycle), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(goldenNoLifecycle, append(data, '\n'), 0o644); err != nil {
			t.Fatal(err)
		}
		return
	}
	data, err := os.ReadFile(goldenNoLifecycle)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]goldenModel
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	if len(want) != len(got) {
		t.Errorf("golden has %d models, the examples %d", len(want), len(got))
	}
	for name, w := range want {
		g, ok := got[name]
		if !ok {
			t.Errorf("%s: missing from the examples", name)
			continue
		}
		wj, _ := json.Marshal(w)
		gj, _ := json.Marshal(g)
		if string(wj) != string(gj) {
			t.Errorf("%s differs from v0.9.0:\nwant %s\ngot  %s", name, wj, gj)
		}
	}
}
