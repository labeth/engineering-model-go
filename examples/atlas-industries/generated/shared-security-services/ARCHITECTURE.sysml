// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'atlas-shared-security-services' {
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
        sourceId = "atlas-shared-security-services";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-SEC-ANALYST";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-SEC-ANALYST';

    @EngineeringElement {
        sourceId = "ADR-SEC-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-SEC-001';

    @EngineeringElement {
        sourceId = "AV-CREDENTIAL-REPLAY";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-CREDENTIAL-REPLAY';

    @EngineeringElement {
        sourceId = "CAP-DEVICE-IDENTITY";
        conceptKind = "port_usage";
    }
    port 'CAP-DEVICE-IDENTITY' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CAP-ZERO-TRUST";
        conceptKind = "port_usage";
    }
    port 'CAP-ZERO-TRUST' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-0E68B43DA0B05D35";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0E68B43DA0B05D35' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Identity issuance and audit monitoring operate in a privileged control plane.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8392B1D6E79B35D3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8392B1D6E79B35D3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Credentials are scoped, short-lived, rotated, and validated at every boundary.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A1F1F8AF73F3AE2D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A1F1F8AF73F3AE2D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Signed audit events feed replay and anomaly detection.";
    }

    @EngineeringElement {
        sourceId = "CTRL-SEC-AUDIT-INTEGRITY";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-SEC-AUDIT-INTEGRITY';

    @EngineeringElement {
        sourceId = "CTRL-SEC-SHORT-LIVED-IDENTITY";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-SEC-SHORT-LIVED-IDENTITY';

    @EngineeringElement {
        sourceId = "CV-SEC-TOKEN-REPLAY";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-SEC-TOKEN-REPLAY' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "DATA-AUDIT-EVENT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-AUDIT-EVENT';

    @EngineeringElement {
        sourceId = "DO-SEC-AUDIT-EVENT";
        conceptKind = "item_definition";
    }
    item def 'DO-SEC-AUDIT-EVENT' {
        attribute 'classification' : 'String' = "security-audit";
    }

    @EngineeringElement {
        sourceId = "EVIDENCE-34978061F7EE8317";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-34978061F7EE8317';

    @EngineeringElement {
        sourceId = "EVIDENCE-F4E6B8C9F13DAB03";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-F4E6B8C9F13DAB03';

    @EngineeringElement {
        sourceId = "EVT-CREDENTIAL-REPLAY-DETECTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-CREDENTIAL-REPLAY-DETECTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "FG-SEC";
        conceptKind = "part_definition";
    }
    part def 'FG-SEC';

    @EngineeringElement {
        sourceId = "FU-SEC-IDENTITY";
        conceptKind = "part_usage";
    }
    part 'FU-SEC-IDENTITY' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-SEC-MONITORING";
        conceptKind = "part_usage";
    }
    part 'FU-SEC-MONITORING' : EngineeringPart;

    @EngineeringElement {
        sourceId = "IF-SEC-TOKEN";
        conceptKind = "interface_definition";
    }
    interface def 'IF-SEC-TOKEN' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "https://identity.atlas.example/token";
    }

    @EngineeringElement {
        sourceId = "MIT-SEC-TOKEN-REPLAY";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-SEC-TOKEN-REPLAY';

    @EngineeringElement {
        sourceId = "REQ-SEC-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-SEC-001' {
        attribute 'text' : 'String' = "The atlas security services shall issue short-lived identity credentials using the identity service.";
    }

    @EngineeringElement {
        sourceId = "REQ-SEC-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-SEC-002' {
        attribute 'text' : 'String' = "When credential replay is detected, the atlas security services shall notify the security analyst.";
    }

    @EngineeringElement {
        sourceId = "RISK-SEC-CREDENTIAL-REPLAY";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-SEC-CREDENTIAL-REPLAY';

    @EngineeringElement {
        sourceId = "SYS-SEC";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-SEC';

    @EngineeringElement {
        sourceId = "TB-SEC-CONTROL-PLANE";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-SEC-CONTROL-PLANE';

    @EngineeringElement {
        sourceId = "TS-SEC-CREDENTIAL-REPLAY";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-SEC-CREDENTIAL-REPLAY';

    @EngineeringRelationship {
        sourceId = "REL-D91E13DDD7620127";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-D91E13DDD7620127' : EngineeringVerification {
        subject 'CV-SEC-TOKEN-REPLAY' : EngineeringItem;
        objective 'CTRL-SEC-SHORT-LIVED-IDENTITY' : EngineeringRequirement;
    }
}
