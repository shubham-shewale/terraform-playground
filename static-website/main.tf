variable "domain_name" {
  description = "The domain name for the website (e.g., example.com)"
  type        = string
}
variable "price_class" {
  type    = string
  default = "PriceClass_100"
}
variable "rate_limit" {
  type    = number
  default = 2000
}
variable "log_lifecycle_days" {
  type    = number
  default = 365
}

locals {
  tags = {
    Environment = "production"
    Project     = "static-website"
    ManagedBy   = "Terraform"
  }
}

module "headers_policy" {
  source = "./modules/headers_policy"
  name   = "security-headers-policy"
}

module "waf" {
  source     = "./modules/waf"
  name       = "static-website-waf"
  rate_limit = var.rate_limit
  tags       = local.tags
  providers = {
    aws = aws.us_east_1
  }
}

module "cloudfront_logs" {
  source         = "./modules/log_bucket"
  name_prefix    = "cloudfront-logs"
  lifecycle_days = var.log_lifecycle_days
  tags           = local.tags
}

module "waf_logs" {
  source         = "./modules/log_bucket"
  name_prefix    = "waf-logs"
  lifecycle_days = var.log_lifecycle_days
  tags           = local.tags
  providers = {
    aws = aws.us_east_1
  }
}

resource "aws_iam_role" "firehose_role" {
  name = "firehose-waf-logs-role"
  assume_role_policy = jsonencode({
    Version   = "2012-10-17"
    Statement = [{ Action = "sts:AssumeRole", Effect = "Allow", Principal = { Service = "firehose.amazonaws.com" } }]
  })
}

resource "aws_iam_role_policy" "firehose_policy" {
  name = "firehose-waf-logs-policy"
  role = aws_iam_role.firehose_role.id
  policy = jsonencode({
    Version = "2012-10-17"
    Statement = [{ Effect = "Allow", Action = [
      "s3:AbortMultipartUpload", "s3:GetBucketLocation", "s3:GetObject", "s3:ListBucket", "s3:ListBucketMultipartUploads", "s3:PutObject"
    ], Resource = [module.waf_logs.bucket_arn, "${module.waf_logs.bucket_arn}/*"] }]
  })
}

resource "aws_kinesis_firehose_delivery_stream" "waf_logs" {
  provider    = aws.us_east_1
  name        = "waf-logs-stream"
  destination = "extended_s3"
  extended_s3_configuration {
    role_arn           = aws_iam_role.firehose_role.arn
    bucket_arn         = module.waf_logs.bucket_arn
    buffering_size     = 128
    buffering_interval = 300
    compression_format = "GZIP"
    prefix             = "year=!{timestamp:yyyy}/month=!{timestamp:MM}/day=!{timestamp:dd}/hour=!{timestamp:HH}/"
  }

  tags = local.tags
}

resource "aws_wafv2_web_acl_logging_configuration" "main" {
  provider                = aws.us_east_1
  log_destination_configs = [aws_kinesis_firehose_delivery_stream.waf_logs.arn]
  resource_arn            = module.waf.arn
  redacted_fields {
    single_header { name = "authorization" }
  }
}

module "website_bucket" {
  source      = "./modules/website_bucket"
  bucket_name = "${var.domain_name}-static-site"
  tags        = local.tags
}

module "cloudfront" {
  source                        = "./modules/cloudfront"
  domain_name                   = var.domain_name
  certificate_domain_name       = var.domain_name
  origin_bucket_regional_domain = module.website_bucket.bucket_regional_domain_name
  response_headers_policy_id    = module.headers_policy.id
  waf_web_acl_arn               = module.waf.arn
  price_class                   = var.price_class
  log_bucket_domain             = module.cloudfront_logs.bucket_domain_name
  tags                          = local.tags
  origin_shield_region          = var.us_east_1_region
  providers = {
    aws           = aws
    aws.us_east_1 = aws.us_east_1
  }
}

data "aws_iam_policy_document" "s3_policy" {
  statement {
    actions   = ["s3:GetObject"]
    resources = ["${module.website_bucket.arn}/*"]
    principals {
      type        = "Service"
      identifiers = ["cloudfront.amazonaws.com"]
    }
    condition {
      test     = "StringEquals"
      variable = "AWS:SourceArn"
      values   = [module.cloudfront.distribution_arn]
    }
  }

  # Deny non-TLS while excluding AWS service principals
  statement {
    sid     = "DenyInsecureTransport"
    effect  = "Deny"
    actions = ["s3:*"]
    resources = [
      module.website_bucket.arn,
      "${module.website_bucket.arn}/*"
    ]
    principals {
      type        = "*"
      identifiers = ["*"]
    }
    condition {
      test     = "Bool"
      variable = "aws:SecureTransport"
      values   = ["false"]
    }
    condition {
      test     = "Bool"
      variable = "aws:PrincipalIsAWSService"
      values   = ["false"]
    }
  }
}

resource "aws_s3_bucket_policy" "website" {
  bucket = module.website_bucket.id
  policy = data.aws_iam_policy_document.s3_policy.json
}

module "route53_alias" {
  source                      = "./modules/route53_alias"
  domain_name                 = var.domain_name
  distribution_domain_name    = module.cloudfront.distribution_domain_name
  distribution_hosted_zone_id = module.cloudfront.distribution_hosted_zone_id
}