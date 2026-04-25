{
  "schema": "lobster-report",
  "version": 2,
  "generator": "lobster_report",
  "levels": [
    {
      "name": "Requirements",
      "kind": "requirements",
      "items": [
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_001",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 2,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_001",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.contract_telemetry_ingest_contract_test_go"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "While coffee machine fleet operation is active, when brew cycle completed event is received, the coffee fleet system shall publish brew telemetry to fleet ingestion api for barista operations.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_002",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 8,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_002",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.integration_logging_pipeline_test_ts"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "While coffee machine fleet operation is active, when telemetry ingested event is received, the coffee fleet system shall persist fleet telemetry metrics for fleet operator.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_003",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 14,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_003",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.e2e_ota_rollout_flow_yaml",
            "act tests.unit_ota_campaign_test_go"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "Where ota campaign is enabled, when ota campaign scheduled event is received, the coffee fleet system shall distribute signed firmware bundle to targeted machine cohort using ota campaign policy.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_004",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 20,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_004",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.unit_ota_campaign_test_go"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "Where ota campaign is enabled, when ota campaign scheduled event is received, the coffee fleet system shall gate firmware apply on firmware signature is valid.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_005",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 26,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_005",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.e2e_ota_rollout_flow_yaml"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If firmware validation failed event is received, then the coffee fleet system shall rollback firmware apply.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_006",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 32,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_006",
          "messages": [
            "missing reference to Tests"
          ],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "tracing_status": "MISSING",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If cloud ingest endpoint is unavailable, then the coffee fleet system shall queue telemetry for retry.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_007",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 38,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_007",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.integration_logging_pipeline_test_ts"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "Where audit logging is enabled, when ota status reported event is received, the coffee fleet system shall persist redacted audit log record with machine identity plus campaign id.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_008",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 44,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_008",
          "messages": [
            "missing reference to Tests"
          ],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "tracing_status": "MISSING",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If telemetry replay abuse is detected, then the coffee fleet system shall throttle duplicate telemetry.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_009",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 50,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_009",
          "messages": [
            "missing reference to Tests"
          ],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "tracing_status": "MISSING",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "While coffee machine fleet operation is active, when telemetry ingested event is received, the coffee fleet system shall persist logging records for fleet operator.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_010",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 56,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_010",
          "messages": [
            "missing reference to Tests"
          ],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "tracing_status": "MISSING",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If firmware validation failed event is received, then the coffee fleet system shall report ota status to platform operator.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_011",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 62,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_011",
          "messages": [
            "missing reference to Tests"
          ],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "tracing_status": "MISSING",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If cloud ingest endpoint is unavailable, then the coffee fleet system shall flush queued telemetry when normal mode is active.",
          "status": null
        },
        {
          "tag": "req CoffeeFleetRequirements.REQ_COF_012",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/trlc/requirements.trlc",
            "line": 68,
            "column": 13
          },
          "name": "CoffeeFleetRequirements.REQ_COF_012",
          "messages": [
            "missing reference to Tests"
          ],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "tracing_status": "MISSING",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If telemetry replay abuse is detected, then the coffee fleet system shall notify security analyst.",
          "status": null
        }
      ],
      "coverage": 50.0
    },
    {
      "name": "Tests",
      "kind": "activity",
      "items": [
        {
          "tag": "act tests.contract_telemetry_ingest_contract_test_go",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/tests/contract/telemetry_ingest_contract_test.go",
            "line": 1,
            "column": 1
          },
          "name": "tests.contract_telemetry_ingest_contract_test_go",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req CoffeeFleetRequirements.REQ_COF_001"
          ],
          "ref_down": [],
          "tracing_status": "OK",
          "framework": "Tests",
          "kind": "test",
          "status": "ok"
        },
        {
          "tag": "act tests.e2e_ota_rollout_flow_yaml",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/tests/e2e/ota_rollout_flow.yaml",
            "line": 1,
            "column": 1
          },
          "name": "tests.e2e_ota_rollout_flow_yaml",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req CoffeeFleetRequirements.REQ_COF_005",
            "req CoffeeFleetRequirements.REQ_COF_003"
          ],
          "ref_down": [],
          "tracing_status": "OK",
          "framework": "Tests",
          "kind": "test",
          "status": "ok"
        },
        {
          "tag": "act tests.integration_logging_pipeline_test_ts",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/tests/integration/logging_pipeline_test.ts",
            "line": 1,
            "column": 1
          },
          "name": "tests.integration_logging_pipeline_test_ts",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req CoffeeFleetRequirements.REQ_COF_007",
            "req CoffeeFleetRequirements.REQ_COF_002"
          ],
          "ref_down": [],
          "tracing_status": "OK",
          "framework": "Tests",
          "kind": "test",
          "status": "ok"
        },
        {
          "tag": "act tests.unit_ota_campaign_test_go",
          "location": {
            "kind": "file",
            "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/tests/unit/ota_campaign_test.go",
            "line": 1,
            "column": 1
          },
          "name": "tests.unit_ota_campaign_test_go",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req CoffeeFleetRequirements.REQ_COF_004",
            "req CoffeeFleetRequirements.REQ_COF_003"
          ],
          "ref_down": [],
          "tracing_status": "OK",
          "framework": "Tests",
          "kind": "test",
          "status": "ok"
        }
      ],
      "coverage": 100.0
    }
  ],
  "policy": {
    "Requirements": {
      "name": "Requirements",
      "kind": "requirements",
      "traces": [],
      "source": [
        {
          "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/lobster/requirements.lobster"
        }
      ],
      "needs_tracing_up": false,
      "needs_tracing_down": true,
      "breakdown_requirements": [
        [
          "Tests"
        ]
      ]
    },
    "Tests": {
      "name": "Tests",
      "kind": "activity",
      "traces": [
        "Requirements"
      ],
      "source": [
        {
          "file": "/home/labeth/ws/engineering-model-go/examples/coffee-fleet-ota-cloud-sample/generated/lobster/activities.lobster"
        }
      ],
      "needs_tracing_up": true,
      "needs_tracing_down": false,
      "breakdown_requirements": []
    }
  },
  "matrix": []
}
