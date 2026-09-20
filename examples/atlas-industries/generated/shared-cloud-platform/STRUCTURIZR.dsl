workspace "Atlas Shared Cloud Platform" "Reusable multi-region application runtime and event services." {
  model {
    sys_atlas_shared_cloud_platform = softwareSystem "Atlas Shared Cloud Platform" "Reusable multi-region application runtime and event services." {
      group "Cloud Platform" {
        fu_fu_cloud_runtime = container "Cloud Runtime" "Hosts product APIs and durable event processing." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_cloud_operator = person "Cloud Operator" "Owns cloud availability and recovery." {
      tags "Actor"
    }
    group_fg_cloud = softwareSystem "Cloud Platform" "Shared resilient cloud capabilities." {
      tags "FunctionalGroup"
    }
    if_if_cloud_events = softwareSystem "Cloud Event Ingestion" "https https://events.atlas.example/v1" {
      tags "Interface"
    }
    data_do_cloud_event = softwareSystem "Cloud Event" "" {
      tags "DataObject,confidential"
    }
    ctrl_ctrl_cloud_backup = softwareSystem "Encrypted Backup" "Encrypts and regularly restores cloud backups." {
      tags "Control,resilience"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_atlas_shared_cloud_platform "context" {
      include *
      autolayout lr
    }

    container sys_atlas_shared_cloud_platform "containers" {
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
