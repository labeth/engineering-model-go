// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package gemara

// Requirement-keyed formal entries of the Evaluation Log (REQ-EMG-147). The
// fixture is a temp copy of the Quint payments testdata model with a module
// and an evidence file written next to it; no committed model is loaded.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	gemaraproj "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
)

const formalFixtureModule = "generated/quint/sm_payments_auth.qnt"
const formalFixtureText = codescan.FormalGeneratedHeader + "SM-PAYMENTS-AUTH; do not edit.\nmodule sm_payments_auth { val x = 1 }\n"

type formalFixtureEntry struct{ req, invariant, status string }

// formalFixture describes a fixture model: stale records a hash that does not
// match the module; withControl adds one control with a control verification.
type formalFixture struct {
	entries     []formalFixtureEntry
	stale       bool
	withControl bool
}

// formalTestOptions are the deterministic options of the formal log tests.
var formalTestOptions = Options{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"}

// formalFixtureModel copies the Quint payments model into a temp dir and, when
// the fixture has entries, writes the module and the evidence file.
//
// TRLC-LINKS: REQ-EMG-147
func formalFixtureModel(t *testing.T, fx formalFixture) string {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join("internal", "export", "quint", "testdata", "payments")
	if err := os.CopyFS(root, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	if fx.withControl {
		writeFormalFixtureAssurance(t, root)
	}
	if len(fx.entries) > 0 {
		writeFormalFixtureEvidence(t, root, fx)
	}
	return filepath.Join(root, "engmod.yml")
}

// writeFormalFixtureAssurance adds one control with a control verification.
//
// TRLC-LINKS: REQ-EMG-147
func writeFormalFixtureAssurance(t *testing.T, root string) {
	t.Helper()
	assurance := `schemaVersion: 3
assurance:
  controls:
  - id: CTRL-PAYMENTS-DEMO
    name: Demo control
    category: identity-access
    description: A control for the fixture.
  controlVerifications:
  - id: CV-PAYMENTS-DEMO
    controlRef: CTRL-PAYMENTS-DEMO
    method: test
    status: pass
    lastTested: 2026-04-10
    findings:
    - fixture finding
`
	if err := os.WriteFile(filepath.Join(root, "model", "assurance.yml"), []byte(assurance), 0o644); err != nil {
		t.Fatal(err)
	}
}

// writeFormalFixtureEvidence writes the generated module and the evidence file
// of the fixture entries.
//
// TRLC-LINKS: REQ-EMG-147
func writeFormalFixtureEvidence(t *testing.T, root string, fx formalFixture) {
	t.Helper()
	sum := sha256.Sum256([]byte(formalFixtureText))
	sha := hex.EncodeToString(sum[:])
	if fx.stale {
		sha = strings.Repeat("a", 64)
	}
	var results []map[string]string
	for _, e := range fx.entries {
		results = append(results, map[string]string{
			"requirement": e.req, "status": e.status, "method": "analysis", "kind": "formal",
			"machine": "SM-PAYMENTS-AUTH", "invariant": e.invariant, "mode": "run",
			"qntFile": formalFixtureModule, "qntSha256": sha, "notes": "fixture",
		})
	}
	data, _ := json.Marshal(map[string]any{"schema": "engmod-quint-results/1", "results": results})
	for name, content := range map[string]string{formalFixtureModule: formalFixtureText, "test-results/quint-formal.json": string(data)} {
		path := filepath.Join(root, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

// TRLC-LINKS: REQ-EMG-147
func formalEvaluationLog(t *testing.T, manifest string) EvaluationResult {
	t.Helper()
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	res, err := GenerateEvaluationLog(bundle, bundle.Requirements, EvalRun{CodeRoot: "", Options: formalTestOptions})
	if err != nil {
		t.Fatalf("generate evaluation log: %v", err)
	}
	return res
}

// TRLC-LINKS: REQ-EMG-147
func formalEvaluation(t *testing.T, res EvaluationResult, req string) *gemaraproj.ControlEvaluation {
	t.Helper()
	for _, ce := range res.EvaluationLog.Evaluations {
		if ce.Control.ReferenceId == gemaraRefRequirements && ce.Control.EntryId == req {
			return ce
		}
	}
	t.Fatalf("no formal evaluation for %s in %s", req, res.YAML)
	return nil
}

// formalFixtureSha is the hash of the fixture module.
func formalFixtureSha() string {
	sum := sha256.Sum256([]byte(formalFixtureText))
	return hex.EncodeToString(sum[:])
}

// containsAll reports whether text contains every part.
func containsAll(text string, parts ...string) bool {
	for _, part := range parts {
		if !strings.Contains(text, part) {
			return false
		}
	}
	return true
}

// A passing result for one requirement and a failing one for another give one
// evaluation entry each with the matching result, the machine, invariant and
// result hash in the message and the evidence payload; the log is detected as
// an Evaluation Log, every reference resolves in the metadata and the OSCAL
// conversion accepts it.
//
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func TestFormalEvaluationLogHoldsRequirementKeyedEntries(t *testing.T) {
	manifest := formalFixtureModel(t, formalFixture{entries: []formalFixtureEntry{
		{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"},
		{"REQ-PAYMENTS-002", "INV-PAYMENTS-002", "fail"},
	}})
	res := formalEvaluationLog(t, manifest)
	if !res.HasContent || len(res.EvaluationLog.Evaluations) != 2 {
		t.Fatalf("want two requirement-keyed evaluations: %s", res.YAML)
	}
	assertFormalPassEntry(t, formalEvaluation(t, res, "REQ-PAYMENTS-001"))
	assertFormalFailEntry(t, formalEvaluation(t, res, "REQ-PAYMENTS-002"))
	if res.EvaluationLog.Result != gemaraproj.Failed {
		t.Fatalf("aggregate = %v, want failed", res.EvaluationLog.Result)
	}
	assertFormalLogReferences(t, res)
	assertFormalCrossReferences(t, res)
	assertFormalOSCALConversion(t, manifest)
}

// assertFormalPassEntry checks the passing entry: result, message, payload.
//
// TRLC-LINKS: REQ-EMG-147
func assertFormalPassEntry(t *testing.T, pass *gemaraproj.ControlEvaluation) {
	t.Helper()
	sha := formalFixtureSha()
	if pass.Result != gemaraproj.Passed || len(pass.AssessmentLogs) != 1 {
		t.Fatalf("pass evaluation: %+v", pass)
	}
	al := pass.AssessmentLogs[0]
	if al.Result != gemaraproj.Passed || !containsAll(al.Message, "SM-PAYMENTS-AUTH", "INV-PAYMENTS-001", sha) {
		t.Fatalf("pass assessment: %+v", al)
	}
	if !strings.Contains(al.Description, "analysis") {
		t.Fatalf("pass assessment description: %+v", al)
	}
	assertFormalPassPayload(t, al.Evidence, sha)
}

// assertFormalPassPayload checks the single evidence entry of the passing result.
//
// TRLC-LINKS: REQ-EMG-147
func assertFormalPassPayload(t *testing.T, evidence []gemaraproj.Evidence, sha string) {
	t.Helper()
	if len(evidence) != 1 {
		t.Fatalf("evidence payload: %+v", evidence)
	}
	payload, _ := evidence[0].Payload.(map[string]string)
	want := map[string]string{"qntSha256": sha, "machine": "SM-PAYMENTS-AUTH", "invariant": "INV-PAYMENTS-001", "method": "analysis"}
	for key, value := range want {
		if payload[key] != value {
			t.Fatalf("evidence payload %s: %+v", key, evidence)
		}
	}
}

// assertFormalFailEntry checks the failing entry and its recommendation.
//
// TRLC-LINKS: REQ-EMG-147
func assertFormalFailEntry(t *testing.T, fail *gemaraproj.ControlEvaluation) {
	t.Helper()
	if fail.Result != gemaraproj.Failed || fail.AssessmentLogs[0].Result != gemaraproj.Failed {
		t.Fatalf("fail evaluation: %+v", fail)
	}
	if fail.AssessmentLogs[0].Recommendation == "" {
		t.Fatalf("fail evaluation has no recommendation: %+v", fail)
	}
}

// assertFormalLogReferences checks the type detection, the declared mapping
// references and the shape of every entry.
//
// TRLC-LINKS: REQ-EMG-147
func assertFormalLogReferences(t *testing.T, res EvaluationResult) {
	t.Helper()
	if at, err := gemaraproj.DetectType([]byte(res.YAML)); err != nil || at.String() != "EvaluationLog" {
		t.Fatalf("DetectType = %q (err %v)", at.String(), err)
	}
	refs := declaredMappingReferences(t, res)
	for _, ce := range res.EvaluationLog.Evaluations {
		if !refs[ce.Control.ReferenceId] {
			t.Fatalf("control reference %q is not declared", ce.Control.ReferenceId)
		}
		for _, l := range ce.AssessmentLogs {
			assertFormalAssessmentShape(t, ce, l)
		}
	}
}

// declaredMappingReferences is the set of declared mapping reference ids; it
// requires the requirements and the control catalog among them.
//
// TRLC-LINKS: REQ-EMG-147
func declaredMappingReferences(t *testing.T, res EvaluationResult) map[string]bool {
	t.Helper()
	refs := map[string]bool{}
	for _, ref := range res.EvaluationLog.Metadata.MappingReferences {
		refs[ref.Id] = true
	}
	if !refs[gemaraRefRequirements] || !refs[gemaraRefControlCatalog] {
		t.Fatalf("mapping references %+v", res.EvaluationLog.Metadata.MappingReferences)
	}
	return refs
}

// assertFormalAssessmentShape checks one assessment log and its evidence.
//
// TRLC-LINKS: REQ-EMG-147
func assertFormalAssessmentShape(t *testing.T, ce *gemaraproj.ControlEvaluation, l *gemaraproj.AssessmentLog) {
	t.Helper()
	if l.Requirement.ReferenceId != ce.Control.ReferenceId || len(l.Steps) == 0 || len(l.Applicability) == 0 {
		t.Fatalf("assessment log shape: %+v", l)
	}
	for _, ev := range l.Evidence {
		assertFormalEvidenceShape(t, ev)
	}
}

// assertFormalEvidenceShape checks that an evidence entry has a payload, an id,
// a type and a collection time.
//
// TRLC-LINKS: REQ-EMG-147
func assertFormalEvidenceShape(t *testing.T, ev gemaraproj.Evidence) {
	t.Helper()
	if ev.Payload == nil || ev.Id == "" {
		t.Fatalf("evidence shape: %+v", ev)
	}
	if ev.Type == "" || ev.CollectedAt == "" {
		t.Fatalf("evidence shape: %+v", ev)
	}
}

// assertFormalCrossReferences: the declared ENGMOD-REQUIREMENTS reference is
// external to the rendered set: the cross-reference check accepts it, and
// still rejects a reference that is not declared.
//
// TRLC-LINKS: REQ-EMG-147
func assertFormalCrossReferences(t *testing.T, res EvaluationResult) {
	t.Helper()
	if errs := CheckCrossReferences(map[string]string{"evaluation-log": res.YAML}); len(errs) != 0 {
		t.Fatalf("cross-reference errors: %+v", errs)
	}
	undeclared := strings.Replace(res.YAML, "id: "+gemaraRefRequirements, "id: ENGMOD-OTHER", 1)
	if errs := CheckCrossReferences(map[string]string{"evaluation-log": undeclared}); len(errs) == 0 {
		t.Fatal("an undeclared requirements reference must be reported")
	}
}

// assertFormalOSCALConversion checks that the OSCAL conversion accepts the entries.
//
// TRLC-LINKS: REQ-EMG-147
func assertFormalOSCALConversion(t *testing.T, manifest string) {
	t.Helper()
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := GenerateOSCALAssessmentResults(bundle, bundle.Requirements, EvalRun{CodeRoot: "", Options: formalTestOptions}); err != nil || out == "" {
		t.Fatalf("OSCAL conversion of the formal entries: %v (%d bytes)", err, len(out))
	}
}

// A stale result is a not-run entry with the reason and is never passed; a
// requirement with a passing and a failing invariant is failed.
//
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func TestFormalEvaluationLogStaleIsNotRun(t *testing.T) {
	res := formalEvaluationLog(t, formalFixtureModel(t, formalFixture{entries: []formalFixtureEntry{{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}}, stale: true}))
	ce := formalEvaluation(t, res, "REQ-PAYMENTS-001")
	if ce.Result != gemaraproj.NotRun || ce.AssessmentLogs[0].Result != gemaraproj.NotRun || res.EvaluationLog.Result == gemaraproj.Passed {
		t.Fatalf("a stale result must be not-run: %+v", ce)
	}
	if msg := ce.AssessmentLogs[0].Message; !containsAll(msg, "Not evaluated", "no longer matches") {
		t.Fatalf("message lacks the reason: %q", msg)
	}

	res = formalEvaluationLog(t, formalFixtureModel(t, formalFixture{entries: []formalFixtureEntry{
		{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}, {"REQ-PAYMENTS-001", "INV-PAYMENTS-002", "fail"},
	}}))
	ce = formalEvaluation(t, res, "REQ-PAYMENTS-001")
	if ce.Result != gemaraproj.Failed || len(ce.AssessmentLogs) != 2 {
		t.Fatalf("pass and fail for one requirement: %+v", ce)
	}
}

// Without formal evidence the log is unchanged: no requirement-keyed entry, no
// extra mapping reference, and a model with controls keeps its control entries
// exactly when evidence is added after them.
//
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func TestFormalEvaluationLogAbsentWithoutEvidence(t *testing.T) {
	none := formalEvaluationLog(t, formalFixtureModel(t, formalFixture{}))
	if none.HasContent || none.YAML != "" {
		t.Fatalf("no controls and no evidence must give no log: %q", none.YAML)
	}
	base := formalEvaluationLog(t, formalFixtureModel(t, formalFixture{withControl: true}))
	assertControlOnlyLog(t, base)
	assertControlEntriesKept(t, base)
}

// assertControlOnlyLog checks that a model with controls and no evidence has
// its control entry and no requirement reference.
//
// TRLC-LINKS: REQ-EMG-147
func assertControlOnlyLog(t *testing.T, base EvaluationResult) {
	t.Helper()
	if !base.HasContent || len(base.EvaluationLog.Evaluations) != 1 {
		t.Fatalf("control-only log: %s", base.YAML)
	}
	if len(base.EvaluationLog.Metadata.MappingReferences) != 1 || strings.Contains(base.YAML, gemaraRefRequirements) {
		t.Fatalf("control-only log references: %s", base.YAML)
	}
}

// assertControlEntriesKept checks that adding evidence keeps the control entries.
//
// TRLC-LINKS: REQ-EMG-147
func assertControlEntriesKept(t *testing.T, base EvaluationResult) {
	t.Helper()
	with := formalEvaluationLog(t, formalFixtureModel(t, formalFixture{entries: []formalFixtureEntry{{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}}, withControl: true}))
	if len(with.EvaluationLog.Evaluations) != 2 {
		t.Fatalf("control entry plus one formal entry: %s", with.YAML)
	}
	want, err := marshalGemara(base.EvaluationLog.Evaluations)
	if err != nil {
		t.Fatal(err)
	}
	got, err := marshalGemara(with.EvaluationLog.Evaluations[:1])
	if err != nil {
		t.Fatal(err)
	}
	if got != want {
		t.Fatalf("control entries changed:\n%s\nwant\n%s", got, want)
	}
}

// The record-only step of an entry agrees with its result: a failed entry
// does not name a passing step and a stale one is not claimed as evaluated.
// The step is never run by the exporter and the log leaves steps-executed
// unset (go-gemara reads only the step name and, for OSCAL, steps-executed).
//
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func TestFormalEvaluationStepAgreesWithResult(t *testing.T) {
	pass := formalEvaluationLog(t, formalFixtureModel(t, formalFixture{entries: []formalFixtureEntry{
		{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}, {"REQ-PAYMENTS-002", "INV-PAYMENTS-002", "fail"},
	}}))
	stale := formalEvaluationLog(t, formalFixtureModel(t, formalFixture{entries: []formalFixtureEntry{{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}}, stale: true}))
	for name, c := range map[string]struct {
		res  EvaluationResult
		req  string
		want gemaraproj.Result
	}{
		"pass": {pass, "REQ-PAYMENTS-001", gemaraproj.Passed}, "fail": {pass, "REQ-PAYMENTS-002", gemaraproj.Failed},
		"stale": {stale, "REQ-PAYMENTS-001", gemaraproj.NotRun},
	} {
		al := formalEvaluation(t, c.res, c.req).AssessmentLogs[0]
		if len(al.Steps) != 1 || al.StepsExecuted != 0 || al.Result != c.want {
			t.Fatalf("%s: %+v", name, al)
		}
		if got, _, _ := al.Steps[0](nil); got != al.Result {
			t.Errorf("%s: the step yields %v but the entry is %v", name, got, al.Result)
		}
	}
	if name := formalEvaluation(t, pass, "REQ-PAYMENTS-002").AssessmentLogs[0].Steps[0].String(); !strings.Contains(name, "Failed") {
		t.Errorf("step name of a failed entry: %s", name)
	}
}

// A crafted evidence file naming a module outside the model root, or a file
// that is not a generated module, is NotRun with the reason; an evidence file
// beside the code root is read (the same directories the verification scan
// reads) and entries of the same requirement are ordered by evidence file.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func TestFormalEvaluationLogContainmentAndScanDirectories(t *testing.T) {
	manifest := formalFixtureModel(t, formalFixture{entries: []formalFixtureEntry{{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}}})
	writeCraftedEvidenceBesideCodeRoot(t, manifest)
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	res, err := GenerateEvaluationLog(bundle, bundle.Requirements, EvalRun{CodeRoot: filepath.Join("code", "src"), Options: formalTestOptions})
	if err != nil {
		t.Fatal(err)
	}
	assertCraftedEvidenceIsNotRun(t, formalEvaluation(t, res, "REQ-PAYMENTS-001"))
	// Without the code root the evidence beside it is not read.
	plain := formalEvaluationLog(t, manifest)
	if len(formalEvaluation(t, plain, "REQ-PAYMENTS-001").AssessmentLogs) != 1 {
		t.Errorf("only the model's test-results is read without a code root")
	}
}

// assertCraftedEvidenceIsNotRun checks that both evidence files are read, the
// crafted one first and not run, the current one passed.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-147
func assertCraftedEvidenceIsNotRun(t *testing.T, ce *gemaraproj.ControlEvaluation) {
	t.Helper()
	if len(ce.AssessmentLogs) != 2 {
		t.Fatalf("both evidence files are read: %+v", ce)
	}
	first, second := ce.AssessmentLogs[0], ce.AssessmentLogs[1]
	if first.Result != gemaraproj.NotRun || !strings.Contains(first.Message, "not a local path") {
		t.Errorf("code/test-results sorts first and is crafted: %+v", first)
	}
	if second.Result != gemaraproj.Passed {
		t.Errorf("test-results is current: %+v", second)
	}
}

// writeCraftedEvidenceBesideCodeRoot writes an evidence file naming a module
// outside the model root next to the code root one level down:
// <root>/code/src -> <root>/code/test-results.
//
// TRLC-LINKS: REQ-EMG-145, REQ-EMG-147
func writeCraftedEvidenceBesideCodeRoot(t *testing.T, manifest string) {
	t.Helper()
	root := filepath.Dir(manifest)
	evidence, err := os.ReadFile(filepath.Join(root, "test-results", "quint-formal.json"))
	if err != nil {
		t.Fatal(err)
	}
	crafted := strings.ReplaceAll(string(evidence), formalFixtureModule, "../../../../dev/zero")
	beside := filepath.Join(root, "code", "test-results")
	if err := os.MkdirAll(beside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(beside, "quint-formal.json"), []byte(crafted), 0o644); err != nil {
		t.Fatal(err)
	}
}
