// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'atlas-industries-portfolio' {
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
        sourceId = "atlas-industries-portfolio";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-PORT-EXECUTIVE";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-PORT-EXECUTIVE';

    @EngineeringElement {
        sourceId = "ADR-PORT-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-PORT-001';

    @EngineeringElement {
        sourceId = "CAP-ATLAS-PORTFOLIO";
        conceptKind = "port_usage";
    }
    port 'CAP-ATLAS-PORTFOLIO' : 'EngineeringPort' {
        out port 'provided_EngineeringPort' : 'EngineeringPort';
    }

    @EngineeringElement {
        sourceId = "CONCERN-1F41E1507AAA87FE";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-1F41E1507AAA87FE' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Company requirements delegate to stable product contracts while products independently reuse shared modules.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F7520471227332DE";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F7520471227332DE' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Aegis and Orion contract capabilities satisfy separate company requirements.";
    }

    @EngineeringElement {
        sourceId = "DATA-PORT-STATUS";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-PORT-STATUS';

    @EngineeringElement {
        sourceId = "DO-PORT-STATUS";
        conceptKind = "item_definition";
    }
    item def 'DO-PORT-STATUS' {
        attribute 'classification' : 'String' = "business-operations";
    }

    @EngineeringElement {
        sourceId = "FG-PORT";
        conceptKind = "part_definition";
    }
    part def 'FG-PORT';

    @EngineeringElement {
        sourceId = "FU-PORT-GOVERNANCE";
        conceptKind = "part_usage";
    }
    part 'FU-PORT-GOVERNANCE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "REQ-PORT-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PORT-001' {
        attribute 'text' : 'String' = "The atlas industries portfolio shall report portfolio status to the portfolio executive.";
    }

    @EngineeringElement {
        sourceId = "REQ-PORT-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PORT-002' {
        attribute 'text' : 'String' = "The atlas industries portfolio shall provide perimeter awareness using the Aegis Sentinel product capability.";
    }

    @EngineeringElement {
        sourceId = "REQ-PORT-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PORT-003' {
        attribute 'text' : 'String' = "The atlas industries portfolio shall provide resilient communications using the Orion Relay product capability.";
    }

    @EngineeringElement {
        sourceId = "SUB-PORT-AEGIS";
        conceptKind = "part_usage";
    }
    part 'SUB-PORT-AEGIS' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-PORT-ORION";
        conceptKind = "part_usage";
    }
    part 'SUB-PORT-ORION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SYS-PORT";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-PORT';

    part 'SUB-PORT-AEGIS/aegis::CAP-AEGIS-MISSION' : EngineeringPart;

    part 'SUB-PORT-ORION/orion::CAP-ORION-MISSION' : EngineeringPart;

    @EngineeringRelationship {
        sourceId = "REL-74019EA66960E842";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-74019EA66960E842-context' {
        part 'REQ-PORT-002' : EngineeringPart;
        part 'SUB-PORT-AEGIS/aegis::CAP-AEGIS-MISSION' : EngineeringPart;
        allocation 'REL-74019EA66960E842' allocate 'REQ-PORT-002' to 'SUB-PORT-AEGIS/aegis::CAP-AEGIS-MISSION';
    }

    @EngineeringRelationship {
        sourceId = "REL-EA55A527ED8FFDBA";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-EA55A527ED8FFDBA-context' {
        part 'REQ-PORT-003' : EngineeringPart;
        part 'SUB-PORT-ORION/orion::CAP-ORION-MISSION' : EngineeringPart;
        allocation 'REL-EA55A527ED8FFDBA' allocate 'REQ-PORT-003' to 'SUB-PORT-ORION/orion::CAP-ORION-MISSION';
    }

    // Publication public from models.atlas.example/products/aegis-sentinel@v0@v0.1.0 (atlas-aegis-sentinel).
    package 'aegis' {
        part def 'ACT-AEG-ENGINEER';
        part def 'ACT-AEG-OPERATOR';
        part def 'CAP-AEGIS-MISSION';
        part def 'DO-AEG-TRACK';
        part def 'FG-AEG';
        part def 'FU-AEG-MISSION';
        part def 'FU-AEG-SENSOR-FUSION';
        part def 'HW-AEG-OPTICAL';
        part def 'HW-AEG-RADAR';
        port def 'HWIF-AEG-SENSOR-BUS';
        port def 'IF-AEG-TRACK';
        requirement def 'REQ-AEG-001';
        requirement def 'REQ-AEG-002';
        requirement def 'REQ-AEG-003';
        requirement def 'REQ-AEG-004';
        requirement def 'REQ-AEG-005';
    }

    // Publication public from models.atlas.example/products/orion-relay@v0@v0.1.0 (atlas-orion-relay).
    package 'orion' {
        part def 'ACT-ORI-CONTROLLER';
        part def 'ACT-ORI-ENGINEER';
        part def 'CAP-ORION-MISSION';
        part def 'DO-ORI-MESSAGE';
        part def 'FG-ORI';
        part def 'FU-ORI-LINK';
        part def 'FU-ORI-ROUTER';
        part def 'HW-ORI-ANTENNA';
        part def 'HW-ORI-RADIO';
        port def 'HWIF-ORI-RF-CONTROL';
        port def 'IF-ORI-MESSAGE';
        requirement def 'REQ-ORI-001';
        requirement def 'REQ-ORI-002';
        requirement def 'REQ-ORI-003';
        requirement def 'REQ-ORI-004';
        requirement def 'REQ-ORI-005';
    }
}
