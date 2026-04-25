{
  "data": [
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_001",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 2,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_001",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "While normal mode is active, when pull request synchronize event is received and webhook signature is valid, the pr review system shall verify webhook signature before processing review workflow.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_002",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 8,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_002",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "While pull request review is active, when pull request opened event is received, the pr review system shall construct diff context from changed files for developer review execution.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_003",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 14,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_003",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "Where ai review is enabled, when diff context is prepared, the pr review system shall request bedrock runtime analysis for review findings.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_004",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 20,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_004",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "Where policy checks are enabled, when pull request review is active, the pr review system shall evaluate deterministic policy checks for secret exposure plus infrastructure risk plus dependency changes.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_005",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 26,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_005",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "While pull request review is active, when review findings are generated, the pr review system shall publish check run summary to repository maintainer.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_006",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 32,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_006",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "If bedrock runtime is unavailable, then the pr review system shall publish deterministic policy-only review outcome.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_007",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 38,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_007",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "Where audit logging is enabled, when review findings are published, the pr review system shall persist redacted audit metadata for security engineer audit traceability.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_008",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 44,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_008",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "If review request rate limit is exceeded, then the pr review system shall defer processing.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_009",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 50,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_009",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "While pull request review is active, when review findings are generated, the pr review system shall publish inline comments with result metadata to repository maintainer.",
      "status": null
    },
    {
      "tag": "req BedrockPRReviewRequirements.REQ_PRR_010",
      "location": {
        "kind": "file",
        "file": "/home/labeth/ws/engineering-model-go/examples/bedrock-pr-review-github-app-sample/generated/trlc/requirements.trlc",
        "line": 56,
        "column": 13
      },
      "name": "BedrockPRReviewRequirements.REQ_PRR_010",
      "messages": [],
      "just_up": [],
      "just_down": [],
      "just_global": [],
      "framework": "TRLC",
      "kind": "Requirement",
      "text": "If review request rate limit is exceeded, then the pr review system shall notify platform operator.",
      "status": null
    }
  ],
  "generator": "lobster-trlc",
  "schema": "lobster-req-trace",
  "version": 4
}
