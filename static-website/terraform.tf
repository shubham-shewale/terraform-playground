terraform {
  required_providers {
    aws = {
      source  = "hashicorp/aws"
      version = "~> 5.0"
    }
    random = {
      source  = "hashicorp/random"
      version = ">= 3.6.0"
    }
  }
}

variable "region" {
  type    = string
  default = "us-east-1"
}
variable "us_east_1_region" {
  type    = string
  default = "us-east-1"
}

provider "aws" {
  region = var.region
}

provider "aws" {
  alias  = "us_east_1"
  region = var.us_east_1_region
}