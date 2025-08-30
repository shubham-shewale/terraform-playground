terraform {
  backend "s3" {
    bucket = "your-terraform-state-bucket"  # Replace with actual bucket
    key    = "cspm-monitor/terraform.tfstate"
    region = "us-east-1"
  }
}