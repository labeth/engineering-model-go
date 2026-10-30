// ENGMODEL-OWNER-UNIT: FU-MODEL-CHANGE
package reqdelta

import (
	"strings"
	"testing"

	"github.com/labeth/engineering-model-go/internal/model"
)

func specFixture() []model.Requirement {
	return []model.Requirement{
		{ID: "REQ-SIM-001", Title: "Publish state", Text: "When the run starts, the simulator shall publish the `entity` state once per period.",
			VerificationCriteria: "When the run starts with a 1 s period, then three states arrive within 3 s."},
		{ID: "REQ-SIM-002", Text: "The simulator shall reject\n  a message   without a sequence number.",
			VerificationCriteria: "A test sends a message without a sequence number and checks it is dropped."},
		{ID: "REQ-OTHER-001", Text: "The other system shall stay out of the delta."},
		{ID: "REQ-SIM-003", Title: "Publish state", Text: "Marshalling shall not change field order.",
			VerificationCriteria: "Given a decoded message, when it is encoded again, then the bytes are equal."},
	}
}

// The renderer prints one ADDED requirement per selected model requirement,
// in model order, with the model text on the REQ line (shall capitalized,
// whitespace collapsed) and one scenario shaped from verificationCriteria.
// Mutation: dropping the shall→SHALL rewrite, emitting the criteria without
// WHEN/THEN shaping, or ignoring the prefix filter fails this golden test.
// TRLC-LINKS: REQ-EMG-081
func TestRenderSpecDeltaGolden(t *testing.T) {
	got, err := RenderSpecDelta(specFixture(), SpecDeltaOptions{Prefix: "REQ-SIM-", Capability: "simulator"})
	if err != nil {
		t.Fatal(err)
	}
	want := "# simulator spec delta\n" +
		"\n" +
		"## ADDED Requirements\n" +
		"\n" +
		"### Requirement: Publish state\n" +
		"REQ-SIM-001: When the run starts, the simulator SHALL publish the `entity` state once per period.\n" +
		"\n" +
		"#### Scenario: Verify REQ-SIM-001\n" +
		"- **WHEN** the run starts with a 1 s period\n" +
		"- **THEN** three states arrive within 3 s.\n" +
		"\n" +
		"### Requirement: REQ-SIM-002\n" +
		"REQ-SIM-002: The simulator SHALL reject a message without a sequence number.\n" +
		"\n" +
		"#### Scenario: Verify REQ-SIM-002\n" +
		"- **WHEN** the requirement is verified\n" +
		"- **THEN** a test sends a message without a sequence number and checks it is dropped.\n" +
		"\n" +
		"### Requirement: Publish state (REQ-SIM-003)\n" +
		"REQ-SIM-003: Marshalling SHALL not change field order.\n" +
		"\n" +
		"#### Scenario: Verify REQ-SIM-003\n" +
		"- **GIVEN** a decoded message\n" +
		"- **WHEN** it is encoded again\n" +
		"- **THEN** the bytes are equal.\n"
	if got.Markdown != want {
		t.Fatalf("markdown mismatch\n--- got\n%s\n--- want\n%s", got.Markdown, want)
	}
	if strings.Join(got.IDs, ",") != "REQ-SIM-001,REQ-SIM-002,REQ-SIM-003" {
		t.Fatalf("ids: %v", got.IDs)
	}
}

// Only whole-word "shall" becomes SHALL, in any case; words containing it
// are left alone, and a requirement without shall or must is reported
// because openspec validate --strict rejects it.
// Mutation: a plain substring replace (marshALLing) or a case-sensitive
// match fails this test.
// TRLC-LINKS: REQ-EMG-081
func TestRenderSpecDeltaShallCapitalization(t *testing.T) {
	reqs := []model.Requirement{
		{ID: "REQ-A-001", Text: "Shall-be fields: the unit Shall keep shallow copies and shall log.", VerificationCriteria: "A log test."},
		{ID: "REQ-A-002", Text: "The unit should log.", VerificationCriteria: "A log test."},
	}
	got, err := RenderSpecDelta(reqs, SpecDeltaOptions{IDs: []string{"REQ-A-001", "REQ-A-002"}})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(got.Markdown, "REQ-A-001: SHALL-be fields: the unit SHALL keep shallow copies and SHALL log.\n") {
		t.Fatalf("capitalization:\n%s", got.Markdown)
	}
	if len(got.Warnings) != 1 || !strings.Contains(strings.Join(got.Warnings, "\n"), "REQ-A-002 has no shall or must") {
		t.Fatalf("warnings: %v", got.Warnings)
	}
}

// Criteria in WHEN/THEN form split into WHEN and THEN bullets (GIVEN kept);
// other criteria become THEN under a generic WHEN; missing criteria are
// warned about. --ids selects exactly the named requirements and an unknown
// ID is an error. MODIFIED renders the MODIFIED section.
// Mutation: treating every criterion as unshaped, or silently skipping an
// unknown ID, fails this test.
// TRLC-LINKS: REQ-EMG-081
func TestRenderSpecDeltaCriteriaShaping(t *testing.T) {
	cases := []struct{ criteria, want string }{
		{"WHEN the cache is full THEN the oldest entry is evicted", "- **WHEN** the cache is full\n- **THEN** the oldest entry is evicted\n"},
		{"when a b, then c d.", "- **WHEN** a b\n- **THEN** c d.\n"},
		{"Golden tests compare\n each sentence.", "- **WHEN** the requirement is verified\n- **THEN** golden tests compare each sentence.\n"},
		{"Checks that whenever x happens then y.", "- **WHEN** the requirement is verified\n- **THEN** checks that whenever x happens then y.\n"},
	}
	for _, c := range cases {
		got, err := RenderSpecDelta([]model.Requirement{{ID: "REQ-B-001", Text: "B shall c.", VerificationCriteria: c.criteria}}, SpecDeltaOptions{IDs: []string{"REQ-B-001"}})
		if err != nil {
			t.Fatal(err)
		}
		if !strings.HasSuffix(got.Markdown, "#### Scenario: Verify REQ-B-001\n"+c.want) {
			t.Fatalf("criteria %q:\n%s", c.criteria, got.Markdown)
		}
	}
	got, err := RenderSpecDelta([]model.Requirement{{ID: "REQ-B-002", Text: "B shall d."}}, SpecDeltaOptions{IDs: []string{"REQ-B-002"}, Operation: SpecDeltaModified})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(got.Markdown, "## MODIFIED Requirements\n") || !strings.Contains(got.Markdown, "- **THEN** REQ-B-002 holds\n") ||
		len(got.Warnings) != 1 || !strings.Contains(got.Warnings[0], "no verificationCriteria") {
		t.Fatalf("modified/no criteria: %v\n%s", got.Warnings, got.Markdown)
	}
	if _, err := RenderSpecDelta(specFixture(), SpecDeltaOptions{IDs: []string{"REQ-SIM-001", "REQ-NOPE-001"}}); err == nil || !strings.Contains(err.Error(), "REQ-NOPE-001") {
		t.Fatalf("unknown id: %v", err)
	}
	if _, err := RenderSpecDelta(specFixture(), SpecDeltaOptions{}); err == nil {
		t.Fatal("no selection must be an error")
	}
	one, err := RenderSpecDelta(specFixture(), SpecDeltaOptions{IDs: []string{"REQ-SIM-002"}})
	if err != nil || strings.Join(one.IDs, ",") != "REQ-SIM-002" {
		t.Fatalf("ids selection: %v %v", one.IDs, err)
	}
}
