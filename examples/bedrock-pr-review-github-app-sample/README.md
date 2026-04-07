# Bedrock + Lambda GitHub PR Review App Sample

This sample models an AI-assisted GitHub App that reviews pull requests using AWS Bedrock and Lambda functions.

The architecture is intentionally authored at stable responsibility boundaries while runtime, code, and verification
layers are inferred from infrastructure, source, and test artifacts.

Highlights:

- GitHub webhook ingress verifies signatures and routes review jobs.
- PR context assembly fetches changed files and builds model-ready review context.
- Review orchestration combines Bedrock findings with deterministic policy checks.
- Review publication posts GitHub check runs and inline comments.
- Platform units own Lambda deployment and secrets/config controls.

Inference fixtures included:

- runtime: `infra/terraform`
- code ownership and requirement traces: `src/`
- verification checks and outcomes: `tests/` + `test-results/`

This sample is synthetic and meant for documentation and traceability workflows.
