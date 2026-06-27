workspace "Machine Telemetry Subsystem" "The machine telemetry subsystem samples coffee machine sensors and reports signed telemetry to the fleet. It is a leaf system of the connected coffee fleet, hosted on the coffee machine controller." {
  model {
    sys_cof_telemetry = softwareSystem "Machine Telemetry Subsystem" "The machine telemetry subsystem samples coffee machine sensors and reports signed telemetry to the fleet. It is a leaf system of the connected coffee fleet, hosted on the coffee machine controller." {
      group "Telemetry" {
        fu_fu_telem_report = container "Telemetry Reporting" "Encodes, signs, and reports telemetry to the fleet ingestion endpoint." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-TELEMETRY"
            "sourceId" "FU-TELEM-REPORT"
          }
        }
        fu_fu_telem_sample = container "Telemetry Sampling" "Samples machine sensors at the configured cadence." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-TELEMETRY"
            "sourceId" "FU-TELEM-SAMPLE"
          }
        }
      }
    }
    group_fg_telemetry = softwareSystem "Telemetry" "Telemetry collection and reporting." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-TELEMETRY"
      }
    }
    if_if_telem_report = softwareSystem "Telemetry Report Interface" "https /telemetry" {
      tags "Interface"
      properties {
        "endpoint" "/telemetry"
        "owner" "FU-TELEM-REPORT"
        "protocol" "https"
        "sourceId" "IF-TELEM-REPORT"
      }
    }
    data_do_telem_record = softwareSystem "Telemetry Record" "" {
      tags "DataObject"
      properties {
        "classification" "internal"
        "sourceId" "DO-TELEM-RECORD"
      }
    }
    ctrl_ctrl_telem_integrity = softwareSystem "Telemetry Integrity" "Sign telemetry records to protect integrity in transit." {
      tags "Control,integrity"
      properties {
        "sourceId" "CTRL-TELEM-INTEGRITY"
      }
    }
    group_fg_telemetry -> fu_fu_telem_report "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-TELEMETRY"
        "mappingType" "contains"
        "toId" "FU-TELEM-REPORT"
      }
    }
    group_fg_telemetry -> fu_fu_telem_sample "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-TELEMETRY"
        "mappingType" "contains"
        "toId" "FU-TELEM-SAMPLE"
      }
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

}
