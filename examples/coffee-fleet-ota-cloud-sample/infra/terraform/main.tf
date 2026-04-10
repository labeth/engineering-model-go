terraform {
  required_version = ">= 1.5.0"
}

# engmodel:runtime-description: receives device telemetry, validates payload shape, and publishes normalized fleet events
resource "aws_lambda_function" "fleet_ingestion_api" {
  function_name = "coffee-fleet-ingestion-api"
  role          = "arn:aws:iam::123456789012:role/coffee-fleet-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

# engmodel:runtime-description: coordinates OTA campaign rollout windows and device cohort targeting
resource "aws_lambda_function" "update_campaign_orchestration" {
  function_name = "coffee-update-campaign-orchestration"
  role          = "arn:aws:iam::123456789012:role/coffee-fleet-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

# engmodel:runtime-description: aggregates telemetry outcomes into fleet operations and security reporting signals
resource "aws_lambda_function" "fleet_observability_reporting" {
  function_name = "coffee-fleet-observability-reporting"
  role          = "arn:aws:iam::123456789012:role/coffee-fleet-lambda-role"
  runtime       = "provided.al2"
  handler       = "bootstrap"
}

# engmodel:runtime-description: schedules OTA campaign orchestration triggers for configured rollout windows
resource "aws_cloudwatch_event_rule" "ota_campaign_schedule" {
  name        = "coffee-ota-campaign-schedule"
  description = "Triggers OTA campaign orchestration windows"
}

# engmodel:runtime-description: routes scheduled OTA campaign events to the orchestration runtime handler
resource "aws_cloudwatch_event_target" "ota_campaign_target" {
  rule = aws_cloudwatch_event_rule.ota_campaign_schedule.name
  arn  = aws_lambda_function.update_campaign_orchestration.arn
}

# engmodel:runtime-description: broadcasts fleet-wide telemetry and OTA lifecycle events to subscribed processors
resource "aws_sns_topic" "fleet_event_topic" {
  name = "coffee-fleet-events"
}

# engmodel:runtime-description: buffers telemetry during ingest outages for retry once normal mode resumes
resource "aws_sqs_queue" "telemetry_retry_queue" {
  name = "coffee-telemetry-retry"
}
