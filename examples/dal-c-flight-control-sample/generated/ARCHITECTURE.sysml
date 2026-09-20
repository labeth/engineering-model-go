// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'DAL-C-FLIGHT-CONTROL-SAMPLE' {
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
        sourceId = "DAL-C-FLIGHT-CONTROL-SAMPLE";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "EVT-YAW-RATE-RECEIVED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-YAW-RATE-RECEIVED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "FG-FLIGHT-CONTROL";
        conceptKind = "part_definition";
    }
    part def 'FG-FLIGHT-CONTROL';

    @EngineeringElement {
        sourceId = "FU-YAW-CONTROL";
        conceptKind = "part_usage";
    }
    part 'FU-YAW-CONTROL' : EngineeringPart;

    @EngineeringElement {
        sourceId = "HW-FLIGHT-CONTROL-COMPUTER";
        conceptKind = "part_definition";
    }
    part def 'HW-FLIGHT-CONTROL-COMPUTER' {
        attribute 'kind' : 'String' = "computer";
    }

    @EngineeringElement {
        sourceId = "IF-RUDDER-COMMAND";
        conceptKind = "interface_definition";
    }
    interface def 'IF-RUDDER-COMMAND' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "sampled-data";
        attribute 'endpoint' : 'String' = "output";
    }

    @EngineeringElement {
        sourceId = "IF-YAW-RATE";
        conceptKind = "interface_definition";
    }
    interface def 'IF-YAW-RATE' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "sampled-data";
        attribute 'endpoint' : 'String' = "input";
    }

    @EngineeringElement {
        sourceId = "REQ-YD-HLR-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-YD-HLR-001' {
        attribute 'text' : 'String' = "When yaw rate input is received, the yaw damping software shall filter the yaw rate input.";
        attribute 'notes' : 'String' = "High-level software requirement.";
    }

    @EngineeringElement {
        sourceId = "REQ-YD-LLR-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-YD-LLR-001' {
        attribute 'text' : 'String' = "When yaw rate input is received, the yaw damping software shall limit the bounded rudder command.";
        attribute 'notes' : 'String' = "Derived low-level limit requiring safety feedback.";
    }

    @EngineeringElement {
        sourceId = "REQ-YD-SYS-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-YD-SYS-001' {
        attribute 'text' : 'String' = "When yaw rate input is received, the yaw damping software shall compute a bounded rudder command.";
        attribute 'notes' : 'String' = "System-allocated behavior.";
    }

    @EngineeringElement {
        sourceId = "STATE-OPERATIONAL";
        conceptKind = "state_definition";
    }
    state def 'STATE-OPERATIONAL';

    @EngineeringElement {
        sourceId = "TERM-RUDDER-COMMAND";
        conceptKind = "engineering_extension";
    }
    attribute 'TERM-RUDDER-COMMAND';

    @EngineeringElement {
        sourceId = "TERM-YAW-RATE";
        conceptKind = "engineering_extension";
    }
    attribute 'TERM-YAW-RATE';
}
