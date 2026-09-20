// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'COF-TELEMETRY' {
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
        sourceId = "COF-TELEMETRY";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "CAP-TELEM-REPORT";
        conceptKind = "port_usage";
    }
    port 'CAP-TELEM-REPORT' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-58636EAE0589C8F2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-58636EAE0589C8F2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "The telemetry subsystem owns sampling cadence and signed reporting at the machine boundary.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-661258B6CC25CCB8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-661258B6CC25CCB8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Telemetry behavior is traced from the subsystem requirements to its functional units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-848B156B8FF873AE";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-848B156B8FF873AE' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Telemetry records are signed so the fleet can verify integrity on ingest.";
    }

    @EngineeringElement {
        sourceId = "CTRL-TELEM-INTEGRITY";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-TELEM-INTEGRITY';

    @EngineeringElement {
        sourceId = "DATA-TELEM-RECORD";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-TELEM-RECORD';

    @EngineeringElement {
        sourceId = "DO-TELEM-RECORD";
        conceptKind = "item_definition";
    }
    item def 'DO-TELEM-RECORD' {
        attribute 'classification' : 'String' = "internal";
    }

    @EngineeringElement {
        sourceId = "EVT-BREW-COMPLETED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-BREW-COMPLETED';

    @EngineeringElement {
        sourceId = "EVT-SAMPLE-DUE";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-SAMPLE-DUE';

    @EngineeringElement {
        sourceId = "EXTENSION-095302628C010BF0";
        conceptKind = "engineering_extension";
    }
    attribute 'EXTENSION-095302628C010BF0';

    @EngineeringElement {
        sourceId = "FEAT-TELEMETRY";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-TELEMETRY';

    @EngineeringElement {
        sourceId = "FG-TELEMETRY";
        conceptKind = "part_definition";
    }
    part def 'FG-TELEMETRY';

    @EngineeringElement {
        sourceId = "FU-TELEM-REPORT";
        conceptKind = "part_usage";
    }
    part 'FU-TELEM-REPORT' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-TELEM-SAMPLE";
        conceptKind = "part_usage";
    }
    part 'FU-TELEM-SAMPLE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "IF-TELEM-REPORT";
        conceptKind = "interface_definition";
    }
    interface def 'IF-TELEM-REPORT' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "/telemetry";
    }

    @EngineeringElement {
        sourceId = "NEED-CAN-BUS";
        conceptKind = "port_usage";
    }
    port 'NEED-CAN-BUS' : ~'EngineeringPort' {
        in port 'required_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "REQ-TELEM-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-TELEM-001' {
        attribute 'text' : 'String' = "Where telemetry collection is enabled, when brew cycle completed event is received, the telemetry subsystem shall report a telemetry record to the fleet.";
        attribute 'notes' : 'String' = "Telemetry reporting behavior; implemented here.";
    }

    @EngineeringElement {
        sourceId = "REQ-TELEM-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-TELEM-002' {
        attribute 'text' : 'String' = "Where telemetry collection is enabled, when sample interval elapsed event is received, the telemetry subsystem shall sample machine sensors.";
        attribute 'notes' : 'String' = "Sampling behavior that produces telemetry records.";
    }

    @EngineeringElement {
        sourceId = "REQ-TELEM-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-TELEM-003' {
        attribute 'text' : 'String' = "Where telemetry collection is enabled, when brew cycle completed event is received, the telemetry subsystem shall sign each telemetry record.";
        attribute 'notes' : 'String' = "Integrity control implementation.";
    }

    @EngineeringElement {
        sourceId = "SYS-TELEMETRY";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-TELEMETRY';

    @EngineeringElement {
        sourceId = "VER-TELEM-INTEGRITY";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-TELEM-INTEGRITY' : EngineeringVerification;

    @EngineeringRelationship {
        sourceId = "REL-D5B663C9C59C847E";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-D5B663C9C59C847E' : EngineeringVerification {
        subject 'VER-TELEM-INTEGRITY' : EngineeringItem;
        objective 'CTRL-TELEM-INTEGRITY' : EngineeringRequirement;
    }
}
