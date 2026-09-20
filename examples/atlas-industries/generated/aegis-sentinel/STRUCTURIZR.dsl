workspace "Aegis Sentinel Product System" "Product model for autonomous perimeter sensing and reviewed response." {
  model {
    sys_atlas_aegis_sentinel = softwareSystem "Aegis Sentinel Product System" "Product model for autonomous perimeter sensing and reviewed response." {
      group "Cloud Platform" {
        fu_cloud_fu_cloud_runtime = container "cloud::FU-CLOUD-RUNTIME — Cloud Runtime" "Hosts product APIs and durable event processing." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Compliance Assurance" {
        fu_compliance_fu_comp_evidence = container "compliance::FU-COMP-EVIDENCE — Compliance Evidence Service" "Collects, retains, and reports control assessment evidence." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_compliance_fu_comp_people = container "compliance::FU-COMP-PEOPLE — People Security and Awareness" "Governs screening, employment responsibilities, awareness training, role changes, and offboarding." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Edge Platform" {
        fu_edge_fu_edge_runtime = container "edge::FU-EDGE-RUNTIME — Edge Runtime" "Hosts signed product workloads and normalizes device observations." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Security Services" {
        fu_security_fu_sec_identity = container "security::FU-SEC-IDENTITY — Identity Service" "Issues short-lived device and workload credentials." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_security_fu_sec_monitoring = container "security::FU-SEC-MONITORING — Security Monitoring" "Correlates authentication and audit events." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Sentinel Mission" {
        fu_fu_aeg_mission = container "Mission Coordination Software" "Presents incidents for operator review and dispatch." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_aeg_sensor_fusion = container "Sensor Fusion Software" "Correlates observations and assigns confidence." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_aeg_engineer = person "Product Security Engineer" "Owns product threats and controls." {
      tags "Actor"
    }
    person_act_aeg_operator = person "Security Operator" "Reviews tracks and authorizes response." {
      tags "Actor"
    }
    person_cloud_act_cloud_operator = person "cloud::ACT-CLOUD-OPERATOR — Cloud Operator" "Owns cloud availability and recovery." {
      tags "Actor"
    }
    person_compliance_act_comp_officer = person "compliance::ACT-COMP-OFFICER — Compliance Officer" "Approves controls and remediation plans." {
      tags "Actor"
    }
    person_compliance_act_employee = person "compliance::ACT-EMPLOYEE — Employee" "Completes policy acknowledgment and security awareness obligations." {
      tags "Actor"
    }
    person_compliance_act_line_manager = person "compliance::ACT-LINE-MANAGER — Line Manager" "Initiates and confirms role-change access reviews." {
      tags "Actor"
    }
    person_compliance_act_people_operations = person "compliance::ACT-PEOPLE-OPERATIONS — People Operations Partner" "Coordinates personnel screening, employment records, and offboarding." {
      tags "Actor"
    }
    person_edge_act_edge_operator = person "edge::ACT-EDGE-OPERATOR — Edge Operator" "Maintains deployed edge nodes." {
      tags "Actor"
    }
    person_security_act_sec_analyst = person "security::ACT-SEC-ANALYST — Security Analyst" "Owns threat monitoring and response." {
      tags "Actor"
    }
    group_fg_aeg = softwareSystem "Sentinel Mission" "Product-specific sensing and incident coordination." {
      tags "FunctionalGroup"
    }
    group_cloud_fg_cloud = softwareSystem "cloud::FG-CLOUD — Cloud Platform" "Shared resilient cloud capabilities." {
      tags "FunctionalGroup"
    }
    group_compliance_fg_comp = softwareSystem "compliance::FG-COMP — Compliance Assurance" "Shared control and evidence lifecycle." {
      tags "FunctionalGroup"
    }
    group_edge_fg_edge = softwareSystem "edge::FG-EDGE — Edge Platform" "Common edge compute and device integration." {
      tags "FunctionalGroup"
    }
    group_security_fg_sec = softwareSystem "security::FG-SEC — Security Services" "Common zero-trust identity and monitoring." {
      tags "FunctionalGroup"
    }
    if_if_aeg_track = softwareSystem "Perimeter Track Interface" "protobuf grpc://mission/tracks" {
      tags "Interface"
    }
    if_cloud_if_cloud_events = softwareSystem "cloud::IF-CLOUD-EVENTS — Cloud Event Ingestion" "https https://events.atlas.example/v1" {
      tags "Interface"
    }
    if_edge_if_edge_telemetry = softwareSystem "edge::IF-EDGE-TELEMETRY — Edge Telemetry Interface" "mqtt mqtt://edge/telemetry" {
      tags "Interface"
    }
    if_security_if_sec_token = softwareSystem "security::IF-SEC-TOKEN — Identity Token Interface" "https https://identity.atlas.example/token" {
      tags "Interface"
    }
    data_do_aeg_track = softwareSystem "Perimeter Track" "" {
      tags "DataObject,confidential"
    }
    data_cloud_do_cloud_event = softwareSystem "cloud::DO-CLOUD-EVENT — Cloud Event" "" {
      tags "DataObject,confidential"
    }
    data_compliance_do_comp_evidence = softwareSystem "compliance::DO-COMP-EVIDENCE — Compliance Evidence Package" "" {
      tags "DataObject,restricted"
    }
    data_compliance_do_comp_personnel_record = softwareSystem "compliance::DO-COMP-PERSONNEL-RECORD — Personnel Lifecycle Record" "" {
      tags "DataObject,restricted"
    }
    data_compliance_do_comp_training_record = softwareSystem "compliance::DO-COMP-TRAINING-RECORD — Security Awareness Training Record" "" {
      tags "DataObject,internal"
    }
    data_edge_do_edge_telemetry = softwareSystem "edge::DO-EDGE-TELEMETRY — Edge Telemetry Record" "" {
      tags "DataObject,internal"
    }
    data_security_do_sec_audit_event = softwareSystem "security::DO-SEC-AUDIT-EVENT — Security Audit Event" "" {
      tags "DataObject,restricted"
    }
    ctrl_ctrl_aeg_track_confidence = softwareSystem "Track Confidence Gate" "Requires multi-sensor confidence before operator presentation." {
      tags "Control,mission-integrity"
    }
    av_av_aeg_sensor_spoof = softwareSystem "Sensor Spoofing" "Adversarial observations attempt to create a false perimeter track." {
      tags "AttackVector"
    }
    ts_ts_aeg_sensor_spoof = softwareSystem "Crafted observations create a false track" "An adversary injects mutually consistent false observations." {
      tags "ThreatScenario,tampering,mitigating"
    }
    person_compliance_act_people_operations -> fu_compliance_fu_comp_people "Employee Security Lifecycle" "Model flow: business" {
      tags "Flow"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_atlas_aegis_sentinel "context" {
      include *
      autolayout lr
    }

    container sys_atlas_aegis_sentinel "containers" {
      include *
      autolayout lr
    }
    dynamic sys_atlas_aegis_sentinel "dynamic_compliance_flow_comp_employee_lifecycle" "Company workflow from pre-employment screening through onboarding, role changes, and offboarding." {
      person_compliance_act_people_operations -> fu_compliance_fu_comp_people "Employee Security Lifecycle" "Model flow: business"
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
