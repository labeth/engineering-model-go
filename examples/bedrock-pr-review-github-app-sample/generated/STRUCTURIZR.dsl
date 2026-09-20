workspace "Bedrock Lambda GitHub PR Review App Architecture" "This document describes a GitHub App that reviews pull requests using AWS Bedrock and Lambda. The authored functional architecture remains stable while deployment/runtime/code evidence is inferred from infrastructure manifests, source files, and verification artifacts." {
  model {
    sys_sample_bedrock_pr_review_model = softwareSystem "Bedrock Lambda GitHub PR Review App Architecture" "This document describes a GitHub App that reviews pull requests using AWS Bedrock and Lambda. The authored functional architecture remains stable while deployment/runtime/code evidence is inferred from infrastructure manifests, source files, and verification artifacts." {
      group "PR Integration" {
        fu_fu_github_webhook_ingress = container "GitHub Webhook Ingress" "Verifies webhook signatures, validates event type, and starts review workflow routing." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_pr_context_assembly = container "PR Context Assembly" "Fetches changed files and diff metadata and prepares bounded review context payloads." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Platform Operations" {
        fu_fu_lambda_runtime_operations = container "Lambda Runtime Operations" "Manages Lambda packaging/deployment lifecycle and operational health guardrails." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_secrets_configuration = container "Secrets and Configuration" "Manages GitHub App credentials, Bedrock config, and redaction-safe runtime configuration." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Review Intelligence" {
        fu_fu_policy_checks = container "Policy Checks" "Runs deterministic checks for secrets, risky IaC changes, and repository policy constraints." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_review_orchestration = container "Review Orchestration" "Calls Bedrock models, merges AI and deterministic findings, and handles graceful degradation paths." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_review_publication = container "Review Publication" "Publishes check-run summaries and inline comments back to GitHub pull requests." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_developer = person "Developer" "Opens pull requests and receives review feedback." {
      tags "Actor"
    }
    person_act_platform_operator = person "Platform Operator" "Operates Lambda deployment and runtime controls." {
      tags "Actor"
    }
    person_act_repository_maintainer = person "Repository Maintainer" "Owns merge decisions and repository-level quality gates." {
      tags "Actor"
    }
    person_act_security_engineer = person "Security Engineer" "Curates policy checks and triages security findings." {
      tags "Actor"
    }
    group_fg_platform_operations = softwareSystem "Platform Operations" "Lambda deployment lifecycle, runtime operations, and secrets/configuration." {
      tags "FunctionalGroup"
    }
    group_fg_pr_integration = softwareSystem "PR Integration" "Ingress and context assembly for GitHub pull request events." {
      tags "FunctionalGroup"
    }
    group_fg_review_intelligence = softwareSystem "Review Intelligence" "AI and deterministic review analysis with result publication." {
      tags "FunctionalGroup"
    }
    ref_ref_aws_bedrock_runtime = softwareSystem "AWS Bedrock Runtime API" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_aws_sdk_go = softwareSystem "AWS SDK for Go" "code" {
      tags "ReferencedElement,third_party_library"
    }
    ref_ref_aws_secrets_manager = softwareSystem "AWS Secrets Manager" "runtime" {
      tags "ReferencedElement,platform_service"
    }
    ref_ref_github_app_api = softwareSystem "GitHub App API" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_github_rest_sdk = softwareSystem "GitHub REST SDK" "code" {
      tags "ReferencedElement,third_party_library"
    }
    ref_ref_policy_ruleset = softwareSystem "Review Policy Ruleset" "code" {
      tags "ReferencedElement,policy_bundle"
    }
    if_if_bedrock_review_publish_api = softwareSystem "Review Publish API" "https /github/check-runs" {
      tags "Interface"
    }
    if_if_bedrock_webhook_api = softwareSystem "Webhook Intake API" "https /webhook/github" {
      tags "Interface"
    }
    data_do_bedrock_pr_context = softwareSystem "Pull Request Context" "schemas/pr-context.json" {
      tags "DataObject,internal"
    }
    data_do_bedrock_review_findings = softwareSystem "Review Findings" "schemas/review-findings.json" {
      tags "DataObject,internal"
    }
    ctrl_ctrl_bedrock_secrets_rotation = softwareSystem "Secret Rotation Enforcement" "Rotate app and model credentials according to policy." {
      tags "Control,credential-management"
    }
    ctrl_ctrl_bedrock_webhook_signature = softwareSystem "Webhook Signature Verification" "Enforce signature verification for all GitHub webhook events." {
      tags "Control,input-integrity"
    }
    av_av_pr_spam_abuse = softwareSystem "Pull Request Spam Abuse" "High-rate PR event flooding that can exhaust review capacity." {
      tags "AttackVector"
    }
    av_av_prompt_injection_in_diff = softwareSystem "Prompt Injection in Diff" "Malicious prompt-like content embedded in pull request diffs." {
      tags "AttackVector"
    }
    av_av_secret_leakage_in_review = softwareSystem "Secret Leakage in Review Output" "Generated findings accidentally exposing secrets/tokens in comments." {
      tags "AttackVector"
    }
    av_av_spoofed_github_webhook = softwareSystem "Spoofed GitHub Webhook" "Forged webhook payload attempting unauthorized review execution." {
      tags "AttackVector"
    }
    tb_tb_bedrock_aws_control = softwareSystem "AWS Control Boundary" "Boundary between app workloads and privileged platform controls." {
      tags "TrustBoundary,control-plane"
    }
    tb_tb_bedrock_github_external = softwareSystem "GitHub External Boundary" "Boundary between external GitHub traffic and internal review processing." {
      tags "TrustBoundary,network"
    }
    ts_ts_bedrock_prompt_injection = softwareSystem "Prompt injection in diff influences review output integrity" "Malicious prompts in diff content attempt to coerce model output away from policy intent." {
      tags "ThreatScenario,tampering,mitigating"
    }
    ts_ts_bedrock_secret_leak_in_publication = softwareSystem "Sensitive tokens leak via review publication payload" "Review output includes sensitive content due to incomplete redaction before publication." {
      tags "ThreatScenario,information-disclosure,mitigating"
    }
    ts_ts_bedrock_webhook_spoof = softwareSystem "Spoofed webhook bypasses ingress verification" "Forged pull request webhook attempts to trigger unauthorized review execution." {
      tags "ThreatScenario,spoofing,mitigating"
    }
    fu_fu_github_webhook_ingress -> tb_tb_bedrock_github_external "External ingress trust boundary." "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_lambda_runtime_operations -> tb_tb_bedrock_aws_control "Runtime platform control boundary." "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_github_webhook_ingress -> if_if_bedrock_webhook_api "Receives webhook payload through signed API endpoint." "Model relationship: calls" {
      tags "Mapping,calls"
    }
    fu_fu_review_publication -> if_if_bedrock_review_publish_api "Calls publication API for check-run output." "Model relationship: calls" {
      tags "Mapping,calls"
    }
    group_fg_platform_operations -> fu_fu_lambda_runtime_operations "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_platform_operations -> fu_fu_secrets_configuration "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_pr_integration -> fu_fu_github_webhook_ingress "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_pr_integration -> fu_fu_pr_context_assembly "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_review_intelligence -> fu_fu_policy_checks "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_review_intelligence -> fu_fu_review_orchestration "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_review_intelligence -> fu_fu_review_publication "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_github_webhook_ingress -> if_if_bedrock_webhook_api "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_review_orchestration -> data_do_bedrock_pr_context "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_review_publication -> data_do_bedrock_review_findings "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_review_publication -> if_if_bedrock_review_publish_api "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_github_webhook_ingress -> fu_fu_pr_context_assembly "Forwards authenticated PR events for context assembly." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_lambda_runtime_operations -> fu_fu_secrets_configuration "Requires secure runtime configuration and credential access." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_lambda_runtime_operations -> ref_ref_aws_secrets_manager "Uses managed secret store for runtime credentials." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_policy_checks -> ref_ref_policy_ruleset "Evaluates repository policy and risk rules." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_pr_context_assembly -> fu_fu_review_orchestration "Sends normalized review context." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_pr_context_assembly -> ref_ref_github_app_api "Fetches pull request files and metadata." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_pr_context_assembly -> ref_ref_github_rest_sdk "Uses the GitHub REST SDK to fetch pull request files and metadata." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_review_orchestration -> fu_fu_policy_checks "Merges deterministic policy findings with model output." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_review_orchestration -> fu_fu_review_publication "Submits final findings for publication." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_review_orchestration -> fu_fu_secrets_configuration "Retrieves model configuration and safety settings." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_review_orchestration -> ref_ref_aws_bedrock_runtime "Requests model analysis for code review findings." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_review_publication -> fu_fu_secrets_configuration "Retrieves app credentials and redaction rules." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_review_publication -> ref_ref_github_app_api "Publishes check runs and inline review comments." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_secrets_configuration -> ref_ref_aws_sdk_go "Uses AWS SDK for Go clients to load managed secret configuration." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    person_act_developer -> fu_fu_github_webhook_ingress "Developer push/update activity triggers pull request webhook events." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_platform_operator -> fu_fu_lambda_runtime_operations "Operates deployments, scaling controls, and runtime observability." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_platform_operator -> fu_fu_secrets_configuration "Rotates credentials and manages secure configuration lifecycle." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_repository_maintainer -> fu_fu_review_publication "Reviews findings and gate decisions before merge." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_security_engineer -> fu_fu_policy_checks "Maintains and evolves deterministic security checks." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    av_av_secret_leakage_in_review -> ctrl_ctrl_bedrock_secrets_rotation "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    av_av_spoofed_github_webhook -> ctrl_ctrl_bedrock_webhook_signature "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    fu_fu_review_orchestration -> data_do_bedrock_pr_context "Reads normalized context before model call." "Model relationship: reads" {
      tags "Mapping,reads"
    }
    fu_fu_review_publication -> data_do_bedrock_review_findings "Reads final findings for publication." "Model relationship: reads" {
      tags "Mapping,reads"
    }
    av_av_pr_spam_abuse -> fu_fu_github_webhook_ingress "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_pr_spam_abuse -> fu_fu_review_orchestration "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_prompt_injection_in_diff -> fu_fu_pr_context_assembly "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_prompt_injection_in_diff -> fu_fu_review_orchestration "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_secret_leakage_in_review -> fu_fu_review_publication "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_secret_leakage_in_review -> fu_fu_secrets_configuration "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_spoofed_github_webhook -> fu_fu_github_webhook_ingress "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    fu_fu_pr_context_assembly -> data_do_bedrock_pr_context "Persists normalized context payload." "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_review_orchestration -> data_do_bedrock_review_findings "Persists merged deterministic and AI findings." "Model relationship: writes" {
      tags "Mapping,writes"
    }
    if_if_bedrock_webhook_api -> fu_fu_review_publication "PR Review Interaction Flow" "https / webhook" {
      tags "Flow"
    }
    person_act_security_engineer -> fu_fu_review_publication "Security Policy Tuning Flow" "internal-api / policy-validation" {
      tags "Flow"
    }
    deploymentEnvironment "external" {
      dn_dep_bedrock_github_boundary = deploymentNode "GitHub Integration Boundary" "github global app" "external" {
        tags "DeploymentTarget,external"
        softwareSystemInstance if_if_bedrock_review_publish_api {
          tags "Deployed"
        }
      }
    }
    deploymentEnvironment "prod" {
      dn_dep_bedrock_lambda_primary = deploymentNode "Primary Lambda Runtime" "app us-east-1 review" "lambda" {
        tags "DeploymentTarget,prod"
        containerInstance fu_fu_github_webhook_ingress {
          tags "Deployed"
        }
        containerInstance fu_fu_review_orchestration {
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

    systemContext sys_sample_bedrock_pr_review_model "context" {
      include *
      autolayout lr
    }

    container sys_sample_bedrock_pr_review_model "containers" {
      include *
      autolayout lr
    }
    dynamic sys_sample_bedrock_pr_review_model "dynamic_flow_bedrock_policy_tuning" "Policy rule update and dry-run validation path before policy promotion." {
      person_act_security_engineer -> fu_fu_review_publication "Security Policy Tuning Flow" "internal-api / policy-validation"
      autolayout lr
    }
    dynamic sys_sample_bedrock_pr_review_model "dynamic_flow_bedrock_pr_review" "Webhook-driven PR review path from ingress through context assembly, analysis, and publication." {
      if_if_bedrock_webhook_api -> fu_fu_review_publication "PR Review Interaction Flow" "https / webhook"
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

  configuration {
    scope softwaresystem
  }

}
