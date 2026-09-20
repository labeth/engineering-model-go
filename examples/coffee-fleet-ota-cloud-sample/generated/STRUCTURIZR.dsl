workspace "Connected Coffee Fleet OTA + Cloud Architecture" "This document models connected coffee machines that send telemetry to a central cloud site, receive OTA firmware updates, and produce operational and audit logging evidence. Authored functional architecture remains stable while runtime, code, and verification layers are inferred from infrastructure, source, and test artifacts." {
  model {
    sys_sample_coffee_fleet_ota_cloud_model = softwareSystem "Connected Coffee Fleet OTA + Cloud Architecture" "This document models connected coffee machines that send telemetry to a central cloud site, receive OTA firmware updates, and produce operational and audit logging evidence. Authored functional architecture remains stable while runtime, code, and verification layers are inferred from infrastructure, source, and test artifacts." {
      group "Cloud" {
        fu_cloud_api_fu_cloud_campaign = container "cloud-api::FU-CLOUD-CAMPAIGN — Cloud Campaign" "Plans and dispatches OTA campaigns." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_cloud_api_fu_cloud_ingest = container "cloud-api::FU-CLOUD-INGEST — Cloud Ingest" "Ingests and persists fleet telemetry." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Cloud Control" {
        fu_fu_fleet_ingestion_api = container "Fleet Ingestion API" "Receives telemetry and machine status, applies validation, and emits ingestion events." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_fleet_observability_reporting = container "Fleet Observability Reporting" "Aggregates telemetry and OTA outcomes into dashboards, alerts, and audit records." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_update_campaign_orchestration = container "Update Campaign Orchestration" "Plans OTA rollout cohorts, drives update commands, and handles rollback flow decisions." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Machine Edge" {
        fu_fu_machine_telemetry_collection = container "Machine Telemetry Collection" "Collects brew/device telemetry and transmits cloud-bound telemetry payloads." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_ota_update_agent = container "OTA Update Agent" "Downloads signed firmware bundles, validates integrity, applies updates, and reports status." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "OTA" {
        fu_ota_agent_fu_ota_apply = container "ota-agent::FU-OTA-APPLY — OTA Apply" "Applies verified firmware and rolls back on failure." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_ota_agent_fu_ota_verify = container "ota-agent::FU-OTA-VERIFY — OTA Verification" "Verifies firmware signature and update eligibility." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Platform Operations" {
        fu_fu_cloud_runtime_operations = container "Cloud Runtime Operations" "Owns runtime deployment lifecycle, scaling controls, and runtime operational safety." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_device_identity_secrets = container "Device Identity and Secrets" "Manages device credentials, cloud tokens, and secure configuration used by edge/cloud units." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Telemetry" {
        fu_telemetry_fu_telem_report = container "telemetry::FU-TELEM-REPORT — Telemetry Reporting" "Encodes, signs, and reports telemetry to the fleet ingestion endpoint." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_telemetry_fu_telem_sample = container "telemetry::FU-TELEM-SAMPLE — Telemetry Sampling" "Samples machine sensors at the configured cadence." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_barista = person "Barista" "Operates coffee machines and observes local device status." {
      tags "Actor"
    }
    person_act_fleet_operator = person "Fleet Operator" "Manages OTA campaigns and fleet-wide operational decisions." {
      tags "Actor"
    }
    person_act_platform_operator = person "Platform Operator" "Operates cloud runtime, deployment, and reliability controls." {
      tags "Actor"
    }
    person_act_security_analyst = person "Security Analyst" "Reviews firmware integrity, identity controls, and audit logging." {
      tags "Actor"
    }
    group_fg_cloud_control = softwareSystem "Cloud Control" "Central cloud ingestion, campaign control, and fleet reporting." {
      tags "FunctionalGroup"
    }
    group_fg_machine_edge = softwareSystem "Machine Edge" "Device-local telemetry and OTA execution responsibilities." {
      tags "FunctionalGroup"
    }
    group_fg_platform_operations = softwareSystem "Platform Operations" "Cloud runtime operations plus identity and secret-management controls." {
      tags "FunctionalGroup"
    }
    group_cloud_api_fg_cloud = softwareSystem "cloud-api::FG-CLOUD — Cloud" "Cloud ingestion and campaign control." {
      tags "FunctionalGroup"
    }
    group_ota_agent_fg_ota = softwareSystem "ota-agent::FG-OTA — OTA" "On-machine OTA update verification and application." {
      tags "FunctionalGroup"
    }
    group_telemetry_fg_telemetry = softwareSystem "telemetry::FG-TELEMETRY — Telemetry" "Telemetry collection and reporting." {
      tags "FunctionalGroup"
    }
    ref_ref_cloud_logging_service = softwareSystem "Cloud Logging Service" "runtime" {
      tags "ReferencedElement,platform_service"
    }
    ref_ref_cloud_runtime_sdk = softwareSystem "Cloud Runtime SDK" "code" {
      tags "ReferencedElement,third_party_library"
    }
    ref_ref_firmware_bundle_store = softwareSystem "Firmware Bundle Store" "runtime" {
      tags "ReferencedElement,object_store"
    }
    ref_ref_firmware_signer_service = softwareSystem "Firmware Signer Service" "runtime" {
      tags "ReferencedElement,signing_service"
    }
    ref_ref_iot_ingest_endpoint = softwareSystem "IoT Ingest Endpoint" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_mqtt_device_sdk = softwareSystem "MQTT Device SDK" "code" {
      tags "ReferencedElement,third_party_library"
    }
    if_if_coffee_ota_command = softwareSystem "OTA Command Interface" "mqtt mqtt://devices/ota/commands" {
      tags "Interface"
    }
    if_if_coffee_telemetry_ingest = softwareSystem "Telemetry Ingestion Interface" "mqtt mqtt://ingest/topics/telemetry" {
      tags "Interface"
    }
    if_cloud_api_if_cloud_ingest = softwareSystem "cloud-api::IF-CLOUD-INGEST — Cloud Ingest Interface" "https /ingest" {
      tags "Interface"
    }
    if_ota_agent_if_ota_apply = softwareSystem "ota-agent::IF-OTA-APPLY — OTA Apply Interface" "internal /ota/apply" {
      tags "Interface"
    }
    if_telemetry_if_telem_report = softwareSystem "telemetry::IF-TELEM-REPORT — Telemetry Report Interface" "https /telemetry" {
      tags "Interface"
    }
    data_do_coffee_ota_plan = softwareSystem "OTA Rollout Plan" "schemas/ota-plan.json" {
      tags "DataObject,internal"
    }
    data_do_coffee_telemetry_event = softwareSystem "Telemetry Event" "schemas/telemetry-event.json" {
      tags "DataObject,internal"
    }
    data_cloud_api_do_fleet_metric = softwareSystem "cloud-api::DO-FLEET-METRIC — Fleet Metric" "" {
      tags "DataObject"
    }
    data_ota_agent_do_firmware_bundle = softwareSystem "ota-agent::DO-FIRMWARE-BUNDLE — Firmware Bundle" "" {
      tags "DataObject"
    }
    data_telemetry_do_telem_record = softwareSystem "telemetry::DO-TELEM-RECORD — Telemetry Record" "" {
      tags "DataObject"
    }
    ctrl_ctrl_coffee_device_identity = softwareSystem "Device Identity Validation" "Require authenticated device identity for telemetry and OTA command channels." {
      tags "Control,identity-access"
    }
    ctrl_ctrl_coffee_firmware_signature = softwareSystem "Firmware Signature Enforcement" "Require signed firmware bundles before applying OTA updates." {
      tags "Control,integrity"
    }
    av_av_log_tampering = softwareSystem "Log Tampering" "Attempts to suppress or alter audit/operational logs." {
      tags "AttackVector"
    }
    av_av_malicious_firmware_bundle = softwareSystem "Malicious Firmware Bundle" "Tampered OTA payload targeting update agent execution." {
      tags "AttackVector"
    }
    av_av_spoofed_device_identity = softwareSystem "Spoofed Device Identity" "Unauthorized identity attempting telemetry/command access." {
      tags "AttackVector"
    }
    av_av_telemetry_replay_abuse = softwareSystem "Telemetry Replay Abuse" "Replayed telemetry intended to overload ingestion and corrupt insights." {
      tags "AttackVector"
    }
    tb_tb_coffee_cloud_control = softwareSystem "Cloud Control Boundary" "Boundary between cloud app workloads and privileged platform controls." {
      tags "TrustBoundary,control-plane"
    }
    tb_tb_coffee_edge_device = softwareSystem "Edge Device Boundary" "Boundary between managed edge devices and cloud control plane." {
      tags "TrustBoundary,device-network"
    }
    ts_ts_coffee_log_tampering = softwareSystem "Audit log tampering suppresses OTA incident visibility" "Adversary attempts to alter or suppress security-relevant logs to hide malicious behavior." {
      tags "ThreatScenario,repudiation,identified"
    }
    ts_ts_coffee_malicious_ota_bundle = softwareSystem "Tampered firmware bundle executes on edge fleet" "Attacker attempts to deliver malicious firmware by replacing or tampering rollout artifact." {
      tags "ThreatScenario,tampering,mitigating"
    }
    ts_ts_coffee_ota_command_spoof = softwareSystem "Spoofed OTA command triggers unauthorized rollout" "Unauthorized sender attempts to push OTA command messages to device fleet topic." {
      tags "ThreatScenario,spoofing,mitigating"
    }
    ts_ts_coffee_telemetry_replay = softwareSystem "Telemetry replay inflates ingest and distorts reporting" "Replayed telemetry bursts attempt to pollute fleet metrics and trigger false alerts." {
      tags "ThreatScenario,replay,mitigating"
    }
    fu_fu_cloud_runtime_operations -> tb_tb_coffee_cloud_control "Cloud control-plane trust separation." "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_ota_update_agent -> tb_tb_coffee_edge_device "OTA execution constrained by edge trust boundary." "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_machine_telemetry_collection -> if_if_coffee_telemetry_ingest "Sends telemetry payloads via telemetry ingest interface." "Model relationship: calls" {
      tags "Mapping,calls"
    }
    fu_fu_update_campaign_orchestration -> if_if_coffee_ota_command "Publishes OTA rollout commands to device fleet." "Model relationship: calls" {
      tags "Mapping,calls"
    }
    group_fg_cloud_control -> fu_fu_fleet_ingestion_api "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_cloud_control -> fu_fu_fleet_observability_reporting "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_cloud_control -> fu_fu_update_campaign_orchestration "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_machine_edge -> fu_fu_machine_telemetry_collection "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_machine_edge -> fu_fu_ota_update_agent "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_platform_operations -> fu_fu_cloud_runtime_operations "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_platform_operations -> fu_fu_device_identity_secrets "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_fleet_ingestion_api -> if_if_coffee_telemetry_ingest "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_machine_telemetry_collection -> data_do_coffee_telemetry_event "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_update_campaign_orchestration -> data_do_coffee_ota_plan "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_update_campaign_orchestration -> if_if_coffee_ota_command "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_cloud_runtime_operations -> fu_fu_device_identity_secrets "Requires runtime credentials and encrypted config." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_cloud_runtime_operations -> ref_ref_cloud_logging_service "Emits platform operation and deployment events." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_fleet_ingestion_api -> fu_fu_device_identity_secrets "Validates device identity and token claims." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_fleet_ingestion_api -> fu_fu_fleet_observability_reporting "Forwards validated telemetry and machine status events." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_fleet_ingestion_api -> ref_ref_cloud_runtime_sdk "Uses cloud SDK clients for ingest and routing operations." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_fleet_observability_reporting -> fu_fu_device_identity_secrets "Uses controlled credentials for audit-log writes." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_fleet_observability_reporting -> ref_ref_cloud_logging_service "Persists telemetry, OTA outcomes, and audit records." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_machine_telemetry_collection -> fu_fu_device_identity_secrets "Uses machine identity material for authenticated publish." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_machine_telemetry_collection -> fu_fu_fleet_ingestion_api "Sends normalized telemetry records to ingestion API." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_machine_telemetry_collection -> ref_ref_iot_ingest_endpoint "Publishes telemetry payloads to cloud ingestion." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_machine_telemetry_collection -> ref_ref_mqtt_device_sdk "Uses MQTT client for edge publish and command channels." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_ota_update_agent -> fu_fu_device_identity_secrets "Uses trust anchors for firmware and endpoint authentication." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_ota_update_agent -> fu_fu_update_campaign_orchestration "Receives OTA rollout commands and policy constraints." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_ota_update_agent -> ref_ref_firmware_bundle_store "Downloads signed firmware bundle." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_update_campaign_orchestration -> fu_fu_device_identity_secrets "Uses trusted control-plane credentials." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_update_campaign_orchestration -> fu_fu_fleet_observability_reporting "Emits OTA rollout and rollback events for fleet visibility." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_update_campaign_orchestration -> ref_ref_firmware_signer_service "Verifies firmware artifact signature metadata." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    person_act_barista -> fu_fu_machine_telemetry_collection "Daily machine operation produces brew and status telemetry." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_fleet_operator -> fu_fu_update_campaign_orchestration "Defines OTA target cohorts and rollout windows." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_platform_operator -> fu_fu_cloud_runtime_operations "Operates deployment and runtime controls." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_security_analyst -> fu_fu_device_identity_secrets "Audits identity trust and secret rotation controls." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    av_av_malicious_firmware_bundle -> ctrl_ctrl_coffee_firmware_signature "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    av_av_spoofed_device_identity -> ctrl_ctrl_coffee_device_identity "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    fu_fu_ota_update_agent -> data_do_coffee_ota_plan "Reads rollout plan before applying firmware." "Model relationship: reads" {
      tags "Mapping,reads"
    }
    av_av_log_tampering -> fu_fu_device_identity_secrets "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_log_tampering -> fu_fu_fleet_observability_reporting "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_malicious_firmware_bundle -> fu_fu_ota_update_agent "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_malicious_firmware_bundle -> fu_fu_update_campaign_orchestration "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_spoofed_device_identity -> fu_fu_fleet_ingestion_api "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_spoofed_device_identity -> fu_fu_machine_telemetry_collection "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_telemetry_replay_abuse -> fu_fu_fleet_ingestion_api "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_telemetry_replay_abuse -> fu_fu_fleet_observability_reporting "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    fu_fu_machine_telemetry_collection -> data_do_coffee_telemetry_event "Writes normalized telemetry event payload." "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_update_campaign_orchestration -> data_do_coffee_ota_plan "Persists rollout plan and cohort metadata." "Model relationship: writes" {
      tags "Mapping,writes"
    }
    person_act_fleet_operator -> fu_fu_ota_update_agent "OTA Rollout Interaction Flow" "mqtt / ota-command" {
      tags "Flow"
    }
    fu_fu_machine_telemetry_collection -> fu_fu_fleet_ingestion_api "Telemetry Ingestion Interaction Flow" "mqtt / telemetry" {
      tags "Flow"
    }
    deploymentEnvironment "edge" {
      dn_dep_coffee_edge_fleet = deploymentNode "Coffee Edge Fleet" "device global devices" "edge-fleet" {
        tags "DeploymentTarget,edge"
        containerInstance fu_fu_ota_update_agent {
          tags "Deployed"
        }
      }
    }
    deploymentEnvironment "prod" {
      dn_dep_coffee_cloud_prod = deploymentNode "Coffee Cloud Production" "cloud us-east-1 coffee" "coffee-cloud" {
        tags "DeploymentTarget,prod"
        containerInstance fu_fu_fleet_ingestion_api {
          tags "Deployed"
        }
        containerInstance fu_fu_update_campaign_orchestration {
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

    systemContext sys_sample_coffee_fleet_ota_cloud_model "context" {
      include *
      autolayout lr
    }

    container sys_sample_coffee_fleet_ota_cloud_model "containers" {
      include *
      autolayout lr
    }
    dynamic sys_sample_coffee_fleet_ota_cloud_model "dynamic_flow_coffee_ota_rollout" "OTA campaign command path from operator scheduling through edge execution and reporting." {
      person_act_fleet_operator -> fu_fu_ota_update_agent "OTA Rollout Interaction Flow" "mqtt / ota-command"
      autolayout lr
    }
    dynamic sys_sample_coffee_fleet_ota_cloud_model "dynamic_flow_coffee_telemetry_ingest" "Telemetry path from device event collection to cloud ingest and reporting." {
      fu_fu_machine_telemetry_collection -> fu_fu_fleet_ingestion_api "Telemetry Ingestion Interaction Flow" "mqtt / telemetry"
      autolayout lr
    }
    deployment sys_sample_coffee_fleet_ota_cloud_model "edge" "deployment_edge" "Deployment view for environment: edge" {
      include *
      autolayout lr
    }
    deployment sys_sample_coffee_fleet_ota_cloud_model "prod" "deployment_prod" "Deployment view for environment: prod" {
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
