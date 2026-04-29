workspace "Bedrock Lambda GitHub PR Review App Architecture" "This document describes a GitHub App that reviews pull requests using AWS Bedrock and Lambda. The authored functional architecture remains stable while deployment/runtime/code evidence is inferred from infrastructure manifests, source files, and verification artifacts." {
  model {
    sys_sample_bedrock_pr_review_model = softwareSystem "Bedrock Lambda GitHub PR Review App Architecture" "This document describes a GitHub App that reviews pull requests using AWS Bedrock and Lambda. The authored functional architecture remains stable while deployment/runtime/code evidence is inferred from infrastructure manifests, source files, and verification artifacts." {
      group "PR Integration" {
        fu_fu_github_webhook_ingress = container "GitHub Webhook Ingress" "Verifies webhook signatures, validates event type, and starts review workflow routing." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PR-INTEGRATION"
            "sourceId" "FU-GITHUB-WEBHOOK-INGRESS"
          }
        }
        fu_fu_pr_context_assembly = container "PR Context Assembly" "Fetches changed files and diff metadata and prepares bounded review context payloads." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PR-INTEGRATION"
            "sourceId" "FU-PR-CONTEXT-ASSEMBLY"
          }
        }
      }
      group "Platform Operations" {
        fu_fu_lambda_runtime_operations = container "Lambda Runtime Operations" "Manages Lambda packaging/deployment lifecycle and operational health guardrails." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PLATFORM-OPERATIONS"
            "sourceId" "FU-LAMBDA-RUNTIME-OPERATIONS"
          }
        }
        fu_fu_secrets_configuration = container "Secrets and Configuration" "Manages GitHub App credentials, Bedrock config, and redaction-safe runtime configuration." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PLATFORM-OPERATIONS"
            "sourceId" "FU-SECRETS-CONFIGURATION"
          }
        }
      }
      group "Review Intelligence" {
        fu_fu_policy_checks = container "Policy Checks" "Runs deterministic checks for secrets, risky IaC changes, and repository policy constraints." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-REVIEW-INTELLIGENCE"
            "sourceId" "FU-POLICY-CHECKS"
          }
        }
        fu_fu_review_orchestration = container "Review Orchestration" "Calls Bedrock models, merges AI and deterministic findings, and handles graceful degradation paths." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-REVIEW-INTELLIGENCE"
            "sourceId" "FU-REVIEW-ORCHESTRATION"
          }
        }
        fu_fu_review_publication = container "Review Publication" "Publishes check-run summaries and inline comments back to GitHub pull requests." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-REVIEW-INTELLIGENCE"
            "sourceId" "FU-REVIEW-PUBLICATION"
          }
        }
      }
    }
    person_act_developer = person "Developer" "Opens pull requests and receives review feedback." {
      tags "Actor"
      properties {
        "sourceId" "ACT-DEVELOPER"
      }
    }
    person_act_platform_operator = person "Platform Operator" "Operates Lambda deployment and runtime controls." {
      tags "Actor"
      properties {
        "sourceId" "ACT-PLATFORM-OPERATOR"
      }
    }
    person_act_repository_maintainer = person "Repository Maintainer" "Owns merge decisions and repository-level quality gates." {
      tags "Actor"
      properties {
        "sourceId" "ACT-REPOSITORY-MAINTAINER"
      }
    }
    person_act_security_engineer = person "Security Engineer" "Curates policy checks and triages security findings." {
      tags "Actor"
      properties {
        "sourceId" "ACT-SECURITY-ENGINEER"
      }
    }
    group_fg_platform_operations = softwareSystem "Platform Operations" "Lambda deployment lifecycle, runtime operations, and secrets/configuration." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-PLATFORM-OPERATIONS"
      }
    }
    group_fg_pr_integration = softwareSystem "PR Integration" "Ingress and context assembly for GitHub pull request events." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-PR-INTEGRATION"
      }
    }
    group_fg_review_intelligence = softwareSystem "Review Intelligence" "AI and deterministic review analysis with result publication." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-REVIEW-INTELLIGENCE"
      }
    }
    ref_ref_aws_bedrock_runtime = softwareSystem "AWS Bedrock Runtime API" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-AWS-BEDROCK-RUNTIME"
      }
    }
    ref_ref_aws_sdk_go = softwareSystem "AWS SDK for Go" "code" {
      tags "ReferencedElement,third_party_library"
      properties {
        "kind" "third_party_library"
        "layer" "code"
        "sourceId" "REF-AWS-SDK-GO"
      }
    }
    ref_ref_aws_secrets_manager = softwareSystem "AWS Secrets Manager" "runtime" {
      tags "ReferencedElement,platform_service"
      properties {
        "kind" "platform_service"
        "layer" "runtime"
        "sourceId" "REF-AWS-SECRETS-MANAGER"
      }
    }
    ref_ref_github_app_api = softwareSystem "GitHub App API" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-GITHUB-APP-API"
      }
    }
    ref_ref_github_rest_sdk = softwareSystem "GitHub REST SDK" "code" {
      tags "ReferencedElement,third_party_library"
      properties {
        "kind" "third_party_library"
        "layer" "code"
        "sourceId" "REF-GITHUB-REST-SDK"
      }
    }
    ref_ref_policy_ruleset = softwareSystem "Review Policy Ruleset" "code" {
      tags "ReferencedElement,policy_bundle"
      properties {
        "kind" "policy_bundle"
        "layer" "code"
        "sourceId" "REF-POLICY-RULESET"
      }
    }
    if_if_bedrock_review_publish_api = softwareSystem "Review Publish API" "https /github/check-runs" {
      tags "Interface"
      properties {
        "endpoint" "/github/check-runs"
        "owner" "FU-REVIEW-PUBLICATION"
        "protocol" "https"
        "sourceId" "IF-BEDROCK-REVIEW-PUBLISH-API"
      }
    }
    if_if_bedrock_webhook_api = softwareSystem "Webhook Intake API" "https /webhook/github" {
      tags "Interface"
      properties {
        "endpoint" "/webhook/github"
        "owner" "FU-GITHUB-WEBHOOK-INGRESS"
        "protocol" "https"
        "sourceId" "IF-BEDROCK-WEBHOOK-API"
      }
    }
    data_do_bedrock_pr_context = softwareSystem "Pull Request Context" "schemas/pr-context.json" {
      tags "DataObject,internal"
      properties {
        "classification" "source-intelligence"
        "retention" "90_days"
        "sourceId" "DO-BEDROCK-PR-CONTEXT"
      }
    }
    data_do_bedrock_review_findings = softwareSystem "Review Findings" "schemas/review-findings.json" {
      tags "DataObject,internal"
      properties {
        "classification" "security-analysis"
        "retention" "365_days"
        "sourceId" "DO-BEDROCK-REVIEW-FINDINGS"
      }
    }
    ctrl_ctrl_bedrock_secrets_rotation = softwareSystem "Secret Rotation Enforcement" "Rotate app and model credentials according to policy." {
      tags "Control,credential-management"
      properties {
        "sourceId" "CTRL-BEDROCK-SECRETS-ROTATION"
      }
    }
    ctrl_ctrl_bedrock_webhook_signature = softwareSystem "Webhook Signature Verification" "Enforce signature verification for all GitHub webhook events." {
      tags "Control,input-integrity"
      properties {
        "sourceId" "CTRL-BEDROCK-WEBHOOK-SIGNATURE"
      }
    }
    av_av_pr_spam_abuse = softwareSystem "Pull Request Spam Abuse" "High-rate PR event flooding that can exhaust review capacity." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-PR-SPAM-ABUSE"
      }
    }
    av_av_prompt_injection_in_diff = softwareSystem "Prompt Injection in Diff" "Malicious prompt-like content embedded in pull request diffs." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-PROMPT-INJECTION-IN-DIFF"
      }
    }
    av_av_secret_leakage_in_review = softwareSystem "Secret Leakage in Review Output" "Generated findings accidentally exposing secrets/tokens in comments." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-SECRET-LEAKAGE-IN-REVIEW"
      }
    }
    av_av_spoofed_github_webhook = softwareSystem "Spoofed GitHub Webhook" "Forged webhook payload attempting unauthorized review execution." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-SPOOFED-GITHUB-WEBHOOK"
      }
    }
    tb_tb_bedrock_aws_control = softwareSystem "AWS Control Boundary" "Boundary between app workloads and privileged platform controls." {
      tags "TrustBoundary,control-plane"
      properties {
        "sourceId" "TB-BEDROCK-AWS-CONTROL"
      }
    }
    tb_tb_bedrock_github_external = softwareSystem "GitHub External Boundary" "Boundary between external GitHub traffic and internal review processing." {
      tags "TrustBoundary,network"
      properties {
        "sourceId" "TB-BEDROCK-GITHUB-EXTERNAL"
      }
    }
    ts_ts_bedrock_prompt_injection = softwareSystem "Prompt injection in diff influences review output integrity" "Malicious prompts in diff content attempt to coerce model output away from policy intent." {
      tags "ThreatScenario,tampering,mitigating"
      properties {
        "impact" "medium"
        "likelihood" "medium"
        "severity" "medium"
        "sourceId" "TS-BEDROCK-PROMPT-INJECTION"
      }
    }
    ts_ts_bedrock_secret_leak_in_publication = softwareSystem "Sensitive tokens leak via review publication payload" "Review output includes sensitive content due to incomplete redaction before publication." {
      tags "ThreatScenario,information-disclosure,mitigating"
      properties {
        "impact" "high"
        "likelihood" "low"
        "severity" "medium"
        "sourceId" "TS-BEDROCK-SECRET-LEAK-IN-PUBLICATION"
      }
    }
    ts_ts_bedrock_webhook_spoof = softwareSystem "Spoofed webhook bypasses ingress verification" "Forged pull request webhook attempts to trigger unauthorized review execution." {
      tags "ThreatScenario,spoofing,mitigating"
      properties {
        "impact" "high"
        "likelihood" "medium"
        "severity" "high"
        "sourceId" "TS-BEDROCK-WEBHOOK-SPOOF"
      }
    }
    fu_fu_github_webhook_ingress -> tb_tb_bedrock_github_external "External ingress trust boundary." {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-GITHUB-WEBHOOK-INGRESS"
        "mappingType" "bounded_by"
        "toId" "TB-BEDROCK-GITHUB-EXTERNAL"
      }
    }
    fu_fu_lambda_runtime_operations -> tb_tb_bedrock_aws_control "Runtime platform control boundary." {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-LAMBDA-RUNTIME-OPERATIONS"
        "mappingType" "bounded_by"
        "toId" "TB-BEDROCK-AWS-CONTROL"
      }
    }
    fu_fu_github_webhook_ingress -> if_if_bedrock_webhook_api "Receives webhook payload through signed API endpoint." {
      tags "Mapping,calls"
      properties {
        "fromId" "FU-GITHUB-WEBHOOK-INGRESS"
        "mappingType" "calls"
        "toId" "IF-BEDROCK-WEBHOOK-API"
      }
    }
    fu_fu_review_publication -> if_if_bedrock_review_publish_api "Calls publication API for check-run output." {
      tags "Mapping,calls"
      properties {
        "fromId" "FU-REVIEW-PUBLICATION"
        "mappingType" "calls"
        "toId" "IF-BEDROCK-REVIEW-PUBLISH-API"
      }
    }
    group_fg_platform_operations -> fu_fu_lambda_runtime_operations "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PLATFORM-OPERATIONS"
        "mappingType" "contains"
        "toId" "FU-LAMBDA-RUNTIME-OPERATIONS"
      }
    }
    group_fg_platform_operations -> fu_fu_secrets_configuration "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PLATFORM-OPERATIONS"
        "mappingType" "contains"
        "toId" "FU-SECRETS-CONFIGURATION"
      }
    }
    group_fg_pr_integration -> fu_fu_github_webhook_ingress "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PR-INTEGRATION"
        "mappingType" "contains"
        "toId" "FU-GITHUB-WEBHOOK-INGRESS"
      }
    }
    group_fg_pr_integration -> fu_fu_pr_context_assembly "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PR-INTEGRATION"
        "mappingType" "contains"
        "toId" "FU-PR-CONTEXT-ASSEMBLY"
      }
    }
    group_fg_review_intelligence -> fu_fu_policy_checks "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-REVIEW-INTELLIGENCE"
        "mappingType" "contains"
        "toId" "FU-POLICY-CHECKS"
      }
    }
    group_fg_review_intelligence -> fu_fu_review_orchestration "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-REVIEW-INTELLIGENCE"
        "mappingType" "contains"
        "toId" "FU-REVIEW-ORCHESTRATION"
      }
    }
    group_fg_review_intelligence -> fu_fu_review_publication "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-REVIEW-INTELLIGENCE"
        "mappingType" "contains"
        "toId" "FU-REVIEW-PUBLICATION"
      }
    }
    fu_fu_github_webhook_ingress -> if_if_bedrock_webhook_api "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-GITHUB-WEBHOOK-INGRESS"
        "mappingType" "contains"
        "toId" "IF-BEDROCK-WEBHOOK-API"
      }
    }
    fu_fu_review_orchestration -> data_do_bedrock_pr_context "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-REVIEW-ORCHESTRATION"
        "mappingType" "contains"
        "toId" "DO-BEDROCK-PR-CONTEXT"
      }
    }
    fu_fu_review_publication -> data_do_bedrock_review_findings "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-REVIEW-PUBLICATION"
        "mappingType" "contains"
        "toId" "DO-BEDROCK-REVIEW-FINDINGS"
      }
    }
    fu_fu_review_publication -> if_if_bedrock_review_publish_api "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-REVIEW-PUBLICATION"
        "mappingType" "contains"
        "toId" "IF-BEDROCK-REVIEW-PUBLISH-API"
      }
    }
    fu_fu_github_webhook_ingress -> fu_fu_pr_context_assembly "Forwards authenticated PR events for context assembly." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-GITHUB-WEBHOOK-INGRESS"
        "mappingType" "depends_on"
        "toId" "FU-PR-CONTEXT-ASSEMBLY"
      }
    }
    fu_fu_lambda_runtime_operations -> fu_fu_secrets_configuration "Requires secure runtime configuration and credential access." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-LAMBDA-RUNTIME-OPERATIONS"
        "mappingType" "depends_on"
        "toId" "FU-SECRETS-CONFIGURATION"
      }
    }
    fu_fu_lambda_runtime_operations -> ref_ref_aws_secrets_manager "Uses managed secret store for runtime credentials." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-LAMBDA-RUNTIME-OPERATIONS"
        "mappingType" "depends_on"
        "toId" "REF-AWS-SECRETS-MANAGER"
      }
    }
    fu_fu_policy_checks -> ref_ref_policy_ruleset "Evaluates repository policy and risk rules." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-POLICY-CHECKS"
        "mappingType" "depends_on"
        "toId" "REF-POLICY-RULESET"
      }
    }
    fu_fu_pr_context_assembly -> fu_fu_review_orchestration "Sends normalized review context." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-PR-CONTEXT-ASSEMBLY"
        "mappingType" "depends_on"
        "toId" "FU-REVIEW-ORCHESTRATION"
      }
    }
    fu_fu_pr_context_assembly -> ref_ref_github_app_api "Fetches pull request files and metadata." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-PR-CONTEXT-ASSEMBLY"
        "mappingType" "depends_on"
        "toId" "REF-GITHUB-APP-API"
      }
    }
    fu_fu_review_orchestration -> fu_fu_policy_checks "Merges deterministic policy findings with model output." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-REVIEW-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-POLICY-CHECKS"
      }
    }
    fu_fu_review_orchestration -> fu_fu_review_publication "Submits final findings for publication." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-REVIEW-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-REVIEW-PUBLICATION"
      }
    }
    fu_fu_review_orchestration -> fu_fu_secrets_configuration "Retrieves model configuration and safety settings." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-REVIEW-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-SECRETS-CONFIGURATION"
      }
    }
    fu_fu_review_orchestration -> ref_ref_aws_bedrock_runtime "Requests model analysis for code review findings." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-REVIEW-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "REF-AWS-BEDROCK-RUNTIME"
      }
    }
    fu_fu_review_publication -> fu_fu_secrets_configuration "Retrieves app credentials and redaction rules." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-REVIEW-PUBLICATION"
        "mappingType" "depends_on"
        "toId" "FU-SECRETS-CONFIGURATION"
      }
    }
    fu_fu_review_publication -> ref_ref_github_app_api "Publishes check runs and inline review comments." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-REVIEW-PUBLICATION"
        "mappingType" "depends_on"
        "toId" "REF-GITHUB-APP-API"
      }
    }
    person_act_developer -> fu_fu_github_webhook_ingress "Developer push/update activity triggers pull request webhook events." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-DEVELOPER"
        "mappingType" "interacts_with"
        "toId" "FU-GITHUB-WEBHOOK-INGRESS"
      }
    }
    person_act_platform_operator -> fu_fu_lambda_runtime_operations "Operates deployments, scaling controls, and runtime observability." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-PLATFORM-OPERATOR"
        "mappingType" "interacts_with"
        "toId" "FU-LAMBDA-RUNTIME-OPERATIONS"
      }
    }
    person_act_platform_operator -> fu_fu_secrets_configuration "Rotates credentials and manages secure configuration lifecycle." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-PLATFORM-OPERATOR"
        "mappingType" "interacts_with"
        "toId" "FU-SECRETS-CONFIGURATION"
      }
    }
    person_act_repository_maintainer -> fu_fu_review_publication "Reviews findings and gate decisions before merge." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-REPOSITORY-MAINTAINER"
        "mappingType" "interacts_with"
        "toId" "FU-REVIEW-PUBLICATION"
      }
    }
    person_act_security_engineer -> fu_fu_policy_checks "Maintains and evolves deterministic security checks." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-SECURITY-ENGINEER"
        "mappingType" "interacts_with"
        "toId" "FU-POLICY-CHECKS"
      }
    }
    av_av_secret_leakage_in_review -> ctrl_ctrl_bedrock_secrets_rotation "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-SECRET-LEAKAGE-IN-REVIEW"
        "mappingType" "mitigated_by"
        "toId" "CTRL-BEDROCK-SECRETS-ROTATION"
      }
    }
    av_av_spoofed_github_webhook -> ctrl_ctrl_bedrock_webhook_signature "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-SPOOFED-GITHUB-WEBHOOK"
        "mappingType" "mitigated_by"
        "toId" "CTRL-BEDROCK-WEBHOOK-SIGNATURE"
      }
    }
    fu_fu_review_orchestration -> data_do_bedrock_pr_context "Reads normalized context before model call." {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-REVIEW-ORCHESTRATION"
        "mappingType" "reads"
        "toId" "DO-BEDROCK-PR-CONTEXT"
      }
    }
    fu_fu_review_publication -> data_do_bedrock_review_findings "Reads final findings for publication." {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-REVIEW-PUBLICATION"
        "mappingType" "reads"
        "toId" "DO-BEDROCK-REVIEW-FINDINGS"
      }
    }
    av_av_pr_spam_abuse -> fu_fu_github_webhook_ingress "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-PR-SPAM-ABUSE"
        "mappingType" "targets"
        "toId" "FU-GITHUB-WEBHOOK-INGRESS"
      }
    }
    av_av_pr_spam_abuse -> fu_fu_review_orchestration "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-PR-SPAM-ABUSE"
        "mappingType" "targets"
        "toId" "FU-REVIEW-ORCHESTRATION"
      }
    }
    av_av_prompt_injection_in_diff -> fu_fu_pr_context_assembly "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-PROMPT-INJECTION-IN-DIFF"
        "mappingType" "targets"
        "toId" "FU-PR-CONTEXT-ASSEMBLY"
      }
    }
    av_av_prompt_injection_in_diff -> fu_fu_review_orchestration "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-PROMPT-INJECTION-IN-DIFF"
        "mappingType" "targets"
        "toId" "FU-REVIEW-ORCHESTRATION"
      }
    }
    av_av_secret_leakage_in_review -> fu_fu_review_publication "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-SECRET-LEAKAGE-IN-REVIEW"
        "mappingType" "targets"
        "toId" "FU-REVIEW-PUBLICATION"
      }
    }
    av_av_secret_leakage_in_review -> fu_fu_secrets_configuration "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-SECRET-LEAKAGE-IN-REVIEW"
        "mappingType" "targets"
        "toId" "FU-SECRETS-CONFIGURATION"
      }
    }
    av_av_spoofed_github_webhook -> fu_fu_github_webhook_ingress "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-SPOOFED-GITHUB-WEBHOOK"
        "mappingType" "targets"
        "toId" "FU-GITHUB-WEBHOOK-INGRESS"
      }
    }
    fu_fu_pr_context_assembly -> data_do_bedrock_pr_context "Persists normalized context payload." {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-PR-CONTEXT-ASSEMBLY"
        "mappingType" "writes"
        "toId" "DO-BEDROCK-PR-CONTEXT"
      }
    }
    fu_fu_review_orchestration -> data_do_bedrock_review_findings "Persists merged deterministic and AI findings." {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-REVIEW-ORCHESTRATION"
        "mappingType" "writes"
        "toId" "DO-BEDROCK-REVIEW-FINDINGS"
      }
    }
    if_if_bedrock_webhook_api -> fu_fu_review_publication "PR Review Interaction Flow" {
      tags "Flow"
      properties {
        "flowId" "FLOW-BEDROCK-PR-REVIEW"
      }
    }
    person_act_security_engineer -> fu_fu_review_publication "Security Policy Tuning Flow" {
      tags "Flow"
      properties {
        "flowId" "FLOW-BEDROCK-POLICY-TUNING"
      }
    }
    deploymentEnvironment "external" {
      dn_dep_bedrock_github_boundary = deploymentNode "GitHub Integration Boundary" "github global app" "external" {
        tags "DeploymentTarget,external"
        properties {
          "account" "github"
          "cluster" "external"
          "environment" "external"
          "namespace" "app"
          "region" "global"
          "sourceId" "DEP-BEDROCK-GITHUB-BOUNDARY"
          "trustZone" "external"
        }
        softwareSystemInstance if_if_bedrock_review_publish_api {
          tags "Deployed"
          properties {
            "sourceId" "IF-BEDROCK-REVIEW-PUBLISH-API"
          }
        }
      }
    }
    deploymentEnvironment "prod" {
      dn_dep_bedrock_lambda_primary = deploymentNode "Primary Lambda Runtime" "app us-east-1 review" "lambda" {
        tags "DeploymentTarget,prod"
        properties {
          "account" "app"
          "cluster" "lambda"
          "environment" "prod"
          "namespace" "review"
          "region" "us-east-1"
          "sourceId" "DEP-BEDROCK-LAMBDA-PRIMARY"
          "trustZone" "app"
        }
        containerInstance fu_fu_github_webhook_ingress {
          tags "Deployed"
          properties {
            "sourceId" "FU-GITHUB-WEBHOOK-INGRESS"
          }
        }
        containerInstance fu_fu_review_orchestration {
          tags "Deployed"
          properties {
            "sourceId" "FU-REVIEW-ORCHESTRATION"
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

    systemContext sys_sample_bedrock_pr_review_model "context" {
      include *
      autolayout lr
    }

    container sys_sample_bedrock_pr_review_model "containers" {
      include *
      autolayout lr
    }
    dynamic sys_sample_bedrock_pr_review_model "dynamic_flow_bedrock_policy_tuning" "Policy rule update and dry-run validation path before policy promotion." {
      person_act_security_engineer -> fu_fu_review_publication "Policy rule update and dry-run validation path before policy promotion."
      autolayout lr
    }
    dynamic sys_sample_bedrock_pr_review_model "dynamic_flow_bedrock_pr_review" "Webhook-driven PR review path from ingress through context assembly, analysis, and publication." {
      if_if_bedrock_webhook_api -> fu_fu_review_publication "Webhook-driven PR review path from ingress through context assembly, analysis, and publication."
      autolayout lr
    }
    deployment sys_sample_bedrock_pr_review_model "external" "deployment_external" "Deployment view for environment: external" {
      include *
      autolayout lr
    }
    deployment sys_sample_bedrock_pr_review_model "prod" "deployment_prod" "Deployment view for environment: prod" {
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
