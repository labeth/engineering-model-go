workspace "Atlas Shared Edge Platform" "Reusable rugged hardware and signed edge runtime." {
  model {
    sys_atlas_shared_edge_platform = softwareSystem "Atlas Shared Edge Platform" "Reusable rugged hardware and signed edge runtime." {
      group "Edge Platform" {
        fu_fu_edge_runtime = container "Edge Runtime" "Hosts signed product workloads and normalizes device observations." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_edge_operator = person "Edge Operator" "Maintains deployed edge nodes." {
      tags "Actor"
    }
    group_fg_edge = softwareSystem "Edge Platform" "Common edge compute and device integration." {
      tags "FunctionalGroup"
    }
    if_if_edge_telemetry = softwareSystem "Edge Telemetry Interface" "mqtt mqtt://edge/telemetry" {
      tags "Interface"
    }
    data_do_edge_telemetry = softwareSystem "Edge Telemetry Record" "" {
      tags "DataObject,internal"
    }
    ctrl_ctrl_edge_signed_workload = softwareSystem "Signed Workload Enforcement" "Verifies workload signatures before execution." {
      tags "Control,integrity"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_atlas_shared_edge_platform "context" {
      include *
      autolayout lr
    }

    container sys_atlas_shared_edge_platform "containers" {
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
