workspace "Sample Payments Layered Architecture" "This document describes the payments architecture using authored functional design and inferred runtime/code realization. It is intended to help product, platform, security, and implementation engineers reason about one shared model. Functional design is kept stable while realization details are inferred from infrastructure and source artifacts." {
  model {
    sys_sample_payments_layered_model = softwareSystem "Sample Payments Layered Architecture" "This document describes the payments architecture using authored functional design and inferred runtime/code realization. It is intended to help product, platform, security, and implementation engineers reason about one shared model. Functional design is kept stable while realization details are inferred from infrastructure and source artifacts." {
      group "Fraud Evaluation" {
        fu_fu_risk_scoring = container "Risk Scoring" "Risk scoring computes and classifies transaction risk before approval decisions are finalized. It provides a stable scoring contract to authorization and supports audit context for later analysis. The unit is focused on decision quality, consistency, and policy-driven classification behavior." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_support_review = container "Support Review" "Support review handles manual decisions for escalated or ambiguous payment cases. It gives support operators context to approve, reject, or request additional verification in a controlled flow. The unit ensures manual intervention remains auditable, policy-constrained, and operationally reliable." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Payments" {
        fu_fu_checkout = container "Checkout Handling" "Checkout handling is the user-facing entrypoint where payment requests are initiated and normalized. It validates request shape, preserves transaction context, and returns clear customer feedback for each outcome. It delegates decision logic to payment authorization while protecting the user experience boundary." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_payment_authorization = container "Payment Authorization" "Payment authorization orchestrates the final transaction decision path. It coordinates fraud scoring, external bank interactions, and escalation to support review when needed. The unit returns deterministic outcomes so downstream behavior remains consistent and testable." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Platform" {
        fu_fu_cluster_provisioning = container "Cluster Provisioning" "Cluster provisioning creates the runtime substrate used by all application workloads. It establishes cluster and namespace structure, baseline controls, and environment-level readiness. This unit is responsible for predictable, repeatable infrastructure foundations." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_gitops_operations = container "GitOps Operations" "GitOps operations continuously reconciles intended release state with actual runtime state. It governs rollout flow, drift correction, and operational delivery confidence across payment and fraud workloads. The unit provides safe and observable release behavior as an ongoing operational capability." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_customer = person "Customer" "Starts checkout and confirms payment." {
      tags "Actor"
    }
    person_act_platform_operator = person "Platform Operator" "Operates GitOps and platform lifecycle." {
      tags "Actor"
    }
    person_act_support = person "Support Agent" "Reviews high-risk and declined-payment cases." {
      tags "Actor"
    }
    group_fg_fraud = softwareSystem "Fraud Evaluation" "Risk scoring and fraud audit domain." {
      tags "FunctionalGroup"
    }
    group_fg_payments = softwareSystem "Payments" "Core payment checkout and authorization domain." {
      tags "FunctionalGroup"
    }
    group_fg_platform = softwareSystem "Platform" "Cluster provisioning and GitOps operations domain." {
      tags "FunctionalGroup"
    }
    ref_ref_bank_gateway_endpoint = softwareSystem "Bank Gateway Endpoint" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_helm_platform = softwareSystem "Helm Runtime Platform" "runtime" {
      tags "ReferencedElement,platform_service"
    }
    ref_ref_postgres_driver = softwareSystem "PostgreSQL Client Driver" "code" {
      tags "ReferencedElement,third_party_library"
    }
    if_if_payments_bank_auth = softwareSystem "Bank Authorization Interface" "https /bank/authorize" {
      tags "Interface"
    }
    if_if_payments_checkout_api = softwareSystem "Checkout API Interface" "https /api/checkout" {
      tags "Interface"
    }
    if_if_payments_risk_score_api = softwareSystem "Risk Score Interface" "https /risk/score" {
      tags "Interface"
    }
    data_do_payments_auth_decision = softwareSystem "Authorization Decision" "schemas/auth-decision.json" {
      tags "DataObject,internal"
    }
    data_do_payments_auth_request = softwareSystem "Authorization Request" "schemas/auth-request.json" {
      tags "DataObject,confidential"
    }
    data_do_payments_review_ticket = softwareSystem "Manual Review Ticket" "schemas/review-ticket.json" {
      tags "DataObject,confidential"
    }
    data_do_payments_risk_signal = softwareSystem "Risk Signal" "schemas/risk-signal.json" {
      tags "DataObject,internal"
    }
    ctrl_ctrl_payments_callback_nonce = softwareSystem "Callback Nonce and Idempotency Guard" "Enforce nonce freshness and idempotency on bank callback processing." {
      tags "Control,protocol-integrity"
    }
    ctrl_ctrl_payments_image_digest = softwareSystem "Immutable Image Digests" "Enforce immutable digest-pinned container image references." {
      tags "Control,supply-chain"
    }
    ctrl_ctrl_payments_sso_mfa = softwareSystem "SSO MFA Enforcement" "Enforce MFA and conditional access for privileged operations." {
      tags "Control,identity-access"
    }
    av_av_compromised_dependency = softwareSystem "Compromised Dependency" "Supply chain compromise in external SDK or image." {
      tags "AttackVector"
    }
    av_av_fraudulent_transaction_pattern = softwareSystem "Fraudulent Transaction Pattern" "Coordinated abuse pattern using synthetic identities, stolen instruments, and velocity anomalies to seek unauthorized approvals." {
      tags "AttackVector"
    }
    av_av_malicious_api_request = softwareSystem "Malicious API Request" "Crafted payloads attempting to bypass authorization controls." {
      tags "AttackVector"
    }
    av_av_replayed_auth_callback = softwareSystem "Replayed Authorization Callback" "Replay attack against external authorization callback flow." {
      tags "AttackVector"
    }
    tb_tb_payments_external_bank = softwareSystem "External Bank Boundary" "Separates internal payment authorization from external bank services." {
      tags "TrustBoundary,network"
    }
    tb_tb_payments_platform_control = softwareSystem "Platform Control Boundary" "Separates app workloads from platform control-plane authority." {
      tags "TrustBoundary,control-plane"
    }
    ts_ts_payments_bank_callback_replay = softwareSystem "Replayed bank callback triggers duplicate authorization transition" "Replayed callback message tries to overwrite or duplicate prior decision outcomes." {
      tags "ThreatScenario,replay,mitigating"
    }
    ts_ts_payments_checkout_spoofing = softwareSystem "Checkout payload spoofing bypasses normalization checks" "Crafted payload shape attempts to force ambiguous authorization behavior and bypass policy checks." {
      tags "ThreatScenario,tampering,mitigating"
    }
    ts_ts_payments_risk_ticket_tamper = softwareSystem "Manual review ticket tampering influences authorization result" "A compromised support or workflow path modifies review ticket outcomes before authorization completes." {
      tags "ThreatScenario,tampering,identified"
    }
    fu_fu_gitops_operations -> tb_tb_payments_platform_control "GitOps control-plane boundaries." "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_payment_authorization -> tb_tb_payments_external_bank "Authorization crosses external bank trust boundary." "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_checkout -> if_if_payments_checkout_api "Receives and validates checkout request payload." "Model relationship: calls" {
      tags "Mapping,calls"
    }
    fu_fu_payment_authorization -> if_if_payments_bank_auth "Calls external bank authorization interface." "Model relationship: calls" {
      tags "Mapping,calls"
    }
    fu_fu_payment_authorization -> if_if_payments_risk_score_api "Calls risk scoring interface for fraud decision support." "Model relationship: calls" {
      tags "Mapping,calls"
    }
    group_fg_fraud -> fu_fu_risk_scoring "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_fraud -> fu_fu_support_review "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_payments -> fu_fu_checkout "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_payments -> fu_fu_payment_authorization "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_platform -> fu_fu_cluster_provisioning "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_platform -> fu_fu_gitops_operations "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_checkout -> if_if_payments_checkout_api "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_payment_authorization -> data_do_payments_auth_decision "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_payment_authorization -> data_do_payments_auth_request "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_payment_authorization -> if_if_payments_bank_auth "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_risk_scoring -> data_do_payments_risk_signal "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_risk_scoring -> if_if_payments_risk_score_api "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_support_review -> data_do_payments_review_ticket "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_checkout -> fu_fu_payment_authorization "Delegates payment authorization." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_gitops_operations -> fu_fu_cluster_provisioning "Relies on provisioned cluster and namespace baseline." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_gitops_operations -> ref_ref_helm_platform "Uses Helm runtime platform for release orchestration." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_payment_authorization -> fu_fu_risk_scoring "Requests fraud scoring before final authorization." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_payment_authorization -> fu_fu_support_review "Escalates high-risk payments for manual review." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_payment_authorization -> ref_ref_bank_gateway_endpoint "Calls external bank authorization endpoint." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_risk_scoring -> ref_ref_postgres_driver "Uses external library for audit persistence." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    person_act_customer -> fu_fu_checkout "Submits payment at checkout." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_platform_operator -> fu_fu_cluster_provisioning "Maintains cluster baseline and environment setup." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_platform_operator -> fu_fu_gitops_operations "Operates delivery and release workflows." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_support -> fu_fu_support_review "Performs manual payment reviews." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    av_av_compromised_dependency -> ctrl_ctrl_payments_image_digest "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    av_av_malicious_api_request -> ctrl_ctrl_payments_sso_mfa "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    fu_fu_risk_scoring -> data_do_payments_risk_signal "Publishes risk signal updates for downstream consumers." "Model relationship: publishes" {
      tags "Mapping,publishes"
    }
    fu_fu_checkout -> data_do_payments_auth_decision "Reads final authorization decision for user-facing response." "Model relationship: reads" {
      tags "Mapping,reads"
    }
    fu_fu_payment_authorization -> data_do_payments_auth_request "Reads normalized request for authorization pipeline." "Model relationship: reads" {
      tags "Mapping,reads"
    }
    fu_fu_payment_authorization -> data_do_payments_review_ticket "Reads manual review outcome before returning decision." "Model relationship: reads" {
      tags "Mapping,reads"
    }
    fu_fu_payment_authorization -> data_do_payments_risk_signal "Reads risk signal before final authorization decision." "Model relationship: reads" {
      tags "Mapping,reads"
    }
    fu_fu_risk_scoring -> data_do_payments_auth_request "Reads normalized request to compute risk signal." "Model relationship: reads" {
      tags "Mapping,reads"
    }
    fu_fu_support_review -> data_do_payments_risk_signal "Subscribes to high-risk signal updates." "Model relationship: subscribes" {
      tags "Mapping,subscribes"
    }
    av_av_compromised_dependency -> ref_ref_postgres_driver "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_fraudulent_transaction_pattern -> fu_fu_payment_authorization "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_fraudulent_transaction_pattern -> fu_fu_risk_scoring "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_fraudulent_transaction_pattern -> fu_fu_support_review "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_malicious_api_request -> fu_fu_checkout "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_replayed_auth_callback -> fu_fu_payment_authorization "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    fu_fu_checkout -> data_do_payments_auth_request "Persists normalized authorization request." "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_payment_authorization -> data_do_payments_auth_decision "Persists final authorization decision." "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_risk_scoring -> data_do_payments_risk_signal "Persists computed risk signal." "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_support_review -> data_do_payments_review_ticket "Persists manual review ticket records." "Model relationship: writes" {
      tags "Mapping,writes"
    }
    person_act_customer -> fu_fu_checkout "Customer Checkout Authorization Flow" "https / public-api" {
      tags "Flow"
    }
    fu_fu_risk_scoring -> fu_fu_support_review "High-Risk Manual Review Flow" "internal-event / review-queue" {
      tags "Flow"
    }
    deploymentEnvironment "prod" {
      dn_dep_payments_bank_edge = deploymentNode "Bank Edge Integration Zone" "external us-east-1 partner" "bank-edge" {
        tags "DeploymentTarget,prod"
        softwareSystemInstance if_if_payments_bank_auth {
          tags "Deployed"
        }
      }
      dn_dep_payments_cluster_prod = deploymentNode "Payments Cluster Production" "shared-platform us-east-1 payments" "payments-prod" {
        tags "DeploymentTarget,prod"
        containerInstance fu_fu_checkout {
          tags "Deployed"
        }
        containerInstance fu_fu_payment_authorization {
          tags "Deployed"
        }
      }
    }
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
    dynamic sys_sample_payments_layered_model "dynamic_flow_customer_checkout" "End-to-end checkout authorization including fraud and bank integration boundaries." {
      person_act_customer -> fu_fu_checkout "Customer Checkout Authorization Flow" "https / public-api"
      autolayout lr
    }
    dynamic sys_sample_payments_layered_model "dynamic_flow_payments_manual_review" "Manual review escalation path for high-risk payment attempts." {
      fu_fu_risk_scoring -> fu_fu_support_review "High-Risk Manual Review Flow" "internal-event / review-queue"
      autolayout lr
    }
    deployment sys_sample_payments_layered_model "prod" "deployment_prod" "Deployment view for environment: prod" {
      include *
      autolayout lr
    }
    styles {
      element "FunctionalUnit" {
        shape RoundedBox
        background "#f8f5ec"
        color "#1f2a30"
      }
      element "DeploymentTarget" {
        shape Hexagon
        background "#edf4ff"
        color "#1f2a30"
      }
      element "ThreatScenario" {
        shape Diamond
        background "#ffeceb"
        color "#1f2a30"
      }
      relationship "Mapping" {
        color "#4b5b63"
      }
    }

    terminology {
      softwareSystem "System"
      container "Functional Unit"
      relationship "Mapping"
    }
  }

  configuration {
    scope softwaresystem
  }

}
