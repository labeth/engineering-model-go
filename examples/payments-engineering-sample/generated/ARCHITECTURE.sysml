// Generated deterministic SysML v2 project source.
// Syntax is validated against SysML 2.0 / KerML 1.0 with the pinned 2026-04 Pilot Implementation.
package 'sample-payments-layered-model' {
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
        sourceId = "sample-payments-layered-model";
        conceptKind = "engineering_model";
    }
    attribute 'engineeringModelInterchange';

    @EngineeringElement {
        sourceId = "ACT-CUSTOMER";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-CUSTOMER';

    @EngineeringElement {
        sourceId = "ACT-PLATFORM-OPERATOR";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-PLATFORM-OPERATOR';

    @EngineeringElement {
        sourceId = "ACT-SUPPORT";
        conceptKind = "stakeholder_definition";
    }
    item def 'ACT-SUPPORT';

    @EngineeringElement {
        sourceId = "ADR-PAY-001";
        conceptKind = "engineering_extension";
    }
    attribute 'ADR-PAY-001';

    @EngineeringElement {
        sourceId = "ASM-PAYMENTS-BANK-SIGNING-VALID";
        conceptKind = "engineering_extension";
    }
    attribute 'ASM-PAYMENTS-BANK-SIGNING-VALID';

    @EngineeringElement {
        sourceId = "AV-COMPROMISED-DEPENDENCY";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-COMPROMISED-DEPENDENCY';

    @EngineeringElement {
        sourceId = "AV-FRAUDULENT-TRANSACTION-PATTERN";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-FRAUDULENT-TRANSACTION-PATTERN';

    @EngineeringElement {
        sourceId = "AV-MALICIOUS-API-REQUEST";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-MALICIOUS-API-REQUEST';

    @EngineeringElement {
        sourceId = "AV-REPLAYED-AUTH-CALLBACK";
        conceptKind = "engineering_extension";
    }
    attribute 'AV-REPLAYED-AUTH-CALLBACK';

    @EngineeringElement {
        sourceId = "CONCERN-00255DF8D04CCDDA";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-00255DF8D04CCDDA' : EngineeringConcern {
        attribute 'narrative' : 'String' = "risk scoring ownership is kept at module/package boundary for policy accountability.\nFine-grained traces support requirements, but ownership remains coarse to avoid architectural churn.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-01247283638ABDBC";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-01247283638ABDBC' : EngineeringConcern {
        attribute 'narrative' : 'String' = "fraud evaluation runs as a critical dependency on the authorization path.\nRuntime behavior must return stable contracts quickly while preserving enough evidence for operator\ndiagnosis and audit when high-risk classifications occur.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-04798A81FEBAF51E";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-04798A81FEBAF51E' : EngineeringConcern {
        attribute 'narrative' : 'String' = "gitops operations security protects source trust and reconciliation authority.\nThe design prevents unauthorized state changes and enforces policy before runtime apply occurs.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-08479963A5090BE8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-08479963A5090BE8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "checkout handling is the ingress runtime boundary and forwards normalized requests downstream.\nRuntime design emphasizes stable request contracts, context propagation, and graceful behavior under\nburst traffic and partial downstream degradation.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-0EB64B8113A2CE37";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-0EB64B8113A2CE37' : EngineeringConcern {
        attribute 'narrative' : 'String' = "risk scoring security assumes adversarial input attempts and model manipulation pressure.\nDesign controls enforce strict input handling and evidence capture for post-event investigation.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-15BD72088803A101";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-15BD72088803A101' : EngineeringConcern {
        attribute 'narrative' : 'String' = "gitops operations ownership is anchored at GitOps configuration and automation module boundaries.\nThis keeps delivery accountability explicit while allowing manifest-level iteration.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-1E793CD743602FD1";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-1E793CD743602FD1' : EngineeringConcern {
        attribute 'narrative' : 'String' = "fraud evaluation defines how risk evidence is transformed into actionable payment decisions.\nIt separates automated risk scoring from human support review so escalation paths are explicit,\nexplainable, and policy-constrained.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-257CCBBB33FB45D9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-257CCBBB33FB45D9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payment authorization ownership resides at the payment engine orchestration boundary.\nTeams may evolve internal algorithms, but ownership of orchestration behavior remains explicit and durable.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-27BB70B1EE7347C8";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-27BB70B1EE7347C8' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payments ownership is defined at package and module boundaries for long-term architectural stability.\nInternal symbols can evolve rapidly, but ownership remains aligned to functional units for predictable\naccountability and maintainable change ownership.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-2FDA35B8A09116DD";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-2FDA35B8A09116DD' : EngineeringConcern {
        attribute 'narrative' : 'String' = "fraud evaluation security focuses on decision integrity and review integrity.\nThe design assumes attempted manipulation of risk inputs, dependency trust, and manual review workflows,\nand therefore requires strong evidence lineage and tamper-evident operator actions.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-363603E5C86A38B2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-363603E5C86A38B2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "cluster provisioning defines the repeatable environment baseline for all runtime workloads.\nFunctional design establishes deterministic creation of cluster and namespace structure.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-4453CE23F7403FCD";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-4453CE23F7403FCD' : EngineeringConcern {
        attribute 'narrative' : 'String' = "support review security requires strong operator identity controls and tamper-evident audit trails.\nThe design prevents silent overrides and guarantees every manual decision is attributable.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-44C44E0CAE94F88D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-44C44E0CAE94F88D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "fraud evaluation deployment is tuned for cautious rollout because scoring changes can alter approval rates.\nDeployment design favors staged promotions and rapid monitoring of scoring outcomes before wider exposure.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-45F0E2103FDA3D3A";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-45F0E2103FDA3D3A' : EngineeringConcern {
        attribute 'narrative' : 'String' = "platform deployment design governs how infrastructure intent is applied and kept in sync over time.\nIt prioritizes deterministic sequencing, drift control, and recoverable change paths before application\nteams depend on new platform states.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-50758E668379FA08";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-50758E668379FA08' : EngineeringConcern {
        attribute 'narrative' : 'String' = "cluster provisioning influences runtime isolation and blast-radius boundaries indirectly but critically.\nRuntime design ensures containment primitives are present before workload traffic depends on them.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-50A4CB678FC9C092";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-50A4CB678FC9C092' : EngineeringConcern {
        attribute 'narrative' : 'String' = "risk scoring produces risk classifications that are actionable for authorization policy.\nFunctional design requires predictable threshold behavior and explainable outputs for operators.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-51F6E1462A93E11F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-51F6E1462A93E11F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "cluster provisioning ownership belongs to infrastructure code modules and provisioning definitions.\nOwnership remains stable across provider changes to preserve platform accountability.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5C3E0E192D984752";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5C3E0E192D984752' : EngineeringConcern {
        attribute 'narrative' : 'String' = "risk scoring deployment uses staged rollout because scoring changes have immediate decision impact.\nDeployment design includes observation windows for score distribution and escalation-rate changes.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-5C567A5DD951E34B";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-5C567A5DD951E34B' : EngineeringConcern {
        attribute 'narrative' : 'String' = "checkout handling is released independently to protect user-facing velocity without destabilizing policy logic.\nDeployment design favors fast rollback and compatibility checks with the current authorization interface.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-68818A97973CC591";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-68818A97973CC591' : EngineeringConcern {
        attribute 'narrative' : 'String' = "cluster provisioning security sets inherited baseline hardening and isolation defaults.\nDesign must detect and remediate security drift early because downstream controls depend on this layer.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6B5C82294DEACF48";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6B5C82294DEACF48' : EngineeringConcern {
        attribute 'narrative' : 'String' = "fraud evaluation ownership is anchored to scoring and review modules rather than symbol-level granularity.\nThis keeps policy accountability stable while enabling frequent implementation changes inside owned modules.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6F1D1EE2C39B3CDF";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6F1D1EE2C39B3CDF' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payment authorization coordinates critical runtime dependencies and enforces decision-order determinism.\nRuntime design prioritizes idempotent transitions, explicit branch visibility, and controlled error normalization.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-6F658D9A8966BBFB";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-6F658D9A8966BBFB' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payments protects public trust boundaries where untrusted input and external callbacks enter the system.\nSecurity design focuses on strict validation, replay resistance, and safe failure behavior that preserves\ncustomer trust during degraded dependency conditions.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-732F50A2F585E635";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-732F50A2F585E635' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payments defines the business capability that turns customer intent into a clear payment outcome.\nIt sets the responsibilities between checkout handling and payment authorization, and defines where\napproval, decline, and fallback decisions are made.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-75B19227BB1CE2BF";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-75B19227BB1CE2BF' : EngineeringConcern {
        attribute 'narrative' : 'String' = "cluster provisioning deployment is infrastructure-first and sequence-sensitive.\nDeployment design requires deterministic change plans, drift awareness, and rollback strategy before apply.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-7876868CED602A0F";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-7876868CED602A0F' : EngineeringConcern {
        attribute 'narrative' : 'String' = "gitops operations defines how declared release intent becomes enforced runtime state.\nFunctional design ensures delivery behavior is predictable and drift is corrected consistently.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-796710C3E04A837B";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-796710C3E04A837B' : EngineeringConcern {
        attribute 'narrative' : 'String' = "support review provides controlled human decision flow for escalated transactions.\nFunctional design ensures review actions are policy-bounded and outcome transitions are explicit.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-82C474BE4FD40F51";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-82C474BE4FD40F51' : EngineeringConcern {
        attribute 'narrative' : 'String' = "platform provides the foundational capabilities required for all product workloads to run safely.\nIt intentionally models provisioning and reconciliation as first-class architecture responsibilities,\nnot background implementation detail.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8A26204A21D44CC9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8A26204A21D44CC9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "support review runtime optimizes for operator clarity and decision traceability over throughput.\nRuntime design emphasizes state visibility, reversible transitions where allowed, and reliable action logging.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8E71DA793D4F6058";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8E71DA793D4F6058' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payment authorization defines the decision policy orchestration for each transaction.\nIt combines fraud input, bank outcomes, and escalation rules into one deterministic business decision.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-8EF92891DD0FFD1D";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-8EF92891DD0FFD1D' : EngineeringConcern {
        attribute 'narrative' : 'String' = "checkout handling owns customer-facing request normalization and response semantics.\nFunctional design ensures the customer receives a coherent outcome regardless of approval, decline,\nor temporary dependency failure conditions.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-937944781657FAE2";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-937944781657FAE2' : EngineeringConcern {
        attribute 'narrative' : 'String' = "platform security design establishes baseline controls inherited by all higher-layer units.\nIt focuses on trusted reconciliation, isolation defaults, and policy enforcement points where compromise\nwould have broad systemic impact.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-A6DAABC8C7B1B7FD";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-A6DAABC8C7B1B7FD' : EngineeringConcern {
        attribute 'narrative' : 'String' = "platform runtime design establishes containment and control-loop behavior for workload operation.\nIt defines how namespaces, clusters, and reconciler processes create predictable runtime boundaries\nthat reduce operational ambiguity during incidents.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-ACE244AC98D3F390";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-ACE244AC98D3F390' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payment authorization deployment is policy-sensitive and therefore rollout-cautious.\nReleases should stage gradually, verify dependency health, and preserve in-flight transaction continuity.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B215039CFB464EFD";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B215039CFB464EFD' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payments runs the public request path and keeps transaction context intact across runtime hops.\nThe runtime design emphasizes low latency, deterministic request sequencing, and clear operational\nsignals when dependencies slow down or fail.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B69EBE749FD87D01";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B69EBE749FD87D01' : EngineeringConcern {
        attribute 'narrative' : 'String' = "platform ownership is centered on infrastructure and GitOps configuration boundaries.\nThis enables platform teams to evolve delivery mechanics independently while preserving explicit ownership\ncontracts for operational behavior.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-B7ED614A677A4660";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-B7ED614A677A4660' : EngineeringConcern {
        attribute 'narrative' : 'String' = "gitops operations runs reconciliation loops that continuously compare desired and observed state.\nRuntime design requires clear reconciliation signals so teams can reason about rollout and recovery behavior.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-BD9517D3C6BB27D3";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-BD9517D3C6BB27D3' : EngineeringConcern {
        attribute 'narrative' : 'String' = "checkout handling security protects the internet-facing ingress.\nThe design requires strict validation, abuse resistance, and traceable rejection behavior so malicious inputs\nare contained without harming legitimate customer flows.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C5914532DC0AA944";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C5914532DC0AA944' : EngineeringConcern {
        attribute 'narrative' : 'String' = "support review ownership is maintained at workflow-module boundary.\nShared helper code may exist, but accountability for review flow and outcome transitions remains explicit.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-C799248825238C0E";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-C799248825238C0E' : EngineeringConcern {
        attribute 'narrative' : 'String' = "checkout handling ownership is maintained at coarse package/module scope.\nThis keeps architecture ownership stable across internal refactors while preserving fine-grained symbol traces\nfor requirement evidence.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-CF7B4201A2610FB9";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-CF7B4201A2610FB9' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payments is deployed so checkout and authorization can roll independently with minimal blast radius.\nDeployment design prioritizes safe promotion, fast rollback, and dependency readiness checks before\npolicy-impacting releases are exposed.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-E84BCE706342B142";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-E84BCE706342B142' : EngineeringConcern {
        attribute 'narrative' : 'String' = "gitops operations deployment defines the release control plane for application artifacts.\nDeployment design prioritizes policy gates, controlled promotion, and resilient rollback mechanics.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F233F0C0AD608ADA";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F233F0C0AD608ADA' : EngineeringConcern {
        attribute 'narrative' : 'String' = "risk scoring runs as a synchronous dependency with strict response-shape stability.\nRuntime design favors deterministic outputs, bounded latency, and clear degradation semantics for callers.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F5879ADDFB5DEB54";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F5879ADDFB5DEB54' : EngineeringConcern {
        attribute 'narrative' : 'String' = "payment authorization security protects decision integrity and callback integrity.\nThe design defends against replay, policy probing, and ambiguous error channels that could be abused.";
    }

    @EngineeringElement {
        sourceId = "CONCERN-F8A06370D5E7F2A4";
        conceptKind = "concern_usage";
    }
    concern 'CONCERN-F8A06370D5E7F2A4' : EngineeringConcern {
        attribute 'narrative' : 'String' = "support review deployment isolates privileged workflows from public ingress components.\nDeployment design must preserve active review sessions across upgrades and ensure queue/review latency visibility.";
    }

    @EngineeringElement {
        sourceId = "COND-3DS-REQUIRED";
        conceptKind = "engineering_extension";
    }
    attribute 'COND-3DS-REQUIRED';

    @EngineeringElement {
        sourceId = "COND-RISK-THRESHOLD-EXCEEDED";
        conceptKind = "engineering_extension";
    }
    attribute 'COND-RISK-THRESHOLD-EXCEEDED';

    @EngineeringElement {
        sourceId = "CTRL-PAYMENTS-CALLBACK-NONCE";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-PAYMENTS-CALLBACK-NONCE';

    @EngineeringElement {
        sourceId = "CTRL-PAYMENTS-IMAGE-DIGEST";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-PAYMENTS-IMAGE-DIGEST';

    @EngineeringElement {
        sourceId = "CTRL-PAYMENTS-SSO-MFA";
        conceptKind = "engineering_extension";
    }
    attribute 'CTRL-PAYMENTS-SSO-MFA';

    @EngineeringElement {
        sourceId = "CV-PAYMENTS-CALLBACK-REPLAY-TEST";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-PAYMENTS-CALLBACK-REPLAY-TEST' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-PAYMENTS-CHECKOUT-SCHEMA-TEST";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-PAYMENTS-CHECKOUT-SCHEMA-TEST' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "CV-PAYMENTS-MANUAL-REVIEW-TRACE";
        conceptKind = "verification_case_usage";
    }
    verification 'CV-PAYMENTS-MANUAL-REVIEW-TRACE' : EngineeringVerification;

    @EngineeringElement {
        sourceId = "DATA-DECLINE-REASON";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-DECLINE-REASON';

    @EngineeringElement {
        sourceId = "DATA-PAYMENT-ID";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-PAYMENT-ID';

    @EngineeringElement {
        sourceId = "DATA-RISK-SCORE";
        conceptKind = "engineering_extension";
    }
    attribute 'DATA-RISK-SCORE';

    @EngineeringElement {
        sourceId = "DEP-PAYMENTS-BANK-EDGE";
        conceptKind = "part_definition";
    }
    part def 'DEP-PAYMENTS-BANK-EDGE' {
        attribute 'environment' : 'String' = "prod";
        attribute 'region' : 'String' = "us-east-1";
        attribute 'namespace' : 'String' = "partner";
        attribute 'trustZone' : 'String' = "external";
    }

    @EngineeringElement {
        sourceId = "DEP-PAYMENTS-CLUSTER-PROD";
        conceptKind = "part_definition";
    }
    part def 'DEP-PAYMENTS-CLUSTER-PROD' {
        attribute 'environment' : 'String' = "prod";
        attribute 'region' : 'String' = "us-east-1";
        attribute 'namespace' : 'String' = "payments";
        attribute 'trustZone' : 'String' = "app";
    }

    @EngineeringElement {
        sourceId = "DO-PAYMENTS-AUTH-DECISION";
        conceptKind = "item_definition";
    }
    item def 'DO-PAYMENTS-AUTH-DECISION' {
        attribute 'schemaRef' : 'String' = "schemas/auth-decision.json";
        attribute 'classification' : 'String' = "regulated-internal";
    }

    @EngineeringElement {
        sourceId = "DO-PAYMENTS-AUTH-REQUEST";
        conceptKind = "item_definition";
    }
    item def 'DO-PAYMENTS-AUTH-REQUEST' {
        attribute 'schemaRef' : 'String' = "schemas/auth-request.json";
        attribute 'classification' : 'String' = "pci";
    }

    @EngineeringElement {
        sourceId = "DO-PAYMENTS-REVIEW-TICKET";
        conceptKind = "item_definition";
    }
    item def 'DO-PAYMENTS-REVIEW-TICKET' {
        attribute 'schemaRef' : 'String' = "schemas/review-ticket.json";
        attribute 'classification' : 'String' = "pii";
    }

    @EngineeringElement {
        sourceId = "DO-PAYMENTS-RISK-SIGNAL";
        conceptKind = "item_definition";
    }
    item def 'DO-PAYMENTS-RISK-SIGNAL' {
        attribute 'schemaRef' : 'String' = "schemas/risk-signal.json";
        attribute 'classification' : 'String' = "fraud-analytics";
    }

    @EngineeringElement {
        sourceId = "EVIDENCE-030F7A8321E0E90A";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-030F7A8321E0E90A';

    @EngineeringElement {
        sourceId = "EVIDENCE-145F11A38FF1EE40";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-145F11A38FF1EE40';

    @EngineeringElement {
        sourceId = "EVIDENCE-14CB996CB354A8A5";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-14CB996CB354A8A5';

    @EngineeringElement {
        sourceId = "EVIDENCE-26157E4B5D71D7CC";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-26157E4B5D71D7CC';

    @EngineeringElement {
        sourceId = "EVIDENCE-42B18A505E8D0796";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-42B18A505E8D0796';

    @EngineeringElement {
        sourceId = "EVIDENCE-47F5C5B827C32279";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-47F5C5B827C32279';

    @EngineeringElement {
        sourceId = "EVIDENCE-61189B4F37FE22CD";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-61189B4F37FE22CD';

    @EngineeringElement {
        sourceId = "EVIDENCE-6D53BD82C1D45AC1";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-6D53BD82C1D45AC1';

    @EngineeringElement {
        sourceId = "EVIDENCE-700609D8B15014C1";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-700609D8B15014C1';

    @EngineeringElement {
        sourceId = "EVIDENCE-747BD3648346B1AA";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-747BD3648346B1AA';

    @EngineeringElement {
        sourceId = "EVIDENCE-7E241CA162BD74E4";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-7E241CA162BD74E4';

    @EngineeringElement {
        sourceId = "EVIDENCE-7F6C6CCE1AD9D2BF";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-7F6C6CCE1AD9D2BF';

    @EngineeringElement {
        sourceId = "EVIDENCE-AA6DBE1FE6258EF1";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-AA6DBE1FE6258EF1';

    @EngineeringElement {
        sourceId = "EVIDENCE-B8B8B48BCA1DDEBA";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-B8B8B48BCA1DDEBA';

    @EngineeringElement {
        sourceId = "EVIDENCE-BB1B20F7EAEFC9F3";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-BB1B20F7EAEFC9F3';

    @EngineeringElement {
        sourceId = "EVIDENCE-C746E9677F001C9E";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-C746E9677F001C9E';

    @EngineeringElement {
        sourceId = "EVIDENCE-C8455D1C63E6F146";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-C8455D1C63E6F146';

    @EngineeringElement {
        sourceId = "EVIDENCE-CCAE3F3E45678B61";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-CCAE3F3E45678B61';

    @EngineeringElement {
        sourceId = "EVIDENCE-D2EC2549AECC7A07";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-D2EC2549AECC7A07';

    @EngineeringElement {
        sourceId = "EVIDENCE-D9297EEF95469ED3";
        conceptKind = "engineering_extension";
    }
    attribute 'EVIDENCE-D9297EEF95469ED3';

    @EngineeringElement {
        sourceId = "EVT-BANK-LINK-UNAVAILABLE";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-BANK-LINK-UNAVAILABLE';

    @EngineeringElement {
        sourceId = "EVT-PAYMENT-AUTH-REQUESTED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-PAYMENT-AUTH-REQUESTED';

    @EngineeringElement {
        sourceId = "EVT-PAYMENT-DECLINED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-PAYMENT-DECLINED';

    @EngineeringElement {
        sourceId = "EVT-PAYMENTS-AUTH-COMPLETED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-PAYMENTS-AUTH-COMPLETED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-PAYMENTS-AUTH-REQUESTED";
        conceptKind = "event_definition";
    }
    event occurrence 'EVT-PAYMENTS-AUTH-REQUESTED' : EngineeringEvent;

    @EngineeringElement {
        sourceId = "EVT-RISK-HIGH";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-RISK-HIGH';

    @EngineeringElement {
        sourceId = "EVT-SUPPORT-NOTIFIED";
        conceptKind = "engineering_extension";
    }
    attribute 'EVT-SUPPORT-NOTIFIED';

    @EngineeringElement {
        sourceId = "EXTENSION-25A8104FBFAB75B2";
        conceptKind = "engineering_extension";
    }
    attribute 'EXTENSION-25A8104FBFAB75B2';

    @EngineeringElement {
        sourceId = "FEAT-FRAUD-CHECK";
        conceptKind = "engineering_extension";
    }
    attribute 'FEAT-FRAUD-CHECK';

    @EngineeringElement {
        sourceId = "FG-FRAUD";
        conceptKind = "part_definition";
    }
    part def 'FG-FRAUD';

    @EngineeringElement {
        sourceId = "FG-PAYMENTS";
        conceptKind = "part_definition";
    }
    part def 'FG-PAYMENTS';

    @EngineeringElement {
        sourceId = "FG-PLATFORM";
        conceptKind = "part_definition";
    }
    part def 'FG-PLATFORM';

    @EngineeringElement {
        sourceId = "FLOW-CUSTOMER-CHECKOUT";
        conceptKind = "action_definition";
    }
    action def 'FLOW-CUSTOMER-CHECKOUT' {
        inout item 'item_DO_PAYMENTS_AUTH_REQUEST' : 'DO-PAYMENTS-AUTH-REQUEST';
        inout item 'item_DO_PAYMENTS_AUTH_DECISION' : 'DO-PAYMENTS-AUTH-DECISION';
    }

    @EngineeringElement {
        sourceId = "FLOW-PAYMENTS-MANUAL-REVIEW";
        conceptKind = "action_definition";
    }
    action def 'FLOW-PAYMENTS-MANUAL-REVIEW' {
        inout item 'item_DO_PAYMENTS_RISK_SIGNAL' : 'DO-PAYMENTS-RISK-SIGNAL';
        inout item 'item_DO_PAYMENTS_REVIEW_TICKET' : 'DO-PAYMENTS-REVIEW-TICKET';
    }

    @EngineeringElement {
        sourceId = "FU-CHECKOUT";
        conceptKind = "part_usage";
    }
    part 'FU-CHECKOUT' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-CLUSTER-PROVISIONING";
        conceptKind = "part_usage";
    }
    part 'FU-CLUSTER-PROVISIONING' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-GITOPS-OPERATIONS";
        conceptKind = "part_usage";
    }
    part 'FU-GITOPS-OPERATIONS' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-PAYMENT-AUTHORIZATION";
        conceptKind = "part_usage";
    }
    part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-RISK-SCORING";
        conceptKind = "part_usage";
    }
    part 'FU-RISK-SCORING' : EngineeringPart;

    @EngineeringElement {
        sourceId = "FU-SUPPORT-REVIEW";
        conceptKind = "part_usage";
    }
    part 'FU-SUPPORT-REVIEW' : EngineeringPart;

    @EngineeringElement {
        sourceId = "IF-PAYMENTS-BANK-AUTH";
        conceptKind = "interface_definition";
    }
    interface def 'IF-PAYMENTS-BANK-AUTH' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "/bank/authorize";
        attribute 'schemaRef' : 'String' = "schemas/bank-auth.json";
    }

    @EngineeringElement {
        sourceId = "IF-PAYMENTS-CHECKOUT-API";
        conceptKind = "interface_definition";
    }
    interface def 'IF-PAYMENTS-CHECKOUT-API' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "/api/checkout";
        attribute 'schemaRef' : 'String' = "schemas/checkout-request.json";
    }

    @EngineeringElement {
        sourceId = "IF-PAYMENTS-RISK-SCORE-API";
        conceptKind = "interface_definition";
    }
    interface def 'IF-PAYMENTS-RISK-SCORE-API' {
        end port 'source' : EngineeringPort;
        end port 'target' : EngineeringPort;
        attribute 'protocol' : 'String' = "https";
        attribute 'endpoint' : 'String' = "/risk/score";
        attribute 'schemaRef' : 'String' = "schemas/risk-score.json";
    }

    @EngineeringElement {
        sourceId = "MAP-PAYMENTS-IMAGE-DIGEST";
        conceptKind = "engineering_extension";
    }
    attribute 'MAP-PAYMENTS-IMAGE-DIGEST';

    @EngineeringElement {
        sourceId = "MAP-PAYMENTS-SSO-MFA";
        conceptKind = "engineering_extension";
    }
    attribute 'MAP-PAYMENTS-SSO-MFA';

    @EngineeringElement {
        sourceId = "MIT-PAYMENTS-CALLBACK-NONCE-CHECK";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-PAYMENTS-CALLBACK-NONCE-CHECK';

    @EngineeringElement {
        sourceId = "MIT-PAYMENTS-CHECKOUT-STRICT-SCHEMA";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-PAYMENTS-CHECKOUT-STRICT-SCHEMA';

    @EngineeringElement {
        sourceId = "MIT-PAYMENTS-MANUAL-REVIEW-AUDIT";
        conceptKind = "engineering_extension";
    }
    attribute 'MIT-PAYMENTS-MANUAL-REVIEW-AUDIT';

    @EngineeringElement {
        sourceId = "MODE-NORMAL";
        conceptKind = "engineering_extension";
    }
    attribute 'MODE-NORMAL';

    @EngineeringElement {
        sourceId = "OOS-PAYMENTS-BANK-HSM-OPERATIONS";
        conceptKind = "engineering_extension";
    }
    attribute 'OOS-PAYMENTS-BANK-HSM-OPERATIONS';

    @EngineeringElement {
        sourceId = "OOS-PAYMENTS-SI-10-LOW-BASELINE";
        conceptKind = "engineering_extension";
    }
    attribute 'OOS-PAYMENTS-SI-10-LOW-BASELINE';

    @EngineeringElement {
        sourceId = "POAM-PAYMENTS-SSO-LEGACY-CLEANUP";
        conceptKind = "engineering_extension";
    }
    attribute 'POAM-PAYMENTS-SSO-LEGACY-CLEANUP';

    @EngineeringElement {
        sourceId = "PROF-NIST-800-53-LOW";
        conceptKind = "engineering_extension";
    }
    attribute 'PROF-NIST-800-53-LOW';

    @EngineeringElement {
        sourceId = "REF-BANK-GATEWAY-ENDPOINT";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-BANK-GATEWAY-ENDPOINT';

    @EngineeringElement {
        sourceId = "REF-HELM-PLATFORM";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-HELM-PLATFORM';

    @EngineeringElement {
        sourceId = "REF-POSTGRES-DRIVER";
        conceptKind = "engineering_extension";
    }
    attribute 'REF-POSTGRES-DRIVER';

    @EngineeringElement {
        sourceId = "REQ-PAY-001";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-001' {
        attribute 'text' : 'String' = "While normal mode is active, when payment authorization is requested, the payments system shall create a payment session.";
        attribute 'notes' : 'String' = "Core authorization path from checkout to bank gateway.";
    }

    @EngineeringElement {
        sourceId = "REQ-PAY-002";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-002' {
        attribute 'text' : 'String' = "Where fraud check is enabled, when payment authorization is requested, the fraud detection system shall calculate a risk score before approval.";
        attribute 'notes' : 'String' = "Fraud scoring gate before approval.";
    }

    @EngineeringElement {
        sourceId = "REQ-PAY-003";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-003' {
        attribute 'text' : 'String' = "If payment is declined, then the payments system shall present a decline reason to customer.";
        attribute 'notes' : 'String' = "Decline handling and customer feedback.";
    }

    @EngineeringElement {
        sourceId = "REQ-PAY-004";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-004' {
        attribute 'text' : 'String' = "While checkout is active, when payment authorization is requested and risk score is high and risk threshold is exceeded, the payments system shall place the payment in payment review is pending.";
        attribute 'notes' : 'String' = "Escalation flow for high-risk payments.";
    }

    @EngineeringElement {
        sourceId = "REQ-PAY-005";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-005' {
        attribute 'text' : 'String' = "Where fraud check is enabled, when payment authorization is requested, the payments system shall persist an audit record with payment id plus risk score.";
        attribute 'notes' : 'String' = "Auditability for authorization decisions.";
    }

    @EngineeringElement {
        sourceId = "REQ-PAY-006";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-006' {
        attribute 'text' : 'String' = "If bank link is unavailable, then the payments system shall present a temporary unavailable message to customer until bank link is available.";
        attribute 'notes' : 'String' = "Fallback behavior when external banking path is down.";
    }

    @EngineeringElement {
        sourceId = "REQ-PAY-007";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-007' {
        attribute 'text' : 'String' = "While checkout is active, when payment authorization is requested and 3DS verification is required, the payments system shall request authorization from bank gateway endpoint.";
        attribute 'notes' : 'String' = "External authorization call contract.";
    }

    @EngineeringElement {
        sourceId = "REQ-PAY-008";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-008' {
        attribute 'text' : 'String' = "If payment is declined, then the payments system shall present a retry option to customer.";
        attribute 'notes' : 'String' = "Customer retry path requirement after decline.";
    }

    @EngineeringElement {
        sourceId = "REQ-PAY-009";
        conceptKind = "requirement_definition";
    }
    requirement def 'REQ-PAY-009' {
        attribute 'text' : 'String' = "While checkout is active, when payment authorization is requested and risk threshold is exceeded, the payments system shall notify platform operator when support agent is notified.";
        attribute 'notes' : 'String' = "Support notification requirement for high-risk review.";
    }

    @EngineeringElement {
        sourceId = "RISK-PAYMENTS-BANK-CALLBACK-REPLAY";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-PAYMENTS-BANK-CALLBACK-REPLAY';

    @EngineeringElement {
        sourceId = "RISK-PAYMENTS-CHECKOUT-SPOOFING";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-PAYMENTS-CHECKOUT-SPOOFING';

    @EngineeringElement {
        sourceId = "RISK-PAYMENTS-MANUAL-REVIEW-TAMPER";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-PAYMENTS-MANUAL-REVIEW-TAMPER';

    @EngineeringElement {
        sourceId = "RISK-PAYMENTS-SSO-MFA-GAP";
        conceptKind = "engineering_extension";
    }
    attribute 'RISK-PAYMENTS-SSO-MFA-GAP';

    @EngineeringElement {
        sourceId = "STATE-BANK-LINK-UP";
        conceptKind = "engineering_extension";
    }
    attribute 'STATE-BANK-LINK-UP';

    @EngineeringElement {
        sourceId = "STATE-CHECKOUT-ACTIVE";
        conceptKind = "engineering_extension";
    }
    attribute 'STATE-CHECKOUT-ACTIVE';

    @EngineeringElement {
        sourceId = "STATE-PAYMENTS-AUTHORIZED";
        conceptKind = "state_definition";
    }
    state def 'STATE-PAYMENTS-AUTHORIZED';

    @EngineeringElement {
        sourceId = "STATE-PAYMENTS-CHECKOUT-RECEIVED";
        conceptKind = "state_definition";
    }
    state def 'STATE-PAYMENTS-CHECKOUT-RECEIVED';

    @EngineeringElement {
        sourceId = "STATE-REVIEW-PENDING";
        conceptKind = "engineering_extension";
    }
    attribute 'STATE-REVIEW-PENDING';

    @EngineeringElement {
        sourceId = "SYS-FRAUD-DETECTION-SYSTEM";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-FRAUD-DETECTION-SYSTEM';

    @EngineeringElement {
        sourceId = "SYS-PAYMENTS-SYSTEM";
        conceptKind = "engineering_extension";
    }
    attribute 'SYS-PAYMENTS-SYSTEM';

    @EngineeringElement {
        sourceId = "TB-PAYMENTS-EXTERNAL-BANK";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-PAYMENTS-EXTERNAL-BANK';

    @EngineeringElement {
        sourceId = "TB-PAYMENTS-PLATFORM-CONTROL";
        conceptKind = "engineering_extension";
    }
    attribute 'TB-PAYMENTS-PLATFORM-CONTROL';

    @EngineeringElement {
        sourceId = "TS-PAYMENTS-BANK-CALLBACK-REPLAY";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-PAYMENTS-BANK-CALLBACK-REPLAY';

    @EngineeringElement {
        sourceId = "TS-PAYMENTS-CHECKOUT-SPOOFING";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-PAYMENTS-CHECKOUT-SPOOFING';

    @EngineeringElement {
        sourceId = "TS-PAYMENTS-RISK-TICKET-TAMPER";
        conceptKind = "engineering_extension";
    }
    attribute 'TS-PAYMENTS-RISK-TICKET-TAMPER';

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
        sourceId = "authorize-payment";
        conceptKind = "action_usage";
    }
    action 'authorize-payment' : EngineeringAction {
        in item 'input_normalized_payment_request' : EngineeringItem;
        out item 'output_authorization_decision' : EngineeringItem;
        inout item 'item_DO_PAYMENTS_AUTH_REQUEST' : 'DO-PAYMENTS-AUTH-REQUEST';
        inout item 'item_DO_PAYMENTS_AUTH_DECISION' : 'DO-PAYMENTS-AUTH-DECISION';
        inout item 'item_DO_PAYMENTS_RISK_SIGNAL' : 'DO-PAYMENTS-RISK-SIGNAL';
    }

    @EngineeringElement {
        sourceId = "complete-review";
        conceptKind = "action_usage";
    }
    action 'complete-review' : EngineeringAction {
        in item 'input_review_ticket' : EngineeringItem;
        inout item 'item_DO_PAYMENTS_REVIEW_TICKET' : 'DO-PAYMENTS-REVIEW-TICKET';
        inout item 'item_DO_PAYMENTS_AUTH_DECISION' : 'DO-PAYMENTS-AUTH-DECISION';
    }

    @EngineeringElement {
        sourceId = "create-review-ticket";
        conceptKind = "action_usage";
    }
    action 'create-review-ticket' : EngineeringAction {
        in item 'input_risk_signal' : EngineeringItem;
        out item 'output_review_ticket' : EngineeringItem;
        inout item 'item_DO_PAYMENTS_RISK_SIGNAL' : 'DO-PAYMENTS-RISK-SIGNAL';
        inout item 'item_DO_PAYMENTS_REVIEW_TICKET' : 'DO-PAYMENTS-REVIEW-TICKET';
    }

    @EngineeringElement {
        sourceId = "detect-high-risk";
        conceptKind = "action_usage";
    }
    action 'detect-high-risk' : EngineeringAction {
        out item 'output_risk_signal' : EngineeringItem;
        inout item 'item_DO_PAYMENTS_RISK_SIGNAL' : 'DO-PAYMENTS-RISK-SIGNAL';
    }

    @EngineeringElement {
        sourceId = "normalize-request";
        conceptKind = "action_usage";
    }
    action 'normalize-request' : EngineeringAction {
        out item 'output_normalized_payment_request' : EngineeringItem;
        inout item 'item_DO_PAYMENTS_AUTH_REQUEST' : 'DO-PAYMENTS-AUTH-REQUEST';
    }

    @EngineeringElement {
        sourceId = "show-outcome";
        conceptKind = "action_usage";
    }
    action 'show-outcome' : EngineeringAction {
        in item 'input_authorization_decision' : EngineeringItem;
        inout item 'item_DO_PAYMENTS_AUTH_DECISION' : 'DO-PAYMENTS-AUTH-DECISION';
    }

    @EngineeringElement {
        sourceId = "submit-payment";
        conceptKind = "action_usage";
    }
    action 'submit-payment' : EngineeringAction {
        in item 'input_payment_method' : EngineeringItem;
        in item 'input_billing_address' : EngineeringItem;
        in item 'input_amount' : EngineeringItem;
        inout item 'item_DO_PAYMENTS_AUTH_REQUEST' : 'DO-PAYMENTS-AUTH-REQUEST';
    }

    @EngineeringRelationship {
        sourceId = "REL-001B4907F6EA13FD";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-001B4907F6EA13FD-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart;
        part 'DEP-PAYMENTS-CLUSTER-PROD' : EngineeringPart;
        allocation 'REL-001B4907F6EA13FD' allocate 'FU-PAYMENT-AUTHORIZATION' to 'DEP-PAYMENTS-CLUSTER-PROD';
    }

    @EngineeringRelationship {
        sourceId = "REL-0A56225E7ABEE474";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-0A56225E7ABEE474-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-RISK-SIGNAL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-0A56225E7ABEE474' from 'FU-PAYMENT-AUTHORIZATION'.pOut to 'DO-PAYMENTS-RISK-SIGNAL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-0B12973D243A202B";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-0B12973D243A202B' first 'create-review-ticket' then 'complete-review';

    @EngineeringRelationship {
        sourceId = "REL-0C1185394D6431F8";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-0C1185394D6431F8' from 'detect-high-risk' to 'FU-RISK-SCORING';

    @EngineeringRelationship {
        sourceId = "REL-11CF8F85B82A44ED";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-11CF8F85B82A44ED' first 'normalize-request' then 'show-outcome';

    @EngineeringRelationship {
        sourceId = "REL-1467856AF8DEDAD9";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-1467856AF8DEDAD9' from 'FU-RISK-SCORING' to 'REF-POSTGRES-DRIVER';

    @EngineeringRelationship {
        sourceId = "REL-1B6E37C829EF473B";
        conceptKind = "transfer";
        relationshipKind = "calls";
    }
    part def 'REL-1B6E37C829EF473B-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-PAYMENTS-RISK-SCORE-API' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-1B6E37C829EF473B' from 'FU-PAYMENT-AUTHORIZATION'.pOut to 'IF-PAYMENTS-RISK-SCORE-API'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-1D40C56E796F3CF6";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-1D40C56E796F3CF6-context' {
        part 'FU-SUPPORT-REVIEW' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-REVIEW-TICKET' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-1D40C56E796F3CF6' from 'FU-SUPPORT-REVIEW'.pOut to 'DO-PAYMENTS-REVIEW-TICKET'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-204FAD9EF6D93F2B";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-204FAD9EF6D93F2B' from 'ACT-PLATFORM-OPERATOR' to 'FU-CLUSTER-PROVISIONING';

    @EngineeringRelationship {
        sourceId = "REL-211223EFBC8DA2C8";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-211223EFBC8DA2C8-context' {
        part 'FU-RISK-SCORING' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-SUPPORT-REVIEW' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-211223EFBC8DA2C8' from 'FU-RISK-SCORING'.pOut to 'FU-SUPPORT-REVIEW'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-27417C6754C469A6";
        conceptKind = "transfer";
        relationshipKind = "publishes";
    }
    part def 'REL-27417C6754C469A6-context' {
        part 'FU-RISK-SCORING' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-RISK-SIGNAL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-27417C6754C469A6' from 'FU-RISK-SCORING'.pOut to 'DO-PAYMENTS-RISK-SIGNAL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-29F9332EC70A347C";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-29F9332EC70A347C-context' {
        part 'FU-CHECKOUT' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-AUTH-REQUEST' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-29F9332EC70A347C' from 'FU-CHECKOUT'.pOut to 'DO-PAYMENTS-AUTH-REQUEST'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-2BC004FA4CC61171";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-2BC004FA4CC61171-context' {
        part 'FU-CHECKOUT' : EngineeringPart { port pOut : EngineeringPort; }
        part 'ACT-CUSTOMER' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-2BC004FA4CC61171' from 'FU-CHECKOUT'.pOut to 'ACT-CUSTOMER'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-2DEB92611DBFCC72";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-2DEB92611DBFCC72' : EngineeringVerification {
        subject 'CV-PAYMENTS-CHECKOUT-SCHEMA-TEST' : EngineeringItem;
        objective 'CTRL-PAYMENTS-SSO-MFA' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-2E3D52D7D4A974DC";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-2E3D52D7D4A974DC-context' {
        part 'FU-RISK-SCORING' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-AUTH-REQUEST' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-2E3D52D7D4A974DC' from 'FU-RISK-SCORING'.pOut to 'DO-PAYMENTS-AUTH-REQUEST'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-2F128215CB3CCA21";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-2F128215CB3CCA21' from 'FU-GITOPS-OPERATIONS' to 'REF-HELM-PLATFORM';

    @EngineeringRelationship {
        sourceId = "REL-3078848886646128";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-3078848886646128' from 'authorize-payment' to 'FU-PAYMENT-AUTHORIZATION';

    @EngineeringRelationship {
        sourceId = "REL-31A31BD318A13DFC";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-31A31BD318A13DFC' first 'normalize-request' then 'authorize-payment';

    @EngineeringRelationship {
        sourceId = "REL-31E0CB220F1F58B5";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-31E0CB220F1F58B5-context' {
        part 'FU-RISK-SCORING' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-RISK-SIGNAL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-31E0CB220F1F58B5' from 'FU-RISK-SCORING'.pOut to 'DO-PAYMENTS-RISK-SIGNAL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-35EB6E1F4EBE86AC";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-35EB6E1F4EBE86AC-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-REVIEW-TICKET' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-35EB6E1F4EBE86AC' from 'FU-PAYMENT-AUTHORIZATION'.pOut to 'DO-PAYMENTS-REVIEW-TICKET'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-37E3A4BB64E4548B";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-37E3A4BB64E4548B' from 'ACT-PLATFORM-OPERATOR' to 'FU-GITOPS-OPERATIONS';

    @EngineeringRelationship {
        sourceId = "REL-394FB96B3227D425";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-394FB96B3227D425' from 'FU-GITOPS-OPERATIONS' to 'TB-PAYMENTS-PLATFORM-CONTROL';

    @EngineeringRelationship {
        sourceId = "REL-3C192DA3627331D4";
        conceptKind = "succession";
        relationshipKind = "flow_error";
    }
    succession 'REL-3C192DA3627331D4' first 'authorize-payment' then 'show-outcome';

    @EngineeringRelationship {
        sourceId = "REL-3FB95161B14A330F";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-3FB95161B14A330F-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-AUTH-DECISION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-3FB95161B14A330F' from 'FU-PAYMENT-AUTHORIZATION'.pOut to 'DO-PAYMENTS-AUTH-DECISION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-41902BD189A35B29";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-41902BD189A35B29-context' {
        part 'FU-CHECKOUT' : EngineeringPart;
        part 'DEP-PAYMENTS-CLUSTER-PROD' : EngineeringPart;
        allocation 'REL-41902BD189A35B29' allocate 'FU-CHECKOUT' to 'DEP-PAYMENTS-CLUSTER-PROD';
    }

    @EngineeringRelationship {
        sourceId = "REL-42190D91D9455EFD";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-42190D91D9455EFD-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-CHECKOUT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-42190D91D9455EFD' from 'FU-PAYMENT-AUTHORIZATION'.pOut to 'FU-CHECKOUT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-4265C0CD7F9F1271";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-4265C0CD7F9F1271' from 'FU-CHECKOUT' to 'FU-PAYMENT-AUTHORIZATION';

    @EngineeringRelationship {
        sourceId = "REL-4389964B492F242B";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-4389964B492F242B-context' : EngineeringState {
        state 'STATE-PAYMENTS-CHECKOUT-RECEIVED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-4389964B492F242B' first 'STATE-PAYMENTS-CHECKOUT-RECEIVED' accept 'EVT-PAYMENTS-AUTH-REQUESTED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-529ABAB0C41FE7CE";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-529ABAB0C41FE7CE' from 'show-outcome' to 'FU-CHECKOUT';

    @EngineeringRelationship {
        sourceId = "REL-553BE2C69F75EEEB";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-553BE2C69F75EEEB-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-PAYMENTS-BANK-AUTH' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-553BE2C69F75EEEB' from 'FU-PAYMENT-AUTHORIZATION'.pOut to 'IF-PAYMENTS-BANK-AUTH'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-57FFA74199281A1E";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-57FFA74199281A1E' first 'authorize-payment' then 'show-outcome';

    @EngineeringRelationship {
        sourceId = "REL-5C346F04EDA21858";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-5C346F04EDA21858-context' {
        part 'FU-CHECKOUT' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-AUTH-DECISION' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-5C346F04EDA21858' from 'FU-CHECKOUT'.pOut to 'DO-PAYMENTS-AUTH-DECISION'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-5D571A88BD03CD2D";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-5D571A88BD03CD2D' from 'FU-PAYMENT-AUTHORIZATION' to 'FU-SUPPORT-REVIEW';

    @EngineeringRelationship {
        sourceId = "REL-5E8441663CE65E78";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-5E8441663CE65E78' from 'AV-REPLAYED-AUTH-CALLBACK' to 'FU-PAYMENT-AUTHORIZATION';

    @EngineeringRelationship {
        sourceId = "REL-5F992C07A2903118";
        conceptKind = "event_trigger";
        relationshipKind = "triggered_by";
    }
    state 'REL-5F992C07A2903118-context' : EngineeringState {
        state 'STATE-PAYMENTS-AUTHORIZED' : EngineeringState;
        action transitionEffect : EngineeringAction;
        transition 'REL-5F992C07A2903118' first 'STATE-PAYMENTS-AUTHORIZED' accept 'EVT-PAYMENTS-AUTH-COMPLETED' : EngineeringItem then transitionEffect;
    }

    @EngineeringRelationship {
        sourceId = "REL-6BAF73F5116F4130";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-6BAF73F5116F4130' from 'complete-review' to 'FU-PAYMENT-AUTHORIZATION';

    @EngineeringRelationship {
        sourceId = "REL-6C5B7F0D48B4706C";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-6C5B7F0D48B4706C' : EngineeringVerification {
        subject 'CV-PAYMENTS-CALLBACK-REPLAY-TEST' : EngineeringItem;
        objective 'CTRL-PAYMENTS-CALLBACK-NONCE' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-6DA0F75DBF2701DF";
        conceptKind = "guard";
        relationshipKind = "guarded_by";
    }
    state 'REL-6DA0F75DBF2701DF-context' : EngineeringState {
        action 'STATE-PAYMENTS-AUTHORIZED' : EngineeringAction;
        action 'CTRL-PAYMENTS-SSO-MFA' : EngineeringAction;
        transition 'REL-6DA0F75DBF2701DF' first 'STATE-PAYMENTS-AUTHORIZED' if true then 'CTRL-PAYMENTS-SSO-MFA';
    }

    @EngineeringRelationship {
        sourceId = "REL-75A3C0F0450631CF";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-75A3C0F0450631CF-context' {
        part 'ACT-CUSTOMER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-CHECKOUT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-75A3C0F0450631CF' from 'ACT-CUSTOMER'.pOut to 'FU-CHECKOUT'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-773A17120F67C8C9";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-773A17120F67C8C9' from 'AV-COMPROMISED-DEPENDENCY' to 'CTRL-PAYMENTS-IMAGE-DIGEST';

    @EngineeringRelationship {
        sourceId = "REL-7CA50C2AC218B38E";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-7CA50C2AC218B38E' from 'AV-FRAUDULENT-TRANSACTION-PATTERN' to 'FU-SUPPORT-REVIEW';

    @EngineeringRelationship {
        sourceId = "REL-7E34822131C74217";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-7E34822131C74217' first 'submit-payment' then 'normalize-request';

    @EngineeringRelationship {
        sourceId = "REL-8E980FF67022810F";
        conceptKind = "verification";
        relationshipKind = "verifies";
    }
    verification 'REL-8E980FF67022810F' : EngineeringVerification {
        subject 'CV-PAYMENTS-MANUAL-REVIEW-TRACE' : EngineeringItem;
        objective 'CTRL-PAYMENTS-SSO-MFA' : EngineeringRequirement;
    }

    @EngineeringRelationship {
        sourceId = "REL-97C928547AA5EA18";
        conceptKind = "succession";
        relationshipKind = "flow_next";
    }
    succession 'REL-97C928547AA5EA18' first 'detect-high-risk' then 'create-review-ticket';

    @EngineeringRelationship {
        sourceId = "REL-986C5F6FBC6D193A";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-986C5F6FBC6D193A' from 'FU-PAYMENT-AUTHORIZATION' to 'REF-BANK-GATEWAY-ENDPOINT';

    @EngineeringRelationship {
        sourceId = "REL-9A5B70D0C50EAA45";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-9A5B70D0C50EAA45-context' {
        part 'FU-CHECKOUT' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-AUTH-REQUEST' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-9A5B70D0C50EAA45' from 'FU-CHECKOUT'.pOut to 'DO-PAYMENTS-AUTH-REQUEST'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-A3193AF968200977";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-A3193AF968200977' from 'submit-payment' to 'ACT-CUSTOMER';

    @EngineeringRelationship {
        sourceId = "REL-A3E7C3F706D9F2ED";
        conceptKind = "transition";
        relationshipKind = "transitions_to";
    }
    state 'REL-A3E7C3F706D9F2ED-context' : EngineeringState {
        action 'STATE-PAYMENTS-CHECKOUT-RECEIVED' : EngineeringAction;
        action 'STATE-PAYMENTS-AUTHORIZED' : EngineeringAction;
        transition 'REL-A3E7C3F706D9F2ED' first 'STATE-PAYMENTS-CHECKOUT-RECEIVED' then 'STATE-PAYMENTS-AUTHORIZED';
    }

    @EngineeringRelationship {
        sourceId = "REL-AE158DC172B5EBB7";
        conceptKind = "transfer";
        relationshipKind = "reads";
    }
    part def 'REL-AE158DC172B5EBB7-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-AUTH-REQUEST' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-AE158DC172B5EBB7' from 'FU-PAYMENT-AUTHORIZATION'.pOut to 'DO-PAYMENTS-AUTH-REQUEST'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-AF7839BB505DE4E4";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-AF7839BB505DE4E4' from 'AV-FRAUDULENT-TRANSACTION-PATTERN' to 'FU-RISK-SCORING';

    @EngineeringRelationship {
        sourceId = "REL-B462A681FC7999D0";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-B462A681FC7999D0' from 'FU-GITOPS-OPERATIONS' to 'FU-CLUSTER-PROVISIONING';

    @EngineeringRelationship {
        sourceId = "REL-BC68C3876E449907";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-BC68C3876E449907' from 'AV-MALICIOUS-API-REQUEST' to 'FU-CHECKOUT';

    @EngineeringRelationship {
        sourceId = "REL-BD79626488700C96";
        conceptKind = "transfer";
        relationshipKind = "subscribes";
    }
    part def 'REL-BD79626488700C96-context' {
        part 'FU-SUPPORT-REVIEW' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-RISK-SIGNAL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-BD79626488700C96' from 'FU-SUPPORT-REVIEW'.pOut to 'DO-PAYMENTS-RISK-SIGNAL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-C8D6C46397B16BE8";
        conceptKind = "dependency";
        relationshipKind = "depends_on";
    }
    dependency 'REL-C8D6C46397B16BE8' from 'FU-PAYMENT-AUTHORIZATION' to 'FU-RISK-SCORING';

    @EngineeringRelationship {
        sourceId = "REL-C94F6ECA1153D424";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-C94F6ECA1153D424' from 'AV-COMPROMISED-DEPENDENCY' to 'REF-POSTGRES-DRIVER';

    @EngineeringRelationship {
        sourceId = "REL-CC54CBDEE38AC7FA";
        conceptKind = "engineering_extension";
        relationshipKind = "mitigated_by";
    }
    dependency 'REL-CC54CBDEE38AC7FA' from 'AV-MALICIOUS-API-REQUEST' to 'CTRL-PAYMENTS-SSO-MFA';

    @EngineeringRelationship {
        sourceId = "REL-DEB5E03BDCEB12EE";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-DEB5E03BDCEB12EE' from 'create-review-ticket' to 'FU-SUPPORT-REVIEW';

    @EngineeringRelationship {
        sourceId = "REL-DFFD16B42D6EA756";
        conceptKind = "transfer";
        relationshipKind = "calls";
    }
    part def 'REL-DFFD16B42D6EA756-context' {
        part 'FU-CHECKOUT' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-PAYMENTS-CHECKOUT-API' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-DFFD16B42D6EA756' from 'FU-CHECKOUT'.pOut to 'IF-PAYMENTS-CHECKOUT-API'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-E040798484425F7C";
        conceptKind = "engineering_extension";
        relationshipKind = "bounded_by";
    }
    dependency 'REL-E040798484425F7C' from 'FU-PAYMENT-AUTHORIZATION' to 'TB-PAYMENTS-EXTERNAL-BANK';

    @EngineeringRelationship {
        sourceId = "REL-E17EC1AF9BFB25A2";
        conceptKind = "allocation";
        relationshipKind = "deployed_to";
    }
    part def 'REL-E17EC1AF9BFB25A2-context' {
        part 'IF-PAYMENTS-BANK-AUTH' : EngineeringPart;
        part 'DEP-PAYMENTS-BANK-EDGE' : EngineeringPart;
        allocation 'REL-E17EC1AF9BFB25A2' allocate 'IF-PAYMENTS-BANK-AUTH' to 'DEP-PAYMENTS-BANK-EDGE';
    }

    @EngineeringRelationship {
        sourceId = "REL-E4437656E5C341F0";
        conceptKind = "dependency";
        relationshipKind = "targets";
    }
    dependency 'REL-E4437656E5C341F0' from 'AV-FRAUDULENT-TRANSACTION-PATTERN' to 'FU-PAYMENT-AUTHORIZATION';

    @EngineeringRelationship {
        sourceId = "REL-E7CDE491595CF41E";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-E7CDE491595CF41E' from 'ACT-CUSTOMER' to 'FU-CHECKOUT';

    @EngineeringRelationship {
        sourceId = "REL-EB7245723D6F7EFD";
        conceptKind = "dependency";
        relationshipKind = "flow_ref";
    }
    dependency 'REL-EB7245723D6F7EFD' from 'normalize-request' to 'FU-CHECKOUT';

    @EngineeringRelationship {
        sourceId = "REL-F0AEB1EF28D2258F";
        conceptKind = "transfer";
        relationshipKind = "calls";
    }
    part def 'REL-F0AEB1EF28D2258F-context' {
        part 'FU-PAYMENT-AUTHORIZATION' : EngineeringPart { port pOut : EngineeringPort; }
        part 'IF-PAYMENTS-BANK-AUTH' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-F0AEB1EF28D2258F' from 'FU-PAYMENT-AUTHORIZATION'.pOut to 'IF-PAYMENTS-BANK-AUTH'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-F2DC1C4B74F751EF";
        conceptKind = "transfer";
        relationshipKind = "writes";
    }
    part def 'REL-F2DC1C4B74F751EF-context' {
        part 'FU-RISK-SCORING' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-RISK-SIGNAL' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-F2DC1C4B74F751EF' from 'FU-RISK-SCORING'.pOut to 'DO-PAYMENTS-RISK-SIGNAL'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-F3A654C61A1F2B9C";
        conceptKind = "guard";
        relationshipKind = "guarded_by";
    }
    state 'REL-F3A654C61A1F2B9C-context' : EngineeringState {
        action 'STATE-PAYMENTS-AUTHORIZED' : EngineeringAction;
        action 'TB-PAYMENTS-EXTERNAL-BANK' : EngineeringAction;
        transition 'REL-F3A654C61A1F2B9C' first 'STATE-PAYMENTS-AUTHORIZED' if true then 'TB-PAYMENTS-EXTERNAL-BANK';
    }

    @EngineeringRelationship {
        sourceId = "REL-FB0CB54DAC313459";
        conceptKind = "transfer";
        relationshipKind = "step_transfer";
    }
    part def 'REL-FB0CB54DAC313459-context' {
        part 'FU-SUPPORT-REVIEW' : EngineeringPart { port pOut : EngineeringPort; }
        part 'DO-PAYMENTS-REVIEW-TICKET' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-FB0CB54DAC313459' from 'FU-SUPPORT-REVIEW'.pOut to 'DO-PAYMENTS-REVIEW-TICKET'.pIn;
    }

    @EngineeringRelationship {
        sourceId = "REL-FDA416D9D551D62B";
        conceptKind = "dependency";
        relationshipKind = "interacts_with";
    }
    dependency 'REL-FDA416D9D551D62B' from 'ACT-SUPPORT' to 'FU-SUPPORT-REVIEW';

    @EngineeringRelationship {
        sourceId = "REL-FFA1941D592FEC39";
        conceptKind = "transfer";
        relationshipKind = "flow_transfer";
    }
    part def 'REL-FFA1941D592FEC39-context' {
        part 'ACT-CUSTOMER' : EngineeringPart { port pOut : EngineeringPort; }
        part 'FU-CHECKOUT' : EngineeringPart { port pIn : EngineeringPort; }
        flow 'REL-FFA1941D592FEC39' from 'ACT-CUSTOMER'.pOut to 'FU-CHECKOUT'.pIn;
    }
}
