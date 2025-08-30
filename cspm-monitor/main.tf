# =============================================================================
# AWS CSPM Monitor Infrastructure
# =============================================================================
#
# This Terraform configuration deploys a comprehensive Cloud Security Posture
# Management (CSPM) monitoring solution with the following components:
#
# - AWS Security Hub integration for security findings
# - DynamoDB for findings storage with encryption and backup
# - Lambda functions for data processing and API endpoints
# - API Gateway for REST API access
# - CloudFront for secure web dashboard delivery
# - EventBridge for event-driven processing
# - SNS for security alerts
# - CloudWatch for monitoring and logging
#
# Enterprise Standards Implemented:
# - Least privilege IAM policies with resource-level restrictions
# - Server-side encryption for all data stores
# - Comprehensive tagging for cost allocation and resource management
# - Conditional resource deployment for flexibility
# - Monitoring and alerting for operational visibility
# - Proper error handling and validation
# - Security best practices (encryption, access controls, logging)
#
# =============================================================================

# Data sources for dynamic configuration
data "aws_caller_identity" "current" {}
data "aws_region" "current" {}

locals {
  account_id = data.aws_caller_identity.current.account_id
  region     = data.aws_region.current.name

  common_tags = {
    Environment = "development"
    Project     = var.project_name
    ManagedBy   = "Terraform"
    Owner       = "Security Team"
    CostCenter  = "Security"
    Backup      = "Daily"
    DataClass   = "Internal"
  }

  tags = merge(local.common_tags, {
    Name = "${var.project_name}-resources"
  })
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
  billing_mode = var.dynamodb_billing_mode
  hash_key     = "id"

  attribute {
    name = "id"
    type = "S"
  }

  attribute {
    name = "severity"
    type = "S"
  }

  attribute {
    name = "timestamp"
    type = "S"
  }

  # Global Secondary Index for severity-based queries
  global_secondary_index {
    name            = "SeverityTimestampIndex"
    hash_key        = "severity"
    range_key       = "timestamp"
    projection_type = "ALL"

    # Add capacity settings for PROVISIONED billing mode
    dynamic "read_capacity" {
      for_each = var.dynamodb_billing_mode == "PROVISIONED" ? [1] : []
      content {
        read_capacity_units = 5
      }
    }

    dynamic "write_capacity" {
      for_each = var.dynamodb_billing_mode == "PROVISIONED" ? [1] : []
      content {
        write_capacity_units = 5
      }
    }
  }

  # Enable server-side encryption
  server_side_encryption {
    enabled = true
  }

  # Enable point-in-time recovery for data protection
  point_in_time_recovery {
    enabled = true
  }

  # Configure deletion protection
  deletion_protection_enabled = true

  # Enable Time-to-Live for automatic data expiration
  dynamic "ttl" {
    for_each = var.dynamodb_ttl_enabled ? [1] : []
    content {
      attribute_name = "ttl_timestamp"
      enabled        = true
    }
  }

  # Resource policy for enhanced security
  resource_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "DenyInsecureTransport"
        Effect = "Deny"
        Principal = "*"
        Action   = "dynamodb:*"
        Resource = [
          aws_dynamodb_table.findings.arn,
          "${aws_dynamodb_table.findings.arn}/index/*"
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      }
    ]
  })

  tags = merge(local.tags, {
    Name            = "${var.project_name}-findings-table"
    DataClass       = "Sensitive"
    Encryption      = "AES256"
    Compliance      = var.compliance_framework
    DataRetention   = "${var.dynamodb_ttl_days}days"
    BackupRetention = var.enable_backup ? "${var.backup_retention_days}days" : "Disabled"
  })

  lifecycle {
    prevent_destroy = true
  }
}

# S3 bucket for security log archival
resource "aws_s3_bucket" "security_archive" {
  count  = var.enable_s3_archival ? 1 : 0
  bucket = "${var.project_name}-security-archive-${local.account_id}"

  tags = merge(local.tags, {
    Name       = "${var.project_name}-security-archive"
    Purpose    = "SecurityLogArchival"
    Retention  = "${var.s3_archive_retention_days}days"
    Compliance = var.compliance_framework
  })
}

resource "aws_s3_bucket_versioning" "security_archive" {
  count  = var.enable_s3_archival ? 1 : 0
  bucket = aws_s3_bucket.security_archive[0].id
  versioning_configuration {
    status = "Enabled"
  }
}

# S3 server-side encryption
resource "aws_s3_bucket_server_side_encryption_configuration" "security_archive" {
  count  = var.enable_s3_archival ? 1 : 0
  bucket = aws_s3_bucket.security_archive[0].id

  rule {
    apply_server_side_encryption_by_default {
      sse_algorithm = "AES256"
    }
    bucket_key_enabled = true
  }
}

# S3 lifecycle configuration for compliance retention
resource "aws_s3_bucket_lifecycle_configuration" "security_archive" {
  count  = var.enable_s3_archival ? 1 : 0
  bucket = aws_s3_bucket.security_archive[0].id

  rule {
    id     = "security_logs_retention"
    status = "Enabled"

    # Move to IA after 30 days
    transition {
      days          = 30
      storage_class = "STANDARD_IA"
    }

    # Move to Glacier after 90 days
    transition {
      days          = 90
      storage_class = "GLACIER"
    }

    # Move to Deep Archive after 1 year
    transition {
      days          = 365
      storage_class = "DEEP_ARCHIVE"
    }

    # Delete after retention period
    expiration {
      days = var.s3_archive_retention_days
    }

    # Clean up incomplete multipart uploads
    abort_incomplete_multipart_upload {
      days_after_initiation = 7
    }
  }
}

# S3 bucket policy for compliance
resource "aws_s3_bucket_policy" "security_archive" {
  count  = var.enable_s3_archival ? 1 : 0
  bucket = aws_s3_bucket.security_archive[0].id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid       = "DenyInsecureTransport"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:*"
        Resource = [
          aws_s3_bucket.security_archive[0].arn,
          "${aws_s3_bucket.security_archive[0].arn}/*"
        ]
        Condition = {
          Bool = {
            "aws:SecureTransport" = "false"
          }
        }
      },
      {
        Sid       = "DenyDeleteWithoutMFA"
        Effect    = "Deny"
        Principal = "*"
        Action    = "s3:DeleteObject"
        Resource = [
          aws_s3_bucket.security_archive[0].arn,
          "${aws_s3_bucket.security_archive[0].arn}/*"
        ]
        Condition = {
          StringNotEquals = {
            "aws:MultiFactorAuthAge" = "0"
          }
        }
      }
    ]
  })
}

# S3 bucket public access block for compliance
resource "aws_s3_bucket_public_access_block" "security_archive" {
  count                   = var.enable_s3_archival ? 1 : 0
  bucket                  = aws_s3_bucket.security_archive[0].id
  block_public_acls       = true
  block_public_policy     = true
  ignore_public_acls      = true
  restrict_public_buckets = true
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
  standards_arn = "arn:aws:securityhub:${var.region}::standards/cis-aws-foundations-benchmark/v/1.4.0"
}

# IAM policy for Lambda with least privilege
resource "aws_iam_role_policy" "lambda_policy" {
  name = "${var.project_name}-lambda-policy"
  role = aws_iam_role.lambda_role.id

  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Sid    = "DynamoDBAccess"
        Effect = "Allow"
        Action = [
          "dynamodb:PutItem",
          "dynamodb:Query",
          "dynamodb:Scan",
          "dynamodb:GetItem",
          "dynamodb:UpdateItem",
          "dynamodb:BatchWriteItem",
          "dynamodb:DescribeTable"
        ]
        Resource = [
          aws_dynamodb_table.findings.arn,
          "${aws_dynamodb_table.findings.arn}/index/*"
        ]
      },
      {
        Sid    = "SecurityHubReadOnly"
        Effect = "Allow"
        Action = [
          "securityhub:GetFindings",
          "securityhub:GetInsightResults",
          "securityhub:ListFindings"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:RequestedRegion" = local.region
          }
        }
      },
      {
        Sid    = "SecurityHubUpdate"
        Effect = "Allow"
        Action = [
          "securityhub:BatchUpdateFindings"
        ]
        Resource = "arn:aws:securityhub:${local.region}:${local.account_id}:finding/*"
      },
      {
        Sid    = "SNSPublish"
        Effect = "Allow"
        Action = [
          "sns:Publish"
        ]
        Resource = aws_sns_topic.alerts.arn
      },
      {
        Sid    = "CloudWatchLogs"
        Effect = "Allow"
        Action = [
          "logs:CreateLogGroup",
          "logs:CreateLogStream",
          "logs:PutLogEvents"
        ]
        Resource = "arn:aws:logs:${local.region}:${local.account_id}:log-group:/aws/lambda/${var.project_name}-*:*"
      },
      {
        Sid    = "SSMParameterAccess"
        Effect = "Allow"
        Action = [
          "ssm:GetParameter",
          "ssm:GetParameters",
          "ssm:GetParametersByPath"
        ]
        Resource = "arn:aws:ssm:${local.region}:${local.account_id}:parameter/${var.project_name}/*"
      },
      {
        Sid    = "S3ArchivalAccess"
        Effect = "Allow"
        Action = [
          "s3:PutObject",
          "s3:GetObject",
          "s3:DeleteObject",
          "s3:ListBucket"
        ]
        Resource = var.enable_s3_archival ? [
          aws_s3_bucket.security_archive[0].arn,
          "${aws_s3_bucket.security_archive[0].arn}/*"
        ] : []
      },
      {
        Sid    = "KMSEncryption"
        Effect = "Allow"
        Action = [
          "kms:GenerateDataKey",
          "kms:Decrypt",
          "kms:DescribeKey"
        ]
        Resource = "*"
        Condition = {
          StringEquals = {
            "aws:RequestedRegion" = local.region
          }
        }
      }
    ]
  })
}

# Create Lambda deployment packages
data "archive_file" "scanner_lambda_zip" {
  type        = "zip"
  source_dir  = "lambda-src"
  output_path = "${path.module}/lambda-src/scanner.zip"
  excludes    = ["api.zip", "archiver.zip", "*.pyc", "__pycache__"]
}

data "archive_file" "api_lambda_zip" {
  type        = "zip"
  source_dir  = "lambda-src"
  output_path = "${path.module}/lambda-src/api.zip"
  excludes    = ["scanner.zip", "archiver.zip", "*.pyc", "__pycache__"]
}

data "archive_file" "archiver_lambda_zip" {
  type        = "zip"
  source_dir  = "lambda-src"
  output_path = "${path.module}/lambda-src/archiver.zip"
  excludes    = ["scanner.zip", "api.zip", "*.pyc", "__pycache__"]
}

# Lambda function for scanning
resource "aws_lambda_function" "scanner" {
  depends_on = [
    aws_iam_role_policy.lambda_policy,
    aws_cloudwatch_log_group.scanner_logs,
    aws_vpc.lambda_vpc,
    aws_security_group.lambda_sg
  ]

  function_name    = "${var.project_name}-scanner"
  runtime          = var.lambda_runtime
  handler          = "scanner.lambda_handler"
  role             = aws_iam_role.lambda_role.arn
  filename         = data.archive_file.scanner_lambda_zip.output_path
  source_code_hash = data.archive_file.scanner_lambda_zip.output_base64sha256

  memory_size = 256
  timeout     = 300

  # VPC configuration for enhanced security
  vpc_config {
    subnet_ids         = aws_subnet.lambda_subnet[*].id
    security_group_ids = [aws_security_group.lambda_sg.id]
  }

  environment {
    variables = {
      DYNAMODB_TABLE_PARAM = "/${var.project_name}/dynamodb-table-name"
      SNS_TOPIC_ARN_PARAM  = "/${var.project_name}/sns-topic-arn"
      DYNAMODB_TTL_DAYS    = var.dynamodb_ttl_days
    }
  }
  tags = local.tags
}

# Lambda function for API
resource "aws_lambda_function" "api" {
  depends_on = [
    aws_iam_role_policy.lambda_policy,
    aws_cloudwatch_log_group.api_logs,
    aws_vpc.lambda_vpc,
    aws_security_group.lambda_sg
  ]

  function_name    = "${var.project_name}-api"
  runtime          = var.lambda_runtime
  handler          = "api.lambda_handler"
  role             = aws_iam_role.lambda_role.arn
  filename         = data.archive_file.api_lambda_zip.output_path
  source_code_hash = data.archive_file.api_lambda_zip.output_base64sha256

  memory_size = 256
  timeout     = 30

  # VPC configuration for enhanced security
  vpc_config {
    subnet_ids         = aws_subnet.lambda_subnet[*].id
    security_group_ids = [aws_security_group.lambda_sg.id]
  }

  environment {
    variables = {
      DYNAMODB_TABLE_PARAM = "/${var.project_name}/dynamodb-table-name"
    }
  }
  tags = local.tags
}

# Lambda function for data archival
resource "aws_lambda_function" "archiver" {
  count = var.enable_s3_archival ? 1 : 0

  depends_on = [
    aws_iam_role_policy.lambda_policy,
    aws_vpc.lambda_vpc,
    aws_security_group.lambda_sg,
    aws_s3_bucket.security_archive
  ]

  function_name    = "${var.project_name}-archiver"
  runtime          = var.lambda_runtime
  handler          = "archiver.lambda_handler"
  role             = aws_iam_role.lambda_role.arn
  filename         = data.archive_file.archiver_lambda_zip.output_path
  source_code_hash = data.archive_file.archiver_lambda_zip.output_base64sha256

  memory_size = 512
  timeout     = 900 # 15 minutes for archival operations

  # VPC configuration for enhanced security
  vpc_config {
    subnet_ids         = module.vpc.subnet_ids
    security_group_ids = [aws_security_group.lambda_sg.id]
  }

  environment {
    variables = merge({
      DYNAMODB_TABLE_PARAM = "/${var.project_name}/dynamodb-table-name"
      RETENTION_DAYS       = var.dynamodb_ttl_days
    }, var.enable_s3_archival ? {
      S3_ARCHIVE_BUCKET = aws_s3_bucket.security_archive[0].bucket
    } : {})
  }

  tags = merge(local.tags, {
    Name = "${var.project_name}-archiver"
  })
}

# VPC for Lambda functions (enhanced security) - using module
module "vpc" {
  source       = "./modules/vpc"
  vpc_cidr     = "10.0.0.0/16"
  subnet_count = 2
  project_name = var.project_name
  tags         = local.tags
}

data "aws_availability_zones" "available" {
  state = "available"
}

# Security group for Lambda functions (VPC deployment for enhanced security)
resource "aws_security_group" "lambda_sg" {
  name_prefix = "${var.project_name}-lambda-"
  description = "Security group for Lambda functions"
  vpc_id      = module.vpc.vpc_id

  # Allow HTTPS outbound for AWS API calls
  egress {
    from_port   = 443
    to_port     = 443
    protocol    = "tcp"
    cidr_blocks = ["0.0.0.0/0"]
    description = "HTTPS outbound for AWS API calls"
  }

  # Allow DNS resolution (restricted to AWS DNS servers only)
  egress {
    from_port   = 53
    to_port     = 53
    protocol    = "tcp"
    cidr_blocks = ["169.254.169.253/32"] # AWS DNS only
    description = "DNS resolution (TCP) - AWS DNS only"
  }

  egress {
    from_port   = 53
    to_port     = 53
    protocol    = "udp"
    cidr_blocks = ["169.254.169.253/32"] # AWS DNS only
    description = "DNS resolution (UDP) - AWS DNS only"
  }

  tags = merge(local.tags, {
    Name = "${var.project_name}-lambda-sg"
  })
}

# API Gateway with enhanced security
resource "aws_api_gateway_rest_api" "api" {
  name        = "${var.project_name}-api"
  description = "API for CSPM dashboard"

  # Enable API Gateway logging and security
  endpoint_configuration {
    types = ["REGIONAL"]
  }

  # Add minimum compression size
  minimum_compression_size = 1024
}

resource "aws_api_gateway_resource" "findings" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "findings"
}

# Health check endpoint
resource "aws_api_gateway_resource" "health" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  parent_id   = aws_api_gateway_rest_api.api.root_resource_id
  path_part   = "health"
}

resource "aws_api_gateway_method" "get_findings" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.findings.id
  http_method   = "GET"
  authorization = "NONE"
}

# Health check method
resource "aws_api_gateway_method" "get_health" {
  rest_api_id   = aws_api_gateway_rest_api.api.id
  resource_id   = aws_api_gateway_resource.health.id
  http_method   = "GET"
  authorization = "NONE"
}

resource "aws_api_gateway_method_response" "get_health" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.health.id
  http_method = aws_api_gateway_method.get_health.http_method
  status_code = "200"

  response_models = {
    "application/json" = "Empty"
  }
}

# Add method response for get_findings
resource "aws_api_gateway_method_response" "get_findings" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.findings.id
  http_method = aws_api_gateway_method.get_findings.http_method
  status_code = "200"

  response_models = {
    "application/json" = "Empty"
  }
}

resource "aws_api_gateway_integration" "get_findings" {
  rest_api_id             = aws_api_gateway_rest_api.api.id
  resource_id             = aws_api_gateway_resource.findings.id
  http_method             = aws_api_gateway_method.get_findings.http_method
  integration_http_method = "POST"
  type                    = "AWS_PROXY"
  uri                     = aws_lambda_function.api.invoke_arn
}

# Health check integration (mock response for monitoring)
resource "aws_api_gateway_integration" "get_health" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.health.id
  http_method = aws_api_gateway_method.get_health.http_method
  type        = "MOCK"

  request_templates = {
    "application/json" = jsonencode({
      statusCode = 200
    })
  }
}

resource "aws_api_gateway_integration_response" "get_health" {
  rest_api_id = aws_api_gateway_rest_api.api.id
  resource_id = aws_api_gateway_resource.health.id
  http_method = aws_api_gateway_method.get_health.http_method
  status_code = "200"

  # Add comprehensive security headers
  response_parameters = {
    "method.response.header.X-Content-Type-Options"     = "'nosniff'"
    "method.response.header.X-Frame-Options"            = "'DENY'"
    "method.response.header.X-XSS-Protection"           = "'1; mode=block'"
    "method.response.header.Strict-Transport-Security"  = "'max-age=31536000; includeSubDomains; preload'"
    "method.response.header.Content-Security-Policy"    = "'default-src 'self'; script-src 'self'; style-src 'self' 'unsafe-inline'; img-src 'self' data: https:; font-src 'self' data:; connect-src 'self''"
    "method.response.header.Referrer-Policy"            = "'strict-origin-when-cross-origin'"
    "method.response.header.Permissions-Policy"         = "'geolocation=(), microphone=(), camera=()'"
    "method.response.header.Cross-Origin-Embedder-Policy" = "'require-corp'"
    "method.response.header.Cross-Origin-Opener-Policy"   = "'same-origin'"
    "method.response.header.Cross-Origin-Resource-Policy" = "'same-origin'"
  }

  response_templates = {
    "application/json" = jsonencode({
      status     = "healthy"
      service    = "cspm-monitor-api"
      timestamp  = "$context.requestTime"
      request_id = "$context.requestId"
      version    = "1.0.0"
    })
  }
}

resource "aws_lambda_permission" "api_gateway" {
  statement_id  = "AllowAPIGatewayInvoke"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.api.function_name
  principal     = "apigateway.amazonaws.com"
  source_arn    = "${aws_api_gateway_rest_api.api.execution_arn}/*/*"
}

resource "aws_api_gateway_deployment" "api" {
  depends_on = [
    aws_api_gateway_integration.get_findings,
    aws_api_gateway_integration.get_health,
    aws_api_gateway_method.get_findings,
    aws_api_gateway_method.get_health,
    aws_api_gateway_resource.findings,
    aws_api_gateway_resource.health,
    aws_api_gateway_integration_response.get_health,
    aws_api_gateway_method_response.get_health
  ]
  rest_api_id = aws_api_gateway_rest_api.api.id
}

resource "aws_api_gateway_stage" "prod" {
  deployment_id = aws_api_gateway_deployment.api.id
  rest_api_id   = aws_api_gateway_rest_api.api.id
  stage_name    = "prod"

  # Enable caching for performance
  cache_cluster_enabled = true
  cache_cluster_size    = "0.5"

  # Enable access logging
  access_log_settings {
    destination_arn = aws_cloudwatch_log_group.api_gateway_logs.arn
    format = jsonencode({
      requestId      = "$context.requestId"
      ip            = "$context.identity.sourceIp"
      requestTime   = "$context.requestTime"
      httpMethod    = "$context.httpMethod"
      resourcePath  = "$context.resourcePath"
      status        = "$context.status"
      responseLength = "$context.responseLength"
      userAgent     = "$context.identity.userAgent"
    })
  }

  tags = local.tags
}

# CloudWatch Log Group for API Gateway
resource "aws_cloudwatch_log_group" "api_gateway_logs" {
  name              = "/aws/apigateway/${var.project_name}-api"
  retention_in_days = 30
  tags              = local.tags
}

# WAF Web ACL for API protection
resource "aws_wafv2_web_acl" "api_waf" {
  name  = "${var.project_name}-api-waf"
  scope = "REGIONAL"

  default_action {
    allow {}
  }

  # AWS Managed Rules - Common Rule Set
  rule {
    name     = "AWSManagedRulesCommonRuleSet"
    priority = 1

    override_action {
      none {}
    }

    statement {
      managed_rule_group_statement {
        vendor_name = "AWS"
        name        = "AWSManagedRulesCommonRuleSet"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name               = "${var.project_name}-waf-common"
      sampled_requests_enabled  = true
    }
  }

  # Rate limiting rule
  rule {
    name     = "RateLimit"
    priority = 2

    action {
      block {}
    }

    statement {
      rate_based_statement {
        limit              = 2000
        aggregate_key_type = "IP"
      }
    }

    visibility_config {
      cloudwatch_metrics_enabled = true
      metric_name               = "${var.project_name}-waf-rate-limit"
      sampled_requests_enabled  = true
    }
  }

  visibility_config {
    cloudwatch_metrics_enabled = true
    metric_name               = "${var.project_name}-waf-overall"
    sampled_requests_enabled  = true
  }

  tags = local.tags
}

# Associate WAF with API Gateway
resource "aws_wafv2_web_acl_association" "api_waf" {
  depends_on = [
    aws_api_gateway_stage.prod,
    aws_wafv2_web_acl.api_waf
  ]

  resource_arn = aws_api_gateway_stage.prod.arn
  web_acl_arn  = aws_wafv2_web_acl.api_waf.arn
}

# API Gateway Usage Plan and Throttling
resource "aws_api_gateway_usage_plan" "main" {
  depends_on = [aws_api_gateway_stage.prod]

  name = "${var.project_name}-usage-plan"

  throttle_settings {
    burst_limit = 200 # Allow bursts for dashboard loads
    rate_limit  = 100 # 100 requests per second sustained
  }

  quota_settings {
    limit  = 50000 # 50K requests per day
    period = "DAY"
  }

  api_stages {
    api_id = aws_api_gateway_rest_api.api.id
    stage  = aws_api_gateway_stage.prod.stage_name
  }

  tags = local.tags
}

# SNS topic for alerts
resource "aws_sns_topic" "alerts" {
  name = "${var.project_name}-alerts"
  tags = local.tags
}

# AWS Systems Manager Parameter Store for secrets
resource "aws_ssm_parameter" "sns_topic_arn" {
  name  = "/${var.project_name}/sns-topic-arn"
  type  = "SecureString"
  value = aws_sns_topic.alerts.arn
  tags  = local.tags
}

resource "aws_ssm_parameter" "dynamodb_table_name" {
  name  = "/${var.project_name}/dynamodb-table-name"
  type  = "SecureString"
  value = aws_dynamodb_table.findings.name
  tags  = local.tags
}

# EventBridge rule for Security Hub findings
resource "aws_cloudwatch_event_rule" "security_hub_findings" {
  name        = "${var.project_name}-security-hub-findings"
  description = "Trigger on Security Hub findings"
  event_pattern = jsonencode({
    source      = ["aws.securityhub"]
    detail-type = ["Security Hub Findings - Imported", "Security Hub Findings - Custom Action"]
    detail = {
      findings = {
        Severity = {
          Label = ["CRITICAL", "HIGH", "MEDIUM"] # Focus on important findings
        }
      }
    }
  })
  tags = local.tags
}

# SQS DLQ for EventBridge
resource "aws_sqs_queue" "eventbridge_dlq" {
  name = "${var.project_name}-eventbridge-dlq"
  tags = local.tags
}

resource "aws_cloudwatch_event_target" "security_hub_target" {
  rule      = aws_cloudwatch_event_rule.security_hub_findings.name
  target_id = "lambda"
  arn       = aws_lambda_function.scanner.arn

  retry_policy {
    maximum_retry_attempts       = 3
    maximum_event_age_in_seconds = 86400 # 24 hours
  }

  dead_letter_config {
    arn = aws_sqs_queue.eventbridge_dlq.arn
  }

  depends_on = [aws_lambda_permission.allow_eventbridge]
}

resource "aws_lambda_permission" "allow_eventbridge" {
  statement_id  = "AllowExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.scanner.function_name
  principal     = "events.amazonaws.com"
  source_arn    = "arn:aws:events:${local.region}:${local.account_id}:rule/${var.project_name}-*"
}

# Optional: Scheduled rule for periodic sync of all findings
resource "aws_cloudwatch_event_rule" "sync_schedule" {
  count               = var.enable_sync_schedule ? 1 : 0
  name                = "${var.project_name}-sync-schedule"
  description         = "Periodic sync of all Security Hub findings"
  schedule_expression = var.sync_schedule_rate
  tags                = local.tags
}

resource "aws_cloudwatch_event_target" "sync_target" {
  count     = var.enable_sync_schedule ? 1 : 0
  rule      = aws_cloudwatch_event_rule.sync_schedule[0].name
  target_id = "lambda-sync"
  arn       = aws_lambda_function.scanner.arn
}

resource "aws_lambda_permission" "allow_sync_eventbridge" {
  count         = var.enable_sync_schedule ? 1 : 0
  statement_id  = "AllowSyncExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.scanner.function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.sync_schedule[0].arn
}

# Scheduled rule for data archival
resource "aws_cloudwatch_event_rule" "archival_schedule" {
  count               = var.enable_s3_archival ? 1 : 0
  name                = "${var.project_name}-archival-schedule"
  description         = "Daily archival of expired security findings"
  schedule_expression = "rate(1 day)"
  tags = merge(local.tags, {
    Purpose = "DataArchival"
  })
}

resource "aws_cloudwatch_event_target" "archival_target" {
  count     = var.enable_s3_archival ? 1 : 0
  rule      = aws_cloudwatch_event_rule.archival_schedule[0].name
  target_id = "lambda-archival"
  arn       = aws_lambda_function.archiver[0].arn

  retry_policy {
    maximum_retry_attempts       = 2
    maximum_event_age_in_seconds = 43200 # 12 hours for archival
  }

  dead_letter_config {
    arn = aws_sqs_queue.eventbridge_dlq.arn
  }

  depends_on = [aws_lambda_permission.allow_archival_eventbridge]
}

resource "aws_lambda_permission" "allow_archival_eventbridge" {
  count         = var.enable_s3_archival ? 1 : 0
  statement_id  = "AllowArchivalExecutionFromEventBridge"
  action        = "lambda:InvokeFunction"
  function_name = aws_lambda_function.archiver[0].function_name
  principal     = "events.amazonaws.com"
  source_arn    = aws_cloudwatch_event_rule.archival_schedule[0].arn
}

# DynamoDB Backup configuration
resource "aws_backup_vault" "security_logs" {
  count = var.enable_backup ? 1 : 0
  name  = "${var.project_name}-security-logs-backup"
  tags = merge(local.tags, {
    Purpose = "SecurityLogBackup"
  })
}

resource "aws_backup_plan" "security_logs" {
  count = var.enable_backup ? 1 : 0
  name  = "${var.project_name}-security-logs-backup-plan"

  rule {
    rule_name         = "${var.project_name}-daily-backup"
    target_vault_name = aws_backup_vault.security_logs[0].name
    schedule          = "cron(0 5 ? * * *)" # Daily at 5 AM

    lifecycle {
      delete_after = var.backup_retention_days
    }
  }

  tags = merge(local.tags, {
    Purpose = "SecurityLogBackup"
  })
}

resource "aws_backup_selection" "security_logs" {
  count        = var.enable_backup ? 1 : 0
  name         = "${var.project_name}-security-logs-selection"
  iam_role_arn = aws_iam_role.backup_role[0].arn
  plan_id      = aws_backup_plan.security_logs[0].id

  resources = [
    aws_dynamodb_table.findings.arn
  ]
}

# IAM role for AWS Backup
resource "aws_iam_role" "backup_role" {
  count = var.enable_backup ? 1 : 0
  name  = "${var.project_name}-backup-role"

  assume_role_policy = jsonencode({
    Version = "2012-10-17"
    Statement = [
      {
        Action = "sts:AssumeRole"
        Effect = "Allow"
        Principal = {
          Service = "backup.amazonaws.com"
        }
      }
    ]
  })

  tags = local.tags
}

resource "aws_iam_role_policy_attachment" "backup_role" {
  count      = var.enable_backup ? 1 : 0
  role       = aws_iam_role.backup_role[0].name
  policy_arn = "arn:aws:iam::aws:policy/service-role/AWSBackupServiceRolePolicyForBackup"
}

# CloudWatch Log Groups for Lambda functions
resource "aws_cloudwatch_log_group" "scanner_logs" {
  name              = "/aws/lambda/${var.project_name}-scanner"
  retention_in_days = 30
  tags              = local.tags
}

resource "aws_cloudwatch_log_group" "api_logs" {
  name              = "/aws/lambda/${var.project_name}-api"
  retention_in_days = 30
  tags              = local.tags
}

# CloudWatch Alarms for Lambda functions
resource "aws_cloudwatch_metric_alarm" "scanner_errors" {
  depends_on = [aws_lambda_function.scanner]

  alarm_name          = "${var.project_name}-scanner-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "3"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "900" # 15 minutes
  statistic           = "Sum"
  threshold           = "5" # Allow some errors before alerting
  alarm_description   = "Alert when Lambda scanner function has excessive errors"
  alarm_actions       = [aws_sns_topic.alerts.arn]

  dimensions = {
    FunctionName = aws_lambda_function.scanner.function_name
  }

  tags = local.tags
}

resource "aws_cloudwatch_metric_alarm" "api_errors" {
  depends_on = [aws_lambda_function.api]

  alarm_name          = "${var.project_name}-api-errors"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "3"
  metric_name         = "Errors"
  namespace           = "AWS/Lambda"
  period              = "900" # 15 minutes
  statistic           = "Sum"
  threshold           = "3" # Allow some errors before alerting
  alarm_description   = "Alert when Lambda API function has excessive errors"

  # Escalation: Send to alerts topic first
  alarm_actions = [aws_sns_topic.alerts.arn]

  dimensions = {
    FunctionName = aws_lambda_function.api.function_name
  }

  tags = local.tags
}

# Critical alert escalation - PagerDuty integration
resource "aws_cloudwatch_metric_alarm" "critical_findings" {
  alarm_name          = "${var.project_name}-critical-findings"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "1"
  metric_name         = "NumberOfMessagesPublished"
  namespace           = "AWS/SNS"
  period              = "300" # 5 minutes
  statistic           = "Sum"
  threshold           = "5" # More than 5 critical alerts in 5 minutes
  alarm_description   = "CRITICAL: Excessive security findings detected"

  # Immediate escalation for critical security issues
  alarm_actions = concat(
    [aws_sns_topic.alerts.arn],
    var.enable_critical_escalation ? [aws_sns_topic.critical_alerts[0].arn] : []
  )

  dimensions = {
    TopicName = aws_sns_topic.alerts.name
  }

  tags = merge(local.tags, {
    Severity    = "Critical"
    Escalation  = "Immediate"
  })
}

# Critical alerts SNS topic (for PagerDuty/ops teams)
resource "aws_sns_topic" "critical_alerts" {
  count = var.enable_critical_escalation ? 1 : 0
  name  = "${var.project_name}-critical-alerts"

  tags = merge(local.tags, {
    Purpose = "CriticalSecurityAlerts"
  })
}

# Auto-scaling alarm for DynamoDB
resource "aws_cloudwatch_metric_alarm" "dynamodb_throttles" {
  depends_on = [aws_dynamodb_table.findings]

  alarm_name          = "${var.project_name}-dynamodb-throttles"
  comparison_operator = "GreaterThanThreshold"
  evaluation_periods  = "2"
  metric_name         = "ThrottledRequests"
  namespace           = "AWS/DynamoDB"
  period              = "300"
  statistic           = "Sum"
  threshold           = "10"
  alarm_description   = "DynamoDB throttling detected - may need capacity increase"
  alarm_actions       = [aws_sns_topic.alerts.arn]

  dimensions = {
    TableName = aws_dynamodb_table.findings.name
  }

  tags = local.tags
}

# Lambda Provisioned Concurrency for performance
resource "aws_lambda_provisioned_concurrency_config" "api" {
  depends_on = [aws_lambda_function.api]

  function_name                     = aws_lambda_function.api.function_name
  provisioned_concurrent_executions = 2
  qualifier                         = "$LATEST"
}

resource "aws_lambda_provisioned_concurrency_config" "scanner" {
  depends_on = [aws_lambda_function.scanner]

  function_name                     = aws_lambda_function.scanner.function_name
  provisioned_concurrent_executions = 1
  qualifier                         = "$LATEST"
}

# CloudFront for website
module "cloudfront" {
  source                        = "../static-website/modules/cloudfront"
  domain_name                   = var.domain_name
  certificate_domain_name       = var.certificate_domain_name
  origin_bucket_regional_domain = module.website_bucket.bucket_regional_domain_name
  response_headers_policy_id    = "" # Use default or create
  waf_web_acl_arn               = "" # Optional
  price_class                   = "PriceClass_100"
  log_bucket_domain             = "" # Optional
  tags                          = local.tags
  origin_shield_region          = var.region
  providers = {
    aws           = aws
    aws.us_east_1 = aws
  }
}

# CloudWatch Query Definitions for log analysis
resource "aws_cloudwatch_query_definition" "error_analysis" {
  name         = "${var.project_name}-error-analysis"
  query_string = <<EOF
fields @timestamp, @message
| filter @message like /ERROR|Exception|Failed/
| sort @timestamp desc
| limit 100
EOF
  log_group_names = [
    aws_cloudwatch_log_group.scanner_logs.name,
    aws_cloudwatch_log_group.api_logs.name
  ]
}

resource "aws_cloudwatch_query_definition" "security_findings" {
  name            = "${var.project_name}-security-findings-analysis"
  query_string    = <<EOF
fields @timestamp, @message
| filter @message like /HIGH|CRITICAL|Security/
| sort @timestamp desc
| limit 50
EOF
  log_group_names = [aws_cloudwatch_log_group.scanner_logs.name]
}

# CloudWatch Dashboard for monitoring
resource "aws_cloudwatch_dashboard" "main" {
  depends_on = [
    aws_lambda_function.scanner,
    aws_lambda_function.api,
    aws_api_gateway_rest_api.api,
    aws_dynamodb_table.findings
  ]

  dashboard_name = "${var.project_name}-dashboard"

  dashboard_body = jsonencode({
    widgets = [
      {
        type   = "metric"
        x      = 0
        y      = 0
        width  = 12
        height = 6

        properties = {
          metrics = concat(
            [
              ["AWS/Lambda", "Errors", "FunctionName", aws_lambda_function.scanner.function_name],
              [".", "Errors", ".", aws_lambda_function.api.function_name]
            ],
            var.enable_s3_archival ? [[".", "Errors", ".", aws_lambda_function.archiver[0].function_name]] : []
          )
          view    = "timeSeries"
          stacked = false
          region  = local.region
          title   = "Lambda Function Errors"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 0
        width  = 12
        height = 6

        properties = {
          metrics = concat(
            [
              ["AWS/Lambda", "Duration", "FunctionName", aws_lambda_function.scanner.function_name],
              [".", "Duration", ".", aws_lambda_function.api.function_name]
            ],
            var.enable_s3_archival ? [[".", "Duration", ".", aws_lambda_function.archiver[0].function_name]] : []
          )
          view    = "timeSeries"
          stacked = false
          region  = local.region
          title   = "Lambda Function Duration"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 0
        y      = 6
        width  = 12
        height = 6

        properties = {
          metrics = [
            ["AWS/ApiGateway", "Count", "ApiName", aws_api_gateway_rest_api.api.name, "Stage", aws_api_gateway_stage.prod.stage_name],
            [".", "4XXError", ".", ".", ".", "."],
            [".", "5XXError", ".", ".", ".", "."]
          ]
          view    = "timeSeries"
          stacked = false
          region  = local.region
          title   = "API Gateway Metrics"
          period  = 300
        }
      },
      {
        type   = "metric"
        x      = 12
        y      = 6
        width  = 12
        height = 6

        properties = {
          metrics = [
            ["AWS/DynamoDB", "ConsumedReadCapacityUnits", "TableName", aws_dynamodb_table.findings.name],
            [".", "ConsumedWriteCapacityUnits", ".", "."]
          ]
          view    = "timeSeries"
          stacked = false
          region  = local.region
          title   = "DynamoDB Capacity Units"
          period  = 300
        }
      }
    ]
  })
}