variable "region" {
  description = "AWS region"
  default     = "us-east-1"
}

variable "vpc_cidr" {
  description = "VPC CIDR block"
  default     = "172.16.0.0/16"
}

variable "azs" {
  type        = list(string)
  description = "Availability Zones"
  default     = ["us-east-1a"]
}

variable "public_subnet_cidrs" {
  type        = list(string)
  description = "Public subnet CIDRs"
  default     = ["172.16.1.0/24"]
}

variable "private_subnet_cidrs" {
  type        = list(string)
  description = "Private subnet CIDRs"
  default     = ["172.16.10.0/24"]
}

variable "key_name" {}

variable "public_key" {}

variable "bastion_ami" {
  description = "AMI for bastion host (e.g., Amazon Linux 2023)"
  default     = "ami-0123456789abcdef0"
}

variable "private_ami" {
  description = "AMI for private instance"
  default     = "ami-0123456789abcdef0"
}
