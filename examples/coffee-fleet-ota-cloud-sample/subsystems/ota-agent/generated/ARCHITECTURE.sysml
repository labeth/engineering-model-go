// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'COF-OTA-AGENT' {
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
        sourceId = "COF-OTA-AGENT";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "CAP-OTA-APPLY";
        conceptKind = "port_usage";
    }
    port 'CAP-OTA-APPLY' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-463F2ABBD4ADA012";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-463F2ABBD4ADA012' : EngineeringConcern {
        attribute 'narrative' : 'String' = "OTA behavior is traced from the subsystem requirements to verification and apply units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-7FCB8EE6EE4E6782";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-7FCB8EE6EE4E6782' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Firmware signatures are verified at the device boundary before any apply.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-831ED2CB09F1622E";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-831ED2CB09F1622E' : EngineeringConcern {
        attribute 'narrative' : 'String' = "The OTA agent verifies firmware trust before applying updates and rolls back on failure.";
    }

    @EngineeringElement {
        sourceId = "CTRL-FIRMWARE-SIGNATURE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-FIRMWARE-SIGNATURE';

    @EngineeringElement {
        sourceId = "DATA-FIRMWARE-BUNDLE";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-FIRMWARE-BUNDLE';

    @EngineeringElement {
        sourceId = "DO-FIRMWARE-BUNDLE";
        conceptKind = "item_definition";
    }
    item def 'DO-FIRMWARE-BUNDLE' {
        attribute 'classification' : 'String' = "confidential";
    }

    @EngineeringElement {
        sourceId = "EVT-CAMPAIGN-SCHEDULED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-CAMPAIGN-SCHEDULED';

    @EngineeringElement {
        sourceId = "EVT-VALIDATION-FAILED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-VALIDATION-FAILED';

    @EngineeringElement {
        sourceId = "EXTENSION-37E773CFEE430F8B";
        conceptKind = "engineering_extension";
    }
    attribute 'EXTENSION-37E773CFEE430F8B';

    @EngineeringElement {
        sourceId = "FEAT-OTA-CAMPAIGN";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-OTA-CAMPAIGN';

    @EngineeringElement {
        sourceId = "FG-OTA";
        conceptKind = "part_definition";
    }
    part def 'FG-OTA';

    @EngineeringElement {
        sourceId = "FU-OTA-APPLY";
        conceptKind = "part_usage";
    }
    part 'FU-OTA-APPLY' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-OTA-VERIFY";
        conceptKind = "part_usage";
    }
    part 'FU-OTA-VERIFY' : EngineeringPart;

    @EngineeringElement {
        sourceId = "IF-OTA-APPLY";
        conceptKind = "interface_definition";
    }
    interface def 'IF-OTA-APPLY' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "internal";
        attribute 'endpoint' : 'String' = "/ota/apply";
    }

    @EngineeringElement {
        sourceId = "NEED-CELLULAR-LINK";
        conceptKind = "port_usage";
    }
    port 'NEED-CELLULAR-LINK' : ~'EngineeringPort' {
        in port 'required_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "NEED-TELEMETRY-FEED";
        conceptKind = "port_usage";
    }
    port 'NEED-TELEMETRY-FEED' : ~'EngineeringPort' {
        in port 'required_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "REQ-OTA-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-OTA-001' {
        attribute 'text' : 'String' = "Where ota campaign is enabled, when ota campaign scheduled event is received, the ota agent subsystem shall apply the signed firmware bundle to the machine.";
        attribute 'notes' : 'String' = "Firmware apply behavior; implemented here.";
    }

    @EngineeringElement {
        sourceId = "REQ-OTA-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-OTA-002' {
        attribute 'text' : 'String' = "Where ota campaign is enabled, when ota campaign scheduled event is received, the ota agent subsystem shall verify the firmware signature before apply.";
        attribute 'notes' : 'String' = "Firmware integrity gate.";
    }

    @EngineeringElement {
        sourceId = "REQ-OTA-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-OTA-003' {
        attribute 'text' : 'String' = "When firmware validation failed event is received, the ota agent subsystem shall rollback the firmware apply.";
        attribute 'notes' : 'String' = "Rollback on validation failure.";
    }

    @EngineeringElement {
        sourceId = "SYS-OTA-AGENT";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-OTA-AGENT';

    @EngineeringElement {
        sourceId = "VER-FIRMWARE-SIGNATURE";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-FIRMWARE-SIGNATURE' : EngineeringVerification;

    @EngineeringRelationship {
        sourceId = "REL-B3A885152D1FC5D7";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-B3A885152D1FC5D7' : EngineeringVerification {
        subject 'VER-FIRMWARE-SIGNATURE' : EngineeringItem;
        objective 'CTRL-FIRMWARE-SIGNATURE' : EngineeringRequirement;
    }
}
