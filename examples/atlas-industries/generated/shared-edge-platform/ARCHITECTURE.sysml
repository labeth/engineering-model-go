// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'atlas-shared-edge-platform' {
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
        sourceId = "atlas-shared-edge-platform";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-EDGE-OPERATOR";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-EDGE-OPERATOR';

    @EngineeringElement {
        sourceId = "ADR-EDGE-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EDGE-001';

    @EngineeringElement {
        sourceId = "CAP-EDGE-COMPUTE";
        conceptKind = "port_usage";
    }
    port 'CAP-EDGE-COMPUTE' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-80D773377BD6B2A3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-80D773377BD6B2A3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "A hardened compute boundary separates product workloads from device interfaces.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-CF481A6695600BA7";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-CF481A6695600BA7' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Secure boot and signature verification protect the execution chain.";
    }

    @EngineeringElement {
        sourceId = "CTRL-EDGE-SIGNED-WORKLOAD";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-EDGE-SIGNED-WORKLOAD';

    @EngineeringElement {
        sourceId = "DATA-EDGE-TELEMETRY";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-EDGE-TELEMETRY';

    @EngineeringElement {
        sourceId = "DO-EDGE-TELEMETRY";
        conceptKind = "item_definition";
    }
    item def 'DO-EDGE-TELEMETRY' {
        attribute 'classification' : 'String' = "operational";
    }

    @EngineeringElement {
        sourceId = "FG-EDGE";
        conceptKind = "part_definition";
    }
    part def 'FG-EDGE';

    @EngineeringElement {
        sourceId = "FU-EDGE-RUNTIME";
        conceptKind = "part_usage";
    }
    part 'FU-EDGE-RUNTIME' : EngineeringPart;

    @EngineeringElement {
        sourceId = "HW-EDGE-COMPUTE";
        conceptKind = "part_definition";
    }
    part def 'HW-EDGE-COMPUTE' {
        attribute 'kind' : 'String' = "lru";
        attribute 'partNumber' : 'String' = "AT-E100";
        attribute 'supplier' : 'String' = "Atlas Industries";
    }

    @EngineeringElement {
        sourceId = "IF-EDGE-TELEMETRY";
        conceptKind = "interface_definition";
    }
    interface def 'IF-EDGE-TELEMETRY' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "mqtt";
        attribute 'endpoint' : 'String' = "mqtt://edge/telemetry";
    }

    @EngineeringElement {
        sourceId = "NEED-EDGE-IDENTITY";
        conceptKind = "port_usage";
    }
    port 'NEED-EDGE-IDENTITY' : ~'EngineeringPort' {
        in port 'required_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "REQ-EDGE-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EDGE-001' {
        attribute 'text' : 'String' = "The atlas edge platform shall execute signed workloads on the edge runtime.";
    }

    @EngineeringElement {
        sourceId = "REQ-EDGE-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EDGE-002' {
        attribute 'text' : 'String' = "The atlas edge platform shall authenticate the edge runtime using device identity.";
    }

    @EngineeringElement {
        sourceId = "SYS-EDGE";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-EDGE';

    @EngineeringRelationship {
        sourceId = "REL-21C85E737E4DC3E5";
        conceptKind = "allocation";
        relationshipKind = "hosts";
    }
    part def 'REL-21C85E737E4DC3E5-context' {
        part 'FU-EDGE-RUNTIME' : EngineeringPart;
        part 'HW-EDGE-COMPUTE' : EngineeringPart;
        allocation 'REL-21C85E737E4DC3E5' allocate 'FU-EDGE-RUNTIME' to 'HW-EDGE-COMPUTE';
    }
}
