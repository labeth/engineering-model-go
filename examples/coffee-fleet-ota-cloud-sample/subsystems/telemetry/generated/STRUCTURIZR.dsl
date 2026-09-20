workspace "Machine Telemetry Subsystem" "The machine telemetry subsystem samples coffee machine sensors and reports signed telemetry to the fleet. It is a leaf system of the connected coffee fleet, hosted on the coffee machine controller." {
  model {
    sys_cof_telemetry = softwareSystem "Machine Telemetry Subsystem" "The machine telemetry subsystem samples coffee machine sensors and reports signed telemetry to the fleet. It is a leaf system of the connected coffee fleet, hosted on the coffee machine controller." {
      group "Telemetry" {
        fu_fu_telem_report = container "Telemetry Reporting" "Encodes, signs, and reports telemetry to the fleet ingestion endpoint." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_telem_sample = container "Telemetry Sampling" "Samples machine sensors at the configured cadence." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    group_fg_telemetry = softwareSystem "Telemetry" "Telemetry collection and reporting." {
      tags "FunctionalGroup"
    }
    if_if_telem_report = softwareSystem "Telemetry Report Interface" "https /telemetry" {
      tags "Interface"
    }
    data_do_telem_record = softwareSystem "Telemetry Record" "" {
      tags "DataObject"
    }
    ctrl_ctrl_telem_integrity = softwareSystem "Telemetry Integrity" "Sign telemetry records to protect integrity in transit." {
      tags "Control,integrity"
    }
    group_fg_telemetry -> fu_fu_telem_report "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_telemetry -> fu_fu_telem_sample "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_cof_telemetry "context" {
      include *
      autolayout lr
    }

    container sys_cof_telemetry "containers" {
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
