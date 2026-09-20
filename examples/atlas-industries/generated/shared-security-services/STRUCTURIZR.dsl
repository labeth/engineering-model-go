workspace "Atlas Shared Security Services" "Company identity, key management, and detection services." {
  model {
    sys_atlas_shared_security_services = softwareSystem "Atlas Shared Security Services" "Company identity, key management, and detection services." {
      group "Security Services" {
        fu_fu_sec_identity = container "Identity Service" "Issues short-lived device and workload credentials." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_sec_monitoring = container "Security Monitoring" "Correlates authentication and audit events." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_sec_analyst = person "Security Analyst" "Owns threat monitoring and response." {
      tags "Actor"
    }
    group_fg_sec = softwareSystem "Security Services" "Common zero-trust identity and monitoring." {
      tags "FunctionalGroup"
    }
    if_if_sec_token = softwareSystem "Identity Token Interface" "https https://identity.atlas.example/token" {
      tags "Interface"
    }
    data_do_sec_audit_event = softwareSystem "Security Audit Event" "" {
      tags "DataObject,restricted"
    }
    ctrl_ctrl_sec_audit_integrity = softwareSystem "Audit Integrity" "Signs and immutably stores security audit events." {
      tags "Control,audit"
    }
    ctrl_ctrl_sec_short_lived_identity = softwareSystem "Short-lived Identity" "Issues scoped credentials with bounded lifetime." {
      tags "Control,identity-access"
    }
    av_av_credential_replay = softwareSystem "Credential Replay" "Captured credentials are replayed against a trusted endpoint." {
      tags "AttackVector"
    }
    tb_tb_sec_control_plane = softwareSystem "Security Control Plane" "Separates privileged identity operations from product workloads." {
      tags "TrustBoundary,control-plane"
    }
    ts_ts_sec_credential_replay = softwareSystem "Captured token is replayed" "An adversary reuses a captured identity token." {
      tags "ThreatScenario,spoofing,mitigating"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_atlas_shared_security_services "context" {
      include *
      autolayout lr
    }

    container sys_atlas_shared_security_services "containers" {
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
