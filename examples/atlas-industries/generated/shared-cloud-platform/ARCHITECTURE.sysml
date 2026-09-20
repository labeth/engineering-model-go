// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'atlas-shared-cloud-platform' {
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
        sourceId = "atlas-shared-cloud-platform";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-CLOUD-OPERATOR";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-CLOUD-OPERATOR';

    @EngineeringElement {
        sourceId = "ADR-CLOUD-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-CLOUD-001';

    @EngineeringElement {
        sourceId = "CAP-CLOUD-OPERATIONS";
        conceptKind = "port_usage";
    }
    port 'CAP-CLOUD-OPERATIONS' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-077D651C5F80637F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-077D651C5F80637F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "The runtime hides availability and event durability mechanics from products.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2C6F00C7DD53B2D5";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2C6F00C7DD53B2D5' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Product services use isolated accounts and durable regional event stores.";
    }

    @EngineeringElement {
        sourceId = "CTRL-CLOUD-BACKUP";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-CLOUD-BACKUP';

    @EngineeringElement {
        sourceId = "DATA-CLOUD-EVENT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-CLOUD-EVENT';

    @EngineeringElement {
        sourceId = "DO-CLOUD-EVENT";
        conceptKind = "item_definition";
    }
    item def 'DO-CLOUD-EVENT' {
        attribute 'classification' : 'String' = "product-operations";
    }

    @EngineeringElement {
        sourceId = "FG-CLOUD";
        conceptKind = "part_definition";
    }
    part def 'FG-CLOUD';

    @EngineeringElement {
        sourceId = "FU-CLOUD-RUNTIME";
        conceptKind = "part_usage";
    }
    part 'FU-CLOUD-RUNTIME' : EngineeringPart;

    @EngineeringElement {
        sourceId = "HW-CLOUD-REGION";
        conceptKind = "part_definition";
    }
    part def 'HW-CLOUD-REGION' {
        attribute 'kind' : 'String' = "cloud";
    }

    @EngineeringElement {
        sourceId = "IF-CLOUD-EVENTS";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLOUD-EVENTS' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "https://events.atlas.example/v1";
    }

    @EngineeringElement {
        sourceId = "NEED-CLOUD-IDENTITY";
        conceptKind = "port_usage";
    }
    port 'NEED-CLOUD-IDENTITY' : ~'EngineeringPort' {
        in port 'required_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "REQ-CLOUD-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-CLOUD-001' {
        attribute 'text' : 'String' = "The atlas cloud platform shall process each cloud event using the cloud runtime.";
    }

    @EngineeringElement {
        sourceId = "REQ-CLOUD-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-CLOUD-002' {
        attribute 'text' : 'String' = "The atlas cloud platform shall authenticate the cloud runtime using workload identity.";
    }

    @EngineeringElement {
        sourceId = "SYS-CLOUD";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-CLOUD';

    @EngineeringRelationship {
        sourceId = "REL-A501C29BE538CCE5";
        conceptKind = "allocation";
        relationshipKind = "hosts";
    }
    part def 'REL-A501C29BE538CCE5-context' {
        part 'FU-CLOUD-RUNTIME' : EngineeringPart;
        part 'HW-CLOUD-REGION' : EngineeringPart;
        allocation 'REL-A501C29BE538CCE5' allocate 'FU-CLOUD-RUNTIME' to 'HW-CLOUD-REGION';
    }
}
