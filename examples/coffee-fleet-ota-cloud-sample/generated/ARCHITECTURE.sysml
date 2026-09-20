// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'sample-coffee-fleet-ota-cloud-model' {
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
        sourceId = "sample-coffee-fleet-ota-cloud-model";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-BARISTA";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-BARISTA';

    @EngineeringElement {
        sourceId = "ACT-FLEET-OPERATOR";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-FLEET-OPERATOR';

    @EngineeringElement {
        sourceId = "ACT-PLATFORM-OPERATOR";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-PLATFORM-OPERATOR';

    @EngineeringElement {
        sourceId = "ACT-SECURITY-ANALYST";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-SECURITY-ANALYST';

    @EngineeringElement {
        sourceId = "ADR-COF-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-COF-001';

    @EngineeringElement {
        sourceId = "ASM-COFFEE-DEVICE-ROOT-OF-TRUST";
        conceptKind = "engineering_extension";
    }
    attribute 'ASM-COFFEE-DEVICE-ROOT-OF-TRUST';

    @EngineeringElement {
        sourceId = "AV-LOG-TAMPERING";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-LOG-TAMPERING';

    @EngineeringElement {
        sourceId = "AV-MALICIOUS-FIRMWARE-BUNDLE";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-MALICIOUS-FIRMWARE-BUNDLE';

    @EngineeringElement {
        sourceId = "AV-SPOOFED-DEVICE-IDENTITY";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-SPOOFED-DEVICE-IDENTITY';

    @EngineeringElement {
        sourceId = "AV-TELEMETRY-REPLAY-ABUSE";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-TELEMETRY-REPLAY-ABUSE';

    @EngineeringElement {
        sourceId = "CONCERN-074A3EB00B5374A6";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-074A3EB00B5374A6' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces signed-firmware policy and campaign authorization boundaries.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-086571BFD59C95A4";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-086571BFD59C95A4' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Sends telemetry over authenticated channels and retries during endpoint outage windows.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-16BC3D515D34D88D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-16BC3D515D34D88D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform reliability requirements map to operational code paths and checks.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-186EEE58B9FFFC8B";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-186EEE58B9FFFC8B' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Provides identity validation and secure config interfaces to edge and cloud units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-18852C914F5636C0";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-18852C914F5636C0' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Machine Edge enforces firmware trust and authenticated telemetry at the device boundary.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-1D863A8915D22AAB";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-1D863A8915D22AAB' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Manages device identity, trust roots, and secret/config distribution.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2DD6ADCDB4E17542";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2DD6ADCDB4E17542' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Coordinates deployment/runtime controls that shape fleet service availability.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-31A9A3DE2A45F3C6";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-31A9A3DE2A45F3C6' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Cloud Control owns centralized ingestion, OTA campaign planning, and fleet-level reporting behavior.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-36F783B8CEE6F9FE";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-36F783B8CEE6F9FE' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces signature validation and rejects untrusted firmware bundles.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-371CAE8F44776BE1";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-371CAE8F44776BE1' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Logging and abuse requirements link to reporting modules and test evidence.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-376F5D5C27083680";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-376F5D5C27083680' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Accepts machine telemetry and forwards normalized records to observability reporting.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-3E86CD990ECBEA0B";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-3E86CD990ECBEA0B' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces operational hardening and security escalation signaling.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-3F2DFF3B3B3A75D7";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-3F2DFF3B3B3A75D7' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Deployed as reporting plane with independent scaling and retention controls.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-430AE6EE0C1827DD";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-430AE6EE0C1827DD' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Machine Edge defines stable device-local responsibilities for telemetry capture and OTA execution.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-482A15010D5289E9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-482A15010D5289E9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Governs runtime deployment and reliability operations for central cloud services.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4916771D1BC6EED2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4916771D1BC6EED2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "OTA and rollback requirements are linked to update-agent code and verification evidence.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4CFC8C0D8FC764C4";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4CFC8C0D8FC764C4' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Ingestion requirements map to trace markers and verification artifacts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-51AD25B4BCD22224";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-51AD25B4BCD22224' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Machine Edge sends telemetry upstream, receives OTA commands, and reports apply/rollback status.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-57C873A9A2241C39";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-57C873A9A2241C39' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Aggregates telemetry/update outcomes into operational and audit records.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5DF698678F9B3F31";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5DF698678F9B3F31' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Deployed as independent cloud ingestion boundary to isolate scaling concerns.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5E920AD2580FDE0A";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5E920AD2580FDE0A' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Applies identity checks and replay throttling for ingest requests.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-60A1591DD9BEDA62";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-60A1591DD9BEDA62' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform operations hardens identity, key rotation, and log integrity boundaries.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-65C52A8C4EC75BB2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-65C52A8C4EC75BB2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Runtime and identity evidence is linked back to platform-owned requirements and controls.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6CA375629BE22765";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6CA375629BE22765' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Requirement mappings align cloud control behavior with inferred runtime/code evidence.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6DF05FD4C167316D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6DF05FD4C167316D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Machine-edge behavior is traced from telemetry and OTA requirements to code and verification.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-7F5F6E86525C3513";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-7F5F6E86525C3513' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Cloud Control coordinates telemetry intake, campaign commands, and observability event fanout.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-83C835D136A3D7FC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-83C835D136A3D7FC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Protects telemetry authenticity and defends against replay abuse patterns.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-83D8D1CAE6FA36F9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-83D8D1CAE6FA36F9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform Operations owns runtime reliability and identity/secret controls for device-cloud trust.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-840550537121B5C3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-840550537121B5C3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Applies signed firmware updates and manages safe rollback behavior.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-9CB840DB14A6B906";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-9CB840DB14A6B906' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Identity and audit requirements trace to secrets/config handling code and tests.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A008810074824EB2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A008810074824EB2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Ingests telemetry/status and emits validated fleet events.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-ADE87C45E4D107C3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-ADE87C45E4D107C3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Owns source-to-runtime release controls and rollback guardrails.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-AEFF9A02EA8CA19D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-AEFF9A02EA8CA19D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Isolated deployment supports safe OTA policy evolution.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-AF855D562DA733EC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-AF855D562DA733EC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Deployed as edge software component with independent release cadence.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B3058E33939664DA";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B3058E33939664DA' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform operations governs release sequencing, runtime placement, and operational guardrails.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BB6201AEF1E45E2C";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BB6201AEF1E45E2C' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Managed as shared control-plane boundary with controlled key rotation.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BC9FB884D5CEA77A";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BC9FB884D5CEA77A' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Coordinates fleet OTA rollout cohorts and rollback decisions.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BE91FD688CAF5F21";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BE91FD688CAF5F21' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Protects credential lifecycle and enforces least-privilege access patterns.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C347897561E3EFDB";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C347897561E3EFDB' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Machine-edge software artifacts are rolled independently from cloud control to isolate device risk.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C356B434277DC569";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C356B434277DC569' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Cloud Control focuses on replay detection, campaign authorization, and audit integrity.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-D056522609BF5F1A";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-D056522609BF5F1A' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Cloud-control units deploy independently to reduce blast radius during campaign logic changes.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-D98276A7344BC931";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-D98276A7344BC931' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform operations coordinates runtime controls and secret distribution contracts with dependent units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-DDB1DF1F3878A1EE";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-DDB1DF1F3878A1EE' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Receives campaign commands and reports update/rollback status back to cloud control.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E6CEF4EC75A5E148";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E6CEF4EC75A5E148' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Campaign requirements connect to orchestration code and verification checks.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-EE911A10A66BB683";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-EE911A10A66BB683' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Versioned and rolled independently from telemetry pipeline components.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-EF9B6F7F7573E305";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-EF9B6F7F7573E305' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Sends OTA commands to update agents and ingests status feedback for campaign control.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F31D99F8B6543753";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F31D99F8B6543753' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Captures machine telemetry and publishes cloud-bound telemetry records.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F9702A3E6D569DCA";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F9702A3E6D569DCA' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Telemetry and resilience requirements map to trace-marked ingestion code paths.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-FA198518E67154C6";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-FA198518E67154C6' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Consumes telemetry and OTA outcomes and emits dashboards/alerts/log events.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-FBA5D4318E3672A5";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-FBA5D4318E3672A5' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Preserves audit integrity and protects sensitive identifiers in persisted records.";
    }

    @EngineeringElement {
        sourceId = "COND-FIRMWARE-SIGNATURE-VALID";
        conceptKind = "engineering_extension";
    }
    attribute 'COND-FIRMWARE-SIGNATURE-VALID';

    @EngineeringElement {
        sourceId = "CTRL-COFFEE-DEVICE-IDENTITY";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COFFEE-DEVICE-IDENTITY';

    @EngineeringElement {
        sourceId = "CTRL-COFFEE-FIRMWARE-SIGNATURE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-COFFEE-FIRMWARE-SIGNATURE';

    @EngineeringElement {
        sourceId = "CV-COFFEE-AUDIT-INTEGRITY-ANALYSIS";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COFFEE-AUDIT-INTEGRITY-ANALYSIS' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-COFFEE-DEVICE-IDENTITY-TEST";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COFFEE-DEVICE-IDENTITY-TEST' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-COFFEE-FW-SIGNATURE-TEST";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COFFEE-FW-SIGNATURE-TEST' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-COFFEE-TELEMETRY-REPLAY";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-COFFEE-TELEMETRY-REPLAY' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "DATA-AUDIT-LOG-RECORD";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-AUDIT-LOG-RECORD';

    @EngineeringElement {
        sourceId = "DATA-BREW-TELEMETRY";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-BREW-TELEMETRY';

    @EngineeringElement {
        sourceId = "DATA-CAMPAIGN-ID";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-CAMPAIGN-ID';

    @EngineeringElement {
        sourceId = "DATA-FIRMWARE-BUNDLE";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-FIRMWARE-BUNDLE';

    @EngineeringElement {
        sourceId = "DATA-MACHINE-IDENTITY";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-MACHINE-IDENTITY';

    @EngineeringElement {
        sourceId = "DATA-OTA-CAMPAIGN-POLICY";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-OTA-CAMPAIGN-POLICY';

    @EngineeringElement {
        sourceId = "DATA-OTA-STATUS";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-OTA-STATUS';

    @EngineeringElement {
        sourceId = "DEP-COFFEE-CLOUD-PROD";
        conceptKind = "part_definition";
    }
    part def 'DEP-COFFEE-CLOUD-PROD' {
        attribute 'environment' : 'String' = "prod";
        attribute 'region' : 'String' = "us-east-1";
        attribute 'namespace' : 'String' = "coffee";
        attribute 'trustZone' : 'String' = "app";
    }

    @EngineeringElement {
        sourceId = "DEP-COFFEE-EDGE-FLEET";
        conceptKind = "part_definition";
    }
    part def 'DEP-COFFEE-EDGE-FLEET' {
        attribute 'environment' : 'String' = "edge";
        attribute 'region' : 'String' = "global";
        attribute 'namespace' : 'String' = "devices";
        attribute 'trustZone' : 'String' = "edge";
    }

    @EngineeringElement {
        sourceId = "DO-COFFEE-OTA-PLAN";
        conceptKind = "item_definition";
    }
    item def 'DO-COFFEE-OTA-PLAN' {
        attribute 'schemaRef' : 'String' = "schemas/ota-plan.json";
        attribute 'classification' : 'String' = "deployment-control";
    }

    @EngineeringElement {
        sourceId = "DO-COFFEE-TELEMETRY-EVENT";
        conceptKind = "item_definition";
    }
    item def 'DO-COFFEE-TELEMETRY-EVENT' {
        attribute 'schemaRef' : 'String' = "schemas/telemetry-event.json";
        attribute 'classification' : 'String' = "operational-metrics";
    }

    @EngineeringElement {
        sourceId = "EVIDENCE-1FDC11AAE5C94C97";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-1FDC11AAE5C94C97';

    @EngineeringElement {
        sourceId = "EVIDENCE-21A0A8A4AC7A81F0";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-21A0A8A4AC7A81F0';

    @EngineeringElement {
        sourceId = "EVIDENCE-280A9F64CEF255C5";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-280A9F64CEF255C5';

    @EngineeringElement {
        sourceId = "EVIDENCE-2D5BFB65EB239AD5";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-2D5BFB65EB239AD5';

    @EngineeringElement {
        sourceId = "EVIDENCE-3C480A04A5468140";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-3C480A04A5468140';

    @EngineeringElement {
        sourceId = "EVIDENCE-42CE9FA4C34E1D3E";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-42CE9FA4C34E1D3E';

    @EngineeringElement {
        sourceId = "EVIDENCE-4E851042020EDDDE";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-4E851042020EDDDE';

    @EngineeringElement {
        sourceId = "EVIDENCE-74F6CA569305D4DB";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-74F6CA569305D4DB';

    @EngineeringElement {
        sourceId = "EVIDENCE-8575E3799BA42D60";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-8575E3799BA42D60';

    @EngineeringElement {
        sourceId = "EVIDENCE-8659649680DD2903";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-8659649680DD2903';

    @EngineeringElement {
        sourceId = "EVIDENCE-8E6991E82B0E72CB";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-8E6991E82B0E72CB';

    @EngineeringElement {
        sourceId = "EVIDENCE-9A1ECC1A8CB428AC";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-9A1ECC1A8CB428AC';

    @EngineeringElement {
        sourceId = "EVIDENCE-A6DA6FB89D5F6196";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-A6DA6FB89D5F6196';

    @EngineeringElement {
        sourceId = "EVIDENCE-AC20CF89D3FE9D7B";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-AC20CF89D3FE9D7B';

    @EngineeringElement {
        sourceId = "EVIDENCE-AE38BC663F841B2A";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-AE38BC663F841B2A';

    @EngineeringElement {
        sourceId = "EVIDENCE-B677B805DC1B7938";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-B677B805DC1B7938';

    @EngineeringElement {
        sourceId = "EVIDENCE-BCF68661601362A8";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-BCF68661601362A8';

    @EngineeringElement {
        sourceId = "EVIDENCE-BDB62AE5C6668521";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-BDB62AE5C6668521';

    @EngineeringElement {
        sourceId = "EVIDENCE-E3F4A48F736F168F";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-E3F4A48F736F168F';

    @EngineeringElement {
        sourceId = "EVIDENCE-F831A921856D8E0D";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-F831A921856D8E0D';

    @EngineeringElement {
        sourceId = "EVIDENCE-FC10C557385DCA72";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-FC10C557385DCA72';

    @EngineeringElement {
        sourceId = "EVT-BREW-CYCLE-COMPLETED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-BREW-CYCLE-COMPLETED';

    @EngineeringElement {
        sourceId = "EVT-CLOUD-INGEST-ENDPOINT-UNAVAILABLE";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-CLOUD-INGEST-ENDPOINT-UNAVAILABLE';

    @EngineeringElement {
        sourceId = "EVT-COFFEE-OTA-COMPLETED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-COFFEE-OTA-COMPLETED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-COFFEE-OTA-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-COFFEE-OTA-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-FIRMWARE-VALIDATION-FAILED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-FIRMWARE-VALIDATION-FAILED';

    @EngineeringElement {
        sourceId = "EVT-OTA-CAMPAIGN-SCHEDULED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-OTA-CAMPAIGN-SCHEDULED';

    @EngineeringElement {
        sourceId = "EVT-OTA-STATUS-REPORTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-OTA-STATUS-REPORTED';

    @EngineeringElement {
        sourceId = "EVT-TELEMETRY-INGESTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-TELEMETRY-INGESTED';

    @EngineeringElement {
        sourceId = "EVT-TELEMETRY-REPLAY-ABUSE-DETECTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-TELEMETRY-REPLAY-ABUSE-DETECTED';

    @EngineeringElement {
        sourceId = "EXTENSION-E1A950D022F15F02";
        conceptKind = "engineering_extension";
    }
    attribute 'EXTENSION-E1A950D022F15F02';

    @EngineeringElement {
        sourceId = "FEAT-AUDIT-LOGGING";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-AUDIT-LOGGING';

    @EngineeringElement {
        sourceId = "FEAT-OTA-CAMPAIGN";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-OTA-CAMPAIGN';

    @EngineeringElement {
        sourceId = "FG-CLOUD-CONTROL";
        conceptKind = "part_definition";
    }
    part def 'FG-CLOUD-CONTROL';

    @EngineeringElement {
        sourceId = "FG-MACHINE-EDGE";
        conceptKind = "part_definition";
    }
    part def 'FG-MACHINE-EDGE';

    @EngineeringElement {
        sourceId = "FG-PLATFORM-OPERATIONS";
        conceptKind = "part_definition";
    }
    part def 'FG-PLATFORM-OPERATIONS';

    @EngineeringElement {
        sourceId = "FLOW-COFFEE-OTA-ROLLOUT";
        conceptKind = "action_definition";
    }
    action def 'FLOW-COFFEE-OTA-ROLLOUT' {
        inout item 'item_DO_COFFEE_OTA_PLAN' : 'DO-COFFEE-OTA-PLAN';
    }

    @EngineeringElement {
        sourceId = "FLOW-COFFEE-TELEMETRY-INGEST";
        conceptKind = "action_definition";
    }
    action def 'FLOW-COFFEE-TELEMETRY-INGEST' {
        inout item 'item_DO_COFFEE_TELEMETRY_EVENT' : 'DO-COFFEE-TELEMETRY-EVENT';
    }

    @EngineeringElement {
        sourceId = "FU-CLOUD-RUNTIME-OPERATIONS";
        conceptKind = "part_usage";
    }
    part 'FU-CLOUD-RUNTIME-OPERATIONS' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-DEVICE-IDENTITY-SECRETS";
        conceptKind = "part_usage";
    }
    part 'FU-DEVICE-IDENTITY-SECRETS' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-FLEET-INGESTION-API";
        conceptKind = "part_usage";
    }
    part 'FU-FLEET-INGESTION-API' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-FLEET-OBSERVABILITY-REPORTING";
        conceptKind = "part_usage";
    }
    part 'FU-FLEET-OBSERVABILITY-REPORTING' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-MACHINE-TELEMETRY-COLLECTION";
        conceptKind = "part_usage";
    }
    part 'FU-MACHINE-TELEMETRY-COLLECTION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-OTA-UPDATE-AGENT";
        conceptKind = "part_usage";
    }
    part 'FU-OTA-UPDATE-AGENT' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-UPDATE-CAMPAIGN-ORCHESTRATION";
        conceptKind = "part_usage";
    }
    part 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "HW-CLOUD";
        conceptKind = "part_definition";
    }
    part def 'HW-CLOUD' {
        attribute 'kind' : 'String' = "cloud";
    }

    @EngineeringElement {
        sourceId = "HW-COFFEE-MACHINE";
        conceptKind = "part_definition";
    }
    part def 'HW-COFFEE-MACHINE' {
        attribute 'kind' : 'String' = "lru";
        attribute 'partNumber' : 'String' = "CM-700";
        attribute 'supplier' : 'String' = "BrewCo";
    }

    @EngineeringElement {
        sourceId = "HW-EDGE-GATEWAY";
        conceptKind = "part_definition";
    }
    part def 'HW-EDGE-GATEWAY' {
        attribute 'kind' : 'String' = "gateway";
    }

    @EngineeringElement {
        sourceId = "HW-FLOW-SENSOR";
        conceptKind = "part_definition";
    }
    part def 'HW-FLOW-SENSOR' {
        attribute 'kind' : 'String' = "sensor";
    }

    @EngineeringElement {
        sourceId = "HWIF-CAN";
        conceptKind = "connection_definition";
    }
    connection def 'HWIF-CAN' {
        end item 'source' : EngineeringItem;
        end item 'target' : EngineeringItem;
        attribute 'busType' : 'String' = "CAN";
        attribute 'direction' : 'String' = "in";
    }

    @EngineeringElement {
        sourceId = "HWIF-CELLULAR";
        conceptKind = "connection_definition";
    }
    connection def 'HWIF-CELLULAR' {
        end item 'source' : EngineeringItem;
        end item 'target' : EngineeringItem;
        attribute 'busType' : 'String' = "cellular";
        attribute 'direction' : 'String' = "out";
    }

    @EngineeringElement {
        sourceId = "HWIF-INTERNET";
        conceptKind = "connection_definition";
    }
    connection def 'HWIF-INTERNET' {
        end item 'source' : EngineeringItem;
        end item 'target' : EngineeringItem;
        attribute 'busType' : 'String' = "ethernet";
        attribute 'direction' : 'String' = "bidirectional";
    }

    @EngineeringElement {
        sourceId = "IF-COFFEE-OTA-COMMAND";
        conceptKind = "interface_definition";
    }
    interface def 'IF-COFFEE-OTA-COMMAND' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "mqtt";
        attribute 'endpoint' : 'String' = "mqtt://devices/ota/commands";
        attribute 'schemaRef' : 'String' = "schemas/ota-command.json";
    }

    @EngineeringElement {
        sourceId = "IF-COFFEE-TELEMETRY-INGEST";
        conceptKind = "interface_definition";
    }
    interface def 'IF-COFFEE-TELEMETRY-INGEST' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "mqtt";
        attribute 'endpoint' : 'String' = "mqtt://ingest/topics/telemetry";
        attribute 'schemaRef' : 'String' = "schemas/telemetry.json";
    }

    @EngineeringElement {
        sourceId = "MAP-COFFEE-DEVICE-IDENTITY";
        conceptKind = "engineering_extension";
    }
    attribute 'MAP-COFFEE-DEVICE-IDENTITY';

    @EngineeringElement {
        sourceId = "MAP-COFFEE-FIRMWARE-SIGNATURE";
        conceptKind = "engineering_extension";
    }
    attribute 'MAP-COFFEE-FIRMWARE-SIGNATURE';

    @EngineeringElement {
        sourceId = "MIT-COFFEE-AUDIT-IMMUTABILITY";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-COFFEE-AUDIT-IMMUTABILITY';

    @EngineeringElement {
        sourceId = "MIT-COFFEE-DEVICE-IDENTITY-ENFORCEMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-COFFEE-DEVICE-IDENTITY-ENFORCEMENT';

    @EngineeringElement {
        sourceId = "MIT-COFFEE-FW-SIGNATURE-VERIFY";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-COFFEE-FW-SIGNATURE-VERIFY';

    @EngineeringElement {
        sourceId = "MIT-COFFEE-TELEMETRY-REPLAY-WINDOW";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-COFFEE-TELEMETRY-REPLAY-WINDOW';

    @EngineeringElement {
        sourceId = "MODE-NORMAL";
        conceptKind = "engineering_extension";
    }
    attribute 'MODE-NORMAL';

    @EngineeringElement {
        sourceId = "OOS-COFFEE-HARDWARE-PHYSICAL-TAMPER";
        conceptKind = "engineering_extension";
    }
    attribute 'OOS-COFFEE-HARDWARE-PHYSICAL-TAMPER';

    @EngineeringElement {
        sourceId = "POAM-COFFEE-REPLAY-WINDOW-HARDENING";
        conceptKind = "engineering_extension";
    }
    attribute 'POAM-COFFEE-REPLAY-WINDOW-HARDENING';

    @EngineeringElement {
        sourceId = "PROF-NIST-800-53-MODERATE";
        conceptKind = "engineering_extension";
    }
    attribute 'PROF-NIST-800-53-MODERATE';

    @EngineeringElement {
        sourceId = "REF-CLOUD-LOGGING-SERVICE";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-CLOUD-LOGGING-SERVICE';

    @EngineeringElement {
        sourceId = "REF-CLOUD-RUNTIME-SDK";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-CLOUD-RUNTIME-SDK';

    @EngineeringElement {
        sourceId = "REF-FIRMWARE-BUNDLE-STORE";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-FIRMWARE-BUNDLE-STORE';

    @EngineeringElement {
        sourceId = "REF-FIRMWARE-SIGNER-SERVICE";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-FIRMWARE-SIGNER-SERVICE';

    @EngineeringElement {
        sourceId = "REF-IOT-INGEST-ENDPOINT";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-IOT-INGEST-ENDPOINT';

    @EngineeringElement {
        sourceId = "REF-MQTT-DEVICE-SDK";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-MQTT-DEVICE-SDK';

    @EngineeringElement {
        sourceId = "REQ-COF-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-001' {
        attribute 'text' : 'String' = "While coffee machine fleet operation is active, when brew cycle completed event is received, the coffee fleet system shall publish brew telemetry to fleet ingestion api for barista operations.";
        attribute 'notes' : 'String' = "Primary machine-to-cloud telemetry contract; delegated to the telemetry subsystem.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-002' {
        attribute 'text' : 'String' = "While coffee machine fleet operation is active, when telemetry ingested event is received, the coffee fleet system shall persist fleet telemetry metrics for fleet operator.";
        attribute 'notes' : 'String' = "Operational observability requirement; delegated to the cloud-api subsystem.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-003' {
        attribute 'text' : 'String' = "Where ota campaign is enabled, when ota campaign scheduled event is received, the coffee fleet system shall distribute signed firmware bundle to targeted machine cohort using ota campaign policy.";
        attribute 'notes' : 'String' = "OTA rollout behavior from central cloud control; delegated to the ota-agent subsystem.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-004";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-004' {
        attribute 'text' : 'String' = "Where ota campaign is enabled, when ota campaign scheduled event is received, the coffee fleet system shall gate firmware apply on firmware signature is valid.";
        attribute 'notes' : 'String' = "Firmware integrity and trust boundary control.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-005";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-005' {
        attribute 'text' : 'String' = "If firmware validation failed event is received, then the coffee fleet system shall rollback firmware apply.";
        attribute 'notes' : 'String' = "Rollback and failure signaling path; implemented here.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-006";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-006' {
        attribute 'text' : 'String' = "If cloud ingest endpoint is unavailable, then the coffee fleet system shall queue telemetry for retry.";
        attribute 'notes' : 'String' = "Resilience for temporary cloud outage.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-007";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-007' {
        attribute 'text' : 'String' = "Where audit logging is enabled, when ota status reported event is received, the coffee fleet system shall persist redacted audit log record with machine identity plus campaign id.";
        attribute 'notes' : 'String' = "Audit/compliance logging requirement.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-008";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-008' {
        attribute 'text' : 'String' = "If telemetry replay abuse is detected, then the coffee fleet system shall throttle duplicate telemetry.";
        attribute 'notes' : 'String' = "Abuse control and security response requirement.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-009";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-009' {
        attribute 'text' : 'String' = "While coffee machine fleet operation is active, when telemetry ingested event is received, the coffee fleet system shall persist logging records for fleet operator.";
        attribute 'notes' : 'String' = "Operational logging persistence requirement.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-010";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-010' {
        attribute 'text' : 'String' = "If firmware validation failed event is received, then the coffee fleet system shall report ota status to platform operator.";
        attribute 'notes' : 'String' = "OTA failure status signaling requirement.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-011";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-011' {
        attribute 'text' : 'String' = "If cloud ingest endpoint is unavailable, then the coffee fleet system shall flush queued telemetry when normal mode is active.";
        attribute 'notes' : 'String' = "Queued telemetry drain behavior after endpoint recovery.";
    }

    @EngineeringElement {
        sourceId = "REQ-COF-012";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-COF-012' {
        attribute 'text' : 'String' = "If telemetry replay abuse is detected, then the coffee fleet system shall notify security analyst.";
        attribute 'notes' : 'String' = "Security notification requirement for replay abuse events.";
    }

    @EngineeringElement {
        sourceId = "RISK-COFFEE-DEVICE-SPOOFING";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-COFFEE-DEVICE-SPOOFING';

    @EngineeringElement {
        sourceId = "RISK-COFFEE-LOG-TAMPERING";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-COFFEE-LOG-TAMPERING';

    @EngineeringElement {
        sourceId = "RISK-COFFEE-MALICIOUS-OTA";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-COFFEE-MALICIOUS-OTA';

    @EngineeringElement {
        sourceId = "RISK-COFFEE-TELEMETRY-REPLAY";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-COFFEE-TELEMETRY-REPLAY';

    @EngineeringElement {
        sourceId = "STATE-COFFEE-OTA-APPLIED";
        conceptKind = "state_definition";
    }
    state def 'STATE-COFFEE-OTA-APPLIED';

    @EngineeringElement {
        sourceId = "STATE-COFFEE-OTA-SCHEDULED";
        conceptKind = "state_definition";
    }
    state def 'STATE-COFFEE-OTA-SCHEDULED';

    @EngineeringElement {
        sourceId = "STATE-FLEET-OPERATION-ACTIVE";
        conceptKind = "engineering_extension";
    }
    attribute 'STATE-FLEET-OPERATION-ACTIVE';

    @EngineeringElement {
        sourceId = "SUB-CLOUD-API";
        conceptKind = "part_usage";
    }
    part 'SUB-CLOUD-API' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-OTA-AGENT";
        conceptKind = "part_usage";
    }
    part 'SUB-OTA-AGENT' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SUB-TELEMETRY";
        conceptKind = "part_usage";
    }
    part 'SUB-TELEMETRY' : EngineeringPart;

    @EngineeringElement {
        sourceId = "SYS-COFFEE-FLEET-SYSTEM";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-COFFEE-FLEET-SYSTEM';

    @EngineeringElement {
        sourceId = "TB-COFFEE-CLOUD-CONTROL";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-COFFEE-CLOUD-CONTROL';

    @EngineeringElement {
        sourceId = "TB-COFFEE-EDGE-DEVICE";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-COFFEE-EDGE-DEVICE';

    @EngineeringElement {
        sourceId = "TS-COFFEE-LOG-TAMPERING";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-COFFEE-LOG-TAMPERING';

    @EngineeringElement {
        sourceId = "TS-COFFEE-MALICIOUS-OTA-BUNDLE";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-COFFEE-MALICIOUS-OTA-BUNDLE';

    @EngineeringElement {
        sourceId = "TS-COFFEE-OTA-COMMAND-SPOOF";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-COFFEE-OTA-COMMAND-SPOOF';

    @EngineeringElement {
        sourceId = "TS-COFFEE-TELEMETRY-REPLAY";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-COFFEE-TELEMETRY-REPLAY';

    @EngineeringElement {
        sourceId = "VIEW-ARCHITECTURE-INTENT";
        conceptKind = "view_usage";
    }
    view 'VIEW-ARCHITECTURE-INTENT' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-COMMUNICATION";
        conceptKind = "view_usage";
    }
    view 'VIEW-COMMUNICATION' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-DEPLOYMENT";
        conceptKind = "view_usage";
    }
    view 'VIEW-DEPLOYMENT' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-INTERACTION-FLOW";
        conceptKind = "view_usage";
    }
    view 'VIEW-INTERACTION-FLOW' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-SECURITY";
        conceptKind = "view_usage";
    }
    view 'VIEW-SECURITY' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-STATE-LIFECYCLE";
        conceptKind = "view_usage";
    }
    view 'VIEW-STATE-LIFECYCLE' : EngineeringView;

    @EngineeringElement {
        sourceId = "VIEW-TRACEABILITY";
        conceptKind = "view_usage";
    }
    view 'VIEW-TRACEABILITY' : EngineeringView;

    @EngineeringElement {
        sourceId = "capture-machine-telemetry";
        conceptKind = "action_usage";
    }
    action 'capture-machine-telemetry' : EngineeringAction;

    @EngineeringElement {
        sourceId = "collect-and-send-telemetry";
        conceptKind = "action_usage";
    }
    action 'collect-and-send-telemetry' : EngineeringAction {
        out item 'output_telemetry_event' : EngineeringItem;
        inout item 'item_DO_COFFEE_TELEMETRY_EVENT' : 'DO-COFFEE-TELEMETRY-EVENT';
    }

    @EngineeringElement {
        sourceId = "execute-update";
        conceptKind = "action_usage";
    }
    action 'execute-update' : EngineeringAction {
        in item 'input_ota_rollout_plan' : EngineeringItem;
        inout item 'item_DO_COFFEE_OTA_PLAN' : 'DO-COFFEE-OTA-PLAN';
    }

    @EngineeringElement {
        sourceId = "ingest-telemetry";
        conceptKind = "action_usage";
    }
    action 'ingest-telemetry' : EngineeringAction {
        in item 'input_telemetry_event' : EngineeringItem;
        inout item 'item_DO_COFFEE_TELEMETRY_EVENT' : 'DO-COFFEE-TELEMETRY-EVENT';
    }

    @EngineeringElement {
        sourceId = "operator-schedules-rollout";
        conceptKind = "action_usage";
    }
    action 'operator-schedules-rollout' : EngineeringAction;

    @EngineeringElement {
        sourceId = "plan-campaign";
        conceptKind = "action_usage";
    }
    action 'plan-campaign' : EngineeringAction {
        out item 'output_ota_rollout_plan' : EngineeringItem;
        inout item 'item_DO_COFFEE_OTA_PLAN' : 'DO-COFFEE-OTA-PLAN';
    }

    @EngineeringElement {
        sourceId = "publish-fleet-observability";
        conceptKind = "action_usage";
    }
    action 'publish-fleet-observability' : EngineeringAction {
        in item 'input_telemetry_event' : EngineeringItem;
        inout item 'item_DO_COFFEE_TELEMETRY_EVENT' : 'DO-COFFEE-TELEMETRY-EVENT';
    }

    @EngineeringElement {
        sourceId = "report-outcome";
        conceptKind = "action_usage";
    }
    action 'report-outcome' : EngineeringAction {
        inout item 'item_DO_COFFEE_OTA_PLAN' : 'DO-COFFEE-OTA-PLAN';
    }

    part 'SUB-CLOUD-API/cloud-api::CAP-CLOUD-INGEST' : EngineeringPart;

    part 'SUB-OTA-AGENT/ota-agent::CAP-OTA-APPLY' : EngineeringPart;

    part 'SUB-TELEMETRY/telemetry::CAP-TELEM-REPORT' : EngineeringPart;

    requirement 'cloud-api::NEED-INTERNET' : EngineeringRequirement;

    requirement 'ota-agent::NEED-CELLULAR-LINK' : EngineeringRequirement;

    requirement 'ota-agent::NEED-TELEMETRY-FEED' : EngineeringRequirement;

    part 'telemetry::CAP-TELEM-REPORT' : EngineeringPart;

    requirement 'telemetry::NEED-CAN-BUS' : EngineeringRequirement;

    @EngineeringRelationship {
        sourceId = "REL-03025667862EA0B7";
        conceptKind = "connection";
        relationshipKind = "hardware_interface";
    }
    part def 'REL-03025667862EA0B7-context' {
        part source : 'HW-EDGE-GATEWAY';
        part target : 'HW-CLOUD';
        connection 'REL-03025667862EA0B7' connect source to target;
    }

    @EngineeringRelationship {
        sourceId = "REL-034C704F9A10372E";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-034C704F9A10372E' from 'ACT-FLEET-OPERATOR' to 'FU-UPDATE-CAMPAIGN-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-03944292DE1C16B1";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-03944292DE1C16B1' : EngineeringVerification {
        subject 'CV-COFFEE-FW-SIGNATURE-TEST' : EngineeringItem;
        objective 'CTRL-COFFEE-FIRMWARE-SIGNATURE' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-0632D237122444D8";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-0632D237122444D8-context' : EngineeringState {
        state 'STATE-COFFEE-OTA-APPLIED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-0632D237122444D8' first 'STATE-COFFEE-OTA-APPLIED' accept 'EVT-COFFEE-OTA-COMPLETED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-0C8339650D08AE89";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-0C8339650D08AE89' from 'FU-FLEET-INGESTION-API' to 'REF-CLOUD-RUNTIME-SDK';

    @EngineeringRelationship {
        sourceId = "REL-10573F75D4F0328A";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-10573F75D4F0328A' from 'AV-LOG-TAMPERING' to 'FU-DEVICE-IDENTITY-SECRETS';

    @EngineeringRelationship {
        sourceId = "REL-11664BA7B7F6947B";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-11664BA7B7F6947B' from 'AV-MALICIOUS-FIRMWARE-BUNDLE' to 'CTRL-COFFEE-FIRMWARE-SIGNATURE';

    @EngineeringRelationship {
        sourceId = "REL-11D3DCCA56394945";
        conceptKind = "connection";
        relationshipKind = "hardware_interface";
    }
    part def 'REL-11D3DCCA56394945-context' {
        part source : 'HW-COFFEE-MACHINE';
        part target : 'HW-EDGE-GATEWAY';
        connection 'REL-11D3DCCA56394945' connect source to target;
    }

    @EngineeringRelationship {
        sourceId = "REL-13D79F89F1869A7A";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-13D79F89F1869A7A-context' {
        part 'FU-OTA-UPDATE-AGENT' : EngineeringPart;
        part 'DEP-COFFEE-EDGE-FLEET' : EngineeringPart;
        allocation 'REL-13D79F89F1869A7A' allocate 'FU-OTA-UPDATE-AGENT' to 'DEP-COFFEE-EDGE-FLEET';
    }

    @EngineeringRelationship {
        sourceId = "REL-150507F20A9D6809";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-150507F20A9D6809' from 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' to 'REF-FIRMWARE-SIGNER-SERVICE';

    @EngineeringRelationship {
        sourceId = "REL-18B3CA8A910E433F";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-18B3CA8A910E433F-context' {
        part 'IF-COFFEE-TELEMETRY-INGEST' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-FLEET-INGESTION-API' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-18B3CA8A910E433F' from 'IF-COFFEE-TELEMETRY-INGEST'.pOut to 'FU-FLEET-INGESTION-API'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-1D84FBFBCFD2C3EE";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-1D84FBFBCFD2C3EE-context' {
        part 'REQ-COF-001' : EngineeringPart;
        part 'SUB-TELEMETRY/telemetry::CAP-TELEM-REPORT' : EngineeringPart;
        allocation 'REL-1D84FBFBCFD2C3EE' allocate 'REQ-COF-001' to 'SUB-TELEMETRY/telemetry::CAP-TELEM-REPORT';
    }

    @EngineeringRelationship {
        sourceId = "REL-21DA2C76632381A2";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-21DA2C76632381A2' from 'FU-FLEET-INGESTION-API' to 'FU-DEVICE-IDENTITY-SECRETS';

    @EngineeringRelationship {
        sourceId = "REL-23EDAB03427D517B";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-23EDAB03427D517B' from 'AV-SPOOFED-DEVICE-IDENTITY' to 'FU-FLEET-INGESTION-API';

    @EngineeringRelationship {
        sourceId = "REL-270397D886EC80AB";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-270397D886EC80AB' from 'AV-LOG-TAMPERING' to 'FU-FLEET-OBSERVABILITY-REPORTING';

    @EngineeringRelationship {
        sourceId = "REL-275C51ECF5E1E065";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-275C51ECF5E1E065' from 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' to 'FU-DEVICE-IDENTITY-SECRETS';

    @EngineeringRelationship {
        sourceId = "REL-293D74A94A71C701";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-293D74A94A71C701' from 'AV-TELEMETRY-REPLAY-ABUSE' to 'FU-FLEET-OBSERVABILITY-REPORTING';

    @EngineeringRelationship {
        sourceId = "REL-2B93239071435232";
        conceptKind = "transition";
        relationshipKind = "transitions_to";
    }
    state 'REL-2B93239071435232-context' : EngineeringState {
        action 'STATE-COFFEE-OTA-SCHEDULED' : EngineeringAction;
        action 'STATE-COFFEE-OTA-APPLIED' : EngineeringAction;
        transition 'REL-2B93239071435232' first 'STATE-COFFEE-OTA-SCHEDULED' then 'STATE-COFFEE-OTA-APPLIED';
    }

    @EngineeringRelationship {
        sourceId = "REL-2DCD89BD5855C90E";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-2DCD89BD5855C90E' from 'FU-CLOUD-RUNTIME-OPERATIONS' to 'FU-DEVICE-IDENTITY-SECRETS';

    @EngineeringRelationship {
        sourceId = "REL-314DD6D1BC67B72B";
        conceptKind = "guard";
        relationshipKind = "guarded_by";
    }
    state 'REL-314DD6D1BC67B72B-context' : EngineeringState {
        action 'STATE-COFFEE-OTA-APPLIED' : EngineeringAction;
        action 'CTRL-COFFEE-FIRMWARE-SIGNATURE' : EngineeringAction;
        transition 'REL-314DD6D1BC67B72B' first 'STATE-COFFEE-OTA-APPLIED' if true then 'CTRL-COFFEE-FIRMWARE-SIGNATURE';
    }

    @EngineeringRelationship {
        sourceId = "REL-3156E46ACE93EC5C";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-3156E46ACE93EC5C-context' {
        part 'REQ-COF-003' : EngineeringPart;
        part 'SUB-OTA-AGENT/ota-agent::CAP-OTA-APPLY' : EngineeringPart;
        allocation 'REL-3156E46ACE93EC5C' allocate 'REQ-COF-003' to 'SUB-OTA-AGENT/ota-agent::CAP-OTA-APPLY';
    }

    @EngineeringRelationship {
        sourceId = "REL-3266E5452C28C359";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-3266E5452C28C359' first 'execute-update' then 'report-outcome';

    @EngineeringRelationship {
        sourceId = "REL-34CA68D3D2F8B798";
        conceptKind = "transfer";
        relationshipKind = "calls";
    }
    part def 'REL-34CA68D3D2F8B798-context' {
        part 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-COFFEE-OTA-COMMAND' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-34CA68D3D2F8B798' from 'FU-UPDATE-CAMPAIGN-ORCHESTRATION'.pOut to 'IF-COFFEE-OTA-COMMAND'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-3526DBF7B4DD0F25";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-3526DBF7B4DD0F25' : EngineeringVerification {
        subject 'CV-COFFEE-DEVICE-IDENTITY-TEST' : EngineeringItem;
        objective 'CTRL-COFFEE-DEVICE-IDENTITY' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-39E3521A07ECD778";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-39E3521A07ECD778-context' {
        part 'FU-MACHINE-TELEMETRY-COLLECTION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-FLEET-INGESTION-API' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-39E3521A07ECD778' from 'FU-MACHINE-TELEMETRY-COLLECTION'.pOut to 'FU-FLEET-INGESTION-API'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-3A35ACDB501D1468";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-3A35ACDB501D1468-context' : EngineeringState {
        state 'STATE-COFFEE-OTA-SCHEDULED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-3A35ACDB501D1468' first 'STATE-COFFEE-OTA-SCHEDULED' accept 'EVT-COFFEE-OTA-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-3D271BC52034D00D";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-3D271BC52034D00D' first 'execute-update' then 'report-outcome';

    @EngineeringRelationship {
        sourceId = "REL-3F1634437CF64B8A";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-3F1634437CF64B8A' from 'FU-OTA-UPDATE-AGENT' to 'FU-DEVICE-IDENTITY-SECRETS';

    @EngineeringRelationship {
        sourceId = "REL-430ADBF440BE0B26";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-430ADBF440BE0B26-context' {
        part 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' : EngineeringPart;
        part 'DEP-COFFEE-CLOUD-PROD' : EngineeringPart;
        allocation 'REL-430ADBF440BE0B26' allocate 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' to 'DEP-COFFEE-CLOUD-PROD';
    }

    @EngineeringRelationship {
        sourceId = "REL-43EB3122728E02B8";
        conceptKind = "satisfaction";
        relationshipKind = "satisfaction";
    }
    requirement def 'REL-43EB3122728E02B8-context' {
        part satisfyingPart : EngineeringPart;
        satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;
    }

    @EngineeringRelationship {
        sourceId = "REL-44AE590A23A3E1CE";
        conceptKind = "satisfaction";
        relationshipKind = "satisfaction";
    }
    requirement def 'REL-44AE590A23A3E1CE-context' {
        part satisfyingPart : EngineeringPart;
        satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;
    }

    @EngineeringRelationship {
        sourceId = "REL-44B1E2D2679ECCE2";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-44B1E2D2679ECCE2' from 'ACT-SECURITY-ANALYST' to 'FU-DEVICE-IDENTITY-SECRETS';

    @EngineeringRelationship {
        sourceId = "REL-44E8F6DEE6CF910A";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-44E8F6DEE6CF910A' first 'plan-campaign' then 'execute-update';

    @EngineeringRelationship {
        sourceId = "REL-466C9AF0EE77FB4B";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-466C9AF0EE77FB4B' from 'publish-fleet-observability' to 'FU-FLEET-OBSERVABILITY-REPORTING';

    @EngineeringRelationship {
        sourceId = "REL-46EF975A8C60AFA4";
        conceptKind = "allocation";
        relationshipKind = "hosts";
    }
    part def 'REL-46EF975A8C60AFA4-context' {
        part 'SUB-TELEMETRY' : EngineeringPart;
        part 'HW-COFFEE-MACHINE' : EngineeringPart;
        allocation 'REL-46EF975A8C60AFA4' allocate 'SUB-TELEMETRY' to 'HW-COFFEE-MACHINE';
    }

    @EngineeringRelationship {
        sourceId = "REL-4B0DC76F556211F0";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-4B0DC76F556211F0' from 'FU-FLEET-OBSERVABILITY-REPORTING' to 'FU-DEVICE-IDENTITY-SECRETS';

    @EngineeringRelationship {
        sourceId = "REL-4EA6246BAAAD7A0E";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-4EA6246BAAAD7A0E-context' {
        part 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-COFFEE-OTA-PLAN' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-4EA6246BAAAD7A0E' from 'FU-UPDATE-CAMPAIGN-ORCHESTRATION'.pOut to 'DO-COFFEE-OTA-PLAN'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-5DBF696D78D2922E";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-5DBF696D78D2922E' from 'AV-MALICIOUS-FIRMWARE-BUNDLE' to 'FU-UPDATE-CAMPAIGN-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-5FB3D4ECA0C4F797";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-5FB3D4ECA0C4F797' from 'capture-machine-telemetry' to 'ACT-BARISTA';

    @EngineeringRelationship {
        sourceId = "REL-62C38F45AC45AF3A";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-62C38F45AC45AF3A' from 'collect-and-send-telemetry' to 'FU-MACHINE-TELEMETRY-COLLECTION';

    @EngineeringRelationship {
        sourceId = "REL-6B08055436A4C291";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-6B08055436A4C291' from 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' to 'FU-FLEET-OBSERVABILITY-REPORTING';

    @EngineeringRelationship {
        sourceId = "REL-70EE7CC0742B4DB8";
        conceptKind = "satisfaction";
        relationshipKind = "satisfaction";
    }
    requirement def 'REL-70EE7CC0742B4DB8-context' {
        part satisfyingPart : EngineeringPart;
        satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;
    }

    @EngineeringRelationship {
        sourceId = "REL-720C5F8183BD103C";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-720C5F8183BD103C-context' {
        part 'FU-OTA-UPDATE-AGENT' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-FLEET-OBSERVABILITY-REPORTING' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-720C5F8183BD103C' from 'FU-OTA-UPDATE-AGENT'.pOut to 'FU-FLEET-OBSERVABILITY-REPORTING'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-787D96E336B5679E";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-787D96E336B5679E' from 'FU-FLEET-INGESTION-API' to 'FU-FLEET-OBSERVABILITY-REPORTING';

    @EngineeringRelationship {
        sourceId = "REL-799B40995B3410E1";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-799B40995B3410E1-context' {
        part 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-COFFEE-OTA-PLAN' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-799B40995B3410E1' from 'FU-UPDATE-CAMPAIGN-ORCHESTRATION'.pOut to 'DO-COFFEE-OTA-PLAN'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-7A9E6B29656C6D14";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-7A9E6B29656C6D14' first 'ingest-telemetry' then 'publish-fleet-observability';

    @EngineeringRelationship {
        sourceId = "REL-7E8DF99E34648B06";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-7E8DF99E34648B06' from 'AV-SPOOFED-DEVICE-IDENTITY' to 'CTRL-COFFEE-DEVICE-IDENTITY';

    @EngineeringRelationship {
        sourceId = "REL-809CBF7BB8765ACE";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-809CBF7BB8765ACE' from 'FU-MACHINE-TELEMETRY-COLLECTION' to 'FU-FLEET-INGESTION-API';

    @EngineeringRelationship {
        sourceId = "REL-8920A9F2E72512CC";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-8920A9F2E72512CC' from 'FU-MACHINE-TELEMETRY-COLLECTION' to 'REF-IOT-INGEST-ENDPOINT';

    @EngineeringRelationship {
        sourceId = "REL-8A730264F46E56B3";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-8A730264F46E56B3-context' {
        part 'ACT-BARISTA' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-MACHINE-TELEMETRY-COLLECTION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-8A730264F46E56B3' from 'ACT-BARISTA'.pOut to 'FU-MACHINE-TELEMETRY-COLLECTION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-8B5D1F31AF101438";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-8B5D1F31AF101438-context' {
        part 'ACT-FLEET-OPERATOR' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-8B5D1F31AF101438' from 'ACT-FLEET-OPERATOR'.pOut to 'FU-UPDATE-CAMPAIGN-ORCHESTRATION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-91F58002CA8FA3DA";
        conceptKind = "allocation";
        relationshipKind = "allocation";
    }
    part def 'REL-91F58002CA8FA3DA-context' {
        part 'REQ-COF-002' : EngineeringPart;
        part 'SUB-CLOUD-API/cloud-api::CAP-CLOUD-INGEST' : EngineeringPart;
        allocation 'REL-91F58002CA8FA3DA' allocate 'REQ-COF-002' to 'SUB-CLOUD-API/cloud-api::CAP-CLOUD-INGEST';
    }

    @EngineeringRelationship {
        sourceId = "REL-940A5227FC5ED298";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-940A5227FC5ED298-context' {
        part 'FU-MACHINE-TELEMETRY-COLLECTION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-COFFEE-TELEMETRY-INGEST' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-940A5227FC5ED298' from 'FU-MACHINE-TELEMETRY-COLLECTION'.pOut to 'IF-COFFEE-TELEMETRY-INGEST'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-A0B337F57F233750";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-A0B337F57F233750-context' {
        part 'FU-FLEET-INGESTION-API' : EngineeringPart;
        part 'DEP-COFFEE-CLOUD-PROD' : EngineeringPart;
        allocation 'REL-A0B337F57F233750' allocate 'FU-FLEET-INGESTION-API' to 'DEP-COFFEE-CLOUD-PROD';
    }

    @EngineeringRelationship {
        sourceId = "REL-A6B469003489F36F";
        conceptKind = "allocation";
        relationshipKind = "hosts";
    }
    part def 'REL-A6B469003489F36F-context' {
        part 'SUB-CLOUD-API' : EngineeringPart;
        part 'HW-CLOUD' : EngineeringPart;
        allocation 'REL-A6B469003489F36F' allocate 'SUB-CLOUD-API' to 'HW-CLOUD';
    }

    @EngineeringRelationship {
        sourceId = "REL-A8FAACFC12608481";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-A8FAACFC12608481' first 'operator-schedules-rollout' then 'plan-campaign';

    @EngineeringRelationship {
        sourceId = "REL-AC2505FA4BD979EB";
        conceptKind = "guard";
        relationshipKind = "guarded_by";
    }
    state 'REL-AC2505FA4BD979EB-context' : EngineeringState {
        action 'STATE-COFFEE-OTA-APPLIED' : EngineeringAction;
        action 'TB-COFFEE-EDGE-DEVICE' : EngineeringAction;
        transition 'REL-AC2505FA4BD979EB' first 'STATE-COFFEE-OTA-APPLIED' if true then 'TB-COFFEE-EDGE-DEVICE';
    }

    @EngineeringRelationship {
        sourceId = "REL-AE0CA1005560F539";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-AE0CA1005560F539' first 'capture-machine-telemetry' then 'collect-and-send-telemetry';

    @EngineeringRelationship {
        sourceId = "REL-B3F83074459013F9";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-B3F83074459013F9-context' {
        part 'FU-FLEET-INGESTION-API' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-FLEET-OBSERVABILITY-REPORTING' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-B3F83074459013F9' from 'FU-FLEET-INGESTION-API'.pOut to 'FU-FLEET-OBSERVABILITY-REPORTING'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-B57BA72EE828673A";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-B57BA72EE828673A' from 'FU-OTA-UPDATE-AGENT' to 'REF-FIRMWARE-BUNDLE-STORE';

    @EngineeringRelationship {
        sourceId = "REL-B5E4121728B8F818";
        conceptKind = "allocation";
        relationshipKind = "hosts";
    }
    part def 'REL-B5E4121728B8F818-context' {
        part 'SUB-OTA-AGENT' : EngineeringPart;
        part 'HW-EDGE-GATEWAY' : EngineeringPart;
        allocation 'REL-B5E4121728B8F818' allocate 'SUB-OTA-AGENT' to 'HW-EDGE-GATEWAY';
    }

    @EngineeringRelationship {
        sourceId = "REL-B821905CBAACDEDF";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-B821905CBAACDEDF' from 'FU-OTA-UPDATE-AGENT' to 'TB-COFFEE-EDGE-DEVICE';

    @EngineeringRelationship {
        sourceId = "REL-B97E291D7352E8F0";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-B97E291D7352E8F0' from 'FU-OTA-UPDATE-AGENT' to 'FU-UPDATE-CAMPAIGN-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-BB316C56ED6E7877";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-BB316C56ED6E7877' : EngineeringVerification {
        subject 'CV-COFFEE-TELEMETRY-REPLAY' : EngineeringItem;
        objective 'CTRL-COFFEE-DEVICE-IDENTITY' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-BC2146CF8E92547E";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-BC2146CF8E92547E' from 'FU-FLEET-OBSERVABILITY-REPORTING' to 'REF-CLOUD-LOGGING-SERVICE';

    @EngineeringRelationship {
        sourceId = "REL-C4006F33639B2A25";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-C4006F33639B2A25' from 'AV-MALICIOUS-FIRMWARE-BUNDLE' to 'FU-OTA-UPDATE-AGENT';

    @EngineeringRelationship {
        sourceId = "REL-C52C62AA1EB18CE6";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-C52C62AA1EB18CE6' from 'execute-update' to 'FU-OTA-UPDATE-AGENT';

    @EngineeringRelationship {
        sourceId = "REL-C5ED3C66BDC90670";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-C5ED3C66BDC90670' from 'FU-CLOUD-RUNTIME-OPERATIONS' to 'TB-COFFEE-CLOUD-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-CD8A63D628E311B7";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-CD8A63D628E311B7' from 'operator-schedules-rollout' to 'ACT-FLEET-OPERATOR';

    @EngineeringRelationship {
        sourceId = "REL-D7220E705C98A4C3";
        conceptKind = "satisfaction";
        relationshipKind = "satisfaction";
    }
    requirement def 'REL-D7220E705C98A4C3-context' {
        part satisfyingPart : EngineeringPart;
        satisfy requirement targetRequirement : EngineeringRequirement by satisfyingPart;
    }

    @EngineeringRelationship {
        sourceId = "REL-D93B440BCB8E2C3C";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-D93B440BCB8E2C3C' from 'ingest-telemetry' to 'FU-FLEET-INGESTION-API';

    @EngineeringRelationship {
        sourceId = "REL-DBCEC722B1545BB8";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-DBCEC722B1545BB8' from 'AV-SPOOFED-DEVICE-IDENTITY' to 'FU-MACHINE-TELEMETRY-COLLECTION';

    @EngineeringRelationship {
        sourceId = "REL-DF1E35DCECD95183";
        conceptKind = "connection";
        relationshipKind = "hardware_interface";
    }
    part def 'REL-DF1E35DCECD95183-context' {
        part source : 'HW-FLOW-SENSOR';
        part target : 'HW-COFFEE-MACHINE';
        connection 'REL-DF1E35DCECD95183' connect source to target;
    }

    @EngineeringRelationship {
        sourceId = "REL-DFB1BDC0507970AD";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-DFB1BDC0507970AD' from 'plan-campaign' to 'FU-UPDATE-CAMPAIGN-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-E23E41CC3F7D5825";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-E23E41CC3F7D5825' : EngineeringVerification {
        subject 'CV-COFFEE-AUDIT-INTEGRITY-ANALYSIS' : EngineeringItem;
        objective 'CTRL-COFFEE-DEVICE-IDENTITY' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-E431ECFC0473BB4F";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-E431ECFC0473BB4F' from 'FU-CLOUD-RUNTIME-OPERATIONS' to 'REF-CLOUD-LOGGING-SERVICE';

    @EngineeringRelationship {
        sourceId = "REL-E47DB0E7631E8AC9";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-E47DB0E7631E8AC9' from 'report-outcome' to 'FU-FLEET-OBSERVABILITY-REPORTING';

    @EngineeringRelationship {
        sourceId = "REL-E78BF97EA502C760";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-E78BF97EA502C760' first 'collect-and-send-telemetry' then 'ingest-telemetry';

    @EngineeringRelationship {
        sourceId = "REL-E8A047A5C3B16F96";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-E8A047A5C3B16F96-context' {
        part 'FU-MACHINE-TELEMETRY-COLLECTION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-COFFEE-TELEMETRY-EVENT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-E8A047A5C3B16F96' from 'FU-MACHINE-TELEMETRY-COLLECTION'.pOut to 'DO-COFFEE-TELEMETRY-EVENT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-E92F7EBAAE8CE5C9";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-E92F7EBAAE8CE5C9' from 'AV-TELEMETRY-REPLAY-ABUSE' to 'FU-FLEET-INGESTION-API';

    @EngineeringRelationship {
        sourceId = "REL-EAEB886CF19C18C7";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-EAEB886CF19C18C7-context' {
        part 'FU-OTA-UPDATE-AGENT' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-COFFEE-OTA-PLAN' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-EAEB886CF19C18C7' from 'FU-OTA-UPDATE-AGENT'.pOut to 'DO-COFFEE-OTA-PLAN'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-EC31A38E44D51B95";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-EC31A38E44D51B95' from 'ACT-BARISTA' to 'FU-MACHINE-TELEMETRY-COLLECTION';

    @EngineeringRelationship {
        sourceId = "REL-EFEB887705072F44";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-EFEB887705072F44-context' {
        part 'FU-UPDATE-CAMPAIGN-ORCHESTRATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-OTA-UPDATE-AGENT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-EFEB887705072F44' from 'FU-UPDATE-CAMPAIGN-ORCHESTRATION'.pOut to 'FU-OTA-UPDATE-AGENT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-F3A6BBF0164523FC";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-F3A6BBF0164523FC' from 'FU-MACHINE-TELEMETRY-COLLECTION' to 'FU-DEVICE-IDENTITY-SECRETS';

    @EngineeringRelationship {
        sourceId = "REL-F7EA289AC5D2C031";
        conceptKind = "transfer";
        relationshipKind = "calls";
    }
    part def 'REL-F7EA289AC5D2C031-context' {
        part 'FU-MACHINE-TELEMETRY-COLLECTION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-COFFEE-TELEMETRY-INGEST' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-F7EA289AC5D2C031' from 'FU-MACHINE-TELEMETRY-COLLECTION'.pOut to 'IF-COFFEE-TELEMETRY-INGEST'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-F829609CE3726298";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-F829609CE3726298' from 'ACT-PLATFORM-OPERATOR' to 'FU-CLOUD-RUNTIME-OPERATIONS';

    @EngineeringRelationship {
        sourceId = "REL-F9A4D44E7C653F8D";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-F9A4D44E7C653F8D-context' {
        part 'ACT-FLEET-OPERATOR' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-OTA-UPDATE-AGENT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-F9A4D44E7C653F8D' from 'ACT-FLEET-OPERATOR'.pOut to 'FU-OTA-UPDATE-AGENT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-FBB11C73716B2D9B";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-FBB11C73716B2D9B' from 'FU-MACHINE-TELEMETRY-COLLECTION' to 'REF-MQTT-DEVICE-SDK';

    // Publication public from models.example/coffee-fleet/cloud-api@v0@v0.1.0 (COF-CLOUD-API).
    package 'cloud-api' {
        part def 'CAP-CLOUD-INGEST';
        part def 'DO-FLEET-METRIC';
        part def 'FG-CLOUD';
        part def 'FU-CLOUD-CAMPAIGN';
        part def 'FU-CLOUD-INGEST';
        port def 'IF-CLOUD-INGEST';
        requirement def 'REQ-CLOUD-001';
        requirement def 'REQ-CLOUD-002';
    }

    // Publication public from models.example/coffee-fleet/ota-agent@v0@v0.1.0 (COF-OTA-AGENT).
    package 'ota-agent' {
        part def 'CAP-OTA-APPLY';
        part def 'DO-FIRMWARE-BUNDLE';
        part def 'FG-OTA';
        part def 'FU-OTA-APPLY';
        part def 'FU-OTA-VERIFY';
        port def 'IF-OTA-APPLY';
        requirement def 'REQ-OTA-001';
        requirement def 'REQ-OTA-002';
        requirement def 'REQ-OTA-003';
    }

    // Publication public from models.example/coffee-fleet/telemetry@v0@v0.1.0 (COF-TELEMETRY).
    package 'telemetry' {
        part def 'DO-TELEM-RECORD';
        part def 'FG-TELEMETRY';
        part def 'FU-TELEM-REPORT';
        part def 'FU-TELEM-SAMPLE';
        port def 'IF-TELEM-REPORT';
        requirement def 'REQ-TELEM-001';
        requirement def 'REQ-TELEM-002';
        requirement def 'REQ-TELEM-003';
    }
}
