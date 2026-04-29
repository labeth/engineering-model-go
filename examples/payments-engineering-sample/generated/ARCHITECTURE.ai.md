# AI View

Schema: `ai-view/v1`

## Model

- ID: `sample-payments-layered-model`
- Title: Sample Payments Layered Architecture
- Counts: FG=3, FU=6, REQ=9, ADR=1, RT=9, CODE=30, VER=3, VIEWS=7

## Entry Points

### EP-FU-EVIDENCE

- Kind: `functional_units`
- Title: Functional units with runtime/code/verification evidence
- Entities: FU-CHECKOUT, FU-CLUSTER-PROVISIONING, FU-GITOPS-OPERATIONS, FU-PAYMENT-AUTHORIZATION, FU-RISK-SCORING, FU-SUPPORT-REVIEW

### EP-LOW-CONFIDENCE-INFERRED

- Kind: `inferred`
- Title: Low-confidence inferred entities
- Entities: none

### EP-REQ-COVERAGE

- Kind: `requirements`
- Title: Requirements with direct support paths
- Entities: REQ-PAY-001, REQ-PAY-002, REQ-PAY-003, REQ-PAY-004, REQ-PAY-005, REQ-PAY-006, REQ-PAY-007, REQ-PAY-008, REQ-PAY-009

### EP-REQ-GAPS

- Kind: `requirements`
- Title: Requirements with low-confidence support
- Entities: none

### EP-VERIFICATION-FAILURES

- Kind: `verification`
- Title: Verification checks with failing/partial status
- Entities: none

## Gaps

- Requirements without verification: REQ-PAY-005, REQ-PAY-007, REQ-PAY-008, REQ-PAY-009
- Requirements low confidence: none
- Functional units without tests: FU-CLUSTER-PROVISIONING, FU-GITOPS-OPERATIONS

## Support Paths

- `PATH-REQ_PAY_001`: REQ-PAY-001 -> FU-CHECKOUT -> RT-HELMRELEASE_PAYMENTS_CHECKOUT_API_C0C858 -> CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C -> IF-PAYMENTS-BANK-AUTH -> DO-PAYMENTS-AUTH-DECISION -> DEP-PAYMENTS-CLUSTER-PROD -> TB-PAYMENTS-EXTERNAL-BANK -> FLOW-CUSTOMER-CHECKOUT -> FLOW-CUSTOMER-CHECKOUT::authorize-payment -> VER-INF-E2E-AUTHORIZED_CHECKOUT_FLOW-8C6597 (confidence: high)
- `PATH-REQ_PAY_002`: REQ-PAY-002 -> FU-RISK-SCORING -> RT-HELMRELEASE_RISK_RISK_SCORER_A06799 -> CODE-LIBRARY_EXTERNAL_ZOD_B6E604 -> IF-PAYMENTS-RISK-SCORE-API -> DO-PAYMENTS-AUTH-REQUEST -> FLOW-PAYMENTS-MANUAL-REVIEW -> FLOW-PAYMENTS-MANUAL-REVIEW::detect-high-risk -> VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1 (confidence: high)
- `PATH-REQ_PAY_003`: REQ-PAY-003 -> FU-CHECKOUT -> RT-HELMRELEASE_PAYMENTS_CHECKOUT_API_C0C858 -> CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C -> IF-PAYMENTS-CHECKOUT-API -> DO-PAYMENTS-AUTH-DECISION -> DEP-PAYMENTS-CLUSTER-PROD -> FLOW-CUSTOMER-CHECKOUT -> FLOW-CUSTOMER-CHECKOUT::normalize-request -> VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26 (confidence: high)
- `PATH-REQ_PAY_004`: REQ-PAY-004 -> FU-PAYMENT-AUTHORIZATION -> RT-HELMRELEASE_PAYMENTS_PAYMENT_ENGINE_C79944 -> CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD -> IF-PAYMENTS-BANK-AUTH -> DO-PAYMENTS-AUTH-DECISION -> DEP-PAYMENTS-CLUSTER-PROD -> TB-PAYMENTS-EXTERNAL-BANK -> FLOW-CUSTOMER-CHECKOUT -> FLOW-CUSTOMER-CHECKOUT::authorize-payment -> VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1 (confidence: high)
- `PATH-REQ_PAY_005`: REQ-PAY-005 -> FU-RISK-SCORING -> RT-HELMRELEASE_RISK_RISK_SCORER_A06799 -> CODE-LIBRARY_EXTERNAL_ZOD_B6E604 -> IF-PAYMENTS-RISK-SCORE-API -> DO-PAYMENTS-AUTH-REQUEST -> FLOW-PAYMENTS-MANUAL-REVIEW -> FLOW-PAYMENTS-MANUAL-REVIEW::detect-high-risk (confidence: medium)
- `PATH-REQ_PAY_006`: REQ-PAY-006 -> FU-CHECKOUT -> RT-HELMRELEASE_PAYMENTS_CHECKOUT_API_C0C858 -> CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C -> IF-PAYMENTS-BANK-AUTH -> DO-PAYMENTS-AUTH-DECISION -> DEP-PAYMENTS-CLUSTER-PROD -> TB-PAYMENTS-EXTERNAL-BANK -> FLOW-CUSTOMER-CHECKOUT -> FLOW-CUSTOMER-CHECKOUT::authorize-payment -> VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26 (confidence: high)
- `PATH-REQ_PAY_007`: REQ-PAY-007 -> FU-CHECKOUT -> RT-HELMRELEASE_PAYMENTS_CHECKOUT_API_C0C858 -> CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C -> IF-PAYMENTS-BANK-AUTH -> DO-PAYMENTS-AUTH-DECISION -> DEP-PAYMENTS-CLUSTER-PROD -> TB-PAYMENTS-EXTERNAL-BANK -> FLOW-CUSTOMER-CHECKOUT -> FLOW-CUSTOMER-CHECKOUT::authorize-payment (confidence: medium)
- `PATH-REQ_PAY_008`: REQ-PAY-008 -> FU-CHECKOUT -> RT-HELMRELEASE_PAYMENTS_CHECKOUT_API_C0C858 -> CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C -> IF-PAYMENTS-CHECKOUT-API -> DO-PAYMENTS-AUTH-DECISION -> DEP-PAYMENTS-CLUSTER-PROD -> FLOW-CUSTOMER-CHECKOUT -> FLOW-CUSTOMER-CHECKOUT::normalize-request (confidence: medium)
- `PATH-REQ_PAY_009`: REQ-PAY-009 -> FU-PAYMENT-AUTHORIZATION -> RT-HELMRELEASE_PAYMENTS_PAYMENT_ENGINE_C79944 -> CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD -> IF-PAYMENTS-BANK-AUTH -> DO-PAYMENTS-AUTH-DECISION -> DEP-PAYMENTS-CLUSTER-PROD -> TB-PAYMENTS-EXTERNAL-BANK -> FLOW-CUSTOMER-CHECKOUT -> FLOW-CUSTOMER-CHECKOUT::authorize-payment (confidence: medium)

## Entities

### FG-FRAUD

- Kind: `functional_group`
- Origin: `authored`
- Status: `stable`
- Title: Fraud Evaluation
- Summary: Risk scoring and fraud audit domain.
- Source Refs: SRC-FUNCTIONAL_GROUP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_20_A2BB9C

### FG-PAYMENTS

- Kind: `functional_group`
- Origin: `authored`
- Status: `stable`
- Title: Payments
- Summary: Core payment checkout and authorization domain.
- Source Refs: SRC-FUNCTIONAL_GROUP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_12_6F3D73

### FG-PLATFORM

- Kind: `functional_group`
- Origin: `authored`
- Status: `stable`
- Title: Platform
- Summary: Cluster provisioning and GitOps operations domain.
- Source Refs: SRC-FUNCTIONAL_GROUP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_27_81E020

### FU-CHECKOUT

- Kind: `functional_unit`
- Origin: `authored`
- Status: `stable`
- Title: Checkout Handling
- Summary: Checkout handling is the user-facing entrypoint where payment requests are initiated and normalized.
It validates request shape, preserves transaction context, and returns clear customer feedback for each outcome.
It delegates decision logic to payment authorization while protecting the user experience boundary.
- Requirements: REQ-PAY-001, REQ-PAY-003, REQ-PAY-006, REQ-PAY-007, REQ-PAY-008
- Runtime: RT-HELMRELEASE_PAYMENTS_CHECKOUT_API_C0C858, RT-NAMESPACE_PAYMENTS_407A87
- Code: CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C, CODE-LIBRARY_FIRST_PARTY_GITHUB_COM_LABETH_ENGINEERING_MODEL_GO_VALIDATE_A9639C, CODE-LIBRARY_STDLIB_FMT_A9639C, CODE-SOURCE_FILE_CHECKOUT_API_GO_A9639C, CODE-SYMBOL_CODE_HANDLEBANKUNAVAILABLE_71F914, CODE-SYMBOL_CODE_SHOWDECLINEREASON_6CF90C, CODE-SYMBOL_CODE_SHOWRETRYOPTION_67E7BB, CODE-SYMBOL_CODE_SHOWTEMPORARYUNAVAILABLE_6CE7C3, CODE-SYMBOL_CODE_STARTSESSION_7DF4A9, CODE-SYMBOL_CODE_SUBMITAUTHORIZATIONREQUEST_6CF6CD
- Verification: VER-INF-E2E-AUTHORIZED_CHECKOUT_FLOW-8C6597, VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26
- Source Refs: SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_141_2007BE, SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_152_1AAF5F, SRC-DEPLOYMENT_TARGET_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_185_9B4916, SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_203_3C2812, SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_3F691B, SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_D4BB3F, SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_36_3C830B, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_0_B8EDF7, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_16_E1FD80, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_27_C26AA7, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_32_BA675C, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_37_A18725, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_42_379B22, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_47_A583AD, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_6_A59B59, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_8_F46CE5, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_9_8CEEAC, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_CHECKOUT_API_YAML_0_B014BE, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_22_1D93E6, SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_121_5217CB

### FU-CLUSTER-PROVISIONING

- Kind: `functional_unit`
- Origin: `authored`
- Status: `stable`
- Title: Cluster Provisioning
- Summary: Cluster provisioning creates the runtime substrate used by all application workloads.
It establishes cluster and namespace structure, baseline controls, and environment-level readiness.
This unit is responsible for predictable, repeatable infrastructure foundations.
- Runtime: RT-CLUSTER_PAYMENTS_407A87
- Source Refs: SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_64_1902C9, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_22_A48834

### FU-GITOPS-OPERATIONS

- Kind: `functional_unit`
- Origin: `authored`
- Status: `stable`
- Title: GitOps Operations
- Summary: GitOps operations continuously reconciles intended release state with actual runtime state.
It governs rollout flow, drift correction, and operational delivery confidence across payment and fraud workloads.
The unit provides safe and observable release behavior as an ongoing operational capability.
- Runtime: RT-GITREPOSITORY_FLUX_SYSTEM_PAYMENTS_SAMPLE_SOURCE_BFF14F, RT-KUSTOMIZATION_FLUX_SYSTEM_PAYMENTS_APPS_E94D9E, RT-NAMESPACE_FLUX_SYSTEM_407A87
- Source Refs: SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_72_F94CDD, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_KUSTOMIZATIONS_PAYMENTS_APPS_YAML_0_DEDB01, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_SOURCES_PAYMENTS_SOURCE_YAML_0_11D961, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_61_3A610D, SRC-TRUST_BOUNDARY_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_508_73A453

### FU-PAYMENT-AUTHORIZATION

- Kind: `functional_unit`
- Origin: `authored`
- Status: `stable`
- Title: Payment Authorization
- Summary: Payment authorization orchestrates the final transaction decision path.
It coordinates fraud scoring, external bank interactions, and escalation to support review when needed.
The unit returns deterministic outcomes so downstream behavior remains consistent and testable.
- Requirements: REQ-PAY-001, REQ-PAY-004, REQ-PAY-006, REQ-PAY-007, REQ-PAY-009
- Runtime: RT-HELMRELEASE_PAYMENTS_PAYMENT_ENGINE_C79944
- Code: CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD, CODE-LIBRARY_FIRST_PARTY_CRATE__DOMAIN__EVENTS__PAYMENTEVENT_FBE0BD, CODE-SOURCE_FILE_DOMAIN_EVENTS_RS_A76A04, CODE-SOURCE_FILE_PAYMENT_ENGINE_RS_FBE0BD, CODE-SYMBOL_CODE_AUTHORIZEPAYMENT_55B9FA, CODE-SYMBOL_CODE_HANDLEBANKLINKUNAVAILABLE_DCBF4C, CODE-SYMBOL_CODE_NOTIFYSUPPORT_EEC3E5, CODE-SYMBOL_CODE_PERSISTAUDITRECORD_5ECB51, CODE-SYMBOL_CODE_PLACEINREVIEW_EEC624, CODE-SYMBOL_CODE_REQUESTBANKAUTHORIZATION_ECC1A3
- Verification: VER-INF-E2E-AUTHORIZED_CHECKOUT_FLOW-8C6597, VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26, VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1
- Source Refs: SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_141_2007BE, SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_152_1AAF5F, SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_163_4E0586, SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_173_46C220, SRC-DEPLOYMENT_TARGET_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_185_9B4916, SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_203_3C2812, SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_282_2537EE, SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_1E8CA1, SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_24DF8A, SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_43_F0410C, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_DOMAIN_EVENTS_RS_0_D181B6, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_0_267B7A, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_11_76A1DC, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_26_ABE137, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_34_375D7D, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_3_134177, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_44_71CA3D, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_4_D231B2, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_52_9D4042, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_60_94C860, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_PAYMENT_ENGINE_YAML_0_F4D319, SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_127_EA6A1C, SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_133_522E55, SRC-TRUST_BOUNDARY_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_503_2EB414

### FU-RISK-SCORING

- Kind: `functional_unit`
- Origin: `authored`
- Status: `stable`
- Title: Risk Scoring
- Summary: Risk scoring computes and classifies transaction risk before approval decisions are finalized.
It provides a stable scoring contract to authorization and supports audit context for later analysis.
The unit is focused on decision quality, consistency, and policy-driven classification behavior.
- Requirements: REQ-PAY-002, REQ-PAY-005
- Runtime: RT-HELMRELEASE_RISK_RISK_SCORER_A06799, RT-NAMESPACE_RISK_407A87
- Code: CODE-LIBRARY_EXTERNAL_ZOD_B6E604, CODE-LIBRARY_FIRST_PARTY___SUPPORT_AUDIT_ENVELOPE_B6E604, CODE-SOURCE_FILE_RISK_SCORER_TS_B6E604, CODE-SOURCE_FILE_SUPPORT_AUDIT_ENVELOPE_TS_5249D4, CODE-SYMBOL_CODE_BUILDREVIEWREASON_A93FAA, CODE-SYMBOL_CODE_CALCULATERISKSCORE_EC259B, CODE-SYMBOL_CODE_CLASSIFYRISK_A84C74, CODE-SYMBOL_CODE_CREATEAUDITRECORD_AA4EB6, CODE-SYMBOL_CODE_ENRICHAUDITMETADATA_A83D6A, CODE-SYMBOL_CODE_ISHIGHRISK_A04C68
- Verification: VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1
- Source Refs: SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_141_2007BE, SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_163_4E0586, SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_282_2537EE, SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_CE06E6, SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_50_524811, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_0_144B84, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_21_9E4A63, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_29_F41669, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_37_B73FD2, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_3_813CFE, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_45_BB65FD, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_4_AC3241, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_50_722704, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_8_1EE47D, SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_SUPPORT_AUDIT_ENVELOPE_TS_0_E91E33, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_RISK_SCORER_YAML_0_FD05D3, SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_26_8DE014, SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_133_522E55

### FU-SUPPORT-REVIEW

- Kind: `functional_unit`
- Origin: `authored`
- Status: `stable`
- Title: Support Review
- Summary: Support review handles manual decisions for escalated or ambiguous payment cases.
It gives support operators context to approve, reject, or request additional verification in a controlled flow.
The unit ensures manual intervention remains auditable, policy-constrained, and operationally reliable.
- Requirements: REQ-PAY-004, REQ-PAY-009
- Verification: VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1
- Source Refs: SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_163_4E0586, SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_173_46C220, SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_282_2537EE, SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_217AD5, SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_57_FC1077

### REQ-PAY-001

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-001
- Summary: While normal mode is active, when payment authorization is requested, the payments system shall create a payment session.
- Verification: VER-INF-E2E-AUTHORIZED_CHECKOUT_FLOW-8C6597
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_8_A95B03, SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TESTS_E2E_AUTHORIZED_CHECKOUT_FLOW_YAML_0_BB6CE5

### REQ-PAY-002

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-002
- Summary: Where fraud check is enabled, when payment authorization is requested, the fraud detection system shall calculate a risk score before approval.
- Verification: VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_15_3D42D6, SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TEST_RESULTS_INTEGRATION_FRAUD_GATE_JSON_0_17A855

### REQ-PAY-003

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-003
- Summary: If payment is declined, then the payments system shall present a decline reason to customer.
- Verification: VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_21_9A34C1, SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TEST_RESULTS_CONTRACT_CHECKOUT_DECLINE_JSON_0_D46FCA

### REQ-PAY-004

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-004
- Summary: While checkout is active, when payment authorization is requested and risk score is high and risk threshold is exceeded, the payments system shall place the payment in payment review is pending.
- Verification: VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_27_6F0F82, SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TEST_RESULTS_INTEGRATION_FRAUD_GATE_JSON_0_17A855

### REQ-PAY-005

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-005
- Summary: Where fraud check is enabled, when payment authorization is requested, the payments system shall persist an audit record with payment id plus risk score.
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_34_AD3304

### REQ-PAY-006

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-006
- Summary: If bank link is unavailable, then the payments system shall present a temporary unavailable message to customer until bank link is available.
- Verification: VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_40_610935, SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TEST_RESULTS_CONTRACT_CHECKOUT_DECLINE_JSON_0_D46FCA

### REQ-PAY-007

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-007
- Summary: While checkout is active, when payment authorization is requested and 3DS verification is required, the payments system shall request authorization from bank gateway endpoint.
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_47_1E7B43

### REQ-PAY-008

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-008
- Summary: If payment is declined, then the payments system shall present a retry option to customer.
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_54_A827C3

### REQ-PAY-009

- Kind: `requirement`
- Origin: `authored`
- Status: `stable`
- Title: REQ-PAY-009
- Summary: While checkout is active, when payment authorization is requested and risk threshold is exceeded, the payments system shall notify platform operator when support agent is notified.
- Source Refs: SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_60_B15CB2

### ADR-PAY-001

- Kind: `decision`
- Origin: `authored`
- Status: `accepted`
- Title: Keep payment authorization separate from checkout handling
- Summary: Model checkout handling and payment authorization as separate functional units with explicit
calls and dependency mappings between them.
- Source Refs: SRC-DECISION_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DECISIONS_YML_2_2AF2CD

### CTRL-PAYMENTS-CALLBACK-NONCE

- Kind: `control`
- Origin: `authored`
- Status: `stable`
- Title: Callback Nonce and Idempotency Guard
- Summary: Enforce nonce freshness and idempotency on bank callback processing.
- Source Refs: SRC-CONTROL_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_454_2FD23F

### CTRL-PAYMENTS-IMAGE-DIGEST

- Kind: `control`
- Origin: `authored`
- Status: `stable`
- Title: Immutable Image Digests
- Summary: Enforce immutable digest-pinned container image references.
- Source Refs: SRC-CONTROL_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_450_486244

### CTRL-PAYMENTS-SSO-MFA

- Kind: `control`
- Origin: `authored`
- Status: `stable`
- Title: SSO MFA Enforcement
- Summary: Enforce MFA and conditional access for privileged operations.
- Source Refs: SRC-CONTROL_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_446_1B646E

### DEP-PAYMENTS-BANK-EDGE

- Kind: `deployment_target`
- Origin: `authored`
- Status: `stable`
- Title: Bank Edge Integration Zone
- Summary: prod bank-edge partner
- Source Refs: SRC-DEPLOYMENT_TARGET_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_193_F21F53

### DEP-PAYMENTS-CLUSTER-PROD

- Kind: `deployment_target`
- Origin: `authored`
- Status: `stable`
- Title: Payments Cluster Production
- Summary: prod payments-prod payments
- Source Refs: SRC-DEPLOYMENT_TARGET_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_185_9B4916

### DO-PAYMENTS-AUTH-DECISION

- Kind: `data_object`
- Origin: `authored`
- Status: `stable`
- Title: Authorization Decision
- Summary: internal
- Source Refs: SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_152_1AAF5F

### DO-PAYMENTS-AUTH-REQUEST

- Kind: `data_object`
- Origin: `authored`
- Status: `stable`
- Title: Authorization Request
- Summary: confidential
- Source Refs: SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_141_2007BE

### DO-PAYMENTS-REVIEW-TICKET

- Kind: `data_object`
- Origin: `authored`
- Status: `stable`
- Title: Manual Review Ticket
- Summary: confidential
- Source Refs: SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_173_46C220

### DO-PAYMENTS-RISK-SIGNAL

- Kind: `data_object`
- Origin: `authored`
- Status: `stable`
- Title: Risk Signal
- Summary: internal
- Source Refs: SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_163_4E0586

### EVT-PAYMENTS-AUTH-COMPLETED

- Kind: `event`
- Origin: `authored`
- Status: `stable`
- Title: Authorization Completed
- Summary: Authorization decision returned to checkout.
- Source Refs: SRC-EVENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_526_5AB470

### EVT-PAYMENTS-AUTH-REQUESTED

- Kind: `event`
- Origin: `authored`
- Status: `stable`
- Title: Authorization Requested
- Summary: Request sent to authorization orchestration.
- Source Refs: SRC-EVENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_523_326B00

### FLOW-CUSTOMER-CHECKOUT

- Kind: `flow`
- Origin: `authored`
- Status: `stable`
- Title: Customer Checkout Authorization Flow
- Summary: entry: submit-payment; exits: show-outcome; steps: 4
- Source Refs: SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_203_3C2812

### FLOW-CUSTOMER-CHECKOUT::authorize-payment

- Kind: `flow_step`
- Origin: `authored`
- Status: `stable`
- Title: Request fraud score and external authorization
- Summary: Request fraud score and external authorization (in: normalized_payment_request; out: authorization_decision)
- Source Refs: SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_1E8CA1

### FLOW-CUSTOMER-CHECKOUT::normalize-request

- Kind: `flow_step`
- Origin: `authored`
- Status: `stable`
- Title: Normalize and validate checkout payload
- Summary: Normalize and validate checkout payload (in: ; out: normalized_payment_request)
- Source Refs: SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_3F691B

### FLOW-CUSTOMER-CHECKOUT::show-outcome

- Kind: `flow_step`
- Origin: `authored`
- Status: `stable`
- Title: Return approval or decline outcome to customer
- Summary: Return approval or decline outcome to customer (in: authorization_decision; out: )
- Source Refs: SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_D4BB3F

### FLOW-CUSTOMER-CHECKOUT::submit-payment

- Kind: `flow_step`
- Origin: `authored`
- Status: `stable`
- Title: Customer submits checkout payment form
- Summary: Customer submits checkout payment form (in: payment_method, billing_address, amount; out: )
- Source Refs: SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_CA5399

### FLOW-PAYMENTS-MANUAL-REVIEW

- Kind: `flow`
- Origin: `authored`
- Status: `stable`
- Title: High-Risk Manual Review Flow
- Summary: entry: detect-high-risk; exits: complete-review; steps: 3
- Source Refs: SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_282_2537EE

### FLOW-PAYMENTS-MANUAL-REVIEW::complete-review

- Kind: `flow_step`
- Origin: `authored`
- Status: `stable`
- Title: Complete authorization using manual review outcome
- Summary: Complete authorization using manual review outcome (in: review_ticket; out: )
- Source Refs: SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_24DF8A

### FLOW-PAYMENTS-MANUAL-REVIEW::create-review-ticket

- Kind: `flow_step`
- Origin: `authored`
- Status: `stable`
- Title: Create manual review ticket for support operations
- Summary: Create manual review ticket for support operations (in: risk_signal; out: review_ticket)
- Source Refs: SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_217AD5

### FLOW-PAYMENTS-MANUAL-REVIEW::detect-high-risk

- Kind: `flow_step`
- Origin: `authored`
- Status: `stable`
- Title: Detect high-risk signal and emit escalation trigger
- Summary: Detect high-risk signal and emit escalation trigger (in: ; out: risk_signal)
- Source Refs: SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_CE06E6

### IF-PAYMENTS-BANK-AUTH

- Kind: `interface`
- Origin: `authored`
- Status: `stable`
- Title: Bank Authorization Interface
- Summary: https /bank/authorize
- Source Refs: SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_127_EA6A1C

### IF-PAYMENTS-CHECKOUT-API

- Kind: `interface`
- Origin: `authored`
- Status: `stable`
- Title: Checkout API Interface
- Summary: https /api/checkout
- Source Refs: SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_121_5217CB

### IF-PAYMENTS-RISK-SCORE-API

- Kind: `interface`
- Origin: `authored`
- Status: `stable`
- Title: Risk Score Interface
- Summary: https /risk/score
- Source Refs: SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_133_522E55

### STATE-PAYMENTS-AUTHORIZED

- Kind: `state`
- Origin: `authored`
- Status: `stable`
- Title: Payment Authorized
- Summary: Authorization decision approved.
- Source Refs: SRC-STATE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_518_A3211E

### STATE-PAYMENTS-CHECKOUT-RECEIVED

- Kind: `state`
- Origin: `authored`
- Status: `stable`
- Title: Checkout Received
- Summary: Payment request accepted at checkout boundary.
- Source Refs: SRC-STATE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_515_5E8B3E

### TB-PAYMENTS-EXTERNAL-BANK

- Kind: `trust_boundary`
- Origin: `authored`
- Status: `stable`
- Title: External Bank Boundary
- Summary: Separates internal payment authorization from external bank services.
- Source Refs: SRC-TRUST_BOUNDARY_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_503_2EB414

### TB-PAYMENTS-PLATFORM-CONTROL

- Kind: `trust_boundary`
- Origin: `authored`
- Status: `stable`
- Title: Platform Control Boundary
- Summary: Separates app workloads from platform control-plane authority.
- Source Refs: SRC-TRUST_BOUNDARY_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_508_73A453

### RT-CLUSTER_PAYMENTS_407A87

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: payments
- Summary: Inferred runtime cluster owned by FU-CLUSTER-PROVISIONING
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_22_A48834

### RT-GITREPOSITORY_FLUX_SYSTEM_PAYMENTS_SAMPLE_SOURCE_BFF14F

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: flux-system/payments-sample-source
- Summary: Inferred runtime gitrepository owned by FU-GITOPS-OPERATIONS
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_SOURCES_PAYMENTS_SOURCE_YAML_0_11D961

### RT-HELMRELEASE_PAYMENTS_CHECKOUT_API_C0C858

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: payments/checkout-api
- Summary: Inferred runtime helmrelease owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_CHECKOUT_API_YAML_0_B014BE

### RT-HELMRELEASE_PAYMENTS_PAYMENT_ENGINE_C79944

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: payments/payment-engine
- Summary: Inferred runtime helmrelease owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_PAYMENT_ENGINE_YAML_0_F4D319

### RT-HELMRELEASE_RISK_RISK_SCORER_A06799

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: risk/risk-scorer
- Summary: Inferred runtime helmrelease owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_RISK_SCORER_YAML_0_FD05D3

### RT-KUSTOMIZATION_FLUX_SYSTEM_PAYMENTS_APPS_E94D9E

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: flux-system/payments-apps
- Summary: Inferred runtime kustomization owned by FU-GITOPS-OPERATIONS
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_KUSTOMIZATIONS_PAYMENTS_APPS_YAML_0_DEDB01

### RT-NAMESPACE_FLUX_SYSTEM_407A87

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: flux_system
- Summary: Inferred runtime namespace owned by FU-GITOPS-OPERATIONS
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_61_3A610D

### RT-NAMESPACE_PAYMENTS_407A87

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: payments
- Summary: Inferred runtime namespace owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_22_1D93E6

### RT-NAMESPACE_RISK_407A87

- Kind: `runtime_element`
- Origin: `inferred`
- Status: `inferred`
- Title: risk
- Summary: Inferred runtime namespace owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_26_8DE014

### CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: gopkg.in/yaml.v3
- Summary: Inferred code library_external owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_9_8CEEAC

### CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: serde_json::json
- Summary: Inferred code library_external owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_4_D231B2

### CODE-LIBRARY_EXTERNAL_ZOD_B6E604

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: zod
- Summary: Inferred code library_external owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_3_813CFE

### CODE-LIBRARY_FIRST_PARTY_CRATE__DOMAIN__EVENTS__PAYMENTEVENT_FBE0BD

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: crate::domain::events::PaymentEvent
- Summary: Inferred code library_first_party owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_3_134177

### CODE-LIBRARY_FIRST_PARTY_GITHUB_COM_LABETH_ENGINEERING_MODEL_GO_VALIDATE_A9639C

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: github.com/labeth/engineering-model-go/validate
- Summary: Inferred code library_first_party owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_8_F46CE5

### CODE-LIBRARY_FIRST_PARTY___SUPPORT_AUDIT_ENVELOPE_B6E604

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: ./support/audit_envelope
- Summary: Inferred code library_first_party owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_4_AC3241

### CODE-LIBRARY_STDLIB_FMT_A9639C

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: fmt
- Summary: Inferred code library_stdlib owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_6_A59B59

### CODE-SOURCE_FILE_CHECKOUT_API_GO_A9639C

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: checkout_api.go
- Summary: Inferred code source_file owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_0_B8EDF7

### CODE-SOURCE_FILE_DOMAIN_EVENTS_RS_A76A04

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: domain/events.rs
- Summary: Inferred code source_file owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_DOMAIN_EVENTS_RS_0_D181B6

### CODE-SOURCE_FILE_PAYMENT_ENGINE_RS_FBE0BD

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: payment_engine.rs
- Summary: Inferred code source_file owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_0_267B7A

### CODE-SOURCE_FILE_RISK_SCORER_TS_B6E604

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: risk_scorer.ts
- Summary: Inferred code source_file owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_0_144B84

### CODE-SOURCE_FILE_SUPPORT_AUDIT_ENVELOPE_TS_5249D4

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: support/audit_envelope.ts
- Summary: Inferred code source_file owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_SUPPORT_AUDIT_ENVELOPE_TS_0_E91E33

### CODE-SYMBOL_CODE_AUTHORIZEPAYMENT_55B9FA

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-AUTHORIZEPAYMENT
- Summary: Inferred code symbol owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_11_76A1DC

### CODE-SYMBOL_CODE_BUILDREVIEWREASON_A93FAA

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-BUILDREVIEWREASON
- Summary: Inferred code symbol owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_50_722704

### CODE-SYMBOL_CODE_CALCULATERISKSCORE_EC259B

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-CALCULATERISKSCORE
- Summary: Inferred code symbol owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_8_1EE47D

### CODE-SYMBOL_CODE_CLASSIFYRISK_A84C74

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-CLASSIFYRISK
- Summary: Inferred code symbol owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_21_9E4A63

### CODE-SYMBOL_CODE_CREATEAUDITRECORD_AA4EB6

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-CREATEAUDITRECORD
- Summary: Inferred code symbol owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_37_B73FD2

### CODE-SYMBOL_CODE_ENRICHAUDITMETADATA_A83D6A

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-ENRICHAUDITMETADATA
- Summary: Inferred code symbol owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_45_BB65FD

### CODE-SYMBOL_CODE_HANDLEBANKLINKUNAVAILABLE_DCBF4C

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-HANDLEBANKLINKUNAVAILABLE
- Summary: Inferred code symbol owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_34_375D7D

### CODE-SYMBOL_CODE_HANDLEBANKUNAVAILABLE_71F914

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-HANDLEBANKUNAVAILABLE
- Summary: Inferred code symbol owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_32_BA675C

### CODE-SYMBOL_CODE_ISHIGHRISK_A04C68

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-ISHIGHRISK
- Summary: Inferred code symbol owned by FU-RISK-SCORING
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_29_F41669

### CODE-SYMBOL_CODE_NOTIFYSUPPORT_EEC3E5

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-NOTIFYSUPPORT
- Summary: Inferred code symbol owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_52_9D4042

### CODE-SYMBOL_CODE_PERSISTAUDITRECORD_5ECB51

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-PERSISTAUDITRECORD
- Summary: Inferred code symbol owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_60_94C860

### CODE-SYMBOL_CODE_PLACEINREVIEW_EEC624

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-PLACEINREVIEW
- Summary: Inferred code symbol owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_44_71CA3D

### CODE-SYMBOL_CODE_REQUESTBANKAUTHORIZATION_ECC1A3

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-REQUESTBANKAUTHORIZATION
- Summary: Inferred code symbol owned by FU-PAYMENT-AUTHORIZATION
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_26_ABE137

### CODE-SYMBOL_CODE_SHOWDECLINEREASON_6CF90C

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-SHOWDECLINEREASON
- Summary: Inferred code symbol owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_37_A18725

### CODE-SYMBOL_CODE_SHOWRETRYOPTION_67E7BB

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-SHOWRETRYOPTION
- Summary: Inferred code symbol owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_42_379B22

### CODE-SYMBOL_CODE_SHOWTEMPORARYUNAVAILABLE_6CE7C3

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-SHOWTEMPORARYUNAVAILABLE
- Summary: Inferred code symbol owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_47_A583AD

### CODE-SYMBOL_CODE_STARTSESSION_7DF4A9

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-STARTSESSION
- Summary: Inferred code symbol owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_16_E1FD80

### CODE-SYMBOL_CODE_SUBMITAUTHORIZATIONREQUEST_6CF6CD

- Kind: `code_element`
- Origin: `inferred`
- Status: `inferred`
- Title: CODE-SUBMITAUTHORIZATIONREQUEST
- Summary: Inferred code symbol owned by FU-CHECKOUT
- Source Refs: SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_27_C26AA7

### VER-INF-E2E-AUTHORIZED_CHECKOUT_FLOW-8C6597

- Kind: `verification`
- Origin: `verification`
- Status: `not-run`
- Title: Authorized Checkout Flow
- Summary: executes authorized checkout flow from session start through payment authorization success
- Requirements: REQ-PAY-001
- Source Refs: SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TESTS_E2E_AUTHORIZED_CHECKOUT_FLOW_YAML_0_BB6CE5

### VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26

- Kind: `verification`
- Origin: `verification`
- Status: `pass`
- Title: Contract Checkout Decline
- Summary: Inferred from test result artifact.
- Requirements: REQ-PAY-003, REQ-PAY-006
- Source Refs: SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TEST_RESULTS_CONTRACT_CHECKOUT_DECLINE_JSON_0_D46FCA

### VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1

- Kind: `verification`
- Origin: `verification`
- Status: `pass`
- Title: Integration Fraud Gate
- Summary: Inferred from test result artifact.
- Requirements: REQ-PAY-002, REQ-PAY-004
- Source Refs: SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TEST_RESULTS_INTEGRATION_FRAUD_GATE_JSON_0_17A855

## Implementation Paths

### IMPL-PAY-001

- Requirement: `REQ-PAY-001`
- Priority: `medium`
- Goal: While normal mode is active, when payment authorization is requested, the payments system shall create a payment session.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-001`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-CHECKOUT`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C`)
  - 4. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-PAYMENT-AUTHORIZATION`)
  - 5. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD`)
  - 6. Update or add tests so verification stays passing for this requirement (`VER-INF-E2E-AUTHORIZED_CHECKOUT_FLOW-8C6597`)

### IMPL-PAY-002

- Requirement: `REQ-PAY-002`
- Priority: `medium`
- Goal: Where fraud check is enabled, when payment authorization is requested, the fraud detection system shall calculate a risk score before approval.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-002`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-RISK-SCORING`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_ZOD_B6E604`)
  - 4. Update or add tests so verification stays passing for this requirement (`VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1`)

### IMPL-PAY-003

- Requirement: `REQ-PAY-003`
- Priority: `medium`
- Goal: If payment is declined, then the payments system shall present a decline reason to customer.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-003`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-CHECKOUT`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C`)
  - 4. Update or add tests so verification stays passing for this requirement (`VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26`)

### IMPL-PAY-004

- Requirement: `REQ-PAY-004`
- Priority: `medium`
- Goal: While checkout is active, when payment authorization is requested and risk score is high and risk threshold is exceeded, the payments system shall place the payment in payment review is pending.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-004`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-PAYMENT-AUTHORIZATION`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD`)
  - 4. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-SUPPORT-REVIEW`)
  - 5. Update or add tests so verification stays passing for this requirement (`VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1`)

### IMPL-PAY-005

- Requirement: `REQ-PAY-005`
- Priority: `high`
- Goal: Where fraud check is enabled, when payment authorization is requested, the payments system shall persist an audit record with payment id plus risk score.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-005`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-RISK-SCORING`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_ZOD_B6E604`)
  - 4. Add new verification coverage (tests/checks) for this requirement

### IMPL-PAY-006

- Requirement: `REQ-PAY-006`
- Priority: `medium`
- Goal: If bank link is unavailable, then the payments system shall present a temporary unavailable message to customer until bank link is available.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-006`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-CHECKOUT`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C`)
  - 4. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-PAYMENT-AUTHORIZATION`)
  - 5. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD`)
  - 6. Update or add tests so verification stays passing for this requirement (`VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26`)

### IMPL-PAY-007

- Requirement: `REQ-PAY-007`
- Priority: `high`
- Goal: While checkout is active, when payment authorization is requested and 3DS verification is required, the payments system shall request authorization from bank gateway endpoint.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-007`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-CHECKOUT`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C`)
  - 4. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-PAYMENT-AUTHORIZATION`)
  - 5. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD`)
  - 6. Add new verification coverage (tests/checks) for this requirement

### IMPL-PAY-008

- Requirement: `REQ-PAY-008`
- Priority: `high`
- Goal: If payment is declined, then the payments system shall present a retry option to customer.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-008`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-CHECKOUT`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C`)
  - 4. Add new verification coverage (tests/checks) for this requirement

### IMPL-PAY-009

- Requirement: `REQ-PAY-009`
- Priority: `high`
- Goal: While checkout is active, when payment authorization is requested and risk threshold is exceeded, the payments system shall notify platform operator when support agent is notified.
  - 1. Confirm requirement intent and acceptance criteria before code changes (`REQ-PAY-009`)
  - 2. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-PAYMENT-AUTHORIZATION`)
  - 3. Update code evidence linked to this unit (`CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD`)
  - 4. Implement or update behavior in functional unit scope and dependent interfaces/flows (`FU-SUPPORT-REVIEW`)
  - 5. Add new verification coverage (tests/checks) for this requirement

## Source Blocks

- `SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_1E8CA1` examples/payments-engineering-sample/architecture.yml [flow_step] entities=FLOW-CUSTOMER-CHECKOUT::authorize-payment
- `SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_217AD5` examples/payments-engineering-sample/architecture.yml [flow_step] entities=FLOW-PAYMENTS-MANUAL-REVIEW::create-review-ticket
- `SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_24DF8A` examples/payments-engineering-sample/architecture.yml [flow_step] entities=FLOW-PAYMENTS-MANUAL-REVIEW::complete-review
- `SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_3F691B` examples/payments-engineering-sample/architecture.yml [flow_step] entities=FLOW-CUSTOMER-CHECKOUT::normalize-request
- `SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_CA5399` examples/payments-engineering-sample/architecture.yml [flow_step] entities=FLOW-CUSTOMER-CHECKOUT::submit-payment
- `SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_CE06E6` examples/payments-engineering-sample/architecture.yml [flow_step] entities=FLOW-PAYMENTS-MANUAL-REVIEW::detect-high-risk
- `SRC-FLOW_STEP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_0_D4BB3F` examples/payments-engineering-sample/architecture.yml [flow_step] entities=FLOW-CUSTOMER-CHECKOUT::show-outcome
- `SRC-FUNCTIONAL_GROUP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_12_6F3D73` examples/payments-engineering-sample/architecture.yml:12 [functional_group] entities=FG-PAYMENTS
- `SRC-FUNCTIONAL_GROUP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_20_A2BB9C` examples/payments-engineering-sample/architecture.yml:20 [functional_group] entities=FG-FRAUD
- `SRC-FUNCTIONAL_GROUP_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_27_81E020` examples/payments-engineering-sample/architecture.yml:27 [functional_group] entities=FG-PLATFORM
- `SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_36_3C830B` examples/payments-engineering-sample/architecture.yml:36 [functional_unit] entities=FU-CHECKOUT
- `SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_43_F0410C` examples/payments-engineering-sample/architecture.yml:43 [functional_unit] entities=FU-PAYMENT-AUTHORIZATION
- `SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_50_524811` examples/payments-engineering-sample/architecture.yml:50 [functional_unit] entities=FU-RISK-SCORING
- `SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_57_FC1077` examples/payments-engineering-sample/architecture.yml:57 [functional_unit] entities=FU-SUPPORT-REVIEW
- `SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_64_1902C9` examples/payments-engineering-sample/architecture.yml:64 [functional_unit] entities=FU-CLUSTER-PROVISIONING
- `SRC-FUNCTIONAL_UNIT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_72_F94CDD` examples/payments-engineering-sample/architecture.yml:72 [functional_unit] entities=FU-GITOPS-OPERATIONS
- `SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_121_5217CB` examples/payments-engineering-sample/architecture.yml:121 [interface] entities=IF-PAYMENTS-CHECKOUT-API
- `SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_127_EA6A1C` examples/payments-engineering-sample/architecture.yml:127 [interface] entities=IF-PAYMENTS-BANK-AUTH
- `SRC-INTERFACE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_133_522E55` examples/payments-engineering-sample/architecture.yml:133 [interface] entities=IF-PAYMENTS-RISK-SCORE-API
- `SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_141_2007BE` examples/payments-engineering-sample/architecture.yml:141 [data_object] entities=DO-PAYMENTS-AUTH-REQUEST
- `SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_152_1AAF5F` examples/payments-engineering-sample/architecture.yml:152 [data_object] entities=DO-PAYMENTS-AUTH-DECISION
- `SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_163_4E0586` examples/payments-engineering-sample/architecture.yml:163 [data_object] entities=DO-PAYMENTS-RISK-SIGNAL
- `SRC-DATA_OBJECT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_173_46C220` examples/payments-engineering-sample/architecture.yml:173 [data_object] entities=DO-PAYMENTS-REVIEW-TICKET
- `SRC-DEPLOYMENT_TARGET_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_185_9B4916` examples/payments-engineering-sample/architecture.yml:185 [deployment_target] entities=DEP-PAYMENTS-CLUSTER-PROD
- `SRC-DEPLOYMENT_TARGET_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_193_F21F53` examples/payments-engineering-sample/architecture.yml:193 [deployment_target] entities=DEP-PAYMENTS-BANK-EDGE
- `SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_203_3C2812` examples/payments-engineering-sample/architecture.yml:203 [flow] entities=FLOW-CUSTOMER-CHECKOUT
- `SRC-FLOW_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_282_2537EE` examples/payments-engineering-sample/architecture.yml:282 [flow] entities=FLOW-PAYMENTS-MANUAL-REVIEW
- `SRC-CONTROL_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_446_1B646E` examples/payments-engineering-sample/architecture.yml:446 [control] entities=CTRL-PAYMENTS-SSO-MFA
- `SRC-CONTROL_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_450_486244` examples/payments-engineering-sample/architecture.yml:450 [control] entities=CTRL-PAYMENTS-IMAGE-DIGEST
- `SRC-CONTROL_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_454_2FD23F` examples/payments-engineering-sample/architecture.yml:454 [control] entities=CTRL-PAYMENTS-CALLBACK-NONCE
- `SRC-TRUST_BOUNDARY_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_503_2EB414` examples/payments-engineering-sample/architecture.yml:503 [trust_boundary] entities=TB-PAYMENTS-EXTERNAL-BANK
- `SRC-TRUST_BOUNDARY_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_508_73A453` examples/payments-engineering-sample/architecture.yml:508 [trust_boundary] entities=TB-PAYMENTS-PLATFORM-CONTROL
- `SRC-STATE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_515_5E8B3E` examples/payments-engineering-sample/architecture.yml:515 [state] entities=STATE-PAYMENTS-CHECKOUT-RECEIVED
- `SRC-STATE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_518_A3211E` examples/payments-engineering-sample/architecture.yml:518 [state] entities=STATE-PAYMENTS-AUTHORIZED
- `SRC-EVENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_523_326B00` examples/payments-engineering-sample/architecture.yml:523 [event] entities=EVT-PAYMENTS-AUTH-REQUESTED
- `SRC-EVENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_ARCHITECTURE_YML_526_5AB470` examples/payments-engineering-sample/architecture.yml:526 [event] entities=EVT-PAYMENTS-AUTH-COMPLETED
- `SRC-DECISION_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DECISIONS_YML_2_2AF2CD` examples/payments-engineering-sample/decisions.yml:2 [decision] entities=ADR-PAY-001
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_6_6D37C3` examples/payments-engineering-sample/design.yml:6 [design_yaml] entities=FG-PAYMENTS
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_39_89B173` examples/payments-engineering-sample/design.yml:39 [design_yaml] entities=FG-FRAUD
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_70_C60C95` examples/payments-engineering-sample/design.yml:70 [design_yaml] entities=FG-PLATFORM
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_104_70B98C` examples/payments-engineering-sample/design.yml:104 [design_yaml] entities=FU-CHECKOUT
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_137_C7139D` examples/payments-engineering-sample/design.yml:137 [design_yaml] entities=FU-PAYMENT-AUTHORIZATION
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_166_F6C0E4` examples/payments-engineering-sample/design.yml:166 [design_yaml] entities=FU-RISK-SCORING
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_195_E0FF70` examples/payments-engineering-sample/design.yml:195 [design_yaml] entities=FU-SUPPORT-REVIEW
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_224_5DA374` examples/payments-engineering-sample/design.yml:224 [design_yaml] entities=FU-CLUSTER-PROVISIONING
- `SRC-DESIGN_YAML_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_DESIGN_YML_253_193DF1` examples/payments-engineering-sample/design.yml:253 [design_yaml] entities=FU-GITOPS-OPERATIONS
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_KUSTOMIZATIONS_PAYMENTS_APPS_YAML_0_DEDB01` examples/payments-engineering-sample/infra/flux/kustomizations/payments-apps.yaml [inferred_runtime] entities=RT-KUSTOMIZATION_FLUX_SYSTEM_PAYMENTS_APPS_E94D9E
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_CHECKOUT_API_YAML_0_B014BE` examples/payments-engineering-sample/infra/flux/releases/checkout-api.yaml [inferred_runtime] entities=RT-HELMRELEASE_PAYMENTS_CHECKOUT_API_C0C858
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_PAYMENT_ENGINE_YAML_0_F4D319` examples/payments-engineering-sample/infra/flux/releases/payment-engine.yaml [inferred_runtime] entities=RT-HELMRELEASE_PAYMENTS_PAYMENT_ENGINE_C79944
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_RELEASES_RISK_SCORER_YAML_0_FD05D3` examples/payments-engineering-sample/infra/flux/releases/risk-scorer.yaml [inferred_runtime] entities=RT-HELMRELEASE_RISK_RISK_SCORER_A06799
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_FLUX_SOURCES_PAYMENTS_SOURCE_YAML_0_11D961` examples/payments-engineering-sample/infra/flux/sources/payments-source.yaml [inferred_runtime] entities=RT-GITREPOSITORY_FLUX_SYSTEM_PAYMENTS_SAMPLE_SOURCE_BFF14F
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_22_1D93E6` examples/payments-engineering-sample/infra/terraform/main.tf:22 [inferred_runtime] entities=RT-NAMESPACE_PAYMENTS_407A87
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_22_A48834` examples/payments-engineering-sample/infra/terraform/main.tf:22 [inferred_runtime] entities=RT-CLUSTER_PAYMENTS_407A87
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_26_8DE014` examples/payments-engineering-sample/infra/terraform/main.tf:26 [inferred_runtime] entities=RT-NAMESPACE_RISK_407A87
- `SRC-INFERRED_RUNTIME_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_INFRA_TERRAFORM_MAIN_TF_61_3A610D` examples/payments-engineering-sample/infra/terraform/main.tf:61 [inferred_runtime] entities=RT-NAMESPACE_FLUX_SYSTEM_407A87
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_8_A95B03` examples/payments-engineering-sample/requirements.yml:8 [requirement] entities=REQ-PAY-001
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_15_3D42D6` examples/payments-engineering-sample/requirements.yml:15 [requirement] entities=REQ-PAY-002
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_21_9A34C1` examples/payments-engineering-sample/requirements.yml:21 [requirement] entities=REQ-PAY-003
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_27_6F0F82` examples/payments-engineering-sample/requirements.yml:27 [requirement] entities=REQ-PAY-004
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_34_AD3304` examples/payments-engineering-sample/requirements.yml:34 [requirement] entities=REQ-PAY-005
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_40_610935` examples/payments-engineering-sample/requirements.yml:40 [requirement] entities=REQ-PAY-006
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_47_1E7B43` examples/payments-engineering-sample/requirements.yml:47 [requirement] entities=REQ-PAY-007
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_54_A827C3` examples/payments-engineering-sample/requirements.yml:54 [requirement] entities=REQ-PAY-008
- `SRC-REQUIREMENT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_REQUIREMENTS_YML_60_B15CB2` examples/payments-engineering-sample/requirements.yml:60 [requirement] entities=REQ-PAY-009
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_0_B8EDF7` examples/payments-engineering-sample/src/checkout_api.go [inferred_code] entities=CODE-SOURCE_FILE_CHECKOUT_API_GO_A9639C
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_6_A59B59` examples/payments-engineering-sample/src/checkout_api.go:6 [inferred_code] entities=CODE-LIBRARY_STDLIB_FMT_A9639C
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_8_F46CE5` examples/payments-engineering-sample/src/checkout_api.go:8 [inferred_code] entities=CODE-LIBRARY_FIRST_PARTY_GITHUB_COM_LABETH_ENGINEERING_MODEL_GO_VALIDATE_A9639C
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_9_8CEEAC` examples/payments-engineering-sample/src/checkout_api.go:9 [inferred_code] entities=CODE-LIBRARY_EXTERNAL_GOPKG_IN_YAML_V3_A9639C
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_16_E1FD80` examples/payments-engineering-sample/src/checkout_api.go:16 [inferred_code] entities=CODE-SYMBOL_CODE_STARTSESSION_7DF4A9
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_27_C26AA7` examples/payments-engineering-sample/src/checkout_api.go:27 [inferred_code] entities=CODE-SYMBOL_CODE_SUBMITAUTHORIZATIONREQUEST_6CF6CD
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_32_BA675C` examples/payments-engineering-sample/src/checkout_api.go:32 [inferred_code] entities=CODE-SYMBOL_CODE_HANDLEBANKUNAVAILABLE_71F914
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_37_A18725` examples/payments-engineering-sample/src/checkout_api.go:37 [inferred_code] entities=CODE-SYMBOL_CODE_SHOWDECLINEREASON_6CF90C
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_42_379B22` examples/payments-engineering-sample/src/checkout_api.go:42 [inferred_code] entities=CODE-SYMBOL_CODE_SHOWRETRYOPTION_67E7BB
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_CHECKOUT_API_GO_47_A583AD` examples/payments-engineering-sample/src/checkout_api.go:47 [inferred_code] entities=CODE-SYMBOL_CODE_SHOWTEMPORARYUNAVAILABLE_6CE7C3
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_DOMAIN_EVENTS_RS_0_D181B6` examples/payments-engineering-sample/src/domain/events.rs [inferred_code] entities=CODE-SOURCE_FILE_DOMAIN_EVENTS_RS_A76A04
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_0_267B7A` examples/payments-engineering-sample/src/payment_engine.rs [inferred_code] entities=CODE-SOURCE_FILE_PAYMENT_ENGINE_RS_FBE0BD
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_3_134177` examples/payments-engineering-sample/src/payment_engine.rs:3 [inferred_code] entities=CODE-LIBRARY_FIRST_PARTY_CRATE__DOMAIN__EVENTS__PAYMENTEVENT_FBE0BD
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_4_D231B2` examples/payments-engineering-sample/src/payment_engine.rs:4 [inferred_code] entities=CODE-LIBRARY_EXTERNAL_SERDE_JSON__JSON_FBE0BD
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_11_76A1DC` examples/payments-engineering-sample/src/payment_engine.rs:11 [inferred_code] entities=CODE-SYMBOL_CODE_AUTHORIZEPAYMENT_55B9FA
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_26_ABE137` examples/payments-engineering-sample/src/payment_engine.rs:26 [inferred_code] entities=CODE-SYMBOL_CODE_REQUESTBANKAUTHORIZATION_ECC1A3
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_34_375D7D` examples/payments-engineering-sample/src/payment_engine.rs:34 [inferred_code] entities=CODE-SYMBOL_CODE_HANDLEBANKLINKUNAVAILABLE_DCBF4C
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_44_71CA3D` examples/payments-engineering-sample/src/payment_engine.rs:44 [inferred_code] entities=CODE-SYMBOL_CODE_PLACEINREVIEW_EEC624
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_52_9D4042` examples/payments-engineering-sample/src/payment_engine.rs:52 [inferred_code] entities=CODE-SYMBOL_CODE_NOTIFYSUPPORT_EEC3E5
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_PAYMENT_ENGINE_RS_60_94C860` examples/payments-engineering-sample/src/payment_engine.rs:60 [inferred_code] entities=CODE-SYMBOL_CODE_PERSISTAUDITRECORD_5ECB51
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_0_144B84` examples/payments-engineering-sample/src/risk_scorer.ts [inferred_code] entities=CODE-SOURCE_FILE_RISK_SCORER_TS_B6E604
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_3_813CFE` examples/payments-engineering-sample/src/risk_scorer.ts:3 [inferred_code] entities=CODE-LIBRARY_EXTERNAL_ZOD_B6E604
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_4_AC3241` examples/payments-engineering-sample/src/risk_scorer.ts:4 [inferred_code] entities=CODE-LIBRARY_FIRST_PARTY___SUPPORT_AUDIT_ENVELOPE_B6E604
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_8_1EE47D` examples/payments-engineering-sample/src/risk_scorer.ts:8 [inferred_code] entities=CODE-SYMBOL_CODE_CALCULATERISKSCORE_EC259B
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_21_9E4A63` examples/payments-engineering-sample/src/risk_scorer.ts:21 [inferred_code] entities=CODE-SYMBOL_CODE_CLASSIFYRISK_A84C74
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_29_F41669` examples/payments-engineering-sample/src/risk_scorer.ts:29 [inferred_code] entities=CODE-SYMBOL_CODE_ISHIGHRISK_A04C68
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_37_B73FD2` examples/payments-engineering-sample/src/risk_scorer.ts:37 [inferred_code] entities=CODE-SYMBOL_CODE_CREATEAUDITRECORD_AA4EB6
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_45_BB65FD` examples/payments-engineering-sample/src/risk_scorer.ts:45 [inferred_code] entities=CODE-SYMBOL_CODE_ENRICHAUDITMETADATA_A83D6A
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_RISK_SCORER_TS_50_722704` examples/payments-engineering-sample/src/risk_scorer.ts:50 [inferred_code] entities=CODE-SYMBOL_CODE_BUILDREVIEWREASON_A93FAA
- `SRC-INFERRED_CODE_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_SRC_SUPPORT_AUDIT_ENVELOPE_TS_0_E91E33` examples/payments-engineering-sample/src/support/audit_envelope.ts [inferred_code] entities=CODE-SOURCE_FILE_SUPPORT_AUDIT_ENVELOPE_TS_5249D4
- `SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TEST_RESULTS_CONTRACT_CHECKOUT_DECLINE_JSON_0_D46FCA` examples/payments-engineering-sample/test-results/contract-checkout-decline.json [verification_artifact] entities=VER-INF-TEST-CONTRACT_CHECKOUT_DECLINE-5D1F26
- `SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TEST_RESULTS_INTEGRATION_FRAUD_GATE_JSON_0_17A855` examples/payments-engineering-sample/test-results/integration-fraud-gate.json [verification_artifact] entities=VER-INF-TEST-INTEGRATION_FRAUD_GATE-FEDAF1
- `SRC-VERIFICATION_ARTIFACT_EXAMPLES_PAYMENTS_ENGINEERING_SAMPLE_TESTS_E2E_AUTHORIZED_CHECKOUT_FLOW_YAML_0_BB6CE5` examples/payments-engineering-sample/tests/e2e/authorized_checkout_flow.yaml [verification_artifact] entities=VER-INF-E2E-AUTHORIZED_CHECKOUT_FLOW-8C6597
