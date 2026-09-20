// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'atlas-shared-compliance-baseline' {
    private import ScalarValues::*;

    private metadata def EngineeringElement {
        attribute sourceId : String;
        attribute conceptKind : String;
    }

    private metadata def EngineeringRelationship :> EngineeringElement {
        attribute relationshipKind : String;
    }

    private metadata def EngineeringProject :> EngineeringElement;

    private item def EngineeringItem;
    private part def EngineeringPart;
    private port def EngineeringPort;
    private action def EngineeringAction;
    private action def EngineeringEvent;
    private state def EngineeringState;
    private requirement def EngineeringRequirement;
    private concern def EngineeringConcern;
    private constraint def EngineeringConstraint;
    private calc def EngineeringCalculation;
    private analysis def EngineeringAnalysis;
    private verification def EngineeringVerification;
    private use case def EngineeringUseCase;
    private view def EngineeringView;
    private viewpoint def EngineeringViewpoint;
    private occurrence def EngineeringOccurrence;

    @EngineeringProject {
        sourceId = "atlas-shared-compliance-baseline";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-COMP-OFFICER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-COMP-OFFICER';

    @EngineeringElement {
        sourceId = "ACT-EMPLOYEE";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-EMPLOYEE';

    @EngineeringElement {
        sourceId = "ACT-LINE-MANAGER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-LINE-MANAGER';

    @EngineeringElement {
        sourceId = "ACT-PEOPLE-OPERATIONS";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-PEOPLE-OPERATIONS';

    @EngineeringElement {
        sourceId = "ADR-COMP-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-COMP-001';

    @EngineeringElement {
        sourceId = "ADR-COMP-002";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-COMP-002';

    @EngineeringElement {
        sourceId = "CAP-CONTROL-BASELINE";
        conceptKind = "port_usage";
    }
    port 'CAP-CONTROL-BASELINE' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-1831DF94423ED6FE";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-1831DF94423ED6FE' : EngineeringConcern {
        attribute 'narrative' : 'String' = "NIST and ISO people controls map to company processes, evidence, risks, verification, and remediation.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-21A37A92C3FC49A1";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-21A37A92C3FC49A1' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Screening, employment responsibilities, awareness, role changes, and offboarding are governed once for every product.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2768AB074AC7956D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2768AB074AC7956D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Profiles map external controls to model controls, implementations, verification, evidence, risks, and POA&M.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-32023834D0B44788";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-32023834D0B44788' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Evidence is immutable, attributable, retained, and exportable for assessment.";
    }

    @EngineeringElement {
        sourceId = "CTRL-COMP-ACCESS-REVIEW";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COMP-ACCESS-REVIEW';

    @EngineeringElement {
        sourceId = "CTRL-COMP-EVIDENCE-RETENTION";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COMP-EVIDENCE-RETENTION';

    @EngineeringElement {
        sourceId = "CTRL-COMP-OFFBOARDING";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COMP-OFFBOARDING';

    @EngineeringElement {
        sourceId = "CTRL-COMP-PERSONNEL-SCREENING";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COMP-PERSONNEL-SCREENING';

    @EngineeringElement {
        sourceId = "CTRL-COMP-POLICY-ACKNOWLEDGMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COMP-POLICY-ACKNOWLEDGMENT';

    @EngineeringElement {
        sourceId = "CTRL-COMP-ROLE-CHANGE-REVIEW";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COMP-ROLE-CHANGE-REVIEW';

    @EngineeringElement {
        sourceId = "CTRL-COMP-SECURITY-AWARENESS";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COMP-SECURITY-AWARENESS';

    @EngineeringElement {
        sourceId = "CV-COMP-ACCESS-REVIEW";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COMP-ACCESS-REVIEW' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-COMP-AWARENESS";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COMP-AWARENESS' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-COMP-OFFBOARDING";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COMP-OFFBOARDING' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-COMP-PERSONNEL-SCREENING";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COMP-PERSONNEL-SCREENING' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-COMP-ROLE-CHANGE";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COMP-ROLE-CHANGE' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "DATA-COMP-EVIDENCE";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-COMP-EVIDENCE';

    @EngineeringElement {
        sourceId = "DATA-PERSONNEL-RECORD";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-PERSONNEL-RECORD';

    @EngineeringElement {
        sourceId = "DATA-POLICY-ACKNOWLEDGMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-POLICY-ACKNOWLEDGMENT';

    @EngineeringElement {
        sourceId = "DATA-TRAINING-RECORD";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-TRAINING-RECORD';

    @EngineeringElement {
        sourceId = "DO-COMP-EVIDENCE";
        conceptKind = "item_definition";
    }
    item def 'DO-COMP-EVIDENCE' {
        attribute 'classification' : 'String' = "assurance";
    }

    @EngineeringElement {
        sourceId = "DO-COMP-PERSONNEL-RECORD";
        conceptKind = "item_definition";
    }
    item def 'DO-COMP-PERSONNEL-RECORD' {
        attribute 'classification' : 'String' = "personal";
    }

    @EngineeringElement {
        sourceId = "DO-COMP-TRAINING-RECORD";
        conceptKind = "item_definition";
    }
    item def 'DO-COMP-TRAINING-RECORD' {
        attribute 'classification' : 'String' = "assurance";
    }

    @EngineeringElement {
        sourceId = "EVIDENCE-2172ACFED486B0C3";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-2172ACFED486B0C3';

    @EngineeringElement {
        sourceId = "EVIDENCE-234DC553A81DE3C7";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-234DC553A81DE3C7';

    @EngineeringElement {
        sourceId = "EVIDENCE-3AED01793A23C2DD";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-3AED01793A23C2DD';

    @EngineeringElement {
        sourceId = "EVIDENCE-727D78E497669D4F";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-727D78E497669D4F';

    @EngineeringElement {
        sourceId = "EVIDENCE-86882F7CC5878D65";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-86882F7CC5878D65';

    @EngineeringElement {
        sourceId = "EVIDENCE-87374032F07B3EF5";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-87374032F07B3EF5';

    @EngineeringElement {
        sourceId = "EVIDENCE-9295AF077C5B39CF";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-9295AF077C5B39CF';

    @EngineeringElement {
        sourceId = "EVIDENCE-9454BE277AE11B23";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-9454BE277AE11B23';

    @EngineeringElement {
        sourceId = "EVIDENCE-9BED245291CB398A";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-9BED245291CB398A';

    @EngineeringElement {
        sourceId = "EVIDENCE-B2B390AE3496BA3F";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-B2B390AE3496BA3F';

    @EngineeringElement {
        sourceId = "EVIDENCE-BC827A860FDF4D31";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-BC827A860FDF4D31';

    @EngineeringElement {
        sourceId = "EVIDENCE-C66112D606FDD9E7";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-C66112D606FDD9E7';

    @EngineeringElement {
        sourceId = "EVIDENCE-FED981B76511A3C8";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-FED981B76511A3C8';

    @EngineeringElement {
        sourceId = "FG-COMP";
        conceptKind = "part_definition";
    }
    part def 'FG-COMP';

    @EngineeringElement {
        sourceId = "FLOW-COMP-EMPLOYEE-LIFECYCLE";
        conceptKind = "action_definition";
    }
    action def 'FLOW-COMP-EMPLOYEE-LIFECYCLE' {
        inout item 'item_DO_COMP_PERSONNEL_RECORD' : 'DO-COMP-PERSONNEL-RECORD';
        inout item 'item_DO_COMP_TRAINING_RECORD' : 'DO-COMP-TRAINING-RECORD';
    }

    @EngineeringElement {
        sourceId = "FU-COMP-EVIDENCE";
        conceptKind = "part_usage";
    }
    part 'FU-COMP-EVIDENCE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-COMP-PEOPLE";
        conceptKind = "part_usage";
    }
    part 'FU-COMP-PEOPLE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "IMPL-COMP-A5";
        conceptKind = "engineering_extension";
    }
    attribute 'IMPL-COMP-A5';

    @EngineeringElement {
        sourceId = "IMPL-COMP-AC2";
        conceptKind = "engineering_extension";
    }
    attribute 'IMPL-COMP-AC2';

    @EngineeringElement {
        sourceId = "IMPL-COMP-ISO-PEOPLE";
        conceptKind = "engineering_extension";
    }
    attribute 'IMPL-COMP-ISO-PEOPLE';

    @EngineeringElement {
        sourceId = "IMPL-COMP-NIST-PEOPLE";
        conceptKind = "engineering_extension";
    }
    attribute 'IMPL-COMP-NIST-PEOPLE';

    @EngineeringElement {
        sourceId = "POAM-COMP-001";
        conceptKind = "engineering_extension";
    }
    attribute 'POAM-COMP-001';

    @EngineeringElement {
        sourceId = "POAM-COMP-002";
        conceptKind = "engineering_extension";
    }
    attribute 'POAM-COMP-002';

    @EngineeringElement {
        sourceId = "PROF-ISO-ATLAS";
        conceptKind = "engineering_extension";
    }
    attribute 'PROF-ISO-ATLAS';

    @EngineeringElement {
        sourceId = "PROF-NIST-ATLAS";
        conceptKind = "engineering_extension";
    }
    attribute 'PROF-NIST-ATLAS';

    @EngineeringElement {
        sourceId = "REQ-COMP-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COMP-001' {
        attribute 'text' : 'String' = "The atlas compliance baseline shall retain compliance evidence using the compliance evidence service.";
    }

    @EngineeringElement {
        sourceId = "REQ-COMP-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COMP-002' {
        attribute 'text' : 'String' = "The atlas compliance baseline shall report control implementation status to the compliance officer.";
    }

    @EngineeringElement {
        sourceId = "REQ-COMP-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COMP-003' {
        attribute 'text' : 'String' = "The people security and awareness shall record employee screening status in the personnel lifecycle record before privileged access.";
    }

    @EngineeringElement {
        sourceId = "REQ-COMP-004";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COMP-004' {
        attribute 'text' : 'String' = "The people security and awareness shall record policy acknowledgment for every employee at onboarding.";
    }

    @EngineeringElement {
        sourceId = "REQ-COMP-005";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COMP-005' {
        attribute 'text' : 'String' = "The people security and awareness shall record annual employee completion in the security awareness training record.";
    }

    @EngineeringElement {
        sourceId = "REQ-COMP-006";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COMP-006' {
        attribute 'text' : 'String' = "The people security and awareness shall request a line manager access review following an employee role change.";
    }

    @EngineeringElement {
        sourceId = "REQ-COMP-007";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COMP-007' {
        attribute 'text' : 'String' = "The people security and awareness shall revoke employee access following employment termination.";
    }

    @EngineeringElement {
        sourceId = "REQ-COMP-008";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COMP-008' {
        attribute 'text' : 'String' = "The people security and awareness shall request people operations partner confirmation of company asset return following employment termination.";
    }

    @EngineeringElement {
        sourceId = "RISK-COMP-ACCESS-REVIEW-GAP";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-COMP-ACCESS-REVIEW-GAP';

    @EngineeringElement {
        sourceId = "RISK-COMP-RETAINED-ACCESS";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-COMP-RETAINED-ACCESS';

    @EngineeringElement {
        sourceId = "RISK-COMP-UNTRAINED-PERSONNEL";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-COMP-UNTRAINED-PERSONNEL';

    @EngineeringElement {
        sourceId = "SYS-COMP";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-COMP';

    @EngineeringElement {
        sourceId = "acknowledge-policy";
        conceptKind = "action_usage";
    }
    action 'acknowledge-policy' : EngineeringAction {
        inout item 'item_DO_COMP_PERSONNEL_RECORD' : 'DO-COMP-PERSONNEL-RECORD';
    }

    @EngineeringElement {
        sourceId = "complete-awareness-training";
        conceptKind = "action_usage";
    }
    action 'complete-awareness-training' : EngineeringAction {
        inout item 'item_DO_COMP_TRAINING_RECORD' : 'DO-COMP-TRAINING-RECORD';
    }

    @EngineeringElement {
        sourceId = "confirm-offboarding";
        conceptKind = "action_usage";
    }
    action 'confirm-offboarding' : EngineeringAction {
        inout item 'item_DO_COMP_PERSONNEL_RECORD' : 'DO-COMP-PERSONNEL-RECORD';
    }

    @EngineeringElement {
        sourceId = "review-role-change";
        conceptKind = "action_usage";
    }
    action 'review-role-change' : EngineeringAction {
        inout item 'item_DO_COMP_PERSONNEL_RECORD' : 'DO-COMP-PERSONNEL-RECORD';
    }

    @EngineeringElement {
        sourceId = "screen-personnel";
        conceptKind = "action_usage";
    }
    action 'screen-personnel' : EngineeringAction {
        inout item 'item_DO_COMP_PERSONNEL_RECORD' : 'DO-COMP-PERSONNEL-RECORD';
    }

    @EngineeringRelationship {
        sourceId = "REL-28069A0F8ECCA81C";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-28069A0F8ECCA81C' from 'confirm-offboarding' to 'ACT-PEOPLE-OPERATIONS';

    @EngineeringRelationship {
        sourceId = "REL-363C589B42F547F3";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-363C589B42F547F3-context' {
        part 'ACT-PEOPLE-OPERATIONS' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-COMP-PEOPLE' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-363C589B42F547F3' from 'ACT-PEOPLE-OPERATIONS'.pOut to 'FU-COMP-PEOPLE'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-3D16FF4DE93A0E75";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-3D16FF4DE93A0E75' : EngineeringVerification {
        subject 'CV-COMP-ACCESS-REVIEW' : EngineeringItem;
        objective 'CTRL-COMP-ACCESS-REVIEW' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-65FD2126419C5FD6";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-65FD2126419C5FD6' first 'complete-awareness-training' then 'review-role-change';

    @EngineeringRelationship {
        sourceId = "REL-6F17752B0206C069";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-6F17752B0206C069-context' {
        part 'ACT-PEOPLE-OPERATIONS' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-COMP-PEOPLE' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-6F17752B0206C069' from 'ACT-PEOPLE-OPERATIONS'.pOut to 'FU-COMP-PEOPLE'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-734858B0098C4EBB";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-734858B0098C4EBB' : EngineeringVerification {
        subject 'CV-COMP-AWARENESS' : EngineeringItem;
        objective 'CTRL-COMP-SECURITY-AWARENESS' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-7728FAFFF363A229";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-7728FAFFF363A229' first 'screen-personnel' then 'acknowledge-policy';

    @EngineeringRelationship {
        sourceId = "REL-789820191DAD357E";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-789820191DAD357E' first 'review-role-change' then 'confirm-offboarding';

    @EngineeringRelationship {
        sourceId = "REL-7A9CDEED407E213C";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-7A9CDEED407E213C-context' {
        part 'ACT-LINE-MANAGER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-COMP-PEOPLE' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-7A9CDEED407E213C' from 'ACT-LINE-MANAGER'.pOut to 'FU-COMP-PEOPLE'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-852460B1BEEE89A5";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-852460B1BEEE89A5' from 'complete-awareness-training' to 'ACT-EMPLOYEE';

    @EngineeringRelationship {
        sourceId = "REL-964E8D95E530D32B";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-964E8D95E530D32B' : EngineeringVerification {
        subject 'CV-COMP-ROLE-CHANGE' : EngineeringItem;
        objective 'CTRL-COMP-ROLE-CHANGE-REVIEW' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-B13E8F2AE8B70978";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-B13E8F2AE8B70978' : EngineeringVerification {
        subject 'CV-COMP-OFFBOARDING' : EngineeringItem;
        objective 'CTRL-COMP-OFFBOARDING' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-B9097EC127B528D8";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-B9097EC127B528D8-context' {
        part 'ACT-EMPLOYEE' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-COMP-PEOPLE' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-B9097EC127B528D8' from 'ACT-EMPLOYEE'.pOut to 'FU-COMP-PEOPLE'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-BB36D9ABF12944FB";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-BB36D9ABF12944FB-context' {
        part 'ACT-PEOPLE-OPERATIONS' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-COMP-PEOPLE' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-BB36D9ABF12944FB' from 'ACT-PEOPLE-OPERATIONS'.pOut to 'FU-COMP-PEOPLE'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-BB70C7E2D88FF082";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-BB70C7E2D88FF082-context' {
        part 'ACT-EMPLOYEE' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-COMP-PEOPLE' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-BB70C7E2D88FF082' from 'ACT-EMPLOYEE'.pOut to 'FU-COMP-PEOPLE'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-C5EAA518A9E51227";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-C5EAA518A9E51227' first 'acknowledge-policy' then 'complete-awareness-training';

    @EngineeringRelationship {
        sourceId = "REL-CED2929569D59EFF";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-CED2929569D59EFF' : EngineeringVerification {
        subject 'CV-COMP-PERSONNEL-SCREENING' : EngineeringItem;
        objective 'CTRL-COMP-PERSONNEL-SCREENING' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-D35CDEB4E8E4AC01";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-D35CDEB4E8E4AC01' from 'acknowledge-policy' to 'ACT-EMPLOYEE';

    @EngineeringRelationship {
        sourceId = "REL-E19CC6D48556D2D4";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-E19CC6D48556D2D4' from 'screen-personnel' to 'ACT-PEOPLE-OPERATIONS';

    @EngineeringRelationship {
        sourceId = "REL-E4B230F1794E080D";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-E4B230F1794E080D' from 'review-role-change' to 'ACT-LINE-MANAGER';
}
