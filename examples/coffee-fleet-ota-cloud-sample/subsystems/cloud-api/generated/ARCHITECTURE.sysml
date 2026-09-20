// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'COF-CLOUD-API' {
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
        sourceId = "COF-CLOUD-API";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "CAP-CLOUD-INGEST";
        conceptKind = "port_usage";
    }
    port 'CAP-CLOUD-INGEST' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-797BF3240457DFD3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-797BF3240457DFD3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Cloud behavior is traced from the subsystem requirements to ingest and campaign units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-79A761A84AA1CC6A";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-79A761A84AA1CC6A' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Telemetry ingestion is authenticated to prevent spoofed machine data.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-7B417C19D67CE606";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-7B417C19D67CE606' : EngineeringConcern {
        attribute 'narrative' : 'String' = "The cloud API subsystem ingests authenticated telemetry and drives OTA campaigns.";
    }

    @EngineeringElement {
        sourceId = "CTRL-INGEST-AUTH";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-INGEST-AUTH';

    @EngineeringElement {
        sourceId = "DATA-FLEET-METRIC";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-FLEET-METRIC';

    @EngineeringElement {
        sourceId = "DO-FLEET-METRIC";
        conceptKind = "item_definition";
    }
    item def 'DO-FLEET-METRIC' {
        attribute 'classification' : 'String' = "internal";
    }

    @EngineeringElement {
        sourceId = "EVT-TELEMETRY-INGESTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-TELEMETRY-INGESTED';

    @EngineeringElement {
        sourceId = "EXTENSION-76484FFFA44D2A36";
        conceptKind = "engineering_extension";
    }
    attribute 'EXTENSION-76484FFFA44D2A36';

    @EngineeringElement {
        sourceId = "FEAT-CLOUD-INGEST";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-CLOUD-INGEST';

    @EngineeringElement {
        sourceId = "FG-CLOUD";
        conceptKind = "part_definition";
    }
    part def 'FG-CLOUD';

    @EngineeringElement {
        sourceId = "FU-CLOUD-CAMPAIGN";
        conceptKind = "part_usage";
    }
    part 'FU-CLOUD-CAMPAIGN' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-CLOUD-INGEST";
        conceptKind = "part_usage";
    }
    part 'FU-CLOUD-INGEST' : EngineeringPart;

    @EngineeringElement {
        sourceId = "IF-CLOUD-INGEST";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLOUD-INGEST' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "/ingest";
    }

    @EngineeringElement {
        sourceId = "NEED-INTERNET";
        conceptKind = "port_usage";
    }
    port 'NEED-INTERNET' : ~'EngineeringPort' {
        in port 'required_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "REQ-CLOUD-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-CLOUD-001' {
        attribute 'text' : 'String' = "Where cloud ingestion is enabled, when telemetry ingested event is received, the cloud api subsystem shall persist a fleet metric.";
        attribute 'notes' : 'String' = "Telemetry ingestion behavior; implemented here.";
    }

    @EngineeringElement {
        sourceId = "REQ-CLOUD-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-CLOUD-002' {
        attribute 'text' : 'String' = "Where cloud ingestion is enabled, when telemetry ingested event is received, the cloud api subsystem shall authenticate the reporting machine.";
        attribute 'notes' : 'String' = "Ingest authentication control.";
    }

    @EngineeringElement {
        sourceId = "SYS-CLOUD-API";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-CLOUD-API';

    @EngineeringElement {
        sourceId = "VER-INGEST-AUTH";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-INGEST-AUTH' : EngineeringVerification;

    @EngineeringRelationship {
        sourceId = "REL-0D77FB2BF40B6ACB";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-0D77FB2BF40B6ACB' : EngineeringVerification {
        subject 'VER-INGEST-AUTH' : EngineeringItem;
        objective 'CTRL-INGEST-AUTH' : EngineeringRequirement;
    }
}
