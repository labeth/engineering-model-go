// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'atlas-orion-relay' {
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
        sourceId = "atlas-orion-relay";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-ORI-CONTROLLER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-ORI-CONTROLLER';

    @EngineeringElement {
        sourceId = "ACT-ORI-ENGINEER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-ORI-ENGINEER';

    @EngineeringElement {
        sourceId = "ADR-ORI-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-ORI-001';

    @EngineeringElement {
        sourceId = "AV-ORI-MESSAGE-FLOOD";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-ORI-MESSAGE-FLOOD';

    @EngineeringElement {
        sourceId = "CAP-ORION-MISSION";
        conceptKind = "port_usage";
    }
    port 'CAP-ORION-MISSION' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-319A980F183C89B0";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-319A980F183C89B0' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Authenticated priority queues preserve critical capacity.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-400729559B36CB12";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-400729559B36CB12' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Link state and priority drive deterministic transmission windows.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-98F930860EBAB1F5";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-98F930860EBAB1F5' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Product radio scheduling is deployed over shared edge, cloud, identity, and compliance contracts.";
    }

    @EngineeringElement {
        sourceId = "CTRL-ORI-ADMISSION";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-ORI-ADMISSION';

    @EngineeringElement {
        sourceId = "CV-ORI-ADMISSION";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-ORI-ADMISSION' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "DATA-ORI-MESSAGE";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-ORI-MESSAGE';

    @EngineeringElement {
        sourceId = "DO-ORI-MESSAGE";
        conceptKind = "item_definition";
    }
    item def 'DO-ORI-MESSAGE' {
        attribute 'classification' : 'String' = "operational-message";
    }

    @EngineeringElement {
        sourceId = "EVIDENCE-37705544565667A5";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-37705544565667A5';

    @EngineeringElement {
        sourceId = "EVIDENCE-D2CEAB3DEB616658";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-D2CEAB3DEB616658';

    @EngineeringElement {
        sourceId = "FG-ORI";
        conceptKind = "part_definition";
    }
    part def 'FG-ORI';

    @EngineeringElement {
        sourceId = "FU-ORI-LINK";
        conceptKind = "part_usage";
    }
    part 'FU-ORI-LINK' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-ORI-ROUTER";
        conceptKind = "part_usage";
    }
    part 'FU-ORI-ROUTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "HW-ORI-ANTENNA";
        conceptKind = "part_definition";
    }
    part def 'HW-ORI-ANTENNA' {
        attribute 'kind' : 'String' = "actuator";
        attribute 'partNumber' : 'String' = "OR-A15";
        attribute 'supplier' : 'String' = "Atlas Industries";
    }

    @EngineeringElement {
        sourceId = "HW-ORI-RADIO";
        conceptKind = "part_definition";
    }
    part def 'HW-ORI-RADIO' {
        attribute 'kind' : 'String' = "lru";
        attribute 'partNumber' : 'String' = "OR-R80";
        attribute 'supplier' : 'String' = "Atlas Industries";
    }

    @EngineeringElement {
        sourceId = "HWIF-ORI-RF-CONTROL";
        conceptKind = "connection_definition";
    }
    connection def 'HWIF-ORI-RF-CONTROL' {
        end item 'source' : EngineeringItem;
        end item 'target' : EngineeringItem;
        attribute 'busType' : 'String' = "ethernet";
        attribute 'direction' : 'String' = "bidirectional";
    }

    @EngineeringElement {
        sourceId = "IF-ORI-MESSAGE";
        conceptKind = "interface_definition";
    }
    interface def 'IF-ORI-MESSAGE' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "quic";
        attribute 'endpoint' : 'String' = "quic://relay/messages";
    }

    @EngineeringElement {
        sourceId = "POAM-ORI-001";
        conceptKind = "engineering_extension";
    }
    attribute 'POAM-ORI-001';

    @EngineeringElement {
        sourceId = "REQ-ORI-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-ORI-001' {
        attribute 'text' : 'String' = "The orion relay system shall deliver each relay message according to network controller priority.";
    }

    @EngineeringElement {
        sourceId = "REQ-ORI-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-ORI-002' {
        attribute 'text' : 'String' = "The orion relay system shall execute the link scheduler software on managed edge compute.";
    }

    @EngineeringElement {
        sourceId = "REQ-ORI-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-ORI-003' {
        attribute 'text' : 'String' = "The orion relay system shall retain relay configuration using managed cloud operations.";
    }

    @EngineeringElement {
        sourceId = "REQ-ORI-004";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-ORI-004' {
        attribute 'text' : 'String' = "The orion relay system shall authenticate product services using shared security services.";
    }

    @EngineeringElement {
        sourceId = "REQ-ORI-005";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-ORI-005' {
        attribute 'text' : 'String' = "The orion relay system shall retain compliance evidence using the company control baseline.";
    }

    @EngineeringElement {
        sourceId = "RISK-ORI-LINK-EXHAUSTION";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-ORI-LINK-EXHAUSTION';

    @EngineeringElement {
        sourceId = "SUB-ORI-CLOUD";
        conceptKind = "part_usage";
    }
    part 'SUB-ORI-CLOUD' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-ORI-COMPLIANCE";
        conceptKind = "part_usage";
    }
    part 'SUB-ORI-COMPLIANCE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-ORI-EDGE";
        conceptKind = "part_usage";
    }
    part 'SUB-ORI-EDGE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-ORI-SECURITY";
        conceptKind = "part_usage";
    }
    part 'SUB-ORI-SECURITY' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SYS-ORI";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-ORI';

    part 'SUB-ORI-CLOUD/cloud::CAP-CLOUD-OPERATIONS' : EngineeringPart;

    part 'SUB-ORI-COMPLIANCE/compliance::CAP-CONTROL-BASELINE' : EngineeringPart;

    part 'SUB-ORI-EDGE/edge::CAP-EDGE-COMPUTE' : EngineeringPart;

    part 'SUB-ORI-SECURITY/security::CAP-ZERO-TRUST' : EngineeringPart;

    requirement 'cloud::NEED-CLOUD-IDENTITY' : EngineeringRequirement;

    requirement 'edge::NEED-EDGE-IDENTITY' : EngineeringRequirement;

    part 'security::CAP-DEVICE-IDENTITY' : EngineeringPart;

    @EngineeringRelationship {
        sourceId = "REL-1ECC77DFA3AF4D7D";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-1ECC77DFA3AF4D7D-context' {
        part 'REQ-ORI-004' : EngineeringPart;
        part 'SUB-ORI-SECURITY/security::CAP-ZERO-TRUST' : EngineeringPart;
        allocation 'REL-1ECC77DFA3AF4D7D' allocate 'REQ-ORI-004' to 'SUB-ORI-SECURITY/security::CAP-ZERO-TRUST';
    }

    @EngineeringRelationship {
        sourceId = "REL-3007AE47ACBC6155";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-3007AE47ACBC6155-context' {
        part 'REQ-ORI-005' : EngineeringPart;
        part 'SUB-ORI-COMPLIANCE/compliance::CAP-CONTROL-BASELINE' : EngineeringPart;
        allocation 'REL-3007AE47ACBC6155' allocate 'REQ-ORI-005' to 'SUB-ORI-COMPLIANCE/compliance::CAP-CONTROL-BASELINE';
    }

    @EngineeringRelationship {
        sourceId = "REL-4EBC91E7BEE130FA";
        conceptKind = "satisfaction";
        relationshipKind = "satisfaction";
    }
    requirement def 'REL-4EBC91E7BEE130FA-context' {
        part satisfyingPart : EngineeringPart;
        satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;
    }

    @EngineeringRelationship {
        sourceId = "REL-5D9466EBF0DD1F54";
        conceptKind = "satisfaction";
        relationshipKind = "satisfaction";
    }
    requirement def 'REL-5D9466EBF0DD1F54-context' {
        part satisfyingPart : EngineeringPart;
        satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;
    }

    @EngineeringRelationship {
        sourceId = "REL-77C6B1E749C1FAA3";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-77C6B1E749C1FAA3-context' {
        part 'REQ-ORI-003' : EngineeringPart;
        part 'SUB-ORI-CLOUD/cloud::CAP-CLOUD-OPERATIONS' : EngineeringPart;
        allocation 'REL-77C6B1E749C1FAA3' allocate 'REQ-ORI-003' to 'SUB-ORI-CLOUD/cloud::CAP-CLOUD-OPERATIONS';
    }

    @EngineeringRelationship {
        sourceId = "REL-B9A301F3DFE62F87";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-B9A301F3DFE62F87' : EngineeringVerification {
        subject 'CV-ORI-ADMISSION' : EngineeringItem;
        objective 'CTRL-ORI-ADMISSION' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-C1BB9A366F3A5CB9";
        conceptKind = "connection";
        relationshipKind = "hardware_interface";
    }
    part def 'REL-C1BB9A366F3A5CB9-context' {
        part source : 'HW-ORI-RADIO';
        part target : 'HW-ORI-ANTENNA';
        connection 'REL-C1BB9A366F3A5CB9' connect source to target;
    }

    @EngineeringRelationship {
        sourceId = "REL-D527153375A79290";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-D527153375A79290-context' {
        part 'REQ-ORI-002' : EngineeringPart;
        part 'SUB-ORI-EDGE/edge::CAP-EDGE-COMPUTE' : EngineeringPart;
        allocation 'REL-D527153375A79290' allocate 'REQ-ORI-002' to 'SUB-ORI-EDGE/edge::CAP-EDGE-COMPUTE';
    }

    // Publication public from models.atlas.example/shared/cloud-platform@v0@v0.1.0 (atlas-shared-cloud-platform).
    package 'cloud' {
        part def 'ACT-CLOUD-OPERATOR';
        part def 'CAP-CLOUD-OPERATIONS';
        part def 'DO-CLOUD-EVENT';
        part def 'FG-CLOUD';
        part def 'FU-CLOUD-RUNTIME';
        part def 'HW-CLOUD-REGION';
        port def 'IF-CLOUD-EVENTS';
        requirement def 'REQ-CLOUD-001';
        requirement def 'REQ-CLOUD-002';
    }

    // Publication public from models.atlas.example/shared/compliance-baseline@v0@v0.1.0 (atlas-shared-compliance-baseline).
    package 'compliance' {
        part def 'ACT-COMP-OFFICER';
        part def 'ACT-EMPLOYEE';
        part def 'ACT-LINE-MANAGER';
        part def 'ACT-PEOPLE-OPERATIONS';
        part def 'CAP-CONTROL-BASELINE';
        part def 'DO-COMP-EVIDENCE';
        part def 'DO-COMP-PERSONNEL-RECORD';
        part def 'DO-COMP-TRAINING-RECORD';
        part def 'FG-COMP';
        action def 'FLOW-COMP-EMPLOYEE-LIFECYCLE';
        part def 'FU-COMP-EVIDENCE';
        part def 'FU-COMP-PEOPLE';
        requirement def 'REQ-COMP-001';
        requirement def 'REQ-COMP-002';
        requirement def 'REQ-COMP-003';
        requirement def 'REQ-COMP-004';
        requirement def 'REQ-COMP-005';
        requirement def 'REQ-COMP-006';
        requirement def 'REQ-COMP-007';
        requirement def 'REQ-COMP-008';
    }

    // Publication public from models.atlas.example/shared/edge-platform@v0@v0.1.0 (atlas-shared-edge-platform).
    package 'edge' {
        part def 'ACT-EDGE-OPERATOR';
        part def 'CAP-EDGE-COMPUTE';
        part def 'DO-EDGE-TELEMETRY';
        part def 'FG-EDGE';
        part def 'FU-EDGE-RUNTIME';
        part def 'HW-EDGE-COMPUTE';
        port def 'IF-EDGE-TELEMETRY';
        requirement def 'REQ-EDGE-001';
        requirement def 'REQ-EDGE-002';
    }

    // Publication public from models.atlas.example/shared/security-services@v0@v0.1.0 (atlas-shared-security-services).
    package 'security' {
        part def 'ACT-SEC-ANALYST';
        part def 'CAP-ZERO-TRUST';
        part def 'DO-SEC-AUDIT-EVENT';
        action def 'EVT-CREDENTIAL-REPLAY-DETECTED';
        part def 'FG-SEC';
        part def 'FU-SEC-IDENTITY';
        part def 'FU-SEC-MONITORING';
        port def 'IF-SEC-TOKEN';
        requirement def 'REQ-SEC-001';
        requirement def 'REQ-SEC-002';
    }
}
