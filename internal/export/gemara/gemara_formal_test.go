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

	"github.com/labeth/engineering-model-go/internal/model"
)

const formalFixtureModule = "generated/quint/sm_payments_auth.qnt"
const formalFixtureText = "module sm_payments_auth { val x = 1 }\n"

type formalFixtureEntry struct{ req, invariant, status string }

// formalFixtureModel copies the Quint payments model into a temp dir and, when
// entries is not empty, writes the module and the evidence file; stale records
// a hash that does not match the module. withControl adds one control with a
// control verification.
//
// TRLC-LINKS: REQ-EMG-147
func formalFixtureModel(t *testing.T, entries []formalFixtureEntry, stale, withControl bool) string {
	t.Helper()
	root := t.TempDir()
	src := filepath.Join("internal", "export", "quint", "testdata", "payments")
	if err := os.CopyFS(root, os.DirFS(src)); err != nil {
		t.Fatal(err)
	}
	if withControl {
		assurance := `schemaVersion: 2
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
	if len(entries) == 0 {
		return filepath.Join(root, "engmod.yml")
	}
	sum := sha256.Sum256([]byte(formalFixtureText))
	sha := hex.EncodeToString(sum[:])
	if stale {
		sha = strings.Repeat("a", 64)
	}
	var results []map[string]string
	for _, e := range entries {
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
	return filepath.Join(root, "engmod.yml")
}

// TRLC-LINKS: REQ-EMG-147
func formalEvaluationLog(t *testing.T, manifest string) EvaluationResult {
	t.Helper()
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatalf("load fixture: %v", err)
	}
	res, err := GenerateEvaluationLog(bundle, bundle.Requirements, "", Options{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"})
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

// A passing result for one requirement and a failing one for another give one
// evaluation entry each with the matching result, the machine, invariant and
// result hash in the message and the evidence payload; the log is detected as
// an Evaluation Log, every reference resolves in the metadata and the OSCAL
// conversion accepts it.
//
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func TestFormalEvaluationLogHoldsRequirementKeyedEntries(t *testing.T) {
	manifest := formalFixtureModel(t, []formalFixtureEntry{
		{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"},
		{"REQ-PAYMENTS-002", "INV-PAYMENTS-002", "fail"},
	}, false, false)
	res := formalEvaluationLog(t, manifest)
	if !res.HasContent || len(res.EvaluationLog.Evaluations) != 2 {
		t.Fatalf("want two requirement-keyed evaluations: %s", res.YAML)
	}
	sum := sha256.Sum256([]byte(formalFixtureText))
	sha := hex.EncodeToString(sum[:])

	pass := formalEvaluation(t, res, "REQ-PAYMENTS-001")
	if pass.Result != gemaraproj.Passed || len(pass.AssessmentLogs) != 1 {
		t.Fatalf("pass evaluation: %+v", pass)
	}
	al := pass.AssessmentLogs[0]
	if al.Result != gemaraproj.Passed || !strings.Contains(al.Message, "SM-PAYMENTS-AUTH") || !strings.Contains(al.Message, "INV-PAYMENTS-001") || !strings.Contains(al.Message, sha) || !strings.Contains(al.Description, "analysis") {
		t.Fatalf("pass assessment: %+v", al)
	}
	payload, _ := al.Evidence[0].Payload.(map[string]string)
	if len(al.Evidence) != 1 || payload["qntSha256"] != sha || payload["machine"] != "SM-PAYMENTS-AUTH" || payload["invariant"] != "INV-PAYMENTS-001" || payload["method"] != "analysis" {
		t.Fatalf("evidence payload: %+v", al.Evidence)
	}
	fail := formalEvaluation(t, res, "REQ-PAYMENTS-002")
	if fail.Result != gemaraproj.Failed || fail.AssessmentLogs[0].Result != gemaraproj.Failed || fail.AssessmentLogs[0].Recommendation == "" {
		t.Fatalf("fail evaluation: %+v", fail)
	}
	if res.EvaluationLog.Result != gemaraproj.Failed {
		t.Fatalf("aggregate = %v, want failed", res.EvaluationLog.Result)
	}

	if at, err := gemaraproj.DetectType([]byte(res.YAML)); err != nil || at.String() != "EvaluationLog" {
		t.Fatalf("DetectType = %q (err %v)", at.String(), err)
	}
	refs := map[string]bool{}
	for _, ref := range res.EvaluationLog.Metadata.MappingReferences {
		refs[ref.Id] = true
	}
	if !refs[gemaraRefRequirements] || !refs[gemaraRefControlCatalog] {
		t.Fatalf("mapping references %+v", res.EvaluationLog.Metadata.MappingReferences)
	}
	for _, ce := range res.EvaluationLog.Evaluations {
		if !refs[ce.Control.ReferenceId] {
			t.Fatalf("control reference %q is not declared", ce.Control.ReferenceId)
		}
		for _, l := range ce.AssessmentLogs {
			if l.Requirement.ReferenceId != ce.Control.ReferenceId || len(l.Steps) == 0 || len(l.Applicability) == 0 {
				t.Fatalf("assessment log shape: %+v", l)
			}
			for _, ev := range l.Evidence {
				if ev.Payload == nil || ev.Id == "" || ev.Type == "" || ev.CollectedAt == "" {
					t.Fatalf("evidence shape: %+v", ev)
				}
			}
		}
	}
	// The declared ENGMOD-REQUIREMENTS reference is external to the rendered
	// set: the cross-reference check accepts it, and still rejects a
	// reference that is not declared.
	if errs := CheckCrossReferences(map[string]string{"evaluation-log": res.YAML}); len(errs) != 0 {
		t.Fatalf("cross-reference errors: %+v", errs)
	}
	undeclared := strings.Replace(res.YAML, "id: "+gemaraRefRequirements, "id: ENGMOD-OTHER", 1)
	if errs := CheckCrossReferences(map[string]string{"evaluation-log": undeclared}); len(errs) == 0 {
		t.Fatal("an undeclared requirements reference must be reported")
	}
	bundle, err := model.LoadBundle(manifest)
	if err != nil {
		t.Fatal(err)
	}
	if out, err := GenerateOSCALAssessmentResults(bundle, bundle.Requirements, "", Options{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"}); err != nil || out == "" {
		t.Fatalf("OSCAL conversion of the formal entries: %v (%d bytes)", err, len(out))
	}
}

// A stale result is a not-run entry with the reason and is never passed; a
// requirement with a passing and a failing invariant is failed.
//
// TRLC-LINKS: REQ-EMG-147
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER
func TestFormalEvaluationLogStaleIsNotRun(t *testing.T) {
	res := formalEvaluationLog(t, formalFixtureModel(t, []formalFixtureEntry{{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}}, true, false))
	ce := formalEvaluation(t, res, "REQ-PAYMENTS-001")
	if ce.Result != gemaraproj.NotRun || ce.AssessmentLogs[0].Result != gemaraproj.NotRun || res.EvaluationLog.Result == gemaraproj.Passed {
		t.Fatalf("a stale result must be not-run: %+v", ce)
	}
	if msg := ce.AssessmentLogs[0].Message; !strings.Contains(msg, "Not evaluated") || !strings.Contains(msg, "no longer matches") {
		t.Fatalf("message lacks the reason: %q", msg)
	}

	res = formalEvaluationLog(t, formalFixtureModel(t, []formalFixtureEntry{
		{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}, {"REQ-PAYMENTS-001", "INV-PAYMENTS-002", "fail"},
	}, false, false))
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
	none := formalEvaluationLog(t, formalFixtureModel(t, nil, false, false))
	if none.HasContent || none.YAML != "" {
		t.Fatalf("no controls and no evidence must give no log: %q", none.YAML)
	}

	base := formalEvaluationLog(t, formalFixtureModel(t, nil, false, true))
	if !base.HasContent || len(base.EvaluationLog.Evaluations) != 1 || len(base.EvaluationLog.Metadata.MappingReferences) != 1 || strings.Contains(base.YAML, gemaraRefRequirements) {
		t.Fatalf("control-only log: %s", base.YAML)
	}
	with := formalEvaluationLog(t, formalFixtureModel(t, []formalFixtureEntry{{"REQ-PAYMENTS-001", "INV-PAYMENTS-001", "pass"}}, false, true))
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
