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
        sourceId = "ACT-SYSTEMS-ENGINEER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-SYSTEMS-ENGINEER';

    @EngineeringElement {
        sourceId = "CONCERN-2F87CC2FF83FAEB7";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2F87CC2FF83FAEB7' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Sampled yaw-rate input is transformed into a sampled bounded rudder-command output.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-35FC0851EDF063DC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-35FC0851EDF063DC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "The software receives yaw rate, filters the signal, and emits a bounded rudder command.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-9A2910DA260B2C56";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-9A2910DA260B2C56' : EngineeringConcern {
        attribute 'narrative' : 'String' = "The flight-control group contains the bounded yaw damping capability.";
    }

    @EngineeringElement {
        sourceId = "DOC-SOFTWARE-DESIGN";
        conceptKind = "view_usage";
    }
    view 'DOC-SOFTWARE-DESIGN' : EngineeringView;

    @EngineeringElement {
        sourceId = "DOC-SOFTWARE-REQUIREMENTS";
        conceptKind = "view_usage";
    }
    view 'DOC-SOFTWARE-REQUIREMENTS' : EngineeringView;

    @EngineeringElement {
        sourceId = "DOC-STAKEHOLDER-REQUIREMENTS";
        conceptKind = "view_usage";
    }
    view 'DOC-STAKEHOLDER-REQUIREMENTS' : EngineeringView;

    @EngineeringElement {
        sourceId = "DOC-SYSTEM-DESIGN";
        conceptKind = "view_usage";
    }
    view 'DOC-SYSTEM-DESIGN' : EngineeringView;

    @EngineeringElement {
        sourceId = "DOC-SYSTEM-REQUIREMENTS";
        conceptKind = "view_usage";
    }
    view 'DOC-SYSTEM-REQUIREMENTS' : EngineeringView;

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
        sourceId = "REF-SYSTEM-SAFETY-ASSESSMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-SYSTEM-SAFETY-ASSESSMENT';

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

    @EngineeringElement {
        sourceId = "VIEW-YAW-CONTROL";
        conceptKind = "view_usage";
    }
    view 'VIEW-YAW-CONTROL' : EngineeringView;

    @EngineeringRelationship {
        sourceId = "REL-06B41FB4A0C22A7B";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-06B41FB4A0C22A7B' from 'DOC-SOFTWARE-REQUIREMENTS' to 'REQ-YD-HLR-001';

    @EngineeringRelationship {
        sourceId = "REL-0A9E2ECF973A0115";
        conceptKind = "dependency";
        relationshipKind = "document_reference";
    }
    dependency 'REL-0A9E2ECF973A0115' from 'DOC-SOFTWARE-DESIGN' to 'REF-SYSTEM-SAFETY-ASSESSMENT';

    @EngineeringRelationship {
        sourceId = "REL-0D01D9B203149B62";
        conceptKind = "dependency";
        relationshipKind = "document_stakeholder";
    }
    dependency 'REL-0D01D9B203149B62' from 'DOC-SOFTWARE-DESIGN' to 'ACT-SYSTEMS-ENGINEER';

    @EngineeringRelationship {
        sourceId = "REL-1510DA30330BF617";
        conceptKind = "dependency";
        relationshipKind = "document_reference";
    }
    dependency 'REL-1510DA30330BF617' from 'DOC-SOFTWARE-REQUIREMENTS' to 'REF-SYSTEM-SAFETY-ASSESSMENT';

    @EngineeringRelationship {
        sourceId = "REL-22123F6C0E9C43F4";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-22123F6C0E9C43F4' from 'DOC-SOFTWARE-REQUIREMENTS' to 'FU-YAW-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-2A16AAEDD203E378";
        conceptKind = "dependency";
        relationshipKind = "document_reference";
    }
    dependency 'REL-2A16AAEDD203E378' from 'DOC-SYSTEM-REQUIREMENTS' to 'REF-SYSTEM-SAFETY-ASSESSMENT';

    @EngineeringRelationship {
        sourceId = "REL-36EB07109AEFD2D2";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-36EB07109AEFD2D2' from 'DOC-SOFTWARE-DESIGN' to 'FU-YAW-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-438CC2198B1F85C4";
        conceptKind = "dependency";
        relationshipKind = "document_stakeholder";
    }
    dependency 'REL-438CC2198B1F85C4' from 'DOC-SOFTWARE-REQUIREMENTS' to 'ACT-SYSTEMS-ENGINEER';

    @EngineeringRelationship {
        sourceId = "REL-53428CFF8DBBCAB1";
        conceptKind = "dependency";
        relationshipKind = "document_stakeholder";
    }
    dependency 'REL-53428CFF8DBBCAB1' from 'DOC-SYSTEM-DESIGN' to 'ACT-SYSTEMS-ENGINEER';

    @EngineeringRelationship {
        sourceId = "REL-548CEBCF4964D4B8";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-548CEBCF4964D4B8' from 'DOC-SOFTWARE-REQUIREMENTS' to 'IF-RUDDER-COMMAND';

    @EngineeringRelationship {
        sourceId = "REL-5EE2FD7F835AE6E1";
        conceptKind = "dependency";
        relationshipKind = "document_stakeholder";
    }
    dependency 'REL-5EE2FD7F835AE6E1' from 'DOC-STAKEHOLDER-REQUIREMENTS' to 'ACT-SYSTEMS-ENGINEER';

    @EngineeringRelationship {
        sourceId = "REL-60766B9154C03BD2";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-60766B9154C03BD2' from 'DOC-SYSTEM-REQUIREMENTS' to 'IF-RUDDER-COMMAND';

    @EngineeringRelationship {
        sourceId = "REL-62F395A90A90AE47";
        conceptKind = "dependency";
        relationshipKind = "document_section_content";
    }
    dependency 'REL-62F395A90A90AE47' from 'DOC-STAKEHOLDER-REQUIREMENTS' to 'FU-YAW-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-66F91EBAB7D5E58F";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-66F91EBAB7D5E58F' from 'DOC-SYSTEM-DESIGN' to 'FG-FLIGHT-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-6F8ECB1564135A98";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-6F8ECB1564135A98' from 'DOC-SYSTEM-DESIGN' to 'IF-YAW-RATE';

    @EngineeringRelationship {
        sourceId = "REL-87C4950730FEC156";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-87C4950730FEC156' from 'DOC-SYSTEM-REQUIREMENTS' to 'IF-YAW-RATE';

    @EngineeringRelationship {
        sourceId = "REL-90C3E39CF063C893";
        conceptKind = "dependency";
        relationshipKind = "document_stakeholder";
    }
    dependency 'REL-90C3E39CF063C893' from 'DOC-SYSTEM-REQUIREMENTS' to 'ACT-SYSTEMS-ENGINEER';

    @EngineeringRelationship {
        sourceId = "REL-9509200568C2C80E";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-9509200568C2C80E' from 'DOC-STAKEHOLDER-REQUIREMENTS' to 'REQ-YD-SYS-001';

    @EngineeringRelationship {
        sourceId = "REL-9E828CB205B20D69";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-9E828CB205B20D69' from 'DOC-SOFTWARE-REQUIREMENTS' to 'IF-YAW-RATE';

    @EngineeringRelationship {
        sourceId = "REL-A6B37280C7B41AFA";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-A6B37280C7B41AFA' from 'DOC-SYSTEM-REQUIREMENTS' to 'REQ-YD-SYS-001';

    @EngineeringRelationship {
        sourceId = "REL-AB6EAAFD96EE720C";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-AB6EAAFD96EE720C' from 'DOC-SYSTEM-DESIGN' to 'FU-YAW-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-AE94902E6FBF7266";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-AE94902E6FBF7266' from 'DOC-SOFTWARE-DESIGN' to 'VIEW-YAW-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-B192A3256BFD2AC8";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-B192A3256BFD2AC8' from 'DOC-SYSTEM-DESIGN' to 'VIEW-YAW-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-B650521AE28EE54E";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-B650521AE28EE54E' from 'DOC-SYSTEM-DESIGN' to 'IF-RUDDER-COMMAND';

    @EngineeringRelationship {
        sourceId = "REL-B9C47DCFC207E184";
        conceptKind = "dependency";
        relationshipKind = "document_reference";
    }
    dependency 'REL-B9C47DCFC207E184' from 'DOC-STAKEHOLDER-REQUIREMENTS' to 'REF-SYSTEM-SAFETY-ASSESSMENT';

    @EngineeringRelationship {
        sourceId = "REL-BAC5F6F6AEB8A260";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-BAC5F6F6AEB8A260' from 'DOC-STAKEHOLDER-REQUIREMENTS' to 'VIEW-YAW-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-CD4457B12C5056C2";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-CD4457B12C5056C2' from 'DOC-SOFTWARE-DESIGN' to 'IF-RUDDER-COMMAND';

    @EngineeringRelationship {
        sourceId = "REL-DCCB57C637468116";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-DCCB57C637468116' from 'DOC-SOFTWARE-DESIGN' to 'IF-YAW-RATE';

    @EngineeringRelationship {
        sourceId = "REL-E588E0BB6EF6A961";
        conceptKind = "dependency";
        relationshipKind = "document_reference";
    }
    dependency 'REL-E588E0BB6EF6A961' from 'DOC-SYSTEM-DESIGN' to 'REF-SYSTEM-SAFETY-ASSESSMENT';

    @EngineeringRelationship {
        sourceId = "REL-EA31AFF8E3B4DCDE";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-EA31AFF8E3B4DCDE' from 'DOC-SOFTWARE-REQUIREMENTS' to 'REQ-YD-LLR-001';

    @EngineeringRelationship {
        sourceId = "REL-EC3599EA2BB95B82";
        conceptKind = "dependency";
        relationshipKind = "document_content";
    }
    dependency 'REL-EC3599EA2BB95B82' from 'DOC-SYSTEM-REQUIREMENTS' to 'FU-YAW-CONTROL';
}
