// ENGMODEL-OWNER-UNIT: FU-MODEL-LOADER
package model

import (
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

const baselineBlock = `health:
  baseline:
    file: model/health-baseline.json
    decision: ADR-DEBT
`

// A manifest with health.baseline loads into the typed HealthBaseline, a
// manifest without it leaves it nil, and the Go type and the CUE contract agree
// on the keys: every yaml key of the type is accepted by the schema.
// TRLC-LINKS: REQ-EMG-153, REQ-EMG-160
func TestManifestHealthBaselineLoads(t *testing.T) {
	dir := writeV2Fixture(t, validManifest+baselineBlock)
	bundle, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
	if err != nil {
		t.Fatal(err)
	}
	want := &HealthBaseline{File: "model/health-baseline.json", Decision: "ADR-DEBT"}
	if !reflect.DeepEqual(bundle.Manifest.Health.Baseline, want) {
		t.Fatalf("baseline = %+v, want %+v", bundle.Manifest.Health.Baseline, want)
	}
	plain := writeV2Fixture(t, validManifest+healthBlock)
	bundle, err = LoadBundle(filepath.Join(plain, "engmod.yml"))
	if err != nil || bundle.Manifest.Health.Baseline != nil {
		t.Fatalf("a health block without baseline must leave it nil: %v %+v", err, bundle.Manifest.Health)
	}
	var keys []string
	typ := reflect.TypeOf(HealthBaseline{})
	for i := 0; i < typ.NumField(); i++ {
		keys = append(keys, strings.Split(typ.Field(i).Tag.Get("yaml"), ",")[0])
	}
	if !reflect.DeepEqual(keys, []string{"file", "decision"}) {
		t.Fatalf("Go keys = %v; the CUE #HealthBaseline must be changed with them", keys)
	}
}

// The CUE contract is closed over the baseline: an unknown key, a missing field,
// an empty value, a non-string value and a baseline that is not a mapping are
// rejected on load with the field.
// TRLC-LINKS: REQ-EMG-153
func TestManifestHealthBaselineRejectsMalformedShapes(t *testing.T) {
	cases := map[string]struct{ block, want string }{
		"unknown key":      {"health:\n  baseline: {file: a.json, decision: ADR-A, until: tomorrow}\n", "until"},
		"missing decision": {"health:\n  baseline: {file: a.json}\n", "decision"},
		"missing file":     {"health:\n  baseline: {decision: ADR-A}\n", "file"},
		"empty file":       {"health:\n  baseline: {file: \"\", decision: ADR-A}\n", "file"},
		"number file":      {"health:\n  baseline: {file: 7, decision: ADR-A}\n", "file"},
		"not a mapping":    {"health:\n  baseline: a.json\n", "baseline"},
	}
	for name, c := range cases {
		t.Run(name, func(t *testing.T) {
			dir := writeV2Fixture(t, validManifest+c.block)
			_, err := LoadBundle(filepath.Join(dir, "engmod.yml"))
			if err == nil || !strings.Contains(err.Error(), c.want) {
				t.Fatalf("expected a rejection naming %q, got %v", c.want, err)
			}
		})
	}
}

// The encoded baseline is canonical: sorted by path, function and code, without
// duplicates, with the schema string, a trailing newline and no timestamp, and
// parsing it gives the same entries back, so writing is idempotent.
// TRLC-LINKS: REQ-EMG-153
func TestHealthBaselineEncodeIsCanonicalAndRoundTrips(t *testing.T) {
	entries := []HealthBaselineEntry{
		{Path: "b.go", Function: "f", Code: "health.large_method", Value: 70},
		{Path: "a.go", Function: "g", Code: "health.complex_method", Value: 12},
		{Path: "a.go", Function: "f", Code: "health.large_method", Value: 55},
		{Path: "a.go", Function: "f", Code: "health.complex_method", Value: 9},
		{Path: "a.go", Function: "f", Code: "health.complex_method", Value: 14},
	}
	data, err := EncodeHealthBaseline(entries)
	if err != nil {
		t.Fatal(err)
	}
	want := `{
  "schema": "engmod-health-baseline/1",
  "entries": [
    {
      "path": "a.go",
      "function": "f",
      "code": "health.complex_method",
      "value": 14
    },
    {
      "path": "a.go",
      "function": "f",
      "code": "health.large_method",
      "value": 55
    },
    {
      "path": "a.go",
      "function": "g",
      "code": "health.complex_method",
      "value": 12
    },
    {
      "path": "b.go",
      "function": "f",
      "code": "health.large_method",
      "value": 70
    }
  ]
}
`
	if string(data) != want {
		t.Fatalf("encoding =\n%s\nwant\n%s", data, want)
	}
	parsed, err := ParseHealthBaseline(data)
	if err != nil {
		t.Fatal(err)
	}
	again, err := EncodeHealthBaseline(parsed.Entries)
	if err != nil || string(again) != want {
		t.Fatalf("re-encoding differs: %v\n%s", err, again)
	}
	empty, err := EncodeHealthBaseline(nil)
	if err != nil || !strings.Contains(string(empty), `"entries": []`) {
		t.Fatalf("an empty baseline must encode entries as []: %v %s", err, empty)
	}
}

// Parsing is strict: another schema string, an unknown field, an unknown or
// non-smell code, an unclean path, an empty function, a missing or non-positive
// value, a duplicate finding (even with another value), a document that is not
// JSON and trailing content are all rejected as ErrHealthBaseline, and an
// unsorted file is accepted and sorted. Of a finding listed twice when
// encoding, the larger value is kept.
// TRLC-LINKS: REQ-EMG-153
func TestParseHealthBaselineIsStrict(t *testing.T) {
	entry := func(p, f, c string) string {
		return `{"path":"` + p + `","function":"` + f + `","code":"` + c + `","value":5}`
	}
	doc := func(schema string, entries ...string) string {
		return `{"schema":"` + schema + `","entries":[` + strings.Join(entries, ",") + `]}`
	}
	good := entry("a.go", "f", "health.large_method")
	bad := map[string]string{
		"wrong schema":                         doc("engmod-health-baseline/2", good),
		"no schema":                            `{"entries":[]}`,
		"unknown field":                        `{"schema":"engmod-health-baseline/1","entries":[],"when":"now"}`,
		"unknown entry key":                    doc("engmod-health-baseline/1", `{"path":"a.go","function":"f","code":"health.large_method","value":5,"line":3}`),
		"no value":                             doc("engmod-health-baseline/1", `{"path":"a.go","function":"f","code":"health.large_method"}`),
		"zero value":                           doc("engmod-health-baseline/1", `{"path":"a.go","function":"f","code":"health.large_method","value":0}`),
		"negative value":                       doc("engmod-health-baseline/1", `{"path":"a.go","function":"f","code":"health.large_method","value":-3}`),
		"string value":                         doc("engmod-health-baseline/1", `{"path":"a.go","function":"f","code":"health.large_method","value":"9"}`),
		"same finding twice with other values": doc("engmod-health-baseline/1", good, `{"path":"a.go","function":"f","code":"health.large_method","value":9}`),
		"unknown code":                         doc("engmod-health-baseline/1", entry("a.go", "f", "health.nope")),
		"waiver_unused":                        doc("engmod-health-baseline/1", entry("a.go", "f", "health.waiver_unused")),
		"absolute path":                        doc("engmod-health-baseline/1", entry("/a.go", "f", "health.large_method")),
		"parent path":                          doc("engmod-health-baseline/1", entry("../a.go", "f", "health.large_method")),
		"unclean path":                         doc("engmod-health-baseline/1", entry("x/../a.go", "f", "health.large_method")),
		"backslash path":                       doc("engmod-health-baseline/1", entry("x\\\\a.go", "f", "health.large_method")),
		"empty function":                       doc("engmod-health-baseline/1", entry("a.go", " ", "health.large_method")),
		"duplicate":                            doc("engmod-health-baseline/1", good, good),
		"not json":                             "schema: engmod-health-baseline/1",
		"trailing content":                     doc("engmod-health-baseline/1", good) + `{}`,
	}
	for name, data := range bad {
		if _, err := ParseHealthBaseline([]byte(data)); !errors.Is(err, ErrHealthBaseline) {
			t.Errorf("%s: err = %v, want ErrHealthBaseline", name, err)
		}
	}
	unsorted := doc("engmod-health-baseline/1", entry("b.go", "f", "health.large_method"), entry("a.go", "f", "health.large_method"))
	parsed, err := ParseHealthBaseline([]byte(unsorted))
	if err != nil || len(parsed.Entries) != 2 || parsed.Entries[0].Path != "a.go" {
		t.Fatalf("unsorted file: %v %+v", err, parsed)
	}
}

// The path of the file is relative to the manifest directory and a clean
// relative file name: absolute paths, .. segments, backslashes, directories and
// the empty path are refused; a missing file and a malformed file are
// ErrHealthBaseline errors that name the file.
// TRLC-LINKS: REQ-EMG-153
func TestLoadHealthBaselineResolvesAgainstTheManifest(t *testing.T) {
	for file, want := range map[string]string{
		"": "is empty", "/etc/x.json": "is absolute", "../x.json": "leaves the manifest directory",
		"a/../../x.json": "leaves the manifest directory", `a\b.json`: "backslash", "dir/": "directory", ".": "directory",
	} {
		if got := HealthBaselinePathProblem(file); !strings.Contains(got, want) {
			t.Errorf("HealthBaselinePathProblem(%q) = %q, want it to contain %q", file, got, want)
		}
	}
	if got := HealthBaselinePathProblem("model/./health-baseline.json"); got != "" {
		t.Errorf("a clean relative path was refused: %q", got)
	}

	dir := t.TempDir()
	manifest := filepath.Join(dir, "engmod.yml")
	if _, err := LoadHealthBaseline(manifest, "model/b.json"); !errors.Is(err, ErrHealthBaseline) || !strings.Contains(err.Error(), "model/b.json") {
		t.Fatalf("a missing file: %v", err)
	}
	if _, err := LoadHealthBaseline(manifest, "../b.json"); !errors.Is(err, ErrHealthBaseline) {
		t.Fatalf("a path outside the manifest directory: %v", err)
	}
	data, _ := EncodeHealthBaseline([]HealthBaselineEntry{{Path: "a.go", Function: "f", Code: "health.large_method", Value: 5}})
	if err := os.MkdirAll(filepath.Join(dir, "model"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "model", "b.json"), data, 0o644); err != nil {
		t.Fatal(err)
	}
	got, err := LoadHealthBaseline(manifest, "model/b.json")
	if err != nil || len(got.Entries) != 1 {
		t.Fatalf("load: %v %+v", err, got)
	}
	if err := os.WriteFile(filepath.Join(dir, "model", "bad.json"), []byte("{"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := LoadHealthBaseline(manifest, "model/bad.json"); !errors.Is(err, ErrHealthBaseline) || !strings.Contains(err.Error(), "model/bad.json") {
		t.Fatalf("a malformed file: %v", err)
	}
}
