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

// goldenExamples is the record of every example model at the tier it
// declares in engmod.yml (ADR-EMG-061, ADR-EMG-066): per model the tier and
// bundles, the diagnostics of model validation, of document generation and of
// the trace build as JSON, and the SHA-256 of the generated document, the
// decisions document and the trace matrix. It replaces the golden of models
// without lifecycle captured with v0.9.0: a model without lifecycle is an
// error now, so there is nothing to keep byte-identical. A change of a rule,
// a severity or a generated artifact shows here. ENGMOD_UPDATE_GOLDEN=1
// rewrites it. The repository's own model is not part of it: its trace
// changes with every code edit.
const goldenExamples = "internal/integration/testdata/examples-golden.json"

// goldenModel is one model's observable result: the diagnostics of model
// validation, document generation and the trace build as JSON, and the
// SHA-256 of the generated document, decisions document and trace matrix.
type goldenModel struct {
	Tier      string          `json:"tier"`
	Bundles   []string        `json:"bundles"`
	Validate  json.RawMessage `json:"validate"`
	Generate  json.RawMessage `json:"generate"`
	Trace     json.RawMessage `json:"trace"`
	Document  string          `json:"document"`
	Decisions string          `json:"decisions"`
	Matrix    string          `json:"matrix"`
}

// Every example model declares a lifecycle and gives the recorded diagnostics
// and outputs at that tier: validation, the AsciiDoc document and its
// diagnostics, and the trace matrix and its diagnostics.
// Mutation: dropping an example's lifecycle, changing its tier or a tier
// rule, or changing a generated artifact fails the comparison.
// TRLC-LINKS: REQ-EMG-124
func TestExamplesGolden(t *testing.T) {
	t.Setenv(model.WorkspaceEnv, "")
	t.Setenv(composition.LockEnv, composition.LockModeCheck)
	t.Setenv(composition.ModuleCacheEnv, composition.ModuleCacheReuse)
	root, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	recorder := goldenRecorder{t: t, root: root}
	got := map[string]goldenModel{}
	for _, manifest := range exampleManifests(t) {
		got[filepath.ToSlash(manifest)] = recorder.record(manifest)
	}
	if os.Getenv("ENGMOD_UPDATE_GOLDEN") == "1" {
		writeGolden(t, got)
		return
	}
	compareGolden(t, readGolden(t), got)
}

// exampleManifests returns the sorted manifests of every example model.
func exampleManifests(t *testing.T) []string {
	t.Helper()
	var manifests []string
	err := filepath.WalkDir("examples", func(path string, d fs.DirEntry, err error) error {
		switch {
		case err != nil:
			return err
		case d.IsDir() && isGeneratedDir(d.Name()):
			return filepath.SkipDir
		case !d.IsDir() && d.Name() == "engmod.yml":
			manifests = append(manifests, path)
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	sort.Strings(manifests)
	return manifests
}

// isGeneratedDir reports whether a directory holds generated outputs or a
// module cache.
func isGeneratedDir(name string) bool {
	return name == "generated" || name == ".engmod"
}

// goldenRecorder records the observable result of a model; the repository
// root is scrubbed from everything it records.
type goldenRecorder struct {
	t    *testing.T
	root string
}

func (r goldenRecorder) scrub(s string) string { return strings.ReplaceAll(s, r.root, "<root>") }

func (r goldenRecorder) raw(v any) json.RawMessage {
	data, err := json.Marshal(v)
	if err != nil {
		r.t.Fatal(err)
	}
	return json.RawMessage(r.scrub(string(data)))
}

func (r goldenRecorder) hash(s string) string {
	return fmt.Sprintf("%x", sha256.Sum256([]byte(r.scrub(s))))
}

// anyErrors reports whether one of the diagnostic lists holds an error.
func anyErrors(lists ...[]engmodel.Diagnostic) bool {
	for _, diags := range lists {
		if engmodel.HasErrors(diags) {
			return true
		}
	}
	return false
}

// record loads one example and records its validation, generation and trace.
func (r goldenRecorder) record(manifest string) goldenModel {
	t := r.t
	bundle, err := engmodel.LoadBundle(manifest)
	if err != nil {
		t.Fatalf("load %s: %v", manifest, err)
	}
	if bundle.Manifest.Lifecycle == nil {
		t.Fatalf("%s declares no lifecycle; every example is at a stated tier", manifest)
	}
	var g goldenModel
	g.Tier, g.Bundles = bundle.Manifest.Lifecycle.Tier, bundle.Manifest.Lifecycle.EffectiveBundles()
	g.Validate = r.raw(engmodel.SortDiagnostics(engmodel.ValidateBundle(bundle)))
	doc, gerr := engmodel.GenerateAsciiDoc(engmodel.AsciiDocInputs{Bundle: bundle, Requirements: bundle.Requirements, Design: bundle.Design}, engmodel.AsciiDocOptions{CodeRoot: filepath.Dir(manifest)})
	if gerr != nil {
		// A generation error is never a golden: it fails the run, also
		// when the goldens are being updated.
		t.Fatalf("generate %s: %v", manifest, gerr)
	}
	g.Generate, g.Document, g.Decisions = r.raw(doc.Diagnostics), r.hash(doc.Document), r.hash(doc.DecisionsDocument)
	matrix, diags, terr := engmodel.BuildTraceMatrixFromFiles(manifest, bundle.RequirementsPath, filepath.Dir(manifest))
	if terr != nil {
		t.Fatalf("trace %s: %v", manifest, terr)
	}
	g.Trace = r.raw(diags)
	if anyErrors(engmodel.ValidateBundle(bundle), doc.Diagnostics, diags) {
		t.Errorf("%s has errors at its declared tier %s; an example is a complete, valid model", manifest, g.Tier)
	}
	data, _ := json.Marshal(matrix)
	g.Matrix = r.hash(string(data))
	return g
}

// writeGolden rewrites the golden file with got.
func writeGolden(t *testing.T, got map[string]goldenModel) {
	t.Helper()
	data, _ := json.MarshalIndent(got, "", " ")
	if err := os.MkdirAll(filepath.Dir(goldenExamples), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(goldenExamples, append(data, '\n'), 0o644); err != nil {
		t.Fatal(err)
	}
}

// readGolden reads the recorded golden file.
func readGolden(t *testing.T) map[string]goldenModel {
	t.Helper()
	data, err := os.ReadFile(goldenExamples)
	if err != nil {
		t.Fatal(err)
	}
	var want map[string]goldenModel
	if err := json.Unmarshal(data, &want); err != nil {
		t.Fatal(err)
	}
	return want
}

// compareGolden reports every recorded model that is missing or differs.
func compareGolden(t *testing.T, want, got map[string]goldenModel) {
	t.Helper()
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
			t.Errorf("%s differs from the golden (ENGMOD_UPDATE_GOLDEN=1 rewrites it):\nwant %s\ngot  %s", name, wj, gj)
		}
	}
}
