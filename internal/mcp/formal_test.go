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

	"github.com/labeth/engineering-model-go/internal/model"
)

// formalMCPEntry is one evidence entry of a fixture: the module is named by
// machine; stale records a hash that does not match it.
type formalMCPEntry struct {
	req, invariant, status, machine string
	stale                           bool
}

// formalMCPServer builds a fixture model with the evidence entries and
// returns an initialized server over it; no entries means no evidence file.
//
// TRLC-LINKS: REQ-EMG-148
func formalMCPServer(t *testing.T, entries ...formalMCPEntry) *Server {
	t.Helper()
	root := t.TempDir()
	if err := os.CopyFS(root, os.DirFS(filepath.Join("..", "export", "quint", "testdata", "payments"))); err != nil {
		t.Fatal(err)
	}
	// Extra requirements for the roll-up cases.
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
	var results []map[string]string
	for _, e := range entries {
		file := "generated/quint/" + strings.ToLower(strings.ReplaceAll(e.machine, "-", "_")) + ".qnt"
		text := "module " + e.machine + " { val x = 1 }\n"
		sum := sha256.Sum256([]byte(text))
		sha := hex.EncodeToString(sum[:])
		if e.stale {
			sha = strings.Repeat("b", 64)
		}
		path := filepath.Join(root, filepath.FromSlash(file))
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(text), 0o644); err != nil {
			t.Fatal(err)
		}
		results = append(results, map[string]string{
			"requirement": e.req, "status": e.status, "method": "analysis", "kind": "formal",
			"machine": e.machine, "invariant": e.invariant, "mode": "run", "qntFile": file, "qntSha256": sha, "notes": "fixture",
		})
	}
	if len(results) > 0 {
		data, _ := json.Marshal(map[string]any{"schema": "engmod-quint-results/1", "results": results})
		path := filepath.Join(root, "test-results", "quint-formal.json")
		if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	s := NewServer()
	if resp := rpcCall(t, s, map[string]any{
		"jsonrpc": "2.0", "id": 1, "method": "initialize",
		"params": map[string]any{"initializationOptions": map[string]any{
			"modelPath": filepath.Join(root, "engmod.yml"), "repoRoot": root,
		}},
	}); resp["error"] != nil {
		t.Fatalf("initialize: %+v", resp["error"])
	}
	return s
}

// formalEntryOf returns the formalResults or formalGaps entry for a requirement.
//
// TRLC-LINKS: REQ-EMG-148
func formalEntryOf(t *testing.T, payload map[string]any, key, req string) map[string]any {
	t.Helper()
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
	status := callToolPayload(t, s, "verification.status", map[string]any{})
	if _, ok := status["controlVerifications"]; !ok {
		t.Fatalf("authored output must stay: %+v", status)
	}
	for req, want := range map[string]string{"REQ-PAYMENTS-001": "pass", "REQ-PAYMENTS-002": "failed", "REQ-PAYMENTS-003": "stale"} {
		entry := formalEntryOf(t, status, "formalResults", req)
		if entry == nil || entry["result"] != want {
			t.Fatalf("%s: want %s, got %+v", req, want, entry)
		}
	}
	pass := formalEntryOf(t, status, "formalResults", "REQ-PAYMENTS-001")["invariants"].([]any)[0].(map[string]any)
	sum := sha256.Sum256([]byte("module SM-ONE { val x = 1 }\n"))
	if pass["machine"] != "SM-ONE" || pass["invariant"] != "INV-F-1" || pass["mode"] != "run" || pass["requirement"] != "REQ-PAYMENTS-001" ||
		pass["result"] != "pass" || pass["qntSha256"] != hex.EncodeToString(sum[:]) {
		t.Fatalf("invariant entry: %+v", pass)
	}
	stale := formalEntryOf(t, status, "formalResults", "REQ-PAYMENTS-003")["invariants"].([]any)[0].(map[string]any)
	if stale["result"] != "stale" || !strings.Contains(stale["notes"].(string), "no longer matches") {
		t.Fatalf("stale invariant entry: %+v", stale)
	}

	// An entityId narrows to a requirement, a machine or an invariant.
	one := callToolPayload(t, s, "verification.status", map[string]any{"entityId": "REQ-PAYMENTS-002"})
	if list, _ := one["formalResults"].([]any); len(list) != 1 {
		t.Fatalf("entityId REQ-PAYMENTS-002: %+v", one)
	}
	if list, _ := callToolPayload(t, s, "verification.status", map[string]any{"id": "SM-ONE"})["formalResults"].([]any); len(list) != 2 {
		t.Fatalf("id SM-ONE should match two requirements: %+v", list)
	}

	gaps := callToolPayload(t, s, "verification.gaps", map[string]any{})
	if _, ok := gaps["requirementsWithoutFiles"]; !ok {
		t.Fatalf("authored output must stay: %+v", gaps)
	}
	if formalEntryOf(t, gaps, "formalGaps", "REQ-PAYMENTS-002")["result"] != "failed" || formalEntryOf(t, gaps, "formalGaps", "REQ-PAYMENTS-003")["result"] != "stale" {
		t.Fatalf("gaps lack the failed and stale requirements: %+v", gaps["formalGaps"])
	}
	if formalEntryOf(t, gaps, "formalGaps", "REQ-PAYMENTS-001") != nil {
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
	status := callToolPayload(t, s, "verification.status", map[string]any{})
	for req, want := range map[string]string{"REQ-PAYMENTS-004": "failed", "REQ-PAYMENTS-005": "failed", "REQ-PAYMENTS-006": "pass", "REQ-PAYMENTS-007": "stale"} {
		if got := formalEntryOf(t, status, "formalResults", req)["result"]; got != want {
			t.Errorf("%s = %v, want %s", req, got, want)
		}
	}
	if n := len(formalEntryOf(t, status, "formalResults", "REQ-PAYMENTS-006")["invariants"].([]any)); n != 2 {
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
	status := callToolPayload(t, s, "verification.status", map[string]any{})
	gaps := callToolPayload(t, s, "verification.gaps", map[string]any{})
	if _, ok := status["formalResults"]; ok {
		t.Fatalf("verification.status: %+v", status)
	}
	if _, ok := gaps["formalGaps"]; ok {
		t.Fatalf("verification.gaps: %+v", gaps)
	}
	matrix := callToolPayload(t, s, "trace.matrix", map[string]any{})
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
	matrix := callToolPayload(t, s, "trace.matrix", map[string]any{})
	pass := payloadRow(t, matrix, "REQ-PAYMENTS-001")
	if pass["status"] != "verified" || len(pass["verifications"].([]any)) != 1 || pass["evidenceMethods"].([]any)[0] != "analysis" {
		t.Fatalf("pass row: %+v", pass)
	}
	if formal := pass["formal"].([]any)[0].(map[string]any); formal["result"] != "pass" || formal["machine"] != "SM-PAYMENTS-AUTH" {
		t.Fatalf("pass formal: %+v", formal)
	}
	stale := payloadRow(t, matrix, "REQ-PAYMENTS-002")
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
	payload := callToolPayload(t, s, "model.authoringContract", map[string]any{})
	contract := payload["contract"].(map[string]any)
	if contract["contractVersion"] != model.AuthoringContractVersion {
		t.Fatalf("contract version %v", contract["contractVersion"])
	}
	machines, ok := contract["machines"].(map[string]any)
	if !ok {
		t.Fatalf("contract lacks machines: %+v", contract)
	}
	ops := map[string]map[string]any{}
	for _, item := range machines["operators"].([]any) {
		op := item.(map[string]any)
		ops[op["operator"].(string)] = op
	}
	if len(ops) != len(model.MachineOperators()) || ops["implies"]["rightAssociative"] != true || ops["in_state"]["form"] != "call" || ops["<="]["result"] != "Boolean" || ops[":="]["sameType"] != true {
		t.Fatalf("operator table: %+v", ops)
	}
	text, _ := json.Marshal(machines)
	for _, want := range []string{"engquint generate", "engquint check", "qntSha256", "stale", "analysis", "precedence", "reserved"} {
		if !strings.Contains(string(text), want) {
			t.Errorf("machine contract does not mention %q", want)
		}
	}
	rules := payload["rules"].(map[string]any)
	sets := rules["fieldSets"].(map[string]any)
	for _, kind := range []string{"machine", "machineVariable", "machineState", "machineTransition", "machineInvariant"} {
		if _, ok := sets[kind]; !ok {
			t.Errorf("fieldSets lack %s", kind)
		}
	}
	prefixes := contract["stableIdentifiers"].(map[string]any)["prefixes"].(map[string]any)
	if prefixes["SM"] == nil || prefixes["TR"] == nil || prefixes["INV"] == nil {
		t.Errorf("identifier prefixes: %+v", prefixes)
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
	eval := callToolPayload(t, s, "gemara.evaluationLog", map[string]any{})
	if eval["evaluationCount"] != float64(2) || eval["aggregateResult"] != "Failed" {
		t.Fatalf("gemara.evaluationLog: %+v", eval)
	}
	valid := callToolPayload(t, s, "gemara.validate", map[string]any{})
	if errs, _ := valid["crossReferenceErrors"].([]any); len(errs) != 0 || valid["valid"] != true {
		t.Fatalf("gemara.validate: %+v", valid)
	}
}
