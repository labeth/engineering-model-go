terraform {
  required_version = ">= 1.5.0"
}

# engmodel:runtime-description: validates GitHub webhook signature and normalizes pull request event payloads
resource "aws_lambda_function" "webhook_ingress" {
  function_name = "pr-review-webhook-ingress"
  role          = "arn:aws:iam::123456789012:role/pr-review-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

# engmodel:runtime-description: assembles diff context and orchestrates deterministic plus Bedrock-backed review analysis
resource "aws_lambda_function" "review_orchestrator" {
  function_name = "pr-review-orchestrator"
  role          = "arn:aws:iam::123456789012:role/pr-review-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

# engmodel:runtime-description: publishes check run summaries and inline review comments back to GitHub
resource "aws_lambda_function" "review_publisher" {
  function_name = "pr-review-publisher"
  role          = "arn:aws:iam::123456789012:role/pr-review-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}
