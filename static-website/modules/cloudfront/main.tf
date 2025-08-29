variable "domain_name" { type = string }
variable "origin_bucket_regional_domain" { type = string }
variable "response_headers_policy_id" { type = string }
variable "waf_web_acl_arn" { type = string }
variable "price_class" { type = string }
variable "log_bucket_domain" { type = string }
variable "tags" { type = map(string) }
variable "origin_shield_region" { 
  type = string 
  default = "us-east-1" 
}

# Managed policies (resolved at apply time)
data "aws_cloudfront_cache_policy" "managed_caching_optimized" {
  name = "Managed-CachingOptimized"
}
data "aws_cloudfront_origin_request_policy" "managed_cors_s3_origin" {
  name = "Managed-CORS-S3Origin"
}
resource "aws_cloudfront_origin_access_control" "oac" {
  name                              = "${var.domain_name}-oac"
  description                       = "OAC for static website"
  origin_access_control_origin_type = "s3"
  signing_behavior                  = "always"
  signing_protocol                  = "sigv4"
}

resource "aws_cloudfront_distribution" "this" {
  origin {
    domain_name              = var.origin_bucket_regional_domain
    origin_access_control_id = aws_cloudfront_origin_access_control.oac.id
    origin_id                = "s3-origin"
    origin_shield {
      enabled              = true
      origin_shield_region = var.origin_shield_region
    }
  }

  enabled             = true
  is_ipv6_enabled     = true
  comment             = "Static website distribution for ${var.domain_name}"
  default_root_object = "index.html"

  aliases = [var.domain_name]
  web_acl_id = var.waf_web_acl_arn

  default_cache_behavior {
    allowed_methods  = ["GET", "HEAD"]
    cached_methods   = ["GET", "HEAD"]
    target_origin_id = "s3-origin"
    cache_policy_id           = data.aws_cloudfront_cache_policy.managed_caching_optimized.id
    origin_request_policy_id  = data.aws_cloudfront_origin_request_policy.managed_cors_s3_origin.id
    viewer_protocol_policy = "redirect-to-https"
    min_ttl  = 0
    default_ttl = 3600
    max_ttl = 86400
    compress = true
    response_headers_policy_id = var.response_headers_policy_id
  }

  # Enable HTTP/3 with fallback to HTTP/2/1.1
  http_version = "http2and3"

  custom_error_response {
    error_code         = 403
    response_code      = 404
    response_page_path = "/error.html"
  }
  custom_error_response {
    error_code         = 404
    response_code      = 404
    response_page_path = "/error.html"
  }

  price_class = var.price_class

  restrictions {
    geo_restriction { restriction_type = "none" }
  }

  viewer_certificate {
    acm_certificate_arn      = aws_acm_certificate_validation.cert.certificate_arn
    ssl_support_method       = "sni-only"
    minimum_protocol_version = "TLSv1.2_2021"
  }

  logging_config {
    include_cookies = false
    bucket          = var.log_bucket_domain
    prefix          = "cloudfront-logs"
  }

  tags = var.tags
}

variable "certificate_domain_name" { type = string }

data "aws_route53_zone" "this" {
  name = var.certificate_domain_name
  private_zone = false
}

resource "aws_acm_certificate" "cert" {
  provider          = aws.us_east_1
  domain_name       = var.certificate_domain_name
  validation_method = "DNS"
  tags              = var.tags
  lifecycle {
    create_before_destroy = true
  }
}

resource "aws_route53_record" "cert_validation" {
  for_each = {
    for dvo in aws_acm_certificate.cert.domain_validation_options : dvo.domain_name => {
      name   = dvo.resource_record_name
      record = dvo.resource_record_value
      type   = dvo.resource_record_type
    }
  }
  allow_overwrite = true
  name    = each.value.name
  records = [each.value.record]
  ttl     = 60
  type    = each.value.type
  zone_id = data.aws_route53_zone.this.zone_id
}

resource "aws_acm_certificate_validation" "cert" {
  provider                = aws.us_east_1
  certificate_arn         = aws_acm_certificate.cert.arn
  validation_record_fqdns = [for record in aws_route53_record.cert_validation : record.fqdn]
}

output "distribution_domain_name" { value = aws_cloudfront_distribution.this.domain_name }
output "distribution_hosted_zone_id" { value = aws_cloudfront_distribution.this.hosted_zone_id }
output "distribution_arn" { value = aws_cloudfront_distribution.this.arn }
output "certificate_arn" { value = aws_acm_certificate_validation.cert.certificate_arn }

