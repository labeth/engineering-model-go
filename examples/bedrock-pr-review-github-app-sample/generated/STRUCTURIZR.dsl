workspace "Bedrock Lambda GitHub PR Review App Architecture" "This document describes a GitHub App that reviews pull requests using AWS Bedrock and Lambda. The authored functional architecture remains stable while deployment/runtime/code evidence is inferred from infrastructure manifests, source files, and verification artifacts." {
  model {
    sys_sample_bedrock_pr_review_model = softwareSystem "Bedrock Lambda GitHub PR Review App Architecture" "This document describes a GitHub App that reviews pull requests using AWS Bedrock and Lambda. The authored functional architecture remains stable while deployment/runtime/code evidence is inferred from infrastructure manifests, source files, and verification artifacts." {
      fu_fu_github_webhook_ingress = container "GitHub Webhook Ingress" "Verifies webhook signatures, validates event type, and starts review workflow routing." "Functional Unit"
      fu_fu_lambda_runtime_operations = container "Lambda Runtime Operations" "Manages Lambda packaging/deployment lifecycle and operational health guardrails." "Functional Unit"
      fu_fu_policy_checks = container "Policy Checks" "Runs deterministic checks for secrets, risky IaC changes, and repository policy constraints." "Functional Unit"
      fu_fu_pr_context_assembly = container "PR Context Assembly" "Fetches changed files and diff metadata and prepares bounded review context payloads." "Functional Unit"
      fu_fu_review_orchestration = container "Review Orchestration" "Calls Bedrock models, merges AI and deterministic findings, and handles graceful degradation paths." "Functional Unit"
      fu_fu_review_publication = container "Review Publication" "Publishes check-run summaries and inline comments back to GitHub pull requests." "Functional Unit"
      fu_fu_secrets_configuration = container "Secrets and Configuration" "Manages GitHub App credentials, Bedrock config, and redaction-safe runtime configuration." "Functional Unit"
    }
    person_act_developer = person "Developer" "Opens pull requests and receives review feedback."
    person_act_platform_operator = person "Platform Operator" "Operates Lambda deployment and runtime controls."
    person_act_repository_maintainer = person "Repository Maintainer" "Owns merge decisions and repository-level quality gates."
    person_act_security_engineer = person "Security Engineer" "Curates policy checks and triages security findings."
    group_fg_platform_operations = softwareSystem "Group: Platform Operations" "Lambda deployment lifecycle, runtime operations, and secrets/configuration."
    group_fg_pr_integration = softwareSystem "Group: PR Integration" "Ingress and context assembly for GitHub pull request events."
    group_fg_review_intelligence = softwareSystem "Group: Review Intelligence" "AI and deterministic review analysis with result publication."
    ref_ref_aws_bedrock_runtime = softwareSystem "Ref: AWS Bedrock Runtime API" "runtime"
    ref_ref_aws_sdk_go = softwareSystem "Ref: AWS SDK for Go" "code"
    ref_ref_aws_secrets_manager = softwareSystem "Ref: AWS Secrets Manager" "runtime"
    ref_ref_github_app_api = softwareSystem "Ref: GitHub App API" "runtime"
    ref_ref_github_rest_sdk = softwareSystem "Ref: GitHub REST SDK" "code"
    ref_ref_policy_ruleset = softwareSystem "Ref: Review Policy Ruleset" "code"
    if_if_bedrock_review_publish_api = softwareSystem "Interface: Review Publish API" "https /github/check-runs"
    if_if_bedrock_webhook_api = softwareSystem "Interface: Webhook Intake API" "https /webhook/github"
    data_do_bedrock_pr_context = softwareSystem "Data: Pull Request Context" "schemas/pr-context.json"
    data_do_bedrock_review_findings = softwareSystem "Data: Review Findings" "schemas/review-findings.json"
    dep_dep_bedrock_github_boundary = softwareSystem "Deployment: GitHub Integration Boundary" "external external app global"
    dep_dep_bedrock_lambda_primary = softwareSystem "Deployment: Primary Lambda Runtime" "prod lambda review us-east-1"
    ctrl_ctrl_bedrock_secrets_rotation = softwareSystem "Control: Secret Rotation Enforcement" "Rotate app and model credentials according to policy."
    ctrl_ctrl_bedrock_webhook_signature = softwareSystem "Control: Webhook Signature Verification" "Enforce signature verification for all GitHub webhook events."
    av_av_pr_spam_abuse = softwareSystem "Attack: Pull Request Spam Abuse" "High-rate PR event flooding that can exhaust review capacity."
    av_av_prompt_injection_in_diff = softwareSystem "Attack: Prompt Injection in Diff" "Malicious prompt-like content embedded in pull request diffs."
    av_av_secret_leakage_in_review = softwareSystem "Attack: Secret Leakage in Review Output" "Generated findings accidentally exposing secrets/tokens in comments."
    av_av_spoofed_github_webhook = softwareSystem "Attack: Spoofed GitHub Webhook" "Forged webhook payload attempting unauthorized review execution."
    tb_tb_bedrock_aws_control = softwareSystem "Boundary: AWS Control Boundary" "Boundary between app workloads and privileged platform controls."
    tb_tb_bedrock_github_external = softwareSystem "Boundary: GitHub External Boundary" "Boundary between external GitHub traffic and internal review processing."
    ts_ts_bedrock_prompt_injection = softwareSystem "Threat: Prompt injection in diff influences review output integrity" "Malicious prompts in diff content attempt to coerce model output away from policy intent."
    ts_ts_bedrock_secret_leak_in_publication = softwareSystem "Threat: Sensitive tokens leak via review publication payload" "Review output includes sensitive content due to incomplete redaction before publication."
    ts_ts_bedrock_webhook_spoof = softwareSystem "Threat: Spoofed webhook bypasses ingress verification" "Forged pull request webhook attempts to trigger unauthorized review execution."
    fu_fu_github_webhook_ingress -> tb_tb_bedrock_github_external "bounded_by: External ingress trust boundary."
    fu_fu_lambda_runtime_operations -> tb_tb_bedrock_aws_control "bounded_by: Runtime platform control boundary."
    fu_fu_github_webhook_ingress -> if_if_bedrock_webhook_api "calls: Receives webhook payload through signed API endpoint."
    fu_fu_review_publication -> if_if_bedrock_review_publish_api "calls: Calls publication API for check-run output."
    group_fg_platform_operations -> fu_fu_lambda_runtime_operations "contains"
    group_fg_platform_operations -> fu_fu_secrets_configuration "contains"
    group_fg_pr_integration -> fu_fu_github_webhook_ingress "contains"
    group_fg_pr_integration -> fu_fu_pr_context_assembly "contains"
    group_fg_review_intelligence -> fu_fu_policy_checks "contains"
    group_fg_review_intelligence -> fu_fu_review_orchestration "contains"
    group_fg_review_intelligence -> fu_fu_review_publication "contains"
    fu_fu_github_webhook_ingress -> if_if_bedrock_webhook_api "contains"
    fu_fu_review_orchestration -> data_do_bedrock_pr_context "contains"
    fu_fu_review_publication -> data_do_bedrock_review_findings "contains"
    fu_fu_review_publication -> if_if_bedrock_review_publish_api "contains"
    fu_fu_github_webhook_ingress -> fu_fu_pr_context_assembly "depends_on: Forwards authenticated PR events for context assembly."
    fu_fu_lambda_runtime_operations -> fu_fu_secrets_configuration "depends_on: Requires secure runtime configuration and credential access."
    fu_fu_lambda_runtime_operations -> ref_ref_aws_secrets_manager "depends_on: Uses managed secret store for runtime credentials."
    fu_fu_policy_checks -> ref_ref_policy_ruleset "depends_on: Evaluates repository policy and risk rules."
    fu_fu_pr_context_assembly -> fu_fu_review_orchestration "depends_on: Sends normalized review context."
    fu_fu_pr_context_assembly -> ref_ref_github_app_api "depends_on: Fetches pull request files and metadata."
    fu_fu_review_orchestration -> fu_fu_policy_checks "depends_on: Merges deterministic policy findings with model output."
    fu_fu_review_orchestration -> fu_fu_review_publication "depends_on: Submits final findings for publication."
    fu_fu_review_orchestration -> fu_fu_secrets_configuration "depends_on: Retrieves model configuration and safety settings."
    fu_fu_review_orchestration -> ref_ref_aws_bedrock_runtime "depends_on: Requests model analysis for code review findings."
    fu_fu_review_publication -> fu_fu_secrets_configuration "depends_on: Retrieves app credentials and redaction rules."
    fu_fu_review_publication -> ref_ref_github_app_api "depends_on: Publishes check runs and inline review comments."
    fu_fu_github_webhook_ingress -> dep_dep_bedrock_lambda_primary "deployed_to: Webhook ingress executes in primary lambda runtime."
    fu_fu_review_orchestration -> dep_dep_bedrock_lambda_primary "deployed_to: Orchestration executes in primary lambda runtime."
    if_if_bedrock_review_publish_api -> dep_dep_bedrock_github_boundary "deployed_to: Publication API crosses GitHub integration boundary."
    person_act_developer -> fu_fu_github_webhook_ingress "interacts_with: Developer push/update activity triggers pull request webhook events."
    person_act_platform_operator -> fu_fu_lambda_runtime_operations "interacts_with: Operates deployments, scaling controls, and runtime observability."
    person_act_platform_operator -> fu_fu_secrets_configuration "interacts_with: Rotates credentials and manages secure configuration lifecycle."
    person_act_repository_maintainer -> fu_fu_review_publication "interacts_with: Reviews findings and gate decisions before merge."
    person_act_security_engineer -> fu_fu_policy_checks "interacts_with: Maintains and evolves deterministic security checks."
    av_av_secret_leakage_in_review -> ctrl_ctrl_bedrock_secrets_rotation "mitigated_by"
    av_av_spoofed_github_webhook -> ctrl_ctrl_bedrock_webhook_signature "mitigated_by"
    fu_fu_review_orchestration -> data_do_bedrock_pr_context "reads: Reads normalized context before model call."
    fu_fu_review_publication -> data_do_bedrock_review_findings "reads: Reads final findings for publication."
    av_av_pr_spam_abuse -> fu_fu_github_webhook_ingress "targets"
    av_av_pr_spam_abuse -> fu_fu_review_orchestration "targets"
    av_av_prompt_injection_in_diff -> fu_fu_pr_context_assembly "targets"
    av_av_prompt_injection_in_diff -> fu_fu_review_orchestration "targets"
    av_av_secret_leakage_in_review -> fu_fu_review_publication "targets"
    av_av_secret_leakage_in_review -> fu_fu_secrets_configuration "targets"
    av_av_spoofed_github_webhook -> fu_fu_github_webhook_ingress "targets"
    fu_fu_pr_context_assembly -> data_do_bedrock_pr_context "writes: Persists normalized context payload."
    fu_fu_review_orchestration -> data_do_bedrock_review_findings "writes: Persists merged deterministic and AI findings."
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
  }
}
