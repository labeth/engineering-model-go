terraform {
  required_version = ">= 1.5.0"
}

resource "aws_lambda_function" "webhook_ingress" {
  function_name = "pr-review-webhook-ingress"
  role          = "arn:aws:iam::123456789012:role/pr-review-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

resource "aws_lambda_function" "review_orchestrator" {
  function_name = "pr-review-orchestrator"
  role          = "arn:aws:iam::123456789012:role/pr-review-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

resource "aws_lambda_function" "review_publisher" {
  function_name = "pr-review-publisher"
  role          = "arn:aws:iam::123456789012:role/pr-review-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}
