workspace "Connected Coffee Fleet OTA + Cloud Architecture" "This document models connected coffee machines that send telemetry to a central cloud site, receive OTA firmware updates, and produce operational and audit logging evidence. Authored functional architecture remains stable while runtime, code, and verification layers are inferred from infrastructure, source, and test artifacts." {
  model {
    sys_sample_coffee_fleet_ota_cloud_model = softwareSystem "Connected Coffee Fleet OTA + Cloud Architecture" "This document models connected coffee machines that send telemetry to a central cloud site, receive OTA firmware updates, and produce operational and audit logging evidence. Authored functional architecture remains stable while runtime, code, and verification layers are inferred from infrastructure, source, and test artifacts." {
      fu_fu_cloud_runtime_operations = container "Cloud Runtime Operations" "Owns runtime deployment lifecycle, scaling controls, and runtime operational safety." "Functional Unit"
      fu_fu_device_identity_secrets = container "Device Identity and Secrets" "Manages device credentials, cloud tokens, and secure configuration used by edge/cloud units." "Functional Unit"
      fu_fu_fleet_ingestion_api = container "Fleet Ingestion API" "Receives telemetry and machine status, applies validation, and emits ingestion events." "Functional Unit"
      fu_fu_fleet_observability_reporting = container "Fleet Observability Reporting" "Aggregates telemetry and OTA outcomes into dashboards, alerts, and audit records." "Functional Unit"
      fu_fu_machine_telemetry_collection = container "Machine Telemetry Collection" "Collects brew/device telemetry and transmits cloud-bound telemetry payloads." "Functional Unit"
      fu_fu_ota_update_agent = container "OTA Update Agent" "Downloads signed firmware bundles, validates integrity, applies updates, and reports status." "Functional Unit"
      fu_fu_update_campaign_orchestration = container "Update Campaign Orchestration" "Plans OTA rollout cohorts, drives update commands, and handles rollback flow decisions." "Functional Unit"
    }
    person_act_barista = person "Barista" "Operates coffee machines and observes local device status."
    person_act_fleet_operator = person "Fleet Operator" "Manages OTA campaigns and fleet-wide operational decisions."
    person_act_platform_operator = person "Platform Operator" "Operates cloud runtime, deployment, and reliability controls."
    person_act_security_analyst = person "Security Analyst" "Reviews firmware integrity, identity controls, and audit logging."
    group_fg_cloud_control = softwareSystem "Group: Cloud Control" "Central cloud ingestion, campaign control, and fleet reporting."
    group_fg_machine_edge = softwareSystem "Group: Machine Edge" "Device-local telemetry and OTA execution responsibilities."
    group_fg_platform_operations = softwareSystem "Group: Platform Operations" "Cloud runtime operations plus identity and secret-management controls."
    ref_ref_cloud_logging_service = softwareSystem "Ref: Cloud Logging Service" "runtime"
    ref_ref_cloud_runtime_sdk = softwareSystem "Ref: Cloud Runtime SDK" "code"
    ref_ref_firmware_bundle_store = softwareSystem "Ref: Firmware Bundle Store" "runtime"
    ref_ref_firmware_signer_service = softwareSystem "Ref: Firmware Signer Service" "runtime"
    ref_ref_iot_ingest_endpoint = softwareSystem "Ref: IoT Ingest Endpoint" "runtime"
    ref_ref_mqtt_device_sdk = softwareSystem "Ref: MQTT Device SDK" "code"
    if_if_coffee_ota_command = softwareSystem "Interface: OTA Command Interface" "mqtt mqtt://devices/ota/commands"
    if_if_coffee_telemetry_ingest = softwareSystem "Interface: Telemetry Ingestion Interface" "mqtt mqtt://ingest/topics/telemetry"
    data_do_coffee_ota_plan = softwareSystem "Data: OTA Rollout Plan" "schemas/ota-plan.json"
    data_do_coffee_telemetry_event = softwareSystem "Data: Telemetry Event" "schemas/telemetry-event.json"
    dep_dep_coffee_cloud_prod = softwareSystem "Deployment: Coffee Cloud Production" "prod coffee-cloud coffee us-east-1"
    dep_dep_coffee_edge_fleet = softwareSystem "Deployment: Coffee Edge Fleet" "edge edge-fleet devices global"
    ctrl_ctrl_coffee_device_identity = softwareSystem "Control: Device Identity Validation" "Require authenticated device identity for telemetry and OTA command channels."
    ctrl_ctrl_coffee_firmware_signature = softwareSystem "Control: Firmware Signature Enforcement" "Require signed firmware bundles before applying OTA updates."
    av_av_log_tampering = softwareSystem "Attack: Log Tampering" "Attempts to suppress or alter audit/operational logs."
    av_av_malicious_firmware_bundle = softwareSystem "Attack: Malicious Firmware Bundle" "Tampered OTA payload targeting update agent execution."
    av_av_spoofed_device_identity = softwareSystem "Attack: Spoofed Device Identity" "Unauthorized identity attempting telemetry/command access."
    av_av_telemetry_replay_abuse = softwareSystem "Attack: Telemetry Replay Abuse" "Replayed telemetry intended to overload ingestion and corrupt insights."
    tb_tb_coffee_cloud_control = softwareSystem "Boundary: Cloud Control Boundary" "Boundary between cloud app workloads and privileged platform controls."
    tb_tb_coffee_edge_device = softwareSystem "Boundary: Edge Device Boundary" "Boundary between managed edge devices and cloud control plane."
    ts_ts_coffee_log_tampering = softwareSystem "Threat: Audit log tampering suppresses OTA incident visibility" "Adversary attempts to alter or suppress security-relevant logs to hide malicious behavior."
    ts_ts_coffee_malicious_ota_bundle = softwareSystem "Threat: Tampered firmware bundle executes on edge fleet" "Attacker attempts to deliver malicious firmware by replacing or tampering rollout artifact."
    ts_ts_coffee_ota_command_spoof = softwareSystem "Threat: Spoofed OTA command triggers unauthorized rollout" "Unauthorized sender attempts to push OTA command messages to device fleet topic."
    ts_ts_coffee_telemetry_replay = softwareSystem "Threat: Telemetry replay inflates ingest and distorts reporting" "Replayed telemetry bursts attempt to pollute fleet metrics and trigger false alerts."
    fu_fu_cloud_runtime_operations -> tb_tb_coffee_cloud_control "bounded_by: Cloud control-plane trust separation."
    fu_fu_ota_update_agent -> tb_tb_coffee_edge_device "bounded_by: OTA execution constrained by edge trust boundary."
    fu_fu_machine_telemetry_collection -> if_if_coffee_telemetry_ingest "calls: Sends telemetry payloads via telemetry ingest interface."
    fu_fu_update_campaign_orchestration -> if_if_coffee_ota_command "calls: Publishes OTA rollout commands to device fleet."
    group_fg_cloud_control -> fu_fu_fleet_ingestion_api "contains"
    group_fg_cloud_control -> fu_fu_fleet_observability_reporting "contains"
    group_fg_cloud_control -> fu_fu_update_campaign_orchestration "contains"
    group_fg_machine_edge -> fu_fu_machine_telemetry_collection "contains"
    group_fg_machine_edge -> fu_fu_ota_update_agent "contains"
    group_fg_platform_operations -> fu_fu_cloud_runtime_operations "contains"
    group_fg_platform_operations -> fu_fu_device_identity_secrets "contains"
    fu_fu_fleet_ingestion_api -> if_if_coffee_telemetry_ingest "contains"
    fu_fu_machine_telemetry_collection -> data_do_coffee_telemetry_event "contains"
    fu_fu_update_campaign_orchestration -> data_do_coffee_ota_plan "contains"
    fu_fu_update_campaign_orchestration -> if_if_coffee_ota_command "contains"
    fu_fu_cloud_runtime_operations -> fu_fu_device_identity_secrets "depends_on: Requires runtime credentials and encrypted config."
    fu_fu_cloud_runtime_operations -> ref_ref_cloud_logging_service "depends_on: Emits platform operation and deployment events."
    fu_fu_fleet_ingestion_api -> fu_fu_device_identity_secrets "depends_on: Validates device identity and token claims."
    fu_fu_fleet_ingestion_api -> fu_fu_fleet_observability_reporting "depends_on: Forwards validated telemetry and machine status events."
    fu_fu_fleet_ingestion_api -> ref_ref_cloud_runtime_sdk "depends_on: Uses cloud SDK clients for ingest and routing operations."
    fu_fu_fleet_observability_reporting -> fu_fu_device_identity_secrets "depends_on: Uses controlled credentials for audit-log writes."
    fu_fu_fleet_observability_reporting -> ref_ref_cloud_logging_service "depends_on: Persists telemetry, OTA outcomes, and audit records."
    fu_fu_machine_telemetry_collection -> fu_fu_device_identity_secrets "depends_on: Uses machine identity material for authenticated publish."
    fu_fu_machine_telemetry_collection -> fu_fu_fleet_ingestion_api "depends_on: Sends normalized telemetry records to ingestion API."
    fu_fu_machine_telemetry_collection -> ref_ref_iot_ingest_endpoint "depends_on: Publishes telemetry payloads to cloud ingestion."
    fu_fu_machine_telemetry_collection -> ref_ref_mqtt_device_sdk "depends_on: Uses MQTT client for edge publish and command channels."
    fu_fu_ota_update_agent -> fu_fu_device_identity_secrets "depends_on: Uses trust anchors for firmware and endpoint authentication."
    fu_fu_ota_update_agent -> fu_fu_update_campaign_orchestration "depends_on: Receives OTA rollout commands and policy constraints."
    fu_fu_ota_update_agent -> ref_ref_firmware_bundle_store "depends_on: Downloads signed firmware bundle."
    fu_fu_update_campaign_orchestration -> fu_fu_device_identity_secrets "depends_on: Uses trusted control-plane credentials."
    fu_fu_update_campaign_orchestration -> fu_fu_fleet_observability_reporting "depends_on: Emits OTA rollout and rollback events for fleet visibility."
    fu_fu_update_campaign_orchestration -> ref_ref_firmware_signer_service "depends_on: Verifies firmware artifact signature metadata."
    fu_fu_fleet_ingestion_api -> dep_dep_coffee_cloud_prod "deployed_to: Ingestion API runs in cloud production target."
    fu_fu_ota_update_agent -> dep_dep_coffee_edge_fleet "deployed_to: OTA agent executes on managed edge fleet."
    fu_fu_update_campaign_orchestration -> dep_dep_coffee_cloud_prod "deployed_to: Orchestration service runs in cloud production target."
    person_act_barista -> fu_fu_machine_telemetry_collection "interacts_with: Daily machine operation produces brew and status telemetry."
    person_act_fleet_operator -> fu_fu_update_campaign_orchestration "interacts_with: Defines OTA target cohorts and rollout windows."
    person_act_platform_operator -> fu_fu_cloud_runtime_operations "interacts_with: Operates deployment and runtime controls."
    person_act_security_analyst -> fu_fu_device_identity_secrets "interacts_with: Audits identity trust and secret rotation controls."
    av_av_malicious_firmware_bundle -> ctrl_ctrl_coffee_firmware_signature "mitigated_by"
    av_av_spoofed_device_identity -> ctrl_ctrl_coffee_device_identity "mitigated_by"
    fu_fu_ota_update_agent -> data_do_coffee_ota_plan "reads: Reads rollout plan before applying firmware."
    av_av_log_tampering -> fu_fu_device_identity_secrets "targets"
    av_av_log_tampering -> fu_fu_fleet_observability_reporting "targets"
    av_av_malicious_firmware_bundle -> fu_fu_ota_update_agent "targets"
    av_av_malicious_firmware_bundle -> fu_fu_update_campaign_orchestration "targets"
    av_av_spoofed_device_identity -> fu_fu_fleet_ingestion_api "targets"
    av_av_spoofed_device_identity -> fu_fu_machine_telemetry_collection "targets"
    av_av_telemetry_replay_abuse -> fu_fu_fleet_ingestion_api "targets"
    av_av_telemetry_replay_abuse -> fu_fu_fleet_observability_reporting "targets"
    fu_fu_machine_telemetry_collection -> data_do_coffee_telemetry_event "writes: Writes normalized telemetry event payload."
    fu_fu_update_campaign_orchestration -> data_do_coffee_ota_plan "writes: Persists rollout plan and cohort metadata."
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_sample_coffee_fleet_ota_cloud_model "context" {
      include *
      autolayout lr
    }

    container sys_sample_coffee_fleet_ota_cloud_model "containers" {
      include *
      autolayout lr
    }
  }
}
