workspace "Engineering Model Go Repository Architecture" "This architecture models the engineering-model-go repository as a model-driven toolchain. It captures authored functional intent, export surfaces, and verification-oriented traceability. The model is intended to guide implementation work in this repository using stable IDs and support paths." {
  model {
    sys_engineering_model_go = softwareSystem "Engineering Model Go Repository Architecture" "This architecture models the engineering-model-go repository as a model-driven toolchain. It captures authored functional intent, export surfaces, and verification-oriented traceability. The model is intended to guide implementation work in this repository using stable IDs and support paths." {
      group "Artifact Generation" {
        fu_fu_ai_view_builder = container "AI View Builder" "Emits AI JSON, edges, and markdown views with support paths and implementation paths." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-ARTIFACT-GENERATION"
            "sourceId" "FU-AI-VIEW-BUILDER"
          }
        }
        fu_fu_asciidoc_generator = container "AsciiDoc Generator" "Renders architecture publication docs and view narratives for human consumption." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-ARTIFACT-GENERATION"
            "sourceId" "FU-ASCIIDOC-GENERATOR"
          }
        }
        fu_fu_structurizr_exporter = container "Structurizr Exporter" "Emits Structurizr DSL and deployment-aware model views." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-ARTIFACT-GENERATION"
            "sourceId" "FU-STRUCTURIZR-EXPORTER"
          }
        }
        fu_fu_threat_exporter = container "Threat Exporter" "Exports Threat Dragon and Open OTM model artifacts." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-ARTIFACT-GENERATION"
            "sourceId" "FU-THREAT-EXPORTER"
          }
        }
        fu_fu_view_projection = container "View Projection" "Builds projection graphs for architecture, traceability, security, and flow views." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-ARTIFACT-GENERATION"
            "sourceId" "FU-VIEW-PROJECTION"
          }
        }
      }
      group "MCP Integration" {
        fu_fu_mcp_server = container "MCP Server" "Serves model-backed tools over MCP with path safety and structured errors." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-MCP-INTEGRATION"
            "sourceId" "FU-MCP-SERVER"
          }
        }
      }
      group "Model Authoring" {
        fu_fu_cli_orchestration = container "CLI Orchestration" "Command entrypoints orchestrating model load, validation, generation, and exports." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-MODEL-AUTHORING"
            "sourceId" "FU-CLI-ORCHESTRATION"
          }
        }
        fu_fu_model_loader = container "Model Loader" "Loads and normalizes architecture, catalog, requirements, and design documents." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-MODEL-AUTHORING"
            "sourceId" "FU-MODEL-LOADER"
          }
        }
      }
      group "Traceability and Compliance" {
        fu_fu_lobster_exporter = container "LOBSTER Exporter" "Generates LOBSTER traceability inputs and reports." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-TRACEABILITY-COMPLIANCE"
            "sourceId" "FU-LOBSTER-EXPORTER"
          }
        }
        fu_fu_oscal_exporter = container "OSCAL Exporter" "Exports OSCAL SSP control and risk views." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-TRACEABILITY-COMPLIANCE"
            "sourceId" "FU-OSCAL-EXPORTER"
          }
        }
        fu_fu_trlc_exporter = container "TRLC Exporter" "Emits TRLC model and requirement artifacts." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-TRACEABILITY-COMPLIANCE"
            "sourceId" "FU-TRLC-EXPORTER"
          }
        }
      }
      group "Validation and Analysis" {
        fu_fu_codemap_inference = container "Codemap Inference" "Infers code/runtime ownership and verification links from source and tests." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-VALIDATION-ANALYSIS"
            "sourceId" "FU-CODEMAP-INFERENCE"
          }
        }
        fu_fu_validation_engine = container "Validation Engine" "Validates authored entities, IDs, references, and mapping consistency." "Functional Unit" {
          tags "FunctionalUnit"
          properties {
            "functionalGroup" "FG-VALIDATION-ANALYSIS"
            "sourceId" "FU-VALIDATION-ENGINE"
          }
        }
      }
    }
    person_act_ai_agent = person "AI Agent" "Uses MCP tools and AI JSON to plan and execute scoped implementation work." {
      tags "Actor"
      properties {
        "sourceId" "ACT-AI-AGENT"
      }
    }
    person_act_architecture_author = person "Architecture Author" "Maintains modeled structure, mappings, and architecture intent." {
      tags "Actor"
      properties {
        "sourceId" "ACT-ARCHITECTURE-AUTHOR"
      }
    }
    person_act_ci_pipeline = person "CI Pipeline" "Executes regression tests, validation checks, and artifact generation." {
      tags "Actor"
      properties {
        "sourceId" "ACT-CI-PIPELINE"
      }
    }
    person_act_compliance_engineer = person "Compliance Engineer" "Uses control/risk and traceability outputs for assurance workflows." {
      tags "Actor"
      properties {
        "sourceId" "ACT-COMPLIANCE-ENGINEER"
      }
    }
    person_act_implementation_engineer = person "Implementation Engineer" "Implements code and tests guided by requirement support paths." {
      tags "Actor"
      properties {
        "sourceId" "ACT-IMPLEMENTATION-ENGINEER"
      }
    }
    group_fg_artifact_generation = softwareSystem "Artifact Generation" "Publication and exchange artifact generation." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-ARTIFACT-GENERATION"
      }
    }
    group_fg_mcp_integration = softwareSystem "MCP Integration" "AI-agent integration and runtime API responsibilities." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-MCP-INTEGRATION"
      }
    }
    group_fg_model_authoring = softwareSystem "Model Authoring" "Inputs and model loading responsibilities." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-MODEL-AUTHORING"
      }
    }
    group_fg_traceability_compliance = softwareSystem "Traceability and Compliance" "Trace and compliance export responsibilities." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-TRACEABILITY-COMPLIANCE"
      }
    }
    group_fg_validation_analysis = softwareSystem "Validation and Analysis" "Validation and inference responsibilities." {
      tags "FunctionalGroup"
      properties {
        "sourceId" "FG-VALIDATION-ANALYSIS"
      }
    }
    ref_ref_go_toolchain = softwareSystem "Go Toolchain" "code" {
      tags "ReferencedElement,platform_service"
      properties {
        "kind" "platform_service"
        "layer" "code"
        "sourceId" "REF-GO-TOOLCHAIN"
      }
    }
    ref_ref_lobster_toolchain = softwareSystem "LOBSTER Toolchain" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-LOBSTER-TOOLCHAIN"
      }
    }
    ref_ref_open_otm_schema = softwareSystem "Open OTM Schema" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-OPEN-OTM-SCHEMA"
      }
    }
    ref_ref_structurizr_validator = softwareSystem "Structurizr Validator" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-STRUCTURIZR-VALIDATOR"
      }
    }
    ref_ref_threat_dragon_schemas = softwareSystem "Threat Dragon Schemas" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-THREAT-DRAGON-SCHEMAS"
      }
    }
    ref_ref_trlc_toolchain = softwareSystem "TRLC Toolchain" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
      properties {
        "kind" "external_service_endpoint"
        "layer" "runtime"
        "sourceId" "REF-TRLC-TOOLCHAIN"
      }
    }
    if_if_cli_engdoc = softwareSystem "engdoc CLI" "cli cmd/engdoc" {
      tags "Interface"
      properties {
        "endpoint" "cmd/engdoc"
        "owner" "FU-ASCIIDOC-GENERATOR"
        "protocol" "cli"
        "sourceId" "IF-CLI-ENGDOC"
      }
    }
    if_if_cli_engdragon = softwareSystem "engdragon CLI" "cli cmd/engdragon" {
      tags "Interface"
      properties {
        "endpoint" "cmd/engdragon"
        "owner" "FU-THREAT-EXPORTER"
        "protocol" "cli"
        "sourceId" "IF-CLI-ENGDRAGON"
      }
    }
    if_if_cli_englobster = softwareSystem "englobster CLI" "cli cmd/englobster" {
      tags "Interface"
      properties {
        "endpoint" "cmd/englobster"
        "owner" "FU-LOBSTER-EXPORTER"
        "protocol" "cli"
        "sourceId" "IF-CLI-ENGLOBSTER"
      }
    }
    if_if_cli_engmcp = softwareSystem "engmcp CLI" "stdio-jsonrpc cmd/engmcp" {
      tags "Interface"
      properties {
        "endpoint" "cmd/engmcp"
        "owner" "FU-MCP-SERVER"
        "protocol" "stdio-jsonrpc"
        "sourceId" "IF-CLI-ENGMCP"
      }
    }
    if_if_cli_engoscal = softwareSystem "engoscal CLI" "cli cmd/engoscal" {
      tags "Interface"
      properties {
        "endpoint" "cmd/engoscal"
        "owner" "FU-OSCAL-EXPORTER"
        "protocol" "cli"
        "sourceId" "IF-CLI-ENGOSCAL"
      }
    }
    if_if_cli_engstruct = softwareSystem "engstruct CLI" "cli cmd/engstruct" {
      tags "Interface"
      properties {
        "endpoint" "cmd/engstruct"
        "owner" "FU-STRUCTURIZR-EXPORTER"
        "protocol" "cli"
        "sourceId" "IF-CLI-ENGSTRUCT"
      }
    }
    if_if_cli_engtrlc = softwareSystem "engtrlc CLI" "cli cmd/engtrlc" {
      tags "Interface"
      properties {
        "endpoint" "cmd/engtrlc"
        "owner" "FU-TRLC-EXPORTER"
        "protocol" "cli"
        "sourceId" "IF-CLI-ENGTRLC"
      }
    }
    if_if_cli_engview = softwareSystem "engview CLI" "cli cmd/engview" {
      tags "Interface"
      properties {
        "endpoint" "cmd/engview"
        "owner" "FU-VIEW-PROJECTION"
        "protocol" "cli"
        "sourceId" "IF-CLI-ENGVIEW"
      }
    }
    data_do_ai_json_artifact = softwareSystem "AI JSON Artifact" "generated/ARCHITECTURE.ai.json" {
      tags "DataObject,internal"
      properties {
        "classification" "machine-interface"
        "retention" "build-artifact"
        "sourceId" "DO-AI-JSON-ARTIFACT"
      }
    }
    data_do_architecture_model = softwareSystem "Architecture Model" "architecture.yml" {
      tags "DataObject,internal"
      properties {
        "classification" "design-source"
        "retention" "repository-history"
        "sourceId" "DO-ARCHITECTURE-MODEL"
      }
    }
    data_do_mcp_tool_result = softwareSystem "MCP Tool Result" "mcp.tool-response.v1" {
      tags "DataObject,internal"
      properties {
        "classification" "runtime-api"
        "retention" "ephemeral"
        "sourceId" "DO-MCP-TOOL-RESULT"
      }
    }
    data_do_structurizr_dsl = softwareSystem "Structurizr DSL" "generated/STRUCTURIZR.dsl" {
      tags "DataObject,internal"
      properties {
        "classification" "exchange-artifact"
        "retention" "build-artifact"
        "sourceId" "DO-STRUCTURIZR-DSL"
      }
    }
    data_do_threat_dragon_json = softwareSystem "Threat Dragon JSON" "generated/threat-dragon-v2.json" {
      tags "DataObject,internal"
      properties {
        "classification" "exchange-artifact"
        "retention" "build-artifact"
        "sourceId" "DO-THREAT-DRAGON-JSON"
      }
    }
    ctrl_ctrl_mcp_path_boundary = softwareSystem "MCP Path Boundary Enforcement" "Enforce repo-root path constraints and reject traversal paths in MCP tools." {
      tags "Control,input-validation"
      properties {
        "sourceId" "CTRL-MCP-PATH-BOUNDARY"
      }
    }
    ctrl_ctrl_strict_mcp_input_schema = softwareSystem "Strict MCP Input Schemas" "Enforce per-tool input schema and reject unknown arguments." {
      tags "Control,input-validation"
      properties {
        "sourceId" "CTRL-STRICT-MCP-INPUT-SCHEMA"
      }
    }
    ctrl_ctrl_traceability_coverage = softwareSystem "Requirement Traceability Coverage" "Require requirement-linked verification evidence for modeled behavior." {
      tags "Control,assurance"
      properties {
        "sourceId" "CTRL-TRACEABILITY-COVERAGE"
      }
    }
    av_av_malformed_model_input = softwareSystem "Malformed Model Input" "Invalid or ambiguous model content causing incorrect parsing or graph interpretation." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-MALFORMED-MODEL-INPUT"
      }
    }
    av_av_path_traversal_in_mcp = softwareSystem "Path Traversal in MCP Calls" "Attempts to read files outside repository root via MCP path arguments." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-PATH-TRAVERSAL-IN-MCP"
      }
    }
    av_av_schema_supply_chain_tamper = softwareSystem "Schema Supply Chain Tamper" "Drift or tampering in external schemas used by export validation." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-SCHEMA-SUPPLY-CHAIN-TAMPER"
      }
    }
    av_av_traceability_gap_drift = softwareSystem "Traceability Gap Drift" "Requirement and verification links diverge from implementation over time." {
      tags "AttackVector"
      properties {
        "sourceId" "AV-TRACEABILITY-GAP-DRIFT"
      }
    }
    tb_tb_external_validation_tools = softwareSystem "External Validation Tools Boundary" "Boundary for external validators, schemas, and compliance toolchains." {
      tags "TrustBoundary,external-tooling"
      properties {
        "sourceId" "TB-EXTERNAL-VALIDATION-TOOLS"
      }
    }
    tb_tb_repo_workspace = softwareSystem "Repository Workspace Boundary" "Boundary limiting file operations to repository-root owned paths." {
      tags "TrustBoundary,filesystem"
      properties {
        "sourceId" "TB-REPO-WORKSPACE"
      }
    }
    ts_ts_mcp_path_traversal = softwareSystem "MCP path traversal accesses non-repo files" "Untrusted MCP input attempts to escape repo root for sensitive file reads." {
      tags "ThreatScenario,tampering,mitigating"
      properties {
        "impact" "high"
        "likelihood" "medium"
        "severity" "high"
        "sourceId" "TS-MCP-PATH-TRAVERSAL"
      }
    }
    ts_ts_traceability_drift = softwareSystem "Requirement traceability drifts from implementation" "Code and tests change without corresponding requirement trace updates." {
      tags "ThreatScenario,repudiation,mitigating"
      properties {
        "impact" "medium"
        "likelihood" "medium"
        "severity" "medium"
        "sourceId" "TS-TRACEABILITY-DRIFT"
      }
    }
    fu_fu_codemap_inference -> tb_tb_repo_workspace "bounded_by" {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-CODEMAP-INFERENCE"
        "mappingType" "bounded_by"
        "toId" "TB-REPO-WORKSPACE"
      }
    }
    fu_fu_lobster_exporter -> tb_tb_external_validation_tools "bounded_by" {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-LOBSTER-EXPORTER"
        "mappingType" "bounded_by"
        "toId" "TB-EXTERNAL-VALIDATION-TOOLS"
      }
    }
    fu_fu_mcp_server -> tb_tb_repo_workspace "bounded_by" {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-MCP-SERVER"
        "mappingType" "bounded_by"
        "toId" "TB-REPO-WORKSPACE"
      }
    }
    fu_fu_structurizr_exporter -> tb_tb_external_validation_tools "bounded_by" {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-STRUCTURIZR-EXPORTER"
        "mappingType" "bounded_by"
        "toId" "TB-EXTERNAL-VALIDATION-TOOLS"
      }
    }
    fu_fu_threat_exporter -> tb_tb_external_validation_tools "bounded_by" {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-THREAT-EXPORTER"
        "mappingType" "bounded_by"
        "toId" "TB-EXTERNAL-VALIDATION-TOOLS"
      }
    }
    fu_fu_trlc_exporter -> tb_tb_external_validation_tools "bounded_by" {
      tags "Mapping,bounded_by"
      properties {
        "fromId" "FU-TRLC-EXPORTER"
        "mappingType" "bounded_by"
        "toId" "TB-EXTERNAL-VALIDATION-TOOLS"
      }
    }
    group_fg_artifact_generation -> fu_fu_ai_view_builder "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-ARTIFACT-GENERATION"
        "mappingType" "contains"
        "toId" "FU-AI-VIEW-BUILDER"
      }
    }
    group_fg_artifact_generation -> fu_fu_asciidoc_generator "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-ARTIFACT-GENERATION"
        "mappingType" "contains"
        "toId" "FU-ASCIIDOC-GENERATOR"
      }
    }
    group_fg_artifact_generation -> fu_fu_structurizr_exporter "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-ARTIFACT-GENERATION"
        "mappingType" "contains"
        "toId" "FU-STRUCTURIZR-EXPORTER"
      }
    }
    group_fg_artifact_generation -> fu_fu_threat_exporter "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-ARTIFACT-GENERATION"
        "mappingType" "contains"
        "toId" "FU-THREAT-EXPORTER"
      }
    }
    group_fg_artifact_generation -> fu_fu_view_projection "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-ARTIFACT-GENERATION"
        "mappingType" "contains"
        "toId" "FU-VIEW-PROJECTION"
      }
    }
    group_fg_mcp_integration -> fu_fu_mcp_server "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-MCP-INTEGRATION"
        "mappingType" "contains"
        "toId" "FU-MCP-SERVER"
      }
    }
    group_fg_model_authoring -> fu_fu_cli_orchestration "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-MODEL-AUTHORING"
        "mappingType" "contains"
        "toId" "FU-CLI-ORCHESTRATION"
      }
    }
    group_fg_model_authoring -> fu_fu_model_loader "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-MODEL-AUTHORING"
        "mappingType" "contains"
        "toId" "FU-MODEL-LOADER"
      }
    }
    group_fg_traceability_compliance -> fu_fu_lobster_exporter "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-TRACEABILITY-COMPLIANCE"
        "mappingType" "contains"
        "toId" "FU-LOBSTER-EXPORTER"
      }
    }
    group_fg_traceability_compliance -> fu_fu_oscal_exporter "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-TRACEABILITY-COMPLIANCE"
        "mappingType" "contains"
        "toId" "FU-OSCAL-EXPORTER"
      }
    }
    group_fg_traceability_compliance -> fu_fu_trlc_exporter "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-TRACEABILITY-COMPLIANCE"
        "mappingType" "contains"
        "toId" "FU-TRLC-EXPORTER"
      }
    }
    group_fg_validation_analysis -> fu_fu_codemap_inference "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-VALIDATION-ANALYSIS"
        "mappingType" "contains"
        "toId" "FU-CODEMAP-INFERENCE"
      }
    }
    group_fg_validation_analysis -> fu_fu_validation_engine "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FG-VALIDATION-ANALYSIS"
        "mappingType" "contains"
        "toId" "FU-VALIDATION-ENGINE"
      }
    }
    fu_fu_asciidoc_generator -> if_if_cli_engdoc "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-ASCIIDOC-GENERATOR"
        "mappingType" "contains"
        "toId" "IF-CLI-ENGDOC"
      }
    }
    fu_fu_lobster_exporter -> if_if_cli_englobster "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-LOBSTER-EXPORTER"
        "mappingType" "contains"
        "toId" "IF-CLI-ENGLOBSTER"
      }
    }
    fu_fu_mcp_server -> if_if_cli_engmcp "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-MCP-SERVER"
        "mappingType" "contains"
        "toId" "IF-CLI-ENGMCP"
      }
    }
    fu_fu_oscal_exporter -> if_if_cli_engoscal "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-OSCAL-EXPORTER"
        "mappingType" "contains"
        "toId" "IF-CLI-ENGOSCAL"
      }
    }
    fu_fu_structurizr_exporter -> if_if_cli_engstruct "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-STRUCTURIZR-EXPORTER"
        "mappingType" "contains"
        "toId" "IF-CLI-ENGSTRUCT"
      }
    }
    fu_fu_threat_exporter -> if_if_cli_engdragon "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-THREAT-EXPORTER"
        "mappingType" "contains"
        "toId" "IF-CLI-ENGDRAGON"
      }
    }
    fu_fu_trlc_exporter -> if_if_cli_engtrlc "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-TRLC-EXPORTER"
        "mappingType" "contains"
        "toId" "IF-CLI-ENGTRLC"
      }
    }
    fu_fu_view_projection -> if_if_cli_engview "contains" {
      tags "Mapping,contains"
      properties {
        "fromId" "FU-VIEW-PROJECTION"
        "mappingType" "contains"
        "toId" "IF-CLI-ENGVIEW"
      }
    }
    fu_fu_ai_view_builder -> fu_fu_codemap_inference "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-AI-VIEW-BUILDER"
        "mappingType" "depends_on"
        "toId" "FU-CODEMAP-INFERENCE"
      }
    }
    fu_fu_ai_view_builder -> fu_fu_view_projection "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-AI-VIEW-BUILDER"
        "mappingType" "depends_on"
        "toId" "FU-VIEW-PROJECTION"
      }
    }
    fu_fu_cli_orchestration -> fu_fu_ai_view_builder "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-CLI-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-AI-VIEW-BUILDER"
      }
    }
    fu_fu_cli_orchestration -> fu_fu_asciidoc_generator "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-CLI-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-ASCIIDOC-GENERATOR"
      }
    }
    fu_fu_cli_orchestration -> fu_fu_model_loader "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-CLI-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-MODEL-LOADER"
      }
    }
    fu_fu_cli_orchestration -> fu_fu_validation_engine "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-CLI-ORCHESTRATION"
        "mappingType" "depends_on"
        "toId" "FU-VALIDATION-ENGINE"
      }
    }
    fu_fu_lobster_exporter -> ref_ref_lobster_toolchain "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-LOBSTER-EXPORTER"
        "mappingType" "depends_on"
        "toId" "REF-LOBSTER-TOOLCHAIN"
      }
    }
    fu_fu_mcp_server -> fu_fu_ai_view_builder "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-MCP-SERVER"
        "mappingType" "depends_on"
        "toId" "FU-AI-VIEW-BUILDER"
      }
    }
    fu_fu_mcp_server -> fu_fu_model_loader "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-MCP-SERVER"
        "mappingType" "depends_on"
        "toId" "FU-MODEL-LOADER"
      }
    }
    fu_fu_mcp_server -> fu_fu_validation_engine "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-MCP-SERVER"
        "mappingType" "depends_on"
        "toId" "FU-VALIDATION-ENGINE"
      }
    }
    fu_fu_structurizr_exporter -> ref_ref_structurizr_validator "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-STRUCTURIZR-EXPORTER"
        "mappingType" "depends_on"
        "toId" "REF-STRUCTURIZR-VALIDATOR"
      }
    }
    fu_fu_threat_exporter -> ref_ref_open_otm_schema "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-THREAT-EXPORTER"
        "mappingType" "depends_on"
        "toId" "REF-OPEN-OTM-SCHEMA"
      }
    }
    fu_fu_threat_exporter -> ref_ref_threat_dragon_schemas "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-THREAT-EXPORTER"
        "mappingType" "depends_on"
        "toId" "REF-THREAT-DRAGON-SCHEMAS"
      }
    }
    fu_fu_trlc_exporter -> ref_ref_trlc_toolchain "depends_on" {
      tags "Mapping,depends_on"
      properties {
        "fromId" "FU-TRLC-EXPORTER"
        "mappingType" "depends_on"
        "toId" "REF-TRLC-TOOLCHAIN"
      }
    }
    av_av_path_traversal_in_mcp -> ctrl_ctrl_mcp_path_boundary "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-PATH-TRAVERSAL-IN-MCP"
        "mappingType" "mitigated_by"
        "toId" "CTRL-MCP-PATH-BOUNDARY"
      }
    }
    av_av_path_traversal_in_mcp -> ctrl_ctrl_strict_mcp_input_schema "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-PATH-TRAVERSAL-IN-MCP"
        "mappingType" "mitigated_by"
        "toId" "CTRL-STRICT-MCP-INPUT-SCHEMA"
      }
    }
    av_av_traceability_gap_drift -> ctrl_ctrl_traceability_coverage "mitigated_by" {
      tags "Mapping,mitigated_by"
      properties {
        "fromId" "AV-TRACEABILITY-GAP-DRIFT"
        "mappingType" "mitigated_by"
        "toId" "CTRL-TRACEABILITY-COVERAGE"
      }
    }
    fu_fu_model_loader -> data_do_architecture_model "reads" {
      tags "Mapping,reads"
      properties {
        "fromId" "FU-MODEL-LOADER"
        "mappingType" "reads"
        "toId" "DO-ARCHITECTURE-MODEL"
      }
    }
    av_av_malformed_model_input -> fu_fu_model_loader "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-MALFORMED-MODEL-INPUT"
        "mappingType" "targets"
        "toId" "FU-MODEL-LOADER"
      }
    }
    av_av_path_traversal_in_mcp -> fu_fu_mcp_server "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-PATH-TRAVERSAL-IN-MCP"
        "mappingType" "targets"
        "toId" "FU-MCP-SERVER"
      }
    }
    av_av_schema_supply_chain_tamper -> ref_ref_open_otm_schema "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-SCHEMA-SUPPLY-CHAIN-TAMPER"
        "mappingType" "targets"
        "toId" "REF-OPEN-OTM-SCHEMA"
      }
    }
    av_av_schema_supply_chain_tamper -> ref_ref_threat_dragon_schemas "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-SCHEMA-SUPPLY-CHAIN-TAMPER"
        "mappingType" "targets"
        "toId" "REF-THREAT-DRAGON-SCHEMAS"
      }
    }
    av_av_traceability_gap_drift -> fu_fu_codemap_inference "targets" {
      tags "Mapping,targets"
      properties {
        "fromId" "AV-TRACEABILITY-GAP-DRIFT"
        "mappingType" "targets"
        "toId" "FU-CODEMAP-INFERENCE"
      }
    }
    fu_fu_ai_view_builder -> data_do_ai_json_artifact "writes" {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-AI-VIEW-BUILDER"
        "mappingType" "writes"
        "toId" "DO-AI-JSON-ARTIFACT"
      }
    }
    fu_fu_mcp_server -> data_do_mcp_tool_result "writes" {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-MCP-SERVER"
        "mappingType" "writes"
        "toId" "DO-MCP-TOOL-RESULT"
      }
    }
    fu_fu_structurizr_exporter -> data_do_structurizr_dsl "writes" {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-STRUCTURIZR-EXPORTER"
        "mappingType" "writes"
        "toId" "DO-STRUCTURIZR-DSL"
      }
    }
    fu_fu_threat_exporter -> data_do_threat_dragon_json "writes" {
      tags "Mapping,writes"
      properties {
        "fromId" "FU-THREAT-EXPORTER"
        "mappingType" "writes"
        "toId" "DO-THREAT-DRAGON-JSON"
      }
    }
    person_act_implementation_engineer -> fu_fu_cli_orchestration "Model Change to Verified Artifacts Flow" {
      tags "Flow"
      properties {
        "flowId" "FLOW-MODEL-CHANGE-TO-VERIFIED-ARTIFACTS"
      }
    }
    deploymentEnvironment "ci" {
      dn_dep_ci_pipeline = deploymentNode "CI Pipeline Runner" "github-actions hosted engineering-model-go" "shared-runner" {
        tags "DeploymentTarget,ci"
        properties {
          "account" "github-actions"
          "cluster" "shared-runner"
          "environment" "ci"
          "namespace" "engineering-model-go"
          "region" "hosted"
          "sourceId" "DEP-CI-PIPELINE"
          "trustZone" "automation"
        }
        containerInstance fu_fu_cli_orchestration {
          tags "Deployed"
          properties {
            "sourceId" "FU-CLI-ORCHESTRATION"
          }
        }
      }
    }
    deploymentEnvironment "dev" {
      dn_dep_local_workspace = deploymentNode "Local Workspace" "workstation local engineering-model-go" "local-shell" {
        tags "DeploymentTarget,dev"
        properties {
          "account" "workstation"
          "cluster" "local-shell"
          "environment" "dev"
          "namespace" "engineering-model-go"
          "region" "local"
          "sourceId" "DEP-LOCAL-WORKSPACE"
          "trustZone" "developer"
        }
        containerInstance fu_fu_mcp_server {
          tags "Deployed"
          properties {
            "sourceId" "FU-MCP-SERVER"
          }
        }
      }
    }
  }

  views {
    systemLandscape "landscape" {
      include *
      autolayout lr
    }

    systemContext sys_engineering_model_go "context" {
      include *
      autolayout lr
    }

    container sys_engineering_model_go "containers" {
      include *
      autolayout lr
    }
    dynamic sys_engineering_model_go "dynamic_flow_model_change_to_verified_artifacts" "Primary engineering workflow from authored model update through validation and artifact generation." {
      person_act_implementation_engineer -> fu_fu_cli_orchestration "Primary engineering workflow from authored model update through validation and artifact generation."
      autolayout lr
    }
    deployment sys_engineering_model_go "ci" "deployment_ci" "Deployment view for environment: ci" {
      include *
      autolayout lr
    }
    deployment sys_engineering_model_go "dev" "deployment_dev" "Deployment view for environment: dev" {
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
