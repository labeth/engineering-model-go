// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'engineering-model-go' {
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
        sourceId = "engineering-model-go";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-AI-AGENT";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-AI-AGENT';

    @EngineeringElement {
        sourceId = "ACT-ARCHITECTURE-AUTHOR";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-ARCHITECTURE-AUTHOR';

    @EngineeringElement {
        sourceId = "ACT-CI-PIPELINE";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-CI-PIPELINE';

    @EngineeringElement {
        sourceId = "ACT-COMPLIANCE-ENGINEER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-COMPLIANCE-ENGINEER';

    @EngineeringElement {
        sourceId = "ACT-IMPLEMENTATION-ENGINEER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-IMPLEMENTATION-ENGINEER';

    @EngineeringElement {
        sourceId = "ADR-EMG-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-001';

    @EngineeringElement {
        sourceId = "ADR-EMG-002";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-002';

    @EngineeringElement {
        sourceId = "ADR-EMG-003";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-003';

    @EngineeringElement {
        sourceId = "ADR-EMG-004";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-004';

    @EngineeringElement {
        sourceId = "ADR-EMG-005";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-005';

    @EngineeringElement {
        sourceId = "ADR-EMG-006";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-006';

    @EngineeringElement {
        sourceId = "ADR-EMG-007";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-007';

    @EngineeringElement {
        sourceId = "ADR-EMG-008";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-008';

    @EngineeringElement {
        sourceId = "ADR-EMG-009";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-009';

    @EngineeringElement {
        sourceId = "ADR-EMG-010";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-010';

    @EngineeringElement {
        sourceId = "ADR-EMG-011";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-011';

    @EngineeringElement {
        sourceId = "ADR-EMG-012";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-012';

    @EngineeringElement {
        sourceId = "ADR-EMG-013";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-013';

    @EngineeringElement {
        sourceId = "ADR-EMG-014";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-014';

    @EngineeringElement {
        sourceId = "ADR-EMG-015";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-015';

    @EngineeringElement {
        sourceId = "ADR-EMG-016";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-016';

    @EngineeringElement {
        sourceId = "ADR-EMG-017";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-017';

    @EngineeringElement {
        sourceId = "ADR-EMG-018";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-018';

    @EngineeringElement {
        sourceId = "ADR-EMG-019";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-019';

    @EngineeringElement {
        sourceId = "ADR-EMG-020";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-020';

    @EngineeringElement {
        sourceId = "ADR-EMG-021";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-021';

    @EngineeringElement {
        sourceId = "ADR-EMG-022";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-022';

    @EngineeringElement {
        sourceId = "ADR-EMG-023";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-023';

    @EngineeringElement {
        sourceId = "ADR-EMG-024";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-024';

    @EngineeringElement {
        sourceId = "ADR-EMG-025";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-025';

    @EngineeringElement {
        sourceId = "ADR-EMG-026";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-026';

    @EngineeringElement {
        sourceId = "ADR-EMG-027";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-027';

    @EngineeringElement {
        sourceId = "ADR-EMG-028";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-EMG-028';

    @EngineeringElement {
        sourceId = "AV-COMPOSED-MODEL-CONTENT";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-COMPOSED-MODEL-CONTENT';

    @EngineeringElement {
        sourceId = "AV-FALSE-ACCEPTANCE-CLAIM";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-FALSE-ACCEPTANCE-CLAIM';

    @EngineeringElement {
        sourceId = "AV-MALFORMED-MODEL-INPUT";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-MALFORMED-MODEL-INPUT';

    @EngineeringElement {
        sourceId = "AV-PATH-TRAVERSAL-IN-MCP";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-PATH-TRAVERSAL-IN-MCP';

    @EngineeringElement {
        sourceId = "AV-SCHEMA-SUPPLY-CHAIN-TAMPER";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-SCHEMA-SUPPLY-CHAIN-TAMPER';

    @EngineeringElement {
        sourceId = "AV-TRACEABILITY-GAP-DRIFT";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-TRACEABILITY-GAP-DRIFT';

    @EngineeringElement {
        sourceId = "AV-UNTRUSTED-CHECKOUT-CONTENT";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-UNTRUSTED-CHECKOUT-CONTENT';

    @EngineeringElement {
        sourceId = "CONCERN-01977FA76B0309BF";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-01977FA76B0309BF' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces stable requirement IDs and deterministic formatting for downstream verification.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0366C3C85D858975";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0366C3C85D858975' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Validation and Analysis ships as shared libraries consumed by CLIs and server workflows.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-04442EEA31C1C4DF";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-04442EEA31C1C4DF' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Carries control references and evidence links into SSP outputs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0506ACEF4E9ABB05";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0506ACEF4E9ABB05' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Publishes compliance-focused output for security and audit consumers.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-07DB81073C700137";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-07DB81073C700137' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Emits the TRACE-MATRIX.json (and CSV) linking each requirement to its code, verification, and delegated subsystem requirement.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-089E3AC120FE1893";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-089E3AC120FE1893' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Artifact Generation validates output schema boundaries before publishing external artifacts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-08AAD9C60465E84F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-08AAD9C60465E84F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traceability and Compliance components execute in export pipelines and are validated against external toolchains.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-08B9F386F9288C13";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-08B9F386F9288C13' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Coordinates generation and validation command workflows from repo entrypoints.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-09CD776C17FF5B81";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-09CD776C17FF5B81' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Infers runtime/code/verification evidence to connect authored intent with implementation reality.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0A77A81F6752123F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0A77A81F6752123F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Processes initialize/list/call JSON-RPC messages and returns structured payloads, including a compact machine-readable model authoring contract.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0DEBD211D2FC822D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0DEBD211D2FC822D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Executed via enggemara and exposed through the MCP gemara tools.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0EA69A2A4BE53F42";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0EA69A2A4BE53F42' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Preserves requirement and evidence identifiers in generated docs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-105387A0CFFCA782";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-105387A0CFFCA782' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces strict schemas, panic containment, and repo-root path constraints.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-13F2D5F9F84445F0";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-13F2D5F9F84445F0' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Runs as stdio server executable in local and automated agent environments.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-142669CFE2436DE8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-142669CFE2436DE8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "MCP Integration exposes model-aware APIs for AI agents with explicit tool contracts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-15D44AEB8E091287";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-15D44AEB8E091287' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Delivers filtered entities and mappings to documentation and exports.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-163CFB0978233072";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-163CFB0978233072' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Serves as canonical requirement export for trace-chain continuity.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-173C63DA82FA5829";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-173C63DA82FA5829' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Builds templated DSL artifacts validated by external Structurizr tooling.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-177A25E0D08C7D51";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-177A25E0D08C7D51' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Integrates projections and design narratives into publication templates.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-1B60BEFFAAF23E0C";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-1B60BEFFAAF23E0C' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Reports cycle, path-boundary, source-conflict, module-fetch, module-identity, missing-ref, clone-failed, and unsatisfied-require diagnostics for every subsystem reference.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-1B935DC75A64C74E";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-1B935DC75A64C74E' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Preserves architecture IDs across DSL elements for cross-tool alignment.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-20CB06FB1709DC43";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-20CB06FB1709DC43' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Artifact Generation runs in local and CI contexts with deterministic output expectations.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-235F316B72F3BD77";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-235F316B72F3BD77' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Protects export integrity through strict schema validation and deterministic mapping.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2411EACA9C708935";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2411EACA9C708935' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Exports requirements and model metadata for formal traceability processing.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2585C9B87C1B534C";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2585C9B87C1B534C' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Protects trace integrity by surfacing missing or inconsistent requirement coverage.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-25CBAE08A74BA41F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-25CBAE08A74BA41F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Preserves stable Engineering Model identifiers and emits executable coverage evidence for every mapped concept.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-26030AD817860C52";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-26030AD817860C52' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Applies validation gates before producing or publishing artifacts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-27A48F024367AA32";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-27A48F024367AA32' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Connects user/CI invocation context to loader, validator, and exporter units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-285B482F4C79A2D9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-285B482F4C79A2D9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Validation and Analysis exchanges diagnostics and inferred evidence with generation and MCP surfaces.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-28D29DABA7EFDD09";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-28D29DABA7EFDD09' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Model Authoring coordinates loading and command entrypoints so downstream generation and validation receive normalized inputs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2C5C3496BD19AD07";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2C5C3496BD19AD07' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Projects architecture graphs for view-specific rendering and analysis.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2CA42F12F38BAD1E";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2CA42F12F38BAD1E' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Converts TRLC-LINKS (requirement) and ENGMODEL-LINKS (model element) markers into stable evidence relationships, existence-checked and scoped to the owning model.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-33D4D7D52CB8AC91";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-33D4D7D52CB8AC91' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Resolves a parent model's downward subsystem references into a federated system-of-systems from local paths, external git repositories, or exact-version CUE modules in OCI registries.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-35D214BFC1C7AC51";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-35D214BFC1C7AC51' : EngineeringConcern {
        attribute 'narrative' : 'String' = "MCP Integration enforces strict argument validation and repository path boundaries.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-389BEC41BA15BDB7";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-389BEC41BA15BDB7' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Preserves stable ID fidelity from source YAML through runtime structures.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-38E7420746BF4151";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-38E7420746BF4151' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Maintains link continuity across architecture, security, and traceability projections.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-39168423392BE2D9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-39168423392BE2D9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Model Authoring enforces input sanity and reference consistency to reduce malformed model ingestion risk.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-3A494D6D4BA9FD2C";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-3A494D6D4BA9FD2C' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Accepts versioned stable-ID operations and returns deterministic summaries plus structured diagnostics.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-3D525685CE5C2E8F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-3D525685CE5C2E8F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Applies NAF 4.1 stakeholder, concern, and viewpoint metadata to canonical model views without creating duplicate capability, service, logical, or resource concepts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-409CC41D48626189";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-409CC41D48626189' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Runs through engnaf locally and in CI using the same YAML and CUE validated model as every other exporter.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-41A3E94A4F1CB69F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-41A3E94A4F1CB69F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Provides a typed mutation boundary for authored model documents without duplicating model validation semantics.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-43D291070FAB5A5D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-43D291070FAB5A5D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Rejects unsupported framework versions, unknown viewpoint codes, and dangling stakeholder, concern, actor, or model-view references.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4A2CC8A9FCAECB1D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4A2CC8A9FCAECB1D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Runs via engstruct in local and CI validation pipelines.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4A62FF7A0E6B239D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4A62FF7A0E6B239D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Shipped as command binaries under cmd for task-specific execution.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4CCD4C19DCAD213E";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4CCD4C19DCAD213E' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Rejects unknown, duplicate, and lossy mappings before publishing an artifact with a conformance claim.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4E12094062BF39C7";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4E12094062BF39C7' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Preserves canonical IDs from each referenced Engmod view in the generated NAF architecture product.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-574438AB9322928F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-574438AB9322928F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Produces requirement-to-test visibility and coverage diagnostics.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-57620ED86CC1CE30";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-57620ED86CC1CE30' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Validation and Analysis defends against ambiguous references and stale traces through strict checks.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-58F002725094037D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-58F002725094037D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Materializes parent-to-subsystem requirement delegation and the consolidated traceability matrix without modifying subsystem models.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-596F9BA5D7EAAE16";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-596F9BA5D7EAAE16' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Generates deterministic architecture products organized by official NAF 4.1 viewpoint codes.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5A37A41BC54764F3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5A37A41BC54764F3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces correctness of IDs, mappings, and cross-reference relationships.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5A4F3597567E4AC2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5A4F3597567E4AC2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Limits inference scope to trusted repo paths and evidence formats.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5C8CD05C84E29105";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5C8CD05C84E29105' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Converts modeled security content into Threat Dragon and Open OTM artifacts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5FC15735786048F9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5FC15735786048F9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Executed via engdragon with schema validation scripts in CI.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-636DA8CDB9D1B586";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-636DA8CDB9D1B586' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Resolves standard SysML and KerML concepts through one mapping registry and exposes engineering-specific concepts as typed extensions.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-662616CC084987A1";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-662616CC084987A1' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Exposes model-aware operations for AI agents using a stable tool contract.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-684CE8475E3742C3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-684CE8475E3742C3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Projects the canonical semantic model into SysML v2 without introducing a parallel authoring vocabulary.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6A30985DAE158443";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6A30985DAE158443' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Model Authoring maintains stable IDs that anchor requirement and ownership traces across generated artifacts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6AE5E47F9B603D4C";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6AE5E47F9B603D4C' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Runs through englobster and report scripts in validation workflows.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6C18BB27FE609A57";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6C18BB27FE609A57' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Artifact Generation turns modeled intent into publication and exchange outputs used by engineers and tools.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6F254BFA753130E5";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6F254BFA753130E5' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traceability and Compliance protects evidence integrity and detects drift in requirement-test links.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-76BDF5060659372C";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-76BDF5060659372C' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Run via engtrace or as part of engdoc; fails the build on dangling code trace links.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-7AF1FF009F5145C9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-7AF1FF009F5145C9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Provides canonical model structures to validators, generators, and MCP handlers.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-7D5202776F5C7153";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-7D5202776F5C7153' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Maps modeled controls, risks, and implementations into OSCAL system security plan, assessment results, and plan of action and milestones structures.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-852616E4E66A267B";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-852616E4E66A267B' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces workspace boundaries, exact semantic versions, one source kind per dependency, CUE module identity, and acyclic references.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8960978BD71D6573";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8960978BD71D6573' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Ensures views respect include/exclude boundaries and stable root scopes.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-897F2A86A4FD8145";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-897F2A86A4FD8145' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Shipped as reusable package APIs consumed by command entrypoints.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8DDD01FA1551F393";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8DDD01FA1551F393' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Artifact Generation coordinates projections, templates, and exporters while preserving stable output contracts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8F182F5747F720B7";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8F182F5747F720B7' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Preserves control, threat, and risk semantics during model-to-Gemara transformation.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8F667F63AC5D10AA";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8F667F63AC5D10AA' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Model Authoring is deployed as CLI and package boundaries that can evolve without disrupting validation and export engines.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8FCDFB755769AC64";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8FCDFB755769AC64' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Maintains deterministic invocation paths for reproducible artifact updates.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-911646C7D11C0EDE";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-911646C7D11C0EDE' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Rolls up requirement status as implemented, verified, delegated, or orphan, and flags untraceable delegations.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-92C3E940828C507B";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-92C3E940828C507B' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Preserves control and risk semantics during model-to-OSCAL transformation.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-93AC01AD1CCA06D2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-93AC01AD1CCA06D2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traceability and Compliance is the system-of-record path for requirement linkage quality.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-9AB2672A425E9570";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-9AB2672A425E9570' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Keeps tool results tied to stable architecture IDs for explainable agent actions.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-9CB6CCD1139E3E22";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-9CB6CCD1139E3E22' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Links each mutation behavior to requirements and preserves stable identifiers through the YAML AST merge.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A0BF897B95B83EB9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A0BF897B95B83EB9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Executed through engtrlc in CI and local assurance workflows.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A88BA99695796FFD";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A88BA99695796FFD' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Ensures requirement/entity links remain analyzable and deterministic.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A8F39111723DA45F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A8F39111723DA45F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Executes during generation and analysis workflows using repository-local sources.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A955E57961E22CAB";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A955E57961E22CAB' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Embedded as reusable generation pipeline functionality.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-AA8A3CDB4934D025";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-AA8A3CDB4934D025' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Integrates TRLC outputs and inferred verification links for report generation.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-AD1112DA7D685F02";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-AD1112DA7D685F02' : EngineeringConcern {
        attribute 'narrative' : 'String' = "MCP Integration processes JSON-RPC requests and returns structured, schema-versioned tool payloads.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-AEF4D1D117DB863E";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-AEF4D1D117DB863E' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Carries security model sections in deterministic, reviewable output format.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-AF1DBB54A931ACA2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-AF1DBB54A931ACA2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Emits diagnostics consumed by CLIs, CI, and MCP preflight logic.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B183907C1AD29438";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B183907C1AD29438' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Invoked through engdoc for local and CI document generation.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B25E6415F9F5FA07";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B25E6415F9F5FA07' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Validation and Analysis preserves model correctness and derives implementation evidence from repository artifacts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B3FBAB33E39F3F84";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B3FBAB33E39F3F84' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Renders the model into OpenSSF Gemara L1-L7 catalogs and logs validated by the Gemara SDK.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BA23612F92FBE512";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BA23612F92FBE512' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Renders authoritative human-facing architecture documents from modeled content.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BA3BE48B371B1491";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BA3BE48B371B1491' : EngineeringConcern {
        attribute 'narrative' : 'String' = "MCP Integration reports tool-specific IDs to keep agent actions tied to model entities.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BABE05ECF7C77362";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BABE05ECF7C77362' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Validation and Analysis links requirements to code/runtime/test evidence for impact-oriented planning.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BB3C73A52FE8E9B6";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BB3C73A52FE8E9B6' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Projects architecture and deployment metadata into Structurizr DSL.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C0A3497ECFCB4EA8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C0A3497ECFCB4EA8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Binds each subsystem's provides/requires contract to the parent's allocations and satisfactions without modifying the subsystem.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C1FB8834BB75ACA0";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C1FB8834BB75ACA0' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Rejects unknown fields, invalid cross-field constraints, invalid parsing states, and unresolved core model references with path-aware diagnostics.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C40A14C329790825";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C40A14C329790825' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Rejects unknown fields, conflicting operations, stale source content, and invalid candidates before replacement.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C76DDBEBD8D44F0C";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C76DDBEBD8D44F0C' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traceability and Compliance ensures requirement lineage and assurance artifacts remain verifiable.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C7A5328168086773";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C7A5328168086773' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Runs as core library checks in local and CI execution paths.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-CD2F9656A07F8403";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-CD2F9656A07F8403' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Model Authoring defines how authored architecture inputs remain stable and composable across the repository.\nIt establishes document contracts for catalog, architecture, requirements, and design files.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-D6F5D0DCC43177F8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-D6F5D0DCC43177F8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Artifact Generation carries stable IDs and evidence references into AI, docs, and exchange formats.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-D8D3E9CDBEC6C024";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-D8D3E9CDBEC6C024' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Carries control-to-threat mappings and verification evidence into the Gemara evaluation and audit logs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-DA6AAD7EC39AF1DC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-DA6AAD7EC39AF1DC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Git repositories and OCI modules are materialized in the gitignored .engmod cache; an engmod.work.yml file can replace module paths with sibling repositories for offline multi-repository development.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-DA84307D39553E15";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-DA84307D39553E15' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Ensures exported topology reflects intended trust boundaries and relationships.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-DCC1FC3B91A5FFDF";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-DCC1FC3B91A5FFDF' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Scopes code to the nearest enclosing model root so a parent never claims a child model's links.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-DD1576187CB999D8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-DD1576187CB999D8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Executed via engoscal as part of assurance pipelines.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E71E920E043549C4";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E71E920E043549C4' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Produces package and requirement artifacts consumed by LOBSTER and checks.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E72B950676ED1DF3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E72B950676ED1DF3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Blocks malformed or unsafe model transitions before downstream processing.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E7984E5DDD499C01";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E7984E5DDD499C01' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Validates schema-versioned canonical YAML inputs, resolves explicit companion-document references with legacy defaults, and assembles one normalized bundle.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E83FAB2A9660507F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E83FAB2A9660507F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Runs through the engchange CLI or reusable Go API in local developer workflows.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E92C203A57544F35";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E92C203A57544F35' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Supplies inferred ownership and trace candidates to documentation, MCP, and reporting flows.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-EA82BCF173D8B259";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-EA82BCF173D8B259' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Maps flows, boundaries, threats, and controls into external schema structures.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-EC1862D562E3A26A";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-EC1862D562E3A26A' : EngineeringConcern {
        attribute 'narrative' : 'String' = "MCP Integration runs as stdio server endpoints invoked by agent runtimes.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-EE18DCA67540A415";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-EE18DCA67540A415' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Publishes vector, principle, guidance, control, capability, threat, risk, evaluation, enforcement, and audit documents, with an optional OSCAL catalog and assessment-results bridge.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F5995E41D1C4E3B0";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F5995E41D1C4E3B0' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Converts requirement and test link evidence into LOBSTER activity traces.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F7295D5F1BEB3081";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F7295D5F1BEB3081' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Maintains stable threat and control IDs through export outputs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-FB84C601F1F7B7F9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-FB84C601F1F7B7F9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Runs through engsysml locally and in CI with deterministic textual and project interchange outputs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-FE65550D2E893910";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-FE65550D2E893910' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traceability and Compliance integrates requirement, test, and report surfaces through shared ID contracts.";
    }

    @EngineeringElement {
        sourceId = "COND-CATALOG-REFERENCES-RESOLVE";
        conceptKind = "engineering_extension";
    }
    attribute 'COND-CATALOG-REFERENCES-RESOLVE';

    @EngineeringElement {
        sourceId = "COND-EXTERNAL-SCHEMA-AVAILABLE";
        conceptKind = "engineering_extension";
    }
    attribute 'COND-EXTERNAL-SCHEMA-AVAILABLE';

    @EngineeringElement {
        sourceId = "COND-REQUIREMENT-TRACE-COMPLETE";
        conceptKind = "engineering_extension";
    }
    attribute 'COND-REQUIREMENT-TRACE-COMPLETE';

    @EngineeringElement {
        sourceId = "CTRL-ACCEPTANCE-RECORD-CHECK";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-ACCEPTANCE-RECORD-CHECK';

    @EngineeringElement {
        sourceId = "CTRL-ARTIFACT-FRESHNESS-GATE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-ARTIFACT-FRESHNESS-GATE';

    @EngineeringElement {
        sourceId = "CTRL-ARTIFACT-SCAN-BOUNDS";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-ARTIFACT-SCAN-BOUNDS';

    @EngineeringElement {
        sourceId = "CTRL-CHILD-USER-NONDISCLOSURE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-CHILD-USER-NONDISCLOSURE';

    @EngineeringElement {
        sourceId = "CTRL-FINGERPRINT-COVERAGE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-FINGERPRINT-COVERAGE';

    @EngineeringElement {
        sourceId = "CTRL-GIT-REF-VALIDATION";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-GIT-REF-VALIDATION';

    @EngineeringElement {
        sourceId = "CTRL-GO-WORK-INPUT-VALIDATION";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-GO-WORK-INPUT-VALIDATION';

    @EngineeringElement {
        sourceId = "CTRL-LINK-REVIEW-RECORD";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-LINK-REVIEW-RECORD';

    @EngineeringElement {
        sourceId = "CTRL-MCP-PATH-BOUNDARY";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-MCP-PATH-BOUNDARY';

    @EngineeringElement {
        sourceId = "CTRL-REPO-LOCAL-WRITES";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-REPO-LOCAL-WRITES';

    @EngineeringElement {
        sourceId = "CTRL-STRICT-MCP-INPUT-SCHEMA";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-STRICT-MCP-INPUT-SCHEMA';

    @EngineeringElement {
        sourceId = "CTRL-TRACE-LINK-INTEGRITY";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-TRACE-LINK-INTEGRITY';

    @EngineeringElement {
        sourceId = "CTRL-TRACEABILITY-COVERAGE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-TRACEABILITY-COVERAGE';

    @EngineeringElement {
        sourceId = "DATA-ARCHITECTURE-DECISION-RECORDS";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-ARCHITECTURE-DECISION-RECORDS';

    @EngineeringElement {
        sourceId = "DATA-ARCHITECTURE-MODEL";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-ARCHITECTURE-MODEL';

    @EngineeringElement {
        sourceId = "DATA-CANONICAL-SEMANTIC-MODEL";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-CANONICAL-SEMANTIC-MODEL';

    @EngineeringElement {
        sourceId = "DATA-CATALOG-DOCUMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-CATALOG-DOCUMENT';

    @EngineeringElement {
        sourceId = "DATA-DECISIONS-DOCUMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-DECISIONS-DOCUMENT';

    @EngineeringElement {
        sourceId = "DATA-DESIGN-DOCUMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-DESIGN-DOCUMENT';

    @EngineeringElement {
        sourceId = "DATA-LOBSTER-TRACE-REPORT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-LOBSTER-TRACE-REPORT';

    @EngineeringElement {
        sourceId = "DATA-MCP-TOOL-RESULT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-MCP-TOOL-RESULT';

    @EngineeringElement {
        sourceId = "DATA-MODEL-AUTHORING-CONTRACT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-MODEL-AUTHORING-CONTRACT';

    @EngineeringElement {
        sourceId = "DATA-NAF-V4-ARCHITECTURE";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-NAF-V4-ARCHITECTURE';

    @EngineeringElement {
        sourceId = "DATA-OSCAL-SSP";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-OSCAL-SSP';

    @EngineeringElement {
        sourceId = "DATA-REQUIREMENTS-DELTA";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-REQUIREMENTS-DELTA';

    @EngineeringElement {
        sourceId = "DATA-REQUIREMENTS-DIFF";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-REQUIREMENTS-DIFF';

    @EngineeringElement {
        sourceId = "DATA-REQUIREMENTS-DOCUMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-REQUIREMENTS-DOCUMENT';

    @EngineeringElement {
        sourceId = "DATA-STRUCTURIZR-DSL";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-STRUCTURIZR-DSL';

    @EngineeringElement {
        sourceId = "DATA-SYSML-V2-METAMODEL";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-SYSML-V2-METAMODEL';

    @EngineeringElement {
        sourceId = "DATA-SYSML-V2-MODEL";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-SYSML-V2-MODEL';

    @EngineeringElement {
        sourceId = "DATA-THREAT-DRAGON-JSON";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-THREAT-DRAGON-JSON';

    @EngineeringElement {
        sourceId = "DATA-TRLC-PACKAGE";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-TRLC-PACKAGE';

    @EngineeringElement {
        sourceId = "DEP-CI-PIPELINE";
        conceptKind = "part_definition";
    }
    part def 'DEP-CI-PIPELINE' {
        attribute 'environment' : 'String' = "ci";
        attribute 'region' : 'String' = "hosted";
        attribute 'namespace' : 'String' = "engineering-model-go";
        attribute 'trustZone' : 'String' = "automation";
    }

    @EngineeringElement {
        sourceId = "DEP-LOCAL-WORKSPACE";
        conceptKind = "part_definition";
    }
    part def 'DEP-LOCAL-WORKSPACE' {
        attribute 'environment' : 'String' = "dev";
        attribute 'region' : 'String' = "local";
        attribute 'namespace' : 'String' = "engineering-model-go";
        attribute 'trustZone' : 'String' = "developer";
    }

    @EngineeringElement {
        sourceId = "DO-ARCHITECTURE-MODEL";
        conceptKind = "item_definition";
    }
    item def 'DO-ARCHITECTURE-MODEL' {
        attribute 'schemaRef' : 'String' = "internal/model/schema/architecture.cue";
        attribute 'classification' : 'String' = "design-source";
    }

    @EngineeringElement {
        sourceId = "DO-CANONICAL-SEMANTIC-MODEL";
        conceptKind = "item_definition";
    }
    item def 'DO-CANONICAL-SEMANTIC-MODEL' {
        attribute 'schemaRef' : 'String' = "internal/model/semantic.go";
        attribute 'classification' : 'String' = "design-source";
    }

    @EngineeringElement {
        sourceId = "DO-MCP-TOOL-RESULT";
        conceptKind = "item_definition";
    }
    item def 'DO-MCP-TOOL-RESULT' {
        attribute 'schemaRef' : 'String' = "mcp.tool-response.v1";
        attribute 'classification' : 'String' = "runtime-api";
    }

    @EngineeringElement {
        sourceId = "DO-MODEL-AUTHORING-CONTRACT";
        conceptKind = "item_definition";
    }
    item def 'DO-MODEL-AUTHORING-CONTRACT' {
        attribute 'schemaRef' : 'String' = "internal/model/authoring.go";
        attribute 'classification' : 'String' = "design-source";
    }

    @EngineeringElement {
        sourceId = "DO-NAF-V4-ARCHITECTURE";
        conceptKind = "item_definition";
    }
    item def 'DO-NAF-V4-ARCHITECTURE' {
        attribute 'schemaRef' : 'String' = "generated/ARCHITECTURE.naf.adoc";
        attribute 'classification' : 'String' = "architecture-product";
    }

    @EngineeringElement {
        sourceId = "DO-REQUIREMENTS-DELTA";
        conceptKind = "item_definition";
    }
    item def 'DO-REQUIREMENTS-DELTA' {
        attribute 'schemaRef' : 'String' = "internal/reqdelta/requirements_delta.go";
        attribute 'classification' : 'String' = "model-change";
    }

    @EngineeringElement {
        sourceId = "DO-REQUIREMENTS-DIFF";
        conceptKind = "item_definition";
    }
    item def 'DO-REQUIREMENTS-DIFF' {
        attribute 'schemaRef' : 'String' = "internal/reqdelta/requirements_delta.go";
        attribute 'classification' : 'String' = "model-change-preview";
    }

    @EngineeringElement {
        sourceId = "DO-REQUIREMENTS-DOCUMENT";
        conceptKind = "item_definition";
    }
    item def 'DO-REQUIREMENTS-DOCUMENT' {
        attribute 'schemaRef' : 'String' = "internal/model/schema/requirements.cue";
        attribute 'classification' : 'String' = "requirements-source";
    }

    @EngineeringElement {
        sourceId = "DO-STRUCTURIZR-DSL";
        conceptKind = "item_definition";
    }
    item def 'DO-STRUCTURIZR-DSL' {
        attribute 'schemaRef' : 'String' = "generated/STRUCTURIZR.dsl";
        attribute 'classification' : 'String' = "exchange-artifact";
    }

    @EngineeringElement {
        sourceId = "DO-SYSML-V2-METAMODEL";
        conceptKind = "item_definition";
    }
    item def 'DO-SYSML-V2-METAMODEL' {
        attribute 'schemaRef' : 'String' = "internal/export/sysml/metamodel";
        attribute 'classification' : 'String' = "external-standard";
    }

    @EngineeringElement {
        sourceId = "DO-SYSML-V2-MODEL";
        conceptKind = "item_definition";
    }
    item def 'DO-SYSML-V2-MODEL' {
        attribute 'schemaRef' : 'String' = "generated/ARCHITECTURE.sysml";
        attribute 'classification' : 'String' = "exchange-artifact";
    }

    @EngineeringElement {
        sourceId = "DO-THREAT-DRAGON-JSON";
        conceptKind = "item_definition";
    }
    item def 'DO-THREAT-DRAGON-JSON' {
        attribute 'schemaRef' : 'String' = "generated/threat-dragon-v2.json";
        attribute 'classification' : 'String' = "exchange-artifact";
    }

    @EngineeringElement {
        sourceId = "EVIDENCE-139EC30ED19D5D0E";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-139EC30ED19D5D0E';

    @EngineeringElement {
        sourceId = "EVIDENCE-73A27A3950E0B1C9";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-73A27A3950E0B1C9';

    @EngineeringElement {
        sourceId = "EVIDENCE-D1A0C0EF613B1206";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-D1A0C0EF613B1206';

    @EngineeringElement {
        sourceId = "EVIDENCE-EF5710F27BBD8EC7";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-EF5710F27BBD8EC7';

    @EngineeringElement {
        sourceId = "EVT-ARTIFACT-GENERATED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-ARTIFACT-GENERATED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-CONTRACT-DIFF-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-CONTRACT-DIFF-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-DEPENDENCY-BUMP-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-DEPENDENCY-BUMP-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-EXAMPLE-VALIDATION-REQUESTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-EXAMPLE-VALIDATION-REQUESTED';

    @EngineeringElement {
        sourceId = "EVT-GENERATION-RUN-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-GENERATION-RUN-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-LINK-MIGRATION-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-LINK-MIGRATION-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-LINK-REVIEW-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-LINK-REVIEW-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-MCP-TOOL-CALL-RECEIVED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-MCP-TOOL-CALL-RECEIVED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-MODEL-UPDATED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-MODEL-UPDATED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-RELEASE-CHECK-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-RELEASE-CHECK-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-REQUIREMENTS-DELTA-APPLY-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-REQUIREMENTS-DELTA-APPLY-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-REQUIREMENTS-DELTA-RECEIVED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-REQUIREMENTS-DELTA-RECEIVED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-SPEC-DELTA-REQUESTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-SPEC-DELTA-REQUESTED';

    @EngineeringElement {
        sourceId = "EVT-VALIDATION-FAILED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-VALIDATION-FAILED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-WORKSPACE-STAGING-REQUESTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-WORKSPACE-STAGING-REQUESTED';

    @EngineeringElement {
        sourceId = "EVT-WORKSPACE-SYNC-REQUESTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-WORKSPACE-SYNC-REQUESTED';

    @EngineeringElement {
        sourceId = "EXTENSION-9AF6071DD4978062";
        conceptKind = "engineering_extension";
    }
    attribute 'EXTENSION-9AF6071DD4978062';

    @EngineeringElement {
        sourceId = "FEAT-ARCHITECTURE-DECISION-RECORDS";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-ARCHITECTURE-DECISION-RECORDS';

    @EngineeringElement {
        sourceId = "FEAT-DO178C-DAL-C-READINESS";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-DO178C-DAL-C-READINESS';

    @EngineeringElement {
        sourceId = "FEAT-GEMARA-EXPORT";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-GEMARA-EXPORT';

    @EngineeringElement {
        sourceId = "FEAT-HARDWARE-INTERFACES";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-HARDWARE-INTERFACES';

    @EngineeringElement {
        sourceId = "FEAT-MCP-SERVER";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-MCP-SERVER';

    @EngineeringElement {
        sourceId = "FEAT-MODEL-WORKSPACE-REPLACEMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-MODEL-WORKSPACE-REPLACEMENT';

    @EngineeringElement {
        sourceId = "FEAT-NAF-EXPORT";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-NAF-EXPORT';

    @EngineeringElement {
        sourceId = "FEAT-NAF-PROFILE";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-NAF-PROFILE';

    @EngineeringElement {
        sourceId = "FEAT-SIX-VIEW-OUTPUT";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-SIX-VIEW-OUTPUT';

    @EngineeringElement {
        sourceId = "FEAT-STRUCTURIZR-EXPORT";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-STRUCTURIZR-EXPORT';

    @EngineeringElement {
        sourceId = "FEAT-SYSML-EXPORT";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-SYSML-EXPORT';

    @EngineeringElement {
        sourceId = "FEAT-SYSTEM-COMPOSITION";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-SYSTEM-COMPOSITION';

    @EngineeringElement {
        sourceId = "FEAT-THREAT-EXPORT";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-THREAT-EXPORT';

    @EngineeringElement {
        sourceId = "FEAT-TRACEABILITY-EXPORT";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-TRACEABILITY-EXPORT';

    @EngineeringElement {
        sourceId = "FG-ARTIFACT-GENERATION";
        conceptKind = "part_definition";
    }
    part def 'FG-ARTIFACT-GENERATION';

    @EngineeringElement {
        sourceId = "FG-MCP-INTEGRATION";
        conceptKind = "part_definition";
    }
    part def 'FG-MCP-INTEGRATION';

    @EngineeringElement {
        sourceId = "FG-MODEL-AUTHORING";
        conceptKind = "part_definition";
    }
    part def 'FG-MODEL-AUTHORING';

    @EngineeringElement {
        sourceId = "FG-TRACEABILITY-COMPLIANCE";
        conceptKind = "part_definition";
    }
    part def 'FG-TRACEABILITY-COMPLIANCE';

    @EngineeringElement {
        sourceId = "FG-VALIDATION-ANALYSIS";
        conceptKind = "part_definition";
    }
    part def 'FG-VALIDATION-ANALYSIS';

    @EngineeringElement {
        sourceId = "FLOW-DEPENDENCY-BUMP";
        conceptKind = "action_definition";
    }
    action def 'FLOW-DEPENDENCY-BUMP' {
        inout item 'item_DO_ARCHITECTURE_MODEL' : 'DO-ARCHITECTURE-MODEL';
    }

    @EngineeringElement {
        sourceId = "FLOW-LINK-REVIEW";
        conceptKind = "action_definition";
    }
    action def 'FLOW-LINK-REVIEW' {
        inout item 'item_DO_ARCHITECTURE_MODEL' : 'DO-ARCHITECTURE-MODEL';
    }

    @EngineeringElement {
        sourceId = "FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS";
        conceptKind = "action_definition";
    }
    action def 'FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS' {
        inout item 'item_DO_ARCHITECTURE_MODEL' : 'DO-ARCHITECTURE-MODEL';
        inout item 'item_DO_MCP_TOOL_RESULT' : 'DO-MCP-TOOL-RESULT';
        inout item 'item_DO_STRUCTURIZR_DSL' : 'DO-STRUCTURIZR-DSL';
        inout item 'item_DO_THREAT_DRAGON_JSON' : 'DO-THREAT-DRAGON-JSON';
    }

    @EngineeringElement {
        sourceId = "FLOW-REQUIREMENTS-DELTA-TO-CANONICAL";
        conceptKind = "action_definition";
    }
    action def 'FLOW-REQUIREMENTS-DELTA-TO-CANONICAL' {
        inout item 'item_DO_REQUIREMENTS_DELTA' : 'DO-REQUIREMENTS-DELTA';
        inout item 'item_DO_REQUIREMENTS_DOCUMENT' : 'DO-REQUIREMENTS-DOCUMENT';
        inout item 'item_DO_ARCHITECTURE_MODEL' : 'DO-ARCHITECTURE-MODEL';
        inout item 'item_DO_REQUIREMENTS_DIFF' : 'DO-REQUIREMENTS-DIFF';
    }

    @EngineeringElement {
        sourceId = "FU-AIRBORNE-ASSURANCE-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-AIRBORNE-ASSURANCE-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-ALLOCATION-TRACE";
        conceptKind = "part_usage";
    }
    part 'FU-ALLOCATION-TRACE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-ASCIIDOC-GENERATOR";
        conceptKind = "part_usage";
    }
    part 'FU-ASCIIDOC-GENERATOR' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-CLI-ORCHESTRATION";
        conceptKind = "part_usage";
    }
    part 'FU-CLI-ORCHESTRATION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-CODEMAP-INFERENCE";
        conceptKind = "part_usage";
    }
    part 'FU-CODEMAP-INFERENCE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-GEMARA-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-GEMARA-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-LOBSTER-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-LOBSTER-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-MCP-SERVER";
        conceptKind = "part_usage";
    }
    part 'FU-MCP-SERVER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-MODEL-CHANGE";
        conceptKind = "part_usage";
    }
    part 'FU-MODEL-CHANGE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-MODEL-LOADER";
        conceptKind = "part_usage";
    }
    part 'FU-MODEL-LOADER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-NAF-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-NAF-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-OSCAL-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-OSCAL-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-STRUCTURIZR-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-STRUCTURIZR-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-SYSML-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-SYSML-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-SYSTEM-COMPOSITION";
        conceptKind = "part_usage";
    }
    part 'FU-SYSTEM-COMPOSITION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-THREAT-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-THREAT-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-TRLC-EXPORTER";
        conceptKind = "part_usage";
    }
    part 'FU-TRLC-EXPORTER' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-VALIDATION-ENGINE";
        conceptKind = "part_usage";
    }
    part 'FU-VALIDATION-ENGINE' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-VIEW-PROJECTION";
        conceptKind = "part_usage";
    }
    part 'FU-VIEW-PROJECTION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "IF-CLI-ENGAIR";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGAIR' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engair";
        attribute 'schemaRef' : 'String' = "internal/export/airborne/airborne_assurance.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGCHANGE";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGCHANGE' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engchange";
        attribute 'schemaRef' : 'String' = "internal/reqdelta/requirements_delta.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGDOC";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGDOC' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engdoc";
        attribute 'schemaRef' : 'String' = "docs/asciidoc-architecture-generator-design.md";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGDRAGON";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGDRAGON' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engdragon";
        attribute 'schemaRef' : 'String' = "internal/export/threatmodel/threat_model_export.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGGEMARA";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGGEMARA' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/enggemara";
        attribute 'schemaRef' : 'String' = "internal/export/gemara/gemara_export.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGLOBSTER";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGLOBSTER' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/englobster";
        attribute 'schemaRef' : 'String' = "internal/export/lobster/lobster_export.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGMCP";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGMCP' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "stdio-jsonrpc";
        attribute 'endpoint' : 'String' = "cmd/engmcp";
        attribute 'schemaRef' : 'String' = "internal/mcp/server.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGNAF";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGNAF' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engnaf";
        attribute 'schemaRef' : 'String' = "internal/export/naf/naf_export.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGOSCAL";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGOSCAL' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engoscal";
        attribute 'schemaRef' : 'String' = "internal/export/oscal/oscal_ssp.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGSPEC";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGSPEC' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engspec";
        attribute 'schemaRef' : 'String' = "internal/reqdelta/spec_delta.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGSTRUCT";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGSTRUCT' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engstruct";
        attribute 'schemaRef' : 'String' = "internal/export/structurizr/structurizr_export.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGSYSML";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGSYSML' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engsysml";
        attribute 'schemaRef' : 'String' = "internal/export/sysml/sysml_export.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGTRACE";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGTRACE' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engtrace";
        attribute 'schemaRef' : 'String' = "internal/trace/trace_matrix.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGTRLC";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGTRLC' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engtrlc";
        attribute 'schemaRef' : 'String' = "internal/export/trlc/trlc_export.go";
    }

    @EngineeringElement {
        sourceId = "IF-CLI-ENGVIEW";
        conceptKind = "interface_definition";
    }
    interface def 'IF-CLI-ENGVIEW' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "cli";
        attribute 'endpoint' : 'String' = "cmd/engview";
        attribute 'schemaRef' : 'String' = "internal/view/project.go";
    }

    @EngineeringElement {
        sourceId = "MODE-CI-VALIDATION";
        conceptKind = "engineering_extension";
    }
    attribute 'MODE-CI-VALIDATION';

    @EngineeringElement {
        sourceId = "MODE-DRY-RUN";
        conceptKind = "engineering_extension";
    }
    attribute 'MODE-DRY-RUN';

    @EngineeringElement {
        sourceId = "MODE-LOCAL-DEVELOPMENT";
        conceptKind = "engineering_extension";
    }
    attribute 'MODE-LOCAL-DEVELOPMENT';

    @EngineeringElement {
        sourceId = "POAM-MCP-PATH-FUZZING";
        conceptKind = "engineering_extension";
    }
    attribute 'POAM-MCP-PATH-FUZZING';

    @EngineeringElement {
        sourceId = "POAM-TRACE-COVERAGE-GATE";
        conceptKind = "engineering_extension";
    }
    attribute 'POAM-TRACE-COVERAGE-GATE';

    @EngineeringElement {
        sourceId = "REF-CUE-OCI-MODULE-REGISTRY";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-CUE-OCI-MODULE-REGISTRY';

    @EngineeringElement {
        sourceId = "REF-GO-TOOLCHAIN";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-GO-TOOLCHAIN';

    @EngineeringElement {
        sourceId = "REF-LOBSTER-TOOLCHAIN";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-LOBSTER-TOOLCHAIN';

    @EngineeringElement {
        sourceId = "REF-NAF-V4-1-SPECIFICATION";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-NAF-V4-1-SPECIFICATION';

    @EngineeringElement {
        sourceId = "REF-OPEN-OTM-SCHEMA";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-OPEN-OTM-SCHEMA';

    @EngineeringElement {
        sourceId = "REF-STRUCTURIZR-VALIDATOR";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-STRUCTURIZR-VALIDATOR';

    @EngineeringElement {
        sourceId = "REF-SYSML-V2-TOOLCHAIN";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-SYSML-V2-TOOLCHAIN';

    @EngineeringElement {
        sourceId = "REF-THREAT-DRAGON-SCHEMAS";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-THREAT-DRAGON-SCHEMAS';

    @EngineeringElement {
        sourceId = "REF-TRLC-TOOLCHAIN";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-TRLC-TOOLCHAIN';

    @EngineeringElement {
        sourceId = "REQ-EMG-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-001' {
        attribute 'text' : 'String' = "While local development mode is active, when model updated event is received, the engineering model go shall validate architecture model before generation run is requested.";
        attribute 'notes' : 'String' = "Validation must gate all generation flows.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-003' {
        attribute 'text' : 'String' = "While model is valid, when generation run is requested, the engineering model go shall produce AsciiDoc architecture publication output without whitespace-only lines.";
        attribute 'notes' : 'String' = "Human-readable architecture output remains required and shall be clean for version control and downstream processing.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-004";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-004' {
        attribute 'text' : 'String' = "Where threat export is enabled, when generation run is requested, the engineering model go shall export threat dragon json artifact plus open threat model artifact.";
        attribute 'notes' : 'String' = "Threat model interchange output contract.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-005";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-005' {
        attribute 'text' : 'String' = "Where structurizr export is enabled, when generation run is requested, the engineering model go shall export structurizr dsl artifact.";
        attribute 'notes' : 'String' = "Structurizr DSL is exported for C4 visualization; the exported DSL is validated best-effort by the validate-all gauntlet when a Structurizr validator is available.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-006";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-006' {
        attribute 'text' : 'String' = "Where traceability export is enabled, when generation run is requested, the engineering model go shall produce trlc package and lobster trace report artifacts.";
        attribute 'notes' : 'String' = "Traceability outputs required for assurance workflows.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-007";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-007' {
        attribute 'text' : 'String' = "While mcp server is enabled, when mcp tool call is received and catalog references resolve successfully, the engineering model go shall return mcp tool result with schema version and structured error fields.";
        attribute 'notes' : 'String' = "MCP response contract stability requirement.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-008";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-008' {
        attribute 'text' : 'String' = "When mcp tool call is received, the engineering model go shall reject path arguments outside repository workspace boundary.";
        attribute 'notes' : 'String' = "Filesystem trust boundary guardrail.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-009";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-009' {
        attribute 'text' : 'String' = "While ci validation mode is active, when generation run is requested, the engineering model go shall execute validation plus tests before artifact generated event is raised.";
        attribute 'notes' : 'String' = "CI installs only the runtimes required by validation, uses mutually compatible exact validator versions and supported command-line semantics for those versions, and configures dependency caches only when their native project metadata exists, so setup and validation do not fail before the validation gauntlet completes.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-010";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-010' {
        attribute 'text' : 'String' = "Where requirement trace is complete, when model updated event is received, the engineering model go shall infer traceability coverage from trace markers.";
        attribute 'notes' : 'String' = "Links runtime, code, and test markers to modeled requirement coverage. Python and JavaScript declarations and lexical Verilog modules support explicit source links. Python decorators preserve definition coordinates; strings cannot create trace or ownership markers, and syntax errors are diagnosed; Verilog extraction does not elaborate HDL or expand macros. Individual source files and overlapping source directories preserve distinct file identities without double counting. Code and metadata scanning excludes dependency caches (node_modules, vendor) and dot-directories (.git, .engmod, tool caches) so coverage reflects only committed first-party source and stays reproducible.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-011";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-011' {
        attribute 'text' : 'String' = "Where external schema dependencies are available, when generation run is requested, the engineering model go shall fail validation if schema checks do not match expected constraints.";
        attribute 'notes' : 'String' = "Protect export integrity when schema trust fails.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-012";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-012' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall generate artifacts deterministically so that regeneration is reproducible.";
        attribute 'notes' : 'String' = "Generation uses stable ordering and controlled metadata so maintained artifacts are bit-for-bit reproducible. The validation gauntlet regenerates every maintained example and applies authoritative parsers or schemas where available plus semantic and idiomatic checks for each published format.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-013";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-013' {
        attribute 'text' : 'String' = "Where traceability export is enabled, when generation run is requested, the engineering model go shall export an OSCAL aggregate profile plus SSP plus assessment plan plus assessment results plus POA&M artifacts when the corresponding compliance and remediation content is applicable.";
        attribute 'notes' : 'String' = "OSCAL documents use version 1.1.2 structures, deterministic metadata, resolvable sibling import links, aggregate every implemented compliance framework, and omit inapplicable documents rather than emitting schema-invalid empty artifacts.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-014";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-014' {
        attribute 'text' : 'String' = "Where architecture decision records are authored, when generation run is requested, the engineering model go shall publish architecture decision records document linked from AsciiDoc architecture publication output.";
        attribute 'notes' : 'String' = "ADRs are first-class model content but are published separately from the main architecture narrative.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-015";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-015' {
        attribute 'text' : 'String' = "Where gemara export is enabled, when generation run is requested, the engineering model go shall render OpenSSF Gemara model artifacts.";
        attribute 'notes' : 'String' = "Gemara is a first-class GRC rendering of the model, validated against the published Gemara schemas via the go-gemara SDK.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-016";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-016' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall resolve the subsystem references declared by the parent system from local subdirectory paths.";
        attribute 'notes' : 'String' = "Recursive system-of-systems composition; local subdirectory references, with external git repositories cloned into the .engmod cache (REQ-EMG-027, ADR-EMG-008, ADR-EMG-010).";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-017";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-017' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall reject any subsystem reference that resolves outside the repository workspace boundary.";
        attribute 'notes' : 'String' = "Downward-only references remain within the workspace; mirrors the MCP path-boundary guardrail.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-018";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-018' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall reject a subsystem reference graph that contains a cycle.";
        attribute 'notes' : 'String' = "Subsystem references must form a directed acyclic graph (ADR-EMG-009).";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-019";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-019' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall resolve cross-system references using subsystem-qualified identifiers.";
        attribute 'notes' : 'String' = "Identifiers are scoped per system; cross-system references are qualified by subsystem to avoid collisions.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-020";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-020' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall materialize parent-to-subsystem allocation traceability in the composed output without modifying any subsystem model.";
        attribute 'notes' : 'String' = "A subsystem is parent-agnostic; parent allocation is materialized only in composed artifacts, never written back to the child (ADR-EMG-009, ADR-EMG-011).";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-021";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-021' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall report each subsystem required interface that the parent system does not satisfy.";
        attribute 'notes' : 'String' = "Subsystems publish a provides-and-requires contract; the parent must satisfy each required interface.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-022";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-022' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall reject an allocation that targets a subsystem identifier not published in the subsystem contract.";
        attribute 'notes' : 'String' = "Parents may bind only to a subsystem's public contract surface, not its internal identifiers.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-023";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-023' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall flag any requirement that links to another requirement in the same document.";
        attribute 'notes' : 'String' = "Requirements carry no tiers. A requirement is either delegated to a subsystem through a downward composition allocation or implemented here, and the delegation link alone tells the two apart. Linking requirements to one another inside a document recreates tier coupling and is confusing across system boundaries, so it is flagged.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-024";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-024' {
        attribute 'text' : 'String' = "Where hardware interfaces are authored, when generation run is requested, the engineering model go shall render the hardware to software interface allocation view.";
        attribute 'notes' : 'String' = "Hardware items and hardware interfaces capture the HW/SW interface (ICD/IRS) for the interface-control view.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-025";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-025' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall produce a system requirement allocation matrix aggregated across the composed subsystems.";
        attribute 'notes' : 'String' = "Aggregate roll-up generation mode for the integrating system.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-026";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-026' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall render in the requirement view the delegation link from each delegated requirement to the specific subsystem requirement that satisfies it.";
        attribute 'notes' : 'String' = "A provided contract entry names (ref) the subsystem requirement that realizes it, so a parent delegation traces to exactly the requirement that satisfies it. An allocated capability with no such ref is flagged as untraceable, and a ref to a requirement the subsystem does not define is rejected.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-027";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-027' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall resolve an external git subsystem reference by cloning the repository into the workspace .engmod cache.";
        attribute 'notes' : 'String' = "A subsystem may reference an external repository with git (and an optional rev and path). The model is cloned once into .engmod/subsystems/<id> — a gitignored, temporary cache reused while it points at the same repository — and resolved like a local subsystem, giving a full cross-repository composed view. References stay downward-only.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-028";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-028' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall reject a code trace link that resolves to no requirement or model element.";
        attribute 'notes' : 'String' = "Code markers (TRLC-LINKS to a requirement, ENGMODEL-LINKS to a model element) are existence-checked against the model, not only shape-checked, so a link to a deleted or mistyped id is a dangling link and fails the build. A trace link is only checked for the model that owns its code (see REQ-EMG-030 scoping).";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-029";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-029' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall flag each requirement that is traced by neither code nor verification nor a downward delegation.";
        attribute 'notes' : 'String' = "An orphan requirement has no implementing code symbol, no verification, and is not delegated to a subsystem; it is a coverage gap surfaced as a warning so it is visible without breaking the build.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-030";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-030' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall export a machine-readable traceability matrix that links each requirement to its implementing code and verification and delegated subsystem requirement.";
        attribute 'notes' : 'String' = "One consolidated matrix (JSON) replaces having to read it out of prose and diagrams. Test-only source links count as verification, not production implementation. Code is attributed to the model whose engmod.yml is its nearest enclosing root, so a parent model never validates or counts a child model's trace links (monorepo scoping).";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-031";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-031' {
        attribute 'text' : 'String' = "When requirements delta is received, the engineering model go shall strictly validate requirements delta schema plus stable identifier operation uniqueness.";
        attribute 'notes' : 'String' = "Version 1 supports add, whole-entity update, and remove operations. Unknown fields, empty operations, duplicate operations, adds of existing IDs, and updates or removals of missing IDs are rejected.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-032";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-032' {
        attribute 'text' : 'String' = "While dry run mode is active, when requirements delta is received, the engineering model go shall validate candidate architecture model without modifying requirements document.";
        attribute 'notes' : 'String' = "The candidate reuses complete-model structural, catalog, EARS, and internal-link validation before any write is permitted.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-033";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-033' {
        attribute 'text' : 'String' = "While model is valid, when requirements delta apply is requested, the engineering model go shall atomically replace requirements document.";
        attribute 'notes' : 'String' = "The YAML AST merger preserves unchanged nodes and comments. Atomic replacement preserves file permissions and refuses a candidate with blocking diagnostics.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-034";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-034' {
        attribute 'text' : 'String' = "When requirements delta is received, the engineering model go shall produce deterministic requirements change summary.";
        attribute 'notes' : 'String' = "The summary sorts added, updated, and removed stable IDs and reports changed fields for whole-entity updates.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-035";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-035' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall normalize architecture model plus requirements document into canonical semantic model.";
        attribute 'notes' : 'String' = "The canonical semantic model is SysML v2 and KerML aligned. Existing Engineering Model concepts map into that model once; legacy YAML remains an input adapter during migration and must not create a parallel set of parts, interfaces, flows, requirements, allocations, or views.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-036";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-036' {
        attribute 'text' : 'String' = "Where sysml export is enabled, when generation run is requested, the engineering model go shall generate sysml v2 model artifact from canonical semantic model.";
        attribute 'notes' : 'String' = "The exporter preserves stable identifiers and maps standard concepts to native SysML v2 or KerML semantics. Engineering-specific concepts may use explicit typed SysML metadata definitions, but canonical data without a native target representation must be omitted rather than encoded in descriptions, opaque payloads, generic property bags, or sidecar files.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-037";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-037' {
        attribute 'text' : 'String' = "While ci validation mode is active, when generation run is requested, the engineering model go shall validate sysml v2 model artifact before artifact generated event is raised.";
        attribute 'notes' : 'String' = "Validation must report unsupported or lossy mappings and progress toward abstract-syntax, textual-syntax, semantic, and model-interchange conformance. A syntactically valid subset export must not be described as a complete SysML v2 semantic superset.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-038";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-038' {
        attribute 'text' : 'String' = "Where sysml export is enabled, when generation run is requested, the engineering model go shall validate canonical semantic model against sysml v2 metamodel.";
        attribute 'notes' : 'String' = "The pinned formal SysML 2.0 and KerML 1.0 metamodel is the source of truth for supported classes, properties, relationships, ownership, and multiplicities. Generated CUE constraints and semantic registries must not be maintained as independent handwritten copies.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-039";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-039' {
        attribute 'text' : 'String' = "While ci validation mode is active, when generation run is requested, the engineering model go shall reject incomplete sysml v2 metamodel coverage.";
        attribute 'notes' : 'String' = "Coverage is complete only when every normative metaclass and property in the pinned formal release is represented, explicitly excluded as derived, or implemented as a typed Engineering Model extension with executable evidence.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-040";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-040' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall preserve the deterministic native sysml v2 source through project interchange round trip.";
        attribute 'notes' : 'String' = "Sysand owns normative KPAR packaging and reopening. Verification compares the reopened SysML source byte-for-byte with a fresh projection and validates both sources with the pinned official parser; it does not claim lossless reconstruction of canonical concepts omitted from SysML.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-041";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-041' {
        attribute 'text' : 'String' = "Where naf profile authoring is enabled, when model updated event is received, the engineering model go shall validate naf v4.1 stakeholder plus concern plus viewpoint metadata against canonical model references.";
        attribute 'notes' : 'String' = "NAF-specific YAML contains profile metadata only. Capability, service, logical, resource, behavior, interface, and requirement concepts remain represented once in the canonical SysML and KerML aligned semantic graph.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-042";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-042' {
        attribute 'text' : 'String' = "Where naf export is enabled, when generation run is requested, the engineering model go shall generate deterministic naf v4.1 architecture document from referenced canonical model views.";
        attribute 'notes' : 'String' = "Each NAF architecture product uses an official NAF v4.1 viewpoint code and references an existing Engmod view so generated content preserves canonical identities and relationships.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-043";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-043' {
        attribute 'text' : 'String' = "While ci validation mode is active, when generation run is requested, the engineering model go shall reject unsupported naf version plus unknown viewpoint code plus dangling profile reference before artifact generated event is raised.";
        attribute 'notes' : 'String' = "Validation covers stakeholder actor bindings, concern ownership, architecture-product concern references, and Engmod view references.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-044";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-044' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall load schema version 2 engmod manifest plus catalog plus requirements plus architecture plus behavior plus assurance plus compliance plus views plus decisions documents into one canonical semantic model.";
        attribute 'notes' : 'String' = "engmod.yml is the only model entry point and every document path is explicit. YAML is the authoring format and CUE is the schema authority. Requirements may carry format-neutral qualification metadata such as title, category, rationale, source references, verification methods and criteria, priority, criticality, status, derived disposition, and tags. Views own design narratives plus generic document definitions, document-control metadata, content selections, authored sections, and external references; architecture does not contain behavior, assurance, compliance, dependency source, NAF, or projection-specific declarations.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-045";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-045' {
        attribute 'text' : 'String' = "While mcp server is enabled, when mcp tool call is received and catalog references resolve successfully, the engineering model go shall return schema version 2 model authoring contract.";
        attribute 'notes' : 'String' = "The contract contains the manifest document map, authoritative CUE schema for each domain document, stable-ID and typed-relationship conventions, dependency publication aliases, valid references, and recommended task-scoped authoring order.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-046";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-046' {
        attribute 'text' : 'String' = "While ci validation mode is active, when generation run is requested, the engineering model go shall reject model input that violates the schema version 2 manifest or any referenced domain document contract before artifact generated event is raised.";
        attribute 'notes' : 'String' = "Missing documents, unsupported versions, unknown fields, duplicate concepts, invalid document-control values, incomplete derived requirement rationale, unresolved document content selections, unresolved local or qualified references, dependency publication violations, and cross-domain ownership violations are blocking.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-047";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-047' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall resolve each exact CUE module version from the OCI registry configured by CUE_REGISTRY.";
        attribute 'notes' : 'String' = "Dependencies are declared only in engmod.yml with an alias, major-qualified module path, exact semantic version, and selected publications. The resolved module contains engmod.yml, its domain documents, and cue.mod/module.cue.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-048";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-048' {
        attribute 'text' : 'String' = "Where a model workspace replacement is declared, when generation run is requested, the engineering model go shall resolve the matching manifest dependency from the replacement repository without changing module identity or selected publications.";
        attribute 'notes' : 'String' = "engmod.work.yml maps module paths to local repositories for offline multi-repository development. A generated engmod.lock.yml records exact module versions and OCI digests but is not an authored semantic document.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-049";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-049' {
        attribute 'text' : 'String' = "While ci validation mode is active, when generation run is requested, the engineering model go shall reject invalid manifest dependency plus publication plus workspace replacement plus qualified reference declarations.";
        attribute 'notes' : 'String' = "Every dependency requires one alias, one major-qualified module path, one exact version, and at least one exported publication. Aliases are unique, module identities match cue.mod/module.cue, workspace paths remain contained, imported references use alias double-colon stable-id syntax, and transitive entities are unavailable unless re-exported. A resolved dependency (registry module or workspace replacement) is confined to its module root: before any of its documents is read, every document reference, inferenceHints runtimeSources entry and inferenceHints exclude glob, engmod.yml and cue.mod/module.cue that is absolute or resolves after symlinks outside the module is rejected as composition.dependency_path_escape naming the dependency alias and field, and no content of the target is read; the root model's own paths are not confined this way. A fetched module zip is checked before extraction (modzip rejects .. and absolute entries, the registry client verifies the blob digest) and an entry that is not a regular file or directory, such as a symlink, fails the fetch.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-050";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-050' {
        attribute 'text' : 'String' = "When example validation is requested, the engineering model go shall validate a schema version 2 company portfolio composed from multiple product systems that reuse published hardware plus software plus security plus compliance facets across repository boundaries.";
        attribute 'notes' : 'String' = "The reference workspace demonstrates explicit domain documents, dependency publication selection, qualified imported references, requirement delegation, contract satisfaction, inherited company controls, technical security controls, employee lifecycle compliance, provenance, exact OCI coordinates, local workspace replacements, and complete native format projections.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-051";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-051' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall import only explicitly selected dependency publications with qualified identity plus source provenance.";
        attribute 'notes' : 'String' = "Each dependency publication exposes typed architecture, behavior, assurance, compliance, requirement, and view references. Imported entities retain module path, exact version, digest, source model, and publication identity; unexported dependency entities remain inaccessible.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-052";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-052' {
        attribute 'text' : 'String' = "Where six-view example output is enabled, when generation run is requested, the engineering model go shall generate deterministic use-case plus logical plus process plus physical plus implementation plus deployment view artifacts and native SysML from one schema version 2 YAML bundle.";
        attribute 'notes' : 'String' = "The acceptance example preserves viewpoint separation, hierarchical capabilities, ordered typed interactions, nested hardware and software structures, ports, connections, engineering values, and software-to-hardware allocations. Mermaid sources and SVG renderings are generated without depending on authored SysML source files.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-053";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-053' {
        attribute 'text' : 'String' = "Where do-178c dal c readiness output is enabled, when model updated event is received, the engineering model go shall load a strict aviation assurance profile plus lifecycle data plus verification evidence plus configuration baselines plus tool assessments plus structural coverage summaries.";
        attribute 'notes' : 'String' = "The model references DO-178C or ED-12C and customer-controlled licensed objective identifiers without reproducing normative objective text. It distinguishes declared, planned, executed, reviewed, accepted, not-applicable, and authority-accepted evidence states.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-054";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-054' {
        attribute 'text' : 'String' = "Where do-178c dal c readiness output is enabled, when generation run is requested, the engineering model go shall reject incomplete or inconsistent requirement classification plus derived requirement disposition plus verification chain plus independence plus baseline plus structural coverage plus tool assessment evidence.";
        attribute 'notes' : 'String' = "DAL assignment and objective applicability remain externally authorized inputs. Statement coverage is imported evidence tied to an exact build and environment; the tool does not perform safety assessment, coverage measurement, reviewer qualification, tool qualification, or certification approval.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-055";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-055' {
        attribute 'text' : 'String' = "Where do-178c dal c readiness output is enabled, when generation run is requested, the engineering model go shall generate deterministic draft lifecycle data index plus objective evidence matrix plus requirements trace matrix plus verification summary plus structural coverage summary plus tool assessment register plus configuration index plus open gap report plus software accomplishment summary.";
        attribute 'notes' : 'String' = "Every output is visibly marked as draft evidence-readiness material, identifies its source baseline and unresolved gaps, and states that responsible human review plus configuration control plus certification authority acceptance remain required.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-056";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-056' {
        attribute 'text' : 'String' = "When example validation is requested, the engineering model go shall validate a schema version 2 DAL C flight control software example with placeholder planning plus requirements plus design plus code plus verification plus coverage plus configuration plus quality assurance plus certification liaison evidence.";
        attribute 'notes' : 'String' = "The example demonstrates realistic output shapes and intentionally open findings without claiming DO-178C compliance or certification.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-057";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-057' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall derive the default code scan root from the enclosing module boundary.";
        attribute 'notes' : 'String' = "Without codeSources or an explicit code root, the scan root is the git working tree root, unless another engmod.yml exists between the manifest directory and that root, in which case it is the manifest directory. Directories holding another engmod.yml belong to that module and are skipped. Files ignored by git and paths matching inferenceHints.exclude glob patterns (relative to the module root, with dir/** for whole subtrees) are skipped. Explicit codeSources and code roots still override the default root.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-058";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-058' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall assign path-derived CODE identifiers to every in-scope file with its declarations.";
        attribute 'notes' : 'String' = "A file identifier is CODE- followed by its module-root-relative path, upper-cased with runs of non-alphanumeric characters folded to one hyphen (app/internal/capture.go becomes CODE-APP-INTERNAL-CAPTURE-GO). A declaration identifier appends two hyphens and the folded declaration name, so same-named declarations in different files stay distinct. Explicit TRACE-ID markers keep their authored value. Identifiers do not carry commit or blob versions because the model and the code are versioned together.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-059";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-059' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall resolve authored references to CODE identifiers against the in-scope module files.";
        attribute 'notes' : 'String' = "An unknown CODE identifier is a broken reference, an identifier produced by more than one file or declaration is rejected as ambiguous, and inferred RT- and AUTO- identifiers remain unreferenceable. The code index is built only when the model references a CODE identifier, so models without such references pay no scanning cost.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-060";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-060' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall reject referenced elements whose uri addresses a local path.";
        attribute 'notes' : 'String' = "Referenced elements describe artifacts outside the repository. A uri without a scheme or with the file scheme is rejected with a hint to reference the CODE identifier instead. A referenced element with a uri must declare a version unless the uri itself is immutable (a doi, or a uri carrying a full 40-hex commit id). Referenced elements without a uri remain valid for offline documents identified by number and edition.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-061";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-061' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall count a requirement as tested only through test entry points.";
        attribute 'notes' : 'String' = "Test entry points are Go Test/Benchmark/Fuzz/Example functions, Rust";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-062";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-062' {
        attribute 'text' : 'String' = "Where gemara export is enabled, when generation run is requested, the engineering model go shall derive each gemara enforcement disposition plus policy risk treatment from the model status.";
        attribute 'notes' : 'String' = "Only a completed POA&M item is Enforced and only an accepted one Tolerated; planned, in-progress, deferred and unset items are Undetermined, and so is the log aggregate while any item is unresolved. The policy lists a risk as mitigated only when its status is closed and as accepted only when its status is accepted or its response is accept; open and mitigating risks are left out with a gemara.policy_risks_untreated warning. The generator author is typed Software unless overridden. gemara.validate reports claims the model does not support as modelConsistencyErrors.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-063";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-063' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall select the model workspace from the ENGMOD_WORK setting.";
        attribute 'notes' : 'String' = "ENGMOD_WORK=off ignores every engmod.work.yml (CI and the pre-push gate); ENGMOD_WORK=<path> uses that file only and skips the parent search; unset searches parent directories as before. A path that does not exist is an error. CLI tools and the MCP server accept --workspace=off|<path> with the same meaning.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-064";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-064' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall write or check the composition lock according to the lock mode.";
        attribute 'notes' : 'String' = "While workspace replacements are active the committed engmod.lock.yml is never modified and the resolved lock goes to engmod.lock.local.yml. With ENGMOD_LOCK=check (or --lock=check) the resolved lock is compared with engmod.lock.yml and composition.lock_missing, composition.lock_invalid or composition.lock_mismatch (missing, extra or version-changed dependency, changed content digest after a moved tag) is reported and nothing is written. Exception: while workspace replacements are active the committed lock is not compared (it pins published content, the replacements are local) and the warning composition.lock_check_skipped says that it is checked only with ENGMOD_WORK=off. Otherwise engmod.lock.yml is written as before. A cached registry module is reused offline, except that ENGMOD_MODULE_CACHE=refresh and lock regeneration (write mode with the workspace off) re-fetch every dependency, ENGMOD_MODULE_CACHE=reuse (ResolutionOptions.ModuleCache) reuses the cache offline in every lock mode, and otherwise lock check mode compares each cached module's recorded manifest digest with the registry (one lookup per dependency, cache reused when offline) and re-fetches a moved tag, so a warm cache never masks it. Diagnostics located in a fetched dependency name it as module@version:path instead of the cache directory.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-065";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-065' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall validate each qualified reference against the published entities of the selected dependencies.";
        attribute 'notes' : 'String' = "Covers relationships, flows and flow steps, appliesTo, interface and data references, ownership (functional unit group, interface owner, trust boundary parent, risk, threat and verification owners), view roots and code ENGMODEL-LINKS. Requirements stay local, so trace counts do not include imported requirements. A qualified reference to an entity the dependency does not publish is reported once as composition.unpublished_reference with its field path, not as a generic broken reference. Only reference-typed fields are checked; prose (descriptions, notes, rationale, flow step actions) and file or URL fields (schemaRef, catalogRef, href) may contain :: freely.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-066";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-066' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall return the composition diagnostics with the generation result.";
        attribute 'notes' : 'String' = "GenerateAsciiDoc and the trace matrix return every composition diagnostic, so callers need not compose again; composition runs once per call and its result is reused. An unsatisfied required interface is an error for a model of kind system that has no parent and a warning otherwise; an allocation whose target is not a contract provides entry is a warning.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-067";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-067' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall show the trace status of each delegated requirement in its target repository.";
        attribute 'notes' : 'String' = "For a delegated row the matrix computes the target requirement's own trace status (implemented, test linked, orphan) from the resolved dependency at its pinned or replaced version and reports it as delegatedStatus. The dependency's code scope is its own module directory (the workspace checkout or the module cache entry) minus its inferenceHints exclude globs, never the consumer repository around it; a dependency without code there shows not-available.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-068";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-068' {
        attribute 'text' : 'String' = "When workspace sync is requested, the engineering model go shall write a local workspace that replaces only changed or pinned checkouts.";
        attribute 'notes' : 'String' = "engws scans direct child directories with engmod.yml. A checkout is changed when its git tree is dirty, HEAD is not at tag v<module.version>, or that version is not in the registry configured by CUE_REGISTRY; .engws.yml pins force local. The root search stops below $HOME (never $HOME or an ancestor) and refuses a world-writable root unless --root names it; every command prints the root first. sync skips with a warning a checkout whose module.path is not a valid major-qualified CUE module path or whose directory name is outside [A-Za-z0-9._-]. sync writes engmod.work.yml with the local checkouts and a go.work with every checkout that has go.mod (a non-local checkout is at its published tag, so go test and go run work in every checkout; go.work is removed when none has go.mod), and writes nothing inside any checkout. status and skew report SKEW when a local checkout's version differs from a version another checkout pins, or when a consumer's go.mod requires a checkout's Go module at another version than the checkout's version tag, and exit 1. check runs the model check in every local checkout affected by the change (reverse dependencies inside the workspace) and the top system with summary output; a repository's own .github/engmod-check is trusted code (stated in the help and the check output; --in-process is the choice for untrusted checkouts) and runs with an allowlisted environment (PATH, HOME, Go, CUE, proxy, certificate, XDG, DOCKER_CONFIG, CI and ENGMOD_* variables; no API keys or other credentials) and GOWORK set to its .github/engmod-check/go.work when present and off otherwise, never the root go.work; the in-process check relies on the library's qualified-reference validation; --refresh re-fetches dependencies; --all does not warn that engmod.work.yml lacks the unchanged checkouts. bump writes inside the consumer checkout: it updates a dependency version in engmod.yml, the matching go.mod require and go.sum (go get with GOWORK=off, after validating the Go module path) and the lock together, and restores all of them on any failure; pin and unpin write .engws.yml at the root.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-069";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-069' {
        attribute 'text' : 'String' = "While mcp server is enabled, when mcp tool call is received, the engineering model go shall return authoring assistance for an edited model document.";
        attribute 'notes' : 'String' = "model.complete returns local and imported qualified IDs (from dependencies pulled into the local cache), catalog terms, allowed enum values and relationship type pairs that match a prefix and an optional field; model.diagnose validates the model with one document's content supplied in the call and returns that document's diagnostics; dependencies resolve from the real project's workspace and module cache while the root system, its composition and its alias::ID references are taken from the submitted content, and no lock is written. Both work offline from the local cache; no central service is involved. Every composing MCP read tool (model.complete, model.diagnose, composition.resolve, trace.matrix) resolves with lock check unless ENGMOD_LOCK is set and with module cache reuse unless ENGMOD_MODULE_CACHE or ENGMOD_LOCK is set, so it never writes engmod.lock.yml and makes no registry request when the cache is warm. composition.resolve returns resolved directories relative to the workspace root, or as module@version for module cache entries, never as absolute paths.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-070";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-070' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall warn about every unconnected behavior element.";
        attribute 'notes' : 'String' = "Three warnings. behavior.event_unused names an authored behavior event that no triggered_by relationship targets, no semantic transition lists as a trigger and no semantic relationship, flow or flow step references (catalog-only events are covered by catalog.term_unreferenced). behavior.state_unreachable names, in a model with at least one transitions_to relationship or semantic transition, an authored state other than the first declared one that no transition enters; the schema has no initial marker, so the first declared state is the initial state, and a model without transitions reports nothing. behavior.flow_step_no_onerror names a flow step with boundaryCrossing true and no onError step, at its onError path. Imported alias::ID entities are skipped.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-071";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-071' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall warn at each catalog term whose name contains an EARS expression operator.";
        attribute 'notes' : 'String' = "catalog.name_has_operator is reported once at the catalog name or alias field (catalog.<group>[i].name or .aliases[j]) of an actor, event, state, feature, mode, condition or data term that contains a standalone and, or, not, a comma or a parenthesis. Such a term cannot match in a requirement clause, because the EARS expression parser splits it at the operator, and the requirement otherwise shows only unresolved-term diagnostics. System, functional group and unit names are matched whole by EARS and are not checked.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-072";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-072' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall report a multiple-response lint only for a requirement with more than one response action.";
        attribute 'notes' : 'String' = "lint.multiple_responses from ears-lint-go is dropped when the text has one shall and no semicolon, no and, or or comma comes before the first with, including or of in the response, that field list holds no then, and no list item after the first reads as a verb with its object (its second word is the, a, an, its, their, each, every or all). \"publish one world state message with UTC time, latitude, longitude and altitude\" is not reported; \"log the event and notify the operator\", \"log the event with the timestamp and notify the operator\" and \"stop all of the motors and close the valve\" still are.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-073";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-073' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall warn about each control verification status that lags its existing tests.";
        attribute 'notes' : 'String' = "A control verification may name its tests in verificationRefs: inferred CODE identifiers of test declarations (CODE-<PATH>--<NAME>) or requirement IDs whose linked test entry points verify it; any other entry is the error model.invalid_control_verification_ref. verification.status_stale is a warning at the status of a control verification whose status is unset or not-run while a referenced test declaration exists in the scanned code or a referenced requirement is test linked, and at the status of a threat mitigation whose status is unset or planned while a control verification it references has such tests. The status stays authored. The AsciiDoc generation and the trace matrix both report it.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-080";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-080' {
        attribute 'text' : 'String' = "When workspace staging is requested, the engineering model go shall publish every workspace checkout into the local staging area in dependency order.";
        attribute 'notes' : 'String' = "engws stage [--version-from manifest|tag] [--serve] [--addr] writes only under <root>/.engmod/staging, never inside a checkout. registry/<module path>/ holds each CUE module as an OCI image layout with the content cue mod publish would push (source kind git: the committed files from git ls-files with VCS metadata; kind self: the directory). goproxy/ is a GOPROXY file tree (<module>/@v/<version>.info, .mod, .zip and list) for each checkout with go.mod at its top, zipped from committed files. The version is module.version from engmod.yml; with --version-from tag HEAD must be at tag v<module.version>. A checkout cue mod publish would refuse (dirty tree, no source field, module path mismatch) is reported, and then nothing is staged. A consumer pinning a family module at another version than the staged one is warned about. CUE has no file registry, so --serve loads the layouts into an in-memory read-only OCI registry on a loopback address only and serves it in the foreground until interrupted. stage prints the gate environment: CUE_REGISTRY routes for the staged modules merged with the current CUE_REGISTRY (with --serve), GOPROXY=file://<staging>/goproxy,off, GONOPROXY=none, GONOSUMDB with the staged Go modules plus the current GONOSUMDB or GOPRIVATE, GOWORK=off and ENGMOD_WORK=off; no -mod flag.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-081";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-081' {
        attribute 'text' : 'String' = "When spec delta is requested, the engineering model go shall render OpenSpec spec delta text from the selected model requirements.";
        attribute 'notes' : 'String' = "engmodel.RenderSpecDelta and RenderSpecDeltaFromModel, and the engspec CLI (--model, --ids or --prefix, --capability, --modified, --out), render ## ADDED (or MODIFIED) Requirements and per requirement a ### Requirement heading (title, else ID; a repeated title names its ID), the line <ID>: <model text> with whitespace collapsed and whole-word shall and must upper-cased, so it starts with the model text under the engineering-flow consistency normalization (lower case, collapsed whitespace, no backticks or trailing period), and one #### Scenario from verificationCriteria: GIVEN, WHEN and THEN bullets when the criteria are shaped [given …,] when …, then …; otherwise WHEN the requirement is verified and THEN the criteria. Requirements without shall or must, or without verificationCriteria, are returned as warnings; an unknown ID is an error.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-082";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-082' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall report each assurance record whose verification links disagree.";
        attribute 'notes' : 'String' = "threat.verification_missing is a warning at a threat scenario's verificationRefs when it omits a control verification that one of its threat mitigations (threatScenarioRef) lists in verificationRefs or that names the threat in threatScenarioRefs. mitigation.verification_control_mismatch is an error at a threat mitigation verificationRefs entry naming a control verification whose controlRef is another control than the mitigation's controlRef. verification.unlinked is a warning at the verificationRefs of a control verification with method test and status pass or partial that has no verificationRefs, so no test declaration or requirement evidences it; analysis, inspection and other methods are evidenced by their evidence entries and are not reported. Imported alias::ID records are skipped. The AsciiDoc generation, the trace and model.diagnose report them through validation.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-083";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-083' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall warn at each control that the gemara evaluation log would mark for review.";
        attribute 'notes' : 'String' = "gemara.control_needs_review is a warning at a control that the Gemara L5 evaluation log gives the result Needs Review: no control verification names it in controlRef, or at least one of its control verifications has a status that needs review (partial, planned, in-progress, review, needs-review) and none failed or has an unrecognized status, since Failed and Unknown outrank Needs Review in the log's aggregate. validate.ClassifyVerificationStatus and validate.ControlNeedsReview are the one status mapping the evaluation log and the warning share. When a control without verification has a threat mitigation that references another control's verification, the message names it (\"<CV> verifies <other control>\"); for a control whose verifications need review it names them with their status. Validation emits it, so the check path (engdoc, the trace, model.diagnose) reports it without rendering Gemara.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-084";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-084' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall infer implementation coverage from trace markers in comments of build and documentation artifacts.";
        attribute 'notes' : 'String' = "The code scanner reads TRLC-LINKS, ENGMODEL-LINKS and ENGMODEL-OWNER-UNIT in # comment lines of files named Dockerfile or Containerfile or ending in .Dockerfile or .Containerfile, and in HTML comments (<!-- TRLC-LINKS: REQ-… -->, also spanning lines) that start a line outside fenced code blocks of .md and .markdown files. The markers apply to the whole file: it becomes one symbol with its file CODE identifier (CODE-<PATH>) that implements the linked requirements, never verifies them, also under tests/. Markers are optional: such a file without them yields no item and no code.missing_trlc_link. The module code scope applies, including git-ignored files and inferenceHints.exclude; an invalid requirement ID is code.invalid_trlc_link.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-085";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-085' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall report each accepted assurance record that lacks an owner acceptance record.";
        attribute 'notes' : 'String' = "A risk with status accepted or response accept, and a threat scenario with status accepted, must carry acceptedBy with actor (an existing ACT-* actor), date (a YYYY-MM-DD calendar date, parsed, so 2026-13-45 is rejected) and statement (the owner's own words) or decisionRef (an existing decision with status accepted whose context quotes the owner); otherwise risk.acceptance_unrecorded is an error at its acceptedBy field naming the missing parts. An acceptedBy actor, date or decisionRef that does not resolve, or a decisionRef to a decision that is not accepted, is model.invalid_acceptance (error). risk.acceptance_not_owner is a warning at the acceptedBy actor of a risk whose owner is set and differs from the actor. risk.accept_while_open is a warning at a risk status when response is accept and status is open, since the Gemara policy would otherwise list it as accepted while it is open. The Gemara policy lists a risk under risks.accepted only when its acceptance record is complete and resolves (validate.AcceptanceRefs, shared with validation) and its status is accepted, or its response is accept and its status is not open; gemara.validate (CheckGemaraModelConsistency) reports an accepted entry that does not meet this. Imported alias::ID records are skipped; the schema rejects :: in authored risk and threat scenario IDs, so a local ID cannot pose as imported. The record is an authored claim: these checks make it complete and consistent, and human review of the diff that adds it is the actual control that the owner gave it.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-086";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-086' {
        attribute 'text' : 'String' = "When workspace sync is requested, the engineering model go shall write a go workspace replace directive for each required version of a workspace go module.";
        attribute 'notes' : 'String' = "engws sync writes, next to the use entries of the family go.work, one line replace <module> <version> => ./<dir> for every Go module of a checkout listed in use and every distinct version at which any checkout's go.mod requires that module; a module no checkout requires gets no replace. go resolves the go.mod of a required version even when the module is in use, so an unpublished family module fails to build without it; an unversioned replace is rejected for a workspace module. The generated-file header and the hand-written file check are unchanged. A repository's .github/engmod-check uses its own go.work or GOWORK=off and is not affected.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-087";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-087' {
        attribute 'text' : 'String' = "While mcp server is enabled, when mcp tool call is received, the engineering model go shall return the allowed fields of each entity kind with the model authoring contract.";
        attribute 'notes' : 'String' = "validate.AuthoringRules (model.authoringContract rules) adds fieldSets, the YAML field names each entity kind accepts (risk, control, threatScenario, flowStep and the others), derived from the strict-decoding Go types, and publicationSections, the manifest publication sections (architecture, behavior, assurance, compliance, requirements, views; decisions cannot be published). fieldRules state that boundaryCrossing true needs trustBoundaryRef (model.missing_flow_boundary_ref), what behavior.event_unused counts as a use (triggered_by targets, semantic relationship sources, targets and triggers, flow and flow step sourceRef and destinationRef, flow step ref; not properties or prose), the lint.multiple_responses field-list exemption (one shall, the list opens with with, including or of, no list item after the first starts a verb with a determiner) and that in strict EARS mode the system term is a catalog system, functional group or functional unit name or alias.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-088";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-088' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall report each requirement whose linked tests are all environment gated.";
        attribute 'notes' : 'String' = "A Go test entry point whose first statements include an if whose condition calls os.Getenv, os.LookupEnv or testing.Short and whose body calls t.Skip, t.Skipf or t.SkipNow is gated (the variable name, or testing.Short). A requirement linked from a test file only by gated tests carries that gate in the verification check; a trace row whose every verification is gated for it has gated true and its gates, and the summary counts it in gatedOnly and lists it in gatedOnlyRequirements. testLinked still counts it. The MCP trace.matrix summary returns gatedOnly and gatedOnlyRequirements and its rows gated.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-089";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-089' {
        attribute 'text' : 'String' = "While mcp server is enabled, when mcp tool call is received, the engineering model go shall return mcp tool result from the module model files on disk at call time.";
        attribute 'notes' : 'String' = "The server fingerprints this module's model files at load: the manifest, every document it names, the requirements and design paths, and the lock and workspace files next to the manifest (engmod.lock.yml, engmod.lock.local.yml, engmod.work.yml). Before each tools/call it recomputes the fingerprint and reloads the model when it differs; a reload that fails is returned as the call error (-32001) instead of the previous model, and the next call retries. Documents of dependencies resolved from sibling checkouts or the module cache are not fingerprinted: after editing a sibling checkout the client re-sends initialize. Only regular files up to 16 MiB are hashed; a larger file is compared by size and modification time and a non-regular file by its mode.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-090";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-090' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall report each requirement without an upward trace.";
        attribute 'notes' : 'String' = "No model can know it is top level, because any system may later be used by one above (decision D22, ADR-EMG-028), so every requirement traces through a capability. A requirement traces upward iff (a) it realizes a local capability or assumption of use (or is named by a deprecated capability ref), (b) it is the parent of a reqLinks entry whose child is a dependency capability (alias::CAP-*), or (c) it is derived: true with derivedRationale. Stakeholder sourceRefs do not count, and no model is exempt (inferenceHints.topLevel does not exist; the strict schema rejects it). Otherwise requirement.no_parent is reported at the requirement with a message naming exactly these three options: an error with requirementLinks strict in every model; a warning with guided (the default) only in a model that publishes a contract item or has reqLinks; nothing in a guided standalone model. A realizes target that is a requirement, an imported alias::ID, an interface entry or unknown is requirement.realizes_unknown (error). requirementLinks other than strict or guided is model.invalid_requirement_links (error). The rule needs only the child repository.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-091";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-091' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall load assumptions of use as contract items with their realizing requirements.";
        attribute 'notes' : 'String' = "contract.assumptions entries have id (AOU-*), text and optional verificationCriteria. They are published in a publication architecture list like capabilities, imported as alias::AOU-* by a composing bundle, and count as contract items: a requirement may realize them and they have a fingerprint. Contract items are contract.provides entries of kind capability and the assumptions; interface entries are not link targets.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-092";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-092' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall report each requirement that realizes a published contract item without being published itself.";
        attribute 'notes' : 'String' = "contract.realizer_unpublished is an error at a requirement that realizes (through realizes or a deprecated ref) a contract item that a publication exposes, when that publication does not list the requirement under requirements. A using system computes the fingerprint from the published content alone, so every realizer must be visible to it.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-093";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-093' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall warn at each published capability without a realizing requirement.";
        attribute 'notes' : 'String' = "contract.unrealized (warning) names a published capability with no realizing requirement. contract.ref_deprecated (warning) is reported at each contract.provides ref; the ref still counts as a realizer for one release.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-094";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-094' {
        attribute 'text' : 'String' = "Where system composition is enabled, when model updated event is received, the engineering model go shall validate each requirement link against the manifest dependency pins.";
        attribute 'notes' : 'String' = "The optional document documents.reqLinks (model/req-links.yml) holds reqLinks entries {id, parent, child, evidence, pinned {version, fingerprint}, status claimed|accepted, rationale, reviewedBy, reviewedOn}, strictly decoded and checked by the CUE schema. A parent that is no local requirement, a link with neither parent nor evidence, or evidence that is no local ID is link.parent_unknown; a child that is not alias::ID of a declared dependency selected by a composition subsystem is link.child_unpublished; pinned.version other than the dependency version is link.version_mismatch (errors); status claimed is link.claimed_only (warning). A dependency version may be a Go-style pseudo-version.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-095";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-095' {
        attribute 'text' : 'String' = "The engineering model go shall compute a deterministic fingerprint of each contract item with its realizing requirements.";
        attribute 'notes' : 'String' = "reqlinks.Fingerprint is sha256:v1:<hex> (v1 the canonical-form version, decision D17) of compact JSON {\"item\":{id,kind,text,verificationCriteria},\"realizers\":[{id,text,notes,verificationCriteria,verificationMethods,criticality,status}...]} with realizers sorted by id, verificationMethods sorted, every string whitespace-collapsed and trimmed, case kept, no HTML escaping and no trailing newline (reqlinks.CanonicalItem). The text of a capability is its note. Requirement title, rationale, appliesTo, priority and tags are not covered: they name, explain, allocate or rank the requirement without changing what it obliges. A using system computes it from the published items and realizers (reqlinks.PublishedItems), so a change to a requirement that realizes no linked item does not change it.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-096";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-096' {
        attribute 'text' : 'String' = "Where system composition is enabled, the engineering model go shall accept a dependency version that is a release tag or a pseudo-version carrying a commit.";
        attribute 'notes' : 'String' = "The CUE #ExactVersion and ParseExactModuleVersion accept a canonical semantic version, including Go-style pseudo-versions v1.2.4-0.20261102103400-abcdef123456. A release only uses tags; engws release-check rejects pseudo-versions (REQ-EMG-103).";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-097";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-097' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall report each requirement link whose contract item changed or disappeared at the pinned dependency version.";
        attribute 'notes' : 'String' = "Composition evaluates each reqLinks entry against the resolved dependency (reqlinks.EvaluateLinks): unchanged, suspect (fingerprint differs; link.suspect error naming both fingerprints and the current item and realizing requirement texts), broken (the item no longer exists; link.broken error), child_unpublished (it exists but is not a published capability or assumption of use; link.child_unpublished error), version_mismatch or unresolved. The diagnostics join the composition diagnostics, so engdoc, engmod-check and model.diagnose report them; they do not stop the lock from being written. The before text is not part of the pinned dependency; engws review-links resolves it for a diff.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-098";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-098' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall report each dependency assumption of use without an evidence link and each system requirement satisfied by nothing.";
        attribute 'notes' : 'String' = "link.aou_unverified is reported for every published assumption of use of a composed dependency that no reqLinks entry with evidence names as child. requirement.unsatisfied is reported by the trace integrity check at a requirement of a model with composition subsystems that no code or test of the model links, no reqLinks entry names as parent, no allocation delegates and that is not derived. Both are errors with requirementLinks strict and warnings with guided where links apply (validate.RequirementLinkRules).";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-099";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-099' {
        attribute 'text' : 'String' = "While mcp server is enabled, when mcp tool call is received, the engineering model go shall describe requirement links in the model authoring contract.";
        attribute 'notes' : 'String' = "The authoring contract field rules describe realizes, contract.assumptions, the reqLinks document and the requirementLinks strictness, and fieldSets list contractAssumption, reqLink and reqLinkPin; the reqLinks document is listed when the manifest names it. Each composition.allocations entry is composition.allocation_deprecated (warning) and still materializes as before for one release.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-100";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-100' {
        attribute 'text' : 'String' = "When dependency bump is requested, the engineering model go shall report each requirement link of the bumping model as unchanged, suspect or broken at the new version.";
        attribute 'notes' : 'String' = "engws bump <repo> <alias> <version> [--allow-suspect] evaluates, after the pin, go.mod and lock update, every reqLinks entry of the consumer on the bumped alias that was pinned to the old version against the re-resolved composition (reqlinks.EvaluateLinks with the pin moved to the new version) and prints one line per link: unchanged, suspect (with the current item and realizer texts) or broken (also an item that is no longer a published contract item). Decision D8: bump moves pinned.version to the new version only for unchanged links (same fingerprint, nothing to review); suspect and broken links keep their old pin and fingerprint, and the fingerprint is never re-stamped. Without --allow-suspect a suspect or broken link aborts the bump with every file restored (engmod.yml, go.mod, go.sum, engmod.lock.yml, model/req-links.yml) and exit 1; with it the bump is written, exit 0, and the model check fails with link.version_mismatch on the kept pins until engws review-links accepts them. link.* diagnostics do not block lock regeneration.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-101";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-101' {
        attribute 'text' : 'String' = "When link review is requested, the engineering model go shall show the child item change of each requirement link that needs review.";
        attribute 'notes' : 'String' = "engws review-links [--repo DIR] [--json] lists every suspect, broken and version_mismatch link of the model with the parent requirement text and the canonical child item (item text, verification criteria and realizing requirements) at the current dependency pin (after) and at the link's pinned.version (before), the latter resolved through the same resolver (module cache or registry, lock check mode, nothing written); a version a workspace replacement or an offline registry cannot provide prints \"before: version vX not available offline\". Exit 0 when nothing needs review, 1 otherwise. --accept LNK-ID --by ACT-ID --rationale TEXT writes into the requirement link file, editing only the touched lines (yaml.v3 node positions, comments and order kept): pinned.fingerprint (current), pinned.version (the dependency pin), reviewedBy, reviewedOn (today), rationale and status accepted; it refuses an actor the model does not define, an unknown link and a broken link (or one whose item cannot be resolved). The accept composes with the workspace off (decision D9): the fingerprint written is that of the published dependency at its pin, never of a workspace checkout; a dependency that does not resolve that way leaves the link unresolved and the accept is refused with nothing written. A file it rewrites keeps its permission bits.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-102";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-102' {
        attribute 'text' : 'String' = "When contract diff is requested, the engineering model go shall classify each contract item change between two git refs with its semantic version impact.";
        attribute 'notes' : 'String' = "engws contract-diff <from-ref> <to-ref> [--repo DIR] [--json] reads engmod.yml and every document it names at each ref (git show ref:path, relative to the repository prefix; a ref that is empty or starts with - is refused), loads them with the model loader from a temporary directory (0700, files 0600), and compares the published contract items (reqlinks.PublishedItems): added, removed, changed (text, realizers) or unchanged. Impact: capability removed, capability text changed (printed as \"text changed: treated as major\"), assumption added, assumption text or verificationCriteria changed, or an item kind changed are major; assumption removed, capability added and changes of realizing requirements only are minor; anything else is patch. Exit 0, or 2 on a usage or IO error.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-103";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-103' {
        attribute 'text' : 'String' = "When release check is requested, the engineering model go shall reject a release that pins an unreleased dependency or whose tag bump is smaller than its contract diff requires.";
        attribute 'notes' : 'String' = "engws release-check [--repo DIR] [--tag vX.Y.Z] reports dependency.unreleased_pin (error) for every engmod.yml dependency or engmod.lock.yml version that is a pre-release or pseudo-version, and contract.semver_too_small (error) when the bump of the tag (--tag, else the highest release tag at HEAD) over the previous release tag (the highest release semver tag below it among the tags merged into the released commit, git tag --merged: the nearest ancestor release) is smaller than contract-diff(previous, the tag if it exists, else HEAD) requires. In v0.x (previous and tag major 0) a minor bump is major-capable and a patch bump minor-capable, so a breaking change needs a minor bump. Without a tag or a previous release tag the semver gate is skipped with a note. Exit 0 ok, 1 findings, 2 usage or IO error.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-104";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-104' {
        attribute 'text' : 'String' = "When link migration is requested, the engineering model go shall convert each deprecated allocation or contract ref into the requirement link model.";
        attribute 'notes' : 'String' = "engws migrate-links [--repo DIR] [--dry-run] writes one reqLinks entry per composition.allocations entry (id LNK-<requirement without REQ->-<ALIAS> upper case, made unique with -2, -3 ...; parent the requirement; child the target; pinned.version the dependency pin; fingerprint the current one from the resolved dependency, or \"\" when the dependency cannot be resolved, which the output names and the check reports as link.suspect until review; status claimed; rationale from the allocation), creates or extends the requirement link file, adds documents.reqLinks: model/req-links.yml to engmod.yml when missing, removes the converted allocations (D13: an allocation whose resolvable target is an interface or other contract entry that is not a published capability or assumption of use is kept in place, keeps reporting composition.allocation_deprecated and is printed as kept allocation REQ → alias::ID with the reason), appends each capability contract.provides[].ref to realizes of the referenced requirement (deduplicated) and drops the ref; a ref on a non-capability entry or naming an unknown requirement is kept and reported. Every edit keeps the other lines of the files; every file to write is snapshot first and, when one write fails, the files already written are restored and the ones it created removed. It prints the requirements that then report requirement.no_parent (a requirement that became the parent of a capability link is traced, D22). A second run changes nothing; --dry-run writes nothing.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-105";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-105' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall report the parents and children of each requirement and roll each linked system requirement up from its realizing child requirements.";
        attribute 'notes' : 'String' = "trace.Matrix rows carry parents (the contract items a requirement realizes through realizes or a deprecated ref, qualified alias::ID on a published requirement row, none for a requirement traced only by its links or derivation; stakeholder sourceRefs are not parents, D22) and children (for a requirement that is the parent of reqLinks entries: each linked child item alias::CAP-* followed by its realizing requirements alias::REQ-* at the resolved version; none for a child requirement), links (id, child, evaluated state, review status) and satisfiedBy. A requirement with no code or test of its own whose every link is accepted, evaluates unchanged and resolves to an item with realizing requirements that are all implemented (or all test linked) in their own repository trace is implemented (or test linked) with satisfiedBy links (decision D18: a claimed, suspect, broken, version-mismatched or outdated link gives satisfiedBy links-unreviewed and does not count), linksImplemented and linksTestLinked, is counted in the summary implemented and testLinked, and is no orphan. The summary counts linksSuspect (also a suspect item behind a stale pin), linksBroken, linksClaimed and unparented (local requirements that fail the upward-trace rule of REQ-EMG-090 where the requirement link rules apply: no realizes, not the parent of a capability link, not derived with derivedRationale). MCP trace.matrix returns the four counts, also with summaryOnly, and parents, children, links and satisfiedBy per row.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-106";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-106' {
        attribute 'text' : 'String' = "While mcp server is enabled, when mcp tool call is received, the engineering model go shall return the requirement links of the loaded model with their state and the trace of each requirement through them.";
        attribute 'notes' : 'String' = "links.list returns every reqLinks entry with id, parent, child, evidence, state and content (reqlinks.EvaluateLinks at the composed pins), review status, pinnedVersion, dependencyVersion, recordedFingerprint, currentFingerprint, reviewedBy and reviewedOn. links.suspect returns the links that need review (suspect, broken, version_mismatch) as engws review-links --json does: parent text, the child item and its realizing requirements after (at the dependency pin) and before (at the pinned version, workspace off, offline: else beforeNote). links.review {linkId, by, rationale?} writes nothing (the MCP server has no writing tool), composes like engws review-links --accept with the workspace off and takes reviewedOn from the server clock: it refuses an actor the model does not define, an unknown and a broken link like engws review-links --accept, and returns the command engws review-links --accept LNK --by ACT --rationale '...', the link file path relative to the model and the line patch the command writes (pinned version and fingerprint, status, rationale, reviewedBy, reviewedOn). requirements.parents {requirementId} returns realizes, parents (the realized items; sourceRefs are no upward trace, D22), sourceRefs, derived, derivedRationale, the links the requirement is parent of with state, status and the linked item (id, kind, text, qualified realizers) and children (each linked item then its realizers). requirements.get adds parents, links and children. Every tool reads only the loaded model and the dependencies it composes, with the lock checked and the module cache reused.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-107";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-107' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall render the requirement link tree of a using model and the contract items of a child model without disclosing the users of the child.";
        attribute 'notes' : 'String' = "A model with reqLinks gets an AsciiDoc chapter \"Requirement Links\": for each requirement a list item, then each link it is the parent of with its state (unchanged, or marked *SUSPECT*, *BROKEN*, *VERSION MISMATCH*, *UNPUBLISHED*) and review status (accepted, or *CLAIMED*), the linked child item alias::ID with its text, and the realizing requirements alias::REQ with their trace status in the dependency (trace.DependencyRequirementStatus), or derived with its rationale, or no requirement link; and \"Assumptions of Use\" with each dependency assumption and the links whose evidence covers it (*UNVERIFIED* otherwise). A model that publishes contract items gets \"Contract Items\": each published capability and assumption of use with its text, verification criteria, realizing published requirements and fingerprint. Only when the caller opts in (AsciiDocOptions.UsedBy, engdoc --used-by; off by default, also when a workspace is found), the render resolves an active workspace (engmod.work.yml found, the workspace not off) and a sibling replacement checkout has reqLinks onto these items through a dependency on this module path, a \"Used By\" section (item, using model id, link id, parent requirement, status) is added, computed at render time by composition.WorkspaceLinkUsers and never written into the model; a document rendered with it must not be committed into the child (D16). By default a child names no using system in AsciiDoc, Gemara or MCP output. link.* findings and the strict-mode trace findings requirement.no_parent, requirement.unsatisfied and contract.realizer_unpublished are reported but do not stop the render (D15).";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-108";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-108' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall ignore each workspace file and sibling checkout that another user could have written or that leaves the workspace.";
        attribute 'notes' : 'String' = "The upward search for engmod.work.yml (FindWorkspaceWith with an empty mode and ENGMOD_WORK unset) refuses a world-writable file and, where the platform has file owners, a file owned by neither the current user nor root, with an error naming ENGMOD_WORK=<path> as the way to use it anyway; an explicit path is used as given. The Used By scan (composition.WorkspaceLinkUsers) reads only replacement directories inside the workspace directory (composition.PathWithin) whose manifest names no path outside its own directory (the dependency confinement, composition.dependency_path_escape), so a crafted workspace or sibling manifest cannot make a render read files elsewhere. The upward search does not stop at a git top-level: a workspace sits above sibling repositories by design.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-109";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-109' {
        attribute 'text' : 'String' = "Where system composition is enabled, when generation run is requested, the engineering model go shall report each requirement link whose recorded fingerprint uses another canonical form than the engineering model go writes.";
        attribute 'notes' : 'String' = "A fingerprint is sha256:v<N>:<hex>, N the canonical-form version (reqlinks.CanonicalVersion, 1). Decision D17: any change to the canonical form is a new version. A recorded fingerprint of another version, or the unversioned sha256:<hex> of the v0.8.0 pre-releases, is not compared: the link is fingerprint_outdated and the composed check reports link.fingerprint_outdated (error, distinct from link.suspect) with the message to check that nothing in the item changed (engws review-links) and re-stamp it with engws review-links --accept, or use an engine that migrates the form. The CUE schema accepts sha256:(vN:)?<64 hex>; engws review-links lists the link and --accept re-stamps it.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-110";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-110' {
        attribute 'text' : 'String' = "When generation run is requested, the engineering model go shall render requirement link and contract item text without AsciiDoc passthrough.";
        attribute 'notes' : 'String' = "Text in the Requirement Links, Assumptions of Use, Contract Items and Used By sections comes partly from dependency models (another repository). Every such string is whitespace-collapsed and its passthrough forms (+++...+++, ++...++, $$...$$, pass:[...] and other pass macros) are broken with {empty}, so it renders literally and cannot emit raw HTML into the document.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-111";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-111' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall report each accepted requirement link without a resolving reviewer and review date.";
        attribute 'notes' : 'String' = "A reqLinks entry with status accepted and no reviewedBy or no reviewedOn is link.review_unrecorded (error); a reviewedBy that is not an actor of the model and a reviewedOn that is not a YYYY-MM-DD calendar date are link.review_unrecorded too, at that field. A claimed link may carry neither. engws review-links --accept writes both; the check shows only that a modeled role and a date are recorded, review of the diff shows that the role gave it.";
    }

    @EngineeringElement {
        sourceId = "REQ-EMG-112";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-EMG-112' {
        attribute 'text' : 'String' = "When model updated event is received, the engineering model go shall report each requirement link to a capability without a parent requirement.";
        attribute 'notes' : 'String' = "A link whose child is alias::CAP-* says that the capability satisfies a local requirement, so it needs parent: link.parent_missing (error) otherwise, also when it carries evidence. A link to an assumption of use (alias::AOU-*) may have evidence instead of a parent.";
    }

    @EngineeringElement {
        sourceId = "RISK-MCP-PATH-ESCAPE";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-MCP-PATH-ESCAPE';

    @EngineeringElement {
        sourceId = "RISK-STALE-ARTIFACTS";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-STALE-ARTIFACTS';

    @EngineeringElement {
        sourceId = "RISK-TRACEABILITY-DRIFT";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-TRACEABILITY-DRIFT';

    @EngineeringElement {
        sourceId = "STATE-ARTIFACTS-FRESH";
        conceptKind = "state_definition";
    }
    state def 'STATE-ARTIFACTS-FRESH';

    @EngineeringElement {
        sourceId = "STATE-LINK-ACCEPTED";
        conceptKind = "state_definition";
    }
    state def 'STATE-LINK-ACCEPTED';

    @EngineeringElement {
        sourceId = "STATE-LINK-BROKEN";
        conceptKind = "state_definition";
    }
    state def 'STATE-LINK-BROKEN';

    @EngineeringElement {
        sourceId = "STATE-LINK-CLAIMED";
        conceptKind = "state_definition";
    }
    state def 'STATE-LINK-CLAIMED';

    @EngineeringElement {
        sourceId = "STATE-LINK-SUSPECT";
        conceptKind = "state_definition";
    }
    state def 'STATE-LINK-SUSPECT';

    @EngineeringElement {
        sourceId = "STATE-MODEL-INVALID";
        conceptKind = "state_definition";
    }
    state def 'STATE-MODEL-INVALID';

    @EngineeringElement {
        sourceId = "STATE-MODEL-VALID";
        conceptKind = "state_definition";
    }
    state def 'STATE-MODEL-VALID';

    @EngineeringElement {
        sourceId = "SYS-ENGINEERING-MODEL-GO";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-ENGINEERING-MODEL-GO';

    @EngineeringElement {
        sourceId = "TB-CHILD-USING-SYSTEM";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-CHILD-USING-SYSTEM';

    @EngineeringElement {
        sourceId = "TB-EXTERNAL-VALIDATION-TOOLS";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-EXTERNAL-VALIDATION-TOOLS';

    @EngineeringElement {
        sourceId = "TB-REPO-WORKSPACE";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-REPO-WORKSPACE';

    @EngineeringElement {
        sourceId = "TM-ACCEPTANCE-RECORD-CHECK";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-ACCEPTANCE-RECORD-CHECK';

    @EngineeringElement {
        sourceId = "TM-ARTIFACT-SCAN-BOUNDS";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-ARTIFACT-SCAN-BOUNDS';

    @EngineeringElement {
        sourceId = "TM-CHILD-USER-NONDISCLOSURE";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-CHILD-USER-NONDISCLOSURE';

    @EngineeringElement {
        sourceId = "TM-FINGERPRINT-COVERAGE";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-FINGERPRINT-COVERAGE';

    @EngineeringElement {
        sourceId = "TM-GIT-REF-VALIDATION";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-GIT-REF-VALIDATION';

    @EngineeringElement {
        sourceId = "TM-GO-WORK-INPUT-VALIDATION";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-GO-WORK-INPUT-VALIDATION';

    @EngineeringElement {
        sourceId = "TM-LINK-REVIEW-RECORD";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-LINK-REVIEW-RECORD';

    @EngineeringElement {
        sourceId = "TM-MCP-PATH-BOUNDARY";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-MCP-PATH-BOUNDARY';

    @EngineeringElement {
        sourceId = "TM-REPO-LOCAL-WRITES";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-REPO-LOCAL-WRITES';

    @EngineeringElement {
        sourceId = "TM-TRACE-COVERAGE";
        conceptKind = "engineering_extension";
    }
    attribute 'TM-TRACE-COVERAGE';

    @EngineeringElement {
        sourceId = "TS-ARTIFACT-PARSE-ABUSE";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-ARTIFACT-PARSE-ABUSE';

    @EngineeringElement {
        sourceId = "TS-CHILD-USER-DISCLOSURE";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-CHILD-USER-DISCLOSURE';

    @EngineeringElement {
        sourceId = "TS-FALSE-LINK-ACCEPTANCE";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-FALSE-LINK-ACCEPTANCE';

    @EngineeringElement {
        sourceId = "TS-FALSE-RISK-ACCEPTANCE";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-FALSE-RISK-ACCEPTANCE';

    @EngineeringElement {
        sourceId = "TS-GIT-REF-OPTION-INJECTION";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-GIT-REF-OPTION-INJECTION';

    @EngineeringElement {
        sourceId = "TS-GO-WORK-INJECTION";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-GO-WORK-INJECTION';

    @EngineeringElement {
        sourceId = "TS-HIDDEN-CHILD-CHANGE";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-HIDDEN-CHILD-CHANGE';

    @EngineeringElement {
        sourceId = "TS-LINK-WRITE-OUTSIDE-REPO";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-LINK-WRITE-OUTSIDE-REPO';

    @EngineeringElement {
        sourceId = "TS-MCP-PATH-TRAVERSAL";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-MCP-PATH-TRAVERSAL';

    @EngineeringElement {
        sourceId = "TS-TRACEABILITY-DRIFT";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-TRACEABILITY-DRIFT';

    @EngineeringElement {
        sourceId = "VER-ACCEPTANCE-RECORD-CHECK";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-ACCEPTANCE-RECORD-CHECK' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-ARTIFACT-SCAN-BOUNDS";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-ARTIFACT-SCAN-BOUNDS' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-CHILD-USER-NONDISCLOSURE";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-CHILD-USER-NONDISCLOSURE' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-FINGERPRINT-COVERAGE";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-FINGERPRINT-COVERAGE' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-GIT-REF-VALIDATION";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-GIT-REF-VALIDATION' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-GO-WORK-INPUT-VALIDATION";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-GO-WORK-INPUT-VALIDATION' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-LINK-REVIEW-RECORD";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-LINK-REVIEW-RECORD' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-MCP-PATH-BOUNDARY";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-MCP-PATH-BOUNDARY' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-REPO-LOCAL-WRITES";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-REPO-LOCAL-WRITES' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VER-TRACEABILITY-COVERAGE";
        conceptKind = "verification_case_usage";
    }
    verification 'VER-TRACEABILITY-COVERAGE' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "VIEW-ARCHITECTURE-INTENT";
        conceptKind = "view_usage";
    }
    view 'VIEW-ARCHITECTURE-INTENT' : EngineeringView;

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
        sourceId = "check-acceptance";
        conceptKind = "action_usage";
    }
    action 'check-acceptance' : EngineeringAction;

    @EngineeringElement {
        sourceId = "edit-model";
        conceptKind = "action_usage";
    }
    action 'edit-model' : EngineeringAction {
        out item 'output_architecture_update' : EngineeringItem;
    }

    @EngineeringElement {
        sourceId = "evaluate-bumped-links";
        conceptKind = "action_usage";
    }
    action 'evaluate-bumped-links' : EngineeringAction;

    @EngineeringElement {
        sourceId = "fail-validation";
        conceptKind = "action_usage";
    }
    action 'fail-validation' : EngineeringAction;

    @EngineeringElement {
        sourceId = "generate-artifacts";
        conceptKind = "action_usage";
    }
    action 'generate-artifacts' : EngineeringAction {
        out item 'output_generated_artifacts' : EngineeringItem;
    }

    @EngineeringElement {
        sourceId = "load-requirements-delta";
        conceptKind = "action_usage";
    }
    action 'load-requirements-delta' : EngineeringAction {
        in item 'input_DO_REQUIREMENTS_DELTA' : 'DO-REQUIREMENTS-DELTA';
    }

    @EngineeringElement {
        sourceId = "merge-requirements-candidate";
        conceptKind = "action_usage";
    }
    action 'merge-requirements-candidate' : EngineeringAction {
        in item 'input_DO_REQUIREMENTS_DELTA' : 'DO-REQUIREMENTS-DELTA';
        in item 'input_DO_REQUIREMENTS_DOCUMENT' : 'DO-REQUIREMENTS-DOCUMENT';
        out item 'output_DO_REQUIREMENTS_DIFF' : 'DO-REQUIREMENTS-DIFF';
    }

    @EngineeringElement {
        sourceId = "pin-version";
        conceptKind = "action_usage";
    }
    action 'pin-version' : EngineeringAction;

    @EngineeringElement {
        sourceId = "publish-artifacts";
        conceptKind = "action_usage";
    }
    action 'publish-artifacts' : EngineeringAction;

    @EngineeringElement {
        sourceId = "record-acceptance";
        conceptKind = "action_usage";
    }
    action 'record-acceptance' : EngineeringAction;

    @EngineeringElement {
        sourceId = "refuse-acceptance";
        conceptKind = "action_usage";
    }
    action 'refuse-acceptance' : EngineeringAction;

    @EngineeringElement {
        sourceId = "regenerate-lock";
        conceptKind = "action_usage";
    }
    action 'regenerate-lock' : EngineeringAction;

    @EngineeringElement {
        sourceId = "reject-requirements-delta";
        conceptKind = "action_usage";
    }
    action 'reject-requirements-delta' : EngineeringAction {
        out item 'output_DO_REQUIREMENTS_DIFF' : 'DO-REQUIREMENTS-DIFF';
    }

    @EngineeringElement {
        sourceId = "repin-unchanged-links";
        conceptKind = "action_usage";
    }
    action 'repin-unchanged-links' : EngineeringAction;

    @EngineeringElement {
        sourceId = "replace-requirements";
        conceptKind = "action_usage";
    }
    action 'replace-requirements' : EngineeringAction {
        in item 'input_DO_REQUIREMENTS_DOCUMENT' : 'DO-REQUIREMENTS-DOCUMENT';
    }

    @EngineeringElement {
        sourceId = "restore-bump-files";
        conceptKind = "action_usage";
    }
    action 'restore-bump-files' : EngineeringAction;

    @EngineeringElement {
        sourceId = "show-links";
        conceptKind = "action_usage";
    }
    action 'show-links' : EngineeringAction;

    @EngineeringElement {
        sourceId = "validate-model";
        conceptKind = "action_usage";
    }
    action 'validate-model' : EngineeringAction {
        in item 'input_architecture_update' : EngineeringItem;
    }

    @EngineeringElement {
        sourceId = "validate-requirements-candidate";
        conceptKind = "action_usage";
    }
    action 'validate-requirements-candidate' : EngineeringAction {
        in item 'input_DO_REQUIREMENTS_DOCUMENT' : 'DO-REQUIREMENTS-DOCUMENT';
        in item 'input_DO_ARCHITECTURE_MODEL' : 'DO-ARCHITECTURE-MODEL';
    }

    @EngineeringRelationship {
        sourceId = "REL-0093A8DE75632A1B";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-0093A8DE75632A1B-context' : EngineeringState {
        state 'STATE-LINK-ACCEPTED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-0093A8DE75632A1B' first 'STATE-LINK-ACCEPTED' accept 'EVT-CONTRACT-DIFF-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-016B944849F917F9";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-016B944849F917F9' from 'FU-STRUCTURIZR-EXPORTER' to 'REF-STRUCTURIZR-VALIDATOR';

    @EngineeringRelationship {
        sourceId = "REL-01A907C5186262EC";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-01A907C5186262EC' : EngineeringVerification {
        subject 'VER-LINK-REVIEW-RECORD' : EngineeringItem;
        objective 'CTRL-LINK-REVIEW-RECORD' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-04F4A20F9A87A948";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-04F4A20F9A87A948' from 'AV-PATH-TRAVERSAL-IN-MCP' to 'CTRL-STRICT-MCP-INPUT-SCHEMA';

    @EngineeringRelationship {
        sourceId = "REL-080A42488B0FD5AB";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-080A42488B0FD5AB' from 'FU-TRLC-EXPORTER' to 'REF-TRLC-TOOLCHAIN';

    @EngineeringRelationship {
        sourceId = "REL-0893DE78A51F8BA1";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-0893DE78A51F8BA1' first 'show-links' then 'check-acceptance';

    @EngineeringRelationship {
        sourceId = "REL-09DA914374D48456";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-09DA914374D48456' from 'ACT-CI-PIPELINE' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-0BC01C9171549ED5";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-0BC01C9171549ED5' first 'regenerate-lock' then 'evaluate-bumped-links';

    @EngineeringRelationship {
        sourceId = "REL-0D4880EAA06B6537";
        conceptKind = "transition";
        relationshipKind = "transitions_to";
    }
    state 'REL-0D4880EAA06B6537-context' : EngineeringState {
        action 'STATE-LINK-CLAIMED' : EngineeringAction;
        action 'STATE-LINK-ACCEPTED' : EngineeringAction;
        transition 'REL-0D4880EAA06B6537' first 'STATE-LINK-CLAIMED' then 'STATE-LINK-ACCEPTED';
    }

    @EngineeringRelationship {
        sourceId = "REL-0E9AC4B59A88F112";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-0E9AC4B59A88F112' from 'fail-validation' to 'FU-VALIDATION-ENGINE';

    @EngineeringRelationship {
        sourceId = "REL-110FD6978371D25B";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-110FD6978371D25B' from 'FU-THREAT-EXPORTER' to 'REF-OPEN-OTM-SCHEMA';

    @EngineeringRelationship {
        sourceId = "REL-11D1F008C1C87E20";
        conceptKind = "transition";
        relationshipKind = "transitions_to";
    }
    state 'REL-11D1F008C1C87E20-context' : EngineeringState {
        action 'STATE-MODEL-INVALID' : EngineeringAction;
        action 'STATE-MODEL-VALID' : EngineeringAction;
        transition 'REL-11D1F008C1C87E20' first 'STATE-MODEL-INVALID' then 'STATE-MODEL-VALID';
    }

    @EngineeringRelationship {
        sourceId = "REL-11EBC1A0720CA807";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-11EBC1A0720CA807-context' : EngineeringState {
        state 'STATE-LINK-CLAIMED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-11EBC1A0720CA807' first 'STATE-LINK-CLAIMED' accept 'EVT-LINK-MIGRATION-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-14A8527C3C1658DF";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-14A8527C3C1658DF-context' {
        part 'ACT-IMPLEMENTATION-ENGINEER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-MODEL-CHANGE' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-14A8527C3C1658DF' from 'ACT-IMPLEMENTATION-ENGINEER'.pOut to 'FU-MODEL-CHANGE'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-19523EB810526624";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-19523EB810526624-context' {
        part 'ACT-IMPLEMENTATION-ENGINEER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-CLI-ORCHESTRATION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-19523EB810526624' from 'ACT-IMPLEMENTATION-ENGINEER'.pOut to 'FU-CLI-ORCHESTRATION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-1A999986F017F7E0";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-1A999986F017F7E0' : EngineeringVerification {
        subject 'VER-FINGERPRINT-COVERAGE' : EngineeringItem;
        objective 'CTRL-FINGERPRINT-COVERAGE' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-1B448352F7DFCECA";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-1B448352F7DFCECA' from 'refuse-acceptance' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-1D6B10631D0F5925";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-1D6B10631D0F5925' from 'FU-MODEL-CHANGE' to 'FU-VALIDATION-ENGINE';

    @EngineeringRelationship {
        sourceId = "REL-1DF7B8675269FA1B";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-1DF7B8675269FA1B' from 'FU-MCP-SERVER' to 'TB-REPO-WORKSPACE';

    @EngineeringRelationship {
        sourceId = "REL-1F66CE721C2ED103";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-1F66CE721C2ED103' from 'record-acceptance' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-2180C452E7CCA478";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-2180C452E7CCA478-context' {
        part 'FU-NAF-EXPORTER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-NAF-V4-ARCHITECTURE' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-2180C452E7CCA478' from 'FU-NAF-EXPORTER'.pOut to 'DO-NAF-V4-ARCHITECTURE'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-258C0AF7BCCCE854";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-258C0AF7BCCCE854' from 'FU-LOBSTER-EXPORTER' to 'TB-EXTERNAL-VALIDATION-TOOLS';

    @EngineeringRelationship {
        sourceId = "REL-25F0590AC1010192";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-25F0590AC1010192' first 'edit-model' then 'validate-model';

    @EngineeringRelationship {
        sourceId = "REL-2A230B0018684311";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-2A230B0018684311' from 'FU-LOBSTER-EXPORTER' to 'REF-LOBSTER-TOOLCHAIN';

    @EngineeringRelationship {
        sourceId = "REL-2A5510B6345D819F";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-2A5510B6345D819F-context' {
        part 'FU-CLI-ORCHESTRATION' : EngineeringPart;
        part 'DEP-CI-PIPELINE' : EngineeringPart;
        allocation 'REL-2A5510B6345D819F' allocate 'FU-CLI-ORCHESTRATION' to 'DEP-CI-PIPELINE';
    }

    @EngineeringRelationship {
        sourceId = "REL-2AEB188DC97B1888";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-2AEB188DC97B1888' : EngineeringVerification {
        subject 'VER-CHILD-USER-NONDISCLOSURE' : EngineeringItem;
        objective 'CTRL-CHILD-USER-NONDISCLOSURE' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-2EAD296790C31C25";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-2EAD296790C31C25' from 'load-requirements-delta' to 'FU-MODEL-CHANGE';

    @EngineeringRelationship {
        sourceId = "REL-2FA99E391022B5E4";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-2FA99E391022B5E4' first 'pin-version' then 'restore-bump-files';

    @EngineeringRelationship {
        sourceId = "REL-2FBD45E16902772F";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-2FBD45E16902772F' from 'FU-MODEL-CHANGE' to 'FU-MODEL-LOADER';

    @EngineeringRelationship {
        sourceId = "REL-30E5FCA45681E4F3";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-30E5FCA45681E4F3' from 'FU-VALIDATION-ENGINE' to 'REF-GO-TOOLCHAIN';

    @EngineeringRelationship {
        sourceId = "REL-316A24455E4139E3";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-316A24455E4139E3-context' : EngineeringState {
        state 'STATE-LINK-ACCEPTED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-316A24455E4139E3' first 'STATE-LINK-ACCEPTED' accept 'EVT-RELEASE-CHECK-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-33A257E5FF509B55";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-33A257E5FF509B55-context' {
        part 'FU-MODEL-CHANGE' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-REQUIREMENTS-DELTA' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-33A257E5FF509B55' from 'FU-MODEL-CHANGE'.pOut to 'DO-REQUIREMENTS-DELTA'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-3605D7D6ACD767E9";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-3605D7D6ACD767E9' from 'regenerate-lock' to 'FU-SYSTEM-COMPOSITION';

    @EngineeringRelationship {
        sourceId = "REL-368CF829865A81D5";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-368CF829865A81D5' from 'FU-SYSTEM-COMPOSITION' to 'REF-CUE-OCI-MODULE-REGISTRY';

    @EngineeringRelationship {
        sourceId = "REL-38D2B6208B0DBBA9";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-38D2B6208B0DBBA9' from 'pin-version' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-3A30D7C94E6939CE";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-3A30D7C94E6939CE' first 'record-acceptance' then 'refuse-acceptance';

    @EngineeringRelationship {
        sourceId = "REL-3A9F16464C64A59F";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-3A9F16464C64A59F' : EngineeringVerification {
        subject 'VER-REPO-LOCAL-WRITES' : EngineeringItem;
        objective 'CTRL-REPO-LOCAL-WRITES' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-3E33EA7E80D13835";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-3E33EA7E80D13835' first 'validate-requirements-candidate' then 'replace-requirements';

    @EngineeringRelationship {
        sourceId = "REL-40C4FE61C6053FE7";
        conceptKind = "transition";
        relationshipKind = "transitions_to";
    }
    state 'REL-40C4FE61C6053FE7-context' : EngineeringState {
        action 'STATE-LINK-ACCEPTED' : EngineeringAction;
        action 'STATE-LINK-SUSPECT' : EngineeringAction;
        transition 'REL-40C4FE61C6053FE7' first 'STATE-LINK-ACCEPTED' then 'STATE-LINK-SUSPECT';
    }

    @EngineeringRelationship {
        sourceId = "REL-41611DCF8363B2A8";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-41611DCF8363B2A8' from 'check-acceptance' to 'FU-ALLOCATION-TRACE';

    @EngineeringRelationship {
        sourceId = "REL-421CC0B05DC0502A";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-421CC0B05DC0502A-context' {
        part 'FU-MCP-SERVER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-MODEL-AUTHORING-CONTRACT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-421CC0B05DC0502A' from 'FU-MCP-SERVER'.pOut to 'DO-MODEL-AUTHORING-CONTRACT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-43718C28B6D7F647";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-43718C28B6D7F647-context' {
        part 'FU-MODEL-CHANGE' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-REQUIREMENTS-DIFF' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-43718C28B6D7F647' from 'FU-MODEL-CHANGE'.pOut to 'DO-REQUIREMENTS-DIFF'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-44F1CC1DBF28232F";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-44F1CC1DBF28232F' : EngineeringVerification {
        subject 'VER-ARTIFACT-SCAN-BOUNDS' : EngineeringItem;
        objective 'CTRL-ARTIFACT-SCAN-BOUNDS' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-4672CF188E6130EB";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-4672CF188E6130EB' first 'generate-artifacts' then 'publish-artifacts';

    @EngineeringRelationship {
        sourceId = "REL-48089B20C8B08D3E";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-48089B20C8B08D3E' first 'validate-model' then 'generate-artifacts';

    @EngineeringRelationship {
        sourceId = "REL-49A4B6D2D8B373AD";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-49A4B6D2D8B373AD' first 'pin-version' then 'regenerate-lock';

    @EngineeringRelationship {
        sourceId = "REL-4B995420ADBC7581";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-4B995420ADBC7581-context' {
        part 'FU-SYSML-EXPORTER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-SYSML-V2-METAMODEL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-4B995420ADBC7581' from 'FU-SYSML-EXPORTER'.pOut to 'DO-SYSML-V2-METAMODEL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-4E21742D2FB84467";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-4E21742D2FB84467' first 'regenerate-lock' then 'restore-bump-files';

    @EngineeringRelationship {
        sourceId = "REL-4E93BAD6FCDAC1E2";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-4E93BAD6FCDAC1E2' first 'evaluate-bumped-links' then 'restore-bump-files';

    @EngineeringRelationship {
        sourceId = "REL-4F1A97CB3EA472DC";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-4F1A97CB3EA472DC-context' {
        part 'FU-SYSML-EXPORTER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-SYSML-V2-MODEL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-4F1A97CB3EA472DC' from 'FU-SYSML-EXPORTER'.pOut to 'DO-SYSML-V2-MODEL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-4FF77277CA72D599";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-4FF77277CA72D599' from 'FU-CODEMAP-INFERENCE' to 'TB-REPO-WORKSPACE';

    @EngineeringRelationship {
        sourceId = "REL-51B543A60F0122B0";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-51B543A60F0122B0-context' {
        part 'ACT-ARCHITECTURE-AUTHOR' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-CLI-ORCHESTRATION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-51B543A60F0122B0' from 'ACT-ARCHITECTURE-AUTHOR'.pOut to 'FU-CLI-ORCHESTRATION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-52FC381077B14D5A";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-52FC381077B14D5A' first 'validate-requirements-candidate' then 'reject-requirements-delta';

    @EngineeringRelationship {
        sourceId = "REL-533FBA1058B9E95C";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-533FBA1058B9E95C' first 'check-acceptance' then 'record-acceptance';

    @EngineeringRelationship {
        sourceId = "REL-53DF7944D31ADC40";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-53DF7944D31ADC40' from 'FU-NAF-EXPORTER' to 'REF-NAF-V4-1-SPECIFICATION';

    @EngineeringRelationship {
        sourceId = "REL-56D94B0E429D076D";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-56D94B0E429D076D' : EngineeringVerification {
        subject 'VER-GIT-REF-VALIDATION' : EngineeringItem;
        objective 'CTRL-GIT-REF-VALIDATION' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-582D7DDAB1367C4D";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-582D7DDAB1367C4D' from 'FU-SYSML-EXPORTER' to 'REF-SYSML-V2-TOOLCHAIN';

    @EngineeringRelationship {
        sourceId = "REL-5AB925CA4BCA07AA";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-5AB925CA4BCA07AA' from 'publish-artifacts' to 'FU-ASCIIDOC-GENERATOR';

    @EngineeringRelationship {
        sourceId = "REL-5C9C6EE212E4FD1D";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-5C9C6EE212E4FD1D' from 'AV-TRACEABILITY-GAP-DRIFT' to 'CTRL-TRACEABILITY-COVERAGE';

    @EngineeringRelationship {
        sourceId = "REL-5FFB43CE4B5786F2";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-5FFB43CE4B5786F2-context' : EngineeringState {
        state 'STATE-LINK-BROKEN' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-5FFB43CE4B5786F2' first 'STATE-LINK-BROKEN' accept 'EVT-LINK-REVIEW-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-6088122D54F9887C";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-6088122D54F9887C' from 'AV-PATH-TRAVERSAL-IN-MCP' to 'FU-MCP-SERVER';

    @EngineeringRelationship {
        sourceId = "REL-615C9FA64AC3816A";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-615C9FA64AC3816A' from 'ACT-IMPLEMENTATION-ENGINEER' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-66C2DF83054D88C8";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-66C2DF83054D88C8' from 'ACT-COMPLIANCE-ENGINEER' to 'FU-OSCAL-EXPORTER';

    @EngineeringRelationship {
        sourceId = "REL-67DABE324CA82BF6";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-67DABE324CA82BF6-context' {
        part 'FU-MODEL-LOADER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-ARCHITECTURE-MODEL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-67DABE324CA82BF6' from 'FU-MODEL-LOADER'.pOut to 'DO-ARCHITECTURE-MODEL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-68040CE5FF37711F";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-68040CE5FF37711F' from 'generate-artifacts' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-6B0FD95359778099";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-6B0FD95359778099' from 'FU-CLI-ORCHESTRATION' to 'FU-ASCIIDOC-GENERATOR';

    @EngineeringRelationship {
        sourceId = "REL-6E736D47074A3CEA";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-6E736D47074A3CEA' from 'FU-TRLC-EXPORTER' to 'TB-EXTERNAL-VALIDATION-TOOLS';

    @EngineeringRelationship {
        sourceId = "REL-7AA33DA291EB7A90";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-7AA33DA291EB7A90' first 'check-acceptance' then 'refuse-acceptance';

    @EngineeringRelationship {
        sourceId = "REL-7C6B59DFA5F97922";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-7C6B59DFA5F97922' from 'FU-CLI-ORCHESTRATION' to 'FU-VALIDATION-ENGINE';

    @EngineeringRelationship {
        sourceId = "REL-7CF609209E449D9B";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-7CF609209E449D9B-context' {
        part 'FU-THREAT-EXPORTER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-THREAT-DRAGON-JSON' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-7CF609209E449D9B' from 'FU-THREAT-EXPORTER'.pOut to 'DO-THREAT-DRAGON-JSON'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-7E42BC15F9A9B753";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-7E42BC15F9A9B753' from 'validate-requirements-candidate' to 'FU-VALIDATION-ENGINE';

    @EngineeringRelationship {
        sourceId = "REL-7F0E9B510AB315EA";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-7F0E9B510AB315EA' from 'ACT-AI-AGENT' to 'FU-MCP-SERVER';

    @EngineeringRelationship {
        sourceId = "REL-804292F70AB37E87";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-804292F70AB37E87-context' {
        part 'FU-STRUCTURIZR-EXPORTER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-STRUCTURIZR-DSL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-804292F70AB37E87' from 'FU-STRUCTURIZR-EXPORTER'.pOut to 'DO-STRUCTURIZR-DSL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-82A111E1D914A0F9";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-82A111E1D914A0F9-context' {
        part 'ACT-IMPLEMENTATION-ENGINEER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-CLI-ORCHESTRATION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-82A111E1D914A0F9' from 'ACT-IMPLEMENTATION-ENGINEER'.pOut to 'FU-CLI-ORCHESTRATION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-8359402B3B322449";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-8359402B3B322449' from 'FU-CLI-ORCHESTRATION' to 'FU-MODEL-LOADER';

    @EngineeringRelationship {
        sourceId = "REL-85CEE36056EB8CFE";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-85CEE36056EB8CFE' : EngineeringVerification {
        subject 'VER-MCP-PATH-BOUNDARY' : EngineeringItem;
        objective 'CTRL-MCP-PATH-BOUNDARY' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-85EF5D66A73D662E";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-85EF5D66A73D662E' from 'evaluate-bumped-links' to 'FU-ALLOCATION-TRACE';

    @EngineeringRelationship {
        sourceId = "REL-87B612D71D6265B1";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-87B612D71D6265B1-context' : EngineeringState {
        state 'STATE-LINK-CLAIMED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-87B612D71D6265B1' first 'STATE-LINK-CLAIMED' accept 'EVT-LINK-REVIEW-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-8AAA646D9D1CF563";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-8AAA646D9D1CF563' first 'repin-unchanged-links' then 'restore-bump-files';

    @EngineeringRelationship {
        sourceId = "REL-99640AD91A774667";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-99640AD91A774667' from 'AV-SCHEMA-SUPPLY-CHAIN-TAMPER' to 'REF-OPEN-OTM-SCHEMA';

    @EngineeringRelationship {
        sourceId = "REL-9F9D0BFE4684AB66";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-9F9D0BFE4684AB66' from 'repin-unchanged-links' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-A02740CF7074A233";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-A02740CF7074A233' from 'FU-THREAT-EXPORTER' to 'REF-THREAT-DRAGON-SCHEMAS';

    @EngineeringRelationship {
        sourceId = "REL-A37383B5AD1D4C19";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-A37383B5AD1D4C19-context' {
        part 'FU-MCP-SERVER' : EngineeringPart;
        part 'DEP-LOCAL-WORKSPACE' : EngineeringPart;
        allocation 'REL-A37383B5AD1D4C19' allocate 'FU-MCP-SERVER' to 'DEP-LOCAL-WORKSPACE';
    }

    @EngineeringRelationship {
        sourceId = "REL-A5F08BB76981DB0A";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-A5F08BB76981DB0A' : EngineeringVerification {
        subject 'VER-ACCEPTANCE-RECORD-CHECK' : EngineeringItem;
        objective 'CTRL-ACCEPTANCE-RECORD-CHECK' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-A6BE9BBEB4A2E9CD";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-A6BE9BBEB4A2E9CD' from 'AV-TRACEABILITY-GAP-DRIFT' to 'FU-CODEMAP-INFERENCE';

    @EngineeringRelationship {
        sourceId = "REL-A7B68DAFAC2CF94B";
        conceptKind = "transition";
        relationshipKind = "transitions_to";
    }
    state 'REL-A7B68DAFAC2CF94B-context' : EngineeringState {
        action 'STATE-LINK-SUSPECT' : EngineeringAction;
        action 'STATE-LINK-ACCEPTED' : EngineeringAction;
        transition 'REL-A7B68DAFAC2CF94B' first 'STATE-LINK-SUSPECT' then 'STATE-LINK-ACCEPTED';
    }

    @EngineeringRelationship {
        sourceId = "REL-A8B1B31ABBE97299";
        conceptKind = "transition";
        relationshipKind = "transitions_to";
    }
    state 'REL-A8B1B31ABBE97299-context' : EngineeringState {
        action 'STATE-LINK-ACCEPTED' : EngineeringAction;
        action 'STATE-LINK-BROKEN' : EngineeringAction;
        transition 'REL-A8B1B31ABBE97299' first 'STATE-LINK-ACCEPTED' then 'STATE-LINK-BROKEN';
    }

    @EngineeringRelationship {
        sourceId = "REL-A9A59951D996EDE9";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-A9A59951D996EDE9' : EngineeringVerification {
        subject 'VER-GO-WORK-INPUT-VALIDATION' : EngineeringItem;
        objective 'CTRL-GO-WORK-INPUT-VALIDATION' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-A9A6ADD0BC7DA4E2";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-A9A6ADD0BC7DA4E2' from 'FU-MCP-SERVER' to 'FU-VALIDATION-ENGINE';

    @EngineeringRelationship {
        sourceId = "REL-A9D8D80AF01E0E73";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-A9D8D80AF01E0E73-context' : EngineeringState {
        state 'STATE-LINK-ACCEPTED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-A9D8D80AF01E0E73' first 'STATE-LINK-ACCEPTED' accept 'EVT-DEPENDENCY-BUMP-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-B1CEFDE3267D1A07";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-B1CEFDE3267D1A07' from 'FU-MCP-SERVER' to 'FU-MODEL-LOADER';

    @EngineeringRelationship {
        sourceId = "REL-B2CC49678F3F7718";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-B2CC49678F3F7718' from 'AV-SCHEMA-SUPPLY-CHAIN-TAMPER' to 'REF-THREAT-DRAGON-SCHEMAS';

    @EngineeringRelationship {
        sourceId = "REL-B5AC617E2209835F";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-B5AC617E2209835F' from 'edit-model' to 'ACT-IMPLEMENTATION-ENGINEER';

    @EngineeringRelationship {
        sourceId = "REL-B83795AEC231AF42";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-B83795AEC231AF42' first 'load-requirements-delta' then 'merge-requirements-candidate';

    @EngineeringRelationship {
        sourceId = "REL-BDFCE1037DF5E841";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-BDFCE1037DF5E841' from 'AV-PATH-TRAVERSAL-IN-MCP' to 'CTRL-MCP-PATH-BOUNDARY';

    @EngineeringRelationship {
        sourceId = "REL-C139691071B5EBB7";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-C139691071B5EBB7' : EngineeringVerification {
        subject 'VER-TRACEABILITY-COVERAGE' : EngineeringItem;
        objective 'CTRL-TRACEABILITY-COVERAGE' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-C14D1E0A5A7D260D";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-C14D1E0A5A7D260D' from 'FU-THREAT-EXPORTER' to 'TB-EXTERNAL-VALIDATION-TOOLS';

    @EngineeringRelationship {
        sourceId = "REL-C5462A84B5B762E0";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-C5462A84B5B762E0' from 'restore-bump-files' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-C95225800FC7BD46";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-C95225800FC7BD46-context' : EngineeringState {
        state 'STATE-ARTIFACTS-FRESH' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-C95225800FC7BD46' first 'STATE-ARTIFACTS-FRESH' accept 'EVT-ARTIFACT-GENERATED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-C9FD7D7F73C1C297";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-C9FD7D7F73C1C297' from 'ACT-ARCHITECTURE-AUTHOR' to 'FU-CLI-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-CBB97983472F3FF7";
        conceptKind = "guard";
        relationshipKind = "guarded_by";
    }
    state 'REL-CBB97983472F3FF7-context' : EngineeringState {
        action 'STATE-MODEL-VALID' : EngineeringAction;
        action 'CTRL-STRICT-MCP-INPUT-SCHEMA' : EngineeringAction;
        transition 'REL-CBB97983472F3FF7' first 'STATE-MODEL-VALID' if true then 'CTRL-STRICT-MCP-INPUT-SCHEMA';
    }

    @EngineeringRelationship {
        sourceId = "REL-CC26ADB229638ACA";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-CC26ADB229638ACA' from 'FU-STRUCTURIZR-EXPORTER' to 'TB-EXTERNAL-VALIDATION-TOOLS';

    @EngineeringRelationship {
        sourceId = "REL-CDF33573A9ECBCE7";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-CDF33573A9ECBCE7' first 'evaluate-bumped-links' then 'repin-unchanged-links';

    @EngineeringRelationship {
        sourceId = "REL-CE24A4335AEBEDD2";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-CE24A4335AEBEDD2-context' {
        part 'FU-MCP-SERVER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-MCP-TOOL-RESULT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-CE24A4335AEBEDD2' from 'FU-MCP-SERVER'.pOut to 'DO-MCP-TOOL-RESULT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-D2DF2108EF6F792E";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-D2DF2108EF6F792E-context' {
        part 'FU-MODEL-LOADER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-CANONICAL-SEMANTIC-MODEL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-D2DF2108EF6F792E' from 'FU-MODEL-LOADER'.pOut to 'DO-CANONICAL-SEMANTIC-MODEL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-DA53286A42297199";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-DA53286A42297199-context' {
        part 'FU-MODEL-CHANGE' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-REQUIREMENTS-DOCUMENT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-DA53286A42297199' from 'FU-MODEL-CHANGE'.pOut to 'DO-REQUIREMENTS-DOCUMENT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-E6134BEEFB9A3A7B";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-E6134BEEFB9A3A7B' first 'merge-requirements-candidate' then 'validate-requirements-candidate';

    @EngineeringRelationship {
        sourceId = "REL-E75B93B894A74F66";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-E75B93B894A74F66' from 'show-links' to 'FU-ALLOCATION-TRACE';

    @EngineeringRelationship {
        sourceId = "REL-E801A87C05EDBF10";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-E801A87C05EDBF10-context' : EngineeringState {
        state 'STATE-LINK-SUSPECT' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-E801A87C05EDBF10' first 'STATE-LINK-SUSPECT' accept 'EVT-LINK-REVIEW-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-EC073BE26257DD9C";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-EC073BE26257DD9C' from 'AV-MALFORMED-MODEL-INPUT' to 'FU-MODEL-LOADER';

    @EngineeringRelationship {
        sourceId = "REL-EC535616F6927588";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-EC535616F6927588' from 'reject-requirements-delta' to 'FU-MODEL-CHANGE';

    @EngineeringRelationship {
        sourceId = "REL-F0B53FC843714C96";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-F0B53FC843714C96' from 'replace-requirements' to 'FU-MODEL-CHANGE';

    @EngineeringRelationship {
        sourceId = "REL-F46F40881E2B363D";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-F46F40881E2B363D-context' : EngineeringState {
        state 'STATE-MODEL-VALID' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-F46F40881E2B363D' first 'STATE-MODEL-VALID' accept 'EVT-MODEL-UPDATED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-F808BADA7946F75D";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-F808BADA7946F75D' first 'validate-model' then 'fail-validation';

    @EngineeringRelationship {
        sourceId = "REL-FB6988A6DDD6C7A0";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-FB6988A6DDD6C7A0' from 'FU-SYSML-EXPORTER' to 'TB-EXTERNAL-VALIDATION-TOOLS';

    @EngineeringRelationship {
        sourceId = "REL-FCE9D28483B45994";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-FCE9D28483B45994' from 'FU-NAF-EXPORTER' to 'FU-VIEW-PROJECTION';

    @EngineeringRelationship {
        sourceId = "REL-FE0645A36FB72976";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-FE0645A36FB72976' from 'merge-requirements-candidate' to 'FU-MODEL-CHANGE';

    @EngineeringRelationship {
        sourceId = "REL-FFE6760421FF8D31";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-FFE6760421FF8D31' from 'validate-model' to 'FU-VALIDATION-ENGINE';
}
