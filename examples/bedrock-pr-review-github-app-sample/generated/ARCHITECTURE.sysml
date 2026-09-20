// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'sample-bedrock-pr-review-model' {
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
        sourceId = "sample-bedrock-pr-review-model";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-DEVELOPER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-DEVELOPER';

    @EngineeringElement {
        sourceId = "ACT-PLATFORM-OPERATOR";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-PLATFORM-OPERATOR';

    @EngineeringElement {
        sourceId = "ACT-REPOSITORY-MAINTAINER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-REPOSITORY-MAINTAINER';

    @EngineeringElement {
        sourceId = "ACT-SECURITY-ENGINEER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-SECURITY-ENGINEER';

    @EngineeringElement {
        sourceId = "ADR-PRR-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-PRR-001';

    @EngineeringElement {
        sourceId = "ASM-BEDROCK-GITHUB-SIGNATURE-TRUST";
        conceptKind = "engineering_extension";
    }
    attribute 'ASM-BEDROCK-GITHUB-SIGNATURE-TRUST';

    @EngineeringElement {
        sourceId = "AV-PR-SPAM-ABUSE";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-PR-SPAM-ABUSE';

    @EngineeringElement {
        sourceId = "AV-PROMPT-INJECTION-IN-DIFF";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-PROMPT-INJECTION-IN-DIFF';

    @EngineeringElement {
        sourceId = "AV-SECRET-LEAKAGE-IN-REVIEW";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-SECRET-LEAKAGE-IN-REVIEW';

    @EngineeringElement {
        sourceId = "AV-SPOOFED-GITHUB-WEBHOOK";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-SPOOFED-GITHUB-WEBHOOK';

    @EngineeringElement {
        sourceId = "CONCERN-00DBA90641A07297";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-00DBA90641A07297' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Owns credentials and runtime configuration contracts used by review units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0434EBD3CAC7280D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0434EBD3CAC7280D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Posts summaries and inline annotations through GitHub App APIs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0A6D7172D6531D7A";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0A6D7172D6531D7A' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Isolated deployment enables cautious model/config rollout and fallback control.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0E04A9D8E942A3FF";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0E04A9D8E942A3FF' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Owns runtime deployment sequencing, control-plane continuity, and operational safety.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0F735EF5786D2BDC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0F735EF5786D2BDC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Managed as a shared operational boundary with controlled rotation and rollout flow.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-13FE40CA63751F28";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-13FE40CA63751F28' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Review Intelligence emphasizes prompt hardening, output redaction, and secure publication boundaries.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-1FBDF65CFCC1809D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-1FBDF65CFCC1809D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traces credential and redaction controls to audit and publication requirements.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-243D3E93E422C968";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-243D3E93E422C968' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Exchanges policy findings with orchestration for final result composition.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-284B5EFAEF4FC26F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-284B5EFAEF4FC26F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Coordinates model inference and deterministic checks into one review outcome.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2A8472D9BF0491AC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2A8472D9BF0491AC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Review Intelligence coordinates Bedrock calls, policy evaluation, and publication sequencing for deterministic outcomes.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2C76449F27902F94";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2C76449F27902F94' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Governs source-to-release pipeline behavior and runtime placement boundaries.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2E19E0417EAF0DB1";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2E19E0417EAF0DB1' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traces each deterministic finding category to requirement ownership.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-32CADB97F382BD07";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-32CADB97F382BD07' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Accepts and verifies GitHub webhook events before any review processing.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4718F1807940A778";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4718F1807940A778' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Deployed independently to evolve policy logic without model coupling.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4D0DD1E902C29AF2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4D0DD1E902C29AF2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "PR Integration receives webhook events and forwards normalized review context to orchestration with explicit handoff semantics.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4DD608A280580BEC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4DD608A280580BEC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traces context fetch and preparation logic to requirement evidence.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-524DA06EA8AC231E";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-524DA06EA8AC231E' : EngineeringConcern {
        attribute 'narrative' : 'String' = "PR Integration protects signature verification and hostile-diff ingress paths to reduce spoofing and prompt injection risk.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5B66A74BD788C82C";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5B66A74BD788C82C' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces signature verification and rejects spoofed or malformed events.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5D4B26B43242E2B5";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5D4B26B43242E2B5' : EngineeringConcern {
        attribute 'narrative' : 'String' = "PR Integration ownership is maintained at ingress and context modules with requirement-level trace markers.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5D9E022FFF0E57F6";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5D9E022FFF0E57F6' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Operates deployment and runtime controls that indirectly shape request-path reliability.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-60A1591DD9BEDA62";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-60A1591DD9BEDA62' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform Operations secures credential paths and operational controls across Lambda deployment lifecycle.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-61F5E8D0327348C8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-61F5E8D0327348C8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Provides deterministic controls for secrets and risky infrastructure changes.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-657462030D83AD38";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-657462030D83AD38' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces safe prompt construction and bounded output handling under adversarial input.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-65C52A8C4EC75BB2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-65C52A8C4EC75BB2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform Operations records ownership and evidence for deployment and secret-management behavior.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-70F7A8627542BB2F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-70F7A8627542BB2F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Applies contextual sanitization to reduce prompt injection risk from diffs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-773024BD308E2EEC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-773024BD308E2EEC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Hardens operational control loops and protects deployment trust boundaries.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-784CCCACE544AFF0";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-784CCCACE544AFF0' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Captures deployment/runtime evidence linking operational ownership to architecture intent.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-83D8D1CAE6FA36F9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-83D8D1CAE6FA36F9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform Operations owns deployability, runtime controls, and secure configuration boundaries for the review app.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8457249B101AF235";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8457249B101AF235' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Deploys as an independently managed Lambda runtime boundary.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8581AF3672AAF1FE";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8581AF3672AAF1FE' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Builds bounded, review-ready context from pull request files and metadata.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-893119DD064684F8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-893119DD064684F8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "PR Integration functions deploy independently to isolate webhook ingress from context-fetch implementation changes.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8CEF3E27D78F1826";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8CEF3E27D78F1826' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Release isolation protects publication reliability during other unit rollouts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-90407301E6FCADDC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-90407301E6FCADDC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Review Intelligence units roll independently to preserve availability and reduce blast radius during model or rule updates.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-9D860B0D21BEBA46";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-9D860B0D21BEBA46' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Review Intelligence traces AI findings and deterministic checks back to requirements and evidence sources.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A17BC51B28E13C37";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A17BC51B28E13C37' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Performs deterministic repository checks independent of model output.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A19BFB69CAEAB5A0";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A19BFB69CAEAB5A0' : EngineeringConcern {
        attribute 'narrative' : 'String' = "PR Integration defines trusted ingestion boundaries and normalized PR context assembly for downstream review units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A3F837EB60AB2619";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A3F837EB60AB2619' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Review Intelligence defines how AI and deterministic signals are combined and published as actionable PR feedback.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B22E47EBD238BDF2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B22E47EBD238BDF2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Publishes review findings to GitHub checks and comments with clear merge-gate signals.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B3058E33939664DA";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B3058E33939664DA' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform Operations governs release orchestration and runtime placement to keep review capacity stable.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B5CBC08C36F1DC63";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B5CBC08C36F1DC63' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Traces model-request and fallback behavior to requirements and evidence artifacts.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-CACF59DF6F0370C4";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-CACF59DF6F0370C4' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Receives webhook payloads, validates them, and routes authorized events to context assembly.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-D626A3C4799D0695";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-D626A3C4799D0695' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Requirement traces capture ingress validation and abuse-protection behavior.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-D98276A7344BC931";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-D98276A7344BC931' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Platform Operations coordinates runtime controls with app units via deployment and secret/config integration interfaces.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E459FFB5B8806328";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E459FFB5B8806328' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Enforces credential isolation, least-privilege access, and redaction-safe metadata handling.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E8DC7D778709CC4A";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E8DC7D778709CC4A' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Applies redaction and policy-aware output filtering before publishing findings.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-EC09B88F1E026B13";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-EC09B88F1E026B13' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Calls GitHub APIs and forwards normalized context to orchestration.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F2698AEE2356BAF3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F2698AEE2356BAF3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Provides secure config retrieval contracts to orchestration and publication units.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F9E4B9FA375F7195";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F9E4B9FA375F7195' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Integrates context input, Bedrock calls, and publication output in a deterministic sequence.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-FB330E3B1E14D427";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-FB330E3B1E14D427' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Deployed separately to allow context shaping changes without ingress disruption.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-FC96F19A79ED11E4";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-FC96F19A79ED11E4' : EngineeringConcern {
        attribute 'narrative' : 'String' = "Maps published outcomes and audit metadata to requirement-level evidence.";
    }

    @EngineeringElement {
        sourceId = "COND-WEBHOOK-SIGNATURE-VALID";
        conceptKind = "engineering_extension";
    }
    attribute 'COND-WEBHOOK-SIGNATURE-VALID';

    @EngineeringElement {
        sourceId = "CTRL-BEDROCK-SECRETS-ROTATION";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-BEDROCK-SECRETS-ROTATION';

    @EngineeringElement {
        sourceId = "CTRL-BEDROCK-WEBHOOK-SIGNATURE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-BEDROCK-WEBHOOK-SIGNATURE';

    @EngineeringElement {
        sourceId = "CV-BEDROCK-PROMPT-INJECTION-RESILIENCE";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-BEDROCK-PROMPT-INJECTION-RESILIENCE' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-BEDROCK-REDACTION-CHECK";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-BEDROCK-REDACTION-CHECK' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-BEDROCK-WEBHOOK-SIGNATURE";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-BEDROCK-WEBHOOK-SIGNATURE' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "DATA-CHECKRUN-SUMMARY";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-CHECKRUN-SUMMARY';

    @EngineeringElement {
        sourceId = "DATA-DIFF-CONTEXT";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-DIFF-CONTEXT';

    @EngineeringElement {
        sourceId = "DATA-REDACTED-AUDIT-METADATA";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-REDACTED-AUDIT-METADATA';

    @EngineeringElement {
        sourceId = "DATA-RESULT-METADATA";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-RESULT-METADATA';

    @EngineeringElement {
        sourceId = "DATA-WEBHOOK-SIGNATURE";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-WEBHOOK-SIGNATURE';

    @EngineeringElement {
        sourceId = "DEP-BEDROCK-GITHUB-BOUNDARY";
        conceptKind = "part_definition";
    }
    part def 'DEP-BEDROCK-GITHUB-BOUNDARY' {
        attribute 'environment' : 'String' = "external";
        attribute 'region' : 'String' = "global";
        attribute 'namespace' : 'String' = "app";
        attribute 'trustZone' : 'String' = "external";
    }

    @EngineeringElement {
        sourceId = "DEP-BEDROCK-LAMBDA-PRIMARY";
        conceptKind = "part_definition";
    }
    part def 'DEP-BEDROCK-LAMBDA-PRIMARY' {
        attribute 'environment' : 'String' = "prod";
        attribute 'region' : 'String' = "us-east-1";
        attribute 'namespace' : 'String' = "review";
        attribute 'trustZone' : 'String' = "app";
    }

    @EngineeringElement {
        sourceId = "DO-BEDROCK-PR-CONTEXT";
        conceptKind = "item_definition";
    }
    item def 'DO-BEDROCK-PR-CONTEXT' {
        attribute 'schemaRef' : 'String' = "schemas/pr-context.json";
        attribute 'classification' : 'String' = "source-intelligence";
    }

    @EngineeringElement {
        sourceId = "DO-BEDROCK-REVIEW-FINDINGS";
        conceptKind = "item_definition";
    }
    item def 'DO-BEDROCK-REVIEW-FINDINGS' {
        attribute 'schemaRef' : 'String' = "schemas/review-findings.json";
        attribute 'classification' : 'String' = "security-analysis";
    }

    @EngineeringElement {
        sourceId = "EVIDENCE-0B26F3516D7370A0";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-0B26F3516D7370A0';

    @EngineeringElement {
        sourceId = "EVIDENCE-0BDF640CF0BFEFFC";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-0BDF640CF0BFEFFC';

    @EngineeringElement {
        sourceId = "EVIDENCE-15A5C2E6F96DFF50";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-15A5C2E6F96DFF50';

    @EngineeringElement {
        sourceId = "EVIDENCE-17C25E5C0131D1D6";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-17C25E5C0131D1D6';

    @EngineeringElement {
        sourceId = "EVIDENCE-435025A270C7DA0F";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-435025A270C7DA0F';

    @EngineeringElement {
        sourceId = "EVIDENCE-456E5D7799DEB0AD";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-456E5D7799DEB0AD';

    @EngineeringElement {
        sourceId = "EVIDENCE-538AC4CDE48A2C34";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-538AC4CDE48A2C34';

    @EngineeringElement {
        sourceId = "EVIDENCE-5990B259F5853B51";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-5990B259F5853B51';

    @EngineeringElement {
        sourceId = "EVIDENCE-7059D168AC46DBEC";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-7059D168AC46DBEC';

    @EngineeringElement {
        sourceId = "EVIDENCE-7214D20F108D0405";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-7214D20F108D0405';

    @EngineeringElement {
        sourceId = "EVIDENCE-8EE401948BB307E3";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-8EE401948BB307E3';

    @EngineeringElement {
        sourceId = "EVIDENCE-8FD3E68D9069FCAD";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-8FD3E68D9069FCAD';

    @EngineeringElement {
        sourceId = "EVIDENCE-A6B8267A5D2E650C";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-A6B8267A5D2E650C';

    @EngineeringElement {
        sourceId = "EVIDENCE-AA2FC813D714A7F5";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-AA2FC813D714A7F5';

    @EngineeringElement {
        sourceId = "EVIDENCE-D7CB2EEDEE6DD3C0";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-D7CB2EEDEE6DD3C0';

    @EngineeringElement {
        sourceId = "EVIDENCE-D7DA23F3C5EB72D9";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-D7DA23F3C5EB72D9';

    @EngineeringElement {
        sourceId = "EVIDENCE-F6008205D231F79A";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-F6008205D231F79A';

    @EngineeringElement {
        sourceId = "EVT-BEDROCK-REVIEW-COMPLETED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-BEDROCK-REVIEW-COMPLETED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-BEDROCK-REVIEW-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-BEDROCK-REVIEW-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-BEDROCK-UNAVAILABLE";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-BEDROCK-UNAVAILABLE';

    @EngineeringElement {
        sourceId = "EVT-CONTEXT-PREPARED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-CONTEXT-PREPARED';

    @EngineeringElement {
        sourceId = "EVT-FINDINGS-GENERATED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-FINDINGS-GENERATED';

    @EngineeringElement {
        sourceId = "EVT-FINDINGS-PUBLISHED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-FINDINGS-PUBLISHED';

    @EngineeringElement {
        sourceId = "EVT-PR-OPENED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-PR-OPENED';

    @EngineeringElement {
        sourceId = "EVT-PR-SYNCHRONIZE";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-PR-SYNCHRONIZE';

    @EngineeringElement {
        sourceId = "EVT-RATE-LIMIT-EXCEEDED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-RATE-LIMIT-EXCEEDED';

    @EngineeringElement {
        sourceId = "EVT-RELEASE-APPROVED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-RELEASE-APPROVED';

    @EngineeringElement {
        sourceId = "EXTENSION-B218705E25057CF9";
        conceptKind = "engineering_extension";
    }
    attribute 'EXTENSION-B218705E25057CF9';

    @EngineeringElement {
        sourceId = "FEAT-AI-REVIEW";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-AI-REVIEW';

    @EngineeringElement {
        sourceId = "FEAT-AUDIT-LOGGING";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-AUDIT-LOGGING';

    @EngineeringElement {
        sourceId = "FEAT-LAMBDA-RUNTIME-OPS";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-LAMBDA-RUNTIME-OPS';

    @EngineeringElement {
        sourceId = "FEAT-POLICY-CHECKS";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-POLICY-CHECKS';

    @EngineeringElement {
        sourceId = "FG-PLATFORM-OPERATIONS";
        conceptKind = "part_definition";
    }
    part def 'FG-PLATFORM-OPERATIONS';

    @EngineeringElement {
        sourceId = "FG-PR-INTEGRATION";
        conceptKind = "part_definition";
    }
    part def 'FG-PR-INTEGRATION';

    @EngineeringElement {
        sourceId = "FG-REVIEW-INTELLIGENCE";
        conceptKind = "part_definition";
    }
    part def 'FG-REVIEW-INTELLIGENCE';

    @EngineeringElement {
        sourceId = "FLOW-BEDROCK-POLICY-TUNING";
        conceptKind = "action_definition";
    }
    action def 'FLOW-BEDROCK-POLICY-TUNING' {
        inout item 'item_DO_BEDROCK_PR_CONTEXT' : 'DO-BEDROCK-PR-CONTEXT';
        inout item 'item_DO_BEDROCK_REVIEW_FINDINGS' : 'DO-BEDROCK-REVIEW-FINDINGS';
    }

    @EngineeringElement {
        sourceId = "FLOW-BEDROCK-PR-REVIEW";
        conceptKind = "action_definition";
    }
    action def 'FLOW-BEDROCK-PR-REVIEW' {
        inout item 'item_DO_BEDROCK_PR_CONTEXT' : 'DO-BEDROCK-PR-CONTEXT';
        inout item 'item_DO_BEDROCK_REVIEW_FINDINGS' : 'DO-BEDROCK-REVIEW-FINDINGS';
    }

    @EngineeringElement {
        sourceId = "FU-GITHUB-WEBHOOK-INGRESS";
        conceptKind = "part_usage";
    }
    part 'FU-GITHUB-WEBHOOK-INGRESS' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-LAMBDA-RUNTIME-OPERATIONS";
        conceptKind = "part_usage";
    }
    part 'FU-LAMBDA-RUNTIME-OPERATIONS' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-POLICY-CHECKS";
        conceptKind = "part_usage";
    }
    part 'FU-POLICY-CHECKS' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-PR-CONTEXT-ASSEMBLY";
        conceptKind = "part_usage";
    }
    part 'FU-PR-CONTEXT-ASSEMBLY' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-REVIEW-ORCHESTRATION";
        conceptKind = "part_usage";
    }
    part 'FU-REVIEW-ORCHESTRATION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-REVIEW-PUBLICATION";
        conceptKind = "part_usage";
    }
    part 'FU-REVIEW-PUBLICATION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-SECRETS-CONFIGURATION";
        conceptKind = "part_usage";
    }
    part 'FU-SECRETS-CONFIGURATION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "IF-BEDROCK-REVIEW-PUBLISH-API";
        conceptKind = "interface_definition";
    }
    interface def 'IF-BEDROCK-REVIEW-PUBLISH-API' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "/github/check-runs";
        attribute 'schemaRef' : 'String' = "schemas/review-output.json";
    }

    @EngineeringElement {
        sourceId = "IF-BEDROCK-WEBHOOK-API";
        conceptKind = "interface_definition";
    }
    interface def 'IF-BEDROCK-WEBHOOK-API' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "/webhook/github";
        attribute 'schemaRef' : 'String' = "schemas/webhook-event.json";
    }

    @EngineeringElement {
        sourceId = "MAP-BEDROCK-SECRETS-ROTATION";
        conceptKind = "engineering_extension";
    }
    attribute 'MAP-BEDROCK-SECRETS-ROTATION';

    @EngineeringElement {
        sourceId = "MAP-BEDROCK-WEBHOOK-SIGNATURE";
        conceptKind = "engineering_extension";
    }
    attribute 'MAP-BEDROCK-WEBHOOK-SIGNATURE';

    @EngineeringElement {
        sourceId = "MIT-BEDROCK-CONTEXT-SANITIZATION";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-BEDROCK-CONTEXT-SANITIZATION';

    @EngineeringElement {
        sourceId = "MIT-BEDROCK-OUTPUT-REDACTION";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-BEDROCK-OUTPUT-REDACTION';

    @EngineeringElement {
        sourceId = "MIT-BEDROCK-WEBHOOK-HMAC";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-BEDROCK-WEBHOOK-HMAC';

    @EngineeringElement {
        sourceId = "MODE-NORMAL";
        conceptKind = "engineering_extension";
    }
    attribute 'MODE-NORMAL';

    @EngineeringElement {
        sourceId = "OOS-BEDROCK-GITHUB-CLOUD-PLATFORM";
        conceptKind = "engineering_extension";
    }
    attribute 'OOS-BEDROCK-GITHUB-CLOUD-PLATFORM';

    @EngineeringElement {
        sourceId = "POAM-BEDROCK-REPLAY-HARDENING";
        conceptKind = "engineering_extension";
    }
    attribute 'POAM-BEDROCK-REPLAY-HARDENING';

    @EngineeringElement {
        sourceId = "PROF-NIST-800-53-MODERATE";
        conceptKind = "engineering_extension";
    }
    attribute 'PROF-NIST-800-53-MODERATE';

    @EngineeringElement {
        sourceId = "REF-AWS-BEDROCK-RUNTIME";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-AWS-BEDROCK-RUNTIME';

    @EngineeringElement {
        sourceId = "REF-AWS-SDK-GO";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-AWS-SDK-GO';

    @EngineeringElement {
        sourceId = "REF-AWS-SECRETS-MANAGER";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-AWS-SECRETS-MANAGER';

    @EngineeringElement {
        sourceId = "REF-GITHUB-APP-API";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-GITHUB-APP-API';

    @EngineeringElement {
        sourceId = "REF-GITHUB-REST-SDK";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-GITHUB-REST-SDK';

    @EngineeringElement {
        sourceId = "REF-POLICY-RULESET";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-POLICY-RULESET';

    @EngineeringElement {
        sourceId = "REQ-PRR-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-001' {
        attribute 'text' : 'String' = "While normal mode is active, when pull request synchronize event is received and webhook signature is valid, the pr review system shall verify webhook signature before processing review workflow.";
        attribute 'notes' : 'String' = "Webhook trust boundary validation.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-002' {
        attribute 'text' : 'String' = "While pull request review is active, when pull request opened event is received, the pr review system shall construct diff context from changed files for developer review execution.";
        attribute 'notes' : 'String' = "Context ingestion from GitHub API.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-003' {
        attribute 'text' : 'String' = "Where ai review is enabled, when diff context is prepared, the pr review system shall request bedrock runtime analysis for review findings.";
        attribute 'notes' : 'String' = "AI-assisted review path.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-004";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-004' {
        attribute 'text' : 'String' = "Where policy checks are enabled, when pull request review is active, the pr review system shall evaluate deterministic policy checks for secret exposure plus infrastructure risk plus dependency changes.";
        attribute 'notes' : 'String' = "Deterministic non-LLM review checks.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-005";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-005' {
        attribute 'text' : 'String' = "While pull request review is active, when review findings are generated, the pr review system shall publish check run summary to repository maintainer.";
        attribute 'notes' : 'String' = "Result publication contract.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-006";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-006' {
        attribute 'text' : 'String' = "If bedrock runtime is unavailable, then the pr review system shall publish deterministic policy-only review outcome.";
        attribute 'notes' : 'String' = "Graceful degradation requirement.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-007";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-007' {
        attribute 'text' : 'String' = "Where audit logging is enabled, when review findings are published, the pr review system shall persist redacted audit metadata for security engineer audit traceability.";
        attribute 'notes' : 'String' = "Evidence and audit lineage.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-008";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-008' {
        attribute 'text' : 'String' = "If review request rate limit is exceeded, then the pr review system shall defer processing.";
        attribute 'notes' : 'String' = "Abuse and capacity protection behavior.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-009";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-009' {
        attribute 'text' : 'String' = "While pull request review is active, when review findings are generated, the pr review system shall publish inline comments with result metadata to repository maintainer.";
        attribute 'notes' : 'String' = "Inline-annotation publication contract.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-010";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-010' {
        attribute 'text' : 'String' = "If review request rate limit is exceeded, then the pr review system shall notify platform operator.";
        attribute 'notes' : 'String' = "Maintainer notification for deferred review processing.";
    }

    @EngineeringElement {
        sourceId = "REQ-PRR-011";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PRR-011' {
        attribute 'text' : 'String' = "Where lambda runtime operations are enabled, when release artifact is approved, the pr review system shall deploy lambda function image to primary runtime.";
        attribute 'notes' : 'String' = "Runtime deployment lifecycle for the Lambda-hosted review application.";
    }

    @EngineeringElement {
        sourceId = "RISK-BEDROCK-PROMPT-INJECTION";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-BEDROCK-PROMPT-INJECTION';

    @EngineeringElement {
        sourceId = "RISK-BEDROCK-SECRET-LEAKAGE";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-BEDROCK-SECRET-LEAKAGE';

    @EngineeringElement {
        sourceId = "RISK-BEDROCK-WEBHOOK-SPOOFING";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-BEDROCK-WEBHOOK-SPOOFING';

    @EngineeringElement {
        sourceId = "STATE-BEDROCK-PR-RECEIVED";
        conceptKind = "state_definition";
    }
    state def 'STATE-BEDROCK-PR-RECEIVED';

    @EngineeringElement {
        sourceId = "STATE-BEDROCK-REVIEW-PUBLISHED";
        conceptKind = "state_definition";
    }
    state def 'STATE-BEDROCK-REVIEW-PUBLISHED';

    @EngineeringElement {
        sourceId = "STATE-PR-REVIEW-ACTIVE";
        conceptKind = "engineering_extension";
    }
    attribute 'STATE-PR-REVIEW-ACTIVE';

    @EngineeringElement {
        sourceId = "SYS-PR-REVIEW-SYSTEM";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-PR-REVIEW-SYSTEM';

    @EngineeringElement {
        sourceId = "TB-BEDROCK-AWS-CONTROL";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-BEDROCK-AWS-CONTROL';

    @EngineeringElement {
        sourceId = "TB-BEDROCK-GITHUB-EXTERNAL";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-BEDROCK-GITHUB-EXTERNAL';

    @EngineeringElement {
        sourceId = "TS-BEDROCK-PROMPT-INJECTION";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-BEDROCK-PROMPT-INJECTION';

    @EngineeringElement {
        sourceId = "TS-BEDROCK-SECRET-LEAK-IN-PUBLICATION";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-BEDROCK-SECRET-LEAK-IN-PUBLICATION';

    @EngineeringElement {
        sourceId = "TS-BEDROCK-WEBHOOK-SPOOF";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-BEDROCK-WEBHOOK-SPOOF';

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
        sourceId = "assemble-context";
        conceptKind = "action_usage";
    }
    action 'assemble-context' : EngineeringAction {
        out item 'output_pr_context' : EngineeringItem;
        inout item 'item_DO_BEDROCK_PR_CONTEXT' : 'DO-BEDROCK-PR-CONTEXT';
    }

    @EngineeringElement {
        sourceId = "orchestrate-policy-validation";
        conceptKind = "action_usage";
    }
    action 'orchestrate-policy-validation' : EngineeringAction {
        in item 'input_policy_check_results' : EngineeringItem;
        out item 'output_validation_findings' : EngineeringItem;
        inout item 'item_DO_BEDROCK_REVIEW_FINDINGS' : 'DO-BEDROCK-REVIEW-FINDINGS';
    }

    @EngineeringElement {
        sourceId = "orchestrate-review";
        conceptKind = "action_usage";
    }
    action 'orchestrate-review' : EngineeringAction {
        in item 'input_pr_context' : EngineeringItem;
        out item 'output_review_findings' : EngineeringItem;
        inout item 'item_DO_BEDROCK_PR_CONTEXT' : 'DO-BEDROCK-PR-CONTEXT';
        inout item 'item_DO_BEDROCK_REVIEW_FINDINGS' : 'DO-BEDROCK-REVIEW-FINDINGS';
    }

    @EngineeringElement {
        sourceId = "publish-policy-validation";
        conceptKind = "action_usage";
    }
    action 'publish-policy-validation' : EngineeringAction {
        in item 'input_validation_findings' : EngineeringItem;
        inout item 'item_DO_BEDROCK_REVIEW_FINDINGS' : 'DO-BEDROCK-REVIEW-FINDINGS';
    }

    @EngineeringElement {
        sourceId = "publish-review";
        conceptKind = "action_usage";
    }
    action 'publish-review' : EngineeringAction {
        in item 'input_review_findings' : EngineeringItem;
        inout item 'item_DO_BEDROCK_REVIEW_FINDINGS' : 'DO-BEDROCK-REVIEW-FINDINGS';
    }

    @EngineeringElement {
        sourceId = "receive-webhook";
        conceptKind = "action_usage";
    }
    action 'receive-webhook' : EngineeringAction;

    @EngineeringElement {
        sourceId = "run-policy-checks";
        conceptKind = "action_usage";
    }
    action 'run-policy-checks' : EngineeringAction {
        out item 'output_policy_check_results' : EngineeringItem;
    }

    @EngineeringElement {
        sourceId = "update-policy-rules";
        conceptKind = "action_usage";
    }
    action 'update-policy-rules' : EngineeringAction;

    @EngineeringRelationship {
        sourceId = "REL-006D64952989E5D6";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-006D64952989E5D6' : EngineeringVerification {
        subject 'CV-BEDROCK-PROMPT-INJECTION-RESILIENCE' : EngineeringItem;
        objective 'CTRL-BEDROCK-WEBHOOK-SIGNATURE' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-020E3C5C0A59AB2B";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-020E3C5C0A59AB2B' first 'receive-webhook' then 'assemble-context';

    @EngineeringRelationship {
        sourceId = "REL-03E153B16C852140";
        conceptKind = "transition";
        relationshipKind = "transitions_to";
    }
    state 'REL-03E153B16C852140-context' : EngineeringState {
        action 'STATE-BEDROCK-PR-RECEIVED' : EngineeringAction;
        action 'STATE-BEDROCK-REVIEW-PUBLISHED' : EngineeringAction;
        transition 'REL-03E153B16C852140' first 'STATE-BEDROCK-PR-RECEIVED' then 'STATE-BEDROCK-REVIEW-PUBLISHED';
    }

    @EngineeringRelationship {
        sourceId = "REL-06DE46BF6F2B6384";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-06DE46BF6F2B6384-context' {
        part 'FU-REVIEW-PUBLICATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-BEDROCK-REVIEW-FINDINGS' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-06DE46BF6F2B6384' from 'FU-REVIEW-PUBLICATION'.pOut to 'DO-BEDROCK-REVIEW-FINDINGS'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-0A5CF9D4AE03CEB1";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-0A5CF9D4AE03CEB1-context' : EngineeringState {
        state 'STATE-BEDROCK-PR-RECEIVED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-0A5CF9D4AE03CEB1' first 'STATE-BEDROCK-PR-RECEIVED' accept 'EVT-BEDROCK-REVIEW-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-0A5F3DCCF5EAEE67";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-0A5F3DCCF5EAEE67' from 'update-policy-rules' to 'ACT-SECURITY-ENGINEER';

    @EngineeringRelationship {
        sourceId = "REL-0B7204B4C622F638";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-0B7204B4C622F638' from 'assemble-context' to 'FU-PR-CONTEXT-ASSEMBLY';

    @EngineeringRelationship {
        sourceId = "REL-0CC08D90BF442207";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-0CC08D90BF442207' from 'publish-policy-validation' to 'FU-REVIEW-PUBLICATION';

    @EngineeringRelationship {
        sourceId = "REL-1CD5F15979BF0C17";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-1CD5F15979BF0C17' first 'orchestrate-review' then 'publish-review';

    @EngineeringRelationship {
        sourceId = "REL-211FBCEE412DEEC3";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-211FBCEE412DEEC3-context' : EngineeringState {
        state 'STATE-BEDROCK-REVIEW-PUBLISHED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-211FBCEE412DEEC3' first 'STATE-BEDROCK-REVIEW-PUBLISHED' accept 'EVT-BEDROCK-REVIEW-COMPLETED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-291F50EFB4DCCFA2";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-291F50EFB4DCCFA2-context' {
        part 'FU-REVIEW-PUBLICATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'REF-GITHUB-APP-API' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-291F50EFB4DCCFA2' from 'FU-REVIEW-PUBLICATION'.pOut to 'REF-GITHUB-APP-API'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-31B108ADA7F85811";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-31B108ADA7F85811' from 'AV-PROMPT-INJECTION-IN-DIFF' to 'FU-PR-CONTEXT-ASSEMBLY';

    @EngineeringRelationship {
        sourceId = "REL-37A26EAAA5CA5346";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-37A26EAAA5CA5346' from 'AV-PROMPT-INJECTION-IN-DIFF' to 'FU-REVIEW-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-3A988BE234A2037F";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-3A988BE234A2037F' from 'orchestrate-review' to 'FU-REVIEW-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-408C54927AAAC801";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-408C54927AAAC801' from 'run-policy-checks' to 'FU-POLICY-CHECKS';

    @EngineeringRelationship {
        sourceId = "REL-40D28B73BDC2CC57";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-40D28B73BDC2CC57' from 'FU-REVIEW-ORCHESTRATION' to 'FU-SECRETS-CONFIGURATION';

    @EngineeringRelationship {
        sourceId = "REL-4447050D23EBC97B";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-4447050D23EBC97B-context' {
        part 'FU-REVIEW-ORCHESTRATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-BEDROCK-PR-CONTEXT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-4447050D23EBC97B' from 'FU-REVIEW-ORCHESTRATION'.pOut to 'DO-BEDROCK-PR-CONTEXT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-4CE3A21523083A6C";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-4CE3A21523083A6C' from 'FU-SECRETS-CONFIGURATION' to 'REF-AWS-SDK-GO';

    @EngineeringRelationship {
        sourceId = "REL-4D8523D5017B3C2F";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-4D8523D5017B3C2F-context' {
        part 'FU-REVIEW-ORCHESTRATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-BEDROCK-REVIEW-FINDINGS' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-4D8523D5017B3C2F' from 'FU-REVIEW-ORCHESTRATION'.pOut to 'DO-BEDROCK-REVIEW-FINDINGS'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-4FDA2A168679615E";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-4FDA2A168679615E' from 'ACT-SECURITY-ENGINEER' to 'FU-POLICY-CHECKS';

    @EngineeringRelationship {
        sourceId = "REL-533ECBA0AEDE9E2D";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-533ECBA0AEDE9E2D' from 'FU-REVIEW-PUBLICATION' to 'REF-GITHUB-APP-API';

    @EngineeringRelationship {
        sourceId = "REL-536B406CDE959BBE";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-536B406CDE959BBE' from 'FU-GITHUB-WEBHOOK-INGRESS' to 'FU-PR-CONTEXT-ASSEMBLY';

    @EngineeringRelationship {
        sourceId = "REL-56F4F5EC26131FCD";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-56F4F5EC26131FCD' first 'assemble-context' then 'orchestrate-review';

    @EngineeringRelationship {
        sourceId = "REL-5A2554C609571E29";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-5A2554C609571E29' from 'FU-GITHUB-WEBHOOK-INGRESS' to 'TB-BEDROCK-GITHUB-EXTERNAL';

    @EngineeringRelationship {
        sourceId = "REL-6599FC4E055CB035";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-6599FC4E055CB035' from 'AV-SECRET-LEAKAGE-IN-REVIEW' to 'FU-SECRETS-CONFIGURATION';

    @EngineeringRelationship {
        sourceId = "REL-69DD8E6631353BA7";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-69DD8E6631353BA7' first 'orchestrate-review' then 'publish-review';

    @EngineeringRelationship {
        sourceId = "REL-6BCF17821CCFAC33";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-6BCF17821CCFAC33-context' {
        part 'FU-PR-CONTEXT-ASSEMBLY' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-BEDROCK-PR-CONTEXT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-6BCF17821CCFAC33' from 'FU-PR-CONTEXT-ASSEMBLY'.pOut to 'DO-BEDROCK-PR-CONTEXT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-6E8F05F5DB5995F1";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-6E8F05F5DB5995F1' from 'AV-SECRET-LEAKAGE-IN-REVIEW' to 'FU-REVIEW-PUBLICATION';

    @EngineeringRelationship {
        sourceId = "REL-73850787740C16CE";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-73850787740C16CE' from 'FU-REVIEW-ORCHESTRATION' to 'REF-AWS-BEDROCK-RUNTIME';

    @EngineeringRelationship {
        sourceId = "REL-755E1B28403A0747";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-755E1B28403A0747' from 'AV-PR-SPAM-ABUSE' to 'FU-GITHUB-WEBHOOK-INGRESS';

    @EngineeringRelationship {
        sourceId = "REL-75C5E9F6159148D1";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-75C5E9F6159148D1' from 'ACT-DEVELOPER' to 'FU-GITHUB-WEBHOOK-INGRESS';

    @EngineeringRelationship {
        sourceId = "REL-7639D9AB08E85D44";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-7639D9AB08E85D44' : EngineeringVerification {
        subject 'CV-BEDROCK-WEBHOOK-SIGNATURE' : EngineeringItem;
        objective 'CTRL-BEDROCK-WEBHOOK-SIGNATURE' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-78295979CC858BDA";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-78295979CC858BDA' from 'AV-PR-SPAM-ABUSE' to 'FU-REVIEW-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-7C1543790C9A3088";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-7C1543790C9A3088-context' {
        part 'FU-REVIEW-PUBLICATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'ACT-REPOSITORY-MAINTAINER' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-7C1543790C9A3088' from 'FU-REVIEW-PUBLICATION'.pOut to 'ACT-REPOSITORY-MAINTAINER'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-7E51F1BEC104913A";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-7E51F1BEC104913A' from 'ACT-REPOSITORY-MAINTAINER' to 'FU-REVIEW-PUBLICATION';

    @EngineeringRelationship {
        sourceId = "REL-846CBA69FC737721";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-846CBA69FC737721-context' {
        part 'ACT-DEVELOPER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-GITHUB-WEBHOOK-INGRESS' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-846CBA69FC737721' from 'ACT-DEVELOPER'.pOut to 'FU-GITHUB-WEBHOOK-INGRESS'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-8A8C6978030F8CCE";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-8A8C6978030F8CCE' from 'FU-POLICY-CHECKS' to 'REF-POLICY-RULESET';

    @EngineeringRelationship {
        sourceId = "REL-8C0131995B190EA2";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-8C0131995B190EA2' from 'FU-REVIEW-ORCHESTRATION' to 'FU-POLICY-CHECKS';

    @EngineeringRelationship {
        sourceId = "REL-8D667BCF628EE473";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-8D667BCF628EE473' from 'receive-webhook' to 'ACT-DEVELOPER';

    @EngineeringRelationship {
        sourceId = "REL-8E4E8F6B67850054";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-8E4E8F6B67850054' first 'run-policy-checks' then 'orchestrate-policy-validation';

    @EngineeringRelationship {
        sourceId = "REL-9852CBC8672CD860";
        conceptKind = "guard";
        relationshipKind = "guarded_by";
    }
    state 'REL-9852CBC8672CD860-context' : EngineeringState {
        action 'STATE-BEDROCK-REVIEW-PUBLISHED' : EngineeringAction;
        action 'CTRL-BEDROCK-WEBHOOK-SIGNATURE' : EngineeringAction;
        transition 'REL-9852CBC8672CD860' first 'STATE-BEDROCK-REVIEW-PUBLISHED' if true then 'CTRL-BEDROCK-WEBHOOK-SIGNATURE';
    }

    @EngineeringRelationship {
        sourceId = "REL-A0094CB4C8A39AF3";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-A0094CB4C8A39AF3' from 'FU-LAMBDA-RUNTIME-OPERATIONS' to 'TB-BEDROCK-AWS-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-A438006AD2D7E68A";
        conceptKind = "transfer";
        relationshipKind = "calls";
    }
    part def 'REL-A438006AD2D7E68A-context' {
        part 'FU-REVIEW-PUBLICATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-BEDROCK-REVIEW-PUBLISH-API' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-A438006AD2D7E68A' from 'FU-REVIEW-PUBLICATION'.pOut to 'IF-BEDROCK-REVIEW-PUBLISH-API'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-A4BDFBD0234931E5";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-A4BDFBD0234931E5' from 'FU-LAMBDA-RUNTIME-OPERATIONS' to 'REF-AWS-SECRETS-MANAGER';

    @EngineeringRelationship {
        sourceId = "REL-A6ADC0F19D9EF316";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-A6ADC0F19D9EF316' from 'FU-REVIEW-PUBLICATION' to 'FU-SECRETS-CONFIGURATION';

    @EngineeringRelationship {
        sourceId = "REL-AAFEDB4787AA69D6";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-AAFEDB4787AA69D6' : EngineeringVerification {
        subject 'CV-BEDROCK-REDACTION-CHECK' : EngineeringItem;
        objective 'CTRL-BEDROCK-SECRETS-ROTATION' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-B8284A03B39E62A9";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-B8284A03B39E62A9-context' {
        part 'FU-REVIEW-ORCHESTRATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-BEDROCK-REVIEW-FINDINGS' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-B8284A03B39E62A9' from 'FU-REVIEW-ORCHESTRATION'.pOut to 'DO-BEDROCK-REVIEW-FINDINGS'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-B8570BEDCC8A73E7";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-B8570BEDCC8A73E7' from 'ACT-PLATFORM-OPERATOR' to 'FU-SECRETS-CONFIGURATION';

    @EngineeringRelationship {
        sourceId = "REL-C200C04A94B2A61E";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-C200C04A94B2A61E-context' {
        part 'ACT-SECURITY-ENGINEER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-POLICY-CHECKS' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-C200C04A94B2A61E' from 'ACT-SECURITY-ENGINEER'.pOut to 'FU-POLICY-CHECKS'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-C5C9C7E606020A5E";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-C5C9C7E606020A5E-context' {
        part 'FU-PR-CONTEXT-ASSEMBLY' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-BEDROCK-PR-CONTEXT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-C5C9C7E606020A5E' from 'FU-PR-CONTEXT-ASSEMBLY'.pOut to 'DO-BEDROCK-PR-CONTEXT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-C732267FB0863670";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-C732267FB0863670' from 'orchestrate-policy-validation' to 'FU-REVIEW-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-C858D9D986152C4E";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-C858D9D986152C4E' from 'FU-PR-CONTEXT-ASSEMBLY' to 'REF-GITHUB-APP-API';

    @EngineeringRelationship {
        sourceId = "REL-CB5D923BB2D29D2B";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-CB5D923BB2D29D2B-context' {
        part 'ACT-SECURITY-ENGINEER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-REVIEW-PUBLICATION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-CB5D923BB2D29D2B' from 'ACT-SECURITY-ENGINEER'.pOut to 'FU-REVIEW-PUBLICATION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-D0A065D1DBF923B1";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-D0A065D1DBF923B1' from 'ACT-PLATFORM-OPERATOR' to 'FU-LAMBDA-RUNTIME-OPERATIONS';

    @EngineeringRelationship {
        sourceId = "REL-D4BB6E021F955308";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-D4BB6E021F955308-context' {
        part 'FU-POLICY-CHECKS' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-REVIEW-ORCHESTRATION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-D4BB6E021F955308' from 'FU-POLICY-CHECKS'.pOut to 'FU-REVIEW-ORCHESTRATION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-D5AB43DFC701F666";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-D5AB43DFC701F666' from 'AV-SPOOFED-GITHUB-WEBHOOK' to 'CTRL-BEDROCK-WEBHOOK-SIGNATURE';

    @EngineeringRelationship {
        sourceId = "REL-D6152466024D15AB";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-D6152466024D15AB-context' {
        part 'IF-BEDROCK-WEBHOOK-API' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-REVIEW-PUBLICATION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-D6152466024D15AB' from 'IF-BEDROCK-WEBHOOK-API'.pOut to 'FU-REVIEW-PUBLICATION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-D65E4D88BD201814";
        conceptKind = "guard";
        relationshipKind = "guarded_by";
    }
    state 'REL-D65E4D88BD201814-context' : EngineeringState {
        action 'STATE-BEDROCK-REVIEW-PUBLISHED' : EngineeringAction;
        action 'TB-BEDROCK-GITHUB-EXTERNAL' : EngineeringAction;
        transition 'REL-D65E4D88BD201814' first 'STATE-BEDROCK-REVIEW-PUBLISHED' if true then 'TB-BEDROCK-GITHUB-EXTERNAL';
    }

    @EngineeringRelationship {
        sourceId = "REL-DB86DF9906A1D827";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-DB86DF9906A1D827' from 'publish-review' to 'FU-REVIEW-PUBLICATION';

    @EngineeringRelationship {
        sourceId = "REL-DC0C521E72279E9A";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-DC0C521E72279E9A-context' {
        part 'FU-GITHUB-WEBHOOK-INGRESS' : EngineeringPart;
        part 'DEP-BEDROCK-LAMBDA-PRIMARY' : EngineeringPart;
        allocation 'REL-DC0C521E72279E9A' allocate 'FU-GITHUB-WEBHOOK-INGRESS' to 'DEP-BEDROCK-LAMBDA-PRIMARY';
    }

    @EngineeringRelationship {
        sourceId = "REL-E2AC935E2391F194";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-E2AC935E2391F194' from 'FU-LAMBDA-RUNTIME-OPERATIONS' to 'FU-SECRETS-CONFIGURATION';

    @EngineeringRelationship {
        sourceId = "REL-E2B80C5CA6ABE423";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-E2B80C5CA6ABE423' from 'FU-PR-CONTEXT-ASSEMBLY' to 'REF-GITHUB-REST-SDK';

    @EngineeringRelationship {
        sourceId = "REL-E912D1F0BE53B4DC";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-E912D1F0BE53B4DC' from 'AV-SPOOFED-GITHUB-WEBHOOK' to 'FU-GITHUB-WEBHOOK-INGRESS';

    @EngineeringRelationship {
        sourceId = "REL-EEFB0E127A694245";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-EEFB0E127A694245' first 'orchestrate-policy-validation' then 'publish-policy-validation';

    @EngineeringRelationship {
        sourceId = "REL-F1661B7712EBBEC7";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-F1661B7712EBBEC7' from 'FU-REVIEW-ORCHESTRATION' to 'FU-REVIEW-PUBLICATION';

    @EngineeringRelationship {
        sourceId = "REL-F2DE7A93BFE1AE2F";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-F2DE7A93BFE1AE2F-context' {
        part 'FU-REVIEW-ORCHESTRATION' : EngineeringPart;
        part 'DEP-BEDROCK-LAMBDA-PRIMARY' : EngineeringPart;
        allocation 'REL-F2DE7A93BFE1AE2F' allocate 'FU-REVIEW-ORCHESTRATION' to 'DEP-BEDROCK-LAMBDA-PRIMARY';
    }

    @EngineeringRelationship {
        sourceId = "REL-F497718B3C6B4414";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-F497718B3C6B4414' from 'FU-PR-CONTEXT-ASSEMBLY' to 'FU-REVIEW-ORCHESTRATION';

    @EngineeringRelationship {
        sourceId = "REL-F5FCD6192454ACCA";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-F5FCD6192454ACCA' first 'update-policy-rules' then 'run-policy-checks';

    @EngineeringRelationship {
        sourceId = "REL-F88E2DFFB7E56CDE";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-F88E2DFFB7E56CDE-context' {
        part 'FU-REVIEW-ORCHESTRATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'REF-AWS-BEDROCK-RUNTIME' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-F88E2DFFB7E56CDE' from 'FU-REVIEW-ORCHESTRATION'.pOut to 'REF-AWS-BEDROCK-RUNTIME'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-F9515179C6B8712E";
        conceptKind = "transfer";
        relationshipKind = "calls";
    }
    part def 'REL-F9515179C6B8712E-context' {
        part 'FU-GITHUB-WEBHOOK-INGRESS' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-BEDROCK-WEBHOOK-API' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-F9515179C6B8712E' from 'FU-GITHUB-WEBHOOK-INGRESS'.pOut to 'IF-BEDROCK-WEBHOOK-API'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-F9A074B485710557";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-F9A074B485710557-context' {
        part 'IF-BEDROCK-REVIEW-PUBLISH-API' : EngineeringPart;
        part 'DEP-BEDROCK-GITHUB-BOUNDARY' : EngineeringPart;
        allocation 'REL-F9A074B485710557' allocate 'IF-BEDROCK-REVIEW-PUBLISH-API' to 'DEP-BEDROCK-GITHUB-BOUNDARY';
    }

    @EngineeringRelationship {
        sourceId = "REL-FF0744F0B9A3D5DA";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-FF0744F0B9A3D5DA' from 'AV-SECRET-LEAKAGE-IN-REVIEW' to 'CTRL-BEDROCK-SECRETS-ROTATION';
}
