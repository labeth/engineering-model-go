// ENGMODEL-OWNER-UNIT: FU-GEMARA-EXPORTER
package engmodel

import (
	"path/filepath"
	"strings"
	"testing"

	gemara "github.com/gemaraproj/go-gemara"

	"github.com/labeth/engineering-model-go/internal/model"
)

// gemaraTruthBundle is a minimal in-memory model with one control, one
// verification, two risks and two POA&M items. Callers mutate it per test.
func gemaraTruthBundle(method string) model.Bundle {
	var b model.Bundle
	b.Architecture.Model = model.ModelMeta{ID: "truth-model", Title: "Truth Model"}
	a := &b.Architecture.AuthoredArchitecture
	a.Controls = []model.Control{{ID: "CTRL-A", Name: "Control A", Category: "access", Description: "Restrict access."}}
	a.ControlVerifications = []model.ControlVerification{{ID: "CV-A", ControlRef: "CTRL-A", RiskRefs: []string{"RISK-A"}, Method: method, Status: "pass"}}
	a.Risks = []model.Risk{
		{ID: "RISK-A", Title: "Risk A", Likelihood: "medium", Impact: "high", Response: "mitigate"},
		{ID: "RISK-B", Title: "Risk B", Likelihood: "low", Impact: "low", Response: "mitigate"},
	}
	a.POAMItems = []model.POAMItem{
		{ID: "POAM-A", RiskRef: "RISK-A", Milestone: "fix", DueDate: "2026-12-01", Status: "in-progress"},
		{ID: "POAM-B", RiskRef: "RISK-B"},
	}
	return b
}

func generateTruth(t *testing.T, b model.Bundle) GemaraExportResult {
	t.Helper()
	res, err := GenerateGemara(b, GemaraExportOptions{AuthorName: "Generator Name", Date: "2026-06-26T00:00:00Z"})
	if err != nil {
		t.Fatalf("generate gemara: %v", err)
	}
	if !res.HasPolicy {
		t.Fatalf("expected a policy")
	}
	return res
}

// A model without owners must not present the generator's author as the
// accountable policy contact.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraPolicyContactsNotSetWithoutOwners(t *testing.T) {
	res := generateTruth(t, gemaraTruthBundle("test"))
	for role, contacts := range map[string][]gemara.Contact{"responsible": res.Policy.Contacts.Responsible, "accountable": res.Policy.Contacts.Accountable} {
		if len(contacts) != 1 || contacts[0].Name != "not set" {
			t.Fatalf("%s contacts = %+v, want a single \"not set\" contact", role, contacts)
		}
	}
	if res.Policy.Metadata.Author.Name != "Generator Name" {
		t.Fatalf("metadata.author.name = %q, want the generator author", res.Policy.Metadata.Author.Name)
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "gemara.policy_contacts_not_set" {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected gemara.policy_contacts_not_set diagnostic, got %+v", res.Diagnostics)
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraPolicyContactsFromModelOwners(t *testing.T) {
	b := gemaraTruthBundle("test")
	a := &b.Architecture.AuthoredArchitecture
	a.Actors = []model.Actor{{ID: "ACT-SEC", Name: "Security Lead"}}
	b.Catalog.Catalog.Actors = []model.CatalogEntry{{ID: "ACT-OPS", Name: "operations engineer"}}
	a.Risks[0].Owner = "ACT-SEC"
	a.ControlVerifications[0].Owner = "ACT-OPS"
	a.Risks[1].Owner = "ACT-SEC"
	res := generateTruth(t, b)
	got := []string{}
	for _, c := range res.Policy.Contacts.Accountable {
		got = append(got, c.Name)
	}
	if strings.Join(got, ",") != "Security Lead,operations engineer" {
		t.Fatalf("accountable contacts = %v", got)
	}
	if len(res.Policy.Contacts.Responsible) != 2 {
		t.Fatalf("responsible contacts = %+v", res.Policy.Contacts.Responsible)
	}
	for _, d := range res.Diagnostics {
		if d.Code == "gemara.policy_contacts_not_set" {
			t.Fatalf("unexpected not-set diagnostic with owners present")
		}
	}
}

// The policy claims no automated gate or continuous evaluation the model does
// not state; automated intent evaluation appears only for test verifications.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraPolicyHasNoUnsupportedAdherenceClaims(t *testing.T) {
	res := generateTruth(t, gemaraTruthBundle("analysis"))
	pol := res.Policy
	for _, m := range pol.Adherence.EnforcementMethods {
		if m.Type == gemara.MethodGate || m.Id == "ENM-GATE" {
			t.Fatalf("policy claims a gate enforcement method: %+v", m)
		}
		if m.Mode == gemara.ModeAutomated {
			t.Fatalf("policy claims automated enforcement: %+v", m)
		}
	}
	for _, m := range pol.Adherence.EvaluationMethods {
		if m.Mode == gemara.ModeAutomated {
			t.Fatalf("analysis-only model must not claim automated evaluation: %+v", m)
		}
	}
	for _, ap := range pol.Adherence.AssessmentPlans {
		for _, m := range ap.EvaluationMethods {
			if m.Mode == gemara.ModeAutomated {
				t.Fatalf("assessment plan %s claims automated evaluation", ap.Id)
			}
		}
	}
	if pol.ImplementationPlan.EvaluationTimeline.Notes != "not set in the model" || pol.ImplementationPlan.EnforcementTimeline.Notes != "not set in the model" {
		t.Fatalf("timeline notes = %q / %q", pol.ImplementationPlan.EvaluationTimeline.Notes, pol.ImplementationPlan.EnforcementTimeline.Notes)
	}
	doc := res.YAML["policy"]
	for _, claim := range []string{"continuous", "ENM-GATE", "Gate", "block release"} {
		if strings.Contains(doc, claim) {
			t.Fatalf("policy YAML contains unsupported claim %q:\n%s", claim, doc)
		}
	}

	auto := generateTruth(t, gemaraTruthBundle("test")).Policy
	hasAutomatedIntent := false
	for _, m := range auto.Adherence.EvaluationMethods {
		if m.Id == "EM-INTENT" && m.Mode == gemara.ModeAutomated {
			hasAutomatedIntent = true
		}
	}
	if !hasAutomatedIntent {
		t.Fatalf("test verification should yield an automated EM-INTENT method: %+v", auto.Adherence.EvaluationMethods)
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraRiskCategoryAppetiteMarkedAsDefault(t *testing.T) {
	res := generateTruth(t, gemaraTruthBundle("test"))
	if len(res.RiskCatalog.Groups) != 1 {
		t.Fatalf("risk categories = %+v", res.RiskCatalog.Groups)
	}
	g := res.RiskCatalog.Groups[0]
	if !strings.Contains(g.Description, "Risk appetite not set in the model; default applied") {
		t.Fatalf("risk category description does not disclose the default: %q", g.Description)
	}
	if g.MaxSeverity != gemara.InvalidSeverity || strings.Contains(res.YAML["risk-catalog"], "max-severity") {
		t.Fatalf("max-severity must stay unset, got %v", g.MaxSeverity)
	}
}

// The enforcement log references only evaluation-log entries that exist and
// does not invent a status.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraEnforcementLogHasNoDanglingReferences(t *testing.T) {
	b := gemaraTruthBundle("test")
	res := generateTruth(t, b)
	// Gemara requires every action to cite at least one assessment: POAM-A
	// (risk with a verification) becomes an action, POAM-B (no evaluated
	// risk) is left out and reported, rather than an action citing nothing.
	if !res.HasEnforcement || len(res.EnforcementLog.Actions) != 1 {
		t.Fatalf("expected one enforcement action, got %d", len(res.EnforcementLog.Actions))
	}
	first := res.EnforcementLog.Actions[0]
	if len(first.Justification.Assessments) != 1 || first.Justification.Assessments[0].Log.EntryId != "CV-A" {
		t.Fatalf("POAM-A should cite evaluation entry CV-A, got %+v", first.Justification.Assessments)
	}
	if first.Justification.Assessments[0].Result != gemara.Passed {
		t.Fatalf("finding result should mirror the verification status, got %v", first.Justification.Assessments[0].Result)
	}
	if first.Message == nil || strings.Contains(*first.Message, "status: open") {
		t.Fatalf("unexpected status default in %v", first.Message)
	}
	unjustified := false
	for _, d := range res.Diagnostics {
		if d.Code == "gemara.enforcement_unjustified" && strings.Contains(d.Message, "POAM-B") {
			unjustified = true
		}
	}
	if !unjustified {
		t.Fatalf("POAM-B must be reported as unjustified: %+v", res.Diagnostics)
	}
	for _, action := range res.EnforcementLog.Actions {
		if len(action.Justification.Assessments) == 0 {
			t.Fatalf("an enforcement action without an assessment violates the Gemara schema: %+v", action)
		}
	}
	// An action for an item without a status says so instead of inventing one.
	unset := gemaraTruthBundle("test")
	unset.Architecture.AuthoredArchitecture.POAMItems = []model.POAMItem{{ID: "POAM-U", RiskRef: "RISK-A"}}
	if msg := generateTruth(t, unset).EnforcementLog.Actions[0].Message; msg == nil || !strings.Contains(*msg, "status: not set") {
		t.Fatalf("unexpected status default in %v", msg)
	}
	// Without any evaluated risk there is no enforcement log at all (Gemara
	// requires at least one action).
	none := gemaraTruthBundle("test")
	none.Architecture.AuthoredArchitecture.ControlVerifications = nil
	if generateTruth(t, none).HasEnforcement {
		t.Fatal("an enforcement log without justified actions must not be rendered")
	}

	evalRes, err := GenerateGemaraEvaluationLog(b, model.RequirementsDocument{}, "", GemaraExportOptions{Date: "2026-06-26T00:00:00Z"})
	if err != nil {
		t.Fatalf("evaluation log: %v", err)
	}
	docs := map[string]string{"evaluation-log": evalRes.YAML}
	for k, v := range res.YAML {
		docs[k] = v
	}
	if errs := CheckGemaraCrossReferences(docs); len(errs) != 0 {
		t.Fatalf("cross-reference errors: %+v", errs)
	}
}

// Every example model's rendered artifact set must be internally resolvable.
// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraCrossReferencesResolveForExamples(t *testing.T) {
	models := append([]string{"engmod.yml"}, gemaraExampleModels...)
	for _, modelPath := range models {
		modelPath := modelPath
		t.Run(filepath.Base(filepath.Dir(modelPath)), func(t *testing.T) {
			docs := renderGemaraSet(t, modelPath)
			if errs := CheckGemaraCrossReferences(docs); len(errs) != 0 {
				t.Fatalf("cross-reference errors: %+v", errs)
			}
		})
	}
}

// TRLC-LINKS: REQ-EMG-015
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraCrossReferenceCheckReportsInjectedDanglingRef(t *testing.T) {
	docs := renderGemaraSet(t, filepath.Join("examples", "payments-engineering-sample", "engmod.yml"))
	risks := docs["risk-catalog"]
	if !strings.Contains(risks, "TS-PAYMENTS-RISK-TICKET-TAMPER") {
		t.Fatalf("fixture changed: risk catalog no longer cites TS-PAYMENTS-RISK-TICKET-TAMPER")
	}
	docs["risk-catalog"] = strings.Replace(risks, "TS-PAYMENTS-RISK-TICKET-TAMPER", "TS-DOES-NOT-EXIST", 1)
	errs := CheckGemaraCrossReferences(docs)
	if len(errs) == 0 {
		t.Fatalf("expected a dangling reference error")
	}
	found := false
	for _, e := range errs {
		if e.Artifact == "risk-catalog" && e.EntryId == "TS-DOES-NOT-EXIST" {
			found = true
		}
	}
	if !found {
		t.Fatalf("dangling TS-DOES-NOT-EXIST not reported: %+v", errs)
	}

	if !strings.Contains(docs["policy"], "reference-id: ENGMOD-CONTROL-CATALOG") {
		t.Fatalf("fixture changed: policy no longer imports ENGMOD-CONTROL-CATALOG")
	}
	docs["policy"] = strings.Replace(docs["policy"], "reference-id: ENGMOD-CONTROL-CATALOG", "reference-id: ENGMOD-MISSING-CATALOG", 1)
	errs = CheckGemaraCrossReferences(docs)
	found = false
	for _, e := range errs {
		if e.Artifact == "policy" && e.ReferenceId == "ENGMOD-MISSING-CATALOG" {
			found = true
		}
	}
	if !found {
		t.Fatalf("unresolved reference-id not reported: %+v", errs)
	}
}

func renderGemaraSet(t *testing.T, modelPath string) map[string]string {
	t.Helper()
	bundle, err := model.LoadBundle(modelPath)
	if err != nil {
		t.Fatalf("load bundle: %v", err)
	}
	opts := GemaraExportOptions{Version: "1.0.0", Date: "2026-06-26T00:00:00Z"}
	res, err := GenerateGemara(bundle, opts)
	if err != nil {
		t.Fatalf("generate gemara: %v", err)
	}
	docs := map[string]string{}
	for k, v := range res.YAML {
		docs[k] = v
	}
	evalRes, err := GenerateGemaraEvaluationLog(bundle, bundle.Requirements, "", opts)
	if err != nil {
		t.Fatalf("evaluation log: %v", err)
	}
	if evalRes.HasContent {
		docs["evaluation-log"] = evalRes.YAML
	}
	return docs
}

// An enforcement action is Enforced only for a completed POA&M item and
// Tolerated only for an accepted one; planned, in-progress, deferred and unset
// items are Undetermined, and so is the aggregate while any item is open.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraEnforcementDispositionFollowsPOAMStatus(t *testing.T) {
	b := gemaraTruthBundle("test")
	a := &b.Architecture.AuthoredArchitecture
	a.POAMItems = []model.POAMItem{
		{ID: "POAM-1", RiskRef: "RISK-A", Milestone: "m", Status: "completed"},
		{ID: "POAM-2", RiskRef: "RISK-A", Milestone: "m", Status: "accepted"},
		{ID: "POAM-3", RiskRef: "RISK-A", Milestone: "m", Status: "planned"},
		{ID: "POAM-4", RiskRef: "RISK-A", Milestone: "m", Status: "in-progress"},
		{ID: "POAM-5", RiskRef: "RISK-A", Milestone: "m", Status: "deferred"},
		{ID: "POAM-6", RiskRef: "RISK-A", Milestone: "m"},
	}
	// Each item has an evaluated risk (RISK-A, CV-A), so each is an action.
	res := generateTruth(t, b)
	want := []gemara.Disposition{gemara.DispositionEnforced, gemara.DispositionTolerated, gemara.DispositionUndetermined, gemara.DispositionUndetermined, gemara.DispositionUndetermined, gemara.DispositionUndetermined}
	if len(res.EnforcementLog.Actions) != len(want) {
		t.Fatalf("actions = %d, want %d", len(res.EnforcementLog.Actions), len(want))
	}
	for i, w := range want {
		if got := res.EnforcementLog.Actions[i].Disposition; got != w {
			t.Errorf("%s disposition = %v, want %v", a.POAMItems[i].ID, got, w)
		}
	}
	if res.EnforcementLog.Disposition != gemara.DispositionUndetermined {
		t.Fatalf("aggregate with open items = %v, want Undetermined", res.EnforcementLog.Disposition)
	}
	if !strings.Contains(res.YAML["enforcement-log"], "disposition: Undetermined") {
		t.Fatalf("Undetermined must serialize:\n%s", res.YAML["enforcement-log"])
	}

	for _, tc := range []struct {
		statuses []string
		want     gemara.Disposition
	}{
		{[]string{"planned", "planned"}, gemara.DispositionUndetermined},
		{[]string{"accepted", "accepted"}, gemara.DispositionTolerated},
		{[]string{"completed", "accepted"}, gemara.DispositionEnforced},
		{[]string{"completed", "completed"}, gemara.DispositionEnforced},
	} {
		a.POAMItems = nil
		for i, st := range tc.statuses {
			a.POAMItems = append(a.POAMItems, model.POAMItem{ID: "POAM-" + string(rune('A'+i)), RiskRef: "RISK-A", Milestone: "m", Status: st})
		}
		if got := generateTruth(t, b).EnforcementLog.Disposition; got != tc.want {
			t.Errorf("aggregate for %v = %v, want %v", tc.statuses, got, tc.want)
		}
	}
}

// Policy risks.mitigated lists only closed risks; accepted lists risks with
// status accepted or response accept; open and mitigating risks are in neither
// list and a diagnostic says so.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraPolicyRisksFollowRiskStatus(t *testing.T) {
	b := gemaraTruthBundle("test")
	a := &b.Architecture.AuthoredArchitecture
	a.Risks = []model.Risk{
		{ID: "RISK-OPEN", Title: "o", Likelihood: "low", Impact: "low", Status: "open", Response: "mitigate"},
		{ID: "RISK-MITIGATING", Title: "m", Likelihood: "low", Impact: "low", Status: "mitigating", Response: "mitigate"},
		{ID: "RISK-CLOSED", Title: "c", Likelihood: "low", Impact: "low", Status: "closed", Response: "mitigate"},
		{ID: "RISK-ACCEPTED", Title: "a", Likelihood: "low", Impact: "low", Status: "accepted", Response: "mitigate"},
		{ID: "RISK-ACCEPT-RESPONSE", Title: "r", Likelihood: "low", Impact: "low", Status: "open", Response: "accept", Rationale: "cheap"},
	}
	a.ControlVerifications[0].RiskRefs = []string{"RISK-OPEN"}
	a.POAMItems = []model.POAMItem{{ID: "POAM-A", RiskRef: "RISK-OPEN", Milestone: "m", Status: "planned"}}
	res := generateTruth(t, b)
	var mitigated, accepted []string
	for _, r := range res.Policy.Risks.Mitigated {
		mitigated = append(mitigated, r.Risk.EntryId)
	}
	for _, r := range res.Policy.Risks.Accepted {
		accepted = append(accepted, r.Risk.EntryId)
	}
	if strings.Join(mitigated, ",") != "RISK-CLOSED" {
		t.Fatalf("mitigated = %v, want only RISK-CLOSED", mitigated)
	}
	if strings.Join(accepted, ",") != "RISK-ACCEPTED,RISK-ACCEPT-RESPONSE" {
		t.Fatalf("accepted = %v", accepted)
	}
	found := false
	for _, d := range res.Diagnostics {
		if d.Code == "gemara.policy_risks_untreated" && strings.Contains(d.Message, "RISK-OPEN") && strings.Contains(d.Message, "RISK-MITIGATING") {
			found = true
		}
	}
	if !found {
		t.Fatalf("expected gemara.policy_risks_untreated naming the open risks, got %+v", res.Diagnostics)
	}
}

// The default metadata author is the generator, typed Software; an explicit
// AuthorType overrides it.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraGeneratorAuthorIsSoftware(t *testing.T) {
	res, err := GenerateGemara(gemaraTruthBundle("test"), GemaraExportOptions{Date: "2026-06-26T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	au := res.Policy.Metadata.Author
	if au.Id != "engmod" || au.Type != gemara.Software {
		t.Fatalf("default author = %+v, want engmod typed Software", au)
	}
	if !strings.Contains(res.YAML["policy"], "type: Software") {
		t.Fatalf("policy YAML must carry author type Software")
	}
	res, err = GenerateGemara(gemaraTruthBundle("test"), GemaraExportOptions{AuthorName: "Jane Doe", AuthorType: gemara.Human, Date: "2026-06-26T00:00:00Z"})
	if err != nil {
		t.Fatal(err)
	}
	if res.Policy.Metadata.Author.Type != gemara.Human {
		t.Fatalf("AuthorType override ignored: %+v", res.Policy.Metadata.Author)
	}
}

// The model consistency check passes on the rendered set and catches an
// Enforced action for an open POA&M item, a non-closed risk listed as
// mitigated, and a non-accepted risk listed as accepted.
// TRLC-LINKS: REQ-EMG-015, REQ-EMG-062
// ENGMODEL-LINKS: FU-GEMARA-EXPORTER, CTRL-TRACEABILITY-COVERAGE
func TestGemaraModelConsistencyCatchesOverclaims(t *testing.T) {
	b := gemaraTruthBundle("test")
	a := &b.Architecture.AuthoredArchitecture
	a.Risks[0].Status = "open"
	a.Risks[1].Status = "closed"
	res := generateTruth(t, b)
	docs := map[string]string{}
	for k, v := range res.YAML {
		docs[k] = v
	}
	if errs := CheckGemaraModelConsistency(b, docs); len(errs) != 0 {
		t.Fatalf("rendered set must be consistent, got %+v", errs)
	}
	enf := docs["enforcement-log"]
	if !strings.Contains(enf, "disposition: Undetermined") {
		t.Fatalf("fixture changed: no Undetermined action:\n%s", enf)
	}
	docs["enforcement-log"] = strings.ReplaceAll(enf, "disposition: Undetermined", "disposition: Enforced")
	pol := docs["policy"]
	docs["policy"] = strings.Replace(pol, "entry-id: RISK-B", "entry-id: RISK-A", 1)
	errs := CheckGemaraModelConsistency(b, docs)
	var enforced, mitigated bool
	for _, e := range errs {
		if e.Artifact == "enforcement-log" && strings.Contains(e.Message, "POAM-A") {
			enforced = true
		}
		if e.Artifact == "policy" && strings.Contains(e.Message, "RISK-A") {
			mitigated = true
		}
	}
	if !enforced || !mitigated {
		t.Fatalf("expected enforcement and policy overclaims, got %+v", errs)
	}
}
