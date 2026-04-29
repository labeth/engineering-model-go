workspace "Sample Payments Layered Architecture" "This document describes the payments architecture using authored functional design and inferred runtime/code realization. It is intended to help product, platform, security, and implementation engineers reason about one shared model. Functional design is kept stable while realization details are inferred from infrastructure and source artifacts." {
  model {
    sys_sample_payments_layered_model = softwareSystem "Sample Payments Layered Architecture" "This document describes the payments architecture using authored functional design and inferred runtime/code realization. It is intended to help product, platform, security, and implementation engineers reason about one shared model. Functional design is kept stable while realization details are inferred from infrastructure and source artifacts." {
      group "Fraud Evaluation" {
        fu_fu_risk_scoring = container "Risk Scoring" "Risk scoring computes and classifies transaction risk before approval decisions are finalized. It provides a stable scoring contract to authorization and supports audit context for later analysis. The unit is focused on decision quality, consistency, and policy-driven classification behavior." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-FRAUD"
            "sourceId" "FU-RISK-SCORING"
          }
        }
        fu_fu_support_review = container "Support Review" "Support review handles manual decisions for escalated or ambiguous payment cases. It gives support operators context to approve, reject, or request additional verification in a controlled flow. The unit ensures manual intervention remains auditable, policy-constrained, and operationally reliable." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-FRAUD"
            "sourceId" "FU-SUPPORT-REVIEW"
          }
        }
      }
      group "Payments" {
        fu_fu_checkout = container "Checkout Handling" "Checkout handling is the user-facing entrypoint where payment requests are initiated and normalized. It validates request shape, preserves transaction context, and returns clear customer feedback for each outcome. It delegates decision logic to payment authorization while protecting the user experience boundary." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PAYMENTS"
            "sourceId" "FU-CHECKOUT"
          }
        }
        fu_fu_payment_authorization = container "Payment Authorization" "Payment authorization orchestrates the final transaction decision path. It coordinates fraud scoring, external bank interactions, and escalation to support review when needed. The unit returns deterministic outcomes so downstream behavior remains consistent and testable." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PAYMENTS"
            "sourceId" "FU-PAYMENT-AUTHORIZATION"
          }
        }
      }
      group "Platform" {
        fu_fu_cluster_provisioning = container "Cluster Provisioning" "Cluster provisioning creates the runtime substrate used by all application workloads. It establishes cluster and namespace structure, baseline controls, and environment-level readiness. This unit is responsible for predictable, repeatable infrastructure foundations." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PLATFORM"
            "sourceId" "FU-CLUSTER-PROVISIONING"
          }
        }
        fu_fu_gitops_operations = container "GitOps Operations" "GitOps operations continuously reconciles intended release state with actual runtime state. It governs rollout flow, drift correction, and operational delivery confidence across payment and fraud workloads. The unit provides safe and observable release behavior as an ongoing operational capability." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PLATFORM"
            "sourceId" "FU-GITOPS-OPERATIONS"
          }
        }
      }
    }
    person_act_customer = person "Customer" "Starts checkout and confirms payment." {
      tags "Actor"
      properties {
        "sourceId" "ACT-CUSTOMER"
      }
    }
    person_act_platform_operator = person "Platform Operator" "Operates GitOps and platform lifecycle." {
      tags "Actor"
      properties {
        "sourceId" "ACT-PLATFORM-OPERATOR"
      }
    }
    person_act_support = person "Support Agent" "Reviews high-risk and declined-payment cases." {
      tags "Actor"
      properties {
        "sourceId" "ACT-SUPPORT"
      }
    }
    group_fg_fraud = softwareSystem "Fraud Evaluation" "Risk scoring and fraud audit domain." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-FRAUD"
      }
    }
    group_fg_payments = softwareSystem "Payments" "Core payment checkout and authorization domain." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-PAYMENTS"
      }
    }
    group_fg_platform = softwareSystem "Platform" "Cluster provisioning and GitOps operations domain." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-PLATFORM"
      }
    }
    ref_ref_bank_gateway_endpoint = softwareSystem "Bank Gateway Endpoint" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-BANK-GATEWAY-ENDPOINT"
      }
    }
    ref_ref_helm_platform = softwareSystem "Helm Runtime Platform" "runtime" {
      tags "ReferencedElement,platform_service"
      properties {
        "kind" "platform_service"
        "layer" "runtime"
        "sourceId" "REF-HELM-PLATFORM"
      }
    }
    ref_ref_postgres_driver = softwareSystem "PostgreSQL Client Driver" "code" {
      tags "ReferencedElement,third_party_library"
      properties {
        "kind" "third_party_library"
        "layer" "code"
        "sourceId" "REF-POSTGRES-DRIVER"
      }
    }
    if_if_payments_bank_auth = softwareSystem "Bank Authorization Interface" "https /bank/authorize" {
      tags "Interface"
      properties {
        "endpoint" "/bank/authorize"
        "owner" "FU-PAYMENT-AUTHORIZATION"
        "protocol" "https"
        "sourceId" "IF-PAYMENTS-BANK-AUTH"
      }
    }
    if_if_payments_checkout_api = softwareSystem "Checkout API Interface" "https /api/checkout" {
      tags "Interface"
      properties {
        "endpoint" "/api/checkout"
        "owner" "FU-CHECKOUT"
        "protocol" "https"
        "sourceId" "IF-PAYMENTS-CHECKOUT-API"
      }
    }
    if_if_payments_risk_score_api = softwareSystem "Risk Score Interface" "https /risk/score" {
      tags "Interface"
      properties {
        "endpoint" "/risk/score"
        "owner" "FU-RISK-SCORING"
        "protocol" "https"
        "sourceId" "IF-PAYMENTS-RISK-SCORE-API"
      }
    }
    data_do_payments_auth_decision = softwareSystem "Authorization Decision" "schemas/auth-decision.json" {
      tags "DataObject,internal"
      properties {
        "classification" "regulated-internal"
        "retention" "400_days"
        "sourceId" "DO-PAYMENTS-AUTH-DECISION"
      }
    }
    data_do_payments_auth_request = softwareSystem "Authorization Request" "schemas/auth-request.json" {
      tags "DataObject,confidential"
      properties {
        "classification" "pci"
        "retention" "400_days"
        "sourceId" "DO-PAYMENTS-AUTH-REQUEST"
      }
    }
    data_do_payments_review_ticket = softwareSystem "Manual Review Ticket" "schemas/review-ticket.json" {
      tags "DataObject,confidential"
      properties {
        "classification" "pii"
        "retention" "730_days"
        "sourceId" "DO-PAYMENTS-REVIEW-TICKET"
      }
    }
    data_do_payments_risk_signal = softwareSystem "Risk Signal" "schemas/risk-signal.json" {
      tags "DataObject,internal"
      properties {
        "classification" "fraud-analytics"
        "retention" "365_days"
        "sourceId" "DO-PAYMENTS-RISK-SIGNAL"
      }
    }
    ctrl_ctrl_payments_callback_nonce = softwareSystem "Callback Nonce and Idempotency Guard" "Enforce nonce freshness and idempotency on bank callback processing." {
      tags "Control,protocol-integrity"
      properties {
        "sourceId" "CTRL-PAYMENTS-CALLBACK-NONCE"
      }
    }
    ctrl_ctrl_payments_image_digest = softwareSystem "Immutable Image Digests" "Enforce immutable digest-pinned container image references." {
      tags "Control,supply-chain"
      properties {
        "sourceId" "CTRL-PAYMENTS-IMAGE-DIGEST"
      }
    }
    ctrl_ctrl_payments_sso_mfa = softwareSystem "SSO MFA Enforcement" "Enforce MFA and conditional access for privileged operations." {
      tags "Control,identity-access"
      properties {
        "sourceId" "CTRL-PAYMENTS-SSO-MFA"
      }
    }
    av_av_compromised_dependency = softwareSystem "Compromised Dependency" "Supply chain compromise in external SDK or image." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-COMPROMISED-DEPENDENCY"
      }
    }
    av_av_fraudulent_transaction_pattern = softwareSystem "Fraudulent Transaction Pattern" "Coordinated abuse pattern using synthetic identities, stolen instruments, and velocity anomalies to seek unauthorized approvals." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-FRAUDULENT-TRANSACTION-PATTERN"
      }
    }
    av_av_malicious_api_request = softwareSystem "Malicious API Request" "Crafted payloads attempting to bypass authorization controls." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-MALICIOUS-API-REQUEST"
      }
    }
    av_av_replayed_auth_callback = softwareSystem "Replayed Authorization Callback" "Replay attack against external authorization callback flow." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-REPLAYED-AUTH-CALLBACK"
      }
    }
    tb_tb_payments_external_bank = softwareSystem "External Bank Boundary" "Separates internal payment authorization from external bank services." {
      tags "TrustBoundary,network"
      properties {
        "sourceId" "TB-PAYMENTS-EXTERNAL-BANK"
      }
    }
    tb_tb_payments_platform_control = softwareSystem "Platform Control Boundary" "Separates app workloads from platform control-plane authority." {
      tags "TrustBoundary,control-plane"
      properties {
        "sourceId" "TB-PAYMENTS-PLATFORM-CONTROL"
      }
    }
    ts_ts_payments_bank_callback_replay = softwareSystem "Replayed bank callback triggers duplicate authorization transition" "Replayed callback message tries to overwrite or duplicate prior decision outcomes." {
      tags "ThreatScenario,replay,mitigating"
      properties {
        "impact" "high"
        "likelihood" "medium"
        "severity" "high"
        "sourceId" "TS-PAYMENTS-BANK-CALLBACK-REPLAY"
      }
    }
    ts_ts_payments_checkout_spoofing = softwareSystem "Checkout payload spoofing bypasses normalization checks" "Crafted payload shape attempts to force ambiguous authorization behavior and bypass policy checks." {
      tags "ThreatScenario,tampering,mitigating"
      properties {
        "impact" "high"
        "likelihood" "medium"
        "severity" "high"
        "sourceId" "TS-PAYMENTS-CHECKOUT-SPOOFING"
      }
    }
    ts_ts_payments_risk_ticket_tamper = softwareSystem "Manual review ticket tampering influences authorization result" "A compromised support or workflow path modifies review ticket outcomes before authorization completes." {
      tags "ThreatScenario,tampering,identified"
      properties {
        "impact" "high"
        "likelihood" "low"
        "severity" "medium"
        "sourceId" "TS-PAYMENTS-RISK-TICKET-TAMPER"
      }
    }
    fu_fu_gitops_operations -> tb_tb_payments_platform_control "GitOps control-plane boundaries." {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-GITOPS-OPERATIONS"
        "mappingType" "bounded_by"
        "toId" "TB-PAYMENTS-PLATFORM-CONTROL"
      }
    }
    fu_fu_payment_authorization -> tb_tb_payments_external_bank "Authorization crosses external bank trust boundary." {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "bounded_by"
        "toId" "TB-PAYMENTS-EXTERNAL-BANK"
      }
    }
    fu_fu_checkout -> if_if_payments_checkout_api "Receives and validates checkout request payload." {
      tags "Mapping,calls"
      properties {
        "fromId" "FU-CHECKOUT"
        "mappingType" "calls"
        "toId" "IF-PAYMENTS-CHECKOUT-API"
      }
    }
    fu_fu_payment_authorization -> if_if_payments_bank_auth "Calls external bank authorization interface." {
      tags "Mapping,calls"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "calls"
        "toId" "IF-PAYMENTS-BANK-AUTH"
      }
    }
    fu_fu_payment_authorization -> if_if_payments_risk_score_api "Calls risk scoring interface for fraud decision support." {
      tags "Mapping,calls"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "calls"
        "toId" "IF-PAYMENTS-RISK-SCORE-API"
      }
    }
    group_fg_fraud -> fu_fu_risk_scoring "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-FRAUD"
        "mappingType" "contains"
        "toId" "FU-RISK-SCORING"
      }
    }
    group_fg_fraud -> fu_fu_support_review "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-FRAUD"
        "mappingType" "contains"
        "toId" "FU-SUPPORT-REVIEW"
      }
    }
    group_fg_payments -> fu_fu_checkout "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PAYMENTS"
        "mappingType" "contains"
        "toId" "FU-CHECKOUT"
      }
    }
    group_fg_payments -> fu_fu_payment_authorization "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PAYMENTS"
        "mappingType" "contains"
        "toId" "FU-PAYMENT-AUTHORIZATION"
      }
    }
    group_fg_platform -> fu_fu_cluster_provisioning "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PLATFORM"
        "mappingType" "contains"
        "toId" "FU-CLUSTER-PROVISIONING"
      }
    }
    group_fg_platform -> fu_fu_gitops_operations "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PLATFORM"
        "mappingType" "contains"
        "toId" "FU-GITOPS-OPERATIONS"
      }
    }
    fu_fu_checkout -> if_if_payments_checkout_api "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-CHECKOUT"
        "mappingType" "contains"
        "toId" "IF-PAYMENTS-CHECKOUT-API"
      }
    }
    fu_fu_payment_authorization -> data_do_payments_auth_decision "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "contains"
        "toId" "DO-PAYMENTS-AUTH-DECISION"
      }
    }
    fu_fu_payment_authorization -> data_do_payments_auth_request "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "contains"
        "toId" "DO-PAYMENTS-AUTH-REQUEST"
      }
    }
    fu_fu_payment_authorization -> if_if_payments_bank_auth "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "contains"
        "toId" "IF-PAYMENTS-BANK-AUTH"
      }
    }
    fu_fu_risk_scoring -> data_do_payments_risk_signal "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-RISK-SCORING"
        "mappingType" "contains"
        "toId" "DO-PAYMENTS-RISK-SIGNAL"
      }
    }
    fu_fu_risk_scoring -> if_if_payments_risk_score_api "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-RISK-SCORING"
        "mappingType" "contains"
        "toId" "IF-PAYMENTS-RISK-SCORE-API"
      }
    }
    fu_fu_support_review -> data_do_payments_review_ticket "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-SUPPORT-REVIEW"
        "mappingType" "contains"
        "toId" "DO-PAYMENTS-REVIEW-TICKET"
      }
    }
    fu_fu_checkout -> fu_fu_payment_authorization "Delegates payment authorization." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-CHECKOUT"
        "mappingType" "depends_on"
        "toId" "FU-PAYMENT-AUTHORIZATION"
      }
    }
    fu_fu_gitops_operations -> fu_fu_cluster_provisioning "Relies on provisioned cluster and namespace baseline." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-GITOPS-OPERATIONS"
        "mappingType" "depends_on"
        "toId" "FU-CLUSTER-PROVISIONING"
      }
    }
    fu_fu_gitops_operations -> ref_ref_helm_platform "Uses Helm runtime platform for release orchestration." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-GITOPS-OPERATIONS"
        "mappingType" "depends_on"
        "toId" "REF-HELM-PLATFORM"
      }
    }
    fu_fu_payment_authorization -> fu_fu_risk_scoring "Requests fraud scoring before final authorization." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "depends_on"
        "toId" "FU-RISK-SCORING"
      }
    }
    fu_fu_payment_authorization -> fu_fu_support_review "Escalates high-risk payments for manual review." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "depends_on"
        "toId" "FU-SUPPORT-REVIEW"
      }
    }
    fu_fu_payment_authorization -> ref_ref_bank_gateway_endpoint "Calls external bank authorization endpoint." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "depends_on"
        "toId" "REF-BANK-GATEWAY-ENDPOINT"
      }
    }
    fu_fu_risk_scoring -> ref_ref_postgres_driver "Uses external library for audit persistence." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-RISK-SCORING"
        "mappingType" "depends_on"
        "toId" "REF-POSTGRES-DRIVER"
      }
    }
    person_act_customer -> fu_fu_checkout "Submits payment at checkout." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-CUSTOMER"
        "mappingType" "interacts_with"
        "toId" "FU-CHECKOUT"
      }
    }
    person_act_platform_operator -> fu_fu_cluster_provisioning "Maintains cluster baseline and environment setup." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-PLATFORM-OPERATOR"
        "mappingType" "interacts_with"
        "toId" "FU-CLUSTER-PROVISIONING"
      }
    }
    person_act_platform_operator -> fu_fu_gitops_operations "Operates delivery and release workflows." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-PLATFORM-OPERATOR"
        "mappingType" "interacts_with"
        "toId" "FU-GITOPS-OPERATIONS"
      }
    }
    person_act_support -> fu_fu_support_review "Performs manual payment reviews." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-SUPPORT"
        "mappingType" "interacts_with"
        "toId" "FU-SUPPORT-REVIEW"
      }
    }
    av_av_compromised_dependency -> ctrl_ctrl_payments_image_digest "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-COMPROMISED-DEPENDENCY"
        "mappingType" "mitigated_by"
        "toId" "CTRL-PAYMENTS-IMAGE-DIGEST"
      }
    }
    av_av_malicious_api_request -> ctrl_ctrl_payments_sso_mfa "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-MALICIOUS-API-REQUEST"
        "mappingType" "mitigated_by"
        "toId" "CTRL-PAYMENTS-SSO-MFA"
      }
    }
    fu_fu_risk_scoring -> data_do_payments_risk_signal "Publishes risk signal updates for downstream consumers." {
      tags "Mapping,publishes"
      properties {
        "fromId" "FU-RISK-SCORING"
        "mappingType" "publishes"
        "toId" "DO-PAYMENTS-RISK-SIGNAL"
      }
    }
    fu_fu_checkout -> data_do_payments_auth_decision "Reads final authorization decision for user-facing response." {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-CHECKOUT"
        "mappingType" "reads"
        "toId" "DO-PAYMENTS-AUTH-DECISION"
      }
    }
    fu_fu_payment_authorization -> data_do_payments_auth_request "Reads normalized request for authorization pipeline." {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "reads"
        "toId" "DO-PAYMENTS-AUTH-REQUEST"
      }
    }
    fu_fu_payment_authorization -> data_do_payments_review_ticket "Reads manual review outcome before returning decision." {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "reads"
        "toId" "DO-PAYMENTS-REVIEW-TICKET"
      }
    }
    fu_fu_payment_authorization -> data_do_payments_risk_signal "Reads risk signal before final authorization decision." {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "reads"
        "toId" "DO-PAYMENTS-RISK-SIGNAL"
      }
    }
    fu_fu_risk_scoring -> data_do_payments_auth_request "Reads normalized request to compute risk signal." {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-RISK-SCORING"
        "mappingType" "reads"
        "toId" "DO-PAYMENTS-AUTH-REQUEST"
      }
    }
    fu_fu_support_review -> data_do_payments_risk_signal "Subscribes to high-risk signal updates." {
      tags "Mapping,subscribes"
      properties {
        "fromId" "FU-SUPPORT-REVIEW"
        "mappingType" "subscribes"
        "toId" "DO-PAYMENTS-RISK-SIGNAL"
      }
    }
    av_av_compromised_dependency -> ref_ref_postgres_driver "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-COMPROMISED-DEPENDENCY"
        "mappingType" "targets"
        "toId" "REF-POSTGRES-DRIVER"
      }
    }
    av_av_fraudulent_transaction_pattern -> fu_fu_payment_authorization "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-FRAUDULENT-TRANSACTION-PATTERN"
        "mappingType" "targets"
        "toId" "FU-PAYMENT-AUTHORIZATION"
      }
    }
    av_av_fraudulent_transaction_pattern -> fu_fu_risk_scoring "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-FRAUDULENT-TRANSACTION-PATTERN"
        "mappingType" "targets"
        "toId" "FU-RISK-SCORING"
      }
    }
    av_av_fraudulent_transaction_pattern -> fu_fu_support_review "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-FRAUDULENT-TRANSACTION-PATTERN"
        "mappingType" "targets"
        "toId" "FU-SUPPORT-REVIEW"
      }
    }
    av_av_malicious_api_request -> fu_fu_checkout "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-MALICIOUS-API-REQUEST"
        "mappingType" "targets"
        "toId" "FU-CHECKOUT"
      }
    }
    av_av_replayed_auth_callback -> fu_fu_payment_authorization "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-REPLAYED-AUTH-CALLBACK"
        "mappingType" "targets"
        "toId" "FU-PAYMENT-AUTHORIZATION"
      }
    }
    fu_fu_checkout -> data_do_payments_auth_request "Persists normalized authorization request." {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-CHECKOUT"
        "mappingType" "writes"
        "toId" "DO-PAYMENTS-AUTH-REQUEST"
      }
    }
    fu_fu_payment_authorization -> data_do_payments_auth_decision "Persists final authorization decision." {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-PAYMENT-AUTHORIZATION"
        "mappingType" "writes"
        "toId" "DO-PAYMENTS-AUTH-DECISION"
      }
    }
    fu_fu_risk_scoring -> data_do_payments_risk_signal "Persists computed risk signal." {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-RISK-SCORING"
        "mappingType" "writes"
        "toId" "DO-PAYMENTS-RISK-SIGNAL"
      }
    }
    fu_fu_support_review -> data_do_payments_review_ticket "Persists manual review ticket records." {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-SUPPORT-REVIEW"
        "mappingType" "writes"
        "toId" "DO-PAYMENTS-REVIEW-TICKET"
      }
    }
    fu_fu_risk_scoring -> fu_fu_support_review "High-Risk Manual Review Flow" {
      tags "Flow"
      properties {
        "flowId" "FLOW-PAYMENTS-MANUAL-REVIEW"
      }
    }
    deploymentEnvironment "prod" {
      dn_dep_payments_bank_edge = deploymentNode "Bank Edge Integration Zone" "external us-east-1 partner" "bank-edge" {
        tags "DeploymentTarget,prod"
        properties {
          "account" "external"
          "cluster" "bank-edge"
          "environment" "prod"
          "namespace" "partner"
          "region" "us-east-1"
          "sourceId" "DEP-PAYMENTS-BANK-EDGE"
          "trustZone" "external"
        }
        softwareSystemInstance if_if_payments_bank_auth {
          tags "Deployed"
          properties {
            "sourceId" "IF-PAYMENTS-BANK-AUTH"
          }
        }
      }
      dn_dep_payments_cluster_prod = deploymentNode "Payments Cluster Production" "shared-platform us-east-1 payments" "payments-prod" {
        tags "DeploymentTarget,prod"
        properties {
          "account" "shared-platform"
          "cluster" "payments-prod"
          "environment" "prod"
          "namespace" "payments"
          "region" "us-east-1"
          "sourceId" "DEP-PAYMENTS-CLUSTER-PROD"
          "trustZone" "app"
        }
        containerInstance fu_fu_checkout {
          tags "Deployed"
          properties {
            "sourceId" "FU-CHECKOUT"
          }
        }
        containerInstance fu_fu_payment_authorization {
          tags "Deployed"
          properties {
            "sourceId" "FU-PAYMENT-AUTHORIZATION"
          }
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
      person_act_customer -> fu_fu_checkout "End-to-end checkout authorization including fraud and bank integration boundaries."
      autolayout lr
    }
    dynamic sys_sample_payments_layered_model "dynamic_flow_payments_manual_review" "Manual review escalation path for high-risk payment attempts." {
      fu_fu_risk_scoring -> fu_fu_support_review "Manual review escalation path for high-risk payment attempts."
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

}
