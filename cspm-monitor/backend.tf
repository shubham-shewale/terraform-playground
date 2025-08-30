terraform {
  backend "s3" {
    bucket = var.terraform_state_bucket
    key    = "${var.project_name}/terraform.tfstate"
    region = var.region

    # Enable encryption for state file
    encrypt = true

    # Enable DynamoDB locking
    dynamodb_table = var.terraform_locks_table

    # Enable versioning for state file recovery
    versioning = true
  }
}