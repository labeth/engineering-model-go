// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/validate"
)

// completionValues returns the candidate values and the candidates by value.
// TRLC-LINKS: REQ-EMG-069
func completionValues(t *testing.T, payload map[string]any) ([]string, map[string]map[string]any) {
	t.Helper()
	rows, ok := payload["candidates"].([]any)
	if !ok {
		t.Fatalf("model.complete payload has no candidates: %+v", payload)
	}
	values := make([]string, 0, len(rows))
	byValue := map[string]map[string]any{}
	for _, item := range rows {
		row, _ := item.(map[string]any)
		value, _ := row["value"].(string)
		values = append(values, value)
		byValue[value] = row
	}
	return values, byValue
}

// Local IDs are matched by a case-insensitive prefix and returned sorted.
// Mutation: a case-sensitive prefix match returns no FU-CHECKOUT for "fu-check".
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT
func TestModelCompleteLocalIDsByPrefix(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "fu-check"})
	values, byValue := completionValues(t, payload)
	row, ok := byValue["FU-CHECKOUT"]
	if !ok {
		t.Fatalf("expected FU-CHECKOUT for prefix fu-check, got %v", values)
	}
	if row["kind"] != "functional_unit" || row["source"] != "local" {
		t.Fatalf("FU-CHECKOUT candidate has wrong kind/source: %+v", row)
	}
	for _, v := range values {
		if !strings.HasPrefix(strings.ToLower(v), "fu-check") {
			t.Fatalf("candidate %q does not match prefix fu-check", v)
		}
	}
	if !sort.SliceIsSorted(values, func(i, j int) bool { return strings.ToLower(values[i]) < strings.ToLower(values[j]) }) {
		t.Fatalf("candidates are not sorted: %v", values)
	}
	limited := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "", "max": "2"})
	if got, _ := completionValues(t, limited); len(got) != 2 || limited["truncated"] != true {
		t.Fatalf("max=2 must return 2 candidates and truncated=true, got %v truncated=%v", got, limited["truncated"])
	}
}

// An enum field returns exactly its allowed values; a bare field name collects
// the values of every enum ending in that name.
// Mutation: ignoring the field returns entity IDs alongside the enum values.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func TestModelCompleteEnumValuesForField(t *testing.T) {
	s := initializedPaymentsServer(t)
	rules := validate.AuthoringRules()
	payload := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "", "field": "risk.status"})
	values, _ := completionValues(t, payload)
	want := append([]string(nil), rules.Enums["risk.status"]...)
	sort.Strings(want)
	sort.Strings(values)
	if strings.Join(values, ",") != strings.Join(want, ",") {
		t.Fatalf("risk.status values: got %v want %v", values, want)
	}
	status := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "PRO", "field": "status"})
	statusValues, byValue := completionValues(t, status)
	if _, ok := byValue["proposed"]; !ok {
		t.Fatalf("field status prefix PRO must include requirement status proposed, got %v", statusValues)
	}
	fields, _ := byValue["proposed"]["fields"].([]any)
	if len(fields) == 0 || !containsAny(fields, "requirement.status") {
		t.Fatalf("proposed must name requirement.status among its fields: %+v", byValue["proposed"])
	}
}

// TRLC-LINKS: REQ-EMG-069
func containsAny(items []any, want string) bool {
	for _, item := range items {
		if item == want {
			return true
		}
	}
	return false
}

// relationship.type returns the allowed mapping types with their kind pairs.
// Mutation: dropping the pairs leaves "contains" without from/to kinds.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func TestModelCompleteRelationshipTypes(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "con", "field": "relationship.type"})
	values, byValue := completionValues(t, payload)
	row, ok := byValue["contains"]
	if !ok {
		t.Fatalf("expected relationship type contains, got %v", values)
	}
	pairs, _ := row["pairs"].([]any)
	if row["kind"] != "relationship.type" || len(pairs) == 0 {
		t.Fatalf("contains must carry kind relationship.type and its pairs: %+v", row)
	}
	first, _ := pairs[0].(map[string]any)
	if first["from"] == "" || first["to"] == "" {
		t.Fatalf("pair without from/to: %+v", first)
	}
	for _, v := range values {
		if strings.HasPrefix(v, "FU-") || strings.HasPrefix(v, "REQ-") {
			t.Fatalf("relationship.type must not return entity IDs, got %q", v)
		}
	}
}

// ID fields return only entities of the expected kind.
// Mutation: ignoring the field-to-kind map returns non-actor IDs for owner.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER
func TestModelCompleteIDFieldFiltersKinds(t *testing.T) {
	s := initializedPaymentsServer(t)
	payload := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "", "field": "owner", "max": "500"})
	values, byValue := completionValues(t, payload)
	if len(values) == 0 {
		t.Fatalf("owner must return actor IDs")
	}
	for _, v := range values {
		if byValue[v]["kind"] != "actor" {
			t.Fatalf("owner candidate %q has kind %v, want actor", v, byValue[v]["kind"])
		}
	}
	catalog := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "pay", "field": "catalog"})
	names, rows := completionValues(t, catalog)
	if len(names) == 0 {
		t.Fatalf("catalog field must return catalog term names")
	}
	for _, n := range names {
		if kind, _ := rows[n]["kind"].(string); !strings.HasPrefix(kind, "catalog.") {
			t.Fatalf("catalog candidate %q has kind %v", n, rows[n]["kind"])
		}
	}
}

// writeAuthoringCompositionFixture writes a parent that imports a child which
// publishes a functional unit, with or without the workspace replacement.
// TRLC-LINKS: REQ-EMG-069
func writeAuthoringCompositionFixture(t *testing.T, withWorkspace bool) string {
	t.Helper()
	root := t.TempDir()
	child := filepath.Join(root, "child")
	parent := filepath.Join(root, "parent")
	writeMCPModule(t, child, "example.com/child@v1", "v1.0.0", "CHILD", `publications:
  - id: public
    architecture: [FU-CHILD-UNIT, CAP-CHILD]
`, `  functionalGroups:
    - id: FG-CHILD
      name: Child group
  functionalUnits:
    - id: FU-CHILD-UNIT
      name: Child unit
      group: FG-CHILD
  contract:
    provides:
      - id: CAP-CHILD
        kind: capability
        ref: REQ-CHILD
`)
	writeMCPModule(t, parent, "example.com/parent@v1", "v1.0.0", "PARENT", `dependencies:
  - alias: child
    path: example.com/child@v1
    version: v1.0.0
    publications: [public]
`, `  functionalGroups:
    - id: FG-PARENT
      name: Parent group
  functionalUnits:
    - id: FU-PARENT-UNIT
      name: Parent unit
      group: FG-PARENT
  composition:
    subsystems:
      - id: SUB-CHILD
        dependency: child
        publication: public
`)
	if withWorkspace {
		writeMCPFile(t, filepath.Join(root, "engmod.work.yml"), "schemaVersion: 2\nreplacements:\n  - module: example.com/child@v1\n    path: ./child\n")
	}
	return parent
}

// TRLC-LINKS: REQ-EMG-069
func initializedServerAt(t *testing.T, modelDir string) *Server {
	t.Helper()
	s := NewServer()
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath": filepath.Join(modelDir, "engmod.yml"),
			"repoRoot":  modelDir,
		}},
	})
	if resp["error"] != nil {
		t.Fatalf("initialize: %+v", resp["error"])
	}
	return s
}

// Imported entities of a selected publication complete as alias::ID, also
// when the prefix names only the ID part.
// Mutation: skipping the composition projection returns no child:: candidates.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func TestModelCompleteImportedQualifiedIDs(t *testing.T) {
	t.Setenv("CUE_REGISTRY", "none")
	parent := writeAuthoringCompositionFixture(t, true)
	s := initializedServerAt(t, parent)
	payload := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "child::"})
	values, byValue := completionValues(t, payload)
	row, ok := byValue["child::FU-CHILD-UNIT"]
	if !ok {
		t.Fatalf("expected child::FU-CHILD-UNIT, got %v (note %v)", values, payload["note"])
	}
	if row["source"] != "child" || row["kind"] != "functional_unit" || row["title"] != "Child unit" {
		t.Fatalf("imported candidate has wrong source/kind/title: %+v", row)
	}
	byTail := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "fu-", "field": "appliesTo"})
	tailValues, _ := completionValues(t, byTail)
	if !containsString(tailValues, "child::FU-CHILD-UNIT") || !containsString(tailValues, "FU-PARENT-UNIT") {
		t.Fatalf("appliesTo prefix fu- must list local and imported units, got %v", tailValues)
	}
	if containsString(tailValues, "child::FG-CHILD") {
		t.Fatalf("unpublished child entity must not be offered: %v", tailValues)
	}
}

// Without a workspace replacement or cached module, offline completion still
// returns local candidates and explains why imports are missing.
// Mutation: returning the composition error as a tool error fails this test.
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-SYSTEM-COMPOSITION
func TestModelCompleteDegradesWhenImportsUnresolved(t *testing.T) {
	t.Setenv("CUE_REGISTRY", "none")
	parent := writeAuthoringCompositionFixture(t, false)
	s := initializedServerAt(t, parent)
	payload := callToolPayload(t, s, "model.complete", map[string]any{"prefix": "FU-"})
	values, _ := completionValues(t, payload)
	if !containsString(values, "FU-PARENT-UNIT") {
		t.Fatalf("local candidates missing: %v", values)
	}
	for _, v := range values {
		if strings.Contains(v, "::") {
			t.Fatalf("no imports expected offline, got %q", v)
		}
	}
	if note, _ := payload["note"].(string); !strings.Contains(note, "import") {
		t.Fatalf("expected a note about unresolved imports, got %q", note)
	}
}

// snapshotDir records every file's size and modification time below dir.
// TRLC-LINKS: REQ-EMG-069
func snapshotDir(t *testing.T, dir string) map[string]string {
	t.Helper()
	out := map[string]string{}
	err := filepath.WalkDir(dir, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		info, err := d.Info()
		if err != nil {
			return err
		}
		out[path] = fmt.Sprintf("%d %d %v", info.Size(), info.ModTime().UnixNano(), info.IsDir())
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
	return out
}

// copyPaymentsModel copies the payments sample manifest and model documents.
// TRLC-LINKS: REQ-EMG-069
func copyPaymentsModel(t *testing.T) string {
	t.Helper()
	src := filepath.Join("..", "examples", "payments-engineering-sample")
	dst := t.TempDir()
	files := []string{"engmod.yml"}
	entries, err := os.ReadDir(filepath.Join(src, "model"))
	if err != nil {
		t.Fatal(err)
	}
	for _, e := range entries {
		files = append(files, filepath.Join("model", e.Name()))
	}
	for _, f := range files {
		data, err := os.ReadFile(filepath.Join(src, f))
		if err != nil {
			t.Fatal(err)
		}
		writeMCPFile(t, filepath.Join(dst, f), string(data))
	}
	return dst
}

// TRLC-LINKS: REQ-EMG-069
func diagnosticCodes(payload map[string]any, key string) []string {
	rows, _ := payload[key].([]any)
	codes := []string{}
	for _, item := range rows {
		row, _ := item.(map[string]any)
		code, _ := row["code"].(string)
		codes = append(codes, code)
	}
	return codes
}

// TRLC-LINKS: REQ-EMG-069
func callToolRaw(t *testing.T, s *Server, name string, args map[string]any) (map[string]any, bool) {
	t.Helper()
	resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": name, "method": "tools/call",
		"params": map[string]any{"name": name, "arguments": args},
	})
	if resp["error"] != nil {
		t.Fatalf("%s rpc error: %+v", name, resp["error"])
	}
	result, _ := resp["result"].(map[string]any)
	isErr, _ := result["isError"].(bool)
	content, _ := result["content"].([]any)
	chunk, _ := content[0].(map[string]any)
	var payload map[string]any
	if err := json.Unmarshal([]byte(chunk["text"].(string)), &payload); err != nil {
		t.Fatal(err)
	}
	return payload, isErr
}

// model.diagnose validates supplied content in memory: unchanged content has no
// errors, an unknown reference is attributed to the document with its line, a
// YAML syntax error is a parse error with line and column, an unknown document
// is a tool error, and the model directory and temp directory are left untouched.
// Mutations: validating the on-disk document instead of content (no
// invalid_requirement_source_ref); writing content in place or leaving the
// temp copy behind (snapshot/temp check); accepting any path as document;
// dropping the schema-error position (no line on the status: banana error).
// TRLC-LINKS: REQ-EMG-069
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MCP-TOOL-RESULT, CTRL-MCP-PATH-BOUNDARY
func TestModelDiagnoseValidatesContentInMemory(t *testing.T) {
	tmp := t.TempDir()
	t.Setenv("TMPDIR", tmp)
	modelDir := copyPaymentsModel(t)
	s := initializedServerAt(t, modelDir)
	before := snapshotDir(t, modelDir)
	original, err := os.ReadFile(filepath.Join(modelDir, "model", "requirements.yml"))
	if err != nil {
		t.Fatal(err)
	}

	valid := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "requirements", "content": string(original)})
	if valid["errors"] != float64(0) || valid["valid"] != true {
		t.Fatalf("unchanged requirements must have 0 errors: %+v", valid)
	}
	if valid["path"] != "model/requirements.yml" {
		t.Fatalf("diagnose must report the document path, got %v", valid["path"])
	}

	broken := strings.Replace(string(original), "- id: REQ-PAY-002\n", "- id: REQ-PAY-002\n  sourceRefs: [REQ-DOES-NOT-EXIST]\n", 1)
	if broken == string(original) {
		t.Fatal("fixture edit did not apply")
	}
	bad := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "model/requirements.yml", "content": broken})
	if !containsString(diagnosticCodes(bad, "diagnostics"), "model.invalid_requirement_source_ref") || bad["valid"] != false {
		t.Fatalf("unknown sourceRef must be reported for the document: %+v", bad)
	}
	for _, item := range bad["diagnostics"].([]any) {
		row := item.(map[string]any)
		if row["code"] == "model.invalid_requirement_source_ref" {
			wantLine := float64(strings.Count(broken[:strings.Index(broken, "- id: REQ-PAY-002")], "\n") + 1)
			if line, _ := row["line"].(float64); line < wantLine || line > wantLine+1 {
				t.Fatalf("diagnostic line %v, want about %v: %+v", row["line"], wantLine, row)
			}
		}
	}

	schema := strings.Replace(string(original), "- id: REQ-PAY-002\n", "- id: REQ-PAY-002\n  status: banana\n", 1)
	schemaResult := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "requirements", "content": schema})
	schemaRows, _ := schemaResult["diagnostics"].([]any)
	if len(schemaRows) == 0 || schemaResult["valid"] != false {
		t.Fatalf("schema violation must be reported for the document: %+v", schemaResult)
	}
	wantSchemaLine := float64(strings.Count(schema[:strings.Index(schema, "status: banana")], "\n") + 1)
	if row := schemaRows[0].(map[string]any); row["line"] != wantSchemaLine || strings.Contains(toString(row["message"]), os.TempDir()) {
		t.Fatalf("schema error must carry line %v and no temp path: %+v", wantSchemaLine, row)
	}

	behavior, err := os.ReadFile(filepath.Join(modelDir, "model", "behavior.yml"))
	if err != nil {
		t.Fatal(err)
	}
	dangling := strings.Replace(string(behavior), "  relationships:\n", "  relationships:\n  - type: contains\n    from: FG-PAYMENTS\n    to: FU-DOES-NOT-EXIST\n", 1)
	behaviorResult := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "behavior", "content": dangling})
	if behaviorResult["errors"] == float64(0) {
		t.Fatalf("dangling relationship must be an error attributed to behavior: %+v", behaviorResult)
	}

	syntax := "schemaVersion: 2\nrequirements:\n  - id: [unclosed\n"
	parse := callToolPayload(t, s, "model.diagnose", map[string]any{"document": "requirements", "content": syntax})
	parseErrors, _ := parse["parseErrors"].([]any)
	if len(parseErrors) == 0 || parse["valid"] != false {
		t.Fatalf("syntax error must be reported as a parse error: %+v", parse)
	}
	first := parseErrors[0].(map[string]any)
	if first["line"] != float64(3) || first["column"] == nil {
		t.Fatalf("parse error must carry line 3 and a column: %+v", first)
	}

	for _, document := range []string{"secrets", "../engmod.yml", "engmod.yml", "model/unknown.yml"} {
		payload, isErr := callToolRaw(t, s, "model.diagnose", map[string]any{"document": document, "content": "x: 1\n"})
		if !isErr {
			t.Fatalf("document %q is not a manifest document and must be rejected: %+v", document, payload)
		}
	}

	if after := snapshotDir(t, modelDir); fmt.Sprint(after) != fmt.Sprint(before) {
		t.Fatalf("model.diagnose changed the model directory")
	}
	if left, _ := os.ReadDir(tmp); len(left) != 0 {
		t.Fatalf("model.diagnose left temp files: %v", left)
	}
}
