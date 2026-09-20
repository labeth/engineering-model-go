workspace "Atlas Shared Compliance Baseline" "NIST and ISO aligned company control baseline with verification evidence." {
  model {
    sys_atlas_shared_compliance_baseline = softwareSystem "Atlas Shared Compliance Baseline" "NIST and ISO aligned company control baseline with verification evidence." {
      group "Compliance Assurance" {
        fu_fu_comp_evidence = container "Compliance Evidence Service" "Collects, retains, and reports control assessment evidence." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_comp_people = container "People Security and Awareness" "Governs screening, employment responsibilities, awareness training, role changes, and offboarding." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_comp_officer = person "Compliance Officer" "Approves controls and remediation plans." {
      tags "Actor"
    }
    person_act_employee = person "Employee" "Completes policy acknowledgment and security awareness obligations." {
      tags "Actor"
    }
    person_act_line_manager = person "Line Manager" "Initiates and confirms role-change access reviews." {
      tags "Actor"
    }
    person_act_people_operations = person "People Operations Partner" "Coordinates personnel screening, employment records, and offboarding." {
      tags "Actor"
    }
    group_fg_comp = softwareSystem "Compliance Assurance" "Shared control and evidence lifecycle." {
      tags "FunctionalGroup"
    }
    data_do_comp_evidence = softwareSystem "Compliance Evidence Package" "" {
      tags "DataObject,restricted"
    }
    data_do_comp_personnel_record = softwareSystem "Personnel Lifecycle Record" "" {
      tags "DataObject,restricted"
    }
    data_do_comp_training_record = softwareSystem "Security Awareness Training Record" "" {
      tags "DataObject,internal"
    }
    ctrl_ctrl_comp_access_review = softwareSystem "Quarterly Access Review" "Reviews privileged access quarterly." {
      tags "Control,access-control"
    }
    ctrl_ctrl_comp_evidence_retention = softwareSystem "Evidence Retention" "Retains signed assessment evidence for seven years." {
      tags "Control,audit"
    }
    ctrl_ctrl_comp_offboarding = softwareSystem "Personnel Offboarding" "Coordinates access revocation, responsibility transfer, and company asset return." {
      tags "Control,personnel-security"
    }
    ctrl_ctrl_comp_personnel_screening = softwareSystem "Personnel Screening" "Applies risk-appropriate screening before privileged access is authorized." {
      tags "Control,personnel-security"
    }
    ctrl_ctrl_comp_policy_acknowledgment = softwareSystem "Employment Security Responsibilities" "Records acceptance of security and acceptable-use responsibilities." {
      tags "Control,personnel-security"
    }
    ctrl_ctrl_comp_role_change_review = softwareSystem "Role Change Review" "Reassesses access and responsibilities when an employee changes role." {
      tags "Control,access-control"
    }
    ctrl_ctrl_comp_security_awareness = softwareSystem "Security Awareness Education" "Provides initial and annual awareness education with completion tracking." {
      tags "Control,awareness-and-training"
    }
    person_act_people_operations -> fu_fu_comp_people "Employee Security Lifecycle" "Model flow: business" {
      tags "Flow"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_atlas_shared_compliance_baseline "context" {
      include *
      autolayout lr
    }

    container sys_atlas_shared_compliance_baseline "containers" {
      include *
      autolayout lr
    }
    dynamic sys_atlas_shared_compliance_baseline "dynamic_flow_comp_employee_lifecycle" "Company workflow from pre-employment screening through onboarding, role changes, and offboarding." {
      person_act_people_operations -> fu_fu_comp_people "Employee Security Lifecycle" "Model flow: business"
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
