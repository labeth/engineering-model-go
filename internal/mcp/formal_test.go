// ENGMODEL-OWNER-UNIT: FU-MCP-SERVER
package mcp

// Requirement-level formal (Quint) results in the verification tools
// (REQ-EMG-148). The fixtures are temp copies of the Quint payments testdata
// model with modules and an evidence file written beside it.

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/codescan"
	"github.com/labeth/engineering-model-go/internal/model"
)

// formalMCPEntry is one evidence entry of a fixture: the module is named by
// machine; stale records a hash that does not match it.
type formalMCPEntry struct {
	req, invariant, status, machine string
	stale                           bool
}

// formalMCPModuleText is the generated module of a machine in the fixtures.
//
// TRLC-LINKS: REQ-EMG-148
func formalMCPModuleText(machine string) string {
	return codescan.FormalGeneratedHeader + machine + "; do not edit.\nmodule " + machine + " { val x = 1 }\n"
}

// formalMCPServer builds a fixture model with the evidence entries and
// returns an initialized server over it; no entries means no evidence file.
//
// TRLC-LINKS: REQ-EMG-148
func formalMCPServer(t *testing.T, entries ...formalMCPEntry) *Server {
	t.Helper()
	root := formalFixtureRoot(t)
	writeFormalEvidence(t, root, entries)
	return initializeModelAt(t, root)
}

// formalFixtureRoot copies the Quint payments model into a temp directory and
// adds the extra requirements of the roll-up cases.
func formalFixtureRoot(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(filepath.Join("..", "export", "quint", "testdata", "payments"))); err != nil {
		t.Fatal(err)
	}
	reqPath := filepath.Join(root, "model", "requirements.yml")
	reqs, err := os.ReadFile(reqPath)
	if err != nil {
		t.Fatal(err)
	}
	extra := string(reqs)
	for _, id := range []string{"003", "004", "005", "006", "007"} {
		extra += "- id: REQ-PAYMENTS-" + id + "\n  text: When authorization is granted, the payment service shall finish the authorization.\n  appliesTo:\n  - FU-PAYMENTS-AUTH\n"
	}
	if err := os.WriteFile(reqPath, []byte(extra), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// writeFormalEvidence writes the module of every entry and the evidence file
// listing them; no entries writes no evidence file.
func writeFormalEvidence(t *testing.T, root string, entries []formalMCPEntry) {
	t.Helper()
	var results []map[string]string
	for _, e := range entries {
		results = append(results, writeFormalModule(t, root, e))
	}
	if len(results) == 0 {
		return
	}
	data, _ := json.Marshal(map[string]any{"schema": "engmod-quint-results/1", "results": results})
	writeMCPFile(t, filepath.Join(root, "test-results", "quint-formal.json"), string(data))
}

// writeFormalModule writes the generated module of one entry and returns its
// evidence result.
func writeFormalModule(t *testing.T, root string, e formalMCPEntry) map[string]string {
	t.Helper()
	file := "generated/quint/" + strings.ToLower(strings.ReplaceAll(e.machine, "-", "_")) + ".qnt"
	text := formalMCPModuleText(e.machine)
	sum := sha256.Sum256([]byte(text))
	sha := hex.EncodeToString(sum[:])
	if e.stale {
		sha = strings.Repeat("b", 64)
	}
	writeMCPFile(t, filepath.Join(root, filepath.FromSlash(file)), text)
	return map[string]string{
		"requirement": e.req, "status": e.status, "method": "analysis", "kind": "formal",
		"machine": e.machine, "invariant": e.invariant, "mode": "run", "qntFile": file, "qntSha256": sha, "notes": "fixture",
	}
}

// formalEntryOf returns the formalResults or formalGaps entry for a requirement.
//
// TRLC-LINKS: REQ-EMG-148
func formalEntryOf(payload map[string]any, key, req string) map[string]any {
	list, _ := payload[key].([]any)
	for _, item := range list {
		if m := item.(map[string]any); m["requirement"] == req {
			return m
		}
	}
	return nil
}

// A requirement with a passing, a failing and a stale formal result is
// reported pass, failed and stale by verification.status with the machine,
// invariant, mode, requirement and qntSha256; verification.gaps lists the
// failed and stale ones and not the passing one.
//
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestVerificationToolsReportFormalPassFailedStale(t *testing.T) {
	s := formalMCPServer(t,
		formalMCPEntry{req: "REQ-PAYMENTS-001", invariant: "INV-F-1", status: "pass", machine: "SM-ONE"},
		formalMCPEntry{req: "REQ-PAYMENTS-002", invariant: "INV-F-2", status: "fail", machine: "SM-ONE"},
		formalMCPEntry{req: "REQ-PAYMENTS-003", invariant: "INV-F-3", status: "pass", machine: "SM-TWO", stale: true},
	)
	status := s.callToolPayload(t, "verification.status", map[string]any{})
	if _, ok := status["controlVerifications"]; !ok {
		t.Fatalf("authored output must stay: %+v", status)
	}
	assertFormalResults(t, status)
	assertFormalInvariantEntries(t, status)
	assertFormalEntityNarrowing(t, s)
	assertFormalGaps(t, s.callToolPayload(t, "verification.gaps", map[string]any{}))
}

// assertFormalResults checks the result of each requirement in formalResults.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertFormalResults(t *testing.T, status map[string]any) {
	t.Helper()
	for req, want := range map[string]string{"REQ-PAYMENTS-001": "pass", "REQ-PAYMENTS-002": "failed", "REQ-PAYMENTS-003": "stale"} {
		entry := formalEntryOf(status, "formalResults", req)
		if entry == nil || entry["result"] != want {
			t.Fatalf("%s: want %s, got %+v", req, want, entry)
		}
	}
}

// assertFormalInvariantEntries checks the invariant entry of the passing and
// of the stale requirement.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertFormalInvariantEntries(t *testing.T, status map[string]any) {
	t.Helper()
	pass := formalEntryOf(status, "formalResults", "REQ-PAYMENTS-001")["invariants"].([]any)[0].(map[string]any)
	sum := sha256.Sum256([]byte(formalMCPModuleText("SM-ONE")))
	want := map[string]any{"machine": "SM-ONE", "invariant": "INV-F-1", "mode": "run", "requirement": "REQ-PAYMENTS-001", "result": "pass", "qntSha256": hex.EncodeToString(sum[:])}
	for key, value := range want {
		if pass[key] != value {
			t.Fatalf("invariant entry: %+v", pass)
		}
	}
	stale := formalEntryOf(status, "formalResults", "REQ-PAYMENTS-003")["invariants"].([]any)[0].(map[string]any)
	if stale["result"] != "stale" || !strings.Contains(stale["notes"].(string), "no longer matches") {
		t.Fatalf("stale invariant entry: %+v", stale)
	}
}

// assertFormalEntityNarrowing checks that an entityId narrows verification
// status to a requirement, a machine or an invariant.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertFormalEntityNarrowing(t *testing.T, s *Server) {
	t.Helper()
	one := s.callToolPayload(t, "verification.status", map[string]any{"entityId": "REQ-PAYMENTS-002"})
	if list, _ := one["formalResults"].([]any); len(list) != 1 {
		t.Fatalf("entityId REQ-PAYMENTS-002: %+v", one)
	}
	if list, _ := s.callToolPayload(t, "verification.status", map[string]any{"entityId": "SM-ONE"})["formalResults"].([]any); len(list) != 2 {
		t.Fatalf("entityId SM-ONE should match two requirements: %+v", list)
	}
}

// assertFormalGaps checks that verification.gaps lists the failed and stale
// requirements and not the passing one.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertFormalGaps(t *testing.T, gaps map[string]any) {
	t.Helper()
	if _, ok := gaps["requirementsWithoutFiles"]; !ok {
		t.Fatalf("authored output must stay: %+v", gaps)
	}
	if formalEntryOf(gaps, "formalGaps", "REQ-PAYMENTS-002")["result"] != "failed" || formalEntryOf(gaps, "formalGaps", "REQ-PAYMENTS-003")["result"] != "stale" {
		t.Fatalf("gaps lack the failed and stale requirements: %+v", gaps["formalGaps"])
	}
	if formalEntryOf(gaps, "formalGaps", "REQ-PAYMENTS-001") != nil {
		t.Fatalf("a passing requirement is no gap: %+v", gaps["formalGaps"])
	}
}

// A requirement whose only invariant fails is failed; one with a failing and a
// passing invariant is failed; one with a passing and a stale invariant is
// pass (stale does not downgrade); one with only a stale invariant is stale.
//
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER
func TestFormalRequirementResultRollUp(t *testing.T) {
	s := formalMCPServer(t,
		formalMCPEntry{req: "REQ-PAYMENTS-004", invariant: "INV-A1", status: "fail", machine: "SM-ONE"},
		formalMCPEntry{req: "REQ-PAYMENTS-005", invariant: "INV-B1", status: "pass", machine: "SM-ONE"},
		formalMCPEntry{req: "REQ-PAYMENTS-005", invariant: "INV-B2", status: "fail", machine: "SM-ONE"},
		formalMCPEntry{req: "REQ-PAYMENTS-006", invariant: "INV-C1", status: "pass", machine: "SM-ONE"},
		formalMCPEntry{req: "REQ-PAYMENTS-006", invariant: "INV-C2", status: "pass", machine: "SM-TWO", stale: true},
		formalMCPEntry{req: "REQ-PAYMENTS-007", invariant: "INV-D1", status: "pass", machine: "SM-TWO", stale: true},
	)
	status := s.callToolPayload(t, "verification.status", map[string]any{})
	for req, want := range map[string]string{"REQ-PAYMENTS-004": "failed", "REQ-PAYMENTS-005": "failed", "REQ-PAYMENTS-006": "pass", "REQ-PAYMENTS-007": "stale"} {
		if got := formalEntryOf(status, "formalResults", req)["result"]; got != want {
			t.Errorf("%s = %v, want %s", req, got, want)
		}
	}
	if n := len(formalEntryOf(status, "formalResults", "REQ-PAYMENTS-006")["invariants"].([]any)); n != 2 {
		t.Errorf("REQ-PAYMENTS-006 lists %d invariants, want both", n)
	}
}

// Without formal evidence the responses of verification.status and
// verification.gaps and the trace.matrix rows carry none of the new keys.
//
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER
func TestVerificationToolsUnchangedWithoutFormalEvidence(t *testing.T) {
	s := formalMCPServer(t)
	status := s.callToolPayload(t, "verification.status", map[string]any{})
	gaps := s.callToolPayload(t, "verification.gaps", map[string]any{})
	if _, ok := status["formalResults"]; ok {
		t.Fatalf("verification.status: %+v", status)
	}
	if _, ok := gaps["formalGaps"]; ok {
		t.Fatalf("verification.gaps: %+v", gaps)
	}
	matrix := s.callToolPayload(t, "trace.matrix", map[string]any{})
	for _, item := range matrix["requirements"].([]any) {
		row := item.(map[string]any)
		if _, ok := row["formal"]; ok {
			t.Fatalf("row %v gained formal: %+v", row["id"], row)
		}
		if _, ok := row["evidenceMethods"]; ok {
			t.Fatalf("row %v gained evidenceMethods: %+v", row["id"], row)
		}
	}
}

// trace.matrix rows carry the formal results and the method analysis: a
// passing current result is verified evidence, a stale one is none.
//
// TRLC-LINKS: REQ-EMG-146, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func TestTraceMatrixRowsCarryFormalEvidence(t *testing.T) {
	s := formalMCPServer(t,
		formalMCPEntry{req: "REQ-PAYMENTS-001", invariant: "INV-PAYMENTS-001", status: "pass", machine: "SM-PAYMENTS-AUTH"},
		formalMCPEntry{req: "REQ-PAYMENTS-002", invariant: "INV-PAYMENTS-002", status: "pass", machine: "SM-OTHER", stale: true},
	)
	matrix := s.callToolPayload(t, "trace.matrix", map[string]any{})
	assertFormalPassRow(t, payloadRow(t, matrix, "REQ-PAYMENTS-001"))
	assertFormalStaleRow(t, payloadRow(t, matrix, "REQ-PAYMENTS-002"))
}

// assertFormalPassRow checks the verified row of a passing current result.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertFormalPassRow(t *testing.T, pass map[string]any) {
	t.Helper()
	if pass["status"] != "verified" || len(pass["verifications"].([]any)) != 1 || pass["evidenceMethods"].([]any)[0] != "analysis" {
		t.Fatalf("pass row: %+v", pass)
	}
	if formal := pass["formal"].([]any)[0].(map[string]any); formal["result"] != "pass" || formal["machine"] != "SM-PAYMENTS-AUTH" {
		t.Fatalf("pass formal: %+v", formal)
	}
}

// assertFormalStaleRow checks that a stale result is no verification evidence.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-ALLOCATION-TRACE
func assertFormalStaleRow(t *testing.T, stale map[string]any) {
	t.Helper()
	if stale["status"] != "orphan" || len(stale["verifications"].([]any)) != 0 || stale["formal"].([]any)[0].(map[string]any)["result"] != "stale" {
		t.Fatalf("stale row: %+v", stale)
	}
	if _, ok := stale["evidenceMethods"]; ok {
		t.Fatalf("a stale result is no analysis evidence: %+v", stale)
	}
}

// model.authoringContract documents the machine contract: the operator table
// with precedence and operand types, the grammar, the Quint workflow and the
// evidence semantics, beside the field sets of the validation rules.
//
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func TestAuthoringContractDocumentsTheMachineContract(t *testing.T) {
	s := formalMCPServer(t)
	payload := s.callToolPayload(t, "model.authoringContract", map[string]any{})
	contract := payload["contract"].(map[string]any)
	if contract["contractVersion"] != model.AuthoringContractVersion {
		t.Fatalf("contract version %v", contract["contractVersion"])
	}
	machines, ok := contract["machines"].(map[string]any)
	if !ok {
		t.Fatalf("contract lacks machines: %+v", contract)
	}
	assertMachineOperators(t, machines)
	assertMachineContractText(t, machines)
	assertMachineFieldSets(t, payload["rules"].(map[string]any)["fieldSets"].(map[string]any))
	prefixes := contract["stableIdentifiers"].(map[string]any)["prefixes"].(map[string]any)
	if prefixes["SM"] == nil || prefixes["TR"] == nil || prefixes["INV"] == nil {
		t.Errorf("identifier prefixes: %+v", prefixes)
	}
}

// assertMachineFieldSets checks that the field sets cover the machine kinds.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func assertMachineFieldSets(t *testing.T, sets map[string]any) {
	t.Helper()
	for _, kind := range []string{"machine", "machineVariable", "machineState", "machineTransition", "machineInvariant"} {
		if _, ok := sets[kind]; !ok {
			t.Errorf("fieldSets lack %s", kind)
		}
	}
}

// assertMachineOperators checks the operator table of the machine contract.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func assertMachineOperators(t *testing.T, machines map[string]any) {
	t.Helper()
	ops := map[string]map[string]any{}
	for _, item := range machines["operators"].([]any) {
		op := item.(map[string]any)
		ops[op["operator"].(string)] = op
	}
	if len(ops) != len(model.MachineOperators()) || !machineOperatorShapesHold(ops) {
		t.Fatalf("operator table: %+v", ops)
	}
}

// machineOperatorShapesHold reports whether the implies, in_state, <= and :=
// operators carry their documented shape.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func machineOperatorShapesHold(ops map[string]map[string]any) bool {
	if ops["implies"]["rightAssociative"] != true || ops["in_state"]["form"] != "call" {
		return false
	}
	return ops["<="]["result"] == "Boolean" && ops[":="]["sameType"] == true
}

// assertMachineContractText checks that the machine contract mentions the
// workflow and evidence terms.
//
// ENGMODEL-LINKS: FU-MCP-SERVER, DO-MODEL-AUTHORING-CONTRACT
func assertMachineContractText(t *testing.T, machines map[string]any) {
	t.Helper()
	text, _ := json.Marshal(machines)
	for _, want := range []string{"engquint generate", "engquint check", "qntSha256", "stale", "analysis", "precedence", "reserved"} {
		if !strings.Contains(string(text), want) {
			t.Errorf("machine contract does not mention %q", want)
		}
	}
}

// gemara.validate accepts the evaluation log with requirement-keyed formal
// entries (the cross-reference check) and gemara.evaluationLog counts them.
//
// TRLC-LINKS: REQ-EMG-147, REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER, FU-GEMARA-EXPORTER
func TestGemaraToolsAcceptFormalEvaluationEntries(t *testing.T) {
	s := formalMCPServer(t,
		formalMCPEntry{req: "REQ-PAYMENTS-001", invariant: "INV-PAYMENTS-001", status: "pass", machine: "SM-PAYMENTS-AUTH"},
		formalMCPEntry{req: "REQ-PAYMENTS-002", invariant: "INV-PAYMENTS-002", status: "fail", machine: "SM-PAYMENTS-AUTH"},
	)
	eval := s.callToolPayload(t, "gemara.evaluationLog", map[string]any{})
	if eval["evaluationCount"] != float64(2) || eval["aggregateResult"] != "Failed" {
		t.Fatalf("gemara.evaluationLog: %+v", eval)
	}
	valid := s.callToolPayload(t, "gemara.validate", map[string]any{})
	if errs, _ := valid["crossReferenceErrors"].([]any); len(errs) != 0 || valid["valid"] != true {
		t.Fatalf("gemara.validate: %+v", valid)
	}
}

// When the trace matrix cannot be built the tools say so with formalError and
// carry no formalResults or formalGaps, so "no formal gaps" is never reported
// for a verification that did not run.
//
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER
func TestVerificationToolsReportFormalErrorWhenMatrixFails(t *testing.T) {
	s := formalMCPServer(t, formalMCPEntry{req: "REQ-PAYMENTS-001", invariant: "INV-F-1", status: "fail", machine: "SM-ONE"})
	// Break only the matrix build: the fingerprint is refreshed so the tool call
	// does not reload the model (and fail there) first.
	s.requirementsPath = filepath.Join(t.TempDir(), "missing-requirements.yml")
	s.fingerprint = s.modelFileFingerprint()

	if _, err := s.formalGaps(); err == nil {
		t.Fatalf("formalGaps must return the matrix error")
	}
	if _, err := s.formalStatus(""); err == nil {
		t.Fatalf("formalStatus must return the matrix error")
	}
	for _, tool := range []string{"verification.status", "verification.gaps"} {
		payload := s.callToolPayload(t, tool, map[string]any{})
		msg, _ := payload["formalError"].(string)
		if !strings.Contains(msg, "trace matrix could not be built") {
			t.Errorf("%s: formalError = %q in %+v", tool, msg, payload)
		}
		if _, ok := payload["formalGaps"]; ok {
			t.Errorf("%s reports gaps next to an error: %+v", tool, payload)
		}
		if _, ok := payload["formalResults"]; ok {
			t.Errorf("%s reports results next to an error: %+v", tool, payload)
		}
	}
}

// The tools look for evidence in the directories the verification scan reads:
// a file in a nested directory of test-results is found by hasFormalEvidence
// and by the matrix rows alike, and one beside the model (outside test-results)
// is found by neither.
//
// TRLC-LINKS: REQ-EMG-148
// ENGMODEL-LINKS: FU-MCP-SERVER
func TestFormalEvidenceDirectoriesAgreeWithTheVerificationScan(t *testing.T) {
	s := formalMCPServer(t, formalMCPEntry{req: "REQ-PAYMENTS-001", invariant: "INV-F-1", status: "pass", machine: "SM-ONE"})
	root := codescan.ModelRootDir(*s.bundle)
	nested := filepath.Join(root, "test-results", "ci", "quint-formal.json")
	moveEvidenceFile(t, filepath.Join(root, "test-results", "quint-formal.json"), nested)
	if !s.hasFormalEvidence() {
		t.Fatalf("a nested evidence file is in the scanned directories")
	}
	assertNestedEvidenceRows(t, s)
	moveEvidenceFile(t, nested, filepath.Join(root, "quint-formal.json"))
	if s.hasFormalEvidence() {
		t.Fatalf("an evidence file outside test-results is not scanned")
	}
	if got, err := s.formalRequirements(); err != nil || got != nil {
		t.Fatalf("no evidence in the scanned directories: %+v (%v)", got, err)
	}
}

// assertNestedEvidenceRows checks that the matrix rows carry the nested
// evidence file.
//
// ENGMODEL-LINKS: FU-MCP-SERVER
func assertNestedEvidenceRows(t *testing.T, s *Server) {
	t.Helper()
	got, err := s.formalRequirements()
	if err != nil || len(got) != 1 || got[0].Invariants[0].Evidence != "test-results/ci/quint-formal.json" {
		t.Fatalf("matrix rows: %+v (%v)", got, err)
	}
}

// moveEvidenceFile renames an evidence file, creating the target directory.
//
// ENGMODEL-LINKS: FU-MCP-SERVER
func moveEvidenceFile(t *testing.T, from, to string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(to), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Rename(from, to); err != nil {
		t.Fatal(err)
	}
}
