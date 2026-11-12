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

    @EngineeringElement {
        sourceId = "SM-RICH";
        conceptKind = "state_definition";
    }
    state def 'SM-RICH' {
        doc /* Uses every operator. Implemented by FU-A. */
        attribute count : Integer := -1;
        attribute flag : Boolean := true;
        attribute label : String := "ready \"set\"";
        attribute 'state' : Integer := 0;
        attribute done_var : Integer := 0; // authored name: done
        entry;
        then 'STATE-R-A';
        state 'STATE-R-A';
        @EngineeringElement about 'STATE-R-A' {
            sourceId = "STATE-R-A";
            conceptKind = "state_usage";
        }
        state 'STATE-R-B';
        @EngineeringElement about 'STATE-R-B' {
            sourceId = "STATE-R-B";
            conceptKind = "state_usage";
        }
        state 'STATE-R-C';
        @EngineeringElement about 'STATE-R-C' {
            sourceId = "STATE-R-C";
            conceptKind = "state_usage";
        }
        state 'STATE-R-D';
        @EngineeringElement about 'STATE-R-D' {
            sourceId = "STATE-R-D";
            conceptKind = "state_usage";
        }
        succession 'REL-7B46D1590AF201E1' first 'STATE-R-C' then done;
        @EngineeringRelationship about 'REL-7B46D1590AF201E1' {
            sourceId = "REL-7B46D1590AF201E1";
            conceptKind = "succession";
            relationshipKind = "machine_final";
        }
        succession 'REL-EEEB7C935A52D15E' first 'STATE-R-D' then done;
        @EngineeringRelationship about 'REL-EEEB7C935A52D15E' {
            sourceId = "REL-EEEB7C935A52D15E";
            conceptKind = "succession";
            relationshipKind = "machine_final";
        }
        transition 'TR-R-001'
            first 'STATE-R-A'
            accept 'on-EVT-A' :> 'EVT-A'
            if count < 3 and not flag
            do action {
                // simultaneous assignment: every right-hand side reads the pre-transition values
                assign count := count + 1;
                assign flag := true;
                assign label := "go";
            }
            then 'STATE-R-B';
        @EngineeringRelationship about 'TR-R-001' {
            sourceId = "TR-R-001";
            conceptKind = "transition";
            relationshipKind = "machine_transition";
        }
        transition 'TR-R-002'
            first 'STATE-R-B'
            accept 'on-EVT-B' :> 'EVT-B'
            if flag or count >= 2 implies label == "go"
            do action {
                // simultaneous assignment: every right-hand side reads the pre-transition values
                assign 'state' := -count * 2;
                assign done_var := 'state' - (-count);
            }
            then 'STATE-R-A';
        @EngineeringRelationship about 'TR-R-002' {
            sourceId = "TR-R-002";
            conceptKind = "transition";
            relationshipKind = "machine_transition";
        }
        transition 'TR-R-003'
            first 'STATE-R-B'
            if count != 3 and count > 0 and count <= 5
            then 'STATE-R-C';
        @EngineeringRelationship about 'TR-R-003' {
            sourceId = "TR-R-003";
            conceptKind = "transition";
            relationshipKind = "machine_transition";
        }
        transition 'TR-R-004'
            first 'STATE-R-B'
            // guard of transition TR-R-004 is not expressible in SysML v2: in_state(...) has no SysML v2 expression: SysML has no state-test operator
            //     authored: in_state(STATE-R-A)
            // effect of transition TR-R-004 is not expressible in SysML v2: in_state(...) has no SysML v2 expression: SysML has no state-test operator
            //     authored: flag := in_state(STATE-R-B)
            then 'STATE-R-D' {
                doc /* Needs a state test */
            }
        @EngineeringRelationship about 'TR-R-004' {
            sourceId = "TR-R-004";
            conceptKind = "transition";
            relationshipKind = "machine_transition";
        }
        transition 'TR-R-005'
            first 'STATE-R-A'
            then 'STATE-R-D';
        @EngineeringRelationship about 'TR-R-005' {
            sourceId = "TR-R-005";
            conceptKind = "transition";
            relationshipKind = "machine_transition";
        }
        constraint 'INV-R-001' {
            doc /* Protects REQ-A. */
            count >= 0 implies flag == true or label != "x"
        }
        @EngineeringElement about 'INV-R-001' {
            sourceId = "INV-R-001";
            conceptKind = "constraint_usage";
        }
        // invariant INV-R-002 is not expressible in SysML v2: in_state(...) has no SysML v2 expression: SysML has no state-test operator
        //     authored: in_state(STATE-R-C) implies flag
        constraint 'INV-R-003' {
            doc /* Protects REQ-A. */
            not (count < 0) and (count + 1) * 2 - 1 >= done_var
        }
        @EngineeringElement about 'INV-R-003' {
            sourceId = "INV-R-003";
            conceptKind = "constraint_usage";
        }
    }
}
