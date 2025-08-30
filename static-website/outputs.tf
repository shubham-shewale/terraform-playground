output "cloudfront_domain" { value = module.cloudfront.distribution_domain_name }
output "s3_bucket_name" { value = module.website_bucket.bucket }

# CloudFront outputs
output "cloudfront_distribution_id" { value = module.cloudfront.distribution_arn }
output "cloudfront_distribution_arn" { value = module.cloudfront.distribution_arn }
output "cloudfront_price_class" { value = var.price_class }
output "origin_shield_enabled" { value = true }
output "origin_shield_region" { value = var.us_east_1_region }
output "compression_enabled" { value = true }

# WAF outputs
output "waf_web_acl_arn" { value = module.waf.arn }
output "waf_rate_limit" { value = var.rate_limit }
output "waf_rule_count" { value = 6 }  # Based on the WAF configuration

# Certificate outputs
output "certificate_arn" { value = module.cloudfront.certificate_arn }
output "certificate_validation_method" { value = "DNS" }

# S3 bucket outputs
output "s3_bucket_arn" { value = module.website_bucket.arn }
output "s3_bucket_regional_domain" { value = module.website_bucket.bucket_regional_domain_name }

# Log retention outputs
output "cloudfront_log_retention_days" { value = var.log_lifecycle_days }
output "waf_log_retention_days" { value = var.log_lifecycle_days }

# CloudTrail outputs
output "cloudtrail_enabled" { value = true }