# tflint configuration for enforcing Terraform best practices
config {
  call_module_type = "all"
  force  = false
}

plugin "aws" {
  enabled = true
  version = "0.24.1"
  source  = "github.com/terraform-linters/tflint-ruleset-aws"
}

# Enforce naming conventions
rule "terraform_naming_convention" {
  enabled = true
}

# Require comments for resources
rule "terraform_documented_outputs" {
  enabled = true
}

rule "terraform_documented_variables" {
  enabled = true
}

# Disallow deprecated syntax
rule "terraform_deprecated_interpolation" {
  enabled = true
}

rule "terraform_deprecated_index" {
  enabled = true
}

# Enforce standard module structure
rule "terraform_standard_module_structure" {
  enabled = true
}

# Security-focused rules
rule "aws_instance_invalid_type" {
  enabled = true
}

rule "aws_security_group_rule_invalid_type" {
  enabled = true
}

rule "aws_security_group_rule_invalid_protocol" {
  enabled = true
}

# Enforce encryption at rest
rule "aws_ebs_volume_encrypted" {
  enabled = true
}

rule "aws_ebs_snapshot_encrypted" {
  enabled = true
}

# Enforce secure defaults
rule "aws_iam_policy_document_gov_friendly_arns" {
  enabled = true
}

rule "aws_iam_role_policy_gov_friendly_arns" {
  enabled = true
}

# Enforce proper tagging
rule "aws_resource_missing_tags" {
  enabled = true
  tags = [
    "Environment",
    "Project",
    "ManagedBy"
  ]
}

# Enforce security group best practices
rule "aws_security_group_rule_cidr_blocks" {
  enabled = true
}

rule "aws_security_group_rule_description" {
  enabled = true
}

# Enforce S3 bucket security
rule "aws_s3_bucket_public_read" {
  enabled = true
}

rule "aws_s3_bucket_public_write" {
  enabled = true
}

# Enforce CloudTrail logging
rule "aws_cloudtrail_insufficient_logging" {
  enabled = true
}

# Enforce VPC Flow Logs
rule "aws_vpc_flow_log_insufficient_logging" {
  enabled = true
}