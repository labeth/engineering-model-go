terraform {
  required_version = ">= 1.5.0"
}

resource "aws_lambda_function" "fleet_ingestion_api" {
  function_name = "coffee-fleet-ingestion-api"
  role          = "arn:aws:iam::123456789012:role/coffee-fleet-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

resource "aws_lambda_function" "update_campaign_orchestration" {
  function_name = "coffee-update-campaign-orchestration"
  role          = "arn:aws:iam::123456789012:role/coffee-fleet-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

resource "aws_lambda_function" "fleet_observability_reporting" {
  function_name = "coffee-fleet-observability-reporting"
  role          = "arn:aws:iam::123456789012:role/coffee-fleet-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

resource "aws_cloudwatch_event_rule" "ota_campaign_schedule" {
  name        = "coffee-ota-campaign-schedule"
  description = "Triggers OTA campaign orchestration windows"
}

resource "aws_cloudwatch_event_target" "ota_campaign_target" {
  rule = aws_cloudwatch_event_rule.ota_campaign_schedule.name
  arn  = aws_lambda_function.update_campaign_orchestration.arn
}

resource "aws_sns_topic" "fleet_event_topic" {
  name = "coffee-fleet-events"
}

resource "aws_sqs_queue" "telemetry_retry_queue" {
  name = "coffee-telemetry-retry"
}
