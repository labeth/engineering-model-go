workspace "Connected Coffee Fleet OTA + Cloud Architecture" "This document models connected coffee machines that send telemetry to a central cloud site, receive OTA firmware updates, and produce operational and audit logging evidence. Authored functional architecture remains stable while runtime, code, and verification layers are inferred from infrastructure, source, and test artifacts." {
  model {
    sys_sample_coffee_fleet_ota_cloud_model = softwareSystem "Connected Coffee Fleet OTA + Cloud Architecture" "This document models connected coffee machines that send telemetry to a central cloud site, receive OTA firmware updates, and produce operational and audit logging evidence. Authored functional architecture remains stable while runtime, code, and verification layers are inferred from infrastructure, source, and test artifacts." {
      group "Cloud Control" {
        fu_fu_fleet_ingestion_api = container "Fleet Ingestion API" "Receives telemetry and machine status, applies validation, and emits ingestion events." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-CLOUD-CONTROL"
            "sourceId" "FU-FLEET-INGESTION-API"
          }
        }
        fu_fu_fleet_observability_reporting = container "Fleet Observability Reporting" "Aggregates telemetry and OTA outcomes into dashboards, alerts, and audit records." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-CLOUD-CONTROL"
            "sourceId" "FU-FLEET-OBSERVABILITY-REPORTING"
          }
        }
        fu_fu_update_campaign_orchestration = container "Update Campaign Orchestration" "Plans OTA rollout cohorts, drives update commands, and handles rollback flow decisions." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-CLOUD-CONTROL"
            "sourceId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
          }
        }
      }
      group "Machine Edge" {
        fu_fu_machine_telemetry_collection = container "Machine Telemetry Collection" "Collects brew/device telemetry and transmits cloud-bound telemetry payloads." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-MACHINE-EDGE"
            "sourceId" "FU-MACHINE-TELEMETRY-COLLECTION"
          }
        }
        fu_fu_ota_update_agent = container "OTA Update Agent" "Downloads signed firmware bundles, validates integrity, applies updates, and reports status." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-MACHINE-EDGE"
            "sourceId" "FU-OTA-UPDATE-AGENT"
          }
        }
      }
      group "Platform Operations" {
        fu_fu_cloud_runtime_operations = container "Cloud Runtime Operations" "Owns runtime deployment lifecycle, scaling controls, and runtime operational safety." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PLATFORM-OPERATIONS"
            "sourceId" "FU-CLOUD-RUNTIME-OPERATIONS"
          }
        }
        fu_fu_device_identity_secrets = container "Device Identity and Secrets" "Manages device credentials, cloud tokens, and secure configuration used by edge/cloud units." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-PLATFORM-OPERATIONS"
            "sourceId" "FU-DEVICE-IDENTITY-SECRETS"
          }
        }
      }
    }
    person_act_barista = person "Barista" "Operates coffee machines and observes local device status." {
      tags "Actor"
      properties {
        "sourceId" "ACT-BARISTA"
      }
    }
    person_act_fleet_operator = person "Fleet Operator" "Manages OTA campaigns and fleet-wide operational decisions." {
      tags "Actor"
      properties {
        "sourceId" "ACT-FLEET-OPERATOR"
      }
    }
    person_act_platform_operator = person "Platform Operator" "Operates cloud runtime, deployment, and reliability controls." {
      tags "Actor"
      properties {
        "sourceId" "ACT-PLATFORM-OPERATOR"
      }
    }
    person_act_security_analyst = person "Security Analyst" "Reviews firmware integrity, identity controls, and audit logging." {
      tags "Actor"
      properties {
        "sourceId" "ACT-SECURITY-ANALYST"
      }
    }
    group_fg_cloud_control = softwareSystem "Cloud Control" "Central cloud ingestion, campaign control, and fleet reporting." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-CLOUD-CONTROL"
      }
    }
    group_fg_machine_edge = softwareSystem "Machine Edge" "Device-local telemetry and OTA execution responsibilities." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-MACHINE-EDGE"
      }
    }
    group_fg_platform_operations = softwareSystem "Platform Operations" "Cloud runtime operations plus identity and secret-management controls." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-PLATFORM-OPERATIONS"
      }
    }
    ref_ref_cloud_logging_service = softwareSystem "Cloud Logging Service" "runtime" {
      tags "ReferencedElement,platform_service"
      properties {
        "kind" "platform_service"
        "layer" "runtime"
        "sourceId" "REF-CLOUD-LOGGING-SERVICE"
      }
    }
    ref_ref_cloud_runtime_sdk = softwareSystem "Cloud Runtime SDK" "code" {
      tags "ReferencedElement,third_party_library"
      properties {
        "kind" "third_party_library"
        "layer" "code"
        "sourceId" "REF-CLOUD-RUNTIME-SDK"
      }
    }
    ref_ref_firmware_bundle_store = softwareSystem "Firmware Bundle Store" "runtime" {
      tags "ReferencedElement,object_store"
      properties {
        "kind" "object_store"
        "layer" "runtime"
        "sourceId" "REF-FIRMWARE-BUNDLE-STORE"
      }
    }
    ref_ref_firmware_signer_service = softwareSystem "Firmware Signer Service" "runtime" {
      tags "ReferencedElement,signing_service"
      properties {
        "kind" "signing_service"
        "layer" "runtime"
        "sourceId" "REF-FIRMWARE-SIGNER-SERVICE"
      }
    }
    ref_ref_iot_ingest_endpoint = softwareSystem "IoT Ingest Endpoint" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-IOT-INGEST-ENDPOINT"
      }
    }
    ref_ref_mqtt_device_sdk = softwareSystem "MQTT Device SDK" "code" {
      tags "ReferencedElement,third_party_library"
      properties {
        "kind" "third_party_library"
        "layer" "code"
        "sourceId" "REF-MQTT-DEVICE-SDK"
      }
    }
    if_if_coffee_ota_command = softwareSystem "OTA Command Interface" "mqtt mqtt://devices/ota/commands" {
      tags "Interface"
      properties {
        "endpoint" "mqtt://devices/ota/commands"
        "owner" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
        "protocol" "mqtt"
        "sourceId" "IF-COFFEE-OTA-COMMAND"
      }
    }
    if_if_coffee_telemetry_ingest = softwareSystem "Telemetry Ingestion Interface" "mqtt mqtt://ingest/topics/telemetry" {
      tags "Interface"
      properties {
        "endpoint" "mqtt://ingest/topics/telemetry"
        "owner" "FU-FLEET-INGESTION-API"
        "protocol" "mqtt"
        "sourceId" "IF-COFFEE-TELEMETRY-INGEST"
      }
    }
    data_do_coffee_ota_plan = softwareSystem "OTA Rollout Plan" "schemas/ota-plan.json" {
      tags "DataObject,internal"
      properties {
        "classification" "deployment-control"
        "retention" "365_days"
        "sourceId" "DO-COFFEE-OTA-PLAN"
      }
    }
    data_do_coffee_telemetry_event = softwareSystem "Telemetry Event" "schemas/telemetry-event.json" {
      tags "DataObject,internal"
      properties {
        "classification" "operational-metrics"
        "retention" "180_days"
        "sourceId" "DO-COFFEE-TELEMETRY-EVENT"
      }
    }
    ctrl_ctrl_coffee_device_identity = softwareSystem "Device Identity Validation" "Require authenticated device identity for telemetry and OTA command channels." {
      tags "Control,identity-access"
      properties {
        "sourceId" "CTRL-COFFEE-DEVICE-IDENTITY"
      }
    }
    ctrl_ctrl_coffee_firmware_signature = softwareSystem "Firmware Signature Enforcement" "Require signed firmware bundles before applying OTA updates." {
      tags "Control,integrity"
      properties {
        "sourceId" "CTRL-COFFEE-FIRMWARE-SIGNATURE"
      }
    }
    av_av_log_tampering = softwareSystem "Log Tampering" "Attempts to suppress or alter audit/operational logs." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-LOG-TAMPERING"
      }
    }
    av_av_malicious_firmware_bundle = softwareSystem "Malicious Firmware Bundle" "Tampered OTA payload targeting update agent execution." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-MALICIOUS-FIRMWARE-BUNDLE"
      }
    }
    av_av_spoofed_device_identity = softwareSystem "Spoofed Device Identity" "Unauthorized identity attempting telemetry/command access." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-SPOOFED-DEVICE-IDENTITY"
      }
    }
    av_av_telemetry_replay_abuse = softwareSystem "Telemetry Replay Abuse" "Replayed telemetry intended to overload ingestion and corrupt insights." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-TELEMETRY-REPLAY-ABUSE"
      }
    }
    tb_tb_coffee_cloud_control = softwareSystem "Cloud Control Boundary" "Boundary between cloud app workloads and privileged platform controls." {
      tags "TrustBoundary,control-plane"
      properties {
        "sourceId" "TB-COFFEE-CLOUD-CONTROL"
      }
    }
    tb_tb_coffee_edge_device = softwareSystem "Edge Device Boundary" "Boundary between managed edge devices and cloud control plane." {
      tags "TrustBoundary,device-network"
      properties {
        "sourceId" "TB-COFFEE-EDGE-DEVICE"
      }
    }
    ts_ts_coffee_log_tampering = softwareSystem "Audit log tampering suppresses OTA incident visibility" "Adversary attempts to alter or suppress security-relevant logs to hide malicious behavior." {
      tags "ThreatScenario,repudiation,identified"
      properties {
        "impact" "high"
        "likelihood" "low"
        "severity" "medium"
        "sourceId" "TS-COFFEE-LOG-TAMPERING"
      }
    }
    ts_ts_coffee_malicious_ota_bundle = softwareSystem "Tampered firmware bundle executes on edge fleet" "Attacker attempts to deliver malicious firmware by replacing or tampering rollout artifact." {
      tags "ThreatScenario,tampering,mitigating"
      properties {
        "impact" "high"
        "likelihood" "medium"
        "severity" "high"
        "sourceId" "TS-COFFEE-MALICIOUS-OTA-BUNDLE"
      }
    }
    ts_ts_coffee_ota_command_spoof = softwareSystem "Spoofed OTA command triggers unauthorized rollout" "Unauthorized sender attempts to push OTA command messages to device fleet topic." {
      tags "ThreatScenario,spoofing,mitigating"
      properties {
        "impact" "high"
        "likelihood" "medium"
        "severity" "high"
        "sourceId" "TS-COFFEE-OTA-COMMAND-SPOOF"
      }
    }
    ts_ts_coffee_telemetry_replay = softwareSystem "Telemetry replay inflates ingest and distorts reporting" "Replayed telemetry bursts attempt to pollute fleet metrics and trigger false alerts." {
      tags "ThreatScenario,replay,mitigating"
      properties {
        "impact" "medium"
        "likelihood" "medium"
        "severity" "medium"
        "sourceId" "TS-COFFEE-TELEMETRY-REPLAY"
      }
    }
    fu_fu_cloud_runtime_operations -> tb_tb_coffee_cloud_control "Cloud control-plane trust separation." {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-CLOUD-RUNTIME-OPERATIONS"
        "mappingType" "bounded_by"
        "toId" "TB-COFFEE-CLOUD-CONTROL"
      }
    }
    fu_fu_ota_update_agent -> tb_tb_coffee_edge_device "OTA execution constrained by edge trust boundary." {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-OTA-UPDATE-AGENT"
        "mappingType" "bounded_by"
        "toId" "TB-COFFEE-EDGE-DEVICE"
      }
    }
    fu_fu_machine_telemetry_collection -> if_if_coffee_telemetry_ingest "Sends telemetry payloads via telemetry ingest interface." {
      tags "Mapping,calls"
      properties {
        "fromId" "FU-MACHINE-TELEMETRY-COLLECTION"
        "mappingType" "calls"
        "toId" "IF-COFFEE-TELEMETRY-INGEST"
      }
    }
    fu_fu_update_campaign_orchestration -> if_if_coffee_ota_command "Publishes OTA rollout commands to device fleet." {
      tags "Mapping,calls"
      properties {
        "fromId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
        "mappingType" "calls"
        "toId" "IF-COFFEE-OTA-COMMAND"
      }
    }
    group_fg_cloud_control -> fu_fu_fleet_ingestion_api "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-CLOUD-CONTROL"
        "mappingType" "contains"
        "toId" "FU-FLEET-INGESTION-API"
      }
    }
    group_fg_cloud_control -> fu_fu_fleet_observability_reporting "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-CLOUD-CONTROL"
        "mappingType" "contains"
        "toId" "FU-FLEET-OBSERVABILITY-REPORTING"
      }
    }
    group_fg_cloud_control -> fu_fu_update_campaign_orchestration "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-CLOUD-CONTROL"
        "mappingType" "contains"
        "toId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
      }
    }
    group_fg_machine_edge -> fu_fu_machine_telemetry_collection "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-MACHINE-EDGE"
        "mappingType" "contains"
        "toId" "FU-MACHINE-TELEMETRY-COLLECTION"
      }
    }
    group_fg_machine_edge -> fu_fu_ota_update_agent "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-MACHINE-EDGE"
        "mappingType" "contains"
        "toId" "FU-OTA-UPDATE-AGENT"
      }
    }
    group_fg_platform_operations -> fu_fu_cloud_runtime_operations "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PLATFORM-OPERATIONS"
        "mappingType" "contains"
        "toId" "FU-CLOUD-RUNTIME-OPERATIONS"
      }
    }
    group_fg_platform_operations -> fu_fu_device_identity_secrets "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-PLATFORM-OPERATIONS"
        "mappingType" "contains"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    fu_fu_fleet_ingestion_api -> if_if_coffee_telemetry_ingest "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-FLEET-INGESTION-API"
        "mappingType" "contains"
        "toId" "IF-COFFEE-TELEMETRY-INGEST"
      }
    }
    fu_fu_machine_telemetry_collection -> data_do_coffee_telemetry_event "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-MACHINE-TELEMETRY-COLLECTION"
        "mappingType" "contains"
        "toId" "DO-COFFEE-TELEMETRY-EVENT"
      }
    }
    fu_fu_update_campaign_orchestration -> data_do_coffee_ota_plan "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
        "mappingType" "contains"
        "toId" "DO-COFFEE-OTA-PLAN"
      }
    }
    fu_fu_update_campaign_orchestration -> if_if_coffee_ota_command "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
        "mappingType" "contains"
        "toId" "IF-COFFEE-OTA-COMMAND"
      }
    }
    fu_fu_cloud_runtime_operations -> fu_fu_device_identity_secrets "Requires runtime credentials and encrypted config." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-CLOUD-RUNTIME-OPERATIONS"
        "mappingType" "depends_on"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    fu_fu_cloud_runtime_operations -> ref_ref_cloud_logging_service "Emits platform operation and deployment events." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-CLOUD-RUNTIME-OPERATIONS"
        "mappingType" "depends_on"
        "toId" "REF-CLOUD-LOGGING-SERVICE"
      }
    }
    fu_fu_fleet_ingestion_api -> fu_fu_device_identity_secrets "Validates device identity and token claims." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-FLEET-INGESTION-API"
        "mappingType" "depends_on"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    fu_fu_fleet_ingestion_api -> fu_fu_fleet_observability_reporting "Forwards validated telemetry and machine status events." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-FLEET-INGESTION-API"
        "mappingType" "depends_on"
        "toId" "FU-FLEET-OBSERVABILITY-REPORTING"
      }
    }
    fu_fu_fleet_ingestion_api -> ref_ref_cloud_runtime_sdk "Uses cloud SDK clients for ingest and routing operations." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-FLEET-INGESTION-API"
        "mappingType" "depends_on"
        "toId" "REF-CLOUD-RUNTIME-SDK"
      }
    }
    fu_fu_fleet_observability_reporting -> fu_fu_device_identity_secrets "Uses controlled credentials for audit-log writes." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-FLEET-OBSERVABILITY-REPORTING"
        "mappingType" "depends_on"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    fu_fu_fleet_observability_reporting -> ref_ref_cloud_logging_service "Persists telemetry, OTA outcomes, and audit records." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-FLEET-OBSERVABILITY-REPORTING"
        "mappingType" "depends_on"
        "toId" "REF-CLOUD-LOGGING-SERVICE"
      }
    }
    fu_fu_machine_telemetry_collection -> fu_fu_device_identity_secrets "Uses machine identity material for authenticated publish." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-MACHINE-TELEMETRY-COLLECTION"
        "mappingType" "depends_on"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    fu_fu_machine_telemetry_collection -> fu_fu_fleet_ingestion_api "Sends normalized telemetry records to ingestion API." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-MACHINE-TELEMETRY-COLLECTION"
        "mappingType" "depends_on"
        "toId" "FU-FLEET-INGESTION-API"
      }
    }
    fu_fu_machine_telemetry_collection -> ref_ref_iot_ingest_endpoint "Publishes telemetry payloads to cloud ingestion." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-MACHINE-TELEMETRY-COLLECTION"
        "mappingType" "depends_on"
        "toId" "REF-IOT-INGEST-ENDPOINT"
      }
    }
    fu_fu_machine_telemetry_collection -> ref_ref_mqtt_device_sdk "Uses MQTT client for edge publish and command channels." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-MACHINE-TELEMETRY-COLLECTION"
        "mappingType" "depends_on"
        "toId" "REF-MQTT-DEVICE-SDK"
      }
    }
    fu_fu_ota_update_agent -> fu_fu_device_identity_secrets "Uses trust anchors for firmware and endpoint authentication." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-OTA-UPDATE-AGENT"
        "mappingType" "depends_on"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    fu_fu_ota_update_agent -> fu_fu_update_campaign_orchestration "Receives OTA rollout commands and policy constraints." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-OTA-UPDATE-AGENT"
        "mappingType" "depends_on"
        "toId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
      }
    }
    fu_fu_ota_update_agent -> ref_ref_firmware_bundle_store "Downloads signed firmware bundle." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-OTA-UPDATE-AGENT"
        "mappingType" "depends_on"
        "toId" "REF-FIRMWARE-BUNDLE-STORE"
      }
    }
    fu_fu_update_campaign_orchestration -> fu_fu_device_identity_secrets "Uses trusted control-plane credentials." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    fu_fu_update_campaign_orchestration -> fu_fu_fleet_observability_reporting "Emits OTA rollout and rollback events for fleet visibility." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-FLEET-OBSERVABILITY-REPORTING"
      }
    }
    fu_fu_update_campaign_orchestration -> ref_ref_firmware_signer_service "Verifies firmware artifact signature metadata." {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "REF-FIRMWARE-SIGNER-SERVICE"
      }
    }
    person_act_barista -> fu_fu_machine_telemetry_collection "Daily machine operation produces brew and status telemetry." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-BARISTA"
        "mappingType" "interacts_with"
        "toId" "FU-MACHINE-TELEMETRY-COLLECTION"
      }
    }
    person_act_fleet_operator -> fu_fu_update_campaign_orchestration "Defines OTA target cohorts and rollout windows." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-FLEET-OPERATOR"
        "mappingType" "interacts_with"
        "toId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
      }
    }
    person_act_platform_operator -> fu_fu_cloud_runtime_operations "Operates deployment and runtime controls." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-PLATFORM-OPERATOR"
        "mappingType" "interacts_with"
        "toId" "FU-CLOUD-RUNTIME-OPERATIONS"
      }
    }
    person_act_security_analyst -> fu_fu_device_identity_secrets "Audits identity trust and secret rotation controls." {
      tags "Mapping,interacts_with"
      properties {
        "fromId" "ACT-SECURITY-ANALYST"
        "mappingType" "interacts_with"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    av_av_malicious_firmware_bundle -> ctrl_ctrl_coffee_firmware_signature "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-MALICIOUS-FIRMWARE-BUNDLE"
        "mappingType" "mitigated_by"
        "toId" "CTRL-COFFEE-FIRMWARE-SIGNATURE"
      }
    }
    av_av_spoofed_device_identity -> ctrl_ctrl_coffee_device_identity "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-SPOOFED-DEVICE-IDENTITY"
        "mappingType" "mitigated_by"
        "toId" "CTRL-COFFEE-DEVICE-IDENTITY"
      }
    }
    fu_fu_ota_update_agent -> data_do_coffee_ota_plan "Reads rollout plan before applying firmware." {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-OTA-UPDATE-AGENT"
        "mappingType" "reads"
        "toId" "DO-COFFEE-OTA-PLAN"
      }
    }
    av_av_log_tampering -> fu_fu_device_identity_secrets "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-LOG-TAMPERING"
        "mappingType" "targets"
        "toId" "FU-DEVICE-IDENTITY-SECRETS"
      }
    }
    av_av_log_tampering -> fu_fu_fleet_observability_reporting "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-LOG-TAMPERING"
        "mappingType" "targets"
        "toId" "FU-FLEET-OBSERVABILITY-REPORTING"
      }
    }
    av_av_malicious_firmware_bundle -> fu_fu_ota_update_agent "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-MALICIOUS-FIRMWARE-BUNDLE"
        "mappingType" "targets"
        "toId" "FU-OTA-UPDATE-AGENT"
      }
    }
    av_av_malicious_firmware_bundle -> fu_fu_update_campaign_orchestration "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-MALICIOUS-FIRMWARE-BUNDLE"
        "mappingType" "targets"
        "toId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
      }
    }
    av_av_spoofed_device_identity -> fu_fu_fleet_ingestion_api "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-SPOOFED-DEVICE-IDENTITY"
        "mappingType" "targets"
        "toId" "FU-FLEET-INGESTION-API"
      }
    }
    av_av_spoofed_device_identity -> fu_fu_machine_telemetry_collection "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-SPOOFED-DEVICE-IDENTITY"
        "mappingType" "targets"
        "toId" "FU-MACHINE-TELEMETRY-COLLECTION"
      }
    }
    av_av_telemetry_replay_abuse -> fu_fu_fleet_ingestion_api "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-TELEMETRY-REPLAY-ABUSE"
        "mappingType" "targets"
        "toId" "FU-FLEET-INGESTION-API"
      }
    }
    av_av_telemetry_replay_abuse -> fu_fu_fleet_observability_reporting "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-TELEMETRY-REPLAY-ABUSE"
        "mappingType" "targets"
        "toId" "FU-FLEET-OBSERVABILITY-REPORTING"
      }
    }
    fu_fu_machine_telemetry_collection -> data_do_coffee_telemetry_event "Writes normalized telemetry event payload." {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-MACHINE-TELEMETRY-COLLECTION"
        "mappingType" "writes"
        "toId" "DO-COFFEE-TELEMETRY-EVENT"
      }
    }
    fu_fu_update_campaign_orchestration -> data_do_coffee_ota_plan "Persists rollout plan and cohort metadata." {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
        "mappingType" "writes"
        "toId" "DO-COFFEE-OTA-PLAN"
      }
    }
    person_act_fleet_operator -> fu_fu_ota_update_agent "OTA Rollout Interaction Flow" {
      tags "Flow"
      properties {
        "flowId" "FLOW-COFFEE-OTA-ROLLOUT"
      }
    }
    deploymentEnvironment "edge" {
      dn_dep_coffee_edge_fleet = deploymentNode "Coffee Edge Fleet" "device global devices" "edge-fleet" {
        tags "DeploymentTarget,edge"
        properties {
          "account" "device"
          "cluster" "edge-fleet"
          "environment" "edge"
          "namespace" "devices"
          "region" "global"
          "sourceId" "DEP-COFFEE-EDGE-FLEET"
          "trustZone" "edge"
        }
        containerInstance fu_fu_ota_update_agent {
          tags "Deployed"
          properties {
            "sourceId" "FU-OTA-UPDATE-AGENT"
          }
        }
      }
    }
    deploymentEnvironment "prod" {
      dn_dep_coffee_cloud_prod = deploymentNode "Coffee Cloud Production" "cloud us-east-1 coffee" "coffee-cloud" {
        tags "DeploymentTarget,prod"
        properties {
          "account" "cloud"
          "cluster" "coffee-cloud"
          "environment" "prod"
          "namespace" "coffee"
          "region" "us-east-1"
          "sourceId" "DEP-COFFEE-CLOUD-PROD"
          "trustZone" "app"
        }
        containerInstance fu_fu_fleet_ingestion_api {
          tags "Deployed"
          properties {
            "sourceId" "FU-FLEET-INGESTION-API"
          }
        }
        containerInstance fu_fu_update_campaign_orchestration {
          tags "Deployed"
          properties {
            "sourceId" "FU-UPDATE-CAMPAIGN-ORCHESTRATION"
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

    systemContext sys_sample_coffee_fleet_ota_cloud_model "context" {
      include *
      autolayout lr
    }

    container sys_sample_coffee_fleet_ota_cloud_model "containers" {
      include *
      autolayout lr
    }
    dynamic sys_sample_coffee_fleet_ota_cloud_model "dynamic_flow_coffee_ota_rollout" "OTA campaign command path from operator scheduling through edge execution and reporting." {
      person_act_fleet_operator -> fu_fu_ota_update_agent "OTA campaign command path from operator scheduling through edge execution and reporting."
      autolayout lr
    }
    dynamic sys_sample_coffee_fleet_ota_cloud_model "dynamic_flow_coffee_telemetry_ingest" "Telemetry path from device event collection to cloud ingest and reporting." {
      fu_fu_machine_telemetry_collection -> fu_fu_fleet_ingestion_api "Telemetry path from device event collection to cloud ingest and reporting."
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

}
