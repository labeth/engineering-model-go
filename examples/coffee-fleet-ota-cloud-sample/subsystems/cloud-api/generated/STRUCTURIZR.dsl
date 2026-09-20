workspace "Cloud API Subsystem" "The cloud API subsystem ingests fleet telemetry, persists metrics, and drives OTA campaigns. It is a leaf system of the connected coffee fleet, hosted on the cloud platform." {
  model {
    sys_cof_cloud_api = softwareSystem "Cloud API Subsystem" "The cloud API subsystem ingests fleet telemetry, persists metrics, and drives OTA campaigns. It is a leaf system of the connected coffee fleet, hosted on the cloud platform." {
      group "Cloud" {
        fu_fu_cloud_campaign = container "Cloud Campaign" "Plans and dispatches OTA campaigns." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_cloud_ingest = container "Cloud Ingest" "Ingests and persists fleet telemetry." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    group_fg_cloud = softwareSystem "Cloud" "Cloud ingestion and campaign control." {
      tags "FunctionalGroup"
    }
    if_if_cloud_ingest = softwareSystem "Cloud Ingest Interface" "https /ingest" {
      tags "Interface"
    }
    data_do_fleet_metric = softwareSystem "Fleet Metric" "" {
      tags "DataObject"
    }
    ctrl_ctrl_ingest_auth = softwareSystem "Ingest Authentication" "Authenticate machines before accepting telemetry." {
      tags "Control,access-control"
    }
    group_fg_cloud -> fu_fu_cloud_campaign "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_cloud -> fu_fu_cloud_ingest "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_cof_cloud_api "context" {
      include *
      autolayout lr
    }

    container sys_cof_cloud_api "containers" {
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
