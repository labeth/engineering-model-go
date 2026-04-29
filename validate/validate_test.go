// ENGMODEL-OWNER-UNIT: FU-VALIDATION-ENGINE
package validate

import (
	"path/filepath"
	"testing"

	"github.com/labeth/engineering-model-go/model"
)

// TRLC-LINKS: REQ-EMG-001, REQ-EMG-009, REQ-EMG-011
func TestBundleValidationNoErrors(t *testing.T) {
	p := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	b, err := model.LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	diags := Bundle(b)
	if HasErrors(diags) {
		t.Fatalf("expected no validation errors, got: %+v", diags)
	}
}

func TestViewIDIsFreeButKindIsStrict(t *testing.T) {
	p := filepath.Join("..", "examples", "payments-engineering-sample", "architecture.yml")
	b, err := model.LoadBundle(p)
	if err != nil {
		t.Fatalf("load bundle failed: %v", err)
	}
	if len(b.Architecture.Views) == 0 {
		t.Fatalf("expected sample views")
	}

	// Free-form IDs should be accepted as long as kind is supported.
	b.Architecture.Views[0].ID = "run"
	b.Architecture.Views[0].Kind = "communication"
	diags := Bundle(b)
	if HasErrors(diags) {
		t.Fatalf("expected no errors for free-form view id with valid kind, got: %+v", diags)
	}

	// Unsupported kind should fail validation regardless of ID.
	b.Architecture.Views[0].Kind = "run"
	diags = Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected errors for unsupported view kind")
	}
	found := false
	for _, d := range diags {
		if d.Code == "model.unknown_view_kind" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected model.unknown_view_kind diagnostic, got: %+v", diags)
	}
}

func TestBundleValidation_Decisions(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{
		Decisions: []model.Decision{{
			ID:           "ADR-A",
			Title:        "Use generated decisions document",
			Status:       "accepted",
			Date:         "2026-04-29",
			Context:      "Decision context.",
			Decision:     "Decision text.",
			Consequences: []string{"Decision consequence."},
		}},
		AuthoredArchitecture: model.AuthoredArchitecture{
			FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
			FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		},
		Views: []model.View{{ID: "V", Kind: "architecture-intent", Roots: []string{"FG-A"}}},
	}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected valid decision to pass, got: %+v", diags)
	}

	b.Architecture.Decisions[0].Date = ""
	b.Architecture.Decisions[0].Status = "done"
	b.Architecture.Decisions[0].Consequences = nil
	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid decision to fail")
	}
	want := map[string]bool{
		"model.missing_decision_date":       false,
		"model.invalid_decision_status":     false,
		"model.empty_decision_consequences": false,
	}
	for _, d := range diags {
		if _, ok := want[d.Code]; ok {
			want[d.Code] = true
		}
	}
	for code, found := range want {
		if !found {
			t.Fatalf("missing %s in diagnostics: %+v", code, diags)
		}
	}
}

func TestBundleValidation_ExpandedMappingTypesAndPairs(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups:   []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:    []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		Actors:             []model.Actor{{ID: "ACT-A", Name: "Actor A"}},
		AttackVectors:      []model.AttackVector{{ID: "AV-A", Name: "Attack"}},
		Interfaces:         []model.Interface{{ID: "IF-A", Name: "API", Owner: "FU-A"}},
		DataObjects:        []model.DataObject{{ID: "DATA-A", Name: "Payload"}},
		DeploymentTargets:  []model.DeploymentTarget{{ID: "DEP-A", Name: "Prod"}},
		Controls:           []model.Control{{ID: "CTRL-A", Name: "Digest Pinning"}},
		TrustBoundaries:    []model.TrustBoundary{{ID: "TB-A", Name: "Boundary"}},
		States:             []model.State{{ID: "STATE-A", Name: "Ready"}, {ID: "STATE-B", Name: "Applied"}},
		Events:             []model.Event{{ID: "EVT-A", Name: "Deploy Requested"}},
		ReferencedElements: []model.ReferencedElement{{ID: "REF-A", Name: "Reference", Kind: "ext", Layer: "external"}},
		Mappings: []model.Mapping{
			{Type: "calls", From: "FU-A", To: "IF-A"},
			{Type: "writes", From: "FU-A", To: "DATA-A"},
			{Type: "deployed_to", From: "FU-A", To: "DEP-A"},
			{Type: "mitigated_by", From: "AV-A", To: "CTRL-A"},
			{Type: "bounded_by", From: "FU-A", To: "TB-A"},
			{Type: "triggered_by", From: "STATE-A", To: "EVT-A"},
			{Type: "transitions_to", From: "STATE-A", To: "STATE-B"},
			{Type: "guarded_by", From: "STATE-B", To: "CTRL-A"},
		},
	}, Views: []model.View{{ID: "V", Kind: "traceability", Roots: []string{"FU-A"}}}}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected expanded mapping set to validate, got: %+v", diags)
	}

	b.Architecture.AuthoredArchitecture.Mappings = append(b.Architecture.AuthoredArchitecture.Mappings, model.Mapping{Type: "writes", From: "ACT-A", To: "DATA-A"})
	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid mapping pair to fail")
	}
	found := false
	for _, d := range diags {
		if d.Code == "model.invalid_mapping_pair" {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected model.invalid_mapping_pair diagnostic, got: %+v", diags)
	}
}

func TestBundleValidation_ViewSpecFields(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
	}, Views: []model.View{{
		ID:              "V",
		Kind:            "communication",
		Roots:           []string{"FU-A"},
		IncludeKinds:    []string{"functional_unit", "interface"},
		ExcludeKinds:    []string{"attack_vector"},
		IncludeMappings: []string{"calls", "depends_on"},
		ExcludeMappings: []string{"targets"},
		MaxDepth:        2,
	}}}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected valid view spec fields, got: %+v", diags)
	}

	b.Architecture.Views[0].IncludeKinds = []string{"invalid-kind"}
	b.Architecture.Views[0].MaxDepth = -1
	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid view spec to fail")
	}
	hasKindErr := false
	hasDepthErr := false
	for _, d := range diags {
		if d.Code == "model.unknown_view_entity_kind" {
			hasKindErr = true
		}
		if d.Code == "model.invalid_view_depth" {
			hasDepthErr = true
		}
	}
	if !hasKindErr || !hasDepthErr {
		t.Fatalf("expected unknown kind and invalid depth diagnostics, got: %+v", diags)
	}
}

func TestBundleValidation_InteractionFlowValid(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		Actors:           []model.Actor{{ID: "ACT-A", Name: "User"}},
		Interfaces:       []model.Interface{{ID: "IF-A", Name: "Control API", Owner: "FU-A"}},
		DataObjects:      []model.DataObject{{ID: "DO-A", Name: "Selection State"}},
		Flows: []model.Flow{{
			ID:    "FLOW-INPUT-SELECTION",
			Title: "Input selection flow",
			Entry: []string{"user-submit"},
			Exits: []string{"state-saved"},
			Steps: []model.FlowStep{
				{ID: "user-submit", Kind: "user_action", Ref: "ACT-A", Action: "Submit selection", DataIn: []string{"input_selection"}, Next: []string{"api-ingest"}},
				{ID: "api-ingest", Kind: "system_action", Ref: "IF-A", Action: "Validate and ingest", DataOut: []string{"normalized_selection"}, Next: []string{"state-saved"}},
				{ID: "state-saved", Kind: "data_move", Ref: "DO-A", Action: "Persist selection"},
			},
		}},
	}, Views: []model.View{{ID: "V-FLOW", Kind: "interaction-flow", Roots: []string{"FLOW-INPUT-SELECTION"}, IncludeMappings: []string{"flow_next", "flow_ref"}}}}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected valid interaction flow model, got: %+v", diags)
	}
}

func TestBundleValidation_InteractionFlowInvalid(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		Flows: []model.Flow{{
			ID:    "FLOW-BROKEN",
			Title: "Broken flow",
			Entry: []string{"missing-step"},
			Steps: []model.FlowStep{
				{ID: "step-1", Kind: "invalid-kind", Ref: "MISSING-ID", Next: []string{"step-2"}},
				{ID: "step-1", Kind: "system_action", Ref: "FU-A"},
			},
		}},
	}, Views: []model.View{{ID: "V-FLOW", Kind: "interaction-flow", Roots: []string{"FLOW-BROKEN"}}}}}

	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid flow model to fail")
	}
	checks := map[string]bool{}
	for _, d := range diags {
		checks[d.Code] = true
	}
	for _, code := range []string{"model.invalid_flow_step_kind", "model.broken_flow_reference", "model.duplicate_flow_step_id", "model.broken_flow_step_link"} {
		if !checks[code] {
			t.Fatalf("expected diagnostic %s, got: %+v", code, diags)
		}
	}
}

func TestBundleValidation_ControlAllocations(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		Actors:           []model.Actor{{ID: "ACT-A", Name: "Owner"}},
		Controls:         []model.Control{{ID: "CTRL-A", Name: "MFA"}},
		ControlAllocations: []model.ControlAllocation{{
			ID:                 "ALLOC-A",
			ControlRef:         "CTRL-A",
			OSCALControlIDs:    []string{"ac-2", "ia-2(1)"},
			AppliesTo:          []string{"FU-A"},
			ImplementationType: "technical",
			Status:             "implemented",
			Narrative:          "MFA enforced through SSO policy.",
			Evidence:           []model.ControlEvidence{{Path: "infra/identity/policy.yaml"}},
			ResponsibleRoles:   []string{"ACT-A"},
		}},
	}, Views: []model.View{{ID: "V", Kind: "traceability", Roots: []string{"FU-A"}}}}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected valid control allocation, got: %+v", diags)
	}

	b.Architecture.AuthoredArchitecture.ControlAllocations[0].OSCALControlIDs = []string{"bad id"}
	b.Architecture.AuthoredArchitecture.ControlAllocations[0].ResponsibleRoles = []string{"MISSING"}
	b.Architecture.AuthoredArchitecture.ControlAllocations[0].ControlRef = "CTRL-MISSING"
	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid control allocation diagnostics")
	}
	seen := map[string]bool{}
	for _, d := range diags {
		seen[d.Code] = true
	}
	for _, code := range []string{"model.invalid_control_ref", "model.invalid_oscal_control_id", "model.invalid_control_responsible_role"} {
		if !seen[code] {
			t.Fatalf("expected diagnostic %s, got %+v", code, diags)
		}
	}
}

func TestBundleValidation_RisksAndPOAM(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		Actors:           []model.Actor{{ID: "ACT-A", Name: "Owner"}},
		AttackVectors:    []model.AttackVector{{ID: "AV-A", Name: "Attack"}},
		Controls:         []model.Control{{ID: "CTRL-A", Name: "Control A"}},
		Risks: []model.Risk{{
			ID:              "RISK-A",
			Title:           "MFA bypass",
			Statement:       "MFA may be bypassed.",
			Status:          "mitigating",
			Likelihood:      "medium",
			Impact:          "high",
			Response:        "mitigate",
			Owner:           "ACT-A",
			AppliesTo:       []string{"FU-A"},
			RelatedControls: []string{"CTRL-A"},
			AttackVectors:   []string{"AV-A"},
			Evidence:        []model.RiskEvidence{{Path: "docs/risk.md"}},
			ResidualRisk:    "medium",
		}},
		POAMItems: []model.POAMItem{{
			ID:              "POAM-A",
			RiskRef:         "RISK-A",
			Milestone:       "Remove legacy account",
			DueDate:         "2026-06-30",
			Status:          "in-progress",
			ResponsibleRole: "ACT-A",
			Artifacts:       []model.POAMArtifact{{Path: "tickets/SEC-1.md"}},
		}},
	}, Views: []model.View{{ID: "V", Kind: "traceability", Roots: []string{"FU-A"}}}}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected valid risk/poam model, got: %+v", diags)
	}

	b.Architecture.AuthoredArchitecture.Risks[0].Owner = "MISSING"
	b.Architecture.AuthoredArchitecture.POAMItems[0].RiskRef = "RISK-MISSING"
	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid risk/poam diagnostics")
	}
	seen := map[string]bool{}
	for _, d := range diags {
		seen[d.Code] = true
	}
	for _, code := range []string{"model.invalid_risk_owner", "model.invalid_poam_risk_ref"} {
		if !seen[code] {
			t.Fatalf("expected diagnostic %s, got %+v", code, diags)
		}
	}
}

func TestBundleValidation_ThreatScenariosAndFlowMetadataValid(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		Actors:           []model.Actor{{ID: "ACT-A", Name: "Sec Owner"}},
		AttackVectors:    []model.AttackVector{{ID: "AV-A", Name: "Replay"}},
		Controls:         []model.Control{{ID: "CTRL-A", Name: "Nonce validation"}},
		Interfaces:       []model.Interface{{ID: "IF-A", Name: "Webhook API", Owner: "FU-A"}},
		DataObjects:      []model.DataObject{{ID: "DATA-A", Name: "Webhook Payload"}},
		TrustBoundaries:  []model.TrustBoundary{{ID: "TB-A", Name: "Edge", Members: []string{"FU-A", "IF-A"}}},
		Risks: []model.Risk{{
			ID:              "RISK-A",
			Title:           "Replay risk",
			Statement:       "Replay can trigger duplicate processing.",
			Likelihood:      "medium",
			Impact:          "high",
			ThreatScenarios: []string{"TS-A"},
		}},
		Flows: []model.Flow{{
			ID:             "FLOW-A",
			Title:          "Webhook ingress",
			Kind:           "security",
			Direction:      "inbound",
			Frequency:      "realtime",
			SourceRef:      "IF-A",
			DestinationRef: "FU-A",
			DataRefs:       []string{"DATA-A"},
			Criticality:    "high",
			Threats:        []string{"TS-A"},
			Entry:          []string{"receive"},
			Exits:          []string{"persist"},
			Steps: []model.FlowStep{
				{ID: "receive", Kind: "external_interaction", FlowType: "interaction", Direction: "inbound", Frequency: "realtime", SourceRef: "IF-A", DestinationRef: "FU-A", Ref: "IF-A", InterfaceRef: "IF-A", Next: []string{"verify"}},
				{ID: "verify", Kind: "security_check", FlowType: "security", Direction: "internal", Frequency: "realtime", SourceRef: "FU-A", DestinationRef: "DATA-A", Ref: "FU-A", DataRefs: []string{"DATA-A"}, BoundaryCrossing: true, TrustBoundaryRef: "TB-A", EncryptionInTransit: "tls1.3", Authentication: "hmac", Next: []string{"persist"}},
				{ID: "persist", Kind: "data_move", FlowType: "data", Ref: "DATA-A"},
			},
		}},
		ThreatAssumptions: []model.ThreatAssumption{{
			ID:        "ASM-A",
			Title:     "Gateway signs callbacks",
			Statement: "Bank gateway callback signatures are generated with managed keys.",
			Status:    "accepted",
			Owner:     "ACT-A",
			AppliesTo: []string{"IF-A", "FU-A"},
			Evidence:  []model.ThreatAssumptionEvidence{{Path: "docs/assumptions/bank-signing.md"}},
		}},
		ThreatOutOfScope: []model.ThreatOutOfScope{{
			ID:        "OOS-A",
			Title:     "Bank HSM internals",
			Reason:    "Managed by external bank provider control environment.",
			Status:    "approved",
			Owner:     "ACT-A",
			AppliesTo: []string{"IF-A"},
			ExpiresOn: "2027-01-01",
			Evidence:  []model.ThreatOutOfScopeEvidence{{Path: "docs/scope/external-bank.md"}},
		}},
		ControlVerifications: []model.ControlVerification{{
			ID:                 "CV-A",
			ControlRef:         "CTRL-A",
			ThreatScenarioRefs: []string{"TS-A"},
			RiskRefs:           []string{"RISK-A"},
			Method:             "test",
			Status:             "pass",
			Owner:              "ACT-A",
			LastTested:         "2026-07-01",
			Evidence:           []model.ControlVerificationEvidence{{Path: "tests/contract/replay_protection_test.go"}},
		}},
		ThreatMitigations: []model.ThreatMitigation{{
			ID:                "MIT-A",
			ThreatScenarioRef: "TS-A",
			ControlRef:        "CTRL-A",
			Status:            "verified",
			Effectiveness:     "high",
			Owner:             "ACT-A",
			VerificationRefs:  []string{"CV-A"},
			Evidence:          []model.ThreatMitigationEvidence{{Path: "docs/controls/nonce-validation.md"}},
		}},
		ThreatScenarios: []model.ThreatScenario{{
			ID:               "TS-A",
			Title:            "Webhook replay with stale signature",
			AttackVectorRef:  "AV-A",
			AppliesTo:        []string{"FU-A"},
			FlowRefs:         []string{"FLOW-A"},
			Likelihood:       "medium",
			Impact:           "high",
			Severity:         "high",
			Status:           "identified",
			Owner:            "ACT-A",
			RiskRef:          "RISK-A",
			RelatedControls:  []string{"CTRL-A"},
			AssumptionRefs:   []string{"ASM-A"},
			OutOfScopeRefs:   []string{"OOS-A"},
			MitigationRefs:   []string{"MIT-A"},
			VerificationRefs: []string{"CV-A"},
			Evidence:         []model.ThreatScenarioEvidence{{Path: "docs/threats/replay.md"}},
		}},
	}, Views: []model.View{{ID: "V", Kind: "security", Roots: []string{"FU-A"}}}}}

	if diags := Bundle(b); HasErrors(diags) {
		t.Fatalf("expected valid threat scenario and flow metadata model, got: %+v", diags)
	}
}

func TestBundleValidation_ThreatScenariosAndFlowMetadataInvalid(t *testing.T) {
	b := model.Bundle{Architecture: model.ArchitectureDocument{AuthoredArchitecture: model.AuthoredArchitecture{
		FunctionalGroups: []model.FunctionalGroup{{ID: "FG-A", Name: "Group A"}},
		FunctionalUnits:  []model.FunctionalUnit{{ID: "FU-A", Group: "FG-A", Name: "Unit A"}},
		Actors:           []model.Actor{{ID: "ACT-A", Name: "Sec Owner"}},
		AttackVectors:    []model.AttackVector{{ID: "AV-A", Name: "Replay"}},
		Controls:         []model.Control{{ID: "CTRL-A", Name: "Nonce validation"}},
		Interfaces:       []model.Interface{{ID: "IF-A", Name: "Webhook API", Owner: "FU-A"}},
		DataObjects:      []model.DataObject{{ID: "DATA-A", Name: "Webhook Payload"}},
		TrustBoundaries:  []model.TrustBoundary{{ID: "TB-A", Name: "Edge", ParentRef: "TB-MISSING", Members: []string{"FU-MISSING"}}},
		Flows: []model.Flow{{
			ID:             "FLOW-A",
			Title:          "Webhook ingress",
			Kind:           "not-a-kind",
			Direction:      "sideways",
			Frequency:      "sometimes",
			SourceRef:      "MISSING-SRC",
			DestinationRef: "MISSING-DST",
			DataRefs:       []string{"DATA-MISSING-FLOW"},
			Threats:        []string{"TS-MISSING"},
			Entry:          []string{"receive"},
			Steps: []model.FlowStep{{
				ID:               "receive",
				Kind:             "external_interaction",
				FlowType:         "not-a-flow",
				Direction:        "up",
				Frequency:        "rare",
				SourceRef:        "MISSING-STEP-SRC",
				DestinationRef:   "MISSING-STEP-DST",
				Ref:              "IF-A",
				InterfaceRef:     "IF-MISSING",
				DataRefs:         []string{"DATA-MISSING"},
				BoundaryCrossing: true,
			}},
		}},
		ThreatAssumptions: []model.ThreatAssumption{{
			ID:        "ASM-A",
			Title:     "",
			Statement: "",
			Status:    "wrong",
			Owner:     "ACT-MISSING",
			AppliesTo: []string{"MISSING-SCOPE"},
			Evidence:  []model.ThreatAssumptionEvidence{{Path: ""}},
		}},
		ThreatOutOfScope: []model.ThreatOutOfScope{{
			ID:        "OOS-A",
			Title:     "",
			Reason:    "",
			Status:    "wrong",
			Owner:     "ACT-MISSING",
			AppliesTo: []string{"MISSING-SCOPE"},
			ExpiresOn: "2026/13/50",
			Evidence:  []model.ThreatOutOfScopeEvidence{{Path: ""}},
		}},
		ControlVerifications: []model.ControlVerification{{
			ID:                 "CV-A",
			ControlRef:         "CTRL-MISSING",
			ThreatScenarioRefs: []string{"TS-MISSING"},
			RiskRefs:           []string{"RISK-MISSING"},
			Method:             "chaos",
			Status:             "done",
			Owner:              "ACT-MISSING",
			LastTested:         "not-a-date",
			Evidence:           []model.ControlVerificationEvidence{{Path: ""}},
		}},
		ThreatMitigations: []model.ThreatMitigation{{
			ID:                "MIT-A",
			ThreatScenarioRef: "TS-MISSING",
			ControlRef:        "CTRL-MISSING",
			Status:            "unknown",
			Effectiveness:     "extreme",
			Owner:             "ACT-MISSING",
			VerificationRefs:  []string{"CV-MISSING"},
			Evidence:          []model.ThreatMitigationEvidence{{Path: ""}},
		}},
		Risks: []model.Risk{{
			ID:              "RISK-A",
			Title:           "Replay risk",
			Statement:       "Replay can trigger duplicate processing.",
			ThreatScenarios: []string{"TS-MISSING"},
		}},
		ThreatScenarios: []model.ThreatScenario{{
			ID:               "TS-A",
			Title:            "Broken threat",
			AttackVectorRef:  "AV-MISSING",
			FlowRefs:         []string{"FLOW-MISSING"},
			Status:           "not-a-status",
			Owner:            "ACT-MISSING",
			RiskRef:          "RISK-MISSING",
			RelatedControls:  []string{"CTRL-MISSING"},
			AssumptionRefs:   []string{"ASM-MISSING"},
			OutOfScopeRefs:   []string{"OOS-MISSING"},
			MitigationRefs:   []string{"MIT-MISSING"},
			VerificationRefs: []string{"CV-MISSING"},
			Evidence:         []model.ThreatScenarioEvidence{{Path: ""}},
		}},
	}, Views: []model.View{{ID: "V", Kind: "security", Roots: []string{"FU-A"}}}}}

	diags := Bundle(b)
	if !HasErrors(diags) {
		t.Fatalf("expected invalid threat scenario and flow metadata diagnostics")
	}
	seen := map[string]bool{}
	for _, d := range diags {
		seen[d.Code] = true
	}
	for _, code := range []string{
		"model.invalid_flow_kind",
		"model.invalid_flow_direction",
		"model.invalid_flow_frequency",
		"model.invalid_flow_source_ref",
		"model.invalid_flow_destination_ref",
		"model.invalid_flow_threat_ref",
		"model.invalid_flow_step_type",
		"model.invalid_flow_step_direction",
		"model.invalid_flow_step_frequency",
		"model.invalid_flow_step_source_ref",
		"model.invalid_flow_step_destination_ref",
		"model.invalid_flow_interface_ref",
		"model.invalid_flow_data_ref",
		"model.missing_flow_boundary_ref",
		"model.invalid_trust_boundary_parent",
		"model.invalid_trust_boundary_member",
		"model.invalid_risk_threat_scenario_ref",
		"model.invalid_threat_scenario_attack_vector",
		"model.invalid_threat_scenario_flow_ref",
		"model.invalid_threat_scenario_status",
		"model.invalid_threat_scenario_owner",
		"model.invalid_threat_scenario_risk_ref",
		"model.invalid_threat_scenario_control_ref",
		"model.invalid_threat_scenario_assumption_ref",
		"model.invalid_threat_scenario_out_of_scope_ref",
		"model.invalid_threat_scenario_mitigation_ref",
		"model.invalid_threat_scenario_verification_ref",
		"model.empty_threat_scenario_evidence_path",
		"model.missing_threat_assumption_title",
		"model.missing_threat_assumption_statement",
		"model.invalid_threat_assumption_status",
		"model.invalid_threat_assumption_owner",
		"model.invalid_threat_assumption_scope_target",
		"model.empty_threat_assumption_evidence_path",
		"model.missing_threat_out_of_scope_title",
		"model.missing_threat_out_of_scope_reason",
		"model.invalid_threat_out_of_scope_status",
		"model.invalid_threat_out_of_scope_owner",
		"model.invalid_threat_out_of_scope_expiry",
		"model.invalid_threat_out_of_scope_target",
		"model.empty_threat_out_of_scope_evidence_path",
		"model.invalid_control_verification_control_ref",
		"model.invalid_control_verification_threat_ref",
		"model.invalid_control_verification_risk_ref",
		"model.invalid_control_verification_method",
		"model.invalid_control_verification_status",
		"model.invalid_control_verification_owner",
		"model.invalid_control_verification_last_tested",
		"model.empty_control_verification_evidence_path",
		"model.invalid_threat_mitigation_scenario_ref",
		"model.invalid_threat_mitigation_control_ref",
		"model.invalid_threat_mitigation_status",
		"model.invalid_threat_mitigation_effectiveness",
		"model.invalid_threat_mitigation_owner",
		"model.invalid_threat_mitigation_verification_ref",
		"model.empty_threat_mitigation_evidence_path",
	} {
		if !seen[code] {
			t.Fatalf("expected diagnostic %s, got %+v", code, diags)
		}
	}
}
