{
  "data": [
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
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
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
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
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
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
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
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
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
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
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
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "If telemetry replay abuse is detected, then the coffee fleet system shall notify security analyst.",
      "status": null
    }
  ],
  "generator": "lobster-trlc",
  "schema": "lobster-req-trace",
  "version": 4
}
