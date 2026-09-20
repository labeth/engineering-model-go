// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'atlas-aegis-sentinel' {
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
        sourceId = "atlas-aegis-sentinel";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-AEG-ENGINEER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-AEG-ENGINEER';

    @EngineeringElement {
        sourceId = "ACT-AEG-OPERATOR";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-AEG-OPERATOR';

    @EngineeringElement {
        sourceId = "ADR-AEG-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-AEG-001';

    @EngineeringElement {
        sourceId = "AV-AEG-SENSOR-SPOOF";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-AEG-SENSOR-SPOOF';

    @EngineeringElement {
        sourceId = "CAP-AEGIS-MISSION";
        conceptKind = "port_usage";
    }
    port 'CAP-AEGIS-MISSION' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-7DC4E694DCFF40F3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-7DC4E694DCFF40F3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Only confidence-gated tracks enter the operator workflow.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BD92F868B93E76BC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BD92F868B93E76BC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Cross-sensor confidence gates protect track integrity.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C1C2FDFDDCF9477D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C1C2FDFDDCF9477D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Product-specific sensing is deployed over shared edge, cloud, identity, and compliance contracts.";
    }

    @EngineeringElement {
        sourceId = "CTRL-AEG-TRACK-CONFIDENCE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-AEG-TRACK-CONFIDENCE';

    @EngineeringElement {
        sourceId = "CV-AEG-TRACK-CONFIDENCE";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-AEG-TRACK-CONFIDENCE' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "DATA-AEG-TRACK";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-AEG-TRACK';

    @EngineeringElement {
        sourceId = "DO-AEG-TRACK";
        conceptKind = "item_definition";
    }
    item def 'DO-AEG-TRACK' {
        attribute 'classification' : 'String' = "mission";
    }

    @EngineeringElement {
        sourceId = "EVIDENCE-9E9420F9C7AB726C";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-9E9420F9C7AB726C';

    @EngineeringElement {
        sourceId = "FG-AEG";
        conceptKind = "part_definition";
    }
    part def 'FG-AEG';

    @EngineeringElement {
        sourceId = "FU-AEG-MISSION";
        conceptKind = "part_usage";
    }
    part 'FU-AEG-MISSION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-AEG-SENSOR-FUSION";
        conceptKind = "part_usage";
    }
    part 'FU-AEG-SENSOR-FUSION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "HW-AEG-OPTICAL";
        conceptKind = "part_definition";
    }
    part def 'HW-AEG-OPTICAL' {
        attribute 'kind' : 'String' = "sensor";
        attribute 'partNumber' : 'String' = "AS-O20";
        attribute 'supplier' : 'String' = "Atlas Industries";
    }

    @EngineeringElement {
        sourceId = "HW-AEG-RADAR";
        conceptKind = "part_definition";
    }
    part def 'HW-AEG-RADAR' {
        attribute 'kind' : 'String' = "sensor";
        attribute 'partNumber' : 'String' = "AS-R40";
        attribute 'supplier' : 'String' = "Atlas Industries";
    }

    @EngineeringElement {
        sourceId = "HWIF-AEG-SENSOR-BUS";
        conceptKind = "connection_definition";
    }
    connection def 'HWIF-AEG-SENSOR-BUS' {
        end item 'source' : EngineeringItem;
        end item 'target' : EngineeringItem;
        attribute 'busType' : 'String' = "ethernet";
        attribute 'direction' : 'String' = "bidirectional";
    }

    @EngineeringElement {
        sourceId = "IF-AEG-TRACK";
        conceptKind = "interface_definition";
    }
    interface def 'IF-AEG-TRACK' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "protobuf";
        attribute 'endpoint' : 'String' = "grpc://mission/tracks";
    }

    @EngineeringElement {
        sourceId = "MIT-AEG-TRACK-CONFIDENCE";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-AEG-TRACK-CONFIDENCE';

    @EngineeringElement {
        sourceId = "REQ-AEG-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-AEG-001' {
        attribute 'text' : 'String' = "The aegis sentinel system shall present each perimeter track to the security operator.";
    }

    @EngineeringElement {
        sourceId = "REQ-AEG-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-AEG-002' {
        attribute 'text' : 'String' = "The aegis sentinel system shall execute the sensor fusion software on managed edge compute.";
    }

    @EngineeringElement {
        sourceId = "REQ-AEG-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-AEG-003' {
        attribute 'text' : 'String' = "The aegis sentinel system shall retain each perimeter track using managed cloud operations.";
    }

    @EngineeringElement {
        sourceId = "REQ-AEG-004";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-AEG-004' {
        attribute 'text' : 'String' = "The aegis sentinel system shall authenticate product services using shared security services.";
    }

    @EngineeringElement {
        sourceId = "REQ-AEG-005";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-AEG-005' {
        attribute 'text' : 'String' = "The aegis sentinel system shall retain compliance evidence using the company control baseline.";
    }

    @EngineeringElement {
        sourceId = "RISK-AEG-FALSE-TRACK";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-AEG-FALSE-TRACK';

    @EngineeringElement {
        sourceId = "SUB-AEG-CLOUD";
        conceptKind = "part_usage";
    }
    part 'SUB-AEG-CLOUD' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-AEG-COMPLIANCE";
        conceptKind = "part_usage";
    }
    part 'SUB-AEG-COMPLIANCE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-AEG-EDGE";
        conceptKind = "part_usage";
    }
    part 'SUB-AEG-EDGE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-AEG-SECURITY";
        conceptKind = "part_usage";
    }
    part 'SUB-AEG-SECURITY' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SYS-AEG";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-AEG';

    @EngineeringElement {
        sourceId = "TS-AEG-SENSOR-SPOOF";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-AEG-SENSOR-SPOOF';

    part 'SUB-AEG-CLOUD/cloud::CAP-CLOUD-OPERATIONS' : EngineeringPart;

    part 'SUB-AEG-COMPLIANCE/compliance::CAP-CONTROL-BASELINE' : EngineeringPart;

    part 'SUB-AEG-EDGE/edge::CAP-EDGE-COMPUTE' : EngineeringPart;

    part 'SUB-AEG-SECURITY/security::CAP-ZERO-TRUST' : EngineeringPart;

    requirement 'cloud::NEED-CLOUD-IDENTITY' : EngineeringRequirement;

    requirement 'edge::NEED-EDGE-IDENTITY' : EngineeringRequirement;

    part 'security::CAP-DEVICE-IDENTITY' : EngineeringPart;

    @EngineeringRelationship {
        sourceId = "REL-5D08DE35DFFD6F53";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-5D08DE35DFFD6F53' : EngineeringVerification {
        subject 'CV-AEG-TRACK-CONFIDENCE' : EngineeringItem;
        objective 'CTRL-AEG-TRACK-CONFIDENCE' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-9B105138F82B9A7A";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-9B105138F82B9A7A-context' {
        part 'REQ-AEG-004' : EngineeringPart;
        part 'SUB-AEG-SECURITY/security::CAP-ZERO-TRUST' : EngineeringPart;
        allocation 'REL-9B105138F82B9A7A' allocate 'REQ-AEG-004' to 'SUB-AEG-SECURITY/security::CAP-ZERO-TRUST';
    }

    @EngineeringRelationship {
        sourceId = "REL-A15A50DDC75E7612";
        conceptKind = "satisfaction";
        relationshipKind = "satisfaction";
    }
    requirement def 'REL-A15A50DDC75E7612-context' {
        part satisfyingPart : EngineeringPart;
        satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;
    }

    @EngineeringRelationship {
        sourceId = "REL-A47F7283A8594020";
        conceptKind = "satisfaction";
        relationshipKind = "satisfaction";
    }
    requirement def 'REL-A47F7283A8594020-context' {
        part satisfyingPart : EngineeringPart;
        satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;
    }

    @EngineeringRelationship {
        sourceId = "REL-B68E3259B6568EDA";
        conceptKind = "connection";
        relationshipKind = "hardware_interface";
    }
    part def 'REL-B68E3259B6568EDA-context' {
        part source : 'HW-AEG-RADAR';
        part target : 'HW-AEG-OPTICAL';
        connection 'REL-B68E3259B6568EDA' connect source to target;
    }

    @EngineeringRelationship {
        sourceId = "REL-BAB141DBF9BC04D6";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-BAB141DBF9BC04D6-context' {
        part 'REQ-AEG-002' : EngineeringPart;
        part 'SUB-AEG-EDGE/edge::CAP-EDGE-COMPUTE' : EngineeringPart;
        allocation 'REL-BAB141DBF9BC04D6' allocate 'REQ-AEG-002' to 'SUB-AEG-EDGE/edge::CAP-EDGE-COMPUTE';
    }

    @EngineeringRelationship {
        sourceId = "REL-BE6D3544886A27B2";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-BE6D3544886A27B2-context' {
        part 'REQ-AEG-005' : EngineeringPart;
        part 'SUB-AEG-COMPLIANCE/compliance::CAP-CONTROL-BASELINE' : EngineeringPart;
        allocation 'REL-BE6D3544886A27B2' allocate 'REQ-AEG-005' to 'SUB-AEG-COMPLIANCE/compliance::CAP-CONTROL-BASELINE';
    }

    @EngineeringRelationship {
        sourceId = "REL-E7851CD623FEB34D";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-E7851CD623FEB34D-context' {
        part 'REQ-AEG-003' : EngineeringPart;
        part 'SUB-AEG-CLOUD/cloud::CAP-CLOUD-OPERATIONS' : EngineeringPart;
        allocation 'REL-E7851CD623FEB34D' allocate 'REQ-AEG-003' to 'SUB-AEG-CLOUD/cloud::CAP-CLOUD-OPERATIONS';
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
