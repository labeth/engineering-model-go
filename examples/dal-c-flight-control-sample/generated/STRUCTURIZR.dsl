workspace "DAL C Yaw Damping Readiness Sample" "Bounded readiness-only example for a non-autonomous yaw damping software item." {
  model {
    sys_dal_c_flight_control_sample = softwareSystem "DAL C Yaw Damping Readiness Sample" "Bounded readiness-only example for a non-autonomous yaw damping software item." {
      group "Flight Control" {
        fu_fu_yaw_control = container "Yaw Damping Software" "Filters yaw rate and produces a bounded command." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_systems_engineer = person "Systems Engineer" "Owns the system and software specification baseline." {
      tags "Actor"
    }
    group_fg_flight_control = softwareSystem "Flight Control" "Bounded flight-control functions." {
      tags "FunctionalGroup"
    }
    ref_ref_system_safety_assessment = softwareSystem "System Safety Assessment" "external" {
      tags "ReferencedElement,source_document"
    }
    if_if_rudder_command = softwareSystem "Rudder Command Output" "sampled-data output" {
      tags "Interface"
    }
    if_if_yaw_rate = softwareSystem "Yaw Rate Input" "sampled-data input" {
      tags "Interface"
    }
    group_fg_flight_control -> fu_fu_yaw_control "The flight-control group contains yaw damping." "Model relationship: contains" {
      tags "Mapping,contains"
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_dal_c_flight_control_sample "context" {
      include *
      autolayout lr
    }

    container sys_dal_c_flight_control_sample "containers" {
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
