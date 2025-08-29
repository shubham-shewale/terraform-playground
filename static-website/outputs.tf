output "cloudfront_domain" { value = module.cloudfront.distribution_domain_name }
output "s3_bucket_name" { value = module.website_bucket.bucket }