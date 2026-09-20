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
          "tag": "req requirements.REQ_PAY_001",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 2,
            "column": 13
          },
          "name": "requirements.REQ_PAY_001",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.e2e_authorized_checkout_flow_yaml"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "While normal mode is active, when payment authorization is requested, the payments system shall create a payment session.",
          "status": null
        },
        {
          "tag": "req requirements.REQ_PAY_002",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 8,
            "column": 13
          },
          "name": "requirements.REQ_PAY_002",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.integration_fraud_gate_test_rs"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "Where fraud check is enabled, when payment authorization is requested, the fraud detection system shall calculate a risk score before approval.",
          "status": null
        },
        {
          "tag": "req requirements.REQ_PAY_003",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 14,
            "column": 13
          },
          "name": "requirements.REQ_PAY_003",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.contract_checkout_decline_response_test_go",
            "act tests.unit_checkout_decline_reason_test_go"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If payment is declined, then the payments system shall present a decline reason to customer.",
          "status": null
        },
        {
          "tag": "req requirements.REQ_PAY_004",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 20,
            "column": 13
          },
          "name": "requirements.REQ_PAY_004",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.integration_fraud_gate_test_rs"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "While checkout is active, when payment authorization is requested and risk score is high and risk threshold is exceeded, the payments system shall place the payment in payment review is pending.",
          "status": null
        },
        {
          "tag": "req requirements.REQ_PAY_005",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 26,
            "column": 13
          },
          "name": "requirements.REQ_PAY_005",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.integration_audit_record_persistence_test_ts"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "Where fraud check is enabled, when payment authorization is requested, the payments system shall persist an audit record with payment id plus risk score.",
          "status": null
        },
        {
          "tag": "req requirements.REQ_PAY_006",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 32,
            "column": 13
          },
          "name": "requirements.REQ_PAY_006",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.contract_checkout_decline_response_test_go"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If bank link is unavailable, then the payments system shall present a temporary unavailable message to customer until bank link is available.",
          "status": null
        },
        {
          "tag": "req requirements.REQ_PAY_007",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 38,
            "column": 13
          },
          "name": "requirements.REQ_PAY_007",
          "messages": [
            "missing reference to Tests"
          ],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "tracing_status": "MISSING",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "While checkout is active, when payment authorization is requested and 3DS verification is required, the payments system shall request authorization from bank gateway endpoint.",
          "status": null
        },
        {
          "tag": "req requirements.REQ_PAY_008",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 44,
            "column": 13
          },
          "name": "requirements.REQ_PAY_008",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [],
          "ref_down": [
            "act tests.unit_checkout_decline_reason_test_go"
          ],
          "tracing_status": "OK",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "If payment is declined, then the payments system shall present a retry option to customer.",
          "status": null
        },
        {
          "tag": "req requirements.REQ_PAY_009",
          "location": {
            "kind": "file",
            "file": "examples/payments-engineering-sample/generated/trlc/requirements.trlc",
            "line": 50,
            "column": 13
          },
          "name": "requirements.REQ_PAY_009",
          "messages": [
            "missing reference to Tests"
          ],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "tracing_status": "MISSING",
          "framework": "TRLC",
          "kind": "Requirement",
          "text": "While checkout is active, when payment authorization is requested and risk threshold is exceeded, the payments system shall notify platform operator when support agent is notified.",
          "status": null
        }
      ],
      "coverage": 77.77777777777777
    },
    {
      "name": "Tests",
      "kind": "activity",
      "items": [
        {
          "tag": "act tests.contract_checkout_decline_response_test_go",
          "location": {
            "kind": "file",
            "file": "contract/checkout_decline_response_test.go",
            "line": 6,
            "column": 4
          },
          "name": "tests.contract_checkout_decline_response_test_go",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req requirements.REQ_PAY_006",
            "req requirements.REQ_PAY_003"
          ],
          "ref_down": [],
          "tracing_status": "OK",
          "framework": "Tests",
          "kind": "test",
          "status": "ok"
        },
        {
          "tag": "act tests.e2e_authorized_checkout_flow_yaml",
          "location": {
            "kind": "file",
            "file": "e2e/authorized_checkout_flow.yaml",
            "line": 2,
            "column": 3
          },
          "name": "tests.e2e_authorized_checkout_flow_yaml",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req requirements.REQ_PAY_001"
          ],
          "ref_down": [],
          "tracing_status": "OK",
          "framework": "Tests",
          "kind": "test",
          "status": "ok"
        },
        {
          "tag": "act tests.integration_audit_record_persistence_test_ts",
          "location": {
            "kind": "file",
            "file": "integration/audit_record_persistence_test.ts",
            "line": 4,
            "column": 4
          },
          "name": "tests.integration_audit_record_persistence_test_ts",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req requirements.REQ_PAY_005"
          ],
          "ref_down": [],
          "tracing_status": "OK",
          "framework": "Tests",
          "kind": "test",
          "status": "ok"
        },
        {
          "tag": "act tests.integration_fraud_gate_test_rs",
          "location": {
            "kind": "file",
            "file": "integration/fraud_gate_test.rs",
            "line": 4,
            "column": 4
          },
          "name": "tests.integration_fraud_gate_test_rs",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req requirements.REQ_PAY_004",
            "req requirements.REQ_PAY_002"
          ],
          "ref_down": [],
          "tracing_status": "OK",
          "framework": "Tests",
          "kind": "test",
          "status": "ok"
        },
        {
          "tag": "act tests.unit_checkout_decline_reason_test_go",
          "location": {
            "kind": "file",
            "file": "unit/checkout_decline_reason_test.go",
            "line": 6,
            "column": 4
          },
          "name": "tests.unit_checkout_decline_reason_test_go",
          "messages": [],
          "just_up": [],
          "just_down": [],
          "just_global": [],
          "ref_up": [
            "req requirements.REQ_PAY_008",
            "req requirements.REQ_PAY_003"
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
          "file": "examples/payments-engineering-sample/generated/lobster/requirements.lobster"
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
          "file": "examples/payments-engineering-sample/generated/lobster/activities.lobster"
        }
      ],
      "needs_tracing_up": true,
      "needs_tracing_down": false,
      "breakdown_requirements": []
    }
  },
  "matrix": []
}
