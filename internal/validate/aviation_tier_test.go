// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

// aviationFixture names the tier, the software level, the state of the five plans and the coverage metric of an aviation model.
type aviationFixture struct{ tier, level, planState, metric string }

// aviationTierBundle is a model at tier with the safety bundle and a DAL
// software level aviation document: the five plans at planState, one
// execution, review, approval and independence record (people as actors),
// and one coverage record of metric.
func aviationTierBundle(f aviationFixture) model.Bundle {
	tier, level, planState, metric := f.tier, f.level, f.planState, f.metric
	b := tierBundle(tier, "safety")
	plans := []string{"PLAN-SOFTWARE-ASPECTS-CERTIFICATION", "SOFTWARE-DEVELOPMENT-PLAN", "SOFTWARE-VERIFICATION-PLAN", "SOFTWARE-CONFIGURATION-MANAGEMENT-PLAN", "SOFTWARE-QUALITY-ASSURANCE-PLAN"}
	var data []model.LifecycleDataRecord
	for i, family := range model.RecognizedLifecycleDataFamilies() {
		state := "executed"
		if i < len(plans) {
			state = planState
		}
		data = append(data, model.LifecycleDataRecord{ID: "DATA-" + strings.ToUpper(family.CategoryCode[:6]) + "-" + string(rune('A'+i)), CategoryCode: family.CategoryCode, CategoryName: family.CategoryName, Owner: "Lead", State: state})
	}
	b.Aviation = &model.AviationDocument{SchemaVersion: model.CurrentSchemaVersion, Aviation: model.AviationProfile{
		Profile:       model.AirborneProfile{ID: "AV", SoftwareLevel: level},
		LifecycleData: data,
		Executions:    []model.VerificationExecution{{ID: "EXEC-1", Producer: "Engineer A", Verifier: "Reviewer B", Status: "executed"}},
		Reviews:       []model.AirborneReview{{ID: "REVIEW-1", Reviewer: "Reviewer B", State: "reviewed"}},
		Approvals:     []model.AirborneApproval{{ID: "APPROVAL-1", Approver: "Manager C", State: "accepted"}},
		Independence:  []model.IndependenceRecord{{ID: "INDEP-1", Producer: "Engineer A", Verifier: "Reviewer B", State: "reviewed"}},
		Coverage:      []model.StructuralCoverage{{ID: "COV-1", Metric: metric, Status: "executed"}},
	}}
	b.Manifest.Documents.Aviation = "model/aviation.yml"
	return b
}

// pathsOf returns the paths of the diagnostics with the code.
func pathsOf(diags []Diagnostic, code string) []string {
	var out []string
	for _, d := range codesOf(diags, code) {
		out = append(out, d.Path)
	}
	return out
}

// The safety bundle's evidence floors by tier: none below mvp; plans at
// least planned at mvp; plans reviewed, executions executed and coverage of
// the level's metric (A mcdc, B decision, C statement) at product; every
// record accepted at operational. not-applicable satisfies a floor only
// with a rationale. aviation.state_below_tier names each record below.
// TRLC-LINKS: REQ-EMG-132
func TestAviationFloorsByTierAndLevel(t *testing.T) {
	assertFloorFindings(t, aviationFixture{"prototype", "C", "declared", "statement"}, 0)
	assertFloorFindings(t, aviationFixture{"mvp", "C", "declared", "statement"}, 5)
	assertFloorFindings(t, aviationFixture{"mvp", "C", "planned", "statement"}, 0)
	assertFloorFindings(t, aviationFixture{"product", "C", "planned", "statement"}, 5)
	assertFloorFindings(t, aviationFixture{"product", "C", "reviewed", "statement"}, 0)
	assertFloorFindings(t, aviationFixture{"product", "D", "reviewed", ""}, 0)
	assertCoverageFloorByLevel(t)
	assertExecutionFloor(t)
	assertOperationalFloor(t)
	assertNotApplicableFloor(t)
	foundation := aviationTierBundle(aviationFixture{"operational", "C", "declared", "statement"})
	foundation.Manifest.Lifecycle.Bundles = nil
	if got := pathsOf(Bundle(foundation), "aviation.state_below_tier"); len(got) != 0 {
		t.Errorf("without the safety bundle: %v", got)
	}
}

// assertFloorFindings checks the number of aviation.state_below_tier findings
// of a model; five findings are the five plans.
func assertFloorFindings(t *testing.T, f aviationFixture, want int) {
	t.Helper()
	got := pathsOf(Bundle(aviationTierBundle(f)), "aviation.state_below_tier")
	if len(got) != want {
		t.Errorf("%+v: %v, want %d findings", f, got, want)
	}
	if want == 5 && len(got) > 0 && !strings.Contains(got[0], "aviation.lifecycleData.DATA-PLAN-S") {
		t.Errorf("%+v: %v, want the five plans", f, got)
	}
}

// assertCoverageFloorByLevel checks that a software level A, B or C without
// coverage of its metric has one finding at aviation.coverage naming the metric.
func assertCoverageFloorByLevel(t *testing.T) {
	t.Helper()
	for level, metric := range map[string]string{"A": "mcdc", "B": "decision", "C": "statement"} {
		other := "statement"
		if metric == "statement" {
			other = "decision"
		}
		got := codesOf(Bundle(aviationTierBundle(aviationFixture{"product", level, "reviewed", other})), "aviation.state_below_tier")
		if len(got) != 1 || got[0].Path != "aviation.coverage" || !strings.Contains(got[0].Message, metric) {
			t.Errorf("level %s without %s coverage: %+v", level, metric, got)
		}
	}
}

// assertExecutionFloor checks that a planned execution is below the product floor.
func assertExecutionFloor(t *testing.T) {
	t.Helper()
	exec := aviationTierBundle(aviationFixture{"product", "C", "reviewed", "statement"})
	exec.Aviation.Aviation.Executions[0].Status = "planned"
	if got := pathsOf(Bundle(exec), "aviation.state_below_tier"); len(got) != 1 || got[0] != "aviation.executions.EXEC-1" {
		t.Errorf("planned execution at product: %v", got)
	}
}

// assertOperationalFloor checks that at operational every reviewed record is below accepted.
func assertOperationalFloor(t *testing.T) {
	t.Helper()
	operational := Bundle(aviationTierBundle(aviationFixture{"operational", "C", "reviewed", "statement"}))
	if got := pathsOf(operational, "aviation.state_below_tier"); len(got) < 22 {
		t.Errorf("operational with reviewed records: %d findings, want every record below accepted", len(got))
	}
}

// assertNotApplicableFloor checks that not-applicable satisfies a floor only with a rationale.
func assertNotApplicableFloor(t *testing.T) {
	t.Helper()
	na := aviationTierBundle(aviationFixture{"mvp", "C", "planned", "statement"})
	na.Aviation.Aviation.LifecycleData[0].State = "not-applicable"
	if got := pathsOf(Bundle(na), "aviation.state_below_tier"); len(got) != 1 {
		t.Errorf("not-applicable plan without rationale: %v", got)
	}
	na.Aviation.Aviation.LifecycleData[0].Rationale = "The certification liaison agreed no PSAC for this item."
	if got := pathsOf(Bundle(na), "aviation.state_below_tier"); len(got) != 0 {
		t.Errorf("not-applicable plan with rationale: %v", got)
	}
}

// With the safety bundle an aviation reviewer, independence or execution
// producer or verifier, or approval approver that is a registered
// engineering-flow agent is aviation.ai_actor_as_credit (error, rank 0), at
// every tier; a person's name is not, and without the safety bundle nothing
// is reported.
// TRLC-LINKS: REQ-EMG-133
func TestAviationAIActorAsCredit(t *testing.T) {
	b := aviationTierBundle(aviationFixture{"poc", "C", "planned", "statement"})
	a := &b.Aviation.Aviation
	a.Reviews[0].Reviewer = "verification-review"
	a.Independence[0].Producer = "ef-implementer"
	a.Independence[0].Verifier = "Safety-Review"
	a.Executions[0].Verifier = " quality-review "
	a.Approvals[0].Approver = "ef-modeler"
	got := codesOf(ApplyTierPolicy(b, Bundle(b)), "aviation.ai_actor_as_credit")
	assertAICreditPaths(t, got)
	if !strings.Contains(got[0].Message, "ef-modeler") || !strings.Contains(got[0].Message, "person") {
		t.Errorf("message: %s", got[0].Message)
	}
	b.Manifest.Lifecycle.Bundles = []string{"security"}
	if got := codesOf(Bundle(b), "aviation.ai_actor_as_credit"); len(got) != 0 {
		t.Errorf("without safety: %+v", got)
	}
	agents := AIAgentNames()
	if len(agents) != 13 || agents[0] != "architecture-review" || agents[len(agents)-1] != "verification-review" {
		t.Errorf("AIAgentNames = %v", agents)
	}
}

// assertAICreditPaths checks that the credit findings are errors at the five expected paths.
func assertAICreditPaths(t *testing.T, got []Diagnostic) {
	t.Helper()
	want := []string{"aviation.approvals.APPROVAL-1.approver", "aviation.executions.EXEC-1.verifier", "aviation.independence.INDEP-1.producer", "aviation.independence.INDEP-1.verifier", "aviation.reviews.REVIEW-1.reviewer"}
	var paths []string
	for _, d := range got {
		if d.Severity != SeverityError {
			t.Errorf("not an error: %+v", d)
		}
		paths = append(paths, d.Path)
	}
	if strings.Join(paths, ",") != strings.Join(want, ",") {
		t.Fatalf("paths %v, want %v", paths, want)
	}
}

// With lifecycle the aviation state codes are validation diagnostics keyed
// by tier (the load no longer fails on them) and an authority-accepted
// state without a liaison authority record is
// aviation.authority_state_unevidenced (error, rank 0).
// TRLC-LINKS: REQ-EMG-132
func TestAviationStateCodesAreTierKeyed(t *testing.T) {
	b := aviationTierBundle(aviationFixture{"prototype", "C", "planned", "statement"})
	b.Aviation.Aviation.Tools = []model.AirborneTool{{ID: "TOOL-1", CreditClaimed: true}}
	b.Aviation.Aviation.Reviews[0].State = "authority-accepted"
	diags := ApplyTierPolicy(b, Bundle(b))
	tool := codesOf(diags, "aviation.tool.incomplete_credit_assessment")
	if len(tool) != 1 || !tool[0].Deferred || tool[0].RequiredFrom != "mvp" {
		t.Errorf("tool credit at prototype: %+v", tool)
	}
	auth := codesOf(diags, "aviation.authority_state_unevidenced")
	if len(auth) != 1 || !isErrorAt(auth[0], "aviation.authority_state_unevidenced", "aviation.reviews.REVIEW-1") {
		t.Errorf("authority state: %+v", auth)
	}
	b.Manifest.Lifecycle = nil
	if got := append(codesOf(Bundle(b), "aviation.tool.incomplete_credit_assessment"), codesOf(Bundle(b), "aviation.authority_state_unevidenced")...); len(got) != 0 {
		t.Errorf("without lifecycle (the load reports them): %+v", got)
	}
}
