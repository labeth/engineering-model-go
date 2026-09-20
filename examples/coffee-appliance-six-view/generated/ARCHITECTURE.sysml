// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'coffee-appliance-six-view' {
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
        sourceId = "coffee-appliance-six-view";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-COFFEE-DRINKER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-COFFEE-DRINKER';

    @EngineeringElement {
        sourceId = "ADR-COFFEE-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-COFFEE-001';

    @EngineeringElement {
        sourceId = "CAP-BREW-BEVERAGE";
        conceptKind = "action_definition";
    }
    action def 'CAP-BREW-BEVERAGE';

    @EngineeringElement {
        sourceId = "CAP-CONTROL-TEMPERATURE";
        conceptKind = "action_definition";
    }
    action def 'CAP-CONTROL-TEMPERATURE';

    @EngineeringElement {
        sourceId = "CAP-MAKE-DRINK";
        conceptKind = "action_definition";
    }
    action def 'CAP-MAKE-DRINK';

    @EngineeringElement {
        sourceId = "CONN-PUMP-HEATER";
        conceptKind = "part_usage";
    }
    part 'CONN-PUMP-HEATER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "DATA-BREW-SELECTION";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-BREW-SELECTION';

    @EngineeringElement {
        sourceId = "DEP-COUNTERTOP-DEVICE";
        conceptKind = "part_definition";
    }
    part def 'DEP-COUNTERTOP-DEVICE';

    @EngineeringElement {
        sourceId = "EVT-BREW-REQUESTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-BREW-REQUESTED';

    @EngineeringElement {
        sourceId = "HW-APPLIANCE-CHASSIS";
        conceptKind = "part_definition";
    }
    part def 'HW-APPLIANCE-CHASSIS';

    @EngineeringElement {
        sourceId = "HW-HEATER";
        conceptKind = "part_usage";
    }
    part 'HW-HEATER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "HW-PUMP";
        conceptKind = "part_usage";
    }
    part 'HW-PUMP' : EngineeringPart;

    @EngineeringElement {
        sourceId = "HW-WATER-SYSTEM";
        conceptKind = "part_usage";
    }
    part 'HW-WATER-SYSTEM' : EngineeringPart;

    @EngineeringElement {
        sourceId = "ITEM-BREW-SELECTION";
        conceptKind = "item_definition";
    }
    item def 'ITEM-BREW-SELECTION';

    @EngineeringElement {
        sourceId = "ITEM-BREW-STATUS";
        conceptKind = "item_definition";
    }
    item def 'ITEM-BREW-STATUS';

    @EngineeringElement {
        sourceId = "ITEM-HOT-WATER";
        conceptKind = "item_definition";
    }
    item def 'ITEM-HOT-WATER';

    @EngineeringElement {
        sourceId = "ITEM-WATER";
        conceptKind = "item_definition";
    }
    item def 'ITEM-WATER';

    @EngineeringElement {
        sourceId = "PART-COFFEE-APPLIANCE";
        conceptKind = "part_definition";
    }
    part def 'PART-COFFEE-APPLIANCE';

    @EngineeringElement {
        sourceId = "PORT-HEATER-IN";
        conceptKind = "port_usage";
    }
    port 'PORT-HEATER-IN' : EngineeringPort;

    @EngineeringElement {
        sourceId = "PORT-HEATER-OUT";
        conceptKind = "port_usage";
    }
    port 'PORT-HEATER-OUT' : EngineeringPort;

    @EngineeringElement {
        sourceId = "PORT-PUMP-OUT";
        conceptKind = "port_usage";
    }
    port 'PORT-PUMP-OUT' : EngineeringPort;

    @EngineeringElement {
        sourceId = "QTY-BREW-TEMPERATURE";
        conceptKind = "quantity_definition";
    }
    attribute def 'QTY-BREW-TEMPERATURE';

    @EngineeringElement {
        sourceId = "REQ-COFFEE-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COFFEE-001' {
        attribute 'text' : 'String' = "While appliance is ready, when brew request is received, the coffee appliance shall prepare the brew selection.";
        attribute 'notes' : 'String' = "Minimal observable brew-cycle requirement for the example.";
    }

    @EngineeringElement {
        sourceId = "STATE-READY";
        conceptKind = "engineering_extension";
    }
    attribute 'STATE-READY';

    @EngineeringElement {
        sourceId = "SW-APPLIANCE-CONTROLLER";
        conceptKind = "part_definition";
    }
    part def 'SW-APPLIANCE-CONTROLLER' {
        attribute 'commandPort';
        attribute 'statusPort';
    }

    @EngineeringElement {
        sourceId = "SW-BREW-COORDINATOR";
        conceptKind = "part_usage";
    }
    part 'SW-BREW-COORDINATOR' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SW-TEMPERATURE-CONTROLLER";
        conceptKind = "part_usage";
    }
    part 'SW-TEMPERATURE-CONTROLLER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SYS-COFFEE-APPLIANCE";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-COFFEE-APPLIANCE';

    @EngineeringElement {
        sourceId = "UC-HEAT-WATER";
        conceptKind = "use_case_definition";
    }
    use case def 'UC-HEAT-WATER';

    @EngineeringElement {
        sourceId = "UC-PREPARE-COFFEE";
        conceptKind = "use_case_definition";
    }
    use case def 'UC-PREPARE-COFFEE';

    @EngineeringElement {
        sourceId = "UNIT-CELSIUS";
        conceptKind = "unit_definition";
    }
    attribute def 'UNIT-CELSIUS';

    @EngineeringElement {
        sourceId = "VIEW-COFFEE-DEPLOYMENT";
        conceptKind = "view_usage";
    }
    view 'VIEW-COFFEE-DEPLOYMENT' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-COFFEE-IMPLEMENTATION";
        conceptKind = "view_usage";
    }
    view 'VIEW-COFFEE-IMPLEMENTATION' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-COFFEE-LOGICAL";
        conceptKind = "view_usage";
    }
    view 'VIEW-COFFEE-LOGICAL' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-COFFEE-PHYSICAL";
        conceptKind = "view_usage";
    }
    view 'VIEW-COFFEE-PHYSICAL' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-COFFEE-PROCESS";
        conceptKind = "view_usage";
    }
    view 'VIEW-COFFEE-PROCESS' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-COFFEE-USE-CASE";
        conceptKind = "view_usage";
    }
    view 'VIEW-COFFEE-USE-CASE' : EngineeringView;

    @EngineeringRelationship {
        sourceId = "MSG-01-SELECTION";
        conceptKind = "transfer";
        relationshipKind = "submit selection";
    }
    part def 'MSG-01-SELECTION-context' {
        part 'ACT-COFFEE-DRINKER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'SW-BREW-COORDINATOR' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'MSG-01-SELECTION' of 'ITEM-BREW-SELECTION' from 'ACT-COFFEE-DRINKER'.pOut to 'SW-BREW-COORDINATOR'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "MSG-02-HEAT";
        conceptKind = "transfer";
        relationshipKind = "request heat";
    }
    part def 'MSG-02-HEAT-context' {
        part 'SW-BREW-COORDINATOR' : EngineeringPart { port pOut : EngineeringPort; }
        part 'SW-TEMPERATURE-CONTROLLER' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'MSG-02-HEAT' of 'QTY-BREW-TEMPERATURE' from 'SW-BREW-COORDINATOR'.pOut to 'SW-TEMPERATURE-CONTROLLER'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "MSG-03-CONTROL-LOOP";
        conceptKind = "transfer";
        relationshipKind = "regulate temperature";
    }
    part def 'MSG-03-CONTROL-LOOP-context' {
        part participant : EngineeringPart { port pOut : EngineeringPort; port pIn : EngineeringPort; }
        flow 'MSG-03-CONTROL-LOOP' of 'QTY-BREW-TEMPERATURE' from participant.pOut to participant.pIn;
    }

    @EngineeringRelationship {
        sourceId = "MSG-04-STATUS";
        conceptKind = "transfer";
        relationshipKind = "report ready";
    }
    part def 'MSG-04-STATUS-context' {
        part 'SW-BREW-COORDINATOR' : EngineeringPart { port pOut : EngineeringPort; }
        part 'ACT-COFFEE-DRINKER' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'MSG-04-STATUS' of 'ITEM-BREW-STATUS' from 'SW-BREW-COORDINATOR'.pOut to 'ACT-COFFEE-DRINKER'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-ACTOR-PREPARE";
        conceptKind = "dependency";
        relationshipKind = "participates";
    }
    dependency 'REL-ACTOR-PREPARE' from 'ACT-COFFEE-DRINKER' to 'UC-PREPARE-COFFEE';

    @EngineeringRelationship {
        sourceId = "REL-CAP-BREW-PARENT";
        conceptKind = "specialization";
        relationshipKind = "capability decomposition";
    }
    dependency 'REL-CAP-BREW-PARENT' from 'CAP-BREW-BEVERAGE' to 'CAP-MAKE-DRINK';

    @EngineeringRelationship {
        sourceId = "REL-CAP-TEMP-PARENT";
        conceptKind = "specialization";
        relationshipKind = "capability decomposition";
    }
    dependency 'REL-CAP-TEMP-PARENT' from 'CAP-CONTROL-TEMPERATURE' to 'CAP-BREW-BEVERAGE';

    @EngineeringRelationship {
        sourceId = "REL-HEAT-QUANTITY";
        conceptKind = "dependency";
        relationshipKind = "temperature target";
    }
    dependency 'REL-HEAT-QUANTITY' from 'HW-HEATER' to 'QTY-BREW-TEMPERATURE';

    @EngineeringRelationship {
        sourceId = "REL-PREPARE-INCLUDES-HEAT";
        conceptKind = "dependency";
        relationshipKind = "include";
    }
    dependency 'REL-PREPARE-INCLUDES-HEAT' from 'UC-PREPARE-COFFEE' to 'UC-HEAT-WATER';

    @EngineeringRelationship {
        sourceId = "REL-PUMP-HEATER";
        conceptKind = "dependency";
        relationshipKind = "carries water";
    }
    dependency 'REL-PUMP-HEATER' from 'PORT-PUMP-OUT' to 'PORT-HEATER-IN';

    @EngineeringRelationship {
        sourceId = "REL-SW-DEPLOYMENT";
        conceptKind = "allocation";
        relationshipKind = "deployed to";
    }
    part def 'REL-SW-DEPLOYMENT-context' {
        part 'SW-APPLIANCE-CONTROLLER' : EngineeringPart;
        part 'DEP-COUNTERTOP-DEVICE' : EngineeringPart;
        allocation 'REL-SW-DEPLOYMENT' allocate 'SW-APPLIANCE-CONTROLLER' to 'DEP-COUNTERTOP-DEVICE';
    }

    @EngineeringRelationship {
        sourceId = "REL-SW-HARDWARE";
        conceptKind = "allocation";
        relationshipKind = "allocated to";
    }
    part def 'REL-SW-HARDWARE-context' {
        part 'SW-APPLIANCE-CONTROLLER' : EngineeringPart;
        part 'HW-APPLIANCE-CHASSIS' : EngineeringPart;
        allocation 'REL-SW-HARDWARE' allocate 'SW-APPLIANCE-CONTROLLER' to 'HW-APPLIANCE-CHASSIS';
    }
}
