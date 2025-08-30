terraform {
  backend "s3" {
    # Replace with your actual S3 bucket for Terraform state
    bucket = "your-terraform-state-bucket"
    key    = "cspm-monitor/terraform.tfstate"
    region = "us-east-1"

    # Enable encryption for state file
    encrypt = true

    # Enable DynamoDB locking
    # dynamodb_table = "your-terraform-locks-table"

    # Enable versioning for state file recovery
    # versioning = true
  }
}