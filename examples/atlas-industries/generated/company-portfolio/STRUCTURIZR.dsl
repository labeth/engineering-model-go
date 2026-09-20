workspace "Atlas Industries Company Portfolio" "Company-level composition of Aegis Sentinel and Orion Relay." {
  model {
    sys_atlas_industries_portfolio = softwareSystem "Atlas Industries Company Portfolio" "Company-level composition of Aegis Sentinel and Orion Relay." {
      group "Portfolio Governance" {
        fu_fu_port_governance = container "Portfolio Governance Service" "Consolidates product capability and assurance status." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Relay Mission" {
        fu_orion_fu_ori_link = container "orion::FU-ORI-LINK — Link Scheduler Software" "Selects available radio links and transmission windows." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_orion_fu_ori_router = container "orion::FU-ORI-ROUTER — Message Router Software" "Stores, prioritizes, and forwards operational messages." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Sentinel Mission" {
        fu_aegis_fu_aeg_mission = container "aegis::FU-AEG-MISSION — Mission Coordination Software" "Presents incidents for operator review and dispatch." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_aegis_fu_aeg_sensor_fusion = container "aegis::FU-AEG-SENSOR-FUSION — Sensor Fusion Software" "Correlates observations and assigns confidence." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_port_executive = person "Portfolio Executive" "Owns company product outcomes." {
      tags "Actor"
    }
    person_aegis_act_aeg_engineer = person "aegis::ACT-AEG-ENGINEER — Product Security Engineer" "Owns product threats and controls." {
      tags "Actor"
    }
    person_aegis_act_aeg_operator = person "aegis::ACT-AEG-OPERATOR — Security Operator" "Reviews tracks and authorizes response." {
      tags "Actor"
    }
    person_orion_act_ori_controller = person "orion::ACT-ORI-CONTROLLER — Network Controller" "Sets delivery priorities and monitors link state." {
      tags "Actor"
    }
    person_orion_act_ori_engineer = person "orion::ACT-ORI-ENGINEER — Relay Security Engineer" "Owns product security risk." {
      tags "Actor"
    }
    group_fg_port = softwareSystem "Portfolio Governance" "Company capability and assurance governance." {
      tags "FunctionalGroup"
    }
    group_aegis_fg_aeg = softwareSystem "aegis::FG-AEG — Sentinel Mission" "Product-specific sensing and incident coordination." {
      tags "FunctionalGroup"
    }
    group_orion_fg_ori = softwareSystem "orion::FG-ORI — Relay Mission" "Product-specific link scheduling and delivery." {
      tags "FunctionalGroup"
    }
    if_aegis_if_aeg_track = softwareSystem "aegis::IF-AEG-TRACK — Perimeter Track Interface" "protobuf grpc://mission/tracks" {
      tags "Interface"
    }
    if_orion_if_ori_message = softwareSystem "orion::IF-ORI-MESSAGE — Relay Message Interface" "quic quic://relay/messages" {
      tags "Interface"
    }
    data_do_port_status = softwareSystem "Portfolio Status" "" {
      tags "DataObject,confidential"
    }
    data_aegis_do_aeg_track = softwareSystem "aegis::DO-AEG-TRACK — Perimeter Track" "" {
      tags "DataObject,confidential"
    }
    data_orion_do_ori_message = softwareSystem "orion::DO-ORI-MESSAGE — Relay Message" "" {
      tags "DataObject,restricted"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_atlas_industries_portfolio "context" {
      include *
      autolayout lr
    }

    container sys_atlas_industries_portfolio "containers" {
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
