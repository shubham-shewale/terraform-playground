output "api_gateway_url" {
  description = "API Gateway URL for the CSPM API"
  value       = "${aws_api_gateway_deployment.api.invoke_url}${aws_api_gateway_stage.prod.stage_name}"
}

output "website_url" {
  description = "CloudFront URL for the CSPM dashboard"
  value       = module.cloudfront.distribution_domain_name
}

output "dynamodb_table_name" {
  description = "DynamoDB table name for findings"
  value       = aws_dynamodb_table.findings.name
}

output "sns_topic_arn" {
  description = "SNS topic ARN for alerts"
  value       = aws_sns_topic.alerts.arn
}