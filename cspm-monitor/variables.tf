variable "project_name" {
  description = "Name of the CSPM project"
  type        = string
  default     = "cspm-monitor"

  validation {
    condition     = can(regex("^[a-z0-9-]+$", var.project_name))
    error_message = "Project name must contain only lowercase letters, numbers, and hyphens."
  }

  validation {
    condition     = length(var.project_name) >= 3 && length(var.project_name) <= 63
    error_message = "Project name must be between 3 and 63 characters."
  }
}

variable "region" {
  description = "AWS region for resource deployment"
  type        = string
  default     = "us-east-1"

  validation {
    condition = contains([
      "us-east-1", "us-east-2", "us-west-1", "us-west-2",
      "eu-west-1", "eu-central-1", "ap-southeast-1", "ap-northeast-1"
    ], var.region)
    error_message = "Region must be a valid AWS region."
  }
}

variable "domain_name" {
  description = "Domain name for CloudFront"
  type        = string
  default     = "cspm-monitor.example.com"

  validation {
    condition     = can(regex("^[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.domain_name))
    error_message = "Domain name must be a valid domain format (e.g., example.com)"
  }
}

variable "certificate_domain_name" {
  description = "Domain name for SSL certificate"
  type        = string
  default     = "cspm-monitor.example.com"

  validation {
    condition     = can(regex("^[a-zA-Z0-9.-]+\\.[a-zA-Z]{2,}$", var.certificate_domain_name))
    error_message = "Certificate domain name must be a valid domain format (e.g., example.com)"
  }
}

variable "lambda_runtime" {
  description = "Runtime for Lambda functions"
  type        = string
  default     = "python3.9"

  validation {
    condition = contains([
      "python3.8", "python3.9", "python3.10", "python3.11",
      "nodejs16.x", "nodejs18.x", "nodejs20.x"
    ], var.lambda_runtime)
    error_message = "Lambda runtime must be a supported runtime version."
  }
}

variable "dynamodb_billing_mode" {
  description = "Billing mode for DynamoDB table"
  type        = string
  default     = "PAY_PER_REQUEST"

  validation {
    condition     = contains(["PROVISIONED", "PAY_PER_REQUEST"], var.dynamodb_billing_mode)
    error_message = "DynamoDB billing mode must be either PROVISIONED or PAY_PER_REQUEST"
  }
}

variable "enable_cloudtrail" {
  description = "Enable CloudTrail integration"
  type        = bool
  default     = false

  validation {
    condition     = var.enable_cloudtrail == false || var.enable_cloudtrail == true
    error_message = "enable_cloudtrail must be a boolean value."
  }
}

variable "sync_schedule_rate" {
  description = "Rate for periodic sync of findings"
  type        = string
  default     = "rate(6 hours)"

  validation {
    condition     = can(regex("^rate\\(\\d+ (hours?|minutes?|days?)\\)$", var.sync_schedule_rate))
    error_message = "Schedule rate must be in format: rate(X hours|minutes|days)"
  }
}

variable "enable_sync_schedule" {
  description = "Enable periodic sync of findings"
  type        = bool
  default     = true
}

variable "enable_api_gateway" {
  description = "Enable API Gateway for the dashboard"
  type        = bool
  default     = true
}

# Data retention and compliance variables
variable "dynamodb_ttl_enabled" {
  description = "Enable DynamoDB Time-to-Live for automatic data expiration"
  type        = bool
  default     = true
}

variable "dynamodb_ttl_days" {
  description = "Number of days to retain security findings in DynamoDB before TTL expiration"
  type        = number
  default     = 90

  validation {
    condition     = var.dynamodb_ttl_days >= 30 && var.dynamodb_ttl_days <= 365
    error_message = "TTL days must be between 30 and 365 days for compliance."
  }
}

variable "enable_s3_archival" {
  description = "Enable S3 archival for long-term security log retention"
  type        = bool
  default     = true
}

variable "s3_archive_retention_days" {
  description = "Number of days to retain archived security logs in S3"
  type        = number
  default     = 2555 # 7 years for compliance

  validation {
    condition     = var.s3_archive_retention_days >= 365 && var.s3_archive_retention_days <= 2555
    error_message = "S3 retention must be between 1 year and 7 years for compliance."
  }
}

variable "enable_backup" {
  description = "Enable automated DynamoDB backups for compliance"
  type        = bool
  default     = true
}

variable "backup_retention_days" {
  description = "Number of days to retain DynamoDB backups"
  type        = number
  default     = 35

  validation {
    condition     = var.backup_retention_days >= 7 && var.backup_retention_days <= 35
    error_message = "Backup retention must be between 7 and 35 days."
  }
}

variable "compliance_framework" {
  description = "Compliance framework (PCI-DSS, SOC2, HIPAA, ISO27001)"
  type        = string
  default     = "PCI-DSS"

  validation {
    condition = contains([
      "PCI-DSS", "SOC2", "HIPAA", "ISO27001", "NIST", "GDPR"
    ], var.compliance_framework)
    error_message = "Compliance framework must be one of: PCI-DSS, SOC2, HIPAA, ISO27001, NIST, GDPR"
  }
}

variable "enable_critical_escalation" {
  description = "Enable critical alert escalation to separate SNS topic"
  type        = bool
  default     = true

  validation {
    condition     = var.enable_critical_escalation == false || var.enable_critical_escalation == true
    error_message = "enable_critical_escalation must be a boolean value."
  }
}

variable "terraform_state_bucket" {
  description = "S3 bucket for Terraform state storage"
  type        = string
  default     = "your-terraform-state-bucket"

  validation {
    condition     = can(regex("^[a-z0-9][a-z0-9-]*[a-z0-9]$", var.terraform_state_bucket))
    error_message = "S3 bucket name must be valid (lowercase, no uppercase, no special chars except hyphens)."
  }
}

variable "terraform_locks_table" {
  description = "DynamoDB table for Terraform state locking"
  type        = string
  default     = "your-terraform-locks-table"

  validation {
    condition     = can(regex("^[a-zA-Z0-9._-]+$", var.terraform_locks_table))
    error_message = "DynamoDB table name must be valid."
  }
}