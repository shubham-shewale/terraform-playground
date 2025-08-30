variable "project_name" {
  description = "Name of the CSPM project"
  type        = string
  default     = "cspm-monitor"
}

variable "region" {
  description = "AWS region"
  type        = string
  default     = "us-east-1"
}

locals {
  tags = {
    Environment = "development"
    Project     = var.project_name
    ManagedBy   = "Terraform"
  }
}

# S3 bucket for website
module "website_bucket" {
  source      = "../static-website/modules/website_bucket"
  bucket_name = "${var.project_name}-website"
  tags        = local.tags
}

# DynamoDB table for findings
resource "aws_dynamodb_table" "findings" {
  name         = "${var.project_name}-findings"
  billing_mode = "PAY_PER_REQUEST"
  hash_key     = "id"
  attribute {
    name = "id"
    type = "S"
  }
  tags = local.tags
}

# IAM role for Lambda
resource "aws_iam_role" "lambda_role" {
  name = "${var.project_name}-lambda-role"
  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{
      Action = "sts:AssumeRole"
      Effect = "Allow"
      Principal = {
        Service = "lambda.amazonaws.com"
      }
    }]
  })
}

# Enable Security Hub
resource "aws_securityhub_account" "main" {}

# Enable CIS AWS Foundations Benchmark
resource "aws_securityhub_standards_subscription" "cis" {
  depends_on    = [aws_securityhub_account.main]
  standards_arn = "arn:aws:securityhub:us-east-1::standards/cis-aws-foundations-benchmark/v/1.4.0"
}

# IAM policy for Lambda
resource "aws_iam_role_policy" "lambda_policy" {
  name = "${var.project_name}-lambda-policy"
  role = aws_iam_role.lambda_role.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:Query",
          "dynamodb:Scan"
        ]
        Resource = aws_dynamodb_table.findings.arn
      },
      {
        Effect = "Allow"
        Action = [
          "securityhub:GetFindings",
          "securityhub:BatchUpdateFindings"
        ]
        Resource = "*"
      },
      {
        Effect = "Allow"
        Action = [
          "sns:Publish"
        ]
        Resource = aws_sns_topic.alerts.arn
      }
    ]
  })
}

# Lambda function for scanning
resource "aws_lambda_function" "scanner" {
  function_name = "${var.project_name}-scanner"
  runtime       = "python3.9"
  handler       = "scanner.lambda_handler"
  role          = aws_iam_role.lambda_role.arn
  filename      = "lambda-src/scanner.zip"
  source_code_hash = filebase64sha256("lambda-src/scanner.zip")
  environment {
    variables = {
      DYNAMODB_TABLE = aws_dynamodb_table.findings.name
      SNS_TOPIC_ARN  = aws_sns_topic.alerts.arn
    }
  }
  tags = local.tags
}

# Lambda function for API
resource "aws_lambda_function" "api" {
  function_name = "${var.project_name}-api"
  runtime       = "python3.9"
  handler       = "api.lambda_handler"
  role          = aws_iam_role.lambda_role.arn
  filename      = "lambda-src/api.zip"
  source_code_hash = filebase64sha256("lambda-src/api.zip")
  environment {
    variables = {
      DYNAMODB_TABLE = aws_dynamodb_table.findings.name
    }
  }
  tags = local.tags
}

# API Gateway
resource "aws_api_gateway_rest_api" "api" {
  name        = "${var.project_name}-api"
  description = "API for CSPM dashboard"
}

resource "aws_api_gateway_resource" "findings" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "findings"
}

resource "aws_api_gateway_method" "get_findings" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.findings.id
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_integration" "get_findings" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.findings.id
  http_method             = aws_api_gateway_method.get_findings.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.api.invoke_arn
}

resource "aws_api_gateway_deployment" "api" {
  depends_on  = [aws_api_gateway_integration.get_findings]
  rest_api_id = aws_api_gateway_rest_api.api.id
}

resource "aws_api_gateway_stage" "prod" {
  deployment_id = aws_api_gateway_deployment.api.id
  rest_api_id   = aws_api_gateway_rest_api.api.id
  stage_name    = "prod"
}

# SNS topic for alerts
resource "aws_sns_topic" "alerts" {
  name = "${var.project_name}-alerts"
  tags = local.tags
}

# EventBridge rule for Security Hub findings
resource "aws_cloudwatch_event_rule" "security_hub_findings" {
  name        = "${var.project_name}-security-hub-findings"
  description = "Trigger on Security Hub findings"
  event_pattern = jsonencode({
    source      = ["aws.securityhub"]
    detail-type = ["Security Hub Findings - Imported", "Security Hub Findings - Custom Action"]
  })
}

resource "aws_cloudwatch_event_target" "security_hub_target" {
  rule      = aws_cloudwatch_event_rule.security_hub_findings.name
  target_id = "lambda"
  arn       = aws_lambda_function.scanner.arn
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.scanner.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.security_hub_findings.arn
}

# Optional: Scheduled rule for periodic sync of all findings
resource "aws_cloudwatch_event_rule" "sync_schedule" {
  name                = "${var.project_name}-sync-schedule"
  description         = "Periodic sync of all Security Hub findings"
  schedule_expression = "rate(6 hours)"
}

resource "aws_cloudwatch_event_target" "sync_target" {
  rule      = aws_cloudwatch_event_rule.sync_schedule.name
  target_id = "lambda-sync"
  arn       = aws_lambda_function.scanner.arn
}

resource "aws_lambda_permission" "allow_sync_eventbridge" {
  statement_id  = "AllowSyncExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.scanner.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.sync_schedule.arn
}

# CloudFront for website
module "cloudfront" {
  source                        = "../static-website/modules/cloudfront"
  domain_name                   = "cspm-monitor.example.com"  # Placeholder
  certificate_domain_name       = "cspm-monitor.example.com"
  origin_bucket_regional_domain = module.website_bucket.bucket_regional_domain_name
  response_headers_policy_id    = ""  # Use default or create
  waf_web_acl_arn               = ""  # Optional
  price_class                   = "PriceClass_100"
  log_bucket_domain             = ""  # Optional
  tags                          = local.tags
  origin_shield_region          = var.region
  providers = {
    aws           = aws
    aws.us_east_1 = aws
  }
}