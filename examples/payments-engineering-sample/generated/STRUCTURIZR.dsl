workspace "Sample Payments Layered Architecture" "This document describes the payments architecture using authored functional design and inferred runtime/code realization. It is intended to help product, platform, security, and implementation engineers reason about one shared model. Functional design is kept stable while realization details are inferred from infrastructure and source artifacts." {
  model {
    sys_sample_payments_layered_model = softwareSystem "Sample Payments Layered Architecture" "This document describes the payments architecture using authored functional design and inferred runtime/code realization. It is intended to help product, platform, security, and implementation engineers reason about one shared model. Functional design is kept stable while realization details are inferred from infrastructure and source artifacts." {
      fu_fu_checkout = container "Checkout Handling" "Checkout handling is the user-facing entrypoint where payment requests are initiated and normalized. It validates request shape, preserves transaction context, and returns clear customer feedback for each outcome. It delegates decision logic to payment authorization while protecting the user experience boundary." "Functional Unit"
      fu_fu_cluster_provisioning = container "Cluster Provisioning" "Cluster provisioning creates the runtime substrate used by all application workloads. It establishes cluster and namespace structure, baseline controls, and environment-level readiness. This unit is responsible for predictable, repeatable infrastructure foundations." "Functional Unit"
      fu_fu_gitops_operations = container "GitOps Operations" "GitOps operations continuously reconciles intended release state with actual runtime state. It governs rollout flow, drift correction, and operational delivery confidence across payment and fraud workloads. The unit provides safe and observable release behavior as an ongoing operational capability." "Functional Unit"
      fu_fu_payment_authorization = container "Payment Authorization" "Payment authorization orchestrates the final transaction decision path. It coordinates fraud scoring, external bank interactions, and escalation to support review when needed. The unit returns deterministic outcomes so downstream behavior remains consistent and testable." "Functional Unit"
      fu_fu_risk_scoring = container "Risk Scoring" "Risk scoring computes and classifies transaction risk before approval decisions are finalized. It provides a stable scoring contract to authorization and supports audit context for later analysis. The unit is focused on decision quality, consistency, and policy-driven classification behavior." "Functional Unit"
      fu_fu_support_review = container "Support Review" "Support review handles manual decisions for escalated or ambiguous payment cases. It gives support operators context to approve, reject, or request additional verification in a controlled flow. The unit ensures manual intervention remains auditable, policy-constrained, and operationally reliable." "Functional Unit"
    }
    person_act_customer = person "Customer" "Starts checkout and confirms payment."
    person_act_platform_operator = person "Platform Operator" "Operates GitOps and platform lifecycle."
    person_act_support = person "Support Agent" "Reviews high-risk and declined-payment cases."
    group_fg_fraud = softwareSystem "Group: Fraud Evaluation" "Risk scoring and fraud audit domain."
    group_fg_payments = softwareSystem "Group: Payments" "Core payment checkout and authorization domain."
    group_fg_platform = softwareSystem "Group: Platform" "Cluster provisioning and GitOps operations domain."
    ref_ref_bank_gateway_endpoint = softwareSystem "Ref: Bank Gateway Endpoint" "runtime"
    ref_ref_helm_platform = softwareSystem "Ref: Helm Runtime Platform" "runtime"
    ref_ref_postgres_driver = softwareSystem "Ref: PostgreSQL Client Driver" "code"
    if_if_payments_bank_auth = softwareSystem "Interface: Bank Authorization Interface" "https /bank/authorize"
    if_if_payments_checkout_api = softwareSystem "Interface: Checkout API Interface" "https /api/checkout"
    if_if_payments_risk_score_api = softwareSystem "Interface: Risk Score Interface" "https /risk/score"
    data_do_payments_auth_decision = softwareSystem "Data: Authorization Decision" "schemas/auth-decision.json"
    data_do_payments_auth_request = softwareSystem "Data: Authorization Request" "schemas/auth-request.json"
    data_do_payments_review_ticket = softwareSystem "Data: Manual Review Ticket" "schemas/review-ticket.json"
    data_do_payments_risk_signal = softwareSystem "Data: Risk Signal" "schemas/risk-signal.json"
    dep_dep_payments_bank_edge = softwareSystem "Deployment: Bank Edge Integration Zone" "prod bank-edge partner us-east-1"
    dep_dep_payments_cluster_prod = softwareSystem "Deployment: Payments Cluster Production" "prod payments-prod payments us-east-1"
    ctrl_ctrl_payments_callback_nonce = softwareSystem "Control: Callback Nonce and Idempotency Guard" "Enforce nonce freshness and idempotency on bank callback processing."
    ctrl_ctrl_payments_image_digest = softwareSystem "Control: Immutable Image Digests" "Enforce immutable digest-pinned container image references."
    ctrl_ctrl_payments_sso_mfa = softwareSystem "Control: SSO MFA Enforcement" "Enforce MFA and conditional access for privileged operations."
    av_av_compromised_dependency = softwareSystem "Attack: Compromised Dependency" "Supply chain compromise in external SDK or image."
    av_av_fraudulent_transaction_pattern = softwareSystem "Attack: Fraudulent Transaction Pattern" "Coordinated abuse pattern using synthetic identities, stolen instruments, and velocity anomalies to seek unauthorized approvals."
    av_av_malicious_api_request = softwareSystem "Attack: Malicious API Request" "Crafted payloads attempting to bypass authorization controls."
    av_av_replayed_auth_callback = softwareSystem "Attack: Replayed Authorization Callback" "Replay attack against external authorization callback flow."
    tb_tb_payments_external_bank = softwareSystem "Boundary: External Bank Boundary" "Separates internal payment authorization from external bank services."
    tb_tb_payments_platform_control = softwareSystem "Boundary: Platform Control Boundary" "Separates app workloads from platform control-plane authority."
    ts_ts_payments_bank_callback_replay = softwareSystem "Threat: Replayed bank callback triggers duplicate authorization transition" "Replayed callback message tries to overwrite or duplicate prior decision outcomes."
    ts_ts_payments_checkout_spoofing = softwareSystem "Threat: Checkout payload spoofing bypasses normalization checks" "Crafted payload shape attempts to force ambiguous authorization behavior and bypass policy checks."
    ts_ts_payments_risk_ticket_tamper = softwareSystem "Threat: Manual review ticket tampering influences authorization result" "A compromised support or workflow path modifies review ticket outcomes before authorization completes."
    fu_fu_gitops_operations -> tb_tb_payments_platform_control "bounded_by: GitOps control-plane boundaries."
    fu_fu_payment_authorization -> tb_tb_payments_external_bank "bounded_by: Authorization crosses external bank trust boundary."
    fu_fu_checkout -> if_if_payments_checkout_api "calls: Receives and validates checkout request payload."
    fu_fu_payment_authorization -> if_if_payments_bank_auth "calls: Calls external bank authorization interface."
    fu_fu_payment_authorization -> if_if_payments_risk_score_api "calls: Calls risk scoring interface for fraud decision support."
    group_fg_fraud -> fu_fu_risk_scoring "contains"
    group_fg_fraud -> fu_fu_support_review "contains"
    group_fg_payments -> fu_fu_checkout "contains"
    group_fg_payments -> fu_fu_payment_authorization "contains"
    group_fg_platform -> fu_fu_cluster_provisioning "contains"
    group_fg_platform -> fu_fu_gitops_operations "contains"
    fu_fu_checkout -> if_if_payments_checkout_api "contains"
    fu_fu_payment_authorization -> data_do_payments_auth_decision "contains"
    fu_fu_payment_authorization -> data_do_payments_auth_request "contains"
    fu_fu_payment_authorization -> if_if_payments_bank_auth "contains"
    fu_fu_risk_scoring -> data_do_payments_risk_signal "contains"
    fu_fu_risk_scoring -> if_if_payments_risk_score_api "contains"
    fu_fu_support_review -> data_do_payments_review_ticket "contains"
    fu_fu_checkout -> fu_fu_payment_authorization "depends_on: Delegates payment authorization."
    fu_fu_gitops_operations -> fu_fu_cluster_provisioning "depends_on: Relies on provisioned cluster and namespace baseline."
    fu_fu_gitops_operations -> ref_ref_helm_platform "depends_on: Uses Helm runtime platform for release orchestration."
    fu_fu_payment_authorization -> fu_fu_risk_scoring "depends_on: Requests fraud scoring before final authorization."
    fu_fu_payment_authorization -> fu_fu_support_review "depends_on: Escalates high-risk payments for manual review."
    fu_fu_payment_authorization -> ref_ref_bank_gateway_endpoint "depends_on: Calls external bank authorization endpoint."
    fu_fu_risk_scoring -> ref_ref_postgres_driver "depends_on: Uses external library for audit persistence."
    fu_fu_checkout -> dep_dep_payments_cluster_prod "deployed_to: Checkout workload deployment target."
    fu_fu_payment_authorization -> dep_dep_payments_cluster_prod "deployed_to: Authorization workload deployment target."
    if_if_payments_bank_auth -> dep_dep_payments_bank_edge "deployed_to: External authorization interface boundary target."
    person_act_customer -> fu_fu_checkout "interacts_with: Submits payment at checkout."
    person_act_platform_operator -> fu_fu_cluster_provisioning "interacts_with: Maintains cluster baseline and environment setup."
    person_act_platform_operator -> fu_fu_gitops_operations "interacts_with: Operates delivery and release workflows."
    person_act_support -> fu_fu_support_review "interacts_with: Performs manual payment reviews."
    av_av_compromised_dependency -> ctrl_ctrl_payments_image_digest "mitigated_by"
    av_av_malicious_api_request -> ctrl_ctrl_payments_sso_mfa "mitigated_by"
    fu_fu_risk_scoring -> data_do_payments_risk_signal "publishes: Publishes risk signal updates for downstream consumers."
    fu_fu_checkout -> data_do_payments_auth_decision "reads: Reads final authorization decision for user-facing response."
    fu_fu_payment_authorization -> data_do_payments_auth_request "reads: Reads normalized request for authorization pipeline."
    fu_fu_payment_authorization -> data_do_payments_review_ticket "reads: Reads manual review outcome before returning decision."
    fu_fu_payment_authorization -> data_do_payments_risk_signal "reads: Reads risk signal before final authorization decision."
    fu_fu_risk_scoring -> data_do_payments_auth_request "reads: Reads normalized request to compute risk signal."
    fu_fu_support_review -> data_do_payments_risk_signal "subscribes: Subscribes to high-risk signal updates."
    av_av_compromised_dependency -> ref_ref_postgres_driver "targets"
    av_av_fraudulent_transaction_pattern -> fu_fu_payment_authorization "targets"
    av_av_fraudulent_transaction_pattern -> fu_fu_risk_scoring "targets"
    av_av_fraudulent_transaction_pattern -> fu_fu_support_review "targets"
    av_av_malicious_api_request -> fu_fu_checkout "targets"
    av_av_replayed_auth_callback -> fu_fu_payment_authorization "targets"
    fu_fu_checkout -> data_do_payments_auth_request "writes: Persists normalized authorization request."
    fu_fu_payment_authorization -> data_do_payments_auth_decision "writes: Persists final authorization decision."
    fu_fu_risk_scoring -> data_do_payments_risk_signal "writes: Persists computed risk signal."
    fu_fu_support_review -> data_do_payments_review_ticket "writes: Persists manual review ticket records."
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_sample_payments_layered_model "context" {
      include *
      autolayout lr
    }

    container sys_sample_payments_layered_model "containers" {
      include *
      autolayout lr
    }
  }
}
