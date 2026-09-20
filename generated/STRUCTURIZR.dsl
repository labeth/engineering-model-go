workspace "Engineering Model Go Repository Architecture" "This architecture models the engineering-model-go repository as a model-driven toolchain. It captures authored functional intent, export surfaces, and verification-oriented traceability. The model is intended to guide implementation work in this repository using stable IDs and support paths." {
  model {
    sys_engineering_model_go = softwareSystem "Engineering Model Go Repository Architecture" "This architecture models the engineering-model-go repository as a model-driven toolchain. It captures authored functional intent, export surfaces, and verification-oriented traceability. The model is intended to guide implementation work in this repository using stable IDs and support paths." {
      group "Artifact Generation" {
        fu_fu_asciidoc_generator = container "AsciiDoc Generator" "Renders architecture publication docs and view narratives for human consumption." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_naf_exporter = container "NAF Exporter" "Validates NAF 4.1 stakeholders, concerns, and viewpoint selections and renders architecture products from existing canonical views." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_structurizr_exporter = container "Structurizr Exporter" "Emits Structurizr DSL and deployment-aware model views." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_sysml_exporter = container "SysML Exporter" "Generates SysML v2 textual and project interchange artifacts from the canonical semantic model while reporting conformance coverage." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_threat_exporter = container "Threat Exporter" "Exports Threat Dragon and Open OTM model artifacts." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_view_projection = container "View Projection" "Builds projection graphs for architecture, traceability, security, and flow views." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "MCP Integration" {
        fu_fu_mcp_server = container "MCP Server" "Serves model-backed tools over MCP with path safety and structured errors." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Model Authoring" {
        fu_fu_cli_orchestration = container "CLI Orchestration" "Command entrypoints orchestrating model load, validation, generation, and exports." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_model_change = container "Model Change" "Plans, validates, summarizes, and atomically applies typed stable-ID changes to authored model documents." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_model_loader = container "Model Loader" "Validates canonical YAML against the authoritative CUE contract before strict Go decoding and normalization." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_system_composition = container "System Composition" "Resolves downward subsystem references from local paths, external git repositories, or exact-version CUE modules fetched from OCI registries into the .engmod cache, and composes a federated system-of-systems model." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Traceability and Compliance" {
        fu_fu_airborne_assurance_exporter = container "Airborne Assurance Exporter" "Validates aviation assurance evidence-readiness profiles and generates clearly marked draft lifecycle-data indexes, trace matrices, gap reports, and certification-support summaries without claiming compliance or authority acceptance." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_allocation_trace = container "Allocation Trace" "Materializes parent-to-subsystem allocation and bidirectional traceability across composed systems, without modifying subsystem models." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_gemara_exporter = container "Gemara Exporter" "Renders OpenSSF Gemara L1-L7 catalogs and logs (vector, principle, guidance, control, capability, threat, risk, evaluation, enforcement, audit) with an optional OSCAL catalog and assessment-results bridge, validated by the Gemara SDK." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_lobster_exporter = container "LOBSTER Exporter" "Generates LOBSTER traceability inputs and reports." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_oscal_exporter = container "OSCAL Exporter" "Exports OSCAL System Security Plan (SSP), Assessment Results (AR), and Plan of Action and Milestones (POA&M)." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_trlc_exporter = container "TRLC Exporter" "Emits TRLC model and requirement artifacts." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
      group "Validation and Analysis" {
        fu_fu_codemap_inference = container "Codemap Inference" "Infers code/runtime ownership and verification links from source and tests." "Functional Unit" {
          tags "FunctionalUnit"
        }
        fu_fu_validation_engine = container "Validation Engine" "Validates authored entities, IDs, references, and mapping consistency." "Functional Unit" {
          tags "FunctionalUnit"
        }
      }
    }
    person_act_ai_agent = person "AI Agent" "Uses MCP tools to plan and execute scoped implementation work." {
      tags "Actor"
    }
    person_act_architecture_author = person "Architecture Author" "Maintains modeled structure, mappings, and architecture intent." {
      tags "Actor"
    }
    person_act_ci_pipeline = person "CI Pipeline" "Executes regression tests, validation checks, and artifact generation." {
      tags "Actor"
    }
    person_act_compliance_engineer = person "Compliance Engineer" "Uses control/risk and traceability outputs for assurance workflows." {
      tags "Actor"
    }
    person_act_implementation_engineer = person "Implementation Engineer" "Implements code and tests guided by requirement support paths." {
      tags "Actor"
    }
    group_fg_artifact_generation = softwareSystem "Artifact Generation" "Publication and exchange artifact generation." {
      tags "FunctionalGroup"
    }
    group_fg_mcp_integration = softwareSystem "MCP Integration" "AI-agent integration and runtime API responsibilities." {
      tags "FunctionalGroup"
    }
    group_fg_model_authoring = softwareSystem "Model Authoring" "Inputs and model loading responsibilities." {
      tags "FunctionalGroup"
    }
    group_fg_traceability_compliance = softwareSystem "Traceability and Compliance" "Trace and compliance export responsibilities." {
      tags "FunctionalGroup"
    }
    group_fg_validation_analysis = softwareSystem "Validation and Analysis" "Validation and inference responsibilities." {
      tags "FunctionalGroup"
    }
    ref_ref_cue_oci_module_registry = softwareSystem "CUE OCI Module Registry" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_go_toolchain = softwareSystem "Go Toolchain" "code" {
      tags "ReferencedElement,platform_service"
    }
    ref_ref_lobster_toolchain = softwareSystem "LOBSTER Toolchain" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_naf_v4_1_specification = softwareSystem "NATO Architecture Framework v4.1" "architecture" {
      tags "ReferencedElement,external_standard"
    }
    ref_ref_open_otm_schema = softwareSystem "Open OTM Schema" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_structurizr_validator = softwareSystem "Structurizr Validator" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_sysml_v2_toolchain = softwareSystem "Pinned SysML 2.0 / KerML 1.0 Toolchain" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_threat_dragon_schemas = softwareSystem "Threat Dragon Schemas" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    ref_ref_trlc_toolchain = softwareSystem "TRLC Toolchain" "runtime" {
      tags "ReferencedElement,external_service_endpoint"
    }
    if_if_cli_engair = softwareSystem "engair CLI" "cli cmd/engair" {
      tags "Interface"
    }
    if_if_cli_engchange = softwareSystem "engchange CLI" "cli cmd/engchange" {
      tags "Interface"
    }
    if_if_cli_engdoc = softwareSystem "engdoc CLI" "cli cmd/engdoc" {
      tags "Interface"
    }
    if_if_cli_engdragon = softwareSystem "engdragon CLI" "cli cmd/engdragon" {
      tags "Interface"
    }
    if_if_cli_enggemara = softwareSystem "enggemara CLI" "cli cmd/enggemara" {
      tags "Interface"
    }
    if_if_cli_englobster = softwareSystem "englobster CLI" "cli cmd/englobster" {
      tags "Interface"
    }
    if_if_cli_engmcp = softwareSystem "engmcp CLI" "stdio-jsonrpc cmd/engmcp" {
      tags "Interface"
    }
    if_if_cli_engnaf = softwareSystem "engnaf CLI" "cli cmd/engnaf" {
      tags "Interface"
    }
    if_if_cli_engoscal = softwareSystem "engoscal CLI" "cli cmd/engoscal" {
      tags "Interface"
    }
    if_if_cli_engstruct = softwareSystem "engstruct CLI" "cli cmd/engstruct" {
      tags "Interface"
    }
    if_if_cli_engsysml = softwareSystem "engsysml CLI" "cli cmd/engsysml" {
      tags "Interface"
    }
    if_if_cli_engtrace = softwareSystem "engtrace CLI" "cli cmd/engtrace" {
      tags "Interface"
    }
    if_if_cli_engtrlc = softwareSystem "engtrlc CLI" "cli cmd/engtrlc" {
      tags "Interface"
    }
    if_if_cli_engview = softwareSystem "engview CLI" "cli cmd/engview" {
      tags "Interface"
    }
    data_do_architecture_model = softwareSystem "Architecture Model" "model/schema/architecture.cue" {
      tags "DataObject,internal"
    }
    data_do_canonical_semantic_model = softwareSystem "Canonical Semantic Model" "model/semantic.go" {
      tags "DataObject,internal"
    }
    data_do_mcp_tool_result = softwareSystem "MCP Tool Result" "mcp.tool-response.v1" {
      tags "DataObject,internal"
    }
    data_do_model_authoring_contract = softwareSystem "Model Authoring Contract" "model/authoring.go" {
      tags "DataObject,internal"
    }
    data_do_naf_v4_architecture = softwareSystem "NAF v4.1 Architecture Document" "generated/ARCHITECTURE.naf.adoc" {
      tags "DataObject,internal"
    }
    data_do_requirements_delta = softwareSystem "Requirements Delta" "requirements_delta.go" {
      tags "DataObject,internal"
    }
    data_do_requirements_diff = softwareSystem "Requirements Change Summary" "requirements_delta.go" {
      tags "DataObject,internal"
    }
    data_do_requirements_document = softwareSystem "Requirements Document" "model/schema/requirements.cue" {
      tags "DataObject,internal"
    }
    data_do_structurizr_dsl = softwareSystem "Structurizr DSL" "generated/STRUCTURIZR.dsl" {
      tags "DataObject,internal"
    }
    data_do_sysml_v2_metamodel = softwareSystem "SysML v2 Metamodel" "tools/sysml/metamodel" {
      tags "DataObject,public"
    }
    data_do_sysml_v2_model = softwareSystem "SysML v2 Model" "generated/ARCHITECTURE.sysml" {
      tags "DataObject,internal"
    }
    data_do_threat_dragon_json = softwareSystem "Threat Dragon JSON" "generated/threat-dragon-v2.json" {
      tags "DataObject,internal"
    }
    ctrl_ctrl_artifact_freshness_gate = softwareSystem "Generated Artifact Freshness Gate" "Generate artifacts deterministically and fail continuous integration when regenerated documents differ from the committed artifacts." {
      tags "Control,assurance"
    }
    ctrl_ctrl_mcp_path_boundary = softwareSystem "MCP Path Boundary Enforcement" "Enforce repo-root path constraints and reject traversal paths in MCP tools." {
      tags "Control,input-validation"
    }
    ctrl_ctrl_strict_mcp_input_schema = softwareSystem "Strict MCP Input Schemas" "Enforce per-tool input schema and reject unknown arguments." {
      tags "Control,input-validation"
    }
    ctrl_ctrl_trace_link_integrity = softwareSystem "Trace Link Integrity" "Reject code trace links (TRLC-LINKS, ENGMODEL-LINKS) that resolve to no requirement or model element, and export a consolidated traceability matrix with an implemented, verified, delegated, and orphan rollup." {
      tags "Control,assurance"
    }
    ctrl_ctrl_traceability_coverage = softwareSystem "Requirement Traceability Coverage" "Require requirement-linked verification evidence for modeled behavior." {
      tags "Control,assurance"
    }
    av_av_malformed_model_input = softwareSystem "Malformed Model Input" "Invalid or ambiguous model content causing incorrect parsing or graph interpretation." {
      tags "AttackVector"
    }
    av_av_path_traversal_in_mcp = softwareSystem "Path Traversal in MCP Calls" "Attempts to read files outside repository root via MCP path arguments." {
      tags "AttackVector"
    }
    av_av_schema_supply_chain_tamper = softwareSystem "Schema Supply Chain Tamper" "Drift or tampering in external schemas used by export validation." {
      tags "AttackVector"
    }
    av_av_traceability_gap_drift = softwareSystem "Traceability Gap Drift" "Requirement and verification links diverge from implementation over time." {
      tags "AttackVector"
    }
    tb_tb_external_validation_tools = softwareSystem "External Validation Tools Boundary" "Boundary for external validators, schemas, and compliance toolchains." {
      tags "TrustBoundary,external-tooling"
    }
    tb_tb_repo_workspace = softwareSystem "Repository Workspace Boundary" "Boundary limiting file operations to repository-root owned paths." {
      tags "TrustBoundary,filesystem"
    }
    ts_ts_mcp_path_traversal = softwareSystem "MCP path traversal accesses non-repo files" "Untrusted MCP input attempts to escape repo root for sensitive file reads." {
      tags "ThreatScenario,tampering,mitigating"
    }
    ts_ts_traceability_drift = softwareSystem "Requirement traceability drifts from implementation" "Code and tests change without corresponding requirement trace updates." {
      tags "ThreatScenario,repudiation,mitigating"
    }
    fu_fu_codemap_inference -> tb_tb_repo_workspace "bounded_by" "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_lobster_exporter -> tb_tb_external_validation_tools "bounded_by" "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_mcp_server -> tb_tb_repo_workspace "bounded_by" "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_structurizr_exporter -> tb_tb_external_validation_tools "bounded_by" "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_sysml_exporter -> tb_tb_external_validation_tools "bounded_by" "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_threat_exporter -> tb_tb_external_validation_tools "bounded_by" "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    fu_fu_trlc_exporter -> tb_tb_external_validation_tools "bounded_by" "Model relationship: bounded_by" {
      tags "Mapping,bounded_by"
    }
    group_fg_artifact_generation -> fu_fu_asciidoc_generator "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_artifact_generation -> fu_fu_naf_exporter "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_artifact_generation -> fu_fu_structurizr_exporter "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_artifact_generation -> fu_fu_sysml_exporter "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_artifact_generation -> fu_fu_threat_exporter "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_artifact_generation -> fu_fu_view_projection "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_mcp_integration -> fu_fu_mcp_server "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_model_authoring -> fu_fu_cli_orchestration "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_model_authoring -> fu_fu_model_change "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_model_authoring -> fu_fu_model_loader "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_model_authoring -> fu_fu_system_composition "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_traceability_compliance -> fu_fu_allocation_trace "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_traceability_compliance -> fu_fu_gemara_exporter "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_traceability_compliance -> fu_fu_lobster_exporter "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_traceability_compliance -> fu_fu_oscal_exporter "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_traceability_compliance -> fu_fu_trlc_exporter "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_validation_analysis -> fu_fu_codemap_inference "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    group_fg_validation_analysis -> fu_fu_validation_engine "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_allocation_trace -> if_if_cli_engtrace "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_asciidoc_generator -> if_if_cli_engdoc "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_gemara_exporter -> if_if_cli_enggemara "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_lobster_exporter -> if_if_cli_englobster "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_mcp_server -> if_if_cli_engmcp "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_model_change -> if_if_cli_engchange "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_naf_exporter -> if_if_cli_engnaf "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_oscal_exporter -> if_if_cli_engoscal "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_structurizr_exporter -> if_if_cli_engstruct "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_sysml_exporter -> if_if_cli_engsysml "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_threat_exporter -> if_if_cli_engdragon "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_trlc_exporter -> if_if_cli_engtrlc "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_view_projection -> if_if_cli_engview "contains" "Model relationship: contains" {
      tags "Mapping,contains"
    }
    fu_fu_cli_orchestration -> fu_fu_asciidoc_generator "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_cli_orchestration -> fu_fu_model_loader "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_cli_orchestration -> fu_fu_validation_engine "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_lobster_exporter -> ref_ref_lobster_toolchain "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_mcp_server -> fu_fu_model_loader "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_mcp_server -> fu_fu_validation_engine "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_model_change -> fu_fu_model_loader "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_model_change -> fu_fu_validation_engine "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_naf_exporter -> fu_fu_view_projection "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_naf_exporter -> ref_ref_naf_v4_1_specification "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_structurizr_exporter -> ref_ref_structurizr_validator "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_sysml_exporter -> ref_ref_sysml_v2_toolchain "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_system_composition -> ref_ref_cue_oci_module_registry "Fetches exact-version reusable model modules using CUE registry configuration and credentials." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_threat_exporter -> ref_ref_open_otm_schema "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_threat_exporter -> ref_ref_threat_dragon_schemas "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_trlc_exporter -> ref_ref_trlc_toolchain "depends_on" "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    fu_fu_validation_engine -> ref_ref_go_toolchain "Runs Go tests and static checks as part of repository validation." "Model relationship: depends_on" {
      tags "Mapping,depends_on"
    }
    person_act_ai_agent -> fu_fu_mcp_server "Uses model-backed MCP tools to inspect and update implementation scope." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_architecture_author -> fu_fu_cli_orchestration "Maintains model and design artifacts through CLI generation workflows." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_ci_pipeline -> fu_fu_cli_orchestration "Executes validation and generation commands in automated checks." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_compliance_engineer -> fu_fu_oscal_exporter "Uses OSCAL and traceability outputs for assurance workflows." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    person_act_implementation_engineer -> fu_fu_cli_orchestration "Runs validation, generation, and export commands during implementation work." "Model relationship: interacts_with" {
      tags "Mapping,interacts_with"
    }
    av_av_path_traversal_in_mcp -> ctrl_ctrl_mcp_path_boundary "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    av_av_path_traversal_in_mcp -> ctrl_ctrl_strict_mcp_input_schema "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    av_av_traceability_gap_drift -> ctrl_ctrl_traceability_coverage "mitigated_by" "Model relationship: mitigated_by" {
      tags "Mapping,mitigated_by"
    }
    fu_fu_model_change -> data_do_requirements_delta "reads" "Model relationship: reads" {
      tags "Mapping,reads"
    }
    fu_fu_model_loader -> data_do_architecture_model "reads" "Model relationship: reads" {
      tags "Mapping,reads"
    }
    fu_fu_sysml_exporter -> data_do_sysml_v2_metamodel "reads" "Model relationship: reads" {
      tags "Mapping,reads"
    }
    av_av_malformed_model_input -> fu_fu_model_loader "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_path_traversal_in_mcp -> fu_fu_mcp_server "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_schema_supply_chain_tamper -> ref_ref_open_otm_schema "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_schema_supply_chain_tamper -> ref_ref_threat_dragon_schemas "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    av_av_traceability_gap_drift -> fu_fu_codemap_inference "targets" "Model relationship: targets" {
      tags "Mapping,targets"
    }
    fu_fu_mcp_server -> data_do_mcp_tool_result "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_mcp_server -> data_do_model_authoring_contract "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_model_change -> data_do_requirements_diff "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_model_change -> data_do_requirements_document "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_model_loader -> data_do_canonical_semantic_model "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_naf_exporter -> data_do_naf_v4_architecture "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_structurizr_exporter -> data_do_structurizr_dsl "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_sysml_exporter -> data_do_sysml_v2_model "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    fu_fu_threat_exporter -> data_do_threat_dragon_json "writes" "Model relationship: writes" {
      tags "Mapping,writes"
    }
    person_act_implementation_engineer -> fu_fu_model_change "Requirements Delta to Canonical Requirements Flow" "cli / local-shell" {
      tags "Flow"
    }
    person_act_implementation_engineer -> fu_fu_cli_orchestration "Model Change to Verified Artifacts Flow" "cli / local-shell" {
      tags "Flow"
    }
    deploymentEnvironment "ci" {
      dn_dep_ci_pipeline = deploymentNode "CI Pipeline Runner" "github-actions hosted engineering-model-go" "shared-runner" {
        tags "DeploymentTarget,ci"
        containerInstance fu_fu_cli_orchestration {
          tags "Deployed"
        }
      }
    }
    deploymentEnvironment "dev" {
      dn_dep_local_workspace = deploymentNode "Local Workspace" "workstation local engineering-model-go" "local-shell" {
        tags "DeploymentTarget,dev"
        containerInstance fu_fu_mcp_server {
          tags "Deployed"
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
      person_act_implementation_engineer -> fu_fu_cli_orchestration "Model Change to Verified Artifacts Flow" "cli / local-shell"
      autolayout lr
    }
    dynamic sys_engineering_model_go "dynamic_flow_requirements_delta_to_canonical" "Strictly plans and validates a requirements delta before atomically updating canonical requirements." {
      person_act_implementation_engineer -> fu_fu_model_change "Requirements Delta to Canonical Requirements Flow" "cli / local-shell"
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

  configuration {
    scope softwaresystem
  }

}
