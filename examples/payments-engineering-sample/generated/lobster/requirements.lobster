{
  "data": [
    {
      "tag": "req PaymentsRequirements.REQ_PAY_001",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 2,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_001",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "While normal mode is active, when payment authorization is requested, the payments system shall create a payment session.",
      "status": null
    },
    {
      "tag": "req PaymentsRequirements.REQ_PAY_002",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 8,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_002",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "Where fraud check is enabled, when payment authorization is requested, the fraud detection system shall calculate a risk score before approval.",
      "status": null
    },
    {
      "tag": "req PaymentsRequirements.REQ_PAY_003",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 14,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_003",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "If payment is declined, then the payments system shall present a decline reason to customer.",
      "status": null
    },
    {
      "tag": "req PaymentsRequirements.REQ_PAY_004",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 20,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_004",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "While checkout is active, when payment authorization is requested and risk score is high and risk threshold is exceeded, the payments system shall place the payment in payment review is pending.",
      "status": null
    },
    {
      "tag": "req PaymentsRequirements.REQ_PAY_005",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 26,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_005",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "Where fraud check is enabled, when payment authorization is requested, the payments system shall persist an audit record with payment id plus risk score.",
      "status": null
    },
    {
      "tag": "req PaymentsRequirements.REQ_PAY_006",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 32,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_006",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "If bank link is unavailable, then the payments system shall present a temporary unavailable message to customer until bank link is available.",
      "status": null
    },
    {
      "tag": "req PaymentsRequirements.REQ_PAY_007",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 38,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_007",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "While checkout is active, when payment authorization is requested and 3DS verification is required, the payments system shall request authorization from bank gateway endpoint.",
      "status": null
    },
    {
      "tag": "req PaymentsRequirements.REQ_PAY_008",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 44,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_008",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "If payment is declined, then the payments system shall present a retry option to customer.",
      "status": null
    },
    {
      "tag": "req PaymentsRequirements.REQ_PAY_009",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/payments-engineering-sample/generated/trlc/requirements.trlc",
        "line": 50,
        "column": 13
      },
      "name": "PaymentsRequirements.REQ_PAY_009",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "While checkout is active, when payment authorization is requested and risk threshold is exceeded, the payments system shall notify platform operator when support agent is notified.",
      "status": null
    }
  ],
  "generator": "lobster-trlc",
  "schema": "lobster-req-trace",
  "version": 4
}
