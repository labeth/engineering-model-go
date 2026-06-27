workspace "OTA Update Agent Subsystem" "The OTA update agent subsystem verifies and applies signed firmware on the machine, rolling back on validation failure. It is a leaf system of the connected coffee fleet, hosted on the fleet edge gateway." {
  model {
    sys_cof_ota_agent = softwareSystem "OTA Update Agent Subsystem" "The OTA update agent subsystem verifies and applies signed firmware on the machine, rolling back on validation failure. It is a leaf system of the connected coffee fleet, hosted on the fleet edge gateway." {
      group "OTA" {
        fu_fu_ota_apply = container "OTA Apply" "Applies verified firmware and rolls back on failure." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-OTA"
            "sourceId" "FU-OTA-APPLY"
          }
        }
        fu_fu_ota_verify = container "OTA Verification" "Verifies firmware signature and update eligibility." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-OTA"
            "sourceId" "FU-OTA-VERIFY"
          }
        }
      }
    }
    group_fg_ota = softwareSystem "OTA" "On-machine OTA update verification and application." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-OTA"
      }
    }
    if_if_ota_apply = softwareSystem "OTA Apply Interface" "internal /ota/apply" {
      tags "Interface"
      properties {
        "endpoint" "/ota/apply"
        "owner" "FU-OTA-APPLY"
        "protocol" "internal"
        "sourceId" "IF-OTA-APPLY"
      }
    }
    data_do_firmware_bundle = softwareSystem "Firmware Bundle" "" {
      tags "DataObject"
      properties {
        "classification" "confidential"
        "sourceId" "DO-FIRMWARE-BUNDLE"
      }
    }
    ctrl_ctrl_firmware_signature = softwareSystem "Firmware Signature Verification" "Verify firmware signatures before apply." {
      tags "Control,integrity"
      properties {
        "sourceId" "CTRL-FIRMWARE-SIGNATURE"
      }
    }
    group_fg_ota -> fu_fu_ota_apply "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-OTA"
        "mappingType" "contains"
        "toId" "FU-OTA-APPLY"
      }
    }
    group_fg_ota -> fu_fu_ota_verify "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-OTA"
        "mappingType" "contains"
        "toId" "FU-OTA-VERIFY"
      }
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_cof_ota_agent "context" {
      include *
      autolayout lr
    }

    container sys_cof_ota_agent "containers" {
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
