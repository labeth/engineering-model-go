// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'MODEL-A' {
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
        sourceId = "MODEL-A";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "EVT-A";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-A' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-B";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-B' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "FU-A";
        conceptKind = "part_usage";
    }
    part 'FU-A' : EngineeringPart;

    @EngineeringElement {
        sourceId = "REQ-A";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-A';

    @EngineeringElement {
        sourceId = "SM-PAY";
        conceptKind = "state_definition";
    }
    state def 'SM-PAY' {
        doc /* Authorizes a payment. Implemented by FU-A. */
        attribute attempts : Integer := 0;
        attribute approved : Boolean := false;
        attribute reason : String := "none";
        entry;
        then 'STATE-PAY-IDLE';
        state 'STATE-PAY-AUTH';
        @EngineeringElement about 'STATE-PAY-AUTH' {
            sourceId = "STATE-PAY-AUTH";
            conceptKind = "state_usage";
        }
        state 'STATE-PAY-DONE';
        @EngineeringElement about 'STATE-PAY-DONE' {
            sourceId = "STATE-PAY-DONE";
            conceptKind = "state_usage";
        }
        state 'STATE-PAY-IDLE';
        @EngineeringElement about 'STATE-PAY-IDLE' {
            sourceId = "STATE-PAY-IDLE";
            conceptKind = "state_usage";
        }
        succession 'REL-805FEC22BFCBFF50' first 'STATE-PAY-DONE' then done;
        @EngineeringRelationship about 'REL-805FEC22BFCBFF50' {
            sourceId = "REL-805FEC22BFCBFF50";
            conceptKind = "succession";
            relationshipKind = "machine_final";
        }
        transition 'TR-PAY-001'
            first 'STATE-PAY-IDLE'
            accept 'on-EVT-A' :> 'EVT-A'
            if attempts < 3
            do assign attempts := attempts + 1
            then 'STATE-PAY-AUTH' {
                doc /* Start */
            }
        @EngineeringRelationship about 'TR-PAY-001' {
            sourceId = "TR-PAY-001";
            conceptKind = "transition";
            relationshipKind = "machine_transition";
        }
        transition 'TR-PAY-002'
            first 'STATE-PAY-AUTH'
            if approved
            then 'STATE-PAY-DONE';
        @EngineeringRelationship about 'TR-PAY-002' {
            sourceId = "TR-PAY-002";
            conceptKind = "transition";
            relationshipKind = "machine_transition";
        }
        constraint 'INV-PAY-001' {
            doc /* Bounded. Protects REQ-A. */
            attempts <= 3
        }
        @EngineeringElement about 'INV-PAY-001' {
            sourceId = "INV-PAY-001";
            conceptKind = "constraint_usage";
        }
    }
}
